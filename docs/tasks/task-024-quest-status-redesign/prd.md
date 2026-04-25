# Quest Status Redesign (Character Detail) — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-04-24
---

## 1. Overview

The Quest Status section on the Character Detail page (`services/atlas-ui/src/pages/CharacterDetailPage.tsx`) currently lists a character's started and completed quests as a single-column stack of rows inside a tabbed card. Each row shows `Quest #<id>`, a raw `#<infoNumber>: <progress>` line pulled verbatim from the backend, and an "Expires:" line that is always a misformatted zero-date today. The content is dense, hard to scan, and mixes identifier noise with actionable information.

This feature replaces that list with a **responsive grid of quest widgets**. Each widget resolves and displays the **quest name** (falling back to `Quest #<id>` when the definition can't be loaded) and is itself a single large click target that navigates to `/quests/:questId`. The raw progress line and the broken expiration line are removed. The Started / Completed tab split is preserved.

The change is atlas-ui only. The backend bug where `ExpirationTime time.Time` serializes as `0001-01-01T00:00:00Z` despite `omitempty` is out of scope and tracked separately.

## 2. Goals

Primary goals:
- Replace the single-column row layout with a responsive grid of widgets on the character detail page.
- Display each quest's human-readable name instead of its numeric ID (fallback to `Quest #<id>` if unresolved).
- Make the entire widget a single click target that navigates to `/quests/:questId`. No secondary "open" button.
- Remove the raw `#<infoNumber>: <progress>` line from the widget.
- Remove the always-broken `Expires:` line from the widget.

Non-goals:
- Redesign of the Inventory, Attributes, or any other Character Detail section.
- Changes to `/quests/:id` (Quest Detail page).
- Fixing the backend `ExpirationTime` `omitempty` serialization bug in `services/atlas-quest/atlas.com/quest/quest/rest.go:22`.
- Adding forfeit / restart / other action buttons to the widget.
- Quest progress visualization (progress bars, objective breakdowns, mob-kill counters, etc).
- Resolving `infoNumber` values to human-readable labels.
- Changing the backend REST surface for character quest status.

## 3. User Stories

- As a GM reviewing a character, I want to scan their started and completed quests by name so that I can identify relevant content without decoding numeric IDs.
- As a GM reviewing a character, I want to click anywhere on a quest widget to open that quest's detail page so that I don't have to aim at a small icon or link.
- As a GM reviewing a character, I want the quest grid to use the full horizontal width of the section so that I can see many quests at once without scrolling through a tall single column.
- As a GM, I want the widget to stay readable when a quest name fails to resolve (e.g. unknown ID, request failure) so that the page doesn't break on data gaps.

## 4. Functional Requirements

### 4.1 Layout

- The Quest Status section continues to render inside the existing `QuestStatusTabs` card in `CharacterDetailPage`. The outer `Card` / `CardHeader` / `Tabs` with Started / Completed triggers, refresh button, and in-progress / completed count summary are preserved.
- Inside each tab's `TabsContent`, the list of `QuestStatusCard` rows is replaced with a responsive CSS grid of widgets:
  - `grid-cols-2` on narrow (default) viewports,
  - `sm:grid-cols-3` at Tailwind's `sm` breakpoint,
  - `lg:grid-cols-4` at `lg`,
  - Gap of `gap-3` between widgets.
- The grid lives inside the existing `ScrollArea` so the tab height stays bounded (current `h-[300px]`).
- Empty states (`No quests in progress`, `No completed quests`) render as today — centered muted text, not a widget slot.

### 4.2 Quest Widget

- Each widget is a single `<Link>` (React Router `Link` from `react-router-dom`) whose `to` is `/quests/${attrs.questId}`. The entire widget area is the click target; there are no nested interactive elements inside the link.
- Visual treatment:
  - Bordered rounded card (`border rounded-lg`).
  - Padding of `p-3`.
  - Hover affordance (`hover:bg-muted/50 transition-colors`) so users recognize the whole card is clickable.
  - Text truncates to prevent layout break on long quest names (`truncate` on the name span; widget has `overflow-hidden`).
- Content inside the widget, in order:
  1. **Quest name** — rendered via the new `QuestName` component (see §4.3). Uses `font-medium`. Truncates to a single line.
  2. **Completion-count badge** — shown only when `attrs.completedCount > 1`. Small `Badge variant="outline"` reading `x<count>`. Positioned inline next to the name (right-aligned on the same row). This is the existing badge behavior preserved.
  3. **Completion timestamp** — on the Completed tab only, if `attrs.completedAt` is present, render a muted secondary line (`text-sm text-muted-foreground`) with the existing `Clock` icon and the formatted date. Uses the existing `formatDate` helper. On the Started tab this line is omitted.
- Content explicitly NOT rendered:
  - Raw `progress` entries (`#<infoNumber>: <progress>`) — removed entirely.
  - `expirationTime` line — removed entirely.
  - The separate `ExternalLink` icon button — removed (the whole widget is the link).

### 4.3 Quest Name Resolution

- Add a new `QuestName` component under `src/components/features/quests/EntityName.tsx` (existing file already contains `NpcName`, `ItemName`, `MobName`, `SkillName`, `JobName`).
- Signature:
  ```ts
  interface QuestNameProps {
      id: number;
      tenant: Tenant | null;
      showId?: boolean;
      className?: string;
  }
  export function QuestName(props: QuestNameProps): JSX.Element;
  ```
- Implementation:
  - Internally calls `useQuest(tenant, String(id))` from `src/lib/hooks/api/useQuests.ts`. The hook's existing 10-minute staleTime and React Query deduplication mean repeated widgets for the same quest share a single request, and repeated renders across navigations hit cache.
  - While the query is loading, render a `Skeleton` sized like a short text line (mirrors the loading behavior of `NpcName`/`ItemName`).
  - On error or missing data, fall back to `Quest #<id>` — same pattern as the other entity-name components.
  - On success, render `questData.attributes.name`. If `showId` is true, append `(#<id>)` in muted text (parity with other `*Name` components, even though the quest widget won't use `showId=true` initially).
- `QuestStatusTabs.tsx` passes the `tenant` prop (already received) through to each widget so `QuestName` can wire the hook.

### 4.4 Interactions

- Clicking anywhere on a widget (including on the name text, the `x<count>` badge area, or empty space inside the widget) navigates to `/quests/:questId` via React Router.
- The refresh button in the `CardHeader` continues to re-run `fetchQuestStatuses`.
- Tabs switch between Started and Completed as today. Default tab remains Started.

### 4.5 Loading, Error, and Empty States

- Loading (initial fetch): preserve the existing `QuestStatusSkeleton` layout.
- Error (fetch failure): preserve the existing error `Card` with the Retry button.
- Empty tab: preserve the existing muted "No quests in progress" / "No completed quests" message.
- Per-widget name load: each widget independently renders a name skeleton until its quest definition resolves. Widgets render their grid position immediately with skeletons so layout doesn't reflow as names come in.

## 5. API Surface

No changes to REST endpoints or request/response shapes. This work only consumes existing endpoints:

- `GET /api/characters/{characterId}/quests/started` (via `questStatusService.getStartedQuests`) — unchanged.
- `GET /api/characters/{characterId}/quests/completed` (via `questStatusService.getCompletedQuests`) — unchanged.
- `GET /api/quests/{id}` (via `questsService.getQuestById` wrapped by `useQuest`) — already exists and is cache-friendly with a 10-minute staleTime.

## 6. Data Model

No data model changes. `CharacterQuestStatus`, `QuestDefinition`, and their attribute shapes in `services/atlas-ui/src/types/models/quest.ts` are consumed as-is.

## 7. Service Impact

- **services/atlas-ui** — only affected service.
  - `src/components/features/quests/QuestStatusTabs.tsx` — rewrite the `QuestStatusCard` function as a `QuestWidget` used inside a grid; remove progress / expiration rendering; switch inner content to use `QuestName`.
  - `src/components/features/quests/EntityName.tsx` — add `QuestName` component using `useQuest`.
  - `src/components/features/quests/index.ts` — export `QuestName` if not already exported.
  - Tests in `src/components/features/quests/__tests__/` (or create this directory if it doesn't exist) — add coverage for the new grid layout, the whole-card link, the fallback rendering, and the completion badge / completed-at behavior.

- **services/atlas-quest** — not affected. The zero-date `ExpirationTime` serialization bug persists but is no longer surfaced in the UI.

## 8. Non-Functional Requirements

- **Performance**: With many quests (e.g. 50+ in a single tab), the widgets must render without perceptible jank. Because `useQuest` deduplicates across widgets and persists in the React Query cache, the first render of this section will issue at most `N` quest-definition requests where `N` = unique quest IDs on both tabs combined. Subsequent renders within the 10-minute staleTime are served from cache. No additional throttling is introduced.
- **Multi-tenancy**: The component continues to receive `tenant: Tenant` and pass it through; tenant headers are already attached by the API client via `TenantProvider`. No tenant-scoping work is added here.
- **Accessibility**: Each widget is a semantic `<a>` (via React Router `Link`) with a concise accessible name derived from the rendered quest name text. Keyboard focus and Enter/Space activation follow default link behavior. Hover and focus styles share the same `bg-muted/50` treatment.
- **Observability**: No new telemetry. Errors loading individual quest definitions fall back silently to `Quest #<id>` per existing `*Name` component conventions.
- **Responsive**: Widgets reflow per §4.1. Long quest names truncate with `truncate` and rely on browser-default tooltip via the `title` attribute populated with the resolved name (or `Quest #<id>` fallback).

## 9. Open Questions

None at drafting time. Decisions on grid density, tabs vs. sections, name-resolution strategy, fallback behavior, fixed inclusions (completion badge, completed-at), and exclusions (progress line, expiration line, external-link button) are documented in §4.

## 10. Acceptance Criteria

- [ ] On the Character Detail page, the Quest Status section renders quest widgets in a CSS grid: 2 columns at the default breakpoint, 3 at `sm`, 4 at `lg`.
- [ ] Each widget displays the resolved quest name; if the definition is still loading, a skeleton placeholder occupies the name slot; if the definition fails to load, `Quest #<id>` renders instead.
- [ ] Clicking any part of a widget (name text, badge area, empty padding, hover region) navigates to `/quests/<questId>` via React Router without a full page reload.
- [ ] The raw `#<infoNumber>: <progress>` line does not appear anywhere in the widget.
- [ ] The `Expires:` line does not appear anywhere in the widget.
- [ ] The separate `ExternalLink` icon button does not appear in the widget.
- [ ] Widgets on the Completed tab display the existing completion-timestamp line (when `completedAt` is present) using `formatDate`. Widgets on the Started tab do not.
- [ ] Widgets with `completedCount > 1` display the `x<count>` badge; widgets with `completedCount <= 1` do not.
- [ ] The `Started (N)` / `Completed (N)` tab counts continue to reflect the fetched list lengths.
- [ ] The section's loading skeleton, error Card with Retry, and empty-state messages are unchanged.
- [ ] The Refresh button continues to re-run `fetchQuestStatuses`.
- [ ] `npm run build` and `npm run test` pass in `services/atlas-ui`.
