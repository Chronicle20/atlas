# atlas-portal-actions

Portal script execution service that processes JSON-based rules to control portal entry behaviors.

## Overview

The `atlas-portal-actions` service manages portal scripts that determine whether characters can use portals and what operations should be executed when they do. Scripts consist of rules with conditions that are evaluated in order, with the first matching rule determining the outcome.

## REST API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/portals/scripts` | Get all portal scripts |
| GET | `/api/portals/scripts/{scriptId}` | Get a specific portal script by ID |
| GET | `/api/portals/{portalId}/scripts` | Get portal script by portal ID |
| POST | `/api/portals/scripts` | Create a new portal script |
| PATCH | `/api/portals/scripts/{scriptId}` | Update an existing portal script |
| DELETE | `/api/portals/scripts/{scriptId}` | Delete a portal script |
| POST | `/api/portals/scripts/seed` | Seed portal scripts from filesystem |

## Kafka Topics

### Consumed Topics

| Environment Variable | Description |
|---------------------|-------------|
| `COMMAND_TOPIC_PORTAL_ACTIONS` | Receives portal entry commands |

#### Portal Entry Command

```json
{
  "worldId": 0,
  "channelId": 1,
  "mapId": 100000000,
  "portalId": 0,
  "type": "ENTER",
  "body": {
    "characterId": 12345,
    "portalName": "east00"
  }
}
```

### Produced Topics

| Environment Variable | Description |
|---------------------|-------------|
| `EVENT_TOPIC_CHARACTER_STATUS` | Sends character status events (stat changed to enable actions) |
| `COMMAND_TOPIC_CHARACTER` | Sends character commands (change map) |

#### Stat Changed Event (Enable Actions)

```json
{
  "characterId": 12345,
  "type": "STAT_CHANGED",
  "worldId": 0,
  "body": {
    "channelId": 1,
    "exclRequestSent": true
  }
}
```

#### Change Map Command

```json
{
  "worldId": 0,
  "characterId": 12345,
  "type": "CHANGE_MAP",
  "body": {
    "channelId": 1,
    "mapId": 200000000,
    "portalId": 0
  }
}
```

## Portal Script Format

Portal scripts are JSON files that define rules for portal entry:

```json
{
  "portalId": "portal_name",
  "mapId": 100000000,
  "description": "Human-readable description",
  "rules": [
    {
      "id": "rule_1",
      "conditions": [
        {
          "type": "level",
          "operator": ">=",
          "value": "30"
        }
      ],
      "onMatch": {
        "allow": true,
        "operations": [
          {
            "type": "warp",
            "params": {
              "mapId": "200000000",
              "portalId": "0"
            }
          }
        ]
      }
    },
    {
      "id": "default",
      "conditions": [],
      "onMatch": {
        "allow": false,
        "operations": []
      }
    }
  ]
}
```

### Condition Types

| Type | Description | Operators |
|------|-------------|-----------|
| `level` | Character level | `==`, `!=`, `<`, `<=`, `>`, `>=` |
| `job` | Character job ID | `==`, `!=` |
| `quest_state` | Quest completion state | `==`, `!=` |
| `item` | Item possession check | `has`, `not_has` |
| `map` | Current map check | `==`, `!=` |

### Operation Types

| Type | Description | Parameters |
|------|-------------|------------|
| `warp` | Warp character to map | `mapId`, `portalId` |
| `message` | Display message | `text`, `type` |
| `enable_actions` | Enable character actions | none |

## Configuration

| Environment Variable | Description | Default |
|---------------------|-------------|---------|
| `REST_PORT` | HTTP server port | - |
| `PORTAL_SCRIPTS_DIR` | Directory containing portal script JSON files | `/scripts/portals` |
| `COMMAND_TOPIC_PORTAL_ACTIONS` | Kafka topic for portal commands | - |
| `EVENT_TOPIC_CHARACTER_STATUS` | Kafka topic for character status events | - |
| `COMMAND_TOPIC_CHARACTER` | Kafka topic for character commands | - |

## Architecture

```
atlas-portal-actions/
├── atlas.com/portal/
│   ├── main.go              # Application entry point
│   ├── character/           # Character command producers
│   ├── database/            # Database connection
│   ├── kafka/               # Kafka infrastructure
│   ├── logger/              # Logging setup
│   ├── rest/                # REST infrastructure
│   ├── script/              # Core domain package
│   │   ├── model.go         # Domain models
│   │   ├── builder.go       # Builder patterns
│   │   ├── entity.go        # GORM entity
│   │   ├── provider.go      # Read operations
│   │   ├── administrator.go # Write operations
│   │   ├── processor_db.go  # Script processor
│   │   ├── resource.go      # REST handlers
│   │   ├── rest.go          # REST models
│   │   ├── consumer.go      # Kafka consumer
│   │   ├── evaluator.go     # Condition evaluation
│   │   ├── executor.go      # Operation execution
│   │   ├── loader.go        # Filesystem loader
│   │   └── seed.go          # Seed data loading
│   ├── service/             # Service lifecycle
│   └── tracing/             # OpenTelemetry setup
├── scripts/                 # Portal script JSON files
└── docs/                    # Documentation
```

## Processing Flow

1. Portal entry command received via Kafka
2. Load portal script from database by portal ID
3. Evaluate rules in order (first match wins)
4. If no rules match, deny entry by default
5. Execute matched rule's operations
6. Enable character actions
