# task-160-skill-cast-consumption — Execution Context

Companion to `plan.md`. Summarizes the key files, locked decisions, and dependencies an implementer needs; the plan has the step-by-step detail.

## What this task does

Three atlas-channel behavior changes (PRD FR-1/2/3), plus one getter in `libs/atlas-packet` and one moved pure function in `libs/atlas-constants`:

1. **FR-1** — skill casts consume the WZ `itemConNo` quantity (was hardcoded `1`), drawn from the lowest-index slot holding ≥ that amount. Shortfall: warn + skip + cast proceeds (unchanged stance).
2. **FR-2** — buff casts with `bulletConsume > 0` (Shadow Stars 4121006) consume that many matching projectiles at cast time from a single qualifying USE slot, and the SHADOW_CLAW statup amount is rewritten to `consumedStarItemId − 2069999` before the buff applies. No qualifying slot: reject the cast with zero side effects.
3. **FR-3** — claw attacks while SHADOW_CLAW is active skip projectile consumption entirely (alongside the existing Soul Arrow skip, before `computeCount` so Shadow Partner ×2 can't resurrect it).

## Key files

| File | Role |
|---|---|
| `services/atlas-channel/atlas.com/channel/skill/handler/common.go` | `UseSkill` cast orchestration; gets the bulletConsume gate (first), the itemCon amount plumbing, and three new test seams |
| `services/atlas-channel/atlas.com/channel/skill/handler/bullet_consume.go` | NEW — `consumeCastBullets`, `rewriteShadowClawAmount`, `shadowClawStarEncodingBase` |
| `services/atlas-channel/atlas.com/channel/consumable/processor.go` | `RequestItemConsume` gains `quantity int16` (before `updateTime`); floors `<1` to 1 |
| `services/atlas-channel/atlas.com/channel/compartment/model.go` | NEW method `FindFirstByItemIdWithQuantity(templateId uint32, quantity int16)` — sorts by slot ascending |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_projectile.go` | SHADOW_CLAW skip in `Plan()`; `requiredClassification` deleted in favor of the lib function |
| `libs/atlas-constants/item/constants.go` | NEW `ProjectileClassificationForWeapon(WeaponType) (Classification, bool)` (moved mapping; DOM-21) |
| `libs/atlas-packet/model/skill_usage_info.go` | NEW getter `SpiritJavelinItemId() uint32` (field + builder setter already exist; decoder reads it at line 32 for skill 4121006) |

Existing `RequestItemConsume` call sites that gain a literal `1` (behavior unchanged): `socket/handler/character_item_use.go` (×3), `character_cash_item_use.go`, `pet_food.go`, `pet_item_use.go`, plus `skill/handler/common.go` (which then gets the real amount in Task 4).

## Locked decisions (from design.md — do not relitigate)

- **Signature change, not a sibling method**, for the quantity parameter — the compiler enforces the call-site sweep (design §5.1).
- **`shadowClawStarEncodingBase = 2069999` is a named constant in domain code**, not tenant config: IDA-verified byte-identical across v83 0x949C4C / v87 0x9C4A50 / v95 0x907461 / jms185 0xA0A2F4 (`+0x1F95EF`), and it's a stat *value packing* like Sharp Eyes `x<<8|y`, not a wire byte (design §2.1, §5.2). The rewrite is REQUIRED — amount 0 makes the client resolve nonexistent item 2069999 and refuse to attack.
- **Single-slot draws only** for both consume paths. The v83 client gates on the aggregate across slots; a 150+150 split therefore casts client-side but the server rejects — benign, warn-logged, documented divergence with a named upgrade path (design §5.3). Do not build the multi-slot draw.
- **Rejection semantics differ deliberately**: itemCon shortfall permits the cast (defense-in-depth stance preserved); bulletConsume shortfall rejects it before HP/MP consume (the star cost *is* the skill).
- **Hint handling** (design §2.2): a classification-valid `SpiritJavelinItemId` restricts candidates to that star; an invalid hint is ignored (forgery guard) and the generic classification scan applies.
- **`rewriteShadowClawAmount` is replace-only** — no SHADOW_CLAW entry means passthrough; appending one would grant free-throw semantics to a hypothetical non-Shadow-Stars bulletConsume buff.
- **Weapon→projectile mapping lives in `libs/atlas-constants/item`** because `skill/handler` cannot import `socket/handler` (cycle) and duplicating the switch is a divergence seed (design §5.4).
- **Optional atlas-consumables pinning test is NOT built** — recorded in plan.md Global Constraints with rationale (no behavior change there; no test seam infra; wire value pinned by the atlas-channel producer test).

## Test strategy

- Package-level var-func seams restored via `t.Cleanup` (existing `common.go` convention). New seams: `loadCasterWithInventoryFunc`, `requestItemConsumeFunc`, `applyBuffStatupsFunc` (the last wraps self-apply + party fan-out so one override captures applied statups).
- Builder-pattern setup only (`character.NewModelBuilder`, `inventory.NewBuilder`, `compartment.NewBuilder`, `asset.NewModelBuilder`, `equipment.NewModel()+Set`, `effect.Extract(effect.RestModel{...})`, `packetmodel.NewSkillUsageInfoBuilder`); no `*_testhelpers.go`.
- `UseSkill` is testable offline with those seams because every other side-effect path is naturally inert in tests: HP/MP/cooldown gated on zero-valued effect fields, mob path early-returns on empty `AffectedMobIds`, dispatcher lookup misses the arbitrary test skill id, mount guards don't match it.
- FR-3 tests drive `ProjectileProcessorImpl.Plan` directly with a `stubBuffProcessor` (4-method `buff.Processor` interface); `cpp` stays nil since `Plan` never touches it.
- `github.com/pkg/errors` is NOT in atlas-channel's go.mod — use stdlib `errors` in tests.

## Task order & dependencies

```
Task 1 (constants move)      ─┐
Task 2 (compartment helper)  ─┼─→ Task 4 (UseSkill itemCon + seams) ─→ Task 6 (bulletConsume gate)
Task 3 (quantity signature)  ─┘                                          ↑
Task 5 (packet getter)       ─────────────────────────────────────────────┘
Task 1 ─→ Task 7 (attack-path skip)    Task 8 (verification) last
```

Tasks 1/2/3/5 are independent of each other. Task 6 consumes Task 4's seams and `statups` local, Task 1's lib function, Task 3's quantity plumbing, and Task 5's getter.

## Verification (Task 8)

`go test -race`, `go vet`, `go build` in `services/atlas-channel/atlas.com/channel`, `libs/atlas-constants`, `libs/atlas-packet`; `docker buildx bake atlas-channel` from the worktree root; `tools/redis-key-guard.sh` from the repo root (no global `GOWORK=off`). No `go.mod`/Dockerfile/`go.work` changes expected. Code review (`superpowers:requesting-code-review`) before any PR.

## Reference points in the current tree (pre-change line numbers)

- Hardcoded quantity: `consumable/processor.go:30`
- itemCon block: `skill/handler/common.go:79-92`; buff apply block: `common.go:107-111`
- Soul Arrow skip (insertion anchor for FR-3): `character_attack_projectile.go:107-111`
- `requiredClassification` (to delete): `character_attack_projectile.go:209-222`; call site: line 90
- Sort-by-slot convention: `resolvePlan`, `character_attack_projectile.go:256`
- Runtime-synthesized statup precedent: `skill/handler/mount.go` `tamedMountStatups`
- SpiritJavelin decode: `libs/atlas-packet/model/skill_usage_info.go:32`
- `TemporaryStatTypeShadowClaw`: `libs/atlas-constants/character/temporary_stat.go:46`
