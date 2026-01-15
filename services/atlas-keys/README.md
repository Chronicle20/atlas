# atlas-keys

Key binding service for characters. Manages keyboard mappings that associate keys with types and actions.

## External Dependencies

- PostgreSQL database
- Kafka (consumes character status events)
- Jaeger (distributed tracing)

## Runtime Configuration

### Database
- `DB_USER` - Postgres user name
- `DB_PASSWORD` - Postgres user password
- `DB_HOST` - Postgres database host
- `DB_PORT` - Postgres database port
- `DB_NAME` - Postgres database name

### Kafka
- `BOOTSTRAP_SERVERS` - Kafka bootstrap servers
- `EVENT_TOPIC_CHARACTER_STATUS` - Topic for character status events

### HTTP Server
- `REST_PORT` - Port for REST API

### Observability
- `JAEGER_HOST` - Jaeger host and port
- `LOG_LEVEL` - Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace)

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST API](docs/rest.md)
- [Storage](docs/storage.md)
