package simplegrpc

import (
	"context"
	"fmt"
	"strconv"
)

// ServerOption is a functional option for extending a gRPC service.
type ServerOption func(*ServerOptions)

// ServerOptions encapsulate the configurable parameters on a gRPC server. This type is meant to be used only by generated code.
type ServerOptions struct {
	Interceptors []Interceptor
	Codecs       []Codec
}

// WithServerInterceptors defines the interceptors for a gRPC server.
func WithServerInterceptors(interceptors ...Interceptor) ServerOption {
	return func(opts *ServerOptions) {
		opts.Interceptors = append(opts.Interceptors, interceptors...)
	}
}

// WithCodecs defines the codecs for a gRPC server.
// Usually, a codec for protobuf is automatically for you.
func WithCodecs(codecs ...Codec) ServerOption {
	return func(opts *ServerOptions) {
		opts.Codecs = append(opts.Codecs, codecs...)
	}
}

// Interceptor is a form of middleware for gRPC requests.
type Interceptor func(Method) Method

// ChainInterceptors chains multiple Interceptors into a single Interceptor.
// The first interceptor wraps the second one, and so on.
// Returns nil if interceptors is empty. Nil interceptors are ignored.
func ChainInterceptors(interceptors ...Interceptor) Interceptor {
	filtered := make([]Interceptor, 0, len(interceptors))
	for _, interceptor := range interceptors {
		if interceptor != nil {
			filtered = append(filtered, interceptor)
		}
	}
	switch n := len(filtered); n {
	case 0:
		return nil
	case 1:
		return filtered[0]
	default:
		first := filtered[0]
		return func(next Method) Method {
			for i := len(filtered) - 1; i > 0; i-- {
				next = filtered[i](next)
			}
			return first(next)
		}
	}
}

// Method is a generic representation of a gRPC-generated RPC method.
// It is used to define Interceptors.
type Method func(ctx context.Context, request interface{}) (interface{}, error)

// Codec defines the interface gRPC uses to encode and decode messages.
type Codec interface {
	// Marshal returns the wire format of v.
	Marshal(ctx context.Context, v interface{}) ([]byte, error)
	// Unmarshal parses the wire format into v.
	Unmarshal(ctx context.Context, data []byte, v interface{}) error
	// Name returns the name of the Codec implementation. The returned string
	// will be used as part of content type in transmission.  The result must be
	// static; the result cannot change between calls.
	Name() string
}

// ToCodec wraps functions to implement Codec.
// This is meant to be used only by generated code.
func ToCodec(name string, marshal func(ctx context.Context, v interface{}) ([]byte, error), unmarshal func(ctx context.Context, data []byte, v interface{}) error) Codec {
	w := codecWraper{
		name:      name,
		marshal:   marshal,
		unmarshal: unmarshal,
	}

	return &w
}

type codecWraper struct {
	marshal   func(ctx context.Context, v interface{}) ([]byte, error)
	unmarshal func(ctx context.Context, data []byte, v interface{}) error
	name      string
}

func (w *codecWraper) Name() string {
	return w.name
}

func (w *codecWraper) Marshal(ctx context.Context, v interface{}) ([]byte, error) {
	return w.marshal(ctx, v)
}

func (w *codecWraper) Unmarshal(ctx context.Context, data []byte, v interface{}) error {
	return w.unmarshal(ctx, data, v)
}

type Code uint32

type Error interface {
	error
	Code() Code
	Message() string
}

func NewError(code Code, message string) Error {
	e := grpcError{
		code:    code,
		message: message,
	}

	return &e
}

func Errorf(code Code, format string, args ...interface{}) Error {
	return NewError(code, fmt.Sprintf(format, args...))
}

type grpcError struct {
	message string
	code    Code
}

func (e *grpcError) Error() string {
	return fmt.Sprintf("code: %s message %s", e.code.String(), e.message)
}

func (e *grpcError) Code() Code {
	return e.code
}

func (e *grpcError) Message() string {
	return e.message
}

// see https://pkg.go.dev/google.golang.org/grpc/codes#Code
const (
	OK                 Code = 0
	Canceled           Code = 1
	Unknown            Code = 2
	InvalidArgument    Code = 3
	DeadlineExceeded   Code = 4
	NotFound           Code = 5
	AlreadyExists      Code = 6
	PermissionDenied   Code = 7
	ResourceExhausted  Code = 8
	FailedPrecondition Code = 9
	Aborted            Code = 10
	OutOfRange         Code = 11
	Unimplemented      Code = 12
	Internal           Code = 13
	Unavailable        Code = 14
	DataLoss           Code = 15
	Unauthenticated    Code = 16
)

func (c Code) String() string {
	switch c {
	case OK:
		return "OK"
	case Canceled:
		return "Canceled"
	case Unknown:
		return "Unknown"
	case InvalidArgument:
		return "InvalidArgument"
	case DeadlineExceeded:
		return "DeadlineExceeded"
	case NotFound:
		return "NotFound"
	case AlreadyExists:
		return "AlreadyExists"
	case PermissionDenied:
		return "PermissionDenied"
	case ResourceExhausted:
		return "ResourceExhausted"
	case FailedPrecondition:
		return "FailedPrecondition"
	case Aborted:
		return "Aborted"
	case OutOfRange:
		return "OutOfRange"
	case Unimplemented:
		return "Unimplemented"
	case Internal:
		return "Internal"
	case Unavailable:
		return "Unavailable"
	case DataLoss:
		return "DataLoss"
	case Unauthenticated:
		return "Unauthenticated"
	default:
		return "Code(" + strconv.FormatInt(int64(c), 10) + ")"
	}
}

type contextKey int

const (
	methodNameKey contextKey = 1 + iota
	serviceNameKey
	packageNameKey
	requestHeaderKey
)

func WithMethodName(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, methodNameKey, name)
}

func WithServiceName(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, serviceNameKey, name)
}

func WithPackageName(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, packageNameKey, name)
}

// MethodName extracts the name of the method being handled in the given
// context. If it is not known, it returns ("", false).
func MethodName(ctx context.Context) (string, bool) {
	name, ok := ctx.Value(methodNameKey).(string)
	return name, ok
}

// ServiceName extracts the name of the service handling the given context. If
// it is not known, it returns ("", false).
func ServiceName(ctx context.Context) (string, bool) {
	name, ok := ctx.Value(serviceNameKey).(string)
	return name, ok
}

// PackageName extracts the fully-qualified protobuf package name of the service
// handling the given context. If it is not known, it returns ("", false). If
// the service comes from a proto file that does not declare a package name, it
// returns ("", true).
//
// Note that the protobuf package name can be very different than the go package
// name; the two are unrelated.
func PackageName(ctx context.Context) (string, bool) {
	name, ok := ctx.Value(packageNameKey).(string)
	return name, ok
}
