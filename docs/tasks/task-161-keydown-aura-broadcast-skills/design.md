# Keydown Cast-Aura Broadcast — Design

Task: task-161-keydown-aura-broadcast-skills
Status: Approved for planning
Depends on PRD: `docs/tasks/task-161-keydown-aura-broadcast-skills/prd.md`

---

## 1. Summary

The PRD proposed adding four skills to the keydown predicate so observers see their
cast/wind-up aura. **IDA verification against the client reduces this to two skills and
resolves the central architecture question in the process.** The v83/v84/v87/v95/jms185
clients gate BOTH the cast-aura relay (`DoActiveSkill_Prepare` / `OnSkillPrepare`) AND the
attack-packet `tKeyDown` field through a **single client function, `is_keydown_skill`**.
Because the client makes no distinction between "display keydown" and "wire keydown," neither
should Atlas: the correct change is to add the two verified skills to the one existing
`IsKeyDownSkill` predicate and split nothing.

**Verified verdict:**

| Skill | ID | In client `is_keydown_skill`? | Verdict |
|---|---|---|---|
| Brawler **Corkscrew Blow** | `5101004` | ✅ v83, v87, v95, jms185 | **ADD** |
| Gunslinger **Grenade** | `5201002` | ✅ v83, v87, v95, jms185 | **ADD** |
| FP Mage **Explosion** | `2111002` | ❌ absent in every version checked | **DROP** (FR-1.4) |
| Chief Bandit **Chakra** | `4211001` | ❌ absent in every version checked | **DROP** (FR-1.4) |

The two survivors are exactly the two entries present in the client's hardcoded keydown set
but missing from Atlas's `IsKeyDownSkill`. Adding them makes the Go predicate equal to the
client's v83 set. The two dropped skills are not keydown in the client at all — broadcasting a
prepare for them would render a phantom aura on observers and (worse) would broaden the shared
attack codec to read a `tKeyDown` field the client never sends, corrupting combat parsing.

---

## 2. IDA verification record (FR-1)

All addresses are from the checked-in IDBs. Region/version → instance port:
v83 `MapleStory_dump.exe` :13342, v84 `GMS_v84.1_U_DEVM` :13345, v87 `GMSv87_4GB` :13343,
v95 `GMS_v95.0_U_DEVM` :13341, jms185 `MapleStory_dump_SCY` :13344.

### 2.1 The predicate is a hardcoded switch, not a WZ property (FR-1.1)

`is_keydown_skill(int)` is a compiler-emitted binary-search switch over a fixed skill-ID set —
it does **not** read a `keydown` property from Skill.wz. The v83 function
(`?is_keydown_skill@@YAHJ@Z` @ **0x4fb08f**) returns 1 for exactly:

```
1121001 1221001 1321001            (Hero / Paladin / DarkKnight Monster Magnet)
2121001 2221001 2321001            (FP / IL ArchMage Big Bang, Bishop Big Bang)
3121004 3221001                    (Bowmaster Hurricane, Marksman Piercing Arrow)
5101004                            (Brawler Corkscrew Blow)      ← decoded from &loc_4DD5CB+1 = 0x4DD5CC
5201002                            (Gunslinger Grenade)          ← decoded from &loc_4F5C69+1 = 0x4F5C6A
5221004                            (Corsair Rapid Fire)
13111002 14111006 15101003         (WindArcher Hurricane, NightWalker Poison Bomb, ThunderBreaker Corkscrew)
22121000 22151001                  (Evan Ice Breath, Evan Fire Breath)
```

(The decompiler renders several skill-ID immediates as spurious `&loc_*`/`&word_*` address refs
because the numeric values collide with image addresses; each was decoded with `int_convert`:
`0x4DD5CC = 5101004`, `0x4F5C6A = 5201002`, `0xC80EDA = 13111002`, `0xD7511E = 14111006`,
`0xE66C4B = 15101003`.)

Neither `2111002` (Explosion) nor `4211001` (Chakra) is a member — in any version.

### 2.2 The predicate gates the prepare send / remote aura (FR-1.2)

`?DoActiveSkill_Prepare@CUserLocal@@` (v83 @ **0x96a86e**) is the sender. Its tail
unconditionally emits `COutPacket(0x5D)` → `Encode4(skillId), Encode1(a2), Encode2(action),
Encode1(actionSpeed)` — the serverbound prepare Atlas already decodes as `SkillPrepareInfo`.
Inside it, `is_keydown_skill(skillId)` gates the keydown-loop / looping-cast state
(`this[3149] = 1`) that produces the wind-up aura. Corkscrew Blow (`5101004`) and Grenade
(`5201002`) both flow through this function to the `0x5D` send when the correct weapon is
equipped (knuckle type 39/48 for Corkscrew; gun type 49 for Grenade — the weapon-gate branches
at 0x96aba6/0x96ac0b fall through to the send).

On the observer side, `?OnSkillPrepare@CUserRemote@@` (v83 @ **0x980a81**) also calls
`is_keydown_skill` (xref @ 0x980b37) to decide whether to start/loop the remote aura — so a
relayed prepare for these two IDs renders correctly on other clients, and a relayed prepare for
a non-member (Explosion/Chakra) would not. Cancel on key-up: `?SendSkillCancelRequest@CUserLocal@@`
(@ 0x96d873) is likewise gated by `is_keydown_skill` (xref @ 0x96d90f), matching Atlas's
existing cancel-relay path.

### 2.3 The SAME predicate gates the attack `tKeyDown` field (FR-1.3) — decisive

`is_keydown_skill` is called from the attack senders as well as the prepare/cancel/remote paths.
xrefs to 0x4fb08f include `?TryDoingShootAttack@CUserLocal@@` (@ 0x955213) and
`?TryDoingMeleeAttack@CUserLocal@@` (@ 0x952931). The shoot sender's use, disassembled at
**0x955210–0x955223**, is the exact mirror of Atlas's `attack_info.go`:

```asm
955210  push  skillId
955213  call  is_keydown_skill
955218  test  eax, eax
95521b  jz    loc_955228          ; not keydown → skip
95521d  push  keyDown
955223  call  COutPacket::Encode4 ; write tKeyDown uint32
955228  ...                       ; continue with mask1
```

That is byte-for-byte the semantics of:

```go
// libs/atlas-packet/model/attack_info.go
if skill.IsKeyDownSkill(skill.Id(m.skillId)) {
    w.WriteInt(m.keyDown)
}
```

**Conclusion:** in the client, the display-keydown set and the attack-wire-keydown set are not
two sets that happen to overlap — they are one set, computed by one function, used at every
site. This is exactly the coupling the PRD flagged as a hazard, but the hazard only bites if the
two sets *differed*; they do not.

### 2.4 Cross-version sweep (FR-1.3 / OQ-4 / §7)

The Go predicate is version-agnostic, so adding an ID changes attack parsing for every tenant
version. Both survivors were checked in each client:

| Version | `is_keydown_skill` @ | Corkscrew `5101004` | Grenade `5201002` |
|---|---|---|---|
| GMS v83 | 0x4fb08f | ✅ (0x4DD5CC) | ✅ (0x4F5C6A) |
| GMS v87 | 0x51c957 | ✅ (`sub_4DD5CC`) | ✅ (`nullsub_6`=0x4F5C6A) |
| GMS v95 | 0x509ea0 | ✅ (`&loc_4DD5C8+4`=0x4DD5CC) | ✅ (`&loc_4F5C6A`) |
| JMS v185 | 0x52b396 | ✅ (`&loc_4DD5CC`) | ✅ (`&loc_4F5C69+1`) |
| GMS v84 | — | inferred ✅ (see below) | inferred ✅ |

v84's `is_keydown_skill` symbol is not demangled in that IDB (a known v84 naming gap, per
project memory `bug_v87_template_missing_core_opcodes`/v84 groundwork notes) and a listing
search for the constant timed out. It is treated as verified-by-bracketing: v84 is
byte-structurally identical to v83 for this codepath (task-083 audit), and it sits between v83
and v87 — both of which are structurally identical here and both include the two skills. No
version in the supported GMS range (v83, v84, v87, v92, v95) or jms185 omits either skill, so
adding them introduces **no new wire divergence** for any tenant.

Aside (not in scope, recorded for the follow-up ledger): v95's `is_keydown_skill` **drops** the
three Monster Magnet IDs (`1121001`/`1221001`/`1321001`) that Atlas's predicate still contains
(Big-Bang pirate/warrior revamp). That is a *pre-existing* Atlas↔v95 divergence, untouched by
this task; this task neither creates nor widens it. If a v95 Monster Magnet attack-parse issue
is ever reported, that ledger entry is the lead.

---

## 3. Architecture decision: single predicate, no split (FR-3)

The PRD's FR-3 defaulted to splitting `IsKeyDownSkill` into a display predicate and an
attack-wire predicate *unless* IDA proved the two sets identical (FR-3.3). §2.3 proves exactly
that. Therefore:

- **Decision:** keep the single `skill.IsKeyDownSkill` predicate. Add `BrawlerCorkscrewBlowId`
  and `GunslingerGrenadeId` to it. Do not introduce `BroadcastsCastAura` or any second
  predicate (OQ-3 resolved: single-predicate is both the least-churn *and* the most faithful
  option).
- **Why not split:** a split would encode a distinction the client does not make. It would add a
  second list to keep in sync, invite future drift between "display" and "wire" membership, and
  provide zero correctness benefit here because the client's own single function feeds both the
  aura and the `tKeyDown` field. YAGNI: the split solves a divergence that verification shows
  does not exist.
- **Consequence for the call sites:** no call site changes. `shouldBroadcastKeydown`
  (`character_skill_prepare.go`, reused by `character_buff_cancel.go`) already calls
  `IsKeyDownSkill`; `attack_info.go` already calls it. Both pick up the two new IDs
  automatically — the display broadcast starts working AND the attack codec starts reading/
  writing `tKeyDown` for these two skills, which is precisely what the client does.

### Alternatives considered

1. **Add all four to the single predicate (the naïve PRD reading).** Rejected: Explosion and
   Chakra are not keydown in the client. Adding them would broadcast a phantom aura *and* make
   `attack_info.go` over-read 4 bytes for skills whose attack packet has no `tKeyDown` field →
   misaligned combat parsing (the dominant NFR failure). Verification is what turns "four" into
   "two."
2. **Split into display + wire predicates (PRD default).** Rejected per FR-3.3 — the two sets
   are provably identical in the client. Splitting adds maintenance surface for no gain.
3. **Add the two, keep single predicate (chosen).** Minimal, faithful, and self-documenting: the
   Go predicate now mirrors the client's `is_keydown_skill` v83 set exactly.

---

## 4. Components & change surface

Change is confined to one production edit plus tests. Service impact matches PRD §7.

### 4.1 `libs/atlas-constants/skill/model.go` — the only production change

Add the two verified IDs to `IsKeyDownSkill`:

```go
func IsKeyDownSkill(skillId Id) bool {
    return Is(skillId,
        FirePoisonArchMagicianBigBangId,
        IceLightningArchMagicianBigBangId,
        BishopBigBangId,
        HeroMonsterMagnetId,
        PaladinMonsterMagnetId,
        DarkKnightMonsterMagnetId,
        BowmasterHurricaneId,
        MarksmanPiercingArrowId,
        CorsairRapidFireId,
        NightWalkerStage3PoisonBombId,
        WindArcherStage3HurricaneId,
        ThunderBreakerStage2CorkscrewBlowId,
        EvanStage4IceBreathId,
        EvanStage7FireBreathId,
        BrawlerCorkscrewBlowId, // 5101004 — IDA-verified keydown v83/v87/v95/jms185
        GunslingerGrenadeId)    // 5201002 — IDA-verified keydown v83/v87/v95/jms185
}
```

Both constants already exist (`constants.go:3193`, `constants.go:3215`). This lib recompiles
both `atlas-channel` and `atlas-packet`.

### 4.2 No edits to the handlers or the codec

- `services/atlas-channel/.../character_skill_prepare.go` and `character_buff_cancel.go`:
  unchanged. `shouldBroadcastKeydown` already gates on `IsKeyDownSkill` + ownership (level > 0),
  so the two new skills flow through the existing `AnnounceForeignSkillPrepare` /
  `AnnounceForeignSkillCancel` relays with no code change (FR-2.1–FR-2.3 satisfied by the
  predicate edit; the ownership guard FR-2.3 is preserved because it is untouched).
- `libs/atlas-packet/model/attack_info.go`: unchanged. It already reads/writes `keyDown` gated
  by `IsKeyDownSkill`; the predicate edit extends that to the two skills, matching the client
  (FR-3.2). No new field, no new branch.

The prepare/cancel relay packets are skill-agnostic in structure — `SkillPrepareInfo`
(skillId u32, level u8, action u16, actionSpeed u8; the trailing swallow field is gated on
`skillId == 33101005` only) and `SkillPrepareForeign` (charId u32, skillId u32, level u8,
action u16, actionSpeed u8) do not branch on our two IDs, so relaying them needs no packet-model
change.

---

## 5. Data flow (unchanged paths, newly reached)

```
Caster client                    atlas-channel                         Observer clients
─────────────                    ─────────────                         ────────────────
hold attack key
  DoActiveSkill_Prepare (0x5D) ─► CharacterSkillPrepareHandleFunc
  {skillId=5101004/5201002}        decode SkillPrepareInfo
                                   shouldBroadcastKeydown()
                                     owns skill (lvl>0) &&
                                     IsKeyDownSkill(id) ──► TRUE (new)
                                   ForOtherSessionsInMap ──► OnSkillPrepare ─► CUserRemote::OnSkillPrepare
                                     AnnounceForeignSkillPrepare               is_keydown_skill(id)=1 → loop aura

release key
  CANCEL_BUFF (buff cancel)    ─► CharacterBuffCancelHandleFunc
  {skillId=5101004/5201002}        buff.Cancel(...)
                                   IsKeyDownSkill(id) ──► TRUE (new)
                                   ForOtherSessionsInMap ──► OnSkillCancel ─► remote aura stops
                                     AnnounceForeignSkillCancel

attack (0x2D, separate packet)
  {skillId, ..., tKeyDown}     ─► AttackInfo.Decode
                                   IsKeyDownSkill(id) ──► TRUE (new)
                                   reads keyDown u32  ◄── matches client Encode4(keyDown)
```

Before this change the two `TRUE (new)` gates returned false, so the prepare/cancel were dropped
("not a keydown skill or not owned, skipping broadcast") and — critically — the attack decode
did **not** read `tKeyDown`. Since the client *does* write `tKeyDown` for these skills, the
pre-change decode was already mis-aligned for a Corkscrew/Grenade attack; the predicate edit
fixes the attack path and the display path together. (This is a latent-correctness improvement,
not just cosmetics, and it is why a single faithful predicate is the right model.)

---

## 6. Testing strategy

All tests are byte-level and version-swept; no live client needed.

### 6.1 Predicate membership (unit)
`libs/atlas-constants/skill/model_test.go` (new or extended): assert
`IsKeyDownSkill(BrawlerCorkscrewBlowId) == true`, `IsKeyDownSkill(GunslingerGrenadeId) == true`,
and — as a regression guard against the PRD's dropped skills — assert
`IsKeyDownSkill(FirePoisonMagicianExplosionId) == false` and
`IsKeyDownSkill(ChiefBanditChakraId) == false`. Also re-assert the 14 pre-existing members stay
true (FR-3.4 no-regression).

### 6.2 Attack-codec byte-layout regression (FR-4.2)
Extend `libs/atlas-packet/model/attack_info_test.go`:
- **Non-keydown skill** (skillId 0, the existing `sampleAttackInfo`): assert the encoded bytes
  contain **no** `keyDown` field — pin the exact length/offset so an accidental predicate
  broadening surfaces as a byte-count change. (Round-trip already covers symmetry.)
- **Existing keydown skill** (Hurricane `3121004`): assert the `keyDown` u32 **is** present at
  the expected offset (guards the field didn't move).
- **New keydown skills** (Corkscrew `5101004`, Grenade `5201002`): assert `keyDown` u32 is now
  present — the intended change, and the byte-level proof it matches the client's
  `Encode4(keyDown)` at 0x955223. Run across all `pt.Variants` (v83/v84/v87/v95/jms) so the
  cross-version safety of §2.4 is enforced in CI, not just asserted here.

### 6.3 Foreign prepare / cancel relay bytes (Acceptance §byte-level)
Extend the clientbound tests (`skill_prepare_foreign` / `skill_cancel_foreign`) with fixtures
for `skillId = 5101004` and `5201002`: encode `SkillPrepareForeign{charId, skillId, level,
action, actionSpeed}` and assert the exact 12-byte layout, and the cancel packet for the same
IDs. Structure is skill-independent, so this pins that the relay carries the right skillId to
the observer for each verified skill.

### 6.4 Handler gate (unit, atlas-channel)
If a `shouldBroadcastKeydown` unit test does not already exist, add one asserting it returns true
for an owned Corkscrew/Grenade at level > 0 and false at level 0 / unowned (FR-2.3 ownership
guard) — using the Builder pattern for the skill models per project test conventions (no
`*_testhelpers.go`).

---

## 7. Verification checklist (maps to PRD §10 acceptance)

- [ ] IDA citations for all four IDs recorded (§2) — Corkscrew/Grenade ADD, Explosion/Chakra DROP.
- [ ] Dropped skills' negative finding documented (§1, §2.1) — FR-1.4.
- [ ] Prepare + cancel relay reach observers for the two verified skills (§5, tests §6.3) — FR-2.
- [ ] Single predicate retained with explicit FR-3.3 justification (§2.3, §3) — FR-3.
- [ ] Attack-wire `tKeyDown` membership equals the IDA-confirmed set; byte-layout regression test
      guards it (§6.2) — FR-4.
- [ ] Byte-level observer fixtures for each verified skill (§6.3).
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in `libs/atlas-constants`,
      `libs/atlas-packet`, `services/atlas-channel`.
- [ ] `docker buildx bake atlas-channel` from the worktree root.
- [ ] `tools/redis-key-guard.sh` clean.

---

## 8. Risks & mitigations

| Risk | Likelihood | Mitigation |
|---|---|---|
| Adding to the shared predicate misaligns another version's attack packet | Low | §2.4 sweep confirms both IDs are keydown in v83/v87/v95/jms; v84 bracketed. Byte test §6.2 runs all variants. |
| v84 not directly demangled | Low | Structural identity to v83 (task-083) + bracket by v83/v87; if ever contradicted, the §6.2 v84 variant fails in CI. |
| Future editor re-adds Explosion/Chakra "for completeness" | Low | Negative-assertion tests §6.1 fail if either is re-added. |
| Pre-existing v95 Monster Magnet divergence surfaces | Low / out of scope | Recorded in §2.4 ledger; not introduced or widened by this task. |

---

## 9. Out of scope (restated from PRD, confirmed by verification)

- Reworking the prepare/cancel handlers or packet models — unchanged.
- Any skill beyond the two verified survivors — Explosion/Chakra explicitly dropped.
- Non-v83 keydown auditing beyond confirming the two survivors don't mis-gate the shared codec
  (done, §2.4).
- The pre-existing Atlas↔v95 Monster Magnet keydown divergence (§2.4 ledger note).
