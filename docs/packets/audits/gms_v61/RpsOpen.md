# RpsOpen (← `CRPSGameDlg::OnPacket#OPEN`)

- **IDA:** 0x63c009
- **Atlas file:** `libs/atlas-packet/rps/clientbound/operation.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (real dispatcher @0x63bf0e (labeled CTrunkDlg::OnPacket in the v61 IDB -- confirmed mislabel, ground truth per docs/tasks/task-132-rps-npc-game/ida-rps-legacy-reaudit.md), Decode1 @0x63bf20; case 8 = OPEN; live IDA port 13338 2026-07-16)` | ✅ |  |
| 1 | int32 | int32 `ante -- participation fee (CInPacket::Decode4 @0x63c009; StringPool(3593) notice that follows is a static resource, not a packet field)` | ✅ |  |

