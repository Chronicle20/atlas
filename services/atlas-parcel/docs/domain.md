# Domain

## Parcel

### Responsibility

Represents one item/mesos handoff held in Duey's custody between the moment a sender's send saga hands it off and the moment the recipient receives it (or it is returned/discarded/expired). A parcel carries at most one item snapshot plus a meso amount; `HasItem` (custody) / a nil `ItemId` (persisted) distinguishes a meso-only parcel from one carrying an item.

### Core Models

**Model**
- `id`: uuid.UUID - Unique identifier
- `tenantId`: uuid.UUID - Tenant identifier
- `worldId`: world.Id - World identifier
- `senderId`, `senderAccountId`, `senderName`: uint32/uint32/string - Sender identity
- `recipientId`, `recipientAccountId`, `recipientName`: uint32/uint32/string - Recipient identity (`recipientName` is populated end-to-end by the send saga so the expiry sweep's return leg can address the return parcel back to the original sender by display name)
- `message`: string - Sender's note
- `mesoAmount`, `feePaid`: uint32/uint32 - Meso amount carried and delivery fee already collected upstream
- `itemId`: *uint32 - nil for a meso-only parcel
- `itemType`, `quantity`: byte/uint16
- `itemSnapshot`: AssetData - full item stat snapshot, taken at send time (equipment stats, cash id, etc.)
- `status`: string - one of `pending`, `received`, `discarded`, `expired`
- `quick`: bool - true for a "quick delivery" parcel with no transit delay
- `returned`: bool - true for a parcel created by the expiry sweep's return leg
- `createdAt`, `receivableAt`, `expiresAt`: time.Time
- `resolvedAt`: *time.Time - set when the parcel leaves `pending`
- `lastNotified`: *time.Time - set once the arrival notification has been sent (design §7.1/§8)

**Builder**
- Constructs Model instances with builder pattern via `NewBuilder()`
- `validate()` enforces required fields (id, tenantId, senderId, recipientId, status, worldId zero-valued is a legitimate world)
- `Build()` returns error on validation failure

### Invariants

- `ReceivableDelay` (24h) is the normal transit delay before a newly-created parcel becomes receivable; the expiry sweep's return leg sets `ReceivableAt == CreatedAt` (no delay) so a returned parcel is claimable immediately.
- `ExpiryWindow` is 29 days, not 30 — the client's own receive guard computes an unsigned 64-bit day-quotient of the remaining life and refuses the receive unless it is strictly less than 30; 29 keeps every parcel (including a return leg with no transit delay) inside that guard for its whole life. See `parcel/entity.go`'s `ExpiryWindow` doc comment for the full derivation.
- A parcel transitions `pending` -> `received` | `discarded` | `expired`, never back.
- Only a `pending` parcel can be discarded or received; discard/receive against a non-pending parcel is a 409, not a silent no-op.

### Processors

**Processor**
- `GetById`: Retrieves a parcel by id
- `GetForRecipient`: Retrieves a recipient's pending mailbox in a world
- `GetPendingForSender`: Retrieves a sender's still-in-flight pending parcels
- `HasInFlight`: Reports whether a character has any pending parcel (as sender or recipient) — the narrow world-transfer gate check
- `Create`: Persists a new parcel row (custody ACCEPT_TO_PARCEL)
- `Receive`: Transitions a pending parcel to `received`, gated on the caller being the recipient (custody RELEASE_FROM_PARCEL)
- `Discard`: Transitions a pending parcel to `discarded`, gated on the caller being the recipient (REST PATCH)
- `MarkNotified`: Stamps `lastNotified` (REST PATCH /notify, driven by the SHOW_PARCEL consumer)

**Administrator (raw DB helpers)**
- `Create`, `UpdateStatus`, `UpdateStatusIfPending`: single-row writes
- `ClaimExpired`: claim-by-update — atomically selects and stamps `resolvedAt`/`status=expired` on every parcel past `ExpiresAt`, up to a batch cap, across every tenant (ExpiryTask)
- `ClaimNotifiable`: claim-by-update — atomically selects and stamps `lastNotified` on every newly-receivable, not-yet-notified parcel, up to a batch cap, across every tenant (NotificationTask)
- `StampNotified`: batch-stamps `lastNotified` for a set of ids

### Background Sweeps

**ExpiryTask** (`parcel/task.go`) — runs every `PARCEL_EXPIRY_INTERVAL_SECONDS`. Claims expired pending parcels across every tenant in one pass (a claim-by-update sweep must see every tenant, so it deliberately runs outside the request-scoped tenant filter), reconstructs each claimed row's own tenant, and re-enters that tenant's context before issuing the return leg (a new parcel addressed back to the original sender) or discarding a non-returnable parcel.

**NotificationTask** (`parcel/notification_task.go`) — runs every `PARCEL_NOTIFICATION_INTERVAL_SECONDS` (default 5 minutes — much tighter than the hourly expiry sweep, since an arrival notification is time-sensitive in a way a 29-day expiry deadline is not). Claims newly-receivable, not-yet-notified parcels across every tenant the same way, reconstructs and re-enters each claimed row's tenant (with this pod's environment identity layered on top), and emits `PARCEL_ARRIVED` on `EVENT_TOPIC_PARCEL_STATUS`. Skips entirely (does not claim, does not stamp) if that topic is not configured for this deployment.
