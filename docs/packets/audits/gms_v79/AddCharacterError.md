# AddCharacterError (← `CLogin::OnCreateNewCharacterResult#AddCharacterError`)

- **IDA:** 0x5ceb55
- **Atlas file:** `libs/atlas-packet/character/clientbound/add_entry_error.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `result code: non-zero = error (10=limit, 26=accountTransfer, 30=cannotUse)` | ✅ |  |

