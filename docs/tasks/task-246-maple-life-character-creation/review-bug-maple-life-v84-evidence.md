# Review: gms_v84 Maple Life evidence promotion (`c5b92a496..0dd5365b3`)

Branch: `task-246-bug-maple-life-v84-registration`
Range: `ee20210c1`, `b9ef11774`, `2fef11a98` (three `packet-verifier` cell
promotions) + `0dd5365b3` (bookkeeping).
Requirement: `docs/tasks/task-246-maple-life-character-creation/bug-maple-life-v84-registration.md`,
"Positive derivation" / "Fix" / "Still owed" sections.

## Verdict

APPROVED_WITH_FINDINGS — 0 blocking, 1 non-blocking.

## Scope confirmed

Reviewed exactly the four commits named in the brief. `git diff --stat
c5b92a496..0dd5365b3` touches only: `docs/packets/audits/{STATUS.md,status.json,
gms_v84/Maplelife*.{json,md}}`, `docs/packets/evidence/gms_v84/maplelife.*.yaml`,
`docs/packets/ida-exports/gms_v84.json`, the three `maplelife/**/*_test.go`
files, the task's `agent-ledger.tsv`, and the bug file itself. No codec source
file (`check_name.go`, `result.go`, `error.go`), no seed-data template, and no
`docs/packets/registry/gms_v84.yaml` changed in this range — consistent with
the brief's premise that the template/registry wiring landed earlier
(`7f1dc0a43`) and is out of scope here. No mismatch between the requested
range and the work found.

## Findings

### 1. No wire change to an already-verified version — PASS

`git show <sha> --stat` for all three commits confirms the only Go files
touched are the three `_test.go` files (comment + fixture-table edits), never
`serverbound/check_name.go`, `clientbound/result.go`, or
`clientbound/error.go`. Diffstats:

- `ee20210c1`: `serverbound_test.go | 27 +++++++++++++---------` (16
  insertions, 11 deletions) — comment rewrite + one new row in
  `mlcnFixtureVersions` + one new `inScope` entry + one new
  `packet-audit:verify` marker. No fixture *byte* table changed for v83/v87/
  v92/v95 (`mapleLifeCheckNameGoldenBody` untouched).
- `b9ef11774` / `2fef11a98`: same shape in `result_test.go` /
  `error_test.go`.

`go test ./maplelife/...` (run from `libs/atlas-packet`) passes:
`ok github.com/Chronicle20/atlas/libs/atlas-packet/maplelife/clientbound`,
`ok .../serverbound`. No existing version's assertions regressed.

### 2. ida-exports splices are additive across all three commits — PASS

Each commit's diff against `docs/packets/ida-exports/gms_v84.json` is a pure
addition, confirmed via `git show <sha> -- docs/packets/ida-exports/gms_v84.json`:

- `ee20210c1`: `@@ -29726,6 +29726,16 @@`, `+10 -0` (adds
  `CUICharacterSaleDlg::SendCheckDuplicateIDPacket`)
- `b9ef11774`: `@@ -29736,6 +29736,21 @@`, `+15 -0` (adds
  `CUICharacterSaleDlg::OnCheckDuplicatedIDResult`)
- `2fef11a98`: `@@ -29751,6 +29751,21 @@`, `+15 -0` (adds
  `CUICharacterSaleDlg::OnCreateNewCharacterResult`)

The three hunk start lines are contiguous and monotonically increasing
(29726 → 29736 → 29751), each landing immediately after the previous commit's
insertion point, and each is `-0` deletions. No existing key was modified or
dropped across the combined range. Independently confirmed:
`sha256sum docs/packets/ida-exports/gms_v84.json` in the worktree =
`636dad10e1c43a2c7357cb7313ee3d3a6c740e3ca2d4274d2b06b0ce9f73a228`, matching
the hash recorded in both `STATUS.md` and `status.json` after the range —
the export file on disk is exactly what the bookkeeping claims was hashed.

### 3. Fixture bytes match the documented read order — PASS

- `docs/packets/audits/gms_v84/MaplelifeCheckName.json`: one row, `AtlasOp: 4`
  (string), comment "sCharName; MAPLELIFE_CHECK_NAME opcode 263 (0x107)" —
  matches the bug file's sender = `COutPacket(263)` + one `EncodeStr`.
- `docs/packets/audits/gms_v84/MaplelifeMapleLifeResult.json`: row 0
  `AtlasOp: 4` (string, "sName"), row 1 `AtlasOp: 0` (Decode1, "nResult") —
  matches `DecodeStr(name)` then `Decode1(result)`.
- `docs/packets/audits/gms_v84/MaplelifeMapleLifeError.json`: row 0
  `AtlasOp: 0` (Decode1, "nType"), row 1 `AtlasOp: 2` (Decode4, "nParam") —
  matches `Decode1(nType)` then `Decode4(nParam)`.

All three reports show `"Verdict": 0` (pass) and `"FlatInvalid": false`. The
new `packet-audit:verify` markers and fixture-table rows added to the
`_test.go` files reuse the same addresses (`0x7fd86a`, `0x7fd949`, `0x7fda6f`)
cited in the bug file's "Positive derivation" and in the corresponding
`docs/packets/evidence/gms_v84/maplelife.*.yaml` `ida.function`/`ida.address`
fields — internally consistent across all four artifact classes (test
comment, evidence yaml, audit json, audit md). I cannot independently verify
the underlying IDA decompile (session `46c2a2eb` is not in the repo); this is
expected per the brief and is noted under "Not evaluable."

### 4. Evidence / matrix / STATUS agreement — PASS

- The three evidence yamls (`docs/packets/evidence/gms_v84/maplelife.*.yaml`)
  each carry a 64-hex-char `decompile_sha256`, matching the format used by
  existing evidence files elsewhere in the tree (spot-checked against
  `docs/packets/evidence/gms_v83/*.yaml`), and each `verifies:` list points
  at test names that exist in the corresponding `_test.go` file.
- `STATUS.md` diff shows exactly the three target cells flipping in the v84
  column (`0x107`, `0x167`, `0x168`, all → `✅`) plus the export-hash line and
  the v84 summary-row count (`451,0,0,326,250,0,58.0%` → `454,0,0,326,247,0,
  58.2%`: +3 total, -3 ❌, unchanged ⬜/n-a) — arithmetically consistent with
  three cells moving from unverified/failed to verified, nothing else.
- `status.json` diff (squashed across the range) shows only the three
  matching `state`/`opcode` pairs changing (`n-a`/-1 → `verified`/263, 359,
  360) plus the export-hash field. No other packet's status entry changed.
- `go run ./tools/packet-audit matrix --check` (run live in the worktree)
  exits 0, emitting only the two pre-existing `n-a evidence consumed` notes
  (`CASHSHOP_CASH_ITEM_GACHAPON_RESULT × gms_v79`,
  `USE_TELEPORT_ROCK × gms_v48`) — unrelated to this range, confirming the
  matrix is internally consistent post-promotion.
- The bookkeeping commit `0dd5365b3`'s edit to the bug file's "Still owed"
  section accurately restates the three commits, their opcodes, and the
  before/after states; `derivation.md` is untouched in the full range (empty
  diff), matching the brief's constraint against renumbering/rewriting it.

## Non-blocking

1. **Stale "four" vs corrected "five" comment left in `ee20210c1`.**
   `libs/atlas-packet/maplelife/serverbound/serverbound_test.go:82` still
   reads "deliberately the same bytes for all four" even though the same
   commit renamed the section header to "FIVE in-scope cells" three lines
   above (`serverbound_test.go:18`) and added the v84 row to
   `mlcnFixtureVersions`. The sibling commits (`b9ef11774`, `2fef11a98`)
   correctly updated the equivalent line in `result_test.go`/`error_test.go`
   to "all five." Cosmetic only — the test loop already iterates five
   versions regardless of the comment — but it is the same class of
   drift the bug file's own reviewer flagged and closed for the earlier
   pass (stale VERSION-ABSENT prose), so it's worth a follow-up sweep rather
   than leaving it to compound.

## Not evaluable

- The underlying IDA decompiles at `0x7fd86a`, `0x7fd949`, `0x7fda6f` in
  session `46c2a2eb` cannot be checked from repo state; only internal
  consistency across the test comments, evidence yaml, and audit
  json/md was verified, as the brief anticipates.
