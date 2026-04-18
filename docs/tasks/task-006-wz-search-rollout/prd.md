# Wz-Data Search Rollout — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-04-18
---

## 1. Overview

Task-005 shipped a derived trigram-indexed `map_search_index` table, a sparse `GET /api/data/maps?search=` fast path, and a debounced type-to-search `MapsPage`. The remaining wz-data list pages in atlas-ui — NPCs, monsters, reactors, and items (item-strings) — still suffer the same bottleneck: `GetAll` loads every `documents` row of the matching type, JSON-unmarshals each full document, and then runs `strings.Contains` in Go before truncating to 50 rows (`services/atlas-data/atlas.com/data/npc/resource.go:67`, `monster/resource.go:52`, `reactor/resource.go:52`, `item/string_resource.go:51`). The UI compensates with explicit "Search" buttons and `Enter`-to-submit handlers that `MapsPage` no longer has.

This task rolls the task-005 pattern out to those four resources. Each gets its own per-tenant search-index table populated during ingest, a fast-path `?search=` handler that returns sparse results, and a debounced search UI with `keepPreviousData`. A small `searchindex` helper package consolidates the shared plumbing (migration helper, query shape, ingest hook) so maps, NPCs, monsters, reactors, and item-strings use one codepath instead of five copy-pasted ones. The existing `map_search_index` is retrofit onto the helper as part of this task.

Backfill is intentionally removed from the pattern. Task-005's `map/backfill.go` was a one-shot catch-up for already-ingested tenants; here we instead document re-ingestion as the canonical way to populate the new indexes, because operators already re-run ingest whenever wz data changes and it exercises the same codepath the migration will depend on going forward.

## 2. Goals

Primary goals:
- Make NPC, monster, reactor, and item-string searches fast enough for debounced as-you-type UX: **p95 < 100ms** server-side for a 2–3 character query.
- Stop scanning full JSONB documents at search time for any of those four resources.
- Remove "Search" buttons from `NpcsPage`, `MonstersPage`, `ReactorsPage`, and `ItemsPage`. Replace with debounced type-to-search (min 2 chars, ~250ms debounce, `keepPreviousData`).
- Factor shared search-index plumbing into a `searchindex` package used by all five resources (maps retrofit + four new).
- Preserve existing `GET /api/data/<type>/{id}` endpoints byte-identically.
- Preserve the composed `filter[storebank]=true` + `?search=` use case for NPCs.

Non-goals:
- Quests. `QuestsPage` has no `?search=` endpoint today; its UX is a client-side category filter over the preloaded list. Address separately if ever needed.
- Item categories other than `ITEM_STRING` (equipment/consumable/setup/etc/cash). The UI only searches via `/api/data/item-strings`, so those category-specific endpoints keep their current behavior.
- Accounts, characters, guilds, bans, merchants, login history, gachapons, services. Different services, different backends.
- Fuzzy search, ranked/similarity ordering, typo tolerance, CJK normalization.
- Changing wz-data extraction from `atlas-wz-extractor`, the `documents` schema, or non-search REST endpoints.
- A one-shot backfill utility. Re-ingest is the supported path.

## 3. User Stories

- As an operator searching NPCs/monsters/reactors/items in atlas-ui, I want matches to appear as I type so I can find what I need without deliberately pressing a button and waiting.
- As an operator filtering NPCs, I want `storebank` filtering and text search to compose — I can type "henesys" and see only storebank NPCs whose name matches.
- As an operator, I want partial name matches and integer ID matches to work, consistent with today.
- As an operator, I want results to feel stable — the table shouldn't flash empty between keystrokes.
- As a developer adding a sixth searchable resource, I want to reach for the `searchindex` helper instead of copy-pasting 300 lines from maps.

## 4. Functional Requirements

### 4.1 Shared `searchindex` helper package

- Location: `services/atlas-data/atlas.com/data/searchindex/` (exact path at implementer's discretion).
- Exposes at minimum:
  - A migration helper that creates a table with columns `tenant_id uuid`, `<entity>_id <int-type>`, `name text`, caller-supplied extension columns (e.g., `street_name text` for maps, `storebank bool` for NPCs), and `updated_at timestamptz`, plus a trigram GIN index on `LOWER(name)` and optional extra-column indexes.
  - A search query helper that, given a tenant, query string, and limit, returns `(entity_id, name, <extras>)` ordered by exact-id match first, then `name` ASC, then `<entity>_id` ASC, with tenant-scoped rows preferred over `uuid.Nil` global rows.
  - A transaction hook each resource's ingest path calls to upsert a search-index row in the same `*gorm.DB` transaction as the `documents` write, and to delete rows during `DbStorage.Clear` / `DeleteAll`.
- `pg_trgm` is already enabled by task-005's migration. The helper does not re-enable it but tolerates either state.
- Resources keep their own typed entity definitions and resource-specific search handlers. The helper is about query/migration/transaction plumbing, not about hiding resource types.

### 4.2 Per-resource search indexes

Four new tables, each following the shared template from §4.1. Full schemas live in `data-model.md`.

| Table                        | Entity id column       | Extra columns     | Ingest hook site                                 |
|------------------------------|------------------------|-------------------|--------------------------------------------------|
| `npc_search_index`           | `npc_id integer`       | `storebank bool`  | `npc.Register` (via `document.Storage.Add`)      |
| `monster_search_index`       | `monster_id integer`   | —                 | `monster.Register`                               |
| `reactor_search_index`       | `reactor_id integer`   | —                 | `reactor.Register`                               |
| `item_string_search_index`   | `item_id integer`      | —                 | item-string registration path in `item/string_registry.go` |

Each has `(tenant_id, <entity>_id)` as primary key and a `GIN (LOWER(name) gin_trgm_ops)` index.

### 4.3 Search semantics

For every resource:
- If the query parses as an integer, return the row whose `<entity>_id` equals it (if any) merged ahead of name matches.
- Otherwise, case-insensitive substring match against `name` — equivalent to `ILIKE '%q%'` on `LOWER(name)`.
- Results are limited to 50 per request. Limit is enforced server-side.
- Response is sparse: `{id, name}` only (plus `storebank` on NPCs). No nested data from the full document.
- Tenant fallback: active tenant first, fill remainder from `uuid.Nil` global rows, deduplicated by `<entity>_id` with tenant-scoped rows winning.
- Ordering: exact-ID match first, then `name` ASC, then `<entity>_id` ASC.

### 4.4 NPC `filter[storebank]` composition

- `npc_search_index` includes a `storebank bool NOT NULL` column populated at ingest from the NPC document.
- `GET /api/data/npcs?search=<q>&filter[storebank]=true` and `?filter[storebank]=true` alone both route to the fast path.
  - `filter[storebank]=true` alone: `WHERE tenant_id = $1 AND storebank = true`, ordered as in §4.3.
  - `?search=<q>&filter[storebank]=true`: both predicates AND together.
- `?filter[storebank]=true` without `?search=` returns up to 50 rows (matches the existing server-side limit on the slow path).
- Without any query parameters, `GET /api/data/npcs` retains legacy full-list behavior unchanged.

### 4.5 Ingest and cleanup hooks

- When a document is registered (`npc.Register`, `monster.Register`, `reactor.Register`, item-string registration), a matching search-index row is upserted inside the same transaction. If the index upsert fails, the `documents` insert rolls back.
- `document.DbStorage.Clear` / `DeleteAll` for each of the five types (maps + four new) also delete from the matching `<type>_search_index` table inside the same transaction.
- **No backfill.** Re-ingestion is the supported path to populate indexes after the migration runs. This must be documented in the atlas-data service README and release notes.
- `map/backfill.go` and `map/backfill_test.go` introduced by task-005 are removed as part of this task.

### 4.6 Maps retrofit

- `services/atlas-data/atlas.com/data/map/storage.go`, `search.go`, and `entity.go` are rewritten to call through the `searchindex` helper.
- The existing `map_search_index` table keeps its name. Its extra column `street_name text` is declared through the helper's extension-column mechanism.
- Behavior is byte-identical before and after the retrofit. Existing `map/storage_test.go` and `map/search_test.go` continue to pass.
- `map/backfill.go` and `map/backfill_test.go` are removed (see §4.5).

### 4.7 atlas-ui changes

For each of `NpcsPage`, `MonstersPage`, `ReactorsPage`, `ItemsPage`:

- Remove the "Search" button, the `Enter`-to-submit handler, and the `handleSearch`/`handleKeyDown` helpers. Keep "Clear".
- Fire the search query on every input change, debounced (~250ms) and gated on `searchInput.trim().length >= 2`.
- Use `placeholderData: keepPreviousData` (React Query v5 idiom) so the table retains prior results between keystrokes.
- The URL `?q=` param is written on debounce-settle so a page refresh restores the last query.
- `NpcsPage` additionally: the storebank toggle (where present) continues to work and composes with debounced search — the React Query `queryFn` passes both `search` and `filter[storebank]` when the toggle is on. The toggle round-trips through the URL alongside `?q=`.
- Service layers (`npcs.service.ts`, `monsters.service.ts`, `reactors.service.ts`, `items.service.ts`) target the existing endpoint paths; no URL changes.
- Hook layers (`useNpcs`, `useMonsters`, etc. where present) are updated to match the debounced-with-kept-previous-data pattern established in `useMaps`.

## 5. API Surface

All four endpoints continue to live at their current paths. Response shapes on the `?search=` / `filter[storebank]` code path become sparse (as defined in §4.3). Response shapes without any query parameters are unchanged.

### 5.1 `GET /api/data/npcs?search=<q>&filter[storebank]=<bool>&limit=<n>`
- Sparse resource attributes: `{ name, storebank }`.
- Composition rules in §4.4.

### 5.2 `GET /api/data/monsters?search=<q>&limit=<n>`
- Sparse resource attributes: `{ name }`.

### 5.3 `GET /api/data/reactors?search=<q>&limit=<n>`
- Sparse resource attributes: `{ name }`.

### 5.4 `GET /api/data/item-strings?search=<q>&limit=<n>`
- Sparse resource attributes: `{ name }`.

### 5.5 Common
- `limit` default 50, max 50.
- Headers: `TENANT_ID`, `REGION`, `MAJOR_VERSION`, `MINOR_VERSION` — unchanged.
- `400` when `search` is present but empty after trim, or when `limit` is not a positive integer ≤ 50.
- `500` on database error.
- JSON:API resource types (`npcs`, `monsters`, `reactors`, `item-strings`) unchanged.

### 5.6 Unchanged endpoints
- `GET /api/data/<type>/{id}` for each of the four resources continues to return the full model. Byte-identical to today.
- `GET /api/data/<type>` with no query parameters continues to return all rows on the legacy slow path.

## 6. Data Model

Four new per-tenant search-index tables plus the maps retrofit. See `data-model.md` for full schemas, query shapes, and rationale.

## 7. Service Impact

| Service      | Changes |
|--------------|---------|
| atlas-data   | New `searchindex` package. Four new tables + migrations. Four new search handlers and ingest/cleanup hooks. Four existing in-memory filter blocks removed (`npc.filterNpcs`, `monster.filterMonsters`, `reactor.filterReactors`, `item.filterItemStrings`). Maps retrofit through the shared helper. `map/backfill.go` removed. |
| atlas-ui     | Four pages (`NpcsPage`, `MonstersPage`, `ReactorsPage`, `ItemsPage`) switched to debounced type-to-search with `keepPreviousData`. Corresponding service/hook tweaks. |
| atlas-tenants, atlas-ingress, others | No changes. |

## 8. Non-Functional Requirements

### Performance
- p95 server-side latency < 100ms for a 2–3 character query on each of the four resources, over a tenant with today's typical document counts.
- p95 < 50ms for exact numeric ID queries.
- Ingest of a single document must not regress by more than ~5% after adding the index upsert; acceptable because ingest is a one-shot offline operation.

### Observability
- Each search handler logs query length, result count, elapsed duration, tenant id, and resource type at debug level.
- Existing request-level OpenTelemetry spans wrap the new DB queries.

### Security
- Queries are parameterized. `ILIKE` argument uses `%q%` binding.
- Input is length-bounded (≤128 chars) before reaching the DB.

### Multi-tenancy
- `(tenant_id, <entity>_id)` primary key. Tenant fallback to `uuid.Nil` mirrors `document.Storage.ByIdProvider`.
- Tenant headers read via `tenant.MustFromContext(ctx)` — same helper as existing code.

### Compatibility
- No breaking change to `GET /api/data/<type>/{id}` or to the no-query `GET /api/data/<type>` legacy path.
- Sparse projection under `?search=` is a subset of today's attributes; UI already requests only `name` via sparse fields hints.

## 9. Open Questions

- Should the legacy no-query `GET /api/data/<type>` path eventually also move onto the sparse projection (and the index table)? Same question task-005 left open. **Default assumption: leave alone — out of scope here, address when a consumer asks for it.**
- `searchindex` package shape — full Go generics vs. thin helpers — is left to implementation judgement. The PRD requires consolidation but doesn't prescribe the exact Go API. **Default assumption: prefer generics when they stay readable; fall back to a thin helper + per-resource query fns if the generic API becomes awkward.**

## 10. Acceptance Criteria

- [ ] `searchindex` package exists and is used by all five resources (maps + four new).
- [ ] `npc_search_index`, `monster_search_index`, `reactor_search_index`, `item_string_search_index` tables exist with the columns, primary keys, and trigram indexes defined in §6.
- [ ] Ingesting a new document via the respective `Register` path writes a matching search-index row in the same transaction. Failure of the index write rolls back the `documents` write.
- [ ] `document.DbStorage.Clear` / `DeleteAll` for each type remove the corresponding search-index rows in the same transaction.
- [ ] `GET /api/data/npcs?search=<q>` returns ≤50 sparse `npcs` resources matching `name` case-insensitively without unmarshaling full `NPC` documents.
- [ ] `GET /api/data/npcs?filter[storebank]=true` and `GET /api/data/npcs?search=<q>&filter[storebank]=true` both route to the fast path and return only storebank NPCs.
- [ ] `GET /api/data/monsters?search=`, `/api/data/reactors?search=`, `/api/data/item-strings?search=` each meet the semantics in §4.3.
- [ ] `GET /api/data/<type>/{id}` byte-identical to before the change for each of the four resources (verified by existing `rest_test.go` / manual request).
- [ ] `NpcsPage`, `MonstersPage`, `ReactorsPage`, `ItemsPage` have no Search button; update as user types with ~250ms debounce and keep previous results between keystrokes.
- [ ] `?q=` URL param is written on debounce-settle for each page.
- [ ] `NpcsPage` storebank toggle composes with debounced search and round-trips through the URL.
- [ ] Maps retrofit leaves `map_search_index` and existing behavior byte-identical; `map/backfill.go` is removed.
- [ ] p95 latency of each `?search=` endpoint is < 100ms over today's typical document counts (measured locally or in staging).
- [ ] Tenant fallback works: tenant-scoped match wins over global for the same id.
- [ ] Unit tests cover: exact-id match, substring name match, limit enforcement, tenant fallback, empty-query rejection, and for NPCs the `filter[storebank]` composition with and without `?search=`.
- [ ] All atlas-data and atlas-ui existing tests continue to pass.
