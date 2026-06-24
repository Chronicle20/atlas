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

## Wire deltas found

(none yet — updated as the campaign proceeds; jms `ServerListEnd` is the prime
real-delta suspect per plan §7)
