# Kafka Integration

## Topics Consumed

### COMMAND_TOPIC_CONSUMABLE

Consumable command topic.

| Command | Description |
|---------|-------------|
| REQUEST_ITEM_CONSUME | Consume item from inventory |
| REQUEST_ITEM_REWARD | Use a reward box item |
| REQUEST_CATCH_MONSTER | Use a bridle (monster catch item) |
| REQUEST_SCROLL | Use scroll on equipment |
| REQUEST_SKILL_BOOK_USE | Use a skill book or mastery book |
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
| CREATED | Compartment creation confirmed |
| CREATION_FAILED | Asset creation failed (e.g. reward-box grant could not be placed) |

### EVENT_TOPIC_ASSET_STATUS

Asset status events emitted by atlas-inventory. Consumed via a dynamically registered one-time handler to await the CREATED confirmation that marks a reward-box grant as successful. Subscribed from the latest offset.

| Event | Description |
|-------|-------------|
| CREATED | Asset created (reward-box grant success signal) |
| QUANTITY_CHANGED | Asset quantity changed |

### EVENT_TOPIC_MONSTER_CATCH

Dedicated, low-volume monster catch-outcome topic owned by atlas-monsters. Consumed via a dynamically registered one-time handler correlated by `(characterId, itemId)`, one per in-flight catch attempt.

| Event | Description |
|-------|-------------|
| CATCH_RESOLVED | Bridle (catch-item) capture attempt resolved (success or failure) |

### EVENT_TOPIC_SAGA_STATUS

Saga status events. Consumed via a dynamically registered one-time handler per skill-book use transaction (no persistent handlers).

| Event | Description |
|-------|-------------|
| COMPLETED | Saga completed |
| FAILED | Saga failed |

## Topics Produced

### EVENT_TOPIC_CONSUMABLE_STATUS

Consumable status events.

| Event | Description |
|-------|-------------|
| ERROR | Consumption error occurred |
| SCROLL | Scroll usage result |
| VEGA_SCROLL | Vega's Spell scroll usage result |
| EFFECT_APPLIED | Consumable effect applied |
| REWARD_EFFECT | Reward-box grant presentation effect |
| REWARD_WON | Reward-box grant world-message announcement |
| VICIOUS_HAMMER | Vicious hammer usage result |
| SKILL_BOOK_RESULT | Skill/mastery book use result |
| CATCH_FAILED | Pre-reserve bridle (catch-item) rejection (use-delay, inventory full, or invalid item) |

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
| CREATE_ASSET | Create an asset (reward-box grant) |

### COMMAND_TOPIC_PET

Pet commands.

| Command | Description |
|---------|-------------|
| AWARD_FULLNESS | Increase pet fullness |
| SET_SKILL | Grant or remove a pet skill (pet skill pouch) |

### COMMAND_TOPIC_MONSTER

Monster commands. Shared topic owned by atlas-monsters; this service produces only `CATCH`.

| Command | Description |
|---------|-------------|
| CATCH | Resolve a bridle (catch-item) capture attempt against a monster |

### COMMAND_TOPIC_SAGA

Saga commands.

| Command | Description |
|---------|-------------|
| (saga envelope) | Submits a saga (e.g. skill_book_use) |

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

### REQUEST_ITEM_REWARD Command

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "characterId": 0,
  "type": "REQUEST_ITEM_REWARD",
  "body": {
    "source": 0,
    "itemId": 0
  }
}
```

### REQUEST_CATCH_MONSTER Command

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "characterId": 0,
  "type": "REQUEST_CATCH_MONSTER",
  "body": {
    "source": 0,
    "itemId": 0,
    "monsterUniqueId": 0
  }
}
```

### REQUEST_SKILL_BOOK_USE Command

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "characterId": 0,
  "type": "REQUEST_SKILL_BOOK_USE",
  "body": {
    "slot": 0,
    "itemId": 0
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

`error` is one of `PET_CANNOT_CONSUME`, `PET_CANNOT_LEARN`, `CONSUME_FAILED`, `INVENTORY_FULL`, or `VEGA_INVALID`, or empty for unclassified pre-reserve errors.

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

### REWARD_EFFECT Event

```json
{
  "characterId": 0,
  "type": "REWARD_EFFECT",
  "body": {
    "boxItemId": 0,
    "effect": ""
  }
}
```

Emitted after a reward-box grant is confirmed, only when the won reward entry declares an effect path.

### REWARD_WON Event

```json
{
  "characterId": 0,
  "type": "REWARD_WON",
  "body": {
    "boxItemId": 0,
    "itemId": 0,
    "message": ""
  }
}
```

Emitted after a reward-box grant is confirmed, only when the won reward entry declares a world message. `message` has the `/name` and `/item` tokens substituted with the character name and reward item name.

### SKILL_BOOK_RESULT Event

```json
{
  "characterId": 0,
  "type": "SKILL_BOOK_RESULT",
  "body": {
    "isMasteryBook": false,
    "skillId": 0,
    "masterLevel": 0,
    "canUse": true,
    "success": false
  }
}
```

Emitted once per REQUEST_SKILL_BOOK_USE request. `canUse: false` reports a pre-saga validation rejection or saga failure. `canUse: true` reports the saga outcome, with `success` carrying the roll result.

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

### CREATE_ASSET Compartment Command

```json
{
  "transactionId": "uuid",
  "characterId": 0,
  "inventoryType": 0,
  "type": "CREATE_ASSET",
  "body": {
    "templateId": 0,
    "quantity": 0,
    "expiration": "2006-01-02T15:04:05Z",
    "ownerId": 0,
    "flag": 0,
    "rechargeable": 0,
    "useAverageStats": false
  }
}
```

Produced to grant a reward-box win. Success is confirmed by a CREATED event on `EVENT_TOPIC_ASSET_STATUS`; failure is reported by a CREATION_FAILED event on `EVENT_TOPIC_COMPARTMENT_STATUS`.

### CREATION_FAILED Compartment Status Event (consumed)

```json
{
  "transactionId": "uuid",
  "characterId": 0,
  "compartmentId": "uuid",
  "type": "CREATION_FAILED",
  "body": {
    "errorCode": "",
    "message": ""
  }
}
```

`errorCode` is one of `CREATE_ASSET_TEMPLATE_NOT_FOUND`, `CREATE_ASSET_INVENTORY_FULL`, or `CREATE_ASSET_UNKNOWN_ERROR`.

### CREATED Asset Status Event (consumed)

```json
{
  "transactionId": "uuid",
  "characterId": 0,
  "compartmentId": "uuid",
  "assetId": 0,
  "templateId": 0,
  "slot": 0,
  "type": "CREATED"
}
```

Consumed from `EVENT_TOPIC_ASSET_STATUS` as the reward-box grant success confirmation. The body is opaque; correlation is by the envelope's `transactionId`.

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

### SET_SKILL Command

```json
{
  "actorId": 0,
  "petId": 0,
  "type": "SET_SKILL",
  "body": {
    "skill": "",
    "enabled": true
  }
}
```

Produced when a pet skill pouch (cash classification 519) is consumed; one command per skill key the item's data carries.

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

### CATCH Command (produced)

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "monsterId": 0,
  "type": "CATCH",
  "body": {
    "characterId": 0,
    "itemId": 0
  }
}
```

Produced on `COMMAND_TOPIC_MONSTER` (atlas-monsters' shared command envelope) once the catch item's reservation is confirmed by `RESERVED`. `monsterId` carries the mob's unique (field object) id.

### CATCH_RESOLVED Event (consumed)

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "uniqueId": 0,
  "monsterId": 0,
  "type": "CATCH_RESOLVED",
  "body": {
    "characterId": 0,
    "itemId": 0,
    "success": true,
    "cause": ""
  }
}
```

Consumed from `EVENT_TOPIC_MONSTER_CATCH` via a one-time handler correlated by `(characterId, itemId)`. `success: true` commits the reservation and grants the reward item; `success: false` cancels the reservation and leaves the catch item untouched. `cause` is populated on failure (`SPECIES_MISMATCH`, `HP_TOO_HIGH`, `ROLL_FAILED`, or the internal-only `UNRESOLVED`).

### CATCH_FAILED Event (produced)

```json
{
  "characterId": 0,
  "type": "CATCH_FAILED",
  "body": {
    "itemId": 0,
    "cause": "USE_DELAY"
  }
}
```

Emitted on `EVENT_TOPIC_CONSUMABLE_STATUS` when a catch request is rejected before the item is reserved. `cause` is one of `USE_DELAY`, `INVENTORY_FULL`, or `INVALID_ITEM`.

### Saga Command (produced)

```json
{
  "transactionId": "uuid",
  "sagaType": "skill_book_use",
  "initiatedBy": "SKILL_BOOK",
  "steps": []
}
```

Produced on `COMMAND_TOPIC_SAGA` to submit a saga (shared `atlas-saga` envelope). Used by the skill-book use flow to sequence book destruction and skill grant/update.

### Saga Status Event (consumed)

```json
{
  "transactionId": "uuid",
  "type": "COMPLETED",
  "body": {}
}
```

Consumed from `EVENT_TOPIC_SAGA_STATUS` via a one-time handler keyed to the transactionId. `type` is `COMPLETED` or `FAILED`; the `FAILED` body carries `errorCode`, `reason`, and `failedStep`.

## Transaction Semantics

Item consumption uses saga-style transactions:

1. Request item reservation via REQUEST_RESERVE command
2. Register one-time handler for RESERVED event (validated by transactionId and itemId)
3. On RESERVED: Execute item logic, then CONSUME or CANCEL_RESERVATION
4. On error: CANCEL_RESERVATION and emit ERROR event

The one-time handler is registered dynamically on the compartment status event topic. It validates that the incoming RESERVED event matches the expected transactionId and itemId before invoking the item consumer callback.

REQUEST_VEGA_SCROLL uses a chained two-reservation variant of the same transactionId: a one-time handler is registered for both the vega (CASH) item and the scroll (USE) item before the first REQUEST_RESERVE (for the vega item) is sent. The vega item's RESERVED confirmation triggers the second REQUEST_RESERVE (for the scroll); the scroll's RESERVED confirmation runs the terminal consumer. REQUEST_FEED and REQUEST_VICIOUS_HAMMER follow the single-reservation pattern above.

REQUEST_ITEM_REWARD reserves the box, then on RESERVED (`ConsumeReward`) rolls a reward and issues CREATE_ASSET, registering one-time handlers on both `EVENT_TOPIC_ASSET_STATUS` (CREATED success) and `EVENT_TOPIC_COMPARTMENT_STATUS` (CREATION_FAILED failure) before the command is sent; exactly one fires and it deregisters its sibling. Success commits the box via CONSUME and emits REWARD_EFFECT/REWARD_WON; failure cancels the box reservation and preserves the box.

REQUEST_CATCH_MONSTER reserves the catch item, then on RESERVED (`ConsumeCatch`) sends a CATCH command to atlas-monsters on `COMMAND_TOPIC_MONSTER` and awaits CATCH_RESOLVED on `EVENT_TOPIC_MONSTER_CATCH`, correlated by `(characterId, itemId)`. Both the CATCH_RESOLVED handler and the RESERVED handler are registered before the reserve request is sent. Success commits the reservation and grants the reward item via CREATE_ASSET; failure cancels the reservation and leaves the catch item untouched.

REQUEST_SKILL_BOOK_USE validates eligibility synchronously (no reservation), then builds and submits a `skill_book_use` saga (COMMAND_TOPIC_SAGA) with a one-time handler registered on `EVENT_TOPIC_SAGA_STATUS` before submission. The saga's terminal COMPLETED or FAILED status drives a single SKILL_BOOK_RESULT event.
