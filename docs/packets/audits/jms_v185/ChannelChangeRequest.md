# ChannelChangeRequest (← `CField::SendTransferChannelRequest`)

- **IDA:** 0x56d886
- **Atlas file:** `../../libs/atlas-packet/channel/serverbound/channel_change.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `nTargetChannel (opcode 0x1E)` | ✅ |  |
| 1 | int32 | int32 `get_update_time() (client tick)` | ✅ |  |

