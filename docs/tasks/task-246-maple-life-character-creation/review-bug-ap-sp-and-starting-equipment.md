# Review: bug-ap-sp-and-starting-equipment.md fix (commit 47dc7bf00)

Range reviewed: `42a238717..47dc7bf00` (single commit `47dc7bf00cc66132bf8ec3d505048e96621a3349`).
Requirement: `docs/tasks/task-246-maple-life-character-creation/bug-ap-sp-and-starting-equipment.md`.

## Scope

Content-only fix, no Go source changed (confirmed by `git diff --stat`):

```
docs/tasks/task-246-maple-life-character-creation/maple-life-content.md         | 112 +++++++++----
services/atlas-configurations/seed-data/templates/template_gms_83_1.json        | 184 +++++++++++++--------
services/atlas-configurations/seed-data/templates/template_gms_87_1.json        | 184 +++++++++++++--------
services/atlas-configurations/seed-data/templates/template_gms_92_1.json        | 184 +++++++++++++--------
services/atlas-configurations/seed-data/templates/template_gms_95_1.json        | 184 +++++++++++++--------
```

Matches the brief's "Files" list exactly — no extra files touched, no file from
the brief omitted.

## Field-by-field verification against the brief's `## Fix` tables

Extracted `mapleLife.classes[]` from all four template JSON files after the
change (Python/json, not text-matching) and compared programmatically against
brief tables §1 (ap/sp per ordinal, both genders), §2 (equipment per ordinal
per gender) and §3 (inventory additions for ordinals 3/4).

Result for `template_gms_83_1.json` (identical output independently reproduced
for `template_gms_87_1.json`, `template_gms_92_1.json`, `template_gms_95_1.json`):

```
ordinal gender ap  sp                          equipment                                                          inventory (added)
0       0      123 61,0,0,0,0,0,0,0,0,0  1040021,1060016,1072039,1302008,1442001,1422001,1312005          (none)
0       1      123 61,0,0,0,0,0,0,0,0,0  1051010,1072039,1302008,1442001,1422001,1312005                  (none)
1       0      138 61,0,0,0,0,0,0,0,0,0  1050003,1072075,1372003,1382017                                  (none)
1       1      138 61,0,0,0,0,0,0,0,0,0  1041041,1061034,1072075,1372003,1382017                          (none)
2       0      133 61,0,0,0,0,0,0,0,0,0  1040067,1060056,1072081,1452005,1462000                          (none)
2       1      133 61,0,0,0,0,0,0,0,0,0  1041054,1061050,1072081,1452005,1462000                          (none)
3       0      133 61,0,0,0,0,0,0,0,0,0  1040057,1060043,1072032,1472008,1332012                          2070000×500
3       1      133 61,0,0,0,0,0,0,0,0,0  1041047,1061043,1072032,1472008,1332012                          2070000×500
4       0      138 61,0,0,0,0,0,0,0,0,0  1052107,1072294,1482004,1492004                                  2330000×800
4       1      138 61,0,0,0,0,0,0,0,0,0  1052107,1072294,1482004,1492004                                  2330000×800
```

This matches the brief's `## Fix` §1, §2, §3 tables row for row, ordinal for
ordinal, gender for gender, across all four template files. No transposition,
no item id typo, no gender row left unedited, no ordinal skipped.

## Byte-identical fix applied to all four templates

The brief states the `mapleLife` block is byte-identical across all four
templates and instructs "apply the same edit to each file." Verified by
diffing each file's patch body with `@@` line-number headers stripped:

```
7a3cadd61c663ed37817ec1345e2b6c0  template_gms_83_1_bodynolineno.diff
7a3cadd61c663ed37817ec1345e2b6c0  template_gms_87_1_bodynolineno.diff
7a3cadd61c663ed37817ec1345e2b6c0  template_gms_92_1_bodynolineno.diff
7a3cadd61c663ed37817ec1345e2b6c0  template_gms_95_1_bodynolineno.diff
```

All four hashes identical — the edit was applied uniformly, no file drifted
from the other three.

## Protected fields untouched

Grepped every file's diff for `+`/`-` lines touching `"stats"`, `"hp"`,
`"mp"`, `"level"`, `"mapId"`, `"meso"`, `"spSkillId"`, `"jobId"`, `"ordinal"`,
`"gender"` keys — zero hits across all four files. Confirmed separately by
reading post-fix values directly (`template_gms_83_1.json`): `stats` per
ordinal are `35/4/4/4` (Warrior, Σ47), `4/4/20/4` (Magician, Σ32),
`4/25/4/4` (Bowman/Thief, Σ37), `4/20/4/4` (Pirate, Σ32) — exactly the Σstats
the brief's `ap` = `170 − Σstats` table assumes, and these sums reproduce
123/138/133/133/138 correctly (`170-47=123`, `170-32=138`, `170-37=133`,
`170-37=133`, `170-32=138`). `jobId`/`level`/`mapId`/`meso`/`spSkillId` also
confirmed unchanged and correctly paired to their ordinal (`jobId` 100/200/
300/400/500 for ordinals 0-4, `spSkillId` present only for ordinals 0/1 as
the pre-existing seed already had it, consistent with the brief's note that
`nSP` deduction at creation only applies to ordinals 0/1).

## Documentation (`maple-life-content.md` §3)

- AP arithmetic replaced: `25 + 5 × 29 = 170` (brief's exact formula),
  25-point creation pool stated explicitly, `ap = 170 − Σstats` per class.
- SP arithmetic replaced: `1 (1st-job advancement grant) + 3 × 20 (level-ups
  11→30) = 61`, citing `computeOnLevelAddedSP`'s Beginner branch and the
  Cosmic `changeJob` `+1` grant, matching the brief.
- `stats` table's `ap` column updated to `123 / 138 / 133 / 133 / 138` for
  ordinals 0-4, matching the brief's required sequence exactly.
- `equipment` section replaced with the HeavenMS/Cosmic per-class table,
  explicitly marked as a user ruling superseding the prior derivation, with
  the prior derivation retained for history (as the brief's item 4 requires:
  "corrected, not merely appended to").
- The old `145`/`87`/`124`/`129`/`114` figures are gone from the corrected
  section (verified — the `29 × 5 = 145` and `29 × 3 = 87` lines were replaced
  by `git diff`, not left alongside the new numbers), old numbers remain only
  inside the explicitly-labeled "prior derivation (superseded, retained for
  history)" subsection, which is intentional per the brief.

## Not evaluable (out of this unit's scope, not a defect in this commit)

- Live re-verification of the Warrior (123) and Bowman/Thief (133) figures
  against the client itself — the brief's own "Not yet answered" section
  explicitly defers this as a cost call, not a blocker for this commit.
- No Go source or Go test was changed by this commit (correctly, per the
  brief: "No Go source changes are required"). This review did not
  independently re-verify the brief's claim of zero grep hits for
  `87,0,0,0,0,0,0,0,0,0` outside `seed-data/templates`, since that check
  pertains to whether the fix could break something outside the diff's
  surface — it's not this diff's correctness, and the code paths it touches
  (`toPreset`, `spendSPPool`) are unchanged Go code the brief already traced.

## Verdict rationale

Every requirement in the brief's `## Fix` section is satisfied exactly:
ap/sp per ordinal and gender, equipment per ordinal and gender (including the
overalls exceptions for Warrior-female/Magician-male/Pirate and the
non-gender-split Pirate weapon list), inventory additions restricted to
ordinals 3 and 4, and the documentation corrected in place rather than
appended. All four template files received the identical edit. No protected
field was touched. No defect found.
