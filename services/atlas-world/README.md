# atlas-world

## Overview

A RESTful service that maintains an in-memory registry of active game worlds, their channel servers, and per-world rate multipliers. The service aggregates channel status events to provide a consolidated view of available game servers per tenant, and supports runtime rate adjustments that are persisted in Redis and propagated via Kafka.

## External Dependencies

- Redis: Channel server registry and rate multiplier storage (via `atlas-redis` TenantRegistry)
- Kafka: Consumes channel status events; produces channel status commands and world rate change events
- Configuration Service: Retrieves tenant and world configuration via REST

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| BOOTSTRAP_SERVERS | Kafka broker host:port |
| REST_PORT | HTTP server port |
| SERVICE_ID | Service identifier UUID |
| BASE_SERVICE_URL | Base URL for configuration service |
| REDIS_URL | Redis host:port |
| REDIS_PASSWORD | Redis password |
| TRACE_ENDPOINT | OpenTelemetry OTLP gRPC endpoint for tracing |
| COMMAND_TOPIC_CHANNEL_STATUS | Kafka topic for channel status commands |
| EVENT_TOPIC_CHANNEL_STATUS | Kafka topic for channel status events |
| EVENT_TOPIC_WORLD_RATE | Kafka topic for world rate change events |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
