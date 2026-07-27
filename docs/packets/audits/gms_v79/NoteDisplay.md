# NoteDisplay (← `CWvsContext::OnMemoResult#Display`)

- **IDA:** 
- **Atlas file:** `libs/atlas-packet/note/clientbound/display.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | unresolved `function not found in IDB` | 🚫 | IDA read-order unresolved: function not found in IDB |
| 1 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 2 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 3 | string | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | string | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | int64 | byte `` | ❌ | atlas: extra — client never reads this field |
| 6 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

