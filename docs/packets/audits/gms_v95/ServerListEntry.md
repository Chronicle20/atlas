# ServerListEntry (← `CLogin::OnWorldInformation`)

- **IDA:** 0x5da7f0
- **Atlas file:** `../../libs/atlas-packet/login/clientbound/server_list_entry.go`
- **Variant:** GMS/v95
- **Branch depth:** 2
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `nWorldID` | ✅ |  |
| 1 | string | string `sName` | ✅ |  |
| 2 | byte | byte `nWorldState` | ✅ |  |
| 3 | string | string `sWorldEventDesc` | ✅ |  |
| 4 | int16 | int16 `nWorldEventEXP_WSE` | ✅ |  |
| 5 | int16 | int16 `nWorldEventDrop_WSE` | ✅ |  |
| 6 | byte | byte `nBlockCharCreation` | ✅ |  |
| 7 | byte | byte `nChannelCount` | ✅ |  |
| 8 | byte | string `channel sName (loop body)` | 🔍 | loop body — see follow-up scan |
| 9 | int16 | int32 `channel nUserNo (load)` | ❌ | width mismatch |
| 10 | byte | byte `channel nWorldID` | ❌ | atlas: short — missing trailing field |
| 11 | byte | byte `channel nChannelID` | ❌ | atlas: short — missing trailing field |
| 12 | byte | byte `channel bAdultChannel` | ❌ | atlas: short — missing trailing field |
| 13 | byte | int16 `nBalloonCount` | ❌ | atlas: short — missing trailing field |
| 14 | byte | int16 `balloon x (loop body)` | ❌ | atlas: short — missing trailing field |
| 15 | byte | int16 `balloon y` | ❌ | atlas: short — missing trailing field |
| 16 | byte | string `balloon msg` | ❌ | atlas: short — missing trailing field |

