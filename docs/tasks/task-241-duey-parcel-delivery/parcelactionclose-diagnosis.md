# Diagnosis: `parcel/serverbound/ParcelActionClose` sub-struct row anomaly

Read-only investigation. No codec, registry, template, evidence record, or
`status.json` was modified.

## Verdict

**Tooling bug, not correct behavior.** The sub-struct row's `incomplete` /
`"no audit report"` state on gms_v83, gms_v84, gms_v87, gms_v92, gms_v95, and
jms_v185 is factually wrong — the report, evidence record, and pinned
`packet-audit:verify` fixture marker all exist on every one of those six
versions. The op-level `DUEY_ACTION` row's `verified` claim is **sound** and
unaffected — it derives from the very same evidence the sub-struct row is
failing to surface. The fname-collision hypothesis is confirmed as the
mechanism, but commit `681ebfdcc` did **not** cause it — the underlying
registry collision is ancient (task-085, commit `69bffe88d`) and the
suppression already existed before `681ebfdcc` landed. The defect class is
**not unique to this row**: the same mechanism suppresses ~650 other
sub-struct×version cells repo-wide, though only a subset of those (checked
selectively, not exhaustively) carry the full independent-evidence chain that
would actually flip them to `verified` if fixed.

## 1. The mechanism (traced in `tools/packet-audit/internal/matrix/build.go`)

`Build()` runs two passes:

1. **Op-row pass.** For each op, `worstCandidateCell` (build.go) looks up
   `fnameWriters[vk][ref.FName]` — every report whose `baseFName(IDAName)`
   equals the op's *registry* `fname` for that version. Every writer name it
   finds is marked `used[vk][wn] = true` (build.go, `worstCandidateCell`,
   the `for _, wn := range writers { used[vk][wn] = true ... }` loop) — this
   happens regardless of whether the op has one candidate or several.

2. **Sub-struct pass.** For each per-version report, if
   `usedWriters[vk][wn]` is true, the report is `continue`d (dropped) unless
   `wn` is also `protectedWriters[vk][wn]` — which only happens if `wn` is
   listed in the explicit allowlist `legacyConsumedSiblingWriters`
   (build.go). A dropped report leaves that (packet, version) cell
   unpopulated; the later gap-fill loop then stamps it
   `Cell{State: StateIncomplete, Note: "no audit report"}` (or `n-a` if
   dispositioned) with **no reference to whether a report file actually
   exists** — the note text is unconditional, not derived from a real
   absence check.

`legacyConsumedSiblingWriters` today lists exactly one entry:
`NOTE_ACTION|serverbound -> {NoteOperationDiscard, NoteOperationSend}`
(task-137). `DUEY_ACTION` has no entry.

### Why gms_v72/gms_v79 grade correctly and the other six don't

`docs/packets/registry/{gms_v72,gms_v79}.yaml` give `DUEY_ACTION` the fnames
`CTabReceive::ReceiveParcel` and `CTabSend::SendParcel` respectively — neither
equals `ParcelActionClose`'s own `IDAName`
(`CParcelDlg::CloseParcelDlg`), so its report is never claimed by the op row
on those two versions and the sub-struct pass grades it from its own
evidence: `verified`.

`docs/packets/registry/{gms_v83,gms_v84,gms_v87,gms_v92,gms_v95,jms_v185}.yaml`
all give `DUEY_ACTION` the fname `CParcelDlg::CloseParcelDlg` — identical to
`ParcelActionClose`'s own `IDAName`. Verified directly:

```
docs/packets/registry/gms_v83.yaml:2228-2231
- op: DUEY_ACTION
  direction: serverbound
  opcode: 65
  fname: CParcelDlg::CloseParcelDlg
```

`fnameWriters["gms_v83"]["CParcelDlg::CloseParcelDlg"]` therefore includes the
`ParcelActionClose` writer, `worstCandidateCell` marks it `used`, and the
sub-struct pass drops it (not protected) on all six versions — producing
exactly the observed `incomplete` / `"no audit report"` split.

## 2. Is it a tooling bug or correct behavior?

Bug. On all six suppressed versions the report is real, and it carries the
full independent-verification chain:

- Report: `docs/packets/audits/{v}/ParcelActionClose.json` — present for
  all 6.
- Evidence: `docs/packets/evidence/{v}/parcel.serverbound.ParcelActionClose.yaml`
  — present for all 8 versions (including the 2 that already grade correctly).
- Pinned fixture marker, one per version, e.g.
  `libs/atlas-packet/parcel/serverbound/v83_test.go:94`:
  `// packet-audit:verify packet=parcel/serverbound/ParcelActionClose version=gms_v83 ida=0x6f5691`
  — confirmed present for all 8 versions (v72, v79, v83, v84, v87, v92, v95,
  jms_v185).

If graded independently (as gms_v72/gms_v79 already are), these six cells
would grade `verified` from the exact same evidence that already backs the
op-level `DUEY_ACTION` row's `verified` state on those versions. Instead they
are silently discarded before grading ever runs, and the row falsely claims
no report exists. This is precisely the defect class `legacyConsumedSiblingWriters`
was built to patch for `NOTE_ACTION` (see the long comment at the top of
`build.go`) — `DUEY_ACTION`/`ParcelActionClose` is the same shape of defect,
simply never added to that allowlist.

## 3. The two hypotheses

**Fname hypothesis — mechanism confirmed, causal claim about commit
`681ebfdcc` rejected.**

The split does fall exactly on which versions' *registry* `fname` for
`DUEY_ACTION` equals `CParcelDlg::CloseParcelDlg` (this sub-struct's own
`IDAName`) — that is the actual, verified predicate driving the
suppression. But commit `681ebfdcc` only touched the **seed templates**
(`tools/packet-audit/seed-data/templates/template_gms_{72,83,84,87,92,95}_1.json`,
`template_jms_185_1.json`), which feed `RoutedNames`/`Routed` — a completely
different consumer from `fnameWriters`, which is built from
`in.Registry` (`docs/packets/registry/*.yaml`), not from templates.

The registry `fname: CParcelDlg::CloseParcelDlg` for `DUEY_ACTION` on
gms_v83 (and the other five versions) is not new. `git log -L` on that field
traces it to:

```
69bffe88d task-085: evidence-graded packet coverage matrix + 5-version coverage (1154 verified) (#747)
+- op: DUEY_ACTION
+  direction: serverbound
+  opcode: 65
+  fname: CParcelDlg::CloseParcelDlg
```

— i.e. the collision predates task-241 entirely. `681ebfdcc` is unrelated to
the sub-struct suppression; it is a coincidental correlation in the same
identifier, not a cause.

**Against-the-hypothesis check — confirmed, and it's the correct falsifier.**
Git history for the DUEY_ACTION/PARCEL verification commits, oldest first:

```
90c44a7a3  docs(packets): verify PARCEL and DUEY_ACTION coverage cells on gms_v83
...
6e3106d87  feat(packets): verify PARCEL and DUEY_ACTION on gms_v84 (batch 4/8)
5f1ea3a35  feat(packets): verify PARCEL and DUEY_ACTION on gms_v87 (batch 5/8)
9d790633d  feat(packets): verify PARCEL and DUEY_ACTION on gms_v92 (batch 6/8)
a8adafb12  fix(packets): gate GMS v92+ equip potential/socket trailer in Asset codec
bdaa657b0  docs(task-241): record RULING 24 fix, RULING 25 defect, and batch reviews
681ebfdcc  fix(packets): correct DUEY_ACTION template fname to the registry primary   <-- the fix in question
011a71c9a  docs(task-241): ledger the RULING 25 fix and controller handoff
8c6cdac73  feat(packets): verify PARCEL and DUEY_ACTION on gms_v95 (batch 7/8)
f12826669  feat(packets): verify PARCEL and DUEY_ACTION on jms_v185 (batch 8/8, FINAL)
```

`681ebfdcc` lands *after* batches 4–6 (v84, v87, v92) and before batches 7–8
(v95, jms_v185). The execution ledger (`.superpowers/sdd/plan/progress.md:3509`,
batch 5 / commit `5f1ea3a35`) states: *"`ParcelActionClose` staying
`incomplete` confirmed as pre-existing on v83/v84, not a regression."* That
observation was made two batches (and one whole session) before `681ebfdcc`
existed — so the suppression on v83/v84 (and, by the same registry-fname
mechanism, v87/v92/v95/jms_v185) is confirmed pre-existing, not something the
fname fix introduced or changed. **Both hypotheses land in the same place:**
the mechanism is the fname collision, but the collision is in the op
registry and dates to task-085 (2026, far earlier in the branch's history),
not to `681ebfdcc`.

## 4. Effect on the campaign's headline claim

**Not affected.** The op-level `DUEY_ACTION` row
(`docs/packets/audits/status.json`, `op: "DUEY_ACTION"`,
`packet: "parcel/serverbound/ParcelActionReceive"`) reads `verified` on all
8 applicable versions. On gms_v83/84/87/92/95/jms_v185 that `verified` state
is derived — via `worstCandidateCell`'s "worst of all candidates sharing this
version's op fname" rule — from grading `ParcelActionClose`'s own report
(since its `IDAName` base-matches the op's registry fname there). So the
op-level claim is independently sound; it already reflects exactly the
evidence the sub-struct row is failing to show. The bug is a *display/row*
defect (the sub-struct row under-reports what the tool already knows), not
an *evidence* defect — nothing about DUEY_ACTION's actual coverage is
overstated.

## 5. Blast radius: repo-wide, not unique to this row

The suppression mechanism (an op-row's registry `fname` matching a
report's own `IDAName`, without a `legacyConsumedSiblingWriters` entry)
was checked mechanically against the full registry + report set. Re-deriving
`fnameWriters` and the op-fname consumption predicate directly from
`docs/packets/registry/*.yaml` and `docs/packets/audits/*/​*.json` (script,
not the real `Build()`, so treat as a strong lead rather than the tool's own
output) found **~650 sub-struct×version cells across dozens of ops** — login,
party, quest, messenger, guild, cash, buddy, character, field/MTS, channel,
account, and more — where a report exists on disk for a cell currently
marked `incomplete` / `"no audit report"` because its writer is claimed by
an unlisted op fname. Spot-checked a few for the deeper "would actually grade
verified" bar (own pinned `packet-audit:verify` marker + evidence yaml, not
just report existence):

- `login/clientbound/AuthSuccess` — has fixture markers for all 9 non-gms_v48
  cells, but evidence yaml only for `{gms_v48,61,72,79,83,84}` — the
  `{gms_v87,95,jms_v185}` legs would *not* fully flip to `verified` even if
  unsuppressed (missing evidence, a separate gap).
- `party/serverbound/PartyOperation`, `quest/serverbound/ActionScriptStart`
  — both have per-version markers on the suppressed legs; evidence chain not
  individually re-verified here.

So the suppression mechanism itself is confirmed to be broad and systemic
(the note text `"no audit report"` is unconditionally wrong wherever a report
file exists, regardless of whether it's independently gradable), but how many
of the ~650 cells would actually promote to `verified` if fixed varies
per-cell and was **not exhaustively re-verified** in this investigation —
that would need a proper sweep using the real `Build()`/`gradeSubStructCell`
path (or an equivalent script that also cross-checks `Evidence`/`Markers`,
not just report existence). `ParcelActionClose` is a confirmed, fully-checked
instance (report + evidence + marker present on all 6 suppressed versions);
it is representative of the defect class, not the full extent of it.

## Recommended fix (not applied — diagnostic only)

Two options, in order of how closely they match the existing pattern:

1. **Narrow, precedented fix (matches `legacyConsumedSiblingWriters`'s own
   stated policy):** add `DUEY_ACTION|serverbound -> {ParcelActionClose}` to
   `legacyConsumedSiblingWriters` in `build.go`, mirroring the NOTE_ACTION
   precedent exactly — same per-cell verification bar the comment already
   requires (own pinned TIER1 evidence, independently verified, never a
   lesser grade). This fixes the six cells in this row and nothing else, at
   the cost of the same one-entry-per-op toil the comment already flags as
   the accepted tradeoff.

2. **Structural fix (bigger, addresses the ~650-cell blast radius):** stop
   defaulting a consumed-but-report-existing cell to the misleading
   `"no audit report"` note. At minimum, change the gap-fill branch in
   `Build()` (the `else { mr.Cells[vk] = Cell{State: StateIncomplete, Note:
   "no audit report", ...} }` branch) to distinguish "genuinely no report"
   from "report exists but consumed by op row" (e.g. a `note:
   "consumed by <OP> op row"` on the latter, and — the real fix — attempt
   `gradeSubStructCell` on every consumed-but-reported writer, keeping the
   op row's own state as a floor rather than unconditionally dropping it).
   The build.go comment explicitly rejected a *general structural predicate*
   before ("both were tried and measured to also flip dozens of unrelated,
   already-correct cells... whose sibling arms are legitimately meant to stay
   suppressed in most versions") — so a structural fix needs to reproduce
   that same care (grade-then-take-max, not blanket-unsuppress) rather than
   just deleting the `continue`. Given that prior finding, option 1
   (extending the explicit allowlist) is the safer near-term fix for this
   task; option 2 is a separate, larger tooling task.
