# Review — task-285 Task 6 (IDB wire derivation, `2df70429f..73939663c`)

Scope: two commits belonging to Task 6 — `2f2361700` (Steps 1/6/7, correct
`MAKER_RESULT` `ida.address` on gms_v72/v79 + coverage manifest) and
`73939663c` (Steps 2-5, `wire-derivation.md`). `6ff166527` (Task 5's
`StartWorker` test fix) sits inside the given range but is out of scope per
the dispatch instructions and was not re-reviewed here; confirmed by
`git show --stat` that its only touched file is
`services/atlas-data/.../data/data/processor_test.go`, unrelated to Task 6's
deliverables.

Deliverable read in full: `docs/tasks/task-285-maker-skill-crafting/wire-derivation.md`
(352 lines). Supporting reads: `2f2361700` diff, `.superpowers/sdd/plan/task-6-brief.md`,
`.superpowers/sdd/plan/task-6-brief-cont.md`, `.superpowers/sdd/plan/task-6-report.md`,
`docs/tasks/task-285-maker-skill-crafting/evidence-maker-skill-v72-v79.md`,
`docs/tasks/task-285-maker-skill-crafting/prd.md` §4.3/§4.4.

I did not have `ida-pro` MCP tool access in this review session (no
`ToolSearch` tool was exposed to me), so I could not independently query the
live IDBs to spot-confirm any address or decompilation. Every disposition
below is a check of the artifact's *internal* evidence — whether the claim is
backed by something quoted/citable inside the deliverable itself, per
CLAUDE.md's evidence rule — not an independent IDB re-derivation. That
limitation is recorded under "Not evaluable."

## Diff-scope facts (verified)

- `git diff --stat 2df70429f..73939663c` (excluding the out-of-scope Task-5
  commit): `docs/packets/registry/gms_v72.yaml` (+6/-1),
  `docs/packets/registry/gms_v79.yaml` (+6/-1),
  `docs/tasks/task-285-maker-skill-crafting/coverage-manifest.yaml` (new,
  +18), `docs/tasks/task-285-maker-skill-crafting/wire-derivation.md` (new,
  +352). No `libs/atlas-packet`, no codec, no template, no `gates.yaml` file
  touched anywhere in the range. `git diff --stat 6ff166527..73939663c` shows
  the Steps 2-5 commit touches only `wire-derivation.md`. Task 6 is
  derivation-only, confirmed.
- The only registry edits are to the two `MAKER_RESULT` entries'
  `ida.address` + a `note:` field (`git diff` hunks shown in `2f2361700`).
  `MAKER_SKILL` is untouched in both registry files across the whole range
  (`git diff ... | grep -i maker` shows no `MAKER_SKILL` hunk). This directly
  supports claim 1.
- Re-ran all three quoted gates myself from the worktree root:
  - `go run ./tools/packet-audit matrix --check` → exit 0 (two informational
    `n-a evidence consumed` notes, unrelated to maker).
  - `go run ./tools/packet-audit fname-doc --check` → exit 0.
  - `go run ./tools/packet-audit gate-check --check` → exit 0, "all 21
    gate(s) have verified byte-fixtures on both straddling versions (1
    partial-by-design)."
  All three genuinely exit 0, as claimed in the commit message and report.

## Claim-by-claim adjudication

### 1. R-1 discharged, no registry file touched — PASS

`wire-derivation.md:81-100` states R-1 discharged with a table citing the
committed decimal `ida.address`, its hex, and the "IDB" value side by side for
both versions, plus a cross-check of `MAKER_RESULT`'s already-corrected
addresses and the untouched `gms_v84` value. This is a genuine per-version
confirmation table inside the deliverable, not a bare assertion in the report
only (the report at `task-6-report.md:214-224` restates the same table). The
diff independently confirms no `MAKER_SKILL` registry hunk exists anywhere in
the range. Claim holds.

### 2. `gms_v84`/`gms_v92` — confirmed vs. inferred — PASS, adequately distinguished

`wire-derivation.md:56-79` gives a dedicated subsection, "The two unsymbolized
versions," that: (a) states plainly neither has a `CUIItemMaker` symbol, (b)
shows the structural-location arithmetic (`OnItemMakeResult` adjacency,
size-match via `lookup_funcs`), and (c) explicitly separates that from the
subsequent decompilation confirmation ("Its decompilation is the
`RequestItemMake` body verbatim... matching the `gms_v84`/`gms_v92` registry
opcode"). This is exactly the "structural location + opcode match +
decompiled body" chain the dispatch brief worried might be presented as
symbol-strength; the doc does not overstate it — it narrates the weaker
identification step honestly before citing the decompiled confirmation. The
one cosmetic gap: the summary table (line 16-17) and per-version tables
(200-207, 286-293) mark these two rows `IDENTICAL` with no footnote pointing
back to this subsection. Non-blocking — the distinction exists in the
document, just not cross-referenced from every table cell.

### 3. All eight IDENTICAL, so C-2 needs no gate — BLOCKING gap for `MAKER_RESULT`

For `MAKER_SKILL` (lines 196-207), the per-version table gives real,
version-specific decompiled evidence for three of the seven non-reference
versions — instruction addresses of the arm's `Encode4` call sites for
`gms_v72`/`v79`/`v83` (e.g. `0x760de7`/`0x760dcd`/`0x760d8c`) — and a
described (if unaddressed) arm shape for the rest.

For `MAKER_RESULT` (lines 282-293), **every** cell for the seven
non-reference versions reads `as above`, with no address, no field list, and
no branch/guard detail beyond a bare `yes` in the "guard" column. The only
per-version fact given is the top-level function entry address (already
listed in the Sessions table at lines 40-49). There is no quoted
decompilation or instruction address anywhere in the `MAKER_RESULT` section
(lines 229-320) for any version other than the `gms_v95` reference, whose own
"decompilation" is never actually quoted either — only a derived pseudocode
layout is given (contrast with the `MAKER_SKILL` section at 128-166, which
does quote real Hex-Rays C).

This matters because the C-2 "no gate required" verdict is exactly the kind
of claim CLAUDE.md's evidence rule targets: a missing gate that should exist
ships silently (no CI check catches an unguarded divergence), whereas an
unnecessary gate would have been caught by `gate-check --check` demanding
fixtures. The report claims "all eight IDB sessions were live and adopted"
and the doc contains genuinely specific, hard-to-fabricate detail elsewhere
(per-version `StringPool` id lists at lines 313-316, `GetItemTypeName`/
`ChatLogAdd` inlining differences at 301-311) that suggests real per-version
decompilation did happen — but none of that specificity is brought to bear on
confirming the *decode order itself* is identical across versions. The
write-up does not let a reader (or Task 7/8's implementer) verify the
`MAKER_RESULT` "IDENTICAL" verdict independently; it has to be taken on
trust from a "yes"/"as above" table.

**Action for a fix**: add, for each of the seven non-reference `MAKER_RESULT`
versions, at minimum the instruction addresses of the `nResult<=1` guard
Decode4, the `nMode` Decode4, and the `bNoItemGain`/`bUsedCatalyst` Decode1
guard calls — matching the specificity already given in the `MAKER_SKILL`
table for `gms_v72`/`v79`/`v83`.

### 4. C-3 contradiction with `evidence-maker-skill-v72-v79.md` / PRD — evidenced, but PRD citation is mislabeled (non-blocking)

The contradiction itself is well evidenced: `evidence-maker-skill-v72-v79.md:70`
literally reads `Encode4  nMode  // echoed` inside the mode-1/2 arm, following
a pre-switch `Encode4 nMode` at line 67 — a genuine double-encode rendering.
`wire-derivation.md:128-166` quotes the actual `gms_v95` decompilation showing
exactly one `Encode4` call, *inside* the switch arm, with no pre-switch mode
encode. This is a real, well-grounded contradiction (quoted code vs. quoted
code), and claim 4 is substantively correct.

However, the PRD citation is mislabeled: `docs/tasks/task-285-maker-skill-crafting/prd.md:184-188`
shows `FR-4.3` is the **`MAKER_RESULT` dispatcher-family** requirement, unrelated
to the double-encode. The double-encode code block that matches
`evidence-maker-skill-v72-v79.md` actually lives in PRD §4.3 "Craft
operations" (prd.md:117-136), as an unlabeled illustrative snippet before
`FR-3.1`. `wire-derivation.md:29` and the commit message both cite "PRD
FR-4.3" — this is the wrong requirement id. Non-blocking (the substance is
correct and the artefact-vs-artefact contradiction is real; only the specific
FR anchor is wrong), but should be corrected before Task 7 goes hunting for
"FR-4.3" and finds an unrelated requirement.

### 5. Three carry-forward findings for Tasks 7-9

- **Mode is `i32`, not a byte** — PASS, well grounded. The quoted `gms_v95`
  decompilation for `RequestItemMake` (128-166) shows `Encode4(&oPacket,
  m_nRecipeClass)`, and the `MAKER_RESULT` layout states `i32 nResult` /
  `i32 nMode` (238, 245) consistent with the brief's own sample
  (`task-6-brief-cont.md:53-57`, `Decode4`). Concrete evidence exists for the
  encode side; the decode side rests on the same thin `MAKER_RESULT`
  documentation flagged in item 3 above, but the specific "4 bytes not 1"
  claim is plausible and consistent with the brief's sampled shape.
- **`bNoItemGain`/`bUsedCatalyst` are real wire bytes** — under-evidenced.
  `wire-derivation.md:275-280` states this in prose ("each appears as a
  single `CInPacket::Decode1(v2)` call evaluated once as the `if`
  condition... This settles the caveat the prior agent flagged") but quotes
  no decompiled code and cites no instruction address for any version. This
  is precisely the ambiguity the first agent explicitly could not resolve
  from the flattened export (`task-6-report.md:111-115`: "Do not build a
  codec off this list"), and the second agent's resolution is the single
  highest-stakes claim in the whole document — it changes whether Task 7/8
  reads 1 wire byte or treats it as decompiler noise. It deserves a quoted
  snippet (e.g. `if ( CInPacket::Decode1(v2) ) { ... }`) the way the
  `MAKER_SKILL` section provides one; it does not have one. Rolls into the
  same blocking finding as item 3.
- **Modes 1/2 share an arm but need two registered arms, plus a bodyless
  `nResult > 1` form** — PASS as a logical inference. Given the encode side
  is shown to encode `m_nRecipeClass` verbatim (not collapsed to a canonical
  value) and the decode side reads `nMode` before dispatching on it, the
  conclusion that 1 and 2 must be preserved as distinct wire values is sound
  reasoning from the (better-evidenced) `MAKER_SKILL` decompilation, applied
  consistently to the decode side. The bodyless `nResult > 1` form and the
  "unsigned compare" characterization (line 238, 244) are asserted without a
  quoted comparison instruction (Hex-Rays would render an unsigned compare
  distinctly, e.g. `(unsigned int)v53 <= 1`), so "UNSIGNED" specifically is
  not independently checkable from the artifact — folded into the same
  finding as item 3.

## Findings summary

**Blocking**

1. `docs/tasks/task-285-maker-skill-crafting/wire-derivation.md:282-293` (and
   the surrounding `MAKER_RESULT` section, 229-280) — the `MAKER_RESULT`
   per-version table is `as above` for all seven non-reference versions with
   no quoted decompilation or instruction address anywhere in the section;
   the "field-for-field IDENTICAL across all eight" claim underlying the C-2
   no-gate verdict, and the specific `bNoItemGain`/`bUsedCatalyst`
   real-byte-vs-guard and unsigned-`nResult`-compare claims that Task 7/8
   will act on, are not traceable to quoted tool output the way the parallel
   `MAKER_SKILL` section's `gms_v95`/`v72`/`v79`/`v83` rows are. A missing
   gate here is a silent wire bug with no CI catch, per the task's own stated
   asymmetry ("an unnecessary gate is cheap; a missing one is a wire bug").

**Non-blocking**

1. `docs/tasks/task-285-maker-skill-crafting/wire-derivation.md:29` — cites
   "PRD FR-4.3" for the double-encode transcription artefact; the actual PRD
   double-encode snippet is in §4.3 "Craft operations" (prd.md:117-136),
   unlabeled by an FR id, while `FR-4.3` (prd.md:184) is the unrelated
   `MAKER_RESULT` dispatcher requirement. Wrong anchor, correct substance.
2. `docs/tasks/task-285-maker-skill-crafting/wire-derivation.md:203-204,207`
   — `gms_v84`, `gms_v87`, `jms_v185` rows in the `MAKER_SKILL` table give no
   instruction address for the mode-encode call site, unlike `v72`/`v79`/
   `v83`. Weaker than ideal but the arm-shape and (for v84) full-body-verbatim
   match give it more support than the `MAKER_RESULT` gap above.
3. `docs/tasks/task-285-maker-skill-crafting/wire-derivation.md:16-17,286-293`
   — the two structurally-located versions (`gms_v84`, `gms_v92`) are marked
   `IDENTICAL` in the summary/per-version tables with no footnote back to the
   "two unsymbolized versions" subsection that actually justifies them.

## Not evaluable

1. No `ida-pro` MCP tool access in this review session (no `ToolSearch` tool
   exposed) — could not independently re-query any of the eight live IDBs to
   spot-confirm a single address, decompilation, or field order. Every
   disposition above evaluates the artifact's internal evidentiary
   sufficiency, not ground truth against the IDBs themselves.
