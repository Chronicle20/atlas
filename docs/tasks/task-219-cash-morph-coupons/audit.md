# Plan Audit — task-219-cash-morph-coupons

**Plan Path:** docs/tasks/task-219-cash-morph-coupons/plan.md
**Audit Date:** 2026-08-12
**Branch:** task-219-cash-morph-coupons
**Base Branch:** main (merge-base c126cb7f9354b17494acec9be33d57d6b5bc16c9)

## Executive Summary

All 8 plan tasks are implemented and match the plan's literal code/tests almost verbatim — this is one of the more faithful executions to audit. `go build`, `go vet`, and `go test -count=1` are clean in all four affected modules (`libs/atlas-packet`, `atlas-data`, `atlas-consumables`, `atlas-channel`), including every morph-coupon-specific test named in the plan and PRD §10. The three repo-root guards named in the plan's own verification step (`redis-key-guard.sh`, `goroutine-guard.sh`, `buff-duration-guard.sh`) are clean. The plan.md checkboxes themselves were left unticked (all 48 `- [ ]`, 0 `- [x]`) despite the work being committed across 8 correctly-titled, task-by-task commits — a paperwork gap, not an implementation gap. No blocking issues found.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | `ItemUseMorphCoupon` sub-body codec | DONE | `libs/atlas-packet/cash/serverbound/item_use_morph_coupon.go` matches plan verbatim (empty sub-body, trailing `updateTime` only when `!updateTimeFirst`); `item_use_morph_coupon_test.go` has all 3 planned tests (`TestItemUseMorphCouponUpdateTimeFirstRoundTrip`, `...NoUpdateTimeFirstRoundTrip`, `...EncodedLength`); commit `11979fb8d`. |
| 2 | `atlas-data` parses `spec/morph`/`spec/hp` | DONE | `cash/rest.go` adds `SpecTypeMorph`/`SpecTypeHp`; `cash/reader.go:139-146` adds the two omit-when-zero parses immediately after the `time` parse, matching plan Step 4 exactly; `TestReaderMorphCoupons` (`reader_test.go:970`) and `TestReaderMorphHpAdditiveOnly` (`:1023`) both present and PASS; commit `9ccfc2867`. |
| 3 | `atlas-consumables` cash REST mirrors spec keys | DONE | `cash/rest.go` adds `SpecTypeMorph`, `SpecTypeHp`, `SpecTypeTime` (all 3, matching the plan's "why three not two" note for FR-3.1); `cash/rest_test.go` has all 3 planned tests (`TestRestModelSpecRoundTripsMorphKeys`, `TestSpecTypeWireValues`, `TestGetSpecAbsentKey`); commit `4ba5809b4`. |
| 4 | Pure planner + routing predicate | DONE | `consumable/morph_coupon.go` has `morphCouponPlan`, `computeMorphCouponPlan` (HP/morph/duration independent, omit-when-zero, unscaled ms), `routesToMorphCoupon` (classification-gated, not type-byte). `morph_coupon_test.go` has `TestComputeMorphCouponPlan` (9-row table matching plan), `TestRoutesToMorphCoupon`, `TestMorphCouponNotStandardConsumer`; commit `5114f6ffd`. |
| 5 | `ConsumeMorphCoupon` consumer | DONE | `morph_coupon.go` implements the `morphCouponDeps`/`consumeMorphCoupon`/`ConsumeMorphCoupon` split described in plan Correction 3 — fallible reads (`cash.GetById`, `fields.GetMap`) run concurrently via `model.NewGroup` and complete *before* `compartment.ConsumeItem` (FR-3.3), which always uses `inventory2.TypeValueCash` (FR-3.4), followed by HP change then morph statup (FR-3.5) with unscaled `plan.duration` (FR-3.6) and no "already morphed" guard (FR-3.8). Tests present: `TestConsumeMorphCouponSuccess`, `...CashFetchFailureKeepsCoupon`, `...ConsumeFailureAppliesNoEffects`, `...ZeroSpecs`, `...ReuseWhileMorphedApplies`, `...BindsRealProcessors`; commit `610db9697`. |
| 6 | Route classification 530 in `RequestItemConsume` | DONE | `consumable/processor.go` adds `} else if routesToMorphCoupon(itemId) { itemConsumer = ConsumeMorphCoupon(...) }` inserted immediately before the reward-table fallback (`p.cdp.GetById`), exactly the placement the plan calls mandatory to avoid a silent `ConsumeBare` fall-through. `TestMorphCouponRoutedBeforeRewardFallback` (source-order guard test) present at `morph_coupon_test.go:433`; commit `c6664273f`. |
| 7 | `atlas-channel` handler arm | DONE | `character_cash_item_use.go` adds the classification-gated arm at the exact planned position (after the megaphone block, before the terminal `l.Warnf`), decodes `cashsb.NewItemUseMorphCoupon(updateTimeFirst)` and forwards via the new `requestItemConsumeFunc` test seam to `consumable.NewProcessor(l, ctx).RequestItemConsume(...)`, then `return`s (no fall-through to the warn). No `session.EnableActions` added (FR-4.3 resolved via code comment citing `OnInventoryOperation @0xa1ead9`). Tests present and passing: `TestCharacterCashItemUseHandleFunc_MorphCouponInvokesConsume`, `...V95NoSubBody`, `...TypeByteCollisions` (both corrected-matrix collision rows: v95 gachapon byte-40, v83 pet-evolution byte-41), `...MismatchedSlotNotInvoked`; commit `0d224fcaf`. |
| 8 | Full verification sweep + documentation | DONE | All sub-steps verified independently in this audit (see Build & Test Results below and the repo-root guard results). `docs/research/missing-features/items-and-consumables.md` retires the "Wholly missing #7" row and records the coupon as implemented with the gms_12 no-op and pre-ingest-inert caveats. `docs/TODO.md` gains the "task-219 follow-up: cash WZ re-ingest" section with the two follow-up checkboxes the plan specifies. PRD `prd.md` §10 has all 12 acceptance-criteria boxes ticked `[x]` with cited evidence, including the corrected (not "pre-95 gachapon") type-byte-collision language. Commits `b136989fb` (docs) and `9cb328576` (PRD ticks). One note: the PRD-mandated code-review step (plan Step 9, `superpowers:requesting-code-review`) is not independently verifiable from git history alone — no separate review-agent commit or audit trail from a prior pass exists in this task folder before this session; this audit itself now serves that gate. |

## Skipped / Deferred Tasks

None. All 8 tasks are DONE. No task was marked `PARTIAL`, `SKIPPED`, or silently deferred beyond what the PRD itself explicitly scoped out (operational WZ re-ingest, §9/§10 — tracked in `docs/TODO.md` as required).

One process gap, not a task gap: **plan.md's 48 checkboxes are all still `- [ ]`** even though every task's commits, files, and tests exist and match the plan's literal content. This does not affect functional completeness (verified independently above, file-by-file), but it means the plan document itself cannot be used as a live completion tracker — anyone reading `plan.md` alone would incorrectly conclude 0% progress. Recommend ticking the boxes (or the plan tool doing so as tasks land) so the document and the branch state agree.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| `libs/atlas-packet` | PASS | PASS | `go build ./...` clean; `go test ./... -count=1` all packages `ok` or `no test files`; `cash/serverbound` morph-coupon tests (`TestItemUseMorphCoupon*`) PASS across all `pt.Variants` (GMS v28/48/61/72/79/83/84/86/87/92/95, JMS v185). |
| `services/atlas-data/atlas.com/data` | PASS | PASS | `go build ./...`, `go vet ./...` clean; `go test ./... -count=1` all `ok`; `cash` package's `TestReaderMorphCoupons`/`TestReaderMorphHpAdditiveOnly` PASS; all pre-existing cash tests (`TestReader`, `TestReaderExpCoupons`, etc.) unaffected. |
| `services/atlas-consumables/atlas.com/consumables` | PASS | PASS | `go build ./...`, `go vet ./...` clean; `go test ./... -count=1` all `ok` (`consumable` package run took 16.2s, no failures); `cash` package round-trip tests PASS. |
| `services/atlas-channel/atlas.com/channel` | PASS | PASS | `go build ./...`, `go vet ./...` clean; `go test ./... -count=1` all `ok` including `socket/handler` (0.772s); all 4 morph-coupon handler tests PASS, including both corrected type-byte-collision rows correctly falling through to the terminal warn (confirmed via captured warn-level log lines during the test run). |

**Repo-root guards** (per plan Task 8 Step 2, all required for this diff): `tools/redis-key-guard.sh` — clean (exit 0). `tools/goroutine-guard.sh` — clean (exit 0). `tools/buff-duration-guard.sh` — clean (exit 0). `tools/lint.sh --check` and the four template guards / `service-registration-guard.sh` were not re-run in this audit pass (lint requires nvm/node22 setup per project memory and is orthogonal to plan-adherence; the template/registration guards are explicitly not implicated per the plan's own Step 2 check, reconfirmed below).

**Global-constraint greps** (plan Task 8 Step 4, re-run independently in this audit):
- No `MajorVersion()`/`Region() ==` literal added in `services`/`libs`: confirmed empty.
- No raw `530` classification-selection literal outside `libs/atlas-constants`: all `\b530\b` hits in the diff are inside comments or `GetClassification(...) == ...`/`want 530` test assertions — none is a raw `if x == 530` gate.
- `git diff --name-only main...HEAD -- '*go.mod' '*go.sum'`: empty — no `go.mod` touched, so `docker buildx bake` is correctly not required.
- `git diff --stat main...HEAD -- services/atlas-configurations/seed-data/templates/`: empty — no seed template edited, consistent with the PRD's gms_12 no-op scoping.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (pending: tick plan.md's checkboxes for documentation hygiene, and confirm the code-review step named in plan Task 8 Step 9 has run — this audit satisfies the plan-adherence half of that gate; a `backend-guidelines-reviewer` pass is still recommended before opening the PR per CLAUDE.md's "Code Review Before PR" rule if it has not already run this session).

## Action Items

1. Tick all 48 checkboxes in `plan.md` (or otherwise reconcile the plan document with the completed branch state) so the file is not misleading to a future reader.
2. Run `backend-guidelines-reviewer` (DOM-* checklist) if it has not already run in this task's history, per CLAUDE.md's mandatory code-review-before-PR rule — this audit only covers plan-adherence, not guideline compliance.
3. Optional/low-priority: run `tools/lint.sh --check` locally with nvm/node22 available to confirm the Go-side formatters are clean (PRD §10's own acceptance criterion already claims this was done in a prior `task-8-report.md` sweep with all 86/86 Go targets reporting "0 issues" and only the unrelated `ui:node-version` target failing on a pre-existing node-version mismatch — not re-verified independently in this audit pass).

---

# Backend Audit — task-219-cash-morph-coupons (DOM-* checklist)

- **Scope:** Changed Go packages only, per `git diff --stat main...HEAD -- libs services`:
  - `libs/atlas-packet/cash/serverbound` (new codec `item_use_morph_coupon.go` + test)
  - `services/atlas-data/atlas.com/data/cash` (`reader.go`, `rest.go` — additive WZ spec parsing)
  - `services/atlas-consumables/atlas.com/consumables/cash` (`rest.go` — additive SpecType mirror)
  - `services/atlas-consumables/atlas.com/consumables/consumable` (new `morph_coupon.go`, one routing branch in `processor.go`)
  - `services/atlas-channel/atlas.com/channel/socket/handler` (new classification-gated arm in `character_cash_item_use.go`)
- **Guidelines Source:** backend-dev-guidelines skill (file-responsibilities.md, ai-guidance.md, anti-patterns.md, patterns-functional.md)
- **Date:** 2026-08-12
- **Build:** PASS (all 4 modules, `go build ./...`)
- **Tests:** PASS (all 4 modules, `go test ./... -count=1`; no FAIL/panic lines in any module's output)
- **Overall:** PASS

## Build & Test Results

`go build ./...` and `go test ./... -count=1` run from each module root:

| Module | Build | Tests |
|---|---|---|
| `libs/atlas-packet` | PASS | PASS |
| `services/atlas-data/atlas.com/data` | PASS | PASS |
| `services/atlas-consumables/atlas.com/consumables` | PASS | PASS |
| `services/atlas-channel/atlas.com/channel` | PASS | PASS |

## Domain Discovery

None of the touched packages are domain packages (no `model.go` introduced/present for a new persisted entity):

- `libs/atlas-packet/cash/serverbound` — packet codec package (struct + `Encode`/`Decode`/`Operation`/`String`), not a DOM/SUB/support REST package; DOM/SUB checklists don't apply, only "does it follow the codec-file convention" (it does — one file per client arm, matching every sibling `item_use_*.go` in the directory).
- `services/atlas-data/atlas.com/data/cash` — support package (WZ reader → `RestModel`, no `model.go`/DB). Change is additive parsing only; `processor.go`/`resource.go`/`registry.go` untouched.
- `services/atlas-consumables/atlas.com/consumables/cash` — support/REST-client package (has `model.go`, `processor.go`, `requests.go`, `rest.go` — pre-existing before this task). Change is additive `SpecType` constants in `rest.go` only.
- `services/atlas-consumables/atlas.com/consumables/consumable` — sub-domain/action package (no `model.go`; `processor.go` + per-item-type files `reward.go`, `skill_book.go`, `vega.go`, `morph.go`, `morph_coupon.go`, `producer.go`, `processor_catch.go`). New file `morph_coupon.go` follows the pre-existing sibling convention exactly (`reward.go`, `vega.go`).
- `services/atlas-channel/atlas.com/channel/socket/handler` — handler package, one new arm inside an existing dispatch function.

## File Responsibilities Checklist

| ID | Check | Status | Evidence |
|---|---|---|---|
| FILE-01 | Processor logic in `processor.go` | PASS | `libs/atlas-packet/cash/serverbound/item_use_morph_coupon.go` has no `Processor` type (not applicable — codec package). `services/atlas-consumables/atlas.com/consumables/cash/processor.go:11-32` unchanged, holds `Processor`/`ProcessorImpl`/`NewProcessor`. `services/atlas-consumables/atlas.com/consumables/consumable/processor.go:68,86,95` holds `Processor` interface/`ProcessorImpl`/`NewProcessor`; the diff only adds one `else if` branch inside the existing `RequestItemConsume` method (`processor.go` diff, +8/-0) — no new Processor method was placed elsewhere. |
| FILE-02 | RestModel + Transform/Extract in `rest.go` | PASS | `services/atlas-consumables/atlas.com/consumables/cash/rest.go:33-66` has `RestModel`, `GetName`/`GetID`/`SetID`, `Extract`. `services/atlas-data/atlas.com/data/cash/rest.go:42-70` has `RestModel` + JSON:API methods (this package's `Transform` equivalent — `Read` — lives in `reader.go`, matching this package's own pre-existing convention, unchanged by this diff). No RestModel logic leaked into `morph_coupon.go` or `item_use_morph_coupon.go`. |
| FILE-03 | Cross-service request funcs in `requests.go` | PASS | `services/atlas-consumables/atlas.com/consumables/cash/requests.go:14-20` (`getBaseRequest`, `requestById`) — unchanged by this diff, still the only place `requests.GetRequest[RestModel]` is called for this package. `morph_coupon.go` calls `cash.Processor.GetById` (the processor method), never `requests.*` directly — correct layering. |
| FILE-04 | Entity + Migration + TableName in `entity.go` | N/A | No `entity.go` touched or required — none of the changed packages persist a new GORM entity. `cash` (atlas-consumables) is a pure REST-client package with no local storage. |
| FILE-05 | Builder/Model/administrator/provider/state placement | N/A (mostly) | No `builder.go`/`administrator.go`/`provider.go` required — the feature adds no new persisted domain object. `cash.Model` (atlas-consumables) is unchanged; its immutable-model shape (`services/atlas-consumables/atlas.com/consumables/cash/model.go:3-9`, private fields + `GetSpec`/`Indexes`/`PetSkills`/`PetSkillAdd` accessors) already satisfies FILE-05's Model half. |
| FILE-06 | No package-named catch-all file | PASS | `morph_coupon.go` (consumable package) holds only: a plan struct, a pure planner function, a routing predicate, a private deps struct, a private consumer function, and the exported entrypoint — all one cohesive responsibility (interpret-then-consume one item type), matching the established per-item-type split already used by `reward.go` (`rollReward`), `morph.go` (`selectMorph`/`rollMorph`), `vega.go`, `skill_book.go` in the same package. It does **not** bundle Processor+RestModel+requests (the task-102 `wallet.go` anti-pattern this check exists to catch) — `Processor`/`ProcessorImpl` stay in `processor.go` (verified above), no `RestModel` or `requests.*` call appears in `morph_coupon.go`. `item_use_morph_coupon.go` (packet lib) is a single-struct codec file, one of ~40 sibling `item_use_*.go` files in the same directory — the established per-client-arm convention for that package, not a catch-all. |

## Domain Checklist (DOM-*) — applicable items only

No package in scope has a `model.go` for a *new* domain object, so the full DOM-01..20 checklist (builder/entity/ToEntity/etc.) is N/A across the board. The applicable cross-cutting DOM items:

| ID | Check | Status | Evidence |
|---|---|---|---|
| DOM-06/07 | Processor accepts `FieldLogger`; handlers pass `d.Logger()` | PASS (unaffected) | `services/atlas-consumables/atlas.com/consumables/cash/processor.go:20` `NewProcessor(l logrus.FieldLogger, ctx context.Context)` — unchanged. `consumable/morph_coupon.go:144` `NewProcessor(l, ctx)` inside `ConsumeMorphCoupon` receives `l logrus.FieldLogger` from the `ItemConsumer` closure, sourced from the Kafka consumer/handler layer, never `logrus.StandardLogger()`. |
| DOM-09 | Transform errors handled | PASS | No new `Transform(` call sites added; `Extract` in `cash/rest.go` (atlas-consumables) is pre-existing and unchanged; `Read(...)` in `atlas-data/cash/reader.go` returns `model.ErrorProvider[[]RestModel](err)` on every parse error (unchanged pattern, new `morph`/`hp` parse lines at `reader.go:142-147` follow the same `GetIntegerWithDefault` + omit-when-zero pattern as the pre-existing `expR`/`drpR`/`time` lines directly above them — no swallowed error path introduced). |
| DOM-12 | No `os.Getenv()` in handlers | PASS | `grep -n os.Getenv services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go` — zero matches. |
| DOM-13/14 | No cross-domain logic / no direct provider calls in handlers | PASS | The new arm in `character_cash_item_use.go:660-668` decodes the sub-body and forwards to `requestItemConsumeFunc` (a package-level test seam, precedent-matched to `cashItemInSlotFunc`/`useRockFunc` in the same file), which wraps `consumable.NewProcessor(l, ctx).RequestItemConsume(...)` — a processor call, not a provider call. |
| DOM-15 | No direct entity creation in handlers | PASS | `grep -n 'db\.\(Create\|Save\|Delete\)' character_cash_item_use.go` — zero matches. |
| DOM-21 | No duplication of atlas-constants types | PASS | `item2.ClassificationTransformationCoupon` = `libs/atlas-constants/item/constants.go:97` (`Classification(530)`), reused directly (`consumable/morph_coupon.go:72`, `character_cash_item_use.go:660`) — not redeclared. `ts.TemporaryStatTypeMorph` = `libs/atlas-constants/character/temporary_stat.go:39`, reused directly (`morph_coupon.go:50`) — not redeclared. No new numeric classification constant or item-id-division helper was introduced in the diff. |
| DOM-24 | Kafka producer stubbed in tests that emit | PASS | `services/atlas-consumables/atlas.com/consumables/consumable/testmain_test.go:15-18` — package `TestMain` installs `producertest.InstallCapturing()` (the shared `libs/atlas-kafka/producer/producertest` package, not a service-local stub) once for the whole package; no `t.Cleanup(producer.ResetInstance)` reverts it anywhere in `morph_coupon_test.go`. `character_cash_item_use.go`'s new arm never emits directly — it calls `RequestItemConsume`, whose Kafka emission path is exercised, if at all, through the already-stubbed `consumable` package tests, and the handler-level test (`character_cash_item_use_test.go`) uses the `requestItemConsumeFunc` seam instead of a live emit path. |
| DOM-25 | Client-interpreted byte values config-resolved | PASS | `ItemUseMorphCoupon` (`libs/atlas-packet/cash/serverbound/item_use_morph_coupon.go`) carries no client wire-code literal — its only field is `updateTime` (a passthrough timestamp) plus a caller-supplied `updateTimeFirst bool` version gate (not a client-interpreted lookup byte). The routing gate in `character_cash_item_use.go:660` (`category == item.ClassificationTransformationCoupon`) is an item-classification check, not a wire-code table lookup — the classification itself is a stable server-side derivation (`itemId / 10000`), already covered by DOM-21. No new `WithResolvedCode`/notice-reason table was warranted by this feature and none was silently hardcoded in its place. |
| DOM-26 | Goroutines via `routine.Go` | PASS | `tools/goroutine-guard.sh` exit 0 (verified from repo root); no bare `go` statement in the diff (`git diff main...HEAD -- libs services` contains no `+\s*go (func|[A-Za-z_])` line). |
| DOM-28 | No silent degradation in decorators/enrichment | PASS (with one documented, non-silent exception) | `consumeMorphCoupon` (`morph_coupon.go:115-127`) does not silently swallow the HP-heal or buff-apply failure paths: each `if err != nil` branch calls `l.WithError(err).Errorf(...)` naming the character, item, and failed effect — this is a logged degradation, not a bare `return m`. It does not additionally call `degrade.Observe` / increment `atlas_enrichment_degraded_total`, but this code path is a **write-side item-consumer effect application** (heal/morph after a committed consume), not a `model.Decorator[...]` read-side enrichment — DOM-28's trigger condition (decorator/enrichment implementations) does not match this shape, so the metric requirement does not apply here. Flagged as a note, not a finding. |

## SEC-* Security Review

Not applicable — none of the four modules in scope handle authentication, authorization, or token management.

## Summary

### Blocking (must fix)

None.

### Non-Blocking (should fix)

None. One informational note only: DOM-28's `degrade.Observe` metric convention was checked against `consumeMorphCoupon`'s two post-commit effect failures and found not to apply (write-path effect application, not a `model.Decorator` read-side enrichment) — recorded above for traceability, not as a deviation.

## Verdict

**PASS.** All four changed modules build and test clean. No DOM/SUB/FILE-responsibilities violation was found with file:line evidence in the changed packages. The `consumable/morph_coupon.go` file-per-item-type shape matches its package's own established convention (`reward.go`, `vega.go`, `skill_book.go`) and does not exhibit the task-102 `<pkg>.go` catch-all anti-pattern (Processor/RestModel/requests stay correctly separated). Kafka emission in tests is stubbed via the shared `producertest` package with no per-test cleanup reversion. No client wire-code literal was introduced. No atlas-constants type was duplicated.
