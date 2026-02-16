# Domain

## definition

### Responsibility

Represents the static structure of a party quest. A definition describes a quest's stages, registration rules, start/fail conditions, rewards, and exit map. Definitions are persisted in the database and loaded from JSON seed files.

### Core Models

**definition.Model** — Immutable model with private fields and getters.

- `Id` — `uuid.UUID`
- `QuestId` — `string`, unique quest identifier
- `Name` — `string`
- `FieldLock` — `string`, one of: `none`, `channel`, `instance`
- `Duration` — `uint64`, global time limit in milliseconds
- `Registration` — embedded `Registration` value object
- `StartRequirements` — `[]condition.Model`
- `StartEvents` — `[]EventTrigger`
- `FailRequirements` — `[]condition.Model`
- `Exit` — `uint32`, map ID for warping characters on completion/failure
- `Stages` — `[]stage.Model`
- `Rewards` — `[]reward.Model`

**definition.Registration** — Value object describing registration behavior.

- `Type` — `string`, one of: `party`, `individual`
- `Mode` — `string`, one of: `instant`, `timed`
- `Duration` — `int64`, registration window duration in milliseconds (for `timed` mode)
- `MapId` — `uint32`, required map for individual registration
- `Affinity` — `string`, one of: `none`, `guild`, `party`

**definition.EventTrigger** — Value object for start events.

- `Type` — `string`
- `Target` — `string`
- `Value` — `string`

### Invariants

- `QuestId` and `Name` are required
- Builder validates required fields on `Build()`

### Processors

**definition.Processor** — Interface + `ProcessorImpl`. Created via `NewProcessor(l, ctx, db)`.

- `Create(model)` — Persists a new definition
- `Update(id, model)` — Updates an existing definition by ID
- `Delete(id)` — Soft-deletes a definition by ID
- `ByIdProvider(id)` — Returns a provider for a single definition by UUID
- `ByQuestIdProvider(questId)` — Returns a provider for a single definition by quest ID
- `AllProvider()` — Returns a provider for all definitions in the tenant
- `DeleteAllForTenant()` — Hard-deletes all definitions for the current tenant
- `Seed()` — Clears all definitions for the tenant, then loads and creates definitions from JSON files on disk

---

## condition

### Responsibility

Represents a conditional check used in start requirements, fail requirements, and stage clear conditions.

### Core Models

**condition.Model** — Immutable model with private fields and getters.

- `Type` — `string`, one of: `item_count`, `monster_kill`, `party_size`, `level_min`, `level_max`
- `Operator` — `string`, one of: `eq`, `gte`, `lte`, `gt`, `lt`
- `Value` — `uint32`
- `ReferenceId` — `uint32`, contextual ID (item template ID, monster ID, etc.)

### Invariants

- `Type` and `Operator` are required

---

## stage

### Responsibility

Represents a single stage within a party quest definition.

### Core Models

**stage.Model** — Immutable model with private fields and getters.

- `Index` — `uint32`, sequential position within the definition
- `Name` — `string`
- `MapIds` — `[]uint32`, maps associated with this stage
- `Type` — `string`, one of: `item_collection`, `monster_killing`, `combination_puzzle`, `reactor_trigger`, `warp_puzzle`, `sequence_memory_game`, `bonus`, `boss`
- `Duration` — `uint64`, stage time limit in milliseconds
- `ClearConditions` — `[]condition.Model`
- `Rewards` — `[]reward.Model`
- `WarpType` — `string`, one of: `all`, `none`
- `Properties` — `map[string]any`, stage-type-specific configuration

### Invariants

- Stage indexes are expected to be sequential starting from 0

---

## reward

### Responsibility

Represents a reward granted upon stage or quest completion.

### Core Models

**reward.Model** — Immutable model with private fields and getters.

- `Type` — `string`, one of: `experience`, `item`, `random_item`
- `Amount` — `uint32`
- `Items` — `[]WeightedItem`

**reward.WeightedItem** — Value object for weighted random item selection.

- `TemplateId` — `uint32`
- `Weight` — `uint32`
- `Quantity` — `uint32`

### Invariants

- `Type` is required
- `TemplateId` is required for `WeightedItem`
- `random_item` rewards must have at least one item

---

## instance

### Responsibility

Represents a live, running party quest instance. Tracks participant characters, current stage, timers, and per-stage state. Instances are held in an in-memory registry (not persisted to the database).

### Core Models

**instance.Model** — Mutable via copy-on-write setters. Private fields with getters.

- `Id` — `uuid.UUID`
- `TenantId` — `uuid.UUID`
- `DefinitionId` — `uuid.UUID`, references the definition this instance is based on
- `QuestId` — `string`
- `State` — `State` (string enum)
- `WorldId` — `world.Id`
- `ChannelId` — `channel.Id`
- `PartyId` — `uint32`
- `Characters` — `[]CharacterEntry`
- `CurrentStageIndex` — `uint32`
- `StartedAt` — `time.Time`
- `StageStartedAt` — `time.Time`
- `RegisteredAt` — `time.Time`
- `FieldInstances` — `[]uuid.UUID`
- `StageState` — `StageState`
- `AffinityId` — `uint32`

**instance.CharacterEntry** — Value object.

- `CharacterId` — `uint32`
- `WorldId` — `world.Id`
- `ChannelId` — `channel.Id`

**instance.StageState** — Mutable tracking state for the current stage.

- `ItemCounts` — `map[uint32]uint32`
- `MonsterKills` — `map[uint32]uint32`
- `Combination` — `[]uint32`
- `Attempts` — `uint32`
- `CustomData` — `map[string]any`

### State Transitions

```
registering -> active    (Start / registration timer expiry)
active      -> clearing  (StageClearAttempt with conditions met)
clearing    -> active    (StageAdvance to next stage)
active      -> completed (StageAdvance past last non-bonus stage)
active      -> failed    (Forfeit / global timer expiry)
completed   -> active    (bonus stage entry, if last stage is bonus type)
any         -> destroyed (Destroy removes from registry)
```

### Processors

**instance.Processor** — Interface + `ProcessorImpl`. Created via `NewProcessor(l, ctx, db)`.

- `Register(mb)(questId, partyId, channelId, mapId, characters)` — Creates a new instance. For `party` registration, resolves all party members via REST. For `individual` registration, resolves affinity and joins an existing registering instance if one matches. Emits `INSTANCE_CREATED` event. If mode is `instant`, calls `Start`. If mode is `timed`, emits `REGISTRATION_OPENED`.
- `RegisterAndEmit(...)` — Side-effecting wrapper around `Register`.
- `Start(mb)(instanceId)` — Transitions instance from `registering` to `active`. Sets stage 0, generates stage state (e.g., combination for puzzle stages), warps characters to stage maps, emits `STARTED` event.
- `StartAndEmit(instanceId)` — Side-effecting wrapper.
- `StageClearAttempt(mb)(instanceId)` — Evaluates clear conditions for the current stage. If met, transitions to `clearing` and emits `STAGE_CLEARED`. If not met, no-op.
- `StageClearAttemptAndEmit(instanceId)` — Side-effecting wrapper.
- `StageAdvance(mb)(instanceId)` — Advances to the next stage. If no more stages, calls `complete`. Otherwise, updates stage index, generates new stage state, warps characters, emits `STAGE_ADVANCED`.
- `StageAdvanceAndEmit(instanceId)` — Side-effecting wrapper.
- `Forfeit(mb)(instanceId)` — Transitions to `failed`, emits `FAILED` event, then destroys the instance.
- `ForfeitAndEmit(instanceId)` — Side-effecting wrapper.
- `UpdateStageState(instanceId, itemCounts, monsterKills)` — Accumulates item counts and monster kills into the current stage state.
- `Destroy(mb)(instanceId, reason)` — Warps all characters to the exit map, emits `INSTANCE_DESTROYED`, removes from registry.
- `DestroyAndEmit(instanceId, reason)` — Side-effecting wrapper.
- `TickGlobalTimer(mb)` — Checks all active instances for global timer expiry. Expired instances are failed and destroyed with reason `time_expired`.
- `TickGlobalTimerAndEmit()` — Side-effecting wrapper.
- `TickStageTimer(mb)` — Checks all active instances for stage timer expiry. Expired stages auto-advance.
- `TickStageTimerAndEmit()` — Side-effecting wrapper.
- `TickBonusTimer(mb)` — Checks active instances on bonus stages for timer expiry. Expired bonus stages destroy the instance with reason `bonus_expired`.
- `TickBonusTimerAndEmit()` — Side-effecting wrapper.
- `TickRegistrationTimer(mb)` — Checks registering instances for registration window expiry. Expired registrations auto-start.
- `TickRegistrationTimerAndEmit()` — Side-effecting wrapper.
- `GracefulShutdown(mb)` — Destroys all instances for the tenant with reason `shutdown` and clears the registry.
- `GracefulShutdownAndEmit()` — Side-effecting wrapper.
- `GetById(instanceId)` — Retrieves an instance from the registry.
- `GetByCharacter(characterId)` — Retrieves an instance by character ID via the registry's character index.
- `GetAll()` — Returns all instances for the tenant.

### Registry

**instance.Registry** — Singleton via `sync.Once`, thread-safe via `sync.Mutex` + per-tenant `sync.RWMutex`.

- In-memory storage keyed by `tenant.Model` and `uuid.UUID`
- Maintains a character-to-instance index for O(1) lookups
- Supports `Create`, `Get`, `GetByCharacter`, `GetAll`, `Update`, `Remove`, `Clear`

---

## party (cross-service client)

### Responsibility

REST client for resolving party membership from `atlas-parties`.

### Processors

**party.Processor** — Interface + `ProcessorImpl`. Created via `NewProcessor(l, ctx)`.

- `GetById(partyId)` — Fetches party with members
- `GetByMemberId(memberId)` — Finds party containing a character
- `GetMembers(partyId)` — Fetches party member list

---

## guild (cross-service client)

### Responsibility

REST client for resolving guild membership from `atlas-guilds`.

### Processors

**guild.Processor** — Interface + `ProcessorImpl`. Created via `NewProcessor(l, ctx)`.

- `GetByMemberId(memberId)` — Finds guild containing a character

---

## tenant (cross-service client)

### Responsibility

REST client for loading all tenants from `atlas-tenants` at startup.

### Processors

**tenant.Processor** — Interface + `ProcessorImpl`. Created via `NewProcessor(l, ctx)`.

- `GetAll()` — Fetches all registered tenants
