# Atlas Packet Remediation Plan

Last Updated: 2026-03-12

## Executive Summary

Complete the writer packet extraction by eliminating remaining inline encoding in services, removing dead duplicate model files, simplifying the NPC conversation adapter, cleaning up the operator indirection hack, creating a shared opcode registry for test client support, and implementing Decode for the remaining server-send-only packets. This work finishes the atlas-packet separation and enables client emulation for integrated testing.

## Current State

- ~325 packet structs extracted to `libs/atlas-packet` across 23 packages
- All structs implement unified `Packet` interface (Operation, String, Encode, Decode)
- atlas-login: fully extracted, thin adapters only
- atlas-channel: 98% extracted, residual inline encoding in world_message.go default branch
- 7 service-layer socket/model files duplicate encoding logic already present in atlas-packet (most are dead code)
- npc_conversation.go (344 lines) duplicates all 15 detail type encoders
- world_message.go uses operator indirection hack to pass typed values through `model.Operator[*response.Writer]`
- Opcode tables are built per-service at startup from atlas-configurations REST data; no shared loader exists
- 3 Decode methods are intentionally no-op (Attack, EffectSkillUse, EffectSkillUseForeign) due to non-self-describing wire format

## Proposed Future State

- Zero inline encoding in any service writer or model file
- All duplicate service model encoders deleted (dead code cleanup)
- npc_conversation.go reduced to thin adapter (~50 lines): string enum resolution + delegation
- world_message.go uses direct typed parameters, no operator indirection
- Shared `libs/atlas-opcodes` provides reusable opcode table loader for services and test clients
- Attack and EffectSkillUse have working Decode methods with constructor-flag-driven parsing

---

## Phase 1: Extract world_message.go Default Branch

### Goal
Move the 4 remaining inline-encoded WorldMessage modes (Unk3, Unk4, Unk7, Unk8) from the service into atlas-packet structs.

### Current State
`services/atlas-channel/.../socket/writer/world_message.go:211-226` has a `default` branch using `response.NewWriter` directly for 4 unknown modes. All other 13 modes already delegate to atlas-packet.

### Tasks

#### 1.1 Create WorldMessageUnknown3 struct in atlas-packet (S)
- File: `libs/atlas-packet/chat/world_message.go` (or new file `world_message_unknown.go`)
- Fields: mode byte, message string, unknownField string, operatorValue uint32
- Encode: mode, message, "doo", operatorValue
- Decode: inverse
- Constructor: `NewWorldMessageUnknown3(mode, message, unknownField, operatorValue)`
- Acceptance: struct compiles, has Operation/String/Encode/Decode

#### 1.2 Create WorldMessageUnknown4 struct (S)
- Same wire format as Unk3 (mode, message, "doo", operatorValue)
- Can share struct with Unk3 if format is identical, differentiated by Operation() return

#### 1.3 Create WorldMessageUnknown7 struct (S)
- Fields: mode byte, message string
- Encode: mode, message, int(0)
- Decode: inverse

#### 1.4 Create WorldMessageUnknown8 struct (S)
- Fields: mode byte, message string, channelId byte, whispersOn bool
- Encode: mode, message, channelId, whispersOn
- Decode: inverse

#### 1.5 Add round-trip tests for all 4 unknown types (S)
- Follow existing pattern in `chat/world_message_test.go`
- Test across all region/version variants

#### 1.6 Update WorldMessageBody switch to delegate (S)
- Replace default branch with explicit cases for Unk3, Unk4, Unk7, Unk8
- Each case creates atlas-packet struct and calls `.Encode(l, ctx)(options)`
- Remove `response.NewWriter` import if no longer needed

#### 1.7 Build and test atlas-channel (S)
- `go build` and `go test ./...` in atlas-channel

### Dependencies: None
### Risk: Low - mechanical extraction

---

## Phase 2: Delete Dead Duplicate Model Encoders

### Goal
Remove 7 service-layer model files that duplicate encoding logic already in atlas-packet. Investigation confirmed most are dead code (unused `.Encoder()` methods).

### Current State
Services already use atlas-packet types directly. The service model files with `.Encoder()` methods are vestigial.

### Tasks

#### 2.1 Delete buddy.go from atlas-channel socket/model (S)
- File: `services/atlas-channel/.../socket/model/buddy.go`
- Confirmed dead code: no callers of `Buddy.Encoder()`
- atlas-packet equivalent: `libs/atlas-packet/model/buddy.go`
- Verify no imports reference this type, then delete
- Build atlas-channel

#### 2.2 Delete guild_member.go from atlas-channel socket/model (S)
- File: `services/atlas-channel/.../socket/model/guild_member.go`
- Confirmed dead code: service constructs `guildpkt.GuildMemberInfo` directly
- atlas-packet equivalent: `libs/atlas-packet/model/guild_member.go`
- Delete, verify build

#### 2.3 Delete macros.go from atlas-channel socket/model (S)
- File: `services/atlas-channel/.../socket/model/macros.go`
- Confirmed dead code: service uses `packetmodel.NewMacro/NewMacros` + `.Encode()` directly
- atlas-packet equivalent: `libs/atlas-packet/model/macros.go`
- Delete, verify build

#### 2.4 Delete note.go from atlas-channel socket/model (S)
- File: `services/atlas-channel/.../socket/model/note.go`
- Confirmed dead code: service converts to `notepkt.NoteEntry` via struct literal
- atlas-packet equivalent: `libs/atlas-packet/note/entry.go`
- Delete, verify build

#### 2.5 Delete pet.go from atlas-channel socket/model (S)
- File: `services/atlas-channel/.../socket/model/pet.go`
- Confirmed dead code: service constructs `packetmodel.Pet` directly
- atlas-packet equivalent: `libs/atlas-packet/model/pet.go`
- Delete, verify build

#### 2.6 Delete mini_game_record.go from atlas-channel socket/model (S)
- File: `services/atlas-channel/.../socket/model/mini_game_record.go`
- Confirmed dead code: type embedded in interaction package
- atlas-packet equivalent: embedded in `libs/atlas-packet/interaction/mini_room.go`
- Delete, verify build

#### 2.7 Delete world_recommendation.go from atlas-login socket/model (S)
- File: `services/atlas-login/.../socket/model/world_recommendation.go`
- Service constructs `packetmodel.WorldRecommendation` via conversion
- atlas-packet equivalent: `libs/atlas-packet/model/world_recommendation.go`
- Verify callers use atlas-packet type, delete, verify build

#### 2.8 Verify both service builds pass (S)
- `go build` for atlas-channel and atlas-login
- `go test ./...` for both

### Dependencies: None (parallel with Phase 1)
### Risk: Low - deleting confirmed dead code. Build verification catches any missed references.

---

## Phase 3: Simplify npc_conversation.go Adapter

### Goal
Reduce the 344-line `npc_conversation.go` to a thin adapter that keeps string-to-byte enum resolution and delegates all 15 detail type encoders to atlas-packet.

### Current State
- Service: `services/atlas-channel/.../socket/model/npc_conversation.go` - 344 lines, 15 detail types
- Packet: `libs/atlas-packet/npc/conversation_writer.go` - has matching structs taking byte msgType + []byte detail
- Single caller: `services/atlas-channel/.../kafka/consumer/npc/conversation/consumer.go`
- Pattern: consumer creates detail struct, wraps in NpcConversation, calls `.Encoder()`

### Tasks

#### 3.1 Audit atlas-packet conversation_writer.go for completeness (S)
- Verify all 15 service detail types have atlas-packet equivalents
- Note: AskYesNoQuest may be missing - check and add if needed
- Document any interface gaps

#### 3.2 Create atlas-packet detail type constructors if missing (M)
- For each service detail type not already in atlas-packet, add it
- Each must implement Encode and Decode
- Add round-trip tests

#### 3.3 Refactor NpcConversation.Encoder() to thin adapter (M)
- Keep: `getNpcConversationMessageType()` for string-to-byte enum resolution
- Keep: `NewNpcConversation()` constructor with string enum msgType
- Change: `Encoder()` body to:
  1. Resolve string msgType to byte
  2. Call `detail.Encode(l, ctx)(options)` to get `[]byte`
  3. Create `npcpkt.NewNpcConversation(speakerTypeId, speakerTemplateId, msgTypeByte, param, secondaryNpcId, detailBytes)`
  4. Return `npcpkt.Encode(l, ctx)(options)`
- Remove: all 15 inline detail encoder implementations

#### 3.4 Update consumer.go callers if interface changed (S)
- File: `services/atlas-channel/.../kafka/consumer/npc/conversation/consumer.go`
- Verify all 4 handler call sites still work with refactored adapter

#### 3.5 Build and test atlas-channel (S)
- `go build` and `go test ./...`

### Dependencies: None (parallel with Phases 1-2)
### Risk: Medium - NPC conversations are gameplay-critical. Test thoroughly.

---

## Phase 4: Eliminate Operator Indirection in world_message.go

### Goal
Replace the `model.Operator[*response.Writer]` indirection with direct typed parameters. Remove `extractUint32FromOperator`, `extractInt32FromOperator`, and all operator factory functions.

### Current State
- `WorldMessageBody()` accepts `operator model.Operator[*response.Writer]`
- Callers pass `ItemIdOperator(itemId)`, `SlotAsIntOperator(slot)`, `NPCIdOperator(npcId)`, or `NoOpOperator`
- `extractUint32FromOperator` / `extractInt32FromOperator` encode to temp buffer then read back as integers
- Gachapon caller in `kafka/consumer/gachapon/consumer.go` passes a complex operator that encodes an Asset

### Tasks

#### 4.1 Audit all WorldMessageBody callers (S)
- Grep for `WorldMessageBody`, `WorldMessage*Body` across atlas-channel
- Identify every operator usage pattern
- Special attention to gachapon consumer's Asset operator

#### 4.2 Refactor WorldMessageBody to accept typed optional params (M)
- Replace `operator model.Operator[*response.Writer]` with explicit typed fields
- Options: `itemId uint32`, `slot int32`, `npcId uint32`
- For gachapon: need `itemId uint32` extracted at call site, not via operator
- Update all convenience functions (`WorldMessageBlueTextBody`, etc.)

#### 4.3 Update gachapon consumer caller (M)
- File: `services/atlas-channel/.../kafka/consumer/gachapon/consumer.go`
- Extract itemId at call site instead of passing Asset encoder as operator
- Pass itemId directly to `WorldMessageGachaponMegaphoneBody`

#### 4.4 Remove dead operator functions (S)
- Delete: `NoOpOperator`, `ItemIdOperator`, `SlotAsIntOperator`, `NPCIdOperator`
- Delete: `extractUint32FromOperator`, `extractInt32FromOperator`
- Remove `encoding/binary` and `atlas-socket/response` imports if no longer needed

#### 4.5 Build and test atlas-channel (S)
- `go build` and `go test ./...`

### Dependencies: Phase 1 must complete first (same file)
### Risk: Low-medium. Gachapon consumer is the complex case.

---

## Phase 5: Shared Opcode Registry

### Goal
Create `libs/atlas-opcodes` that loads opcode tables from atlas-configurations REST data, builds bidirectional maps (operation string <-> opcode uint16), and can be used by both services and test clients.

### Current State
- Opcodes come from JSON templates in `atlas-configurations` seed data
- Each service fetches tenant config via REST: `/api/configurations/tenants/{tenantId}`
- Services build handler/writer maps manually in `main.go`
- atlas-socket provides: `MessageGetter`, `ProducerGetter`, `BodyFunc`, `Producer`
- Writer config: `{opCode: "0x0C", writer: "ServerIP", options: {...}}`
- Handler config: `{opCode: "0x01", validator: "NoOpValidator", handler: "LoginHandle", options: {...}}`

### Tasks

#### 5.1 Create libs/atlas-opcodes module (M)
- `go.mod` with `module github.com/Chronicle20/atlas-opcodes`
- Core types:
  ```go
  type HandlerEntry struct { OpCode uint16; Name string; Options map[string]interface{} }
  type WriterEntry struct { OpCode uint16; Name string; Options map[string]interface{} }
  type Registry struct { handlers []HandlerEntry; writers []WriterEntry }
  ```
- Bidirectional lookups:
  ```go
  func (r Registry) WriterOpCode(name string) (uint16, bool)
  func (r Registry) WriterName(opcode uint16) (string, bool)
  func (r Registry) HandlerOpCode(name string) (uint16, bool)
  func (r Registry) HandlerName(opcode uint16) (string, bool)
  func (r Registry) WriterOptions(name string) map[string]interface{}
  func (r Registry) HandlerOptions(name string) map[string]interface{}
  ```

#### 5.2 Create config loader from REST models (M)
- Parse handler/writer REST models into Registry
- Reuse existing `configuration/tenant/socket/handler/rest.go` and `writer/rest.go` types
- Or define portable equivalents in atlas-opcodes

#### 5.3 Create writer producer builder (M)
- Extract common pattern from both services' `getWriterProducer()`
- `func BuildWriterProducer(registry Registry, availableWriters []string, opWriter OpWriter) writer.Producer`
- Uses atlas-socket's `MessageGetter` and `ProducerGetter`

#### 5.4 Create handler producer builder (M)
- Extract common pattern from both services' `handlerProducer()`
- Generic over session type: `func BuildHandlerProducer[S any](registry Registry, adapter Adapter[S], validators map[string]MessageValidator[S], handlers map[string]MessageHandler[S]) map[uint16]request.Handler`

#### 5.5 Add to go.work (S)
- Add `libs/atlas-opcodes` to workspace

#### 5.6 Refactor atlas-login main.go to use shared registry (M)
- Replace `getWriterProducer()` with `opcodes.BuildWriterProducer()`
- Replace `handlerProducer()` with `opcodes.BuildHandlerProducer()`
- Keep `produceWriters()` and `produceHandlers()` (service-specific registrations)
- Build and test

#### 5.7 Refactor atlas-channel main.go to use shared registry (M)
- Same changes as atlas-login
- Build and test

#### 5.8 Verify Docker builds for both services (M)
- Run Docker builds to catch any dependency issues

### Dependencies: None (parallel with Phases 1-4)
### Risk: Medium. Touches service bootstrap in both services. The generic handler producer may need careful type parameterization.

---

## Phase 6: Implement Decode for Attack and EffectSkillUse

### Goal
Implement real Decode methods for the 3 remaining no-op server-send packets: Attack, EffectSkillUse, EffectSkillUseForeign.

### Current State
- Attack.Encode has complex conditional logic: skill presence, region/version gates, attack action cutoff, meso explosion flag, attack type, keydown flag
- Attack.Decode is no-op because the wire format is NOT self-describing - conditional fields depend on constructor flags (isMesoExplosion, hasKeydown, attackType, isStrafe)
- EffectSkillUse.Encode has conditional booleans (isBerserk, isDragonFury, isMonsterMagnet) that are NOT on the wire
- AttackInfo model already has full Decode (client-send direction), but Attack (server-send) is different wire format

### Design Decision
Since the wire format is not self-describing, Decode must receive the same constructor flags that Encode uses. The caller must construct the struct with the correct flags before calling Decode. This is consistent with how a test client would use it: create `Attack{attackType: "melee", isMesoExplosion: false, hasKeydown: true}` then call Decode to fill in the data fields.

### Tasks

#### 6.1 Implement Attack.Decode (L)
- Mirror the Encode logic branch-by-branch
- Read characterId, packed damage/hits byte, level
- Conditional skill read (check if next bytes represent skillId > 0)
- Region/version gate for strafe passive SLV
- Read option byte, packed action/left int16
- Conditional damage block (attackAction <= 0x110): actionSpeed, mastery, bulletItemId, damage info array
- Ranged coordinates conditional
- Keydown conditional
- Constructor flags (isMesoExplosion, hasKeydown, attackType, isStrafe) must be set before Decode

#### 6.2 Implement EffectSkillUse.Decode (M)
- Read: mode, skillId, characterLevel, skillLevel
- Conditional booleans require constructor flags (isBerserk, isDragonFury, isMonsterMagnet)
- If isBerserk: read berserkDarkForce bool
- If isDragonFury: read dragonFuryCreate bool
- If isMonsterMagnet: read monsterMagnetLeft bool

#### 6.3 Implement EffectSkillUseForeign.Decode (M)
- Read characterId prefix, then delegate to EffectSkillUse.Decode pattern

#### 6.4 Add round-trip tests for Attack (L)
- Test all 4 attack types: Melee, Ranged, Magic, Energy
- Test with/without meso explosion
- Test with/without keydown
- Test with/without skill
- Test across all region/version variants (GMS v28, v83, v95, JMS v185)
- Use existing DamageInfo model for test data construction

#### 6.5 Add round-trip tests for EffectSkillUse (M)
- Test with each boolean flag combination
- Test EffectSkillUseForeign variant
- Test across all region variants

#### 6.6 Build and test atlas-packet (S)
- `go test ./...` in libs/atlas-packet

### Dependencies: None (parallel with other phases)
### Risk: Medium-high. Attack is the most complex packet. The non-self-describing wire format means Decode correctness depends on correct constructor flags. Exhaustive test coverage is essential.

---

## Execution Order

```
Independent - can run in parallel:
  Phase 1 (world_message default)     ← smallest, do first
  Phase 2 (dead model cleanup)        ← trivial deletions
  Phase 5 (opcode registry)           ← largest, start early
  Phase 6 (Attack/Effect Decode)      ← complex, start early

Sequential:
  Phase 1 must complete before Phase 4 (same file)

After Phase 1:
  Phase 4 (operator hack cleanup)

After Phase 2:
  Phase 3 (npc_conversation adapter) ← establishes pattern from Phase 2
```

## Success Metrics

1. Zero `response.NewWriter` imports in any service writer file
2. Zero duplicate model encoder files in service socket/model directories
3. npc_conversation.go under 100 lines
4. No `extractUint32FromOperator` or `extractInt32FromOperator` in codebase
5. `libs/atlas-opcodes` usable by test client without service dependencies
6. All Decode methods functional (no no-ops except genuinely empty packets)
7. All builds pass: atlas-login, atlas-channel, atlas-packet, atlas-opcodes
8. All tests pass with round-trip coverage across region/version variants

## Effort Estimates

| Phase | Effort | Files Changed | Risk |
|-------|--------|---------------|------|
| 1: world_message default | S | 2-3 | Low |
| 2: dead model cleanup | S | 7-8 | Low |
| 3: npc_conversation | M | 2-3 | Medium |
| 4: operator hack | M | 3-4 | Low-Medium |
| 5: opcode registry | L | 8-10 (new lib + 2 services) | Medium |
| 6: Attack/Effect Decode | L | 3-4 + tests | Medium-High |
