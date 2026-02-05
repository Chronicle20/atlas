# Reactor REST API

## Endpoints

### GET /api/reactors/{reactorId}

Retrieves a reactor by its unique ID.

**Parameters:**

| Name      | In   | Type   | Required | Description             |
|-----------|------|--------|----------|-------------------------|
| reactorId | path | uint32 | Yes      | Unique reactor ID       |

**Request Headers:**

| Name          | Required | Description           |
|---------------|----------|-----------------------|
| TENANT_ID     | Yes      | Tenant identifier     |
| REGION        | Yes      | Region code           |
| MAJOR_VERSION | Yes      | Major version number  |
| MINOR_VERSION | Yes      | Minor version number  |

**Response Model:**

JSON:API document with type `reactors`.

| Attribute      | Type   | Description                    |
|----------------|--------|--------------------------------|
| worldId        | byte   | World identifier               |
| channelId      | byte   | Channel identifier             |
| mapId          | uint32 | Map identifier                 |
| classification | uint32 | Reactor type/classification ID |
| name           | string | Reactor name                   |
| state          | int8   | Current reactor state          |
| eventState     | byte   | Event state                    |
| x              | int16  | X coordinate position          |
| y              | int16  | Y coordinate position          |
| delay          | uint32 | Respawn delay in milliseconds  |
| direction      | byte   | Facing direction               |

**Error Conditions:**

| Code | Condition          |
|------|--------------------|
| 404  | Reactor not found  |

---

### GET /api/worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/reactors

Retrieves all reactors in a specific world/channel/map/instance.

**Parameters:**

| Name       | In   | Type   | Required | Description         |
|------------|------|--------|----------|---------------------|
| worldId    | path | byte   | Yes      | World identifier    |
| channelId  | path | byte   | Yes      | Channel identifier  |
| mapId      | path | uint32 | Yes      | Map identifier      |
| instanceId | path | uuid   | Yes      | Instance identifier |

**Request Headers:**

| Name          | Required | Description           |
|---------------|----------|-----------------------|
| TENANT_ID     | Yes      | Tenant identifier     |
| REGION        | Yes      | Region code           |
| MAJOR_VERSION | Yes      | Major version number  |
| MINOR_VERSION | Yes      | Minor version number  |

**Response Model:**

JSON:API document with array of type `reactors`.

**Error Conditions:**

| Code | Condition             |
|------|-----------------------|
| 500  | Internal server error |

---

### GET /api/worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/reactors/{reactorId}

Retrieves a specific reactor within a map/instance context.

**Parameters:**

| Name       | In   | Type   | Required | Description         |
|------------|------|--------|----------|---------------------|
| worldId    | path | byte   | Yes      | World identifier    |
| channelId  | path | byte   | Yes      | Channel identifier  |
| mapId      | path | uint32 | Yes      | Map identifier      |
| instanceId | path | uuid   | Yes      | Instance identifier |
| reactorId  | path | uint32 | Yes      | Unique reactor ID   |

**Request Headers:**

| Name          | Required | Description           |
|---------------|----------|-----------------------|
| TENANT_ID     | Yes      | Tenant identifier     |
| REGION        | Yes      | Region code           |
| MAJOR_VERSION | Yes      | Major version number  |
| MINOR_VERSION | Yes      | Minor version number  |

**Response Model:**

JSON:API document with type `reactors`.

**Error Conditions:**

| Code | Condition                              |
|------|----------------------------------------|
| 404  | Reactor not found or not in specified map |

---

### POST /api/worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/reactors

Creates a new reactor in the specified map/instance. Request is processed asynchronously via Kafka.

**Parameters:**

| Name       | In   | Type   | Required | Description         |
|------------|------|--------|----------|---------------------|
| worldId    | path | byte   | Yes      | World identifier    |
| channelId  | path | byte   | Yes      | Channel identifier  |
| mapId      | path | uint32 | Yes      | Map identifier      |
| instanceId | path | uuid   | Yes      | Instance identifier |

**Request Headers:**

| Name          | Required | Description           |
|---------------|----------|-----------------------|
| TENANT_ID     | Yes      | Tenant identifier     |
| REGION        | Yes      | Region code           |
| MAJOR_VERSION | Yes      | Major version number  |
| MINOR_VERSION | Yes      | Minor version number  |

**Request Model:**

JSON:API document with type `reactors`.

| Attribute      | Type   | Required | Description                    |
|----------------|--------|----------|--------------------------------|
| classification | uint32 | Yes      | Reactor type/classification ID |
| name           | string | Yes      | Reactor name                   |
| state          | int8   | Yes      | Initial reactor state          |
| x              | int16  | Yes      | X coordinate position          |
| y              | int16  | Yes      | Y coordinate position          |
| delay          | uint32 | No       | Respawn delay in milliseconds  |
| direction      | byte   | No       | Facing direction               |

**Error Conditions:**

| Code | Condition                             |
|------|---------------------------------------|
| 202  | Accepted - request queued             |
| 500  | Internal server error                 |
