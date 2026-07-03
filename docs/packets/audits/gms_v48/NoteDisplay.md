# NoteDisplay (← `CWvsContext::OnMemoResult#Display`)

- **IDA:** 0x71d8e2
- **Atlas file:** `libs/atlas-packet/note/clientbound/display.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (=2, Display/SHOW) — raw sub-op 2 (Decode1-2==0 @0x71d8fe)` | ✅ |  |
| 1 | byte | byte `count — number of GW_Memo entries @0x71d9a9` | ✅ |  |
| 2 | int32 | int32 `dwSN (memo id) — GW_Memo loop body sub_49CCDB @0x49cced (count iterations; analyzer flattens)` | ✅ |  |
| 3 | string | string `sSender (sender name) @0x49ccf5` | ✅ |  |
| 4 | string | string `sContent (message body) @0x49cd16` | ✅ |  |
| 5 | int64 | bytes `dateSent (8-byte FILETIME via DecodeBuffer(8)) @0x49cd33` | ✅ |  |
| 6 | byte | byte `nFlag (memo status flag) @0x49cd3f` | ✅ |  |

