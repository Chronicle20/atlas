# Fix round — frontend blocking findings (B1, B2, B3, B4) — report

All four items from `.superpowers/sdd/plan/fix-frontend-brief.md` implemented and committed separately.

## B1 — FR-19 map pins on the field overview

- Verified by inspection: `MapImageOverlay.tsx`'s `computeMarkers<T extends { id: string; attributes: { x: number; y: number } }>` and `MonsterMarker` (`monster.attributes.template`) confirm the overlay reads exactly `id`, `attributes.x`, `attributes.y`, `attributes.template` off a monster — nothing else. No adaptation of the plan was needed.
- Added `PositionedMonster` (exported from `map-entities.service.ts`) as the structural minimum: `{ id: string; attributes: { template: number; x: number; y: number } }`. `MapMonsterData` already satisfies it structurally, so `MapDetailPage.tsx` (which passes `MapMonsterData[]` straight through) compiles unchanged — confirmed via `npx tsc -b --noEmit`.
- `MapImagePanel.tsx` and `MapImageOverlay.tsx` (including `MonsterMarker`'s prop type) narrowed from `MapMonsterData[]` to `PositionedMonster[]`.
- `FieldDetailPage.tsx` now calls `useMapPortals`, `useMapNpcs`, `useMapReactors` (same hook set `MapDetailPage.tsx` uses) and passes `portals`/`npcs`/`reactors` straight through. A local `toPositionedMonsters` adapter maps `monstersQuery.data` (live monsters, per the resolved ambiguity — NOT declared spawn points) to `{ id, attributes: { template: monsterId, x, y } }` with no invented values.
- Test: `FieldDetailPage.test.tsx` mocks `MapImagePanel` (props spy) and the three new hooks, asserting non-empty `portals`/`npcs`/`reactors` and the live→pin adapted `monsters` array.

## B2 — pagination regression tests

- New `fields.service.test.ts`: mocks `@/lib/api/client`, asserts `api.getList` is called with a URL containing `page[size]=250` (URL-encoded as `page%5Bsize%5D=250`, confirmed against the actual `URLSearchParams.toString()` output) for both `getFields` and `getFieldCharacters`, and that `filter[worldId]`/`filter[channelId]`/`filter[mapId]` are present only when the corresponding filter is supplied and absent otherwise.
- New `live-monsters.service.test.ts`: same assertion for `liveMonstersService.getMonsters`.
- Mocking idiom copied from `transports.service.test.ts` / `bans.service.test.ts` (module-level `vi.mock("@/lib/api/client", ...)` with a captured `getList` spy).

## B3 — false empty state while loading

- `FieldMonstersTab.tsx`: split the single `!monsters || monsters.length === 0` empty-state guard into two branches — `monsters === undefined` renders a "Loading monsters..." affordance (matching `MapObjectsTable.tsx`'s `objects === undefined` pattern), then `monsters.length === 0` is the real resolved-empty state.
- `FieldObjectsTab.tsx`: added a `defined === undefined` loading branch ("Loading map objects...") before the existing merge/empty logic. Deliberately gated only on `defined` (the definition-entities query), not `tracked` — `tracked` is documented in the component's own docstring to be permanently `undefined` until Task 22 wires it up, which is a real "no state tracked" signal, not a loading state; treating it as loading would break the existing `untracked-only` tests and misrepresent that field. Simplified `const untrackedObjects = (defined ?? []).filter(...)` to `defined.filter(...)` since `defined` is narrowed to non-undefined past the new guard.
- Tests added to both components' existing `__tests__` files: "does not show empty state while loading" (renders with `undefined` data, asserts loading text present / empty text absent) and "shows empty state once resolved-and-empty" (renders with `[]`, asserts the reverse).

## B4 — worlds.service.ts consistency (non-blocking)

- `getWorlds()` and `getChannels()` had no query params at all — genuinely a one-line-plus-comment change. Added the same `PAGE_SIZE = 250` constant and explanatory comment shape from `fields.service.ts:21-24`, applied to both methods in the class (they share the constant and the same silent-truncation risk).

## Tests run

```
cd services/atlas-ui
npx vitest run src/pages/__tests__/FieldDetailPage.test.tsx \
  src/services/api/__tests__/fields.service.test.ts \
  src/services/api/__tests__/live-monsters.service.test.ts \
  src/components/features/fields/__tests__/FieldMonstersTab.test.tsx \
  src/components/features/fields/__tests__/FieldObjectsTab.test.tsx
```
Result: `Test Files  5 passed (5)` / `Tests  40 passed (40)`.

```
npx tsc -b --noEmit
```
Result: no output (clean).

```
npm run lint
```
Result: `9 problems (0 errors, 9 warnings)` — all 9 warnings are pre-existing (`react-hooks/incompatible-library` on `form.watch()` call sites in unrelated dialog components, and two `exhaustive-deps` warnings on `AccountsPage.tsx`/`QuestsPage.tsx`), none in files touched by this fix round.

## Files changed

- `services/atlas-ui/src/services/api/map-entities.service.ts` — new exported `PositionedMonster` interface.
- `services/atlas-ui/src/components/features/maps/MapImagePanel.tsx` — `monsters` prop narrowed to `PositionedMonster[]`.
- `services/atlas-ui/src/components/features/maps/MapImageOverlay.tsx` — `monsters` prop and `MonsterMarker` narrowed to `PositionedMonster`.
- `services/atlas-ui/src/pages/FieldDetailPage.tsx` — three definition-entity hooks, `toPositionedMonsters` adapter, four new `MapImagePanel` props.
- `services/atlas-ui/src/pages/__tests__/FieldDetailPage.test.tsx` — mocks for the new hooks, a `MapImagePanel` props spy, and a new pin-passthrough assertion test.
- `services/atlas-ui/src/services/api/__tests__/fields.service.test.ts` — new.
- `services/atlas-ui/src/services/api/__tests__/live-monsters.service.test.ts` — new.
- `services/atlas-ui/src/components/features/fields/FieldMonstersTab.tsx` — loading/empty split.
- `services/atlas-ui/src/components/features/fields/FieldObjectsTab.tsx` — loading/empty split.
- `services/atlas-ui/src/components/features/fields/__tests__/FieldMonstersTab.test.tsx` — two new cases.
- `services/atlas-ui/src/components/features/fields/__tests__/FieldObjectsTab.test.tsx` — two new cases.
- `services/atlas-ui/src/services/api/worlds.service.ts` — explicit `page[size]=250`.

## Commits

- `1943cd687` fix(atlas-ui): reuse MapImagePanel entity pins on the field overview (B1/FR-19)
- `24680b283` fix(atlas-ui): add regression tests for the page[size]=250 pagination fix (B2)
- `8209d85d7` fix(atlas-ui): gate field-detail empty states on the loading flag (B3)
- `dafc01a56` fix(atlas-ui): request page[size]=250 explicitly in worlds.service (B4)

## Self-review

- No Go files touched; no `git add -A`/`git add .` used — every commit staged specific paths by name.
- No new `*_testhelpers.go`-style files; no invented API fields — `PositionedMonster` and the test fixtures for portals/npcs/reactors were built strictly from the existing `MapPortalData`/`MapNpcData`/`MapReactorData`/`LiveMonsterData` interfaces already in the codebase.
- `MapDetailPage.tsx` was not modified and still compiles (verified by `tsc -b`).
- Verified the branch stayed `task-292-map-definition-field-split` and the worktree root stayed correct after every commit.

## Concerns

None. All four items are complete, tested, and typecheck-clean.
