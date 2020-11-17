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
	"strings"
	"sync"

	"golang.org/x/net/http2"
)

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

	c.request.Header.Set("Content-Type", baseContentType+"+"+c.codec.Name())

	return c, nil
}

type ClientStream interface {
	Context() context.Context
	CloseSend() error
	SendMsg(m interface{}) error
	RecvMsg(m interface{}) error
}

type unaryRequestStream struct {
	ctx         context.Context
	clientConn  *clientConn
	request     *http.Request
	response    *http.Response
	requestSent chan struct{}
}

func (c *clientConn) NewStream(ctx context.Context, desc *StreamDesc, method string) (ClientStream, error) {
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

	if !desc.ClientStreams {
		return &unaryRequestStream{
			ctx:         ctx,
			clientConn:  c,
			request:     request,
			requestSent: make(chan struct{}),
		}, nil
	}

	return newStreamingRequestStream(ctx, c, request), nil
}

type unaryUnaryStream struct {
	ctx        context.Context
	clientConn *clientConn
	request    *http.Request
	readReady  chan struct{}
	reader     io.ReadCloser
}

func (u *unaryUnaryStream) Context() context.Context {
	return u.ctx
}

func (u *unaryUnaryStream) CloseSend() error {
	return nil
}

func (u *unaryUnaryStream) SendMsg(message interface{}) error {
	select {
	case <-u.readReady:
		return errors.New("SendMsg called multiple times for non-streaming client")
	default:

	}

	defer func() {
		close(u.readReady)
	}()

	var buff bytes.Buffer
	if err := sendMsg(&buff, u.clientConn.codec, u.clientConn.compressor, message); err != nil {
		close(u.readReady)
		return err
	}

	u.request.Body = ioutil.NopCloser(&buff)

	client := &http.Client{
		Transport: u.clientConn.transport,
	}

	resp, err := client.Do(u.request)
	if err != nil {
		close(u.readReady)
		return err
	}

	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		close(u.readReady)
		return fmt.Errorf("unexpected http status: %d", resp.StatusCode)
	}

	// TODO: check content type, compression
	// create a helper function

	u.reader = resp.Body

	close(u.readReady)

	return nil
}

func (u *unaryRequestStream) Context() context.Context {
	return u.ctx
}

func (u *unaryRequestStream) CloseSend() error {
	return nil
}

func (u *unaryRequestStream) SendMsg(message interface{}) error {
	if u.response != nil {
		return errors.New("SendMsg called multiple times for non-streaming client")
	}

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
		close(u.requestSent)
		return err
	}

	if resp.StatusCode != http.StatusOK {
		_ = u.response.Body.Close()
		close(u.requestSent)
		return fmt.Errorf("unexpected http status: %d", resp.StatusCode)
	}

	// TODO: check content type, compression
	u.response = resp

	close(u.requestSent)

	return nil
}

func (u *unaryRequestStream) RecvMsg(message interface{}) error {
	select {
	case <-u.ctx.Done():
		return u.ctx.Err()
	case <-u.requestSent:
	}

	if u.request == nil {
		return errors.New("no http response found")
	}

	if err := recvMsg(u.response.Body, u.clientConn.codec, u.clientConn.compressor, message); err != nil {
		_ = u.response.Body.Close()
		return err
	}

	return nil
}

type streamingRequestStream struct {
	ctx            context.Context
	clientConn     *clientConn
	request        *http.Request
	writeReady     chan struct{}
	writer         io.WriteCloser
	readReady      chan struct{}
	reader         io.ReadCloser
	wroteCloseOnce sync.Once
	err            error // only written by newStreamingRequestStream
	errReady       chan struct{}
}

func newStreamingRequestStream(ctx context.Context, clientConn *clientConn, request *http.Request) *streamingRequestStream {
	reader, writer := io.Pipe()
	request.Body = reader

	s := &streamingRequestStream{
		ctx:        ctx,
		clientConn: clientConn,
		request:    request,
		writeReady: make(chan struct{}),
		readReady:  make(chan struct{}),
		errReady:   make(chan struct{}),
		writer:     writer,
	}

	go func() {
		client := &http.Client{
			Transport: s,
		}

		if _, err := client.Do(s.request); err != nil {
			s.err = err
			close(s.errReady)
			_ = s.CloseSend()
		}
	}()

	return s
}

func (s *streamingRequestStream) RoundTrip(req *http.Request) (*http.Response, error) {
	close(s.writeReady)

	resp, err := s.clientConn.transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("unexpected http status: %d", resp.StatusCode)
	}

	if resp.Header.Get("Content-Type") != baseContentType+"+"+s.clientConn.codec.Name() {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("unexpected content type %q", resp.Header.Get("Content-Type"))
	}

	// todo, check grpc-encoding

	s.reader = resp.Body
	close(s.readReady)

	return resp, err
}

func (s *streamingRequestStream) Context() context.Context {
	return s.ctx
}

func (s *streamingRequestStream) SendMsg(message interface{}) error {
	select {
	case <-s.ctx.Done():
		return s.ctx.Err()
	case <-s.errReady:
		return fmt.Errorf("failed to send message because of prior error: %w", s.err)
	case <-s.writeReady:
	}

	if err := sendMsg(s.writer, s.clientConn.codec, s.clientConn.compressor, message); err != nil {
		_ = s.CloseSend()
		//_ = s.reader.Close()
		return err
	}

	return nil
}

func (s *streamingRequestStream) CloseSend() error {
	s.wroteCloseOnce.Do(func() {
		_ = s.writer.Close()
	})
	return nil
}

func (s *streamingRequestStream) RecvMsg(message interface{}) error {
	select {
	case <-s.ctx.Done():
		return s.ctx.Err()
	case <-s.errReady:
		return fmt.Errorf("failed to receive message because of prior error: %w", s.err)
	case <-s.readReady:
	}

	if err := recvMsg(s.reader, s.clientConn.codec, s.clientConn.compressor, message); err != nil {
		_ = s.reader.Close()
		return err
	}

	return nil
}
