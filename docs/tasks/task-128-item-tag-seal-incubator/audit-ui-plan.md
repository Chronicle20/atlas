# Plan Audit — task-128-item-tag-seal-incubator (UI-surfacing plan)

**Plan Path:** docs/tasks/task-128-item-tag-seal-incubator/plan-ui-surfacing.md
**Audit Date:** 2026-07-16
**Branch:** task-128-item-tag-seal-incubator
**Base Branch:** main (diff range `a4aa4a73b54e382a2707ed173f469ae803abb8df..60880a28fe67deeb51f3b21df2819d2f0497ec8c`)

Note: every checkbox in `plan-ui-surfacing.md` is unticked (`- [ ]`). Per instructions, checkbox state was ignored; every finding below is against the actual committed diff (`git diff a4aa4a73b..60880a28f`) and the resulting files on disk.

## Executive Summary

All 15 tasks were faithfully implemented — nothing was silently skipped or deferred. Phase A (incubator-rewards admin page), Phase B (inventory tag/seal indicators), and Phase C (MTS listing owner/flag threading) all match the plan's interfaces and, in several places (Task 4's test suite, Task 13's owner-threading), exceed the plan's minimum bar by also covering the two secondary snapshot-capture sites (holding chain, saga compensator) that the plan's Task 10 investigation flagged as "in scope if item-tag owner must survive every custody hop." Verification was independently re-run in this audit: `go build/vet/test -race` is clean on all three touched Go modules (atlas-mts, atlas-saga-orchestrator, libs/atlas-saga), `docker buildx bake atlas-mts` succeeds, the redis-key-guard and goroutine-guard are clean, `npm run build` succeeds, and all 975 Vitest tests across 123 files pass. The one real gap: the plan's own Task 15 Step 1 constraint of "no new lint errors" is not met — the branch introduces 5 new `@typescript-eslint/no-explicit-any` errors, all inside two new test files (`incubator-rewards.service.test.ts`, `useIncubatorRewards.test.tsx`) whose `as any` mock-casting pattern was copied verbatim from the plan's own prescribed test code. CI does not block on this (the lint step runs with `continue-on-error: true`), and the PR (#909) is green on all 73 checks, but it is a genuine local-verification gap the plan's own global constraints call out.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | incubator-rewards service | DONE | `services/atlas-ui/src/services/api/incubator-rewards.service.ts` — matches plan's interface exactly (`INCUBATOR_REWARDS_RESOURCE_TYPE`, `list/create/update/remove/seed`). Test file matches plan's test verbatim. |
| 2 | useIncubatorRewards hook | DONE | `services/atlas-ui/src/lib/hooks/api/useIncubatorRewards.ts` — matches plan exactly (`incubatorRewardsKeys`, `useIncubatorRewards`, 4 mutation hooks with `onSettled` invalidation). |
| 3 | incubator-rewards zod schema | DONE | `services/atlas-ui/src/lib/schemas/incubator-rewards.schema.ts` — `incubatorRewardSchema`/`IncubatorRewardFormData`; same validation rules as plan (int + positive on all 3 fields), just with added JSDoc. |
| 4 | incubator-rewards page (table + dialog + seed) | DONE | `services/atlas-ui/src/pages/tenants-incubator-rewards-form.tsx` (306 lines) + `TenantsIncubatorRewardsPage.tsx`. All plan-pinned behaviors present: `useParams` tenantId (`:61`), `totalWeight` reduce (`:73`), chance-% cell (`:195-196`), Add/Seed buttons, single Add/Edit `Dialog` with zod-resolved form, per-row Edit/Delete with `AlertDialog` confirms. Deviates from the plan's suggested import paths (`ItemNameCell` from `@/components/item-name-cell` not `marketplace-columns`; `createErrorFromUnknown` from `@/types/api/errors` not `@/lib/api/errors`) — plan explicitly said "grep to confirm path," both paths exist and are used consistently between component and test mock. Test suite (7 cases) exceeds the plan's 2-case sketch (adds edit-prefill, submit, seed-confirm, delete-confirm, empty-state cases). |
| 5 | wire route + nav link | DONE | `services/atlas-ui/src/App.tsx:66` (lazy import), `:130` (route `/tenants/:id/incubator-rewards`); `TenantDetailLayout.tsx:20` (`{ title: "Incubator Rewards", href: ... }`, directly after MTS Configuration at `:19`, as specified). |
| 6 | type `owner` field + asset-flags util | DONE | `inventory.service.ts:69` (`owner: string;` after `ownerId`); `src/lib/utils/asset-flags.ts` — `FLAG_LOCK=0x01` (verified against `libs/atlas-constants/asset/flag.go:6` `FlagLock Flag = 0x01`), `ZERO_DATE`, `isSealed`, `isTagged`, all matching plan signatures exactly. |
| 7 | tooltip owner + seal lines | DONE | `AssetTooltipContent.tsx:6` (import), `:172-195` — owner line gated on `isTagged`, seal block with `SEALED`/`SEALED UNTIL:` gated on `isSealed`, EXPIRES only rendered in the `else` branch (non-sealed). Matches plan's replacement block exactly, including the "sealed never reads EXPIRES" semantics from the global constraints. |
| 8 | EquipmentCell badges + gold ring | DONE | `EquipmentCell.tsx:1,9,22,38-43` — `ring-1 ring-amber-400/60` on seal, `Tag`/`Lock` icons with `data-testid="tag-icon"`/`"seal-icon"` positioned top-right/bottom-right over a `relative` image wrapper. Matches plan's exact JSX. |
| 9 | InventoryCard badges + gold ring | DONE | `InventoryCard.tsx:9,193,270-275` — ring on `Card` root when `isSealed(asset)`, same `Tag`/`Lock` badges in the success-render branch. Badges only render in the "Success State" branch (not the loading/error/skeleton branches) — a reasonable scoping choice since there's no image container to badge in those states; not a plan deviation since the plan only specified "mirror Task 8." |
| 10 | verify flag carry + locate snapshot capture site | DONE | `docs/tasks/task-128-item-tag-seal-incubator/mts-owner-flag-notes.md` — thorough 10-step trace from `expandTransferToMts` (`saga/processor.go:1600`, `Flags: foundAsset.Flag`) through to `listing/rest.go`, correctly concludes flags already carry end-to-end and only `owner` needs threading, plus documents 2 secondary capture sites (holding chain, saga compensator) as optional-scope follow-ups. Committed as its own commit (`7ab93cbd9`). |
| 11 | add `owner` to atlas-mts listing model | DONE | `listing/model.go:81,139` (field + `Owner()` getter), `listing/builder.go:50,230-233,343` (field, `SetOwner`, `Build()` wiring), `listing/entity.go:78` (`Owner string \`gorm:"column:owner;not null;default:''"\``). Test: `listing/builder_test.go` `TestBuilder_SetOwnerRoundTrip` (new). |
| 12 | expose `owner` on listing REST model | DONE | `listing/rest.go:46` (`Owner string \`json:"owner"\``), `:158` (`Owner: m.Owner()` in `Transform`). Test: `listing/rest_test.go` `TestTransformOwner` (new file, matches plan's test). |
| 13 | capture owner (+flag) at listing creation | DONE, exceeds plan scope | Full pipeline threaded per the notes' checklist: `libs/atlas-saga/payloads.go:663` → `saga/processor.go:1601` (`expandTransferToMts`) → `saga/handler.go:2059` → `mts/processor.go:54` → `mts/producer.go:51` → both `AcceptToMtsListingCommandBody` wire structs (orchestrator `kafka/message/mts/custody/kafka.go:75`, atlas-mts `kafka/message/custody/kafka.go:96`, kept field-identical) → `consumer.go:127` → `processor_custody.go` `AcceptRequest`/`Accept()`/`SettleMove()` (`:58,161,332`) → listing `provider.go:277`/`administrator.go:126`. Additionally covers the two "if in scope" secondary sites the notes flagged: the full `holding.*` chain (`builder.go`, `model.go`, `entity.go`, `provider.go`, `administrator.go`, `rest.go`) and `listing/processor.go:632` (`transitionToSellerHolding`), plus `saga/compensator.go:1579` (`assetDataFromMtsListingSnapshot`). Tests: `builder_test.go`, `rest_test.go`, `consumer_test.go` (asserts owner survives the wire hop into the persisted listing row), `mts_expansion_test.go` (asserts saga expansion captures `foundAsset.Owner`). |
| 14 | render owner + lock on Marketplace rows | DONE | `mts-listings.service.ts:55` (`owner: string;` on `MtsListingAttributes`); `MarketplacePage.tsx` new exported `ListingItemCell` component (`:287-318`) rendering `Tag`+owner text when tagged and `Lock` (`data-testid="seal-icon"`) when `flags & FLAG_LOCK`, imported `FLAG_LOCK` from the Task-6 util rather than redefining it. Test: `MarketplacePage.test.tsx` — 3 new cases (`seal-icon` only, `tag-icon`+owner text, neither) matching the plan's spec. |
| 15 | full verification + review | PARTIAL | See Build & Test Results and Action Items below. Steps 1–3 and 5 pass; Step 4 (code review) is only partially evidenced. |

**Completion Rate:** 14/15 tasks fully DONE, 1/15 (Task 15) PARTIAL — 93%
**Skipped without approval:** 0
**Partial implementations:** 1 (Task 15, code-review sub-step)

## Skipped / Deferred Tasks

None of Tasks 1–14 were skipped or deferred. Task 15 ("full verification + review") is the only PARTIAL:

- **Step 1 (atlas-ui build + test, "no new lint errors"):** build and tests pass cleanly (see below), but `npm run lint` shows 5 NEW errors introduced by this branch — all `@typescript-eslint/no-explicit-any` inside two new test files: `src/services/api/__tests__/incubator-rewards.service.test.ts:14,20,30` (3) and `src/lib/hooks/api/__tests__/useIncubatorRewards.test.tsx:21,29` (2). These are not files that existed before the branch, so any lint error inside them is by definition new. The `as any` casts are copied verbatim from the plan's own Step-1 test code blocks for Tasks 1 and 2 (e.g. plan line 51: `(api.getList as any).mockResolvedValue([])`), so the implementer followed the plan exactly — the plan itself is the source of the pattern. Impact: low. CI's "Test UI" job runs `npm run lint` with `continue-on-error: true` (`.github/actions/node-test/action.yml`), so this does not block merge or fail CI (confirmed: PR #909 shows 73/73 checks green, "Test UI" included). It is, however, a real miss against the plan's own explicit global constraint ("gate on `npm run build` + `npm run test` + no new lint errors") and against project CLAUDE.md's atlas-ui section. All other new/touched files (EquipmentCell, InventoryCard, AssetTooltipContent, asset-flags, MarketplacePage, mts-listings.service, inventory.service, TenantDetailLayout, App.tsx) introduce zero new lint findings.
- **Step 4 (code review):** a backend-guidelines-style review of the Go diff exists at `docs/tasks/task-128-item-tag-seal-incubator/audit-ui-backend.md` (dated 2026-07-16, **uncommitted/untracked** — `git status` shows it as an untracked file, not part of any commit on this branch) and concludes NEEDS-WORK with exactly one Minor, non-blocking finding (DOM-20: two new Go tests are not table-driven, consistent with the surrounding file's existing style). No frontend-guidelines-reviewer output for the ui-surfacing plan was found anywhere in the task folder, and no plan-adherence audit against `plan-ui-surfacing.md` existed prior to this report (this document is the first). So "dispatches backend-guidelines + frontend-guidelines + plan-adherence" (the plan's own description of what Step 4 should do) is only 1-of-3 evidenced, and even that 1 is not committed to the branch.

Neither gap blocks correctness — the underlying feature code is complete and tested — but they are real process gaps against the plan's own Task 15 checklist.

## Build & Test Results

| Service / Module | Build | Vet | Test (`-race`) | Notes |
|---|---|---|---|---|
| services/atlas-mts/atlas.com/mts | PASS | PASS | PASS | All packages `ok`, race-clean, re-run in this audit. |
| services/atlas-saga-orchestrator/atlas.com/saga-orchestrator | PASS | PASS | PASS | All packages `ok`, race-clean, re-run in this audit. |
| libs/atlas-saga | PASS | PASS | PASS | `ok github.com/Chronicle20/atlas/libs/atlas-saga`, re-run in this audit. |
| `docker buildx bake atlas-mts` | PASS | — | — | Re-run in this audit; image `atlas-mts:local` built successfully (mostly cache-hit layers, no new COPY-line gaps). |
| `tools/redis-key-guard.sh` | PASS | — | — | Re-run; exit 0, no new violations. |
| `tools/goroutine-guard.sh` | PASS | — | — | Re-run; exit 0, no new violations. |
| services/atlas-ui (`npm run build`) | PASS | — | — | `tsc -b && vite build` succeeds; only pre-existing chunk-size warnings (unrelated to this branch). |
| services/atlas-ui (`npm run test`) | — | — | PASS | 975/975 tests pass across 123 files, re-run in this audit. |
| services/atlas-ui (`npm run lint`) | — | — | **5 new errors** | See "Skipped / Deferred Tasks" above — all in 2 new test files, CI-non-blocking (`continue-on-error: true`), pattern copied from plan's own test code. |

CI (PR #909, commit `60880a28f`): all 73 checks report `SUCCESS`, including `Test UI`, `Redis Key Guard`, `Goroutine Guard`, `Service Registration Guard`, and all 4 `Build Docker (bake)` shards.

## Overall Assessment

- **Plan Adherence:** MOSTLY_COMPLETE
- **Recommendation:** READY_TO_MERGE (with a minor, non-blocking follow-up recommended)

## Action Items

1. (Optional, low priority) Fix the 5 new `@typescript-eslint/no-explicit-any` lint errors in `incubator-rewards.service.test.ts` and `useIncubatorRewards.test.tsx` by typing the `vi.mock` return values instead of casting `as any`, to satisfy the plan's own "no new lint errors" constraint. Not CI-blocking; can be a fast-follow.
2. Commit or discard `docs/tasks/task-128-item-tag-seal-incubator/audit-ui-backend.md` (currently untracked) so the backend-guidelines review evidence isn't lost when the worktree is cleaned up.
3. Before closing out Task 15, run (or confirm someone has run) the `frontend-guidelines-reviewer` agent against the atlas-ui diff — it has not been evidenced anywhere in the task folder for the ui-surfacing plan specifically.
4. This report (`audit-ui-plan.md`) now serves as the plan-adherence-reviewer evidence for Task 15 Step 4's third leg; no further action needed there.
