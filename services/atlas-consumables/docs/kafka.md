# Kafka Integration

## Topics Consumed

### COMMAND_TOPIC_CONSUMABLE

Consumable command topic.

| Command | Description |
|---------|-------------|
| REQUEST_ITEM_CONSUME | Consume item from inventory |
| REQUEST_SCROLL | Use scroll on equipment |
| APPLY_CONSUMABLE_EFFECT | Apply item effects without consuming |
| CANCEL_CONSUMABLE_EFFECT | Cancel consumable buff effects |

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
| EFFECT_APPLIED | Consumable effect applied |

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
    "locked": false,
    "spikes": false,
    "karmaUsed": false,
    "cold": false,
    "canBeTraded": false,
    "levelType": 0,
    "level": 0,
    "experience": 0,
    "hammersApplied": 0,
    "expiration": "2006-01-02T15:04:05Z"
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
