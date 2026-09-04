# bug-fields-ui fix report — items 5-19 (Fields list, Field detail, routing/breadcrumb)

Scope: items 5 through 19 of `bug-fields-ui-feedback.md` — the Fields list
page, the Field detail page, and the routing/breadcrumb collapse. Item 6's
`<Link to={...}>` target update in `LiveFieldsSection.tsx` was also done
(explicitly carved out for this agent); no other part of that file, and
`MapEntitySummary.tsx`, were touched (items 1-4 belong to a second agent).

## What was implemented

### Routing collapse (items 5, 6)
- `src/App.tsx`: removed the `/fields/:worldId/:channelId/:mapId/:instanceId`
  route and its `FieldDetailPage` lazy import. `/fields` now renders only
  `FieldsPage`.
- `src/lib/breadcrumbs/routes.ts`: removed the four nested
  `/fields/[worldId]…` grouping nodes and the `FIELD_DETAIL` constant. Only
  the `/fields` → "Fields" node remains, so the breadcrumb for any
  `/fields` URL (list or detail-via-query-param) resolves to "Home / Fields".
- `src/lib/breadcrumbs/__tests__/fields-routes.test.ts`: rewritten for the
  collapsed shape (no nested-trail test, `FIELD_DETAIL` asserted absent).
- `src/pages/FieldsPage.tsx`: dispatches to `<FieldDetailPage />` when
  `?instance=` is present in the query string; otherwise renders the list.
  `FieldDetailPage` takes no props — it reads world/channel/map/instance
  from its own `useSearchParams()` call.
- `src/pages/FieldDetailPage.tsx`: reads `world`/`channel`/`map`/`instance`
  from `useSearchParams()` instead of `useParams()`. `paramsValid` logic is
  otherwise unchanged.
- Every producer of the old `/fields/<world>/<channel>/<map>/<instance>`
  href was swept and converted to the query-param form:
  `src/components/features/maps/LiveFieldsSection.tsx` (mine, item 6 tie-in)
  and `src/components/features/fields/FieldsResultTable.tsx` (mine, item 13
  owner). `src/components/app-sidebar-items.ts` was confirmed to already
  point at plain `/fields` (grepped, no change needed).

### Field detail page (items 7, 8, 9, 11, 12)
- `src/components/features/fields/FieldHeader.tsx`: map name is now the
  sole content of the `<h1>`; the row below it holds
  `SurfaceKindBadge kind="runtime"` plus `Badge`s for World (resolved name
  or raw id fallback), Channel (`+1` for display only), and Instance. The
  "View Map Definition" link and `mapId` prop were removed from this
  component (moved to the page's action row).
- `src/pages/FieldDetailPage.tsx`:
  - "View Map Definition" is now a `Tooltip`-wrapped icon-only `Link`
    (lucide `Map` icon, `aria-label="View Map Definition"`, styled via
    `buttonVariants({ variant: "outline", size: "icon" })`) in the action
    row next to Refresh.
  - Refresh is icon-only (`size="icon"`), same `Tooltip` treatment,
    `aria-label="Refresh"`, keeps `disabled`/`aria-busy`/spin behaviour.
  - `useGridRefresh` now takes `alsoRefresh: () =>
    queryClient.invalidateQueries({ queryKey: characterKeys.details() })`
    (real bug, item 11) — `FieldCharacterRow`'s per-row `useCharacter` query
    is not itself in the refetch list, so without this the character x/y
    shown in the Characters tab never updated on Refresh.
  - `FieldSummaryPanels` usage removed; `MapImagePanel` no longer sits in a
    `grid md:grid-cols-[2fr_1fr]` — it now renders full width above
    `FieldTabs` (only consumer of that grid was the deleted panel).
- `src/components/features/fields/FieldSummaryPanels.tsx` deleted (`git rm`)
  per item 12. Swept for other references — none remained after the page
  edit.

### Field Characters tab (item 10)
- `src/components/features/fields/FieldCharactersTab.tsx`: dropped the
  "Character ID" column. The Name cell is now the clickable `Link`,
  `Tooltip`-wrapped (`TooltipProvider`/`Tooltip`/`TooltipTrigger asChild`/
  `TooltipContent copyable`), exposing the raw id as copyable tooltip
  content — same pattern as `map-cell.tsx`. The unresolved-row branch still
  renders (no link, no tooltip), showing the raw id as the "name" cell.

### Fields list (items 13, 14, 15, 16, 17, 18, 19)
- `src/components/features/fields/FieldsResultTable.tsx`: wrapped in a
  `Card` with a "Results (N fields)" `CardHeader`. Column order is now
  Channel, Map, Instance, Characters. Channel displays `channelId + 1`; the
  link/query value stays 0-based. The Map cell is a `Badge` (map name, or
  the raw id as fallback text) wrapped in the same copyable-tooltip pattern
  as `map-cell.tsx`, linking to `/fields?world=…&channel=…&map=…&instance=…`.
- `src/components/features/fields/FieldsFilterBar.tsx`: wrapped in a Card
  matching `ItemsPage`'s Search card (`CardHeader`/`CardTitle`
  "Search Fields"/`CardDescription` with a live result count,
  `CardContent` with the Input+Clear row and the World/Channel selects
  below). Channel `SelectItem` *values* stay 0-based; only the visible
  label is `channelId + 1`. Added `resultCount` and `onClear` props.
- `src/pages/FieldsPage.tsx`: "Fields" title and `SurfaceKindBadge` now sit
  on separate rows (own `flex-col` group). Refresh is icon-only with the
  same `Tooltip` treatment as the detail page. Passes `resultCount` /
  `onClear` through to `FieldsFilterBar`.

## Testing

Module-local, from `services/atlas-ui`:

```
npx vitest run
```
Result: `Test Files 299 passed (299)`, `Tests 2509 passed (2509)`. Output
pristine apart from one pre-existing, unrelated `"Not implemented:
navigation to another Document"` jsdom notice (not from any file touched
here).

```
npm run build
```
Result: `tsc -b && vite build` succeeded (`✓ built in 1.58s`) — this is the
project's type-check gate and it passed with no errors across all touched
files, including tests.

```
npx eslint <all touched .ts/.tsx files>
```
Result: no output — clean.

Individual files run in isolation during development (all passing,
included in the full-suite run above):
- `src/lib/breadcrumbs/__tests__/fields-routes.test.ts` — 3/3
- `src/components/features/fields/__tests__/FieldCharactersTab.test.tsx` — 9/9
- `src/components/features/maps/__tests__/LiveFieldsSection.test.tsx` — 8/8
- `src/pages/__tests__/FieldsPage.test.tsx` — 19/19
- `src/pages/__tests__/FieldDetailPage.test.tsx` — 13/13

### Notable test changes (new behaviour, not deletions of coverage)
- Removed the three `FieldSummaryPanels`-specific tests from
  `FieldDetailPage.test.tsx` (character count / monster grouping / tracked
  object count) — the component they exercised is deleted per item 12; the
  character/monster counts are still asserted via the tab labels
  ("tabs render with counts").
- `FieldDetailPage.test.tsx` gained a new test asserting the item-11 fix
  directly: clicking Refresh calls
  `queryClient.invalidateQueries({ queryKey: characterKeys.details() })`.
  This required changing the file's `vi.mock("@/lib/hooks/api/useCharacters"
  , ...)` to spread `vi.importActual(...)` (keeping the real `characterKeys`
  export) while still overriding `useCharacter`.
- `FieldsPage.test.tsx` gained tests for items 6 (dispatch to
  `FieldDetailPage` on `?instance=`, `FieldDetailPage` itself stubbed since
  its own rendering is `FieldDetailPage.test.tsx`'s job), 18 (title/badge on
  separate rows), and 19 (Search Fields card + Clear button). Two
  pre-existing tests ("world and channel options come from the API",
  "empty state echoes the filters by name") were updated to select the
  channel option by its new 1-indexed label while asserting the underlying
  0-based value is what actually gets used.
- `FieldCharactersTab.test.tsx`: "renders the settled columns" now checks
  the id is *not* visible until the Name link is hovered, then
  `screen.findByText` after `userEvent.hover`. Added a
  "has no Character ID or State column" test and a dedicated tooltip test
  for item 10.

## Files changed

- `src/App.tsx`
- `src/lib/breadcrumbs/routes.ts`
- `src/lib/breadcrumbs/__tests__/fields-routes.test.ts`
- `src/pages/FieldsPage.tsx`
- `src/pages/FieldDetailPage.tsx`
- `src/components/features/fields/FieldHeader.tsx`
- `src/components/features/fields/FieldCharactersTab.tsx`
- `src/components/features/fields/FieldsResultTable.tsx`
- `src/components/features/fields/FieldsFilterBar.tsx`
- `src/components/features/fields/FieldSummaryPanels.tsx` (deleted)
- `src/components/features/maps/LiveFieldsSection.tsx` (Link target only)
- `src/pages/__tests__/FieldsPage.test.tsx`
- `src/pages/__tests__/FieldDetailPage.test.tsx`
- `src/components/features/maps/__tests__/LiveFieldsSection.test.tsx`
- `src/components/features/fields/__tests__/FieldCharactersTab.test.tsx`

## Self-review findings

- Confirmed `MapEntitySummary.tsx` was never opened or edited (item 2 is out
  of scope for this agent).
- Confirmed the icon-button + tooltip composition avoids double-`asChild`
  nesting: the Refresh button is `TooltipTrigger asChild > Button` (single
  Slot layer); "View Map Definition" is `TooltipTrigger asChild > Link`
  styled via `buttonVariants` rather than `TooltipTrigger asChild > Button
  asChild > Link` (which would stack two `Slot` layers) — this follows the
  existing `NpcsPage.tsx`/`MonsterTableRow.tsx` convention of a styled
  anchor directly under `TooltipTrigger asChild`.
- Swept `grep -rln FieldSummaryPanels src` post-deletion — zero remaining
  references.
- Swept `grep -rn '/fields/' src` (excluding the two already-fixed
  producers, the breadcrumb node history, and this report) — no other
  literal `/fields/<segments>` href construction remained.

## Issues or concerns

- None outstanding. The single ruling in the brief (item 6's single-route
  collapse) was followed as specified — no need to fall back to the
  `/fields/:instanceId` alternative.
- `FieldsResultTable`'s `Card` is not `flex-1 min-h-0` the way
  `ItemsPage`'s Results card is; `FieldsPage`'s existing
  `overflow-y-auto` container already scrolls the whole page, so this was
  left simple rather than importing `ItemsPage`'s sticky-header layout,
  which was not requested.

## Review follow-up — closing the two non-blocking findings

`docs/tasks/task-292-map-definition-field-split/reviews/bug-fields-ui.md`
flagged two non-blocking findings (APPROVED_WITH_FINDINGS verdict). Both
closed:

1. **`FieldsPage.tsx` EmptyState channel label was 0-based.**
   `channelLabel` (`FieldsPage.tsx:122`) now reads
   `channelId === null ? "Any channel" : String(channelId + 1)`, matching
   the 1-indexed display convention used everywhere else the channel is
   shown (items 4/7/15). No API value, query key, `SelectItem` value, or
   URL param was touched — `channelId` state and the `useFields` filter
   call are unchanged. Updated the one existing test that asserted the old
   (bug) behaviour: `FieldsPage.test.tsx` "empty state echoes the filters
   by name, not a missing map" selected the channel-4 option (raw
   `channelId` 3) and asserted the empty state showed `"3"`; it now
   asserts `"4"`, matching the selected option's own label.

2. **Refresh icon-button tests only asserted `aria-label`.** Added a
   sibling assertion in both `FieldDetailPage.test.tsx` and
   `FieldsPage.test.tsx` — "Refresh is an icon-only button, not a labelled
   button" — using the same `not.toHaveTextContent` pattern as the
   existing item-8 "View Map Definition" test:
   ```ts
   const button = screen.getByRole("button", { name: /refresh/i });
   expect(button).not.toHaveTextContent(/refresh/i);
   ```
   This fails if the button ever grows a visible "Refresh" text child
   alongside its `aria-label`, closing the gap the review called out.

### Files changed
- `src/pages/FieldsPage.tsx` — channel label 1-indexing fix.
- `src/pages/__tests__/FieldsPage.test.tsx` — updated the empty-state
  channel-label assertion to the corrected 1-indexed value; added the
  Refresh icon-only test.
- `src/pages/__tests__/FieldDetailPage.test.tsx` — added the Refresh
  icon-only test.

### Verification (module-local)
- `npx vitest run src/pages/__tests__/FieldsPage.test.tsx
  src/pages/__tests__/FieldDetailPage.test.tsx` → `Test Files  2 passed
  (2)`, `Tests  34 passed (34)`.
- `npx eslint src/pages/FieldsPage.tsx
  src/pages/__tests__/FieldsPage.test.tsx
  src/pages/__tests__/FieldDetailPage.test.tsx` → `ESLint: No issues
  found`.
- `npx tsc -b` → no output (clean type-check across the whole app,
  including test files per `tsconfig.app.json`).
