# Attack-Side Drain HP Gain — Design

Task: task-147-attack-drain-hp-gain
Status: Approved for planning
PRD: `docs/tasks/task-147-attack-drain-hp-gain/prd.md`

## 1. Summary

Wire the drain-family heal (Assassin Drain 4101005, Marauder Energy Drain 5111004, Thunder Breaker Energy Drain 15111001, Night Walker Vampire 14101006) into `processAttack`'s per-monster post-damage hook in `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go`, following the MP Eater proc as the structural template. Per damaged monster: `heal = min(monsterMaxHp, floor(totalDamage × X / 100), effectiveMaxHp / 2)`, emitted via the existing `character.Processor.ChangeHP` command path. All math lives in a pure, table-testable helper; all failures are logged and swallowed.

Only atlas-channel changes. No new packets, topics, endpoints, or data-model changes.

## 2. Key design decisions

### D1 — Hook placement: extend `onDamageApplied` (chosen) vs. alternatives

The heal must fire once per damaged monster, after damage application, including for monsters the attack kills. Three candidate placements:

**A (chosen): extend the existing `onDamageApplied` hook.** `processDamageInfoEntry` already invokes `deps.onDamageApplied` exactly once per non-reflected `DamageInfo` after damage + status apply (`character_attack_common.go:177-179`) — MP Eater's hook. Drain becomes a second branch inside the same closure `processAttack` builds. This is the position the PRD mandates (FR-4) and it inherits the reflect exclusion for free: a reflected entry deals no damage, so it correctly yields no heal.

**B (rejected): a second loop over `ai.DamageInfo()` after the main loop.** Duplicates the reflect/zero-damage filtering the main loop already performs — a second copy of "which entries actually damaged a monster" logic that can drift from the first.

**C (rejected): bake drain directly into `processDamageInfoEntry`.** Bloats the shared entry processor with skill-specific policy and forces every test of the entry processor to consider drain. The hook exists precisely so per-skill passives stay out of the shared path.

### D2 — Hook signature: pass the damage total through the hook

Drain needs the per-monster damage total; the current hook is `onDamageApplied func(monsterId uint32)`. Options:

- **Chosen: widen the hook to `func(monsterId uint32, totalDamage uint32)`.** `processDamageInfoEntry` computes the sum of `di.Damages()` once (it's already iterating them for the reflect path when applicable) and passes it. MP Eater ignores the new argument. One-line change at each of the two existing call-site references (`deps` struct doc + closure), and the flow tests updated mechanically.
- Rejected: pass the whole `packetmodel.DamageInfo` — leaks packet-model surface into the hook contract for one field; the hook's consumers only ever need "who" and "how much".
- Rejected: recompute the total inside the drain closure by re-finding the matching `DamageInfo` in `ai.DamageInfo()` — an O(n) re-scan per monster and a correctness hazard if two entries share a monster id.

Damage values are `[]uint32`; summing in `uint64` internally and clamping guards overflow, but the passed total stays `uint32` — a single monster's per-attack damage cannot legitimately exceed it (v83 per-line cap 999,999 × 15 lines).

### D3 — X source: reuse the attack's `se` effect model

`processAttack` already resolves `se = skill2.GetEffect(ai.SkillId(), sk.Level())` for the attacking skill (line 292). For the four drain skills the attack skill *is* the drain skill, so `se.X()` is exactly the per-level X the formula needs. No second effect fetch, no per-monster fetch (PRD FR-4). This differs from MP Eater, which must fetch a *different* skill's effect (the passive), and is simpler.

Guard: the drain branch only runs when `ai.SkillId() > 0` and the id is in the drain set, and `se` is only populated in that same `ai.SkillId() > 0` block — so a zero-value `se` can never reach the drain math with a spurious X.

### D4 — Effective stats: generalize the existing lazy loader (chosen) vs. a second loader

The handler already has a lazy, once-per-attack `effective_stats` fetch (`loadVenomStats`, lines 325-339) whose error behavior — log, return zero `RestModel` — is exactly the drain fail-safe the PRD requires: a zero `MaxHp` caps the heal at `0/2 = 0`, so a failed fetch yields no heal rather than an uncapped one.

**Chosen:** rename the closure and cache to `loadEffectiveStats` / `effectiveStats*`, generalize its error log line (it currently says "venom DPT will fall back to zero"), and hand it to both the venom path (via the existing `loadVenomStats` dep field, renamed to `loadEffectiveStats` in `damageInfoEntryDeps`) and the drain closure. One fetch per attack even when venom and drain both fire.

Rejected: a second, drain-specific lazy loader — two identical REST fetches in the same attack when a venom-applying drain build exists, and two copies of the same caching boilerplate.

The rename touches the `damageInfoEntryDeps` field and its uses in `processDamageInfoEntry` (lines 87, 123, 170) plus existing tests that construct the deps struct — mechanical, and it makes the field name honest now that it has two consumers.

### D5 — Cap math as a pure function

All arithmetic in one pure helper alongside `mpEaterAbsorbAmount`:

```go
// drainHealAmount computes the drain-family HP gain for one damaged
// monster: floor(totalDamage * x / 100), capped by the monster's max HP
// and by half the attacker's effective (buff-inclusive) max HP, then
// defensively clamped to int16 range for ChangeHP. Returns 0 for
// non-positive x, zero damage, or zero effectiveMaxHp (fail-safe when
// the effective-stats fetch failed).
func drainHealAmount(totalDamage uint32, x int16, monsterMaxHp uint32, effectiveMaxHp uint32) int16
```

Internals: compute `uint64(totalDamage) * uint64(x) / 100` (floor via integer division, matching Cosmic's `(int)` cast), take `min` with `uint64(monsterMaxHp)` and `uint64(effectiveMaxHp) / 2`, clamp to `math.MaxInt16`, return `int16`. Zero-guards make every failure mode (X=0, no damage, stats fetch failed) collapse to "return 0, emit nothing".

### D6 — Skill-set membership

A package-level helper:

```go
// isDrainSkill reports whether id is one of the four attack-side
// drain-family skills that heal the attacker from damage dealt.
func isDrainSkill(id skill3.Id) bool
```

implemented as a `switch` over `skill3.AssassinDrainId`, `skill3.MarauderEnergyDrainId`, `skill3.ThunderBreakerStage3EnergyDrainId`, `skill3.NightWalkerStage2VampireId` (all verified present in `libs/atlas-constants/skill/constants.go`). No numeric literals (DOM-21). A `switch` over four ids beats a package-level map: no init-order concerns, trivially inlinable, and the constants read as documentation.

Aran Combo Drain is deliberately absent (PRD non-goal); its TODO line stays.

### D7 — Per-monster emit via `ChangeHP` (no batching)

Each damaged monster produces one `cp.ChangeHP(s.Field(), s.CharacterId(), heal)` with a positive amount — the same command path the handler already uses for HP cost (line 305). Matches Cosmic (heal applied inside the per-monster loop) and PRD FR-2/NFR. Batching the capped per-monster heals into one emit was considered and rejected: it saves at most 3 Kafka messages on a 4-target Vampire while diverging from the reference behavior's observable intermediate states, and downstream current-HP clamping is per-command in atlas-character.

Downstream clamping to current max HP stays owned by atlas-character (PRD FR-5) — the handler does no current-HP arithmetic.

### D8 — Proc orchestration: `drainTryHeal` mirroring `mpEaterTryProc`

A small orchestrator keeps the closure in `processAttack` one line per passive:

```go
// drainTryHeal computes and emits the drain-family heal for one damaged
// monster. Called once per damaged monster after damage apply. Errors
// are logged and swallowed — never abort the attack pipeline.
func drainTryHeal(
    l logrus.FieldLogger,
    getMonster func(monsterId uint32) (monster.Model, error),
    changeHP func(f field.Model, characterId uint32, amount int16) error,
    loadEffectiveStats func() effective_stats.RestModel,
    x int16,
    skillId uint32,
    monsterId uint32,
    totalDamage uint32,
    f field.Model,
    characterId uint32,
)
```

Flow: fetch monster snapshot via `getMonster(monsterId)` (log at debug + skip on failure, per FR-4 — the registry snapshot survives a killing blow because damage application is an async emit); call `loadEffectiveStats()`; compute `drainHealAmount`; if `> 0`, log the debug line (caster, skill, monster, damage, X, heal — FR-6) and emit `changeHP`, logging emit errors at error level. All three collaborators are injected funcs — the same shape as `damageInfoEntryDeps.getMonster` — deliberately unlike `mpEaterTryProc`, which takes a concrete `*monster.Processor` and consequently has no flow-level tests. Production wiring passes `mp.GetById` and `cp.ChangeHP`.

The wiring inside `processAttack`'s `onDamageApplied` closure:

```go
onDamageApplied: func(monsterId uint32, totalDamage uint32) {
    if ai.AttackType() == packetmodel.AttackTypeMagic && ai.SkillId() > 0 {
        mpEaterTryProc(l, ctx, mp, c, monsterId, s.Field(), s.CharacterId())
    }
    if ai.SkillId() > 0 && isDrainSkill(skill3.Id(ai.SkillId())) {
        drainTryHeal(l, mp.GetById, cp.ChangeHP, loadEffectiveStats, se.X(), ai.SkillId(), monsterId, totalDamage, s.Field(), s.CharacterId())
    }
},
```

No attack-type gate on drain — the four skills span melee/ranged/energy attack types and the skill-id check is the authoritative filter.

Finally, the `// TODO increase HP from Energy Drain, Vampire, or Drain` line (currently line 409) is deleted; adjacent TODOs untouched (FR-7).

## 3. Data flow

1. Client attack packet → `processAttack` resolves character `c`, owned skill `sk`, effect `se` (existing).
2. Per `DamageInfo` entry: `processDamageInfoEntry` applies damage/status (existing), sums `di.Damages()`, and calls `onDamageApplied(monsterId, totalDamage)` for non-reflected entries.
3. Drain branch: monster snapshot (in-memory registry) → lazy effective-stats REST fetch (once per attack, shared with venom) → `drainHealAmount` → `ChangeHP` Kafka command (positive amount).
4. atlas-character consumes the command, clamps to current max HP, emits the stat-change the client renders (existing, untouched).

## 4. Error handling

| Failure | Behavior |
|---|---|
| Monster snapshot fetch fails | Debug log, skip heal for that monster (FR-4) |
| Effective-stats fetch fails | Error log inside shared loader, zero `RestModel` → heal computes to 0, nothing emitted (fail-safe, FR-4) |
| `X ≤ 0`, zero damage, heal computes to 0 | Silently no emit (FR-2) |
| `ChangeHP` emit fails | Error log, swallowed (FR-6) |

Nothing in the drain path can abort or delay damage application, the attack broadcast, or projectile consumption — the hook fires after damage apply and before broadcast, and every branch swallows its errors, identical to MP Eater's isolation policy.

## 5. Testing

All in `character_attack_common_*_test.go` style already present in the package (table-driven, Builder pattern, no `*_testhelpers.go`):

1. **`drainHealAmount` table tests** — X-percentage math against FR-3 spot values (e.g. Drain L30 X=45, Vampire L20 X=10), monster-max-HP cap, half-effective-max-HP cap, zero damage, X=0, effectiveMaxHp=0 (stats-failure fail-safe), floor semantics (e.g. 333 × 16 / 100 = 53), int16 defensive clamp with pathological inputs.
2. **`isDrainSkill`** — all four ids true; adjacent ids (e.g. Aran Combo Drain, MP Eater ids) false.
3. **`drainTryHeal` flow tests** — fake `changeHP` / `loadEffectiveStats` / monster fetch: emits positive amount on success; skips on monster-fetch error; skips on zero stats; per-monster invocation semantics (two monsters → two emits, individually capped).
4. **Hook-signature regression** — existing `processDamageInfoEntry` flow tests updated for the widened `onDamageApplied`, plus an assertion that the hook receives the summed damage total and does not fire for reflected/zero-damage entries.

Verification per CLAUDE.md: `go test -race ./...`, `go vet ./...`, `go build ./...` in atlas-channel; `tools/redis-key-guard.sh` from repo root. `go.mod` is not touched, so no bake is expected; if it is touched, `docker buildx bake atlas-channel`.

## 6. Out of scope (unchanged from PRD)

Energy-charge server validation, all other TODOs in the post-attack block (including Aran Combo Drain), packet changes, MP Eater changes, atlas-character changes.
