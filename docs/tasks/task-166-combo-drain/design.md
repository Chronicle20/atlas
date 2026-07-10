# Combo Drain (Aran) — Design

Task: task-166-combo-drain
Status: Approved PRD → design phase
Created: 2026-07-10

## 1. Problem Recap

The `COMBO_DRAIN` buff (Aran skill 21100005) applies correctly, but the attack
pipeline never consults it, so the heal never fires. The gap is the
`// TODO Combo Drain` at
`services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go:420`.
Required behavior (PRD): once per accepted attack, heal the attacker
`totalDamage * x / 100` HP, where `x` is the buff statup amount, clamped to
`math.MaxInt16`, emitted through the existing `character.Processor.ChangeHP`
Kafka command. Buff-only gate, all attack types, no per-monster running-total
quirk.

Single service touched: **atlas-channel**.

## 2. Approaches Considered

### Approach A — Independent buff fetch at the TODO site

Combo Drain calls `buff.NewProcessor(l, ctx).GetByCharacterId(characterId)`
itself at line 420, exactly as FR-1 is worded. Zero churn to existing code.

- Cost: ranged attacks that reach the projectile planner's buff check perform
  **two** buff REST reads per attack (the planner already fetches inside
  `ProjectileProcessorImpl.Plan`, `character_attack_projectile.go:97`).
- The PRD's performance NFR explicitly says to prefer reusing the projectile
  path's fetch over a second one.

### Approach B — Single shared fetch per attack, threaded into Plan (recommended)

`processAttack` fetches buffs **once**, early (where `pp.Plan` is called
today), and passes the `[]buff.Model` slice to both consumers:

- `ProjectileProcessor.Plan` gains a `buffs []buff.Model` parameter and stops
  fetching internally (its `bp buff.Processor` field is removed).
- Combo Drain evaluates the same slice at the TODO site, post-damage.

Exactly one buff lookup per attack, ever — for melee/magic/energy this is the
one lookup Combo Drain inherently needs (there is no cheap short-circuit; see
§3.1), for ranged it replaces the planner's existing lookup rather than adding
one. This is the PRD's stated preference.

- Cost: `ProjectileProcessor` interface signature change and mechanical
  updates to the projectile tests that currently inject a fake
  `buff.Processor`. Those tests get *simpler* (pass a slice instead of faking
  a processor).

### Approach C — Lazy memoized buff loader shared by both (loadVenomStats pattern)

A per-attack `loadBuffs()` closure memoized like `loadVenomStats`
(`character_attack_common.go:325-339`), injected into the projectile
processor and used by Combo Drain.

- Rejected: laziness buys nothing. Combo Drain demands buffs unconditionally
  on every attack (buff-only gate, no pre-check available), so the memoized
  loader always fires — it is Approach B with extra machinery and an
  awkward function-valued constructor parameter.

### Approach D — Skill-ownership short-circuit before fetching

Skip the buff fetch when `c.Skills()` (already in hand) does not contain
21100005.

- Rejected: it changes semantics. FR-3 mandates a buff-only gate; a
  COMBO_DRAIN statup granted by any non-skill source (GM command, item
  effect) would be silently ignored. The PRD only permits short-circuits that
  are semantics-preserving, and none is available.

**Decision: Approach B.**

Deviation note (deliberate, semantics-preserving): FR-1's letter says "fetch
… at the line-420 TODO site". Under Approach B the *fetch* moves to the top of
the attack flow (pre-Plan) to satisfy the performance NFR's single-fetch
preference; the *evaluation and emission* stay at the TODO site, post-damage,
as FR-3 orders. The buff snapshot is taken at attack-accept time, which is the
correct moment for "was the buff active when the attack landed".

## 3. Architecture

### 3.1 Buff acquisition (in `processAttack`)

Immediately before `pp.Plan(...)` (`character_attack_common.go:316`):

```go
buffs, buffErr := buff.NewProcessor(l, ctx).GetByCharacterId(s.CharacterId())
if buffErr != nil {
    l.WithError(buffErr).Warnf("Unable to load buffs for character [%d] attack; assuming none active.", s.CharacterId())
    buffs = nil
}
```

One warning replaces the planner's internal one
(`character_attack_projectile.go:100-106`, which is deleted along with the
fetch). The degraded posture serves both consumers and preserves each
feature's PRD-mandated behavior:

- Projectile planner: `buffs = nil` ⇒ "assume no Soul Arrow / no buff-driven
  count change" — identical to its current failure posture (over-consume
  rather than break the attack).
- Combo Drain: `buffs = nil` ⇒ no heal — FR-1's failure posture.

The attack pipeline is never aborted by a buff-service failure (resilience
NFR).

### 3.2 ProjectileProcessor change

```go
type ProjectileProcessor interface {
    Plan(c character.Model, ai packetmodel.AttackInfo, se effect.Model, buffs []buff.Model) (*ProjectilePlan, bool)
    Emit(characterId uint32, plan *ProjectilePlan) error
}
```

`ProjectileProcessorImpl` drops its `bp buff.Processor` field;
`NewProjectileProcessor(l, ctx)` keeps its signature. Inside `Plan`, the
fetch+warn block is replaced by direct use of the parameter. `hasBuff` and
`computeCount` already operate on `[]buff.Model` and are untouched.

This is the only refactor to existing behavior, and it is behavior-neutral:
the same data reaches the same decision points, fetched once instead of once
per consumer.

### 3.3 Combo Drain evaluation — pure helpers + proc function

New file `socket/handler/character_attack_combo_drain.go`, following the
established handler style (small pure functions pinned by tests —
`computeReflect`, `mpEaterAbsorbAmount` — plus a `TryProc` orchestrator with
injected side effects — `mpEaterTryProc`):

```go
// buffStatAmount returns the Amount of the first stat change of statType
// carried by a non-expired buff, mirroring hasBuff's matching rules.
func buffStatAmount(buffs []buff.Model, statType ts.TemporaryStatType) (int32, bool)

// attackTotalDamage sums every damage line across every DamageInfo entry.
// uint64 so a full multi-target attack cannot overflow the sum.
func attackTotalDamage(ai packetmodel.AttackInfo) uint64

// comboDrainHealAmount computes totalDamage * percent / 100 in integer
// arithmetic, returning 0 when percent <= 0 or totalDamage == 0, and
// clamping to math.MaxInt16 before narrowing.
func comboDrainHealAmount(totalDamage uint64, percent int32) int16

// comboDrainTryProc evaluates Combo Drain for one accepted attack and emits
// at most one ChangeHP via the injected changeHP. Failures are logged and
// swallowed — never abort the attack pipeline.
func comboDrainTryProc(
    l logrus.FieldLogger,
    buffs []buff.Model,
    changeHP func(f field.Model, characterId uint32, amount int16) error,
    f field.Model,
    characterId uint32,
    ai packetmodel.AttackInfo,
)
```

`comboDrainTryProc` logic:

1. `percent, ok := buffStatAmount(buffs, ts.TemporaryStatTypeComboDrain)`;
   return if `!ok`.
2. `total := attackTotalDamage(ai)`; `heal := comboDrainHealAmount(total, percent)`;
   return if `heal <= 0` (covers `total == 0`, `percent <= 0`).
3. Debug log `characterId`, `total`, `percent`, `heal` (observability NFR).
4. `changeHP(f, characterId, heal)`; on error, `Errorf` and continue.

### 3.4 Call site (replaces the line-420 TODO)

In `processAttack`, at the TODO block (post-broadcast, after the projectile
`Emit`, where the sibling post-damage effects live):

```go
comboDrainTryProc(l, buffs, cp.ChangeHP, s.Field(), s.CharacterId(), ai)
```

`cp` is the `character.Processor` already constructed at the top of the
handler; `ChangeHP` (`character/processor.go:271`) emits the Kafka command and
atlas-character owns max-HP clamping downstream. Ordering satisfies FR-3:
after all per-monster damage processing (`ai.DamageInfo()` is final), and
independent of broadcast success. The `// TODO Combo Drain` line is removed.

### 3.5 Edge-case decisions

- **Multiple buffs carrying COMBO_DRAIN**: first non-expired match wins
  (deterministic, mirrors `hasBuff`). In practice only one source exists.
- **Expired buffs**: skipped via `b.Expired()`, same as `hasBuff`.
- **Reflected entries**: their damage lines still count toward `totalDamage`.
  FR-2 defines the total as the plain sum over `di.Damages()` with no
  carve-out, and the client-declared damage is "damage dealt by the attack".
- **Overflow**: sum in `uint64` (`total ≤ 15 targets × 15 lines × MaxUint32
  ≈ 9.7e11` cannot wrap). Before multiplying, early-saturate: for any
  `percent ≥ 1`, `total ≥ math.MaxInt16 * 100` already guarantees
  `total*percent/100 ≥ MaxInt16`, so return `MaxInt16` without multiplying.
  Below that bound, `total * uint64(percent) ≤ 3.28e6 × MaxInt32 ≈ 7e15`
  fits `uint64` comfortably; divide by 100, clamp to `math.MaxInt16`, narrow
  to `int16`. No intermediate can wrap for any `int32` percent.
- **Zero-amount commands**: never emitted (`heal <= 0` gate), per FR-2.
- **No job / attack-type / skill-ownership check**: the buff is the whole
  gate (FR-3, Approach D rejection).

## 4. Data Flow

```
client attack packet
  → processAttack (character_attack_common.go)
      → buff fetch (once)  ──────────────┐
      → pp.Plan(c, ai, se, buffs)        │  (ranged consumption gate)
      → per-monster damage loop          │
      → broadcast, projectile Emit       │
      → comboDrainTryProc(l, buffs, …) ◄─┘
          → character.Processor.ChangeHP
              → Kafka CHANGE_HP command → atlas-character (clamps to max HP)
              → stat-update packet to client (existing path)
```

No new REST endpoints, Kafka topics, message shapes, packets, or template
changes. No schema changes. All identifiers come from `libs/atlas-constants`
(`TemporaryStatTypeComboDrain`; `AranStage2ComboDrainId` is not needed at
runtime — the gate is the stat, not the skill). Multi-tenancy: everything
runs on the handler's request `ctx`, as today.

## 5. Testing

New `socket/handler/character_attack_combo_drain_test.go`, table-driven,
using existing production constructors (`packetmodel.NewAttackInfo` /
`NewDamageInfo` builders, `buff.NewBuff`, `stat.NewStat`) — no
`*_testhelpers.go`, per project rule.

Pure-helper tests:

- `attackTotalDamage`: single monster single line; multi monster multi line;
  empty `DamageInfo`.
- `buffStatAmount`: present; absent; expired buff skipped; first-match wins
  across multiple buffs; matching stat on a buff alongside other stats.
- `comboDrainHealAmount`: nominal (`D*x/100`); truncation (integer div);
  zero damage; `percent <= 0`; clamp boundary — inputs producing exactly
  `MaxInt16` (unclamped) and `MaxInt16 + 1` (clamped to `MaxInt16`).

Proc tests (`comboDrainTryProc` with a recording `changeHP` closure):

- Buff present, single monster → exactly one call with expected amount.
- Buff present, multi monster/lines → one call, plain total (no per-monster
  over-heal) — pins the anti-Cosmic-quirk AC.
- Buff absent → zero calls.
- Zero total damage → zero calls.
- `buffs == nil` → zero calls. This is also the buff-fetch-error AC: the
  handler collapses a fetch error to `nil` buffs (§3.1), so the nil case *is*
  the error case at proc level.
- `changeHP` returns error → logged, no panic, no retry.

Projectile regression: existing `character_attack_projectile_test.go` cases
updated mechanically to pass `buffs` into `Plan` instead of injecting a fake
`buff.Processor`; assertions unchanged (Soul Arrow skip, count computation,
fetch-failure ⇒ assume-none is now the caller's nil-slice case).

Attack-type coverage (melee/ranged/magic/energy AC): `comboDrainTryProc` is
attack-type-blind by construction — the proc table includes one case per
`AttackType` to pin that no type filter creeps in.

## 6. Verification Gates

- `go test -race ./...`, `go vet ./...`, `go build ./...` clean in
  `services/atlas-channel/atlas.com/channel`.
- `tools/redis-key-guard.sh` clean from repo root.
- `docker buildx bake atlas-channel` from the worktree root (mandatory —
  `go.mod` untouched but the service builds anyway per project rule).
- `docs/TODO.md` Combo Drain line item checked off; TODO marker gone from
  `character_attack_common.go`.

## 7. Out of Scope (unchanged from PRD)

Combo-orb mechanics, sibling TODOs in the same block (Energy Drain/Vampire/
Drain is task-147, Mortal Blow is task-152, etc.), packet/writer changes,
Cosmic's running-total over-heal. Note for the plan phase: several sibling
tasks edit the same TODO block in their own worktrees; keep the diff to this
block minimal (delete exactly one TODO line, insert one call) to ease later
merges.
