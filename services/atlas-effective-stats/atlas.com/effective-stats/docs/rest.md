# REST API

## Endpoints

### GET /worlds/{worldId}/channels/{channelId}/characters/{characterId}/stats

Retrieves computed effective stats for a character.

#### Parameters

| Name | In | Type | Description |
|------|-----|------|-------------|
| worldId | path | byte | World identifier |
| channelId | path | byte | Channel identifier |
| characterId | path | uint32 | Character identifier |

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
| bonuses | []BonusRestModel | All active bonuses |

##### BonusRestModel

| Field | Type | Description |
|-------|------|-------------|
| source | string | Source identifier |
| statType | string | Stat type affected |
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
| 500 | Internal error retrieving effective stats |
