# CharacterSitResult (← `CUserLocal::OnSitResult`)

- **IDA:** 0x865e68
- **Atlas file:** `libs/atlas-packet/character/clientbound/sit_result.go`
- **Variant:** GMS/v72
- **Branch depth:** 1
- **Verdict:** 🚫

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | int16 | int16 `` | ✅ |  |
| 2 | byte | unresolved `packet var passed to unresolved/indirect call; hand-trace` | 🚫 | IDA read-order unresolved: packet var passed to unresolved/indirect call; hand-trace |

