# atlas-inventory
Mushroom game inventory Service

## Overview

A RESTful resource which provides inventory services.

## Environment

- JAEGER_HOST_PORT - Jaeger [host]:[port] for distributed tracing
- LOG_LEVEL - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace
- REST_PORT - Port for the REST server
- BOOTSTRAP_SERVERS - Kafka bootstrap servers for message consumers

### Kafka Topics

- EVENT_TOPIC_ASSET_STATUS - Topic for asset status events (created, deleted, moved, quantity changed)
- EVENT_TOPIC_COMPARTMENT_STATUS - Topic for compartment status events (created, deleted, capacity changed, reserved, reservation cancelled)
- COMMAND_TOPIC_COMPARTMENT - Topic for compartment commands (equip, unequip, move, drop, request reserve, consume, destroy, recharge, etc.)
- EVENT_TOPIC_CHARACTER_STATUS - Topic for character status events (created, deleted)
- COMMAND_TOPIC_DROP - Topic for drop commands (spawn from character)
- EVENT_TOPIC_INVENTORY_STATUS - Topic for inventory status events (created, deleted)
- EVENT_TOPIC_DROP_STATUS - Topic for drop status events
- EVENT_TOPIC_EQUIPABLE_STATUS - Topic for equipable status events

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

#### Inventory Endpoints

- `GET /characters/{characterId}/inventory` - Get a character's inventory
- `POST /characters/{characterId}/inventory` - Create a default inventory for a character
- `DELETE /characters/{characterId}/inventory` - Delete a character's inventory

#### Compartment Endpoints

- `GET /characters/{characterId}/inventory/compartments/{compartmentId}` - Get a specific compartment for a character

#### Asset Endpoints

- `GET /characters/{characterId}/inventory/compartments/{compartmentId}/assets` - Get all assets in a compartment
- `DELETE /characters/{characterId}/inventory/compartments/{compartmentId}/assets/{assetId}` - Delete a specific asset

### Kafka Commands

The service supports the following Kafka commands through the COMMAND_TOPIC_COMPARTMENT topic:

- EQUIP - Equip an item from one slot to another
- UNEQUIP - Unequip an item from equipment to inventory
- MOVE - Move an item from one slot to another within the same compartment
- DROP - Drop an item from inventory to the map
- REQUEST_RESERVE - Reserve items for a transaction
- CONSUME - Consume a reserved item
- DESTROY - Destroy an item in inventory
- CANCEL_RESERVATION - Cancel a reservation
- INCREASE_CAPACITY - Increase the capacity of a compartment
- CREATE_ASSET - Create a new asset in a compartment
- RECHARGE - Recharge an asset in a compartment (for TypeValueUse compartment type only)
