# CharacterExpression (← `CUser::OnEmotion`)

- **IDA:** 0x8c8d90
- **Atlas file:** `libs/atlas-packet/character/clientbound/expression.go`
- **Variant:** GMS/v92
- **Branch depth:** 3
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `` | ✅ |  |
| 1 | int32 | int32 `` | ✅ |  |
| 2 | int32 | byte `` | ❌ | width mismatch |
| 3 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |

