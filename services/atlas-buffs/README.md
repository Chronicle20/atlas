# atlas-buffs

Buff management service for the Atlas platform.

The service manages temporary stat modifications (buffs) for game characters. It maintains a Redis-backed registry of active buffs per character, tracks buff durations, and handles automatic expiration. Buffs are applied and cancelled via Kafka commands, with status events emitted for buff lifecycle changes. The service processes periodic HP effects (poison, dragon blood, recovery) for characters with active periodic buffs, and tracks Dark Knight Berserk aura state per character, re-evaluating and broadcasting it on a schedule.

## External Dependencies

- Redis: Buff and berserk-tracking state storage via TenantRegistry
- Kafka: Message-based command and event processing
- OpenTelemetry Collector: Distributed tracing via OTLP gRPC
- atlas-character (CHARACTERS): HTTP dependency for character HP/level lookups
- atlas-skills (SKILLS): HTTP dependency for Berserk skill level lookups
- atlas-effective-stats (EFFECTIVE_STATS): HTTP dependency for effective max HP lookups
- atlas-data (DATA): HTTP dependency for Berserk skill effect data

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| REDIS_URL | Redis server address |
| REDIS_PASSWORD | Redis authentication password |
| TRACE_ENDPOINT | OpenTelemetry collector endpoint for distributed tracing |
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| REST_PORT | HTTP server port |
| BOOTSTRAP_SERVERS | Kafka bootstrap servers |
| COMMAND_TOPIC_CHARACTER_BUFF | Topic for buff commands |
| EVENT_TOPIC_CHARACTER_BUFF_STATUS | Topic for buff status events |
| COMMAND_TOPIC_CHARACTER | Topic for character commands (periodic HP changes) |
| EVENT_TOPIC_CHARACTER_STATUS | Topic for consumed character status events (login/logout/stat/map/channel change) |
| EVENT_TOPIC_SKILL_STATUS | Topic for consumed skill status events (Berserk skill updates/deletions) |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
