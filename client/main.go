package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"time"

	pb "helloworld/helloworld"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func main() {
	conn, err := grpc.Dial(
		":50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(unaryInterceptor),
		grpc.WithStreamInterceptor(streamInterceptor),
	)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewGreeterClient(conn)

	// unaryCallWithMetadata(c)
	// serverStreamingWithMetadata(c)
	// clientStreamWithMetadata(c)
	bidirectionalWithMetadata(c)
}

func unaryCallWithMetadata(c pb.GreeterClient) {
	fmt.Println("--- unaryCall ---")

	r, err := c.UnaryEcho(context.Background(), &pb.HelloRequest{Name: "unaryCall"})
	if err != nil {
		log.Fatalf("调用UnaryEcho失败:%v", err)
	}

	fmt.Println("response:")
	fmt.Printf(" - %s\n", r.Message)
}

func serverStreamingWithMetadata(c pb.GreeterClient) {
	fmt.Printf("--- server streaming ---\n")

	stream, err := c.ServerStreamingEcho(context.Background(), &pb.HelloRequest{Name: "serverStreamingWithMetadata"})
	if err != nil {
		log.Fatalf("调用ServerStreamingEcho失败: %v", err)
	}

	// 读取所有的responses
	fmt.Printf("response:\n")
	var rpcStatus error
	for {
		r, err := stream.Recv()
		if err != nil {
			rpcStatus = err
			break
		}

		fmt.Printf(" - %s\n", r.Message)
	}

	if rpcStatus != io.EOF {
		log.Fatalf("failed to finish server streaming: %v", rpcStatus)
	}

	// 解析trailer
	trailer := stream.Trailer()

	if t, ok := trailer["timestamp"]; ok {
		fmt.Printf("timestamp from trailer:\n")
		for i, e := range t {
			fmt.Printf(" %d. %s\n", i, e)
		}
	} else {
		log.Fatal("需要timestamp，但header中不存在timestamp")
	}
}

func clientStreamWithMetadata(c pb.GreeterClient) {
	fmt.Printf("--- client streaming ---\n")

	// Make RPC using the context with the metadata.
	stream, err := c.ClientStreamingEcho(context.Background())
	if err != nil {
		log.Fatalf("ClientStreamingEcho 调用失败: %v\n", err)
	}

	// Send all requests to the server.
	for i := 0; i < 10; i++ {
		if err := stream.Send(&pb.HelloRequest{Name: "clientStreamWithMetadata"}); err != nil {
			log.Fatalf("failed to send streaming: %v\n", err)
		}
	}

	// Read the response.
	r, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatalf("failed to CloseAndRecv: %v\n", err)
	}
	fmt.Printf("response:\n")
	fmt.Printf(" - %s\n\n", r.Message)

	// 解析trailer
	trailer := stream.Trailer()

	if t, ok := trailer["timestamp"]; ok {
		fmt.Printf("timestamp from trailer:\n")
		for i, e := range t {
			fmt.Printf(" %d. %s\n", i, e)
		}
	} else {
		log.Fatal("需要timestamp，但header中不存在timestamp")
	}
}

func bidirectionalWithMetadata(c pb.GreeterClient) {
	fmt.Printf("--- bidirectional ---\n")

	// Make RPC using the context with the metadata.
	stream, err := c.BidirectionalStreamingEcho(context.Background())
	if err != nil {
		log.Fatalf("failed to call BidirectionalStreamingEcho: %v\n", err)
	}

	go func() {
		// Send all requests to the server.
		for i := 0; i < 10; i++ {
			if err := stream.Send(&pb.HelloRequest{Name: "clientStreamWithMetadata"}); err != nil {
				log.Fatalf("failed to send streaming: %v\n", err)
			}
		}
		stream.CloseSend()
	}()

	// Read all the responses.
	var rpcStatus error
	fmt.Printf("response:\n")
	for {
		r, err := stream.Recv()
		if err != nil {
			rpcStatus = err
			break
		}
		fmt.Printf(" - %s\n", r.Message)
	}
	if rpcStatus != io.EOF {
		log.Fatalf("failed to finish server streaming: %v", rpcStatus)
	}

	// 解析trailer
	trailer := stream.Trailer()

	if t, ok := trailer["timestamp"]; ok {
		fmt.Printf("timestamp from trailer:\n")
		for i, e := range t {
			fmt.Printf(" %d. %s\n", i, e)
		}
	} else {
		log.Fatal("需要timestamp，但header中不存在timestamp")
	}
}

// unaryInterceptor is an example unary interceptor.
func unaryInterceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	fmt.Printf("---unaryInterceptor---\n")

	// 创建metadata到context中.
	md := metadata.Pairs("timestamp", time.Now().Format(time.StampNano))
	ctx = metadata.NewOutgoingContext(ctx, md)

	reqStr, _ := json.Marshal(req)
	fmt.Printf("RPC: %s,req:%s\n", method, reqStr)

	var header, trailer metadata.MD
	opts = append(opts, grpc.Header(&header), grpc.Trailer(&trailer))

	err := invoker(ctx, method, req, reply, cc, opts...)

	if t, ok := header["timestamp"]; ok {
		fmt.Printf("timestamp from header:\n")
		for i, e := range t {
			fmt.Printf("%d.%s\n", i, e)
		}
	} else {
		log.Fatal("需要timestamp，但header中不存在timestamp")
	}

	if l, ok := header["location"]; ok {
		fmt.Printf("location from header:\n")
		for i, e := range l {
			fmt.Printf(" %d. %s\n", i, e)
		}
	} else {
		log.Fatal("需要location，但是header中不存在location")
	}

	if t, ok := trailer["timestamp"]; ok {
		fmt.Printf("timestamp from trailer:\n")
		for i, e := range t {
			fmt.Printf(" %d. %s\n", i, e)
		}
	} else {
		log.Fatal("需要timestamp，但header中不存在timestamp")
	}

	return err
}

// streamInterceptor is an example stream interceptor.
func streamInterceptor(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	fmt.Printf("---streamInterceptor---\n")

	// 创建metadata到context中.
	md := metadata.Pairs("timestamp", time.Now().Format(time.StampNano))
	ctx = metadata.NewOutgoingContext(ctx, md)

	// 执行具体业务
	s, err := streamer(ctx, desc, cc, method, opts...)
	if err != nil {
		return nil, err
	}

	// 解析 header
	header, err := s.Header()
	if err != nil {
		log.Fatalf("无法从stream中获取header: %v", err)
	}

	if t, ok := header["timestamp"]; ok {
		fmt.Printf("timestamp from header:\n")
		for i, e := range t {
			fmt.Printf(" %d. %s\n", i, e)
		}
	} else {
		log.Fatal("需要timestamp，但header中不存在timestamp")
	}
	if l, ok := header["location"]; ok {
		fmt.Printf("location from header:\n")
		for i, e := range l {
			fmt.Printf(" %d. %s\n", i, e)
		}
	} else {
		log.Fatal("需要location，但是header中不存在location")
	}

	return s, nil
}
