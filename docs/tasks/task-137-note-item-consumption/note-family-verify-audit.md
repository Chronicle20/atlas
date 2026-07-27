# Note-family verify audit (post-live-test, task-137 expanded scope)

Triggered by live testing on `atlas-pr-1105` (GMS **v83.1**, tenant `bbbcc406-…`, Atlas=char1 sender, Chronicle=char2 receiver). Two live bugs whose root causes exposed that the note-family matrix ✅ cells are **not client-validated** (round-trip fixtures against our own assumptions). User elected a full 9-version note-family re-verify.

## Confirmed root causes (grounded)

### Bug 2 — OK/discard crashes client to login + note not cleared (serverbound codec)
v83 `CMemoListDlg::SetRet` @0x64aa57 (decompiled live) writes the discard body:
```
mode=1, totalCount(u8), specialCount(u8), emptySlots(u8),
per entry: normal -> id(int32), flag(u8)
           special (flag==3, slot available) -> id(int32), flag(u8), extra(int32)
           special (flag==3, no slot) -> OMITTED from wire (client shows inbox-full notice)
```
Our `libs/atlas-packet/note/serverbound/operation_discard.go` decode reads `count, emptySlots` for non-JMS (`specialCount` gated to `isJMS` only). **It omits `specialCount` for GMS** → 1-byte misalignment. Live proof: real note id=1, emptySlots=24(0x18) → wire `01 00 18 01 00 00 00 ..` → misread `id=ReadUint32()=0x00000118=280`. `note_operation.go` DISCARD arm: `GetById(280)` → 404 → `session.Destroy(s)` → **client dropped to login; note never discarded** (atlas-notes log: repeated `GET /api/notes/280` → record not found → 500).

Fix: implement the uniform header (`specialCount` for ALL versions) + per-version special-flag/extra, replacing the `isJMS` gate. Also harden the DISCARD handler: a decode/lookup miss or flag mismatch must NOT `Destroy(s)` (over-brutal; a legitimate flow shouldn't nuke the session).

### Bug 1 — display shows wrong "received"/"gained fame" text (flag value, not codec)
v83 per-note decode `sub_4E4ADB` reads `id(int32), sender(str), msg(str), time(8), flag(u8)` — **matches our clientbound `NoteEntry.Encode` exactly**. So the display field order is correct. The bug is the **flag VALUE**: we send `flag=1` (buildNoteSendSaga + CreateNotePayload + design §2.4 "flag stays 1", never validated). The special/gift flag is 2 (v48/61) or 3 (v72+), NOT 1; v48 fires a `0x66` follow-up for `flag==1`. `flag=1` is a non-plain memo type (renders as fame/popularity). Correct plain-note flag TBD by audit (expected 0).

## Per-version serverbound discard spec (design §1.5 + v83 live; VALIDATE each vs client)
| ver | note-op | SetRet | special flag | extra field(s) |
|---|---|---|---|---|
| v48 | 0x65 | 0x534dc4 | 2 | reward i32 (+ 0x66 follow-up for flag==1) |
| v61 | 0x77 | 0x5ad50c | 2 | itemId i32 |
| v72 | 0x81 | 0x5fb443 | 3 | mesos i32 |
| v79 | 0x80 | 0x619f32 | 3 | value i32 |
| v83 | 0x83 | 0x64aa57 | 3 | 1×i32 (LIVE-CONFIRMED) |
| v84 | — | — | 3? | 1×i32? (confirm) |
| v87 | — | — | 3? | 1×i32? (confirm) |
| v95 | — | — | 3? | 1×i32? (confirm) |
| jms | 0x86 | 0x6c2d43 | 3 | 2×i32 (Task-17 derived) |

Header `specialCount` present in ALL (v83 confirmed; design §1.5). Wire-entry count = totalCount − max(0, specialCount − emptySlots) (special entries omitted when slot budget exhausted).

## Plan
1. DERIVE/validate per version (IDA, read-only): SetRet header+special shape (confirm specialCount in all; per-version flag/extra), OnMemoResult display order + the correct plain-note flag value. IDBs: v48=0bb5f11a v61=965202bf v72=90e36cb0 v79=9a7d3642 v83=ce4ff298(done) v84=79511a2a v87=81f32170 v95=e4abcb98 jms=3c4bb8b1.
2. FIX serverbound `operation_discard.go`: per-version header (specialCount for all) + special-flag/extra; fixtures from REAL client bytes (not round-trip).
3. FIX bug 1 flag value in `buildNoteSendSaga` + `CreateNotePayload` sites (+ verify clientbound display renders plain with the correct flag).
4. HARDEN `note_operation.go` DISCARD arm: no `Destroy(s)` on decode/lookup miss.
5. RE-VERIFY each note-family cell (packet-verifier, fixtures from client bytes) + regenerate matrix; confirm no non-note regression.
6. Full gate + push to PR #1105.
