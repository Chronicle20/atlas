# atlas-messages
Mushroom game messages Service

## Overview

A service which handles character messages.

## Environment

- JAEGER_HOST - Jaeger [host]:[port]
- LOG_LEVEL - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace
- COMMAND_TOPIC_CHARACTER_GENERAL_CHAT - Kafka Topic for transmitting message commands.
- EVENT_TOPIC_CHARACTER_GENERAL_CHAT - Kafka Topic for transmitting message events.
