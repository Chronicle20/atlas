# Storage

This service does not use a relational database. All persistent domain state is owned by external services and communicated via Kafka commands.

## Redis

The character location registry uses Redis via the `atlas-redis` `TenantRegistry` abstraction.

| Key Prefix | Value Type | Description |
|------------|------------|-------------|
| `consumable-map-character` | field.Model | Maps character IDs to their current field context (world, channel, map, instance), scoped per tenant |

The registry is populated from character status events (LOGIN, LOGOUT, MAP_CHANGED, CHANNEL_CHANGED). Data persists across service restarts via Redis.

## External Data

All other state is fetched on demand from external services via REST:

- Character data (atlas-characters)
- Inventory data (atlas-inventory)
- Pet data (atlas-pets)
- Reference data: consumable, equipable, cash item, map, portal, drop position (atlas-data)
