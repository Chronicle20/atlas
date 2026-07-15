# Kafka

## Topics Consumed

| Topic Variable | Consumer Group Name | Direction |
|---|---|---|
| `COMMAND_TOPIC_MERCHANT` | merchant_command | Command |
| `EVENT_TOPIC_CHARACTER_STATUS` | character_status | Event |
| `EVENT_TOPIC_COMPARTMENT_STATUS` | compartment_status | Event |

All three consumers run under a single Kafka consumer group id (`"Merchant Service"`, `main.go:34`, `main.go:93-104`).

## Topics Produced

| Topic Variable | Direction |
|---|---|
| `EVENT_TOPIC_MERCHANT_STATUS` | Event |
| `EVENT_TOPIC_MERCHANT_LISTING` | Event |
| `COMMAND_TOPIC_COMPARTMENT` | Command |
| `COMMAND_TOPIC_CHARACTER` | Command |

## Message Types

### Consumed Commands (COMMAND_TOPIC_MERCHANT)

Envelope `Command[E]` (`kafka/message/merchant/kafka.go:34-40`):

```
Command[E] {
  worldId, channelId, characterId, type, body: E
}
```

| Type constant | Wire string | Body Struct | Description |
|---|---|---|---|
| `CommandPlaceShop` | `PLACE_SHOP` | CommandPlaceShopBody | Create a new shop (Draft) |
| `CommandOpenShop` | `OPEN_SHOP` | CommandOpenShopBody | Transition shop from Draft to Open |
| `CommandCloseShop` | `CLOSE_SHOP` | CommandCloseShopBody | Close a shop manually |
| `CommandEnterMaintenance` | `ENTER_MAINTENANCE` | CommandEnterMaintenanceBody | Enter maintenance mode |
| `CommandExitMaintenance` | `EXIT_MAINTENANCE` | CommandExitMaintenanceBody | Exit maintenance mode |
| `CommandAddListing` | `ADD_LISTING` | CommandAddListingBody (embeds asset.AssetData) | Add an item listing |
| `CommandRemoveListing` | `REMOVE_LISTING` | CommandRemoveListingBody | Remove a listing by index |
| `CommandUpdateListing` | `UPDATE_LISTING` | CommandUpdateListingBody | Update listing price/bundles |
| `CommandPurchaseBundle` | `PURCHASE_BUNDLE` | CommandPurchaseBundleBody | Purchase bundles from a listing |
| `CommandEnterShop` | `ENTER_SHOP` | CommandEnterShopBody (ShopId, VisitorName) | Enter a shop as visitor |
| `CommandExitShop` | `EXIT_SHOP` | CommandExitShopBody | Exit a shop as visitor |
| `CommandSendMessage` | `SEND_MESSAGE` | CommandSendMessageBody | Send a chat message in a shop |
| `CommandRetrieveFrederick` | `RETRIEVE_FREDERICK` | CommandRetrieveFrederickBody (empty) | Retrieve items/mesos from Frederick |
| `CommandRecordItemSearch` | `RECORD_ITEM_SEARCH` | CommandRecordItemSearchBody (ItemId) | Record an item-listing search; increments the per-(tenant, world, item) search counter |
| `CommandWithdrawMeso` | `WITHDRAW_MESO` | CommandWithdrawMesoBody | Withdraw a hired merchant's accumulated meso balance |
| `CommandOrganizeListings` | `ORGANIZE_LISTINGS` | CommandOrganizeListingsBody | Re-order/compact listing display order |
| `CommandAddBlacklist` | `ADD_BLACKLIST` | CommandBlacklistBody (Name, BannedCharacterId) | Add a name to the shop blacklist |
| `CommandRemoveBlacklist` | `REMOVE_BLACKLIST` | CommandBlacklistBody | Remove a name from the shop blacklist |

Constants: `kafka/message/merchant/kafka.go:14-31`; handlers registered in `kafka/consumer/merchant/consumer.go:34-51`. `RECORD_ITEM_SEARCH` is handled synchronously by `searchcount.Processor` (no topic of its own); `ADD_BLACKLIST` / `REMOVE_BLACKLIST` are handled by the shop processor, which emits `BLACKLIST_UPDATED` (and may emit `VISITOR_EJECTED`).

### Consumed Events (EVENT_TOPIC_CHARACTER_STATUS)

Envelope `StatusEvent[E]` (`kafka/message/character/kafka.go:22-26`):

```
StatusEvent[E] {
  characterId, type, body: E
}
```

| Type constant | Wire string | Body Struct | Description |
|---|---|---|---|
| `EventCharacterStatusTypeLogout` | `LOGOUT` | StatusEventLogoutBody (empty) | Character disconnected; closes or exits-maintenance the character's active shops |

`LOGIN` and `MAP_CHANGED` type constants are declared but not handled (`kafka/message/character/kafka.go:9-12`, handler `kafka/consumer/character/consumer.go:36`).

### Consumed Events (EVENT_TOPIC_COMPARTMENT_STATUS)

Envelope `StatusEvent[E]` (`kafka/message/compartment/kafka.go:44-50`):

```
StatusEvent[E] {
  transactionId, characterId, compartmentId, type, body: E
}
```

| Type constant | Wire string | Body Struct | Description |
|---|---|---|---|
| `StatusEventTypeAccepted` | `ACCEPTED` | AcceptedEventBody | Item accept confirmation (logged) |
| `StatusEventTypeReleased` | `RELEASED` | ReleasedEventBody | Item release confirmation (logged) |
| `StatusEventTypeError` | `ERROR` | ErrorEventBody (ErrorCode, TransactionId) | Compartment operation error (logged) |

Handlers `kafka/consumer/compartment/consumer.go:27-29`. All three currently log only; no compensating action is taken.

### Produced Events (EVENT_TOPIC_MERCHANT_STATUS)

Envelope `StatusEvent[E]` (`kafka/message/merchant/kafka.go:175-179`):

```
StatusEvent[E] {
  characterId, type, body: E
}
```

| Type constant | Wire string | Body Struct | Description |
|---|---|---|---|
| `StatusEventShopSetup` | `SHOP_SETUP` | StatusEventShopOpenedBody | Shop created (Draft) and registered |
| `StatusEventShopOpened` | `SHOP_OPENED` | StatusEventShopOpenedBody | Shop transitioned to Open |
| `StatusEventShopClosed` | `SHOP_CLOSED` | StatusEventShopClosedBody (CloseReason byte) | Shop closed |
| `StatusEventShopCreateFailed` | `SHOP_CREATE_FAILED` | StatusEventShopCreateFailedBody (Reason) | Placement/creation rejected |
| `StatusEventShopUpdated` | `SHOP_UPDATED` | StatusEventShopUpdatedBody | Shop or listings changed |
| `StatusEventMaintenanceEntered` | `MAINTENANCE_ENTERED` | StatusEventVisitorBody | Shop entered maintenance |
| `StatusEventMaintenanceExited` | `MAINTENANCE_EXITED` | StatusEventVisitorBody | Shop exited maintenance |
| `StatusEventVisitorEntered` | `VISITOR_ENTERED` | StatusEventVisitorBody | Visitor entered shop |
| `StatusEventVisitorExited` | `VISITOR_EXITED` | StatusEventVisitorBody | Visitor exited shop |
| `StatusEventVisitorEjected` | `VISITOR_EJECTED` | StatusEventVisitorBody (LeaveReason) | Visitor ejected from shop |
| `StatusEventEnterFailed` | `ENTER_FAILED` | StatusEventEnterFailedBody (Reason) | Visitor entry rejected |
| `StatusEventCapacityFull` | `CAPACITY_FULL` | StatusEventCapacityFullBody | Shop at max visitor capacity |
| `StatusEventPurchaseFailed` | `PURCHASE_FAILED` | StatusEventPurchaseFailedBody (Reason) | Purchase attempt failed |
| `StatusEventMessageSent` | `MESSAGE_SENT` | StatusEventMessageSentBody (CharacterId, Slot, Content) | Chat message sent in shop |
| `StatusEventBlacklistUpdated` | `BLACKLIST_UPDATED` | StatusEventBlacklistUpdatedBody | Shop blacklist changed |
| `StatusEventFrederickNotification` | `FREDERICK_NOTIFICATION` | StatusEventFrederickNotificationBody (DaysSinceStorage) | Frederick retrieval reminder |

Constants: `kafka/message/merchant/kafka.go:141-156`. Reason/leave codes are carried as string fields inside the bodies (not separate topics):

- `StatusEventVisitorBody.LeaveReason` (`kafka.go:212-216`): `SHOP_CLOSED`, `OUT_OF_STOCK`, `USER_BANNED`.
- `StatusEventEnterFailedBody.Reason` (`kafka.go:160-164`): `UNDERGOING_MAINTENANCE`, `ROOM_CLOSED`, `BLACKLISTED`.
- `StatusEventShopCreateFailedBody.Reason` (`kafka.go:168-172`): `TOO_CLOSE_TO_PORTAL`, `TOO_CLOSE_TO_SHOP`, `NOT_FREE_MARKET`, `UNABLE`.
- `StatusEventPurchaseFailedBody.Reason` (set at `kafka/consumer/merchant/consumer.go:269-273`): `unavailable`, `version_conflict`, `insufficient_bundles`.

`CloseReason` on `StatusEventShopClosedBody` is a service-local `byte` enum (`shop/state.go:24-34`): `0` None, `1` SoldOut, `2` ManualClose, `3` Disconnect, `4` Expired, `5` ServerRestart, `6` Empty.

### Produced Events (EVENT_TOPIC_MERCHANT_LISTING)

Envelope `ListingEvent[E]` (`kafka/message/merchant/kafka.go:263-267`):

```
ListingEvent[E] {
  shopId, type, body: E
}
```

| Type constant | Wire string | Body Struct | Description |
|---|---|---|---|
| `ListingEventPurchased` | `LISTING_PURCHASED` | ListingEventPurchasedBody (ListingIndex, BuyerCharacterId, BundleCount, BundlesRemaining) | Listing bundles purchased |

### Produced Commands (COMMAND_TOPIC_COMPARTMENT)

Envelope `Command[E]` (`kafka/message/compartment/kafka.go:24-30`):

```
Command[E] {
  transactionId, characterId, inventoryType, type, body: E
}
```

| Type constant | Wire string | Body Struct | Description |
|---|---|---|---|
| `CommandRelease` | `RELEASE` | ReleaseCommandBody (TransactionId, AssetId, Quantity) | Release item from inventory (on listing add) |
| `CommandAccept` | `ACCEPT` | AcceptCommandBody (TransactionId, TemplateId, asset.AssetData) | Grant item to inventory (purchase, removal, close, Frederick retrieval) |

### Produced Commands (COMMAND_TOPIC_CHARACTER)

Envelope `Command[E]` (`kafka/message/character/kafka.go:31-37`):

```
Command[E] {
  transactionId, worldId, characterId, type, body: E
}
```

| Type constant | Wire string | Body Struct | Description |
|---|---|---|---|
| `CommandRequestChangeMeso` | `REQUEST_CHANGE_MESO` | RequestChangeMesoBody (ActorId, ActorType, Amount) | Deduct or credit mesos (purchase, withdrawal, Frederick retrieval); ActorType is `MERCHANT` or `FREDERICK` |

## Transaction Semantics

- All consumed topics parse `SpanHeader` and `TenantHeader` (`consumer.SetHeaderParsers`); all produced messages carry `SpanHeaderDecorator` and `TenantHeaderDecorator` (`kafka/producer/producer.go:12-20`).
- `transactionId` is not a Kafka header. It is a JSON envelope/body field on the compartment `Command`/`StatusEvent` and the character `Command` types only; the merchant `Command`, merchant `StatusEvent`, and `ListingEvent` envelopes carry no `transactionId`.
- Most merchant status/listing events and compartment/character commands are enqueued to a transactional outbox and published by the outbox drainer (`outbox.EmitProvider` inside `database.ExecuteTransaction`; `main.go:75-83`). The visitor enter/exit paths emit directly through the producer (no accompanying Postgres write).
- Compartment commands carry a `transactionId` for correlation with compartment status events; compartment status events are consumed for logging only.
</content>
