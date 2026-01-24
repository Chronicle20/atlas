# Atlas Reactor Actions

Service for handling JSON-based reactor scripting. Receives hit and trigger commands from `atlas-reactors`, loads the appropriate script from the database, evaluates rules against reactor state, and executes operations via saga orchestration.

## External Dependencies

- **PostgreSQL**: Stores reactor scripts with tenant isolation
- **Kafka**: Consumes reactor action commands, produces saga commands
- **OpenTelemetry**: Distributed tracing

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| `DB_HOST` | PostgreSQL host |
| `DB_PORT` | PostgreSQL port |
| `DB_NAME` | Database name |
| `DB_USER` | Database user |
| `DB_PASSWORD` | Database password |
| `BOOTSTRAP_SERVERS` | Kafka broker addresses |
| `COMMAND_TOPIC_REACTOR_ACTIONS` | Topic for reactor action commands |
| `COMMAND_TOPIC_SAGA` | Topic for saga commands |
| `REACTOR_ACTIONS_DIR` | Scripts directory for seeding |
| `REST_PORT` | REST API port |
| `LOG_LEVEL` | Logging level |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
