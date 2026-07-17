# atlas-monster-book

## Overview

The atlas-monster-book service tracks each character's monster card collection: card ownership and level, the aggregate monster-book collection stats (book level, normal/special card counts, EXP bonus percent), and the collection's selected cover card. It consumes monster-card pickup and cover-selection commands, persists the resulting state, and publishes monster-book status events and character EXP distribution events in response.

## External Dependencies

- PostgreSQL database for card and collection storage
- Kafka for command consumption and event production
- atlas-data service (REST) for resolving a cover card item id to its represented monster id

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| TRACE_ENDPOINT | OpenTelemetry collector endpoint |
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| DB_USER | Postgres user name |
| DB_PASSWORD | Postgres user password |
| DB_HOST | Postgres database host |
| DB_PORT | Postgres database port |
| DB_NAME | Postgres database name |
| BOOTSTRAP_SERVERS | Kafka host:port |
| REST_PORT | HTTP server port |
| BASE_SERVICE_URL | Base service URL (scheme://host:port/api/) used as the fallback for downstream service lookups |
| DATA_SERVICE_URL | atlas-data service URL (overrides BASE_SERVICE_URL for consumable lookups) |
| EVENT_TOPIC_CHARACTER_STATUS | Kafka topic for character status events (consumed and produced) |
| COMMAND_TOPIC_MONSTER_BOOK | Kafka topic for monster-book commands |
| EVENT_TOPIC_MONSTER_BOOK_STATUS | Kafka topic for monster-book status events |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
