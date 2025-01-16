# atlas-keys
Mushroom game keys Service

## Overview

A RESTful resource which provides keys services.

## Environment

- JAEGER_HOST - Jaeger [host]:[port]
- LOG_LEVEL - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace
- DB_USER - Postgres user name
- DB_PASSWORD - Postgres user password
- DB_HOST - Postgres Database host
- DB_PORT - Postgres Database port
- DB_NAME - Postgres Database name
- BOOTSTRAP_SERVERS - Kafka [host]:[port]
- EVENT_TOPIC_CHARACTER_STATUS - Kafka Topic for transmitting character status events

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

#### [GET] Get Keys

```/api/characters/{characterId}/keys```

#### [DELETE] Reset Keys

```/api/characters/{characterId}/keys```

#### [PATCH] Update Keys

```/api/characters/{characterId}/keys/{keyId}```