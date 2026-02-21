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
- Cooldown state is maintained in Redis via the CooldownRegistry.

### Processors

#### Processor

| Operation | Description |
|-----------|-------------|
| ByCharacterIdProvider | Returns all skills for a character |
| ByIdProvider | Returns a specific skill by character and skill ID |
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

- Macro name is required.

### Processors

#### Processor

| Operation | Description |
|-----------|-------------|
| ByCharacterIdProvider | Returns all macros for a character |
| Update | Replaces all macros for a character |
| UpdateAndEmit | Replaces all macros and emits a status event |
| Delete | Deletes all macros for a character |
