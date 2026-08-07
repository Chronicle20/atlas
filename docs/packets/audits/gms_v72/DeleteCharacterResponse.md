# DeleteCharacterResponse (← `CLogin::OnDeleteCharacterResult`)

- **IDA:** 0x5b3a18
- **Atlas file:** `libs/atlas-packet/character/clientbound/delete_response.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId @0x5b3a3c` | ✅ |  |
| 1 | byte | byte `code @0x5b3a3f` | ✅ |  |

