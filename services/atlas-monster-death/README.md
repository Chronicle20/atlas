# atlas-monster-death

A Kafka consumer service that handles monster death events. When a monster is killed, this service evaluates and creates item/meso drops based on monster drop tables and distributes experience to characters who damaged the monster.

This service has no REST endpoints and no persistent storage. It operates purely through Kafka message consumption and production.

## External Dependencies

- **Kafka**: Message broker for event consumption and command production
- **OpenTelemetry Collector**: Distributed tracing via OTLP gRPC

## External Service Dependencies

This service makes REST calls to:

- **Character Service** (`CHARACTERS`): Retrieve character information (level)
- **Map Service** (`MAPS`): Get character IDs currently in a map
- **Data Service** (`DATA`): Get monster information (HP, experience), calculate drop positions, and retrieve base equipment statistics
- **Drop Information Service** (`DROPS_INFORMATION`): Get monster drop tables
- **Quest Service** (`QUESTS`): Get started quests for quest-aware drop filtering
- **Party Service** (`PARTIES`): Get party membership for drop ownership
- **Rate Service** (`RATES`): Get character rate multipliers (exp, meso, item drop)

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| `BOOTSTRAP_SERVERS` | Kafka bootstrap servers |
| `TRACE_ENDPOINT` | OpenTelemetry Collector gRPC endpoint |
| `LOG_LEVEL` | Logging level |
| `EVENT_TOPIC_MONSTER_STATUS` | Topic for monster status events |
| `COMMAND_TOPIC_DROP` | Topic for drop spawn commands |
| `COMMAND_TOPIC_CHARACTER` | Topic for character commands |
| `CHARACTERS` | Base URL for character service |
| `MAPS` | Base URL for map service |
| `DATA` | Base URL for data service |
| `DROPS_INFORMATION` | Base URL for drop information service |
| `PARTIES` | Base URL for party service |
| `QUESTS` | Base URL for quest service |
| `RATES` | Base URL for rate service |

## Multi-Tenancy

This service supports multi-tenancy through Kafka headers. Headers are propagated from incoming Kafka messages to outgoing REST calls and Kafka commands.

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
