# FieldViciousHammerSuccess (← `CField::OnItemUpgrade#Success`)

- **IDA:** 0x52a430
- **Atlas file:** `libs/atlas-packet/field/clientbound/vicious_hammer.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (= 65; success) — v5 stored to this->m_nReturnResult, then `if (v5==65)` in OnItemUpgradeResult` | ✅ |  |
| 1 | int32 | int32 `flag (v8; 0 = success -> "N upgrades left" message; non-0 -> "Unknown error %d" using this->m_nResult)` | ✅ |  |

