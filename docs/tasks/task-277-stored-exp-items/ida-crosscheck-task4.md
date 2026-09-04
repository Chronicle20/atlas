# IDA cross-check of task-277 Task 4 export entries (stored-EXP items)

**Verdict: MATCH on both columns checked (gms_v95, gms_v72). No mismatches found.**

## Scope

Independent, read-only re-decompile of the two claimed addresses on two columns,
per the audit brief. IDBs available for these two columns per `idb_list`:

- gms_v95 → session `ecc757f4`, `GMS_v95.0_U_DEVM.exe.i64`
- gms_v72 → session `99e435d8`, `GMS_v72.1_U_DEVM.exe.i64`

(Only these two of the eight claimed columns were checked, per the brief's
"spot-check, do not sweep all eight" instruction. gms_v95 is the repo's primary
reference IDB; gms_v72 was chosen as the other readily-available column.)

## gms_v95 — `CWvsContext::SendExpUpItemUseRequest` @ 0x9db1c0

Decompiled directly from session `ecc757f4`. Confirmed:
- Function name: `CWvsContext::SendExpUpItemUseRequest` (thiscall, params include
  `nPOS`, `nItemID`).
- `COutPacket::COutPacket(&oPacket, 181)` — opcode 181 (0xB5), matches claim.
- Encode sequence, in order:
  1. `COutPacket::Encode4(&oPacket, update_time)` where `update_time = get_update_time()` — 4 bytes.
  2. `COutPacket::Encode2(&oPacket, nPOS)` — 2 bytes.
  3. `COutPacket::Encode4(&oPacket, v7)` where `v7 = nItemID` — 4 bytes.
- Followed by `CClientSocket::SendPacket(TSingleton<CClientSocket>::ms_pInstance, &oPacket)`.

Matches claimed sequence: Decode4(updateTime) / Decode2(slot) / Decode4(itemId).

## gms_v95 — `CWvsContext::SendTempExpUseRequest` @ 0x9db430

Decompiled directly from session `ecc757f4`. Confirmed:
- Function name: `CWvsContext::SendTempExpUseRequest` (thiscall, no item/slot params).
- `COutPacket::COutPacket(&oPacket, 182)` — opcode 182 (0xB6), matches claim.
- Encode sequence: exactly one call —
  `COutPacket::Encode4(&oPacket, update_time)` where `update_time = get_update_time()` — 4 bytes.
  No further Encode calls before `CClientSocket::SendPacket`.

Matches claimed sequence: Decode4(updateTime) only, nothing else.

## gms_v72 — `CWvsContext::SendExpUpItemUseRequest` @ 0x90cb20

Decompiled directly from session `99e435d8`. Confirmed:
- Function name: `CWvsContext::SendExpUpItemUseRequest` (thiscall on `CWvsContext`).
- `COutPacket::COutPacket((COutPacket *)v21, 156)` — opcode 156, matches claim.
- Encode sequence, in order:
  1. `COutPacket::Encode4((COutPacket *)v21, v12)` where `v12` is fed from
     `get_update_time(v11)` immediately prior — 4 bytes.
  2. `COutPacket::Encode2((COutPacket *)v21, a2)` — 2 bytes (`a2` is the slot/`nPOS` param).
  3. `COutPacket::Encode4((COutPacket *)v21, v7)` where `v7 = a3` (itemId param) — 4 bytes.
- Followed by `CClientSocket::SendPacket((CClientSocket *)g_pClientSocketInstance, ...)`.

Matches claimed sequence: Decode4(updateTime) / Decode2(slot) / Decode4(itemId).

## gms_v72 — `CWvsContext::SendTempExpUseRequest` @ 0x90cd28

Decompiled directly from session `99e435d8`. Confirmed:
- Function name: `CWvsContext::SendTempExpUseRequest` (thiscall on `CWvsContext`, no params).
- `COutPacket::COutPacket((COutPacket *)v14, 157)` — opcode 157, matches claim.
- Encode sequence: exactly one call —
  `COutPacket::Encode4((COutPacket *)v14, v7)` where `v7` is fed from
  `get_update_time(v6)` immediately prior — 4 bytes.
  No further Encode calls before `CClientSocket::SendPacket`.

Matches claimed sequence: Decode4(updateTime) only, nothing else.

## Conclusion

Both checked columns (gms_v95, gms_v72), both functions each (4 decompiles total),
match the committed export entries exactly: function identity, opcode, and the
Encode4/Encode2/Encode4 vs. single-Encode4 call sequences and widths. No evidence
of fabrication or mis-transcription was found in this spot-check. This does not
constitute proof for the other six columns (gms_v79, gms_v83, gms_v84, gms_v87,
gms_v92, jms_v185), which were not re-verified here per the brief's bounded scope.
