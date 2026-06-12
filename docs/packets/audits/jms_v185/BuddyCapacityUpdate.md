# BuddyCapacityUpdate (← `CWvsContext::OnFriendResult#CapacityUpdate`)

- **IDA:** 0xb2a873
- **Atlas file:** `../../libs/atlas-packet/buddy/clientbound/capacity_update.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `sub-op mode byte` | ✅ |  |
| 1 | byte | int32 `case 0x14: buddy count (CapacityUpdate)` | ❌ | width mismatch |
| 2 | byte | byte `case 0x15: new capacity byte` | ❌ | atlas: short — missing trailing field |

