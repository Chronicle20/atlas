# FieldWarpToMap (← `CStage::OnSetField#WarpToMap`)

- **IDA:** 0x71a0a0
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/warp_to_map.go`
- **Variant:** GMS/v95
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `DecodeOpt — atlas WriteShort(0) gated (GMS>83 \|\| JMS)` | ✅ |  |
| 1 | int32 | int32 `m_nChannelID (channel id)` | ✅ |  |
| 2 | int32 | int32 `m_dwOldDriverID — v95 reads Decode4 unconditionally; atlas now emits it gated GMS>=95 (RESOLVED, v83 omits)` | ✅ |  |
| 3 | byte | byte `sNotifierMessage flag (atlas writes 0)` | ✅ |  |
| 4 | byte | byte `bCharacterData flag (atlas writes 0 — warp path)` | ✅ |  |
| 5 | int16 | int16 `nNotifierCheck (atlas writes 0)` | ✅ |  |
| 6 | byte | byte `revive flag (else branch, line 180) — atlas WriteByte(0) gated (GMS>28 \|\| JMS)` | ✅ |  |
| 7 | int32 | int32 `dwPosMap (target map id)` | ✅ |  |
| 8 | byte | byte `nPortal (target portal id)` | ✅ |  |
| 9 | int32 | int32 `nHP — v95 Decode4; atlas was WriteShort(2), FIXED to WriteInt(4)` | ✅ |  |
| 10 | byte | byte `m_bChaseEnable — atlas WriteBool(false) gated (GMS>28)` | ✅ |  |
| 11 | int64 | int64 `timestamp (DecodeBuffer p,8u); atlas WriteInt64` | ✅ |  |

