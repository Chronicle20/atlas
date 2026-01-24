# atlas-invites

Invitation management service for tracking and coordinating invitations between characters.

## Overview

This service manages invitations sent between characters for various social features. It provides an in-memory registry for invite tracking, processes invite lifecycle commands via Kafka, and exposes a REST API for querying pending invites. Invites automatically expire after a configurable timeout period.

## External Dependencies

- Kafka - Message broker for invite commands and status events
- Jaeger - Distributed tracing

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| JAEGER_HOST_PORT | Jaeger host:port for tracing |
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| REST_PORT | Port for the REST server |
| BOOTSTRAP_SERVERS | Kafka bootstrap servers |
| COMMAND_TOPIC_INVITE | Kafka topic for invite commands |
| EVENT_TOPIC_INVITE_STATUS | Kafka topic for invite status events |

## Documentation

- [Domain](docs/domain.md) - Domain models and processors
- [Kafka](docs/kafka.md) - Kafka integration
- [REST](docs/rest.md) - REST API endpoints
