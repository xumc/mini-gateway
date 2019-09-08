package main

import (
	"net/http"
	"time"
)

type mh struct{}

func (m *mh) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	//time.Sleep(3 * time.Second)
	s := "url :" + req.URL.String() + "\n" + time.Now().String()
	rw.Write([]byte(s))
}
