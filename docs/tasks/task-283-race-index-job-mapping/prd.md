# Version-Aware Race-Index → Job Mapping — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-28
---

## 1. Overview

When a client creates a character it sends `m_nCurSelectedRace` (int32) and
`m_nCurSelectedSubJob` (int16) — an ordinal into the race carousel the login
screen drew, not a job id. `atlas-character-factory` converts that ordinal to a
beginner job with `JobFromIndex`
(`services/atlas-character-factory/atlas.com/character-factory/job/model.go:5-21`),
a version-invariant switch: `0→NoblesseId`, `1→BeginnerId`, `2→LegendId`,
`3→EvanId`, anything else → `BeginnerId`.

That switch encodes exactly one client's carousel order. The GMS v95.0 client
orders the carousel differently, so on a v95 tenant every race except Explorer
binds to the wrong job and, because the start map comes from the matching
`characters.templates` row rather than from the job, lands the character in the
wrong world entirely. Choosing Cygnus produces an Aran-line beginner standing in
Aran's start map; choosing Aran produces an Evan; choosing Evan is rejected
outright because no template row exists for that ordinal. Nothing catches any of
this: `validJob` (`factory/processor.go:649`) is
`func validJob(_ uint32, _ uint32) bool { return true }`.

This task makes the mapping a function of the tenant's client version instead of
a constant, establishes what each supported version's carousel order actually is
by reading the client binaries rather than inferring it from seed data, and turns
`validJob` into a real gate so an ordinal the tenant's client could not have sent
is rejected instead of silently coerced to Beginner.

## 2. Goals

Primary goals:

- Bind race ordinal → beginner job correctly for every supported client version,
  driven by the tenant's `MajorVersion`/`MinorVersion`/`Region`.
- Establish the carousel order for every version column from the client binary
  (IDA), not from inference over seed data. Every mapping row lands with a cited
  address.
- Determine whether GMS v95.0 actually offers the Resistance slot before adding
  any Resistance job constant or template row.
- Replace the no-op `validJob` with a version-aware check that rejects an
  out-of-range or unreleased ordinal.
- Eliminate the duplicated mapping function — `job.FromIndex`
  (`libs/atlas-constants/job/model.go:106-123`) is a byte-identical, currently
  unreferenced twin of the service-local `JobFromIndex`. Exactly one
  implementation may survive.
- Keep the frontend template-editor labels
  (`services/atlas-ui/src/components/features/characters/templates/jobNames.ts`)
  consistent with whatever the backend concludes, so a v95 tenant's admin is not
  shown pre-Big-Bang class names.

Non-goals:

- Implementing Resistance or Dual Blade **gameplay** — skill trees, job
  advancement, class-specific quests. This task binds creation-time identity
  only.
- Changing the `CreateCharacter` codec. `docs/packets/audits/gms_v95/CreateCharacter.md`
  is already ✅ and the field layout is correct; the bug is entirely in
  interpretation of an already-correctly-decoded field.
- The login-side race-availability flags that decide which races the client
  *draws*. Reading them is in scope (they answer the Resistance question);
  changing them is not.
- Re-homing the start map out of `characters.templates`. The template row keeps
  owning `mapId`.

## 3. User Stories

- As a **player on a v95 tenant**, I want to pick Cygnus Knight and receive a
  Noblesse in Ereve, so that the class I selected is the class I get.
- As a **player on a v95 tenant**, I want to pick Evan and have creation succeed,
  so that a race the client offers me is not rejected by the server.
- As a **player on a pre-Big-Bang tenant**, I want character creation to keep
  working exactly as it does today, so that fixing v95 does not regress v83.
- As a **server operator**, I want a race ordinal my client cannot send to be
  rejected with a clear error, so that a malformed or hostile packet cannot mint
  an off-carousel character.
- As a **tenant admin** using the character-template editor, I want the class
  labels to match my tenant's client version, so that I am not editing an "Aran"
  row that my client presents as Cygnus.

## 4. Functional Requirements

### 4.1 Version-aware mapping

- FR-1. The mapping is a pure function of `(region, majorVersion, minorVersion,
  raceIndex, subJobIndex)` returning `(job.Id, ok bool)`. `ok=false` means the
  ordinal is not valid for that version and MUST NOT be coerced to `BeginnerId`.
- FR-2. The version predicate uses the established `MajorAtLeast`-style idiom
  (see `docs/packets/gates.yaml` for the convention). A raw `> N` comparison is
  not acceptable.
- FR-3. The tenant model already carries what is needed: `tenant.Model` exposes
  `Region()`, `MajorVersion()`, and `MinorVersion()`
  (`libs/atlas-tenant/tenant.go:25`), and `Create` already holds
  `t := tenant.MustFromContext(ctx)` at `factory/processor.go:104` — the same
  function whose line 206 calls the mapper. No new plumbing through the call
  chain is required.
- FR-4. Exactly one mapping implementation survives. Given the mapper now needs
  version input and `job.FromIndex` in `atlas-constants` is unreferenced, the
  expected resolution is to delete the `atlas-constants` twin and keep a
  version-aware mapper in the service — but if the constants package is chosen
  as the home instead, the service-local copy must be deleted. A stale twin left
  behind fails this requirement.
- FR-5. Sub-job handling is part of the mapping, not a separate special case.
  Today `subJobIndex == 1` under Explorer hits an empty branch
  (`// jobId = job.BladeRecruit TODO`) and falls through to `BeginnerId`, so a
  Dual Blade is created as a plain Beginner on every version that seeds a
  `(1,1)` row — `gms_92`, `gms_95`, and `jms_185` all do. The mapping must
  either bind Dual Blade to its correct creation job id or return `ok=false`;
  silently producing a Beginner is not an acceptable outcome.

### 4.2 IDA verification of every version column

- FR-6. For each version column with a checked-in IDA export
  (`docs/packets/ida-exports/`: `gms_v48`, `gms_v61`, `gms_v72`, `gms_v79`,
  `gms_v83`, `gms_v84`, `gms_v87`, `gms_v92`, `gms_v95`, `gms_jms_185`), the
  race-ordinal → class binding MUST be read out of the client and recorded with
  the function name and address it came from.
- FR-7. The v95 claim to confirm or correct is: `CLogin::Update` at `0x5dee90`
  gives `0 = Resistance, 1 = Explorer (subJob 1 = Dual Blade), 2 = Cygnus,
  3 = Aran, 4 = Evan`. This arrived as a lead, not as a finding this task has
  verified; it must be re-derived independently before being encoded.
- FR-8. The pre-Big-Bang claim to confirm or correct is:
  `0 = Cygnus, 1 = Explorer, 2 = Aran, 3 = Evan` for the v83–v92 range. This is
  currently an **inference from seed data**, consistent with the rows but never
  read out of a binary.
- FR-9. `jms_185` MUST be verified independently rather than assumed to share
  either mapping. Its seed template carries the same `(0,0)/(1,0)/(1,1)/(2,0)/(3,0)`
  row set as `gms_95`, which is suggestive but not evidence.
- FR-10. `gms_12` has a seed template (`template_gms_12_1.json`, a single `(1,0)`
  row for both genders) but **no IDA export**. It is therefore not verifiable by
  this task's method. Record it as unverified rather than guessing; its lone
  Explorer row is unaffected by any plausible mapping.
- FR-11. Findings are written to a durable artifact in this task folder — one row
  per (version, ordinal, subJob) with the address it was read from — before any
  mapping code is written.

### 4.3 The Resistance question

- FR-12. Determine from the v95 client whether the Resistance slot is
  **selectable**, not merely present in the enum. The race-availability flags
  consulted by the login screen are the evidence; enum membership alone is not.
- FR-13. If Resistance is not selectable on v95.0, add no `Citizen` job constant
  and no `(0,0)` v95 template row. The mapping should return `ok=false` for that
  ordinal so a spoofed packet is rejected.
- FR-14. If Resistance **is** selectable, a new beginner job constant is
  required: `libs/atlas-constants/job/constants.go` currently defines
  `BeginnerId = 0`, `NoblesseId = 1000`, `LegendId = 2000`, `EvanId = 2001` and
  has **no** Citizen/Resistance entry. The id must be read from game data, never
  assumed. A matching v95 template row with a start map read from WZ data is
  then also required.
- FR-15. Note that `BladeRecruit` exists today only as an `Identity` (`= 430`,
  `libs/atlas-constants/job/identities_gen.go:41`), not as a creation-time
  `job.Id`. FR-5 may therefore require a constant addition on the same terms as
  FR-14 — value read from data, not assumed.

### 4.4 Validation

- FR-16. `validJob` (`factory/processor.go:649`) stops returning `true`
  unconditionally. It rejects any `(raceIndex, subJobIndex)` for which the
  version-aware mapping returns `ok=false`.
- FR-17. Rejection is a clean, logged failure returned to the client, consistent
  with how `ErrTemplateNotFound` is already surfaced at
  `factory/processor.go:111-114`. The client must receive a failure rather than
  hang.
- FR-18. The existing template lookup remains the second gate: a mapping may
  succeed while `findCreationTemplate` (`factory/processor.go:80`) still finds no
  row, and that path keeps its current behavior.

### 4.5 Seed data

- FR-19. Any version whose verified carousel exposes an ordinal with no
  corresponding template row needs that row added, with `mapId` read from WZ
  data. The known instance is the v95 Evan ordinal: no version currently seeds a
  `(4,0)` row.
- FR-20. Where verification shows an existing row's `mapId` contradicts the
  verified class for that ordinal, the row is corrected. On `gms_95_1.json`,
  ordinal 2 currently carries `mapId 140090000` (Aran's start map) and ordinal 0
  carries `130010220`; if the v95 carousel is confirmed as claimed, both are
  wrong for the class that ordinal actually selects.
- FR-21. Pre-Big-Bang seed rows are not to be touched unless IDA verification
  positively contradicts them.

### 4.6 Frontend

- FR-22. `jobNames.ts:42-45` hardcodes `Cygnus Knights (0.0)`,
  `Adventurer (1.0)`, `Aran (2.0)`, `Evan (3.0)` and its test asserts it
  "mirrors JobFromIndex". Once the backend mapping is version-aware, this list is
  wrong for post-Big-Bang tenants. The editor must label rows using the class the
  ordinal maps to **for the tenant being edited**.
- FR-23. `worldNameFromJobIndex`'s fallback (`Job ${jobIndex}`) must still cover
  an ordinal with no known label; the editor must not crash or mislabel on an
  unrecognized index.

## 5. API Surface

No new endpoints and no change to the request or response shape of character
creation. `factory.RestModel` keeps `jobIndex` and `subJobIndex`
(`factory/rest.go:16-17`) as the wire-facing names.

Behavioral changes to the existing creation endpoint:

| Condition | Today | After |
|---|---|---|
| v95 tenant, race 2 (Cygnus) | Creates a Legend at Aran's start map | Creates the verified Cygnus beginner at its verified start map |
| v95 tenant, race 3 (Aran) | Creates an Evan | Creates the verified Aran beginner |
| v95 tenant, race 4 (Evan) | `ErrTemplateNotFound` | Creates an Evan (once FR-19 seeds the row) |
| v95 tenant, race 0 (Resistance) | Creates a Noblesse at `130010220` | Per FR-12/13/14 |
| Any tenant, ordinal outside the verified carousel | Silently coerced to `BeginnerId` | Rejected with a validation error |
| `(1,1)` Dual Blade on v92/v95/jms_185 | Creates a plain Beginner | Per FR-5 |

The tenant configuration read path is unchanged: the endpoint already calls
`configuration.GetTenantConfig(t.Id())` at `factory/processor.go:105-109`.

## 6. Data Model

No database schema change. No new entity. No migration.

`characters.templates` rows may gain instances (FR-19) and have `mapId` values
corrected (FR-20), but `template.RestModel`
(`configuration/tenant/characters/template/rest.go`) keeps its current field set
— `jobIndex`, `subJobIndex`, `mapId`, `gender`, and the appearance/equipment
option lists. Because the chosen approach derives the job in code from the
tenant version, no `jobId` field is added to the template row.

Current seed row inventory, as read from
`services/atlas-configurations/seed-data/templates/` (each `(jobIndex,
subJobIndex)` present for both genders):

| Template | Rows `(jobIndex, subJobIndex) → mapId` |
|---|---|
| `gms_12` | (1,0)→10000 |
| `gms_48`, `gms_61`, `gms_72`, `gms_79`, `gms_83` | (0,0)→130010220, (1,0)→10000, (2,0)→140090000 |
| `gms_84` | above + (3,0)→100030102 |
| `gms_87` | above + (3,0)→100030100 |
| `gms_92`, `gms_95`, `jms_185` | above + (1,1)→10000, (3,0)→100030100 |

Constants that may need to grow (`libs/atlas-constants/job/constants.go`),
each gated on FR-14/FR-15 and on a value read from game data:

- a Resistance/Citizen beginner id — no such constant exists today
- a Dual Blade creation id — `BladeRecruit` exists only as `Identity = 430`

## 7. Service Impact

**`atlas-character-factory`** — primary. `job/model.go` gains the version
parameter (or is deleted in favor of the constants package, per FR-4).
`factory/processor.go` passes the tenant version at line 206 and implements the
real `validJob` at line 649. `factory/processor_test.go` currently asserts the
mapping by calling the very function under test —
`expectedJobId := job2.JobFromIndex(input.JobIndex, input.SubJobIndex)` at lines
400 and 1070 — which is a tautology that cannot fail. Those assertions must be
replaced with per-version expected job ids taken from the FR-11 findings table.

**`libs/atlas-constants/job`** — `model.go:106-123` (`FromIndex`) is deleted or
becomes the single version-aware home (FR-4). `constants.go` may gain beginner
ids per FR-14/FR-15. Note `advancement_test.go:15-18` and `model.go:57`
(`IsA(jobId, BeginnerId, NoblesseId, LegendId, EvanId)`) enumerate the beginner
set; a new beginner id must be added there too or beginner-identity checks will
silently exclude it.

**`atlas-configurations`** — seed template edits per FR-19/FR-20. Up to 11 files
exist; only versions whose verification demands a change are touched.

**`atlas-ui`** — `templates/jobNames.ts` plus `IdentitySection.tsx` (which builds
its class dropdown from `jobIndex`/`subJobIndex` and labels via
`worldNameFromJobIndex` at line 39) and the two affected test files under
`templates/__tests__/`.

**No packet-layer change.** `libs/atlas-packet` already decodes the field
correctly and the v95 audit is ✅.

## 8. Non-Functional Requirements

- **Multi-tenancy.** The mapping must be derived per request from the tenant in
  context. No process-global or package-level version state; two tenants on
  different client versions must be able to create characters concurrently and
  correctly.
- **Backward compatibility.** Pre-Big-Bang tenants must observe byte-identical
  creation behavior. This is the highest-risk property of the change and needs
  explicit regression coverage, not just a passing build.
- **Evidence.** Every mapping row traces to a cited IDA address or is marked
  unverified (`gms_12`, per FR-10). No row is populated from remembered
  MapleStory knowledge.
- **Observability.** A rejected ordinal logs the tenant, version, and the
  `(raceIndex, subJobIndex)` pair, matching the existing rejection log style at
  `factory/processor.go:112`.
- **Security.** `validJob` becoming a real gate closes the current hole where an
  arbitrary `raceIndex` is accepted and coerced rather than rejected.

## 9. Open Questions

1. **Is Resistance selectable on GMS v95.0?** Deferred to IDA investigation per
   FR-12. Drives whether a Citizen constant and a v95 `(0,0)` row are needed, and
   what start map that row would carry. This is the one question that can change
   the size of the change materially.
2. **Where does the surviving mapper live** — `atlas-character-factory/job` or
   `libs/atlas-constants/job`? The constants package is the repo's stated home
   for domain types, but it currently has no notion of client version, so putting
   a version-gated function there may drag a version abstraction into it. Design
   phase decides; FR-4 only requires that one copy die.
3. **What is Dual Blade's correct creation job id**, and is it the same across
   v92, v95, and jms_185? `BladeRecruit = 430` is an `Identity`, and whether the
   client expects a character created at that identity or at Explorer-beginner
   with a sub-job marker is unresolved.
4. **Does `jms_185` share the v95 carousel?** Its seed rows match v95's shape but
   this is not evidence (FR-9).
5. **Is `gms_12` acceptable as permanently unverified**, given no IDA export
   exists for it? Its single Explorer row is insensitive to the mapping, so the
   risk appears nil, but it leaves one column outside the evidence standard.
6. **Do any existing characters need repair?** Characters already created on a
   v95 tenant carry a wrong `jobId` and were spawned in the wrong map. Whether a
   data-repair pass is in scope, or whether this fix is forward-only, is
   undecided.

## 10. Acceptance Criteria

- [ ] A findings artifact in this task folder records, per version column and per
      ordinal, the class binding and the IDA function + address it was read from,
      covering all ten exports in `docs/packets/ida-exports/`; `gms_12` is listed
      as unverified with the reason.
- [ ] The v95 claim (`CLogin::Update` `0x5dee90`) is independently re-derived and
      either confirmed or corrected in that artifact.
- [ ] The pre-Big-Bang mapping is IDA-verified rather than inferred, and any
      contradiction with existing seed rows is recorded and resolved.
- [ ] `jms_185` is verified independently of `gms_95`.
- [ ] The Resistance selectability question (FR-12) is answered with cited
      evidence, and FR-13 or FR-14 is applied accordingly.
- [ ] Race→job mapping is a function of the tenant's version, using the
      `MajorAtLeast` idiom, with no raw `> N` version comparison.
- [ ] Exactly one mapping implementation exists in the repo; `grep -rn
      "FromIndex" --include="*.go"` shows no orphaned twin.
- [ ] `validJob` rejects an ordinal outside the tenant version's verified
      carousel; an accepted ordinal is never silently coerced to `BeginnerId`.
- [ ] `(1,1)` Dual Blade no longer produces a plain Beginner on `gms_92`,
      `gms_95`, or `jms_185` — it either binds correctly or is rejected.
- [ ] A v95 tenant creating each selectable race yields the verified job id and
      the verified start map; covered by tests keyed to the findings table, not
      by calling the mapper to compute its own expectation.
- [ ] The tautological assertions at `factory/processor_test.go:400` and `:1070`
      are replaced with version-keyed expected values.
- [ ] Regression tests prove pre-Big-Bang tenant creation behavior is unchanged
      for every existing seeded `(jobIndex, subJobIndex, gender)` row.
- [ ] Any new beginner job id is added to the beginner set at
      `libs/atlas-constants/job/model.go:57` and its advancement test.
- [ ] The template editor labels classes according to the tenant's version;
      `jobNames.ts` no longer claims to mirror a version-invariant
      `JobFromIndex`, and its tests are updated.
- [ ] Flagless `tools/verify.sh` exits 0.
- [ ] Code review completed before PR.
