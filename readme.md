# 1. golang环境处理

## 1.1. 安装[golang环境](https://golang.org/)

## 1.2.下载 [protoc](https://github.com/protocolbuffers/protobuf/releases) 并且解压。然后将其添加到系统环境变量中

## 1.3.下载协议编译器的Go 插件
```
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2
```
***

# 2.开始helloworld项目
## 2.1 `go mod init helloworld` 开启一个项目

## 2.2 安装 `grpc` mod
```
go get google.golang.org/grpc
```

## 2.3 创建 `helloworld\helloworld.proto` 文件
```
syntax = "proto3";

option go_package = "./helloworld";

package helloworld;

service Greeter {
  rpc SayHello (HelloRequest) returns (HelloReply) {}
}

message HelloRequest {
  string name = 1;
}

message HelloReply {
  string message = 1;
}
```

## 2.4 编译 `.proto` 文件
```
protoc --go_out=. --go_opt=paths=source_relative  --go-grpc_out=. --go-grpc_opt=paths=source_relative  helloworld/helloworld.proto
```

## 2.5 创建服务端文件 `server\main.go`
```
package main

import (
	"context"
	"log"
	"net"

	"google.golang.org/grpc"

	pb "helloworld/helloworld"
)

type server struct {
	pb.UnimplementedGreeterServer
}

func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
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
```

## 2.6 创建客户端文件`client\main.go`
```
package main

import (
	"context"
	"log"
	"time"

	pb "helloworld/helloworld"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.Dial(":50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewGreeterClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	r, err := c.SayHello(ctx, &pb.HelloRequest{Name: "hello"})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}

	log.Printf("Greeting: %s", r.GetMessage())
}
```

# 总结
+ `go run main.go` 分别启动服务端和客户端，grpc 环境以及helloworld完成

[源码](https://github.com/shenqil/grpc-example)
