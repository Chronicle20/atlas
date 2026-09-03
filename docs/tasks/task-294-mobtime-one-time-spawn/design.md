# Design — One-Time Spawn for `mobTime == -1` Spawn Points

Task: `task-294-mobtime-one-time-spawn`
Inputs: `prd.md` (approved), `evidence.md`
Status: Draft for `/plan-task`

---

## 1. Problem restated in structural terms

`atlas-maps` keeps exactly one Redis structure per field for spawning: a hash of
`storedSpawnPoint` keyed by spawn-point id, seeded once from the *filtered* provider
`GetSpawnableSpawnPoints` (`map/monster/registry.go:205`). Everything downstream —
`Count`, `ReserveEligibleSpawnPoints`, `ResetCooldown` — reads that one hash and assumes
every member is a recurring, cooldown-driven point.

The defect is that one-time points have no home. They are dropped at the provider boundary
(`data/map/monster/processor.go:58`) because the only structure that exists cannot express
them: putting them in the recurring hash would subject them to the cooldown/reserve
mechanism (they would respawn) and would inflate `Count`, which is the input to
`getMonsterMax` (FR-2.5 forbids this).

So the shape of the fix is: **give one-time points their own structure, and give the field
an arming marker**, rather than trying to overload the recurring hash with a
classification flag.

---

## 2. Verified facts this design rests on

Each was read out of the worktree, not recalled.

| Fact | Source |
|---|---|
| `Extract` drops `RestModel.Hide`; `SpawnPoint` has no `Hide` field | `data/map/monster/rest.go`, `data/map/monster/model.go` |
| `GetSpawnableSpawnPoints` is called from exactly one non-test site | `map/monster/registry.go:205` (grep over `services/`) |
| Only `mockDataProcessor` (`map/monster/processor_test.go:743,753`) implements the data `Processor` interface besides `ProcessorImpl` | grep over `services/` |
| `TenantKeyedHash` key format is `<prefix>:<namespace>:<TenantKey(t)>:<keyFn(k)>`, and `TenantKey` starts with the bare tenant UUID | `libs/atlas-redis/keyed_hash.go`, `libs/atlas-redis/keys.go` |
| `TenantKeyedHash.ClearForTenantId` SCANs `<prefix>:<namespace>:<tenantId>:*` — i.e. the whole namespace for that tenant, not one keyFn shape | `libs/atlas-redis/keyed_hash.go` |
| `TenantKeyedHash` already exposes `SetNX` (HSETNX) and `Len` (HLEN) | `libs/atlas-redis/keyed_hash.go` |
| `map.ProcessorImpl.Exit` today does exactly two things: `cp.Exit` then buffer a `CHARACTER_EXIT` on `EnvEventTopicMapStatus`. There is **no** field-empties block — PR #1566 / task-278 has not merged into this branch's `main` | `map/processor.go:105-110`; `git log --oneline main \| grep task-278` → no match |
| `character.ProcessorImpl.Exit` removes from an **in-memory** registry synchronously, so `GetCharactersInMap` immediately after it reflects the removal | `map/character/processor.go:39-61` |
| `atlas-maps` runs `replicas: 1` and gets the whole `atlas-env` ConfigMap via `envFrom` | `deploy/k8s/base/atlas-maps.yaml:7,21-23` |
| `COMMAND_TOPIC_MONSTER` is declared in the shared `atlas-env` ConfigMap | `deploy/k8s/base/env-configmap.yaml:68` |
| `atlas-monsters` already consumes a field-scoped `DESTROY_FIELD` command with an **empty body** on `COMMAND_TOPIC_MONSTER` | `services/atlas-monsters/.../kafka/consumer/monster/kafka.go:26,103,252`; `consumer.go:356-367` |
| Nothing destroys monsters when a field empties — `CHARACTER_EXIT` in `atlas-monsters` only reassigns *control* | `services/atlas-monsters/.../kafka/consumer/map/consumer.go:65-95` |
| `NewRespawn` runs every 10 s over `GetMapsWithCharacters()` | `main.go:130` |

---

## 3. Decisions

### D1 — Three Redis keys per field, one namespace

Split the per-field state into three hashes, all under the existing `maps:spawn`
namespace and all keyed by the existing `character.MapKey`:

| Role | `keyFn` output | Contents |
|---|---|---|
| Recurring | `v2:<w>:<c>:<map>:<inst>` | `storedSpawnPoint` per recurring, non-hidden point — **byte-identical in shape to today** |
| One-time | `v2:onetime:<w>:<c>:<map>:<inst>` | `storedSpawnPoint` per one-time, non-hidden point (static; `nextSpawnAt` unused) |
| Meta | `v2:meta:<w>:<c>:<map>:<inst>` | `seeded` = `1`; `onetimeFired` = fire timestamp (present ⇒ **disarmed**) |

Implemented as three `TenantKeyedHash[character.MapKey]` instances on
`SpawnPointRegistry`, constructed in `InitRegistry`, sharing namespace `"maps:spawn"`.

*Why this and not a flag on the existing hash.* A `oneTime bool` inside `storedSpawnPoint`
would force every existing Lua script to filter (`reserveEligibleScript`,
`resetCooldownScript`), would make `Count` wrong for FR-2.5 (it is `HLEN`, which cannot
filter), and would require a second HGETALL-and-count round trip on every pass just to get
the recurring denominator. Splitting the keys makes the entire recurring path — `Count`,
`ReserveEligibleSpawnPoints`, `ResetCooldown` — **completely untouched code**, which is the
strongest possible answer to the PRD's primary regression risk (§8 Backward compatibility,
and the `104040000` acceptance criterion).

*Why the shared namespace matters.* `ClearForTenantId` globs
`<prefix>:maps:spawn:<tenantId>:*`, which is namespace-wide per tenant and therefore sweeps
all three key shapes. **`FlushTenant` and its `DATA_UPDATED` handler need no change at
all** — FR-3.5 is satisfied by construction. Same for `Reset` → `ClearAllAcrossTenants`.

### D2 — `v2:` key-schema token, resolving open question 4

Every `keyFn` gains a leading `v2:` segment.

The migration hazard is real and not the one the PRD guessed. `storedSpawnPoint`'s JSON
shape does **not** change (see D4), so decoding is safe. The hazard is *content*: a field
already seeded by the old code holds only the recurring subset, and `InitializeForMap`'s
"already present ⇒ skip" guard means it would never be re-seeded. For the 61 mixed maps the
one-time points would never appear. (The 991 all-one-time maps are unaffected — their old
hash is empty, so no key exists.)

Changing the key shape makes every field re-seed exactly once after deploy. Old `v1`-shaped
keys become unreachable; they are inert (no reader), bounded (one hash per field per tenant
ever visited), and are reaped by the next `DATA_UPDATED` flush, whose SCAN pattern is
namespace-wide and matches them. No manual Redis cleanup is required.

*Alternative considered and rejected:* a `seedVersion` field in the meta hash plus a
delete-and-reseed branch inside `InitializeForMap`. It avoids the orphan keys but adds a
Go-side read-then-act across two keys, a reseed race between the enter path and the 10 s
task, and an extra round trip on the hot path — all to avoid a one-shot, self-reaping,
bounded key set. Not worth it.

*Alternative rejected:* bumping the namespace to `maps:spawn2`. That moves the orphans
*outside* `FlushTenant`'s SCAN pattern, turning a self-reaping remnant into a permanent
leak.

### D3 — Classify once, in the data package, off a single HTTP drain

`data/map/monster` gains a pure partition function and drops the `Spawnable` predicate:

```go
// Classified is the three-way partition of a map's monster life entries.
type Classified struct {
    Recurring []SpawnPoint // MobTime >= 0, not hidden
    OneTime   []SpawnPoint // MobTime <  0, not hidden
    Hidden    []SpawnPoint // Hide == true, either MobTime
}

func Classify(points []SpawnPoint) Classified
```

`Processor` interface after the change:

```go
type Processor interface {
    SpawnPointProvider(mapId _map.Id) model.Provider[[]SpawnPoint]
    GetSpawnPoints(mapId _map.Id) ([]SpawnPoint, error)
}
```

`SpawnableSpawnPointProvider`, `GetSpawnableSpawnPoints`, and `Spawnable` are **deleted**.
They have exactly one non-test consumer, which this task rewrites; leaving a predicate named
`Spawnable` that no longer means "spawnable" is a live trap for the next reader.
`mockDataProcessor` in `map/monster/processor_test.go` shrinks to match.

*Why a pure function over more filtered providers.* Two providers would mean two
`DrainProvider` runs and two paginated HTTP fetches of the same endpoint per field
initialization — the endpoint that already shows up in the evidence log. One `GetSpawnPoints`
call plus an in-memory partition is one fetch. `Classify` is also directly unit-testable
without a fake HTTP server, which is where the FR-1.2/FR-1.3/FR-1.4 boundary tests belong.

`MobTime < 0` — not `== -1` — per FR-1.2.

### D4 — `Hide` is carried into the model but excluded at seed time; `storedSpawnPoint` is unchanged

`SpawnPoint` gains `Hide bool`; `Extract` carries `rm.Hide` into it (FR-1.1).

`storedSpawnPoint` deliberately does **not** gain a `Hide` field. Hidden points never enter
either hash (FR-1.4), so "hidden" is not a state the registry can be in; storing the flag
would only create a second, never-read representation. `fromStored` therefore yields
`Hide == false` by construction, which is accurate for every point the registry holds. This
is what keeps the Redis JSON shape stable across the rolling deploy.

Consequence, accepted per FR-1.4 and open question 2: `600020300` (Wolf Spider Cavern,
`9400545`) and `800020130` (Encounter with the Buddha, `9400013`) each lose their single
spawn point and become empty of static monsters. Both are one-point maps in the sweep; both
currently spawn. This is a deliberate, PRD-sanctioned content change, and it must appear in
the PR body's behavior-change section, not just in a test.

### D5 — Fire via one atomic claim-and-fetch Lua script

```lua
-- KEYS[1] = meta hash, KEYS[2] = one-time hash
-- ARGV[1] = nowMilli
-- Returns: {} when the field is disarmed or has no one-time points,
--          otherwise HGETALL of the one-time hash.
if redis.call('HSETNX', KEYS[1], 'onetimeFired', ARGV[1]) == 0 then
    return {}
end
local entries = redis.call('HGETALL', KEYS[2])
if #entries == 0 then
    return {}
end
return entries
```

`HSETNX` is the claim: it is a single atomic Redis write, strictly stronger than the
read-then-write FR-2.3 forbids, and folding the payload fetch into the same script makes
both the firing path and the disarmed path exactly **one round trip** (the §8 performance
requirement).

Exposed as `ClaimOneTimeSpawnPoints(ctx, mapKey) ([]*CooldownSpawnPoint, error)`. Every
returned point is spawned; there is no cap, no `getMonsterMax`, no character-count scaling
(FR-2.2).

Note the meta hash is claimed even when the one-time hash is empty. That is intentional and
harmless: it costs one wasted `HSETNX` on the first pass for a recurring-only field and
keeps the script single-branch. The field is then "disarmed with nothing to fire", which is
indistinguishable from "fired" and equally correct.

*Crash window.* If the pod dies between the script returning and `CreateMonster` being
issued, the batch is lost until the field re-arms. This is inherent to any
claim-then-act split and is not made worse by any alternative considered; `CreateMonster` is
already fire-and-forget on the recurring path (`routine.Go`, no error propagation). Not
mitigated; noted.

### D6 — Re-arm in `map.ProcessorImpl.Exit`, `HDEL` of one field

```go
func (p *ProcessorImpl) Exit(mb *message.Buffer) func(...) error {
    return func(transactionId uuid.UUID, f field.Model, characterId uint32) error {
        p.cp.Exit(transactionId, f, characterId)
        if remaining, err := p.cp.GetCharactersInMap(transactionId, f); err == nil && len(remaining) == 0 {
            // field just emptied — re-arm one-time spawns (task-294)
        }
        return mb.Put(mapKafka.EnvEventTopicMapStatus, exitMapProvider(transactionId, f, characterId))
    }
}
```

`cp.Exit` mutates the in-memory registry synchronously, so the `GetCharactersInMap`
immediately after it observes the departure — no ordering hazard.

Re-arm is `HDEL meta onetimeFired`, wrapped as:

```go
func (r *SpawnPointRegistry) RearmOneTime(ctx context.Context, mapKey character.MapKey) (bool, error)
```

returning `true` iff the `HDEL` actually removed the marker — i.e. iff the field had fired
and is now armed again. One round trip, one key, no Lua needed (`HDEL` is atomic).

This satisfies FR-3.3 by construction: the recurring hash and the one-time hash are
different keys and are not touched. It satisfies FR-3.4 because `keyFn` already embeds
world, channel, map **and instance** — which also resolves **open question 3**: instanced
fields key independently today and continue to.

`Exit` is the single funnel for logout, `TransitionMap`, and `TransitionChannel`
(`map/processor.go:120,135`), so all three paths are covered by one block. Both transition
paths call `Exit(oldField)` then `Enter(newField)` on distinct field keys, so re-arming the
old field cannot race the new field's spawn pass.

**Rebase note (FR-3.2).** PR #1566 / task-278 is confirmed *not* in this branch's history,
so this task **introduces** the field-empties block. If #1566 lands first, the rebase must
merge into a single `len(remaining) == 0` block with both bodies — never two adjacent
emptiness checks.

### D7 — Re-arm despawns the field's residual monsters, via the existing `DESTROY_FIELD` command

**This resolves open question 1, and it resolves it as in-scope.**

Verified: nothing despawns monsters when a field empties. Without this, the sequence "party
kills 6 of 10 Lucidas, leaves, next party enters" yields 4 survivors + 10 fresh = 14 in a
10-point field, and the PRD's own acceptance criterion ("leave, re-enter, confirm 10 again")
fails on the very map the task exists to fix. Leaving that as a follow-up would ship the
feature broken on its second use.

It is also producible now without violating the PRD's non-goal. `atlas-monsters` already
consumes `CommandTypeDestroyField = "DESTROY_FIELD"` on `COMMAND_TOPIC_MONSTER`, with a
field-scoped envelope and an **empty body**, and calls `DestroyInField`. `atlas-maps`
already has `envFrom: atlas-env`, which carries `COMMAND_TOPIC_MONSTER`. So the entire cost
is a producer in `atlas-maps`:

- `kafka/message/monster/kafka.go` — add `EnvCommandTopic topic.Token = "COMMAND_TOPIC_MONSTER"`,
  `CommandTypeDestroyField = "DESTROY_FIELD"`, a `FieldCommand[E]` envelope matching
  `atlas-monsters`' `fieldCommand` field-for-field, and `DestroyFieldBody struct{}`.
- `monster/producer.go` in `atlas-maps` — `DestroyFieldCommandProvider(f field.Model)`.
- `map/processor.go` `Exit` — `mb.Put(monsterKafka.EnvCommandTopic, ...)` on the same buffer
  as the existing `CHARACTER_EXIT` put.

**Zero change to `atlas-monsters`, `atlas-channel`, or `atlas-data`** — the PRD non-goal is
respected literally and in spirit. The envelope is a cross-service contract copy; the plan
must include a test asserting the emitted JSON matches the consumer's `fieldCommand` shape
(`worldId`, `channelId`, `mapId`, `instance`, `type`, `body`), because a silent field-name
drift here fails open with no error anywhere.

**Scoping — this is the backward-compatibility hinge.** The destroy is emitted **only when
`RearmOneTime` returns `true`**, i.e. only for a field that actually fired a one-time batch.
The 4,207 unaffected maps and every recurring-only map never fire, so they never emit, and
their emptying behavior is bit-identical to `main`. This is why re-arm returns a bool rather
than being a fire-and-forget delete.

*Alternative rejected:* despawn unconditionally on every field empty. Cleaner conceptually
(a vacated field is a fresh field), but it changes behavior for all 5,261 maps to fix a
defect on 1,052 of them, and would silently kill a boss that a player left the map to
re-buff for. Out of proportion.

*Alternative rejected:* despawn only the one-time templates. Not expressible — the
`DESTROY_BY_SOURCE` command matches on a provenance pair that `atlas-maps` does not set when
it calls `CreateMonster`, and setting it would require an `atlas-monsters`-side contract
change. `DESTROY_FIELD` on an empty field is the proportionate tool.

### D8 — Recurring target uses the recurring hash only; one-time monsters still count against the deficit

`registry.Count` becomes `HLEN` of the **recurring** hash. `getMonsterMax(c, recurringCount)`
is therefore `ceil((0.70 + 0.05*min(6,chars)) * recurringCount)` exactly as the acceptance
criterion demands, with one-time points excluded from the denominator (FR-2.5).

**Accepted consequence, stated plainly:** `monstersInMap` (`mp.CountInMap`) counts *every*
monster in the field, including the one-time ones. On a mixed map, once the one-time batch
is alive it will usually exceed `monstersMax`, so `toSpawn <= 0` and recurring spawns are
suppressed until enough one-time monsters are killed.

- This is not a regression: no mixed map spawns anything from its one-time set today.
- It matches the reference implementation, where the respawn deficit is computed against the
  map-wide monster count and one-time mobs are part of it.
- The acceptance criterion "a mixed map fires its one-time batch **and** continues to spawn
  its recurring points" still passes: on the first armed pass the claim fires before the
  deficit is computed, `CreateMonster` is asynchronous, and `monstersInMap` is still ~0 at
  that instant — so both sets spawn on that pass.
- Attributing the count per-origin is not possible without an `atlas-monsters` contract
  change (the same provenance gap as in D7), which is a PRD non-goal.

Documented as a comment at the `monstersInMap` site so the next reader does not "fix" it.

### D9 — Observability (FR-4)

- FR-4.1: after the claim, if `recurringCount == 0`, `HLEN` the one-time hash; if that is
  also `0`, `p.l.Debugf("Field [%s] has no spawn points: one-time [0] recurring [0].", ...)`
  and return `nil`. The extra round trip is confined to the zero-recurring branch. Checking
  both counts is what stops the log from lying on an already-disarmed one-time field.
- FR-4.2: on a non-empty claim, debug with field id and point count, before the
  `CreateMonster` loop.
- FR-4.3: on `RearmOneTime` returning true, debug with the field id.
- The `if totalCount == 0 { return nil }` early return is **removed** (FR-2.6); the claim now
  runs before any count is consulted.

### D10 — No change to `FlushTenant` or the `DATA_UPDATED` handler

Consequence of D1's shared namespace. `ClearForTenantId`'s pattern is
`<prefix>:maps:spawn:<tenantId>:*`, which already matches the one-time and meta keys.
FR-3.5 is satisfied without touching `kafka/consumer/data/handler.go`. The plan must still
carry a *test* for FR-3.5 (flush → previously-disarmed field fires again), because "satisfied
by construction" is a claim about the key pattern that only a test can pin.

---

## 4. `SpawnMonsters` control flow after the change

```
InitializeForMap(mapKey)            -- HEXISTS meta 'seeded'; on miss: 1 HTTP drain + 1 Lua seed
characters := GetCharactersInMap()
if len(characters) == 0 -> return nil                       (unchanged)

fired := ClaimOneTimeSpawnPoints(mapKey)                    -- 1 Lua round trip  [FR-2.1..2.4]
if len(fired) > 0:
    debug "firing N one-time points"                        [FR-4.2]
    for each: routine.Go -> CreateMonster                   -- no cap            [FR-2.2]

recurringCount := Count(mapKey)                             -- HLEN recurring    [FR-2.5]
if recurringCount == 0:
    if CountOneTime(mapKey) == 0: debug "no spawn points"   [FR-4.1]
    return nil
                                                            -- from here: unchanged code
monstersInMap := mp.CountInMap(...)
monstersMax   := getMonsterMax(len(characters), recurringCount)
toSpawn       := monstersMax - monstersInMap
if toSpawn <= 0 -> return nil
reserved := ReserveEligibleSpawnPoints(mapKey, toSpawn, defaultSpawnCooldown, seed)
...
```

`InitializeForMap` seeding script (KEYS: recurring, one-time, meta):

```lua
if redis.call('HEXISTS', KEYS[3], 'seeded') == 1 then return 0 end
local nRec = tonumber(ARGV[1])
local i = 2
for k = 1, nRec do redis.call('HSET', KEYS[1], ARGV[i], ARGV[i+1]); i = i + 2 end
while i <= #ARGV do redis.call('HSET', KEYS[2], ARGV[i], ARGV[i+1]); i = i + 2 end
redis.call('HSET', KEYS[3], 'seeded', '1')
return 1
```

Idempotent, atomic across all three keys (single-node `goredis.Client`, not a cluster
client — multi-key Lua is safe). A field with zero points of any kind still gets
`seeded`, so it costs one `HEXISTS` per pass thereafter instead of an HTTP drain — a
side benefit for the 991 currently-refetching maps visible in the evidence log.

---

## 5. Concurrency model

`atlas-maps` is `replicas: 1` and its character registry is in-memory, so the multi-pod
scenario the PRD describes occurs only during a rolling-restart overlap. The atomic claim is
still load-bearing in steady state: within one pod, the character-enter path and the 10 s
`NewRespawn` task run on independent goroutines against the same field and can enter
`SpawnMonsters` concurrently. `HSETNX` makes exactly one of them win.

`RearmOneTime` (`HDEL`) racing a concurrent `ClaimOneTimeSpawnPoints` (`HSETNX`) is
serialized by Redis. The two orderings are: claim-then-rearm (batch fires into a field the
last player just left; the monsters are then destroyed by D7's command, and the field is
armed — correct), or rearm-then-claim (the field is armed and the entering player's pass
fires it — correct). Neither produces a double batch.

---

## 6. Test strategy

Existing `map/monster/processor_test.go` and `registry_test.go` run against a real Redis
(they call `GetRegistry()`), so the Lua paths are exercised for real, not faked.

New coverage, keyed to the acceptance criteria:

| Test | Criterion |
|---|---|
| `Classify` table test: `mobTime` `-1`/`-2`/`0`/`30`, `hide` true/false | FR-1.2, FR-1.3, FR-1.4 |
| `Extract` carries `Hide` | FR-1.1 |
| Seed: 10 one-time points ⇒ recurring `HLEN` 0, one-time `HLEN` 10 | `920010920` shape |
| Fire: armed field returns all 10, ignoring character count | FR-2.2 |
| Second `SpawnMonsters` on the same field returns 0 points | FR-2.4, "second character does not re-fire" |
| Two concurrent goroutines calling `ClaimOneTimeSpawnPoints` — exactly one gets a non-empty batch | FR-2.3, concurrency criterion |
| Hidden point is absent from both hashes | `600020300` / `800020130` |
| Mixed map: recurring `HLEN` excludes one-time; `getMonsterMax` over the recurring count | FR-2.5 |
| Regression: recurring-only map seeds, reserves, and cools down identically | `104040000` |
| `RearmOneTime` returns true then false; a re-armed field fires a fresh full batch | FR-3.1, "leave and re-enter" |
| `RearmOneTime` leaves the recurring hash and its `nextSpawnAt` values untouched | FR-3.3 |
| Re-arm on channel 0 does not re-arm channel 1; instanced fields are independent | FR-3.4, open question 3 |
| `FlushTenant` clears meta; a disarmed field fires again after the flush | FR-3.5 |
| `Exit` of the last character re-arms **and** buffers one `DESTROY_FIELD`; `Exit` with characters remaining does neither; `Exit` of a never-fired field emits no `DESTROY_FIELD` | FR-3.2, D7 scoping |
| Emitted `DESTROY_FIELD` JSON matches `atlas-monsters`' `fieldCommand` envelope key-for-key | D7 cross-service seam |
| Zero-spawn-point field logs both counts | FR-4.1 |

The cross-service seam test is not optional: `CLAUDE.md`'s review gate calls out exactly this
class — a change crossing a service boundary needs a test asserting the new contract, because
`verify.sh` cannot see it.

---

## 7. Files touched

| File | Change |
|---|---|
| `data/map/monster/model.go` | `SpawnPoint.Hide` |
| `data/map/monster/rest.go` | `Extract` carries `Hide` |
| `data/map/monster/processor.go` | delete `Spawnable`/`SpawnableSpawnPointProvider`/`GetSpawnableSpawnPoints`; add `Classified` + `Classify` |
| `map/monster/registry.go` | three `TenantKeyedHash` instances with `v2:` keyFns; new seed script; `ClaimOneTimeSpawnPoints`; `RearmOneTime`; `CountOneTime`; `Count` → recurring hash |
| `map/monster/processor.go` | claim-and-fire before the count; recurring count from the recurring hash; remove the silent `totalCount == 0` return; FR-4 logs |
| `map/processor.go` | `Exit` field-empties block: re-arm + conditional `DESTROY_FIELD` |
| `kafka/message/monster/kafka.go` (maps) | `EnvCommandTopic`, `CommandTypeDestroyField`, `FieldCommand[E]`, `DestroyFieldBody` |
| `monster/producer.go` (maps, new) | `DestroyFieldCommandProvider` |
| `map/monster/processor_test.go` | `mockDataProcessor` shrinks to the two-method interface |

No migration, no schema change, no manifest change, no `atlas-data` / `atlas-monsters` /
`atlas-channel` change.

---

## 8. Risks

| Risk | Mitigation |
|---|---|
| Recurring regression on 4,207 maps | The recurring hash, `Count`, `ReserveEligibleSpawnPoints`, and `ResetCooldown` are structurally untouched (D1); pinned by the `104040000` regression test |
| `DESTROY_FIELD` envelope drift between the two services | Field-for-field JSON assertion test (D7); the consumer's body is `struct{}`, so only the envelope can drift |
| `DESTROY_FIELD` fires too broadly | Gated on `RearmOneTime() == true` — only fields that actually fired a batch (D7) |
| Orphaned `v1` keys after deploy | Bounded, inert, reaped by the next `DATA_UPDATED` flush; documented in the PR body (D2) |
| Recurring suppression on the 61 mixed maps | Accepted and documented in code (D8); matches reference behavior; not a regression from `main` |
| `600020300` / `800020130` lose their monster | PRD-sanctioned (FR-1.4); must be named in the PR body as a behavior change |
| Rebase collision with PR #1566 in `Exit` | Confirmed not yet merged; reconcile into one emptiness block, never two (D6) |

---

## 9. Open questions from the PRD — dispositions

1. **Residual monsters on re-arm** — Resolved as **in scope**: re-arm emits `DESTROY_FIELD`,
   scoped to fields that actually fired. Producible with an `atlas-maps`-only change; without
   it the PRD's own re-entry acceptance criterion fails (D7).
2. **`hide` semantics** — FR-1.4 as written: excluded from the registry entirely, no
   `storedSpawnPoint` field. Two maps lose their only spawn point; surfaced in the PR body
   rather than buried (D4).
3. **Instance keying** — Already correct: `keyFn` embeds `Field.Instance()`, and all three
   new keys inherit it (D6).
4. **Redis shape migration** — The JSON shape does not change; the *content* would have gone
   stale. Handled by the `v2:` key-schema token (D2).

---

## 10. Not decided here

Nothing is left for the planner to decide about architecture. Two items are deliberately
left to implementation judgement: the exact wording of the FR-4 debug lines, and whether
`CountOneTime` is its own method or an inlined `HLEN` at the one call site.
