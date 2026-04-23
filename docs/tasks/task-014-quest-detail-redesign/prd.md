# Quest Detail Redesign — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-04-20
---

## 1. Overview

`QuestDetailPage` in atlas-ui today (`services/atlas-ui/src/pages/QuestDetailPage.tsx`) is a collage of four loosely-related UI patterns: a prose header with a mandatory "Quest #{id}" literal; a "Quest Information" card with a title + description line that mostly repeats the header; a 2×2 grid of independently-collapsible cards for Start Requirements / Start Actions / Completion Requirements / Completion Rewards; an opportunistic "Quest Chain" card that shows `View Quest #{nextQuest}` as a raw integer; and — as of task-012 follow-up work — a new Conversation card that renders both state machines as tabs. The page scrolls only because the outer wrapper was retrofitted with `overflow-y-auto` during the conversation-card work; the underlying layout was never designed for that scroll to exist.

The information the page *does* surface is good — requirements, rewards, the full conversation, the quest chain — but the framing pulls the reader in four directions. Requirements and actions are separated by completion phase *and* by "requirements vs actions", so a reviewer checking "what does Cassandra's quest accept look like end to end" has to assemble the picture from two cards on the grid. The brand-new conversation editor, meanwhile, lives in its own card outside the Start/Completion dichotomy — the start state machine (which runs on quest acceptance) and the end state machine (which runs on completion) are squeezed into tabs inside one card, rather than co-located with the requirements and actions they belong to.

This task restructures the page around the quest's two actual lifecycle phases — **Start** and **Completion** — and makes every cross-reference a visual widget instead of a raw id. The header gets the task-012 treatment (quest name as primary title, category as a badge, a tooltip-to-copy id). The middle of the page becomes a single column: a slim non-collapsible metadata strip, a non-collapsible quest-chain row that resolves `nextQuest` to a human-readable name, and then two parallel "Start" and "Completion" cards each containing three sections — Requirements, Actions, and the corresponding state machine pulled out of the conversation editor's tabs. Lists that can grow long (jobs, items, mobs) are rendered as wrapping widget grids rather than single inline rows that overflow the card. Every numerical id (npc, item, mob, map, skill, next quest, buff item) becomes an icon+name widget with the raw id in a copyable tooltip — mirroring the convention the item/monster/map/npc redesigns landed on.

The existing quest-conversation card component (`QuestConversationCard`, shipped in task-013 follow-up work, with its internal tabs switching between `startStateMachine` and `endStateMachine`) is split so each state machine becomes a section of its owning lifecycle card rather than a tab. The underlying editor (`ConversationEditorPanel`) is reused — only the outer wrapping changes. The draft/save/revert controls stay at the quest level, not per-machine. No backend changes are required for this task; the quest definition is already returned in full by `atlas-data`, and the quest conversation is already returned in full by `atlas-npc-conversations`.

Out of scope is any write-path support for quest *definitions* — those live in JSON sources under `atlas-quest` and are not mutable from the UI. The new surface edits quest conversations only. Also deferred: character-quest-status overlays (live player progress on this quest), and multi-hop chain visualization (>1 hop) beyond a single "next quest" link.

## 2. Goals

Primary goals:

- Collapse the page onto a single column whose structure matches the quest's two real lifecycle phases: Start and Completion.
- Make every numerical cross-reference (NPC, item, mob, map, skill, next-quest, buff-item) a clickable icon+name widget with the raw id in a tooltip, eliminating the current "ID: N" inline pattern.
- Co-locate the start state machine with start-phase requirements + actions, and the end state machine with completion-phase requirements + actions — retiring the separate Conversation card's internal tabs.
- Drop the literal `Quest #{id}` heading and card title/description chrome; put the quest name in the header and the id behind a tooltip-to-copy affordance.
- Resolve `endActions.nextQuest` to the target quest's human-readable name so the Quest Chain row reads naturally.
- Render long list-valued fields (jobs, items, mobs, maps, quests, pets, skills) as wrapping widget grids that stay within their card.

Non-goals:

- Editing the quest definition itself (name, requirements, actions, rewards). Only the conversation state machines remain editable. Quest definitions are supplied by `atlas-data` from static sources.
- Surfacing live character quest status on this page (who has this quest open, completion count, etc.).
- Backward quest chain visualization (predecessor lookup). Forward-only via `endActions.nextQuest`, which is what the schema gives us natively.
- Tree-style multi-hop chain views. "Next quest" is a one-hop link; clicking it lands on that quest's detail page.
- A new filter/search surface on `QuestsPage`.
- Backend changes. Every field needed is already on the existing quest definition + quest conversation responses.
- Rendering `startScript` / `endScript` fields. The fields exist on the TypeScript model but the corpus never populates them — the state machine has fully replaced the slot. The renderer ignores them.
- Quest page icon next to the name (no art asset exists; empty space is cleaner than a generic glyph).
- Collapsible cards. Every card on the page is fixed / non-collapsible. Long quests rely on the outer page's `overflow-y-auto` scroll.
- A navigation target for `kind: "skill"` widgets. atlas-ui has no `/skills/:id` detail page yet; skill widgets render with icon + name + copyable id tooltip but are not clickable until a skill detail page ships.

## 3. User Stories

- As a content designer reviewing a quest, I want the page to read top-to-bottom as "here's the quest, here's its chain, here's everything about starting it, here's everything about completing it" — without bouncing between columns.
- As a GM triaging "what does this quest reward", I want every reward item rendered with its icon and name so I recognize it at a glance, and I want the raw item id in a tooltip I can copy for ad-hoc queries.
- As a quest writer iterating on acceptance dialogue, I want the start state machine editor to sit directly next to the start requirements + actions it runs alongside, not behind a tab on a separate card.
- As a designer doing a patch-notes pass, I want to click the quest's category badge and the name of the next quest in the chain rather than manually retyping either.
- As an operator investigating "why is this quest's next-quest link broken", I want `endActions.nextQuest = 2217` resolved to `"Dyle's Request"` so I know which quest I'm jumping to before I click.
- As a reviewer auditing a long-running quest (say, a job advancement with ~40 required items), I want items rendered as a wrapping widget grid that fits the card rather than a single inline comma list that cuts off the card edge.
- As a content engineer authoring a quest conversation, I want to continue using the Save / Revert model established in task-012's NPC detail work — one save persists the whole quest conversation, including any edits I made to both state machines.
- As any operator, I want the top-of-page scroll to actually work — the page's contents are tall enough that the conversation-graph sections need room to breathe without losing the header.

## 4. Functional Requirements

### 4.1 Page shell

The outer container becomes a single-column scrolling stack: `flex flex-col flex-1 space-y-6 p-10 pb-16 overflow-y-auto`, matching `NpcDetailPage` post-task-012. The `<Toaster />` mount stays at the bottom of the tree but is not a visible card.

### 4.2 Header

Replaces the current heading block plus the "Quest Information" `CardTitle`/`CardDescription`:

- **Quest name** as the primary heading (`<h2 class="text-2xl font-bold">`). If the quest definition has no name, fall back to `"(Unnamed Quest)"`. No icon to the left of the name — quests have no authored art asset in the corpus, and we prefer no icon over a generic glyph.
- **Category badge** (`attributes.parent`) rendered with `Badge variant="outline"` right of the name, and — when not null — wrapping as a link to a future `/quests?category=…` filter (not required; render the badge even if not linked for now).
- **Quest id** accessible via a tooltip-to-copy affordance on a small `#` icon or subdued id chip to the right of the name. Pattern matches the template-id treatment established on `MapHeader` / `MonsterHeader` / the item/npc detail pages.
- **Back button** on the left stays (`ArrowLeft`).

No `RefreshCw` button — consistent with the direction the other detail pages took in task-008 / 010 / 011 / 012.

### 4.3 Quest Information card

A **fixed, non-collapsible** `<Card>` with no `CardHeader` (no title, no description — both are redundant now that the header carries the name and context). The card body shows the metadata grid that currently sits inside the card, plus the summary block:

- Left grid (4 equal columns on `md:`+, 2 columns on mobile): **Auto Start**, **Auto Complete**, **Time Limit**, **Area / Order**. Rendering rules match today's:
  - `autoStart === true` → `Badge` with `Zap` icon and the label "Yes"; `false` → text "No".
  - `autoComplete === true` → secondary `Badge` with `CheckCircle` and "Yes"; `false` → text "No".
  - `timeLimit > 0` → outline `Badge` with `Clock` icon and `formatTime(seconds)` (helper already lives in the file).
  - Area / Order → `{area ?? 0} / {order ?? 0}`.
- Below the grid, when any of `summary`, `demandSummary`, `rewardSummary` are set, render a stacked `<div>` list with small label + muted-foreground text for each present field. Separator line above with a top margin (existing behavior).

If every field is empty, the card still renders but the metadata grid is enough to justify it (auto-start defaults, area=0 etc.); do not conditionally hide the card.

### 4.4 Quest Chain card

Shown only when `attributes.endActions.nextQuest` is set. A **non-collapsible** `<Card>` with no separate title — a tight row with:

- Small muted prefix "Next quest →"
- A button that links to `/quests/{nextQuest}` and displays the target quest's `name` (resolved via `useQuest(nextQuest)`), not the id. If the lookup is loading, show "Loading…"; if the lookup 404s (broken chain), show `"Quest #{nextQuest}"` with a destructive `AlertTriangle` icon and a tooltip "Target quest not found".
- The raw target id in a copyable tooltip on the same row, same pattern as §4.2.

This replaces the current `ArrowRight`-headed card with its separate title+description+body.

### 4.5 Conversation save toolbar

A single Save / Revert / Unsaved-changes row sits immediately above the Start card — the placement nearest the conversation-editor sections it controls. The toolbar is hidden entirely when `useQuestConversation` resolves to `null` (no conversation defined for this quest — nothing to save). When a conversation is present but neither state machine has been touched, the buttons are disabled with "Unsaved changes" hidden; on first edit the indicator appears and the buttons enable. Saving PATCHes the whole quest conversation in one call (§4.9).

### 4.6 Start card

A **fixed, non-collapsible** `<Card>` with `CardHeader` title `"Start"` (no description). Contains three `section` blocks separated by subtle dividers — no nested cards, no collapsibles. Order:

1. **Requirements** — renders `attributes.startRequirements` via the rewritten `RequirementGrid` (§4.8). Even when empty, show "No start requirements." as muted text so the section is discoverable.
2. **Actions** — renders `attributes.startActions` via the rewritten `RewardGrid` (§4.8). Same empty-state handling.
3. **State machine** — renders the quest conversation's `startStateMachine` via `ConversationEditorPanel`. When there is no quest conversation for this quest, show "No start conversation defined." muted text.

### 4.7 Completion card

Mirrors §4.6 for the completion phase:

1. **Requirements** — `attributes.endRequirements`
2. **Actions** — `attributes.endActions`, minus `nextQuest` which is already rendered in §4.4 (don't double-show it here).
3. **State machine** — `endStateMachine` when present. When the quest conversation has `startStateMachine` only and no `endStateMachine`, show "No completion conversation defined." muted text.

### 4.8 Requirement / Reward rendering

Replaces `RequirementRenderer` and `RewardRenderer`. The two current components render a `flex flex-wrap gap-2` of chip-style `RequirementItem` / `RewardItem` cards, each a single inline unit. That's fine for scalars (Level, Meso, Fame, Time) but wrong for list-valued fields — today `jobs`, `items`, `mobs`, `quests`, `fieldEnter`, `skills`, `pet` all get crammed into one `.value` string or a comma-separated list that overflows the card horizontally.

Split rendering into two concentric layers:

1. **`RequirementGrid` / `RewardGrid`** — ordered vertically. Scalars come first as a `flex flex-wrap gap-2` of chip items (unchanged visual pattern). List-valued fields come after, each as its own labelled block containing a **wrapping widget grid** (`grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2`). The widget grid contains one `EntityWidget` per list entry (§4.8). This ensures a 40-item rewards list wraps cleanly inside the card at any width.

2. **`EntityWidget`** (§4.8) — the unit that replaces all existing `NpcName` / `ItemName` / `MobName` / `SkillName` / `JobName` inline text components for the case where we're rendering a reference. Those text-only components can stay for inline cases (tooltips, descriptions), but anywhere we render a row or a widget for a referenced entity, use `EntityWidget`.

Scalar fields that stay as chips (no change to shape): `levelMin`/`levelMax` → "Level", `fameMin` → "Fame", `mesoMin`/`mesoMax` → "Meso", `timeLimit`/`timeLimit2`, `dayOfWeek`, `interval`, `completionCount`, `petTamenessMin`, `infoNumber`, `start`/`end` date windows, `normalAutoStart`, `selectedMob`.

**Jobs** (`requirements.jobs[]`) render as a single "Jobs" chip whose value is the comma-joined job names (`JobName` text helper, reused as-is). Jobs have no detail route and no authored icon; a widget grid would be overkill. Treat this field as a scalar-display with a list-valued body.

List-valued fields that become widget grids:

| Field | Widget kind | Label |
|---|---|---|
| `requirements.npcId` (scalar, but EntityWidget) | NPC | "Start NPC" (start) / "End NPC" (end) |
| `actions.npcId` (scalar) | NPC | "NPC" |
| `requirements.quests[]` | Quest (with state badge) | "Quest prereqs" |
| `requirements.items[]` | Item (with `count` chip — negative = consumed, positive = held) | "Items" |
| `requirements.mobs[]` | Mob (with `count` chip) | "Mob kills" |
| `requirements.fieldEnter[]` | Map | "Field enter" |
| `requirements.pet[]` | Item (pet template id — same maplestory.io lookup path as items) | "Pets" |
| `actions.items[]` | Item (with `count`, plus inline `prop`/`period`/`job`/`gender` affixes — existing logic preserved) | "Items" |
| `actions.skills[]` | Skill | "Skills" |
| `actions.buffItemId` (scalar) | Item | "Buff item" |
| `actions.exp`, `actions.money`, `actions.fame`, `actions.levelMin`, `actions.interval` | chips | (scalar) |

Legacy script slots (`requirements.startScript` / `requirements.endScript`) are **not surfaced** on the page. The fields exist on the TypeScript model (`services/atlas-ui/src/types/models/quest.ts:47-48`) and on the atlas-channel transport model, but a corpus sweep of every quest definition and every quest conversation JSON under `services/atlas-npc-conversations/conversations/quests/` and the atlas-data sources shows zero populated instances — the state machine has fully replaced this slot. The renderer ignores them.

### 4.9 `EntityWidget`

A new shared component under `components/features/quests/EntityWidget.tsx` (or `components/common/EntityWidget.tsx` if the shape is broadly reusable). Props:

```ts
interface EntityWidgetProps {
  kind: "npc" | "item" | "mob" | "map" | "skill" | "quest" | "pet";
  id: number;
  // Optional display affixes — caller provides what's relevant to its list context.
  count?: number;         // item/mob counts; negative = consumed
  state?: 0 | 1 | 2;      // quest prereq state (not-started / started / completed)
  prop?: number;          // item reward prop (-1 guaranteed / 0 selection / 1+ chance)
  period?: number;        // item reward period (minutes; 0 = permanent)
  job?: number;           // reward item per-job filter
  gender?: number;        // reward item per-gender filter
}
```

Renders as a link (`react-router-dom` `Link`) to the detail route for that kind:

- `npc` → `/npcs/:id`, uses `useNpcData` for name + iconUrl.
- `item` → `/items/:id`, uses `useItemData` or batch-fetched via `useItemBatchData`.
- `mob` → `/monsters/:id`, uses `useMobData` for name + sprite.
- `map` → `/maps/:id`, uses map data hook (or fall back to the id until one exists).
- `skill` → `/skills/:id` when that route exists, else a no-link chip for now; `useSkillData` already present.
- `quest` → `/quests/:id`, uses `useQuest` for name.
- `pet` → same path as `item` (pets are item templates).

`job` is intentionally not a `kind` — jobs are rendered as a chip via `JobName` (§4.8).

Visual shape matches `NpcShopCommodityWidget` / `MonsterSpawnMapWidget`: a small rounded-md `border bg-card p-2` with `flex items-center gap-3` layout — `8×8` icon on the left (or a fallback `Package`/`UserCircle2`/`Skull`/`MapPin`/`Sparkles` lucide icon when no art), the name + optional affix line in the middle, and — only on hover — a small `#id` badge on the right. The numeric id is always accessible via a tooltip on the widget body.

Affix rendering rules inside the widget body's subtitle line:

- `count` on items: ` × 3` or ` × -1 (consumed)`.
- `count` on mobs: ` × 50` as "50 kills".
- `state` on quests: small badge — "not started" / "in progress" / "completed".
- `prop` on item rewards: "Guaranteed" / "Selection" / `{prop}% chance`.
- `period` on item rewards: when non-zero, "30d" etc., formatted via existing duration helpers.
- `job` on item rewards: comma-joined job names via `JobName`, prefixed "for".
- `gender` on item rewards: "♂" / "♀" / omit for -1.

The widget is clickable as a whole; affixes do not break click-through.

### 4.10 Draft / save model for quest conversation

The quest conversation draft lifecycle stays exactly as it is in `QuestConversationCard` today: one draft holds the full `QuestConversation` (both state machines), edits to either machine mark the draft dirty, Save PATCHes the whole attributes object via `questConversationsService.update(id, attributes)`. The only change is presentation: instead of `TabsList` → `[Start machine, End machine]` with one `ConversationEditorPanel` per tab, the editor panels are rendered in-place inside the Start and Completion cards respectively. The Save / Revert toolbar (§4.5) sits immediately above the Start card — nearest the state-machine sections it controls, without being tied to either one individually (since edits to either dirty the same shared draft).

When the quest has no conversation at all (`useQuestConversation` resolves to `null`), both state-machine sections show the "No start/end conversation defined." muted-text empty state — the Save / Revert toolbar is hidden in that case (nothing to save).

### 4.11 `overflow-y-auto` on the scroll container

The retrofit added in the conversation-work remains. §4.1 codifies it as the canonical shell.

## 5. API Surface

No new or modified endpoints. All data is already available:

- `GET /api/data/quests/:id` — quest definition (already consumed via `useQuest`).
- `GET /api/quests/:questId/conversation` — quest conversation (already consumed via `useQuestConversation`, shipped in the task-013 follow-up work).
- `PATCH /api/quests/conversations/:conversationId` — quest conversation update (already consumed via `questConversationsService.update`).

Per-entity lookups for `EntityWidget`:

- `GET /api/data/npcs/:id` — already used by `useNpcData`.
- `GET /api/data/items/:id` — already used by `useItemData`.
- `GET /api/data/monsters/:id` — already used by `useMobData`.
- `GET /api/data/maps/:id` — already consumed by existing map hooks (name/street lookup).
- `GET /api/data/skills/:id` — already used by `useSkillData`.
- Quest-by-id for the Quest Chain resolution and for `requirements.quests[]` entries — already consumed via `useQuest`.

## 6. Data Model

No persistent schema changes. No new tenant-scoped tables.

All field references are on the existing `QuestAttributes` model defined in `services/atlas-ui/src/types/models/quest.ts` (surfaced through `atlas-data`'s quest resource). The widget rendering changes do not alter the wire shape.

## 7. Service Impact

**atlas-ui** (primary):

- `pages/QuestDetailPage.tsx` — rewrite. Collapse the 4-grid into the Start / Completion card model; move Save/Revert to the page level; swap `RequirementRenderer` / `RewardRenderer` for `RequirementGrid` / `RewardGrid` from §4.7.
- `components/features/quests/RequirementRenderer.tsx` — replaced by `RequirementGrid`. May be deleted or kept as a thin wrapper used by other callers if any exist (grep result expected: none).
- `components/features/quests/RewardRenderer.tsx` — replaced by `RewardGrid`. Same disposition.
- `components/features/quests/EntityWidget.tsx` — new. Takes the role today played by `NpcName` / `ItemName` / `MobName` / `JobName` / `SkillName` in the renderer files. The `EntityName` text components stay around for inline uses (tooltips, prose) but are no longer the primary render path for quest-referenced entities.
- `components/features/quests/conversation/QuestConversationCard.tsx` — keep the draft state machinery; remove the internal `Tabs` wrapping. Export two smaller components:
  - `QuestConversationToolbar` (Save / Revert / Unsaved indicator — promoted to the page level).
  - `QuestConversationMachineEditor({ machine: "start" | "end" })` — renders one `ConversationEditorPanel` wired to the corresponding state-machine slice of the draft. Both editors share the same draft context via a lightweight React context exposed by the top-level provider component, so that changes to either machine mark the shared draft dirty.
  - Alternative without a context: keep the card a single component that exposes a `renderStartMachine` / `renderEndMachine` slot pattern (render-props). Preferred shape depends on how cleanly we can colocate the editors without threading props through the page. Decision at implementation time.
- `pages/__tests__/QuestDetailPage.test.tsx` — update to the new structure. Assert that Start requirements, start actions, and start state machine all render inside the Start card; same for Completion.

**atlas-data**: no change.

**atlas-quest**: no change.

**atlas-npc-conversations**: no change.

**Data migrations**: none.

## 8. Non-Functional Requirements

- **Performance**: `EntityWidget` per-entity lookups hit the existing tenant-scoped REST endpoints. For reward/requirement lists, rely on the already-present batch hooks where they exist (`useItemBatchData`, `useMobBatchData` if/when it exists); otherwise a list of N items will fire N `useItemData` calls — acceptable for the scale typical of a single quest (rarely >20 items per requirement/reward set), and no worse than the current renderer which already calls `useItemData` per reference. If any quest in the corpus has >50 referenced entities in a single list, fall back to the batch hook or inline chips without icons.
- **Multi-tenancy**: all reads and the single Save PATCH go through the existing tenant-scoped service wrappers (`useTenant`-aware). No new tenant injection.
- **Accessibility**: `EntityWidget` is a `<Link>` wrapping a focusable container; icons retain `aria-label`; id tooltips use the standard shadcn `Tooltip` primitive so they surface via keyboard focus as well as hover.
- **404 handling**: for a quest with no conversation, the page must not churn the network with retries — the fix landed in the task-013 follow-up (`retry: (count, err) => err.statusCode === 404 ? false : count < 2`). Keep that shape.
- **Error surfaces**: broken `nextQuest` target (§4.4) does not break the page — renders a destructive-tinted chip with the raw id and a "target not found" tooltip, rather than a toast or error boundary.
- **Scroll**: outer container is `overflow-y-auto`. No element inside imposes an inner scroll — the conversation graph continues to scroll only within its own bounded canvas height, driven by React Flow.

## 9. Open Questions

None of the scope decisions have open questions. Implementation-time choices flagged for resolution:

1. Context vs render-props shape for `QuestConversationCard` after tab removal (§7). Decide once the refactor starts and the natural split becomes obvious.
2. Whether `EntityWidget` lives under `components/features/quests/` or is promoted to `components/common/` for reuse on future detail pages that also need entity-widget rows. Defaulting to `components/features/quests/` for this PR; promote later if another page imports it.
3. Pet-widget icon: pets are item templates (e.g., `5000000`-range), so `useItemData` returns the correct sprite. No special handling required unless we discover pet icons aren't on the maplestory.io asset path — confirm during implementation.
4. Whether the referenced `requirements.quests[]` prereqs need any deduplication when the same prereq appears in both `startRequirements` and `endRequirements`. PRD says render them in whichever section(s) they appear; revisit if the prereq list commonly duplicates across start+end and the duplication is visually noisy.

## 10. Acceptance Criteria

- [ ] The page is a single column with `overflow-y-auto` scrolling applied to the outer `<div>`.
- [ ] Header shows the quest name as the primary `<h2>`; the category is an outline `Badge` to its right when present; the quest id is accessible via a copyable tooltip on a dedicated affordance next to the name. No `Quest #{id}` literal appears in the header.
- [ ] Quest Information card has no `CardTitle` / `CardDescription` — just the metadata grid and, when present, the stacked summary block. It is not collapsible.
- [ ] Quest Chain card renders only when `endActions.nextQuest` is set; it shows `"Next quest →"` plus a button whose label is the target quest's resolved `name` (not its id); the id is accessible via tooltip. Broken targets render with a destructive affordance rather than an error.
- [ ] Two cards labelled `"Start"` and `"Completion"` each contain exactly three sections: `Requirements`, `Actions`, and the corresponding state-machine editor. Neither card is collapsible.
- [ ] The Start card's state-machine section renders `startStateMachine` via `ConversationEditorPanel`; the Completion card's section renders `endStateMachine`. When a machine is absent, the section shows "No start/end conversation defined." muted text.
- [ ] A single Save / Revert / Unsaved-changes toolbar sits immediately above the Start card (locational to the conversation sections it controls) and applies to the entire quest conversation. Editing either state machine marks the draft dirty; Save PATCHes the whole attributes object; Revert restores both to the last-saved conversation. Toolbar is hidden when the quest has no conversation.
- [ ] All scalar requirement/action fields render as chip-style `RequirementItem` / `RewardItem` units. `jobs[]` renders as a single chip whose body is the comma-joined job names (`JobName` helper). All other list-valued fields (`items`, `mobs`, `fieldEnter`, `quests`, `pet`, `skills`) render as a wrapping `EntityWidget` grid within the relevant section — at no width does a list overflow the card.
- [ ] Every numerical reference on the page — `requirements.npcId`, `actions.npcId`, each `items[].id`, each `mobs[].id`, each `fieldEnter` entry, each `quests[].id`, `actions.buffItemId`, each `skills[].id`, `endActions.nextQuest` — is rendered via `EntityWidget` with icon + human-readable name, and the raw id is available via tooltip on the widget. No "#{id}" literal appears on the page outside tooltips.
- [ ] `requirements.startScript` and `requirements.endScript` are not rendered on the page.
- [ ] `useQuestConversation` does not retry on 404; the "No conversation defined for this quest." empty state renders on first render without a spinner churn.
- [ ] No backend changes are shipped with this task.
- [ ] Existing deep links into the quest page (`/quests/:id`) continue to resolve and render identically to the pre-redesign page's core data (metadata + requirements/actions coverage), with the addition of the conversation sections.
- [ ] The legacy `RequirementRenderer.tsx` and `RewardRenderer.tsx` components are either deleted or retained as thin compatibility wrappers only if still imported by non-quest-page callers (grep confirmation at implementation time).
