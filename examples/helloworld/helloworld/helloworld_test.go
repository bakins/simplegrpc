package helloworld

import (
	"context"
	"net"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/encoding/gzip"

	"github.com/bakins/simplegrpc"
	"github.com/bakins/simplegrpc/codes"
	"github.com/bakins/simplegrpc/status"
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

type server struct {
	code codes.Code
	UnimplementedGreeterServer
}

func (s *server) SayHello(ctx context.Context, in *HelloRequest) (*HelloReply, error) {
	if s.code != codes.OK {
		return nil, status.Error(s.code, s.code.String())
	}
	return &HelloReply{Message: "Hello " + in.GetName()}, nil
}

func TestSayHelloClient(t *testing.T) {
	tests := []struct {
		name string
		code codes.Code
	}{
		{
			name: "ok",
			code: codes.OK,
		},
		{
			name: "not found",
			code: codes.NotFound,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			lis, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)

			g := grpc.NewServer()

			RegisterGreeterServer(g, &server{code: test.code})

			defer func() {
				g.Stop()
				_ = lis.Close()
			}()

			go func() {
				err := g.Serve(lis)
				assert.NoError(t, err)
			}()

			conn, err := simplegrpc.NewClientConn("http://"+lis.Addr().String(), simplegrpc.WithCompressor(simplegrpc.GzipCompressor))
			require.NoError(t, err)

			client := NewGreeterSimpleClient(conn)

			req := HelloRequest{
				Name: "world",
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			defer cancel()

			resp, err := client.SayHello(ctx, &req)

			if test.code != codes.OK {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, test.code, st.Code())
				return
			}
			require.NoError(t, err)

			require.Equal(t, "Hello world", resp.Message)
		})
	}
}

func TestSayHelloServer(t *testing.T) {
	h := simplegrpc.NewHandler()
	RegisterGreeterSimpleServer(h, &server{})

	svr := httptest.NewServer(h2c.NewHandler(h, &http2.Server{}))
	defer svr.Close()

	u, err := url.Parse(svr.URL)
	require.NoError(t, err)

	conn, err := grpc.Dial(u.Host, grpc.WithInsecure())
	require.NoError(t, err)

	client := NewGreeterClient(conn)

	req := HelloRequest{
		Name: "world",
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	resp, err := client.SayHello(ctx, &req)
	require.NoError(t, err)

	require.Equal(t, "Hello world", resp.Message)
}
