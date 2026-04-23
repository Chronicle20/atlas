# Shared Field-Scoped Object IDs — Context

Created: 2026-04-20

## The bug

`atlas-monsters`, `atlas-reactors`, and `atlas-drops` each run an independent Redis-backed allocator in the same numeric range (1,000,000,000 – 2,000,000,000). The v83 client keys map objects by `oid`; when two services mint the same value for the same field, the client's map-object registry gets overwritten and later dereferences crash the game.

## Evidence collected 2026-04-21

Sampled 178 distinct entity IDs from `atlas-channel` logs on tenant `ec876921-c363-4cc6-9c51-5bb8d57f9553`:

- 89 drops, 25 reactors, 64 monster uniqueIds
- **73 cross-kind collisions** in the sample
- Every reactor ID in `1000000001–1000000015` also appeared as a monster `uniqueId`; every drop ID from `1000000016` upward also appeared as both a reactor ID and a monster `uniqueId`.

Representative collisions:

```
1000000003  reactorId + uniqueId
1000000018  dropId    + reactorId + uniqueId
1000000025  dropId    + reactorId + uniqueId
```

## Why restarts masked it

Restarting `atlas-reactors` resets the reactor counter to 1B+1 while monsters/drops counters advance independently. For a short window the per-oid class assignments on the client diverge, so the crash doesn't reproduce. Once the counters re-converge, collisions resume.

## Current allocator locations

- `services/atlas-monsters/atlas.com/monsters/monster/id_allocator.go` — per-tenant counter `atlas:monster-ids:{tenantId}:next`, LIFO recycle via `atlas:monster-ids:{tenantId}:free`.
- `services/atlas-reactors/atlas.com/reactors/reactor/registry.go` (~line 20–89) — global counter `reactors:next_id`, no recycle.
- `services/atlas-drops/...` — per-tenant counter; no recycle (confirm during implementation).

## Uniqueness scope

The MapleStory v83 client only requires oids to be unique **within a single field** (world, channel, map, instance). Per-tenant is stricter than the client's threat model but matches the server-side storage model: each service keys entities by `(tenant, id)`, so per-tenant allocation keeps storage lookups simple. 2B ids per tenant is far more than needed.

## Related

Separately identified in the same session: reactor 2001's `type: 999` state event causes `persistsAtFinalState()` in `atlas-reactors/.../reactor/processor.go` to return true, triggering drops twice per destruction. That bug amplifies collision risk (more drops per event) but is not the root cause of the crash and is tracked separately.
