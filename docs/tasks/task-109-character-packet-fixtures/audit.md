# task-109 audit — KeyMapChange serverbound consistency (all 5 versions)

## Scope
The entire KeyMapChange serverbound packet (opcode 0x87 v83 / 0x8B v84 / 0x8F v87 /
0x9F v95 / 0x8A jms) across gms_v83, gms_v84, gms_v87, gms_v95, jms_v185 — the 10
matrix cells (5 `CHANGE_KEYMAP` op rows + 5 `None` sub-struct rows).

## Problem (re-confirmed, not trusted blindly)
KeyMapChange is ONE physical serverbound packet. The registry `CHANGE_KEYMAP` entry
carries a primary fname + 2 alts, all sub-handlers of the same dispatcher
(`SaveFuncKeyMap`, `ChangePetConsumeItemID`, `ChangePetConsumeMPItemID`).

`tools/packet-audit/cmd/run.go` `candidatesFromFName` maps **only** `SaveFuncKeyMap`
→ `{name: "KeyMapChange"}`; `ChangePetConsumeItemID`/`ChangePetConsumeMPItemID` →
`nil` (no report, "covered by KeyMapChange mode!=0"). So there is exactly ONE audit
report per version, named `KeyMapChange.json`, keyed by IDAName base `SaveFuncKeyMap`.

The op row resolves its report via `FNameToWriter[version][primaryFName]`
(`grade.go:findReport`). When the registry **primary** is `SaveFuncKeyMap`, the op
row finds the report and consumes the writer (`build.go usedWriters`) → no orphan
`None` sub-struct row is emitted. When the primary is `ChangePetConsumeItemID`, the
op row's `findReport` misses (that fname produces no report) → op row `incomplete`,
and the unconsumed `KeyMapChange` report becomes the verified `None` sub-struct row.

Pre-change state (re-confirmed from regenerated status.json):
| version | CHANGE_KEYMAP | None | primary |
|---|---|---|---|
| gms_v83 | verified | incomplete | SaveFuncKeyMap (prior task 4b) |
| gms_v84 | verified | incomplete | SaveFuncKeyMap |
| gms_v87 | incomplete | verified | ChangePetConsumeItemID |
| gms_v95 | incomplete | verified | ChangePetConsumeItemID |
| jms_v185 | incomplete | verified | ChangePetConsumeItemID |

(The v83/v84 `None` cells are union-row gap-fills — `build.go:178` `StateIncomplete`
"no audit report" — because v87/v95/jms emit the orphan sub-struct row.)

## Model chosen: option (b) — uniform primary = SaveFuncKeyMap in all 5 versions
WHY: The report is keyed on `SaveFuncKeyMap` and is the ONLY report `run.go`
produces for this dispatcher. Making `SaveFuncKeyMap` the registry primary in all 5
versions:
- resolves the `KeyMapChange` report on the `CHANGE_KEYMAP` op row in every version →
  all 5 op cells verified (tier-1 promote: marker.Found && hasEvidence &&
  evidence.Fresh — all present; verdict advisory per grade.go:198);
- consumes the report writer via the op row in every version → the orphan `None`
  sub-struct row is no longer emitted at all (it disappears, rather than leaving 5
  gap-filled incomplete cells).

This is the smallest, most uniform registry change (3 fname-primary swaps, the two
ChangePet* demoted to alts; v83/v84 already had this shape) and creates ZERO new
incomplete cells. Option (a) — verify both rows per version — is not cleanly
supported: there is no second report (run.go returns nil for the ChangePet* arms),
so a `None` sub-struct row would need a fabricated second report. Option (b) is the
tooling-native shape.

The two ChangePet* arms remain documented as `fname_alts` (the same dispatcher's
mode!=0 sub-handlers, covered by the codec's `mode != 0` branch).

## TRUNCATION exception — confirmed per version (IDA, task-109)
The `KeyMapChange.json` report is `FlatInvalid: true, verdicts [0,0,2,2,2,2]`
(v84: `[0,0,0,2,2,2]`) in every version — verdict 2 = width mismatch, the documented
benign loop-vs-flattened static-diff artifact (the export models one flattened entry;
the codec emits a variable-length loop). v83/v84 were confirmed by the prior task.
v87/v95/jms re-confirmed live this task:

- **v87** `CFuncKeyMappedMan::SaveFuncKeyMap @0x5bd3f4` (GMSv87_4GB.exe, port 13340):
  `COutPacket(0x8F=143)` + `Encode4(0)`=mode + `Encode4(count=*(v9-4))` + per-entry
  loop `Encode4(keyIdx)` + `sub_503876 @0x503876 = EncodeBuffer(Src, 5)`. Per-entry =
  4 + 5 = **9 bytes**.
- **v95** `CFuncKeyMappedMan::SaveFuncKeyMap @0x568a60` (GMS_v95.0_U_DEVM.exe, port
  13339): `COutPacket(159)` + `Encode4(0)`=mode + count + per-entry `Encode4(keyIdx)`
  + `FUNCKEY_MAPPED::Encode @0x4f6d80 = EncodeBuffer(this, 5)`. Per-entry = **9 bytes**.
- **jms** `CFuncKeyMappedMan::SaveFuncKeyMap @0x5e7b48` (MapleStory_dump_SCY.exe, port
  13338 — see caveat): `COutPacket(0x8A=138)` + `Encode4(0)`=mode + `Encode4(count)`
  + per-entry `Encode4(keyIdx)` + `FUNCKEY_MAPPED::Encode @0x510939 =
  EncodeBuffer(this, 5)`. Per-entry = **9 bytes**. The internal key-table scan loop is
  `< 94` (vs 89 GMS) — that bounds the scan over the key table, NOT the wire count
  (wire count is the dynamic changed-entry count `*(v9-4)`); it is exactly the
  documented benign loop-count artifact.

All three match the Atlas codec
`libs/atlas-packet/character/serverbound/key_map_change.go`: `mode int32` +
`count int32` + per-entry `[KeyId int32 + TheType int8 + Action int32]` (9 bytes).
**No per-version wire delta. Codec unchanged. Verification-only.**

### jms IDB caveat (surfaced, not blocking)
The plan prefers the jms `*_U_DEVM` build, but the only reachable jms instance is the
SMC retail dump (`MapleStory_dump_SCY.exe`, port 13338). `SaveFuncKeyMap @0x5e7b48`
decompiles cleanly there (not in an SMC-obfuscated region) and the wire structure is
unambiguous and identical to GMS, so the confirmation holds. The existing jms
evidence record (`category: TRUNCATION`) already documented the same artifact and was
not altered.

## Final state (from status.json)
All 10 KMC cells resolve to: the 5 `CHANGE_KEYMAP` op cells = `verified`; the `None`
sub-struct row no longer exists (consumed). No `n-a` used. Incomplete character cell
count: 40 → 35 (cleared exactly the 5 KMC incompletes; introduced none).
`matrix --check` EXIT 0, 0 conflicts, 0 character problem lines (unchanged from
baseline). `fname-doc --check` OK; `operations --check` OK (1 pre-existing unrelated
jms NoteOperation note). `go test -race`/`vet`/`build` green in libs/atlas-packet.

---

## Task 8b — jms Phase-A byte-fixtures (CharacterList, CharacterAppearanceUpdate, EffectQuest ×2)

Four jms_v185 Phase-A cells (#17, #22, #27, #32) verified against the jms IDB
`MapleStory_dump_SCY.exe` (port 13338). jms was the prime real-wire-delta suspect;
the full nested read orders were decompiled live and hand-computed bytes were
asserted. **Outcome: no wire delta — the Atlas codec already emits the jms-correct
wire for all four cells. Verification-only; no codec fix required.**

### Decompiled jms read orders (live)

- **CharacterList** — `CLogin::OnSelectWorldResult @0x66f3d8`, success path (v34==0||12):
  `Decode1` status /*0x66f411*/ → `DecodeStr` (jms leading empty string) /*0x66f72e*/
  → `Decode1` count /*0x66f73d*/ → per entry `GW_CharacterStat::Decode(_,_,0)`
  /*0x66f76c*/ + `AvatarLook::Decode` /*0x66f77a*/ + `Decode1` family /*0x66f78e*/ +
  `Decode1` rankEnabled (→ `DecodeBuffer(16)`) /*0x66f79b/7b6*/ → `Decode1` hasPic
  /*0x66f815*/ → `Decode1` m_bQuerySSN (jms extra) /*0x66f822*/ → `Decode4` slots
  /*0x66f832*/ → `Decode4` nBuyCharCount (jms unconditional) /*0x66f83f*/.
  - **`GW_CharacterStat::Decode @0x50ec17`**: HP/MaxHP/MP/MaxMP are 4×`Decode2`
    (**int16**, NOT v95-widened) /*0x50ed2d/41/55/69*/. jms tail after spawnPoint:
    `Decode2` + `DecodeBuffer(8)` + nPlaytime `Decode4` + `Decode4` + `Decode4`
    /*0x50ee65/7c/83/90/9d*/. Matches `character_statistics.go` JMS branch
    (WriteShort(0)+WriteLong(0)+3×WriteInt(0) = 2+8+4+4+4) byte-exact.
  - **`AvatarLook::Decode @0x51517e`**: gender/skin/face/!mega/hair, equip 0xFF loop,
    masked 0xFF loop, cashWeapon `Decode4`, pets `DecodeBuffer(12)` — matches codec.
- **CharacterAppearanceUpdate** — `CUserRemote::OnAvatarModified @0xa57221`: `Decode1`
  flags /*0xa57230*/ → (&1) AvatarLook → (&2/&4) optional Decode1 → crush/friend/
  marriage markers (Decode1 each) /*0xa572ca/5733b/573af*/; marriage if/else has NO
  trailing unconditional Decode4 (as v83/v84). Codec's trailing WriteInt(0) is benign
  slack. Matches.
- **EffectQuest** — `CUser::OnEffect @0x9f6395`, switch on `Decode1` /*0x9f63c0*/. The
  quest/item-gain body is **case 3** (block head @0x9f6981) — the GMS discriminator,
  NOT shifted to 5 like v95. Body: `Decode1` count /*0x9f698d*/, count==0 →
  `DecodeStr` /*0x9f6b1d*/ + `Decode4` nEffect /*0x9f6b4f*/, else loop `Decode4`
  itemId /*0x9f69a2*/ + `Decode4` amount /*0x9f69ac*/. Codec is mode-agnostic; fixture
  passes discriminator 3. Demux siblings EffectSimple (case 0, `Decode1` only) and
  EffectSkillUse (case 1, skillId `Decode4` /*0x9f6480*/ + charLvl `Decode1` /*0x9f648a*/
  + skillLvl `Decode1` /*0x9f64a7*/ + berserk trailing `Decode1` /*0x9f68b6*/) also
  fixtured (worst-of-three demux). All match the codec.

### Reports / verdicts

The regenerated jms reports carry `FlatInvalid: true` with verdict-2 rows
(CharacterList 0x66f3d8, AppearanceUpdate 0xa57221, Effect* 0x9f6395). These are the
**advisory** static-diff-vs-loop artifacts from the 8a-harvested export's `Unresolved`
/ `DecodeSub` sub-bodies (the export models flattened entries; the codec emits
variable-length loops). Per the playbook, tier-1 promotion = marker + byte-fixture +
fresh evidence, NOT a verdict-clean static diff. The hand-computed byte-fixtures are
the real verification and pass against the codec.

### Result
All four cells `verified` in status.json. `matrix --check` EXIT 0, 0 conflicts, no
character problem lines. `fname-doc --check` OK; `operations --check` OK (1
pre-existing unrelated jms NoteOperation note). `go test -race`/`vet`/`build` green.

---

## Task 8c — jms Class-A character cells (clientbound AddCharacterEntry, BuffGive,
## BuffGiveForeign, CharacterInfo, CharacterSpawn; serverbound CheckName,
## CreateCharacter, DeleteCharacter)

IDB: jms `MapleStory_dump_SCY.exe` @ port 13338 (only jms instance available;
the committed `gms_jms_185.json` export was harvested from it).

### REAL WIRE DELTA FOUND — CharacterSpawn jms (fixed-first)

`libs/atlas-packet/character/clientbound/spawn.go` emitted **two bytes the jms
client never reads**, proven against `CUserRemote::Init @0xa52876` (called by
`CUserPool::OnUserEnterField @0xa43ddd`) and the 8a-harvested jms export
`CUserRemote::Init` call list:

1. **`bShowAdminEffect` byte** — the codec wrote a byte after the foothold short
   and before the pet loop. The jms client reads `Decode2` foothold (call 18) then
   goes straight into the pet while-loop `while(Decode1())` (call 19) with **no
   admin byte**. (GMS `CUserRemote::Init` is not in any committed GMS export, so
   the GMS admin byte stays as-is and unverified-from-IDA; the codec keeps it for
   GMS, gated `Region()!="JMS"`.)
2. **trailing `team` byte** — the codec wrote berserk + a jms byte + team. The jms
   client's last two packet reads are call 46 (dragon/effect-1320006 flag) and call
   47 (final-effect flag) — only **two** trailing bytes. The codec's `team` byte is
   GMS-only (gated `Region()!="JMS"`).

Fix: gate both bytes off for jms in `CharacterSpawn.Encode` **and** `.Decode`
(kept symmetric so the round-trip test still holds). jms body 240→238 bytes; GMS
v83/v87/v95 byte output unchanged (233/237/196). Landed as its own commit
`fix(character): spawn.go jms wire — drop GMS-only bShowAdminEffect + team bytes`
before the CharacterSpawn verification commit. `go test -race ./...` green.

### No-delta cells (codec already jms-correct)

- **CheckName** — `CLogin::SendCheckDuplicateIDPacket @0x66e467`: COutPacket(8) +
  EncodeStr(name). Single ASCII string. Matches; report FlatInvalid false.
- **DeleteCharacter** — `CLogin::SendDeleteCharPacket @0x66e0f9`: COutPacket(0xD) +
  Encode4(selected char id). No PIC/DOB for jms. Matches; report FlatInvalid false.
- **CreateCharacter** — `CLogin::SendNewCharPacket @0x66e2ab` (non-charSale, op 0xB):
  EncodeStr(name)+Encode4(race/job)+Encode2(subJob)+6×Encode4(avatar templates).
  jms skips hairColor/skinColor/gender. Matches.
- **AddCharacterEntry** — `CLogin::OnCreateNewCharacterResult @0x66ffa8`: Decode1(code)
  + GW_CharacterStat::Decode @0x50ec17 + AvatarLook::Decode @0x51517e, then the
  list-entry rank trailer. jms GW_CharacterStat is 18 bytes wider than v83. Matches.
- **BuffGive / BuffGiveForeign** — `SecondaryStat::DecodeForLocal @0x7fcc73` /
  `DecodeForRemote @0x804dbf`: 16-byte UINT128 flag word (4×Decode4) then per-set-bit
  blocks + trailer. jms TwoState/base group occupies shifts 110-116 → first int
  0x001FC000 (jms-distinct from v83 0x0000FC01). EncodeMask emits the jms word. Matches.
- **CharacterInfo** — `CWvsContext::OnCharacterInfo @0xb0aa6e`: header + SetMultiPetInfo
  @0x9bb959 (bool-terminated pets) + mount + wishlist + SomethingMonsterBook @0x70522a
  (5 ints, jms-gated) + MedalAchievementInfo::Decode @0x9bcacf (medalId + short count)
  + trailing Decode4 count (jms-only, codec emits 0). Matches; trailing int is the jms
  4-byte delta over v83 (99 vs 95).

### Golden coverage added
Each cell got a jms golden-byte assertion (not a bare round-trip): TestCheckNameJMSGolden,
TestDeleteCharacterJMSGolden, TestCreateCharacterJMSGolden, TestAddCharacterEntryJMSGolden,
TestBuffGiveJMSMask, TestBuffGiveForeignJMSMask, TestCharacterInfoJMSGolden,
TestCharacterSpawnJMSGolden. Serverbound (CheckName/CreateCharacter/DeleteCharacter)
already routed in template_jms_185_1.json (CharacterCheckNameHandle / CreateCharacterHandle
/ DeleteCharacterHandle). All 8 cells `verified`; `matrix --check` EXIT 0, no character
problem lines.
