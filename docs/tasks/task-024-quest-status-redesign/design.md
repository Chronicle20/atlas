# Quest Status Redesign (Character Detail) — Design

Version: v1
Status: Approved
Created: 2026-04-24
PRD: [prd.md](prd.md)

---

## 1. Purpose

This document resolves the architectural and implementation-shape questions left
open by the PRD. The PRD specifies *what* changes on the Character Detail page's
Quest Status section (responsive grid of widgets, resolved quest names,
whole-card link, removed progress / expiration lines). This design specifies
*how* the change lands in the atlas-ui codebase.

## 2. Scope

Only `services/atlas-ui` is touched. Within atlas-ui, three production files
are edited and two test files are added:

| File | Action |
| --- | --- |
| `src/components/features/quests/EntityName.tsx` | Add `QuestName` export |
| `src/components/features/quests/index.ts` | Add `QuestName` to the existing `EntityName.tsx` barrel re-export |
| `src/components/features/quests/QuestStatusTabs.tsx` | Replace `QuestStatusCard` with `QuestStatusWidget`; swap row container for grid |
| `src/components/features/quests/__tests__/QuestName.test.tsx` | New — unit coverage for `QuestName` |
| `src/components/features/quests/__tests__/QuestStatusTabs.test.tsx` | New — integration coverage for the tabs + grid + widget |

No changes to `EntityWidget.tsx` are in scope. Its internal `QuestWidget` case
already fetches `useQuest` via `useTenant()` context; extracting `QuestName`
does not require refactoring that existing caller, because both paths share the
same React Query cache entry via the `questKeys.detail` key. A follow-up PR may
collapse `EntityWidget.tsx#QuestWidget` onto `QuestName`, but it is not
required to ship this feature.

No backend changes. No changes to routing, services, hooks, or the API client.

## 3. Architectural decisions

### 3.1 Name primitive — new `QuestName` in the `*Name` family

Decision: add a `QuestName` component to `EntityName.tsx` alongside `NpcName`,
`ItemName`, `MobName`, `SkillName`, and `JobName`.

Rationale: the PRD's widget needs one concern — "render a resolved quest name
or a fallback" — that is a reusable primitive. Placing it in `EntityName.tsx`
gives future callers (other pages that need to show a quest name inline) a
uniform API with the existing `*Name` components. This also matches the
pattern `EntityWidget.tsx` already relies on, so there is no conceptual split
between name resolution in quest widgets and name resolution elsewhere.

Alternative rejected: build resolution logic directly inside the new widget in
`QuestStatusTabs.tsx`. Rejected because it creates a second source of truth
for "how do we display a quest name" in the same directory as
`EntityWidget.tsx`'s existing `QuestWidget`.

### 3.2 Tenant source — `useTenant()` context, not a prop

Decision: `QuestName` calls `useTenant()` internally. Its public signature is
`QuestName({ id, showId?, className? })` — identical to the other `*Name`
components.

Rationale: the entire app tree below `TenantProvider` already pulls tenant
from context. The PRD's initial prop-based proposal predates the finding that
`EntityWidget.tsx#QuestWidget` already uses this pattern. Matching that
existing pattern keeps the `*Name` API uniform and means `QuestStatusTabs`
does not have to thread `tenant` through to each widget only so `QuestName`
can re-obtain what context already has.

Alternative rejected: explicit `tenant: Tenant | null` prop (the PRD's initial
sketch). Rejected because it diverges from the rest of the `*Name` family and
adds plumbing without benefit inside a codebase where tenant is always
available from context.

### 3.3 Widget location — inline in `QuestStatusTabs.tsx`

Decision: the new `QuestStatusWidget` is an inner function in
`QuestStatusTabs.tsx`, replacing the existing inner `QuestStatusCard`
function. It is not extracted to its own file.

Rationale: the widget is tightly coupled to `CharacterQuestStatus` — it
consumes `completedCount` and `completedAt`, both quest-*status*-specific
fields, not generic quest fields. No other page in atlas-ui needs to render a
`CharacterQuestStatus` row. Keeping the widget inline keeps all status-card
chrome in the one file that owns the Quest Status UI.

Alternative rejected: extract to `QuestStatusWidget.tsx` for parity with
`EntityWidget.tsx`. Rejected as premature — no second caller exists; if one
appears later, extraction is trivial.

### 3.4 Truncation tooltip — `title` on `QuestName`'s inner `<span>`

Decision: `QuestName` sets `title` on its rendered `<span>` — populated with
the resolved name on success, or with `Quest #<id>` on the error / missing-
data fallback. Native browser hover tooltip handles the "show full name when
truncated" behavior from PRD § 8.

Rationale: the resolved text lives inside `QuestName`. Pushing it up to the
parent `<Link>` (via children-callback, ref, or a second `useQuest` call in
the widget) would re-couple the widget to `useQuest` — the exact dependency
the `QuestName` extraction was meant to hide. Placing `title` on the span
keeps that encapsulation intact with no extra data flow.

Alternative rejected: widget-level `title` by duplicating `useQuest` in
`QuestStatusWidget`. Rejected because it recreates the coupling the
abstraction removes, with no UX benefit — the truncated text and the tooltip
target are the same element either way.

Alternative rejected: no tooltip at all. Rejected because PRD § 8 calls out
truncation tooltips as an accessibility affordance for long names; the cost
of satisfying it is a single `title` attribute.

### 3.5 EntityWidget left unchanged

Decision: `EntityWidget.tsx` is not edited. Its `QuestWidget` inner function
continues to call `useQuest(activeTenant, String(id))` directly.

Rationale: both `QuestName` and `EntityWidget#QuestWidget` hit the same
React Query cache entry (key: `questKeys.detail(tenant, id)`), so there is no
runtime duplication — the second caller for the same `(tenant, id)` pair
within the 10-minute staleTime window is a cache hit. Refactoring
`EntityWidget` to delegate to `QuestName` would be cosmetic, would touch a
file unrelated to the visible change, and would widen the diff with risk of
regressing the reward/requirement UI that depends on it.

## 4. Component specifications

### 4.1 `QuestName` (new, in `EntityName.tsx`)

```ts
interface EntityNameProps {
    id: number
    showId?: boolean
    className?: string
}

export function QuestName({ id, showId = false, className }: EntityNameProps): JSX.Element
```

Internal wiring:

- `const { activeTenant } = useTenant()`
- `const { data, isLoading, isError } = useQuest(activeTenant, String(id))`
- The `useQuest` hook is already `enabled: !!tenant?.id && !!id`, so the
  pre-tenant-selection case is a safe no-op returning `isLoading: true`.

Render cases:

| State | Output |
| --- | --- |
| `isLoading` | `<Skeleton className="h-4 w-16 inline-block" />` (parity with `NpcName`) |
| Success (`data?.attributes.name` present) | `<span className={className} title={name}>{name}{showId && <span className="text-muted-foreground ml-1">(#{id})</span>}</span>` |
| Error or missing name | `<span className={className} title={\`Quest #${id}\`}>Quest #{id}</span>` |

The `title` attribute is intentionally set on the span, not a wrapping
element, so that browser-default tooltips fire when hovering the truncated
text itself.

### 4.2 `QuestStatusWidget` (new inner component in `QuestStatusTabs.tsx`)

```ts
interface QuestStatusWidgetProps {
    quest: CharacterQuestStatus
    showCompletionTime?: boolean
}

function QuestStatusWidget({ quest, showCompletionTime }: QuestStatusWidgetProps): JSX.Element
```

Render:

```tsx
<Link
  to={`/quests/${attrs.questId}`}
  className="block border rounded-lg p-3 overflow-hidden hover:bg-muted/50 transition-colors"
>
  <div className="flex items-center justify-between gap-2 min-w-0">
    <QuestName id={attrs.questId} className="font-medium truncate" />
    {attrs.completedCount > 1 && (
      <Badge variant="outline" className="text-xs shrink-0">
        x{attrs.completedCount}
      </Badge>
    )}
  </div>
  {showCompletionTime && attrs.completedAt && (
    <div className="mt-1 text-sm text-muted-foreground flex items-center gap-1">
      <Clock className="h-3 w-3" />
      {formatDate(attrs.completedAt)}
    </div>
  )}
</Link>
```

Key invariants:

- The `<Link>` IS the card. No nested interactive elements. No `ExternalLink`
  icon button.
- `min-w-0` on the inner flex row enables `truncate` on `QuestName`'s span.
- `shrink-0` on the badge prevents it from being squeezed when the name
  truncates.
- `showProgress` is dropped from the prop surface — progress rendering is
  removed.
- `expirationTime` rendering is removed entirely.
- `formatDate` helper stays in `QuestStatusTabs.tsx` (only caller).

### 4.3 `QuestStatusTabs` — container changes

Inside each `TabsContent`'s `ScrollArea`, the inner container changes:

```diff
- <div className="space-y-3">
-   {quests.map(q => <QuestStatusCard key={q.id} quest={q} showProgress />)}
- </div>
+ <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-3">
+   {quests.map(q => <QuestStatusWidget key={q.id} quest={q} />)}
+ </div>
```

On the Completed tab, widgets receive `showCompletionTime`. The Started tab
omits it.

Imports:

- Remove `ExternalLink` from `lucide-react` (unused after the icon-button is
  deleted).
- Keep `Clock` (still used by the completion-timestamp row).
- Add `QuestName` from `./EntityName`.

Outer `Card` + `CardHeader` + Refresh button + tab triggers + count summary
+ loading skeleton + error Card + empty-state messages — all unchanged.

## 5. Data flow

1. `QuestStatusTabs` mounts → `fetchQuestStatuses` fires → two parallel
   requests to `questStatusService.getStartedQuests` and `getCompletedQuests`.
2. Each returned `CharacterQuestStatus` drives a `<QuestStatusWidget>` in the
   grid.
3. Each widget renders `<QuestName id={attrs.questId} />`.
4. `QuestName` pulls `activeTenant` from context and calls
   `useQuest(activeTenant, String(id))`. First unique `(tenant, id)` pair →
   HTTP request; subsequent callers → React Query cache hit (10-minute
   staleTime).
5. On resolution, the widget's name span swaps from skeleton to text; layout
   reflow is one-line-height per widget, acceptable for the PRD's "no
   perceptible jank" NFR.

## 6. Loading, error, empty states

All three preserved per PRD § 4.5:

- Initial fetch loading → existing `QuestStatusSkeleton` (tabs-shaped
  placeholder).
- Fetch error → existing error `Card` with Retry button.
- Empty tab → existing centered muted `"No quests in progress"` /
  `"No completed quests"` message, rendered outside the grid.

Per-widget name resolution loading and error states are handled by
`QuestName` — the widget itself always renders its grid position immediately
so the grid layout never reflows as names come in.

## 7. Testing

### 7.1 Framework

Vitest + `@testing-library/react` + `vi.mock`. New `__tests__` directory
under `src/components/features/quests/`. No Jest-era patterns.

### 7.2 `QuestName.test.tsx` — unit

Mocks `useQuest` and `useTenant`. Cases:

1. Loading → renders a `Skeleton` element.
2. Success → renders `attributes.name`; span carries `title={name}`.
3. Error → renders `Quest #<id>`; span carries `title="Quest #<id>"`.
4. Missing data (not loading, not error, `data: undefined`) → same fallback
   as case 3.
5. `showId` → appends a muted `(#<id>)` span after the name.

### 7.3 `QuestStatusTabs.test.tsx` — integration

Mocks `questStatusService` fetchers, `useTenant` (returns a fake active
tenant), and `QuestName` (renders `Quest #<id>` synchronously so the
integration test does not re-exercise resolution logic). Wrapped in a
`MemoryRouter` so `Link` renders an anchor with a real `href`. Cases:

1. Default tab is Started; grid container carries `grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-3`.
2. Empty started list → `"No quests in progress"` muted message; grid not
   rendered.
3. Empty completed list on Completed tab → `"No completed quests"`.
4. Started widgets: each is an `<a href="/quests/<id>">`; no `ExternalLink`
   button present; no `#<infoNumber>` text present; no `"Expires:"` text
   present.
5. `completedCount > 1` → `x<count>` badge visible; `<= 1` → no badge.
6. On the Completed tab, widgets with `completedAt` show the formatted
   date row with a `Clock` icon; on the Started tab that row is absent even
   when `completedAt` is populated.
7. Refresh button click → fetchers called a second time.
8. Error path → existing error Card with Retry renders; clicking Retry
   re-runs the fetchers.

### 7.4 Explicitly not tested

- Actual layout behavior at breakpoint widths — class names are asserted,
  viewport-sized layout is a toolchain concern, not a unit-test concern.
- `formatDate` output — existing helper, unchanged.

## 8. Risks and open questions

None at drafting time.

The PRD's § 9 "Open Questions" is also empty. The two architectural surfaces
that needed resolution — reuse of `EntityWidget` vs. new widget, and tenant
source for `QuestName` — are resolved in §§ 3.1–3.3 and 3.5.

## 9. Out of scope (reaffirmed from PRD)

- Backend `ExpirationTime` serialization bug at
  `services/atlas-quest/atlas.com/quest/quest/rest.go:22`.
- Any redesign of Quest Detail (`/quests/:id`).
- Progress visualization, `infoNumber` label resolution, forfeit/restart
  buttons.
- Refactor of `EntityWidget.tsx#QuestWidget` to delegate to `QuestName`
  (possible follow-up).
- Migration of the legacy Jest test suite.
