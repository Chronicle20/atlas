# atlas-mini-games

Owns mini-room mini-games (Omok and Match Cards): room lifecycle (create, visit, leave, chat, expel), in-game commands (ready, start, move stone, flip card, tie, retreat, give up, skip, exit-after-game), and each character's persistent win/tie/loss record per game type.

Live room state lives in a process-wide in-memory registry (`game.Registry`); only the win/tie/loss records are persisted. Because room state is in-memory and not shared across replicas, the service runs as a single replica.

## External Dependencies

- PostgreSQL: persists `game_records` and the transactional outbox table
- Kafka: consumes mini-game commands and character/session status events; produces mini-game status events via a transactional outbox drainer
- atlas-character: REST API for the character's HP (alive check)
- atlas-data: REST API for the map's field limit (mini-room eligibility)
- atlas-inventory: REST API for the character's inventory compartments (room-creation item check)
- atlas-chalkboards: REST API for whether the character has an open chalkboard
- OpenTelemetry: distributed tracing via OTLP/gRPC

## Runtime Configuration

| Variable | Description |
|----------|--------------|
| DB_HOST / DB_PORT / DB_USER / DB_PASSWORD / DB_NAME | PostgreSQL connection |
| BOOTSTRAP_SERVERS | Kafka host:port |
| REST_PORT | HTTP server port |
| CHARACTERS_SERVICE_URL | atlas-character REST API base URL (falls back to BASE_SERVICE_URL) |
| DATA_SERVICE_URL | atlas-data REST API base URL (falls back to BASE_SERVICE_URL) |
| INVENTORY_SERVICE_URL | atlas-inventory REST API base URL (falls back to BASE_SERVICE_URL) |
| CHALKBOARDS_SERVICE_URL | atlas-chalkboards REST API base URL (falls back to BASE_SERVICE_URL) |
| COMMAND_TOPIC_MINI_GAME | Kafka topic for mini-game commands (consumed) |
| EVENT_TOPIC_MINI_GAME_STATUS | Kafka topic for mini-game status events (produced) |
| EVENT_TOPIC_SESSION_STATUS | Kafka topic for session status events (consumed) |
| EVENT_TOPIC_CHARACTER_STATUS | Kafka topic for character status events (consumed) |
| TRACE_ENDPOINT | OpenTelemetry collector gRPC endpoint |
| LOG_LEVEL | Logging level |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
