# atlas-channel

Mushroom game Channel Service

## Overview

A service which provides channel game services.

## Environment

- JAEGER_HOST - Jaeger [host]:[port]
- LOG_LEVEL - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace
- BOOTSTRAP_SERVERS - Kafka [host]:[port]
- BASE_SERVICE_URL - [scheme]://[host]:[port]/api/
- SERVICE_ID=[uuid]
- SERVICE_TYPE=channel-service
- EVENT_TOPIC_ACCOUNT_STATUS - Kafka Topic for receiving account status events
- EVENT_TOPIC_CHAIR_STATUS
- EVENT_TOPIC_CHARACTER_GENERAL_CHAT - Kafka Topic for receiving character general chat events
- EVENT_TOPIC_CHARACTER_MOVEMENT - Kafka Topic for receiving character movement events
- EVENT_TOPIC_CHARACTER_STATUS - Kafka Topic for receiving character status events
- EVENT_TOPIC_MAP_STATUS - Kafka Topic for receiving map status events
- EVENT_TOPIC_SESSION_STATUS - Kafka Topic for receiving session events
- COMMAND_TOPIC_ACCOUNT_SESSION - Kafka Topic for transmitting Account Session Commands
- COMMAND_TOPIC_CHAIR
- COMMAND_TOPIC_CHANNEL_STATUS - Kafka Topic for issuing Channel Service commands
    - Used for requesting started channel services to identify status
- COMMAND_TOPIC_CHARACTER_GENERAL_CHAT - Kafka Topic for issuing general chat commands
- COMMAND_TOPIC_CHARACTER_MOVEMENT - Kafka Topic for issuing character movement commands
- COMMAND_TOPIC_MONSTER_MOVEMENT - Kafka Topic for issuing monster movement commands
- COMMAND_TOPIC_PORTAL - Kafka Topic for transmitting portal commands
- 
