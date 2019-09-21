package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	_ "net/http/pprof"
	"sync"
	"time"
)

func main() {
	server := &Server{
		port:          8080, // TODO configurable
		httpTransport: http.DefaultTransport,
		grpcTransport: NewDefaultGrpcTransport(),
		running:       false,
		mu:            sync.Mutex{},
	}

	proxy := &httputil.ReverseProxy{Director: server.Director, Transport: server}

	timeoutHandler := http.TimeoutHandler(proxy, 60*time.Second, "gateway timeout") // TODO configurable
	rateLimiterHandler := NewRateLimiterHandler(timeoutHandler)
	server.handler = rateLimiterHandler

	server.running = true

	//go http.ListenAndServe("0.0.0.0:8085", nil)

	fmt.Println(server.StartServe())
}

func timeMonitor(msg string, f func()) {
	begin := time.Now()

	//cpuf, err := os.Create("/Users/xumc/go/src/github.com/xumc/mini-gateway/cpu_profile")
	//if err != nil {
	//	log.Fatal(err)
	//}
	//pprof.StartCPUProfile(cpuf)
	//defer pprof.StopCPUProfile()
	//f()
	fmt.Println(msg, time.Now().Sub(begin).String())
}
