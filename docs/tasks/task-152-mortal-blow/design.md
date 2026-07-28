# Mortal Blow (Ranger/Sniper) — Design

Version: v1
Status: Approved for planning
Created: 2026-07-10
PRD: `docs/tasks/task-152-mortal-blow/prd.md`

---

## 1. Summary

The server half of Mortal Blow is a per-monster proc in `processAttack`'s ranged path: when an attack arrives tagged with skill id 3110001/3210001, the channel evaluates the HP threshold (`hp ≤ maxHp × x / 100`) against its monster snapshot and rolls the kill chance (`roll 1–100 ≤ y`), both from the tenant's skill effect at the character's owned level. On success it emits a new `KILL` command to atlas-monsters, which re-checks the boss flag authoritatively and delivers the kill through the existing `Damage` kill flow — so damaged/killed events, EXP credit, drops, cooldown clears, and revives all run unchanged.

Two services change: `atlas-channel` (proc block, `Y()` accessor, comment fix) and `atlas-monsters` (`KILL` command consumer + `Kill` processor method). No REST, no packets, no data model changes.

## 2. Architecture Decisions

### Decision 1 — Kill delivery: dedicated `KILL` command with atlas-monsters-side boss guard

**Chosen:** a new monster command `CommandTypeKill = "KILL"` (body: `CharacterId`, `SkillId`), emitted by the channel on proc success, consumed by atlas-monsters, which re-checks the boss flag before killing.

Alternatives considered:

- **(A) Reuse `DamageCommandProvider` with a max-damage line.** Zero new Kafka surface, closest to Cosmic's `damageMonster(…, Integer.MAX_VALUE, …)`. Rejected: a `DAMAGE` command carries no provenance, so atlas-monsters cannot distinguish a Mortal Blow kill from a legitimate large hit and has no seam to apply a boss guard. FR-4 requires the kill to be impossible against a boss *even if the channel misfires*; that guard must live where the boss flag lives.
- **(B) Channel-side boss lookup (monster information REST) + plain `DAMAGE`.** Rejected for the same reason — the guard would not be authoritative — plus it adds a REST round-trip per proc. The channel's monster snapshot (`monster.Model`) carries no boss flag, and adding one just for this would widen the snapshot for a guard that still couldn't be trusted.
- **(C — chosen) Dedicated `KILL` command.** Follows the `DRAIN_MP` precedent exactly (task-049 MP Eater): the channel pre-screens what it can see, atlas-monsters re-checks the guard it owns. Commands are keyed by monster unique id, so `DAMAGE` (the triggering attack) and `KILL` land on the same partition in emit order — the kill processes after the attack's damage, and if the attack itself already killed the monster, `Kill` finds it gone from the registry and drops silently.

### Decision 2 — Threshold and roll live in the channel

FR-2/FR-3 mandate this, and it is also the right boundary: the skill effect (`se`) is already resolved at the character's owned level by the existing `processAttack` pipeline, and the monster snapshot is one registry-backed `GetById` away. atlas-monsters has no skill-effect access and should not grow one. The alternative — shipping `x`/`y` in the command and letting atlas-monsters evaluate — was rejected because the threshold read is defined against the channel's pre-attack snapshot (FR-2 timing semantics) and splitting the decision across services buys nothing: the only guard that must be authoritative (boss) doesn't depend on `x`/`y`.

**Key simplification vs. MP Eater:** MP Eater must look up the *passive's* effect separately because the attack's skill differs from the passive. Here the attack's skill **is** Mortal Blow, so `se.X()` / `se.Y()` come straight from the already-resolved effect — no extra effect lookup, no owned-skill re-scan (ownership was already enforced upstream: an unowned skill id destroys the session, which is FR-1's forgery guard). The per-proc I/O is exactly one monster snapshot fetch plus one Kafka emit.

### Decision 3 — Hook point: the existing `onDamageApplied` callback

`processDamageInfoEntry` already exposes `onDamageApplied(monsterId)`, invoked once per non-reflected, damage-carrying entry after damage and status apply — precisely the proc timing FR-2 specifies (post-damage-emit in the handler; pre-attack HP in the snapshot because propagation to atlas-monsters is async). The closure in `processAttack` gains one branch alongside MP Eater's:

```go
onDamageApplied: func(monsterId uint32) {
    if ai.AttackType() == packetmodel.AttackTypeMagic && ai.SkillId() > 0 {
        mpEaterTryProc(l, ctx, mp, c, monsterId, s.Field(), s.CharacterId())
    }
    if ai.AttackType() == packetmodel.AttackTypeRanged &&
        (skill3.Id(ai.SkillId()) == skill3.RangerMortalBlowId || skill3.Id(ai.SkillId()) == skill3.SniperMortalBlowId) {
        mortalBlowTryProc(l, mp, se, monsterId, s.Field(), s.CharacterId(), uint32(ai.SkillId()))
    }
}
```

This inherits two correct behaviors for free: status-only entries (no damage lines) never proc, and **reflected hits never proc** — `processDamageInfoEntry` returns before `onDamageApplied` on reflect. Skill ids come from `libs/atlas-constants/skill` (DOM-21; both constants exist).

`// TODO Mortal Blow` (line 421) is deleted.

### Decision 4 — RNG seam: pure decision helpers, roll passed as a parameter

Follow `mpEaterShouldProc(prop, roll)` exactly — no interface, no injectable rand struct:

```go
// mortalBlowEligible: hp ≤ maxHp × x / 100, integer truncating division
// (Cosmic parity). Defensive: false when x ≤ 0 or maxHp == 0.
func mortalBlowEligible(hp, maxHp uint32, x int16) bool

// mortalBlowKillRoll: roll ≤ y for a uniform roll in [1,100].
// Defensive: false when y ≤ 0.
func mortalBlowKillRoll(roll int, y int16) bool
```

Threshold math widens through `uint64` before multiplying (`uint64(maxHp) * uint64(x) / 100`), mirroring `mpEaterAbsorbAmount`. The production site rolls `rand.Intn(100) + 1`. Tests pin boundaries by driving the helpers directly (FR-8's injectability requirement, satisfied the way the file already satisfies it).

### Decision 5 — `mortalBlowTryProc` (channel)

```go
func mortalBlowTryProc(
    l logrus.FieldLogger,
    mp *monster.Processor,
    se effect.Model,
    monsterId uint32,
    f field.Model,
    characterId uint32,
    skillId uint32,
)
```

Flow (every failure logged and swallowed — FR-5):

1. `x, y := se.X(), se.Y()`; return if `x ≤ 0 || y ≤ 0` (silent — malformed/absent tenant data means the passive is inert, matching MP Eater's `Prop() <= 0` skip).
2. `mon, err := mp.GetById(monsterId)` — on error, `Debugf` and return (monster may have despawned; same level as MP Eater's snapshot miss).
3. `if !mortalBlowEligible(mon.Hp(), mon.MaxHp(), x)` return.
4. `roll := rand.Intn(100) + 1`; `Debugf` the threshold pass; `if !mortalBlowKillRoll(roll, y)` return.
5. `Debugf` proc success (characterId, monsterId, skillId, roll — NFR observability shape); `mp.Kill(f, monsterId, characterId, skillId)` — on error, `Errorf` and return.

No channel-side boss check: the snapshot has no boss flag and atlas-monsters is the authoritative guard (this asymmetry vs. DrainMp — where the channel pre-screens `MaxMp == 0` — exists because MaxMp is on the snapshot and boss is not).

### Decision 6 — atlas-monsters `Kill`: boss guard fail-closed, deliver via the existing kill flow

New consumer arm `handleKillCommand` → `ProcessorImpl.Kill(uniqueId, characterId, skillId uint32)`:

1. `GetMonsterRegistry().GetMonster(t, uniqueId)` — missing → `Debugf` and drop (the triggering attack already killed it, or it despawned; unlike `DRAIN_MP` there is no cosmetic refund to synthesize).
2. `!m.Alive()` → drop.
3. Boss check via `information.GetById` behind the existing `testInformationLookup` seam (DrainMp precedent). **Fail-closed:** if the lookup errors, `Errorf` and drop the kill. This deliberately diverges from DrainMp's fail-open — losing a legitimate proc during an atlas-data hiccup is acceptable; killing a boss is not (FR-4: "impossible even if the channel misfires"). `infoModel.Boss()` → drop silently (defense-in-depth; Debugf).
4. Deliver the kill through the shared damage core so the standard kill flow runs: damaged event (final HP-bar packet), killed event, cooldown/drop-timer clears, status-cancel emits, registry removal, revive spawning, and — critically — the damage-summary credit that drives EXP and drops (Cosmic parity: `Integer.MAX_VALUE` damage, the "reduced EXP gain" fix).

**Small refactor required:** `Damage` currently runs `checkReflect` before applying lines. A Mortal Blow kill must not roll a second reflect (the channel already gated the triggering hit on reflect, and a kill "attack" has no attack type). Split `Damage` into `checkReflect` + an unexported `damageCore(m, characterId, damages)` holding everything from the info-fetch down; `Damage` keeps its exact behavior (`checkReflect` then `damageCore`), and `Kill` calls `damageCore` directly with a single damage line.

**Kill line amount:** `math.MaxUint32` for Cosmic parity, *provided* the plan phase verifies `GetMonsterRegistry().ApplyDamage` clamps the recorded/summarized damage to the HP actually removed. If the registry records raw lines into the damage summary (which feeds EXP splits and the damaged event), use `m.Hp()` at read time instead — equally lethal under the same-partition ordering guarantee, and it keeps the summary honest. This is a verify-then-pick note for the plan, not an open design question: prefer `MaxUint32`, fall back to `m.Hp()` on evidence.

### Decision 7 — Kafka surface

Channel side (`services/atlas-channel/atlas.com/channel/`):

- `kafka/message/monster/kafka.go`: `CommandTypeKill = "KILL"`; `KillCommandBody{ CharacterId uint32; SkillId uint32 }` (doc comment: skill id is for traceability/logging only — atlas-monsters does not resolve it).
- `monster/producer.go`: `KillCommandProvider(f, monsterId, characterId, skillId)` — same `monster2.Command[...]` envelope and `producer.CreateKey(int(monsterId))` key as every sibling.
- `monster/processor.go`: `Kill(f field.Model, monsterId, characterId, skillId uint32) error` emitting on `EnvCommandTopic`.

Monsters side (`services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/`):

- `kafka.go`: mirrored constant + `killCommandBody`.
- `consumer.go`: register `handleKillCommand` (type-gated like every sibling handler) delegating to `ProcessorImpl.Kill`.

No other consumers of the monster command topic exist, and command bodies are additive — no compatibility concern.

### Decision 8 — `Y()` accessor (FR-7)

`Extract` already threads `rm.Y` into `Model.y` (`rest.go` line 115); the only gap is the accessor. Add to `data/skill/effect/model.go`, mirroring `X()`:

```go
// Y exposes the skill's generic `y` attribute (e.g. Mortal Blow's
// instant-kill chance percent).
func (m Model) Y() int16 {
    return m.y
}
```

### Decision 9 — Stale comment fix (FR-6)

`character_attack_projectile.go` line 118: remove "Mortal Blow" from the `TODO(task-007)` passive no-consume list (Expert Marksmanship / Claw Mastery stay). §4.5 of the PRD proved Mortal Blow consumes an arrow client-side; the server plan already matches.

## 3. Data Flow

```
client ──(ranged attack, skillId=3110001/3210001)──▶ atlas-channel processAttack
  ├─ ownership guard (existing): unowned skill id ⇒ session destroyed
  ├─ se resolved at owned level (existing)
  ├─ per DamageInfo: reflect? ─yes→ no damage, no proc
  │                   └─no→ applyDamage (DAMAGE cmd) → status apply → onDamageApplied
  │                          └─ ranged + MB skill id ⇒ mortalBlowTryProc
  │                              ├─ x/y ≤ 0 ⇒ skip          (tenant data)
  │                              ├─ snapshot GetById (pre-attack HP; async drift accepted, FR-2)
  │                              ├─ hp ≤ maxHp·x/100 ? roll 1–100 ≤ y ?
  │                              └─ KILL cmd ──▶ atlas-monsters (same partition as DAMAGE, after it)
  └─ broadcast + projectile emit (unchanged; one arrow consumed)

atlas-monsters handleKillCommand → Kill
  ├─ monster missing/dead ⇒ drop (attack already killed it)
  ├─ boss lookup: error ⇒ drop (fail-closed) ; Boss() ⇒ drop
  └─ damageCore(single max line) ⇒ damaged + killed events, EXP/drops credit,
     cooldown clears, registry removal, revives — identical to any kill
```

## 4. Error Handling

| Failure | Where | Behavior |
|---|---|---|
| `x`/`y` missing or ≤ 0 in tenant data | channel | silent skip (inert passive) |
| Monster snapshot fetch fails | channel | Debugf, skip; attack pipeline unaffected |
| `KILL` emit fails | channel | Errorf, swallow; attack pipeline unaffected |
| Monster gone/dead at consume time | monsters | Debugf, drop (expected race with the attack's own kill) |
| Boss info lookup fails | monsters | Errorf, drop — **fail-closed**, diverges from DrainMp's fail-open by design (FR-4) |
| Monster is a boss | monsters | Debugf, drop (channel cannot pre-screen; this is the authoritative guard) |

Nothing in the proc path can abort or delay damage application, broadcast, or projectile emission (FR-5).

## 5. Testing

Channel (`character_attack_common_test.go` conventions, Builder pattern, no `*_testhelpers.go`):

- `mortalBlowEligible`: hp exactly at threshold ⇒ true; one above ⇒ false; truncating division pinned (e.g. maxHp=999, x=20 ⇒ threshold 199); `x=0`, `maxHp=0` ⇒ false; no overflow at maxHp near `MaxUint32` (uint64 widening).
- `mortalBlowKillRoll`: `roll==y` ⇒ true; `roll==y+1` ⇒ false; `y=0` ⇒ false; `y=100` ⇒ always true.
- Gating via `processDamageInfoEntry`/`onDamageApplied` with fake deps: non-Mortal-Blow ranged skill ⇒ no snapshot fetch; melee/magic with the id ⇒ no proc; reflected entry ⇒ no proc; status-only entry ⇒ no proc.
- Failure isolation: snapshot-fetch error and emit error both leave the entry's damage application already done and return normally.

Monsters (`processor` tests with `testInformationLookup` stub, as DrainMp tests do):

- Non-boss at any HP ⇒ killed event emitted, monster removed from registry, damage summary credits the character.
- Boss ⇒ no kill, no events.
- Info lookup error ⇒ no kill (fail-closed pinned by test).
- Missing/dead monster ⇒ no-op.
- `Damage` behavior unchanged post-refactor (existing tests must stay green; reflect still fires for `Damage`, never for `Kill`).

## 6. File Touch List

| File | Change |
|---|---|
| `services/atlas-channel/.../socket/handler/character_attack_common.go` | `mortalBlowEligible`, `mortalBlowKillRoll`, `mortalBlowTryProc`; ranged branch in `onDamageApplied`; delete `// TODO Mortal Blow` |
| `services/atlas-channel/.../socket/handler/character_attack_projectile.go` | FR-6 comment fix |
| `services/atlas-channel/.../data/skill/effect/model.go` | `Y()` accessor |
| `services/atlas-channel/.../kafka/message/monster/kafka.go` | `CommandTypeKill`, `KillCommandBody` |
| `services/atlas-channel/.../monster/producer.go` | `KillCommandProvider` |
| `services/atlas-channel/.../monster/processor.go` | `Kill(...)` |
| `services/atlas-monsters/.../kafka/consumer/monster/kafka.go` | mirrored constant + body |
| `services/atlas-monsters/.../kafka/consumer/monster/consumer.go` | `handleKillCommand` registration |
| `services/atlas-monsters/.../monster/processor.go` | `Damage` → `checkReflect` + `damageCore` split; `Kill(...)` |
| Tests | per §5, both services |

## 7. Verification

`go test -race ./...`, `go vet ./...`, `go build ./...` in both changed modules; `docker buildx bake atlas-channel atlas-monsters` from the worktree root; `tools/redis-key-guard.sh` from repo root. No shared-lib changes, so no Dockerfile `COPY` additions.

## 8. Non-Goals Reconfirmed

Per the PRD: no HP/MP-on-kill, no job-range trigger, no client-side work, no projectile changes beyond the comment, no new packets, no distance validation, none of the neighboring TODOs.
