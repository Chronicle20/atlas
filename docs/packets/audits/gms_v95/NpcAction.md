# NpcAction (← `CNpc::OnMove`)

- **IDA:** 0x678060
- **Atlas file:** `libs/atlas-packet/npc/clientbound/action.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `npcId (consumed by CNpcPool::OnNpcPacket@0x679260 dispatcher before OnMove; atlas Action.objectId)` | ✅ |  |
| 1 | byte | byte `action / m_nOneTimeAction (atlas unk)` | ✅ |  |
| 2 | byte | byte `chatIdx / nChatIdx (atlas unk2)` | ✅ |  |
| 3 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 6 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

## Conditional movement body (tool limitation)

Rows 3–6 are the optional `model.Movement` body (`StartX int16`, `StartY int16`,
element count `byte`, element type `byte`, ...). The flat analyzer cannot model
the conditional — it appends the movement fields unconditionally and flags them
as "extra".

Verified against IDA `CNpc::OnMove@0x678060` (lines 271–280): the movement body
is read via `CMovePath::OnMovePacket(...)` **only when `this->m_pTemplate->bMove`
is set** — a client-side NPC-template flag, not a packet field. Server-side,
atlas gates the movement encode on `hasMovement` (set by `NewNpcActionMove` vs
`NewNpcActionAnimation`), so an animation-only action emits exactly `objectId(4)
+ action(1) + chatIdx(1)` with NO movement, and a move action appends the
`CMovePath`-compatible movement body. Both server variants align with the client's
template-gated read.

The leading `objectId` (row 0) is the dispatcher prefix consumed by
`CNpcPool::OnNpcPacket@0x679260` (Decode4) before `CNpc::OnMove` runs — the same
dispatcher-prefix pattern used by the other per-NPC packets.

**Verdict: ⚠️ (tool-limitation, manually verified — wire is correct).**

Ack: world-audit Phase 2e on 2026-05-28
