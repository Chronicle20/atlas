# Combo Drain (Aran) — Execution Context

Task: task-166-combo-drain
Companion to: `plan.md` (implementation plan), `design.md` (approved design), `prd.md` (requirements)

## What this task is

The `COMBO_DRAIN` buff (Aran skill 21100005) applies and renders correctly, but the attack pipeline never reads it, so the heal never fires. This task replaces the `// TODO Combo Drain` in atlas-channel's `processAttack` with a once-per-attack heal of `totalDamage * x / 100` HP (buff statup amount `x`), clamped to `math.MaxInt16`, emitted via the existing `ChangeHP` Kafka command. Single service: **atlas-channel**. No new REST endpoints, Kafka topics, packets, templates, or schema.

## Key files

| File | Role |
|---|---|
| `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go` | `processAttack` — buff fetch hoisted before `pp.Plan` (~line 315); proc call replaces the `// TODO Combo Drain` line (~line 420). `cp` (`character.Processor`) and `s` (session) are already in scope. |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_projectile.go` | `ProjectileProcessor.Plan` gains a `buffs []buff.Model` param; internal `bp buff.Processor` field + fetch block removed. `hasBuff`/`computeCount` already take `[]buff.Model` — untouched. |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_combo_drain.go` | NEW — pure helpers `buffStatAmount`, `attackTotalDamage`, `comboDrainHealAmount` + orchestrator `comboDrainTryProc` (MP Eater style: pure functions + TryProc with injected side effect). |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_combo_drain_test.go` | NEW — table-driven tests, production constructors only. |
| `services/atlas-channel/atlas.com/channel/character/buff/model.go` | `buff.Model`: `Expired()`, `Changes() []stat.Model`; `stat.Model`: `Type() string`, `Amount() int32`. |
| `services/atlas-channel/atlas.com/channel/character/processor.go:271` | `ChangeHP(f field.Model, characterId uint32, amount int16) error` — emits the Kafka command; atlas-character owns max-HP clamping. |
| `libs/atlas-constants/character/temporary_stat.go:75` | `TemporaryStatTypeComboDrain = "COMBO_DRAIN"` — the gate stat. |
| `libs/atlas-packet/model/attack_info.go`, `damage_info.go` | `AttackInfo.DamageInfo()`, `DamageInfo.Damages() []uint32`; test builders `NewAttackInfo(at).AddDamageInfo(*NewDamageInfo(hits).SetMonsterId(..).SetDamages(..))`. |
| `docs/TODO.md:108` | `- [ ] Combo Drain` line item to check off. |

## Decisions locked in design (do not re-litigate)

- **Approach B**: one buff fetch per attack in `processAttack`, threaded into `Plan` and into `comboDrainTryProc`. Fetch failure ⇒ warn + `buffs = nil`; never abort the attack. The nil-slice case IS the fetch-error case at proc level.
- **Deliberate FR-1 deviation** (documented in design §2): the *fetch* moves pre-`Plan` to satisfy the single-fetch NFR; *evaluation/emission* stay at the TODO site, post-damage.
- **Buff-only gate**: no job / skill-ownership / attack-type check (Approach D rejected — a COMBO_DRAIN statup from any source must work).
- **One heal per attack** from the plain total over all monsters and hit lines — Cosmic's per-monster running-total over-heal is explicitly NOT replicated.
- **Overflow discipline**: sum in `uint64`; early-saturate when `totalDamage >= MaxInt16*100` (so the multiply can never wrap for any `int32` percent); clamp to `MaxInt16` before narrowing to `int16`. No emission when heal `<= 0`.
- **First non-expired match wins** when multiple buffs carry the stat (mirrors `hasBuff`). Reflected entries' damage lines still count toward the total.
- **Merge hygiene**: sibling tasks (147, 152, 167, …) edit the same TODO block in their own worktrees — replace exactly the one `// TODO Combo Drain` line in place, touch nothing adjacent.

## Discoveries made during planning (corrections to design assumptions)

- Design §5 anticipated "mechanical updates" to projectile tests that "inject a fake `buff.Processor`". **Verified false**: `character_attack_projectile_test.go` only exercises pure helpers (`computeCount`, `resolvePlan`, `hasBuff`, `requiredClassification`) with `[]buff.Model` slices directly — nothing constructs `ProjectileProcessorImpl` or calls `Plan`. `Plan` has exactly one production call site (`character_attack_common.go:316`). So the interface change requires **zero test edits**.
- Package-level test helper names already taken in `package handler`: `buffWithStat`/`expiredBuffWithStat` (projectile test, amount hardcoded to 1) and `testField(mapId)` (mystic door test — reuse it). The new test file uses `comboDrainBuffWithAmount`/`expiredComboDrainBuffWithAmount`/`attackWithDamages` to avoid redeclaration.
- The pattern to mirror for the orchestrator is `mpEaterTryProc` (`character_attack_common.go:206-263`): injected side effects, errors logged and swallowed, debug log on proc.

## Dependencies

- No cross-task/service dependencies. atlas-data already emits the `COMBO_DRAIN` statup (`skill/reader.go:372-373`); atlas-buffs REST read and atlas-character `ChangeHP` consumer are used as-is.
- Task order within the plan matters: Task 3 (wiring) needs Task 2's `comboDrainTryProc`, which needs Task 1's helpers.

## Verification gates (design §6)

From `services/atlas-channel/atlas.com/channel`: `go build ./... && go vet ./... && go test -race ./...` clean.
From worktree root: `tools/redis-key-guard.sh` clean; `docker buildx bake atlas-channel` succeeds (mandatory despite untouched `go.mod`).
`grep -rn "TODO Combo Drain"` over `services/` and `docs/TODO.md` must be empty; TODO.md item checked.
Then `superpowers:requesting-code-review` before any PR (project rule).
