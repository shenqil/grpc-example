package main

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "helloworld/helloworld"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func main() {
	conn, err := grpc.Dial(":50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewGreeterClient(conn)

	unaryCallWithMetadata(c)
}

func unaryCallWithMetadata(c pb.GreeterClient) {
	fmt.Println("--- unaryCall ---")

	// 创建metadata到context中.
	md := metadata.Pairs("timestamp", time.Now().Format(time.StampNano))
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	// 使用metadata的上下文创建RPC
	var header, trailer metadata.MD
	r, err := c.UnaryEcho(ctx, &pb.HelloRequest{Name: "unaryCall"}, grpc.Header(&header), grpc.Trailer(&trailer))
	if err != nil {
		log.Fatalf("调用UnaryEcho失败:%v", err)
	}

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
	fmt.Println("response:")
	fmt.Printf(" - %s\n", r.Message)

	if t, ok := trailer["timestamp"]; ok {
		fmt.Printf("timestamp from trailer:\n")
		for i, e := range t {
			fmt.Printf(" %d. %s\n", i, e)
		}
	} else {
		log.Fatal("需要timestamp，但header中不存在timestamp")
	}
}
