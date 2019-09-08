package main

import (
	"net/http"
)

func init() {
	registeredFilters["auth"] = &AuthFilter{}
}

type AuthFilter struct{}

func (a *AuthFilter) GetType() string {
	return "PRE"
}
func (a *AuthFilter) GetOrder() int {
	return 0
}

func (a *AuthFilter) ShouldFilter(r *http.Request) (bool, error) {
	return true, nil
}

func (a *AuthFilter) Run(r *http.Request) error {
	//fmt.Println("authing ...")
	return nil
}
