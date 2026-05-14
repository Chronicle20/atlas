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

A single pod's parallelism is bounded by `WZ_EXTRACT_PARALLELISM` (default `runtime.NumCPU()`). The consumer uses the opt-in prefix-commit worker pool from `libs/atlas-kafka` (see `consumer.SetMaxInFlight`): up to N units run concurrently per pod, and offsets are committed in partition order (failed units block the commit cursor, matching at-least-once redelivery semantics).

`WZ_EXTRACT_PARALLELISM` controls three things:
1. **Topic provisioning hint**: partition count must be ≥ this value so cross-pod parallelism scales.
2. **In-pod consumer concurrency** (`SetMaxInFlight`): how many units a single pod processes simultaneously.
3. **Legacy `Extract` whole-list pool size** (used by tests).

With 16 partitions and `replicas=3`, a job's units distribute across partitions (different `wzFile` keys → different partitions); each pod processes its assigned partitions with up to `WZ_EXTRACT_PARALLELISM` concurrent goroutines. Wall-clock for a job ≈ `max(per-WZ-time) / min(replicas, partitions) / NumCPU_per_pod`.

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
