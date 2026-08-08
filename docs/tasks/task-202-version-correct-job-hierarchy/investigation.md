# task-202 — Source Investigation

Evidence gathered 2026-08-07 that establishes the three defects in `prd.md`.
Recorded here so the design and execution phases do not re-derive it, and so a
future reader can distinguish what was verified from what was assumed.

## How the Jobs page builds its hierarchy today

Three inputs, none version-aware:

1. **Shape** — `services/atlas-ui/src/lib/jobs/job-advancement-tree.ts`,
   `JOB_GRAPH`: a static `Record<number, {id, name, parent}>` keyed by v83 wire
   ids. Everything derives from it (`rootOf`, `childrenOf`, `jobTreePath`,
   `tierLabel`, `advancementChains`, `jobName`). A near-duplicate name table
   lives at `services/atlas-ui/src/lib/jobs.ts` (`jobNameMap`).
2. **Rail grouping** — `services/atlas-ui/src/components/features/jobs/rail-groups.ts`,
   `RAIL_GROUPS`: Explorers (100/200/300/400/500), Cygnus Knights (1000),
   Legends (2000/2001), Special (800/900).
3. **Visibility** — `JobsPage.tsx` calls `useJobs` → `GET /api/data/jobs`, which
   reports whatever job images the tenant's `Skill.wz` contained
   (`services/atlas-data/atlas.com/data/job/reader.go`).

`GET /api/data/job-availability` (task-187,
`services/atlas-data/atlas.com/data/jobavailability/processor.go`) already
returns the version's *released* identities with version-correct wire ids and
names. Its only consumer is `usePresetJobOptions` (the presets picker). The Jobs
page predates it and never adopted it.

## Finding 1 — v0.48 wire ids 500/510

Already correct backend-side. `libs/atlas-constants/job/version_gms_48_1_gen.go`
binds `500: Gm`, `510: SuperGm`, pinned by
`libs/atlas-constants/job/identity_test.go`
(`TestSet_ResolveWire_v48GmNotPirate`, `TestSet_ResolveWire_v72PirateNotGm`).

The defect is UI-only: a v0.48 `Skill.wz` contains `500.img`/`510.img`, so
`/api/data/jobs` reports them, and `JOB_GRAPH` renders them as "Pirate →
Brawler" under the Explorers rail. The Special/GM rail entry is missing because
wire id 900 is not in that version's WZ.

## Finding 2 — Cygnus at v0.72, Aran at v0.79

Already correct in the audit ledger:

- `availability.csv:34` — `gms,72,1,job,Cygnus,false` ("WZ stub present,
  unreleased"). `version_gms_72_1_gen.go` has no Cygnus in its available set.
- `availability.csv:45` — `gms,79,1,job,Aran,false`. `version_gms_79_1_gen.go`
  has Cygnus but no Aran/Legend.

Again UI-only: the WZ stubs put ids 1000 / 2000 in `/api/data/jobs`, so the
rails render regardless of release status.

## Finding 3 — Cygnus 4th job is genuinely over-claimed

**WZ evidence.** Walking the v0.84 `Skill.wz` with `libs/atlas-wz`:

```
1112  skill children=0     1212  skill children=0     1312  skill children=0
1412  skill children=0     1512  skill children=0
1111  skill children=218   (3rd job, for contrast)
```

The v83 extracted XML agrees — `Skill.wz/1112.img.xml` is 255 bytes:

```xml
<imgdir name="1112.img">
  <imgdir name="info"> … </imgdir>
  <imgdir name="skill"></imgdir>
</imgdir>
```

Note the `skill` node is **present but empty** — this is the FR-1.2 case, and it
is why FR-1.1's "no `skill` node" condition must not be widened to "no skills".

**Live baseline evidence.** `GET /api/data/jobs/{id}/skills` on `atlas-main`,
all supported versions (gms 79/83/84/87/92/95, jms 185): `1112`, `1212`, `1312`,
`1412`, `1512` all return `{"skills":[]}`, while `1111` returns its eight
third-job skills on every one.

**Cause.** `availability.csv` tracks release at class granularity, and
`classOf` (`libs/atlas-constants/gen/availability.go`) maps the whole
`1000`–`1599` range to the single `Cygnus` label. There is no way to express
"Cygnus, but not tier 4" without splitting the label.

## Finding 4 — Evan JOB documents are blanked at v0.84+

**Symptom.** Every Evan stage (`2200`–`2218`) returns `{"skills":[]}` at gms
84/87/92/95 and jms 185, while Evan *skill* documents ingest correctly — v0.84
`GET /api/data/skills/22001001` returns "Magic Missile", `maxLevel 20`, with
populated effects. v83 and v48 correctly 404 that skill, so version gating is
fine. This is **not** the `common`-formula parsing bug (resolved) and not a UI
problem.

**The WZ has the skills.** Same walk as above:

```
2200 → children 22000000, 22001001
2210 → children 22101000, 22101001
2218 → children 22181000, 22181001, 22181002, 22181003
```

**Same ingest run wrote both.** From the `documents` table: SKILL `22001001` at
`17:03:21.635`, JOB `2200` (empty) at `17:03:24.768` — three seconds later, same
tenant, same run.

**Root cause.** `Skill.wz` at v0.84+ has a subdirectory the top-level image list
hides:

```
root: images=88 subdirs=1
DIR  /Dragon  (images=10 subdirs=0)
  IMG /Dragon/2200  skillChildren=-1   (-1 = no skill node at all)
  IMG /Dragon/2210  skillChildren=-1
  …  2211–2218
```

`Skill.wz/Dragon/` holds Evan's Mir animation images, named **exactly**
`2200.img`–`2218.img`. Then:

- `registerAllInDirectory` (`services/atlas-data/atlas.com/data/data/workers/walk.go`)
  is `filepath.WalkDir` — recursive. Both copies are registered.
- `job.Read` (`services/atlas-data/atlas.com/data/job/reader.go`) derives the
  job id from the root imgdir name — `2200.img` for both — and swallows a
  missing `skill` child (`if ssxml, err := exml.ChildByName("skill"); err ==
  nil`), emitting a model with an empty skill list rather than no model.
- `Storage.Add` upserts on `(tenant_id, type, document_id)` with
  `DoUpdates: content` (`services/atlas-data/atlas.com/data/document/db_storage.go`)
  — last write wins. ASCII `Dragon` sorts after every numeric filename, so the
  dragon image is always written last and blanks the real document.

**Why skills survive.** `skill.Read` hard-errors on the missing `skill` node
(surfacing as a `register 2200.img.xml` warn) and skill documents are keyed
individually by skill id, so nothing is erased. Only the JOB document — one row
per job id — is destroyable this way.

**Why v0.84+ only.** v83 and earlier `Skill.wz` archives have no subdirectories
at all.

## Reproducing the WZ walk

The subdirectory is invisible to `wz.File.Root().Images()`; you must walk
`Root().Directories()`. Fetch the archive from MinIO
(`atlas-wz/shared/regions/GMS/versions/<major>.<minor>/Skill.wz`, credentials in
the `atlas-minio-credentials` secret) and open it with `libs/atlas-wz`
(`wz.Open`), then recurse. Live document state is queryable via
`GET /api/data/jobs/{id}/skills` and `GET /api/data/skills/{id}` on an
`atlas-data` pod with the four tenant headers; the pod image is busybox, so use
`wget -O - --header 'K: V'`, not `curl`.
