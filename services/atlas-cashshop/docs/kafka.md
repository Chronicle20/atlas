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
| OPEN_SURPRISE | OpenSurpriseCommandBody | Open a Cash Shop Surprise box (task-207); see Surprise domain doc |
| REQUEST_COUPON_REDEMPTION | RequestCouponRedemptionCommandBody | Redeem a coupon code for the requesting character's account (task-206) |
| REQUEST_LOCKER_REBATE | RequestLockerRebateCommandBody | Refund one locker asset's purchase price and remove it (task-240 task 11) |
| REQUEST_GIFT_PURCHASE | RequestGiftPurchaseCommandBody | Purchase a commodity as a gift, charging the sender and delivering into the recipient's locker (task-240 task 13) |
| REQUEST_PACKAGE_PURCHASE | RequestPackagePurchaseCommandBody | Purchase a cash package (buy-for-self or gift, discriminated by `recipientCharacterId`) (task-240 task 16) |
| REQUEST_RING_PURCHASE | RequestRingPurchaseCommandBody | Purchase a couple/friendship ring pair for the buyer and a partner (task-240 task 19) |
| REQUEST_EQUIP_SLOT_INCREASE | RequestEquipSlotIncreaseCommandBody | Purchase an equip-slot (pendant2) extension (task-240 task 23) |
| ACKNOWLEDGE_GIFTS | AcknowledgeGiftsCommandBody | Mark a set of locker assets as "gift list presented" (task-240 Defect H) |
| MARK_GIFT_NOTE_SENT | MarkGiftNoteSentCommandBody | Mark a locker asset's gift-forward note as sent (task-240 Defect I) |
| EXTEND_EQUIP_SLOT | ExtendEquipSlotCommandBody | Internal follow-up command, minted via the outbox by REQUEST_EQUIP_SLOT_INCREASE's handler, that performs the deferred atlas-character equip-slot write (task-240 task 24c); never sent by atlas-channel |

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
| ERROR | ErrorEventBody | Operation failed; `error` is one of `NOT_ENOUGH_CASH`, `INVENTORY_FULL`, `UNKNOWN_ERROR`, or an operation-specific reason string (e.g. `CANNOT_GIFT_RECIPIENT_INVENTORY_FULL`, `PARTNER_INVENTORY_FULL`); `operation` names the cash shop arm the failure belongs to (one of the `ErrorOperation*` constants), empty for the legacy capacity-increase arm |
| SURPRISE_OPENED | SurpriseOpenedEventBody | Cash Shop Surprise box opened; reward asset granted (task-207) |
| SURPRISE_FAILED | SurpriseFailedEventBody | Cash Shop Surprise open rejected; `reason` is a log/operator-only field, never surfaced to the client (task-207) |
| COUPON_REDEEMED | CouponRedeemedBody | Coupon redeemed; currency and/or cash item rewards granted (task-206) |
| COUPON_FAILED | CouponFailedBody | Coupon redemption rejected; `error` is one of the coupon `ErrorKey*` values (task-206) |
| LOCKER_REBATED | LockerRebatedBody | Locker asset rebated; purchase price refunded to the originating currency bucket (task-240 task 11) |
| GIFT_PURCHASED | GiftPurchasedBody | Gift purchase completed; commodity delivered into the recipient's locker (task-240 task 13) |
| PACKAGE_PURCHASED | PackagePurchasedBody | Cash package purchased; one asset created per package member (task-240 task 16) |
| RING_PURCHASED | RingPurchasedBody | Ring pair purchased; reports the buyer's own half (task-240 task 19) |
| EQUIP_SLOT_INCREASED | EquipSlotIncreasedBody | Equip-slot extension completed on atlas-character (task-240 task 23/24c) |

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

### COMMAND_TOPIC_CASH_SHOP (produced)
COMMAND_TOPIC_CASH_SHOP also carries an internal loopback command, minted via the outbox rather than sent by atlas-channel.

| Command Type | Body Type | Description |
|--------------|-----------|-------------|
| EXTEND_EQUIP_SLOT | ExtendEquipSlotCommandBody | Deferred atlas-character equip-slot write, queued by REQUEST_EQUIP_SLOT_INCREASE once the wallet debit and purchase record have committed (task-240 task 24c) |

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
  "transactionId": "uuid",
  "currency": 1,
  "serialNumber": 67890,
  "operation": "BUY_NORMAL"
}
```
`transactionId` is an opaque correlation id (zero UUID means no correlation, for backward compatibility). `operation` names the cash shop arm requesting the purchase, echoed back on `PurchaseEventBody`/`ErrorEventBody`; empty means the generic BUY arm.

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

#### OpenSurpriseCommandBody
```json
{
  "transactionId": "uuid",
  "accountId": 12345,
  "cashId": 67890
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

#### RequestCouponRedemptionCommandBody
```json
{
  "code": "ABCD1234EF"
}
```
`code` is the coupon code, already normalized (trimmed + uppercased) by the caller. The owning account is resolved server-side from the command's `characterId`.

#### RequestLockerRebateCommandBody
```json
{
  "transactionId": "uuid",
  "accountId": 12345,
  "cashId": 12345
}
```
`cashId` is the asset's `cash_assets.cash_id` (the client's `GW_ItemSlotBase::liCashItemSN`), not the row id.

#### RequestGiftPurchaseCommandBody
```json
{
  "transactionId": "uuid",
  "serialNumber": 67890,
  "recipientCharacterId": 54321,
  "senderName": "Sender",
  "message": "Happy birthday!"
}
```

#### RequestPackagePurchaseCommandBody
```json
{
  "transactionId": "uuid",
  "currency": 1,
  "serialNumber": 67890,
  "recipientCharacterId": 0,
  "senderName": "Sender"
}
```
`recipientCharacterId` of 0 means buy-for-self; non-zero means gift.

#### RequestRingPurchaseCommandBody
```json
{
  "transactionId": "uuid",
  "currency": 1,
  "serialNumber": 67890,
  "partnerCharacterId": 54321,
  "senderName": "Sender",
  "message": "Forever",
  "ringType": "COUPLE"
}
```
`ringType` is `COUPLE` or `FRIENDSHIP`.

#### RequestEquipSlotIncreaseCommandBody
```json
{
  "transactionId": "uuid",
  "currency": 1,
  "serialNumber": 67890
}
```

#### AcknowledgeGiftsCommandBody
```json
{
  "accountId": 12345,
  "cashIds": [12345, 67890]
}
```

#### MarkGiftNoteSentCommandBody
```json
{
  "accountId": 12345,
  "cashId": 12345
}
```

#### ExtendEquipSlotCommandBody
```json
{
  "transactionId": "uuid",
  "slotIndex": 3,
  "days": 30
}
```
`slotIndex` is the Atlas canonical equipped-inventory position (the pendant2 slot).

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
  "assetId": 42,
  "transactionId": "uuid",
  "operation": "BUY_NORMAL"
}
```
`operation` echoes `RequestPurchaseCommandBody.operation`; empty means the generic BUY arm.

#### SurpriseOpenedEventBody
```json
{
  "compartmentId": "uuid",
  "boxCashId": 67890,
  "boxRemaining": 2,
  "rewardAssetId": 42,
  "rewardTemplateId": 5000,
  "rewardCount": 1
}
```

#### SurpriseFailedEventBody
```json
{
  "reason": "LOCKER_FULL"
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
  "cashItemId": 12345,
  "transactionId": "uuid",
  "operation": "GIFT"
}
```
`operation` is one of the `ErrorOperation*` constants (`GIFT`, `BUY_NORMAL`, `REBATE`, `COUPLE`, `FRIENDSHIP`, `BUY_PACKAGE`, `GIFT_PACKAGE`, `ENABLE_EQUIP_SLOT`) naming the arm this failure belongs to; empty means the legacy capacity-increase arm.

#### CouponRedeemedBody
```json
{
  "compartmentId": "uuid",
  "assetIds": [42],
  "maplePoints": 0,
  "credit": 0
}
```
`maplePoints`/`credit` are the deltas this coupon awarded, not balances. `compartmentId` is the zero UUID when `assetIds` is empty (a currency-only coupon).

#### CouponFailedBody
```json
{
  "error": "COUPON_EXPIRED"
}
```
`error` is one of `INVALID_COUPON_CODE`, `COUPON_NOT_REGISTERED`, `COUPON_EXPIRED`, `COUPON_ALREADY_USED`, `COUPON_USAGE_LIMIT`, `INVENTORY_FULL`, `UNKNOWN_ERROR`.

#### LockerRebatedBody
```json
{
  "transactionId": "uuid",
  "cashId": 12345,
  "amount": 100,
  "currency": 1
}
```
`currency` is the wallet bucket credited (1 = credit/NX, 2 = Maple Points, anything else = prepaid).

#### GiftPurchasedBody
```json
{
  "transactionId": "uuid",
  "recipientName": "Recipient",
  "templateId": 5000,
  "quantity": 1,
  "price": 100,
  "recipientCharacterId": 54321
}
```

#### PackagePurchasedBody
```json
{
  "transactionId": "uuid",
  "compartmentId": "uuid",
  "assetIds": [42, 43],
  "packageTemplateId": 5000,
  "price": 100,
  "recipientCharacterId": 0,
  "recipientName": "Buyer"
}
```
On a buy-for-self purchase, `recipientCharacterId`/`recipientName` echo the buyer's own identity.

#### RingPurchasedBody
```json
{
  "transactionId": "uuid",
  "compartmentId": "uuid",
  "assetId": 42,
  "partnerName": "Partner",
  "templateId": 5000,
  "quantity": 1,
  "ringType": "COUPLE",
  "pairId": "uuid"
}
```
Reports the buyer's own half; `pairId` correlates it with the partner's own view.

#### EquipSlotIncreasedBody
```json
{
  "transactionId": "uuid",
  "slotIndex": 3,
  "days": 30
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
- REQUEST_LOCKER_REBATE, REQUEST_GIFT_PURCHASE, REQUEST_PACKAGE_PURCHASE, REQUEST_RING_PURCHASE, and REQUEST_EQUIP_SLOT_INCREASE each claim their `transactionId` via the shared idempotency ledger as the first statement of their transaction; a Kafka redelivery of an already-claimed transaction id is treated as success-without-effect (no error, no event)
- REQUEST_COUPON_REDEMPTION redemption is a single local database transaction (claimed via a unique index on `(tenant_id, coupon_id, account_id)` rather than the shared idempotency ledger); coupon redemption rate limiting is tracked in Redis and fails open on a Redis outage
- REQUEST_EQUIP_SLOT_INCREASE defers its atlas-character write behind an internal EXTEND_EQUIP_SLOT command minted via the outbox, so the cross-service HTTP call only happens once the wallet debit and purchase record have durably committed
