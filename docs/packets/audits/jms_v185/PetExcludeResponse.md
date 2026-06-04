# PetExcludeResponse (← `CPet::OnLoadExceptionList`)

- **IDA:** 0x76be76
- **Atlas file:** `../../libs/atlas-packet/pet/clientbound/exclude.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId — read by dispatcher` | ✅ |  |
| 1 | byte | byte `slot — read by dispatcher` | ✅ |  |
| 2 | int64 | bytes `petLockerSN (8 bytes)` | ✅ |  |
| 3 | byte | byte `nCount` | ✅ |  |
| 4 | int32 | int32 `excluded item id per entry` | ✅ |  |

