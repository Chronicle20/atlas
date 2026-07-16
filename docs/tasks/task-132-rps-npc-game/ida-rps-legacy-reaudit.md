# RPS legacy-version re-audit (live IDA) — task-132

Live IDA verification of the RPS (`CRPSGameDlg`) packet family in the four GMS
legacy client versions (v48.1 / v61.1 / v72.1 / v79.1), 2026-07-16.

## ⚠️ Correction notice (read first)
An earlier pass of this audit **mis-identified** the v48 and v61 `CRPSGameDlg::OnPacket`
functions. The **v48 and v61 IDBs ship with the dialog `OnPacket` symbols MISLABELED**
(the functions labeled `CRPSGameDlg::OnPacket` / `CTrunkDlg::OnPacket` are swapped with
other dialogs). The first pass trusted those labels and analyzed the wrong dispatchers,
producing a **false** "older-generation / no-throw / no-ante / shared-opcode / mini-room"
conclusion — all of which is wrong. The real functions were identified by ground truth
(StringPool refs, `ms_RTTI_CRPSGameDlg`, item-decode signature, and the `CField::OnPacket`
dispatch case) and match the addresses the project owner supplied. This document reflects
the corrected findings.

## Corrected verdict: v48/v61 RPS is the standard MODERN NPC-vs-server game
Every audited version (v48/v61/v72/v79) is the same standalone-`CDialog` NPC-vs-server
minigame as v83/v95, with **dedicated** opcodes (no multiplexing) and a serverbound throw.
Structure in all: `Decode1` mode → mode 8 OPEN = `Decode4` ante + participation-fee
StringPool string; mode 11 RESULT via the delegate; mode 13 END = `CWnd::Destroy`;
serverbound = 6-helper send set (sub-ops 0–5) with the player throw at sub-op 1
(`Encode1(1)+Encode1(throw)`). Byte-identical bodies across all versions; only the opcode
shifts.

| Ver | `CRPSGameDlg::OnPacket` (real) | Clientbound recv opcode | Serverbound opcode | Notes |
|---|---|---|---|---|
| v48 | **0x5ADB94** | **237 (0xED)** | **111 (0x6F)** | IDB mislabeled; real fn was unlabeled `sub_5ADB94`. OPEN StringPool(3313). |
| v61 | **0x63BF0E** | **242 (0xF2)** | **124 (0x7C)** | IDB labeled this `CTrunkDlg::OnPacket`. OPEN StringPool(3593). |
| v72 | 0x69c54b | 278 (0x116) | 134 (0x86) | correctly labeled. OPEN StringPool(3650). |
| v79 | 0x6c1d5b | 290 (0x122) | 133 (0x85) | correctly labeled. OPEN StringPool(3654). |
| v83 (ref) | 0x73fff1 | 312 (0x138) | 136 (0x88) | verified baseline. |
| v95 (ref, PDB) | 0x6d9e00 | 371 | — | real symbols; OPEN StringPool(0xE83), `ms_RTTI_CRPSGameDlg`. |

### What the mislabeled functions actually are (do NOT treat as RPS)
- **v48 0x5d5544** (opcodes 234/235): a channel / find-player dialog. NOT RPS.
- **v61 0x607cf7** (opcode 252, labeled `CTrunkDlg::OnPacket`): the trunk dialog. NOT RPS.

## Consequence for the matrix (Path A — full legacy support)
Because v48/v61 RPS is modern and byte-compatible with v83/v72/v79, the existing
`libs/atlas-packet/rps` codec already handles it (no new codec). RPS is now wired into the
legacy tenant templates (48/61/72/79) with the real opcodes — the clientbound `RPSGame`
writer (via `docs/packets/dispatchers/rps_game.yaml` + `operations generate`) and the
serverbound `RPSActionHandle` handler (hand-added with `LoggedInValidator`) — which clears
the template-wiring-gap conflict. Registry corrected to the real opcodes (v48 237/111,
v61 242/124; v72/v79 were already correct). Each legacy cell is then verified via the
shared-codec path (byte-identical to v83), same as v72/v79.

## IDB symbol state
The v48/v61 IDBs were corrected in place: the real dispatchers marked
`CRPSGameDlg_OnPacket_REAL_recv237` (v48 0x5ADB94) / `_recv242` (v61 0x63BF0E), the six real
send helpers per version, and the false-labeled functions flagged `z_MISLABELED_notRPS_*`.
The v72/v79/v83/v95 IDBs were already correct.
