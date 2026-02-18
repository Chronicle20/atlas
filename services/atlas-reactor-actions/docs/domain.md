# Script Domain

## Responsibility

Manages reactor script definitions and evaluates them when reactors are hit or triggered. Loads scripts from storage, matches rules based on reactor state, and delegates operation execution to saga orchestration.

## Core Models

### ReactorScript

Represents a reactor script loaded from storage.

| Field | Type | Description |
|-------|------|-------------|
| reactorId | string | Reactor classification identifier |
| description | string | Human-readable description |
| hitRules | []Rule | Rules evaluated when reactor is hit |
| actRules | []Rule | Rules evaluated when reactor triggers |

### Rule

Represents a single rule with conditions and operations.

| Field | Type | Description |
|-------|------|-------------|
| id | string | Rule identifier |
| conditions | []condition.Model | Conditions that must all be true |
| operations | []operation.Model | Operations to execute when matched |

### ProcessResult

Represents the result of processing a reactor script.

| Field | Type | Description |
|-------|------|-------------|
| MatchedRule | string | ID of matched rule or "no_script"/"no_match" |
| Operations | []operation.Model | Operations to execute |
| Error | error | Error if evaluation failed |

### ReactorContext

Context information for reactor operation execution.

| Field | Type | Description |
|-------|------|-------------|
| Field | field.Model | Field context (world, channel, map, instance) |
| ReactorId | uint32 | Reactor instance identifier |
| Classification | string | Reactor classification |
| ReactorName | string | Reactor name |
| X | int16 | X coordinate |
| Y | int16 | Y coordinate |

### SeedResult

Result of a seed operation.

| Field | Type | Description |
|-------|------|-------------|
| DeletedCount | int | Scripts deleted |
| CreatedCount | int | Scripts created |
| FailedCount | int | Scripts that failed to load |
| Errors | []string | Error messages |

## Invariants

- Rules are evaluated in order; first matching rule wins
- Empty conditions list means the rule always matches
- All conditions within a rule must be true (AND logic)
- If no script exists for a reactor, no action is taken
- Operations are executed sequentially

## Processors

### ScriptProcessor

Interface for reactor script processing.

**CRUD Operations:**
- `Create(model ReactorScript) (ReactorScript, error)`
- `Update(id uuid.UUID, model ReactorScript) (ReactorScript, error)`
- `Delete(id uuid.UUID) error`

**Query Operations:**
- `ByIdProvider(id uuid.UUID) model.Provider[ReactorScript]`
- `ByReactorIdProvider(reactorId string) model.Provider[ReactorScript]`
- `AllProvider() model.Provider[[]ReactorScript]`

**Seeding:**
- `DeleteAllForTenant() (int64, error)`
- `Seed() (SeedResult, error)`

**Execution:**
- `ProcessHit(reactorId string, reactorState int8, characterId uint32) ProcessResult`
- `ProcessTrigger(reactorId string, reactorState int8, characterId uint32) ProcessResult`

### ConditionEvaluator

Evaluates conditions for reactor scripts.

- `EvaluateCondition(reactorState int8, characterId uint32, cond condition.Model) (bool, error)`
- `EvaluateRule(reactorState int8, characterId uint32, rule Rule) (bool, error)`

**Supported Condition Types:**
- `reactor_state`: Compares reactor state value
- `pq_custom_data`: Compares party quest custom data value; uses the condition's `step` field as the custom data key name; queries atlas-party-quests service for the character's PQ instance; missing key or non-numeric value is treated as 0

**Supported Operators:**
- `=`, `!=`, `>`, `<`, `>=`, `<=`

### OperationExecutor

Executes reactor script operations via saga orchestration.

- `ExecuteOperation(rc ReactorContext, characterId uint32, op operation.Model) error`
- `ExecuteOperations(rc ReactorContext, characterId uint32, ops []operation.Model) error`

**Supported Operation Types:**
- `drop_items`: Spawns reactor drops via saga
- `spawn_monster`: Spawns monsters at reactor location via saga
- `spray_items`: Sprays items with delay (delegates to drop_items with spray type)
- `weaken_area_boss`: Weakens a boss monster (not yet implemented)
- `move_environment`: Moves map environment object (not yet implemented)
- `kill_all_monsters`: Kills all monsters in map (not yet implemented)
- `drop_message`: Sends message to character via saga
- `update_pq_state`: Updates party quest custom data via saga; params: `updates` (comma-separated key=value pairs), `increments` (comma-separated key names to increment); queries atlas-party-quests for the character's PQ instance
- `hit_reactor`: Hits another reactor by name via saga; params: `reactorName`
- `broadcast_pq_message`: Broadcasts a message to all PQ members via saga; params: `message`, `type` (defaults to PINK_TEXT); queries atlas-party-quests for the character's PQ instance
- `stage_clear_attempt`: Triggers a stage clear attempt on the character's party quest instance via saga; queries atlas-party-quests for the character's PQ instance

## Builders

### ReactorScriptBuilder

Builds ReactorScript instances.

- `NewReactorScriptBuilder() *ReactorScriptBuilder`
- `SetReactorId(reactorId string) *ReactorScriptBuilder`
- `SetDescription(description string) *ReactorScriptBuilder`
- `AddHitRule(rule Rule) *ReactorScriptBuilder`
- `AddActRule(rule Rule) *ReactorScriptBuilder`
- `Build() ReactorScript`

### RuleBuilder

Builds Rule instances.

- `NewRuleBuilder() *RuleBuilder`
- `SetId(id string) *RuleBuilder`
- `AddCondition(cond condition.Model) *RuleBuilder`
- `AddOperation(op operation.Model) *RuleBuilder`
- `Build() Rule`
