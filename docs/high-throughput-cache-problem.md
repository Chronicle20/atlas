# High-Throughput Cache Problem

## Overview

While most Atlas services can migrate from in-memory `sync.Once` + `map` registries to Redis with negligible latency impact (~0.1-0.5ms per operation), a subset of services have update frequencies where naive per-operation Redis calls could create throughput pressure.

This document captures those services for separate, targeted optimization work.

---

## Latency Context

| Operation | In-Memory | Redis (same-network) | Factor |
|-----------|-----------|---------------------|--------|
| Single key lookup | ~100ns | ~0.1-0.5ms | ~1000x |
| Typical game action (3-5 lookups + writes) | ~500ns | ~0.5-2ms | ~1000-4000x |
| High-frequency positional update | ~100ns | ~0.1-0.5ms per call | ~1000x |

For standard game actions (NPC conversations, party operations, buff application, skill cooldowns), the absolute latency cost of Redis is 1-2ms — imperceptible to players. The services below are the exceptions where update volume or latency sensitivity requires a different approach.

---

## Affected Services

### 1. atlas-monsters — Monster Position/HP/Status Updates

**File**: `services/atlas-monsters/atlas.com/monsters/monster/registry.go`

**Problem**: Monster state (position, HP, MP, status effects, controller assignment) is updated at high frequency — every monster movement tick, every damage event, every status effect tick. A busy map with 30+ monsters each updating position multiple times per second could generate hundreds of Redis round trips per second per map.

**Scale**: Per-map, per-monster. Multiplied across all active maps and all tenants.

**Considerations**:
- Position updates are transient and lossy-tolerant — a slightly stale position is acceptable for brief intervals.
- HP/damage updates are NOT lossy-tolerant — missed damage or double-damage is a correctness issue.
- Monster AI decisions read current state and write new state in tight loops.

---

### 2. atlas-character — Real-Time Character Position (Temporal Data)

**File**: `services/atlas-character/atlas.com/character/character/temporal_data.go`

**Problem**: Tracks real-time character position (x, y) and stance. Updated on every character movement packet — potentially dozens of times per second per active character.

**Scale**: Per-character. Multiplied across all online characters.

**Considerations**:
- Position data is consumed by other services (maps, monsters, drops) for proximity checks.
- Staleness of ~100-200ms is likely acceptable for most consumers.
- Write volume is the primary concern, not read latency.

---

### 3. atlas-pets — Real-Time Pet Position (Temporal Data)

**File**: `services/atlas-pets/atlas.com/pets/pet/temporal_data.go`

**Problem**: Identical pattern to character position. Pet position (x, y, stance, foothold) updated on every pet movement.

**Scale**: Per-active-pet. Lower volume than characters (not all characters have active pets).

**Considerations**:
- Same staleness tolerance as character position.
- Could share the same optimization strategy.

---

### 4. atlas-maps — Monster Spawn Cooldown Tracking

**File**: `services/atlas-maps/atlas.com/maps/map/monster/registry.go`

**Problem**: Tracks `NextSpawnAt` timestamps for every spawn point on every active map. Updated whenever a monster spawns or is killed. A periodic task checks all spawn points for respawn eligibility.

**Scale**: Per-spawn-point, per-map. Potentially thousands of spawn points across all active maps.

**Considerations**:
- The periodic check iterates ALL spawn points — doing this as individual Redis reads would be expensive.
- Spawn timing precision matters (boss respawn fairness).

---

## Potential Optimization Strategies

These are captured for future evaluation — not prescriptive.

### Write Batching / Pipelining

Redis pipelines batch multiple commands into a single network round trip. Instead of N individual SET calls, a pipeline sends all N at once and reads all N responses. This reduces the effective per-operation cost from ~0.5ms to ~0.5ms/N.

**Applicability**: Monster position updates, character position updates, spawn point checks.

### Write Coalescing

Buffer high-frequency writes in-memory and flush to Redis at a fixed interval (e.g., every 100-200ms). Intermediate states are lost, but only the latest state matters for positional data.

**Applicability**: Character position, pet position, monster position.

**Trade-off**: Adds a small staleness window. In-memory buffer reintroduces a small amount of process-local state, but it's transient and self-healing (next flush corrects it).

### Lua Scripting for Atomic Operations

Redis Lua scripts execute atomically on the server. Complex read-modify-write operations (monster damage calculation, spawn point iteration) can run server-side, eliminating multiple round trips.

**Applicability**: Monster HP updates (read current HP, apply damage, check death), spawn point batch checks.

### Hybrid: Redis + Local Read Cache

Keep a short-lived local read cache (50-100ms TTL) for frequently-read, infrequently-written data. Writes go directly to Redis. Reads check local cache first.

**Applicability**: Monster position (for AI decisions that read position many times between writes), character position (for proximity queries).

**Trade-off**: Reintroduces bounded staleness. Acceptable for lossy-tolerant data like positions.

---

## Next Steps

- Benchmark actual update frequencies per service in a representative game scenario
- Prototype write coalescing for character/pet position with configurable flush intervals
- Prototype Redis pipelining for monster state batch updates
- Measure Redis throughput under expected load to validate capacity
- Determine which strategy (or combination) best fits each service's consistency requirements
