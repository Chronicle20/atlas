# Review — task-292 pre-PR fix round (frontend, TS-only)

Scope: commits `1943cd687`, `24680b283`, `8209d85d7`, `dafc01a56` in
`services/atlas-ui`. Go commits `041d8a784`/`379839b3e` are interleaved from a
concurrent backend implementer and are explicitly out of scope; no `.go` file
appears in this diff (`git diff --stat 1943cd687^..dafc01a56` confirms all 17
changed files are under `services/atlas-ui/`).

Brief: `.superpowers/sdd/plan/fix-frontend-brief.md`
Report: `.superpowers/sdd/plan/fix-frontend-report.md`

## B1 / FR-19 — map pins on the field overview

**PASS.**

- `MapImageOverlay.tsx` was read in full. `computeMarkers<T extends { id:
  string; attributes: { x: number; y: number } }>` and `MonsterMarker`
  (`services/atlas-ui/src/components/features/maps/MapImageOverlay.tsx:268-292`)
  read exactly `id`, `attributes.x`, `attributes.y`, `attributes.template` off
  a monster — nothing else (verified by reading the full component, not
  spot-checked). The report's claim matches the code.
- `PositionedMonster` (`services/atlas-ui/src/services/api/map-entities.service.ts:82-95`)
  is `{ id: string; attributes: { template: number; x: number; y: number } }`
  — the exact structural minimum, no invented fields.
- `MapMonsterData` (`map-entities.service.ts:64-79`) is a structural superset
  of `PositionedMonster` (adds `mobTime`, `team`, `cy`, `f`, `fh`, `rx0`,
  `rx1`, `hide`), so `MapDetailPage.tsx` (which still passes
  `MapMonsterData[]` unmodified, per `git diff` — that file is untouched in
  this range) continues to satisfy the narrowed prop structurally. `npx tsc -b
  --noEmit` run locally reproduces the report's "clean" result.
- `FieldDetailPage.tsx:34-46` adapts `LiveMonsterData` (`id`,
  `attributes.monsterId`, `attributes.x`, `attributes.y` — confirmed against
  `live-monsters.service.ts:15-27`) to `PositionedMonster` with a 1:1 field
  mapping and no fabricated defaults; `FieldDetailPage.tsx:90-95,224-227`
  wires `useMapPortals`/`useMapNpcs`/`useMapReactors` (the same hook set
  `MapDetailPage.tsx` uses) and passes the live-derived monsters, not
  declared spawns.
- Test: `FieldDetailPage.test.tsx:454-473` — `"FR-19: passes definition
  entity pins and live-monster pins to MapImagePanel"` — spies on the
  (mocked) `MapImagePanel`'s received props and asserts `portals`/`npcs`/
  `reactors` equal the mocked hook fixtures, and `monsters` equals the
  live-monster-derived array with `template` sourced from
  `attributes.monsterId` (not `attributes.template`, which live monsters
  don't have) and `x`/`y` passed through. This assertion would fail if the
  adapter or the prop wiring were reverted — genuine regression coverage, not
  a pass-either-way test.
- Ran `npx vitest run src/pages/__tests__/FieldDetailPage.test.tsx ...` (the
  five files from the report) locally: `Test Files 5 passed (5)` / `Tests 40
  passed (40)`, confirming the report's claimed run.

## B2 — pagination regression tests

**PASS.**

- `fields.service.test.ts` and `live-monsters.service.test.ts` both
  `vi.mock("@/lib/api/client", ...)` at the module boundary and capture the
  `getList` call's URL argument directly — they exercise the service's own
  query-string construction, not a re-mock of the service itself.
- All three call sites are covered: `fieldsService.getFields` (asserts
  `page%5Bsize%5D=250` and, separately, that `filter[worldId]`/
  `filter[channelId]`/`filter[mapId]` are present only when supplied and
  absent otherwise — `fields.service.test.ts:16-42`), `getFieldCharacters`
  (`:44-53`), and `liveMonstersService.getMonsters`
  (`live-monsters.service.test.ts:16-30`).
- Confirmed against `fields.service.ts:20-56`: `PAGE_SIZE = 250` is set
  unconditionally via `URLSearchParams({ "page[size]": String(PAGE_SIZE) })`
  in both methods, and filters are conditionally `.set()` only when
  `!== undefined` — the test assertions match the actual implementation, so
  they would fail if the fix were reverted.

## B3 — false empty state while loading

**PASS.**

- `FieldMonstersTab.tsx:46-62` now branches `monsters === undefined` (loading
  affordance) before `monsters.length === 0` (real empty state), matching
  `MapObjectsTable.tsx:16-29`'s `objects === undefined` / `objects.length ===
  0` two-branch shape exactly (same guard ordering, same `undefined` vs
  `[]` semantics).
- `FieldObjectsTab.tsx:55-67` adds the analogous `defined === undefined`
  branch. The choice to gate only on `defined` (not `tracked`) is
  well-reasoned and documented inline (`tracked` is a genuine "no state
  tracked" signal until Task 22, per the component's own pre-existing
  docstring at `FieldObjectsTab.tsx:22-31`) — this does not silently drop
  the requirement, it is a scoped, justified narrowing.
- Both components' test files gained a "loading" case (asserts the empty
  copy is absent, the loading copy is present, rendered with `undefined`
  data) and a "resolved-and-empty" case (reverse assertions, rendered with
  `[]`) — `FieldMonstersTab.test.tsx:155-173`, `FieldObjectsTab.test.tsx:116-132`.
  These are two independent renders per component, so they cannot pass by
  accident of shared state.
- Confirmed via the same local `vitest run` above: all four new test cases
  pass.

## B4 — worlds.service.ts consistency (non-blocking)

**PASS, correctly scoped as non-blocking.** `worlds.service.ts:39-55` adds
the same `PAGE_SIZE = 250` constant/comment shape and applies it to both
`getWorlds()` and `getChannels()`. No test was required by the brief for B4
and none was added; consistent with the brief's "only if genuinely one-line"
carve-out.

## Multi-tenancy / React Query conventions (regression check)

**PASS.** No hook files were modified in this range — `useMapPortals`/
`useMapNpcs`/`useMapReactors` (`lib/hooks/api/useMapEntities.ts:19-53`) and
`useFieldCharacters`/`useLiveMonsters` (`lib/hooks/api/useFieldRuntime.ts`)
already carry `enabled: !!mapId && !!activeTenant` / `enabled: enabled &&
!!activeTenant`, and `FieldDetailPage.tsx` only adds new *call sites* to
already-tenant-guarded hooks. Query keys (`mapEntityKeys.portals/npcs/
reactors`, all `["maps", mapId, <kind>]`) are unchanged and remain disjoint
from field-runtime keys. No `refetchInterval` was introduced anywhere in this
diff (grep confirms no occurrence in the touched files).

## Non-blocking findings

1. **`FieldDetailPage.tsx:97-102`** — `useGridRefresh([mapQuery,
   charactersQuery, monstersQuery, objectsQuery])` was not extended to
   include `portalsQuery`/`npcsQuery`/`reactorsQuery`. `objectsQuery` (the
   same class of map-definition data, also freshly displayed via the pins
   this fix round wires up) *is* refreshed by the page's Refresh button, but
   the three new portal/npc/reactor queries are not — clicking "Refresh"
   will not update those pins. This is outside the brief's explicit B1
   requirement (which only asked for prop passthrough + a passthrough test,
   both of which are satisfied), and definition entities are long-lived
   (`staleTime`/`gcTime` of 10 minutes) so the practical impact is small, but
   it is a real inconsistency with the freshly-touched `objectsQuery`
   sibling and worth a follow-up one-liner.
2. The loading affordance text (`"Loading monsters..."` /
   `"Loading map objects..."`) is plain text, not a `<Skeleton>` component,
   which the `frontend-dev-guidelines` skill's checklist calls out as the
   house convention for content-area loading states
   (`.claude/skills/frontend-dev-guidelines/SKILL.md:37`,
   `resources/anti-patterns.md:111-118`). This fix round correctly followed
   the brief's explicit instruction to copy `MapObjectsTable.tsx:21`'s
   existing (pre-established, plain-text) pattern rather than introduce a
   new convention mid-fix-round, so this is not attributable to this unit —
   flagging only because the skill was named as a consult target for this
   review.

## Not evaluable

None — the full scoped diff (17 files, all in `services/atlas-ui/`) was read,
the referenced sibling files (`MapObjectsTable.tsx`, `MapDetailPage.tsx`,
`live-monsters.service.ts`, `map-entities.service.ts`,
`lib/hooks/api/useMapEntities.ts`, `lib/hooks/useGridRefresh.ts`) were read
where the change's correctness genuinely depended on their contracts, and the
report's test/typecheck claims were reproduced locally.

## Verdict

APPROVED_WITH_FINDINGS — all four brief items (B1-B4) are genuinely closed
with real regression coverage; the two notes above are pre-existing/low-impact
and do not block.
