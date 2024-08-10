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
- MAP_SERVICE_URL - [scheme]://[host]:[port]/api/mas/
- MONSTER_SERVICE_URL - [scheme]://[host]:[port]/api/mos/
- WORLD_SERVICE_URL - [scheme]://[host]:[port]/api/wrg/
- TOPIC_CHANNEL_SERVICE - Kafka Topic for transmitting Channel Status Events
- EVENT_TOPIC_CHARACTER_STATUS - Kafka Topic for transmitting character status events
- EVENT_TOPIC_SESSION_STATUS - Kafka Topic for capturing session events.
- EVENT_TOPIC_MAP_STATUS - Kafka Topic for transmitting map status events
- COMMAND_TOPIC_PORTAL - Kafka Topic for transmitting portal commands.
- COMMAND_TOPIC_CHANNEL_STATUS - Kafka Topic for issuing Channel Service commands.
    - Used for requesting started channel services to identify status.