# Projectile Consumption on Attack — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-04-18
---

## 1. Overview

Today, when a player performs a ranged attack in Atlas, the client sends an attack packet
containing the slot of the projectile being fired (throwing star, bullet, arrow, or crossbow
bolt). The channel service broadcasts the attack animation and applies damage, but never
instructs the inventory to decrement the ammunition. As a result, ranged attackers have
effectively infinite ammo — a gameplay bug that trivializes economies (NPC shops, drops,
cash shop covers) built around ammunition turnover.

This feature wires consumption into the attack pipeline. On every eligible ranged attack,
the channel service locates the appropriate projectile in the character's consumable
inventory and emits a `CONSUME` command (or commands, for multi-projectile skills and
multi-slot draws) to the compartment system. The per-attack count is driven by the
skill's `BulletConsume` field when > 0, else 1, doubled for claw attacks when Shadow
Partner is active. Cash projectiles are treated as cosmetic covers only — the real
underlying projectile is always consumed. Certain skills
(`IsShootSkillNotConsumingBullet`, Soul Arrow buff) bypass consumption by design.
Throwing stars and bullets are rechargeable: when the final unit is consumed, the
inventory row is retained at qty=0 rather than deleted, so the player can refill it
at an NPC shop.

The feature touches `atlas-channel` (new consumption logic), `atlas-inventory` (a bug
fix so rechargeable items persist at qty=0 instead of being deleted), and
`libs/atlas-constants/item` (a new `IsArrow` helper). No new Kafka topics, no DB
migrations, no tenant-config changes.

## 2. Goals

Primary goals:
- Consume projectiles on ranged attacks, driven by the skill's `BulletConsume` field
  when > 0 (e.g., Strafe = 4 arrows, Triple Throw = 3 stars), else 1 per attack.
- Cover all four ranged weapon families: claw (throwing stars), gun (bullets), bow
  (arrows), crossbow (crossbow bolts).
- Double total consumption (stars) while Shadow Partner buff is active on a claw
  attack — including when stacked with multi-projectile skills.
- Skip consumption while Soul Arrow buff is active (bow/crossbow).
- Skip consumption for skills listed in `IsShootSkillNotConsumingBullet`.
- Treat cash shop projectiles as cosmetic covers — the underlying non-cash item is
  still consumed.
- Preserve rechargeable items (throwing stars, bullets) at qty=0 instead of deleting
  the inventory row, so players can recharge them at a shop.
- Gracefully handle the "nothing to consume" edge case (log, don't crash, don't roll
  back the attack).

Non-goals:
- Weapon↔projectile type validation (rejecting a bow user firing a throwing star). The
  client enforces this; we trust it.
- Replenishment, purchasing, inventory UI, or drop changes.
- The other 26 unimplemented attack effects listed in `docs/TODO.md` under "Character
  Attack Effects".
- Passive no-consume mechanics (Mortal Blow, Expert Marksmanship, claw mastery no-consume
  chance, etc.). Tracked as an explicit TODO — see §9.
- Magic, melee, and energy attacks — no projectile consumption applies.
- Javelin-flagged packet path — leave a TODO at the read site (see §9).

## 3. User Stories

- As a **player** firing a bow, I want each attack to consume one arrow from my
  consumable inventory so that ammunition management is a real gameplay concern.
- As a **player** using Soul Arrow, I want my attacks to not consume arrows while the
  buff is active so the skill delivers its canonical benefit.
- As a **player** using Shadow Partner, I want each throwing-star attack to consume
  double the stars a non-buffed attack would (including multi-projectile skills), so
  my ammo drain matches the doubled output.
- As a **player** using a multi-projectile skill (Strafe, Triple Throw, etc.), I want
  the full skill-defined `BulletConsume` count deducted per attack.
- As a **player** using a cash-shop throwing-star cover over real stars, I want the real
  stars to be consumed (not the cash item) so my cover persists indefinitely, as expected.
- As a **player** firing a skill flagged as non-consuming (e.g., Power Knockback), I want
  no ammo consumed so the skill behaves as designed.
- As a **player** who fires my last throwing star or bullet, I want the empty stack to
  remain in my inventory (qty=0) so I can recharge it at an NPC shop, matching classic
  MapleStory behavior.
- As a **server operator**, I want consumption failures (empty slot, race condition) to
  log and continue rather than crash the channel, so a single inventory desync doesn't
  kick a player.

## 4. Functional Requirements

### 4.1 Trigger conditions
Consumption is attempted when **all** are true:
- Attack is received via `CharacterRangedAttackHandleFunc` and `processAttack` runs.
- `AttackInfo.AttackType() == AttackTypeRanged`.
- `AttackInfo.Javlin() == false`. The `javlin` flag maps to a specific skill
  mechanic whose gameplay semantics are not yet fully understood (see §9 TODO #1).
  Until then, attacks flagged `javlin=true` bail out of consumption entirely to
  avoid mis-consuming; this is explicit, not accidental. Note: the field
  `javlin` is currently private on `AttackInfo`
  (`libs/atlas-packet/model/attack_info.go:60`) with no getter — a `Javlin() bool`
  accessor must be added alongside the existing `BulletItemId()`,
  `ProperBulletPosition()`, etc. getters to read it from the channel handler.
- The skill used is **not** in `skill.IsShootSkillNotConsumingBullet(skillId)`.
- The attacker's equipped weapon is a ranged weapon type (bow, crossbow, claw, gun).
  If not, we skip silently (unusual but possible with GM tooling — log at debug).
- For bow/crossbow weapons only: the attacker does **not** have the `SOUL_ARROW`
  temporary-stat buff active.

### 4.2 Consume count
- **Base count** = `skill.BulletConsume` if that effect field is > 0, else 1. The
  field is already parsed into `data/skill/effect/rest.go:59 BulletConsume`.
- **Shadow Partner multiplier**: if the attacker has the `SHADOW_PARTNER`
  temporary-stat buff active AND the weapon is a claw (throwing-star attack),
  multiply the base count by 2. Applied on top of the skill's `BulletConsume`
  (e.g., Triple Throw + Shadow Partner = 6 stars).
- The multiplier does not apply to bow/crossbow/gun attacks — Shadow Partner is a
  Hermit/Night Lord skill only.

### 4.3 Projectile lookup and slot resolution
The client sends `ProperBulletPosition` (slot in Use inventory) and
`CashBulletPosition` (slot in cash inventory) in the attack packet. We treat these as
**suggestions**, not authoritative. Resolution proceeds in this order:

1. Load the character's Use compartment via
   `character.Model.Inventory().Consumable()` (see §7 for path verification).
2. **Preferred single-slot path**: find a slot whose item matches the required
   projectile classification (see §4.4) AND has `quantity >= consumeCount`. If the
   client-supplied `ProperBulletPosition` points to such a slot, use it; otherwise
   pick the first matching slot in **ascending slot-index order**. Emit one
   `CONSUME` command with the full count.
3. **Multi-slot fallback**: if no single slot has enough, draw from multiple slots
   of the same classification in **ascending slot-index order**. Emit one
   `CONSUME` command per slot, each for the partial quantity drawn from that slot,
   until the required count is reached.
4. **Shortfall**: if the combined quantity across all matching slots is still less
   than the required count, consume everything available (emit commands for the
   partial total) and log a warning including `characterId`, `weaponItemId`,
   `skillId`, `required`, `available`. This should not occur organically — the
   client enforces ammo sufficiency before sending the attack — so persistent
   warnings indicate client-desync or tampering.
5. Cash projectiles (`CashBulletPosition`) are **never** consumed. The value is
   ignored for consumption; it affects only the rendered attack animation, handled
   by existing writers.

### 4.4 Weapon-to-projectile mapping and lookup paths
The equipped weapon is read from slot `-11` of the character's Equip compartment via
`character.NewProcessor(l, ctx).GetEquipableInSlot(characterId, -11)` (at
`services/atlas-channel/atlas.com/channel/character/processor.go:154`), or
equivalently `model.Inventory().Equipable().FindBySlot(-11)`.

Weapon type is derived via `item.GetWeaponType(equippedWeaponItemId)`:

| Weapon type                    | Required projectile classification |
|--------------------------------|------------------------------------|
| `WeaponTypeBow (15)`           | `ClassificationConsumableArrow (206)` (arrow subrange) |
| `WeaponTypeCrossbow (16)`      | `ClassificationConsumableArrow (206)` (bolt subrange)  |
| `WeaponTypeClaw (17)`          | `ClassificationConsumableThrowingStar (207)`           |
| `WeaponTypeGun (19)`           | `ClassificationBullet (233)`                           |

Bow arrows and crossbow bolts share MapleStory classification 206 — a single
`item.IsArrow()` helper covers both. We do not enforce the bow-arrow vs
crossbow-bolt subrange split; the client prevents cross-equipping, and the
inventory scan matches whatever valid classification-206 item the player has.

### 4.5 Order of operations
Compute the full consumption plan (resolved slots + per-slot counts) **before**
broadcasting the attack packet to nearby players, so any logic error or panic in
the planner is caught before visible side effects. Emit the `CONSUME` Kafka
commands **after** the attack broadcast, fire-and-forget. This keeps the attack
hot path off the Kafka round-trip while keeping the planning step in a place where
failures are surfaced early.

If the broadcast step itself errors after the plan has been computed, the
consumption emit still runs. Classic semantics: the projectile was expended the
moment the server accepted the attack packet, regardless of whether nearby
players saw the animation. The broadcast failure is logged separately; it does
not cancel consumption.

### 4.6 Consumption call
Each `CONSUME` command is emitted on `COMMAND_TOPIC_COMPARTMENT` with command type
`CommandConsume` and body `ConsumeCommandBody{ TransactionId, Slot }`, targeting
the Use inventory (`inventory.TypeValueUse`, value `2`). Uses the existing
compartment command producer pattern used elsewhere in atlas-channel for
inventory mutations. Each command carries a distinct transaction ID.

### 4.7 Rechargeable preservation (atlas-inventory change)
`services/atlas-inventory/atlas.com/inventory/compartment/processor.go` `ConsumeAsset`
currently deletes the asset row when the resulting quantity would be ≤ 0
(around line 819–820). This is incorrect for rechargeable items. Modify
`ConsumeAsset` so that when `item.IsRechargeable(templateId)` is true AND the
final quantity would be ≤ 0, the row is **updated to qty=0** and retained,
rather than deleted. All other item types retain the current delete-at-zero
behavior. The change is a property of the item (rechargeability), not of the
caller, so every consume path benefits — not just this feature.

A new helper `item.IsRechargeable(itemId Id) bool` is added to
`libs/atlas-constants/item` alongside `IsArrow`. Its current implementation is
`IsThrowingStar(itemId) || IsBullet(itemId)`; future rechargeable
classifications (if any) can be added there in one place.

### 4.8 Failure handling
- Weapon not ranged or no matching projectile classification → skip silently, debug log.
- No matching slots with any quantity → warning log, attack still resolves.
- Partial shortfall (some but not enough) → consume what's available, warning log,
  attack still resolves.
- Kafka emit failure on any `CONSUME` command → error log, continue with the
  remaining commands in the plan, attack still resolves.
- Downstream `CONSUME` command fails server-side (race) → inventory emits a
  failure status event; channel does not act on it. No client rollback of the
  attack.

Rationale: the attack packet has been broadcast by the time Kafka results return.
Rolling back would create a worse UX than a silent log. A determined cheater with
a modified client could occasionally get one free shot; real defense belongs in
client-authority hardening, which is out of scope.

## 5. API Surface

No new REST endpoints. No new Kafka topics. No changes to existing message shapes.

**Existing Kafka command reused:**
- Topic: `COMMAND_TOPIC_COMPARTMENT` (env-configured)
- Command: `CommandConsume = "CONSUME"`
- Body: `ConsumeCommandBody{ TransactionId, Slot }`
- Emitted via the existing compartment command producer pattern.

**New helpers in `libs/atlas-constants/item`:**
```go
func IsArrow(itemId Id) bool {
    return GetClassification(itemId) == ClassificationConsumableArrow
}

func IsRechargeable(itemId Id) bool {
    return IsThrowingStar(itemId) || IsBullet(itemId)
}
```

## 6. Data Model

No database changes. No new entities. No migrations.

Runtime state read (atlas-channel):
- Equipped weapon: `character.NewProcessor(l, ctx).GetEquipableInSlot(characterId, -11)`
  (`services/atlas-channel/atlas.com/channel/character/processor.go:154`), or via
  `model.Inventory().Equipable().FindBySlot(-11)`. Weapon is always slot `-11`.
- Use compartment: `model.Inventory().Consumable()` or
  `compartment.NewProcessor(l, ctx).GetByType(characterId, inventory.TypeValueUse)`.
- Temporary stats for `SOUL_ARROW` and `SHADOW_PARTNER` (already resident in channel
  session state).
- Skill effect data (including `BulletConsume`) via the existing skill effect reader
  at `services/atlas-channel/atlas.com/channel/data/skill/effect/rest.go`.

Runtime behavior change (atlas-inventory):
- `ConsumeAsset` retains asset rows at qty=0 when the template ID satisfies
  `item.IsThrowingStar || item.IsBullet`. Schema is unchanged; the `quantity`
  column simply holds 0 instead of the row being deleted.

## 7. Service Impact

### atlas-channel
- Extend `processAttack` in `socket/handler/character_attack_common.go`:
  - **Before** the existing attack-broadcast step: compute the consumption plan
    (gate on attack type, skill-not-consuming, weapon type, Soul Arrow; resolve
    slots + counts; apply Shadow Partner multiplier).
  - **After** the broadcast: emit the resolved `CONSUME` commands, fire-and-forget.
- New file (suggested): `socket/handler/character_attack_projectile.go` containing
  the planner and emitter as an Interface + Impl pair per project conventions,
  constructed via `NewProcessor(l, ctx)`.
- Reuse existing compartment command producer infrastructure used by
  `character_inventory_move.go` and siblings.
- The javelin-flagged read at `libs/atlas-packet/model/attack_info.go:153` is
  deliberately left alone — see TODO in §9.

### libs/atlas-constants/item
- Add `IsArrow(itemId Id) bool` mirroring `IsBullet` and `IsThrowingStar`.
- Add `IsRechargeable(itemId Id) bool` returning
  `IsThrowingStar(itemId) || IsBullet(itemId)`.

### libs/atlas-packet
- Add a `Javlin() bool` getter on `AttackInfo`
  (`libs/atlas-packet/model/attack_info.go`, alongside the existing
  `BulletItemId()` etc. getters) so the channel handler can read the flag.

### atlas-inventory
- Modify `ConsumeAsset` in
  `services/atlas-inventory/atlas.com/inventory/compartment/processor.go`
  (around lines 794–839): when `item.IsRechargeable(templateId)` is true AND the
  post-consume quantity would be ≤ 0, update the row to qty=0 rather than
  deleting it. All other item types retain the current delete-at-zero behavior.

### atlas-consumables
- No code changes. Existing `CONSUME` command contract is unchanged.

### atlas-data, atlas-assets, atlas-cashshop, atlas-inventory (schema), others
- No changes.

## 8. Non-Functional Requirements

- **Performance**: at least one additional Kafka emit per ranged attack, scaling
  up to N emits where N is the number of slots touched (multi-slot draw for
  multi-projectile skills under Shadow Partner, etc.). In the typical case N=1.
  All emits are fire-and-forget and do not block the attack broadcast path. The
  planner must not add measurable latency to the attack hot path — if the
  compartment lookup requires a remote fetch, it should execute concurrently
  with or after the attack broadcast.
- **Observability**: log at **debug** when consumption is skipped by design
  (Soul Arrow, non-consuming skill, javelin flag) — these can fire per-attack
  for whole classes of players and must not flood info-level output. Log at
  warn when no matching projectile is found in inventory or a partial shortfall
  occurs (anomaly indicators). Log at error for Kafka emit failures. All logs
  must include `characterId`, `weaponItemId`, `skillId`.
- **Multi-tenancy**: all Kafka emits must flow through the existing tenant-aware
  producer pattern (`tenant.MustFromContext(ctx)`). No cross-tenant leakage risk;
  consume commands carry the tenant header.
- **Security**: trust the client for slot suggestion but verify slot content matches
  expected projectile classification. Never consume from a slot whose item is not a
  valid projectile of the correct type.
- **Testability**: consumption helper should be a pure-enough function that can be
  unit-tested with mocked compartment state. Interface + Impl pattern per project
  conventions.

## 9. Open Questions

- None blocking implementation. All answered during scoping.

**Explicit TODOs to leave in code**, documenting deferred scope:

1. At `libs/atlas-packet/model/attack_info.go:153` (the `m.javlin && !IsShootSkillNotConsumingBullet`
   block that reads `bulletItemId`): add a `TODO` noting that the `javlin` flag is tied
   to a specific skill mechanic (poor translation of the original source) and that
   its interaction with projectile consumption is intentionally deferred. Also add
   a matching `TODO` in the projectile-consumption planner at the `javlin`-bailout
   gate (§4.1) so the two sites point at each other. Consumption is currently
   bypassed when `javlin=true` to avoid mis-consuming; when the mechanic is
   understood, revisit the gate and remove the bailout if appropriate.
2. In the new projectile-consumption helper: add a `TODO` for passive no-consume
   mechanics — Mortal Blow, Expert Marksmanship, Claw Mastery no-consume chance, and
   similar class-passive roll-to-preserve effects. These require reading passive skill
   levels and performing an RNG roll; intentionally out of scope for v1.

## 10. Acceptance Criteria

- [ ] Basic bow/crossbow/claw/gun attack (no special skill, no buffs): projectile
      stack decrements by 1 per shot.
- [ ] Multi-projectile skill (e.g., Strafe on a bow): stack decrements by the
      skill's `BulletConsume` value per attack.
- [ ] Triple Throw without Shadow Partner: star count decrements by 3 per attack.
- [ ] Triple Throw with Shadow Partner active: star count decrements by 6 per attack.
- [ ] Basic claw attack with Shadow Partner active: star count decrements by 2 per shot.
- [ ] Bow attack with Soul Arrow active: arrow count does not change.
- [ ] Crossbow attack with Soul Arrow active: bolt count does not change.
- [ ] Attack using a skill in `IsShootSkillNotConsumingBullet`: no consumption.
- [ ] Cash projectile cover equipped alongside real projectile: the underlying real
      projectile decrements; the cash item quantity does not change.
- [ ] Required count exceeds quantity in the client-suggested slot, but another slot
      has enough of the same classification: that other slot is used, single
      command, no warning.
- [ ] Required count exceeds any single slot but combined matching slots have
      enough: multiple `CONSUME` commands are emitted across slots, no warning.
- [ ] Required count exceeds total available: available quantity is consumed across
      all matching slots, warning is logged, attack still resolves.
- [ ] Firing the final throwing star: row is retained in inventory with qty=0
      (verify via inventory REST/DB inspection), player can return to NPC shop to
      recharge.
- [ ] Firing the final bullet: row is retained with qty=0.
- [ ] Consuming an empty non-rechargeable item to zero (any other consumable item
      type): row is deleted as before — regression check on existing behavior.
- [ ] Melee, magic, and energy attacks: no consumption attempted, no new logs.
- [ ] `item.IsArrow()` helper returns true for classification-206 items and false
      otherwise (mirrors `IsBullet` / `IsThrowingStar`).
- [ ] `item.IsRechargeable()` helper returns true for any throwing star or bullet
      item ID and false otherwise; called from `atlas-inventory` `ConsumeAsset`.
- [ ] Attack with `AttackInfo.Javlin() == true`: no consumption is attempted,
      skip logged at debug.
- [ ] TODO comments exist at the locations listed in §9 (packet `javlin` read
      site, planner bailout gate, passive no-consume block).
- [ ] All modified services build cleanly (`go build ./...`) and existing tests pass.
