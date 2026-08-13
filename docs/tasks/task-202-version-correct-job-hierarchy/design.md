# Version-Correct Job Hierarchy — Design

Task: task-202-version-correct-job-hierarchy
Status: Approved
Created: 2026-08-07
Inputs: [`prd.md`](prd.md), [`investigation.md`](investigation.md)

---

## 1. Shape of the change

Four independent defects, one shared root cause: *structure that varies by client
version is encoded as a constant somewhere.* The fix is one directional flow —

```
availability.csv          identities.yaml + per-version semantics
  (release ledger)              (identity <-> wire binding)
        \                              /
         \                            /
          v                          v
     libs/atlas-constants/job  ...  Set{byWire, byIdentity, available, names}
                    + parents (NEW, version-blind, identity-keyed)
                              |
                              v
      GET /api/data/job-availability   -> {id: wire, name, parent: wire|null, identity}
                              |
                              v          (intersect)
                    GET /api/data/jobs  -> WZ-present job ids
                              |
                              v
                     useJobGraph()  -> Map<wireId, JobEntry>
                              |
                              v
     JobsPage / rail-groups / advancement-flow / every name consumer
```

Nothing downstream of `libs/atlas-constants` ever compares a version to a
literal. The frontend loses `JOB_GRAPH` and `jobNameMap` outright.

The four defects map to four workstreams that are almost fully independent —
FR-1 (ingest), FR-2 (availability ledger), FR-3 (library + API), FR-4 (UI). Only
FR-4 depends on FR-3; FR-1 and FR-2 can land in any order.

---

## 2. FR-1 — JOB ingest must not blank documents

### D1. Absent `skill` node → no model; present-but-empty → empty model

`job.Read` (`services/atlas-data/atlas.com/data/job/reader.go:47`) currently
writes:

```go
skills := make([]uint32, 0)
if ssxml, err := exml.ChildByName("skill"); err == nil {
    ...
}
return model.FixedProvider([]RestModel{{Id: jobId, Skills: skills}})
```

The `err != nil` branch falls through to emitting a model with zero skills. That
is what lets `Skill.wz/Dragon/2200.img` — an animation image with **no** `skill`
child — upsert over the real Evan document.

Replace with an explicit three-way classification:

| Image | `skill` node | Result |
|---|---|---|
| non-numeric name (`MobSkill.img`) | — | `[]RestModel{}`, no error (unchanged) |
| numeric, no `skill` child | absent | `[]RestModel{}`, no error (**new**) |
| numeric, `skill` child present | present, 0..N children | one `RestModel` with 0..N skills (unchanged) |

The second and third rows differ *only* on node presence, never on child count.
The implementation must branch on the `ChildByName` error and must not collapse
to `len(skills) == 0` — `1112.img` (Cygnus 4th job) is a real image with a real,
empty `skill` node, and FR-1.2 requires it to keep producing a document.

This makes the fix order-independent (FR-1.3): `Dragon/2200.img` never produces a
model at all, so it cannot win a last-write-wins upsert no matter where
`filepath.WalkDir` visits it.

**Rejected alternative — filter the walk.** Teaching
`registerAllInDirectory` to skip subdirectories, or the JOB pass to ignore
`Dragon/`, fixes this archive and not the class of bug. A future archive with a
differently-named subdirectory reintroduces it silently. The reader owning
"what is a job document" is the smaller, more durable contract.

**Rejected alternative — hard-error on a missing `skill` node** (what
`skill.Read` does). That would add a `register 2200.img.xml` warn per dragon
image on every v0.84+ ingest — ten lines of noise describing correct behaviour.
FR-1.4's counters give the same visibility without crying wolf.

### D2. Observability: images seen vs documents written

`logJobDocCount` (`data/workers/skill.go:138`) reports only `written=N`. Silence
about skipped images is what let this live for months.

The worker already has everything it needs to classify without changing the
processor signature: `imgID` (`walk.go:58`) parses a numeric image name from a
path. Extend `countingRegister` into a small accumulator tracking three counters:

- `images` — every `.img.xml` handed to the register
- `numeric` — those whose basename parses as a job id
- `written` — documents actually upserted (the existing `RegisterJob` return)

and emit `job documents: images=%d numeric=%d written=%d skipped=%d` where
`skipped = numeric - written`. The existing zero-documents `Warnf` stays
verbatim. On the v0.84+ archives this prints `skipped=10`, which is the expected
`Dragon/` count and is now diagnosable from logs alone.

No new `data.Workers` entry, no processor signature change.

---

## 3. FR-2 — Tier-accurate release availability

### D3. Split Cygnus 4th job into its own release class

`availability.csv` gates by class label; `classOf`
(`libs/atlas-constants/gen/availability.go:57`) maps the whole `1000`–`1599`
range to `Cygnus`. There is no way to say "Cygnus, but not tier 4" without a new
label.

Add class `CygnusStage4`, matched by an **explicit token list** placed *before*
the `1000..1599` range case:

```go
case t == 1112, t == 1212, t == 1312, t == 1412, t == 1512:
    return "CygnusStage4"
case t >= 1000 && t <= 1599:
    return "Cygnus"
```

An explicit list, not `t%10 == 2 && t >= 1100 && t <= 1599`. The arithmetic form
is currently exact but is a fact about today's five branches, not a rule; the
list is greppable, and a sixth Cygnus branch would have to be added
deliberately rather than inherited by accident.

The FR-2.2 floor-by-10000 relationship is untouched, so the skill domain gets
the split for free: skill token `11121000 / 10000 == 1112` → `CygnusStage4`.

`availability.csv` gains 11 rows (one per provisioned version key), all
`released=false`. `loadReleaseMatrix` only errors when a version has *no* rows at
all, so a missing `CygnusStage4` row would silently default to `false` — the
right answer for the wrong reason. Write all 11 explicitly.

`audit_validate_test.go` enforces per-row shape (non-empty `identityName`,
non-empty `meymink`, `released ∈ {true,false}`, provisioned version key) and does
**not** enforce a class count, so a new label needs no validator change. The
generated `available_*` maps are regenerated and the five `*Stage4` identities
drop out of gms 79/83/84/87/92/95 and jms 185.

**Rejected alternative — remove the `*Stage4` identities from `identities.yaml`.**
They are genuinely *present* in the WZ at those versions (`1112.img` exists, with
an empty `skill` node). Deleting them would conflate presence with release,
which is precisely the two-axis distinction task-187 built. `Resolve`/`Wire` must
keep answering for them; only `Available` flips.

### D4. FR-2.3 audit — desk audit over existing evidence, no WZ walk

Per the design-phase decision, this task does **not** pull eleven `Skill.wz`
archives from MinIO. The audit is a desk pass over evidence already in hand, and
is bounded by one observation that shrinks it dramatically:

> A tier-level over-claim is only observable where the class is
> `released=true`. Where a class is `released=false`, every tier is already
> unavailable and a tier split changes nothing.

That reduces the FR-2.3 surface to the `released=true` cells:

| Class | `released=true` at | Evidence available |
|---|---|---|
| Evan | gms 84/87/92/95, jms 185 | live-baseline sweep (`investigation.md` Finding 4) — WZ carries skills; documents are blank only because of the FR-1 bug |
| Aran | gms 83/84/87/92/95, jms 185 | live-baseline sweep |
| Cygnus | gms 79/83/84/87/92/95, jms 185 | live-baseline sweep (`investigation.md` Finding 3) — tiers 1–3 populated, tier 4 empty |
| Pirate | gms 72/79/83/84/87/92/95, jms 185 | live-baseline sweep covers 79+; gms 72 has **no** sweep evidence |
| GM / SuperGM | every version incl. gms 12/48/61/72 | no sweep evidence below gms 79 |

The unevidenced cells (gms 12/48/61/72 for GM/SuperGM, gms 72 for Pirate) are
cheaply closable *without* a WZ walk by querying a provisioned tenant at that
version: `GET /api/data/jobs/{id}/skills` with the four tenant headers, exactly
the method `investigation.md` already used. Execution should do that where the
version is provisioned. Where it is not, the cell is recorded as
**UNVERIFIED**, with the reason stated — never inferred from the patch log, and
never silently omitted.

Output: `docs/tasks/task-202-version-correct-job-hierarchy/availability-audit.md`,
one section per class, with an explicit verdict per (class, version): CORRECT,
OVER-CLAIMED (with the fix applied), or UNVERIFIED (with the blocker). Classes
found correct get a written verdict — a silent pass is not a result (FR-2.3).

The jms 185 provenance question (PRD §9 Q6) is answered in that document: the
live sweep returned byte-identical Cygnus/Evan results to GMS, which is direct
evidence for the *content* question. The `meymink` caveat about *release timing*
stays as-is — it is a different claim, and the file's convention (WZ wins for
content, patch log wins for dates, FR-2.4) already covers it.

`docs/TODO.md`'s 4th-job preset entry is amended in the same commit: the Cygnus
half is struck with a pointer to this task's finding; the Aran/Legend half is
left untouched (PRD §9 Q5).

---

## 4. FR-3 — Advancement edges in the library and the API

### D5. The parent relation is version-blind and identity-keyed

**This resolves PRD §9 Q1: the identity-level parent does not vary by version.**

Grounding, not assumption. Every job row in
`docs/tasks/task-187-version-aware-id-semantics/audit/divergences.csv` is a
*wire-binding* divergence — the same Identity bound to a different wire id — and
never a structural re-parenting:

```
gms,12,1,500,Gm      gms,61,1,500,Pirate    gms,72,1,1000,Noblesse
gms,48,1,510,SuperGm gms,61,1,900,Gm        gms,84,1,2001,Evan
```

`Gm`'s parent is `Beginner` in every one of those worlds; only the wire id
differs (500 at v0.48, 900 at v0.61+). Identity constants are keyed by canonical
(v83-era) tokens (`identities_gen.go`), so one identity-keyed table plus the
existing per-version `Set.Wire` binding reproduces every version's edges exactly.

So: a single hand-written `parents map[Identity]Identity` in
`libs/atlas-constants/job`, **not** eleven generated per-version maps. Roughly 80
entries against ~900 generated lines, and a new version costs zero new edges.

The table is explicit rather than arithmetic (`id/10*10` etc. would cover the
Explorer branches but not GM, Evan, Aran, or the Cygnus stage lines). Explicit
entries are auditable and greppable; a formula with four exceptions is neither.

Roots (no entry in the table): `Beginner`, `MapleLeafBrigadier`, `Noblesse`,
`Legend`, `Evan`.

API on `Set`:

```go
// ParentIdentity: version-blind structural edge.
func ParentIdentity(id Identity) (Identity, bool)

// ParentWire: this version's edge, in wire ids, availability-filtered.
func (s Set) ParentWire(id Identity) (Id, bool)
```

### D6. FR-3.2 — task-182's GM display convention moves into the library

`libs/atlas-constants/job/constants.go` models GM (900) and Super GM (910) as
independent roots; the game presents them as an advancement line from Beginner,
which is why `JOB_GRAPH` diverged (task-182). That divergence moves into the
shared table: `Gm -> Beginner`, `SuperGm -> Gm`.

This is a deliberate transfer of ownership and it MUST carry a comment at the
definition site saying so — naming task-182 and task-202, and stating that
`constants.go`'s rootedness is the *registry* view while this table is the
*advancement/display* view. Without that note the next reader "corrects" it back
and silently regresses the v0.48 acceptance criterion.

### D7. FR-3.4 — an unavailable parent makes the entry a root

`Set.ParentWire` returns `(0, false)` when the identity's parent is not
`Available` at this version, per the PRD's literal wording. It does **not** walk
up to the nearest available ancestor.

Rationale: reparenting invents an edge the game never had. If a version ever
ships a job whose parent it did not ship, "root" is an honest rendering of a
genuinely odd situation; a synthesised grandparent edge is a lie that renders
convincingly.

The two policies are indistinguishable on today's version set — every available
identity's parent is also available — so the choice is currently unobservable.
Pin that with a test (D12) so the day it *becomes* observable is a test failure
that forces a decision, not a silent rendering change.

### D8. Null-parent representation on the wire

**This resolves PRD §9 Q2.** `RestModel.Parent` is `*uint16` with
`json:"parent"`. A nil pointer marshals to `"parent": null`, unambiguously
distinct from `"parent": 0` (Beginner is a legitimate wire id 0). api2go's
`jsonapi` marshals the attributes object through `encoding/json` on the struct,
so no custom marshaller is needed — but this is asserted by a round-trip test
(D12), not assumed, because it is the one place where being wrong is invisible
until a v0.48 tenant renders Beginner as its own child.

The endpoint is read-only, so `SetID` and unmarshalling are unaffected.

**Rejected — a JSON:API `relationships` entry.** Structurally the more correct
JSON:API modelling for a self-referential edge, but it requires
`GetReferences`/`GetReferencedIDs` plumbing on `RestModel` and changes the
document shape `availabilityService` already consumes. No gain for a scalar
same-type edge; PRD §5 specifies an attribute.

**Rejected — a `-1` sentinel in an `int32`.** Needs its own prose to explain and
is one careless `>= 0` away from being wrong.

### D9. Add an `identity` attribute — an amendment to PRD §5

The PRD specifies `name` and `parent`. Design adds a third attribute:
`identity` (the version-blind canonical token, `uint16`).

**Why it is required, not a nice-to-have.** `RAIL_GROUPS`
(`components/features/jobs/rail-groups.ts:23`) keys rail membership on wire ids —
`500` sits in the Explorers group with the `--c-pirate` accent. On a v0.48
tenant wire id 500 *is* Gm. A wire-keyed rail therefore renders "Gm" inside the
Explorers rail in pirate colours — which violates the PRD's own acceptance
criterion that "the Special group shows Gm with Super Gm beneath it", and does so
by hardcoding a version assumption, which FR-4.7 forbids.

Rail curation ("Explorers", "Cygnus Knights", "Legends", "Special" and their
accent colours) is an editorial grouping that cannot be derived from graph
structure — at v0.48 every one of Warrior, Magician, Bowman, Rogue and Gm is a
depth-1 child of Beginner, and nothing in the shape says which are Explorers.
So the client needs a version-stable key for a job *concept*. The canonical
identity token is exactly that, it already exists, and exposing it is purely
additive.

Alternatives rejected: matching on display name (fragile, and names vary by
version — the exact thing being fixed); moving rail curation server-side
(atlas-data has no business owning UI accent colours).

The processor already holds the `Identity` in its loop
(`jobavailability/processor.go:36`), so this costs one struct field and one
assignment.

Resulting resource:

```json
{ "type": "job-availability", "id": "500",
  "attributes": { "name": "Gm", "parent": 0, "identity": 900 } }
```

on gms 48.1, versus

```json
{ "type": "job-availability", "id": "500",
  "attributes": { "name": "Pirate", "parent": 0, "identity": 500 } }
```

on gms 72.1. Pagination and ordering (ascending by wire id) are unchanged
(FR-3.5). `usePresetJobOptions` reads only `id` and `name`, so it is unaffected
(FR-3.3 additivity).

---

## 5. FR-4 — The UI sources everything from the API

### D10. One hook owns the graph; the tree helpers become pure functions over it

New `src/lib/hooks/api/useJobGraph.ts` composes the two queries and returns the
intersected, re-rooted graph:

```ts
export interface JobNode {
  id: number;          // wire id
  identity: number;    // canonical token — rail curation keys on this
  name: string;        // version-correct, from availability
  parent: number | null;
}

export interface JobGraphResult {
  graph: ReadonlyMap<number, JobNode>;
  isSuccess: boolean;  // both queries succeeded
  isPending: boolean;
  isError: boolean;    // either query failed
}
```

Construction:

1. Start from the availability entries (FR-4.2: names and parents come from here).
2. Keep only ids also present in `GET /api/data/jobs` (FR-4.1: the intersection).
3. Null any `parent` whose id did not survive step 2 — the same re-rooting rule as
   D7, applied a second time because the intersection can drop a parent whose
   child survives. Applying it in exactly one place (graph construction) means no
   downstream helper ever has to handle a dangling edge.

`job-advancement-tree.ts` keeps its helpers but they take the graph as their
first argument: `childrenOf(graph, id)`, `rootOf(graph, id)`,
`jobTreePath(graph, id)`, `tierLabel(graph, id)`, `advancementChains(graph, id)`,
`subtreeCount(graph, id)`. They stay pure and unit-testable against a literal
fixture graph.

`JOB_GRAPH`, `JOB_ROOTS`, `JOB_LIST`, `jobName`, `visibleRoots` and
`visibleChildrenOf` are **deleted**. The last two collapse naturally: the graph
*is* the available set, so the separate `available: ReadonlySet<number>` second
parameter threaded through six helpers disappears — a real simplification, not
just a relocation.

### D11. Name migration is complete; both static tables are deleted

Per the design-phase decision, every consumer migrates and both
`JOB_GRAPH.name`/`jobName` and `lib/jobs.ts`'s `jobNameMap`/`getJobNameById` are
removed. They are wrong at v0.48 in exactly the same way as the Jobs page, and a
table left in place gets re-adopted by the next feature.

| Consumer | Today | After |
|---|---|---|
| `pages/JobsPage.tsx` | `JOB_GRAPH` | `useJobGraph()` |
| `components/features/jobs/rail-groups.ts` | `JOB_GRAPH` + wire-id keys | graph + **identity** keys (D9) |
| `components/features/jobs/advancement-flow.tsx` | `JOB_GRAPH[id]?.name` | node from graph |
| `components/features/rankings/LeaderboardRow.tsx` | `jobName` | `useJobGraph()` name lookup |
| `components/features/characters/presets/PresetEditor.tsx` | `jobName` | same |
| `components/features/characters/presets/PresetCard.tsx` | `jobName` | same |
| `components/features/characters/presets/JobCombobox.tsx` | `jobName` fallback | same |
| `components/features/characters/SkillsSection.tsx` | `jobTreePath` | graph-parameterised `jobTreePath` |
| `lib/breadcrumbs/routes.ts` | `getJobNameById` | see below |
| `pages/characters-columns.tsx` | `getJobNameById` | same |
| `pages/GuildDetailPage.tsx` | `getJobNameById` | same |

Three of those are **not** React components and cannot call a hook:
`breadcrumbs/routes.ts` is a plain resolver table, and the two column/cell
builders run inside table definitions. For those, expose a small
`useJobNameLookup()` returning `(id: number) => string` from the same cached
query, and have the call sites take the resolver as a parameter rather than
importing a module-level function. The breadcrumb resolver already receives
params; it gains the lookup the same way. Where a name is genuinely needed
before the query resolves, the fallback is `Job ${id}` — the existing
`jobName` fallback, which is honest about not knowing rather than asserting a
v83 name.

`usePresetJobOptions` currently falls back to `JOB_LIST` while availability is
pending, on the reasoning that "a picker must never be blank". With `JOB_LIST`
deleted that fallback becomes an empty list plus the query's pending state,
which the combobox renders as a loading affordance. This is strictly more
correct — the old fallback offered v83 job names to a v0.48 tenant — and it is
the one behaviour change outside the Jobs page. Call it out in the PR.

### D12. Query gating — task-182 D10 preserved across two queries

The existing single-query discipline generalises:

- `isSuccess` = both queries succeeded. Only then may the page redirect an
  invalid `/jobs/{id}`, or treat an absent id as absent.
- `isPending` = either still pending. The route param is retained untouched — no
  redirect (FR-4.4). This is the state immediately after every tenant switch,
  because `TenantProvider` calls `queryClient.clear()`.
- `isError` = either failed → the existing `jobs-load-error` card renders and
  nothing else does (FR-4.5).

Both queries are keyed by tenant id (`jobAvailabilityKeys.list(tenantId)` and
the jobs query's existing key), so no cross-tenant bleed is possible even
without the cache clear.

Rail groups drop empty groups against the intersected set (FR-4.6), which is
already `visibleRailGroups`' behaviour — it just receives the graph instead of a
raw id set.

---

## 6. Testing

Every WZ-derived fact in the PRD becomes a pinned test, because every regression
here is silent by nature.

**atlas-data — `job/reader_test.go`**

- numeric image, no `skill` node → zero models
- numeric image, present-but-empty `skill` node → one model, `Skills` empty
  (distinct test, distinct fixture — these two must never share a helper)
- non-numeric image → zero models, no error (existing, unchanged)

**atlas-data — worker-level walk-order test**

A temp tree containing `2200.img.xml` (with skills) and `Dragon/2200.img.xml`
(no `skill` node), registered in **both** orders, must yield document `2200`
carrying the real skills. Order is forced explicitly rather than relying on
`WalkDir`'s ASCII ordering — the point of the test is that the outcome no longer
depends on order.

**atlas-data — summary counters**

Assert the emitted summary distinguishes images/numeric/written/skipped, and
that the zero-documents warn still fires.

**libs/atlas-constants — availability**

- `Set.Available` is false for all five `*Stage4` identities at all 11 version
  keys (table-driven, in the style of `identity_test.go`'s v48/v72 tests)
- Cygnus tiers 1–3 remain available at gms 79+ and unavailable at gms 72 and
  earlier — the no-regression guard on the split
- `Set.Resolve`/`Set.Wire` still answer for `*Stage4` (presence ≠ release)

**libs/atlas-constants — parent relation**

- v0.48: `Gm` parent `Beginner`; wire-level `ParentWire` gives 500 → 0, 510 → 500
- v0.72: wire 500 (Pirate) → 0 and wire 900 (Gm) → 0, independently
- **Invariant across all 11 versions:** for every available identity, if
  `ParentWire` returns a wire id then that wire id belongs to an identity that is
  itself available at that version (FR-3.4)
- **D7 policy guard:** across all 11 versions, no available identity has an
  unavailable parent — i.e. literal-root and nearest-available-ancestor agree
  today. A future version that breaks this fails here and forces the decision.

**atlas-data — `jobavailability/resource_test.go`**

- a root marshals `"parent": null`; Beginner marshals `"id": "0"` with
  `"parent": null`; a child marshals a numeric parent — the D8 assumption,
  asserted
- `identity` is present and is the canonical token, not the wire id (assert on a
  v0.48 fixture where the two differ: wire 500, identity 900)

**atlas-ui**

- `useJobGraph` intersection: an id in availability but not in jobs is absent; an
  id in jobs but not in availability is absent; a surviving child of a dropped
  parent is re-rooted
- v0.48 fixture: Explorers rail has no Pirate entry; Special group contains Gm
  with Super Gm beneath it
- v0.72 fixture: Cygnus Knights group absent entirely
- v0.79 fixture: Legends group absent entirely
- v0.83 fixture: every Cygnus branch ends at 3rd job
- pending state does not redirect a valid `/jobs/112` (task-182 D10)
- either query failing renders `jobs-load-error`
- grep-style guard in review: no `src/**` file compares a tenant major version to
  a literal for job naming, parenting, or visibility

---

## 7. Explicitly out of scope

Carried forward from PRD §2 and §9 Q4, restated so execution does not re-litigate:

- **Re-ingesting `Skill.wz` and republishing baselines.** The FR-1 reader fix
  lands with tests; the already-persisted blank Evan documents in live
  environments are untouched. Evan continues to show zero skills everywhere until
  a separate operational follow-up re-ingests v0.84/87/92/95 + jms 185, republishes
  the baselines, and verifies `GET /api/data/jobs/2200/skills`. This task's UI
  will render Evan's stages correctly *structurally* (names, parents, visibility)
  while its skill lists stay empty.
- Skill membership and skill-effect parsing.
- Cygnus 4th-job content that the game never shipped.
- Jobs page layout, rail grouping semantics, skill-detail panel.
- Resistance, Dual Blade, Mechanic — no identity in the namespace; `classOf`
  never returns them and their `availability.csv` rows stay inert.

---

## 8. Verification

Per CLAUDE.md §Build & Verification, on the affected modules:

- `go test -race ./...` and `go vet ./...` clean in `libs/atlas-constants`,
  `libs/atlas-constants/gen`, and `services/atlas-data`
- `npm run build` clean in `services/atlas-ui` (it type-checks tests; `npm run
  test` alone does not)
- `tools/lint.sh --check` clean from the repo root
- `docker buildx bake atlas-data` if any `go.mod` was touched
- `superpowers:requesting-code-review` run **before** the PR is opened
  (backend + frontend guideline reviewers both apply)
