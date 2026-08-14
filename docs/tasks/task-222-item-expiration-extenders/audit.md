# Plan Audit — task-222-item-expiration-extenders

**Plan Path:** docs/tasks/task-222-item-expiration-extenders/plan.md
**Audit Date:** 2026-08-13
**Branch:** task-222-item-expiration-extenders
**Base/merge-base:** 6496b9c87f3b5e8a4e602d108c9c3e0327f943eb
**HEAD:** eebe997e6

## Executive Summary

All 15 plan tasks were faithfully implemented; every code artifact described in plan.md is present at the file:line locations the plan specified, and the 20 commits on the branch map cleanly one-to-one onto the plan's tasks. Task 8 deliberately implements REJECT-over-cap rather than the CLAMP that plan.md's Task-8 prose/snippet describe — this is the authorized D1 resolution to plan.md's internal self-contradiction (Global Constraints line 15 vs. Task 8 body), confirmed by the reject-returning code, the `errors.New("requested expiration exceeds the extender's server-derived cap")` return, and the test named `TestExtendAssetExpirationRejectsOverCap`. All eight silent-failure saga registration sites named in design §9 / context §4 are present and correctly wired. Task 15's live-client verification step (Step 7) was not performed — no running server/client in this environment — and is explicitly flagged rather than silently skipped, which is the authorized second divergence. Build/test/guard/bake verification was already run and confirmed green prior to this audit (per the task brief) and was not re-run here.

One documentation-only gap: none of plan.md's 111 checkbox items (`- [ ]`) were ever checked to `- [x]`, despite every step's code being implemented. This is a tracking/hygiene gap in the plan document itself, not a functional gap — verified directly against the working code below.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | atlas-data parses `addTime`/`maxDays` | DONE | `services/atlas-data/atlas.com/data/cash/rest.go:46-49` (`AddTime`, `MaxDays` fields), `reader.go:79-80` (parse); commit `f32e26d77`. |
| 2 | atlas-data parses equipment `notExtend` | DONE | `services/atlas-data/atlas.com/data/equipment/rest.go:49`, `reader.go:115`; commit `e64bc45f2`. |
| 3 | `ClassificationExpirationExtender` in atlas-constants | DONE | `libs/atlas-constants/item/constants.go:109` (`Classification(550)`); classifier updated at `services/atlas-channel/.../character_cash_item_use.go:1116` (`if category == item.ClassificationExpirationExtender`); commit `a8407965f`. |
| 4 | Rename shared codec to `ItemUseTargetSlot` | DONE | `libs/atlas-packet/cash/serverbound/item_use_target_slot.go` + `_test.go` present; old name `item_use_item_tag.go` absent; `grep -rn "ItemUseItemTag"` over the tree returns zero hits; call sites at `character_cash_item_use.go:204,340` both use `NewItemUseTargetSlot`; commit `3926d7c6e`. |
| 5 | `libs/atlas-saga` type/action/payload/unmarshal | DONE | `model.go:42` (`ExpirationExtenderUse`), `:226` (`ExtendAssetExpiration` action), `payloads.go:1111` (`ExtendAssetExpirationPayload`), `unmarshal.go:588-589` (decode arm); commit `a9d59b489`. |
| 6 | atlas-inventory `EXTEND_EXPIRATION` contract + cash client | DONE | `kafka/message/compartment/kafka.go:36` (`CommandExtendExpiration`), `:198` (`ExtendExpirationCommandBody`); `data/cash/{model,processor,requests,rest}.go` and `data/cash/mock/processor.go` all present; commit `85961dadf`. |
| 7 | atlas-inventory asset `ExtendExpiration` | DONE | `asset/processor.go:356` implements the curried, flag-preserving method; mock at `asset/mock/processor.go:226`; commit `c0d7621e7`. |
| 8 | atlas-inventory compartment `ExtendAssetExpiration` + cap re-validation | DONE (authorized divergence from plan text) | `compartment/processor.go:1122-1160` re-derives `serverCap` from `cash.Processor.GetById` and **rejects** (`errors.New(...)`, line ~1137-1138) rather than clamping, per the human-ruled D1 resolution to plan.md's self-contradiction. `WithCashProcessor` seam at `:163`. Test `TestExtendAssetExpirationRejectsOverCap` at `processor_test.go:1487` (not the plan's clamp-named test). Commit `74c285a0b`. This is the explicitly authorized divergence — not a defect. |
| 9 | atlas-inventory consumes `EXTEND_EXPIRATION` | DONE | `kafka/consumer/compartment/consumer.go:99` (registration), `:409-411` (`handleExtendExpirationCommand`, type guard against `CommandExtendExpiration`); commit `c4327f0a9`. |
| 10 | atlas-saga-orchestrator mirrors contract + producer | DONE | `kafka/message/compartment/kafka.go:33,171` (mirrored `CommandExtendExpiration`/`ExtendExpirationCommandBody`, struct body byte-identical to atlas-inventory's copy — verified via diff, no output); `compartment/producer.go:138` (`RequestExtendExpirationCommandProvider`); `compartment/processor.go:45,135` (interface + impl); `compartment/mock/processor.go:24,116` (mock); wire-shape pin test `TestExtendExpirationCommandBodyWireShape` at `kafka_test.go:593`; commit `2dcd36f1b`. |
| 11 | atlas-saga-orchestrator aliases, step handler, acceptance entry | DONE | `saga/model.go:45,226,333,1620-1621` (aliases + local unmarshal arm); `saga/handler.go:952-953,1156` (`case ExtendAssetExpiration` dispatch + `handleExtendAssetExpiration`); `saga/event_acceptance.go:127` (`sharedsaga.ExtendAssetExpiration: {EventKindAssetUpdated}`); commit `4660ea30f`. |
| 12 | atlas-saga-orchestrator timer + compensator registration | DONE — all 4 sites present | `saga/timer.go:177` (`reverseWalkSagaTypes`), `:206` (`allSagaTypes`), `:238` (`dispatchTimeoutRollbacks` switch); `saga/compensator.go:268` (`CompensateFailedStep` branch). This is design §9's highest-risk item and all four listed sites are confirmed wired. Commit `1e7a6c6a0`. |
| 13 | atlas-channel mirrors data models | DONE | `data/cash/rest.go:17,21` (`AddTime`/`MaxDays`); `data/equipment/rest.go:12,49` + `model.go:8,25` (`NotExtend` field, `Extract` threading, getter); commit `ea0734b1e`. |
| 14 | atlas-channel `CASH_ITEM_USE` arm | DONE | New file `character_cash_item_use_expiration_extender.go` (resolver `expirationExtenderCashSlotItemType` at line 23, pure evaluator `evaluateExpirationExtension` at line 63); arm wired into `character_cash_item_use.go:333-405` (decode, resolve cash/equipment data, evaluate, build two-step saga); type consts at `:754-755`; saga aliases in `saga/model.go:31,65,99`; commit `a0dc1d85a`. |
| 15 | Version confirmation + verification sweep | PARTIAL — Step 7 unmet (authorized) | Steps 1-6, 8: design.md updated across 3 commits (`58a8a4650`, `f9f13ca44`, `eebe997e6`) with v84/v92 findings correctly hedged as "corroborated, not directly observed" (design.md lines 62-134) rather than falsely claimed as directly verified — matches CLAUDE.md's grounding/honesty rule. Template registration confirmed: all 10 in-scope templates (`template_gms_{48,61,72,79,83,84,87,92,95}_1.json`, `template_jms_185_1.json`) carry `CharacterCashItemUseHandle`. **Step 7 (live client verification) was not performed** — no running server/client in this environment; this is the second explicitly authorized divergence, to be declared in the PR, not a silent skip. |

**Completion Rate:** 15/15 tasks implemented (100%); 14/15 fully as specified, 1/15 (Task 8) implements the authorized D1 correction to the plan's self-contradictory text, 1/15 (Task 15) has one authorized-unmet sub-step (live verification).
**Skipped without approval:** 0
**Partial implementations:** 1 (Task 15, live-verification sub-step only — pre-authorized as unverifiable in this environment)

## Skipped / Deferred Tasks

None skipped or deferred without authorization.

- **Task 8** diverges from plan.md's literal Task-8 text (clamp) but implements the Global-Constraints-mandated D1 behavior (reject). This was pre-authorized by the human and is confirmed correct in code, comments, and test naming. plan.md itself was intentionally left uncorrected, so the document still shows the contradiction — expected, not a defect.
- **Task 15 Step 7** (live client verification: drag a 7-day sandglass onto a time-limited equip on a v83 tenant, confirm tooltip update without relog and single consumption, confirm rejection path leaves the sandglass) was not performed. No running server/game client exists in this sandboxed environment. This is a known, declared gap for the PR description — not evidence of unfinished server-side code, since every code path it would exercise (gate evaluation, saga creation, two-step saga execution, cap re-derivation, refund-on-failure) is otherwise covered by unit/integration tests per the earlier (already-confirmed) `go test -race ./...` runs across all 7 touched modules.

## Build & Test Results

Per the task brief, all builds/tests/guards/bakes were already run and confirmed green immediately prior to this audit and were **not re-run** here (explicitly instructed as settled/expensive):

| Service/Module | Build | Tests (-race) | Vet | Notes |
|---|---|---|---|---|
| libs/atlas-constants | PASS (prior run) | PASS (prior run) | PASS (prior run) | Not re-run this session |
| libs/atlas-packet | PASS (prior run) | PASS (prior run) | PASS (prior run) | Not re-run this session |
| libs/atlas-saga | PASS (prior run) | PASS (prior run) | PASS (prior run) | Not re-run this session |
| atlas-data | PASS (prior run) | PASS (prior run) | PASS (prior run) | docker bake confirmed |
| atlas-channel | PASS (prior run) | PASS (prior run) | PASS (prior run) | docker bake confirmed |
| atlas-inventory | PASS (prior run) | PASS (prior run) | PASS (prior run) | docker bake confirmed |
| atlas-saga-orchestrator | PASS (prior run) | PASS (prior run) | PASS (prior run) | docker bake confirmed |

Guards (redis-key, goroutine, skill-job-id, buff-duration, lint --check): all exit 0 (prior run, not re-run).
Working tree is clean (`git status --short` empty) at time of this audit.

## Overall Assessment

- **Plan Adherence:** FULL (accounting for the two explicitly pre-authorized divergences — the D1 reject-vs-clamp correction and the environment-blocked live-verification step)
- **Recommendation:** READY_TO_MERGE, contingent on the PR description explicitly declaring Task 15 Step 7 (live client verification) as not performed in this environment, per CLAUDE.md's honesty rule.

## Action Items

1. Before opening the PR, add an explicit note in the PR description stating that Task 15 Step 7 (live v83 client verification of tooltip refresh and single-consumption) was not performed — no running server/client was available in the development environment — per the plan's own Step 7 instruction ("If the live check cannot be run in this environment, say so explicitly in the PR rather than marking the criterion met").
2. Optional/cosmetic: plan.md's 111 checkboxes remain all unchecked (`- [ ]`) despite full implementation. Not blocking, but future audits would be faster if `/execute-task` checked boxes as it went. No code action needed.
3. No other action items — all 15 tasks are DONE, all four saga registration sites are wired, and the cross-module contract mirror (atlas-inventory ↔ atlas-saga-orchestrator `ExtendExpirationCommandBody`) is byte-identical and pinned by a wire-shape test.

---

# Backend Guidelines Audit — task-222-item-expiration-extenders

- **Guidelines Source:** backend-dev-guidelines skill (DOM-*/SUB-*/FILE-*/EXT-*/SEC-*)
- **Date:** 2026-08-13
- **Scope:** 57 changed files, `6496b9c87f3b5e8a4e602d108c9c3e0327f943eb..eebe997e6`, across `libs/atlas-constants`, `libs/atlas-packet`, `libs/atlas-saga`, `services/atlas-data`, `services/atlas-channel`, `services/atlas-inventory`, `services/atlas-saga-orchestrator`.
- **Build/Test/Guards:** Per task brief, already verified green in all 7 modules (not re-run here; audited code against guidelines instead).

## Design-Constraint Verification (feature-specific, treated as hard requirements)

| Constraint | Verdict | Evidence |
|---|---|---|
| Over-cap is REJECTED, never clamped, anywhere in the feature | PASS | `services/atlas-channel/.../character_cash_item_use_expiration_extender.go:76-79` returns `extensionOutcome{Reason: ...}` on `proposed.After(ceiling)`, no clamp. `services/atlas-inventory/.../compartment/processor.go:1129-1133` returns `errors.New("requested expiration exceeds the extender's server-derived cap")` on `expiration.After(serverCap)`, no clamp. Grepped both files and their tests for `min(`/`Clamp`/`math.Min` — zero matches. |
| `ExtenderTemplateId` carried end-to-end so atlas-inventory re-derives maxDays itself | PASS | `libs/atlas-saga/payloads.go:1111-1118` (`ExtendAssetExpirationPayload.ExtenderTemplateId`) → `saga/handler.go:1161` (orchestrator passes it to `RequestExtendExpiration`) → `compartment/producer.go:138-152` (wire body) → `services/atlas-inventory/.../consumer.go:409-421` → `compartment/processor.go:1126` (`p.cashProcessor.GetById(extenderTemplateId)` — server re-derives `serverCap` independently, at `:1130`, not trusting the channel-computed `expiration` for the cap check). |
| Set-to-absolute, never increment expiration (at-least-once safety) | PASS | `ExtendAssetExpirationPayload.Expiration` is `time.Time` (absolute), doc comment `payloads.go:1105-1109` states it explicitly. `asset/processor.go:373-374`: a redelivered command carrying the already-applied value (`expiration.Equal(a.Expiration())`) is a no-op re-emit, not a second addition — confirmed by `TestExtendExpirationRedeliveryIsIdempotent` (`asset/processor_test.go:244`). |
| `asset.ApplyLock` never reused for this feature | PASS | `asset/processor.go:345-373` defines `ExtendExpiration` as a standalone method, doc comment at `:345-350` states it is "the deliberate mirror of ApplyLock, never a reuse of it." It calls `updateFlagAndExpiration(..., a.Flag(), expiration)` (existing flag preserved), never `AddFlag(af.FlagLock)`. Grepped `compartment/processor.go`'s `ExtendAssetExpiration` for any call to `ApplyLock`/`assetProcessor.ApplyLock` — zero matches. |
| DOM-25: `CashSlotItemType` 61/62 confined to `expirationExtenderCashSlotItemType()` and `GetCashSlotItemType()` | PASS | `grep -rn "\b61\b\|\b62\b"` over the new handler file and its test returns only doc-comment prose and the two `CashSlotItemType(61)`/`CashSlotItemType(62)` const declarations at `character_cash_item_use.go:754-755`; no other production call site holds a bare `61`/`62` literal. |
| DOM-21: no atlas-constants duplication | PASS | `libs/atlas-constants/item/constants.go:109` adds `ClassificationExpirationExtender = Classification(550)` to the existing table (not a service-local reinvention); consumed via the shared constant at `character_cash_item_use.go:1116` (`if category == item.ClassificationExpirationExtender`), not a re-declared local 550. |

## File Responsibilities Checklist — new/materially-changed packages

### `services/atlas-inventory/atlas.com/inventory/data/cash` (new package — REST-client reader, no `model.go`-style persistence)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor in `processor.go` | PASS | `data/cash/processor.go:11-28` — interface + `ProcessorImpl` + `NewProcessor` all here. |
| FILE-02 | RestModel/Transform/Extract/JSON:API methods in `rest.go` | PASS | `data/cash/rest.go:10-45` — `RestModel`, `GetName`/`GetID`/`SetID`, `Extract`. |
| FILE-03 | Cross-service request funcs in `requests.go` | PASS | `data/cash/requests.go:13-19` — `getBaseRequest()`, `requestById()`. |
| FILE-05 | Builder in `builder.go` | **FAIL** | `data/cash/model.go:19-37` defines `ModelBuilder`/`NewModelBuilder`/`Build()` inside `model.go`, not a separate `builder.go`. The file-responsibilities table designates `builder.go` as the builder's home; this package collapses Model+Builder into one file. Per the audit ruling, prevalence of this shape elsewhere in the repo does not exempt it — grading against the table only. Severity: Important (structural, not merely stylistic — it's the exact `<pkg>.go`-collapse pattern the checklist exists to catch, just landing in `model.go` instead of a package-named file). |
| FILE-06 | No catch-all file carrying ≥2 responsibilities | PASS | No single file bundles Processor+RestModel+requests; the FILE-05 finding above is a two-responsibility collapse (Model+Builder) inside `model.go`, which is a documented file — narrower than a full catch-all, hence graded under FILE-05 rather than FILE-06. |

### External HTTP Client Checklist — `services/atlas-inventory/atlas.com/inventory/data/cash` (new; calls atlas-data via `requests.GetRequest[RestModel]`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| EXT-01 | JSON:API relationship interfaces present | PASS | `data/cash/rest.go:35-41` — `SetToOneReferenceID`/`SetToManyReferenceIDs`, no-op but present. |
| EXT-02 | httptest-backed integration test | PASS | `data/cash/processor_test.go:15-40` — `httptest.NewServer` serving a JSON:API fixture (`{"data":{"type":"cash_items",...}}`), asserts `AddTime`/`MaxDays` populate through `GetById`. |
| EXT-03 | 404 distinguished from other failures | **FAIL** | `data/cash/processor.go:26-28` (`GetById`) and its only caller, `compartment/processor.go:1126-1129`, treat every `GetById` error identically — logged and returned as a generic "unable to resolve extender cash data; refusing to extend expiration" rejection, with no `errors.Is(err, requests.ErrNotFound)` branch. A transport failure or a 5xx from atlas-data is indistinguishable from a genuine "extender template id doesn't exist" 404 in logs or behavior. Severity: Minor-Important — functionally the two cases are both correctly rejected (with sandglass refund via the saga compensator), so there's no over-grant risk, but the checklist requires the distinction and it's absent. `grep -rn "ErrNotFound" services/atlas-inventory/atlas.com/inventory/data/cash/` returns zero hits. |
| EXT-04 | URL via `RootUrl`, not hardcoded | PASS | `data/cash/requests.go:13-15` — `requests.RootUrl("DATA")`. |

### `services/atlas-inventory/atlas.com/inventory/asset` (existing domain package — `ExtendExpiration` addition)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | New method in `processor.go` | PASS | `asset/processor.go:356-373`. |
| FILE-05 | Write path stays in `administrator.go` | PASS | `ExtendExpiration` reuses the pre-existing `updateFlagAndExpiration` write helper at `asset/administrator.go:65`, no new inline DB mutation in `processor.go`. |
| — | Mock kept in sync | PASS | `asset/mock/processor.go:37,226-229`. |
| — | Table/case coverage | PASS | `asset/processor_test.go:182` (flag preservation), `:220` (rejects locked/permanent), `:244` (redelivery idempotence). |

### `services/atlas-inventory/atlas.com/inventory/compartment` (existing domain package — `ExtendAssetExpiration`/`ExtendAssetExpirationAndEmit` addition)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | New methods in `processor.go` | PASS | `compartment/processor.go:1100-1153` (`ExtendAssetExpirationAndEmit`), `:1122-1152` (`ExtendAssetExpiration`). |
| DOM-24 | Kafka producer stubbed in emitting tests | PASS | `TestMain` at `processor_test.go:82-91` calls `producertest.InstallNoop()`; the two new tests (`TestExtendAssetExpirationRejectsOverCap`, `TestExtendAssetExpirationHonorsInBoundsRequest`, `processor_test.go:1487,1534`) run under it. No `t.Cleanup(producer.ResetInstance)` follows the `TestMain` install — the one `ResetInstance()` call in this file (`:73`, inside `installCapturingProducer`) is pre-existing infra (confirmed via `git diff` on this file — those lines predate this branch) whose returned restore closure re-installs `producertest.InstallNoop()`, not a bare reset. |
| — | Mock kept in sync | PASS | `compartment/mock/processor.go:59-60,392-402`. |

### `services/atlas-saga-orchestrator` (`compartment`, `saga` packages — additive wiring only)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01/03 | `RequestExtendExpiration` in `processor.go`, wire builder in `producer.go` | PASS | `compartment/processor.go:45,135-137` (interface+impl, one-line delegate to producer); `compartment/producer.go:138-152` (`RequestExtendExpirationCommandProvider`). |
| — | Saga handler dispatch | PASS | `saga/handler.go:952-953` (`case ExtendAssetExpiration:`), `:1155-1165` (`handleExtendAssetExpiration`, payload type-asserted, error path calls `logActionError`). |
| — | Event-acceptance / timer / compensator registration (4-site pattern, design §9's highest-risk item) | PASS | `saga/event_acceptance.go:127`; `saga/timer.go:177` (`reverseWalkSagaTypes`), `:206` (`allSagaTypes`), `:238` (`dispatchTimeoutRollbacks` routes to the shared `DispatchCashItemUseRollbacks`); `saga/compensator.go:268` (`CompensateFailedStep` routes `ExpirationExtenderUse` to the shared `compensateCashItemUse`, which reverse-walks `DestroyAsset → CreateItem` to refund the consumed sandglass). |
| — | Mirror-struct byte-identity (`ExtendExpirationCommandBody`, atlas-inventory ↔ atlas-saga-orchestrator, separate Go modules) | PASS | Both structs (`services/atlas-inventory/.../kafka/message/compartment/kafka.go:192-198` and `services/atlas-saga-orchestrator/.../kafka/message/compartment/kafka.go:167-175`) have identical field names/types/json tags; the orchestrator copy carries an explicit "MIRRORS ..." doc comment naming the twin file. |

### `services/atlas-channel` — new `character_cash_item_use_expiration_extender.go` (pure-function support file, no `model.go`) + `character_cash_item_use.go` arm

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-13/14/15 | Handler delegates to processors only, no direct provider/DB calls | PASS | The new arm (`character_cash_item_use.go:333-405`) calls `character2.NewProcessor(...)`, `cashData.NewProcessor(...)`, `equipmentData.NewProcessor(...)`, `saga.NewProcessor(...)` — all `Processor` constructors, matching the file's pre-existing sibling arms (ItemTag, SealingLock). No `db.Create`/`db.Save`/provider-function call sites. |
| DOM-20 | Table-driven tests | PASS | `character_cash_item_use_expiration_extender_test.go:60-172` (`TestEvaluateExpirationExtension`, 10-case table incl. exactly-at-cap, over-cap, locked, notExtend, permanent, cash-equip, zero-maxDays); `:20-42` and `:44-56` (version-scoping tables). |
| — | `evaluateExpirationExtension` purity (no I/O, no DB) | PASS | `character_cash_item_use_expiration_extender.go:58-83` — pure function of `(now, target, addTime, maxDays)`, no side effects. |

## Data-Model Additions (`atlas-data`, `atlas-channel` mirrors)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| — | New `RestModel` fields carry `,omitempty` / correct json tags, additive (no field reordering that would break json:api attribute maps) | PASS | `services/atlas-data/atlas.com/data/cash/rest.go` (`AddTime`, `MaxDays`), `equipment/rest.go` (`NotExtend`) — additive struct fields, gofmt-realigned but not reordered relative to json semantics. |
| DOM-20 | Table-driven reader tests for new WZ fields | PASS | `services/atlas-data/atlas.com/data/cash/reader_test.go` (`TestReaderSandglassAddTimeAndMaxDays`, 6-case table incl. absent-defaults-to-zero); `equipment/reader_test.go` (`TestReaderNotExtend`, 3-case table incl. absent-defaults-false). |
| EXT-02 | httptest coverage for channel-side mirror reads | PASS | `services/atlas-channel/atlas.com/channel/data/equipment/processor_test.go` — `TestGetByIdReadsNotExtend`, `TestGetById_NotExtendDefaultsFalse` both use `httptest.NewServer` serving real JSON:API fixtures. |

## SEC-* (Security Review)

Not applicable — atlas-channel/atlas-inventory/atlas-saga-orchestrator/atlas-data are not auth/token services; no JWT, session, or credential-handling code touched by this diff.

## Summary

### Blocking (must fix)

None. Both findings below are Important-severity file-hygiene/robustness gaps in a genuinely new package, not functional defects — the feature's core safety properties (reject-not-clamp, absolute-not-increment, cap re-derivation, ApplyLock non-reuse, DOM-25 confinement, DOM-21 reuse) all PASS with direct file:line evidence.

### Non-Blocking (should fix)

- **FILE-05** — `services/atlas-inventory/atlas.com/inventory/data/cash/model.go:19-37`: `ModelBuilder` lives inside `model.go` instead of a dedicated `builder.go`, per the file-responsibilities table.
- **EXT-03** — `services/atlas-inventory/atlas.com/inventory/data/cash/processor.go:26-28`: `GetById` does not distinguish `requests.ErrNotFound` from transport/5xx failures; both paths in `compartment/processor.go:1126-1129` are already correctly fail-closed (reject + saga-compensator refund), so this is a diagnostic/observability gap, not a correctness gap.
