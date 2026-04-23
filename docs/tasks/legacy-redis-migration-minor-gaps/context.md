# Redis Migration Minor Gaps — Context

**Last Updated: 2026-02-19**

## Key Files

### atlas-messengers

| File | Role |
|------|------|
| `services/atlas-messengers/atlas.com/messengers/messenger/processor.go` | Contains `createAndJoinLock` (line 58). 4 lock sites: Create (63), RequestInvite (276), CreateAndEmit (432), RequestInviteAndEmit (614) |
| `services/atlas-messengers/atlas.com/messengers/messenger/registry.go` | Already Redis-backed via `atlas.TenantRegistry` + `atlas.IDGenerator` |
| `services/atlas-messengers/atlas.com/messengers/main.go` | Service init — already calls `atlas.Connect` + `messenger.InitRegistry` + `character2.InitRegistry` |

### atlas-npc-shops

| File | Role |
|------|------|
| `services/atlas-npc-shops/atlas.com/npc/shops/cache.go` | `ConsumableCache` singleton: `sync.Once` + `sync.RWMutex` + `map[uuid.UUID][]consumable.Model` |
| `services/atlas-npc-shops/atlas.com/npc/data/consumable/model.go` | `consumable.Model` — 50+ unexported fields. Nested: `SummonModel`, `RewardModel`, maps, slices |
| `services/atlas-npc-shops/atlas.com/npc/shops/registry.go` | Already Redis-backed via `atlas.TenantRegistry` + Redis Sets for reverse index |
| `services/atlas-npc-shops/atlas.com/npc/main.go` | Service init — already calls `atlas.Connect` + `shops.InitRegistry` |

### Shared Infrastructure

| File | Role |
|------|------|
| `libs/atlas-redis/lock.go` | `Lock` type: `Acquire` (SETNX), `Release` (DEL), `Extend` (EXPIRE). No spin-wait, no ownership. |
| `services/atlas-inventory/atlas.com/inventory/compartment/lock_registry.go` | Reference: `DistributedMutex` with spin-wait + Lua-based safe release. More robust but service-local. |

## Key Decisions

1. **Per-character lock key (messengers)** — Use `messenger:create-lock:{tenantKey}:{characterId}` rather than a global lock. This scopes contention to the specific character being operated on, allowing concurrent messenger creation for different characters. The race condition is per-character (two requests for the same character), not global.

2. **Retry loop in processor, not library (messengers)** — The `atlas-redis` `Lock.Acquire` is single-shot by design. Add a spin-wait loop in `processor.go` rather than modifying the shared library. This keeps the library simple and lets each consumer choose its own retry strategy.

3. **TenantRegistry for consumable cache (npc-shops)** — Use `TenantRegistry[uint32, consumable.Model]` keyed by `itemId`, rather than storing the entire `[]consumable.Model` slice as a single value. This follows the established pattern and allows individual item lookups if needed in the future.

4. **Custom JSON serialization (npc-shops)** — `consumable.Model` needs `MarshalJSON`/`UnmarshalJSON` in a new `model_json.go` file. Follow the same pattern used for `ConversationContext` in atlas-npc-conversations and `projection.Model` in atlas-storage: parallel struct with exported fields.

5. **Deadlock fix is prerequisite (messengers)** — The latent deadlock in `RequestInvite` → `Create` must be fixed before introducing distributed locks, because the fix changes the lock acquisition pattern that the distributed lock will implement.

## Dependencies Between Tasks

```
Phase 1 (messengers):
  1.1 Fix deadlock (extract internal create) ─┐
  1.2 Replace sync.RWMutex with Redis lock ────┤── depends on 1.1
  1.3 Update tests ────────────────────────────┘── depends on 1.1 + 1.2

Phase 2 (npc-shops):
  2.1 Add MarshalJSON/UnmarshalJSON ───────────┐
  2.2 Replace ConsumableCache with Redis ───────┤── depends on 2.1
  2.3 Update main.go ──────────────────────────┤── depends on 2.2
  2.4 Add cache tests ─────────────────────────┘── depends on 2.2
```

Phases 1 and 2 are independent and can be done in either order.
