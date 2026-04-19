# Item Detail Redesign — API Contracts

Supplementary detail for §5 of `prd.md`. Describes every new or modified endpoint, along with request/response examples.

All endpoints are tenant-scoped via the existing `TENANT_ID` / `REGION` / `MAJOR_VERSION` / `MINOR_VERSION` headers. Content type for JSON:API routes is `application/vnd.api+json`.

---

## 1. `GET /commodities/items/{itemId}` — atlas-npc-shops

Reverse lookup: "which NPC shops in this tenant sell this item?"

**Request**

```
GET /commodities/items/1002357
Accept: application/vnd.api+json
TENANT_ID: ...
```

**Response — 200 OK**

```json
{
  "data": [
    {
      "type": "commodities",
      "id": "f0a3…-uuid",
      "attributes": {
        "npcId": 9200000,
        "templateId": 1002357,
        "mesoPrice": 50000,
        "discountRate": 0,
        "tokenTemplateId": 0,
        "tokenPrice": 0,
        "period": 0,
        "levelLimit": 0
      }
    },
    {
      "type": "commodities",
      "id": "c1b9…-uuid",
      "attributes": {
        "npcId": 2040000,
        "templateId": 1002357,
        "mesoPrice": 0,
        "discountRate": 0,
        "tokenTemplateId": 4031000,
        "tokenPrice": 5,
        "period": 0,
        "levelLimit": 30
      }
    }
  ]
}
```

**Response — 200 OK, empty**

```json
{ "data": [] }
```

**Errors**

| Status | Cause |
|---|---|
| 400 | `itemId` not a valid uint32 |
| 500 | DB error |

No 404 — an item no one sells returns an empty array.

---

## 2. `GET /data/npcs/{npcId}/map` — atlas-data

Returns the primary map an NPC spawns on in the active tenant.

**Request**

```
GET /data/npcs/9200000/map
Accept: application/vnd.api+json
```

**Response — 200 OK**

```json
{
  "data": {
    "type": "npc-maps",
    "id": "9200000",
    "attributes": {
      "mapId": 100000000,
      "name": "Henesys",
      "streetName": "Victoria Road",
      "spawnCount": 1
    }
  }
}
```

**Errors**

| Status | Cause |
|---|---|
| 400 | `npcId` not a valid uint32 |
| 404 | NPC has no `npc_spawn_index` row in this tenant |
| 500 | DB error |

The 404 path is the signal the caller uses to hide the map badge on the NPC shop widget.

---

## 3. `GET /data/commodity/by-item/{itemId}` — atlas-data

Returns every cash-shop commodity row that points at the given item. Distinct from the existing `GET /data/commodity/items/{itemId}` (which is keyed by commodity SN, not by itemId).

**Request**

```
GET /data/commodity/by-item/1002357
Accept: application/vnd.api+json
```

**Response — 200 OK**

```json
{
  "data": [
    {
      "type": "commodities",
      "id": "51001",
      "attributes": {
        "itemId": 1002357,
        "count": 1,
        "price": 3900,
        "period": 30,
        "priority": 100,
        "gender": 2,
        "onSale": true
      }
    }
  ]
}
```

**Response — 200 OK, empty**

```json
{ "data": [] }
```

**Errors**

| Status | Cause |
|---|---|
| 400 | `itemId` not a valid uint32 |
| 500 | Registry access failure (should not happen in practice) |

---

## 4. `GET /data/equipment/{itemId}` — modified (atlas-data)

Additive fields only. Existing fields unchanged.

**Response — 200 OK (additive fields called out)**

```jsonc
{
  "data": {
    "type": "statistics",
    "id": "1002357",
    "attributes": {
      "strength": 15,
      "dexterity": 15,
      "intelligence": 15,
      "luck": 15,
      "hp": 0,
      "mp": 0,
      "weaponAttack": 0,
      "magicAttack": 0,
      "weaponDefense": 150,
      "magicDefense": 150,
      "accuracy": 20,
      "avoidability": 20,
      "speed": 0,
      "jump": 0,
      "slots": 10,

      // new
      "reqLevel": 50,
      "reqJob": 0,
      "reqStr": 0,
      "reqDex": 0,
      "reqInt": 0,
      "reqLuk": 0,
      "reqPop": 0,
      "reqFame": 0,

      "cash": false,
      "price": 500000,
      "timeLimited": false
    }
  }
}
```

**Field semantics**

| Field | Type | Notes |
|---|---|---|
| `reqLevel` | uint16 | Minimum character level |
| `reqJob` | uint16 | Job bitmask. `0` = any. `1` = warrior, `2` = magician, `4` = bowman, `8` = thief, `16` = pirate. Combine with bitwise OR for "any-of" |
| `reqStr` / `reqDex` / `reqInt` / `reqLuk` | uint16 | Minimum stat values |
| `reqPop` | uint16 | Some regions store fame under this key |
| `reqFame` | uint16 | Other regions use this key |

Both `reqPop` and `reqFame` are read. Emit whichever the WZ populated; suppress the other row on the UI when zero.

---

## 5. Endpoints NOT changing (called out for contrast)

- `GET /api/data/items/{itemId}/name` — existing, used as-is.
- `GET /api/data/consumables/{itemId}` / `setups` / `etcs` / `cash-items` — unchanged.
- `GET /api/drops?filter[itemId]=…` — unchanged.
- `GET /api/data/commodity/items/{itemId}` — unchanged (keyed by SN, kept for backwards-compatibility — `by-item` is the new reverse-lookup path).
- `GET /api/npcs/{npcId}/shop?include=commodities` — unchanged; the link target for the `ItemNpcShopWidget` still lands on `/npcs/{id}/shop` which uses this endpoint.
