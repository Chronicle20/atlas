# atlas-monsters

Mushroom game monsters Service

## Overview

A RESTful resource which provides monsters services.

## Environment

- JAEGER_HOST - Jaeger [host]:[port]
- LOG_LEVEL - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace
- BOOTSTRAP_SERVERS - Kafka [host]:[port]
- BASE_SERVICE_URL - [scheme]://[host]:[port]/api/
- EVENT_TOPIC_MAP_STATUS - Kafka Topic for transmitting Map Status events.
- EVENT_TOPIC_MONSTER_STATUS - Kafka Topic for transmitting Monster Status events.
- EVENT_TOPIC_MONSTER_MOVEMENT - Kafka Topic for transmitting Monster Movement events.
- COMMAND_TOPIC_MONSTER_DAMAGE - Kafka Topic for issuing Monster Damage commands.
- COMMAND_TOPIC_MONSTER_MOVEMENT - Kafka Topic for issuing Monster Movement commands.

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
