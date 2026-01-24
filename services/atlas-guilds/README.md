# atlas-guilds

Manages guild lifecycle, membership, titles, and bulletin board threads for game characters.

## Overview

This service handles guild creation, member management, emblem customization, title configuration, and guild bulletin board functionality. It coordinates guild creation agreements among party members and processes member status updates based on character login/logout events.

## External Dependencies

- **PostgreSQL**: Persistent storage for guilds, members, titles, threads, replies, and character-guild mappings
- **Kafka**: Asynchronous command/event messaging for guild operations, thread management, character status, and invite handling
- **Jaeger**: Distributed tracing

## Runtime Configuration

### General
- `JAEGER_HOST_PORT` - Jaeger host:port for distributed tracing
- `LOG_LEVEL` - Logging level (Panic / Fatal / Error / Warn / Info / Debug / Trace)

### Database
- `DB_USER` - PostgreSQL user name
- `DB_PASSWORD` - PostgreSQL user password
- `DB_HOST` - PostgreSQL database host
- `DB_PORT` - PostgreSQL database port
- `DB_NAME` - PostgreSQL database name

### Kafka
- `BOOTSTRAP_SERVERS` - Kafka host:port
- `COMMAND_TOPIC_GUILD` - Topic for guild commands
- `COMMAND_TOPIC_GUILD_THREAD` - Topic for thread commands
- `COMMAND_TOPIC_INVITE` - Topic for invite commands
- `EVENT_TOPIC_CHARACTER_STATUS` - Topic for character status events
- `EVENT_TOPIC_INVITE_STATUS` - Topic for invite status events
- `EVENT_TOPIC_GUILD_STATUS` - Topic for guild status events
- `EVENT_TOPIC_GUILD_THREAD_STATUS` - Topic for thread status events

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
