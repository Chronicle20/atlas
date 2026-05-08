# atlas-wz-extractor Kafka topology

## Topics

| Direction | Env var | Purpose | Recommended config |
|---|---|---|---|
| Consume | `COMMAND_TOPIC_WZ_EXTRACTION` | One `START_EXTRACTION_UNIT` message per WZ file in a job. | partitions ≥ 16 (must be ≥ `WZ_EXTRACT_PARALLELISM`); replication 3; cleanup `delete`; retention 24h. |

The dispatcher (REST `POST /api/wz/extractions`) is also a producer on this
topic; it does not run any unit synchronously even with `replicas=1`.

## Consumer group

- Group ID: `wz-extractor-extraction`
- Header parsers: `consumer.SpanHeaderParser`, `consumer.TenantHeaderParser`
- **Start offset: `kafka.FirstOffset`** (deviates from atlas-data's `LastOffset` parity — see below)
- Persistent handler config (matches atlas-data)

### Why `FirstOffset` here

Atlas-data uses `kafka.LastOffset` because its commands are fire-and-forget — losing a few startup-time messages is acceptable. Atlas-wz-extractor's unit messages are tied to durable Redis job state and **must not** be silently dropped on first-start, group rename, or operator-driven offset reset.

`FirstOffset` replays from offset 0 only on first-ever start of a brand-new consumer group; on every subsequent restart, the committed offset wins. Replay risk is bounded by Kafka retention (24h) and is harmless thanks to the WATCH guard in `MarkUnitRunning` (already-terminal units skip cleanly) and the orphan-handling path in the handler (messages whose job hash has expired log + skip + commit).

## Within-pod parallelism

A single pod's parallelism is bounded by the Kafka partitions assigned to it.
With partition count of 16 and `replicas=3`, each pod is assigned ~5
partitions, which means up to 5 units run in parallel per pod. With
`replicas=1`, all 16 partitions are assigned to one pod.

## Message envelope

```
{
  "type": "START_EXTRACTION_UNIT",
  "body": {
    "jobId": "uuid-v4",
    "wzFile": "Map.wz",
    "xmlOnly": false,
    "imagesOnly": false
  }
}
```

Headers: standard tenant + span headers (`SpanHeaderDecorator`,
`TenantHeaderDecorator`). Key: hash of jobId so a job's units have a stable
key, but partition count > 1 ensures cross-job parallelism.

## Idempotency

Units are at-least-once. The consumer guards against redelivery via Redis
WATCH/MULTI/EXEC on the unit's status field — see `docs/storage.md`.
