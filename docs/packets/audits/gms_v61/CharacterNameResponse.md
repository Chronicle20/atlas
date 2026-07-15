# CharacterNameResponse (← `CLogin::OnCheckDuplicatedIDResult`)

- **IDA:** 0x566c86
- **Atlas file:** `libs/atlas-packet/character/clientbound/name_response.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | int32 `` | ❌ | width mismatch |
| 1 | byte | byte `` | ✅ |  |

