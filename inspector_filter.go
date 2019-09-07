package main

import (
	"fmt"
	"net/http"
)

func init() {
	registeredFilters["inspector"] = &InspectorFilter{}
}

type InspectorFilter struct{}

func (a *InspectorFilter) GetType() string {
	return "POST"
}
func (a *InspectorFilter) GetOrder() int {
	return 1
}

func (a *InspectorFilter) ShouldFilter(r *http.Request) (bool, error) {
	return true, nil
}

func (a *InspectorFilter) Run(r *http.Request, resp *http.Response, upstreamError error) error {
	fmt.Println("inspector ...")
	return nil
}
