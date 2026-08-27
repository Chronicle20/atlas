# Review: Task 14 — propagate sub-directory parse failures; count ingest failures

Range reviewed: `8c86277..8e1f7201f` (single commit `8e1f7201f`, 4 files, +294/-10).

## Scope confirmed

Matches the brief exactly: `libs/atlas-wz/wz/directory.go` (the `Warnf`-and-continue
replaced by a returned error), a new `libs/atlas-wz/wz/directory_error_test.go`, the
`serializeDirectory`/`SerializeToDirectory` accounting change in
`services/atlas-data/atlas.com/data/data/wztoxml/adapter.go`, and a new test in
`adapter_test.go`. `runtime.go` has zero diff in this range (confirmed with
`git diff 8c86277..8e1f7201f -- .../runtime.go` — empty). No scope drift.

## FR-8: directory-parse failure propagation

**PASS.** `libs/atlas-wz/wz/directory.go:122` now does
`return nil, fmt.Errorf("parse sub-directory [%s]: %w", entryName, err)` inside the
`r.Peek` closure; the enclosing `parseDirectory` (`directory.go:118-120`) immediately
returns `nil, fmt.Errorf("parse sub-directory [%s]: %w", entryName, err)` on any error
from `r.Peek`, so no partially-populated `*Directory` escapes. Traced up through
`file.go:171-174` (`openWithReader`): on `parseRoot` error it does `f.Close(); return
nil, fmt.Errorf(...)` — the file handle is closed and `Open` returns a nil `*File`
alongside the error. No leak, no partial object.

The error names the failing entry (`entryName`) and wraps the cause with `%w` — both
required by the brief.

### Independent verification (not taken on trust)

- Reverted `directory.go` to the pre-commit blob (`git show 8c86277:...`) and reran
  `TestSubDirectoryParseFailurePropagates`/`TestValidNestedArchiveStillOpens`: the first
  test genuinely FAILS against old behaviour (`Open succeeded on a corrupted nested
  directory, want error`), confirming the RED evidence in the implementer's report is
  real, not fabricated. Restored the file afterward (`git diff --stat` clean).
- Same independent revert-and-rerun for `adapter.go` against
  `TestSerializeDirectoryCountsFailures`: genuinely FAILS old behaviour (`per-archive
  summary lines = 0, want 1`). Restored afterward, confirmed clean.
- Ran the new tests forward (fix applied): all four PASS.
- **Full-sweep re-run of the "legitimate archives still open" claim** (not a
  spot-check). Located a real v83 GMS archive set at
  `/mnt/e/Programs/Nexon/GMS/MapleStory83/` — the same 17 archives as the report's
  table — and ran `wz.Open` against **every one**, recursively walking the whole
  directory tree (`countTree`) so a silently-dropped subtree would show up as a
  reduced image count, via a temporary uncommitted test (`wz/zzz_sweep_test.go`,
  deleted after use; `git diff --stat -- libs/atlas-wz/` clean afterward). Result with
  the fix applied — 16 of 17 open cleanly:

  `Base.wz` dirs=15 images=3 · `Character.wz` dirs=17 images=7201 · `Effect.wz`
  images=17 · `Etc.wz` images=22 · `Item.wz` dirs=6 images=155 · `Map.wz` dirs=14
  images=5602 · `Mob.wz` dirs=1 images=1568 · `Morph.wz` images=42 · `Npc.wz`
  images=1620 · `Quest.wz` images=6 · `Reactor.wz` images=419 · `Skill.wz` images=76 ·
  `Sound.wz` images=44 · `String.wz` images=20 · `TamingMob.wz` images=7 · `UI.wz`
  images=19.

  These match the report's table exactly on every archive (including the
  `Reactor.wz` 419 figure) except `Map.wz`, where I count 5602 images against the
  report's 5552. That is a different fixture directory (`MapleStory83/` here vs the
  report's `tmp/83.1_wz/` — 83 vs 83.1), so the 50-image delta is a fixture
  difference, not a decode difference; the archives that matter for this change all
  open and enumerate their full nested trees.

  Critically, `Character.wz` (dirs=17), `Base.wz` (dirs=15), `Map.wz` (dirs=14),
  `Item.wz` (dirs=6) and `Mob.wz` (dirs=1) all contain genuine **nested
  sub-directories** — exactly the code path this change made fail-hard — and all
  enumerate fully. The change does not break real nesting.

- The one sweep failure, `List.wz`
  (`unable to parse WZ header: invalid WZ magic: expected PKG1, got`), is
  **confirmed pre-existing**, not caused by this change. I re-ran it against the
  reverted pre-commit `directory.go` and got the byte-identical error. It is also
  mechanically impossible for this change to cause it: `parseHeader`
  (`file.go:161`) runs and returns before `parseRoot` (`file.go:171`) is ever
  reached. `List.wz` is simply not a PKG1 archive.

## Blast radius

**PASS, with one gap in the brief's own survey (not a code defect).** Grepped the
whole repo for `wz.Open(` outside tests:

- `services/atlas-data/atlas.com/data/data/workers/runtime.go:144,209` — both already
  check the error and return before touching the (nil) `*wz.File`. Confirmed by reading
  `monolithFile` and `OpenArchive`.
- `libs/atlas-wz/wzdiff/run.go:50,170`, `libs/atlas-wz/wzdiff/selfcheck.go:55` — all
  wrap and return the error immediately, no partial-file use.
- `services/atlas-renders/atlas.com/renders/storage/wzcache.go:135` — **not named in
  the brief's blast-radius survey**, but independently checked: it also
  removes the downloaded file and returns the wrapped error on `wz.Open` failure
  (`wzcache.go:133-138`), never touching a partial file. Not a defect — the brief's
  survey was simply incomplete (it covers only `atlas-data`, not `atlas-renders`) — but
  worth flagging since the brief explicitly asserted the survey was exhaustive
  ("already surveyed"). No behavioural gap results from the omission.

No caller anywhere in the repo proceeds with a nil/partial `*wz.File` after an `Open`
error. Blast-radius claim holds.

## Observability NFR

**PASS.** `serializeDirectory` (`adapter.go:51-71`) accumulates `failed`/`total`
recursively and returns them; `SerializeToDirectory` (`adapter.go:28-42`) logs exactly
one `Warnf` per archive: `"wz archive [%s]: %d of %d images failed to serialize"`. The
per-image `Warnf` (`adapter.go:57`) gained the archive name, matching the brief. No
per-property logging exists anywhere in the diff — confirmed by reading the full diff,
not just the test's own assertion.

`TestSerializeDirectoryCountsFailures` (`adapter_test.go:91-172`) asserts:
exactly one per-image failure line naming `Bad`, exactly one summary line
(`"1 of 2 images failed to serialize"`), **no** failure line for `Good`, and **no**
line containing `"price"` (the property name) — this is the absence check the brief
demanded, not just a presence check. Confirmed the test genuinely fails without the
fix (see independent verification above).

One non-blocking observation: the summary line is logged unconditionally at `Warn`
level even when `failed == 0` (e.g., a clean `Base.wz` archive would log `"0 of 3
images failed to serialize"` at Warn). The brief does not specify a log level or gate
the summary on `failed > 0`, so this is not a brief violation, but it means every
successful archive ingest now emits one Warn-level line that isn't actually a warning.
Worth a follow-up, not blocking.

## Exported signature stability

**PASS.** `git diff 8c86277..8e1f7201f -- .../runtime.go` is empty — confirmed zero
changes to the only production caller. `grep '^func '` on `adapter.go` shows
`SerializeToDirectory(l logrus.FieldLogger, f *wz.File, outputDir string) error` and
`SerializeImage(img *wz.Image, outputPath string) error` unchanged; only the
unexported `serializeDirectory` gained the `archiveName string` parameter and two
`int` return values, exactly as the brief specified.

## Test quality (TDD honesty)

**PASS**, independently reproduced (see above) rather than taken from the report.
Both new test files build synthetic archives via `wztest.Builder` and locate exact
byte offsets to corrupt using key-length-independent arithmetic, with a sanity
assertion (`data[offset] == expected`) before corrupting — a real defense against a
future `wztest` encoding change silently breaking the fixture rather than the test.

## Repo conventions

- No literal home/absolute paths in any committed file (`git grep` for `/home/`,
  `/mnt/`, `tmp/83` across the four changed files: no hits).
- Builder pattern used for test setup (`wztest.NewBuilder()...AddDir/AddImage`); no
  `*_testhelpers.go` file added.
- No new domain type/alias/numeric constant introduced; nothing to check against
  `libs/atlas-constants/`.
- Line endings: diff is pure additions/one-line replacement; nothing suggests a
  CRLF→LF normalization.

## Not evaluable

- Did not run `go test -race ./...` repo-wide or `tools/verify.sh` — out of scope per
  instructions (a separate gate is already running it), and the brief itself defers
  `-race` to that gate.

(The earlier "could not re-run the report's archive table" gap was **closed** during
this review — a real v83 archive set was located and all 17 archives were swept. See
Independent verification above.)

## Concurrent working-tree change (not part of the reviewed commit)

While this review was running, another agent modified
`services/atlas-data/atlas.com/data/data/wztoxml/adapter.go` in the working tree
(uncommitted, +5/-1), gating the summary line on `failed > 0` and dropping the clean
case to `Infof` — i.e. acting on non-blocking finding #2 above. This is **not** part
of commit `8e1f7201f` and is not covered by this review's verdict. I confirmed it
still builds and that `atlas-data/data/wztoxml` tests pass with it applied
(`go build ./... && go test ./data/wztoxml/...` → `ok`), and that the existing
`TestSerializeDirectoryCountsFailures` remains valid under it (its failure case has
`failed == 1 > 0`, so the asserted Warn-level summary line is still emitted). It
should be reviewed with whatever commit ultimately carries it.

Note that this change, as written, leaves the clean-archive summary line at `Infof`
with the same "N of M images failed to serialize" wording — reading "0 of 3 images
failed to serialize" at Info level is harmless but still slightly odd phrasing for a
success path. Cosmetic only.

## Working-tree hygiene

All files I reverted for RED verification were restored byte-exactly:
`git diff --stat -- libs/atlas-wz/` is empty, and `adapter.go`'s only remaining
delta from HEAD is the foreign hunk described above. I introduced no lasting change
to any source file; the three temporary tests I wrote
(`wz/zzz_openspot_test.go`, `wz/zzz_sweep_test.go`, `wz/zzz_list_test.go`) were each
deleted immediately after use and none were ever committed.

## Verdict rationale

No blocking findings. The core FR-8 change is correct, narrowly scoped, and verified
independently (not just re-stated from the report) at every point the brief called out
as risk: blast radius, error naming/wrapping, exported-signature stability, and NFR
compliance (summary-line presence + per-property-line absence). The one gap (brief's
blast-radius survey omitting `atlas-renders/storage/wzcache.go`) does not change the
correctness conclusion since that call site is also safe — it is reported as a
non-blocking accuracy note on the brief's own claim, not a code defect.
