# atlas-character
Mushroom game Character Service

## Overview

A RESTful resource which provides character services.

## Environment

- JAEGER_HOST - Jaeger [host]:[port]
- LOG_LEVEL - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace
- DB_USER - Postgres user name
- DB_PASSWORD - Postgres user password
- DB_HOST - Postgres Database host
- DB_PORT - Postgres Database port
- DB_NAME - Postgres Database name
- BASE_SERVICE_URL - [scheme]://[host]:[port]/api/
- BOOTSTRAP_SERVERS - Kafka [host]:[port]
- COMMAND_TOPIC_CHARACTER - Kafka Topic for transmitting character commands
- COMMAND_TOPIC_EQUIP_ITEM - Kafka Topic for transmitting equip item commands
- COMMAND_TOPIC_UNEQUIP_ITEM - Kafka Topic for transmitting unequip item commands
- COMMAND_TOPIC_CHARACTER_MOVEMENT - Kafka Topic for transmitting character movement commands
- COMMAND_TOPIC_DROP - Kafka Topic for transmitting drop commands
- COMMAND_TOPIC_SKILL - Kafka Topic for transmitting skill commands
- COMMAND_TOPIC_INVENTORY - Kafka Topic for transmitting inventory commands
- EVENT_TOPIC_CHARACTER_STATUS - Kafka Topic for transmitting character status events
- EVENT_TOPIC_INVENTORY_CHANGED - Kafka Topic for transmitting inventory change events
- EVENT_TOPIC_SESSION_STATUS - Kafka Topic for capturing session events
- EVENT_TOPIC_CHARACTER_MOVEMENT - Kafka Topic for transmitting character movement events
- EVENT_TOPIC_DROP_STATUS - Kafka Topic for transmitting drop status events
- REST_PORT - Port number for the REST server

## API

### Header

All RESTful requests require the supplied header information to identify the server instance.

```
TENANT_ID:083839c6-c47c-42a6-9585-76492795d123
REGION:GMS
MAJOR_VERSION:83
MINOR_VERSION:1
```

### Character APIs

#### [GET] Get Characters - By Account and World
```/api/cos/characters?accountId={accountId}&worldId={worldId}```

#### [GET] Get Characters - By World and Map
```/api/cos/characters?worldId={worldId}&mapId={mapId}```

#### [GET] Get Characters - By Name
```/api/cos/characters?name={name}```

#### [GET] Get Character - By Id
```/api/cos/characters/{characterId}```

#### [POST] Create Character
```/api/cos/characters```

#### [DELETE] Delete Character
```/api/cos/characters/{characterId}```

### Inventory APIs

#### [GET] Get Item By Slot
```/api/cos/characters/{characterId}/inventories/{inventoryType}/items?slot={slot}```

#### [POST] Create Item
```/api/cos/characters/{characterId}/inventories/{inventoryType}/items```

### Equipment APIs

#### [POST] Equip Item
```/api/cos/characters/{characterId}/equipment/{slotType}/equipable```

#### [DELETE] Unequip Item
```/api/cos/characters/{characterId}/equipment/{slotType}/equipable```

### Response Format

All responses follow the JSON:API specification format. The service supports:
- Resource relationships
- Sparse fieldsets
- Includes for related resources
- Pagination