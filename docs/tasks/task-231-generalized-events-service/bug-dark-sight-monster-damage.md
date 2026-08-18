# bug: monsters still damage a character in Rogue Dark Sight

**Reproduced:** tenant `15d000e1-260e-414b-bdf0-4c2b68b8995c`, GMS 83.1,
namespace `atlas-pr-1375`. Character 1 (Rogue-line, Dark Sight 4001003 at
level 20) casts Dark Sight and stands in a monster's path.

**Observed:** the character takes full touch damage while the Dark Sight buff is
active. From `atlas-channel` (times UTC, 2026-08-18):

```
13:31:52.661  Character [1] using skill [4001003] at level [20].
13:32:05.538  [CharacterDamageHandle] read [characterId [1], … nAttackIdx [-1], … damage [0] …]
13:32:24.679  [CharacterDamageHandle] read [characterId [1], … nAttackIdx [-1], … damage [1882] …]
13:32:24.687  Character [1] damage [1882] mitigated to hp [1882] mp [0] meso [0] reflect [0]
```

Level 20 Dark Sight lasts 200 s (confirmed by `atlas-buffs`: applied 13:21:05 →
`Expired buff for character [1] from [4001003]` 13:24:30, i.e. 205 s), so the
13:32:24 hit lands ~32 s into a 200 s window. An earlier hit at 13:28:04
(`damage [1910]`) likewise falls inside the 13:24:56 cast's window.

`nAttackIdx == -1` on every one of these — mob touch/collision damage, not a mob
skill.

**Expected:** monsters do not target or touch-damage a character in Dark Sight;
this is the skill's defining behavior.

**Root cause: RETRACTED (2026-08-18) — see `## Update 2026-08-18` at the
end of this file. Everything in this section and in `## Fix` below is the
FIRST pass's reasoning and is superseded.**

~~Root cause: ESTABLISHED.~~ The wire side was already settled — the local
`TemporaryStatSet` this server sends on a Rogue Dark Sight cast does carry
`DARK_SIGHT` with a non-zero value, so the packet-encoding-defect fix shape
is ruled out. The client side is now settled too, by decompiling the v83
client (`idb_list` session `41f09cce`,
`E:\Programs\Nexon\IDBs_v9\GMS\v83_Me\MapleStory_dump.exe.i64`): the function
that actually produces the mob-touch/collision damage report against the
local player (`sub_666362`, called from `CMob::Update`) never calls
`CUser::IsDarkSight` or `CUser::IsSneak` at all. The only place in the mob-AI
code that consults `IsDarkSight` is `CMob::IsTargetInAttackRange`
(`0x66a517`), which gates the mob's *named-skill attack selection*
(`nAttackIdx >= 0`, reachable via `CMob::TryDoingSkill` /
`CMob::GenerateMovePath` at `Update+0x634` / `0x667bdc`) — a different
codepath from the one that produced the `nAttackIdx == -1` reports in this
bug. So the v83 client's mob-touch/collision damage path genuinely does not
gate on Dark Sight; this is not a defect anywhere in what this server sends,
it is baseline v83 client behavior. The fix must therefore be server-side
suppression, as already scoped in `## Fix` below — nothing there needs to be
retracted. See "Client side: CONFIRMED (not gated)" for the decompile
evidence.

**Wire side: RULED OUT (packet-encoding defect).**
`libs/atlas-packet/model/character_temporary_stat.go`'s local `Encode` (the
path `writer.CharacterBuffGiveBody` calls, at line 879) is source-id-agnostic:
`DARK_SIGHT` is not in `baseStatNames` (line 1129), so every stat instance
walks the same per-stat loop at lines 906-930 regardless of which skill
sourced it — `WriteInt16(int16(v.Value()))` followed by the 4-byte source id
and the remaining-duration int32. The existing fixture
`TestNoExpiryStatEncodesSaturatedDuration`
(`libs/atlas-packet/model/character_temporary_stat_test.go:711-728`) drives
exactly this function with `AddStat(...)("DARK_SIGHT", 9101004, 1, 1, ...)`
and asserts the resulting byte layout is `mask(16) + value int16 + sourceId
int32 + expiry int32` — i.e. it already proves the local encoder writes a
non-zero `DARK_SIGHT` value block whenever `AddStat` is given a non-zero
amount. The only thing that fixture doesn't cover is the Rogue source id
(4001003 instead of 9101004), and the encode path never branches on source
id, so that's not a live variable.

Tracing the amount from the WZ statup down to that `AddStat` call, every hop
is a straight passthrough with no DARK_SIGHT-specific handling:
- `services/atlas-channel/atlas.com/channel/data/skill/effect/statup/rest.go`
  `Extract` copies `RestModel.Amount` (the API's `amount: 1`, already
  confirmed in the ruled-out list above) verbatim into `Model.amount`.
- `services/atlas-channel/atlas.com/channel/skill/handler/common.go:180`
  (the generic USE_SKILL buff-apply path Dark Sight goes through — it is not
  a mount, not Shadow Stars, not Echo of Hero) calls
  `buff.NewProcessor(...).Apply(f, characterId, skillId, level, e.Duration(),
  statupsToApply)` with `statupsToApply == e.StatUps()` unmodified.
- `services/atlas-channel/atlas.com/channel/character/buff/processor.go:79-82`
  → `ApplyCommandProvider`
  (`services/atlas-channel/atlas.com/channel/character/buff/producer.go:14-42`)
  puts `su.Amount()` straight into the Kafka command's `StatChange.Amount`.
- `services/atlas-buffs/atlas.com/buffs/kafka/consumer/character/consumer.go:66-68`
  copies `cs.Amount` straight into `stat.Model` with no filtering, and
  `buff.NewBuff` / `Registry.Apply`
  (`services/atlas-buffs/atlas.com/buffs/buff/model.go:158-174`,
  `services/atlas-buffs/atlas.com/buffs/character/registry.go:70-`) store the
  `changes` slice unmodified.
- `services/atlas-buffs/atlas.com/buffs/character/producer.go:16-23`
  (`appliedStatusEventProvider`) copies `su.Amount()` straight back into the
  `character2.StatChange` on the applied-status Kafka event.
- `services/atlas-channel/atlas.com/channel/kafka/consumer/buff/consumer.go`'s
  `announceBuffGive` (around line 93-97) builds `stat.NewStat(cm.Type,
  cm.Amount)` from that event's `StatChange` and hands it to
  `writer.CharacterBuffGiveBody`, which is the `cts.AddStat(...)` call this
  section started from.

So the amount 1 confirmed at the WZ/API layer survives unmodified through
every hop to the byte the client receives; there is no point in this chain
that zeroes or drops `DARK_SIGHT`'s value for a Rogue-sourced cast. The
byte-fixture pattern this exercises is identical to the one already asserted
for the GM-hide source. A packet-encoding defect is therefore ruled out as
the explanation for this bug.

**Client side: CONFIRMED (not gated) — v83 IDB, session `41f09cce`.**
`func_query` on `name_regex: "IsDarkSight"` resolves one symbol:
`CUser::IsDarkSight` at `0x4f0d45`. `xrefs_to` on that address returns 18
call sites; exactly one is inside mob-AI code:
`CMob::IsTargetInAttackRange` (`0x66a517`), at instruction `0x66a844`, inside
the per-`MobAttackInfo`-entry selection loop:

```
if ( (AttackInfo[36] || !v5->m_bDoFirstAttack)          /*0x66a854*/
  && !AttackInfo[4]
  && AttackInfo[5] <= _ZtlSecureFuse<long>(v5->_ZtlSecureTear_m_nMP, v5->_ZtlSecureTear_m_nMP_CS)
  && (a2 || !CAvatar::IsHideMorphed(v93 + 34))
  && (!v19[40] || v5->m_bAttackReady)
  && (a2
   || v19[7]
   || _ZtlSecureFuse<unsigned long>(v5->_ZtlSecureTear_m_dwMobID, v5->_ZtlSecureTear_m_dwMobID_CS) == (&loc_8F7111 + 1)
   || !CUser::IsDarkSight(v93) && !CUser::IsSneak(v93)) )
{
  break; /*0x66a854*/
}
```

`IsTargetInAttackRange` is called from three places (`xrefs_to 0x66a517`):
`CMob::Update+0x634` (`0x667bdc`, with `a2=3`), `CMob::TryFirstAttack`
(`0x66e41c`/`0x66e555`/`0x66e5b9`), and one unnamed function
(`sub_9BCB02`). At the `CMob::Update` call site, this function is used only
to gate `CMob::GenerateMovePath` — the mob's decision to move toward a
target and use one of its named `MobAttackInfo` skill attacks:

```
if ( CMob::TryDoingSkill(this, &v173, &v170, &a14)         /*0x667bdc*/
  || *(v171 + 592) && CMob::IsTargetInAttackRange(this, 3, &v173, &v170, &a14) )
{
  ...
  CMob::GenerateMovePath(this, v173.m_wstr, v135, ...);
}
```

That is a different attack class from the bug's symptom — `IsTargetInAttackRange`
walks `CMobTemplate::GetAttackInfo` entries, which are named skill attacks
that would surface as `nAttackIdx >= 0`, not the `nAttackIdx == -1`
touch/collision reports actually observed.

The function that produces the `nAttackIdx == -1` touch/collision report is
`sub_666362` (unnamed; called from `CMob::Update` at `0x6685c8` on every
tick — `sub_666362(this, v173.m_RefCount) /*0x6685c8*/`), which walks the
mob's linked list of `MobAttackInfo` entries (`this[52]`) and, for a
type-0 ("touch") entry whose body rect overlaps the local avatar's rect,
directly invokes `SetDamaged` on the local `CUser` through its vtable:

```
if ( dword_BF04A8(&v171, &v168, v113 + 2) )     /*rect-overlap test, touch branch, v117==0*/
{
  VecCtrl = CUser::GetVecCtrl(v191, &v189);
  v119 = *VecCtrl != 0 ? *VecCtrl - 12 : 0;
  if (v189) (*(*v189 + 8))(v189);
  if ( !*(a2 + 9) || *(v119 + 272) )
    (*(*v191 + 18))(v191, 0, 0, 0, 0, v193, v113[7], v113[6] != 0 ? -1 : 1, 0, 1, 1); /*0x66727d region*/
  v112 = v191;
}
```

`v191` is `dword_BEBF98`, the local `CUser*`; `(*(*v191 + 18))(...)` is the
vtable dispatch to `SetDamaged` (vtable slot 18 — matches
`CUserLocal::SetDamaged@0x9581a9` being a `virtual` per its `UAEX` mangling).
The full external-call list (`refs`) of `sub_666362`, decompiled in its
entirety, contains **no** call to `CUser::IsDarkSight` or `CUser::IsSneak`
anywhere — confirmed by inspecting the complete `refs` array returned with
the decompile, which enumerates every named symbol the function calls.

So: the mob-touch/collision damage path that produced this bug's
`nAttackIdx == -1` reports is client behavior, not gated on Dark Sight at
all in v83. The only Dark Sight gate that exists client-side governs a mob's
choice to move in and use a named skill attack — an entirely separate code
path from the one that actually fired in this reproduction.

What is ruled out, with evidence (prior to this pass):

- **The WZ effect data is correct.** `GET /api/data/skills/4001003` on this
  tenant returns, for every level, `"statups": [{"type":"SPEED", …},
  {"type":"DARK_SIGHT","amount":1}]`. The amount is 1, not 0 — which matters,
  because the v83 client's `CUser::IsDarkSight` tests the stat `!= 0`
  (`services/atlas-channel/atlas.com/channel/skill/handler/hide/hide.go:148-150`).
- **The buff is applied and expires on schedule.** `atlas-channel` logs
  `Character [1] applying effect from source [4001003]` on each cast, and
  `atlas-buffs` logs the matching expiry at the correct 200 s offset.
- **`DARK_SIGHT` is registered in the temporary-stat encoder.**
  `libs/atlas-packet/model/character_temporary_stat.go:91` registers it in the
  mask (non-diseased, `NoOpForeignValueWriter`), and it appears in
  `foreignReadOrder` at line 1020 as flag-only. So the stat has a mask bit on
  v83 and a flag-only foreign shape.
- **It is not GM-hide interference.** `atlas-monsters`' buff consumer reacts
  only to `SuperGmHide` and passes Dark Sight through untouched
  (`services/atlas-monsters/atlas.com/monsters/kafka/consumer/buff/consumer.go:65-80`);
  `buff.IsGmHidden` keys on `SourceId`, not on the `DARK_SIGHT` stat
  (`services/atlas-channel/atlas.com/channel/character/buff/hidden.go:18-19`).
- **There is no server-side suppression to have regressed.** The damage-taken
  pipeline (`services/atlas-channel/atlas.com/channel/socket/handler/character_damage.go`,
  `processDamageTaken`) has short-circuits for protective mist and for the
  `damage == -1` block sentinel, and a mitigation chain for Magic Guard /
  Power Guard / Achilles / etc. It has **no** `DARK_SIGHT` branch at all — this
  has never been implemented server-side.
- **Only one character was online** (character 1 throughout the session), so the
  attacking mob was controlled by the victim's own client. A remote controller
  not knowing about the buff cannot be the explanation here.

So the character's own client produced touch damage while its own Dark Sight was
active, and — per the wire-side finding above — it did so with a correctly
delivered, non-zero `DARK_SIGHT` local temp-stat already in hand.

## Fix

Both halves of the root-cause question are now closed, and they converge on
the same fix shape. The wire side rules out the packet-encoding-defect shape:
the local set already carries `DARK_SIGHT` with a non-zero value on every hop
from WZ amount to wire byte, so there is nothing to fix in the encoder. The
client side (see "Client side: CONFIRMED (not gated)" above) rules out
"the client already handles this, so a server-side change would be masking
the real fix" — the v83 client's mob-touch/collision damage path
(`sub_666362`, called every tick from `CMob::Update`) does not consult
`IsDarkSight` or `IsSneak` anywhere, so there is no client behavior to
preserve or avoid duplicating. The server must **suppress mob touch damage**
while the buff is active — this was never implemented anywhere, client or
server, for the touch/collision case:

- `services/atlas-channel/atlas.com/channel/socket/handler/character_damage.go`
  — a short-circuit in `processDamageTaken` alongside the existing protective-mist
  one, gated on an active `DARK_SIGHT` statup in `deps.getBuffs` and on
  `p.AttackIdx() == -1`.
- `extractBuffAmounts` in the same file — add the `DARK_SIGHT` case.
- `services/atlas-channel/atlas.com/channel/socket/handler/character_damage_test.go`
  — a case driving `processDamageTaken` with a Dark Sight buff and
  `attackIdx -1`, asserting `changeHP` is never called; must fail before and
  pass after.

## Not yet answered

The root-cause question (wire side and client side) is now fully closed —
see "Wire side: RULED OUT" and "Client side: CONFIRMED (not gated)" above.
Nothing further blocks writing the fix in `## Fix`.

Two smaller items remain open, neither blocking:

- Whether the two `damage [0]` reports at 13:32:05 and elsewhere inside a
  Dark Sight window are a partial suppression (mob bumped but dealt nothing)
  or unrelated. Do not treat "some hits are 0" as evidence the buff partly
  works without establishing what produces a 0-damage report. (`sub_666362`'s
  touch branch computes actual damage via a call further down the same
  function not quoted above — a 0-damage roll there is plausible and would
  not require any Dark Sight involvement; this was not traced in this pass.)
- `sub_666362` and `CMob::IsTargetInAttackRange` are both unnamed/partially
  named in the v83 IDB beyond what `func_query` resolved
  (`sub_666362` has no C++ symbol at all). This investigation was read-only
  and made no IDB changes, so a future pass may want to name `sub_666362`
  (e.g. `CMob::ProcessAttackInfoList` or similar, once its exact role is
  fully typed) to make it discoverable by name next time.

---

# Update 2026-08-18 — first pass retracted, new evidence

## What live testing showed

Reported by the tester on `atlas-pr-1375` (tenant
`15d000e1-260e-414b-bdf0-4c2b68b8995c`, GMS 83.1), after the first pass:

- Monsters spawned by the map's own life data: **do not** touch-damage a
  dark-sighted character.
- Monsters spawned by the GM `!spawn` command: **do not** either.
- A monster the character already has aggro on: **does** hit through Dark
  Sight (believed correct).
- Crimson Balrog (8150000) spawned by the CRIMSON_BALROG event: **does** hit
  through Dark Sight. The same template GM-spawned did not.
- When the client behaves correctly, **no damage report reaches the server at
  all** — so the suppression that works in the other cases is client-side.

That refutes the first pass's conclusion ("the v83 client never gates
touch damage; the fix must be server-side suppression"). The client does
suppress it; the first pass looked in the wrong function. Do NOT implement
the server-side `DARK_SIGHT` short-circuit described in `## Fix` above on the
strength of that reasoning.

## Where the client gate actually is

Full decompile evidence in
[`client-analysis-dark-sight-touch-damage.md`](client-analysis-dark-sight-touch-damage.md).
Summary: the only mob-related call to `CUser::IsDarkSight` (`0x4f0d45`) in the
whole binary is inside `CMob::IsTargetInAttackRange` (`0x66a517`). While the
player is dark-sighted, no attack index validates there, so
`CMob::GenerateMovePath` never calls `CMob::DoAttack` (`0x66d9c0`) — the ONLY
producer of the `CMob::m_lpLayerASAni` (offset `0xE4`) hit-record queue that
`sub_666362` drains to dispatch `CUserLocal::SetDamaged`. The suppression is
"the mob never arms an attack against an invisible player," not a check at
the moment of contact. The wire's `nAttackIdx == -1` is `SetDamaged`
reporting a type-0 (plain contact) attack, not a missing attack index.

## The server-side delta between the working and broken spawn paths

All three spawn paths converge on the same code
(`atlas-monsters` `handleSpawnFieldCommand` → `monster.Processor.Create`,
`monster/processor.go:216-225`), so only the command's field values differ:

| path | foothold sent | observed |
|---|---|---|
| map life data (`atlas-maps`, `data/map/monster/rest.go:14` `fh`) | real `fh` from Map.wz | Dark Sight works |
| GM `!spawn` (`atlas-messages`, `command/monster/commands.go:248-256`) | resolved via `POST data/maps/{id}/footholds/below` | Dark Sight works |
| CRIMSON_BALROG event (`atlas-events`) | **always 0** | Dark Sight fails |

The event path is the only one that never resolves a foothold:
`crimsonbalrog.Position` carries only `x`/`y`
(`services/atlas-events/atlas.com/events/events/crimsonbalrog/config.go:24-28`)
and `spawnFieldCommandProvider` leaves `Fh` (and `Team`) at their zero value
(`.../crimsonbalrog/producer.go:63-80`), even though
`monster.SpawnFieldCommandBody` has the field
(`.../kafka/message/monster/kafka.go:48-56`).

It compounds downstream: `atlas-channel`'s `SnapMobPosition` returns early
when `fh == 0` (`services/atlas-channel/atlas.com/channel/data/map/processor.go:96-98`),
so the spawn Y is never validated against the deck surface either.

Live `atlas-data` on `atlas-pr-1375`, queried against the seeded spawn points
(`deploy/seed/shared/all/events/definitions/event-crimson-balrog.json`):

- map 200090010, x=339 → foothold **10**, `(335,156)→(665,113)`; interpolated
  surface at x=339 is y≈155. The seed spawns at y=**148** — ~7px above the
  deck, with no foothold.
- map 200090000, x=-538 → foothold **10**, `(-769,113)→(-439,156)`;
  interpolated surface at x=-538 is y≈143. The seed spawns at y=143 — on the
  deck, but still with `fh = 0`.

`CMobPool::OnMobEnterField` → `CMob::Init` (`0x662981`-`0x66299a`) feeds an
unresolved spawn foothold through as `a3 = 0` into `CVecCtrlMob::Init`, so
`fh = 0` genuinely produces a differently-initialized mob physics state
client-side.

## Ruled out this pass

- **`spawnSourceType` is not a behavioral delta.** It is pure provenance in
  `atlas-monsters`: the only read outside logging/echo is the
  `DESTROY_BY_SOURCE` match (`monster/processor.go:1363`). GM omits it
  (normalized to `CYCLIC`, `kafka/consumer/monster/consumer.go:357-361`), the
  event sends `EVENT`; nothing branches on it.
- **`team` is not the delta.** GM and event both send `0`; only the map-life
  path sends `-1`. GM works, so `0` is not what breaks it.
- **Not the monster template by itself.** The same template (8150000)
  GM-spawned did not hit through Dark Sight.
- **Not a packet-encoding defect in the Dark Sight temp stat** — see the
  first pass's "Wire side: RULED OUT", which still stands.

## Root cause: NOT ESTABLISHED

`fh = 0` is the only behavioral difference found between a spawn path that
works and one that does not, and it is independently a defect. But the
decompile does not yet explain HOW it produces the symptom, and the obvious
mechanism is refuted: the whole attack-arming block in `CMob::Update`
(`TryDoingSkill(...) || (*(m_pvcActive-12+0x250) && IsTargetInAttackRange(...))`)
is itself gated on `*(m_pvcActive-12 + 0x18)` — the same flag
`CMob::TryFirstAttack` requires (`0x66e356`). An ungrounded mob does not skip
the Dark Sight check and still reach `DoAttack`; it reaches neither. If that
flag is "landed," `fh = 0` predicts FEWER touch hits, not more.

Do not write the fix as "this is the root cause" until one of the open
questions below closes it.

## Open questions, in priority order

1. **Was the GM-spawn control test run in the same boat map (200090010 /
   200090000) during a crossing, or in a different map?** If it was a
   different map, the control is much weaker and the boat map / ContiMove
   context is unexcluded.
2. **How does the event-spawned Balrog behave visually** — standing on the
   deck, sunk into it, floating, sliding, not moving? That distinguishes
   "ungrounded and never lands" from "lands immediately, `fh` irrelevant."
3. Is `AttackInfo[7]` (the per-attack "ignore stealth" override in the
   `IsTargetInAttackRange` gate) set on 8150000's contact attack? If it is,
   this template ignores Dark Sight unconditionally and the spawn path is a
   red herring — the GM-spawn control would then need re-running to confirm
   the negative was real.
4. Does the `-12`-adjusted `+0x18` flag ever get set for a mob initialized
   with `fh = 0`, and how fast? Setter not yet located; candidates
   `CVecCtrl::CalcFloat` (`0x9b2c3c`), `CVecCtrl::WorkUpdateActive`
   (`0x9b19d0`), `CVecCtrlMob::WorkUpdateActive` (`0x9bca2a`),
   `CVecCtrlMob::CtrlUpdateActiveMove` (`0x9bccaf`).
5. Is there a route to a queued `m_lpLayerASAni` node that bypasses the local
   client's `IsTargetInAttackRange` call — e.g. a server-driven
   `m_nOneTimeAction` change? Not ruled out.

## Fix (superseding the `## Fix` section above)

Independent of whether it proves to be this bug's root cause, the event path
sending `fh = 0` is a defect and should be fixed:

- `services/atlas-events/atlas.com/events/events/crimsonbalrog/producer.go`
  `spawnFieldCommandProvider` — populate `Fh`.
- Resolve it the way the other producers do, via
  `POST data/maps/{mapId}/footholds/below` on the `DATA` service. Mirror
  `services/atlas-messages/atlas.com/messages/data/foothold/`
  (`processor.go` / `requests.go` / `rest.go` + `mock/`); `atlas-events`'s
  existing `external/maps` package is the local convention for a REST client.
- Alternative, preferred if the invariant should hold for every producer:
  resolve it in `atlas-monsters` instead — `handleSpawnFieldCommand`
  (`kafka/consumer/monster/consumer.go:364-383`) or
  `monster.Processor.Create` (`monster/processor.go:216-225`), when the
  command's `fh` is 0. `atlas-monsters` already has `DATA` REST clients under
  `monster/information`, `monster/mobskill` to mirror. One enforcement point
  covers present and future producers; the lookup only runs when `fh == 0`,
  so the cyclic path pays nothing.
- Whichever side it lands on, a test must assert a non-zero `fh` reaches the
  spawn command / `Create` for an event-sourced spawn, failing before the fix.

Do NOT implement the first pass's server-side `DARK_SIGHT` short-circuit in
`character_damage.go` — it would mask a client-side gate that demonstrably
works for every other spawn path.
