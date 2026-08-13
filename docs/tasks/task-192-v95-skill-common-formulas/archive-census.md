# Archive verification — OQ-5 census and the differential corpus

## Provenance

- Source: MinIO `atlas-wz` bucket, `shared/regions/<REGION>/versions/<VERSION>/Skill.wz`
  (endpoint via Secret `atlas-minio-credentials`, namespace `atlas-main`; reached through
  `kubectl -n minio port-forward svc/minio 19000:9000` + `curl --aws-sigv4`).
- The bucket was enumerated (not assumed): `GET /atlas-wz?list-type=2&prefix=shared/regions/&delimiter=/`
  returned exactly two regions, **GMS** and **JMS**. Enumerating each region's
  `versions/` prefix returned exactly the version set in the table below — **10 archives
  total**, every one of which has a `Skill.wz` object.
- Analysis program: a standalone scratch Go module (`wzscan`) under the session scratchpad,
  outside this repository, with `replace` directives to `libs/atlas-wz` and to
  `services/atlas-data/atlas.com/data` (so it imports `atlas-data/skill/formula` — the exact
  evaluator under test, Task 1's `formula.Parse`/`Expr.Evaluate`). Nothing in the repo was
  modified by the scratch module; nothing from it is committed except this document and the
  two files listed in the task brief. Every number below is a full-archive census (the walker
  recurses every `Directory`/`Image` in the file — root images and the `Dragon/` subdirectory
  — and visits every WZ leaf), not a sample.

## 1. Every archive found, censused

For each archive: `wz.Open`, assert `File.GameVersion()`, recursively walk every image whose
name matches `^[0-9]+$` (the numeric jobId convention `skill.ParseJobId` also uses), and for
each entry under `<jobId>/skill/<skillId>/` count whether it carries a `common` child, a
`level` child, both, or neither.

| region | version | `GameVersion()` | md5 (local download) | size (bytes) | total skills | `level` only | `common` only | both | neither | `common` nodes |
|---|---|---|---|---|---|---|---|---|---|---|
| GMS | 48.1 | 48 | `8ecfa4255e2fdbcb49c59dee0ba80dd8` | 40 799 768 | 277 | 277 | 0 | 0 | 0 | 0 |
| GMS | 61.1 | 61 | `0fd2b8d748c7d73b221a517083c3d427` | 49 080 651 | 337 | 337 | 0 | 0 | 0 | 0 |
| GMS | 72.1 | 72 | `511e56bc440ee6ef8251e8b0e914eace` | 63 104 709 | 453 | 453 | 0 | 0 | 0 | 0 |
| GMS | 79.1 | 79 | `0c4e02c190ebcb9ca2f61f7965b8a21e` | 69 425 260 | 509 | 509 | 0 | 0 | 0 | 0 |
| GMS | 83.1 | 83 | `b6017b42febe39bb45643ce3641dc788` | 76 874 780 | 534 | 534 | 0 | 0 | 0 | 0 |
| GMS | 84.1 | 84 | `b204811692dbb531135b6f9f0171f2c5` | 118 522 707 | 626 | 626 | 0 | 0 | 0 | 0 |
| GMS | 87.1 | 87 | `b7354fa0ec4aee4b795f65a52bf0673b` | 123 734 056 | 675 | 675 | 0 | 0 | 0 | 0 |
| GMS | 92.1 | 92 | `09884c7df174c2dd505cc3358af3b493` | 157 106 755 | 931 | 931 | 0 | 0 | 0 | 0 |
| **GMS** | **95.1** | **95** | `2d77583108eb928b65f2904196a894ef` | 171 667 159 | **954** | 319 | 633 | 2 | 0 | 635 |
| JMS | 185.1 | 185 | `28c420e60c2202d7f1a63f54586d5f46` | 144 728 113 | 795 | 795 | 0 | 0 | 0 | 0 |

The GMS 95.1 row reproduces `docs/tasks/task-192-v95-skill-common-formulas/wz-common-grammar.md`'s
counts exactly (954 / 319 / 633 / 2 / 635), from an independent re-run of the walk in this
task's own scratch program — not copied from that document.

The downloaded GMS 95.1 md5 matches the value pinned in this task's brief and in
`wz-common-grammar.md` (`2d77583108eb928b65f2904196a894ef`), confirming the same archive is
under test.

## 2. Does the fix apply outside GMS 95.1?

**No.** Every archive other than GMS 95.1 — all eight legacy GMS versions (48.1 through
92.1) and the one JMS archive (185.1) — has `common`-only count 0 and both-count 0: **zero
`common` nodes anywhere in the file.** Every skill entry in those nine archives carries only
a `level` subtree. This was checked exhaustively for every archive, not sampled.

Because no archive other than GMS 95.1 has a non-zero `common` count, the brief's follow-up
step ("tokenize every one of its `common` string values and report whether the character
classes and function names stay inside the FR-3 grammar") does not apply to any archive —
there is nothing to tokenize. **Task 192's `common`-formula fix is scoped to GMS 95.1 only;
it is a correctly-targeted fix, not one that happens to also need to cover any other censused
tenant version.** This is a positive finding from a full run, not an assumption: the walker
that found 635 `common` nodes in GMS 95.1 is the exact same code path that found 0 in each of
the other nine archives.

## 3. The GMS v95.1 differential corpus (design §10.3)

### Extraction

From the GMS v95.1 archive, the scratch program walked every `common` node (635 of them) and,
for each node:

- Read `maxLevel` (present on every node; `int`-typed 632×, `string`-typed 3× — both forms
  handled, matching `wz-common-grammar.md` §7).
- Collected every other **string-typed** child's raw value as an expression candidate, except
  the key `action` (the archive's one non-expression string, a client animation name —
  `formula.Parse("slashStorm2")` would legitimately fail, so it is excluded from the corpus
  by name, not silently swallowed as a Parse-error workaround). Non-string children (the
  `lt`/`rb` vectors, and the two rare int-typed outliers noted in `wz-common-grammar.md` §2 —
  `/311/skill/3111003/common/time` and `/3311/skill/33111005/common/x`) are not expressions
  and were skipped by type, not by value.
- Paired each surviving `(expression, maxLevel)` and deduplicated exact pairs.

This produced **843 distinct `(expression, maxLevel)` pairs** (640 distinct expression
strings) — every `(expression, maxLevel)` pair that exists under any `common` node in the
archive.

### Differential check

For every pair and every level `1..maxLevel`:

- **A** — this task's evaluator: `formula.Parse` (imported from `atlas-data/skill/formula` via
  the scratch module's `replace` directive — the literal Task 1 code, not a re-implementation)
  + `Expr.Evaluate(level)`.
- **B** — a from-scratch naive reference implemented in the scratch module only (never
  committed): standard precedence (`*`/`/` one tier, tightest, left-to-right; `+`/`-` one
  tier, loosest, left-to-right), `math.Ceil` for `u(...)`, `math.Floor` for `d(...)`, and a
  single `math.Trunc` toward zero applied once to the final result (no per-operation
  truncation, unlike A).

Result, over **14 711 total `(expression, level)` evaluations**:

- **`Parse` failures: 0.** Every one of the 843 pairs parsed under `formula.Parse` — the FR-3
  grammar is complete for this archive; there was nothing to escalate as a Task 1 grammar gap.
- **`Evaluate` failures: 0.**
- **A ≠ B disagreements: 0.** Every one of the 14 711 evaluations agreed between the
  client-accurate evaluator and the naive standard-precedence reference.

**Why zero disagreements is expected, not suspicious, and was independently checked rather
than assumed:** `wz-common-grammar.md` §3 already established that the `/` operator only ever
occurs *inside* a `u(...)`/`d(...)` call argument in this archive, never at the top level. This
census's own extraction confirms that directly: grepping the 640 distinct expression strings
for a `/` that is not immediately inside a `u(`/`d(` call's parentheses returns **zero
matches**. The client's precedence (`+` → `-` → `/` → `*`, loosest to tightest — see
`skill/formula/formula.go`'s package doc) and the naive standard precedence (`*`/`/` tied,
tightest; `+`/`-` tied, loosest) differ from each other *only* in how a bare `/` interacts with
a bare `*` or `-` at the same nesting level outside a call. Since this archive's data never
puts a bare `/` outside a `u()`/`d()` argument, that divergence never has an opportunity to
manifest — which is exactly what the differential run measured, not inferred. Had the archive
contained even one expression like `x/2*3` at the top level, A and B would disagree (client:
`x/(2*3)`; naive: `(x/2)*3`), and this section would instead report and adjudicate that
finding via IDA per the task's escalation rule. **No such expression exists in GMS v95.1's
`common` nodes**, so no IDA adjudication was required for this pass, and none was performed.

Spot-check against the task brief's own worked example (`6+2*u(x/5)`, present verbatim in the
archive as skill `1001003`'s `mpCon`): the corpus program independently produced
`6+2*u(x/5),1,8` and `6+2*u(x/5),20,14` — the exact two rows given in the brief's Step 3
example — confirming the extraction and evaluation pipeline reproduces the intended semantics
before the corpus was written out.

### Corpus written

`services/atlas-data/atlas.com/data/skill/formula/testdata/common_corpus.csv` contains one row
per distinct `(expression, level)` pair (deduplicated across the 843 source pairs, since two
pairs can share an expression and overlapping level range) — **14 711 rows**, plus a header.
Values are written via `encoding/csv`, which RFC-4180-quotes the one leading-space value
(`" 375+5*x"`, skill `2111002`'s `damage`) automatically.

## 4. Verification

```
$ go test ./skill/formula/... -run TestArchiveCorpus -v
=== RUN   TestArchiveCorpus
--- PASS: TestArchiveCorpus (0.01s)
PASS
ok  	atlas-data/skill/formula	0.008s

$ go test -race ./skill/formula/...
ok  	atlas-data/skill/formula	1.043s
```

## Summary

- 10 archives found (9 GMS, 1 JMS), all censused exhaustively.
- Only GMS v95.1 has any `common` nodes; the Task 192 fix is correctly scoped to that one
  archive/tenant version — every other censused archive is `level`-only and unaffected.
- The GMS v95.1 differential check ran 14 711 `(expression, level)` evaluations across 843
  distinct `(expression, maxLevel)` pairs from all 635 `common` nodes: 0 `Parse` failures, 0
  `Evaluate` failures, 0 A≠B disagreements between the client-accurate evaluator and a
  from-scratch naive standard-precedence reference. No IDA adjudication was required because
  no disagreement was found to adjudicate.

---

## Verification

Recorded at branch head, after merging `origin/main` (merge commit `f82f8305f`) and after
the FILE-05 refactor (`27ed73211`). Values below are observed command output, not
expectations.

### Go gates — `services/atlas-data/atlas.com/data`

| Gate | Result |
|---|---|
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `go test -race ./...` | exit 0 — all 65 packages `ok`, 0 failures |

Run both before and after the `origin/main` merge; clean both times.

`go.mod` unchanged (`git diff --stat -- services/atlas-data/atlas.com/data/go.mod` empty
against the merge-base), so `docker buildx bake atlas-data` was **not** required and was
not run.

### Repo-root guards

| Guard | Result |
|---|---|
| `tools/redis-key-guard.sh` | exit 0 |
| `tools/goroutine-guard.sh` | exit 0 |
| `tools/lint.sh --check` | exit 1 — see below |

`tools/lint.sh --check` exits 1, but not for this branch's code. It was run twice, alone,
with Node 22 on PATH. The failing-target set differs between runs — first run:
`atlas-transports`; second run: `atlas-skills`, `atlas-storage`, `atlas-summons` — and every
failure reads `Error: parallel golangci-lint is running`. That is the known cross-worktree
golangci-lint lock-contention footgun, not a lint violation. `services/atlas-data` — the only
Go module this branch touches — is absent from the FAIL list in both runs. `atlas-ui` reports
0 errors and 5 pre-existing baseline warnings, none in files this branch changed.

### atlas-ui

`npm run build` (Node 22.22.2) succeeded; 1881/1881 tests passed across 229 files. The 33
new `SkillEffect` members and 33 new `FIELD_LABELS` entries were cross-checked against the Go
`json:"..."` tags in `skill/effect/rest.go` — identical 33/33, character for character.

### Not performed: live deploy, re-ingest, and end-to-end checks

**Plan Task 8 Steps 3 and 4 were deliberately not performed.** The image was not deployed,
`POST /data/process` was not called, no re-ingest Job was created, the serving pods were not
restarted, and the live assertions on skill 1001003 (`maxLevel == 20`, `MPConsume` 8 → 14,
`duration` 110000 → 300000 ms) and the served-document census (expected 0 of 954 with
`maxLevel == 0`, down from 635) were **not** run. This was an explicit decision to keep the
live-cluster half out of this change.

Consequently the end-to-end evidence the plan calls for is **outstanding**, and the plan
treats Step 3's failure count as a blocking acceptance gate. The runbook in PRD §11 must be
executed before this change can be considered verified against a real archive. Note also
that the change serves nothing until a re-ingest rewrites the `content` blobs, and that the
ingest Job pod is a different process from the serving pods — the serving pods need a restart
afterwards.

What *is* verified without the cluster: the evaluator against a 14,711-row corpus extracted
from the real GMS v95.1 archive (0 parse failures, 0 A≠B disagreements), the full census of
all 10 available archives, and the module's own test suite.
