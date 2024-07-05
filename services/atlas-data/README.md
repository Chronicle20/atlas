# atlas-data
Mushroom game data Service

## Overview

A RESTful resource which provides data services for Mushroom game data. The service is tenant aware, and will try and read the supplied directory for both tenant and region game data. Tenant data will supersede region game data.

## Environment

- JAEGER_HOST - Jaeger [host]:[port]
- LOG_LEVEL - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace
- GAME_DATA_ROOT_DIR - Root directory of game data

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

#### [GET] Get Default Equipment Statistics

```/api/gis/equipment/{itemId}```