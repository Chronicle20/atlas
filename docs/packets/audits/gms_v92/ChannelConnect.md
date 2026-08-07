# ChannelConnect (← `CClientSocket::OnConnect#ChannelConnect`)

- **IDA:** 
- **Atlas file:** `libs/atlas-packet/socket/serverbound/channel_connect.go`
- **Variant:** GMS/v92
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | unresolved `dispatcher-family arm not harvested for gms_v92 (see notes)` | 🚫 | IDA read-order unresolved: dispatcher-family arm not harvested for gms_v92 (see notes) |
| 1 | bytes | byte `` | ❌ | atlas: extra — client never reads this field |
| 2 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 3 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | int64 | byte `` | ❌ | atlas: extra — client never reads this field |

