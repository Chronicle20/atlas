# Task 116 — Processor Gen3 Unification: Execution Context

Companion to `plan.md`. Read this before executing any task.

## What this task is

A strictly behavior-preserving refactor converging every non-mock processor package under `services/` onto the Gen3 idiom (`Processor` interface + `ProcessorImpl` struct + `NewProcessor(...) Processor` + `mock/processor.go`). No logic moves. ~130 processor files across ~30 services.

## Key files

| File | Role |
|---|---|
| `docs/tasks/task-116-processor-gen3-unification/prd.md` | Requirements (FR-1..FR-8, acceptance criteria §10) |
| `docs/tasks/task-116-processor-gen3-unification/design.md` | Approved design: target pattern §2, recipes §3, resolved questions §4, sequencing §5 |
| `docs/tasks/task-116-processor-gen3-unification/plan.md` | The implementation plan (37 tasks); recipes R1–R7 defined once at the top |
| `docs/tasks/task-116-processor-gen3-unification/inventory.md` | **Ground truth** (created by Task 1): per-file classification, statuses, sanctioned deviations, characterization-test list, deferred findings |
| `services/atlas-notes/atlas.com/notes/note/processor.go` | Gen3 exemplar (interface/impl/constructor shape) |
| `services/atlas-notes/atlas.com/notes/note/mock/processor.go` | Mock exemplar (func-field `ProcessorMock`, nested zero-value closures for curried methods) |
| `services/atlas-channel/atlas.com/channel/macro/processor.go` | Gen2 worked example (recipe R2, converted in plan Task 22) |
| `services/atlas-rates/atlas.com/rates/buffs/processor.go` | Gen1 worked example (recipe R3, converted in plan Task 30) |
| `services/atlas-pets/atlas.com/pets/pet/processor.go` | CP-2 worked example incl. the `With(opts) *ProcessorImpl` option-pattern wrinkle (recipe R1, Task 5) |

## Decisions already made (do not re-litigate)

1. **One PR, per-service commits** (design §4.1); per-package-group for atlas-channel, per-package for atlas-data. Tier boundaries fall on commit boundaries so a per-tier PR split remains a rebase-only fallback.
2. **One visit per service** (design §5): a service's CP-2 + Gen2 + Gen1 work all lands in that service's visit, regardless of the PRD's tier tables.
3. **`atlas-configurations/configuration/data` and `atlas-character-factory/data`** get the rename-only R4 conversion (ctx-per-call REST clients; full Gen3 would change lifecycle/failure timing). These are the only sanctioned shape deviations.
4. **Interface granularity:** every package gets the full `Processor`/`ProcessorImpl`/`NewProcessor` triple, even single-function packages (design §4.4 — uniformity over minimalism).
5. **Mocks are hand-written**, following the existing func-field convention; existing mocks/fakes are kept and adapted, never rewritten (design §3.4, §7.1).
6. **Capture semantics preserved** (design §2.4): never move a `tenant.MustFromContext` / `producer.ProviderImpl` / collaborator-construction point.
7. **Currying preserved** (design §7.6): only the outer `F(l)(ctx)` levels of Gen1 functions collapse; `message.Buffer`/provider currying inside methods is untouched.
8. **Non-processor `processor.go` files are renamed, not exempted** (plan R6): `atlas-messages/command` → `types.go`; `atlas-gachapons/test` and `atlas-npc-shops/test` → `fixtures.go`.
9. **atlas-monster-death is a Gen1 service** (design §4.3), converted whole in Phase C with characterization tests first (plan Task 31 has the actual test code).

## Scan findings beyond the PRD (plan-time, commit fde55e232)

The live FR-8 scan found work the PRD's tables missed — all in scope per FR-4's "newly revealed files convert under the matching recipe":

- **New services:** atlas-reactor-actions (`script` Gen2.5), atlas-account (`ban` Gen1), atlas-monsters (4 Gen1), atlas-rates (5 Gen1), atlas-character-factory (`data` R4 client), atlas-messages / atlas-gachapons / atlas-npc-shops (R6 renames).
- **New packages in listed services:** atlas-channel `server` (Gen1, wide test fan-in — plan Task 23), atlas-messengers `invite` (Gen1 — Task 13), atlas-map-actions / atlas-portal-actions `script` (Gen2.5 — Tasks 10/11).
- Plan-time counts: 20 CP-2, 58 Gen2, 5 Gen2.5, 50 Gen1 (45 R3 + 2 R4 + 3 R6). Task 1's committed scan supersedes these numbers.

## Dependencies and hazards

- **In-flight worktrees** task-102 (MTS), task-111 (resurrection), task-113 (gms-legacy), task-114 (outbox), task-115 (safe-goroutine) may touch the same services; **atlas-channel is the hot spot**. Rebase before PR; per-service commits keep conflicts local.
- **Test files reference internal names** — renames break them; same-commit test updates are mandatory (known project gotcha).
- **atlas-data has wide fan-in** (REST resource layer, `data` orchestrator, `workers` higher-order plumbing). Leaf packages first with green builds between; orchestrator + workers last (plan Tasks 32→35). Higher-order call sites become method values (`npc.NewProcessor(l, ctx, db).RegisterNpc`).
- **Verification per CLAUDE.md**: `go test -race` / `go vet` / `go build` per commit in the touched module; `docker buildx bake atlas-<svc>` per service visit (mandatory even though no `go.mod` changes); `tools/redis-key-guard.sh` per phase end.
- **Module names are short** (e.g. `atlas-monster` style) — check each `go.mod` `module` line before writing import paths in new mock/test files; never assume the full repo path.
- Go workspace: `go.work` at repo root covers all modules; run module commands from the module dir (the directory containing `go.mod`, e.g. `services/atlas-doors/atlas.com/doors`).

## Acceptance (evaluated in plan Task 36 against the final inventory scan)

- Zero `NewProcessor(...) *ProcessorImpl` outside mocks; zero `type Processor struct` outside mocks (exact greps in plan).
- Every non-mock `processor.go` package declares `type Processor interface`; deviations limited to the two R4 clients (documented) — the R6 files are no longer named `processor.go`.
- Every converted package: `var _ Processor = (*ProcessorImpl)(nil)` + `mock/processor.go` with `var _ <pkg>.Processor = (*ProcessorMock)(nil)`.
- All builds/tests/vet/bakes/redis-key-guard clean; no diff outside `services/` + the task folder.
- Code review (`superpowers:requesting-code-review`) before any PR.
