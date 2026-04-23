# Shared Field-Scoped Object IDs — Implementation Plan

Created: 2026-04-20

## Phase 1 — Library

1. Create `libs/atlas-object-id/` with its own `go.mod`. Module path: `github.com/Chronicle20/atlas/libs/atlas-object-id`.
2. Define `FieldKey` and `Allocator` in `allocator.go` (see prd.md §3.1).
3. Implement `redisAllocator` with one `EVAL`-based Lua script that:
   - Pops from the free list (LPOP) if non-empty, returning that value.
   - Otherwise `INCR` on the counter and returns the new value.
   - Input keys: counter, free list. Input args: none for `Allocate`; the id to push for `Release`.
4. Implement `Release` via LPUSH on the free list, guarded by a short SISMEMBER-style dedupe if we adopt a set-shadow (decide during implementation; LPUSH alone is fine if callers release exactly once).
5. Implement `Clear` via `DEL` on both keys.
6. Unit tests against a real Redis (miniredis or a containerized Redis — match the pattern already used in `atlas-monsters/.../id_allocator_test.go`).

## Phase 2 — atlas-monsters

1. Replace the allocator in `services/atlas-monsters/atlas.com/monsters/monster/id_allocator.go` with a thin wrapper that calls `atlas-object-id`.
2. Pass the spawn's `FieldKey` into every `Allocate` call. All existing monster-spawn call sites already have the field info; route it through.
3. On death / despawn, call `Release` with the same `FieldKey`. This replaces the existing per-tenant free-list push.
4. Delete the old per-tenant Redis key names from the monster service; the shared library owns the keys now.
5. Update monster tests (`id_allocator_test.go`) to mock/stand up the shared allocator.
6. `go test ./... -count=1` and `go build`.

## Phase 3 — atlas-reactors

1. In `services/atlas-reactors/atlas.com/reactors/reactor/registry.go`, remove the `reactors:next_id` counter and replace with a call to the shared allocator using the reactor's field.
2. In `Destroy` (`reactor/processor.go:119`), call `Release` after `GetRegistry().Remove`.
3. In `DestroyInField` and `DestroyAll`, ensure each destroyed reactor releases its id (or use `Clear(FieldKey)` as a shortcut in `DestroyInField`).
4. Registry init no longer needs to `SetNX` a starting value.
5. Tests + build.

## Phase 4 — atlas-drops

1. Locate the drop allocator (not yet read; expect a similar pattern to monsters).
2. Replace with shared allocator call keyed on the drop's field.
3. On pickup / despawn / consume, call `Release`.
4. Tests + build.

## Phase 5 — Cutover

1. Deploy all three updated services simultaneously (or in any order, since none reads another's Redis keys anymore).
2. `kubectl rollout restart deployment/atlas-monsters deployment/atlas-reactors deployment/atlas-drops -n atlas`.
3. After all pods are Ready, flush the legacy keys. The new per-tenant counter lives at `atlas:oid:{tenantId}:next`; everything below is pre-task cruft:
   ```bash
   kubectl -n atlas exec <redis-pod> -- redis-cli --scan --pattern 'atlas:monster-ids:*' | xargs -r redis-cli DEL
   kubectl -n atlas exec <redis-pod> -- redis-cli DEL reactors:next_id drops:next_id
   # Legacy per-entity storage keys (tenant scoping is new) — drop these too so
   # stale global-keyed records don't linger:
   kubectl -n atlas exec <redis-pod> -- redis-cli --scan --pattern 'reactor:[0-9]*' | xargs -r redis-cli DEL
   kubectl -n atlas exec <redis-pod> -- redis-cli --scan --pattern 'drop:[0-9]*' | xargs -r redis-cli DEL
   ```
4. Sanity check: spawn one monster, one reactor, one drop on a test map; verify IDs are sequential starting from `1,000,000` within the tenant and that types do not collide.

## Phase 6 — Verification

1. Re-run the log-sampling query that found the 73 collisions:
   ```
   kubectl -n atlas logs <channel-pod> | grep -oE '\\"(reactorId|dropId|uniqueId)\\":[0-9]+' | sort -u
   ```
   Expect zero cross-kind duplicates.
2. Play a session that triggers monster kills, reactor destructions, and drop flurries on map 1000000. No client crashes.
3. Verify LIFO reuse: kill a monster, confirm the next spawned monster on the same map takes the freed oid.

## Dependencies / order

Phases 1 must land first. 2–4 are independent and can ship in separate PRs. Phase 5 must only happen after all three services are on the new library.
