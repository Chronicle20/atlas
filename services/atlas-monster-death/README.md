# atlas-monster-death
Mushroom game Monster Death Event Handler

## Overview

A Kafka consumer service that handles monster death events. When a monster is killed, this service:
1. Evaluates and creates item/meso drops based on monster drop tables
2. Distributes experience to characters who damaged the monster

This service has no REST endpoints - it operates purely through Kafka message consumption and production.

## Architecture

```
Monster Killed Event (Kafka)
         │
         ▼
┌─────────────────────┐
│  atlas-monster-death │
│                     │
│  • Evaluate drops   │
│  • Calculate exp    │
└─────────────────────┘
         │
         ├──► Spawn Drop Commands (Kafka)
         │
         └──► Award Experience Commands (Kafka)
```

## Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| JAEGER_HOST | Jaeger tracing endpoint | `jaeger:6831` |
| LOG_LEVEL | Logging level | `Panic` / `Fatal` / `Error` / `Warn` / `Info` / `Debug` / `Trace` |
| BOOTSTRAP_SERVERS | Kafka bootstrap servers | `kafka:9092` |

## Kafka Topics

### Consumed Topics
- Monster killed events

### Produced Topics
- Spawn drop commands
- Award experience commands

## Multi-Tenancy

This service supports multi-tenancy through Kafka headers:

```
TENANT_ID: 083839c6-c47c-42a6-9585-76492795d123
REGION: GMS
MAJOR_VERSION: 83
MINOR_VERSION: 1
```

These headers are propagated from incoming Kafka messages to outgoing REST calls and Kafka commands.

## External Service Dependencies

This service makes REST calls to the following services:
- **Character Service** - Get character information (level)
- **Map Service** - Get characters currently in map
- **Monster Data Service** - Get monster drop tables and information

## Testing

Run tests with:
```bash
go test ./...
```
