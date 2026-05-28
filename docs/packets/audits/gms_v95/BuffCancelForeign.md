# BuffCancelForeign (← `CUserRemote::OnResetTemporaryStat`)

- **IDA:** 0x953e40
- **Atlas file:** `../../libs/atlas-packet/character/clientbound/buff_cancel.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId — read by CUserPool::OnUserRemotePacket before dispatch` | ✅ |  |
| 1 | byte | bytes `uFlagTemp: 16-byte UINT128 stat mask (DecodeBuffer 0x10)` | ❌ | width mismatch |

