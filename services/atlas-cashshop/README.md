# atlas-cashshop

Cash shop management service for game accounts and characters.

## Overview

Manages cash shop functionality including wallets, wishlists, and cash inventories. Currency balances (credit, points, prepaid) are tracked per account. Character wishlists reference commodities by serial number. Cash inventories are organized by character type (Explorer, Cygnus, Legend) into compartments, each containing flattened assets that hold all item data directly. Purchases (including gifts, cash packages, ring pairs, equip-slot extensions, and locker rebates), coupon codes, Cash Shop Surprise boxes, inventory capacity increases, and asset lifecycle (creation, release, expiration) are coordinated through Kafka commands and events. Purchase history is tracked per account/commodity, and cash shop administration (coupons, coupon batches) is exposed over REST.

## External Dependencies

- **PostgreSQL**: Persistent storage for wallets, wishlists, compartments, assets, coupons, coupon batches/redemptions, ring pairs, purchase records, and the outbox
- **Redis**: Coupon redemption rate limiting (fails open when unavailable)
- **Kafka**: Message broker for commands and events
- **Jaeger**: Distributed tracing
- **atlas-characters** (REST): Character data lookups (job type, account ID) and equip-slot extension writes
- **atlas-inventory** (REST): Character inventory data lookups (compartment capacities)
- **atlas-data** (REST): Commodity catalog lookups, pet template data lookups, and cash package lookups
- **atlas-pets** (REST): Pet creation for cash shop pet purchases
- **atlas-reward-pools** (REST): Reward selection for Cash Shop Surprise boxes
- **Configurations service** (REST): Tenant configuration including hourly expiration settings, coupon rate limits, and Surprise box template ids

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
| REDIS_URL | Redis address for the coupon rate limiter (defaults to localhost:6379) |
| REDIS_PASSWORD | Redis password for the coupon rate limiter |
| CHARACTERS | Base URL for the atlas-characters service |
| INVENTORY | Base URL for the atlas-inventory service |
| DATA | Base URL for the atlas-data service (commodity lookups, pet template lookups, cash package lookups) |
| PETS | Base URL for the atlas-pets service (pet creation on cash shop pet purchase) |
| GACHAPONS | Base URL for the atlas-reward-pools service (Cash Shop Surprise reward selection) |
| CONFIGURATIONS | Base URL for the configurations service (tenant config / hourly expirations / coupon rate limit / surprise box template ids) |
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
| COMMAND_TOPIC_COMPARTMENT | Kafka topic for character inventory compartment commands (capacity increase) |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
