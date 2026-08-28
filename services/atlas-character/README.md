# atlas-character

Character management service for the Atlas game platform. This service handles character lifecycle, stats, appearance, progression, and position tracking. It provides both REST APIs for synchronous operations and Kafka messaging for asynchronous event-driven workflows.

The service coordinates with external services for skill management, drop handling, and session tracking through Kafka messaging.

## External Dependencies

- PostgreSQL database
- Redis (session registry, temporal data)
- Kafka message broker
- OpenTelemetry collector (optional)
- atlas-skill service (via REST and Kafka)
- atlas-drop service (via Kafka)
- atlas-data service (via REST for portal and skill data)
- atlas-effective-stats service (via REST)
- atlas-maps service (via REST for character location lookup)
- atlas-tenants service (via REST, for per-tenant pending-change configuration and tenant resolution during the expiry sweep)
- atlas-world, atlas-account, atlas-ban, atlas-guilds, atlas-families, atlas-trades, atlas-merchant, atlas-mts, atlas-parcel, atlas-parties, atlas-buddies services (via REST, world-transfer eligibility gates and saga snapshotting)
- Saga orchestration (via Kafka, cash-shop coupon consume/refund and world-transfer saga)
- Session status events (via Kafka)

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| TRACE_ENDPOINT | OpenTelemetry collector endpoint |
| TRACE_SAMPLING_RATIO | Trace sampling ratio, 0.0-1.0 (optional, default 1.0) |
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| DB_USER | Postgres user name |
| DB_PASSWORD | Postgres user password |
| DB_HOST | Postgres database host |
| DB_PORT | Postgres database port |
| DB_NAME | Postgres database name |
| BASE_SERVICE_URL | Base service URL (scheme://host:port/api/) |
| BOOTSTRAP_SERVERS | Kafka host:port |
| REDIS_URL | Redis host:port |
| REDIS_PASSWORD | Redis password |
| REST_PORT | REST server port |
| COMMAND_TOPIC_CHARACTER | Character commands topic |
| COMMAND_TOPIC_CHARACTER_MOVEMENT | Character movement commands topic |
| COMMAND_TOPIC_DROP | Drop commands topic |
| COMMAND_TOPIC_SKILL | Skill commands topic |
| EVENT_TOPIC_CHARACTER_STATUS | Character status events topic |
| EVENT_TOPIC_SESSION_STATUS | Session status events topic |
| EVENT_TOPIC_DROP_STATUS | Drop status events topic |
| EVENT_TOPIC_ACCOUNT_STATUS | Account status events topic (consumed) |
| COMMAND_TOPIC_TELEPORT_ROCK | Teleport rock commands topic |
| EVENT_TOPIC_TELEPORT_ROCK_STATUS | Teleport rock status events topic |
| EVENT_TOPIC_CHARACTER_PENDING_CHANGE | Character pending-change events topic |
| COMMAND_TOPIC_SAGA | Saga commands topic |
| SERVICE_MODE | Service mode (READ_ONLY or MIXED, default MIXED) |
| \<DOMAIN\>_SERVICE_URL | Per-domain override of BASE_SERVICE_URL for a specific outbound REST dependency (for example EFFECTIVE_STATS_SERVICE_URL, DATA_SERVICE_URL, MAPS_SERVICE_URL, SKILLS_SERVICE_URL, TENANTS_SERVICE_URL, WORLDS_SERVICE_URL, ACCOUNTS_SERVICE_URL, BANS_SERVICE_URL, GUILDS_SERVICE_URL, FAMILIES_SERVICE_URL, TRADES_SERVICE_URL, MERCHANT_SERVICE_URL, MTS_SERVICE_URL, PARCEL_SERVICE_URL, PARTIES_SERVICE_URL, BUDDIES_SERVICE_URL) |

## Documentation

- [Domain](docs/domain.md) - Domain models and processors
- [Kafka](docs/kafka.md) - Kafka integration
- [REST](docs/rest.md) - REST API endpoints
- [Storage](docs/storage.md) - Database schema
