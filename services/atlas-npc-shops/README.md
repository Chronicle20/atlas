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
