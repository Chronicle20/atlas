# Atlas Tenants Service

A RESTful microservice that provides tenant management for the Atlas game platform. This service manages tenant information and tenant-specific configurations including routes, vessels, and instance routes.

## External Dependencies

- PostgreSQL database
- Kafka message broker
- OpenTelemetry-compatible trace collector (optional, for distributed tracing)

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

- `TRACE_ENDPOINT` - OpenTelemetry OTLP gRPC endpoint for distributed tracing
- `LOG_LEVEL` - Logging level (Panic / Fatal / Error / Warn / Info / Debug / Trace)
- `ROUTES_SEED_PATH` - Filesystem path to route seed JSON files (default: `/configurations/routes`)
- `INSTANCE_ROUTES_SEED_PATH` - Filesystem path to instance route seed JSON files (default: `/configurations/instance-routes`)
- `VESSELS_SEED_PATH` - Filesystem path to vessel seed JSON files (default: `/configurations/vessels`)

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
