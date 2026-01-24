# Atlas Tenants Service

A RESTful microservice that provides tenant management for the Atlas game platform. This service manages tenant information and tenant-specific configurations including routes and vessels.

## External Dependencies

- PostgreSQL database
- Kafka message broker
- Jaeger (optional, for distributed tracing)

## Runtime Configuration

### Required Environment Variables

- `REST_PORT` - Port for the REST API server
- `BOOTSTRAP_SERVERS` - Kafka bootstrap servers
- `DB_HOST` - PostgreSQL database host
- `DB_PORT` - PostgreSQL database port
- `DB_USER` - PostgreSQL database username
- `DB_PASSWORD` - PostgreSQL database password
- `DB_NAME` - PostgreSQL database name

### Optional Environment Variables

- `JAEGER_HOST_PORT` - Jaeger agent host and port for distributed tracing
- `LOG_LEVEL` - Logging level (Panic / Fatal / Error / Warn / Info / Debug / Trace)

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
