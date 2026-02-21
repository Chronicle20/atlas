# atlas-ban

IP, HWID, and account-level banning service with login history tracking for the Atlas platform.

The service manages ban records (IP address, HWID, account ID) with support for permanent and temporary bans, CIDR range matching, and expired ban cleanup. It also records login history from account session events for audit purposes, with configurable retention and automatic purging.

## External Dependencies

- PostgreSQL: Persistent storage for bans and login history
- Kafka: Message-based command processing and event consumption
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
| COMMAND_TOPIC_BAN | Topic for ban commands |
| EVENT_TOPIC_BAN_STATUS | Topic for ban status events |
| EVENT_TOPIC_ACCOUNT_SESSION_STATUS | Topic for account session status events |
| REST_PORT | HTTP server port |
| TRACE_ENDPOINT | OpenTelemetry trace endpoint |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
