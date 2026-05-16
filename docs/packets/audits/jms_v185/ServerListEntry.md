# ServerListEntry (← `CLogin::OnWorldInformation`)

- **IDA:** 0x66f107
- **Atlas file:** `libs/atlas-packet/login/clientbound/server_list_entry.go`
- **Variant:** JMS/v185
- **Branch depth:** 2
- **Verdict:** ❌

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
| 7 | string | byte `nChannelCount` | ❌ | width mismatch |
| 8 | int32 | string `channel sName (loop body)` | ❌ | width mismatch |
| 9 | byte | int32 `channel nUserNo (load)` | ❌ | width mismatch |
| 10 | byte | byte `channel nWorldID` | ✅ |  |
| 11 | byte | byte `channel nChannelID` | ✅ |  |
| 12 | int16 | byte `channel bAdultChannel` | ❌ | width mismatch |
| 13 | int16 | int16 `nBalloonCount` | ✅ |  |
| 14 | int16 | int16 `balloon x (loop body)` | ✅ |  |
| 15 | string | int16 `balloon y` | ❌ | width mismatch |
| 16 | byte | string `balloon msg` | ⚠️ | loop body — atlas emits zero iterations (count==0) |

