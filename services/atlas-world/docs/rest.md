# REST

## Endpoints

### GET /api/worlds/

Returns all worlds for the tenant.

#### Parameters

| Name | In | Type | Required | Description |
|------|-----|------|----------|-------------|
| include | query | string | No | Include related resources (channels) |

#### Request Headers

| Name | Required | Description |
|------|----------|-------------|
| TENANT_ID | Yes | Tenant identifier UUID |
| REGION | Yes | Region code |
| MAJOR_VERSION | Yes | Major version number |
| MINOR_VERSION | Yes | Minor version number |

#### Response Model

Type: `worlds`

| Field | Type | Description |
|-------|------|-------------|
| id | string | World identifier |
| name | string | World name |
| state | byte | World state flag |
| message | string | Server message |
| eventMessage | string | Event message |
| recommended | bool | Whether world is recommended |
| recommendedMessage | string | Recommendation message |
| capacityStatus | uint16 | Capacity status |

Relationships:
- channels (to-many): Associated channel resources

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 500 | Internal server error |

---

### GET /api/worlds/{worldId}

Returns a specific world.

#### Parameters

| Name | In | Type | Required | Description |
|------|-----|------|----------|-------------|
| worldId | path | byte | Yes | World identifier |
| include | query | string | No | Include related resources (channels) |

#### Request Headers

| Name | Required | Description |
|------|----------|-------------|
| TENANT_ID | Yes | Tenant identifier UUID |
| REGION | Yes | Region code |
| MAJOR_VERSION | Yes | Major version number |
| MINOR_VERSION | Yes | Minor version number |

#### Response Model

Type: `worlds`

| Field | Type | Description |
|-------|------|-------------|
| id | string | World identifier |
| name | string | World name |
| state | byte | World state flag |
| message | string | Server message |
| eventMessage | string | Event message |
| recommended | bool | Whether world is recommended |
| recommendedMessage | string | Recommendation message |
| capacityStatus | uint16 | Capacity status |

Relationships:
- channels (to-many): Associated channel resources

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 404 | World not found |
| 500 | Internal server error |

---

### GET /api/worlds/{worldId}/channels

Returns all channels for a world.

#### Parameters

| Name | In | Type | Required | Description |
|------|-----|------|----------|-------------|
| worldId | path | byte | Yes | World identifier |

#### Request Headers

| Name | Required | Description |
|------|----------|-------------|
| TENANT_ID | Yes | Tenant identifier UUID |
| REGION | Yes | Region code |
| MAJOR_VERSION | Yes | Major version number |
| MINOR_VERSION | Yes | Minor version number |

#### Response Model

Type: `channels`

| Field | Type | Description |
|-------|------|-------------|
| id | uuid | Channel unique identifier |
| worldId | byte | World identifier |
| channelId | byte | Channel identifier |
| ipAddress | string | Server IP address |
| port | int | Server port |
| currentCapacity | uint32 | Current player count |
| maxCapacity | uint32 | Maximum player capacity |
| createdAt | time | Registration timestamp |

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 500 | Internal server error |

---

### GET /api/worlds/{worldId}/channels/{channelId}

Returns a specific channel.

#### Parameters

| Name | In | Type | Required | Description |
|------|-----|------|----------|-------------|
| worldId | path | byte | Yes | World identifier |
| channelId | path | byte | Yes | Channel identifier |

#### Request Headers

| Name | Required | Description |
|------|----------|-------------|
| TENANT_ID | Yes | Tenant identifier UUID |
| REGION | Yes | Region code |
| MAJOR_VERSION | Yes | Major version number |
| MINOR_VERSION | Yes | Minor version number |

#### Response Model

Type: `channels`

| Field | Type | Description |
|-------|------|-------------|
| id | uuid | Channel unique identifier |
| worldId | byte | World identifier |
| channelId | byte | Channel identifier |
| ipAddress | string | Server IP address |
| port | int | Server port |
| currentCapacity | uint32 | Current player count |
| maxCapacity | uint32 | Maximum player capacity |
| createdAt | time | Registration timestamp |

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 404 | Channel not found |
| 500 | Internal server error |

---

### POST /api/worlds/{worldId}/channels

Registers a new channel server by emitting a started event.

#### Parameters

| Name | In | Type | Required | Description |
|------|-----|------|----------|-------------|
| worldId | path | byte | Yes | World identifier |

#### Request Headers

| Name | Required | Description |
|------|----------|-------------|
| TENANT_ID | Yes | Tenant identifier UUID |
| REGION | Yes | Region code |
| MAJOR_VERSION | Yes | Major version number |
| MINOR_VERSION | Yes | Minor version number |

#### Request Model

Type: `channels`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| channelId | byte | Yes | Channel identifier |
| ipAddress | string | Yes | Server IP address |
| port | int | Yes | Server port |
| currentCapacity | uint32 | Yes | Current player count |
| maxCapacity | uint32 | Yes | Maximum player capacity |

#### Response

| Status | Description |
|--------|-------------|
| 202 | Accepted |

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 500 | Internal server error |
