# bug: JMS 185.1 client crashes on entering a channel

Status: **crash mechanism proven; the SetField payload is exonerated; the
specific trigger is not yet identified.** CharacterData × jms_v185 was swept
field-by-field and is clean (see §7 under "Ruled out"). Suspicion has moved to
the enter-field burst (see `## Next unit`).

## Reproduced

Yes — captured live in `atlas-main`, pod `atlas-channel-75c9d7b679-msgpq`,
image `ghcr.io/chronicle20/atlas-channel/atlas-channel:main-051617c`.

- Tenant `abedf3b4-1d7c-4b3b-bc52-70f62ab09418`, region `JMS`, `ms.version 185.1`
- World 0, channel 0, character 40 ("Chronicle", level 1), map 10000, instance `00000000-…-0000`
- Session `827ab027-33cb-4896-a67a-e57c554298c3`
- Last known good: "days ago", exact build unknown (user).

Timeline (pod logs):

| time (UTC) | event |
|---|---|
| 23:13:11.330 | client connected |
| 23:13:11.343 | `CharacterLoggedInHandle` characterId 40 |
| 23:13:11.514 | `GET /api/rings?filter[characterId]=40` |
| 23:13:11.573 | `Writing SetField for character [40]` |
| 23:13:11.597 | `SpawnForSelf: character [40] in map [10000]`; enter-field fan-out |
| 23:13:12.719 | client → server op `0xEA` (unhandled) |
| 23:13:12.729 | client → server op `0xDA` (unhandled) |
| 23:13:42.734 | idle, server PING; no further client traffic |
| 23:15:05.495 | connection ended |

## Observed

Client access violation at `0x44F5E2`. Call stack (x32dbg, JMS185 IDB session
`a977912e`, `MapleStory_dump_SCY.exe.i64`):

```
CWvsApp::WindowProc            0xae23b3
CClientSocket::OnRead          0x4b125c
CClientSocket::ManipulatePacket 0x4b1717
CClientSocket::ProcessPacket   0x4b17eb
CField::OnPacket               0x56e721   (ret 0x56e985)
CNpcPool::OnPacket             0x720430   (ret 0x7204a4)
  -> CNpcPool::OnNpcEnterField 0x72068f   (call at 0x72049f)
     -> sub_44F5DF             0x44f5df   AV at 0x44f5e2
```

## Root cause mechanism (proven)

`CField::OnPacket` does **not** own the pools. It routes by opcode range to
*global singletons*:

```
0x56e957  mov ecx, dword_CD5698   ; CMobPool       (0x0FD..0x115)
0x56e979  mov ecx, dword_CD7590   ; CNpcPool       (0x116..0x11D)
0x56e99b  mov ecx, dword_CD749C   ; CEmployeePool  (0x11E..0x120)
```

`dword_CD7590` is written in exactly two places:

- `sub_71FED9` @0x71FEF2 — CNpcPool constructor, `dword_CD7590 = this`
- `sub_71FF65` @0x71FFB8 — CNpcPool destructor, `dword_CD7590 = 0`

`sub_44F5DF` faults on its **first** dereference of that pointer
(`0x44f5e2 cmp dword ptr [esi+4], 0`), so at crash time `dword_CD7590` is null.

**Therefore: opcode `0x116` (SpawnNPC) reached `CNpcPool::OnPacket` while no
CNpcPool existed** — the field was constructed far enough to install
`CField::OnPacket` as the stage handler, then torn down (or never finished
building its pools) before the enter-field burst was processed. The NPC packet
is the victim, not the cause; the failure is in the SetField / enter-field
sequence.

Opcode routing is correct and is not the fault: `template_jms_185_1.json` has
`SetField 0x7B`, `SpawnMonster 0xFD`, `SpawnNPC 0x116`,
`SpawnNPCRequestController 0x118` — all inside the client's expected ranges
(`CClientSocket::ProcessPacket` @0x4b1887 sends `0x1B..0x7A` to `CWvsContext`
and everything else to the stage; `CField::OnPacket` ranges above).

## Ruled out (with evidence)

1. **NpcSpawn body shape is correct for JMS 185.** `CNpc::Init` @0x716da2 reads
   `Decode2` @0x716dd6, `Decode2` @0x716de4, `Decode1` @0x716e0c,
   `Decode2` @0x716e1c, `Decode2` @0x716e3c, `Decode2` @0x716e4a,
   `Decode1` @0x716edf — exactly what
   `libs/atlas-packet/npc/clientbound/spawn.go` writes for a JMS tenant
   (`hasEnabledFlag` → `true`). `docs/packets/audits/jms_v185/NpcSpawn.json`
   used to record the body as an "opaque npc position/appearance block"; the
   export entry and the report have since been expanded to the seven per-field
   reads (see §7).

2. **The SetField *frame* is correct for JMS 185.** `CStage::OnSetField`
   @0x7eea69 reads, in order: `CClientOptMan::DecodeOpt` @0x7eea92,
   `Decode4` @0x7eeaa6 (channel), `Decode1` @0x7eeab6, `Decode4` @0x7eeac1,
   `Decode1` @0x7eeae4 (sNotifierMessage), `Decode1` @0x7eeaf1
   (bCharacterData), `Decode2` @0x7eeb08 (nNotifierCheck, gates a DecodeStr
   list), then — when bCharacterData — `Decode4` ×3 @0x7eebac/0x7eebb6/0x7eebcb
   (seeds), `CharacterData::Decode` @0x7eebf4,
   `CWvsContext::OnSetLogoutGiftConfig` @0x7eebfc, and finally
   `CInPacket::DecodeBuffer(p, 8)` @0x7eed25. That is field-for-field the JMS
   branch of `libs/atlas-packet/field/clientbound/set_field.go`.

3. **Rings (`60d999251`) changed no bytes for this character.**
   `GET /api/rings?filter[characterId]=40` → `"data":[], "total":0`; empty
   `model.RingRecords.EncodeRecords` emits the same `WriteShort(0)` ×3 as the
   pre-task stub (`libs/atlas-packet/model/ring.go:255-270`), and an all-nil
   `RingSet.EncodeField` emits three zero bytes on JMS as on GMS.

4. **spawnPoint plumbing (`051617c5e`, the deployed image) changed no bytes for
   this character.** It replaced `character.Model.SpawnPoint()`'s hardcoded `0`
   with the real field, narrowed via `byte(...)` in
   `services/atlas-channel/…/socket/writer/character_data.go:48`.
   `GET /api/characters/40` reports `spawnPoint = 0`, so the emitted byte is
   unchanged. (Still worth re-checking if the character's location-scoped
   spawn point differs from the base resource's.)

5. **`845a5c992` (gms_v95 verification batch)** touched no JMS template and no
   shared codec — its only non-test source change is
   `libs/atlas-packet/test/movement_types.go`.

6. **task-251 (player NPCs) is unmerged** — branch only.

7. **`CharacterData` is correct for JMS 185** (this character's shape, swept
   field-by-field against `CharacterData::Decode` @0x5137af with
   `bBackwardUpdate = 0`). So are the two neighbours that were unchecked:
   `CClientOptMan::DecodeOpt` @0x4ae41d consumes exactly the two bytes
   `set_field.go` writes when the leading short is 0, and
   `CWvsContext::OnSetLogoutGiftConfig` @0xae81c0 reads exactly four
   `Decode4`s. Full derivation, plus two *latent* JMS divergences that this
   character cannot reach (Dual-Blade master level; the extended-SP arm for JMS
   jobs 3xxx/22xx/2001), in
   [`characterdata-jms185-verification.md`](characterdata-jms185-verification.md).

## Next unit

The SetField payload is exonerated end to end. The remaining candidates are in
the **enter-field burst** that follows (`SpawnForSelf` fan-out at 23:13:11.597),
not in SetField:

- a packet in the burst routed to `CField::OnPacket` *before* `set_stage`
  @0x7eed4e finishes constructing the field and its pools;
- another op in the burst whose body desyncs the stream, so the field is torn
  down between the stage swap and the `0x116` delivery;
- the unhandled client → server ops `0xEA` / `0xDA` at 23:13:12.7.

## Fix

Not yet determinable — the specific field is unidentified. File inventory to be
filled in once the desync is located.

## Not yet answered

- Which packet in the enter-field burst reaches `CField::OnPacket` before the
  field's pools exist (CharacterData, DecodeOpt and the logout-gift block are
  all now ruled out).
- Whether ops `0xEA` / `0xDA` (client → server, unhandled on JMS 185) are a
  normal post-load pair or a symptom.
- Exact last-known-good build. User reports "days ago"; the deployed image at
  crash time was `main-051617c`.
- Whether the same crash is reachable on other client versions.
