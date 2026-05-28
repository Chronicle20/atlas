# CharacterSitResult (← `CUserLocal::OnSitResult`)

- **IDA:** 0xa244fd
- **Atlas file:** `../../libs/atlas-packet/character/clientbound/sit_result.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `sitting flag (0=cancel sit, 1=sit in chair)` | ✅ |  |
| 1 | int16 | int16 `chairId / nSeat (only if sitting flag == 1)` | ✅ |  |
| 2 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

