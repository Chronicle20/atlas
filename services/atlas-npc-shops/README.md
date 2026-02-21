# atlas-npc-shops

NPC Shops Service for the Atlas game platform.

## Overview

A RESTful service that provides NPC shop functionality. This service manages shop data, commodities sold by NPCs, and processes buy, sell, and recharge operations for characters interacting with NPC shops. It supports seeding shop data from JSON files on disk.

The service maintains a Redis-backed registry of characters currently browsing shops and automatically removes characters when they log out, change maps, or change channels. A Redis-backed per-tenant consumable cache avoids repeated REST calls to the data service for rechargeable item metadata.

## External Dependencies

- PostgreSQL database for persistent storage of shops and commodities
- Redis for character-shop registry and rechargeable consumable cache
- Kafka for command/event messaging
- Jaeger for distributed tracing
- atlas-data service for item reference data (consumable, equipable, setup, etc)
- atlas-character service for character data
- atlas-inventory service for inventory state
- atlas-skill service for character skill data (recharge calculations)

## Runtime Configuration

| Variable           | Description                                              |
|--------------------|----------------------------------------------------------|
| JAEGER_HOST_PORT   | Jaeger host:port for distributed tracing                 |
| LOG_LEVEL          | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace)  |
| REST_PORT          | Port for REST API                                        |
| DB_USER            | PostgreSQL database user                                 |
| DB_PASSWORD        | PostgreSQL database password                             |
| DB_HOST            | PostgreSQL database host                                 |
| DB_PORT            | PostgreSQL database port                                 |
| DB_NAME            | PostgreSQL database name                                 |
| BOOTSTRAP_SERVERS  | Kafka bootstrap server addresses                         |

## Documentation

- [Domain Models](docs/domain.md)
- [Kafka Integration](docs/kafka.md)
- [REST API](docs/rest.md)
- [Storage](docs/storage.md)
