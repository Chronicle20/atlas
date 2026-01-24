# atlas-account

Account management service for the Atlas platform.

The service manages user accounts including authentication, session state tracking, and account attribute updates. It maintains an in-memory registry of active sessions across multiple services (login, channel) and handles state transitions between logged-in, logged-out, and transitioning states.

## External Dependencies

- PostgreSQL: Persistent storage for account data
- Kafka: Message-based command and event processing
- Jaeger: Distributed tracing

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| JAEGER_HOST_PORT | Jaeger host:port for distributed tracing |
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| DB_USER | PostgreSQL user name |
| DB_PASSWORD | PostgreSQL user password |
| DB_HOST | PostgreSQL database host |
| DB_PORT | PostgreSQL database port |
| DB_NAME | PostgreSQL database name |
| BOOTSTRAP_SERVERS | Kafka host:port |
| COMMAND_TOPIC_CREATE_ACCOUNT | Topic for account creation commands |
| COMMAND_TOPIC_ACCOUNT_SESSION | Topic for session commands |
| EVENT_TOPIC_ACCOUNT_STATUS | Topic for account status events |
| EVENT_TOPIC_ACCOUNT_SESSION_STATUS | Topic for session status events |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
