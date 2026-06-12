# StorageUpdateAssets (← `CTrunkDlg::OnPacket#UpdateAssets`)

- **IDA:** 0x7c5dfd
- **Atlas file:** `../../libs/atlas-packet/storage/clientbound/update_assets.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (9/13/15/19)` | ✅ |  |
| 1 | byte | byte `slotCount (m_nSlotCount)` | ✅ |  |
| 2 | int64 | int64 `tab-flag bitmask (8 bytes via DecodeBuffer; WriteLong-compatible width)` | ✅ |  |
| 3 | byte | int32 `meso (m_nMoney; ONLY if tab-flag bit 1 set — runtime callers never set it)` | ❌ | width mismatch |
| 4 | byte | byte `PER-TAB count byte; repeated per set tab bit (4/8/16/32/64), each followed by count*GW_ItemSlotBase::Decode` | 🔍 | opaque type: model.Asset — register boundary (see opaque registry) |

