# REST

## Endpoints

### GET /api/worlds/{worldId}/channels/{channelId}/characters/{characterId}/rates

Retrieves computed rates and contributing factors for a character.

**Parameters**

| Name | In | Type | Description |
|------|-----|------|-------------|
| worldId | path | integer | World identifier |
| channelId | path | integer | Channel identifier |
| characterId | path | uint32 | Character identifier |

**Request Model**

None.

**Response Model**

JSON:API resource type: `rates`

| Field | Type | Description |
|-------|------|-------------|
| id | string | Character identifier |
| expRate | float64 | Computed experience rate multiplier |
| mesoRate | float64 | Computed meso drop multiplier |
| itemDropRate | float64 | Computed item drop multiplier |
| questExpRate | float64 | Computed quest experience multiplier |
| factors | []FactorRestModel | Contributing rate factors |

**FactorRestModel**

| Field | Type | Description |
|-------|------|-------------|
| source | string | Factor source (e.g., `world`, `buff:2311003`, `item:1234567`) |
| rateType | string | Rate type (`exp`, `meso`, `item_drop`, `quest_exp`) |
| multiplier | float64 | Factor multiplier value |

**Error Conditions**

| Status | Condition |
|--------|-----------|
| 400 | Invalid worldId, channelId, or characterId in path |
| 500 | Internal error retrieving rates |
