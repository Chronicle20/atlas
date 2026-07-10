# Task 152 — Mortal Blow: Implementation Context

Companion to `plan.md`. Everything an implementer (or reviewer) needs that isn't a plan step.

## What this is

Server half of pre-Big-Bang Mortal Blow (Ranger 3110001 / Sniper 3210001): a ranged attack arriving with one of those skill ids rolls, per damaged non-boss monster, an instant kill — monster HP ≤ `maxHP·x/100` (pre-attack snapshot), then roll 1–100 ≤ `y`. `x`/`y` come from the tenant skill effect at the character's owned level. The kill is delivered through atlas-monsters' standard damage path so EXP/drops credit normally. The client half (point-blank detection, `prop` success roll, damage%) is entirely client-owned and IDA-verified — see PRD §4.

## Key documents

- `docs/tasks/task-152-mortal-blow/prd.md` — requirements; §4 is the IDA-verified client behavior contract (ground truth).
- `docs/tasks/task-152-mortal-blow/design.md` — nine numbered decisions; the plan implements them 1:1 except one declared refinement (below).

## Key files

| File | Role |
|---|---|
| `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go` | Proc home: helpers, `mortalBlowDeps`, `mortalBlowTryProc`, `onDamageApplied` wiring. MP Eater (task-049) in the same file is the structural template. `// TODO Mortal Blow` (line ~421) gets deleted. |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_projectile.go` | FR-6 comment fix only (~line 118) — Mortal Blow DOES consume an arrow. |
| `services/atlas-channel/atlas.com/channel/data/skill/effect/{model,rest}.go` | `y` already threads REST→model (`rest.go:42,115`); only the `Y()` accessor is added. |
| `services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka.go` | `CommandTypeKill` + `KillCommandBody{CharacterId, SkillId}`. |
| `services/atlas-channel/atlas.com/channel/monster/{producer,processor}.go` | `KillCommandProvider` (keyed by monster unique id) + `Processor.Kill`. |
| `services/atlas-monsters/atlas.com/monsters/monster/processor.go` | `Damage` split into `checkReflect` + `damageCore`; new `Kill(uniqueId, characterId, skillId)` with fail-closed boss guard, delivering `[]uint32{math.MaxUint32}` via `damageCore`. |
| `services/atlas-monsters/atlas.com/monsters/monster/registry.go:427-483` | `ApplyDamage` — clamps recorded damage to remaining HP. This is why `MaxUint32` keeps the damage summary honest (Decision 6's verify-then-pick, resolved at plan time). |
| `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/{kafka,consumer}.go` | Mirrored `killCommandBody` + type-gated `handleKillCommand`. |

## Decisions already made (do not relitigate)

1. **Dedicated `KILL` command** (not `DAMAGE` reuse, not channel-side boss lookup) — atlas-monsters owns the boss guard because it owns the boss flag; `DRAIN_MP` (task-049) is the precedent. Same partition as `DAMAGE` (keyed by monster unique id) ⇒ kill processes after the triggering attack; if the attack already killed the monster, `Kill` finds it gone and drops silently.
2. **Boss guard is FAIL-CLOSED** — info-lookup error ⇒ drop the kill. Deliberately diverges from `DrainMp`'s fail-open (FR-4: boss kill must be impossible even if the channel misfires).
3. **Threshold reads the channel's pre-attack snapshot** — damage propagates to atlas-monsters async, so the snapshot hasn't absorbed the current attack. This is Cosmic parity (its check runs before damage application) and is the *specified* behavior, not a bug.
4. **No job-range check, no extra effect lookup** — the attack's skill IS the passive: `se` is already resolved at owned level by `processAttack`, and unowned skill ids already destroy the session (forgery guard).
5. **Kill line = `math.MaxUint32`** — Cosmic parity; safe because `ApplyDamage` clamps (verified). The design's fallback (`m.Hp()`) is NOT needed.
6. **No HP/MP-on-kill, no packets, no projectile behavior change** — PRD non-goals; post-BB Mortal Blow is a different mechanic. v95+/JMS clients never send these ids, so the feature is naturally inert there.
7. **RNG seam = pure helpers + injected roll func** — no rand interface; matches `mpEaterShouldProc(prop, roll)`.

## Declared deviation from design.md

Decision 5 wrote `mortalBlowTryProc(l, mp *monster.Processor, se, …)`. The plan uses a `mortalBlowDeps{getMonster, emitKill, roll}` struct instead, wired from `mp.GetById` / `mp.Kill` / `rand.Intn(100)+1` at the call site. Reason: design §5 requires failure-isolation tests (snapshot-fetch error, emit error) which cannot be written against the concrete `monster.Processor` (live REST/Kafka); `damageInfoEntryDeps` in the same file is the established seam pattern. Flow and production behavior are exactly Decision 5's.

## Dependencies & test seams

- Constants: `skill3.RangerMortalBlowId`/`SniperMortalBlowId` exist (`libs/atlas-constants/skill/constants.go:3097,3119`); `RangerStrafeId` (`:860`) used as the negative-case skill in tests.
- Channel monster snapshots for tests: `monster.NewModelBuilder(uniqueId, f, monsterId).SetHp(..).SetMaxHp(..).MustBuild()` — only invariant is non-zero uniqueId.
- Channel effect for tests: `effect.Extract(effect.RestModel{X: 20, Y: 5})`.
- Monsters-side tests: `newRecordingProcessorWithBodies` (`processor_test.go:234`) intercepts `emit`; `testInformationLookup` (`processor.go:68`) stubs the boss lookup; `information.NewModelBuilder().SetBoss(..).Build()`.
- Existing kill-path tests pin DAMAGED→KILLED event order — the Task 6 refactor must keep them green untouched.

## Task order & dependency graph

Tasks 1→5 are channel-side, 6→8 monsters-side, 9 verification. Hard edges: 2→4 (helpers), 3→4 (`mp.Kill`), 6→7 (`damageCore`), 7→8 (`Kill`). Tasks 1–5 and 6–8 are independent module groups; within the constraint edges, 1 can land any time before 4 (Task 4's `se.Y()` needs it).

## Verification gate (CLAUDE.md, non-negotiable)

`go test -race ./...`, `go vet ./...`, `go build ./...` in BOTH modules; `docker buildx bake atlas-channel atlas-monsters` from the worktree root; `tools/redis-key-guard.sh`. Then `superpowers:requesting-code-review` BEFORE any PR.
