package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fullstorydev/grpcurl"
	"github.com/jhump/protoreflect/grpcreflect"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	reflectpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

type GrpcTransport interface {
	RoundTrip(*http.Request) (*http.Response, error)
}

// TODO we should use connection pool to improve performance, but you know, its a prototype project now.
type DefaultGrpcTransport struct{}

func (g *DefaultGrpcTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.RawQuery != "" {
		str, err := g.transformRawQueryToBodyJson(req.URL.RawQuery)
		if err != nil {
			return nil, err
		}
		req.Body = ioutil.NopCloser(bytes.NewBufferString(str))
	}

	reqBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	reqContent := string(reqBytes)

	target := req.URL.Host
	symbol := req.Method

	respStr, err := g.invokeRPC(reqContent, target, symbol)
	resp := &http.Response{}
	if err != nil {
		resp.StatusCode = http.StatusInternalServerError
	} else {
		resp.StatusCode = http.StatusOK
	}
	resp.Body = ioutil.NopCloser(bytes.NewBufferString(respStr))

	return resp, nil
}

func (g *DefaultGrpcTransport) transformRawQueryToBodyJson(rq string) (string, error) {
	// TODO this is a simplest transformer, we'd better build a enhanced one.
	m := make(map[string]string)

	kvs := strings.Split(rq, "&")

	for _, kv := range kvs {
		kvc := strings.Split(kv, "=")
		k := kvc[0]
		v := kvc[1]

		m[k] = v
	}

	ret, err := json.Marshal(m)
	if err != nil {
		return "", err
	}

	return string(ret), nil
}

func (g *DefaultGrpcTransport) invokeRPC(reqContent, target, symbol string) (string, error) {
	ctx := context.Background()

	dial := func() *grpc.ClientConn {
		network := "tcp"
		cc, err := grpcurl.BlockingDial(ctx, network, target, nil)
		if err != nil {
			fmt.Println(err, "Failed to dial target host %q", target)
		}
		return cc
	}

	var cc *grpc.ClientConn
	var descSource grpcurl.DescriptorSource
	var refClient *grpcreflect.Client

	md := grpcurl.MetadataFromHeaders([]string{})
	refCtx := metadata.NewOutgoingContext(ctx, md)
	cc = dial()
	refClient = grpcreflect.NewClient(refCtx, reflectpb.NewServerReflectionClient(cc))
	descSource = grpcurl.DescriptorSourceFromServer(ctx, refClient)

	// arrange for the RPCs to be cleanly shutdown
	reset := func() {
		if refClient != nil {
			refClient.Reset()
			refClient = nil
		}
		if cc != nil {
			cc.Close()
			cc = nil
		}
	}
	defer reset()
	exit := func(code int) {
		// since defers aren't run by os.Exit...
		reset()
		os.Exit(code)
	}

	// Invoke an RPC
	if cc == nil {
		cc = dial()
	}

	in := strings.NewReader(reqContent)

	rf, formatter, err := grpcurl.RequestParserAndFormatterFor(grpcurl.Format("json"), descSource, false, true, in)
	if err != nil {
		fmt.Println(err, "Failed to construct request parser and formatter for %q", "json")
	}

	out := &bytes.Buffer{}
	h := grpcurl.NewDefaultEventHandler(out, descSource, formatter, false)

	err = grpcurl.InvokeRPC(ctx, descSource, cc, symbol, []string{}, h, rf.Next)
	if err != nil {
		fmt.Println(err, "Error invoking method %q", symbol)
	}

	if h.Status.Code() != codes.OK {
		return "", errors.New("upstream return !OK")
		exit(1)
	}

	return out.String(), nil

}
