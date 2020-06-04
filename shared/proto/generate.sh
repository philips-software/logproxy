#!/bin/bash

set -eu

SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ] ; do SOURCE="$(readlink "$SOURCE")"; done
DIR="$( cd -P "$( dirname "$SOURCE" )" && pwd )"

cd "$DIR"

protoc --proto_path=. --go_out=plugins=grpc:. --go_opt=paths=source_relative resource.proto
