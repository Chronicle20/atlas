# bug: JMS 185.1 client crashes on entering a channel

Status: **root cause NOT established.** This file records what is confirmed and
what is ruled out. Do not dispatch a fix agent against it until the "Blocking"
section is answered.

## Reproduced

Yes — captured live in `atlas-main`, pod `atlas-channel-75c9d7b679-msgpq`.

- Tenant `abedf3b4-1d7c-4b3b-bc52-70f62ab09418`, region `JMS`, `ms.version 185.1`
- World 0, channel 0, character 40, map 10000, instance `00000000-…-0000`
- Deployed overlay: `de7a1a6` (main `903537fe8`)
- Session `827ab027-33cb-4896-a67a-e57c554298c3`

Timeline (from pod logs):

| time (UTC) | event |
|---|---|
| 23:13:11.330 | client connected |
| 23:13:11.343 | `CharacterLoggedInHandle` characterId 40 |
| 23:13:11.514 | `GET /api/rings?filter[characterId]=40` (new in `60d999251`) |
| 23:13:11.573 | `Writing SetField for character [40]` |
| 23:13:11.597 | `SpawnForSelf: character [40] in map [10000]`; map NPC/mob/etc. fan-out |
| 23:13:12.719 | client → server op `0xEA` (unhandled) |
| 23:13:12.729 | client → server op `0xDA` (unhandled) |
| 23:13:42.734 | session idle, server PING (no further client traffic) |
| 23:15:05.495 | connection ended |

The client transmits twice after the enter-field burst and then goes completely
silent — consistent with the crash landing ~1.1 s after `SetField`.

## Observed

Client access violation at `0x44F5E2`.

`0x44F5E2` is inside `sub_44F5DF` (JMS185 IDB `MapleStory_dump_SCY.exe.i64`,
session `a977912e`), a generic hash-map lookup:

```
44f5df  push esi
44f5e0  mov  esi, ecx
44f5e2  cmp  dword ptr [esi+4], 0      <-- faults
...
44f5f9  div  dword ptr [esi+8]
```

The faulting instruction is the **first dereference of the map object**, so the
map/pool pointer itself is invalid — this is a null/garbage `this`, not a
mis-decoded packet body.

`xrefs_to 0x44f5df` (13, complete) — every caller is a pool lookup:

- `CMobPool::GetMob` @0x448819, `CMobPool::OnMobLeaveField` @0x6f8a1f, `sub_6F76D6`
- `CEmployeePool::OnEmployee{EnterField,LeaveField,MiniRoomBalloon}` @0x542a71/0x542b0e/0x542b6c, `sub_5428F7`
- `CNpcPool::{SetLocalNpc,SetRemoteNpc,GetNpc,OnNpcEnterField,OnNpcLeaveField}`
  @0x720242 / 0x7202ea / 0x720623 / **0x72068f** / 0x720724

`CNpcPool::OnNpcEnterField` @0x72068f calls it as `sub_44F5DF(this + 1, …)`, so
the user's read that `OnNpcEnterField` is a plausible ancestor is consistent.

## Expected

Client enters map 10000, spawns NPCs, and stays connected.

## Ruled out (with evidence)

1. **NpcSpawn body shape is correct for JMS 185.**
   `CNpc::Init` @0x716da2 reads, in order:
   `Decode2` @0x716dd6 (x), `Decode2` @0x716de4 (cy), `Decode1` @0x716e0c (f),
   `Decode2` @0x716e1c (fh), `Decode2` @0x716e3c (rx0), `Decode2` @0x716e4a (rx1),
   `Decode1` @0x716edf (bEnabled) — 7 reads after the two `Decode4`s in
   `OnNpcEnterField`. That is exactly what
   `libs/atlas-packet/npc/clientbound/spawn.go` writes for a JMS tenant
   (`hasEnabledFlag` returns `true` for non-GMS). No field mismatch.
   Note this also corrects `docs/packets/audits/jms_v185/NpcSpawn.json`, which
   records the body as an "opaque npc position/appearance block".

2. **The crash signature is not a body-decode error.** A wrong NpcSpawn body
   would misalign or throw inside `CInPacket`, not fault on the pool pointer's
   first dereference.

3. **Rings (`60d999251`, the most recent shared-codec change) did not alter the
   bytes for this character.** `GET /api/rings?filter[characterId]=40` returns
   `"data":[], "total":0`, and `model.RingRecords.EncodeRecords` with all three
   slices empty emits the identical `WriteShort(0)` ×3 that the pre-task
   `CharacterData.encodeRings` stub emitted
   (`libs/atlas-packet/model/ring.go:255-270`). `RingSet.EncodeField` with all
   arms nil likewise emits three zero bytes on JMS as on GMS. CharacterData /
   SetField is byte-identical to pre-rings for character 40.

4. **`845a5c992` (gms_v95 verification batch) touched nothing JMS-facing.**
   Its only non-test source change is `libs/atlas-packet/test/movement_types.go`;
   `template_jms_185_1.json` is untouched.

5. **task-251 (player NPCs) is not merged** — branch only; the NPC writer path
   on `main` has not changed recently.

## Leading hypothesis (unconfirmed)

The client is dispatching a field-scoped pool packet while `CField` / the pool
is not constructed, i.e. the failure is **upstream of the NPC packet**, in
whatever the client did (or failed to do) with the enter-field burst. Two
sub-hypotheses, neither yet tested:

- (H1) A packet in the enter-field burst is malformed for JMS 185, the client
  swallows the decode exception, leaves the field half-built, and the next
  pool-scoped packet faults on the null pool.
- (H2) An opcode in `template_jms_185_1.json` routes a packet the client maps
  into the pool range before the field exists.

## Blocking — needs the user

1. **When did this last work?** Which deployed overlay / date was the last good
   JMS 185.1 channel enter? The candidate range on `main` is only three deploys
   wide (`845a5c9` → `60d9992` → `de7a1a6`); a wider "last known good" opens it
   up considerably and changes which diff to bisect.
2. **A wire capture of the enter-field burst** (or the client-side crash
   dialog's full text / call stack). Without the bytes, H1 vs H2 cannot be
   separated, and neither the writer list nor the opcode table can be checked
   against what the client actually received.

## Fix

Not yet determinable. File inventory will be filled in once the root cause is
established.

## Not yet answered

- Which of the ~30 writers in the enter-field burst is the one the client chokes
  on (if H1).
- Whether ops `0xEA` / `0xDA` (client → server, currently unhandled on JMS 185)
  are a normal post-load pair or a symptom.
- Whether the crash is reachable on other client versions or is JMS-185-only.
