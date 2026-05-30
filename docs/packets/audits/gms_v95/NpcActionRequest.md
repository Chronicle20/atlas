# NpcActionRequest (← `CNpc::GenerateMovePath`)

- **IDA:** 0x671590
- **Atlas file:** `libs/atlas-packet/npc/serverbound/action.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `npcId (m_dwNpcId; atlas ActionRequest.objectId)` | ✅ |  |
| 1 | byte | byte `nAction (atlas unk)` | ✅ |  |
| 2 | byte | byte `nChatIdx (atlas unk2)` | ✅ |  |
| 3 | int16 | bytes `movement body (CMovePath::Flush) -- gated on m_pTemplate->bMove; atlas optional WriteByteArray(movement)` | ❌ | width mismatch |
| 4 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 6 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

## Conditional movement body (tool limitation)

Rows 0–2 match the v95 client exactly (`objectId int32`, `unk/action byte`,
`unk2/chatIdx byte`). Rows 3–6 are the optional `model.Movement` body (`StartX
int16`, `StartY int16`, element count `byte`, element type `byte`, ...). The flat
analyzer cannot model the conditional — it appends the movement fields
unconditionally and flags them as "extra"/width-mismatch against the single
`EncodeBuffer` movement-body entry.

Verified against IDA `CNpc::GenerateMovePath@0x671590`: the client builds
`COutPacket(241)` then `Encode4(m_dwNpcId)` (0x6716af) + `Encode1(nAction)`
(0x6716f5) + `Encode1(nChatIdx)` (0x671743), and appends the movement body via
`CMovePath::Flush` (0x671765) **only when `this->m_pTemplate->bMove` is set**
(0x671750) — a client-side NPC-template flag, not a packet field. Server-side,
atlas gates the movement encode on `hasMovement` (set when a move action is
issued), so an animation-only action emits exactly `objectId(4) + action(1) +
chatIdx(1)` with NO movement, and a move action appends the `CMovePath`-compatible
movement body. Both server variants align with the client's template-gated read.

Unlike the clientbound `NpcAction` (which has a `CNpcPool::OnNpcPacket` dispatcher
that consumes `npcId` before the per-handler body), this SERVERBOUND request is
built entirely by the client `Send*` site: `GenerateMovePath` writes the `npcId`
as the first field itself. There is NO dispatcher prefix on the serverbound side.

**Verdict: ⚠️ (tool-limitation, manually verified — wire is correct).**


Ack: world-audit Phase 2g on 2026-05-28
