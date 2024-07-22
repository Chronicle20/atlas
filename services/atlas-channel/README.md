# atlas-channel
Mushroom game Channel Service

## Overview

A RESTful resource which provides Channel services.

## Environment

- JAEGER_HOST - Jaeger [host]:[port]
- LOG_LEVEL - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace
- CONFIG_FILE - Location of service configuration file.
- BOOTSTRAP_SERVERS - Kafka [host]:[port]
- ACCOUNT_SERVICE_URL - [scheme]://[host]:[port]/api/aos/
- CHARACTER_SERVICE_URL - [scheme]://[host]:[port]/api/cos/
- GAME_DATA_SERVICE_URL - [scheme]://[host]:[port]/api/gis/
- WORLD_SERVICE_URL - [scheme]://[host]:[port]/api/wrg/
- TOPIC_CHANNEL_SERVICE - Kafka Topic for transmitting Channel Status Events
- EVENT_TOPIC_SESSION_STATUS - Kafka Topic for capturing session events.
- EVENT_TOPIC_MAP_STATUS - Kafka Topic for transmitting map status events
