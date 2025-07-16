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
- `COMMAND_TOPIC_SAGA` - Kafka topic for saga commands
- `COMMAND_TOPIC_GUILD` - Kafka topic for guild commands
- `COMMAND_TOPIC_COMPARTMENT` - Kafka topic for compartment commands
- `COMMAND_TOPIC_CHARACTER` - Kafka topic for character commands
- `EVENT_TOPIC_GUILD_STATUS` - Kafka topic for guild status events
- `EVENT_TOPIC_COMPARTMENT_STATUS` - Kafka topic for compartment status events
- `EVENT_TOPIC_CHARACTER_STATUS` - Kafka topic for character status events

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

- `COMMAND_TOPIC_SAGA` - Processes saga commands for orchestrating distributed transactions
- `EVENT_TOPIC_GUILD_STATUS` - Processes guild status events for saga step completion
- `EVENT_TOPIC_COMPARTMENT_STATUS` - Processes compartment status events for saga step completion
- `EVENT_TOPIC_CHARACTER_STATUS` - Processes character status events for saga step completion

### Message Format

#### Saga Command

```json
{
  "transaction_id": "uuid-string",
  "saga_type": "inventory_transaction|quest_reward|trade_transaction|character_creation",
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
- `guild_management` - Manages guild-related operations
- `character_creation` - Manages character creation workflows

### Supported Actions

- `award_asset` - Awards items to a character's inventory
  - Payload: `{"characterId": 12345, "item": {"templateId": 2000, "quantity": 1}}`
  - Triggers a compartment command to create the item
  - Completes when the item is successfully added to the inventory

- `award_inventory` - (Deprecated: Use `award_asset` instead) Awards items to a character's inventory
  - Payload: `{"characterId": 12345, "item": {"templateId": 2000, "quantity": 1}}`
  - Triggers a compartment command to create the item
  - Completes when the item is successfully added to the inventory

- `award_experience` - Awards experience points to a character
  - Payload: `{"characterId": 12345, "worldId": 0, "channelId": 0, "distributions": [{"experienceType": "WHITE", "amount": 1000, "attr1": 0}]}`
  - Triggers a character command to award experience
  - Completes when the StatusEventTypeExperienceChanged event is received

- `award_level` - Awards levels to a character
  - Payload: `{"characterId": 12345, "worldId": 0, "channelId": 0, "amount": 1}`
  - Triggers a character command to award levels
  - Completes when the StatusEventTypeLevelChanged event is received

- `award_mesos` - Awards mesos (currency) to a character
  - Payload: `{"characterId": 12345, "worldId": 0, "channelId": 0, "actorId": 0, "actorType": "SYSTEM", "amount": 1000}`
  - Triggers a character command to award mesos
  - Completes when the StatusEventTypeMesoChanged event is received

- `warp_to_random_portal` - Warps a character to a random portal in a field
  - Payload: `{"characterId": 12345, "fieldId": 100000000}`
  - Triggers a character command to warp to a random portal
  - Completes when the StatusEventTypeMapChanged event is received

- `warp_to_portal` - Warps a character to a specific portal in a field
  - Payload: `{"characterId": 12345, "fieldId": 100000000, "portalId": 1}`
  - Triggers a character command to warp to the specified portal
  - Completes when the StatusEventTypeMapChanged event is received

- `destroy_asset` - Destroys an asset in a character's inventory
  - Payload: `{"characterId": 12345, "templateId": 2000, "quantity": 5}`
  - Triggers a compartment command to destroy the item
  - Completes when the StatusEventTypeDeleted event is received

- `equip_asset` - Equips an item from inventory to an equipment slot
  - Payload: `{"characterId": 12345, "inventoryType": 1, "source": 1, "destination": -1}`
  - Triggers a compartment command to equip the item
  - Completes when the StatusEventTypeEquipped event is received

- `unequip_asset` - Unequips an item from an equipment slot to inventory
  - Payload: `{"characterId": 12345, "inventoryType": 1, "source": -1, "destination": 1}`
  - Triggers a compartment command to unequip the item
  - Completes when the StatusEventTypeUnequipped event is received

- `change_job` - Changes a character's job
  - Payload: `{"characterId": 12345, "worldId": 0, "channelId": 0, "jobId": 100}`
  - Triggers a character command to change the job
  - Completes when the StatusEventTypeJobChanged event is received

- `create_skill` - Creates a skill for a character
  - Payload: `{"characterId": 12345, "skillId": 1000, "level": 1, "masterLevel": 1, "expiration": "2023-01-01T00:00:00Z"}`
  - Triggers a skill command to create the skill
  - Completes when the StatusEventTypeCreated event is received

- `update_skill` - Updates a skill for a character
  - Payload: `{"characterId": 12345, "skillId": 1000, "level": 2, "masterLevel": 2, "expiration": "2023-01-01T00:00:00Z"}`
  - Triggers a skill command to update the skill
  - Completes when the StatusEventTypeUpdated event is received

- `validate_character_state` - Validates a character's state against a set of conditions
  - Payload: `{"characterId": 12345, "conditions": [{"type": "jobId", "operator": "=", "value": 100}, {"type": "meso", "operator": ">=", "value": 1000}]}`
  - Makes a synchronous HTTP call to the query-aggregator service's validation endpoint
  - Completes when all conditions pass, fails if any condition fails
  - Supported condition types: "jobId", "meso", "mapId", "fame", "item" (requires additional "itemId" field)

- `request_guild_name` - Initiates the guild name change dialog
  - Payload: `{"characterId": 12345, "worldId": 0, "channelId": 0}`
  - Triggers a guild command to request a name change
  - Completes when the StatusEventTypeRequestAgreement event is received

- `request_guild_emblem` - Initiates the guild emblem change dialog
  - Payload: `{"characterId": 12345, "worldId": 0, "channelId": 0}`
  - Triggers a guild command to request an emblem change
  - Completes when the StatusEventTypeEmblemUpdated event is received

- `request_guild_disband` - Requests a guild disband
  - Payload: `{"characterId": 12345, "worldId": 0, "channelId": 0}`
  - Triggers a guild command to request a disband
  - Completes when the StatusEventTypeDisbanded event is received

- `request_guild_capacity_increase` - Requests a guild capacity increase
  - Payload: `{"characterId": 12345, "worldId": 0, "channelId": 0}`
  - Triggers a guild command to request a capacity increase
  - Completes when the StatusEventTypeCapacityUpdated event is received

- `create_character` - Creates a new character
  - Payload: `{"accountId": 12345, "name": "NewCharacter", "worldId": 1, "channelId": 0, "jobId": 0, "face": 20000, "hair": 30000, "hairColor": 0, "skin": 0, "top": 1040002, "bottom": 1060002, "shoes": 1072001, "weapon": 1302000}`
  - Triggers a character command to create a new character
  - Completes when the StatusEventTypeCreated event is received with matching transaction ID
  - Fails when the StatusEventTypeCreationFailed or StatusEventTypeError event is received
