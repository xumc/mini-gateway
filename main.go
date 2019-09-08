package main

import (
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/http/httputil"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

type Server struct {
	port      int
	transport http.RoundTripper
	running   bool
	mu        sync.Mutex
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

		subMatches := reg.FindStringSubmatch(r.URL.Path)
		r.URL.Path = "/" + subMatches[1]

		// random select one upstream
		index := rand.Intn(len(route.Upstreams))
		upstream := route.Upstreams[index]

		r.URL.Host = upstream.Host
		r.URL.Scheme = upstream.Schema

		r.Header.Set(filtersHeaderKey, strings.Join(route.Filters, ","))

		setOriginHeader(r)

		break
	}
}

func (s *Server) RoundTrip(r *http.Request) (*http.Response, error) {
	filterNames := strings.Split(r.Header.Get(filtersHeaderKey), ",")
	r.Header.Del(filtersHeaderKey)
	if len(filterNames) == 1 && filterNames[0] == "" {
		return s.transport.RoundTrip(r)
	}

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

	resp, upstreamError := s.transport.RoundTrip(r)

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

type mh struct{}

func (m *mh) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	s := "url :" + req.URL.String() + "\n" + time.Now().String()
	rw.Write([]byte(s))
}

func main() {
	server := &Server{
		port:      8080, // TODO configurable
		transport: http.DefaultTransport,
		running:   false,
		mu:        sync.Mutex{},
	}

	proxy := &httputil.ReverseProxy{Director: server.Director, Transport: server}

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", server.port))
	if err != nil {
		fmt.Printf("can not initialize listener: %+v", err)
	}

	httpS := &http.Server{
		Handler: proxy,
	}

	server.running = true

	go http.ListenAndServe(":8081", &mh{})

	fmt.Println(httpS.Serve(l))
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
