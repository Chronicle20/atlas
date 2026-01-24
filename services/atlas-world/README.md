# atlas-world

## Overview

A RESTful service that maintains an in-memory registry of active game worlds and their channel servers. The service aggregates channel status events to provide a consolidated view of available game servers per tenant.

## External Dependencies

- Kafka: Consumes channel status events and produces channel status commands
- Configuration Service: Retrieves tenant and world configuration via REST

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| JAEGER_HOST | Jaeger host:port for tracing |
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| BOOTSTRAP_SERVERS | Kafka broker host:port |
| REST_PORT | HTTP server port |
| SERVICE_ID | Service identifier UUID |
| BASE_SERVICE_URL | Base URL for configuration service |
| COMMAND_TOPIC_CHANNEL_STATUS | Kafka topic for channel status commands |
| EVENT_TOPIC_CHANNEL_STATUS | Kafka topic for channel status events |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
