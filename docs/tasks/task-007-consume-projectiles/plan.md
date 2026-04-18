# Projectile Consumption on Attack — Implementation Plan

Last Updated: 2026-04-18

Supplements: `prd.md` (same directory). The PRD is the authoritative spec;
this plan turns it into sequenced phases, tasks, and acceptance gates.

## 1. Executive Summary

Ranged attacks in Atlas currently broadcast animation and apply damage without
decrementing ammunition — infinite ammo. This task wires consumption into the
`processAttack` pipeline so every eligible ranged attack emits one or more
`CONSUME` commands on `COMMAND_TOPIC_COMPARTMENT`, gated by skill, buffs, and
weapon class, sized by the skill's `BulletConsume` field and optionally doubled
by Shadow Partner on claws. As a supporting change, `atlas-inventory.ConsumeAsset`
is fixed to preserve rechargeable rows (throwing stars, bullets) at qty=0 rather
than deleting them, unblocking NPC-shop recharge.

Surface area: `atlas-channel`, `atlas-inventory`, `libs/atlas-constants/item`,
`libs/atlas-packet`. No DB migrations, no new topics, no tenant-config changes.

Effort: ~1 engineer-week. Risk: medium — touches the attack hot path and a
shared inventory primitive; mitigated by fire-and-forget emit order and unit
tests on the planner.

## 2. Current State Analysis

### 2.1 Attack pipeline (atlas-channel)

`services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go:23`
implements `processAttack`: damage application, monster status, and per-session
broadcast. No projectile consumption. Ranged-attack routing to this function
lives in `character_attack_ranged.go`.

### 2.2 Packet model (libs/atlas-packet)

`libs/atlas-packet/model/attack_info.go` exposes `BulletItemId()`,
`ProperBulletPosition()`, `CashBulletPosition()` as getters. The `javlin` flag
(line 60, read at line 153) is private and lacks a public getter — the channel
handler cannot read it today.

### 2.3 Inventory rechargeable handling (atlas-inventory)

`services/atlas-inventory/atlas.com/inventory/compartment/processor.go`
`ConsumeAsset` (≈ lines 794–839) deletes the asset row when post-consume
quantity would be ≤ 0, regardless of item type. Rechargeable items
(throwing stars, bullets) therefore disappear on last-shot, breaking NPC shop
recharge.

### 2.4 Shared helpers (libs/atlas-constants/item)

`IsBullet`, `IsThrowingStar`, `GetClassification`, `GetWeaponType` exist.
`ClassificationConsumableArrow = 206` is defined, but there is **no** `IsArrow`
or `IsRechargeable` helper.

### 2.5 Temp stats

`SOUL_ARROW` and `SHADOW_PARTNER` are both resident in the channel session
temporary-stat state. Read access patterns match existing buff gates elsewhere
in the handler tree.

## 3. Proposed Future State

- `processAttack` computes a consumption plan (resolved slots + per-slot counts)
  **before** broadcasting the attack, then emits fire-and-forget `CONSUME`
  commands **after** broadcast.
- Gates: `AttackType()==Ranged`, `!Javlin()`, skill not in
  `IsShootSkillNotConsumingBullet`, equipped weapon is a ranged class, and —
  for bow/crossbow — `SOUL_ARROW` is not active.
- Count: `se.BulletConsume()` if > 0 else 1; doubled on claw attacks while
  `SHADOW_PARTNER` is active.
- Slot resolution: prefer the client-supplied `ProperBulletPosition` when it
  matches classification and has enough quantity; else scan ascending slot
  order; emit one `CONSUME` per slot drawn. Shortfall logs a warning.
- Cash projectile positions are ignored for consumption — cosmetic only.
- `atlas-inventory.ConsumeAsset` retains the row at qty=0 when
  `item.IsRechargeable(templateId)` is true. All other items keep
  delete-at-zero.
- New helpers: `item.IsArrow`, `item.IsRechargeable`. New getter:
  `AttackInfo.Javlin() bool`.
- TODO comments at the two javelin sites and the passive-no-consume stub
  document deferred scope.

## 4. Implementation Phases

Phases are sequential. Tasks within a phase can parallelize where noted.

### Phase A — Shared helpers (S)

Goal: land the small, dependency-free additions so everything downstream can
import them.

1. **A1. `item.IsArrow`** — Add to `libs/atlas-constants/item`. Returns
   `GetClassification(itemId) == ClassificationConsumableArrow`. Mirrors
   `IsBullet`/`IsThrowingStar`. Unit test: known arrow, known non-arrow.
   Effort: S.
2. **A2. `item.IsRechargeable`** — Add alongside A1. Returns
   `IsThrowingStar(itemId) || IsBullet(itemId)`. Unit test: true for a star
   and a bullet, false for an arrow and a generic consumable. Effort: S.
3. **A3. `AttackInfo.Javlin()` getter** — Add to
   `libs/atlas-packet/model/attack_info.go` beside existing getters.
   Exports the private `javlin` field. Effort: S.

### Phase B — atlas-inventory rechargeable preservation (M)

Goal: fix the shared primitive so rechargeable rows survive at qty=0. This
phase is independently shippable and benefits every consume path, not just
this feature.

1. **B1. Modify `ConsumeAsset`.** In
   `services/atlas-inventory/atlas.com/inventory/compartment/processor.go`
   (≈ lines 794–839): when the post-consume quantity ≤ 0 AND
   `item.IsRechargeable(templateId)` is true, update the row to quantity 0
   instead of deleting. Preserve existing delete-at-zero for all other items.
   Effort: M. Depends on A2.
2. **B2. Unit test: rechargeable preserved.** Seed a throwing-star asset at
   qty=1, consume 1, assert row exists with qty=0. Repeat for bullets.
   Effort: S.
3. **B3. Regression test: non-rechargeable deletion.** Seed a generic
   consumable at qty=1, consume 1, assert row deleted. Effort: S.
4. **B4. Service build.** `go build ./...` in atlas-inventory passes.
   Verify via Docker build per CLAUDE.md. Effort: S.

### Phase C — Consumption planner (M)

Goal: stand up the planner as a testable Interface + Impl pair with no
wiring into `processAttack` yet.

1. **C1. New file `socket/handler/character_attack_projectile.go`** in
   atlas-channel. Define `Processor` interface + `ProcessorImpl`,
   constructed via `NewProcessor(l, ctx)` per project conventions.
   Effort: S.
2. **C2. Gate logic.** Implement `shouldConsume(ai, c, weapon, buffs, se) bool`:
   returns false if any of — attack type not ranged, `ai.Javlin()` true,
   skill in `IsShootSkillNotConsumingBullet`, weapon not a ranged class,
   bow/crossbow with `SOUL_ARROW` active. Logs at debug on skip.
   Effort: M. Depends on A3.
3. **C3. Count computation.** `computeCount(weaponType, se, buffs) int`:
   base = `se.BulletConsume()` if > 0 else 1; doubled on claw if
   `SHADOW_PARTNER` active. Pure function, unit-testable. Effort: S.
4. **C4. Classification mapping.** `requiredClassification(weaponType) Id`
   returning the classification constant per PRD §4.4 table. Unit-tested.
   Effort: S. Depends on A1.
5. **C5. Slot resolution.** `resolvePlan(consumable, classification,
   clientSlot, count) []SlotDraw`:
   - If `clientSlot` names a matching slot with enough quantity, one draw.
   - Else find the single slot with the lowest index meeting the full
     count.
   - Else multi-slot ascending draw until count reached.
   - Else consume all available across matching slots and flag shortfall.
   Returns a slice plus a shortfall indicator. Pure function over a
   slot snapshot. Effort: M.
6. **C6. Planner unit tests.** Exercise every PRD §10 slot-resolution
   acceptance row against a mocked compartment. Must include: single-slot
   hit, client-hint miss with fallback, multi-slot draw, total shortfall,
   no-matching-slot. Effort: M.

### Phase D — Wire planner into `processAttack` (M)

Goal: call the planner in `character_attack_common.go` and emit `CONSUME`
commands in the right order.

1. **D1. Compute plan pre-broadcast.** Inject the planner call into
   `processAttack` after the skill-effect load and before the
   per-session broadcast loop. Compute the plan; hold it in a local.
   Effort: S. Depends on C5.
2. **D2. Emit post-broadcast.** After the broadcast loop, iterate the
   plan and emit one `CONSUME` command per `SlotDraw` on
   `COMMAND_TOPIC_COMPARTMENT` with `CommandConsume` and
   `ConsumeCommandBody{ TransactionId, Slot }`. Reuse the existing
   compartment command producer used by
   `character_inventory_move.go`. Tenant-aware via `tenant.MustFromContext`.
   Effort: M.
3. **D3. Logging.** Debug on designed skips (Soul Arrow, non-consuming,
   javlin, non-ranged weapon). Warn on no-match and partial shortfall with
   `characterId`, `weaponItemId`, `skillId`, `required`, `available`.
   Error on Kafka emit failure; continue with remaining commands.
   Effort: S.
4. **D4. Failure isolation.** Confirm planner panics / errors are caught
   before broadcast; broadcast failures do not cancel the emit. Add a test
   or structured log review. Effort: S.

### Phase E — TODO markers and cross-references (S)

1. **E1. Packet-side TODO.** At
   `libs/atlas-packet/model/attack_info.go:153` (the
   `m.javlin && !IsShootSkillNotConsumingBullet` block), add a TODO noting
   the javlin flag's gameplay semantics are deferred and pointing at the
   planner bailout. Effort: S.
2. **E2. Planner-side TODO.** At the javlin bailout in the planner, add a
   matching TODO pointing back at the packet site. Effort: S.
3. **E3. Passive no-consume TODO.** In the planner, add a TODO for
   Mortal Blow / Expert Marksmanship / Claw Mastery class-passive
   roll-to-preserve mechanics — out of scope for v1. Effort: S.

### Phase F — Verification (M)

1. **F1. Unit-test suite passes.** All Phase C tests + Phase B tests green.
   Effort: S.
2. **F2. Multi-service build.** `go build ./...` across atlas-channel,
   atlas-inventory, libs/atlas-constants, libs/atlas-packet. Docker build
   check on atlas-channel and atlas-inventory per CLAUDE.md's shared-library
   rule. Effort: S.
3. **F3. Manual QA matrix.** Run through PRD §10 acceptance criteria
   against a live channel. Document results in `RELEASE_NOTES.md` (create
   alongside if not present). Effort: M.
4. **F4. Docs update.** `/dev-docs-update` sweep after merge — update
   `docs/TODO.md` to strike projectile consumption from the attack-effect
   backlog and reflect the deferred items (javlin, passives). Effort: S.

## 5. Risk Assessment and Mitigation

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| `ConsumeAsset` change breaks non-projectile consume paths | Low | High | Gate on `IsRechargeable(templateId)`; regression test B3; review all call sites during B1. |
| Consumption fires before broadcast and errors cancel the attack visibly | Low | Medium | PRD-specified order: plan before, emit after. D1/D2 enforce this; D4 verifies. |
| Kafka emit latency dragged onto attack hot path | Med | Low | Fire-and-forget post-broadcast; no awaiting acks. Keep planner in-memory only. |
| Client-supplied slot points at wrong/empty slot | Med | Low | Treat as a hint; verify classification before consuming; multi-slot fallback. |
| Shadow Partner / Soul Arrow buff state stale | Low | Low | Read buffs from the same session state existing handlers use; no new caching. |
| TODO comments rot | Med | Low | E1/E2 cross-reference each other so touching one surfaces the other. |
| Planner complexity balloons | Low | Medium | Keep `resolvePlan` pure and unit-testable; defer passive no-consume to TODO per PRD §9. |

## 6. Success Metrics

- PRD §10 acceptance checklist fully green.
- Zero regression in non-projectile consume paths (B3 passes; no ticket
  inbound within one release cycle).
- No measurable p95 attack-handler latency increase on the channel hot
  path (spot-check via existing channel logs; no formal SLO in place).
- Server warn-rate on the new "no matching projectile" log is effectively
  zero in production — persistent warnings indicate client desync or
  tampering, which is itself a useful signal.

## 7. Required Resources and Dependencies

- Go toolchain + Docker (shared-library builds per CLAUDE.md).
- atlas-channel, atlas-inventory running locally (or in a staging
  channel) for F3 manual QA.
- Read access to PRD §4.4 classification constants in
  `libs/atlas-constants/item`.
- No new third-party packages, no new infra.

## 8. Timeline Estimates

- Phase A: 0.25d
- Phase B: 0.5d (incl. tests + Docker build)
- Phase C: 1.5d (planner + unit tests dominate)
- Phase D: 0.5d
- Phase E: 0.25d
- Phase F: 1d (manual QA is the long pole)

Total: ~4 engineer-days of focused work, ~1 week wall-clock allowing for
review cycles.
