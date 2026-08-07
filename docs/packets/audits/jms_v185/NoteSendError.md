# NoteSendError (← `CWvsContext::OnMemoResult#SendError`)

- **IDA:** 0xb0c6d0
- **Atlas file:** `libs/atlas-packet/note/clientbound/operation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (5 = SendError)` | ✅ |  |
| 1 | byte | byte `errorCode` | ✅ |  |

