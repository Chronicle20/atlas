# REST

## Endpoints

### GET /api/characters/{characterId}/skills

Returns all skills for a character.

#### Parameters

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | yes | Character identifier |

#### Request Headers

| Header | Required | Description |
|--------|----------|-------------|
| TENANT_ID | yes | Tenant UUID |

#### Response Model

Resource type: `skills`

| Field | Type | Description |
|-------|------|-------------|
| id | string | Skill identifier |
| level | byte | Current skill level |
| masterLevel | byte | Master level |
| expiration | time.Time | Expiration timestamp |
| cooldownExpiresAt | time.Time | Cooldown expiration timestamp |

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 500 | Internal error retrieving skills |

---

### POST /api/characters/{characterId}/skills

Requests creation of a skill for a character.

#### Parameters

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | yes | Character identifier |

#### Request Headers

| Header | Required | Description |
|--------|----------|-------------|
| TENANT_ID | yes | Tenant UUID |

#### Request Model

Resource type: `skills`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | string | yes | Skill identifier |
| level | byte | yes | Skill level |
| masterLevel | byte | yes | Master level |
| expiration | time.Time | yes | Expiration timestamp |

#### Response

| Status | Description |
|--------|-------------|
| 202 | Command accepted |

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 500 | Internal error sending command |

---

### GET /api/characters/{characterId}/skills/{skillId}

Returns a specific skill for a character.

#### Parameters

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | yes | Character identifier |
| skillId | path | uint32 | yes | Skill identifier |

#### Request Headers

| Header | Required | Description |
|--------|----------|-------------|
| TENANT_ID | yes | Tenant UUID |

#### Response Model

Resource type: `skills`

| Field | Type | Description |
|-------|------|-------------|
| id | string | Skill identifier |
| level | byte | Current skill level |
| masterLevel | byte | Master level |
| expiration | time.Time | Expiration timestamp |
| cooldownExpiresAt | time.Time | Cooldown expiration timestamp |

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 500 | Internal error retrieving skill |

---

### PATCH /api/characters/{characterId}/skills/{skillId}

Requests update of a skill for a character.

#### Parameters

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | yes | Character identifier |
| skillId | path | uint32 | yes | Skill identifier |

#### Request Headers

| Header | Required | Description |
|--------|----------|-------------|
| TENANT_ID | yes | Tenant UUID |

#### Request Model

Resource type: `skills`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| level | byte | yes | Skill level |
| masterLevel | byte | yes | Master level |
| expiration | time.Time | yes | Expiration timestamp |

#### Response

| Status | Description |
|--------|-------------|
| 202 | Command accepted |

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 500 | Internal error sending command |

---

### GET /api/characters/{characterId}/macros

Returns all macros for a character.

#### Parameters

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | yes | Character identifier |

#### Request Headers

| Header | Required | Description |
|--------|----------|-------------|
| TENANT_ID | yes | Tenant UUID |

#### Response Model

Resource type: `macros`

| Field | Type | Description |
|-------|------|-------------|
| id | string | Macro identifier |
| name | string | Macro name |
| shout | bool | Shout flag |
| skillId1 | uint32 | First skill ID |
| skillId2 | uint32 | Second skill ID |
| skillId3 | uint32 | Third skill ID |

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 500 | Internal error retrieving macros |
