# REST

## Endpoints

### GET /api/worlds/

Returns all worlds for the tenant.

#### Parameters

| Name | In | Type | Required | Description |
|------|-----|------|----------|-------------|
| include | query | string | No | Include related resources (channels) |
| page[number] | query | int | No | Page number (default 1) |
| page[size] | query | int | No | Page size (default 50, max 250) |

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
| expRate | float64 | Experience rate multiplier |
| mesoRate | float64 | Meso rate multiplier |
| itemDropRate | float64 | Item drop rate multiplier |
| questExpRate | float64 | Quest experience rate multiplier |

Relationships:
- channels (to-many): Associated channel resources

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Invalid page[number]/page[size] |
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
| expRate | float64 | Experience rate multiplier |
| mesoRate | float64 | Meso rate multiplier |
| itemDropRate | float64 | Item drop rate multiplier |
| questExpRate | float64 | Quest experience rate multiplier |

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
| page[number] | query | int | No | Page number (default 1) |
| page[size] | query | int | No | Page size (default 50, max 250) |

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
| expRate | float64 | Experience rate multiplier |
| mesoRate | float64 | Meso rate multiplier |
| itemDropRate | float64 | Item drop rate multiplier |
| questExpRate | float64 | Quest experience rate multiplier |

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Invalid page[number]/page[size] |
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
| expRate | float64 | Experience rate multiplier |
| mesoRate | float64 | Meso rate multiplier |
| itemDropRate | float64 | Item drop rate multiplier |
| questExpRate | float64 | Quest experience rate multiplier |

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 404 | Channel not found |
| 500 | Internal server error |

---

### DELETE /api/worlds/{worldId}/channels/{channelId}

Unregisters a channel server.

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

#### Response

| Status | Description |
|--------|-------------|
| 204 | No Content |

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

---

### GET /api/worlds/{worldId}/rates

Returns current rate multipliers for a world.

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

Type: `rates`

| Field | Type | Description |
|-------|------|-------------|
| id | string | Rate identifier (format: world-{worldId}) |
| expRate | float64 | Experience rate multiplier |
| mesoRate | float64 | Meso rate multiplier |
| itemDropRate | float64 | Item drop rate multiplier |
| questExpRate | float64 | Quest experience rate multiplier |

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 500 | Internal server error |

---

### PATCH /api/worlds/{worldId}/rates

Updates a rate multiplier for a world.

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

Type: `rates`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| rateType | string | Yes | Rate type (exp, meso, item_drop, quest_exp) |
| multiplier | float64 | Yes | New rate multiplier value |

#### Response

| Status | Description |
|--------|-------------|
| 204 | No Content |

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Invalid rate type |
| 500 | Internal server error |

---

### GET /api/worlds/{worldId}/broadcast-queues/{family}

Returns the current broadcast queue state for a (world, family) pair.

#### Parameters

| Name | In | Type | Required | Description |
|------|-----|------|----------|-------------|
| worldId | path | byte | Yes | World identifier |
| family | path | string | Yes | Broadcast family (TV or AVATAR) |

#### Request Headers

| Name | Required | Description |
|------|----------|-------------|
| TENANT_ID | Yes | Tenant identifier UUID |
| REGION | Yes | Region code |
| MAJOR_VERSION | Yes | Major version number |
| MINOR_VERSION | Yes | Minor version number |

#### Response Model

Type: `broadcast-queues`

| Field | Type | Description |
|-------|------|-------------|
| id | string | Family (TV or AVATAR) |
| family | string | Broadcast family |
| activeRemainingSeconds | uint32 | Time remaining on the active entry, in seconds (0 if idle or expired but not yet swept) |
| pendingCount | int | Number of entries waiting behind the active entry |
| waitSeconds | uint32 | Estimated wait, in seconds, a newly-enqueued entry would be given |

If no queue has been created yet for the (world, family) pair, the response represents an idle queue (activeRemainingSeconds 0, pendingCount 0, waitSeconds 0).

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | family is not TV or AVATAR |
| 500 | Internal server error |
