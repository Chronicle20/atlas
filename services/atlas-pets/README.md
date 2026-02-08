# Atlas Pets Service

A microservice for managing in-game pets. This service handles pet creation, retrieval, spawning, despawning, attribute management, and lifecycle operations. It maintains an in-memory character registry to track logged-in characters for the periodic hunger evaluation task, and an in-memory temporal registry to track pet positions and stances.

The service consumes character status events, asset status events, and pet commands via Kafka. It reads character and inventory data from external services via REST. Pet data is fetched from a reference data service for template-level attributes like hunger rate and command skills.

## External Dependencies

- PostgreSQL database
- Kafka cluster
- Jaeger (tracing)
- atlas-characters service (REST)
- atlas-inventory service (REST)
- Pet reference data service (REST)
- Foothold / position reference data service (REST)
- Skill reference data service (REST)

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| JAEGER_HOST | Jaeger host and port for tracing |
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| REST_PORT | Port for the REST API |
| DB_HOST | PostgreSQL database host |
| DB_PORT | PostgreSQL database port |
| DB_USER | PostgreSQL database username |
| DB_PASSWORD | PostgreSQL database password |
| DB_NAME | PostgreSQL database name |
| BOOTSTRAP_SERVERS | Kafka broker address |
| EVENT_TOPIC_CHARACTER_STATUS | Character status events topic |
| EVENT_TOPIC_ASSET_STATUS | Asset status events topic |
| COMMAND_TOPIC_PET | Pet commands topic |
| COMMAND_TOPIC_PET_MOVEMENT | Pet movement commands topic |
| EVENT_TOPIC_PET_STATUS | Pet status events topic |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
