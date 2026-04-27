# The Inbox Pattern

An **inbox** is an in-process, per-key map that holds a single-use handoff between an asynchronous producer (e.g. a Kafka consumer) and a synchronous consumer (e.g. a packet handler) inside the same process. The producer overwrites entries (last-writer-wins); the consumer reads-and-clears.

## When to use

Reach for an inbox when:

- An external decision needs to influence the **next** packet/response that fires for a given key.
- The decision arrives at a different time than the consumption point — usually via Kafka or another async channel.
- The handoff is single-use: a stale entry should not be served twice.

## How it differs from neighbouring patterns

| | Inbox | Registry | Cache |
|---|---|---|---|
| Lifetime | Single-use; cleared by reader | Long-lived; multi-read | Look-aside; backed by a source of truth |
| Eviction | Read clears; explicit `Evict` on lifecycle events | Owned by writer | TTL or LRU |
| Reader semantics | `TakeAndClear` returns one value once | `Get` is repeatable | `Get` repeats; on miss, fetch from origin |

If a consumer needs to read the same value multiple times, it is a registry, not an inbox.
If the value can be re-fetched from a source of truth on miss, it is a cache, not an inbox.

## Reference implementation

`services/atlas-channel/atlas.com/channel/monster/inbox.go` (`nextSkillInbox`):

- atlas-monsters' picker emits `NEXT_SKILL_DECIDED` events.
- atlas-channel's consumer calls `Put(tenantModel, uniqueId, decision)`.
- atlas-channel's MoveLife handler calls `TakeAndClear(tenantModel, uniqueId)` to inject the decision into the next `MoveMonsterAck` to the controller.
- `MONSTER_DESTROYED` events trigger `Evict` to keep the inbox bounded.

## Implementation checklist

- Singleton via `sync.Once` (mirrors `cooldownRegistry`).
- `sync.RWMutex` over the inner map.
- Tenant-scoped: outer key is `tenant.Id`, inner key is the per-resource id.
- Three methods: `Put`, `TakeAndClear`, `Evict`. No more.
- No persistence — the inbox is per-process and re-hydrates from the next producer cycle on restart.
