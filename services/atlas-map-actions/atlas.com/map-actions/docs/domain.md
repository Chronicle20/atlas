# Domain — atlas-map-actions

## Script

### Responsibility

Represents a map entry script definition containing rules that determine behavior when a character enters a map.

### Core Models

**MapScript** (`script/model.go`)

| Field | Type | Description |
|-------|------|-------------|
| `scriptName` | `string` | Script identifier (e.g., `"goSwordman"`, `"108010301"`) |
| `scriptType` | `string` | `"onFirstUserEnter"` or `"onUserEnter"` |
| `description` | `string` | Human-readable description |
| `rules` | `[]Rule` | Ordered list of rules |

**Rule** (`script/model.go`)

| Field | Type | Description |
|-------|------|-------------|
| `id` | `string` | Rule identifier within the script |
| `conditions` | `[]condition.Model` | Conditions that must all be true (AND logic) |
| `operations` | `[]operation.Model` | Operations to execute when rule matches |

**ProcessResult** (`script/model.go`)

| Field | Type | Description |
|-------|------|-------------|
| `MatchedRule` | `string` | ID of the matched rule, `"no_script"`, or `"no_match"` |
| `Operations` | `[]operation.Model` | Operations from the matched rule |
| `Error` | `error` | Error encountered during processing |

### Invariants

- All model fields are private with getter accessors. Construction is via builders only.
- `MapScriptBuilder` produces `MapScript`; `RuleBuilder` produces `Rule`.
- `scriptName` and `scriptType` are required for REST input extraction. `Extract` returns an error if either is empty.

### Processors

**ScriptProcessor** (`script/processor.go`)

| Method | Description |
|--------|-------------|
| `Create(model)` | Persists a new map script |
| `Update(id, model)` | Updates an existing map script by ID |
| `Delete(id)` | Soft-deletes a map script by ID |
| `ByIdProvider(id)` | Returns a provider for a single script by UUID |
| `ByScriptNameProvider(name)` | Returns a provider for all scripts with a given name |
| `ByScriptNameAndTypeProvider(name, type)` | Returns a provider for a single script by name and type |
| `AllProvider()` | Returns a provider for all scripts belonging to the current tenant |
| `DeleteAllForTenant()` | Hard-deletes all scripts for the current tenant; returns count |
| `Seed()` | Deletes all scripts for the tenant, loads JSON files from the filesystem, and creates them in the database. Returns `SeedResult` |
| `Process(field, characterId, scriptName, scriptType)` | Loads the script by name and type, evaluates rules in order, executes operations for the first matching rule. Returns `ProcessResult` |

**ConditionEvaluator** (`script/evaluator.go`)

Evaluates a single condition for a character. `map_id` conditions are evaluated locally using the field model. All other condition types (`gender`, `job`, `level`, `quest_status`) are delegated to atlas-query-aggregator via the validation processor.

**OperationExecutor** (`script/executor.go`)

Executes operations by creating saga commands. Supported operation types:

| Operation Type | Saga Step Action | Required Params |
|----------------|-----------------|-----------------|
| `field_effect` | `FieldEffect` | `path` |
| `show_intro` | `ShowInfo` | `path` |
| `spawn_monster` | `SpawnMonster` | `monsterId`; optional: `x`, `y`, `count`, `mapId` |
| `drop_message` | `SendMessage` | `message`; optional: `messageType` |
| `unlock_ui` | (no-op) | (none) |

## Saga

### Responsibility

Sends saga commands to the saga orchestrator for operation execution.

### Processors

**Processor** (`saga/processor.go`)

| Method | Description |
|--------|-------------|
| `Create(saga)` | Emits a saga command message to the saga command topic |

## Validation

### Responsibility

Delegates character condition evaluation to atlas-query-aggregator via REST.

### Processors

**Processor** (`validation/processor.go`)

| Method | Description |
|--------|-------------|
| `ValidateCharacterState(characterId, conditions)` | Sends a POST request to the query aggregator's validation endpoint and returns the result |
