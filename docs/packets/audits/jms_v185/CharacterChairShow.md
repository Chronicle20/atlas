# CharacterChairShow (← `CUserRemote::OnSetActivePortableChair`)

- **IDA:** 0xa44324
- **Atlas file:** `libs/atlas-packet/character/clientbound/chair_show.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId (consumed by CUserPool::OnUserRemotePacket)` | ✅ |  |
| 1 | int32 | int32 `m_nPortableChairID (chairId)` | ✅ |  |

