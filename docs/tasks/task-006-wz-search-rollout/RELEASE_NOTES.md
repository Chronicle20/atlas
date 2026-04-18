# Task-006 — Wz-Data Search Rollout Release Notes

## What changed

- Four new per-tenant search-index tables ship with this release:
  `npc_search_index`, `monster_search_index`, `reactor_search_index`,
  `item_string_search_index`. Each backs a sparse `?search=` fast path served
  by atlas-data.
- `map_search_index` (task-005) is now backed by the same shared
  `searchindex` helper package as the four new tables.
- `GET /api/data/maps/search-index/backfill` (task-005's one-shot backfill
  endpoint) and `services/atlas-data/atlas.com/data/map/backfill.go` have been
  removed. Re-ingest is now the canonical populate path for every search
  index.
- atlas-ui's `NpcsPage`, `MonstersPage`, `ReactorsPage`, and `ItemsPage` are
  converted to debounced type-to-search with `keepPreviousData`. `?q=` is
  written to the URL on debounce-settle. `NpcsPage` adds a storebank toggle
  that composes with search and round-trips through the URL as
  `?storebank=true`.

## Deploy order

1. Deploy atlas-data. Its migrations will create the four new tables and
   their trigram indexes. Existing tenants' tables start empty.
2. **Re-ingest wz data for every active tenant.** This populates each
   search-index table in the same transaction that writes the source
   `documents` row. Until a tenant re-ingests, `?search=` returns an empty
   result set for that tenant and the UI shows "no matches."
3. Deploy atlas-ui. The new UI assumes the fast-path endpoints return sparse
   results; without the atlas-data deploy it would still work (legacy slow
   path is unchanged for requests without `?search=`), but the faster list
   pages rely on the new endpoints.

## Rollback

- Dropping any `<type>_search_index` table is safe: ingest becomes a no-op
  for that index and the `?search=` handler returns 500 until the table is
  recreated.
- Rolling back the code without dropping the tables is safe: the tables stop
  being written; re-ingest on the forward deploy will re-populate them.

## Verification

Use the SQL count check in `services/atlas-data/README.md` under "Search
Indexes" — each `<type>_search_index` row count should equal the `documents`
count for the matching type once re-ingest completes.
