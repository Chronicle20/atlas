# atlas-monster-death

A Kafka consumer service that handles monster death events. When a monster is killed, this service evaluates and creates item/meso drops based on monster drop tables and distributes experience to characters who damaged the monster.

This service has no REST endpoints and no persistent storage. It operates purely through Kafka message consumption and production.

## External Dependencies

- **Kafka**: Message broker for event consumption and command production
- **Jaeger**: Distributed tracing

## External Service Dependencies

This service makes REST calls to:

- **Character Service** (`CHARACTERS`): Retrieve character information (level)
- **Map Service** (`MAPS`): Get character IDs currently in a map
- **Data Service** (`DATA`): Get monster information (HP, experience) and calculate drop positions
- **Drop Information Service** (`DROPS_INFORMATION`): Get monster drop tables

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| `BOOTSTRAP_SERVERS` | Kafka bootstrap servers |
| `JAEGER_HOST` | Jaeger tracing endpoint |
| `LOG_LEVEL` | Logging level |
| `EVENT_TOPIC_MONSTER_STATUS` | Topic for monster status events |
| `COMMAND_TOPIC_DROP` | Topic for drop spawn commands |
| `COMMAND_TOPIC_CHARACTER` | Topic for character commands |
| `CHARACTERS` | Base URL for character service |
| `MAPS` | Base URL for map service |
| `DATA` | Base URL for data service |
| `DROPS_INFORMATION` | Base URL for drop information service |

## Multi-Tenancy

This service supports multi-tenancy through Kafka headers. Headers are propagated from incoming Kafka messages to outgoing REST calls and Kafka commands.

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
