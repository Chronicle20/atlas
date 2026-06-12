# StorageShow (← `CTrunkDlg::OnPacket#Show`)

- **IDA:** 0x76a990
- **Atlas file:** `../../libs/atlas-packet/storage/clientbound/show.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (22)` | ✅ |  |
| 1 | int32 | int32 `npcTemplateId (m_dwNpcTemplateID)` | ✅ |  |
| 2 | byte | byte `slotCount (m_nSlotCount)` | ✅ |  |
| 3 | int64 | int64 `tab-flag bitmask (8 bytes via DecodeBuffer; WriteLong-compatible width)` | ✅ |  |
| 4 | int32 | int32 `meso (m_nMoney; ONLY if flag&2)` | ✅ |  |
| 5 | byte | byte `PER-TAB count byte; repeated once per set tab bit (4/8/16/32/64), each followed by count*GW_ItemSlotBase::Decode` | ✅ |  |
| 6 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

