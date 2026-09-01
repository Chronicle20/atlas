# Bug: Hall of Fame Player NPC appears twice, and clicking one logs the player out

Task: task-251-player-npcs · PR env: `atlas-pr-1475` · Branch: `task-251-player-npcs`

Tenant `eb4b264b-bfce-44b4-a34a-058be08ab655` — **GMS 83.1** (confirmed via
`GET /api/tenants` on `atlas-tenants` in `atlas-pr-1475`).

This file covers three separate defects reported after live testing. (A fourth
item, "merge main into the branch," is not a bug and was handled directly:
merge commit `c597a940c`.)

---

## 1. `@playernpc` is missing from `@help`

### Reproduced
Not re-run live; the defect is static and visible in source.

### Observed
`services/atlas-messages/atlas.com/messages/command/help/commands.go:15`'s
`commandSyntaxList` has 24 entries and none of them mention `@playernpc`,
although `command/playernpc/commands.go:30-31` registers two GM commands:

```
^@playernpc\s+add\s+(\S+)$
^@playernpc\s+remove\s+(\S+)(?:\s+(here))?$
```

### Expected
`@help` lists every registered GM command, `@playernpc add` / `@playernpc
remove` included.

### Root cause
Task-251 added the command package but never touched the hard-coded
`commandSyntaxList`. There is no test or guard that ties the two together, so
nothing caught it.

---

## 2. The deployed Player NPC is rendered twice

### Reproduced
Live, in `atlas-pr-1475`, on map **102000004** (Hall of Warriors).

### Observed
One deployed Player NPC exists server-side. `GET
/api/player-npcs?filter[mapId]=102000004` on `atlas-player-npcs` returns
exactly one row:

```
characterId 2, name "Chronicle", mapId 102000004, scriptId 9901000,
objectId 101000, x 2, cy 35, fh 5, rx0 52, rx1 -48, dir 1
```

But `GET /api/data/maps/102000004/npcs` on `atlas-data` returns **9** static
WZ NPC life entries, and they are precisely the Player NPC placeholder
templates:

```
oid 1 template 9901000  x -1  cy 35  fh 5  rx0 -51  rx1 49
oid 2 template 9901001  ...
...
oid 9 template 9901008  ...
```

So on map entry the channel sends two SPAWN_NPC packets carrying template
`9901000`: one for WZ oid `1` (from `spawnNPCForSession`) and one for oid
`101000` (from `spawnPlayerNpcsForSession`), a few pixels apart. Both then
receive the imitated look, because the client's overlay is keyed on
**template id**, not oid.

### Client evidence (GMS v83, `MapleStory_dump.exe.i64`, session `754107bf`)
`CNpcPool::OnNpcImitateData` @ `0x6d97c6` decompiles to a pure *overlay*, not
a spawn:

```c
v2 = CInPacket::Decode1(a2);            // count
do {
  v3 = CInPacket::Decode4(a2);          // templateId
  CInPacket::DecodeStr(a2, v9);         // name
  AvatarLook::Decode(v7, a2);
  v4 = *(v8 + 10);                      // walk the existing CNpcPool list
  while (v4) {
    v5 = *(v4 + 16); v4 = *(v4 + 4);
    if (v5 && **(v5 + 42) == v3) {      // NPC's template == entry template
      NpcTemplate = CNpcTemplate::GetNpcTemplate(v3);
      ZXString<char>::operator=((NpcTemplate + 4), v9);   // rename template
      CNpc::SetImitatedLook(v5, v7);
    }
  }
} while (--v10);
```

It creates nothing, and it applies to **every** pooled NPC whose template
matches — which is why both objects wear Chronicle's appearance, and why the
WZ placeholder is the intended carrier of the look.

### Expected
Exactly one figure per deployed Player NPC; the unoccupied WZ placeholder
slots (`9901001`–`9901008` here) render as nothing.

### Root cause
Design D-4/FR-7.1 assumed the Hall of Fame maps had no NPC objects to overlay
and therefore had `spawnPlayerNpcsForSession` emit its own SPAWN_NPC. The WZ
map data *does* carry one placeholder life entry per Player NPC slot, and
`spawnNPCForSession` already spawns all nine. Nothing suppresses them, so
every deployed Player NPC is doubled and eight empty placeholders are spawned
and controller-claimed besides.

Keeping the channel's own SPAWN_NPC (rather than relying on the WZ
placeholder) is the right call: PRD FR-3.3 supports GM deployment onto an
arbitrary non-Hall map, where no placeholder exists, and the placeholder count
(9) is far below the script-id band capacity (100 per job). The placeholders
are what must go.

---

## 3. Double-clicking the Player NPC logs the character out

### Reproduced
Live, same map. Double-clicking the object at oid `101000` disconnects the
client to the login screen. Double-clicking the WZ placeholder (oid `1`) does
nothing, as expected — it has no conversation script.

`kubectl logs` against `atlas-channel-d6688db4f-2g58k` in `atlas-pr-1475` timed
out repeatedly (three attempts, `--tail=20`/`500`/`3000`, all exit 124), so the
log line is not quoted here. The code path is unambiguous without it.

### Observed / root cause
`services/atlas-channel/atlas.com/channel/socket/handler/npc_start_conversation.go:25-30`:

```go
n, err := npcData.NewProcessor(l, ctx).GetInMapByObjectId(s.MapId(), oid)
if err != nil {
    l.WithError(err).Errorf("Character [%d] is interacting with a map object [%d] that is not found in map [%d].", ...)
    _ = session.NewProcessor(l, ctx).Destroy(s)
    return
}
```

Player NPC oids live in the reserved band `[100000, 999999]`
(`libs/atlas-object-id/reserved.go:13`, design D-5) and are **never** present
in `atlas-data`'s per-map NPC list, whose oids are the WZ life indices `1..n`.
So `GetInMapByObjectId(102000004, 101000)` always fails, and the handler's
anti-cheat branch destroys the session.

`socket/handler/npc_item_use.go` performs the same probe and needs the same
treatment.

### Expected
Player NPCs are non-interactive (PRD FR-7.4 / design §7.3). A click on one is a
no-op, never a disconnect.

---

## 4. `rx0`/`rx1` are derived swapped (found while diagnosing #2)

`services/atlas-player-npcs/atlas.com/player-npcs/playernpc/builder.go:124-130`:

```go
// SetX sets x and derives rx0 = x + 50, rx1 = x - 50 (design §3.1).
b.rx0 = x + 50
b.rx1 = x - 50
```

Every WZ NPC life entry on 102000004 uses the opposite convention — `rx0 =
x - 50`, `rx1 = x + 50`, i.e. `rx0 < rx1` (see the `atlas-data` dump in §2:
`x -1, rx0 -51, rx1 49`). The deployed row shows the inverted range
(`x 2, rx0 52, rx1 -48`). Design §3.1 has it backwards.

Low severity (it bounds the NPC's idle wander range, not its placement), but
it is a one-line correction with direct WZ evidence, so it is in scope here.

---

## Fix

### `@help` (defect 1)
- `services/atlas-messages/atlas.com/messages/command/help/commands.go` — add
  two `commandSyntaxList` entries, matching the existing phrasing style:
  - `"@playernpc add <target> - Deploy a Player NPC for a character at your position"`
  - `"@playernpc remove <target> [here] - Remove a character's Player NPCs (all, or just this map)"`

### Duplicate render (defect 2)
- `libs/atlas-object-id/reserved.go` — add an exported predicate for the
  Player NPC *script id* pool (design §4.2: `9901000`–`9906599`), next to
  `PlayerNpcObjectIdFor`. Name it to match the file's existing vocabulary.
- `services/atlas-channel/atlas.com/channel/data/npc/processor.go` — filter
  the imitate-band templates out of `InMapModelProvider` (the
  `requests.DrainProvider` already takes `model.Filters[Model]()`; add the
  predicate there). This is the single choke point: `ForEachInMap` builds on
  it, so both `spawnNPCForSession`
  (`kafka/consumer/map/consumer.go:240`, `:677`) and the controller sweep
  (`npc/controller/processor.go`) stop seeing the placeholders, and no
  controller is elected for them either.
  - Note `GetInMapByObjectId` goes through `InMapByObjectIdModelProvider`, a
    *different* provider; leave it alone — defect 3's fix covers that path.
- `services/atlas-channel/atlas.com/channel/data/npc/processor_test.go` (new,
  or extend the existing `rest_test.go` package) — assert a map whose NPC list
  mixes imitate-band and ordinary templates yields only the ordinary ones.

### Click logout (defect 3)
- `services/atlas-channel/atlas.com/channel/socket/handler/npc_start_conversation.go`
  — before the `GetInMapByObjectId` call, return early (debug log, no
  `Destroy`) when `oid >= objectid.PlayerNpcObjectIdBase && oid <
  objectid.MinId`. Cite PRD FR-7.4 in the comment.
- `services/atlas-channel/atlas.com/channel/socket/handler/npc_item_use.go` —
  same guard on its probe (see the comment at `:30` / `:80` naming the shared
  probe).
- Tests alongside both handlers asserting a Player NPC oid does **not** destroy
  the session and does **not** start a conversation.

### `rx0`/`rx1` (defect 4)
- `services/atlas-player-npcs/atlas.com/player-npcs/playernpc/builder.go:124-130`
  — swap to `rx0 = x - 50`, `rx1 = x + 50`; update the doc comment to cite the
  WZ convention rather than design §3.1.
- `services/atlas-player-npcs/atlas.com/player-npcs/playernpc/model_test.go:188-192`
  — the existing expectations (`RX0() == 150`, `RX1() == 50` for `x = 100`)
  invert accordingly.
- `docs/tasks/task-251-player-npcs/design.md` §3.1 — correct the stated
  derivation so the doc and the code agree.

## Not yet answered

- Whether any map outside the Hall of Fame set carries a `9901000`–`9906599`
  life entry that a player is *meant* to interact with. Nothing in the PRD or
  design suggests one, and the band is defined as the imitate pool, but the
  filter in defect 2 is repo-wide by template, so a counter-example would
  matter. If the implementer finds one, stop and surface it.
- Whether the eight now-suppressed empty placeholders were the only thing
  making the Hall look populated on a fresh tenant. Cosmetic; no action.
- The `atlas-channel` pod logs could not be read (`kubectl logs` exit 124 on
  every attempt). Defect 3's diagnosis rests on source, not on the log line.

## Outcome

_(to be filled in: fix commit, gate verdict, live re-test result)_
