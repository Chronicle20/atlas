# Category 1 Decode Completion — Context

Last Updated: 2026-03-11

## Key Files

### atlas-packet (library — primary target)
| File | Role | Current State |
|------|------|--------------|
| `libs/atlas-packet/character/info.go` | CharacterInfo packet | All typed fields, no-op Decode |
| `libs/atlas-packet/character/spawn.go` | CharacterSpawn packet | Typed models (cts, avatar, pets), no-op Decode |
| `libs/atlas-packet/field/set_field.go` | SetField packet | `characterInfoBytes []byte`, no-op Decode |
| `libs/atlas-packet/cash/shop_open.go` | CashShopOpen packet | `characterInfoBytes []byte`, no-op Decode |
| `libs/atlas-packet/interaction/interaction_writer.go` | InteractionEnter + EnterResultSuccess | `visitorBytes []byte` + `roomBytes []byte`, no-op Decode |
| `libs/atlas-packet/model/pet.go` | Pet model | Has Encode, NO Decode |
| `libs/atlas-packet/model/character_temporary_stat.go` | CTS model | Has EncodeForeign, NO DecodeForeign |
| `libs/atlas-packet/model/avatar.go` | Avatar model | Has Encode AND Decode (working) |
| `libs/atlas-packet/model/asset.go` | Asset model | Has Encode AND Decode (working) |
| `libs/atlas-packet/tool/uint128.go` | 128-bit mask operations | And, Or, ShiftLeft, ShiftRight (complete) |
| `libs/atlas-packet/character/buff_give_writer.go` | BuffGive + BuffGiveForeign | Uses CTS.Encode/EncodeForeign, no-op Decode |
| `libs/atlas-packet/character/buff_cancel_writer.go` | BuffCancelW + BuffCancelForeign | Uses CTS.EncodeMask, no-op Decode |
| `libs/atlas-packet/stat/changed.go` | StatChanged | Mask-driven stat encoding, no-op Decode |
| `libs/atlas-constants/job/model.go` | Job + IsFourthJob | `IdFromSkillId()` + `IsFourthJob()` (existing) |

### atlas-channel (service — adapters to update)
| File | Role | Current State |
|------|------|--------------|
| `services/atlas-channel/.../socket/writer/set_field.go` | WriteCharacterInfo + sub-functions | ~400 lines of character data encoding |
| `services/atlas-channel/.../socket/writer/cash_shop_open.go` | CashShopOpen adapter | Calls WriteCharacterInfo, passes bytes |
| `services/atlas-channel/.../socket/writer/character_spawn.go` | CharacterSpawn adapter | Builds typed models, passes to atlas-packet |
| `services/atlas-channel/.../socket/writer/character_interaction.go` | Interaction adapter | Pre-encodes visitor/room bytes, passes to atlas-packet |
| `services/atlas-channel/.../socket/model/mini_room.go` | MiniRoom + visitor interfaces | Polymorphic Enter() implementations |
| `services/atlas-channel/.../socket/model/mini_game_record.go` | MiniGameRecord | 5×uint32 fixed format |

## Key Decisions

### 1. CharacterInfo Decode — Pet slot iteration
- Encode iterates slots 0,1,2 and writes `true` + pet data OR nothing (no `false` marker for absent slots)
- Wait — actually looking at the code more carefully: the loop writes either `true` + data for present pets, but there's NO `false` written for absent slots. The `WriteBool(false)` at line 82 is for "more pets" after slot 2.
- Correction: Re-reading the code — the loop runs for slots 0-2 but only writes data for found pets. There is NO bool written for empty slots. After the loop, `WriteBool(false)` terminates.
- This means the pet section is: [for each occupied slot: true(1) + data] + false(1) terminator
- Actually wait — the loop writes nothing if a slot is empty. So if slot 0 has a pet and slots 1,2 don't, the wire has: true + pet0Data + false. If all 3 slots are occupied: true + pet0 + true + pet1 + true + pet2 + false.
- **Decode approach:** Read bool; while true, read pet data (slot can be inferred from order 0,1,2 but we need to track which slot); read until false.
- Problem: The slot is NOT written to the wire — the Encode just writes the pets in slot order. For Decode, we need to assign slots. Since Encode iterates 0,1,2 in order and only writes present ones, the order IS the slot order. But we can't know which slot each pet belongs to.
- **Resolution:** For Decode, assign sequential slots starting from 0. Round-trip won't be exact for sparse pet arrays (e.g., pet in slot 2 only). However, the current Encode always writes pets in slot 0,1,2 order and only writes present ones, so Decode should assign them to sequential slots. The round-trip test should use contiguous slots (0, then 0+1, then 0+1+2).
- Actually, looking more carefully: the bool is only written when a pet IS found. Empty slots produce NO output at all. So the wire format is simply: [true + petData]* + false. Decode reads bools until false. Each true-marked entry is a pet. The Slot field in InfoPet is used for slot lookup during Encode but is NOT on the wire for this packet.
- **Test strategy:** Use pets with slots 0,1,2 in order. On Decode, assign slot by position (0-indexed from first pet encountered).

### 2. CharacterSpawn — enteringField
- enteringField controls `y-42` and `stance=6` during Encode. These values are on the wire but the flag isn't.
- **Decision:** Decode reads x, y, stance as literal wire values. `enteringField` defaults to `false` after Decode.
- Round-trip test uses `enteringField=false` to get exact identity.
- Additional test verifies: Encode with enteringField=true produces expected y-42 and stance=6.

### 3. CharacterTemporaryStat.DecodeForeign — Mask-driven dispatch
- The 4×uint32 mask identifies which stats are present
- Need to iterate all registered stat types (in shift order), check if bit is set, then read the foreign value
- Foreign value sizes: NoOp=0, Byte=1, Short=2, Int=4, LevelSource=4, ValueSourceLevel=6
- **Decision:** Add a `foreignValueSize` field to each stat type registration. DecodeForeign reads mask, iterates types in order, reads sized values for set bits.
- Alternative: Add `ForeignValueReader` function that reads from `request.Reader`. This is more symmetric with the writer approach.
- **Chosen:** Add `ForeignValueReader` (symmetric with `ForeignValueWriter`).
- **CONFIRMED SAFE:** All 7 base stats (EnergyCharge, DashSpeed, DashJump, MonsterRiding, SpeedInfusion, HomingBeacon, Undead) use NoOpForeignValueWriter (0 bytes). The mask always includes these 7 bits, but they contribute 0 bytes to the foreign value section. No ambiguity during decode.

### 4. CharacterData — Inventory termination pattern
- Equipment lists use a position/slot byte as the first byte of each entry. Position 0 signals end of list.
- For GMS>28, the terminator is a uint16(0) instead of byte(0).
- **CRITICAL:** model.Asset.Decode does NOT read the slot position. Asset.Encode writes slot via `encodeSlot()` as the first field, but Asset.Decode starts after the slot. Existing callers (Storage Show/UpdateAssets) use `zeroPosition=true` which skips slot writing entirely.
- **Decision:** InventoryData.Decode reads the slot byte/short at inventory level FIRST. If zero, it's the terminator. Otherwise, dispatch to Asset.Decode for the remaining fields.
- **ASYMMETRY FOUND:** Equipment sections (regularEquip, cashEquip) use WriteShort(0) terminator for GMS>28. But the equipable inventory section uses WriteInt(0) as terminator despite WriteShort slot prefix — this is unexplained. A byte-comparison test must verify this before implementing Decode (Phase 9.0).

### 5. CharacterData — Skill 4th job detection
- WriteSkillInfo writes `masterLevel` only if `s.IsFourthJob()` is true
- **CONFIRMED:** `IsFourthJob()` already exists in `libs/atlas-constants/job/model.go`. Uses `job.IdFromSkillId(skillId)` (which does `math.Floor(skillId / 10000)`) then checks against 23 enumerated fourth job IDs.
- **Decision:** Include `FourthJob bool` in `SkillEntry`. During Encode, use it to conditionally write masterLevel. During Decode, use `job.IdFromSkillId(skillId)` + `job.IsFourthJob()` to detect — this exactly mirrors the service-layer logic. No duplication needed.

### 5b. CharacterData — Time-dependent values
- Fields like `getTime(-2)`, `msTime()`, cooldown durations are computed from `time.Now()` in the service layer.
- **Decision:** CharacterData stores these as raw `int64` wire values, NOT Go `time.Time`. This ensures exact round-trip. The `Expiration` field in SkillEntry and `CompletedAt` in QuestCompleted are int64.

### 6. Interaction visitor type discrimination
- InteractionEnter sends a single visitor. The wire doesn't explicitly tag the visitor type.
- MerchantOwnerVisitor: writes slot=0 + itemId(uint32) + merchantName(string)
- MiniRoomVisitorBase: writes slot + avatar bytes + name(string)
- The Avatar.Encode starts with gender(byte) which is 0 or 1, and MerchantOwner writes slot=0 then a uint32 itemId.
- These ARE distinguishable on wire IF we know the room context, but InteractionEnter doesn't carry room type.
- **Decision:** Add a `visitorType` field to the InteractionEnter struct. Service adapter sets it based on room context. This field is encoded as part of the wire format (1 extra byte after mode). Wait — that changes the wire format. Instead: keep visitorType as a construction-time parameter that controls Encode/Decode dispatch but is NOT written to wire. For Decode, the caller must know the visitor type to call the right variant. OR: include it as the first byte after mode.
- **Better decision:** Since the visitor bytes were already opaque, we can change the struct signature without changing the wire format. The visitor type is set at construction and stored in the struct. For standalone round-trip testing, we encode the visitor type as part of the test setup. The actual wire format remains: mode + visitor bytes. The Decode takes a `visitorType` hint parameter (or it's set on the struct before calling Decode).
- **Final decision:** Use a variant approach. Keep InteractionEnter simple but add a constructor per visitor type:
  - `NewInteractionEnterBase(mode, slot, avatar, name)`
  - `NewInteractionEnterGame(mode, slot, avatar, name, record)`
  - `NewInteractionEnterMerchant(mode, itemId, merchantName)`
  Each stores the visitor type internally. Decode reads based on stored type. For standalone Decode (from raw bytes), the caller sets visitorType first.

### 7. Interaction room type discrimination
- First byte of room bytes IS the room type (1,2,4,5). Decode dispatches on this.
- Visitor count within rooms: visitors are written until 0xFF sentinel. Decode reads visitors until byte=0xFF.
- Visitor type within rooms is determined by room type:
  - Game rooms (1,2): all visitors are MiniGameRoomVisitor (base + record)
  - Personal shop (4): all visitors are MiniRoomVisitorBase (slot + avatar + name)
  - Merchant shop (5): first visitor is MerchantOwnerVisitor (slot=0 + itemId + name), rest are base
- **Decision:** Room.Decode dispatches on room type byte, then decodes visitors using room-appropriate visitor type.

## Version Branches

Many encoding sections have version-conditional branches. Key patterns:

| Condition | Meaning |
|-----------|---------|
| GMS && MajorVersion > 28 | Post-beta GMS (most common branch) |
| GMS && MajorVersion > 83 | Post-Big Bang GMS |
| GMS && MajorVersion > 87 | Late v83+ GMS |
| GMS && MajorVersion <= 12 | Very early GMS |
| JMS | Japanese MapleStory |

Test variants cover: GMS v28, GMS v83, GMS v95 (from `test.Variants`).

## Dependency Graph

```
Phase 6: CharacterInfo.Decode (standalone)
    |
Phase 7A: Pet.Decode (standalone)
Phase 7B: CTS.DecodeForeign (standalone)
    |           |
    +-----------+
          |
Phase 8: CharacterSpawn.Decode (needs 7A + 7B)
Phase 8B: BuffGive/Cancel Decode (needs 7B — CTS Decode/DecodeForeign)
Phase 8C: StatChanged Decode (standalone — uses options context)

Phase 9.0: Inventory byte-comparison test (standalone — prerequisite for 9.6)
Phase 9A-C: CharacterData struct (standalone, uses existing Asset.Decode)
    |
Phase 9D: SetField.Decode (needs 9A-C)
Phase 9E: CashShopOpen.Decode (needs 9A-C)
Phase 9F: Service adapter updates (needs 9D + 9E)

Phase 10A: Visitor types (standalone, uses existing Avatar.Decode)
Phase 10B: Room types (needs 10A, uses existing Asset.Decode)
    |
Phase 10C: InteractionEnter.Decode (needs 10A)
Phase 10D: InteractionEnterResultSuccess.Decode (needs 10B)
Phase 10E: Service adapter updates (needs 10C + 10D)

Phase 11: Verification (needs all above)
```
