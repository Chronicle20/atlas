# Expression Domain

## Responsibility

Manages character facial expressions with automatic expiration. Expressions are stored in Redis with a TTL and revert to the default state (expression 0) after a fixed duration.

## Core Models

### Model

Represents an active expression for a character.

| Field | Type | Description |
|-------|------|-------------|
| tenant | tenant.Model | Tenant context |
| characterId | uint32 | Character identifier |
| field | field.Model | Location (worldId, channelId, mapId, instance) |
| expression | uint32 | Expression identifier |
| expiration | time.Time | When the expression expires |

Convenience getters expose `WorldId()`, `ChannelId()`, `MapId()`, and `Instance()` from the embedded field.

### ModelBuilder

Fluent builder for constructing Model instances. Requires tenant, characterId, and expiration. Supports `CloneModelBuilder` for deriving new models from existing ones.

## Invariants

- Expressions expire 5 seconds after creation
- One active expression per character per tenant
- Setting a new expression replaces any existing expression for that character
- Clearing an expression removes it from the registry without emitting a revert

## Processors

### Processor

| Operation | Description |
|-----------|-------------|
| Change | Adds or replaces an expression in the registry and buffers an event |
| ChangeAndEmit | Changes an expression and immediately emits the event |
| Clear | Removes an expression from the registry |
| ClearAndEmit | Clears an expression and immediately emits an event |

### RevertTask

Background task that runs every 50ms, checks for expired expressions across all tracked tenants, and emits events to revert them to expression 0.
