# Review: Task 21-A — W1 `relocate` codemod, batch A (second half)

Range reviewed: `61032d0fa..HEAD` (8 commits: `af4f138fe`, `47869a92e`, `940e8ddc7`,
`260952918`, `49fcb23bb`, `37b380e81`, `719a9c0a0`, `59afb53af`).

Brief: `.superpowers/sdd/plan/task-21a-brief.md`
Report: `.superpowers/sdd/plan/task-21a-report.md`

## Scope confirmed

`git diff --stat 61032d0fa..HEAD --name-only` touches exactly: the codemod's own
`relocate.go`/`relocate_test.go`, the new `ledger-relocate-b.tsv`, and the six
service/lib module pairs (`model.go`/`connection.go`/`producer.go`/`recipients.go`
+ new `builder.go`) for `atlas-quest`, `atlas-channel` (2 groups), `atlas-constants`
(2 groups), `atlas-database`, `atlas-storage`, `atlas-saga-orchestrator`. No file
outside batch A's module list was touched. `atlas-tenants` has no diff, matching
the claim that both its rows are `ENTITY-BUILDER` and therefore skipped by design
— confirmed directly against `classify-file05.tsv` lines 99–100
(`services/atlas-tenants/atlas.com/tenants/configuration|tenant entity_builder.go
ENTITY-BUILDER`), which the codemod's `staticSkipReason` always skips.
`go.work.sum` remains modified in the working tree but is not part of any commit
in the range (`git log --oneline --all -- go.work.sum` shows no commit here) —
consistent with the pre-existing dirty state noted at dispatch.

## 1. Behavior preservation (pure relocation)

For each of the six module commits, ran `git diff -M --stat <c>^ <c>` (no rename
detected — expected, the codemod doesn't `git mv`) then the corrected
symmetric-blank-line asymmetry check
(`grep -vE '^\+package |^[+-]$|^\+import|^\+\t"|^\+\)|^-import|^-\t"|^-\)'`) on
each of:

- `47869a92e` (atlas-quest, `kafka/producer/saga/{producer,builder}.go`)
- `940e8ddc7` (atlas-channel, `party/{model,builder}.go` and
  `skill/handler/{recipients,builder}.go`)
- `260952918` (atlas-constants, `channel/{model,builder}.go` and
  `field/{model,builder}.go`)
- `49fcb23bb` (atlas-database, `{connection,builder}.go`)
- `37b380e81` (atlas-storage, `projection/{model,builder}.go`)
- `719a9c0a0` (atlas-saga-orchestrator, `validation/{model,builder}.go`)

Every non-import, non-package, non-blank line appears exactly once as `-` and
once as `+` with identical text — no signature, field, receiver, doc comment, or
body differs between the moved pair. No `init()` functions are present in any
touched file. No unexported helper changed visibility (`builder`/`Builder`
receivers and their exported/unexported status carry over unchanged, e.g.
`atlas-channel/party`'s lower-case `builder` struct stays lower-case in
`builder.go`). PASS.

One legitimate cosmetic import change, not a behavior change: in
`services/atlas-quest/.../producer.go`, `"github.com/segmentio/kafka-go"`
(unaliased) becomes `kafka "github.com/segmentio/kafka-go"` (explicit alias).
The package's declared name is `kafka` and the only remaining use
(`producer.go:16`, `model.Provider[[]kafka.Message]`) already refers to it as
`kafka`, so this is goimports/gofumpt making an implicit alias explicit — same
identifier, same binding, no functional change. Within "formatting
normalization" per the brief.

## 2. Import correctness (`libs/atlas-constants` alias fix)

`libs/atlas-constants/field/builder.go` imports
`_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"` (package `map`,
import path ending `.../map` — the exact alias-mismatch case `852c472b4` fixed).
The alias is correctly emitted, used consistently (`_map.Id`), and not
duplicated between `model.go` and `builder.go` — `model.go`'s own remaining
`FromId`/etc. no longer import `uuid`, `channel`, `_map`, or `world` since
those are only used by the moved `Builder` (confirmed via `git show 260952918 --
libs/atlas-constants/field/model.go`: a clean 45-line deletion with no import
line touched, meaning the codemod's import-pruning correctly determined
`model.go` no longer needs those symbols). No import dropped, no import
duplicated. PASS.

## 3. `af4f138fe` — the `-append` flag (non-mechanical, highest scrutiny)

Read `runRelocate`'s new branch (`relocate.go:628-634`) and
`splitLedgerLines`/`mergeLedgerLines`/`ledgerLineKey` (`relocate.go:639-691`).

- **Missing file**: `os.ReadFile` returns `os.IsNotExist(err) == true`, which is
  explicitly tolerated (`err != nil && !os.IsNotExist(err)` only errors on other
  I/O failures); `existing` is then `nil` bytes, `splitLedgerLines("")` returns
  `nil` (verified: `strings.TrimRight("", "\n")` is `""`,
  `strings.Split("", "\n")` is `[""]`, and the loop skips the one empty
  element). Correct — matches the real first invocation (atlas-quest, no
  pre-existing ledger) per the report.
- **Trailing newline present vs. absent**: `splitLedgerLines` does
  `strings.TrimRight(data, "\n")` before splitting, so a file with or without a
  final `\n` produces the same line set; each recovered line has its own `"\n"`
  re-appended before merge. Verified this is not just a claim — traced the
  logic by hand against both cases.
- **Duplicate-row risk on re-run**: `mergeLedgerLines` keys by
  `ledgerLineKey` (text before the first tab), which is the file path — the
  same key `relocateRepo`/`ledgerLines` use to record any row (`g.key()` =
  `filepath.Join(pkgDir, file)`, `relocate.go:70`). A key present in both
  `existing` and `additional` is overwritten, never duplicated, and the
  `order` slice only appends a key the first time it's seen. Re-running the
  same batch's invocation with `-append` therefore replaces its own rows in
  place rather than doubling them. Confirmed no duplication is possible by
  construction, not just by test.
- Format compatibility: `mergeLedgerLines`'s output format matches
  `relocateRepo`'s (`ledgerLines(ledger)` delegates to `Ledger.WriteTo`'s
  `"%s\tAPPLIED\t\n"` / `"%s\tSKIPPED\t%s\n"`, `report.go:108/110`), so a
  batch-B/C `-append` run reading batch A's on-disk ledger recovers exactly the
  same key format `ledgerLineKey` expects. Traced end-to-end, not assumed.

**Regression tests** (`TestMergeLedgerLinesAppendsDisjointBatches`,
`TestMergeLedgerLinesReplacesSameKey`, `TestSplitLedgerLinesRoundTripsWriteTo`):
these exercise `mergeLedgerLines`/`splitLedgerLines` as pure functions with
in-memory `[]string` fixtures. They constrain the disjoint-union and
same-key-replace behavior directly (a real regression would fail
`TestMergeLedgerLinesReplacesSameKey` if a future edit made duplication
possible), which is real coverage, not just exercise. What they do **not**
cover: `runRelocate`'s actual `os.ReadFile` + `os.IsNotExist` branch reading a
real file from disk (missing, present-with-newline, present-without-newline) —
that path is only validated by hand-tracing above and by the real
seven-invocation apply recorded in the report (first call had no file,
subsequent calls appended). Given this codemod's history (six defects, all
found on the real tree, none by unit tests), the missing integration-level test
for the disk-read branch is a gap worth naming, but the logic itself checks out
by inspection and by the real run. Non-blocking.

## 4. Ledger accuracy — defect found

`git show af4f138fe..59afb53af --stat -- docs/tasks/task-263-backend-guideline-conformance/ledger-relocate-b.tsv`
and direct inspection of the committed file:

```
$ grep -c APPLIED ledger-relocate-b.tsv   # 8
$ grep -c SKIPPED ledger-relocate-b.tsv   # 4
$ wc -l ledger-relocate-b.tsv             # 12
```

The 12 rows match `classify-file05.tsv`'s rows for batch A's seven modules
exactly (lines 1–3, 24–25, 94–100 — 12 rows), and the six per-module commits'
APPLIED count (8: atlas-constants×2, atlas-database×1, atlas-channel×2,
atlas-quest×1, atlas-saga-orchestrator×1, atlas-storage×1) plus the four
`ENTITY-BUILDER` skips is internally consistent and correct.

**`.superpowers/sdd/plan/task-21a-report.md:162` states "9 APPLIED, 4 SKIPPED,
13 rows total"** — this does not match the report's own printed ledger table
immediately above it (lines 148–160, which lists 8 APPLIED + 4 SKIPPED = 12
rows), and does not match the actual committed `ledger-relocate-b.tsv` (8
APPLIED, 4 SKIPPED, 12 rows, verified above). This is a real off-by-one in the
report's summary count, not a transcription artifact of this review — the
report's own table and the committed file agree with each other and disagree
with the report's prose.

This matters specifically because the review brief and `progress.md` both
flag that these counts feed Task 25 and the **required** 72-vs-73 group-count
reconciliation in Task 21-C (`progress.md`, "Open threads... 1. 72/73
reconciliation"). `progress.md` has not yet recorded batch A's tally (checked
`tail -60`; the last entry is the Gate 20 re-run, nothing about batch A yet),
so the erroneous "9/13" figure exists only in the report right now — but if a
future session (Task 21-C or 25) trusts the report's prose count instead of
re-deriving from the ledger file itself, it will silently corrupt the
reconciliation this task exists partly to support. The committed ledger file
is correct; only the report's narrative is wrong. **Blocking**: the report
must be corrected (or the next session must be explicitly told to ignore the
report's stated count and use the ledger file) before Task 21-C's
reconciliation runs.

## 5. Build/test/lint — spot-verified independently

Ran (not just re-read the report's claims):
- `libs/atlas-constants`: `go build ./... && go vet ./... && go test ./...` →
  all packages build, tests pass.
- `services/atlas-quest/atlas.com/quest`: same → pass.
- `services/atlas-channel/atlas.com/channel`,
  `libs/atlas-database`, `services/atlas-storage/atlas.com/storage`,
  `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`: `go build
  ./...` → all clean.
- `docs/tasks/task-263-backend-guideline-conformance/codemod`: `GOWORK=off go
  build/vet/test ./...` → `ok`.
- `tools/lint.sh --check --go <module>` (the repo's actual formatting/lint
  authority, `golangci-lint fmt` + lint) on all six touched modules → `0
  issues. lint.sh: OK` for every one, including atlas-storage and
  atlas-saga-orchestrator, which the report separately flagged as exercising a
  Step-2-regex blank-line false positive (verified: that false positive is in
  the *review-time diff check*, not in the committed code, and does not affect
  formatting-gate correctness).

Side note (non-blocking, informational): a bare `go run mvdan.cc/gofumpt -l`
(not the repo's pinned `golangci-lint fmt`) flags
`services/atlas-storage/atlas.com/storage/projection/builder.go` for import
grouping (wants `errors` separated from `atlas-storage/asset` into its own
group). This is a false alarm relative to this repo's actual authority —
`golangci-lint fmt --diff -c .golangci.yml` (the exact invocation
`tools/lint.sh --check` uses, v2.13.1, gofumpt+goimports with
`local-prefixes: github.com/Chronicle20/atlas`) produces **no diff** on the
same file. Recorded here only so a future reviewer doesn't re-chase the same
red herring with bare gofumpt.

## Not evaluable

None. The unit's full diff surface (codemod change, six module relocations,
ledger, report) was reviewed within scope.

## Self-review of this review

- Every PASS above is backed by a command actually run in this session, not
  by re-stating the report's claims.
- The one blocking finding (ledger count mismatch in the report prose) is
  located at a specific line (`task-21a-report.md:162`) and confirmed against
  both the report's own table and the committed ledger file, not asserted from
  memory.
