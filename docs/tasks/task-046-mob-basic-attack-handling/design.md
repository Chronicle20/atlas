# Design — Mob Basic Attack Handling (Magic / Ranged)

## Status

Approved by user; ready for `/plan-task`.

## Context

See `prd.md` for the full investigation and bug write-up. The short version:

- v83 magic/ranged-attacking mobs (e.g. Samiho `5100004`) fire their basic attack once after aggro, then never attack again. Melee mobs work fine. Named skills (picker pipeline) work fine.
- Verified cause: atlas does not implement Cosmic's basic-attack pipeline (`Monster.canUseAttack` + `usedAttack` at `~/source/Cosmic/src/main/java/server/life/Monster.java:1530-1576`). atlas-monsters never decrements mob MP for basic attacks, so the v83 client's mob state machine — which expects post-attack MP in the move ack — locks the mob into "I just attacked" mode indefinitely.
- atlas-data also doesn't expose `attack{1,2,3}/info` metadata (`mpCon`, `attackAfter`) needed to do the decrement correctly.

## Architectural choices

Two architectural decisions were made during brainstorming and locked in before this design:

1. **Dispatch shape: B (atlas-channel optimistic ack + atlas-monsters async authoritative decrement).**
   - Rejected A (sync REST hop on every move packet) — too much hop traffic for a high-rate packet.
   - Rejected C (atlas-channel-local authoritative MP) — splits ownership of mob state.
   - B mirrors the existing skill picker's inbox pattern: atlas-channel optimistically projects the post-decrement MP into the ack using cached attack metadata, then async-emits a Kafka command. atlas-monsters remains the sole writer of mob MP.

2. **Cooldown registry: C (separate `MonsterAttackCooldown` registry).**
   - Rejected A (no server-side cooldown) — leaves the door open for client misbehavior or future anti-cheat.
   - Rejected B (reuse skill cooldown registry) — conflates skill-id keys (100-200) with attack-position keys (0-2). Less clean semantically.
   - C gives a clean `(uniqueId, attackPos)` keying with independent eviction, mirroring the existing skill cooldown registry's shape.

Per-design conventions and trade-offs:

- atlas-monsters is the **only writer** of mob MP. atlas-channel's optimistic ack is *forecast*, not authority.
- atlas-channel never gates: it always acks with `max(0, mp - mpCon)`. atlas-monsters silently rejects on cooldown / insufficient MP — the ack already shipped, the client respects its own local cooldown anyway.
- The race window between atlas-channel's optimistic computation and atlas-monsters' authoritative decrement is bounded by Kafka in-cluster latency (~ms). It self-corrects on the next move packet's ack.

## Data flow

```
v83 client                         atlas-channel                       atlas-monsters                       atlas-data
   │                                    │                                    │                                    │
   │ MOB_MOVE (nActionAndDir=basic-atk) │                                    │                                    │
   ├───────────────────────────────────►│                                    │                                    │
   │                                    │ GET /api/data/monsters/{id}        │                                    │
   │                                    │ ── (cached, attacks[]) ────────────┼───────────────────────────────────►│
   │                                    │ ◄── attacks: [{pos,conMP,after}]   │                                    │
   │                                    │                                    │                                    │
   │                                    │ GET /api/monsters/{uid}            │                                    │
   │                                    │ ── (current MP) ──────────────────►│                                    │
   │                                    │ ◄── mp                             │                                    │
   │                                    │                                    │                                    │
   │ MoveMonsterAck (mp = mp - mpCon)   │                                    │                                    │
   │ ◄──────────────────────────────────┤                                    │                                    │
   │                                    │ USE_BASIC_ATTACK kafka cmd         │                                    │
   │                                    │ ──────────────────────────────────►│                                    │
   │                                    │                                    │ cooldown gate (MonsterAttack-      │
   │                                    │                                    │   Cooldown registry, attackPos)    │
   │                                    │                                    │ MP gate (m.Mp() >= mpCon)          │
   │                                    │                                    │ DeductMp(mpCon)                    │
   │                                    │                                    │ RegisterCooldown(now+attackAfter)  │
```

## Service-by-service changes

### atlas-data

**`services/atlas-data/atlas.com/data/monster/reader.go`**

Add a sibling parser to `getAnimationTimes` that walks `<imgdir name="attackN">` (N ∈ {1, 2, 3}) and reads each one's `info` subdirectory. Skip silently if the directory is absent.

For each present `attackN/info`, extract:

| Field | Source | Default if missing |
|---|---|---|
| `pos` | derived from N (1, 2, or 3) | required |
| `conMP` | `info/conMP` (int) | `0` |
| `attackAfter` | `info/attackAfter` (int, ms) | `0` |

**`services/atlas-data/atlas.com/data/monster/rest.go`**

Add to `RestModel`:

```go
type AttackInfo struct {
    Pos         uint8 `json:"pos"`         // 1, 2, or 3 — matches WZ attackN naming
    ConMP       int32 `json:"conMP"`
    AttackAfter int32 `json:"attackAfter"` // ms cooldown
}

type RestModel struct {
    // ...existing fields...
    Attacks []AttackInfo `json:"attacks"`
}
```

JSON wire example for Samiho:

```json
{
  "data": {
    "type": "monsters",
    "id": "5100004",
    "attributes": {
      "name": "Samiho",
      ...,
      "attacks": [
        {"pos": 1, "conMP": 0, "attackAfter": 0},
        {"pos": 2, "conMP": 5, "attackAfter": 1500}
      ]
    }
  }
}
```

`pos` is 1-indexed to match WZ directory naming. The 0-indexed `attackPos` used elsewhere derives from `(rawActionAndDir - 24) / 2` and is converted at the use site (atlas-monsters' `findAttackByPos` does the +1 reconciliation, or stores 0-indexed internally — implementation detail for the plan).

**Backwards compatibility:** the JSON addition is a superset. Empty `attacks: []` is the default for mobs with no attack data. atlas-monsters and atlas-channel both unmarshal mob templates via `json` struct tags and tolerate unknown fields. No consumer breaks.

**Tests:**

- `monster/reader_test.go` — fixture-based parse for Samiho (`attack1` no info, `attack2/info/conMP=5`) and Beetle (`attack1` only, no `info`).
- `monster/rest_test.go` — round-trip the `attacks` array through JSON.

**Performance:** parsing happens once at template load and is cached in atlas-data's existing storage layer. No per-request overhead. No startup regression expected.

### atlas-monsters

**New cooldown registry** — `services/atlas-monsters/atlas.com/monsters/monster/attack_cooldown.go`

Mirrors the existing skill cooldown registry pattern at `monster/cooldown.go`. Sketch:

```go
type AttackCooldownRegistry struct {
    client *goredis.Client
    // ... mirror Cooldown registry shape ...
}

// Key:   tenant + uniqueId + attackPos
// Value: nowMs + attackAfter (when the cooldown expires)

func (r *AttackCooldownRegistry) IsOnCooldown(ctx, t, uniqueId uint32, attackPos uint8) bool
func (r *AttackCooldownRegistry) Register(ctx, t, uniqueId uint32, attackPos uint8, expiresAtMs int64)
func (r *AttackCooldownRegistry) Clear(ctx, t, uniqueId uint32) // on monster destroy
```

Singleton via `sync.Once`. Sweep task runs alongside the existing skill cooldown sweep — drops entries with `expiresAt < now`. `Clear(uniqueId)` is invoked on monster destroy / map exit so a respawned mob with the same uniqueId starts fresh.

**Information model extension** — `services/atlas-monsters/atlas.com/monsters/monster/information/`

Add `Attacks []AttackInfo` to the `Model`, plumbed through Extract / RestModel mirror just like existing `Skills()` / `Resistances()`.

**New processor method** — `services/atlas-monsters/atlas.com/monsters/monster/processor.go`

```go
func (p *ProcessorImpl) UseBasicAttack(uniqueId uint32, attackPos uint8) {
    m, err := GetMonsterRegistry().GetMonster(p.t, uniqueId)
    if err != nil { return }
    if !m.Alive() { return }

    info, err := information.GetById(p.l)(p.ctx)(m.MonsterId())
    if err != nil {
        p.l.WithError(err).Debugf("UseBasicAttack: can't fetch template for monster [%d]", uniqueId)
        return
    }
    atk, ok := findAttack(info.Attacks(), attackPos)
    if !ok {
        p.l.Debugf("UseBasicAttack: monster [%d] has no attack pos %d", uniqueId, attackPos)
        return
    }

    // Cooldown gate.
    if GetAttackCooldownRegistry().IsOnCooldown(p.ctx, p.t, uniqueId, attackPos) {
        p.l.Debugf("UseBasicAttack: monster [%d] attack pos %d on cooldown", uniqueId, attackPos)
        return
    }

    // MP gate.
    if atk.ConMP() > 0 && m.Mp() < uint16(atk.ConMP()) {
        p.l.Debugf("UseBasicAttack: monster [%d] insufficient MP for pos %d", uniqueId, attackPos)
        return
    }

    if atk.ConMP() > 0 {
        if _, err := GetMonsterRegistry().DeductMp(p.t, uniqueId, uint16(atk.ConMP())); err != nil {
            p.l.WithError(err).Errorf("UseBasicAttack: DeductMp failed for monster [%d]", uniqueId)
            return
        }
    }

    if atk.AttackAfter() > 0 {
        expiresAtMs := time.Now().UnixMilli() + int64(atk.AttackAfter())
        GetAttackCooldownRegistry().Register(p.ctx, p.t, uniqueId, attackPos, expiresAtMs)
    }
}
```

**Silent on failure**: every reject path (no attack info, on cooldown, insufficient MP, mob dead/gone) just returns. The atlas-channel ack already shipped — there's nothing to communicate back. Debug-level logs only.

**New Kafka consumer** — `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/consumer.go`

Mirror `handleUseSkillCommand` (line 127). Add:

```go
const CommandTypeUseBasicAttack = "USE_BASIC_ATTACK"

type useBasicAttackCommandBody struct {
    AttackPos uint8 `json:"attackPos"`
}

func handleUseBasicAttackCommand(l, ctx, c command[useBasicAttackCommandBody]) {
    if c.Type != CommandTypeUseBasicAttack { return }
    monster.NewProcessor(l, ctx).UseBasicAttack(c.MonsterId, c.Body.AttackPos)
}
```

Same `EnvCommandTopic` as `USE_SKILL` — no new topic.

**Tests:**

- `attack_cooldown_test.go` — register / `IsOnCooldown` / `Clear` / sweep, parity with existing `cooldown_test.go`.
- `processor_test.go` additions:
  - `TestUseBasicAttack_OnCooldown_Skips`
  - `TestUseBasicAttack_InsufficientMp_Skips`
  - `TestUseBasicAttack_HappyPath_DeductsMpAndRegistersCooldown`
  - `TestUseBasicAttack_NoAttackInfo_Skips`
  - `TestUseBasicAttack_DeadMonster_Skips`
  - `TestUseBasicAttack_ZeroConMpAndZeroAttackAfter_NoOp` (melee parity)
- Kafka consumer test mirroring existing `kafka_test.go` `USE_SKILL` coverage.

### atlas-channel

**Action classification helper** — new file `services/atlas-channel/atlas.com/channel/movement/action.go`

```go
// Cosmic-equivalent classification of MoveLife.nActionAndDir:
//   [24, 41] → basic attack (attackPos = (raw - 24) / 2)
//   [42, 59] → named skill (handled by existing inbox path)
const (
    basicAttackRangeLo int8 = 24
    basicAttackRangeHi int8 = 41
)

func basicAttackPos(rawActionAndDir int8) (uint8, bool) {
    if rawActionAndDir < basicAttackRangeLo || rawActionAndDir > basicAttackRangeHi {
        return 0, false
    }
    return uint8((rawActionAndDir - basicAttackRangeLo) / 2), true
}
```

These are wire-protocol classification constants; they live in atlas-channel. If atlas-monsters or another service ever needs the same classification, promote to `libs/atlas-constants/monster/`.

**`ForMonster` extension** — `services/atlas-channel/atlas.com/channel/movement/processor.go:109`

Insert a basic-attack branch alongside the existing skill-id branch (line 153). Refactored shape:

```go
func (p *Processor) ForMonster(...) error {
    mo, err := monster.NewProcessor(p.l, p.ctx).GetById(objectId)
    if err != nil { /* existing */ }
    // ...field mismatch check...

    // Compute the ack MP. For basic attacks, optimistically subtract mpCon.
    ackMp := uint16(mo.Mp())
    var basicAttack *basicAttackContext
    if pos, ok := basicAttackPos(skill); ok {
        info, ierr := information.NewProcessor(p.l, p.ctx).GetById(mo.MonsterId())
        if ierr == nil {
            if atk, found := findAttackByPos(info.Attacks(), pos); found && atk.ConMP > 0 {
                if uint16(atk.ConMP) > ackMp {
                    ackMp = 0
                } else {
                    ackMp -= uint16(atk.ConMP)
                }
            }
            basicAttack = &basicAttackContext{Pos: pos}
        }
    }

    // Existing inbox path for predicted skills.
    go func() {
        useSkills := false
        var skillIdByte, skillLevelByte byte
        if d, hit := monster.GetNextSkillInbox().TakeAndClear(p.t, objectId); hit && !d.IsSentinel() {
            useSkills = true
            skillIdByte = d.SkillId
            skillLevelByte = d.SkillLevel
        }
        op := session.Announce(...)(monsterpkt.NewMonsterMovementAck(
            objectId, moveId, ackMp, useSkills, skillIdByte, skillLevelByte,
        ).Encode)
        // ...existing ack send...
    }()

    // Existing broadcast + summary fold + skill UseSkill emit — unchanged.

    // NEW: basic-attack execution.
    if basicAttack != nil {
        go func() {
            err := monster.NewProcessor(p.l, p.ctx).UseBasicAttack(f, objectId, basicAttack.Pos)
            if err != nil {
                p.l.WithError(err).Errorf("Unable to issue basic attack for monster [%d]", objectId)
            }
        }()
    }
    return nil
}
```

**Channel-side processor method** — `services/atlas-channel/atlas.com/channel/monster/processor.go`

Mirror existing `UseSkill` (line 60):

```go
func (p *Processor) UseBasicAttack(f field.Model, monsterId uint32, attackPos uint8) error {
    p.l.Debugf("Monster [%d] using basic attack pos [%d].", monsterId, attackPos)
    return producer.ProviderImpl(p.l)(p.ctx)(monster2.EnvCommandTopic)(
        UseBasicAttackCommandProvider(f, monsterId, attackPos),
    )
}
```

`UseBasicAttackCommandProvider` writes the `USE_BASIC_ATTACK` command body to the existing `EnvCommandTopic`.

**Information processor extension** — `services/atlas-channel/atlas.com/channel/monster/information/`

Add `Attacks() []AttackInfo` to the model, mirroring existing `Skills()` / `Resistances()` plumbing.

**Edge cases:**

| Case | Behavior |
|---|---|
| Client sends `nActionAndDir = 25` (attackPos 0, no `attack1` info exists) | `findAttackByPos` returns false → no MP decrement, no Kafka emit. Ack uses unchanged MP. (Functional parity with melee today.) |
| `nActionAndDir = 30` but mob only has `attack1` (pos 1) | Same — `findAttackByPos(pos=4)` returns false. Pass-through. |
| `mpCon > current mp` | atlas-channel clamps `ackMp` to 0. atlas-monsters silently rejects the Kafka command and does not decrement. Slight visual drift; self-corrects on next move packet. |
| atlas-data unreachable when looking up template | atlas-channel skips both the optimistic decrement AND the Kafka emit. Falls back to current behavior (ack with unchanged MP). The bug regresses for that one packet but doesn't cascade. |
| Monster destroyed mid-flight | Existing aliveness check. atlas-monsters' `UseBasicAttack` checks `GetMonster` and returns early on not-found. |

**Tests:**

- `movement/processor_test.go`:
  - `TestForMonster_BasicAttack_DecrementsAckMpAndEmitsKafka`
  - `TestForMonster_BasicAttack_NoAttackInfoForPos_Passthrough`
  - `TestForMonster_BasicAttack_MpClampsAtZero`
  - `TestForMonster_NamedSkill_StillUsesInbox` (regression for the existing skill path)
- `monster/processor_test.go` additions: `TestUseBasicAttack_EmitsKafkaCommand` (mirror existing `TestUseSkill_EmitsKafkaCommand`).

## Risks & open questions for the plan

- **Cooldown registry key shape.** Plan should specify whether keys are `tenant:uniqueId:attackPos` strings, structured Redis hash keys, or in-memory map equivalents. Defer to whatever existing skill cooldown does.
- **Sweep task wiring.** New cooldown registry needs a sweep task. Plan should decide: integrate into the existing skill-cooldown sweep loop, or new task with the same interval. Simpler to add it to the existing loop.
- **Information cache invalidation.** atlas-channel caches mob info today; the cache duration is a known footgun if a tenant updates mob templates. Plan should specify whether `attacks` cache TTL matches existing fields. Default: same TTL.
- **`pos` indexing convention.** REST model is 1-indexed (matches WZ). The `attackPos = (raw - 24) / 2` formula is 0-indexed. Plan must commit to one internal representation and document the conversion at the boundary.

## Success criteria

(Mirror PRD; reproduced for plan-task convenience.)

- Samiho (`5100004`) at Fox Ridge fires its magic attack repeatedly across an encounter (manual gameplay test).
- Melee mobs unchanged (no regression).
- New atlas-monsters tests cover MP gate, cooldown gate, melee passthrough, dead monster, missing attack info.
- New atlas-data tests cover attack-info parsing for at least one magic-attacker (Samiho) and one melee-only (Beetle).
- New atlas-channel tests cover the basic-attack branch and existing-skill-path regression.
- No new "Read a unhandled message with op 0xXX" lines around basic-attack actions.

## References

- Cosmic v83 reference: `~/source/Cosmic/src/main/java/server/life/Monster.java:1467-1576`, `net/server/channel/handlers/MoveLifeHandler.java:80-180`.
- Atlas attack MP gate (skills): `services/atlas-monsters/atlas.com/monsters/monster/processor.go:528-540`.
- Atlas mob-move handler: `services/atlas-channel/atlas.com/channel/movement/processor.go:109-167`.
- Atlas data reader: `services/atlas-data/atlas.com/data/monster/reader.go:174-186`.
- Atlas data REST: `services/atlas-data/atlas.com/data/monster/rest.go:5-43`.
- Recently merged context: PR #365 (mob skill effects on v83 — picker aggro gate, disease BuffGive shape, mist disease duration unit).
