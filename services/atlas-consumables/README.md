# Atlas Consumables Service

## Overview

Atlas Consumables is a microservice that manages consumable items in the game, including their effects, interactions, and usage. It handles various types of consumables such as:

- Standard consumables (HP/MP potions, stat buffs, etc.)
- Town scrolls and teleportation items
- Pet food and pet-related consumables
- Summoning sacks and monster spawning items
- Equipment enhancement scrolls

### Environment Variables
- `JAEGER_HOST_PORT` - Jaeger [host]:[port] for distributed tracing
- `LOG_LEVEL` - Logging level (Panic / Fatal / Error / Warn / Info / Debug / Trace)
- `BOOTSTRAP_SERVERS` - Kafka bootstrap servers
- `COMMAND_TOPIC_CONSUMABLE` - Kafka topic for consumable commands
- `EVENT_TOPIC_CONSUMABLE_STATUS` - Kafka topic for consumable status events
- `COMMAND_TOPIC_CHARACTER` - Kafka topic for character commands
- `EVENT_TOPIC_CHARACTER_STATUS` - Kafka topic for character status events
- `COMMAND_TOPIC_COMPARTMENT` - Kafka topic for compartment commands
- `EVENT_TOPIC_COMPARTMENT_STATUS` - Kafka topic for compartment status events
- `COMMAND_TOPIC_CHARACTER_BUFF` - Kafka topic for character buff commands
- `COMMAND_TOPIC_EQUIPABLE` - Kafka topic for equipable commands
- `COMMAND_TOPIC_PET` - Kafka topic for pet commands

## API

### Kafka Integration

The service communicates with other services using Kafka for asynchronous messaging.

#### Command Topics
- `COMMAND_TOPIC_CONSUMABLE` - For receiving commands to consume items or use scrolls

#### Event Topics
- `EVENT_TOPIC_CONSUMABLE_STATUS` - For emitting events about consumable usage results

#### Command Types
- `REQUEST_ITEM_CONSUME` - Request to consume an item
- `REQUEST_SCROLL` - Request to use a scroll on equipment

#### Event Types
- `ERROR` - Reports errors during consumable usage
- `SCROLL` - Reports results of scroll usage (success/failure)

## Usage Examples

### Consuming an Item

To consume an item, send a Kafka message to the `COMMAND_TOPIC_CONSUMABLE` topic:

```json
{
  "worldId": 0,
  "channelId": 1,
  "characterId": 12345,
  "type": "REQUEST_ITEM_CONSUME",
  "body": {
    "source": 2,
    "itemId": 2000000,
    "quantity": 1
  }
}
```

### Using a Scroll

To use a scroll on equipment, send a Kafka message to the `COMMAND_TOPIC_CONSUMABLE` topic:

```json
{
  "worldId": 0,
  "channelId": 1,
  "characterId": 12345,
  "type": "REQUEST_SCROLL",
  "body": {
    "scrollSlot": 1,
    "equipSlot": 5,
    "whiteScroll": false,
    "legendarySpirit": false
  }
}
```
