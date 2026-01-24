# Reactor Kafka Integration

## Topics Consumed

### COMMAND_TOPIC_REACTOR

Environment variable: `COMMAND_TOPIC_REACTOR`

Receives commands for reactor operations.

## Topics Produced

### EVENT_TOPIC_REACTOR_STATUS

Environment variable: `EVENT_TOPIC_REACTOR_STATUS`

Emits reactor status events.

### COMMAND_TOPIC_REACTOR_ACTIONS

Environment variable: `COMMAND_TOPIC_REACTOR_ACTIONS`

Emits commands to atlas-reactor-actions for script execution.

## Message Types

### Commands (Consumed)

#### CREATE Command

| Field     | Type   | Description        |
|-----------|--------|--------------------|
| worldId   | byte   | World identifier   |
| channelId | byte   | Channel identifier |
| mapId     | uint32 | Map identifier     |
| type      | string | "CREATE"           |
| body      | object | CreateCommandBody  |

**CreateCommandBody:**

| Field          | Type   | Description                    |
|----------------|--------|--------------------------------|
| classification | uint32 | Reactor type/classification ID |
| name           | string | Reactor name                   |
| state          | int8   | Initial reactor state          |
| x              | int16  | X coordinate position          |
| y              | int16  | Y coordinate position          |
| delay          | uint32 | Respawn delay in milliseconds  |
| direction      | byte   | Facing direction               |

#### HIT Command

| Field     | Type   | Description       |
|-----------|--------|-------------------|
| worldId   | byte   | World identifier  |
| channelId | byte   | Channel identifier|
| mapId     | uint32 | Map identifier    |
| type      | string | "HIT"             |
| body      | object | HitCommandBody    |

**HitCommandBody:**

| Field       | Type   | Description              |
|-------------|--------|--------------------------|
| reactorId   | uint32 | Reactor instance ID      |
| characterId | uint32 | Character performing hit |
| stance      | uint16 | Character stance         |
| skillId     | uint32 | Skill used (0 if none)   |

### Events (Produced)

#### CREATED Event

| Field     | Type   | Description            |
|-----------|--------|------------------------|
| worldId   | byte   | World identifier       |
| channelId | byte   | Channel identifier     |
| mapId     | uint32 | Map identifier         |
| reactorId | uint32 | Reactor instance ID    |
| type      | string | "CREATED"              |
| body      | object | createdStatusEventBody |

**createdStatusEventBody:**

| Field          | Type      | Description                    |
|----------------|-----------|--------------------------------|
| classification | uint32    | Reactor type/classification ID |
| name           | string    | Reactor name                   |
| state          | int8      | Reactor state                  |
| eventState     | byte      | Event state                    |
| delay          | uint32    | Respawn delay in milliseconds  |
| direction      | byte      | Facing direction               |
| x              | int16     | X coordinate position          |
| y              | int16     | Y coordinate position          |
| updateTime     | time.Time | Creation timestamp             |

#### DESTROYED Event

| Field     | Type   | Description              |
|-----------|--------|--------------------------|
| worldId   | byte   | World identifier         |
| channelId | byte   | Channel identifier       |
| mapId     | uint32 | Map identifier           |
| reactorId | uint32 | Reactor instance ID      |
| type      | string | "DESTROYED"              |
| body      | object | destroyedStatusEventBody |

**destroyedStatusEventBody:**

| Field | Type  | Description           |
|-------|-------|-----------------------|
| state | int8  | Final reactor state   |
| x     | int16 | X coordinate position |
| y     | int16 | Y coordinate position |

#### HIT Event

| Field     | Type   | Description        |
|-----------|--------|--------------------|
| worldId   | byte   | World identifier   |
| channelId | byte   | Channel identifier |
| mapId     | uint32 | Map identifier     |
| reactorId | uint32 | Reactor instance ID|
| type      | string | "HIT"              |
| body      | object | hitStatusEventBody |

**hitStatusEventBody:**

| Field          | Type   | Description                    |
|----------------|--------|--------------------------------|
| classification | uint32 | Reactor type/classification ID |
| state          | int8   | Current reactor state          |
| x              | int16  | X coordinate position          |
| y              | int16  | Y coordinate position          |
| direction      | byte   | Facing direction               |
| destroyed      | bool   | Whether reactor was destroyed  |

### Commands (Produced to atlas-reactor-actions)

#### HIT Command

| Field          | Type   | Description                    |
|----------------|--------|--------------------------------|
| worldId        | byte   | World identifier               |
| channelId      | byte   | Channel identifier             |
| mapId          | uint32 | Map identifier                 |
| reactorId      | uint32 | Reactor instance ID            |
| classification | string | Reactor classification as string |
| reactorName    | string | Reactor name                   |
| reactorState   | int8   | Current reactor state          |
| x              | int16  | X coordinate position          |
| y              | int16  | Y coordinate position          |
| type           | string | "HIT"                          |
| body           | object | hitActionsBody                 |

**hitActionsBody:**

| Field       | Type   | Description              |
|-------------|--------|--------------------------|
| characterId | uint32 | Character performing hit |
| skillId     | uint32 | Skill used               |
| isSkill     | bool   | Whether hit was by skill |

#### TRIGGER Command

| Field          | Type   | Description                    |
|----------------|--------|--------------------------------|
| worldId        | byte   | World identifier               |
| channelId      | byte   | Channel identifier             |
| mapId          | uint32 | Map identifier                 |
| reactorId      | uint32 | Reactor instance ID            |
| classification | string | Reactor classification as string |
| reactorName    | string | Reactor name                   |
| reactorState   | int8   | Current reactor state          |
| x              | int16  | X coordinate position          |
| y              | int16  | Y coordinate position          |
| type           | string | "TRIGGER"                      |
| body           | object | triggerActionsBody             |

**triggerActionsBody:**

| Field       | Type   | Description                    |
|-------------|--------|--------------------------------|
| characterId | uint32 | Character that triggered       |

## Transaction Semantics

- Commands are consumed with tenant header parsing
- Message keys are based on mapId for CREATE commands and reactorId for status events
- Consumer group ID: "Reactors Service"
