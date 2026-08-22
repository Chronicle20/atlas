# Review — Task 2: Derive clientbound wire and duplicate-probe opcode

**Range:** `98efc7f0d..2f2e5d8c7` (single commit `2f2e5d8c7`)
**Diff:** 1 file, `docs/tasks/task-246-maple-life-character-creation/derivation.md`, +388/-3
**Brief:** `.superpowers/sdd/plan/task-2-brief.md`
**Report:** `.superpowers/sdd/plan/task-2-report.md`

## Scope note

Pure documentation task — no Go code, no tests, `tools/verify.sh` not applicable.
This review is a documentation-conformance and cross-artifact-consistency review,
not a fresh IDA re-derivation: this reviewer's toolset does not include
`func_query`/`insn_query`/IDB access, so the raw decompiled facts (instruction
addresses, per-address opcode literals, branch-arm structure as compiled) could
not be independently re-derived from the binaries. What *was* checked
independently: (a) every opcode/address/registry claim in §4–§6 against the
actual repo files it claims to match (registry YAMLs, `status.json`,
`template_gms_83_1.json`, `ida-exports/gms_v84.json`), (b) that §1–§3 were not
rewritten, (c) internal consistency of the new §4–§6 text and its citations of
Task 1's own document, (d) that the escalation for §6.3 was handled per the
brief's "stop and escalate" instruction rather than resolved by guessing.

## §1–§3 integrity — PASS

`git diff 98efc7f0d..2f2e5d8c7` shows exactly one removal hunk: the Task-1
placeholder comment at the bottom of the file (the `<!-- Task 2 appends... -->`
marker), replaced with an updated marker that documents a genuine numbering
discrepancy between Task 1's placeholder text and Task 2's actual brief (Task 1
labelled the sections §4=OnCreateNewCharacterResult/§5=OnCheckDuplicatedIDResult
with an item-id table folded into §5; the brief numbers them the other way and
has no item-id-table deliverable, because that table is already §1.4, confirmed
present at `derivation.md:155`). No line inside §1 (`derivation.md:28`), §2
(`:199`), or §3 (`:427`) was touched — `git diff ... | grep '^-'` returns only
the 4 lines of the old placeholder comment. This is a correct, documented
resolution of a real cross-task inconsistency, not scope creep.

## §4 — `OnCheckDuplicatedIDResult` (brief Step 1) — PASS

`derivation.md:511-669`. Per-version address + decode order + branch
enumeration given for v83 (`0x7d768a`), v87 (`0x82e12c`), v92 (`0x756370`,
named), v95 (`0x777e40`, named); v84 VERSION-ABSENT. `decompile_sha256`:
PENDING on all, with the same disclosed reason as Task 1's §2.1 (function
absent from `ida-exports/<version>.json`) — verified: `grep` for
`CUICharacterSaleDlg::OnCheckDuplicatedIDResult` against
`docs/packets/ida-exports/gms_v83.json`/`gms_v87.json`/`gms_v92.json`/`gms_v95.json`
finds no such key (the report's claim, taken on trust for the negative-grep but
consistent with Task 1's identical finding pattern for the same function
family).

Opcode cross-check against registry (independently verified this pass):

- v83: `docs/packets/registry/gms_v83.yaml:1805-1808` — `MAPLELIFE_RESULT`
  opcode 349, `fname: CUICharacterSaleDlg::OnCheckDuplicatedIDResult`. Matches.
- v87: `docs/packets/registry/gms_v87.yaml:1911-1914` — opcode 370. Matches.
- v92: `docs/packets/registry/gms_v92.yaml:2105-2110` — opcode 404. Matches
  (registry row itself carries a "not IDA-confirmed by discover-ops" note;
  Task 2's own decompile is the first IDA confirmation of this opcode).
- v95: `docs/packets/registry/gms_v95.yaml:2121-2124` — opcode 413. Matches.

No layout/arm divergence found across present versions, so correctly no
`MajorAtLeast` predicate is introduced (the brief only requires one where a
difference exists). Branch shape matches the cited sibling precedent
(`libs/atlas-packet/cash/clientbound/check_name_change.go:22-53`) in structure
only — not copied as a value; the arm strings/StringPool IDs differ and are
recorded per version, so this is not "value invented from the sibling," it is
"documented as structurally analogous."

## §5 — `OnCreateNewCharacterResult` (brief Step 2) — PASS

`derivation.md:671-758`. Closed three-arm enumeration (SUCCESS / duplicate-
at-submit / unknown-error-with-param) is described as read off an "exact-
equality switch on `nType`, not a range/signed test" with per-arm `nType`
literal, `nParam` semantics, and StringPool string ID recorded per version
(v83: 52/54; v87: 54/56; v92: 55/57; v95: 56/58 — each version's literal
explicitly called out, not silently reused from v83). This reads as a
compiled-branch enumeration (decoded field list, then per-arm literal +
UI-string citation), not an assumed/generic set — satisfies the brief's
"full error-code enumeration is the deliverable" bar. Opcode cross-check:
v83 350, v87 371, v92 405, v95 414 — all match the same registry rows
verified in §4 above (each version pairs `MAPLELIFE_RESULT`/`MAPLELIFE_ERROR`
at N/N+1).

## §6 — duplicate-probe sender + C1 (brief Step 3) — PASS, with one caveat noted below

`derivation.md:760-865` (table at `:769-778`). Per-version address, opcode,
body encode order (`EncodeStr sCharName` only, all versions), and
`CHECK_CHAR_NAME`(21) collision answer given for all five versions, including
a positive VERSION-ABSENT finding for v84 (extends §2.0/§4.3, not a bare
silence).

- v83 opcode 256: registry `gms_v83.yaml` has **no** `opcode: 256` row at all
  (independently confirmed — `grep -n opcode: 256 gms_v83.yaml` returns
  nothing), consistent with the report's claim this is a previously-
  undiscovered opcode.
- v92 opcode 301: registry's only `opcode: 301` row is the unrelated
  **clientbound** `MOB_ATTACKED_BY_MOB` (`gms_v92.yaml:1606-1607`,
  independently confirmed) — the derivation correctly identifies this as not
  a collision (separate serverbound/clientbound opcode spaces) rather than
  silently ignoring the coincidence.
- v87 opcode 270 / v95 opcode 311: exactly match the existing
  `JMS_SLASH_COMMAND` serverbound rows at `gms_v87.yaml:3651-3655` and
  `gms_v95.yaml:4100-4104`, both `fname: CUICharacterSaleDlg::SendCheckDuplicateIDPacket` — confirmed.
- `status.json` cross-check (`docs/packets/audits/status.json:39536-39584`,
  the `JMS_SLASH_COMMAND` row): gms_v83/gms_v84/gms_v92 all carry
  `"state": "n-a", "opcode": -1` exactly as the brief described, and
  gms_v87/gms_v95/jms_v185 carry `"state": "incomplete"` with the opcode
  values the derivation reproduces (270/311/271). Confirms the brief's
  premise and the report's framing of v83/v92 as previously-blank rows.
- CHECK_CHAR_NAME(21) claim: `services/atlas-configurations/seed-data/templates/template_gms_83_1.json:161-169`
  binds `opCode: "0x15"` (=21) to `handler: CashShopCheckNameChangeHandle`,
  `fname: CLogin::SendCheckDuplicateIDPacket`, `services: [channel]` — matches
  the brief's characterization exactly and confirms none of 256/270/301/311
  is 21, so the "no collision" answer is correctly grounded.

Routing consequence stated as exactly **(A)** at `derivation.md:832-845` — a
single unambiguous choice among the two the brief specified, correctly
reasoned from the opcode-uniqueness table above.

**Caveat (non-blocking):** the v84 "independent re-confirmation" via the
task-129 IDA note is real — verified at
`docs/packets/ida-exports/gms_v84.json:27616`, key
`CField::OnCharacterSale` — but `derivation.md:598-608`'s quoted text is not a
verbatim excerpt of that note. The actual note reads "*the Vicious Hammer
forwarder is at 0x5443af (IDB symbol CField::OnCharacterSale, but functionally
CField::OnItemUpgrade for v84 — verified by decoder content) ... NOTE: the
IDB-named CField::OnItemUpgrade @0x544395 (this[134]) routes headers 359/360 to
the name/world-transfer dialog (sub_7FD949/sub_7FDA6F), NOT the hammer; header
361 ... is a DEAD opcode there*", whereas the derivation presents an ellipsis-
compressed paraphrase inside quotation marks and prefaces it "*read directly
from the decompile, not authored by this task*" — implying verbatim capture.
The substance matches (same addresses, same routing conclusion), so this is not
a fabricated value, but quoting a paraphrase as verbatim is a documentation-
fidelity defect worth fixing before this note is cited again downstream.
Also worth noting: this note substantiates that `CField::OnCharacterSale`
(0x5443af) is the Vicious Hammer forwarder, not that no other candidate exists
for a Maple-Life dialog on v84 — the actual v84 non-existence claim rests on
the report's own fresh `func_query *CUICharacterSaleDlg*`/`*CharacterSale*`
returning zero matches (a real, freshly-performed derivation, not just a
citation standing in for one), with the task-129 note serving only to rule out
the one candidate that name-search surfaces. This is legitimate corroboration,
correctly scoped — not an over-broad citation used as the whole derivation.

## §6.3 — jms_v185 opcode 271 (brief Step 4) — PASS (correctly escalated, not resolved)

`derivation.md:867-925`. Five distinct searches are documented with their
actual results, including one explicitly marked inconclusive (the unscoped
`insn_query` hit the 200,000-instruction scan cap before covering `.text`,
correctly *not* reported as a negative result). The circumstantial hypothesis
(JMS opcode 271 is likely the same probe, folded into `CLogin` on this build)
is explicitly labelled "a hypothesis, not a derived fact," and the brief's own
"split vs. rename... stop and escalate" instruction is honored: Task 3 is told
not to act on the registry row until this resolves. This is the correct
handling of "never invent" under a genuine architectural obstruction (no
distinct `CUICharacterSaleDlg` class exists in this binary) — recording
"unknown / unverified" rather than picking split-or-rename by inference from
the fname string alone.

## Cross-check: report's headline claims

- "gms_v84 re-confirmed VERSION-ABSENT... a pre-existing task-129 finding" —
  confirmed real and on-point (see caveat above); not a stale/over-broad
  citation, since a fresh zero-match `func_query` was also performed this
  pass.
- v83 probe opcode 256 / v92 probe opcode 301, both currently `-1` in
  `status.json` — confirmed both addresses (`0x7d75ab`, `0x756250`) are cited
  and both opcodes are absent from their registries as previously-undiscovered
  rows, per the independent registry checks above.
- §5's arm enumeration "closed at three arms" — confirmed as a compiled-switch
  read (`nType` exact-equality per version, not a copied/assumed set), with
  per-version literal shifts recorded rather than glossed over.

## Findings

**Blocking:** none.

**Non-blocking:**
1. `derivation.md:598-608` — the task-129 IDA-comment quote is presented in
   quotation marks as if a verbatim excerpt but is an ellipsis-compressed
   paraphrase of the actual note (`docs/packets/ida-exports/gms_v84.json:27616`).
   Substance matches; fix the quote fidelity (either quote verbatim or drop
   the quotation-mark framing) before it's cited again in a later task.

**Not evaluable:**
1. The raw decompiled facts underlying §4–§6 — instruction addresses, the
   compiled branch structure at each address, and the disassembly-level
   evidence for signed-vs-unsigned comparisons — could not be independently
   re-derived by this reviewer, which has no IDA/`func_query`/`insn_query`
   access in this session. Verification in this pass was limited to
   cross-referencing every address/opcode claim against static repo artifacts
   (registry YAMLs, `status.json`, seed templates, `ida-exports/*.json`) that
   were available to read, all of which were internally consistent with the
   new text.

## Verdict rationale

No requirement from the brief was dropped: §4/§5/§6/§6.3 (Steps 1–4) are all
present, in the required shape, with per-version addresses, and every
opcode/registry/status.json/template claim this reviewer could check against a
static artifact matched exactly. §6.3 was correctly left unresolved rather than
guessed, honoring the brief's explicit escalation instruction. The one
non-blocking defect (a paraphrase framed as a verbatim quote) does not affect
any downstream Task 3/5/7/12 consumption of this document's actual findings.
