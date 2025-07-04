# atlas-maps

Mushroom game maps Service

## Overview

A RESTful resource which provides maps services, including character tracking, monster spawning, and reactor management.

## Environment Variables

### General Configuration
- JAEGER_HOST - Jaeger [host]:[port] for distributed tracing
- LOG_LEVEL - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace
- BOOTSTRAP_SERVERS - Kafka [host]:[port]
- BASE_SERVICE_URL - [scheme]://[host]:[port]/api/

### Kafka Topics
- EVENT_TOPIC_CHARACTER_STATUS - Kafka Topic for transmitting character status events
- EVENT_TOPIC_MAP_STATUS - Kafka Topic for transmitting map status events
- EVENT_TOPIC_CASH_SHOP_STATUS - Kafka Topic for transmitting cash shop status events
- COMMAND_TOPIC_REACTOR - Kafka Topic for transmitting reactor commands

## REST API

### Header

All RESTful requests require the supplied header information to identify the server instance.

```
TENANT_ID:083839c6-c47c-42a6-9585-76492795d123
REGION:GMS
MAJOR_VERSION:83
MINOR_VERSION:1
```

### Endpoints

#### Map Characters
- `GET /{worldId}/channels/{channelId}/maps/{mapId}/characters` - Get all characters in a specific map

### Requests

Detailed API documentation is available via Bruno collection.

## Kafka Message API

### Character Status Events
Messages published to `EVENT_TOPIC_CHARACTER_STATUS` with the following types:
- `LOGIN` - Character logged in
- `LOGOUT` - Character logged out
- `CHANNEL_CHANGED` - Character changed channels
- `MAP_CHANGED` - Character changed maps

### Map Status Events
Messages published to `EVENT_TOPIC_MAP_STATUS` with the following types:
- `CHARACTER_ENTER` - Character entered a map
- `CHARACTER_EXIT` - Character exited a map

### Cash Shop Status Events
Messages published to `EVENT_TOPIC_CASH_SHOP_STATUS` with the following types:
- `CHARACTER_ENTER` - Character entered the cash shop
- `CHARACTER_EXIT` - Character exited the cash shop

### Reactor Commands
Messages published to `COMMAND_TOPIC_REACTOR` with the following types:
- `CREATE` - Create a reactor

All Kafka messages include a transaction ID (UUID) to track message flow through the system.
