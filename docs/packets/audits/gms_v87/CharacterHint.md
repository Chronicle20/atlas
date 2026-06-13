# CharacterHint (← `CUserLocal::OnBalloonMsg`)

- **IDA:** 0x9dff6a
- **Atlas file:** `libs/atlas-packet/character/clientbound/hint.go`
- **Variant:** GMS/v87
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `msg (hint text)` | ✅ |  |
| 1 | int16 | int16 `width (balloon width)` | ✅ |  |
| 2 | int16 | int16 `duration/height` | ✅ |  |
| 3 | byte | byte `notAtPoint flag` | ✅ |  |
| 4 | int32 | int32 `x coordinate (only if notAtPoint == 0)` | ✅ |  |
| 5 | int32 | int32 `y coordinate (only if notAtPoint == 0)` | ✅ |  |

