# build stage
FROM golang:1.12.5-alpine3.9 AS builder
RUN apk add --no-cache git openssh gcc musl-dev

WORKDIR /logproxy
COPY go.mod .
COPY go.sum .
# Get dependancies - will also be cached if we won't change mod/sum
RUN go mod download

# Build
COPY . .
RUN go build .

FROM alpine:latest 
MAINTAINER Andy Lo-A-Foe <andy.lo-a-foe@philips.com>
WORKDIR /app
COPY --from=builder /logproxy/logproxy /app
COPY --from=builder /logproxy/local.yml /app
RUN apk --no-cache add ca-certificates

EXPOSE 8080
CMD ["/app/logproxy"]
