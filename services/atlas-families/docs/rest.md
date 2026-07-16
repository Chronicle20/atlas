# REST API

## Endpoints

### POST /families/{characterId}/juniors

Adds a junior to a senior's family.

#### Parameters

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | Yes | Senior character ID |

#### Request Model

```go
type AddJuniorRequest struct {
    WorldId     world.Id `json:"worldId"`
    SeniorLevel uint16   `json:"seniorLevel"`
    JuniorId    uint32   `json:"juniorId"`
    JuniorLevel uint16   `json:"juniorLevel"`
}
```

#### Response Model

Returns RestFamilyMember (JSON:API type: familyMembers)

```go
type RestFamilyMember struct {
    ID          string   `json:"id"`
    Type        string   `json:"type"`
    CharacterId uint32   `json:"characterId"`
    TenantId    string   `json:"tenantId"`
    SeniorId    *uint32  `json:"seniorId,omitempty"`
    JuniorIds   []uint32 `json:"juniorIds"`
    Rep         uint32   `json:"rep"`
    DailyRep    uint32   `json:"dailyRep"`
    Level       uint16   `json:"level"`
    World       world.Id `json:"world"`
    CreatedAt   string   `json:"createdAt"`
    UpdatedAt   string   `json:"updatedAt"`
}
```

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 Bad Request | Junior ID is zero |
| 400 Bad Request | Self-reference (senior equals junior) |
| 404 Not Found | Senior not found |
| 404 Not Found | Junior not found |
| 404 Not Found | Member not found |
| 409 Conflict | Senior has too many juniors |
| 409 Conflict | Junior already linked |
| 409 Conflict | Level difference too large |
| 409 Conflict | Not on same map |
| 500 Internal Server Error | Internal error |

---

### DELETE /families/links/{characterId}

Breaks a family link for a character.

#### Parameters

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | Yes | Character ID |
| reason | query | string | No | Reason for breaking link (default: "Member requested link break") |
| page[number] | query | int | No | Page number (default 1) |
| page[size] | query | int | No | Page size (default 250, max 250) |

The legacy `limit` query parameter is rejected.

#### Request Model

None

#### Response Model

Returns a paginated collection of RestFamilyMember (JSON:API type: familyMembers) containing every member updated by the break (the character, its former senior, and/or its former juniors), stable-sorted by characterId, with a JSON:API `meta`/`links` pagination envelope.

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 Bad Request | Invalid page[number]/page[size] (non-integer, out of range, or legacy limit param used) |
| 404 Not Found | Member not found |
| 409 Conflict | No link to break |
| 500 Internal Server Error | Internal error |

---

### GET /families/tree/{characterId}

Retrieves the complete family tree for a character.

#### Parameters

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | Yes | Character ID |
| page[number] | query | int | No | Page number (default 1) |
| page[size] | query | int | No | Page size (default 250, max 250) |

The legacy `limit` query parameter is rejected.

#### Request Model

None

#### Response Model

Returns a paginated collection of RestFamilyMember (JSON:API type: familyMembers) containing the character, its senior (if any), its juniors (if any), and its siblings (other juniors of the same senior), stable-sorted by characterId, with a JSON:API `meta`/`links` pagination envelope.

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 Bad Request | Invalid page[number]/page[size] (non-integer, out of range, or legacy limit param used) |
| 404 Not Found | Member not found |
| 500 Internal Server Error | Internal error |

---

## Resource Types

### familyMembers

JSON:API resource type for family members.

| Field | Type | Description |
|-------|------|-------------|
| id | string | Resource identifier |
| type | string | "familyMembers" |
| characterId | uint32 | Game character ID |
| tenantId | string | Tenant UUID string |
| seniorId | *uint32 | Senior character ID (omitted if null) |
| juniorIds | []uint32 | Junior character IDs |
| rep | uint32 | Total reputation |
| dailyRep | uint32 | Daily reputation |
| level | uint16 | Character level |
| world | world.Id | World identifier |
| createdAt | string | RFC3339 timestamp |
| updatedAt | string | RFC3339 timestamp |

## Error Response Format

```json
{
  "error": {
    "status": <int>,
    "title": "<string>",
    "detail": "<string>"
  }
}
```
