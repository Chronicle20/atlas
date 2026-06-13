# AddCharacterError (← `CLogin::OnCreateNewCharacterResult#AddCharacterError`)

- **IDA:** 0x60f268
- **Atlas file:** `libs/atlas-packet/character/clientbound/add_entry_error.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | byte | unresolved `function not found in IDB` | ❌ | atlas: short — missing trailing field |
| 2 | byte | unresolved `function not found in IDB` | ❌ | atlas: short — missing trailing field |

