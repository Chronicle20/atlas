# atlas-drop-information

A RESTful service providing drop information for monsters, continents, and reactors. Data is stored in a PostgreSQL database and can be seeded from JSON files.

## External Dependencies

- PostgreSQL database
- OpenTelemetry collector (optional, for distributed tracing via OTLP/gRPC)

## Configuration

| Variable | Description |
|----------|-------------|
| `DB_HOST` | PostgreSQL database host |
| `DB_PORT` | PostgreSQL database port |
| `DB_NAME` | PostgreSQL database name |
| `DB_USER` | PostgreSQL user name |
| `DB_PASSWORD` | PostgreSQL user password |
| `REST_PORT` | HTTP server port |
| `TRACE_ENDPOINT` | OpenTelemetry tracing endpoint (host:port) |
| `LOG_LEVEL` | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| `SEED_CATALOG_ROOT` | Path to the seed catalog root directory (default: ./deploy/seed) |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST API](docs/rest.md)
- [Storage](docs/storage.md)
