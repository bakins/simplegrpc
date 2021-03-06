package simplegrpc

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/bakins/simplegrpc/codes"
	"github.com/bakins/simplegrpc/status"
	"golang.org/x/net/http2"
)

// ClientConn ...
type ClientConn interface {
	NewStream(ctx context.Context, desc *StreamDesc, method string) (ClientStream, error)
}

type clientConn struct {
	request     *http.Request
	compressor  Compressor
	codec       Codec
	transport   http.RoundTripper
	interceptor StreamClientInterceptor
}

// TransportForEndpoint returns an HTTP/2 transport to be used with the endpoint
// It uses h2c for http endpoints
func TransportForEndpoint(endpoint string) (*http2.Transport, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	if strings.ToLower(u.Scheme) == "http" {
		return &http2.Transport{
			AllowHTTP: true,
			DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
				return net.Dial(network, addr)
			},
		}, nil
	}

	return &http2.Transport{}, nil
}

// TransportWrapper wraps a RoundTripper
type TransportWrapper func(http.RoundTripper) http.RoundTripper

type Options struct {
	transport   http.RoundTripper
	wrapper     TransportWrapper
	codec       Codec
	compressor  Compressor
	interceptor StreamClientInterceptor
}

type Option func(*Options)

// WithTransport sets the transport. No wrappers are called.
// If no transport is set, TransportForEndpoint is used and the wrapper is called.
// Note: http.DefaultTransport does not work with h2c and client streams may not work as expected with it.
func WithTransport(transport http.RoundTripper) Option {
	return func(o *Options) {
		o.transport = transport
	}
}

// WithTransportWrapper sets the transport wrapper. called to wrap the internal default transport.
// Not called if WithTransport is used
func WithTransportWrapper(wrapper TransportWrapper) Option {
	return func(o *Options) {
		o.wrapper = wrapper
	}
}

// WithCodec sets the codec to use. Default is proto
func WithCodec(codec Codec) Option {
	return func(o *Options) {
		o.codec = codec
	}
}

// WithCompressor sets the compressor to use. There is no default
func WithCompressor(compressor Compressor) Option {
	return func(o *Options) {
		o.compressor = compressor
	}
}

// StreamClientInterceptor intercepts the creation of a ClientStream.
type StreamClientInterceptor func(ctx context.Context, desc *StreamDesc, cc ClientConn, method string, streamer Streamer) (ClientStream, error)

// Streamer is called by StreamClientInterceptor to create a ClientStream.
type Streamer func(ctx context.Context, desc *StreamDesc, cc ClientConn, method string) (ClientStream, error)

// NewClientConn creates a new clientconn
func NewClientConn(endpoint string, options ...Option) (ClientConn, error) {
	request, err := http.NewRequest(http.MethodPost, endpoint, nil)
	if err != nil {
		return nil, err
	}

	request.Header.Del("Content-Length")
	request.ContentLength = -1
	request.ProtoMinor = 2
	request.ProtoMinor = 0

	c := &clientConn{
		codec:   ProtoCodec,
		request: request,
	}

	opts := Options{}
	for _, o := range options {
		o(&opts)
	}

	var transport http.RoundTripper

	if opts.transport != nil {
		transport = opts.transport
	} else {
		t, err := TransportForEndpoint(endpoint)
		if err != nil {
			return nil, err
		}

		transport = t

		if opts.wrapper != nil {
			transport = opts.wrapper(transport)
		}
	}

	c.transport = transport

	if opts.codec != nil {
		c.codec = opts.codec
	}

	c.compressor = opts.compressor
	if c.compressor != nil {
		c.request.Header.Set("Grpc-Encoding", c.compressor.Name())
	}

	c.request.Header.Set("Content-Type", baseContentType+"+"+c.codec.Name())

	return c, nil
}

// ClientStream ...
type ClientStream interface {
	Context() context.Context
	SendMsg(m interface{}) error
	RecvMsg(m interface{}) error
}

// unary request streaming response
type unaryStreamRequest struct {
	ctx         context.Context
	clientConn  *clientConn
	request     *http.Request
	response    *http.Response
	requestSent chan struct{}
}

// unary request unary response
type unaryUnaryRequest struct {
	unaryStreamRequest
}

func (c *clientConn) NewStream(ctx context.Context, desc *StreamDesc, method string) (ClientStream, error) {
	if desc.ClientStreams {
		return nil, errors.New("client streams are not currently supported")
	}

	// TODO: ensure resp body is closed always
	if c.interceptor == nil {
		return clientStreamer(ctx, desc, c, method)
	}

	return c.interceptor(ctx, desc, c, method, clientStreamer)
}

func clientStreamer(ctx context.Context, desc *StreamDesc, cc ClientConn, method string) (ClientStream, error) {
	c, ok := cc.(*clientConn)
	if !ok {
		return nil, errors.New("unexpected type passed to streamer")
	}

	request := c.request.Clone(ctx)
	request.URL.Path = method

	if desc.ServerStreams {
		return &unaryStreamRequest{
			ctx:         ctx,
			clientConn:  c,
			request:     request,
			requestSent: make(chan struct{}),
		}, nil
	}

	return &unaryUnaryRequest{
		unaryStreamRequest: unaryStreamRequest{
			ctx:         ctx,
			clientConn:  c,
			request:     request,
			requestSent: make(chan struct{}),
		},
	}, nil
}

// close body after receiving single message
func (u *unaryUnaryRequest) RecvMsg(message interface{}) error {
	err := u.unaryStreamRequest.RecvMsg(message)
	u.closeRecv()
	return err
}

func (u *unaryStreamRequest) Context() context.Context {
	return u.ctx
}

func (u *unaryStreamRequest) closeRecv() {
	_, _ = io.Copy(io.Discard, u.response.Body)
	_ = u.response.Body.Close()
}

func (u *unaryStreamRequest) SendMsg(message interface{}) error {
	if u.response != nil {
		return errors.New("SendMsg called multiple times for non-streaming client")
	}

	defer close(u.requestSent)

	var buff bytes.Buffer
	if err := sendMsg(&buff, u.clientConn.codec, u.clientConn.compressor, message); err != nil {
		return err
	}

	u.request.Body = ioutil.NopCloser(&buff)

	client := &http.Client{
		Transport: u.clientConn.transport,
	}

	resp, err := client.Do(u.request)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		u.closeRecv()
		return fmt.Errorf("unexpected http status: %d", resp.StatusCode)
	}

	// TODO: check content type, compression
	u.response = resp

	return nil
}

func (u *unaryStreamRequest) RecvMsg(message interface{}) error {
	select {
	case <-u.ctx.Done():
		return u.ctx.Err()
	case <-u.requestSent:
	}

	if u.request == nil {
		return errors.New("no http response found")
	}

	if err := recvMsg(u.response.Body, u.clientConn.codec, u.clientConn.compressor, message); err != nil {
		u.closeRecv()

		if code := getGrpcStatus(u.response); code != codes.OK {
			msg := getGrpcMessage(u.response)
			if msg == "" {
				msg = code.String()
			}
			return status.Error(code, msg)
		}

		return err
	}

	return nil
}

var (
	grpcStatus  = http.CanonicalHeaderKey("Grpc-Status")
	grpcMessage = http.CanonicalHeaderKey("Grpc-Message")
)

func getGrpcStatus(resp *http.Response) codes.Code {
	v := resp.Header.Get(grpcStatus)
	if v == "0" {
		return codes.OK
	}

	if v == "" {
		v = resp.Trailer.Get(grpcStatus)
	}

	if v == "0" || v == "" {
		return codes.OK
	}

	c, err := strconv.Atoi(v)
	if err != nil {
		// should return error?
		return codes.OK
	}

	return codes.Code(c)
}

func getGrpcMessage(resp *http.Response) string {
	v := resp.Header.Get(grpcMessage)
	if v != "" {
		return v
	}

	return resp.Trailer.Get(grpcMessage)
}
