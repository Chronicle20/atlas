# NoteSendError (← `CWvsContext::OnMemoResult#SendError`)

- **IDA:** 0x9f9da0
- **Atlas file:** `libs/atlas-packet/note/clientbound/operation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (=5, SEND_ERROR)` | ✅ |  |
| 1 | byte | byte `errorCode — 0=RECEIVER_ONLINE, 1=RECEIVER_UNKNOWN, 2=RECEIVER_INBOX_FULL` | ✅ |  |

