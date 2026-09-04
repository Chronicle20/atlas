# Review: Task 21 — Scaffolding and architecture documentation (FR-5)

Range reviewed as one unit: `06bb4733f..b86769b` (commits `3871257` docs change,
`b86769b` fix round correcting a stale cross-reference the first commit broke).

## Scope

`git diff --stat 06bb4733f..b86769b`:

```
.claude/skills/backend-dev-guidelines/resources/audit-checklist.md      |  2 +-
.claude/skills/backend-dev-guidelines/resources/scaffolding-checklist.md| 36 +++++++++++++++----
docs/architectural-improvements.md                                     | 18 +++++++++--
3 files changed, 47 insertions(+), 9 deletions(-)
```

Matches the brief's expected file set (two files in the primary commit, plus
`audit-checklist.md` in the fix round). `docs/adding-a-new-service.md` has an
empty diff over the range (`git diff 06bb4733f..b86769b -- docs/adding-a-new-service.md`
produced no output) — confirms it was read-only per FR-5.3. No Go source file
is touched anywhere in the range (`git diff --name-only ... | grep '\.go$'`
empty).

## Findings

### PASS — RESOLVED claim is true, not plausible

Re-ran all four verification commands independently:

```
find services -path '*/rest/request.go'          → (no output)
find services -path '*/database/connection.go'    → (no output)
grep -rl 'type HandlerDependency struct' --include=handler.go services → (no output)
grep -rn 'd\.DB()' --include='*.go' services | wc -l → 0
```

All match the brief's controller-verified facts exactly. `docs/architectural-improvements.md:355-379`
correctly states the entry as fully RESOLVED.

### PASS — 175 (not 174) call-site count

Recomputed independently: `git diff -U0 31a791e3a..HEAD -- services | grep '^-' | grep -o 'd\.DB()' | wc -l` → **175**;
same on `+` lines → 0. The committed text at `docs/architectural-improvements.md:366`
reads "175 call sites" — correct, and does not carry forward the plan's stale
174 figure.

### PASS — 19 services / 21 files / 2 deletions

`git diff --name-status 31a791e3a..HEAD -- services | grep 'rest/handler.go'`
returns 21 rows, 2 of which are `D` (`atlas-events`, `atlas-fame`) → 19 net
conversions. `docs/architectural-improvements.md:364-365,371` states this
correctly and names the same two deleted packages.

### PASS — alias-form count referenced correctly

`grep -rl '= server.HandlerDependency' --include=handler.go services | wc -l` → 59,
matching the brief. This number is not itself quoted verbatim anywhere in the
committed docs (the docs cite "19 services" and "175 call sites" instead),
so there is nothing to cross-check it against in the diff — noted for
completeness, not a defect.

### PASS — SCAFFOLD-10 row and heading

`## Audit verification — SCAFFOLD-01..10` (scaffolding-checklist.md:173).
Exactly 10 `| SCAFFOLD-` rows present (`SCAFFOLD-01` through `SCAFFOLD-10`,
scaffolding-checklist.md:190-199), none missing or duplicated. The new row
matches the existing `| ID | How to verify | Pass criteria |` column form and
its markdown table syntax renders as a valid row (verified by inspection of
lines 189-199, consistent pipe count and cell structure with sibling rows).

### PASS — renumbering integrity

Full internal heading grep:

```
13:## 1. Build & CI registration
19:## 2. Kubernetes wiring
27:## 3. Bruno Collection (REST services only)
90:## 4. REST Handler Scaffolding (REST services only)
113:## 5. Ingress Route (REST services only)
128:## 6. Tenant Opcode Template (atlas-channel packet writers/handlers only)
154:## 7. Post-Scaffold Verification
```

`## Conditional Steps` (scaffolding-checklist.md:167-169) reads:

```
- Steps 3, 4, and 5 only apply to services that expose REST endpoints. Kafka-only services skip Bruno, REST handler scaffolding, and ingress.
- Step 6 only applies when the change introduces new atlas-channel packet writers or recv handlers. Pure-REST services and Kafka-only services skip the opcode template seed.
```

Correctly updated: the new step 4 (REST Handler Scaffolding) is folded into
the REST-only exemption alongside the renumbered steps 3 and 5, and marked
REST-only as the brief required. No stale `Step N` or `## N.` reference found
anywhere else in the file.

### PASS — fix round's anchor correction

`audit-checklist.md:64` diff in `b86769b`:

```
-| **Scaffolding** (SCAFFOLD-01..09) | ... | [scaffolding-checklist.md](scaffolding-checklist.md#audit-verification--scaffold-0109) |
+| **Scaffolding** (SCAFFOLD-01..10) | ... | [scaffolding-checklist.md](scaffolding-checklist.md#audit-verification--scaffold-0110) |
```

Both the visible label and the anchor were corrected. The anchor
`#audit-verification--scaffold-0110` matches the GitHub slug for the new
heading `## Audit verification — SCAFFOLD-01..10` (em dash and non-alphanumerics
stripped/hyphenated the same way as the surrounding, unedited anchors in the
same table use for their own headings — confirmed by the pattern of sibling
rows in the table, e.g. `#audit-verification--dom-29`).
`grep -rn 'audit-verification--scaffold' --include='*.md' .` repo-wide shows
exactly one inbound reference, and it is the corrected line. No other stale
reference to the old anchor survives.

### PASS — house style and preservation

`docs/architectural-improvements.md`'s "Low: Duplicated Database/REST
Boilerplate" entry (line 355 on) now mirrors the "Low: Kafka Retry Logic"
entry immediately below it: `### Status: RESOLVED` → `**Implemented:**` block
→ `**Files:**` block → `### Original Problem` → `### Original Recommendation`.
The original `### Problem` / `### Recommendation` prose ("copy-pasted across
25+ services...", "Extract into shared libraries...") survives verbatim under
the `### Original Problem` / `### Original Recommendation` headings — the
entry was amended, not deleted.

### PASS — hygiene

- No CRLF introduced: `git diff 06bb4733f..b86769b | grep -cP '\r$'` → 0.
- No literal absolute/home path written into any committed file:
  `git diff 06bb4733f..b86769b | grep -n '/home/\|/Users/'` → no output.
- No Go source file touched in the range.
- Diff line counts (47 insertions / 9 deletions across 3 files) are consistent
  with genuine content edits, not a wholesale reformat.

## Non-blocking finding

- **`scaffolding-checklist.md:94` self-contradicts its own named reference
  file.** The new REST Handler Scaffolding step tells implementers to "copy
  the pattern from `services/atlas-guilds/atlas.com/guilds/rest/handler.go`,
  not from an older service" (line 93-94), then two lines later states "Do
  not declare a local `ParseInput` wrapper — no service calls one" (line 99).
  Independently verified: `services/atlas-guilds/atlas.com/guilds/rest/handler.go`
  (at current HEAD, `8c28a70b3`) itself declares exactly such a local
  `ParseInput[M]` wrapper (lines 20-22), which is indeed uncalled elsewhere in
  the repo (`grep -rn 'ParseInput\[' --include='*.go' services/atlas-guilds/`
  shows only the declaration itself, and the wrapper's caller-set is empty
  across the fleet — confirmed the wider "no service calls one" claim is
  literally true across all 36 services that still declare this dead wrapper).
  So the "no service calls one" sentence is factually accurate, but the
  checklist points a future implementer at a copy-source file that itself
  contains the exact anti-pattern the very next sentence tells them not to
  replicate. A reader diffing their new file against the named reference will
  see the wrapper present and may copy it. This is pre-existing dead code in
  `atlas-guilds/rest/handler.go` (untouched by task-301's diff, out of this
  unit's edit scope to fix), but the inconsistency was introduced by this
  task's own new prose and is worth a follow-up one-line note (e.g. "guilds'
  file still carries a dead `ParseInput` wrapper — do not copy that part") or
  picking a cleaner reference service.

## Not evaluable

- None. Every factual claim added by this range (RESOLVED status, the three
  verification commands, the 175/19/2 counts, the SCAFFOLD-10 row, the
  renumbering, the anchor, the house-style mirroring) was checked directly
  against repo state within the review surface.

## Verdict rationale

All controller-verified facts reproduce exactly. The RESOLVED claim is true.
The 175-vs-174 correction was applied. The renumbering left no stale internal
references. The fix round's anchor correction is accurate and complete, with
no other stale inbound reference surviving repo-wide. House style is mirrored
correctly and the original entry preserved. One non-blocking documentation
inconsistency found (self-contradictory reference-file guidance for
`ParseInput`), not blocking because it is factually true as written and the
underlying dead code is out of this task's edit scope.
