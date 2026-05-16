# Backend Audit — chore/todo-saga-race

- **Worktree:** `.worktrees/chore-todo-saga-race`
- **Branch:** `chore/todo-saga-race`
- **Diff range:** `9ae8c3fcb..d4af6ee07`
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-05-16
- **Build:** PASS (per user — not re-run)
- **Tests:** PASS (per user — not re-run)
- **Overall:** NEEDS-WORK

## Scope

All packages touched in this diff are **support packages** (Kafka producer/consumer wiring, message-schema mirrors, saga step plumbing, factory step composition). None contain a `model.go` defining a DDD aggregate — no `builder.go` / `entity.go` / `rest.go` / `administrator.go` checklist applies to the new packages. The atlas-inventory `inventory/inventory` package already had a `model.go`; the changes there are limited to producer signatures and an error-emit branch in the existing `Create` function, so no domain-shape regression is introduced.

The relevant checks therefore reduce to:

- Build/test/vet/gofmt hygiene.
- Kafka producer/consumer conventions (`message.Buffer`, `message.Emit`, curried `InitConsumers(l)(cmf)(groupId)`, `consumer.SetHeaderParsers(SpanHeaderParser, TenantHeaderParser)`, `consumer.SetStartOffset(kafka.LastOffset)`).
- DOM-21: redundant types vs `libs/atlas-constants/`.
- DOM-23: Kafka topic name in `env-configmap.yaml`, no literal overrides in service deployments.
- Wire-format back-compat for `inventory.StatusEvent`.
- Saga lifecycle correctness (`AcceptEvent` → `StepCompleted`, `forwardCharacterCreationResult` payload substitution, dispatcher unknown-action guard).

## Findings

### Blocking

| ID | Description | Evidence |
|----|-------------|----------|
| FMT-01 | `libs/atlas-saga/model.go` is not `gofmt`-clean. The diff added `AwaitInventoryCreated` to the `// Character creation actions` block, and the `Action` column alignment in that block is now wrong: `gofmt` re-aligns `CreateCharacter        Action` → `CreateCharacter       Action` (one extra space currently present). | `libs/atlas-saga/model.go:133-137`. Verified with `gofmt -l libs/atlas-saga/model.go` (file listed) and `gofmt -d libs/atlas-saga/model.go` (shows the realignment in the character-creation block). Pre-existing column drift in other blocks (`InventoryTransaction`, `ShowStorage`, `FieldEffect`, `Action = "ui_lock"`) also surfaces from the same command; if the intent was to clean only the touched block, the touched block still fails. CI's gofmt gate will reject this. |
| FMT-02 | `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go` is not `gofmt`-clean. The diff added `AwaitInventoryCreated = sharedsaga.AwaitInventoryCreated` to the character-creation re-export block; the `=` column for `CreateCharacter` and `AwaitInventoryCreated` is misaligned (one extra space after `CreateCharacter`). | `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go:142-143`. Verified with `gofmt -l ./saga/model.go` (file listed). |

`go vet ./...` is clean in both modules, and `go build` / `go test` were reported clean by the submitter, so this is a formatting gate only — but CI's `gofmt` check is a hard fail, so it is blocking.

### Non-blocking observations

| ID | Description | Evidence |
|----|-------------|----------|
| OBS-01 | `services/atlas-inventory/atlas.com/inventory/inventory/producer.go:34` `DeletedEventStatusProvider(characterId uint32)` was NOT updated to accept a `transactionId`, while the new sibling providers (`CreatedEventStatusProvider`, `CreationFailedEventStatusProvider`) and the other Deleted providers in this service (`compartment/producer.go:28`, `asset/producer.go:66`) all do. The saga orchestrator's new inventory consumer only registers handlers for `CREATED` and `CREATION_FAILED` (`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/inventory/consumer.go:30-35`), so the existing `DELETED` emit will continue to ship without a `transactionId` field, but any future compensation step that wants to wait on inventory deletion will need this column reworked. Non-blocking because no current consumer reads it; document or defer. |
| OBS-02 | `services/atlas-inventory/atlas.com/inventory/kafka/message/inventory/kafka.go:15-20` removed `omitempty` from `TransactionId` (commit `1d701522d`). The wire format now always emits `"transactionId":"00000000-0000-0000-0000-000000000000"` for non-saga emits. The only current consumer of this topic is the new saga-orchestrator handler at `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/inventory/consumer.go:44-46`, which calls `AcceptEvent(e.TransactionId, ...)`; `AcceptEvent` will fail to find a saga keyed by `uuid.Nil` and the handler returns early. Back-compat is preserved as documented in the kafka.go header comment. Verified by grepping `services/ libs/` for `EVENT_TOPIC_INVENTORY_STATUS` — only producer + the new mirror consumer. |
| OBS-03 | Mirror struct duplication between `services/atlas-inventory/atlas.com/inventory/kafka/message/inventory/kafka.go` and `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/inventory/kafka.go` is the established pattern in this repo — every consumer service under `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/` mirrors the producer's struct (e.g. `character/`, `compartment/`, `asset/`, `skill/`, `quest/`, `pet/`, `guild/`). The new `inventory/` mirror has a `// Mirrors ...` doc comment at the top and otherwise matches the producer schema. Not a smell in context. |
| OBS-04 | The CREATION_FAILED error-emit path (`services/atlas-inventory/atlas.com/inventory/inventory/processor.go:107-113`) uses a fresh `message.Emit(...)` call, which builds a new buffer (`libs/atlas-kafka/.../message/message.go:46 NewBuffer()`), so the rolled-back inner buffer's `CREATED` event is correctly discarded. The outer `CreateAndEmit` (`processor.go:67-75`) short-circuits via `Emit`'s `if err != nil { return err }` (`message/message.go:48-50`), so the rolled-back `mb` is never flushed to Kafka either. Correctness PASS. The inline comment at `processor.go:107-108` is accurate. |
| OBS-05 | The saga-orchestrator inventory consumer (`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/inventory/consumer.go:18-23`) uses the curried `InitConsumers(l)(rf)(consumerGroupId)` shape, registers via `consumer2.NewConfig(l)("inventory_status_event")(...EnvEventTopicInventoryStatus)(consumerGroupId)`, sets `SpanHeaderParser, TenantHeaderParser`, and `consumer.SetStartOffset(kafka.LastOffset)` — exact parity with the sibling `character/consumer.go:18-23`. PASS. |
| OBS-06 | `main.go:98` wiring (`inventoryConsumer.InitConsumers(l)(cmf)(consumerGroupId)`) is placed in alphabetical order between `guild` (line 97) and `pet` (line 99). PASS. |
| OBS-07 | `handleAwaitInventoryCreated` is wired into the dispatcher's switch at `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go:753-754` and defined as a no-op at lines 2867-2870 (with an explanatory comment pointing at the actual advancer in the kafka consumer). This satisfies the "unknown-action guard" the comment references at `saga/processor.go:947`. PASS. |
| OBS-08 | `forwardCharacterCreationResult` in `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/processor.go:1455-1460` was extended with a `case AwaitInventoryCreatedPayload` so the sentinel `CharacterId=0` in `AwaitInventoryCreatedPayload` is rewritten with the real id emitted by `handleCharacterCreatedEvent`. The integration test at `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/await_inventory_created_integration_test.go:55-60` asserts both the await and award payloads receive the substituted id. PASS. |
| OBS-09 | DOM-21 (atlas-constants): the new `AwaitInventoryCreatedPayload.CharacterId uint32` (`libs/atlas-saga/payloads.go:626`) and the new producer signatures (`inventory/producer.go:12, 23`) use raw `uint32` rather than `character.Id` from `libs/atlas-constants/character/constants.go:3`. However, every existing payload in `libs/atlas-saga/payloads.go` uses `uint32` (verified: lines 16, 31, 37, 48, 57, 65, 76, 84, 94, 103 all `CharacterId uint32`), so the new code matches the established convention in this lib. No DOM-21 regression introduced by this diff. Migrating the whole lib to `character.Id` is out of scope. |
| OBS-10 | DOM-23 (Kafka topic naming): `EVENT_TOPIC_INVENTORY_STATUS` is defined in `deploy/k8s/base/env-configmap.yaml:109` with the required `KEY: "KEY"` shape; `deploy/k8s/base/atlas-saga-orchestrator.yaml` uses `envFrom: configMapRef: name: atlas-env` (lines 21-23) and contains no literal `- name: EVENT_TOPIC_INVENTORY_STATUS` override. PASS. |
| OBS-11 | DOM-22 (Dockerfile/lib drift): no `go.mod` or `Dockerfile` is modified in this diff range (`git diff --stat ... -- services/<svc>/go.mod services/<svc>/Dockerfile libs/atlas-saga/go.mod` returns empty). atlas-saga is already a direct dependency of saga-orchestrator and character-factory, and no new `Chronicle20/atlas/libs/*` was introduced for atlas-inventory. Not applicable to this diff. |
| OBS-12 | The `handleInventoryCreatedEvent` and `handleInventoryCreationFailedEvent` handlers (`kafka/consumer/inventory/consumer.go:40-69`) ignore the `StepCompleted` return value with `_ =`. This is the established pattern in every sibling consumer (e.g. `character/consumer.go:72, 83, 94, 105, ...`). Not a regression. |
| OBS-13 | `event_acceptance.go:164-167` declares the acceptance entries with the comment `// Inventory (rollup of all compartments for a character).` and explicitly distinguishes the rollup from per-compartment events. The negative assertion in `event_acceptance_test.go:172-173` proves `EventKindCompartmentCreated` is NOT accepted by `AwaitInventoryCreated`. PASS. |

## Summary

### Blocking (must fix)

- FMT-01: `libs/atlas-saga/model.go` not gofmt-clean (touched lines).
- FMT-02: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go` not gofmt-clean (touched lines).

Run `gofmt -w libs/atlas-saga/model.go libs/atlas-saga/payloads.go services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go` before opening the PR. (Note: `payloads.go` also shows up in `gofmt -l` from pre-existing alignment drift, not from this diff; fixing it is harmless.)

### Non-blocking (consider)

- OBS-01: `inventory/producer.go` `DeletedEventStatusProvider` is the odd one out — every other Deleted provider in atlas-inventory takes a `transactionId`. Worth threading it through for consistency the next time compensation needs to wait on inventory deletion. Defer.
