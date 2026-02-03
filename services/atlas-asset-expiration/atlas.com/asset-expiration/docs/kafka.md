# Kafka Integration

## Topics Consumed

| Topic | Environment Variable | Direction |
|-------|---------------------|-----------|
| Session Status | `EVENT_TOPIC_SESSION_STATUS` | Event |

## Topics Produced

| Topic | Environment Variable | Direction |
|-------|---------------------|-----------|
| Asset Expire | `COMMAND_TOPIC_ASSET_EXPIRE` | Command |

## Message Types

### StatusEvent (Consumed)

| Field | Type |
|-------|------|
| SessionId | uuid.UUID |
| AccountId | uint32 |
| CharacterId | uint32 |
| WorldId | byte |
| ChannelId | byte |
| Issuer | string |
| Type | string |

Issuer values: `LOGIN`, `CHANNEL`

Type values: `CREATED`, `DESTROYED`

### ExpireCommand (Produced)

| Field | Type |
|-------|------|
| TransactionId | uuid.UUID |
| CharacterId | uint32 |
| AccountId | uint32 |
| WorldId | byte |
| AssetId | uint32 |
| TemplateId | uint32 |
| InventoryType | int8 |
| Slot | int16 |
| ReplaceItemId | uint32 |
| ReplaceMessage | string |
| Source | string |

Source values: `INVENTORY`, `STORAGE`, `CASHSHOP`

## Transaction Semantics

- Expire commands are keyed by AssetId
- Each expire command includes a unique TransactionId
- Commands are emitted independently (no batching)

## Required Headers

- `TENANT_ID`: Propagated via TenantHeaderDecorator
- Span context: Propagated via SpanHeaderDecorator
