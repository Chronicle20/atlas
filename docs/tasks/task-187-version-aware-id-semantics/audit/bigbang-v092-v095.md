# The Big Bang v0.92 ↔ v0.95 reorg

Provisioned columns: **gms 92** (v0.92, "New Formulas" — Big Bang begins)
and **gms 95** (v0.95, "Mechanics" release; Resistance already released at
the intervening non-provisioned v0.94).

## Method

Full `GET /api/data/jobs?page[size]=200` drains for gms_92
(tenant `db1dbfb3-4345-4731-9223-c40b0c7f6457`) and gms_95
(tenant `c794c706-aea3-4882-90a6-a3b7ee314f52`) were diffed programmatically
(exact skill-id-set per job, both directions) to find every job whose
skill-id membership changed between the two versions. This is the
"multi-boundary" reorg the task-1 brief calls "genuinely hard" — it affects
far more of the job tree than the single-class boundaries in
`v048-v062.md`.

## Headline finding: the job-id set itself did not change

Both v92 and v95 have **exactly 101 jobs**, and the job-id sets are
identical (`set(jobs_v92) == set(jobs_v95)`, confirmed by diff — zero jobs
added or removed). Big Bang did **not** renumber or remove any job wireId
in this window. What changed is the **skill-id membership within existing
jobs** — individual skill wireIds were added to / removed from a job's
learnable-skill set.

## Full scope: 56 of 101 jobs show skill-id churn

The complete list of affected jobs (job id → removed skill ids, added
skill ids), grounded directly in the two `GET /api/data/jobs` drains:

```
100: removed [1000000,1000001,1000002] added [1000006]
110: removed [1100001,1100003,1101005] added [1100009,1101008]
111: removed [1110001,1111004,1111006] added [1110009,1111010]
112: removed [1120005] added [1120012]
120: removed [1200001,1200003,1201005] added [1200009,1201008]
121: removed [1210000,1211003,1211005,1211007] added [1211010,1211011]
122: removed [1221001,1221003] added [1220013]
130: removed [1300001,1300003,1301005] added [1300009,1301008]
131: removed [1311002,1311004] added [1310009]
132: removed [] added [1320011]
200: removed [2000000,2000001] added [2000006]
210: removed [] added [2100006]
211: removed [] added [2111007,2111008]
212: removed [] added [2120009]
220: removed [] added [2200006]
221: removed [] added [2211007,2211008]
222: removed [] added [2220009]
230: removed [] added [2300006]
231: removed [] added [2310008,2311007]
232: removed [] added [2320010]
300: removed [3000000] added []
310: removed [] added [3100006]
311: removed [] added [3110007]
312: removed [] added [3120010,3120011]
320: removed [] added [3200006]
321: removed [] added [3210007]
322: removed [] added [3220009,3220010]
410: removed [4100002] added [4100006]
411: removed [] added [4111007]
412: removed [] added [4120010]
420: removed [4200001] added [4200006]
421: removed [] added [4211007,4211008,4211009]
422: removed [] added [4220009]
431: removed [4310000] added [4310004]
510: removed [5100000] added [5100008,5100009]
511: removed [] added [5110008,5111007]
512: removed [] added [5120011]
520: removed [] added [5200007]
521: removed [] added [5211007]
522: removed [] added [5220012]
1100: removed [11000000] added [11000005]
1200: removed [12000000] added [12000005]
1210: removed [] added [12100007]
1510: removed [15100000] added [15100007]
3000: removed [30001065] added [30001024,30001035,30001036,30001037,30001038,30001039,30001040,30001061,30001062,30001063,30001064,30001068,30001069]
3200: removed [32001004,32001005,32001006] added []
3210: removed [32100000] added [32100006,32101000]
3211: removed [32110002,32111012,32111013,32111014] added [32111002]
3212: removed [32120002,32121009,32121010,32121011,32121012,32121013,32121014] added [32120009,32121002]
3300: removed [33001004,33001005] added []
3310: removed [] added [33100009]
3312: removed [33121003] added [33120010]
3500: removed [35001000] added []
3510: removed [35101001,35101002] added [35100008,35101007,35101009,35101010]
3511: removed [35110000,35110008,35111003,35111006,35111007,35111012] added [35110014,35111013,35111015]
3512: removed [35120002,35121004] added [35121012,35121013]
```

The `1xxx`, `2xxx` (excl. 2000), `3xxx` families beyond `3000` are Cygnus
Knight and other class families this audit did not independently
name-identify (job labels in this document are wireIds only, not asserted
real-world class names, except where I pulled skill names directly — see
below). This full table is itself grounded, machine-derived evidence
(no invented rows) and is the raw material Task 3/4 will need for a
complete generator; **not every row below has an individually verified
name-level classification** — see "Coverage and gaps" at the end.

## Classified sample (name + effect evidence pulled per identity)

For the jobs below I additionally pulled `GET /api/data/skills/{id}`
(name, description, `maxLevel`, `effects` length) on both sides of the
boundary. FR-1.3 / OQ-4 require classifying each affected identity as
**same-identity rename**, **merge**, or **no-counterpart**. The evidence
below supports those three plus one pattern the brief's taxonomy doesn't
name — a **1→N split** — which I report explicitly rather than force into
"merge."

### MERGE (N old skills → 1 new skill)

- **job 100** (Warrior 1st job tier): `1000000`="Improved HP Recovery"
  (maxLevel 16), `1000001`="Improved MaxHP Increase" (maxLevel 10),
  `1000002`="Endure" (maxLevel 8) — all three removed at v95, replaced by
  a single `1000006`="HP Boost" at v95.
- **job 200** (Magician 1st job tier): `2000000`="Improved MP Recovery",
  `2000001`="Improved MaxMP Increase" — both removed, replaced by
  `2000006`="MP Boost" at v95. Mirrors job 100's pattern one tier over for
  the Magician branch.

### MERGE (N old skills → 2 new skills)

- **job 110** (Fighter, per weapon-mastery naming): `1100001`="Axe Mastery"
  (maxLevel 20), `1100003`="Final Attack : Axe" (maxLevel 30) removed →
  `1100009`="Enhanced Basics" added; `1101005`="Axe Booster" removed →
  `1101008`="Ground Smash" added.

### RENAME (1:1, same-identity)

- **job 410 / job 420** (Thief 2nd-job branches): `4100002`/`4200001` =
  "Endure" (maxLevel 20) → `4100006`/`4200006` = "Shadow Resistance" at
  v95. Same slot count, same functional role by description ("permanently
  increased" survivability passive), classified as a rename.
- **job 1100 / job 1200** (a Cygnus branch — wireId not independently
  name-identified beyond the skill evidence itself): `11000000`="Max HP
  Enhancement" → `11000005`="HP Boost"; `12000000`="Increasing Max MP." →
  `12000005`="MP Boost". Same rename pattern as the Explorer HP/MP-Boost
  consolidation, extended to this job family.

### SPLIT (1 old skill → 2 new skills — not covered by the brief's 3-way taxonomy)

- **job 510** (Pirate 2nd job — Infighter/Brawler, confirmed identity per
  `v048-v062.md`): `5100000`="Improve MaxHP" (maxLevel 10) removed,
  replaced by **two** new skills: `5100008`="Critical Punch" and
  `5100009`="HP Boost". This is a genuine 1→2 split, not a merge — flagged
  as a taxonomy gap for Task 2's generator to handle explicitly rather than
  force-fit into "merge."

### NO-COUNTERPART (pure removal)

- **job 300** (Bowman 1st job tier): `3000000`="The Blessing of Amazon"
  (maxLevel 16) removed at v95, **with no replacement skill id added to
  job 300's set**. This is the clean no-counterpart case the brief asks
  for: recorded as absent at v95, not guessed at.

### UNVERIFIED (evidence insufficient to classify with confidence)

- **job 112** (Warrior 4th job tier — Hero, per standard job-tree
  convention, not independently WZ-confirmed by name in this audit):
  `1120005`="Guardian" (maxLevel 30, full 30-level effect data: block
  chance vs. melee attacks, stun-on-block) removed at v95, replaced by
  `1120012`="Combat Mastery" (description: "Ignores a portion of a
  monster's defense while attacking"). **The names and descriptions do not
  correspond** ("Guardian" is a block/stun defensive skill; "Combat
  Mastery" is a defense-ignore offensive passive) — this does not read as
  a functional rename of the same mechanic. I do not have enough evidence
  to assert rename, merge, or no-counterpart here and am flagging it
  **UNVERIFIED / escalated** rather than guessing. Task 2/3 should pull
  the raw WZ Skill.wz XML or IDA cross-reference for job 112's skill list
  before generating a binding for this pair.

## A systematic anomaly across every v95 replacement skill checked

**Every** v95-side replacement skill queried above — 11 distinct wireIds
across the 10 classified jobs (`1000006`="HP Boost", `2000006`="MP Boost",
`1100009`="Enhanced Basics", `1101008`="Ground Smash",
`4100006`="Shadow Resistance", `4200006`="Shadow Resistance",
`5100008`="Critical Punch", `5100009`="HP Boost", `11000005`="HP Boost",
`12000005`="MP Boost", `1120012`="Combat Mastery" — 7 distinct names, of
which "HP Boost" recurs at 3 wireIds and "MP Boost" / "Shadow Resistance"
each recur at 2 — all 11 named here) — returns `maxLevel: 0` and
`effects: []` from `GET /api/data/skills/{id}` against the gms_95 tenant —
i.e. the WZ record resolves a name and description but **no per-level
effect data** is present in this baseline, in contrast to every v92-side
skill checked, which has full leveled `effects` arrays (e.g. `1120005`
"Guardian" has 30 populated effect levels).

This is consistent across all 11 independently-checked replacement
skills (`divergences.csv` carries the corresponding 11 `maxLevel=0`-annotated
rows), so it reads as a systematic characteristic of how this particular
v95 baseline's Skill.wz was ingested for Big-Bang-introduced skills, not
as noise on any one row. Two explanations are plausible and **neither is
confirmed** here:
(a) an atlas-data WZ-ingestion gap specific to this baseline/version, or
(b) these skills' effect data is genuinely defined via formula/reference
elsewhere in the client rather than per-level WZ nodes at this point in
Big Bang's rollout. I am not asserting either — this is flagged as an open
question for whoever consumes this baseline next (Task 3, or a
re-ingestion of the gms_95 WZ files), not resolved by this audit.

## Coverage and gaps

Of the 56 affected jobs, **10** have individually pulled name/effect
evidence and a stated classification above (100, 110, 200, 300, 112, 410,
420, 510, 1100, 1200 — 10 distinct jobs covering 6 classification
outcomes: 2× merge-to-1, 1× merge-to-2, 4× rename (410, 420, 1100, 1200 —
see the RENAME section above), 1× split, 1× no-counterpart, 1×
unverified). The remaining jobs in the full diff table
(111, 120-122, 130-132, 210-232 excl. 200, 310-322, 411-422 excl. 410/420,
431, 511-522 excl. 510, 1210, 1510, and the entire `3xxx` family) are
recorded as **raw wireId-diff evidence only** — I did not individually pull
skill names/effects for each, and per CLAUDE.md's no-invention policy I am
not classifying them without that evidence. This is a bounded, explicit
scope decision: the raw diff itself is complete and grounded (not a
placeholder), and it is exactly the input Task 3/4's generator will need
to extend this classification exercise. `divergences.csv` contains rows only for these 10 individually-classified
jobs' skill pairs from this Big Bang diff — it does not contain unverified
rows for the remaining 46 jobs, consistent with the validator's "every row
has non-empty, real evidence" requirement. (Job 430's Dual Blade
release-boundary rows in `divergences.csv` are a separate finding — the
v87→v92 release boundary, not this v92→v95 Big Bang skill-slot churn; job
431's own Big Bang skill change, `4310000`→`4310004`, appears only in the
raw diff table above and was not individually name-verified.)
