# StorageShow (← `CTrunkDlg::OnPacket#Show`)

- **IDA:** 0x819648
- **Atlas file:** `../../libs/atlas-packet/storage/clientbound/show.go`
- **Variant:** GMS/v87
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (16/21/22; consumed by OnPacket dispatcher then SetGetItems this[66])` | ✅ |  |
| 1 | int32 | int32 `npcTemplateId (dispatcher; not in SetGetItems@0x819648 body — tool-limitation residual same as v83/v95)` | ✅ |  |
| 2 | byte | byte `slotCount (dispatcher; tool-limitation residual)` | ✅ |  |
| 3 | int64 | int64 `tab-flag bitmask (8 bytes via DecodeBuffer; &v22, line 37)` | ✅ |  |
| 4 | int32 | int32 `meso (v2[67]; ONLY if flag&2, line 38)` | ✅ |  |
| 5 | byte | byte `PER-TAB count byte; repeated once per set tab bit (4/8/16/32/64 = i=1..5, lines 46-152), each followed by count*GW_ItemSlotBase::Decode. Per-tab segmentation identical to v83/v95 — unconditional Show fix holds.` | ✅ |  |
| 6 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

