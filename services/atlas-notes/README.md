# atlas-notes

Manages notes sent between characters.

## Overview

A RESTful service that provides note storage and retrieval for characters. Notes are messages sent from one character to another, stored persistently, and accessible via REST API or Kafka commands.

## External Dependencies

- PostgreSQL database for note persistence
- Kafka for event publishing and command consumption
- Jaeger for distributed tracing

## Runtime Configuration

### Logging and Tracing
- `JAEGER_HOST` - Jaeger host:port
- `LOG_LEVEL` - Logging level (Panic / Fatal / Error / Warn / Info / Debug / Trace)

### Database
- `DB_USER` - Database user
- `DB_PASSWORD` - Database password
- `DB_HOST` - Database host
- `DB_PORT` - Database port
- `DB_NAME` - Database name

### Kafka
- `BOOTSTRAP_SERVERS` - Kafka bootstrap servers
- `EVENT_TOPIC_NOTE_STATUS` - Topic for note status events
- `COMMAND_TOPIC_NOTE` - Topic for note commands
- `EVENT_TOPIC_CHARACTER_STATUS` - Topic for character status events
- `COMMAND_TOPIC_SAGA` - Topic for saga commands

### REST
- `REST_PORT` - HTTP server port

## Documentation

- [Domain Model](docs/domain.md)
- [Kafka Integration](docs/kafka.md)
- [REST API](docs/rest.md)
- [Storage](docs/storage.md)
