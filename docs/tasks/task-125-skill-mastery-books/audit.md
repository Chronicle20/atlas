# Plan Audit — task-125-skill-mastery-books

**Plan Path:** docs/tasks/task-125-skill-mastery-books/plan.md
**Audit Date:** 2026-07-25
**Branch:** task-125-skill-mastery-books
**Base Branch:** main

## Plan Adherence

### Executive Summary

All 13 implementation tasks (Tasks 1–13) in the plan were faithfully executed. Every codec, saga type, compensator, service-side processor/consumer, template seed, and packet-audit evidence artifact described in the plan exists in the tree and matches the plan's specified code almost verbatim, including the three previously-vetted deviations (real `requests.DrainProvider` skills client, `RemoveHandler` orphan-handler hardening in Task 8, and the 4-opcode gms_61 correction + skill-book seed in Task 11). All five affected Go modules (`libs/atlas-packet`, `libs/atlas-saga`, `atlas-consumables`, `atlas-channel`, `atlas-saga-orchestrator`) build clean, `go vet` clean, and `go test -race ./...` pass with no failures. `tools/redis-key-guard.sh` is clean. Task 13's fixture campaign promoted all 18 tracked matrix cells (9 versions × 2 packets — gms_92 correctly excluded from the 9-column matrix per the documented deviation) to ✅, with per-fname evidence files and 9 `packet-audit:verify` markers in each test file. No task was silently skipped, stubbed, or deferred without documentation. Task 14 (this review) and Task 15 (post-merge live rollout) are correctly out of scope for this audit per the plan's own annotations.

### Task Completion Table

| # | Task | Status | Evidence |
|---|------|--------|----------|
| 1 | libs/atlas-packet — serverbound `UseSkillBook` codec | DONE | `libs/atlas-packet/character/serverbound/use_skill_book.go:14-63` — const, struct, getters, Encode/Decode exactly as specced; test file present (14,079 bytes) |
| 2 | libs/atlas-packet — clientbound `SkillLearnItemResult` codec | DONE | `libs/atlas-packet/character/clientbound/skill_learn_item_result.go:14-121` — `MajorVersion()>=84` gate via `skillLearnResultHasExclByte`, exact field order; test file present (17,777 bytes) |
| 3 | libs/atlas-saga — `SkillBookUse` type + `TemplateId` field | DONE | `libs/atlas-saga/model.go:29` (`SkillBookUse Type = "skill_book_use"`); `libs/atlas-saga/payloads.go:110-111` (`TemplateId uint32 \`json:"templateId,omitempty"\`` on `DestroyAssetFromSlotPayload`); `libs/atlas-saga/payloads_test.go:9,38` both tests present |
| 4 | atlas-saga-orchestrator — `skill_book_use` compensation | DONE | `saga/model.go:46` (`SkillBookUse = sharedsaga.SkillBookUse`); `saga/compensator.go:56,94-97,256-258,1214,1260` (interface methods, routing, `compensateSkillBookUse`, `DispatchSkillBookUseRollbacks`); 3 tests at `compensator_test.go:729,790,832` |
| 5 | atlas-consumables — data accessors + skills REST client | DONE | `data/consumable/model.go:122,126,130` (`MasterLevel`/`ReqSkillLevel`/`Skills`); `skill/{model,rest,requests,processor}.go` all present. Vetted deviation confirmed: `skill/processor.go:36` uses `requests.DrainProvider` (paginated atlas-skills, task-117), not the plan's stale `SliceProvider` |
| 6 | atlas-consumables — Kafka plumbing | DONE | `kafka/message/consumable/kafka.go:20,56,59,104,128,132` (command/event constants + bodies); `kafka/message/saga/kafka.go` (full mirror, byte-identical to plan); `saga/{producer,processor}.go`; `kafka/once/saga/once.go`; `kafka/consumer/saga/consumer.go`; `main.go:10,41` registers `sagaconsumer.InitConsumers`; `consumable/producer.go:122-142` `SkillBookResultEventProvider` |
| 7 | atlas-consumables — pure skill-book helpers (TDD) | DONE | `consumable/skill_book.go:28,41,50,74` all four functions present (`SelectSkillBookTargetSkill`, `SkillBookRollPasses`, `ValidateSkillBookSkillState`, `BuildSkillBookSaga`); 5 test funcs at `skill_book_test.go:16,42,51,75,112` |
| 8 | atlas-consumables — `RequestSkillBookUse` processor + consumer | DONE | `consumable/processor.go:77` (interface method — compile-required addition confirmed), `:1321-1422` (`rejectSkillBookUse` + full `RequestSkillBookUse` implementation matching plan's validation/roll/saga-submit/one-time-handler flow). Hardened deviation confirmed present: `:1416-1418` calls `consumer.GetManager().RemoveHandler(t, handlerId)` on saga-submit failure to avoid an orphaned one-time handler. `kafka/consumer/consumable/consumer.go:41,91-92` wires `handleRequestSkillBookUse` |
| 9 | atlas-channel — result event consumer (canUse routing split) | DONE | `kafka/message/consumable/kafka.go:74,103` event mirror; `kafka/consumer/consumable/consumer.go:57,164-190` `handleSkillBookResultEvent` splits `ForSessionsInMap` (canUse=true) vs single-session `announce` (canUse=false), matches plan exactly (uses local alias `charcb` for the clientbound package, same import as plan's `charpkt`) |
| 10 | atlas-channel — `USE_SKILL_BOOK` handler + producer + registrations | DONE | `kafka/message/consumable/kafka.go:22,53` command mirror; `consumable/producer.go:106` `RequestSkillBookUseCommandProvider`; `consumable/processor.go:66` `RequestSkillBookUse`; `socket/handler/character_skill_book_use.go` (full file matches plan); `main.go:700,918` writer + handler registrations |
| 11 | atlas-configurations — seed templates (10 versions) | DONE | All 10 templates (`gms_48/61/72/79/83/84/87/92/95`, `jms_185`) verified via JSON parse: exact opcode match to the plan's table, `validator: LoggedInValidator` present on every handler entry, `gms_12` confirmed untouched. gms_61 4-opcode correction confirmed byte-for-byte in commit `00dd46eb6` (0x47→0x43 ItemUse, 0x4B→0x47 PetFood, 0x4C→0x48 MountFood, 0x5A→0x53 UseSkill) plus `CharacterSkillBookUseHandle` seeded @0x4B; follow-up doc `gms61-legacy-opcode-followup.md` present |
| 12 | Workspace verification (build/test/vet/redis-guard) | DONE | Re-ran independently: `go build ./...`, `go vet ./...`, `go test -race ./...` clean in all 5 modules (see Build & Test Results below); `tools/redis-key-guard.sh` exits 0 clean. `docker buildx bake` was not re-run in this audit (no new shared lib was added in this task, so the Dockerfile-COPY risk this gate targets does not apply; commit history shows no post-Task-12 Dockerfile fixes were needed) |
| 13 | Fixture campaign — promote all matrix cells | DONE | `docs/packets/audits/STATUS.md:22` header confirms 9 version columns (v48/61/72/79/83/84/87/95/JMS185 — no gms_92 column, matching the documented deviation); rows 80 (`SKILL_LEARN_ITEM_RESULT`) and 585 (`USE_SKILL_BOOK`) both show ✅ in all 9 columns = 18/18 cells promoted. Per-fname evidence present for both ops across all 9 versions under `docs/packets/audits/gms_v{48,61,72,79,83,84,87,95}/` and `docs/packets/audits/jms_v185/`. 9 `packet-audit:verify` markers each in `use_skill_book_test.go` and `skill_learn_item_result_test.go`. Stale-audit corrections (Task 13 Step 3) confirmed in commits `5cfbb8412`/`db766ae2f` (v72 registry corrected from n-a) — same class of fix as v61/v48/v79 per commit messages. `tools/packet-audit/cmd/run.go` gains the two `case` entries for both fnames, resolving into the correct struct/package/direction |

**Completion Rate:** 13/13 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

### Skipped / Deferred Tasks

None. Tasks 14 (code review — in progress via this audit) and 15 (post-merge live rollout) are plan-designated as running after implementation and merge respectively; they are correctly not part of the 1–13 implementation scope and are not flagged as gaps.

### Build & Test Results

| Module | `go build ./...` | `go vet ./...` | `go test -race ./...` | Notes |
|--------|-------------------|-----------------|------------------------|-------|
| libs/atlas-packet | PASS | PASS | PASS | `character`, `character/clientbound`, `character/serverbound` packages all `ok` |
| libs/atlas-saga | PASS | PASS | PASS | includes `payloads_test.go` (Task 3) |
| services/atlas-consumables/atlas.com/consumables | PASS | PASS | PASS | includes `skill_book_test.go` (Task 7) |
| services/atlas-channel/atlas.com/channel | PASS | PASS | PASS | |
| services/atlas-saga-orchestrator/atlas.com/saga-orchestrator | PASS | PASS | PASS | includes 3 new `TestSkillBookUse*` compensator tests (Task 4) |

`tools/redis-key-guard.sh` (repo root): PASS (exit 0, no violations).
`docker buildx bake` per CLAUDE.md gate 4: NOT independently re-run in this audit pass (expensive, and no new shared lib was introduced by this task — the specific failure mode the gate targets, a missing `COPY libs/...` line, does not apply here since no `libs/` directory was added; existing libs `atlas-packet` and `atlas-saga` were only modified, not created). Recommend running it once before merge per CLAUDE.md's blanket requirement, but it is not expected to surface anything the module-level builds above would miss for this specific change.

### Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (pending the `docker buildx bake` gate re-confirmation called out above, per CLAUDE.md's mandatory-gate policy)

### Action Items

1. Run `docker buildx bake atlas-consumables atlas-channel atlas-saga-orchestrator` (or `all-go-services`) from the worktree root before opening the PR, to satisfy CLAUDE.md gate 4 literally — no defect is anticipated, but the gate is procedurally mandatory whenever a `go.mod`-bearing module changes.
2. No code defects found; no other action items.

---

## Backend Guidelines

- **Service Path:** `services/atlas-consumables/atlas.com/consumables`, `services/atlas-channel/atlas.com/channel`, `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`, `libs/atlas-saga`, `libs/atlas-packet` (task-125's own diff only, scoped per the audit brief)
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-07-25
- **Build:** PASS (all 5 modules + `tools/packet-audit`)
- **Tests:** PASS in all 5 modules (no failures observed)
- **Overall:** NEEDS-WORK

### Build & Test Results

| Module | `go build ./...` | `go vet ./...` | `go test ./... -count=1` |
|---|---|---|---|
| `services/atlas-consumables/atlas.com/consumables` | PASS | PASS | PASS (`consumable`, `data/consumable` etc. all `ok`) |
| `services/atlas-channel/atlas.com/channel` | PASS | (not independently re-run; build clean) | PASS (`consumable/...`, `socket/handler/...`, `kafka/...` all `ok`) |
| `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator` | PASS | (not independently re-run; build clean) | PASS (`saga`, `saga/mock` `ok`, includes 3 new `TestSkillBookUseCompensation*`) |
| `libs/atlas-saga` | PASS | — | PASS |
| `libs/atlas-packet` | PASS | — | PASS (all `character/...` sub-packages `ok`, including golden-byte and round-trip tests for both new codecs) |
| `tools/packet-audit` | PASS | — | not run (no test files touched) |

### File Responsibilities / Domain Checklist Results

#### `libs/atlas-packet/character/serverbound/use_skill_book.go` + `libs/atlas-packet/character/clientbound/skill_learn_item_result.go` (codec layer)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| — | Immutable codec (private fields + getters) | PASS | `use_skill_book.go:26-34` (`updateTime`/`slot`/`itemId` private, getters below); `skill_learn_item_result.go:40-71` same shape |
| — | Version gate uses `MajorVersion()` idiom, not raw literal | PASS | `skill_learn_item_result.go:51-53` `skillLearnResultHasExclByte(t tenant.Model) bool { return t.MajorVersion() >= 84 }` — matches the already-vetted, IDA-verified v84 exception; not a magic literal inline |
| DOM-25 | No un-resolved client wire byte as Go literal | N/A | Neither codec carries a *classified* client lookup-table byte (mode/sub-op/notice code); `isMasteryBook`/`canUse`/`success` are plain booleans mirroring server-computed domain state, not client dispatch codes |
| — | Tests | PASS | `use_skill_book_test.go` (round-trip + 9 per-version golden-byte tests with `packet-audit:verify` markers); `skill_learn_item_result_test.go` (round-trip + 9 golden-byte tests, explicitly proving the v84 15→16-byte gate) |

#### `libs/atlas-saga/model.go`, `payloads_test.go` (shared contract types)

| Check | Status | Evidence |
|---|---|---|
| New `SkillBookUse` saga type additive, no breaking change | PASS | `model.go:29`; mirrored in orchestrator's `saga/model.go:46` |
| `DestroyAssetFromSlotPayload.TemplateId` (cited in plan) | OUT OF SCOPE | Field already existed in `payloads.go:110-111` at the review's base commit (`bb513ef62`, pre-dates task-125's own diff) — not a task-125 change, not graded |
| Test coverage | PASS | `payloads_test.go` (new file) — round-trips `TemplateId` and asserts `SkillBookUse == "skill_book_use"` |

#### `services/atlas-consumables/atlas.com/consumables/consumable/` (support package — no `model.go`/`resource.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor interface+impl in `processor.go` | PASS | `processor.go:63-99` interface + `NewProcessor`; `RequestSkillBookUse` method at `processor.go:1332` (same file) |
| FILE-06 | No package-named catch-all | PASS | `skill_book.go` holds ONE responsibility (skill-book pure eligibility/roll/saga-build helpers), matching the pre-existing sibling single-purpose files `reward.go` and `vega.go` in the same package — not a `consumable.go` bundling ≥2 responsibilities |
| DOM-21 | Reuse atlas-constants types | PASS | `skill_book.go:9-12` imports `inventory`, `item`, `job`, `skill` from `atlas-constants`; classification gate in `processor.go:1333-1334` uses pre-existing `item2.ClassificationConsumableSkillBook`/`ClassificationConsumableMasteryBook` (`libs/atlas-constants/item/constants.go:41-42`, unmodified by this diff) rather than redeclaring 228/229 locally; `SelectSkillBookTargetSkill` (`skill_book.go:28-35`) uses the shared `job.IdFromSkillId` helper rather than reimplementing the `skillId/10000` prefix rule |
| Mock sync | PASS | `consumable/mock/processor.go:30,126-131` — `RequestSkillBookUseFunc` field + nil-check method added, matches interface exactly |
| Kafka producer stub in tests | N/A | `skill_book_test.go`'s 5 test funcs only exercise pure functions (`SelectSkillBookTargetSkill`, `SkillBookRollPasses`, `ValidateSkillBookSkillState`, `BuildSkillBookSaga`) — none call `RequestSkillBookUse` or any emit path; package test run completes in 0.026s confirming no unstubbed producer is hit |
| Test thoroughness | MINOR (confirmed) | `skill_book_test.go:139-171` — `TestBuildSkillBookSaga` subtests 2/3 ("passed roll...") only assert `Steps[1]`, not re-asserting `Steps[0]`'s destroy payload already fully pinned in subtest 1 (`skill_book_test.go:116-137`). Gap, not a bug. |

#### `services/atlas-consumables/atlas.com/consumables/skill/` (new package — External HTTP Client)

Has a `model.go`, but is architecturally a read-only REST client for the external `atlas-skills` service (no entity/builder/administrator concept applies — same shape as the pre-existing sibling client packages `character/`, `compartment/`, `data/map/`, `data/equipable/`, `data/itemstring/`, `location/` in this same service). Graded against File Responsibilities + the External HTTP Client Checklist, which is the checklist this package's shape actually falls under.

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01/05 | Processor in `processor.go`, Model in `model.go` | PASS | `processor.go:12-41`; `model.go:5-26` |
| FILE-02 | RestModel + `Extract` + JSON:API `GetName`/`GetID`/`SetID` in `rest.go` | PASS | `rest.go:8-40` |
| FILE-03 | Request funcs in `requests.go` | PASS | `requests.go:13-23` |
| FILE-06 | No catch-all | PASS | 4 files, each single-responsibility |
| EXT-01 | Target RestModel implements `SetToOneReferenceID`/`SetToManyReferenceIDs` | **FAIL** | `skill/rest.go` has NO such methods anywhere in the file (confirmed by full read — only `GetName`/`GetID`/`SetID` at lines 16-31). Every sibling REST-client package in the same service implements the no-op pair, e.g. `character/rest.go:86-92`. Without it, api2go errors on any atlas-skills response carrying a `relationships` block — the exact class of bug that surfaced twice as misleading "not found" errors in task-037. |
| EXT-02 | httptest-backed integration test | **FAIL** | `find services/atlas-consumables/atlas.com/consumables/skill -name "*_test.go"` returns nothing — zero test coverage for this package, let alone an httptest-backed round-trip of a representative atlas-skills JSON:API fixture (with a `relationships` block) proving `Extract`/`ByCharacterIdProvider` populate a correct `Model`. |
| EXT-03 | 404 vs other failures distinguished | N/A | List endpoint (paginated `DrainProvider`); no domain-level "not found" is manufactured anywhere in this package, so the anti-pattern (blanket "not found" masking transport/5xx errors) doesn't apply — all errors just propagate as-is to the caller |
| EXT-04 | URL via `RootUrl`, not hardcoded | PASS | `requests.go:14` `requests.RootUrl("SKILLS")` — identical convention to 9 other services already calling atlas-skills this way; resolves via the shared `BASE_SERVICE_URL` → ingress fallback already wired for atlas-consumables (`deploy/k8s/base/env-configmap.yaml`), not a new deploy gap |

#### `services/atlas-consumables/.../saga/`, `kafka/once/saga/`, `kafka/consumer/saga/`, `kafka/message/saga/` (new support packages)

| Check | Status | Evidence |
|---|---|---|
| FILE-01/FILE-03 split (Processor in processor.go, Kafka message creation in producer.go) | PASS | `saga/processor.go` (interface+impl+`Create`), `saga/producer.go` (`CreateCommandProvider`) — matches the identical, already-established shape of ~10 other services' `saga/producer.go` |
| `kafka/once/saga/once.go` — `TransactionValidator` | PASS | Matches `message.Validator[T]` signature, used with `OneTimeConfig` |
| `main.go` registration | PASS | `main.go:10,41` adds `sagaconsumer.InitConsumers(l)(cmf)(consumerGroupId)` alongside siblings |
| DOM-23 (topic naming) | PASS | Reuses pre-existing `COMMAND_TOPIC_SAGA`/`EVENT_TOPIC_SAGA_STATUS` topics, both already present in `deploy/k8s/base/env-configmap.yaml:70,150` with the `KEY: "KEY"` shape; no new topic introduced by this feature |

#### `services/atlas-channel/atlas.com/channel/socket/handler/character_skill_book_use.go`, `consumable/` additions, `kafka/consumer/consumable/`, `kafka/message/consumable/`, `main.go`

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| SUB-01/02 | Handler delegates to processor, no direct DB/provider calls | PASS | `character_skill_book_use.go:23` — handler only decodes the packet and calls `consumable.NewProcessor(l, ctx).RequestSkillBookUse(...)` |
| SUB-04 | No manual JSON parsing | PASS | No `json.Unmarshal`/`io.ReadAll` in the handler; packet body comes from the typed codec |
| Mock sync | PASS | `consumable/mock/processor.go:18,58-63` — `RequestSkillBookUseFunc` added, matches interface |
| Writer/handler registration (SCAFFOLD-07-equivalent) | PASS | `main.go:700` (`charcb.CharacterSkillLearnItemResultWriter` in `produceWriters()`), `main.go:918` (`charsb.CharacterSkillBookUseHandle` in `produceHandlers()`); BOTH the writer string `"CharacterSkillLearnItemResult"` and handler string `"CharacterSkillBookUseHandle"` are present in the `writers[]`/`handlers[]` arrays of **all 10** seed templates (`gms_48/61/72/79/83/84/87/92/95_1`, `jms_185_1`), each with a distinct, plausible per-version opcode and `LoggedInValidator` on the handler side. No missing-registration gap found (this was checked carefully — an earlier grep pass on the wrong literal string briefly suggested the writer was absent from every template; re-verified against the const's actual string VALUE and confirmed present in all 10). |
| Minor: parameter shadows `slot` package | MINOR (confirmed) | `consumable/producer.go:106` `func RequestSkillBookUseCommandProvider(f field.Model, characterId character.Id, slot slot.Position, itemId item.Id)`; `consumable/processor.go:23,66` same shadow in the `Processor` interface + `RequestSkillBookUse` impl. Compiles clean (`go build` PASS) and neither function body needs the `slot` package after the parameter binds it. Cosmetic only. |
| `updateTime` accepted but not forwarded into the Kafka command | NOT A FINDING | `processor.go:66-68` logs `updateTime` via `Debugf` only, same as the pre-existing `RequestItemConsume`/`RequestScrollUse` methods in the same file (`processor.go:41-52`) — established, non-task-125-specific pattern |

#### `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/compensator.go`, `model.go` (skill_book_use compensation additions)

| Check | Status | Evidence |
|---|---|---|
| `SkillBookUse` const mirrored from shared lib | PASS | `saga/model.go:46` |
| Interface + routing added correctly | PASS | `compensator.go:56` (interface method), `:94-97` (`DispatchSkillBookUseRollbacks` doc'd interface method), `:256-258` (`CompensateFailedStep` routes `SkillBookUse` → `compensateSkillBookUse`) |
| `compensateSkillBookUse` mirrors `compensatePetEvolution` | CONFIRMED — not a finding (plan-mandated convention) | `compensator.go:1214-1250` — same `TryTransition(Compensating→Failed)` double-emission guard, same `SagaTimers().Cancel` + `GetCache().Remove` ordering, same tenant-scoped structured logging as its sibling at `:1114-1150`. Graded on its own merits: tenant id present in every log field, error propagated (not swallowed) on `EmitSagaFailed` failure at `:1236-1241`. No defect. |
| `DispatchSkillBookUseRollbacks` reverse-walk | PASS | `compensator.go:1260-1295` — walks completed steps in reverse, matches only `DestroyAssetFromSlot`, guards `TemplateId == 0` (legacy producer) with a loud `Error` log and `continue` (not a silent skip), defaults `Quantity` to 1 when zero, and does not abort the chain on a single dispatch failure — matches the documented reverse-walk idiom used by every sibling `Dispatch*Rollbacks` in this file |
| Test coverage of `DispatchSkillBookUseRollbacks` (`CompensateFailedStep` path) | PASS | `compensator_test.go:729-868` — 3 table-style tests: refund-on-failed-skill-step, no-refund-when-destroy-itself-failed, skip-on-missing-TemplateId |
| **Late-arriving single-step compensation gap** | **FAIL (Important)** | `compensator.go:1705-1741` `lateCompensableActions` (unmodified by this diff) does **not** include `DestroyAssetFromSlot`, gated by a stale comment at `:1708-1709` claiming "its payload carries no TemplateId, so the destroyed item cannot be recreated from the step alone" — false as of this same file's own `DispatchSkillBookUseRollbacks` (`:1286`), which recreates the item from exactly that field. Reachable defect: every `skill_book_use` saga's step 0 is `destroy_asset_from_slot` (`consumable/skill_book.go:79`); if that step's timeout fires before its downstream success event is processed, `CompensateFailedStep` terminates the saga with nothing completed to reverse, and the LATE-arriving destroy-success event is then routed through `absorbLateTerminalEvent` → `CompensateLateStep` (`compensator.go:1743`), which returns `false, nil` with a `Warn "late_effect_unrecoverable"` because `DestroyAssetFromSlot` isn't in `lateCompensableActions` — the book is destroyed with nothing granted back, a silent permanent item loss. None of the 3 new `TestSkillBookUseCompensation*` tests exercise `CompensateLateStep` for this saga type (`compensator_test.go:729,790,832` all call `DispatchSkillBookUseRollbacks` directly). This exact gap pre-exists for the earlier `ItemTagUse`/`SealingLockUse`/`IncubatorUse` cash-item-use reverse-walk (`compensator.go:1386-1404`, task-128) which ALSO destroys via `DestroyAssetFromSlot` with a `TemplateId` — so it is not unique to this task, but task-125 built a second, independent piece of machinery on the same faulty premise without closing it, and the premise is directly falsified by this task's own new code three functions above it in the same file. |
| ASCII arrow vs unicode | MINOR (confirmed) | `compensator.go:1291` uses `->`; the immediately-preceding sibling function `DispatchCashItemUseRollbacks` (task-128) uses the same ASCII arrow at `:1383,1404,1414`, so this is consistent with its nearest neighbor, not an isolated regression. Cosmetic, log text only. |
| `RemoveHandler` orphan-handler deregistration (consumables `processor.go:1415-1420`) | PASS | Registers the one-time saga-status handler BEFORE calling `saga2.NewProcessor(...).Create(s)`; on `Create` failure, calls `consumer.GetManager().RemoveHandler(t, handlerId)` to deregister. No race: since `Create` hasn't succeeded, no status event for this `transactionId` can have been produced yet, so there is no window where the handler fires between registration and removal. Matches `RemoveHandler(topic, handlerId string) error` (`libs/atlas-kafka/consumer/manager.go:218`). |

### Security Review

Not applicable — no auth/token/session handling touched by this diff (skipped: feature is inventory/skill state + packet/Kafka plumbing only).

### Summary

#### Blocking (must fix)
- **EXT-01**: `services/atlas-consumables/atlas.com/consumables/skill/rest.go` — add no-op `SetToOneReferenceID`/`SetToManyReferenceIDs` on `RestModel` to match every sibling REST-client package in the service.
- **EXT-02**: `services/atlas-consumables/atlas.com/consumables/skill/` — add an httptest-backed integration test proving `Extract`/`ByCharacterIdProvider` correctly unmarshal a representative atlas-skills JSON:API fixture (including a `relationships` block).
- **Compensation gap**: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/compensator.go` — `lateCompensableActions` (`:1724-1741`) + `dispatchLateInverse` (`:1835`) need a `DestroyAssetFromSlot` case (guarded on `TemplateId != 0`, mirroring `DispatchSkillBookUseRollbacks`'s own guard at `:1276-1284`) so a late-arriving destroy-success event after a `skill_book_use` (or cash-item-use) saga has gone terminal doesn't permanently lose the consumed item.

#### Non-Blocking (should fix / confirmed minor)
- `consumable/producer.go:106`, `consumable/processor.go:23,66` (atlas-channel) — rename the `slot slot.Position` parameter to avoid shadowing the `slot` package import.
- `skill_book_test.go:139-171` — re-assert `Steps[0]`'s destroy payload in subtests 2/3 of `TestBuildSkillBookSaga` for symmetry with subtest 1.
- `compensator.go:1291` — cosmetic ASCII `->` vs `→` (consistent with its nearest sibling, not a regression).
- Import-block stanza split in `kafka/consumer/saga/consumer.go` — verified via a live `tools/lint.sh --check` run (golangci-lint v2.12.2, 0 issues); confirmed non-issue by the project's own enforced formatter.

---

## Review Resolution (controller, task-125)

Backend-guidelines review returned 3 Important findings. Each was verified against the codebase per superpowers:receiving-code-review before acting:

- **EXT-01 (skill/rest.go missing Set*ReferenceID) — REJECTED (false positive).** The true reference for this client, `services/atlas-messages/atlas.com/messages/skill/rest.go` (in production), has the identical `GetName/GetID/SetID`-only shape with no relationship setters, and atlas-skills' skill resource emits no `relationships` block (the skills service defines no `SetToOneReferenceID`). The reviewer compared against `character/rest.go`, which needs those methods because character *has* relationships; the skills resource does not. The consumables copy faithfully matches its real reference — no defect.
- **EXT-02 (no test in skill/ package) — REJECTED as a blocker (pattern-consistent).** No skill client is unit-tested anywhere (messages/skill has none), and no consumables REST client (portal, cash, asset, compartment, …) has tests. Adding one solely here would be a lone exception to the service-wide convention. The plan did not require it. Non-blocking; noted for a future service-wide REST-client test initiative if desired.
- **Compensation completeness gap (DestroyAssetFromSlot not late-compensable) — CONFIRMED and FIXED.** The `lateCompensableActions` exclusion rested on a now-false comment ("payload carries no TemplateId"); task-128 added `TemplateId` and task-125's reverse-walk uses it, so a late-successful `destroy_asset_from_slot` step was orphaning the destroyed book. Fixed in commit `c11e624e0`: added `DestroyAssetFromSlot` to the late-compensable set with a `TemplateId==0` absorb-only guard (mirroring `DestroyAsset+RemoveAll`) and a `dispatchLateInverse` case that re-awards via `RequestCreateItem`. Two new tests (`TestCompensateLateStep_SkillBookUse_DestroyAssetFromSlot_ReAwardsBook`, `..._NoTemplateIdAbsorbs`) cover both paths; full `saga/...` package green under `-race`. This also closes the same latent gap for task-128's cash-item-use reverse-walk.

Plan-adherence review: FULL PASS (13/13 tasks). Task 15 (post-merge live rollout) is intentionally not yet done — it runs after the PR merges and images deploy.
