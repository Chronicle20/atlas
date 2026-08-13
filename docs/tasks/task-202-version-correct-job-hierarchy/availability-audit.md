# task-202 FR-2.3 — Release-class audit

A written per-(release class, version) verdict for every cell where
`docs/tasks/task-187-version-aware-id-semantics/audit/availability.csv`
claims `released=true`. Task 3's Cygnus-4th-job fix (`CygnusStage4`, commit
509bcc0) is the worked example this audit builds on, not the only finding it
produces — see [Finding: SuperGM at jms 185](#finding-supergm-at-jms-185-over-claimed)
below for the second one this pass turned up.

Verdicts:

- **CORRECT** — the class's WZ/live-baseline content matches the
  `released=true` claim; no over-claim.
- **OVER-CLAIMED** — the WZ/live baseline shows the claimed content does not
  exist; fixed in this task by the same mechanism Task 3 used.
- **UNVERIFIED** — no live evidence could be obtained; the blocker is named.

## Step 1 — the audit surface

```bash
awk -F, 'NR>1 && $6=="true" {print $5" "$1" "$2"."$3}' docs/tasks/task-187-version-aware-id-semantics/audit/availability.csv | sort
```

Output (verbatim, 2026-08-07):

```
Aran gms 83.1
Aran gms 84.1
Aran gms 87.1
Aran gms 92.1
Aran gms 95.1
Aran jms 185.1
Cygnus gms 79.1
Cygnus gms 83.1
Cygnus gms 84.1
Cygnus gms 87.1
Cygnus gms 92.1
Cygnus gms 95.1
Cygnus jms 185.1
DualBlade gms 92.1
DualBlade gms 95.1
DualBlade jms 185.1
Evan gms 84.1
Evan gms 87.1
Evan gms 92.1
Evan gms 95.1
Evan jms 185.1
GM gms 12.1
GM gms 48.1
GM gms 61.1
GM gms 72.1
GM gms 79.1
GM gms 83.1
GM gms 84.1
GM gms 87.1
GM gms 92.1
GM gms 95.1
GM jms 185.1
Mechanic gms 95.1
Mechanic jms 185.1
Pirate gms 72.1
Pirate gms 79.1
Pirate gms 83.1
Pirate gms 84.1
Pirate gms 87.1
Pirate gms 92.1
Pirate gms 95.1
Pirate jms 185.1
Resistance gms 95.1
Resistance jms 185.1
SuperGM gms 12.1
SuperGM gms 48.1
SuperGM gms 61.1
SuperGM gms 72.1
SuperGM gms 79.1
SuperGM gms 83.1
SuperGM gms 84.1
SuperGM gms 87.1
SuperGM gms 92.1
SuperGM gms 95.1
SuperGM jms 185.1
```

This is the entire audit surface: a tier-level over-claim is only observable
where a class is `released=true` (where it is `false`, every tier is already
unavailable and a tier split changes nothing).

### DualBlade / Mechanic / Resistance are inert — not audited per-cell

`classOf` (`libs/atlas-constants/gen/availability.go:51-56`) documents that
these three classes have **no identity in the job/skill namespace at all** —
no `canonicalToken` maps to them, so `classOf` never returns those labels and
their `availability.csv` rows never gate anything the generator emits. There
is no cell to over-claim: `Set.Available()` can never return `true` for an
identity these rows would falsely release, because no identity is bound to
them. Confirmed by reading `gen/identities.yaml`: no `DualBlade`, `Mechanic`,
or `Resistance` job/skill entries exist. No further action; listed here so
the 9 rows are accounted for, not silently skipped.

## Step 2 — cells closed from `investigation.md`

### Cygnus (worked example)

`investigation.md` Finding 3: v0.84 `Skill.wz` walk shows `1112/1212/1312/1412/1512`
(the five Cygnus 4th-job branches) have a `skill` node that is **present but
empty** (`1112.img.xml` is 255 bytes, `skill` child has 0 children), while
`1111` (3rd job) has 218 skill children. A live
`GET /api/data/jobs/{id}/skills` sweep on `atlas-main`, gms
79/83/84/87/92/95 + jms 185, returns `{"skills":[]}` for all five branches at
every version.

| Cell | Verdict | Evidence |
|---|---|---|
| Cygnus tier 4 (`1112/1212/1312/1412/1512`), all released=true versions | **OVER-CLAIMED — fixed in Task 3** | `investigation.md` Finding 3; `CygnusStage4` class added to `classOf`, 11 CSV rows `released=false`, `libs/atlas-constants/job/availability_test.go` pins it (`TestAvailable_CygnusStage4NeverReleased`, `TestResolveWire_CygnusStage4StillPresent`) |
| Cygnus tiers 1–3 (Noblesse + DawnWarrior/BlazeWizard/WindArcher/NightWalker/ThunderBreaker stages 1–3), gms 79/83/84/87/92/95, jms 185 | **CORRECT** | `investigation.md` Finding 3 sweep methodology; spot-checked live in this task: `GET /api/data/jobs/1000/skills` (Noblesse) at gms 83 returns 25 real skill ids, `GET /api/data/jobs/1100/skills` (DawnWarrior stage 1) returns 5, `GET /api/data/jobs/1111/skills` (DawnWarrior stage 3) returns 8 — none empty. `libs/atlas-constants/job/availability_test.go`'s `TestAvailable_CygnusTiers1To3NoRegression` pins `Available()=true` at gms79+/jms185 and `=false` at gms72 and earlier for every tier-1–3 identity that has a wire binding at that version. |

### Aran

`investigation.md` does not carry a live-content sweep for Aran (its Finding
2 discusses only the `released=false` gms 79 stub, a different cell than the
six `released=true` cells below). Closed here with fresh live queries
against `atlas-main`'s provisioned tenants, run for all five Aran job
identities in `gen/identities.yaml` at every `released=true` version:
`AranBeginner` (wire 2000), `AranStage1` (2100), `AranStage2` (2110),
`AranStage3` (2111), `AranStage4` (2112). `AranStage4` is the critical check
— its token (2112) sits in the same "…12" position as the five Cygnus 4th-
job branches that turned out to be empty stubs, so it cannot be assumed
non-empty by analogy; it must be queried directly.

```sh
wget -qO- --header 'TENANT_ID: ec876921-c363-4cc6-9c51-5bb8d57f9553' --header 'REGION: GMS' \
  --header 'MAJOR_VERSION: 83' --header 'MINOR_VERSION: 1' http://localhost:8080/api/data/jobs/2112/skills
```

Real output, 2026-08-07, `atlas-data` pod `atlas-data-6ddd7fb-7js2x`:

| Version | 2000 (Beginner) | 2100 (Stage1) | 2110 (Stage2) | 2111 (Stage3) | 2112 (Stage4) |
|---|---|---|---|---|---|
| gms 83.1 | 27 skills, e.g. `[20001000,...,20001031]` | `[21000002,21000000,21001001,21001003]` (4) | `[21100001,...,21100005]` (6) | `[21110002,...,21110008]` (9) | `[21121000,21120002,...,21120010]` (11) |
| gms 84.1 | 36 skills | 4 (identical to gms83) | 6 (identical) | 9 (identical) | 11 (identical) |
| gms 87.1 | 44 skills | 4 (identical) | 6 (identical) | 9 (identical) | 11 (identical) |
| gms 92.1 | 68 skills | 4 (identical) | 6 (identical) | 9 (identical) | 11 (identical) |
| gms 95.1 | 68 skills (identical to gms92) | 4 (identical) | 6 (identical) | 9 (identical) | 11 (identical) |
| jms 185.1 | 40 skills (JMS-specific ordering/subset) | 4 (identical) | 6 (identical) | 9 (identical) | 11 (identical) |

None of the five job ids ever returns `{"skills":[]}` at any of the six
versions — including `2112` (Stage4), the one that would have mirrored the
Cygnus-tier-4 pattern had it been an empty stub. The beginner tier (2000)'s
skill count grows across versions (27→36→44→68), which is the normal
incremental-patch pattern already seen for Aran's beginner skills and for
GM/SuperGM/Pirate above — not evidence of a defect.

| Cell | Verdict | Evidence |
|---|---|---|
| Aran, gms 83/84/87/92/95, jms 185 | **CORRECT** | Live queries above — all five job ids (2000/2100/2110/2111/2112) return non-empty skill lists at every `released=true` version, including the Stage4 id whose token position is analogous to Cygnus's empty tier 4. No hidden empty tier exists in the Aran class; no fix needed. |

### Evan

`investigation.md` Finding 4: the v0.84+ `Skill.wz` has both a top-level
`2200.img`…`2218.img` (real job images, e.g. `2200 → children 22000000,
22001001`) AND a `Skill.wz/Dragon/` subdirectory with animation images named
identically. `registerAllInDirectory` walks recursively; `job.Read`
(pre-Task-1/2) derived the job id from the image name alone and emitted an
empty-skill model for the dragon-animation copy; last-write-wins on
`(tenant, type, document_id)` and ASCII `Dragon` sorting after every numeric
filename meant the blank animation copy always overwrote the real job
document.

| Cell | Verdict | Evidence |
|---|---|---|
| Evan, gms 84/87/92/95, jms 185 | **CORRECT** | `investigation.md` Finding 4 — WZ has the skills (`2200→22000000,22001001`; `2218→22181000..22181003`); `GET /api/data/skills/22001001` on gms 84 returns "Magic Missile" maxLevel 20 with populated effects. The blank `GET /api/data/jobs/{id}/skills` documents are the FR-1 reader bug (fixed by Task 1/2's `job.Read`, which now requires the `skill` node to be present at all — see `services/atlas-data/atlas.com/data/job/reader.go:20-27`), not a release over-claim. Live-verified in this task on `atlas-main` (2026-08-07): `GET /api/data/jobs/2200/skills` still returns `{"skills":[]}` at gms 84/87/92/95 and jms 185 — this is the *already-ingested* documents from before the reader fix landed; the fix only changes future ingest runs, it does not retroactively repair rows already written with `DoUpdates: content` (the same pattern noted for the mob-disease-duration bug in project memory — "needs re-ingest + baseline republish"). This is expected and does not change the verdict: the release class itself is correctly `true`; the live blankness is a data-freshness issue outside this task's scope, already fully diagnosed in `investigation.md` Finding 4. |

### Pirate

| Cell | Verdict | Evidence |
|---|---|---|
| Pirate, gms 79/83/84/87/92/95, jms 185 | **CORRECT** | `investigation.md` sweep methodology + live-verified in this task: `GET /api/data/jobs/500/skills` at gms 79/83/84/87/92/95/jms185 all return non-empty skill lists, e.g. gms 83: `[5000000,5001001,5001002,5001003,5001005]`. `classOf`'s `500..599` range contains only the standard Buccaneer/Corsair branch (`Pirate, Brawler, Marauder, Buccaneer, Gunslinger, Outlaw, Corsair` — verified against `gen/identities.yaml`); there is no later-released sub-branch hidden inside the range the way Cygnus's 4th tier was, so there is no tier-split risk analogous to Task 3's finding. |
| Pirate, gms 72 | **CORRECT** (closed in Step 3, see below) | Live query |

## Step 3 — unevidenced cells closed by live query

Provisioned tenants on `atlas-main` (`GET /api/tenants` on `atlas-tenants`,
2026-08-07):

```
GMS v48  e1f06ae2-80c1-47f7-bb6f-38a9f50d23dd
GMS v61  0d250dc9-64c4-45ae-8bc2-fc0a9cdb5578
GMS v72  48d415ca-59de-4953-9aed-0c4156a09bc9
GMS v79  92adbe47-5ada-4f3b-8224-f58c80a4a2d5
GMS v83  ec876921-c363-4cc6-9c51-5bb8d57f9553
GMS v84  4936dff2-7121-4f46-b9eb-1ae541f4a85f
GMS v87  86da65d2-b9fa-4176-985a-6a5df586220c
GMS v92  db1dbfb3-4345-4731-9223-c40b0c7f6457
GMS v95  c794c706-aea3-4882-90a6-a3b7ee314f52
JMS v185 abedf3b4-1d7c-4b3b-bc52-70f62ab09418
```

No `gms 12.1` tenant exists.

### GM / SuperGM

Wire ids move: at gms 48/61, GM=500/SuperGM=510; at gms 72+ and jms 185,
GM=900/SuperGM=910 (confirmed by reading `libs/atlas-constants/job/version_gms_61_1_gen.go:48-49`
and `version_gms_72_1_gen.go:49-50` — both bind `900: Gm, 910: SuperGm`).
Queries run from an `atlas-data` pod, e.g.:

```sh
wget -qO- --header 'TENANT_ID: e1f06ae2-80c1-47f7-bb6f-38a9f50d23dd' --header 'REGION: GMS' \
  --header 'MAJOR_VERSION: 48' --header 'MINOR_VERSION: 1' http://localhost:8080/api/data/jobs/500/skills
```

| Version | Job (wire id) | Result | Verdict |
|---|---|---|---|
| gms 12.1 | — | no tenant provisioned | **UNVERIFIED** — "gms 12.1 has no provisioned tenant in atlas-main" |
| gms 48.1 | GM (500) | `{"skills":[5001000,5001001,5001002]}` | **CORRECT** |
| gms 48.1 | SuperGM (510) | `{"skills":[5101000,5101001,5101002,5101003,5101004,5101005]}` | **CORRECT** |
| gms 61.1 | GM (900) | `{"skills":[9001000,9001001,9001002]}` | **CORRECT** |
| gms 61.1 | SuperGM (910) | `{"skills":[9101000..9101008]}` (9 ids) | **CORRECT** |
| gms 72.1 | GM (900) | `{"skills":[9001000,9001001,9001002]}` | **CORRECT** |
| gms 72.1 | SuperGM (910) | `{"skills":[9101000..9101008]}` | **CORRECT** |
| gms 79/83/84/87/92/95.1 | GM (900) | `{"skills":[9001000,9001001,9001002]}` (identical every version) | **CORRECT** |
| gms 79/83/84/87/92/95.1 | SuperGM (910) | `{"skills":[9101000..9101008]}` (identical every version) | **CORRECT** |
| jms 185.1 | GM (900) | `{"skills":[9001000..9001009]}` (10 ids — JMS-specific set, real content) | **CORRECT** |
| jms 185.1 | SuperGM (910) | **HTTP 404** | **OVER-CLAIMED — fixed in this task**, see below |

### Pirate at gms 72

```sh
wget -qO- --header 'TENANT_ID: 48d415ca-59de-4953-9aed-0c4156a09bc9' --header 'REGION: GMS' \
  --header 'MAJOR_VERSION: 72' --header 'MINOR_VERSION: 1' http://localhost:8080/api/data/jobs/500/skills
```

Result: `{"skills":[5000000,5001001,5001002,5001003,5001005]}` — real,
non-empty content. **CORRECT.**

## Finding: SuperGM at jms 185 — OVER-CLAIMED

Not anticipated by the brief; found while closing the GM/SuperGM live-query
cells above.

**Live evidence.** `GET /api/data/jobs/910/skills` on the jms 185 tenant
returns HTTP 404 (confirmed twice, on two different `atlas-data` pods across
a rolling deploy). `GET /api/data/skills/9101000` and `.../9101001` on the
same tenant also 404. Every GMS version in the released=true surface
(48/61/72/79/83/84/87/92/95) returns real SuperGM content for the equivalent
wire id.

**WZ evidence.** Fetched `JMS/versions/185.1/Skill.wz` from MinIO
(`atlas-wz` bucket) and walked it with `libs/atlas-wz` (`wz.Open` +
`Root().Images()`, same method as `investigation.md`'s reproduction
section): the root image list contains `900` (Gm) but **no `910` image at
all**. Cross-checked `GMS/versions/95.1/Skill.wz` the same way: its root
contains both `900` and `910`. This is a real content difference between the
two archives, not a naming or ingest quirk — the JMS v185 client WZ simply
never shipped a SuperGm job image.

**Why this wasn't caught by `investigation.md`'s Cygnus/Evan sweep.** That
sweep covered Cygnus, Aran, Evan, and Pirate — never GM/SuperGM (Finding 1's
GM/SuperGM discussion is about the *wire-id* defect at v0.48, not a content
sweep). The `availability.csv` row itself already carried a caveat flagging
this exact doubt: *"meymink is a GMS-only patch log; JMS v185 release timing
for this identity is NOT independently confirmed from meymink -- carried
forward from the GMS anchor only as a cross-region reference point."* This
audit resolves that caveat: `false`, confirmed by direct WZ + live evidence,
not by the patch log.

**Why the fix needed no new release-class label.** Unlike Cygnus tier 4
(which required a new `CygnusStage4` label because `classOf` maps the whole
`1000-1599` range to a single `Cygnus` class and there was no way to express
"released except tier 4"), GM and SuperGM are already **separate** classes
in `classOf` (`t==900 → "GM"`, `t==910 → "SuperGM"`), and
`availability.csv` already carries one row per `(region, major, minor)` —
so the fix is a single-row flip: `jms,185,1,job,SuperGM` `released` from
`true` to `false`, with the `meymink` column replaced by the WZ+live
evidence above (see the CSV diff in this task's commit).

**Regeneration.** `cd libs/atlas-constants/gen && go run .` produced **no
diff** in any generated file (`git status --short libs/atlas-constants/`
after regenerating showed nothing). This confirms the row was already inert
in practice: `identities.yaml`'s own per-version wire join (built
independently, from the same "no 910 image in the JMS WZ" fact) never bound
`SuperGm` to a wire id at jms 185 in the first place, so
`computeAvailable`'s `present ∩ released` intersection was already empty for
this cell regardless of the CSV's stale `true`. `go run . -check` confirms
no drift: `OK: ... are up to date`.

**Test.** `libs/atlas-constants/job/availability_test.go` gained
`TestResolveWire_SuperGmNotBoundAtJms185`, asserting `Wire(SuperGm)` does not
resolve and `Available(SuperGm)` is `false` at jms 185.1 — a regression
guard in case the presence data is ever regenerated from an updated/corrected
JMS WZ and the CSV's `released` flag is the only thing standing between a
future contributor and silently re-enabling a job that was never shipped in
that region.

`go build ./...`, `go vet ./...`, and `go test -race ./...` all clean in
`libs/atlas-constants` after this change.

## Step 4 — jms 185 provenance answer (PRD §9 Q6)

The live sweep in `investigation.md` returned byte-identical Cygnus/Evan
results for jms 185 and GMS — direct evidence answering the *content*
question (does JMS's WZ carry the same skill data as GMS at the equivalent
version). The `meymink` caveat on every jms-185 row concerns *release
timing* — a claim meymink (a GMS-only patch log) cannot independently
confirm for a different region — which is a different question entirely.
Per FR-2.4's convention (WZ wins for content, patch log wins for dates), the
caveat correctly stays as written for every row except the one this audit
resolved (SuperGM), where direct WZ content evidence (no `910` image at all,
not merely an unconfirmed date) settles the question outright regardless of
patch-log timing.

## Observations outside this task's scope (not fixed here)

Step 1 explicitly scopes the audit surface to `released=true` rows — a
tier-level over-claim is only observable there. While closing the Aran and
Cygnus cells, two `released=false` rows were incidentally observed to carry
**non-empty** live skill content (the opposite direction of defect from
Task 3's finding — a possible *under*-claim, not an over-claim):

- `gms,72,1,job,Cygnus,false` ("WZ stub present, unreleased"): live
  `GET /api/data/jobs/1000/skills` at gms 72 returns 12 real skill ids
  (`[10001000,10001001,...,10000012]`), not an empty stub.
- `gms,79,1,job,Aran,false` ("WZ stub present, unreleased"): live
  `GET /api/data/jobs/2000/skills` at gms 79 returns 18 real skill ids.

These are **not** in the `released=true` surface this task audits (FR-2.3),
and changing a `false→true` row is a materially different, broader claim
than the `true→false` corrections this task's mechanism is built for — it
would need its own release-timing verification (was this a real early
opt-in test population, a GM-only preview, or something else?) rather than
the WZ-presence-vs-empty test that resolved Cygnus tier 4 and jms-185
SuperGM. Recorded here per "never silently omit a cell," not fixed here.

## Summary

| Class | Cells in released=true surface | Verdict |
|---|---|---|
| GM | 11 (gms 12/48/61/72/79/83/84/87/92/95, jms 185) | 10 CORRECT, 1 UNVERIFIED (gms 12) |
| SuperGM | 11 (same versions) | 9 CORRECT, 1 UNVERIFIED (gms 12), 1 OVER-CLAIMED — fixed (jms 185) |
| Pirate | 8 (gms 72/79/83/84/87/92/95, jms 185) | 8 CORRECT |
| Aran | 6 (gms 83/84/87/92/95, jms 185) | 6 CORRECT — live-queried all 5 job ids (2000/2100/2110/2111/2112) per version, see Step 2 Aran section |
| Evan | 5 (gms 84/87/92/95, jms 185) | 5 CORRECT |
| Cygnus | 7 (gms 79/83/84/87/92/95, jms 185) | 7 CORRECT (tiers 1–3; tier 4 already split out and fixed under `CygnusStage4`) |
| CygnusStage4 | 0 (all `released=false`, this is the fixed state) | worked example, see above |
| DualBlade / Mechanic / Resistance | 8 (3 rows have no gating identity) | inert, not audited per-cell — see Step 1 note |

Totals across the audited (non-inert) surface: **45 cells CORRECT, 2
UNVERIFIED (gms 12, no provisioned tenant), 2 OVER-CLAIMED (Cygnus tier 4 —
fixed in Task 3; SuperGM at jms 185 — fixed in this task).**
