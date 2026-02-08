# atlas-inventory

Inventory management service for character inventories, compartments, and assets. Manages the full lifecycle of character-owned items across five inventory types (Equip, Use, Setup, ETC, Cash) using a unified asset model that stores all item data -- equipment stats, stackable quantities, and cash item metadata -- in a single flattened structure.

## External Dependencies

- PostgreSQL (GORM)
- Kafka
- Jaeger (tracing)
- atlas-pets service (REST, for pet creation during cash item asset creation)
- atlas-data services (REST, for consumable/setup/etc slot max lookups and equipment statistics)
- atlas-drops service (Kafka commands, for item drop and pickup coordination)

## Runtime Configuration

- `JAEGER_HOST_PORT` - Jaeger host:port for distributed tracing
- `LOG_LEVEL` - Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace)
- `REST_PORT` - Port for the REST server
- `BOOTSTRAP_SERVERS` - Kafka bootstrap servers

### Kafka Topics

- `EVENT_TOPIC_ASSET_STATUS` - Asset status events (produced)
- `EVENT_TOPIC_COMPARTMENT_STATUS` - Compartment status events (produced)
- `EVENT_TOPIC_INVENTORY_STATUS` - Inventory status events (produced)
- `COMMAND_TOPIC_COMPARTMENT` - Compartment commands (consumed)
- `COMMAND_TOPIC_DROP` - Drop commands (produced)
- `EVENT_TOPIC_CHARACTER_STATUS` - Character status events (consumed)
- `EVENT_TOPIC_DROP_STATUS` - Drop status events (consumed)

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
