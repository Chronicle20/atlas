# Atlas Rates Service

The atlas-rates service provides centralized rate management for character multipliers affecting experience gain, meso drops, item drops, and quest experience.

## Overview

This service aggregates rate factors from multiple sources (world settings, buffs, items) and computes final rate multipliers for each character. Other services query this service when they need to apply rate multipliers to gameplay calculations.

## Rate Types

| Type | Description |
|------|-------------|
| `exp` | Experience points multiplier |
| `meso` | Meso drop amount multiplier |
| `item_drop` | Item drop chance multiplier |
| `quest_exp` | Quest reward experience multiplier |

## Rate Factors

Rates are computed by multiplying factors from different sources:

- **World Rate**: Base rate for the entire world (configured in tenant settings, managed by atlas-world)
- **Buff Rate**: From active buffs (e.g., Holy Symbol for exp, Meso Up for meso)
- **Item Rate**: From equipped bonusExp items and cash coupons

Final rate formula:
```
FinalRate = WorldRate × BuffRate × ItemRate
```

All factors default to 1.0 if not set.

## REST API

### Get Character Rates

```
GET /api/rates/{worldId}/{channelId}/{characterId}
```

Response:
```json
{
  "type": "rates",
  "id": "12345",
  "attributes": {
    "expRate": 1.5,
    "mesoRate": 1.0,
    "itemDropRate": 2.0,
    "questExpRate": 1.0,
    "factors": [
      {"source": "world", "rateType": "exp", "multiplier": 1.0},
      {"source": "buff:2311003", "rateType": "exp", "multiplier": 1.5},
      {"source": "item:1002357", "rateType": "exp", "multiplier": 1.1},
      {"source": "world", "rateType": "item_drop", "multiplier": 2.0}
    ]
  }
}
```

## Kafka Integration

### Consumed Events

| Topic | Event Type | Description |
|-------|------------|-------------|
| `EVENT_TOPIC_CHARACTER_BUFF_STATUS` | `APPLIED` | Buff applied to character |
| `EVENT_TOPIC_CHARACTER_BUFF_STATUS` | `EXPIRED` | Buff expired on character |
| `EVENT_TOPIC_WORLD_RATE` | `RATE_CHANGED` | World rate updated by admin |
| `EVENT_TOPIC_ASSET_STATUS` | `MOVED` | Item equipped/unequipped |

### Buff-to-Rate Mapping

Buffs use game client stat types, not rate-specific types. The service maps these automatically:

| Buff Stat Type | Rate Type | Conversion |
|----------------|-----------|------------|
| `HOLY_SYMBOL` | `exp` | Additive: `1.0 + amount/100` (amount=50 → 1.5x) |
| `MESO_UP` | `meso` | Direct: `amount/100` (amount=103 → 1.03x) |

## Item Rate Integration

### BonusExp Equipment

Equipment items with `bonusExp` properties provide time-based EXP bonuses:

```json
{
  "bonusExp": [
    {"incExpR": 10, "termStart": 0},   // +10% immediately
    {"incExpR": 20, "termStart": 24},  // +20% after 24 hours
    {"incExpR": 30, "termStart": 72}   // +30% after 72 hours
  ]
}
```

The service tracks equipped items and calculates the current tier based on hours equipped.

### Cash Coupons

Cash coupons (521xxxx for EXP, 536xxxx for Drop) provide time-limited bonuses:

| Template ID Range | Rate Type |
|-------------------|-----------|
| 5210000-5219999 | `exp` |
| 5360000-5369999 | `item_drop` |

Coupons have a `rate` (multiplier) and `time` (duration in minutes). Expired coupons are automatically cleaned up.

## Lazy Initialization

Characters are initialized lazily on first rate query or map change. This queries:
1. **Equipped items** - via atlas-inventory → atlas-equipables
2. **Cash coupons** - via atlas-inventory → atlas-cashshop
3. **Active buffs** - via atlas-buffs

This ensures rate tracking survives service restarts while characters are online.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                       atlas-rates                            │
│  ┌───────────────────────┐  ┌───────────────────────┐      │
│  │   Character Registry   │  │    Item Tracker       │      │
│  │  map[tenant][charId]   │  │  map[tenant][charId]  │      │
│  │  → factors, rates      │  │  → tracked items      │      │
│  └───────────────────────┘  └───────────────────────┘      │
└─────────────────────────────────────────────────────────────┘
         ▲                              │
         │ Kafka Events                 │ REST Queries
         │                              ▼
    ┌────┴────┐                  ┌──────┴──────┐
    │atlas-   │                  │atlas-       │
    │world    │                  │monster-death│
    │atlas-   │                  │atlas-saga-  │
    │buffs    │                  │orchestrator │
    │atlas-   │                  │             │
    │inventory│                  │             │
    └─────────┘                  └─────────────┘
```

## Timestamp Ownership

Timestamps are owned by the authoritative services:
- **Equipment `createdAt`**: atlas-equipables
- **Cash item `createdAt`**: atlas-cashshop
- **Proxied via**: atlas-inventory

atlas-rates queries these timestamps during initialization and uses them to calculate time-based bonuses.

## Environment Variables

| Variable | Description |
|----------|-------------|
| `REST_PORT` | Port for REST API (default: 8080) |
| `BOOTSTRAP_SERVERS` | Kafka broker addresses |
| `JAEGER_HOST_PORT` | Jaeger tracing endpoint |
| `LOG_LEVEL` | Logging level (debug, info, warn, error) |
| `EVENT_TOPIC_CHARACTER_BUFF_STATUS` | Kafka topic for buff events |
| `EVENT_TOPIC_WORLD_RATE` | Kafka topic for world rate events |
| `EVENT_TOPIC_ASSET_STATUS` | Kafka topic for inventory events |
