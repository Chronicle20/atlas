# FieldWarpToMap (← `CStage::OnSetField#WarpToMap`)

- **IDA:** 0x6c0c9b
- **Atlas file:** `libs/atlas-packet/field/clientbound/warp_to_map.go`
- **Variant:** GMS/v72
- **Branch depth:** 4
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int32 `m_nChannelID (channel id) @0x6c0cce — v72 NO DecodeOpt, NO m_dwOldDriverID` | ❌ | width mismatch |
| 1 | int32 | byte `sNotifierMessage flag @0x6c0ced` | ❌ | width mismatch |
| 2 | byte | byte `bCharacterData flag (warp path = 0) @0x6c0cfa` | ✅ |  |
| 3 | byte | int16 `nNotifierCheck @0x6c0d11` | ❌ | width mismatch |
| 4 | int16 | int32 `dwPosMap (target map id) @0x6c0e59 — NO revive byte in v72 (else branch reads mapId immediately, v72 < v83)` | ❌ | width mismatch |
| 5 | int32 | byte `nPortal (target portal id) @0x6c0e77` | ❌ | width mismatch |
| 6 | byte | int16 `nHP @0x6c0e88 — v72 Decode2 (2 bytes)` | ❌ | width mismatch |
| 7 | int16 | byte `m_bChaseEnable @0x6c0ea0` | ❌ | width mismatch |
| 8 | byte | int64 `timestamp (DecodeBuffer p,8u FILETIME) @0x6c0f38` | ❌ | width mismatch |
| 9 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 10 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 11 | int64 | byte `` | ❌ | atlas: extra — client never reads this field |

