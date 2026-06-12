# NoteDisplay (← `CWvsContext::OnMemoResult#Display`)

- **IDA:** 0xa2508b
- **Atlas file:** `../../libs/atlas-packet/note/clientbound/display.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (3 = Display)` | ✅ |  |
| 1 | byte | byte `note count (loop)` | ✅ |  |
| 2 | int32 | int32 `note.id` | ✅ |  |
| 3 | string | string `note.senderName` | ✅ |  |
| 4 | string | string `note.message` | ✅ |  |
| 5 | int64 | bytes `note.timestamp FILETIME (8 bytes)` | ✅ |  |
| 6 | byte | byte `note.flag` | ✅ |  |

