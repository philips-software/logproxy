#! /bin/sh

cd /src
go mod download
go get -u github.com/jstemmer/go-junit-report
go get -u github.com/t-yuki/gocover-cobertura
rm -rf build/ && mkdir -p build
go build -v -o build/logproxy .
go test  -coverprofile build/coverage.out -covermode count -v ./... 2>&1  | tee build/test-result.txt 
go-junit-report < build/test-result.txt | tee build/TEST-report.xml
gocover-cobertura < build/coverage.out > build/coverage-cobertura.xml
