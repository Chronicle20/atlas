# atlas-inventory
Mushroom game inventory Service

## Overview

A RESTful resource which provides inventory services.

## Environment

- JAEGER_HOST - Jaeger [host]:[port]
- LOG_LEVEL - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace

### Kafka Topics

- EVENT_TOPIC_ASSET_STATUS - Topic for asset status events (created, deleted, moved, quantity changed)
- EVENT_TOPIC_COMPARTMENT_STATUS - Topic for compartment status events (created, deleted, capacity changed, reserved, reservation cancelled)
- COMMAND_TOPIC_COMPARTMENT - Topic for compartment commands (equip, unequip, move, drop, request reserve, consume, destroy, etc.)
- EVENT_TOPIC_CHARACTER_STATUS - Topic for character status events (created, deleted)
- COMMAND_TOPIC_DROP - Topic for drop commands (spawn from character)
- EVENT_TOPIC_INVENTORY_STATUS - Topic for inventory status events (created, deleted)

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
