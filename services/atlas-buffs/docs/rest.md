# REST API

## Endpoints

### GET /characters/{characterId}/buffs

Retrieves all active buffs for a character.

#### Parameters

| Name | Location | Type | Required |
|------|----------|------|----------|
| characterId | path | uint32 | yes |
| page[number] | query | int | no (default 1) |
| page[size] | query | int | no (default 250, max 250) |

The legacy `limit` query parameter is rejected.

#### Request Model

None.

#### Response Model

Array of Buff resources, sorted by internal buff key ascending (composite `"<sourceId>"` or `"<sourceId>:<statType>"` string).

JSON:API `meta` block:

| Field | Type | Description |
|-------|------|--------------|
| total | int | Total count of matching buffs across all pages |
| page.number | int | Current page number |
| page.size | int | Current page size |
| page.last | int | Last page number |

| Field | Type | JSON Key |
|-------|------|----------|
| Id | string | (resource id) |
| SourceId | int32 | sourceId |
| Level | byte | level |
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

Resource type: `stats`

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 200 OK | Buffs retrieved |
| 400 Bad Request | Invalid page[number]/page[size] (non-integer, out of range, or legacy limit param used) |
| 404 Not Found | Character not found in registry |
| 500 Internal Server Error | Transformation error |
