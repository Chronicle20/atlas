# Data-Grid Refresh Feedback & Tenant-Switch Staleness Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give every atlas-ui data grid real refresh feedback (spinner + disabled + toast) and make tenant switching deterministically surface the new tenant's data.

**Architecture:** A shared `useGridRefresh` hook standardizes all grids on React Query `refetch()`, derives an `isRefreshing` signal from `isFetching`, and toasts on settle; `DataTable`/`DataTableWrapper` gain an `isRefreshing` prop that spins + disables the button. The tenant-switch race is closed by extracting an `applyTenant` helper that sets `api.setTenant` + `queryClient.clear()` **synchronously** inside the user-action `setActiveTenant` handler (before the state update), with the existing effect kept as the catch-all for programmatic switches. `LoginHistoryPage` (search-driven, not React-Query) resets its local results on tenant change.

**Tech Stack:** Vite + React 19 + React Router, TanStack React Query 5, Sonner toasts, Vitest + `@testing-library/react`. Client-only — no Go/Kafka/DB changes.

**All paths below are under `services/atlas-ui/` unless stated otherwise. Run all `npm`/`npx` commands from `services/atlas-ui/`.**

**Read `context.md` (same folder) first** — it has the full per-file current state, the per-page refresh inventory, and 7 non-obvious gotchas.

---

## Task 1: `useGridRefresh` shared hook

**Files:**
- Create: `src/lib/hooks/useGridRefresh.ts`
- Test: `src/lib/hooks/__tests__/useGridRefresh.test.ts`

- [ ] **Step 1: Write the failing test**

Create `src/lib/hooks/__tests__/useGridRefresh.test.ts`:

```ts
import { renderHook, act } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import * as toast from "@/lib/utils/toast";
import { useGridRefresh, type RefreshableQuery } from "@/lib/hooks/useGridRefresh";

vi.mock("@/lib/utils/toast", () => ({
  success: vi.fn(),
  error: vi.fn(),
}));

function makeQuery(overrides: Partial<RefreshableQuery> = {}): RefreshableQuery {
  return {
    isFetching: false,
    refetch: vi.fn().mockResolvedValue({ isError: false, error: null }),
    ...overrides,
  } as unknown as RefreshableQuery;
}

describe("useGridRefresh", () => {
  beforeEach(() => {
    vi.mocked(toast.success).mockReset();
    vi.mocked(toast.error).mockReset();
  });

  it("isRefreshing is true when any query is fetching", () => {
    const { result } = renderHook(() =>
      useGridRefresh([makeQuery({ isFetching: false }), makeQuery({ isFetching: true })]),
    );
    expect(result.current.isRefreshing).toBe(true);
  });

  it("isRefreshing is false when no query is fetching", () => {
    const { result } = renderHook(() =>
      useGridRefresh([makeQuery(), makeQuery()]),
    );
    expect(result.current.isRefreshing).toBe(false);
  });

  it("onRefresh refetches every query and toasts success once", async () => {
    const q1 = makeQuery();
    const q2 = makeQuery();
    const { result } = renderHook(() => useGridRefresh([q1, q2]));

    await act(async () => {
      await result.current.onRefresh();
    });

    expect(q1.refetch).toHaveBeenCalledTimes(1);
    expect(q2.refetch).toHaveBeenCalledTimes(1);
    expect(toast.success).toHaveBeenCalledTimes(1);
    expect(toast.success).toHaveBeenCalledWith("Data refreshed");
    expect(toast.error).not.toHaveBeenCalled();
  });

  it("uses a custom success message when provided", async () => {
    const { result } = renderHook(() =>
      useGridRefresh([makeQuery()], { successMessage: "Maps refreshed" }),
    );
    await act(async () => {
      await result.current.onRefresh();
    });
    expect(toast.success).toHaveBeenCalledWith("Maps refreshed");
  });

  it("toasts error (not success) when a refetch resolves with isError", async () => {
    const boom = new Error("network down");
    const failing = makeQuery({
      refetch: vi.fn().mockResolvedValue({ isError: true, error: boom }),
    } as Partial<RefreshableQuery>);
    const ok = makeQuery();
    const { result } = renderHook(() => useGridRefresh([ok, failing]));

    await act(async () => {
      await result.current.onRefresh();
    });

    expect(toast.error).toHaveBeenCalledTimes(1);
    expect(toast.error).toHaveBeenCalledWith(boom, { context: { action: "refresh" } });
    expect(toast.success).not.toHaveBeenCalled();
  });
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `npm run test -- src/lib/hooks/__tests__/useGridRefresh.test.ts`
Expected: FAIL — cannot resolve module `@/lib/hooks/useGridRefresh` (file does not exist yet).

- [ ] **Step 3: Write the hook**

Create `src/lib/hooks/useGridRefresh.ts`:

```ts
import type { UseQueryResult } from "@tanstack/react-query";
import * as toast from "@/lib/utils/toast";

/** Minimal slice of a React Query result the refresh hook needs. */
export type RefreshableQuery = Pick<UseQueryResult, "isFetching" | "refetch">;

export interface UseGridRefreshResult {
  isRefreshing: boolean;
  onRefresh: () => Promise<void>;
}

/**
 * Centralizes grid refresh feedback. Accepts the page's query/queries,
 * refetches them in parallel, and surfaces success/error via the app toast.
 *
 * `isRefreshing` is sourced from React Query's own `isFetching` (FR-1.2), not a
 * local timer, so it covers user-clicked and background refetches alike and
 * returns to idle exactly when React Query says fetching ended.
 *
 * NOTE: `refetch()` RESOLVES (it does not reject — React Query v5 default
 * `throwOnError: false`). Error detection therefore inspects each resolved
 * result's `isError`/`error`; do not rely on a thrown exception.
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

- [ ] **Step 4: Run the test to verify it passes**

Run: `npm run test -- src/lib/hooks/__tests__/useGridRefresh.test.ts`
Expected: PASS (5 tests).

- [ ] **Step 5: Commit**

```bash
git add src/lib/hooks/useGridRefresh.ts src/lib/hooks/__tests__/useGridRefresh.test.ts
git commit -m "feat(atlas-ui): add useGridRefresh hook for shared grid refresh feedback"
```

---

## Task 2: `DataTable` `isRefreshing` prop (spinner + disabled)

**Files:**
- Modify: `src/components/data-table.tsx`
- Test: `src/components/__tests__/data-table.test.tsx`

- [ ] **Step 1: Write the failing test**

Create `src/components/__tests__/data-table.test.tsx`:

```tsx
import { render, screen, fireEvent } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { ColumnDef } from "@tanstack/react-table";
import { DataTable } from "@/components/data-table";

type Row = { id: string; name: string };
const columns: ColumnDef<Row>[] = [{ accessorKey: "name", header: "Name" }];
const data: Row[] = [{ id: "1", name: "Alpha" }];

describe("DataTable refresh button", () => {
  it("spins and disables the button while refreshing", () => {
    render(<DataTable columns={columns} data={data} onRefresh={vi.fn()} isRefreshing />);
    const button = screen.getByTitle("Refresh");
    expect(button).toBeDisabled();
    expect(button.querySelector("svg")).toHaveClass("animate-spin");
  });

  it("is enabled and not spinning when not refreshing, and clicking calls onRefresh", () => {
    const onRefresh = vi.fn();
    render(<DataTable columns={columns} data={data} onRefresh={onRefresh} isRefreshing={false} />);
    const button = screen.getByTitle("Refresh");
    expect(button).not.toBeDisabled();
    expect(button.querySelector("svg")).not.toHaveClass("animate-spin");
    fireEvent.click(button);
    expect(onRefresh).toHaveBeenCalledTimes(1);
  });

  it("does not call onRefresh when disabled (refreshing) — no overlapping refetch", () => {
    const onRefresh = vi.fn();
    render(<DataTable columns={columns} data={data} onRefresh={onRefresh} isRefreshing />);
    fireEvent.click(screen.getByTitle("Refresh"));
    expect(onRefresh).not.toHaveBeenCalled();
  });
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `npm run test -- src/components/__tests__/data-table.test.tsx`
Expected: FAIL — `isRefreshing` is not a prop yet, so the button is never disabled and the icon never has `animate-spin` (first two assertions fail).

- [ ] **Step 3: Add the prop and wire the button**

In `src/components/data-table.tsx`:

(a) Add the `cn` import near the other imports at the top:

```tsx
import { cn } from "@/lib/utils";
```

(b) Add `isRefreshing` to the props interface (after `onRefresh`):

```tsx
interface DataTableProps<TData, TValue> {
    initialVisibilityState?: string[]
    columns: ColumnDef<TData, TValue>[]
    data: TData[]
    onRefresh?: () => void
    isRefreshing?: boolean
    headerActions?: DataTableHeaderAction[]
}
```

(c) Destructure it in the component signature (add `isRefreshing,` after `onRefresh,`):

```tsx
export function DataTable<TData, TValue>({
                                             initialVisibilityState,
                                             columns,
                                             data,
                                             onRefresh,
                                             isRefreshing,
                                             headerActions,
                                         }: DataTableProps<TData, TValue>) {
```

(d) Replace the refresh `Button` block (currently lines ~48-57) with:

```tsx
                    {onRefresh && (
                        <Button
                            variant="outline"
                            size="icon"
                            onClick={onRefresh}
                            disabled={isRefreshing}
                            className="hover:bg-accent cursor-pointer"
                            title="Refresh"
                            aria-busy={isRefreshing}
                        >
                            <RefreshCw className={cn("h-4 w-4", isRefreshing && "animate-spin")}/>
                        </Button>
                    )}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `npm run test -- src/components/__tests__/data-table.test.tsx`
Expected: PASS (3 tests).

- [ ] **Step 5: Commit**

```bash
git add src/components/data-table.tsx src/components/__tests__/data-table.test.tsx
git commit -m "feat(atlas-ui): DataTable spins and disables the refresh button while fetching"
```

---

## Task 3: `DataTableWrapper` forwards `isRefreshing`

**Files:**
- Modify: `src/components/common/DataTableWrapper.tsx`

(No new test — covered by the `DataTable` unit test and the page-level wiring. The wrapper is a thin pass-through.)

- [ ] **Step 1: Add `isRefreshing` to the props interface**

In `src/components/common/DataTableWrapper.tsx`, add the prop after `onRefresh?: () => void;`:

```tsx
  onRefresh?: () => void;
  isRefreshing?: boolean;
```

- [ ] **Step 2: Destructure it**

Add `isRefreshing,` to the destructured params (after `onRefresh,`):

```tsx
  onRefresh,
  isRefreshing,
```

- [ ] **Step 3: Forward it to `DataTable` via conditional spread**

In the final `return` (the "Show data table with data" branch), add the forward right after the `onRefresh` spread:

```tsx
      <DataTable
        columns={columns}
        data={data}
        {...(initialVisibilityState && { initialVisibilityState })}
        {...(onRefresh && { onRefresh })}
        {...(typeof isRefreshing === "boolean" && { isRefreshing })}
        {...(headerActions && { headerActions })}
      />
```

- [ ] **Step 4: Verify the build is clean**

Run: `npm run build`
Expected: PASS (`tsc -b` + vite build succeed; the new optional prop type-checks).

- [ ] **Step 5: Commit**

```bash
git add src/components/common/DataTableWrapper.tsx
git commit -m "feat(atlas-ui): DataTableWrapper forwards isRefreshing to DataTable"
```

---

## Task 4: Tenant-switch synchronous `applyTenant` (close the header race)

**Files:**
- Modify: `src/context/tenant-context.tsx`
- Test: `src/context/__tests__/tenant-context.test.tsx` (rewrite test #1 only)

> **Gotcha (context.md #2):** `applyTenant` is called from BOTH the sync handler AND the catch-all effect. Per user switch, `api.setTenant` fires **twice** (sync set + effect same-id re-set for rename propagation) but `queryClient.clear()` fires **once per distinct id**. The existing test #1's exact `setTenant` count assertions must change; the `clearSpy` counts stay (1, then 2). Tests #2 and #3 stay green untouched.

- [ ] **Step 1: Rewrite test #1 to assert synchronous ordering**

In `src/context/__tests__/tenant-context.test.tsx`, replace the entire first `it(...)` block (the one titled "invokes api.setTenant and queryClient.clear on tenant switch (not on initial null mount)") with:

```tsx
  it("sets api.setTenant synchronously and clears the cache once per distinct-id switch", async () => {
    const tenantA = makeTenant("aaa");
    const tenantB = makeTenant("bbb");
    getAllTenantsMock.mockResolvedValueOnce([]);

    const queryClient = new QueryClient();
    const clearSpy = vi.spyOn(queryClient, "clear");

    let ctxRef: ReturnType<typeof useTenant> | undefined;
    render(
      <QueryClientProvider client={queryClient}>
        <TenantProvider>
          <Harness onReady={(c) => { ctxRef = c; }} />
        </TenantProvider>
      </QueryClientProvider>
    );

    await waitFor(() => {
      expect(ctxRef).toBeDefined();
    });

    // Initial mount with activeTenant === null (empty tenant list) fires neither hook.
    expect(setTenantMock).not.toHaveBeenCalled();
    expect(clearSpy).not.toHaveBeenCalled();

    // Switch to tenant A. Asserting INSIDE the act callback (before React commits
    // and the catch-all effect runs) proves headers are set + cache cleared
    // SYNCHRONOUSLY, ahead of the re-render that recomputes child query keys
    // (FR-2.1 / FR-2.4 — closes task-004 R6).
    act(() => {
      ctxRef!.setActiveTenant(tenantA);
      expect(setTenantMock).toHaveBeenCalledWith(tenantA);
      expect(clearSpy).toHaveBeenCalledTimes(1);
    });
    // After commit, the catch-all effect re-applies the same id (no extra clear).
    expect(setTenantMock).toHaveBeenLastCalledWith(tenantA);
    expect(clearSpy).toHaveBeenCalledTimes(1);

    // Switch to tenant B — second distinct id → second clear.
    act(() => {
      ctxRef!.setActiveTenant(tenantB);
      expect(setTenantMock).toHaveBeenLastCalledWith(tenantB);
      expect(clearSpy).toHaveBeenCalledTimes(2);
    });
    expect(setTenantMock).toHaveBeenLastCalledWith(tenantB);
    expect(clearSpy).toHaveBeenCalledTimes(2);
  });
```

Leave the other two tests ("rehydrates activeTenant on refresh…" and "reselects when active tenant was removed…") exactly as they are.

- [ ] **Step 2: Run the test to verify it fails**

Run: `npm run test -- src/context/__tests__/tenant-context.test.tsx`
Expected: FAIL — today `api.setTenant`/`clear` run in an effect, so the assertions INSIDE the `act` callback (synchronous, before the effect flushes) see zero calls.

- [ ] **Step 3: Implement the synchronous `applyTenant` helper**

In `src/context/tenant-context.tsx`:

(a) Add `useCallback` to the React import (first line):

```tsx
import {createContext, type ReactNode, useCallback, useContext, useEffect, useRef, useState} from "react";
```

(b) Replace the existing wiring effect (currently lines ~31-45, the `previousTenantRef` declaration through the `}, [activeTenant, queryClient]);` effect) with:

```tsx
    // Centralise tenant wiring. `applyTenant` updates the API client tenant and
    // clears the React Query cache exactly once per distinct tenant id. Called
    // synchronously from the user-action handler (so headers are set BEFORE any
    // tenant-keyed query can fetch) AND from the catch-all effect below (which
    // covers programmatic switches: initial hydration, refreshTenants,
    // refreshAndSelectTenant). Double calls per switch are idempotent — the
    // same-id branch re-sets headers cheaply without a second cache clear.
    const previousTenantRef = useRef<Tenant | null>(null);

    const applyTenant = useCallback((tenant: Tenant | null) => {
        // Initial null→null mount: nothing to wire, nothing to clear.
        if (tenant === null && previousTenantRef.current === null) {
            return;
        }
        // Same id (e.g. rename rehydrate): wire the fresh object, DO NOT clear.
        if (previousTenantRef.current?.id === tenant?.id) {
            previousTenantRef.current = tenant;
            api.setTenant(tenant);
            return;
        }
        previousTenantRef.current = tenant;
        api.setTenant(tenant);
        queryClient.clear();
    }, [queryClient]);

    // Catch-all for programmatic tenant changes that bypass setActiveTenant
    // (initial hydration, refreshTenants, refreshAndSelectTenant).
    useEffect(() => {
        applyTenant(activeTenant);
    }, [activeTenant, applyTenant]);
```

(c) Replace the `setActiveTenant` handler (currently lines ~66-70) with the synchronous version:

```tsx
    // User-action handler: wire the API client + clear the cache SYNCHRONOUSLY,
    // before the state update re-renders the grids — eliminating the
    // parent-effect-vs-child-fetch header race.
    const setActiveTenant = (tenant: Tenant) => {
        applyTenant(tenant);
        setActiveTenantState(tenant);
        localStorage.setItem(LOCAL_STORAGE_KEY, tenant.id);
    };
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `npm run test -- src/context/__tests__/tenant-context.test.tsx`
Expected: PASS (3 tests — the rewritten #1 plus the two unchanged ones).

- [ ] **Step 5: Commit**

```bash
git add src/context/tenant-context.tsx src/context/__tests__/tenant-context.test.tsx
git commit -m "fix(atlas-ui): set tenant headers + clear cache synchronously on tenant switch"
```

---

## Task 5: `LoginHistoryPage` resets results on tenant change

**Files:**
- Modify: `src/pages/LoginHistoryPage.tsx`
- Test: `src/pages/__tests__/LoginHistoryPage.test.tsx`

- [ ] **Step 1: Write the failing test**

Create `src/pages/__tests__/LoginHistoryPage.test.tsx`:

```tsx
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { LoginHistoryPage } from "@/pages/LoginHistoryPage";

const activeTenant: { current: { id: string } | null } = { current: { id: "aaa" } };
const searchMock = vi.fn();

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({ activeTenant: activeTenant.current }),
}));

vi.mock("@/services/api/login-history.service", () => ({
  loginHistoryService: { search: (...args: unknown[]) => searchMock(...args) },
}));

vi.mock("sonner", () => ({
  Toaster: () => null,
  toast: { error: vi.fn(), info: vi.fn(), success: vi.fn() },
}));

vi.mock("@/components/features/bans/CreateBanDialog", () => ({
  CreateBanDialog: () => null,
}));

describe("LoginHistoryPage tenant switching", () => {
  beforeEach(() => {
    searchMock.mockReset();
    activeTenant.current = { id: "aaa" };
  });

  it("clears prior-tenant results when the active tenant changes", async () => {
    searchMock.mockResolvedValueOnce([
      {
        id: "1",
        attributes: {
          accountId: 1,
          accountName: "Alice",
          ipAddress: "1.1.1.1",
          hwid: "hw1",
          success: true,
          failureReason: "",
        },
      },
    ]);

    const { rerender } = render(<LoginHistoryPage />);

    fireEvent.change(screen.getByLabelText("IP Address"), { target: { value: "1.1.1.1" } });
    fireEvent.click(screen.getByRole("button", { name: /search/i }));

    await waitFor(() => {
      expect(screen.getByText("Results")).toBeInTheDocument();
    });
    expect(searchMock).toHaveBeenCalledTimes(1);

    // Switch tenant — the reset effect must drop the previous tenant's rows.
    activeTenant.current = { id: "bbb" };
    rerender(<LoginHistoryPage />);

    await waitFor(() => {
      expect(screen.queryByText("Results")).not.toBeInTheDocument();
    });
  });
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `npm run test -- src/pages/__tests__/LoginHistoryPage.test.tsx`
Expected: FAIL — without a reset effect, the "Results" card (gated on `hasSearched`) is still mounted after the tenant change, so `queryByText("Results")` still finds it.

- [ ] **Step 3: Add the reset effect**

In `src/pages/LoginHistoryPage.tsx`:

(a) Add `useEffect` to the React import:

```tsx
import { useCallback, useEffect, useState } from "react";
```

(b) Add the reset effect immediately after the `useState` declarations (after the `prefillData` state, before `handleSearch`):

```tsx
    // Reset search results when the active tenant changes so a switch never
    // leaves the previous tenant's rows on screen (FR-2.5).
    useEffect(() => {
        setEntries([]);
        setHasSearched(false);
        setSearchCriteria({ ip: "", hwid: "", accountId: "" });
    }, [activeTenant?.id]);
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `npm run test -- src/pages/__tests__/LoginHistoryPage.test.tsx`
Expected: PASS (1 test).

- [ ] **Step 5: Commit**

```bash
git add src/pages/LoginHistoryPage.tsx src/pages/__tests__/LoginHistoryPage.test.tsx
git commit -m "fix(atlas-ui): reset LoginHistory results on tenant change"
```

---

## Task 6: Wire the `refetch()`-based grids (Maps, Merchants, Services, Gachapons)

**Files:**
- Modify: `src/pages/MapsPage.tsx`
- Modify: `src/pages/MerchantsPage.tsx`
- Modify: `src/pages/ServicesPage.tsx`
- Modify: `src/pages/GachaponsPage.tsx`

These four already hold a refetchable query. Swap each ad-hoc `onRefresh` for the shared hook and pass `isRefreshing`. **Read each file first** to confirm the exact surrounding lines (line numbers below are from the inventory and may have drifted).

- [ ] **Step 1: MapsPage**

Add the import (with the other `@/lib/hooks` / `@/` imports):

```tsx
import { useGridRefresh } from "@/lib/hooks/useGridRefresh";
```

After `const mapsQuery = useQuery<...>({ ... })` (≈L43), add:

```tsx
    const { isRefreshing, onRefresh } = useGridRefresh([mapsQuery]);
```

Replace the `DataTableWrapper`'s `onRefresh={() => mapsQuery.refetch()}` (≈L99) with:

```tsx
        onRefresh={onRefresh}
        isRefreshing={isRefreshing}
```

Leave any existing `mapsQuery.isFetching` / loading wiring untouched.

- [ ] **Step 2: MerchantsPage**

Add the import:

```tsx
import { useGridRefresh } from "@/lib/hooks/useGridRefresh";
```

After `const shopsQuery = useQuery<...>({ ... })` (≈L71), add:

```tsx
    const { isRefreshing, onRefresh } = useGridRefresh([shopsQuery]);
```

Replace `onRefresh={() => shopsQuery.refetch()}` (≈L151) with:

```tsx
        onRefresh={onRefresh}
        isRefreshing={isRefreshing}
```

(`searchResultsQuery` and `tenantConfigQuery` are intentionally NOT included — the toolbar refresh refetches the shops grid, matching today's behavior.)

- [ ] **Step 3: ServicesPage**

Add the import:

```tsx
import { useGridRefresh } from "@/lib/hooks/useGridRefresh";
```

Replace the destructure (≈L18) `const { data: services, isLoading, error, refetch } = useServices();` with:

```tsx
    const servicesQuery = useServices();
    const { data: services, isLoading, error } = servicesQuery;
    const { isRefreshing, onRefresh } = useGridRefresh([servicesQuery]);
```

Replace `onRefresh={refetch}` (≈L63) with:

```tsx
        onRefresh={onRefresh}
        isRefreshing={isRefreshing}
```

Leave the `invalidateAll` from `useInvalidateServices` in place (it is still used by mutation-success handlers).

- [ ] **Step 4: GachaponsPage**

Add the import:

```tsx
import { useGridRefresh } from "@/lib/hooks/useGridRefresh";
```

Replace the destructure (≈L8) `const { data: gachapons, isLoading, error, refetch } = useGachapons();` with:

```tsx
    const gachaponsQuery = useGachapons();
    const { data: gachapons, isLoading, error } = gachaponsQuery;
    const { isRefreshing, onRefresh } = useGridRefresh([gachaponsQuery]);
```

Replace `onRefresh={() => refetch()}` (≈L24) with:

```tsx
        onRefresh={onRefresh}
        isRefreshing={isRefreshing}
```

- [ ] **Step 5: Verify build + lint**

Run: `npm run build && npm run lint`
Expected: PASS. If lint flags an unused `refetch`/local, confirm it was fully removed from the destructure (Steps 3–4 drop it).

- [ ] **Step 6: Commit**

```bash
git add src/pages/MapsPage.tsx src/pages/MerchantsPage.tsx src/pages/ServicesPage.tsx src/pages/GachaponsPage.tsx
git commit -m "feat(atlas-ui): wire refetch-based grids to useGridRefresh feedback"
```

---

## Task 7: Wire Characters + Accounts grids (incl. column row-action refresh)

**Files:**
- Modify: `src/pages/CharactersPage.tsx`
- Modify: `src/pages/AccountsPage.tsx`

Both pages pass `onRefresh` to their column factory (used by row actions) AND to the wrapper. Both must receive the hook's `onRefresh`. **Read each file first.**

- [ ] **Step 1: CharactersPage**

Add the import:

```tsx
import { useGridRefresh } from "@/lib/hooks/useGridRefresh";
```

The page already holds `charactersQuery`, `accountsQuery`, `tenantConfigQuery` (≈L11-13). Replace the `refresh` function (≈L24-27):

```tsx
const refresh = () => {
  invalidateCharacters();
  invalidateAccounts();
};
```

with the hook:

```tsx
    const { isRefreshing, onRefresh } = useGridRefresh([
        charactersQuery,
        accountsQuery,
        tenantConfigQuery,
    ]);
```

Update the column factory call (≈L30) to pass the hook's `onRefresh`:

```tsx
    const columns = getColumns({ tenant: activeTenant, tenantConfig, accountMap, onRefresh });
```

Update the `DataTableWrapper` (≈L48) `onRefresh={refresh}` to:

```tsx
        onRefresh={onRefresh}
        isRefreshing={isRefreshing}
```

Then remove the now-unused `useInvalidateCharacters`/`useInvalidateAccounts` calls (≈L14-15) and their imports — **but first** `grep -n "invalidateCharacters\|invalidateAccounts\|useInvalidateCharacters\|useInvalidateAccounts" src/pages/CharactersPage.tsx` and only remove symbols with no remaining reference (a mutation-success handler may still use one — keep those).

- [ ] **Step 2: AccountsPage**

Add the import:

```tsx
import { useGridRefresh } from "@/lib/hooks/useGridRefresh";
```

The page already holds `accountsQuery` (≈L18). After it, add:

```tsx
    const { isRefreshing, onRefresh } = useGridRefresh([accountsQuery]);
```

Update the column factory call (≈L105-107) `onRefresh: () => invalidateAccounts()` to:

```tsx
        onRefresh,
```

Update the `DataTableWrapper` (≈L133) `onRefresh={() => invalidateAccounts()}` to:

```tsx
        onRefresh={onRefresh}
        isRefreshing={isRefreshing}
```

Remove the now-unused `useInvalidateAccounts`/`invalidateAccounts` (≈L19) only if no other reference remains — `grep -n "invalidateAccounts\|useInvalidateAccounts" src/pages/AccountsPage.tsx` first.

- [ ] **Step 3: Verify build + lint + tests**

Run: `npm run build && npm run lint`
Expected: PASS. (`tsc -b` catches any leftover unused local from the invalidate-hook removal.)

- [ ] **Step 4: Commit**

```bash
git add src/pages/CharactersPage.tsx src/pages/AccountsPage.tsx
git commit -m "feat(atlas-ui): wire Characters + Accounts grids to useGridRefresh feedback"
```

---

## Task 8: Wire Guilds, Bans, Quests, Templates grids

**Files:**
- Modify: `src/pages/GuildsPage.tsx`
- Modify: `src/pages/BansPage.tsx`
- Modify: `src/pages/QuestsPage.tsx`
- Modify: `src/pages/TemplatesPage.tsx`

**Read each file first.** For every page, `grep` the page for each `invalidateAll`/`useInvalidateXxx`/named-refresh symbol before deleting it — some are still used by mutation-success or dialog `onSuccess` handlers and MUST stay.

- [ ] **Step 1: GuildsPage**

Add the import:

```tsx
import { useGridRefresh } from "@/lib/hooks/useGridRefresh";
```

The page holds `guildsQuery`, `charactersQuery`, `tenantConfigQuery` (≈L12-14). Replace the `refresh` function (≈L25-28):

```tsx
const refresh = () => {
  invalidateGuilds();
  invalidateCharacters();
};
```

with:

```tsx
    const { isRefreshing, onRefresh } = useGridRefresh([
        guildsQuery,
        charactersQuery,
        tenantConfigQuery,
    ]);
```

Update `DataTableWrapper` (≈L49) `onRefresh={refresh}` to:

```tsx
        onRefresh={onRefresh}
        isRefreshing={isRefreshing}
```

Remove the now-unused `useInvalidateGuilds`/`useInvalidateCharacters` (≈L15-16) only if unreferenced (grep first).

- [ ] **Step 2: BansPage**

Add the import:

```tsx
import { useGridRefresh } from "@/lib/hooks/useGridRefresh";
```

After `const bansQuery = useBans(...)` (≈L57), add:

```tsx
    const { isRefreshing, onRefresh } = useGridRefresh([bansQuery]);
```

Update `DataTableWrapper` (≈L113) `onRefresh={() => invalidateAll()}` to:

```tsx
        onRefresh={onRefresh}
        isRefreshing={isRefreshing}
```

**KEEP `invalidateAll`/`useInvalidateBans`** — it is still used by `CreateBanDialog`'s `onSuccess` (≈L132).

- [ ] **Step 3: QuestsPage**

Add the import:

```tsx
import { useGridRefresh } from "@/lib/hooks/useGridRefresh";
```

The page holds `questsQuery`, `categoriesQuery` (≈L39-40). After them add:

```tsx
    const { isRefreshing, onRefresh } = useGridRefresh([questsQuery, categoriesQuery]);
```

Update `DataTableWrapper` (≈L96) `onRefresh={() => invalidateAll()}` to:

```tsx
        onRefresh={onRefresh}
        isRefreshing={isRefreshing}
```

Remove `invalidateAll`/`useInvalidateQuests` (≈L41) only if unreferenced (grep first; keep if a mutation handler uses it).

- [ ] **Step 4: TemplatesPage**

Add the import:

```tsx
import { useGridRefresh } from "@/lib/hooks/useGridRefresh";
```

After `const templatesQuery = useTemplates();` (≈L48), add:

```tsx
    const { isRefreshing, onRefresh } = useGridRefresh([templatesQuery]);
```

Update `DataTableWrapper` (≈L217) `onRefresh={fetchDataAgain}` to:

```tsx
        onRefresh={onRefresh}
        isRefreshing={isRefreshing}
```

Remove the `fetchDataAgain` local (≈L89) once its only use is replaced. Keep `invalidateAll`/`useInvalidateTemplates` if a mutation-success handler still uses it (grep `src/pages/TemplatesPage.tsx` for `invalidateAll` / `fetchDataAgain` before deleting either).

- [ ] **Step 5: Verify build + lint**

Run: `npm run build && npm run lint`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add src/pages/GuildsPage.tsx src/pages/BansPage.tsx src/pages/QuestsPage.tsx src/pages/TemplatesPage.tsx
git commit -m "feat(atlas-ui): wire Guilds, Bans, Quests, Templates grids to useGridRefresh feedback"
```

---

## Task 9: Close TODO R6 and run the full verification gate

**Files:**
- Modify: `docs/TODO.md` (repo root, NOT under `services/atlas-ui/`)

- [ ] **Step 1: Locate the TODO R6 line**

Run (from the worktree root): `grep -n "tenant switching invalidates the React Query cache" docs/TODO.md`
Expected: one match, under the `### Tenant-switch invariant (correctness)` heading.

- [ ] **Step 2: Check off the automated-coverage item**

In `docs/TODO.md`, edit that bullet to mark it done and note the test that now covers it. Change:

```markdown
- [ ] Manual smoke test: tenant switching invalidates the React Query cache (new invariant from Phase 2, see `docs/tasks/task-004-atlas-ui-vite-migration/risks.md` R6). The Vitest covers the effect firing; a real-tenant E2E check is still needed.
```

to:

```markdown
- [x] ~~Manual smoke test: tenant switching invalidates the React Query cache (new invariant from Phase 2, see `docs/tasks/task-004-atlas-ui-vite-migration/risks.md` R6).~~ Done (task-091) — the synchronous `applyTenant` set-before-fetch ordering is now covered by an automated unit test in `services/atlas-ui/src/context/__tests__/tenant-context.test.tsx` (assertions run inside the `act` callback, before the re-render). The header-passthrough smoke test below remains.
```

(Leave the second bullet — the `MAJOR_VERSION`/`REGION` header smoke test — as-is; this task does not cover it.)

- [ ] **Step 3: Run the full atlas-ui verification gate**

Run (from `services/atlas-ui/`):

```bash
npm run build && npm run lint && npm run test
```

Expected: all three clean. `npm run test` runs the full Vitest suite including the new `useGridRefresh`, `data-table`, `tenant-context`, and `LoginHistoryPage` tests.

If any pre-existing (unrelated) test was already failing before this task, note it but do not let an unrelated red bar block — confirm the four new/updated test files are green.

- [ ] **Step 4: Commit**

```bash
git add docs/TODO.md
git commit -m "docs(todo): close task-004 R6 — tenant-switch ordering now unit-tested (task-091)"
```

---

## Self-Review checklist (completed during planning)

- **Spec coverage:**
  - FR-1.1/1.4/1.5 → Task 2 (`isRefreshing` prop, disabled, spin) + Task 3 (wrapper forward) + Tasks 6-8 (uniform wiring).
  - FR-1.2 (`isFetching` source) → Task 1 hook.
  - FR-1.3/1.6 (success/error toast via existing mechanism) → Task 1 hook (`@/lib/utils/toast`).
  - FR-2.1/2.4 (header set before fetch; no stale headers) → Task 4 sync `applyTenant` + ordering test.
  - FR-2.2 (cache clear on switch) → Task 4.
  - FR-2.3 (initial-mount no-clear guard) → Task 4 early return; tenant-context tests #1/#2 preserved.
  - FR-2.5 (LoginHistory reset) → Task 5.
  - FR-2.6 (same-tenant rename skips clear) → Task 4 same-id branch; tenant-context test #2 preserved.
  - Acceptance: `docs/TODO.md` R6 → Task 9; build/lint/test gate → Task 9.
- **Placeholder scan:** none — every code step shows full code; the only `grep`-then-decide steps are explicit unused-symbol removals with the exact command given.
- **Type consistency:** `useGridRefresh(queries, options?)` returns `{ isRefreshing, onRefresh }`; the `isRefreshing` prop name is identical across `DataTable`, `DataTableWrapper`, and all 10 pages; `applyTenant(tenant: Tenant | null)` signature is consistent between its definition, the effect call, and the `setActiveTenant` call.
