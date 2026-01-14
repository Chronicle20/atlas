# atlas-equipables

Mushroom game equipables Service

## Overview

A RESTful resource which provides equipables services.

## Environment

- JAEGER_HOST - Jaeger [host]:[port]
- LOG_LEVEL - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace
- DB_USER - Postgres user name
- DB_PASSWORD - Postgres user password
- DB_HOST - Postgres Database host
- DB_PORT - Postgres Database port
- DB_NAME - Postgres Database name
- BASE_SERVICE_URL - [scheme]://[host]:[port]/api/

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

#### [POST] Create Equipable

```/api/equipables```

#### [POST] Create Random Stat Equipable

```/api/equipables?random=true```

#### [GET] Get Equipable By Id

```/api/equipables/{equipmentId}```

#### [PATCH] Update Equipable By Id

```/api/equipables/{equipmentId}```

#### [DELETE] Delete Equipable By Id

```/api/equipables/{equipmentId}```