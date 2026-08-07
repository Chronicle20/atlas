# Attack-Side Drain HP Gain — Implementation Context

Task: task-147-attack-drain-hp-gain
Companion to `plan.md`; read both before executing.

## What this builds

The four drain-family attack skills (Assassin Drain 4101005, Marauder Energy Drain 5111004, Thunder Breaker Energy Drain 15111001, Night Walker Vampire 14101006) heal the attacker per damaged monster: `heal = min(monsterMaxHp, floor(totalDamage × X / 100), effectiveMaxHp / 2)`, clamped to `int16`, emitted via the existing `ChangeHP` Kafka command. Reference behavior: Cosmic `AbstractDealDamageHandler.java:314-315`.

## Key files

| File | Role |
|---|---|
| `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go` | The ONLY production file changed. Gains `isDrainSkill`, `drainHealAmount`, `drainTryHeal`; hook widened; loader renamed; TODO removed. |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_drain_test.go` | New test file (all tasks append to it). |
| `libs/atlas-constants/skill/constants.go` | Read-only: `AssassinDrainId` (:3150), `MarauderEnergyDrainId` (:3205), `ThunderBreakerStage3EnergyDrainId` (:3289), `NightWalkerStage2VampireId` (:3367), `AranStage2ComboDrainId` (:3400, excluded). |
| `services/atlas-channel/atlas.com/channel/monster/model.go` + `builder.go` | `Model.MaxHp() uint32` (model.go:120); tests build via `NewModelBuilder(uniqueId, field, monsterId).SetMaxHp(n).Build()` (builder.go:30,59). |
| `services/atlas-channel/atlas.com/channel/monster/status_mirror.go:39-47` | `ReflectInfo` exported fields (Kind/Percent/LtX/LtY/RbX/RbY/MaxDamage) used by the reflect-path hook test. |
| `services/atlas-channel/atlas.com/channel/effective_stats/rest.go:12` | `RestModel.MaxHp uint32` — the buff-inclusive cap source. |
| `services/atlas-channel/atlas.com/channel/character/processor.go:43,276` | `ChangeHP(f field.Model, characterId uint32, amount int16) error` — existing emit path, unchanged. |

## Key decisions (from design.md — do not relitigate)

- **D1/D2**: heal fires from the existing `onDamageApplied` hook, widened to `func(monsterId uint32, totalDamage uint32)`. Reflected and zero-damage entries never reach the hook, so drain inherits those exclusions for free.
- **D3**: `X` comes from the already-fetched `se` (`effect.Model.X() int16`) — the attack skill IS the drain skill. No second effect fetch, no per-monster fetch.
- **D4**: the venom lazy effective-stats loader is renamed `loadVenomStats` → `loadEffectiveStats` (closure, cache vars, deps field) and shared. Its fail-safe (zero `RestModel` on error → heal caps to 0) is exactly the drain policy.
- **D5**: all math in pure `drainHealAmount(totalDamage uint32, x int16, monsterMaxHp uint32, effectiveMaxHp uint32) int16`; uint64 internals, floor division, int16 clamp.
- **D6**: `isDrainSkill` is a `switch` over the four constants; Aran Combo Drain deliberately excluded (its TODO stays).
- **D7**: one `ChangeHP` per damaged monster (no batching) — matches Cosmic's per-monster loop.
- **D8**: `drainTryHeal` takes injected funcs (`getMonster`, `changeHP`, `loadEffectiveStats`) — unlike `mpEaterTryProc`'s concrete `*monster.Processor` — precisely so flow tests exist. No attack-type gate on drain (the skills span melee/ranged/energy); skill id is the filter.

## Gotchas discovered during planning

- **No existing test constructs `damageInfoEntryDeps`** — existing "flow" tests (`TestReflectFlow_*`) compose pieces manually. The design's mention of updating "existing tests that construct the deps struct" is moot; the hook widening touches production code + new tests only.
- `ai.SkillId()` returns `uint32` (`attack_info.go:397`); the `uint32(ai.SkillId())` cast at `character_attack_common.go:128` is redundant legacy. Pass `ai.SkillId()` straight into `drainTryHeal`.
- `se` is a zero `effect.Model` when `ai.SkillId() == 0`; the drain branch gates on `ai.SkillId() > 0 && isDrainSkill(...)`, so zero-`se` X can't leak in.
- Damage sum: `di.Damages()` is `[]uint32`; sum in `uint64` and clamp to `math.MaxUint32` before passing as `uint32` (v83 legit max ≈ 999,999 × 15 lines, far below).
- Test fixtures: `packetmodel.NewAttackInfo(type).SetSkillId(id)`, `packetmodel.NewDamageInfo(hits).SetMonsterId(id).SetDamages([]uint32{...})` — deref to values (`*ptr`) for `processDamageInfoEntry`. Field: `field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(...)).SetInstance(uuid.Nil).Build()`. Tenant: `tenant.Create(uuid.New(), "GMS", 83, 1)`.
- The module root is `services/atlas-channel/atlas.com/channel` (module `atlas-channel`). Run `go test/vet/build` there; run `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh` and `tools/lint.sh --check` from the worktree root WITHOUT a global `GOWORK=off` prefix.
- The TODO block sits near the end of `processAttack` (pre-change line 410, block spans 403-426, 24 TODO lines); earlier tasks shift line numbers — locate the drain TODO by content, delete only that line.
- **No `MajorVersion` gating in this feature.** GMS 48/61/72/79 are supported on main, but `processAttack` destroys the session when the character does not own the cast skill (`character_attack_common.go:283-291`), so a version-absent drain skill is structurally unreachable. See PRD §8.1.
- This branch predates task-171's lint baseline. Run `tools/lint.sh` (fix mode) once after the first code edit rather than debugging individual gofumpt/goimports diffs.

## Dependencies / ordering

Tasks 1 and 2 are independent pure helpers. Task 3 (hook widening + loader rename) is independent of 1-2 but must precede Task 5. Task 4 needs Task 2 (`drainHealAmount`). Task 5 needs all of 1-4. Execute in plan order; each task commits independently and leaves the module green.

## Out of scope (PRD non-goals — do not add)

Energy-charge server validation; every other TODO in the post-attack block (including `// TODO Combo Drain`); packet/opcode/template changes; MP Eater changes; atlas-character changes; new REST/Kafka surface.

## Verification gate (before claiming done)

`go build ./... && go vet ./... && go test -race ./...` clean in `services/atlas-channel/atlas.com/channel`; `tools/redis-key-guard.sh` clean from repo root; `go.mod` untouched (so no bake needed). Code review via `superpowers:requesting-code-review` before any PR.
