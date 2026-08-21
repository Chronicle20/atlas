# Refreshable Empty States — Implementation Plan

Task: `task-258-refreshable-empty-states`
Spec: `docs/tasks/task-258-refreshable-empty-states/design.md`
PRD: `docs/tasks/task-258-refreshable-empty-states/prd.md`
Created: 2026-08-21

Module root for every task: `services/atlas-ui`. Test command:
`npm test -- <path>` (vitest). Type-check: `npx tsc -b`.

Task order matters: Tasks 1–3 change the shared contracts every later task
consumes. Tasks 4–9 are independent of each other once 1–3 have landed.

---

## Task 1: `useGridRefresh` — structural `RefreshableQuery` + `lastUpdatedAt`

Implements D1, D2 (FR-5.1 … FR-5.5).

### Files

- `services/atlas-ui/src/lib/hooks/useGridRefresh.ts` — replace the `Pick`
  alias with a hand-written interface; add `lastUpdatedAt` to the return
- `services/atlas-ui/src/lib/hooks/__tests__/useGridRefresh.test.ts` — extend
  `makeQuery` with `dataUpdatedAt`; add the min/null cases

Patterns to copy: the existing suite in the same test file (mock of
`@/lib/utils/toast`, `renderHook` + `act`).

- [ ] **Step 1: Write the failing tests**

Add `dataUpdatedAt: 0` to `makeQuery`'s default object (before the
`...overrides` spread) so every existing case keeps compiling, then add a
`describe("lastUpdatedAt")` block to
`src/lib/hooks/__tests__/useGridRefresh.test.ts`:

| subtest name | queries (`dataUpdatedAt` values) | expect `result.current.lastUpdatedAt` |
|---|---|---|
| `is the minimum non-zero stamp across queries` | `[5000, 9000]` | `5000` |
| `ignores zero stamps when some query has resolved` | `[0, 9000]` | `9000` |
| `is null when no query has ever resolved` | `[0, 0]` | `null` |
| `is null for an empty query list` | `[]` | `null` |
| `is the single stamp for a single query` | `[7000]` | `7000` |

The `[5000, 9000] → 5000` case is the one that fails loudly on a
max-vs-min regression; do not soften it to `toBeGreaterThan`.

Existing cases (`isRefreshing`, success/error toast, `alsoRefresh`) must be
left byte-identical apart from the `makeQuery` default — that is the FR-5.5
backward-compatibility proof.

- [ ] **Step 2: Implement**

Replace the type alias:

```ts
/** Minimal structural contract the refresh hook needs from a data source. */
export interface RefreshableQuery {
  isFetching: boolean;
  dataUpdatedAt: number;
  refetch: () => Promise<{ isError: boolean; error: unknown }>;
}
```

Delete the now-unused `import type { UseQueryResult } from "@tanstack/react-query";`.
Keep the existing doc comment about `refetch()` resolving rather than
rejecting, and add a line recording why this is a structural interface rather
than `Pick<UseQueryResult, …>` (D1: non-React-Query sources such as
`TenantsPage`'s tenant-context shim must be expressible).

Add to `UseGridRefreshResult`:

```ts
export interface UseGridRefreshResult {
  isRefreshing: boolean;
  onRefresh: () => Promise<void>;
  lastUpdatedAt: number | null;
}
```

In the hook body, above `onRefresh`, compute (D2 — no `useMemo`):

```ts
const stamps = queries.map((q) => q.dataUpdatedAt).filter((t) => t > 0);
const lastUpdatedAt = stamps.length ? Math.min(...stamps) : null;
```

and return it alongside the existing two values.

- [ ] **Step 3: Verify**

```
npm test -- src/lib/hooks/__tests__/useGridRefresh.test.ts
```

Run from `services/atlas-ui`. All cases green.

---

## Task 2: `EmptyState` — refresh button, action row, last-updated caption

Implements D3, D4 (FR-1.*, FR-2.*, FR-3.*).

### Files

- `services/atlas-ui/src/components/common/EmptyState.tsx` — new props, action
  row, caption
- `services/atlas-ui/src/components/common/__tests__/EmptyState.test.tsx` — new file
- `services/atlas-ui/src/components/data-table.tsx` — read-only; lines 76-92 are
  the visual contract (icon, `animate-spin`, `disabled`, `aria-busy`) to mirror

Patterns to copy: `services/atlas-ui/src/components/__tests__/data-table.test.tsx:12-42`
(spin/disabled assertions via `button.querySelector("svg")`),
`services/atlas-ui/src/components/common/__tests__/Pager.test.tsx:1-20`
(plain `render` + `screen`, no providers needed).

- [ ] **Step 1: Write the failing tests**

New file `src/components/common/__tests__/EmptyState.test.tsx`.
Fixture constant: `const TS = 1_735_732_920_000;` (assert the ISO string as
`new Date(TS).toISOString()` so the suite does not depend on the runner's
`TZ`/`LANG` — D4/Q4).

`describe("EmptyState refresh control")`:

| subtest name | props | assertions |
|---|---|---|
| `renders no refresh button when onRefresh is absent` | `title="Empty"` | `screen.queryByTestId("empty-state-refresh")` is `null`; `screen.getByTestId("empty-state")` still present |
| `renders a Refresh button when onRefresh is supplied` | `title="Empty"`, `onRefresh={fn}` | `getByTestId("empty-state-refresh")` present; `getByRole("button", { name: /refresh/i })` is that same node; `fireEvent.click` → `fn` called once |
| `disables the button and spins the icon while refreshing` | `onRefresh={fn}`, `isRefreshing` | button `toBeDisabled()`; `toHaveAttribute("aria-busy", "true")`; `button.querySelector("svg")` `toHaveClass("animate-spin")` |
| `does not spin or disable when isRefreshing is false` | `onRefresh={fn}`, `isRefreshing={false}` | button not disabled; svg does not have `animate-spin` |
| `does not invoke onRefresh while disabled` | `onRefresh={fn}`, `isRefreshing` | `fireEvent.click` → `fn` not called |

`describe("EmptyState action precedence")`:

| subtest name | props | assertions |
|---|---|---|
| `renders the custom action alone when onRefresh is absent` | `action={{ label: "Create Account", onClick: fn }}` | exactly one button, accessible name `Create Account` |
| `renders both, action first, refresh second` | `action={{ label: "Create Account", onClick: a }}`, `onRefresh={r}` | `screen.getAllByRole("button")` has length 2; `[0]` has text `Create Account`; `[1]` is the `empty-state-refresh` node. Assert order by index, never by CSS class |
| `keeps refresh on the outline variant with and without a sibling action` | rendered twice (with and without `action`) | the refresh button's `className` contains `border` in both renders (the `outline` variant token) — assert the *same* class list both times |

`describe("EmptyState last-updated caption")`:

| subtest name | `lastUpdatedAt` | assertions |
|---|---|---|
| `renders the caption for a positive timestamp` | `TS` | `getByTestId("empty-state-last-updated")` present; its text starts with `Last updated`; `toHaveAttribute("title", new Date(TS).toISOString())` |
| `renders no caption for null` | `null` | `queryByTestId("empty-state-last-updated")` is `null` |
| `renders no caption for zero` | `0` | `queryByTestId("empty-state-last-updated")` is `null` |
| `renders no caption when absent` | prop omitted | `queryByTestId("empty-state-last-updated")` is `null` |
| `advances the caption when the timestamp changes` | `TS` then rerender with `TS + 3_600_000` | after `rerender`, the `title` attribute equals `new Date(TS + 3_600_000).toISOString()` |

- [ ] **Step 2: Implement**

Extend the props interface (exported shape per PRD §5):

```ts
interface EmptyStateProps {
  icon?: React.ReactNode;
  title: string;
  description?: string;
  action?: { label: string; onClick: () => void };
  onRefresh?: () => void | Promise<void>;
  isRefreshing?: boolean;
  lastUpdatedAt?: number | null;
  className?: string;
}
```

Import `RefreshCw` from `lucide-react` (`cn` and `Button` are already
imported). Replace the current single-`action` block with an action row and a
caption:

- Row: `<div className="mt-4 flex items-center justify-center gap-2">`,
  rendered only when `action || onRefresh`.
- Custom `action` first in DOM order, default `Button` variant, **without**
  the `mt-4` it carries today (the row now owns the margin).
- Refresh second: `variant="outline"`, `onClick={() => void onRefresh()}`
  (D3 — do not leak the promise into the DOM handler),
  `disabled={isRefreshing}`, `aria-busy={isRefreshing}`,
  `data-testid="empty-state-refresh"`, body
  `<RefreshCw className={cn("h-4 w-4", isRefreshing && "animate-spin")} />`
  followed by the text `Refresh`.
- Caption, rendered only when `lastUpdatedAt != null && lastUpdatedAt > 0`:

```tsx
<p
  className="mt-3 text-xs text-muted-foreground"
  data-testid="empty-state-last-updated"
  title={new Date(lastUpdatedAt).toISOString()}
>
  Last updated{" "}
  {new Date(lastUpdatedAt).toLocaleTimeString(undefined, {
    hour: "2-digit",
    minute: "2-digit",
  })}
</p>
```

No timer, no clock subscription (NFR performance; FR-3.5).

- [ ] **Step 3: Verify**

```
npm test -- src/components/common/__tests__/EmptyState.test.tsx
```

---

## Task 3: `DataTableWrapper` — forward refresh into the empty branch

Implements D5 (FR-4.*).

### Files

- `services/atlas-ui/src/components/common/DataTableWrapper.tsx` — add
  `lastUpdatedAt` prop; forward three props on the empty branch
- `services/atlas-ui/src/components/common/__tests__/DataTableWrapper.test.tsx` — new file

Patterns to copy: `services/atlas-ui/src/components/__tests__/data-table.test.tsx:1-12`
(the `Row` type + `DataTableColumnDef<Row>[]` fixture; no providers required).

- [ ] **Step 1: Write the failing tests**

New file `src/components/common/__tests__/DataTableWrapper.test.tsx`.
Fixtures:

```ts
type Row = { id: string; name: string };
const columns: DataTableColumnDef<Row>[] = [
  { accessorKey: "name", header: "Name" },
];
const TS = 1_735_732_920_000;
```

| subtest name | props | assertions |
|---|---|---|
| `renders the empty-state refresh button when onRefresh is supplied` | `data={[]}`, `onRefresh={fn}` | `getByTestId("empty-state-refresh")` present; click → `fn` called once |
| `renders no refresh button on the empty branch without onRefresh` | `data={[]}` | `queryByTestId("empty-state-refresh")` is `null` |
| `forwards isRefreshing to the empty-state button` | `data={[]}`, `onRefresh={fn}`, `isRefreshing` | button `toBeDisabled()` |
| `renders both the custom action and refresh on the empty branch` | `data={[]}`, `onRefresh={fn}`, `emptyState={{ title: "No accounts found", action: { label: "Create Account", onClick: a } }}` | `getByText("No accounts found")` present; `getAllByRole("button")` length 2, `[0]` text `Create Account`, `[1]` is `empty-state-refresh` |
| `forwards lastUpdatedAt to the caption` | `data={[]}`, `onRefresh={fn}`, `lastUpdatedAt={TS}` | `getByTestId("empty-state-last-updated")` has `title` equal to `new Date(TS).toISOString()` |
| `renders the default empty copy when no emptyState is supplied` | `data={[]}` | `getByText("No data available")` and `getByText("There are no items to display at this time.")` present (FR-4.5) |
| `renders the loader on the loading branch` | `data={[]}`, `loading`, `onRefresh={fn}` | `queryByTestId("empty-state")` is `null` |
| `renders the error branch with a retry, not an empty state` | `data={[]}`, `error="boom"`, `onRefresh={fn}` | `queryByTestId("empty-state")` is `null`; a button with an accessible name matching `/try again/i` is present |

- [ ] **Step 2: Implement**

Add `lastUpdatedAt?: number | null;` to `DataTableWrapperProps` next to
`isRefreshing`, destructure it, and change the empty branch's render to:

```tsx
<EmptyState
  {...defaultEmptyState}
  {...(onRefresh && { onRefresh })}
  {...(typeof isRefreshing === "boolean" && { isRefreshing })}
  {...(lastUpdatedAt != null && { lastUpdatedAt })}
/>
```

The conditional-spread idiom is required by `exactOptionalPropertyTypes` and
already used on the data branch. Do **not** add a nested
`emptyState.onRefresh` (FR-4.3). Loading, error, and data branches are
untouched (FR-4.4).

- [ ] **Step 3: Verify**

```
npm test -- src/components/common/__tests__/DataTableWrapper.test.tsx
```

---

## Task 4: Pass `lastUpdatedAt` through the eleven already-wired pages

Implements FR-6.1/FR-6.3 for the pages that already hand the wrapper
`onRefresh` and `isRefreshing`. This is the same one-line change repeated
eleven times — it batches, per the plan-format sizing rule.

### Files

Each file gets two edits: destructure `lastUpdatedAt` from the existing
`useGridRefresh(...)` call, and add `lastUpdatedAt={lastUpdatedAt}` to the
`<DataTableWrapper …>` props next to `isRefreshing`.

- `services/atlas-ui/src/pages/AccountsPage.tsx` — hook at :33, wrapper at :162
- `services/atlas-ui/src/pages/BansPage.tsx` — hook at :76, wrapper at :146
- `services/atlas-ui/src/pages/CharactersPage.tsx` — hook at :36, wrapper at :87
- `services/atlas-ui/src/pages/CouponsPage.tsx` — hook at :84, wrapper at :223
- `services/atlas-ui/src/pages/GuildsPage.tsx` — hook at :65, wrapper at :116
- `services/atlas-ui/src/pages/MapsPage.tsx` — hook at :58, wrapper at :105
- `services/atlas-ui/src/pages/MerchantsPage.tsx` — hook at :122, wrapper at :203
- `services/atlas-ui/src/pages/QuestsPage.tsx` — hook at :42, wrapper at :97
- `services/atlas-ui/src/pages/ReportsPage.tsx` — hook at :55, wrapper at :92
- `services/atlas-ui/src/pages/ServicesPage.tsx` — hook at :23, wrapper at :65
- `services/atlas-ui/src/pages/TemplatesPage.tsx` — hook at :89, wrapper at :263

Line numbers are from the pre-change tree and are a locator, not an assertion.

- [ ] **Step 1: Apply the pass-through**

For each file:

```
const { isRefreshing, onRefresh } = useGridRefresh([...]);
→ const { isRefreshing, onRefresh, lastUpdatedAt } = useGridRefresh([...]);
```

and in the wrapper JSX, immediately after `isRefreshing={isRefreshing}`:

```
lastUpdatedAt={lastUpdatedAt}
```

`lastUpdatedAt` is typed `number | null` and the prop accepts `number | null`,
so pass it directly — no conditional spread is needed at the call site.

Do not change any page's own header refresh button, filters, or layout. Two
pages that call `useGridRefresh` are deliberately **not** in this list —
`EventDefinitionsPage` and `RewardPoolsPage` are Task 5, and
`EventOccurrencesPage` is Task 8.

- [ ] **Step 2: Verify**

```
npx tsc -b
npm test -- src/pages/__tests__
```

Run from `services/atlas-ui`. Existing page suites must pass unchanged — no
test edits belong to this task.

---

## Task 5: `EventDefinitionsPage` and `RewardPoolsPage` — one refresh control per view

These two pages call `useGridRefresh` but deliberately withhold `onRefresh`
from the wrapper, because each renders its own header refresh button; today
that leaves their empty states dead-ended. Handing the wrapper `onRefresh`
also lights `DataTable`'s toolbar control (D5), so the bespoke header button
must move rather than double up. `RewardPoolsPage.test.tsx:121` asserts the
single-control property explicitly, so this is enforced, not cosmetic.

### Files

- `services/atlas-ui/src/pages/EventDefinitionsPage.tsx` — remove the header
  refresh button (:112-124); pass the three props to the wrapper (:127)
- `services/atlas-ui/src/pages/RewardPoolsPage.tsx` — move the tabs-row refresh
  button (:124-136) into `poolTable`'s wrapper (:93) and add an equivalent
  control to the hand-built Global tab (:171-175)
- `services/atlas-ui/src/pages/__tests__/RewardPoolsPage.test.tsx` — update the
  `renders a single refresh control next to the tabs` case (:121-129)
- `services/atlas-ui/src/pages/__tests__/EventDefinitionsPage.test.tsx` —
  read-only; confirm nothing asserts the removed header button

- [ ] **Step 1: `EventDefinitionsPage`**

Delete the header `<Button …><RefreshCw …/></Button>` block, leaving the
title row as `<div className="flex items-center justify-between">` with the
`<h2>` alone. Drop the now-unused `RefreshCw` import if nothing else in the
file uses it (`cn` may still be used elsewhere — check before removing).

Destructure `lastUpdatedAt` and pass all three to the wrapper:

```tsx
onRefresh={onRefresh}
isRefreshing={isRefreshing}
lastUpdatedAt={lastUpdatedAt}
```

`onRefresh` is `() => Promise<void>` and the wrapper's prop is `() => void`;
`Promise<void>` is assignable to `void` in a return position, so pass it
directly — this is what `AccountsPage.tsx:162` already does. If eslint's
`no-misused-promises` objects, wrap as
`onRefresh={() => void onRefresh()}` (the idiom at
`services/atlas-ui/src/pages/CouponsPage.tsx:227`).

- [ ] **Step 2: `RewardPoolsPage`**

Give `poolTable` the refresh wiring (it closes over `onRefresh`,
`isRefreshing`, `lastUpdatedAt` from the existing hook call at :51):

```tsx
<DataTableWrapper
  columns={poolColumns}
  data={data}
  error={error}
  onRefresh={onRefresh}
  isRefreshing={isRefreshing}
  lastUpdatedAt={lastUpdatedAt}
  emptyState={{ title: emptyTitle, description: emptyDescription }}
/>
```

Remove the refresh `<Button>` from the tabs row so the four pool tabs each show
exactly one control (the `DataTable` toolbar on the data branch, the
`EmptyState` button on the empty branch).

The Global tab is a hand-built `<Table>`, not a grid, so it keeps a control of
its own: add the same `variant="outline" size="icon"` refresh button next to
the existing `Add Item` button in the
`<div className="flex justify-end">` row, with `onClick={onRefresh}`,
`disabled={isRefreshing}`, `title="Refresh"`, `aria-busy={isRefreshing}`, and
`<RefreshCw className={cn("h-4 w-4", isRefreshing && "animate-spin")} />`.
Keep `RefreshCw` and `cn` imported for it.

- [ ] **Step 3: Update the `RewardPoolsPage` test**

Replace the single case at :121 with two, keeping the "exactly one" property
per tab (Radix unmounts inactive `TabsContent`, so `getByRole` remains
unambiguous):

| subtest name | steps | assertions |
|---|---|---|
| `renders exactly one refresh control on a pool tab` | render, `await waitFor` on `getByText("Henesys")` | `getAllByRole("button", { name: /refresh/i })` has length 1 |
| `renders exactly one refresh control on the Global tab` | render, wait, `user.click(screen.getByRole("tab", { name: /global/i }))` | `getAllByRole("button", { name: /refresh/i })` has length 1 |

- [ ] **Step 4: Verify**

```
npm test -- src/pages/__tests__/RewardPoolsPage.test.tsx src/pages/__tests__/EventDefinitionsPage.test.tsx
npx tsc -b
```

---

## Task 6: `tenant-context` + `TenantsPage` — shim, timestamp, skeleton guard

Implements D6 (FR-6.2). This is the one task where the change leaves the UI
layer.

### Files

- `services/atlas-ui/src/context/tenant-context.tsx` — `tenantsUpdatedAt` in
  the value (:16-24, :35, :80, :113, :150, :188-197); `refreshTenants` returns
  a result (:108-141)
- `services/atlas-ui/src/pages/TenantsPage.tsx` — shim + `useGridRefresh` +
  local refreshing flag + skeleton guard (:45, :124, :143-149)
- `services/atlas-ui/src/pages/__tests__/TenantsPage.test.tsx` — mock returns
  `{ ok: true }` (:7, :53); new empty-state cases
- `services/atlas-ui/src/context/__tests__/tenant-context.test.tsx` —
  read-only; it awaits `refreshTenants()` and ignores the result, so it must
  keep passing untouched

- [ ] **Step 1: Widen the context**

In `src/context/tenant-context.tsx`:

```ts
export type TenantRefreshResult = { ok: true } | { ok: false; error: unknown };
```

- `TenantContextType.refreshTenants` becomes
  `() => Promise<TenantRefreshResult>`.
- Add `tenantsUpdatedAt: number;` to `TenantContextType` and to the provider
  value object.
- Add `const [tenantsUpdatedAt, setTenantsUpdatedAt] = useState<number>(0);`
  beside the other state.
- Call `setTenantsUpdatedAt(Date.now());` immediately after **each** of the
  three `setTenants(data)` calls: the initial-load effect, `refreshTenants`,
  and `refreshAndSelectTenant`.
- `refreshTenants` returns `{ ok: true }` as the last statement of its `try`
  block and `{ ok: false, error: errorInfo }` from its `catch`; the `finally`
  that clears `loading` is unchanged. Keep the existing `setError` and
  `console.error` calls — the shim consumes the returned error, and the
  existing state is still what other consumers see.

`app-tenant-switcher.tsx:128` passes `refreshTenants` as an `onSuccess`
callback typed `() => void`; a function returning a value is assignable to a
`void`-returning type, so that call site needs no change.

- [ ] **Step 2: Wire `TenantsPage`**

Destructure `tenantsUpdatedAt` alongside the existing
`{ tenants, loading, refreshTenants }`, then add:

```tsx
const [isRefreshingTenants, setIsRefreshingTenants] = useState(false);
const tenantsSource: RefreshableQuery = {
  isFetching: isRefreshingTenants,
  dataUpdatedAt: tenantsUpdatedAt,
  refetch: async () => {
    setIsRefreshingTenants(true);
    try {
      const result = await refreshTenants();
      return result && result.ok === false
        ? { isError: true, error: result.error }
        : { isError: false, error: null };
    } finally {
      setIsRefreshingTenants(false);
    }
  },
};
const { isRefreshing, onRefresh, lastUpdatedAt } = useGridRefresh([
  tenantsSource,
]);
```

`result &&` is deliberate: a bare `vi.fn()` mock resolves `undefined`, and an
absent result must read as success rather than as a failure (D6b risk row).

Import `useGridRefresh` and `type RefreshableQuery` from
`@/lib/hooks/useGridRefresh`.

Change the skeleton guard (:124) to:

```tsx
if (loading && !isRefreshingTenants) {
  return <TenantPageSkeleton />;
}
```

so a refresh keeps the grid on screen with the spinner on the button (D6d).
First load is unaffected — `isRefreshingTenants` starts `false`.

Pass the three props to the wrapper at :143 alongside the existing
`columns`/`data`/`emptyState`.

- [ ] **Step 3: Extend the `TenantsPage` tests**

Existing file edits:
- In `beforeEach`, add `refreshTenantsMock.mockResolvedValue({ ok: true });`
  (after `vi.clearAllMocks()`).
- Add `tenantsUpdatedAt: 0,` to `defaultUseTenantValue()`.
- Add a module mock for the grid toast, since the page now routes refresh
  feedback through it (the file already mocks `sonner` for the rename flow):

```ts
vi.mock("@/lib/utils/toast", () => ({ success: vi.fn(), error: vi.fn() }));
```

  and `import * as gridToast from "@/lib/utils/toast";` for the assertions.

New `describe("TenantsPage empty-state refresh")` — each case overrides
`useTenantMock` before rendering:

| subtest name | context override | assertions |
|---|---|---|
| `offers a refresh control when no tenants exist` | `{ ...defaultUseTenantValue(), tenants: [] }` | `getByText("No tenants found")` present; `getByTestId("empty-state-refresh")` present |
| `clicking refresh calls refreshTenants and toasts success` | same | `user.click` the refresh button → `refreshTenantsMock` called once; `gridToast.success` called with `"Data refreshed"`; `gridToast.error` not called |
| `toasts an error when the refresh reports failure` | same, with `refreshTenantsMock.mockResolvedValue({ ok: false, error: new Error("network down") })` | `gridToast.error` called once with the same `Error` instance and `{ context: { action: "refresh" } }`; `gridToast.success` not called |
| `treats an undefined refresh result as success` | same, with `refreshTenantsMock.mockResolvedValue(undefined)` | `gridToast.success` called once |
| `shows the grid, not the skeleton, while refreshing` | start from `{ ...defaultUseTenantValue(), tenants: [tenantA, tenantB] }`; make `refreshTenantsMock` return a promise you resolve by hand, click refresh, then re-point `useTenantMock` at the same value with `loading: true` and rerender before resolving | `screen.getByRole("heading", { name: "Tenants" })` still present and `screen.getByText("Acme")` still present — `TenantPageSkeleton` renders no text, so a visible "Tenants" heading proves the real page is on screen |
| `renders the skeleton on first load` | `{ ...defaultUseTenantValue(), loading: true }`, no click | `screen.queryByRole("heading", { name: "Tenants" })` is `null` and `queryByText("No tenants found")` is `null` |

The existing rename/delete cases must keep passing untouched.

- [ ] **Step 4: Verify**

```
npm test -- src/pages/__tests__/TenantsPage.test.tsx src/context/__tests__/tenant-context.test.tsx
npx tsc -b
```

---

## Task 7: `GuildDetailPage` — first-time refresh wiring

Implements the design's "Pages — new wiring" row for `GuildDetailPage`.

### Files

- `services/atlas-ui/src/pages/GuildDetailPage.tsx` — add the hook call
  (after :25) and the three wrapper props (:116)

Patterns to copy: `services/atlas-ui/src/pages/GuildsPage.tsx:65-70`
(multi-query `useGridRefresh` over the same two query hooks).

- [ ] **Step 1: Implement**

Immediately after `const tenantConfigQuery = useTenantConfiguration(...)` —
and before any early return, so the hook order is unconditional — add:

```tsx
const { isRefreshing, onRefresh, lastUpdatedAt } = useGridRefresh([
  guildQuery,
  tenantConfigQuery,
]);
```

with `import { useGridRefresh } from "@/lib/hooks/useGridRefresh";`.

Pass `onRefresh={onRefresh}`, `isRefreshing={isRefreshing}`, and
`lastUpdatedAt={lastUpdatedAt}` to the `<DataTableWrapper>` at :116.

The page keeps its own `PageLoader`/`ErrorDisplay` early returns; the wrapper's
loading and error branches remain unreached, which is unchanged behavior. As
D5 notes, the members grid also gains a toolbar refresh on the populated
branch — that is intended, not scope creep.

- [ ] **Step 2: Verify**

```
npx tsc -b
npm test -- src/pages/__tests__
```

There is no `GuildDetailPage` test suite today and this task does not add one:
the wiring is identical to `GuildsPage` and is covered at the contract level by
Tasks 1–3. Record that in `context.md`, not as a silent omission.

---

## Task 8: `EventOccurrencesPage` — direct `EmptyState` props

Implements D8. The page renders `EmptyState` directly, not through
`DataTableWrapper`, so it takes the props by hand.

### Files

- `services/atlas-ui/src/pages/EventOccurrencesPage.tsx` — hook at :109,
  `EmptyState` at :204
- `services/atlas-ui/src/pages/__tests__/EventOccurrencesPage.test.tsx` — new
  empty-branch case

- [ ] **Step 1: Implement**

Destructure `lastUpdatedAt` from the existing
`useGridRefresh([occurrencesQuery])` and pass all three to the `EmptyState`:

```tsx
<EmptyState
  title="No event occurrences found"
  description="No occurrences match the current filters."
  onRefresh={onRefresh}
  isRefreshing={isRefreshing}
  lastUpdatedAt={lastUpdatedAt}
/>
```

The page's own header refresh button stays — it is the only control on the
loading and error branches, and it sits in the page header rather than beside
the empty-state button.

- [ ] **Step 2: Write the test**

Add to `src/pages/__tests__/EventOccurrencesPage.test.tsx`:

| subtest name | service mock | assertions |
|---|---|---|
| `offers a refresh control when no occurrences match` | `getOccurrences` resolves `{ data: [], meta: null }` | `await screen.findByText("No event occurrences found")`; `getByTestId("empty-state-refresh")` present; `user.click` it → `eventsService.getOccurrences` called a second time (`toHaveBeenCalledTimes(2)`) |

If the click's toast trips on the unmocked `@/lib/utils/toast`, add
`vi.mock("@/lib/utils/toast", () => ({ success: vi.fn(), error: vi.fn() }))`
to the file rather than skipping the assertion.

- [ ] **Step 3: Verify**

```
npm test -- src/pages/__tests__/EventOccurrencesPage.test.tsx
```

---

## Task 9: `TransportsPage` — `EmptyState` on the scheduled tab

Implements D7 (FR-7.1 / Q1: in place, no `DataTableWrapper` migration).

### Files

- `services/atlas-ui/src/pages/TransportsPage.tsx` — replace the
  `routes.length === 0` `<p>` (:119-123) with an `EmptyState`
- `services/atlas-ui/src/pages/__tests__/TransportsPage.test.tsx` — extend the
  zero-routes case (:242-252)

- [ ] **Step 1: Implement**

Add `const { isRefreshing, onRefresh } = useGridRefresh([scheduledQuery]);`
near the other page state, and replace the empty `<p>` branch with:

```tsx
<EmptyState
  title="No scheduled routes configured"
  onRefresh={onRefresh}
  isRefreshing={isRefreshing}
/>
```

Import `EmptyState` from `@/components/common/EmptyState` and `useGridRefresh`
from `@/lib/hooks/useGridRefresh`.

**No `lastUpdatedAt`** — the page header already renders `FreshnessIndicator`
bound to `scheduledQuery.dataUpdatedAt`, and a second freshness readout on the
same query would be redundant and could disagree visually (D7).

Leave the data branch's `onRefresh={() => void scheduledQuery.refetch()}`
exactly as it is. D7 scopes this task to the empty branch; changing the
populated branch's refresh to the toasting hook is a behavior change the design
does not sanction.

- [ ] **Step 2: Extend the test**

Keep the existing zero-routes case (its `findByText(/no scheduled routes
configured/i)` still matches the `EmptyState` title) and add to it, or add a
sibling case:

| subtest name | mock | assertions |
|---|---|---|
| `offers a refresh control on the empty scheduled tab` | `getScheduledRoutes` resolves `[]` | `await screen.findByText(/no scheduled routes configured/i)`; `getByTestId("empty-state-refresh")` present; `user.click` it → `transportsService.getScheduledRoutes` called a second time; `screen.getAllByText(/updated \d+s ago/i)` has length 1 (the header `FreshnessIndicator` at `src/components/features/transports/FreshnessIndicator.tsx:48`, still unduplicated) |

- [ ] **Step 3: Verify**

```
npm test -- src/pages/__tests__/TransportsPage.test.tsx
```

---

## Task 10: Record the FR-6.5 call-site sweep

### Files

- `docs/tasks/task-258-refreshable-empty-states/call-site-sweep.md` — new file

- [ ] **Step 1: Re-run the sweep against the finished branch**

From the repo root:

```
grep -rn "<DataTableWrapper" services/atlas-ui/src
grep -rn "<EmptyState" services/atlas-ui/src
grep -rn "useGridRefresh(" services/atlas-ui/src
```

- [ ] **Step 2: Write the artifact**

`call-site-sweep.md` records one row per site with the three props ticked:

- Every `<DataTableWrapper>` call site, with `onRefresh` / `isRefreshing` /
  `lastUpdatedAt` marked present.
- The two non-wrapper grids that take the props directly:
  `EventOccurrencesPage` (all three) and `TransportsPage` (two, with the D7
  reason `lastUpdatedAt` is deliberately omitted).
- `src/pages/event-occurrences-columns.tsx` — **not applicable**: the
  `DataTableWrapper` mention is a comment at line 4, not a render (FR-6.4).
- The known pre-existing gap from Q3: `AccountsPage`'s ban-status fan-out runs
  outside React Query and is not in its `alsoRefresh`. It only runs when
  `accounts.length > 0`, so it cannot affect the empty state; recorded, not
  fixed.
- `RewardPoolsPage`'s Global tab: a hand-built `<Table>` with its own refresh
  button (Task 5), not a grid.

Any site the sweep finds without the props is a defect to fix in this task,
not a note to leave behind.

- [ ] **Step 3: Verify**

```
tools/verify.sh
```

Flagless, from the repo root; must exit 0 before the branch is called done.
