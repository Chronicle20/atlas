# KeyMapChange (← `CFuncKeyMappedMan::SaveFuncKeyMap`)

- **IDA:** 0x568a60
- **Atlas file:** `libs/atlas-packet/character/serverbound/key_map_change.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `mode (0 = key mapping change; 1 = HP-pet item; 2 = MP-pet item)` | ✅ |  |
| 1 | int32 | int32 `count (number of changed key slot indices); per-entry below verified via FUNCKEY_MAPPED::Encode@0x4f6d80` | ✅ |  |
| 2 | int32 | int32 `[loop] keyId (key slot index from anChangedIdx array)` | 🔍 | ack:tool-limitation — loop body not linearisable; IDA export limited to invariant header (mode+count); FUNCKEY_MAPPED::Decode@0x4f2b20 confirms 5-byte struct nType(1)+nID(4) |
| 3 | byte | byte `[loop] theType (FUNCKEY_MAPPED::nType)` | 🔍 | ack:tool-limitation — see row 2 |
| 4 | int32 | int32 `[loop] action (FUNCKEY_MAPPED::nID)` | 🔍 | ack:tool-limitation — see row 2 |
| 5 | int32 | int32 `[mode≠0] itemId (pet-consume item ID; mode=1→HP-pot, mode=2→MP-pot)` | 🔍 | ack:tool-limitation — mode-branch field; verified via ChangePetConsumeItemID@0x568920 (mode=1) and ChangePetConsumeMPItemID@0x5689c0 (mode=2) |

