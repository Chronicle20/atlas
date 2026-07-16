# Domain

## Skill

### Responsibility

Represents a character's acquired skill with its current level, master level, expiration time, and cooldown state.

### Core Models

#### Model

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Skill identifier |
| level | byte | Current skill level |
| masterLevel | byte | Maximum achievable level |
| expiration | time.Time | Time when the skill expires |
| cooldownExpiresAt | time.Time | Time when the cooldown ends |

### Invariants

- Skill id is required and must be non-zero.
- Cooldown state is maintained in Redis via the skill package's cooldown Registry.
- A skill row is keyed by the composite (tenant, character, id): a skill id is shared across every character, so the row does not exist independently of its owning character.
- TransferSp rejects the transfer (emits an ERROR status event, does not mutate state) when: the source or target skill does not belong to the caller's job tree; the source or target skill is point-reset-excluded; the target skill's job tier does not match the supplied item tier, or the source skill's job tier is below 1 or above the item tier; the source skill's level is 0; or the target skill's level is at or above its cap (the supplied targetMaxLevel, or the target's own master level when the target job is a 4th job).
- TransferSp moves exactly one point from the source skill to the target skill; master levels are not modified.
- When TransferSp drops the source skill to level 0, any macro referencing that skill has the reference cleared (set to skill id 0) in the same transaction.

### Processors

#### Processor

| Operation | Description |
|-----------|-------------|
| ByCharacterIdProvider | Returns one page of skills for a character, decorated with cooldown state |
| ByIdProvider | Returns a specific skill by character and skill ID, decorated with cooldown state |
| Create | Creates a new skill for a character |
| CreateAndEmit | Creates a skill and emits a status event |
| Update | Updates an existing skill |
| UpdateAndEmit | Updates a skill and emits a status event |
| SetCooldown | Applies a cooldown to a skill |
| SetCooldownAndEmit | Applies a cooldown and emits a status event |
| ClearAll | Clears all cooldowns for a character |
| Delete | Deletes all skills for a character |
| CooldownDecorator | Decorates a skill model with cooldown information from the registry |
| RequestCreate | Sends a command to create a skill |
| RequestUpdate | Sends a command to update a skill |
| DeleteForSagaCompensation | Deletes a single skill for a character idempotently and buffers a DELETED status event; used for saga compensation |
| DeleteForSagaCompensationAndEmit | Runs DeleteForSagaCompensation and emits the buffered event |
| TransferSp | Moves one skill point from a source skill to a target skill, re-validating job tree, exclusion list, tier, and level/cap state, and clearing macro references to the source skill when it reaches level 0 |
| TransferSpAndEmit | Runs TransferSp and emits the buffered event(s) |
| WithTransaction | Returns a Processor bound to the given database transaction |

A background task (ExpirationTask) periodically scans the cooldown registry and clears any cooldown past its expiration, emitting a COOLDOWN_EXPIRED status event for each.

---

## Macro

### Responsibility

Represents a skill macro configuration that binds up to three skills to a single activation.

### Core Models

#### Model

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Macro identifier |
| name | string | Macro display name |
| shout | bool | Whether to announce macro activation |
| skillId1 | skill.Id | First skill in the macro |
| skillId2 | skill.Id | Second skill in the macro |
| skillId3 | skill.Id | Third skill in the macro |

### Invariants

- Macro id, name, and skill references are not independently validated by the model builder; an empty name is permitted.

### Processors

#### Processor

| Operation | Description |
|-----------|-------------|
| ByCharacterIdProvider | Returns all macros for a character |
| ByCharacterIdPagedProvider | Returns one page of macros for a character |
| Update | Replaces all macros for a character |
| UpdateAndEmit | Replaces all macros and emits a status event |
| Delete | Deletes all macros for a character |
| WithTransaction | Returns a Processor bound to the given database transaction |
