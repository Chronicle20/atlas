# bug: JMS 185.1 client crashes on entering a channel

Status: **crash mechanism proven; the SetField payload is exonerated; the
specific trigger is not yet identified.** CharacterData × jms_v185 was swept
field-by-field and is clean (see §7 under "Ruled out"). Suspicion has moved to
the enter-field burst (see `## Next unit`).

> **Round 3 supersedes part of this file.** The faulting packet is
> `SpawnNPCRequestController` (**0x118**), not `SpawnNPC` (0x116), and the
> client is in its normal message loop at the time — not mid-`OnSetField`.
> See [`## Round 3`](#round-3--packet-content-log--fresh-crash-stack) at the
> bottom; read that before acting on anything above.

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

## Controller note after the CharacterData verification (task-273, round 2)

CharacterData, DecodeOpt and OnSetLogoutGiftConfig are all clean on JMS 185
(see `characterdata-jms185-verification.md`), so the SetField payload is fully
exonerated. That forces a re-reading of the timeline, and it points somewhere
different from the "pools not built yet" hypothesis:

**The field was alive before it died.** The client transmitted ops `0xEA` and
`0xDA` at 23:13:12.719/.729 — over a second *after* the enter-field burst at
23:13:11.597. A client that never finished building its field does not send
field-stage packets. So `dword_CD7590` was non-null at 23:13:12.7 and null a
moment later: the CNpcPool was **destructed** (`sub_71FF65` @0x71FFB8), not
never-constructed. Something tore the field down, and a later SpawnNPC arrived
against the corpse.

That reorders the candidates. The next unit should ask, in this order:

1. **What tears the field down here?** In `CStage::OnSetField` @0x7eea69 the
   client first swaps to a `CInterStage` (`set_stage` @0x7eecf3, guarded by the
   `off_CD7770` check @0x7eecc4) and only then installs the real field
   (`set_stage` @0x7eed4e). A *second* SetField — or any stage transition —
   destroys the current CField and with it the pool. Check whether atlas-channel
   emits a second SetField, a field transition, or any stage-changing writer to
   this session after 23:13:11.6.
2. **Is a SpawnNPC arriving late?** The `CHARACTER_ENTER` consumer at
   23:13:11.790 ("Processing character [40] entering map [10000]") runs a second
   fan-out after the `SpawnForSelf` one at .597. If either path emits SpawnNPC
   after a teardown, that is the crash.
3. **Identify client→server ops `0xEA` and `0xDA` on JMS 185** — they are the
   last thing the client did before dying and are currently unhandled by
   atlas-channel. If `0xEA` is a transfer-field / portal-script request, the
   client may be initiating the very transition that tears the field down while
   the server, having no handler, never completes it.

The pod logs for the reproduction are gone with the pod; a re-test will be
needed to confirm what the server sends in the second window.

## Round 3 — packet-content log + fresh crash stack

Two new sources: per-tenant packet-content debug logging (landed on `main` in
09c886f7e) and a fresh x32dbg call stack from the user for the *same* build.

### Reproduction (fresh)

Pod `atlas-channel-9c768dd44-56jz9`, namespace `atlas-main`. Two consecutive
crashing sessions; the second is transcribed below.

- Session `5ee750c0-6e22-43a5-a814-9080f7d5819d`, character 40, map 10000,
  world 0 / channel 0.

| time (UTC) | direction | packet |
|---|---|---|
| 13:20:52.635 | → | `<hello>` (16 B) |
| 13:20:53.143 | ← | `CharacterLoggedInHandle` op 0x07 |
| 13:20:53.378 | → | `SetField` **0x7B** (720 B) |
| 13:20:53.379 | → | `BuddyOperation` 0x39 |
| 13:20:53.385 | → | `CharacterBuffGive` 0x1E |
| 13:20:53.392–.393 | → | `CharacterKeyMap` **0x170**, `…AutoHp` 0x171, `…AutoMp` 0x172 |
| 13:20:53.395–.396 | → | `SpawnNPC` **0x116** ×3 (SN 1/2/3, tmpl 0x835/0x834/0x7D7) |
| 13:20:53.400 | → | `SpawnNPCRequestController` **0x118** ×3 (SN 2/1/3, control byte `01`) |
| 13:20:53.401 | → | `CharacterSkillMacro` 0x7A |
| 13:20:53.412 | → | `WorldMessage` 0x3E, `FieldTransportState` 0x92 |
| 13:20:53.599 | → | `StatChanged` 0x1D — **last server packet of the session** |
| 13:20:54.171 | ← | op **0xEA** (unhandled) |
| 13:20:54.178 | ← | op **0xDA** (unhandled) |
| 13:20:56.001 | ← | op **0xEA** (unhandled) |
| 13:20:56.007 | — | Connection ended (client dead) |

Full hex for every packet above is reproducible from the pod log with
`[PKT OUT]` / `[PKT IN ]` and the session id.

### 0xEA and 0xDA identified — they are a client-side clock for `CField::Init`

Both are bodiless client→server ops, and in the JMS 185 IDB (session
`a977912e`, `MapleStory_dump_SCY.exe.i64`) each has exactly one emitter, and
both emitters are reached only from `CField::Init` (`0x56186b`):

- `0xEA` — `CWvsContext::SendCancelPartyWanted` @0xb29783 (`COutPacket(0xEA)`),
  called from `CWvsContext::StopPartySearch` @0xaf6517, called from
  `CWvsContext::OnEnterField` @0xae81f7, whose **only** xref is
  `CField::Init` @0x561996.
- `0xDA` — `CUserLocal::ResetNLCPQ` @0xa2f78b (`COutPacket(0xDA)`), whose
  **only** xref is `CField::Init` @0x561a50.

`0xEA` also has a second reachable emitter path — `sub_AE84F4` @0xae8afe, the
leave-game path invoked from `set_stage`'s "stage is not field/cash-shop/ITC"
arm — which is the likely source of the *second* 0xEA at 13:20:56.001 (client
tearing down).

**Consequence:** the client did not enter `CField::Init` until **13:20:54.171**
— 776 ms *after* the server finished sending the enter-field burst, and 793 ms
after `SetField`. The whole burst was written to a socket whose peer had no
field object yet.

### Fresh crash stack (user-supplied, x32dbg, same build)

Main thread 16092, module `maplestory` (base 0x400000, so addresses map 1:1 to
the IDB):

```
0044F5E2   sub_44F5DF+3            <- access violation
007207B3   CNpcPool::SetLocalNpc ret site, inside CNpcPool::OnNpcChangeController
0056E985   CNpcPool::OnPacket ret site, inside CField::OnPacket
004B18A9   CClientSocket::ProcessPacket
004B17C3   CClientSocket::ManipulatePacket
004B1378   CClientSocket::OnRead
00AE25DE   CWvsApp::WindowProc
           user32.DispatchMessageA  <- normal message loop
```

Two things differ from the round-1 stack recorded under "## Observed":

1. **The faulting packet is 0x118 `SpawnNPCRequestController`, not 0x116
   `SpawnNPC`.** `CNpcPool::OnPacket` @0x720430 routes `case 0x118` to
   `CNpcPool::OnNpcChangeController` @0x720782, which reads
   `Decode1` (control) then `Decode4` (SN) and — because our control byte is
   `01` — calls `CNpcPool::SetLocalNpc` @0x720242 at 0x7207ae (ret 0x7207b3).
   `SetLocalNpc`'s first act is `sub_44F5DF(this + 1, &v11, &v10)`, and
   `sub_44F5DF` faults at its first dereference (+3).
2. **The client is in its normal `DispatchMessageA` message loop**, not nested
   inside `CStage::OnSetField`. So this is an ordinary socket read after the
   field was already constructed.

### What this rules out, and the contradiction it exposes

Routing is correct. `CField::OnPacket` @0x56e721 was decompiled in full; the
ranges are 0x9E–0xFC `CUserPool`, 0xFD–0x115 `CMobPool`, **0x116–0x11D
`CNpcPool` (`dword_CD7590`)**, 0x11E–0x120 `CEmployeePool`, 0x121–0x122
`CDropPool`, 0x123–0x125 `CMessageBoxPool`, 0x126–0x127 `CAffectedAreaPool`,
0x128–0x129 `CTownPortalPool`, 0x12D–0x130 `CReactorPool`, 0x7B–0x7D
`CStage`, 0x170–0x173 `CFuncKeyMappedMan`. Every opcode we sent lands where
the template says it should. `CClientSocket::ProcessPacket` @0x4b17eb confirms
0x1B–0x7A → `CWvsContext` and everything else → the stage.

`CInterStage` does **not** inherit `CField::OnPacket` — the 33 xrefs to
0x56e721 do not include the `CInterStage` vtables at 0xbe7c78/0xbe7c7c/
0xbe7c80/0xbe7ccc. So the "packet arrived while the InterStage was up" variant
is dead.

**The contradiction to resolve next:** `CNpcPool::OnNpcEnterField` @0x72068f
opens with the *same* `sub_44F5DF(this + 1, …)` call as `SetLocalNpc`. If
`dword_CD7590` were null, the first of the three 0x116 packets — sent 5 ms
*before* the 0x118s and necessarily processed first — would have faulted at
the same instruction. It did not. Either the pool pointer is non-null but
stale/freed, or the three 0x116 packets were consumed in an earlier pump while
the pool was still valid and the pool died between the two groups.

Pool lifetime, established from the IDB:

- `dword_CD7590` is written in exactly two places, the `CNpcPool` ctor
  `sub_71FED9` @0x71fef2 and dtor `sub_71FF65` @0x71ffb8.
- ctor reached only via `sub_AFCAA0` ← `sub_AFA942` (a `ZRef` reset: release
  old, then construct new), which has two callers:
  `CWvsContext::OnEnterGame` @0xae7d01 and `sub_AE8325` @0xae835c.
- `set_stage` @0x7effc0 calls `sub_AE8325` when the incoming stage is a
  `CInterStage` **and** `GetCharacterData()->p != 0`, and calls
  `CWvsContext::OnEnterGame` when the incoming stage is a
  `CField`/`CCashShop`/`CITC` **and** `GetCharacterData()->p == 0`.
- `set_stage` nulls `g_pStage` on entry (0x7effec) and restores it at 0x7f011c,
  immediately before calling `CField::Init` at 0x7f012c. So pool creation
  always precedes `g_pStage` becoming a `CField`.
- Teardown: `sub_AE8B31` (→ pool dtor) from `sub_AE84F4`, which `set_stage`
  calls when the incoming stage is null or is none of field/cash-shop/ITC.

`CStage::OnSetField` @0x7eea69 skips its own `CInterStage` swap when `g_pStage`
is already a `CInterStage` (guard @0x7eecc4) — and on a channel enter it always
is, because `CClientSocket::OnMigrateCommand` @0x4b1924 installs a
`CInterStage` at 0x4b1991 before reconnecting. `sub_7F019A` @0x7f019a only
allocates a *local* `ZRef<CharacterData>`; it does not install into
`CWvsContext`, so `GetCharacterData()->p` should still be 0 at `set_stage(field)`
and `OnEnterGame` should fire. That is why "the pool was never created" does
not yet add up either.

### Not yet answered (round 3)

- The register state at the fault. **`ESI` at 0x44F5E2, and the dword at
  `0x00CD7590`, decide between "null pool" and "stale/freed pool" and are the
  single cheapest next datum.**
- Whether the three 0x116 packets were processed successfully before the fault
  (they should have been, from the same socket read).
- Why the client took 793 ms after `SetField` to reach `CField::Init`, and
  where the burst was buffered in the meantime.

### Fix

Not yet determinable. Do not dispatch an implementer against this file until
the ESI / `dword_CD7590` reading above resolves the null-vs-stale question.

## Round 4 — the pool really is null; narrowed to one binary fact

Second x32dbg capture (user, re-run) settles the null-vs-stale question from
round 3, and reverses part of round 3's reasoning.

### Registers at the fault

```
EIP  0044F5E2      cmp dword ptr ds:[esi+4],0
ESI  00000004      <- this + 1, so this == 0
ECX  00000004
EDI  00000000
dword ptr ds:[esi+04]=[8]=???      (x32dbg: unreadable)
```

Stack window at the fault:

```
[esp]    00000116                     <- nType
[esp+4]  007206B9  return to maplestory.007206B9   (CNpcPool::OnNpcEnterField)
...      007204A4  return to maplestory.007204A4   (CNpcPool::OnPacket)
```

So **`dword_CD7590` is null**, confirming the round-1 conclusion by direct
observation rather than inference. `sub_44F5DF` is a hash-map probe
(`__lrotr(key,5)` / `div [esi+8]` / bucket array at `[esi+4]`); with `this == 0`
it faults reading address 8.

This capture faulted on **0x116** (`OnNpcEnterField`); the round-3 capture
faulted on **0x118** (`OnNpcChangeController` → `SetLocalNpc`). Both handlers
open with the identical `sub_44F5DF(this + 1, …)`, so whichever NPC packet is
processed first dies. Round 3's "the 0x116s must have succeeded" argument is
therefore **withdrawn** — they do not succeed; the crash is on the first NPC
packet of the burst either way.

User-reported symptom matches: "I can't see anything, the screen is black.
Sometimes I just see the map background then crash." No pools ⇒ nothing to
render.

### The pool is never constructed

`CWvsContext::OnEnterGame` @0xae7c9f was decompiled in full. The `CNpcPool`
allocator `sub_AFA942` is called **unconditionally** at 0xae7d01, in the opening
run of pool constructions (`sub_AFA8DC` … `sub_AFAAA7` @0xae7ceb–0xae7d4e).
There is no guard. So if `OnEnterGame` ran, `dword_CD7590` would be non-null.
It is null ⇒ **`OnEnterGame` did not run**.

`set_stage` @0x7effc0 creates the pools through exactly two mutually exclusive
arms, keyed on `CWvsContext`'s `CharacterData`:

| incoming stage | condition | pool creator |
|---|---|---|
| `CInterStage` (`off_CD7770`) | `GetCharacterData()->p != 0` | `sub_AE8325` @0xae835c |
| `CField` / `CCashShop` / `CITC` | `GetCharacterData()->p == 0` | `CWvsContext::OnEnterGame` @0xae7d01 |

The two conditions are complementary, so a single `set_stage` always creates
the pools — **unless** `CharacterData` is null at one `set_stage` and non-null
at a later one. Skipping both requires that null→non-null flip to land between
two `set_stage` calls.

### Leading hypothesis (unconfirmed)

A channel enter opens exactly that gap:

1. `CClientSocket::OnMigrateCommand` @0x4b1924 installs a `CInterStage` at
   0x4b1991 (guarded at 0x4b1954: only if `g_pStage` is not already one), then
   `CWvsContext::IssueConnect` to the channel. At this `set_stage`,
   `CharacterData` is null → `sub_AE8325` is **not** called → no pools.
2. The channel's `SetField` arrives. `CStage::OnSetField` @0x7eea69 decodes
   `CharacterData` (0x7eebf4) *before* its InterStage guard at 0x7eecc4.
3. That guard skips creating a second `CInterStage` because `g_pStage` already
   is one — so `sub_AE8325` is **not** called here either.
4. `set_stage(field)` @0x7eed4e: if `CharacterData` is now non-null,
   `OnEnterGame` is **not** called.
5. `g_pStage` becomes the `CField`, `CField::Init` runs (emitting 0xEA at
   0x561996 and 0xDA at 0x561a50 — both observed at 13:20:54.171/.178), and
   `dword_CD7590` is still 0. The first 0x116/0x118 crashes.

The hole in this story: `sub_7F019A` @0x7f019a, the allocation preceding
`CharacterData::Decode` in `OnSetField`, writes into a **local** `ZRef`, not
into `CWvsContext`. If nothing installs the decoded `CharacterData` into
`CWvsContext` before 0x7eed4e, step 4's condition is false and `OnEnterGame`
should fire. So either something else installs it, or the pool is constructed
and then destroyed.

### The one experiment that collapses this

Two breakpoints in x32dbg, then log in:

- `0x0071FED9` — `CNpcPool` constructor (writes `dword_CD7590 = this` @0x71fef2)
- `0x0071FF65` — `CNpcPool` destructor (writes `dword_CD7590 = 0` @0x71ffb8)

Outcomes:

- **Neither hit** → the pool is never created; `OnEnterGame`/`sub_AE8325` were
  both skipped, and the fix is on the `CharacterData`-polarity path above.
- **Ctor hit, then dtor hit** → the pool is created and torn down; the call
  stack at the dtor names what tears it down.
- **Ctor hit, no dtor** → `dword_CD7590` is being clobbered by something other
  than the ctor/dtor pair, and the write is a memory-write breakpoint away.

### Fix

Still not determinable; do not dispatch an implementer. The breakpoint result
above is the gate.
