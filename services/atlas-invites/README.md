# atlas-invites

Invitation management service for tracking and coordinating invitations between characters.

## Overview

This service manages invitations sent between characters for various social features. It provides a Redis-backed registry for invite tracking, processes invite lifecycle commands via Kafka, and exposes a REST API for querying pending invites. Invites automatically expire after a configurable timeout period. When a character is deleted, all associated invites are removed and rejection events are emitted.

## External Dependencies

- Redis - Invite registry storage and indexing
- Kafka - Message broker for invite commands and status events
- OpenTelemetry (OTLP/gRPC) - Distributed tracing

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| TRACE_ENDPOINT | OTLP gRPC endpoint for tracing |
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| REST_PORT | Port for the REST server |
| BOOTSTRAP_SERVERS | Kafka bootstrap servers |
| COMMAND_TOPIC_INVITE | Kafka topic for invite commands |
| EVENT_TOPIC_INVITE_STATUS | Kafka topic for invite status events |
| EVENT_TOPIC_CHARACTER_STATUS | Kafka topic for character status events |

## Documentation

- [Domain](docs/domain.md) - Domain models and processors
- [Kafka](docs/kafka.md) - Kafka integration
- [REST](docs/rest.md) - REST API endpoints
- [Storage](docs/storage.md) - Storage structures
