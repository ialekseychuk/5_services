# 5-Service gRPC Communication System

This project implements a distributed system with 5 services that communicate via bidirectional gRPC streams in a Docker network. Each service discovers its neighbors, establishes connections, and exchanges messages periodically.

## Architecture

- **5 Services**: Deployed as separate Docker containers
- **gRPC Communication**: Bidirectional streaming between services
- **Service Discovery**: Automatic neighbor detection using Docker's internal DNS
- **Message Exchange**: Services send random strings to neighbors every second
- **Logging**: Comprehensive logging of all communication events

## Project Structure

```
.
├── cmd/
│   └── service/
│       └── main.go          # Service entry point
├── internal/
│   ├── config/
│   │   └── config.go        # Configuration management
│   ├── logger/
│   │   └── logger.go        # Logger setup
│   └── service/
│       └── service.go       # Core service implementation
├── pkg/
│   └── random/
│       └── random_string.go # Random string generation
├── proto/
│   ├── service.proto        # gRPC service definition
│   └── gen/                 # Generated gRPC code
├── Dockerfile               # Service Docker image definition
├── docker-compose.yml       # Multi-container deployment
└── go.mod                   # Go module dependencies
```

## Prerequisites

- Docker
- Docker Compose
- Go 1.21 (for local development)

## Getting Started

### Deploy with Docker Compose

1. Build and start all services:
   ```bash
   docker-compose up --build
   ```

2. View service logs:
   ```bash
   docker-compose logs -f
   ```

3. Stop all services:
   ```bash
   docker-compose down
   ```

### Environment Variables

Each service can be configured with the following environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVICE_ID` | Unique identifier for the service | `service1` |
| `SERVICE_PORT` | Port on which the service listens | `5001` |
| `POLLING_PERIOD_SECONDS` | Neighbor discovery interval in seconds | `10` |
| `MESSAGE_PERIOD_SECONDS` | Message sending interval in seconds | `1` |

## How It Works

1. **Service Discovery**: Each service periodically polls the network to discover neighbors by attempting to resolve service names (service1, service2, etc.) using Docker's built-in DNS.

2. **Connection Establishment**: When a neighbor is discovered, a bidirectional gRPC stream is established between the services.

3. **Message Exchange**: Services send random strings to all connected neighbors every second and log all sent and received messages.

4. **Neighbor Management**: Services automatically handle neighbor connections and disconnections, maintaining an up-to-date list of active neighbors.

## gRPC Service Definition

The communication between services is defined in `proto/service.proto`:

```protobuf
syntax = "proto3";

package proto;

option go_package = "./proto";

// Service definition
service NeighborService {
  // Bidirectional streaming for communication between neighbors
  rpc Communicate(stream Message) returns (stream Message);
}

// Message structure
message Message {
  string sender = 1;
  string content = 2;
  int64 timestamp = 3;
}
```

## Development

### Local Development Setup

1. Install dependencies:
   ```bash
   go mod tidy
   ```

2. Generate gRPC code:
   ```bash
   protoc --go_out=./proto/gen --go-grpc_out=./proto/gen proto/service.proto
   ```

3. Run a single service instance:
   ```bash
   SERVICE_ID=service1 SERVICE_PORT=5001 go run cmd/service/main.go
   ```

### Project Dependencies

- `google.golang.org/grpc` - gRPC implementation for Go
- `github.com/sirupsen/logrus` - Structured logger for Go
- `google.golang.org/protobuf` - Protocol buffer support

## Monitoring and Logging

Services log various events including:
- Service startup and shutdown
- Neighbor discovery events
- Connection establishment and termination
- Message sending and receiving
- Error conditions

Logs can be viewed using:
```bash
docker-compose logs -f <service_name>
```

## Troubleshooting

### Common Issues

1. **Services not discovering neighbors**: Ensure all services are running in the same Docker network.
2. **Connection failures**: Check that service names are correctly configured in docker-compose.yml.
3. **Port conflicts**: Make sure the host ports mapped in docker-compose.yml are available.

### Debugging

To view detailed logs:
```bash
docker-compose logs --tail=100
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.