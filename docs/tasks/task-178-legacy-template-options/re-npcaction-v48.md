# task-178 addendum — v48 NPCActionHandle gap (content-verified)

## Context / correction
An initial pass concluded `NPCActionHandle` (and `ItcOperationHandle`) were
"correctly absent" in v48 because the v48 IDB had no *named* `SetLocalNpc` /
`GenerateMovePath` / `CITC::On*` symbols. **That reasoning was unsound** — the
GMS `*_DEVM` IDBs are incompletely/rotated-named, so a missing symbol name is not
evidence the code is absent. Re-verified by content (decompilation + opcode
disassembly), not by symbol names. Functions below were renamed in the v48 IDB
using the canonical mangled symbols from the PDB-backed v95 IDB.

## NPCActionHandle — GENUINE GAP in v48 (fixed here)
The v48 client DOES control NPCs and send NPC-action/move packets:
- `?OnNpcChangeController@CNpcPool@@IAEXAAVCInPacket@@@Z` (v48 `0x56d617`) →
  `?SetLocalNpc@CNpcPool@@IAEXKAAVCInPacket@@@Z` (`0x56d267`) /
  `?SetRemoteNpc@CNpcPool@@IAEXK@Z` (`0x56d30c`) — byte-identical to v61.
- `?GenerateMovePath@CNpc@@IAEXJJ@Z` (v48 `0x5688e9`) builds the send:
  ```
  push 8Ah                       ; opcode 0x8A
  call ??0COutPacket@@QAE@J@Z     ; COutPacket::COutPacket(long)
  push [edi+88h]                 ; npcId
  call ?Encode4@COutPacket@@…     ; Encode4(npcId)
  push arg_0                     ; action
  … Encode1(action); Encode1(a3);
  ?Flush@CMovePath@@… ; SendPacket
  ```
  Structurally identical to v61 `GenerateMovePath` (which sends at 0xA4 = v61's
  `NPCActionHandle` opcode). So the v48 serverbound NPC-action opcode is **0x8A**.
- v48 template routed nothing at 0x8A (CharacterMove 0x21, Pet 0x71, Monster 0x81).

**Fix:** added `NPCActionHandle` to the v48 template — `opCode 0x8A`,
`LoggedInValidator`, `types` copied from `CharacterMoveHandle` (23 entries) — and
PATCHed the live v48 tenant (200). (v48 is a parked tenant not currently bound to
a channel, so no runtime handler is instantiated yet; the config is now correct.)

## ItcOperationHandle — correctly absent in v48 (content-verified)
Unlike NPC, there is no ITC operation *send* to find. The exhaustive xref set of
the opcode-bearing `COutPacket` ctor (`0x57b77e`, 317 sites — the same set that
surfaced the NPC send at `0x5688e9`) contains **no** `CITC`/ITC operation cluster
and nothing in the ITC UI regions (`0x43c…`, `0x448…`). `CITCWnd_Inventory::OnCreate`
(`0x43c290`) is pure rendering (COM/`IWzGr2DLayer`/canvas/font/`Putoverlay`) — no
`COutPacket`/`SendPacket`. So v48 carries only vestigial ITC *display* UI
(`CITCWnd_Inventory`, `CRegisterAuctionEntryDlg`) with no functional operation send.
The Cash-Item Trading Center operations were not implemented in v48; `ItcOperationHandle`
is correctly not routed.

## Lesson
Never infer feature presence/absence from symbol NAMES in an incompletely-named
IDB. Verify by content: decompile the dispatch/send, read the opcode literal, and
use the exhaustive `COutPacket`-ctor xref set to enumerate what the client actually
sends.
