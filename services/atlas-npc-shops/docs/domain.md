# Domain

## Shop

### Responsibility

Represents an NPC shop that sells commodities to characters.

### Core Models

#### Shop Model

| Field       | Type               | Description                                    |
|-------------|--------------------|------------------------------------------------|
| npcId       | uint32             | NPC template identifier                        |
| commodities | []Commodity        | List of commodities sold by this shop          |
| recharger   | bool               | Whether the shop supports recharging throwables|

#### Commodity Model

| Field           | Type      | Description                                     |
|-----------------|-----------|-------------------------------------------------|
| id              | uuid.UUID | Unique commodity identifier                     |
| npcId           | uint32    | NPC template identifier                         |
| templateId      | uint32    | Item template identifier                        |
| mesoPrice       | uint32    | Price in mesos                                  |
| discountRate    | byte      | Discount percentage (0-100)                     |
| tokenTemplateId | uint32    | Alternative currency item identifier            |
| tokenPrice      | uint32    | Price in alternative currency                   |
| period          | uint32    | Time limit on purchase in minutes (0=unlimited) |
| levelLimit      | uint32    | Minimum level required to purchase (0=no limit) |
| unitPrice       | float64   | Unit price for rechargeable items               |
| slotMax         | uint32    | Maximum stack size for the item                 |

### Invariants

- Shop npcId must be non-zero
- Commodity id must be non-nil
- Commodity templateId must be non-zero

### Processors

#### Shop Processor

- Retrieves shops by NPC ID
- Retrieves all shops for a tenant
- Creates shops with commodities
- Updates shops with commodities
- Manages shop entry and exit for characters
- Processes buy, sell, and recharge operations
- Tracks characters currently in shops via an in-memory registry
- Decorates shops with rechargeable consumables when the shop is a recharger

#### Commodity Processor

- Retrieves commodities by NPC ID
- Retrieves all commodities for a tenant
- Creates, updates, and deletes commodities
- Decorates commodities with item data (unitPrice, slotMax) based on item type

#### Seed Processor

- Seeds shop data from JSON files into the database
- Deletes existing shops and commodities before seeding
