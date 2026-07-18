# Shadow Stars (Night Lord 4121006) — Design

Status: Draft
Created: 2026-07-10
PRD: `docs/tasks/task-158-shadow-stars/prd.md`

---

## 1. Summary

Shadow Stars today applies a `SHADOW_CLAW` buff whose value is `0`, drops the
client's chosen star id, and never suppresses per-attack star consumption. This
design makes the skill correct end to end with three coordinated changes, all in
`services/atlas-channel` plus one getter in `libs/atlas-packet`:

1. **Expose** the decoded star id (`SkillUsageInfo.SpiritJavelinItemId()`).
2. **Inject** that id into the `SHADOW_CLAW` statup amount at cast, by rewriting
   the statup slice in the channel skill orchestrator before `buff.Apply` — the
   same technique `mount.go` already uses to inject a vehicle id into
   `MONSTER_RIDING`.
3. **Gate** per-attack consumption: skip projectile consume for a claw when
   `SHADOW_CLAW` is active, mirroring the existing `SOUL_ARROW` skip for
   bow/crossbow.
4. **Charge** the one-time cast cost: consume `bulletCount` of the chosen star at
   cast, reusing the compartment reserve→consume machinery the projectile path
   already uses.

`atlas-data`'s `reader.go:298` continues to emit the `SHADOW_CLAW` placeholder
`0`; the channel layer overwrites it. No Kafka payload change, no schema change,
no `reader.go` change.

## 2. Architecture & data flow

```
client OnUserSkillUseRequest  (skillId=4121006, spiritJavelinItemId=<star>)
   │
   ▼
SkillUsageInfo.Decode  ── reads spiritJavelinItemId (already wired)
   │   FR-1: add SpiritJavelinItemId() getter
   ▼
UseSkill (skill/handler/common.go)
   │
   ├─ (A) Shadow-Stars pre-flight  ── NEW, runs FIRST for skillId==4121006
   │        load caster inventory (seam)
   │        validate star: throwing-star classification + owned   (FR-5)
   │        resolve consume draws for bulletCount of the star      (FR-4)
   │        on validation failure → warn log, ABORT cast (return nil)
   │
   ├─ HP / MP consume, cooldown           (unchanged; skipped if aborted above)
   ├─ mount short-circuit                  (unchanged)
   ├─ buff.Apply with statups             ── SHADOW_CLAW amount rewritten to star id (FR-2)
   ├─ emit star-consume reserve→consume    ── NEW (FR-4)
   ├─ applyToMobs / per-skill dispatcher   (unchanged)
   ▼
buff projection → caster + observers render the chosen star (existing infra)

character_attack_projectile.go Plan()   ── per ranged attack
   │  FR-3: claw + active SHADOW_CLAW → return (nil,false)  (skip consume)
```

The remote-render requirement (observers see the correct star) needs **no new
code**: every buff applied via `buff.Apply` is already projected to observers, so
carrying the real star id in the statup amount is sufficient — the same reason
`MONSTER_RIDING`'s vehicle id renders remotely.

## 3. Components & changes

### 3.1 `libs/atlas-packet` — `SkillUsageInfo.SpiritJavelinItemId()` (FR-1)

Add the missing getter next to the existing getters in
`model/skill_usage_info.go`. No wire change; the field is already decoded at
line 33 and settable via `SetSpiritJavelinItemId`.

```go
func (m *SkillUsageInfo) SpiritJavelinItemId() uint32 {
    return m.spiritJavelinItemId
}
```

### 3.2 `atlas-channel` effect model — `BulletCount()` getter

`data/skill/effect/model.go` holds `bulletCount uint16` but exposes no getter
(only `BulletConsume()` exists). Add:

```go
func (m Model) BulletCount() uint16 {
    return m.bulletCount
}
```

This is the WZ-defined one-time cast cost (200 in reference data). The REST
transport (`rest.go`) already carries `BulletCount`, so no plumbing change.

### 3.3 `atlas-channel` skill orchestrator — `skill/handler/common.go`

Three cohesive additions, kept in small focused helpers so `UseSkill` stays
readable and the pure logic is unit-testable without Kafka/REST.

**(a) Star validation + consume-plan resolution (pure + one seam).**

A new pure function resolves the draw plan for a *known* item id (simpler than
the projectile `resolvePlan`, which filters by classification and honors a
client slot hint — neither is needed here since the star id is already known and
validated):

```go
// resolveStarConsume draws `count` of exactly starItemId across ascending
// consumable slots. Returns the per-slot draws and the total available.
// available < count signals a shortfall (caller consumes what's present).
func resolveStarConsume(assets []asset.Model, starItemId uint32, count int) (draws []StarDraw, available int)
```

Validation (FR-5) is a pure predicate over the caster's consumable assets:

```go
// validateShadowStar returns true iff starItemId is a throwing-star
// classification AND present (qty>0) in the caster's consumable inventory.
func validateShadowStar(assets []asset.Model, starItemId uint32) bool {
    if item.GetClassification(item.Id(starItemId)) != item.ClassificationConsumableThrowingStar {
        return false
    }
    // ... any matching asset with quantity > 0
}
```

Classification derives purely from the item id via `atlas-constants/item`
(`GetClassification` / `IsThrowingStar`) — no WZ round-trip, no data-service
call.

Caster-inventory load goes through a **new package-level seam**
(`loadCasterInventoryFunc`) mirroring the existing `loadCasterFunc` seam, so
tests inject a deterministic inventory. The production impl calls
`cp.GetById(cp.InventoryDecorator)(characterId)` (the same decorated load the
existing item-consume block at `common.go:81` uses).

**(b) SHADOW_CLAW statup rewrite (FR-2).**

Mirror `mount.go:tamedMountStatups`: return a copy of the effect's statups with
the `SHADOW_CLAW` entry's amount replaced by the star id.

```go
// rewriteShadowClawStatups returns e.StatUps() with the SHADOW_CLAW amount
// set to starItemId. Non-SHADOW_CLAW statups pass through unchanged.
func rewriteShadowClawStatups(statups []statup.Model, starItemId uint32) []statup.Model
```

The rewritten slice is passed to the existing `buff.NewProcessor(...).Apply(...)`
call at `common.go:108` for the Shadow-Stars case only.

**(c) Cast-cost consume emit (FR-4).**

Mirror the projectile `Emit` pattern: for each resolved `StarDraw`, register a
one-time handler on the compartment status topic that consumes the reservation,
then `RequestReserve` the slot. Reuses `compartment.Processor.RequestReserve` /
`Consume` — the identical machinery in
`character_attack_projectile.go:150-184`.

**Wiring into `UseSkill`.** For `skillId == NightLordShadowStarsId` only:

- Run pre-flight (load inventory, validate, resolve plan) at the **top** of
  `UseSkill`, before HP/MP/cooldown. On validation failure: warn log with
  `characterId`, `skillId`, offending `spiritJavelinItemId`, then `return nil` —
  the cast is fully aborted (no MP/cooldown spent, no buff, no consume). This is
  the cleanest anti-cheat posture and guarantees "no unowned/mistyped id reaches
  the client or the consume path" (FR-5).
- On success, let the normal flow run, but (i) pass the rewritten statups to
  `buff.Apply`, and (ii) after `buff.Apply`, emit the star-consume.

All other skills are completely unaffected — the pre-flight is gated on the
skill id and returns immediately for everything else.

### 3.4 `atlas-channel` projectile gate — `socket/handler/character_attack_projectile.go` (FR-3)

Add one carve-out immediately after the `SOUL_ARROW` check at line 107, scoped
to claws:

```go
if weaponType == item.WeaponTypeClaw && hasBuff(buffs, ts.TemporaryStatTypeShadowClaw) {
    p.l.WithField("characterId", c.Id()).WithField("skillId", ai.SkillId()).
        Debugf("Skipping projectile consumption: Shadow Stars active.")
    return nil, false
}
```

`hasBuff` already ignores expired buffs, so this is inactive-safe: a claw attack
with no live `SHADOW_CLAW` falls through to normal consumption (the FR-3
regression requirement). Bow/crossbow/gun are unaffected (guarded on
`WeaponTypeClaw`). The Shadow-Partner `computeCount` doubling at line 232 becomes
dead for Shadow-Stars attacks because we return before reaching it — resolving
PRD Open Question 5 with no extra code.

## 4. Resolved open questions (PRD §9)

| # | Question | Decision | Rationale |
|---|----------|----------|-----------|
| 1 | Cast cost in P7 vs gated behind P8 | **Ship self-contained consume in P7** | No generic P8 consume framework exists in-repo; a self-contained consume avoids ever shipping the free-stars exploit. If a generic path lands later, this consume can be re-homed. |
| 2 | Validate vs trust-client | **Validate (classification + ownership)** | The id now drives both a client-visible throw and a bulk consume; an unvalidated id is an anti-cheat hole. Validation is offline/cheap and runs once per cast. |
| 3 | Injection mechanism | **Rewrite the statup amount in `common.go` before `buff.Apply`** | Exact precedent in `mount.go` (vehicle id → `MONSTER_RIDING`). Avoids a multi-tenant/versioned Kafka payload change; the channel already holds the star id at cast time. |
| 4 | Shortfall on cast cost | **Consume what's available + warn log** | Matches the existing projectile shortfall posture (`character_attack_projectile.go:139-146`); the client already gates on owning ≥ the required count. |
| 5 | Shadow Partner interaction | **Moot** | FR-3 returns before `computeCount`, so the claw doubling never applies to Shadow-Stars attacks. No extra handling. |

**Added design decision — validation ordering.** The star pre-flight runs at the
very top of `UseSkill` (before HP/MP/cooldown), so a bogus star aborts the whole
cast without burning MP or cooldown or applying a zero-value buff. This is
stricter than the "cast permitted, defense-in-depth only" posture of the generic
item-consume block, which is appropriate because here the id drives a real
consume.

## 5. Testing strategy

Maps 1:1 to PRD §10 acceptance criteria. Uses the project Builder pattern
(`SkillUsageInfoBuilder`, character/asset builders); no `*_testhelpers.go`.

- **Decode getter** (`libs/atlas-packet`): byte-fixture decode of a 4121006
  `SkillUsageInfo` asserts `SpiritJavelinItemId()` returns the wire value.
- **Statup rewrite** (`skill/handler`): `rewriteShadowClawStatups` maps
  `SHADOW_CLAW` amount → star id and passes other statups through unchanged.
- **Validation** (`skill/handler`): `validateShadowStar` rejects a
  non-throwing-star id and an unowned star id; accepts an owned star.
- **Consume resolution** (`skill/handler`): `resolveStarConsume` targets the
  chosen item id across slots and totals `bulletCount`; shortfall path returns
  `available < count`.
- **Projectile gate** (`socket/handler`): claw + active `SHADOW_CLAW` → `Plan`
  returns `(nil,false)`; claw with **no** `SHADOW_CLAW` still returns a
  consuming plan (regression); bow + `SOUL_ARROW` unchanged.
- **Orchestration** (`skill/handler`): via the `loadCasterInventoryFunc` seam,
  an invalid star aborts the cast (no buff apply, no consume, warn logged); a
  valid star yields a buff whose `SHADOW_CLAW` amount equals the star id and a
  consume plan for `bulletCount`.

## 6. Impact, risks, non-goals

**Modules touched:** `libs/atlas-packet` (getter only) and `services/atlas-channel`.
Both `go.mod`s are exercised, so per CLAUDE.md: `go test -race`, `go vet`,
`go build` clean in each; `docker buildx bake atlas-channel`; `redis-key-guard.sh`
clean from repo root. `atlas-data` is untouched.

**Risks:**
- *Inventory staleness* — the pre-flight loads inventory once at cast; a race
  against a concurrent inventory change is possible but bounded by the same
  reserve→consume atomicity the projectile path already relies on (the
  reservation fails cleanly if the slot no longer holds the item).
- *Double-package `handler`* — `resolvePlan` in `socket/handler` is not reused
  (different package, and star-specific draw is simpler); the new
  `resolveStarConsume` is a small focused function, not a duplicated framework
  (honors the PRD "no generic batch-consume" non-goal).

**Version stability:** the decode gate and `SHADOW_CLAW` stat live in
version-agnostic shared code (`skill_usage_info.go`, `atlas-constants`), so no
per-version opcode work is anticipated (NFR confirmed).

**Non-goals (unchanged from PRD):** no change to other buffs, no generic
batch-consume framework, no Night Walker changes, no `reader.go` functional
change beyond the retained `0` placeholder.
