# bug-fields-ui-feedback

Nineteen items of post-implementation UI feedback on the task-292 Fields /
Maps surfaces. Eighteen are presentation-consistency defects (the new Fields
screens do not follow the conventions the rest of atlas-ui already uses);
item 11 is a genuine data-staleness bug with an established root cause.

All work is frontend-only, inside
`services/atlas-ui/src`. Paths below are relative to that directory.

## Reproduced

Reported by the user against the running branch build
(`task-292-map-definition-field-split`, head `5d59a1f65`). The presentation
items are verifiable by reading the components listed under `## Fix`; the
current markup is quoted in each item. Item 11 was reproduced by the user
("Refresh doesn't appear to update the character position") and its cause is
confirmed in source, below.

## Observed / Expected

### Maps → Live Fields (`components/features/maps/LiveFieldsSection.tsx`)

1. **Grid is bare.** The component renders `<section>` + `<Table>` with no
   `Card` wrapper, unlike every other grid in the app.
   *Expected:* the fields grid sits inside a `Card` (see `pages/ItemsPage.tsx`
   Results card, and `components/features/fields/FieldCharactersTab.tsx`'s
   empty-state card, for the house pattern).
2. **Heading verbosity** — `components/features/maps/MapEntitySummary.tsx:117`,
   `:128`, `:154` render "Configured Monster Spawns".
   *Expected:* "Monster Spawns" (all three occurrences, including the
   `({order.length})` variant at `:154`).
3. **World id shown raw.** `LiveFieldsSection.tsx:38` renders
   `{worldId}` — "0".
   *Expected:* the resolved world name ("Scania"). `useWorlds()` from
   `lib/hooks/api/useWorlds.ts` is the existing source; `pages/FieldDetailPage.tsx:177-179`
   already does this lookup (`worlds.find((w) => w.id === String(worldId))?.attributes.name`).
   Fall back to the raw id when the lookup has not resolved.
4. **Channel is 0-indexed on screen.** `LiveFieldsSection.tsx:39` renders the
   raw `channelId`.
   *Expected:* display `channelId + 1`. Display only — the value sent to the
   API, used in links, and used in query keys stays 0-based.

### Field detail (`pages/FieldDetailPage.tsx`, `components/features/fields/FieldHeader.tsx`)

5. **Breadcrumb is a mess.** `lib/breadcrumbs/routes.ts:217-249` defines four
   nested nodes (`/fields/[worldId]`, `.../[channelId]`, `.../[mapId]`,
   `.../[instanceId]`) producing "Fields / World 0 / Channel 0 / 100000000 /
   Instance <uuid>".
   *Expected:* the breadcrumb reduces to "Fields".
6. **URL is a mess.** The route is
   `/fields/:worldId/:channelId/:mapId/:instanceId` (`App.tsx:386`).
   *Expected:* `/fields` with query parameters `world`, `channel`, `map`,
   `instance`.
   **Ruling (mine, made after the report):** collapse to a single `/fields`
   route. `FieldsPage` renders the field-detail view when an `instance` query
   param is present and the list view otherwise; there is no second path.
   The list's existing `?map=` filter param is unchanged and coexists.
   Keep channel 0-based *in the URL* — the 1-indexing in items 4/7/15 is a
   render-time concern only.
7. **Header layout.** `FieldHeader.tsx:27-38` puts the runtime badge inline
   with the map name and the world/channel/instance as a plain `<p>`.
   *Expected:* map name on its own line; on the row **below** it, the
   `SurfaceKindBadge kind="runtime"` plus badges for world (resolved name,
   e.g. "Scania"), channel (1-indexed for display), and instance.
8. **"View Map Definition" is a text link** (`FieldHeader.tsx:36-38`).
   *Expected:* an icon button next to Refresh in the page's action row, not a
   link inside the header block. Pick an appropriate lucide icon (e.g. `Map`)
   and give it an accessible label; wrap in a tooltip per the app's
   icon-button convention.
9. **Refresh is a labelled button** (`FieldDetailPage.tsx:194-205`).
   *Expected:* icon-only button (`size="icon"`), keeping the spinning
   `RefreshCw`, `disabled`/`aria-busy` behaviour and an accessible name.
10. **Character ID column is redundant** —
    `components/features/fields/FieldCharactersTab.tsx:47` header and `:78`,
    `:95` cells.
    *Expected:* drop the column. The Name cell becomes the clickable element
    wrapped in a tooltip exposing the copyable id, matching
    `components/map-cell.tsx:48-58` and `pages/ItemsPage.tsx:434-447`
    (`TooltipProvider` / `Tooltip` / `TooltipTrigger asChild` /
    `TooltipContent copyable`). The unresolved-character row (`:72-84`) must
    keep rendering — show the id as the name there.
11. **Refresh does not update character positions.** *(real bug)*
    **Root cause:** `FieldDetailPage.tsx:97-105` passes only the page-level
    queries to `useGridRefresh`. Character rows enrich themselves through
    `useCharacter(activeTenant, characterId)` inside `FieldCharacterRow`
    (`FieldCharactersTab.tsx:69`), and those per-row detail queries are not in
    the refresh list, so `onRefresh()` never refetches them and `x`/`y` stay
    stale.
    *Expected fix:* pass `alsoRefresh` to `useGridRefresh` invalidating the
    character-detail prefix — `characterKeys.details()` from
    `lib/hooks/api/useCharacters.ts:34` — so mounted row queries refetch.
12. **Summary card is redundant.** `FieldSummaryPanels`
    (`components/features/fields/FieldSummaryPanels.tsx`) duplicates the
    character/monster counts already in the tab labels, plus a permanently
    "—" tracked-objects panel.
    *Expected:* remove it from `FieldDetailPage.tsx:232-235` and delete the
    component (and its imports). Adjust the surrounding
    `grid md:grid-cols-[2fr_1fr]` so `MapImagePanel` still lays out sensibly.

### Fields list (`pages/FieldsPage.tsx`, `components/features/fields/FieldsResultTable.tsx`, `FieldsFilterBar.tsx`)

13. **Map cell is a mono text link** (`FieldsResultTable.tsx:44-51`, rendering
    `"Name (id)"`).
    *Expected:* a `Badge` with the map name, wrapped in a tooltip whose
    content is the copyable map id — same pattern as
    `components/map-cell.tsx`. It stays a link into the field detail.
14. **Column order** is Map, Channel, Instance (`FieldsResultTable.tsx:30-32`).
    *Expected:* Channel, Map, Instance (Characters stays last).
15. **Channel 0-indexed** (`FieldsResultTable.tsx:52`, and the channel
    `Select` options in `FieldsFilterBar.tsx:85`).
    *Expected:* display `channelId + 1` in both the grid and the filter
    dropdown labels. The `SelectItem` *values*, the state, and the API filter
    stay 0-based.
16. **Refresh is a labelled button** (`FieldsPage.tsx:122-133`).
    *Expected:* icon button, same treatment as item 9.
17. **Grid is bare** — `FieldsResultTable` renders a naked `<Table>`.
    *Expected:* wrap the results grid in a `Card` (ItemsPage Results card is
    the reference).
18. **Runtime badge is inline with the title** (`FieldsPage.tsx:116-120`).
    *Expected:* "Fields" title on its own row, `SurfaceKindBadge kind="runtime"`
    on the row below it.
19. **Filter bar is unstyled** — `FieldsFilterBar.tsx:47` is a bare flex row.
    *Expected:* follow `pages/ItemsPage.tsx:271-374` — a `Card` with
    `CardHeader`/`CardTitle`/`CardDescription` ("Search Fields" + a
    result-count description) and the controls in `CardContent`, including
    the map free-text `Input` and a "Clear" button.

## Fix

Frontend only; no Go, no API, no contract change.

| File | Items |
|---|---|
| `src/App.tsx` | 6 (single `/fields` route; drop the 4-segment route) |
| `src/lib/breadcrumbs/routes.ts` | 5, 6 (remove the `/fields/[worldId]…` chain and the `FIELD_DETAIL` entry at `:726`) |
| `src/lib/breadcrumbs/__tests__/fields-routes.test.ts` | 5, 6 (rewrite for the collapsed shape) |
| `src/pages/FieldsPage.tsx` | 6, 16, 17, 18, 19 (+ dispatch to detail view on `?instance=`) |
| `src/pages/FieldDetailPage.tsx` | 6 (read world/channel/map/instance from query params, not `useParams`), 8, 9, 11, 12 |
| `src/components/features/fields/FieldHeader.tsx` | 7, 8 |
| `src/components/features/fields/FieldCharactersTab.tsx` | 10 |
| `src/components/features/fields/FieldsResultTable.tsx` | 13, 14, 15, 17 |
| `src/components/features/fields/FieldsFilterBar.tsx` | 15, 19 |
| `src/components/features/fields/FieldSummaryPanels.tsx` | 12 (delete) |
| `src/components/features/maps/LiveFieldsSection.tsx` | 1, 3, 4 (+ link target now query-param form) |
| `src/components/features/maps/MapEntitySummary.tsx` | 2 |
| `src/pages/__tests__/FieldsPage.test.tsx` | all list-side items |
| `src/pages/__tests__/FieldDetailPage.test.tsx` | all detail-side items |
| `src/components/features/maps/__tests__/LiveFieldsSection.test.tsx` | 1, 3, 4 |
| `src/components/features/fields/__tests__/FieldCharactersTab.test.tsx` | 10 |

Also sweep for any other producer of a `/fields/<world>/<channel>/<map>/<instance>`
href (`grep -rn '/fields/' src`) — every one becomes the query-param form.
`src/components/app-sidebar-items.ts` points at `/fields` already and needs no
change; confirm rather than assume.

Reference patterns to copy rather than invent:
- copyable-id badge: `src/components/map-cell.tsx:48-58`
- filter card + results card: `src/pages/ItemsPage.tsx:271-374`, `:376-388`
- world-name lookup: `src/pages/FieldDetailPage.tsx:177-179`

## Not yet answered

- Nothing blocking. The one judgment call (item 6's single-route collapse) is
  recorded as a ruling above; if the query-param detail view proves awkward
  the fallback is `/fields/:instanceId` with the rest as query params, but do
  not switch to it without asking.
- Item 12 deletes the only consumer of `FieldSummaryPanels`; if a test or doc
  references it (`grep -rn FieldSummaryPanels`), update or delete that too.

## Resolution

Fixed across three commits on `task-292-map-definition-field-split`:

| Commit | Scope |
|---|---|
| `e25b083f1` | Items 5-19 — routing collapse to a single `/fields` route, breadcrumb reduction, Fields list card/filter-card/icon-button/badge work, field-detail header, character-tab tooltip column, item 11 refresh wiring, `FieldSummaryPanels` deletion |
| `947013114` | Items 1-4 — Live Fields card wrapper, resolved world name, 1-indexed channel display, "Monster Spawns" heading |
| `f29dde41a` | Two non-blocking review findings — 1-indexed channel in the Fields empty-state summary, tightened icon-only Refresh assertions |

Item 6 was implemented as the ruling above specified: a single `/fields`
route that renders the detail view when `?instance=` is present.

**Gates.** `tools/verify.sh --quick` exit 0 over `5d59a1f65..947013114` and
again over `947013114..f29dde41a`. The flagless `tools/verify.sh` then passed
at `f29dde41a` with exit 0 — full suite including the docker bake, `-race`,
every guard, and the atlas-ui tests and build.

**Review.** `task-reviewer` over `5d59a1f65..HEAD`:
`APPROVED_WITH_FINDINGS`, 0 blocking, 2 non-blocking (both closed by
`f29dde41a`). Artifact: `reviews/bug-fields-ui.md`.

**Live testing.** Deferred to the PR ephemeral environment, per the project's
normal flow — not re-tested locally. The item to watch there is item 11:
confirm character positions actually update on Refresh, since the unit tests
assert the query-invalidation wiring rather than the observed behaviour that
was originally reported.
