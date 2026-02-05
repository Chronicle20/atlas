# REST API

## Endpoints

### GET /chairs/{characterId}

Retrieves chair for a specific character.

#### Parameters

| Name | Location | Type | Required |
|------|----------|------|----------|
| characterId | path | uint32 | yes |

#### Request Model

None.

#### Response Model

Single Chair resource.

| Field | Type | JSON Key |
|-------|------|----------|
| Id | uint32 | (resource id) |
| Type | string | type |
| CharacterId | uint32 | characterId |

Resource type: `chairs`

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 200 OK | Chair retrieved |
| 404 Not Found | Character not sitting on chair |
| 500 Internal Server Error | Transformation error |

---

### GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/chairs

Retrieves all chairs in use in a specific map instance.

#### Parameters

| Name | Location | Type | Required |
|------|----------|------|----------|
| worldId | path | world.Id | yes |
| channelId | path | channel.Id | yes |
| mapId | path | map.Id | yes |
| instanceId | path | uuid.UUID | yes |

#### Request Model

None.

#### Response Model

Array of Chair resources (see GET /chairs/{characterId}).

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 200 OK | Chairs retrieved |
| 500 Internal Server Error | Transformation error |
