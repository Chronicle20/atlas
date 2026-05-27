# CharacterViewAllError (← `CLogin::OnViewAllCharResult#CharacterViewAllSearchFailed`)

- **IDA:** 0x5de284
- **Atlas file:** `libs/atlas-packet/character/clientbound/view_all.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `code byte (case 2/3/6/7 = error — no further reads in v95)` | ✅ |  |

