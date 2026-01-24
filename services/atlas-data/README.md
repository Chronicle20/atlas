# atlas-data

A RESTful resource providing static game data for Mushroom game clients.

## Overview

The atlas-data service serves static game data parsed from XML data files. It is tenant-aware, supporting tenant-specific data that supersedes region-based defaults. Data is loaded from XML files and stored in PostgreSQL, with an in-memory registry for caching.

## External Dependencies

- PostgreSQL database
- Kafka (for receiving data upload commands)
- Jaeger (for tracing)

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| DB_USER | PostgreSQL username |
| DB_PASSWORD | PostgreSQL password |
| DB_HOST | PostgreSQL host |
| DB_PORT | PostgreSQL port |
| DB_NAME | PostgreSQL database name |
| JAEGER_HOST_PORT | Jaeger OTLP gRPC endpoint (host:port) |
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| REST_PORT | Port for the REST API server |
| COMMAND_TOPIC_DATA | Kafka topic for data commands |
| ZIP_DIR | Directory for storing uploaded ZIP files |

## Documentation

- [Domain Models](docs/domain.md)
- [Kafka Integration](docs/kafka.md)
- [REST API](docs/rest.md)
- [Storage](docs/storage.md)
