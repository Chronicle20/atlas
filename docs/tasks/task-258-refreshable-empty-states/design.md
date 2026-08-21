# Refreshable Empty States — Design

Task: `task-258-refreshable-empty-states`
Input: `docs/tasks/task-258-refreshable-empty-states/prd.md` (approved, v1)
Status: Draft for planning
Created: 2026-08-21

---

## 1. Design summary

The change is small at the centre and wide at the edges. Three shared modules gain
props/return values (`EmptyState`, `DataTableWrapper`, `useGridRefresh`), and every list
page passes one more value through. Nothing about the refresh mechanism itself is new:
`useGridRefresh` already owns refetch fan-out, `isFetching`-derived in-flight state, and the
success/error toast. This task moves the *control* into a branch that previously discarded it,
and surfaces one derived value (`dataUpdatedAt`) that React Query has always kept.

The interesting work is not in `EmptyState`. It is in the four call sites that do not fit the
"page holds one or more `UseQueryResult`s" shape the hook was written for:

| Call site | Shape | Consequence |
|---|---|---|
| `TenantsPage` | Data comes from `useTenant()` context, **not** React Query | No `refetch`, no `dataUpdatedAt`, and `refreshTenants()` swallows its own errors |
| `GuildDetailPage` | Two real queries, wrapper gets neither `loading` nor `error` | Straightforward, but wiring `onRefresh` also lights the toolbar control on the data branch |
| `EventOccurrencesPage` | Renders `EmptyState` **directly**, not through `DataTableWrapper` | FR-6.3 lists it as a wrapper page; it is not. Needs direct props |
| `TransportsPage` | Renders `DataTable` directly, bespoke empty `<p>`, and already has a `FreshnessIndicator` | FR-7.1 / Q1 decision required |

Two PRD statements are contradicted by the code and are resolved below (D1, D6). Everything
else follows the PRD as written.

---

## 2. Current state (verified)

- `EmptyState` (`src/components/common/EmptyState.tsx`) renders icon / title / description /
  optional single `action` button. No refresh, no timestamp.
- `DataTableWrapper` (`src/components/common/DataTableWrapper.tsx`) branches
  loading → `PageLoader`, error → `ErrorDisplay` (passing `retry: onRefresh` when present),
  `!data.length` → `<EmptyState {...defaultEmptyState} />` (props dropped), else → `DataTable`.
- `DataTable` (`src/components/data-table.tsx:77-92`) renders the toolbar `RefreshCw` button:
  `variant="outline"`, `size="icon"`, `disabled={isRefreshing}`, `aria-busy`, `title="Refresh"`,
  icon `cn("h-4 w-4", isRefreshing && "animate-spin")`. This is the visual contract to mirror.
- `useGridRefresh` (`src/lib/hooks/useGridRefresh.ts`) takes
  `RefreshableQuery[] = Pick<UseQueryResult, "isFetching" | "refetch">[]`, derives
  `isRefreshing` from `isFetching`, refetches in parallel, inspects each resolved result's
  `isError` (refetch resolves, never rejects), and toasts.
- `toast.error(error: unknown, options)` — accepts `unknown`, which matters for D2.
- `tsconfig.app.json` sets `strict`, `noUncheckedIndexedAccess`, and
  **`exactOptionalPropertyTypes: true`**. Optional props must be passed with the conditional
  spread idiom (`{...(x && { x })}`) already used throughout `DataTableWrapper`, not as
  `prop={undefined}`.
- 14 pages already call `useGridRefresh` and destructure `{ isRefreshing, onRefresh }`.
- `src/pages/event-occurrences-columns.tsx` mentions `DataTableWrapper` **only in a comment**
  (line 4: "This deliberately does NOT use `DataTableColumnDef`/`DataTableWrapper`"). FR-6.4 is
  therefore satisfied by recording it as not applicable — no code change.

---

## 3. Design decisions

### D1 — `RefreshableQuery` becomes a hand-written structural interface, not a widened `Pick`

**PRD says (FR-5.1):** widen the `Pick` to
`Pick<UseQueryResult, "isFetching" | "refetch" | "dataUpdatedAt">`.

**Problem:** `TenantsPage` has no `UseQueryResult` to hand it (D6). A shim object cannot satisfy
`Pick<UseQueryResult, "refetch">`, because that type's return is
`Promise<QueryObserverResult<unknown, Error>>` — a ~20-field union no hand-built object can
produce without a lie-cast.

**Decision:** define the contract the hook actually needs:

```ts
export interface RefreshableQuery {
  isFetching: boolean;
  dataUpdatedAt: number;
  refetch: () => Promise<{ isError: boolean; error: unknown }>;
}
```

Every `UseQueryResult<T, E>` is structurally assignable to this — fewer declared parameters on
`refetch` is fine, `QueryObserverResult` is assignable to `{ isError: boolean; error: unknown }`,
and `toast.error` already takes `unknown`. So all 14 existing call sites compile untouched,
which is what FR-5.1 was actually protecting, while non-React-Query sources become expressible.

**Alternative considered:** keep the `Pick` and give `TenantsPage` its own bespoke refresh handler
that duplicates the toast calls. Rejected — it forks the toast contract that NFR "Consistency"
exists to prevent, and the fork would drift the first time the toast copy changes.

**Alternative considered:** overload the hook to accept `RefreshableQuery | RefreshSource`.
Rejected — two shapes for one job, with no gain over one honest structural type.

Deviation from the PRD's literal spelling; the PRD's stated *substance* ("include
`dataUpdatedAt`", "existing destructuring compiles unchanged") is preserved.

### D2 — `lastUpdatedAt` is the minimum non-zero `dataUpdatedAt`, computed inline

Per FR-5.2/5.3, a view is only as fresh as its stalest constituent:

```ts
const stamps = queries.map((q) => q.dataUpdatedAt).filter((t) => t > 0);
const lastUpdatedAt = stamps.length ? Math.min(...stamps) : null;
```

No `useMemo`. This is an O(n≤3) reduce over values that change exactly when a query resolves;
memoizing it would cost more than it saves and adds a dependency array to keep correct.

### D3 — `EmptyState` renders an action **row**, refresh always `outline`

```
[ icon ]
  Title
  Description
  [ Primary action ] [ ⟳ Refresh ]     ← flex row, gap-2, mt-4
  Last updated 14:32                    ← muted caption, mt-3
```

- Action row is a `div` with `mt-4 flex items-center justify-center gap-2`, rendered only when
  `action || onRefresh`. Custom `action` first in DOM order (FR-2.2), default `Button` variant.
- Refresh is `variant="outline"` unconditionally (FR-2.4), so it does not change appearance when
  a sibling action appears or disappears.
- Refresh is a **labelled** button (icon + text "Refresh"), unlike the toolbar's icon-only
  `size="icon"` control. In the empty state there is no toolbar context to make an unlabelled
  glyph legible, and FR-1.2/FR-1.5 require the visible label and accessible name. Icon,
  `animate-spin`, `disabled`, and `aria-busy` treatment are copied verbatim from
  `data-table.tsx:77-92` so the two controls read as one mechanism.
- `data-testid="empty-state-refresh"`, `data-testid="empty-state-last-updated"` (FR-1.5, FR-3.4).

`onRefresh` is typed `() => void | Promise<void>` and invoked as `onClick={() => void onRefresh()}`
so a returned promise is not leaked into the DOM handler and `no-misused-promises` stays quiet.

### D4 — Caption formatting: locale time in the text, ISO in `title`

```tsx
<p className="mt-3 text-xs text-muted-foreground"
   data-testid="empty-state-last-updated"
   title={new Date(lastUpdatedAt).toISOString()}>
  Last updated {new Date(lastUpdatedAt).toLocaleTimeString(undefined, {
    hour: "2-digit", minute: "2-digit",
  })}
</p>
```

Answers **Q4** with the PRD's recommendation: locale-default rendering, tests assert the
`title` ISO string rather than the localized body, so the suite does not depend on the runner's
`TZ`/`LANG`. Guard is `lastUpdatedAt > 0` — `null`, `0`, and `undefined` all render nothing
(FR-3.3). No timer, no `useClock` (NFR performance; FR-3.5 explicitly does not require ticking).

**Alternative considered:** reuse the existing `FreshnessIndicator`
(`src/components/features/transports/FreshnessIndicator.tsx`), which shows relative age and
already ticks off the shared `useClock`. Rejected for the empty state — it is a transports
feature component with an error/loading vocabulary of its own, it subscribes to a global clock
(a per-second re-render the NFR forbids here), and it renders relative age where FR-3.2 asks for
wall-clock time. It stays where it is and keeps owning the `TransportsPage` header (D7).

### D5 — `DataTableWrapper` forwards, it does not re-declare

The empty branch becomes:

```tsx
<EmptyState
  {...defaultEmptyState}
  {...(onRefresh && { onRefresh })}
  {...(typeof isRefreshing === "boolean" && { isRefreshing })}
  {...(lastUpdatedAt != null && { lastUpdatedAt })}
/>
```

Conditional spreads are required by `exactOptionalPropertyTypes` and match the idiom already
used on the data branch. Per FR-4.3 there is **no** nested `emptyState.onRefresh`: a page that
wires the toolbar gets the empty-state control for free, which is the FR-6.5 / "cannot ship a
dead-end empty grid by omission" property. Loading, error, and data branches are untouched
(FR-4.4).

Note a side effect on the two pages gaining `onRefresh` for the first time: the wrapper already
forwards `onRefresh` as `ErrorDisplay`'s `retry`, and `DataTable`'s toolbar refresh. So
`TenantsPage` and `GuildDetailPage` also gain a working toolbar refresh on the populated branch.
That is desirable and consistent; it is called out here so review does not read it as scope creep.

### D6 — `TenantsPage`: stamp the timestamp in the context, report the error through the shim

**PRD says (FR-6.2):** `TenantsPage` "gains a `useGridRefresh` call over their existing query or
queries". It has none — `const { tenants, loading, refreshTenants } = useTenant()`
(`src/pages/TenantsPage.tsx:45`). Three sub-problems, three decisions:

**(a) No `refetch`.** Adapt via a `RefreshableQuery` shim, made possible by D1.

**(b) `refreshTenants()` swallows errors.** `tenant-context.tsx` catches, calls `setError` on
state it does not expose in the provider value, and resolves `void`. A shim built on that would
toast "Data refreshed" after a failed refresh — a worse outcome than today's dead end.
Decision: widen the context function to
`refreshTenants: () => Promise<{ ok: true } | { ok: false; error: unknown }>`.
Every current caller ignores the return, so this is source-compatible; the only consumer is the
shim. Test mocks that are bare `vi.fn()` return `undefined`, so `TenantsPage` must treat a
falsy/absent result as success (`result && result.ok === false` → error) **and**
`src/pages/__tests__/TenantsPage.test.tsx`'s mock is updated to return `{ ok: true }`. The other
mock sites do not consume the return and need no change.

**(c) No `dataUpdatedAt`.** Stamp it where the fetch happens rather than inferring it in the
page: `tenant-context.tsx` gains `tenantsUpdatedAt: number` (0 until first success), set beside
each `setTenants(data)` — the initial-load effect, `refreshTenants`, and `refreshAndSelectTenant`.

**Alternative considered for (c):** a page-local `useState` stamped by an effect on
`[loading, tenants]`. Rejected — it infers a fetch boundary from render state, double-fires, and
re-stamps on unrelated re-renders. Three lines in the context are simpler and true.

**(d) The skeleton flash.** `TenantsPage` returns `<TenantPageSkeleton />` whenever context
`loading` is true, and `refreshTenants` sets `loading = true`. Clicking Refresh in the empty
state would blank the entire page into a skeleton and back — the exact "dead end" feeling this
task removes. Decision: `TenantsPage` keeps a local `isRefreshing` flag set around its own
refresh call and changes the guard to `if (loading && !isRefreshing) return <TenantPageSkeleton/>`.
The shim's `isFetching` reads that local flag, not context `loading`, so the spinner lives on the
button where it belongs. This is a `TenantsPage`-local concern; context `loading` semantics are
unchanged for every other consumer.

### D7 — `TransportsPage`: in-place `EmptyState`, not a `DataTableWrapper` migration

Answers **Q1 / FR-7.1**. `TransportsPage` can absolutely render an empty grid: the scheduled tab
has an explicit `routes.length === 0` branch rendering
`<p>No scheduled routes configured.</p>` with no control (`src/pages/TransportsPage.tsx`).

Decision: replace that `<p>` with `<EmptyState title="No scheduled routes configured"
onRefresh={...} isRefreshing={scheduledQuery.isFetching} />`, driven by a `useGridRefresh`
over `scheduledQuery` so the toast contract matches every other grid. **No** `lastUpdatedAt` is
passed — the page header already renders `FreshnessIndicator` bound to
`scheduledQuery.dataUpdatedAt`, and a second freshness readout on the same query would be
redundant and could disagree visually (relative vs wall-clock).

**Alternative considered:** migrate the tab to `DataTableWrapper` for uniformity. Rejected —
the wrapper would replace the page's bespoke loading text and its inline `AlertTriangle` error
line with `PageLoader`/`ErrorDisplay`, changing two branches this task has no mandate over, on a
page with hand-tuned layout comments about `DataTable`'s fixed-height block. Larger diff,
regression risk in exchange for uniformity the user cannot see. FR-7.1 permits either.

The instance and vessels tabs render `InstanceRoutesTable` / `VesselsTable`, which are neither
`DataTableWrapper` nor `DataTable` grids; they are out of scope per the PRD's "bespoke no-data
panels elsewhere are not [in scope]".

### D8 — `EventOccurrencesPage` gets direct `EmptyState` props

FR-6.3 lists it among "pages that already wire `useGridRefresh`" needing only the
`lastUpdatedAt` pass-through, implying it renders `DataTableWrapper`. It does not — it renders
`EmptyState` directly (`src/pages/EventOccurrencesPage.tsx:204`) above a hand-built `<Table>`.
It does already call `useGridRefresh([occurrencesQuery])` (line 109). Decision: pass
`onRefresh`, `isRefreshing`, and `lastUpdatedAt` straight to that `EmptyState`. Same outcome,
one less indirection; no migration.

---

## 4. Change inventory

**Shared modules (3)**

| File | Change |
|---|---|
| `src/lib/hooks/useGridRefresh.ts` | `RefreshableQuery` → structural interface with `dataUpdatedAt` (D1); return `lastUpdatedAt` (D2) |
| `src/components/common/EmptyState.tsx` | `onRefresh` / `isRefreshing` / `lastUpdatedAt` props; action row; caption (D3, D4) |
| `src/components/common/DataTableWrapper.tsx` | `lastUpdatedAt` prop; forward all three on the empty branch (D5) |

**Context (1)**

| File | Change |
|---|---|
| `src/context/tenant-context.tsx` | `tenantsUpdatedAt: number` in the value; `refreshTenants` returns `{ ok }` (D6b, D6c) |

**Pages — pass-through only (14)**

`AccountsPage`, `BansPage`, `CharactersPage`, `CouponsPage`, `EventDefinitionsPage`,
`GuildsPage`, `MapsPage`, `MerchantsPage`, `QuestsPage`, `ReportsPage`, `RewardPoolsPage`,
`ServicesPage`, `TemplatesPage` — destructure `lastUpdatedAt` from `useGridRefresh` and pass it
to `DataTableWrapper`. Plus `EventOccurrencesPage` (D8, direct `EmptyState`).

**Pages — new wiring (3)**

| File | Change |
|---|---|
| `src/pages/TenantsPage.tsx` | Shim + `useGridRefresh` + local `isRefreshing` skeleton guard (D6) |
| `src/pages/GuildDetailPage.tsx` | `useGridRefresh([guildQuery, tenantConfigQuery])`, pass all three |
| `src/pages/TransportsPage.tsx` | Scheduled-tab empty branch → `EmptyState` with refresh (D7) |

**No change**

`src/pages/event-occurrences-columns.tsx` — comment-only reference (FR-6.4, recorded as N/A).
`src/components/data-table.tsx` — the toolbar is already correct.

**FR-6.5 sweep artifact.** The implementation records
`docs/tasks/task-258-refreshable-empty-states/call-site-sweep.md` containing the output of a
`grep -rn "<DataTableWrapper" src/` enumeration with each site's three props ticked, plus the two
non-wrapper grids (`EventOccurrencesPage`, `TransportsPage`) and the N/A comment reference.

---

## 5. Testing strategy

Vitest + React Testing Library, alongside the existing suites.

**`src/components/common/__tests__/EmptyState.test.tsx` (new)**
- No `onRefresh` → no `empty-state-refresh` node; output otherwise identical to today (FR-1.4).
- `onRefresh` supplied → button present, accessible name "Refresh", click invokes the callback.
- `isRefreshing` → button `disabled`, `aria-busy="true"`, icon carries `animate-spin`.
- `action` + `onRefresh` → both render; the action precedes refresh in DOM order (assert via
  `within(row).getAllByRole("button")` index, not by CSS); action is default variant, refresh
  `outline`.
- `lastUpdatedAt` positive → caption present, `title` equals the expected ISO string;
  `null` / `0` / absent → caption absent.
- Caption advances when `lastUpdatedAt` changes across a rerender (FR-3.5).

**`src/lib/hooks/__tests__/useGridRefresh.test.ts` (extend)**
- `lastUpdatedAt` is the **minimum** non-zero stamp across queries (the "stalest constituent"
  rule — a test with `[5000, 9000]` must yield `5000`, so a max-vs-min regression fails loudly).
- All-zero / empty → `null`; a mix of zero and non-zero ignores the zeros.
- Existing `{ isRefreshing, onRefresh }` behavior and toast assertions unchanged (FR-5.4, FR-5.5).

**`DataTableWrapper` (new or extended)**
- Empty data + `onRefresh` → the empty-state refresh button renders and clicking it calls through.
- Empty data + `emptyState.action` + `onRefresh` → both render (the FR-2 precedence path end-to-end).
- Empty data, no `onRefresh` → no refresh button (the wrapper does not invent one).
- `lastUpdatedAt` forwarded to the caption.
- Loading and error branches unchanged.

**`src/pages/__tests__/TenantsPage.test.tsx` (extend)**
- Mock returns `{ ok: true }` from `refreshTenants` (D6b).
- Empty tenants list → refresh button visible; clicking calls `refreshTenants`.
- While refreshing, the page shows the grid with a spinning button, **not** `TenantPageSkeleton` (D6d).
- `refreshTenants` resolving `{ ok: false }` → error toast, not success.

**`src/pages/__tests__/TransportsPage.test.tsx` (extend)**
- Scheduled tab with zero routes → `EmptyState` with a working refresh; header
  `FreshnessIndicator` still present and unduplicated.

Existing suites (`data-table.test.tsx`, tenant-context, breadcrumbs, transports) must pass
untouched apart from the two mock updates named above — that is the FR-5.5 backward-compatibility
proof.

---

## 6. Risks

| Risk | Mitigation |
|---|---|
| D1's structural widening silently accepts a malformed shim | The shim exists in exactly one place (`TenantsPage`) and is unit-tested for both `ok` outcomes |
| D6b's context signature change breaks a bare `vi.fn()` mock | `TenantsPage` treats a falsy result as success; only the one consuming mock is updated |
| D6d's skeleton guard change regresses first-load behavior | Local `isRefreshing` starts `false`, so initial load is bit-identical; covered by an explicit first-load test |
| A page is missed, leaving a latent dead end | FR-6.5 sweep artifact + the D5 property that wrapper pages need no extra prop for the button to appear |
| `exactOptionalPropertyTypes` rejects `prop={undefined}` | Conditional-spread idiom mandated in D5; `tools/verify.sh` type-checks |

---

## 7. Open questions — resolved

- **Q1** (`TransportsPage` migrate vs in-place) → **in place**, D7.
- **Q2** (caption on populated grids too) → **empty state only**, per PRD scope. `TransportsPage`
  keeps `FreshnessIndicator` as the populated-grid precedent; generalizing it is a follow-up.
- **Q3** (`AccountsPage` ban-status fan-out outside React Query) → **leave and note**, per the
  PRD's own recommendation. The fan-out runs only when `accounts.length > 0`, so it cannot affect
  the empty state this task is about. Recorded in the sweep artifact as a known pre-existing gap.
- **Q4** (time format) → **locale-default text, ISO in `title`, tests assert the `title`**, D4.

## 8. PRD deviations to carry into the plan

1. **FR-5.1** — `RefreshableQuery` is a hand-written structural interface rather than a widened
   `Pick`. Substance preserved, spelling changed. (D1)
2. **FR-6.2** — `TenantsPage` has no React Query source; it is wired through a context shim, and
   the change reaches `tenant-context.tsx`. (D6)
3. **FR-6.3** — `EventOccurrencesPage` renders `EmptyState` directly, not `DataTableWrapper`; it
   gets direct props instead of a wrapper pass-through. (D8)
4. **FR-6.4** — `event-occurrences-columns.tsx` references `DataTableWrapper` only in a comment.
   Recorded as not applicable; no code change.
