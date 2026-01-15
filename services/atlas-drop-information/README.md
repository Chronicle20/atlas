# atlas-drop-information

A RESTful service providing drop information for monsters and continents. Data is stored in a PostgreSQL database and can be seeded from JSON files.

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

## Documentation

- [Domain](docs/domain.md)
- [REST API](docs/rest.md)
- [Storage](docs/storage.md)
