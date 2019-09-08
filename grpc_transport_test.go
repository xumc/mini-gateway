package main

import "testing"

func TestInvokeRPC(t *testing.T) {
	g := DefaultGrpcTransport{}
	g.invokeRPC()
}
