# StorageUpdateAssets (← `CTrunkDlg::OnPacket#UpdateAssets`)

- **IDA:** 0x76a990
- **Atlas file:** `../../libs/atlas-packet/storage/clientbound/update_assets.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (9/13/15/19)` | ✅ |  |
| 1 | byte | byte `slotCount (m_nSlotCount)` | ✅ |  |
| 2 | int64 | int64 `tab-flag bitmask (8 bytes via DecodeBuffer; WriteLong-compatible width)` | ✅ |  |
| 3 | byte | int32 `meso (m_nMoney; ONLY if flag&2 — runtime callers never set bit 2)` | ❌ | width mismatch |
| 4 | byte | byte `PER-TAB count byte; repeated once per set tab bit (4/8/16/32/64), each followed by count*GW_ItemSlotBase::Decode` | 🔍 | sub-struct: model.Asset — see _substruct/ |

