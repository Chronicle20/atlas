# Surface item-tag / seal / incubator in atlas-ui (+ MTS) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an admin page to manage the incubator-rewards pool, and show item-tag ownership + sealing-lock status on character equipment/inventory and on MTS listings.

**Architecture:** Phases A and B are atlas-ui-only (React + TanStack Query + zod). Phase C threads the item `owner` name and lock `flag` from the asset into the atlas-mts listing snapshot, then renders them in the Marketplace. Design: `docs/tasks/task-128-item-tag-seal-incubator/design-ui-surfacing.md`.

**Tech Stack:** atlas-ui = Vite + React Router v7 + TanStack Query 5 + react-hook-form + zod + shadcn/ui + Vitest. Backend = Go (atlas-mts, libs/atlas-saga), GORM.

## Global Constraints

- atlas-ui: run under **nvm 22** (`source nvm; nvm use 22` before `npm`); `npm run build` type-checks `*.test.ts` — update any changed test call sites in the same commit; gate on `npm run build` + `npm run test` + no new lint errors.
- atlas-ui conventions: **named exports** on pages (no default); `@/` alias; compose `api` primitives from `@/lib/api/client` (never reach into `apiClient`); React Query hooks under `src/lib/hooks/api/`; zod schemas under `src/lib/schemas/`; services under `src/services/api/`.
- **DOM-21 / no reinvented constants:** the lock bit is `libs/atlas-constants/asset/flag.go:6` → `FlagLock = 0x01`. Do NOT redefine it in Go; only the UI mirrors `0x01` with a doc comment.
- Go: `go build/vet/test` clean on every changed module; `docker buildx bake atlas-mts` if atlas-mts code changed.
- The lock/expiration semantics: a sealed item has `flag & 0x01`; it may also carry an `expiration` (timed seal) — but a locked item *unlocks* at expiry, it is not destroyed, so a sealed item must read as "Sealed", never "Expires".
- Commit after every task. This work lands on the `task-128-item-tag-seal-incubator` branch (extends PR #909).

---

# Phase A — Incubator-rewards admin page (atlas-ui only)

Backend: per-tenant collection at `/api/tenants/{tenantId}/configurations/incubator-rewards` with per-row CRUD (`GET` list, `POST` create, `PATCH /{id}`, `DELETE /{id}`) + `POST /seed`. Row attributes: `{ itemId, quantity, weight }`, string `id`.

### Task A1: incubator-rewards service

**Files:**
- Create: `services/atlas-ui/src/services/api/incubator-rewards.service.ts`
- Test: `services/atlas-ui/src/services/api/__tests__/incubator-rewards.service.test.ts`

**Interfaces:**
- Produces: `INCUBATOR_REWARDS_RESOURCE_TYPE`, `IncubatorRewardAttributes { itemId: number; quantity: number; weight: number }`, `IncubatorReward { id: string; attributes: IncubatorRewardAttributes }`, `incubatorRewardsService.{ list, create, update, remove, seed }`.

- [ ] **Step 1: Write the failing test**

```ts
import { describe, it, expect, vi, beforeEach } from "vitest";
import { incubatorRewardsService } from "../incubator-rewards.service";
import { api } from "@/lib/api/client";

vi.mock("@/lib/api/client", () => ({
  api: { getList: vi.fn(), post: vi.fn(), patch: vi.fn(), delete: vi.fn() },
}));

describe("incubatorRewardsService", () => {
  const t = "tenant-1";
  beforeEach(() => vi.clearAllMocks());

  it("list GETs the tenant collection", async () => {
    (api.getList as any).mockResolvedValue([]);
    await incubatorRewardsService.list(t);
    expect(api.getList).toHaveBeenCalledWith(`/api/tenants/${t}/configurations/incubator-rewards`, undefined);
  });

  it("create POSTs a JSON:API envelope", async () => {
    (api.post as any).mockResolvedValue({ id: "r1", attributes: { itemId: 2000000, quantity: 1, weight: 50 } });
    await incubatorRewardsService.create(t, { itemId: 2000000, quantity: 1, weight: 50 });
    expect(api.post).toHaveBeenCalledWith(
      `/api/tenants/${t}/configurations/incubator-rewards`,
      { data: { type: "incubator-rewards", attributes: { itemId: 2000000, quantity: 1, weight: 50 } } },
      undefined,
    );
  });

  it("update PATCHes by id with the envelope", async () => {
    (api.patch as any).mockResolvedValue(undefined);
    await incubatorRewardsService.update(t, "r1", { itemId: 3, quantity: 2, weight: 10 });
    expect(api.patch).toHaveBeenCalledWith(
      `/api/tenants/${t}/configurations/incubator-rewards/r1`,
      { data: { id: "r1", type: "incubator-rewards", attributes: { itemId: 3, quantity: 2, weight: 10 } } },
      undefined,
    );
  });

  it("remove DELETEs by id", async () => {
    await incubatorRewardsService.remove(t, "r1");
    expect(api.delete).toHaveBeenCalledWith(`/api/tenants/${t}/configurations/incubator-rewards/r1`, undefined);
  });

  it("seed POSTs the seed endpoint", async () => {
    await incubatorRewardsService.seed(t);
    expect(api.post).toHaveBeenCalledWith(`/api/tenants/${t}/configurations/incubator-rewards/seed`, {}, undefined);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ui && npm run test -- incubator-rewards.service`
Expected: FAIL — module not found.

- [ ] **Step 3: Write the service**

```ts
import { api } from "@/lib/api/client";
import type { ServiceOptions } from "@/lib/api/query-params";

/**
 * Per-tenant incubator reward pool. Backed by the atlas-tenants `configurations`
 * resource exposed as the `incubator-rewards` JSON:API collection:
 *   GET/POST /api/tenants/{tenantId}/configurations/incubator-rewards
 *   PATCH/DELETE .../{incubatorRewardId}
 *   POST .../seed   (repopulate from the seed pool)
 * Writes use the JSON:API envelope {data:{type:"incubator-rewards",...}} — bare bodies 400.
 */
export const INCUBATOR_REWARDS_RESOURCE_TYPE = "incubator-rewards";

export interface IncubatorRewardAttributes {
  itemId: number;
  quantity: number;
  weight: number;
}

export interface IncubatorReward {
  id: string;
  attributes: IncubatorRewardAttributes;
}

function path(tenantId: string): string {
  return `/api/tenants/${tenantId}/configurations/incubator-rewards`;
}

export const incubatorRewardsService = {
  async list(tenantId: string, options?: ServiceOptions): Promise<IncubatorReward[]> {
    return api.getList<IncubatorReward>(path(tenantId), options);
  },
  async create(tenantId: string, attributes: IncubatorRewardAttributes, options?: ServiceOptions): Promise<IncubatorReward> {
    return api.post<IncubatorReward>(
      path(tenantId),
      { data: { type: INCUBATOR_REWARDS_RESOURCE_TYPE, attributes } },
      options,
    );
  },
  async update(tenantId: string, id: string, attributes: IncubatorRewardAttributes, options?: ServiceOptions): Promise<void> {
    await api.patch<void>(
      `${path(tenantId)}/${id}`,
      { data: { id, type: INCUBATOR_REWARDS_RESOURCE_TYPE, attributes } },
      options,
    );
  },
  async remove(tenantId: string, id: string, options?: ServiceOptions): Promise<void> {
    await api.delete<void>(`${path(tenantId)}/${id}`, options);
  },
  async seed(tenantId: string, options?: ServiceOptions): Promise<void> {
    await api.post<void>(`${path(tenantId)}/seed`, {}, options);
  },
};
```

> Note: confirm `api.post`/`api.patch`/`api.delete`/`api.getList` signatures in `src/lib/api/client.ts` (they take `(url, body?, options?)` / `(url, options?)`); adjust the `undefined` trailing arg in the test to match the real arity if the client omits `options`.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-ui && npm run test -- incubator-rewards.service`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/services/api/incubator-rewards.service.ts services/atlas-ui/src/services/api/__tests__/incubator-rewards.service.test.ts
git commit -m "feat(ui): incubator-rewards API service"
```

### Task A2: incubator-rewards React Query hook

**Files:**
- Create: `services/atlas-ui/src/lib/hooks/api/useIncubatorRewards.ts`
- Test: `services/atlas-ui/src/lib/hooks/api/__tests__/useIncubatorRewards.test.tsx`

**Interfaces:**
- Consumes: `incubatorRewardsService` (Task A1).
- Produces: `incubatorRewardsKeys`, `useIncubatorRewards(tenantId)`, `useCreateIncubatorReward()`, `useUpdateIncubatorReward()`, `useDeleteIncubatorReward()`, `useSeedIncubatorRewards()`.

- [ ] **Step 1: Write the failing test** (mirrors `__tests__/useMtsConfig` style — render the hook in a `QueryClientProvider`, assert the list query calls the service and mutations invalidate)

```tsx
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement, type ReactNode } from "react";
import { useIncubatorRewards, useCreateIncubatorReward } from "../useIncubatorRewards";
import { incubatorRewardsService } from "@/services/api/incubator-rewards.service";

vi.mock("@/services/api/incubator-rewards.service", () => ({
  incubatorRewardsService: { list: vi.fn(), create: vi.fn(), update: vi.fn(), remove: vi.fn(), seed: vi.fn() },
}));

function wrapper() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return ({ children }: { children: ReactNode }) => createElement(QueryClientProvider, { client: qc }, children);
}

describe("useIncubatorRewards", () => {
  beforeEach(() => vi.clearAllMocks());

  it("fetches the reward list for a tenant", async () => {
    (incubatorRewardsService.list as any).mockResolvedValue([{ id: "r1", attributes: { itemId: 1, quantity: 1, weight: 5 } }]);
    const { result } = renderHook(() => useIncubatorRewards("t1"), { wrapper: wrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(incubatorRewardsService.list).toHaveBeenCalledWith("t1");
    expect(result.current.data).toHaveLength(1);
  });

  it("create mutation calls the service", async () => {
    (incubatorRewardsService.create as any).mockResolvedValue({ id: "r2", attributes: { itemId: 2, quantity: 1, weight: 3 } });
    const { result } = renderHook(() => useCreateIncubatorReward(), { wrapper: wrapper() });
    await result.current.mutateAsync({ tenantId: "t1", attributes: { itemId: 2, quantity: 1, weight: 3 } });
    expect(incubatorRewardsService.create).toHaveBeenCalledWith("t1", { itemId: 2, quantity: 1, weight: 3 });
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ui && npm run test -- useIncubatorRewards`
Expected: FAIL — module not found.

- [ ] **Step 3: Write the hook**

```tsx
import { useMutation, useQuery, useQueryClient, type UseQueryResult } from "@tanstack/react-query";
import { incubatorRewardsService, type IncubatorReward, type IncubatorRewardAttributes } from "@/services/api/incubator-rewards.service";

export const incubatorRewardsKeys = {
  all: ["incubator-rewards"] as const,
  lists: () => [...incubatorRewardsKeys.all, "list"] as const,
  list: (tenantId: string) => [...incubatorRewardsKeys.lists(), tenantId] as const,
};

export function useIncubatorRewards(tenantId: string): UseQueryResult<IncubatorReward[], Error> {
  return useQuery({
    queryKey: incubatorRewardsKeys.list(tenantId),
    queryFn: () => incubatorRewardsService.list(tenantId),
    enabled: !!tenantId,
    gcTime: 10 * 60 * 1000,
  });
}

export function useCreateIncubatorReward() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ tenantId, attributes }: { tenantId: string; attributes: IncubatorRewardAttributes }) =>
      incubatorRewardsService.create(tenantId, attributes),
    onSettled: (_d, _e, vars) => qc.invalidateQueries({ queryKey: incubatorRewardsKeys.list(vars.tenantId) }),
  });
}

export function useUpdateIncubatorReward() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ tenantId, id, attributes }: { tenantId: string; id: string; attributes: IncubatorRewardAttributes }) =>
      incubatorRewardsService.update(tenantId, id, attributes),
    onSettled: (_d, _e, vars) => qc.invalidateQueries({ queryKey: incubatorRewardsKeys.list(vars.tenantId) }),
  });
}

export function useDeleteIncubatorReward() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ tenantId, id }: { tenantId: string; id: string }) => incubatorRewardsService.remove(tenantId, id),
    onSettled: (_d, _e, vars) => qc.invalidateQueries({ queryKey: incubatorRewardsKeys.list(vars.tenantId) }),
  });
}

export function useSeedIncubatorRewards() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ tenantId }: { tenantId: string }) => incubatorRewardsService.seed(tenantId),
    onSettled: (_d, _e, vars) => qc.invalidateQueries({ queryKey: incubatorRewardsKeys.list(vars.tenantId) }),
  });
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-ui && npm run test -- useIncubatorRewards`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/lib/hooks/api/useIncubatorRewards.ts services/atlas-ui/src/lib/hooks/api/__tests__/useIncubatorRewards.test.tsx
git commit -m "feat(ui): useIncubatorRewards query + mutation hooks"
```

### Task A3: incubator-rewards zod schema

**Files:**
- Create: `services/atlas-ui/src/lib/schemas/incubator-rewards.schema.ts`
- Test: `services/atlas-ui/src/lib/schemas/__tests__/incubator-rewards.schema.test.ts`

**Interfaces:**
- Produces: `incubatorRewardSchema`, `type IncubatorRewardFormData`.

- [ ] **Step 1: Write the failing test**

```ts
import { describe, it, expect } from "vitest";
import { incubatorRewardSchema } from "../incubator-rewards.schema";

describe("incubatorRewardSchema", () => {
  it("accepts positive integers", () => {
    expect(incubatorRewardSchema.safeParse({ itemId: 2000000, quantity: 1, weight: 50 }).success).toBe(true);
  });
  it("rejects zero weight", () => {
    expect(incubatorRewardSchema.safeParse({ itemId: 2000000, quantity: 1, weight: 0 }).success).toBe(false);
  });
  it("rejects non-integer itemId", () => {
    expect(incubatorRewardSchema.safeParse({ itemId: 1.5, quantity: 1, weight: 5 }).success).toBe(false);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ui && npm run test -- incubator-rewards.schema`
Expected: FAIL — module not found.

- [ ] **Step 3: Write the schema**

```ts
import { z } from "zod";

export const incubatorRewardSchema = z.object({
  itemId: z.number({ invalid_type_error: "Item ID is required" }).int("Item ID must be an integer").positive("Item ID must be positive"),
  quantity: z.number({ invalid_type_error: "Quantity is required" }).int("Quantity must be an integer").positive("Quantity must be positive"),
  weight: z.number({ invalid_type_error: "Weight is required" }).int("Weight must be an integer").positive("Weight must be positive"),
});

export type IncubatorRewardFormData = z.infer<typeof incubatorRewardSchema>;
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-ui && npm run test -- incubator-rewards.schema`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/lib/schemas/incubator-rewards.schema.ts services/atlas-ui/src/lib/schemas/__tests__/incubator-rewards.schema.test.ts
git commit -m "feat(ui): incubator-rewards zod schema"
```

### Task A4: incubator-rewards page (table + dialog + seed)

**Files:**
- Create: `services/atlas-ui/src/pages/tenants-incubator-rewards-form.tsx`
- Create: `services/atlas-ui/src/pages/TenantsIncubatorRewardsPage.tsx`
- Test: `services/atlas-ui/src/pages/__tests__/tenants-incubator-rewards-form.test.tsx`

**Interfaces:**
- Consumes: hooks (A2), schema (A3), `ItemNameCell` (`src/pages/` marketplace helper — import from its module; grep `ItemNameCell` to confirm path), `TenantDetailLayout`, shadcn `Table`, `Dialog`, `AlertDialog`, `Form`, `Input`, `Button`, `sonner` `toast`, `createErrorFromUnknown` (`@/lib/api/errors`).
- Produces: `IncubatorRewardsForm`, `TenantsIncubatorRewardsPage`.

- [ ] **Step 1: Write the failing test** — render `IncubatorRewardsForm` in a router + query provider with the hooks mocked; assert the table shows rows with a computed chance %, and that "Add" opens the dialog.

```tsx
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { IncubatorRewardsForm } from "../tenants-incubator-rewards-form";
import * as hooks from "@/lib/hooks/api/useIncubatorRewards";

vi.mock("@/lib/hooks/api/useIncubatorRewards");
// Stub ItemNameCell so the test doesn't need item-data; render the raw itemId.
vi.mock("@/pages/marketplace-columns", () => ({ ItemNameCell: ({ templateId }: { templateId: number }) => <span>item:{templateId}</span> }), { virtual: true });

function renderForm() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={["/tenants/t1/incubator-rewards"]}>
        <Routes><Route path="/tenants/:id/incubator-rewards" element={<IncubatorRewardsForm />} /></Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

const noopMut = { mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false } as any;

describe("IncubatorRewardsForm", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    (hooks.useIncubatorRewards as any).mockReturnValue({ data: [
      { id: "r1", attributes: { itemId: 2000000, quantity: 1, weight: 30 } },
      { id: "r2", attributes: { itemId: 2000001, quantity: 2, weight: 10 } },
    ], isLoading: false });
    (hooks.useCreateIncubatorReward as any).mockReturnValue(noopMut);
    (hooks.useUpdateIncubatorReward as any).mockReturnValue(noopMut);
    (hooks.useDeleteIncubatorReward as any).mockReturnValue(noopMut);
    (hooks.useSeedIncubatorRewards as any).mockReturnValue(noopMut);
  });

  it("renders rows with computed chance %", () => {
    renderForm();
    expect(screen.getByText("75.0%")).toBeInTheDocument(); // 30/40
    expect(screen.getByText("25.0%")).toBeInTheDocument(); // 10/40
  });

  it("opens the add dialog", async () => {
    renderForm();
    await userEvent.click(screen.getByRole("button", { name: /add/i }));
    await waitFor(() => expect(screen.getByLabelText(/item id/i)).toBeInTheDocument());
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ui && npm run test -- tenants-incubator-rewards-form`
Expected: FAIL — module not found.

- [ ] **Step 3: Implement the form + page**

Model the file on `src/pages/tenants-mts-config-form.tsx` (imports, `useParams`, loading/empty guards, toast + `createErrorFromUnknown`) and the marketplace table for the `<Table>` shell. Key requirements the test pins:
- `const { id: tenantId = "" } = useParams();`
- `const rewards = useIncubatorRewards(tenantId).data ?? [];`
- `const totalWeight = rewards.reduce((s, r) => s + r.attributes.weight, 0);`
- Chance cell: `totalWeight > 0 ? ((r.attributes.weight / totalWeight) * 100).toFixed(1) + "%" : "—"`.
- Header: `<Button onClick={openAdd}>Add</Button>` and a **Seed defaults** `<Button>` that opens an `AlertDialog` confirm → `seedMut.mutate({ tenantId })`.
- Add/Edit: a single `<Dialog>` with `useForm<IncubatorRewardFormData>({ resolver: zodResolver(incubatorRewardSchema) })`, three `<FormField>` number inputs (`itemId`, `quantity`, `weight`) with `<FormLabel>` text "Item ID"/"Quantity"/"Weight" and `onChange={e => field.onChange(e.target.valueAsNumber)}`; on submit call create (no editing id) or update (with id); toast success/error.
- Per-row: `Edit` button (prefill the dialog) and a delete `<Button>` opening an `AlertDialog` → `removeMut.mutate({ tenantId, id })`.
- Item column: `<ItemNameCell templateId={r.attributes.itemId} />` (import from the marketplace columns module — grep `export function ItemNameCell` / `export const ItemNameCell` to get the exact path; the mock path in the test must match it).

Then the page wrapper `TenantsIncubatorRewardsPage.tsx`:

```tsx
import { TenantDetailLayout } from "@/components/features/tenants/TenantDetailLayout";
import { IncubatorRewardsForm } from "./tenants-incubator-rewards-form";

export function TenantsIncubatorRewardsPage() {
  return (
    <TenantDetailLayout>
      <IncubatorRewardsForm />
    </TenantDetailLayout>
  );
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-ui && npm run test -- tenants-incubator-rewards-form`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/pages/tenants-incubator-rewards-form.tsx services/atlas-ui/src/pages/TenantsIncubatorRewardsPage.tsx services/atlas-ui/src/pages/__tests__/tenants-incubator-rewards-form.test.tsx
git commit -m "feat(ui): incubator-rewards admin page (table + dialog + seed)"
```

### Task A5: wire route + nav link

**Files:**
- Modify: `services/atlas-ui/src/App.tsx` (lazy import near line 65; `<Route>` near line 128)
- Modify: `services/atlas-ui/src/components/features/tenants/TenantDetailLayout.tsx` (sidebar array ~lines 12-20)

**Interfaces:**
- Consumes: `TenantsIncubatorRewardsPage` (A4).

- [ ] **Step 1: Add the lazy import** in `App.tsx` next to the other `TenantsMtsConfigPage` lazy import:

```tsx
const TenantsIncubatorRewardsPage = lazy(() =>
  import("@/pages/TenantsIncubatorRewardsPage").then((m) => ({ default: m.TenantsIncubatorRewardsPage })),
);
```

- [ ] **Step 2: Add the route** inside the `AppShell` route group next to the `mts-config` route:

```tsx
<Route path="/tenants/:id/incubator-rewards" element={<TenantsIncubatorRewardsPage />} />
```

- [ ] **Step 3: Add the nav link** in `TenantDetailLayout.tsx`, right after the MTS Configuration entry:

```tsx
{ title: "Incubator Rewards", href: `/tenants/${id}/incubator-rewards` },
```

- [ ] **Step 4: Build to verify wiring + types**

Run: `cd services/atlas-ui && npm run build`
Expected: build succeeds (tsc + vite).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/App.tsx services/atlas-ui/src/components/features/tenants/TenantDetailLayout.tsx
git commit -m "feat(ui): route + nav for incubator-rewards page"
```

---

# Phase B — Inventory tag/seal indicators (atlas-ui only)

### Task B1: type the `owner` field + asset-flags util

**Files:**
- Modify: `services/atlas-ui/src/services/api/inventory.service.ts` (Asset.attributes, after `ownerId`)
- Create: `services/atlas-ui/src/lib/utils/asset-flags.ts`
- Test: `services/atlas-ui/src/lib/utils/__tests__/asset-flags.test.ts`

**Interfaces:**
- Produces: `FLAG_LOCK`, `ZERO_DATE`, `isSealed(a: Asset)`, `isTagged(a: Asset)`.

- [ ] **Step 1: Write the failing test**

```ts
import { describe, it, expect } from "vitest";
import { isSealed, isTagged, FLAG_LOCK } from "../asset-flags";
import type { Asset } from "@/services/api/inventory.service";

function asset(over: Partial<Asset["attributes"]>): Asset {
  return { type: "assets", id: "1", attributes: { flag: 0, owner: "", expiration: "", templateId: 1, id: 1, slot: 0, createdAt: "", quantity: 1, ownerId: 0, rechargeable: 0, strength: 0, dexterity: 0, intelligence: 0, luck: 0, hp: 0, mp: 0, weaponAttack: 0, magicAttack: 0, weaponDefense: 0, magicDefense: 0, accuracy: 0, avoidability: 0, hands: 0, speed: 0, jump: 0, slots: 0, levelType: 0, level: 0, experience: 0, hammersApplied: 0, equippedSince: "", cashId: "", commodityId: 0, purchaseBy: 0, petId: 0, ...over } };
}

describe("asset-flags", () => {
  it("FLAG_LOCK is 0x01", () => expect(FLAG_LOCK).toBe(0x01));
  it("isSealed true when lock bit set", () => expect(isSealed(asset({ flag: 0x01 }))).toBe(true));
  it("isSealed false when lock bit clear", () => expect(isSealed(asset({ flag: 0x02 }))).toBe(false));
  it("isTagged true when owner non-empty", () => expect(isTagged(asset({ owner: "Chronicle" }))).toBe(true));
  it("isTagged false when owner empty/whitespace", () => expect(isTagged(asset({ owner: "  " }))).toBe(false));
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ui && npm run test -- asset-flags`
Expected: FAIL — module not found / `owner` not on `Asset`.

- [ ] **Step 3a: Add `owner` to the Asset type.** In `inventory.service.ts`, in `Asset.attributes`, add after the `ownerId: number;` line:

```ts
    owner: string;
```

- [ ] **Step 3b: Write the util** `src/lib/utils/asset-flags.ts`:

```ts
import type { Asset } from "@/services/api/inventory.service";

/** Asset flag bit for a sealing lock. Mirrors libs/atlas-constants/asset/flag.go:6 (FlagLock = 0x01). */
export const FLAG_LOCK = 0x01;

/** Sentinel the backend emits for "no expiration". */
export const ZERO_DATE = "0001-01-01T00:00:00Z";

export function isSealed(a: Asset): boolean {
  return (a.attributes.flag & FLAG_LOCK) !== 0;
}

export function isTagged(a: Asset): boolean {
  return a.attributes.owner.trim() !== "";
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-ui && npm run test -- asset-flags`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/services/api/inventory.service.ts services/atlas-ui/src/lib/utils/asset-flags.ts services/atlas-ui/src/lib/utils/__tests__/asset-flags.test.ts
git commit -m "feat(ui): type asset owner + asset-flags helpers (isSealed/isTagged)"
```

### Task B2: tooltip owner + seal lines (suppress EXPIRES when sealed)

**Files:**
- Modify: `services/atlas-ui/src/components/features/characters/AssetTooltipContent.tsx` (expiration block ~lines 165-178)
- Test: `services/atlas-ui/src/components/features/characters/__tests__/AssetTooltipContent.test.tsx` (create or extend)

**Interfaces:**
- Consumes: `isSealed`, `isTagged`, `ZERO_DATE` (B1).

- [ ] **Step 1: Write the failing test** — render the tooltip for four cases and assert the lines.

```tsx
import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { AssetTooltipContent } from "../AssetTooltipContent";
import type { Asset } from "@/services/api/inventory.service";

function asset(over: Partial<Asset["attributes"]>): Asset {
  return { type: "assets", id: "1", attributes: { flag: 0, owner: "", expiration: "", templateId: 1040000, id: 1, slot: 0, createdAt: "", quantity: 1, ownerId: 0, rechargeable: 0, strength: 0, dexterity: 0, intelligence: 0, luck: 0, hp: 0, mp: 0, weaponAttack: 0, magicAttack: 0, weaponDefense: 0, magicDefense: 0, accuracy: 0, avoidability: 0, hands: 0, speed: 0, jump: 0, slots: 7, levelType: 0, level: 0, experience: 0, hammersApplied: 0, equippedSince: "", cashId: "", commodityId: 0, purchaseBy: 0, petId: 0, ...over } };
}

describe("AssetTooltipContent tag/seal", () => {
  it("shows owner when tagged", () => {
    render(<AssetTooltipContent asset={asset({ owner: "Chronicle" })} />);
    expect(screen.getByText(/Chronicle/)).toBeInTheDocument();
  });
  it("shows 'Sealed' (no date) for permanent seal and no EXPIRES", () => {
    render(<AssetTooltipContent asset={asset({ flag: 0x01, expiration: "0001-01-01T00:00:00Z" })} />);
    expect(screen.getByText(/SEALED/i)).toBeInTheDocument();
    expect(screen.queryByText(/EXPIRES/i)).not.toBeInTheDocument();
  });
  it("shows 'Sealed until' for a timed seal and suppresses EXPIRES", () => {
    render(<AssetTooltipContent asset={asset({ flag: 0x01, expiration: "2026-08-01T00:00:00Z" })} />);
    expect(screen.getByText(/SEALED UNTIL/i)).toBeInTheDocument();
    expect(screen.queryByText(/EXPIRES/i)).not.toBeInTheDocument();
  });
  it("keeps EXPIRES for a non-sealed timed item", () => {
    render(<AssetTooltipContent asset={asset({ flag: 0, expiration: "2026-08-01T00:00:00Z" })} />);
    expect(screen.getByText(/EXPIRES/i)).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ui && npm run test -- AssetTooltipContent`
Expected: FAIL — no "SEALED" text.

- [ ] **Step 3: Edit the tooltip.** Add the import at the top:

```tsx
import { isSealed, isTagged } from "@/lib/utils/asset-flags";
```

Replace the existing expiration block (currently):

```tsx
      {a.expiration && a.expiration !== "" && a.expiration !== ZERO_DATE && (
        <div className="text-xs">
          <span className="text-muted-foreground">EXPIRES: </span>
          <span>{new Date(a.expiration).toLocaleDateString()}</span>
        </div>
      )}
```

with (note: `a` is `asset.attributes`; the helpers take the `asset` prop):

```tsx
      {isTagged(asset) && (
        <div className="text-xs">
          <span className="text-muted-foreground">OWNER: </span>
          <span>{a.owner}</span>
        </div>
      )}

      {isSealed(asset) ? (
        <div className="text-xs">
          <span className="text-muted-foreground">
            {a.expiration && a.expiration !== "" && a.expiration !== ZERO_DATE ? "SEALED UNTIL: " : "SEALED"}
          </span>
          {a.expiration && a.expiration !== "" && a.expiration !== ZERO_DATE && (
            <span>{new Date(a.expiration).toLocaleDateString()}</span>
          )}
        </div>
      ) : (
        a.expiration && a.expiration !== "" && a.expiration !== ZERO_DATE && (
          <div className="text-xs">
            <span className="text-muted-foreground">EXPIRES: </span>
            <span>{new Date(a.expiration).toLocaleDateString()}</span>
          </div>
        )
      )}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-ui && npm run test -- AssetTooltipContent`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/components/features/characters/AssetTooltipContent.tsx services/atlas-ui/src/components/features/characters/__tests__/AssetTooltipContent.test.tsx
git commit -m "feat(ui): tooltip owner + seal lines; suppress EXPIRES when sealed"
```

### Task B3: EquipmentCell badges + gold ring

**Files:**
- Modify: `services/atlas-ui/src/components/features/characters/EquipmentCell.tsx`
- Test: `services/atlas-ui/src/components/features/characters/__tests__/EquipmentCell.test.tsx` (create or extend)

**Interfaces:**
- Consumes: `isSealed`, `isTagged` (B1); lucide `Lock`, `Tag`; `cn` (`@/lib/utils`).

- [ ] **Step 1: Write the failing test**

```tsx
import { describe, it, expect } from "vitest";
import { render } from "@testing-library/react";
import { EquipmentCell } from "../EquipmentCell";
import type { Asset } from "@/services/api/inventory.service";
import type { Tenant } from "@/services/api/tenants.service";

const tenant = { id: "t1", attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 } } as unknown as Tenant;
function asset(over: Partial<Asset["attributes"]>): Asset { /* same builder as B1 test, templateId 1040000 */ 
  return { type: "assets", id: "1", attributes: { flag: 0, owner: "", expiration: "", templateId: 1040000, id: 1, slot: -5, createdAt: "", quantity: 1, ownerId: 0, rechargeable: 0, strength: 0, dexterity: 0, intelligence: 0, luck: 0, hp: 0, mp: 0, weaponAttack: 0, magicAttack: 0, weaponDefense: 0, magicDefense: 0, accuracy: 0, avoidability: 0, hands: 0, speed: 0, jump: 0, slots: 0, levelType: 0, level: 0, experience: 0, hammersApplied: 0, equippedSince: "", cashId: "", commodityId: 0, purchaseBy: 0, petId: 0, ...over } };
}

describe("EquipmentCell indicators", () => {
  it("renders a lock icon when sealed", () => {
    const { container } = render(<EquipmentCell slotId={-5} slotName="Top" asset={asset({ flag: 0x01 })} tenant={tenant} />);
    expect(container.querySelector('[data-testid="seal-icon"]')).toBeTruthy();
  });
  it("renders a tag icon when tagged", () => {
    const { container } = render(<EquipmentCell slotId={-5} slotName="Top" asset={asset({ owner: "Chronicle" })} tenant={tenant} />);
    expect(container.querySelector('[data-testid="tag-icon"]')).toBeTruthy();
  });
  it("renders neither when plain", () => {
    const { container } = render(<EquipmentCell slotId={-5} slotName="Top" asset={asset({})} tenant={tenant} />);
    expect(container.querySelector('[data-testid="seal-icon"]')).toBeFalsy();
    expect(container.querySelector('[data-testid="tag-icon"]')).toBeFalsy();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ui && npm run test -- EquipmentCell`
Expected: FAIL — no seal-icon.

- [ ] **Step 3: Edit EquipmentCell.** Add imports:

```tsx
import { Lock, Tag } from "lucide-react";
import { cn } from "@/lib/utils";
import { isSealed, isTagged } from "@/lib/utils/asset-flags";
```

In the filled (`asset ?`) branch, change the outer container to add the gold ring when sealed and wrap the `<img>` in a `relative` container with corner badges:

```tsx
    <div className={cn("aspect-square border rounded", asset && isSealed(asset) && "ring-1 ring-amber-400/60")}>
```

and replace the `<img .../>` with:

```tsx
                <div className="relative w-full h-full">
                  <img
                    src={getAssetIconUrl(
                      tenant.id, tenant.attributes.region,
                      tenant.attributes.majorVersion, tenant.attributes.minorVersion,
                      "item", asset.attributes.templateId,
                    )}
                    alt={itemName ?? slotName}
                    className="w-full h-full object-contain"
                  />
                  {isTagged(asset) && (
                    <Tag data-testid="tag-icon" className="absolute top-0 right-0 h-3 w-3 text-amber-500" aria-label="Named item" />
                  )}
                  {isSealed(asset) && (
                    <Lock data-testid="seal-icon" className="absolute bottom-0 right-0 h-3 w-3 text-amber-500" aria-label="Sealed item" />
                  )}
                </div>
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-ui && npm run test -- EquipmentCell`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/components/features/characters/EquipmentCell.tsx services/atlas-ui/src/components/features/characters/__tests__/EquipmentCell.test.tsx
git commit -m "feat(ui): tag/seal badges + gold ring on EquipmentCell"
```

### Task B4: InventoryCard badges + gold ring

**Files:**
- Modify: `services/atlas-ui/src/components/features/characters/InventoryCard.tsx`
- Test: `services/atlas-ui/src/components/features/characters/__tests__/InventoryCard.test.tsx` (create or extend)

**Interfaces:**
- Consumes: `isSealed`, `isTagged` (B1); lucide `Lock`, `Tag`; `cn`.

- [ ] **Step 1: Read `InventoryCard.tsx`** to locate the item-image container and the card root. It renders one card per `asset`.

- [ ] **Step 2: Write the failing test** — mirror B3's three cases (`seal-icon`, `tag-icon`, neither) against `InventoryCard` with its required props (read the component's `Props` to supply them).

- [ ] **Step 3: Run test to verify it fails**

Run: `cd services/atlas-ui && npm run test -- InventoryCard`
Expected: FAIL.

- [ ] **Step 4: Add the same badges + gold ring** as B3: import `Lock`/`Tag`/`cn`/`isSealed`/`isTagged`; add `ring-1 ring-amber-400/60` (via `cn`) to the card root when `isSealed(asset)`; add the two absolute-positioned `data-testid` icons over the item image (`Tag` top-right, `Lock` bottom-right).

- [ ] **Step 5: Run test to verify it passes**

Run: `cd services/atlas-ui && npm run test -- InventoryCard`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-ui/src/components/features/characters/InventoryCard.tsx services/atlas-ui/src/components/features/characters/__tests__/InventoryCard.test.tsx
git commit -m "feat(ui): tag/seal badges + gold ring on InventoryCard"
```

---

# Phase C — MTS listing tag/seal indicators (backend + atlas-ui)

The listing snapshot must carry the item's tag `owner` and lock `flag`. The listing already has `flags` end-to-end (model → Transform → REST → UI), so **first verify whether `listing.flags` already reflects the equip `FlagLock`** (Task C1). Then add `owner` (Tasks C2-C4) and render both in the UI (Task C5).

### Task C1: verify flag carry + locate the snapshot capture site

**Files:** (investigation only)

- [ ] **Step 1: Trace the listing snapshot build.** From the seller's asset to the persisted listing: grep the transfer/custody path for where the item stat snapshot (`flags`, stats) is written onto the listing entity.

Run:
```bash
grep -rn "Flags\|flags\b" services/atlas-mts/atlas.com/mts --include=*.go | grep -iv _test | grep -i "flag"
grep -rn "AcceptToMtsListing\|snapshot\|TransferToMts\|SetFlags\|flags:" services/atlas-mts services/atlas-saga-orchestrator services/atlas-channel libs/atlas-saga --include=*.go | grep -iv _test | head -40
```

- [ ] **Step 2: Record the finding** in a scratch note in the task folder (`docs/tasks/task-128-item-tag-seal-incubator/mts-owner-flag-notes.md`): the exact file:line where the listing entity's `flags`/stat snapshot is populated from the asset. If `flags` is populated from the asset flag there, the seal side is already correct and only `owner` needs adding at the SAME site; if not, `flags` must be populated there too (alongside `owner`).

- [ ] **Step 3: Commit the note**

```bash
git add docs/tasks/task-128-item-tag-seal-incubator/mts-owner-flag-notes.md
git commit -m "docs(task-128): locate MTS listing snapshot capture site for owner/flag"
```

### Task C2: add `owner` to the atlas-mts listing model

**Files:**
- Modify: `services/atlas-mts/atlas.com/mts/listing/model.go` (struct field ~line 80 area; getter next to `Flags()` at line 137; builder)
- Modify: the listing entity (DB struct — grep `type Entity` / gorm tags in the listing package) to add an `owner` column
- Test: `services/atlas-mts/atlas.com/mts/listing/model_test.go` (create or extend)

**Interfaces:**
- Produces: `Model.Owner() string`, builder `SetOwner(string)`, entity `Owner` column.

- [ ] **Step 1: Write the failing test**

```go
func TestModelOwner(t *testing.T) {
	m := NewBuilder(/* required args as the existing builder needs */).SetOwner("Chronicle").Build()
	if m.Owner() != "Chronicle" {
		t.Fatalf("Owner() = %q, want Chronicle", m.Owner())
	}
}
```

(Read the existing `model_test.go` / builder constructor to supply the required builder args; mirror an existing builder test.)

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-mts/atlas.com/mts && go test ./listing/ -run TestModelOwner`
Expected: FAIL — `Owner`/`SetOwner` undefined.

- [ ] **Step 3: Implement.** Add `owner string` to the `Model` struct (near `flags uint16`), `func (m Model) Owner() string { return m.owner }` next to `Flags()`, a builder field + `SetOwner`, and thread `owner` through `Build()` and any model↔entity mapping (add an `owner` column with a gorm tag to the entity struct, and map it both directions in the entity→model / model→entity functions).

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-mts/atlas.com/mts && go test ./listing/ -run TestModelOwner`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-mts/atlas.com/mts/listing/
git commit -m "feat(mts): add owner to listing model + entity"
```

### Task C3: expose `owner` on the listing REST model

**Files:**
- Modify: `services/atlas-mts/atlas.com/mts/listing/rest.go` (`RestModel` add field after `Flags` line 45; `Transform` add mapping after line 157)
- Test: `services/atlas-mts/atlas.com/mts/listing/rest_test.go` (create or extend)

**Interfaces:**
- Consumes: `Model.Owner()` (C2).
- Produces: `RestModel.Owner string json:"owner"`.

- [ ] **Step 1: Write the failing test**

```go
func TestTransformOwner(t *testing.T) {
	m := NewBuilder(/* required args */).SetOwner("Chronicle").Build()
	rm, err := Transform(m)
	if err != nil { t.Fatal(err) }
	if rm.Owner != "Chronicle" {
		t.Fatalf("rm.Owner = %q, want Chronicle", rm.Owner)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-mts/atlas.com/mts && go test ./listing/ -run TestTransformOwner`
Expected: FAIL — `rm.Owner` undefined.

- [ ] **Step 3: Implement.** In `rest.go`, add to `RestModel` (after the `Flags uint16 json:"flags"` line):

```go
	Owner string `json:"owner"`
```

and in `Transform`, add (after `Flags: m.Flags(),`):

```go
		Owner: m.Owner(),
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-mts/atlas.com/mts && go test ./listing/ -run TestTransformOwner`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-mts/atlas.com/mts/listing/rest.go services/atlas-mts/atlas.com/mts/listing/rest_test.go
git commit -m "feat(mts): expose listing owner on REST model"
```

### Task C4: capture owner (+ flag) at listing creation

**Files:**
- Modify: `libs/atlas-saga/payloads.go` (the TransferToMts listing-snapshot payload struct near lines 655-675 — add `Owner string json:"owner"`; confirm `Flags uint16` is present)
- Modify: `libs/atlas-saga/unmarshal.go` / `unmarshal_test.go` if the payload is in the discriminated set
- Modify: the snapshot-capture site found in Task C1 (populate `owner` — and `flags` if C1 showed it wasn't already — onto the listing from the seller's asset)
- Modify: the payload producer (atlas-channel or atlas-saga-orchestrator) that builds the TransferToMts payload from the asset — set `Owner` from the asset's `owner`
- Test: a payload round-trip test + the capture-site unit test (extend the nearest existing test in the touched package)

**Interfaces:**
- Consumes: asset `owner`/`flag` (already on the inventory asset model and its AssetData mirrors — task-128 threaded `owner` through storage/merchant AssetData; confirm the MTS transfer AssetData carries it, add if missing).
- Produces: `owner` populated on the created listing.

- [ ] **Step 1: Write the failing test** — at the capture site, given an asset with `owner="Chronicle"` and `flag=0x01`, assert the built listing model has `Owner()=="Chronicle"` and `Flags()&0x01 != 0`. (Extend the nearest existing creation/expansion test; if none, add a focused unit test around the snapshot-build function.)

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./... -run <the new test>` in the touched module.
Expected: FAIL — owner empty.

- [ ] **Step 3: Implement.** Add `Owner string json:"owner"` to the TransferToMts snapshot payload; set it from the asset owner where the payload is produced; at the capture site, `SetOwner(payload.Owner)` (and `SetFlags(payload.Flags)`/`asset.flag` if C1 showed flags weren't carried) on the listing builder.

- [ ] **Step 4: Run tests**

Run: `go test ./...` in each touched module (libs/atlas-saga, atlas-mts, and the producer service).
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-saga services/atlas-mts services/atlas-channel services/atlas-saga-orchestrator
git commit -m "feat(mts): thread item owner (+lock flag) into the listing snapshot"
```

### Task C5: render owner + lock on Marketplace rows

**Files:**
- Modify: `services/atlas-ui/src/services/api/mts-listings.service.ts` (`MtsListingAttributes` add `owner: string`)
- Modify: the Marketplace item cell (`src/pages/MarketplacePage.tsx` and/or its `ItemNameCell` in `marketplace-columns` — grep to confirm)
- Test: extend the marketplace-row/`ItemNameCell` test (create if none)

**Interfaces:**
- Consumes: `FLAG_LOCK` from `@/lib/utils/asset-flags` (B1); lucide `Lock`, `Tag`.

- [ ] **Step 1: Write the failing test** — render the item cell with a listing whose `flags & 0x01` set and `owner="Chronicle"`; assert a `data-testid="seal-icon"` and the owner text render; and neither when `flags=0`/`owner=""`.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ui && npm run test -- <marketplace test name>`
Expected: FAIL.

- [ ] **Step 3: Implement.**
- Add `owner: string;` to `MtsListingAttributes`.
- In the item cell, render `<Tag data-testid="tag-icon" ... />` + the owner text when `owner.trim() !== ""`, and `<Lock data-testid="seal-icon" ... />` when `(flags & FLAG_LOCK) !== 0` (import `FLAG_LOCK` from `@/lib/utils/asset-flags`).

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-ui && npm run test -- <marketplace test name>`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/services/api/mts-listings.service.ts services/atlas-ui/src/pages/
git commit -m "feat(ui): show tag owner + seal lock on Marketplace listings"
```

---

# Final verification

### Task V1: full verification + review

- [ ] **Step 1: atlas-ui build + test**

Run: `cd services/atlas-ui && source ~/.nvm/nvm.sh && nvm use 22 && npm run build && npm run test`
Expected: build + all tests pass; no new lint errors (`npm run lint` — compare to baseline).

- [ ] **Step 2: Go build/vet/test on changed modules**

Run (per touched module — atlas-mts, libs/atlas-saga, and any producer service):
```bash
go build ./... && go vet ./... && go test ./...
```
Expected: clean.

- [ ] **Step 3: Docker bake atlas-mts** (its Go code changed)

Run: `docker buildx bake atlas-mts`
Expected: success.

- [ ] **Step 4: Code review** — run `superpowers:requesting-code-review` (dispatches backend-guidelines for the Go changes + frontend-guidelines for the TS changes + plan-adherence against this plan). Address findings.

- [ ] **Step 5: Push** the branch (updates PR #909) and confirm CI + the `deploy-env` lane are green.

---

## Notes for the implementer

- The exact `api` client method arities (`post(url, body, options?)` etc.) are in `src/lib/api/client.ts` — verify and match; the service test's trailing `undefined` args must reflect the real signatures.
- `ItemNameCell` (Marketplace) resolves an item template id → name; reuse it for the incubator table's Item column and grep its exact export path for the import + the test mock.
- Jest-era test files exist; new tests use Vitest (`vi.*`) and live under `__tests__/`. `tsc -b` type-checks non-excluded tests — keep new tests type-correct.
- Do not reintroduce the "known follow-up" language anywhere; the tag/seal/updateTime backend work is landed.
