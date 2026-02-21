# atlas-configurations

Configuration management service for the Atlas platform.

## Overview

This service provides centralized management of configuration templates, tenants, and service configurations. Templates define version-specific configuration schemas that tenants derive from. The service supports automatic seeding of template data on startup.

## External Dependencies

- PostgreSQL database for persistent storage
- OpenTelemetry-compatible collector for distributed tracing (OTLP gRPC)

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| `TRACE_ENDPOINT` | OpenTelemetry collector endpoint (OTLP gRPC) |
| `LOG_LEVEL` | Logging level (Panic / Fatal / Error / Warn / Info / Debug / Trace) |
| `DB_USER` | PostgreSQL database username |
| `DB_PASSWORD` | PostgreSQL database password |
| `DB_HOST` | PostgreSQL database host |
| `DB_PORT` | PostgreSQL database port |
| `DB_NAME` | PostgreSQL database name |
| `REST_PORT` | Port for HTTP server |
| `SEED_DATA_PATH` | Path to seed data directory (default: `/seed-data`) |
| `SEED_ENABLED` | Enable/disable automatic seeding on startup (default: `true`) |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST API](docs/rest.md)
- [Storage](docs/storage.md)
