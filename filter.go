package main

import "net/http"

const filtersHeaderKey = "MINI-GATEWAY-FILTERS"

var registeredFilters = map[string]Filter{}

type Filter interface {
	GetType() string // "PRE / POST"
	GetOrder() int
	ShouldFilter(r *http.Request) (bool, error)
}

type PreFilter interface {
	Filter
	Run(r *http.Request) error
}

type PostFilter interface {
	Filter
	Run(r *http.Request, resp *http.Response, upstreamError error) error
}
