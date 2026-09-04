# bug-fields-ui-round2 — field/map UI polish, second live-testing round

Task: task-292-map-definition-field-split
Branch: task-292-map-definition-field-split
Reported: live testing of the field-detail and map-definition pages.

This is a UI-only polish round (four items) against the frontend that landed
in this task. No backend, contract, or packet surface is touched.

## Reproduced

All four observed in the running UI on the task branch by the user; each is
confirmed against the source below, not re-derived from the running app.

## Item 1 — Monster grid renders the raw template id

**Observed.** `services/atlas-ui/src/components/features/fields/FieldMonstersTab.tsx`,
`FieldMonsterRow`, renders the Monster cell as a bare underlined link whose
text is the numeric `monsterId`:

```tsx
<Link to={`/monsters/${monsterId}`} className="underline">
  {monsterId}
</Link>
```

**Expected.** The monster resolves to its **name**, rendered as a **badge**,
with a **copyable tooltip carrying the template id**, still linking to the
monster definition page (`/monsters/{monsterId}`, routed at
`src/App.tsx:393`).

**Root cause.** The cell was written to the id, never to a resolved name. The
name resolver already exists and is already used elsewhere on this same page's
map overlay: `useMobData(templateId)` (`src/lib/hooks/useMobData.ts`) returns
`{ name, iconUrl, ... }` and is consumed by `MonsterMarker` in
`MapImageOverlay.tsx`. The copyable-tooltip idiom also already exists —
`<TooltipContent copyable>` (`src/components/ui/tooltip.tsx:41,121-137`), used
by `FieldCharactersTab.tsx`'s name cell and by
`src/components/common/CopyableIdHeader.tsx`.

Fall back to the raw id as the badge text when `useMobData` has no name
(loading or errored) — the row must never unmount or blank out, matching the
`FieldCharacterRow` degradation rule already in this feature.

## Item 2 — Spawn column is opaque and misleading

**Observed.** The Monster grid's `Spawn` column renders
`spawnSourceType / spawnSourceId` and links to `/maps/{mapId}?tab=monsters`.

**Root cause / finding.** These are *provenance* fields, not a spawn-row
reference. `services/atlas-monsters/atlas.com/monsters/monster/model.go:15-23`
defines the type domain as `CYCLIC` (also the normalization target for an
absent value, applied once at the Kafka consumer boundary —
`kafka/consumer/monster/consumer.go:397-406`), `EVENT`, `SCRIPT`, `GM`.
`model.go:77-84` states the id is opaque: atlas-monsters stores, echoes and
compares it for equality but never interprets it. It therefore cannot select a
definition spawn row, and the link is unresolvable by construction.

**Decision (user, this round): remove the column entirely.** Not "badge it",
not "keep it" — the field-detail Monsters grid drops `Spawn`. Columns become
Object ID, Monster, HP, Position.

## Item 3 — "Live Monsters" column header

**Observed.** `src/components/features/maps/LiveFieldsSection.tsx:104` —
`<TableHead>Live Monsters</TableHead>` in the map-definition page's Live Fields
grid.

**Expected.** `Monsters`. The section is already titled "Live Fields"; the
"Live" prefix on the column is redundant.

## Item 4 — Map image shows monster pins but not characters

**Observed.** On the field-detail page the map image overlays monster pins.
`FieldDetailPage.tsx` adapts live monsters into `PositionedMonster[]` via
`toPositionedMonsters()` and passes them to `MapImagePanel` → `MapImageOverlay`,
which renders `MonsterMarker`s. Characters get no pin.

**Expected.** Characters currently in the field render as pins too, with a
visually distinct marker (monsters are `bg-rose-500/70`; NPC sky, portal
emerald, reactor amber — pick an unused hue, e.g. violet/indigo) and a tooltip
showing the character name.

**Root cause.** Character positions are not available at page scope. The
`characters` endpoint returns **ids only** (`FieldCharacterData` in
`src/services/api/fields.service.ts:26-29`; the roster hook is
`useFieldCharacters` in `src/lib/hooks/api/useFieldRuntime.ts`). `x`/`y` come
from per-character enrichment, which today happens *inside each table row* —
`FieldCharacterRow` calls `useCharacter(activeTenant, characterId)`
(`FieldCharactersTab.tsx`). The overlay lives above the tabs and has no access
to those per-row results, and rows for the non-active tab may not even be
mounted.

The fix is to lift the enrichment to a batch hook so both the overlay and the
table read the same cache entries. React Query dedupes by key, so a
`useQueries` batch built with the **existing** `characterKeys.detail(tenant, id)`
key and `charactersService.getById` shares cache with `FieldCharacterRow` — no
extra network requests, and the existing Refresh path
(`alsoRefresh: invalidateQueries({ queryKey: characterKeys.details() })` in
`FieldDetailPage.tsx`) keeps working unchanged. Note `characterKeys.detail`
already accepts `Tenant | null` specifically so batched callers need no
non-null assertion.

`useQueries` batch precedents in this codebase:
`src/lib/hooks/api/useActorNames.ts:50,70`, `src/lib/hooks/api/useItemNames.ts:16`,
`src/lib/hooks/api/useMaps.ts:125`.

## Fix

Worktree: `.worktrees/task-292-map-definition-field-split/`

Files to change (all under `services/atlas-ui/`):

- `src/components/features/fields/FieldMonstersTab.tsx` — item 1 + item 2.
  Monster cell becomes a `Badge` (`@/components/ui/badge`) wrapped in a
  `Link` to `/monsters/{monsterId}` inside a `Tooltip` whose
  `<TooltipContent copyable>` holds the template id; badge text is
  `useMobData(monsterId).name` with the raw id as fallback. Delete the `Spawn`
  header, the `Spawn` cell, and the now-unused `spawnSourceType` /
  `spawnSourceId` / `hasSpawn` / `mapId` plumbing. If `mapId` becomes unused,
  drop the prop from `FieldMonstersTabProps` and from its call site in
  `src/pages/FieldDetailPage.tsx`. Update the file's doc-comment: it currently
  documents the D13 column list and the FR-30 spawn behaviour, both of which
  this change supersedes.
- `src/components/features/fields/__tests__/FieldMonstersTab.test.tsx` —
  update column assertions, assert the name badge + copyable tooltip content +
  href, assert no `Spawn` column. `useMobData` will need mocking (see the
  existing mock style in the maps `__tests__` that cover `MapImageOverlay`/
  `MapEntitySummary`).
- `src/components/features/maps/LiveFieldsSection.tsx:104` — item 3, header
  text `Live Monsters` → `Monsters`.
- `src/components/features/maps/__tests__/LiveFieldsSection.test.tsx` — update
  any assertion on that header text.
- `src/lib/hooks/api/useFieldRuntime.ts` (or a sibling hook file) — item 4,
  add a batched character-detail hook over `characterIds`, built on
  `useQueries` + `characterKeys.detail` + `charactersService.getById`, keys
  identical to `useCharacter`'s so the table rows dedupe against it.
- `src/pages/FieldDetailPage.tsx` — item 4, call the batch hook, adapt
  resolved characters to positioned markers (mirroring
  `toPositionedMonsters`), pass a `characters` prop to `MapImagePanel`; drop
  the `mapId` prop passed to `FieldMonstersTab` if item 2 made it unused.
- `src/components/features/maps/MapImagePanel.tsx` — item 4, accept and
  forward a `characters` prop to both `MapImageOverlay` call sites (preview
  and expanded dialog).
- `src/components/features/maps/MapImageOverlay.tsx` — item 4, add
  `CharacterMarker` + its `computeMarkers` pass + the DEV out-of-bounds warn
  entry, distinct colour, tooltip = character name.
- `src/components/features/maps/HoverHighlightContext.tsx` — item 4, add a
  `{ kind: "character"; characterId: string }` variant to `HoverTarget` and
  its `isHovered` case.
- Tests for the overlay/panel/page touched above
  (`src/components/features/maps/__tests__/`,
  `src/pages/__tests__/FieldDetailPage.test.tsx`).

Constraints:

- Characters pass through `MapImagePanel` only as position data. Do **not**
  add a character fetch inside `MapImageOverlay` — the batch hook at page
  scope is the single fetch point (contrast `MonsterMarker`, which calls
  `useMobData` per marker only because that resolver is a shared, long-cached
  name lookup, not a runtime read).
- No backend change. No new service method beyond reusing
  `charactersService.getById`.
- Follow `frontend-dev-guidelines`; the frontend lint/format guard applies.

## Not yet answered

- Whether the character marker should also participate in hover-highlight
  pairing with the Characters tab rows (monsters/NPCs pair with their tables).
  Implement the overlay-side `HoverTarget` variant; wiring the *table row*
  side is optional and may be skipped if the Characters tab has no existing
  hover-highlight plumbing.

## Resolution

- Fix commit: _pending_
- Gate: _pending_
- Live re-test: _pending_
