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
- atlas-configurations service (Kafka config projection: service + tenant configuration is consumed from Kafka, not fetched via REST)

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| TRACE_ENDPOINT | OpenTelemetry collector gRPC endpoint |
| TRACE_SAMPLING_RATIO | OpenTelemetry trace sampling ratio |
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| BOOTSTRAP_SERVERS | Kafka host:port |
| SERVICE_ID | UUID identifying the service instance |
| REST_PORT | Port for the internal REST server (debug/readiness endpoints) |
| KAFKA_CONSUMER_GROUP | Optional override for the Kafka consumer group id template |
| DRAIN_DEADLINE_MS | Listener drain deadline in milliseconds on shutdown (default 2000, clamped to a 5000ms ceiling) |
| PROJECTION_CATCHUP_TIMEOUT_S | Seconds to wait for the configuration projection to catch up at startup (default 300) |
| ACCOUNTS | Base URL for account service |
| CHARACTERS | Base URL for character service |
| CHARACTER_FACTORY | Base URL for character factory service |
| WORLDS | Base URL for world service |
| CHANNELS | Base URL for channel service |
| INVENTORY | Base URL for inventory service |
| GUILDS | Base URL for guild service |
| COMMAND_TOPIC_ACCOUNT_SESSION | Kafka topic for account session commands |
| EVENT_TOPIC_ACCOUNT_SESSION_STATUS | Kafka topic for account session status events |
| EVENT_TOPIC_ACCOUNT_STATUS | Kafka topic for account status events |
| EVENT_TOPIC_SESSION_STATUS | Kafka topic for session status events |
| EVENT_TOPIC_SEED_STATUS | Kafka topic for character seed status events |
| EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS | Kafka topic for service configuration projection events |
| EVENT_TOPIC_CONFIGURATION_TENANT_STATUS | Kafka topic for tenant configuration projection events |

## Documentation

- [Domain](docs/domain.md)
- [Kafka Integration](docs/kafka.md)
- [REST Integration](docs/rest.md)
- [Storage](docs/storage.md)
