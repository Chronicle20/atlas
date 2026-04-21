# Context: Party Quest Admin Commands

Last Updated: 2026-02-16

## Key Files

### atlas-party-quests (`services/atlas-party-quests/atlas.com/party-quests/`)

| File | Purpose |
|------|---------|
| `kafka/message/party_quest/kafka.go` | All Kafka command/event types. Add `FORCE_STAGE_COMPLETE`. |
| `kafka/consumer/party_quest/consumer.go` | Consumer handlers. Add handler for new command. |
| `instance/processor.go` | Core business logic. Add `ForceStageComplete` method. |
| `instance/model.go` | Instance model with states, character entries. Reference only. |
| `instance/registry.go` | In-memory registry with `GetByCharacter(characterId)`. Reference only. |
| `instance/producer.go` | Kafka event message providers. Reference only (reuse existing providers). |

### atlas-messages (`services/atlas-messages/atlas.com/messages/`)

| File | Purpose |
|------|---------|
| `kafka/message/party_quest/kafka.go` | **CREATE** — Kafka message types for PQ commands. |
| `command/party_quest/commands.go` | **CREATE** — `@pq register` and `@pq stage` command producers. |
| `main.go` | Register new command producers. |
| `command/help/commands.go` | Update help text with new commands. |

### Reference Files (patterns to follow)

| File | Pattern |
|------|---------|
| `atlas-messages: command/monster/commands.go` | Direct Kafka command pattern (regex, GM check, producer call, pink text) |
| `atlas-messages: kafka/message/monster/kafka.go` | Kafka message type + provider pattern |
| `atlas-messages: command/help/commands.go` | Help text registration pattern |

## Key Decisions

1. **FORCE_STAGE_COMPLETE takes characterId, not instanceId** — Atlas-messages doesn't know the instance ID; the party-quest service resolves it via `GetByCharacter`.

2. **Single atomic operation** — Force-clear + advance in one handler, avoiding async ordering issues with two separate Kafka messages.

3. **GM-only commands** — Both commands check `c.Gm()` before executing.

4. **No REST client needed** — Both commands are fire-and-forget Kafka messages. Pink text confirms the command was sent; the party-quest service logs any processing errors.

## Key Types

### Party Quest Command Envelope (reused across services)
```go
type Command[E any] struct {
    WorldId     world.Id `json:"worldId"`
    CharacterId uint32   `json:"characterId"`
    Type        string   `json:"type"`
    Body        E        `json:"body"`
}
```

### atlas-messages Command Type Signatures
```go
type Producer func(l logrus.FieldLogger) func(ctx context.Context) func(ch channel.Model, character character.Model, m string) (Executor, bool)
type Executor func(l logrus.FieldLogger) func(ctx context.Context) error
```

### Producer call pattern
```go
producer.ProviderImpl(l)(ctx)(topicEnvVar)(messageProvider)
```
