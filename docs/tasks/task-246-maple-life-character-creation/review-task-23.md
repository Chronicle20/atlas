# Review: Task 23 — `POST /factory/characters/maple-life`

**Commit:** `646946970` (range `8eda3ee6b..646946970`, single commit — `git log --oneline` confirms no other commits in range)
**Brief:** `.superpowers/sdd/plan/task-23-brief.md`
**Report:** `.superpowers/sdd/plan/task-23-report.md`

## Scope

`git diff --stat 8eda3ee6b..646946970`:

```
services/atlas-character-factory/atlas.com/character-factory/factory/maple_life_rest.go   | 10 +++
services/atlas-character-factory/atlas.com/character-factory/factory/resource.go          | 50 ++++++++++++++
services/atlas-character-factory/atlas.com/character-factory/factory/resource_test.go     | 77 ++++++++++++++++++++++
3 files changed, 137 insertions(+)
```

Matches the brief's Files list plus one undeclared new file (`maple_life_rest.go`), which the report flags and this review evaluates below. `maple_life.go` (Task 22) is untouched — confirmed by the diff stat.

## Requirement-by-requirement

1. **Route registered** — `resource.go:33`: `fr.HandleFunc("/maple-life", rest.RegisterInputHandler[MapleLifeCreateRestModel](l)(si)(CreateMapleLife, handleCreateMapleLife)).Methods(http.MethodPost)`, under the existing `/factory/characters` subrouter beside `/from-preset`. Matches the brief's Step 2 exactly. PASS.
2. **Operation constant** — `CreateMapleLife = "create_maple_life"` added beside `CreateCharacter`/`CreateFromPreset` (`resource.go:19`). PASS.
3. **Status categoriser** (`categorizeMapleLifeError`, `resource.go:57-78`) — checked every row of the brief's table against the diff:
   - `ErrClassOrdinalUnknown` → 400 ✓
   - `ErrLookInvalid` → 400 ✓
   - `ErrSPInvalid` → 400 ✓
   - `*NameInvalidError` (via `errors.As`) → 400 ✓
   - `ErrPresetValidation` → 400 ✓
   - `ErrNameDuplicate` → 409 ✓ (load-bearing per the brief for Task 25's `NAME_TAKEN_AT_SUBMIT` arm)
   - `ErrAtlasDataUnreachable` → 502 ✓
   - `ErrMapleLifeNotConfigured` → 500 ✓
   - `errors.New("boom")` (default) → 500 ✓
   All 9 covered by `TestCategorizeMapleLifeError` (`resource_test.go:112-132`), one subtest per row. Ran the targeted test — PASS, matches report's captured output.
4. **Handler mirrors `handleCreateFromPreset`** — `handleCreateMapleLife` (`resource.go:87-101`) has the same shape: JSON:API input in, `202 Accepted` + `CreateCharacterResponse{TransactionId: ...}` out, `categorizeMapleLifeError` on failure. The one structural difference — the `mapleLifeProcessor` indirection instead of an inline `NewProcessor(d.Logger())` call — is judged separately below (Deviation 2). `handleCreateFromPreset` itself is confirmed byte-identical apart from surrounding additions (diff shows no changes inside its body).
5. **`TestMapleLifeRouteReturnsAcceptedWithTransactionId`** — POSTs a JSON:API body of type `"maple-life-create"` through `postMapleLife` → `server.ParseInput[MapleLifeCreateRestModel]` → `handleCreateMapleLife`, with `mapleLifeProcessor` swapped to a fake returning `"tx-1"`. Asserts `202` and body containing `"transactionId":"tx-1"`.

## Test honesty — mutation-tested, not eyeballed

Per the task instruction, I mutated the handler and reran the test, then reverted (confirmed `git status --short factory/resource.go` clean afterward):

- Changed `http.StatusAccepted` → `http.StatusCreated` inside `handleCreateMapleLife` only: test failed — `Expected status 202, got 201`.
- Changed the response body's `TransactionId` to `transactionId + "-mutated"`: test failed — `Expected body to carry transactionId "tx-1", got: ...tx-1-mutated...`.

Both assertions are load-bearing; the test is not hollow. Verified `go build ./...` and `go test ./...` (module-local) both exit 0 after the revert.

## Deviation 1 — `factory/maple_life_rest.go` (undeclared new file)

Adds `GetName`/`GetID`/`SetID` on `MapleLifeCreateRestModel` (defined in Task 22's `maple_life.go`) so `rest.RegisterInputHandler`'s `api2go/jsonapi.UnmarshalPayload` can decode the request at all.

Checked this is a real, not invented, requirement: `UnmarshalPayload` (`api2go@v1.0.4/jsonapi/unmarshal.go:208`) calls `checkType(data.Type, castedTarget)`, and `checkType` (`unmarshal.go:286-292`) compares the incoming `"type"` string against `getStructType(target)`. `getStructType` (`marshal.go:448-458`) type-asserts the target to `EntityNamer` and calls `.GetName()` when present — so without these three methods, `MapleLifeCreateRestModel` isn't even an `UnmarshalIdentifier` and the input handler cannot compile/register generically the way `RegisterInputHandler[T]` is used at all three call sites in this file. This is a genuine, correctly-diagnosed prerequisite gap, not an invented one.

**Type-string check against the sibling model** — `preset_rest.go`:
```go
func (r PresetCreateRestModel) GetName() string     { return "preset-create" }
func (r PresetCreateRestModel) GetID() string       { return "" }
func (r *PresetCreateRestModel) SetID(string) error { return nil }
```
vs. `maple_life_rest.go`:
```go
func (r MapleLifeCreateRestModel) GetName() string     { return "maple-life-create" }
func (r MapleLifeCreateRestModel) GetID() string       { return "" }
func (r *MapleLifeCreateRestModel) SetID(string) error { return nil }
```
Method set, receiver types (value for `GetName`/`GetID`, pointer for `SetID`), and the no-op `GetID`/`SetID` bodies match `preset_rest.go` exactly — this is a faithful mirror of the established sibling convention, not an invented one.

**Type-string check against the channel side (Task 24)** — Task 24 has since landed in this worktree (uncommitted ahead of this review, out of my scope to re-review, but directly answers the contract-drift question the task asked me to check). `docs/tasks/task-246-maple-life-character-creation/review-task-24.md:47-48` confirms: channel's `rest.go` `GetName()` also returns `"maple-life-create"`, matching the factory side exactly. **No type-string drift.** Had these two disagreed, every request would fail `checkType` at runtime with the unit tests on both sides still green (they mock past the JSON:API wire boundary) — the task's framing of the risk was correct, and the risk did not materialize.

**Verdict on Deviation 1: acceptable.** Correctly diagnosed as a producible prerequisite gap per CLAUDE.md, executed as a faithful mirror of the existing convention, and confirmed (via Task 24's independent review) not to have introduced contract drift.

## Deviation 2 — `var mapleLifeProcessor = NewProcessor` in `resource.go`

**Checked for an established injection pattern in this package first.** All three handlers in `resource.go` — `handleCreateCharacter` (`:154`), `handleCreateFromPreset` (`:101`, now `:99`), and the new `handleCreateMapleLife` — are the entire handler surface of this file. `handleCreateCharacter` and `handleCreateFromPreset` both construct `NewProcessor(d.Logger())` inline with no seam. Grepping the rest of `atlas-character-factory`'s factory-style handlers (only this file has `func handle*`) confirms there is no existing DI pattern this deviates from — it is the first of its kind in the package, not a departure from a convention that should have been reused.

The package does have `NewProcessorWithClients` (`processor.go:71-72`, doc-commented "the test seam — allows injection of mocks") — genuinely unreachable from inside a handler that only ever calls the zero-arg `NewProcessor`. The implementer's diagnosis that this seam "isn't reachable from inside the handler" is accurate.

**Judgment: this is a real smell, and non-blocking rather than blocking, for these reasons:**
- It is package-level *mutable* state used purely as a test hook — a smell CLAUDE.md's builder-pattern guidance is adjacent to (though not the exact case it names, which is about `*_testhelpers.go` constructor files, not swappable production vars).
- In production, the var is set once at package init to `NewProcessor` and never reassigned outside `resource_test.go`; there is no code path in production that mutates it, so there is no production behavior risk today.
- It does make `handleCreateMapleLife` structurally inconsistent with its two siblings (one handler is now swappable, two are not) — a real, if minor, architectural drift from the brief's "mirrors `handleCreateFromPreset` exactly" instruction, which this deviates from precisely for testability the sibling doesn't have.
- A `t.Parallel()` future addition to this test file, or a `-race` run against a hypothetical concurrent caller of `handleCreateMapleLife` while a test is mid-override, would be a genuine hazard — currently moot since no test in the package parallelizes and this is a build-time-scoped var, not a runtime one, but worth a reviewer/maintainer follow-up rather than silent acceptance. A closure-based injection (constructing the processor once in `InitResource` and closing over it, or accepting a constructor func as a handler-factory parameter) would have achieved the same testability without a mutable global, and is the kind of thing that should have been raised as a design question rather than decided unilaterally — the report does flag it explicitly, which is the right call, but the pattern itself is not clean.

This is flagged as a **non-blocking finding** for the reasons above: it does not affect current correctness, it is well-documented, and it is easily correctable in a follow-up without touching the wire contract Task 24/25 depend on.

## Not evaluable

- Whether Task 24's channel-side model and this commit's server-side model will remain in sync going forward is outside this commit's diff; I confirmed the current snapshot matches via Task 24's own review artifact, but that is a fact about a different unit of work, not something this commit's diff itself proves.

## Summary

- Route, constant, and status categoriser all match the brief precisely, verified against the table row-by-row.
- The success-path test is not hollow — mutation-tested on both the status code and the body under review.
- Deviation 1 (`maple_life_rest.go`) is judged **acceptable**: a correctly-diagnosed, correctly-executed prerequisite gap, mirroring the sibling convention, with the cross-service type-string risk checked and found not to have materialized.
- Deviation 2 (`mapleLifeProcessor` package var) is judged a genuine but **non-blocking** smell: package-level mutable test-only state with no current production impact, well-disclosed, but architecturally inconsistent with its two sibling handlers and worth a follow-up cleanup.

## Verdict

APPROVED_WITH_FINDINGS
