# task-187 Task 1: multi-boundary skill/job semantics audit

This is the grounding deliverable for task-187 (version-aware skill/job ID
semantics). It authors grounded data — no production code. Every row in
`divergences.csv` and `availability.csv` is backed by one of two evidence
sources, per the task-1 brief's interface contract:

1. **WZ evidence via the live atlas-data baseline** (`bee` cluster,
   namespace `atlas-main`) — the primary evidence source used in this
   audit. atlas-data ingests the real client WZ files per tenant and serves
   version-filtered skill/job data off `TENANT_ID`/`REGION`/`MAJOR_VERSION`/
   `MINOR_VERSION` headers, so the same drain methodology that resolves
   identity names also confirms what a wireId means at a given version.
2. **IDA `func_query` evidence** — attempted per Step 3 for the v48/v61/v72
   boundary (see "IDA evidence attempt" below). It did not yield
   job/skill-constant-named symbols for these three IDBs (their naming
   density is low compared to v83+, consistent with project memory
   `reference_v83_richer_than_v95_for_handlers`), so WZ evidence carries the
   identity claims in this audit; one structural IDA finding (the
   job-hierarchy algorithm) is cited as corroborating, non-identity evidence.

## Provisioned scope

Per `deploy/k8s/base/versions.json`, the provisioned (region,major,minor)
set is exactly:

`gms {12,48,61,72,79,83,84,87,92,95}` (all minor=1) + `jms 185` (minor=1)

Every row in `divergences.csv` / `availability.csv` uses a tuple from this
set. meymink anchors that reference non-provisioned historical versions
(e.g. v0.62 Pirate release, v0.73 Cygnus release, v0.80 Aran release, v0.88
Dual Blade release, v0.94 Resistance release) are cited as **dates only**
— they establish the release-vs-not-released boundary that the *provisioned*
rows straddle; they are never used as row keys themselves.

## Live baseline drain provenance (Step 2)

- **Cluster/context:** `bee`, namespace `atlas-main`.
- **Service:** `atlas-data` (pods `atlas-data-697ff4787d-*`), REST on
  `:8080`, endpoints `GET /api/data/jobs` (paginated, `page[size]=200`
  returns full job→skill-id-set per tenant) and
  `GET /api/data/skills/{id}` (per-skill name/description/effects).
- **Method:** `kubectl exec` into an `atlas-data` pod, `wget -q -O-` with
  the four tenant headers (`TENANT_ID`, `REGION`, `MAJOR_VERSION`,
  `MINOR_VERSION`), per project memory
  `reference_query_atlas_data_skill_per_version`.
- **Tenant discovery:** `atlas-tenants` (`GET /api/tenants`) lists tenants
  already provisioned in the `atlas-main` baseline. It returned **10 of the
  11** provisioned tuples as of the drain (see "Blocked: gms 12" below).
  Tenant IDs used:

  | region | major | tenant id |
  |---|---|---|
  | gms | 48 | `e1f06ae2-80c1-47f7-bb6f-38a9f50d23dd` |
  | gms | 61 | `0d250dc9-64c4-45ae-8bc2-fc0a9cdb5578` |
  | gms | 72 | `48d415ca-59de-4953-9aed-0c4156a09bc9` |
  | gms | 79 | `92adbe47-5ada-4f3b-8224-f58c80a4a2d5` |
  | gms | 83 | `ec876921-c363-4cc6-9c51-5bb8d57f9553` |
  | gms | 84 | `4936dff2-7121-4f46-b9eb-1ae541f4a85f` |
  | gms | 87 | `86da65d2-b9fa-4176-985a-6a5df586220c` |
  | gms | 92 | `db1dbfb3-4345-4731-9223-c40b0c7f6457` |
  | gms | 95 | `c794c706-aea3-4882-90a6-a3b7ee314f52` |
  | jms | 185 | `abedf3b4-1d7c-4b3b-bc52-70f62ab09418` |

- **Timestamp:** drain performed 2026-07-30 during this task-1 session.
- **Scope actually drained:** full job id→skill-id-set for every tenant
  above (`GET /api/data/jobs?page[size]=200`, single page, confirmed
  `meta.page.last=1` in every response); targeted `GET
  /api/data/skills/{id}` calls for every skill id referenced in
  `divergences.csv`'s evidence column (name + `maxLevel` + `effects`
  length, not full payload, except two calls pulled the full record to
  inspect `effects`/`maxLevel` directly — see `bigbang-v092-v095.md`).
  Raw JSON drains were **not** stashed under `audit/raw/` (optional per the
  task-1 brief) — this provenance section plus the per-skill evidence
  citations in the CSVs are the required record. Task 3 can re-drain from
  the same tenants/endpoints on demand.

### Blocked: gms 12 has no live tenant

`GET /api/tenants` on the `atlas-main` baseline returns exactly 10 tenants
(the list above) — **no GMS v12 tenant exists**. Confirmed independently:
`kubectl exec` into `atlas-data` with an invalid `TENANT_ID` and
`MAJOR_VERSION: 12` header returns a non-200/error response (exit code 1
from `wget`). The login/channel load-balancer *does* provision gms_12
socket ports (`ATLAS_CHANNEL_LB_SERVICE_PORT_ATLAS_CHANNEL_GMS_12=1201` /
`ATLAS_LOGIN_LB_SERVICE_PORT_ATLAS_LOGIN_GMS_12=1200` visible in the
`atlas-data` pod's environment, confirming gms_12 is provisioned per
`versions.json`), but no WZ-backed data tenant has been created for it in
this environment.

**Consequence:** `divergences.csv` and `availability.csv` contain **no gms
12 rows**. This is a genuine, reported gap — not a fabricated absence. All
identities audited here (GM/SuperGM, Pirate, Cygnus, Aran, Evan, Dual
Blade, Resistance, Mechanic) unambiguously predate or postdate gms_12
relative to the release anchors in `divergences`/`availability`
(gms_12 = Nov 2005 per meymink v0.12, before every boundary in scope
except GM/SuperGM, which — per the earliest meymink GM mention, v0.05 Jul
2005 — was almost certainly already present at gms_12 under the pre-v61
`500`/`510` binding). No gms_12 row was added on that inference alone
because it is not independently WZ-confirmed; a future task with tenant
access to a gms_12 baseline should add it.

**Update (coordinator follow-up):** gms_12 now mirrors gms_48's full
job/skill divergent-override set (11 rows each: `job` 500/510, `skill`
5001000-5001002/5101000-5101005), including the two job rows (500=Gm,
510=SuperGm) and the `skill,5101004,SuperGmHide` row that were initially
missed. Grounding for the job rows: the Pirate/Brawler class released
meymink v0.62 (Nov 2008), after gms_12 (v0.12, Nov 2005) — so job 500/510
mean Gm/SuperGm at gms_12 exactly as they do at gms_48, corroborated by
`gms_12_1.json`'s job/skill id-sets being byte-identical to `gms_48_1.json`
(independently confirmed, not inferred).

**Update (task-187 Task 1 normalization pass) — DualBlade job identity gap:**
rows 20-21 of `divergences.csv` (gms_87/gms_92, `job`, wireId `430`) record
`DualBlade` as the identityName, but **no `DualBlade` identity exists
anywhere in `libs/atlas-constants/gen/identities.yaml` or the generated
`job/identities_gen.go`/`job/constants.go`** — job token 430 was never
captured in the identity namespace at all (confirmed by enumerating every
`domain: job` `canonicalToken` in `identities.yaml`: the full ascending set
jumps `...,420,421,422,500,510,...` with no 430-434 entries). These two
rows are left **unchanged** rather than normalized, because normalizing
them to a bare `DualBlade` would silently imply a resolvable identity that
does not exist. This is a genuine, reported gap requiring a design/generator
decision (add the missing job-430 identity to `identities.yaml`, which is
generator-code territory out of scope for this data-normalization pass) —
not a fabricated binding.

**Update (task-187 Task 1 normalization pass):** the GM/SuperGM
5001xxx/5101xxx skill-override rows added to `divergences.csv` for gms_48
(see `v048-gm-supergm-skill-ranges.md`) were duplicated for gms_12, because
`gen/wzsnapshot/gms_12_1.json`'s skill id-set for that range is byte-for-byte
identical to `gms_48_1.json`'s (independently confirmed, not inferred), and
the WZ *names* at those ids were drained from the live gms_48 tenant (no
live gms_12 tenant exists, per the gap above). This is evidence from the
checked-in wzsnapshot JSON, not a live gms_12 drain — the gms_12
job-domain rows (500/510/900/910, mirroring rows 2-11) are still not added,
since no snapshot-level job-roster evidence was gathered for them.

## Step 1: meymink release anchors (see `cygnus-anchor.md` for the OQ-3 pin)

Fetched via `curl -s
https://raw.githubusercontent.com/meymink/Maplestory-Patch-Logs/master/README.md`
(HTTP 200, 4707 lines). All anchors below are direct quotes from that file;
line numbers refer to the fetched copy as of 2026-07-30.

| Anchor | meymink version | Date | Line | Patch-note text |
|---|---|---|---|---|
| Earliest GM mention | v0.05 | Jul 14, 2005 | ~4674-4680 | "GM Events" |
| GMS official launch | v0.02 | May 11, 2005 | ~4701-4706 | "Official open patch" |
| Last pre-Pirate | v0.61 | Oct 15, 2008 | ~4095-4105 | "Pre Pirate Quests" |
| Pirate Class release | v0.62 | Nov 12, 2008 | ~4082-4092 | "Pirate Class" |
| Cygnus Knights release | v0.73 | Jul 29, 2009 | ~3943-3956 | "Cygnus Knights Class" (see `cygnus-anchor.md`, OQ-3) |
| Pre-Aran | v0.79 | Nov 11, 2009 | ~3854-3866 | "Pre Aran Quests" |
| Aran Class release | v0.80 | Dec 10, 2009 | ~3833-3850 | "Aran Class" |
| Evan Class release | v0.84 | Mar 31, 2010 | ~3774-3790 | "Evan Class" |
| Pre-Dual-Blade | v0.87 | Jun 23, 2010 | ~3721-3728 | "Pre Dual Blade Quests" |
| Dual Blade Class release | v0.88 | Jul 21, 2010 | ~3701-3717 | "Dual Blade Class" |
| Big Bang ("New Formulas") | v0.92 | Nov 23, 2010 | ~3628-3659 | "New Formulas" (among other Big Bang changes) |
| Resistance Class release | v0.94 | Dec 20, 2010 | ~3611-3624 | "Resistance Class" |
| Mechanic release | v0.95 | Jan 19, 2011 | ~3595-3608 | "Mechanics" |

**Correction to the brief's Step 1 framing:** the brief groups "v0.95
Mechanic/Resistance" as one anchor. The meymink log shows these are two
separate patch versions: **Resistance releases at v0.94** (Dec 20, 2010)
and **Mechanic releases at v0.95** (Jan 19, 2011). Both anchors are
recorded and used correctly in `availability.csv` (Resistance keys off
v0.94, Mechanic off v0.95); nothing was left `UNVERIFIED` here — the
correction is fully sourced from the same fetched log.

No anchor required in Step 1 came back unconfirmable; none are marked
`UNVERIFIED`.

## Version-number mapping note

meymink's `0.XX` client-version numbers map directly to this project's
`majorVersion` (atlas major = GMS × 100 is the wire convention documented
in project memory; the meymink anchor's raw `0.XX` equals atlas's `XX`).
This was cross-checked against the live `atlas-data` pod's environment,
which exposes login/channel LB ports for exactly `GMS_12, GMS_48, GMS_61,
GMS_72, GMS_79, GMS_83, GMS_84, GMS_87, GMS_92, GMS_95, JMS_185` — an exact
match to the provisioned set and to the meymink anchors used above.

## IDA evidence attempt (Step 3)

Sessions used: gms_48=`0bb5f11a`, gms_61=`965202bf`, gms_72=`90e36cb0`.

`mcp__ida-pro__func_query` with `name_regex` searches for
`(?i)(supergm|gmhide|hide)`, `(?i)(brawler|corkscrew|pirate)`,
`(?i)(job5|job9|_500|_510|_900|_910)`, and broader `(?i)job` / `(?i)skill`
filters returned **no function names that encode the job/skill constants
in question** for any of the three IDBs — matches are either empty or
coincidental `sub_XXXXXX` auto-names whose hex address happens to fall in
the 0x500000–0x910000 range (not a real symbol match). This is consistent
with these IDBs' documented lower naming density versus v83+ (project
memory `reference_v83_richer_than_v95_for_handlers`).

One structural (non-identity) IDA finding **was** confirmed and is cited
in `v048-v062.md`: `is_correct_job_for_skill_root` is byte-for-byte
identical between v48 (`0x5c1737`) and v72 (`0x6b682a`) — the client's
job-hierarchy matching algorithm (`skillRoot/100 == job/100` tier check;
`job/10 == skillRoot/10 && job%10 >= skillRoot%10` sub-tier check) is
unchanged across the boundary. This corroborates that the 500/510→900/910
GM relocation and the Pirate/Brawler assignment at 500/510 is a pure
WZ-data reassignment, not an engine-code change — but it does not by
itself identify what 500/510/900/910 *mean* at each version, hence WZ
evidence (not IDA) carries the actual identity claims in
`divergences.csv`.

## Documents in this folder

- `v048-v062.md` — the GM/SuperGM ↔ Pirate/Brawler boundary (Step 3).
- `bigbang-v092-v095.md` — the Big Bang v0.92→v0.95 reorg (Step 4).
- `cygnus-anchor.md` — the Cygnus original-GMS-release pin (Step 1, OQ-3).
- `divergences.csv` — 65 rows, machine-readable wireId→identity bindings.
  (46 original Task-1 rows, normalized in a later pass to strip display
  annotations and correct two identityNames against
  `job/identities_gen.go` — see the "Update" note above and
  `v048-gm-supergm-skill-ranges.md` — plus 16 rows added to complete the
  v48/gms_12 GM/SuperGM 5001xxx/5101xxx skill-override coverage.)
- `availability.csv` — 90 rows, machine-readable release/unreleased flags.
- `v048-gm-supergm-skill-ranges.md` — the v48 GM/SuperGM
  5001xxx/5101xxx skill-range coverage table and misfire assessment
  (task-187 normalization pass).
- `validate.go` (+ `go.mod`) — the Step 6 structural validator.
