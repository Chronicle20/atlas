# NoteDisplay (← `CWvsContext::OnMemoResult#Display`)

- **IDA:** 0xabccc2
- **Atlas file:** `libs/atlas-packet/note/clientbound/display.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (3)` | ✅ |  |
| 1 | byte | byte `count of memos` | ✅ |  |
| 2 | int32 | bytes `GW_Memo::Decode per entry (loop count)` | ❌ | width mismatch |
| 3 | string | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | string | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | int64 | byte `` | ❌ | atlas: extra — client never reads this field |
| 6 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

