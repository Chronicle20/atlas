# Review: Task 9 — W2, FR-15 triage and the sole-builder renames

**Commit range:** `49106bbcd..f07717eae`
**Commits reviewed:** `2b07d2776`, `13de0adaf`, `d8240e381`, `f07717eae`
**Brief:** `.superpowers/sdd/plan/task-9-brief.md`
**Report:** `.superpowers/sdd/plan/task-9-report.md`

## Scope note

`git diff --stat 49106bbcd..f07717eae` shows 30 files, but two of those paths
(`agent-ledger.tsv`, `progress.md`, `review-task-*.md`, `task-*-report.md`)
come from `6d1d1b432` and `1f2dbeee9`, which sit inside the given range but
predate Task 9's own commits and were already reviewed/landed as part of
Tasks 5-8's wrap-up. `git log --oneline 49106bbcd..f07717eae` confirms the
four commits named in the dispatch are exactly `2b07d2776`, `13de0adaf`,
`d8240e381`, `f07717eae`; this review evaluates those four only. No scope
mismatch.

## Findings

### 1. `-targets` flag (R2) — PASS, with a disclosed design deviation

- `codemod/rename.go:105-108` (`2b07d2776`): `renameImpl` now does
  `pairs := renamePairs[:]; if len(t.Pairs) > 0 { pairs = t.Pairs }` before
  the scope-lookup loop. This is the correct fix for the exact bug the
  controller's "CONTROLLER CORRECTION to R2" identified (`Target.From`/`To`
  were write-only; `renameImpl` always iterated the fixed `renamePairs`).
  Confirmed by direct reading of the pre-image (`git show 49106bbcd:...`):
  `Target.From`/`To` were indeed dead fields before this change.
- `renamePairs` still exists and is still the fallback for the empty-`Pairs`
  case (`rename.go:105-108`) — pre-existing `rename_test.go` cases (lines 87,
  128, 163, 202, 244, 275, 309) all construct `Target{..., From: "...", To:
  "..."}` with no `Pairs`, and per the report's build evidence and my own
  re-run (`GOWORK=off go test ./...` in `codemod/`, `ok`), they still pass —
  exercising the fallback path Task 9a depends on.
- `-inventory` code path (`targetsFromInventory`) is untouched: the diff hunk
  at `rename.go:365` only *appends* `targetsFromTSV` after it; the function
  body itself has no changed lines.
- Ledger emits exactly one row per input `pkgDir`: `targetsFromTSV`
  (`rename.go` new function) merges same-`pkgDir` lines into one `Target`
  before returning, and `ledger-rename-fr15.tsv` contains exactly 3 rows for
  3 `pkgDir`s, matching `ledger.Verify()`'s one-row-per-target contract.
- The R2 collision guard (`rename.go:93-103`, the hard-coded
  `modelBuilder`/`builder` scope check) is byte-for-byte unmodified — verified
  by reading the current file directly, not just the diff.

**Deviation (non-blocking):** the "CONTROLLER CORRECTION to R2" section of
the brief explicitly overrides the original triples design and mandates a
**two-column** `pkgDir<TAB>stem` targets file with four pairs
(`New<Stem>Builder`, `<Stem>Builder`, lowercase-`stem` variant,
`Clone<Stem>Builder`) derived at runtime, plus a test for "an empty `Pairs`
still falls back to `renamePairs`" and "the lowercase-stem variant". The
implementer instead built a **three-column** `pkgDir\tfrom\tto` explicit-pair
format (`targetsFromTSV`, `rename.go`), reverting to the *original*,
pre-correction R2 text. The `Pairs [][2]string` field and its use inside
`renameImpl` match the correction's core technical fix; only the *format* of
the `-targets` file and its derivation logic differ from what the override
explicitly specified. `rename_test.go`'s `TestTargetsFromTSV` covers
well-formed/merged, malformed-row, and comment/blank-skip cases, but not a
lowercase-stem case (N/A under the chosen format) or an explicit
empty-`Pairs`-falls-back case (covered only implicitly, by the untouched
pre-existing tests). This is functionally inconsequential for Task 9's three
actual targets — all three packages already declare `type Builder` (not
`<X>Builder`), and none declares a `Clone<X>Builder` (confirmed by reading
`builder.go` in each of the three packages pre-change) — so the simpler
explicit-pair format produces an identical outcome here. The implementer's
report acknowledges correcting the brief's factual claim about `Target.From`/
`To` being read, but does not flag that it also diverged from the corrected
TSV *format* the override specified. Worth a note for continuity: any future
`-targets` caller (Task 9a or later) inherits the explicit-triples format,
not the stem-derivation format the design record says was decided.

### 2. `atlas-drop-information` renames — PASS, complete

- `13de0adaf` renames `NewContinentDropBuilder`/`NewMonsterDropBuilder`/
  `NewReactorDropBuilder` → `NewBuilder` across `builder.go`, `provider.go`,
  `subdomain.go`, and both same-package and cross-package test files
  (`continent/resource_test.go:53`, `monster/drop/provider_test.go`,
  `monster/drop/resource_test.go`).
- Repo-wide grep at `f07717eae` for the three old constructor names returns
  no hits — no leftover use sites.
- No `Clone<X>Builder` existed in any of the three packages before the
  change (confirmed by reading each pre-image `builder.go`); the fr15-targets
  file correctly carries one triple per package, not three.
- The exported type in all three packages was already `type Builder struct`
  before this task (not `<X>Builder`), so no type rename was needed or
  performed — consistent with the brief's premise that only the constructor
  needed renaming.
- Build/vet/test independently re-run for `services/atlas-drop-information/atlas.com/dis`:
  all packages `ok`/`[no test files]`, no failures.

### 3. FR-17 — PASS

`git diff 49106bbcd..f07717eae -- services/atlas-drop-information services/atlas-pets`
filtered for string-literal lines outside identifier renames returns nothing.
No JSON tag, REST route, or Kafka topic literal changed in the range.

### 4. `atlas-pets` test-name commit (`d8240e381`, R3) — PASS

Full diff of `d8240e381` touches only `builder_test.go` `func Test...`
declarations (`TestModelBuilder_*` → `TestBuilder_*`,
`TestModelBuilderSetName*` → `TestBuilderSetName*`) at the exact lines R3
named. No production identifier, no call-site change, no import change.
Independently re-ran `go build ./... && go vet ./... && go test ./pet/...`
in `services/atlas-pets/atlas.com/pets` — `ok`.

### 5. Scope fence (R4) — PASS

`git diff --name-only 49106bbcd..f07717eae` touches no file outside the
`codemod/`, docs-artifact, `atlas-drop-information` (3 packages), and
`atlas-pets/pet/builder_test.go` set named in the brief. Directly grepped
`services/atlas-pets` for `ModelBuilder` and confirmed `type ModelBuilder`
declarations in `asset/builder.go`, `compartment/model.go`, and
`inventory/model.go` — the exact Task 9a scope — are present, unmodified,
and untouched by this range.

## Not evaluable

None. All five weighted areas were within the diff's own surface (the
`-targets` flag's caller-visible contract, the three renamed packages, and
their direct call sites), and all were checked against the actual pre/post
file content, not inferred from the report.

## Verdict rationale

No blocking defects found: every requirement in the brief and its two
controller corrections (R1, R3, R4) is met and independently verified against
file content and rebuilt/retested code. R2's core technical requirement
(making `-targets` actually functional via `Target.Pairs`, preserving the
`renamePairs` fallback, keeping the collision guard and ledger contract
intact) is also met. The one deviation — the `-targets` file format differs
from the explicit "overrides R2 above" text — has zero observable effect on
this task's actual output and is disclosed here as a non-blocking
documentation/continuity note for whoever next extends `-targets`.
