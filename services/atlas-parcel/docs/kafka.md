# Kafka

## Topics Consumed

### COMMAND_TOPIC_PARCEL_CUSTODY

Parcel custody commands, driven by the send/withdraw sagas (and their compensators).

| Command Type | Body Type | Description |
|--------------|-----------|-------------|
| `ACCEPT_TO_PARCEL` | AcceptToParcelCommandBody | Create a parcel row in custody (send saga forward leg) |
| `RELEASE_FROM_PARCEL` | ReleaseFromParcelCommandBody | Transition a pending parcel to received and release custody (withdraw saga forward leg) |
| `RESTORE_PARCEL` | RestoreParcelCommandBody | Un-resolve a parcel released by a withdraw whose downstream accept then failed (late-comp inverse of RELEASE_FROM_PARCEL). Idempotent: 0 rows affected is success. Fire-and-forget — emits ERROR on failure only. |
| `REMOVE_PARCEL` | RemoveParcelCommandBody | Hard-delete a still-pending parcel row created by a late ACCEPT_TO_PARCEL after its saga already compensated (late-comp inverse of ACCEPT_TO_PARCEL). Idempotent: 0 rows affected is success. Fire-and-forget — emits ERROR on failure only. |

---

## Topics Produced

### EVENT_TOPIC_PARCEL_CUSTODY_STATUS

Parcel custody status (ack) events, echoing the command's `transactionId` so the saga orchestrator can complete/fail the step.

| Event Type | Body Type | Description |
|------------|-----------|-------------|
| `ACCEPTED` | StatusEventAcceptedBody | Parcel created in custody |
| `RELEASED` | StatusEventReleasedBody | Parcel released from custody (received) |
| `ERROR` | StatusEventErrorBody | Custody operation failed |

### EVENT_TOPIC_PARCEL_STATUS

Player-facing parcel status events — a sibling of the custody status topic, not the same topic: custody acks saga steps, this one notifies players. Addressed by `characterId` to whichever party the event concerns.

| Event Type | Body Type | Description |
|------------|-----------|-------------|
| `PARCEL_ARRIVED` | StatusEventParcelArrivedBody | A parcel has become receivable; emitted once per parcel by NotificationTask, addressed to the recipient (design §7.1) |
| `PARCEL_SENT` | StatusEventParcelSentBody | A parcel's send saga completed (accept_to_parcel, the saga's last step); emitted by the custody consumer's ACCEPT_TO_PARCEL handler, addressed to the sender |
| `PARCEL_RECEIVED` | StatusEventParcelReceivedBody | A parcel was released from custody (received); emitted by the custody consumer's RELEASE_FROM_PARCEL handler, addressed to the recipient |

---

## Message Types

### Command Messages

**Command[E]**
```
transactionId: UUID
type: string
body: E
```

**AcceptToParcelCommandBody**
```
parcelId: UUID
characterId: uint32
worldId: world.Id
senderAccountId: uint32
senderName: string
recipientId: uint32
recipientAccountId: uint32
recipientName: string
mesoAmount: uint32
feePaid: uint32
quick: bool
message: string
receivableAt: time.Time
expiresAt: time.Time
hasItem: bool
templateId: uint32
quantity: uint32
strength, dexterity, intelligence, luck, hp, mp: uint16
weaponAttack, magicAttack, weaponDefense, magicDefense: uint16
accuracy, avoidability, hands, speed, jump, slots: uint16
level: byte
itemLevel: byte
itemExp: uint32
ringId: uint32
viciousCount: uint32
flags: uint16
owner: string
```

**ReleaseFromParcelCommandBody**
```
parcelId: UUID
recipientId: uint32
```

**RestoreParcelCommandBody**
```
parcelId: UUID
```

**RemoveParcelCommandBody**
```
parcelId: UUID
```

### Status (Ack) Messages

**StatusEvent[E]** (custody)
```
transactionId: UUID
type: string
body: E
```

**StatusEventAcceptedBody**
```
parcelId: UUID
```

**StatusEventReleasedBody**
```
parcelId: UUID
```

**StatusEventErrorBody**
```
error: string
```

### Arrival Notification Messages

**StatusEvent[E]** (parcel)
```
characterId: uint32
type: string
body: E
```

**StatusEventParcelArrivedBody**
```
senderName: string
hasItem: bool
```

**StatusEventParcelSentBody**
```
(no fields)
```

**StatusEventParcelReceivedBody**
```
parcelId: UUID
```

---

## Transaction Semantics

- Custody commands carry `transactionId` so the saga orchestrator can correlate the ack with the step that issued the command.
- `ACCEPT_TO_PARCEL` and `RELEASE_FROM_PARCEL` are the forward legs of the send and withdraw sagas, respectively; `RESTORE_PARCEL` and `REMOVE_PARCEL` are their late-compensation inverses and are idempotent (0 rows affected is success, not an error).
- The custody status topic and the parcel status topic are independent: a parcel's custody lifecycle (accepted/released) is orthogonal to whether its arrival, send, or receipt has been announced to the player.
- The arrival notification sweep (NotificationTask) is skipped entirely — no claim, no stamp — when `EVENT_TOPIC_PARCEL_STATUS` is not configured, so an unconfigured deployment never silently marks a parcel notified without actually notifying.
- `PARCEL_SENT` and `PARCEL_RECEIVED` are emitted synchronously from the custody command consumer's `ACCEPT_TO_PARCEL` and `RELEASE_FROM_PARCEL` handlers, in the same buffered emit as the corresponding custody status ack.
