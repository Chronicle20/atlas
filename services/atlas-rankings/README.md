# atlas-rankings

Computes per-world character rankings (overall and per job category) for
each tenant on a configurable cadence and serves them over REST. Consumed
by atlas-login to populate the character-select info board (rank, job rank,
movement arrows).

## Endpoints

| Method | Path | Description |
|---|---|---|
| GET | `/api/rankings/characters?ids={id},{id},…` | Bulk fetch. One `rankings` resource per requested character id that has an entry; unknown ids are omitted (callers default to zeros). Empty/unparseable ids → 400. |
| GET | `/api/rankings/characters/{characterId}` | Single fetch; 404 when no entry exists. |

Resource attributes: `worldId`, `rank`, `rankMove`, `jobRank`, `jobRankMove`
(moves are signed: positive = moved up), `computedAt`. Tenant headers
required. No write endpoints — rankings are computed, never client-mutated.

## Recompute

A 60s base ticker (leader-gated via libs/atlas-lock lease
`rankings-recompute`, so the standard 2 replicas never double-compute)
re-enumerates tenants from atlas-tenants each tick and runs a recompute for
every tenant whose configured interval has elapsed. Each cycle:

1. `GET /characters` from atlas-character (full tenant scan).
2. Exclude `gm > 0` characters entirely (not ranked, not counted).
3. Per world: order by `level DESC, experience DESC, characterId ASC`
   (1-based, unique); job rank is the same order restricted to
   `jobId / 100` categories.
4. Moves are `previousRank − newRank` against the prior cycle; first-seen
   characters move 0.
5. Batch upsert on `(tenant_id, character_id)`, then prune rows not
   restamped by this cycle (deleted/became-GM characters drop out) — unless
   the character scan came back empty against a non-empty rankings table,
   in which case the prune is skipped for that cycle to avoid wiping live
   rankings on a possibly-transient empty scan.

The cycle is idempotent and convergent; a crash mid-cycle is repaired by the
next run (moves may read 0 for one cycle). One tenant's failure is logged
and skipped, never fatal.

## Configuration

Per-tenant cadence lives in atlas-tenants:
`GET/POST/PATCH/DELETE /api/tenants/{tenantId}/configurations/rankings` with
attribute `recomputeIntervalMinutes`. Absent/zero → default 60 minutes. The
config is re-read every tick — changes apply without a redeploy.

Environment: standard DB_* and REST_PORT; `REDIS_URL` (leader lease);
`CHARACTERS_SERVICE_URL` / `TENANTS_SERVICE_URL` with `BASE_SERVICE_URL`
fallback; `RANKINGS_LEADER_ELECTION_ENABLED|TTL|REFRESH|BACKOFF` (defaults
true/30s/TTL÷3/5s).

## Scaling note

Recompute cost scales with total tenant character count: one full
`GET /characters` read per tenant per cycle and an O(n log n) in-memory
sort. Acceptable at tens of thousands of characters; adopt list-endpoint
pagination (task-117) as a drop-in improvement if populations outgrow it.
