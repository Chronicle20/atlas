# Review: Task 1 — Registry correction, MAKER_SKILL on gms_v72/gms_v79

Range: `45ee51248..6361eb1db` (1 commit: `6361eb1db fix(packets): register MAKER_SKILL on gms_v72 and gms_v79`)

## Scope

`git diff --stat` shows exactly 4 files touched:

```
docs/packets/audits/STATUS.md      |  6 +++---
docs/packets/audits/status.json    | 10 ++++++----
docs/packets/registry/gms_v72.yaml |  8 ++++++++
docs/packets/registry/gms_v79.yaml |  8 ++++++++
```

This matches the brief's "Files" section exactly (two registry files hand-authored, two audit
artifacts regenerated). No other files touched. Confirmed via `git diff` (full patch read, not
sliced — diff is small).

## Findings

### PASS — gms_v72.yaml entry matches brief verbatim

`docs/packets/registry/gms_v72.yaml` (post-change, inserted between `LOTTERY_ITEM_USE_REQUEST`
and `SUE_CHARACTER`, keeping the file opcode-ordered):

```yaml
- op: MAKER_SKILL
  direction: serverbound
  opcode: 112
  fname: CUIItemMaker::RequestItemMake
  provenance: ida-discovered
  ida:
    address: 7736515
  note: 'task-285: v72 COutPacket(0x70) = 112; ...'
```

opcode `112`, `ida.address: 7736515` (decimal, not hex), `fname: CUIItemMaker::RequestItemMake`,
`provenance: ida-discovered` — all match the brief's Step 2 block and the exact-values table
verbatim. Insertion point (immediately before `- op: SUE_CHARACTER`) matches Step 2 / the
insertion-point table.

### PASS — gms_v79.yaml entry matches brief verbatim

Same shape, `opcode: 111`, `ida.address: 7953859`, same `fname`/`provenance`, inserted
immediately before `- op: SUE_CHARACTER` in `gms_v79.yaml`. Matches Step 3 and the exact-values
table verbatim.

### PASS — `ida.address` stored as decimal integer, not hex string

Both entries store `7736515` and `7953859` as bare YAML integers (no quotes, no `0x` prefix),
consistent with the brief's "Verified facts" note and the existing `gms_v72.yaml:2640` pattern
this task was told to copy.

### PASS — status.json / STATUS.md regenerated, correct end state, no hand-editing artifacts

`status.json` diff for the `MAKER_SKILL` row:
- `gms_v72`: `state: n-a, opcode: -1` → `state: incomplete, note: "no audit report", opcode: 112`
- `gms_v79`: `state: n-a, opcode: -1` → `state: incomplete, note: "no audit report", opcode: 111`

This is the correct end state per the brief's explicit framing: Task 1 should land these two
cells at `❌ incomplete`, not `✅ verified` (Task 7 promotes them later by implementing the
codec). Confirmed `gms_v83` unchanged at `opcode: 113, state: incomplete` in the same block
(`docs/packets/audits/status.json:18199-18203`).

Read the full `MAKER_SKILL` cell block directly out of `status.json` (lines 18174-18213+): the
`note: "no audit report"` field is the tool's own generated marker for `incomplete` cells (also
present on v83/v84/v87), not a hand-edit — consistent with a regenerated file, not a manually
patched one.

`gms_v48` and `gms_v61` remain `state: n-a, opcode: -1`, unmoved, satisfying the brief's Step 5
requirement that these two MUST NOT move.

`STATUS.md` row diff for `MAKER_SKILL` shows the two new `0x070 | ❌` / `0x06F | ❌` cells
inserted in the v72/v79 columns, and the summary-table rows for v72/v79 shift by exactly one
`incomplete` (243/410 vs previous 242/411 for v72; 239/369 vs 238/370 for v79) — consistent
with one op flipping from unlisted/n-a to incomplete on each version, nothing else.

### PASS — no MAKER_RESULT scope creep

`git diff 45ee51248..6361eb1db | grep -i MAKER_RESULT` returns no matches (exit 1, empty). The
diff touches only `MAKER_SKILL` cells/entries. Task 6's ownership of the `MAKER_RESULT`
placeholder-address correction in these same two files is undisturbed.

### PASS — packet-audit gates independently re-run, all exit 0

Per instructions I did not re-run `tools/verify.sh`, but the four audit gates are the
task-specific correctness oracle for a registry-only change, so I ran them directly against the
worktree HEAD:

```
go run ./tools/packet-audit matrix --check       → exit 0
go run ./tools/packet-audit fname-doc --check    → exit 0
go run ./tools/packet-audit operations --check   → exit 0
go run ./tools/packet-audit dispatcher-lint      → exit 0
```

Matches the implementer's report of all four gates passing.

### PASS — no wire change to an already-verified version

Diff only touches `gms_v72` and `gms_v79`, both of which had `MAKER_SKILL` at `n-a` (not
`verified`) before this change. No `verified` cell for any op/version pair was touched.

### PASS — no literal absolute/home path, line endings preserved

`grep -rn '/home/'` over the four changed files returns no hits. `cat -A` on the new YAML lines
shows plain `$` (LF) line terminators, consistent with the rest of the files — no CRLF
normalization.

### PASS — opcodes resolved from config, not hard-coded in application code

This task only touches registry/audit data files, not Go source; the "opcodes never
hard-coded" constraint targets application code (e.g. dispatchers), which this diff does not
touch. Not applicable here, and correctly not touched.

## Not evaluable

- I could not independently verify the raw evidence (the `0x760cc3` / `0x795dc3` IDA addresses,
  the underlying `.i64` IDB files, or that these decimal conversions are correct against the
  actual binaries) — that evidence lives outside this diff's surface and was presumably
  established in the task's design/evidence phase, not in this commit. The decimal-to-hex
  arithmetic itself is internally consistent with the brief's stated `0x` values
  (`0x760cc3` = 7736515, `0x795dc3` = 7953859 — confirmed by calculation), so this is
  cross-checked at the arithmetic level, but not against the IDB itself.

## Task-quality verdict

The commit is a single, minimal, correctly-scoped registry-data change. It matches the brief's
step-by-step instructions exactly, produces the intended non-terminal (`incomplete`, not
`verified`) end state, does not regress adjacent rows/cells, does not creep into the
out-of-scope `MAKER_RESULT` correction, and all four audit gates pass under independent
re-execution. No defects found.

## Spec-compliance verdict

Matches every requirement in `task-1-brief.md`: exact opcode, fname, provenance, decimal
address, insertion point, and regenerated (not hand-edited) audit artifacts, for both
`gms_v72` and `gms_v79`.
