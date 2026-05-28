# NoteSendError (← `CWvsContext::OnMemoResult#SendError`)

- **IDA:** 0xa2508b
- **Atlas file:** `libs/atlas-packet/note/clientbound/operation.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (5)` | ✅ |  |
| 1 | byte | byte `errorCode` | ✅ |  |

