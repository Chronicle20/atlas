# DeleteCharacterResponse (← `CLogin::OnDeleteCharacterResult`)

- **IDA:** 0x5f9d15
- **Atlas file:** `../../libs/atlas-packet/character/clientbound/delete_response.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwCharacterID (characterId of the deleted character)` | ✅ |  |
| 1 | byte | byte `result code (0=success; error codes: 6=system error, 9=unknown, 10=too many connections, 18=?, 20=secondary pin mismatch, 22=guild master, 24=engaged, 26=account transfer, 29=has family)` | ✅ |  |

