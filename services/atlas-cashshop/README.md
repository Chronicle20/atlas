# atlas-cashshop
Mushroom game cashshop Service

## Overview

A RESTful resource which provides cashshop services.

## Environment

### General
- LOG_LEVEL - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace
- REST_PORT - Port for the REST server

### Database
- DB_USER - Database username
- DB_PASSWORD - Database password
- DB_HOST - Database host
- DB_PORT - Database port
- DB_NAME - Database name

### Tracing
- JAEGER_HOST_PORT - Jaeger [host]:[port]

### Kafka
- BOOTSTRAP_SERVERS - Kafka bootstrap servers
- EVENT_TOPIC_CHARACTER_STATUS - Topic for character status events
- EVENT_TOPIC_ACCOUNT_STATUS - Topic for account status events
- EVENT_TOPIC_WALLET_STATUS - Topic for wallet status events
- EVENT_TOPIC_WISHLIST_STATUS - Topic for wishlist status events
- COMMAND_TOPIC_CASH_SHOP - Topic for cash shop commands
- EVENT_TOPIC_CASH_SHOP_STATUS - Topic for cash shop status events
- COMMAND_TOPIC_INVENTORY - Topic for inventory commands

## Kafka Messaging

### Consumers

#### Account Consumer
Listens for account status events:
- CREATED: When an account is created
- DELETED: When an account is deleted

#### Character Consumer
Listens for character status events:
- CREATED: When a character is created
- DELETED: When a character is deleted

#### Cash Shop Consumer
Processes cash shop commands:
- REQUEST_PURCHASE: Request to purchase an item
- REQUEST_INVENTORY_INCREASE_BY_TYPE: Request to increase inventory capacity by type
- REQUEST_INVENTORY_INCREASE_BY_ITEM: Request to increase inventory capacity by item
- REQUEST_STORAGE_INCREASE: Request to increase storage capacity
- REQUEST_STORAGE_INCREASE_BY_ITEM: Request to increase storage capacity by item
- REQUEST_CHARACTER_SLOT_INCREASE_BY_ITEM: Request to increase character slot capacity by item

### Producers

#### Cash Shop Status Events
Emits cash shop status events:
- INVENTORY_CAPACITY_INCREASED: When inventory capacity is increased
- ERROR: When an error occurs

#### Wallet Status Events
Emits wallet status events:
- CREATED: When a wallet is created
- UPDATED: When a wallet is updated
- DELETED: When a wallet is deleted

#### Wishlist Status Events
Emits wishlist status events:
- ADDED: When an item is added to a wishlist
- DELETED: When an item is removed from a wishlist
- DELETED_ALL: When all items are removed from a wishlist

#### Inventory Commands
Sends inventory commands:
- INCREASE_CAPACITY: Command to increase inventory capacity

## API

### Header

All RESTful requests require the supplied header information to identify the server instance.

```
TENANT_ID:083839c6-c47c-42a6-9585-76492795d123
REGION:GMS
MAJOR_VERSION:83
MINOR_VERSION:1
```

### REST Resources

#### Wallet
- GET /accounts/{accountId}/wallet - Get wallet information for an account
- POST /accounts/{accountId}/wallet - Create a wallet for an account
- PATCH /accounts/{accountId}/wallet - Update a wallet for an account

Wallet Model:
```json
{
  "accountId": 12345,
  "credit": 1000,
  "points": 500,
  "prepaid": 200
}
```

#### Wishlist
- GET /characters/{characterId}/cash-shop/wishlist - Get wishlist items for a character
- POST /characters/{characterId}/cash-shop/wishlist - Add an item to a character's wishlist
- DELETE /characters/{characterId}/cash-shop/wishlist - Clear a character's wishlist
- DELETE /characters/{characterId}/cash-shop/wishlist/{itemId} - Remove a specific item from a character's wishlist

Wishlist Item Model:
```json
{
  "characterId": 12345,
  "serialNumber": 67890
}
```

#### Items
- GET /cash-shop/items - Get cash shop items (with an itemId query parameter)

Item Model:
```json
{
  "cashId": 12345678,
  "templateId": 5000,
  "quantity": 1,
  "owner": 12345,
  "flag": 0,
  "purchasedBy": 12345
}
```

#### Cash Inventory
- GET /accounts/{accountId}/cash-shop/inventory - Get cash inventory for an account
- POST /accounts/{accountId}/cash-shop/inventory - Create a cash inventory for an account

Cash Inventory Model (JSON:API format):
```json
{
  "data": {
    "type": "cash-inventories",
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "attributes": {
      "accountId": 12345
    },
    "relationships": {
      "compartments": {
        "data": [
          { "type": "compartments", "id": "550e8400-e29b-41d4-a716-446655440001" },
          { "type": "compartments", "id": "550e8400-e29b-41d4-a716-446655440002" },
          { "type": "compartments", "id": "550e8400-e29b-41d4-a716-446655440003" }
        ]
      }
    }
  },
  "included": [
    {
      "type": "compartments",
      "id": "550e8400-e29b-41d4-a716-446655440001",
      "attributes": {
        "accountId": 12345,
        "type": "explorer",
        "capacity": 55
      },
      "relationships": {
        "assets": {
          "data": []
        }
      }
    },
    {
      "type": "compartments",
      "id": "550e8400-e29b-41d4-a716-446655440002",
      "attributes": {
        "accountId": 12345,
        "type": "cygnus",
        "capacity": 55
      },
      "relationships": {
        "assets": {
          "data": []
        }
      }
    },
    {
      "type": "compartments",
      "id": "550e8400-e29b-41d4-a716-446655440003",
      "attributes": {
        "accountId": 12345,
        "type": "legend",
        "capacity": 55
      },
      "relationships": {
        "assets": {
          "data": []
        }
      }
    }
  ]
}
```

#### Cash Compartment
- GET /accounts/{accountId}/cash-shop/inventory/compartments - Get all cash compartments for an account
- GET /accounts/{accountId}/cash-shop/inventory/compartments?type={compartmentType} - Get a specific cash compartment by type

Cash Compartment Model (JSON:API format):
```json
{
  "data": {
    "type": "compartments",
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "attributes": {
      "accountId": 12345,
      "type": "explorer",
      "capacity": 55
    },
    "relationships": {
      "assets": {
        "data": [
          { "type": "assets", "id": "550e8400-e29b-41d4-a716-446655440001" }
        ]
      }
    }
  },
  "included": [
    {
      "type": "assets",
      "id": "550e8400-e29b-41d4-a716-446655440001",
      "attributes": {
        "compartmentId": "550e8400-e29b-41d4-a716-446655440000"
      },
      "relationships": {
        "item": {
          "data": { "type": "items", "id": "1001" }
        }
      }
    },
    {
      "type": "items",
      "id": "1001",
      "attributes": {
        "cashId": 12345678,
        "templateId": 5000,
        "quantity": 1,
        "flag": 0,
        "purchasedBy": 12345
      }
    }
  ]
}
```
