# atlas-guilds
Mushroom game guilds Service

## Overview

A RESTful resource which provides guilds services.

## Environment

- JAEGER_HOST - Jaeger [host]:[port]
- LOG_LEVEL - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace
- DB_USER - Postgres user name
- DB_PASSWORD - Postgres user password
- DB_HOST - Postgres Database host
- DB_PORT - Postgres Database port
- DB_NAME - Postgres Database name
- BASE_SERVICE_URL - [scheme]://[host]:[port]/api/
- CHARACTER_SERVICE_URL - [scheme]://[host]:[port]/api/cos/
- BOOTSTRAP_SERVERS - Kafka [host]:[port]
- COMMAND_TOPIC_GUILD - Kafka topic for transmitting Guild commands
- COMMAND_TOPIC_GUILD_THREAD - Kafka topic for transmitting Guild Thread commands
- COMMAND_TOPIC_INVITE - Kafka topic for transmitting Invite commands
- EVENT_TOPIC_CHARACTER_STATUS - Kafka Topic for receiving Character status events
- EVENT_TOPIC_INVITE_STATUS - Kafka Topic for receiving Invite status events
- EVENT_TOPIC_GUILD_STATUS - Kafka Topic for receiving Guild status events
- EVENT_TOPIC_GUILD_THREAD_STATUS - Kafka Topic for receiving Guild Thread status events

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
