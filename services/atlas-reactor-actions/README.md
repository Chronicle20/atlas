# Atlas Reactor Actions

Service for handling JSON-based reactor scripting. Receives hit/trigger commands from `atlas-reactors`, loads the appropriate script, evaluates rules, and executes operations via saga orchestration.

## Overview

Reactors have two touch points:
1. **Hit** (`hitRules`) - When a reactor is attacked/hit by a player
2. **Trigger** (`actRules`) - When a reactor reaches its final state and activates

## REST Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/reactors/actions` | List all reactor scripts |
| GET | `/api/reactors/actions/{scriptId}` | Get script by ID |
| GET | `/api/reactors/{reactorId}/actions` | Get script by reactor classification ID |
| POST | `/api/reactors/actions` | Create a new script |
| PATCH | `/api/reactors/actions/{scriptId}` | Update an existing script |
| DELETE | `/api/reactors/actions/{scriptId}` | Delete a script |
| POST | `/api/reactors/actions/seed` | Seed scripts from filesystem |

## Kafka Commands (Consumed)

| Topic | Command Type | Description |
|-------|--------------|-------------|
| `COMMAND_TOPIC_REACTOR_ACTIONS` | `HIT` | Reactor was hit by player |
| `COMMAND_TOPIC_REACTOR_ACTIONS` | `TRIGGER` | Reactor reached final state and activated |

### HIT Command Body

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 100000000,
  "reactorId": 123,
  "classification": "2000",
  "reactorName": "box01",
  "reactorState": 1,
  "x": 100,
  "y": 200,
  "type": "HIT",
  "body": {
    "characterId": 12345,
    "skillId": 0,
    "isSkill": false
  }
}
```

### TRIGGER Command Body

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 100000000,
  "reactorId": 123,
  "classification": "2000",
  "reactorName": "box01",
  "reactorState": 2,
  "x": 100,
  "y": 200,
  "type": "TRIGGER",
  "body": {
    "characterId": 12345
  }
}
```

## Script Format

Scripts are stored as JSON files in `/scripts/reactors/`:

```json
{
  "reactorId": "2000",
  "description": "Maple Island Box - drops various items",
  "hitRules": [],
  "actRules": [
    {
      "id": "drop_items",
      "conditions": [],
      "operations": [
        {
          "type": "drop_items",
          "params": {
            "meso": "true",
            "minMeso": "2",
            "maxMeso": "8",
            "mesoRange": "15",
            "item": "1"
          }
        }
      ]
    }
  ]
}
```

## Supported Operations

| Operation | Description | Parameters |
|-----------|-------------|------------|
| `drop_items` | Drop items at reactor location | `meso`, `minMeso`, `maxMeso`, `mesoRange`, `item` |
| `spawn_monster` | Spawn monster at location | `monsterId`, `count` |
| `spray_items` | Spray items around reactor | (none) |
| `weaken_area_boss` | Weaken a boss monster | `monsterId`, `message` |
| `move_environment` | Move map environment object | `name`, `value` |
| `kill_all_monsters` | Kill all monsters in map | (none) |
| `drop_message` | Send message to character | `type`, `message` |

## Supported Conditions

| Condition | Description | Operators |
|-----------|-------------|-----------|
| `reactor_state` | Current reactor state | `=`, `!=`, `>`, `<`, `>=`, `<=` |

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DB_HOST` | PostgreSQL host | - |
| `DB_PORT` | PostgreSQL port | `5432` |
| `DB_NAME` | Database name | `atlas-reactor-actions` |
| `DB_USER` | Database user | - |
| `DB_PASSWORD` | Database password | - |
| `BOOTSTRAP_SERVERS` | Kafka broker addresses | - |
| `COMMAND_TOPIC_REACTOR_ACTIONS` | Command topic name | - |
| `REACTOR_ACTIONS_DIR` | Scripts directory path | `/scripts/reactors` |
| `REST_PORT` | REST API port | `8080` |
| `LOG_LEVEL` | Logging level | `info` |

## Development

### Local Setup

1. Copy `.env` and adjust values for local development
2. Ensure PostgreSQL and Kafka are accessible
3. Run: `go run main.go`

### Seeding Scripts

Call the seed endpoint to load scripts from the filesystem:

```bash
curl -X POST -H "TENANT_ID: your-tenant-id" http://localhost:8080/api/reactors/actions/seed
```

## Architecture

```
atlas-channel → atlas-reactors → atlas-reactor-actions → atlas-saga-orchestrator
     ↓                ↓                    ↓
  (hit packet)    (state mgmt)      (script eval)         (execute ops)
```
