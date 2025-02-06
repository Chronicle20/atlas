# atlas-messages
Mushroom game messages Service

## Overview

A service which handles character messages.

## Environment

- JAEGER_HOST - Jaeger [host]:[port]
- LOG_LEVEL - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace
- BASE_SERVICE_URL - [scheme]://[host]:[port]/api/
- BOOTSTRAP_SERVERS - Kafka [host]:[port]
- COMMAND_TOPIC_CHARACTER - Kafka Topic for transmitting character commands
- COMMAND_TOPIC_CHARACTER_GENERAL_CHAT - Kafka Topic for transmitting message commands.
- EVENT_TOPIC_CHARACTER_GENERAL_CHAT - Kafka Topic for transmitting message events.
