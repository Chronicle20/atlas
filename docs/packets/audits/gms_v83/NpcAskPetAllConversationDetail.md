# NpcAskPetAllConversationDetail (← `CScriptMan::OnAskPetAll#AskPetAll`)

- **IDA:** 0x74775c
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `message text (v27)` | ✅ |  |
| 1 | byte | byte `pet count (v4)` | ✅ |  |
| 2 | byte | byte `exceptionExists flag (v24)` | ✅ |  |
| 3 | int64 | bytes `cashId (8-byte buffer) -- loop body` | ❌ | width mismatch |
| 4 | byte | byte `unused byte per pet -- loop body` | ✅ |  |


## Loop body (tool limitation)

Rows 0–2 (message string + pet-count byte + exceptionExists byte) match the v83
client exactly. Row 3 flags an int64-vs-bytes "width mismatch" for the same
reason as NpcAskPetConversationDetail: atlas emits a **loop** of
`WriteLong(cashId) + WriteByte(0)` the flat analyzer cannot model.

Verified against IDA `CScriptMan::OnAskPetAll@0x74775c`: `DecodeStr(message)` +
`Decode1(count @0x74778d)` + `Decode1(exceptionExists @0x74779c)` + `loop count ×
{DecodeBuffer(8) cashId @0x7477e3 + Decode1 @0x7477eb}`. Atlas writes
`AsciiString + Byte(len) + Bool(ExceptionExists) + loop{Long(8) + Byte(0)}` —
field order (count before exceptionExists) and the 8-byte cashId both match.

**Verdict: ⚠️ (tool-limitation, manually verified — wire is correct for v83).**

Ack: world-audit Phase 3 v83 (12b npc) on 2026-05-28
