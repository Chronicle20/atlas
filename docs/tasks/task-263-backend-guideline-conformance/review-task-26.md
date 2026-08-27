# Review: Task 26 — FR-17/FR-18 behavior-preservation verification

Commit under review: `29b52636b` (docs-only, appends `## Task 26 — FR-17/FR-18 verification` to
`progress.md`).

Brief: `.superpowers/sdd/plan/task-26-brief.md`
Implementer report: `.superpowers/sdd/plan/task-26-report.md`

This review treats the implementer's "zero defects, every deletion paired" claim as a hypothesis
to falsify, not a document to proofread. Each of the six audit commands was re-run independently
against `main...HEAD` (base `eaa5ce6f7`, head `29b52636b`), and the pairing logic behind Steps 5
and 6 (the two highest-risk steps) was reconstructed from scratch with an independent script
rather than trusted from the prose.

## Step-by-step reproduction

### Step 1 — FR-18 excluded trees

```
git diff --name-only main...HEAD | grep -E '\.sql$|docker-compose.*\.ya?ml$|\.tpl$|^libs/atlas-packet/'
```
Reproduced: no output, exit 1. Matches pasted evidence. **PASS.**

### Step 2 — FR-17 JSON tags

Full-context `git diff -G'json:"' main...HEAD` is 7716 lines (implementer reported 7601 — a
~1.5% variance, not investigated further since it's the noisy form that both the brief and the
implementer explicitly discard in favor of the reduced form).

Reduced form (`-U0`, `grep -v '^[+-][+-][+-]'`) reproduces at **13** lines, not the reported 11.
The 2-line delta is real and traced: `design.md`/`plan.md` (prose about the `ModelBuilder` FR-14
rename's JSON-tag safety, e.g. `` `json:"modelBuilder"` `` inside a markdown table) plus the same
text quoted a second time in `rename_test.go`'s `mustContain(t, got, \`json:"modelBuilder"\`)`
assertion (a codemod unit test verifying the tag survives, not a tag edit). All 13 lines are
additions in docs or test-assertion strings; zero are `.go`-source JSON-tag deletions or edits
(confirmed separately: `git diff -U0 main...HEAD -- '*.go' | grep -E '^[+-].*json:"'` → empty).
**PASS**, with a non-blocking note that the pasted "11" undercounts by 2 — an immaterial miscount,
not a hidden defect.

### Step 3 — `GetName() string`

```
git diff -U0 main...HEAD | grep -E '^[+-].*GetName\(\) string'
```
Re-running this literally today returns one hit — but it is the implementer's own prose line
("Output: (none). **PASS** — no `GetName() string` declaration touched...") inside
`progress.md`, which is itself part of the diff now that `29b52636b` (HEAD) includes that
sentence. This is expected self-reference, not a reproduction failure: scoping to `*.go` files
removes it entirely (`git diff -U0 main...HEAD -- '*.go' | grep ...` → empty). **PASS**, confirmed.

### Step 4 — route/Kafka topic literals

```
git diff -U0 main...HEAD -- '*/resource.go' '*/producer.go' '*/consumer.go' | grep '^[+-]' | grep -v '^[+-][+-]'
```
Reproduces at exactly **188** lines, matching the pasted evidence. Spot-checked ~20 of the
constructor-rename lines (`asset.NewModelBuilder(...)` → `asset.NewBuilderWithId(...)`,
`drop`/`monster`/`pet`/`reactor` equivalents): arguments are byte-identical across the rename in
every sampled case. `grep -iE 'topic|"/|Path\(|Route\('` over the reduced set: no hits. **PASS.**

### Step 5 — `Build()` validation errors (highest-risk step)

```
git diff -U0 main...HEAD -- '*.go' | grep -E '^[+-].*errors\.New\(|^[+-].*fmt\.Errorf\(' | grep -v '^[+-][+-]'
```
Reproduces at exactly **346** lines (136 `-`, 210 `+`), matching the pasted evidence exactly.

Independently re-derived the pairing (not trusted from prose): normalized both sides
(strip leading `-`/`+` and surrounding whitespace only — no other transformation) and diffed as
sorted multisets. Result: **0 unmatched deletions**, matching the report's `comm -23` claim.
Went further than the report: wrote a small script that tracks each `-`/`+` line's *source file*
(not just its text) to check for cross-package false-pairs — i.e., a message deleted from package
A that only coincidentally matches an unrelated addition in package B, which `comm` alone cannot
distinguish from a real relocation. For every uniquely-occurring (non-duplicated) error message,
the deletion file and the addition file are either identical or in the same package directory.
For duplicated messages (e.g. `fmt.Errorf("referenceId is required for item conditions")`, ×6),
del-count and add-count match exactly per string. **No false pairing found. PASS, independently
confirmed to a stronger standard than the report's own method.**

### Step 6 — struct field type changes (second-highest-risk step)

```
git diff -U0 main...HEAD -- '*/model.go' '*/entity.go' | grep '^-' | grep -v '^---'
```
Raw deletion count reproduces exactly at **6785**, matching "6785 raw deletions" in the report.

First-pass multiset match (normalized deletions vs. all branch-wide additions, `comm -23`)
reproduces exactly at **548 unmatched**, matching the report's "First pass: 548 unmatched."

This is where the report's evidence stops reproducing. The report states: *"Applying the branch's
own `ModelBuilder`→`Builder` rename to the deletion set before re-matching reduced this to 17
unmatched, which were then checked individually,"* then enumerates exactly **6** items (a bucket
of "5 `modelBuilder`(lowercase)/`SkillModelBuilder`/import-block lines" plus "1 `stance` line").
Three independent attempts to reproduce "17":

1. Literal `sed 's/ModelBuilder/Builder/g'` on the 548-line residual, re-`comm`'d against the same
   addition set → **8** unmatched, not 17.
2. Full `Counter`-based multiset diff (not two independent `comm` passes, which can silently
   "recover" a real deficit against the wrong pool) with the same naive substitution → **26**
   distinct lines / **45** total unmatched instances, not 17.
3. Full `Counter`-based multiset diff with the *complete* identifier rename the branch's own FR-14
   actually performs (`ModelBuilder`→`Builder`, `NewModelBuilder`→`NewBuilder`,
   `CloneModelBuilder`→`CloneBuilder`, confirmed against `design.md`'s own rename table at
   `docs/tasks/task-263-backend-guideline-conformance/design.md:16970-16976`) → exactly **1**
   unmatched: `stance             byte` (del=6, add=5) — the same line the report's own write-up
   independently traced by hand and confirmed belongs to the `Builder` struct, not `Model`.

None of these three reproduces "17." Method 3 — the most careful one, using the branch's actual
rename map rather than a partial guess — converges on the same substantive conclusion the report
reached (zero dropped `Model`/entity fields; the one real gap is the traced, benign `stance`
builder field). **The underlying PASS conclusion for Step 6 holds** under independent
verification stronger than the report's own.

However, the specific number pasted into `progress.md` (`17 unmatched, ... checked individually`)
does not reproduce under any of the three methods tried, and the write-up's own itemization
accounts for only 6 of the 17 items it claims to have individually checked — an internal
inconsistency in the pasted evidence itself, independent of my inability to reproduce the count.
Per this review's explicit charge ("A pasted output that does not reproduce is blocking"), this is
a **blocking finding**: not because a real Model-field drop was found (none was, under a strictly
more rigorous re-check), but because the audit's own second-highest-risk step ships a number that
cannot be reproduced and a claim ("17... checked individually") that its own write-up does not
support. That gap between the number claimed and the evidence shown is exactly the "spot-check
wearing a sweep's clothing" risk this review was asked to falsify.

## Commit hygiene

- `git show 29b52636b --stat`: only `docs/tasks/task-263-backend-guideline-conformance/progress.md`
  changed, 115 insertions, 0 deletions. No `.go` file touched. **Confirmed.**
- `git show 29b52636b | grep -nE '/home/[a-zA-Z0-9_.-]+'`: no hits. No literal home/absolute path
  in the committed text. **Confirmed.**

## Findings

### Blocking

- `docs/tasks/task-263-backend-guideline-conformance/progress.md:3961-3963` (Step 6 evidence) —
  the pasted "17 unmatched" count for the post-rename `comm -23` pass does not reproduce (I get 8,
  26, or 1 depending on rename-completeness, none of which is 17), and the write-up itemizes only
  6 of the 17 items it claims were "checked individually." The substantive conclusion (no dropped
  `Model`/entity field) holds under a stronger independent re-check, but the evidence as pasted
  cannot be trusted at face value and must be corrected or re-derived with a script whose output
  is pasted verbatim, per the brief's own Step 7 instruction ("Paste each command and its verbatim
  output").

### Non-blocking

- `docs/tasks/task-263-backend-guideline-conformance/progress.md:3959-3963` (Step 2 evidence) —
  pasted count is "11 lines total, all 11 are +"; independent reproduction gets 13 (2 additional
  lines in `design.md`/`rename_test.go`, both benign prose/test-assertion hits). Immaterial to the
  PASS conclusion but is the same class of miscounted evidence as the Step 6 finding above, just
  smaller — worth fixing in the same pass if Step 6 is corrected.
- Step 3's literal command now self-matches its own pasted "(none)" sentence once the Task 26
  commit itself is part of `main...HEAD` — an artifact of auditing-the-audit, not a defect. Future
  audit tasks of this shape should scope FR-17 code-pattern greps to `-- '*.go'` up front to avoid
  this self-reference, rather than relying on the reader to notice it (as I had to here).

## Not evaluable

- None. All six steps were independently re-run and their pairing logic independently
  reconstructed within this review's scope (the `29b52636b` diff plus the `main...HEAD` range it
  audits).

## Scope confirmation

Reviewed exactly the stated scope: commit `29b52636b` (docs-only, `progress.md` append) and the
six audit commands it claims to have run against `main...HEAD`, base `eaa5ce6f7`. No other file in
the branch was inspected beyond what was necessary to verify a pairing claim (e.g. `design.md`'s
FR-14 rename table, consulted only to confirm the correct identifier rename map for Step 6). No
scope mismatch — the commit is exactly what the brief describes.
