# BuffGive (← `CWvsContext::OnTemporaryStatSet`)

- **IDA:** 0xb0701f
- **Atlas file:** `libs/atlas-packet/character/clientbound/buff_give.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | bytes | bytes `stat mask (DecodeBuffer 16 = UINT128)` | ✅ |  |
| 1 | int16 | int16 `first stat nValue (mask bit)` | ✅ |  |
| 2 | byte | int32 `first stat rExpireTime (mask bit)` | ❌ | width mismatch |
| 3 | byte | unresolved `remaining mask-gated nValue/rExpireTime stat reads (hand-trace)` | ❌ | atlas: short — missing trailing field |
| 4 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 5 | byte | byte `` | ❌ | atlas: short — missing trailing field |

