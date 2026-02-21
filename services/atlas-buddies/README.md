# atlas-buddies

Buddy list management service for game characters.

## Overview

Manages buddy lists for characters including adding/removing buddies, tracking online status, channel changes, and cash shop presence. Supports buddy invitations through coordination with an external invite service.

## External Dependencies

- **PostgreSQL**: Persistent storage for buddy lists and buddy entries
- **Kafka**: Message broker for commands and events
- **Jaeger**: Distributed tracing
- **atlas-character**: External service for character information lookups
- **atlas-invite**: External service for buddy invitation management

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| JAEGER_HOST_PORT | Jaeger host:port |
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| DB_USER | Postgres user name |
| DB_PASSWORD | Postgres user password |
| DB_HOST | Postgres database host |
| DB_PORT | Postgres database port |
| DB_NAME | Postgres database name |
| REST_PORT | HTTP server port |
| BOOTSTRAP_SERVERS | Kafka host:port |
| BASE_SERVICE_URL | Base URL for external service calls |
| COMMAND_TOPIC_BUDDY_LIST | Kafka topic for buddy list commands |
| COMMAND_TOPIC_INVITE | Kafka topic for invite commands |
| EVENT_TOPIC_BUDDY_LIST_STATUS | Kafka topic for buddy list status events |
| EVENT_TOPIC_CASH_SHOP_STATUS | Kafka topic for cash shop status events |
| EVENT_TOPIC_CHARACTER_STATUS | Kafka topic for character status events |
| EVENT_TOPIC_INVITE_STATUS | Kafka topic for invite status events |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
