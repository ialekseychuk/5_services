# Build stage
FROM golang:1.21-alpine AS builder

# Install protobuf compiler
RUN apk add --no-cache protobuf

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy source code
COPY . .

# Generate gRPC code
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.31.0 && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0 && \
    mkdir -p proto/gen && \
    protoc --go_out=./proto/gen --go-grpc_out=./proto/gen proto/service.proto

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o service cmd/service/main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/service .

# Expose port
EXPOSE 5001

# Run the service
CMD ["./service"]