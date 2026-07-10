# Sacrifice Self-HP Cost (Dragon Knight 1311005) — Design

Version: v1
Status: Approved for planning
Created: 2026-07-10
PRD: `docs/tasks/task-148-sacrifice-hp-cost/prd.md`

---

## 1. Problem Recap

Sacrifice's attack pipeline works end-to-end in atlas-channel, but the skill's defining self-HP cost is a TODO at `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go:405`. Cosmic reference behavior (verified from source, cited in the PRD): after a melee Sacrifice that hit, the caster loses `firstDamageLine × X / 100` HP, clamped so the caster keeps at least 1 HP.

All interview questions were resolved in the PRD; this design covers architecture, placement, alternatives, and the exact contracts.

## 2. Verified Code Facts (source-checked in this worktree)

| Fact | Location |
|---|---|
| `skill.DragonKnightSacrificeId` constant exists | `libs/atlas-constants/skill/constants.go` (Skill entry at :396, id map at :2448) |
| `effect.Model.X() int16` | `services/atlas-channel/atlas.com/channel/data/skill/effect/model.go:144` |
| `DamageInfo.Damages() []uint32` | `libs/atlas-packet/model/damage_info.go:80` |
| `character.Model.Hp() uint16` | `services/atlas-channel/atlas.com/channel/character/model.go:131` |
| `ChangeHP(f field.Model, characterId uint32, amount int16) error` — Kafka command, fire-and-forget | `services/atlas-channel/atlas.com/channel/character/processor.go:271` |
| MP Eater precedent: pure helpers + orchestration fn, errors swallowed | `character_attack_common.go:182-264` |
| Existing pure-helper test file convention | `socket/handler/character_attack_mp_eater_test.go` |
| Generic `hpCon`/`mpCon` cast-cost block (must stay untouched) | `character_attack_common.go:303-310` |

## 3. Architecture

### 3.1 Shape: pure helper + thin orchestration at the TODO site

Follow the MP Eater pattern exactly — it is the one existing post-attack side effect in this file and sets the local idiom:

1. **Pure helper** `sacrificeHpCost(firstLine uint32, x int16, currentHp uint16) uint16` — all arithmetic and clamping, no I/O, unit-testable with plain table tests.
2. **First-line extraction helper** (or inline guard) that safely pulls `ai.DamageInfo()[0].Damages()[0]`, returning 0 when the slices are empty.
3. **Orchestration block** in `processAttack`, placed where the TODO sits today (the post-broadcast side-effects block, alongside the projectile emit): gate on skill id, compute, log, emit `ChangeHP`, swallow errors.

### 3.2 Placement: post-broadcast TODO block (chosen)

Alternatives considered for where the cost applies inside `processAttack`:

- **A (chosen): at the TODO site, after broadcast + projectile emit.** Matches the file's established semantics — "the projectile is expended the moment the server accepts the attack, regardless of broadcast success" (comment at :393-395). The cost is a fire-and-forget Kafka command either way; ordering relative to the broadcast has no client-visible effect because the stat update rides its own event/packet flow from atlas-character. Keeps all post-attack skill side effects grouped where the remaining TODOs live, which is where future readers (and the neighboring TODO implementations) will look.
- **B: immediately after the per-monster damage loop, before broadcast.** Functionally equivalent (same fire-and-forget emit), but splits post-attack side effects into two locations and buys nothing — `ChangeHP` cannot fail synchronously in a way the broadcast should observe.
- **C: inside the `onDamageApplied` per-monster hook.** Rejected. That hook fires per damaged monster after monster-side status apply and is gated on the monster fetch/damage path succeeding. Sacrifice's cost is a caster-side effect keyed to the *packet's* first damage line, not to per-monster application success; routing it through the hook would couple it to monster registry availability and fire-once semantics would need extra state.

### 3.3 Trigger and data flow

```
processAttack (skillId > 0 branch already fetched: c, sk, se)
  ... damage loop, broadcast, projectile emit ...
  if skill3.Id(ai.SkillId()) == skill.DragonKnightSacrificeId:
      firstLine := first damage line of first entry (0 if absent)
      cost := sacrificeHpCost(firstLine, se.X(), c.Hp())
      if cost > 0:
          Debugf(caster, skill, firstLine, X, cost)
          err := cp.ChangeHP(s.Field(), s.CharacterId(), -int16(cost))
          if err != nil: Errorf + swallow
```

No new fetches: `c` (with `Hp()`), `se` (with `X()`), and `cp` already exist in scope (FR under §8 of the PRD). Non-Sacrifice attacks pay one integer comparison.

The generic `hpCon`/`mpCon` block at :303-310 is untouched — Sacrifice has no per-skill dispatcher entry, so its flat cast cost continues to apply independently (FR-9).

## 4. Helper Contracts

### 4.1 `sacrificeHpCost`

```go
// sacrificeHpCost computes the self-HP cost of Dragon Knight Sacrifice:
// firstLine × x / 100 (truncating integer division, Cosmic parity),
// clamped so the caster is left with at least 1 HP. Returns 0 when the
// first line is 0 (miss), x is non-positive, or currentHp <= 1.
func sacrificeHpCost(firstLine uint32, x int16, currentHp uint16) uint16
```

Rules, in order:
1. `firstLine == 0 || x <= 0 || currentHp <= 1` → 0.
2. `cost := uint64(firstLine) * uint64(x) / 100` — widen to `uint64` before multiplying (same overflow discipline as `mpEaterAbsorbAmount` at :199); truncating division matches Cosmic's Java `int` math (FR-3).
3. Survival clamp: `if cost >= uint64(currentHp) { cost = uint64(currentHp) - 1 }` (FR-4).
4. Defensive narrowing guard: `if cost > math.MaxInt16 { cost = math.MaxInt16 }`. On supported versions max HP is ≤ 30000 so the FR-4 clamp already bounds `cost < 32767`, but `Hp()` is `uint16` (theoretical max 65535) and the call site negates into `int16`; this one-line cap makes the helper safe by construction instead of by data assumption. Documented in the helper comment.
5. Return `uint16(cost)`; call site emits `-int16(cost)`.

### 4.2 First-line extraction

```go
// sacrificeFirstDamageLine returns the first damage line of the first
// damage entry, or 0 when the attack has no entries or the first entry
// has no lines. Sacrifice's cost basis is only ever this line (FR-2).
func sacrificeFirstDamageLine(ai packetmodel.AttackInfo) uint32
```

Guards `len(ai.DamageInfo()) > 0 && len(ai.DamageInfo()[0].Damages()) > 0`. Kept as a named helper (rather than inline) so the FR-2 "first line only, never sum" decision is test-pinned — a future edit that sums lines breaks a test, not just parity.

## 5. Error Handling

- `ChangeHP` error → `Errorf` with caster id and skill id, then swallowed; broadcast and projectile consumption have already run and are unaffected (FR-6, MP Eater convention at :261-263).
- No error paths in the helpers — they are total functions returning 0 for every degenerate input (miss, empty slices, `X ≤ 0`, `Hp ≤ 1`).
- No session destroy / attack abort is ever triggered by this feature.

## 6. Observability

- `Debugf` on applied cost: caster id, skill id, first line, `X`, clamped cost — mirroring the MP Eater proc log at :258-259 (PRD §8).
- `Errorf` on emit failure only. Nothing logged for non-Sacrifice attacks or zero-cost outcomes.

## 7. Testing

New file `socket/handler/character_attack_sacrifice_test.go` (naming mirrors `character_attack_mp_eater_test.go`), plain table tests against the two pure helpers — no socket/session scaffolding, no `*_testhelpers.go` (FR-7):

**`sacrificeHpCost`:**
- Normal computation: `firstLine=1000, x=30, hp=5000` → 300.
- Truncating division: `firstLine=99, x=30, hp=5000` → 29 (not 29.7 rounded).
- `x = 0` and `x < 0` → 0.
- `firstLine = 0` (miss) → 0.
- Clamp to `hp−1`: `firstLine=100000, x=100, hp=500` → 499.
- Exact-kill boundary: `cost == hp` → `hp−1`.
- `hp = 1` and `hp = 0` → 0.
- Narrowing guard: `hp=65535, firstLine=100000, x=100` → 32767 (MaxInt16), proving `-int16(cost)` cannot overflow.
- Large-line overflow: `firstLine=4294967295, x=100` computes without wrap (uint64 widening).

**`sacrificeFirstDamageLine`:**
- No damage entries → 0.
- First entry with empty `Damages()` → 0.
- Multi-line first entry → returns line[0] only.
- Multi-target attack → ignores second entry entirely (FR-2 pin). Builds `DamageInfo` via the existing builder methods (`SetMonsterId` etc.) on `libs/atlas-packet/model`.

The orchestration block itself (gate + emit + swallow) follows the same untested-thin-glue precedent as `mpEaterTryProc`'s call site; manual validation on v83 and v95 tenants covers the wiring (PRD acceptance criteria).

## 8. Multi-Tenancy / Versioning

Version-agnostic by construction: `X` resolves per-tenant from atlas-data through the effect model already fetched at :292. No `MajorVersion()` branches, no template/seed changes, no new config (PRD §8, memory rule DOM-25 not implicated — no client wire values are produced here; the stat update rides the existing atlas-character event flow).

## 9. Scope of Change

- `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go` — two helpers + one gated block replacing the line-405 TODO (other TODOs untouched, FR-8).
- `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_sacrifice_test.go` — new.
- No other services, no API/data-model changes, no `go.mod` changes (so `docker buildx bake atlas-channel` is not triggered by the CLAUDE.md rule, though running it is harmless).

## 10. Risks & Accepted Tradeoffs

- **Stale `c.Hp()` vs concurrent damage:** the clamp reads the character model fetched at attack start; a monster hit landing between fetch and emit could theoretically make the −HP command lethal at atlas-character. Accepted per interview decision #3 — Cosmic has the identical read-then-write shape, and atlas-character's own HP handling floors at death semantics it already owns.
- **Fire-and-forget emit:** a dropped Kafka command means one free Sacrifice, consistent with how `hpCon`/`mpCon` cast costs already behave (`_ = cp.ChangeHP(...)` at :305). Not worth a stronger guarantee for a per-attack cosmetic-adjacent cost.
- **First-entry assumption:** Sacrifice is single-target in all supported versions; if a client ever sends multiple entries, we deliberately still charge only entry[0] line[0] (FR-2, interview decision #2 — "never more").
