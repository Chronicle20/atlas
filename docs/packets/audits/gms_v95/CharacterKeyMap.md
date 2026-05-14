# CharacterKeyMap (← `CFuncKeyMappedMan::OnInit`)

- **IDA:** 0x568c30
- **Atlas file:** `libs/atlas-packet/character/clientbound/keymap.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `resetToDefault flag (non-zero = reset to defaults; zero = read key bindings from packet)` | ✅ |  |
| 1 | byte | byte `FUNCKEY_MAPPED::nType (key type byte; loop 90 entries, only if resetToDefault==0 and packet length >= 0x1BD)` | ✅ |  |
| 2 | byte | int32 `FUNCKEY_MAPPED::nID (key action int32; loop 90 entries, only if resetToDefault==0)` | ❌ | width mismatch |
| 3 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

## ack

❌ verdict is a **tool-limitation false positive (loop linearization)** — no wire bug present.

`CFuncKeyMappedMan::OnInit` reads:
- `Decode1` (resetToDefault flag)  
- then if `resetToDefault==0`: 90 × (`FUNCKEY_MAPPED::Decode` = `Decode1(nType)` + `Decode4(nID)`)

Atlas `CharacterKeyMap.Encode` writes:
- `WriteByte(resetToDefault 0/1)`
- then if not resetToDefault: 90 × (`WriteInt8(KeyType)` + `WriteInt32(KeyAction)`)

`WriteInt8` is 1 byte (matches `Decode1`) and `WriteInt32` is 4 bytes (matches `Decode4`). Wire formats are identical. The flat IDA export has only 3 entries (Decode1 + 2 loop-body entries), while the atlas encoder emits 1+90×2=181 sequential calls. The diff tool aligns the 3 IDA reads against the first 3 atlas writes, producing spurious mismatches for rows 2–5.

Resolution: loop-aware analyzer in Phase 3.

Causes: (1) loop not modelable in flat IDA export; (2) sequential diff of loop body against repeated atlas writes produces false width mismatch.
