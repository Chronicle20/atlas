# atlas-world
Mushroom game World Service

## Overview

A RESTful resource which provides world services.

## Environment

- JAEGER_HOST - Jaeger [host]:[port]
- LOG_LEVEL - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace
- CONFIG_FILE - Location of service configuration file.
- BOOTSTRAP_SERVERS - Kafka [host]:[port]
- TOPIC_CHANNEL_SERVICE - Kafka Topic for transmitting Channel Status Events

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

```/api/wrg/worlds/```

#### [GET] Get World By Id

```/api/wrg/worlds/{worldId}```

#### [GET] Get Channels For World

```/api/wrg/worlds/{worldId}/channels```

#### [GET] Get Channel By Id

```/api/wrg/worlds/{worldId}/channels/{channelId}```

#### [POST] Register Channel

```/api/wrg/worlds/{worldId}/channels```

#### [DELETE] Unregister Channel

```/api/wrg/worlds/{worldId}/channels/{channelId}```