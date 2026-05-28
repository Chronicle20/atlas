# FieldWarpToMap (← `CStage::OnSetField#WarpToMap`)

- **IDA:** 0x776020
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/warp_to_map.go`
- **Variant:** GMS/v83
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `m_nChannelID (channel id) — v83 NO DecodeOpt, NO m_dwOldDriverID` | ✅ |  |
| 1 | byte | byte `sNotifierMessage flag` | ✅ |  |
| 2 | byte | byte `bCharacterData flag (warp path = 0)` | ✅ |  |
| 3 | int16 | int16 `nNotifierCheck` | ✅ |  |
| 4 | byte | byte `revive flag (else branch, v10)` | ✅ |  |
| 5 | int32 | int32 `dwPosMap (target map id, v13)` | ✅ |  |
| 6 | byte | byte `nPortal (target portal id, v17)` | ✅ |  |
| 7 | int16 | int16 `nHP — v83 reads Decode2 (2 bytes), v95 reads Decode4 (v20)` | ✅ |  |
| 8 | byte | byte `m_bChaseEnable (v23)` | ✅ |  |
| 9 | int64 | int64 `timestamp (DecodeBuffer p,8u FILETIME)` | ✅ |  |


Ack: world-audit Phase 3 v83 on 2026-05-28
