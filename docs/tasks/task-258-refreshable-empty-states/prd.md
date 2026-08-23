# Refreshable Empty States — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-21
---

## 1. Overview

Every list screen in the Atlas Web UI routes its table through
`services/atlas-ui/src/components/common/DataTableWrapper.tsx`. That component has three
mutually exclusive branches: loading, error, and data. When the fetch succeeds but returns
zero rows, the wrapper short-circuits to a bare `<EmptyState />` and discards the
`onRefresh`, `isRefreshing`, and `headerActions` props it was handed. The refresh control
lives inside `DataTable`'s toolbar (`src/components/data-table.tsx:77-92`), which is only
rendered on the data branch. The result is that an empty grid is a dead end: the user
cannot re-query without navigating away and back, or reloading the page.

This matters because empty is frequently a transient or ambiguous condition, not a
terminal one. A tenant that was just provisioned, a service that has not finished
seeding, a grid the user is watching while a backend job populates it — all present as
"No accounts found" with no way to ask again. It is also the one branch where the user
has the least information: the error branch at least tells them what went wrong and
offers "Try Again" (`ErrorDisplay` already accepts a `retry` callback), and the loading
branch is self-resolving. The empty branch offers neither an explanation nor an action.

The predecessor task `task-091-ui-grid-feedback-tenant-staleness` built the
`useGridRefresh` hook that already standardizes grid refresh across the app: it refetches
a set of React Query results in parallel, derives `isRefreshing` from React Query's own
`isFetching`, and surfaces success/failure through the app toast. This task extends that
same contract to the empty branch rather than inventing a parallel mechanism, and adds a
last-updated indicator so a user looking at an empty grid can tell how stale that
emptiness is.

## 2. Goals

Primary goals:

- Every empty grid in the Web UI offers a working refresh control without leaving the screen.
- The refresh action on an empty grid uses the same `useGridRefresh` contract as a populated
  grid — same refetch semantics, same success/error toasts.
- A user looking at an empty grid can see when the data was last fetched.
- Pages that render `DataTableWrapper` without wiring refresh at all are brought up to the
  standard, so the fix is real across the surface rather than latent.

Non-goals:

- Automatic polling or background auto-refresh of empty grids. Refresh stays user-initiated.
- Changing the error branch. `ErrorDisplay` already exposes a working retry and is out of scope.
- Changing pagination, filtering, sorting, or column-visibility behavior.
- Empty-state panels on detail pages (e.g. `GuildDetailPage`'s sub-tables are in scope only
  insofar as they go through `DataTableWrapper`; bespoke "no data" panels elsewhere are not).
- Backend changes. This task touches `services/atlas-ui` only.

## 3. User Stories

- As an operator viewing a freshly provisioned tenant's Accounts grid, I want to click Refresh
  directly in the empty state so that I can see the first account appear without reloading the page.
- As an operator, I want the empty state to tell me when the data was last fetched so that I can
  distinguish "nothing exists" from "I have been staring at a stale view for ten minutes".
- As an operator, I want a confirmation toast after refreshing an empty grid so that I know the
  refresh actually ran and the grid is genuinely empty, rather than the button being inert.
- As an operator on a grid that has a Create action (e.g. "Create Account"), I want that action to
  remain the visually primary button in the empty state so that the intended next step stays obvious,
  with Refresh available alongside it.
- As a developer adding a new list page, I want refresh in the empty state to come for free from
  `DataTableWrapper` so that I cannot ship a dead-end empty grid by omission.

## 4. Functional Requirements

### 4.1 EmptyState refresh control

- **FR-1.1** `EmptyState` accepts an optional `onRefresh: () => void | Promise<void>` prop. When
  supplied, it renders a Refresh button in the empty-state body, below the title/description block.
- **FR-1.2** The Refresh button is labelled "Refresh" and carries the `RefreshCw` icon from
  `lucide-react`, matching the toolbar control in `data-table.tsx` and the "Try Again" control in
  `ErrorDisplay`.
- **FR-1.3** `EmptyState` accepts an optional `isRefreshing: boolean` prop. While `isRefreshing` is
  true the Refresh button is `disabled` and its icon carries the `animate-spin` class, mirroring the
  toolbar's existing in-flight treatment. Disabling is required to prevent overlapping refetches from
  repeated clicks.
- **FR-1.4** When `onRefresh` is not supplied, no Refresh button renders and `EmptyState`'s output is
  unchanged from today.
- **FR-1.5** The Refresh button carries a stable test hook (`data-testid="empty-state-refresh"`) and an
  accessible name of "Refresh". While refreshing it sets `aria-busy="true"`.

### 4.2 Action-slot precedence

- **FR-2.1** When a page supplies both a custom `action` (e.g. "Create Account") and `onRefresh`, both
  buttons render side by side in a single horizontal row.
- **FR-2.2** The custom `action` renders as the primary button (default `Button` variant) and appears
  first in DOM and visual order. Refresh renders as a secondary button (`variant="outline"`) and appears
  second.
- **FR-2.3** When only `action` is supplied, it renders exactly as it does today.
- **FR-2.4** When only `onRefresh` is supplied, Refresh renders alone as the sole button. Its variant is
  `outline` in both cases, so its appearance does not shift depending on whether a sibling action exists.

### 4.3 Last-updated indicator

- **FR-3.1** `EmptyState` accepts an optional `lastUpdatedAt: number | null` prop, expressed as an epoch
  milliseconds timestamp (the shape React Query's `dataUpdatedAt` already uses).
- **FR-3.2** When `lastUpdatedAt` is a positive number, `EmptyState` renders a muted caption below the
  action row reading `Last updated <time>`, where `<time>` is the locale-formatted wall-clock time of
  that timestamp (hours and minutes).
- **FR-3.3** When `lastUpdatedAt` is null, zero, or absent, no caption renders. A query that has never
  resolved must not display a misleading epoch-zero timestamp.
- **FR-3.4** The caption carries `data-testid="empty-state-last-updated"` and a `title` attribute holding
  the full ISO-8601 timestamp, so the exact instant is available on hover.
- **FR-3.5** The caption text updates when `lastUpdatedAt` changes — i.e. after a successful refresh the
  displayed time advances. It is not required to tick on a timer between fetches.

### 4.4 DataTableWrapper plumbing

- **FR-4.1** `DataTableWrapper`'s empty branch forwards its `onRefresh` and `isRefreshing` props into
  `EmptyState`.
- **FR-4.2** `DataTableWrapper` accepts an optional `lastUpdatedAt: number | null` prop and forwards it
  into `EmptyState` on the empty branch.
- **FR-4.3** The wrapper's existing `emptyState` prop object continues to control title, description,
  icon, and custom action. Refresh wiring is driven by the wrapper's top-level `onRefresh`/`isRefreshing`
  props, not by a second copy nested inside `emptyState`, so a page that already wires the toolbar gets
  the empty-state control with no additional prop.
- **FR-4.4** The loading and error branches are unchanged. The data branch is unchanged.
- **FR-4.5** Existing default empty-state copy ("No data available" / "There are no items to display at
  this time.") is retained for pages that supply no `emptyState` override.

### 4.5 useGridRefresh: exposing last-updated

- **FR-5.1** `RefreshableQuery` is widened to include React Query's `dataUpdatedAt` field. Because
  `UseQueryResult` always provides it, this is a `Pick` widening, not a behavioral change for callers
  that pass real query results.
- **FR-5.2** `useGridRefresh` returns an additional `lastUpdatedAt: number | null` value, computed as the
  **minimum** non-zero `dataUpdatedAt` across the supplied queries. Minimum, not maximum: when a page
  refreshes several queries together, the view is only as fresh as its stalest constituent.
- **FR-5.3** When every supplied query has a zero or absent `dataUpdatedAt` (nothing has ever resolved),
  `lastUpdatedAt` is `null`.
- **FR-5.4** `isRefreshing` and `onRefresh` semantics are unchanged, including the existing behavior that
  `refetch()` resolves rather than rejects and errors are detected via each result's `isError`.
- **FR-5.5** The return value remains backward compatible: existing destructuring of
  `{ isRefreshing, onRefresh }` continues to compile and behave identically.

### 4.6 Page coverage

- **FR-6.1** Every page that renders `DataTableWrapper` passes `onRefresh`, `isRefreshing`, and
  `lastUpdatedAt` sourced from `useGridRefresh`.
- **FR-6.2** Pages currently rendering `DataTableWrapper` with an `emptyState` but no refresh wiring —
  `TenantsPage` (`src/pages/TenantsPage.tsx:146`) and `GuildDetailPage`
  (`src/pages/GuildDetailPage.tsx:120`) — gain a `useGridRefresh` call over their existing query or
  queries and pass the three props through.
- **FR-6.3** Pages that already wire `useGridRefresh` gain only the `lastUpdatedAt` pass-through:
  AccountsPage, BansPage, CharactersPage, CouponsPage, EventDefinitionsPage, EventOccurrencesPage,
  GuildsPage, MapsPage, MerchantsPage, QuestsPage, ReportsPage, RewardPoolsPage, ServicesPage,
  TemplatesPage.
- **FR-6.4** `src/pages/event-occurrences-columns.tsx` references `DataTableWrapper`; its usage is
  audited and brought to the same standard if it renders a grid, or explicitly recorded as not
  applicable if the reference is type-only.
- **FR-6.5** An implementation-time sweep confirms no remaining `DataTableWrapper` call site lacks
  `onRefresh`. The sweep result is recorded in the task folder.

### 4.7 Out-of-wrapper grids

- **FR-7.1** `TransportsPage` renders `DataTable` directly rather than through `DataTableWrapper`. Its
  empty behavior is audited. If it can present an empty grid without a refresh control, it is either
  migrated to `DataTableWrapper` or given the equivalent empty-state treatment. Which of the two is a
  design-phase decision.

## 5. API Surface

No HTTP endpoints are added, removed, or modified. The API surface of this task is the TypeScript
component and hook contracts.

### `EmptyState` (`src/components/common/EmptyState.tsx`)

```ts
interface EmptyStateProps {
  icon?: React.ReactNode;
  title: string;
  description?: string;
  action?: { label: string; onClick: () => void };
  // NEW
  onRefresh?: () => void | Promise<void>;
  isRefreshing?: boolean;
  lastUpdatedAt?: number | null;
  className?: string;
}
```

### `DataTableWrapper` (`src/components/common/DataTableWrapper.tsx`)

```ts
interface DataTableWrapperProps<TData extends RowData> {
  // ...existing...
  onRefresh?: () => void;
  isRefreshing?: boolean;
  // NEW
  lastUpdatedAt?: number | null;
}
```

### `useGridRefresh` (`src/lib/hooks/useGridRefresh.ts`)

```ts
export type RefreshableQuery = Pick<
  UseQueryResult,
  "isFetching" | "refetch" | "dataUpdatedAt"
>;

export interface UseGridRefreshResult {
  isRefreshing: boolean;
  onRefresh: () => Promise<void>;
  // NEW
  lastUpdatedAt: number | null;
}
```

Error cases: none new. Refresh failures continue to surface through the existing
`toast.error(failed.error, { context: { action: "refresh" } })` path in `useGridRefresh`, which is
reached identically whether the grid was empty or populated.

## 6. Data Model

No persisted entities, database tables, or migrations. No `tenant_id` scoping concerns at the storage
layer, because nothing is stored.

The only new state is derived and ephemeral: `lastUpdatedAt` is read from React Query's cache entry
(`dataUpdatedAt`) and rendered. It is not written anywhere, not persisted across reloads, and lives
entirely within the existing per-tenant React Query cache keys.

## 7. Service Impact

| Service | Change |
|---|---|
| `services/atlas-ui` | All changes. Three shared modules (`EmptyState`, `DataTableWrapper`, `useGridRefresh`), sixteen-plus page call sites, and their unit tests. |
| All backend services | None. No API, contract, event, or schema change. |

There is no cross-service seam in this task, so the "trace the event into its consumers" review
obligation does not apply. The seam that does exist is the internal one between `useGridRefresh` and the
components — the plan should ensure a test asserts the new `lastUpdatedAt` contract at both ends.

## 8. Non-Functional Requirements

- **Performance.** No new network requests are introduced. The refresh control triggers exactly the same
  `refetch()` calls the toolbar control already triggers. `lastUpdatedAt` is read from cached query state
  and involves no I/O. Rendering the caption must not introduce a per-second re-render.
- **Accessibility.** The Refresh button is reachable by keyboard and has an accessible name of "Refresh".
  While in flight it is `disabled` and `aria-busy="true"`. The last-updated caption is plain text with a
  `title` attribute; it is not an interactive element and is not focusable.
- **Consistency.** The empty-state Refresh control uses the same icon, the same spin-while-busy treatment,
  and the same toast outcomes as the toolbar control. A user must not be able to tell that two different
  code paths are involved.
- **Multi-tenancy.** Refresh operates on the active tenant's existing React Query keys. Refreshing an
  empty grid must not fetch across tenants, and must not resurrect data from a previously active tenant.
  Tenant switching behavior established by `task-091` is preserved unchanged.
- **Observability.** Success and failure both produce a toast via the existing `useGridRefresh` path.
  No new logging or metrics are required.
- **Styling.** Tailwind utility classes and existing `shadcn/ui` `Button` variants only. No new CSS files,
  no new design tokens.
- **Testing.** Vitest + React Testing Library, consistent with the existing suites at
  `src/components/__tests__/` and `src/lib/hooks/__tests__/useGridRefresh.test.ts`.

## 9. Open Questions

- **Q1.** `TransportsPage` uses `DataTable` directly. Migrate it to `DataTableWrapper` (uniform, larger
  diff, small regression risk on a page with its own layout) or hand it an equivalent empty-state
  treatment in place? Deferred to the design phase; FR-7.1 accepts either.
- **Q2.** Should the last-updated caption also appear on populated grids, or only in the empty state? This
  PRD scopes it to the empty state only, where staleness is least visible. Extending it to the toolbar is a
  plausible follow-up but is deliberately not in scope.
- **Q3.** `AccountsPage` fetches ban statuses in a `useEffect` outside React Query, and its
  `useGridRefresh` call does not include that fan-out via `alsoRefresh`. This is a pre-existing gap, not
  one introduced here. Fold it into this task or leave it? Recommendation: leave it, and note it, because
  the fan-out only runs when `accounts.length > 0` and therefore cannot affect the empty state.
- **Q4.** Time format for the caption — locale-default `toLocaleTimeString` (respects the user's locale,
  varies across environments and needs a pinned locale in tests) versus a fixed `HH:MM`. Recommendation:
  locale-default in the component, with tests pinning the timestamp and asserting via a `title`-attribute
  ISO match rather than the localized string.

## 10. Acceptance Criteria

- [ ] `EmptyState` renders a Refresh button when `onRefresh` is supplied, and renders none when it is not.
- [ ] The Refresh button is disabled and its icon spins while `isRefreshing` is true.
- [ ] Clicking Refresh in an empty grid invokes the page's `useGridRefresh` `onRefresh`, and a success toast
      appears on success / an error toast on failure.
- [ ] When both a custom action and `onRefresh` are supplied, both buttons render, the custom action is the
      primary variant, and it precedes Refresh in DOM order.
- [ ] `EmptyState` renders `Last updated <time>` when `lastUpdatedAt` is a positive number, and renders
      nothing when it is null or zero.
- [ ] The last-updated caption exposes the full ISO timestamp via its `title` attribute.
- [ ] `useGridRefresh` returns `lastUpdatedAt` as the minimum non-zero `dataUpdatedAt` across its queries,
      and `null` when none has resolved.
- [ ] Existing `{ isRefreshing, onRefresh }` destructuring at every current call site compiles and behaves
      unchanged.
- [ ] `TenantsPage` and `GuildDetailPage` wire `useGridRefresh` and show a working Refresh in their empty
      states.
- [ ] Every `DataTableWrapper` call site passes `onRefresh`, `isRefreshing`, and `lastUpdatedAt`; the sweep
      confirming this is recorded in the task folder.
- [ ] `TransportsPage`'s empty grid offers a refresh control (by migration or equivalent treatment), or the
      audit records that it cannot render an empty grid.
- [ ] Unit tests cover: refresh button presence/absence, disabled-while-refreshing, action + refresh
      precedence and ordering, caption present/absent/format, and the `useGridRefresh` minimum-timestamp and
      null cases.
- [ ] `frontend-guidelines-reviewer` passes on the changed TypeScript files.
- [ ] Flagless `tools/verify.sh` exits 0.
