# FieldWarpToMap (← `CStage::OnSetField#WarpToMap`)

- **IDA:** 0x7c429c
- **Atlas file:** `libs/atlas-packet/field/clientbound/warp_to_map.go`
- **Variant:** GMS/v87
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `DecodeOpt — atlas WriteShort(0) gated (GMS>83 \|\| JMS); present v87` | ✅ |  |
| 1 | int32 | int32 `m_nChannelID (channel id, @0x7c42db)` | ✅ |  |
| 2 | byte | byte `sNotifierMessage flag (@0x7c42fa; atlas writes 0) — no m_dwOldDriverID before it in v87` | ✅ |  |
| 3 | byte | byte `bCharacterData flag (@0x7c4307; atlas writes 0 — warp path)` | ✅ |  |
| 4 | int16 | int16 `nNotifierCheck (@0x7c431e; atlas writes 0)` | ✅ |  |
| 5 | byte | byte `revive flag (else branch @0x7c4423) — atlas WriteByte(0) gated (GMS>28 \|\| JMS)` | ✅ |  |
| 6 | int32 | int32 `dwPosMap (target map id, @0x7c447b)` | ✅ |  |
| 7 | byte | byte `nPortal (target portal id, @0x7c449d)` | ✅ |  |
| 8 | int16 | int16 `nHP — v87 Decode2 (2 bytes, LOWORD @0x7c44a8), matches v83; v95 reads Decode4. atlas WriteShort(2) for GMS<95` | ✅ |  |
| 9 | byte | byte `m_bChaseEnable (@0x7c44c2) — atlas WriteBool(false) gated (GMS>28)` | ✅ |  |
| 10 | int64 | int64 `timestamp (DecodeBuffer p,8u); atlas WriteInt64` | ✅ |  |


Ack: world-audit Phase 3 v87 cross-version on 2026-05-28
