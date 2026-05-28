# Context — chore/todo-saga-race

## Problem (one-liner)

Character-creation saga dispatches `AwardAsset` immediately on `CHARACTER_STATUS.CREATED`, but `atlas-inventory` creates inventory compartments in a separate transaction reactively from the same event. Under cross-node Postgres latency the lookup races, `compartment.GetByCharacterAndType` returns `record not found`, the asset event never fires, the saga's 10s backstop fires, compensation deletes the character.

## Fix (one-liner)

Insert an `AwaitInventoryCreated` saga step between `create_character` and the first asset/equipment step. atlas-inventory's compartment-creation tx emits `INVENTORY_STATUS.CREATED` carrying the saga `TransactionId`; the orchestrator's new consumer for `EVENT_TOPIC_INVENTORY_STATUS` advances the await step on receipt.

## Key Files

### libs/atlas-saga

- `model.go` — `Action` constants. Add `AwaitInventoryCreated` next to existing `AwaitCharacterCreated` (line 135).
- `payloads.go` — payload structs. Add `AwaitInventoryCreatedPayload` after `AwaitCharacterCreatedPayload` (line 616).
- `unmarshal.go` — switch on `Action`. Add `case AwaitInventoryCreated:` branch (alongside line 396 `AwaitCharacterCreated` case).
- `unmarshal_test.go` — TDD entry point for the new payload's JSON round-trip.

### services/atlas-inventory

- `kafka/message/inventory/kafka.go` — wire-format struct `StatusEvent[E]`. Currently no `TransactionId`. Add it with `json:"transactionId,omitempty"`. Add `StatusEventTypeCreationFailed` constant and `CreationFailedStatusEventBody` struct.
- `inventory/producer.go` — `CreatedEventStatusProvider`. Update to take `transactionId uuid.UUID` and embed it. Add `CreationFailedEventStatusProvider(transactionId, characterId, reason)`.
- `inventory/processor.go` — `Create` at line 77. Already threads `transactionId`. Update line 103 emit to pass `transactionId`. Add failure-path emit when `database.ExecuteTransaction` returns `txErr` at line 105.
- `kafka/consumer/character/consumer.go` — `handleStatusEventCreated` at line 43-53. Replace `uuid.New()` at line 48 with `e.TransactionId`. The upstream `character.StatusEvent` already carries `TransactionId` (see `kafka/message/character/kafka.go:15`).

### services/atlas-saga-orchestrator

- `saga/event_acceptance.go` — add two `EventKind*` constants (after line 64) and one acceptance-table entry (after line 162). Pattern mirrors `AwaitCharacterCreated`.
- `saga/event_acceptance_test.go` — extend `allActions` (line 12) and add test rows for new kinds.
- `saga/handler.go` — `GetHandler` switch at line 703. Add `case AwaitInventoryCreated:` returning `h.handleAwaitInventoryCreated`. Implement `handleAwaitInventoryCreated` as a no-op (return nil) so the dispatcher's unknown-action guard (`processor.go:947`) doesn't fail. Step is advanced by the inbound event, not by command dispatch.
- `kafka/message/inventory/kafka.go` — **new file**. Mirror atlas-inventory's struct shape so the orchestrator can deserialize.
- `kafka/consumer/inventory/consumer.go` — **new file**. Mirror `kafka/consumer/character/consumer.go` shape. Two handlers (`handleInventoryCreatedEvent`, `handleInventoryCreationFailedEvent`), each calling `p.AcceptEvent(e.TransactionId, ...)` then `p.StepCompleted(...)`.
- `main.go` — register the new consumer alongside character (`inventory.InitConsumers(l)(cmf)(consumerGroupId)` at line 93, then `inventory.InitHandlers` at line 115).

### services/atlas-character-factory

- `factory/processor.go` — `buildCharacterCreationSaga` (line 174) and `buildPresetCharacterCreationSaga` (line 333). Insert an `await_inventory_created` step immediately after `create_character` in both. Payload `CharacterId: 0` — forwarded by orchestrator's existing `forwardCharacterCreationResult` (substitutes the sentinel after `create_character` completes).

## Key Mechanics

- **Result forwarding**: `handleCharacterCreatedEvent` (`kafka/consumer/character/consumer.go:135`) calls `p.StepCompletedWithResult(e.TransactionId, true, map[string]any{"characterId": e.CharacterId})`. The orchestrator's `forwardCharacterCreationResult` (`saga/processor.go:1418`) walks remaining pending steps and substitutes any `CharacterId: 0` sentinel in their payloads with the actual id. This is why the await step uses `CharacterId: 0` — inheritance is free.
- **Skip on nil txid**: `event_acceptance.go:221` defines `SkipReasonNilTransactionId`. Any `INVENTORY_STATUS.CREATED` event with `TransactionId == uuid.Nil` is filtered by the existing `AcceptEvent` path — non-saga inventory creations are ignored.
- **Backward-compat on wire**: `TransactionId uuid.UUID \`json:"transactionId,omitempty"\`` is purely additive. Existing consumers (`atlas-cashshop` is the only other listener on `EVENT_TOPIC_INVENTORY_STATUS`) decode the same struct shape with an extra ignored field.
- **Topic config**: `EVENT_TOPIC_INVENTORY_STATUS` already exists in `dev/k8s/.../env-configmap.yaml`. No new topic to provision.

## Dockerfile Touchpoints

- `libs/atlas-saga` is already wired into atlas-saga-orchestrator's and atlas-character-factory's Dockerfiles (verified at `Dockerfile:20,38,55,68` for saga-orchestrator; equivalent block in character-factory).
- atlas-inventory does NOT use `libs/atlas-saga` (it only edits its own kafka message struct). No Dockerfile changes needed.
- Per `CLAUDE.md` §Build & Verification, mandatory `docker build` after every change to a service whose `go.mod` or `Dockerfile` is touched. **No `go.mod`/`Dockerfile` touches required by this plan**, but the existing services that ingest the new lib symbols still need their Docker builds verified.

## Decisions Locked

| Decision | Rationale |
|---|---|
| Passive await step (event-driven), not active command | Mirrors existing `AwaitCharacterCreated` shape and avoids inventing a new command topic for atlas-inventory. |
| `TransactionId` field added with `omitempty` | Backward-compatible for atlas-cashshop and any third-party consumers. |
| atlas-inventory keeps its reactive `CHARACTER_STATUS.CREATED → CreateInventory` consumer | Out of scope per design §7. A future task can replace with an explicit `CreateInventory` command. |
| `AwaitCharacterCreated` left untouched | Pre-existing constant, no saga currently uses it; deferred per design §7. |
| `handleAwaitInventoryCreated` is a no-op handler | Pure presence is enough to satisfy `GetHandler` and the dispatcher's unknown-action guard at `processor.go:947`. |
| Use `e.TransactionId` (already on `character.StatusEvent`) for the reactive inventory create | One-line consumer swap; no upstream wire change. |

## Risks / Edge Cases

- **Inventory tx exceeds 10s saga timeout**: saga still compensates, but the failure mode is now a real DB problem instead of a phantom race.
- **Duplicate Kafka delivery**: `AcceptEvent` returns `false` for non-pending steps. Idempotent.
- **Non-saga inventory creations** (e.g., legacy bootstrap, manual repair): `TransactionId == uuid.Nil`, filtered by the existing nil-txid skip path. No-op for the orchestrator.
- **`omitempty` ordering**: receivers decode JSON case-insensitive and tolerant; field order is irrelevant.

## Out of Scope (per design §7)

- Migrating to an explicit `CreateInventory` orchestrator-dispatched step.
- Wiring `AwaitCharacterCreated` into any saga.
- Adding similar await steps for skills/buddy-list/etc.
- Topic configuration changes.

## Verification Commands

Per `CLAUDE.md` §Build & Verification:

```bash
# From worktree root:
cd libs/atlas-saga && go test -race ./... && go vet ./... && go build ./...
cd services/atlas-inventory/atlas.com/inventory && go test -race ./... && go vet ./... && go build ./...
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test -race ./... && go vet ./... && go build ./...
cd services/atlas-character-factory/atlas.com/character-factory && go test -race ./... && go vet ./... && go build ./...

# From worktree root (no go.mod/Dockerfile changes expected, but verify anyway):
docker build -f services/atlas-inventory/Dockerfile .
docker build -f services/atlas-saga-orchestrator/Dockerfile .
docker build -f services/atlas-character-factory/Dockerfile .
```
