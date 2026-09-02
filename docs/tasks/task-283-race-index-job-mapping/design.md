# Version-Aware Race-Index → Job Mapping — Design

Version: v1
Status: Draft
Created: 2026-08-28
Consumes: `prd.md` (v1, approved)
---

## 1. Summary

The change replaces one version-invariant `if/else` chain with a **declarative,
per-version carousel table** selected from the tenant in context, plus a real
validation gate. The table is populated only from IDA-derived evidence recorded
in a findings artifact written *before* any mapping code exists.

The single most important architectural property is that the two questions the
PRD leaves open — is Resistance selectable on v95 (PRD §9.1), and what is Dual
Blade's creation job id (PRD §9.3) — must resolve into **table rows, not code
paths**. Both answers change what the table contains; neither changes the shape
of the mapper, the validator, the tests, or the frontend. That property is the
reason this design chooses a table over the obvious "add a version parameter to
the existing switch."

## 2. Decisions

These are the design-phase decisions the PRD deferred. Each is settled here;
the plan phase implements them and does not re-litigate them.

### D-1. The mapper lives in `atlas-character-factory`, not `atlas-constants`

**Decision.** Keep the mapper at
`services/atlas-character-factory/atlas.com/character-factory/job/model.go`.
Delete `FromIndex` from `libs/atlas-constants/job/model.go:106-123`.

**Rationale.** `libs/atlas-constants/go.mod` requires exactly one direct
dependency (`github.com/google/uuid`). It does not depend on
`libs/atlas-tenant`, and nothing in `libs/atlas-constants/` references it
(verified: `grep -rn "atlas-tenant" libs/atlas-constants/` is empty). A
version-aware mapper needs `tenant.Model` for the `MajorAtLeast`/`IsRegion`
idiom that FR-2 mandates, so homing it in the constants package would add a new
dependency edge to the repo's leaf domain-vocabulary module solely to express a
client-presentation concern.

The division of ownership that falls out is clean and worth stating explicitly:

- `libs/atlas-constants/job` owns **what jobs exist** — `BeginnerId`,
  `NoblesseId`, `LegendId`, `EvanId`, and any new beginner id FR-14/FR-15
  requires. This is version-invariant domain vocabulary.
- `atlas-character-factory/job` owns **which carousel slot a given client's
  login screen drew for which job**. This is a per-version UI-ordering fact and
  belongs to the service that interprets the creation packet.

Alternative rejected: making `atlas-constants/job` the home and passing
`(region string, major, minor uint16)` scalars to avoid the tenant import. This
keeps go.mod clean but forfeits the `MajorAtLeast` idiom FR-2 requires, and
re-implements version comparison by hand at the exact site where the PRD warns
against raw `> N`. Rejected.

FR-4 is satisfied by deletion: after the change,
`grep -rn "FromIndex" --include="*.go" .` must show the service-local mapper and
its callers only.

### D-2. A table, not a switch

**Decision.** Model the carousel as data.

```go
// Slot is a (raceIndex, subJobIndex) pair as sent by the client.
type Slot struct {
    RaceIndex   uint32
    SubJobIndex uint32
}

// Carousel is one client version's race-selection screen: the exact set of
// slots that client can send, and the beginner job each one creates.
type Carousel map[Slot]job.Id

// carouselFor selects the carousel for the tenant's client version.
func carouselFor(t tenant.Model) Carousel

// FromIndex maps a client-sent race ordinal to its beginner job id.
// ok=false means the tenant's client could not have sent this slot; the
// caller MUST reject rather than substitute a default.
func FromIndex(t tenant.Model, raceIndex, subJobIndex uint32) (job.Id, bool)
```

`FromIndex` is a single map lookup. **There is no default branch and no
fallback to `BeginnerId`** — absence from the map *is* `ok=false`. This is the
mechanism by which FR-1 holds structurally rather than by discipline: there is
no line of code that could coerce an unknown ordinal, because coercion would
require adding one.

**Rationale.** The alternative — extending the existing `if/else` chain with
version conditions — produces a nested predicate soup whose branch count is
(versions × ordinals) and which cannot be exhaustively read against the findings
table by eye. The map form is directly diffable against `findings.md`: one entry
per row, same key, same value. It also makes the FR-19 seed-row obligation
mechanical (every slot in a version's carousel must have a matching template
row, and that correspondence is testable — see D-7).

**YAGNI note.** `Carousel` is a plain map, not a struct with methods. It gains
no `Contains`, no `Slots()`, no iteration order. If a later need appears, add it
then.

### D-3. Version selection is explicit, ordered, and closed

**Decision.** `carouselFor` is an ordered predicate chain over named package
vars, most-specific first:

```go
func carouselFor(t tenant.Model) Carousel {
    switch {
    case t.IsRegion("JMS") && t.MajorAtLeast(185):
        return jmsV185Carousel   // contents per findings.md, FR-9
    case t.IsRegion("GMS") && t.MajorAtLeast(95):
        return gmsV95Carousel    // contents per findings.md, FR-7
    ...
    default:
        return preBigBangCarousel // FR-8
    }
}
```

Every predicate uses `IsRegion` / `MajorAtLeast` / `MajorInRange`
(`libs/atlas-tenant/tenant.go:87-105`), matching the established idiom at e.g.
`services/atlas-channel/atlas.com/channel/battleship/processor.go:78`. No raw
`> N` anywhere (FR-2).

**The number of distinct carousels is an output of the findings, not an input.**
The chain above is illustrative shape only. If IDA verification shows v83
through v92 share one order and only v95 diverges, there are two carousels and
two arms. If v84 introduced Evan at a different ordinal than v87, there are
more. The plan phase writes the arms the findings dictate — it does not
pre-commit to a count.

**Default arm.** The default returns the verified pre-Big-Bang carousel rather
than an empty map. This is what preserves `gms_12` (FR-10): it has no IDA export
and cannot be verified, but its lone seeded slot is `(1,0)` → Explorer, which is
present in every candidate mapping, so it is insensitive to the ambiguity. An
empty default would break it. The default arm carries a comment saying exactly
this.

### D-4. Evidence precedes code — `findings.md` is a hard gate

**Decision.** `docs/tasks/task-283-race-index-job-mapping/findings.md` is
written and committed **before** any mapping code is written (FR-11), and no
carousel entry may exist without a corresponding row in it.

Row schema — one row per (version, raceIndex, subJobIndex):

| version key | raceIndex | subJobIndex | class | job id | IDA function | address | notes |
|---|---|---|---|---|---|---|---|

Rules the artifact enforces:

- **Derivation, not confirmation.** FR-7's lead (`CLogin::Update` at `0x5dee90`,
  `0=Resistance, 1=Explorer, 2=Cygnus, 3=Aran, 4=Evan`) is written into the
  brief as *the claim under test*, and the investigation must reach its own
  ordering before comparing. A row that says "matches the lead" without an
  independently reached address is not evidence.
- **`jms_185` is derived from `gms_jms_185.json`, never copied from
  `gms_v95`** (FR-9). Matching seed-row shape is suggestive and is recorded as
  such in `notes`; it is not a source.
- **Pre-Big-Bang is derived, not inherited from the seed rows** (FR-8). The
  existing `(0,0)→130010220, (1,0)→10000, (2,0)→140090000` row set is the
  hypothesis under test, and deriving the carousel *from* it would be circular
  — those rows are exactly what this task suspects of being wrong on some
  columns.
- **`gms_12` gets a row marked `unverified`** with the reason "no IDA export"
  and a note that its single `(1,0)` slot is insensitive to the ambiguity
  (FR-10).
- **A cell that cannot be derived is recorded `unverified`, never guessed.**

Ten version columns have exports (`docs/packets/ida-exports/`: `gms_v48`,
`gms_v61`, `gms_v72`, `gms_v79`, `gms_v83`, `gms_v84`, `gms_v87`, `gms_v92`,
`gms_v95`, `gms_jms_185`); `gms_12` has a seed template but no export.

**Rationale.** This is the ordering the PRD's non-functional evidence
requirement implies, but stating it as a gate matters more than it looks:
writing the table first is what prevents the mapping code from becoming the
thing the findings are reconciled *to*. It also means the two open questions are
answered from one investigation pass rather than discovered mid-implementation.

### D-5. The two open questions resolve as table content

**Resistance (PRD §9.1, FR-12/13/14).** The investigation reads the v95 client's
race-availability flags — the values the login screen consults to decide which
slots it *draws* — not merely the race enum's membership. Enum membership proves
the concept exists in the binary; it does not prove the slot is selectable.

- **Not selectable** → `gmsV95Carousel` simply has no entry for that ordinal.
  No `Citizen` constant, no `(0,0)` v95 template row, and a spoofed `raceIndex`
  for it is rejected by the same absent-key path as any other invalid ordinal
  (FR-13). Zero additional code.
- **Selectable** → one new constant in `libs/atlas-constants/job/constants.go`
  with its id **read from game data**, one carousel entry, one seeded template
  row with `mapId` read from WZ data, plus the beginner-set updates in D-6
  (FR-14).

**Dual Blade (PRD §9.3, FR-5).** Same structure. `BladeRecruit` exists today
only as `Identity = 430` (`libs/atlas-constants/job/identities_gen.go:41`), not
as a creation-time `job.Id`. The investigation determines what job id the client
expects a `(1,1)` character to be created at. Outcomes:

- A distinct creation job id exists in game data → new constant (same terms as
  Resistance: value read, never assumed), carousel entry `{1,1} → thatId`, plus
  D-6 updates if it is a beginner.
- The client expects Explorer-beginner with a sub-job marker → the carousel maps
  `{1,1} → BeginnerId`. This is *not* the current silent fallback: it is an
  explicit, evidenced entry, distinguishable in the table and in tests from the
  absent-key case.
- The slot is not actually offered → no entry, and `(1,1)` is rejected. This
  outcome also obliges removing or annotating the `(1,1)` seed rows in
  `gms_92`, `gms_95`, and `jms_185`, which currently exist.

Whichever outcome lands, the acceptance criterion "no longer produces a plain
Beginner by fallthrough" is met, because the fallthrough path no longer exists
(D-2).

### D-6. New beginner ids must join the beginner set

**Decision.** Any new beginner job id added under FR-14/FR-15 is added in the
same change to:

- `libs/atlas-constants/job/constants.go` — the id itself
- the `Jobs` registry map in the same file — or `job.Jobs[id]` lookups
  (`IsFourthJob`, `FromSkillId`) silently miss it
- `IsBeginner` at `libs/atlas-constants/job/model.go:57`, currently
  `IsA(jobId, BeginnerId, NoblesseId, LegendId, EvanId)`
- the beginner enumeration in `libs/atlas-constants/job/advancement_test.go:15-18`

**Rationale.** This is the seam a green build cannot see. `IsBeginner` is a
hand-maintained allow-list; omitting a new id there produces a character that is
a beginner in every respect except the one predicate the rest of the codebase
asks. It is called out as its own decision rather than left implicit in the
constants edit because the failure is silent and downstream.

### D-7. `validJob` becomes the mapper's own `ok`, and the call moves

**Decision.** `validJob` (`factory/processor.go:649`) is replaced by using the
mapper's `ok` return directly. The job id is resolved **once**, early, and both
validated and reused.

The call site moves. Today `validJob` is invoked at `factory/processor.go:101`,
three lines *before* `t := tenant.MustFromContext(ctx)` at line 104 — so it
cannot see the tenant. The check moves below the tenant fetch:

```go
t := tenant.MustFromContext(ctx)

jobId, ok := job2.FromIndex(t, input.JobIndex, input.SubJobIndex)
if !ok {
    p.l.Errorf("Race index [%d] subJobIndex [%d] is not selectable on region [%s] version [%d.%d]; rejecting creation.",
        input.JobIndex, input.SubJobIndex, t.Region(), t.MajorVersion(), t.MinorVersion())
    return "", ErrInvalidRaceIndex
}
```

and the resolved `jobId` is passed to the saga payload at line 206 in place of
today's `job2.JobFromIndex(input.JobIndex, input.SubJobIndex)` call. Resolving
once and reusing removes the possibility of the validator and the payload
disagreeing.

**Ordering note for the plan.** Moving the check below the tenant fetch changes
which error a request with *both* a bad name/gender and a bad race index
receives — name and gender are still checked first, so ordering among the
existing validators is preserved; only race-index validation moves later. This
is a deliberate, stated behavior change, not an accident.

**Error surfacing (FR-17).** A new sentinel `ErrInvalidRaceIndex` alongside
`ErrTemplateNotFound`, surfaced through the same path so the client receives a
failure rather than hanging, and logged in the style already at
`factory/processor.go:112`. The log line carries tenant region and version
(NFR observability) because "ordinal 4 rejected" is unactionable without knowing
which carousel rejected it.

**Two gates remain (FR-18).** Mapping success does not imply a template exists.
`findCreationTemplate` (`factory/processor.go:80`) keeps its current behavior
and its `ErrTemplateNotFound`. A slot that maps but has no template row is a
seed-data gap (FR-19), and it is better for that to surface as the distinct
error it already is.

### D-8. Seed-data changes are consequences of findings, not of this design

Per FR-19/FR-20/FR-21, `services/atlas-configurations/seed-data/templates/`
changes only where verification demands it:

- A verified slot with no template row gets one, `mapId` read from WZ data. The
  known candidate is the v95 Evan ordinal — no template currently seeds `(4,0)`.
- A row whose `mapId` contradicts the verified class for its ordinal is
  corrected. The known candidates are `gms_95_1.json` ordinal 2
  (`mapId 140090000`, Aran's start map) and ordinal 0 (`130010220`) — both wrong
  if the v95 carousel is confirmed as the lead claims.
- Pre-Big-Bang rows are **not** touched unless IDA positively contradicts them
  (FR-21). "The findings didn't mention it" is not a contradiction.

**Correspondence test.** A test asserts that for every version with both a
carousel and a seed template, every carousel slot has a template row for both
genders, and every template row's slot is in that version's carousel. This
converts FR-19/FR-20 from a manual sweep into a gate, and it is the check that
would have caught the original v95 Evan gap.

### D-9. Frontend parity is enforced by a shared fixture, not by a comment

**Problem.** `jobNames.ts:1-11` currently documents itself as mirroring
`JobFromIndex`, and its test asserts that mirroring. Once the backend is
version-aware the TS side needs the same per-version knowledge, which
reintroduces exactly the duplication FR-4 eliminates on the Go side — this time
across a language boundary, where the compiler cannot help.

**Decision.** Promote the findings into a machine-readable artifact,
`docs/packets/race-carousels.json`, keyed by version, and have **both** sides
assert against it in tests:

- Go: a unit test in `atlas-character-factory/job` reads the JSON and asserts
  each `carouselFor` result matches it exactly.
- TS: a Vitest test reads the same file via `fs` (relative path from
  `services/atlas-ui/`) and asserts the label table covers exactly the same
  slots.

Neither side loads the JSON at runtime. Go keeps its compiled table; TS keeps
its literal. The file is a **parity fixture**, and a drift on either side turns
a silent mislabel into a failing test.

**Alternatives rejected.**
- *Code generation from Go into TS.* Real parity, but it adds a generator, a
  `make generate` step, and a CI staleness check to keep two ~20-entry tables in
  sync. Disproportionate.
- *A backend endpoint serving the carousel to the UI.* Perfect parity, but it
  adds an API surface the PRD explicitly excludes (§5, "no new endpoints") for
  the sake of admin-editor labels.
- *Hand-mirroring with a comment.* This is the status quo, and it is exactly the
  failure being fixed.

**UI behavior.** `worldNameFromJobIndex`, `KNOWN_CLASSES`, and
`IdentitySection.tsx`'s class dropdown become functions of the selected tenant's
version. The tenant is already in scope via `useTenant()`
(`services/atlas-ui/src/context/tenant-context.tsx:216`) and carries
`majorVersion` (FR-22). The `` `Job ${jobIndex}` `` fallback stays: an ordinal
with no label must render, not crash (FR-23). The stale comment at
`jobNames.ts:1-4` claiming a version-invariant mirror is deleted.

### D-10. The fix is forward-only

**Decision.** No data-repair pass for characters already created on a v95
tenant with a wrong `jobId` and start map (PRD §9.6). Out of scope.

**Rationale.** A repair needs a rule for what a mis-created character *should*
have been, and that rule is unrecoverable from the stored row — the original
`raceIndex` is not persisted, so a Legend created from ordinal 2 is
indistinguishable from a Legend created legitimately. Any repair would be
heuristic, and a heuristic that rewrites live character identity is worse than
the bug. Existing per-character admin tooling can correct individual cases.

**Flagged for the user:** if affected characters exist in a live v95 tenant and
you want them addressed, that is a separate task with its own PRD, not an
addition here.

## 3. Component boundaries

| Unit | Purpose | Depends on |
|---|---|---|
| `findings.md` | Evidence of record: version × slot × class × address | IDA exports |
| `docs/packets/race-carousels.json` | Machine-readable projection of findings; cross-language parity fixture | `findings.md` |
| `atlas-character-factory/job` | `Slot`, `Carousel`, `carouselFor`, `FromIndex` | `atlas-constants/job`, `atlas-tenant` |
| `atlas-constants/job` | Job id vocabulary + beginner set | (unchanged deps) |
| `factory/processor.go` | Resolve-once, reject-or-proceed, saga payload | `atlas-character-factory/job` |
| seed templates | `(slot, gender) → mapId` per version | findings |
| `atlas-ui/templates` | Version-correct class labels | parity fixture (test-time only) |

Each is independently testable. The mapper takes a `tenant.Model` and two
`uint32`s and returns `(job.Id, bool)` — no I/O, no context, no clients — so
the per-version table tests are pure and fast.

## 4. Testing strategy

**Kill the tautology first.** `factory/processor_test.go:400` and `:1070`
currently assert `expectedJobId := job2.JobFromIndex(input.JobIndex,
input.SubJobIndex)` — the function under test computing its own expectation.
These are replaced with literal expected job ids taken from `findings.md`
(FR-11 / acceptance criteria). This must happen *before* the mapper changes, so
the new expectations are written against evidence rather than against whatever
the new code happens to produce.

**Layers:**

1. **Table tests** (`atlas-character-factory/job`) — for each version arm, every
   slot in `findings.md` maps to its recorded job id, and a representative set
   of off-carousel ordinals (including `raceIndex` values beyond any carousel's
   range) returns `ok=false`. Expectations are literals from the findings, never
   derived by calling the mapper.
2. **Parity test** (Go + TS, D-9) — both tables match
   `race-carousels.json`.
3. **Seed correspondence test** (D-8) — carousel slots ↔ template rows, both
   genders, per version.
4. **Pre-Big-Bang regression** — the highest-risk property (NFR backward
   compatibility). A test enumerates every currently-seeded
   `(jobIndex, subJobIndex, gender)` row for every pre-Big-Bang template and
   asserts the resulting job id is unchanged from today's behavior. These
   expectations are captured from the *current* code before the refactor and
   frozen as literals — otherwise the regression test refactors along with the
   bug.
5. **Rejection path** (`factory/processor.go`) — an off-carousel ordinal returns
   `ErrInvalidRaceIndex`, not a coerced Beginner, and the failure reaches the
   caller.
6. **Multi-tenancy** (NFR) — two tenants on different versions resolve
   different job ids for the same ordinal within one test, proving no
   package-level version state. The table vars are read-only after init, which
   is what makes this hold.

## 5. Sequencing

The dependency order is strict; the plan phase should not reorder it.

1. IDA investigation → `findings.md` (all ten exports, `gms_12` marked
   unverified). Answers PRD §9.1, §9.3, §9.4. **Gate: no code before this
   lands.**
2. `race-carousels.json` projected from findings.
3. Capture pre-Big-Bang current behavior as frozen test literals; replace the
   two tautological assertions.
4. Constants: new beginner ids if D-5 requires, plus the D-6 beginner-set
   updates.
5. Mapper: `Slot`/`Carousel`/`carouselFor`/`FromIndex`; delete
   `libs/atlas-constants/job.FromIndex`.
6. `processor.go`: resolve-once, `ErrInvalidRaceIndex`, move the check below the
   tenant fetch, reuse `jobId` at the saga payload.
7. Seed data per FR-19/FR-20 + correspondence test.
8. Frontend: version-aware labels + parity test.
9. Flagless `tools/verify.sh` → 0, then code review.

Step 1 is the schedule risk. If a version's ordering cannot be derived from its
export, that column is recorded `unverified` and its arm falls to the default
carousel with a comment — the task does not stall, and it does not guess.

## 6. Risks

| Risk | Mitigation |
|---|---|
| Pre-Big-Bang regression — the highest-risk property in the PRD | Frozen-literal regression test (§4.4) captured *before* the refactor |
| IDA derivation confirms the lead only because it was told the lead | Findings brief states the claim as under test; independent ordering required before comparison (D-4) |
| A version's ordering is underivable | Recorded `unverified`, default arm, comment — not guessed (D-3, D-4) |
| New beginner id missing from `IsBeginner` — silent, downstream | D-6 makes the four edit sites a single decision |
| Frontend drifts from backend after this task | Parity fixture test on both sides (D-9) |
| Seed rows and carousel drift after this task | Correspondence test (D-8) |
| `(1,1)` seed rows exist on three versions but Dual Blade may not be offered | D-5 third outcome obliges removing/annotating them in the same change |

## 7. Traceability

- FR-1, FR-5 → D-2 (no fallback branch exists), D-5
- FR-2, FR-3 → D-3
- FR-4 → D-1 (deletion, verified by `grep -rn "FromIndex" --include="*.go" .`)
- FR-6 … FR-11 → D-4
- FR-12 … FR-15 → D-5, D-6
- FR-16 … FR-18 → D-7
- FR-19 … FR-21 → D-8
- FR-22, FR-23 → D-9
- NFR multi-tenancy → §4.6; backward compatibility → §4.4; evidence → D-4;
  observability, security → D-7
- PRD §9.1 → D-5; §9.2 → D-1; §9.3 → D-5; §9.4 → D-4; §9.5 → D-3/D-4;
  §9.6 → D-10

## 8. Out of scope

Unchanged from PRD §2 non-goals, plus D-10 (no data repair). No packet-layer
change: `libs/atlas-packet` decodes the field correctly and
`docs/packets/audits/gms_v95/CreateCharacter.md` is ✅. No new endpoint. No
`jobId` field on `template.RestModel`. No Resistance or Dual Blade gameplay.
