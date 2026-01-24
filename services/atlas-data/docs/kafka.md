# Kafka Integration

## Topics Consumed

| Topic | Environment Variable | Description |
|-------|---------------------|-------------|
| Data Command | COMMAND_TOPIC_DATA | Receives commands to start data workers |

## Topics Produced

None.

## Message Types

### Commands Consumed

#### START_WORKER

Triggers a data worker to process XML files at a specified path.

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

## Transaction Semantics

- Consumer uses `kafka.LastOffset` as start offset
- Messages are processed with persistent configuration
- Tenant context is extracted from message headers
