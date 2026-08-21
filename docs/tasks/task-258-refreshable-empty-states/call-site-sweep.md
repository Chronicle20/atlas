# FR-6.5 call-site sweep

Re-run against the finished `task-258-refreshable-empty-states` branch. Confirms
every grid that renders an empty state wires `onRefresh`, `isRefreshing`, and
`lastUpdatedAt` from `useGridRefresh`, and records the documented exceptions.

Commands run from the repo root:

```
grep -rn "<DataTableWrapper" services/atlas-ui/src
grep -rn "<EmptyState" services/atlas-ui/src
grep -rn "useGridRefresh(" services/atlas-ui/src
```

## `<DataTableWrapper>` call sites

`DataTableWrapper` renders `EmptyState` internally
(`services/atlas-ui/src/components/common/DataTableWrapper.tsx`), so every
call site below needs the three props forwarded to it.

| File | `onRefresh` | `isRefreshing` | `lastUpdatedAt` |
|---|---|---|---|
| `services/atlas-ui/src/pages/CharactersPage.tsx` | present | present | present |
| `services/atlas-ui/src/pages/GuildsPage.tsx` | present | present | present |
| `services/atlas-ui/src/pages/GuildDetailPage.tsx` | present | present | present |
| `services/atlas-ui/src/pages/CouponsPage.tsx` | present | present | present |
| `services/atlas-ui/src/pages/BansPage.tsx` | present | present | present |
| `services/atlas-ui/src/pages/RewardPoolsPage.tsx` | present | present | present |
| `services/atlas-ui/src/pages/MerchantsPage.tsx` | present | present | present |
| `services/atlas-ui/src/pages/ReportsPage.tsx` | present | present | present |
| `services/atlas-ui/src/pages/QuestsPage.tsx` | present | present | present |
| `services/atlas-ui/src/pages/AccountsPage.tsx` | present | present | present |
| `services/atlas-ui/src/pages/TemplatesPage.tsx` | present | present | present |
| `services/atlas-ui/src/pages/MapsPage.tsx` | present | present | present |
| `services/atlas-ui/src/pages/EventDefinitionsPage.tsx` | present | present | present |
| `services/atlas-ui/src/pages/TenantsPage.tsx` | present | present | present |
| `services/atlas-ui/src/pages/ServicesPage.tsx` | present | present | present |

All 14 `DataTableWrapper` call sites in `src/pages` forward all three props.
No defects found.

## Non-wrapper grids that take the props directly

| File | `onRefresh` | `isRefreshing` | `lastUpdatedAt` | Notes |
|---|---|---|---|---|
| `services/atlas-ui/src/pages/EventOccurrencesPage.tsx` | present | present | present | Renders `<EmptyState>` directly (not through `DataTableWrapper`) because the row shape needs a per-row `data-testid`/class for FR-UI5; all three props present at line 206. |
| `services/atlas-ui/src/pages/TransportsPage.tsx` | present | present | **deliberately omitted** | `<EmptyState>` at line 124 passes `onRefresh`/`isRefreshing` but not `lastUpdatedAt` — design decision D7. Not a defect. |

## Documented non-applicable / pre-existing sites

- `services/atlas-ui/src/pages/event-occurrences-columns.tsx` — **not
  applicable**. The `DataTableWrapper` mention at line 4 is a comment
  explaining why this page's column shape deliberately does *not* use the
  shared `DataTableWrapper`/`DataTable` (FR-6.4); it is not a render.
- `services/atlas-ui/src/pages/AccountsPage.tsx` — **known pre-existing gap
  (Q3), recorded not fixed**. The ban-status fan-out (`banStatuses` /
  `banStatusLoading`, populated by an effect starting at line 99) runs
  outside React Query and is not part of the `alsoRefresh` list passed to
  `useGridRefresh` (line 33). It only runs when `accounts.length > 0`, so it
  cannot affect the empty state.
- `services/atlas-ui/src/pages/RewardPoolsPage.tsx` (Global tab) — not a
  grid. The Global tab renders a hand-built `<Table>` (imported from
  `@/components/ui/table`, not `DataTableWrapper`) with its own refresh
  button, added in Task 5. The pool tabs on the same page do use
  `DataTableWrapper` and are covered in the table above.
- `services/atlas-ui/src/pages/EventOccurrenceDetailPage.tsx` — **not
  applicable**. The `<EmptyState>` at line 162 renders the transitions
  sub-table for a single occurrence's detail view. This page has no
  `useGridRefresh` call and no refresh capability anywhere (grep confirms
  no `useGridRefresh(` hit for this file); it is a static detail page
  outside FR-6.5's grid-refresh scope, not a grid missing instrumentation.

## `useGridRefresh(` call sites

```
services/atlas-ui/src/pages/CharactersPage.tsx
services/atlas-ui/src/pages/GuildsPage.tsx
services/atlas-ui/src/pages/GuildDetailPage.tsx
services/atlas-ui/src/pages/CouponsPage.tsx
services/atlas-ui/src/pages/BansPage.tsx
services/atlas-ui/src/pages/RewardPoolsPage.tsx
services/atlas-ui/src/pages/MerchantsPage.tsx
services/atlas-ui/src/pages/ReportsPage.tsx
services/atlas-ui/src/pages/TransportsPage.tsx
services/atlas-ui/src/pages/QuestsPage.tsx
services/atlas-ui/src/pages/AccountsPage.tsx
services/atlas-ui/src/pages/TemplatesPage.tsx
services/atlas-ui/src/pages/MapsPage.tsx
services/atlas-ui/src/pages/EventDefinitionsPage.tsx
services/atlas-ui/src/pages/EventOccurrencesPage.tsx
services/atlas-ui/src/pages/TenantsPage.tsx
services/atlas-ui/src/pages/ServicesPage.tsx
```

(Plus the hook's own definition, `services/atlas-ui/src/lib/hooks/useGridRefresh.ts`,
and its test, `services/atlas-ui/src/lib/hooks/__tests__/useGridRefresh.test.ts`.)

## Result

No defects found. Every `DataTableWrapper` and non-wrapper grid empty state
wires `onRefresh`, `isRefreshing`, and `lastUpdatedAt` (or omits
`lastUpdatedAt` per the documented D7 exception on `TransportsPage`). All
other sites the sweep surfaces are documented exceptions or pre-existing,
recorded gaps, not FR-6.5 defects.
