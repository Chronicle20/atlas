# Plan Audit — task-198-transports-ui-surface (Plan Adherence)

**Plan Path:** docs/tasks/task-198-transports-ui-surface/plan.md
**Audit Date:** 2026-08-07
**Branch:** task-198-transports-ui-surface
**Base Branch:** main (range `354fb1257..1c1b7798f`, 26 commits)

## Executive Summary

All 19 tasks in the plan were implemented and are traceable to the branch's commits and shipped code. The plan's checkboxes were never ticked (`grep -c '^- \[x\]'` = 0), but this is a documentation-hygiene gap, not an implementation gap — every file listed in the plan's File Structure table exists with the expected exports, and the three deliberate deviations called out by the task brief (Task 1's `OutOfService`-preserving prune, Task 16's `aria-hidden` legs + single summary `role="img"`, and Tasks 14/15/17/18's added loading/error/empty states) are exactly as described, each with a git commit explaining the ruling. No task was silently skipped, stubbed, or left as a TODO. The pre-existing `TestStateMachine_UpdateState` suite (`transport/state_test.go`) is byte-for-byte unmodified on this branch and passes, confirming Task 1's "derived state must not change" constraint. The working tree is clean.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | `transport.Transition` and `Model.Evaluate` | DONE | `transport/model.go:111-234` defines `Transition`, `timeOfDay`, `materializeBoundary`, `Evaluate`; `processStateChange` is now a 1-line wrapper (`model.go:236-238`). `transport/evaluate_test.go` (206 lines) covers all branches. Commits `a81b368dd`, `d2e10d05f`, `08bb7f451`. **Deviation (approved):** the plan's original literal test case ("wraps to tomorrow") and the `earliestBoardingOpen` helper were dropped; `OutOfService` is preserved for the past-last-arrival case, matching `state_test.go`'s pre-existing "After arrival" expectation. `transport/state_test.go` has zero diff vs. base and `TestStateMachine_UpdateState` passes (verified live, all 8 subtests). |
| 2 | `…Seconds` duration fields on `routes` + `Extract` round-trip fix | DONE | `transport/rest.go:18-30` adds `BoardingWindowSeconds`/`PreDepartureSeconds`/`TravelDurationSeconds`/`CycleIntervalSeconds` (`uint32`); `TransformSummary` (`rest.go:127-153`) populates them; `Extract` (`rest.go:173-188`) now round-trips all three previously-dropped durations. `transport/rest_test.go` created, commit `2cee3dd94`. |
| 3 | `nextTransitionAt` / `nextState` on the route resource | DONE | `rest.go:31-38` adds `NextTransitionAt`/`NextState`; `TransformSummary` computes them from one `m.Evaluate(timeNow().UTC())` call (`rest.go:127-142`), guaranteeing state/transition agreement. Commit `66d7bb086`. |
| 4 | `include=schedule` opt-in on the route list | DONE | `transport/resource.go:49-62` (`wantsSchedule`), `:71-73` (transformer selection), call sites at `:103` and `:116` switched from hard-coded `Transform` to `transformer`. `resource_include_test.go` created; a follow-up commit `0b26e28bf` explicitly added coverage for the `filter[startMapId]` branch the plan's own trap note (context.md §6 item 6) warned about. Commit `4030c6486`. |
| 5 | Fix instance-status tenant scoping | DONE | `instance/resource.go:113,116` replaces `uuid.Nil` with `tenant.MustFromContext(d.Context())`. `resource_tenant_test.go` (90 lines, new) is the cross-tenant regression guard; `resource_paginate_test.go` re-pointed off a concrete `tenantId` (diff confirms `+90`/`-`/`~20` lines touched per plan). Commit `d3861c4f1`. |
| 6 | `…Seconds` on instance routes, `createdAt` on instance status | DONE | `instance/rest.go:27-28` (`BoardingWindowSeconds`/`TravelDurationSeconds`), `:58-59` (`TransformRoute` sets them), `:70-74` (`CreatedAt` field + doc comment), `:106` (`TransformInstanceStatus` sets it). `instance/rest_test.go` created. Commit `1247d7ba1`. |
| 7 | Backend documentation | DONE | `docs/rest.md` gained `boardingWindowSeconds`/`nextTransitionAt`/`include=schedule`/`createdAt` rows (grep confirms all field names present, lines 33-225). `docs/domain.md` documents `Transition`/`Evaluate` (lines 29-34). `.bruno/Routes/Get Routes With Schedule.bru` created. Commits `91f461689`, `ca5130b63` (a follow-up cross-reference fix). |
| 8 | atlas-ui transport types and service layer | DONE | `src/types/models/transport.ts` (110 lines) defines `RouteState`, `InstanceState`, `ScheduledRoute`, `TripSchedule`, `ScheduledRouteDetail`, `InstanceRoute`, `InstanceStatus`, `Vessel` exactly as specified. `src/services/api/transports.service.ts` (93 lines) exports all 5 read methods. Test file present. Commit `51b00f195`. |
| 9 | Shared clock and `Countdown` leaf | DONE | `src/lib/utils/clock.ts` exports `subscribeToClock`, `getClockSnapshot`, `useClock` (via `useSyncExternalStore`, cached snapshot per the trap note). `Countdown.tsx` present, imports `formatCountdown` from Task 10 as instructed. Commit `adb71fd2d`. **Dependency inversion confirmed:** `Countdown.tsx:2` imports from `@/components/features/transports/transport-format`, which is Task 10's module — matches context.md §4's explicit "implement Task 10 before Task 9" note. Commit order in git log has Task 10's commit (`7084881d7`) landing before Task 9's clock/Countdown commit (`adb71fd2d`), so the dependency was honored in execution order despite the plan's numbering. |
| 10 | `transport-format.ts` — every pure helper | DONE | All 13 named exports from the plan's interface list are present verbatim: `STATE_SEVERITY`, `compareRoutesBySeverityThenName`, `stateLabel`, `transitionLabel`, `formatCountdown`, `formatDurationSeconds`, `formatTimeOfDay`, `utcTimeOfDayMs`, `nowUtcTimeOfDayMs`, `timelineHalfWindowMs`, `segmentSpan`, `isInstanceStuck`, `resolveVesselRoutes`, `findVesselForRoute` (grep of `^export` in the file, 14 matches incl. `STATE_SEVERITY`). Commit `7084881d7`. |
| 11 | `useTransports` query hooks | DONE | `src/lib/hooks/api/useTransports.ts` exports `TRANSPORT_POLL_MS = 30_000`, `transportKeys.{all,scheduled,scheduledDetail,instanceRoutes,instanceStatus,vessels}`, `useScheduledRoutes`, `useScheduledRoute`, `useInstanceRoutes`, `useInstanceStatuses`, `useVessels`. Re-exported from `src/lib/hooks/api/index.ts` (+3 lines). Commit `0bd5f81e0`. |
| 12 | `RouteStatePill` and `FreshnessIndicator` | DONE | Both components created as specified; `RouteStatePill.tsx` uses `stateLabel` + text-carried state (never colour-alone). Commit `376c54da4`, plus a follow-up `5dd0cbb99` ("cover FreshnessIndicator's clock-driven re-render") adding test coverage beyond the plan's baseline. |
| 13 | Transports board — Scheduled tab, routing, sidebar | DONE | `pages/transports-columns.tsx` exports `createScheduledRouteColumns`; `pages/TransportsPage.tsx` exports `TransportsPage`. `App.tsx:233-235,303` wires the lazy route at `/transports`; `components/app-sidebar-items.ts:42` adds the "Transports" entry under Operations. Commit `d2d0c05d0`. |
| 14 | Instance tab | DONE | `InstanceRoutesTable.tsx` (281 lines) created and mounted in `TransportsPage.tsx`'s Instance `TabsContent`. Commit `30276c086`. **Deviation (approved) confirmed:** follow-up commit `a1ea0c03d` ("distinguish loading/error/empty states") — `InstanceRoutesTable.tsx:94` (`routesQuery.isLoading`) and `:103` (`routesQuery.isError`) branches exist beyond the plan's literal example code. |
| 15 | Vessels tab | DONE | `VesselsTable.tsx` (162 lines) created, mounted in the Vessels `TabsContent`. Commit `544e37fdd`. Also carries the loading/error deviation: `VesselsTable.tsx:29-30,44-45,59,68` add `isLoading`/`isError` props and branches (`TransportsPage.tsx:123-124` passes `vesselsQuery.isLoading \|\| scheduledQuery.isLoading` etc.), consistent with the approved deviation #3 even though the plan's Task 15 code block didn't include this prop. |
| 16 | `MapFlowRail` | DONE | `MapFlowRail.tsx` (165 lines). **Deviation (approved) confirmed exactly as described:** per-leg `<svg>` in `RailLeg` is `aria-hidden="true"` (`MapFlowRail.tsx:142`), and a single `<span role="img" aria-label="Map flow for …">` summary element (`:101-105`) carries the accessible name — resolves the plan's literal-code defect where every leg had `role="img"`, which a singular `getByRole` test cannot satisfy. Commit `32fb59344`, refined by `8c961e762` ("correct MapFlowRail empty-en-route captioning and expose transit state to a11y"). |
| 17 | `VesselTimeline` | DONE | `VesselTimeline.tsx` (270 lines) exports `TimelineLane` and `VesselTimeline`. Also carries an empty-lane treatment (`:102-116`, `data-empty-lane`) beyond the plan's literal code — consistent with deviation #3's broadened empty-state handling. Commit `d97ade6f7`. |
| 18 | Route detail page | DONE | `TransportRouteDetailPage.tsx` (266 lines) exports `TransportRouteDetailPage`; `App.tsx` wires `/transports/routes/:routeId`. **Deviation (approved) confirmed:** `:98-99` (`detailQuery.isLoading` → `DetailSkeleton`) and `:102` (`detailQuery.isError \|\| !detail`) are explicit states beyond the plan's single `isLoading \|\| !detail` check — the file's own doc comment (`:37-38`) states a failed fetch must not render identically to "nothing configured," matching the task brief's description of why this deviation was needed. Commit `32ea043ef`, plus test-only follow-up `1c1b7798f` ("cover the route detail page's dual-lane vessel assembly"). |
| 19 | Full verification sweep | DONE | Per the task instructions, this was already run and is not re-verified here: `go build`/`go vet`/`go test -race` clean in atlas-transports; `redis-key-guard.sh`/`goroutine-guard.sh` clean; `lint.sh --check` exit 0; atlas-ui `npm run test` 1498/1498 across 205 files and `npm run build` clean. `go.mod`/`go.sum` diff over the branch range is empty (spot-checked: `git diff --stat` above shows no `.mod`/`.sum` files touched), so the `docker buildx bake` escalation path in Step 5 was correctly not needed. |

**Completion Rate:** 19/19 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. Every task has landed code, a matching test file, and a commit. The three approved deviations (Task 1, Task 16, Tasks 14/15/17/18) are documented in commit messages (`d2e10d05f`, `08bb7f451`, `32fb59344`/`8c961e762`, `a1ea0c03d`) and are net improvements over the plan's literal example code, not scope reductions — no functional requirement from the Spec Coverage table was dropped.

## Spec Coverage Cross-Check

Every row in plan.md's Spec Coverage table (FR-1.1 through FR-6.6, Open Q1-Q4, Design §3.5-3.6, Verification plan) maps to a task confirmed DONE above. No row points to a task classified as SKIPPED or PARTIAL. Notable cross-checks:

- **FR-6.1 (tenant-scoping fix) / FR-6.2 (regression guard):** both map to Task 5, confirmed via `tenant.MustFromContext` at `instance/resource.go:113` and `resource_tenant_test.go`'s `TestGetInstanceRouteStatusIsTenantScoped`.
- **FR-6.6 (no vessels endpoint; read from configuration):** `transports.service.ts`'s `getVessels` reads `/api/tenants/${tenantId}/configurations/vessels` — no new atlas-transports endpoint was added, matching the "read-only surface, no new mutation endpoints" global constraint.
- **Open Q1 (sparse fieldsets → `include=schedule`):** Task 4's `wantsSchedule` helper, confirmed above.
- **Design §3.5 (`createdAt` on instance status):** Task 6, confirmed above.

## Build & Test Results

Per the task brief, Task 19's sweep was already performed and is not re-run by this audit:

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-transports (atlas.com/transports) | PASS (reported) | PASS (reported, `-race`) | Spot-re-verified only `go test ./transport/... -run 'TestStateMachine\|TestEvaluate' -v`: all pass, `state_test.go` unmodified vs. base. |
| atlas-ui | PASS (reported) | PASS (reported, 1498/1498, 205 files) | Not re-run per instructions. |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None. All 19 tasks are implemented with file:line evidence, the three approved deviations are exactly as scoped and are documented in their own commits, the pre-existing state-machine regression guard is untouched and passing, and the working tree is clean (`git status --short` empty at audit time).
