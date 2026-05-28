# CharacterHint (← `CUserLocal::OnBalloonMsg`)

- **IDA:** 0x95d88b
- **Atlas file:** `../../libs/atlas-packet/character/clientbound/hint.go`
- **Variant:** GMS/v83
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `msg (hint text)` | ✅ |  |
| 1 | int16 | int16 `width (balloon width)` | ✅ |  |
| 2 | int16 | int16 `duration/height (1000 * value ms)` | ✅ |  |
| 3 | byte | byte `notAtPoint flag (0=use explicit x,y; non-zero=use avatar position)` | ✅ |  |
| 4 | int32 | int32 `x coordinate (only if notAtPoint == 0)` | ✅ |  |
| 5 | int32 | int32 `y coordinate (only if notAtPoint == 0)` | ✅ |  |

