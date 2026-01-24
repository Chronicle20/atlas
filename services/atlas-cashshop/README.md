# atlas-cashshop

Cash shop management service for game accounts and characters.

## Overview

Manages cash shop functionality including wallets, wishlists, inventories, compartments, assets, and items. Handles currency balances (credit, points, prepaid), character wishlists, and cash item storage organized by character type (Explorer, Cygnus, Legend).

## External Dependencies

- **PostgreSQL**: Persistent storage for wallets, wishlists, inventories, compartments, assets, and items
- **Kafka**: Message broker for commands and events
- **Jaeger**: Distributed tracing

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| JAEGER_HOST_PORT | Jaeger host:port |
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| REST_PORT | Port for the REST server |
| DB_USER | Postgres user name |
| DB_PASSWORD | Postgres user password |
| DB_HOST | Postgres database host |
| DB_PORT | Postgres database port |
| DB_NAME | Postgres database name |
| BOOTSTRAP_SERVERS | Kafka host:port |
| EVENT_TOPIC_ACCOUNT_STATUS | Kafka topic for account status events |
| EVENT_TOPIC_CHARACTER_STATUS | Kafka topic for character status events |
| COMMAND_TOPIC_CASH_SHOP | Kafka topic for cash shop commands |
| EVENT_TOPIC_CASH_SHOP_STATUS | Kafka topic for cash shop status events |
| COMMAND_TOPIC_CASH_COMPARTMENT | Kafka topic for cash compartment commands |
| EVENT_TOPIC_CASH_COMPARTMENT_STATUS | Kafka topic for cash compartment status events |
| EVENT_TOPIC_CASH_INVENTORY_STATUS | Kafka topic for cash inventory status events |
| COMMAND_TOPIC_CASH_ITEM | Kafka topic for cash item commands |
| STATUS_TOPIC_CASH_ITEM | Kafka topic for cash item status events |
| EVENT_TOPIC_WALLET_STATUS | Kafka topic for wallet status events |
| COMMAND_TOPIC_WALLET | Kafka topic for wallet commands |
| EVENT_TOPIC_WISHLIST_STATUS | Kafka topic for wishlist status events |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
