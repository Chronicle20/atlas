# REST

## Endpoints

### PATCH /api/wz/input

Stages a `.wz` archive for the requesting tenant. Streams the multipart body to a tempfile, validates the zip, and extracts the flat entries into `$INPUT_WZ_DIR/<tenantId>/<region>/<major>.<minor>/`. Replaces any prior contents for the tenant.

#### Request

- `Content-Type: multipart/form-data`
- One part named `zip_file` — a `.zip` archive whose entries are flat `.wz` files.

Tenant identity is parsed from the standard tenant headers.

#### Validation

Every entry must pass:

- No path separator (`/` or `\`) in the entry name.
- No `..` segment or absolute path (zip-slip guard).
- Not a directory, symlink, or non-regular file.
- Extension is `.wz` (case-insensitive).

Any violation rejects the entire upload; nothing is written on 400.

#### Response

- `202 Accepted` — empty body. Files are on disk.
- `400 Bad Request` — `{"error": "..."}` with the first validation failure.
- `409 Conflict` — `{"error": "tenant busy: another upload or extraction is in progress"}` when the per-tenant mutex is held.
- `500 Internal Server Error` — filesystem failure during extract. The destination may be partial.

#### Example

```bash
curl -X PATCH http://localhost:8083/api/wz/input \
  -H "TENANT_ID: <uuid>" -H "REGION: GMS" \
  -H "MAJOR_VERSION: 83" -H "MINOR_VERSION: 1" \
  -F "zip_file=@wz-bundle.zip"
```

---

### GET /api/wz/input

Filesystem state of the tenant's staged WZ input. Always `200`.

#### Response

```json
{
  "data": {
    "type": "wzInputStatus",
    "id": "<tenantId>/<region>/<major>.<minor>",
    "attributes": {
      "fileCount": 12,
      "totalBytes": 524288000,
      "updatedAt": "2026-04-17T18:00:00Z"
    }
  }
}
```

- `fileCount` counts top-level `*.wz` entries.
- `totalBytes` sums their sizes.
- `updatedAt` is the max `mtime`, RFC 3339, or `null` when `fileCount == 0`.

---

### POST /api/wz/extractions

Triggers asynchronous WZ file extraction for the requesting tenant. Returns immediately with a 202 status. Reads from `$INPUT_WZ_DIR/<tenantId>/<region>/<major>.<minor>/*.wz`.

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `xmlOnly` | query | `string` | No | If `"true"`, only XML serialization is performed |
| `imagesOnly` | query | `string` | No | If `"true"`, only icon extraction is performed |

Tenant identity is parsed from request headers via `ParseTenant` middleware.

#### Response

**202 Accepted**

```json
{
  "status": "started"
}
```

Extraction runs asynchronously in a background goroutine. Success or failure is logged to service logs.

#### Error Conditions

| Condition | Behavior |
|---|---|
| Missing tenant headers | Request rejected by `ParseTenant` middleware |
| No WZ files found | Logged as error; no output produced |
| Individual WZ file parse failure | Logged as error; extraction continues with remaining files |
| Tenant mutex held (upload or other extract in flight) | Goroutine blocks until the mutex is released |

---

### GET /api/wz/extractions

Filesystem state of the tenant's extracted XML output. Always `200`. Coexists with the `POST` verb on the same path.

#### Response

```json
{
  "data": {
    "type": "wzExtractionStatus",
    "id": "<tenantId>/<region>/<major>.<minor>",
    "attributes": {
      "fileCount": 2341,
      "totalBytes": 1073741824,
      "updatedAt": "2026-04-17T18:05:00Z"
    }
  }
}
```

- Recursive `.xml` count under `$OUTPUT_XML_DIR/<tenantId>/<region>/<major>.<minor>/`.
- Same null semantics as `GET /api/wz/input`.
