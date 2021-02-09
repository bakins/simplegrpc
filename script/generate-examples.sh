#!/bin/bash
set -eu

ROOT="$(git rev-parse --show-toplevel)"

protoc \
    --proto_path=./examples/helloworld/helloworld \
    --go_out=./examples/helloworld/helloworld \
    --go_opt=paths=source_relative \
    --go-simple-grpc_out=./examples/helloworld/helloworld \
    --go-simple-grpc_opt=paths=source_relative \
    --plugin=protoc-gen-go-simple-grpc=./script/gen.sh \
    --go-grpc_out=./examples/helloworld/helloworld \
    --go-grpc_opt=paths=source_relative \
    ./examples/helloworld/helloworld/helloworld.proto

protoc \
    --proto_path=./examples/routeguide/routeguide \
    --go_out=./examples/routeguide/routeguide  \
    --go_opt=paths=source_relative \
    --go-simple-grpc_out=./examples/routeguide/routeguide  \
    --go-simple-grpc_opt=paths=source_relative \
    --plugin=protoc-gen-go-simple-grpc=./script/gen.sh \
    ./examples/routeguide/routeguide/routeguide.proto
