# Fix: FieldDetailPage refresh omits FR-19 pin queries

## What was done

1. `services/atlas-ui/src/lib/hooks/useGridRefresh.ts` — read first. Its
   `RefreshableQuery` contract only needs `{ isFetching, dataUpdatedAt, refetch }`.
   `useMapPortals`/`useMapNpcs`/`useMapReactors` are plain React Query
   `useQuery` results (same shape as `useMapObjects`, which was already in
   the array), so they satisfy the contract with no adaptation needed.

2. `services/atlas-ui/src/pages/FieldDetailPage.tsx` — added `portalsQuery`,
   `npcsQuery`, `reactorsQuery` to the `useGridRefresh([...])` array (now
   7 entries: map, characters, monsters, objects, portals, npcs, reactors).

3. `services/atlas-ui/src/pages/__tests__/FieldDetailPage.test.tsx` — added a
   new test, `"refresh also refetches the FR-19 pin queries (portals, npcs,
   reactors), not just map/characters/monsters/objects"`. It clicks the
   Refresh button via `userEvent` and asserts `.refetch` was called on all
   seven mocked query results (map, characters, monsters, objects, portals,
   npcs, reactors), pulling each mock's return value via
   `useXMock.mock.results[0]?.value`.

4. `services/atlas-ui/src/pages/MapDetailPage.tsx` — checked. This page
   calls `useMapPortals`/`useMapNpcs`/`useMapReactors`/`useMapObjects`
   directly but does **not** call `useGridRefresh` at all — no Refresh
   button exists on that page. It is not affected by this finding and
   needs no matching change. Confirmed via `grep -n "Refresh\|refresh"
   services/atlas-ui/src/pages/MapDetailPage.tsx` (no matches) and a grep
   for `useGridRefresh` (no matches). No fix applied, per instructions.

## Testing

```
cd services/atlas-ui && npx vitest run src/pages/__tests__/FieldDetailPage.test.tsx
```
Result: `Test Files 1 passed (1)`, `Tests 15 passed (15)` (14 existing + 1 new).

```
cd services/atlas-ui && npx tsc -b --noEmit
```
Result: no output (clean).

```
cd services/atlas-ui && npm run lint
```
Result: `9 problems (0 errors, 9 warnings)` — all 9 warnings are pre-existing,
in unrelated files (`GenerateCouponBatchDialog.tsx`, `PoolFormDialog.tsx`,
`PoolItemDialog.tsx`, `CreateTenantDialog.tsx`, `AccountsPage.tsx`,
`QuestsPage.tsx`), none touching the two files changed here. Zero errors.

## Files changed

- `services/atlas-ui/src/pages/FieldDetailPage.tsx`
- `services/atlas-ui/src/pages/__tests__/FieldDetailPage.test.tsx`

## Self-review

- The three added queries are the same "static definition data" class as
  `objectsQuery`, which was already present — consistent with the existing
  pattern, no new abstraction introduced.
- New test follows the file's existing `queryResult()` mock-builder pattern
  and asserts on real `refetch` mock calls rather than implementation
  details of `useGridRefresh`.
- No changes to `MapDetailPage.tsx` — confirmed out of scope, reported per
  the brief.

## Issues or concerns

None. Branch confirmed as `task-292-map-definition-field-split` after commit,
worktree clean except pre-existing untracked docs files not touched by this
fix.
