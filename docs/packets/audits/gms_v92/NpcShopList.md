# NpcShopList (← `CShopDlg::SetShopDlg`)

- **IDA:** 0x6df6c0
- **Atlas file:** `libs/atlas-packet/npc/clientbound/shop_list.go`
- **Variant:** GMS/v92
- **Branch depth:** 0
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `` | ✅ |  |
| 1 | int16 | int16 `` | ✅ |  |
| 2 | int32 | int32 `` | ✅ |  |
| 3 | int32 | int32 `` | ✅ |  |
| 4 | byte | byte `` | ✅ |  |
| 5 | int32 | int32 `` | ✅ |  |
| 6 | int32 | int32 `` | ✅ |  |
| 7 | int32 | int32 `` | ✅ |  |
| 8 | int16 | int32 `` | ❌ | width mismatch |
| 9 | int64 | bytes `` | ✅ |  |
| 10 | int16 | int16 `` | ✅ |  |
| 11 | byte | int16 `` | ❌ | atlas: short — missing trailing field |

