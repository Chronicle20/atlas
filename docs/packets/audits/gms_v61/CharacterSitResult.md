# CharacterSitResult (← `CUserLocal::OnSitResult`)

- **IDA:** 0x7ab9d9
- **Atlas file:** `libs/atlas-packet/character/clientbound/sit_result.go`
- **Variant:** GMS/v61
- **Branch depth:** 1
- **Verdict:** ⚠️

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | int16 | int16 `` | ✅ |  |
| 2 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |

