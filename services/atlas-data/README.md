# atlas-data

A RESTful resource providing static game data for Mushroom game clients.

## Overview

The atlas-data service serves static game data parsed from XML data files. It is tenant-aware, supporting tenant-specific data isolation. Data is loaded from WZ/XML files, stored in PostgreSQL as JSON documents, and served via JSON:API compliant REST endpoints.

## External Dependencies

- PostgreSQL database
- Kafka (for internal worker dispatch during data processing)
- Jaeger (for tracing)

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| DB_USER | PostgreSQL username |
| DB_PASSWORD | PostgreSQL password |
| DB_HOST | PostgreSQL host |
| DB_PORT | PostgreSQL port |
| DB_NAME | PostgreSQL database name |
| BOOTSTRAP_SERVERS | Kafka broker address |
| JAEGER_HOST_PORT | Jaeger OTLP gRPC endpoint (host:port) |
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| REST_PORT | Port for the REST API server |
| COMMAND_TOPIC_DATA | Kafka topic for data processing commands |
| ZIP_DIR | Root directory for WZ data files |

## Documentation

- [Domain Models](docs/domain.md)
- [Kafka Integration](docs/kafka.md)
- [REST API](docs/rest.md)
- [Storage](docs/storage.md)

## Search Indexes

Five per-tenant, trigram-indexed search tables back the `?search=` fast paths used by atlas-ui list pages:

| Table | Source | Fast path |
|-------|--------|-----------|
| `map_search_index` | `map.Storage.Add` | `GET /api/data/maps?search=` |
| `npc_search_index` | `npc.Storage.Add` | `GET /api/data/npcs?search=` or `?filter[storebank]=true` |
| `monster_search_index` | `monster.Storage.Add` | `GET /api/data/monsters?search=` |
| `reactor_search_index` | `reactor.Storage.Add` | `GET /api/data/reactors?search=` |
| `item_string_search_index` | `item.StringStorage.Add` | `GET /api/data/item-strings?search=` |

The helper that backs all five lives in `searchindex/`.

### Populating the indexes after a deploy

A new migration run creates the tables but leaves them empty. **Re-ingest wz data for every active tenant** — the ingest path writes the matching search-index row in the same transaction as the `documents` insert, so re-ingestion is the canonical way to populate these tables. Until a tenant re-ingests, `?search=` returns an empty list for that tenant and the UI shows the normal "no matches" state.

### Verification SQL

After re-ingestion, the row count in each `<type>_search_index` for a tenant should match the `documents` row count for the corresponding type:

```sql
SELECT
  (SELECT COUNT(*) FROM documents WHERE tenant_id = :tenant AND type = 'MAP')       AS map_docs,
  (SELECT COUNT(*) FROM map_search_index WHERE tenant_id = :tenant)                 AS map_idx,
  (SELECT COUNT(*) FROM documents WHERE tenant_id = :tenant AND type = 'NPC')       AS npc_docs,
  (SELECT COUNT(*) FROM npc_search_index WHERE tenant_id = :tenant)                 AS npc_idx,
  (SELECT COUNT(*) FROM documents WHERE tenant_id = :tenant AND type = 'MONSTER')   AS mon_docs,
  (SELECT COUNT(*) FROM monster_search_index WHERE tenant_id = :tenant)             AS mon_idx,
  (SELECT COUNT(*) FROM documents WHERE tenant_id = :tenant AND type = 'REACTOR')   AS rct_docs,
  (SELECT COUNT(*) FROM reactor_search_index WHERE tenant_id = :tenant)             AS rct_idx,
  (SELECT COUNT(*) FROM documents WHERE tenant_id = :tenant AND type = 'ITEM_STRING') AS item_docs,
  (SELECT COUNT(*) FROM item_string_search_index WHERE tenant_id = :tenant)         AS item_idx;
```

Each `*_docs` / `*_idx` pair should be equal. A mismatch indicates an interrupted re-ingest.
