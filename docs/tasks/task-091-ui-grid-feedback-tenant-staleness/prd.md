# Data-Grid Refresh Feedback & Tenant-Switch Staleness — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-06-12
---

## 1. Overview

The atlas-ui admin frontend renders most of its domain data (characters, accounts,
bans, guilds, maps, monsters, NPCs, reactors, etc.) through a shared `DataTable`
grid. Two related defects degrade trust in that grid:

1. **The refresh button gives no feedback.** The button is a static `RefreshCw`
   icon (`src/components/data-table.tsx:47‑57`) wired to an `onRefresh` callback.
   When clicked, the underlying React Query refetch *does* fire, but the button
   never spins, never disables, and never reports completion. If the refetched
   data is identical to what's on screen (common — admins refresh to check
   "did anything change?"), or the refetch is briefly retrying a transient error,
   the user cannot tell the click did anything. The reported symptom — "sometimes
   hitting refresh doesn't refresh the data" — is a **feedback gap, not a fetch
   gap**: the request fires; the UI just doesn't say so.

2. **Switching tenants intermittently shows the previous tenant's data.** Tenant
   identity reaches the network through two decoupled channels. React Query keys
   every grid query by `tenant.id` (e.g. `characterKeys.list`,
   `src/lib/hooks/api/useCharacters.ts:22`), which is correct. But the actual HTTP
   tenant headers (`TENANT_ID`, `REGION`, `MAJOR_VERSION`, `MINOR_VERSION`) are
   read at request time from a **mutable singleton** — `apiClient.tenant`
   (`src/lib/api/client.ts:80,88‑104`) — whose only runtime writer is the
   `TenantProvider` effect (`src/context/tenant-context.tsx:35‑45`). On a tenant
   switch, child grid components recompute their query key to the new tenant and
   React Query schedules a fetch **during the child commit**, which runs *before*
   the parent `TenantProvider` effect executes `api.setTenant(newTenant)`. The
   in-flight request can therefore go out carrying the **old** tenant's headers
   and be stored under the **new** tenant's key, while `queryClient.clear()` races
   against it. The result is non-deterministic ("sometimes") stale or cross-tenant
   data.

   A secondary contributor: `LoginHistoryPage` (`src/pages/LoginHistoryPage.tsx`)
   is the one remaining grid that fetches via a manual `useCallback` search
   handler rather than React Query, so `queryClient.clear()` doesn't affect it and
   it never re-fetches on tenant change — it keeps showing the previous tenant's
   results until the user manually searches again.

This task fixes both: add real refresh feedback, and make tenant switching
deterministically surface the new tenant's data.

## 2. Goals

Primary goals:

- The grid refresh control visibly reflects fetch state: it spins and is disabled
  while a refetch is in flight, and surfaces completion/error via a toast.
- Switching the active tenant deterministically results in every grid showing the
  newly selected tenant's data — no stale or cross-tenant rows, on any switch.
- Eliminate the tenant-header race by ensuring `apiClient.tenant` is updated
  **before** any tenant-keyed query can issue a request (synchronous-set approach,
  option 1a from the design discussion).
- `LoginHistoryPage` participates in the tenant-switch refresh like every other
  grid.
- Close the open follow-up `docs/TODO.md:339` (task-004 risk R6: "tenant switching
  invalidates the React Query cache — a real-tenant E2E check is still needed").

Non-goals:

- No backend / Go service changes. The tenant header contract
  (`TENANT_ID`/`REGION`/`MAJOR_VERSION`/`MINOR_VERSION`) is unchanged.
- No hard `window.location.reload()` on tenant switch. We keep the SPA experience
  (no white flash, no bundle re-download, no loss of unrelated UI state).
- No removal of the `apiClient` singleton or rewrite of the ~20 service modules to
  derive headers inside each `queryFn` (option 1c). That is a larger refactor
  tracked separately; this task closes the race with the minimal, synchronous-set
  approach instead.
- No changes to the character-image service worker (`public/sw-character-cache.js`)
  — it caches images, not grid data, and is out of scope.
- No global React Query config overhaul (`staleTime`, `gcTime`, retry policy stay
  as-is).

## 3. User Stories

- As an admin, when I click a grid's refresh button, I want the button to show it
  is working (spinner + disabled) and tell me when it's done, so I can trust that
  the data on screen is current even when nothing changed.
- As an admin who manages multiple tenants, when I switch the active tenant, I want
  every grid to immediately show the selected tenant's data, so I never act on the
  wrong tenant's records.
- As an admin on the Login History page, when I switch tenants, I want the page to
  stop showing the previous tenant's results, so I don't mistake stale data for
  current data.

## 4. Functional Requirements

### 4.1 Refresh feedback (Issue 1)

- FR-1.1 `DataTable` MUST accept an optional fetch-state signal (e.g. an
  `isRefreshing?: boolean` prop) and, when truthy, render the refresh icon in a
  spinning state and disable the button to prevent overlapping refetches.
- FR-1.2 The fetch-state signal MUST be sourced from React Query's actual fetch
  status for the grid's query (e.g. `isFetching` / `isRefetching` from the
  relevant `useQuery`/`use<Resource>` hook), not a locally faked timer.
- FR-1.3 On refresh **success**, the page MUST surface a brief success toast
  (e.g. "Data refreshed"). On refresh **error**, the page MUST surface an error
  toast carrying a meaningful message derived from the API error.
- FR-1.4 Refresh feedback wiring MUST be applied consistently across all grids that
  expose a refresh button, via the shared `DataTable`/`DataTableWrapper`
  components, so behavior is uniform rather than per-page bespoke.
- FR-1.5 While a refetch is in flight, repeated clicks MUST NOT stack multiple
  concurrent refetches (the disabled state satisfies this).
- FR-1.6 Toasts MUST use the existing toast mechanism already wired in the app
  (the `<Toaster />` in the provider stack); no new toast library.

### 4.2 Tenant-switch determinism (Issue 2)

- FR-2.1 When the active tenant changes, `apiClient.tenant` MUST be updated to the
  new tenant **before** any tenant-keyed React Query can issue a network request
  for that tenant. Concretely, `api.setTenant(newTenant)` is invoked synchronously
  in the tenant-selection path (the `setActiveTenant` handler), ahead of the React
  state update that re-renders the grids — eliminating the parent-effect-vs-child-
  fetch ordering race.
- FR-2.2 The React Query cache MUST be cleared (`queryClient.clear()`) on tenant
  change so no other tenant's cached entries remain, retaining the existing
  isolation guarantee.
- FR-2.3 The initial-mount guard (do not clear/reset on the first `null → tenant`
  hydration) MUST be preserved; existing behavior covered by
  `src/context/__tests__/tenant-context.test.tsx` MUST continue to pass (updated as
  needed to reflect the new call ordering).
- FR-2.4 After a tenant switch, no grid may issue or surface a request carrying the
  previous tenant's headers. This MUST hold for the React-Query grids and for any
  in-flight request at the moment of the switch.
- FR-2.5 `LoginHistoryPage` MUST re-fetch (or reset) on tenant change rather than
  retaining the previous tenant's results. Preferred approach: migrate it to a
  React Query hook keyed by tenant id (consistent with every other grid); an
  acceptable alternative is to key/reset its local result state on the active
  tenant id. Either way, switching tenants MUST NOT leave stale prior-tenant rows
  on screen.
- FR-2.6 The rehydrate-same-tenant path (`refreshTenants` re-selecting the same
  tenant after a rename, `tenant-context.tsx:80‑93`) MUST continue to skip the
  cache clear (id-compare guard), so a rename does not needlessly wipe the cache.

## 5. API Surface

No HTTP API changes. This is a client-only change.

Internal (frontend) surface changes:

- `DataTable` props gain an optional fetch-state input (e.g. `isRefreshing?: boolean`).
- `DataTableWrapper` forwards that input from the page's query state.
- `TenantContextType.setActiveTenant` behavior changes (now also performs the
  synchronous tenant-client update + cache clear); its signature is unchanged.

## 6. Data Model

No persistent data model changes. No new entities, columns, or migrations. All
state involved is client-side (React Query cache + the `apiClient` tenant
singleton + React component state).

## 7. Service Impact

Only **atlas-ui** is affected. No Go service, Kafka topic, or database is touched.

Files expected to change:

- `src/components/data-table.tsx` — accept + render the fetch-state (spinner +
  disabled refresh button).
- `src/components/common/DataTableWrapper.tsx` — forward fetch-state and refresh
  callback.
- `src/context/tenant-context.tsx` — move `api.setTenant` + `queryClient.clear()`
  into the synchronous `setActiveTenant` path; keep the initial-mount and
  same-tenant guards.
- `src/lib/api/client.ts` — only if the tenant setter needs to be callable
  synchronously from the selection handler (it already is via `api.setTenant`);
  no behavior change to header construction.
- `src/pages/LoginHistoryPage.tsx` — refetch/reset on tenant change (preferably
  via a tenant-keyed React Query hook).
- Pages that own a grid + refresh button — pass `isRefreshing` (the query's
  `isFetching`) and add success/error toasts. Wiring should be centralized as much
  as possible to avoid per-page divergence.
- Tests: `src/context/__tests__/tenant-context.test.tsx` and any
  `DataTable`/grid tests updated to cover the new behavior.

## 8. Non-Functional Requirements

- **Multi-tenancy:** The four tenant headers and their SCREAMING_SNAKE_CASE names
  are unchanged and still injected by `tenantHeaders`. The fix strengthens tenant
  isolation by removing a race, not by altering the contract.
- **Performance:** No measurable regression. Synchronous `api.setTenant` is an
  in-memory assignment; `queryClient.clear()` already runs on switch today.
- **Accessibility:** The refresh button retains its accessible name/title; the
  disabled state during fetch is conveyed via the native disabled attribute.
- **Consistency:** Refresh feedback must look and behave the same across all grids
  (shared component, not copy-pasted per page).
- **Testability:** The tenant-switch ordering guarantee (header set before fetch)
  and the refresh feedback states must be unit-testable with Vitest +
  `@testing-library/react`.
- **No new dependencies.** Reuse the existing toast and React Query primitives.

## 9. Open Questions

- Should the success toast appear on **every** refresh click, or be suppressed when
  the refetch resolves with byte-identical data? Default for this PRD: show a
  success toast on every successful refresh (simplest, and directly answers the
  "did it do anything?" complaint). Revisit if it feels noisy in practice.
- For `LoginHistoryPage`, do we fully migrate to a React Query hook now, or do the
  minimal "reset on tenant change" fix? PRD preference: full React Query migration
  for consistency, but the design phase may choose the minimal path if migration
  proves disproportionately large.
- Is there any grid that intentionally has **no** refresh button and should stay
  that way? Design phase should confirm the inventory so feedback wiring isn't
  forced onto grids that don't refresh.

## 10. Acceptance Criteria

- [ ] Clicking a grid's refresh button shows the icon spinning and the button
      disabled for the duration of the refetch, then returns to idle.
- [ ] A successful refresh surfaces a success toast; a failed refresh surfaces an
      error toast with a meaningful message.
- [ ] Rapidly clicking refresh does not launch overlapping refetches.
- [ ] Refresh feedback is implemented once in the shared grid component and applies
      uniformly to every grid that has a refresh button.
- [ ] Switching tenants results in every React-Query grid showing the new tenant's
      data, with no observable previous-tenant rows, verified across repeated
      switches.
- [ ] No network request carries the previous tenant's headers after a switch
      (the synchronous-set ordering is in place and unit-tested).
- [ ] `LoginHistoryPage` no longer shows previous-tenant results after a switch.
- [ ] The initial-mount no-clear guard and the same-tenant (rename) no-clear guard
      are both preserved and covered by tests.
- [ ] `docs/TODO.md:339` (task-004 R6) is resolved/checked off, with the
      tenant-switch behavior now covered by an automated test.
- [ ] `npm run build`, `npm run lint`, and `npm run test` are clean in
      `services/atlas-ui`.
