# Monster Detail Redesign — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-04-19
---

## 1. Overview

`MonsterDetailPage` in atlas-ui today (`services/atlas-ui/src/pages/MonsterDetailPage.tsx:20-151`) is a functional but low-density page: a header row with an id next to the name, three stat cards, an optional skills table, and a flat drops table. For operators investigating monster balance, quest drop chains, or spawn bugs it fails to surface the highest-value information quickly: you can't copy the template id, you can't tell at a glance which compartment a drop belongs to, and you have no way to answer "where does this monster spawn?" without scanning every map.

This task refactors the page to mirror the information-dense pattern established by `MapDetailPage` in task-008: a header with a tooltip-to-copy template id, compressed stat cards, drops grouped by item compartment (Equipment / Consumable / Setup / Etc / Cash + a separate Mesos section), and a new "Spawn Locations" grid sorted by spawn density. The skills card is compressed into name+level chips; this requires adding mob-skill name storage to `atlas-data` alongside the new redesign.

The spawn-location view requires a new aggregation on `atlas-data`: existing map documents store monster spawn entries inside each map document, but there is no per-monster index for a reverse lookup. We add a `monster_spawn_index` table populated during map registration (same hook `map_search_index` uses — see `services/atlas-data/atlas.com/data/map/storage.go:50-75`) and expose it via a new `GET /data/monsters/{monsterId}/maps` endpoint. Re-ingest is required on first deploy; no back-fill migration.

## 2. Goals

Primary goals:
- Give operators an at-a-glance, information-dense view of a monster: stats, drops grouped by compartment, and where it spawns — all without tab-switching.
- Make every reference (drop item, skill id, spawn map) clickable to its detail page so tracing flows are one click away.
- Eliminate the "raw id next to the name" pattern in favor of the tooltip-to-copy pattern already used by `MapHeader` and `map-cell`.
- Expose mob-skill names through the same WZ-string-registry pattern already used by monsters/NPCs/player-skills/items.
- Preserve the existing route `/monsters/:id`, existing breadcrumbs, and existing API contracts for drops/monsters.

Non-goals:
- Editing monster stats, drops, skills, or spawn data from this page.
- Visualizing spawn positions on the map render.
- Filtering/sorting drops by chance or quantity.
- A per-map spawn rate (mob_time) or capacity display — spawn counts only.
- New tenant-wide search indexes beyond the single `monster_spawn_index` table.
- Changes to `MonstersPage` list or columns, or to `ItemDetailPage`'s "Dropped By" section.
- Back-filling the `monster_spawn_index` from existing map documents — a re-ingest is acceptable.
- Rendering mob-skill icons. Skill chips show name + level only; icon lookup is out of scope unless the existing `useSkillData` pipeline transparently works for mob skills (it does not — see §4.5).

## 3. User Stories

- As an operator triaging a drop-rate bug report, I want to land on the monster page and immediately see the drops grouped by compartment so I can spot an item in the wrong section.
- As a GM writing patch notes, I want to hover the monster name and copy its template id into my notes.
- As a designer balancing a quest, I want to see at a glance which maps this monster spawns on, sorted by density, so I can pick a training map that matches the intended difficulty curve.
- As an operator investigating "the boss won't summon", I want the skills card to show a named skill ("Summon Minions · L3") instead of a bare numeric id.
- As a platform engineer, I want the new spawn index to rebuild automatically on re-ingest rather than requiring a custom migration script.

## 4. Functional Requirements

### 4.1 Page layout (`MonsterDetailPage.tsx`)

From top to bottom, inside a scrolling container (`flex flex-col flex-1 space-y-6 p-10 pb-16 overflow-y-auto` — matches the existing page, intentionally keeps the outer container scrollable since some bosses have large drop tables):

1. **Header row** — monster icon + name, laid out with the existing `flex items-center gap-3` classes.
   - Monster name wrapped in a `Tooltip` via `@/components/ui/tooltip`. Trigger is the `<h2>` with classes `text-2xl font-bold tracking-tight cursor-help inline-block w-fit focus:outline-none focus-visible:ring-2 focus-visible:ring-ring rounded` (same treatment as `MapHeader.tsx:23-28`).
   - Monster icon (`<img>`) wrapped in the same `Tooltip` — hovering the icon also opens the tooltip.
   - `TooltipContent` uses the existing `copyable` prop (`<TooltipContent copyable><p>{monster.id}</p></TooltipContent>`) so clicking the tooltip copies the template id.
   - The current raw `#<id>` span between name and badges is **removed**. Badges (`Boss`, `Undead`, `Friendly`) remain on the same row, right of the name.
   - If `monsterIconUrl` is null/loading, render nothing in that slot (no placeholder) — same behavior as today.

2. **Stats row (compressed)** — three `Card`s in a `grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3` (tighter than today's `gap-4`). Each card:
   - `CardHeader` with `py-2 px-4` overrides to shrink vertical padding; `CardTitle` stays at `text-sm font-medium`.
   - `CardContent` uses `py-2 px-4 space-y-1 text-sm` (tighter than today's `space-y-2`).
   - Card 1 "Combat Stats": Level, HP, MP, EXP (unchanged fields; tighter spacing only).
   - Card 2 "Attack / Defense": Weapon Attack, Weapon Defense, Magic Attack, Magic Defense.
   - Card 3 "Properties": First Attack, FFA Loot, Explosive Reward, CP.
   - Number formatting unchanged — `toLocaleString()` for HP/MP/EXP, plain for others.

3. **Skills card (compressed)** — rendered only if `attrs.skills.length > 0`. Replaces the current table. Layout: `flex flex-wrap gap-2` inside `CardContent`, one chip per skill.
   - Chip component: a small rounded pill (`Badge variant="outline"` from shadcn) showing `{skill.name} · L{skill.level}` when the name is available, `{skill.id} · L{skill.level}` as a loading/fallback label.
   - Name is fetched via a new `useMobSkillData(skill.id)` hook (§4.5). While the query is pending or errored, show the numeric id.
   - Chips wrap horizontally; no fixed columns. Card title stays "Skills".
   - No click/navigation behavior — mob skills don't have detail pages.

4. **Drops card** (renamed "Drops" header stays, with count in parens) — rewritten to use per-compartment widget grids (§4.2).

5. **Spawn Locations card** (new) — grid of maps where this monster spawns, sorted by count desc (§4.3).

### 4.2 Drops card

Replaces the current flat `Table` with a grouped multi-column widget layout.

**Grouping logic** — for each `DropData` in `drops`:
- If `drop.attributes.itemId === 0` → group `"mesos"`.
- Otherwise use `getItemType(String(drop.attributes.itemId))` from `@/types/models/item` (`services/atlas-ui/src/types/models/item.ts:3-15`) — returns `"Equipment" | "Consumable" | "Setup" | "Etc" | "Cash" | "Pet" | "Unknown"`.
- `"Pet"` is never returned by `getItemType` today (cash-prefix items are classified as `"Cash"`) — group under `"Cash"` rather than introducing a Pet section, per scope decision.
- `"Unknown"` is routed to an "Other" group (last, after Cash).

**Rendering order** (top to bottom within the card, each group wrapped in a labeled subsection only when non-empty):
1. Mesos
2. Equipment
3. Consumable
4. Setup
5. Etc
6. Cash
7. Other (only if any `Unknown`)

Each subsection:
- Header: a small `<h3>` label with `text-sm font-medium text-muted-foreground uppercase tracking-wide`, followed by a count in parens (`Equipment (7)`).
- Grid: `grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2` of `MonsterDropWidget` items.
- No card-within-card; the subsections are visual groupings inside the single Drops `Card`.

**Non-meso widget (`MonsterDropWidget`)** — a new component at `components/features/monsters/MonsterDropWidget.tsx`:
- Rendered as a `Link` to `/items/{itemId}` — entire widget is clickable.
- Inside the link: `flex items-center gap-3 rounded-md border bg-card p-2 hover:bg-accent transition-colors`.
- Left: `<img>` at `width={32} height={32} loading="lazy"` sourced from `useItemData(itemId).iconUrl`. If missing, render a `lucide-react Package` fallback at the same size.
- Right: two-line block — name (`text-sm font-medium truncate`) on top, id (`text-xs font-mono text-muted-foreground`) below.
- Whole widget wrapped in a `Tooltip`; hovering surfaces:
  - Chance (formatted as the raw numerator per atlas-drop-information convention — `drop.attributes.chance.toLocaleString()` — same as current page).
  - Min Qty / Max Qty.
  - Quest ID (if `questId > 0`, else the row is omitted from the tooltip).
- Clicking the widget navigates to `/items/{itemId}` — do **not** open the tooltip's "copyable" behavior; tooltip here is purely informational (no `copyable` prop).

**Meso widget (`MonsterMesoWidget`)** — distinct from item widgets so it reads as currency, not a droppable item:
- Not a link (no `/items/0` page).
- `flex items-center gap-3 rounded-md border border-amber-300/40 bg-amber-50/50 dark:bg-amber-950/20 p-2` — the amber tint differentiates from item widgets without being loud.
- Left: a `lucide-react Coins` icon at size 20 with `text-amber-500`.
- Right: label "Mesos" (`text-sm font-medium`), and a min/max range line (`text-xs text-muted-foreground`) showing `{minimumQuantity.toLocaleString()} – {maximumQuantity.toLocaleString()}`.
- Tooltip on hover shows Chance only (qty is already on the widget).
- If there are multiple meso drop rows for a single monster (edge case), render each as its own widget.

**Empty state** — when `drops && drops.length === 0`: keep the current message `"No drops configured for this monster."`. When `drops` is `undefined` (loading), show the current `"Loading drops..."` placeholder.

### 4.3 Spawn Locations card

New card, rendered below the Drops card.

- Title: `Spawn Locations` with count in parens (`Spawn Locations (4)`).
- Hook: `useMonsterMaps(monsterId)` — calls `GET /api/data/monsters/{monsterId}/maps` (§5).
- Data shape: `Array<{ id: string; attributes: { name: string; streetName: string; spawnCount: number } }>`. UI sorts client-side by `spawnCount` desc, tie-breaker by `name` asc. Backend returns in this order too (§5); client sort is defensive.
- Grid: `grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2`.
- Each cell is a `MonsterSpawnMapWidget`:
  - Root: `Link` to `/maps/{id}` — `flex flex-col gap-1 rounded-md border bg-card p-3 hover:bg-accent transition-colors`.
  - Line 1: map name (`text-sm font-medium truncate`) + `Badge variant="secondary"` with `streetName` (only if non-empty).
  - Line 2: `Badge variant="outline"` with `"{count} spawn{count === 1 ? '' : 's'}"`.
- Empty state: `"This monster does not spawn on any loaded map."` in `text-sm text-muted-foreground`.
- Loading state: `"Loading spawn locations..."` — same treatment as Drops.
- Error state: use the existing `ErrorDisplay` component if the query errors, fitted to the card's content area.

### 4.4 Page data dependencies

The existing `MonsterDetailPage` fetches:
- `useMonster(id)` — the monster document.
- `useMonsterDrops(id)` — the drops list.
- `useMobData(parseInt(id))` — icon URL and cached monster name.

Add:
- `useMonsterMaps(id)` — new hook (§5).
- `useItemBatchData(itemIds)` — already exists in `src/lib/hooks/useItemData.ts:170-262`; pre-warms icons + names for every drop's `itemId` so widgets don't waterfall one request per drop. Called once in the page with `useMemo(() => drops?.filter(d => d.attributes.itemId !== 0).map(d => d.attributes.itemId) ?? [], [drops])`.
- `useMobSkillData(skillId)` — new hook (§4.5), called inside each skill chip via `useMobSkillData(skill.id)`. Per-chip `useQuery` is acceptable because mob skill count per monster is small (typically ≤ 5) and the shared cache deduplicates across chips/pages.

### 4.5 Mob-skill name lookup

Mob skills today are identified only by numeric `skillId` + `level`; `RestModel` in `services/atlas-data/atlas.com/data/mobskill/rest.go:8-26` has no name. Player skills, monsters, NPCs, and items all load names from `String.wz/*.img.xml` via a per-domain string registry (see `services/atlas-data/atlas.com/data/monster/string_registry.go`). This task adds the same for mob skills.

Backend changes:
- New file `services/atlas-data/atlas.com/data/mobskill/string_registry.go`:
  - `MobSkillString` value type (fields `id string`, `name string`) mirroring `MonsterString`.
  - `InitString(t tenant.Model, path string)` reads `String.wz/MobSkill.img.xml`. Each entry's name lives at `<entry name="{skillId}"><string name="name">…</string>`. Missing names default to `"MISSINGNO"` per the monster pattern.
  - `GetMobSkillStringRegistry()` returns the singleton `document.Registry[string, MobSkillString]`.
- `mobskill.Read` (`mobskill/reader.go:11-44`) is extended to accept the tenant and, after computing `skillId`, look up the name from the registry and stamp it onto every level row (one name per skillId, repeated on each level's RestModel so every row is self-describing).
- `mobskill.RestModel` gains `Name string `json:"name"``.
- `data/processor.go` worker `WorkerMobSkill` calls `mobskill.InitString(t, filepath.Join(path, "String.wz", "MobSkill.img.xml"))` before `RegisterMobSkill`, and `GetMobSkillStringRegistry().Clear(t)` after (mirrors the `WorkerMonster` pattern at `processor.go:107-118`).

Frontend changes:
- New `services/atlas-ui/src/services/api/mob-skills.service.ts` with `getMobSkillName(skillId: number): Promise<string>` — requests `GET /api/data/mob-skills/{skillId}` (existing `get_mob_skills_by_type` endpoint, see `mobskill/resource.go:22`), which returns the list of all level rows; returns `rows[0].attributes.name` (same on every row).
- New `services/atlas-ui/src/lib/hooks/useMobSkillData.ts` with `useMobSkillData(skillId: number)` following the `useSkillData` shape (tenant-scoped query, 30min stale, 24h gc, name-only — no icon).

Failure modes:
- If `MobSkill.img.xml` doesn't exist in the tenant's String.wz (older dumps), `InitString` logs a warning and returns without populating — `name` defaults to `""`. The UI falls back to the numeric id in that case. Do not hard-fail the worker.

## 5. API Surface

### 5.1 New: `GET /data/monsters/{monsterId}/maps` (atlas-data)

Returns the list of maps where a monster spawns, with spawn counts, for the active tenant.

- Route registration: in `monster.InitResource` (`services/atlas-data/atlas.com/data/monster/resource.go:19-30`), add:
  ```go
  r.HandleFunc("/{monsterId}/maps", registerGet("get_monster_maps", handleGetMonsterMapsRequest(db))).Methods(http.MethodGet)
  ```
- Handler: parses `monsterId` via `rest.ParseMonsterId`, queries the new `monster_spawn_index` table filtered by `tenant_id = ?, monster_id = ?`, orders by `spawn_count DESC, name ASC`, marshals as JSON:API.
- Response rest model `MonsterSpawnMapRestModel` (new, colocated in `monster/` as it's monster-scoped):
  ```go
  type MonsterSpawnMapRestModel struct {
      MapId       uint32 `json:"-"`
      Name        string `json:"name"`
      StreetName  string `json:"streetName"`
      SpawnCount  uint32 `json:"spawnCount"`
  }
  func (r MonsterSpawnMapRestModel) GetName() string { return "monster-spawn-maps" }
  func (r MonsterSpawnMapRestModel) GetID() string   { return strconv.Itoa(int(r.MapId)) }
  ```
- Errors:
  - `400` if `monsterId` is unparseable (existing `ParseMonsterId` behavior).
  - `500` on DB failure.
  - A monster with zero spawns returns `200` with an empty `data` array — not `404`. This keeps the UI's empty-state simple.
- Content type: `application/vnd.api+json`.

### 5.2 No change: existing endpoints

- `GET /api/data/monsters/{monsterId}` — unchanged.
- `GET /api/data/drops/monsters/{monsterId}` (the route used by `useMonsterDrops`) — unchanged.
- `GET /api/data/mob-skills/{skillId}` — payload shape gains a top-level `name` attribute (§4.5); existing consumers ignore unknown fields, so this is backwards-compatible.

## 6. Data Model

### 6.1 New table: `monster_spawn_index`

Backing entity under `services/atlas-data/atlas.com/data/monster/spawn_index.go` (new file; colocated with monster rather than map because the primary read path is monster-scoped):

```go
type SpawnIndexEntity struct {
    TenantId   uuid.UUID `gorm:"type:uuid;primaryKey"`
    MonsterId  uint32    `gorm:"primaryKey"`
    MapId      uint32    `gorm:"primaryKey"`
    Name       string    `gorm:"not null"`
    StreetName string    `gorm:"not null"`
    SpawnCount uint32    `gorm:"not null"`
    UpdatedAt  time.Time `gorm:"autoUpdateTime"`
}

func (SpawnIndexEntity) TableName() string { return "monster_spawn_index" }
```

- Composite primary key `(tenant_id, monster_id, map_id)` — a single monster can spawn on many maps; a single map holds many monsters.
- `spawn_count` is the number of spawn entries for that (monster, map) pair inside the map document.
- `name` / `street_name` are denormalized from the map for query-time read without a join against `map_search_index`. They are rewritten whenever the map is re-registered.
- Indexes: implicit on the primary key; add `CREATE INDEX IF NOT EXISTS idx_monster_spawn_index_lookup ON monster_spawn_index (tenant_id, monster_id, spawn_count DESC)` for the primary read path.

Migration: added to atlas-data's migration sequence alongside `map.Migration` — see `map/entity.go:24-31` for the existing pattern. The migration uses `gorm.AutoMigrate` followed by the explicit `CREATE INDEX IF NOT EXISTS` statement.

### 6.2 Population

`map.Storage.Add` (`services/atlas-data/atlas.com/data/map/storage.go:50-75`) is the single write hook for map registration. Extend the existing transaction to also:

1. `DELETE FROM monster_spawn_index WHERE tenant_id = ? AND map_id = ?` (to clear stale rows if the map is re-registered with different monsters).
2. Aggregate `m.Monsters` by `monster.Template` into a `map[uint32]uint32{templateId: count}`.
3. For each `(templateId, count)`: `INSERT INTO monster_spawn_index (tenant_id, monster_id, map_id, name, street_name, spawn_count, updated_at) VALUES (...)`.

All inside the same transaction as the document upsert and `searchindex.Upsert` so partial failures roll back consistently.

### 6.3 Back-fill

Not required. On first deploy after the migration ships, existing tenants re-ingest via the normal `ProcessData` flow and the index populates naturally. No separate back-fill job or migration script.

## 7. Service Impact

| Service | Changes |
|---|---|
| **atlas-ui** | Rewrite `pages/MonsterDetailPage.tsx`. Add `components/features/monsters/MonsterHeader.tsx`, `MonsterDropWidget.tsx`, `MonsterMesoWidget.tsx`, `MonsterSpawnMapWidget.tsx`, `MonsterSkillChip.tsx` (small, per-widget files; no cross-feature shared module). Add `lib/hooks/api/useMonsterMaps.ts`. Add `lib/hooks/useMobSkillData.ts`. Add `services/api/mob-skills.service.ts`. Add `types/models/monster.ts` additions for the spawn-map response type. No changes to `MonstersPage` or other pages. |
| **atlas-data** | Add `monster/spawn_index.go` (entity + migration). Add `monster/resource.go` route + handler. Add `mobskill/string_registry.go`. Extend `mobskill/reader.go` to stamp names. Add `name` field to `mobskill/rest.go` RestModel. Extend `map/storage.go` `Add` to populate the spawn index. Extend `data/processor.go` `WorkerMobSkill` branch to init/clear the mob-skill string registry. Update atlas-data migration registration to include `monster.Migration` (new) and run it. |

No other services are touched. No Kafka topics, no new configuration.

## 8. Non-Functional Requirements

**Performance:**
- `monster_spawn_index` lookup is O(rows-per-monster) via the compound index. Expected p99 < 20ms for the read path on a tenant with ~2k maps and ~500 unique monsters.
- Map registration transaction gains one DELETE + bulk INSERT per map. Map ingest is already slow (document serialization dominates); adding a few hundred row-ops per map is a small fraction (< 5% expected overhead). No change to user-facing latency.
- UI: `useItemBatchData` pre-warms drop icons/names so the Drops card doesn't N+1. Skills and spawn-maps render in parallel with drops — no sequential waterfalls.

**Multi-tenancy:**
- Every new row in `monster_spawn_index` carries `tenant_id` and is filtered by `tenant.FromContext`. The handler reuses the existing `rest.HandlerDependency.Context()` which already carries the tenant. No tenant-bypass paths.
- Mob-skill string registry is per-tenant (follows the `monster_string_registry.Add(t, …)` pattern) and cleared after the ingest worker finishes.

**Observability:**
- Ingest: log one line per map registration summarizing spawn-index rows written (e.g., `monster_spawn_index: tenant=<id> map=<id> rows=<n>`), at `Debug`. Reuses the existing map ingest logger.
- API: existing `rest.RegisterHandler` instrumentation covers the new route — no additional metrics.

**Security:**
- No new auth surface. Tenant scoping inherited from context.
- Input validation: `ParseMonsterId` handles malformed ids. `ParseMobSkillId` already exists.

**Backwards compatibility:**
- Adding a `name` field to `mobskill.RestModel` is additive — existing consumers unaffected.
- The spawn-index table is new; migration is forward-only.
- UI's existing `/monsters/:id` route still mounts `MonsterDetailPage`, just rewritten.

## 9. Open Questions

_None remaining._ The three previously-outstanding items are resolved as follows:

1. **Tooltip reuse on header** — **Resolved: single shared trigger.** Wrap icon + name in one `<span tabIndex={0}>` used as the `TooltipTrigger asChild`. Rationale: shadcn's `TooltipTrigger` takes a single child, and since neither the icon nor the name is independently interactive, collapsing them into one tabstop is simpler and matches the stated intent ("hovering either opens the same tooltip"). Avoids rendering two duplicate Tooltip instances with identical content.

2. **`MobSkill.img.xml` availability** — **Resolved: assume present, degrade gracefully.** `String.wz/MobSkill.img` is standard in GMS-lineage WZ dumps (v40+). atlas-wz-extractor unpacks `String.wz` as a whole rather than enumerating individual `.img` files (see `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/upload_test.go:86-97`), so whatever the upstream producer emitted will be on disk. The graceful-degradation path (missing file → empty `name` → numeric id chip) already covers the unlikely case it's absent. No pre-flight confirmation needed before implementation.

3. **`map.Storage.Add` as sole write path** — **Resolved: confirmed.** Code-verified via grep of `services/atlas-data/atlas.com/data/map/`: `Storage.Add` is called only from `map/processor.go:31` (the ingest `Register` function) and from `storage.go:55` (the transactional wrapper calling its own scoped instance). `map.InitResource` (`map/resource.go:24-44`) exposes only GET routes — no PATCH/PUT/DELETE. The DELETE-then-INSERT strategy in §6.2 is safe.

## 10. Acceptance Criteria

Backend (atlas-data):
- [ ] `monster_spawn_index` migration runs on startup; table exists with the indexes in §6.1.
- [ ] Map re-ingest populates `monster_spawn_index` rows for every spawned monster, with accurate `spawn_count`. Verified by a unit test covering `map.Storage.Add` that seeds a map with two copies of monster A and one of monster B, then asserts two rows written.
- [ ] `GET /data/monsters/{monsterId}/maps` returns rows sorted by `spawn_count DESC, name ASC`, tenant-scoped. 200 with empty array when no spawns exist. Unit test coverage in `monster/resource_test.go`.
- [ ] `mobskill.InitString` populates names for all ids in `String.wz/MobSkill.img.xml`. Unit test using a fixture XML file.
- [ ] `mobskill.RestModel.Name` is populated on every level row after `Read`. Unit test in `mobskill/reader_test.go`.
- [ ] Missing `MobSkill.img.xml` does not break the `WorkerMobSkill` run — logged as a warning, worker completes.
- [ ] Docker build for atlas-data succeeds.

Frontend (atlas-ui):
- [ ] `MonsterDetailPage` header hover surfaces a copyable tooltip showing the template id. Hovering either the icon or the name opens the same tooltip.
- [ ] Stat cards are visually tighter (vertical padding reduced from `space-y-2` to `space-y-1`, card body uses `py-2`) while preserving the same fields.
- [ ] Skills card renders one chip per skill, showing the skill name + level when available, numeric id as a fallback. Chip wrap is `flex flex-wrap gap-2`.
- [ ] Drops card groups drops into Mesos / Equipment / Consumable / Setup / Etc / Cash / Other sections — only non-empty sections render, in that fixed order.
- [ ] Each non-meso drop widget shows icon + name + id, links to `/items/{itemId}`, and surfaces chance / min / max / questId in a hover tooltip.
- [ ] Meso widgets render distinctly (amber tint + Coins icon + range line) and are not links.
- [ ] Spawn Locations card renders a grid of maps sorted by spawn count desc; each cell links to `/maps/{id}` and shows name + street-name badge + count badge.
- [ ] Empty and loading states are handled for drops and spawns.
- [ ] Scrolling works — the page overflows vertically for drop-heavy monsters and the outer container scrolls (no trapped overflow).
- [ ] `npm run build` and `npm run test` pass. No new ESLint errors.
- [ ] Tenant switching invalidates the new `useMonsterMaps` and `useMobSkillData` caches (automatic via `queryClient.clear()` in `TenantProvider` — verify with a focused manual test).

Cross-cutting:
- [ ] Docker compose up, re-ingest one tenant's data, then load `/monsters/{id}` for a monster with known drops and spawns — all three new features (header tooltip, grouped drops, spawn locations) render correctly.
- [ ] No regressions on `MapDetailPage` or `ItemDetailPage` (spot-check).
