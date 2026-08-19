# Report: bug-no-event-definition-seed-control

## What I implemented

Frontend-only fix per the bug file's `## Fix` section, using the
`void`/202 seed shape (matching `seedTransportRoutes`/`seedInstanceRoutes`),
not `SeedResult`.

- `services/atlas-ui/src/services/api/seed.service.ts`
  - Added `EventDefinitionsSeedStatus { definitionCount: number; updatedAt: string | null }`.
  - Added `seedEventDefinitions(): Promise<void>` -> `api.post("/api/events/definitions/seed", {})`.
  - Added `getEventDefinitionsSeedStatus(tenant)` -> `fetchSeedStatus("/api/events/definitions/seed/status", tenant)`,
    returning `{ definitionCount: subdomainCount(s, "definition.event"), updatedAt: s.tenantSeededAt ?? s.updatedAt }`.
- `services/atlas-ui/src/lib/hooks/api/useSeed.ts`
  - Added `eventDefinitionsSeedStatusKey(tenantId)`.
  - Added `useSeedEventDefinitions()` mutation (invalidates the status key on success) and
    `useEventDefinitionsSeedStatus()` query (`staleTime: 0`, `refetchInterval: 5000`), mirroring
    `useSeedInstanceRoutes` / `useInstanceRoutesSeedStatus`.
  - Imported `EventDefinitionsSeedStatus` type.
- `services/atlas-ui/src/pages/SetupPage.tsx`
  - Imported the two new hooks and `CalendarClock` icon.
  - Called both hooks alongside the existing ones.
  - Appended a seed row `label: "Event Definitions"` with `CalendarClock` icon and badge
    `${formatCount(d.definitionCount)} ${pluralize(d.definitionCount, "definition", "definitions")}`.
- `services/atlas-ui/src/pages/EventDefinitionsPage.tsx`
  - Corrected the empty-state description from "Event definitions are seeded by atlas-events at
    startup." to "Event definitions are seeded from the Setup page."
- Tests
  - `services/atlas-ui/src/lib/hooks/api/__tests__/useSeed.test.tsx` - added the new mock methods
    (`getEventDefinitionsSeedStatus`, `seedEventDefinitions`), a new `describe.each` case for the
    status hook (enable/disable-on-tenant coverage, same as every other status hook), and a mutation
    test (`useSeedEventDefinitions posts to the event definitions seed endpoint`) alongside the
    existing transport mutation tests.
  - `services/atlas-ui/src/pages/__tests__/SetupPage.test.tsx` - added
    `useSeedEventDefinitions`/`useEventDefinitionsSeedStatus` mocks, extended the row-count test to
    twelve rows including "Event Definitions", and added a new test that clicks the row's Seed
    button and asserts `mutate` was called.
  - `services/atlas-ui/src/pages/__tests__/EventDefinitionsPage.test.tsx` - checked; no test in this
    file was pinned to the old empty-state copy, so no change was needed there.

No Go change was made; no startup auto-seed was added, per the rulings.

## What I tested and the results

From `services/atlas-ui`:

```
npm run build
```
-> succeeded (tsc -b type-checks all of `src` including tests, then `vite build`), no errors.

```
npx vitest run src/lib/hooks/api/__tests__/useSeed.test.tsx src/pages/__tests__/SetupPage.test.tsx src/pages/__tests__/EventDefinitionsPage.test.tsx
```
-> `Test Files 3 passed (3)`, `Tests 52 passed (52)`.

```
npm run test
```
-> `Test Files 257 passed (257)`, `Tests 2092 passed (2092)`. Output is pristine aside from a
pre-existing jsdom `Not implemented: navigation to another Document` warning unrelated to this
change (present before my edits too).

```
npm run lint
```
-> `9 problems (0 errors, 9 warnings)`. All 9 warnings are pre-existing (React Compiler
`incompatible-library` warnings on `react-hook-form`'s `watch()` in unrelated components, and
two `react-hooks/exhaustive-deps` warnings in `AccountsPage.tsx`/`QuestsPage.tsx`) - none touch
files I changed.

## Files changed

- `services/atlas-ui/src/services/api/seed.service.ts`
- `services/atlas-ui/src/lib/hooks/api/useSeed.ts`
- `services/atlas-ui/src/pages/SetupPage.tsx`
- `services/atlas-ui/src/pages/EventDefinitionsPage.tsx`
- `services/atlas-ui/src/lib/hooks/api/__tests__/useSeed.test.tsx`
- `services/atlas-ui/src/pages/__tests__/SetupPage.test.tsx`
- `docs/tasks/task-231-generalized-events-service/bug-no-event-definition-seed-control.md` (added
  to git tracking so it lands with the fix)

## Self-review findings

- Followed the exact shape ruling: `seedEventDefinitions(): Promise<void>` and status via
  `subdomainCount(s, "definition.event")`, matching `seedInstanceRoutes`/`getInstanceRoutesSeedStatus`
  in structure.
- The row-click test locates the button via the row's label ancestor rather than by assuming a
  fixed DOM index, so it stays correct if row order changes.
- Verified no other test in the repo pins the old "seeded by atlas-events at startup" copy
  (`EventDefinitionsPage.test.tsx` has no such assertion), so nothing else needed updating.
- Did not touch `atlas-party-quests`' equivalent gap - explicitly out of scope per the bug file's
  "Not yet answered" section.

## Issues or concerns

None. The change is additive and mirrors an existing, already-tested pattern
(`useSeedInstanceRoutes`/`useInstanceRoutesSeedStatus`) exactly.
