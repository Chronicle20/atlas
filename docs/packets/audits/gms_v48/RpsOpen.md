# RpsOpen (← `CRPSGameDlg::OnPacket#OPEN`)

- **IDA:** 0x5adc8f
- **Atlas file:** `libs/atlas-packet/rps/clientbound/operation.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (real dispatcher sub_5ADB94 @0x5adba6, live IDA re-audit port 13337 2026-07-16; the v48 IDB mislabels this function -- ground truth per docs/tasks/task-132-rps-npc-game/ida-rps-legacy-reaudit.md; case 8 = OPEN)` | ✅ |  |
| 1 | int32 | int32 `ante -- participation fee (CInPacket::Decode4 @0x5adc8f; StringPool(3313) notice that follows is a static resource, not a packet field)` | ✅ |  |

