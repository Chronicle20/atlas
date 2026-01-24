# atlas-quest

Quest state management and progress tracking service. Handles quest lifecycle operations (start, complete, forfeit) and tracks progress for quest objectives including monster kills and map visits.

## External Dependencies

| Dependency | Type | Purpose |
|------------|------|---------|
| PostgreSQL | Database | Quest status and progress persistence |
| Kafka | Messaging | Command/event communication |
| Jaeger | Tracing | Distributed tracing |
| atlas-data | Service | Quest definitions |
| query-aggregator | Service | Character state validation |
| saga-orchestrator | Service | Rewards distribution |

## Configuration

| Variable | Description |
|----------|-------------|
| JAEGER_HOST_PORT | Jaeger host:port |
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| DB_USER | PostgreSQL username |
| DB_PASSWORD | PostgreSQL password |
| DB_HOST | PostgreSQL host |
| DB_PORT | PostgreSQL port |
| DB_NAME | PostgreSQL database name |
| BOOTSTRAP_SERVERS | Kafka host:port |
| BASE_SERVICE_URL | Service base URL |
| COMMAND_TOPIC_QUEST | Quest command topic |
| COMMAND_TOPIC_SAGA | Saga command topic |
| EVENT_TOPIC_QUEST_STATUS | Quest status event topic |
| EVENT_TOPIC_MONSTER_STATUS | Monster status event topic |
| EVENT_TOPIC_ASSET_STATUS | Asset status event topic |
| EVENT_TOPIC_CHARACTER_STATUS | Character status event topic |
| DATA_BASE_URL | atlas-data service URL |
| QUERY_AGGREGATOR_BASE_URL | query-aggregator service URL |

## Documentation

- [Domain](docs/domain.md) - Domain models, invariants, and processors
- [Kafka](docs/kafka.md) - Kafka topics and message types
- [REST](docs/rest.md) - REST API endpoints
- [Storage](docs/storage.md) - Database schema
