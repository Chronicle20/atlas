# NpcAskPetConversationDetail (← `CScriptMan::OnAskPet#AskPet`)

- **IDA:** 0x7b7fc2
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `message (@0x7b7fde)` | ✅ |  |
| 1 | byte | byte `pet count (@0x7b7ff8)` | ✅ |  |
| 2 | int64 | bytes `cashId (8 bytes) -- loop body (@0x7b8046)` | ❌ | width mismatch |
| 3 | byte | byte `unused -- loop body (@0x7b804e)` | ✅ |  |


## Manual verdict (JMS v185, `CScriptMan::OnAskPet` @0x7b7fc2)

Row 2 (❌ "int64 vs bytes") is a representation artifact, NOT a wire bug. Atlas writes the
8-byte cashId via `WriteLong(uint64)`; JMS185 reads it via `DecodeBuffer(8)`. These are the
SAME 8 bytes on the wire — the audit flags int64-vs-buffer as a width mismatch. The message,
count, per-entry cashId(8) and trailing unused byte all match (rows 0,1,3 ✅). NOTE: JMS185
maps dialog-type case 9 to AskPet (GMS v95 case 9 was AskMembershopAvatar, which is ABSENT
in JMS185). Carry-forward manual-verify.

Ack: world-audit Phase 3 JMS185 npc domain on 2026-05-28
