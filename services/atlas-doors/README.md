# atlas-doors

Owns the Mystic Door registry and lifecycle: spawning the area/town door pair, computing the caster's party door slot, reslotting town portals on party membership changes, and removing doors on expiry, recast, logout, channel change, and field departure.

## Overview

This service maintains a Redis-backed registry of active Mystic Door instances across all tenants, worlds, channels, and maps. Each door is a paired object (an area-side door in the source field and a town-side door in the return town) owned by the caster and scoped to a party. The service consumes door commands and character/party status events to drive the door lifecycle, and produces door status events (`CREATED`, `REMOVED`, `SLOT_CHANGED`) for downstream consumers. A leader-elected expiry sweep removes doors whose configured lifetime has elapsed.

## External Dependencies

- Redis: All state storage (door registry, three secondary indices, object-id allocation, leader-election lock)
- Kafka: Consumes door commands and character/party status events; produces door status events
- atlas-data: REST API for retrieving map metadata (town flag, field limit, return/forced-return map, door-type portals) and skill effect duration
- atlas-parties: REST API for retrieving party membership and party-by-member lookups
- OpenTelemetry: Distributed tracing via OTLP/gRPC

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| BOOTSTRAP_SERVERS | Kafka host:port |
| REST_PORT | HTTP server port |
| DATA | atlas-data REST API base URL |
| PARTIES | atlas-parties REST API base URL |
| COMMAND_TOPIC_DOOR | Kafka topic for door commands (consumed) |
| EVENT_TOPIC_CHARACTER_STATUS | Kafka topic for character status events (consumed) |
| EVENT_TOPIC_PARTY_STATUS | Kafka topic for party status events (consumed) |
| EVENT_TOPIC_DOOR_STATUS | Kafka topic for door status events (produced) |
| DOOR_LEADER_ELECTION_ENABLED | Enables leader election for the expiry sweep (default true) |
| DOOR_LEADER_TTL | Leader lock TTL (default 30s; range 5s–5m) |
| DOOR_LEADER_REFRESH | Leader lock refresh interval (default TTL/3; range 1s–TTL/2) |
| DOOR_LEADER_BACKOFF | Leader acquisition backoff (default 5s; range 1s–1m) |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
