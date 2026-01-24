# Expression Domain

## Responsibility

Manages character expressions with automatic expiration. Expressions are stored in memory and revert to a default state after a fixed duration.

## Core Models

### Model

Represents an active expression for a character.

| Field | Type | Description |
|-------|------|-------------|
| tenant | tenant.Model | Tenant context |
| characterId | uint32 | Character identifier |
| worldId | world.Id | World identifier |
| channelId | channel.Id | Channel identifier |
| mapId | map.Id | Map identifier |
| expression | uint32 | Expression identifier |
| expiration | time.Time | When the expression expires |

### Registry

Singleton in-memory store for active expressions, keyed by tenant and character.

## Invariants

- Expressions expire 5 seconds after creation
- One active expression per character per tenant
- Setting a new expression replaces any existing expression for that character

## Processors

### Processor

| Operation | Description |
|-----------|-------------|
| Change | Adds or replaces an expression in the registry and buffers an event |
| ChangeAndEmit | Changes an expression and immediately emits the event |
| Clear | Removes an expression from the registry |
| ClearAndEmit | Clears an expression and immediately emits an event |

### RevertTask

Background task that periodically checks for expired expressions and emits events to revert them to expression 0.
