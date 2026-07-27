# Task 148 — Sacrifice Self-HP Cost: Context

Companion to `plan.md`. Everything an executor needs that isn't a plan step.

## What this is

Dragon Knight Sacrifice (1311005) attacks work end-to-end in atlas-channel but cost the caster nothing. This task adds the self-HP cost — `firstDamageLine × X / 100`, clamped to leave ≥ 1 HP — resolving the TODO at `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go:669`. Reference behavior source-verified from Cosmic (`CloseRangeDamageHandler.java:142-149`, `AbstractCharacterObject.safeAddHP`).

> All `character_attack_common.go` line numbers here were re-verified against `main` after a rebase. If drifted, re-locate by symbol (`comboOrbTryUpdate`, `mpEaterAbsorbAmount`, the Sacrifice TODO comment).

## Key files

| File | Role |
|---|---|
| `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go` | The only production file changed: two pure helpers + one gated block in `processAttack` |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_sacrifice_test.go` | New test file (naming mirrors `character_attack_mp_eater_test.go`) |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_mp_eater_test.go` | Style/idiom template for the tests |
| `libs/atlas-constants/skill/constants.go:3002` | `DragonKnightSacrificeId = Id(1311005)` |
| `libs/atlas-packet/model/attack_info.go` / `damage_info.go` | `AttackInfo.DamageInfo() []DamageInfo`, `DamageInfo.Damages() []uint32` (`:90`), builders for tests |
| `services/atlas-channel/atlas.com/channel/data/skill/effect/model.go:154` | `effect.Model.X() int16` — the tenant-resolved cost percentage |
| `services/atlas-channel/atlas.com/channel/character/model.go:132` | `character.Model.Hp() uint16` |
| `services/atlas-channel/atlas.com/channel/character/processor.go:276` | `ChangeHP(f field.Model, characterId uint32, amount int16) error` — fire-and-forget Kafka command |

## Import-aliasing trap

Inside `character_attack_common.go` (imports at :3-29):
- `skill3` = `github.com/Chronicle20/atlas/libs/atlas-constants/skill` → the constant is **`skill3.DragonKnightSacrificeId`**.
- bare `skill` = `atlas-channel/character/skill` (the character-owned skill model) — not the constants package.
- `skill2` = `atlas-channel/data/skill` (effect processor), `packetmodel` = `libs/atlas-packet/model`.
- `math` is already imported (:18) — no import changes needed in the production file.

## Locked decisions (from PRD interview + design)

1. **Never kill the caster:** clamp channel-side to `Hp − 1`; at `Hp ≤ 1` the cost is 0 and nothing is emitted. atlas-character is not changed.
2. **Cost basis is entry[0] line[0] only** — never sum lines/targets, even on a multi-entry packet. Test-pinned (FR-2).
3. **Clamp reads `c.Hp()`** from the model fetched at attack start; effective stats are not consulted. Staleness vs concurrent damage is accepted (Cosmic has the same shape).
4. **Version-agnostic:** `X` comes per-tenant from atlas-data via the `se` effect already fetched at :528. No `MajorVersion()` branches. The single path covers all 11 versions main exposes; validate across the supported set (§10 of the PRD), not just v83+v95. **Data verified 2026-07-27** (live `atlas-main`): 1311005 serves a nonzero `x` (5%→20% by level) on v48/61/72/79/83/84/87/92/jms185; **v95 has no skill data ingested** (every id 404s) — re-verify after its WZ data is loaded. Cost safely no-ops where `x ≤ 0`.
5. **Placement:** post-broadcast side-effects block (where the TODO sits), after the projectile emit — design §3.2 alternative A; per-monster `onDamageApplied` hook explicitly rejected.
6. **The generic `se.HPConsume()`/`se.MPConsume()` cast-cost block (:539-546, `handler.Lookup`-gated) is untouched** — Sacrifice is unregistered, so it pays both the flat cast cost and this damage-proportional cost (FR-9).
7. **Resilience:** `ChangeHP` errors → `Errorf` + swallow; nothing aborts the attack pipeline (MP Eater convention).
8. **Narrowing guard:** helper caps at `math.MaxInt16` so `-int16(cost)` cannot overflow even for out-of-spec HP (design §4.1 rule 4).

## Dependencies / interactions

- **No new fetches:** `c`, `se`, `cp` are already in scope in `processAttack`. Non-Sacrifice attacks pay one integer comparison.
- **No API/data/schema changes:** the HP change rides the existing atlas-channel → atlas-character CHANGE_HP command and its stat-update event/packet flow.
- **Out of scope:** the still-open neighboring TODOs (cooldown, dark-sight/wind-walk cancel, Meso Explosion meso-destroy, Bandit Steal, on-hit debuff procs…), Brawler MP Recovery (5101005). Note: combo orbs, Drain/Energy Drain/Vampire HP gain, and Pick Pocket already landed on main (`comboOrbTryUpdate`, `drainTryHeal`, `pickPocketTryProc`) — no longer TODOs, still not this task.

## Verification bar (CLAUDE.md)

`go test -race ./...`, `go vet ./...`, `go build ./...` clean in `services/atlas-channel/atlas.com/channel`; `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, and `tools/lint.sh --check` clean from the worktree root (the CLAUDE.md bar grew since this plan was first written). No `go.mod` change expected, so `docker buildx bake atlas-channel` is not mandatory. The per-version data check + in-game validation across the supported set is human-driven, post-deploy (PRD §10).
