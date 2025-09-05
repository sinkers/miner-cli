#!/bin/bash

# Generate Go code from protobuf files
PROTO_DIR="."
OUT_DIR=".."
export PATH=$PATH:$(go env GOPATH)/bin

# Generate all proto files
for proto_file in $(find $PROTO_DIR -name "*.proto"); do
    echo "Generating Go code for: $proto_file"
    protoc \
        --go_out=$OUT_DIR \
        --go_opt=paths=source_relative \
        --go-grpc_out=$OUT_DIR \
        --go-grpc_opt=paths=source_relative \
        -I $PROTO_DIR \
        $proto_file
done

echo "Protobuf generation complete!"