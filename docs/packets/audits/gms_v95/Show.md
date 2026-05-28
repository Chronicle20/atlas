# Show (← `CTrunkDlg::OnPacket#Show`)

- **IDA:** 0x76a990
- **Atlas file:** `../../libs/atlas-packet/storage/clientbound/show.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (22)` | ✅ |  |
| 1 | int32 | int32 `npcTemplateId (m_dwNpcTemplateID)` | ✅ |  |
| 2 | byte | byte `slotCount (m_nSlotCount)` | ✅ |  |
| 3 | int64 | int64 `tab-flag bitmask (8 bytes via DecodeBuffer; WriteLong-compatible width)` | ✅ |  |
| 4 | int32 | int32 `meso (m_nMoney; ONLY if flag&2)` | ✅ |  |
| 5 | int16 | byte `PER-TAB count byte; repeated once per set tab bit (4/8/16/32/64), each followed by count*GW_ItemSlotBase::Decode` | ❌ | width mismatch |
| 6 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 7 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 8 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 9 | byte | byte `` | ❌ | atlas: extra — client never reads this field |


> defer: ❌ real wire bug — per-tab item segmentation + 3 spurious padding bytes
> vs v95 `CTrunkDlg::SetGetItems`@0x76a390. Structural rewrite of a
> version-agnostic encoder; only v95 readable this session, so a blind change
> risks v83/v87/v92/v111/JMS185. See `docs/packets/ida-exports/_pending.md` →
> "Show clientbound — per-tab item segmentation + spurious padding (storage)".
