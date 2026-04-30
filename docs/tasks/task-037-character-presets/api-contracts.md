# Character Presets — API Contracts

Companion to `prd.md`. All endpoints are JSON:API and honor the four tenant headers (`TENANT_ID`, `REGION`, `MAJOR_VERSION`, `MINOR_VERSION`) per project convention.

---

## atlas-tenants — `characters` resource (extended in place)

No new atlas-tenants endpoints. The existing `characters` configuration endpoints (tenant-scoped `GET/PUT /configurations/characters` and template-scoped `GET/PUT /templates/{templateId}/configurations/characters`) are extended to include a `presets` array next to the existing `templates` array.

### Document shape (after this change)

```jsonc
{
  "characters": {
    "templates": [
      // Existing player-creation option-lists. Unchanged shape:
      // { "jobIndex": 0, "subJobIndex": 0, "gender": 0, "mapId": 0,
      //   "faces": [...], "hairs": [...], "hairColors": [...],
      //   "skinColors": [...], "tops": [...], "bottoms": [...],
      //   "shoes": [...], "weapons": [...], "items": [...], "skills": [...] }
    ],
    "presets": [
      {
        "id": "5e1c0b6e-8a52-4c33-9f4a-6c2c1bc9c1d7",
        "attributes": {
          "name": "Hero — 4th job",
          "description": "Full 4th-job Hero with Combat Orders, Stance, etc.",
          "tags": ["4th-job", "warrior", "explorer"],
          "jobId": 112,
          "gender": 0,
          "face": 20000,
          "hair": 30030,
          "hairColor": 0,
          "skinColor": 0,
          "mapId": 240000000,
          "level": 200,
          "meso": 100000000,
          "gm": 0,
          "stats": { "str": 999, "dex": 25, "int": 4, "luk": 4, "hp": 30000, "mp": 6000 },
          "defaultName": "AdminHero",
          "equipment": [
            { "templateId": 1002357, "useAverageStats": true },
            { "templateId": 1040002, "useAverageStats": true },
            { "templateId": 1060002, "useAverageStats": true },
            { "templateId": 1072001, "useAverageStats": true },
            { "templateId": 1402039, "useAverageStats": true }
          ],
          "inventory": [
            { "templateId": 2000000, "quantity": 200 },
            { "templateId": 2070005, "quantity": 800 }
          ],
          "skills": [
            { "skillId": 1121008, "level": 30 },
            { "skillId": 1121011, "level": 30 }
          ]
        }
      }
    ]
  }
}
```

A document with no `presets` key (e.g. a tenant predating this change) is treated as `presets: []`. A PUT may omit `presets` to leave the existing array unchanged, matching the partial-update behavior the existing `templates` UI relies on (`updateTemplate.mutate({ id, updates: { characters: { templates: data.templates } } })` in `services/atlas-ui/src/pages/templates-character-templates-form.tsx`); a PUT that includes `presets` replaces it atomically. The same partial-update semantics apply at template scope.

### Validation (PUT)

In addition to the existing `templates` validation, the `presets` array is validated:

- `id` must be a valid UUID (or absent — server generates one).
- `name` non-empty, ≤ 64 chars.
- `description` ≤ 512 chars.
- `jobId` resolves to a known `job.Id`.
- `gender ∈ {0,1}`; `level ∈ [1,250]`.
- `face`, `hair`, `hairColor`, `skinColor` non-negative.
- `equipment[*].templateId` exists in atlas-data; no two entries resolve to the same equip slot (atlas-data lookup at validation time).
- `inventory[*].templateId` exists in atlas-data; `quantity ≥ 1`.
- `skills[*].skillId` exists in atlas-data; `level ≥ 1`.

Validation failures return `400` with a JSON:API `errors[]` body. Errors against the `presets` array carry `meta.path` rooted at `characters.presets[i]` so the UI can attach them to the right preset.

---

## atlas-character-factory — preset application

### `POST /factory/characters/from-preset`

Instantiate a character from a preset.

**Request body**

```json
{
  "presetId": "5e1c0b6e-8a52-4c33-9f4a-6c2c1bc9c1d7",
  "accountId": 12345,
  "worldId": 0,
  "name": "AdminHero"
}
```

**Response** `202 Accepted`

```json
{ "transactionId": "9c57b9b4-1d2c-4b3b-8e3f-2a4d6f8a0c11" }
```

The response is identical in shape to the existing `POST /factory/characters` endpoint so the same client code can poll the saga status by `transactionId`.

**Errors**

- `400 Bad Request` — invalid input shape, or name validation failure (regex / blocked-name / duplicate-in-world). Body is JSON:API `errors[]`.
- `404 Not Found` — preset id not present in the active tenant's `characters.presets` array.
- `409 Conflict` — name already taken in the target world (returned synchronously; the factory checks via atlas-character before emitting the saga).
- `502 Bad Gateway` — atlas-data skill-master-level lookup failed unrecoverably (factory falls back to `level` as `masterLevel` and emits the saga rather than failing — see FR-11; this status is reserved for unrecoverable upstream errors).

### `GET /factory/characters/name-validity`

Check whether a character name would be accepted before invoking the apply call. Used by the Admin Bootstrap wizard and the Apply Preset dialog to surface validation errors inline.

**Query parameters**

- `name` — candidate character name. Required.
- `worldId` — uint8. Required (uniqueness check is per-world).

**Response** `200 OK`

```json
{ "valid": true }
```

or

```json
{ "valid": false, "reason": "duplicate", "detail": "Name already taken in world 0." }
```

`reason` is one of `regex`, `length`, `blocked`, `duplicate`. `detail` is a human-readable string suitable for display.

**Errors**

- `400 Bad Request` — missing or malformed query parameters.

---

## Compatibility notes

- The existing `POST /factory/characters` endpoint and its `Factory RestModel` payload (documented in `services/atlas-character-factory/docs/domain.md`) are unchanged.
- The existing `characters.templates` array shape is unchanged. `characters.presets` is a new sibling array within the same `characters` configuration document — no new resource, no new URL.
- The shared `CreateAndEquipAssetPayload` in the saga library gains an optional boolean `UseAverageStats`; missing-field semantics (`omitempty` → false) preserve the existing player-creation behavior.
