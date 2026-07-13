# Processor Gen3 Unification — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-02
---

## 1. Overview

The Atlas codebase contains 377 non-mock `processor.go` files across ~50 Go services, spanning three coexisting generations of the Processor idiom (architectural-improvements CP-1) plus a constructor-signature defect within the modern generation (CP-2):

- **Gen1** — package-level curried functions with no Processor type at all: `func GetById(l) func(ctx) func(id) (Model, error)`. 28 files across 4 services (atlas-data ×19, atlas-asset-expiration ×5, atlas-portals ×2, atlas-reactors ×2).
- **Gen2** — a concrete `type Processor struct` with methods but no interface: 60 files across 8 services (atlas-channel ×26, atlas-consumables ×16, atlas-inventory ×9, atlas-configurations ×3, atlas-storage ×2, atlas-chairs ×1, atlas-login ×1, atlas-monster-death ×1). A variant ("Gen2.5") in atlas-messengers ×2 names the struct `ProcessorImpl` but defines no interface.
- **Gen3 (canonical)** — `type Processor interface` + `type ProcessorImpl struct` + `func NewProcessor(l, ctx, ...) Processor`, with an optional `mock/` package. Exemplar: `services/atlas-notes/atlas.com/notes/note/processor.go`. 49 services already define at least one Gen3 interface; 68 `mock` processor files exist.
- **CP-2 defect** — 20 `NewProcessor` signatures across 12 services return `*ProcessorImpl` instead of the interface, making the interface decorative: callers bind to the concrete type and mocks cannot substitute.

The cost is concrete: there is no uniform mocking seam (tests in Gen1/Gen2 packages cannot substitute collaborators), developers must relearn the idiom per service — sometimes per package — and the `backend-guidelines-reviewer` DOM checklist flags drift it cannot retroactively fix. This task converges every processor package in the tree onto Gen3, adds a `mock/` package for every converted processor, and fixes the CP-2 signatures — strictly behavior-preserving.

## 2. Goals

Primary goals:
- Every non-mock processor package in `services/` defines `type Processor interface`, implements it with `type ProcessorImpl struct`, and constructs it via `NewProcessor(...) Processor` (interface return type).
- Zero `func NewProcessor(...) *ProcessorImpl` signatures outside `mock/` packages.
- Zero `type Processor struct` (concrete-named) declarations outside `mock/` packages.
- Every converted package gains a `mock/processor.go` following the established `ProcessorMock` func-field convention (see §4 FR-5).
- All existing tests pass unmodified in behavior; call sites updated mechanically in the same commit as the package they call.
- Functional coverage exists for converted logic: where a converted package's non-trivial behavior has no existing test, characterization tests are added before/with the conversion.

Non-goals:
- No behavior changes of any kind — no new features, no bug fixes discovered en route (file those separately), no changes to Kafka topics, REST routes, or packet handling.
- No renaming of packages or inner module directories (CP-6 is a separate concern).
- No documentation of the state-vs-action service-pair ownership boundary (the docs half of CP-7). The generation-alignment half of CP-7 is subsumed by this task automatically.
- No CI analyzer/lint guard for the pattern — `backend-guidelines-reviewer` (DOM checklist) remains the enforcement mechanism.
- No changes to `libs/` — this is a services-only refactor.
- No conversion of non-processor curried helpers (writers, providers, `kafka/producer` wrappers); only the Processor idiom is in scope.

## 3. User Stories

- As a service developer, I want every collaborator processor to be an interface so that I can substitute a `ProcessorMock` in unit tests without spinning up registries, databases, or Kafka.
- As a new contributor, I want one Processor idiom across all services so that knowledge from one service transfers to every other service.
- As a code reviewer (human or `backend-guidelines-reviewer`), I want the canonical pattern to be the only pattern so that drift is a visible diff, not the status quo.
- As a maintainer of cross-service refactors, I want constructors to return interfaces so that decorators/instrumentation can wrap processors without touching call sites.

## 4. Functional Requirements

### FR-1: Canonical pattern (target state)

Every processor package MUST match the Gen3 shape, modeled on `atlas-notes/note`:

1. `type Processor interface { ... }` — the complete exported method set.
2. `type ProcessorImpl struct { l logrus.FieldLogger; ctx context.Context; ... }` — private fields; captures tenant via `tenant.MustFromContext(ctx)` where the package is tenant-aware; captures `producer.Provider`, `*gorm.DB`, and collaborator processors as fields.
3. `func NewProcessor(l logrus.FieldLogger, ctx context.Context, <extra deps>) Processor` — returns the **interface**. Extra deps (e.g. `db *gorm.DB`) follow `l, ctx` positionally, matching existing Gen3 services.
4. A compile-time conformance assertion `var _ Processor = (*ProcessorImpl)(nil)` in the package (in `processor.go` or its test file).
5. Method conventions preserved from the existing code: pure/buffered logic in `Method(mb *message.Buffer)`, side-effecting variants in `MethodAndEmit(...)`, providers as `XxxProvider(...) model.Provider[T]`.

### FR-2: CP-2 signature fixes (interface exists, constructor returns concrete type)

The 18 files below (12 services minus the 2 atlas-messengers files handled under FR-3) change only the `NewProcessor` return type from `*ProcessorImpl` to `Processor`, plus any caller that declared a variable/field of type `*ProcessorImpl`:

- `atlas-doors`: `party/processor.go`, `door/processor.go`, `data/skill/processor.go`, `data/map/processor.go`
- `atlas-summons`: `data/skill/processor.go`, `effectivestats/processor.go`, `inventory/processor.go`
- `atlas-channel`: `data/portal/processor.go`, `data/skill/processor.go`, `portal/processor.go`
- `atlas-saga-orchestrator`: `validation/processor.go`
- `atlas-portal-actions`: `validation/processor.go`
- `atlas-pets`: `pet/processor.go`
- `atlas-npc-conversations`: `validation/processor.go`
- `atlas-mounts`: `mount/processor.go`
- `atlas-map-actions`: `validation/processor.go`
- `atlas-login`: `inventory/processor.go`
- `atlas-inventory`: `data/consumable/processor.go`

If any of these files' interfaces are missing methods that callers use on the concrete type, the interface is extended to the full exported method set (that is a declaration change, not a behavior change).

### FR-3: Gen2 → Gen3 (struct exists, no interface)

For each of the 60 Gen2 files plus the 2 atlas-messengers Gen2.5 files:

1. Rename `type Processor struct` → `type ProcessorImpl` (messengers already uses the name).
2. Extract `type Processor interface` from the exported method set (mechanical: every exported method with a `Processor`/`ProcessorImpl` receiver).
3. Change/introduce `NewProcessor(...) Processor`.
4. Update all call sites within the service (struct fields, function params, local vars typed as the old concrete type → the interface).

Affected services and file counts: atlas-channel (26 + 3 CP-2 overlap handled in FR-2), atlas-consumables (16), atlas-inventory (9), atlas-configurations (3), atlas-storage (2), atlas-chairs (1), atlas-login (1), atlas-monster-death (1), atlas-messengers (2). The authoritative list is the FR-8 inventory scan, not this prose.

### FR-4: Gen1 → Gen3 (no Processor type at all)

For each of the 28 Gen1 files — atlas-data (19), atlas-asset-expiration (5), atlas-portals (2), atlas-reactors (2):

1. Introduce `Processor` interface + `ProcessorImpl` + `NewProcessor(l, ctx)` in the package.
2. Convert package-level curried functions `F(l)(ctx)(args)` into `ProcessorImpl` methods `F(args)`; the `l`/`ctx` currying levels collapse into the struct fields. Package-level provider functions used only within the package may remain package-private helpers.
3. Update every call site: REST handlers, Kafka consumer handlers, tickers, and `main.go` wiring in the owning service construct a processor (`NewProcessor(l, ctx)`) at the top of the handler and call methods on it — matching how existing Gen3 services wire handlers.
4. Where a Gen1 package's functions are pure data lookups delegating to a registry or REST requester (the common shape in atlas-data), the interface still gets the full exported set; no logic moves.

atlas-data is the largest single unit (19 files, wide fan-in from its REST resource layer) and MUST be converted package-by-package with a green build between packages.

Additionally, any interface-less processor files revealed by the FR-8 inventory inside otherwise-Gen3 services (e.g. the 5 non-struct processor files in atlas-monster-death) are converted under whichever of FR-3/FR-4 matches their current shape. The 9 services currently having zero Processor interfaces: atlas-data, atlas-asset-expiration, atlas-portals, atlas-reactors, atlas-consumables, atlas-configurations, atlas-monster-death, atlas-messengers (atlas-renders has no processors and is out of scope).

### FR-5: Mocks for every converted package

Every package converted under FR-2/FR-3/FR-4 gains `<pkg>/mock/processor.go`:

- `package mock`, `type ProcessorMock struct` with one `XxxFunc` field per interface method (same signature).
- Each method delegates to the func field when non-nil and returns zero values otherwise, following the existing convention in `atlas-notes/note/mock/processor.go`.
- `var _ <pkg>.Processor = (*ProcessorMock)(nil)` conformance assertion.
- Existing mocks in the 12 CP-2 services are audited: if a mock already exists it is kept and updated to match any interface extension (per project memory, mocks must be updated when the interface changes).

### FR-6: Test coverage

1. Existing tests are updated in the same commit as the package they exercise (test files reference internal names; renames break them — known gotcha).
2. Compile-time conformance assertions (FR-1.4, FR-5) are mandatory for every converted package and mock.
3. Where a converted package contains non-trivial logic (branching, state transitions, emission batching) with no existing test exercising it, characterization tests are written against the pre-conversion behavior and must pass unchanged post-conversion. Pure delegation shims (one-line registry/requester calls) do not require new tests.
4. Test setup uses the project's Builder pattern; no `*_testhelpers.go` files with test-only constructors.

### FR-7: Batching and sequencing

Work proceeds in three tiers, each tier landing as per-service (or per-package for atlas-data/atlas-channel) commits with green verification between commits:

- **Tier A — CP-2 signature fixes** (FR-2): 11 services, mechanical, lowest risk. Establishes the pattern and the verification cadence.
- **Tier B — Gen2 extraction** (FR-3): 9 services, mechanical interface extraction. atlas-channel is the bulk (29 packages) and is committed in package groups.
- **Tier C — Gen1 modernization** (FR-4): 4 services, highest touch (call-site rewiring). atlas-data last, package-by-package.

### FR-8: Inventory as ground truth

Before Tier A begins, a classification scan (grep-based, checked into the task folder as `inventory.md`) enumerates every non-mock `processor.go` with its classification (Gen1 / Gen2 / Gen2.5 / Gen3-concrete-return / Gen3-conforming). The scan is re-run after each tier; the acceptance criteria in §10 are defined against this scan, not against the counts in this document (which are a point-in-time snapshot of main at 38d4d0ba2).

## 5. API Surface

None. No REST endpoints, request/response shapes, Kafka topics, message schemas, or packet formats change. This is an internal-idiom refactor; the JSON:API resource layer and consumer registrations only change in *how they obtain* a processor, never in what they do with it.

## 6. Data Model

None. No entities, migrations, or schema changes. `*gorm.DB` handles move from curried parameters into `ProcessorImpl` fields where applicable, with identical usage.

## 7. Service Impact

| Service | Tier | Scope |
|---|---|---|
| atlas-doors | A | 4 signature fixes |
| atlas-summons | A | 3 signature fixes |
| atlas-saga-orchestrator, atlas-portal-actions, atlas-pets, atlas-npc-conversations, atlas-mounts, atlas-map-actions | A | 1 signature fix each |
| atlas-channel | A+B | 3 signature fixes + 26 Gen2 extractions (largest Tier B unit) |
| atlas-login | A+B | 1 signature fix + 1 Gen2 extraction |
| atlas-inventory | A+B | 1 signature fix + 9 Gen2 extractions |
| atlas-consumables | B | 16 Gen2 extractions |
| atlas-configurations | B | 3 Gen2 extractions (+ triage of `configurations/data`) |
| atlas-storage | B | 2 Gen2 extractions |
| atlas-chairs | B | 1 Gen2 extraction |
| atlas-monster-death | B | 1 Gen2 extraction + triage of 5 interface-less files |
| atlas-messengers | B | 2 Gen2.5 interface extractions |
| atlas-portals, atlas-reactors | C | 2 Gen1 conversions each |
| atlas-asset-expiration | C | 5 Gen1 conversions |
| atlas-data | C | 19 Gen1 conversions, package-by-package |

Every converted package additionally gains a `mock/` package (FR-5). No service's external contract changes.

## 8. Non-Functional Requirements

- **Behavior preservation**: byte-identical runtime behavior. The refactor moves declarations, not logic. Any pre-existing bug found during conversion is reported, not fixed in this branch.
- **Multi-tenancy**: `tenant.MustFromContext(ctx)` capture semantics are preserved exactly — packages that resolve tenant per-call keep per-call resolution unless they already capture at construction; construction-time capture is not introduced where it would change failure timing (`MustFromContext` panics on missing tenant).
- **Performance**: interface dispatch overhead is negligible for this codebase's call patterns; no hot inner loops call processors per-frame. No benchmarking required.
- **Verification protocol** (per CLAUDE.md, per touched service, per tier): `go test -race ./...`, `go vet ./...`, `go build ./...` clean in every changed module; `docker buildx bake atlas-<svc>` for every service whose `go.mod` module was touched; `tools/redis-key-guard.sh` clean from repo root.
- **Observability**: logger fields and log lines are unchanged.

## 9. Open Questions

1. **PR strategy**: all three tiers on this one branch produces a very large PR (~90 converted files + call sites + ~90 new mock files + tests). Options: (a) one PR with per-service commits; (b) sequential PRs cut from this branch per tier (A → B → C), rebasing between merges. Decide at design time; default is (a) unless review load argues otherwise.
2. **`atlas-configurations/configuration/data`** has a `mock/processor.go` but the service greps as having zero `Processor interface` declarations — triage during Tier B planning to determine what that mock implements.
3. **Interface granularity for atlas-data**: one `Processor` per package (matching Gen3 convention) is the default; if a package's exported surface is a single function, the interface still wraps it (uniformity over minimalism), pending design confirmation.

## 10. Acceptance Criteria

- [ ] `grep -rn "func NewProcessor(" services/ --include="*.go" | grep -v "_test.go\|/mock/" | grep "\*ProcessorImpl"` returns **zero** rows.
- [ ] `grep -rln "type Processor struct" services/ --include="*.go" | grep -v mock` returns **zero** rows.
- [ ] Every non-mock `processor.go` under `services/` belongs to a package that declares `type Processor interface`, verified by the FR-8 inventory scan showing 100% Gen3-conforming (documented exemptions require justification in `inventory.md`; expected exemptions: none).
- [ ] Every converted package has `mock/processor.go` with a `ProcessorMock` and a compile-time conformance assertion; every touched pre-existing mock still compiles against its (possibly extended) interface.
- [ ] Every converted package has `var _ Processor = (*ProcessorImpl)(nil)`.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in every changed module.
- [ ] `docker buildx bake atlas-<svc>` succeeds for every touched service.
- [ ] `tools/redis-key-guard.sh` clean.
- [ ] No diff outside `services/` and `docs/tasks/task-116-processor-gen3-unification/` (no `libs/` changes, no behavior-bearing changes).
- [ ] Characterization tests added for previously-untested non-trivial converted logic (list maintained in `inventory.md`).
- [ ] Code review (`superpowers:requesting-code-review` → `backend-guidelines-reviewer` + `plan-adherence-reviewer`) run before the PR opens.
