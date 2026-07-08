# Task 20 seed templates — per-version verification record

Records the grounding for every mini-game mode value and opcode seeded into the
six tenant templates (`template_gms_{83,84,87,92,95}_1.json`, `template_jms_185_1.json`).
Registry-note style banner for the one **UNVERIFIED** version (gms_v92).

## Mode enum (MEMORY_GAME sub-modes) — verification per version

| Version | Status | Evidence (this session unless noted) |
|---|---|---|
| gms_v83 | **IDA-verified** | ida-notes §G5: `COmokDlg::OnPacket` 0x6e37eb / `CMemoryGameDlg::OnPacket` 0x64db30 |
| gms_v84 | **IDA-verified** | port 13345: `CMemoryGameDlg::SendTurnUpCard` 0x664afc encodes 0x44 (FIP_CARD=68); `COmokDlg::OnRetreatRequest` 0x6fb416 replies 0x37 (RETREAT_ANSWER=55); base dispatcher `sub_673DB5` VISIT=4/CHAT=6/EXIT=10 → **= v83** |
| gms_v87 | **IDA-verified** | port 13343: `COmokDlg::OnPacket` 0x721300 + `CMemoryGameDlg::OnPacket` 0x687a37 full switches → **= v83** |
| gms_v92 | **UNVERIFIED (derived)** | **No v92 IDB exists** (not in the loaded instance set; "v92 mount-food parked, no v92 IDB" in project memory) and v92 is **outside `matrix.VersionKeys`** so the packet-audit tool does not manage it. Mode values copied from the GMS v83 enum, bracketed by identical IDA-verified GMS neighbours (v87 and v95 both = v83). Opcodes from the csv-import CSV (below). **Not IDA-confirmed for v92 specifically.** |
| gms_v95 | **IDA-verified** | ida-notes §G5: `COmokDlg::OnPacket` 0x688b70 / `CMemoryGameDlg::OnPacket` 0x634020 → = v83 |
| jms_v185 | **IDA-verified** | port 13344: `COmokDlg::OnPacket` 0x72ad22 + `CMemoryGameDlg::OnPacket` 0x6c792b full switches; serverbound-only sends `SendClaimGiveUp` 0x72ff50 (FORFEIT=0x31=49), `OnClickEndButton` 0x7302fb (EXIT_AFTER_GAME=0x35=53, CANCEL=0x36=54), `OnClickBanButton` 0x73027d (EXPEL=0x39=57). **Uniform −3 shift vs v83.** |

### jms −3 shift (correction to the task-20 brief)

The brief assumed jms sub-modes follow v83. They do **not**: the entire trade/store/
merchant/game sub-mode block (mode ≥ 14) is shifted **−3** in jms. This was first
flagged by the pre-existing jms handler ops (`MERCHANT_BUY`=31 vs v83 34; already-shifted
`UPDATE_MERCHANT`=22 vs 25) and then confirmed byte-for-byte on every game mode via the
jms IDB. Copying v83 blind would have been a silent wire error. Base lifecycle modes
(INVITE=2, INVITE_RESULT=3, ENTER/VISIT=4, ENTER_RESULT=5, CHAT=6, AVATAR=9, LEAVE/EXIT=10,
CREATE=0) are **unshifted** in jms (base dispatcher `CMiniRoomBaseDlg::OnPacketBase` 0x6da198).

| MEMORY_GAME key | gms_v83/84/87/92/95 | jms_v185 (−3) |
|---|---|---|
| ASK_TIE | 50 | 47 |
| TIE_ANSWER | 51 | 48 |
| FORFEIT (handler only) | 52 | 49 |
| ASK_RETREAT | 54 | 51 |
| RETREAT_ANSWER | 55 | 52 |
| EXIT_AFTER_GAME (handler only) | 56 | 53 |
| CANCEL_EXIT_AFTER_GAME (handler only) | 57 | 54 |
| READY | 58 | 55 |
| UNREADY | 59 | 56 |
| EXPEL (handler only) | 60 | 57 |
| START | 61 | 58 |
| RESULT (writer only) | 62 | 59 |
| SKIP | 63 | 60 |
| MOVE_STONE | 64 | 61 |
| FIP_CARD (typo load-bearing) | 68 | 65 |

## Opcodes (source of truth = `docs/packets/registry/*.yaml`; v92 = CSV)

| Version | SB handler `PLAYER_INTERACTION` | balloon `UPDATE_CHAR_BOX` | CB writer `PLAYER_INTERACTION` | Source |
|---|---|---|---|---|
| gms_v83 | 0x7B | 0xA5 | 0x13A | registry (existing template) |
| gms_v84 | 0x7D | 0xA8 | 0x141 | registry (IDA, task-100) |
| gms_v87 | 0x81 | 0xB0 | 0x14B | registry (IDA) |
| gms_v92 | 0x8D | 0xB6 | 0x16D | **CSV `MapleStory Ops` GMS v92 column** (csv-import; no registry/IDB). Cross-validated: the CSV's v83/v87/v95/jms columns match the IDA registries exactly, so the v92 column is trustworthy by consistency — but **not IDA-verified for v92**. |
| gms_v95 | 0x90 | 0xB8 | 0x175 | registry (IDA) |
| jms_v185 | 0x7C | 0xA3 | 0x153 | registry (IDA) |

All balloon opcodes matched the brief's candidates exactly (0xA5/0xA8/0xB0/0xB8/0xA3),
and every registry-sourced opcode was cross-checked against the CSV.

## Where each piece is seeded (mechanism)

- **Serverbound handler** operations (`CharacterInteractionHandle`): hand-set in the six
  templates. CREATE/VISIT/CHAT/EXIT + the 14 serverbound MEMORY_GAME ops. All six carry
  `"validator": "LoggedInValidator"`.
- **Clientbound writer** operations (`CharacterInteraction`, 11 clientbound arms incl.
  RESULT): source of truth is `docs/packets/dispatchers/character_interaction.yaml`;
  generated into the 5 matrix templates by `packet-audit operations`. gms_v92 is outside
  the matrix, so its writer rows are hand-set from the GMS values.
- **Balloon writer** (`MiniRoom`, `UPDATE_CHAR_BOX`): hand-set in all six templates.

`packet-audit operations --check` → exit 0. `jq` parse of all templates → OK.
