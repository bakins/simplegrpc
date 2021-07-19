#!/bin/bash
set -eu

ROOT="$(git rev-parse --show-toplevel)"

protoc \
    --proto_path=./examples/helloworld \
    --go_out=./examples \
    --go-simple-grpc_out=./examples \
     --plugin=protoc-gen-go-simple-grpc=./script/gen.sh \
     --go-grpc_out=./examples \
    ./examples/helloworld/helloworld.proto
