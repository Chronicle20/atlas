# Review: Task 2 — jms_v185, gms_v95, gms_v48, gms_v61 structures

Range reviewed: `b9bd002d1..e45a6ed34`
Diff source: `.superpowers/sdd/plan/review-b9bd002d1..e45a6ed34.diff`
Commit: `e45a6ed34` — "docs(task-281): jms_v185 + gms_v95 layout, gms_v48 VERSION-ABSENT, gms_v61 present"

## Scope

`git diff --stat` for the range shows exactly four new files, all under
`docs/tasks/task-281-map-back-effects/structures/`:

```
structures/gms_v48.md   | 113 ++
structures/gms_v61.md   | 146 ++
structures/gms_v95.md   |  86 ++
structures/jms_v185.md  |  85 ++
```

No file outside `structures/` is touched. `absence-clear-v72-v79.md` was **not**
created (Step 3 cancellation honored). `structures/gms_v72.md` and
`structures/gms_v79.md` (Task 1's files) are untouched (confirmed: not in the
diff, still present on disk, no modification). Documentation-only, as required
— no Go, no test suite, no build touched. Scope confirmed as specified.

## Findings

### 1. Skeleton / layout consistency — PASS

`gms_v95.md`, `jms_v185.md` follow the `gms_v87.md` reference skeleton: IDB
session header, Step-0 already-implemented check, Router (switch table +
opcode cross-check against the registry), SET read-order (decode callee C
listing + 4-row table), Branch shape, Verdict-vs-reference line, CLEAR section
(thunk listing, "Packet reads: none", Verdict line). `gms_v48.md`/`gms_v61.md`
follow the brief's absence/positive skeletons (router arms, positive-absence
evidence with hit counts / positive-presence evidence with decode+thunk).

### 2. jms_v185.md — PASS

- Router: `case 0x7E (126) -> OnSetBackEffect`, `case 0x80 (128) ->
  OnClearBackEffect` (`jms_v185.md:15-19`). Matches the brief's stated
  opcodes (SET=126, CLEAR=128) and the registry cross-check at
  `jms_v185.md:22-24`, verified directly against
  `docs/packets/registry/jms_v185.yaml:625` (opcode 126) and `:635` (opcode
  128) — both true; the file's cited line numbers (`623`, `633`) are off by 2
  from the actual `opcode:` lines (registry entries have a header line before
  the field), a cosmetic citation drift, same class as a deferred Task 1
  finding — non-blocking.
- Read-order table: `Decode1(nEffect) / Decode4(nFieldID) / Decode1(nPageID) /
  Decode4(tDuration)`, 10 bytes total (`jms_v185.md:39-52`) — matches the
  Global Constraint reference layout exactly, in the same field order.
- Branch shape documents the two-value enum (0=show fade-in alpha 255, 1=hide
  fade-out alpha 0) and states "any other value: neither arm is entered ...
  falls to its epilogue without touching the field" (`jms_v185.md:63-65`) —
  matches the Global Constraint.
- Verdict: "IDENTICAL... No JMS read-order divergence" (`jms_v185.md:66-67`) —
  a real verdict statement against the reference, not a bare assertion; the
  read-order table backs it.
- CLEAR: thunk to `ReloadBack`, "Packet reads: none" (`jms_v185.md:76-79`),
  matching the Global Constraint (empty body, opcode only).
- All addresses attributed to the live IDB session `a977912e`, no bare
  unattributed address.

### 3. gms_v95.md — PASS

Explicitly headed "Source of this record: transcription of
`docs/tasks/task-281-map-back-effects/design.md` §1.1 / §1.2, not a fresh
decompile run in task 2" (`gms_v95.md:5-8`), naming the session the design's
derivation ran against (`ecc757f4`) and stating addresses are quoted from the
design document verbatim. This matches the transcription pattern the review
brief asked for (Task 1's v72/v84 treatment). Layout table matches the
reference exactly; router opcodes (144/145/146) match the brief's Files list.
The CLEAR section documents the `ReloadBack` reference shape used to
fingerprint the handler on every other version — this is the shape that
`gms_v61.md`, `gms_v48.md`, `jms_v185.md`, and Task 1's files all cite.

### 4. gms_v48.md — absence proof — PASS

- Router arms enumerated exhaustively: no `CMapLoadable::OnPacket` exists on
  this version at all (33 named `*::OnPacket` methods enumerated, none is
  CMapLoadable's); the actual field router `CField::OnPacket @ 0x4c66f2` is
  read in full, direct-switch table plus fall-through ranges
  (`gms_v48.md:12-46`).
- Opcode-slot ownership named: opcode 95 = `CField::OnSetQuestTime`, opcode 96
  = `CField::OnWarnMessage` (`gms_v48.md:57-62`) — verified independently
  against `docs/packets/registry/gms_v48.yaml:679-689` (`SET_QUEST_TIME`
  opcode 95, `ARIANT_RESULT` opcode 96 — close enough; the file's own
  IDA-derived names differ slightly from the registry's op-name but the
  opcode/address correspondence is what the file relies on, not the
  registry's op label).
- Two binary-wide searches recorded with hit counts: `CUser::GetVecCtrl` xref
  sweep (82 xrefs, "complete list, not truncated") isolates one
  `ReloadBack`-shaped candidate that has exactly 1 xref from
  `CConfig::ApplySysOpt` (a video-option reload, not a packet handler)
  (`gms_v48.md:73-86`).
- Does not rest on `unresolved: true`: the file explicitly says so —
  "Recorded for completeness only — neither was treated as evidence; the
  router enumeration and the three searches above are the proof"
  (`gms_v48.md:88-90`). This directly satisfies the brief's warning.
- Verified independently: `docs/packets/registry/gms_v48.yaml` has no
  `SET_BACK_EFFECT`/`CLEAR_BACK_EFFECT`/`BackEffect` entry at all (grep
  returns nothing) — the file's closing claim is true.
- Discloses the scope limitation itself ("not an image-wide
  instruction-pattern scan... that scan was not run") rather than hiding it —
  per the task instructions this is accepted by the controller and is noted
  here only as disclosure quality, not a defect.

### 5. gms_v61.md — positive-presence proof — mostly rigorous, one factual defect

Positive-presence documentation is thorough and matches the brief's ask:
router arm (`CField::OnPacket @ 0x4e9ea3` narrowing to `CMapLoadable::OnPacket
@ 0x5a81b9`, opcode window 95..96), decode callee with the 4-read table
(identical widths/order to the reference), branch shape, thunk target
(`sub_5A871B` -> `sub_5A81E2` = `ReloadBack`), zero-read claim for the clear
arm, and structural 4-for-4 comparison to the v95 reference shape.

**Blocking defect — false claim about registry state (`gms_v61.md:10-11`):**

```
> `docs/packets/registry/gms_v61.yaml` currently has **no** entry for either op.
> This is the same failure mode the controller ruling flagged for v72/v79.
```

This is false and was checkable at commit time. `docs/packets/registry/gms_v61.yaml`
already carried, at the task's base commit (`b9bd002d1`, unchanged since
`3d77511d0`), two placeholder entries whose addresses match this file's own
derivation exactly:

```
- op: IDA_0X05F
  opcode: 95
  fname: sub_5A8316
  address: 5931798        # = 0x5A8316
  note: '... v61-region low op; exact op-name Stage E.'
- op: IDA_0X060
  opcode: 96
  fname: sub_5A871B
  address: 5932827        # = 0x5A871B
  note: '... v61-region low op; exact op-name Stage E.'
```

(`docs/packets/registry/gms_v61.yaml:648-665`)

Both entries exist, at the correct opcodes and the correct addresses, waiting
on a name resolution — not absent. The characterization "same failure mode as
v72/v79" additionally mischaracterizes the nature of the problem: v72/v79's
failure mode (per the controller ruling this task carries) was a *wrong*
entry present with high confidence and no decompile behind it
(`provenance: manual`, opcode mis-attributed to `SET_MAP_OBJECT_VISIBLE`).
v61's actual state is a *correctly addressed but unnamed* placeholder
(`provenance: manual`, explicit "exact op-name Stage E" note) — a Stage-E
name-resolution gap, not a mis-attribution. These are different problems with
different fixes (rename vs. remove-and-replace). A Task 4 implementer who
trusts this file's framing at face value could try to add fresh entries
rather than rename the existing two, or could go looking for a wrong entry
to remove that does not exist.

The file's own "Downstream consequences" section (`gms_v61.md:143-148`) softens
this to "registry needs entries added" — still imprecise (they need renaming,
not addition) but not as actively misleading as the opening callout's "no
entry... same failure mode as v72/v79."

This is a documentation-accuracy defect inside an evidence-grade record other
tasks are meant to trust without re-deriving. It does not affect the derived
opcodes/addresses/read-order (all independently verified correct above), but
it misrepresents a fact the implementer could and should have checked with
one `grep` before writing it, in a task whose entire brief is "do not accept
[a claim] as evidence... read it yourself."

### 6. Step-3 cancellation honored — PASS

No `absence-clear-v72-v79.md` in the diff or on disk. `gms_v72.md`/`gms_v79.md`
untouched.

### 7. Commit message — PASS

Amended to `docs(task-281): jms_v185 + gms_v95 layout, gms_v48 VERSION-ABSENT,
gms_v61 present`, with a commit-body note explaining why it departs from the
controller's literally specified subject. This is accurate: it reflects the
real per-file outcome (gms_v48 absent, gms_v61 present, jms_v185 + gms_v95
derived), rather than the now-falsified "VERSION-ABSENT for both" premise.
Per the review brief, the controller already endorsed the departure; this
check confirms the resulting text is itself correct, which it is.

### 8. Address attribution — PASS

Every address in all four files is either (a) attributed to a live IDB
session read this task ran (`a977912e` for jms_v185, `12a398ce` for gms_v48,
`921fdbb5` for gms_v61), or (b) explicitly marked as a transcription from
`design.md` §1.1/§1.2 quoting the design's own session (`ecc757f4`) for
gms_v95. No bare unattributed address found.

## Controller-accepted items (not re-litigated)

- gms_v61 carrying both ops (95/96) is treated as true per the controller's
  independent IDB verification; reviewed only for documentation rigor above.
- gms_v48's absence proof lacking an image-wide instruction-pattern scan is
  accepted by the controller; noted above as a disclosure-quality positive
  (the file says so itself), not re-raised as blocking.
- The commit-subject amendment is accepted; checked here only for accuracy of
  the resulting text, which is correct.
- v61's handler functions being unnamed `sub_*` in the IDB is a known,
  controller-owned escalated item; not raised here.

## Not evaluable

None. The full review surface (the four new files, the diff, and the cited
registry lines) was directly inspectable from the worktree.

## Verdict

CHANGES_REQUIRED — one blocking, locatable factual defect
(`gms_v61.md:10-11`) misrepresenting the registry's current state to a degree
that could misdirect Task 4's registry work. Everything else — the core
derivation (opcodes, read order, branch shape, thunk targets, absence proof
for gms_v48, scope, Step-3 cancellation, commit message) — is accurate and
well-evidenced.
