# Kite Domain

## Responsibility

Manages Kite instances (cash category 508 message boxes): validating and applying placement requests against a per-tenant policy, enforcing the one-kite-per-character and per-map-cap invariants, and removing a character's kite on owner logout, map departure, or channel change.

## Core Models

### Model

Represents a single placed kite. Immutable; constructed via `Builder` (`NewBuilder`).

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Kite wire id (registry-allocated; distinct from characterId) |
| f | field.Model | Field the kite is placed in (world, channel, map, instance) |
| characterId | uint32 | Owning character id |
| name | string | Owner's character name, captured at placement |
| templateId | uint32 | Kite template/item id |
| message | string | Kite message text |
| x | int16 | X coordinate |
| y | int16 | Y coordinate |
| createdAt | time.Time | Placement time |

## Invariants

- A character owns at most one kite at a time. `Create` refuses with `ErrAlreadyPlaced` if the character already has a kite placed.
- A field holds at most `configuration.Model.MaxPerMap()` kites (default 10). `Create` refuses with `ErrMapFull` once a field's count meets the cap, and also refuses with `ErrMapFull` when the per-field placement lock cannot be acquired (lock contention is treated as full).
- `Create` refuses with `ErrMapForbidden` when the field's map id falls in one of the tenant's `BlockedMapPrefixes` (default: prefix `91`, the Free Market map range).
- `Create` refuses with `ErrMessageTooLong` when the command's message exceeds `configuration.Model.MaxMessageLength()` (default 182 characters).
- The map-policy, message-length, and one-per-character checks are character-local and run before the per-field lock; only the per-map cap check runs under the lock.
- Every refusal buffers a `CREATION_FAILED` status event before returning its typed error; no refusal ever emits `CREATED`.
- `CreateAndEmit` rolls back the registry insert (removes the character's kite) if the buffered events fail to flush after a successful insert.
- `DestroyAndEmit` treats the registry removal as authoritative: once the kite is removed from the registry, a failure to emit `DESTROYED` is logged but does not fail the call. A genuine domain error (no such kite, a registry failure) is always returned.
- `Destroy` reads the kite's field before removing it from the registry, so the `DESTROYED` event fans out to the map the kite was actually placed in.
- The wire id (`Model.Id`) is allocated per-tenant from a Redis counter (`Registry.NextId`), independent of `characterId`.

## Processors

### Processor

Interface defining kite processing operations. Constructed via `NewProcessor(l, ctx)`, which wires the project's standard Kafka `producer.Provider` and the real `configuration.Registry`.

**Queries:**
- `GetByCharacterId`: Retrieves the kite owned by a character, or `ErrNotFound` if the character has none placed.
- `InMapModelProvider` / `GetInMap`: Retrieves every kite currently placed in a field, by intersecting the character-presence index for that field (from the `character` domain) with kite ownership.

**Commands:**
- `Create`: Buffers a `CREATE` result into a `message.Buffer`; validates policy, allocates the wire id, inserts the kite, and buffers `CREATED`. Every refusal buffers `CREATION_FAILED` instead and returns a typed error (`ErrMapForbidden`, `ErrMessageTooLong`, `ErrAlreadyPlaced`, `ErrMapFull`).
- `CreateAndEmit`: Wraps `Create` in `message.Emit`, flushing buffered events to Kafka; rolls back the insert on a post-insert emit failure.
- `Destroy`: Buffers removal of a character's kite and a `DESTROYED` event with the given reason. Returns `ErrNotFound` if the character has no kite.
- `DestroyAndEmit`: Wraps `Destroy` in `message.Emit`; logs (rather than fails) an emit failure once the registry removal has already applied.

## Character Presence Domain

### Responsibility

Maintains a per-field index of which characters are currently present, used by the kite domain to resolve "which kites are in this field."

### Core Models

#### MapKey

Composite key identifying a tenant-scoped field.

| Field | Type | Description |
|-------|------|-------------|
| Tenant | tenant.Model | Owning tenant |
| Field | field.Model | Field (world, channel, map, instance) |

### Invariants

- A character is a member of at most the field it last entered; `TransitionMap` and `TransitionChannel` both perform an `Exit` from the old field followed by an `Enter` into the new field.
- A Redis failure reading the field index is propagated to the caller rather than coerced to an empty result, so the kite domain's per-map cap check fails loudly instead of under-counting on a Redis error.

### Processors

#### Processor

Interface defining character-presence operations. Constructed via `NewProcessor(l, ctx)`.

- `InMapProvider` / `GetCharactersInMap`: Returns every characterId currently indexed under a field.
- `Enter`: Adds a character to a field's index. Errors are logged, not returned.
- `Exit`: Removes a character from a field's index. Errors are logged, not returned.
- `TransitionMap`: Exits the old field and enters the new field (map change).
- `TransitionChannel`: Exits the old field and enters the new field (channel change).

## Configuration Domain

### Responsibility

Resolves and caches each tenant's kite placement policy (blocked map prefixes, maximum message length, maximum kites per map), fetched from atlas-tenants, with a compiled default used whenever a tenant has not provisioned it.

### Core Models

#### Model

Immutable per-tenant kite placement policy.

| Field | Type | Description |
|-------|------|-------------|
| maxPerMap | int | Maximum kites allowed in one field (default 10) |
| maxMessageLength | int | Maximum kite message length in characters (default 182) |
| blockedMapPrefixes | []uint32 | Map-id prefixes (mapId / 10,000,000) that refuse placement (default `[91]`, the Free Market range) |

`IsMapBlocked(mapId)` reports whether `mapId / 10,000,000` matches one of `blockedMapPrefixes`.

### Invariants

- `DefaultConfig()` is used whenever a tenant has not provisioned a `kite-configs` resource, and independently for any individual knob left at its zero value in a fetched config (`Extract`).
- A fetch that returns `requests.ErrNotFound` is treated as "tenant not yet provisioned" and logged at Info; any other fetch error is logged at Warn. Either way the registry caches `DefaultConfig()` for that tenant so the fallback is a one-time cost per tenant per process.

### Processors

Configuration has no processor of its own; it is read through `Registry.GetTenantConfig(l, ctx, tenantId)`, a lazy per-tenant cache (`Registry`) backed by a `fetcher` that calls atlas-tenants. The kite `Processor` reads a tenant's policy through this registry on every `Create`.
