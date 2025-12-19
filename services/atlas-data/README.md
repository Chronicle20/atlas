# atlas-data
Mushroom game data Service

## Overview

A RESTful resource which provides data services for Mushroom game data. The service is tenant aware, and will try and read the supplied directory for both tenant and region game data. Tenant data will supersede region game data.

## Environment

- JAEGER_HOST_PORT - Jaeger [host]:[port]
- LOG_LEVEL - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace
- GAME_DATA_ROOT_DIR - Root directory of game data
- REST_PORT - Port for the REST API server

## API

### Header

All RESTful requests require the supplied header information to identify the server instance.

```
TENANT_ID:083839c6-c47c-42a6-9585-76492795d123
REGION:GMS
MAJOR_VERSION:83
MINOR_VERSION:1
```

### Common Query Parameters

All endpoints support the following JSON API query parameters:
- `fields[resourceType]` - Comma-separated list of fields to include in the response
- `include` - Comma-separated list of related resources to include

### Requests

#### Cash Items

##### [GET] Get Cash Item

```/api/data/cash/items/{itemId}```

#### Character Templates

##### [GET] Get Character Templates

```/api/data/characters/templates```

#### Commodity Items

##### [GET] Get Commodity Item

```/api/data/commodity/items/{itemId}```

#### Consumables

##### [GET] Get All Consumables

```/api/data/consumables```

Query Parameters:
- `filter[rechargeable]` - Filter consumables by rechargeable status (true/false)

##### [GET] Get Consumable

```/api/data/consumables/{itemId}```

#### Equipment

##### [GET] Get Equipment Statistics

```/api/data/equipment/{equipmentId}```

##### [GET] Get Equipment Slots

```/api/data/equipment/{equipmentId}/slots```

#### ETC Items

##### [GET] Get ETC Item

```/api/data/etcs/{itemId}```

#### Maps

##### [GET] Get All Maps

```/api/data/maps```

##### [GET] Get Map Information

```/api/data/maps/{mapId}```

##### [GET] Get Portal Information In Map

```/api/data/maps/{mapId}/portals```

##### [GET] Get Portal Information In Map By Portal Name

```/api/data/maps/{mapId}/portals?name={name}```

##### [GET] Get Specific Portal Information In Map

```/api/data/maps/{mapId}/portals/{portalId}```

##### [GET] Get Reactor Information In Map

```/api/data/maps/{mapId}/reactors```

##### [GET] Get NPC Information In Map

```/api/data/maps/{mapId}/npcs```

##### [GET] Get Specific NPC Information In Map

```/api/data/maps/{mapId}/npcs/{npcId}```

##### [GET] Get NPC Information In Map By Object Id

```/api/data/maps/{mapId}/npcs?objectId={objectId}```

##### [GET] Get Monster Information In Map

```/api/data/maps/{mapId}/monsters```

##### [POST] Get Drop Position In Map

```/api/data/maps/{mapId}/drops/position```

Request Body:
```json
{
  "initialX": 0,
  "initialY": 0,
  "fallbackX": 0,
  "fallbackY": 0
}
```

##### [POST] Get Foothold Below Position In Map

```/api/data/maps/{mapId}/footholds/below```

Request Body:
```json
{
  "x": 0,
  "y": 0
}
```

#### Monsters

##### [GET] Get Monster Information

```/api/data/monsters/{monsterId}```

##### [GET] Get Lose Items For Monster

```/api/data/monsters/{monsterId}/loseItems```

#### Pets

##### [GET] Get Pet Information

```/api/data/pets/{petId}```

#### Reactors

##### [GET] Get Reactor Information

```/api/data/reactors/{reactorId}```

#### Setups

##### [GET] Get Setup Information

```/api/data/setups/{setupId}```

#### Skills

##### [GET] Get Skill Information

```/api/data/skills/{skillId}```
