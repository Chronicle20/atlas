# atlas-ban

IP, HWID, and account-level banning service with login history tracking for the Atlas platform.

The service manages ban records (IP address, HWID, account ID) with support for permanent and temporary bans, CIDR range matching, and expired ban cleanup. It also records login history from account session events for audit purposes, with configurable retention and automatic purging. It also accepts player-submitted reports (sue/claim) against other characters, resolving the accused and a corroborating chat transcript via atlas-character and atlas-messages, and exposes them to GMs for status triage (open/reviewed/actioned).

## External Dependencies

- PostgreSQL: Persistent storage for bans, login history, and reports
- Kafka: Message-based command processing and event consumption
- atlas-character: Resolves reporter/accused character identity for reports
- atlas-messages: Supplies the corroborating chat transcript for reports
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
| COMMAND_TOPIC_REPORT | Topic for report commands |
| EVENT_TOPIC_REPORT_STATUS | Topic for report status events |
| CHARACTERS_SERVICE_URL | atlas-character base URL for report accused/reporter resolution (optional, falls back to BASE_SERVICE_URL) |
| MESSAGES_SERVICE_URL | atlas-messages base URL for report chat transcripts (optional, falls back to BASE_SERVICE_URL) |
| REST_PORT | HTTP server port |
| TRACE_ENDPOINT | OpenTelemetry trace endpoint |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
