# Frontend Audit — task-198-transports-ui-surface

- **Audit Scope:** atlas-ui diff, commit range `354fb1257..1c1b7798f` (branch `task-198-transports-ui-surface`) — 31 new files + 3 small additions (`src/App.tsx`, `src/components/app-sidebar-items.ts`, `src/lib/hooks/api/index.ts`) under `services/atlas-ui/src/{types/models/transport.ts, services/api/transports.service.ts, components/features/transports/**, lib/hooks/api/useTransports.ts, lib/utils/clock.ts, pages/Transports*.tsx, pages/transports-columns.tsx}`.
- **Guidelines Source:** frontend-dev-guidelines skill + project-specific rules supplied in the audit brief (ONE CLOCK, tenant-scoped query keys, trip-schedule stale-date rule, FR-6.5 duration rule, MapCell string coercion, DataTable-has-no-sort rule, three-state loading/error/empty, read-only surface, non-colour-only accessibility).
- **Date:** 2026-08-07
- **Build:** PASS (per requester: `tsc -b && vite build` clean, pre-existing >500kB chunk warning for `ConversationEditorPanel` unrelated to this branch — not re-run by this audit)
- **Tests:** 1498/1498 passed across 205 files (per requester, not re-run in full). Targeted re-run of the 13 new transports test files during this audit: **109/109 passed** (`npx vitest run src/components/features/transports src/lib/hooks/api/__tests__/useTransports.test.tsx src/lib/utils/__tests__/clock.test.ts src/pages/__tests__/TransportsPage.test.tsx src/pages/__tests__/TransportRouteDetailPage.test.tsx src/services/api/__tests__/transports.service.test.ts`). One non-fatal React `act(...)` warning observed in `TransportRouteDetailPage.test.tsx` ("ticks the timeline's now marker off the shared clock") — tests still pass, noted as non-blocking test hygiene.
- **Overall:** NEEDS-WORK

## Build & Test Results

Verbatim from requester (not re-run in full by this audit, per instructions): `tsc -b && vite build` clean aside from the pre-existing chunk-size warning; `npm test` → 1498/1498 passed, 205 files.

This audit's own scoped re-run:
```
Test Files  13 passed (13)
     Tests  109 passed (109)
```

## File Inventory

- **Type:** `src/types/models/transport.ts` — `RouteState`, `InstanceState`, `ScheduledRoute(Attributes)`, `TripSchedule(Attributes)`, `ScheduledRouteDetail`, `InstanceRoute(Attributes)`, `InstanceStatus(Attributes)`, `Vessel(Attributes)`.
- **Service:** `src/services/api/transports.service.ts` — object-literal read-only adapter (`getScheduledRoutes`, `getScheduledRoute`, `getInstanceRoutes`, `getInstanceStatuses`, `getVessels`).
- **Hook:** `src/lib/hooks/api/useTransports.ts` — `transportKeys` factory + `useScheduledRoutes`, `useScheduledRoute`, `useInstanceRoutes`, `useInstanceStatuses`, `useVessels`.
- **Other (pure lib):** `src/lib/utils/clock.ts` — the single shared 1s clock (`useSyncExternalStore`).
- **Component:** `src/components/features/transports/Countdown.tsx`, `RouteStatePill.tsx`, `FreshnessIndicator.tsx`, `InstanceRoutesTable.tsx`, `VesselsTable.tsx`, `MapFlowRail.tsx`, `VesselTimeline.tsx`.
- **Other (pure lib):** `src/components/features/transports/transport-format.ts` — pure formatting/derivation helpers.
- **Page:** `src/pages/TransportsPage.tsx`, `src/pages/TransportRouteDetailPage.tsx`.
- **Other (columns):** `src/pages/transports-columns.tsx` — `createScheduledRouteColumns`.
- **Other (wiring):** `src/App.tsx` (+2 lazy routes), `src/components/app-sidebar-items.ts` (+1 nav entry), `src/lib/hooks/api/index.ts` (+1 re-export).
- **Tests:** one `__tests__` file per component/hook/service/page/lib file listed above (13 files, 109 tests) — see Testing Checklist.

## Anti-Pattern Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-01 | No `any` type | PASS | `grep -rn ': any\|as any'` across all in-scope non-test files returned zero matches. |
| FE-02 | No manual class concatenation | PASS | `grep -rn 'className={"'` returned zero matches; all conditional classes route through `cn()`, e.g. `src/components/features/transports/RouteStatePill.tsx:29`, `MapFlowRail.tsx:147`, `FreshnessIndicator.tsx:42`. |
| FE-03 | No direct API client calls in components | PASS | `grep -rn 'from "@/lib/api/client"'` across `components/features/transports` and the three page files returned zero matches; only `src/services/api/transports.service.ts:1` imports `api`. |
| FE-04 | No inline Zod schemas in components | PASS (N/A) | Zero `z.object(`/`z.string(` matches in scope; surface is read-only, no forms. |
| FE-05 | No spinners for content loading | PASS | `grep -rn 'animate-spin'` in scope returned zero matches. Loading states use text (`InstanceRoutesTable.tsx:100` "Loading instance routes…", `VesselsTable.tsx:65` "Loading vessels…") or `Skeleton` (`TransportRouteDetailPage.tsx:212-216`). |
| FE-06 | No hardcoded colors | **FAIL** | `src/components/features/transports/FreshnessIndicator.tsx:43` — `"h-2 w-2 rounded-full bg-emerald-500"`. `src/components/features/transports/VesselTimeline.tsx:32-34` — `fill-emerald-500/70`, `fill-amber-500/70`, `fill-sky-500/70`. Raw Tailwind palette classes, not semantic tokens (`bg-background`, etc.). Note: this matches a pervasive existing pattern elsewhere in the codebase (e.g. `src/components/features/npc/conversation/stateMeta.ts:54,59`, `ConversationCanvas.tsx:98`, `MapImageOverlay.tsx:212`) — it is not a novel regression introduced by this branch, but it is a literal violation of the documented rule. |
| FE-07 | No state mutation | PASS | Only mutation-shaped calls found are on freshly-spread local copies, not props/state/cache data: `transport-format.ts:134` (`[...boardingOpenMs].sort(...)`), `:138` (`gaps.push` on a locally-declared array), `:142` (`gaps.sort`), `TransportsPage.tsx:44` (`[...(scheduledQuery.data ?? [])].sort(...)` — spreads before sorting, does not mutate the React Query cache array). |
| FE-08 | No default exports for components | PASS | `grep -rn 'export default'` across all in-scope non-test files returned zero matches; all components/pages are named exports (e.g. `TransportsPage.tsx:22`, `TransportRouteDetailPage.tsx:41`), consistent with this project's "no default exports on pages" convention (`services/atlas-ui/CLAUDE.md`). |
| FE-09 | Tenant guard in hooks | PASS | Every hook in `useTransports.ts` calls `useTenant()` and gates on `enabled: !!activeTenant` — `useScheduledRoutes` (lines 50-58), `useScheduledRoute` (61-73, also gates on `!!routeId`), `useInstanceRoutes` (75-85), `useInstanceStatuses` (95-109), `useVessels` (111-121). |
| FE-10 | Tenant ID in query keys | PASS | `transportKeys` (useTransports.ts:29-41) — every branch takes `tenantId` and includes it in the tuple; callers pass `activeTenant?.id ?? "no-tenant"` (e.g. lines 51, 65, 77, 99, 113). |
| FE-11 | Error handling with `createErrorFromUnknown` | PASS (N/A) | No manual `.catch()` blocks exist in scope (`grep` returned zero); this is a pure React-Query read surface, and errors are surfaced via `isError`/`error` — consumed distinctly in `InstanceRoutesTable.tsx:103-114`, `VesselsTable.tsx:68-79`, `FreshnessIndicator.tsx:24-31`, and `TransportRouteDetailPage.tsx:102-117` (via `ErrorDisplay` + `isNotFoundError`). Codebase-wide, `createErrorFromUnknown` is used for manual promise chains only (one hook, `useCreateAndPollAccount.ts`), not for React Query hooks — consistent with prevailing convention. |
| FE-12 | JSON:API model shape | PASS | Every resource type in `src/types/models/transport.ts` follows `{ id: string, attributes: {...} }`: `ScheduledRoute` (37-40), `TripSchedule` (54-57), `InstanceRoute` (74-77), `InstanceStatus` (88-91), `Vessel` (107-110). `ScheduledRouteDetail` (59-62) is a composite return shape, not itself a resource. |
| FE-13 | Service extends `BaseService` (when applicable) | PASS | `transports.service.ts` uses the object-literal direct pattern, which is an established alternative already used by ≥5 other services in this codebase (`characterSkills.service.ts`, `commodities.service.ts`, `locations.service.ts`, `jobs.service.ts`, `services.service.ts`) — no validation/transformation need justifies skipping `BaseService`. |
| FE-14 | Query key factory uses `as const` | PASS | `useTransports.ts:29-41` — every key branch ends `as const`. |
| FE-15 | Forms use `react-hook-form` + `zodResolver` | PASS (N/A) | Read-only surface, no forms present. |
| FE-16 | Schema in `lib/schemas/` with inferred type | PASS (N/A) | No Zod schemas needed or present in scope. |

## Architecture Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| Route wiring | Lazy route + named export, uses `lazyWithReload` | PASS | `src/App.tsx` diff — `TransportsPage`/`TransportRouteDetailPage` both wrapped in `lazyWithReload(() => import(...))`, matching the project convention documented in `services/atlas-ui/CLAUDE.md` ("Use `lazyWithReload`, not bare `React.lazy`, for any new route page"). |
| Sidebar | Nav entry added | PASS | `src/components/app-sidebar-items.ts` diff — `{ title: "Transports", url: "/transports" }` added. |
| Hook re-export | `lib/hooks/api/index.ts` | PASS | `export * from "./useTransports";` added. |

## Project-Specific Rules

| # | Rule | Verdict | Evidence |
|---|------|---------|----------|
| 1 | One clock for the whole surface | PASS | `grep -rn 'setInterval\|setTimeout\|Date\.now('` across all in-scope non-test files returned matches only inside `src/lib/utils/clock.ts` (lines 19, 20, 23, 32-33). `VesselTimeline.tsx` takes `nowEpochMs: number` as a prop (line 23) and reads no clock of its own (confirmed no `useClock`/`Date.now` import in that file). `TransportRouteDetailPage.tsx:44` calls `const now = useClock();` and passes it at line 188 (`<VesselTimeline lanes={lanes} nowEpochMs={now} />`). |
| 2 | Tenant scoping in every React Query key | PASS | See FE-10 above — every `transportKeys` branch (useTransports.ts:29-41) includes `tenantId`, sourced as `activeTenant?.id ?? "no-tenant"`. |
| 3 | Trip-schedule timestamps carry a stale date | PASS | `formatTimeOfDay` (`transport-format.ts:100-105`) uses `getUTCHours()`/`getUTCMinutes()` exclusively. All `boardingOpen`/`boardingClosed`/`departure`/`arrival` reads in `VesselTimeline.tsx` route through `formatTimeOfDay` or `utcTimeOfDayMs` (lines 176, 194-198, 257-263). `grep` for `toLocaleString`/`toLocaleDateString`/`getDate()`/`getMonth()`/`getFullYear()`/`getHours()`/`getMinutes()` (non-UTC) across all in-scope files returned zero matches. `nextTransitionAt` (a real absolute instant) is correctly fed to `Countdown`/`Date.parse` instead (`Countdown.tsx:25`, `transports-columns.tsx:53`, `TransportRouteDetailPage.tsx:130`), not through `formatTimeOfDay`. |
| 4 | FR-6.5 duration rule | PASS | Every `...Seconds` field renders via `formatDurationSeconds` (`transport-format.ts:81-89`), called at `InstanceRoutesTable.tsx:191,194`, `VesselsTable.tsx:122`, `transports-columns.tsx:97`, `TransportRouteDetailPage.tsx:156,159,162,165`. `transport.ts:1-8` explicitly documents that legacy ns-valued fields (`cycleInterval`, `boardingWindow`, `travelDuration`) are deliberately *not* declared in the types, so nothing can read them by accident. The only inline `* 1000`/`/ 1000` occurrences are legitimate seconds↔ms conversions on already-`...Seconds`-suffixed values (`transport-format.ts:211` `maxLifetimeMs` from `boardingWindowSeconds`/`travelDurationSeconds`; `FreshnessIndicator.tsx:37` ms-age→seconds display) — not silent ns collapses. |
| 5 | `MapCell` takes `string`, route attrs are `number` | PASS | All 7 call sites wrap in `String(...)`: `InstanceRoutesTable.tsx:198,204`, `MapFlowRail.tsx:114,131`, `transports-columns.tsx:61,71`, `TransportRouteDetailPage.tsx:151`. |
| 6 | `DataTable` has no sorting row model | PASS | `src/components/data-table.tsx` confirmed to have no `getSortedRowModel`/`SortingState`/`enableSorting` (grep returned zero matches). `TransportsPage.tsx:42-46` computes `routes` via `useMemo` + `.sort(compareRoutesBySeverityThenName)` *before* handing off to `DataTable`/`createScheduledRouteColumns`. `InstanceRoutesTable.tsx` and `VesselsTable.tsx` render straight from query/prop data with no additional in-table sort dependency. |
| 7 | Loading / error / empty are three visually distinct states everywhere reachable | **FAIL** | `InstanceRoutesTable.tsx:94-124` and `VesselsTable.tsx:59-89` correctly branch `isLoading` → `isError` → `length === 0` → data, each with distinct copy/iconography. `TransportRouteDetailPage.tsx:98-117` correctly branches `isLoading` (skeleton) → `isError`/`!detail` (`ErrorDisplay`, further split not-found vs generic error) → success. **However**, `TransportsPage.tsx`'s Scheduled tab (the *default* tab, lines 97-107) passes `data={routes}` straight into `<DataTable>` with no `isLoading`/`isError` branching at all; `scheduledQuery.isLoading`/`.isError` are read elsewhere on the page (`FreshnessIndicator` props at 79-80, and handed to `VesselsTable` at 123-124) but never consulted for the Scheduled tab's own content area. `src/components/data-table.tsx:143-171` (unrelated pre-existing shared component, not itself in scope) renders the identical "No results." row regardless of whether `data` is empty because of a genuine empty configuration, a fetch error, or the initial loading tick before the first response lands — the only differentiator is the small `FreshnessIndicator` text in the page header (`TransportsPage.tsx:77-81`), which a user scanning the primary table body could easily miss. A failed initial fetch of scheduled routes therefore renders visually as "no routes configured" in the main content area, which is exactly the anti-pattern this rule prohibits. Not exercised by `TransportsPage.test.tsx` either — no test asserts an error-state render for the Scheduled tab. |
| 8 | Read-only surface (no mutations) | PASS | `grep -rn 'useMutation\|api\.post(\|api\.patch(\|api\.delete(\|api\.put('` across all in-scope non-test files returned zero matches. No create/edit/delete buttons or forms anywhere in the 7 components + 2 pages. |
| 9 | Accessibility — meaning never by colour/position alone | PASS | `RouteStatePill.tsx:26-33` always renders `stateLabel(state)` text inside the `Badge`, never colour alone. `FreshnessIndicator.tsx:17-51` pairs the coloured dot with explicit text ("Updated Xs ago" / "Stale — last refresh failed" / "Loading…"). `MapFlowRail.tsx` — the in-transit highlight (colour + stroke-width on a decorative, `aria-hidden` SVG) is backed by a `transitClause` appended to the rail's `role="img"` accessible name (lines 90-97, 101-105) documenting which stops are currently traversed. `VesselTimeline.tsx` — the whole strip is one `role="img"` SVG whose `aria-label` (`ariaLabel`, lines 82-88, built via `laneAriaPhrase`, 186-202) spells out every trip's board/close/depart/arrive times and the "now" position per lane, so the colour-coded segments are not the sole channel. `InstanceRoutesTable.tsx:271-278` pairs the "stuck" state with an `AlertTriangle` icon and "Approaching stuck timeout" text, not colour alone. |

## Testing Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-17 | Tests exist for changed components | PASS (with one WARN) | One `__tests__` file per component/hook/service/page/lib-module: `Countdown.test.tsx`, `FreshnessIndicator.test.tsx`, `InstanceRoutesTable.test.tsx`, `MapFlowRail.test.tsx`, `RouteStatePill.test.tsx`, `VesselsTable.test.tsx`, `VesselTimeline.test.tsx`, `transport-format.test.ts`, `useTransports.test.tsx`, `clock.test.ts`, `transports.service.test.ts`, `TransportsPage.test.tsx`, `TransportRouteDetailPage.test.tsx` — 13 files, 109 tests, all passing on a scoped re-run. **WARN:** `src/pages/transports-columns.tsx` (`createScheduledRouteColumns`) has no dedicated unit test file; it is only indirectly exercised through `TransportsPage.test.tsx` (which does cover the name link, state pill, next-change em-dash, and start/destination map cells — `TransportsPage.test.tsx:97-151` — but not the `vessel` or `cycleInterval` columns). Non-blocking: coverage exists, just not isolated/complete for every column. |
| FE-18 | Mocks updated when services changed | PASS (N/A) | No `__mocks__/` directory convention exists anywhere in this codebase (`find -type d -name __mocks__` returns nothing); all service mocking is done inline via `vi.mock(...)`, which is present and correctly shaped in every consuming test file (e.g. `TransportsPage.test.tsx:37-45`). |

## Summary

### Blocking (must fix)

- **Rule 7 (loading/error/empty three-state)** — `TransportsPage.tsx`'s Scheduled tab (lines 97-107) does not branch on `scheduledQuery.isLoading`/`scheduledQuery.isError` before handing `data` to `DataTable`. A fetch failure on the default tab renders the same "No results." row as a genuinely empty route list; only the header `FreshnessIndicator` distinguishes them. Bring the Scheduled tab's own content area in line with the distinct-state treatment already implemented in `InstanceRoutesTable.tsx` and `VesselsTable.tsx` (e.g. render an inline error/loading state above or instead of `DataTable` when `scheduledQuery.isLoading`/`isError` is true), and add a test asserting the error case is visually distinguishable.

### Non-Blocking (should fix)

- **FE-06 (hardcoded colors)** — `FreshnessIndicator.tsx:43` (`bg-emerald-500`) and `VesselTimeline.tsx:32-34` (`fill-emerald-500/70`, `fill-amber-500/70`, `fill-sky-500/70`) use raw Tailwind palette classes instead of semantic CSS variables. This mirrors a widespread pre-existing pattern elsewhere in the codebase (npc conversation editor, map overlay), so it's not a novel regression, but it is a literal violation of the documented styling rule and worth a follow-up sweep rather than blocking this PR alone.
- **FE-17 (test coverage)** — `transports-columns.tsx`'s `vessel` and `cycleInterval` columns have no direct assertion in any test file; only indirectly plausible via `TransportsPage.test.tsx`'s existing coverage of other columns.
- Minor test hygiene: `TransportRouteDetailPage.test.tsx` produces a non-fatal `act(...)` warning on the "ticks the timeline's now marker off the shared clock" test (clock tick not wrapped in `act`). Tests still pass; recommend wrapping the tick advance in `act()` to silence it.
