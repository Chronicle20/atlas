# FieldWarpToMap (← `CStage::OnSetField#WarpToMap`)

- **IDA:** 0x6f07d9
- **Atlas file:** `libs/atlas-packet/field/clientbound/warp_to_map.go`
- **Variant:** GMS/v79
- **Branch depth:** 4
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int32 `m_nChannelID @0x6f080c` | ❌ | width mismatch |
| 1 | int32 | byte `sNotifierMessage @0x6f082b` | ❌ | width mismatch |
| 2 | byte | byte `bCharacterData flag (warp path = 0) @0x6f0838` | ✅ |  |
| 3 | byte | int16 `nNotifierCheck @0x6f084f` | ❌ | width mismatch |
| 4 | int16 | int32 `dwPosMap (target map id) @0x6f0997 — NO revive byte in v79 (else branch reads mapId immediately)` | ❌ | width mismatch |
| 5 | int32 | byte `nPortal (target portal id) @0x6f09b5` | ❌ | width mismatch |
| 6 | byte | int16 `nHP @0x6f09c6 — v79 Decode2 (2 bytes)` | ❌ | width mismatch |
| 7 | int16 | byte `m_bChaseEnable @0x6f09de` | ❌ | width mismatch |
| 8 | byte | int64 `timestamp (DecodeBuffer p,8u FILETIME) @0x6f0a76` | ❌ | width mismatch |
| 9 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 10 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 11 | int64 | byte `` | ❌ | atlas: extra — client never reads this field |

