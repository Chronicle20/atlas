# atlas-data

A RESTful resource providing static game data for Mushroom game clients.

## Overview

The atlas-data service serves static game data parsed from WZ archives. It is tenant-aware, supporting tenant-specific data isolation as well as a version-scoped "canonical" (shared) dataset. Source WZ archives are uploaded to MinIO, ingested by a Kubernetes Job (`MODE=ingest`) into per-domain JSON documents in PostgreSQL, and served via JSON:API compliant REST endpoints. The service also derives image/atlas assets (icons, minimaps, character sprite atlases, world icons) into MinIO during ingest, and supports publishing/restoring a canonical dataset as a portable baseline dump.

## External Dependencies

- PostgreSQL database
- MinIO (object storage for uploaded WZ archives, derived assets, and canonical baseline dumps)
- Kafka (legacy in-process worker dispatch; see [Kafka Integration](docs/kafka.md))
- Redis (ingest-job heartbeat/lifecycle tracking)
- Kubernetes API (in-cluster; used to create and watch `MODE=ingest` Jobs)
- Jaeger/OTLP collector (for tracing)

## Runtime Modes

The `MODE` environment variable selects the process's role at startup:

| MODE | Behavior |
|------|----------|
| (unset) | Runs the REST API server. `POST /api/data/process` is unavailable (returns 503) because the Kubernetes JobCreator is not provisioned. |
| `rest` | Runs the REST API server, provisions a Kubernetes `JobCreator` and restart-recovery, and starts a `Watchdog` goroutine that deletes stale ingest Jobs. `POST /api/data/process` is available. |
| `ingest` | Runs no REST server. Reads `SCOPE`/`REGION`/`MAJOR_VERSION`/`MINOR_VERSION`/`SCRATCH_DIR` from the environment, fetches WZ archives from MinIO, and runs the registered ingest Workers. Exits when ingest completes. Intended to run as a Kubernetes Job rendered by the REST pod. |

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| DB_USER | PostgreSQL username |
| DB_PASSWORD | PostgreSQL password |
| DB_HOST | PostgreSQL host |
| DB_PORT | PostgreSQL port |
| DB_NAME | PostgreSQL database name |
| BOOTSTRAP_SERVERS | Kafka broker address |
| TRACE_ENDPOINT | OTLP gRPC trace collector endpoint |
| TRACE_SAMPLING_RATIO | Trace sampling ratio (optional) |
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| REST_PORT | Port for the REST API server |
| MODE | Process role: unset, `rest`, or `ingest` (see Runtime Modes) |
| COMMAND_TOPIC_DATA | Kafka topic for the legacy data-processing command pipeline |
| EVENT_TOPIC_DATA | Kafka topic for the `DATA_UPDATED` event |
| DATA_EVENTS_PRODUCER_ENABLED | When set to a parseable bool, enables/disables `DATA_UPDATED` event emission (default enabled) |
| ZIP_DIR | Root directory for WZ data files consumed by the legacy Kafka-triggered worker path |
| MINIO_ENDPOINT | MinIO endpoint |
| MINIO_ACCESS_KEY | MinIO access key |
| MINIO_SECRET_KEY | MinIO secret key |
| MINIO_USE_SSL | `"true"` to use SSL against MinIO |
| MINIO_BUCKET_WZ | Bucket for uploaded WZ archives (default `atlas-wz`) |
| MINIO_BUCKET_ASSETS | Bucket for derived assets (icons, minimaps, atlases) (default `atlas-assets`) |
| MINIO_BUCKET_RENDERS | Bucket for rendered composites (default `atlas-renders`) |
| MINIO_BUCKET_CANONICAL | Bucket for baseline publish/restore dumps (default `atlas-canonical`) |
| REDIS_URL | Redis connection address (ingest-job heartbeat/lifecycle registry) |
| REDIS_PASSWORD | Redis password |
| ATLAS_ENV | Environment-scoped Redis key prefix |
| POD_NAMESPACE | Kubernetes namespace the REST pod runs in (falls back to the in-cluster service account namespace file, then `default`) |
| SCOPE | `MODE=ingest` only: `shared` or `tenants/<tenantId>` |
| REGION | `MODE=ingest` only: region code for the archive set to ingest |
| MAJOR_VERSION | `MODE=ingest` only: client major version |
| MINOR_VERSION | `MODE=ingest` only: client minor version |
| SCRATCH_DIR | `MODE=ingest` only: local scratch directory for downloaded/serialized WZ data (default `/scratch`) |
| INGEST_MAX_PARALLEL | `MODE=ingest` only: max concurrent Workers after the `String` prerequisite completes (default 4) |

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
| `item_string_search_index` | `item.StringStorage.Add` | `GET /api/data/item-strings?search=` or `?filter[*]=` |

The helper that backs all five lives in `searchindex/`. Two additional lookup tables, `monster_spawn_index` and `npc_spawn_index`, are written alongside `map_search_index` (from `map.Storage.Add`) and back `GET /api/data/monsters/{monsterId}/maps` and `GET /api/data/npcs/{npcId}/maps`.

Every search-index read resolves a single tenant partition: if the active tenant has any rows in the resource's search-index table, only that tenant's rows are visible; otherwise the request falls back wholesale to the version-scoped canonical partition (see `docs/domain.md#Canonical`). See `docs/rest.md` for the per-endpoint tenant semantics note.

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
