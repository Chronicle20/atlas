# atlas-data
Mushroom game data Service

## Overview

A RESTful resource which provides data services for Mushroom game data. The service is tenant aware, and will try and read the supplied directory for both tenant and region game data. Tenant data will supersede region game data.

## Environment

- JAEGER_HOST_PORT - Jaeger [host]:[port]
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

#### [GET] Get Expected Equipment Slots

```/api/gis/equipment/{itemId}/slots```

#### [GET] Get Map Information

```/api/gis/maps/{mapId}```

#### [GET] Get Portal Information In Map

```/api/gis/maps/{mapId}/portals```

#### [GET] Get Portal Information In Map By Portal Name

```/api/gis/maps/{mapId}/portals?name={name}```

#### [GET] Get Specific Portal Information In Map

```/api/gis/maps/{mapId}/portals/{portalId}```

#### [GET] Get Reactor Information In Map

```/api/gis/maps/{mapId}/reactors```

#### [GET] Get NPC Information In Map

```/api/gis/maps/{mapId}/npcs```

#### [GET] Get Specific NPC Information In Map

```/api/gis/maps/{mapId}/npcs/{npcId}```

#### [GET] Get NPC Information In Map By Object Id

```/api/gis/maps/{mapId}/npcs?objectId={objectId}```

#### [GET] Get Monster Information In Map

```/api/gis/maps/{mapId}/monsters```

#### [POST] Get Drop Position In Map

```/api/gis/maps/{mapId}/dropPosition```

#### [GET] Get Monster Information

```/api/gis/monsters/{monsterId}```

#### [GET] Get Lose Items For Monster

```/api/gis/monsters/{monsterId}/loseItems```