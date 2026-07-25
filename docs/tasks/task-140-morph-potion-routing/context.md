# Morph Potion Routing — Execution Context

Task: task-140-morph-potion-routing
Plan: `docs/tasks/task-140-morph-potion-routing/plan.md` (6 tasks)
Worktree: `.worktrees/task-140-morph-potion-routing`, branch `task-140-morph-potion-routing`

## What this task does

Classification-221 (transformation) use-tab potions currently fall through to `ConsumeBare` — item decrements, no effect. The morph applier already exists in `ApplyItemEffects`. This task (1) routes 221 through `ConsumeStandard`, (2) adds weighted-random selection for the `morphRandom` spec, (3) extracts a pure `computeEffectPlan` from `ApplyItemEffects` so the behavior is unit-testable, (4) adds a `Morphs()` getter on the data model. Only `atlas-consumables` changes.

## Key files

All under `services/atlas-consumables/atlas.com/consumables/` (the Go module root — run test/vet/build from there):

| File | Role |
|---|---|
| `consumable/processor.go` | `usesStandardConsumer` (line ~77, routing switch), `collectCureTypes` (~89), `ApplyItemEffects` (~112, refactor target), fixed-morph branch (~172-174). NOTE: imports `math/rand` as `rand` — that's why `morph.go` is a separate file. |
| `consumable/morph.go` | NEW — `selectMorph` (pure) + `rollMorph` (crypto/rand). |
| `consumable/processor_test.go` | Existing tests incl. `TestUsesStandardConsumer` table (~321) and the `makeCureModel` helper. New plan/morph tests append here; new `extractConsumable` + `discardLogger` helpers. |
| `data/consumable/model.go` | `Model.morphs` field exists (~104); add `Morphs()` getter after `MonsterSummons()` (~186). |
| `data/consumable/rest.go` | `RestModel.Morphs` (~70) already deserialized; `Extract` (~92) already populates `morphs` (~163). Test fixtures go RestModel-literal → `Extract` (design §4.4; no test-only constructors). |
| `character/model.go` | `NewModelBuilder()` (365), `SetMaxHp`/`SetMaxMp`, `Build()` (403) — character fixtures for hpR/mpR math. |
| `character/buff/stat/model.go` | `stat.Model{Type character.TemporaryStatType; Amount int32}`. |

Shared constant: `item.ClassificationConsumableTransformation = Classification(221)` at `libs/atlas-constants/item/constants.go:39` (imported as `item2` in processor.go, `item` in the test file).

## Key decisions (from approved design)

- **Randomness seam = task-131 precedent, pattern only.** task-131's `consumable/reward.go` lives on the unmerged `task-131-random-reward-items` branch — it is NOT in this worktree, so there is no code dependency and no merge-order coupling. We replicate its shape (dedicated pure-helper file, `crypto/rand` roll, no seeded PRNG); determinism under test comes from exhaustively enumerating rolls against the pure `selectMorph`.
- **`selectMorph` sorts morph ids ascending** — Go map iteration is randomized; sorting is what makes selection a deterministic function of the roll.
- **FR-7 precedence is structural**: `if morph > 0 { fixed } else if len(Morphs()) > 0 { roll }` — double-apply impossible by construction.
- **`computeEffectPlan` is a pure move**: ordered `hpChanges`/`mpChanges` slices preserve the exact per-call `ChangeHP`/`ChangeMP` sequence; execution order in `ApplyItemEffects` stays cures → HP/MP → single `bp.Apply`. The task-051 D3 cure-ordering comment moves with the code. `ApplyItemEffects` is shared with NPC-initiated `ApplyConsumableEffect` — the T8 regression tests exist to protect that path.
- **Zero-total morph table**: warn + skip only the morph statup; other specs apply, consumption stands (matches existing "errors logged, consumption not rolled back" semantics).
- **Routing lands last (Task 5)** so intermediate commits never ship 221 on a half-wired path.
- **200/201/202/205 stay as raw `Classification(n)` literals** — no named constants exist for them; renaming is explicitly out of scope.

## Out of scope (do not implement)

- 2212000 "morph another player" packet flow (`SendRandomMorphOtherRequest`/`OnRandomMorphRes`) — the v83 client intercepts 2212xxx double-clicks before any use-item packet; filed as a follow-up backlog note in Task 6 (still route 221 uniformly, including 2212000).
- Cash morph coupons (classification 530); anti-cheat for attacking while morphed; morph-cancel-on-hit (verified non-mechanic); any packet/writer changes. Death cancellation already exists via the respawn saga `CancelAllBuffs`.

## Verification & gotchas

- Per-task: `go test -race ./consumable/` (or `./data/consumable/`) from the module root. Final: `go test -race ./...`, `go vet ./...`, `go build ./...` clean, plus `tools/redis-key-guard.sh` from the worktree root (never with a `GOWORK=off` prefix).
- No `go.mod` change expected → no docker bake; if go.mod changes anyway, `docker buildx bake atlas-consumables` from the worktree root is mandatory.
- Diff-scope acceptance check: `git diff --name-only $(git merge-base HEAD origin/main)..HEAD` → only `services/atlas-consumables/**` + `docs/tasks/task-140-morph-potion-routing/**`.
- The item-effects backlog doc for the Task 6 follow-up note is UNTRACKED in the main repo checkout (`../../docs/research/missing-features/items-and-consumables.md` from the worktree root); edit it there, do not commit it on this branch.
- testify `v1.11.1` already in go.mod; test files are package-internal (`package consumable`) so unexported `computeEffectPlan`/`selectMorph` are reachable.
- Morph-table values in tests are synthetic fixtures shaped like the real data (three entries summing to 100 for the exhaustive test, plus a non-100 table) — do not present them as WZ-verified values.
- Code review (`superpowers:requesting-code-review`) is mandatory before the PR.
