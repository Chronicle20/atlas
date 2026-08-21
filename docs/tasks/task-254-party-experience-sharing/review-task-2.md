# Review: Task 2 — atlas-monster-death party members model, REST relationship, builders

Range reviewed: `f4c17c98e..b2e9d21ee` (two commits, `8a0c9d70a` + `b2e9d21ee`, reviewed as one
unit per the controller's instruction — the split is an explained incident recovery, not a
defect).

Files touched:
- `services/atlas-monster-death/atlas.com/monster/party/model.go`
- `services/atlas-monster-death/atlas.com/monster/party/rest.go`
- `services/atlas-monster-death/atlas.com/monster/party/builder.go` (new)
- `services/atlas-monster-death/atlas.com/monster/party/rest_test.go` (new)

`party/requests.go` and `party/processor.go` confirmed byte-identical between
`f4c17c98e` and `b2e9d21ee` (`diff` exit 0 both) — the brief marked them read-only and they
were left untouched.

## Scope confirmation

Matches the brief exactly: no files outside the `party` package were touched, no
service-boundary reach-in, no unrelated edits. `scope_confirmed`: yes.

## Requirement-by-requirement (brief "Interfaces" section)

- `party.MemberModel` with `Id() uint32`, `Name() string`, `Level() byte`, `JobId() job.Id`,
  `Field() field.Model`, `Online() bool` — present, `model.go:26-57`. All value receivers,
  all fields unexported (`model.go:26-33`). PASS.
- `party.Model.LeaderId() uint32` and `party.Model.Members() []MemberModel` — present,
  `model.go:18-24`. PASS.
- `party.NewBuilder(id uint32) *Builder` and `party.NewMemberBuilder(id uint32) *MemberBuilder`
  — present, `builder.go:14-16` and `builder.go:52-56`. PASS.

## REST port fidelity

Diffed `party/rest.go` in this module against `services/atlas-channel/atlas.com/channel/party/rest.go`
at the same commit. Identical line-for-line except the one deliberate divergence the brief
called out: this module's `SetToManyReferenceIDs` seed literal explicitly sets
`Instance: uuid.Nil` (`rest.go:84`), where the channel copy omits the field (relying on the
zero value). Verified by direct diff of both files — no other divergence. PASS.

`ExtractMember` builds the field via `field.NewBuilder(rm.WorldId, rm.ChannelId, rm.MapId).SetInstance(rm.Instance).Build()`
(`rest.go:137`) — matches brief exactly.

`GetReferencedIDs`/`GetReferencedStructs` iterate `r.Members` by `range` over a slice
(`rest.go:50`, `rest.go:62`) — insertion order preserved, no map involved. No non-determinism
introduced by this unit (FR-9.1 concerns members/solo-character ordering in the distributor,
a later task; this package does not sort and does not need to, since it only mirrors REST
response order).

## Builder pattern

`Builder`/`MemberBuilder` in `builder.go` are chainable pointer-receiver builders returning
the plain value model from `Build()` (no error, matching the brief's "no validation to
perform"). Not a `*_testhelpers.go` file — it's a first-class package file, consistent with
repo convention. `rest_test.go` uses it directly (`TestBuilders_ProduceReadableModel`,
`rest_test.go:138-156`). PASS.

## Immutability

`Model` and `MemberModel`: all fields unexported, only value receivers, construction only
through `Extract`/`ExtractMember` or `Builder`/`MemberBuilder`. No exported struct fields on
either domain model. PASS. (`RestModel`/`MemberRestModel` are wire DTOs, exported by
convention — matches the channel service's own file, not a defect.)

## Constants reuse

`job.Id`, `field.Model`, `world.Id`, `channel.Id`, `_map.Id` all imported from
`libs/atlas-constants/*` (`model.go:4-5`, `rest.go:9-13`) — no redeclaration. PASS.

## Tests

All four tests specified in the brief are present and match the brief's assertions exactly:
`TestExtract_MapsMembers`, `TestSetToManyReferenceIDs_SeedsMembers`,
`TestGetReferencedIDs_And_Structs`, `TestBuilders_ProduceReadableModel` (`rest_test.go`).

Ran the suite directly:
```
go build ./... && go test ./...
```
from `services/atlas-monster-death/atlas.com/monster` — all packages pass, including
`atlas-monster-death/party`. Confirmed independently in this review (not just trusting the
implementer's report).

Confirmed the commit split is real and matches the report's narrative: `git show
8a0c9d70a:.../party/model.go` shows the original 9-line stub (no `leaderId`/`members`/
`MemberModel`) still present after the first commit; `b2e9d21ee` adds exactly those 49 lines.
Commit 1 alone (`rest.go`+`builder.go`+`rest_test.go` referencing `Model.Members()`) would not
compile in isolation — expected, since the two commits are explained as one unit and the
combined tree builds and tests clean.

## Line endings

Checked `model.go`, `rest.go`, `builder.go` for CRLF — all LF, consistent with the rest of the
Go module. No normalization concern (nothing to preserve; these are new/rewritten files in a
Go module that is uniformly LF).

## Non-blocking observations

- `MemberBuilder.SetJobId` and `MemberBuilder.SetField` (`builder.go:63-70`) are not exercised
  by any test in this unit — `TestBuilders_ProduceReadableModel` only chains `SetLevel`,
  `SetName`, `SetOnline`. Not a defect (the brief's spec test only calls those three), but
  worth flagging since these two setters are currently untested code paths. Low risk — they
  are trivial field assignments identical in shape to the tested setters.
- No mock/consumer of `party.Model.Members()` exists yet in this module — expected, since
  wiring the distributor to consume it is a later task in the plan.

## Not evaluable

None. The unit's surface (four files in one package) was fully reviewable within scope; no
file the diff depends on was out of reach.

## Verdicts

**Spec-compliance verdict: PASS.** Every interface, file, and test in the brief is present
and matches the specified shape; the one deliberate divergence from the channel port
(`Instance: uuid.Nil`) is correctly implemented.

**Task-quality verdict: PASS with one non-blocking note** (untested `SetJobId`/`SetField`
builder setters — trivial, not blocking).
