package main

var registeredRoutes = []RouteSpec{
	{
		Path: "^/svc1/(.*)",
		Upstreams: []Upstream{
			{
				Host:   "localhost:8081",
				Schema: "http",
			},
		},
		Filters: []string{"auth", "inspector"},
	},
}

type Upstream struct {
	Host   string
	Schema string
}

type RouteSpec struct {
	Path      string
	Upstreams []Upstream
	Filters   []string
}
