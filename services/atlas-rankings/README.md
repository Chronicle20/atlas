# atlas-rankings

Computes per-world, per-tenant character rankings (overall and per job category) on a configurable cadence and serves them over REST.

## Overview

A leader-elected periodic task re-enumerates tenants from atlas-tenants on a 60-second base tick, and for each tenant whose configured recompute interval has elapsed, reads the full character scan from atlas-character, ranks non-GM characters per world (overall and by job category), and persists the results to a Postgres-backed store. The REST API serves single-character, bulk-character, and per-world leaderboard lookups.

## External Dependencies

- Postgres: Ranking rows (`character_rankings`) and recompute cycle bookkeeping (`ranking_cycles`)
- Redis: Leader-election lock for the recompute task
- atlas-character: REST API for the tenant's full character scan (paginated, drained)
- atlas-tenants: REST API for tenant enumeration and per-tenant rankings configuration (`recomputeIntervalMinutes`)
- OpenTelemetry: Distributed tracing via OTLP/gRPC

This service does not consume or produce Kafka messages.

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| REST_PORT | HTTP server port |
| DB_HOST | Postgres host |
| DB_PORT | Postgres port |
| DB_NAME | Postgres database name |
| DB_USER | Postgres user |
| DB_PASSWORD | Postgres password |
| REDIS_URL | Redis address (leader-election lock) |
| REDIS_PASSWORD | Redis password |
| CHARACTERS_SERVICE_URL | atlas-character REST API base URL (falls back to BASE_SERVICE_URL) |
| TENANTS_SERVICE_URL | atlas-tenants REST API base URL (falls back to BASE_SERVICE_URL) |
| BASE_SERVICE_URL | Default REST API base URL used when a per-domain `*_SERVICE_URL` override is not set |
| RANKINGS_LEADER_ELECTION_ENABLED | Enables leader election for the recompute task (default true) |
| RANKINGS_LEADER_TTL | Leader lock TTL (default 30s; range 5s-5m) |
| RANKINGS_LEADER_REFRESH | Leader lock refresh interval (default TTL/3, minimum 1s; range 1s-TTL/2) |
| RANKINGS_LEADER_BACKOFF | Leader acquisition backoff (default 5s; range 1s-1m) |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
