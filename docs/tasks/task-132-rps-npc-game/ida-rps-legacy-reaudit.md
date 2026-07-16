# RPS legacy-version re-audit (live IDA) — task-132

Live IDA verification of the RPS (`CRPSGameDlg`) packet family in the four GMS
legacy client versions, performed 2026-07-16 against the connected ida-pro MCP
(192.168.20.3:13337). Supersedes the earlier export-derived disposition, which
was based only on the checked-in `docs/packets/ida-exports/gms_v{48,61,72,79}.json`
and got two things wrong (v48 clientbound; all-four serverbound) and misread
v72/v79 as "only mode 8".

Instances matched by binary name: v48→`GMS_v48_1_DEVM.exe`, v61→`GMS_v61.1_U_DEVM.exe`,
v72→`GMS_v72.1_U_DEVM.exe`, v79→`GMS_v79_1_DEVM.exe`, v83 reference→`MapleStory_dump.exe`.

## Reference (v83)
- `CRPSGameDlg::OnPacket` @0x73fff1 — modern dispatcher: 8=OPEN / 11=RESULT (delegate `sub_74024B`) / 13=END.
- Serverbound opcode **0x88 (136)**: `OnBtStart` @0x7403d0 (`COutPacket(0x88);Encode1(0)`), `SendSelection` @0x7405a0 (`COutPacket(0x88);Encode1(1);Encode1(throw)`).

## Per-version findings (all evidence live)

| Ver | Clientbound `CRPSGameDlg::OnPacket` | Wired into inbound dispatch? | Mode generation | Serverbound send path | Serverbound opcode |
|---|---|---|---|---|---|
| v48 | @0x5d5544 (2-arg, opcode-guarded) | **YES** — sole xref `CField::OnPacket` @0x4c66f2 (call site 0x4c6a30) | OLD gen: clientbound opcodes 234/235, sub-modes 27–31 / 32–35 (NOT 8/11/13) | **EXISTS** — `sub_5D53DD` (start), `sub_5D5442` (continue), `sub_5D443A` (button vtable @0x7a1188) | **0x33 (51)** |
| v61 | @0x607cf7 | **YES** — `CField::OnPacket` @0x4e9ea3 (site 0x4ea23e) | OLD gen: modes 8/23/24/25/26/27 + default(18); DecodeStr invites, CUIFadeYesNo (NOT 8/11/13) | **EXISTS** — `sub_607C9C` (sub-op 7), called from `sub_6075A7`/`sub_607743` + OnPacket | **0x3D (61)** |
| v72 | @0x69c54b (size 0x25a) | **YES** — `CField::OnPacket` @0x515879 (site 0x515ca9) | **MODERN** — byte-identical to v83: 8=OPEN / 9–12 delegate `sub_69C7A5` / 13=END. (export "only mode 8" was a MISREAD) | **EXISTS** — dialog input handler `sub_69B69A` (`COutPacket(134);Encode1(2)`) | **0x86 (134)** |
| v79 | @0x6c1d5b (size 0x25a) | **YES** — `CField::OnPacket` @0x51c90f (site 0x51cd61) | **MODERN** — byte-identical to v83: 8=OPEN / 9–12 delegate `sub_6C1FB5` / 13=END. (export "only mode 8" was a MISREAD) | **EXISTS** — dialog input handler `sub_6C0EAA` (`COutPacket(133);Encode1(2)`) | **0x85 (133)** |

## Resolution of the two open questions
1. **v48 clientbound `n-a` vs `incomplete`:** `CRPSGameDlg::OnPacket` @0x5d5544 is **wired** — its only xref is a code call from `CField::OnPacket` (the field inbound-packet dispatcher). Reachable, not dead code. → v48 clientbound is **`incomplete`, not `n-a`** (older 234/235 form).
2. **Serverbound `n-a` for all four:** every legacy IDB builds and sends a real RPS `COutPacket` (v48 0x33, v61 0x3D, v72 0x86, v79 0x85) via `CClientSocket::SendPacket`. → serverbound `n-a` is a **false disposition in all four**; each should be **`incomplete`**.

## Matrix cells that should change (5)
- `rps/serverbound/RpsOperation` (RPS_ACTION) — **gms_v48, gms_v61, gms_v72, gms_v79**: `n-a` → `incomplete` (send opcodes 0x33/0x3D/0x86/0x85).
- `rps/clientbound/RpsEnd` (RPS_GAME) — **gms_v48**: `n-a` → `incomplete` (OnPacket present + wired; clientbound opcode 234/235).
- v61/v72/v79 clientbound `incomplete` are **already correct**.

## Promotion opportunity (separate from the correction)
v72 and v79 clientbound `CRPSGameDlg::OnPacket` are **byte-identical to the already-verified v83** dispatcher (only the opcode shifts). Those two cells are strong candidates for full `verified` promotion via the shared-codec wrap+verify path (docs/packets/audits/VERIFYING_A_PACKET.md), rather than merely `incomplete`. This is a `packet-verifier` fan-out, not a disposition edit.

## IDB symbols renamed (persisted to the four .i64 files)
- **v48:** `CRPSGameDlg__SendStart_op33_17` (0x5d53dd), `CRPSGameDlg__PromptContinue_Send_op33_18` (0x5d5442), `CRPSGameDlg__Send_op33_19` (0x5d443a), `CRPSGameDlg__OpenDialog_mode32` (0x5d5a05)
- **v61:** `CRPSGameDlg__Send_op3D_sub7` (0x607c9c), `CRPSGameDlg__OnOpen_mode8_delegate` (0x607831), `CRPSGameDlg__AllocDialog` (0x6064ec)
- **v72:** `CRPSGameDlg__UpdateInput_Send_op86_sub2` (0x69b69a), `CRPSGameDlg__OnResult_delegate` (0x69c7a5)
- **v79:** `CRPSGameDlg__UpdateInput_Send_op85_sub2` (0x6c0eaa), `CRPSGameDlg__OnResult_delegate` (0x6c1fb5)
