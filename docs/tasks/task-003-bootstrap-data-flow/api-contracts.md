---
name: Bootstrap Data Flow — API Contracts
description: Detailed request/response shapes for the endpoints introduced in prd.md.
type: api-contracts
task: task-003-bootstrap-data-flow
---

# Bootstrap Data Flow — API Contracts

All endpoints assume the standard Atlas tenant resolution middleware (`tenant.MustFromContext(ctx)`) and JSON:API conventions. The `<tenantId>/<region>/<major>.<minor>` components are derived server-side from the tenant header; the client never sends the path.

---

## atlas-wz-extractor

### PATCH /api/wz/input

Stage raw `.wz` files for the current tenant into the extractor's input volume.

**Request**

- `Content-Type: multipart/form-data`
- Exactly one part named `zip_file`, value is a `.zip` archive.
- Zip archive requirements (enforced by server; violating entries reject the whole upload):
  - Every entry's `Name` has no `/` or `\` path separator.
  - Every entry's `Name` matches `*.wz` (case-insensitive).
  - No entry is a directory, symlink, device, or has a `..`/absolute-path zip-slip form.

**Responses**

- `202 Accepted` — extraction to `$INPUT_WZ_DIR/<tenantId>/<region>/<major>.<minor>/` complete. Empty body.
- `400 Bad Request` — missing `zip_file` part, not a valid zip, or zip contents violate the "flat `.wz` only" rule.
  ```json
  { "error": "zip entry 'String.wz/Map.img.xml' contains a path separator" }
  ```
  No on-disk writes on 400.
- `409 Conflict` — another upload or extraction is currently running for this tenant.
  ```json
  { "error": "tenant busy: another upload or extraction is in progress" }
  ```
- `500 Internal Server Error` — filesystem failure during extract (quota exceeded, I/O error). The destination directory may be partial; client may retry.

**Side effects**

- Prior contents of `$INPUT_WZ_DIR/<tenantId>/<region>/<major>.<minor>/` are removed before extraction.
- Acquires the per-tenant mutex for the duration of the request.

---

### GET /api/wz/input

Filesystem status of the tenant's staged WZ input.

**Response** — always `200`:

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

- `fileCount: 0`, `totalBytes: 0`, `updatedAt: null` when the directory does not exist or contains no `.wz` files.
- `fileCount` counts `*.wz` entries at the TOP level only (matches the extractor's glob).
- `updatedAt` is the maximum `mtime` across those files.

---

### GET /api/wz/extractions

Filesystem status of the tenant's extracted XML output. Coexists with the existing `POST /api/wz/extractions` trigger endpoint on the same path.

**Response** — always `200`:

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
- Same null semantics as `/api/wz/input`.

---

### POST /api/wz/extractions (unchanged, documented for completeness)

Trigger extraction. Async; returns `202` with `{ "status": "started" }`. See existing implementation.

After this task's changes, the handler reads from `$INPUT_WZ_DIR/<tenantId>/<region>/<major>.<minor>/` (was: flat `$INPUT_WZ_DIR/`). On empty tenant input, returns "no WZ files found" as before.

---

## atlas-data

### GET /api/data/status

Database status of the tenant's ingested documents.

**Response** — always `200`:

```json
{
  "data": {
    "type": "dataStatus",
    "id": "<tenantId>",
    "attributes": {
      "documentCount": 18204,
      "updatedAt": "2026-04-17T18:10:00Z"
    }
  }
}
```

- `documentCount` = `SELECT COUNT(*) FROM documents WHERE tenant_id = ?`.
- `updatedAt` = `SELECT MAX(updated_at) FROM documents WHERE tenant_id = ?`, RFC 3339.
- `documentCount: 0`, `updatedAt: null` when no rows exist for the tenant.

---

### POST /api/data/process (unchanged, documented for completeness)

Trigger ingest. Deletes all documents for the tenant, then dispatches one Kafka command per worker. See existing implementation. After this task's changes, rows re-written by ingest will have `updated_at` populated automatically by GORM.
