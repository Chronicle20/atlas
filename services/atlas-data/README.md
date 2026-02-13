# atlas-data

A RESTful resource providing static game data for Mushroom game clients.

## Overview

The atlas-data service serves static game data parsed from XML data files. It is tenant-aware, supporting tenant-specific data isolation. Data is loaded from WZ/XML files, stored in PostgreSQL as JSON documents, and served via JSON:API compliant REST endpoints.

## External Dependencies

- PostgreSQL database
- Kafka (for internal worker dispatch during data processing)
- Jaeger (for tracing)

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| DB_USER | PostgreSQL username |
| DB_PASSWORD | PostgreSQL password |
| DB_HOST | PostgreSQL host |
| DB_PORT | PostgreSQL port |
| DB_NAME | PostgreSQL database name |
| BOOTSTRAP_SERVERS | Kafka broker address |
| JAEGER_HOST_PORT | Jaeger OTLP gRPC endpoint (host:port) |
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| REST_PORT | Port for the REST API server |
| COMMAND_TOPIC_DATA | Kafka topic for data processing commands |
| ZIP_DIR | Root directory for WZ data files |

## Documentation

- [Domain Models](docs/domain.md)
- [Kafka Integration](docs/kafka.md)
- [REST API](docs/rest.md)
- [Storage](docs/storage.md)
