# atlas-skills

A microservice that manages character skills and skill macros. It handles skill creation, updates, cooldown tracking, and macro configurations.

## External Dependencies

- PostgreSQL database for persistent skill and macro storage
- Redis for skill cooldown tracking
- Kafka for asynchronous command processing and event emission
- OpenTelemetry-compatible trace collector (OTLP gRPC) for distributed tracing

## Runtime Configuration

| Environment Variable | Description |
|---------------------|-------------|
| `REST_PORT` | Port for the REST server |
| `TRACE_ENDPOINT` | OpenTelemetry collector endpoint in format [host]:[port] |
| `LOG_LEVEL` | Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace |
| `DB_USER` | Database username |
| `DB_PASSWORD` | Database password |
| `DB_HOST` | Database host |
| `DB_PORT` | Database port |
| `DB_NAME` | Database name |
| `BOOTSTRAP_SERVERS` | Kafka bootstrap servers |
| `COMMAND_TOPIC_SKILL` | Kafka topic for skill commands |
| `COMMAND_TOPIC_SKILL_MACRO` | Kafka topic for macro commands |
| `EVENT_TOPIC_CHARACTER_STATUS` | Kafka topic for character status events |
| `EVENT_TOPIC_SKILL_STATUS` | Kafka topic for skill status events |
| `STATUS_EVENT_TOPIC_SKILL_MACRO` | Kafka topic for macro status events |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
