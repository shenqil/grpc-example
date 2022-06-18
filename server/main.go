package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	pb "helloworld/helloworld"
)

type server struct {
	pb.UnimplementedGreeterServer
}

func (s *server) UnaryEcho(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	fmt.Println("---UnaryEcho---")

	defer func() {
		trailer := metadata.Pairs("timestamp", time.Now().Format(time.StampNano))
		grpc.SetTrailer(ctx, trailer)
	}()

	md, ok := metadata.FromIncomingContext(ctx)

	if !ok {
		return nil, status.Errorf(codes.DataLoss, "无法获取元数据")
	}

	if t, ok := md["timestamp"]; ok {
		fmt.Println("timestamp from metadata:")
		for i, e := range t {
			fmt.Printf("%d.%s\n", i, e)
		}
	}

	header := metadata.New(map[string]string{"location": "MTV", "timestamp": time.Now().Format(time.StampNano)})
	grpc.SendHeader(ctx, header)

	fmt.Printf("已接受到的请求:%v,发送响应\n", in)

	return &pb.HelloReply{Message: "Hello again " + in.GetName()}, nil
}

func main() {

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterGreeterServer(grpcServer, &server{})
	log.Printf("server listening at %v", lis.Addr())

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
