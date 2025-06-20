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

#### [GET] Get World By Id

```/api/worlds/{worldId}```

#### [GET] Get Channels For World

```/api/worlds/{worldId}/channels```

#### [GET] Get Channel By Id

```/api/worlds/{worldId}/channels/{channelId}```

#### [POST] Register Channel

```/api/worlds/{worldId}/channels```

#### [DELETE] Unregister Channel

```/api/worlds/{worldId}/channels/{channelId}```