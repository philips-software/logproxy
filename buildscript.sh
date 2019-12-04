#!/bin/sh

set -x

if [ ! ${GOPATH} ]; then
  echo GOPATH is not set
  exit 1
fi

GIT_COMMIT=$(git rev-parse --short HEAD)

CURRENT_DIR="$(pwd)"
cd "${GOPATH}"
wget -O - -q https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s v1.19.1

cd src
go mod download
go get -u github.com/jstemmer/go-junit-report
go get -u github.com/t-yuki/gocover-cobertura

rm -rf build/ && mkdir -p build
cd "${CURRENT_DIR}"
go build -ldflags "-X main.commit=${GIT_COMMIT}" -o build/logproxy .
go test -coverprofile build/coverage.out -covermode count -v ./... 2>&1  > build/test-result.txt
go-junit-report < build/test-result.txt > build/TEST-report.xml
gocover-cobertura < build/coverage.out > build/coverage-cobertura.xml
golangci-lint run
