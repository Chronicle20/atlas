# atlas-rps

Owns rock/paper/scissors NPC minigame sessions: opening a session at an NPC, running rounds (throw submission, adjudication, ladder progression), and ending a session (collect, quit, retry, or disconnect) with the associated prize payout.

## Overview

This service maintains a Redis-backed TTL registry of active RPS sessions, one per (tenant, character). A session is opened via REST (the entry point atlas-saga-orchestrator calls) and then driven to completion via Kafka commands (`BEGIN`, `SELECT`, `CONTINUE`, `RETRY`, `COLLECT`, `QUIT`). Each round's opponent throw is drawn server-side and adjudicated server-side. Session progress and termination are published as Kafka events (`GAME_OPENED`, `ROUND_STARTED`, `ROUND_RESULT`, `GAME_ENDED`) for downstream consumers. Prize and fee payouts are submitted as sagas to atlas-saga-orchestrator. A periodic sweep task reclaims sessions abandoned past their TTL.

## External Dependencies

- Redis: Session registry (TTL-backed) and per-tenant tenant-tracking set
- Kafka: Consumes RPS commands; produces RPS events and saga commands
- atlas-tenants: REST API for retrieving the tenant's rps-rewards configuration (entry cost, consolation meso, reward ladder)
- OpenTelemetry: Distributed tracing via OTLP/gRPC (sweep task span)

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| BOOTSTRAP_SERVERS | Kafka host:port |
| REST_PORT | HTTP server port |
| TENANTS | atlas-tenants REST API base URL |
| COMMAND_TOPIC_RPS | Kafka topic for RPS commands (consumed) |
| EVENT_TOPIC_RPS | Kafka topic for RPS events (produced) |
| COMMAND_TOPIC_SAGA | Kafka topic for atlas-saga-orchestrator commands (produced) |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
