# RpsOperation (← `CRPSGameDlg::OnBtStart`)

- **IDA:** 0x5adf94
- **Atlas file:** `libs/atlas-packet/rps/serverbound/operation.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `sub-op byte (0=OnBtStart @0x5adf94/COutPacket(111); bodyless arm, no further fields -- docs/tasks/task-132-rps-npc-game/ida-rps-legacy-reaudit.md; live IDA port 13337 2026-07-16)` | ✅ |  |

