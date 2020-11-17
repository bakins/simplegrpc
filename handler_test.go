package simplegrpc

/*
func TestUnary(t *testing.T) {
	h := &Handler{
		codecs:      make(map[string]Codec),
		compressors: make(map[string]Compressor),
	}

	h.codecs[ProtoCodec.Name()] = ProtoCodec
	h.compressors[GzipCompressor.Name()] = GzipCompressor

	helloworld.RegisterSimpleGreeterServer(h, &server{})

	svr := httptest.NewServer(h2c.NewHandler(h, &http2.Server{}))
	defer svr.Close()

	u, err := url.Parse(svr.URL)
	require.NoError(t, err)

	conn, err := grpc.Dial(u.Host, grpc.WithInsecure(), grpc.WithBlock())
	require.NoError(t, err)

	client := pb.NewGreeterClient(conn)

	req := pb.HelloRequest{
		Name: "world",
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	resp, err := client.SayHello(ctx, &req, grpc.UseCompressor(gzip.Name))
	require.NoError(t, err)

	require.Equal(t, "Hello world", resp.Message)

}

type server struct{}

func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	return &pb.HelloReply{Message: "Hello " + in.GetName()}, nil
}

type GreeterServer interface {
	SayHello(context.Context, *pb.HelloRequest) (*pb.HelloReply, error)
}

func RegisterSimpleGreeterServer(h *Handler, srv GreeterServer) {
	h.RegisterService(&_Greeter_serviceDesc, srv)
}

func _Greeter_SayHello_Handler(srv interface{}, stream ServerStream) error {
	impl, ok := srv.(GreeterServer)
	if !ok {
		return fmt.Errorf("invalid server type")
	}

	var in pb.HelloRequest
	if err := stream.RecvMsg(&in); err != nil {
		return err
	}

	out, err := impl.SayHello(stream.Context(), &in)
	if err != nil {
		return err
	}

	return stream.SendMsg(out)
}

var _Greeter_serviceDesc = ServiceDesc{
	ServiceName: "helloworld.Greeter",
	HandlerType: (*GreeterServer)(nil),
	Streams: []StreamDesc{
		{
			StreamName: "SayHello",
			Handler:    _Greeter_SayHello_Handler,
		},
	},
	Metadata: "examples/helloworld/helloworld/helloworld.proto",
}

*/
