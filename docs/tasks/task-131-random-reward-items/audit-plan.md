# Plan Audit — task-131-random-reward-items

**Plans audited:**
- `docs/tasks/task-131-random-reward-items/plan.md` (backend feature, 15 tasks)
- `docs/tasks/task-131-random-reward-items/plan-ui-possible-rewards.md` (UI add-on, 2 tasks)

**Audit Date:** 2026-07-16
**Branch:** task-131-random-reward-items
**Diff Scope:** `b1c50b67d36c9c7174bdd2977de635b8b074051c..23e9d3c20`
**Base Branch:** main

## Executive Summary

The backend feature and the UI add-on were both faithfully implemented — every
numbered task in both plans has corresponding code, and the implementation
frequently exceeds the plan's literal sketch (e.g. a split success/failure
once-handler design keyed on the correct topics, a real atlas-inventory
accommodation endpoint instead of a locally-derived guess). `go build ./...`
and `go test ./... -count=1` are clean with zero failures across all four
Go modules (`atlas-consumables`, `atlas-inventory`, `atlas-data`,
`atlas-channel`); `atlas-ui`'s `npm run build` and the item-feature Vitest
suite (15/15) both pass. No `// TODO`, stub, or 501 was found in the diff.
The four session fixes named in the audit brief are all present, correctly
wired, and internally coherent with each other. The only defect found is
**documentation drift**: `rollout.md` (authored before the mid-session scope
expansion to v72/v79/v92/jms) still says "do not patch v92/jms," but the
current seed templates already register `CharacterItemUseLotteryHandle` for
both — the runbook needs a follow-up edit before it is used operationally.
`tools/redis-key-guard.sh` reports repo-wide FAIL, but this traces to
pre-existing `go.mod`-needs-tidy breakage in four unrelated, untouched
services (atlas-mounts, atlas-mts, atlas-doors, atlas-monster-book) — the
four services task-131 actually touched are individually clean under the
guard.

## Task Completion — plan.md (backend)

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | atlas-data — parse per-entry Effect/worldMsg/period | DONE | `services/atlas-data/atlas.com/data/consumable/reader.go:167-177`, `rest.go:129-131`; `go test ./consumable/... ` passes |
| 2 | atlas-consumables — mirror reward fields + getters | DONE | `data/consumable/model.go:190,215-247` (`Rewards()`, getters, `RewardModelBuilder`); `rest.go:169-185` |
| 3 | atlas-packet — LotteryItemUse serverbound codec + fixtures | DONE | `libs/atlas-packet/inventory/serverbound/lottery_item_use.go` (const, struct, Encode/Decode); `lottery_item_use_test.go`; `go test ./inventory/serverbound/...` passes |
| 4 | atlas-consumables — CREATE_ASSET command + CREATED/CREATION_FAILED contract | DONE | `kafka/message/compartment/kafka.go:16,20,67-98` (`CommandCreateAsset`, `TransactionId`, `CreateResultEventBody`); `compartment/processor.go:28,68` (`RequestCreateItem`) |
| 5 | atlas-consumables — item-string data client | DONE | `data/itemstring/{rest,requests,processor}.go`; `GetName` implemented, with the plan's documented `identity()` fallback used instead of `model.Identity` (`processor.go:23,27`) |
| 6 | atlas-consumables — pure reward helpers | DONE | `consumable/reward.go:16` (`rollReward`, crypto/rand), `:70` (`rewardExpiration`), `:79` (`substituteWorldMsg`) |
| 7 | atlas-consumables — reward presentation event contract | DONE | `kafka/message/consumable/kafka.go` (`EventTypeRewardEffect`, `EventTypeRewardWon`, `ErrorTypeInventoryFull`); `consumable/producer.go` (`RewardEffectEventProvider`, `RewardWonEventProvider`) |
| 8 | atlas-consumables — RequestItemReward + ConsumeReward flow | DONE (materially superseded by session fixes, see below) | `consumable/processor.go:933` (`RequestItemReward`), `:1025` (`ConsumeReward`) — the shipped design splits confirm/fail across two topics (asset vs compartment status) rather than the plan's single-topic sketch; this is Session Fix #1, a real bug fix over the plan, see Task Completion table entry for the fixes below |
| 9 | atlas-consumables — command consumer arm for REQUEST_ITEM_REWARD | DONE | `kafka/consumer/consumable/consumer.go:34,67-`; `kafka/message/consumable/kafka.go:21` (`CommandRequestItemReward`) |
| 10 | atlas-channel — serverbound handler + REQUEST_ITEM_REWARD emit | DONE | `main.go:856` (`handlerMap[invsb.CharacterItemUseLotteryHandle]`); `socket/handler/character_item_use.go:53` (`CharacterItemUseLotteryHandleFunc`); `consumable/producer.go:33`, `consumable/processor.go:43` |
| 11 | atlas-channel — presentation consumer arms | DONE | `kafka/consumer/consumable/consumer.go:56,61` (handler registration), `:101` (inventory-full arm), `:158,183` (`handleRewardEffectConsumableEvent`, `handleRewardWonConsumableEvent`) |
| 12 | Packet matrix promotion | DONE | `docs/packets/audits/STATUS.md:649` — `LOTTERY_ITEM_USE_REQUEST` row shows ✅ for v72/v79/v83/v84/v87/v95/jms_v185; v92 correctly absent (no IDB) |
| 13 | atlas-configurations — seed-template handler entries (v83/84/87/95) | DONE | Confirmed present with correct opcodes 0x70/0x70/0x73/0x7C and `LoggedInValidator` in all four `template_gms_{83,84,87,95}_1.json` files |
| 14 | Rollout documentation | DONE, but now **stale** | `rollout.md` exists and is well-formed, but Step 3 ("v92/jms out of scope, do not patch") contradicts the current templates — see Skipped/Deferred section |
| 15 | Full verification gate + code review | PARTIAL | `go build`/`go test` clean in all four modules (this audit re-ran them); `redis-key-guard` clean for the four touched modules but the repo-wide script FAILs on unrelated modules (pre-existing, see below); `docker buildx bake` for the four services was **not** re-verified by this audit (not requested by the audit brief; see Action Items) |

**Completion Rate (plan.md):** 15/15 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 1 (Task 15 — verification gate; docker-bake step not exercised by this audit)

## Task Completion — plan-ui-possible-rewards.md

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Reward type + PossibleRewardsCard component (TDD) | DONE | `src/types/models/item.ts:88-95,111` (`RewardModel`, `ConsumableAttributes.rewards?`); `src/components/features/items/PossibleRewardsCard.tsx`; `__tests__/PossibleRewardsCard.test.tsx` — 15/15 tests pass. Final component diverges from the plan's draft per later refinement commits (`8d2014e17`, `50ad0de3a`): 3-decimal percentages instead of 1, raw weight badge dropped, and a responsive multi-column grid layout instead of a single column — all documented, deliberate UX refinements, not regressions |
| 2 | Wire the card into the item detail page + verify | DONE | `src/pages/ItemDetailPage.tsx:28,312-313` (`import`, conditional render on `a.rewards`); `npm run build` and `npx vitest run src/components/features/items/` both pass |

**Completion Rate (UI plan):** 2/2 tasks (100%)

## Session Fixes (verified against the audit brief)

| # | Fix | Verified | Evidence |
|---|-----|----------|----------|
| 1 | Reward success keyed off asset-status CREATED/QUANTITY_CHANGED, not compartment-status CREATED | YES | `kafka/message/asset/kafka.go:1-32` (new `StatusEvent`, `EnvEventTopicStatus = "EVENT_TOPIC_ASSET_STATUS"`); `kafka/once/asset/once.go:21` (`GrantConfirmedValidator`, matches `StatusEventTypeCreated` OR `StatusEventTypeQuantityChanged`); `kafka/consumer/asset/consumer.go:23` (new `InitConsumers`, `SetStartOffset(kafka.LastOffset)`); registered in `main.go:34` (`assetconsumer.InitConsumers`); `consumable/processor.go:1042,1070` (`ConsumeReward` registers on the asset topic, `grantRewardOnConfirmed` is the handler); `kafka/once/compartment/once.go:18-27` documents in-code why compartment-status CREATED is now dead for this purpose |
| 2 | atlas-inventory `CreateAssetAndEmit` re-emits CREATION_FAILED on failure | YES | `services/atlas-inventory/atlas.com/inventory/compartment/processor.go:993-1025` — captures the buffered rejection before the rolled-back tx discards it, then re-emits via a direct producer outside the tx (mirrors the pre-existing drop-pickup reject path) |
| 3 | Strict, merge-aware pre-roll accommodation check via new atlas-inventory endpoint | YES | `services/atlas-inventory/atlas.com/inventory/compartment/accommodation.go` (`CanAccommodate`, `accommodatesOne` — free-slot-or-full-merge logic mirroring `CreateAsset`); `accommodation_rest.go` (POST `/characters/{characterId}/inventory/accommodation`, registered in `resource.go:26`); atlas-consumables side: `inventory/accommodation.go` + `inventory/processor.go:CanAccommodate` calling it over REST; called from `consumable/processor.go:953-965` before reserve/roll. Commit `23e9d3c20`'s message explicitly confirms the old local `inventoryAccommodatesRewards` check and its test were removed (`reward_accommodation_test.go` deleted, confirmed in `git show --stat`) |
| 4 | No duplicate inventory-full message | YES | `consumable/processor.go:1095-1107` (`grantRewardOnFailed`) — its doc comment explicitly states it does NOT emit a consumable ERROR, because atlas-channel's generic `CREATION_FAILED` handler already renders the inventory-full status message; only the pre-roll path (`rewardInventoryFull`, `:987-992`) emits `ErrorTypeInventoryFull`, and that path has no corresponding CREATION_FAILED event to duplicate against |

All four fixes are present, correctly wired end-to-end (topic → validator → consumer registration → handler), and consistent with each other and with the in-code documentation explaining why each exists.

## Skipped / Deferred Tasks

**None outright skipped.** One documentation-drift issue found:

- **`rollout.md` is stale relative to the shipped scope.** It was authored at
  commit `faf24e9a6`, *before* the mid-session scope expansion (`1d09c2685
  docs(task-131): expand version scope after main merge`, `5f7ab1613
  feat(task-131): route lottery opcode in v72/v79/v92/jms + matrix`). Its
  Step 3 says: *"Do not add the `CharacterItemUseLotteryHandle` handler entry
  ... for v92 ... jms."* But the current seed templates already contain that
  entry for both:
  - `services/atlas-configurations/seed-data/templates/template_gms_92_1.json:160`
  - `services/atlas-configurations/seed-data/templates/template_jms_185_1.json:317`

  **Impact:** an operator following the current `rollout.md` verbatim would
  incorrectly believe v92/jms tenants have no handler entry in their seed
  template and skip a PATCH step that is actually unnecessary for *new*
  tenants (seed templates auto-apply at creation) but would still need the
  live-tenant-config PATCH treatment described in Step 2 for *existing*
  v92/jms tenants once support is intentionally extended to them. The
  practical risk is low (v92/jms are still real gaps — no IDA verification of
  the client-side body assumption exists for either, per `context.md` and the
  Global Constraints in `plan.md`), but the document's factual claim about
  what's in the templates is now wrong and should be corrected before this
  runbook is executed operationally.

## Build & Test Results

| Module | Build | Tests | Notes |
|--------|-------|-------|-------|
| services/atlas-consumables/atlas.com/consumables | PASS | PASS | `go build ./...` clean; `go test ./... -count=1` — all packages `ok` or `[no test files]`, zero FAIL |
| services/atlas-inventory/atlas.com/inventory | PASS | PASS | same; includes new `compartment/accommodation_test.go` |
| services/atlas-data/atlas.com/data | PASS | PASS | same; includes `consumable/reader_test.go` reward-field test |
| services/atlas-channel/atlas.com/channel | PASS | PASS | same; large module, zero FAIL across ~80 packages |
| libs/atlas-packet | PASS | PASS | `go test ./inventory/...` — `serverbound` and `clientbound` both `ok` |
| services/atlas-ui | PASS | PASS | `npm run build` (tsc -b + vite build) clean; `npx vitest run src/components/features/items/` — 3 files, 15/15 tests pass |

**Additional gates:**
- `GOWORK=off tools/redis-key-guard.sh` (repo-wide): **FAIL**, but traced to
  four *unrelated, untouched* modules — `atlas-mounts`, `atlas-mts`,
  `atlas-doors`, `atlas-monster-book` — each erroring with `./... matched no
  packages` because their `go.mod` needs `go mod tidy` under `GOWORK=off`
  (confirmed via `cd services/atlas-mounts/... && GOWORK=off go build ./...`
  → `go: updates to go.mod needed`). This is pre-existing environment
  breakage, not a regression from this branch (`go.work` is untouched in the
  diff). Running the guard binary directly against each of the four
  task-131-touched modules individually returns **exit 0** for all four —
  no raw keyed redis calls were introduced by this task.
- `tools/goroutine-guard.sh`: **PASS** (exit 0), full repo.
- `docker buildx bake atlas-{data,consumables,channel,configurations}`: **not
  run by this audit** — the audit brief specified `go build`/`go test` only;
  this is a documented gap against CLAUDE.md's mandatory bake step, not a
  claim that it would fail.

No `// TODO`, `FIXME`, stub, or `501` marker was found in the diff (`git diff
b1c50b67d..23e9d3c20 -- '*.go' | grep -n "TODO\|FIXME\|XXX\|501"` — the one
hit was a git blob hash containing the substring "501", not a code marker).

## Overall Assessment

- **Plan Adherence:** FULL (both plans, 17/17 tasks across the two documents)
- **Recommendation:** NEEDS_FIXES (one documentation correction) before
  READY_TO_MERGE; code and tests themselves are ready.

## Action Items

1. Update `rollout.md` Step 3 to reflect that v92 and jms seed templates now
   *do* contain the `CharacterItemUseLotteryHandle` entry (added in
   `5f7ab1613`), and clarify what that means for existing vs. new tenants of
   those versions (existing v92/jms tenants still need the live-config PATCH
   from Step 2 if/when v92/jms support is turned on operationally; new
   tenants get it automatically from the seed).
2. Before the branch is declared merge-ready per CLAUDE.md's mandatory gate,
   run `docker buildx bake atlas-data`, `atlas-consumables`, `atlas-channel`,
   and `atlas-configurations` from the worktree root — this audit did not
   execute that step (out of the scope requested), and CLAUDE.md treats it as
   non-optional for any branch touching those `go.mod` files.
3. (Optional, out of scope for task-131) The repo-wide
   `tools/redis-key-guard.sh` failure on `atlas-mounts`/`atlas-mts`/
   `atlas-doors`/`atlas-monster-book` (`go.mod` needs `go mod tidy` under
   `GOWORK=off`) is pre-existing and unrelated to this branch; worth a
   separate ticket so the repo-wide gate is green again, but it is not a
   blocker for task-131's own PR.
