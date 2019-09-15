package main

import (
	"testing"
)

func TestInvokeRPC(t *testing.T) {
	g := DefaultGrpcTransport{}
	content := "{\"hello\":\"nihao\"}"
	timeMonitor(func() {
		g.invokeRPC(content, "localhost:8081", "proto.GrpcUpstreamService/Hello")
	})
}

func BenchmarkInvokeRPC(b *testing.B) {
	g := DefaultGrpcTransport{}
	content := "{\"hello\":\"nihao\"}"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.invokeRPC(content, "localhost:8081", "proto.GrpcUpstreamService/Hello")
	}
}
