# Kafka Integration

## Topics Consumed

| Topic | Environment Variable | Consumer Name | Description |
|-------|---------------------|---------------|-------------|
| Data Command | COMMAND_TOPIC_DATA | data_command | Receives commands to start data workers |

## Topics Produced

| Topic | Environment Variable | Description |
|-------|---------------------|-------------|
| Data Command | COMMAND_TOPIC_DATA | Produces START_WORKER commands to trigger worker processing |

## Message Types

### Commands Consumed

#### START_WORKER

Triggers a data worker to process WZ/XML files at a specified path.

```go
type command[E any] struct {
    Type string `json:"type"`
    Body E      `json:"body"`
}

type startWorkerCommandBody struct {
    Name string `json:"name"`
    Path string `json:"path"`
}
```

Worker names: MAP, MONSTER, CHARACTER, REACTOR, SKILL, PET, CONSUME, CASH, COMMODITY, ETC, SETUP, CHARACTER_CREATION, QUEST, NPC, FACE, HAIR, MOB_SKILL.

### Commands Produced

#### START_WORKER

Produced by `POST /api/data/process` to dispatch processing for each worker type. One message is produced per worker.

## Transaction Semantics

- Consumer group ID: `Data Service`
- Consumer uses `kafka.LastOffset` as start offset
- Messages are processed with persistent configuration
- Tenant context is extracted from message headers via `TenantHeaderParser`
- Span context is extracted from message headers via `SpanHeaderParser`
- Producer decorates messages with `SpanHeaderDecorator` and `TenantHeaderDecorator`
- Broker address is configured via `BOOTSTRAP_SERVERS` environment variable
