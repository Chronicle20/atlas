# ChannelConnect (← `CClientSocket::OnConnect#ChannelConnect`)

- **IDA:** 
- **Atlas file:** `../../libs/atlas-packet/socket/serverbound/channel_connect.go`
- **Variant:** GMS/v84
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | unresolved `function not found in IDB` | 🚫 | IDA read-order unresolved: function not found in IDB |
| 1 | bytes | byte `` | ❌ | atlas: extra — client never reads this field |
| 2 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 3 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | int64 | byte `` | ❌ | atlas: extra — client never reads this field |

