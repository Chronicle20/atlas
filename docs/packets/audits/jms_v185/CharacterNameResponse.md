# CharacterNameResponse (← `CLogin::OnCheckDuplicatedIDResult`)

- **IDA:** 0x66f957
- **Atlas file:** `../../libs/atlas-packet/character/clientbound/name_response.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `character name string (ZXString)` | ✅ |  |
| 1 | byte | byte `v3 result code: 0=OK, 1=alreadyRegistered, 2=notAllowed, else=systemError` | ✅ |  |

