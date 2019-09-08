package main

import (
	"golang.org/x/time/rate"
	"net/http"
)

type rateLimiterHandler struct {
	next http.Handler

	lim *rate.Limiter
}

func NewRateLimiterHandler(next http.Handler) http.Handler {
	lim := rate.NewLimiter(rate.Limit(1), 1) // TODO configurable

	limHandler := &rateLimiterHandler{
		next: next,
		lim:  lim,
	}

	return limHandler
}

func (r *rateLimiterHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	if r.lim.Allow() {
		r.next.ServeHTTP(resp, req)
	} else {
		resp.WriteHeader(429)
	}
}
