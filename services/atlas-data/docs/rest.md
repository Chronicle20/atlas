# REST API

## Headers

All requests require tenant identification headers:

| Header | Description |
|--------|-------------|
| TENANT_ID | Tenant UUID |
| REGION | Region code (e.g., GMS) |
| MAJOR_VERSION | Major version number |
| MINOR_VERSION | Minor version number |

Operator-gated endpoints (baseline publish/restore/list, tenant purge, `scope=shared` on WZ upload/status/process/status) additionally require:

| Header | Description |
|--------|-------------|
| X-Atlas-Operator | Must be exactly `1`; otherwise 403 |

## Common Query Parameters

All GET endpoints support JSON:API sparse fieldsets:
- `fields[resourceType]` - Comma-separated list of fields to include
- `include` - Comma-separated list of related resources to include

### Pagination

Most collection endpoints (everything documented below as "Paginated") use JSON:API §4.4 page-number pagination:

- `page[number]` (optional, default 1): must be >= 1.
- `page[size]` (optional): default and maximum vary by endpoint — either the standard default 50 / max 250, or the search-endpoint cap default 50 / max 50 (noted per endpoint below).
- Legacy `?limit=` is **rejected with 400**.
- Out-of-range/non-integer page params return 400.
- URL-encode the brackets as `%5B`/`%5D`.

A paginated response has the shape:

```json
{
  "data": [ ... ],
  "meta": {
    "total": 1234,
    "page": { "number": 1, "size": 50, "last": 25 }
  },
  "links": {
    "self":  "/api/data/<resource>?page%5Bnumber%5D=1&page%5Bsize%5D=50",
    "first": "/api/data/<resource>?page%5Bnumber%5D=1&page%5Bsize%5D=50",
    "next":  "/api/data/<resource>?page%5Bnumber%5D=2&page%5Bsize%5D=50",
    "last":  "/api/data/<resource>?page%5Bnumber%5D=25&page%5Bsize%5D=50"
  }
}
```

`meta.total` is the absolute matching row/item count across all pages. `meta.page.last` is `ceil(total/size)` with a floor of 1. `links.prev` is omitted on page 1; `links.next` is omitted on the last page. For past-end requests (`page[number] > meta.page.last`), `data` is empty and `links.prev` recovers to the last page.

### Tenant semantics for search-index-backed endpoints

`item-strings`, `maps`, `npcs`, `monsters`, and `reactors` (in `?search=` mode, and `item-strings` in filter mode) each resolve a single tenant partition per request: if the active tenant has any rows in the resource's search-index table, only that tenant's rows are visible; otherwise the global version-scoped canonical partition is used wholesale. There is no per-row merge.

## Endpoints

### POST /api/data/process

Creates a Kubernetes ingest Job (`MODE=ingest`) that fetches WZ archives for the resolved scope/region/version from MinIO and re-ingests them. Requires the service to be running with `MODE=rest` (otherwise the Kubernetes JobCreator is unavailable).

#### Query Parameters

- `scope` (optional): `""` or `"tenant"` (default) targets the caller's own tenant (`tenants/<tenantId>`); `"shared"` targets the version-scoped canonical dataset and requires `X-Atlas-Operator: 1`.

#### Response

- 202 Accepted: `{"jobName": "...", "scope": "...", "version": "<major>.<minor>"}`
- 400 Bad Request: invalid `scope` value
- 403 Forbidden: `scope=shared` without `X-Atlas-Operator: 1`
- 503 Service Unavailable: Kubernetes JobCreator unavailable (not running `MODE=rest`, or in-cluster config/ConfigMap unavailable)

---

### GET /api/data/process

Lists active/recent ingest Jobs this service manages.

#### Response

- 200: `{"jobs": [{"name","scope","region","version","tenant","active","succeeded","failed","startTime"}, ...]}` (raw JSON, not a JSON:API document)
- 503 Service Unavailable: Kubernetes client unavailable

---

### GET /api/data/status

Returns the ingested-document state for a scope. Always 200 (absent a scope/auth error).

#### Query Parameters

- `scope` (optional): `""` or `"tenant"` (default) reads the caller's own tenant's rows; `"shared"` reads the version-scoped canonical rows and requires `X-Atlas-Operator: 1`.

#### Response Model

```json
{
  "data": {
    "type": "dataStatus",
    "id": "<resolved tenantId>",
    "attributes": {
      "documentCount": 18204,
      "updatedAt": "2026-04-17T18:10:00Z"
    }
  }
}
```

- `documentCount` — number of `documents` rows with `tenant_id = <resolved tenantId>`.
- `updatedAt` — `MAX(updated_at)` across those rows, RFC 3339; `null` when `documentCount` is 0.
- 400 Bad Request: invalid `scope` value
- 403 Forbidden: `scope=shared` without `X-Atlas-Operator: 1`

---

### GET /api/data/cash/items

Returns all cash items. Paginated (default 50, max 250).

#### Response Model

```json
{
  "data": [{
    "type": "cash_items",
    "id": "5000000",
    "attributes": {
      "slotMax": 100,
      "spec": {},
      "timeWindows": []
    }
  }]
}
```

---

### GET /api/data/cash/items/{itemId}

Returns a specific cash item.

#### Parameters

- itemId (path): Cash item ID

#### Response Model

- 200: cash_items resource
- 404: Not found

---

### GET /api/data/characters/templates

Returns all character templates. Paginated (default 50, max 250).

#### Response Model

```json
{
  "data": [{
    "type": "characterTemplates",
    "id": "0",
    "attributes": {
      "characterType": "explorer",
      "faces": [],
      "hairStyles": [],
      "hairColors": [],
      "skinColors": [],
      "tops": [],
      "bottoms": [],
      "shoes": [],
      "weapons": []
    }
  }]
}
```

---

### GET /api/data/commodity/items

Returns all commodity items. Paginated (default 50, max 250).

#### Response Model

- 200: Array of commodities resources

---

### GET /api/data/commodity/items/{itemId}

Returns a specific commodity item.

#### Parameters

- itemId (path): Commodity item ID

#### Response Model

- 200: commodities resource
- 404: Not found

---

### GET /api/data/commodity/by-item/{itemId}

Returns all commodity rows for a given underlying item ID. Paginated (default 50, max 250).

#### Parameters

- itemId (path): Item ID

#### Response Model

- 200: Array of commodities resources, sorted by commodity id

---

### GET /api/data/consumables

Returns all consumables. Paginated (default 50, max 250).

#### Query Parameters

- filter[rechargeable]: Filter by rechargeable status (true/false)

#### Response Model

- 200: Array of consumables resources

---

### GET /api/data/consumables/{itemId}

Returns a specific consumable.

#### Parameters

- itemId (path): Consumable item ID

#### Response Model

- 200: consumables resource
- 404: Not found

---

### GET /api/data/cosmetics/faces

Returns all faces. Paginated (default 50, max 250).

#### Response Model

- 200: Array of faces resources

---

### GET /api/data/cosmetics/faces/{faceId}

Returns a specific face.

#### Parameters

- faceId (path): Face ID

#### Response Model

- 200: faces resource
- 404: Not found

---

### GET /api/data/cosmetics/hairs

Returns all hairs. Paginated (default 50, max 250).

#### Response Model

- 200: Array of hairs resources

---

### GET /api/data/cosmetics/hairs/{hairId}

Returns a specific hair.

#### Parameters

- hairId (path): Hair ID

#### Response Model

- 200: hairs resource
- 404: Not found

---

### GET /api/data/equipment/{equipmentId}

Returns equipment statistics.

#### Parameters

- equipmentId (path): Equipment ID

#### Response Model

```json
{
  "data": {
    "type": "statistics",
    "id": "1000000",
    "attributes": {
      "strength": 0,
      "dexterity": 0,
      "intelligence": 0,
      "luck": 0,
      "hp": 0,
      "mp": 0,
      "weaponAttack": 0,
      "magicAttack": 0,
      "weaponDefense": 0,
      "magicDefense": 0,
      "accuracy": 0,
      "avoidability": 0,
      "speed": 0,
      "jump": 0,
      "slots": 7,
      "cash": false,
      "price": 0
    },
    "relationships": {
      "slots": {}
    }
  }
}
```

- 404: Not found

---

### GET /api/data/equipment/{equipmentId}/slots

Returns equipment slots. Paginated (default 50, max 250).

#### Parameters

- equipmentId (path): Equipment ID

#### Response Model

```json
{
  "data": [{
    "type": "slots",
    "id": "helmet",
    "attributes": {
      "name": "helmet",
      "WZ": "Hp",
      "slot": -1
    }
  }]
}
```

- 404: Not found

---

### GET /api/data/etcs

Returns all ETC items. Paginated (default 50, max 250).

#### Response Model

- 200: Array of etcs resources

---

### GET /api/data/etcs/{itemId}

Returns a specific ETC item.

#### Parameters

- itemId (path): ETC item ID

#### Response Model

- 200: etcs resource
- 404: Not found

---

### GET /api/data/item-strings

Returns item strings with optional search and filter. Paginated (default 50, max 50 — search-endpoint cap).

#### Query Parameters

- `search` (optional): Filter by item ID prefix or name substring (case-insensitive). Max length 128.
- `filter[compartment]` (optional): One of `equipment`, `use`, `setup`, `etc`, `cash`.
- `filter[subcategory]` (optional): Subcategory name within the chosen compartment.
- `filter[class]` (optional): Comma-separated equipment classes (e.g. `warrior,bowman`) or the literal `any`.
- `page[number]` (optional, default 1): Page number, must be >= 1.
- `page[size]` (optional, default 50, max 50): Page size, must be in `[1, 50]`.
- Legacy `?limit=` is **rejected with 400**.

Validation order: `search` -> `filter[*]` -> `page[*]` -> `?limit=` rejection.

Item strings for item ids below 1,000,000 (character-appearance templates: faces, hairs, skins, etc.) are always excluded.

#### Response Model

```json
{
  "data": [{
    "type": "item-strings",
    "id": "1000000",
    "attributes": {
      "name": "Sword",
      "compartment": "equipment",
      "subcategory": "one-handed-sword"
    }
  }],
  "meta": { "total": 1234, "page": { "number": 1, "size": 50, "last": 25 } }
}
```

See [Tenant semantics](#tenant-semantics-for-search-index-backed-endpoints).

---

### GET /api/data/item-strings/{itemId}

Returns the name for a specific item.

#### Parameters

- itemId (path): Item ID

#### Response Model

- 200: item-strings resource
- 404: Not found

---

### GET /api/data/maps

Returns all maps, or searches maps by id/name/street name. Paginated (default 50, max 250 for the plain list; default 50, max 50 when `search` is present).

#### Query Parameters

- `search` (optional): Filter by map ID, name, or street name (case-insensitive). 400 if present and empty, or longer than 128 characters.

#### Response Model

- 200: Array of maps resources

See [Tenant semantics](#tenant-semantics-for-search-index-backed-endpoints) (applies only when `search` is used).

---

### GET /api/data/maps/{mapId}

Returns a specific map.

#### Parameters

- mapId (path): Map ID

#### Response Model

```json
{
  "data": {
    "type": "maps",
    "id": "100000000",
    "attributes": {
      "name": "Henesys",
      "streetName": "Henesys",
      "returnMapId": 100000000,
      "monsterRate": 1.0,
      "onFirstUserEnter": "",
      "onUserEnter": "",
      "fieldLimit": 0,
      "mobInterval": 0,
      "time_mob": null,
      "mapArea": {},
      "footholdTree": {},
      "areas": [],
      "seats": 0,
      "clock": false,
      "everLast": false,
      "town": true,
      "decHP": 0,
      "protectItem": 0,
      "forcedReturnMapId": 999999999,
      "boat": false,
      "timeLimit": -1,
      "fieldType": 0,
      "mobCapacity": 0,
      "recovery": 1.0,
      "backgroundTypes": [],
      "x_limit": {}
    },
    "relationships": {
      "portals": {},
      "reactors": {},
      "npcs": {},
      "monsters": {}
    }
  }
}
```

- 404: Not found

---

### GET /api/data/maps/{mapId}/portals

Returns all portals in a map. Paginated (default 50, max 250).

#### Parameters

- mapId (path): Map ID

#### Query Parameters

- name: Filter by portal name

#### Response Model

- 200: Array of portals resources
- 404: Map not found

---

### GET /api/data/maps/{mapId}/portals/{portalId}

Returns a specific portal in a map.

#### Parameters

- mapId (path): Map ID
- portalId (path): Portal ID

#### Response Model

- 200: portals resource
- 404: Not found

---

### GET /api/data/maps/{mapId}/reactors

Returns all reactors in a map. Paginated (default 50, max 250).

#### Parameters

- mapId (path): Map ID

#### Response Model

- 200: Array of reactors resources (map reactor sub-model)
- 404: Map not found

---

### GET /api/data/maps/{mapId}/npcs

Returns all NPCs in a map. Paginated (default 50, max 250).

#### Parameters

- mapId (path): Map ID

#### Query Parameters

- objectId: Filter by object ID

#### Response Model

- 200: Array of npcs resources (map NPC sub-model)
- 404: Map not found

---

### GET /api/data/maps/{mapId}/npcs/{npcId}

Returns a specific NPC in a map.

#### Parameters

- mapId (path): Map ID
- npcId (path): NPC ID

#### Response Model

- 200: npcs resource (map NPC sub-model)
- 404: Not found

---

### GET /api/data/maps/{mapId}/monsters

Returns all monsters in a map. Paginated (default 50, max 250).

#### Parameters

- mapId (path): Map ID

#### Response Model

- 200: Array of monsters resources (map monster sub-model)
- 404: Map not found

---

### POST /api/data/maps/{mapId}/drops/position

Calculates drop position in a map.

#### Parameters

- mapId (path): Map ID

#### Request Model

```json
{
  "data": {
    "type": "positions",
    "attributes": {
      "initialX": 0,
      "initialY": 0,
      "fallbackX": 0,
      "fallbackY": 0
    }
  }
}
```

#### Response Model

```json
{
  "data": {
    "type": "points",
    "attributes": {
      "x": 0,
      "y": 0
    }
  }
}
```

- 404: Not found

---

### POST /api/data/maps/{mapId}/footholds/below

Finds the foothold below a position in a map.

#### Parameters

- mapId (path): Map ID

#### Request Model

```json
{
  "data": {
    "type": "positions",
    "attributes": {
      "x": 0,
      "y": 0
    }
  }
}
```

#### Response Model

```json
{
  "data": {
    "type": "footholds",
    "id": "1",
    "attributes": {
      "first": {"x": 0, "y": 0},
      "second": {"x": 100, "y": 0}
    }
  }
}
```

---

### GET /api/data/monsters

Returns all monsters, or searches monsters by id/name. Paginated (default 50, max 250 for the plain list; default 50, max 50 when `search` is present).

#### Query Parameters

- `search` (optional): Filter by monster ID prefix or name substring (case-insensitive). 400 if present and empty, or longer than 128 characters.

#### Response Model

- 200: Array of monsters resources

See [Tenant semantics](#tenant-semantics-for-search-index-backed-endpoints) (applies only when `search` is used).

---

### GET /api/data/monsters/{monsterId}

Returns monster information.

#### Parameters

- monsterId (path): Monster ID

#### Response Model

- 200: monsters resource
- 404: Not found

---

### GET /api/data/monsters/{monsterId}/loseItems

Returns lose items for a monster. Paginated (default 50, max 250).

#### Parameters

- monsterId (path): Monster ID

#### Response Model

- 200: Array of lose item objects
- 404: Monster not found

---

### GET /api/data/monsters/{monsterId}/maps

Returns the maps a monster spawns on, with per-map spawn counts, sourced from `monster_spawn_index`. Paginated (default 50, max 250).

#### Parameters

- monsterId (path): Monster ID

#### Response Model

```json
{
  "data": [{
    "type": "monster-spawn-maps",
    "id": "100000000",
    "attributes": {
      "name": "Henesys",
      "streetName": "Henesys",
      "spawnCount": 3
    }
  }]
}
```

Sorted by spawn count descending, then map name.

---

### GET /api/data/npcs

Returns all NPCs, or searches NPCs by id/name/storebank status. Paginated (default 50, max 250 for the plain list; default 50, max 50 when `search` or `filter[storebank]` is present).

#### Query Parameters

- `filter[storebank]`: `true` to filter to storebank NPCs (triggers search-index mode even without `search`)
- `search` (optional): Filter by NPC ID prefix or name substring (case-insensitive). 400 if present and empty, or longer than 128 characters.

#### Response Model

- 200: Array of npcs resources

See [Tenant semantics](#tenant-semantics-for-search-index-backed-endpoints) (applies whenever `search` or `filter[storebank]` is used).

---

### GET /api/data/npcs/{npcId}

Returns a specific NPC.

#### Parameters

- npcId (path): NPC ID

#### Response Model

- 200: npcs resource
- 404: Not found

---

### GET /api/data/npcs/{npcId}/maps

Returns the maps an NPC spawns on, with per-map spawn counts, sourced from `npc_spawn_index`. Paginated (default 50, max 250).

#### Parameters

- npcId (path): NPC ID

#### Response Model

```json
{
  "data": [{
    "type": "npc-maps",
    "id": "100000000",
    "attributes": {
      "mapId": 100000000,
      "name": "Henesys",
      "streetName": "Henesys",
      "spawnCount": 1
    }
  }]
}
```

Sorted by spawn count descending, then map id; deduplicated by map id.

---

### GET /api/data/npcs/{npcId}/quests

Returns the quests that reference an NPC (as start/end requirement or start/end action NPC). Paginated (default 50, max 250).

#### Parameters

- npcId (path): NPC ID

#### Response Model

- 200: Array of quests resources, sorted by quest id

---

### GET /api/data/pets

Returns all pets. Paginated (default 50, max 250).

#### Response Model

- 200: Array of pets resources

---

### GET /api/data/pets/{itemId}

Returns a specific pet.

#### Parameters

- itemId (path): Pet item ID

#### Response Model

- 200: pets resource with skills relationship
- 404: Not found

---

### GET /api/data/quests

Returns all quests. Paginated (default 50, max 250).

#### Response Model

- 200: Array of quests resources

---

### GET /api/data/quests/auto-start

Returns all auto-start quests. Paginated (default 50, max 250).

#### Response Model

- 200: Array of quests resources (filtered by autoStart = true)

---

### GET /api/data/quests/{questId}

Returns a specific quest.

#### Parameters

- questId (path): Quest ID

#### Response Model

- 200: quests resource
- 404: Not found

---

### GET /api/data/reactors

Returns all reactors, or searches reactors by id/name. Paginated (default 50, max 250 for the plain list; default 50, max 50 when `search` is present).

#### Query Parameters

- `search` (optional): Filter by reactor ID prefix or name substring (case-insensitive). 400 if present and empty, or longer than 128 characters.

#### Response Model

- 200: Array of reactors resources

See [Tenant semantics](#tenant-semantics-for-search-index-backed-endpoints) (applies only when `search` is used).

---

### GET /api/data/reactors/{reactorId}

Returns reactor information.

#### Parameters

- reactorId (path): Reactor ID

#### Response Model

- 200: reactors resource
- 404: Not found

---

### GET /api/data/setups

Returns all setup items. Paginated (default 50, max 250).

#### Response Model

- 200: Array of setups resources

---

### GET /api/data/setups/{itemId}

Returns a specific setup item.

#### Parameters

- itemId (path): Setup item ID

#### Response Model

- 200: setups resource
- 404: Not found

---

### GET /api/data/mob-skills

Returns all mob skills. Paginated (default 50, max 250).

#### Response Model

- 200: Array of mob-skills resources

---

### GET /api/data/mob-skills/{skillId}

Returns all mob skills for a specific skill type. Paginated (default 50, max 250).

#### Parameters

- skillId (path): Mob skill type ID

#### Response Model

- 200: Array of mob-skills resources filtered by skill ID, sorted by level
- 400: Bad Request (invalid skillId)

---

### GET /api/data/mob-skills/{skillId}/{level}

Returns a specific mob skill by skill type and level.

#### Parameters

- skillId (path): Mob skill type ID
- level (path): Mob skill level

#### Response Model

```json
{
  "data": {
    "type": "mob-skills",
    "id": "120-1",
    "attributes": {
      "mp_con": 0,
      "duration": 0,
      "hp": 100,
      "x": 0,
      "y": 0,
      "prop": 100,
      "interval": 0,
      "count": 1,
      "limit": 0,
      "lt_x": 0,
      "lt_y": 0,
      "rb_x": 0,
      "rb_y": 0,
      "summon_effect": 0,
      "summons": []
    }
  }
}
```

- 404: Not found
- 400: Bad Request (invalid skillId or level)

---

### GET /api/data/skills

Returns skills matching an id set or a name substring. Paginated (default 50, max 250).

#### Query Parameters

- `ids` (repeatable and/or comma-separated): exact skill id match set.
- `name`: case-insensitive substring match against skill name. Ignored if `ids` is present.
- Exactly one of `ids` or `name` is required.

#### Response Model

- 200: Array of skills resources, sorted by skill id
- 400: Bad Request (neither `ids` nor `name` supplied, or an `ids` value does not parse as an integer)

---

### GET /api/data/skills/{skillId}

Returns skill information.

#### Parameters

- skillId (path): Skill ID

#### Response Model

- 200: skills resource with effects
- 404: Not found

---

### GET /api/data/jobs/{jobId}/skills

Returns the skill ids associated with a job class (from `libs/atlas-constants/job`).

#### Parameters

- jobId (path): Job ID

#### Response Model

```json
{
  "data": {
    "type": "jobs",
    "id": "0",
    "attributes": { "skills": [1001, 1002] }
  }
}
```

- 404: Not found (unknown job id)

---

### PATCH /api/data/wz

Uploads a WZ archive zip for a tenant or the shared canonical scope. The request body must be `multipart/form-data` with a `zip_file` field containing the zip; each entry's path (e.g. `Item.wz/...`) becomes the MinIO object key under the resolved scope/region/version.

#### Query Parameters

- `scope` (optional): `""` or `"tenant"` (default) targets the caller's own tenant; `"shared"` targets the version-scoped canonical dataset and requires `X-Atlas-Operator: 1`.

#### Request

- `multipart/form-data`, field name `zip_file`, containing a zip whose entries are `.wz`-suffixed files (no path traversal, no symlinks).

#### Response

- 202 Accepted: upload stored
- 400 Bad Request: not multipart, missing `zip_file` field, invalid `scope`, unreadable/invalid zip, or an entry fails validation (path traversal, symlink, non-`.wz` suffix)
- 403 Forbidden: `scope=shared` without `X-Atlas-Operator: 1`
- 503 Service Unavailable: MinIO unavailable

---

### GET /api/data/wz

Returns aggregate upload status for a scope.

#### Query Parameters

- `scope` (optional): same semantics as `PATCH /api/data/wz`.

#### Response Model

```json
{
  "data": {
    "type": "wzInputStatus",
    "id": "current",
    "attributes": {
      "fileCount": 42,
      "totalBytes": 1717986918,
      "updatedAt": "2026-04-17T18:10:00Z"
    }
  }
}
```

- 400 Bad Request: invalid `scope`
- 403 Forbidden: `scope=shared` without `X-Atlas-Operator: 1`
- 503 Service Unavailable: MinIO unavailable

---

### POST /api/data/baseline/publish

Publishes the canonical (version-scoped) subset of the searchable tables to MinIO as a tar dump plus a sha256 sidecar. Requires `X-Atlas-Operator: 1`.

#### Request Model

```json
{
  "data": {
    "type": "baselinePublishes",
    "attributes": {
      "region": "GMS",
      "majorVersion": 83,
      "minorVersion": 1
    }
  }
}
```

#### Response Model

- 202 Accepted:

```json
{
  "data": {
    "type": "baselinePublishes",
    "id": "GMS/83.1",
    "attributes": { "sha256": "..." }
  }
}
```

- 403 Forbidden: missing `X-Atlas-Operator: 1`
- 503 Service Unavailable: MinIO unavailable
- 500: publish failure (dump/upload error)

---

### POST /api/data/baseline/restore

Restores a published baseline into a single target tenant. Destructive for that tenant's rows in the dump's tables. Requires `X-Atlas-Operator: 1`.

#### Request Model

```json
{
  "data": {
    "type": "baselineRestores",
    "attributes": {
      "region": "GMS",
      "majorVersion": 83,
      "minorVersion": 1,
      "tenantId": "00000000-0000-0000-0000-000000000000"
    }
  }
}
```

#### Response

- 202 Accepted: no body
- 403 Forbidden: missing `X-Atlas-Operator: 1`
- 422 Unprocessable Entity: dump schema version mismatch, or downloaded dump's sha256 does not match its sidecar
- 503 Service Unavailable: MinIO unavailable
- 500: other restore failure (a failure here also best-effort deletes any partially-restored rows for the target tenant)

---

### GET /api/data/baselines

Lists published baselines. Paginated (default 50, max 250). Requires `X-Atlas-Operator: 1`.

#### Response Model

```json
{
  "data": [{
    "type": "baselines",
    "id": "GMS/83.1",
    "attributes": {
      "region": "GMS",
      "majorVersion": 83,
      "minorVersion": 1,
      "sha256": "...",
      "publishedAt": "2026-04-17T18:10:00Z",
      "sizeBytes": 104857600
    }
  }]
}
```

`sha256` is `""` when the sidecar object is missing or unreadable (the baseline still appears). Sorted by (region, majorVersion, minorVersion) ascending.

- 403 Forbidden: missing `X-Atlas-Operator: 1`
- 503 Service Unavailable: MinIO unavailable

---

### DELETE /api/data/tenants/{id}

Purges all data for a tenant: deletes rows for the tenant from every per-tenant table and best-effort removes its MinIO object prefixes. Requires `X-Atlas-Operator: 1`.

#### Parameters

- id (path): Tenant UUID to purge

#### Response

- 202 Accepted: purge completed (Postgres rows deleted; MinIO removal is best-effort and not reflected in the response)
- 400 Bad Request: `id` is not a valid UUID
- 403 Forbidden: missing `X-Atlas-Operator: 1`, or `id` is the canonical sentinel UUID or the caller's version-scoped canonical tenant id
- 503 Service Unavailable: MinIO unavailable
- 500: purge failure
