// Code generated by protoc-gen-go-grpc-simple. DO NOT EDIT.

package routeguide

import (
	context "context"
	errors "errors"
	simplegrpc "github.com/bakins/simplegrpc"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = simplegrpc.SupportPackageIsVersion1

// RouteGuideSimpleClient is the client API for RouteGuide service.
type RouteGuideSimpleClient interface {
	// A simple RPC.
	//
	// Obtains the feature at a given position.
	//
	// A feature with an empty name is returned if there's no feature at the given
	// position.
	GetFeature(ctx context.Context, in *Point) (*Feature, error)
	// A server-to-client streaming RPC.
	//
	// Obtains the Features available within the given Rectangle.  Results are
	// streamed rather than returned at once (e.g. in a response message with a
	// repeated field), as the rectangle may cover a large area and contain a
	// huge number of features.
	ListFeatures(ctx context.Context, in *Rectangle) (RouteGuide_ListFeaturesSimpleClient, error)
	// A client-to-server streaming RPC.
	//
	// Accepts a stream of Points on a route being traversed, returning a
	// RouteSummary when traversal is completed.
	RecordRoute(ctx context.Context) (RouteGuide_RecordRouteSimpleClient, error)
	// A Bidirectional streaming RPC.
	//
	// Accepts a stream of RouteNotes sent while a route is being traversed,
	// while receiving other RouteNotes (e.g. from other users).
	RouteChat(ctx context.Context) (RouteGuide_RouteChatSimpleClient, error)
}

type routeGuideSimpleClient struct {
	cc simplegrpc.ClientConn
}

func NewRouteGuideSimpleClient(cc simplegrpc.ClientConn) RouteGuideSimpleClient {
	return &routeGuideSimpleClient{cc: cc}
}

func (c *routeGuideSimpleClient) GetFeature(ctx context.Context, in *Point) (*Feature, error) {
	stream, err := c.cc.NewStream(ctx, &_RouteGuide_simple_serviceDesc.Streams[0], "/routeguide.RouteGuide/GetFeature")
	if err != nil {
		return nil, err
	}
	if err := stream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := stream.CloseSend(); err != nil {
		return nil, err
	}
	var out Feature
	if err := stream.RecvMsg(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *routeGuideSimpleClient) ListFeatures(ctx context.Context, in *Rectangle) (RouteGuide_ListFeaturesSimpleClient, error) {
	stream, err := c.cc.NewStream(ctx, &_RouteGuide_simple_serviceDesc.Streams[1], "/routeguide.RouteGuide/ListFeatures")
	if err != nil {
		return nil, err
	}
	x := &routeGuideListFeaturesSimpleClient{ClientStream: stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type RouteGuide_ListFeaturesSimpleClient interface {
	Recv() (*Feature, error)
	simplegrpc.ClientStream
}

type routeGuideListFeaturesSimpleClient struct {
	simplegrpc.ClientStream
}

func (x *routeGuideListFeaturesSimpleClient) Recv() (*Feature, error) {
	var m Feature
	if err := x.ClientStream.RecvMsg(&m); err != nil {
		return nil, err
	}
	return &m, nil
}

func (c *routeGuideSimpleClient) RecordRoute(ctx context.Context) (RouteGuide_RecordRouteSimpleClient, error) {
	stream, err := c.cc.NewStream(ctx, &_RouteGuide_simple_serviceDesc.Streams[2], "/routeguide.RouteGuide/RecordRoute")
	if err != nil {
		return nil, err
	}
	x := &routeGuideRecordRouteSimpleClient{ClientStream: stream}
	return x, nil
}

type RouteGuide_RecordRouteSimpleClient interface {
	Send(*Point) error
	CloseAndRecv() (*RouteSummary, error)
	simplegrpc.ClientStream
}

type routeGuideRecordRouteSimpleClient struct {
	simplegrpc.ClientStream
}

func (x *routeGuideRecordRouteSimpleClient) Send(m *Point) error {
	return x.ClientStream.SendMsg(m)
}

func (x *routeGuideRecordRouteSimpleClient) CloseAndRecv() (*RouteSummary, error) {
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}

	var m RouteSummary
	if err := x.ClientStream.RecvMsg(&m); err != nil {
		return nil, err
	}
	return &m, nil
}

func (c *routeGuideSimpleClient) RouteChat(ctx context.Context) (RouteGuide_RouteChatSimpleClient, error) {
	stream, err := c.cc.NewStream(ctx, &_RouteGuide_simple_serviceDesc.Streams[3], "/routeguide.RouteGuide/RouteChat")
	if err != nil {
		return nil, err
	}
	x := &routeGuideRouteChatSimpleClient{ClientStream: stream}
	return x, nil
}

type RouteGuide_RouteChatSimpleClient interface {
	Send(*RouteNote) error
	Recv() (*RouteNote, error)
	simplegrpc.ClientStream
}

type routeGuideRouteChatSimpleClient struct {
	simplegrpc.ClientStream
}

func (x *routeGuideRouteChatSimpleClient) Send(m *RouteNote) error {
	return x.ClientStream.SendMsg(m)
}

func (x *routeGuideRouteChatSimpleClient) Recv() (*RouteNote, error) {
	var m RouteNote
	if err := x.ClientStream.RecvMsg(&m); err != nil {
		return nil, err
	}
	return &m, nil
}

// RouteGuideSimpleServer is the simple server API for RouteGuide service.
type RouteGuideSimpleServer interface {
	// A simple RPC.
	//
	// Obtains the feature at a given position.
	//
	// A feature with an empty name is returned if there's no feature at the given
	// position.
	GetFeature(context.Context, *Point) (*Feature, error)
	// A server-to-client streaming RPC.
	//
	// Obtains the Features available within the given Rectangle.  Results are
	// streamed rather than returned at once (e.g. in a response message with a
	// repeated field), as the rectangle may cover a large area and contain a
	// huge number of features.
	ListFeatures(*Rectangle, RouteGuide_ListFeaturesSimpleServer) error
	// A client-to-server streaming RPC.
	//
	// Accepts a stream of Points on a route being traversed, returning a
	// RouteSummary when traversal is completed.
	RecordRoute(RouteGuide_RecordRouteSimpleServer) error
	// A Bidirectional streaming RPC.
	//
	// Accepts a stream of RouteNotes sent while a route is being traversed,
	// while receiving other RouteNotes (e.g. from other users).
	RouteChat(RouteGuide_RouteChatSimpleServer) error
}

func RegisterRouteGuideSimpleServer(s simplegrpc.ServiceRegistrar, srv RouteGuideSimpleServer) {
	s.RegisterService(&_RouteGuide_simple_serviceDesc, srv)
}

func _RouteGuide_GetFeature_Simple_Handler(srv interface{}, stream simplegrpc.ServerStream) error {
	impl, ok := srv.(RouteGuideSimpleServer)
	if !ok {
		return errors.New("invalid server type - expected RouteGuideSimpleServer")
	}
	var in Point
	if err := stream.RecvMsg(&in); err != nil {
		return err
	}
	out, err := impl.GetFeature(stream.Context(), &in)
	if err != nil {
		return err
	}
	return stream.SendMsg(out)
}

func _RouteGuide_ListFeatures_Simple_Handler(srv interface{}, stream simplegrpc.ServerStream) error {
	m := new(Rectangle)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(RouteGuideSimpleServer).ListFeatures(m, &routeGuideListFeaturesServer{stream})
}

type RouteGuide_ListFeaturesSimpleServer interface {
	Send(*Feature) error
	simplegrpc.ServerStream
}

type routeGuideListFeaturesServer struct {
	simplegrpc.ServerStream
}

func (x *routeGuideListFeaturesServer) Send(m *Feature) error {
	return x.ServerStream.SendMsg(m)
}

func _RouteGuide_RecordRoute_Simple_Handler(srv interface{}, stream simplegrpc.ServerStream) error {
	return srv.(RouteGuideSimpleServer).RecordRoute(&routeGuideRecordRouteServer{stream})
}

type RouteGuide_RecordRouteSimpleServer interface {
	SendAndClose(*RouteSummary) error
	Recv() (*Point, error)
	simplegrpc.ServerStream
}

type routeGuideRecordRouteServer struct {
	simplegrpc.ServerStream
}

func (x *routeGuideRecordRouteServer) SendAndClose(m *RouteSummary) error {
	return x.ServerStream.SendMsg(m)
}

func (x *routeGuideRecordRouteServer) Recv() (*Point, error) {
	m := new(Point)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _RouteGuide_RouteChat_Simple_Handler(srv interface{}, stream simplegrpc.ServerStream) error {
	return srv.(RouteGuideSimpleServer).RouteChat(&routeGuideRouteChatServer{stream})
}

type RouteGuide_RouteChatSimpleServer interface {
	Send(*RouteNote) error
	Recv() (*RouteNote, error)
	simplegrpc.ServerStream
}

type routeGuideRouteChatServer struct {
	simplegrpc.ServerStream
}

func (x *routeGuideRouteChatServer) Send(m *RouteNote) error {
	return x.ServerStream.SendMsg(m)
}

func (x *routeGuideRouteChatServer) Recv() (*RouteNote, error) {
	m := new(RouteNote)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

var _RouteGuide_simple_serviceDesc = simplegrpc.ServiceDesc{
	ServiceName: "routeguide.RouteGuide",
	HandlerType: (*RouteGuideSimpleServer)(nil),
	Streams: []simplegrpc.StreamDesc{
		{
			StreamName:    "GetFeature",
			Handler:       _RouteGuide_GetFeature_Simple_Handler,
			ServerStreams: false,
			ClientStreams: false,
		},
		{
			StreamName:    "ListFeatures",
			Handler:       _RouteGuide_ListFeatures_Simple_Handler,
			ServerStreams: true,
			ClientStreams: false,
		},
		{
			StreamName:    "RecordRoute",
			Handler:       _RouteGuide_RecordRoute_Simple_Handler,
			ServerStreams: false,
			ClientStreams: true,
		},
		{
			StreamName:    "RouteChat",
			Handler:       _RouteGuide_RouteChat_Simple_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "routeguide.proto",
}
