# task-258 — Implementation Context

Companion to `plan.md`. Everything here was read out of the tree at plan time;
where the design and the code disagreed, the code won and the resolution is
recorded below.

## Key files

| File | Role |
|---|---|
| `services/atlas-ui/src/lib/hooks/useGridRefresh.ts` | The refresh contract. Owns refetch fan-out, `isFetching`-derived in-flight state, and the success/error toast. Gains `dataUpdatedAt` and `lastUpdatedAt`. |
| `services/atlas-ui/src/components/common/EmptyState.tsx` | The dead-end branch. Gains the refresh button, the action row, and the caption. |
| `services/atlas-ui/src/components/common/DataTableWrapper.tsx` | Three-branch router. Its empty branch currently drops `onRefresh`/`isRefreshing` on the floor — that is the bug. |
| `services/atlas-ui/src/components/data-table.tsx:76-92` | The toolbar refresh control. Read-only: it is the visual contract the empty-state button mirrors (`RefreshCw`, `animate-spin`, `disabled`, `aria-busy`, `title="Refresh"`). |
| `services/atlas-ui/src/context/tenant-context.tsx` | The only non-React-Query data source in scope. Gains `tenantsUpdatedAt` and a result-returning `refreshTenants`. |

## Decisions carried from `design.md`

D1 (structural `RefreshableQuery`), D2 (minimum non-zero stamp, no `useMemo`),
D3 (action row, refresh always `outline`, action first), D4 (locale text +
ISO `title`), D5 (wrapper forwards, no nested `emptyState.onRefresh`),
D6 (tenant-context shim + local skeleton guard), D7 (`TransportsPage` in
place, no wrapper migration), D8 (`EventOccurrencesPage` direct props).

## Decisions made at plan time

**D9 — `EventDefinitionsPage` and `RewardPoolsPage` move their refresh
control instead of doubling it.** The design's change inventory lists these
two among the "pass-through only" pages. They are not: both call
`useGridRefresh` but deliberately withhold `onRefresh` from the wrapper,
because each renders its own header refresh button. Handing the wrapper
`onRefresh` (which FR-6.1 requires, and which is the only way to light the
empty-state button under D5) also lights `DataTable`'s toolbar control, so
they would ship two identical controls a few pixels apart.
`src/pages/__tests__/RewardPoolsPage.test.tsx:121` — "renders a single refresh
control next to the tabs" — asserts against exactly that, and `getByRole`
throws on two matches, so this is enforced rather than cosmetic.

Resolution (Task 5): the bespoke header button moves into the wrapper.
`EventDefinitionsPage` simply loses its header button. `RewardPoolsPage` loses
the tabs-row button for its four pool tabs but keeps an equivalent control on
the Global tab, which is a hand-built `<Table>` rather than a grid and would
otherwise lose refresh entirely. Net effect: exactly one refresh control per
visible tab, before and after.

**D10 — `TransportsPage`'s data branch keeps its non-toasting refresh.** The
scheduled tab's `DataTable` uses `onRefresh={() => void scheduledQuery.refetch()}`,
which produces no toast, while the new empty-state control routes through
`useGridRefresh` and does. Unifying them would change the populated branch's
behavior, which D7 explicitly scoped out of this task. The two controls never
render simultaneously, so no user sees the inconsistency. Worth a follow-up,
not worth widening this task.

**D11 — `GuildDetailPage` ships without a test suite of its own.** It has none
today, and Task 7 does not add one: the wiring is a verbatim copy of
`GuildsPage.tsx:65-70`, and the behavior it enables is covered at the contract
level by the `EmptyState`, `DataTableWrapper`, and `useGridRefresh` suites.
Stated here so review reads it as a decision rather than an omission.

## Task sizing

No task is deliberately oversized. Task 4 touches eleven files and will trip
plan-lint's F4 warning: it is the same two-line mechanical edit repeated across
eleven pages, which the plan-format rule explicitly allows to batch. Task 6
touches four files but they are one coupled change — the context signature and
its only consumer, plus both suites.

Every task is confined to `services/atlas-ui`, so the >1-service split rule
never applies. There is no cross-service seam in this task; the seam that does
exist is internal, between `useGridRefresh` and the two components, and Tasks
1–3 each assert their own end of it.

## Dependencies and ordering

Tasks 1–3 change the shared contracts. Every later task consumes them, so they
land first and in that order (`DataTableWrapper` forwards props `EmptyState`
must already accept). Tasks 4–9 are mutually independent afterwards. Task 10
is the closing sweep and must run against the finished branch.

## Traps found while reading the code

- `tsconfig.app.json` sets `exactOptionalPropertyTypes: true`. Optional props
  are passed with the conditional-spread idiom (`{...(x && { x })}`), never
  `prop={undefined}`. `lastUpdatedAt` is typed `number | null`, so page call
  sites may pass it plainly; the wrapper→`EmptyState` hop still spreads.
- `useGridRefresh`'s existing test builds queries with
  `as unknown as RefreshableQuery`, so a missing `dataUpdatedAt` would compile
  silently. Task 1 adds the field to the factory default rather than relying on
  the cast.
- `refetch()` resolves and never rejects (React Query v5, `throwOnError:
  false`). Error detection reads each resolved result's `isError`. The
  `TenantsPage` shim must honour the same convention.
- A bare `vi.fn()` mock of `refreshTenants` resolves `undefined`. The shim
  treats a falsy result as success, otherwise every test with an unupdated mock
  would toast a false failure.
- `TenantPageSkeleton` renders no text at all, so "is the skeleton showing?"
  is asserted as the absence of the `Tenants` heading.
- Radix `TabsContent` unmounts inactive tabs, which is what keeps
  `RewardPoolsPage`'s per-tab `getByRole("button", { name: /refresh/i })`
  unambiguous.

## Recorded, not fixed

- **Q3** — `AccountsPage` refreshes ban statuses in a `useEffect` outside React
  Query and does not include that fan-out in its `alsoRefresh`. Pre-existing;
  the fan-out only runs when `accounts.length > 0` and so cannot affect the
  empty state. Goes in `call-site-sweep.md`.
- **FR-6.4** — `src/pages/event-occurrences-columns.tsx` mentions
  `DataTableWrapper` only in a comment at line 4. Not applicable, no code
  change, recorded in the sweep.
- **Q2** — the last-updated caption stays empty-state-only.
  `TransportsPage`'s `FreshnessIndicator` remains the sole populated-grid
  precedent; generalizing it is a follow-up.
