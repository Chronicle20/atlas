# Atlas Consumables Service

## Overview

Atlas Consumables manages consumable item usage in the game. It handles consumption of potions, scrolls, pet food, summoning sacks, and equipment enhancement scrolls. The service processes item effects including HP/MP recovery, temporary stat buffs, teleportation, pet feeding, and equipment stat modifications.

## External Dependencies

- Kafka: Asynchronous messaging for commands and events
- Jaeger: Distributed tracing

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| `JAEGER_HOST_PORT` | Jaeger host:port for distributed tracing |
| `LOG_LEVEL` | Logging level (Panic / Fatal / Error / Warn / Info / Debug / Trace) |
| `BOOTSTRAP_SERVERS` | Kafka bootstrap servers |
| `COMMAND_TOPIC_CONSUMABLE` | Topic for consumable commands |
| `EVENT_TOPIC_CONSUMABLE_STATUS` | Topic for consumable status events |
| `COMMAND_TOPIC_CHARACTER` | Topic for character commands |
| `EVENT_TOPIC_CHARACTER_STATUS` | Topic for character status events |
| `COMMAND_TOPIC_COMPARTMENT` | Topic for compartment commands |
| `EVENT_TOPIC_COMPARTMENT_STATUS` | Topic for compartment status events |
| `COMMAND_TOPIC_CHARACTER_BUFF` | Topic for character buff commands |
| `COMMAND_TOPIC_EQUIPABLE` | Topic for equipable commands |
| `COMMAND_TOPIC_PET` | Topic for pet commands |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
