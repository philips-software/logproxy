FROM golang:1.16.6-alpine3.13 as build_base
RUN apk add --no-cache git openssh gcc musl-dev
WORKDIR /logproxy
COPY go.mod .
COPY go.sum .

# Get dependancies - will also be cached if we won't change mod/sum
RUN go mod download
LABEL builder=true

# Build
FROM build_base AS builder
WORKDIR /logproxy
COPY . .
RUN ./buildscript.sh

FROM golang:1.16.6-alpine3.13
LABEL maintainer="Andy Lo-A-Foe <andy.lo-a-foe@philips.com>"
RUN apk --no-cache add ca-certificates
ENV HOME /root
WORKDIR /app
COPY --from=builder /logproxy/build/logproxy /app/logproxy
EXPOSE 8080
CMD ["/app/logproxy"]
