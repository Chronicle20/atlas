# Review: task-27 — drop the spurious gms_v84 MapleLife registry rows

Commit under review: `77b302206` (range `7b2196004..77b302206`)
Brief: `.superpowers/sdd/plan/task-27-registry-correction-brief.md`
Report: `.superpowers/sdd/plan/task-27-report.md`

## Scope

`git diff --stat 7b2196004..77b302206`:

```
docs/packets/audits/STATUS.md      |  6 +++---
docs/packets/audits/status.json    | 10 ++++------
docs/packets/registry/gms_v84.yaml | 12 ------------
3 files changed, 7 insertions(+), 21 deletions(-)
```

Matches the brief's "Files" list exactly. No Go source, no other version's
registry, no codec/template touched.

## Findings

### 1. No collateral cell movement — PASS

`git show 77b302206 -- docs/packets/audits/status.json` contains exactly two
hunks, one per deleted opcode:

```
gms_v84: {"state":"incomplete","note":"no audit report","opcode":349} -> {"state":"n-a","opcode":-1}
gms_v84: {"state":"incomplete","note":"no audit report","opcode":350} -> {"state":"n-a","opcode":-1}
```

No other op's cell, and no other version's cell for these two ops, appears in
the diff. `STATUS.md`'s diff touches only the `MAPLELIFE_RESULT`/`MAPLELIFE_ERROR`
row's `gms_v84` column (`0x15D ❌`/`0x15E ❌` -> blank/⬜) and the v84 summary
row (`331 incomplete/248 n-a` -> `329/250`, exactly two rows moving category).

Cross-checked all five registry files directly (not from the report):

```
gms_v83.yaml:1805 MAPLELIFE_RESULT / :1810 MAPLELIFE_ERROR / :3470 MAPLELIFE_CHECK_NAME
gms_v87.yaml:1908 / :1913 / :3651
gms_v92.yaml:2102 / :2108 / :3931
gms_v95.yaml:2118 / :2123 / :4100
gms_v84.yaml: no MAPLELIFE_RESULT/ERROR/CHECK_NAME rows remain
```

v83/v87/v92/v95 rows for all three ops are present and untouched (absent from
`git diff --stat` entirely), consistent with them remaining `verified`.

### 2. Deletion is complete and surgical — PASS

`git show 77b302206 -- docs/packets/registry/gms_v84.yaml` is a single
contiguous 12-line removal (both full row blocks, including their `note:`
lines), with the unchanged `VICIOUS_HAMMER` row immediately following. Grep
of the post-commit file for `opcode: 349$|opcode: 350$|opcode: 359$|opcode: 360$`
returns nothing. No other line in `gms_v84.yaml` changed; no other version's
registry file appears in the diff.

### 3. Artifacts are genuinely tool-produced — PASS (independently verified)

Re-ran `go run ./tools/packet-audit matrix` from repo root against the
current worktree: exit 0, and `git status`/`git diff --stat` on
`status.json`/`STATUS.md` show **zero diff** afterward — a clean
regeneration from repo root reproduces exactly what's committed.

Checked for the stray-path artifact the report flags (`tools/packet-audit`
run from its own directory writing to a bogus relative path): `ls
tools/packet-audit/` has no `docs/` subdirectory, no stray `.json`/`STATUS.md`
files anywhere under `tools/packet-audit/`, and `git status --porcelain -- tools/packet-audit/`
is empty. `git status --porcelain --untracked-files=all` at large shows only
pre-existing, unrelated task-review artifacts (`docs/tasks/task-246-.../review-task-*.md`,
etc.) — nothing packet-audit-related. The implementer's cleanup claim holds.

### 4. Evidence citations accurate as quoted — PASS (citations checked, not re-derived)

Read the actual `VICIOUS_HAMMER` note in `gms_v84.yaml` (task-129, the row the
brief cites): "Matches the +10 v84 clientbound shift (v83 0x162 -> v84
0x16C)" and "...whose gate (sub_7FD845) only handles 359 (name-change) and
360 (world-transfer) — 361 is a DEAD opcode there." This matches the brief's
paraphrase of the shift and the gate's handled range.

`MAPLELIFE_CHECK_NAME` has no v84 row — confirmed by the cross-registry grep
above (present in v83/v87/v92/v95, absent from v84).

No opcode 359/360 collision exists in `gms_v84.yaml`, before or after the
deletion (grep empty both times, consistent with the report's pre-check and
my post-check).

I do not have IDA access and did not re-derive the `CField::OnPacket`
dispatch-ladder claims or the `CharacterSaleDlg` RTTI-absence claim from the
binary itself — those are outside this review's evidence surface and are
reported here as not independently re-verified, per the task instructions.

## Not evaluable

- The binary-level claims underlying the brief (opcode 349's dispatch-ladder
  fallthrough, opcode 350's routing to the this[134] forwarder, the
  `CharacterSaleDlg` RTTI-string absence) were not re-derived from IDA; this
  review checked only that the brief's in-repo citations (the task-129 note
  text, the MAPLELIFE_CHECK_NAME-has-no-v84-row claim) are accurately quoted.

## Verdict

APPROVED. No collateral cell movement, deletion is complete and surgical,
regeneration is verifiably clean (reproduced independently), and the
in-repo evidence citations are accurate as quoted.
