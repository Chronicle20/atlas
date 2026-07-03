# NoteSendSuccess (← `CWvsContext::OnMemoResult#SendSuccess`)

- **IDA:** 0x71d8e2
- **Atlas file:** `libs/atlas-packet/note/clientbound/operation.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (=3, SEND_SUCCESS) — no further bytes; success path shows UI notification only` | ✅ |  |

