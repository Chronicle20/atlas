# Kafka Integration

## Consumer Group

`Compartment Transfer Service`

## Topics Consumed

| Topic Variable | Consumer | Message Type |
|----------------|----------|--------------|
| `COMMAND_TOPIC_COMPARTMENT_TRANSFER` | compartment_transfer_command | TransferCommand |
| `EVENT_TOPIC_COMPARTMENT_STATUS` | compartment_status_event | StatusEvent[AcceptedEventBody], StatusEvent[ReleasedEventBody], StatusEvent[ErrorEventBody] |
| `EVENT_TOPIC_CASH_COMPARTMENT_STATUS` | cash_compartment_status_event | StatusEvent[StatusEventAcceptedBody], StatusEvent[StatusEventReleasedBody], StatusEvent[StatusEventErrorBody] |
| `EVENT_TOPIC_STORAGE_COMPARTMENT_STATUS` | storage_compartment_status_event | StatusEvent[StatusEventAcceptedBody], StatusEvent[StatusEventReleasedBody], StatusEvent[StatusEventErrorBody] |

## Topics Produced

| Topic Variable | Message Type |
|----------------|--------------|
| `COMMAND_TOPIC_COMPARTMENT` | Command[AcceptCommandBody], Command[ReleaseCommandBody] |
| `COMMAND_TOPIC_CASH_COMPARTMENT` | Command[AcceptCommandBody], Command[ReleaseCommandBody] |
| `COMMAND_TOPIC_STORAGE_COMPARTMENT` | Command[AcceptCommandBody], Command[ReleaseCommandBody] |
| `EVENT_TOPIC_COMPARTMENT_TRANSFER_STATUS` | StatusEvent[StatusEventCompletedBody] |

## Message Types

### TransferCommand (Consumed)

| Field | Type | Description |
|-------|------|-------------|
| transactionId | uuid.UUID | Unique transfer transaction ID |
| worldId | byte | World identifier |
| accountId | uint32 | Account identifier |
| characterId | uint32 | Character identifier |
| assetId | uint32 | Asset identifier |
| fromCompartmentId | uuid.UUID | Source compartment ID |
| fromCompartmentType | byte | Source compartment type |
| fromInventoryType | string | Source inventory type (CHARACTER, CASH_SHOP, STORAGE) |
| toCompartmentId | uuid.UUID | Destination compartment ID |
| toCompartmentType | byte | Destination compartment type |
| toInventoryType | string | Destination inventory type (CHARACTER, CASH_SHOP, STORAGE) |
| referenceId | uint32 | Reference identifier |
| templateId | uint32 | Template identifier (optional) |
| referenceType | string | Reference type (optional) |
| slot | int16 | Slot number (optional) |

### Character Compartment Command (Produced)

| Field | Type | Description |
|-------|------|-------------|
| characterId | uint32 | Character identifier |
| inventoryType | byte | Inventory type |
| type | string | Command type (ACCEPT, RELEASE) |
| body | AcceptCommandBody or ReleaseCommandBody | Command body |

### Cash Shop Compartment Command (Produced)

| Field | Type | Description |
|-------|------|-------------|
| accountId | uint32 | Account identifier |
| compartmentType | byte | Compartment type |
| type | string | Command type (ACCEPT, RELEASE) |
| body | AcceptCommandBody or ReleaseCommandBody | Command body |

### Storage Compartment Command (Produced)

| Field | Type | Description |
|-------|------|-------------|
| worldId | byte | World identifier |
| accountId | uint32 | Account identifier |
| type | string | Command type (ACCEPT, RELEASE) |
| body | AcceptCommandBody or ReleaseCommandBody | Command body |

### StatusEvent[StatusEventCompletedBody] (Produced)

| Field | Type | Description |
|-------|------|-------------|
| characterId | uint32 | Character identifier |
| type | string | Event type (COMPLETED) |
| body.transactionId | uuid.UUID | Transaction ID |
| body.accountId | uint32 | Account identifier |
| body.assetId | uint32 | Asset identifier |
| body.compartmentId | uuid.UUID | Destination compartment ID |
| body.compartmentType | byte | Destination compartment type |
| body.inventoryType | string | Destination inventory type |

## Headers

All messages include:
- Span headers (for distributed tracing)
- Tenant headers (for multi-tenancy)
