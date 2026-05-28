# NpcAction (← `CNpc::OnMove`)

- **IDA:** 0x6d2e07
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/action.go`
- **Variant:** GMS/v83
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `npcId (consumed by CNpcPool::OnNpcPacket@0x6d98de dispatcher before OnMove; atlas Action.objectId)` | ✅ |  |
| 1 | byte | byte `action / v3 (atlas unk)` | ✅ |  |
| 2 | byte | byte `chatIdx / v4 (atlas unk2)` | ✅ |  |
| 3 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 6 | byte | byte `` | ❌ | atlas: extra — client never reads this field |


## Conditional movement body (tool limitation)

Rows 3–6 are the optional `model.Movement` body. The flat analyzer cannot model
the conditional — it appends the movement fields unconditionally and flags them
as "extra".

Verified against IDA `CNpc::OnMove@0x6d2e07`: the movement body is read via
`CMovePath::OnMovePacket(...)` **only when the NPC template's `bMove` flag is
set** — a client-side template flag, not a packet field. Server-side, atlas gates
the movement encode on `hasMovement` (set by `NewNpcActionMove` vs
`NewNpcActionAnimation`), so an animation-only action emits exactly `objectId(4)
+ action(1) + chatIdx(1)` with NO movement. Both server variants align with the
client's template-gated read. The leading `objectId` (row 0) is the dispatcher
prefix consumed by `CNpcPool::OnNpcPacket@0x6d98de` (Decode4) before
`CNpc::OnMove` runs. Identical shape to v95.

**Verdict: ⚠️ (tool-limitation, manually verified — wire is correct for v83).**

Ack: world-audit Phase 3 v83 (12b npc) on 2026-05-28
