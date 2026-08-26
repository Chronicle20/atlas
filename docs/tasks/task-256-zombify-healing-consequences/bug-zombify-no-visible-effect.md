# bug: zombify (UNDEAD) shows no effect on the diseased character, self or observer

Task: task-256-zombify-healing-consequences
Environment: `atlas-pr-1449` (PR #1449), tenant `5ae4fd69-6971-4c88-aa5c-d55a8c861cd2`, GMS **83.1**
Filed: 2026-08-26

## Reproduced

Partially — the server-side half is reproduced from live logs; the client-side
half (what the v83 client does with what we send) is **not yet verified**.

`@disease <name> zombify` in map 240011000, `atlas-pr-1449`:

```
18:30:42.804  atlas-channel  [CharacterChatGeneralHandle] read [msg [@disease chronicle zombify] ...]
18:30:42.918  atlas-buffs    APPLY  {"fromId":0,"sourceId":0,"level":1,"duration":10000,
                                     "changes":[{"type":"UNDEAD","amount":1}]}
18:30:42.974  atlas-channel  APPLIED {"characterId":3, ... "changes":[{"type":"UNDEAD","amount":1}]}
18:30:48.305  atlas-channel  Heal: caster=[3] level=[30] recipients=[2] perTarget=[2546] zombified=[true].
```

This confirms the plumbing fixed in `49465ebac`
(`bug-disease-command-emits-nonexistent-zombify-stat.md`) has reached the
namespace: the stat name is now `UNDEAD`, no "cannot find it" error is logged,
and `buff.IsZombified` resolves `true` on the Heal path. The *functional*
consequence works. Only the visual is missing.

## Observed

No icon, no animation, no state change is visible on the zombified character —
neither in that character's own client nor in an observer's.

## Expected

Some client-visible indication that the character is zombified, for the diseased
character and for observers in the map. Exactly what the v83 client is capable of
rendering here is the open question below.

## Root cause

**Established**, from the GMS v83 IDB (`v83_Me/MapleStory_dump.exe.i64`,
ida-pro-mcp session `754107bf`).

Atlas emits the `UNDEAD` two-state base block **fully zeroed** — `nOption = 0`,
`rOption = 0`. The v83 client keys *both* of its zombify visuals off that block's
`rOption`, so zero suppresses both. The client is not missing the feature; we are
sending it nothing to render.

### Where the observer's animation is lost

`CUser::UpdateAffectedSkillList` @0x93e344 builds the affected-skill animation map
from a fixed list of `SecondaryStat` field offsets. The two-state array begins at
byte 3236 (`DecodeForRemote` @0x788156 walks it as `v65 = this + 809`, `v65 += 2`,
7 iterations), so slot *i* is at `3236 + 8*i`. Slot 5 = GuidedBullet at 3276 —
which is the offset `OnTemporaryStatSet` uses for `CMob::SetGuided`, confirming
the stride. Slot 6 = **Undead at 3284**, and it has its own entry:

```c
if ( *ZFatalSection::Lock(*(v4 + 3284)) )      // Undead slot set
{
  v5  = *(v4 + 3284);
  v38 = 0;
  v6  = sub_672293(v5);                        // -> &this[4]  == rOption
  ZMap<long,int,long>::Insert(&v34, v6, &v38); // animation key
}
```

`sub_672293` @0x672293 returns `this + 4`; `sub_6635CD` @0x6635CD returns
`this[9]`, the GuidedBullet-only `dwMobId` that sits past the common base. Taking
index 0/1 as the presence flag the guard dereferences and 2/3 as `nOption`, index
4 is `rOption` — consistent with Atlas's own GuidedBullet note that "rOption
carries the source skill id (SetGuided reason)".

That map's keys are exactly the ones `CUser::ShowAffectedSkillAni` splits as
`id | (level << 16)` to index the MobSkill table — the composite that
`MobSkillReasonForeignValueWriter` already documents and writes for every other
disease. `CUser` is the base of both `CUserLocal` and `CUserRemote`, so this one
path covers the diseased character *and* observers. With `rOption = 0` the key
resolves to no mob skill and nothing is drawn.

### Where the character's own icon is lost

`CWvsContext::OnTemporaryStatSet` @0xa202be does not drive the buff/disease icon
bar off the mask. It walks the `ZMap<long, ZRef<VIEWELEM>>` that
`SecondaryStat::DecodeForLocal` populates, and branches on each element's reason
`v8`:

```c
if ( v8 >= 0 ) {
    if ( CSkillInfo::GetSkill(dword_BE78DC, v8) ) v56 = 2;   // player-skill icon
    if ( v56 <= 0 ) goto LABEL_26;                           // <-- skipped
} else {
    v56 = 1;                                                 // disease icon
}
...
if ( v56 == 1 ) v8 = -v8;
CTemporaryStatView::SetTemporary(..., v56, SkillLevel, ...);
```

A reason of **0** takes the `v8 >= 0` branch, `GetSkill(0)` fails, `v56` stays 0,
and `goto LABEL_26` drops the element before `SetTemporary` is ever reached. A
**negative** reason is what marks an element as a disease and produces the disease
icon.

### Why UNDEAD alone is affected

`UNDEAD` is in `baseStatNames`
(`libs/atlas-packet/model/character_temporary_stat.go:1128-1137`), so it is
excluded from the per-stat value loop at `:905-929` — the loop that writes
`Short(value) Short(mobSkillId) Short(mobSkillLevel) Int(duration)` for every
other disease and gives them their reason. Instead it falls to the `twoStateDynamic`
default in `getBaseTemporaryStats` (`:1396-1410`), which emits
`NewCharacterTemporaryStatBase(true, narrow)` — zeros — under a comment that says
so outright: *"no evidence was gathered for what their clients read."* The
evidence now exists, and it says the client reads `rOption`.

### Second, independent contributor on the GM-command path

The `@disease` command emits `sourceId: 0, level: 1` (see the live APPLY body
above). Even once the encoder forwards `rOption`, that path has no mob skill to
forward. The mob-inflicted path carries monster skill type 133
(`libs/atlas-constants/monster/skill.go:44,132-133`) and would supply a real one.

### Ruled out

### Ruled out

- **The stat never reaches the client at all.** It does. Both packets are sent
  unconditionally by `announceBuffGive`
  (`services/atlas-channel/atlas.com/channel/kafka/consumer/buff/consumer.go:99-122`):
  `CharacterBuffGive` to the owner's session and `CharacterBuffGiveForeign` to
  every other session in the map via `ForOtherSessionsInMap`. There is no
  disease-specific filter on either.
- **The mask bit is dropped on the foreign path.** It is not. `UNDEAD` is a
  member of `baseStatNames`
  (`libs/atlas-packet/model/character_temporary_stat.go:1128-1137`), and
  `EncodeForeign` (`:1091-1125`) emits the base-stat blocks after the two
  `nDefenseAtt`/`nDefenseState` bytes on every non-legacy version. On GMS v83
  `twoStateBaseStats` (`:1187-1219`) includes `Undead` as slot 7 — it is dropped
  only on GMS v61 and GMS v95+.
- **A stat-name mismatch.** Fixed in `49465ebac`; the live log above shows
  `UNDEAD` on the wire and `zombified=[true]` downstream.
- **Task-256 caused it.** It did not. task-256 only reads zombify state; it never
  touches the encode path.

- **The client cannot render zombify on v83.** It can. `UpdateAffectedSkillList`
  has a dedicated Undead entry, and the icon path accepts any negative reason.

## Fix

The wire change is one field. The cost is in re-pinning the fixtures, because the
affected cells are currently verified **against the zeros**.

| File | Change |
|---|---|
| `libs/atlas-packet/model/character_temporary_stat.go` | `getBaseTemporaryStats` `twoStateDynamic` branch (`:1396-1410`) — give `UNDEAD` the same treatment `EnergyCharge` already gets at `:1406-1409`: emit `NewCharacterTemporaryStatBaseWithOptions(true, s.Value(), s.SourceId(), narrow)` so `rOption` carries the mob-skill composite. Replace the "no evidence was gathered" comment with the `UpdateAffectedSkillList` @0x93e344 / `OnTemporaryStatSet` @0xa202be citations from this file. Leave `DashSpeed` / `DashJump` on the zeroed default — this evidence covers Undead only. |
| `libs/atlas-packet/model/character_temporary_stat.go` | `:259` — `UNDEAD` is registered `NoOpForeignValueWriter, NoOpForeignValueReader`. Confirm no change is needed: it is a base stat, so the foreign path reaches it through the base-block loop in `EncodeForeign`, not through the per-bit value writer. Do **not** move it out of `baseStatNames`. |
| `services/atlas-messages/atlas.com/messages/command/disease/commands.go` | The GM command emits `sourceId: 0`. Supply the real mob skill id + level for the typed disease so a GM-applied zombify has a reason to render, matching what the mob path sends. |
| `libs/atlas-packet/.../*_test.go`, `docs/packets/audits/` | Re-pin every affected byte fixture and evidence record. Any cell whose expected bytes contain the zeroed Undead block degrades on this change. |

The reason value must be the mob-skill composite `mobSkillId | (level << 16)`,
the same convention `MobSkillReasonForeignValueWriter` (`:306-329`) documents and
`ShowAffectedSkillAni` splits back apart — **not** the disease amount.

This change crosses a wire contract with verified coverage cells, so it is a
`packet-verifier` fan-out, not a plain implementer edit.

## Not yet answered

- **The icon half is unconfirmed.** Populating `rOption` definitively fixes the
  *animation*, because `UpdateAffectedSkillList` reads `rOption` straight off the
  stat. The *icon* runs through `VIEWELEM`s that `SecondaryStat::DecodeForLocal`
  builds, and `OnTemporaryStatSet` only classifies an element as a disease when
  its reason is **negative** (`v56 = 1; ... v8 = -v8`). Our `rOption` composite is
  positive, so the icon appears only if `DecodeForLocal` itself negates the reason
  for stats it treats as diseases — which is presumably how the other diseases get
  their icons, since Atlas writes their reason as the same positive composite. Two
  unknowns follow: whether `DecodeForLocal` registers a `VIEWELEM` for the
  two-state Undead slot at all, and whether it applies that negation to it.
  `DecodeForLocal` @0x781d0e is 0x4ce7 bytes and needs a scoped search rather than
  a whole-function decompile. **Cheapest discriminator is now the live re-test:**
  after the encode change, look at whether the animation plays, the icon appears,
  or both.
- The reference (`~/source/Cosmic`, `client/Character.java` `giveDebuff`) sends
  **only** the debuff stat — self plus foreign, no separate effect packet — so
  there is no precedent for adding an extra packet here.

## Resolution

Fixed by `2e91e814d` — "fix(atlas-packet,atlas-messages): populate UNDEAD rOption
so zombify renders". The `UNDEAD` two-state block now emits
`nOption = value`, `rOption = sourceId | (level << 16)`; `DashSpeed` / `DashJump`
keep the zeroed default and `UNDEAD` stays in `baseStatNames`. The `@disease` GM
command now supplies `monster.SkillTypeUndead` (133) as the buff APPLY `sourceId`
instead of `0`, so a GM-applied zombify has a MobSkill to resolve. Full detail in
`report-bug-zombify-encoder.md`.

Review: APPROVED_WITH_FINDINGS, 0 blocking / 1 non-blocking — see
`review-bug-zombify-encoder.md`. The reviewer verified the load-bearing
"no pre-existing fixture needed re-pinning" claim by **enumerating** every
UNDEAD-touching test pre-commit rather than trusting the green suite, and
hand-traced the seam atlas-messages `commands.go` → `buff.ApplyCommandProvider` →
atlas-channel's buff consumer `buff.NewBuff` → `character_buff_give.go`'s
`cts.AddStat` → the encoder's `s.SourceId()/s.Level()`, checking each hop's
positional signature for a swapped or dropped argument. The non-blocking finding
is the outstanding `packet-verifier` fan-out below.

Gate: `tools/verify.sh --quick --base b9904f7f6` — fails on the same
**pre-existing golangci-lint toolchain mismatch** already recorded against
`bug-heal-party-xp-magnitude.md` and
`bug-disease-command-emits-nonexistent-zombify-stat.md` (pinned v2.12.2 built with
go1.26, panics under the local go1.27.0). Because this commit touches
`libs/atlas-packet`, lint scope widened to the whole tree: the guard reports
`✗ lint & format guard (89 module(s))`, i.e. it panics on *every* module,
almost all of which this commit never touched. That is conclusive evidence the
failure is environmental rather than caused by this change.

The lint guard is the **only** failing check. `go build/vet` passed on all 89
modules, as did the go analyzer guards, skill/job id, scope, producer seam and
env domain guards.

Per CLAUDE.md "Done means verified", this fix is **not** cleared for PR until a
flagless `tools/verify.sh` exits 0.

### Still outstanding

1. **`packet-verifier` fan-out.** `BuffGive` / `BuffGiveForeign` cells for
   gms_v72, gms_v79, gms_v83, gms_v84, gms_v87, gms_v92, gms_v95, jms_v185 have no
   audited fixture exercising a *populated* UNDEAD block. The `gms_v48`
   "legacy path never reaches this block" claim is asserted **by inference only**
   and must be confirmed, not assumed. `docs/packets/audits/OPAQUE_LEDGER.md`
   cites the test file generally rather than pinning bytes — also for the verifier.
2. **The icon half**, unchanged from below. The animation path is now correct; the
   icon is not established. Live re-test discriminates.

### Settled

The `TwoStateTemporaryStat` field layout is no longer an inference.
`TemporaryStatBase<long>::DecodeForClient` @0x793ef2 decodes in wire order:

```c
CInPacket::DecodeBuffer(a2, this + 3, 4);   // nOption
CInPacket::DecodeBuffer(a2, this + 4, 4);   // rOption
this[5] = DecodeTime(a2);                   // tLastUpdated
```

`sub_672293` returns `this + 4`, so `UpdateAffectedSkillList` keys the animation
off **`rOption`**. Field order matches Atlas's `CharacterTemporaryStatBase`
encode order exactly, and the `twoStateDynamic` wrapper @0x793e28 appends the
`DecodeTime` + `Decode2` tail on top of that base.
- The reference (`~/source/Cosmic`, `client/Character.java` `giveDebuff`) sends
  **only** the debuff stat — self plus foreign, no separate effect packet — so
  there is no precedent for adding an extra packet here. Cosmic's
  `Disease.ZOMBIFY` value `0x4000` collides with `BuffStat.PUPPET` and carries no
  `MobSkillType`, i.e. it is dead code there and is **not** a usable source.
