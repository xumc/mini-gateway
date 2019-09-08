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
	{
		Path: "^/svc2/grpc_hello$",
		Upstreams: []Upstream{
			{
				Host:         "localhost:8081",
				Schema:       "grpc",
				GrpcEndPoint: "proto.GrpcUpstreamService/Hello",
			},
		},
		Filters: []string{"auth", "inspector"},
	},
}

type Upstream struct {
	Host         string
	Schema       string
	GrpcEndPoint string
}

type RouteSpec struct {
	Path      string
	Upstreams []Upstream
	Filters   []string
}
