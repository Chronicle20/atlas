# CharacterViewAllError (← `CLogin::OnViewAllCharResult#CharacterViewAllSearchFailed`)

- **IDA:** 0x6328eb
- **Atlas file:** `libs/atlas-packet/character/clientbound/view_all.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `code byte (case 2/3/6/7 = error)` | ✅ |  |

