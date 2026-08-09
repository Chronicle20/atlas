# task-204 — Dual Blade missing from the gms 92 / gms 95 job hierarchy

Direct fix, no PRD phase. This document is the evidence record for the
naming and advancement-shape decisions the change encodes, because neither
is derivable from the repo alone.

## Symptom

atlas-ui's Jobs page renders no Dual Blade branch on any tenant, including
gms 92.1, gms 95.1 and jms 185.1 where the class genuinely shipped (GMS
v0.88). task-202 corrected job names, parents and release availability for
the identities that exist; Dual Blade was never one of them.

## Root cause

The Jobs page graph is `availability ∩ WZ presence`
(`services/atlas-ui/src/lib/hooks/api/useJobGraph.ts`). WZ presence was fine
— the availability half could never emit the branch:

1. `libs/atlas-constants/gen/identities.yaml` had no identity whose
   `canonicalToken` fell in 430–434, and none in the `43xxxxx` skill range.
2. So `classOf` (`gen/availability.go`) never returned `"DualBlade"` — its
   own doc comment said as much: *"DualBlade, Mechanic, and Resistance …
   have NO identity in the namespace at all … their availability.csv rows
   are inert for this generator."*
3. So `availability.csv`'s `gms,92,1,job,DualBlade,true` (and the gms 95 /
   jms 185 rows) gated nothing.
4. So `Set.Job.AvailableIdentities()` never yielded them, so
   `GET /data/job-availability`
   (`services/atlas-data/atlas.com/data/jobavailability/processor.go`)
   emitted nothing, so `buildJobGraph` intersected the branch away.

The gap was known and documented rather than closed — `gen/semantics/*.yaml`
files carry a "the DualBlade (job token 430) gap" header, and
`divergences.csv` already had job-430 rows for gms 87 and gms 92.

## WZ presence (`gen/wzsnapshot/*.json`)

| version | jobs 430–434 | 43xxxxx skill images |
|---|---|---|
| gms 12 – gms 84 | absent | 0 |
| gms 87 | present | 17 |
| gms 92 | present | 26 |
| gms 95 | present | 26 |
| jms 185 | present | 26 |

gms 87 is a genuine unreleased stub, not a release: it is missing
`4330001`, `4340001`, `4341000`, `4341002`, `4341003`, `4341004`,
`4341006`, `4341007`, `4341008`, and `divergences.csv` records that
`4300000` has no WZ name there. `availability.csv` already said
`released=false` for gms 87 and `true` for jms 185 — the one place these
two otherwise-similar columns diverge on this branch — and that call is
confirmed correct by the skill-set delta above. It was left untouched.

## Advancement shape — verified

Read from the provisioned gms 92.1 tenant via
`GET /api/data/quests` (atlas-data, 2026-08-09).

- **Rooted at Rogue (400), not Beginner.** Quest `2351` ("First Mission:
  Infiltration") has `demandSummary` = *"Make a job advancement as a
  #bRogue#k"* and a start gate of
  `jobs:[0,400,410,420,430,411,421,431,412,422,432,433,434]`. Every Dual
  Blade id appears in the Thief-branch gates throughout — e.g. quest `2140`
  ("Beginner Thief's First Training Session") gates on
  `[400,410,411,412,420,421,422,430,431,432,433,434]`.
- **Linear five-step chain.** WZ job gates order the branch by tier —
  `400,410,420,430,411,421,431,412,422,432,433,434` — placing 430 with
  Assassin/Bandit, 431 with Hermit/ChiefBandit, 432 with
  NightLord/Shadower, and 433/434 past the Explorer tiers. The level-70+
  gate (quest `3121`) admits 432/433/434 but not 430/431. The
  `10610`–`10618` quest series ("Reach Dual Blade Lv. 20!" … "Lv. 100!")
  corroborates the level spread.
- **Class name.** "Dual Blade" appears verbatim in WZ quest names (`2350`
  "Dual Blade: The Seal of Destiny", `2363`/`2369` "Dual Blade: Time for
  the Awakening", `10620` "Dual Blade: Top Secret").

## Naming — provenance, and what could not be verified

**Skill display names are WZ-sourced**: `GET /api/data/skills/{id}` against
the gms 92.1 tenant for all 26, plus gms 95.1 for `4310004`.

**Job display names are repo-authored**, as every other job identity's is.
Sources checked and ruled out:

- `String.wz` has no `Job.img`. The only job-adjacent string is
  `String.wz/Skill.img/<jobId>/bookName`, which is a skill-book title
  ("Chief Bandit's Tricks"), not a job name — and atlas-data's
  `skill/string_registry.go` deliberately discards it, so it is not served.
- The client binaries do not carry them: a string search for `"Blade"`
  across the gms 92.1, gms 95.0 and jms 185 IDBs returns **zero** matches
  (the search itself is sound — `"UIWindow"`/`"Skill"` hit normally).
- No quest name, summary or item description names the individual stages.

`BladeRecruit` for 430 was already pinned by the pre-existing
`// jobId = job.BladeRecruit TODO` references in
`libs/atlas-constants/job/model.go` and
`services/atlas-character-factory/atlas.com/character-factory/job/model.go`.
`BladeAcolyte` / `BladeSpecialist` / `BladeLord` / `BladeMaster` were
chosen by the repo owner. They are **not** WZ- or client-verified; if a
gms 92/95 `String.wz` extraction ever becomes available, `Skill.img/43x/
bookName` is the node to check them against.

## Canonical-token convention widened

`gen/identities.go` documents `canonicalToken` as "the current v83 wire id".
Dual Blade has no v83 presence at all, so these five job identities and
their skills use the **v0.92** wire id. The token's only real contract is
stability + uniqueness within a domain, which the v92 ids satisfy; the
widening is noted in `identities.yaml` at the block itself.

## The one true wire divergence

`4310000` (Endure, gms 87 / gms 92 / jms 185) → `4310004` (Shadow
Resistance, gms 95). Same shape as the existing
`AssassinEndure 4100002 → AssassinShadowResistance 4100006` and
`BanditEndure 4200001 → 4200006` Big Bang renames, and modelled the same
way: two identities, each auto-binding only where its wire id is present,
with the relationship recorded as a documentation-only row pair in
`divergences.csv`. jms 185 keeps `4310000`.

## Known adjacent gap, deliberately not changed

`job.Advancement` (`libs/atlas-constants/job/advancement.go`) derives a
0–4 tier as `2 + jobId%10` and returns `-1` above 4. Dual Blade has five
advancements, so 433 and 434 fall out as `-1` — the same escape hatch Evan's
stage line already uses. Its two consumers
(`services/atlas-channel/.../pointreset/model.go`,
`services/atlas-skills/.../skill/processor.go`) validate SP transfers on
that tier, so a Blade Lord / Blade Master SP transfer would be rejected as
wrong-tier.

Left alone on purpose: it is unreachable today (Dual Blade characters cannot
be created — see below), and picking a tier for a five-advancement branch
means deciding what the client actually sends in the SP-reset `tier` byte.
That is a wire question with no evidence gathered here, and guessing it
would be inventing. It needs its own change with a client-side read.

## Out of scope

Character **creation** is a separate, still-open gap: `JobFromIndex`
(`libs/atlas-constants/job/model.go`, mirrored in atlas-character-factory)
still has an empty `subJobIndex == 1` branch, so creating a Dual Blade on a
v92/v95 tenant produces a plain Beginner. This change makes the job tree
render correctly; it does not make the class creatable. The identities this
task adds are the prerequisite that branch was waiting on — `job.BladeRecruit`
now exists — but wiring creation also needs the version-aware race/sub-job
index (the v95 client's race enum differs from the pre-Big-Bang one) and
subJob persistence in atlas-character, neither of which this change touches.
