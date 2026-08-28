# Review — Task 1 (pin the trailing 4-byte ring field from IDA)

Range reviewed: `d6e57eecc..c134a7fc9` (1 commit, `c134a7fc9`).
Inputs: `.superpowers/sdd/plan/task-1-brief.md`, `.superpowers/sdd/plan/task-1-report.md`,
`.superpowers/sdd/plan/review-d6e57eecc..c134a7fc9.diff`, plus a direct read of the produced
artifact `docs/tasks/task-269-ring-pair-behavior/ring-field-derivation.md`.

## Diff scope check

`git diff --stat d6e57eecc..c134a7fc9` → one file, `ring-field-derivation.md`, 356 insertions,
0 deletions. `git diff --stat -- docs/packets/ida-exports/` over the same range → empty. Matches
the brief's Step 6 constraint (exports must not be hand-edited) and the report's claim that
nothing else changed. No code touched, consistent with "Task 1 is a pure IDA derivation."

## Spec compliance — brief's seven steps

1. **Resolve IDA sessions** — report names `GMS_v95.0_U_DEVM.exe.i64` → `ecc757f4` and
   `MapleStory_dump.exe.i64` → `754107bf`, matched by binary name per
   `docs/reverse-engineering.md:13-16`, which documents exactly this procedure ("Resolve the
   session from `idb_list` by binary **name**... port-based selection is dead"). PASS on
   procedure; the actual session ids are IDA-session state I have no tool access to confirm in
   this environment (see Not evaluable).
2. **Decompile the three v95 registrars** — report explains the brief's literal symbol names
   didn't resolve and pivots to `func_query {name_regex: "RecordAdd"}`, a disclosed and
   reasonable deviation, not a silent skip. Decompile output is quoted in the artifact for all
   three (`ring-field-derivation.md:133-146` couple, `:216-222` friend, `:241-250` marriage).
   Not independently re-run (no IDA tool access this session).
3. **Cross-check the caller on v83** — `CUserRemote::Init` @0x97f55d and `OnAvatarModified`
   @0x98367e both quoted with raw disassembly (`ring-field-derivation.md:65-120`). The brief's
   "run `xrefs_to_field`" step is explicitly addressed: the report explains there is no member
   store to target (the `Decode4` result is pushed straight into the registrar call, never
   written to a `CUser` field) and that this absence is itself evidence, then reports it ran
   `xrefs_to_field` anyway on three plausible fields, all returning "No cross-references" — a
   documented, non-silent handling of an instruction that didn't apply as literally written.
4. **Marriage arm first field** — addressed directly: `OnMarriageRecordAdd`'s first parameter is
   matched against another `CUser`'s `m_dwMarriagePairCharacterID`
   (`ring-field-derivation.md:242-243, 252-255`), and `SetMarriagePairCharacterID` is used to
   pin which of bride/groom it is (`:257-274`). Explicitly refutes `design.md`'s `MarriageId`
   name while confirming the offset — verified below.
5. **Derivation record with per-block tables + one-sentence verdict + explicit wrong-comment
   call** — present. Each of the three blocks has a `field name | wire width | IDA address |
   what reads it` table (`:203-208`, `:229-234`, `:290-294`). The `## Verdict` section
   (`:298-322`) states both required verdicts decisively, in the required one-sentence-first
   form, with no hedging. `## Defect in a checked-in export` (`:324-356`) names `gms_v83.json`
   as wrong and `gms_jms_185.json`'s `Init` entry as right, exactly as Step 5 requires.
6. **Do not hand-edit the export** — confirmed by the diff stat above; only the new derivation
   file changed.
7. **Commit** — `c134a7fc9`, message `derive(task-269): pin the trailing 4-byte ring field from
   IDA`, matches the brief's suggested message closely enough (imperative, task-269 scoped).

All seven steps: **satisfied**, with the caveat on steps 1-4 noted under Not evaluable (the
underlying IDA session content is unverifiable from this environment; what's independently
checkable — the export-file quotes — is fully confirmed, see below).

## Correctness checks I could perform directly

**Export quotes verified byte-for-byte against the checked-in JSON** (this is the part of the
"no invented values" constraint I can mechanically confirm without IDA access):

- `docs/packets/ida-exports/gms_v83.json` `functions["CUserRemote::OnAvatarModified"]`
  (`address: "0x98367e"`) — comments are exactly `"liCoupleItemSN (8 bytes) + liPairItemSN (8
  bytes) + dwPairCharacterId (4 bytes)"` and `"liFriendshipItemSN (8 bytes) +
  liFriendshipPairItemSN (8 bytes) + dwFriendCharacterId (4 bytes)"`, and the marriage triple is
  exactly `dwMarriageCharacterID` / `dwMarriagePairCharacterID` / `nWeddingRingID` — matches the
  derivation doc's quotes verbatim.
- `docs/packets/ida-exports/gms_v87.json` (`0xa090f4`) and `gms_v95.json` (`0x954110`) carry the
  identical wrong wording — matches the doc's claim that all three (v83/v87/v95) share the
  defect.
- `docs/packets/ida-exports/gms_jms_185.json` `functions["CUserRemote::OnAvatarModified"]`
  (`0xa57221`) has `"pair characterId (per entry)"` and `"friendship pair characterId (per
  entry)"` — matches. The same file's `functions["CUserRemote::Init"]` (`0xa52876`) has
  `"couple-ring itemId (per entry)"` and `"friendship-ring itemId (per entry)"` — matches the
  doc's claim that this file contradicts itself and that `Init` carries the correct wording.

Every export-level factual claim in the derivation doc that I could mechanically check against
the checked-in JSON is accurate, including the more subtle claim (gms_jms_185.json contradicting
itself between two functions).

**Internal arithmetic consistency** (a proxy for whether the disassembly transcription is real
rather than fabricated, since I cannot re-run the decompiler): the v83 raw-offset math is
self-consistent across independent representations in the same document —
`this + 7952` (`ring-field-derivation.md:71`) = `0x1F10` as claimed; `this + 7960` = `0x1F18`;
`this + 8072` = `0x1F88`; `this + 8080` = `0x1F90` — all check out by hand. The `dwCharacterID`
member is cited twice via two different indexing conventions in the same doc — `a2[1130]`
(int-indexed, `:161`) and `*(v14 + 4520)` (byte-indexed, `:160`) — and `1130 * 4 = 4520 = 0x11A8`,
matching the claimed offset both places. The marriage members `this + 1997/1998` (`:113,115`,
int-indexed) convert to `0x1F34`/`0x1F38` (`1997*4=7988=0x1F34`), matching the byte-offset
citations used elsewhere in the same doc for the identical members. This cross-representation
consistency is evidence against fabrication but is not itself proof the underlying decompile
lines are real — see Not evaluable.

**`design.md` cross-check**: `design.md:217` reads `MarriageId uint32 // -> CUser+0x1F34` — the
derivation doc's claim that `design.md` §3.1 names the offset correctly but the field name wrong
is accurate; the offset matches, and the name `MarriageId` is exactly what the derivation
verdict refutes.

**Structural requirement (extensibility for Task 3)**: the doc is sectioned per-arm (`## Block 1
— couple ring`, `## Block 2 — friendship ring`, `## Block 3 — marriage`) with a `## Verdict` and
`## Defect...` section after, matching the brief's Step 5 structure and the referenced pattern
document's style. Task 3 can append a new `##` section without touching the verdict tables.

**Path hygiene**: `grep -n "/home/\|/Users/\|C:\\\\"` over the new file returned no matches — no
literal absolute paths committed.

## Not evaluable

This review environment has no `mcp__ida-pro__*` tools available, so I could not independently
call `idb_list`, `decompile`, `func_query`, `xrefs_to`, or `insn_query` to re-derive any of the
report's IDA-sourced claims. Concretely, not evaluable:

- Whether the two session ids (`ecc757f4` for v95, `754107bf` for v83) are real, currently-open
  IDA sessions.
- Whether every quoted decompile/disassembly line (all of Blocks 1-3, the wire block, the
  producer-side `SetPairCharacterID`/`SetMarriagePairCharacterID` functions) is a verbatim
  transcription of what IDA actually shows, rather than a plausible reconstruction. I could only
  corroborate this indirectly via the internal arithmetic consistency described above and via the
  fully-matching export-file quotes (which are independently checkable and all matched).
- Whether `xrefs_to`/`callees` on `CUser::SetCoupleItemEffect` @0x8f05d0 and
  `CUser::SetWeddingRingEffect` @0x8f18e0 actually return only `Format`/`LoadLayer` as claimed
  (the "no character lookup" argument central to the couple/friendship verdict).

Per the task instructions, I spot-checked against the checked-in exports instead; every claim
that touches the exports (five files) matched exactly, with no discrepancy found.

## Verdict

No blocking findings. The derivation is decisive on both required questions, names the wrong
export comment explicitly (and catches an internal self-contradiction in `gms_jms_185.json`
beyond what the brief asked for), touches no export file, uses no absolute paths, and is
structured for Task 3's append. Every claim I could mechanically verify (all export-file quotes,
the `design.md` offset, the internal offset arithmetic) checked out exactly. The IDA
decompile/disassembly content itself is unverifiable from this environment and is reported under
Not evaluable, not folded into the approval.
