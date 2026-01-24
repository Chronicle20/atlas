# atlas-drop-information

A RESTful service providing drop information for monsters, continents, and reactors. Data is stored in a PostgreSQL database and can be seeded from JSON files.

## External Dependencies

- PostgreSQL database
- Jaeger (optional, for distributed tracing)

## Configuration

| Variable | Description |
|----------|-------------|
| `DB_HOST` | PostgreSQL database host |
| `DB_PORT` | PostgreSQL database port |
| `DB_NAME` | PostgreSQL database name |
| `DB_USER` | PostgreSQL user name |
| `DB_PASSWORD` | PostgreSQL user password |
| `JAEGER_HOST_PORT` | Jaeger tracing endpoint (host:port) |
| `LOG_LEVEL` | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| `REACTOR_DROPS_PATH` | Path to reactor drops directory (default: /drops/reactors) |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST API](docs/rest.md)
- [Storage](docs/storage.md)
