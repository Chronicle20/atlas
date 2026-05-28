# UpdateAssets (← `CTrunkDlg::OnPacket#UpdateAssets`)

- **IDA:** 0x7c5dfd
- **Atlas file:** `../../libs/atlas-packet/storage/clientbound/update_assets.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (9/13/15/19; consumed by OnPacket dispatcher then SetGetItems)` | ✅ |  |
| 1 | byte | byte `slotCount (SetGetItems *(this+62))` | ✅ |  |
| 2 | int64 | int64 `tab-flag bitmask (8 bytes via DecodeBuffer; v21)` | ✅ |  |
| 3 | byte | int32 `meso (*(this+63); ONLY if flag&2 — runtime callers never set bit 2)` | ❌ | width mismatch |
| 4 | byte | byte `PER-TAB count byte; repeated once per set tab bit (4/8/16/32/64), each followed by count*GW_ItemSlotBase::Decode` | 🔍 | sub-struct: Asset — see _substruct/ |

