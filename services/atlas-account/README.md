# atlas-account

Account management service for the Atlas platform.

The service manages user accounts including authentication, session state tracking, and account attribute updates. It maintains an in-memory registry of active sessions across multiple services (login, channel) and handles state transitions between logged-in, logged-out, and transitioning states. During login, the service checks ban status via the atlas-ban REST API using a fail-open strategy. PIN and PIC attempt tracking enforces limits and issues temporary bans via Kafka when exceeded.

## External Dependencies

- PostgreSQL: Persistent storage for account data
- Redis: Session state registry
- Kafka: Message-based command and event processing
- OpenTelemetry (OTLP/gRPC): Distributed tracing
- atlas-ban: Ban status verification via REST API

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| TRACE_ENDPOINT | OpenTelemetry OTLP/gRPC endpoint for distributed tracing |
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| REST_PORT | HTTP server port |
| DB_USER | PostgreSQL user name |
| DB_PASSWORD | PostgreSQL user password |
| DB_HOST | PostgreSQL database host |
| DB_PORT | PostgreSQL database port |
| DB_NAME | PostgreSQL database name |
| BOOTSTRAP_SERVERS | Kafka host:port |
| COMMAND_TOPIC_ACCOUNT | Topic for account commands (create, delete) |
| COMMAND_TOPIC_ACCOUNT_SESSION | Topic for session commands |
| COMMAND_TOPIC_BAN | Topic for ban commands |
| EVENT_TOPIC_ACCOUNT_STATUS | Topic for account status events |
| EVENT_TOPIC_ACCOUNT_SESSION_STATUS | Topic for session status events |
| BANS | Base URL for atlas-ban REST API |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
