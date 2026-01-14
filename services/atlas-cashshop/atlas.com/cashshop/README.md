# atlas-cashshop

Cash shop microservice for the Atlas platform. Manages account wallets, character wishlists, cash inventory with compartments, and purchasable cash items.

## Overview

The cash shop service provides:

- **Wallet Management**: Account-level currency tracking (credit, points, prepaid)
- **Wishlist Management**: Character-specific wishlist for cash shop items
- **Cash Inventory**: Account-scoped inventory with typed compartments (Explorer/Cygnus/Legend)
- **Cash Items**: Purchasable items with unique cash IDs and expiration tracking
- **Purchase Processing**: Handles cash shop purchase requests and inventory management

## Architecture

### Domain Model

```
Wallet (account-scoped)
├── id (UUID)
├── accountId (uint32)
├── credit (uint32)
├── points (uint32)
└── prepaid (uint32)

Wishlist (character-scoped)
├── id (UUID)
├── characterId (uint32)
└── serialNumber (uint32)

Inventory (account-scoped, virtual aggregate)
├── accountId (uint32)
├── explorer (Compartment)
├── cygnus (Compartment)
└── legend (Compartment)

Compartment
├── id (UUID)
├── accountId (uint32)
├── type (byte: 1=Explorer, 2=Cygnus, 3=Legend)
├── capacity (uint32, default: 55)
└── assets []Asset

Asset
├── id (UUID)
├── compartmentId (UUID)
├── slot (int16)
├── templateId (uint32)
├── itemId (uint32)
└── expiration (time.Time)

Item (cash item)
├── id (uint32, auto-increment)
├── cashId (int64, unique)
├── templateId (uint32)
├── quantity (uint32)
├── flag (uint16)
├── purchasedBy (uint32)
└── expiration (time.Time)
```

### Compartment Types

| Type | Value | Description |
|------|-------|-------------|
| Explorer | 1 | Standard character compartment |
| Cygnus | 2 | Cygnus Knight character compartment |
| Legend | 3 | Legend character compartment |

### Currency Types

| Type | Value | Description |
|------|-------|-------------|
| Credit | 1 | NX Credit (direct purchase) |
| Points | 2 | NX Prepaid converted to points |
| Prepaid | 3 | NX Prepaid |

## API

### REST Endpoints

#### Wallet

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/accounts/{accountId}/wallet` | Get account wallet |
| POST | `/api/accounts/{accountId}/wallet` | Create wallet (explicit) |
| PATCH | `/api/accounts/{accountId}/wallet` | Update wallet currency |

#### Wishlist

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/characters/{characterId}/cash-shop/wishlist` | Get character wishlist |
| POST | `/api/characters/{characterId}/cash-shop/wishlist` | Add item to wishlist |
| DELETE | `/api/characters/{characterId}/cash-shop/wishlist` | Clear entire wishlist |
| DELETE | `/api/characters/{characterId}/cash-shop/wishlist/{itemId}` | Remove item from wishlist |

#### Inventory

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/accounts/{accountId}/cash-shop/inventory` | Get full inventory with all compartments |
| POST | `/api/accounts/{accountId}/cash-shop/inventory` | Create inventory (explicit) |

#### Compartments

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/accounts/{accountId}/cash-shop/inventory/compartments?type={type}` | Get compartments (optional type filter) |

#### Assets

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/accounts/{accountId}/cash-shop/inventory/compartments/{compartmentId}/assets/{assetId}` | Get specific asset |

#### Cash Items

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/cash-shop/items/{itemId}` | Get cash item by ID |
| POST | `/api/cash-shop/items` | Create cash item |

### Kafka Topics

#### Commands (Consumed)

| Topic | Environment Variable | Commands |
|-------|---------------------|----------|
| Cash Shop | `COMMAND_TOPIC_CASH_SHOP` | `REQUEST_PURCHASE`, `REQUEST_INVENTORY_INCREASE_BY_TYPE`, `REQUEST_INVENTORY_INCREASE_BY_ITEM`, `REQUEST_STORAGE_INCREASE`, `REQUEST_STORAGE_INCREASE_BY_ITEM`, `REQUEST_CHARACTER_SLOT_INCREASE_BY_ITEM` |
| Cash Item | `COMMAND_TOPIC_CASH_ITEM` | `CREATE` |
| Cash Compartment | `COMMAND_TOPIC_CASH_COMPARTMENT` | `ACCEPT`, `RELEASE` |
| Wallet | `COMMAND_TOPIC_WALLET` | `ADJUST_CURRENCY` |

#### Events (Consumed)

| Topic | Environment Variable | Events |
|-------|---------------------|--------|
| Account Status | `EVENT_TOPIC_ACCOUNT_STATUS` | `CREATED`, `DELETED` |

#### Events (Produced)

| Topic | Environment Variable | Events |
|-------|---------------------|--------|
| Wallet Status | `EVENT_TOPIC_WALLET_STATUS` | `CREATED`, `UPDATED`, `DELETED` |
| Wishlist Status | `EVENT_TOPIC_WISHLIST_STATUS` | `ADDED`, `DELETED`, `DELETED_ALL` |
| Cash Shop Status | `EVENT_TOPIC_CASH_SHOP_STATUS` | `PURCHASE`, `INVENTORY_CAPACITY_INCREASED`, `ERROR` |
| Cash Item Status | `STATUS_TOPIC_CASH_ITEM` | `CREATED` |
| Cash Inventory Status | `EVENT_TOPIC_CASH_INVENTORY_STATUS` | `CREATED`, `UPDATED`, `DELETED` |
| Cash Compartment Status | `EVENT_TOPIC_CASH_COMPARTMENT_STATUS` | `CREATED`, `UPDATED`, `DELETED`, `ACCEPTED`, `RELEASED`, `ERROR` |

## Event Flows

### Account Creation

When an account is created (`EVENT_TOPIC_ACCOUNT_STATUS` → `CREATED`):
1. Create wallet with zero balances
2. Create inventory with default compartments (Explorer, Cygnus, Legend) each with capacity 55

### Account Deletion

When an account is deleted (`EVENT_TOPIC_ACCOUNT_STATUS` → `DELETED`):
1. Delete wallet
2. Delete inventory (cascades to compartments and assets)

### Purchase Flow

1. Receive `REQUEST_PURCHASE` command with character ID, currency type, and serial number
2. Validate sufficient currency balance in wallet
3. Create cash item with unique cash ID
4. Deduct currency from wallet
5. Create asset in appropriate compartment
6. Emit `PURCHASE` status event

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DATABASE_HOST` | PostgreSQL host | - |
| `DATABASE_PORT` | PostgreSQL port | 5432 |
| `DATABASE_NAME` | Database name | - |
| `DATABASE_USER` | Database user | - |
| `DATABASE_PASS` | Database password | - |
| `REST_PORT` | HTTP server port | - |
| `BOOTSTRAP_SERVERS` | Kafka bootstrap servers | - |
| `COMMAND_TOPIC_CASH_SHOP` | Cash shop command topic | - |
| `COMMAND_TOPIC_CASH_ITEM` | Cash item command topic | - |
| `COMMAND_TOPIC_CASH_COMPARTMENT` | Compartment command topic | - |
| `COMMAND_TOPIC_WALLET` | Wallet command topic | - |
| `EVENT_TOPIC_ACCOUNT_STATUS` | Account status event topic | - |
| `EVENT_TOPIC_WALLET_STATUS` | Wallet status event topic | - |
| `EVENT_TOPIC_WISHLIST_STATUS` | Wishlist status event topic | - |
| `EVENT_TOPIC_CASH_SHOP_STATUS` | Cash shop status event topic | - |
| `STATUS_TOPIC_CASH_ITEM` | Cash item status topic | - |
| `EVENT_TOPIC_CASH_INVENTORY_STATUS` | Inventory status event topic | - |
| `EVENT_TOPIC_CASH_COMPARTMENT_STATUS` | Compartment status event topic | - |
| `CHARACTERS_URL` | atlas-character service URL | - |
| `COMMODITIES_URL` | atlas-commodity service URL | - |

## Development

### Running Tests

```bash
go test ./...
```

### Building

```bash
go build -o atlas-cashshop
```

### Database Migrations

Migrations are handled automatically via GORM AutoMigrate on startup for:
- Wallet entities
- Wishlist entities
- Item entities
- Compartment entities
- Asset entities

## Dependencies

- **atlas-account**: Account lifecycle events (creates wallet/inventory on account creation)
- **atlas-character**: Character data lookups for purchase validation
- **atlas-commodity**: Commodity/item catalog lookups for pricing and metadata
