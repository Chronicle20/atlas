# atlas-messages

Atlas Messages is a service that handles character messages and commands in the Mushroom game. It processes various types of chat messages including general chat, whispers, multi-recipient messages, messenger messages, and pet messages. The service provides a command system that allows Game Masters (GMs) to execute administrative commands through the chat interface.

It also captures recently sent, player-authored chat lines (general, buddy,
party, guild, alliance, whisper, messenger) into a short-retention,
tenant-scoped Redis buffer — pet-echoed chat and system-issued pink text are
never captured. `GET /api/chat/history` lets atlas-ban read that buffer to
snapshot a transcript when a player report is created. This endpoint **is**
routed through nginx/ingress and reachable at the ingress host without
authentication, exposing captured chat including whispers — see
[REST](docs/rest.md) for the accepted-risk citation. See
[Storage](docs/storage.md) for the buffer's retention semantics.

## External Dependencies

- Kafka (message streaming)
- Redis (short-retention chat-capture buffer)
- OpenTelemetry (distributed tracing via OTLP/gRPC)
- atlas-character service (REST API)
- atlas-skills service (REST API)
- atlas-data service (REST API for maps, equipables, skills)
- atlas-maps service (REST API)
- atlas-rates service (REST API)
- atlas-party-quests service (REST API)
- atlas-pets service (REST API)

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| `TRACE_ENDPOINT` | OpenTelemetry collector endpoint (host:port) |
| `LOG_LEVEL` | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| `BASE_SERVICE_URL` | Base URL for REST API calls (scheme://host:port/api/) |
| `BOOTSTRAP_SERVERS` | Kafka bootstrap servers (host:port) |
| `COMMAND_TOPIC_CHARACTER_CHAT` | Kafka topic for receiving chat commands |
| `EVENT_TOPIC_CHARACTER_CHAT` | Kafka topic for emitting chat events |
| `COMMAND_TOPIC_SAGA` | Kafka topic for emitting saga commands |
| `COMMAND_TOPIC_CHARACTER_BUFF` | Kafka topic for emitting buff commands |
| `COMMAND_TOPIC_MONSTER` | Kafka topic for emitting monster commands |
| `COMMAND_TOPIC_PARTY_QUEST` | Kafka topic for emitting party quest commands |
| `COMMAND_TOPIC_MAP` | Kafka topic for emitting map commands |
| `COMMAND_TOPIC_PET` | Kafka topic for emitting pet commands |
| `COMMAND_TOPIC_REACTOR` | Kafka topic for emitting reactor commands |
| `CHAT_CAPTURE_RETENTION_SECONDS` | Max age of a retained chat line, in seconds (default 900) |
| `CHAT_CAPTURE_MAX_LINES` | Max chat lines retained per character (default 200) |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
