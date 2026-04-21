# Shared Field-Scoped Object IDs — Risks

Created: 2026-04-20

## R1 — Redis as single point of failure for all spawns

Severity: medium

Today a Redis outage breaks monster/drop/reactor spawns in each service independently. After this change, one Redis still covers all three, but the blast radius per allocation call is the same. No net change. No local fallback — a local counter would re-introduce collisions.

Mitigation: existing retry/backoff on Redis errors in each service; surface allocation failures loudly in logs.

## R2 — Drift if a service forgets to release

Severity: medium

If one service allocates but never releases (e.g. a reactor destruction path that skips the new `Release` call), the counter grows without reuse. The field's free list stays small, new oids keep climbing.

Mitigation: audit every destroy/despawn/consume path during Phases 2–4. Add a periodic cleanup task (optional, phase-2 follow-up) that compares live registry size to counter value and logs drift.

## R3 — Per-field Redis key explosion

Severity: low

Every field instance gets its own counter + free-list pair. Long-lived instanced fields (expeditions, party quests) accumulate keys that are only deleted via `Clear`.

Mitigation: `Clear` on field teardown is mandatory, not optional. Add a TTL on the keys (e.g. 24h, refreshed on activity) as a safety net if teardown paths miss a case.

## R4 — Cutover leaves phantom in-flight entities

Severity: medium (mitigated by user's choice)

User approved a rolling restart that kicks players out of fields. Phantom entities from before the restart are destroyed as part of the restart, so post-cutover counters starting from 1 won't collide with anything real.

Mitigation: confirm each service's startup routine actually drops all in-memory entity state (no disk-backed registry restore). Verify during Phase 5.

## R5 — Lua script bugs are hard to roll back

Severity: low

The allocator's correctness lives in one Lua script. A bug there (e.g. returning 0, returning a duplicate) would propagate to all three services.

Mitigation: unit tests in Phase 1 before any service migrates. Keep the script trivial: one branch (pop-or-incr). Resist feature creep.

## R6 — `uint32` exhaustion

Severity: very low

Range is `1 – 2,147,483,647` per field. Even at 1M allocations/day per field (implausible), exhaustion is ~5,880 years out.

Mitigation: none needed.

## R7 — Library module boundary regret

Severity: low

If we later want allocation features that require state beyond Redis (auditing, metrics collection, cross-field quotas), we'll want a real `atlas-object-ids` service and this library becomes a client of it. The `Allocator` interface is designed to let that swap happen without callers changing.

Mitigation: keep the interface narrow (the three methods in PRD §3.1). Don't leak Redis types into the public API.
