# CharacterChairShow (← `CUserRemote::OnSetActivePortableChair`)

- **IDA:** 0x925bf0
- **Atlas file:** `libs/atlas-packet/character/clientbound/chair_show.go`
- **Variant:** GMS/v92
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `` | ✅ |  |
| 1 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

