# chore/todo-saga-race — Fix character-creation saga race against inventory-compartment creation

## 1. Problem

The character-creation saga advances `create_character` → `award_item_0`
(action `AwardAsset`, dispatched as a `CREATE_ASSET` command) the moment
`EVENT_TOPIC_CHARACTER_STATUS` arrives with `Type=CREATED`. atlas-inventory
independently consumes the same `CHARACTER_STATUS.CREATED` event and, in a
separate transaction, creates the 5 inventory compartments
(Equipable/Use/Setup/Etc/Cash). When Postgres latency exceeds the gap between
`CHARACTER_CREATED` reception in saga-orchestrator and the
`compartment.GetByCharacterAndType` lookup inside atlas-inventory's
`AwardAsset` handler, the lookup returns `record not found`, no asset event
is emitted, the saga's `award_item_0` step never completes, the saga
backstop timer fires at 10s, compensation runs, and the character is
deleted.

The race always existed; pre-migration single-node Postgres committed
compartments inside the ~67ms window between the two events. Cross-node
Postgres (multi-namespace deploy, observed 2026-05-15 on atlas-pr-461)
loses the window consistently.

### Code locations

- Race trigger: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/event_acceptance.go:161` advances `create_character` immediately on `EventKindCharacterCreated`.
- Inventory's reactive create: `services/atlas-inventory/atlas.com/inventory/kafka/consumer/character/consumer.go:43-53` calls `inventory.CreateAndEmit(uuid.New(), e.CharacterId)` — note the freshly generated `uuid.New()`, *not* the saga's transactionId.
- Lookup that fails: `services/atlas-inventory/atlas.com/inventory/compartment/processor.go` `GetByCharacterAndType` (called from the `AwardAsset` handler).
- Builder that constructs the racy saga: `services/atlas-character-factory/atlas.com/character-factory/factory/processor.go:174-247` (`buildCharacterCreationSaga`) and `:330-402` (`buildPresetCharacterCreationSaga`).

## 2. Approach

Insert a new `AwaitInventoryCreated` saga action between `create_character`
and the first inventory-touching step. The orchestrator marks the await
step pending; atlas-inventory's compartment-creation tx emits an
`INVENTORY_STATUS.CREATED` event carrying the saga's `TransactionId`; the
orchestrator's new consumer for `EVENT_TOPIC_INVENTORY_STATUS` advances
the await step on receipt. Only then does the orchestrator dispatch the
first `AwardAsset`.

This is the same shape as the pre-existing-but-unused
`AwaitCharacterCreated` action (`libs/atlas-saga/model.go:135`,
`payloads.go:615-620`, `event_acceptance.go:162`): a passive wait keyed
off a status event from another service. Nothing in the orchestrator
currently consumes `EVENT_TOPIC_INVENTORY_STATUS` — that's the missing
piece we add here.

### Alternatives considered

| Option | Why rejected |
|---|---|
| **Explicit `CreateInventory` orchestrator step** (saga dispatches a command to atlas-inventory instead of relying on the reactive consumer) | Cleaner long-term but bigger blast radius: requires a new command topic + handler, deprecating the existing `CHARACTER_STATUS.CREATED → CreateInventory` side-effect contract, and auditing every other CharacterCreated emission path. Out of scope for a chore branch. |
| **Retry inside atlas-inventory's `AwardAsset` handler** when compartment is missing | Hides the control-flow problem under a sleep loop; couples retry budget to the saga timeout; doesn't address the symmetric race for `equip_*` steps. |
| **Create compartments synchronously in atlas-character's transaction** | Violates service ownership (atlas-character writing inventory tables). Not viable. |

## 3. Concrete changes

### 3.1 `libs/atlas-saga`

- `model.go`: add `AwaitInventoryCreated Action = "await_inventory_created"` alongside the existing character-creation actions.
- `payloads.go`: add
  ```go
  type AwaitInventoryCreatedPayload struct {
      CharacterId uint32 `json:"characterId"` // set by orchestrator via result-forwarding
  }
  ```
  Mirrors `AwaitCharacterCreatedPayload`'s shape but keyed off characterId — the result-forward (`event_acceptance.go:135` `StepCompletedWithResult`) already injects `characterId` into subsequent steps' payloads via the existing forwarding mechanism, so the await step receives it the same way every downstream step does.
- `unmarshal.go`: add a `case AwaitInventoryCreated:` branch alongside the existing `case AwaitCharacterCreated:` block. Add a unit test entry in `unmarshal_test.go`.

### 3.2 `services/atlas-inventory`

**Wire-format addition on `inventory.StatusEvent`** (`kafka/message/inventory/kafka.go`):

```go
type StatusEvent[E any] struct {
    TransactionId uuid.UUID `json:"transactionId,omitempty"` // NEW
    CharacterId   uint32    `json:"characterId"`
    Type          string    `json:"type"`
    Body          E         `json:"body"`
}
```

Plus a new event-type constant:

```go
const (
    StatusEventTypeCreated        = "CREATED"
    StatusEventTypeCreationFailed = "CREATION_FAILED" // NEW
    StatusEventTypeDeleted        = "DELETED"
)

type CreationFailedStatusEventBody struct {
    Reason string `json:"reason"` // free-form, for log telemetry; orchestrator just fails the step
}
```

`omitempty` keeps the addition backward-compatible for any third-party
consumer that doesn't yet know about `TransactionId`. The orchestrator's
new consumer treats `uuid.Nil` as "not saga-correlated, skip"
(consistent with `SkipReasonNilTransactionId` at
`event_acceptance.go:221`).

**Producer changes** (`services/atlas-inventory/atlas.com/inventory/inventory/producer.go`):

- `CreatedEventStatusProvider` takes `transactionId uuid.UUID` and embeds it in the event.
- New `CreationFailedEventStatusProvider(transactionId, characterId, reason)`.

**Processor change** (`services/atlas-inventory/atlas.com/inventory/inventory/processor.go:67-112`):

`CreateAndEmit` and `Create` already accept `transactionId uuid.UUID` —
the existing code path threads it through. The success-path emit at
line 103 becomes:

```go
return mb.Put(inventory2.EnvEventTopicStatus, CreatedEventStatusProvider(transactionId, characterId))
```

Add a failure-path emit: when `database.ExecuteTransaction` returns
`txErr` at line 105, the function currently logs and returns. Instead,
emit `CreationFailedEventStatusProvider(transactionId, characterId, txErr.Error())`
on a *new* `message.Emit` (we cannot reuse the in-flight buffer because
the transaction failed). Then return the error as today.

**Consumer change** (`services/atlas-inventory/atlas.com/inventory/kafka/consumer/character/consumer.go:43-53`):

Replace `uuid.New()` with `e.TransactionId`:

```go
_, err := inventory.NewProcessor(l, ctx, db).CreateAndEmit(e.TransactionId, e.CharacterId)
```

`character.StatusEvent` already carries `TransactionId`
(`services/atlas-character/atlas.com/character/kafka/message/character/kafka.go:55`),
so this is a one-line swap — no upstream wire-format change needed.

### 3.3 `services/atlas-saga-orchestrator`

**Event acceptance** (`saga/event_acceptance.go`):

```go
EventKindInventoryCreated        EventKind = "inventory.created"          // NEW
EventKindInventoryCreationFailed EventKind = "inventory.creation_failed"  // NEW

// in acceptanceTable:
sharedsaga.AwaitInventoryCreated: {EventKindInventoryCreated, EventKindInventoryCreationFailed},
```

Add the new action to the `event_acceptance_test.go` coverage assertion
(it iterates `sharedsaga.Action` constants and fails if any lack a table
entry).

**Handler dispatch** (`saga/handler.go:703`, inside `GetHandler`):

```go
case AwaitInventoryCreated:
    return h.handleAwaitInventoryCreated, true
```

`handleAwaitInventoryCreated` is a no-op — the step is passive and is
advanced by the inbound event handler, not by command dispatch. It must
exist purely to keep the dispatcher's unknown-action guard
(`processor.go:947`) happy. Pattern: return `nil` immediately.

**New consumer** (`kafka/consumer/inventory/consumer.go`, new package):

Mirror the existing `kafka/consumer/character/consumer.go` shape:

- `InitConsumers` registers a consumer on `EVENT_TOPIC_INVENTORY_STATUS`
  with `consumer_group_id = "saga_orchestrator"`,
  `consumer.SetStartOffset(kafka.LastOffset)` (matches the character consumer at line 21).
- `InitHandlers` registers two handlers: `handleInventoryCreatedEvent`
  and `handleInventoryCreationFailedEvent`.
- Each handler checks `e.Type`, calls `p.AcceptEvent(e.TransactionId, EventKind*)`,
  and dispatches `p.StepCompleted(e.TransactionId, true|false)`.
- `e.TransactionId == uuid.Nil` is filtered by the existing
  `AcceptEvent` skip path (`SkipReasonNilTransactionId`).

Register the new consumer in `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/main.go`
alongside the character-status registration.

**Kafka message struct** (`kafka/message/inventory/kafka.go`, new file):

Copy the shape of the atlas-inventory struct so the orchestrator can
deserialize what atlas-inventory produces. Keep `EnvEventTopicStatus =
"EVENT_TOPIC_INVENTORY_STATUS"` and the two status-type constants.

### 3.4 `services/atlas-character-factory`

In **both** `buildCharacterCreationSaga` and
`buildPresetCharacterCreationSaga`, insert the await step immediately
after `create_character`:

```go
builder.AddStep("await_inventory_created", saga.Pending, saga.AwaitInventoryCreated, saga.AwaitInventoryCreatedPayload{
    CharacterId: 0, // forwarded by orchestrator after create_character completes
})
```

The orchestrator's `forwardCharacterCreationResult`
(`saga/processor.go:1418`) already substitutes `CharacterId=0` sentinels
in every remaining pending step's payload with the value emitted by
`handleCharacterCreatedEvent`
(`StepCompletedWithResult` at `consumer/character/consumer.go:135`).
The await step inherits the same forwarding for free.

### 3.5 Kafka topic configuration

`EVENT_TOPIC_INVENTORY_STATUS` already exists (atlas-inventory emits to
it today; the cash-shop service consumes it). No new topic to create or
configmap entry to add — verify against `dev/k8s/.../env-configmap.yaml`
during implementation; documentation only.

## 4. Failure handling

| Scenario | Behaviour |
|---|---|
| Compartments commit fast (success) | `INVENTORY_STATUS.CREATED` emitted with saga's TransactionId. Orchestrator advances `await_inventory_created`, dispatches `award_item_0`. |
| Compartments commit slow (the race we're fixing) | Saga blocks on `await_inventory_created` until the event arrives. Default saga timeout is 10s; if compartment-creation exceeds 10s the saga still compensates, but now for a *real* infrastructure problem rather than a phantom race. |
| Compartment-creation tx fails | atlas-inventory emits `INVENTORY_STATUS.CREATION_FAILED`. Orchestrator's new consumer calls `StepCompleted(false)`, the saga fails fast, compensation deletes the half-created character. No 10s wait. |
| `INVENTORY_STATUS.CREATED` arrives with `TransactionId == uuid.Nil` (some non-saga path created an inventory) | `AcceptEvent` returns `false` via the existing nil-txn skip; no saga state mutated. |
| Same event seen twice (Kafka redelivery) | `AcceptEvent` returns `false` on the second delivery — the step is already `Completed`, no pending step matches. Same idempotency as every other event handler. |

## 5. Testing

### Unit

- `libs/atlas-saga/unmarshal_test.go`: add a case for `AwaitInventoryCreated` round-trip.
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/event_acceptance_test.go`: extend the coverage assertion. Add explicit test rows for `(AwaitInventoryCreated, EventKindInventoryCreated)` → accept, `(AwaitInventoryCreated, EventKindInventoryCreationFailed)` → accept, `(AwaitInventoryCreated, EventKindCompartmentCreated)` → reject.
- New consumer-level test: `inventory_consumer_test.go` asserting `handleInventoryCreatedEvent` calls `StepCompleted(true)` when the event matches a pending `AwaitInventoryCreated` step, and skips otherwise.

### Integration

- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/createandequip_integration_test.go` and `preset_integration_test.go`: extend to assert the `await_inventory_created` step exists in the saga and is completed before any `award_asset` step transitions to `Completed`.
- New integration test: simulate the race — emit `CHARACTER_STATUS.CREATED` followed by a delayed `INVENTORY_STATUS.CREATED`. Assert that `award_item_0` doesn't dispatch until the inventory event arrives. (Use `processor_testseam.go`-style hooks to introspect dispatch order.)

### Manual / e2e

- Verify in a multi-namespace PR environment (the original repro): create a character, confirm no `SAGA_TIMEOUT` on `award_item_0`, confirm character row persists, confirm 5 compartment rows exist.

## 6. Backward compatibility

- Adding `TransactionId uuid.UUID \`json:"transactionId,omitempty"\`` to `inventory.StatusEvent` is purely additive at the JSON wire level. Existing consumers (cash-shop, anything else on `EVENT_TOPIC_INVENTORY_STATUS`) decode the same struct shape with an extra ignored field; the producer omits the field when zero, matching the prior on-wire form.
- The new `CREATION_FAILED` event type is opt-in for consumers — they continue to filter on `e.Type == "CREATED"` or `"DELETED"` and ignore unknown types. The orchestrator is the only intended consumer.
- The new `AwaitInventoryCreated` saga action is opt-in for saga authors. Existing sagas that don't use it are unaffected; only character-factory's two builders are amended.
- The pre-existing unused `AwaitCharacterCreated` action is left alone — out of scope.

## 7. Out of scope

- Migrating to an explicit `CreateInventory` orchestrator-dispatched step (Option B). If the team wants to consolidate command flow, that's a follow-up task; this change keeps atlas-inventory's reactive consumer as today.
- Wiring `AwaitCharacterCreated` into any saga (the constant exists but no saga uses it — a related follow-up, not required for this fix).
- Adding similar await steps for other "creation cascades" (e.g., skills service, buddy-list service). Only inventory has demonstrated the race; mirror this pattern in those services *if and when* a race shows up there.
- Topic configuration changes — `EVENT_TOPIC_INVENTORY_STATUS` already exists.

## 8. Sequencing

The work has natural Docker-build dependencies (per `CLAUDE.md` §Build & Verification):

1. `libs/atlas-saga` (Action constant, payload, unmarshal switch) — pure Go, no Docker.
2. `services/atlas-inventory` (wire-format additive change, producer/processor/consumer threading the transactionId) — `docker build -f services/atlas-inventory/Dockerfile .`.
3. `services/atlas-saga-orchestrator` (acceptance table, handler stub, new consumer, kafka message struct, main.go wiring) — `docker build -f services/atlas-saga-orchestrator/Dockerfile .`.
4. `services/atlas-character-factory` (builder change in both code paths) — `docker build -f services/atlas-character-factory/Dockerfile .`.

Atlas-inventory's wire-format change ships with `omitempty`, so step 2 can deploy before step 3 without breaking the cash-shop consumer.
