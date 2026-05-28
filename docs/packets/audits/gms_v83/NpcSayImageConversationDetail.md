# NpcSayImageConversationDetail (← `CScriptMan::OnSayImage#SayImage`)

- **IDA:** 0x961275
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `image count (v9)` | ❌ | width mismatch |
| 1 | string | string `image path -- loop body (count iterations; analyzer flattens)` | ✅ |  |


## Loop body + dialog-type availability (tool limitation)

Atlas `SayImageConversationDetail.Encode` emits `WriteByte(len)` + a **loop** of
`WriteAsciiString(image)`. The flat analyzer cannot model the loop, so beyond the
modeled single-image iteration the comparison misaligns — a tool limitation.

Verified against IDA `CScriptMan::OnSayImage@0x961275`: `Decode1(count @0x961302)`
+ `loop count × DecodeStr(image @0x961318)`. The count is a **single byte**
(`Decode1`), confirming the Phase-2 unconditional `WriteByte` fix is correct for
v83 (atlas previously wrote `WriteInt`).

NOTE on dialog availability: v83's `CScriptMan::OnScriptMessage` switch has **no
SayImage case** — its dialog-type enum is shifted (msgType 1 = AskYesNo in v83 vs
SayImage in v95). The `CScriptMan::OnSayImage` handler exists in the v83 binary
(invoked via `SetUtilDlgEx_IMAGE`) but is not reachable through the standard
script-message `msgType` dispatch; in v83 a Say-with-images is a Say(0) variant.
The atlas `SayImageConversationDetail` encoder's BYTE-COUNT wire shape is
nonetheless correct for the v83 client handler, and the msgType→byte mapping is
tenant-config-driven (a v83 template simply would not route to SAY_IMAGE).

**Verdict: ⚠️ (tool-limitation, manually verified — count byte + per-image string correct for v83).**

Ack: world-audit Phase 3 v83 (12b npc) on 2026-05-28
