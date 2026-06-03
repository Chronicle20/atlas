# FieldWarpToMap (← `CStage::OnSetField#WarpToMap`)

- **IDA:** 0x7eea69
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/warp_to_map.go`
- **Variant:** JMS/v185
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `DecodeOpt — atlas WriteShort(0) gated (GMS>83 \|\| JMS)` | ✅ |  |
| 1 | int32 | int32 `m_nChannelID (channel id)` | ✅ |  |
| 2 | byte | byte `JMS-only byte — atlas WriteByte(0) gated JMS` | ✅ |  |
| 3 | int32 | int32 `JMS-only int — atlas WriteInt(0) gated JMS` | ✅ |  |
| 4 | byte | byte `sNotifierMessage flag (atlas writes 0)` | ✅ |  |
| 5 | byte | byte `bCharacterData flag (atlas writes 0 — warp path)` | ✅ |  |
| 6 | int16 | int16 `nNotifierCheck (atlas writes 0)` | ✅ |  |
| 7 | byte | byte `revive flag (@line170; else branch) — atlas WriteByte(0) gated (GMS>28 \|\| JMS)` | ✅ |  |
| 8 | int32 | int32 `dwPosMap / target map id (@0x7eec6b)` | ✅ |  |
| 9 | byte | byte `nPortal / target portal id (@0x7eec87)` | ✅ |  |
| 10 | int16 | int16 `nHP (@0x7eec9d Decode2 — 2 bytes; JMS did NOT widen with GMS v95) — atlas WriteShort gated (else branch)` | ✅ |  |
| 11 | int64 | int64 `timestamp (DecodeBuffer p,8; FILETIME) — atlas WriteInt64` | ✅ |  |


Ack: world-audit Phase 3 JMS185 field+portal on 2026-05-28
