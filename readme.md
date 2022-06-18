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

+ 1.在`defer`中调用`SetTrailer`;会在`grpc`关闭时在发送`metadata`数据,可以调用多次，多次调用会合并`metadata`数据

+ 2.调用`FromIncomingContext`;从`context`中解析出`client`请求时携带的`metadata`数据

+ 3.`SendHeader`发送`metadata`到客户端，只能调用一次

+ 4.返回数据到客户端

## 2.2客户端代码
```go
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
```

+ 1.`NewOutgoingContext` 将创建的`metadata`数据放入`context`中，在`rpc`调用时通过`context`将携带的`metadata`发送给服务端

+ 2.创建`header`和`trailer`用于`rpc`调用完成后，得到服务端返回的`metadata`数据,`header`是服务端刚开始调用时填充的数据,`trailer`是服务端调用完成后填充的数据

## 整个流程

+ 1. 客户端调用`metadata.NewOutgoingContext`将`metadata`填充到`context`中;然后调用服务端方法`c.UnaryEcho`,传入`context`

+ 2. 服务端方法被调用时,调用`metadata.FromIncomingContext`从`context`中拿到`metadata`数据

+ 3. 服务端调用`grpc.SendHeader`,发送的`Header`携带`metadata`数据

+ 4. 服务端处理完毕返回时，执行`defer`，调用`grpc.SetTrailer`,设置最后的`metadata`数据

+ 5. 客户端收到返回的数据时，会同时拿到`Header`和`Trailer`，以及内部携带的`metadata`

***
# 3.服务端Stream`metadata`数据使用

## 3.1 服务端代码

```go
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
```
+ 与普通函数调用流程一致，只是将`grpc.`全部换成`stream.`

## 3.2 客户端代码
```
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
```

## 整个流程

+ 1. 客户端客户端调用`metadata.NewOutgoingContext`将`metadata`放入`context`中;调用服务端方法`c.ServerStreamingEcho`,传入`context`

+ 2. 服务端方法被调用,使用`metadata.FromIncomingContext`,取出`context`中的`metadata`

+ 3. 服务端调用`stream.SendHeader`，发送`Header`中携带`metadata`,这个方法只能调用一次

+ 4. 客户端调用`stream.Header()`,然后从header中拿到`metadata`

+ 5. 服务端所有`stream`处理完毕，执行`defer`调用`stream.SetTrailer`，设置`Trailer`中的`metadata`

+ 6. 客户端接受完服务端所有的`stream`后，调用`stream.Trailer()`,从中拿到最后的`metadata`

***

# 4.客户端Stream`metadata`数据使用

## 4.1 服务端代码

```go
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
```

## 4.2 客户端代码

```go
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
```

## 整个流程

+ 1. 客户端客户端调用`metadata.NewOutgoingContext`将`metadata`放入`context`中;接着调用服务端方法`c.ClientStreamingEcho(ctx)`,并且传入`context`

+ 2. 服务端方法被调用，使用`metadata.FromIncomingContext(stream.Context())`,从`context`取到`metadata`

+ 3. 服务端调用`stream.SendHeader`,发送带有`metadata`的`Header`给客户端

+ 4. 客户端调用`stream.Header()`,解析`Header`中的`metadata`

+ 5. 服务端开始接受处理客户端发送的`stream`信息，处理完成后,运行`defer`调用`stream.SetTrailer(trailer)`,设置`Trailer`的`metadata`

+ 6. 客户端收到服务端处理完`stream`的返回后，调用`stream.Trailer()`,拿到最后放在`Trailer`的`metadata`

***
# 5.双向Stream`metadata`数据使用

## 5.1 服务端代码

```go
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
```

## 5.2 客户端代码

```go
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
```

## 整个流程

+ 1. 客户端客户端调用`metadata.NewOutgoingContext`将`metadata`放入`context`中;接着调用服务端方法`c.BidirectionalStreamingEcho(ctx)`,并且传入`context`

+ 2. 服务端方法被调用，使用`metadata.FromIncomingContext(stream.Context())`,从`context`取到`metadata`

+ 3. 服务端调用`stream.SendHeader`,发送带有`metadata`的`Header`给客户端

+ 4. 客户端开启一个协程，调用`stream.Header()`,解析`Header`中的`metadata`，并且开始发送`stream`到服务端

+ 5. 服务端接收到客户端发送的`stream`数据，并通过`stream`响应数据到客户端,接收到客户端的`stream.CloseSend()`后,运行`defer`调用`stream.SetTrailer(trailer)`,设置`Trailer`的`metadata`

+ 6. 客户端处理服务端发送的`stream`的返回后，调用`stream.Trailer()`,拿到最后放在`Trailer`的`metadata`

[源码]()
