# CharacterSkillChange (← `CWvsContext::OnChangeSkillRecordResult`)

- **IDA:** 0x9f5f30
- **Atlas file:** `libs/atlas-packet/character/clientbound/skill_change.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `exclRequestSent flag (bExclRequestSent)` | ✅ |  |
| 1 | int16 | int16 `count of skill entries` | ✅ |  |
| 2 | int32 | int32 `nSkillID (per entry, loop count times)` | ✅ |  |
| 3 | int32 | int32 `nLevel (nInfo, per entry)` | ✅ |  |
| 4 | int32 | int32 `nMasterLevel (per entry, for skills needing master level)` | ✅ |  |
| 5 | int64 | bytes `dateExpire: 8-byte FILETIME (per entry)` | ❌ | width mismatch |
| 6 | byte | byte `sn / MovementAffectingStat (after loop)` | ✅ |  |

---

ack: tool type-classification false positive — row 5 shows int64 (Encode8) vs bytes (DecodeBuf 8) as a width mismatch, but both are 8 bytes on the wire. IDA calls DecodeBuffer(iPacket, &dateExpire, 8u) which the tool classifies as DecodeBuf (width=-2) while atlas writes WriteInt64 (Encode8, width=8). Functionally identical 8-byte wire read; no wire bug. All other 6 fields ✅.
