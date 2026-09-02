# atlas-dragons

Owns Evan's dragon (CDragon) lifecycle: creating a dragon for a dragon-bearing character on login/map-change/channel-change, destroying it on logout/field-departure/job-change out of range, and relaying its movement.

## Overview

This service maintains a Redis-backed registry of active dragons, one per owner character id (a dragon has no identity of its own; the owner character id is the primary key). The service consumes dragon commands (`CREATE`, `DESTROY`, `MOVE`) from `atlas-channel` and character-status lifecycle events (`LOGIN`, `LOGOUT`, `MAP_CHANGED`, `CHANNEL_CHANGED`, `JOB_CHANGED`) from `atlas-character` and `atlas-maps`, and produces dragon status events (`CREATED`, `MOVED`, `DESTROYED`) for downstream consumers. Whether a character has a dragon is determined by resolving the character's job id, on the tenant's client version, to an Evan growth stage.

## External Dependencies

- Redis: All state storage (dragon registry, field index)
- Kafka: Consumes dragon commands and character status events; produces dragon status events
- atlas-character (`CHARACTERS`): REST API for retrieving a character's job id and position
- OpenTelemetry: Distributed tracing via OTLP/gRPC (via `atlas-service` bootstrap)

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| BOOTSTRAP_SERVERS | Kafka host:port |
| REST_PORT | HTTP server port |
| CHARACTERS | atlas-character REST API base URL |
| COMMAND_TOPIC_DRAGON | Kafka topic for dragon commands (consumed) |
| EVENT_TOPIC_CHARACTER_STATUS | Kafka topic for character status events (consumed) |
| EVENT_TOPIC_DRAGON_STATUS | Kafka topic for dragon status events (produced) |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
