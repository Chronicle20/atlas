# Chakra (Chief Bandit 4211001) — Design

Task: task-213-chakra-hp-restore
Status: Approved for planning
Created: 2026-08-12
Inputs: [`prd.md`](prd.md)

---

## 0. Executive summary

The design phase's mandatory derivations (PRD §9 OQ-1…OQ-9) are complete. Three of
them overturn premises the PRD carried forward from community sources, and one of
them (OQ-1) has a definitive *negative* answer that PRD FR-7 did not anticipate:

| # | PRD premise | Derived result |
|---|---|---|
| OQ-1 | Heal formula is derivable from the client (FR-7.1, "explicit instruction") | **False.** The client never computes Chakra's HP restore on any of the ten IDBs. It sends a prepare packet, then a plain `USE_SKILL` at animation end, and takes whatever HP the server reports. The base-recovery term does not exist in any available artifact. §3.4 |
| OQ-2 | Keydown status is disputed between two in-repo records | **Not keydown.** `is_keydown_skill` excludes 4211001 on every version. `model_test.go` is right; research-doc P7 is wrong. FR-10.3 applies. §3.5 |
| OQ-3 | Recovery window "~1500 ms", unverified | **1500 ms exactly**, and now derived: the WZ `prepare` node is 15 frames × 100 ms delay, identical at v48 and v83. §4.3 |
| OQ-4 | Amplifier ordering vs task-157's roster is unknown | **First.** The client rewrites the raw damage before Magic Guard, Achilles, Combo Barrier, Magic Shield, Meso Guard and Power Guard all read it. §3.3 |
| OQ-5 | Client may track a Chakra CTS/flag | **No CTS.** State is `CUserLocal::m_nPrepareSkillID`, purely local. No packet model change. §3.6 |
| OQ-6 | Which WZ fields carry recovery / damage-taken | **`y` = recovery rate %, `x` = damage-taken %.** Both already parsed by atlas-data. No atlas-data change. §4 |
| FR-6.2 | The provided reference table is the pre-Big-Bang table | **It is the v12/v48 table only.** v61 through v92 and JMS 185 ship a completely different, *inverted* table; v95 ships a `common` formula with maxLevel 10. §4.1 |
| FR-4.1 | "incoming damage MUST be multiplied … amplified" | True only on v12/v48 (`x` 200→112). On v61+ `x` is 99→70 — Chakra **reduces** incoming damage. One formula covers both; the WZ supplies the direction. §4.2 |
| OQ-7 | Where the activation gate sits vs the cost block | An existing precedent already solves it: the Hero Enrage gate in `character_skill_use.go` rejects *before* `handler.UseSkill`. §5.2 |
| OQ-8 | What counts as "movement" for interruption | The client makes the caster **immovable** for the whole window, so authentic clients never move. Server-side interruption is a defence against crafted clients, not a simulation of client behaviour. §3.7 |
| OQ-9 | 50 % boundary | Client computes `HP*100/MaxHP >= 50 → reject`, i.e. exactly `2*HP >= MaxHP`. Integer, no float. §3.2 |

The net effect on scope: **no packet/opcode work, no `libs/atlas-packet` change, no
`libs/atlas-constants` change, no atlas-data change.** The change is confined to
`atlas-channel` plus two seed templates that are missing an already-existing handler
binding (§6.3).

---

## 1. Evidence base

Ten IDBs were examined via `ida-pro-mcp`, resolved by binary name per CLAUDE.md.
GMS v12 has **no IDB** (the only artifact for that version is `Data.wz`), so every
v12 statement in this document is WZ-derived, never IDA-derived — recorded per FR-9.4.

Chakra appears in exactly **three** client functions on every version examined
(v92 shows a fourth, an inlined duplicate inside `SetDamaged`). The set never varies:

| Version | IDB | `SetDamaged` (damage multiplier) | `DoActiveSkill_Prepare` (HP gate) | `TryDoingPreparedSkill` (completion) | `DoActiveSkill` (entry) |
|---|---|---|---|---|---|
| GMS 12 | — (no IDB) | n/a | n/a | n/a | n/a |
| GMS 48 | `GMS_v48_1_DEVM.exe` | `CUserLocal__SetDamaged_MobHit` @ `0x6A6159`, `0x6A6183` | `0x6ADD4C` gate @ `0x6AE086` | `0x6A6584` | — |
| GMS 61 | `GMS_v61.1_U_DEVM.exe` | `0x7AA748` @ `0x7AB06D`, `0x7AB500` | `0x7B8001` gate @ `0x7B84B7` | `sub_7BA074` @ `0x7BA1B0` | `sub_7B5977` @ `0x7B6A61` |
| GMS 72 | `GMS_v72.1_U_DEVM.exe` | `0x864B92` @ `0x865553`, `0x8659DF` | `0x874C8F` gate @ `0x87527B` | `sub_87706C` @ `0x8771A8` | `sub_871D35` @ `0x872C70` |
| GMS 79 | `GMS_v79_1_DEVM.exe` | `0x8B0277` @ `0x8B0D94`, `0x8B1313` | `0x8C17F2` gate @ `0x8C1DE4` | `sub_8C3B94` @ `0x8C3CD3` | `sub_8BE4FE` @ `0x8BF53B` |
| GMS 83 | `MapleStory_dump.exe` | `0x9581A9` @ `0x958CCD`, `0x95924F` | `0x96A86E` gate @ `0x96AE74` | `0x96CF1D` @ `0x96D05C` | `0x966F7A` @ `0x968075` |
| GMS 84 | `GMS_v84.1_U_DEVM` | `0x99634D` @ `0x996E6F`, `0x9973F5` | `sub_9A9761` gate @ `0x9A9D76` | `sub_9ACCA9` @ `0x9ACE0C` | `sub_9A6142` @ `0x9A7347` |
| GMS 87 | `GMSv87_4GB.exe` | `0x9DA6F2` @ `0x9DB26E`, `0x9DB7F5` | `0x9EE1E6` gate @ `0x9EE801` | `sub_9F18BB` @ `0x9F1A1E` | `0x9EA7B9` @ `0x9EBB0E` |
| GMS 92 | `GMS_v92_1_DEVM.exe` | `0x913BB0` @ `0x914A5D`, `0x914A89`, `0x9152AF` | `0x91DF00` gate @ `0x91E3C8` | `sub_9217E0` @ `0x921A51` | `sub_921B10` @ `0x9235E0` |
| GMS 95 | `GMS_v95.0_U_DEVM.exe` | `0x9343C0` @ `0x935397`, `0x935C67` | `0x941710` gate @ `0x941EE3` | `0x944270` @ `0x9444E1` | — |
| JMS 185 | `MapleStory_dump_SCY.exe` | `0xA228F8` @ `0xA236CD`, `0xA23D7E` | `0xA39CFD` gate @ `0xA3A3EE` | `sub_A3D8AE` @ `0xA3DB22` | `0xA35C3F` @ `0xA36F21` |

GMS v95's IDB is PDB-backed and names the relevant symbols outright — `s_pChakra`,
`SKILLLEVELDATA::_ZtlSecureGet_nX`, `GW_CharacterStat::_ZtlSecureGet_nHP`,
`BasicStat::_ZtlSecureGet_nMHP`. Every legacy version's decompilation is
structurally identical to v95's at each of the three sites, which is what lets the
unnamed legacy struct offsets (`+0x130` on v48, `+0x13C` on v83, `+0x154` on JMS 185)
be identified as the same field, `nX`.

WZ evidence was read directly from local `Skill.wz` / `String.wz` / `Data.wz` copies
using a throwaway program built against `libs/atlas-wz` (scratchpad only, not
committed). Concrete paths are environment-specific and deliberately not recorded here.

---

## 2. Client control flow (what actually happens)

```
key press
  └─ CUserLocal::DoActiveSkill
       └─ CUserLocal::DoActiveSkill_Prepare
            ├─ [4211001 only] if HP*100/MaxHP >= 50  → return 0   ← nothing is sent
            ├─ m_nPrepareSkillID = 4211001
            ├─ m_tPrepareSkillEnd = now + prepare-animation delay (1500 ms)
            ├─ CUser::ShowSkillPrepare(...)
            └─ send  CP_UserSkillPrepareRequest {skillId, level, action, speed}
                                                    ↑ atlas: CharacterSkillPrepareHandle

   (during the window: CUserLocal::IsImmovable() == true — the caster cannot walk,
    jump, climb or rope. CUserLocal::SetDamaged rewrites incoming damage by nX/100
    before every other mitigation term reads it.)

CUserLocal::Update (per frame)
  └─ CUserLocal::TryDoingPreparedSkill
       └─ once now >= m_tPrepareSkillEnd:
            memset(m_nPrepareSkillID..+12, 0)          ← state cleared
            └─ [4211001] CUserLocal::DoActiveSkill_StatChange
                 └─ CUserLocal::SendSkillUseRequest    ← ordinary USE_SKILL
                                                          ↑ atlas: CharacterSkillUseHandle
```

Three consequences fall directly out of this shape and drive the whole design:

1. **The server hears about Chakra twice**, and the two packets already have handlers.
   `CharacterSkillPrepareHandle` and `CharacterSkillUseHandle` both exist and are
   already wired; nothing new needs decoding.
2. **The heal belongs on `USE_SKILL`**, i.e. in the ordinary per-skill `Handler`
   registry, at completion — which satisfies FR-3.4 for free.
3. **MP is charged at completion, not at keypress**, because `UseSkill`'s cost block
   only runs when the `USE_SKILL` packet arrives. An interrupted cast therefore costs
   nothing rather than costing MP that is not refunded. FR-5.4 ("MUST NOT refund MP")
   is satisfied vacuously; FR-8.3 (exactly once) is satisfied structurally.

---

## 3. Derivations

### 3.1 The damage multiplier

v95, symbol-named (`0x935397`–`0x93542B`):

```c
if (this->m_nPrepareSkillID == 4211001) {
    nSLV = CSkillInfo::GetSkillLevel(characterData, 4211001, &s_pChakra);
    if (nSLV && s_pChakra) {
        nDamage = s_pChakra->GetLevelData(nSLV)->_ZtlSecureGet_nX() * nDamage / 100;
        if (nDamage <= 1) nDamage = 1;
    }
}
```

v83 (`0x958CCD`–`0x958D4D`) and v48 (`0x6A6159`–`0x6A61E0`) are instruction-for-instruction
the same shape with an unnamed `SKILLLEVELDATA` offset in place of the accessor call.
JMS 185 (`0xA236CD`–`0xA2374D`) likewise. Notes that matter for the port:

- The multiplier is **not gated on the attack source.** There is no `attackIdx`,
  mob-sourced, or magic/physical test around the branch.
- The **`<= 1 → 1` floor** applies to the multiplied value, not the original.
- It is re-derived from the caster's *current* skill level on every hit; a caster who
  somehow lost the skill mid-window gets no multiplier.
- It uses `nX` — the same WZ `x` the tooltip calls "Damage during the recovery if hit".

### 3.2 The activation gate

v95, symbol-named (`0x941EE3`–`0x941F0C`):

```c
if (nSkillID == 4211001
    && GW_CharacterStat::_ZtlSecureGet_nHP() * 100 / BasicStat::_ZtlSecureGet_nMHP() >= 50)
    return 0;
```

`GW_CharacterStat::nHP` is the character record's current HP; `BasicStat::nMHP` is the
**derived** max HP (gear + buffs), not the base stat. That settles FR-1.2 in favour of
`atlas-effective-stats`, and the existing `effectiveMaxHpOrBase` narrowing in
`skill/handler/heal/heal.go:29-38` is the right defensive fallback.

`floor(100*HP/MHP) >= 50` is exactly `100*HP >= 50*MHP`, i.e. **`2*HP >= MaxHP`**. Use
that form — it is integer, needs no float, and OQ-9's boundary (HP exactly 50 %) falls
out correctly: reject.

JMS 185 (`0xA3A3EE`) expresses the same test with inverted branch polarity
(`jge → xor eax,eax; return 0`); the semantics are identical. v48 (`0x6AE086`) matches
GMS v83/v95 exactly.

The gate is **the only thing preventing a cast**; there is no post-gate HP re-check
anywhere in the client, corroborating FR-1.3 (threshold checked at activation only).

### 3.3 Ordering vs task-157's mitigation roster

In `CUserLocal::SetDamaged` the Chakra branch writes back to the **same stack slot**
that carries the damage (`mov [ebp+arg_0], eax` at v83 `0x958D48`), and every task-157
term reads that slot afterwards in straight-line flow:

| v83 address | Step |
|---|---|
| `0x958CCD`–`0x958D4D` | **Chakra: `damage = damage * x / 100`, floor 1** |
| `0x959327` | Magic Guard MP absorb (`sub_95CDC3`) |
| `0x959365` | Achilles / Combo Barrier permille reduction (`* (1000-n) / 1000`) |
| later | Magic Shield, Meso Guard, Power Guard, Mana Reflection |

So the Chakra factor is applied to **raw damage, before every existing mitigation term**.
An implementation that applied it after Achilles would produce different numbers, which
is precisely what FR-4.3 warned about.

### 3.4 The heal formula — negative result (OQ-1)

`TryDoingPreparedSkill` dispatches Chakra to `CUserLocal::DoActiveSkill_StatChange`
(v83 `0x969E21`). That function is the generic stat-change caster: it checks
ladder/rope, map flags, weapon type and the field's skill ban, resolves party/mob
targets, and then calls `CUserLocal::SendSkillUseRequest` followed by
`CUser::ShowSkillEffect`. **It computes no HP.** There is no `nHP`, `nY`, `nMHP` or
LUK read anywhere on Chakra's path, on any of the ten IDBs.

That is a definitive answer, not a search failure: the three Chakra sites enumerated in
§1 are exhaustive (a whole-binary immediate scan for `4211001`), and none of them
touches HP. In authentic MapleStory the server owns this number and reports it back via
a stat-change packet; the value is not present in any client-side artifact, nor in WZ
(which carries only `y`, described as "Recovery rate `y`%", with no base term stated —
see the v95 tooltip template in §4.1).

FR-7.1's instruction ("MUST be derived from the client binary") therefore has no
satisfiable answer, and FR-7.5's escalation clause is what applies. **Decision (owner
approved):** ship a deterministic LUK-based term, explicitly recorded as
community-sourced and unverified, rather than block the feature:

```
healAmount = ChakraRecovery(base, y)
base       = LUK * 29 / 10          // 2.9 x effective LUK, integer
ChakraRecovery(base, y) = base * y / 100
```

2.9 is the midpoint of the 2.3–3.5 range used by the odinms-derived server lineage
(Cosmic `StatEffect.makeHealHP`). Taking the midpoint deterministically rather than
rolling within the range honours FR-7.6 — randomisation stays out absent evidence for
it. `base` and `y` remain separate arguments per FR-7.4, so if a better-grounded base
term ever surfaces, only one function changes and its unit tests re-pin.

**This must be labelled in code.** The `base` term carries a comment stating that it is
community-sourced and that IDA on all ten IDBs proved the client does not compute the
heal. No Cosmic file/line citation goes in the comment (project rule: no Cosmic
citations in code comments) — the citation belongs here, in the design doc.

LUK comes from `atlas-effective-stats` (`RestModel.Luck`), which the channel already
consumes; the caster's base LUK is the fallback when the upstream errors, mirroring
`heal.go`'s treatment of Intelligence.

### 3.5 Keydown verdict (OQ-2, FR-10)

v83 `is_keydown_skill` (`0x4FB08F`) — full membership:

```
1121001, 1221001, 1321001   Monster Magnet x3
2121001, 2221001, 2321001   Big Bang x3
3121004, 3221001            Hurricane, Piercing Arrow
5101004, 5201002            Corkscrew Blow, Grenade
13111002, 14111006, 15101003  WB Hurricane, NW Poison Bomb, TB Corkscrew
22121000, 22151001          Evan breaths
```

**4211001 is absent.** v95's `is_keydown_skill` (`0x509EA0`) is the same set. This
independently reproduces task-161's finding, so:

- FR-10.2 does **not** apply. `libs/atlas-constants/skill/model.go` and
  `model_test.go:33-37` stay untouched; the `attack_info.go` over-read the pinning
  comment warns about never becomes a risk.
- FR-10.3 applies: research-doc P7 (`docs/research/missing-features/skills-and-buffs.md`,
  line 103) is wrong about Chakra and must be corrected so it stops re-seeding tasks.
  **Note for the plan phase:** that file is *not tracked in git* and does not exist in
  this worktree — it lives untracked in the main working tree. The correction is
  therefore an edit to an untracked research artifact, not a commit to `main`; the
  authoritative verdict is recorded here in §3.5 either way, and that record is what
  the four-phase workflow consumes.

There is a genuine subtlety worth writing down so it is not "re-discovered" as a
contradiction: `SetDamaged` *does* mention 4211001 next to the keydown machinery
(v83 `0x95924F`, v95 `0x935C67`) in the form
`if (id == 4211001 || is_keydown_skill(id)) { if (is_keydown_skill(id)) {…} }`. The
inner test re-filters Chakra straight back out, so the `OnKeyDownSkillEnd` block never
runs for it. The compare is inert for Chakra, on both v83 and v95. It is *not* evidence
of keydown membership, and the design records it explicitly because it is exactly the
kind of near-miss that would reopen this question a third time.

### 3.6 No client-side Chakra stat (OQ-2 / OQ-5 / FR-2.5)

The recovery flag is `CUserLocal::m_nPrepareSkillID` (v83 `this+0x2AE8`, v95
`this+0x3AD4`, JMS 185 `this+0x3010`), a plain local field. A whole-binary write scan
on v83 finds exactly one production writer (`DoActiveSkill_Prepare` `0x96AEB9`) and one
clear (the 12-byte `memset` at the head of `TryDoingPreparedSkill`'s completion branch).
It is never encoded into a packet, and there is no `CHAKRA` entry in the character
temporary stat set on any version.

**Conclusion: `libs/atlas-packet` is not touched.** Inventing a CTS entry would
broadcast a stat no client reads, which is what FR-2.5 was written to prevent.

### 3.7 Immovability, not interruption (OQ-8)

`CUserLocal::IsImmovable` (v83 `0x95F914`) returns true whenever
`m_nPrepareSkillID` is set and the prepared skill is not in the small
"movable while charging" set (`sub_95F96F` — the Monster Magnets, Big Bangs, Hurricane,
Piercing Arrow, Corkscrew, Rapid Fire, Poison Bomb, Evan breaths). Chakra is not in
that set, so an authentic client physically cannot walk, jump, climb or rope during the
1500 ms window.

The client also does **not** cancel Chakra on damage — §3.5's near-miss block is the
only damage-side mention, and it excludes Chakra.

Nexon's own tooltip disagrees with its own client here, and says so on 10 of 11
versions: *"Uses MP to recover HP. Only works when the HP is less than 50%, and it'll
stop if either attacked or moved."* (v12, v48 … v92, JMS 185 `String.wz/Skill.img/4211001/desc`).
The v95 description drops the second clause entirely: *"Uses MP to recover HP. Only
works when you have less than 50% HP."*

**Decision:** implement server-side interruption on damage and on movement uniformly,
per PRD FR-5 (an approved requirement) and corroborated by Nexon's description on ten
of eleven versions. Practically it is a server-authority measure — an authentic client
never triggers it — so it costs nothing in fidelity and closes the crafted-client hole
where a player kites through the window collecting a free heal and a damage reduction.
The v95 silence is silence, not a denial, and is recorded as such.

#### 3.7.1 Addendum — "an authentic client never triggers it" was wrong

Live testing on `atlas-pr-1326` (2026-08-12, GMS 83.1) falsified the claim that the
movement interrupt is inert against an authentic client. `IsImmovable` stops the player
from *walking*; it does not stop the client from *sending a MOVE packet*. Two of every
three legitimate casts died:

```
19:36:35.576  Chakra recovery window opened ... hp [3070] of [30000]
19:36:35.707  Chakra recovery window for character [1] interrupted by movement   (+131 ms)
19:36:36.890  Character [1] sent Chakra USE_SKILL with no open recovery window; rejecting
```

The caster had sent no MOVE for the preceding 19 s (previous one at `19:36:16.325`), so
this is the client emitting a path of its own accord, not the player walking. Across six
casts the interrupt fired at +103, +131, +186, +366, +389 and +409 ms; the casts that
healed were simply the ones no MOVE happened to land in. Symptom: animation plays,
nothing else happens.

**Revision:** the interrupt gates on **displacement**, not on packet arrival —
`movement.Displaces` folds the path with the same folder the movement processor uses to
publish the authoritative position, and cancels only when the path ends somewhere other
than it started. A self-flush lands where it began and passes; a walk, jump-and-land or
teleport does not. The anti-kiting guard is unchanged in substance: kiting is
displacement by definition.

---

## 4. WZ data (OQ-6, FR-6.2, FR-6.3)

### 4.1 Field semantics

`Skill.wz/421.img/skill/4211001` has, per level, exactly: `hs`, `mpCon`, `time`, `x`, `y`.
There is **no `hp` field, no `damage` field, no `prop`, no `cooltime`, no `lt`/`rb`.**

The v95 `String.wz` tooltip template resolves the two ambiguous names:

> `[h] MP Cost: #mpCon, #y% Recovery Rate: #x% of the damage taken`

cross-checked against v48's fully-expanded per-level strings
(`[h1] MP - 15, Recovery rate 9%; Damage during the recovery if hit: 200%` with
`mpCon=15, y=9, x=200`). So:

- **`y` = recovery rate %** → feeds the heal (§3.4).
- **`x` = damage-taken %** → feeds the multiplier (§3.1), and the client confirms it
  independently via `_ZtlSecureGet_nX`.
- `time = 1` on every level of every version. Chakra has no statup and applies no buff,
  so `UseSkill`'s `e.Duration() > 0 && len(statups) > 0` guard already skips the buff
  path. `time` is vestigial and must not be read as the recovery window.

Both `x` and `y` are already parsed by atlas-data
(`services/atlas-data/atlas.com/data/skill/reader.go:266-267`) and already exposed as
`effect.Model.X()` / `.Y()`. **atlas-data needs no change** — FR-6.3 resolved negative.

### 4.2 The per-version tables — FR-6.2 is a v12/v48 table

The PRD's reference table is reproduced faithfully by **GMS 12 and GMS 48 only**. Every
other version ships different numbers, and the sign of the damage effect flips:

| Versions | maxLevel | `mpCon` | `x` (damage taken %) | `y` (recovery %) | Source |
|---|---|---|---|---|---|
| GMS 12, GMS 48 | 30 | 15 (L1-10) / 21 (L11-20) / 27 (L21-30) | **200 → 112**, amplifies | 9 → 200 | explicit `level` nodes |
| GMS 61, 72, 79, 83, 84, 87, 92; JMS 185 | 30 | 15 / 21 / 27 (same breakpoints) | **99 → 70**, *reduces*; `x = 100 − L` | `y = 60 + 8·L` (68 → 300) | explicit `level` nodes, byte-identical across all eight |
| GMS 95 | 10 | `40 + 10·L` (50 → 140) | `100 − 4·L` (96 → 60), reduces | `100 + 20·L` (120 → 300) | `common` formula node |

Verified level-by-level for all 30 levels on GMS 48 and GMS 83; spot-verified at
L1/2/3/5/10/20/29/30 on the rest. The GMS 61→92 + JMS 185 table is a clean pair of
linear rules, but it is still read from WZ per level rather than regenerated — the v12/v48
table is deliberately non-uniform (`y` steps by 9, then 8, then 7, then 6, then 5) and
must not be interpolated.

A trap worth flagging: on v83 the `String.wz` `h1..h30` strings are **stale**, still
carrying the v48 numbers (9 %/200 % at L1) while `Skill.wz` says 68 %/99 %. The `Skill.wz`
values are what the client's code reads. Do not use the tooltip strings as a data source.

**FR-6.2 discrepancy record (required by FR-6.2 / Acceptance Criteria):** the PRD's
table is correct for GMS 12 and GMS 48 and incorrect for the other nine columns. The WZ
values win; the implementation reads `x` and `y` from `effect.Model` per level and per
tenant version and hardcodes nothing.

**FR-4.1 amendment:** "the incoming-damage penalty" is a penalty only on v12/v48. The
single expression `damage * x / 100` (floor 1) produces amplification on v12/v48 and
reduction on v61+ with **no version gate** — the WZ data carries the divergence.
FR-9.3's `MajorAtLeast` idiom is therefore not needed for this term, and adding one
would be the bug rather than the fix.

**v95 `common` risk.** GMS 95 is the only version routed through task-192's
`synthesizeCommonNodes` expansion. A test must pin that 4211001 expands to 10 levels
with `x` 96→60, `y` 120→300, `mpCon` 50→140; a silent expansion failure there yields
`x=0`, which would zero every hit the caster takes during the window.

### 4.3 The recovery window is 1500 ms — derived (OQ-3)

`DoActiveSkill_Prepare` sets the end time as `now + <one-time-action delay>` and
`TryDoingPreparedSkill` fires when `now - end >= 0`. The delay is the `prepare`
animation's length, which WZ states directly: `4211001/prepare` has **15 frames, each
`delay = 100`** → **1500 ms**, identical on GMS 48 and GMS 83 (`action: 0 = alert`).

Independently, both `DoActiveSkill_Prepare` and `SetDamaged` write the constant
**5000** into `CUserLocal+0x56C` on the Chakra path — the client's own outer bound on a
skill in progress.

**Design consequence:** the server does not simulate the window; the client tells it
when the cast completes by sending `USE_SKILL`. The server-side TTL is a *safety bound*,
not a timing model. Use **5000 ms**, matching the client's own bound and leaving headroom
over 1500 ms for latency. It is not level-dependent and not version-dependent.

---

## 5. Architecture

### 5.1 Components

```
services/atlas-channel/atlas.com/channel/
  skill/handler/chakra/
    chakra.go      Handler registered on skill2.ChiefBanditChakra (USE_SKILL path)
    formula.go     ChakraRecovery(base, y) + chakraBase(luk)   ← pure, unit-tested
    state.go       recovery-state registry (singleton, tenant-scoped, TTL)
  skill/handler/registrations/registrations.go   + blank import
  socket/handler/character_skill_use.go          + pre-cost activation gate
  socket/handler/character_skill_prepare.go      + start recovery state
  socket/handler/character_damage.go             + chakraPct into mitigationInput
  socket/handler/character_damage_mitigation.go  + chakraPct term, applied first
  socket/handler/character_move.go               + interrupt hook
```

`state.go` follows the service's existing singleton-registry convention
(`sync.Once` + `sync.RWMutex`), keyed on `(tenant.Id, characterId)`, holding
`{skillLevel byte, x int16, y int16, startedAt time.Time}`. It is in-process by
design: FR-2.4 and the NFR forbid a cross-service call per hit, and the state is
inherently channel-local (the caster is standing still in one map on one channel for
1500 ms). Redis is not used, so `tools/redis-key-guard.sh` is not engaged.

Expiry is lazy — every read compares `startedAt` against the 5 s TTL and treats an
expired entry as absent — plus one `routine.Go` sweeper per registry that evicts stale
entries. Lazy expiry is what makes correctness independent of the sweeper; the sweeper
only bounds memory. Any goroutine goes through `routine.Go`
(`tools/goroutine-guard.sh`).

The registry snapshots `x` and `y` **at prepare time**, so the damage path never needs
an atlas-data round trip per hit, and a mid-window skill-book change cannot desync the
multiplier from the heal.

### 5.2 Data flow

```
CharacterSkillPrepareHandle  (already exists, already bound)
    │  skillId resolves to ChiefBanditChakra?
    ├─ no  → existing keydown-broadcast behaviour, unchanged
    └─ yes → load caster + effective stats + effect(level)
             gate: 2*HP >= effectiveMaxHP → log, do nothing (no state, no packet)
             else → chakra.Start(tenant, charId, level, x, y)
                    (no foreign broadcast: Chakra is not keydown, §3.5)

CharacterDamageHandle → processDamageTaken
    └─ chakra.Get(tenant, charId) → present?
         └─ in.chakraPct = x     → computeMitigation applies it FIRST (§3.3)
         └─ after damage is applied: chakra.Interrupt(..., "damaged")

CharacterMoveHandle
    └─ chakra.Interrupt(..., "moved")           (crafted-client defence, §3.7)

CharacterSkillUseHandle  (USE_SKILL, ~1500 ms later)
    ├─ pre-cost gate, mirroring the Hero Enrage precedent at
    │  character_skill_use.go:116-139 — reject BEFORE handler.UseSkill so no MP,
    │  no cooldown, and call enableActions(...) to unlock the client:
    │     • 2*HP >= effectiveMaxHP        → reject
    │     • no recovery state present     → reject (never prepared, or interrupted)
    └─ handler.UseSkill → cost/cooldown → chakra.Apply:
           heal  = ChakraRecovery(chakraBase(luk), y)
           delta = clamp(heal, 0, effectiveMaxHP - currentHP)
           character.ChangeHP(field, charId, delta)     (skip when delta == 0)
           CharacterEffect (self) + CharacterEffectForeign (map)
           chakra.Clear(tenant, charId)
```

FR-8.4 (no XP) is satisfied by omission — Chakra has no `AwardExperience` call at all,
unlike `heal.go` step 7.

Death (FR-5.3) needs no new hook: the pending heal only fires when a `USE_SKILL` packet
arrives, and a dead character's cast is already rejected upstream; the state entry is
additionally cleared on the damage path that killed them, and expires regardless.
Map change / channel change / disconnect (FR-5.5) are covered by the TTL plus an
explicit clear on the existing session-destroy path.

### 5.3 The mitigation change

`mitigationInput` gains one field and `computeMitigation` gains a three-line prologue,
keeping it a pure function:

```go
// chakraPct is the WZ `x` of the caster's active Chakra recovery window
// (0 = not recovering). The client rewrites the raw damage by this factor
// before every other term reads it (design §3.3), so it is applied first.
// x > 100 amplifies (GMS 12/48); x < 100 reduces (GMS 61+) — the WZ data
// carries the direction, so there is no version gate here.
chakraPct int32
```

```go
if in.chakraPct > 0 {
    raw = raw * in.chakraPct / 100
    if raw <= 1 {
        raw = 1
    }
}
```

The `<= 1 → 1` floor is the client's, verbatim, and is deliberately *not* `< 1`.
`mitigationBreakdown` gains a `chakraAmplified int32` so the existing per-hit debug line
reports the pre- and post-Chakra damage — without it, "Chakra did nothing" is
undiagnosable from logs (NFR: Observability).

Placing the term inside `computeMitigation` rather than pre-multiplying `raw` at the
call site keeps every damage-math assertion in one pure, table-testable function, which
is how task-157 left it and how the existing `character_damage_mitigation_test.go` is
structured.

### 5.4 Version handling

There is **no `MajorAtLeast` gate anywhere in this change.** Every version difference
Chakra has is a data difference that `effect.Model` already carries per tenant version:
the table swap at v61, the `common` expansion at v95, the maxLevel change from 30 to 10.
The handler registers once on `skill2.ChiefBanditChakra` (FR-9.1), which is what
`tools/skill-job-id-guard.sh` requires, and no code path compares a raw wire id.

GMS 12 is the one column with no IDA corroboration (§1). Its WZ data is read and is
identical to v48's, and its `Data.wz` `Skill/421` contains 4211001, so the data path is
verified; only the client-behaviour claims (gate, multiplier placement, immovability)
are inferred there from v48. Recorded per FR-9.4.

---

## 6. Alternatives considered

### 6.1 Where to start the recovery state — prepare packet vs. `USE_SKILL`

| Option | Verdict |
|---|---|
| **Start on `CharacterSkillPrepareHandle` (chosen)** | The only option that makes the damage multiplier work at all. The window *is* the gap between prepare and use; starting at `USE_SKILL` would open it exactly when the client closes it. Handler and template bindings already exist. |
| Start on `USE_SKILL`, apply the multiplier retroactively | Would require buffering damage events for 1500 ms and re-applying them, and would break Power Guard reflects (already emitted at raw). Rejected. |
| Skip the state; trust a client flag on the damage packet | The damage packet carries no Chakra flag, and trusting one would violate the FR-4.2 / task-157 server-authority posture. Rejected. |

### 6.2 Where the activation gate lives

| Option | Verdict |
|---|---|
| **Pre-cost gate in `character_skill_use.go` + a gate on the prepare packet (chosen)** | Follows the Hero Enrage precedent already in that file (`:116-139`): reject before `handler.UseSkill`, call `enableActions` so the client unlocks. Zero MP, zero cooldown on rejection (FR-1.4). Gating the prepare packet as well means a crafted client that skips prepare still fails at `USE_SKILL`. |
| Gate inside the Chakra `Handler` | `UseSkill` charges MP and applies cooldown *before* it dispatches to the registry (`common.go:130-160`). A gate there would have to refund, which is the task-200 double-accounting trap inverted. Rejected. |
| Add a generic `CastGate` registry beside the `Handler` registry | Attractive and probably right eventually, but it is speculative generality for one caller today. Chakra should follow Enrage; if a third gated skill appears, extract then. Rejected (YAGNI). |

### 6.3 Two templates are missing the prepare-handler binding

`CharacterSkillPrepareHandle` is bound in nine of eleven seed templates.
`template_gms_12_1.json` and `template_gms_92_1.json` bind only the
`CharacterSkillPrepareForeign` **writer**, not the handler — so on those two versions
the prepare packet is dropped and the recovery window never opens.

This is in scope: it is a prerequisite this task can produce itself, not a follow-up.
Adding the two handler entries engages `tools/template-opcode-order-guard.sh`
(sorted insertion, never appended next to a related entry),
`tools/template-duplicate-binding-guard.sh` and
`tools/template-movement-types-guard.sh`. The opcode for each version comes from the
registry/matrix, not from a neighbouring template.

**Plan-phase correction — GMS 12 is NOT edited.** Verified in the task-213
worktree at plan time: `template_gms_12_1.json` has 24 handlers total, ending
at `0xB1 SummonDamageHandle`, and it has no `CharacterUseSkillHandle`, so the
second half of Chakra's two-packet flow (`USE_SKILL`) has nowhere to land on
that column regardless of whether the prepare handler is bound. Its writer
list contains no `CharacterSkillPrepareForeign` — that writer exists only on
`template_gms_92_1.json`, not on GMS 12 as this section originally stated.
There is no `docs/packets/registry/gms_v12.yaml` and no
`docs/packets/ida-exports/gms_v12.json` — GMS 12 has no IDB at all (§1) —
so there is no authority for a `SKILL_EFFECT` opcode on that column, and
copying a neighbour's would be a fabricated wire value. GMS 12 is therefore
recorded as out of reach for this feature per PRD FR-9.4: Chakra is inert on
that column (prepare packet unrouted, `USE_SKILL` unhandled) and needs no Go
change to stay inert, since the `USE_SKILL` gate already rejects on a missing
recovery window. Only `template_gms_92_1.json` was edited for this task.

### 6.4 Randomised heal

Rejected — see §3.4. `ChakraRecovery` takes no RNG parameter, so re-introducing
randomisation later is a signature change that forces its tests to be revisited, which
is the desired friction.

---

## 7. Testing

Pure functions, table-driven, no service round trips:

| Test | Pins |
|---|---|
| `chakraCanActivate` | 49 % / exactly 50 % / 51 % of MaxHP; the `2*HP >= MaxHP` integer form against the client's `HP*100/MHP >= 50` over a swept MaxHP range including odd values; zero/absurd effective MaxHP → base fallback |
| `ChakraRecovery` | `base * y / 100` at v48 `y` ∈ {9, 200}, v83 `y` ∈ {68, 300}, v95 `y` ∈ {120, 300}; `y = 0`; negative/zero base → 0, never a negative delta (FR-3.5) |
| clamp | `heal > MaxHP - HP` → exactly `MaxHP - HP`; at full HP → 0 and **no `ChangeHP` call** |
| `computeMitigation` chakra term | `x = 200` (v48 L1) doubles; `x = 112` (v48 L30); `x = 99` (v83 L1) and `x = 70` (v83 L30) *reduce*; `x = 60` (v95 L10); the `<= 1 → 1` floor; `chakraPct = 0` is a no-op; **ordering**: with Achilles + Combo Barrier + Magic Guard all active, the result equals the chain applied to the already-multiplied damage, not the raw |
| state registry | expiry at TTL; clear on completion; clear on interrupt; tenant isolation (same characterId in two tenants); `go test -race` with concurrent prepare / damage-path read / move-path interrupt / sweeper |
| interruption | damaged → amplified damage applied **and** state cleared (FR-4.5); moved → cleared; a `USE_SKILL` arriving after an interrupt is rejected and spends nothing |
| atlas-data expansion | 4211001 at GMS 95 expands from `common` to 10 levels with `x` 96→60, `y` 120→300, `mpCon` 50→140; at GMS 83 to 30 levels with `x` 99→70, `y` 68→300 |
| keydown regression | `IsKeyDownSkill(ChiefBanditChakraId) == false` still holds — task-161's existing pin, re-run, not rewritten |

Test setup uses the project Builder pattern; no `*_testhelpers.go`.

## 8. Verification gates

`go test -race ./...`, `go vet ./...`, `go build ./...` in every changed module;
`tools/lint.sh --check`, `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`,
`tools/skill-job-id-guard.sh` from the repo root; the three template guards for §6.3;
`docker buildx bake atlas-channel` only if `go.mod` is touched (it should not be).
Code review via `superpowers:requesting-code-review` before the PR.

## 9. Risks

| Risk | Mitigation |
|---|---|
| The heal magnitude is community-sourced and could be wrong by a constant factor | Isolated in `ChakraRecovery` + `chakraBase`; changing it is a one-function, one-test-file edit. The uncertainty is documented in code and here, not hidden. |
| The v61 table inversion gets "corrected" back to the PRD table by a later reader | §4.2 records the WZ evidence; the mitigation tests pin `x = 99` and `x = 70` as *reductions*, so a revert fails loudly. |
| v95 `common` expansion regresses → `x = 0` → zero damage during the window | Explicit expansion test in §7; `chakraPct <= 0` is treated as "no term", never as "zero damage". |
| A future editor adds Chakra to `IsKeyDownSkill` citing the `SetDamaged` compare | §3.5 documents why that compare is inert, and task-161's negative-assertion tests fail if it is added. |
| Hot-path cost on the damage handler | One `sync.RWMutex` read of an in-process map per hit, and only when an entry exists; no I/O (FR-2.4, NFR Hot-path cost). |
