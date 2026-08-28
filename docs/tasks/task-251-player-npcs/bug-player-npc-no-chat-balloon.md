# Bug: deployed Player NPCs never speak (task-251 bug report #5)

## Symptom

A deployed Player NPC renders with the correct look, but never shows the canned
chat balloon ("I am <name> who had achieved level 200."). The balloon *was*
observed before commit `ba201d4d4` ("drop Player NPC imitate placeholders from
map NPC lists"), on the duplicate object that commit removed.

## Where the balloon comes from

It is entirely client-side. `Npc.wz/9901000.img/info` carries:

```
imitate = 1, hideName = 1
speak = { 0: "n0", 1: "n1" }
script/0/script = "rank_user"
```

and `String.wz/Npc.img/9901000/n0` is `"I am /name who had achieved level 200."`.
The client substitutes `/name` from `CNpcTemplate::sName`, which
`CNpcPool::OnNpcImitateData` overwrites with the decoded name immediately
before calling `CNpc::SetImitatedLook`. The server sends no balloon text.

## Root cause

The balloon is emitted by `CNpc::DoActionOrChat` (GMS v95 `0x6702b0`), called
from `CNpc::Update` (`0x677b50`) behind:

```c
if (this->m_bEnabled) {
  ...
  v10 = this->m_pvcActive.m_pInterface - 3;
  if (v10[6].lpVtbl) {                 // requires m_pvcActive to be set up
    if (!this->m_bMovePathSent) { ... CNpc::DoActionOrChat(...); }
```

`m_pvcActive` is populated only by `CNpc::SetActive(npc, 1)` (`0x6710b0`),
which also seeds `m_tWaitTimeForNextActionOrChat`. Its call graph is closed:

- `CNpcPool::SetLocalNpc` (`0x679440`) -> `SetActive(.., 1)` — the only enable path
- `CNpcPool::SetRemoteNpc` (`0x6791a0`) -> `SetActive(.., 0)`
- both reached only from `CNpcPool::OnNpcChangeController` (`0x679730`), i.e.
  the `SPAWN_NPC_REQUEST_CONTROLLER` packet

Plain `SPAWN_NPC` (`CNpcPool::OnNpcEnterField`) materializes and renders the
object but never calls `SetActive`, so it never chats.

Design section 7.2 (D-4) deliberately chose plain `SPAWN_NPC` with no controller
grant, on the reasoning that "granting control of an object that never moves
would be meaningless at best". That reasoning is wrong: the grant is precisely
what enables the chat balloon. The PRD's original "reuse
`NpcSpawnRequestControllerWriter`" was correct.

Before `ba201d4d4`, the WZ placeholder life entry went through the ordinary NPC
path (`spawnNPCForSession`), which *does* controller-elect. That object was
active and chatting, with the imitate overlay painting the player look and name
onto it; our own no-controller spawn sat beside it as the silent duplicate. The
commit removed the working object and kept the mute one.

Two corollaries:

- The unoccupied placeholder siblings that commit also suppressed are harmless:
  `stand/0` is a 1-pixel-wide canvas with `hideName = 1`, i.e. invisible until
  imitate data paints them.
- Design section 7.2's stated fallback, "grant-then-immediately-revoke", would
  not work either — the revoke is `SetRemoteNpc` -> `SetActive(.., 0)`, which
  switches the balloon straight back off.

## Fix

Keep `ba201d4d4`'s placeholder filter (our spawn reuses the same template id, so
removing the filter reintroduces the duplicate) and add the controller grant for
Player NPC object ids, reusing the existing per-map single-controller election so
exactly one client controls each NPC — the same machinery ordinary NPCs use.

FR-7.4 ("Player NPC object ids must never participate in controller assignment")
is amended: Player NPCs now enter the controller registry like any other NPC.
The requirement design section 7.3 was actually protecting — that a Player NPC is
never left dangling in the registry after its owner leaves — is satisfied by the
existing `ReleaseFor`/`ElectFor` exit path, which is object-id agnostic.

## Evidence

IDB: `GMS_v95.0_U_DEVM.exe` (session `ecc757f4`). WZ: `Npc.wz/9901000.img`,
`String.wz/Npc.img/9901000`, `Map.wz/Map/Map1/102000004.img` life entries
`9901000`-`9901008`.
