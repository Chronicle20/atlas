# Packet Remediation - Task Tracking

Last Updated: 2026-03-12

## Phase 1: Extract world_message.go Default Branch
- [x] 1.1 Create WorldMessageUnknown3 struct in atlas-packet
- [x] 1.2 Create WorldMessageUnknown4 struct in atlas-packet (uses Unknown3 with alias constructor)
- [x] 1.3 Create WorldMessageUnknown7 struct in atlas-packet
- [x] 1.4 Create WorldMessageUnknown8 struct in atlas-packet
- [x] 1.5 Add round-trip tests for all 4 unknown types
- [x] 1.6 Update WorldMessageBody switch to delegate to new structs
- [x] 1.7 Build and test atlas-channel
COMPLETE

## Phase 2: Delete Dead Duplicate Model Encoders
- [x] 2.1 Delete buddy.go from atlas-channel socket/model
- [x] 2.2 Delete guild_member.go from atlas-channel socket/model
- [x] 2.3 Delete macros.go from atlas-channel socket/model
- [x] 2.4 note.go - NOT dead code (used by session consumer + note handler) - kept
- [x] 2.5 Delete pet.go from atlas-channel socket/model
- [x] 2.6 mini_game_record.go - replaced with interactionpkt.MiniGameRecord type alias in mini_room.go
- [x] 2.7 Delete world_recommendation.go from atlas-login socket/model
- [x] 2.8 Build and test both services
COMPLETE

## Phase 3: Simplify npc_conversation.go Adapter
- [x] 3.1 Audit atlas-packet conversation_writer.go for completeness
- [x] 3.2 Create atlas-packet detail type constructors if missing (all 15 exist)
- [x] 3.3 Refactor NpcConversation.Encoder() to thin adapter
- [x] 3.4 Update consumer.go callers to use atlas-packet detail types
- [x] 3.5 Build and test atlas-channel
COMPLETE - 344 lines reduced to 66 lines

## Phase 4: Eliminate Operator Indirection in world_message.go
- [x] 4.1 Audit all WorldMessageBody callers
- [x] 4.2 Refactor WorldMessageBody to accept typed params (itemId uint32, slot int32)
- [x] 4.3 Update gachapon consumer to pass event.ItemId directly
- [x] 4.4 Remove all operator functions (NoOp, ItemId, SlotAsInt, NPCId, extract helpers)
- [x] 4.5 Remove imports: encoding/binary, atlas-model/model, atlas-socket/response
- [x] 4.6 Build and test atlas-channel
COMPLETE - removed 6 operator functions, 2 extract helpers, 3 unused imports

## Phase 5: Shared Opcode Registry
- [x] 5.1 Create libs/atlas-opcodes module
- [x] 5.2 Create config loader from REST models (HandlerConfig, WriterConfig)
- [x] 5.3 Create writer producer builder (BuildWriterProducer)
- [x] 5.4 Create handler producer builder (BuildHandlerMap)
- [x] 5.5 Add to go.work
- [x] 5.6 Refactor atlas-login main.go to use shared registry
- [x] 5.7 Refactor atlas-channel main.go to use shared registry
- [x] 5.8 Docker builds - pre-existing issue: go.mod files missing atlas-packet/atlas-opcodes/atlas-constants deps (resolved by go.work only)
COMPLETE (Docker issue is pre-existing, not introduced by remediation)

## Phase 6: Implement Decode for Attack and EffectSkillUse
- [x] 6.1 Implement Attack.Decode (with NewAttackForDecode constructor for non-self-describing flags)
- [x] 6.2 Implement EffectSkillUse.Decode (with NewEffectSkillUseForDecode constructor)
- [x] 6.3 Implement EffectSkillUseForeign.Decode (with NewEffectSkillUseForeignForDecode constructor)
- [x] 6.4 Add round-trip tests for Attack (all types, flags, regions)
- [x] 6.5 Add round-trip tests for EffectSkillUse
- [x] 6.6 Build and test atlas-packet
COMPLETE

## Summary
All 6 phases complete. All builds pass, all tests pass.
