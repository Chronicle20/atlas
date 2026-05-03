# Context — Mob Basic Attack Handling (task-046)

Quick reference for executing agents. The full bug write-up is in `prd.md`; the
locked-in architecture decisions live in `design.md`. This file is the
"where things live, what to mirror" cheat sheet.

## Goal in one sentence

Implement Cosmic's `Monster.canUseAttack` / `usedAttack` flow so v83 magic /
ranged mobs (Samiho, Wraiths, Voodoos, Fire Boars, …) keep attacking after
the first cast — by giving atlas-monsters server-side `attackPos` cooldowns +
MP decrement, atlas-channel an optimistic ack-MP forecast, and atlas-data the
`attack{1,2,3}/info` metadata that drives both.

## Three services touched

| Service | What changes |
|---|---|
| `services/atlas-data` | Parse `attack{1,2,3}/info` from mob WZ, expose as `attacks: []` on the JSON:API mob response. |
| `services/atlas-monsters` | New `AttackCooldown` Redis registry, new `UseBasicAttack` processor method, new `USE_BASIC_ATTACK` Kafka command + handler, `Attacks` field on `information.Model`. |
| `services/atlas-channel` | New `monster/information/` package (proxies atlas-data), new `movement/action.go` classification helper, optimistic `ackMp` forecast in `movement.ForMonster`, new `USE_BASIC_ATTACK` Kafka producer + processor method. |

## Key files (current state — line numbers as of branch HEAD)

### atlas-data
- `services/atlas-data/atlas.com/data/monster/reader.go:174-186` — `getAnimationTimes` skips `info` subdirs; we add a sibling `getAttacks` parser.
- `services/atlas-data/atlas.com/data/monster/rest.go:5-43` — `RestModel`; we add `Attacks []AttackInfo`.
- `services/atlas-data/atlas.com/data/monster/reader.go:32-101` — `Read` builds the `RestModel`; one new line plumbs `getAttacks(exml)` into it.

### atlas-monsters
- `services/atlas-monsters/atlas.com/monsters/monster/cooldown.go` — pattern to mirror for `attack_cooldown.go`. Uses Redis with TTL via `r.client.Set(..., duration)` (no sweep needed; Redis expires keys).
- `services/atlas-monsters/atlas.com/monsters/monster/cooldown_test.go` — pattern to mirror for `attack_cooldown_test.go` (uses `miniredis`).
- `services/atlas-monsters/atlas.com/monsters/monster/processor.go:483-615` — `UseSkill`; mirror its gate-then-deduct-then-register shape for `UseBasicAttack`.
- `services/atlas-monsters/atlas.com/monsters/monster/processor.go:339,446,970` — places where the existing skill-cooldown is cleared (kill, friendly-kill, destroy). The new attack-cooldown registry must be cleared at the same points.
- `services/atlas-monsters/atlas.com/monsters/monster/information/model.go` / `rest.go` / `builder.go` / `processor.go` — `Attacks` plumbing target. Existing `Skills`/`Resistances` are the template to follow.
- `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka.go:10-21` — `EnvCommandTopic` + command-type constants. Add `CommandTypeUseBasicAttack = "USE_BASIC_ATTACK"` and `useBasicAttackCommandBody`.
- `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/consumer.go:127-134` — `handleUseSkillCommand`; mirror for `handleUseBasicAttackCommand` and register in `InitHandlers`.
- `services/atlas-monsters/atlas.com/monsters/main.go:50` — wire `InitAttackCooldownRegistry(rc)` next to the existing `InitCooldownRegistry`.

### atlas-channel
- `services/atlas-channel/atlas.com/channel/movement/processor.go:109-167` — `ForMonster`; insert basic-attack branch.
- `services/atlas-channel/atlas.com/channel/monster/processor.go:60-63` — `UseSkill`; mirror for `UseBasicAttack`.
- `services/atlas-channel/atlas.com/channel/monster/producer.go:33-49` — `UseSkillCommandProvider`; mirror for `UseBasicAttackCommandProvider`.
- `services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka.go:10-17` — command-type constants. Add `CommandTypeUseBasicAttack` and `UseBasicAttackCommandBody`.
- `services/atlas-channel/atlas.com/channel/monster/information/` — **new package**. Mirrors `services/atlas-monsters/atlas.com/monsters/monster/information/` shape (Model + RestModel + Extract + GetById + builder), but proxies atlas-data instead of atlas-monsters (`requests.RootUrl("DATA")` + `data/monsters/{id}` resource path).
- `services/atlas-channel/atlas.com/channel/movement/action.go` — **new file**. Classification helper for `nActionAndDir ∈ [24, 41]`.
- `libs/atlas-packet/monster/clientbound/movement_ack.go` — `MonsterMovementAck` is unchanged; we just feed it a different `mp` value when the move is a basic attack.

### Reference (do not modify)
- `~/source/Cosmic/src/main/java/server/life/Monster.java:1467-1576` — Cosmic's `canUseAttack` / `usedAttack`. Source of the `attackPos = (raw - 24) / 2` formula and the cooldown/MP-decrement semantics.
- `~/source/Cosmic/src/main/java/net/server/channel/handlers/MoveLifeHandler.java:80-180` — Cosmic's classification of `rawActivity`.

## Architectural decisions (locked)

1. **Dispatch shape: B (optimistic ack + async authoritative decrement)**
   atlas-channel computes the post-decrement MP locally from cached
   atlas-data attack metadata and sends the ack synchronously. atlas-monsters
   asynchronously consumes a `USE_BASIC_ATTACK` Kafka command on
   `EnvCommandTopic` (the same topic as `USE_SKILL`) and authoritatively
   updates MP and the cooldown registry. The race window is bounded by Kafka
   in-cluster latency (~ms) and self-corrects on the next move packet.

2. **Cooldown registry: separate `AttackCooldown` registry (option C)**
   Keyed `(tenant, uniqueId, attackPos)`. Independent from skill cooldowns
   (which are keyed `(tenant, uniqueId, skillId)` where `skillId ∈ 100-200`).
   Kept distinct so `attackPos = 0` and `skillId = 0` don't collide.

3. **`pos` indexing convention**
   - Wire / WZ / atlas-data REST: 1-indexed (`pos: 1, 2, 3` matching `attack1`,
     `attack2`, `attack3`).
   - In-process (atlas-monsters / atlas-channel): 0-indexed (`attackPos =
     (rawActionAndDir - 24) / 2`, range 0-2 for the [24, 41] band — but
     practically only 0, 1, 2 for the three attack slots).
   - Conversion happens at the atlas-channel boundary: when calling
     `findAttackByPos(info.Attacks(), pos0)` we look up by adding 1 (so
     `pos0=0` matches `Pos=1`). The Kafka command body carries the 0-indexed
     `attackPos` because that's what the registry key uses.

4. **Silent-on-failure for `UseBasicAttack`**
   Every reject path (no info, on cooldown, insufficient MP, dead/missing
   monster) returns silently with a debug log. The atlas-channel ack already
   shipped to the client; there's nothing to communicate back. This matches
   `UseSkill`'s own silent-rejection style.

## Patterns / gotchas

- **Redis TTL handles cooldown expiry** — no separate sweep task. Look at
  `cooldown.go:55-59`: `r.client.Set(ctx, key, expiryMs, duration)`. The
  `duration` is the third arg to `Set` and Redis expires the key
  automatically. Mirror this exactly for attack cooldowns.
- **`tenant.MustFromContext(ctx)`** in every processor / handler.
- **`monster2.EnvCommandTopic`** — both `USE_SKILL` and the new
  `USE_BASIC_ATTACK` ride this same topic; we add a new command-type
  constant + new body type, not a new topic.
- **JSON:API for atlas-data REST** — `RestModel.GetName()` returns
  `"monsters"`. The `attacks` field is added as a plain JSON field; api2go
  marshals it via the standard `attributes` envelope.
- **Information cache** — atlas-monsters' `information.GetById` is the
  cached gateway. atlas-channel will get its own `information.GetById` that
  hits atlas-data directly. Both should tolerate empty `Attacks []` (mobs
  with no attack-info subnodes).
- **Builder pattern in tests** — the existing
  `information.NewModelBuilder()` only exposes a few fields. We extend it
  with `SetAttacks` for tests that need to inject attack metadata without
  going through atlas-data.
- **`int8` skill arg in `ForMonster`** — `skill int8` is the raw
  `nActionAndDir` byte. The existing skill branch only fires when
  `skillId > 0` (the named skill). Our basic-attack branch fires when
  `skill ∈ [24, 41]`. They are independent; both can be true on the same
  packet (a basic attack with a queued skill prediction), and the existing
  skill-prediction inbox is unrelated to the basic-attack branch.

## Test discipline

- TDD throughout: red → green → refactor. Commit after each green.
- Use `miniredis` in registry tests — same as `cooldown_test.go`.
- Use `httptest`-fixture XML for atlas-data reader tests — same as
  `reader_test.go`.
- No mocking of atlas-data / atlas-monsters HTTP in atlas-channel
  `movement/processor_test.go`; instead, dispatch the basic-attack work
  through a stubbed channel-side `monster.Processor` (mirror how the
  existing skill path is tested if there is a test, or go through a
  stubbed emitter the same way `processor_test.go` in atlas-monsters does).
- After every cross-service change, build all three services
  (`go build ./...` from each service root) and run their tests
  (`go test ./...`).

## Success criteria (mirrored from PRD)

- Samiho (`5100004`) attacks repeatedly across an encounter (manual gameplay
  test).
- `6090003` (melee) and other melee mobs unchanged (no regression).
- atlas-data parses `attack{1,2}/info` for at least Samiho and is empty for
  Beetle.
- atlas-monsters tests cover MP gate, cooldown gate, melee passthrough, dead
  monster, missing attack info.
- atlas-channel tests cover the basic-attack branch and a regression test for
  the existing-skill path.
- `go build ./...` and `go test ./...` clean in all three services.
