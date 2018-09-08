# build stage
FROM golang:1.11.0-alpine3.8 AS builder
RUN apk add --no-cache git openssh gcc musl-dev
WORKDIR /logproxy
COPY . /logproxy
RUN cd /logproxy && go build -o logproxy

FROM alpine:latest 
MAINTAINER Andy Lo-A-Foe <andy.loafoe@aemian.com>
WORKDIR /app
COPY --from=builder /logproxy/logproxy /app
RUN apk --no-cache add ca-certificates

EXPOSE 8080
CMD ["/app/logproxy"]
