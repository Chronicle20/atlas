# SummonMove (← `CSummonedPool::OnMove`)

- **IDA:** 0x7a6861
- **Atlas file:** `libs/atlas-packet/summon/clientbound/move.go`
- **Variant:** GMS/v83
- **Branch depth:** 2
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `cid — read by CSummonedPool::OnPacket@0x938dd7 before dispatch (pool is cid-keyed; NO oid on v83)` | ✅ |  |
| 1 | int32 | int16 `startX — CMovePath::OnMovePacket movement-blob head (CSummonedPool::OnMove@0x7a687a)` | ❌ | width mismatch |
| 2 | int16 | int16 `startY — CMovePath::OnMovePacket movement-blob head` | ✅ |  |
| 3 | int16 | bytes `rawMovement blob — CMovePath::OnMovePacket@0x68b371 (variable-length movement path)` | ✅ |  |
| 4 | bytes | byte `` | ✅ | absorbed by trailing opaque buffer |

