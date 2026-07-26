# Backend Audit — task-139-pet-auto-pot-validation

- **Service Path:** `libs/atlas-constants`, `libs/atlas-packet`, `services/atlas-data`, `services/atlas-pets`, `services/atlas-consumables`, `services/atlas-channel`, `services/atlas-configurations` (seed JSON), `tools/packet-audit`
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-07-26
- **Build:** PASS (evidence reused from `.superpowers/sdd/plan/task-14-report.md` per instructions — not re-run this pass)
- **Tests:** PASS, 0 failing (all 7 touched modules, `go test -race ./... && go vet ./... && go build ./...` clean per task-14 report; `docker buildx bake` clean; `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh` clean; `tools/lint.sh --check` clean for Go — only a pre-existing `ui:node-missing` environmental gap, not a Go regression)
- **Overall:** NEEDS-WORK

## Build & Test Results

Reused from `.superpowers/sdd/plan/task-14-report.md` (per instructions, not re-run this pass):
- `go test -race ./... && go vet ./... && go build ./...` — EXIT=0 for all 7 modules (`libs/atlas-constants`, `libs/atlas-packet`, `services/atlas-data`, `services/atlas-pets`, `services/atlas-consumables`, `services/atlas-channel`, `services/atlas-configurations`).
- `docker buildx bake --progress=plain atlas-data atlas-pets atlas-consumables atlas-channel atlas-configurations` — EXIT=0.
- `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh` — EXIT=0.
- `tools/lint.sh --check` — Go: 0 issues across 83/83 module runs; only failing target is `ui:node-missing` (no Node.js in the container, unrelated to this Go-only branch).
- `tools/template-opcode-order-guard.sh` — re-run this pass: `OK: 22 template arrays are in ascending opcode order.`

No code/test changes were made during this audit. This audit adds new findings (EXT-01, EXT-02, FILE-05, DOM-28) not present in the build/test gate — those gates check compilation and test-pass, not file-placement or resilience conventions.

## Domain Discovery

No package in the diff has a `model.go` (no full **domain** packages touched). Classified packages:

| Package | Classification | Notes |
|---|---|---|
| `services/atlas-channel/.../socket/handler` (pet_item_use.go et al.) | Support (socket packet handler) | Not a REST `resource.go`; governed by FILE-* only. |
| `services/atlas-channel/.../data/consumable` (new) | Support (REST client) | New package — full FILE-* + EXT-* checklist applies. |
| `services/atlas-channel/.../data/equipment` (new) | Support (REST client) | New package — full FILE-* + EXT-* checklist applies. |
| `services/atlas-pets/.../pet` | Domain (pre-existing, has model.go/builder.go/entity.go) | Only `SetSkill`/`SetSkillAndEmit` added to existing processor/administrator/producer. |
| `services/atlas-consumables/.../pet`, `.../consumable`, `.../cash` | Domain/support (pre-existing) | Extended, not restructured. |
| `libs/atlas-constants/pet/skill` (new) | Shared constants package | New; DOM-21 target. |
| `libs/atlas-packet/cash/serverbound`, `pet/serverbound`, `model` | Packet codec library | DOM-25 target. |
| `services/atlas-data/.../cash`, `.../equipment` (reader.go) | Support (XML→REST reader) | Pre-existing packages, extended. |

## FILE-* Checklist Results

### `services/atlas-channel/atlas.com/channel/data/consumable` (new package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor in processor.go | PASS | `data/consumable/processor.go:11-27` — interface + `NewProcessor` + `GetById` method, no processor code elsewhere. |
| FILE-02 | RestModel + Transform/Extract + JSON:API methods in rest.go | PARTIAL | `data/consumable/rest.go:11-31` has `RestModel`/`GetName`/`GetID`/`SetID`/`Extract` — present, but see FILE-05 (Model also lives here). |
| FILE-03 | Request funcs in requests.go | PASS | `data/consumable/requests.go:11-19` — `getBaseRequest`/`requestById`. |
| FILE-04 | n/a (no entity.go — no DB-backed entity in a REST-client package) | N/A | No local persistence. |
| **FILE-05** | Domain `Model` in `model.go`; RestModel+Extract in `rest.go` | **FAIL** | `data/consumable/rest.go:39-55` defines `type Model struct` and its accessors in `rest.go`, not a separate `model.go`. Sibling packages introduced the same way in this same service split them correctly: `services/atlas-channel/atlas.com/channel/data/item/model.go:1-13` (Model only) vs `data/item/rest.go` (RestModel/Extract only), and `data/map/model.go:1-32` vs `data/map/rest.go`. The new package collapses the two. |
| FILE-06 | No package-named catch-all file | PASS | No `consumable.go`; three purpose-named files. |

### `services/atlas-channel/atlas.com/channel/data/equipment` (new package)

Same pattern, same finding:

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor in processor.go | PASS | `data/equipment/processor.go:11-27`. |
| FILE-02 | RestModel + Extract + JSON:API methods in rest.go | PARTIAL | `data/equipment/rest.go:5-15` — present, but see FILE-05. |
| FILE-03 | Request funcs in requests.go | PASS | `data/equipment/requests.go:11-19`. |
| **FILE-05** | Domain Model in model.go | **FAIL** | `data/equipment/rest.go:26-33` defines `type Model struct` + accessors inline; no `model.go` file exists in this package. Same sibling-convention comparison as above (`data/map/model.go`, `data/item/model.go`). |
| FILE-06 | No catch-all file | PASS | Three purpose-named files. |

### `services/atlas-pets/atlas.com/pets/pet` (pre-existing domain package, extended)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | `SetSkill`/`SetSkillAndEmit` in processor.go | PASS | `pet/processor.go:8403-8446` (diff) — both methods and the interface addition live in `processor.go`. |
| FILE-05 | `updateFlag` write in administrator.go | PASS | `pet/administrator.go:8257-8273` (diff). |
| FILE-05 | `flagChangedEventProvider` in producer.go | PASS | `pet/producer.go:8569-8581` (diff). |
| Mock sync | `mock/processor.go` updated for new interface methods | PASS | `pet/mock/processor.go:8300-8342` (diff) — `SetSkillAndEmitFunc`/`SetSkillFunc` added with nil-check defaults. |

### `services/atlas-consumables` (pre-existing domain/support packages, extended)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01/05 | `ConsumePetSkillPouch` in `consumable/processor.go` | PASS | `consumable/processor.go:7448-7491` (diff) — colocated with the other `Consume*` item-consumer factory functions in the same file, matching existing convention (`ConsumePetFood`, `ConsumeCashPetFood`). |
| Mock sync | `pet/mock/processor.go` updated for `SetSkill` | PASS | `pet/mock/processor.go:7683-7688` (diff). |

## External HTTP Client Checklist (`requests.GetRequest[T]` packages)

Both new packages call `atlas-data` via `requests.GetRequest[RestModel]`, triggering the full EXT-* checklist.

### `services/atlas-channel/atlas.com/channel/data/consumable`

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| **EXT-01** | RestModel implements `SetToOneReferenceID`/`SetToManyReferenceIDs` | **FAIL** | `data/consumable/rest.go` — `RestModel` (lines 11-31) implements only `GetName()`/`GetID()`/`SetID()`. No relationship methods. Sibling `data/item/rest.go` and `data/map/rest.go` both implement these (`data/map/rest.go:75-81`). Per `libs/atlas-rest/CLAUDE.md` and the historical task-037 bug, api2go errors on unmarshal of any response carrying a `relationships` block without these methods — a real risk here since `atlas-data`'s cash/consumable resources are JSON:API and may legitimately carry relationships. |
| **EXT-02** | httptest-backed integration test | **FAIL** | No test file exists in `services/atlas-channel/atlas.com/channel/data/consumable/` (confirmed via `git diff --stat` over `*_test.go` for this branch — no `data/consumable` or `data/equipment` test file appears in the changed-file list). Sibling `data/item/processor_test.go:1-70` uses `httptest.NewServer` against a fixture response; this convention was not followed for the new package. |
| EXT-03 | 404 vs other failures distinguished | N/A | `GetById` (`data/consumable/processor.go:26-28`) is a bare pass-through of whatever error `requests.Provider` returns — it does not itself misclassify any error as "not found," so nothing to flag as a false 404 mapping. (The caller, `pet_item_use.go`, deliberately treats every fetch error as reject-worthy per the FR-1..3 fail-closed design — that is intentional, not a client-layer defect.) |
| EXT-04 | URL not hardcoded | PASS | `data/consumable/requests.go:15-17` — `requests.RootUrl("DATA")`. |

### `services/atlas-channel/atlas.com/channel/data/equipment`

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| **EXT-01** | Relationship interfaces | **FAIL** | `data/equipment/rest.go` — same gap as consumable: `RestModel` (lines 5-15) has no `SetToOneReferenceID`/`SetToManyReferenceIDs`. |
| **EXT-02** | httptest-backed integration test | **FAIL** | No test file in `services/atlas-channel/atlas.com/channel/data/equipment/`. |
| EXT-03 | 404 handling | N/A | Same pass-through reasoning as consumable. |
| EXT-04 | URL not hardcoded | PASS | `data/equipment/requests.go:15-17` — `requests.RootUrl("DATA")`. |

## DOM-* Checklist (targeted items per the task brief)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-21 | Shared atlas-constants used, not redeclared | PASS | `libs/atlas-constants/item/constants.go:87` adds `ClassificationPetSkill = Classification(519)` in the shared lib; `libs/atlas-constants/pet/skill/constants.go` is the new canonical skill-bit package, consumed via `petskill.Has`/`petskill.Apply`/`petskill.BitFor` everywhere (`pet/processor.go:8415,8424`, `pet_item_use.go:6191`, `model/asset.go:4647-4651`) — no service redeclares the bits. The equip-family exception is evidenced and intentional: `services/atlas-data/atlas.com/data/equipment/reader.go:4977-4980` (diff) has its own `petAbilityKeys` list with an explicit comment explaining the equip family spells `sweepForDrop` where the 0519 pouch family spells `dropSweep` — matches the task brief's documented exception exactly. |
| DOM-25 | Client wire values config-resolved, never hardcoded | PASS | `libs/atlas-packet/model/asset.go:4642-4658` (`resolvePetSkillWireMask`) resolves every bit through `atlas_packet.ResolveCode16(l, options, "petSkill", string(k))`; unconfigured bits are logged at Debug and omitted (`asset.go:4654` — "encoding as absent"), never guessed. `libs/atlas-packet/resolve.go:5041-5082` (`ResolveCode16`) is a new soft (miss-tolerant) resolver, distinct from the loud-default `ResolveCode`/`WithResolvedCode` used for mandatory mode bytes. Every routed version's seed template carries the `skillGate` option (`grep -c '"skillGate"' template_*.json` → 1 for all 9 routed versions: gms_48/61/72/79/83/84/87/95, jms_185; 0 for the two non-matrix templates gms_12/gms_92, which also have no `PetItemUseHandle` entry at all — confirmed via `grep -n PetItemUseHandle template_gms_12_1.json template_gms_92_1.json` returning no matches, so there is no fail-open gap there). The socket handler itself fails closed on an unconfigured/unknown gate value (`pet_item_use.go:6028-6032`, reason `skill_gate_unconfigured`). |
| DOM-26 | No bare `go` statements | PASS | `grep -nE '^\s*go (func\|[A-Za-z_])'` over every non-test `.go` file changed in this branch returned zero matches. The only concurrency introduced (`model.NewGroup`/`model.Submit` in `pet_item_use.go:6046-6076`) is a pre-existing `libs/atlas-model/model/parallel_group.go` helper (not touched by this diff) built on `errgroup.Group.Go`, not a bare `go` statement in application code. |
| Concurrency / data races | `model.Submit` closures write disjoint variables | PASS | Each of the four goroutines in `pet_item_use.go:6046-6076` writes to a mutually-exclusive variable pair (`pm/pmErr` XOR `spawnedPets/spawnedErr` depending on `hasPetId`; `c/cErr`; `ci/ciErr`) — no variable is written by more than one goroutine, and all reads happen after `pg.Wait()` (line 6076), which is the happens-before barrier. `go test -race` was clean for `services/atlas-channel` per the task-14 report. |
| Fail-closed behavior | Every fetch failure / unconfigured gate / oversized petId rejects | PASS | `p.PetId() > math.MaxUint32` → reject (`pet_item_use.go:6022-6025`); unconfigured/unknown `skillGate` → reject (`:6028-6032`); `pmErr`/`spawnedErr`/`ciErr` all reject before consuming (`:6079-6104`); `evaluateAutoPot` rejects on not-owned/not-spawned/dead/no-recovery/no-matching-skill-source (`:6162-6182`). No path both rejects and forwards — the single `RequestItemConsume` call is the last statement in the function (`:6120`), reached only after every guard passes. |
| Cross-service Kafka contract | `SET_SKILL`/`FLAG_CHANGED`/`RequestItemConsumeBody.PetId` agree name+type across producer/consumer | PASS | `SET_SKILL`: consumables emits `Command[SetSkillCommandBody]{Skill string, Enabled bool}` with `PetId uint32` (`atlas-consumables/kafka/message/pet/kafka.go:7620-7635`, this diff fixes a prior `uint64`→`uint32` mismatch against atlas-pets' `PetId uint32`, `atlas-pets/kafka/message/pet/kafka.go`); atlas-pets consumes it identically (`kafka/consumer/pet/consumer.go:8101-8111`). `FLAG_CHANGED`: `pet/producer.go:8569-8581` emits `{Slot int8, Flag uint16}`; atlas-channel's consumer expects the same shape (`kafka/message/pet/kafka.go:5817-5820`) and `kafka/consumer/pet/consumer.go:5737-5752` handles it, gated on `pet2.StatusEventTypeFlagChanged`/tenant match. `RequestItemConsumeBody.PetId uint64` matches on both the atlas-channel producer side (`kafka/message/consumable/kafka.go:5768`) and the atlas-consumables consumer side (`kafka/message/consumable/kafka.go:7572`), and the consumer call site was updated to pass it through (`kafka/consumer/consumable/consumer.go:7546`). |
| **DOM-28** | No silent degradation in fetch-and-degrade paths | **FAIL** | `services/atlas-channel/atlas.com/channel/socket/handler/pet_item_use.go:233` (`compartment.NewProcessor(...).GetByType` error) and `:245-248` (`ep.GetById(a.TemplateId())` error inside the per-worn-item loop) both swallow the fetch error with no `l.WithError(err)` log and no metric — the per-item branch (`:245-248`) is the sharper case: if one of two worn items fails to resolve and the other succeeds, `sawData` becomes `true` from the surviving item and the whole failure is invisible (no log line at all, at any level, for the specific failed template id). The codebase has an established pattern for exactly this situation (`libs/atlas-rest/degrade/degrade.go`, already used in this same service family at `services/atlas-consumables/atlas.com/consumables/consumable/processor.go`), but it is not applied here. The aggregate case (`len(worn)==0` or `sawData==false` end-to-end) does eventually surface via `reject("equip_data_missing")`'s Warn log at the call site (`:6111`, i.e. `resolveSkillSources`'s `ok=false` return), so total data unavailability is not silent — only partial, per-item degradation is. |

## Summary

### Blocking (must fix)

- **EXT-01**: `services/atlas-channel/atlas.com/channel/data/consumable/rest.go` and `services/atlas-channel/atlas.com/channel/data/equipment/rest.go` — `RestModel` missing `SetToOneReferenceID`/`SetToManyReferenceIDs`. Add the same no-op pair every sibling `data/*` package has (e.g. `data/map/rest.go:75-81`).
- **EXT-02**: Both new packages have zero test coverage of the actual HTTP round-trip (only pure-function tests exist elsewhere in the branch). Add an `httptest.NewServer`-backed test per package, mirroring `data/item/processor_test.go`.
- **FILE-05**: Both new packages define their domain `Model` inside `rest.go` instead of a separate `model.go`, diverging from the `data/item` and `data/map` convention introduced in the same service. Split `Model`/accessors into `model.go`, leaving `rest.go` to `RestModel`/`Extract`/JSON:API methods only.
- **DOM-28**: `pet_item_use.go`'s `resolveSkillSources` silently drops per-equip and per-compartment fetch failures (lines 233, 245-248) with no log and no metric, while the codebase's established `degrade` pattern (already in use in `atlas-consumables/consumable/processor.go`) goes unused here. Log at Warn (or wire through `degrade.Observe`) on both failure branches.

### Non-Blocking (should fix)

- None identified beyond the above — the DOM-25 config-resolution work, the cross-service Kafka contract, the fail-closed validation pipeline, the concurrency pattern, and the atlas-constants usage are all solidly evidenced and pass.
