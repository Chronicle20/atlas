# Plan Adherence Review (FR-1)

**Plan Path:** `docs/tasks/task-160-skill-cast-consumption/plan.md`
**Audit Date:** 2026-07-27
**Branch:** `task-160-skill-cast-consumption`
**Base Commit for diff:** `33ed1e14b` (parent of the first FR-1 commit)
**HEAD:** `e653f3658`
**Scope:** FR-1 only, per the plan's 2026-07-27 "Scope reconciliation" note. FR-2/FR-3 (Shadow Stars) and the `SpiritJavelinItemId()` getter are recorded in the plan's "Superseded scope" table as already shipped by task-158 (PR #1003) and are explicitly out of scope for this review — confirmed untouched below.

## Executive Summary

All three FR-1 implementation tasks (compartment slot-selection helper, `RequestItemConsume` quantity parameter, `UseSkill` itemCon wiring) were faithfully implemented exactly as specified in the plan, each as its own commit with the plan's exact commit message. The Task 4 verification sweep (test/vet/build, docker bake, redis-key-guard, goroutine-guard, lint) all pass clean. No changes exist outside `services/atlas-channel/`, and the Shadow Stars files (`shadow_stars.go`, `character_attack_projectile.go`) are untouched, confirming the descoped items were correctly left alone. Plan adherence is FULL.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | `compartment.FindFirstByItemIdWithQuantity` — lowest-slot-with-quantity lookup | DONE | `services/atlas-channel/atlas.com/channel/compartment/model.go:68-85` — method added after `FindFirstByItemId` (line 61) with `"sort"` import added; sorts matches by slot ascending, skips slots holding less than the requested quantity, per-plan doc comment retained verbatim. Tests: `compartment/model_test.go` — all 5 plan-specified tests (`TestFindFirstByItemIdWithQuantity_LowestSlotWinsUnsortedInput`, `_SkipsShortSlots`, `_ExactBoundary`, `_NoSlotQualifies`, `_ItemAbsent`) present and passing (`go test -race ./compartment/... -v` — all PASS). Commit `175bc683f` matches plan's exact message "feat(channel): compartment lowest-slot-with-quantity lookup (task-160 FR-1)". |
| 2 | `RequestItemConsume` gains a `quantity` parameter | DONE | Interface: `consumable/processor.go:18` — `RequestItemConsume(f field.Model, characterId character.Id, itemId item.Id, source slot.Position, quantity int16, updateTime uint32) error`. Impl: `consumable/processor.go:41-48` — `quantity < 1 → 1` floor guard with FR-1 comment, then passes `quantity` into `RequestItemConsumeCommandProvider`. All 8 call sites across 7 files updated with literal `1`: `skill/handler/common.go` (interim, later replaced by Task 3), `shopscanner/processor.go:79`, `socket/handler/pet_food.go:23`, `socket/handler/pet_item_use.go:23`, `socket/handler/character_cash_item_use.go:66`, `socket/handler/character_item_use.go:23,32,50` — confirmed via `git diff 33ed1e14b..HEAD` on each file. `consumable/mock/processor.go` was also updated (`ProcessorMock.RequestItemConsumeFunc`/`RequestItemConsume` signature) — not explicitly listed in the plan's file table but required by the interface change and consistent with the plan's "compiler enforces the call-site sweep" intent; correctly caught. Regression-pin test `consumable/producer_test.go` (`TestRequestItemConsumeCommandProvider_CarriesQuantity`) present verbatim per plan Step 1 and passing. Commit `7642195ab` matches plan's exact message. |
| 3 | `UseSkill` itemCon path — real amount + lowest-qualifying-slot + seams | DONE | `skill/handler/common.go:36-46` — both seams added exactly as specified: `loadCasterWithInventoryFunc` (line 39, delegates to `cp.GetById(cp.InventoryDecorator)`) and `requestItemConsumeFunc` (line 45, delegates to `p.RequestItemConsume`). itemCon block rewritten at `common.go:118-131`: `amount := int16(e.ItemConsumeAmount())` (line 122), floors `<1` to `1`, calls `FindFirstByItemIdWithQuantity(itemId, amount)` (line 127), and on shortfall logs the plan's exact updated warn message "no single slot holds enough; cast permitted" (line 130). Test file `skill/handler/common_consume_test.go` (178 lines) created with all 4 plan-specified tests (`TestUseSkill_ItemConsumeAmountPlumbed`, `_ItemConsumeAmountZeroFloorsToOne`, `_ItemConShortfallSkipsButCastProceeds`, `_ItemConCasterLoadFailureCastProceeds`) — all PASS (`go test -race ./skill/handler/... -v`). Commit `e653f3658` matches plan's exact message. |
| 4 | Full verification sweep | DONE | See Build & Test Results below — every sub-step (test/vet/build, docker bake, redis-key-guard, goroutine-guard, lint --check) run fresh during this audit and passed clean. No verification-fix commit exists on the branch (only the 3 plan commits), consistent with plan Step 5's "if nothing changed, no commit." |

**Completion Rate:** 4/4 tasks (100%) — all 21 plan checkboxes remain unchecked (`- [ ]`) in `plan.md` itself (the plan document was never re-saved with `[x]` marks), but every step's file-level deliverable and every specified test is present in the diff and passing; classified DONE on evidence, not on checkbox state.
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. The plan's own "Superseded scope" table (lines 644-653) lists FR-2/FR-3 Shadow Stars items and the `SpiritJavelinItemId()` getter as intentionally not part of this plan (shipped by task-158/PR #1003) — verified these files are absent from the `33ed1e14b..HEAD` diff:
- `skill/handler/shadow_stars.go` — 0 lines changed
- `socket/handler/character_attack_projectile.go` — 0 lines changed

## Build & Test Results

| Service | Build | Tests (`-race`) | `go vet` | Notes |
|---------|-------|-------|----------|-------|
| atlas-channel | PASS | PASS | PASS | `go build ./...`, `go test -race ./...` (all packages `ok`, including `compartment`, `consumable`, `skill/handler`, `socket/handler`), `go vet ./...` — all clean from `services/atlas-channel/atlas.com/channel`. |

Additional CLAUDE.md-mandated verification, all clean:
- `docker buildx bake atlas-channel` — exit 0, image built and exported (`docker.io/library/atlas-channel:local`).
- `tools/redis-key-guard.sh` (repo root) — exit 0, no keyed-redis violations.
- `tools/goroutine-guard.sh` (repo root) — exit 0, no bare `go` statements.
- `tools/lint.sh --check` (repo root) — every one of 29 module lint runs reported "0 issues." (including `services/atlas-channel/atlas.com/channel`); stray `level=warning` lines reference stale files in unrelated sibling worktrees (`task-147-...`, `task-161-...`) from a shared lint cache and are not failures in this branch.
- `service-registration-guard.sh` / `template-opcode-order-guard.sh` — correctly N/A per plan (no services.json/k8s/docker-bake/go.work/template changes in this diff; confirmed via the file-scope check above).

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None. No gaps found.

---

# Backend Guidelines Review (FR-1)

- **Service Path:** services/atlas-channel/atlas.com/channel
- **Guidelines Source:** backend-dev-guidelines skill (DOM-*/SUB-*/FILE-*/EXT-*/SEC-*)
- **Scope:** `git diff 33ed1e14b..HEAD` (HEAD `e653f3658`) — 12 files, FR-1 (skill-cast itemCon quantity plumbing). Shadow Stars (task-158) explicitly excluded per instructions.
- **Date:** 2026-07-27
- **Build:** PASS — `go build ./...` in `services/atlas-channel/atlas.com/channel` clean.
- **Tests:** PASS — `go test ./... -count=1` clean, all packages `ok` (compartment, consumable, skill/handler, socket/handler included). `go vet` clean on all touched packages.
- **Overall:** PASS (zero blocking findings against documented DOM-*/SUB-*/FILE-*/EXT-* items in the diffed lines)

## Files in scope

| File | Kind |
|---|---|
| `compartment/model.go` | domain package method (`FindFirstByItemIdWithQuantity`) |
| `compartment/model_test.go` | new tests |
| `consumable/mock/processor.go` | mock signature update |
| `consumable/processor.go` | `Processor`/`ProcessorImpl.RequestItemConsume` signature + floor logic |
| `consumable/producer_test.go` | new test |
| `shopscanner/processor.go` | call-site arg addition (`1,`) |
| `skill/handler/common.go` | new seams (`loadCasterWithInventoryFunc`, `requestItemConsumeFunc`) + itemCon block rewrite |
| `skill/handler/common_consume_test.go` | new tests |
| `socket/handler/character_cash_item_use.go` | call-site arg addition |
| `socket/handler/character_item_use.go` | call-site arg addition (×3) |
| `socket/handler/pet_food.go` | call-site arg addition |
| `socket/handler/pet_item_use.go` | call-site arg addition |

## Package Classification (Phase 2)

- `compartment` — domain package (has `model.go`). Only `model.go`/`model_test.go` are in the diff; the rest of the domain checklist (entity/administrator/provider — this package has no `entity.go`, consistent with atlas-channel being a stateless read-through service) is pre-existing and out of scope per task instructions.
- `consumable` — support package (`processor.go` + `producer.go`, no `model.go`/`resource.go`). Runs File Responsibilities + a Kafka-emission check (no REST client, so EXT-* does not trigger).
- `skill/handler` — support/orchestration package (no `model.go`, no `resource.go`; peer files are `mount.go`, `shadow_stars.go`, `mob_select.go`, `registry.go`). `common.go` holds cross-cutting seam vars and the `UseSkill` orchestrator — this is the package's established shape (not a Processor/RestModel/requests collapse), so FILE-06 does not trigger.
- `socket/handler`, `shopscanner` — call-site-only changes (one integer literal argument added per call); no new symbols.

## Domain Checklist Results

### compartment

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-05 | Domain `Model` methods live in `model.go` | PASS | `compartment/model.go:73` — `FindFirstByItemIdWithQuantity` is a method on `Model`, placed in `model.go` alongside the sibling `FindFirstByItemId` (line 61) and `FindByPetId`. |
| DOM-21 | No duplication of atlas-constants types | PASS | `quantity int16` (`compartment/model.go:73`) matches the existing wire type `RequestItemConsumeBody.Quantity int16` (`kafka/message/consumable/kafka.go:38`, pre-existing) and the sibling `compartment.Drop`/`DropAssetCommandProvider` quantity parameters (`compartment/processor.go:24`, `compartment/producer.go:57`, pre-existing `int16` convention in this package). No new type was declared; `libs/atlas-constants/asset.Quantity` (`uint32`) is a different concern (asset-instance quantity) and is not what the wire body carries. |
| DOM-20 | Table-driven tests | WARN (non-blocking) | `compartment/model_test.go:76-129` — the five new `TestFindFirstByItemIdWithQuantity_*` functions are each a standalone `func Test...(t *testing.T)`, not a `tests := []struct{...}{}` + `t.Run` table. `testing-guide.md:18` states table-driven is a *preference* ("Prefer table-driven tests"), not a hard MUST, and the pre-existing sibling test `TestFindFirstByClassification` (`compartment/model_test.go:1-42`, unchanged) uses the same non-table style, so this is a pattern the file already had — flagged as non-blocking per the guideline's own "prefer" wording, not softened for prevalence. |

### consumable

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor interface + impl live in `processor.go` | PASS | `consumable/processor.go:17` (`type Processor interface`), `:41` (`func (p *ProcessorImpl) RequestItemConsume`, the changed method) — both in `processor.go`. |
| DOM-06 | Processor accepts `logrus.FieldLogger` | PASS | `consumable/processor.go:29` — `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor`. |
| FILE-01 | Mock lives in `mock/` subpackage, mirrors interface | PASS | `consumable/mock/processor.go:23` — `ProcessorMock.RequestItemConsume` signature matches `Processor.RequestItemConsume` exactly (both add `quantity int16` in the same position); `var _ consumable.Processor = (*ProcessorMock)(nil)` (`consumable/mock/processor.go:16`, unchanged) statically enforces this. |
| DOM-24 | Kafka producer stubbed in tests that emit | PASS (N/A — no real emit) | `consumable/producer_test.go:23` calls `RequestItemConsumeCommandProvider(...)` and then `provider()` directly — this only builds a `kafka.Message` slice (`consumable/producer.go:16-30`) and never reaches `producer.ProviderImpl`/`message.Emit`. No stub is required because no network/retry path is exercised. |
| — | Floor logic (`<1 → 1`) placed at processor layer per design | PASS | `consumable/processor.go:42-45` — `if quantity < 1 { quantity = 1 }`, with an explicit comment citing FR-1; this is the documented defense-in-depth duplicate the task context blesses, not flagged. |

### skill/handler (support package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| — | New seams follow the file's established `var funcName = func(...)` test-seam convention | PASS | `skill/handler/common.go:36-38` (`loadCasterWithInventoryFunc`) and `:41-44` (`requestItemConsumeFunc`) match the shape and doc-comment style of the pre-existing `loadCasterFunc` (`:29-33`), `rectQueryFunc` (`:47-50`), `propRollFunc` (`:53-62`) seams — each documents what it replaces and why. |
| DOM-13/14 equivalent | Orchestrator calls only its own processor / a documented seam, no direct cross-domain writes | PASS | `skill/handler/common.go:129` — `requestItemConsumeFunc(consumable.NewProcessor(l, ctx), f, ...)` goes through the `consumable.Processor` interface; no `db.Create`/`db.Save` or bypass of the processor layer. |
| DOM-26 | No bare `go` statements introduced | PASS | `git diff 33ed1e14b..HEAD \| grep -nE '^\+.*\bgo (func\|[A-Za-z_])'` returns no matches — no goroutines added in this diff. |
| DOM-24 | Kafka producer stubbed in tests that emit (transitive) | PASS | `skill/handler/common_consume_test.go:107-113` (`runUseSkill`) drives `UseSkill(...)`, which for the new tests' `consumeEffect(itemConsume, amount)` fixture (`:100-109`) has `HPConsume()==0`, `MPConsume()==0`, `Cooldown()==0`, `Duration()==0`/empty `statupsToApply`, empty `AffectedMobIds()`, and `testConsumeSkillId` unregistered in `Lookup` (`skill/handler/registry.go:35`) — every other emit-capable branch in `UseSkill` (`skill/handler/common.go:117-171`) is a no-op for this fixture, and the one branch that would emit (`requestItemConsumeFunc`) is replaced by the test's recorder (`common_consume_test.go:56-61`). No real `producer.ProviderImpl`/`message.Emit` call is reachable from these four new tests, so no stub is required. |
| Test Helper Pattern (CLAUDE.md) | Builder pattern used, no `*_testhelpers.go` | PASS | `common_consume_test.go:75` (`asset.NewModelBuilder(...).SetSlot(...).SetQuantity(...).Build()`) and `:86` (`compartment.NewBuilder(...)`) use the domain builders; the file is `common_consume_test.go` (a `_test.go` file colocated with `common.go`), not a `*_testhelpers.go` constructor file. |

### socket/handler, shopscanner (call-site plumbing)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| — | Call sites pass a correct literal (`1`) for the pre-existing single-item paths | PASS | `socket/handler/character_item_use.go:23,32,50`, `socket/handler/pet_food.go:23`, `socket/handler/pet_item_use.go:23`, `socket/handler/character_cash_item_use.go:66`, `shopscanner/processor.go:79` — each adds `, 1,` before `updateTime`/at the new `quantity` position, preserving prior single-item-consume behavior for paths not driven by `effect.Model.ItemConsumeAmount()`. Confirmed by `go build ./...` succeeding (a wrong arg count/order would not compile) and `go test ./socket/handler/... ./shopscanner/...` passing unchanged. |

## Sub-Domain Checklist Results

N/A — no package in the diff has a `resource.go` without a `model.go` (no new action-event sub-domain package was added or touched).

## External HTTP Client Checklist

N/A — no file in the diff calls `requests.GetRequest[T]`/`requests.PostRequest[T]`/`requests.RootUrl(...)`. `consumable.RequestItemConsume` emits a Kafka command (`consumable/producer.go:16`), and `loadCasterWithInventoryFunc` reuses the pre-existing `character.Processor.GetById` REST call path (unchanged by this diff).

## Security Review

N/A — atlas-channel's skill-cast/consumable path is not an authentication/authorization/token-management surface; Phase 4 SEC-* checks do not apply to this diff.

## Notes (not guideline violations — informational only)

- `skill/handler/common.go:120` — `amount := int16(e.ItemConsumeAmount())` narrows a `uint32` (`data/skill/effect/model.go:93`) to `int16`. No DOM/anti-pattern item in the guidelines addresses narrowing conversions, so this is not scored as a finding, but is noted per the audit's adversarial mandate: if WZ `itemConNo` ever exceeded 32767 the cast to `int16` would wrap negative, and the immediately-following `if amount < 1 { amount = 1 }` (`:121-123`) would silently floor a corrupted large value to `1` rather than surfacing an error. Not evaluated further — out of the mechanical DOM-*/SUB-* checklist scope, and no WZ data was inspected to determine whether any real `itemConNo` value approaches that range.

## Summary

### Blocking (must fix)

None.

### Non-Blocking (should fix)

- DOM-20 (compartment): `compartment/model_test.go:76-129` — the 5 new `FindFirstByItemIdWithQuantity` tests are standalone functions rather than a table-driven `t.Run` block. Matches the pre-existing sibling test's style; guideline wording is "prefer," not mandatory.
