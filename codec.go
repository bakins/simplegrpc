package simplegrpc

import "github.com/golang/protobuf/proto"

type Codec interface {
	Name() string
	Marshal(v interface{}) ([]byte, error)
	Unmarshal(data []byte, v interface{}) error
}

var ProtoCodec Codec = protoCodec{}

type protoCodec struct{}

func (p protoCodec) Name() string {
	return "proto"
}

func (p protoCodec) Marshal(v interface{}) ([]byte, error) {
	return proto.Marshal(v.(proto.Message))
}

func (p protoCodec) Unmarshal(data []byte, v interface{}) error {
	return proto.Unmarshal(data, v.(proto.Message))
}
