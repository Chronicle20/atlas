# CharacterAppearanceUpdate (← `CUserRemote::OnAvatarModified`)

- **IDA:** 0x98367e
- **Atlas file:** `../../libs/atlas-packet/character/clientbound/appearance_update.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId — read by CUserPool::OnUserRemotePacket before dispatch` | ✅ |  |
| 1 | byte | byte `v4 flags byte: bit0=avatarLook, bit1=speed, bit2=carryItem` | ✅ |  |
| 2 | bytes | bytes `AvatarLook::Decode — full avatar look data (guard: v4 & 1)` | ✅ |  |
| 3 | byte | byte `nSpeed (guard: v4 & 2)` | ✅ |  |
| 4 | byte | byte `nCarryItemEffect (guard: v4 & 4)` | ✅ |  |
| 5 | byte | byte `bCouple flag` | ✅ |  |
| 6 | int32 | bytes `liCoupleItemSN (8 bytes) + liPairItemSN (8 bytes) + dwPairCharacterId (4 bytes)` | ✅ |  |
| 7 | byte | byte `bFriendship flag` | ❌ | atlas: short — missing trailing field |
| 8 | byte | bytes `liFriendshipItemSN (8 bytes) + liFriendshipPairItemSN (8 bytes) + dwFriendCharacterId (4 bytes)` | ❌ | atlas: short — missing trailing field |
| 9 | byte | byte `bMarriage flag` | ❌ | atlas: short — missing trailing field |
| 10 | byte | int32 `dwMarriageCharacterID (guard: bMarriage)` | ❌ | atlas: short — missing trailing field |
| 11 | byte | int32 `dwMarriagePairCharacterID (guard: bMarriage)` | ❌ | atlas: short — missing trailing field |
| 12 | byte | int32 `nWeddingRingID (guard: bMarriage)` | ❌ | atlas: short — missing trailing field |

