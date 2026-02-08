# REST Integration

This service does not expose REST endpoints. All external communication occurs via Kafka messaging.

The service makes outbound REST calls to fetch data from other services:

## Outbound REST Calls

### Character Service (CHARACTERS)

- `GET /characters/{id}` - Fetch character by ID
- `GET /characters/{id}?include=inventory` - Fetch character with inventory

### Inventory Service (INVENTORY)

- `GET /characters/{id}/inventory` - Fetch full inventory for a character

### Pet Service (PETS)

- `GET /pets/{id}` - Fetch pet by ID
- `GET /characters/{id}/pets` - Fetch pets by owner

### Consumable Data Service (DATA)

- `GET /data/consumables/{id}` - Fetch consumable template data

### Equipable Data Service (DATA)

- `GET /data/equipment/{id}` - Fetch equipable template data

### Cash Item Data Service (DATA)

- `GET /data/cash/items/{id}` - Fetch cash item template data

### Map Data Service (DATA)

- `GET /data/maps/{id}` - Fetch map data (return map ID)

### Portal Data Service (DATA)

- `GET /data/maps/{id}/portals` - Fetch portals in a map

### Monster Drop Position Service (DATA)

- `POST /data/maps/{id}/drops/position` - Calculate drop position in a map

  Request body:
  ```json
  {
    "initialX": 0,
    "initialY": 0,
    "fallbackX": 0,
    "fallbackY": 0
  }
  ```

  Response:
  ```json
  {
    "x": 0,
    "y": 0
  }
  ```

### Monster Service (MONSTERS)

- `POST /worlds/{worldId}/channels/{channelId}/maps/{mapId}/monsters` - Create monster instance

  Request body:
  ```json
  {
    "monsterId": 0,
    "x": 0,
    "y": 0,
    "fh": 0,
    "team": 0
  }
  ```
