# Projectile Consumption on Attack — Context

Last Updated: 2026-04-18

## Key Files

### atlas-channel

- `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go:23` —
  `processAttack`. Consumption planner call goes after skill-effect load and
  before the per-session broadcast; emit goes after the broadcast loop.
- `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_ranged.go` —
  ranged-attack routing into `processAttack`.
- `services/atlas-channel/atlas.com/channel/socket/handler/character_inventory_move.go` —
  reference for the existing compartment command producer pattern to reuse
  in Phase D2.
- `services/atlas-channel/atlas.com/channel/character/processor.go:154` —
  `GetEquipableInSlot`. Weapon is always slot `-11`.
- `services/atlas-channel/atlas.com/channel/data/skill/effect/rest.go:59` —
  `BulletConsume` effect field already parsed.
- **New:** `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_projectile.go` —
  planner + emitter as Interface + Impl, `NewProcessor(l, ctx)`.

### atlas-inventory

- `services/atlas-inventory/atlas.com/inventory/compartment/processor.go`
  around lines 794–839 — `ConsumeAsset`. Gate delete-at-zero on
  `!item.IsRechargeable(templateId)`.

### libs/atlas-constants/item

- Add `IsArrow(itemId Id) bool`.
- Add `IsRechargeable(itemId Id) bool` returning
  `IsThrowingStar(itemId) || IsBullet(itemId)`.
- `ClassificationConsumableArrow = 206`, `ClassificationConsumableThrowingStar = 207`,
  `ClassificationBullet = 233` already defined.
- `GetWeaponType`: `WeaponTypeBow=15`, `WeaponTypeCrossbow=16`,
  `WeaponTypeClaw=17`, `WeaponTypeGun=19`.

### libs/atlas-packet

- `libs/atlas-packet/model/attack_info.go:60` — private `javlin` field.
- `libs/atlas-packet/model/attack_info.go:153` — javlin read site; TODO E1 lands here.
- Add `Javlin() bool` getter alongside `BulletItemId()` etc.

## Key Decisions

1. **Plan before broadcast, emit after.** Planner errors surface before visible
   side effects; broadcast failures do not cancel consumption because the
   projectile is semantically expended on accept.
2. **Client slot is a hint, not authority.** Verify classification; fall back
   to ascending-slot scan; multi-slot draw for shortfalls on multi-projectile
   skills.
3. **Cash projectile positions are cosmetic only.** Never consumed.
4. **`IsRechargeable` lives in `libs/atlas-constants/item`.** Property of the
   item, not the caller — every consume path benefits.
5. **Javlin flag bails out of consumption for now.** Explicit, not accidental;
   TODO cross-referenced between packet site and planner.
6. **Passive no-consume mechanics deferred.** Mortal Blow / Expert Marksmanship /
   Claw Mastery out of scope; planner-side TODO documents the stub.
7. **No new Kafka topics, no DB migrations, no tenant-config.** Reuses
   `COMMAND_TOPIC_COMPARTMENT` + `CommandConsume`.
8. **Fire-and-forget emit.** Do not block the attack hot path on Kafka acks.
   Kafka failures log at error and move on.

## Dependencies Between Phases

- **A → B:** `IsRechargeable` must exist before `ConsumeAsset` can call it.
- **A → C:** `IsArrow` and `Javlin()` getter must exist before the planner
  can gate on them.
- **B → D:** Rechargeable fix should ship before the planner starts emitting
  last-shot consumes, so the first recharge-at-shop use case works in QA.
- **C → D:** Planner must exist and be unit-tested before `processAttack`
  wires it in.
- **D → E:** TODOs reference planner locations, so they land with or after D.
- **F depends on all prior.**

Phases B and C can start in parallel once A lands.

## External References

- PRD for this task: `prd.md` in this directory. §4 (functional requirements),
  §7 (service impact), §9 (deferred TODOs), and §10 (acceptance criteria)
  are the primary implementer references.
- `docs/TODO.md` "Character Attack Effects" backlog — this task checks off
  projectile consumption; 26 other effects remain.
- task-005 / task-006 `context.md` files as structural precedent.

## Open Questions

None blocking implementation. The PRD resolved all scoping questions during
its authoring pass. Deferred items are captured as explicit TODOs (§E1–E3
in `plan.md`).
