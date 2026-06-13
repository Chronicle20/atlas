# FieldWarpToMap (← `CStage::OnSetField#WarpToMap`)

- **IDA:** 0x776020
- **Atlas file:** `libs/atlas-packet/field/clientbound/warp_to_map.go`
- **Variant:** GMS/v83
- **Branch depth:** 2
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int32 `m_nChannelID (channel id) — v83 NO DecodeOpt, NO m_dwOldDriverID` | ❌ | width mismatch |
| 1 | int32 | byte `sNotifierMessage flag` | ❌ | width mismatch |
| 2 | byte | byte `bCharacterData flag (warp path = 0)` | ✅ |  |
| 3 | byte | int16 `nNotifierCheck` | ❌ | width mismatch |
| 4 | int16 | byte `revive flag (else branch, v10)` | ❌ | width mismatch |
| 5 | byte | int32 `dwPosMap (target map id, v13)` | ❌ | width mismatch |
| 6 | int32 | byte `nPortal (target portal id, v17)` | ❌ | width mismatch |
| 7 | byte | int16 `nHP — v83 reads Decode2 (2 bytes), v95 reads Decode4 (v20)` | ❌ | width mismatch |
| 8 | int16 | byte `m_bChaseEnable (v23)` | ❌ | width mismatch |
| 9 | byte | int64 `timestamp (DecodeBuffer p,8u FILETIME)` | ❌ | width mismatch |
| 10 | int64 | byte `` | ❌ | atlas: extra — client never reads this field |

