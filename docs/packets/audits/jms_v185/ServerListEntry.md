# ServerListEntry (← `CLogin::OnWorldInformation`)

- **IDA:** 0x66f107
- **Atlas file:** `libs/atlas-packet/login/clientbound/server_list_entry.go`
- **Variant:** JMS/v185
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
| 6 | byte | byte `nChannelCount` | ✅ |  |
| 7 | string | string `channel sName (loop body)` | ✅ |  |
| 8 | int32 | int32 `channel nUserNo (load)` | ✅ |  |
| 9 | byte | byte `channel nWorldID` | ✅ |  |
| 10 | byte | byte `channel nChannelID` | ✅ |  |
| 11 | byte | byte `channel bAdultChannel` | ✅ |  |
| 12 | int16 | int16 `nBalloonCount` | ✅ |  |
| 13 | int16 | int16 `balloon x (loop body)` | ✅ |  |
| 14 | int16 | int16 `balloon y` | ✅ |  |
| 15 | string | string `balloon msg` | ✅ |  |

