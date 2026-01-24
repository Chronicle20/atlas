# Equipable Domain

## Responsibility

The equipable domain manages equipment instances. Each equipable represents a specific piece of equipment with individualized statistics that may differ from the base template.

## Core Models

### Model

Represents an equipment instance with the following properties:

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Unique identifier |
| itemId | uint32 | Reference to equipment template |
| strength | uint16 | Strength stat bonus |
| dexterity | uint16 | Dexterity stat bonus |
| intelligence | uint16 | Intelligence stat bonus |
| luck | uint16 | Luck stat bonus |
| hp | uint16 | HP stat bonus |
| mp | uint16 | MP stat bonus |
| weaponAttack | uint16 | Weapon attack bonus |
| magicAttack | uint16 | Magic attack bonus |
| weaponDefense | uint16 | Weapon defense bonus |
| magicDefense | uint16 | Magic defense bonus |
| accuracy | uint16 | Accuracy bonus |
| avoidability | uint16 | Avoidability bonus |
| hands | uint16 | Hands stat |
| speed | uint16 | Speed bonus |
| jump | uint16 | Jump bonus |
| slots | uint16 | Available upgrade slots |
| ownerName | string | Owner name |
| locked | bool | Lock status |
| spikes | bool | Spike property |
| karmaUsed | bool | Karma usage status |
| cold | bool | Cold property |
| canBeTraded | bool | Trade restriction |
| levelType | byte | Level type |
| level | byte | Equipment level |
| experience | uint32 | Equipment experience |
| hammersApplied | uint32 | Number of hammers applied |
| expiration | time.Time | Expiration timestamp |

### ModelBuilder

Builder pattern implementation for constructing Model instances. Supports both Set methods (absolute values) and Add methods (delta values with bounds checking).

## Invariants

- Stat values cannot go below 0 when applying negative deltas
- Stat values cannot exceed type maximums when applying positive deltas
- All equipables are tenant-scoped

## Processors

### Processor

Handles equipable operations:

- **GetById**: Retrieves an equipable by ID
- **Create**: Creates an equipable from provided stats or fetches template stats from atlas-data service if stats are all zero
- **CreateRandom**: Creates an equipable with randomized stats based on template values (variance of 10% capped at configurable maximums)
- **Update**: Updates an equipable with changed fields only
- **DeleteById**: Deletes an equipable by ID

All create, update, and delete operations emit corresponding Kafka events within transactions.
