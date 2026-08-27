# bug: Maple Life AP/SP disagree with the client's own preview, and starting equipment should follow HeavenMS/Cosmic

Task: task-246-maple-life-character-creation
PR: atlas-pr-1466
Client/tenant under test: **gms_v83** (user-confirmed). The `mapleLife` block is
byte-identical across `template_gms_83_1.json`, `template_gms_87_1.json`,
`template_gms_92_1.json` and `template_gms_95_1.json` (verified by dumping all
four), so the defect and its fix are version-independent.

## Reproduced

User ran the Maple Life flow live on gms_v83 and created a class whose displayed
primary stat was **DEX 20** — that is ordinal 4 (Pirate, `4/20/4/4`); ordinal 1
(Magician, `4/4/20/4`) has the same stat sum and the same predicted AP, so the
observation is unambiguous either way.

## Observed

- The client's pre-submit preview promised **138 unused AP** and **61 SP**
  (with every skill blank).
- The created character received **129 unused AP** and **87 SP**.

Both received values come straight from the seed: `mapleLife.classes[].ap` and
`mapleLife.classes[].sp` (`"87,0,0,0,0,0,0,0,0,0"`), projected onto the preset
by `toPreset`
(`services/atlas-character-factory/atlas.com/character-factory/factory/maple_life.go:104-112`).

## Expected

The server must award what the client's dialog promises: AP `170 − Σstats` per
class, SP `61` for all five.

## Root cause

`docs/tasks/task-246-maple-life-character-creation/maple-life-content.md` §3
("`ap` / `sp` — the accumulated totals and the arithmetic") models a level-30
Maple Life character as *29 level-ups at the advanced job*, and nothing else:

- **AP** — it takes `total AP by level 30 = 29 × 5 = 145` and subtracts the
  stat-floor spend. It omits the **25-point character-creation stat pool**. A
  level-1 MapleStory character has `Σstats = 25`, not the `4/4/4/4 = 16` that
  `ResetStats`' `baseStat` leaves behind; the missing **9 points** are exactly
  the shortfall the user saw (138 − 129 = 9). Correct total pool at level 30 is
  `25 + 5 × 29 = 170`, and the per-class unspent AP is `170 − Σstats`.
- **SP** — it takes `total SP by level 30 = 29 × 3 = 87`, i.e. it charges 3 SP
  for every level from 1. This contradicts this repo's own level-up code:
  `computeOnLevelAddedSP`
  (`services/atlas-character/atlas.com/character/character/processor.go:1741-1749`)
  returns 3 only for a non-Beginner job, and a real character is a Beginner
  until the 1st-job advancement at level 10. The client's 61 is
  `1 (1st-job advancement grant) + 3 × 20 (level-ups 11→30)`. The `+1` is
  Nexon/Cosmic behaviour (`Cosmic/src/main/java/client/Character.java:1154`,
  `int spGain = 1` in `changeJob`); Atlas's own `ChangeJob`
  (`processor.go:592-614`) does **not** grant it, which is why no Atlas-internal
  model reproduces 61. Per user ruling, the client's number is the contract.

Both corrected formulas reproduce the two independently observed client values
exactly, from a single coherent model.

Separately, the user has ruled that the seeded starting equipment should follow
the HeavenMS/Cosmic per-class tables rather than the generic
Beginner-creation apparel (`1040002/1060002/1072001` etc.) currently seeded.

## Fix

All four templates carry an identical `mapleLife.classes[]` array of 10 entries
(5 ordinals × 2 genders). Apply the same edit to each file.

### Files

- `services/atlas-configurations/seed-data/templates/template_gms_83_1.json`
- `services/atlas-configurations/seed-data/templates/template_gms_87_1.json`
- `services/atlas-configurations/seed-data/templates/template_gms_92_1.json`
- `services/atlas-configurations/seed-data/templates/template_gms_95_1.json`
- `docs/tasks/task-246-maple-life-character-creation/maple-life-content.md`
  (§3 `ap`/`sp` arithmetic, §3 `stats` table's `ap` column, and §3 `equipment`)

No Go source changes are required: `toPreset` already passes `e.AP` and
`e.SP` through unmodified, and `equipment` is already a list (§5.2 shipped two
Pirate weapons in one entry). No Go test hard-codes the seeded `ap`/`sp`/
equipment values — verified by grepping `87,0,0,0,0,0,0,0,0,0` across
`services/` and `libs/` outside `seed-data/templates` (zero hits) and by
reviewing the 16 `*_test.go` files that mention Maple Life, all of which build
synthetic fixtures.

### 1. `ap` and `sp` per ordinal (both gender rows)

| ordinal | class | stats (str/dex/int/luk) | `ap` now → new | `sp` now → new |
|---|---|---|---|---|
| 0 | Warrior | 35/4/4/4 (Σ 47) | 114 → **123** | `"87,0,…"` → `"61,0,0,0,0,0,0,0,0,0"` |
| 1 | Magician | 4/4/20/4 (Σ 32) | 129 → **138** | → `"61,0,0,0,0,0,0,0,0,0"` |
| 2 | Bowman | 4/25/4/4 (Σ 37) | 124 → **133** | → `"61,0,0,0,0,0,0,0,0,0"` |
| 3 | Thief | 4/25/4/4 (Σ 37) | 124 → **133** | → `"61,0,0,0,0,0,0,0,0,0"` |
| 4 | Pirate | 4/20/4/4 (Σ 32) | 129 → **138** | → `"61,0,0,0,0,0,0,0,0,0"` |

`stats`, `hp`, `mp`, `level`, `mapId`, `meso` and `spSkillId` are **unchanged**.
`spendSPPool` (`maple_life.go:137-146`) continues to deduct the player's `nSP`
(0..10) from slot 0 of the new 61 pool; 61 ≥ 10 so no underflow.

### 2. `equipment` per ordinal, per gender

`useAverageStats: true` on every equipment entry, matching the existing rows.
Every listed weapon is awarded (user ruling; consistent with §5.2).

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

### 3. `inventory` per ordinal

Keep the existing package on every class — `2000002 ×100`, `2000006 ×100`,
`3010000 ×1` — and **add**:

- ordinal 3 (Thief): `2070000 ×500`
- ordinal 4 (Pirate): `2330000 ×800`

Ordinals 0, 1, 2 keep the existing three entries unchanged.

### 4. Documentation

`maple-life-content.md` §3 must be corrected, not merely appended to — it is
the document a future session will consult:

- §3 `ap` / `sp`: replace the `29 × 5 = 145` AP total with
  `25 + 5 × 29 = 170` and state the 25-point creation pool explicitly; replace
  the `29 × 3 = 87` SP total with `1 + 3 × 20 = 61`, citing
  `computeOnLevelAddedSP`'s Beginner branch and the 1st-job advancement grant.
- §3 `stats` table: update the `ap` column to 123/138/133/133/138.
- §3 `equipment`: replace the derived per-gender apparel/weapon table with the
  HeavenMS/Cosmic table above, marked as a user ruling (the same way §5.1/§5.2
  are marked), superseding the earlier derivation.

## Not yet answered

- **Only one class was observed live.** The user tested the DEX-20 class; the
  other four `ap` rows are the same formula extrapolated, not observed. The
  client's own per-class preview strings live in
  `CUICharacterSaleDlg::m_strInitStatDesc[5]` and `m_strInitSPDesc[5]`
  (gms_v95 struct offsets `0x280` / `0x294`, ctor `0x778270`) and resolve
  through the XOR-obfuscated `StringPool` (`ms_aString` table `0xc5a878`,
  `ms_aKey` `0xb98830`; the cipher is fully documented in `derivation.md` §1).
  Decoding that block would confirm all five rows and all five SP rows from the
  client itself. **Not done** — deliberately deferred as a cost call, not a
  blocker. Until then, confirm Warrior (expect 123) and Bowman/Thief (expect
  133) by live re-test after the fix lands.
- Atlas's `ChangeJob` does not grant the 1st-job advancement SP that Nexon and
  Cosmic both grant. Maple Life now hard-codes the resulting 61, but an
  ordinary Atlas character advancing at level 10 will still end level 30 with
  60 SP, not 61. That is a pre-existing divergence outside this task's scope
  and is recorded here so it is not rediscovered.

## Resolution

- **Fixed by** `47dc7bf00` — "fix(maple-life): correct AP/SP totals and starting
  equipment to match client". All four templates plus `maple-life-content.md`
  §3. No Go source changed.
- **Gate** — `tools/verify.sh --quick --base 42a238717`, exit 0, 6 changed
  paths; template opcode order, duplicate binding, movement types and operator
  cancel guards all passed. This is the `--quick` run and does **not** satisfy
  the flagless `verify.sh` bar; a full run is still owed before the PR.
- **Review** — `task-reviewer` APPROVED
  (`review-bug-ap-sp-and-starting-equipment.md`), 0 blocking, 0 non-blocking.
  Verified programmatically that all 10 rows in all four templates match the
  `## Fix` tables, that the four edits are byte-identical, and that
  `stats`/`hp`/`mp`/`level`/`mapId`/`meso`/`spSkillId`/`jobId`/`ordinal`/
  `gender` are untouched.
- **Live re-test — NOT YET DONE.** The fix has not been exercised against a
  running client. Confirm on gms_v83 that a created character receives the AP
  and SP the dialog promises, and cover Warrior (expect 123) and
  Bowman/Thief (expect 133) as well as the already-observed DEX-20 class
  (expect 138) — those three rows are still formula-extrapolated, per
  "Not yet answered" above. Also eyeball that the new equipment actually
  equips (the Warrior rows now carry four weapons in one `equipment` list).
