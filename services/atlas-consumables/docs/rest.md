# REST Integration

This service does not expose REST endpoints. All external communication occurs via Kafka messaging.

The service makes outbound REST calls to fetch data from other services:

## Outbound REST Calls

### Character Service

- `GET /characters/{id}` - Fetch character by ID
- `GET /characters/{id}?include=inventory` - Fetch character with inventory

### Pet Service

- `GET /pets/{id}` - Fetch pet by ID
- `GET /pets?ownerId={ownerIds}` - Fetch pets by owner

### Consumable Data Service

- `GET /consumables/{id}` - Fetch consumable template

### Equipable Data Service

- `GET /equipables/{id}` - Fetch equipable template

### Map Data Service

- `GET /maps/{id}` - Fetch map template

### Portal Service

- `GET /portals?mapId={mapId}` - Fetch map portals

### Monster Drop Position Service

- `GET /monster-drops/positions?mapId={mapId}` - Fetch drop positions

### Monster Service

- `POST /monsters` - Create monster instance
