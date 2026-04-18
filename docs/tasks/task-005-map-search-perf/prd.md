# Map Search Performance — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-04-18
---

## 1. Overview

The `MapsPage` in atlas-ui searches maps by ID, name, or street name, backed by `GET /api/data/maps?search=<q>` in atlas-data. Today that endpoint loads every `MAP` row from the `documents` table, JSON-unmarshals each full map document (portals, foothold tree, monsters, NPCs, background types, rectangles, etc.), and then runs a case-insensitive `strings.Contains` over `name` / `streetName` in Go before truncating to 50 rows (`services/atlas-data/atlas.com/data/map/resource.go:44-84`). With ~30k maps per tenant, that scan is slow enough that the UI currently gates the query behind a "Search" button (`services/atlas-ui/src/pages/MapsPage.tsx:28-48`) rather than firing as the user types.

This task makes map search fast enough to support debounced as-you-type search by filtering server-side against a derived, narrowly-scoped index — not the full JSONB document. atlas-ui is updated to consume the new endpoint, drop the Search button, and use `keepPreviousData` so the table doesn't flash between keystrokes.

The same bottleneck applies to the other wz-data list pages (NPCs, monsters, items, reactors). This task intentionally targets **maps only** so we can validate the pattern end-to-end before replicating it. A follow-up task ("unify wz-data search UX") will apply the same approach to the remaining resources.

## 2. Goals

Primary goals:
- Make map search latency low enough to feel instant as the user types: **p95 < 100ms** server-side for a 2–3 character query over a ~30k-map tenant.
- Stop scanning full JSONB map documents at search time.
- Remove the "Search" button from `MapsPage` and replace it with debounced type-to-search (min 2 chars, ~250ms debounce).
- Preserve existing `GET /api/data/maps/{mapId}` behavior exactly — single-map lookups still return the full model.

Non-goals:
- Applying the same pattern to NPCs, monsters, items, or reactors. Those are tracked as a separate follow-up task.
- Fuzzy search, ranked/similarity ordering, typo tolerance, accent folding, or CJK normalization.
- Changing how maps are ingested from `atlas-wz-extractor` or altering the `MAP` document schema.
- Removing or restructuring `documents` itself.
- Merchants search or login-history search — different services, different backends.

## 3. User Stories

- As an operator searching maps in atlas-ui, I want matches to appear as I type so I can find a map without deliberately pressing a button and waiting.
- As an operator, I want to search by partial name ("nesys" → Henesys) and by street name, not just by prefix or exact ID.
- As an operator, I want search results to feel stable — the list shouldn't flash empty between keystrokes.
- As a developer, I want a single, well-indexed query path for map search so I don't have to reason about per-call unmarshal cost.

## 4. Functional Requirements

### 4.1 Server-side map search

- atlas-data exposes a search capability that filters by `id`, `name`, and `streetName` without unmarshaling the full `MAP` document.
- Matching semantics:
  - If the query parses as an integer, return the map whose `id` equals that integer (if any), merged ahead of name/street matches.
  - Otherwise, case-insensitive substring match against `name` and `streetName` (equivalent to `ILIKE '%q%'` on `LOWER(name)` / `LOWER(street_name)`).
- Results are limited to 50 rows per request. The limit is enforced server-side.
- Response is sparse: each returned `maps` resource includes only `id`, `name`, and `streetName`. It does **not** include portals, NPCs, monsters, reactors, footholds, or background types.
- Tenant resolution mirrors the existing `document.Storage` fallback: query the active tenant first; if it returns fewer than `limit` rows, fill the remainder with `uuid.Nil` ("global") rows, deduplicated by `map_id` with the tenant-scoped row winning.
- Results are returned in a stable order: exact ID match first (if any), then by `name` ASC, then `map_id` ASC.

### 4.2 Index maintenance

- When a map is ingested (`RegisterMap` → `document.Storage.Add`), a corresponding row in `map_search_index` is upserted inside the same database transaction. If the index upsert fails, the `documents` insert must roll back — the two must not drift.
- When a map document is deleted or replaced (today this happens via bulk reload paths; see `document.DbStorage.Clear` and `DeleteAll`), the matching index rows are deleted in the same transaction. Where existing code only truncates `documents` for a given type, the cleanup extends to `map_search_index`.
- A one-shot backfill is provided to populate `map_search_index` from existing `documents` rows of type `MAP`. The backfill is idempotent.

### 4.3 Existing endpoints

- `GET /api/data/maps/{mapId}` is unchanged — it continues to read the full document and return the complete model.
- `GET /api/data/maps` without a `search` query parameter retains today's behavior (return all maps). Its response shape does not change in this task. Callers that rely on it are out of scope.
- `GET /api/data/maps?search=<q>` routes to the new fast path and returns the sparse projection described in 4.1. This is the behavior change observed by atlas-ui.

### 4.4 atlas-ui changes

- `services/atlas-ui/src/pages/MapsPage.tsx`:
  - Remove the explicit "Search" button and the `Enter`-to-submit handler.
  - Fire the query on every input change, debounced (~250ms) and gated on `searchInput.trim().length >= 2`.
  - React Query uses `keepPreviousData: true` (or the v5 equivalent `placeholderData: keepPreviousData`) so the table retains the prior result set between keystrokes.
  - The URL `?q=` param is still written on debounce-settle so a page refresh restores the last query.
  - "Clear" button still resets the input and URL.
- `services/atlas-ui/src/services/api/maps.service.ts`: `searchMaps` and `searchMapsByName` target the fast-path endpoint. The `fields[maps]` sparse-fields hint can stay but becomes redundant — the server response is already sparse.
- `services/atlas-ui/src/lib/hooks/api/useMaps.ts`: any `useMapsSearch`-style hook is updated to reflect the debounced/keep-previous-data pattern.

## 5. API Surface

### 5.1 `GET /api/data/maps?search=<q>&limit=<n>`

- **Method**: `GET`
- **Path**: `/api/data/maps`
- **Query parameters**:
  - `search` (string, required to trigger fast path; omit for legacy full-list behavior)
  - `limit` (integer, optional; default 50, max 50)
- **Headers**: standard tenant headers (`TENANT_ID`, `REGION`, `MAJOR_VERSION`, `MINOR_VERSION`) — unchanged from today.
- **Response 200** (`application/vnd.api+json`):
  ```json
  {
    "data": [
      { "id": "100000000", "type": "maps", "attributes": { "name": "Henesys", "streetName": "Victoria Road" } }
    ]
  }
  ```
  Only `name` and `streetName` appear on search results. Portals/NPCs/monsters/reactors/footholds/backgroundTypes are **not** included.
- **Response 400**: `search` present but empty after trimming, or `limit` is not a positive integer ≤ 50.
- **Response 500**: database error. Message is logged; response body is the existing error shape.
- **Ordering**: exact-ID match first (when `search` is numeric and matches an indexed row), then `name` ASC, then `map_id` ASC.

### 5.2 `GET /api/data/maps/{mapId}` (unchanged)

Full model, including portals, NPCs, monsters, reactors, footholds, background types, foothold tree, and map area.

## 6. Data Model

### 6.1 New table: `map_search_index`

| Column        | Type         | Notes                                                  |
|---------------|--------------|--------------------------------------------------------|
| `tenant_id`   | `uuid`       | Part of primary key. `uuid.Nil` for global defaults.   |
| `map_id`      | `integer`    | Part of primary key. Matches `documents.document_id`.  |
| `name`        | `text`       | From the map document's `name` attribute.              |
| `street_name` | `text`       | From the map document's `streetName` attribute.        |
| `updated_at`  | `timestamptz`| `autoUpdateTime`.                                      |

**Primary key**: `(tenant_id, map_id)`.

**Indexes**:
- `idx_map_search_index_name_trgm` — `USING GIN (LOWER(name) gin_trgm_ops)` for case-insensitive substring match.
- `idx_map_search_index_street_trgm` — `USING GIN (LOWER(street_name) gin_trgm_ops)`.
- `idx_map_search_index_tenant_map_id` — covered by the primary key; used for exact-ID lookup.

**Extension**: `pg_trgm` (enabled in the schema migration if not already present).

See `data-model.md` for rationale and rejected alternatives (JSONB expression index on `documents.content`).

## 7. Service Impact

| Service      | Changes                                                                                                                                                     |
|--------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------|
| atlas-data   | New `map_search_index` table + migration + `pg_trgm` extension check. New search query (`map.SearchByQuery`) and handler on the existing `/data/maps` route. `RegisterMap` / `document.Storage.Add` extended to upsert the index row in the same transaction. `DbStorage.Clear` / `DeleteAll` extended to clean up the index. Backfill command/endpoint. |
| atlas-ui     | `MapsPage.tsx` drops the Search button, adds debounced type-to-search with `keepPreviousData`. Service and hook tweaks as described in 4.4.                 |
| atlas-tenants| No changes.                                                                                                                                                 |
| Other        | No changes.                                                                                                                                                 |

## 8. Non-Functional Requirements

### Performance
- p95 server-side latency < 100ms for a 2–3 character query over a ~30k-map tenant.
- p95 server-side latency < 50ms when the query is an exact numeric ID match.
- Ingest of a single map must not regress by more than ~5% after adding the index upsert; acceptable because ingest is a one-shot offline operation.

### Observability
- The search handler logs query length, result count, elapsed duration, and tenant id at debug level. Errors include the underlying Postgres error.
- Existing request-level tracing (OpenTelemetry) wraps the new DB query.

### Security
- Query is parameterized — no string concatenation into SQL. `ILIKE` argument uses `%q%` binding.
- Input is length-bounded (reasonable cap, e.g., 128 chars) before reaching the DB.

### Multi-tenancy
- Index is keyed by `(tenant_id, map_id)`. Tenant fallback to `uuid.Nil` mirrors `document.Storage.ByIdProvider` behavior (active tenant first, global fallback).
- Tenant headers are read from context via `tenant.MustFromContext(ctx)` — same helper as existing code.

### Compatibility
- No breaking change to `GET /api/data/maps` (without `search`) or `GET /api/data/maps/{mapId}`.
- JSON:API resource type remains `maps`. The sparse projection is a subset of existing attributes — existing clients that request more fields will simply see less data when `search` is set. The UI already requests only `name` / `streetName`.

## 9. Open Questions

- Should the un-searched `GET /api/data/maps` (no `search`) also switch to the sparse projection? Today the UI calls it via `fields[maps]=name,streetName`, but the server still unmarshals full documents. Out of scope for this task unless it turns out to be cheap to include. **Default assumption: leave it alone, revisit in the follow-up task.**
- Do we want a separate `GET /api/data/maps/search` path instead of overloading `?search=` on the collection route? **Default assumption: overload, matching the pattern already established by monsters (commit `6b4f81e6e`).**
- Should the backfill run automatically on service start, or only via an operator-triggered command? **Default assumption: operator-triggered to keep startup lean; document the command clearly.**

## 10. Acceptance Criteria

- [ ] `map_search_index` table exists with the columns, primary key, and GIN trigram indexes defined in §6. `pg_trgm` is enabled.
- [ ] Ingesting a new map via `RegisterMap` writes a matching `map_search_index` row in the same transaction. Failure of the index write rolls back the `documents` write.
- [ ] Bulk clear paths (`document.DbStorage.Clear`, `DeleteAll`) remove the corresponding `map_search_index` rows.
- [ ] Backfill command populates `map_search_index` from existing `documents` WHERE `type = 'MAP'` and is idempotent (safe to re-run).
- [ ] `GET /api/data/maps?search=henesys` returns ≤ 50 sparse `maps` resources, matching `name` or `streetName` case-insensitively, without unmarshaling any full `MAP` document.
- [ ] `GET /api/data/maps?search=100000000` returns the exact-ID match first when one exists.
- [ ] `GET /api/data/maps/{mapId}` behavior is byte-identical to before the change (verified by existing `rest_test.go` / manual request).
- [ ] p95 latency of `/api/data/maps?search=<q>` is < 100ms on a ~30k-map tenant (measured locally or in the staging env).
- [ ] `MapsPage` in atlas-ui has no Search button; the table updates as the user types with a ~250ms debounce and keeps previous results between keystrokes.
- [ ] `?q=` URL param is written on debounce-settle and restores the search on page reload.
- [ ] Tenant fallback: a tenant-scoped match wins over a global match with the same `map_id`.
- [ ] Unit tests cover: exact-ID match, substring match on name, substring match on street name, limit enforcement, tenant fallback, and empty-query rejection.
- [ ] Existing atlas-data and atlas-ui tests continue to pass.
