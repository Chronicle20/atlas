# ChannelChannelChange (← `CClientSocket::OnMigrateCommand`)

- **IDA:** 0x48e30a
- **Atlas file:** `libs/atlas-packet/channel/clientbound/change.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | bytes | int32 `` | ✅ |  |
| 2 | int16 | int16 `` | ✅ |  |

