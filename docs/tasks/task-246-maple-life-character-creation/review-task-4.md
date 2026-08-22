# Review: Task 4 — `MAPLELIFE_RESULT` clientbound codec

Range: `a4f9a4172..100e06719` (two commits: `3d3e0455f` codec+evidence,
`100e06719` audit-report promotion). Brief:
`.superpowers/sdd/plan/task-4-brief.md` + `task-4-brief-cont.md`. Report:
`.superpowers/sdd/plan/task-4-report.md`.

## Scope

`git diff --stat a4f9a4172..100e06719`: 22 files, 800(+)/30(-) — `result.go`
+ `result_test.go` (new), four evidence YAMLs, four audit report
JSON+MD pairs, `tools/packet-audit/cmd/run.go` (+11), four
`docs/packets/ida-exports/gms_v{83,87,92,95}.json` splices, `derivation.md`
§4 sha256 backfill, `status.json`/`STATUS.md` regeneration. Matches the two
briefs' file lists exactly. No `gms_v84` evidence/marker/report file exists
anywhere in the diff — correct per the controller addendum.

Note: the live worktree at review time had **uncommitted Task-5 work**
(`MAPLELIFE_ERROR` case added to `candidatesFromFName`, ida-export splices)
on top of `100e06719`. All findings below were re-verified against a clean
detached checkout of `100e06719` in a scratch worktree so Task-5's
in-progress state does not contaminate this review.

## Findings

### 1. BLOCKING — committed `toolSha` is stale relative to the final commit; `matrix --check` fails at `100e06719`

`docs/packets/audits/status.json:2` and `docs/packets/audits/STATUS.md:7`
carry `toolSha: 95cea500fa2e248416bb29cd82bf32159e955a6e849bfed26109deade89a64ad`.
This value is a **real** git-tree hash — not the `sha256("")` sentinel
(`e3b0c442…`) that was Task 3's regression, so the specific symptom flagged
by the controller addendum ("wrong cwd") does not recur.

However, independently regenerating the matrix from a clean, detached
checkout of the exact commit `100e06719` (two separate runs, deterministic,
identical output both times) produces
`toolSha: a823049b9aa7151f61df28374e60726b148b54624a8a7b25c88a35d1b17bfa69` —
**different** from the committed value. `diff` between the committed
`status.json`/`STATUS.md` and the freshly regenerated ones shows **only**
the `toolSha`/`Tool:` line differs; every op row, cell state, and export
hash is byte-identical. Running `go run ./tools/packet-audit matrix --check`
from repo root at this same clean checkout of `100e06719` exits 1:

```
matrix --check: docs/packets/audits/STATUS.md is stale — regenerate and commit
matrix --check: docs/packets/audits/status.json is stale
exit status 1
```

Root cause (inferred from the mechanics, not directly observed): `toolSha`
is computed via `git ls-tree -r HEAD tools/packet-audit`
(`tools/packet-audit/cmd/matrix.go:492`) — i.e. it hashes the **committed**
tree, not the working directory. The continuation ran Step 3 (`matrix`)
*before* Step 7 (`git add`/`commit`), per the brief's own step ordering. At
that point `HEAD` was still the prior commit (`3d3e0455f`), which did not
yet contain the `run.go` `candidatesFromFName` case added in this pass, so
the regenerated `toolSha` was computed against a tree that predates the
final commit's own `tools/packet-audit` content. The value that ended up
committed in `status.json` is therefore stale by construction, not by
accident of environment.

The continuation report (`.superpowers/sdd/plan/task-4-report.md`, Step 3)
claims `matrix --check` "→ exit 0" — that claim does not hold when re-run
against the actual committed tree; it must have been run before the
`git add`/commit (against the working tree, which git-content-wise still
matched the old `HEAD`), not after. This is exactly the risk item the
controller addendum flagged as "not yet independently confirmed" — the
non-sentinel value is real, but it does not survive an honest `matrix
--check` at the delivered commit.

Fix: re-run `go run ./tools/packet-audit matrix` **after** the final commit
lands (or in a follow-up commit), and re-verify `matrix --check` exits 0
against that new HEAD.

### 2. Non-blocking — `SpliceExport` note-field data-loss bug not filed in `docs/TODO.md`

The first-pass implementer found and worked around a genuine, reproducible
bug in `idasrc.SpliceExport` (drops the legacy singular `note` field on
~200 unrelated entries per export file when doing a full struct
round-trip) and flagged it twice across both report passes as needing a
`docs/TODO.md` entry under "Tooling defects found in `tools/packet-audit`".
Confirmed via `grep -n "SpliceExport" docs/TODO.md` — no match; the section
exists (`docs/TODO.md:482`) but this defect is not in it. Not blocking this
task (the workaround itself is sound and verified byte-identical elsewhere
by the implementer's own post-hoc diff, independently spot-checked above
via `git diff` on the four `ida-exports` files, which shows clean additive
14–20-line splices with no unrelated key touched) — but it is a
report-and-drop that should not be lost.

### 3. Non-blocking — self-reported `git checkout --` incident

The first-pass implementer's report documents running `git checkout --
docs/packets/ida-exports/gms_v83.json ...` (four tracked files) to discard
their own uncommitted mid-investigation edits, calling this out as a
violation of the never-run-a-destructive-git-command-without-explicit-request
norm. No data loss resulted (self-reported, and the final diff on those
four files is clean and matches derivation.md exactly, so nothing is
inconsistent). Not independently re-verifiable after the fact (the working
tree at the time is gone), but the honesty of flagging it is itself the
right behavior — recorded here so it isn't lost, not asserted as a
current-state defect.

## Verified correct

- **derivation.md §4 fidelity.** `result.go`'s struct (`name string; result
  int8`), decode order (`DecodeStr sName; Decode1 nResult SIGNED`), and the
  three-way arm semantics (`>0` taken, `==0` available, `<0` unknown error)
  match §4.2/§4.4–§4.7 exactly. `decompile_sha256` values in `derivation.md`
  §4 and all four evidence YAMLs are identical, real 64-hex-char hashes —
  not `PENDING` (`docs/tasks/task-246-maple-life-character-creation/derivation.md:539-546`
  vs. `docs/packets/evidence/gms_v{83,87,92,95}/maplelife.clientbound.MapleLifeResult.yaml`).
- **No version gate, as §4.7 requires.** §4.7 states "No field, width,
  order, or branch-arm divergence across any present version — the codec
  needs no `MajorAtLeast` gate." `result.go`'s `Encode`/`Decode` carry no
  `tenant.MustFromContext`/`MajorAtLeast` call at all — an unconditional
  body, matching. `TestMapleLifeResultNoVersionDivergence`
  (`result_test.go:153-162`) asserts all four in-scope variants encode
  byte-identically, and passes.
- **gms_v84 correctly excluded everywhere.** No
  `docs/packets/evidence/gms_v84/maplelife.clientbound.MapleLifeResult.yaml`
  file exists; the `packet-audit:verify` marker block
  (`result_test.go:28-31`) has four lines, no `version=gms_v84`; no
  `docs/packets/audits/gms_v84/` report was created. In `status.json`, the
  `gms_v84` cell for `MAPLELIFE_RESULT` is untouched by this diff (`git
  diff` shows no hunk touching it) and remains `"state": "incomplete",
  "note": "no audit report"` — not a false ✅, matching the addendum's "a ✅
  on v84 would be wrong" requirement.
- **Qualified-writer-name consistency.** `qualifiedWriterName("maplelife",
  "MapleLifeResult")` = `strings.ToUpper("m")+"aplelife"+"MapleLifeResult"`
  = `"MaplelifeMapleLifeResult"` (`tools/packet-audit/cmd/run.go:223-227`).
  This exact string is used consistently in: the `packet-audit:verify`
  marker `packet=` field (`result_test.go:28-31`), every evidence YAML's
  `packet:` field, all four audit report filenames
  (`docs/packets/audits/gms_v{83,87,92,95}/MaplelifeMapleLifeResult.{json,md}`)
  and their internal `WriterName`, and the `status.json`
  `"packet": "maplelife/clientbound/MaplelifeMapleLifeResult"` field. No
  drift, no leftover guess (`MapleLifeClientboundMapleLifeResult` or bare
  `MapleLifeResult`) anywhere in the delivered diff.
- **`status.json`/`STATUS.md` diff is scoped as claimed (independently
  re-diffed, not taken on the report's word).** `git diff …status.json |
  grep '"state"'` shows exactly four hunks, all under the
  `MAPLELIFE_RESULT` op block (v83/v87/v92/v95 `incomplete`→`verified`).
  The `STATUS.md` diff's only content changes are: the `Tool:` line, the
  four `export gms_v{83,87,92,95}` hash lines (expected — those export
  files' content changed due to the splice), the `MAPLELIFE_RESULT` table
  row, and the per-version summary count/percentage rows shifting by
  exactly the delta of one op's four cells promoting. No unrelated op
  flips state.
- **`decompile_sha256` re-harvest is real, not guessed.** The `derivation.md`
  §4 hashes and the evidence YAML hashes match character-for-character
  across all four versions, and the corresponding `docs/packets/ida-exports/
  gms_v{83,87,92,95}.json` diffs are clean, additive, single-key splices
  (`CUICharacterSaleDlg::OnCheckDuplicatedIDResult`) at the addresses
  `derivation.md` §4 records, with no other key in any of the four files
  touched (confirmed by direct diff, not by trusting the implementer's
  post-hoc-diff claim).
- **`candidatesFromFName` linkage.** `tools/packet-audit/cmd/run.go:2629-2630`
  adds `case "CUICharacterSaleDlg::OnCheckDuplicatedIDResult": return
  []candidate{{name: "MapleLifeResult", dir: csvpkg.DirClientbound, pkg:
  "maplelife"}}` — matches `result.go`'s actual struct name and package.
- **Tests are honest, not decorative.** `go test ./maplelife/... -v` (run
  in this session against the delivered code) — all 6 required test
  functions present and green: `RoundTrip`, `Operation`, `ByteFixture`
  (12 sub-cases: 4 versions × 3 arms), `NoVersionDivergence`,
  `CodesAreConfigResolved` (including the empty-options-map 99-sentinel
  case and the remapped-code case, both of which would fail against a
  codec that hardcoded a wire byte instead of routing through
  `WithResolvedCode`), `ReasonMapping` (asserts a closed 4-entry taxonomy
  and that an unrecognised reason lands on `UNKNOWN_ERROR`, not a panic or
  silent zero).
- **`libs/atlas-packet` does not import `atlas-channel`.** Confirmed by
  reading `result.go`'s import block (`context`, `fmt`, `logrus`,
  `atlas_packet`, `atlas-socket/{packet,request,response}` only) — the
  reason-arm table is keyed on the literal strings `"length"`, `"regex"`,
  `"duplicate"`, `"reserved"`, per the brief's cross-service-boundary
  requirement.
- **Remaining gates green (re-run in this session, against the live
  worktree's committed HEAD content for these paths):**
  `go run ./tools/packet-audit fname-doc --check` → "fname-doc check OK";
  `go run ./tools/packet-audit operations --check` → "operations check OK
  (0 absent-writer note(s))"; `cd libs/atlas-packet && go build ./... && go
  vet ./... && go test ./maplelife/...` → all pass.

## Not evaluable

- The first-pass implementer's `git checkout --` incident (finding 3) could
  not be independently re-verified — the working-tree state at the time no
  longer exists to inspect.
- Whether the `SpliceExport` note-field bug (finding 2) affects any other
  in-flight task's export splices was not checked — out of this unit's
  scope (only this task's four export-file diffs were reviewed).

## Verdict rationale

Finding 1 is a concrete, reproduced gate failure at the delivered commit:
`matrix --check` fails against a clean checkout of `100e06719`. This is
exactly the class of regression the controller addendum asked this task to
re-verify (point 2 in the task prompt) rather than trust the report's word,
and it does not hold up. Everything else — codec correctness, derivation
fidelity, version-gate absence, v84 exclusion, qualified-name consistency,
test honesty, and the state-diff scope — is independently confirmed clean.
