# CharacterViewAllCount (← `CLogin::OnViewAllCharResult#CharacterViewAllCount`)

- **IDA:** 0x5facca
- **Atlas file:** `libs/atlas-packet/character/clientbound/view_all.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `code byte (case 1 = CHARACTER_COUNT)` | ✅ |  |
| 1 | int32 | int32 `m_nCountRelatedSvrs (server/world count)` | ✅ |  |
| 2 | int32 | int32 `m_nCountCharacters (total character count)` | ✅ |  |

