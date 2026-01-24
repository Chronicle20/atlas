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
| REQUEST_PURCHASE | RequestPurchaseCommandBody | Request to purchase an item |
| REQUEST_INVENTORY_INCREASE_BY_TYPE | RequestInventoryIncreaseByTypeCommandBody | Request to increase inventory capacity by type |
| REQUEST_INVENTORY_INCREASE_BY_ITEM | RequestInventoryIncreaseByItemCommandBody | Request to increase inventory capacity using an item |
| REQUEST_STORAGE_INCREASE | RequestStorageIncreaseBody | Request to increase storage capacity |
| REQUEST_STORAGE_INCREASE_BY_ITEM | RequestStorageIncreaseByItemCommandBody | Request to increase storage capacity using an item |
| REQUEST_CHARACTER_SLOT_INCREASE_BY_ITEM | RequestCharacterSlotIncreaseByItemCommandBody | Request to increase character slots using an item |

### COMMAND_TOPIC_CASH_COMPARTMENT
Cash compartment commands.

| Command Type | Body Type | Description |
|--------------|-----------|-------------|
| ACCEPT | AcceptCommandBody | Accept an asset into a compartment |
| RELEASE | ReleaseCommandBody | Release an asset from a compartment |

### COMMAND_TOPIC_CASH_ITEM
Cash item commands.

| Command Type | Body Type | Description |
|--------------|-----------|-------------|
| CREATE | CreateCommandBody | Create a new cash item |

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
| PURCHASE | PurchaseEventBody | Item purchased |
| ERROR | ErrorEventBody | Operation failed |

### EVENT_TOPIC_CASH_INVENTORY_STATUS
Cash inventory status events.

| Event Type | Body Type | Description |
|------------|-----------|-------------|
| CREATED | StatusEventCreatedBody | Inventory created |
| DELETED | StatusEventDeletedBody | Inventory deleted |

### EVENT_TOPIC_CASH_COMPARTMENT_STATUS
Cash compartment status events.

| Event Type | Body Type | Description |
|------------|-----------|-------------|
| CREATED | StatusEventCreatedBody | Compartment created |
| UPDATED | StatusEventUpdatedBody | Compartment updated |
| DELETED | StatusEventDeletedBody | Compartment deleted |
| ACCEPTED | StatusEventAcceptedBody | Asset accepted into compartment |
| RELEASED | StatusEventReleasedBody | Asset released from compartment |
| ERROR | StatusEventErrorBody | Operation failed |

### STATUS_TOPIC_CASH_ITEM
Cash item status events.

| Event Type | Body Type | Description |
|------------|-----------|-------------|
| CREATED | StatusEventCreatedBody | Item created |

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

#### Compartment Command
```json
{
  "accountId": 12345,
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
  "referenceId": 12345
}
```

#### ReleaseCommandBody
```json
{
  "transactionId": "uuid",
  "compartmentId": "uuid",
  "assetId": 12345
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
  "assetId": "uuid",
  "itemId": 12345
}
```

#### ErrorEventBody
```json
{
  "error": "ERROR_CODE",
  "cashItemId": 12345
}
```

#### Compartment StatusEvent
```json
{
  "compartmentId": "uuid",
  "compartmentType": 1,
  "type": "EVENT_TYPE",
  "body": {}
}
```

---

## Transaction Semantics

- Commands include optional `transactionId` for saga coordination
- Status events include `transactionId` when command included one
- Wallet adjustments are atomic and validated for sufficient balance
- Events are buffered and emitted after successful transaction commit
