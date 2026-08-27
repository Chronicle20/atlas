# Review — Task 22 (task-263)

**Unit:** commit `006158302` — `refactor(atlas-cashshop): move reference data builders into builder.go`
**Brief:** `.superpowers/sdd/plan/task-22-brief.md`
**Report:** `.superpowers/sdd/plan/task-22-report.md`

## Scope

Single-commit hand split of `services/atlas-cashshop/atlas.com/cashshop/asset/reference_data.go`
(1038 → 245 lines) into a new sibling `asset/builder.go` (800 lines, new), moving
the seven `*ReferenceDataBuilder` declaration sets. This is a hand (non-codemod)
transformation, so I independently re-derived verbatim-ness from the two file
snapshots rather than trusting the report's filtered diff.

`git show --stat 006158302`:
```
 .../atlas.com/cashshop/asset/builder.go            | 800 +++++++++++++++++++++
 .../atlas.com/cashshop/asset/reference_data.go     | 793 --------------------
 2 files changed, 800 insertions(+), 793 deletions(-)
```
Exactly the two files the brief names — no extraneous files rode along.

## 1. Verbatim-ness of all seven declaration sets — PASS

Extracted `git show b7f3e8d90:.../reference_data.go` (the pre-move blob) and
sliced each of the seven declaration-set line ranges the report claims
(73–333, 397–663, 684–722, 738–769, 785–816, 842–887, 930–1038), then sliced
the corresponding block in the committed `builder.go` (9–270, 271–538,
539–578, 579–611, 612–644, 645–691, 692–800) and diffed each pair directly
(`diff -u`), independent of the report's filtered-diff method.

All seven diffed byte-identical. The only differences reported by `diff` were
a single trailing blank line in five of the seven new-file slices — an
artifact of my own slice boundary (each new-file range I chose ended one line
past the actual declaration set, at the blank separator before the next
type), not a real content difference. Confirmed by inspecting context around
each reported diff line (e.g. `EquipableReferenceDataBuilder`'s `SetExpiration`
method body is identical text on both sides, `builder.go:259-261` vs
`reference_data.go` old `330-332`). `ConsumableReferenceDataBuilder`,
`SetupReferenceDataBuilder`, `EtcReferenceDataBuilder`, `CashReferenceDataBuilder`
all showed the same single-trailing-blank-line artifact and nothing else.
`PetReferenceDataBuilder` (930–1038, EOF) diffed with zero differences,
including the trailing-blank-line artifact, since it was the last set in the
old file.

No reformatting, renaming, comment edits, reordering, or body changes found in
any of the seven sets.

## 2. Completeness of the move, both directions — PASS

- `grep -n 'Builder' services/atlas-cashshop/atlas.com/cashshop/asset/reference_data.go`
  → no output (exit 1). No builder-related identifier remains in the model file.
- `grep -n '^type' reference_data.go` → exactly the seven models
  (`EquipableReferenceData`, `CashEquipableReferenceData`, `ConsumableReferenceData`,
  `SetupReferenceData`, `EtcReferenceData`, `CashReferenceData`, `PetReferenceData`),
  `reference_data.go:9,73,135,154,168,182,206`.
- `grep -n '^type .*ReferenceDataBuilder struct' builder.go` → exactly the
  seven builders, `builder.go:9,271,539,579,612,645,692`. All seven
  `New<X>Builder` constructors and every `*<X>Builder`-receiver method
  (including the two `Clone` methods on `EquipableReferenceDataBuilder` and
  `CashEquipableReferenceDataBuilder`) are present, matching the enumeration
  from the pre-move file 1:1.
- Nothing non-builder was dragged along: `builder.go` contains no `type
  *ReferenceData struct` (model) declarations; `reference_data.go` retains all
  model accessor methods (spot-checked `IsKarmaUsed`/`IsLocked`/`HasSpikes`
  comment block on `EquipableReferenceData`, untouched).

## 3. Recomputed import blocks — PASS

- `reference_data.go:3-7` imports `time` and `af "github.com/Chronicle20/atlas/libs/atlas-constants/asset"`.
  Usage counts: `time\.` → 5 occurrences, `af\.` → 10 occurrences in the file.
  Both genuinely required; import block unchanged from original but correctly
  still needed by the surviving model code.
- `builder.go:3-7` imports the same two packages. Usage counts: `time\.` → 6,
  `af\.` → 28 occurrences (the `AddFlag`/`RemoveFlag`/flag-setter methods on
  `EquipableReferenceDataBuilder` and `CashEquipableReferenceDataBuilder`
  reference `af.Flag`/`af.HasFlag` heavily). The `af` import the report calls
  out as new content (not moved) is genuinely required by the moved builder
  code — confirmed by both the usage grep and a clean `go build`.
- `go build ./...` and `go vet ./asset/...` from
  `services/atlas-cashshop/atlas.com/cashshop` both pass with no
  unused-import or undefined-symbol diagnostics.

## 4. Constructors not renamed — PASS

All seven `New<X>Builder` names in `builder.go` are byte-identical to the
pre-move names (`NewEquipableReferenceDataBuilder`,
`NewCashEquipableReferenceDataBuilder`, `NewConsumableReferenceDataBuilder`,
`NewSetupReferenceDataBuilder`, `NewEtcReferenceDataBuilder`,
`NewCashReferenceDataBuilder`, `NewPetReferenceDataBuilder`) — confirmed as
part of the verbatim declaration-set diff in §1, which necessarily covers the
constructor signature lines.

## 5. Commit hygiene — PASS

- Only `asset/builder.go` (new) and `asset/reference_data.go` (modified) in
  the commit; no other file touched.
- `karma_roundtrip_test.go` (explicitly called out in the brief as read-only)
  has zero diff across the commit (`git diff 006158302~1 006158302 --
  .../karma_roundtrip_test.go` produced no output).
- `go test ./asset/...` → `ok atlas-cashshop/asset 0.002s`, including the
  `TestKarmaUsedRoundTrip` / `TestKarmaUsedLeavesSpikesAlone` subtests the
  report calls out.
- No dead-field/unchecked-error fixes folded into this commit — none needed
  fixing to make this a pure relocation; the moved code is unchanged.

## Lint

`golangci-lint` is not available in this review environment either
(`which golangci-lint` → not found). Per the task instruction, I did not
substitute a bare `gofumpt` run (a known false positive in this repo since
`tools/lint.sh` is the sole formatting authority). This is left to the
repo-wide verification gate.

## Not evaluable

- `tools/lint.sh` itself was not run (tool unavailable in this environment) —
  deferred to the repo-wide gate per the task's own instruction.

## Verdict rationale

All five review priorities pass with direct, independently-derived evidence
(not the report's own filtered diff). Build, vet, and the pre-existing test
suite (including the specifically-named `karma_roundtrip_test.go`) all pass.
No scope mismatch: the commit contains exactly the relocation the brief
describes, nothing more, nothing less.
