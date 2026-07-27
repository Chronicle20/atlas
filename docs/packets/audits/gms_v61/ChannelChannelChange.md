# ChannelChannelChange (← `CClientSocket::OnMigrateCommand`)

- **IDA:** 0x47451a
- **Atlas file:** `libs/atlas-packet/channel/clientbound/change.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | bytes | int32 `` | ✅ |  |
| 2 | int16 | int16 `` | ✅ |  |

