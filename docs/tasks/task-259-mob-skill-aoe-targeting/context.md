# task-259 — Implementation Context

Companion to `plan.md`. Everything here was established by reading the code at plan
time; an implementer should not have to rediscover it.

## Scope in one line

Rewrite `atlas-monsters`' shared mob-skill disease selector so AoE targets come from the
WZ bounding box rather than the whole field, cap only SEDUCE, and select deterministically.

## Module and worktree

- Worktree: `.worktrees/task-259-mob-skill-aoe-targeting`, branch `task-259-mob-skill-aoe-targeting`
- Every `go build` / `go test` in the plan runs from `services/atlas-monsters/atlas.com/monsters` (module `atlas-monsters`)
- No other service is modified. `atlas-character` and `atlas-maps` are consumed only.

## Key files

| File | Role |
|---|---|
| `services/atlas-monsters/atlas.com/monsters/monster/processor.go` | The whole change's centre of gravity. `ProcessorImpl` struct + seams (99-101), `NewProcessor` wiring (113-121), `executeDebuff`/`executeBanish`/`executeDispel` (1216-1277), the old `getDiseaseTargets` (1279-1305) |
| `services/atlas-monsters/atlas.com/monsters/monster/disease_targets.go` | New. Pure selector (Task 2) + I/O shell and fan-out (Task 3) |
| `services/atlas-monsters/atlas.com/monsters/character/position/` | New. Read-only `atlas-character` client |
| `services/atlas-monsters/atlas.com/monsters/monster/mobskill/builder.go` | Gains `SetCount` — tests need it and it does not exist today |
| `services/atlas-monsters/atlas.com/monsters/monster/information/builder.go` | Gains `SetBanish` — the banish test needs a non-zero `Banish().MapId` |
| `services/atlas-maps/atlas.com/maps/character/` | Read-only pattern source for the new position client |

## Decisions carried in from the design

- Per-character `GET /characters/{id}` with a bounded fan-out (8 in flight), not a bulk
  query. If live latency ever bites, the fix is a bulk filter on `atlas-character` and only
  `resolvePositions` changes — the pure selector and its tests are untouched by construction.
- Stdlib `sync.WaitGroup` + buffered-channel semaphore, **not** `errgroup`.
  `golang.org/x/sync` is an indirect dependency today (`go.mod:40`) and
  `errgroup.WithContext` cancels siblings on first error — the opposite of the
  degrade-don't-abort rule.
- No position cache. Positions change continuously during combat; the fan-out already
  collapses a cast into one round-trip's wall clock.
- No dead-character (`hp`) filter, so `hp` is not projected in the new `RestModel`.
- No facing-direction mirroring of the rectangle.

## Discoveries made at plan time (not in the design)

These are real gaps between the design's stated test strategy and the code as it stands.
Each is folded into a plan task; none is optional.

1. **`mobskill.ModelBuilder` has no `SetCount`.** `Model.count` exists and `Count()` reads
   it, but the builder never sets it, so no test could construct a capped skill. Added in
   Task 2.
2. **`executeDebuff`/`executeBanish`/`executeDispel` publish through
   `producer.ProviderImpl` directly, not through the `p.emit` seam.** The design's Layer-2
   and Layer-3 tests assume the emit seam. Task 4 routes those three call sites through
   `p.emit` — topic constants and provider expressions unchanged, and `NewProcessor`
   already binds `emit` to `producer.ProviderImpl`, so production behavior is identical.
3. **`executeBanish` calls `information.NewProcessor(...).GetById` directly**, unlike the
   five other information lookups in the file which honour the `testInformationLookup`
   override. Task 4 adds the same guard, copying the form at `monster/processor.go:1692-1696`.
4. **`information.ModelBuilder` has no `SetBanish`**, so a banish test cannot get past the
   `banishMapId == 0` early return. Added in Task 4.
5. **`monster.Model` has no `NewModelBuilder`.** `Clone(Model) *ModelBuilder` is the only
   builder entry point; `Clone(Model{}).SetX(...)...Build()` is how a test builds one from
   scratch. Registry-backed construction (`r.CreateMonster`) is unnecessary here because
   `getDiseaseTargets` reads only `X()`, `Y()`, `Field()`, `UniqueId()`, and
   `ControlCharacterId()`.
6. **`math/rand` must stay imported.** `rand.Intn` at `monster/processor.go:709` drives the
   basic-attack damage formula. Only `rand.Shuffle` at line 1300 leaves. The PRD's
   acceptance criterion is conditional ("removed if it has no other use") — it has one.

## Deliberately out of scope

**The PRD's final acceptance criterion — updating
`docs/research/missing-features/monsters-and-bosses.md` §8 and its "Unverified / needs
deeper data" bullet — is not in this plan.** That file does not exist on this branch: it is
an untracked, uncommitted file in the main working tree (`git status` shows it as `??`).
No implementer inside this worktree can edit it, and copying it onto the branch would land
an unrelated in-progress research document in this PR. Confirmed with the user at plan
time: drop the doc task, and make the §8 edit in the main working tree separately.

The substance of that edit, so it is not re-derived: box-scoped AoE targeting, SEDUCE-only
capping, and deterministic ordering now match the reference server; GMS-canonical seduce
*ordering* and facing-direction mirroring remain **unverified** and still need an IDA/WZ
pass. The task changes what is implemented, not what is proven.

## Task sizing

Four tasks, each one service and 3-5 files. None is deliberately oversized.

- Task 1 (position client) is independent and can run first or in parallel with Task 2.
- Task 2 (pure selector) is independent of Task 1.
- Task 3 consumes both and must run after them.
- Task 4 must run after Task 3 (it depends on the `skillId` signatures Task 3 introduces).

Tasks 2-4 all touch `monster/disease_targets.go` or `monster/processor.go`, so they must
run sequentially, not in parallel.

## Open questions unchanged by this task

- GMS-canonical seduce target ordering (unverified — reference parity implemented as a
  documented approximation).
- Whether the v83 client mirrored the skill rectangle by mob facing (unverified — not
  mirrored, matching the reference).
- How stale `atlas-character`'s `x`/`y` can be during active combat. This bounds achievable
  fidelity regardless of implementation quality.
