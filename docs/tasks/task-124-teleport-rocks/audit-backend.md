# Backend Audit — task-124 Teleport Rocks

- **Scope:** Go changes on branch `task-124-teleport-rocks`, base `c9490b724`..`2585df656`
- **Guidelines Source:** `.claude/skills/backend-dev-guidelines/resources/*`
- **Date:** 2026-07-17
- **Build:** PASS (atlas-character, atlas-channel, libs/atlas-packet, libs/atlas-saga, libs/atlas-constants)
- **Tests:** All packages `ok` under `go test ./... -count=1` (no failures observed in any of the four modules); `go vet ./...` clean in atlas-character and atlas-channel; `tools/goroutine-guard.sh` and `tools/redis-key-guard.sh` clean from repo root.
- **Overall:** NEEDS-WORK

## Build & Test Results

```
services/atlas-character/atlas.com/character: go build ./... -> clean
services/atlas-character/atlas.com/character: go test ./... -count=1 -> ok (all packages, incl. teleport_rock, kafka/consumer/teleportrock, kafka/message/character)
services/atlas-channel/atlas.com/channel:     go build ./... -> clean
services/atlas-channel/atlas.com/channel:     go test ./... -count=1 -> ok (all packages, incl. teleportrock, character/teleportrock, kafka/consumer/teleportrock, socket/handler, socket/writer)
libs/atlas-packet:                            go build && go test ./... -> ok (incl. teleportrock, teleportrock/clientbound, teleportrock/serverbound, cash/serverbound)
libs/atlas-saga, libs/atlas-constants:         go build && go test ./... -> ok
```

No test package exceeded ~0.3s, consistent with no unstubbed Kafka producer paths in the new tests (DOM-24 spot-checked, see below).

## Domain Checklist Results

### `atlas-character/teleport_rock` (domain package — has `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | builder.go exists, Build() validates | WARN | `teleport_rock/builder.go:32-38` — `Build() Model` has no error return / invariant check. Same shape as sibling builders in this service (`saved_location/builder.go:44-52`, `character/builder.go` `Build() Model`), so this is a pre-existing service-wide characteristic, not a regression — still a documented-pattern deviation (file-responsibilities.md / patterns-functional.md require `Build() (Model, error)`). |
| DOM-02 | `Make(Entity)` in entity.go | FAIL | `teleport_rock/entity.go` has only `Migration` and `TableName` (lines 9, 22) — no `func Make(`. The domain aggregates N rows into one `Model`; `modelFromEntities(characterId, es)` in `provider.go:7` fills the role but under a different name/shape than the checklist requires. |
| DOM-03 | `ToEntity()` in entity.go | FAIL | No `func (m Model) ToEntity()` anywhere in `teleport_rock/entity.go` or `model.go` — same aggregate-shape reason as DOM-02. |
| DOM-06 | Processor accepts `FieldLogger` | PASS | `teleport_rock/processor.go:33` — `NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor`. |
| DOM-07 | Handlers pass `d.Logger()` | PASS | `teleport_rock/resource.go:28` — `NewProcessor(d.Logger(), d.Context(), d.DB())`. |
| DOM-08 | POST/PATCH use RegisterInputHandler | N/A | Domain has only a GET endpoint (`resource.go:20`); all mutation is via Kafka commands, not REST. |
| DOM-09 | Transform errors handled | PASS | `resource.go:35-38` checks the `Transform` error explicitly. |
| DOM-10 | Test DB registers tenant callbacks | PASS | `administrator_test.go:33`, `processor_test.go` (via `testDatabase`), `kafka/consumer/teleportrock/consumer_test.go:33` all call `database.RegisterTenantCallbacks(l, db)`. |
| DOM-11 | Providers use lazy evaluation | PASS | `provider.go` returns a plain value from `getByCharacterId`; called from the processor, no eager `FixedProvider` wrapping of a live query. |
| DOM-12 | No `os.Getenv` in resource.go | PASS | None found. |
| DOM-13/14 | No cross-domain logic / no direct provider calls in handlers | PASS | `resource.go:28` calls `NewProcessor(...).GetByCharacterId(characterId)` only. |
| DOM-15 | No direct entity creation in handlers | PASS | No `db.Create/Save/Delete` in `resource.go`. |
| DOM-16 | administrator.go exists | PASS | `teleport_rock/administrator.go` (create/delete via `replaceList`/`deleteByCharacterId`). |
| DOM-17 | Error → HTTP status mapping | WARN | `resource.go:30` and `:37` write `http.StatusInternalServerError` directly for *every* failure (DB error and REST-transform error) — no 404/400 distinction, and see DOM-27 below for the bare-500 issue specifically. |
| DOM-18 | JSON:API interface on RestModel | PASS | `rest.go:17-28` — `GetName`/`GetID`/`SetID`. |
| DOM-20 | Table-driven tests | PASS | `processor_test.go`, `administrator_test.go` use per-scenario `Test*` functions with clear fixtures (not classic `[]struct` tables but scenario-per-function with shared helpers — acceptable per testing-guide.md's "Focus Areas", each processor path is exercised). |
| **DOM-21** | Reuse atlas-constants types | PASS | Uses `_map.Id`, `world.Id`, `item.Id`/`item.GetClassification`/`item.ClassificationTeleportRock` (pre-existing in `libs/atlas-constants/item/constants.go:74`) throughout. `EligibleForRegistration`/`continent()` numeric logic (`teleport_rock/model.go:30-36`, `channel/teleportrock/use.go:68-70`) has no existing atlas-constants equivalent — verified no continent/event-block helper exists in `libs/atlas-constants/map/model.go`. |
| **DOM-22** | Dockerfile 4-mention rule | N/A | No new `libs/atlas-X` direct-require added to either service's `go.mod` by this diff (teleport-rock reuses `atlas-constants`, `atlas-database`, `atlas-kafka`, `atlas-model`, `atlas-tenant`, `atlas-rest`, `atlas-saga`, `atlas-packet`, `atlas-socket`, `atlas-routine`, `atlas-outbox` — all pre-existing deps of both services). |
| **DOM-23** | Kafka topic naming convention | PASS | `COMMAND_TOPIC_TELEPORT_ROCK` / `EVENT_TOPIC_TELEPORT_ROCK_STATUS` (`kafka/message/teleportrock/kafka.go:10,14`) follow the SHOUTY_SNAKE convention; both keys present in `deploy/k8s/env-configmap.yaml` as `KEY: "KEY"` (verified: `grep -c 'TELEPORT_ROCK' deploy/k8s/env-configmap.yaml` finds both), no literal override in the atlas-character/atlas-channel k8s manifests. |
| **DOM-24** | Kafka producer stubbed in tests that emit | PASS | `teleport_rock/processor_test.go` calls the pure `AddMap(mb)`/`RemoveMap(mb)` forms only (buffers into `message.Buffer`, never `AndEmit`); `kafka/consumer/teleportrock/consumer_test.go` exercises only the wrong-type no-op branch, never reaching `AndEmit`. No test in either service's new packages invokes `producer.ProviderImpl`/`message.Emit` directly — confirmed by uniformly sub-second test times (no ~42s retry-loop symptom). |
| **DOM-25** | Client wire values config-resolved | PASS | `libs/atlas-packet/teleportrock/result_body.go:31,39` — both `MapTransferResultListBody`/`MapTransferResultErrorBody` route through `atlas_packet.WithResolvedCode("operations", key, ...)`. All 9 mode keys (`DELETE_LIST` … `MAPLE_ISLAND_LEVEL7`) verified present with identical byte values across all 6 targeted seed templates (`template_gms_{83,84,87,92,95}_1.json`, `template_jms_185_1.json`), each carrying `"writer": "MapTransferResult"` with a full `operations` map. Handlers (`TeleportRockAddMapHandle`, `TeleportRockUseHandle`) are present with `"validator": "LoggedInValidator"` in all 6 templates too. |
| DOM-26 | Goroutines via routine.Go | PASS | `tools/goroutine-guard.sh` exits 0 (repo-wide; no new bare `go` statements in the diff). |
| **DOM-27** | Transient DB errors → 503, never bare 500 | **FAIL** | `teleport_rock/resource.go:30` and `:37` call `w.WriteHeader(http.StatusInternalServerError)` directly, even though `atlas-character/main.go:80-86` already registers `server.RegisterTransientErrorClassifier` and sibling domains in the **same service** already use the mandated replacement (`saved_location/resource.go:50,57,82,89` — `server.WriteErrorResponse(d.Logger())(w)(err)`). A transient DB pool-exhaustion error hitting the new `GET /characters/{characterId}/teleport-rock-maps` endpoint surfaces as a generic 500 instead of 503 + `Retry-After`. |
| **DOM-28** | No silent degradation in decorators/enrichment | **FAIL** | Three new call sites fetch the teleport-rock lists as an enrichment step for `CharacterData` and degrade on any error with only a Warn log — no `degrade.Observe(...)` call, so `atlas_enrichment_degraded_total` never increments: `socket/writer/set_field.go:33-37`, `socket/writer/cash_shop_open.go:21-25`, `socket/writer/set_itc.go:40-44` (all: `if err != nil { l.WithError(err).Warnf(...); trm = teleportrock.Model{} }`). `libs/atlas-rest/degrade/degrade.go` and `patterns-resilience.md`'s "No silent degradation" policy are both present and used elsewhere (e.g. `shopscanner/processor.go`'s comparable resolve-with-fallback logic), so this is a real, actionable gap, not an unavailable pattern. |

### `atlas-channel/character/teleportrock` (support/REST-client package — no `model.go`... has one, but no DB; classified as External HTTP Client)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | File responsibilities | PASS | Cleanly split: `model.go` (domain model), `processor.go` (Processor iface+impl), `producer.go` (Kafka producers), `requests.go` (REST client calls), `rest.go` (RestModel+Extract). No catch-all file. |
| **EXT-01** | JSON:API relationship interface stubs | **FAIL** | `character/teleportrock/rest.go:7-24` — `RestModel` has `GetName`/`GetID`/`SetID` but no `SetToOneReferenceID`/`SetToManyReferenceIDs`, contrary to `libs/atlas-rest/CLAUDE.md`'s explicit "required even when you don't care about relationships" rule (the task-037 precedent this rule exists to prevent). |
| **EXT-02** | httptest-backed integration test | **FAIL** | Package contains only `model_test.go` (tests `Model.Contains`) — no `httptest.NewServer`-backed test exercises `requestByCharacterId`/`GetByCharacterId`'s unmarshal path. |
| EXT-03 | 404 vs other-failure distinction | N/A / PASS-by-absence | `processor.go:33-35` bubbles the raw `err` from `requests.Provider` unmodified — no misclassification. The upstream `atlas-character` `GET .../teleport-rock-maps` handler never itself returns 404 (it always 200s with an aggregate, even if empty — `teleport_rock/resource.go` has no not-found branch), so there is no genuine 404 case this client needs to special-case today. |
| EXT-04 | URL via RootUrl, not hardcoded | PASS | `requests.go:14` — `requests.RootUrl("CHARACTERS")`. |

### `atlas-channel/teleportrock` (sub-domain / orchestration package — `use.go`, no `model.go`/`resource.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| SUB-01 | Business logic not in handler | PASS | `UseRock` in `teleportrock/use.go` is called from the thin `socket/handler/teleport_rock_use.go` and cash-item-use handler; validation/orchestration lives in `use.go`. |
| SUB-02 | Writes via administrator, not handler | PASS | No `db.Create/Save` here — mutation is delegated to the saga (`WarpToRandomPortal`, `DestroyAsset` steps) and to `atlas-character`'s Kafka-driven administrator. |
| SEC (slot-ownership anti-spoof) | PASS | `socket/handler/teleport_rock_use.go:61-65` verifies `itemInSlotFunc` returns the claimed `p.ItemId()` before invoking `useRockFunc`; `character_cash_item_use.go:47-51` does the equivalent `cashItemInSlotFunc` check before dispatching to any cash-item branch including teleport rock. Test coverage: `teleport_rock_use_test.go: TestTeleportRockUseHandleFunc_SlotMismatchNotInvoked` explicitly asserts a mismatched slot/item never reaches `useRockFunc`. |
| SEC (validation-before-consume) | PASS | `teleportrock/use.go`'s `UseRock` only appends the `consume_rock` (`DestroyAsset`) saga step after all five validation gates pass; `teleportrock/use_test.go: TestUseRockRejections` asserts `sagaCreated == nil` (no saga, hence no consumption) for every rejection case. |

## File Responsibilities Checklist — `atlas-character/teleport_rock`

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor in processor.go | PASS | `teleport_rock/processor.go` — interface + impl + methods, no leakage elsewhere. |
| FILE-02 | RestModel/Transform in rest.go | PASS | `teleport_rock/rest.go`. |
| FILE-03 | N/A (no outbound cross-service requests from this package) | N/A | — |
| FILE-04 | entity.go: entity struct + Migration + TableName | PASS | `teleport_rock/entity.go:9,13,22`. |
| FILE-05 | Builder/Model/administrator/provider placement | PASS | `builder.go`, `model.go`, `administrator.go`, `provider.go` each single-purpose. |
| FILE-06 | No package-named catch-all file | PASS | No `teleport_rock.go`; every file is single-purpose. |

## Multitenancy Pattern (anti-patterns.md)

| Check | Status | Evidence |
|-------|--------|----------|
| No manual `tenant_id = ?` filtering; providers/update/delete don't take `tenantId` | **FAIL** | `teleport_rock/provider.go:9` — `getByCharacterId(db *gorm.DB, tenantId uuid.UUID, characterId uint32)` builds `db.Where("tenant_id = ? AND character_id = ?", tenantId, characterId)` explicitly; `administrator.go:21` (`replaceList`) and `:43` (`deleteByCharacterId`) do the same. This is the exact anti-pattern anti-patterns.md lists twice ("Manual `Where("tenant_id = ?", ...)`" and "Passing TenantId to providers/update/delete"), and it contradicts the **sibling convention in the same service** — `saved_location/administrator.go:29,38` (`getByCharacterIdAndType(db, characterId, locationType)`, `deleteByCharacterIdAndType(db, characterId, locationType)`) take no `tenantId` at all, relying purely on `db.WithContext(ctx)` GORM callback injection. Functionally harmless today (the processor always calls these via `p.db.WithContext(p.ctx)`, so the manual filter is redundant rather than wrong), but it is a genuine, citable convention deviation, not a documented exception. |

## Security Review

Not an auth service; no SEC-01..04 applicable. Anti-spoof and validation-before-consume checks are covered under the sub-domain table above and both PASS with direct test evidence.

## Packet-Layer Quality Note (informational, not a checklist item)

`libs/atlas-packet/teleportrock/target.go`'s `hasTrailingUpdateTime` gate (commit `8c91089ba`) is well-executed: it correctly threads `updateTimeFirst` from the shared `character_cash_item_use.go` gate (`t.MajorVersion() >= 87`, matching the established `CashItemUsePointReset` convention) through to `Target.Decode`, with IDA citations for v83/v84/v87/v95/jms and byte-level round-trip tests per version (`item_use_teleport_rock_test.go`, `use_test.go`, `result_test.go`). No correctness issue found in this gate.

## Summary

### Blocking (must fix) — Important

- **DOM-27**: `teleport_rock/resource.go:30,37` — replace bare `w.WriteHeader(http.StatusInternalServerError)` with `server.WriteErrorResponse(d.Logger())(w)(err)`, matching the classifier already registered in `main.go:80` and the sibling `saved_location/resource.go` pattern.
- **DOM-28**: `socket/writer/set_field.go:33-37`, `cash_shop_open.go:21-25`, `set_itc.go:40-44` — the teleport-rock-list enrichment fetch degrades silently (Warn log only); add `degrade.Observe(l, "channel.character.teleportrock", c.Id(), err)` (or equivalent component name) at each site per `patterns-resilience.md`.
- **EXT-01**: `character/teleportrock/rest.go` — add no-op `SetToOneReferenceID`/`SetToManyReferenceIDs` stubs to `RestModel` per `libs/atlas-rest/CLAUDE.md`.
- **EXT-02**: `character/teleportrock` package — add an `httptest.NewServer`-backed test for `GetByCharacterId`/`requestByCharacterId` that serves a real fixture response and asserts a populated `Model`.
- **Multitenancy anti-pattern**: `teleport_rock/provider.go` / `administrator.go` — drop the explicit `tenantId` parameter and manual `tenant_id = ?` clause from `getByCharacterId`/`deleteByCharacterId` (keep it only where `saved_location`-style convention keeps it, i.e. the create half of `replaceList`), relying on `db.WithContext(ctx)` per `patterns-multitenancy-context.md`.

## Fixes applied

- **DOM-27**: `services/atlas-character/atlas.com/character/teleport_rock/resource.go:30,37` — replaced both bare `w.WriteHeader(http.StatusInternalServerError)` calls with `server.WriteErrorResponse(d.Logger())(w)(err)`, matching the sibling idiom in `saved_location/resource.go` (e.g. lines 50/57/82/89) and the transient-error classifier already registered in `main.go`.
- **EXT-01**: `services/atlas-channel/atlas.com/channel/character/teleportrock/rest.go` — added no-op `SetToOneReferenceID(_, _ string) error` and `SetToManyReferenceIDs(_ string, _ []string) error` stub methods on `RestModel`, matching the sibling signatures in `character/rest.go:76,80`.
- Verified: `go build ./...`, `go vet ./...`, and the package tests (`teleport_rock/...` in atlas-character, `character/teleportrock/...` in atlas-channel) all pass clean in both modules after the fixes.

### Non-Blocking (should fix) — Minor

- **DOM-01**: `teleport_rock/builder.go`'s `Build()` has no validation/error return. Pre-existing, fleet-wide characteristic of this service's builders (`saved_location`, `character` domains show the same shape) — not a regression introduced by this task, but still a documented-pattern gap worth closing service-wide.
- **DOM-02/DOM-03**: `teleport_rock/entity.go` has no `Make(Entity)`/`ToEntity()` — architecturally justified by the many-rows-to-one-Model aggregate shape (`modelFromEntities` fills the role), but the checklist's named symbols are absent.
- Test-coverage gap: no test explicitly creates a `teleport_rock` row, deletes the owning character, and asserts the row is gone — `character/processor.go:354`'s `teleport_rock.DeleteForCharacter` call is exercised transitively by `TestDeleteForSagaCompensation_Existing` but that test never seeds a teleport_rock row, so a regression in the cleanup call itself would not be caught.
