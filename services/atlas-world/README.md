# atlas-world
Mushroom game World Service

## Overview

A RESTful resource which provides world services.

## Environment

- JAEGER_HOST - Jaeger [host]:[port]
- LOG_LEVEL - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace
- BOOTSTRAP_SERVERS - Kafka [host]:[port]
- BASE_SERVICE_URL - [scheme]://[host]:[port]/api/
- COMMAND_TOPIC_CHANNEL_STATUS - Kafka Topic for issuing Channel Service commands.
  - Used for requesting started channel services to identify status.

## API

### Header

All RESTful requests require the supplied header information to identify the server instance.

```
TENANT_ID:083839c6-c47c-42a6-9585-76492795d123
REGION:GMS
MAJOR_VERSION:83
MINOR_VERSION:1
```

### Requests

#### [GET] Get Worlds

```/api/worlds/```

Returns a list of all worlds. Each world now includes its channels as related resources following the JSON:API specification.

#### [GET] Get World By Id

```/api/worlds/{worldId}```

Returns a specific world by ID. The world now includes its channels as related resources following the JSON:API specification.

#### [GET] Get Channels For World

```/api/worlds/{worldId}/channels```

#### [GET] Get Channel By Id

```/api/worlds/{worldId}/channels/{channelId}```

#### [POST] Register Channel

```/api/worlds/{worldId}/channels```

#### [DELETE] Unregister Channel

```/api/worlds/{worldId}/channels/{channelId}```

### JSON:API Relationships

The world endpoints now follow the [JSON:API specification](https://jsonapi.org/) for relationships. This means that:

1. World resources include a `relationships` object with a `channels` relationship
2. Channel resources are included in the `included` array of the response
3. You can use the `include` query parameter to control which related resources are included

Example request to get a world with its channels:

```
GET /api/worlds/0?include=channels
```

Example response:

```json
{
  "data": {
    "type": "worlds",
    "id": "0",
    "attributes": {
      "name": "Scania",
      "flag": 0,
      "message": "Welcome to Scania!",
      "eventMessage": "",
      "recommended": true,
      "recommendedMessage": "This world is recommended for new players.",
      "capacityStatus": 0
    },
    "relationships": {
      "channels": {
        "data": [
          { "type": "channels", "id": "123e4567-e89b-12d3-a456-426614174000" },
          { "type": "channels", "id": "123e4567-e89b-12d3-a456-426614174001" }
        ]
      }
    }
  },
  "included": [
    {
      "type": "channels",
      "id": "123e4567-e89b-12d3-a456-426614174000",
      "attributes": {
        "worldId": 0,
        "channelId": 0,
        "ipAddress": "127.0.0.1",
        "port": 8585,
        "createdAt": "2023-06-01T12:00:00Z"
      }
    },
    {
      "type": "channels",
      "id": "123e4567-e89b-12d3-a456-426614174001",
      "attributes": {
        "worldId": 0,
        "channelId": 1,
        "ipAddress": "127.0.0.1",
        "port": 8586,
        "createdAt": "2023-06-01T12:01:00Z"
      }
    }
  ]
}
