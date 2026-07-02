# ServerListEntry (← `CLogin::OnWorldInformation`)

- **IDA:** 0x56663f
- **Atlas file:** `libs/atlas-packet/login/clientbound/server_list_entry.go`
- **Variant:** GMS/v61
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `nWorldID @0x566660` | ✅ |  |
| 1 | string | string `sName @0x5666c4` | ✅ |  |
| 2 | byte | byte `nWorldState @0x5666f8` | ✅ |  |
| 3 | string | string `sWorldEventDesc @0x566701` | ✅ |  |
| 4 | int16 | int16 `nWorldEventEXP_WSE @0x566737` | ✅ |  |
| 5 | int16 | int16 `nWorldEventDrop_WSE @0x566744` | ✅ |  |
| 6 | byte | byte `nBlockCharCreation @0x566751` | ✅ |  |
| 7 | byte | byte `nChannelCount @0x566754` | ✅ |  |
| 8 | string | string `channel sName @0x56677b (loop body)` | ✅ |  |
| 9 | int32 | int32 `channel nUserNo @0x5667ad` | ✅ |  |
| 10 | byte | byte `channel nWorldID @0x5667ba` | ✅ |  |
| 11 | byte | byte `channel nChannelID @0x5667c7` | ✅ |  |
| 12 | byte | byte `channel bAdultChannel @0x5667ca` | ✅ |  |
| 13 | int16 | int16 `nBalloonCount @0x5667ea` | ✅ |  |
| 14 | int16 | int16 `balloon x @0x56681c (loop body)` | ✅ |  |
| 15 | int16 | int16 `balloon y @0x566827` | ✅ |  |
| 16 | string | string `balloon msg @0x566830` | ✅ |  |

