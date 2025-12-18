# atlas-compartment-transfer
Mushroom game atlas-compartment-transfer Service

## Overview

A Kafka-based microservice that handles the transfer of items between different compartments (character inventory, cash shop, etc.) in the Atlas system.

## Environment Variables

### Required Environment Variables
- `BOOTSTRAP_SERVERS` - Kafka bootstrap servers (e.g., `localhost:9092`)
- `JAEGER_HOST_PORT` - Jaeger host and port for distributed tracing (e.g., `jaeger:4317`)
- `LOG_LEVEL` - Logging level (`panic`, `fatal`, `error`, `warn`, `info`, `debug`, `trace`)
- `BASE_SERVICE_URL` - Base URL for service communication

### Kafka Topic Configuration
- `COMMAND_TOPIC_CASH_COMPARTMENT` - Topic for cash compartment commands
- `COMMAND_TOPIC_COMPARTMENT` - Topic for compartment commands
- `COMMAND_TOPIC_COMPARTMENT_TRANSFER` - Topic for compartment transfer commands
- `EVENT_TOPIC_CASH_COMPARTMENT_STATUS` - Topic for cash compartment status events
- `EVENT_TOPIC_COMPARTMENT_STATUS` - Topic for compartment status events
- `EVENT_TOPIC_COMPARTMENT_TRANSFER_STATUS` - Topic for compartment transfer status events

## Kafka Messaging

### Consumer Groups
The service uses the consumer group ID "Compartment Transfer Service" for all Kafka consumers.

### Message Types

#### Commands
- `TransferCommand` - Command to transfer an item between compartments
  - Contains transaction ID, account ID, character ID, asset ID, source and destination compartment details

#### Events
- `StatusEvent` - Generic event structure with a type parameter for the body
  - `StatusEventCompletedBody` - Event body for completed transfers

### Inventory Types
- `CHARACTER` - Character inventory
- `CASH_SHOP` - Cash shop inventory

### Messaging Pattern
The service follows a command-event pattern:
1. Receives commands on command topics
2. Processes the commands
3. Emits status events on event topics
