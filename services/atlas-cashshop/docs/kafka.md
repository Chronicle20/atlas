# Kafka

## Topics Consumed

### EVENT_TOPIC_ACCOUNT_STATUS
Account status events from external account service.

| Event Type | Body Type | Description |
|------------|-----------|-------------|
| CREATED | StatusEvent | Account created - initializes wallet with zero balances and inventory with default compartments |
| DELETED | StatusEvent | Account deleted - removes wallet and inventory |

### EVENT_TOPIC_CHARACTER_STATUS
Character status events from external character service.

| Event Type | Body Type | Description |
|------------|-----------|-------------|
| DELETED | DeletedStatusEventBody | Character deleted - clears character's wishlist |

### COMMAND_TOPIC_CASH_SHOP
Cash shop commands.

| Command Type | Body Type | Description |
|--------------|-----------|-------------|
| REQUEST_PURCHASE | RequestPurchaseCommandBody | Request to purchase a commodity |
| REQUEST_INVENTORY_INCREASE_BY_TYPE | RequestInventoryIncreaseByTypeCommandBody | Request to increase inventory capacity by type |
| REQUEST_INVENTORY_INCREASE_BY_ITEM | RequestInventoryIncreaseByItemCommandBody | Request to increase inventory capacity using a commodity |
| REQUEST_STORAGE_INCREASE | RequestStorageIncreaseBody | Unconditionally produces an EVENT_TOPIC_CASH_SHOP_STATUS ERROR event with code `UNKNOWN_ERROR` |
| REQUEST_STORAGE_INCREASE_BY_ITEM | RequestCharacterSlotIncreaseByItemCommandBody | Unconditionally produces an EVENT_TOPIC_CASH_SHOP_STATUS ERROR event with code `UNKNOWN_ERROR` |
| REQUEST_CHARACTER_SLOT_INCREASE_BY_ITEM | RequestCharacterSlotIncreaseByItemCommandBody | Unconditionally produces an EVENT_TOPIC_CASH_SHOP_STATUS ERROR event with code `UNKNOWN_ERROR` |
| EXPIRE | ExpireCommandBody | Expire a cash shop asset, optionally creating a replacement |

### COMMAND_TOPIC_CASH_COMPARTMENT
Cash compartment commands.

| Command Type | Body Type | Description |
|--------------|-----------|-------------|
| ACCEPT | AcceptCommandBody | Accept an asset into a compartment (creates flattened asset with preserved cashId) |
| RELEASE | ReleaseCommandBody | Release an asset from a compartment |

### COMMAND_TOPIC_CASH_ITEM
Cash item commands.

| Command Type | Body Type | Description |
|--------------|-----------|-------------|
| CREATE | CreateCommandBody | Create a new cash asset |

### COMMAND_TOPIC_WALLET
Wallet commands.

| Command Type | Body Type | Description |
|--------------|-----------|-------------|
| ADJUST_CURRENCY | AdjustCurrencyCommand | Adjust currency balance |

---

## Topics Produced

### EVENT_TOPIC_WALLET_STATUS
Wallet status events.

| Event Type | Body Type | Description |
|------------|-----------|-------------|
| CREATED | StatusEventCreatedBody | Wallet created |
| UPDATED | StatusEventUpdatedBody | Wallet balances updated |
| DELETED | StatusEventDeletedBody | Wallet deleted |
| ERROR | StatusEventErrorBody | A transactional ADJUST_CURRENCY command failed; only emitted when the command carried a non-nil transaction ID |

### EVENT_TOPIC_WISHLIST_STATUS
Wishlist status events.

| Event Type | Body Type | Description |
|------------|-----------|-------------|
| ADDED | StatusEventAddedBody | Item added to wishlist |
| DELETED | StatusEventDeletedBody | Item removed from wishlist |
| DELETED_ALL | StatusEventDeletedAllBody | All items removed from wishlist |

### EVENT_TOPIC_CASH_SHOP_STATUS
Cash shop status events.

| Event Type | Body Type | Description |
|------------|-----------|-------------|
| INVENTORY_CAPACITY_INCREASED | InventoryCapacityIncreasedBody | Inventory capacity increased |
| PURCHASE | PurchaseEventBody | Commodity purchased, asset created |
| ERROR | ErrorEventBody | Operation failed; `error` is one of `NOT_ENOUGH_CASH`, `INVENTORY_FULL`, `UNKNOWN_ERROR` |

### EVENT_TOPIC_CASH_INVENTORY_STATUS
Cash inventory status events.

| Event Type | Body Type | Description |
|------------|-----------|-------------|
| CREATED | StatusEventCreatedBody | Inventory created (empty body) |
| DELETED | StatusEventDeletedBody | Inventory deleted (empty body) |

### EVENT_TOPIC_CASH_COMPARTMENT_STATUS
Cash compartment status events.

| Event Type | Body Type | Description |
|------------|-----------|-------------|
| CREATED | StatusEventCreatedBody | Compartment created |
| UPDATED | StatusEventUpdatedBody | Compartment updated |
| DELETED | StatusEventDeletedBody | Compartment deleted |
| ACCEPTED | StatusEventAcceptedBody | Asset accepted into compartment |
| RELEASED | StatusEventReleasedBody | Asset released from compartment |
| ERROR | StatusEventErrorBody | Operation failed; `errorCode` is one of `UNKNOWN_ERROR`, `ASSET_CREATION_FAILED`, `ITEM_NOT_FOUND` |

### STATUS_TOPIC_CASH_ITEM
Cash item status events (produced by asset processor).

| Event Type | Body Type | Description |
|------------|-----------|-------------|
| CREATED | StatusEventCreatedBody | Asset created (includes cashId, templateId, quantity, purchasedBy, flag) |
| EXPIRED | StatusEventExpiredBody | Asset expired (includes isCash flag, optional replaceItemId and replaceMessage) |

### COMMAND_TOPIC_COMPARTMENT
Character inventory compartment commands (produced during inventory capacity increase purchases).

| Command Type | Body Type | Description |
|--------------|-----------|-------------|
| INCREASE_CAPACITY | IncreaseCapacityCommandBody | Increase character inventory compartment capacity |

---

## Message Types

### Command Messages

#### Cash Shop Command
```json
{
  "characterId": 12345,
  "type": "COMMAND_TYPE",
  "body": {}
}
```

#### RequestPurchaseCommandBody
```json
{
  "currency": 1,
  "serialNumber": 67890
}
```

#### RequestInventoryIncreaseByTypeCommandBody
```json
{
  "currency": 1,
  "inventoryType": 1
}
```

#### RequestInventoryIncreaseByItemCommandBody
```json
{
  "currency": 1,
  "serialNumber": 67890
}
```

#### RequestStorageIncreaseBody
```json
{
  "currency": 1
}
```

#### RequestStorageIncreaseByItemCommandBody
```json
{
  "currency": 1,
  "serialNumber": 67890
}
```

#### RequestCharacterSlotIncreaseByItemCommandBody
```json
{
  "currency": 1,
  "serialNumber": 67890
}
```

#### ExpireCommandBody
```json
{
  "accountId": 12345,
  "worldId": 0,
  "assetId": 42,
  "templateId": 5000,
  "inventoryType": -1,
  "slot": 0,
  "replaceItemId": 5001,
  "replaceMessage": "Your item has expired."
}
```

#### Compartment Command
```json
{
  "accountId": 12345,
  "characterId": 67890,
  "compartmentType": 1,
  "type": "COMMAND_TYPE",
  "body": {}
}
```

#### AcceptCommandBody
```json
{
  "transactionId": "uuid",
  "compartmentId": "uuid",
  "cashId": 12345,
  "templateId": 5000,
  "quantity": 1,
  "commodityId": 100,
  "purchasedBy": 67890,
  "flag": 0
}
```

#### ReleaseCommandBody
```json
{
  "transactionId": "uuid",
  "compartmentId": "uuid",
  "assetId": 42,
  "cashId": 12345,
  "templateId": 5000
}
```

#### Item Command
```json
{
  "characterId": 12345,
  "type": "COMMAND_TYPE",
  "body": {}
}
```

#### CreateCommandBody (Item)
```json
{
  "templateId": 5000,
  "commodityId": 100,
  "quantity": 1,
  "purchasedBy": 12345
}
```

#### AdjustCurrencyCommand
```json
{
  "transactionId": "uuid",
  "accountId": 12345,
  "currencyType": 1,
  "amount": -100,
  "type": "ADJUST_CURRENCY"
}
```

#### Character Compartment Command
```json
{
  "characterId": 12345,
  "inventoryType": 1,
  "type": "COMMAND_TYPE",
  "body": {}
}
```

#### IncreaseCapacityCommandBody
```json
{
  "amount": 8
}
```

### Status Event Messages

#### Wallet StatusEvent
```json
{
  "accountId": 12345,
  "type": "EVENT_TYPE",
  "body": {}
}
```

#### StatusEventCreatedBody (Wallet)
```json
{
  "credit": 1000,
  "points": 500,
  "prepaid": 200
}
```

#### StatusEventUpdatedBody (Wallet)
```json
{
  "credit": 900,
  "points": 500,
  "prepaid": 200,
  "transactionId": "uuid"
}
```

#### StatusEventErrorBody (Wallet)
```json
{
  "transactionId": "uuid",
  "reason": "insufficient credit balance"
}
```

#### Wishlist StatusEvent
```json
{
  "characterId": 12345,
  "type": "EVENT_TYPE",
  "body": {}
}
```

#### StatusEventAddedBody (Wishlist)
```json
{
  "serialNumber": 67890,
  "itemId": "uuid"
}
```

#### Cash Shop StatusEvent
```json
{
  "characterId": 12345,
  "type": "EVENT_TYPE",
  "body": {}
}
```

#### PurchaseEventBody
```json
{
  "templateId": 5000,
  "price": 100,
  "compartmentId": "uuid",
  "assetId": 42
}
```

#### InventoryCapacityIncreasedBody
```json
{
  "inventoryType": 1,
  "capacity": 32,
  "amount": 8
}
```

#### ErrorEventBody
```json
{
  "error": "ERROR_CODE",
  "cashItemId": 12345
}
```

#### Inventory StatusEvent
```json
{
  "accountId": 12345,
  "type": "EVENT_TYPE",
  "body": {}
}
```

#### Compartment StatusEvent
```json
{
  "accountId": 12345,
  "characterId": 67890,
  "compartmentId": "uuid",
  "compartmentType": 1,
  "type": "EVENT_TYPE",
  "body": {}
}
```

#### StatusEventCreatedBody (Compartment)
```json
{
  "capacity": 55
}
```

#### StatusEventUpdatedBody (Compartment)
```json
{
  "capacity": 60
}
```

#### StatusEventAcceptedBody (Compartment)
```json
{
  "transactionId": "uuid",
  "assetId": 42
}
```

#### StatusEventReleasedBody (Compartment)
```json
{
  "transactionId": "uuid",
  "assetId": 42,
  "cashId": 12345,
  "templateId": 5000
}
```

#### StatusEventErrorBody (Compartment)
```json
{
  "errorCode": "ASSET_CREATION_FAILED",
  "transactionId": "uuid"
}
```

#### Item StatusEvent (produced by asset processor)
```json
{
  "characterId": 12345,
  "type": "EVENT_TYPE",
  "body": {}
}
```

#### StatusEventCreatedBody (Item)
```json
{
  "cashId": 12345,
  "templateId": 5000,
  "quantity": 1,
  "purchasedBy": 67890,
  "flag": 0
}
```

#### StatusEventExpiredBody (Item)
```json
{
  "isCash": true,
  "replaceItemId": 5001,
  "replaceMessage": "Your item has expired."
}
```

---

## Transaction Semantics

- Commands include optional `transactionId` for saga coordination
- Status events include `transactionId` when the originating command included one
- Wallet adjustments are atomic and validated for sufficient balance
- Purchase and other write operations execute within a database transaction; state-asserting events are buffered into a `message.Buffer` and routed through a transactional outbox (`atlas-outbox`) that is committed atomically with the database write, then drained to Kafka asynchronously
- Failure-path events that reflect no committed state change (for example, wallet ADJUST_CURRENCY failure, cash shop INVENTORY_FULL/UNKNOWN_ERROR rejections) are emitted on the direct Kafka producer path instead of the outbox, so they publish regardless of any rollback
- Compartment Accept uses find-or-create by cashId for idempotent asset creation
- Compartment Release validates asset existence before deletion
