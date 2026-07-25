# Kafka Integration

## Topics Consumed

### COMMAND_TOPIC_CONSUMABLE

Consumable command topic.

| Command | Description |
|---------|-------------|
| REQUEST_ITEM_CONSUME | Consume item from inventory |
| REQUEST_SCROLL | Use scroll on equipment |
| REQUEST_VEGA_SCROLL | Apply a scroll at a Vega's Spell boosted rate |
| REQUEST_VICIOUS_HAMMER | Use a vicious hammer on equipment |
| APPLY_CONSUMABLE_EFFECT | Apply item effects without consuming |
| CANCEL_CONSUMABLE_EFFECT | Cancel consumable buff effects |

### COMMAND_TOPIC_TAMING_MOB_FOOD

Taming-mob (mount) food command topic.

| Command | Description |
|---------|-------------|
| REQUEST_FEED | Feed a taming-mob with a revitalizer item |

### COMMAND_TOPIC_ITEM_CONSUMED_ON_PICKUP

Item-consumed-on-pickup command topic.

| Command | Description |
|---------|-------------|
| ITEM_CONSUMED_ON_PICKUP | Item was consumed at pickup time (e.g. monster card) |

### EVENT_TOPIC_CHARACTER_STATUS

Character status events for location tracking.

| Event | Description |
|-------|-------------|
| LOGIN | Character login |
| LOGOUT | Character logout |
| MAP_CHANGED | Character changed maps |
| CHANNEL_CHANGED | Character changed channels |

### EVENT_TOPIC_COMPARTMENT_STATUS

Compartment status events for transaction handling. Consumed via dynamically registered one-time handlers.

| Event | Description |
|-------|-------------|
| RESERVED | Item reservation confirmed |
| RESERVATION_CANCELLED | Item reservation cancelled |

## Topics Produced

### EVENT_TOPIC_CONSUMABLE_STATUS

Consumable status events.

| Event | Description |
|-------|-------------|
| ERROR | Consumption error occurred |
| SCROLL | Scroll usage result |
| VEGA_SCROLL | Vega's Spell scroll usage result |
| EFFECT_APPLIED | Consumable effect applied |
| VICIOUS_HAMMER | Vicious hammer usage result |

### EVENT_TOPIC_TAMING_MOB_FOOD

Taming-mob (mount) food event topic.

| Event | Description |
|-------|-------------|
| (flat, untyped) | Revitalizer consumed; carries the tiredness heal amount |

### COMMAND_TOPIC_MONSTER_BOOK

Monster book command topic.

| Command | Description |
|---------|-------------|
| CARD_PICKED_UP | Monster card was picked up |

### COMMAND_TOPIC_CHARACTER

Character commands.

| Command | Description |
|---------|-------------|
| CHANGE_HP | Modify character HP |
| CHANGE_MP | Modify character MP |
| CHANGE_MAP | Teleport character |

### COMMAND_TOPIC_CHARACTER_BUFF

Character buff commands.

| Command | Description |
|---------|-------------|
| APPLY | Apply temporary stat buff |
| CANCEL | Cancel buff |

### COMMAND_TOPIC_COMPARTMENT

Compartment commands.

| Command | Description |
|---------|-------------|
| REQUEST_RESERVE | Reserve items for transaction |
| CONSUME | Commit item consumption |
| DESTROY | Destroy item |
| CANCEL_RESERVATION | Cancel item reservation |
| MODIFY_EQUIPMENT | Update equipment stats |

### COMMAND_TOPIC_PET

Pet commands.

| Command | Description |
|---------|-------------|
| AWARD_FULLNESS | Increase pet fullness |

## Message Types

### REQUEST_ITEM_CONSUME Command

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "characterId": 0,
  "type": "REQUEST_ITEM_CONSUME",
  "body": {
    "source": 0,
    "itemId": 0,
    "quantity": 0
  }
}
```

### REQUEST_SCROLL Command

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "characterId": 0,
  "type": "REQUEST_SCROLL",
  "body": {
    "scrollSlot": 0,
    "equipSlot": 0,
    "whiteScroll": false,
    "legendarySpirit": false
  }
}
```

### REQUEST_VEGA_SCROLL Command

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "characterId": 0,
  "type": "REQUEST_VEGA_SCROLL",
  "body": {
    "vegaSlot": 0,
    "vegaItemId": 0,
    "scrollSlot": 0,
    "equipSlot": 0
  }
}
```

### REQUEST_VICIOUS_HAMMER Command

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "characterId": 0,
  "type": "REQUEST_VICIOUS_HAMMER",
  "body": {
    "hammerSlot": 0,
    "equipSlot": 0
  }
}
```

### APPLY_CONSUMABLE_EFFECT Command

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "characterId": 0,
  "type": "APPLY_CONSUMABLE_EFFECT",
  "body": {
    "itemId": 0
  }
}
```

### CANCEL_CONSUMABLE_EFFECT Command

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "characterId": 0,
  "type": "CANCEL_CONSUMABLE_EFFECT",
  "body": {
    "itemId": 0
  }
}
```

### ERROR Event

```json
{
  "characterId": 0,
  "type": "ERROR",
  "body": {
    "error": "PET_CANNOT_CONSUME"
  }
}
```

`error` is one of `PET_CANNOT_CONSUME` or `VEGA_INVALID`, or empty for unclassified errors.

### SCROLL Event

```json
{
  "characterId": 0,
  "type": "SCROLL",
  "body": {
    "success": true,
    "cursed": false,
    "legendarySpirit": false,
    "whiteScroll": false
  }
}
```

### VEGA_SCROLL Event

```json
{
  "characterId": 0,
  "type": "VEGA_SCROLL",
  "body": {
    "success": true,
    "cursed": false
  }
}
```

### VICIOUS_HAMMER Event

```json
{
  "characterId": 0,
  "type": "VICIOUS_HAMMER",
  "body": {
    "success": true,
    "reason": ""
  }
}
```

`reason` is one of `""` (success), `UNKNOWN`, `NOT_UPGRADABLE`, `CAP_REACHED`, or `HORNTAIL`.

### EFFECT_APPLIED Event

```json
{
  "characterId": 0,
  "type": "EFFECT_APPLIED",
  "body": {
    "itemId": 0,
    "transactionId": "uuid"
  }
}
```

### CHANGE_HP Command

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "characterId": 0,
  "type": "CHANGE_HP",
  "body": {
    "channelId": 0,
    "amount": 0
  }
}
```

### CHANGE_MP Command

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "characterId": 0,
  "type": "CHANGE_MP",
  "body": {
    "channelId": 0,
    "amount": 0
  }
}
```

### CHANGE_MAP Command

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "characterId": 0,
  "type": "CHANGE_MAP",
  "body": {
    "channelId": 0,
    "mapId": 0,
    "instance": "uuid",
    "portalId": 0
  }
}
```

### APPLY Buff Command

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "characterId": 0,
  "type": "APPLY",
  "body": {
    "fromId": 0,
    "sourceId": -2000000,
    "level": 0,
    "duration": 0,
    "changes": [
      {
        "type": "ACCURACY",
        "amount": 0
      }
    ]
  }
}
```

Note: sourceId uses negative item ID for consumable buffs.

### CANCEL Buff Command

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "characterId": 0,
  "type": "CANCEL",
  "body": {
    "sourceId": -2000000
  }
}
```

### MODIFY_EQUIPMENT Compartment Command

```json
{
  "transactionId": "uuid",
  "characterId": 0,
  "inventoryType": 1,
  "type": "MODIFY_EQUIPMENT",
  "body": {
    "assetId": 0,
    "strength": 0,
    "dexterity": 0,
    "intelligence": 0,
    "luck": 0,
    "hp": 0,
    "mp": 0,
    "weaponAttack": 0,
    "magicAttack": 0,
    "weaponDefense": 0,
    "magicDefense": 0,
    "accuracy": 0,
    "avoidability": 0,
    "hands": 0,
    "speed": 0,
    "jump": 0,
    "slots": 0,
    "flag": 0,
    "levelType": 0,
    "level": 0,
    "experience": 0,
    "hammersApplied": 0,
    "expiration": "2006-01-02T15:04:05Z"
  }
}
```

### REQUEST_FEED Command (consumed)

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "characterId": 0,
  "type": "REQUEST_FEED",
  "body": {
    "slot": 0,
    "itemId": 0
  }
}
```

### Taming-Mob Food Event (produced)

Flat, untyped struct (no envelope `type` field) on `EVENT_TOPIC_TAMING_MOB_FOOD`.

```json
{
  "worldId": 0,
  "characterId": 0,
  "itemId": 0,
  "tirednessHeal": 0
}
```

### ITEM_CONSUMED_ON_PICKUP Command (consumed)

Flat struct (no envelope wrapper) on `COMMAND_TOPIC_ITEM_CONSUMED_ON_PICKUP`.

```json
{
  "tenantId": "uuid",
  "characterId": 0,
  "itemId": 0,
  "transactionId": "uuid",
  "type": "ITEM_CONSUMED_ON_PICKUP"
}
```

### CARD_PICKED_UP Command (produced)

```json
{
  "tenantId": "uuid",
  "characterId": 0,
  "eventId": "uuid",
  "type": "CARD_PICKED_UP",
  "body": {
    "cardId": 0,
    "source": "drop_pickup"
  }
}
```

### AWARD_FULLNESS Command

```json
{
  "actorId": 0,
  "petId": 0,
  "type": "AWARD_FULLNESS",
  "body": {
    "amount": 0
  }
}
```

### REQUEST_RESERVE Command

```json
{
  "transactionId": "uuid",
  "characterId": 0,
  "inventoryType": 0,
  "type": "REQUEST_RESERVE",
  "body": {
    "transactionId": "uuid",
    "items": [
      {
        "source": 0,
        "itemId": 0,
        "quantity": 0
      }
    ]
  }
}
```

### CONSUME Command

```json
{
  "transactionId": "uuid",
  "characterId": 0,
  "inventoryType": 0,
  "type": "CONSUME",
  "body": {
    "transactionId": "uuid",
    "slot": 0
  }
}
```

### DESTROY Command

```json
{
  "transactionId": "uuid",
  "characterId": 0,
  "inventoryType": 0,
  "type": "DESTROY",
  "body": {
    "slot": 0
  }
}
```

### CANCEL_RESERVATION Command

```json
{
  "transactionId": "uuid",
  "characterId": 0,
  "inventoryType": 0,
  "type": "CANCEL_RESERVATION",
  "body": {
    "transactionId": "uuid",
    "slot": 0
  }
}
```

### LOGIN Event (consumed)

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "characterId": 0,
  "type": "LOGIN",
  "body": {
    "channelId": 0,
    "mapId": 0,
    "instance": "uuid"
  }
}
```

### LOGOUT Event (consumed)

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "characterId": 0,
  "type": "LOGOUT",
  "body": {
    "channelId": 0,
    "mapId": 0,
    "instance": "uuid"
  }
}
```

### MAP_CHANGED Event (consumed)

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "characterId": 0,
  "type": "MAP_CHANGED",
  "body": {
    "channelId": 0,
    "oldMapId": 0,
    "oldInstance": "uuid",
    "targetMapId": 0,
    "targetInstance": "uuid",
    "targetPortalId": 0
  }
}
```

### CHANNEL_CHANGED Event (consumed)

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "characterId": 0,
  "type": "CHANNEL_CHANGED",
  "body": {
    "channelId": 0,
    "oldChannelId": 0,
    "mapId": 0,
    "instance": "uuid"
  }
}
```

### RESERVED Event (consumed)

```json
{
  "characterId": 0,
  "compartmentId": "uuid",
  "type": "RESERVED",
  "body": {
    "transactionId": "uuid",
    "itemId": 0,
    "slot": 0,
    "quantity": 0
  }
}
```

## Transaction Semantics

Item consumption uses saga-style transactions:

1. Request item reservation via REQUEST_RESERVE command
2. Register one-time handler for RESERVED event (validated by transactionId and itemId)
3. On RESERVED: Execute item logic, then CONSUME or CANCEL_RESERVATION
4. On error: CANCEL_RESERVATION and emit ERROR event

The one-time handler is registered dynamically on the compartment status event topic. It validates that the incoming RESERVED event matches the expected transactionId and itemId before invoking the item consumer callback.

REQUEST_VEGA_SCROLL uses a chained two-reservation variant of the same transactionId: a one-time handler is registered for both the vega (CASH) item and the scroll (USE) item before the first REQUEST_RESERVE (for the vega item) is sent. The vega item's RESERVED confirmation triggers the second REQUEST_RESERVE (for the scroll); the scroll's RESERVED confirmation runs the terminal consumer. REQUEST_FEED and REQUEST_VICIOUS_HAMMER follow the single-reservation pattern above.
