# Character render — API contracts

## Endpoint shape

Two routes participate. The dynamic render route lives on atlas-wz-extractor; the static cache route is served by atlas-assets nginx out of the existing PVC. Both share the same URL prefix so nginx can `try_files` to the cache before falling back to the dynamic route.

### Dynamic render

```
GET /api/character/{tenant}/{region}/{majorVersion}.{minorVersion}/render?
    skin={int}&
    hair={int}&
    face={int}&
    stance={string}&
    frame={int}&
    resize={int}&
    items={csv}
```

**Path parameters:**

- `tenant` — UUID identifying the tenant whose Character.wz extract drives this render.
- `region` — region string (e.g. `GMS`).
- `majorVersion.minorVersion` — game version (e.g. `83.1`).

**Query parameters:**

- `skin` — internal skin id, integer 0–10. Server maps to WZ skin id (see `data-model.md`).
- `hair` — hair templateId, e.g. 30030. Includes hair-color digit (last digit).
- `face` — face templateId, e.g. 20000.
- `stance` — one of `stand1`, `stand2`, `walk1`, `alert`, `jump`. Defaults to `stand1`.
- `frame` — non-negative integer. Validated against the stance's max frame index. Defaults to 0.
- `resize` — integer scale factor 1–4. Defaults to 2.
- `items` — comma-separated list of equipment templateIds, ascending-sorted by client (server normalizes regardless). Empty list permitted (bare body).

**Response (success):**

```
200 OK
Content-Type: image/png
Cache-Control: public, max-age=86400, immutable
ETag: "{loadout-hash}"
X-Render-Cache: miss            # or "hit" if cache served
X-Render-Ms: 312                # render duration; absent on cache hit
```

The PNG is the composited character at native dimensions `(96 * resize) × (128 * resize)`. The character's foot row is at `124 * resize`.

**Response (errors):**

```
400 Bad Request                 invalid stance / frame out of range / unknown templateId
404 Not Found                   tenant/region/version has no Character.wz extract
500 Internal Server Error       compositor failure (logged with hash)
```

Error body shape:

```json
{
  "errors": [
    {
      "status": "400",
      "code": "unknown-template-id",
      "title": "Equipment templateId not present in Character.wz extract",
      "detail": "templateId 1002357 not found in tenant ec876921.../GMS/83.1/Character.wz",
      "meta": { "templateId": 1002357 }
    }
  ]
}
```

### Static cache (atlas-assets)

```
GET /api/assets/{tenant}/{region}/{v}/character/{loadout-hash}.png
```

Standard nginx `try_files`. The same path is what the dynamic endpoint writes on render. Hit characteristics: `Cache-Control: public, max-age=86400` (already configured in `services/atlas-assets/nginx.conf:24`).

### Loadout hashing

The dynamic endpoint computes the hash from the canonical input string (see `data-model.md`), writes the PNG to `{tenant}/{region}/{v}/character/{hash}.png` on the PVC, and returns the bytes inline. Subsequent requests for the same tuple at the static path hit the same file.

Per FR-11, concurrent renders of the same hash deduplicate via a per-key file lock (`flock` on a `.lock` sidecar, or in-process sync.Map keyed by hash — implementation choice deferred to design phase).

### URL builder (atlas-ui)

`mapleStoryService.generateCharacterUrl()` is replaced with `characterRenderService.generateCharacterUrl()`:

```ts
function generateCharacterUrl(
  tenantId: string,
  region: string,
  majorVersion: number,
  minorVersion: number,
  skin: number,
  hair: number,
  face: number,
  equipment: { [slot: string]: number },
  options: {
    stance?: 'stand1' | 'stand2' | 'walk1' | 'alert' | 'jump';
    frame?: number;
    resize?: number;
  } = {},
): string;
```

Returns the dynamic render URL. The browser fetches it; the response is the PNG. No 302 dance, no redirect chain (compared to maplestory.io's two-hop pattern).

## Error mapping (atlas-ui)

The current `useCharacterImage` hook surfaces errors via `onError`. With our endpoint:

- 400 → `'character image failed to load: invalid input'` (bug in atlas-ui, surface to error logger)
- 404 → `'character assets not extracted for this tenant'` (admin-actionable)
- 5xx → `'character render service unavailable'` (retry with backoff)

The `frameMode='platform'` JS pixel-scan path is removed entirely — the new endpoint guarantees foot row position, so no client-side compensation is needed.

## OTel attributes

`character.render` span attributes:

- `tenant.id`
- `region`
- `version`
- `stance`
- `frame`
- `resize`
- `equipped_slot_count`
- `cache_hit` (bool)
- `loadout_hash`
- `render_ms` (only on miss)

Counters (Prometheus):

- `character_render_total{cache, stance}` — counter
- `character_render_errors_total{reason}` — counter; `reason ∈ {invalid-input, missing-asset, compositor-error}`
- `character_render_duration_ms` — histogram, render path only
