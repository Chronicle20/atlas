# Kafka

## Topics Consumed

| Topic Environment Variable      | Description                          |
|---------------------------------|--------------------------------------|
| COMMAND_TOPIC_NPC_SHOP          | Shop commands (enter, exit, buy, sell, recharge) |
| EVENT_TOPIC_CHARACTER_STATUS    | Character status events (logout, map change, channel change) |

## Topics Produced

| Topic Environment Variable      | Description                          |
|---------------------------------|--------------------------------------|
| EVENT_TOPIC_NPC_SHOP_STATUS     | Shop status events (entered, exited, error) |
| COMMAND_TOPIC_CHARACTER         | Character commands (change meso)     |
| COMMAND_TOPIC_COMPARTMENT       | Compartment commands (create asset, destroy, recharge) |

## Message Types

### Commands Consumed

#### Shop Commands (COMMAND_TOPIC_NPC_SHOP)

| Type     | Body Struct               | Description                        |
|----------|---------------------------|------------------------------------|
| ENTER    | CommandShopEnterBody      | Character enters a shop            |
| EXIT     | CommandShopExitBody       | Character exits a shop             |
| BUY      | CommandShopBuyBody        | Character purchases an item        |
| SELL     | CommandShopSellBody       | Character sells an item            |
| RECHARGE | CommandShopRechargeBody   | Character recharges a throwable    |

**CommandShopEnterBody**

| Field         | Type   | Description           |
|---------------|--------|-----------------------|
| npcTemplateId | uint32 | NPC template ID       |

**CommandShopBuyBody**

| Field          | Type   | Description                |
|----------------|--------|----------------------------|
| slot           | uint16 | Shop slot position         |
| itemTemplateId | uint32 | Item template ID           |
| quantity       | uint32 | Quantity to purchase       |
| discountPrice  | uint32 | Discounted price           |

**CommandShopSellBody**

| Field          | Type   | Description                |
|----------------|--------|----------------------------|
| slot           | int16  | Inventory slot position    |
| itemTemplateId | uint32 | Item template ID           |
| quantity       | uint32 | Quantity to sell           |

**CommandShopRechargeBody**

| Field | Type   | Description              |
|-------|--------|--------------------------|
| slot  | uint16 | Inventory slot position  |

#### Character Status Events (EVENT_TOPIC_CHARACTER_STATUS)

| Type            | Body Struct                   | Description                     |
|-----------------|-------------------------------|---------------------------------|
| LOGOUT          | StatusEventLogoutBody         | Character logged out            |
| MAP_CHANGED     | StatusEventMapChangedBody     | Character changed maps          |
| CHANNEL_CHANGED | ChangeChannelEventLoginBody   | Character changed channels      |

### Events Produced

#### Shop Status Events (EVENT_TOPIC_NPC_SHOP_STATUS)

| Type    | Body Struct             | Description                     |
|---------|-------------------------|---------------------------------|
| ENTERED | StatusEventEnteredBody  | Character entered a shop        |
| EXITED  | StatusEventExitedBody   | Character exited a shop         |
| ERROR   | StatusEventErrorBody    | Error occurred during operation |

**StatusEventEnteredBody**

| Field         | Type   | Description           |
|---------------|--------|-----------------------|
| npcTemplateId | uint32 | NPC template ID       |

**StatusEventErrorBody**

| Field      | Type   | Description                      |
|------------|--------|----------------------------------|
| error      | string | Error code                       |
| levelLimit | uint32 | Level requirement (if applicable)|
| reason     | string | Error reason (if applicable)     |

**Error Codes**

| Code                      | Description                           |
|---------------------------|---------------------------------------|
| OK                        | No error                              |
| OUT_OF_STOCK              | Item out of stock                     |
| NOT_ENOUGH_MONEY          | Insufficient mesos                    |
| INVENTORY_FULL            | No free inventory slots               |
| OUT_OF_STOCK_2            | Item out of stock (variant)           |
| OUT_OF_STOCK_3            | Item out of stock (variant)           |
| NOT_ENOUGH_MONEY_2        | Insufficient mesos for recharge       |
| NEED_MORE_ITEMS           | Insufficient quantity to sell         |
| OVER_LEVEL_REQUIREMENT    | Character level too high              |
| UNDER_LEVEL_REQUIREMENT   | Character level too low               |
| TRADE_LIMIT               | Trade limit reached                   |
| GENERIC_ERROR             | Generic error                         |
| GENERIC_ERROR_WITH_REASON | Generic error with reason             |

### Commands Produced

#### Character Commands (COMMAND_TOPIC_CHARACTER)

| Type                | Body Struct             | Description                |
|---------------------|-------------------------|----------------------------|
| REQUEST_CHANGE_MESO | RequestChangeMesoBody   | Request meso change        |

**RequestChangeMesoBody**

| Field     | Type   | Description                    |
|-----------|--------|--------------------------------|
| actorId   | uint32 | Actor ID triggering change     |
| actorType | string | Actor type (e.g., "SHOP")      |
| amount    | int32  | Meso amount (negative=deduct)  |

#### Compartment Commands (COMMAND_TOPIC_COMPARTMENT)

| Type         | Body Struct              | Description                |
|--------------|--------------------------|----------------------------|
| CREATE_ASSET | CreateAssetCommandBody   | Create item in inventory   |
| DESTROY      | DestroyCommandBody       | Remove item from inventory |
| RECHARGE     | RechargeCommandBody      | Recharge item quantity     |

**CreateAssetCommandBody**

| Field        | Type      | Description                   |
|--------------|-----------|-------------------------------|
| templateId   | uint32    | Item template ID              |
| quantity     | uint32    | Item quantity                 |
| expiration   | time.Time | Expiration time               |
| ownerId      | uint32    | Owner ID                      |
| flag         | uint16    | Item flags                    |
| rechargeable | uint64    | Rechargeable data             |

**DestroyCommandBody**

| Field    | Type   | Description              |
|----------|--------|--------------------------|
| slot     | int16  | Inventory slot position  |
| quantity | uint32 | Quantity to destroy      |

**RechargeCommandBody**

| Field    | Type   | Description              |
|----------|--------|--------------------------|
| slot     | int16  | Inventory slot position  |
| quantity | uint32 | Quantity to add          |

## Transaction Semantics

- Shop commands are consumed with tenant header parsing
- Messages are keyed by character ID for ordering
- Shop entry/exit maintains in-memory registry state
- Buy and sell operations emit commands to other services
