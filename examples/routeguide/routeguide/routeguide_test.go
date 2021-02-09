package routeguide

import (
	context "context"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/bakins/simplegrpc"
	"github.com/bakins/simplegrpc/codes"
	"github.com/bakins/simplegrpc/status"
)

func setup(t *testing.T) RouteGuideSimpleClient {
	h := simplegrpc.NewHandler()
	h.RegisterCompressor(simplegrpc.GzipCompressor)

	RegisterRouteGuideSimpleServer(h, &server{})

	svr := httptest.NewServer(h2c.NewHandler(h, &http2.Server{}))

	t.Cleanup(svr.Close)

	conn, err := simplegrpc.NewClientConn(svr.URL)
	require.NoError(t, err)

	return NewRouteGuideSimpleClient(conn)
}

func TestGetFeature(t *testing.T) {
	client := setup(t)

	req := Point{
		Longitude: 1,
		Latitude:  100,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	resp, err := client.GetFeature(ctx, &req)
	require.NoError(t, err)

	require.Equal(t, "testing", resp.Name)
}

func TestListFeatures(t *testing.T) {
	client := setup(t)

	req := Rectangle{
		Lo: &Point{
			Longitude: 1,
			Latitude:  100,
		},
		Hi: &Point{
			Longitude: 1,
			Latitude:  100,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	resp, err := client.ListFeatures(ctx, &req)
	require.NoError(t, err)

	count := 0
	for {
		resp, err := resp.Recv()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		require.Equal(t, "testing", resp.Name)
		count++
	}

	require.Equal(t, 10, count)
}

func TestRecordRoute(t *testing.T) {
	client := setup(t)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	_, err := client.RecordRoute(ctx)
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.Unimplemented, st.Code())
}

func TestRouteChat(t *testing.T) {
	client := setup(t)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	_, err := client.RouteChat(ctx)
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.Unimplemented, st.Code())
}

type server struct{}

func (s *server) GetFeature(ctx context.Context, point *Point) (*Feature, error) {
	return &Feature{
		Name: "testing",
	}, nil
}

func (s *server) ListFeatures(rectangle *Rectangle, simpleServer RouteGuide_ListFeaturesSimpleServer) error {
	f := Feature{
		Name: "testing",
	}

	for i := 0; i < 10; i++ {
		if err := simpleServer.Send(&f); err != nil {
			return err
		}
	}

	return nil
}

func (s *server) RecordRoute(simpleServer RouteGuide_RecordRouteSimpleServer) error {
	count := 0
	for {
		_, err := simpleServer.Recv()
		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		count++
	}

	resp := RouteSummary{
		PointCount: int32(count),
	}

	return simpleServer.SendAndClose(&resp)
}

func (s *server) RouteChat(simpleServer RouteGuide_RouteChatSimpleServer) error {
	for {
		in, err := simpleServer.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		out := RouteNote{
			Location: &Point{
				Latitude: in.Location.Latitude * 100,
			},
		}

		err = simpleServer.Send(&out)
		if err != nil {
			return err
		}

	}

	return nil
}
