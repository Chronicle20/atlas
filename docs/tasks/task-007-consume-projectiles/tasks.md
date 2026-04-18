# Projectile Consumption on Attack — Task Checklist

Last Updated: 2026-04-18 (implementation complete; F3 manual QA + F4 docs sweep remain)

Status legend: `[ ]` todo · `[~]` in progress · `[x]` done · `[-]` skipped.

## Phase A — Shared helpers

- [x] **A1.** `item.IsArrow(itemId Id) bool` in `libs/atlas-constants/item`,
      mirroring `IsBullet`/`IsThrowingStar`. Unit test: arrow true, non-arrow
      false. _Effort: S._
- [x] **A2.** `item.IsRechargeable(itemId Id) bool` returning
      `IsThrowingStar(itemId) || IsBullet(itemId)`. Unit test: star + bullet
      true; arrow + generic consumable false. _Effort: S._
- [x] **A3.** `AttackInfo.Javlin() bool` getter in
      `libs/atlas-packet/model/attack_info.go`. _Effort: S._

## Phase B — atlas-inventory rechargeable preservation

- [x] **B1.** Gate delete-at-zero in `ConsumeAsset`
      (`services/atlas-inventory/atlas.com/inventory/compartment/processor.go`
      ≈ 794–839) on `!item.IsRechargeable(templateId)`; rechargeable rows
      update to qty=0 instead. _Effort: M._ Depends on A2.
- [x] **B2.** Unit test: throwing-star qty=1 → consume 1 → row exists with
      qty=0. Same for bullets. _Effort: S._
- [x] **B3.** Regression test: generic consumable qty=1 → consume 1 → row
      deleted. _Effort: S._
- [x] **B4.** atlas-inventory `go build ./...` passes; Docker build passes
      (shared-lib rule per CLAUDE.md). _Effort: S._

## Phase C — Consumption planner (atlas-channel)

- [x] **C1.** New file `socket/handler/character_attack_projectile.go`:
      `Processor` interface + `ProcessorImpl`, `NewProcessor(l, ctx)`.
      _Effort: S._
- [x] **C2.** `shouldConsume(ai, c, weapon, buffs, se) bool` covering:
      attack type ranged, `!ai.Javlin()`, skill not in
      `IsShootSkillNotConsumingBullet`, weapon is bow/crossbow/claw/gun,
      bow/crossbow without `SOUL_ARROW`. Debug-log skips. _Effort: M._
      Depends on A3.
- [x] **C3.** `computeCount(weaponType, se, buffs) int`: base
      `BulletConsume` else 1; x2 on claw with `SHADOW_PARTNER`. Pure, unit
      tested. _Effort: S._
- [x] **C4.** `requiredClassification(weaponType) Id` per PRD §4.4 table.
      Unit tested. _Effort: S._ Depends on A1.
- [x] **C5.** `resolvePlan(consumable, classification, clientSlot, count)`
      returns `[]SlotDraw` + shortfall flag. Priority: client-slot
      match-with-qty, then lowest-index single-slot match, then multi-slot
      ascending draw, then consume-all-available + shortfall. _Effort: M._
- [x] **C6.** Planner unit tests covering every PRD §10 slot-resolution
      row: single-slot, client-hint miss with fallback, multi-slot draw,
      total shortfall, no matching slot. _Effort: M._

## Phase D — Wire planner into `processAttack`

- [x] **D1.** In `character_attack_common.go` `processAttack`, compute the
      consumption plan after skill-effect load and before the broadcast
      loop. _Effort: S._ Depends on C5.
- [x] **D2.** After the broadcast loop, emit one `CONSUME` command per
      `SlotDraw` on `COMMAND_TOPIC_COMPARTMENT` via the existing
      compartment command producer used by `character_inventory_move.go`;
      fire-and-forget; tenant-aware. _Effort: M._
- [x] **D3.** Logging: debug on designed skips, warn on no-match /
      shortfall with `characterId`/`weaponItemId`/`skillId`/`required`/
      `available`, error on emit failure. _Effort: S._
- [x] **D4.** Verify planner failures do not cancel broadcast and
      broadcast failures do not cancel emit. _Effort: S._

## Phase E — Deferred-scope TODO markers

- [x] **E1.** TODO at `libs/atlas-packet/model/attack_info.go:153`
      documenting deferred javlin semantics and pointing at the planner
      bailout. _Effort: S._
- [x] **E2.** TODO at the planner's javlin bailout gate pointing back at
      the packet site. _Effort: S._
- [x] **E3.** TODO in the planner for passive no-consume mechanics —
      Mortal Blow, Expert Marksmanship, Claw Mastery roll-to-preserve.
      _Effort: S._

## Phase F — Verification

- [x] **F1.** All unit tests green across Phases B and C. _Effort: S._
- [x] **F2.** `go build ./...` across atlas-channel, atlas-inventory,
      libs/atlas-constants, libs/atlas-packet. Docker builds green.
      _Effort: S._
- [ ] **F3.** Manual QA matrix — run PRD §10 acceptance criteria against
      a live channel. Check specifically:
      - [ ] Basic bow/crossbow/claw/gun attack: −1 per shot.
      - [ ] Multi-projectile skill: decrements by `BulletConsume`.
      - [ ] Triple Throw w/o Shadow Partner: −3 per attack.
      - [ ] Triple Throw + Shadow Partner: −6 per attack.
      - [ ] Claw basic + Shadow Partner: −2 per shot.
      - [ ] Bow + Soul Arrow: no decrement.
      - [ ] Crossbow + Soul Arrow: no decrement.
      - [ ] `IsShootSkillNotConsumingBullet` skill: no decrement.
      - [ ] Cash cover equipped: real projectile decrements, cover does not.
      - [ ] Client hint points to insufficient slot, other slot suffices:
            single command to the other slot, no warning.
      - [ ] Required > any single slot, combined sufficient: multi-command
            emit, no warning.
      - [ ] Required > combined total: partial consume, warning logged.
      - [ ] Last throwing star fired: row retained at qty=0.
      - [ ] Last bullet fired: row retained at qty=0.
      - [ ] Non-rechargeable at qty=1 → 0: row deleted (regression).
      - [ ] Melee/magic/energy attacks: no consumption attempted.
      - [ ] `AttackInfo.Javlin()==true`: no consumption, debug log.
      - [ ] `IsArrow` / `IsRechargeable` helpers behave per spec.
      - [ ] TODO markers E1–E3 present in code.
      _Effort: M._
- [ ] **F4.** Run `/dev-docs-update` post-merge; strike projectile
      consumption from `docs/TODO.md` "Character Attack Effects"; record
      deferred scope (javlin, passive no-consume). _Effort: S._
