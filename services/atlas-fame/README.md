# atlas-fame

## Overview

The atlas-fame service manages character fame (reputation) transactions. It validates fame change requests, enforces business rules around fame timing restrictions, and records fame transaction logs.

## External Dependencies

- PostgreSQL database for fame transaction logs
- Kafka for command consumption and event production
- atlas-character service (REST) for character data retrieval

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| JAEGER_HOST | Jaeger host:port |
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| DB_USER | Postgres user name |
| DB_PASSWORD | Postgres user password |
| DB_HOST | Postgres database host |
| DB_PORT | Postgres database port |
| DB_NAME | Postgres database name |
| BOOTSTRAP_SERVERS | Kafka host:port |
| BASE_SERVICE_URL | Base service URL (scheme://host:port/api/) |
| COMMAND_TOPIC_FAME | Kafka topic for fame commands |
| COMMAND_TOPIC_CHARACTER | Kafka topic for character commands |
| EVENT_TOPIC_FAME_STATUS | Kafka topic for fame status events |
| EVENT_TOPIC_CHARACTER_STATUS | Kafka topic for character status events |
| CHARACTERS | Character service URL |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
