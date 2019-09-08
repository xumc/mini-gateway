package main

import (
	"golang.org/x/time/rate"
	"net/http"
	"strings"
)

// TODO how to customize condtions.
type condition struct {
	ip string
}

type rateLimiterHandler struct {
	next http.Handler

	// TODO lims are read-only after gateway starts, no need to lock and unlock. Afterward, if we
	// want to support dynamic loading rate limit configuration, we need to add sync.RWlock.
	lims map[condition]*rate.Limiter
}

func NewRateLimiterHandler(next http.Handler) http.Handler {
	limHandler := &rateLimiterHandler{
		next: next,
		lims: make(map[condition]*rate.Limiter),
	}

	lim := rate.NewLimiter(rate.Limit(1), 1) // TODO configurable
	con := condition{ip: "127.0.0.1"}

	limHandler.lims[con] = lim

	return limHandler
}

func (r *rateLimiterHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	con := condition{
		ip: strings.Split(req.RemoteAddr, ":")[0], // TODO handle special ip.
	}

	var lim *rate.Limiter
	var ok bool
	if lim, ok = r.lims[con]; !ok {
		r.next.ServeHTTP(resp, req)
		return
	}

	if lim.Allow() {
		r.next.ServeHTTP(resp, req)
	} else {
		resp.WriteHeader(429)
	}
}
