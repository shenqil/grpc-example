# Interceptor 使用

上一篇我们介绍了[metadata](https://github.com/shenqil/grpc-example/tree/metadata)的使用方法，但是我们在每个方法内部都需要设置相同重复的`metadata`,比如调用时间戳，调用链等；能不能把这些相同的重复性设置，统一放在一个地方，方便后面修改和维护，答案就是拦截器-`Interceptor`.

# 1.普通调用 `Interceptor`的使用
## 1.1 服务端修改后代码

+ 服务端拦截器代码

```go
func unaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	fmt.Println("---unaryInterceptor---")

	// 解析请求携带的信息
	str, _ := json.Marshal(req)
	fmt.Printf("req: %s\n", str)
	fmt.Printf("Method: %s\n", info.FullMethod)

	defer func() {
		trailer := metadata.Pairs("timestamp", time.Now().Format(time.StampNano))
		grpc.SetTrailer(ctx, trailer)
	}()

	// 解析请求的metadata
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

	// 创建携带metadata的Header
	header := metadata.New(map[string]string{"location": "MTV", "timestamp": time.Now().Format(time.StampNano)})
	grpc.SendHeader(ctx, header)

	// 方法调用
	m, err := handler(ctx, req)
	if err != nil {
		fmt.Printf("RPC failed with error %v", err)
	}
	return m, err
}
```

+ 服务端Handle代码

```
func (s *server) UnaryEcho(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	fmt.Println("---UnaryEcho---")

	fmt.Printf("已接受到的请求:%v,发送响应\n", in)

	return &pb.HelloReply{Message: "Hello again " + in.GetName()}, nil
}

```

+ 1. 在拦截器里面我们可以打印出调用方法名和调用方法时，请求的参数

+ 2. 从`context`中解析`metadata`

+ 3. 设置`Header`里面的`metadata`

+ 4. 调用业务处理handle

+ 5. `defer`时,设置`Trailer`里面的`metadata`

## 1.2客户端修改后代码

+ 客户端拦截器代码

```go
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
```
+ 客户端Handle代码

```go
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
```

+ 1.创建`metadata`并且放入`context`中

+ 2.打印请求方法名和请求方法参数

+ 3.定义用于存放服务端返回`header`, `trailer`

+ 4.调用业务处理handle

+ 5.解析`header`, `trailer`

## 总结

+ 可以看到用了`Interceptor`之后，我们业务代码变的很干净，只用关心业务层面的逻辑
+ 再拦截器里我们可以加入公共逻辑，log,错误处理，以及recover

***

# 2.stream调用 `Interceptor`的使用

## 2.1 服务端修改后代码
### 拦截器代码
```go
func streamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	fmt.Printf("--- streamInterceptor ---\n")

	// 调用完成时设置SetTrailer
	defer func() {
		trailer := metadata.Pairs("timestamp", time.Now().Format(time.StampNano))
		ss.SetTrailer(trailer)
	}()

	// 从Stream的Context中解析出metadata
	md, ok := metadata.FromIncomingContext(ss.Context())
	if !ok {
		return status.Errorf(codes.DataLoss, "ServerStreamingEcho: 无法获取metadata")
	}
	if t, ok := md["timestamp"]; ok {
		fmt.Printf("timestamp from metadata:\n")
		for i, e := range t {
			fmt.Printf("%d.%s\n", i, e)
		}
	}

	// 设置Header里面的metadata
	header := metadata.New(map[string]string{"location": "MTV", "timestamp": time.Now().Format(time.StampNano)})
	ss.SendHeader(header)

	err := handler(srv, ss)
	if err != nil {
		fmt.Printf("RPC failed with error %v", err)
	}
	return err
}
```

### 服务端双向流Handle代码

```go
func (s *server) BidirectionalStreamingEcho(stream pb.Greeter_BidirectionalStreamingEchoServer) error {
	fmt.Printf("--- BidirectionalStreamingEcho ---\n")

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

+ 1.`stream`模式下调用与普通调用修改逻辑基本一直，可以将所有的`metadata`操作都放入`Interceptor`中

## 2.2 客户端修改后代码
### 客户端拦截器代码
```go
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
```
### 客户端双向流Handle代码
```go
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
```

+ 1. `stream.Trailer`是不能放入`Interceptor`中的，只能写在对应的handle中，否则会报error：**RPC failed with error rpc error: code = Canceled desc = context canceled**

***

[源码](https://github.com/shenqil/grpc-example/tree/interceptor)