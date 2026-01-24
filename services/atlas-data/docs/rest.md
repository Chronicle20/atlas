# REST API

## Headers

All requests require tenant identification headers:

| Header | Description |
|--------|-------------|
| TENANT_ID | Tenant UUID |
| REGION | Region code (e.g., GMS) |
| MAJOR_VERSION | Major version number |
| MINOR_VERSION | Minor version number |

## Common Query Parameters

All endpoints support JSON:API query parameters:
- `fields[resourceType]` - Comma-separated list of fields to include
- `include` - Comma-separated list of related resources to include

## Endpoints

### PATCH /api/data

Uploads game data from a ZIP file.

#### Request

- Content-Type: multipart/form-data
- Form field: `zip_file` (ZIP file, max 1GB)

#### Response

- 202 Accepted: Upload processing started
- 400 Bad Request: Invalid file
- 413 Request Entity Too Large: File exceeds size limit

---

### GET /api/data/cash/items

Returns all cash items.

#### Response Model

```json
{
  "data": [{
    "type": "cash_items",
    "id": "5000000",
    "attributes": {
      "slotMax": 100,
      "spec": {}
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

Returns all character templates.

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

Returns all commodity items.

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

### GET /api/data/consumables

Returns all consumables.

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

---

### GET /api/data/equipment/{equipmentId}/slots

Returns equipment slots.

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

---

### GET /api/data/etcs

Returns all ETC items.

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

### GET /api/data/maps

Returns all maps.

#### Response Model

- 200: Array of maps resources

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

---

### GET /api/data/maps/{mapId}/portals

Returns all portals in a map.

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

Returns all reactors in a map.

#### Parameters

- mapId (path): Map ID

#### Response Model

- 200: Array of reactors resources (map reactor sub-model)
- 404: Map not found

---

### GET /api/data/maps/{mapId}/npcs

Returns all NPCs in a map.

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

Returns all monsters in a map.

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

### GET /api/data/monsters/{monsterId}

Returns monster information.

#### Parameters

- monsterId (path): Monster ID

#### Response Model

- 200: monsters resource
- 404: Not found

---

### GET /api/data/monsters/{monsterId}/loseItems

Returns lose items for a monster.

#### Parameters

- monsterId (path): Monster ID

#### Response Model

- 200: Array of lose item objects
- 404: Monster not found

---

### GET /api/data/pets

Returns all pets.

#### Response Model

- 200: Array of pets resources

---

### GET /api/data/pets/{petId}

Returns a specific pet.

#### Parameters

- petId (path): Pet ID

#### Response Model

- 200: pets resource with skills relationship
- 404: Not found

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

Returns all setup items.

#### Response Model

- 200: Array of setups resources

---

### GET /api/data/setups/{setupId}

Returns a specific setup item.

#### Parameters

- setupId (path): Setup ID

#### Response Model

- 200: setups resource
- 404: Not found

---

### GET /api/data/skills/{skillId}

Returns skill information.

#### Parameters

- skillId (path): Skill ID

#### Response Model

- 200: skills resource with effects
- 404: Not found
