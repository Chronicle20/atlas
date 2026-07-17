# TvTvSendMessageResult (← `CMapleTVMan::OnSendMessageResult`)

- **IDA:** 0x6373a0
- **Atlas file:** `libs/atlas-packet/tv/clientbound/send_message_result.go`
- **Variant:** GMS/v83
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `hasError` | ✅ |  |
| 1 | byte | byte `code` | ✅ |  |

