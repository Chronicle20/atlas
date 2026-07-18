# Shadow Stars (Night Lord 4121006) — Implementation Context

Companion to `plan.md`. Key files, verified facts, and decisions an implementer
needs without re-deriving them. All paths relative to the worktree root
`.worktrees/task-158-shadow-stars/`.

## Constants (verified)

- `skill.NightLordShadowStarsId = Id(4121006)` — `libs/atlas-constants/skill/constants.go:3158`.
- `character.TemporaryStatTypeShadowClaw = "SHADOW_CLAW"` — `libs/atlas-constants/character/temporary_stat.go:46`.
- `character.TemporaryStatTypeShadowPartner = "SHADOW_PARTNER"` — same file line 32.
- `character.TemporaryStatTypeSoulArrow` — same file (used by the existing bow/crossbow carve-out).
- `item.ClassificationConsumableThrowingStar = Classification(207)` — `libs/atlas-constants/item/constants.go:35`. `item.GetClassification(item.Id)` / `item.IsThrowingStar(item.Id)` at lines 173–174. Classification derives purely from the item id — no WZ round-trip.
- `item.WeaponTypeClaw`, `WeaponTypeBow`, `WeaponTypeCrossbow`, `WeaponTypeGun` — used by the projectile gate; `item.GetWeaponType(item.Id)`.
- Reference star ids for tests: `2070000` Subi, `2070006` Ilbi (both classification 207). `2000000` Red Potion = a non-star (classification 200).

## Decode ordering (verified — matters for Task 1 fixture)

`SkillUsageInfo.Decode` (`libs/atlas-packet/model/skill_usage_info.go:23-51`) reads
`spiritJavelinItemId` (line 33) only for `4121006`. `4121006` is NOT in
`isAntiRepeatBuffSkill`, `isPartyBuff`, or `isMobAffectingBuff` (grepped: only one
occurrence of `NightLordShadowStars` in the file — the decode gate). So the wire
layout for Shadow Stars is exactly:

```
updateTime (uint32 LE) | skillId (uint32 LE) | skillLevel (byte) | spiritJavelinItemId (uint32 LE)
```

No castX/castY, no party bitmap, no mob list. The field is already settable via
`SetSpiritJavelinItemId` (builder line 109); only the getter is missing.

`SkillUsageInfo` has **no `Encode`** — it's serverbound decode-only, so the `pt.RoundTrip`
helper can't be used. Build a raw `request.Request(bytes)` →
`request.NewRequestReader(&req, 0)` and call `Decode(l, ctx)(&reader, opts)` directly
(`Decode` ignores both logger and context — `context.Background()` is fine, no tenant needed).

## Injection precedent — `mount.go` (the FR-2 pattern)

`skill/handler/mount.go:61-76` `tamedMountStatups` copies `e.StatUps()` and overrides
the `MONSTER_RIDING` entry's amount with the runtime vehicle id. `rewriteShadowClawStatups`
is the same shape for `SHADOW_CLAW` ← star id. The remote-render requirement (observers
see the correct star) needs no new code: every buff applied via `buff.Apply` is already
projected to observers — carrying the real id in the statup amount is sufficient (same
reason `MONSTER_RIDING`'s vehicle id renders remotely).

`statup.Model` is `{buffType string, amount int32}`, built via
`statup.NewModel(mask string, amount int32)` (`data/skill/effect/statup/model.go`). The
star id is `uint32`; cast to `int32` for the amount. `SHADOW_CLAW` wire-encodes as an int
foreign value (`libs/atlas-packet/model/character_temporary_stat.go:124`), so the client
reads the amount directly as the star id.

## Consume precedent — projectile path (the FR-4 pattern)

`socket/handler/character_attack_projectile.go`:
- `Plan()` (lines 64-148) resolves per-slot draws via the pure `resolvePlan` (lines 246-291).
- `Emit()` (lines 150-175) — for each draw: `once.ReservationValidator(txId, itemId)` +
  `reservedToConsume` handler registered via `consumer.GetManager().RegisterHandler(t,
  message.AdaptHandler(message.OneTimeConfig(validator, handler)))`, then
  `cpp.RequestReserve(txId, characterId, inventory.TypeValueUse, []compartmentMsg.ItemBody{...})`.
- Topic: `topic.EnvProvider(l)(compartmentMsg.EnvEventTopicStatus)()`.
- `compartmentMsg.ItemBody{Source int16, ItemId uint32, Quantity int16}` — `kafka/message/compartment/kafka.go:66`.
- `compartment.Processor` (`compartment/processor.go`): `RequestReserve(txId, characterId, inventory.Type, []ItemBody)` (line 75), `Consume(txId, characterId, inventory.Type, slot int16)` (line 79).

`emitStarConsume` (Task 5) mirrors `Emit` but lives in the `skill/handler` package (a
different package from `socket/handler`), so it re-declares a small local
`reservedStarToConsume` handler rather than importing `reservedToConsume`. This is a
focused function, not a duplicated framework — honors the "no generic batch-consume" non-goal.

## Projectile gate — `character_attack_projectile.go` (FR-3)

Existing Soul Arrow carve-out at lines 107-111 (bow/crossbow + `SOUL_ARROW`). Task 3 folds
it into a pure `projectileConsumptionSkipped(weaponType, buffs)` and adds the claw +
`SHADOW_CLAW` case. `hasBuff` (line 195) already skips expired buffs, so the carve-out is
inactive-safe (a claw attack with no live `SHADOW_CLAW` falls through to normal consumption
— the FR-3 regression requirement). `computeCount`'s Shadow-Partner doubling (line 232)
becomes dead for Shadow-Stars attacks because the gate returns before reaching it — resolves
PRD Open Question 5 with no extra code.

`Plan()` loads buffs internally via `p.bp.GetByCharacterId` (Kafka/REST) and is therefore
not unit-testable offline; only the pure gate function is tested. Existing test helpers
`buffWithStat` / `expiredBuffWithStat` / `makeAsset` are in
`character_attack_projectile_test.go:26-68` and are reused.

## Orchestrator wiring — `skill/handler/common.go` `UseSkill`

`UseSkill(l)(ctx)(wp, f, characterId, info packetmodel.SkillUsageInfo, e effect.Model) error`
(lines 70-126). Body order today: HP consume → MP consume → item consume (defense-in-depth)
→ cooldown → mount short-circuit (returns for mounts) → generic buff apply (lines 107-111)
→ `applyToMobs` → per-skill dispatcher `Lookup`.

Shadow Stars is NOT a mount and NOT mob-affecting and has no registered per-skill handler,
so it flows through the generic buff-apply path. The plan changes exactly three things for
`4121006`:
1. Pre-flight at the very top (before HP/MP/cooldown): load inventory via the new
   `loadCasterInventoryFunc` seam, call `resolveShadowStarsCast`, abort (`return nil`) on
   invalid star or inventory load failure.
2. The statups handed to the generic `buff.Apply` become the rewritten set (`SHADOW_CLAW`
   amount = star id).
3. After `buff.Apply`, `emitStarConsume` charges the cast cost.

Seam precedent: `loadCasterFunc` at `common.go:31` — a package-level `var` function the
tests replace. `loadCasterInventoryFunc` mirrors it. Production impl calls
`cp.GetById(cp.InventoryDecorator)(characterId)` then `c.Inventory().Consumable().Assets()`
(the same decorated load the generic item-consume block uses at `common.go:81`).

`character.NewProcessor(l, ctx)` returns `character.Processor`. `effect.Model.BulletCount()`
(added in Task 2) is the WZ one-time cost (200 in reference data); already populated by
`rest.go:59,130` — only the getter is missing (`BulletConsume()` exists but is the per-attack
count, a different field).

## Resolved decisions (from design §4)

1. Cast cost ships self-contained in this task (no generic P8 framework exists in-repo).
2. Validate (classification + ownership), not trust-client — the id drives a real consume.
3. Inject by rewriting the statup amount in `common.go` before `buff.Apply` (mount precedent);
   no Kafka payload change.
4. Shortfall → consume what's available + warn (matches projectile shortfall posture).
5. Shadow Partner interaction is moot — FR-3 returns before `computeCount`.
- **Added:** validation runs at the very top of `UseSkill`, so a bogus star aborts before
  any HP/MP/cooldown spend or buff apply.

## Modules & build gates

Touched `go.mod`s: `libs/atlas-packet` and `services/atlas-channel`. Per CLAUDE.md:
`go test -race ./...`, `go vet ./...`, `go build ./...` clean in each;
`docker buildx bake atlas-channel` from the worktree root; `tools/redis-key-guard.sh`
clean from repo root (run without a global `GOWORK=off` prefix). No new shared lib is added,
so the shared `Dockerfile` and `go.work` need no edits. `atlas-data` is untouched
(`reader.go:298` keeps the `SHADOW_CLAW` placeholder `0`).

This task adds no Redis usage, so the key-guard is a formality here.

## Test construction cheatsheet

- Assets: `asset.NewModelBuilder(1, uuid.New(), templateId).SetSlot(slot).SetQuantity(qty).MustBuild()`.
- Buffs: `buff.NewBuff(sourceId, level, duration, []stat.Model{stat.NewStat(string(tsType), amount)}, createdAt, expiresAt)`; `Expired()` = `expiresAt.Before(time.Now())` — future `expiresAt` = active.
- Statups: `statup.NewModel(string(charconst.TemporaryStatTypeShadowClaw), amount)`.
- Raw decode reader: `req := request.Request(bytes); reader := request.NewRequestReader(&req, 0)`.
