# atlas-fame
Mushroom game fame Service

## Overview

A resource which provides fame services.

## API

### REST API

The service does not expose any REST endpoints directly.

### Kafka Message API

#### Commands

##### Character Fame Command
- Topic: `COMMAND_TOPIC_CHARACTER`
- Type: `REQUEST_CHANGE_FAME`
- Structure:
  ```json
  {
    "transactionId": "uuid",
    "worldId": "world.Id",
    "characterId": "uint32",
    "type": "REQUEST_CHANGE_FAME",
    "body": {
      "actorId": "uint32",
      "actorType": "CHARACTER",
      "amount": "int8"
    }
  }
  ```

##### Fame Command
- Topic: `COMMAND_TOPIC_FAME`
- Type: `REQUEST_CHANGE`
- Structure:
  ```json
  {
    "transactionId": "uuid",
    "worldId": "world.Id",
    "characterId": "uint32",
    "type": "REQUEST_CHANGE",
    "body": {
      "channelId": "channel.Id",
      "mapId": "_map.Id",
      "targetId": "uint32",
      "amount": "int8"
    }
  }
  ```

#### Events

##### Fame Status Event
- Topic: `EVENT_TOPIC_FAME_STATUS`
- Type: `ERROR`
- Structure:
  ```json
  {
    "transactionId": "uuid",
    "worldId": "world.Id",
    "characterId": "uint32",
    "type": "ERROR",
    "body": {
      "channelId": "channel.Id",
      "error": "string"
    }
  }
  ```
- Error Types:
  - `NOT_TODAY` - Character has already given fame today
  - `NOT_THIS_MONTH` - Character has already given fame to this target this month
  - `INVALID_NAME` - Target character does not exist
  - `NOT_MINIMUM_LEVEL` - Character is not at least level 15
  - `UNEXPECTED` - An unexpected error occurred

## Environment

- JAEGER_HOST - Jaeger [host]:[port]
- LOG_LEVEL - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace
- DB_USER - Postgres user name
- DB_PASSWORD - Postgres user password
- DB_HOST - Postgres Database host
- DB_PORT - Postgres Database port
- DB_NAME - Postgres Database name
- BOOTSTRAP_SERVERS - Kafka [host]:[port]
- BASE_SERVICE_URL - [scheme]://[host]:[port]/api/
- COMMAND_TOPIC_CHARACTER - Kafka topic for character commands
- COMMAND_TOPIC_FAME - Kafka topic for fame commands
- EVENT_TOPIC_FAME_STATUS - Kafka topic for fame status events
- CHARACTERS - Character service URL
