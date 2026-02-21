# Kafka

## Topics Consumed

| Topic | Environment Variable | Description |
|-------|---------------------|-------------|
| Portal Commands | `COMMAND_TOPIC_PORTAL` | Portal enter, warp, block, and unblock commands |
| Character Status Events | `EVENT_TOPIC_CHARACTER_STATUS` | Character status events (logout) |

## Topics Produced

| Topic | Environment Variable | Description |
|-------|---------------------|-------------|
| Character Status Events | `EVENT_TOPIC_CHARACTER_STATUS` | Enable actions events |
| Character Commands | `COMMAND_TOPIC_CHARACTER` | Change map commands |
| Portal Actions Commands | `COMMAND_TOPIC_PORTAL_ACTIONS` | Script execution commands |

## Message Types

### Consumed

**commandEvent[enterBody]** (Portal Commands)

| Field | Type | Description |
|-------|------|-------------|
| worldId | byte | World identifier |
| channelId | byte | Channel identifier |
| mapId | uint32 | Map identifier |
| instance | uuid.UUID | Map instance identifier |
| portalId | uint32 | Portal identifier |
| type | string | `ENTER` |
| body.characterId | uint32 | Character identifier |

**warpEvent** (Portal Commands)

| Field | Type | Description |
|-------|------|-------------|
| worldId | byte | World identifier |
| channelId | byte | Channel identifier |
| mapId | uint32 | Map identifier |
| instance | uuid.UUID | Map instance identifier |
| type | string | `WARP` |
| body.characterId | uint32 | Character identifier |
| body.targetMapId | uint32 | Target map identifier |

**commandEvent[blockBody]** (Portal Commands)

| Field | Type | Description |
|-------|------|-------------|
| worldId | byte | World identifier |
| channelId | byte | Channel identifier |
| mapId | uint32 | Map identifier |
| instance | uuid.UUID | Map instance identifier |
| portalId | uint32 | Portal identifier |
| type | string | `BLOCK` |
| body.characterId | uint32 | Character identifier |

**commandEvent[unblockBody]** (Portal Commands)

| Field | Type | Description |
|-------|------|-------------|
| worldId | byte | World identifier |
| channelId | byte | Channel identifier |
| mapId | uint32 | Map identifier |
| instance | uuid.UUID | Map instance identifier |
| portalId | uint32 | Portal identifier |
| type | string | `UNBLOCK` |
| body.characterId | uint32 | Character identifier |

**StatusEvent[StatusEventLogoutBody]** (Character Status Events)

| Field | Type | Description |
|-------|------|-------------|
| transactionId | uuid.UUID | Transaction identifier |
| worldId | byte | World identifier |
| characterId | uint32 | Character identifier |
| type | string | `LOGOUT` |
| body.channelId | byte | Channel identifier |
| body.mapId | uint32 | Map identifier |
| body.instance | uuid.UUID | Map instance identifier |

### Produced

**statusEvent[statusEventStatChangedBody]** (Character Status Events)

| Field | Type | Description |
|-------|------|-------------|
| transactionId | uuid.UUID | Transaction identifier |
| worldId | byte | World identifier |
| characterId | uint32 | Character identifier |
| type | string | `STAT_CHANGED` |
| body.channelId | byte | Channel identifier |
| body.exclRequestSent | bool | Exclusive request sent flag |

**commandEvent[changeMapBody]** (Character Commands)

| Field | Type | Description |
|-------|------|-------------|
| worldId | byte | World identifier |
| characterId | uint32 | Character identifier |
| type | string | `CHANGE_MAP` |
| body.channelId | byte | Channel identifier |
| body.mapId | uint32 | Target map identifier |
| body.instance | uuid.UUID | Map instance identifier |
| body.portalId | uint32 | Target portal identifier |

**commandEvent[enterBody]** (Portal Actions Commands)

| Field | Type | Description |
|-------|------|-------------|
| worldId | byte | World identifier |
| channelId | byte | Channel identifier |
| mapId | uint32 | Map identifier |
| instance | uuid.UUID | Map instance identifier |
| portalId | uint32 | Portal identifier |
| type | string | `ENTER` |
| body.characterId | uint32 | Character identifier |
| body.portalName | string | Portal script name |

## Transaction Semantics

- Portal commands require `Tenant-Id` header.
- Character status events require `Tenant-Id` header.
- Messages are keyed by character id.
