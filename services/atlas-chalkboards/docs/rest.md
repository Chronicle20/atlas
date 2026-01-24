# REST API

## Endpoints

### GET /chalkboards/{characterId}

Retrieves chalkboard message for a specific character.

#### Parameters

| Name | Location | Type | Required |
|------|----------|------|----------|
| characterId | path | uint32 | yes |

#### Request Model

None.

#### Response Model

Single Chalkboard resource.

| Field | Type | JSON Key |
|-------|------|----------|
| Id | uint32 | (resource id) |
| Message | string | message |

Resource type: `chalkboards`

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 200 OK | Chalkboard retrieved |
| 404 Not Found | Character has no chalkboard message |
| 500 Internal Server Error | Transformation error |

---

### GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/chalkboards

Retrieves all chalkboard messages in a specific map.

#### Parameters

| Name | Location | Type | Required |
|------|----------|------|----------|
| worldId | path | world.Id | yes |
| channelId | path | channel.Id | yes |
| mapId | path | map.Id | yes |

#### Request Model

None.

#### Response Model

Array of Chalkboard resources (see GET /chalkboards/{characterId}).

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 200 OK | Chalkboards retrieved |
| 500 Internal Server Error | Transformation error |
