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

## Condition Types

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
