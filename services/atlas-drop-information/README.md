# atlas-drop-information

Drop Information Service for Mushroom Game

## Overview

A RESTful service providing drop information for monsters and continents. Data is stored in a Postgres database and can be seeded from JSON files. Based on GMS v83 drop data provided by HeavenMS.

## Environment Variables

| Variable | Description |
|----------|-------------|
| `DB_HOST` | Postgres database host |
| `DB_PORT` | Postgres database port |
| `DB_NAME` | Postgres database name |
| `DB_USER` | Postgres user name |
| `DB_PASSWORD` | Postgres user password |
| `JAEGER_HOST_PORT` | Jaeger tracing endpoint (host:port) |
| `LOG_LEVEL` | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |

## API Endpoints

### Monster Drops

```
GET /api/monsters/{monsterId}/drops
```

Returns all drop entries for a specific monster.

### Continent Drops

```
GET /api/continents/drops
```

Returns all continent-wide drop entries (global drops that apply across continents).

### Reactor Drops

```
GET /api/reactors/{reactorId}/drops
```

Returns all drop entries for a specific reactor.

### Seed Data

```
POST /api/drops/seed
```

Seeds the database with drop data from JSON files. This operation:
1. Deletes all existing drop data for the current tenant
2. Loads and inserts data from JSON files in `/drops/monsters/`, `/drops/continents/`, and `/drops/reactors/`

Returns a summary of the seeding operation:
```json
{
  "monsterDrops": {
    "deletedCount": 1000,
    "createdCount": 1500,
    "failedCount": 0
  },
  "continentDrops": {
    "deletedCount": 4,
    "createdCount": 4,
    "failedCount": 0
  },
  "reactorDrops": {
    "deletedCount": 10,
    "createdCount": 15,
    "failedCount": 0
  }
}
```

## Seed Data Format

### Monster Drops (`/drops/monsters/*.json`)

```json
[
  {
    "monsterId": 100100,
    "itemId": 2000000,
    "minimumQuantity": 1,
    "maximumQuantity": 5,
    "questId": 0,
    "chance": 50000
  }
]
```

### Continent Drops (`/drops/continents/*.json`)

```json
[
  {
    "continentId": -1,
    "itemId": 4001126,
    "minimumQuantity": 1,
    "maximumQuantity": 2,
    "chance": 8000
  }
]
```

Note: `continentId` of `-1` indicates a global drop that applies to all continents.

### Reactor Drops (`/drops/reactors/*.json`)

Reactor drops use JSON:API format:

```json
{
  "data": [
    {
      "type": "reactor-drops",
      "attributes": {
        "reactorId": 1001,
        "itemId": 4001126,
        "questId": 0,
        "chance": 50000
      }
    }
  ]
}
```
