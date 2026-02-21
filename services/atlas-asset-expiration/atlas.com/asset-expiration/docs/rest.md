# REST Interface

## Endpoints

This service does not expose any REST endpoints. It operates as a background Kafka consumer and producer.

The service acts as a REST client to the following external services:

| Service | Root URL Key | Resources Consumed |
|---------|-------------|-------------------|
| atlas-data | `DATA` | `equipment/{id}`, `consumables/{id}`, `setup/{id}`, `etc/{id}` |
| atlas-inventory | `INVENTORY` | `characters/{id}/inventory`, `characters/{id}/inventory/compartments/{id}/assets` |
| atlas-storage | `STORAGE` | `storage/accounts/{id}/assets?worldId={id}` |
| atlas-cashshop | `CASHSHOP` | `accounts/{id}/cash-shop/inventory/compartments` |
