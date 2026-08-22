# Review: Task 5 — MAPLELIFE_ERROR clientbound codec

Range: `100e06719..a79ead298` (single commit `a79ead298`, amended per the
controller addendum's commit-then-regenerate-then-amend sequence).

Brief: `.superpowers/sdd/plan/task-5-brief.md` (body + controller addendum).
Report: `.superpowers/sdd/plan/task-5-report.md`.

## Scope confirmed

`git diff --stat 100e06719..a79ead298` touches exactly: the new codec +
test (`libs/atlas-packet/maplelife/clientbound/error.go`,
`error_test.go`), four evidence YAMLs (v83/v87/v92/v95, no v84), four
ida-export splices (v83/v87/v92/v95, no v84), four audit report pairs
(`docs/packets/audits/<version>/MaplelifeMapleLifeError.{json,md}`), the
`tools/packet-audit/cmd/run.go` `candidatesFromFName` case, and
`docs/packets/audits/{STATUS.md,status.json}`. No files outside this set
changed. This matches the brief plus the controller addendum's four
required additions (evidence, marker lines, `run.go` case, matrix
regeneration). `derivation.md` is untouched, as required (brief lists it
read-only).

Worktree was already clean of any source/doc-under-review conflicts at
review start (only unrelated untracked review artifacts from sibling
tasks were present), so no detached checkout was needed.

## Findings

### PASS — gms_v84 correctly excluded

No `gms_v84` evidence YAML, no `gms_v84` marker line, no
`template_gms_84_1.json` reference anywhere in the diff. `error_test.go:17-19`
documents the exclusion explicitly. `status.json`'s `gms_v84` cell for
`MAPLELIFE_ERROR` is untouched by the diff (confirmed via
`git diff` hunk boundaries — the `"gms_v84": {` block is pure context, no
`+`/`-` lines inside it) and still reads `state: incomplete, note: "no audit
report", opcode: 350` — identical shape/value to the sibling
`MAPLELIFE_RESULT` row's `gms_v84` cell (`docs/packets/audits/status.json`,
verified by direct read of both rows). This is the same precedent Task 4
established, not a new inconsistency.

### PASS — export splices are pure additions, shape matches twins

`git diff --no-color` on all four `docs/packets/ida-exports/gms_{v83,v87,v92,v95}.json`
shows `+14 -0` each — zero deletions, one new top-level key
(`CUICharacterSaleDlg::OnCreateNewCharacterResult`) inserted per file. No
`note` field or any other pre-existing entry was touched, refuting any
regression from the CLI's known `-splice` bug. Diffed the new entry's shape
against the already-landed Task 4 twin (`OnCheckDuplicatedIDResult`) in the
same files: both carry `{address, direction, calls:[{op, comment}, ...]}`.
The new entry's `calls` is `[{"op":"Decode1"}, {"op":"Decode4"}]`, matching
derivation.md §5's decode order (`Decode1 nType; Decode4 nParam`) field by
field and width by width — `Decode1` = 1 byte, `Decode4` = 4 bytes, which is
exactly what `error.go`'s `Encode`/`Decode` implement (`WriteByte`/`ReadByte`
then `WriteInt`(uint32)/`ReadUint32`).

### PASS — `qualifiedWriterName` consistency (the Task-4 failure mode)

`qualifiedWriterName("maplelife", "MapleLifeError")` = `"Maplelife" +
"MapleLifeError"` = `MaplelifeMapleLifeError` (confirmed by reading
`tools/packet-audit/cmd/run.go:223-228` directly, not by re-deriving from
memory). This exact string appears consistently in:
- the `packet-audit:verify` marker lines (`error_test.go:28-31`)
- each evidence YAML's `packet:` field (all four, e.g.
  `docs/packets/evidence/gms_v83/maplelife.clientbound.MapleLifeError.yaml:1`)
- the audit report directory/file names
  (`docs/packets/audits/gms_v83/MaplelifeMapleLifeError.{json,md}`) and each
  report's internal `"WriterName"` field
- the `status.json`/`STATUS.md` row's new `packet` column

No mismatch found anywhere in the four places this string must agree.

### PASS — version-gate idiom / field-shape claim

`error.go` has no `MajorAtLeast` gate at all (`Encode`/`Decode` at
`error.go:113-127` are unconditional). This matches derivation.md §5.5's
explicit finding: "No `MajorAtLeast` gate is needed for field shape
(identical on all four present versions)." `TestMapleLifeErrorNoVersionDivergence`
(`error_test.go:152-161`) pins byte-identical encoding across all four
in-scope versions, and `TestMapleLifeErrorByteFixture` exercises all four
versions with the same wire bytes. No raw `> N` comparison anywhere in the
diff.

### PASS — arm enumeration and literal-vs-config separation

The three exported constants (`error.go:51,55,60`) exactly match
derivation.md §5.5's closed three-arm enumeration
(`SUCCESS`/`NAME_TAKEN_AT_SUBMIT`/`UNKNOWN_ERROR`), including the success
arm per design §5.4 — the doc comment at `error.go:45-47` explicitly calls
out that SUCCESS ships through this op, not `MAPLELIFE_RESULT`.
`TestMapleLifeErrorArmsAreExhaustive` (`error_test.go:228-252`) pins the set
literally both ways (missing/extra). No `nType` literal appears as a Go
constant anywhere in `error.go` — the only literals are in tests
(`mleOptions()`, byte fixtures), consistent with DOM-25 (per-version literal
lives in tenant-template config, resolved via
`atlas_packet.WithResolvedCode`).

### PASS — `toolSha` / `status.json` / `STATUS.md` diff scope (Task-4's residual risk)

Independently reran `go run ./tools/packet-audit matrix --check` at
`a79ead298` (repo root cwd): exit 0, only two informational `n-a evidence
consumed` notes unrelated to this task. `toolSha` in the committed
`status.json`/`STATUS.md` is `46d5ca74…` — matches the report's claimed
final hash, is neither `sha256("")` nor Task 4's stale leftover value.

Diffed `status.json` and `STATUS.md` across the full range by hand:
- `toolSha` line: changed (expected).
- `exportHashes`: only `gms_v83`, `gms_v87`, `gms_v92`, `gms_v95` changed
  (expected — those four files were spliced); `gms_v84`, `gms_v48/61/72/79`,
  `jms_v185` unchanged.
- `rows`: the only `op` entry touched is `MAPLELIFE_ERROR` — gained a
  `packet` field and four cells (`gms_v83/87/92/95`) flipped
  `incomplete`→`verified`; `gms_v84` cell inside the same row is untouched.
  No other row in the (very large) `rows` array appears in the diff.
- `STATUS.md`: same row, plus the version-summary totals table rows for
  v83/v87/v92/v95 (recomputed percentages — expected side effect of more
  cells promoting), plus the header tool-hash/export-hash lines.

No unrelated op flipped `state`, no cell silently degraded. This closes out
the specific residual risk flagged from Task 4's review.

Also independently reran `operations --check` (exit 0, "0 absent-writer
note(s)"), `fname-doc --check` (exit 0, "268 structs without an audit
report carry no fname" — matches the report's number), and
`dispatcher-lint` (exit 0, "clean").

### PASS — codec correctness and test honesty

`go build ./... && go test ./maplelife/... -run TestMapleLifeError -v` at
`a79ead298`: all subtests pass, including all four
`TestMapleLifeErrorByteFixture` version×arm cases and
`TestMapleLifeErrorCodesAreConfigResolved`'s empty-options-sentinel and
remapped-byte assertions (proves the wire byte is config-resolved, not a Go
constant — this sub-case would fail without `WithResolvedCode` wired
correctly). `Encode`/`Decode` are exact mirrors (`WriteByte`/`ReadByte`,
`WriteInt`(uint32 LE)/`ReadUint32`), matching `response.Writer`/
`request.Reader`'s actual signatures (checked directly, not assumed).
Since `error.go`/`error_test.go` are wholly new files, "does it fail before
the change" is trivially true (undefined symbols); the report's own
Step-2 log is consistent with this.

### PASS — `candidatesFromFName` case placement

`tools/packet-audit/cmd/run.go:2642-2643` — new `case
"CUICharacterSaleDlg::OnCreateNewCharacterResult":` inserted between the
existing `OnCheckDuplicatedIDResult` (Task 4) and
`CCashShop::OnCheckTransferWorldPossibleResult` cases, syntactically valid,
no collision with the unrelated pre-existing `CLogin::OnCreateNewCharacterResult`
case at line 781.

### Non-blocking observations

1. **`derivation.md` §5 still literally reads `decompile_sha256: **PENDING**`**
   (`derivation.md:693-696`) even though the implementer's report states the
   hashes were re-harvested and now live in the evidence YAMLs. This is not a
   Task 5 defect — the brief explicitly marks `derivation.md` read-only for
   this task, and the report is transparent about the discrepancy between
   the addendum's claim ("already re-harvested") and what it actually found
   (still PENDING, so it did the harvest itself). But it does leave §5 as a
   stale/misleading source of truth for any later task or reviewer that
   trusts derivation.md's own text over the evidence YAMLs. Worth a
   follow-up note update to §5 whenever `derivation.md` is next legitimately
   in scope.
2. The four evidence YAMLs' `verifies:` lists (4 entries each) don't include
   `TestMapleLifeErrorNoVersionDivergence` or `TestMapleLifeErrorOperation`
   — cosmetic incompleteness relative to what the codec actually tests; the
   sibling `MapleLifeResult` evidence file has a similar gap (its own
   `TestMapleLifeResultReasonMapping` is listed but not a divergence test).
   Not a defect, just inconsistent with the fuller `CashCheckNameChange`
   evidence file's `verifies:` list, which enumerates every relevant test.

## Not evaluable

- The IDA-side function renames (`sub_7D77B0`→mangled symbol on v83,
  `sub_82E252`→mangled symbol on v87) live in the IDA databases, not the
  git repository, and are not part of this diff's review surface — took the
  report's description at face value since there is no repo artifact to
  check it against.
- Consumers of `MapleLifeErrorWriter`/`MapleLifeErrorBody` (Tasks 7, 13, 14)
  do not exist yet in this range — cross-service-seam tracing is not
  applicable here; this is a leaf codec with no consumer in scope yet.

## Verdict rationale

Every requirement in the brief body and the controller addendum was traced
to concrete evidence: four-only version scope, exact `qualifiedWriterName`
agreement across all four required locations, `MajorAtLeast`-idiom
compliance (here: correctly absent, per §5.5's no-divergence finding), the
`status.json`/`STATUS.md` diff scoped to exactly this codec's rows plus
`toolSha`, and all four DoD gates (`matrix --check`, `operations --check`,
`fname-doc --check`, plus `dispatcher-lint`) independently re-run and
confirmed exit 0 at `a79ead298`. No blocking defect found.
