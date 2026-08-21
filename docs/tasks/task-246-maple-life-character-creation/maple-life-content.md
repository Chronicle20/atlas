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
| 3 | Thief | `400` (`RogueId`) | `libs/atlas-constants/job/constants.go:127` — this codebase's constant name for the Thief-track first job is `RogueId`; the wire/guide name is "Thief" |
| 4 | Pirate | `500` (`PirateId`) | `libs/atlas-constants/job/constants.go:130` |

### `level`

`30` for all five, per §11 A2's user ruling. Not derived; a ruling, restated
here as instructed.

### `ap` / `sp` — the accumulated totals and the arithmetic

**AP.** `computeOnLevelAddedAP(jobId, level)`
(`services/atlas-character/atlas.com/character/character/processor.go:1685-1697`)
returns a flat `5` for any non-Cygnus job — all five Maple Life classes are
non-Cygnus (`job.IsCygnus` checks `GetType(jobId) == TypeCygnus`, and
100/200/300/400/500 are all `TypeExplorer`). A level-30 character has gone
through 29 level-ups (level 1→2, 2→3, … 29→30), each granting AP by this
formula while already at the character's final job (Maple Life skips the
Beginner phase entirely — the character is created already-advanced, so every
level-up in this hypothetical history uses the advanced-job formula, not the
Beginner one):

```
total AP by level 30 = 29 × 5 = 145
```

**SP.** `computeOnLevelAddedSP(jobId, effectiveLevel)`
(`processor.go:1702-1709`) returns `3` for any job that is not
`job.IsBeginner` — none of the five Maple Life classes are Beginner-family
(`job.IsBeginner` checks membership in `{BeginnerId, NoblesseId, LegendId,
EvanId}`), so:

```
total SP by level 30 = 29 × 3 = 87
```

**`ap` (unspent remainder) per class** = `145 − (job's minimum-AP spend)`, where
the minimum-AP spend is `floor − 4` (the stat floor required for 1st-job
advancement, minus the `baseStat` of `4` the game-neutral other three stats
sit at — `ResetStats`, `processor.go:2178` — `rebalance_ap`'s own semantics:
zero all four primary stats to `4`, then raise the one required stat to its
floor, and the AP spent doing that redistribution is exactly `floor − 4`).

**`sp` per class** = `87` (the full accumulated total) for **every** class.
For ordinals 0/1 only, `nSP` (the player's 0..10 choice) is deducted from this
pool **at creation time by the submit-handling code**, per §11 A4 — that
deduction is a runtime computation on the player's wire value, not a content
value Task 16 can pre-compute, so `sp` here is recorded as the pre-deduction
total for all five classes.

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
fourth is raised to its floor. `ap` is `145 − spend`, restated from above.

| ordinal | class | str | dex | int | luk | `ap` (unspent) |
|---|---|---|---|---|---|---|
| 0 | Warrior | **35** | 4 | 4 | 4 | `145 − 31 = 114` |
| 1 | Magician | 4 | 4 | **20** | 4 | `145 − 16 = 129` |
| 2 | Bowman | 4 | **25** | 4 | 4 | `145 − 21 = 124` |
| 3 | Thief | 4 | **25** | 4 | 4 | `145 − 21 = 124` |
| 4 | Pirate | 4 | **20** | 4 | 4 | `145 − 16 = 129` |

### `hp` / `mp` — UNCONFIRMED, methodology stated

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

Task 20/21 must pick a deterministic policy inside this interval (e.g. the
lower bound, for content stability) — that policy choice belongs to whichever
task consumes this value, not to Task 16, since it is not a fact this pass
can resolve. See §4.

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

### `mapId` — UNCONFIRMED (candidate maps found, not disambiguated)

Attempted via each class's job-advancement NPC's home map, read from local
`<local-wz-root>/Map.wz/Map/` life data (`grep` for the NPC's numeric id
inside each map's `life` node):

| ordinal | class | NPC id | map candidate(s) found | confidence |
|---|---|---|---|---|
| 0 | Warrior | `1022000` | `102000003` (single match) | fair — one candidate |
| 1 | Magician | `1032001` | `101000003` (single match) | fair — one candidate |
| 2 | Bowman | `1012100` | `100000201` (single match) | fair — one candidate |
| 3 | Thief | `1052001` | `105100301`, `103000003` (two matches) | **UNCONFIRMED** — not disambiguated |
| 4 | Pirate | `1090000` | `120000101`, `912010200` (two matches) | **UNCONFIRMED** — `912010200` is very likely a JMS-only map (the `912` map-id prefix pattern is not a GMS Pirate-area convention seen elsewhere in this evidence) but this was not independently confirmed |

**Caveat that applies to all five, including the "fair" ones:** a
job-instructor NPC's home map is a reasonable proxy for "the class's starting
area" but was not independently confirmed to be the exact map ident a live
client would use as a Maple Life character's spawn point (as opposed to, say,
a nearby town square one tile of navigation away). Treat all five as
UNCONFIRMED for live-testing purposes; the single-candidate ones are lower
risk than the multi-candidate ones.

### `equipment` — top / bottom / shoes / weapon, per gender

**Top/bottom/shoes: not per-class.** §1 established that
`Etc/MakeCharInfo.img/Info/CharMale`/`CharFemale` slots 4-7 are the generic
Beginner-creation equipment option lists, shared by all classes (this is the
same resource `characters.templates[]`'s `jobIndex: 0` — Beginner — entry
already uses). No per-class top/bottom/shoes WZ resource was found this pass.
Recorded value: **the first (index-0) option in each per-gender list**, matching
the option that sits at the front of the client's own `ZArray<ASITEM>` before
any "next/previous" UI interaction (`GetSelectedAL` reads element 0 — see
`selected-al-derivation.md` Q1).

| gender | top | bottom | shoes |
|---|---|---|---|
| male | `1040002` | `1060002` | `1072001` |
| female | `1041002` | `1061002` | `1072001` |

**Weapon: per-class, resolved.** Sourced from the same job-advancement NPC
scripts used for the stat floors above — each carries an `award_item`
operation for the class-specific starter weapon immediately alongside its
`change_job`, and `String.wz/Eqp.img.xml` names each item id, confirming it is
the class's own beginner-tier weapon (not a generic item):

| ordinal | class | weapon item id | name (`String.wz/Eqp.img.xml`) |
|---|---|---|---|
| 0 | Warrior | `1302077` | "Beginner Warrior's sword" |
| 1 | Magician | `1372043` | "Beginner Magician's wand" |
| 2 | Bowman | `1452051` | "Beginner Bowman's bow" |
| 3 | Thief | `1332063` | "Beginner Thief's short sword" |
| 4 | Pirate | **UNCONFIRMED — two candidates, not disambiguated** | `1482000` "Steel Knuckler" or `1492000` "Pistol" (both awarded by `npc-1090000.json`; 1st-job Pirates in this era have not yet branched into Brawler/Gunslinger, so the job-advance script hands over both weapon types rather than one) |

Weapon ids are **not gendered** in this data (unlike faces/hairs/tops/bottoms) —
the same weapon id applies to both genders for a given class.

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

2. **`hp`/`mp` per class — UNCONFIRMED, interval only.** §3. The client's own
   gain formula is randomized per level-up; there is no deterministic
   WZ-sourced total. An interval (lower/upper bound across 29 level-ups) is
   recorded per class instead of a point value. **Task 20/21 must pick a
   policy** (e.g. lower bound) — that choice is downstream of this task, not
   a fact Task 16 can resolve.

3. **`mapId` for Thief and Pirate — UNCONFIRMED, ambiguous.** §3. Two map
   candidates were found for each and not disambiguated within budget
   (Thief: `105100301` vs `103000003`; Pirate: `120000101` vs `912010200`,
   the latter suspected-but-unconfirmed to be JMS-only). Warrior/Magician/Bowman
   have a single candidate each but were not independently confirmed to be
   the exact live-client spawn map (see the caveat in §3) — treat all five as
   needing live-client confirmation before shipping, with Thief/Pirate as the
   higher-risk two.

4. **Pirate's `equipment.weapon` — UNCONFIRMED, two candidates.** §3. The
   job-advancement script for Pirate awards both a Steel Knuckler (`1482000`)
   and a Pistol (`1492000`) rather than a single weapon, because 1st-job
   Pirate in this era has not yet branched into Brawler/Gunslinger. Task 20/21
   must decide whether the `mapleLife` config's single `weapon` field takes
   one of these (and which) or whether the schema needs to carry both.

5. **Top/bottom/shoes equipment is not per-class — a synthesis, not a direct
   read.** §3. No per-class (as opposed to per-gender-generic) WZ resource
   for starting top/bottom/shoes was found. The recorded values are the
   generic Beginner-creation option lists' first entry, reused across all
   five classes — this is a documented inference bridging two separately
   cited sources (§1's per-gender option lists + the class-specific weapon
   from job-advancement data), not a single WZ node that names "Warrior's
   starting shoes." Flagged so Task 20/21 does not read it as a more direct
   citation than it is.

6. **Gender is not player-selectable per the guide, yet the wire carries
   `m_nGender` and the client's `OnButtonClicked` toggles it on a control** —
   carried forward from §11 A9, not re-investigated this pass; out of this
   task's scope (Task 16 is content values, not the gender-handling design
   decision).
