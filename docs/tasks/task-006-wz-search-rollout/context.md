# Wz-Data Search Rollout — Context

Last Updated: 2026-04-18

## Key Files

### atlas-data (backend)

**Reference implementation (task-005, maps):**
- `services/atlas-data/atlas.com/data/map/storage.go` — transactional upsert into `map_search_index`
- `services/atlas-data/atlas.com/data/map/search.go` — sparse `?search=` handler with tenant fallback + exact-id union
- `services/atlas-data/atlas.com/data/map/entity.go` — GORM entity for `map_search_index`
- `services/atlas-data/atlas.com/data/map/storage_test.go`, `search_test.go` — must keep passing after retrofit
- `services/atlas-data/atlas.com/data/map/backfill.go`, `backfill_test.go` — **to be removed**

**To be refactored / extended:**
- `services/atlas-data/atlas.com/data/npc/resource.go:67` — `filterNpcs` slow path (delete)
- `services/atlas-data/atlas.com/data/monster/resource.go:52` — `filterMonsters` slow path (delete)
- `services/atlas-data/atlas.com/data/reactor/resource.go:52` — `filterReactors` slow path (delete)
- `services/atlas-data/atlas.com/data/item/string_resource.go:51` — `filterItemStrings` slow path (delete)
- `services/atlas-data/atlas.com/data/npc/registry.go` — ingest-time upsert hook site
- `services/atlas-data/atlas.com/data/monster/registry.go` — ingest-time upsert hook site
- `services/atlas-data/atlas.com/data/reactor/registry.go` — ingest-time upsert hook site
- `services/atlas-data/atlas.com/data/item/string_registry.go` — ingest-time upsert hook site

**New:**
- `services/atlas-data/atlas.com/data/searchindex/` — shared helper package (exact files TBD in Phase A1)

### atlas-ui (frontend)

**Reference implementation (task-005, MapsPage):**
- `services/atlas-ui/src/pages/MapsPage.tsx` — debounced `search`, `keepPreviousData`, URL-backed `?q=`

**To migrate:**
- `services/atlas-ui/src/pages/NpcsPage.tsx` — additionally handles `storebank` toggle
- `services/atlas-ui/src/pages/MonstersPage.tsx`
- `services/atlas-ui/src/pages/ReactorsPage.tsx`
- `services/atlas-ui/src/pages/ItemsPage.tsx`

**Service / hook layers (paths unchanged, pattern matched to `useMaps`):**
- `services/atlas-ui/src/…/npcs.service.ts`, `useNpcs`
- `services/atlas-ui/src/…/monsters.service.ts`, `useMonsters`
- `services/atlas-ui/src/…/reactors.service.ts`, `useReactors`
- `services/atlas-ui/src/…/items.service.ts`, `useItems`

## Key Decisions

1. **No backfill.** Re-ingestion is the populate path. `map/backfill.go` goes away in
   this task. Documented in migration-plan §2–§3.
2. **Five resources, one helper.** Maps is retrofit through the new `searchindex`
   helper. If any resource requires bypassing the helper, flag during implementation
   review (data-model §"Shared searchindex helper").
3. **Helper API shape (generics vs. thin helpers) — implementer's call.** PRD §9 open
   question. Preference: generics when readable; fall back to thin helpers if awkward.
4. **NPC storebank composes via a partial index** (`WHERE storebank = true`). Keeps
   toggle-only and toggle+search paths both cheap (data-model §3 NPC schema).
5. **Sparse response only under `?search=` or NPC `filter[storebank]`.** No-query
   `GET /api/data/<type>` retains legacy full-list behavior unchanged.
6. **UI pattern: `MapsPage` is the template.** 250ms debounce, min 2 chars,
   `placeholderData: keepPreviousData`, `?q=` URL param on debounce-settle.
7. **Item categories other than `ITEM_STRING` are out of scope.** UI only searches via
   `/api/data/item-strings`.
8. **Quests are out of scope.** `QuestsPage` uses client-side category filter, not
   `?search=`.

## Dependencies Between Phases

- **A → B, C, D, E, F:** Helper must exist before any resource consumes it.
- **B precedes C, D, E, F recommended:** Retrofitting maps proves the helper on known-
  good behavior before applying it to four new consumers.
- **C, D, E, F are parallelizable** across distinct resource directories.
- **G (UI) depends on C, D, E, F** building and shipping cleanly — UI contracts on
  sparse response shape.
- **H (docs, perf, full builds) runs last.**

## External References

- task-005 PRD (if archived) for the maps precedent.
- `services/atlas-data/atlas.com/data/map/` as the live reference for
  migration/query/upsert/cleanup patterns.
- `pg_trgm` extension — enabled by task-005's migration; all new migrations tolerate
  either state.

## Gotchas

- GORM `AutoMigrate` doesn't model expression indexes; use raw `Exec` for both the
  trigram GIN index and the NPC storebank partial index (matches task-005's approach).
- Tests reference internal functions (per Atlas memory) — renaming handlers breaks
  tests. Prefer extending in place during the retrofit.
- `tenant.MustFromContext(ctx)` is the canonical tenant accessor; keep it consistent
  across handlers.
- Docker builds must be verified when touching shared libraries (CLAUDE.md
  Build & Verification section).
- Per-resource registries are upstream of `document.Storage.Add`; the upsert hook must
  land inside the same `ExecuteTransaction` as the `documents` write.
- `tenant.Model.Region()` returns `string`, not `world.Id` — unrelated but frequently
  confused.
