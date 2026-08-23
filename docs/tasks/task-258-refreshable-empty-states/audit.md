# Plan Audit — task-258-refreshable-empty-states

**Plan Path:** docs/tasks/task-258-refreshable-empty-states/plan.md
**Audit Date:** 2026-08-21
**Branch:** task-258-refreshable-empty-states
**Base Branch:** main (merge-base 1461bfc96)

## Executive Summary

All 10 plan tasks landed, each as its own commit, and every commit's diff matches
the plan's prescribed implementation almost verbatim — including the four design
decisions the audit brief flagged for extra scrutiny (D1, D7, D9, D10). The
FR-6.5 call-site sweep artifact was independently re-derived from source via
`grep` and matches the tree exactly, including a self-corrected gap (`fbb9736f7`
added two missed `EmptyState` sites the first sweep pass omitted). An isolated
scratch worktree at `e209ae745` built clean (`npm run build`, `tsc -b` + `vite
build`) and ran the full suite green: 259 test files / 2131 tests passed,
matching the branch's own recorded flagless `verify.sh` result. No skipped or
partial tasks found. The plan.md checkboxes were never ticked (`- [ ]`
throughout), which is a documentation-hygiene gap, not an implementation gap —
every task has direct commit and test evidence of completion.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | `useGridRefresh` — structural `RefreshableQuery` + `lastUpdatedAt` (D1, D2) | DONE | `bd8033925`; `services/atlas-ui/src/lib/hooks/useGridRefresh.ts` — hand-written `RefreshableQuery` interface (no `Pick<UseQueryResult,...>`), `lastUpdatedAt = stamps.length ? Math.min(...stamps) : null` computed inline, no `useMemo`. All 5 planned min/null/empty/single test cases present in `useGridRefresh.test.ts`, including the min-vs-max regression case `[5000,9000]→5000`. |
| 2 | `EmptyState` — refresh button, action row, caption (D3, D4) | DONE | `13fb57f35`; `EmptyState.tsx` adds action row (`action` then `onRefresh`, `variant="outline"`, `data-testid="empty-state-refresh"`, `RefreshCw`/`animate-spin`/`disabled`/`aria-busy`), caption guarded `lastUpdatedAt != null && lastUpdatedAt > 0` with ISO `title`. New `EmptyState.test.tsx` covers all 16 planned subtests verbatim (refresh presence/disable/click, action precedence + order + outline-class equality, caption presence/absence/rerender). |
| 3 | `DataTableWrapper` forwards refresh props (D5) | DONE | `2bbbea51a`; `lastUpdatedAt?: number \| null` added, empty branch uses the conditional-spread idiom for all three props exactly as specified in D5 (no `emptyState.onRefresh` nesting). New `DataTableWrapper.test.tsx` (8 cases) covers the planned matrix. |
| 4 | Pass `lastUpdatedAt` through eleven wired pages | DONE | `b749bf064`; all 11 named files present (`AccountsPage`, `BansPage`, `CharactersPage`, `CouponsPage`, `GuildsPage`, `MapsPage`, `MerchantsPage`, `QuestsPage`, `ReportsPage`, `ServicesPage`, `TemplatesPage`); each destructures `lastUpdatedAt` and forwards it as a direct prop (no conditional spread needed, matching the plan's rationale that the type is `number \| null`). |
| 5 | `EventDefinitionsPage`/`RewardPoolsPage` — one control per view (D9) | DONE | `350430ab1`; `EventDefinitionsPage.tsx` header button + unused `RefreshCw`/`cn` imports removed, three props passed to wrapper. `RewardPoolsPage.tsx` tabs-row button removed, `poolTable`'s wrapper gets the three props, Global tab gains its own equivalent icon button (imports retained, confirmed still used at line 176/39). `RewardPoolsPage.test.tsx` replaced the single "renders a single refresh control" case with two `getAllByRole(...).toHaveLength(1)` cases — one per pool tab and one per Global tab, exactly as specified. |
| 6 | `tenant-context` + `TenantsPage` shim/timestamp/skeleton guard (D6) | DONE | `5192874e3`; `TenantRefreshResult` type added, `refreshTenants` returns `{ok:true}`/`{ok:false,error}`, `tenantsUpdatedAt` stamped after all three `setTenants(data)` call sites (initial load, `refreshTenants`, `refreshAndSelectTenant`). `TenantsPage.tsx` builds the `RefreshableQuery` shim with `result && result.ok === false` falsy-safe handling, skeleton guard changed to `loading && !isRefreshingTenants`. New `describe("TenantsPage empty-state refresh")` block has all 6 planned cases including the undefined-result-as-success case and the skeleton-vs-grid-during-refresh case. Two later commits (`5d42a44c8`, `51c453c84`) fixed a flaky-timer issue in this same test file with documented diagnoses (`bug-tenants-suite-failure.md`); not scope creep. |
| 7 | `GuildDetailPage` — first-time refresh wiring | DONE | `511ef9f55`; `useGridRefresh([guildQuery, tenantConfigQuery])` called unconditionally before the `loading`/`error` early returns (verified by reading the surrounding source — hook order preserved), three props forwarded to the wrapper. No new test suite added; explicitly recorded as D11 in context.md ("wiring is a verbatim copy of GuildsPage.tsx:65-70... covered at the contract level"), matching plan Task 7 Step 2's own statement that no suite is expected. |
| 8 | `EventOccurrencesPage` — direct `EmptyState` props (D8) | DONE | `f16ab125a`; `lastUpdatedAt` destructured, all three props passed directly to the hand-built `<EmptyState>` (not via wrapper, matching D8). New test asserts a second `getOccurrences` call after clicking the empty-state refresh button. |
| 9 | `TransportsPage` — `EmptyState` on scheduled tab, no `lastUpdatedAt` (D7) | DONE | `112d2cf14`; `<p>` empty branch replaced with `<EmptyState title=... onRefresh={onRefresh} isRefreshing={isRefreshing} />` — `lastUpdatedAt` is **not** passed, confirmed by direct read of the JSX. Populated-branch refresh (`onRefresh={() => void scheduledQuery.refetch()}`) at line 133 is untouched and does not route through the toasting hook, confirming D10. New test asserts the header `FreshnessIndicator` remains the sole freshness readout (`getAllByText(/updated \d+s ago/i)).toHaveLength(1)`). |
| 10 | Record the FR-6.5 call-site sweep | DONE | `560c8b996` + fix `fbb9736f7`; `call-site-sweep.md` re-derived independently via `grep -rln "<DataTableWrapper"/"<EmptyState"/"useGridRefresh("` against the finished tree — all 15 `DataTableWrapper` page call sites confirmed to pass all three props by direct `awk`/`grep` inspection; the two non-wrapper grids (`EventOccurrencesPage` all three, `TransportsPage` two with `lastUpdatedAt` omitted) confirmed; the comment-only `event-occurrences-columns.tsx` reference and the `AccountsPage` ban-status Q3 gap confirmed against source. The doc's initial pass missed `PresetLibrary.tsx`/`CharacterTemplatesEditor.tsx` `<EmptyState>` sites; `fbb9736f7` corrected this before landing — confirmed both have no `useGridRefresh` call, so exclusion is correct. |

**Completion Rate:** 10/10 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. All ten tasks have direct commit evidence matching the plan's prescribed
implementation, including the four flagged design decisions:

- **D7** (`TransportsPage`): `lastUpdatedAt` is verifiably absent from the
  `EmptyState` call in `TransportsPage.tsx` (only `title`, `onRefresh`,
  `isRefreshing` are passed), and the header `FreshnessIndicator` remains the
  sole freshness readout, confirmed by both source and the new test's
  `toHaveLength(1)` assertion.
- **D9** (`EventDefinitionsPage`/`RewardPoolsPage`): bespoke header refresh
  buttons removed from both pages' grid views; `RewardPoolsPage`'s Global tab
  (a hand-built `<Table>`, not a grid) retains its own equivalent control; the
  updated test enforces exactly one refresh control per visible tab.
- **D10** (`TransportsPage` data branch): `onRefresh={() => void
  scheduledQuery.refetch()}` on the populated branch is unchanged — it does
  not route through `useGridRefresh`'s toasting `onRefresh`, confirmed by
  direct source read.
- **D1** (structural `RefreshableQuery`): the `Pick<UseQueryResult,...>` alias
  was fully replaced with the hand-written interface specified in the design,
  not merely widened.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-ui | PASS | PASS | Run in an isolated scratch worktree at `e209ae745` (`npm ci && npm run build` → `tsc -b && vite build` succeeded; `npm test -- --run` → 259 test files / 2131 tests passed). Matches the branch's own recorded flagless `tools/verify.sh` result; not re-run per task instructions. |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None required. Optional housekeeping: the plan.md task checkboxes (`- [ ]`)
were never ticked to `- [x]` despite all steps being implemented and verified;
tick them for documentation hygiene, but this has no bearing on code
correctness or completeness.
