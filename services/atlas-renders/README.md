# atlas-renders

atlas-renders serves PNG image renders over HTTP: composited character
sprites (assembled from equipped-item part atlases) and composited map
images (assembled from Map.wz layer data), plus a redirect to pre-rendered
minimap assets. It caches finished renders and stages downloaded WZ archives
so repeat requests for the same loadout or map avoid recomputation.

## External Dependencies

- **MinIO** (S3-compatible object storage) — the service's only persistent
  store. Three buckets are used: asset atlases/manifests/layouts, a render
  cache, and raw `.wz` archives. See `docs/storage.md`.
- No relational database.
- No Kafka topics are consumed or produced. See `docs/kafka.md`.

## Runtime Configuration

Configuration is read from environment variables (`storage.ConfigFromEnv`,
`main.go`):

| Variable | Default | Purpose |
|---|---|---|
| `REST_PORT` | `8080` | HTTP listen port |
| `MINIO_ENDPOINT` | — | MinIO endpoint |
| `MINIO_ACCESS_KEY` | — | MinIO access key |
| `MINIO_SECRET_KEY` | — | MinIO secret key |
| `MINIO_USE_SSL` | `false` (unset) | `true` enables TLS to MinIO |
| `MINIO_BUCKET_ASSETS` | `atlas-assets` | Bucket for atlases/manifests/layouts |
| `MINIO_BUCKET_RENDERS` | `atlas-renders` | Bucket for the render cache |
| `MINIO_BUCKET_WZ` | `atlas-wz` | Bucket for raw `.wz` archives |
| `WZ_SCRATCH_DIR` | `/scratch/wz` | Local filesystem path where downloaded `.wz` archives are staged for parsing |

If MinIO storage initialization fails at startup, the service still starts
but the character/map render handlers respond `503 storage-unavailable`.

Every request other than `/healthz` and `/readyz` must carry the tenant
headers `TENANT_ID`, `REGION`, `MAJOR_VERSION`, `MINOR_VERSION`; requests
missing or failing to parse these are rejected with `400`.

## Further Documentation

- [`docs/domain.md`](docs/domain.md) — domain logic and invariants
- [`docs/kafka.md`](docs/kafka.md) — Kafka integration surface
- [`docs/rest.md`](docs/rest.md) — HTTP endpoints
- [`docs/storage.md`](docs/storage.md) — persistent storage representation
