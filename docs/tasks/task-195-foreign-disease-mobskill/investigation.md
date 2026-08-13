# task-195 — Foreign disease packets render no debuff state on observers

Resolves [#1196](https://github.com/Chronicle20/atlas/issues/1196).

All findings below are derived from the client binaries via ida-pro-mcp, not
from another server implementation. The issue text proposed a specific fix
(port a known-good v83 server's "write `mobSkillId + level` for **every**
disease"); the client disagrees with that in one place, and the client wins —
see §4.

---

## 1. How a remote observer renders a disease

Four client functions, in order (addresses are GMS v83,
`MapleStory_dump.exe.i64`):

| # | Function | Address | Role |
|---|---|---|---|
| 1 | `CUserRemote::OnSetTemporaryStat` | `0x98385d` | packet handler |
| 2 | `SecondaryStat::DecodeForRemote` | `0x788156` | the foreign CTS decoder |
| 3 | `CUser::UpdateAffectedSkillList` | `0x93e344` | collects the animation keys |
| 4 | `CUser::ShowAffectedSkillAni` | `0x932da6` | resolves the key → WZ → layer |

`OnSetTemporaryStat` decodes the CTS, reads one `Decode2` (tDelay), and calls
the `CUser::OnTemporaryStatChanged` virtual (`0x93cdbc`, vtable +32). That
function branches on only five mask bits (Aran combo, morph, ghost, …) — it
never touches a disease bit. Its last statement is
`CUser::UpdateAffectedSkillList(this, tDelay, 0)`, and **that** is the disease
render path.

`UpdateAffectedSkillList` builds a `ZMap<long,int,long>` of animation keys. It
has fourteen identical blocks of the form:

```c
if ( _ZtlSecureFuse<long>(pSS + <nOption>, *(pSS + <nOption>+8)) )   // stat present?
{
    v39 = _ZtlSecureFuse<long>(pSS + <rOption>, *(pSS + <rOption>+8)); // key = REASON
    ZMap<long,int,long>::Insert(&map, &v39, &v38);
}
```

The `<nOption>/<rOption>` byte offsets map one-to-one onto the fields
`DecodeForRemote` writes. Cross-referenced, the fourteen blocks are exactly:
**Seal, Stun, Weaken, Curse, Darkness, Poison, Slow, Seduce, StopPortion,
StopMotion, Fear, BanMap, Confuse**, plus one ride-vehicle entry. The key is
always the stat's **reason (`rOption`)** field, never its value.

`ShowAffectedSkillAni` then splits that key:

```c
v6 = *i;                      // the key
v8 = v7 >> 16;                // ==> mob skill LEVEL
v9 = sub_7632F2(v77[4]);      // MobSkill lookup by the LOW half ==> mob skill ID
if ( v9 && v8 <= levelCount )
{
    v12 = 92 * v8 + *(v9 + 4) - 92;   // level entry
    v13 = *sub_933527(v12, &v99);     // the "affected" WZ node
    ...
}
```

and the accompanying chat-log `switch` is over raw mob-skill ids — `0x78`(120)
Seal, `0x79`(121) Darkness, `0x7A`(122) Weaken, `0x7B`(123) Stun, `0x7C`(124)
Curse, `0x7D`(125) Poison, `0x80`(128) Seduce, `0x85`(133) Confuse,
`0x86`/`0x87`(134/135) StopPortion/StopMotion, `0x88`(136) Fear →
`CField::OnFearEffect`.

**So the reason field of a foreign disease must be `mobSkillId | (level << 16)`.**
This is the same composite the already-working local path emits as two shorts
(`Encode` writes `Short(sourceId) + Short(level)` where the client does one
`Decode4`).

GMS v95 settles it by name — its PDB-backed symbols call the `Decode4` targets
`rStun`, `rSeal`, `rDarkness`, `rWeakness`, `rCurse`, `rPoison`, `rAttract`,
`rReverseInput` (`SecondaryStat::DecodeForRemote` `@0x72b7b0`).

## 2. What Atlas was writing instead

| stat | old foreign writer | bytes | what the client wanted |
|---|---|---|---|
| STUN, SEAL, DARKNESS, WEAKEN, CURSE | `ValueAsInt` → `Int(value)` | 4 | `Short(mobSkillId) + Short(level)` |
| POISON | `ValueSourceLevel` → `Short(value)+Short(level)+Short(src)` | 6 | `Short(value) + Short(mobSkillId) + Short(level)` |
| SEDUCE, CONFUSE | `LevelSource` → `Short(level)+Short(src)` | 4 | `Short(mobSkillId) + Short(level)` |

Every one of them is the right **size** — which is why the packet never
desynced and why the bug looks like "nothing happens" rather than a crash. The
contents were simply not a mob-skill key, so `sub_7632F2` resolved nothing and
no layer was created.

## 3. Second defect found in the same encoder: write order

`SecondaryStat::DecodeForLocal` reads per-stat blocks in **mask/shift order**
(v83 code positions ascend with shift: Stun@`0x782d9d` < Poison@`0x782ea2` <
Combo@`0x783157` < Weaken@`0x783976` < Slow@`0x783b44`).

`SecondaryStat::DecodeForRemote` does **not**. It is a hand-written sequence:

```
Speed(7), Combo(21), WeaponCharge(22), Stun(17), Darkness(20), Seal(19),
Weaken(30), Curse(31), Poison(18)×2, ShadowPartner(26), DarkSight(10),
SoulArrow(16), Morph(33), GhostMorph(49), Seduce(39), ShadowClaw(40),
BanMap(46), Barrier(50), DojangShield(62), Confuse(51), RespectPImmune(53),
RespectMImmune(54), DefenseAtt(55), DefenseState(56), …
```

(v83 code positions: Combo@`0x7881b0` < Stun@`0x788234` < Weaken@`0x78830f` <
Poison@`0x7883a1` — decisively not shift order.)

`EncodeForeign` was sorting by shift, so **any two value-carrying stats present
at once swapped their payloads**. For diseases that is the common case: SEAL +
DARKNESS, POISON + WEAKEN, STUN + anything. Fixing the reason content without
fixing the order would leave two-disease cases still rendering the wrong
animation.

Verified identical relative order on every supported client:

| version | function | evidence |
|---|---|---|
| gms_v48 | `sub_5CBA1F` | 8-byte mask; bit tests in order `0x80, 0x200000, 0x400000, 0x20000, 0x100000, 0x80000, 0x40000000, <0, 0x40000×2, …` = Speed, Combo, WeaponCharge, Stun, Darkness, Seal, Weaken, Curse, Poison |
| gms_v61 | `SecondaryStat::DecodeForRemote` `@0x667c5f` | same sequence, ends after DefenseState |
| gms_v72 | `@0x6cfe78` | same (byte-identical size `0x760` to v79) |
| gms_v79 | `@0x701539` | 28 mask-test blocks, same sequence |
| gms_v83 | `@0x788156` | symbolized `CTS_*` globals, sequence above |
| gms_v84 | `@0x7ac409` | structural clone of v83 + 3 trailing stats |
| gms_v87 | `@0x7d8533` | same global sequence |
| gms_v92 | `@0x711240` | same global sequence (globals align 1:1 with v95's `CTS_*` addresses) |
| gms_v95 | `@0x72b7b0` | PDB-symbolized, sequence above + the v95 tail |
| jms_v185 | `@0x804dbf` | checked-in export `docs/packets/ida-exports/gms_jms_185.json` records the same `D1,D1,D4×6,D2,D4,D2,D2,D4…` shape. The IDB was unresponsive during this pass, so the per-bit mapping was not re-derived; jms behaviour is unchanged by this task either way (see §4). |

Registry shift assignments were re-verified against the v83 `CTS_*` globals
(read as four little-endian dwords, wire order `H>>32, H&0xFFFFFFFF, L>>32,
L&0xFFFFFFFF`): Speed=7, Stun=17, Poison=18, Seal=19, Darkness=20, Combo=21,
WeaponCharge=22, Weaken=30, Curse=31, Slow=32, Morph=33, Seduce=39, BanMap=46,
GhostMorph=49, Barrier=50, Confuse=51. All match the Go registry.

## 4. SLOW cannot render on a remote observer, on any supported client

`SecondaryStat::DecodeForRemote` **has no `CTS_Slow` branch**. Proven twice:

- v83 `xrefs_to CTS_Slow (0xbeffc0)` → `sub_77DC78`, `SecondaryStat::Reset`,
  `DecodeForLocal` (×2), `CheckByTime` (×2), the dynamic initializer. Not
  `DecodeForRemote`.
- v95 `xrefs_to CTS_Slow (0xc6c9a0)` → `IsMovementAffectingStat`, `CheckByTime`,
  `Reset`, `DecodeForLocal` (×2). Not `DecodeForRemote`.
- v87 and v92 by position: the global that sits between Weaken's and Morph's in
  the shift-ordered `DecodeForLocal` (v87 `CA2B70`, v92 `C37948` — the latter
  aligns exactly with v95's `CTS_Slow` at the same table offset) never appears
  in `DecodeForRemote`.
- v48/v61/v72/v79/v84 by exhaustive block enumeration of their remote decoders.

Consequently Slow's `nOption`/`rOption` (v83 offsets 1284/1296, setter
`sub_786F32`) stay zero on an observer's client, `UpdateAffectedSkillList`
block 7 never fires, and **no Slow animation is possible for a remote player**.
Atlas's `NoOpForeignValueWriter` for SLOW is therefore *correct* and is left
unchanged.

This is where the issue's proposed fix would have been wrong. Writing
`mobSkillId + level` unconditionally, as the referenced v83 server does, emits
4 bytes the v83 client does not read — the client would consume them as
`nDefenseAtt`/`nDefenseState` and then read the tDelay short out of the
mob-skill level. It happens not to crash only because the trailer is the same
total length; nothing renders either way.

**Net effect for the reporter's exact repro (SLOW from mob skill 126):** still
no remote indicator, because the client cannot show one. The diseases that
*can* render remotely — STUN, SEAL, DARKNESS, WEAKEN, CURSE, POISON, SEDUCE,
CONFUSE — now do.

## 5. What changed

All of it in `libs/atlas-packet/model/character_temporary_stat.go`; no service
code, no `go.mod`, no seed template, no opcode.

- New `MobSkillReasonForeignValueWriter`/`Reader` — `Short(mobSkillId) +
  Short(level)`, the client's 4-byte reason. Now the foreign shape for STUN,
  SEAL, DARKNESS, WEAKEN, CURSE, SEDUCE and CONFUSE.
- New `ValueMobSkillReasonForeignValueWriter`/`Reader` — the same, prefixed with
  the stat value. POISON only.
- `ValueSourceLevelForeignValueWriter`/`Reader` removed (POISON was its only
  user, and its field order was wrong).
- `foreignReadOrder` + `sortForeign` replace the shift sort in `EncodeForeign`
  and the shift-ordered walk in `DecodeForeign`.
- SLOW's `NoOpForeignValueWriter` kept, with the reason recorded at the call
  site so it does not look like an oversight.

Unchanged on purpose: the local `Encode`/`Decode` path (the local decoder *is*
shift-ordered — §3), the mask, the trailers, the two-state base blocks, and
`LevelSourceForeignValueWriter` (still used by ShadowPartner — §6).

Tests added to `character_temporary_stat_test.go`: byte-level pins for the STUN
and POISON foreign blocks, the SLOW mask-only pin, a SEAL+DARKNESS ordering pin
(the pair the shift sort inverted), an all-version multi-disease round trip, and
two guards — `TestForeignReadOrderCoversEveryValueCarryingStat` (anything that
writes foreign bytes must be named in `foreignReadOrder`) and
`TestForeignReadOrderNamesOnlyRealStats`. The coverage guard was confirmed to
fail when STUN is removed from the list, rather than assumed to be live.

Not verified: the two-client in-map observation from the issue's Verification
section. That needs two live clients and is the one check outstanding.

## 6. Deliberately out of scope

Pre-existing mismatches found while sweeping, none of them diseases, none
touched here:

- `ShadowPartner` (v87+) foreign-writes `Short(level)+Short(sourceId)`, which
  truncates a 7-digit player skill id. The client reads one `Decode4` reason.
- `BanMap` foreign-writes 4 bytes on gms_v48, whose remote decoder reads none
  for that bit.
- v95's remote decoder reads a `Decode4` reason for `Mechanic`, `DarkAura`,
  `BlueAura`, `YellowAura`; the registry has them `NoOp`. Atlas never
  originates these stats, so the bit is never set.
- gms_v61's remote decoder has no `ReverseInput` (Confuse) branch at all.

---

# Addendum — two party-buff defects found during live testing

Found while verifying the disease work on the `atlas-pr-1202` k3s environment
(tenant `0162160f-…`, GMS 83.1). Neither is a regression from the foreign-CTS
change above; both are pre-existing and were simply exposed by the same test
session. Reproduced from `atlas-channel` pod logs: Haste (`4101004`, cast by
character 2) produced `Character [2] applying effect` **and** `Character [1]
applying effect`; Sharp Eyes (`3221002`, cast by character 1, jobId 322 =
Marksman) produced only `Character [1] applying effect`, on all four casts.

## 7. Marksman Sharp Eyes never reached the party

`CUserLocal::SendSkillUseRequest` (gms_v83 `@0x96d399`) writes:

```
Encode4 updateTime · Encode4 skillId · Encode1 nSLV
if is_antirepeat_buff_skill(skillId): Encode2 castX, Encode2 castY
if skillId == 4121006:                Encode4 nSpiritJavelinItemID
if dwAffectedMemberBitmap:            Encode1 bitmap  (+ Encode2 when 2311001)
if nMobCount >= 0:                    Encode1 nMobCount, Encode4 × nMobCount
Encode2 delay                          (unconditional)
```

`is_antirepeat_buff_skill` lists `3121000`, `3121002` (Bowmaster Sharp Eyes)
and `3221000` — and stops. **`3221002` is in it on no supported client.** The
v83 disassembly `@0x96d6ca` is decisive: `sub 0x2F9F68` → 3121000, `dec/dec` →
3121002, `sub 0x1869E` (99,998) → 3221000, `sub 0xD6D84` → 4101004. Confirmed
independently per version:

| version | function | 3221002 present? |
|---|---|---|
| gms_v72 | `@0x877789` | no |
| gms_v79 | `@0x8c42bd` | no |
| gms_v83 | `@0x96d6ca` | no |
| gms_v84 | `@0x9ad4e4` | no |
| gms_v87 | `@0x9f20fc` | no |
| gms_v92 | `@0x919150` | no |
| gms_v95 | `@0x939dc0` | no |
| jms_v185 | `@0xa3e223` | no |

(gms_v48 and gms_v61 have no symbol for this function in their IDBs; not
re-derived — neither is affected by the fix either way, since the change only
removes an entry.)

`libs/atlas-packet/model/skill_usage_info.go` listed `MarksmanSharpEyesId` in
`isAntiRepeatBuffSkill`, so the decoder consumed 4 castX/castY bytes the client
never sent. The affected-party bitmap read then ran past the end of the 12-byte
packet and returned 0, and `selectPartyMembers`' `memberBitmap == 0` gate
resolved caster-only. Haste (`4101004`) *is* anti-repeat client-side, which is
why it aligned and worked — the asymmetry is a Nexon omission, not a pattern.
Same failure mode as task-155 / Time Leap.

`3221002` also had to come out of `isMobAffectingBuff`: gms_v83
`CUserLocal::DoActiveSkill` compares against 3221002 at `0x967ff7` and
dispatches to `loc_969275`, which pushes `dwTargetFlag = 2` — the party bit
only, never the mob bit (4) — so `DoActiveSkill_StatChange` `@0x969e21` passes
`nMobCount = -1` and no mob block is emitted. Left in, the decoder would have
read the trailing delay short's low byte as a mob count and manufactured
phantom target ids.

Fixed in `skill_usage_info.go`; pinned by
`TestDecodeMarksmanSharpEyesReadsPartyBitmap` (12-byte wire fixture, with the
non-zero trailing delay acting as the mob-block canary) and
`TestIsAntiRepeatExcludesMarksmanSharpEyesOnly`. Both were confirmed to fail
when the entry is reinstated.

### Still open in the same lists

Four more entries are in Atlas's `isAntiRepeatBuffSkill` but in no client's
`is_antirepeat_buff_skill`: `HermitShadowWeb 4111003`,
`CorsairSpeedInfusion 5221010`, `NightWalkerStage3ShadowWeb 14111001`,
`AranStage4ComboBarrier 21120007`. Each misaligns its own skill's decode the
same way. They are **not** fixed here: unlike 3221002, none of them appears as
an immediate in `DoActiveSkill`, so their `dwTargetFlag` is set by the
skill-type jump table and their `isMobAffectingBuff` membership could not be
derived. Removing them from one list without knowing the other would relocate
the misalignment rather than remove it. Fixing them needs a reversal of the
`DoActiveSkill` type dispatch.

## 8. Party-buff recipients got no SKILL_AFFECTED effect

`CUser::OnEffect` (gms_v83 `@0x9377d9`) case **2** reads `Decode4 skillId` +
`Decode1 nSLV` and calls `CUser::ShowSkillAffected` `@0x93632a` — the
buff-received animation drawn over the *affected* player. Case 1 (`SKILL_USE`)
draws the *caster's* cast animation and is not a substitute.

`libs/atlas-packet/character/effect_body.go` has defined
`CharacterSkillAffectedEffectBody` / `…ForeignBody` all along, but no service
ever called either — they were dead code. `character_skill_use.go:176-178`
announces only `SKILL_USE` (self + foreign) for the caster, so a party-buff
recipient received `CharacterBuffGive` and nothing else: no effect over the
player who received Haste, no animation on the recipient of Sharp Eyes.

Now emitted from the party-apply site: `applyToParty` returns the ids it
applied to, and `announceSkillAffected` writes `CharacterEffect`
(`SKILL_AFFECTED`) to each non-caster recipient's own session plus
`CharacterEffectForeign` to everyone else on their map. The caster is excluded —
their client renders the cast locally
(`DoActiveSkill_StatChange` → `CUser::ShowSkillEffect`) and already receives
`SKILL_USE`.

### Template coverage (pre-existing, unchanged here)

No template edit was needed: `SKILL_AFFECTED = 2` is already in the
`operations` table of every `CharacterEffect` / `CharacterEffectForeign` writer
that exists. But the writer set itself is uneven, and this predates and equally
affects the already-shipping `SKILL_USE` effect:

| template | CharacterEffect | CharacterEffectForeign |
|---|---|---|
| gms_61 / 72 / 79 / 83 / 84 / 87 | ✅ | ✅ |
| gms_95, jms_185 | ✅ (two entries) | ❌ — the second entry is also named `CharacterEffect`, so foreign effect writes find no writer |
| gms_12, gms_48, gms_92 | ❌ absent | ❌ absent |

v83 — the version under test — is fully wired. The other rows are a separate
config defect and were not touched: naming gms_95's `0xE0` / jms_185's `0xCC`
as the foreign writer is an inference from opcode ordering in the v61–v87
templates, not client-derived, and gms_12/48/92 need per-version opcode
evidence before entries are added.
