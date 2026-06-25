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

## Task 8d — final 4 jms character cells (Movement, Chair, HealOverTime, ViewAll)

IDB: `MapleStory_dump_SCY.exe` (port 13338), confirmed by name.

### REAL WIRE DELTA — HealOverTime jms (fix-first, commit 29f1af951)

`HealOverTime` (HEAL_OVER_TIME, opcode 0x54) is sent by
`CWvsContext::SendStatChangeRequestByItemOption @0xb054d6` on jms (the symbol name is
misleading; ground truth is `COutPacket::COutPacket(_, 0x54)`, and it is the only 0x54
sender — called from `CWvsContext::TryRecovery @0xae6f5a` auto-recovery). Its wire body is:

    Encode4(updateTime) + Encode4(val=0x1400) + Encode2(hp) + Encode2(mp)
    + Encode1(option) + Encode4(extra = dword_CDA4F8)   ← 17 bytes

The GMS v83/v87/v95 senders (`CWvsContext::SendStatChangeRequest`) stop after the option
byte (12 bytes). The Atlas codec previously gated the option byte to `GMS && <=95` only,
so for jms it neither read the option byte NOR the trailing validation dword — leaving 5
unconsumed bytes on decode (the live `CharacterHealOverTimeHandleFunc` decodes this packet).
**Fix:** added an `extra uint32` field + `Extra()` getter; encode/decode the option byte for
`(GMS<=95) || JMS` and the trailing dword for `JMS` only. Round-trip asserts the dword is
present on jms and absent on GMS. Also wired opcode `0x54 → CharacterHealOverTimeHandle` into
`template_jms_185_1.json` (the jms template had a gap at 0x54 between DistributeAp 0x52 and
DistributeSp 0x55), so the live jms channel routes the heal packet. Report verdicts
[0,0,0,0,0,0].

### No-delta cells (codec already jms-correct, verification-only)

- **CharacterMovement** — `CUserRemote::OnMove @0xa443ee` is a thunk to
  `CMovePath::OnMovePacket @0x70c5dc` → `CMovePath::Decode @0x70b3ce` (opaque move-path
  block); characterId read by the pool dispatcher prefix. Byte-identical structure to
  v83/v87/v95 — the ❌ was a `calls:null` exporter descent gap (same as the v84 case).
  Re-spliced the 2-call entry; verdicts [0,0]. Codec unchanged.
- **CharacterChairShow** — jms SHOW_CHAIR (opcode 0xCA) is read INLINE in
  `CUserPool::OnUserRemotePacket` case 0xCA @0xa44324: `*(RemoteUser+16516) = Decode4(chairId)`,
  characterId from the leading `Decode4 @0xa44250`. Same inline shape as v83 (case 0xC4) /
  v87 (case 0xD1). No separate `OnSetActivePortableChair` receive fn — spliced a synthetic
  export entry mirroring the verified twins. Verdicts [0,0]. Codec unchanged.
- **CharacterViewAllCharacters** — the 3 `CLogin::OnViewAllCharResult#CharacterViewAll{Characters,
  Count,SearchFailed}` `#suffix` keys were ABSENT on jms (only the base resolved). Decompiled
  `CLogin::OnViewAllCharResult @0x6709e4` (mode 0 NORMAL / mode 1 COUNT / modes 2-5 error) and
  spliced the 3 suffixes from the live read order. The jms `GW_CharacterStat::Decode @0x50ec17`
  genuinely differs from v83/v84 (nAP widened to int32, jms extendSP/posMap tail) — but the
  Atlas `CharacterListEntry` jms branch (byte-verified by the 8c CharacterList fixture) already
  emits it exactly; **no codec delta**. Reports FlatInvalid:false (Count/SearchFailed/Error
  all-zero; Characters carries the advisory static-diff-over-loop-expansion verdict-2 rows, same
  family as the verified jms CharacterList — the byte-level round-trip is the real verification).

All 4 cells `verified`; `matrix --check` EXIT 0, 0 conflicts, no character problem lines;
`fname-doc --check` OK; `operations --check` OK (the pre-existing `NoteOperation` writer-absent
note is unrelated). `git diff` of `gms_jms_185.json` shows only the intended keys (OnMove
modified; chair / heal sender / 3 ViewAll suffixes added).

---

## Backend-Guidelines Review (DOM-*/SUB-*/SEC-* adversarial audit)

- **Reviewer:** backend-guidelines-reviewer (adversarial; default FAIL until file:line proves PASS)
- **Date:** 2026-06-24
- **Scope:** the two IDA-justified production Go changes of the campaign —
  `libs/atlas-packet/character/clientbound/spawn.go` (commit `e4803b0fd`) and
  `libs/atlas-packet/character/serverbound/heal_over_time.go` (commit `29f1af951`),
  plus the jms route in `template_jms_185_1.json` (commit `8b7d37de0`) and the
  touched byte-fixture tests.
- **Note on checklist applicability:** the DOM-* checklist targets a *service* domain
  package (`model.go` + `builder.go` + `processor.go` + `resource.go` + administrator
  + JSON:API REST models). These changes are **wire codecs in a shared library**
  (`libs/atlas-packet`), not a service domain package. DOM-01..20, DOM-22..24, SUB-*,
  EXT-*, SCAFFOLD-* therefore do not apply (no model/builder/processor/REST/Dockerfile/
  Kafka surface is touched). The relevant bar is the package's own established codec
  idiom (immutable struct + private fields + getters + region/version-gated
  Encode/Decode symmetry) plus DOM-21 (no atlas-constants duplication) and SEC-*.

### Objective gate — Build & Test

- `go vet ./...` in `libs/atlas-packet`: **EXIT 0** (clean).
- `go test -race -count=1 ./character/...`: **PASS** (all 5 packages green; verified
  uncached for `TestCharacterSpawn*` and `TestHealOverTime*`).
- GMS byte-length regression check (instrumented run): GMS v83=233, v87=237, v95=196,
  v28=218, v84=233, v86=233 — **exactly the pre-change values** the commit claims
  (233/237/196); JMS=238 (was 240). The GMS wire is provably unchanged.

### Findings

#### spawn.go (`CharacterSpawn`, commit e4803b0fd) — verdict: PASS

- **Encode/Decode symmetry — PASS.** Both GMS-only bytes are gated with the identical
  predicate on each side: `bShowAdminEffect` Encode `spawn.go:106` / Decode
  `spawn.go:210` (`if t.Region() != "JMS"`); trailing `team` byte Encode `spawn.go:154`
  / Decode `spawn.go:250` (`if t.Region() != "JMS"`). The pre-existing jms final-effect
  byte stays symmetric (Encode `spawn.go:149-151`, Decode `spawn.go:247-249`). The
  round-trip test (`spawn_test.go:89-153`) exercises all 7 variants and passes, which
  only holds if Encode and Decode consume the same byte count per region.
- **No GMS regression — PASS.** The change converts two previously-unconditional
  `w.WriteByte(0)` calls into `if t.Region() != "JMS"` guards. For any GMS tenant the
  guard is true, so both bytes are still written; measured GMS lengths are identical to
  HEAD~1 (233/237/196). The mutation is strictly additive on the JMS arm.
- **Region idiom — PASS.** `t.Region() != "JMS"` / `== "JMS"` is the established gating
  idiom in this exact package — see `character/data.go:131,143,190,202,296,372`,
  `character/clientbound/list.go:47,66,81,101`, `clientbound/item_upgrade.go:98,119`.
  `IsRegion`/`MajorAtLeast` and raw `Region()==` coexist throughout; the chosen form
  is consistent with the surrounding file (the file already uses `t.Region() == "JMS"`
  at `spawn.go:79,149`).
- **Comments cite IDA evidence — PASS.** `spawn.go:102-105` and `spawn.go:152-153` name
  `CUserRemote::Init @0xa52876` and the specific call indices (foothold call 18 → pet
  call 19; final-effect call 47). No magic numbers without explanation.
- **Golden fixture is real (not length-only) — PASS.** `TestCharacterSpawnJMSGolden`
  (`spawn_test.go:52-80`) asserts the exact header+mask bytes (`got[:48]`) and the
  exact tail bytes (`got[165:]`) against hand-derived hex, plus the 238-byte length.
  The deterministic-prefix/deterministic-tail split is justified in the doc comment
  (`spawn_test.go:47-51`: the cts base-stat block carries a time interval). The wire
  delta lives entirely in the asserted tail, so this is not a false pass.

#### heal_over_time.go (`HealOverTime`, commit 29f1af951) — verdict: PASS

- **Immutable-struct idiom — PASS.** The new field is private (`extra uint32`,
  `heal_over_time.go:37`) with a getter `Extra() uint32` (`heal_over_time.go:61-63`),
  matching the existing `unknown`/`Unknown()` pattern (`heal_over_time.go:36,56-58`).
  No Builder is expected here: no packet codec in `serverbound/` or `clientbound/` uses
  a Builder — construction is via `New*()` + `Decode` populating private fields
  (e.g. `serverbound/attack_request.go:44`, `serverbound/skill_prepare.go:31`). The
  field is wired through Decode (`heal_over_time.go:101-103`) and Encode
  (`heal_over_time.go:84-86`) consistently.
- **Encode/Decode symmetry — PASS.** Option byte gated identically on both sides:
  Encode `heal_over_time.go:81` / Decode `heal_over_time.go:98`
  (`(t.Region() == "GMS" && t.MajorVersion() <= 95) || t.Region() == "JMS"`). Trailing
  dword gated identically: Encode `heal_over_time.go:84` / Decode `heal_over_time.go:101`
  (`t.Region() == "JMS"`). The round-trip test asserts the dword is preserved on JMS and
  is zero on GMS (`heal_over_time_test.go:38-44`) — a real differential assertion, not a
  bare round-trip.
- **No GMS regression — PASS.** Before this change the option byte was gated
  `t.Region() == "GMS" && t.MajorVersion() <= 95`; the new predicate only *adds* the
  `|| t.Region() == "JMS"` arm, leaving the GMS sub-expression byte-identical. The
  trailing dword is `JMS`-only, so it can never appear on a GMS wire. GMS v83/v87/v95
  round-trips still pass.
- **Comments cite IDA evidence — PASS.** `heal_over_time.go:15-30` documents the per-
  version wire body with concrete addresses (`@0xa1e997/.../0x9f2a00`, jms `@0xb054d6`),
  flags the misleading `SendStatChangeRequestByItemOption` symbol, and names the
  validation dword `dword_CDA4F8`. The 0x54 opcode is the stated ground truth.

#### template_jms_185_1.json route (commit 8b7d37de0) — verdict: PASS

- **Correct & non-colliding — PASS.** `0x54 → CharacterHealOverTimeHandle` with
  `LoggedInValidator`; `0x54` appears exactly once in the template (slotted between
  `0x52` DistributeAp and `0x55` DistributeSp). Handler constant is the one the codec
  exports (`heal_over_time.go:13`) and is registered in
  `services/atlas-channel/.../main.go:861` with a concrete handler func
  (`socket/handler/character_heal_over_time.go:14`). `LoggedInValidator` is registered
  (`main.go:907`) — so the handler will not be silently dropped (per the known
  missing-validator bug pattern).

### DOM-21 (no atlas-constants duplication) — PASS

No new domain type, alias, or numeric constant is introduced. `extra uint32` is a raw
wire field, not a reclassification of item/inventory/world/job ids; nothing in
`libs/atlas-constants/` is shadowed.

### SEC-* — N/A / PASS

No auth, token, redirect, secret, or `os.Getenv` surface is touched. The codecs read/
write fixed-width integers and pre-existing string fields; no untrusted-length
allocation or unbounded loop is added (the pet loop is pre-existing and bool-terminated).
No finding.

### Verdict on the two codec changes

**Both PASS — APPROVE.** `spawn.go` and `heal_over_time.go` are idiomatically consistent
with the package, Encode/Decode are symmetric on every gated field, the GMS wire is
provably unchanged (byte-length-verified against HEAD~1), the new `extra` field follows
the immutable private-field + getter convention, comments cite IDA addresses for every
gated byte, the golden/round-trip tests make real differential assertions (not length-
or round-trip-only false passes), `go vet` and `go test -race` are green, and the jms
route is correctly wired to a registered handler+validator with no opcode collision.

**No blocking findings. No non-blocking findings.**

#### Minor (optional, non-blocking)

- `HealOverTime.String()` (`heal_over_time.go:69-71`) was not updated to include the new
  `extra` field — its format string still ends at `unknown`. Cosmetic only (debug
  logging); does not affect wire output or correctness. Same pre-existing omission
  applies to `unknown`'s sibling fields, so this is consistent with the existing
  `String()` scope. Not required by any guideline.

---

## Plan-Adherence Review

**Reviewer:** plan-adherence audit (independent re-verification, read-only)
**Date:** 2026-06-24
**Branch:** `task-109-character-packet-fixtures` @ `631d53747`
**Base:** `5d9c42ff3` (main overlay baseline)

### Verdict: PLAN FAITHFULLY IMPLEMENTED — READY TO MERGE

All 47 in-scope `character/*` cells are genuinely `verified` with real artifacts;
acceptance gates are green; the two production codec changes are IDA-justified and
GMS-safe; export edits are surgical; no silent skips, stubs, or regressions found.
The plan's literal Stage-1/Stage-2 split dissolved per its own §C escalation rule
(execution-log.md), but the GOAL — every cell promoted via coupled, machine-checked
artifacts — is met. Two Minor documentation-vs-tooling mismatches (below) are
cosmetic, not coverage gaps.

### 1. All 47 cells verified — reconciled against baseline

`status.json` incomplete `character/*` cells = **0** (337 verified, 13 pre-existing
`n-a`, all 13 `n-a` unchanged from baseline). Baseline `5d9c42ff3` had **exactly 47**
incomplete character cells; **46 promoted directly** to verified, and the **5
KeyMapChange cells** (4 `CHANGE_KEYMAP` op rows v83/v87/v95/jms = plan #9/#10/#11/#47,
plus the v84 `op=None` sub-struct = plan #12) resolved via the documented Task-K
consolidation (commit `0486fe57f`): promoting `SaveFuncKeyMap` to the uniform registry
primary in v87/v95/jms makes the `CHANGE_KEYMAP` op row consume the single
`KeyMapChange` report in every version, and the orphan `op=None` sub-struct rows
**vanish** (consumed, not downgraded). Net: 5 KMC op cells verified, 5 None rows
removed. The diff's apparent "downgrades" (None v83/v87/v95/jms verified→gone, None
v84 incomplete→gone) are all this single tooling-native consolidation — **no genuine
coverage lost; no previously-verified character cell regressed to incomplete.**

Verified set == plan's 47 (cross-checked the promotion list against §C #1–#47): jms
Class-A (#1–8), GMS+jms KeyMapChange (#9–12,#47), Phase-A CharacterList/Appearance/
EffectQuest×2 across 5 versions (#13–32), Class-E Expression/Chair/CheckName (#33–39),
ViewAll/Movement/AutoDistributeAp/HealOverTime (#40–46). No out-of-scope packet
touched; `new keys in current = []`.

### 2. Promotion mechanism is real, not a false pass

- **Phase-A full-body golden bytes confirmed.** `list_test.go`,
  `appearance_update_test.go`, `effect_quest_test.go` assert **real per-field bytes**
  (`bytes.Equal(got, want)`) across all 5 versions, exercising the nested
  GW_CharacterStat + AvatarLook blocks (CharacterList) / effect mode body (EffectQuest,
  both op variants self/foreign × rewards/no-rewards). Each field carries a decompile
  line-address citation. Per-version structural deltas are explicitly encoded and
  asserted: v87 trailing `nSubJob` short, v95 widened Decode4 HP/MP + unconditional
  `nBuyCharCount`, jms int16 HP/MP + jms tail + leading empty string + querySSN byte.
  This is exactly plan §D.7; **not** a length-only/mode-only/round-trip-only assertion.
- **jms Class-A golden assertions present** (audit.md 8c): all 8
  (`TestCheckNameJMSGolden`, `TestCharacterSpawnJMSGolden`, `TestBuffGiveJMSMask`, etc.)
  exist in the test tree.
- **Grader enforces the promotion.** `grade.go:117` sets `tier1 = Tier1[pkt] ||
  rep.FlatInvalid`; `:197–203` makes the diff verdict **advisory** for tier-1 and
  promotes only on `marker.Found && hasEvidence && evidence.Fresh`. A fresh
  `matrix` regen produced **zero** diff to the committed `status.json`/`STATUS.md` —
  i.e. re-running the grader against the committed exports/markers/evidence reproduces
  all 47 as verified, proving each promoted cell has an on-disk fresh-hash evidence
  record + marker (spot-checked: `jms_v185/character.clientbound.CharacterList.yaml`
  carries `decompile_sha256` + `verifies: …#TestCharacterListByteOutputJMS`). 443
  `packet-audit:verify` character markers; 88–89 character evidence records per version.

### 3. The two production codec changes are IDA-justified and GMS-safe

- **`spawn.go` (`e4803b0fd`)** — gates `bShowAdminEffect` + trailing `team` bytes off
  for `Region()=="JMS"` only, **symmetric Encode/Decode**, citing
  `CUserRemote::Init @0xa52876`. GMS branches unchanged (the `!="JMS"` guard preserves
  prior unconditional behavior for GMS). jms body 240→238 bytes; GMS output unchanged.
- **`heal_over_time.go` (`29f1af951`)** — adds `extra uint32` + `Extra()`; the GMS
  guard `Region()=="GMS" && MajorVersion()<=95` is **unchanged in effect for GMS** (the
  added `|| Region()=="JMS"` only widens to jms), and the trailing dword is JMS-only.
  Cites `CWvsContext::SendStatChangeRequestByItemOption @0xb054d6` (opcode 0x54 ground
  truth). The new jms route `0x54 → CharacterHealOverTimeHandle` in
  `template_jms_185_1.json:340–342` carries **`"validator": "LoggedInValidator"`** —
  not validator-less (avoids the silently-dropped-handler bug).
- Both deltas are documented in audit.md (8c/8d) and execution-log.md with decompile
  evidence and the live-tenant config-patch caveat.

### 4. Acceptance gates pass

`(cd libs/atlas-packet && go test -race ./... && go vet ./... && go build ./...)` →
all **EXIT 0** (character pkgs: clientbound/serverbound/monsterbook all ok).
`go run ./tools/packet-audit matrix --check` → **EXIT 0, 0 conflicts, 0 character
lines** (matches baseline; NB: this gate must run in the workspace — `GOWORK=off`
breaks the tool's module resolution and yields a spurious EXIT 1).
`fname-doc --check` → EXIT 0 OK. `operations --check` → EXIT 0 OK (the 1
`NoteOperation` writer-absent note is pre-existing/unrelated). `tools/redis-key-guard.sh`
→ EXIT 0. Consumers build clean: `services/atlas-channel/atlas.com/channel` EXIT 0,
`services/atlas-login/atlas.com/login` EXIT 0. No `go.mod` touched → `docker buildx
bake` correctly not required.

### 5. Export hygiene — surgical, no drift

Function-key diff `5d9c42ff3..HEAD` (parsed JSON, not line-grep):
- `gms_v83.json`: +3 (`CUser::OnEmotion`, `CUserRemote::OnSetActivePortableChair`,
  `CLogin::SendCheckDuplicateIDPacket`), 0 changed, 0 removed — exactly the Class-E v83
  cluster (#33/#35/#38).
- `gms_v84.json`: +3 (same Class-E fns #34/#36/#39), 6 changed (ViewAll ×3 suffix #41,
  `OnMove` #42, `SendAbilityUpRequest#{DistributeAp,AutoDistributeAp}` #44/#45).
- `gms_jms_185.json`: +11 (6 shared-helper foundation splices + chair + ViewAll ×3 +
  HealOverTime sender), 4 changed (foundation DecodeSub-stub expansion of
  OnCreateNewCharacterResult/OnSelectWorldResult/OnMove/OnCharacterInfo).
All intended; **no ~150-key drift**.

### 6. No silent skips / stubs / deferrals

No `// TODO`, stub, or 501 introduced. The `TODO`s in `character/effect_body.go` and
`serverbound/create.go` are **pre-existing** (those files have zero commits in
`5d9c42ff3..HEAD`). No cell marked verified without marker+fresh-evidence (grader
enforces; idempotent regen confirms). The two out-of-scope observations in
execution-log.md (v95 seed missing the effect `operations` table; jms live-tenant
config patch for the new heal route) are legitimately deferred — both are
config/template-wiring follow-ups outside a byte-fixture verification campaign, and
neither weakens any promoted cell (byte-fixtures pass mode literally).

### Issues

**Critical:** none.
**Important:** none.
**Minor (cosmetic, documentation-vs-tooling):**
1. Plan §C/§D called for "its own evidence record per version (op-keyed)" for the two
   `EffectQuest` ops (SHOW_FOREIGN_EFFECT / SHOW_ITEM_GAIN_INCHAT). Execution used **one**
   shared `EffectQuest` report + evidence + fixture covering both op variants. This is
   correct, not a gap: both CSV ops share fname `CUser::OnEffect` → one `EffectQuest`
   packet id, and the grader keys evidence/markers by `(packet-id, version)` not by op
   (`grade.go:115–116`); the single fixture asserts both op bodies. The plan's "op-keyed"
   wording was over-prescriptive for a shared-fname packet. (By contrast AutoDistributeAp's
   two ops *do* have distinct evidence — `#DistributeAp` / `#AutoDistributeAp` — because
   they resolve to distinct `#suffix` fnames; that op-keying is correct.)
2. Plan §C "Verdict-clean rule" stated ✅ requires `FlatInvalid:false` AND all verdicts 0,
   with KeyMapChange TRUNCATION as the *one* exception. In practice several Phase-A/jms
   reports are `FlatInvalid:true` with advisory verdict-2 rows (loop-vs-flattened
   static-diff artifacts), promoted on the byte-fixture. This is tooling-native
   (`grade.go:117/197` make the verdict advisory whenever `FlatInvalid`), and audit.md
   documents it per-cell, but it is a broader application of the "advisory verdict"
   path than the plan's prose anticipated. No false pass results — every such cell is
   backed by a real full-body golden fixture (verified in §2 above).

### Action items
None blocking. Optionally, for documentation accuracy, note in the PR description that
the EffectQuest two ops share one fixture/evidence record (tooling keys by packet id),
and that FlatInvalid-advisory tier-1 promotion (byte-fixture is the verifier) was the
operative mechanism for the Phase-A packets — broader than plan §C's verdict-clean prose.
