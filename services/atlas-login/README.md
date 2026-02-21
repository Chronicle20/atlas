# atlas-login

A stateful, multi-tenant, multi-version login service for a mushroom game. This service handles client authentication, session management, and world/channel selection during the login flow.

## External Dependencies

- Kafka (message broker)
- OpenTelemetry (distributed tracing)
- atlas-account service (REST)
- atlas-character service (REST)
- atlas-character-factory service (REST)
- atlas-world service (REST)
- atlas-channel service (REST)
- atlas-inventory service (REST)
- atlas-guild service (REST)
- atlas-configurations service (REST)

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| TRACE_ENDPOINT | OpenTelemetry collector gRPC endpoint |
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| BOOTSTRAP_SERVERS | Kafka host:port |
| SERVICE_ID | UUID identifying the service instance |
| SERVICE_TYPE | Service type identifier (login-service) |
| ACCOUNTS | Base URL for account service |
| CHARACTERS | Base URL for character service |
| CHARACTER_FACTORY | Base URL for character factory service |
| WORLDS | Base URL for world service |
| CHANNELS | Base URL for channel service |
| INVENTORY | Base URL for inventory service |
| GUILDS | Base URL for guild service |
| CONFIGURATIONS | Base URL for configuration service |
| COMMAND_TOPIC_ACCOUNT_SESSION | Kafka topic for account session commands |
| EVENT_TOPIC_ACCOUNT_SESSION_STATUS | Kafka topic for account session status events |
| EVENT_TOPIC_ACCOUNT_STATUS | Kafka topic for account status events |
| EVENT_TOPIC_SESSION_STATUS | Kafka topic for session status events |
| EVENT_TOPIC_SEED_STATUS | Kafka topic for character seed status events |

## Documentation

- [Domain](docs/domain.md)
- [Kafka Integration](docs/kafka.md)
- [REST Integration](docs/rest.md)
- [Storage](docs/storage.md)
