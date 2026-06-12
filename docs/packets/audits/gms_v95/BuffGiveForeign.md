# BuffGiveForeign (← `CUserRemote::OnSetTemporaryStat`)

- **IDA:** 0xb13200
- **Atlas file:** `../../libs/atlas-packet/character/clientbound/buff_give.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ⚠️

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId (read by CUserPool dispatcher before OnSetTemporaryStat)` | ✅ |  |
| 1 | bytes | bytes `SecondaryStat::DecodeForRemote — opaque remote stat block` | ✅ |  |
| 2 | int16 | int16 `tDelay` | ✅ |  |
| 3 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |

