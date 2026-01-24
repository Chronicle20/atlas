# atlas-inventory

Inventory management service for character inventories, compartments, and assets.

## Overview

This service manages character inventories composed of typed compartments (Equip, Use, Setup, ETC, Cash) that contain assets. Assets reference type-specific data including equipables, stackables, cash items, and pets.

## External Dependencies

- PostgreSQL (GORM)
- Kafka
- Jaeger (tracing)

## Runtime Configuration

- `JAEGER_HOST_PORT` - Jaeger host:port for distributed tracing
- `LOG_LEVEL` - Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace)
- `REST_PORT` - Port for the REST server
- `BOOTSTRAP_SERVERS` - Kafka bootstrap servers

### Kafka Topics

- `EVENT_TOPIC_ASSET_STATUS` - Asset status events
- `EVENT_TOPIC_COMPARTMENT_STATUS` - Compartment status events
- `EVENT_TOPIC_INVENTORY_STATUS` - Inventory status events
- `COMMAND_TOPIC_COMPARTMENT` - Compartment commands
- `COMMAND_TOPIC_DROP` - Drop commands
- `EVENT_TOPIC_CHARACTER_STATUS` - Character status events (consumed)
- `EVENT_TOPIC_DROP_STATUS` - Drop status events (consumed)
- `EVENT_TOPIC_EQUIPABLE_STATUS` - Equipable status events (consumed)

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
