package helloworld

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/bakins/simplegrpc"
)

func TestSayHello(t *testing.T) {
	h := simplegrpc.NewHandler()
	h.RegisterCompressor(simplegrpc.GzipCompressor)

	RegisterGreeterSimpleServer(h, &server{})

	svr := httptest.NewServer(h2c.NewHandler(h, &http2.Server{}))
	defer svr.Close()

	conn, err := simplegrpc.NewClientConn(svr.URL)
	require.NoError(t, err)

	client := NewGreeterSimpleClient(conn)

	req := HelloRequest{
		Name: "world",
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	resp, err := client.SayHello(ctx, &req)
	require.NoError(t, err)

	require.Equal(t, "Hello world", resp.Message)

}

type server struct{}

func (s *server) SayHello(ctx context.Context, in *HelloRequest) (*HelloReply, error) {
	return &HelloReply{Message: "Hello " + in.GetName()}, nil
}
