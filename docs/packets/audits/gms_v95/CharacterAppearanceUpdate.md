# CharacterAppearanceUpdate (← `CUserRemote::OnAvatarModified`)

- **IDA:** 0x954110
- **Atlas file:** `libs/atlas-packet/character/clientbound/appearance_update.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId — read by CUserPool::OnUserRemotePacket before dispatch` | ✅ |  |
| 1 | byte | byte `v4 flags byte: bit0=avatarLook, bit1=speed, bit2=carryItem` | ✅ |  |
| 2 | byte | bytes `AvatarLook::Decode — full avatar look data (guard: v4 & 1)` | ❌ | width mismatch |
| 3 | byte | byte `nSpeed (guard: v4 & 2)` | ✅ |  |
| 4 | int32 | byte `nCarryItemEffect (guard: v4 & 4)` | ❌ | width mismatch |
| 5 | byte | byte `bCouple flag` | ✅ |  |
| 6 | int32 | bytes `liCoupleItemSN (8 bytes) + liPairItemSN (8 bytes) + dwPairCharacterId (4 bytes)` | ❌ | width mismatch |
| 7 | byte | byte `bFriendship flag` | ✅ |  |
| 8 | int32 | bytes `liFriendshipItemSN (8 bytes) + liFriendshipPairItemSN (8 bytes) + dwFriendCharacterId (4 bytes)` | ❌ | width mismatch |
| 9 | byte | byte `bMarriage flag` | ✅ |  |
| 10 | byte | int32 `dwMarriageCharacterID (guard: bMarriage)` | ❌ | width mismatch |
| 11 | int32 | int32 `dwMarriagePairCharacterID (guard: bMarriage)` | ✅ |  |
| 12 | byte | int32 `nWeddingRingID (guard: bMarriage)` | ❌ | width mismatch |
| 13 | int32 | int32 `nCompletedSetItemID` | ✅ |  |
| 14 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 15 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 16 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 17 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 18 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 19 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 20 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

---

**ack: tool-limitation false positive**

The IDA export models `AvatarLook::Decode` as a single `DecodeBuf` placeholder
(row 2), while the atlas analyzer fully expands `m.avatar.Encode(l, ctx)(options)`
via `WriteByteArray` recursion into Avatar's complete field sequence. This creates
an asymmetry: IDA has fewer rows than atlas, so the diff aligns incorrectly and
generates "atlas: extra" warnings for rows 14–20.

The actual wire format is correct. `CharacterAppearanceUpdate.Encode` sends:
characterId (int32) | flags=1 (byte) | avatar block (WriteByteArray) |
couple=0 (byte) | friendship=0 (byte) | marriage=0 (byte) | completedSetItemId=0 (int32).

This matches `CUserRemote::OnAvatarModified` with `(v4 & 1) != 0` and all ring
flags = 0. Rows 10, 12 show `byte` vs `int32` mismatches at the ring guards
due to alignment skew from row 2's over-counting. Resolution requires expanding
the IDA export with AvatarLook sub-fields; deferred to Phase 3.
