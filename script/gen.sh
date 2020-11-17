#!/bin/bash
set -eu
ROOT=$(git rev-parse --show-toplevel)

cd "$ROOT"

exec go run ./cmd/protoc-gen-go-simple-grpc "$@"
