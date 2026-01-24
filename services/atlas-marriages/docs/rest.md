# Marriage REST API

## Endpoints

### GET /api/characters/{characterId}/marriage

Returns the current marriage information for a character.

**Parameters**

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | yes | Character identifier |

**Request Headers**

| Header | Required | Description |
|--------|----------|-------------|
| Content-Type | yes | application/json |
| TENANT_ID | yes | Tenant UUID |
| REGION | yes | Region identifier |
| MAJOR_VERSION | yes | Major version number |
| MINOR_VERSION | yes | Minor version number |

**Response Model**

JSON:API resource type: `marriage`

```go
type RestMarriage struct {
    ID           uint32        `json:"id"`
    CharacterId1 uint32        `json:"characterId1"`
    CharacterId2 uint32        `json:"characterId2"`
    Status       string        `json:"status"`
    ProposedAt   time.Time     `json:"proposedAt"`
    EngagedAt    *time.Time    `json:"engagedAt,omitempty"`
    MarriedAt    *time.Time    `json:"marriedAt,omitempty"`
    DivorcedAt   *time.Time    `json:"divorcedAt,omitempty"`
    CreatedAt    time.Time     `json:"createdAt"`
    UpdatedAt    time.Time     `json:"updatedAt"`
    Partner      *RestPartner  `json:"partner,omitempty"`
    Ceremony     *RestCeremony `json:"ceremony,omitempty"`
}

type RestPartner struct {
    CharacterID uint32 `json:"characterId"`
}

type RestCeremony struct {
    ID           uint32     `json:"id"`
    Status       string     `json:"status"`
    ScheduledAt  time.Time  `json:"scheduledAt"`
    StartedAt    *time.Time `json:"startedAt,omitempty"`
    CompletedAt  *time.Time `json:"completedAt,omitempty"`
    CancelledAt  *time.Time `json:"cancelledAt,omitempty"`
    PostponedAt  *time.Time `json:"postponedAt,omitempty"`
    InviteeCount int        `json:"inviteeCount"`
}
```

**Error Conditions**

| Status | Title | Description |
|--------|-------|-------------|
| 404 | Not Found | Character is not married |
| 500 | Internal Server Error | Failed to retrieve or transform data |

---

### GET /api/characters/{characterId}/marriage/history

Returns the complete marriage history for a character.

**Parameters**

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | yes | Character identifier |

**Request Headers**

| Header | Required | Description |
|--------|----------|-------------|
| Content-Type | yes | application/json |
| TENANT_ID | yes | Tenant UUID |
| REGION | yes | Region identifier |
| MAJOR_VERSION | yes | Major version number |
| MINOR_VERSION | yes | Minor version number |

**Response Model**

JSON:API resource type: `marriage` (array)

```go
[]RestMarriage
```

**Error Conditions**

| Status | Title | Description |
|--------|-------|-------------|
| 500 | Internal Server Error | Failed to retrieve or transform data |

---

### GET /api/characters/{characterId}/marriage/proposals

Returns all pending proposals for a character (both sent and received).

**Parameters**

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | yes | Character identifier |

**Request Headers**

| Header | Required | Description |
|--------|----------|-------------|
| Content-Type | yes | application/json |
| TENANT_ID | yes | Tenant UUID |
| REGION | yes | Region identifier |
| MAJOR_VERSION | yes | Major version number |
| MINOR_VERSION | yes | Minor version number |

**Response Model**

JSON:API resource type: `proposal` (array)

```go
type RestProposal struct {
    ID             uint32     `json:"id"`
    ProposerID     uint32     `json:"proposerId"`
    TargetID       uint32     `json:"targetId"`
    Status         string     `json:"status"`
    ProposedAt     time.Time  `json:"proposedAt"`
    RespondedAt    *time.Time `json:"respondedAt,omitempty"`
    ExpiresAt      time.Time  `json:"expiresAt"`
    RejectionCount uint32     `json:"rejectionCount"`
    CooldownUntil  *time.Time `json:"cooldownUntil,omitempty"`
    CreatedAt      time.Time  `json:"createdAt"`
    UpdatedAt      time.Time  `json:"updatedAt"`
}
```

**Error Conditions**

| Status | Title | Description |
|--------|-------|-------------|
| 500 | Internal Server Error | Failed to retrieve or transform data |
