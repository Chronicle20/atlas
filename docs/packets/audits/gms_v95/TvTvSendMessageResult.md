# TvTvSendMessageResult (← `CMapleTVMan::OnSendMessageResult`)

- **IDA:** 0x60f5f0
- **Atlas file:** `libs/atlas-packet/tv/clientbound/send_message_result.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `hasError` | ✅ |  |
| 1 | byte | byte `code - code1=GM_MESSAGE(SP 3998), code2=WRONG_USER(SP 4000, the v5==0/'else' branch), code3=QUEUE_TOO_LONG(SP 3999, the v5==1 branch); same branch-order structure as gms_v83` | ✅ |  |

