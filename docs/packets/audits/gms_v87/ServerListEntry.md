# ServerListEntry (← `CLogin::OnWorldInformation`)

- **IDA:** 0x630e7c
- **Atlas file:** `../../libs/atlas-packet/login/clientbound/server_list_entry.go`
- **Variant:** GMS/v87
- **Branch depth:** 2
- **Verdict:** ✅

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
| 8 | string | string `channel sName (loop body)` | ✅ |  |
| 9 | int32 | int32 `channel nUserNo (load)` | ✅ |  |
| 10 | byte | byte `channel nWorldID` | ✅ |  |
| 11 | byte | byte `channel nChannelID` | ✅ |  |
| 12 | byte | byte `channel bAdultChannel` | ✅ |  |
| 13 | int16 | int16 `nBalloonCount` | ✅ |  |
| 14 | int16 | int16 `balloon x (loop body)` | ✅ |  |
| 15 | int16 | int16 `balloon y` | ✅ |  |
| 16 | string | string `balloon msg` | ✅ |  |

