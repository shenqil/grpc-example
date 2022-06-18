# metadata 使用

# 1.修改 `helloworld.proto`

```proto
syntax = "proto3";

option go_package = "./helloworld";

package helloworld;

service Greeter {
  // 普通调用
  rpc UnaryEcho (HelloRequest) returns (HelloReply) {}
  // 服务流调用
  rpc ServerStreamingEcho(HelloRequest) returns (stream HelloReply) {}
  // 客户端流调用
  rpc ClientStreamingEcho(stream HelloRequest) returns (HelloReply) {}
  // 双向流调用
  rpc BidirectionalStreamingEcho(stream HelloRequest) returns (stream HelloReply) {}
}

message HelloRequest {
  string name = 1;
}

message HelloReply {
  string message = 1;
}
```

***

# 2.普通调用`metadata`数据使用

## 2.1服务端代码

```go
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
```

+ 1.在`defer`中调用`SetTrailer`;会在`grpc`关闭时在尾部添加`metadata`数据

+ 2.调用`FromIncomingContext`;从`context`中解析出`client`请求时携带的`metadata`数据

+ 3.`SendHeader`在响应头部添加`metadata`

+ 4.返回数据到客户端

## 2.2客户端代码
```go
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
```

+ 1.`NewOutgoingContext` 将创建的`metadata`数据放入`context`中，在`rpc`调用时通过`context`将携带的`metadata`发送给服务端

+ 2.创建`header`和`trailer`用于`rpc`调用完成后，得到服务端返回的`metadata`数据,`header`是服务端刚开始调用时填充的数据,`trailer`是服务端调用完成后填充的数据