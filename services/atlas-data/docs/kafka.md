# Kafka Integration

## Topics Consumed

| Topic | Environment Variable | Consumer Name | Description |
|-------|---------------------|---------------|-------------|
| Data Command | COMMAND_TOPIC_DATA | data_command | Receives commands to start a legacy, in-process data worker. Registered at startup; see note below on current reachability. |

## Topics Produced

| Topic | Environment Variable | Description |
|-------|---------------------|-------------|
| Data Command | COMMAND_TOPIC_DATA | Producer code exists (`data.ProcessorImpl.InstructWorker`) to emit `START_WORKER` commands, one per legacy worker type. |
| Data Event | EVENT_TOPIC_DATA | Produces `DATA_UPDATED` after a legacy worker (triggered by a consumed `START_WORKER` command) completes. |

**Current reachability note:** the primary ingest path (`POST /api/data/process`, see `docs/rest.md`) creates a Kubernetes `MODE=ingest` Job that reads WZ archives from MinIO directly — it does not go through Kafka. The `START_WORKER` producer (`data.ProcessorImpl.InstructWorker`, called only from `data.ProcessorImpl.ProcessData`) has no caller anywhere in this codebase, so nothing currently publishes to `COMMAND_TOPIC_DATA`. The consumer and the `DATA_UPDATED` producer remain registered and would activate if a `START_WORKER` command were published by some other means.

## Message Types

### Commands Consumed

#### START_WORKER

Instructs the legacy in-process worker to parse a local, `ZIP_DIR`-rooted XML tree for one data type.

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

Same shape as above. Emitted by `data.ProcessorImpl.InstructWorker`, one message per legacy worker type, only when `data.ProcessorImpl.ProcessData` runs (currently unreachable — see note above).

### Events Produced

#### DATA_UPDATED

Emitted after a legacy worker (`data.ProcessorImpl.StartWorker`) finishes processing one data type, unless `DATA_EVENTS_PRODUCER_ENABLED` parses as `false`. Emission failures are logged and counted (metric), not retried.

```go
type event[E any] struct {
    Type string `json:"type"`
    Body E      `json:"body"`
}

type dataUpdatedEventBody struct {
    TenantId    string `json:"tenantId"`
    Worker      string `json:"worker"`
    CompletedAt string `json:"completedAt"` // RFC 3339
}
```

Message key is the tenant id.

## Transaction Semantics

- Consumer group ID: `Data Service`
- Consumer uses `kafka.LastOffset` as start offset
- Messages are processed with persistent configuration
- Tenant context is extracted from message headers via `TenantHeaderParser`
- Span context is extracted from message headers via `SpanHeaderParser`
- Producer decorates messages with `SpanHeaderDecorator` and `TenantHeaderDecorator`
- Broker address is configured via `BOOTSTRAP_SERVERS` environment variable
