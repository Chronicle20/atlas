# NoteDisplay (← `CWvsContext::OnMemoResult#Display`)

- **IDA:** 0x9f9da0
- **Atlas file:** `../../libs/atlas-packet/note/clientbound/display.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (=3, SHOW)` | ✅ |  |
| 1 | byte | byte `count — number of GW_Memo entries` | ✅ |  |
| 2 | int32 | int32 `dwSN (memo id) — GW_Memo::Decode loop body (count iterations; analyzer flattens)` | ✅ |  |
| 3 | string | string `sSender (sender name, max 13 chars)` | ✅ |  |
| 4 | string | string `sContent (message body, max 201 chars)` | ✅ |  |
| 5 | int64 | bytes `dateSent (8-byte FILETIME timestamp via DecodeBuffer(8))` | ❌ | width mismatch |
| 6 | byte | byte `nFlag (memo status flag)` | ✅ |  |

