package main

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"sync"
)

type Server struct {
	port        int
	transport   http.RoundTripper
	running     bool
	mu          sync.Mutex
}

func (s *Server) Director(r *http.Request) {
	r.URL.Host = "www.csdn.com"
	r.URL.Scheme = "http"
}

func (s *Server) RoundTrip(r *http.Request) (*http.Response, error) {
	return s.transport.RoundTrip(r)
}

func main() {
	server := &Server{
		port: 8080, // TODO configurable
		transport: http.DefaultTransport,
		running: false,
		mu: sync.Mutex{},
	}

	proxy := &httputil.ReverseProxy{Director: server.Director, Transport: server}

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", server.port))
	if err != nil {
		fmt.Printf("can not initialize listener: %+v", err)
	}

	httpS := &http.Server{
		Handler:      proxy,
	}

	server.running = true

	fmt.Println(httpS.Serve(l))
}
