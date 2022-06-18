package main

import (
	"context"
	"fmt"
	"io"
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

func (s *server) ServerStreamingEcho(in *pb.HelloRequest, stream pb.Greeter_ServerStreamingEchoServer) error {
	fmt.Printf("--- ServerStreamingEcho ---\n")

	defer func() {
		trailer := metadata.Pairs("timestamp", time.Now().Format(time.StampNano))
		stream.SetTrailer(trailer)
	}()

	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		return status.Errorf(codes.DataLoss, "ServerStreamingEcho: 无法获取metadata")
	}
	if t, ok := md["timestamp"]; ok {
		fmt.Printf("timestamp from metadata:\n")
		for i, e := range t {
			fmt.Printf("%d.%s\n", i, e)
		}
	}

	header := metadata.New(map[string]string{"location": "MTV", "timestamp": time.Now().Format(time.StampNano)})
	stream.SendHeader(header)

	fmt.Printf("收到的请求: %v\n", in)

	for i := 0; i < 10; i++ {
		fmt.Printf("echo message %v\n", in.Name)

		err := stream.Send(&pb.HelloReply{Message: "Hello again " + in.GetName()})
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *server) ClientStreamingEcho(stream pb.Greeter_ClientStreamingEchoServer) error {
	fmt.Printf("--- ClientStreamingEcho ---\n")

	defer func() {
		trailer := metadata.Pairs("timestamp", time.Now().Format(time.StampNano))
		stream.SetTrailer(trailer)
	}()

	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		return status.Errorf(codes.DataLoss, "ClientStreamingEcho: failed to get metadata")
	}
	if t, ok := md["timestamp"]; ok {
		fmt.Printf("timestamp from metadata:\n")
		for i, e := range t {
			fmt.Printf(" %d. %s\n", i, e)
		}
	}

	header := metadata.New(map[string]string{"location": "MTV", "timestamp": time.Now().Format(time.StampNano)})
	stream.SendHeader(header)

	// Read requests and send responses.
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			fmt.Printf("echo last received message\n")
			return stream.SendAndClose(&pb.HelloReply{Message: "Hello again " + in.GetName()})
		}

		fmt.Printf("request received: %v, building echo\n", in)
		if err != nil {
			return err
		}
	}
}

func (s *server) BidirectionalStreamingEcho(stream pb.Greeter_BidirectionalStreamingEchoServer) error {
	fmt.Printf("--- BidirectionalStreamingEcho ---\n")

	defer func() {
		trailer := metadata.Pairs("timestamp", time.Now().Format(time.StampNano))
		stream.SetTrailer(trailer)
	}()

	// Read metadata from client.
	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		return status.Errorf(codes.DataLoss, "BidirectionalStreamingEcho: failed to get metadata")
	}

	if t, ok := md["timestamp"]; ok {
		fmt.Printf("timestamp from metadata:\n")
		for i, e := range t {
			fmt.Printf(" %d. %s\n", i, e)
		}
	}

	// Create and send header.
	header := metadata.New(map[string]string{"location": "MTV", "timestamp": time.Now().Format(time.StampNano)})
	stream.SendHeader(header)

	// Read requests and send responses.
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		fmt.Printf("request received %v, sending echo\n", in)
		if err := stream.Send(&pb.HelloReply{Message: "Hello again " + in.GetName()}); err != nil {
			return err
		}
	}
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
