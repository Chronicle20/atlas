# atlas-saga-orchestrator
Mushroom game saga-orchestrator Service

## Overview

The Atlas Saga Orchestrator is a service that manages distributed transactions (sagas) across multiple microservices in the Mushroom game ecosystem. It provides:

- Saga pattern implementation for maintaining data consistency across services
- Transaction tracking and management
- RESTful API for querying saga status
- Kafka integration for receiving saga commands

This service acts as a central coordinator for complex operations that span multiple services, ensuring that either all steps complete successfully or compensating actions are taken to maintain system consistency.

## Environment Variables

- `BOOTSTRAP_SERVERS` - Kafka bootstrap servers (comma-separated list of host:port pairs)
- `JAEGER_HOST_PORT` - Jaeger host and port for distributed tracing
- `LOG_LEVEL` - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace
- `REST_PORT` - Port for the REST API server
- `SAGA_COMMAND_TOPIC` - Kafka topic for saga commands

## API

### Header

All RESTful requests require the supplied header information to identify the server instance.

```
TENANT_ID:083839c6-c47c-42a6-9585-76492795d123
REGION:GMS
MAJOR_VERSION:83
MINOR_VERSION:1
```

### Endpoints

#### GET /api/sagas
Returns a list of all sagas in the system.

**Response**: JSON:API collection of saga resources

#### GET /api/sagas/{transactionId}
Returns a specific saga by its transaction ID.

**Parameters**:
- `transactionId`: UUID of the saga transaction

**Response**: JSON:API resource representing a saga

## Kafka Integration

### Consumers

The service consumes messages from the following Kafka topics:

- `SAGA_COMMAND_TOPIC` - Processes saga commands for orchestrating distributed transactions

### Message Format

#### Saga Command

```json
{
  "transaction_id": "uuid-string",
  "saga_type": "inventory_transaction|quest_reward|trade_transaction",
  "initiated_by": "string",
  "steps": [
    {
      "step_id": "string",
      "status": "pending|completed|failed",
      "action": "award_inventory",
      "payload": {
        "character_id": 12345,
        "items": [
          {
            "template_id": 2000,
            "quantity": 1
          }
        ]
      },
      "created_at": "2023-01-01T00:00:00Z",
      "updated_at": "2023-01-01T00:00:00Z"
    }
  ]
}
```

### Supported Saga Types

- `inventory_transaction` - Manages inventory-related transactions
- `quest_reward` - Handles quest reward distribution
- `trade_transaction` - Manages player-to-player trading

### Supported Actions

- `award_inventory` - Awards items to a character's inventory
