# Storage

This service does not use a relational database. All persistent domain state is owned by external services and communicated via Kafka commands.

## Redis

The character location registry and the catch-item delay registry use Redis via the `atlas-redis` `TenantRegistry` abstraction.

| Key Prefix | Value Type | Description |
|------------|------------|-------------|
| `consumable-map-character` | field.Model | Maps character IDs to their current field context (world, channel, map, instance), scoped per tenant |
| `consumable-catch-delay` | bool | Maps `(characterId, itemId)` to a TTL-bound cooldown marker enforcing a catch item's WZ useDelay, scoped per tenant |

The character location registry is populated from character status events (LOGIN, LOGOUT, MAP_CHANGED, CHANNEL_CHANGED). The catch-delay registry entries expire via their TTL. Data persists across service restarts via Redis.

## External Data

All other state is fetched on demand from external services via REST:

- Character data (atlas-characters)
- Character location data (atlas-maps)
- Inventory data (atlas-inventory)
- Pet data (atlas-pets)
- Reference data: consumable, equipable, cash item, map, portal, drop position (atlas-data)
