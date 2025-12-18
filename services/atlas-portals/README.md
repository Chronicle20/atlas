# atlas-portals
Mushroom game portals Service

## Overview

A RESTful resource which provides portals services.

## Environment

- JAEGER_HOST - Jaeger [host]:[port]
- LOG_LEVEL - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace
- BOOTSTRAP_SERVERS - Kafka [host]:[port]
- GAME_DATA_SERVICE_URL - [scheme]://[host]:[port]/api/gis/
- COMMAND_TOPIC_PORTAL - Kafka Topic for transmitting portal commands.
- EVENT_TOPIC_CHARACTER_STATUS - Kafka Topic for transmitting character status events
