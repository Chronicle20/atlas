# SummonRemove (← `CSummonedPool::OnRemoved`)

- **IDA:** 0x828502
- **Atlas file:** `libs/atlas-packet/summon/clientbound/remove.go`
- **Variant:** JMS/v185
- **Branch depth:** 2
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `cid — read by CSummonedPool::OnPacket@0x9f7fad before dispatch (pool is cid-keyed; NO oid on jms185)` | ✅ |  |
| 1 | int32 | byte `leave-type/animated flag (0/2/3/4) — consumed in the remove path sub_828502@0x828517; atlas writes 4 or 1` | ❌ | width mismatch |
| 2 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |

