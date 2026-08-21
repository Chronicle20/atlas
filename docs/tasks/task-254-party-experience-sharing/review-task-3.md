# Review: Task 3 — `atlas-monster-death` monster information gains `Level`/`Name` and a `Processor`

Range reviewed: `b2e9d21ee..8f24b5dc4` (single commit `8f24b5dc4`)
Brief: `.superpowers/sdd/plan/task-3-brief.md`
Report: `.superpowers/sdd/plan/task-3-report.md`

## Scope confirmed

`git diff --stat b2e9d21ee..8f24b5dc4` shows exactly the 6 files named in the brief's "Files" section, all under `services/atlas-monster-death/atlas.com/monster/monster/information/`:

- `builder.go` (+14)
- `mock/processor.go` (new, +18)
- `model.go` (+10)
- `processor.go` (new, +26)
- `rest.go` (+2)
- `rest_test.go` (new, +41)

No other files touched. `provider.go` untouched as the brief requires ("read-only; the existing free function stays"). Scope matches the brief exactly — no drift, no unrelated edits.

## Requirement-by-requirement

1. **`information.Model.Level() uint32`, `Name() string`** — PASS. `model.go:10-15`: `level uint32`, `name string` fields (unexported), value-receiver accessors `Level()`/`Name()`. No exported struct fields — model immutability preserved.
2. **Builder gains `SetLevel`/`SetName`, zero defaults** — PASS. `builder.go:6-7` adds fields; `builder.go:27-35` adds `SetLevel`/`SetName`, both wired into `Build()` (`builder.go:39-46`). `NewBuilder()` (builder.go:10-15) is unchanged and does not initialize `level`/`name` — they default to zero value / empty string, exactly as the brief mandates ("do not invent a non-zero level or name").
3. **`rest.go` `Extract` maps `Level`→`level`, `Name`→`name`** — PASS. `rest.go` (Extract func): `level: rm.Level, name: rm.Name` added; `RestModel.Level` (json:"level") and `RestModel.Name` (json:"name") already existed at the RestModel struct definition, matching the brief's claim these fields pre-existed.
4. **`information.Processor` interface + `ProcessorImpl` + `NewProcessor`** — PASS. `processor.go` (new file) matches the brief's supplied code verbatim: `Processor{GetById(monsterId uint32) (Model, error)}`, `ProcessorImpl{l, ctx}`, `NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor`, `var _ Processor = (*ProcessorImpl)(nil)`, and `GetById` delegates to the pre-existing free function `GetById(p.l)(p.ctx)(monsterId)` in `provider.go` — provider.go is untouched, confirming "the processor delegates to it" without reach-in or duplication.
5. **`informationmock.ProcessorMock`** — PASS. `mock/processor.go` matches the brief and mirrors the shape of `party/mock/processor.go` exactly: `GetByIdFunc func(monsterId uint32) (information.Model, error)`, `var _ information.Processor = (*ProcessorMock)(nil)`, nil-func fallthrough returning `information.Model{}, nil`.
6. **Pattern conformance to `party/processor.go` / `party/mock/processor.go`** — PASS, verified by direct diff comparison (see below). Structurally identical: same field names (`l`, `ctx`), same constructor shape, same `var _` assertion placement, same mock nil-fallthrough idiom.
7. **Test file `rest_test.go`** — PASS. `TestExtract_CarriesLevelAndName` and `TestBuilder_SetsLevelAndName` match the brief's exact input values and assertions (`Extract(RestModel{Id: 100100, Name: "Blue Snail", Hp: 8, Experience: 3, Level: 2})`; `NewBuilder().SetHp(1000).SetExperience(500).SetLevel(125).SetName("Zakum").Build()`).

## Correctness of the change itself

- No error paths introduced beyond the existing `(Model, error)` signature threading — `Extract` and `Build` both still return `nil` error unconditionally, consistent with the pre-existing code (not a regression, not a defect scoped to this task).
- `ProcessorImpl.GetById` correctly threads `p.l`/`p.ctx` into the existing curried free function; no logic duplicated from `provider.go`.
- Determinism (FR-9.1): no map iteration in any touched file. N/A here — no maps introduced.
- Immutable models: confirmed — no exported struct fields on `Model` or `Builder`; construction is exclusively through `Builder.Build()` or `Extract()`.
- No stray `// TODO`, no stub returns, no placeholder logic anywhere in the diff.

## Test honesty

Reran independently:
```
go build ./... && go test ./monster/information/... -v
=== RUN   TestExtract_CarriesLevelAndName
--- PASS: TestExtract_CarriesLevelAndName (0.00s)
=== RUN   TestBuilder_SetsLevelAndName
--- PASS: TestBuilder_SetsLevelAndName (0.00s)
PASS
```
Full module: `go test ./...` from `services/atlas-monster-death/atlas.com/monster` — all packages `ok` or `[no test files]`, no failures.

Both tests assert on the new `Level()`/`Name()` accessors and would fail to compile against the pre-change `model.go`/`builder.go` (no `Level`/`Name`/`SetLevel`/`SetName` existed) — this is a genuine compile-time RED→GREEN, not a vacuously-passing test. The brief's expected Step 2 failure ("FAIL to compile") is consistent with this; the implementer landed test+impl in one commit rather than two, which is a minor process deviation from the literal step list but does not weaken the evidence — the test cannot pass without the new fields, and it does now.

## Repo conventions

- Module root respected: no edits outside `services/atlas-monster-death/atlas.com/monster`.
- Builder pattern used for test setup (`NewBuilder()...Build()`); no `*_testhelpers.go` file created.
- No constants redeclared (`world.Id`/`channel.Id`/`_map.Id`/`job.Id`/`field.Model` not touched by this task).
- No service-boundary reach-in: `Processor` in `information` package only calls package-local `GetById`; no cross-service import added.
- Directory/module correctly `monster/monster/information` per the brief's `atlas-monster-death` = `monster` (not `monster-death`) note.

## Not evaluable

None. The full surface named in the brief was reviewed and is small enough (111 lines) to read in full; no file exceeded the slice-first threshold.

## Verdicts

- **Spec compliance:** PASS — every interface, file, and behavior named in the brief is present and matches exactly, including the verbatim code blocks supplied in the brief.
- **Task quality:** PASS — pattern-consistent with `party/processor.go`/`party/mock/processor.go`, builds and tests clean, no invented defaults, no scope creep, no stubs, immutability preserved.

No blocking or non-blocking findings.
