# atlas-buddies
Mushroom game buddies Service

## Overview

A RESTful resource which provides buddies services.

## Environment

- JAEGER_HOST - Jaeger [host]:[port]
- LOG_LEVEL - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace
- DB_USER - Postgres user name
- DB_PASSWORD - Postgres user password
- DB_HOST - Postgres Database host
- DB_PORT - Postgres Database port
- DB_NAME - Postgres Database name
- BOOTSTRAP_SERVERS - Kafka [host]:[port]
- COMMAND_TOPIC_BUDDY_LIST - Kafka Topic for transmitting buddy list commands.

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

#### [GET] Get Characters Buddy List

```/api/characters/{characterId}/buddy-list```

#### [POST] Create Characters Buddy List

```/api/characters/{characterId}/buddy-list```
