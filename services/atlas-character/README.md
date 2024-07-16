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
- GAME_DATA_SERVICE_URL - [scheme]://[host]:[port]/api/gis/
- EQUIPABLE_SERVICE_URL - [scheme]://[host]:[port]/api/ess/
- BOOTSTRAP_SERVERS - Kafka [host]:[port]
- COMMAND_TOPIC_EQUIP_ITEM - Kafka Topic for transmitting equip item commands
- COMMAND_TOPIC_UNEQUIP_ITEM - Kafka Topic for transmitting unequip item commands
- EVENT_TOPIC_CHARACTER_STATUS - Kafka Topic for transmitting character status events
- EVENT_TOPIC_ITEM_GAIN - Kafka Topic for transmitting item gain events
- EVENT_TOPIC_EQUIP_CHANGED - Kafka Topic for transmitting equip changed events
- EVENT_TOPIC_SESSION_STATUS - Kafka Topic for capturing session events

## API

### Header

All RESTful requests require the supplied header information to identify the server instance.

```
TENANT_ID:083839c6-c47c-42a6-9585-76492795d123
REGION:GMS
MAJOR_VERSION:83
MINOR_VERSION:1
```

### Requests

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

#### [POST] Create Item

```/api/cos/characters/{characterId}/inventories/{inventoryType}/items```

#### [POST] Equip Item

```/api/cos/characters/{characterId}/equipment/{slotType}/equipable```

#### [DELETE] Unequip Item

```/api/cos/characters/{characterId}/equipment/{slotType}/equipable```