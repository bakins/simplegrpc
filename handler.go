package simplegrpc

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/bakins/simplegrpc/codes"
	"github.com/bakins/simplegrpc/status"
)

// The SupportPackageIsVersion variables are referenced from generated protocol buffer files to ensure compatibility with the gRPC version used.
// The latest support package version is 1.
const (
	SupportPackageIsVersion1 = true
)

// StreamServerInterceptor provides a hook to intercept the execution of a streaming RPC on the server.
// info contains all the information of this RPC the interceptor can operate on. And handler is the
// service method implementation. It is the responsibility of the interceptor to invoke handler to
// complete the RPC.
type StreamServerInterceptor func(srv interface{}, ss ServerStream, info *StreamServerInfo, handler StreamHandler) error

// Handler is an HTTP handler for grpc services
type Handler struct {
	methodHandlers map[string]*method
	services       map[string]*service
	codecs         map[string]Codec
	compressors    map[string]Compressor
	interceptor    StreamServerInterceptor
}

type service struct {
	methods     []*method
	serviceDesc ServiceDesc
}

type method struct {
	streamDesc StreamDesc
	server     interface{}
}

// ServiceInfo contains unary RPC method info, streaming RPC method info and metadata for a service.
type ServiceInfo struct {
	Methods []MethodInfo
	// Metadata is the metadata specified in ServiceDesc when registering service.
	Metadata interface{}
}

// MethodInfo contains the information of an RPC including its method name and type.
type MethodInfo struct {
	// Name is the method name only, without the service name or package name.
	Name string
	// IsClientStream indicates whether the RPC is a client streaming RPC.
	IsClientStream bool
	// IsServerStream indicates whether the RPC is a server streaming RPC.
	IsServerStream bool
}

// NewHandler creates a new handler. Only protobuff codec is registered.
func NewHandler() *Handler {
	h := &Handler{}
	h.RegisterCodec(ProtoCodec)
	return h
}

// RegisterCodec registers a codec. This must be called before the handler takes requests
func (h *Handler) RegisterCodec(codec Codec) {
	if h.codecs == nil {
		h.codecs = make(map[string]Codec)
	}

	h.codecs[codec.Name()] = codec
}

// RegisterCompressor registers a Compressor. This must be called before the handler takes requests
func (h *Handler) RegisterCompressor(compressor Compressor) {
	if h.compressors == nil {
		h.compressors = make(map[string]Compressor)
	}

	h.compressors[compressor.Name()] = compressor
}

// GetServiceInfo returns a map from service names to ServiceInfo.
// Service names include the package names, in the form of <package>.<service>.
func (h *Handler) GetServiceInfo() map[string]ServiceInfo {
	out := make(map[string]ServiceInfo)

	for _, s := range h.services {
		info := ServiceInfo{}

		for _, m := range s.methods {
			mi := MethodInfo{
				Name:           m.streamDesc.StreamName,
				IsClientStream: m.streamDesc.ClientStreams,
				IsServerStream: m.streamDesc.ServerStreams,
			}

			info.Methods = append(info.Methods, mi)
		}
		out[s.serviceDesc.ServiceName] = info
	}
	return out
}

// ServeHTP ...
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	codec, err := h.getCodec(r.Header.Get("Content-Type"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Add("Trailer", "grpc-status, grpc-message")

	compressor, err := h.getCompressor(r.Header.Get("Grpc-Encoding"))
	if err != nil {
		for k := range h.compressors {
			w.Header().Add("grpc-accept-encoding", k)
		}

		w.WriteHeader(http.StatusOK)
		statusTrailer(w, err)

		return
	}

	if compressor != nil {
		w.Header().Set("Grpc-Encoding", compressor.Name())
	}

	w.Header().Set("Content-Type", baseContentType+"+"+codec.Name())
	w.WriteHeader(http.StatusOK)

	m, ok := h.methodHandlers[r.URL.Path]
	if !ok {
		err := status.Errorf(codes.Unimplemented, "service method %q is not implemented by this server", r.URL.Path)
		statusTrailer(w, err)

		return
	}

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	r = r.WithContext(ctx)

	stream, err := h.newServerStream(w, r, codec, compressor)

	if h.interceptor == nil {
		err = m.streamDesc.Handler(m.server, stream)
		statusTrailer(w, err)
		return
	}

	// func(srv interface{}, ss ServerStream, info *StreamServerInfo, handler StreamHandler) error
	info := StreamServerInfo{
		FullMethod:     r.URL.Path,
		IsClientStream: m.streamDesc.ClientStreams,
		IsServerStream: m.streamDesc.ServerStreams,
	}

	err = h.interceptor(m.server, stream, &info, m.streamDesc.Handler)
	statusTrailer(w, err)
}

// assumes WriteHeader has allready been called
func statusTrailer(w http.ResponseWriter, err error) {
	if err == nil {
		w.Header().Set("grpc-status", "0")
		w.Header().Set("grpc-message", "OK")
	}

	st, ok := status.FromError(err)

	var code string
	var message string

	if !ok {
		code = strconv.Itoa(int(codes.Unknown))
		message = err.Error()
	} else {
		code = strconv.Itoa(int(st.Code()))

		message = st.Message()
		if message == "" {
			message = st.Code().String()
		}
	}

	w.Header().Set("grpc-status", code)
	w.Header().Set("grpc-message", message)
}

// RegisterService registers a service and its implementation to the gRPC
// server. It is called from the IDL generated code.
// This is not safe for concurrent use and should be called before the handler is used to serve
// http requests
func (h *Handler) RegisterService(sd *ServiceDesc, ss interface{}) {
	if h.methodHandlers == nil {
		h.methodHandlers = make(map[string]*method)
	}

	if h.services == nil {
		h.services = make(map[string]*service)
	}

	ht := reflect.TypeOf(sd.HandlerType).Elem()
	st := reflect.TypeOf(ss)
	if !st.Implements(ht) {
		panic(fmt.Sprintf("grpc: RegisterService found the handler of type %v that does not satisfy %v", st, ht))
	}

	if _, ok := h.services[sd.ServiceName]; ok {
		panic(fmt.Sprintf("grpc: RegisterService found duplicate service registration for %q", sd.ServiceName))
	}

	svc := &service{
		serviceDesc: *sd,
	}

	h.services[sd.ServiceName] = svc

	for _, m := range sd.Streams {
		fullMethod := "/" + sd.ServiceName + "/" + m.StreamName

		if _, ok := h.methodHandlers[fullMethod]; ok {
			panic(fmt.Sprintf("grpc: RegisterService found duplicate method registration for %q", fullMethod))
		}

		mth := &method{
			streamDesc: m,
			server:     ss,
		}

		svc.methods = append(svc.methods, mth)

		h.methodHandlers[fullMethod] = mth
	}
}

func (h *Handler) getCompressor(accept string) (Compressor, error) {
	if accept == "" || accept == "identity" {
		return nil, nil
	}

	c, ok := h.compressors[accept]
	if !ok {
		return nil, status.Errorf(codes.Unimplemented, "unsupported compression %q", accept)
	}

	return c, nil
}

func (h *Handler) getCodec(contentType string) (Codec, error) {
	subType := contentSubtype(contentType)
	if subType == "" {
		return nil, fmt.Errorf("unsupported content-type %q", contentType)
	}

	codec, ok := h.codecs[subType]
	if !ok || codec == nil {
		return nil, fmt.Errorf("unsupported sub-content-type in %q", contentType)
	}

	return codec, nil
}

type serverStream struct {
	ctx        context.Context
	reader     io.ReadCloser
	writer     io.Writer
	codec      Codec
	compressor Compressor
}

func (h *Handler) newServerStream(w http.ResponseWriter, r *http.Request, codec Codec, compressor Compressor) (*serverStream, error) {
	// TODO: compression
	return &serverStream{
		ctx:        r.Context(),
		reader:     r.Body,
		writer:     w,
		codec:      codec,
		compressor: compressor,
	}, nil
}

func (s *serverStream) Context() context.Context {
	return s.ctx
}

const maxReceiveMessageSize = 1024 * 1024 * 1024 * 2

func recvMsg(reader io.Reader, codec Codec, compressor Compressor, message interface{}) error {
	prefix := []byte{0, 0, 0, 0, 0}

	if _, err := reader.Read(prefix); err != nil {
		fmt.Println("read failed", err)
		// EOF here means end of stream
		return err
	}

	length := binary.BigEndian.Uint32(prefix[1:])

	var body []byte

	// TODO: handle maximum message size
	if length > 0 {
		body = make([]byte, length)

		if n, err := reader.Read(body); err != nil {
			// if is EOF and we read it all, it is fine.  next read will return EOF
			if err != io.EOF {
				return err
			}

			if n != int(length) {
				return errors.New("unexpected EOF")
			}
		}
	}

	// todo compress check should be a bit check
	if compressor != nil && prefix[0] == 1 {
		data, err := decompress(compressor, body)
		if err != nil {
			return err
		}

		body = data
	}

	return codec.Unmarshal(body, message)
}

func (s *serverStream) RecvMsg(m interface{}) error {
	return recvMsg(s.reader, s.codec, s.compressor, m)
}

func decompress(compressor Compressor, in []byte) ([]byte, error) {
	if compressor == nil {
		return in, nil
	}

	return compressor.Decompress(in)
}

func sendMsg(writer io.Writer, codec Codec, compressor Compressor, message interface{}) error {
	data, err := codec.Marshal(message)
	if err != nil {
		return err
	}

	data, err = compress(compressor, data)
	if err != nil {
		return err
	}
	prefix := []byte{0, 0, 0, 0, 0}

	// TODO should be a bit flag

	if compressor != nil {
		prefix[0] = 1
	}

	binary.BigEndian.PutUint32(prefix[1:], uint32(len(data)))

	if _, err = writer.Write(prefix); err != nil {
		return err
	}

	_, err = writer.Write(data)
	return err
}

func (s *serverStream) SendMsg(m interface{}) error {
	return sendMsg(s.writer, s.codec, s.compressor, m)
}

func compress(compressor Compressor, in []byte) ([]byte, error) {
	if compressor == nil {
		return in, nil
	}

	return compressor.Compress(in)
}

const baseContentType = "application/grpc"

func contentSubtype(contentType string) string {
	if contentType == baseContentType {
		return "proto"
	}

	if !strings.HasPrefix(contentType, baseContentType) {
		return ""
	}

	// guaranteed since != baseContentType and has baseContentType prefix
	switch contentType[len(baseContentType)] {
	case '+', ';':
		// this will return true for "application/grpc+" or "application/grpc;"
		// which the previous validContentType function tested to be valid, so we
		// just say that no content-subtype is specified in this case
		return contentType[len(baseContentType)+1:]
	default:
		return ""
	}
}

// ServiceDesc represents an RPC service's specification.
type ServiceDesc struct {
	ServiceName string
	HandlerType interface{}
	Streams     []StreamDesc
	Metadata    string
}

// StreamDesc represents a streaming RPC service's method specification.
type StreamDesc struct {
	StreamName    string
	Handler       StreamHandler
	ServerStreams bool
	ClientStreams bool
}

// StreamServerInfo ...
type StreamServerInfo struct {
	FullMethod     string
	IsClientStream bool
	IsServerStream bool
}

// StreamHandler ...
type StreamHandler func(srv interface{}, stream ServerStream) error

// ServerStream ...
type ServerStream interface {
	Context() context.Context
	SendMsg(m interface{}) error
	RecvMsg(m interface{}) error
}

// ServiceRegistrar wraps a single method that supports service registration. It
// enables users to pass concrete types other than grpc.Server to the service
// registration methods exported by the IDL generated code.
type ServiceRegistrar interface {
	// RegisterService registers a service and its implementation to the
	// concrete type implementing this interface.  It may not be called
	// once the server has started serving.
	// desc describes the service and its methods and handlers. impl is the
	// service implementation which is passed to the method handlers.
	RegisterService(desc *ServiceDesc, impl interface{})
}
