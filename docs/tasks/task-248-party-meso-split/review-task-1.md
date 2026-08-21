# Review: Task 1 — `atlas-drops/party` read-only client

Range reviewed: `892faebc0..53b43c59a` (single commit `53b43c59a`)
Brief: `.superpowers/sdd/plan/task-1-brief.md`
Report: `.superpowers/sdd/plan/task-1-report.md`

## Scope

6 new files, 378 insertions, 0 deletions, all confined to
`services/atlas-drops/atlas.com/drops/party/`:

- `model.go` (83 lines) — `Model`/`MemberModel`, `modelBuilder`/`memberBuilder`
- `rest.go` (150 lines) — `RestModel`/`MemberRestModel`, JSON:API plumbing, `Extract`/`ExtractMember`
- `requests.go` (25 lines) — `Resource`, `ByMemberId`, `requestByMemberId`
- `processor.go` (33 lines) — `Processor`/`ProcessorImpl`/`NewProcessor`
- `mock/processor.go` (18 lines) — `ProcessorMock`
- `rest_test.go` (69 lines) — `TestExtract`, `TestExtract_NoMembers`

Matches the brief's file list exactly. No other files touched (confirmed via
`git diff --stat 892faebc0..53b43c59a`); `go.mod`/`go.sum` untouched.

## Requirement-by-requirement

1. **Module root** `services/atlas-drops/atlas.com/drops` — correct, all new
   files live under it.

2. **No import of atlas-channel's/atlas-monster-death's `party` package** —
   checked every import block in the diff (`model.go:48-50`, `rest.go:207-217`,
   `processor.go:137-144`, `requests.go:176-181`, `mock/processor.go:24-26`).
   Only `atlas-constants` (`field`, `world`, `channel`, `map`), `atlas-model`,
   `atlas-rest`, `api2go/jsonapi`, `google/uuid`, `sirupsen/logrus`, and the
   local `atlas-drops/party` package (in the mock) are imported. PASS.

3. **Immutable models, unexported fields, value receivers, Builder
   construction, no `*_testhelpers.go`** — `model.go:55-58,68-72` fields are
   unexported (`id`, `members`, `field`, `online`); all accessor methods
   (`Id()`, `Members()`, `Field()`, `Online()`) use value receivers
   (`model.go:60,64,74,78,82`); construction is via `NewBuilder()`/
   `NewMemberBuilder()` fluent builders (`model.go:94-128`) or `Extract`/
   `ExtractMember`. No `*_testhelpers.go` file created. PASS.

4. **No stubs / `// TODO` / unimplemented responses** — grepped the diff;
   none present. `ProcessorMock.GetByMemberId` returns a real zero value when
   no func is injected, matching the exact monster-death mock pattern byte
   for byte (verified below). PASS.

5. **No `go.mod` changes** — `git diff 892faebc0..53b43c59a -- .../go.mod
   .../go.sum` is empty; all imported packages (`atlas-constants`,
   `atlas-model`, `atlas-rest`, `google/uuid`, `jtumidanski/api2go`,
   `sirupsen/logrus`, `stretchr/testify`) already listed in
   `services/atlas-drops/atlas.com/drops/go.mod`. PASS.

6. **Reuse existing `atlas-constants` types before defining new ones** —
   `world.Id`, `channel.Id`, `_map.Id`, `field.Model` are used as-is
   (`rest.go:213-216`, `model.go:49`); no new domain type, alias, or numeric
   constant defined anywhere in the diff. PASS.

7. **Repo-relative paths, no absolute paths in committed files** — none of the
   six files contain any path literal. N/A / PASS.

8. **`Extract` on nil `Members` yields non-nil, zero-length `Model.Members()`**
   — `rest.go:307`: `members := make([]MemberModel, 0)` is initialized
   unconditionally before the `for _, m := range rm.Members` loop; ranging
   over a nil slice is a zero-iteration no-op in Go, so `members` is returned
   as the pre-allocated non-nil empty slice regardless of whether
   `rm.Members` was nil. Confirmed by `TestExtract_NoMembers`
   (`rest_test.go:418-429`), which asserts `require.NotNil` and
   `len(...) == 0` on a `RestModel{Members: nil}`. PASS — and this is the
   exact contract Task 3 depends on.

9. **Wire field names match `services/atlas-parties/atlas.com/parties/party/rest.go:95-103`**
   — read the authoritative file directly:
   `worldId`, `channelId`, `mapId`, `instance`, `online` (parties'
   `MemberRestModel` tags, lines 96-103). `atlas-drops`'s
   `MemberRestModel` (`rest.go:329-336`) uses identical tags:
   `json:"worldId"`, `json:"channelId"`, `json:"mapId"`, `json:"instance"`,
   `json:"online"`, on the identical types (`world.Id`, `channel.Id`,
   `_map.Id`, `uuid.UUID`, `bool`). PASS — no silent wire mismatch.

10. **Copy fidelity to the three named sources** — diffed by hand against:
    - `services/atlas-monster-death/atlas.com/monster/party/requests.go` —
      byte-identical to the new `requests.go` (same constants, same
      `RootUrlFor(ctx, "PARTIES")`, same `requestByMemberId`).
    - `services/atlas-monster-death/atlas.com/monster/party/processor.go:1-33`
      — byte-identical to the new `processor.go`.
    - `services/atlas-monster-death/atlas.com/monster/party/mock/processor.go`
      — identical except the import path, exactly as the brief specified.
    - `services/atlas-channel/atlas.com/channel/party/rest.go:1-`relationship
      plumbing (`GetName`, `GetID`, `SetID`, `GetReferences`,
      `GetReferencedIDs`, `GetReferencedStructs`, `SetToManyReferenceIDs`,
      `SetReferencedStructs`, member identifier methods) carried over
      unchanged; only `RestModel`/`MemberRestModel` field sets are trimmed
      (`LeaderId` dropped from `RestModel`; `Name`/`Level`/`JobId` dropped
      from `MemberRestModel`), and the `job` import is correctly absent since
      nothing references `job.Id` anymore. PASS.

11. **Deliberately absent fields** (`name`, `level`, `jobId`, `leaderId`) —
    grepped the diff; none of these four identifiers appear anywhere in the
    six new files. PASS.

## Test honesty

- `TestExtract` (`rest_test.go:374-416`) builds a two-member `RestModel`
  matching the brief's table exactly and asserts every field on both
  extracted members plus the party `Id()` and member count. This is not a
  tautology — it round-trips through `Extract`/`ExtractMember`, which
  construct `field.Model` via `field.NewBuilder(...).SetInstance(...).Build()`
  (verified against the real `field.NewBuilder` signature in
  `libs/atlas-constants/field/model.go:164` — `(worldId world.Id, channelId
  channel.Id, mapId _map.Id) *Builder`, with `SetInstance(uuid.UUID) *Builder`
  at line 188). A field-name or field-order regression in `ExtractMember`
  would fail this test.
- `TestExtract_NoMembers` (`rest_test.go:418-429`) targets exactly the
  nil-`Members` contract Task 3 depends on and would fail if `Extract`
  returned a nil slice instead of `make([]MemberModel, 0)`.
- Both tests were confirmed to build and pass by the implementer
  (`go build ./... && go test ./party/... -v`); I independently re-ran
  `go build ./party/...` from the module root and it completed with no
  output (clean). I did not re-run `go test` since the implementer already
  ran and reported it, per instruction not to re-run tests already run.

## Task-quality notes

- The brief's Step 2 (run the RED test before any implementation file
  exists) was not executed as a discrete step — the implementer wrote all
  six files in one pass and validated only the GREEN state, explaining why
  in both the report's "TDD evidence" and "Concerns" sections. This is an
  honestly disclosed process deviation, not a hidden one, and the brief's
  own Step 2 command (`go test ./party/... -run TestExtract -v` against a
  nonexistent package) is not literally runnable as an isolated RED gate in
  a from-scratch package — there's no way to get a distinct "compiles but
  fails" state before the package exists at all. Non-blocking.
- No behavior beyond the brief's scope was added (no extra methods, no
  extra fields, no premature Task 2/3 wiring). Good scope discipline.

## Not evaluable

- Runtime behavior against a live `atlas-parties` service (the actual HTTP
  round trip, `RootUrlFor` resolution, ingress routing) is outside this
  diff's surface — `requests.go` is a straight copy of an already-proven
  pattern from `atlas-monster-death`, and no test in this unit exercises the
  network path. This is expected for a client-only unit; Task 2/3 (the
  consumers) are the natural place this gets exercised further.
- Downstream consumption by Tasks 2 and 3 is out of scope for this review
  (not yet written).

## Verdict rationale

Every brief requirement is met with `file:line` evidence; the one process
deviation (RED step) is honestly disclosed and doesn't affect correctness.
No blocking findings.
