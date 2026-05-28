# DeleteCharacterResponse (← `CLogin::OnDeleteCharacterResult`)

- **IDA:** 0x66f9fe
- **Atlas file:** `../../libs/atlas-packet/character/clientbound/delete_response.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwCharacterID (characterId of the deleted character)` | ✅ |  |
| 1 | byte | byte `result code (0=success; error codes match GMS v95)` | ✅ |  |

