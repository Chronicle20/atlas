# NpcAskPetAllConversationDetail (← `CScriptMan::OnAskPetAll#AskPetAll`)

- **IDA:** 0x7b8250
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `message (@0x7b826a)` | ✅ |  |
| 1 | byte | byte `pet count (@0x7b8281)` | ✅ |  |
| 2 | byte | byte `exceptionExists (@0x7b8290)` | ✅ |  |
| 3 | int64 | bytes `cashId (8 bytes) -- loop body (@0x7b82d7)` | ❌ | width mismatch |
| 4 | byte | byte `unused -- loop body (@0x7b82df)` | ✅ |  |


## Manual verdict (JMS v185, `CScriptMan::OnAskPetAll` @0x7b8250)

The ❌ row is a representation artifact (int64 `WriteLong` vs `DecodeBuffer(8)`), NOT a wire
bug — the same 8 cashId bytes. Atlas writes `AsciiString(message)+Byte(count)+Bool(exceptionExists)
+loop[Long(cashId)+Byte(0)]`, matching JMS185 `DecodeStr + Decode1(count) + Decode1(exceptionExists)
+ loop[DecodeBuffer(8)+Decode1]`. Carry-forward manual-verify.

Ack: world-audit Phase 3 JMS185 npc domain on 2026-05-28
