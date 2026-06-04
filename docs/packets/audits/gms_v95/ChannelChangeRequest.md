# ChannelChangeRequest (← `CField::SendTransferChannelRequest`)

- **IDA:** 0x52efa0
- **Atlas file:** `../../libs/atlas-packet/channel/serverbound/channel_change.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `nTargetChannel (target channel ID, 0-based byte)` | ✅ |  |
| 1 | int32 | int32 `get_update_time() (client tick / update time, uint32)` | ✅ |  |

