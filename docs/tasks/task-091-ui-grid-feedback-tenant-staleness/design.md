# Data-Grid Refresh Feedback & Tenant-Switch Staleness — Design

Task: task-091-ui-grid-feedback-tenant-staleness
Status: Approved
Created: 2026-06-12
Supersedes input: `prd.md` (v1, approved)

---

## 1. Summary

Two client-only (atlas-ui) defects, fixed independently but landed together:

1. **Refresh feedback gap** — the shared `DataTable` refresh button never spins,
   never disables, and never confirms completion, so admins can't tell a refresh
   did anything (especially when data is unchanged). Fix: a shared
   `useGridRefresh` hook that standardizes every grid on React Query `refetch()`,
   exposes an `isRefreshing` signal sourced from `isFetching`, and toasts on
   settle. `DataTable`/`DataTableWrapper` gain an `isRefreshing` prop that drives
   spinner + disabled state.

2. **Tenant-switch staleness** — a parent-effect-vs-child-fetch ordering race lets
   an in-flight request carry the *old* tenant's headers while being keyed under
   the *new* tenant. Fix: set `api.setTenant` + `queryClient.clear()`
   **synchronously** inside the user-action `setActiveTenant` handler (PRD option
   1a), ahead of the React state update, while keeping the existing effect as the
   catch-all for programmatic tenant changes. Plus `LoginHistoryPage` resets its
   local results on tenant change.

No backend, Kafka, DB, or tenant-header-contract changes. No new dependencies. No
`window.location.reload()`. No removal of the `apiClient` singleton.

Three design decisions were confirmed with the user up front:

- Refresh feedback is centralized in a shared `useGridRefresh` hook (not per-page).
- `LoginHistoryPage` resets local state on tenant change (not a full React Query
  migration).
- The success toast fires on **every** successful refresh (no data-diff
  suppression).

---

## 2. Issue 1 — Refresh feedback

### 2.1 Current state

- `DataTable` (`src/components/data-table.tsx`) renders a static `RefreshCw` icon
  wired to `onRefresh?: () => void`. No fetch-state awareness.
- `DataTableWrapper` (`src/components/common/DataTableWrapper.tsx`) forwards
  `onRefresh` conditionally.
- ~11 pages pass `onRefresh`, but the callbacks are **heterogeneous**:
  - `refetch()`-based: GachaponsPage, MerchantsPage, MapsPage, ServicesPage.
  - `invalidateAll()`-based: CharactersPage, GuildsPage, AccountsPage, BansPage,
    QuestsPage, TemplatesPage.
- `invalidateAll()` returns a promise that resolves immediately (it schedules a
  refetch; it does not await it), so it cannot drive a "done" toast. `refetch()`
  both flips `isFetching` **and** returns a settle-able promise.

This heterogeneity is the root reason a single shared component can't currently
give uniform feedback. The fix standardizes the refresh primitive.

### 2.2 The shared hook: `useGridRefresh`

New file: `src/lib/hooks/useGridRefresh.ts`.

```ts
import type { UseQueryResult } from "@tanstack/react-query";
import * as toast from "@/lib/utils/toast";

type RefreshableQuery = Pick<UseQueryResult, "isFetching" | "refetch">;

export interface UseGridRefreshResult {
  isRefreshing: boolean;
  onRefresh: () => Promise<void>;
}

/**
 * Centralizes grid refresh feedback. Accepts the page's query/queries,
 * refetches them in parallel, and surfaces success/error via the app toast.
 * `isRefreshing` is sourced from React Query's own `isFetching` (FR-1.2).
 */
export function useGridRefresh(
  queries: RefreshableQuery[],
  options?: { successMessage?: string },
): UseGridRefreshResult {
  const isRefreshing = queries.some((q) => q.isFetching);

  const onRefresh = async (): Promise<void> => {
    const results = await Promise.all(queries.map((q) => q.refetch()));
    const failed = results.find((r) => r.isError);
    if (failed) {
      toast.error(failed.error, { context: { action: "refresh" } });
      return;
    }
    toast.success(options?.successMessage ?? "Data refreshed");
  };

  return { isRefreshing, onRefresh };
}
```

Key correctness points:

- **`refetch()` resolves, it does not reject** (React Query v5, default
  `throwOnError: false`). So error detection inspects each resolved result's
  `isError`/`error`, rather than relying on a thrown exception. This is the single
  most important subtlety in the hook and must be preserved.
- **`isRefreshing` is derived from `isFetching`, not a local timer** (satisfies
  FR-1.2). It naturally covers both the user-clicked refetch and any background
  refetch, and it returns to idle exactly when React Query says fetching ended.
- **Re-entrancy** is prevented at the UI layer: while `isRefreshing` is true the
  button is `disabled`, so overlapping refetches can't be triggered by clicks
  (FR-1.5). The hook itself stays stateless.
- **Toast source**: uses the existing `@/lib/utils/toast` wrapper over Sonner
  (rendered once as `<Toaster />` in `App.tsx`). `toast.error(unknown)` already
  transforms API errors into user-facing messages (FR-1.3, FR-1.6). No new toast
  library, no per-page `<Toaster>`.

Accepting an **array** of queries handles multi-query pages cleanly
(CharactersPage = characters + accounts + tenantConfig; GuildsPage = guilds +
characters) without special-casing.

### 2.3 `DataTable` / `DataTableWrapper` changes

`DataTable` gains one prop:

```ts
interface DataTableProps<TData, TValue> {
  // …existing…
  onRefresh?: () => void
  isRefreshing?: boolean   // NEW
}
```

Refresh button becomes:

```tsx
<Button
  variant="outline"
  size="icon"
  onClick={onRefresh}
  disabled={isRefreshing}
  className="hover:bg-accent cursor-pointer"
  title="Refresh"
  aria-busy={isRefreshing}
>
  <RefreshCw className={cn("h-4 w-4", isRefreshing && "animate-spin")} />
</Button>
```

- `animate-spin` (Tailwind) spins the icon while refreshing.
- `disabled` blocks overlapping clicks and is the native a11y signal (FR-1.5,
  Non-Functional a11y). The accessible name (`title="Refresh"`) is retained;
  `aria-busy` adds an explicit in-progress hint.

`DataTableWrapper` forwards the new prop (same conditional-spread style already
used for `onRefresh`):

```tsx
{...(typeof isRefreshing === "boolean" && { isRefreshing })}
```

`DataTableWrapper` keeps showing `PageLoader` only on the **initial** `loading`
(first load, no data yet). A refresh of already-rendered data is *not* a full-page
loader — it's the spinning button — so pages continue to pass `loading` =
`isLoading` (initial), and `isRefreshing` = the hook's signal. The two are
distinct and must not be conflated.

### 2.4 Per-page wiring (the ~11 grids)

Each grid converts its ad-hoc `refresh` to the shared hook. Representative
before/after for CharactersPage:

```tsx
// before
const refresh = () => { invalidateCharacters(); invalidateAccounts(); };
// …
<DataTableWrapper … onRefresh={refresh} />

// after
const { isRefreshing, onRefresh } = useGridRefresh([
  charactersQuery, accountsQuery, tenantConfigQuery,
]);
// …
<DataTableWrapper … onRefresh={onRefresh} isRefreshing={isRefreshing} />
```

Notes:

- Pages already holding the query objects (`charactersQuery`, `mapsQuery`, etc.)
  pass them straight in. Pages that currently only destructure `{ data, refetch }`
  keep the query handle instead so `isFetching` is available.
- `invalidateAll()` usages are replaced by `refetch()` via the hook. Behavioral
  difference is acceptable and desirable: `refetch()` re-runs the *current* query
  keys (which is what the user wants — "refresh what I'm looking at"), and unlike
  `invalidateAll()` it gives a real completion signal. Where a page invalidated a
  *related* resource purely as a side effect (e.g. Characters also refreshing
  Accounts), that query is simply included in the hook's array.
- `characters-columns.tsx` / `accounts-columns.tsx` accept an `onRefresh` used by
  row actions (e.g. post-mutation refresh). Those continue to receive the hook's
  `onRefresh` so a row action and the toolbar button share one path.
- **Inventory of grids with a refresh button** (confirms FR-1.4 scope; PRD open
  question #3): CharactersPage, AccountsPage, GuildsPage, BansPage, MapsPage,
  MerchantsPage, ServicesPage, GachaponsPage, QuestsPage, TemplatesPage. Pages
  with **no** refresh button (e.g. read-only/derived views) are intentionally left
  alone — the hook is opt-in, not forced. The plan phase will enumerate the exact
  final list from `grep -l onRefresh src/pages`.

---

## 3. Issue 2 — Tenant-switch determinism

### 3.1 Root cause (confirmed)

Tenant identity reaches the network via two decoupled channels:

- **React Query keys** include `tenant.id` (correct, recomputed during child
  render on switch).
- **HTTP headers** are read at request time from the mutable singleton
  `apiClient.tenant`, whose only writer today is the `TenantProvider` **effect**
  (`tenant-context.tsx:35-45`).

On a switch, child grids recompute their key to the new tenant and React Query
schedules a fetch **during the child commit**, which runs *before* the parent
effect executes `api.setTenant(newTenant)`. The in-flight request can go out with
the **old** headers and be stored under the **new** key, while `queryClient.clear()`
races it. Result: non-deterministic stale/cross-tenant rows.

### 3.2 Fix — synchronous set in the user-action path (option 1a)

Extract the wiring into one guarded helper and call it **synchronously** from the
user-action handler, *before* the React state update. Keep the effect as the
catch-all for programmatic tenant changes (initial hydration, `refreshTenants`,
`refreshAndSelectTenant`) that do not flow through `setActiveTenant`.

```tsx
// Guarded helper: idempotent per (id) switch. Updates the API client tenant
// and clears the React Query cache exactly once per distinct tenant id.
const previousTenantRef = useRef<Tenant | null>(null);

const applyTenant = useCallback((tenant: Tenant | null) => {
  // Initial null→null mount: nothing to wire, nothing to clear.
  if (tenant === null && previousTenantRef.current === null) return;
  // Same id (e.g. rename rehydrate): wire the fresh object, but DO NOT clear.
  if (previousTenantRef.current?.id === tenant?.id) {
    previousTenantRef.current = tenant;
    api.setTenant(tenant);
    return;
  }
  previousTenantRef.current = tenant;
  api.setTenant(tenant);
  queryClient.clear();
}, [queryClient]);
```

User-action handler now wires **before** re-render:

```tsx
const setActiveTenant = (tenant: Tenant) => {
  applyTenant(tenant);                 // sync: headers set + cache cleared FIRST
  setActiveTenantState(tenant);        // then trigger the grid re-render
  localStorage.setItem(LOCAL_STORAGE_KEY, tenant.id);
};
```

The effect stays, but delegates to the same helper so programmatic paths remain
covered and double-calls are no-ops:

```tsx
useEffect(() => {
  applyTenant(activeTenant);
}, [activeTenant, applyTenant]);
```

### 3.3 Why this satisfies every FR-2

- **FR-2.1 (header set before any tenant-keyed fetch):** `applyTenant` runs
  synchronously inside the click handler, before `setActiveTenantState` schedules
  the re-render that recomputes child query keys. By the time any child issues a
  request, `apiClient.tenant` is already the new tenant.
- **FR-2.2 (cache cleared on switch):** `queryClient.clear()` runs in
  `applyTenant` on a distinct-id change — synchronously on user switches, via the
  effect for programmatic switches.
- **FR-2.3 (initial-mount no-clear guard):** the `tenant === null &&
  previousTenantRef.current === null` early return preserves the existing
  behavior; the effect still fires once on first `null → tenant` hydration via
  `applyTenant`, setting headers without an erroneous extra clear of an
  already-fresh cache. Existing test intent preserved.
- **FR-2.4 (no request carries previous headers post-switch):** guaranteed by the
  ordering in FR-2.1. Any request in flight *at the instant of* the switch was
  issued under the old key and is dropped by `queryClient.clear()`; new requests
  carry new headers.
- **FR-2.6 (same-tenant rename skips clear):** the `previousTenantRef.current?.id
  === tenant?.id` branch wires the fresh object (so attribute changes like a
  rename propagate to headers if needed) but explicitly does **not** clear the
  cache. `refreshTenants` re-selecting the same id goes through the effect →
  `applyTenant` → this branch.

### 3.4 Idempotency / double-call analysis

On a user switch the sequence is:

1. `setActiveTenant` → `applyTenant(new)`: ref now `new`, headers set, cache
   cleared.
2. `setActiveTenantState(new)` → re-render → effect runs `applyTenant(new)`:
   `previousTenantRef.current?.id === new.id` → same-id branch → re-sets headers
   (cheap, identical object) → **no second clear**.

So exactly one `queryClient.clear()` and one logical header-set per switch — the
behavior the existing test asserts (`clearSpy` called once per switch). The test
will be updated only insofar as the *call ordering* changed (clear now precedes
the state commit); the call *counts* per switch are unchanged.

### 3.5 `LoginHistoryPage` (FR-2.5)

`LoginHistoryPage` is search-driven (manual `useCallback` + local `entries`
state), not a list-on-mount grid, so `queryClient.clear()` never touched it and it
kept showing the previous tenant's results. Per the confirmed decision, fix with a
local reset rather than a React Query migration:

```tsx
useEffect(() => {
  setEntries([]);
  setHasSearched(false);
  setSearchCriteria({ ip: "", hwid: "", accountId: "" });
}, [activeTenant?.id]);
```

This clears stale prior-tenant rows on every switch. The page already guards
`handleSearch` on `activeTenant`, so a post-switch search hits the new tenant's
headers (now correctly set synchronously per §3.2). Secondary cleanup: this page
renders its own `<Toaster richColors />` and imports `toast` from `sonner`
directly — left as-is for this task (out of scope; the global `<Toaster />`
coexists), but noted so the plan doesn't accidentally duplicate refresh toasts
here (LoginHistory has no `DataTable` refresh button).

---

## 4. Components & boundaries

| Unit | Responsibility | Depends on | Consumers |
|------|----------------|------------|-----------|
| `useGridRefresh` (new) | Refetch one+ queries in parallel; derive `isRefreshing` from `isFetching`; toast on settle | `@/lib/utils/toast`, React Query result shape | grid pages |
| `DataTable` | Render table + refresh button; spin/disable from `isRefreshing` | `isRefreshing`, `onRefresh` props | `DataTableWrapper` |
| `DataTableWrapper` | Forward `isRefreshing`/`onRefresh`; own initial loading/error/empty states | `DataTable` | grid pages |
| `TenantProvider` | Own active tenant; wire `api.setTenant` + cache clear synchronously on user switch, via effect for programmatic | `api`, `queryClient` | whole app |
| grid pages | Hold queries; pass them to `useGridRefresh`; wire wrapper | `useGridRefresh`, `DataTableWrapper` | routes |
| `LoginHistoryPage` | Reset local results on tenant change | `useTenant` | route |

Each unit is independently testable: `useGridRefresh` with mock query objects;
`DataTable` with a boolean prop; `TenantProvider` with the existing harness.

---

## 5. Testing strategy

All Vitest + `@testing-library/react`, colocated under `__tests__/`.

1. **`useGridRefresh` (new test):**
   - `isRefreshing` is `true` when any supplied query has `isFetching: true`,
     else `false`.
   - `onRefresh` calls `refetch()` on every supplied query (parallel).
   - On all-success, fires `toast.success` once with the message.
   - On any resolved `isError` result, fires `toast.error` (with the error) and
     **not** `toast.success`. (Asserts the resolve-not-reject handling.)
   - Mock `@/lib/utils/toast`; assert spies.

2. **`DataTable` (new/extended test):**
   - `isRefreshing` → refresh button `disabled` and icon has `animate-spin`.
   - `!isRefreshing` → enabled, no spin.
   - Clicking while not refreshing calls `onRefresh`; clicking while disabled does
     not (re-entrancy / FR-1.5).

3. **`tenant-context` (update existing
   `src/context/__tests__/tenant-context.test.tsx`):**
   - Existing three tests stay green (initial-mount no-fire; rename no-clear;
     reselect-on-delete).
   - **New ordering assertion (FR-2.1, FR-2.4 / closes TODO R6):** on
     `setActiveTenant(tenantB)`, assert `api.setTenant` is called with `tenantB`
     **before** the child re-render issues a fetch — implemented by spying call
     order: a child whose render reads `api.getTenant()` must observe the new
     tenant. Concretely, assert `setTenantMock` is invoked synchronously within
     the `act(() => setActiveTenant(...))` block (i.e. before `await waitFor`),
     and `clearSpy` count is exactly one per distinct-id switch.
   - Per-switch `clear` count unchanged (one per switch); same-id rehydrate still
     zero additional clears.

4. **`LoginHistoryPage` (new/extended test):**
   - After populating `entries` and switching `activeTenant.id`, the results table
     is cleared (no prior-tenant rows; `hasSearched` reset).

5. **Manual / acceptance** (documented, not automated): repeated tenant switches
   across a couple of grids show no cross-tenant rows; refresh button spins +
   toasts.

`docs/TODO.md:339` (task-004 R6, "tenant switching … a real-tenant E2E check is
still needed") is closed by test (3)'s ordering assertion — the race is now
covered by an automated unit test. The plan will check that line off.

---

## 6. Verification gate

Per `services/atlas-ui/CLAUDE.md`, all three must be clean before "done":

- `npm run build` (`tsc -b` + vite — note: build type-checks `*.test.ts`, so test
  call-site changes must land in the same commit as signature changes).
- `npm run lint`.
- `npm run test`.

No Go services touched → no `docker buildx bake`, `go test`, `go vet`, or
`redis-key-guard` runs required for this task.

---

## 7. Risks & mitigations

- **`refetch()` error semantics.** If the hook treated `refetch()` as throwing, no
  error toast would ever fire. Mitigated by inspecting resolved `isError` (§2.2)
  and a dedicated unit test (§5.1).
- **Behavioral shift from `invalidateAll()` → `refetch()`.** `invalidateAll()`
  marks broad keys stale (incl. inactive queries); `refetch()` re-runs the
  page's active queries. For the toolbar "refresh what I see" action this is the
  more correct, more responsive behavior, and any related resource a page relied
  on is included in the hook's query array. Low risk; covered by each page still
  rendering its data post-refresh.
- **Double `applyTenant` per switch.** Analyzed in §3.4 — the id-compare guard
  makes the effect's second call a no-op clear-wise; only headers are re-set
  (idempotent). Unit-tested via unchanged `clearSpy` counts.
- **Initial-mount hydration.** The first `null → tenant` still flows through the
  effect (the user-action handler isn't used for hydration), setting headers
  without an erroneous extra clear. Guard preserved and tested.
- **Test-file type-checking.** `tsc -b` compiles tests; any prop/signature change
  (e.g. `DataTable` gaining `isRefreshing`) must update its tests in the same
  commit or the production build breaks. Called out so the plan sequences edits
  together.

---

## 8. Out of scope (reaffirmed from PRD)

- No backend/Go/Kafka/DB changes; tenant header contract unchanged.
- No `window.location.reload()` on switch.
- No removal of the `apiClient` singleton / no per-`queryFn` header derivation
  (option 1c) — tracked separately.
- No service-worker (`sw-character-cache.js`) changes.
- No global React Query config (`staleTime`/`gcTime`/retry) overhaul.
- No data-diff suppression of the success toast (decided: toast on every success).
- No consolidation of the duplicate `sonner` vs `@/lib/utils/toast` usage beyond
  what these grids touch.
