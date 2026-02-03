# Quest REST API

## Endpoints

### GET /api/characters/{characterId}/quests

Retrieves all quest statuses for a character.

#### Parameters

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | Yes | Character identifier |

#### Request Headers

| Header | Required | Description |
|--------|----------|-------------|
| TENANT_ID | Yes | Tenant identifier (UUID) |
| REGION | Yes | Region code |
| MAJOR_VERSION | Yes | Major version |
| MINOR_VERSION | Yes | Minor version |

#### Response Model

Array of `quest-status` resources.

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 500 | Database error |

---

### GET /api/characters/{characterId}/quests/started

Retrieves all started quests for a character.

#### Parameters

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | Yes | Character identifier |

#### Response Model

Array of `quest-status` resources with state = 1.

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 500 | Database error |

---

### GET /api/characters/{characterId}/quests/completed

Retrieves all completed quests for a character.

#### Parameters

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | Yes | Character identifier |

#### Response Model

Array of `quest-status` resources with state = 2.

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 500 | Database error |

---

### GET /api/characters/{characterId}/quests/{questId}

Retrieves a specific quest status for a character.

#### Parameters

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | Yes | Character identifier |
| questId | path | uint32 | Yes | Quest identifier |

#### Response Model

Single `quest-status` resource.

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 404 | Quest not found |
| 500 | Database error |

---

### POST /api/characters/{characterId}/quests/{questId}/start

Starts a quest for a character.

#### Parameters

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | Yes | Character identifier |
| questId | path | uint32 | Yes | Quest identifier |

#### Request Model

`start-quest-input`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| worldId | byte | No | World identifier |
| channelId | byte | No | Channel identifier |
| mapId | uint32 | No | Map identifier |
| skipValidation | bool | No | Skip requirement validation (default: false) |

#### Response Model

`quest-status` resource.

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Interval not elapsed, other errors |
| 409 | Quest already started or completed |
| 422 | Start requirements not met (returns `validation-failed`) |
| 500 | Database error |

---

### POST /api/characters/{characterId}/quests/{questId}/complete

Completes a quest for a character.

#### Parameters

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | Yes | Character identifier |
| questId | path | uint32 | Yes | Quest identifier |

#### Request Model

`complete-quest-input`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| worldId | byte | No | World identifier |
| channelId | byte | No | Channel identifier |
| mapId | uint32 | No | Map identifier |
| skipValidation | bool | No | Skip requirement validation (default: false) |

#### Response Model

Returns 204 No Content on success, or `complete-quest-response` if quest has a chain.

`complete-quest-response`

| Field | Type | Description |
|-------|------|-------------|
| nextQuestId | uint32 | Next quest in chain |

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Other errors |
| 404 | Quest not found |
| 409 | Quest not in started state |
| 410 | Quest expired |
| 422 | End requirements not met |
| 500 | Database error |

---

### POST /api/characters/{characterId}/quests/{questId}/forfeit

Forfeits a quest for a character.

#### Parameters

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | Yes | Character identifier |
| questId | path | uint32 | Yes | Quest identifier |

#### Response Model

Returns 204 No Content.

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Quest not in started state |
| 404 | Quest not found |
| 500 | Database error |

---

### GET /api/characters/{characterId}/quests/{questId}/progress

Retrieves progress entries for a quest.

#### Parameters

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | Yes | Character identifier |
| questId | path | uint32 | Yes | Quest identifier |

#### Response Model

Array of `progress` resources.

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 404 | Quest not found |
| 500 | Database error |

---

### PATCH /api/characters/{characterId}/quests/{questId}/progress

Updates progress for a quest objective.

#### Parameters

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | Yes | Character identifier |
| questId | path | uint32 | Yes | Quest identifier |

#### Request Model

`progress`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| infoNumber | uint32 | Yes | Objective identifier |
| progress | string | Yes | Progress value |

#### Response Model

Returns 204 No Content.

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Quest not in started state |
| 404 | Quest not found |
| 500 | Database error |

---

### DELETE /api/characters/{characterId}/quests

Deletes all quest data for a character.

#### Parameters

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | Yes | Character identifier |

#### Response Model

Returns 204 No Content.

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 500 | Database error |

---

## Resource Types

### quest-status

| Field | Type | Description |
|-------|------|-------------|
| characterId | uint32 | Character identifier |
| questId | uint32 | Quest identifier |
| state | byte | Quest state (0=not started, 1=started, 2=completed) |
| startedAt | time.Time | Start timestamp |
| completedAt | time.Time | Completion timestamp (optional) |
| expirationTime | time.Time | Expiration timestamp (optional) |
| completedCount | uint32 | Times completed |
| forfeitCount | uint32 | Times forfeited |
| progress | []progress | Progress entries |

### progress

| Field | Type | Description |
|-------|------|-------------|
| infoNumber | uint32 | Objective identifier |
| progress | string | Progress value |

### start-quest-input

| Field | Type | Description |
|-------|------|-------------|
| worldId | byte | World identifier |
| channelId | byte | Channel identifier |
| mapId | uint32 | Map identifier |
| skipValidation | bool | Skip requirement validation |

### complete-quest-input

| Field | Type | Description |
|-------|------|-------------|
| worldId | byte | World identifier |
| channelId | byte | Channel identifier |
| mapId | uint32 | Map identifier |
| skipValidation | bool | Skip requirement validation |

### complete-quest-response

| Field | Type | Description |
|-------|------|-------------|
| nextQuestId | uint32 | Next quest in chain |

### validation-failed

| Field | Type | Description |
|-------|------|-------------|
| failedConditions | []string | List of failed requirement types |
