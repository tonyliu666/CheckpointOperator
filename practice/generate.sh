#!/bin/bash

if [ "$1" == "api" ]; then
    protoc --proto_path=agent/api/ --go_out=agent/api --go-grpc_out=agent/api/ api.proto
fi