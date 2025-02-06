# atlas-parties
Mushroom game parties Service

## Overview

A RESTful resource which provides parties services.

## Environment

- JAEGER_HOST - Jaeger [host]:[port]
- LOG_LEVEL - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace
- BASE_SERVICE_URL - [scheme]://[host]:[port]/api/
- BOOTSTRAP_SERVERS - Kafka [host]:[port]

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

#### [GET] Get Parties

```/api/parties```

#### [GET] Get Parties - By Member Id

```/api/parties?filter[members.id]={memberId}```

#### [POST] Create Party

```/api/parties```

#### [GET] Get Party - By Id

```/api/parties/{partyId}```

#### [PATCH] Update Party

```/api/parties/{partyId}```

#### [GET] Get Party Members

```/api/parties/{partyId}/members```

#### [POST] Add Party Member

```/api/parties/{partyId}/members```

#### [GET] Get Party Member

```/api/parties/{partyId}/members/{memberId}```