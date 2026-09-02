# atlas-kites

Owns the Kite (cash category 508 message box) registry and lifecycle: placement, per-character and per-map placement policy enforcement, and teardown on owner logout, map change, and channel change.

## Overview

This service maintains a Redis-backed registry of active kites across all tenants, worlds, channels, and maps, keyed by owning character. Kite placement is driven by commands from atlas-channel and validated against a per-tenant policy (blocked map prefixes, maximum message length, maximum kites per map). It also maintains a Redis-backed index of which characters are currently present in each field, built from consumed character status events, used to answer "which kites are in this field" queries. The service consumes kite commands and character status events, and produces kite status events (`CREATED`, `DESTROYED`, `CREATION_FAILED`) for downstream consumers.

## External Dependencies

- Redis: All state storage (kite registry, wire-id allocation, per-field placement lock, character-in-field index)
- Kafka: Consumes kite commands and character status events; produces kite status events
- atlas-tenants: REST API for retrieving a tenant's kite placement-policy configuration (`kite-configs`)
- OpenTelemetry: Distributed tracing via OTLP/gRPC

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| BOOTSTRAP_SERVERS | Kafka host:port |
| REST_PORT | HTTP server port |
| TENANTS | atlas-tenants REST API base URL |
| COMMAND_TOPIC_KITE | Kafka topic for kite commands (consumed) |
| EVENT_TOPIC_CHARACTER_STATUS | Kafka topic for character status events (consumed) |
| EVENT_TOPIC_KITE_STATUS | Kafka topic for kite status events (produced) |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
