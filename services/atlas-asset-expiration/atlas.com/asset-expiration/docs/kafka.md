# Kafka Integration

## Topics Consumed

| Topic | Environment Variable | Direction |
|-------|---------------------|-----------|
| Session Status | `EVENT_TOPIC_SESSION_STATUS` | Event |

## Topics Produced

| Topic | Environment Variable | Direction |
|-------|---------------------|-----------|
| Storage Expire | `COMMAND_TOPIC_STORAGE` | Command |
| Cash Shop Expire | `COMMAND_TOPIC_CASH_SHOP` | Command |
| Compartment Expire | `COMMAND_TOPIC_COMPARTMENT` | Command |

## Message Types

### StatusEvent (Consumed)

| Field | Type |
|-------|------|
| SessionId | uuid.UUID |
| AccountId | uint32 |
| CharacterId | uint32 |
| WorldId | world.Id |
| ChannelId | channel.Id |
| Issuer | string |
| Type | string |

Issuer values: `LOGIN`, `CHANNEL`

Type values: `CREATED`, `DESTROYED`

Only `LOGIN` + `CREATED` events trigger an immediate expiration check and session tracking. `DESTROYED` events remove the session from tracking.

### StorageExpireCommand (Produced)

| Field | Type |
|-------|------|
| TransactionId | uuid.UUID |
| WorldId | world.Id |
| AccountId | uint32 |
| Type | string |
| Body.CharacterId | uint32 |
| Body.AssetId | uint32 |
| Body.TemplateId | uint32 |
| Body.InventoryType | int8 |
| Body.Slot | int16 |
| Body.ReplaceItemId | uint32 |
| Body.ReplaceMessage | string |

Type value: `EXPIRE`

### CashShopExpireCommand (Produced)

| Field | Type |
|-------|------|
| CharacterId | uint32 |
| Type | string |
| Body.AccountId | uint32 |
| Body.WorldId | world.Id |
| Body.AssetId | uint32 |
| Body.TemplateId | uint32 |
| Body.InventoryType | int8 |
| Body.Slot | int16 |
| Body.ReplaceItemId | uint32 |
| Body.ReplaceMessage | string |

Type value: `EXPIRE`

### CompartmentExpireCommand (Produced)

| Field | Type |
|-------|------|
| TransactionId | uuid.UUID |
| CharacterId | uint32 |
| InventoryType | byte |
| Type | string |
| Body.AssetId | uint32 |
| Body.TemplateId | uint32 |
| Body.Slot | int16 |
| Body.ReplaceItemId | uint32 |
| Body.ReplaceMessage | string |

Type value: `EXPIRE`

## Transaction Semantics

- All expire commands are keyed by AssetId
- `StorageExpireCommand` and `CompartmentExpireCommand` include a unique TransactionId
- `CashShopExpireCommand` does not include a TransactionId
- Commands are emitted independently (no batching)

## Required Headers

- Tenant context: Propagated via `TenantHeaderDecorator`
- Span context: Propagated via `SpanHeaderDecorator`

Consumer header parsers: `SpanHeaderParser`, `TenantHeaderParser`
