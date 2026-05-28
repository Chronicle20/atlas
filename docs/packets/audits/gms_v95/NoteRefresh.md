# NoteRefresh (← `CWvsContext::OnMemoResult#Refresh`)

- **IDA:** 0x9f9da0
- **Atlas file:** `libs/atlas-packet/note/clientbound/operation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (=7, REFRESH) — no further bytes; calls OnMemoNotify_Receive internally to send REQUEST packet` | ✅ |  |

