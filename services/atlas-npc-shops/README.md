# atlas-npc-shops
Mushroom game NPC Shops Service

## Overview

A RESTful service that provides NPC shop functionality for the Mushroom game. This service allows retrieving shop information for specific NPCs, including the commodities they sell with pricing details.

## Environment Variables

- `JAEGER_HOST_PORT` - Jaeger [host]:[port] for distributed tracing
- `LOG_LEVEL` - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace
- `REST_PORT` - Port on which the REST API will listen
- `DB_USER` - PostgreSQL database user
- `DB_PASSWORD` - PostgreSQL database password
- `DB_HOST` - PostgreSQL database host
- `DB_PORT` - PostgreSQL database port
- `DB_NAME` - PostgreSQL database name

## Dependencies

- Go 1.24.2
- PostgreSQL database
- Jaeger for distributed tracing
- Chronicle20/atlas-model
- Chronicle20/atlas-rest
- Chronicle20/atlas-tenant
- GORM for database operations
- Gorilla Mux for routing

## Setup

1. Ensure PostgreSQL is running and accessible
2. Set the required environment variables
3. Run the service using `go run atlas.com/npc/main.go` or build and run the Docker container

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

#### Get Shop by NPC ID

Retrieves shop information for a specific NPC, including all commodities sold by that NPC.

- **URL**: `/api/npcs/{npcId}/shop`
- **Method**: GET
- **URL Parameters**: 
  - `npcId` - The ID of the NPC
- **Response**: JSON object containing shop information and commodities

Example Response:
```json
{
  "data": {
    "type": "shops",
    "id": "shop-9000001",
    "attributes": {
      "npcId": 9000001,
      "commodities": [
        {
          "id": "550e8400-e29b-41d4-a716-446655440000",
          "templateId": 2000,
          "mesoPrice": 1000,
          "perfectPitchPrice": 0
        },
        {
          "id": "550e8400-e29b-41d4-a716-446655440001",
          "templateId": 2001,
          "mesoPrice": 1500,
          "perfectPitchPrice": 0
        }
      ]
    }
  }
}
```

#### Add Commodity to Shop

Adds a new commodity to an NPC's shop.

- **URL**: `/api/npcs/{npcId}/shop/commodities`
- **Method**: POST
- **URL Parameters**: 
  - `npcId` - The ID of the NPC
- **Request Body**: JSON object containing commodity details
  ```json
  {
    "data": {
      "type": "commodities",
      "id": "00000000-0000-0000-0000-000000000000",
      "attributes": {
        "templateId": 2002,
        "mesoPrice": 2000,
        "perfectPitchPrice": 0
      }
    }
  }
  ```
- **Response**: JSON object containing the created commodity
  ```json
  {
    "data": {
      "type": "commodities",
      "id": "550e8400-e29b-41d4-a716-446655440002",
      "attributes": {
        "templateId": 2002,
        "mesoPrice": 2000,
        "perfectPitchPrice": 0
      }
    }
  }
  ```

#### Update Commodity

Updates an existing commodity in a shop.

- **URL**: `/api/npcs/{npcId}/shop/commodities/{commodityId}`
- **Method**: PUT
- **URL Parameters**: 
  - `npcId` - The ID of the NPC
  - `commodityId` - The UUID of the commodity
- **Request Body**: JSON object containing updated commodity details
  ```json
  {
    "data": {
      "type": "commodities",
      "id": "00000000-0000-0000-0000-000000000000",
      "attributes": {
        "templateId": 2002,
        "mesoPrice": 2500,
        "perfectPitchPrice": 0
      }
    }
  }
  ```
- **Response**: JSON object containing the updated commodity
  ```json
  {
    "data": {
      "type": "commodities",
      "id": "550e8400-e29b-41d4-a716-446655440002",
      "attributes": {
        "templateId": 2002,
        "mesoPrice": 2500,
        "perfectPitchPrice": 0
      }
    }
  }
  ```

#### Remove Commodity

Removes a commodity from a shop.

- **URL**: `/api/npcs/{npcId}/shop/commodities/{commodityId}`
- **Method**: DELETE
- **URL Parameters**: 
  - `npcId` - The ID of the NPC
  - `commodityId` - The UUID of the commodity
- **Response**: No content (204)

## API Testing with Bruno

This repository includes a [Bruno](https://www.usebruno.com/) collection for testing the API endpoints. Bruno is an open-source API client that makes it easy to test and debug API requests.

To use the Bruno collection:

1. Install Bruno from https://www.usebruno.com/
2. Open Bruno and click "Open Collection"
3. Navigate to the `.bruno` directory in this repository
4. The collection will be loaded with all the requests

## Docker

The service can be run using Docker with the provided Dockerfile:

```bash
# Build the Docker image
./docker-build.sh

# Run the Docker container
docker run -p 8080:8080 \
  -e JAEGER_HOST_PORT=jaeger:6831 \
  -e LOG_LEVEL=Info \
  -e REST_PORT=8080 \
  -e DB_USER=postgres \
  -e DB_PASSWORD=postgres \
  -e DB_HOST=postgres \
  -e DB_PORT=5432 \
  -e DB_NAME=atlas_npc_shops \
  atlas-npc-shops
```
