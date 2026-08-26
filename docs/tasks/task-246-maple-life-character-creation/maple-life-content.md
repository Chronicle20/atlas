# Maple Life content values

Task 16 of the task-246 plan, Amendment 1. This is the **single source** every
later content value (Tasks 20/21) is copied from. Per the plan's Amendment
Global Constraints: "no task from Task 20 onward may write a content value
that Task 16 did not produce." Every id below carries its WZ file path, its
IDA address, or its repo-source citation. Anything that could not be closed
this pass is listed in §4 as UNRESOLVED or UNCONFIRMED — it is not filled in
with a plausible number.

WZ paths below use `<local-wz-root>` as a placeholder for the machine-local
read-only WZ data root supplied by the controller for this task
(`Cosmic/wz/`, a v83-era GMS WZ snapshot) — not a repo path.

Scope note: per the plan's Amendment Global Constraints, the in-scope tenants
for the `mapleLife` block are `template_gms_83_1.json`, `template_gms_87_1.json`,
`template_gms_92_1.json`, `template_gms_95_1.json` only. `template_gms_84_1.json`
gets no `mapleLife` block (VERSION-ABSENT). The content values below are
derived against the gms_v95 IDB (session `ecc757f4`) and the local
`<local-wz-root>/` tree; §11 A3 already established that v87/v92/v95's
`SendCreateNewCharacter` encode order is identical in shape, so these values
are recorded once and are expected to apply to all four in-scope tenants
unless a later task's own derivation finds a version-specific divergence.

---

## §1 — the two WZ paths and the per-gender option lists

### The StringPool ids, resolved

`CUICharacterSaleDlg::LoadNewCharInfo` (gms_v95 `0x777790`) resolves the male
path through `StringPool::GetBSTR(id = 1525)` (decompiled: `v44 = 1525;` at
`0x777845`) and the female path through `id = 1526` (`v44 = 1526;` at
`0x7779bd`). Neither literal string is present as plain UTF-16 in the binary
(confirmed: `find_regex`/UTF-16LE search for `MakeCharInfo` returns zero
matches), because `StringPool::GetString` (`0x746750`) stores every string
**byte-encoded and XOR-obfuscated**, not as a plain literal.

**Resolved this pass, by decoding the cipher directly rather than guessing a
node name:**

- `StringPool::GetString` (`0x746750`) reads `StringPool::ms_aString[nIdx]` —
  a global pointer table at `0xc5a878` (confirmed via `get_int`: index 1525 is
  at `0xC5C04C` → pointer `0xB8BCA4`; index 1526 is at `0xC5C050` → pointer
  `0xB8BC7C`). The first byte at the pointer is a seed byte; the remaining
  bytes (up to the first literal `0x00`) are the ciphertext.
- The cipher is `StringPool::Key` (`0x746470`/`0x746230`): a 16-byte static
  key (`StringPool::ms_aKey`, `0xb98830`, bytes
  `d6 de 75 86 46 64 a3 71 e8 e6 7b d3 33 30 e7 2e`), **bit-rotated left by
  the seed value** (confirmed via `rotatel<unsigned char>`, `0x746270` — the
  rotation amount is in bits, not bytes), then XORed byte-for-byte against the
  ciphertext (with the identity special-case `if (byte == key) byte = key`
  from `Decode<char>`, `0x746520`, to avoid an accidental embedded `0x00`).
- Reading the raw ciphertext bytes with `get_bytes` and replaying this cipher
  in Python (brute-forcing the seed-as-bit-shift against both buffers,
  scoring on 100%-printable-ASCII output) decisively resolves both strings:

  | id | seed (raw first byte) | decoded string |
  |---|---|---|
  | 1525 (male) | `0x2c` = 44 | `Etc/MakeCharInfo.img/Info/CharMale` |
  | 1526 (female) | `0x2d` = 45 | `Etc/MakeCharInfo.img/Info/CharFemale` |

  Both decodes are unique — only the correct bit-shift (44 and 45
  respectively, matching each buffer's own seed byte) produces
  100%-printable output; every other shift in `0..127` produces binary
  garbage. This is decisive, not a guess.

**This settles the `PremiumChar{Male,Female}` lead as WRONG.** The resolved
path is `Info/CharMale` / `Info/CharFemale`, not `PremiumCharMale` /
`PremiumCharFemale`. (For what it's worth, in the local v83-era WZ snapshot
the two node families happen to hold byte-identical data, so the choice
between them would not have changed the option-list *values* — but the path
itself is now a decoded fact, not an assumption, and the two nodes are **not**
guaranteed to stay identical across content patches, so using the wrong one
would have been a live bug waiting to happen.)

### The per-gender option lists

Read directly from `<local-wz-root>/Etc.wz/MakeCharInfo.img.xml`, node
`Info/CharMale` and `Info/CharFemale`. Per
`services/atlas-data/atlas.com/data/characters/templates/reader.go`'s
child-index convention (`0`=faces, `1`=hair styles, `2`=hair colours,
`3`=skin colours, `4`=tops, `5`=bottoms, `6`=shoes, `7`=weapons), and per §11
A3's slot mapping (only child indices `0..3` are wired to `al[0..3]`):

**`Info/CharMale`:**

| slot | wire field | values |
|---|---|---|
| 0 — faces | `al0` | `20000, 20001, 20002` |
| 1 — hair styles | `al1` | `30030, 30020, 30000` |
| 2 — hair colours | `al2` | `0, 7, 3, 2` |
| 3 — skin colours | `al3` | `0, 1, 2, 3` |
| 4 — tops (not on the wire) | — | `1040002, 1040006, 1040010` |
| 5 — bottoms (not on the wire) | — | `1060002, 1060006` |
| 6 — shoes (not on the wire) | — | `1072001, 1072005, 1072037, 1072038` |
| 7 — weapons (not on the wire) | — | `1302000, 1322005, 1312004` |

**`Info/CharFemale`:**

| slot | wire field | values |
|---|---|---|
| 0 — faces | `al0` | `21000, 21001, 21002` |
| 1 — hair styles | `al1` | `31000, 31040, 31050` |
| 2 — hair colours | `al2` | `0, 7, 3, 2` |
| 3 — skin colours | `al3` | `0, 1, 2, 3` |
| 4 — tops (not on the wire) | — | `1041002, 1041006, 1041010, 1041011` |
| 5 — bottoms (not on the wire) | — | `1061002, 1061008` |
| 6 — shoes (not on the wire) | — | `1072001, 1072005, 1072037, 1072038` |
| 7 — weapons (not on the wire) | — | `1302000, 1322005, 1312004` |

Cross-check: `services/atlas-configurations/seed-data/templates/template_gms_95_1.json`'s
`characters.templates[]` entry with `jobIndex: 0, subJobIndex: 0` (Beginner)
carries the identical `faces`/`hairs`/`skinColors`/`tops`/`bottoms`/`shoes`/`weapons`
value sets per gender (its `hairColors` lists the same four values in a
different order, `[0,2,3,7]` vs the WZ file's `[0,7,3,2]` — same set). This
confirms `Info/CharMale`/`Info/CharFemale` is the same resource the vanilla
(Beginner) creation screen already uses — CUICharacterSaleDlg (the Maple Life
dialog) reuses the vanilla creation resource for its four "look" choices, and
per §11 A3 the dialog exposes only the first four slots (no equipment choice)
even though the underlying resource also carries tops/bottoms/shoes/weapons
option lists (slots 4-7). **Slots 4-7 of this per-gender node are generic
Beginner-creation options, not per-class equipment** — see §3's equipment
derivation for why they are not reused directly for the five Maple Life
classes.

---

## §2 — the ordinal → class table

| ordinal | class | confidence |
|---|---|---|
| 0 | Warrior | **derived**, §11 A2 (already closed; not re-derived this pass) |
| 1 | Magician | **derived**, §11 A2 (already closed; not re-derived this pass) |
| 2 | Bowman | **UNCONFIRMED** — see below |
| 3 | Thief | **UNCONFIRMED** — see below |
| 4 | Pirate | **UNCONFIRMED** — see below |

**What this pass tried for 2/3/4 (§11 A6, bounded):** `CUICharacterSaleDlg::OnCreate`
(gms_v95 `0x77adc0`) is the function that builds `m_apCanvasClass` and would
settle the order, but it decompiles to **124,912 characters of pseudocode**
(`size 0x3ff8`, by far the largest function in the class) — full decompilation
would consume the bulk of this task's tool-call budget on its own.
`CUICharacterSaleDlg::ShowClass` (`0x7761b0`) was decompiled in full instead —
it confirms `m_apCanvasClass.a[m_nCurrentClass]` and
`m_apCanvasClass.a[m_nCurrentClass + 5]` are the two per-class canvas slots
(icon + name, 5 classes × 2), but contains no string literal or id that
identifies which canvas belongs to which class name. A `search_text` sweep of
`OnCreate`'s address range for `StringPool::GetBSTR` calls returned **51 call
sites** — an order of magnitude more than the single-digit run
`LoadSPInfo` used for its 11 SP-description strings — meaning the class-name
ids are not laid out as a clean consecutive run the way `strSPWarrior`/`strSPMagician`
were; isolating the 5 relevant calls among 51 (most of which are almost
certainly unrelated UI layout resource paths for `OnCreate`'s many controls)
was judged not to fit this task's bounded pass, per the brief's explicit
instruction not to burn the whole budget on this one sub-question.

**Ruling (per §11 A6, restated): ship with the guide's point-5 ordering as the
shipped value** — Bowman (2), Thief (3), Pirate (4) — flagged UNCONFIRMED in
tenant seed data. §11 A6 already rules this is ordered tenant configuration,
so a wrong order is a seed-data fix, not a code change; Task 20 must carry
the same UNCONFIRMED marking, and it blocks live testing (not shipping).

---

## §3 — the five class entries

### Job ids

Sourced from `libs/atlas-constants/job/constants.go` (this repo's canonical
job-id constant table, already used across every service that models a job
id):

| ordinal | class | `jobId` | source |
|---|---|---|---|
| 0 | Warrior | `100` (`WarriorId`) | `libs/atlas-constants/job/constants.go:96` |
| 1 | Magician | `200` (`MagicianId`) | `libs/atlas-constants/job/constants.go:106` |
| 2 | Bowman | `300` (`BowmanId`) | `libs/atlas-constants/job/constants.go:116` |
| 3 | Thief | `400` (`RogueId`) | `libs/atlas-constants/job/constants.go:123` — this codebase's constant name for the Thief-track first job is `RogueId`; the wire/guide name is "Thief" (corrected from `:127` — see the task-16 review's Finding 2; the other four job-id line citations in this table were re-checked against the same file and are accurate: `WarriorId:96`, `MagicianId:106`, `BowmanId:116`, `PirateId:130`) |
| 4 | Pirate | `500` (`PirateId`) | `libs/atlas-constants/job/constants.go:130` |

### `level`

`30` for all five, per §11 A2's user ruling. Not derived; a ruling, restated
here as instructed.

### `ap` / `sp` — the accumulated totals and the arithmetic

**USER RULING (post-live-test correction — see the accompanying bug report,
`bug-ap-sp-and-starting-equipment.md`): the client's own pre-submit preview is
the contract, not the level-up-arithmetic model this subsection originally
derived in isolation.** The corrected model below still uses the same
level-up primitives (`computeOnLevelAddedAP`/`computeOnLevelAddedSP`), but
adds the character-creation stat pool and the 1st-job-advancement SP grant
that the original derivation omitted — both omissions were confirmed against
a live client observation (DEX-20 class, ordinal 4/Pirate: client promised
138 AP / 61 SP).

**AP.** `computeOnLevelAddedAP(jobId, level)`
(`services/atlas-character/atlas.com/character/character/processor.go:1685-1697`)
returns a flat `5` for any non-Cygnus job — all five Maple Life classes are
non-Cygnus (`job.IsCygnus` checks `GetType(jobId) == TypeCygnus`, and
100/200/300/400/500 are all `TypeExplorer`). A level-30 character has gone
through 29 level-ups (level 1→2, 2→3, … 29→30), each granting AP by this
formula while already at the character's final job (Maple Life skips the
Beginner phase entirely — the character is created already-advanced, so every
level-up in this hypothetical history uses the advanced-job formula, not the
Beginner one). **A real level-1 MapleStory character also starts with a
25-point creation stat pool** (`Σstats = 25`, not `ResetStats`' `baseStat`
floor of `4/4/4/4 = 16`) — the original derivation omitted this pool, which is
exactly the 9-point shortfall (138 − 129) the live test exposed:

```
total AP pool by level 30 = 25 + 5 × 29 = 170
```

**SP.** `computeOnLevelAddedSP(jobId, effectiveLevel)`
(`processor.go:1741-1749`) returns `3` only for a job that is not
`job.IsBeginner`. A real character is a Beginner (job id `0`) until its
1st-job advancement at level 10, so the `3`-per-level rate only applies for
levels 11→30 (20 level-ups), not from level 1 as the original derivation
assumed. On top of that, Nexon/Cosmic's `changeJob` grants **`+1` SP at the
moment of 1st-job advancement** (`Cosmic/src/main/java/client/Character.java:1154`,
`int spGain = 1`) — Atlas's own `ChangeJob`
(`services/atlas-character/atlas.com/character/character/processor.go:592-614`)
does not grant this `+1` (a pre-existing divergence from Nexon/Cosmic, out of
this task's scope — see the bug report's "Not yet answered" section), but the
client's own preview promises it, and per user ruling the client's number is
the contract:

```
total SP pool by level 30 = 1 (1st-job advancement grant) + 3 × 20 (level-ups 11→30) = 61
```

**`ap` (unspent remainder) per class** = `170 − Σstats`, where `Σstats` is the
class's four primary stats summed (the stat floor plus the three stats left
at `baseStat = 4` — `ResetStats`, `processor.go:2178`).

**`sp` per class** = `61` (the full accumulated total) for **every** class.
For ordinals 0/1 only, `nSP` (the player's 0..10 choice) is deducted from this
pool **at creation time by the submit-handling code**, per §11 A4 — that
deduction is a runtime computation on the player's wire value, not a content
value Task 16 can pre-compute, so `sp` here is recorded as the pre-deduction
total for all five classes (`spendSPPool`,
`services/atlas-character-factory/atlas.com/character-factory/factory/maple_life.go:137-146`,
deducts from slot 0 of this 61 pool; 61 ≥ 10 so no underflow).

### Stat floors (minimum AP the 1st job requires)

Sourced from this repo's own already-seeded job-advancement NPC conversation
scripts (`deploy/seed/gms/95_1/npc-conversations/npc/*.json`), each of which
carries a `rebalance_ap` operation with an explicit `targets` floor
immediately before its `change_job` operation to the class's job id — this is
the same mechanism (`operation_executor.go:2366-2392`, `RebalanceAPPayload`)
a live 1st-job advancement uses, and it is genuine repo source, not WZ, but it
is itself derived content already vetted for this exact purpose (advancing
Beginner → 1st job) and cited per-file below:

| ordinal | class | NPC (job-advance instructor) | job id it advances to | stat floor | AP spent (`floor − 4`) |
|---|---|---|---|---|---|
| 0 | Warrior | `npc-1022000.json` | `100` | `strength: 35` | `31` |
| 1 | Magician | `npc-1032001.json` | `200` | `intelligence: 20` | `16` |
| 2 | Bowman | `npc-1012100.json` | `300` | `dexterity: 25` | `21` |
| 3 | Thief | `npc-1052001.json` | `400` | `dexterity: 25` | `21` |
| 4 | Pirate | `npc-1090000.json` | `500` | `dexterity: 20` | `16` |

(Note the NPC-id-to-job-name mapping is **not** alphabetical/sequential —
confirmed by reading each file's own `change_job` `jobId` param, not assumed
from the NPC id.)

### `stats` — full per-class stat block

`baseStat = 4` (`processor.go:2168`) for the three stats not raised; the
fourth is raised to its floor. `ap` is `170 − Σstats`, per the corrected
model above.

| ordinal | class | str | dex | int | luk | Σstats | `ap` (unspent) |
|---|---|---|---|---|---|---|---|
| 0 | Warrior | **35** | 4 | 4 | 4 | 47 | `170 − 47 = 123` |
| 1 | Magician | 4 | 4 | **20** | 4 | 32 | `170 − 32 = 138` |
| 2 | Bowman | 4 | **25** | 4 | 4 | 37 | `170 − 37 = 133` |
| 3 | Thief | 4 | **25** | 4 | 4 | 37 | `170 − 37 = 133` |
| 4 | Pirate | 4 | **20** | 4 | 4 | 32 | `170 − 32 = 138` |

### `hp` / `mp` — USER RULING: midpoint of the interval, skill-excluded (§5 has the seeded values)

**Not cleanly derivable as a single number.** `resolveHPMPGainParams`
(`processor.go:1720-1802`) gives each job-family a **random range** per
level-up (e.g. Warrior-family: `hpLower=24, hpUpper=28, mpLower=4, mpUpper=6`),
not a fixed per-level amount — this is genuinely randomized in live gameplay,
and there is no deterministic WZ-sourced total to cite. Recording a single
"plausible" number here would be exactly the kind of invented value CLAUDE.md
prohibits.

What is cited and reproducible: a new character's base HP/MP before any
level-ups is `Hp: 50, Mp: 5` (`services/atlas-character-factory/atlas.com/character-factory/factory/processor_test.go:1524`,
`services/atlas-character/atlas.com/character/character/builder.go:126` —
this repo's own creation-default convention, not WZ, but consistently used
across this codebase's Beginner-creation tests and builders). Applying each
class's `hpLower..hpUpper` / `mpLower..mpUpper` range across 29 level-ups gives
an **interval**, not a point value:

| ordinal | class | hp range (base 50 + 29×[lower,upper]) | mp range (base 5 + 29×[lower,upper]) |
|---|---|---|---|
| 0 | Warrior | `50 + 29×24..29×28 = 746..862` | `5 + 29×4..29×6 = 121..179` |
| 1 | Magician | `50 + 29×10..29×14 = 340..456` | `5 + 29×22..29×24 = 643..701` |
| 2 | Bowman | `50 + 29×20..29×24 = 630..746` | `5 + 29×14..29×16 = 411..469` |
| 3 | Thief | `50 + 29×20..29×24 = 630..746` | `5 + 29×14..29×16 = 411..469` |
| 4 | Pirate | `50 + 29×22..29×28 = 688..862` | `5 + 29×18..29×23 = 527..672` |

**The user has since ruled: the midpoint of this interval.** §5 shows the
midpoint arithmetic per class (the `StatBlock.Hp`/`Mp` value Task 20 seeds)
and — because the user's ruling explicitly calls out the Warrior/Magician
`spSkillId`'s own level-up contribution — the separate, submit-time skill
table Task 22 must add on top of that seeded midpoint for those two classes.
See §5; this table's intervals remain here as the skill-excluded base the
midpoint is computed from.

### `spSkillId` (ordinals 0/1 only)

Sourced from `libs/atlas-constants/skill/constants.go`, matching §11 A4's
naming ("Improved Max HP Increase" for Warrior, "Improved Max MP Increase" for
Magician):

| ordinal | class | `spSkillId` | source |
|---|---|---|---|
| 0 | Warrior | `1000001` (`WarriorImprovedMaxHpIncreaseId`) | `libs/atlas-constants/skill/constants.go:2933` |
| 1 | Magician | `2000001` (`MagicianImprovedMaxMpIncreaseId`) | `libs/atlas-constants/skill/constants.go:3023` |
| 2 | Bowman | absent — no SP step offered (§11 A2/A4) | — |
| 3 | Thief | absent — no SP step offered (§11 A2/A4) | — |
| 4 | Pirate | absent — no SP step offered (§11 A2/A4) | — |

### `mapId` — USER RULING: the class's town map (§5 supersedes the prior candidate sweep)

**This subsection is superseded by §5.** The prior pass (job-advancement
NPC's home map, by `grep`) produced an ambiguous two-candidate result for
Thief and Pirate, and — per the task-16 review's Finding 1 — an
under-swept "single match" claim for Warrior that a full sweep did not
actually support. The user has since ruled: **the starting map is the
class's town map** (Warrior → Perion, Magician → Ellinia, Bowman →
Henesys, Thief → Kerning City, Pirate → Nautilus Harbor). See §5 for the
five sourced ids and the Nautilus disambiguation. The `mapId` row in the
per-class table below is the ruled town-map id, not a job-advancement-NPC
proxy.

| ordinal | class | `mapId` | source |
|---|---|---|---|
| 0 | Warrior | `102000000` (Perion) | `libs/atlas-constants/map/constants.go:158` (`VictoriaRoadPerionId`); see §5 |
| 1 | Magician | `101000000` (Ellinia) | `libs/atlas-constants/map/constants.go:99` (`VictoriaRoadElliniaId`); see §5 |
| 2 | Bowman | `100000000` (Henesys) | `libs/atlas-constants/map/constants.go:42` (`VictoriaRoadHenesysId`); see §5 |
| 3 | Thief | `103000000` (Kerning City) | `libs/atlas-constants/map/constants.go:183` (`VictoriaRoadKerningCityId`); see §5 |
| 4 | Pirate | `120000000` (Nautilus Harbor) | `libs/atlas-constants/map/constants.go:521` (`VictoriaRoadNautilusHarborId`); see §5 for why this node, not one of Nautilus's several interior sub-maps |

### `equipment` — USER RULING: the HeavenMS/Cosmic per-class starting-equipment tables

**This subsection is superseded by the user's ruling in
`bug-ap-sp-and-starting-equipment.md`.** The prior pass's derivation (below,
retained for history) synthesized generic Beginner-creation apparel with the
per-class weapon from job-advancement NPC data — a bridge across two
separately-cited sources, not a single per-class table. The user has since
ruled: **starting equipment follows the HeavenMS/Cosmic per-class tables**,
awarding every listed piece (apparel and weapon) per ordinal and gender, with
`useAverageStats: true` on every entry (consistent with the existing rows and
with §5.2's Pirate two-weapon precedent).

| ordinal | gender 0 (male) | gender 1 (female) |
|---|---|---|
| 0 Warrior | `1040021, 1060016, 1072039, 1302008, 1442001, 1422001, 1312005` | `1051010, 1072039, 1302008, 1442001, 1422001, 1312005` |
| 1 Magician | `1050003, 1072075, 1372003, 1382017` | `1041041, 1061034, 1072075, 1372003, 1382017` |
| 2 Bowman | `1040067, 1060056, 1072081, 1452005, 1462000` | `1041054, 1061050, 1072081, 1452005, 1462000` |
| 3 Thief | `1040057, 1060043, 1072032, 1472008, 1332012` | `1041047, 1061043, 1072032, 1472008, 1332012` |
| 4 Pirate | `1052107, 1072294, 1482004, 1492004` | `1052107, 1072294, 1482004, 1492004` |

(Warrior female `1051010` and Magician male `1050003` and Pirate `1052107` are
overalls, which is why those rows have no separate bottom. Pirate is not
gender-split in the source table, so both gender rows get the same list.)

**Prior derivation (superseded, retained for history).** §1 established that
`Etc/MakeCharInfo.img/Info/CharMale`/`CharFemale` slots 4-7 are the generic
Beginner-creation equipment option lists, shared by all classes (this is the
same resource `characters.templates[]`'s `jobIndex: 0` — Beginner — entry
already uses). No per-class top/bottom/shoes WZ resource was found this pass.
Recorded value at that time: **the first (index-0) option in each per-gender
list**, matching the option that sits at the front of the client's own
`ZArray<ASITEM>` before any "next/previous" UI interaction (`GetSelectedAL`
reads element 0 — see `selected-al-derivation.md` Q1):

| gender | top | bottom | shoes |
|---|---|---|---|
| male | `1040002` | `1060002` | `1072001` |
| female | `1041002` | `1061002` | `1072001` |

Weapon (prior derivation): sourced from the same job-advancement NPC scripts
used for the stat floors above — each carries an `award_item` operation for
the class-specific starter weapon immediately alongside its `change_job`, and
`String.wz/Eqp.img.xml` names each item id:

| ordinal | class | weapon item id(s) | name (`String.wz/Eqp.img.xml`) |
|---|---|---|---|
| 0 | Warrior | `1302077` | "Beginner Warrior's sword" |
| 1 | Magician | `1372043` | "Beginner Magician's wand" |
| 2 | Bowman | `1452051` | "Beginner Bowman's bow" |
| 3 | Thief | `1332063` | "Beginner Thief's short sword" |
| 4 | Pirate | `1482000` and `1492000` — both, per USER RULING (§5) | "Steel Knuckler" (`1482000`) and "Pistol" (`1492000`); both awarded by `npc-1090000.json` since 1st-job Pirate in this era has not yet branched into Brawler/Gunslinger |

Weapon ids are **not gendered** in this data (unlike faces/hairs/tops/bottoms) —
the same weapon id(s) apply to both genders for a given class. The `mapleLife`
schema's `equipment` field is a list (`Equipment []EquipmentEntry`), so
carrying two weapon entries for Pirate is not a schema change — see §5.

### `inventory` + `meso` — the fixed package

Per §11 A2, from the MapleSEA guide (a ruling, not independently re-derived
from WZ this pass — the guide's text is the source cited in `design.md` §11,
and this task's own item-id resolution closes the ids the guide's prose
could not carry):

| item | template id | source |
|---|---|---|
| White Potion ×100 | `2000002` | `<local-wz-root>/Item.wz/Consume/0200.img.xml` node `02000002`; name confirmed via `String.wz/Consume.img.xml` (`imgdir 2000016` era duplicate aside — `02000002`'s `price=160` matches White Potion's known-cheap price point) |
| Mana Elixir ×100 | `2000006` | same file, node `02000006`, `price=310` |
| The Relaxer ×1 | `3010000` | `<local-wz-root>/Item.wz/Install/0301.img.xml` node `03010000`; name confirmed via `String.wz/Ins.img.xml` node `3010000` = "The Relaxer" |
| Meso | `100,000` | §11 A2, the guide's stated fixed amount |

Same package for all five classes and both genders (the guide states this is
a flat, class-independent package).

---

## §4 — UNRESOLVED / UNCONFIRMED (explicit list)

This section is a first-class deliverable, not a footnote. Everything below
is a genuine gap this pass could not close within its bounded budget, or a
value with more than one candidate and no tiebreaker found. Nothing in §1-§3
above stands in for these; where a table cell says UNCONFIRMED, that is the
authoritative status of that cell.

**Closed by the user's rulings this pass (§5):** `hp`/`mp` per class (now the
sourced midpoint, plus the SP-skill's per-level table for Warrior/Magician —
no longer UNCONFIRMED), `mapId` for all five classes (now the class's town
map, sourced from `libs/atlas-constants`/`String.wz`), and Pirate's
`equipment.weapon` (now both `1482000` and `1492000`, per the schema's
existing list field). These three items are removed from this list; see §5
for the closing derivation. §3's tables have been edited in place so no
stale UNCONFIRMED survives for any of them.

1. **Class ordinal order for 2/3/4 (Bowman/Thief/Pirate) — UNCONFIRMED.**
   §2. The definitive source (`CUICharacterSaleDlg::OnCreate`, gms_v95
   `0x77adc0`) is a ~125,000-character decompile that this pass judged too
   large to fully read within budget; a narrower `search_text` sweep for
   `StringPool::GetBSTR` calls in its range returned 51 hits, too many to
   individually disambiguate in a bounded pass. Shipped value: the guide's
   point-5 ordering (Bowman, Thief, Pirate), per §11 A6's explicit ruling that
   this is acceptable and that a wrong order is a seed-data fix. **Pin before
   live testing** — either continue the `OnCreate` IDA pass (the brief's own
   suggested path) or read the received ordinal from channel logs while
   picking each class in a real client.

2. **Top/bottom/shoes equipment is not per-class — a synthesis, not a direct
   read.** §3. No per-class (as opposed to per-gender-generic) WZ resource
   for starting top/bottom/shoes was found. The recorded values are the
   generic Beginner-creation option lists' first entry, reused across all
   five classes — this is a documented inference bridging two separately
   cited sources (§1's per-gender option lists + the class-specific weapon
   from job-advancement data), not a single WZ node that names "Warrior's
   starting shoes." Flagged so Task 20/21 does not read it as a more direct
   citation than it is.

3. **Gender is not player-selectable per the guide, yet the wire carries
   `m_nGender` and the client's `OnButtonClicked` toggles it on a control** —
   carried forward from §11 A9, not re-investigated this pass; out of this
   task's scope (Task 16 is content values, not the gender-handling design
   decision).

---

## §5 — user-ruled values and the SP-skill HP/MP contract

Task 16's first pass produced four open items; the user has since ruled on
three. This section turns each ruling into a sourced value and, for the
HP/MP ruling, into the two separable pieces Task 20 (seeded config) and
Task 22 (submit-time computation) each need. Nothing here is a fresh
derivation of §1-§3's already-closed material — only the four items §4
flagged.

### 5.1 — Starting maps: the class's town map

**User ruling:** the starting map is the class's town map. Sourced from
`libs/atlas-constants/map/constants.go` first (repo constant, preferred over
a WZ read), and independently cross-checked against
`<local-wz-root>/String.wz/Map.img.xml`'s `mapDesc` strings, each of which
literally states "you can choose to become a &lt;class&gt; here" for the four
overworld towns and the equivalent phrasing for Nautilus:

| ordinal | class | town | `mapId` | `libs/atlas-constants` source | `String.wz/Map.img.xml` confirmation |
|---|---|---|---|---|---|
| 0 | Warrior | Perion | `102000000` | `map/constants.go:158` (`VictoriaRoadPerionId`) | node `102000000`: `mapName="Perion"`, `mapDesc="It's a warrior town located at the high mountainous area, and you can choose to become a warrior here."` |
| 1 | Magician | Ellinia | `101000000` | `map/constants.go:99` (`VictoriaRoadElliniaId`) | node `101000000`: `mapName="Ellinia"`, `mapDesc="It's a magician town surrounded by the forest, and you can choose to become a magician here."` |
| 2 | Bowman | Henesys | `100000000` | `map/constants.go:42` (`VictoriaRoadHenesysId`) | node `100000000`: `mapName="Henesys"`, `mapDesc="It's a bowman town on a wide prairie, and you can choose to become a bowman here."` |
| 3 | Thief | Kerning City | `103000000` | `map/constants.go:183` (`VictoriaRoadKerningCityId`) | node `103000000`: `mapName="Kerning City"`, `mapDesc="It's a thief town in the middle of the city where the sun sets. You can choose to become a thief here."` |
| 4 | Pirate | Nautilus Harbor | `120000000` | `map/constants.go:521` (`VictoriaRoadNautilusHarborId`) | node `120000000`: `mapName="Nautilus Harbor"`, `mapDesc="The harbor where Nautilus is at anchor. One will be able to become a dauntless pirate here."` |

**Nautilus disambiguation, recorded per the brief's instruction.** The
Nautilus ship has several sub-maps under `120000xxx`
(`TheNautilusTopFloorHallwayId 120000100`, `TheNautilusNavigationRoomId
120000101`, `TheNautilusLordJonathanSRoomId 120000102`,
`TheNautilusCafeteriaId 120000103`, two training rooms, a mid-floor hallway
and conference room/bedroom, a bottom-floor hallway and generator room) —
these are the same two candidates (`120000101`, plus the JMS-suspect
`912010200`) the first pass's job-advancement-NPC sweep found for Pirate
without a tiebreaker. `120000000` (Nautilus Harbor) is not one of the
interior rooms and is not the NPC's own room; it is the harbor exterior the
ship is docked at. It is taken as the town/spawn node — not an interior or a
quest variant — because its `mapDesc` is the only Nautilus-family string that
carries the exact "become a &lt;class&gt; here" creation-town phrasing found
on the other four towns (the Navigation Room's own `mapDesc`, by contrast, is
flavor text about the ship's captain, Kyrin, with no creation-related
language). This is a single unambiguous town map on the v83 tree, not a
second UNCONFIRMED multi-candidate case — the prior pass's ambiguity was
between two *interior* candidates; the harbor exterior was not among them and
is the one the `mapDesc` text actually names as the creation point.

This **replaces** §3's prior `mapId` candidate-sweep entirely, including the
task-16 review's Finding 1 (Warrior's "single match" claim not surviving a
full `Map.wz` sweep) — that finding is moot now, since the value is no
longer derived from an NPC-proximity sweep at all.

### 5.2 — Pirate equipment: award both weapons

**User ruling:** award both weapons. `1482000` (Steel Knuckler) and
`1492000` (Pistol) both go in the Pirate class entry's `equipment` list. Both
ids were already cited in §3 (from `npc-1090000.json`'s two `award_item`
operations, names confirmed via `String.wz/Eqp.img.xml`) — this ruling
closes which of the two candidates ships, not their sourcing, so neither id
is re-derived here. This matches this repo's own job-advancement seed script
(`deploy/seed/gms/95_1/npc-conversations/npc/npc-1090000.json`), which
already awards both to a live 1st-job Pirate. The `mapleLife` schema's
`equipment` field is `Equipment []EquipmentEntry` (a list), so carrying two
weapon entries for one class is not a schema change.

### 5.3 — HP/MP: the midpoint, and the SP-skill's own contribution

**User ruling:** midpoint of the interval, "but do keep in mind the hp/mp
increase skills of warrior and magician, which will be leveled, in that
calculation." As the brief frames it, this does not collapse into one static
config number: the player picks the skill's level (`nSP` ∈ `0..10`) at
submit time, so the seeded `StatBlock.Hp`/`Mp` must stay skill-excluded and
Task 22 must add the skill's own contribution on top, computed from the
player's chosen level. Both halves below are separable and sourced
independently.

#### (a) The base midpoint, per class, skill-excluded — what Task 20 seeds

Midpoint of §3's already-recorded `50 + 29×[hpLower,hpUpper]` /
`5 + 29×[mpLower,mpUpper]` interval, computed as `base + 29×avg(lower,
upper)`:

| ordinal | class | hp midpoint | mp midpoint |
|---|---|---|---|
| 0 | Warrior | `50 + 29×26 = 804` (avg of 24,28 = 26) | `5 + 29×5 = 150` (avg of 4,6 = 5) |
| 1 | Magician | `50 + 29×12 = 398` (avg of 10,14 = 12) | `5 + 29×23 = 672` (avg of 22,24 = 23) |
| 2 | Bowman | `50 + 29×22 = 688` (avg of 20,24 = 22) | `5 + 29×15 = 440` (avg of 14,16 = 15) |
| 3 | Thief | `50 + 29×22 = 688` (avg of 20,24 = 22) | `5 + 29×15 = 440` (avg of 14,16 = 15) |
| 4 | Pirate | `50 + 29×25 = 775` (avg of 22,28 = 25) | `5 + 29×20.5 = 599.5` (avg of 18,23 = 20.5) |

Pirate's mp midpoint is a genuine half-integer, because its recorded
mp interval (`18..23`) has an odd span (`23 − 18 = 5`); `StatBlock.Mp` is an
integer field, so Task 20 must round `599.5` one direction. Which direction
is a policy choice for Task 20, the same class of choice the first pass
already deferred for the pre-ruling interval — not a fact this pass can
resolve, and not large enough (±0.5 MP) to warrant blocking on it.

These five rows are the values Task 20 seeds into `StatBlock.Hp`/`Mp` for
all five classes; they do **not** include the Warrior/Magician SP-skill's
contribution — that is (b) below, and Task 22's job, not Task 20's.

#### (b) The SP-skill's per-level contribution — what Task 22 computes at submit time

Sourced from `<local-wz-root>/Skill.wz/100.img.xml` node `1000001` (Warrior,
`WarriorImprovedMaxHpIncreaseId`) and `<local-wz-root>/Skill.wz/200.img.xml`
node `2000001` (Magician, `MagicianImprovedMaxMpIncreaseId`) — both are the
same two ids §3 already cites for `spSkillId`. Each `level/N` node carries
two raw ints, `x` and `y`, for `N` in `1..10` (there is no `level/0` node —
per this game's universal skill-WZ convention, an unlearned skill sits at
level 0 with no data node and contributes 0; this is not read from an
explicit WZ node, it is the standard convention every other skill in this
tree also follows):

| skill level | Warrior `1000001` `x` | Warrior `1000001` `y` | Magician `2000001` `x` | Magician `2000001` `y` |
|---|---|---|---|---|
| 0 (not learned) | 0 | 0 | 0 | 0 |
| 1 | 4 | 3 | 2 | 1 |
| 2 | 8 | 6 | 4 | 2 |
| 3 | 12 | 9 | 6 | 3 |
| 4 | 16 | 12 | 8 | 4 |
| 5 | 20 | 15 | 10 | 5 |
| 6 | 24 | 18 | 12 | 6 |
| 7 | 28 | 21 | 14 | 7 |
| 8 | 32 | 24 | 16 | 8 |
| 9 | 36 | 27 | 18 | 9 |
| 10 | 40 | 30 | 20 | 10 |

Both tables are exact linear formulas in the raw WZ data (`x = 4L`, `y = 3L`
for Warrior; `x = 2L`, `y = L` for Magician), recorded as a formula here only
because the WZ data itself is one — not collapsed by this pass.

**Which field this task's HP/MP interval needs, established from this
repo's own live code, not guessed from the WZ field names:**
`services/atlas-character/atlas.com/character/character/processor.go`'s
`data/skill/effect.Model.X()`/`.Y()` read straight off the WZ node's `x`/`y`
ints (`services/atlas-character/atlas.com/character/data/skill/effect/model.go:68-74`).
Two different call sites use them for two different mechanics:

- `resolveHPMPGainParams` (`processor.go:1721-1822`) sets
  `params.hpBonus = se.X()` (Warrior, `processor.go:1810`) /
  `params.mpBonus = se.X()` (Magician, `processor.go:1817`) — this is the
  **per-level-up gain bonus**, added inside `rollHPMPGain`
  (`processor.go:1823-1830`) to the random `hpLower..hpUpper`/
  `mpLower..mpUpper` roll §3 already uses for the skill-excluded interval.
  This is the field the level 1→30 HP/MP total needs, and both classes agree
  on it (`x`, not `y`).
- `getMaxHpGrowth`/`getMaxMpGrowth` (`processor.go:1180-1247`, `1249-1324`)
  are the **per-AP-point-invested bonus**, applied only when a player
  manually allocates one AP into HP/MP via `RequestDistributeAp` — not
  relevant here, since Maple Life's §3 `stats` table raises only the class's
  floor stat (str/int/dex) and leaves the remainder as unspent `ap`, so no AP
  is invested into HP/MP at creation. Recorded for completeness, this
  repo's own code is **not symmetric** between the two: `getMaxHpGrowth`
  reads `se.Y()` (`processor.go:1243`) for the Warrior AP-invested case, but
  `getMaxMpGrowth` reads `se.X()` (`processor.go:1319`, the *same* field the
  level-up path already uses) for the Magician AP-invested case — a genuine
  asymmetry in the live code, quoted as found, not resolved or "fixed" here
  since it is out of this docs-only task's scope and does not affect the
  conclusion below (both consumers of the level-1→30 total agree it is `x`).

So the table Task 22 needs is **`x` only**: Warrior `x = 4×nSP`, Magician
`x = 2×nSP`.

**How the two combine — sourced from `ProcessLevelChange`'s own control
flow, not invented.** `ProcessLevelChange` (`processor.go:1579-1679`) calls
`p.resolveHPMPGainParams(c)` **once**, before the level-up loop, reading the
character's skill level at that single point in time
(`c.GetSkillLevel(uint32(improvingHPSkillId))`,
`processor.go:1807`/`1814`). The resulting `hpMPParams` (with the resolved
`hpBonus`/`mpBonus` already baked in) is then reused **unchanged** for every
iteration of `for i := range amount` (`processor.go:1602-1624`) —
`rollHPMPGain(hpMPParams)` is called once per level-up inside that loop, and
`hpMPParams` never changes inside it. In other words: within a single batch
of level-ups processed by one call, the code applies the *currently known*
skill level's bonus **uniformly to every level in that batch** — it does not
distinguish "levels already gained before the skill was learned" from
"levels gained after," because the skill level is resolved once, up front,
for the whole batch.

Maple Life's level-1→30 history is not a sequence of separate real
`ProcessLevelChange` calls over time — it is a single synthetic construction
at creation time, with the player's `nSP` fixed before any of the 29
level-ups is computed. That is the same shape as one `ProcessLevelChange`
call with `amount = 29`: one skill-level resolution, reused across the whole
batch. So the sourced-consistent combining rule — mirroring this repo's own
single-batch behavior rather than inventing a new one — is:

```
total HP (Warrior)   = base midpoint (5.3a) + 29 × x(nSP)   = 804 + 29×4×nSP  = 804 + 116×nSP
total MP (Magician)  = base midpoint (5.3a) + 29 × x(nSP)   = 672 + 29×2×nSP  = 672 + 58×nSP
```

for `nSP ∈ 0..10` (at `nSP = 0` this reduces to exactly the base midpoint,
consistent with "no SP invested → no skill contribution"). At `nSP = 10`:
Warrior HP = `804 + 1160 = 1964`; Magician MP = `672 + 580 = 1252`.

**What this does and does not settle, stated explicitly per the brief's
instruction not to invent a combining rule.** This *is* sourced, decisively,
for the shape Maple Life actually needs — a single synthetic construction
with the skill level fixed before the history is computed, exactly mirroring
`ProcessLevelChange`'s own single-resolution-per-batch behavior. What it does
**not** settle, and what this pass makes no claim about, is the general
live-gameplay question of whether *learning or raising* the skill mid a real
character's progression retroactively re-grants the bonus for levels already
banked in *earlier, separate* `ProcessLevelChange` calls — the code shows
each such call resolves the skill level fresh and only affects the levels
processed in *that* call, so previously-banked `addedHP`/`addedMP` from
earlier calls is never revisited. That question does not arise for Maple
Life's single-batch creation and is out of this task's scope.
