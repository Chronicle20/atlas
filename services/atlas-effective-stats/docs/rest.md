# REST API

## Endpoints

### GET /api/worlds/{worldId}/channels/{channelId}/characters/{characterId}/stats

Retrieves computed effective stats for a character. If the character has not been initialized yet, lazy initialization is performed by fetching data from external services.

#### Parameters

| Name | In | Type | Description |
|------|-----|------|-------------|
| worldId | path | int | World identifier |
| channelId | path | int | Channel identifier |
| characterId | path | int | Character identifier |

#### Request Model

None.

#### Response Model

JSON:API resource of type `effective-stats`.

| Field | Type | Description |
|-------|------|-------------|
| id | string | Character ID |
| strength | uint32 | Effective strength |
| dexterity | uint32 | Effective dexterity |
| luck | uint32 | Effective luck |
| intelligence | uint32 | Effective intelligence |
| maxHP | uint32 | Effective maximum HP |
| maxMP | uint32 | Effective maximum MP |
| weaponAttack | uint32 | Effective physical attack |
| weaponDefense | uint32 | Effective physical defense |
| magicAttack | uint32 | Effective magic attack |
| magicDefense | uint32 | Effective magic defense |
| accuracy | uint32 | Effective accuracy |
| avoidability | uint32 | Effective avoidability |
| speed | uint32 | Effective movement speed |
| jump | uint32 | Effective jump height |
| bonuses | []BonusRestModel | All active bonuses (omitted if empty) |

##### BonusRestModel

| Field | Type | Description |
|-------|------|-------------|
| source | string | Source identifier (e.g., `equipment:1001`, `buff:2311003`, `passive:1000001`) |
| statType | string | Stat type affected (e.g., `strength`, `max_hp`) |
| amount | int32 | Flat bonus value |
| multiplier | float64 | Percentage bonus value |

#### Response Example

```json
{
  "data": {
    "type": "effective-stats",
    "id": "12345",
    "attributes": {
      "strength": 150,
      "dexterity": 80,
      "luck": 45,
      "intelligence": 50,
      "maxHP": 8500,
      "maxMP": 2500,
      "weaponAttack": 120,
      "weaponDefense": 85,
      "magicAttack": 60,
      "magicDefense": 70,
      "accuracy": 95,
      "avoidability": 40,
      "speed": 120,
      "jump": 115,
      "bonuses": [
        {
          "source": "equipment:1001",
          "statType": "strength",
          "amount": 15,
          "multiplier": 0
        },
        {
          "source": "buff:2311003",
          "statType": "max_hp",
          "amount": 0,
          "multiplier": 0.6
        }
      ]
    }
  }
}
```

#### Error Conditions

| Status Code | Condition |
|-------------|-----------|
| 400 | Invalid worldId, channelId, or characterId path parameter |
| 500 | Internal error retrieving effective stats |
