# Review: Task 3 — Packet coverage matrix, re-verify MonsterDestroy (task-253)

Commit range: `81c719bec..cf248f49a`
Single commit: `cf248f49a docs(packets): re-verify MonsterDestroy cells behind the v92 swallow gate`

## Scope confirmed

The diff matches the task-3 brief's file inventory exactly, plus the two
controller-authorized marker-only edits to `destroy_test.go` and
`v61_test.go`. 14 files changed, 43 insertions / 14 deletions — small, fully
readable diff (`git diff --stat 81c719bec..cf248f49a`, `.superpowers/sdd/plan/review-81c719bec..cf248f49a.diff`).

Files touched:
- `docs/packets/audits/STATUS.md`, `docs/packets/audits/status.json` (regenerated)
- `docs/packets/evidence/{gms_v61,gms_v72,gms_v79,gms_v83,gms_v84,gms_v87,gms_v92,gms_v95,jms_v185}/monster.clientbound.MonsterDestroy.yaml`
- `docs/packets/gates.yaml`
- `libs/atlas-packet/monster/clientbound/destroy_test.go`
- `libs/atlas-packet/monster/clientbound/v61_test.go`

No file outside this set was touched (`git diff --stat` above is exhaustive).

## Spec compliance against the brief (+ controller amendment)

### Step 1 — Pin v92 evidence: PASS
`docs/packets/evidence/gms_v92/monster.clientbound.MonsterDestroy.yaml` created
with `function: CMobPool::OnMobLeaveField`, `address: "0x64bb90"`. Verified
against `docs/packets/ida-exports/gms_v92.json:6675` — the
`"CMobPool::OnMobLeaveField"` entry's `"address"` field is `"0x64bb90"`
(confirmed by direct read, not paraphrase). Matches the brief's expected value
exactly. `category: TIER1-FIXTURE` as specified.

### Step 2 — `verifies:` on all nine YAMLs: PASS
All nine evidence files (`gms_v61, gms_v72, gms_v79, gms_v83, gms_v84, gms_v87,
gms_v92, gms_v95, jms_v185`) now carry both:
- `destroy_test.go#TestMonsterDestroySwallowGate`
- `destroy_test.go#TestMonsterDestroyNonSwallowTypesAreFiveBytes`

Indentation preserved per-file's existing style: `gms_v72` and `gms_v79`
already used 2-space indent under `verifies:` for their prior single entry —
the new lines were appended at 2-space to match (diff hunks confirm). All
other files (`gms_v61, gms_v83, gms_v84, gms_v87, gms_v92, gms_v95, jms_v185`)
use 4-space, matching the existing `ida:` block style in each. `gms_v61`'s
pre-existing `verifies:` entry (`v61_test.go#TestMonsterDestroyBytesV61`) is
preserved, not overwritten.

### Step 3 — Register the gate: PASS
`docs/packets/gates.yaml:117-123` — new
`# ---- boundary v87 / v92 ------------------------------------------------`
block inserted in correct version order, between the `>=87` boundary group and
the pre-existing `# ---- boundary v87 / v95 ----` group. Text is verbatim
against the brief (packet, direction, field, boundary, lower/upper keys all
match).

### Step 4 — Regenerate the matrix: PASS
Independently re-ran `go run ./tools/packet-audit matrix --check` in this
review — exits 0. `docs/packets/audits/status.json`'s `KILL_MONSTER` row,
`gms_v92` cell is `{"state": "verified", "opcode": 277}` — the `"partial"`
state and its `"note"` field are gone, replaced cleanly (diff hunk at
`status.json:25433` shows only that cell's state/note lines change; no other
cell in the row touched). `STATUS.md`'s corresponding row cell moved from ❌ to
✅ (diff at `STATUS.md:264`) and the v92 rollup row changed from
`52/0/148/679/141 5.9%` to `53/0/147/679/141 6.0%` — consistent with exactly
one additional verified cell. Both files show only the tool's own generated
diff shape (table rows, rollup numbers) — no evidence of hand-editing.

### Step 5 — Remaining `--check` passes: PASS (independently re-run)
- `fname-doc --check` → exit 0, `"fname-doc check OK (268 structs without an
  audit report carry no fname)"`
- `operations --check` → exit 0, `"operations check OK (0 absent-writer
  note(s))"`
- `dispatcher-lint` → exit 0, `"dispatcher-lint: clean"`
- `gate-lint --check` was not re-run/required — pre-existing 38 raw-comparison
  failures in unrelated packages are out of scope per the controller's prior
  ruling; confirmed none of those files are in this diff.

### Step 6 — Commit: PASS
Exactly the Task 3 file set staged (confirmed by `git diff --stat` above);
commit `cf248f49a` exists on the branch with the expected message.

## Controller-amended requirement: marker deletion, verified line-by-line

Requirement: delete only the five duplicate marker lines on
`destroy_test.go` (5 lines, one per version, above `TestMonsterDestroy`) plus
the 1-line markers above `TestMonsterDestroyBytesV79` and
`TestMonsterDestroyBytesV72`, and the 1-line marker above
`TestMonsterDestroyBytesV61` in `v61_test.go` — test function bodies and all
other packets' markers must remain untouched.

Verified by direct read of both files post-commit:

- `destroy_test.go` diff (`.superpowers/…/review-…diff`): removes exactly 5
  marker lines above `TestMonsterDestroy` (gms_v83/v87/v95/jms_v185/v84) and 1
  line each above `TestMonsterDestroyBytesV79` (gms_v79) and
  `TestMonsterDestroyBytesV72` (gms_v72) — 7 lines total, no other lines in the
  hunks. The bodies of `TestMonsterDestroy`, `TestMonsterDestroyBytesV79`, and
  `TestMonsterDestroyBytesV72` are byte-for-byte unchanged around the deleted
  comments (confirmed by reading the post-commit file, `destroy_test.go:1-70`).
- `v61_test.go` diff: removes exactly 1 line (the gms_v61 marker above
  `TestMonsterDestroyBytesV61`); no other change in the file.
- The consolidated 9-marker block (`destroy_test.go:93-101`) attached to
  `TestMonsterDestroySwallowGate` is present, untouched, and now the only
  MonsterDestroy marker set, covering gms_v61 through jms_v185 (9 versions,
  9 lines) — confirmed by direct read.
- All other packet markers in `v61_test.go` (MonsterSpawn, MonsterControl,
  MonsterMovement, MonsterMovementAck, MonsterStatSet, MonsterStatReset,
  MonsterDamage, MonsterMonsterSpecialEffectBySkill, MonsterMobCrcKeyChanged,
  MonsterHealth, MonsterCatchMonster) are present at lines 33-276, unmodified
  — confirmed via `grep -n "packet-audit:verify"` on the post-commit file:
  the only removed line was the MonsterDestroy one; every other packet's
  marker for gms_v61 remains.
- No version lost marker coverage: gms_v61 through jms_v185 (9 versions) all
  still have exactly one MonsterDestroy marker after the edit (in the
  consolidated block); none dropped.
- Independently re-ran `matrix --check` — confirmed exit 0, no duplicate- or
  missing-marker errors for MonsterDestroy or any other packet.

This requirement is fully satisfied; no over-reach found.

## Global constraints

- No invented values: v92 address `0x64bb90` traced to
  `docs/packets/ida-exports/gms_v92.json:6675`. Gate boundary text and field
  description are copied verbatim from the brief.
- `STATUS.md`/`status.json` show only tool-shaped diffs (state transition +
  rollup numbers) — consistent with regeneration, not hand-editing.
- Evidence YAML indentation matches each file's pre-existing style (verified
  per-file above).
- `gms_v48` untouched — confirmed via `git diff --stat` scoped to
  `docs/packets/evidence/gms_v48/*` showing no changes.
- Line endings: all touched files are LF-only pre- and post-change (checked
  for `\r` occurrences — none found in any touched file).

## Test honesty

This is a docs/evidence-registration task, not new production code; the only
test-file changes are marker-comment deletions with no assertion or logic
changes. `go build ./... && go test ./monster/clientbound/...` in
`libs/atlas-packet` passes (re-run independently, `ok`). Not applicable in the
usual "does a new test fail without the change" sense — there is no new test
logic in this task, only marker bookkeeping and evidence YAML/gate additions.

## Not evaluable

None. Full diff surface was read directly; both `git diff` and independent
tool re-runs were performed rather than relying on the implementer's report.

## Verdict

**Spec compliance:** the brief's six steps, as amended by the controller's
ruling on the duplicate-marker resolution, are all satisfied. No requirement
silently dropped, no scope creep, `gms_v48` untouched, `gate-lint --check`
correctly treated as out of scope per prior ruling.

**Task quality:** the implementer stopped and reported rather than guessing at
an out-of-brief file edit when it hit the duplicate-marker failure — the
correct move per the brief's own instruction and CLAUDE.md's grounding rule.
After the controller's ruling, the fix was applied minimally and precisely
(marker-line deletions only, no collateral edits), and every `--check` command
was re-verified rather than assumed. No defects found.
