# atlas-npc-conversations
Mushroom game NPC Conversations Service

## Overview

A RESTful resource which provides npc services.

## Environment

- JAEGER_HOST - Jaeger [host]:[port]
- LOG_LEVEL - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace
- CONFIG_FILE - Location of service configuration file.
- BOOTSTRAP_SERVERS - Kafka [host]:[port]
- COMMAND_TOPIC_GUILD - Kafka topic for transmitting Guild commands
- COMMAND_TOPIC_NPC - Kafka topic for transmitting NPC commands 
- COMMAND_TOPIC_NPC_CONVERSATION - Kafka topic for transmitting NPC Conversation commands
- EVENT_TOPIC_CHARACTER_STATUS - Kafka Topic for receiving Character status events

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
