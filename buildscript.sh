#! /bin/sh

cd /src
go mod download
go get -u github.com/jstemmer/go-junit-report
rm -rf build/ && mkdir -p build
go build -v -o build/logproxy .
go test  -coverprofile build/coverage.out -v ./... 2>&1  | tee build/test-result.txt 
go-junit-report < build/test-result.txt | tee build/TEST-report.xml
