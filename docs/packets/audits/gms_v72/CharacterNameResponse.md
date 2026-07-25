# CharacterNameResponse (← `CLogin::OnCheckDuplicatedIDResult`)

- **IDA:** 0x5b3983
- **Atlas file:** `libs/atlas-packet/character/clientbound/name_response.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `name @0x5b39a2` | ✅ |  |
| 1 | byte | byte `code @0x5b39ad` | ✅ |  |

