# atlas-character-factory
Mushroom game character-factory Service

## Overview

A RESTful resource which provides character-factory services.

## Environment

- JAEGER_HOST - Jaeger [host]:[port]
- LOG_LEVEL - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace
- CONFIG_FILE - Location of service configuration file.
- BOOTSTRAP_SERVERS - Kafka [host]:[port]
- EVENT_TOPIC_CHARACTER_STATUS - Kafka Topic for transmitting character status events
- EVENT_TOPIC_ITEM_GAIN - Kafka Topic for transmitting item gain events
- EVENT_TOPIC_EQUIP_CHANGED - Kafka Topic for transmitting equip changed events
- CHARACTER_SERVICE_URL - [scheme]://[host]:[port]/api/cos/
- GAME_DATA_SERVICE_URL - [scheme]://[host]:[port]/api/gis/

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
