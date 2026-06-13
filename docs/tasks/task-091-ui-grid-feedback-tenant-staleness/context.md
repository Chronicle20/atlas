# task-091 — Context

Companion to `plan.md`. Captures the key files, decisions, and gotchas an
engineer needs before touching code. All paths are relative to the worktree root
`<repo-root>/.worktrees/task-091-ui-grid-feedback-tenant-staleness`. All
frontend paths are under `services/atlas-ui/`.

## What this task is

Two client-only (atlas-ui) defects, fixed together:

1. **Refresh feedback gap** — the shared `DataTable` refresh button never spins,
   disables, or confirms. Fix: a shared `useGridRefresh` hook (refetch one+
   queries in parallel, derive `isRefreshing` from `isFetching`, toast on
   settle) + an `isRefreshing` prop on `DataTable`/`DataTableWrapper`.
2. **Tenant-switch staleness** — a parent-effect-vs-child-fetch ordering race
   lets an in-flight request carry the *old* tenant's headers while keyed under
   the *new* tenant. Fix: set `api.setTenant` + `queryClient.clear()`
   **synchronously** inside the user-action `setActiveTenant` handler (before the
   React state update), keeping the existing effect as the catch-all for
   programmatic switches. Plus `LoginHistoryPage` resets local results on tenant
   change.

No backend / Go / Kafka / DB changes. No new dependencies. No
`window.location.reload()`. No removal of the `apiClient` singleton.

## Confirmed design decisions (do not re-litigate)

- Refresh feedback is centralized in one `useGridRefresh` hook — not per-page.
- `LoginHistoryPage` **resets local state** on tenant change; it is NOT migrated
  to React Query in this task.
- The success toast fires on **every** successful refresh (no data-diff
  suppression).
- The success message is `"Data refreshed"` (hook default).

## Key files (current state)

| File | Role | Notes |
|---|---|---|
| `src/components/data-table.tsx` | Shared grid + static `RefreshCw` button | `onRefresh?: () => void` at L19; button L48-57, no fetch-state. Add `isRefreshing?: boolean`. |
| `src/components/common/DataTableWrapper.tsx` | Wraps DataTable; owns initial loading/error/empty states | Forwards `onRefresh` via conditional spread (L89). `loading` (L46) renders `PageLoader` — initial load only; do NOT conflate with `isRefreshing`. |
| `src/context/tenant-context.tsx` | Owns active tenant; wires `api.setTenant` + `queryClient.clear()` in an effect (L35-45) | `setActiveTenant` (L67-70) only sets state + localStorage today. Effect is the only writer. `refreshTenants`/`refreshAndSelectTenant` use `setActiveTenantState` directly. |
| `src/context/__tests__/tenant-context.test.tsx` | 3 existing tests | Mocks `api.setTenant` + `tenantsService`. Test #1 asserts exact `setTenant` counts — these change (see Gotchas). |
| `src/lib/api/client.ts` | `apiClient` singleton | `setTenant(tenant)` L80-82, `getTenant()` L84-86, `createHeaders` reads `this.tenant` L88-104. No change needed. |
| `src/lib/utils/toast.ts` | Sonner wrapper | Named exports `success(message, opts)` and `error(unknown, opts)`. `error` transforms API errors to user messages. Import as `import * as toast from "@/lib/utils/toast"`. |
| `src/pages/LoginHistoryPage.tsx` | Search-driven (not list-on-mount); local `entries` state | Has its OWN `<Toaster richColors />` (L282) + imports `toast` from `sonner` directly. No `DataTable` refresh button. `handleClear` (L77-81) already does the reset we want. |
| `src/App.tsx` | Provider stack | Global `<Toaster />` from `@/components/ui/sonner` at L68. Reused by `useGridRefresh` toasts. |
| `docs/TODO.md` | Tracking | Section `### Tenant-switch invariant (correctness)`; first bullet ("Manual smoke test: tenant switching invalidates the React Query cache … a real-tenant E2E check is still needed") is task-004 R6 — check it off. |

## Per-page refresh inventory (10 grids with a refresh button)

Two pre-existing patterns; the fix standardizes both onto `useGridRefresh`.

**`refetch()`-based** (already hold a query object; just swap the callback):
- `MapsPage.tsx` — `mapsQuery` (raw `useQuery`, L43); `onRefresh={() => mapsQuery.refetch()}` L99.
- `MerchantsPage.tsx` — `shopsQuery` (raw `useQuery`, L71); `onRefresh={() => shopsQuery.refetch()}` L151.
- `ServicesPage.tsx` — destructures `{ data, isLoading, error, refetch }` from `useServices()` L18; `onRefresh={refetch}` L63. Must capture the query object to get `isFetching`.
- `GachaponsPage.tsx` — destructures `{ data, isLoading, error, refetch }` from `useGachapons()` L8; `onRefresh={() => refetch()}` L24. Must capture the query object.

**`invalidateAll()`-based** (replace with `useGridRefresh`):
- `CharactersPage.tsx` — `charactersQuery`/`accountsQuery`/`tenantConfigQuery` L11-13; `refresh` calls `invalidateCharacters()`+`invalidateAccounts()` L24-27; passes `onRefresh: refresh` to columns L30 AND `onRefresh={refresh}` L48.
- `AccountsPage.tsx` — `accountsQuery` L18; `invalidateAccounts` L19; passes `onRefresh: () => invalidateAccounts()` to columns L105-107 AND `onRefresh={() => invalidateAccounts()}` L133.
- `GuildsPage.tsx` — `guildsQuery`/`charactersQuery`/`tenantConfigQuery` L12-14; `refresh` calls `invalidateGuilds()`+`invalidateCharacters()` L25-28; `onRefresh={refresh}` L49. Columns get no `onRefresh`.
- `BansPage.tsx` — `bansQuery` L57; `invalidateAll` L58; `onRefresh={() => invalidateAll()}` L113. **Keep `invalidateAll`** — used by `CreateBanDialog` `onSuccess` L132.
- `QuestsPage.tsx` — `questsQuery`/`categoriesQuery` L39-40; `invalidateAll` L41; `onRefresh={() => invalidateAll()}` L96.
- `TemplatesPage.tsx` — `templatesQuery` L48; `invalidateAll` L51; `fetchDataAgain = () => invalidateAll()` L89; `onRefresh={fetchDataAgain}` L217.

`LoginHistoryPage` is NOT in this list (no `DataTable`); it is handled separately
by FR-2.5 (local reset).

### `invalidateAll()` vs `refetch()` (why the swap)

`useInvalidateXxx().invalidateAll` (e.g. `useCharacters.ts:126-136`) returns
`queryClient.invalidateQueries({ queryKey })` — marks queries stale and resolves
**immediately**; it does not await a refetch, so it can't drive a "done" toast.
`refetch()` flips `isFetching` AND returns a settle-able promise carrying
`isError`/`error`. The hook uses `refetch()`. Where a page invalidated a related
resource as a side effect (Characters → Accounts), that query is simply added to
the hook's array. Page mutation-success `invalidateAll()` calls are a separate
concern — leave them in place.

## Gotchas / non-obvious correctness points

1. **`refetch()` resolves, it does not reject** (React Query v5, default
   `throwOnError: false`). The hook must inspect each resolved result's
   `isError`/`error` — NOT rely on a thrown exception. This is the single most
   important subtlety; it has a dedicated unit test.

2. **`tenant-context` test `setTenant` counts change.** The new `applyTenant`
   helper is called from BOTH the sync `setActiveTenant` AND the effect. On a
   user switch: (1) sync `applyTenant(new)` → ref set, `setTenant(new)`,
   `clear()`; (2) re-render → effect `applyTenant(new)` → same-id branch →
   `setTenant(new)` AGAIN (for rename propagation), **no** second clear. So
   `api.setTenant` fires **twice per switch**; `queryClient.clear()` fires
   **once per distinct-id switch** (unchanged). The existing test #1 asserts
   exact `setTenant` counts (1, then 2) — rewrite those to assert
   `toHaveBeenLastCalledWith` + synchronous-ordering instead, and keep the
   `clearSpy` count assertions (1, then 2). Tests #2 and #3 stay green unchanged.

3. **Initial hydration uses `setActiveTenantState`, not `setActiveTenant`.** The
   data-load effect (`tenant-context.tsx:48-64`) sets state directly, so the
   first `null → tenant` still flows through the *effect* → `applyTenant`. The
   `tenant === null && previousTenantRef.current === null` early return preserves
   the "no clear of a fresh cache on mount with no tenants" behavior.

4. **`tsc -b` type-checks `*.test.ts(x)`.** A prop/signature change (e.g.
   `DataTable` gaining `isRefreshing`) and its test updates MUST land in the same
   commit or `npm run build` breaks. (Per
   `reference_atlas_ui_build_typechecks_tests` — test excludes were dropped.)

5. **Removing now-unused invalidate hooks/imports.** After swapping a page to
   `useGridRefresh`, some `useInvalidateXxx`/`invalidateAll` locals may become
   unused → `noUnusedLocals` lint error. `grep` each symbol in the file first;
   remove only if no other reference (mutation-success handlers keep theirs).

6. **`DataTableWrapper.loading` ≠ `isRefreshing`.** `loading` (initial, no data
   yet) renders the full-page `PageLoader`. `isRefreshing` (refetch of already
   rendered data) only spins the button. Keep them distinct; pages pass
   `loading = <query>.isLoading` and `isRefreshing = <hook>.isRefreshing`.

7. **`LoginHistoryPage` keeps its own `sonner` Toaster.** Out of scope to
   de-dupe. It has no `DataTable` refresh button — do not add `useGridRefresh`
   there; only add the reset-on-tenant-change effect.

## Verification gate (`services/atlas-ui/`)

All three must be clean before "done" (per `services/atlas-ui/CLAUDE.md`):

```bash
cd services/atlas-ui
npm run build   # tsc -b + vite — type-checks tests too
npm run lint
npm run test
```

No Go services touched → no `docker buildx bake`, `go test/vet`, or
`redis-key-guard` required.

## Testing tools available

- Vitest + `@testing-library/react` (`renderHook`, `render`, `act`, `waitFor`,
  `fireEvent`) + `jsdom`. Setup: `src/test/setup.ts`.
- Existing example to mirror: `src/context/__tests__/tenant-context.test.tsx`
  (explicit `import { ... } from "vitest"`, `vi.mock` of `@/lib/api/client` and
  `@/services/api`).
