# atlas-events

Manages server-wide and scoped gameplay events: their definitions (configuration, enabled state), live occurrences (running instances with stage/state and monster tracking), and the durable scheduled work that evaluates and advances them. Two concrete event types are currently registered: `ANNIVERSARY` (a scheduled EXP/drop-rate window with a login-time buff grant) and `CRIMSON_BALROG` (a chance-based monster attack during an atlas-transports boat voyage).

The service is built around a generic definition/occurrence/transition/scheduling core (`event/definition`, `event/occurrence`, `event/transition`, `event/scheduling`) that knows nothing about any specific event's gameplay. Event-specific behavior is resolved only through the `event/registry.Handler` interface, implemented by each event's own package (`events/anniversary`, `events/crimsonbalrog`) and registered once at startup. A background poller (`event/scheduling.Poller`) claims and executes due scheduled work across every tenant.

## External Dependencies

- PostgreSQL: persistent storage for event definitions, occurrences (with map scope and monster-tracking child tables), occurrence transitions, and scheduled work, via GORM
- Kafka: consumes character status, monster status, and transport status events; produces character-buff commands, monster field commands, and event visual events
- atlas-maps: REST API for listing characters currently in a map instance (CRIMSON_BALROG's "is anyone aboard" check)
- atlas-transports: REST API for reading a route's current voyage state
- atlas-tenants: REST API for listing tenants
- OpenTelemetry: distributed tracing via OTLP/gRPC (through the shared server/consumer libraries)

## Runtime Configuration

| Variable | Description |
|----------|--------------|
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| BOOTSTRAP_SERVERS | Kafka host:port |
| REST_PORT | HTTP server port |
| DB_USER, DB_PASSWORD, DB_HOST, DB_PORT, DB_NAME | PostgreSQL connection |
| MAPS | atlas-maps REST API base URL |
| TRANSPORTS | atlas-transports REST API base URL |
| TENANTS | atlas-tenants REST API base URL |
| EVENT_TOPIC_CHARACTER_STATUS | Kafka topic for character status events (consumed) |
| EVENT_TOPIC_MONSTER_STATUS | Kafka topic for monster status events (consumed) |
| EVENT_TOPIC_TRANSPORT_STATUS | Kafka topic for transport status events (consumed) |
| COMMAND_TOPIC_CHARACTER_BUFF | Kafka topic for character-buff commands (produced) |
| COMMAND_TOPIC_MONSTER | Kafka topic for monster commands (produced) |
| EVENT_TOPIC_EVENT_VISUAL | Kafka topic for event visual notifications (produced) |
| EVENTS_POLL_INTERVAL | Scheduling poller tick interval (default 5s) |
| EVENTS_WORK_LEASE | How long a claimed work row may stay PROCESSING before it is reclaimed (default 5m) |
| EVENTS_POLL_BATCH_SIZE | Maximum rows claimed per poller tick (default 50) |
| EVENTS_WORK_MAX_ATTEMPTS | Retry ceiling for a failing work row (default 5) |
| EVENTS_WORK_BACKOFF | Delay applied to a work row after a failed attempt (default 30s) |
| SEED_CATALOG_ROOT | Filesystem root for the event-definition seed catalog (default `./deploy/seed`) |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
