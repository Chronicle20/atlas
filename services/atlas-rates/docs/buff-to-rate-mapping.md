# Buff-to-Rate Lookup Model

This document describes how buff stat types from the game client are mapped to rate multipliers in the atlas-rates service.

## Overview

Buffs in MapleStory use game client stat types (e.g., `HOLY_SYMBOL`, `MESO_UP`) rather than rate-specific types. The atlas-rates service maintains a lookup model that:

1. Identifies which buff stat types affect rates
2. Maps stat types to the appropriate rate type
3. Converts stat amounts to rate multipliers using the correct formula

## Stat Type to Rate Type Mappings

| Buff Stat Type | Rate Type | Conversion Method | Description |
|----------------|-----------|-------------------|-------------|
| `HOLY_SYMBOL` | `exp` | Additive | EXP bonus from Holy Symbol skill |
| `MESO_UP` | `meso` | Direct | Meso rate bonus from Meso Up skill |

## Conversion Methods

Different buffs use different formulas to calculate their rate multiplier:

### Additive Conversion

Formula: `multiplier = 1.0 + (amount / 100.0)`

Used when the buff provides a **bonus percentage** on top of the base rate.

**Example: Holy Symbol**
- Skill provides 50% bonus EXP
- Event payload: `{ "type": "HOLY_SYMBOL", "amount": 50 }`
- Calculation: `1.0 + (50 / 100.0) = 1.50`
- Result: 1.50x EXP multiplier

### Direct Conversion

Formula: `multiplier = amount / 100.0`

Used when the buff amount represents the **total percentage** (including base).

**Example: Meso Up**
- Skill provides 103% meso rate
- Event payload: `{ "type": "MESO_UP", "amount": 103 }`
- Calculation: `103 / 100.0 = 1.03`
- Result: 1.03x meso multiplier

## Event Structure

Buff events are received on the `EVENT_TOPIC_CHARACTER_BUFF_STATUS` Kafka topic.

### Applied Event

```json
{
  "worldId": 0,
  "channelId": 1,
  "characterId": 12345,
  "type": "APPLIED",
  "body": {
    "fromId": 12345,
    "sourceId": 2311003,
    "duration": 120000,
    "changes": [
      { "type": "HOLY_SYMBOL", "amount": 50 }
    ],
    "createdAt": "2026-01-29T10:00:00Z",
    "expiresAt": "2026-01-29T10:02:00Z"
  }
}
```

### Expired Event

```json
{
  "worldId": 0,
  "channelId": 1,
  "characterId": 12345,
  "type": "EXPIRED",
  "body": {
    "sourceId": 2311003,
    "duration": 120000,
    "changes": [
      { "type": "HOLY_SYMBOL", "amount": 50 }
    ],
    "createdAt": "2026-01-29T10:00:00Z",
    "expiresAt": "2026-01-29T10:02:00Z"
  }
}
```

## Processing Flow

1. **Receive buff event** on Kafka topic
2. **Filter by event type** (APPLIED or EXPIRED)
3. **For each stat change** in the event:
   - Check if stat type is in the mapping (`IsRateStatType`)
   - Get the rate mapping (`GetRateMapping`)
   - Calculate multiplier using the conversion method (`CalculateMultiplier`)
4. **Update rate registry**:
   - APPLIED: Add factor with source `buff:{sourceId}`
   - EXPIRED: Remove factor with source `buff:{sourceId}`

## Adding New Buff Types

To add support for a new rate-affecting buff:

1. Add the stat type constant in `kafka/message/buff/kafka.go`:
   ```go
   const (
       StatTypeHolySymbol = "HOLY_SYMBOL"
       StatTypeMesoUp     = "MESO_UP"
       StatTypeNewBuff    = "NEW_BUFF_STAT_TYPE"  // Add new constant
   )
   ```

2. Add the mapping in `buffToRateMappings`:
   ```go
   var buffToRateMappings = map[string]RateMapping{
       StatTypeHolySymbol: {RateType: "exp", Conversion: ConversionAdditive},
       StatTypeMesoUp:     {RateType: "meso", Conversion: ConversionDirect},
       StatTypeNewBuff:    {RateType: "item_drop", Conversion: ConversionAdditive},
   }
   ```

3. Determine the correct conversion method:
   - **Additive**: Buff gives a bonus (e.g., +50% means amount=50)
   - **Direct**: Buff gives a total rate (e.g., 150% means amount=150)

## Known Rate-Affecting Skills

| Skill Name | Skill ID | Stat Type | Rate Type | Typical Amount |
|------------|----------|-----------|-----------|----------------|
| Holy Symbol | 2311003 | `HOLY_SYMBOL` | `exp` | 50 (50% bonus) |
| Advanced Blessing | 2321005 | `HOLY_SYMBOL` | `exp` | 50 (50% bonus) |
| Meso Up | 4111001 | `MESO_UP` | `meso` | 103 (103% rate) |
| Meso Guard | 4211005 | `MESO_UP` | `meso` | Varies |

## Implementation Files

- `kafka/message/buff/kafka.go` - Mapping definitions and conversion functions
- `kafka/consumer/buff/consumer.go` - Event handler implementation
- `character/initializer.go` - Buff initialization on character load

## Related Documentation

- [atlas-rates README](../README.md) - Service overview
- [Rate System Tasks](../../../dev/active/rate-system/rate-system-tasks.md) - Implementation tracking
