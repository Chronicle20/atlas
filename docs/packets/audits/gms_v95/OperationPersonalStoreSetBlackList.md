# OperationPersonalStoreSetBlackList (← `CPersonalShopDlg::DeliverBlackList`)

- **IDA:** 0x69b0d0
- **Atlas file:** `../../libs/atlas-packet/interaction/serverbound/operation_personal_store_set_black_list.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `count (config blacklist size)` | ✅ |  |
| 1 | byte | string `name[] (per-entry EncodeStr, count times)` | ❌ | width mismatch |


> defer: REAL ❌ — atlas reads byte[] but client sends string[] (count x EncodeStr).
> Structural + version-sensitive; no cross-version IDA. See
> `docs/packets/ida-exports/_pending.md` → "OperationPersonalStoreSetBlackList".
