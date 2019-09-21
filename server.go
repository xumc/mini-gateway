package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	ServerFdOffset = 3
)

type Server struct {
	running bool
	mu      sync.Mutex

	httpTransport http.RoundTripper
	grpcTransport GrpcTransport

	*http.Server
	port         int
	listener     net.Listener
	handler      http.Handler
	isChild      bool
	sigChan      chan os.Signal
	shutdownChan chan struct{}
}

func (s *Server) StartServe() error {
	s.isChild = os.Getenv("MINI_GATEWAY_CONTINUE") != ""

	listener, err := s.getListener()
	if err != nil {
		return err
	}

	s.listener = listener

	s.Server = &http.Server{
		Handler:      s.handler,
		ReadTimeout:  60 * time.Second, // TODO configuable
		WriteTimeout: 60 * time.Second, // TODO configuable
	}

	s.sigChan = make(chan os.Signal)
	go s.handleSignals()

	s.shutdownChan = make(chan struct{}, 1)

	if s.isChild {
		syscall.Kill(syscall.Getppid(), syscall.SIGTERM)
	}

	err = s.Server.Serve(s.listener)
	if err != http.ErrServerClosed {
		fmt.Println(err)
		return err
	}

	fmt.Println("waiting for connections closed.")
	<-s.shutdownChan
	fmt.Println("all connections closed.")
	return nil
}

func (s *Server) handleSignals() {
	var sig os.Signal

	signal.Notify(
		s.sigChan,
		syscall.SIGHUP,
		syscall.SIGINT,
	)

	pid := syscall.Getpid()
	for {
		sig = <-s.sigChan
		switch sig {
		case syscall.SIGHUP:
			log.Println(pid, "Received SIGHUP. forking.")
			err := s.fork()
			if err != nil {
				log.Println("Fork err:", err)
			}
		case syscall.SIGINT:
			log.Println(pid, "Received SIGINT.")
			s.shutdown()
			fmt.Println("after shutdown")
		}
	}
}

func (s *Server) fork() (err error) {
	file, err := s.listener.(*net.TCPListener).File()
	if err != nil {
		return err
	}

	env := append(
		os.Environ(),
		"MINI_GATEWAY_CONTINUE=1",
	)

	// log.Println(files)
	path := os.Args[0]
	var args []string
	if len(os.Args) > 1 {
		args = os.Args[1:]
	}

	cmd := exec.Command(path, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.ExtraFiles = []*os.File{file}
	cmd.Env = env

	err = cmd.Start()
	if err != nil {
		log.Fatalf("Restart: Failed to launch, error: %v", err)
	}

	return
}

func (s *Server) getListener() (net.Listener, error) {
	if s.isChild {
		f := os.NewFile(ServerFdOffset, "")
		l, err := net.FileListener(f)
		if err != nil {
			return nil, fmt.Errorf("net.FileListener error: %v", err)
		}
		return l, nil
	} else {
		l, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
		if err != nil {
			return nil, fmt.Errorf("can not initialize listener: %+v", err)
		}

		return l, nil
	}
}

func (s *Server) shutdown() {
	fmt.Println("pre shutdown")
	err := s.Server.Shutdown(context.Background())
	fmt.Println("post shutdown")
	if err != nil {
		fmt.Println(err)
	}
	close(s.shutdownChan)
}

func (s *Server) Director(r *http.Request) {
	for _, route := range registeredRoutes {
		reg, err := regexp.Compile(route.Path)
		if err != nil {
			fmt.Println("invalid config item, ignore")
		}

		matched := reg.MatchString(r.URL.Path)
		if !matched {
			continue
		}

		// random select one upstream
		index := rand.Intn(len(route.Upstreams))
		upstream := route.Upstreams[index]

		r.URL.Host = upstream.Host
		r.URL.Scheme = upstream.Schema

		if upstream.Schema != "grpc" {
			subMatches := reg.FindStringSubmatch(r.URL.Path)
			r.URL.Path = "/" + subMatches[1]
		} else {
			r.Method = upstream.GrpcEndPoint
		}

		r.Header.Set(filtersHeaderKey, strings.Join(route.Filters, ","))

		setOriginHeader(r)

		break
	}
}

func (s *Server) RoundTrip(r *http.Request) (*http.Response, error) {
	filterNames := strings.Split(r.Header.Get(filtersHeaderKey), ",")
	r.Header.Del(filtersHeaderKey)
	if len(filterNames) == 1 && filterNames[0] == "" {
		filterNames = []string{}
	}

	// TODO cache filters to improve performance
	preFilters := make([]Filter, 0, len(filterNames))
	postFilters := make([]Filter, 0, len(filterNames))
	for _, fn := range filterNames {
		filter := registeredFilters[fn]

		switch filter.GetType() {
		case "PRE":
			preFilters = append(preFilters, filter)
		case "POST":
			postFilters = append(postFilters, filter)
		}
	}

	filterSorter := func(i, j int) bool {
		return preFilters[i].GetOrder() < preFilters[j].GetOrder()
	}

	sort.Slice(preFilters, filterSorter)

	for _, f := range preFilters {
		ok, err := f.ShouldFilter(r)
		if err != nil {
			fmt.Println(err)
			// TODO handler gateway error
		}
		if ok {
			err := f.(PreFilter).Run(r)
			if err != nil {
				fmt.Println(err)
				// TODO handler gateway error
			}
		}
	}

	var resp *http.Response
	var upstreamError error

	if r.URL.Scheme == "grpc" {
		resp, upstreamError = s.grpcTransport.RoundTrip(r)
	} else {
		resp, upstreamError = s.httpTransport.RoundTrip(r)
	}

	sort.Slice(postFilters, filterSorter)

	for _, f := range postFilters {
		ok, err := f.ShouldFilter(r)
		if err != nil {
			fmt.Println(err)
			// TODO handler gateway error
		}
		if ok {
			err := f.(PostFilter).Run(r, resp, upstreamError)
			if err != nil {
				fmt.Println(err)
				// TODO handler gateway error
			}
		}
	}

	return resp, upstreamError
}

func setOriginHeader(r *http.Request) {
	// do nothing for non-GET requests
	if strings.ToUpper(r.Method) != "GET" || r.URL == nil {
		return
	}
	if r.Header == nil {
		r.Header = make(http.Header)
	}
	if h := r.URL.Host; h == "" {
		if hh := r.Header.Get("Host"); hh != "" {
			r.URL.Host = hh
		}

		if r.Host != "" {
			r.URL.Host = r.Host
		}
	}
}
