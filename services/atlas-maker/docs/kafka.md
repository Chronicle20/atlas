# Kafka — atlas-maker

## Topics Consumed

| Environment Variable | Direction | Consumer Group |
|---------------------|-----------|----------------|
| `EVENT_TOPIC_SAGA_STATUS` | Event | `Maker Service` |

## Topics Produced

| Environment Variable | Direction |
|---------------------|-----------|
| `COMMAND_TOPIC_SAGA` | Command |

## Message Types

### EVENT_TOPIC_SAGA_STATUS

**StatusEvent[StatusEventCompletedBody]** (`kafka/message/saga/kafka.go`)

Consumed when type is `"COMPLETED"`. Releases the character's in-flight
craft guard (`craft/inflight.go`), Track-ed against this transaction id when
the craft saga was emitted. Logged at debug level.

```
{
  "transactionId": uuid.UUID,
  "type":          "COMPLETED",
  "body":          {}
}
```

**StatusEvent[StatusEventFailedBody]** (`kafka/message/saga/kafka.go`)

Consumed when type is `"FAILED"`. Also releases the in-flight craft guard --
a failed saga that never released would reproduce the same `craft_in_progress`
lockout as a completed one that never released. Logged at warn level.

```
{
  "transactionId": uuid.UUID,
  "type":          "FAILED",
  "body": {
    "errorCode":  string,
    "reason":     string,
    "failedStep": string
  }
}
```

Headers: `SpanHeaderParser`, `TenantHeaderParser`, `EnvHeaderParser`. Release
is tenant-scoped: the guard's key is `(tenantId, characterId)`, and the
handler routes through `(tenantId, transactionId)` (the tenant reconciled by
`TenantHeaderParser` onto the message context, and the transaction id
carried by the event itself), so a terminal event for one tenant can never
release another tenant's entry.

### COMMAND_TOPIC_SAGA

**saga.Saga** (`craft/emitter.go`)

Produced when an accepted `POST .../crafts` request builds and emits its
craft saga (`InventoryTransaction`). Key is the saga transaction ID.

## In-Flight Craft Guard

`craft/inflight.go`'s `craftGuard` is a process-local, non-durable
`(tenantId, characterId)` suppression window (design §7 / §4.2.6):

1. `Processor.Create` acquires the guard before any validation work. A
   second `MAKER_SKILL` for the same character while one is still resolving
   is rejected immediately with `craft_in_progress`.
2. A synchronous rejection (validation failure) releases the guard before
   returning -- nothing was emitted, so nothing is left to wait on.
3. An accepted craft leaves the guard held and Tracks the saga's transaction
   id against it, since the eventual terminal event carries only that
   transaction id, never a character id.
4. `kafka/consumer/saga`'s terminal-event handlers release the guard by
   transaction id on both `COMPLETED` and `FAILED`.

A process restart drops every entry; the ordinary server-authoritative
validation path still prevents double-spending a material that is no longer
held, so this is a deliberate design property, not a durability gap.
