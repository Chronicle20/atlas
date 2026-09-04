# Review: bug-fields-ui (5d59a1f65..HEAD)

Commits: `e25b083f1` (routing collapse + Fields UI house-pattern alignment),
`947013114` (Live Fields Card wrap + world/channel display fix).

Requirement: `docs/tasks/task-292-map-definition-field-split/bug-fields-ui-feedback.md`
(19 numbered items).

## Scope reviewed

`git diff --stat 5d59a1f65..HEAD` — 18 files, all under
`services/atlas-ui/src` plus the two fix-report docs. Read every touched
component/page/route/test file in full; ran `npx vitest run` (full suite),
`npm run build` (tsc -b + vite build), and `npx eslint` on the touched files.

## Item-by-item verification

1. **Live Fields grid in a Card** — PASS.
   `components/features/maps/LiveFieldsSection.tsx:95-125` wraps the `Table`
   in `<Card><CardContent>`. Test:
   `components/features/maps/__tests__/LiveFieldsSection.test.tsx` "renders
   the grid inside a Card" asserts `table.closest('[data-slot="card"]')`.

2. **"Configured Monster Spawns" → "Monster Spawns"** — PASS.
   `components/features/maps/MapEntitySummary.tsx:116,125,150` all say
   "Monster Spawns" (including the `({order.length})` variant). Swept
   `grep -rn "Configured Monster" src` — zero hits.

3. **World id resolved to name** — PASS.
   `LiveFieldsSection.tsx:39-41` looks up `worlds?.find(w => w.id ===
   String(worldId))?.attributes.name ?? String(worldId)`. Test asserts both
   the resolved-name case ("Scania") and the unresolved fallback ("0").

4. **Channel display 1-indexed, value 0-based** — PASS.
   `LiveFieldsSection.tsx:46` renders `{channelId + 1}`; the `Link` at
   `:49` uses raw `channelId` in `channel=${channelId}`. Test
   `LiveFieldsSection.test.tsx` explicitly asserts the link is built with
   `channel=0` while the display cell shows `1` (diff of commit
   `947013114`, hunk against the pre-existing "clicking a row navigates"
   test).

5/6. **Breadcrumb + routing collapse** — PASS, and swept clean.
   - `App.tsx`: `/fields/:worldId/:channelId/:mapId/:instanceId` route and
     its lazy `FieldDetailPage` import removed (diff `-9` lines, no
     `FieldDetailPage` lazy import remains — confirmed via `grep -n
     FieldDetailPage src/App.tsx` returning nothing).
   - `lib/breadcrumbs/routes.ts:207-222`: the four nested `/fields/[worldId]…`
     grouping nodes and `FIELD_DETAIL` constant are gone; only the single
     `/fields` → "Fields" node remains.
   - `lib/breadcrumbs/__tests__/fields-routes.test.ts`: rewritten; asserts
     `getBreadcrumbsFromRoute("/fields", ...)` → `["Home", "Fields"]` and
     `"FIELD_DETAIL" in ROUTE_PATTERNS === false`.
   - `pages/FieldsPage.tsx:126-128`: `?instance=` query param dispatches to
     `<FieldDetailPage />`; no second route.
   - `pages/FieldDetailPage.tsx:64-68`: reads `world`/`channel`/`map`/
     `instance` via `useSearchParams()`, not `useParams()`. Confirmed no
     `useParams` reference remains in either file.
   - Dead-reference sweep (`grep -rn '/fields/'`,
     `grep -rn ':worldId\|:channelId\|:instanceId\|:mapId'`,
     `grep -rn FIELD_DETAIL`, `grep -rn FieldDetailPage`) inside
     `services/atlas-ui/src` — the only `/fields?...` producers left are the
     new query-param hrefs in `LiveFieldsSection.tsx:49` and
     `FieldsResultTable.tsx:61`; `app-sidebar-items.ts` already points at
     plain `/fields` (report confirms it was grepped, not assumed — matches
     what I independently found). No dangling 4-segment href, breadcrumb
     entry, route constant, or test fixture found anywhere.

7. **Header layout** — PASS.
   `components/features/fields/FieldHeader.tsx:29-39`: map name is the sole
   `<h1>` content; the row below holds `SurfaceKindBadge kind="runtime"`
   plus World/Channel/Instance `Badge`s. Channel is `String(Number(channelId)
   + 1)` (`:26`) — display only, the raw `channelId` prop passed in from
   `FieldDetailPage.tsx:200-201` (`channelIdParam`) is untouched and is the
   same value used in the queries.

8. **"View Map Definition" as icon button** — PASS.
   `pages/FieldDetailPage.tsx:208-221`: `Tooltip` > `TooltipTrigger asChild`
   > `Link` styled via `buttonVariants({variant:"outline",size:"icon"})`
   with `MapIcon`, `aria-label="View Map Definition"`, `TooltipContent`
   text. Sits in the action row next to Refresh, not inside `FieldHeader`.
   Test: `FieldDetailPage.test.tsx:357-362` asserts the link has no visible
   "View Map Definition" text content.

9. **Refresh icon-only button** — PASS by source inspection.
   `FieldDetailPage.tsx:222-238`: `Button size="icon"` with only a
   `RefreshCw` child (conditional `animate-spin`), `disabled={isRefreshing}`,
   `aria-busy={isRefreshing}`, `aria-label="Refresh"`, wrapped in the same
   `Tooltip` pattern. **Minor test gap** (non-blocking, see below): no test
   asserts the button lacks a visible text label — only
   `getByRole("button", {name: /refresh/i})` (matches on `aria-label`, which
   would pass whether or not visible text were also present).

10. **Character ID column dropped** — PASS.
    `components/features/fields/FieldCharactersTab.tsx:48-56` header row is
    Name/Level/Job/Position — no ID column. The Name cell (`:94-107`) is the
    `Link`, wrapped in `TooltipProvider`/`Tooltip`/`TooltipTrigger asChild`/
    `TooltipContent copyable`, exposing `characterId`. The unresolved row
    (`:77-88`) still renders and shows `characterId` as the Name cell
    content, per the requirement.

11. **Refresh invalidates per-row character-detail queries (real bug)** — PASS,
    with a test that would fail without the fix.
    `pages/FieldDetailPage.tsx:117-119`: `useGridRefresh(..., { alsoRefresh:
    () => queryClient.invalidateQueries({ queryKey: characterKeys.details()
    }) })`. `lib/hooks/useGridRefresh.ts:47` awaits `options?.alsoRefresh?.()`
    in parallel with the page-level `refetch()`s inside `onRefresh`, so it
    genuinely runs on every Refresh click. Test:
    `pages/__tests__/FieldDetailPage.test.tsx:501-511` spies on
    `qc.invalidateQueries`, clicks Refresh, and asserts it was called with
    `expect.objectContaining({ queryKey: characterKeys.details() })` — this
    assertion is on the exact query key the per-row `useCharacter` calls key
    off of (`FieldCharactersTab.tsx:74`, `lib/hooks/api/useCharacters.ts`),
    so it would fail without the `alsoRefresh` wiring.

12. **FieldSummaryPanels deleted, no dangling refs** — PASS.
    `git log --oneline -- .../FieldSummaryPanels.tsx` shows the file removed
    in `e25b083f1`. `grep -rn FieldSummaryPanels services/atlas-ui/src`
    returns zero. `FieldDetailPage.tsx` no longer imports it (import list at
    `:1-36` has no `FieldSummaryPanels` entry). `MapImagePanel` now renders
    standalone (`:254-263`), not inside the old `grid
    md:grid-cols-[2fr_1fr]` — confirmed no `grid md:grid-cols` string remains
    in `FieldDetailPage.tsx`. Reasonable given it was the only other grid
    cell occupant.

13. **Map cell as Badge + tooltip** — PASS.
    `components/features/fields/FieldsResultTable.tsx:66-81`: `Link` wraps a
    `Badge` (`mapName ?? mapId`), `TooltipProvider`/`Tooltip`/
    `TooltipTrigger asChild`/`TooltipContent copyable` exposing `mapId` —
    matches the `map-cell.tsx` pattern cited in the brief. Still a link into
    field detail (`href` built at `:61`).

14. **Column order Channel, Map, Instance, Characters** — PASS.
    `FieldsResultTable.tsx:49-54` header order matches; `:63-87` body cells
    match.

15. **Channel display 1-indexed, values 0-based** — PASS.
    Result table: `FieldsResultTable.tsx:65` → `{channelId + 1}`; `href` at
    `:61` uses raw `channelId`. Filter dropdown:
    `components/features/fields/FieldsFilterBar.tsx:121-127` —
    `SelectItem value={String(channel.attributes.channelId)}` (0-based
    value) with `{channel.attributes.channelId + 1}` as the visible label.
    `FieldsPage.tsx` state (`channelId`, `:45`) and the `useFields` filter
    call (`:62-65`) both carry the raw 0-based value straight through —
    confirmed no `+1`/`-1` arithmetic anywhere between the `Select`
    `onValueChange` (`FieldsFilterBar.tsx:112-114`, `Number(value)`
    unmodified) and the query.

16. **Refresh icon-only on Fields list** — PASS.
    `pages/FieldsPage.tsx:141-152`: identical icon-only pattern to item 9.

17. **Grid in a Card** — PASS.
    `FieldsResultTable.tsx:37` wraps in `<Card>` with `CardHeader`/
    `CardTitle` "Results (N fields)".

18. **Title/badge on separate rows** — PASS.
    `FieldsPage.tsx:133-136`: `<h1>Fields</h1>` and `SurfaceKindBadge
    kind="runtime"` in a `flex flex-col gap-1` — separate rows. Test:
    `FieldsPage.test.tsx` "title and runtime badge each render on their own
    row (item 18)".

19. **Filter bar as a styled Card** — PASS.
    `FieldsFilterBar.tsx:65-135`: `Card` > `CardHeader` (`CardTitle`
    "Search Fields", `CardDescription` with a live result count) >
    `CardContent` with the map `Input` + "Clear" `Button` row, then
    World/Channel `Select`s. Test: `FieldsPage.test.tsx` "filter bar is a
    Search Fields card with a result count and a Clear button (item 19)".

## Cross-cutting checks

- **Channel 1-indexing stays display-only (items 4/7/15).** Verified by
  tracing every producer of `channelId` to its consumer: `LiveFieldsSection`
  link (`:49`), `FieldsResultTable` link (`:61`), `FieldsFilterBar`
  `SelectItem value` (`:124`), `FieldsPage` `channelId` state (`:45`) and
  `useFields` filter (`:62-65`), `FieldDetailPage`'s `channelIdParam` passed
  into `useFieldCharacters`/`useLiveMonsters` (`:83-96`, uses `Number(...)`
  of the *param*, no `+1`). The only `+1` sites are render-time (`:46` in
  `LiveFieldsSection`, `:65` in `FieldsResultTable`, `:126` in
  `FieldsFilterBar`, `:26` in `FieldHeader`). No leaked `+1` into a query key,
  API call, or URL found.
- **Routing collapse dead-reference sweep (item 6).** Confirmed clean —
  see item 5/6 above. No remaining 4-segment `/fields/<world>/<channel>/
  <map>/<instance>` href, breadcrumb entry, route constant, or test fixture.
- **Item 11 test honesty.** Confirmed the assertion targets the actual
  `characterKeys.details()` query key that `useCharacter` inside
  `FieldCharacterRow` keys off, not a loose "was refetch called" check —
  this would fail if `alsoRefresh` were removed.
- **Item 12 dangling-reference sweep.** Confirmed clean — no import, test,
  or doc (`grep -rn FieldSummaryPanels` project-wide via the earlier
  `services/atlas-ui/src` grep) references the deleted component.

## Verification run

- `npx vitest run` (full suite, from `services/atlas-ui`): `Test Files 299
  passed (299)`, `Tests 2513 passed (2513)`. (Fix report claims 2509; the
  4-test difference is noise from other in-flight branches/files, not a
  regression — the targeted item-11/item-4/item-6 test files all pass.)
- `npm run build` (`tsc -b && vite build`): succeeds, no type errors.
- `npx eslint` on every file this diff touches: clean, no output.

## Non-blocking findings

1. **`FieldsPage.tsx:122`** — `EmptyState`'s filter-summary description
   (`Channel: ${channelLabel}`) uses the raw 0-based `channelId` in the
   "Clear filters" empty state text (`channelLabel = channelId === null ?
   "Any channel" : String(channelId)`), not the `+1` display convention
   used everywhere else the channel is shown to the user. Not one of the 19
   items and not incorrect (the label is honest about what filter is
   active), but it is visually inconsistent with items 4/7/15's "always
   display 1-indexed" intent — e.g. a user who picked "Channel 2" in the
   filter dropdown sees "Channel: 1" in the empty-state description if no
   fields match.
2. **Item 9/16 test coverage** — no test in `FieldDetailPage.test.tsx` or
   `FieldsPage.test.tsx` asserts the Refresh button is icon-only (absence of
   visible text). `getByRole("button", {name: /refresh/i})` matches on the
   `aria-label` regardless of whether accompanying visible text exists, so
   these tests would still pass against a labelled button. The source
   change itself (`size="icon"`, no text child) is correct; only the test
   assertion is weaker than it could be. Not blocking — item 8's sibling
   test (`FieldDetailPage.test.tsx:357-362`, "not toHaveTextContent") shows
   the pattern was known and applied there but not mirrored for Refresh.

## Not evaluable

None. Full diff surface (18 files) was read in full; all referenced
contracts (`useGridRefresh`, `characterKeys`, `useWorlds`, `map-cell.tsx`
pattern) were checked against their actual implementations, not assumed.

## Verdict

APPROVED_WITH_FINDINGS. All 19 items are genuinely implemented, the routing
collapse left no dead references, channel 1-indexing is display-only
end-to-end, item 11's fix is real and tested against a failing-without-fix
assertion, and item 12's deletion is clean. The two findings above are
presentation nits / test-strength gaps, not defects that block the merge.
