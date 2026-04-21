# Shared Field-Scoped Object IDs — PRD

Version: v1
Status: Draft
Created: 2026-04-20

## 1. Overview

Three services (`atlas-monsters`, `atlas-reactors`, `atlas-drops`) currently allocate client-visible object IDs independently in the same numeric range. Because the v83 client stores map objects in a per-field registry keyed by oid, collisions between services overwrite live entities and crash the client.

This task replaces the three independent allocators with a single shared allocation mechanism scoped per tenant. All three services will continue to own their own domain state; only the ID they stamp onto each entity changes source.

Scope note: the client only requires uniqueness within a single field (map instance). Per-tenant is a strictly stronger invariant, chosen here because (a) each service's Redis storage keys entities by `(tenant, id)` with no field component, so per-field allocation would let two entities on different fields collide at the storage layer, and (b) 2B ids per tenant is far larger than any realistic workload consumes.

## 2. Goals

Primary:
- A monster, a reactor, and a drop that coexist in the same field can never be assigned the same oid.
- Allocation is atomic across services — no double-INCR races produce the same value.
- Released IDs (monsters that die, drops that despawn, reactors that are destroyed) return to a per-field free list and are reused LIFO, matching current monster behavior.
- ID range and encoding remain compatible with `uint32` on the wire; no client-side changes.

Non-goals:
- No new microservice. The allocator is a shared Redis key accessed from a small shared Go library.
- No change to the domain models, REST APIs, or Kafka topics of the three services beyond the ID source.
- No change to static-data IDs (NPC template ids, monster classification ids, reactor classification ids). Those live in game data and are not allocated at runtime.
- No backfill of pre-existing in-flight entities. Rolling restart will clear them.

## 3. Design

### 3.1 Shared allocator library

New library `libs/atlas-object-id/` exposes:

```go
type Allocator interface {
    Allocate(ctx context.Context, t tenant.Model) (uint32, error)
    Release(ctx context.Context, t tenant.Model, id uint32) error
    Clear(ctx context.Context, t tenant.Model) error // tenant reset
}

func NewRedisAllocator(client *goredis.Client) Allocator
```

Implementation is Redis-backed, using one Lua script that atomically pops from the free list or increments the counter.

### 3.2 Redis key layout

Per tenant:
- Counter: `atlas:oid:{tenantId}:next`
- Free list (LIFO): `atlas:oid:{tenantId}:free`

Range: `1,000,000 – 2,147,483,647` (positive `int32`, safe for the v83 wire format). The floor of 1,000,000 leaves room for static NPC oids (assigned per-map from WZ data, typically 1–50) without collision. First `Allocate` on a fresh tenant returns `1,000,000`.

### 3.3 Call-site changes

| Service | Replace | With |
|---|---|---|
| atlas-monsters | `monster/id_allocator.go` usage | shared `Allocator.Allocate` keyed on spawn field; keep release on death |
| atlas-reactors | `reactor/registry.go` INCR of `reactors:next_id` | shared `Allocator.Allocate` keyed on reactor's field; add release on `Destroy` |
| atlas-drops | drop-id generator | shared `Allocator.Allocate` keyed on drop's field; add release on despawn/pickup |

No topic or REST contract changes.

### 3.4 Cutover

User has approved a rolling restart with player interruption. Sequence:

1. Deploy new library + updated services together.
2. Destroy all active fields (or hard-restart all three services so their in-memory entity registries are empty on restart).
3. Flush legacy Redis keys: `atlas:monster-ids:*`, `reactors:next_id`, and the drops counter.
4. New allocations begin at `1` per field. No overlap with legacy IDs because no legacy entities remain.

## 4. Functional Requirements

- `Allocate` MUST return a value distinct from all other unreleased IDs for the same tenant, across services, under concurrent load.
- `Release` MUST push the ID to the tenant's free list. Callers should release exactly once per allocation.
- `Clear` MUST delete both counter and free list for the tenant. Used for tenant reset.
- On Redis unavailability, `Allocate` returns an error; services either propagate it or fall back to `MinId` (monster allocator preserves the old fallback semantics).
- Free-list LIFO ordering is preserved so recently released IDs are reused quickly (matches current monster behavior and minimizes gap growth).

## 5. Success Criteria

- On a post-cutover tenant, running the same log-sampling query yields zero cross-kind ID collisions in any sample window.
- Monster kill → respawn still reuses the freed oid (verified via logs).
- No client crash during a 30-minute play session involving monster kills, reactor destructions, and drop spawns on a multi-entity map.

## 6. Out of scope (tracked separately)

- Reactor `type: 999` double-trigger bug in `atlas-reactors/.../processor.go` — tracked as a separate fix.
- Reactor REST `Extract` missing `updateTime` — separate cleanup.
- `WriteAsciiString` ShiftJIS encoding mismatch for GMS tenants — separate investigation.
