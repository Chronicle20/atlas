# CharacterChairShow (← `CUserRemote::OnSetActivePortableChair`)

- **IDA:** 0x949240
- **Atlas file:** `libs/atlas-packet/character/clientbound/chair_show.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId (consumed by CUserPool::OnUserRemotePacket dispatcher, case 222 = 0xDE, before calling this function)` | ✅ |  |
| 1 | int32 | int32 `m_nPortableChairID (chairId)` | ✅ |  |

