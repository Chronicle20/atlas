## Endpoints

All routes except `/healthz` and `/readyz` require the tenant headers
`TENANT_ID` (uuid), `REGION` (string), `MAJOR_VERSION` (uint16),
`MINOR_VERSION` (uint16). A missing or unparsable header on any other
route yields `400` with a plain-text body describing the missing/invalid
header.

### GET /api/wz/character/render/{tenant}/{region}/{version}/{hash}.png

Renders (or returns a cached render of) a character composite.

**Parameters**

Path:
- `tenant` — must equal the request-context tenant id (`TENANT_ID` header,
  as a string).
- `region` — must equal the request-context tenant region.
- `version` — must equal `{MAJOR_VERSION}.{MINOR_VERSION}` of the
  request-context tenant.
- `hash` — the 16-hex-character loadout hash produced by
  `character.LoadoutHash` over the canonicalized query below; must match
  the query the request also supplies.

Query:
- `skin` (required, int) — internal skin id 0..10.
- `hair` (required, int)
- `face` (required, int)
- `stance` (optional, string, default `stand1`)
- `frame` (optional, int, default `0`, must be `>= 0`)
- `resize` (optional, int, default `2`, must be `1..4`)
- `items` (optional, comma-separated list of ints) — equipped item ids
- `gender` (optional, int) — must be `0` or `1` if present

**Request model**

No request body.

**Response model**

- `200` — `Content-Type: image/png`; body is the PNG image bytes.
  Response headers: `Cache-Control: public, max-age=86400, immutable`,
  `ETag: "<hash>"`, `X-Render-Cache: hit` (served from the render cache)
  or `X-Render-Cache: miss` (composited on this request). A `miss`
  response additionally carries `X-Render-Ms` (composite duration in
  milliseconds).
- On error: `Content-Type: application/vnd.api+json`, body is a JSON:API
  errors array with one entry:
  `{"errors":[{"status","code","title","detail"?,"meta"?}]}`.

**Error conditions**

| Status | Code | Title | Notes |
|---|---|---|---|
| 503 | `storage-unavailable` | MinIO storage not configured | Storage failed to initialize at startup |
| 400 | `tenant-mismatch` | Tenant not present in request context | |
| 400 | `invalid-input` | Missing path component | Any of `hash`/`tenant`/`region`/`version` empty |
| 400 | `tenant-mismatch` | Path tenant does not match request context | Path `tenant`/`region`/`version` differ from the request-context tenant |
| 400 | `invalid-input` | Invalid query | Query fails `ParseRenderQuery` validation |
| 400 | `hash-mismatch` | URL hash does not match query | `meta.expected`/`meta.got` carry the two hashes |
| 400 | `invalid-stance` | Unknown stance | `meta.supported` lists the valid stance values |
| 400 | `invalid-skin` | Skin id out of range | |
| 400 | `frame-out-of-range` | Frame index out of range | |
| 404 | `missing-asset` | Required sprite missing from extract | Body skin atlas not found |
| 500 | `compositor-error` | Compositor failed | Any other compositing error |
| 500 | `compositor-error` | PNG encode failed | |

### GET /api/wz/map/render/{tenant}/{region}/{version}/{mapId}/{kind}.png

Serves a map render or redirects to the pre-rendered minimap. The path
segments `tenant`, `region`, and `version` are matched by the route but
are not read by the handler; the tenant, region, and version actually
used are taken from the request-context tenant (the tenant headers).

**Parameters**

Path:
- `mapId` — uint32 map id.
- `kind` — `minimap` or `render`.

No query parameters.

**Request model**

No request body.

**Response model**

- `kind=minimap`: `302 Found` redirect to
  `/api/assets/{tenantID}/{region}/{version}/map/{mapId}/minimap.png`,
  where `{tenantID}`/`{region}`/`{version}` are the request-context
  tenant's id/region/`{major}.{minor}`.
- `kind=render`, success: `200`, `Content-Type: image/png`,
  `Cache-Control: public, max-age=86400, immutable`, body is the PNG
  image bytes. No `ETag` or `X-Render-*` headers are set.
- On error: plain-text body via `http.Error` (not JSON:API).

**Error conditions**

| Status | Body | Notes |
|---|---|---|
| 503 | `storage unavailable` | Storage failed to initialize at startup |
| 400 | `invalid mapId` | `mapId` path segment does not parse as uint32 |
| 400 | `invalid kind; expected minimap\|render` | |
| 500 | `resolve scope: <error>` | Scope resolution against the assets bucket failed |
| 404 | `map data not found` | Map layout not found for the resolved scope |
| 503 | `wz cache unavailable` | WZ archive cache was not initialized |
| 500 | `wz cache: <error>` | Downloading/opening `Map.wz` failed |
| 500 | `<error>` | Map compositing failed (`CompositeFromWZ` error text) |
| 500 | `<error>` | PNG encoding failed |

### /healthz

Liveness probe. Bypasses the tenant-header check. The route is registered
without an HTTP method restriction, so it responds to any method.

**Parameters**: none.

**Request model**: no request body.

**Response model**: `200`, plain-text body `ok`.

**Error conditions**: none.

### /readyz

Readiness probe. Bypasses the tenant-header check. The route is
registered without an HTTP method restriction, so it responds to any
method.

**Parameters**: none.

**Request model**: no request body.

**Response model**: `200`, plain-text body `ready`, when the runtime
reports ready.

**Error conditions**: `503`, plain-text body `not ready`, when the
runtime does not report ready.
