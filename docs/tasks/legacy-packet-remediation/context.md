# Packet Remediation - Context & Key Files

Last Updated: 2026-03-12

## Branch
`merchant-service`

## Key Files

### Phase 1: world_message.go Default Branch
- **Service writer**: `services/atlas-channel/atlas.com/channel/socket/writer/world_message.go`
  - Lines 211-226: default branch with inline encoding for Unk3/4/7/8
  - Lines 128-178: operator functions to remove (Phase 4)
  - Lines 180-229: WorldMessageBody dispatcher
- **Packet structs**: `libs/atlas-packet/chat/world_message.go` - existing WorldMessage* structs
- **Packet extra**: `libs/atlas-packet/chat/world_message_extra.go` - WorldMessageWeather
- **Tests**: `libs/atlas-packet/chat/world_message_test.go` - existing round-trip tests
- **Caller**: `services/atlas-channel/.../kafka/consumer/gachapon/consumer.go` (operator pattern)
- **Caller**: `services/atlas-channel/.../kafka/consumer/system_message/consumer.go`

### Phase 2: Dead Duplicate Model Encoders
All confirmed dead code - `.Encoder()` methods unused:

| Service File | Status | atlas-packet Equivalent |
|---|---|---|
| `services/atlas-channel/.../socket/model/buddy.go` | Dead | `libs/atlas-packet/model/buddy.go` |
| `services/atlas-channel/.../socket/model/guild_member.go` | Dead | `libs/atlas-packet/model/guild_member.go` |
| `services/atlas-channel/.../socket/model/macros.go` | Dead | `libs/atlas-packet/model/macros.go` |
| `services/atlas-channel/.../socket/model/note.go` | Dead | `libs/atlas-packet/note/entry.go` |
| `services/atlas-channel/.../socket/model/pet.go` | Dead | `libs/atlas-packet/model/pet.go` |
| `services/atlas-channel/.../socket/model/mini_game_record.go` | Dead | embedded in `libs/atlas-packet/interaction/mini_room.go` |
| `services/atlas-login/.../socket/model/world_recommendation.go` | Dead | `libs/atlas-packet/model/world_recommendation.go` |

**Keep as-is** (not duplicates):
- `services/atlas-channel/.../socket/model/asset.go` - type alias + domain converter (already migrated)
- `services/atlas-channel/.../socket/model/mini_room.go` - proper abstraction boundary
- `services/atlas-channel/.../socket/model/cash_item_special.go` - unique to service
- `services/atlas-channel/.../socket/model/category_discount.go` - unique to service

### Phase 3: NPC Conversation Adapter
- **Service model**: `services/atlas-channel/.../socket/model/npc_conversation.go` (344 lines)
  - 15 detail types with inline Encode logic
  - `getNpcConversationMessageType()` for string-to-byte enum resolution
  - `NewNpcConversation()` constructor
- **Packet writer**: `libs/atlas-packet/npc/conversation_writer.go`
  - Takes byte msgType + []byte conversationDetail (pre-encoded)
  - Has Encode and Decode
- **Single caller**: `services/atlas-channel/.../kafka/consumer/npc/conversation/consumer.go`
  - 4 handler call sites (lines 137, 151, 164, 177)
  - Pattern: create detail → wrap in NpcConversation → `.Encoder()`

### Phase 4: Operator Indirection
- **Same file as Phase 1**: `services/atlas-channel/.../socket/writer/world_message.go`
- Functions to remove:
  - `NoOpOperator` (line 128)
  - `ItemIdOperator` (line 132)
  - `SlotAsIntOperator` (line 139)
  - `NPCIdOperator` (line 146)
  - `extractUint32FromOperator` (line 160)
  - `extractInt32FromOperator` (line 170)
- Callers that pass operators:
  - `WorldMessageBlueTextBody` → `ItemIdOperator(0)` or `ItemIdOperator(itemId)`
  - `WorldMessageNPCBody` → `NPCIdOperator(l)(npcId)`
  - `WorldMessageItemMegaphoneBody` → caller-provided operator (SlotAsIntOperator)
  - `WorldMessageGachaponMegaphoneBody` → caller-provided operator (Asset encoder)
- **Gachapon consumer**: `services/atlas-channel/.../kafka/consumer/gachapon/consumer.go:96-102`
  - Complex operator: writes Asset info to temp writer, value extracted as uint32 (itemId)

### Phase 5: Opcode Registry
- **New library**: `libs/atlas-opcodes/`
- **atlas-socket infrastructure**:
  - `libs/atlas-socket/writer/writer.go` - BodyFunc, Producer, MessageGetter, ProducerGetter
  - `libs/atlas-socket/handler/handler.go` - MessageValidator, MessageHandler, Adapter (generic)
  - `libs/atlas-socket/server.go` - OpWriter, OpReader, ShortReadWriter, ByteReadWriter
- **Configuration REST models**:
  - Handler: `services/atlas-configurations/.../tenants/socket/handler/rest.go`
  - Writer: `services/atlas-configurations/.../tenants/socket/writer/rest.go`
- **Service bootstrap (login)**: `services/atlas-login/.../login/main.go`
  - `produceWriters()` (lines 137-162) - list of available writer names
  - `getWriterProducer()` (lines 202-220) - builds name→BodyFunc map
  - `produceHandlers()` (lines 164-193) - map of handler name→function
  - `handlerProducer()` (lines 222-254) - builds opcode→Handler map
- **Service bootstrap (channel)**: `services/atlas-channel/.../channel/main.go` - identical patterns
- **JSON templates**: `services/atlas-configurations/seed-data/templates/template_gms_95_1.json`

### Phase 6: Attack & EffectSkillUse Decode
- **Attack**: `libs/atlas-packet/character/attack_writer.go`
  - Encode: lines 74-127 (complex conditional logic)
  - Decode: lines 129-134 (no-op)
  - Constructor flags: attackType, isMesoExplosion, hasKeydown, isStrafe
  - Wire format is NOT self-describing
- **EffectSkillUse**: `libs/atlas-packet/character/effect_skill_use.go`
  - Encode: uses conditional booleans (isBerserk, isDragonFury, isMonsterMagnet)
  - Decode: no-op (conditional bools not self-describing)
- **AttackInfo model**: `libs/atlas-packet/model/attack_info.go` (client-send, has full Decode)
- **DamageInfo model**: `libs/atlas-packet/model/damage_info.go` (has full Decode)
- **Test infrastructure**: `libs/atlas-packet/test/roundtrip.go`, `test/context.go`
- **Existing tests**: `libs/atlas-packet/character/effect_test.go`, `effect_foreign_test.go`

## Key Design Decisions

1. **Dead code deletion over refactoring**: Phase 2 models are unused - delete, don't refactor
2. **Constructor flags for non-self-describing Decode**: Attack/EffectSkillUse Decode requires pre-set flags
3. **Opcode registry is read-only**: loads once at startup, no runtime mutation
4. **NPC conversation keeps string enums in service**: resolution is a service concern
5. **WorldMessage operator replacement**: gachapon caller extracts itemId at call site

## Dependencies Between Phases

- Phase 4 depends on Phase 1 (same file, Phase 1 modifies switch, Phase 4 removes operators)
- All other phases are independent and can run in parallel
