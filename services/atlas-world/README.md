# atlas-world

## Overview

A RESTful service that maintains an in-memory registry of active game worlds, their channel servers, and per-world rate multipliers. The service aggregates channel status events to provide a consolidated view of available game servers per tenant, and supports runtime rate adjustments that are persisted in Redis and propagated via Kafka. It also coordinates a per (tenant, world, family) serialized queue of Maple TV and avatar-megaphone broadcasts, sweeping expired active entries on a leader-elected pod.

## External Dependencies

- Redis: Channel server registry, rate multiplier storage, and broadcast queue storage (via `atlas-redis` TenantRegistry), plus the broadcast sweep leader election lock
- Kafka: Consumes channel status events, tenant configuration status events, and world broadcast enqueue commands; produces channel status commands, world rate change events, and world broadcast status events
- Configuration Service: Tenant and world configuration is projected from Kafka tenant configuration status events (config projection)

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| BOOTSTRAP_SERVERS | Kafka broker host:port |
| REST_PORT | HTTP server port |
| REDIS_URL | Redis host:port |
| REDIS_PASSWORD | Redis password |
| TRACE_ENDPOINT | OpenTelemetry OTLP gRPC endpoint for tracing |
| COMMAND_TOPIC_CHANNEL_STATUS | Kafka topic for channel status commands |
| EVENT_TOPIC_CHANNEL_STATUS | Kafka topic for channel status events |
| EVENT_TOPIC_WORLD_RATE | Kafka topic for world rate change events |
| EVENT_TOPIC_CONFIGURATION_TENANT_STATUS | Kafka topic for tenant configuration status events (config projection) |
| COMMAND_TOPIC_WORLD_BROADCAST | Kafka topic for world broadcast enqueue commands |
| EVENT_TOPIC_WORLD_BROADCAST_STATUS | Kafka topic for world broadcast status events |
| PROJECTION_CATCHUP_TIMEOUT_S | Seconds to wait for the configuration projection to catch up at startup (default 300) |
| WORLD_BROADCAST_LEADER_ELECTION_ENABLED | Whether the broadcast sweep runs only on the leader-elected pod (default true; if false, the sweep runs unconditionally on this pod) |
| WORLD_BROADCAST_LEADER_TTL | Broadcast sweep leader election lock TTL (default 30s, range 5s-5m) |
| WORLD_BROADCAST_LEADER_REFRESH | Broadcast sweep leader election lock refresh interval (default TTL/3, minimum 1s; range 1s-TTL/2) |
| WORLD_BROADCAST_LEADER_BACKOFF | Broadcast sweep leader election retry backoff (default 5s, range 1s-1m) |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
