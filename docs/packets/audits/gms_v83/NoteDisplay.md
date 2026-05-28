# NoteDisplay (← `CWvsContext::OnMemoResult#Display`)

- **IDA:** 0xa2508b
- **Atlas file:** `libs/atlas-packet/note/clientbound/display.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (3)` | ✅ |  |
| 1 | byte | byte `item count` | ✅ |  |
| 2 | int32 | byte `item[i].senderFlag (loop — sub_4E4ADB)` | ❌ | width mismatch |
| 3 | string | int32 `item[i].noteId` | ❌ | width mismatch |
| 4 | string | string `item[i].senderName` | ✅ |  |
| 5 | int64 | string `item[i].message` | ❌ | width mismatch |
| 6 | byte | int32 `item[i].timestamp` | ❌ | width mismatch |
| 7 | byte | byte `item[i].flags` | ❌ | atlas: short — missing trailing field |

