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
	)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewGreeterClient(conn)

	unaryCallWithMetadata(c)
	// serverStreamingWithMetadata(c)
	// clientStreamWithMetadata(c)
	// bidirectionalWithMetadata(c)
}

func unaryCallWithMetadata(c pb.GreeterClient) {
	fmt.Println("--- unaryCall ---")

	// 使用metadata的上下文创建RPC

	r, err := c.UnaryEcho(context.Background(), &pb.HelloRequest{Name: "unaryCall"})
	if err != nil {
		log.Fatalf("调用UnaryEcho失败:%v", err)
	}

	fmt.Println("response:")
	fmt.Printf(" - %s\n", r.Message)
}

func serverStreamingWithMetadata(c pb.GreeterClient) {
	fmt.Printf("--- server streaming ---\n")

	// 创建metadata到context中.
	md := metadata.Pairs("timestamp", time.Now().Format(time.StampNano))
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	stream, err := c.ServerStreamingEcho(ctx, &pb.HelloRequest{Name: "serverStreamingWithMetadata"})
	if err != nil {
		log.Fatalf("调用ServerStreamingEcho失败: %v", err)
	}

	header, err := stream.Header()
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
	// Create metadata and context.
	md := metadata.Pairs("timestamp", time.Now().Format(time.StampNano))
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	// Make RPC using the context with the metadata.
	stream, err := c.ClientStreamingEcho(ctx)
	if err != nil {
		log.Fatalf("ClientStreamingEcho 调用失败: %v\n", err)
	}

	// Read the header when the header arrives.
	header, err := stream.Header()
	if err != nil {
		log.Fatalf("failed to get header from stream: %v", err)
	}
	// Read metadata from server's header.
	if t, ok := header["timestamp"]; ok {
		fmt.Printf("timestamp from header:\n")
		for i, e := range t {
			fmt.Printf(" %d. %s\n", i, e)
		}
	} else {
		log.Fatal("timestamp expected but doesn't exist in header")
	}

	if l, ok := header["location"]; ok {
		fmt.Printf("location from header:\n")
		for i, e := range l {
			fmt.Printf(" %d. %s\n", i, e)
		}
	} else {
		log.Fatal("location expected but doesn't exist in header")
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

	// Read the trailer after the RPC is finished.
	trailer := stream.Trailer()
	// Read metadata from server's trailer.
	if t, ok := trailer["timestamp"]; ok {
		fmt.Printf("timestamp from trailer:\n")
		for i, e := range t {
			fmt.Printf(" %d. %s\n", i, e)
		}
	} else {
		log.Fatal("timestamp expected but doesn't exist in trailer")
	}
}

func bidirectionalWithMetadata(c pb.GreeterClient) {
	fmt.Printf("--- bidirectional ---\n")
	// Create metadata and context.
	md := metadata.Pairs("timestamp", time.Now().Format(time.StampNano))
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	// Make RPC using the context with the metadata.
	stream, err := c.BidirectionalStreamingEcho(ctx)
	if err != nil {
		log.Fatalf("failed to call BidirectionalStreamingEcho: %v\n", err)
	}

	go func() {
		// Read the header when the header arrives.
		header, err := stream.Header()
		if err != nil {
			log.Fatalf("failed to get header from stream: %v", err)
		}
		// Read metadata from server's header.
		if t, ok := header["timestamp"]; ok {
			fmt.Printf("timestamp from header:\n")
			for i, e := range t {
				fmt.Printf(" %d. %s\n", i, e)
			}
		} else {
			log.Fatal("timestamp expected but doesn't exist in header")
		}
		if l, ok := header["location"]; ok {
			fmt.Printf("location from header:\n")
			for i, e := range l {
				fmt.Printf(" %d. %s\n", i, e)
			}
		} else {
			log.Fatal("location expected but doesn't exist in header")
		}

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

	// Read the trailer after the RPC is finished.
	trailer := stream.Trailer()
	// Read metadata from server's trailer.
	if t, ok := trailer["timestamp"]; ok {
		fmt.Printf("timestamp from trailer:\n")
		for i, e := range t {
			fmt.Printf(" %d. %s\n", i, e)
		}
	} else {
		log.Fatal("timestamp expected but doesn't exist in trailer")
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
