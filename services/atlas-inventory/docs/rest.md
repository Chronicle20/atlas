# REST

## Endpoints

### GET /characters/{characterId}/inventory

Retrieves the inventory for a character.

#### Parameters

| Name | Type | Location | Required |
|------|------|----------|----------|
| characterId | uint32 | path | yes |

#### Request Model

None

#### Response Model

```
{
  "data": {
    "type": "inventories",
    "id": "<uuid>",
    "attributes": {
      "characterId": <uint32>
    },
    "relationships": {
      "compartments": {
        "data": [
          { "type": "compartments", "id": "<uuid>" }
        ]
      }
    }
  },
  "included": [
    {
      "type": "compartments",
      "id": "<uuid>",
      "attributes": {
        "type": <byte>,
        "capacity": <uint32>
      },
      "relationships": {
        "assets": {
          "data": [
            { "type": "assets", "id": "<uint32>" }
          ]
        }
      }
    },
    {
      "type": "assets",
      "id": "<uint32>",
      "attributes": {
        "slot": <int16>,
        "templateId": <uint32>,
        "expiration": "<timestamp>",
        "createdAt": "<timestamp>",
        "quantity": <uint32>,
        "ownerId": <uint32>,
        "flag": <uint16>,
        "rechargeable": <uint64>,
        "strength": <uint16>,
        "dexterity": <uint16>,
        "intelligence": <uint16>,
        "luck": <uint16>,
        "hp": <uint16>,
        "mp": <uint16>,
        "weaponAttack": <uint16>,
        "magicAttack": <uint16>,
        "weaponDefense": <uint16>,
        "magicDefense": <uint16>,
        "accuracy": <uint16>,
        "avoidability": <uint16>,
        "hands": <uint16>,
        "speed": <uint16>,
        "jump": <uint16>,
        "slots": <uint16>,
        "levelType": <byte>,
        "level": <byte>,
        "experience": <uint32>,
        "hammersApplied": <uint32>,
        "equippedSince": "<timestamp|null>",
        "cashId": "<int64 as string>",
        "commodityId": <uint32>,
        "purchaseBy": <uint32>,
        "petId": <uint32>
      }
    }
  ]
}
```

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 404 | Inventory not found |
| 500 | Internal error |

---

### POST /characters/{characterId}/inventory

Creates a default inventory for a character.

#### Parameters

| Name | Type | Location | Required |
|------|------|----------|----------|
| characterId | uint32 | path | yes |

#### Request Model

None

#### Response Model

Same as GET /characters/{characterId}/inventory

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 500 | Internal error or inventory already exists |

---

### DELETE /characters/{characterId}/inventory

Deletes a character's inventory.

#### Parameters

| Name | Type | Location | Required |
|------|------|----------|----------|
| characterId | uint32 | path | yes |

#### Request Model

None

#### Response Model

None

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 500 | Internal error |

---

### GET /characters/{characterId}/inventory/compartments/{compartmentId}

Retrieves a specific compartment by ID.

#### Parameters

| Name | Type | Location | Required |
|------|------|----------|----------|
| characterId | uint32 | path | yes |
| compartmentId | uuid | path | yes |

#### Request Model

None

#### Response Model

```
{
  "data": {
    "type": "compartments",
    "id": "<uuid>",
    "attributes": {
      "type": <byte>,
      "capacity": <uint32>
    },
    "relationships": {
      "assets": {
        "data": [
          { "type": "assets", "id": "<uint32>" }
        ]
      }
    }
  },
  "included": [
    {
      "type": "assets",
      "id": "<uint32>",
      "attributes": {
        "slot": <int16>,
        "templateId": <uint32>,
        "expiration": "<timestamp>",
        "createdAt": "<timestamp>",
        "quantity": <uint32>,
        "ownerId": <uint32>,
        "flag": <uint16>,
        "rechargeable": <uint64>,
        "strength": <uint16>,
        "dexterity": <uint16>,
        "intelligence": <uint16>,
        "luck": <uint16>,
        "hp": <uint16>,
        "mp": <uint16>,
        "weaponAttack": <uint16>,
        "magicAttack": <uint16>,
        "weaponDefense": <uint16>,
        "magicDefense": <uint16>,
        "accuracy": <uint16>,
        "avoidability": <uint16>,
        "hands": <uint16>,
        "speed": <uint16>,
        "jump": <uint16>,
        "slots": <uint16>,
        "levelType": <byte>,
        "level": <byte>,
        "experience": <uint32>,
        "hammersApplied": <uint32>,
        "equippedSince": "<timestamp|null>",
        "cashId": "<int64 as string>",
        "commodityId": <uint32>,
        "purchaseBy": <uint32>,
        "petId": <uint32>
      }
    }
  ]
}
```

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 404 | Compartment not found |
| 500 | Internal error |

---

### GET /characters/{characterId}/inventory/compartments

Retrieves a compartment by inventory type.

#### Parameters

| Name | Type | Location | Required |
|------|------|----------|----------|
| characterId | uint32 | path | yes |
| type | int | query | yes |

#### Request Model

None

#### Response Model

Same as GET /characters/{characterId}/inventory/compartments/{compartmentId}

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Missing or invalid type parameter |
| 404 | Compartment not found |
| 500 | Internal error |

---

### GET /characters/{characterId}/inventory/compartments/{compartmentId}/assets

Retrieves all assets in a compartment.

#### Parameters

| Name | Type | Location | Required |
|------|------|----------|----------|
| characterId | uint32 | path | yes |
| compartmentId | uuid | path | yes |

#### Request Model

None

#### Response Model

```
{
  "data": [
    {
      "type": "assets",
      "id": "<uint32>",
      "attributes": {
        "slot": <int16>,
        "templateId": <uint32>,
        "expiration": "<timestamp>",
        "createdAt": "<timestamp>",
        "quantity": <uint32>,
        "ownerId": <uint32>,
        "flag": <uint16>,
        "rechargeable": <uint64>,
        "strength": <uint16>,
        "dexterity": <uint16>,
        "intelligence": <uint16>,
        "luck": <uint16>,
        "hp": <uint16>,
        "mp": <uint16>,
        "weaponAttack": <uint16>,
        "magicAttack": <uint16>,
        "weaponDefense": <uint16>,
        "magicDefense": <uint16>,
        "accuracy": <uint16>,
        "avoidability": <uint16>,
        "hands": <uint16>,
        "speed": <uint16>,
        "jump": <uint16>,
        "slots": <uint16>,
        "levelType": <byte>,
        "level": <byte>,
        "experience": <uint32>,
        "hammersApplied": <uint32>,
        "equippedSince": "<timestamp|null>",
        "cashId": "<int64 as string>",
        "commodityId": <uint32>,
        "purchaseBy": <uint32>,
        "petId": <uint32>
      }
    }
  ]
}
```

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 500 | Internal error |

---

### DELETE /characters/{characterId}/inventory/compartments/{compartmentId}/assets/{assetId}

Deletes a specific asset.

#### Parameters

| Name | Type | Location | Required |
|------|------|----------|----------|
| characterId | uint32 | path | yes |
| compartmentId | uuid | path | yes |
| assetId | uint32 | path | yes |

#### Request Model

None

#### Response Model

None (204 No Content)

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 500 | Internal error |
