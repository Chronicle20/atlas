# Task 12 batch 1 review

Range reviewed: `3a0d27c57..a7fd717e1` (9 per-service commits + ledger commit + report commit).

Reviewer scope note per brief: the generator's own logic (Tasks 10-11) is not re-reviewed here.
This review checks the *application*: scoping, ledger integrity, and that the diff is nothing
but generator output for the 9 named services.

## Discovery

```
git diff --stat 3a0d27c57..a7fd717e1
```
74 files changed, 1544 insertions(+), 2 deletions(-): 72 `rest.go`/`rest_test.go` files across
the 9 services, plus the new ledger TSV and the batch report.

## Findings

### 1. Service scope — PASS

Each of the 9 per-service commits touches exactly one service's `services/<S>/` tree, confirmed
via `git show <sha> --name-only` for all 9 commits (`81922b270`, `3c21ef059`, `7a7e489b0`,
`60c3ec452`, `e66e25ceb`, `9185d8bbb`, `8ed6edaf3`, `0dddbda0d`, `f949577ac`). No cross-service
leakage.

The ledger commit (`d18c4e883`) and report commit (`a7fd717e1`) are docs-only (43-row TSV,
100-line report respectively) — confirmed via `git show --stat`.

`git diff --name-only 3a0d27c57..a7fd717e1` outside `services/atlas-{monster-death,messages,
consumables,npc-shops,inventory,query-aggregator,pets,monsters,maps}/` returns only the two docs
files. No non-tier-A, non-listed-service file was touched.

### 2. Diff content is pure generator output — PASS

`git diff --name-only ... -- services` restricted to non-`rest.go`/`rest_test.go` paths returns
nothing — every touched file under `services/` is one of those two filenames.

Every touched `rest.go` got exactly one `func Transform(...)` insertion (`grep -c "^+func
Transform"` = 36, matching 36 touched `rest.go` files 1:1). Spot-checked
`services/atlas-monster-death/atlas.com/monster/party/{rest.go,rest_test.go}` — shape matches
the Task-11-approved generator output exactly (flat-literal `Transform`, `TestTransformRoundTrip`
using `reflect.DeepEqual`).

FR-17 import-fold delta: total deletions across all 36 touched `rest_test.go` files = 2, both in
`services/atlas-messages/.../data/{foothold,monster}/rest_test.go`, both single-line
`import "testing"` folded into a group (`-import "testing"` / `+import (...)`) — exactly the
documented known-good fact, not a defect.

### 3. Ledger integrity (the manual-concatenation check) — PASS

`ledger-transform-rest-1.tsv`: 43 rows total, 36 `APPLIED` / 7 `SKIPPED`.

- No duplicate package rows (`cut -f1 | sort | uniq -d` → empty).
- The 43 rows are exactly the 43 tier-A rows from `classify-dom04.tsv` filtered to these 9
  services (independently re-derived per-service A-row counts from the classification file and
  summed to 43; per-service breakdown: monster-death 8, messages 8, consumables 7, npc-shops 4,
  inventory 4, query-aggregator 3, pets 3, monsters 3, maps 3 — matches report table exactly).
- The 36 `APPLIED` ledger rows are byte-identical (as a sorted set) to the 36 unique package
  directories touched by `rest.go`/`rest_test.go` changes in the diff (`diff` of the two sorted
  lists → "Files are identical"). No APPLIED row lacks a diff; no diff'd package is missing from
  the ledger.
- No SKIPPED package has any `rest.go`/`rest_test.go` change in the diff (36 touched dirs vs. 36
  APPLIED, 0 overlap with the 7 SKIPPED dirs).

The manual `/tmp` concatenation the implementer describes (working around the missing `-append`
flag) produced a clean, non-duplicated, non-dropped ledger.

### 4. SKIPPED rows are genuine — PASS

Verified all 7 skip reasons against source, not just trusted the ledger text:

- `atlas-messages/.../character`, `atlas-npc-shops/.../character`,
  `atlas-monsters/.../monster/consumable`, `atlas-maps/.../data/map/info`: all 4 confirmed
  single-return `Build() Model` (no `(Model, error)`) via `grep -n "func.*Build()"` on each
  package's `model.go`/`builder.go` — matches the documented `transform.go:401-406` limitation.
- `atlas-consumables/.../cash`: confirmed `spec map[SpecType]int32` field in `model.go:6` —
  genuinely unsupported map-typed field.
- `atlas-monsters/.../monster/mobskill`: confirmed `Summons []uint32` field in `rest.go:25` —
  genuinely unsupported slice field.
- `atlas-messages/.../data/map`: confirmed `Extract(_ RestModel) (Model, error) { return
  Model{}, nil }` — Extract discards the RestModel entirely and returns a zero-value Model with
  no field mapping the generator can invert; matches classify-dom04.tsv's "flat literal, 0
  fields" annotation and the ledger's "Extract maps no fields" reason.

No skip looks like silently dropped work; all 7 are genuine generator limitations already known
from Task 11, applied consistently.

### 5. Build/vet/test evidence — PASS (independently re-run)

Report claims `go build && go vet && go test` passed clean per service. Re-ran `go build ./...
&& go vet ./... && go test ./...` from each of the 9 service module roots in this worktree
(current HEAD, i.e. post-batch state): all 9 built and vetted clean; `go test ./... | grep -i
fail` returned no matches for all 9 (`atlas-monster-death`, `atlas-messages`, `atlas-consumables`,
`atlas-npc-shops`(`atlas-npc` module), `atlas-inventory`, `atlas-query-aggregator`, `atlas-pets`,
`atlas-monsters`, `atlas-maps`).

Note: `GOWORK=off` (used for the codemod invocation per the report's workaround) breaks these
services' own `go build` because they depend on `go.work` to resolve sibling `libs/` modules —
this is unrelated to the codemod and not a defect; the correct per-service verification command
omits `GOWORK=off`, which is what the report's Step 1 build/test loop already does (the
`GOWORK=off` is only used for invoking the *codemod itself*, not the per-service build/test
step — confirmed by re-reading the report's Step 1 example command, which drops `GOWORK=off` for
the `go build && go vet && go test` line).

### 6. Report accuracy — PASS

All per-service APPLIED/SKIPPED counts, skip reasons, and commit SHAs in the report table match
the ledger and the actual commits. The "4 single-return-builder skips" total is correct (verified
independently in Finding 4). The report's invocation-note about the missing `-append` flag and
the nested-Go-module `go run` workaround is a legitimate deviation from the brief's literal
command and is disclosed, not silently worked around.

## Not evaluable

None. All checks in scope were resolvable from the diff, the ledger, and the touched
packages' source.

## Scope confirmation

The work found matches the range and brief exactly: 9 per-service commits (one per named
service), a ledger commit, and a report commit — no extra services, no extra commits, no
unrelated file changes.
