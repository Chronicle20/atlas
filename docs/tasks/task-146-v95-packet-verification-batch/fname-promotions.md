# task-146 W0b — fname promotion evidence

Database: `ecc757f4` (GMS_v95.0_U_DEVM.exe.i64), read live via the IDA MCP
server (`http://192.168.20.3:8745/mcp`) using `decompile`, `xrefs_to`, and
`lookup_funcs`. Ground truth for each serverbound promotion is the literal
opcode argument in `COutPacket::COutPacket(&pkt, N)`; ground truth for the
clientbound promotion is the dispatch arm reached from the packet's
`OnPacket` router.

## `CHANGE_MAP` sb → `CField::SendTransferFieldRequest`

- Address: `0x5345c0`
- Build site: `COutPacket::COutPacket(&oPacket, 41); /*0x534615*/`
- Opcode read: **41** (`0x29`)
- Registry opcode: 41
- Verdict: **match — admissible**

## `NPC_TALK` sb → `CUserLocal::TalkToNpc`

- Address: `0x9321f0`
- Build site: `COutPacket::COutPacket(&oPacket, 63); /*0x93224d*/`
  (in the branch taken when `pNpc->m_aQuest` is empty; the quest-list branch
  calls `CNpc::ShowQuestList` instead of sending a packet, confirming
  `ShowQuestList` is not a sender for this op)
- Opcode read: **63** (`0x3F`)
- Registry opcode: 63
- Verdict: **match — admissible**

## `CHANGE_MAP_SPECIAL` sb → `CUserLocal::CheckPortal_Collision`

- Address: `0x919a10`
- Build site: `COutPacket::COutPacket(&oPacket, 112); /*0x919b07*/`
  (portal-type `case 9` branch of the `nType` switch)
- Opcode read: **112** (`0x70`)
- Registry opcode: 112
- Verdict: **match — admissible**

## `NPC_TALK` cb → `CScriptMan::OnScriptMessage`

- Address: `0x6de0f0`
- Dispatch-arm address: `0x6de368` / `0x6de36f`, in
  `CScriptMan::OnPacket@0x6de360`:
  ```
  void __thiscall CScriptMan::OnPacket(CScriptMan *this, int nType, CInPacket *iPacket)
  {
    if ( nType == 363 ) /*0x6de368*/
      CScriptMan::OnScriptMessage(this, iPacket); /*0x6de36f*/
  }
  ```
  Confirmed via `xrefs_to` on `0x6de0f0`: its only caller is
  `CScriptMan::OnPacket@0x6de360`.
- Opcode read: **363** (`0x16B`), the sole `nType` compared in `OnPacket`
- Registry opcode: 363
- Verdict: **match — admissible**

All four promotions confirmed live and admissible; none is a possible
second codec (design §5.2 does not apply).
