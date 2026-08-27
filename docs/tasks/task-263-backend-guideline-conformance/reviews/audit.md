# Backend Audit — task-263-backend-guideline-conformance

- **Service Path:** whole-branch diff, `main` (`eaa5ce6f7`) → `HEAD` (task-263-backend-guideline-conformance)
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-27
- **Build/Tests:** not run — the flagless `tools/verify.sh` gate is running separately per the
  controller's instruction; this audit is scoped strictly to DOM-01, DOM-04, FILE-05 semantics.
- **Overall:** NEEDS-WORK

## Scope and method

Diff surface: `git diff --stat main..HEAD` = 1305 files changed (48372 insertions, 57430
deletions) across 247 commits. Non-test Go files changed resolve to **339 changed packages**
(`git diff --name-only main..HEAD -- '*.go' | grep -v _test.go | xargs -n1 dirname | sort -u`).

For each of the 339 changed packages this audit ran, independently of `sweep.sh`/
`split-by-model.sh`:

1. **DOM-01 / FILE-05 exhaustive re-derivation.** For every changed package, checked
   `model.go`/`builder.go` presence and, separately, grepped **every non-`builder.go`,
   non-test `.go` file in every changed package** for `^type [A-Za-z0-9_]*[Bb]uilder(\[...\])? struct`
   and `^func (New[A-Za-z0-9_]*[Bb]uilder|NewBuilder)(\[...\])?\(` — a superset of `sweep.sh`'s
   literal `Builder struct` regex, deliberately widened to catch lower-cased type names and
   generic type parameters, which `sweep.sh`'s regex cannot match. This turned up exactly 7
   misplaced-builder declarations branch-wide (not 5, as `inventory-file05-builders.txt`
   reports) — see Findings below.
2. **DOM-04 literal-detector re-derivation.** For every changed package with `rest.go`, grepped
   for `^func Transform(`. 13 packages fail the literal detector; all 13 were cross-checked
   against `exemptions.md` by file/package name and confirmed disposed of there (12 as "no
   domain `Model`" / "detector artifact", 1 — `atlas-data/data/skill` — documented under the
   recursive-directory detector-artifact class with an explicit cross-reference to the
   "no domain Model" disposition).
3. **DOM-04 lossy-Transform spot check.** Diffed `main..HEAD` per package to find the 172
   packages where `func Transform(` is new on this branch (absent on `main`). Field-compared
   `Model`/`RestModel` structs for a 15-package random sample (`shuf` seeded off the finding
   file, reproducible) not already named in `exemptions.md`'s "lossy Extract" section. Found the
   established resolution-#3/#4 pattern (Task 16) applied consistently; one additional instance
   of the same accepted pattern not individually cataloged in `exemptions.md` (non-blocking,
   noted below). No new uncataloged lossy case that violates the resolution-#3/#4 policy was
   found in the sample.

## Findings

### Blocking — FILE-05 (and consequently DOM-01), not in `exemptions.md`

**1. `services/atlas-cashshop/atlas.com/cashshop/asset/model.go:106`**
`type Builder[E any] struct {` — and `model.go:116` `func NewBuilder[E any](id uint32,
templateId uint32, referenceId uint32, referenceType ReferenceType) *Builder[E]`. This is the
package's *primary* domain-model builder (constructs `Model[E]`, `model.go:16`), not one of the
seven reference-data builders. `exemptions.md`'s "DOM-01 — sibling builders over distinct types"
section (line 403-409) documents that "the package's own `NewBuilder` stayed in `model.go`" as
a factual note while disposing of the *sibling-duplication* question (DOM-01) — but it does not
dispose of the **file-placement** question (FILE-05): `Builder[E any]` belongs in `builder.go`
per the file-responsibilities table, and it is not there. `builder.go` in this package holds
only the seven `*ReferenceDataBuilder` types; the domain `Builder[E]` was left behind.
`inventory-file05-builders.txt` (regenerated at HEAD per `exemptions.md:516`) misses this because
its underlying regex (`sweep.sh:34`, `'^type [A-Za-z0-9_]*Builder struct'`) requires the literal
suffix `Builder struct` with no generic-parameter bracket between them; `Builder[E any] struct`
does not match.
- **DOM-01:** FAIL — `builder.go` exists in the package but does not contain the domain model's
  `NewBuilder`/`Build()` (verification per `file-responsibilities.md:190`: "File `builder.go`
  exists ... Present, with `NewBuilder()`, fluent setters, and a `Build()`"; the domain model's
  `Build()` is at `model.go:142`, not `builder.go`).
- **FILE-05:** FAIL — `type Builder[E any] struct` (`model.go:106`) and `func NewBuilder[E any](`
  (`model.go:116`) sit in `model.go`, not `builder.go`.

**2. `services/atlas-channel/atlas.com/channel/account/model.go:52`**
`type builder struct {` — and `model.go:69` `func NewBuilder() *builder`. Neither declaration
appears anywhere in `sweep.sh`'s output (`inventory-file05-builders.txt` has zero rows for this
package) because the exported constructor name is `NewBuilder` (correct) but the *type* it
returns is the unexported `builder` (lowercase), which does not match the sweep's literal
`Builder struct` suffix (case-sensitive: lowercase `builder` does not end in the capitalized
string `Builder`). This package has no `builder.go` file at all — confirmed
`ls services/atlas-channel/atlas.com/channel/account/builder.go` fails. Not listed under any
DOM-01/FILE-05 heading in `exemptions.md`. The package is in the branch's changed-package set
because `rest.go` was touched to add `Transform` (`rest.go:80-97`, this branch's own DOM-04 fix)
— the pre-existing, untouched `builder` in `model.go` was never relocated.
- **DOM-01:** FAIL — no `builder.go` exists in the package at all
  (`file-responsibilities.md:190`).
- **FILE-05:** FAIL — `type builder struct` (`model.go:52`) and `func NewBuilder()`
  (`model.go:69`) sit in `model.go`, not `builder.go`.

Both findings are genuine detector blind spots of the exact kind the audit brief named
(case-sensitivity and generics defeating a regex-based sweep) and are not covered by any
`exemptions.md` entry under a FILE-05 or DOM-01 heading for either package.

### Non-blocking — out of task-263's target scope, confirmed pre-existing

**144 changed packages have `model.go` but no builder pattern anywhere in the package** (no
`builder.go`, and — per the exhaustive scan above, which covers every non-test `.go` file, not
just `builder.go` — no `type *Builder struct`/`func New*Builder(` hiding in any other file
either). Examples: `services/atlas-ban/atlas.com/ban/character/model.go` (2-field `Model`,
`id`/`name`, no constructor of any kind), `services/atlas-buddies/atlas.com/buddies/character`,
`services/atlas-channel/atlas.com/channel/session`, and 141 more (full list generated at
`/tmp/dom01_findings.txt` during this audit — not committed, reproducible via the script in
"Scope and method").

This is **literally** a DOM-01 trigger-fires-and-fails condition per the checklist's unconditional
"Applies when: package has `model.go`." I am not marking it N/A. But it is not a task-263
regression or an incomplete task-263 fix:

- None of the 144 appear in `classify-dom01-fr15.tsv`, `inventory-dom01-*.txt`, or
  `classify-file05.tsv` — confirmed by grep across all four files, zero hits for any of the
  sampled package names.
- A 12-package random sample was checked against `main` directly
  (`git show main:<pkg>/builder.go`): all 12 return "does not exist in 'main'" — the state is
  byte-identical to pre-branch.
- The PRD's own stated scope (`prd.md:19-21`) quantifies "repo-wide" DOM-01/FILE-05 closure as
  100 misplaced-`Builder`-struct relocations + 59 `NewModelBuilder`→`NewBuilder` renames (against
  132 pre-existing correct `NewBuilder`s) — i.e., closing every *existing* builder's naming and
  placement, not introducing a builder pattern to packages (like `ban/character`) that never had
  one. This matches `exemptions.md`'s own scope statements throughout.

Recorded here rather than silently passed, per the audit's anti-rationalization instruction, but
not counted as blocking against task-263's acceptance criterion — this is pre-existing, wider
repository debt outside "the three checklist rows this branch exists to close," and would need
its own task (adding builder.go to ~144 packages is a much larger, materially different piece of
work than what this branch's PRD scoped and than what `exemptions.md` catalogs).

**`services/atlas-messages/atlas.com/messages/pet/rest.go:46-55`** — `Transform`/`Extract` map
only `Id`/`Slot`/`Name`; `RestModel` carries 12 additional fields (`CashId`, `TemplateId`,
`Level`, `Closeness`, `Fullness`, `Expiration`, `OwnerId`, `X`, `Y`, `Stance`, `FH`, `Flag`,
`PurchaseBy`) with no `Model` counterpart. This is the same Task-16 "resolution #3" pattern
documented for sibling packages (e.g. `atlas-messages/messages/character`,
`exemptions.md:371-373`) — `Transform` does not invent values for fields `Extract` never reads —
and it has a `TestTransformRoundTrip` (`rest_test.go:8`) consistent with that pattern. It is not
a new violation, but it is not individually cataloged in `exemptions.md`'s "DOM-04 — lossy
Extract" section either (that section states its count as "19 entries, as found in
`handwork-notes.md`" — an enumeration, not a closed set). Flagging as a documentation-completeness
gap, not a code defect.

## Checklist Results — DOM-01 / DOM-04 / FILE-05, branch-wide

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` exists with `NewBuilder()`, setters, `Build()` — every changed package with `model.go` | **FAIL** | `services/atlas-cashshop/atlas.com/cashshop/asset` (no `NewBuilder`/`Build()` for the domain `Model` in `builder.go`); `services/atlas-channel/atlas.com/channel/account` (no `builder.go` at all) |
| DOM-01 | (same rule, remaining ~193 changed packages with `model.go`) | PASS or N/A | PASS where `builder.go` exists with `NewBuilder` at HEAD (spot-verified citations above); N/A per `exemptions.md` "trigger not fired (no model.go)" (5 entries) and "sibling builders over distinct types" (25 packages, all re-derived at HEAD, see exemptions.md:388-478) and "NewBuilder name already taken" (2 entries, re-verified above) |
| DOM-01 | (144 changed packages, `model.go` present, no builder pattern anywhere) | **FAIL, non-blocking** (see above) | pre-existing on `main`; not in any task-263 target inventory |
| DOM-04 | `func Transform(` in `rest.go` — every changed package with `rest.go` | PASS or N/A | 13 literal misses, all disposed in `exemptions.md` (12 "no domain Model"/detector-artifact clusters + `atlas-data/data/skill` cross-referenced to the same class); remaining ~150 changed packages with `rest.go` carry a literal `func Transform(` — verified via grep sweep |
| DOM-04 | Round-trip / lossy-field coverage (spot check of 172 newly-added `Transform`s) | PASS (sampled) | 15-package random sample field-compared against `RestModel`; consistent with Task-16 resolution #3/#4 policy; one uncataloged-but-consistent instance noted non-blocking (`atlas-messages/messages/pet`) |
| FILE-05 | `type <X>Builder struct` in `builder.go` — exhaustive scan of every non-`builder.go`, non-test file in all 339 changed packages | **FAIL** | `services/atlas-cashshop/atlas.com/cashshop/asset/model.go:106,116`; `services/atlas-channel/atlas.com/channel/account/model.go:52,69` |
| FILE-05 | (remaining 5 misplaced-builder declarations found by the same exhaustive scan) | N/A / exempt | 4 `entityBuilder` entries (`exemptions.md:518-521`, re-confirmed present at HEAD) + `libs/atlas-packet/model/skill_usage_info.go:143` (excluded tree, `exemptions.md:530-532`) |

## Not evaluable from the diff

- Full field-by-field lossy-Transform audit of all 172 newly-added `Transform` functions: only a
  15-package random sample was independently re-derived; the remaining 157 rely on the branch's
  own Task 16-18b handwork (`handwork-notes.md`) and per-batch review artifacts
  (`review-task-16-*.md` etc.), which this audit did not re-walk line-by-line. Would need a full
  re-derivation of each package's `Model`/`RestModel` field sets to close this out completely.
- The five DOM-01 "trigger not fired (no model.go)" and two "NewBuilder name already taken"
  entries were not independently re-verified in this pass beyond a spot check of two of the seven
  (`atlas-channel/asset`, `atlas-character/character`) — `exemptions.md` states they were
  re-derived at HEAD with `grep -n`, and the two spot-checked matched exactly.

## Summary

### Blocking (must fix)
- DOM-01 / FILE-05 — `services/atlas-cashshop/atlas.com/cashshop/asset/model.go:106,116`:
  `type Builder[E any] struct` / `func NewBuilder[E any](` must move to `builder.go`; not in
  `exemptions.md`.
- DOM-01 / FILE-05 — `services/atlas-channel/atlas.com/channel/account/model.go:52,69`:
  `type builder struct` / `func NewBuilder()` must move to `builder.go` (which does not exist for
  this package); not in `exemptions.md`.

### Non-Blocking (should fix)
- 144 changed packages with `model.go` and no builder pattern at all — pre-existing, out of
  task-263's stated scope, confirmed identical to `main`; worth a follow-up task, not a blocker
  here.
- `services/atlas-messages/atlas.com/messages/pet/rest.go` — add its `Extract`/`Transform` field
  gap to `exemptions.md`'s "DOM-04 — lossy Extract" catalog for completeness (matches an existing
  disposed pattern, no code change needed).
