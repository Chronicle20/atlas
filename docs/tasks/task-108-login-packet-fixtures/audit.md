# Task 108 — Execution Audit & Plan Corrections

## Material plan correction: demux op-rows are graded worst-of-candidates

**Discovered during execution (Task 1).** The plan's per-cell recipes and its
§C "verdict-clean rule" misdiagnose how the coverage matrix grades the login
op-rows. Verified against `tools/packet-audit/internal/matrix/{build,grade}.go`:

1. **Op-rows aggregate all writers sharing the op's base FName**
   (`worstCandidateCell`, build.go:261) and take the **worst** by state severity.
   The login op-rows are demux families:
   - `LOGIN_STATUS` → `CLogin::OnCheckPasswordResult` → **4 writers**:
     `AuthLoginFailed`, `AuthSuccess`, `AuthPermanentBan`, `AuthTemporaryBan`.
   - `PICK_ALL_CHAR` / `VIEW_ALL_WITH_PIC` / `VIEW_ALL_PIC_REGISTER`
     → `CLogin::SendSelectCharPacketByVAC` → **3 writers** (the three
     `AllCharacterListSelect*`). All three op-rows grade worst-of the same 3.
   - `CHAR_SELECT` / `REGISTER_PIC` / `CHAR_SELECT_WITH_PIC`
     → `CLogin::SendSelectCharPacket` → **3 writers** (the three
     `CharacterSelect*`). All three op-rows grade worst-of the same 3.

   **Consequence:** ALL writers in a family must individually grade Verified
   before ANY of the family's op-rows flips. The displayed cell *note* is the
   worst arm's note, not necessarily the writer the cell is named after. The
   plan named cells after one writer and prescribed fixing that writer; the real
   blocker is often a *sibling* arm.

2. **For a tier-1 / FlatInvalid writer the diff verdict is ADVISORY**
   (grade.go:195-211). Such a cell promotes on **marker + fresh evidence**, NOT
   on "FlatInvalid:false and every Verdict==0". The plan §C verdict-clean rule is
   wrong for the branchy writers (`AuthSuccess`, base `CharacterSelect`, etc.),
   whose flat positional diff is a known modeling limitation resolved by
   byte-level fixtures. A real `Verdict != 0` on a **tier-0** (FlatInvalid=false)
   report IS a real wire delta → decompile / fix-first, never pin over it.

### Corrected grading rule applied for the rest of the campaign

> An in-scope login op-row is Verified iff **every** writer sharing the op's base
> FName is individually Verified for that version. Per-writer: tier-0 needs
> `toolPass(verdict 0) + marker`; tier-1 (FlatInvalid OR tier1 packet) needs
> `marker + fresh evidence` (verdict advisory). `matrix --check` (baseline exit 0,
> 0 conflicts, 0 login lines) is the arbiter.

### Task-structure deviations from plan.md

- **Plan Tasks 3 + 4 merged** into one "v84 `SendSelectCharPacket` family" unit:
  the three CHAR_SELECT/REGISTER_PIC/CHAR_SELECT_WITH_PIC op-rows are worst-of the
  same 3 writers, so none flips until all three writers are verified. RegisterPic
  and WithPic already carry v84 marker+evidence; only the base `CharacterSelect`
  needs work (byte-fixture v84 row + marker + evidence).
- **Cell #1 (AuthLoginFailed gms_v83)** was promoted by pinning a fresh
  `AuthSuccess` **gms_v83** evidence record (the FlatInvalid sibling arm that
  was dragging the op-row), NOT by touching AuthLoginFailed. AuthSuccess maps to
  the base `CLogin::OnCheckPasswordResult` key (mirrors the existing gms_v84
  AuthSuccess record). Committed `925554c`.

## Export / fname deviations (grounded, documented)

- **v84 `ServerStatus` annotation fix (controller-verified, not a wire delta).**
  `CLogin::OnCheckUserLimitResult` reads 2×`Decode1` on BOTH v83 (@0x5f92ae) and
  v84 (@0x60e275); atlas writes one `WriteShort` = 2 bytes → wire-equivalent. The
  v84 export entry listed two literal `Decode1` ops (→ "width mismatch"); the
  verified v83 entry collapses them to one `Decode2` with a wire-equivalence
  comment. Fixed the v84 entry to mirror v83. Grounded by decompiling both.
- **v84 `ServerListRequest` — same inline as v83.** No discrete `ChangeStep`/
  `ChangeStepImmediate` symbol in v84; the bodyless `COutPacket(4)+SendPacket`
  lives in `sub_609165` (the v84 step-machine analog), in the `*(CWvsContext+8228)
  ==1` block. Spliced under the canonical name with the real v84 address + note;
  controller re-decompiled and confirmed bodyless opcode-4.
- **v83 `ServerListRequest` — `CLogin::ChangeStepImmediate` does not exist as a
  discrete symbol in the v83 IDB** (unlike v87/v95/jms). The immediate
  server-list-request send is **inlined into `CLogin::ChangeStep` @0x5f53c0**
  (`m_nBaseStep==1` block: `COutPacket(opcode 4)+SendPacket`, no Encode calls →
  bodyless). The export key was spliced under the canonical matrix logical name
  `CLogin::ChangeStepImmediate` (the `candidatesFromFName` mapping at run.go:546)
  with the **real** inline address and a `note` documenting the inline.
  Independently re-decompiled and confirmed (controller). This is a grounded
  unblock per "No Deferring Producible Work" — the fname was *found and verified*
  inlined, not fabricated — NOT a faked-hash escalation case.

## Wire deltas found

- **none through v83.** v83 `AllCharacterListRequest` (`SendViewAllCharPacket`
  @0x5fac34) sends opcodes 0xC/0xD with zero Encode calls → bodyless on v83; the
  atlas decoder gates all 5 field reads behind `MajorAtLeast(87)` → reads nothing
  on v83. The report's 5 "atlas: extra" verdict-2 rows are a flat-diff modeling
  artifact (analyzer models the v87 fields statically), confirmed by decompile —
  NOT a real over-read. Tier-1 advisory → promoted via marker + fresh evidence.
- jms `ServerListEnd` remains the prime real-delta suspect (Task 8).
