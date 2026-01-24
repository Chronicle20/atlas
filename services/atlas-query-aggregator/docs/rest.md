# REST API

## Endpoints

### POST /api/validations

Validates a set of conditions against a character's state.

#### Parameters

None.

#### Request Model

```json
{
  "data": {
    "id": "<characterId>",
    "type": "validations",
    "attributes": {
      "conditions": [
        {
          "type": "<conditionType>",
          "operator": "<operator>",
          "value": <value>,
          "values": [<value>, ...],
          "referenceId": <referenceId>,
          "step": "<step>",
          "worldId": <worldId>,
          "channelId": <channelId>,
          "includeEquipped": <boolean>
        }
      ]
    }
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | string | Yes | Character ID |
| conditions | array | Yes | Conditions to validate |
| conditions[].type | string | Yes | Condition type |
| conditions[].operator | string | Yes | Comparison operator |
| conditions[].value | int | Yes | Expected value |
| conditions[].values | []int | No | Values for "in" operator |
| conditions[].referenceId | uint32 | Conditional | Required for item, quest, map, transport, skill, buff conditions |
| conditions[].step | string | Conditional | Required for questProgress conditions |
| conditions[].worldId | byte | No | World ID for mapCapacity conditions |
| conditions[].channelId | byte | No | Channel ID for mapCapacity conditions |
| conditions[].includeEquipped | bool | No | Include equipped items in item quantity checks |

#### Response Model

```json
{
  "data": {
    "type": "validations",
    "id": "<characterId>",
    "attributes": {
      "passed": <boolean>,
      "results": [
        {
          "passed": <boolean>,
          "description": "<description>",
          "type": "<conditionType>",
          "operator": "<operator>",
          "value": <value>,
          "itemId": <itemId>,
          "actualValue": <actualValue>
        }
      ]
    }
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| passed | bool | Whether all conditions passed |
| results | array | Individual condition results |
| results[].passed | bool | Whether condition passed |
| results[].description | string | Human-readable description |
| results[].type | string | Condition type |
| results[].operator | string | Operator used |
| results[].value | int | Expected value |
| results[].itemId | uint32 | Item ID for item conditions |
| results[].actualValue | int | Actual value from character state |

#### Condition Types

| Type | Description | referenceId |
|------|-------------|-------------|
| jobId | Character job ID | No |
| meso | Character currency | No |
| mapId | Character map ID | No |
| fame | Character fame | No |
| item | Item quantity in inventory | Item template ID |
| gender | Character gender (0=male, 1=female) | No |
| level | Character level | No |
| reborns | Character rebirth count | No |
| dojoPoints | Dojo points | No |
| vanquisherKills | Vanquisher kill count | No |
| gmLevel | GM privilege level | No |
| guildId | Guild membership ID | No |
| guildLeader | Guild leader status (0=not leader, 1=leader) | No |
| guildRank | Guild rank | No |
| questStatus | Quest state (0=not started, 1=started, 2=completed) | Quest ID |
| questProgress | Quest progress value | Quest ID |
| hasUnclaimedMarriageGifts | Marriage gift availability (0=none, 1=has gifts) | No |
| strength | Strength stat | No |
| dexterity | Dexterity stat | No |
| intelligence | Intelligence stat | No |
| luck | Luck stat | No |
| buddyCapacity | Buddy list capacity | No |
| petCount | Spawned pet count | No |
| mapCapacity | Player count in map | Map ID |
| inventorySpace | Available inventory slots | Item ID |
| transportAvailable | Transport route availability (0=unavailable, 1=available) | Start map ID |
| skillLevel | Skill level | Skill ID |
| hp | Current HP | No |
| maxHp | Maximum HP | No |
| buff | Active buff status (0=inactive, 1=active) | Buff source ID |

#### Operators

| Operator | Description |
|----------|-------------|
| = | Equals |
| > | Greater than |
| < | Less than |
| >= | Greater than or equal |
| <= | Less than or equal |
| in | Value in list |

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Invalid condition format |
| 400 | Unsupported condition type |
| 400 | Unsupported operator |
| 400 | Missing required referenceId |
| 400 | Missing required step for questProgress |
| 400 | Failed to retrieve character data |

## Headers

All requests require tenant identification headers.

| Header | Description |
|--------|-------------|
| TENANT_ID | Tenant UUID |
| REGION | Region code |
| MAJOR_VERSION | Major version number |
| MINOR_VERSION | Minor version number |
