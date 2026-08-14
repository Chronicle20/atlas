# CashCheckTransferWorldPossibleResult (← `CCashShop::OnCheckTransferWorldPossibleResult`)

- **IDA:** 0x48e7a6
- **Atlas file:** `libs/atlas-packet/cash/clientbound/check_transfer_world_possible_result.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `` | ✅ |  |
| 1 | byte | byte `` | ✅ |  |
| 2 | int32 | byte `` | ❌ | width mismatch |
| 3 | byte | int32 `` | ❌ | width mismatch |
| 4 | int32 | string `` | ❌ | width mismatch |
| 5 | string | byte `` | ❌ | atlas: extra — client never reads this field |

