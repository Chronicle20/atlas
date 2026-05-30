# NpcAskPetConversationDetail (← `CScriptMan::OnAskPet#AskPet`)

- **IDA:** 0x7474a2
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `message text (v26)` | ✅ |  |
| 1 | byte | byte `pet count (v4)` | ✅ |  |
| 2 | int64 | bytes `cashId (8-byte buffer) -- loop body` | ❌ | width mismatch |
| 3 | byte | byte `unused byte per pet -- loop body` | ✅ |  |


## Loop body (tool limitation)

Rows 0–1 (message string + pet-count byte) match the v83 client exactly. Row 2
flags an int64-vs-bytes "width mismatch": atlas `AskPetConversationDetail.Encode`
emits a **loop** of `WriteLong(cashId) + WriteByte(0)` per pet, which the flat
analyzer cannot model (it sees the 8-byte long against the IDA `DecodeBuffer(8)`
which it renders as a `bytes` op of the same width).

Verified against IDA `CScriptMan::OnAskPet@0x7474a2`: `DecodeStr(message)` +
`Decode1(count @0x7474d5)` + `loop count × {DecodeBuffer(8) cashId @0x74751f +
Decode1 @0x747527}`. Atlas writes `AsciiString + Byte(len) + loop{Long(8) +
Byte(0)}` — the 8-byte cashId (atlas `WriteLong`) maps to the client's
`DecodeBuffer(8)`, and the per-pet trailing byte matches. Wire is identical.

**Verdict: ⚠️ (tool-limitation, manually verified — wire is correct for v83).**

Ack: world-audit Phase 3 v83 (12b npc) on 2026-05-28
