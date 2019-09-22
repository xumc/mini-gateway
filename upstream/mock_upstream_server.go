package main

import (
	"context"
	"fmt"
	"github.com/soheilhy/cmux"
	"github.com/xumc/mini-gateway/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"log"
	"net"
	"net/http"
	"time"
)

type mh struct{}

func (m *mh) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	//time.Sleep(3 * time.Second)
	s := "url :" + req.URL.String() + "\n" + time.Now().String()
	s += "version 3"
	rw.Write([]byte(s))
}

type GrpcMockServer struct{}

func (g *GrpcMockServer) Hello(_ context.Context, req *proto.Request) (*proto.Reply, error) {
	fmt.Println(req.Hello)

	return &proto.Reply{
		World: "世界",
	}, nil
}

func StartMockUpstreamServer() {
	lis, err := net.Listen("tcp", ":8081")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	m := cmux.New(lis)

	grpcL := m.MatchWithWriters(cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"))
	httpL := m.Match(cmux.HTTP1Fast())

	grpcServer := grpc.NewServer()
	proto.RegisterGrpcUpstreamServiceServer(grpcServer, &GrpcMockServer{})
	reflection.Register(grpcServer)

	go func() {
		err = grpcServer.Serve(grpcL)
		if err != nil {
			panic(fmt.Sprintf("can not serve grpc %s", err.Error()))
		}
	}()

	go func() {
		err := http.Serve(httpL, &mh{})
		if err != nil {
			panic(fmt.Sprintf("can not serve http %s", err.Error()))
		}
	}()

	fmt.Println(m.Serve())
}
