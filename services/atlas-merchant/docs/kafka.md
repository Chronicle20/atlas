# Kafka

## Topics Consumed

| Topic Variable | Consumer Name | Direction |
|---|---|---|
| `COMMAND_TOPIC_MERCHANT` | merchant_command | Command |
| `EVENT_TOPIC_CHARACTER_STATUS` | character_status | Event |
| `EVENT_TOPIC_COMPARTMENT_STATUS` | compartment_status | Event |

## Topics Produced

| Topic Variable | Direction |
|---|---|
| `EVENT_TOPIC_MERCHANT_STATUS` | Event |
| `EVENT_TOPIC_MERCHANT_LISTING` | Event |
| `COMMAND_TOPIC_COMPARTMENT` | Command |
| `COMMAND_TOPIC_CHARACTER` | Command |

## Message Types

### Consumed Commands (COMMAND_TOPIC_MERCHANT)

All merchant commands share the envelope:

```
Command[E] {
  worldId, channelId, characterId, type, body: E
}
```

| Type | Body Struct | Description |
|---|---|---|
| `PLACE_SHOP` | CommandPlaceShopBody | Create a new shop |
| `OPEN_SHOP` | CommandOpenShopBody | Transition shop from Draft to Open |
| `CLOSE_SHOP` | CommandCloseShopBody | Close a shop manually |
| `ENTER_MAINTENANCE` | CommandEnterMaintenanceBody | Enter maintenance mode |
| `EXIT_MAINTENANCE` | CommandExitMaintenanceBody | Exit maintenance mode |
| `ADD_LISTING` | CommandAddListingBody | Add an item listing |
| `REMOVE_LISTING` | CommandRemoveListingBody | Remove a listing by index |
| `UPDATE_LISTING` | CommandUpdateListingBody | Update listing price/bundles |
| `PURCHASE_BUNDLE` | CommandPurchaseBundleBody | Purchase bundles from a listing |
| `ENTER_SHOP` | CommandEnterShopBody | Enter a shop as visitor |
| `EXIT_SHOP` | CommandExitShopBody | Exit a shop as visitor |
| `SEND_MESSAGE` | CommandSendMessageBody | Send a chat message in a shop |
| `RETRIEVE_FREDERICK` | CommandRetrieveFrederickBody | Retrieve items/mesos from Frederick |

### Consumed Events (EVENT_TOPIC_CHARACTER_STATUS)

```
StatusEvent[E] {
  characterId, type, body: E
}
```

| Type | Body Struct | Description |
|---|---|---|
| `LOGOUT` | StatusEventLogoutBody | Character disconnected; closes active character shops |

### Consumed Events (EVENT_TOPIC_COMPARTMENT_STATUS)

```
StatusEvent[E] {
  transactionId, characterId, compartmentId, type, body: E
}
```

| Type | Body Struct | Description |
|---|---|---|
| `ACCEPTED` | AcceptedEventBody | Item accept confirmation (logged) |
| `RELEASED` | ReleasedEventBody | Item release confirmation (logged) |
| `ERROR` | ErrorEventBody | Compartment operation error (logged) |

### Produced Events (EVENT_TOPIC_MERCHANT_STATUS)

```
StatusEvent[E] {
  characterId, type, body: E
}
```

| Type | Body Struct | Description |
|---|---|---|
| `SHOP_OPENED` | StatusEventShopOpenedBody | Shop transitioned to Open |
| `SHOP_CLOSED` | StatusEventShopClosedBody | Shop closed |
| `MAINTENANCE_ENTERED` | StatusEventVisitorBody | Shop entered maintenance |
| `MAINTENANCE_EXITED` | StatusEventVisitorBody | Shop exited maintenance |
| `VISITOR_ENTERED` | StatusEventVisitorBody | Visitor entered shop |
| `VISITOR_EXITED` | StatusEventVisitorBody | Visitor exited shop |
| `VISITOR_EJECTED` | StatusEventVisitorBody | Visitor ejected from shop |
| `CAPACITY_FULL` | StatusEventCapacityFullBody | Shop at max visitor capacity |
| `PURCHASE_FAILED` | StatusEventPurchaseFailedBody | Purchase attempt failed |
| `FREDERICK_NOTIFICATION` | StatusEventFrederickNotificationBody | Frederick retrieval reminder |

### Produced Events (EVENT_TOPIC_MERCHANT_LISTING)

```
ListingEvent[E] {
  shopId, type, body: E
}
```

| Type | Body Struct | Description |
|---|---|---|
| `LISTING_PURCHASED` | ListingEventPurchasedBody | Listing bundle purchased |

### Produced Commands (COMMAND_TOPIC_COMPARTMENT)

```
Command[E] {
  transactionId, characterId, inventoryType, type, body: E
}
```

| Type | Body Struct | Description |
|---|---|---|
| `RELEASE` | ReleaseCommandBody | Release item from inventory (on listing add) |
| `ACCEPT` | AcceptCommandBody | Grant item to inventory (on purchase, removal, close, Frederick retrieval) |

### Produced Commands (COMMAND_TOPIC_CHARACTER)

```
Command[E] {
  transactionId, worldId, characterId, type, body: E
}
```

| Type | Body Struct | Description |
|---|---|---|
| `REQUEST_CHANGE_MESO` | RequestChangeMesoBody | Deduct or credit mesos (purchase, Frederick retrieval) |

## Transaction Semantics

- All commands and events include `SpanHeader` and `TenantHeader` via Kafka header parsers/decorators.
- Compartment commands include a `transactionId` for correlation with status events.
- Character meso commands include a `transactionId`.
- Purchase messages are keyed by characterId for ordering.
- Compartment status events are consumed for logging; no compensating actions are implemented.
