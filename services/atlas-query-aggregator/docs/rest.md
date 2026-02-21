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
          "itemId": <itemId>,
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
| conditions | array | Yes | At least one condition required |
| conditions[].type | string | Yes | Condition type |
| conditions[].operator | string | Yes | Comparison operator |
| conditions[].value | int | Yes | Expected value |
| conditions[].values | []int | No | Values for "in" operator |
| conditions[].referenceId | uint32 | Conditional | Required for item, quest, map, transport, skill, buff, inventorySpace, excessSp conditions |
| conditions[].itemId | uint32 | No | Deprecated; use referenceId instead. Maps to referenceId internally. Cannot be combined with referenceId. |
| conditions[].step | string | Conditional | Required for questProgress conditions |
| conditions[].worldId | world.Id | No | World ID for mapCapacity conditions |
| conditions[].channelId | channel.Id | No | Channel ID for mapCapacity conditions |
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
          "Passed": <boolean>,
          "Description": "<description>",
          "Type": "<conditionType>",
          "Operator": "<operator>",
          "Value": <value>,
          "ItemId": <itemId>,
          "ActualValue": <actualValue>
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
| results[].Passed | bool | Whether condition passed |
| results[].Description | string | Human-readable description |
| results[].Type | string | Condition type |
| results[].Operator | string | Operator used |
| results[].Value | int | Expected value |
| results[].ItemId | uint32 | Item ID (populated for item conditions) |
| results[].ActualValue | int | Actual value from character state |

#### Condition Types

The REST input validation layer accepts the following condition types. Additional types (buddyCapacity, petCount, mapCapacity, inventorySpace, transportAvailable, skillLevel, guildLeader, buff, excessSp, partyId, partyLeader, partySize, pqCustomData) are supported by the internal condition builder but are rejected by the REST input validator with "unsupported condition type."

| Type | Description | referenceId |
|------|-------------|-------------|
| jobId | Character job ID | No |
| meso | Character currency | No |
| mapId | Character map ID | No |
| fame | Character fame | No |
| item | Item quantity in inventory | Item template ID (required) |
| gender | Character gender (0=male, 1=female) | No |
| level | Character level | No |
| reborns | Character rebirth count | No |
| dojoPoints | Dojo points | No |
| vanquisherKills | Vanquisher kill count | No |
| gmLevel | GM privilege level | No |
| guildId | Guild membership ID (value must be > 0) | No |
| guildRank | Guild rank (value must be 0-5) | No |
| questStatus | Quest state (0=not started, 1=started, 2=completed; value must be 0-3) | Quest ID (required) |
| questProgress | Quest progress value | Quest ID (required); step also required |
| hasUnclaimedMarriageGifts | Marriage gift availability (0=none, 1=has gifts; only "=" operator) | No |
| strength | Strength stat | No |
| dexterity | Dexterity stat | No |
| intelligence | Intelligence stat | No |
| luck | Luck stat | No |
| hp | Current HP | No |
| maxHp | Maximum HP | No |

#### Operators

| Operator | Description |
|----------|-------------|
| = | Equals |
| > | Greater than |
| < | Less than |
| >= | Greater than or equal |
| <= | Less than or equal |
| in | Value in list (requires values array) |

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Missing character ID (id field is 0) |
| 400 | No conditions provided |
| 400 | Missing condition type |
| 400 | Missing operator |
| 400 | Unsupported condition type |
| 400 | Unsupported operator |
| 400 | 'in' operator requires values array |
| 400 | Missing required referenceId for item, quest, or other reference-requiring conditions |
| 400 | Missing required step for questProgress |
| 400 | Both itemId and referenceId specified (use referenceId only) |
| 400 | Invalid value ranges for specific condition types |
| 400 | Failed to extract validation parameters |
| 400 | Failed to validate conditions (e.g., character data retrieval failure) |
| 500 | Failed to transform validation result |

## Headers

All requests require tenant identification headers (parsed by middleware).

| Header | Description |
|--------|-------------|
| TENANT_ID | Tenant UUID |
| REGION | Region code |
| MAJOR_VERSION | Major version number |
| MINOR_VERSION | Minor version number |
