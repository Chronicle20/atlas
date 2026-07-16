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
| `ByScriptNameProvider(name, page)` | Returns a provider for one page of scripts with a given name (across all types) |
| `ByScriptNameAndTypeProvider(name, type)` | Returns a provider for a single script by name and type |
| `AllProvider(page)` | Returns a provider for one page of scripts belonging to the current tenant |
| `DeleteAllForTenant()` | Hard-deletes all scripts for the current tenant; returns count |
| `Count()` | Returns the number of map scripts for the current tenant and the max `updated_at` timestamp. Returns `(0, nil, nil)` when the tenant has no rows |
| `Process(field, characterId, scriptName, scriptType)` | Loads the script by name and type, evaluates rules in order, executes operations for the first matching rule. Returns `ProcessResult` |

**ConditionEvaluator** (`script/evaluator.go`)

Evaluates a single condition for a character. `map_id` conditions are evaluated locally using the field model. All other condition types (`gender`, `job`, `level`, `quest_status`) are delegated to atlas-query-aggregator via the validation processor.

**OperationExecutor** (`script/executor.go`)

Executes operations by creating saga commands. Supported operation types:

| Operation Type | Saga Step Action | Required Params |
|----------------|-----------------|-----------------|
| `field_effect` | `FieldEffect` | `path` |
| `lock_ui` | `UiLock` | (none) |
| `unlock_ui` | `UiLock` | (none) |
| `show_intro` | `ShowIntro` | `path` |
| `spawn_monster` | `SpawnMonster` | `monsterId`; optional: `x`, `y`, `count`, `mapId` |
| `drop_message` | `SendMessage` | `message`; optional: `messageType` |

### Seed Adapters

**OnFirstUserEnterSubdomain** / **OnUserEnterSubdomain** (`script/subdomain_on_first_user_enter.go`, `script/subdomain_on_user_enter.go`)

Implement the shared `seeder.Subdomain[jsonMapScript, MapScript]` interface for the `onFirstUserEnter` and `onUserEnter` script types respectively.

| Method | Description |
|--------|-------------|
| `Name()` | Returns the script type (`"onFirstUserEnter"` or `"onUserEnter"`) |
| `Path()` | Returns the catalog subdirectory (`"map-actions/onFirstUserEnter"` or `"map-actions/onUserEnter"`) |
| `Type()` | Returns `"map-action"` |
| `EntityIDPattern()` | Matches catalog filenames of the form `map-<name>.json`, capturing the script name |
| `DeleteAllForTenant(db)` | Hard-deletes all scripts of this subdomain's script type for the tenant, via `DeleteAllByType` |
| `Decode(payload)` | Decodes a catalog entry's attributes into a `jsonMapScript` |
| `Build(t, entityID, attrs)` | Constructs a single-element `[]MapScript` from the decoded attributes, using the catalog entity ID as the script name |
| `BulkCreate(db, models)` | Inserts the built `MapScript` models via `BulkCreate` |
| `Count(db)` | Returns the count of scripts of this subdomain's script type for the tenant |

**Package-level persistence functions** (`script/administrator.go`)

| Function | Description |
|----------|-------------|
| `DeleteAllByType(db, scriptType)` | Hard-deletes all map scripts of a given `script_type` in the tenant context |
| `BulkCreate(db, tenantId, models)` | Inserts a slice of `MapScript` models as new entities |

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

### Core Models

**ConditionInput** (`validation/model.go`)

| Field | Type | Description |
|-------|------|-------------|
| `Type` | `string` | Condition type |
| `Operator` | `string` | Comparison operator |
| `Value` | `int` | Value to compare against |
| `ReferenceId` | `uint32` | Reference identifier (optional) |
| `Step` | `string` | Step identifier (optional) |
| `WorldId` | `world.Id` | World identifier (optional) |
| `ChannelId` | `channel.Id` | Channel identifier (optional) |
| `IncludeEquipped` | `bool` | Whether to include equipped items (optional) |

**ValidationResult** (`validation/model.go`)

| Field | Type | Description |
|-------|------|-------------|
| `characterId` | `uint32` | Character that was validated |
| `passed` | `bool` | Whether the validation passed |

### Processors

**Processor** (`validation/processor.go`)

| Method | Description |
|--------|-------------|
| `ValidateCharacterState(characterId, conditions)` | Sends a POST request to the query aggregator's validation endpoint and returns the result |
