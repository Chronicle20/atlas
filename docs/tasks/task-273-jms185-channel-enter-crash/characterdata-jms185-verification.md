# CharacterData × jms_v185 — field-by-field verification

**Verdict: CharacterData is CLEAN on JMS 185 for the crashing character.** Every
byte Atlas emits on the JMS branch of `libs/atlas-packet/character/data.go` is
consumed by `CharacterData::Decode` @0x5137af in the order the client reads it,
including the nested `GW_CharacterStat::Decode`, `GW_ItemSlotBase`/`Equip`/
`Bundle` decoders, the ring records, the monster book and the JMS-only trailers.
The two unchecked neighbours named in the bug file are **also clean**:
`CClientOptMan::DecodeOpt` @0x4ae41d consumes exactly the two bytes `set_field.go`
writes when the leading short is 0, and `CWvsContext::OnSetLogoutGiftConfig`
@0xae81c0 reads exactly four `Decode4`s.

So the suspicion moves off the SetField payload and onto the **enter-field
burst** that follows it.

Two *latent* JMS divergences were found while sweeping. Neither is reachable by
character 40 (level 1, job 0, no skills, empty inventory), so neither explains
this crash — see "Latent divergences" below.

Evidence source: IDB session `a977912e`,
`E:\Programs\Nexon\IDBs_v9\JMS\v185\MapleStory_dump_SCY.exe.i64`. Every address
below is a decompile line from that IDB.

---

## 1. Call site and mode

`CStage::OnSetField` @0x7eea69 calls `CharacterData::Decode(v109, v3, 0)`
@0x7eebf4 — **`bBackwardUpdate = 0`**. That selects the plain-decode arm
everywhere the function branches on it, so the whole `if (bBackwardUpdate)`
block @0x5138f1–0x513a41 and the post-pass @0x51434a–0x5145af are skipped; they
contain no `CInPacket` reads either way.

Atlas writes `dbcharFlag = -1` (`data.go:133`), so **every** flag-gated section
below is taken by the client.

## 2. Read order vs Atlas, top level

| # | Client read (address) | Atlas write (file:line) | Verdict |
|---|---|---|---|
| 1 | `DecodeBuffer(p, 8)` @0x5137cf — dbcharFlag | `WriteInt64(-1)` `data.go:133` | ✅ |
| 2 | `Decode1` @0x5137d6 — SN-list flag (0 ⇒ the two `Decode4`-counted `DecodeBuffer(8)` loops @0x5137f3 / @0x513825 are skipped) | `WriteByte(0)` `data.go:143` | ✅ |
| 3 | `GW_CharacterStat::Decode` @0x513852 (flag&1) | `encodeStats` `data.go:146` | ✅ (§3) |
| 4 | `Decode1` @0x513863 — buddy capacity | `WriteByte(m.BuddyCapacity)` `data.go:147` | ✅ |
| 5 | `Decode1` @0x513869 — linked-name flag (0 ⇒ no `DecodeStr` @0x513878) | `WriteByte(0)` `data.go:153` | ✅ |
| 6 | `GW_CharacterStat::DecodeMoney` @0x5138a8 → one `Decode4` @0x50eeb6 | `WriteInt(m.Meso)` `data.go:155` | ✅ |
| 7 | `DecodeBuffer(..., 0xC)` @0x5138b8 — 12 opaque JMS bytes | `WriteInt(Id)` + `WriteInt(0)` ×2 `data.go:159-161` | ✅ |
| 8 | 5 × `Decode1` @0x513a86 (loop j=1..5 @0x513a47, flag&0x80) — inventory slot limits | 5 capacity bytes `data.go:444-448` | ✅ |
| 9 | `Decode4` @0x513b23 + `Decode4` @0x513b39 (flag&0x100000) — equip-slot-extension expiry | `WriteInt64(EquipSlotExtExpire)` `data.go:456` | ✅ |
| 10 | flag&4: four `Decode2`-terminated item lists — @0x513b7c equipped, @0x513c3c cash-equipped, @0x513cf5 equip inventory, @0x513db4 dragon/mechanic (accepted only for slot 1000..1003 @0x513ddb) | `data.go:460-495` — regular + `Short(0)`, cash + `Short(0)`, equip-inv + `Int(0)` (= the equip-inv terminator **plus** the empty 4th list's terminator) | ✅ |
| 11 | loop j=2..5 @0x513e07: per set bit, a `Decode1`-terminated list @0x513e60 — use / setup / etc / cash | four `WriteByte(0)`-terminated lists `data.go:497-531` | ✅ |
| 12 | `Decode2` skill count @0x513f46 (flag&0x100); per skill `Decode4` id @0x513f85, `Decode4` level @0x513f8d, `DecodeBuffer(8)` expiry @0x513fa7, and `Decode4` master level @0x513fcf when `is_skill_need_master_level` @0x513fbe | `encodeSkills` `data.go:653-673` | ✅ shape (see §6.1 for the predicate) |
| 13 | `Decode2` cooldown count @0x51400e (flag&0x8000); per entry `Decode4` @0x514027 + `Decode2` @0x514031 | `data.go:675-681` | ✅ |
| 14 | `Decode2` started-quest count @0x514065 (flag&0x200); per entry `Decode2` id @0x51407a + `DecodeStr` @0x514080 | `data.go:715-719` | ✅ |
| 15 | **second** `Decode2` count @0x5140b1; per entry `DecodeStr` @0x5140c3 + `DecodeStr` @0x5140d2 | `WriteShort(0)` (JMS-only) `data.go:721-723` | ✅ |
| 16 | `Decode2` completed-quest count @0x51411d (flag&0x4000); per entry `Decode2` @0x51413f + `DecodeBuffer(8)` @0x514146 | `data.go:725-731` | ✅ |
| 17 | `Decode2` mini-game-record count @0x514172 (flag&0x400); per record 5 × `Decode4` @0x5141ab…@0x5141df | `encodeMiniGame` → `WriteShort(0)` `data.go:759` | ✅ |
| 18 | `Decode2` couple @0x514227, `Decode2` friend @0x514255, `Decode2` marriage @0x514283 (flag&0x800); records are `DecodeBuffer` 0x21 @0x510ddb / 0x25 @0x510df9 / 0x30 @0x510d81 | `RingRecords.EncodeRecords` `model/ring.go:263-280` (JMS ⇒ all three counts) | ✅ |
| 19 | 5 × `Decode4` @0x5142dc then 10 × `Decode4` @0x5142f5 (flag&0x1000) — teleport rocks | `encodeTeleports` `data.go:778-795` | ✅ |
| 20 | `Decode2` count @0x51431c (flag&0x7C) of `sub_510F3A` @0x510f3a records (`Decode4`, `Decode4`, `Decode2`, `DecodeStr`) | `WriteShort(0)` (JMS-only) `data.go:170-172` | ✅ |
| 21 | `Decode4` @0x5145c9 (flag&0x20000) — monster-book cover | `encodeMonsterBookCover` `data.go:817` | ✅ |
| 22 | `sub_51121F` @0x5145f9 (flag&0x10000): `Decode1` mode @0x51123b; **mode 0** ⇒ `Decode2` count @0x5113be then per card `Decode2` @0x5113e6 + `Decode1` @0x5113f6 | `encodeMonsterBookCards` — `WriteByte(0)`, count, (short, byte) `data.go:822-829` | ✅ |
| 23 | `Decode2` QuestEx count @0x51467c (flag&0x40000); per entry `Decode2` @0x514695 + `DecodeStr` @0x51469b | `WriteShort(0)` (JMS arm) `data.go:194` | ✅ |
| 24 | `Decode2` count @0x5146e1 (flag&0x80000); per entry `Decode4` @0x5146fa + `Decode2` @0x514705 | `WriteShort(0)` `data.go:196` | ✅ |

No further `CInPacket` call follows @0x5146e1 — the function ends at
`sub_518068` @0x51471d. Atlas writes nothing more.

## 3. `GW_CharacterStat::Decode` @0x50ec17 (bBackwardUpdate = 0)

`Decode4` id @0x50ec3a · `DecodeBuffer(13)` name @0x50ec4b · `Decode1` gender
@0x50ec62 · `Decode1` skin @0x50ec77 · `Decode4` face @0x50ec8c · `Decode4` hair
@0x50eca1 · `DecodeBuffer(24)` three pet locker SNs @0x50ecac · `Decode1` level
@0x50ecbb · `Decode2` job @0x50ecc7 · **nine** `Decode2` @0x50ecdd, @0x50ecf1,
@0x50ed05, @0x50ed19, @0x50ed2d, @0x50ed41, @0x50ed55, @0x50ed69, @0x50ed7d
(STR, DEX, INT, LUK, HP, MaxHP, MP, MaxMP, AP) · `Decode2` SP @0x50edd2 ·
`Decode4` EXP @0x50edf7 · `Decode2` fame @0x50ee11 · `Decode4` gachaExp
@0x50ee2b · `Decode4` map id @0x50ee45 · `Decode1` portal @0x50ee5f · `Decode2`
@0x50ee6a · `DecodeBuffer(8)` @0x50ee7c · `Decode4` @0x50ee8a · `Decode4`
@0x50ee97 · `Decode4` @0x50eea3.

That is exactly `encodeStats` on the JMS branch (`data.go:275-357`), including
the JMS tail `WriteShort(0)` / `WriteLong(0)` / `WriteInt(0)` ×3 at
`data.go:350-356`. **Match.**

## 4. Items — `GW_ItemSlotBase::Decode` @0x50f611 and the three raw decoders

`Decode1` type @0x50f626 selects `CreateItem` @0x50f697: 1 = equip
(vtable `off_BE5AD0`), 2 = bundle (`off_BE5910`), 3 = pet (`off_BE5988`). Slot
+0x70 of each vtable is the raw decoder: equip 0x50feb9, bundle 0x5102ba, pet
0x510562.

`GW_ItemSlotBase::RawDecode` @0x50f813 (common prefix): `Decode4` itemId
@0x50f81d · `Decode1` cash flag @0x50f82d (`DecodeBuffer(8)` cash SN @0x50f83e
when set) · `DecodeBuffer(8)` dateExpire @0x50f855.

**Equip** @0x50feb9 then reads: `Decode1` @0x50fed3 (slots) · `Decode1`
@0x50fee7 (level) · **`Decode1` @0x50fefb — a JMS-only third byte** · fifteen
`Decode2` @0x50ff07…@0x51003d (STR/DEX/INT/LUK/MaxHP/MaxMP/PAD/MAD/PDD/MDD/
ACC/EVA/Craft/Speed/Jump) · `DecodeStr` owner @0x51005b · `Decode2` flag
@0x510079 · `Decode1` levelType @0x51009e · `Decode1` level @0x5100b8 ·
`Decode4` experience @0x5100c7 · `Decode4` hammers @0x5100e1 · **JMS-only**
`Decode1` @0x510106 + five `Decode2` @0x510115/@0x51012f/@0x510149/@0x510163/
@0x51017d + `Decode4` @0x510197 · then `DecodeBuffer(8)` @0x5101c2 **only when
the cash SN is zero** · `DecodeBuffer(8)` @0x5101e2 · `Decode4` @0x5101ef.

That is `Asset.encodeEquipableInfo` on the JMS branch verbatim
(`model/asset.go:234-317`), including the JMS extra byte at `asset.go:248`, the
JMS block at `asset.go:294-302`, and the trailing buffers at `asset.go:308-314`
(the client reads the first of those only when the cash SN is zero, @0x5101af —
Atlas writes it unconditionally from the non-cash `encodeEquipableInfo` arm,
which is the only arm that reaches it). `encodeEquipmentStats` (`asset.go:545-561`) writes exactly 15
shorts. **Match.**

**Bundle** @0x5102ba: `Decode2` quantity @0x5102cc · `DecodeStr` title @0x5102e4
· `Decode2` attribute @0x5102ff · `DecodeBuffer(8)` @0x510337 **only when
`itemId/10000 ∈ {207, 233}`** @0x51032d. Atlas gates the same 8 bytes on
`item.IsBullet || item.IsThrowingStar` (`asset.go:380`), and those
classifications are literally 233 and 207 (`libs/atlas-constants/item/constants.go:35,45`).
**Match.**

The equip-section slot is `Decode2` (@0x513b7c etc.) and the stackable-section
slot is `Decode1` (@0x513e60); Atlas uses `WriteShort` for JMS
(`asset.go:531`) and `WriteInt8` for stackables (`asset.go:371`). **Match.**

## 5. The two unchecked neighbours

- **`CClientOptMan::DecodeOpt` @0x4ae41d** — `Decode2` entry count @0x4ae430,
  then per entry `Decode4` @0x4ae44d + `Decode4` @0x4ae455. With the count 0 it
  consumes **exactly** the two bytes `set_field.go:50` writes. ✅
- **`CWvsContext::OnSetLogoutGiftConfig` @0xae81c0** — `Decode4` @0xae81cf plus
  a three-iteration loop `Decode4` @0xae81e5: **exactly four** `Decode4`s, which
  is what `set_field.go:89-93` writes. ✅

The rest of the frame also re-verifies clean against `CStage::OnSetField`
@0x7eea69: `Decode4` channel @0x7eeaa6, `Decode1` @0x7eeab6, `Decode4`
@0x7eeac1, `Decode1` sNotifierMessage @0x7eeae4, `Decode1` bCharacterData
@0x7eeaf1, `Decode2` nNotifierCheck @0x7eeb08, three seed `Decode4`s
@0x7eebac/@0x7eebb6/@0x7eebcb, `CharacterData::Decode` @0x7eebf4,
`OnSetLogoutGiftConfig` @0x7eebfc, `DecodeBuffer(p, 8)` @0x7eed25.

## 6. Latent divergences (NOT this crash)

Both are unreachable for character 40 (level 1, job 0, no skills). They are
recorded here as real, IDA-grounded defects and are **not fixed** in this unit:
both live in `libs/atlas-constants/skill` / `data.go` code paths shared by every
region, so a fix needs its own gating decision rather than a drive-by edit.

### 6.1 `skill.NeedsMasterLevel` has no JMS Dual-Blade branch

`is_skill_need_master_level` @0x47d2a8 has a branch Atlas does not model
(disassembly @0x47d2ee–0x47d325): when `jobId/10 == 43` the client returns true
if `get_job_level(jobId) == 4` (i.e. job 434 ⇒ every skill 4340000-4349999) or
the skill id is one of `4311003` (@0x47d307), `4321000` (@0x47d30f), `4331002`
(@0x47d317) or `4331005` (@0x47d31f).

`libs/atlas-constants/skill/model.go:115-130` falls through to `jobId%10 == 2`
for all of 430..434 and therefore returns **false** for every Dual Blade skill.
On JMS a Dual Blade would be short one `Decode4` per affected skill and every
later section would shift.

(The Evan arm is fine: the client returns true when `get_job_level` is 9 or 10
@0x47d2d3/@0x47d2e0, and `get_job_level` @0x47d347 yields 9/10 exactly for job
ids 2217/2218 — the pair Atlas hardcodes at `model.go:118`.)

### 6.2 Extended-SP arm in `GW_CharacterStat::Decode` is not gated for JMS

At @0x50eda2 the client tests `sub_5163A2(job)` — `job/1000 == 3 || job/100 ==
22 || job == 2001` (@0x5163a2). When true it skips the SP `Decode2` and instead
calls `sub_50E8B0` @0x50edc9, which reads `Decode1` count @0x50e8c6 then
`Decode1`+`Decode1` per entry @0x50e8de/@0x50e8e0.

`data.go:316` gates that same block on `t.IsRegion("GMS") && t.MajorAtLeast(84)
&& isEvanJob(...)`, so on JMS Atlas always writes a plain `WriteShort(Sp)`. A
JMS Evan (22xx), a JMS job 2001, **or any JMS job 3000-3999** would desync at
this field. Note the client's JMS predicate is wider than Atlas's `isEvanJob`
(`data.go:271`), which has no `job/1000 == 3` arm at all.

## 7. What changed in the repo

- `libs/atlas-packet/character/data_jms185_test.go` (new) —
  `TestCharacterDataByteOutputJMS185`, a byte-exact fixture for the crashing
  character's CharacterData with a decompile citation per field.
- `libs/atlas-packet/field/clientbound/set_field_test.go` —
  `TestSetFieldByteOutputJMS185`, a byte-exact fixture for the jms_v185 frame
  (the cell had a marker and evidence but no JMS byte fixture). It pins the
  `DecodeOpt` short and the four logout-gift ints from §5.
- `docs/packets/ida-exports/gms_jms_185.json` —
  `CNpcPool::OnNpcEnterField` no longer records `CNpc::Init` as one opaque
  `DecodeBuffer`; the seven reads are expanded with addresses, mirroring the
  descent already recorded on the sibling `CNpcPool::OnNpcChangeController`
  entry. Re-derived here from `CNpc::Init` @0x716da2: `Decode2` @0x716dd6,
  `Decode2` @0x716de4, `Decode1` @0x716e0c, `Decode2` @0x716e1c, `Decode2`
  @0x716e3c, `Decode2` @0x716e4a, `Decode1` @0x716edf.
- `docs/packets/audits/jms_v185/NpcSpawn.{json,md}` — regenerated from that
  export; all nine rows now match per field with no "absorbed by trailing opaque
  buffer" rows.
- `docs/packets/evidence/jms_v185/npc.clientbound.NpcSpawn.yaml` — re-pinned
  (hash drift from the export edit, per VERIFYING_A_PACKET §"Failure modes").
- `docs/packets/evidence/jms_v185/field.clientbound.FieldSetField.yaml` — gained
  a `verifies:` list naming both new fixtures.
- `docs/packets/audits/STATUS.md` / `status.json` — regenerated;
  `packet-audit matrix --check` exits 0.

No wire bytes changed for any version.

## 8. Where the investigation goes next

CharacterData, the SetField frame, `DecodeOpt`, the logout-gift block and the
NpcSpawn body are all now positively verified for JMS 185. The remaining
candidates for the null-`CNpcPool` crash are in the **enter-field burst**
(`SpawnForSelf` fan-out at 23:13:11.597), not in SetField:

1. A packet in the burst that the client routes to `CField::OnPacket` *before*
   `set_stage` @0x7eed4e finishes constructing the field and its pools.
2. Another op in the burst whose body desyncs the stream so that `CField` tear
   down happens between the stage swap and the `0x116` delivery.
3. The unhandled client → server ops `0xEA` / `0xDA` observed at 23:13:12.7 —
   still uncharacterised.

> **Tracked:** the two latent JMS divergences below are filed as
> [Chronicle20/atlas#1544](https://github.com/Chronicle20/atlas/issues/1544)
> and are deliberately NOT fixed on this branch.
