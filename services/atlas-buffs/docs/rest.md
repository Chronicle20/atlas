# REST API

## Endpoints

### GET /characters/{characterId}/buffs

Retrieves all active buffs for a character.

#### Parameters

| Name | Location | Type | Required |
|------|----------|------|----------|
| characterId | path | uint32 | yes |

#### Request Model

None.

#### Response Model

Array of Buff resources.

| Field | Type | JSON Key |
|-------|------|----------|
| Id | string | (resource id) |
| SourceId | int32 | sourceId |
| Duration | int32 | duration |
| Changes | []StatChange | changes |
| CreatedAt | time.Time | createdAt |
| ExpiresAt | time.Time | expiresAt |

Resource type: `buffs`

##### StatChange (nested)

| Field | Type | JSON Key |
|-------|------|----------|
| Type | string | type |
| Amount | int32 | amount |

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 200 OK | Buffs retrieved |
| 404 Not Found | Character not found in registry |
| 500 Internal Server Error | Transformation error |
