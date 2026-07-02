# Processor Gen3 Unification — Design

Version: v1
Status: Approved for planning
Created: 2026-07-02
PRD: `docs/tasks/task-116-processor-gen3-unification/prd.md`

---

## 1. Summary

Converge every non-mock processor package under `services/` onto the Gen3 idiom (`Processor` interface + `ProcessorImpl` struct + `NewProcessor(...) Processor`), add a `mock/` package per converted package, and fix the CP-2 concrete-return signatures — strictly behavior-preserving. This document fixes the target pattern, the per-generation conversion recipe, the sequencing model, and resolves the PRD's three open questions plus one triage result discovered during design exploration.

## 2. Target pattern (canonical Gen3 shape)

Exemplar: `services/atlas-notes/atlas.com/notes/note/processor.go` and its `mock/processor.go`.

Every converted package ends in exactly this shape:

```go
type Processor interface {
    // every exported method implemented by ProcessorImpl
}

type ProcessorImpl struct {
    l   logrus.FieldLogger
    ctx context.Context
    // db *gorm.DB, t tenant.Model, producer producer.Provider,
    // collaborator processors — only the fields the package already needs
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context /*, extra deps */) Processor {
    return &ProcessorImpl{ ... }
}

var _ Processor = (*ProcessorImpl)(nil)
```

Rules, in decreasing order of strictness:

1. **Interface = complete exported method set.** Every exported method with a `ProcessorImpl` receiver appears in the interface. Exported package-level functions *without* a receiver (pure helpers, `Extract`, builders, event providers) stay package functions — they are not forced into the interface.
2. **`NewProcessor` returns `Processor`, never `*ProcessorImpl`.** Extra dependencies follow `l, ctx` positionally (matching `atlas-notes`: `NewProcessor(l, ctx, db)`).
3. **Conformance assertion lives in `processor.go`** (not a test file). Uniform location; compiles in the production build so it can never be skipped by a test-tag or missing test file. The three existing assertions in `atlas-messages` `*_test.go` files stay where they are — they already conform; relocating them is churn.
4. **Capture semantics are preserved, not normalized.** A package that resolves `tenant.MustFromContext(ctx)` per-call keeps per-call resolution; construction-time capture is only kept where it already exists (`MustFromContext` panics on missing tenant — moving the capture point changes failure timing, which violates behavior preservation). The same rule governs `producer.ProviderImpl` and collaborator-processor construction: keep the existing acquisition point.
5. **Method conventions unchanged**: buffered logic in `Method(mb *message.Buffer)`, side-effecting in `MethodAndEmit(...)`, providers as `XxxProvider(...) model.Provider[T]`. Curried inner signatures (`func(a) func(b) ...`) are preserved exactly as they exist today — this task does not flatten currying inside method bodies.

## 3. Conversion recipes

### 3.1 CP-2 signature fix (Tier A shape)

Files where the interface already exists but `NewProcessor` returns `*ProcessorImpl` (e.g. `services/atlas-pets/atlas.com/pets/pet/processor.go:96`):

1. Change the return type to `Processor`.
2. If callers use methods on the concrete type that are missing from the interface, extend the interface to the full exported method set (declaration change only).
3. Update any caller that declared a variable/struct field/function parameter as `*ProcessorImpl` → `Processor`.
4. Add the conformance assertion if absent.
5. Audit the package's existing `mock/` (all 12 CP-2 services): update `ProcessorMock` for any interface extension.

### 3.2 Gen2 → Gen3 (struct exists, no interface)

E.g. `services/atlas-channel/atlas.com/channel/macro/processor.go`:

1. Rename `type Processor struct` → `type ProcessorImpl struct` (all receivers follow). atlas-messengers ("Gen2.5") already uses the `ProcessorImpl` name — skip this step there.
2. Extract `type Processor interface` from the exported receiver-method set.
3. Change `NewProcessor(...) *Processor` (or `*ProcessorImpl`) → `NewProcessor(...) Processor`.
4. Update call sites within the service: vars, struct fields, and params typed `*Processor`/`*ProcessorImpl` → `Processor`. Test files constructing `&Processor{...}` literals switch to `NewProcessor(...)` (known gotcha: test files reference internal names; same-commit fix is mandatory).
5. Add assertion + `mock/processor.go`.

### 3.3 Gen1 → Gen3 (curried package functions, no type)

E.g. `services/atlas-data/atlas.com/data/npc/processor.go`, `services/atlas-monster-death/atlas.com/monster/monster/processor.go`:

1. Introduce `Processor` / `ProcessorImpl` / `NewProcessor(l, ctx /*, db*/)`.
2. Each exported curried function `F(l)(ctx)(args...)` becomes method `F(args...)`; the `l`/`ctx` (and `db`, where curried in, as in `RegisterNpc(db)(l)(ctx)(path)`) levels collapse into struct fields. Private curried helpers may collapse the same way or stay private package functions — implementer's choice per file, biasing toward methods when they use `l`/`ctx`.
3. Call-site rewiring: REST handlers, Kafka consumers, tickers, and `main.go` construct `NewProcessor(l, ctx, ...)` at the top of the handler and call methods — matching existing Gen3 handler wiring.
4. **Higher-order call sites use method values.** atlas-data passes curried functions as values (`registerAllInDirectory(l, ctx, dir, npc.RegisterNpc(db))` in `data/workers/npc.go`, similarly `data/processor.go`). Post-conversion these become method values: `npc.NewProcessor(l, ctx, db).RegisterNpc` with type `func(path string) error`, and the higher-order plumbing's parameter type simplifies accordingly. No logic moves.
5. Package-level registry/storage singletons (`GetModelRegistry()`, `NewStorage`) stay package-private helpers — the interface exposes only the operational surface.

### 3.4 Mocks (all tiers)

Per converted package, `<pkg>/mock/processor.go` following `services/atlas-notes/atlas.com/notes/note/mock/processor.go`:

- `type ProcessorMock struct` with one `XxxFunc` field per interface method (identical signature); each method delegates to the field when non-nil, else returns zero values.
- `var _ <pkg>.Processor = (*ProcessorMock)(nil)`.
- **Existing mocks are kept, not rewritten.** Where a package already has a differently-shaped mock (see §4.2 `FakeClient`), it is retained, updated to satisfy the (possibly renamed/extended) interface, and gains a conformance assertion. Rewriting working fakes to the func-field shape is churn with test-behavior risk.
- Mocks are hand-written. A mock code generator was considered and rejected (§7.1).

## 4. Resolved open questions and triage results

### 4.1 PR strategy (PRD §9.1) — one PR, per-service commits

**Decision: option (a)** — a single PR from this branch, with one commit per service (per package-group for atlas-channel and atlas-data), ordered by tier.

Rationale: the change is mechanical and its correctness is grep-verifiable (PRD §10 acceptance greps) plus machine-verified (test/vet/build/bake), so review effort scales with the *pattern*, not the line count; per-service commits give bisectability and commit-by-commit review. Sequential per-tier PRs (option b) would cost two extra review/CI/rebase round-trips and force the branch to straddle merges while four other worktrees (task-102/111/113/114/115) are in flight. Fallback: if review proves unwieldy, tier boundaries fall on commit boundaries, so a per-tier PR split can be cut by rebase at PR time without rework — consistent with the one-worktree-per-task rule.

### 4.2 `atlas-configurations/configuration/data` triage (PRD §9.2) — rename-only conversion, keep the fake

Exploration result: the package is a REST **client** abstraction, not a processor — `type Client interface` + `ClientImpl` + `NewClient(l) *ClientImpl`, methods taking `ctx` per call, with a stateful map-based `FakeClient` mock. It is constructed once at route-registration time (`tenants/resource.go:87`, `templates/resource.go:122`) where no request/tenant context exists, and receives `ctx` per method call. Forcing full Gen3 (`ctx` in the struct, per-request construction) would change its lifecycle and is exactly the failure-timing change §2.4 forbids.

**Decision:** rename-only conversion to satisfy the acceptance greps and naming uniformity:

- `Client` → `Processor`, `ClientImpl` → `ProcessorImpl`, `NewClient(l)` → `NewProcessor(l logrus.FieldLogger) Processor`.
- Methods keep their per-call `ctx` parameter — documented in `inventory.md` as the one sanctioned shape deviation (long-lived, wired-at-startup processor; ctx-per-call).
- `FakeClient` → `ProcessorMock` name-wise but keeps its map-based design (per §3.4 keep-existing-mocks rule); gains the conformance assertion. Its tests update in the same commit.

### 4.3 atlas-monster-death triage — it is a Gen1 service; move to Tier C

Exploration result: of its 6 processor files, only `data/equipment/statistics/processor.go` is Gen2; the other 5 (`party`, `monster`, `character`, `monster/drop`, `monster/drop/position`) are Gen1 curried functions. The PRD's Tier B placement ("1 Gen2 + triage of 5") is corrected: the whole service converts in **Tier C** in a single visit (recipe §3.3 for the five, §3.2 for statistics). Its `monster` package (drop success evaluation, experience-distribution standard-deviation logic) is the design's leading candidate for FR-6.3 characterization tests.

### 4.4 atlas-data interface granularity (PRD §9.3) — one `Processor` per package, no exceptions

Confirmed: even single-function packages get the full `Processor`/`ProcessorImpl`/`NewProcessor` triple. Uniformity over minimalism — the acceptance greps and the reviewer checklist depend on there being exactly one shape to check. A shared generic interface in `libs/` was considered and rejected (§7.2).

## 5. Sequencing: one visit per service

The PRD's tiers classify the *work*; the schedule visits each **service exactly once** and does all of its FR-2/FR-3/FR-4 work in that visit. This halves verification cost for the multi-tier services (atlas-channel would otherwise be baked and reviewed twice) and keeps cross-package collaborator updates (e.g. monster-death's `drop` → `statistics` dependency) inside one commit series.

Visit order (risk ramp preserved — pure signature fixes first, heaviest call-site rewiring last):

| Phase | Services (in order) | Work per visit |
|---|---|---|
| A — signature fixes | atlas-doors, atlas-summons, atlas-saga-orchestrator, atlas-portal-actions, atlas-pets, atlas-npc-conversations, atlas-mounts, atlas-map-actions | §3.1 only; one commit per service |
| B — Gen2 extraction | atlas-chairs, atlas-storage, atlas-messengers, atlas-configurations (incl. §4.2), atlas-consumables, atlas-login (CP-2 + Gen2), atlas-inventory (CP-2 + Gen2), atlas-channel (CP-2 + Gen2, committed in package groups) | §3.1 + §3.2 per visit |
| C — Gen1 modernization | atlas-portals, atlas-reactors, atlas-asset-expiration, atlas-monster-death (per §4.3), atlas-data (package-by-package, last) | §3.3 (+§3.2 for monster-death) |

Within B and C, small services go first to shake out the recipe before the big units. atlas-channel commits in package groups of roughly 5–8 related packages (`data/*` together; social packages together; etc. — exact grouping fixed at plan time) with a green module build between groups. atlas-data converts package-by-package with a green build between packages (PRD FR-4 mandate); its `data` orchestrator package and `workers` higher-order call sites (§3.3.4) convert last, after all 19 leaf packages.

The FR-8 inventory scan (`inventory.md`, classification greps checked into the task folder) runs before phase A, after each phase, and at the end; it — not this document or the PRD prose — is the authoritative file list.

## 6. Testing and verification

- **Existing tests** update in the same commit as their package (internal-name references break on rename).
- **Conformance assertions** (impl in `processor.go`, mock in `mock/processor.go`) are part of the recipe, not a separate pass.
- **Characterization tests** (FR-6.3): during each visit, the implementer classifies each converted package as *delegation shim* (one-line registry/requester/producer calls — no new tests) or *logic-bearing* (branching, state transitions, emission batching). Logic-bearing packages with no existing coverage get characterization tests written against pre-conversion behavior, committed with (or immediately before) the conversion, passing unchanged after. The running list lives in `inventory.md`. Known candidates from exploration: atlas-monster-death `monster` (drop-rate evaluation, exp-distribution stddev math) and `monster/drop` (random equipment stat generation — seed/spread preservation, position calculation).
- **Test setup** uses the Builder pattern; no `*_testhelpers.go` constructors.
- **Verification cadence:** per commit — `go test -race ./...`, `go vet ./...`, `go build ./...` in the touched module. Per service visit (after its last commit) — `docker buildx bake atlas-<svc>`. Per phase end and at branch end — `tools/redis-key-guard.sh` from the repo root, full inventory re-scan, and a re-run of every touched service's bake. No `go.mod` files change, but the bake step stays mandatory per CLAUDE.md.
- **Code review before PR:** `superpowers:requesting-code-review` → `backend-guidelines-reviewer` + `plan-adherence-reviewer`, findings to `docs/tasks/task-116-processor-gen3-unification/audit.md`.

## 7. Alternatives considered

1. **Generated mocks (mockgen/moq or a bespoke generator).** Rejected: the repo's 68 existing mocks are hand-written func-field mocks with curried zero-value returns that off-the-shelf generators don't produce; introducing a generator creates a second mock dialect and a new build dependency for a one-time task. Hand-writing follows the existing convention exactly.
2. **A shared generic `Processor` contract in `libs/`.** Rejected: processor method sets are entirely package-specific; there is no useful common abstraction, and the PRD excludes `libs/` changes. Per-package interfaces are the established Gen3 convention.
3. **Big-bang conversion (one commit per tier or per repo).** Rejected: unbisectable, unreviewable, and a single compile error blocks everything. Per-service (per-package-group for the two giants) commits with green verification between is the PRD's FR-7 cadence and survives interruption.
4. **Sequential per-tier PRs.** Rejected as the default for the reasons in §4.1; preserved as a zero-rework fallback because tier boundaries land on commit boundaries.
5. **Exempting `configurations/data` as a "client, not a processor."** Rejected in favor of the §4.2 rename-only conversion: an exemption leaves a permanent asterisk on the acceptance criteria and the reviewer checklist for a package whose rename costs a few call sites; the lifecycle-preserving rename gets full uniformity of names without the behavior risk of full Gen3.
6. **Flattening curried method signatures during conversion.** Rejected: tempting while touching every file, but it multiplies the diff, breaks the `message.Buffer`/`model.Flip` composition idiom, and converts a rename-refactor into a logic refactor. Out of scope; the currying stays.

## 8. Risks and mitigations

| Risk | Mitigation |
|---|---|
| Merge conflicts with in-flight worktrees (task-102/111/113/114/115) — this branch touches ~30 services | Land promptly after review; per-service commits make conflict resolution local; rebase before PR; coordinate merge order with in-flight branches that touch the same services (atlas-channel is the hot spot) |
| Test files constructing `&Processor{...}` or calling unexported helpers break on rename | Same-commit test updates are part of the recipe (§3.2.4); `go test -race` per commit catches stragglers before the next visit |
| Interface extension (CP-2 step 2) accidentally exports a method that was intended private-by-omission | The interface mirrors the *existing* exported method set only; extension happens solely when a caller already uses the method on the concrete type — i.e., it was already de-facto public |
| Tenant/producer capture-point drift during Gen1 collapse changes failure timing | §2.4 rule: keep the existing acquisition point; reviewer checks this explicitly (it is a DOM-checklist concern) |
| atlas-data's wide fan-in (REST resource layer, workers, orchestrator) breaks mid-conversion | Leaf packages first with green builds between; orchestrator/`workers` call sites converted last in one commit (§5); method-value passing keeps higher-order wiring shape-compatible |
| Point-in-time counts in the PRD drift from reality | FR-8 inventory scan is the ground truth; re-run per phase; acceptance is defined against the final scan |

## 9. Acceptance

PRD §10 criteria apply unchanged, evaluated against the final `inventory.md` scan. Design-level additions: the single sanctioned shape deviation is `atlas-configurations/configuration/data` (ctx-per-call methods, §4.2), documented in `inventory.md`; expected exemptions from the `type Processor interface` criterion: none.
