# Review: Task 21-C — batch C of `relocate` + ledger close-out

**Range:** `b7f3e8d90..9a3a85e29` (commits `180176796`, `c3335b9b6`, `9a3a85e29`)
**Brief:** `.superpowers/sdd/plan/task-21c-brief.md`
**Report:** `.superpowers/sdd/plan/task-21c-report.md`

## Scope confirmed

`git diff --stat b7f3e8d90..9a3a85e29` shows 33 changed files: 28 code files (14
`model.go`/`builder.go` pairs across `services/atlas-npc-conversations` and
`services/atlas-cashshop`), plus 5 docs files (`agent-ledger.tsv`,
`ledger-relocate-b.tsv`, `progress.md`, `review-task-21a.md`,
`review-task-21b.md`). The three docs files `agent-ledger.tsv`,
`review-task-21a.md`, `review-task-21b.md` were actually introduced by
`94d288cb5`, a prior handoff commit that sits inside the given range but
precedes and is unrelated to 21-C's three commits (`180176796`, `c3335b9b6`,
`9a3a85e29`) — not this unit's work, and not itself a problem (docs-only,
already reviewed/gated per `progress.md`). Noted for completeness, not a
scope-mismatch finding.

The 14 touched `model.go` files break down exactly as the brief specified: 5 of
6 `services/atlas-npc-conversations` rows (`conversation/item`,
`conversation/npc`, `conversation/recipe`, `saved_location`, `validation` —
`conversation/quest/model.go` correctly not applied, see below) and all 9 of
`services/atlas-cashshop`'s non-`reference_data.go` rows.

## 1. Purity of the relocation — PASS

Re-derived independently (not via the report's filter). For each of the 14
touched `model.go`/`builder.go` pairs, extracted the `-` lines removed from
`model.go` and the `+` lines added to `builder.go`, stripped package/import/
blank lines with a corrected `grep -P` filter (GNU `grep -E` does not expand
`\t` to a literal tab here — confirmed independently:
`printf '\t"time"\n' | grep -E '^\t?"'` → no match, `grep -P` → match; this is
the same defect the report called out), then `diff`'d the two line sets
directly (not sorted — so line order is also checked). All 14 files came back
byte-identical with zero remaining lines after filtering:

```
services/atlas-cashshop/.../cashshop/inventory/asset/{model,builder}.go       SYMMETRIC
services/atlas-cashshop/.../cashshop/inventory/compartment/{model,builder}.go SYMMETRIC
services/atlas-cashshop/.../cashshop/inventory/{model,builder}.go             SYMMETRIC
services/atlas-cashshop/.../character/compartment/{model,builder}.go         SYMMETRIC
services/atlas-cashshop/.../character/inventory/{model,builder}.go           SYMMETRIC
services/atlas-cashshop/.../character/{model,builder}.go                     SYMMETRIC
services/atlas-cashshop/.../coupon/batch/{model,builder}.go                  SYMMETRIC
services/atlas-cashshop/.../coupon/{model,builder}.go                        SYMMETRIC
services/atlas-cashshop/.../coupon/redemption/{model,builder}.go             SYMMETRIC
services/atlas-npc-conversations/.../conversation/item/{model,builder}.go    SYMMETRIC
services/atlas-npc-conversations/.../conversation/npc/{model,builder}.go     SYMMETRIC
services/atlas-npc-conversations/.../conversation/recipe/{model,builder}.go  SYMMETRIC
services/atlas-npc-conversations/.../saved_location/{model,builder}.go       SYMMETRIC
services/atlas-npc-conversations/.../validation/{model,builder}.go           SYMMETRIC
```

Spot-checked `character/model.go` (`services/atlas-cashshop/atlas.com/cashshop/character/model.go:303-451`
in the pre-image) directly: the tail-removal in `model.go` is a straight
"deleted from here, appears verbatim in `builder.go`" — no re-derivation of
signatures or field names.

`saved_location/model.go`'s aliased import
(`_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"`) and the
various `builder.go` files' own newly-introduced import blocks (`"time"`,
`"github.com/google/uuid"`, etc. — legitimately new because a package's
`import` block is not itself "moved," it's re-derived per file from what each
file's remaining code needs) were the source of my filter's first-pass false
positives too, before correcting to `grep -P`. Consistent with the report's
own account of needing the same fix mid-task.

`go build`, `go vet`, `go test` were re-run for both modules from a clean
worktree (not trusting the report's pasted output) and are green:

```
$ cd services/atlas-cashshop/atlas.com/cashshop && go build ./... && go vet ./... && go test ./...
ok in all packages
$ cd services/atlas-npc-conversations/atlas.com/npc && go build ./... && go vet ./... && go test ./...
ok in all packages
```

No CRLF spot-checked in three of the touched `builder.go` files (`grep -rlP
'\r'` → none) — consistent with the report's claim of zero CRLF across all 14.

## 2. Reserved files untouched — PASS

```
$ git diff --stat b7f3e8d90..9a3a85e29 -- '*conversation/model.go' '*reference_data.go'
(empty)
```

Neither `services/atlas-npc-conversations/.../conversation/model.go` (Task 23)
nor `services/atlas-cashshop/.../asset/reference_data.go` (Task 22) appear
anywhere in the diff.

## 3. Ledger counts — PASS, independently reconciled

```
$ wc -l ledger-relocate-a.tsv ledger-relocate-b.tsv
 34 ledger-relocate-a.tsv
 36 ledger-relocate-b.tsv
$ cat ledger-relocate-a.tsv ledger-relocate-b.tsv | cut -f2 | sort | uniq -c
     64 APPLIED
      6 SKIPPED
```

64 + 6 = 70 = 34 + 36. Matches the report's claimed **64 APPLIED / 6 SKIPPED /
70 rows** exactly — no off-by-one, unlike batches A and B.

Append-only check on the TSV itself:

```
$ git diff b7f3e8d90..9a3a85e29 -- ledger-relocate-b.tsv | grep -c '^-[^-]'
0
```

Zero deletion lines in the diff to `ledger-relocate-b.tsv` — the append is
purely additive, and the 15 new `+` lines (9 `atlas-cashshop` `APPLIED`, 5
`atlas-npc-conversations` `APPLIED`, 1 `atlas-npc-conversations` `SKIPPED` for
`conversation/quest/model.go`) match the 14 files-touched + 1
codemod-internally-skipped file exactly. `ledger-relocate-a.tsv` has zero diff
across the whole range (untouched, as required — it's Task 20's read-only
half).

**"APPLIED + SKIPPED must equal the count of `RELOCATE` + `HAS-BUILDER-GO` rows
in `classify-file05.tsv`" (brief Step 5, literal reading) — does not hold
literally, but the report/`progress.md` resolve why via Step 6, and I verified
the resolution arithmetically:**

```
$ cut -f4 classify-file05.tsv | sort | uniq -c
      4 ENTITY-BUILDER
      1 EXCLUDED-TREE
      2 HAS-BUILDER-GO
     93 RELOCATE
$ awk -F'\t' '($4=="RELOCATE"||$4=="HAS-BUILDER-GO"){print $1"\t"$2}' classify-file05.tsv | sort -u | wc -l
68
```

93 + 2 = 95 raw rows collapse to 68 distinct `(pkgDir,file)` groups (four files
carry >1 builder row: `reference_data.go`, `conversation/model.go`,
`conversation/quest/model.go`, `pets/data/pet/model.go`). Neither 95 nor 68
equals the ledger's 70. I traced the gap by hand and it closes exactly:

- Of the 68 `RELOCATE`/`HAS-BUILDER-GO` groups, 2 (`reference_data.go`,
  `conversation/model.go`) are the Task 22/23 reserved files — never passed
  to the codemod, so they never enter the ledger at all (confirmed: `grep -iE
  'reference_data|conversation/model.go' ledger-relocate-a.tsv
  ledger-relocate-b.tsv` → no match). 68 − 2 = 66.
- Of those 66, 2 (`pets/data/pet/model.go`,
  `conversation/quest/model.go`) are `SKIPPED` by the codemod's own
  "multiple builders in file" check → 64 `APPLIED` + 2 `SKIPPED` = 66. ✓
  matches the exact `APPLIED` count.
- Separately, the ledger *also* contains 4 `SKIPPED` rows for files classified
  `ENTITY-BUILDER` in `classify-file05.tsv` (`entity_builder.go` in
  `atlas-quest` ×2, `atlas-tenants` ×2 — confirmed via `grep -E
  'entity_builder' classify-file05.tsv`, disposition is `ENTITY-BUILDER`, not
  `RELOCATE`/`HAS-BUILDER-GO`). These were evidently in scope for earlier
  codemod invocations (batch A/B, prior to this batch) and recorded as
  `SKIPPED` there.
- 66 + 4 = **70**, exactly the ledger total.

This means the report's Step 5 output is numerically correct, but neither the
report nor `progress.md`'s Step 6 write-up ever states the explicit `66 = 68 −
2` / `70 = 66 + 4` chain that reconciles "70" against the brief's literal
"count of `RELOCATE`+`HAS-BUILDER-GO` rows" framing — it jumps straight to the
72/73 comparison (which is the more load-bearing number, see §4) without
closing this particular loop. I verified it by hand above and it is airtight
and self-consistent; **this is a documentation-completeness gap, not an
arithmetic error**, non-blocking.

## 4. The 72/73 reconciliation — PASS, independently re-derived

Re-derived from `classify-file05.tsv` alone, not from the report's or either
prior narrative's numbers:

- 100 raw rows = 93 `RELOCATE` + 2 `HAS-BUILDER-GO` + 4 `ENTITY-BUILDER` + 1
  `EXCLUDED-TREE`.
- `RELOCATE`+`HAS-BUILDER-GO` collapse to 68 distinct `(pkgDir,file)` groups
  (verified above).
- 68 + 4 `ENTITY-BUILDER` = **72**, matching `design.md:141-147`'s "Distinct
  packages: 72" for FILE-05 exactly (confirmed by reading
  `design.md:141-147` directly).
- `libs/atlas-packet/model/skill_usage_info.go` is confirmed the sole
  `EXCLUDED-TREE` row (`grep -P '\tEXCLUDED-TREE$' classify-file05.tsv` → one
  match), and `design.md:576-579` (§8.1) explicitly lists "FILE-05 excluded
  tree (1 — `libs/atlas-packet/model/skill_usage_info.go:143`, PRD §7/FR-18)"
  as a category distinct from the 72-package FILE-05 transform surface.
  72 + 1 = **73**, matching Task 19's real-tree dry-run total (64 APPLIED + 9
  SKIPPED, per `progress.md:2753-2754`).

Both prior narratives (Task 19's "grouping-unit effect" and the Task 19
reviewer's "quest/pets are undiscovered multi-builder files") are indeed
disproven by this arithmetic exactly as the report states: collapsing rows
into groups only ever *reduces* the count (100→68), so it cannot explain a
number 1 *above* 72; and `conversation/quest/model.go` /
`pets/data/pet/model.go` are already two of the four multi-builder files
folded into the 68-group total that sums to 72 — removing them would give 70,
not push the total to 73. The `EXCLUDED-TREE` explanation is the only one
consistent with the source data, and I independently confirm it.

This conclusion is safe for Task 25 to build `exemptions.md` from.

## 5. The two unattributed `SKIPPED` rows — confirmed genuine

```
$ cat ledger-relocate-a.tsv ledger-relocate-b.tsv | awk -F'\t' '$2=="SKIPPED"'
services/atlas-pets/atlas.com/pets/data/pet/model.go	SKIPPED	multiple builders in file
services/atlas-npc-conversations/atlas.com/npc/conversation/quest/model.go	SKIPPED	multiple builders in file
services/atlas-quest/atlas.com/quest/quest/entity_builder.go	SKIPPED	entity builder
services/atlas-quest/atlas.com/quest/quest/progress/entity_builder.go	SKIPPED	entity builder
services/atlas-tenants/atlas.com/tenants/configuration/entity_builder.go	SKIPPED	entity builder
services/atlas-tenants/atlas.com/tenants/tenant/entity_builder.go	SKIPPED	entity builder
```

Checked `design.md`'s §8.1 exemptions enumeration (`design.md:576-579`)
directly: it lists "FILE-05 entity builders (4)" and "FILE-05 excluded tree
(1)" — nothing for a "multiple builders in file" category, and no entry
naming either `pets/data/pet/model.go` or
`conversation/quest/model.go`. Confirmed: these two are genuinely
unaccounted for anywhere in the design's exemption list, not Task 22 (scoped
to `reference_data.go` only), not Task 23 (scoped to `conversation/model.go`
only). The report's finding is accurate and was disclosed transparently
(surfaced under "Concerns," not buried). This is a real, correctly-identified
open item for Task 25 to route (hand-split commit or new `exemptions.md`
entry) — not a defect in 21-C's own work, since the codemod's `SKIPPED`
disposition for genuinely-multi-builder files is the safe, correct behavior
for *this* task.

## Other checks

- Module-scoped `go build`/`go vet`/`go test` re-run independently from a
  clean tree for both `services/atlas-cashshop/atlas.com/cashshop` and
  `services/atlas-npc-conversations/atlas.com/npc` — both green, matching the
  report's pasted output.
- Commit shape: `180176796` (npc-conversations relocation), `c3335b9b6`
  (cashshop relocation), `9a3a85e29` (ledger + progress.md, docs-only) — each
  commit's file list matches its stated scope; no code and ledger changes are
  interleaved in one commit.
- No hand-editing of `ledger-relocate-b.tsv` detected (all 15 new lines are
  additive tail insertions matching the codemod's `-append` output shape, not
  scattered edits).

## Not evaluable

- Whether the codemod's "multiple builders in file" skip condition itself is
  correctly implemented — out of this unit's diff (`codemod/relocate.go` was
  not touched by `180176796`/`c3335b9b6`/`9a3a85e29`); the codemod's behavior
  is inherited from prior batches' fix (`ac3fa65a3`), already reviewed there.
- `tools/lint.sh --check --go` / `tools/verify.sh` for this range — explicitly
  out of scope per the dispatch (a concurrent `--quick` gate covers it).

## Verdict rationale

No blocking findings. The relocation is verbatim-pure across all 14 files
(independently re-derived, not trusting the report's filter), the two reserved
files are untouched, the ledger counts are grounded and internally consistent,
and the 72/73 reconciliation is correct and independently reproducible. The
two open, non-blocking items are: (a) the report/`progress.md` never states
the explicit `66 = 68 − 2` / `70 = 66 + 4` chain that reconciles the ledger
total against the brief's literal Step 5 framing (I derived and verified it
myself; it holds), and (b) the two unattributed `SKIPPED` rows genuinely need
routing before Task 25, correctly flagged by the report itself.

```text
verdict: APPROVED_WITH_FINDINGS
artifact: docs/tasks/task-263-backend-guideline-conformance/review-task-21c.md
scope_confirmed: diff of b7f3e8d90..9a3a85e29 (commits 180176796, c3335b9b6, 9a3a85e29) — 14 model.go/builder.go relocation pairs across services/atlas-npc-conversations and services/atlas-cashshop, plus the ledger-relocate-b.tsv append and progress.md Step 5/6 reconciliation. Reserved files conversation/model.go and reference_data.go confirmed untouched.
blocking: 0
non_blocking: 2
  - docs/tasks/task-263-backend-guideline-conformance/progress.md (Step 5/6, ~line 3258-3298) — the ledger's 70-row total is never explicitly reconciled against the brief's literal "APPLIED + SKIPPED must equal RELOCATE+HAS-BUILDER-GO row count" framing (95 raw / 68 groups); the 72/73 answer is given instead. I verified by hand that 70 = 66 (68 groups − 2 Task-22/23 reserved, never ledgered) + 4 (ENTITY-BUILDER rows also recorded SKIPPED) and it is airtight, but the write-up should say so for the next reader.
  - services/atlas-pets/atlas.com/pets/data/pet/model.go and services/atlas-npc-conversations/atlas.com/npc/conversation/quest/model.go — both SKIPPED with no destination in design.md's §8.1 exemptions list (confirmed absent). Correctly surfaced by the report as an open item; must be resolved (hand-split or new exemptions.md entry) before Task 25.
not_evaluable: 2
