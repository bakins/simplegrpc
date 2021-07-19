package helloworld

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
)

func TestServer(t *testing.T) {
	hw := NewGreeterGRPCServer(&server{})
	mux := http.NewServeMux()

	mux.Handle(hw.PathPrefix(), hw)

	handler := h2c.NewHandler(mux, &http2.Server{})

	svr := httptest.NewServer(handler)
	defer svr.Close()

	conn, err := grpc.Dial(strings.TrimPrefix(svr.URL, "http://"), grpc.WithInsecure())
	require.NoError(t, err)

	client := NewGreeterClient(conn)

	resp, err := client.SayHello(context.Background(), &HelloRequest{Name: "world"})
	require.NoError(t, err)

	fmt.Println(resp)
}

type server struct{}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *HelloRequest) (*HelloReply, error) {
	// fmt.Println("I am in SayHello")
	return &HelloReply{Message: "Hello " + in.GetName()}, nil
	// return nil, errors.New("bad stuff")
}
