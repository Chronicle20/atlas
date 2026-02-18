# Portal Script Domain

## Responsibility

The portal script domain manages rules-based portal entry behavior. It determines whether characters can use portals and executes operations when rules match.

## Core Models

### PortalScript

Represents a portal script with ordered rules for portal entry.

| Field | Type | Description |
|-------|------|-------------|
| portalId | string | Portal identifier |
| mapId | uint32 | Map where portal exists |
| description | string | Human-readable description |
| rules | []Rule | Ordered list of rules |

### Rule

Represents a single rule with conditions and outcome.

| Field | Type | Description |
|-------|------|-------------|
| id | string | Rule identifier |
| conditions | []condition.Model | Conditions that must all pass (AND logic) |
| onMatch | RuleOutcome | Outcome when rule matches |

### RuleOutcome

Represents the outcome when a rule matches.

| Field | Type | Description |
|-------|------|-------------|
| allow | bool | Whether portal entry is allowed |
| operations | []operation.Model | Operations to execute |

### ProcessResult

Represents the result of processing a portal script.

| Field | Type | Description |
|-------|------|-------------|
| Allow | bool | Whether portal entry was allowed |
| MatchedRule | string | ID of the matched rule |
| Operations | []operation.Model | Operations that were executed |
| Error | error | Any error that occurred |

### SeedResult

Represents the result of a seed operation.

| Field | Type | Description |
|-------|------|-------------|
| DeletedCount | int | Number of scripts deleted |
| CreatedCount | int | Number of scripts created |
| FailedCount | int | Number of scripts that failed to load or create |
| Errors | []string | Error messages for failed scripts |

## Invariants

- Rules are evaluated in order; first matching rule wins
- Empty conditions list means the rule always matches (default rule)
- All conditions in a rule must pass for the rule to match (AND logic)
- If no rules match, entry is denied by default
- If no script exists for a portal, entry is allowed by default

## Processors

### ScriptProcessor

Handles CRUD operations, seeding, and script execution.

| Method | Description |
|--------|-------------|
| Create | Creates a new portal script |
| Update | Updates an existing portal script |
| Delete | Deletes a portal script |
| ByIdProvider | Retrieves a portal script by UUID |
| ByPortalIdProvider | Retrieves a portal script by portal ID |
| AllProvider | Retrieves all portal scripts for tenant |
| DeleteAllForTenant | Deletes all scripts for current tenant |
| Seed | Clears existing scripts and loads from filesystem |
| Process | Processes a portal entry request |

### ConditionEvaluator

Evaluates conditions against character state via the validation service.

| Method | Description |
|--------|-------------|
| EvaluateCondition | Evaluates a single condition for a character |

### OperationExecutor

Executes operations via saga messages.

| Method | Description |
|--------|-------------|
| ExecuteOperation | Executes a single operation |
| ExecuteOperations | Executes multiple operations |

### Loader

Loads portal scripts from JSON files on the filesystem. Maintains an in-memory cache keyed by portal ID with thread-safe access.

| Method | Description |
|--------|-------------|
| LoadByPortalId | Loads a portal script by ID (cache-through) |
| ClearCache | Clears the in-memory script cache |
| Preload | Loads all scripts from directory into cache |

## Condition Types

Condition types are defined by the `atlas-script-core/condition` library. The evaluator forwards conditions to the validation service for evaluation.

| Type | Description |
|------|-------------|
| level | Character level |
| job | Character job ID |
| quest_state | Quest completion state |
| item | Item possession check |
| buff | Active buff check |

## Operation Types

| Type | Description | Parameters |
|------|-------------|------------|
| warp | Warp character to map | mapId, portalId, portalName |
| play_portal_sound | Play portal sound effect | none |
| drop_message | Display message to player | message, messageType |
| show_hint | Show hint popup | hint, width, height |
| block_portal | Block portal for character | mapId, portalId |
| create_skill | Create skill for character | skillId, level, masterLevel, expiration |
| update_skill | Update skill for character | skillId, level, masterLevel, expiration |
| start_instance_transport | Start instance-based transport | routeName, failureMessage |
| apply_consumable_effect | Apply consumable effect (buff) | itemId |
| cancel_consumable_effect | Cancel consumable effect (buff) | itemId |
| save_location | Save character location | locationType, mapId, portalId |
| warp_to_saved_location | Warp to previously saved location | locationType |

---

# Validation Domain

## Responsibility

The validation domain evaluates character state conditions by delegating to an external validation service via HTTP.

## Core Models

### ConditionInput

Represents a condition to validate against character state.

| Field | Type | Description |
|-------|------|-------------|
| Type | string | Condition type |
| Operator | string | Comparison operator |
| Value | int | Comparison value |
| ReferenceId | uint32 | Reference identifier |
| Step | string | Quest step |
| IncludeEquipped | bool | Include equipped items |

### ValidationResult

Represents the result of a validation.

| Field | Type | Description |
|-------|------|-------------|
| characterId | uint32 | Character that was validated |
| passed | bool | Whether the validation passed |

## Processors

### Processor

Validates character state via HTTP POST to the query aggregator service.

| Method | Description |
|--------|-------------|
| ValidateCharacterState | Validates character state against conditions |

---

# Action Domain

## Responsibility

The action domain tracks pending portal actions that require saga completion. Used for operations like `start_instance_transport` where the result of the saga determines whether a failure message needs to be sent to the character.

## Core Models

### PendingAction

Represents a pending portal action awaiting saga completion.

| Field | Type | Description |
|-------|------|-------------|
| CharacterId | uint32 | Character identifier |
| WorldId | world.Id | World identifier |
| ChannelId | channel.Id | Channel identifier |
| FailureMessage | string | Message to display on saga failure |

## Invariants

- Registry is a singleton via `sync.Once`
- Thread-safe via `sync.RWMutex`
- Keyed by tenant ID and saga transaction ID
- Entries are removed on saga completion or failure

## Processors

### Registry

Thread-safe in-memory registry for pending actions.

| Method | Description |
|--------|-------------|
| Add | Registers a pending action for a saga |
| Get | Retrieves a pending action by saga ID |
| Remove | Removes a pending action by saga ID |

---

# Saga Domain

## Responsibility

The saga domain produces saga command messages to the saga orchestrator and handles saga status events (completion and failure).

## Processors

### Processor

Creates saga commands via Kafka.

| Method | Description |
|--------|-------------|
| Create | Produces a saga command message |
