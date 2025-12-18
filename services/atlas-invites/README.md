# atlas-invites
Mushroom game invites Service

## Overview

A RESTful resource which provides invites services.

## Environment

- JAEGER_HOST_PORT - Jaeger [host]:[port] for tracing
- LOG_LEVEL - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace
- REST_PORT - Port for the REST server
- BOOTSTRAP_SERVERS - Kafka bootstrap servers
- COMMAND_TOPIC_INVITE - Kafka topic for invite commands
- EVENT_TOPIC_INVITE_STATUS - Kafka topic for invite status events

## API

### Header

All RESTful requests require the supplied header information to identify the server instance.

```
TENANT_ID:083839c6-c47c-42a6-9585-76492795d123
REGION:GMS
MAJOR_VERSION:83
MINOR_VERSION:1
```

### Endpoints

#### GET /characters/{characterId}/invites

Retrieves all invites for a specific character.

**Response**

```json
{
  "data": [
    {
      "type": "invites",
      "id": "1",
      "attributes": {
        "type": "BUDDY",
        "referenceId": 12345,
        "originatorId": 1000,
        "targetId": 2000,
        "age": "2023-04-01T12:34:56Z"
      }
    }
  ]
}
```

## Kafka Message Structure

### Command Messages

Commands are sent to the `COMMAND_TOPIC_INVITE` topic.

#### Command Types
- CREATE - Create a new invite
- ACCEPT - Accept an invite
- REJECT - Reject an invite

#### Invite Types
- BUDDY - Buddy invite
- FAMILY - Family invite
- FAMILY_SUMMON - Family summon invite
- MESSENGER - Messenger invite
- TRADE - Trade invite
- PARTY - Party invite
- GUILD - Guild invite
- ALLIANCE - Alliance invite

#### Command Message Format

```json
{
  "worldId": 0,
  "inviteType": "BUDDY",
  "type": "CREATE",
  "body": {
    "field1": "value1",
    "field2": "value2"
  }
}
```

Note: The body structure depends on the command type as shown below.

##### CREATE Command Body
```json
{
  "originatorId": 1000,
  "targetId": 2000,
  "referenceId": 12345
}
```

##### ACCEPT Command Body
```json
{
  "targetId": 2000,
  "referenceId": 12345
}
```

##### REJECT Command Body
```json
{
  "targetId": 2000,
  "originatorId": 1000
}
```

### Status Event Messages

Status events are sent to the `EVENT_TOPIC_INVITE_STATUS` topic.

#### Event Types
- CREATED - Invite created
- ACCEPTED - Invite accepted
- REJECTED - Invite rejected

#### Status Event Message Format

```json
{
  "worldId": 0,
  "inviteType": "BUDDY",
  "referenceId": 12345,
  "type": "CREATED",
  "body": {
    "field1": "value1",
    "field2": "value2"
  }
}
```

Note: The body structure depends on the event type as shown below.

##### CREATED Event Body
```json
{
  "originatorId": 1000,
  "targetId": 2000
}
```

##### ACCEPTED Event Body
```json
{
  "originatorId": 1000,
  "targetId": 2000
}
```

##### REJECTED Event Body
```json
{
  "originatorId": 1000,
  "targetId": 2000
}
```
