# Combo Drain (Aran) — Design

Task: task-166-combo-drain
Status: Approved PRD (v2) → design phase
Created: 2026-07-10
Revised: 2026-08-07 — rebased onto `main`; approach decision re-taken against
the current `processAttack` (B → C); version dimension added (§6).
Revised: 2026-08-13 — merged `main`; every line reference re-derived; §2
extended with the post-damage effects that landed in the window (task-216
Energy Charge, task-217 Aran Combo Counter) and why neither changes the
one-read ceiling or Approach C. Approach decision unchanged.

## 1. Problem Recap

The `COMBO_DRAIN` buff (Aran skill 21100005) applies correctly, but the attack
pipeline never consults it, so the heal never fires. The gap is the
`// TODO Combo Drain` at
`services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go:1117`.
Required behavior (PRD): once per accepted attack, heal the attacker
`totalDamage * x / 100` HP, where `x` is the buff statup amount, clamped to
`math.MaxInt16`, emitted through the existing `character.Processor.ChangeHP`
Kafka command. Buff-only gate, all attack types, no per-monster running-total
quirk, and no version branch (PRD FR-5).

Single service touched: **atlas-channel**.

## 2. What `main` looks like now

The v1 design was written against a `processAttack` that had one buff consumer.
It now has two, plus a third buff-adjacent emitter:

| Consumer | Location | Fetch behavior |
|---|---|---|
| Projectile consumption gate | `character_attack_projectile.go:99`, inside `Plan` | Fetches **only** for ranged attacks that clear the Spirit-Javelin / non-consuming / weapon gates |
| Pick Pocket | `character_attack_common.go:276` `pickPocketResolveState`, called at `:919` | Whitelist gate first; fetches **only** for whitelisted skill ids |
| Aran combo orbs | `character_attack_combo.go:147` `comboOrbProductionDeps`, called at `:1027` | Does **not** fetch — delegates "is the buff active" to atlas-buffs via `UpdateStatValue` |
| Aran combo counter (task-217) | `character_aran_combo.go:165` `aranComboRefreshEligibility`, called at `:1032` | Does **not** fetch — evaluates the gate off the character fetch `processAttack` already paid for and caches it in the `character/combo` mirror |
| Energy Charge (task-216) | `character_attack_energy_charge.go:120` `energyChargeProductionDeps`, called at `:1050` | Does **not** fetch — `UpdateStatValue` with `CreateIfMissing` is exactly what keeps the attack path read-free. Its one `GetByCharacterId` (`:198`) is on the *rejected* Energy Blast path, which returns before the TODO block |

So today a melee or magic attack with a non-whitelisted skill performs **zero**
buff reads — and the three post-damage effects added since v1 kept it that way,
so the ceiling this design has to respect is still set by the two gate-first
consumers above. There is also an established per-attack memoization idiom in
the same function: `loadEffectiveStats` (`character_attack_common.go:898-912`),
a closure with a `loaded` flag, shared by the venom apply and by `drainTryHeal`.

Three further facts that invalidate v1 details:

- `buff.NewBuff` now takes seven arguments (`…, expiresAt time.Time, noExpiry bool`).
- The TODO block moved from line 420 to line 1117, and now sits after Sacrifice,
  Homing Beacon, combo-orb bookkeeping, Aran combo eligibility, Energy Charge
  and the per-skill attack-cast dispatcher.
- `isDrainSkill` (`character_attack_common.go:89`) already carries the comment
  "Aran Combo Drain is buff-driven and excluded" — the drain-family heal that
  landed alongside it deliberately left this task's gap open rather than
  folding Combo Drain into a skill-id switch. That is the same call FR-3 makes.

## 3. Approaches Considered

### Approach A — Independent buff fetch at the TODO site

Combo Drain calls `buff.NewProcessor(l, ctx).GetByCharacterId(characterId)`
itself at the TODO site. Zero churn to existing code.

- Rejected. It adds an unconditional REST read to every attack while the
  ranged and Pick-Pocket paths still issue their own, so a ranged Pick Pocket
  attack would perform three reads. Violates the PRD's one-read ceiling.

### Approach B — Single eager fetch threaded into every consumer (v1's choice)

`processAttack` fetches buffs once, eagerly, before `pp.Plan`, and passes the
`[]buff.Model` slice to `Plan`, to `pickPocketResolveState`, and to Combo
Drain. `ProjectileProcessor.Plan` and `pickPocketResolveState` lose their
internal fetches.

- Was the right call when `Plan` was the only other consumer. It is no longer:
  both existing consumers deliberately gate *before* fetching, and an eager
  hoist discards that. Every attack — including the melee/magic majority that
  today read nothing — would pay a REST read even when no consumer needs one.
- It also changes two public-ish signatures (`ProjectileProcessor.Plan`,
  `pickPocketResolveState`) and inverts their documented failure postures into
  the caller, for no behavioral gain.

### Approach C — Per-attack memoized buff loader (recommended)

A `loadBuffs()` closure built exactly like the neighbouring
`loadEffectiveStats`: fetch on first call, cache the result (including the
degraded `nil` on error) for the rest of the attack. It is injected into the
two existing consumers in place of their raw processor, and called by Combo
Drain at the TODO site.

- Ceiling of one read per attack, which is what the PRD asks for.
- Floor is preserved for the paths that gate first: they still call the loader
  only when they would have fetched, and Combo Drain's own call is the only
  unconditional one — so the melee/magic case goes 0 → 1, not 0 → 2.
- Both existing consumers already take a *function* rather than a processor
  (`pickPocketResolveState(l, getBuffs func(uint32) ([]buff.Model, error), …)`),
  so Pick Pocket needs no signature change at all — it receives the loader
  instead of `buff.NewProcessor(l, ctx).GetByCharacterId`. Only the projectile
  planner changes shape.
- Mirrors an idiom a reviewer of this file already knows.

v1 rejected this ("laziness buys nothing") on the premise that Combo Drain
demands buffs unconditionally so the loader always fires. That premise held
when the alternative was a single shared eager fetch; with two gate-first
consumers the laziness is what preserves *their* current cost, and the
memoization is what caps the total. **Decision reversed: Approach C.**

### Approach D — Skill-ownership short-circuit before fetching

Skip the buff read when `c.Skills()` (already in hand) does not contain
21100005 — which would keep the melee/magic floor at zero reads.

- Rejected, unchanged from v1: it changes semantics. FR-3 mandates a buff-only
  gate; a `COMBO_DRAIN` statup granted by any non-skill source (GM command,
  item effect) would be silently ignored. Noted here because the cost argument
  for it is stronger than it was in v1 — if the owner later decides the read is
  too expensive on the melee hot path, this is the lever, and it is a PRD
  change, not a design change.

**Decision: Approach C.**

Deviation note (deliberate, semantics-preserving): FR-1's letter says the
attacker's buffs are read "at the TODO site". Under Approach C the *read* may
have already happened earlier in the attack (if the projectile gate or Pick
Pocket triggered it); the *evaluation and emission* stay at the TODO site,
post-damage, as FR-3 orders. The snapshot is taken at attack-accept time, which
is the correct moment for "was the buff active when the attack landed".

## 4. Architecture

### 4.1 The loader (in `processAttack`)

Placed immediately before `pp := NewProjectileProcessor` (currently
`character_attack_common.go:888`), so every consumer below can take it:

```go
// One buff snapshot per attack, shared by the projectile consumption
// gate, Pick Pocket, and post-damage buff-driven effects (Combo Drain).
// Fetched at most once and only when a consumer actually needs it; a
// lookup failure is cached as "no buffs active" for every consumer and
// never aborts the attack. Mirrors loadEffectiveStats below.
var attackBuffs []buff.Model
attackBuffsLoaded := false
loadBuffs := func(characterId uint32) ([]buff.Model, error) {
    if attackBuffsLoaded {
        return attackBuffs, nil
    }
    attackBuffsLoaded = true
    bs, bErr := buff.NewProcessor(l, ctx).GetByCharacterId(characterId)
    if bErr != nil {
        l.WithError(bErr).Warnf("Unable to load buffs for character [%d] attack; assuming none active.", characterId)
        attackBuffs = nil
        return nil, bErr
    }
    attackBuffs = bs
    return attackBuffs, nil
}
```

The signature deliberately matches
`buff.Processor.GetByCharacterId(uint32) ([]buff.Model, error)` so it is a drop-in
for both existing consumers, and it propagates the error on the first call so
each consumer keeps logging its own domain-specific warning exactly as today.
Subsequent calls return the cached (possibly `nil`) slice with `nil` error —
correct, because the failure has already been surfaced once and each consumer's
degraded behavior is "assume no buffs".

Degraded posture per consumer is unchanged:

- Projectile planner: no buffs ⇒ "assume no Soul Arrow / Shadow Stars" ⇒
  over-consume rather than break the attack.
- Pick Pocket: no buffs ⇒ proc disabled.
- Combo Drain: no buffs ⇒ no heal (FR-1).

### 4.2 `ProjectileProcessor` change

```go
type ProjectileProcessor interface {
    Plan(c character.Model, ai packetmodel.AttackInfo, se effect.Model, getBuffs func(characterId uint32) ([]buff.Model, error)) (*ProjectilePlan, bool)
    Emit(characterId uint32, plan *ProjectilePlan) error
}
```

`ProjectileProcessorImpl` drops its `bp buff.Processor` field;
`NewProjectileProcessor(l, ctx)` keeps its signature. Inside `Plan`, the sole
change is `p.bp.GetByCharacterId(c.Id())` → `getBuffs(c.Id())`; the warn block,
`projectileConsumptionSkipped` and `computeCount` are untouched, and the fetch
stays behind the same gates it is behind today. Behavior-neutral.

`Plan` has exactly one production call site. No test constructs
`ProjectileProcessorImpl` or calls `Plan` (the projectile tests exercise the
pure helpers `computeCount`/`resolvePlan`/`hasBuff`/`requiredClassification`),
so no projectile test edits are needed — re-verify this before assuming it.

### 4.3 Pick Pocket change

One argument at the call site (`character_attack_common.go:919-925`):
`buff.NewProcessor(l, ctx).GetByCharacterId` → `loadBuffs`. No signature
change, no behavior change — it already accepts a `getBuffs` function and still
calls it only for whitelisted skills.

### 4.4 Combo Drain evaluation — pure helpers + proc function

New file `socket/handler/character_attack_combo_drain.go`, following the
established handler style (small pure functions pinned by tests —
`computeReflect`, `drainHealAmount`, `mpEaterAbsorbAmount` — plus a `TryProc`
orchestrator with injected side effects — `mpEaterTryProc`, `drainTryHeal`,
`pickPocketTryProc`):

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
    getBuffs func(characterId uint32) ([]buff.Model, error),
    changeHP func(f field.Model, characterId uint32, amount int16) error,
    f field.Model,
    characterId uint32,
    ai packetmodel.AttackInfo,
)
```

`comboDrainTryProc` logic:

1. `buffs, err := getBuffs(characterId)`; on error, `Debugf` and return (the
   loader already logged the warning at Warn level — FR-1).
2. `percent, ok := buffStatAmount(buffs, ts.TemporaryStatTypeComboDrain)`;
   return if `!ok`.
3. `total := attackTotalDamage(ai)`; `heal := comboDrainHealAmount(total, percent)`;
   return if `heal <= 0` (covers `total == 0`, `percent <= 0`).
4. Debug log `characterId`, `total`, `percent`, `heal` (observability NFR).
5. `changeHP(f, characterId, heal)`; on error, `Errorf` and continue.

Taking the loader rather than a slice keeps the "at most one read, and only if
someone needs it" property in one place and makes the proc directly testable
against a counting fake (PRD AC "at most one buff REST read per attack").

### 4.5 Call site (replaces the line-1117 TODO)

In `processAttack`, at the TODO block (post-broadcast, after the projectile
`Emit` and the sibling post-damage effects):

```go
comboDrainTryProc(l, loadBuffs, cp.ChangeHP, s.Field(), s.CharacterId(), ai)
```

`cp` is the `character.Processor` constructed at `character_attack_common.go:777`;
`ChangeHP` emits the Kafka command and atlas-character owns max-HP clamping
downstream. Ordering satisfies FR-3: after all per-monster damage processing
(`ai.DamageInfo()` is final), and independent of broadcast success. Exactly one
line is removed (`// TODO Combo Drain`) and one inserted.

### 4.6 Edge-case decisions

- **Multiple buffs carrying COMBO_DRAIN**: first non-expired match wins
  (deterministic, mirrors `hasBuff`). In practice only one source exists.
- **Expired buffs**: skipped via `b.Expired()`, same as `hasBuff`.
- **Reflected entries**: their damage lines still count toward `totalDamage`.
  FR-2 defines the total as the plain sum over `di.Damages()` with no carve-out,
  and the client-declared damage is "damage dealt by the attack".
- **Overflow**: sum in `uint64` (`total ≤ 15 targets × 15 lines × MaxUint32
  ≈ 9.7e11` cannot wrap). Before multiplying, early-saturate: for any
  `percent ≥ 1`, `total ≥ math.MaxInt16 * 100` already guarantees
  `total*percent/100 ≥ MaxInt16`, so return `MaxInt16` without multiplying.
  Below that bound, `total * uint64(percent) ≤ 3.28e6 × MaxInt32 ≈ 7e15` fits
  `uint64` comfortably; divide by 100, clamp to `math.MaxInt16`, narrow to
  `int16`. No intermediate can wrap for any `int32` percent.
- **Zero-amount commands**: never emitted (`heal <= 0` gate), per FR-2.
- **No job / attack-type / skill-ownership / version check**: the buff is the
  whole gate (FR-3, FR-5, Approach D rejection).

## 5. Data Flow

```
client attack packet (melee | ranged | magic | touch handler)
  → processAttack (character_attack_common.go)
      → loadBuffs (memoized, ≤1 REST read) ──┐
      → pp.Plan(c, ai, se, loadBuffs)        │  (ranged consumption gate)
      → pickPocketResolveState(…, loadBuffs) │  (whitelisted skills only)
      → per-monster damage loop              │
      → broadcast, projectile Emit           │
      → comboDrainTryProc(l, loadBuffs, …) ◄─┘
          → character.Processor.ChangeHP
              → Kafka CHANGE_HP command → atlas-character (clamps to max HP)
              → STAT_CHANGED packet to client (existing path)
```

No new REST endpoints, Kafka topics, message shapes, packets, or template
changes. No schema changes. All identifiers come from `libs/atlas-constants`
(`TemporaryStatTypeComboDrain`; `AranStage2ComboDrainId` is not needed at
runtime — the gate is the stat, not the skill). Multi-tenancy: everything runs
on the handler's request `ctx`, as today.

## 6. Version Dimension

### 6.1 Why there is no version branch

The design is version-blind by construction, and that is the intended outcome
rather than an omission:

- The gate is a temporary-stat value in the character's live buff list. On a
  version whose client has no Aran, no `COMBO_DRAIN` statup can ever be
  produced (atlas-data emits it only for skill 21100005,
  `skill/reader.go:407-408`, and the skill does not exist in those tenants' WZ),
  so the gate never fires. Availability is data-driven.
- The percent is the statup amount, read from the tenant's own WZ through
  atlas-data, so a per-version difference in the level→`x` curve is honored
  with no code change and no hard-coded value.
- The `COMBO_DRAIN` bit is allocated unconditionally for every version —
  `libs/atlas-packet/model/character_temporary_stat.go:163`, inside the
  contiguous pre-SoulStone block that precedes the first version-gated slot
  (`TemporaryStatTypeFlying`, bit 82, behind the `post87` gate at `:186`) — so
  the buff already encodes and decodes correctly everywhere. Re-verified after
  the 2026-08-13 merge.
- No skill-id comparison is introduced, so `tools/skill-job-id-guard.sh` is not
  engaged; independently, `21100005` has no row in
  `docs/tasks/task-187-version-aware-id-semantics/audit/divergences.csv`.
- Precedent 1: the Aran combo-orb handler in the same package gates on the
  character's learned skills, not on version
  (`character_attack_combo.go:37` `comboSkillIds`, called at `:173`), and its
  only version note is a comment recording that the id it compares is
  version-stable per the task-187 audit.
- Precedent 2 (task-217, Aran Combo Counter): it faced a genuinely
  version-varying value — the client's own `ClearCombo` idle timer is 3000 ms
  on v83/v84/v87/v92/jms185 and 5000 ms on v95 — and still refused a compiled
  major-version branch, resolving it as tenant handler configuration
  (`idleResetMs`, `character_aran_combo.go:38` `idleWindowFromOptions`). Combo
  Drain has no such value at all: its one variable, the percent, already
  arrives from the tenant's WZ via the buff statup. So the bar this design has
  to clear for "no version branch" is lower than one an adjacent Aran feature
  already cleared.

A `MajorVersion`/`MajorAtLeast`/`IsRegion` check appearing in this diff should
be treated as a defect (PRD FR-5, NFR).

### 6.2 Per-version applicability

Reproduced from PRD §4A; each cell is backed by a checked-in artifact. Both
columns were re-derived from the repo after the 2026-08-13 `main` merge (the
two commands in plan.md Task 4 Step 3) and are unchanged.

| Tenant template | Aran `21100005` present | Attack handlers routed | Combo Drain scope |
|---|---|---|---|
| `gms_12_1`  | no  | none                            | N/A — no Aran, no attack routing |
| `gms_48_1`  | no  | melee, ranged, magic (no touch) | N/A — no Aran |
| `gms_61_1`  | no  | melee, ranged, magic, touch     | N/A — no Aran |
| `gms_72_1`  | no  | melee, ranged, magic, touch     | N/A — no Aran |
| `gms_79_1`  | yes | melee, ranged, magic, touch     | **in scope** |
| `gms_83_1`  | yes | melee, ranged, magic, touch     | **in scope** |
| `gms_84_1`  | yes | melee, ranged, magic, touch     | **in scope** |
| `gms_87_1`  | yes | melee, ranged, magic, touch     | **in scope** |
| `gms_92_1`  | yes | **none**                        | blocked — template bring-up (§6.4) |
| `gms_95_1`  | yes | melee, ranged, magic, touch     | **in scope** |
| `jms_185_1` | yes | melee, ranged, magic, touch     | **in scope** |

Sources: `libs/atlas-constants/gen/wzsnapshot/<key>.json` (`skills`), with
`PROVENANCE.md` for the drain method; the generated
`libs/atlas-constants/skill/version_<key>_gen.go` and
`libs/atlas-constants/job/version_<key>_gen.go` agree (jobs 2100/2110/2111/2112
present in exactly the same seven versions);
`services/atlas-configurations/seed-data/templates/template_<key>.json` for
handler routing.

Note on attack types: there is no `CharacterEnergyAttackHandle`.
`AttackTypeEnergy` is produced by `CharacterTouchAttackHandle`
(`character_attack_touch.go:18`), so the "energy" acceptance criterion is
exercised through the touch path — routed on every in-scope version.

### 6.3 Wire coverage backing the in-scope versions

From `docs/packets/audits/status.json` (nine matrix columns —
`docs/packets/PROCESS.md`):

| Op | Direction | Coverage |
|---|---|---|
| `CLOSE_RANGE_ATTACK` | serverbound | `verified` on all 9 |
| `RANGED_ATTACK` | serverbound | `verified` on all 9 |
| `MAGIC_ATTACK` | serverbound | `verified` on all 9 |
| `TOUCH_MONSTER_ATTACK` | serverbound | `n-a` on `gms_v48`; `verified` on the other 8 |
| `STAT_CHANGED` | clientbound | `verified` on all 9 |
| `GIVE_BUFF` | clientbound | `verified` on all 9 |

Every in-scope version therefore sums damage from a verified decoder and
renders the heal through a verified stat-update codec. This task adds no codec
and no version gate, so it promotes no matrix cell and declares no
`coverage-manifest.yaml`; `status.json` must be unchanged by the branch.

### 6.4 `gms_92_1` and `gms_12_1` — blocked, with cause

`template_gms_92_1.json` routes ~49 handlers (vs ~131 for `gms_95_1`) and no
attack handler; `template_gms_12_1.json` routes ~24. Aran exists on the v92
client, so Combo Drain is unreachable there only because the attack packets are
not routed.

Routing them is not producible within this task: it needs `gms_v92` serverbound
opcodes for the four attack ops, and `gms_v92` is not a matrix column — there
is no verified opcode source, and guessing one would violate the project's
grounding rule. That is a version bring-up (`/bringup-version`), not a skill
feature. Because the implementation is version-blind, Combo Drain will work on
both versions the moment their attack handlers are routed, with no further work
in this area.

## 7. Testing

New `socket/handler/character_attack_combo_drain_test.go`, table-driven, using
existing production constructors (`packetmodel.NewAttackInfo` /
`NewDamageInfo` builders, `buff.NewBuff` — **seven** arguments now,
`stat.NewStat`) — no `*_testhelpers.go`, per project rule. Package-level names
already taken: `buffWithStat`/`expiredBuffWithStat`
(`character_attack_projectile_test.go:58,63` — both fixed-amount, hence the new
amount-parameterized helpers) and `testField`
(`mystic_door_enter_test.go:30`, reuse as-is). All three re-confirmed present
after the 2026-08-13 merge.

Pure-helper tests:

- `attackTotalDamage`: single monster single line; multi monster multi line;
  empty `DamageInfo`; large lines summed in `uint64`.
- `buffStatAmount`: present; absent; expired buff skipped; first live match
  wins across multiple buffs; matching stat alongside other stats.
- `comboDrainHealAmount`: nominal (`D*x/100`); truncation (integer div); zero
  damage; `percent <= 0`; clamp boundary — inputs producing exactly `MaxInt16`
  (unclamped) and `MaxInt16 + 1` (clamped).

Proc tests (`comboDrainTryProc` with a recording `changeHP` closure and a
counting `getBuffs` fake):

- Buff present, single monster → exactly one call with expected amount.
- Buff present, multi monster/lines → one call, plain total (no per-monster
  over-heal) — pins the anti-Cosmic-quirk AC.
- Buff absent → zero calls. Expired-only → zero calls.
- Zero total damage → zero calls. Heal truncating to zero → zero calls.
- `getBuffs` returns an error → zero calls, no panic (the buff-fetch-error AC).
- `changeHP` returns an error → logged, no panic, no retry.
- One case per `AttackType` (melee, ranged, magic, energy) to pin that no type
  filter creeps in. This is the version-facing AC too: the same table passes
  unchanged for every version because the proc has no version input.

Loader test (`processAttack`-level cost AC, PRD §10): a test over `loadBuffs`
asserting that N calls produce exactly one underlying fetch, and that a failed
first fetch is cached as `nil` without re-fetching.

Projectile regression: existing `character_attack_projectile_test.go` should
need no substantive change — confirm before assuming, since `Plan`'s signature
changes.

Version testing posture: there is deliberately **no** per-version test matrix
for this feature. With no version input reaching any function in the diff, a
per-version table would assert only that Go ignores an unused variable. The
version claims are instead pinned by the evidence citations in §6.2/§6.3 and by
the AC that the diff contains no version branch and touches no template.

## 8. Verification Gates

- `go test -race ./...`, `go vet ./...`, `go build ./...` clean in
  `services/atlas-channel/atlas.com/channel`.
- From the repo root: `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`,
  `tools/skill-job-id-guard.sh`, `tools/buff-duration-guard.sh`,
  `tools/lint.sh --check` all clean.
- `docker buildx bake atlas-channel` from the worktree root (mandatory —
  `go.mod` untouched but the project rule requires it).
- Version-scope gates: `git diff main` shows no `MajorVersion` /
  `MajorAtLeast` / `IsRegion` in added lines;
  `git diff --stat main -- services/atlas-configurations/seed-data/templates/`
  and `git diff --stat main -- docs/packets/` are both empty.
- `docs/TODO.md:151` Combo Drain item checked off; TODO marker gone from
  `character_attack_common.go`.
- `superpowers:requesting-code-review` before opening the PR (project rule).

## 9. Out of Scope

Combo-orb mechanics (already landed as `comboOrbTryUpdate`), the Aran combo
counter and its `character/combo` mirror (already landed as task-217), Energy
Charge (task-216), the sibling TODOs
remaining in the same block (Flame Thrower, Snow Charge, Hamstring, Slow,
Blind, Paladin/White Knight charges, Three Snails, Heavens Hammer,
ComboTempest, BodyPressure), packet/writer changes, Cosmic's running-total
over-heal, and the `gms_92_1`/`gms_12_1` template bring-ups (§6.4). Sibling
tasks edit neighbouring TODO lines in their own worktrees — keep the diff to
this block to exactly one deleted line and one inserted call.
