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

Retrieves shop information for a specific NPC.

- **URL**: `/api/npcs/{npcId}/shop`
- **Method**: GET
- **URL Parameters**: 
  - `npcId` - The ID of the NPC
- **Query Parameters**:
  - `include` - Optional. Specify "commodities" to include the commodities associated with the shop in the response.
- **Response**: JSON object containing shop information and optionally commodities

Example Response (with include=commodities):
```json
{
  "data": {
    "type": "shops",
    "id": "shop-9000001",
    "attributes": {
      "npcId": 9000001,
      "recharger": true
    },
    "relationships": {
      "commodities": {
        "data": [
          {
            "type": "commodities",
            "id": "550e8400-e29b-41d4-a716-446655440000"
          },
          {
            "type": "commodities",
            "id": "550e8400-e29b-41d4-a716-446655440001"
          }
        ]
      }
    },
    "included": [
      {
        "type": "commodities",
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "attributes": {
          "templateId": 2000,
          "mesoPrice": 1000,
          "tokenPrice": 0,
          "unitPrice": 1.0,
          "slotMax": 100
        }
      },
      {
        "type": "commodities",
        "id": "550e8400-e29b-41d4-a716-446655440001",
        "attributes": {
          "templateId": 2001,
          "mesoPrice": 1500,
          "tokenPrice": 0,
          "unitPrice": 1.0,
          "slotMax": 100
        }
      }
    ]
  }
}
```

#### Add Commodity to Shop

Adds a new commodity to an NPC's shop.

- **URL**: `/api/npcs/{npcId}/shop/relationships/commodities`
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
        "tokenPrice": 0,
        "unitPrice": 1.0,
        "slotMax": 100
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
        "tokenPrice": 0,
        "unitPrice": 1.0,
        "slotMax": 100
      }
    }
  }
  ```

#### Update Commodity

Updates an existing commodity in a shop.

- **URL**: `/api/npcs/{npcId}/shop/relationships/commodities/{commodityId}`
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
        "tokenPrice": 0,
        "unitPrice": 1.0,
        "slotMax": 100
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
        "tokenPrice": 0,
        "unitPrice": 1.0,
        "slotMax": 100
      }
    }
  }
  ```

#### Remove Commodity

Removes a commodity from a shop.

- **URL**: `/api/npcs/{npcId}/shop/relationships/commodities/{commodityId}`
- **Method**: DELETE
- **URL Parameters**: 
  - `npcId` - The ID of the NPC
  - `commodityId` - The UUID of the commodity
- **Response**: No content (204)

#### Create Shop

Creates a new shop for a specific NPC with the provided commodities.

- **URL**: `/api/npcs/{npcId}/shop`
- **Method**: POST
- **URL Parameters**: 
  - `npcId` - The ID of the NPC
- **Request Body**: JSON object containing shop details with commodities
  ```json
  {
    "data": {
      "type": "shops",
      "id": "shop-9000001",
      "attributes": {
        "npcId": 9000001,
        "recharger": true
      },
      "relationships": {
        "commodities": {
          "data": [
            {
              "type": "commodities",
              "id": "00000000-0000-0000-0000-000000000000"
            },
            {
              "type": "commodities",
              "id": "00000000-0000-0000-0000-000000000000"
            }
          ]
        }
      }
    },
    "included": [
      {
        "type": "commodities",
        "id": "00000000-0000-0000-0000-000000000000",
        "attributes": {
          "templateId": 2000,
          "mesoPrice": 1000,
          "tokenPrice": 0,
          "unitPrice": 1.0,
          "slotMax": 100
        }
      },
      {
        "type": "commodities",
        "id": "00000000-0000-0000-0000-000000000000",
        "attributes": {
          "templateId": 2001,
          "mesoPrice": 1500,
          "tokenPrice": 0,
          "unitPrice": 1.0,
          "slotMax": 100
        }
      }
    ]
  }
  ```
- **Response**: JSON object containing the created shop with commodities
  ```json
  {
    "data": {
      "type": "shops",
      "id": "shop-9000001",
      "attributes": {
        "npcId": 9000001,
        "recharger": true
      },
      "relationships": {
        "commodities": {
          "data": [
            {
              "type": "commodities",
              "id": "550e8400-e29b-41d4-a716-446655440000"
            },
            {
              "type": "commodities",
              "id": "550e8400-e29b-41d4-a716-446655440001"
            }
          ]
        }
      },
      "included": [
        {
          "type": "commodities",
          "id": "550e8400-e29b-41d4-a716-446655440000",
          "attributes": {
            "templateId": 2000,
            "mesoPrice": 1000,
            "tokenPrice": 0,
            "unitPrice": 1.0,
            "slotMax": 100
          }
        },
        {
          "type": "commodities",
          "id": "550e8400-e29b-41d4-a716-446655440001",
          "attributes": {
            "templateId": 2001,
            "mesoPrice": 1500,
            "tokenPrice": 0,
            "unitPrice": 1.0,
            "slotMax": 100
          }
        }
      ]
    }
  }
  ```

#### Update Shop

Updates an existing shop for a specific NPC by deleting all existing commodities and recreating the shop with the provided commodities.

- **URL**: `/api/npcs/{npcId}/shop`
- **Method**: PUT
- **URL Parameters**: 
  - `npcId` - The ID of the NPC
- **Request Body**: JSON object containing shop details with commodities
  ```json
  {
    "data": {
      "type": "shops",
      "id": "shop-9000001",
      "attributes": {
        "npcId": 9000001,
        "recharger": true
      },
      "relationships": {
        "commodities": {
          "data": [
            {
              "type": "commodities",
              "id": "00000000-0000-0000-0000-000000000000"
            },
            {
              "type": "commodities",
              "id": "00000000-0000-0000-0000-000000000000"
            }
          ]
        }
      }
    },
    "included": [
      {
        "type": "commodities",
        "id": "00000000-0000-0000-0000-000000000000",
        "attributes": {
          "templateId": 2000,
          "mesoPrice": 1000,
          "tokenPrice": 0,
          "unitPrice": 1.0,
          "slotMax": 100
        }
      },
      {
        "type": "commodities",
        "id": "00000000-0000-0000-0000-000000000000",
        "attributes": {
          "templateId": 2001,
          "mesoPrice": 1500,
          "tokenPrice": 0,
          "unitPrice": 1.0,
          "slotMax": 100
        }
      }
    ]
  }
  ```
- **Response**: JSON object containing the updated shop with commodities
  ```json
  {
    "data": {
      "type": "shops",
      "id": "shop-9000001",
      "attributes": {
        "npcId": 9000001,
        "recharger": true
      },
      "relationships": {
        "commodities": {
          "data": [
            {
              "type": "commodities",
              "id": "550e8400-e29b-41d4-a716-446655440000"
            },
            {
              "type": "commodities",
              "id": "550e8400-e29b-41d4-a716-446655440001"
            }
          ]
        }
      },
      "included": [
        {
          "type": "commodities",
          "id": "550e8400-e29b-41d4-a716-446655440000",
          "attributes": {
            "templateId": 2000,
            "mesoPrice": 1000,
            "tokenPrice": 0,
            "unitPrice": 1.0,
            "slotMax": 100
          }
        },
        {
          "type": "commodities",
          "id": "550e8400-e29b-41d4-a716-446655440001",
          "attributes": {
            "templateId": 2001,
            "mesoPrice": 1500,
            "tokenPrice": 0,
            "unitPrice": 1.0,
            "slotMax": 100
          }
        }
      ]
    }
  }
  ```


#### Get All Shops

Retrieves all shops for the current tenant.

- **URL**: `/api/shops`
- **Method**: GET
- **Query Parameters**:
  - `include` - Optional. Specify "commodities" to include the commodities associated with each shop in the response.
- **Response**: JSON array containing shop information and optionally commodities

Example Response (with include=commodities):
```json
{
  "data": [
    {
      "type": "shops",
      "id": "shop-9000001",
      "attributes": {
        "npcId": 9000001,
        "recharger": true
      },
      "relationships": {
        "commodities": {
          "data": [
            {
              "type": "commodities",
              "id": "550e8400-e29b-41d4-a716-446655440000"
            }
          ]
        }
      }
    },
    {
      "type": "shops",
      "id": "shop-9000002",
      "attributes": {
        "npcId": 9000002,
        "recharger": false
      },
      "relationships": {
        "commodities": {
          "data": [
            {
              "type": "commodities",
              "id": "550e8400-e29b-41d4-a716-446655440001"
            }
          ]
        }
      }
    }
  ],
  "included": [
    {
      "type": "commodities",
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "attributes": {
        "templateId": 2000,
        "mesoPrice": 1000,
        "tokenPrice": 0,
        "unitPrice": 1.0,
        "slotMax": 100
      }
    },
    {
      "type": "commodities",
      "id": "550e8400-e29b-41d4-a716-446655440001",
      "attributes": {
        "templateId": 2001,
        "mesoPrice": 1500,
        "tokenPrice": 0,
        "unitPrice": 1.0,
        "slotMax": 100
      }
    }
  ]
}
```

#### Delete All Shops

Deletes all shops for the current tenant.

- **URL**: `/api/shops`
- **Method**: DELETE
- **Response**: No content (204)

#### Delete All Commodities for an NPC

Deletes all commodities associated with a specific NPC's shop.

- **URL**: `/api/npcs/{npcId}/shop/relationships/commodities`
- **Method**: DELETE
- **URL Parameters**:
  - `npcId` - The ID of the NPC
- **Response**: No content (204)

#### Seed Shops

Seeds the database with shop data from JSON files included in the container. This operation deletes all existing shops and commodities for the tenant and creates new ones from the seed data.

- **URL**: `/api/shops/seed`
- **Method**: POST
- **Request Body**: None
- **Response**: JSON object containing seed operation results
  ```json
  {
    "deletedShops": 50,
    "deletedCommodities": 1500,
    "createdShops": 99,
    "createdCommodities": 3194,
    "failedCount": 0
  }
  ```

Example Request:
```bash
curl -X POST http://localhost:8080/api/shops/seed \
  -H "TENANT_ID: 083839c6-c47c-42a6-9585-76492795d123" \
  -H "REGION: GMS" \
  -H "MAJOR_VERSION: 83" \
  -H "MINOR_VERSION: 1"
```

## Shop Seed Data

The service includes default shop data in JSON format, located in the `/shops` directory within the container. Each JSON file represents one NPC shop.

### JSON File Format

**Example shop file (`11000.json`):**
```json
{
  "npcId": 11000,
  "recharger": false,
  "commodities": [
    {
      "templateId": 1332005,
      "mesoPrice": 500,
      "discountRate": 0,
      "tokenTemplateId": 0,
      "tokenPrice": 0,
      "period": 0,
      "levelLimit": 0
    }
  ]
}
```

### Field Descriptions

**Shop Fields:**
- `npcId` (required): The NPC identifier
- `recharger` (optional, default: false): Whether the shop supports recharging throwable items
- `commodities` (required): Array of items sold in the shop

**Commodity Fields:**
- `templateId` (required): Item template ID
- `mesoPrice` (required): Price in mesos
- `discountRate` (optional, default: 0): Discount percentage (0-100)
- `tokenTemplateId` (optional, default: 0): Alternative currency item ID
- `tokenPrice` (optional, default: 0): Price in alternative currency
- `period` (optional, default: 0): Time limit on purchase in minutes (0 = unlimited)
- `levelLimit` (optional, default: 0): Minimum level required to purchase (0 = no limit)

### JSON Schema

A JSON schema is provided at `/shops/schema.json` for validating shop definition files.

### Modifying Seed Data

To add or modify default shop data:

1. Edit or create JSON files in the `shops/` directory of the source
2. Rebuild the Docker image
3. Deploy the new image
4. Call `POST /api/shops/seed` to load the updated data for each tenant
