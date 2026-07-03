# NoteSendError (← `CWvsContext::OnMemoResult#SendError`)

- **IDA:** 0x71d8e2
- **Atlas file:** `libs/atlas-packet/note/clientbound/operation.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (=4, SEND_ERROR) — raw sub-op 4 (Decode1-2==2)` | ✅ |  |
| 1 | byte | byte `errorCode @0x71d937 — 0/1/2 → Notice 2372/2373/2374` | ✅ |  |

