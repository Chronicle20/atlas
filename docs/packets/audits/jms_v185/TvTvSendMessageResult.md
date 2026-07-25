# TvTvSendMessageResult (← `CMapleTVMan::OnSendMessageResult`)

- **IDA:** 0x6ab68d
- **Atlas file:** `libs/atlas-packet/tv/clientbound/send_message_result.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `hasError` | ✅ |  |
| 1 | byte | byte `code` | ✅ |  |

