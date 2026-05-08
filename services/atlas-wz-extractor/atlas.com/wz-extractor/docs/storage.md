# atlas-wz-extractor storage

## Redis schema

All keys live under the namespace `wz-extractor:`.

### `wz-extractor:job:{jobId}` (HASH)

| Field | Type | Notes |
|---|---|---|
| `tenantId` | string (UUIDv4) | informational; jobId itself is the access key |
| `region` | string | e.g. "GMS" |
| `majorVersion` | int (decimal string) | |
| `minorVersion` | int (decimal string) | |
| `status` | enum (`pending`, `running`, `completed`, `completed_with_errors`, `failed`) | |
| `unitsTotal` | int (decimal string) | |
| `unitsCompleted` | int (decimal string) | `HINCRBY` target |
| `unitsFailed` | int (decimal string) | `HINCRBY` target |
| `xmlOnly` | bool (`true` / `false`) | |
| `imagesOnly` | bool | |
| `createdAt` | RFC3339 | |
| `updatedAt` | RFC3339 | |
| `completedAt` | RFC3339 \| "" | empty until terminal |

TTL: 24h, set on Create.

### `wz-extractor:job:{jobId}:units` (HASH)

Field name = WZ file base name (e.g. `Map.wz`). Value = JSON:

```json
{
  "status": "pending|running|succeeded|failed|skipped",
  "startedAt": "RFC3339",
  "completedAt": "RFC3339",
  "error": "optional error string"
}
```

TTL: shares the parent's 24h.

### `wz-extractor:tenant-lock:{tenantId}:{region}:{maj}.{min}` (STRING)

Value: `jobId` of the holder (so debugging tools can identify ownership).
TTL: 60 minutes; auto-refreshed every 20 minutes by the dispatcher's
goroutine. Released by the "last one home" consumer with a Lua
compare-and-delete (only if the value still matches the holder's jobId).

## Idempotency invariants

1. `MarkUnitRunning` is gated by WATCH on the unit's hash field. If the unit
   is already in a terminal state (succeeded / failed / skipped), the call
   returns `claimed=false` and the consumer skips the work. This is the
   redelivery guard.
2. `FinalizeUnit` is gated the same way. A redelivered finalize over an
   already-terminal unit does not increment counters.
3. `MarkJobTerminal` uses WATCH on the job's `status` field. Only a transition
   from `running` to a terminal status succeeds. The "last one home" race is
   resolved by exactly one CAS winner.

## Multi-tenancy

The job hash is keyed by jobId (UUIDv4) and stores `tenantId` as a field. The
tenant lock key is tenant-scoped. The status endpoint receives only the
jobId; UUIDv4 unguessability is the access control, consistent with the rest
of the service today.

## Connecting

`atlas-wz-extractor` uses the shared `libs/atlas-redis` connection. Required
env: `REDIS_URL`. Optional: `REDIS_PASSWORD`. Default `REDIS_URL` is
`localhost:6379` per the lib.
