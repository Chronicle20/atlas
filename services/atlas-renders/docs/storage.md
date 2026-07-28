## Tables

atlas-renders has no relational database and defines no tables. Its
persistent storage is MinIO object storage across three configured
buckets (bucket names are environment-configured; see `README.md`):
`atlas-assets`, `atlas-renders`, and `atlas-wz`.

Object key shapes, by bucket:

**`atlas-assets`** — `scope` is `tenants/<tenantID>` or `shared`.

| Key shape | Content |
|---|---|
| `<scope>/regions/<region>/versions/<version>/atlases/<partClass>/<id>.png` | Character part atlas image |
| `<scope>/regions/<region>/versions/<version>/atlases/<partClass>/<id>.json` | Character part atlas manifest (JSON) |
| `<scope>/regions/<region>/versions/<version>/map/<mapID>/layout.json` | Map layout (JSON) |
| `<scope>/regions/<region>/versions/<version>/character-meta/smap.json` | Layer-name → slot-codes map (JSON) |
| `<scope>/regions/<region>/versions/<version>/character-meta/zmap.json` | Ordered layer-name render-order list (JSON) |

**`atlas-renders`** — always tenant-scoped (`tenants/<tenantID>`), never
`shared`.

| Key shape | Content |
|---|---|
| `tenants/<tenantID>/regions/<region>/versions/<version>/character/<hash>.png` | Cached character render |
| `tenants/<tenantID>/regions/<region>/versions/<version>/map/<mapID>/render.png` | Cached map render |

**`atlas-wz`** — `scope` is `tenants/<tenantID>` or `shared`.

| Key shape | Content |
|---|---|
| `<scope>/regions/<region>/versions/<version>/<archive>` | Raw `.wz` archive (e.g. `Map.wz`) |

## Relationships

- `scope` (`tenants/<tenantID>` or `shared`) links a tenant/region/version
  to the `atlas-assets` prefix under which its atlases, layouts, and
  character-meta sidecars are stored.
- `character-meta/smap.json` and `character-meta/zmap.json` are
  co-located under the same scope and are treated as emitted together.
- `atlas-renders` cache keys for both character and map renders are
  always tenant-scoped, independent of whether the corresponding
  `atlas-assets` lookup resolved to `tenants/<tenantID>` or `shared`.
- `atlas-wz` archive keys use the same `<scope>/regions/<region>/versions/<version>/` prefix shape as `atlas-assets`.

## Indexes

Not applicable — MinIO provides key-based object lookup only; there are
no index structures over object storage.

## Migration Rules

Not applicable — there is no relational schema and no migration tooling.
