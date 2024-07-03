FROM golang:1.23rc1-alpine as builder
RUN apk --no-cache add git
WORKDIR /build
COPY go.mod .
COPY go.sum .
RUN go mod download -x

# Build
COPY . .
RUN git rev-parse --short HEAD
RUN GIT_COMMIT=$(git rev-parse --short HEAD) && \
    CGO_ENABLED=0 go build -o app -ldflags "-X main.GitCommit=${GIT_COMMIT}"

FROM alpine:3.20.1
RUN apk --no-cache add ca-certificates
ENV HOME /root
WORKDIR /app
COPY --from=builder /build/app /app/logproxy
EXPOSE 8080
CMD ["/app/logproxy"]
