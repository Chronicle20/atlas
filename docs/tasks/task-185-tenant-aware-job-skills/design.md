# Tenant-Aware Job Skill Enumeration — Design

Task: `task-185-tenant-aware-job-skills`
PRD: [`prd.md`](prd.md) (approved)
Status: Draft
Created: 2026-07-27

---

## 1. Shape of the change

The job→skill mapping stops being a compiled-in table and becomes a per-tenant
document derived from that tenant's own `Skill.wz`. Four pieces move:

| Piece | Before | After |
|---|---|---|
| Mapping source | `libs/atlas-constants/job/constants.go` (82 literals) | `JOB` documents in `documents`, written at ingest |
| Read path | `constJob.Jobs[id].Skills()` | `document.Storage[string, job.RestModel]` |
| Job existence (backend) | key set of `constJob.Jobs` | presence of a `JOB` document |
| Job existence (UI) | `BRANCH_FLOORS` / `NODE_FLOORS` / `floorOf` | `GET /data/jobs` id set |

No relational migration, no new service, no new shared lib, no new worker.

---

## 2. Decisions

### D1 — The `JOB` writer lives in the `job` package, driven by a second pass in the SKILL worker

`job` must import `skill` (the compound list resource embeds `skill.RestModel`
in `included`, FR-4.2). Therefore `skill` **must not** import `job`, which rules
out folding the JOB write into `skill.ProcessorImpl.RegisterSkill`.

The seam that already imports both is `data/workers` (package `workers`). So:

- `job/reader.go` — `Read(l)(ctx)(np model.Provider[xml.Node]) model.Provider[[]RestModel]`,
  parallel to `skill/reader.go`.
- `job/processor.go` — `RegisterJob(path string) error`, parallel to
  `skill.ProcessorImpl.RegisterSkill`.
- `data/workers/skill.go` — one added line, immediately after the existing skill
  registration (`skill.go:49`):

```go
if err := registerAllInDirectory(l, ctx, filepath.Join(root, "Skill.wz"), job.NewProcessor(l, ctx, db).RegisterJob); err != nil {
    return err
}
```

This is the same shape `mobskill` already uses inside the SKILL worker, so
FR-2.1 ("no new entry in `data.Workers`") holds by construction, and FR-2.5
(monolithic v12 `Data.wz`) is inherited for free — the pass reads the same
serialized `root/Skill.wz` tree the skill pass reads.

**Cost:** a second `os.ReadFile` + `xml.Unmarshal` per per-job image (~90 images
per version). `xml.FromPathProvider` (`xml/reader.go:28`) reads eagerly and
returns a `model.FixedProvider`, so re-invoking a *shared* provider is free —
but calling `FromPathProvider` twice on the same path is a genuine second parse.
Accepted: this is a batch ingest job that already serializes the whole archive
to XML on disk first.

**Alternatives rejected:**

- *Write from the worker's existing icon loop* (`skill.go:60-95`), which already
  walks `file.Root().Images()` → `imgID` → `findSub(props, "skill")` → child
  names — i.e. exactly `(jobId, skillIds)` with zero extra parsing. Rejected
  because it works on the WZ object model rather than the serialized XML, so it
  cannot be covered by the fixture-based reader tests §8 requires, and it would
  fork the "how do we enumerate a job's skills" logic across two representations.
- *Single parse, dual register* — hoist `xml.FromPathProvider` into `workers`
  and hand the one provider to both `skill.Register` and `job.Register`.
  Rejected: it forces the `database.ExecuteTransaction` + `NewStorage` plumbing
  currently owned by each package's `Register*` method up into `workers`.

### D2 — Two REST types (resolves PRD §9.3 / FR-4.4)

`document.DbStorage.Add` persists `json.Marshal(jsonapi.MarshalToStruct(m, …))`
(`document/db_storage.go:123-130`). Verified against api2go v1.0.4: both
`marshalSlice` and `marshalStruct` populate `Document.Included` whenever the
model implements `MarshalIncludedRelations` (`jsonapi/marshal.go:150,186,203,433-441`)
— there is no "only when asked" gate at that layer. A single type carrying the
relationship interfaces would therefore write `relationships` (and, once any
detail is attached, `included` stubs) into the stored `content` column, and would
*also* add a `relationships` block to `GET /data/jobs/{jobId}/skills`, whose
shape PRD §5 pins as unchanged.

So the persisted shape and the compound response shape are kept distinct by
being **different Go types**:

```go
// job/rest.go — persisted document AND the /{jobId}/skills response.
// Unchanged from today. Implements MarshalIdentifier + EntityNamer +
// UnmarshalIdentifier only. Deliberately implements NO jsonapi relationship
// interface: this is the type document.DbStorage marshals into `content`.
type RestModel struct {
    Id     uint32   `json:"-"`
    Skills []uint32 `json:"skills"`
}

// job/list_rest.go — the list-endpoint projection. Never persisted.
type ListRestModel struct {
    Id       uint32            `json:"-"`
    Skills   []uint32          `json:"skills"`
    resolved []skill.RestModel `json:"-"` // populated only when include=skills
}
```

`ListRestModel` implements `GetID`/`GetName`/`GetReferences`/`GetReferencedIDs`/
`GetReferencedStructs`. It does **not** implement the `Unmarshal*` /
`SetToManyReferenceIDs` / `SetReferencedStructs` counterparts that
`shops.RestModel` carries — nothing ever unmarshals it, and adding them would
imply it is a persistable/inbound type, which it is not.

A `job.ListFrom(RestModel) ListRestModel` transform is the only bridge.

### D3 — Linkage always; `included` only when requested

`GetReferencedIDs` derives from `Skills` (the stored id list) and is therefore
always present. `GetReferencedStructs` returns `resolved`, which is empty unless
the handler filled it — an empty slice yields no `included` key
(`marshal.go:203`, `if len(includedElements) > 0`).

This is a deliberate divergence from the `shops` reference implementation, where
linkage itself is a function of the `include` decorator
(`shops/resource.go:75-84`) and disappears when `include` is absent. FR-4.3
requires linkage unconditionally, so `GetReferencedIDs` reads the id list, not
the detail slice.

Neither `server.MarshalResponse` nor `server.MarshalPaginatedResponse` knows
about `include` — `jsonapi.FilterSparseFields` handles `fields[type]` only. The
handler parses `r.URL.Query()["include"]` itself, exactly as
`shops/resource.go:77` does.

### D4 — Resolving `included` is one drain, never N lookups

When `include=skills` is present, the handler collects the union of skill ids
across the page's jobs and resolves them with a single
`skill.NewStorage(l, db).DrainAllProvider(ctx)()`, indexed into a
`map[uint32]skill.RestModel`. Per-id `GetById` would be up to ~3,000 storage
round trips for one 50-job page; the registry cache only helps after the first
miss.

When `include` is absent, no skill storage is touched at all — which is the NFR's
explicit requirement ("must not trigger a full skill-table drain when `included`
is not requested").

### D5 — Reader semantics

`job.Read` returns a provider of **0 or 1** `RestModel`:

| Input image | Result | Requirement |
|---|---|---|
| Non-numeric name (`MobSkill.img`, `BFSkill.img`) | `[]` — no document, **no error** | FR-2.3 |
| Numeric name, `skill` child present with children | one model, ids in WZ document order | FR-1.2 |
| Numeric name, `skill` child present but empty | one model, `Skills: []uint32{}` | FR-2.4 |
| Numeric name, `skill` child absent | one model, `Skills: []uint32{}` | FR-2.4 |
| Numeric child of `skill` that is not a number | skipped | — |

Two deliberate divergences from `skill/reader.go`:

- `skill.Read` returns an *error* when `parseJobId` fails, so every non-numeric
  image currently produces a `register MobSkill.img.xml` warn from
  `registerAllInDirectory` (`data/workers/walk.go:45-47`). `job.Read` returns
  an empty list instead, so the new pass adds no warn noise.
- `skill.Read` errors when the `skill` child is missing. `job.Read` treats that
  as "present with zero skills", because FR-2.4 requires the empty case to be
  representable and distinguishable from absence.

Job id comes from the image filename via the same `parseJobId` logic
(`skill/reader.go:26-37`), never from dividing skill ids (FR-2.2). The helper is
duplicated in `job/reader.go` rather than exported from `skill` — a three-line
unexported parser is cheaper than a cross-package dependency in the direction we
just spent D1 avoiding.

Skill id order is WZ document order, not sorted (FR-1.2 says "the ordered
list … found in that job's `Skill.wz` image"). It is deterministic per archive,
so re-ingest is byte-stable and baseline dumps do not churn.

### D6 — Read path and 404 semantics

```go
func NewStorage(l logrus.FieldLogger, db *gorm.DB) *document.Storage[string, RestModel] {
    return document.NewStorage(l, db, GetModelRegistry(), "JOB")
}
```

- `GetSkillsForJob(jobId)` → `Storage.GetById(ctx)(strconv.Itoa(jobId))`.
  `Storage.ByIdProvider` gives registry cache → tenant rows → canonical
  `canonical.TenantId(region, major, minor)` fallback for free
  (`document/storage.go:28-64`), satisfying FR-1.3.
- Any error (including `gorm.ErrRecordNotFound`) → `ok = false` → HTTP 404.
  "Unknown job id" and "job absent from this version" collapse to the same
  response (FR-3.2). `rest.ParseJobId` still 400s on a non-numeric segment.
- `Processor` signature becomes `NewProcessor(l, ctx, db)`; `job/mock` grows the
  matching field. `job.InitResource(db)(si)` mirrors `skill.InitResource`
  (FR-3.3), so `main.go:184` becomes `job.InitResource(db)(GetServer())`.

`document_id` is `uint32` and `DbStorage.Add` derives it by `strconv.Atoi` on
`GetID()` (`db_storage.go:133`) — `job.RestModel.GetID()` already returns the
job id. Job id `0` (Beginner) is a legitimate `document_id`; a round-trip test
covers it.

### D7 — `GET /data/jobs`

Follows `commodity/resource.go:33-55` verbatim (DB-side paging), not
`skill`'s drain-then-`paginate.Slice`:

```go
page, err := paginate.ParseParams(query, paginate.DefaultPageSize, paginate.MaxPageSize)
paged, err := NewStorage(d.Logger(), db).AllPagedProvider(d.Context())(page)()
items := ListFromAll(paged.Items)                  // + resolved skills if include=skills
server.MarshalPaginatedResponse[[]ListRestModel](...)(items, paginate.EnvelopeFor(paged), r)
```

`Storage.AllPagedProvider` carries the same canonical fallback as the per-id path
(`document/storage.go:80-106`), so a version whose data lives only in the
canonical dataset lists correctly. Defaults are `DefaultPageSize=50`,
`MaxPageSize=250`; ~82 jobs means the UI fits in one `page[size]=250` request,
but the UI still follows `links.next` (D10) so the ceiling is not load-bearing.

### D8 — atlas-constants collapse

`job/constants.go` keeps three things: `type Id uint16`, the `Id` constant block,
the `Type` constants — plus `Jobs` rewritten as a literal map:

```go
var Jobs = map[Id]Job{
    BeginnerId: {id: BeginnerId},
    // …
    HeroId:     {id: HeroId, fourthJob: true},
    // …
}
```

`job/model.go` drops the `skills` field, `Skills()`, and `Buffs()`. `Job` stays a
struct — FR-5.4: `FromSkillId` returns `(Job, bool)` and callers use `.Id()`
(`atlas-character/character/processor.go:1045`) and `.IsFourthJob()`
(`atlas-character/skill/model.go:34`), so `map[Id]bool` is not a drop-in.

The `skill` import leaves `constants.go`; `model.go` keeps it for
`mpEaterSkillIds`, `FromSkillId`, and `IdFromSkillId`, so `go.mod` is untouched
(FR-5.6).

**Verified before deletion** (repo-wide enumeration of every `job.*` /
`constJob.*` identifier outside `libs/atlas-constants`):

- Zero external references to any of the 82 exported `Job` value vars. The only
  repo hits for names like `job.BladeRecruit` are two comments
  (`libs/atlas-constants/job/model.go:123`,
  `services/atlas-character-factory/.../job/model.go:13`).
- `Buffs()` has zero callers — the `.Buffs()` hits in `atlas-buffs` belong to
  that service's own `character.Model` (`atlas-buffs/.../character/processor_test.go`).
- `Jobs[id]` existence is used by exactly two call sites, both preserved:
  `atlas-configurations/.../templates/characters/preset/validator.go:76` and
  `.../tenants/characters/preset/validator.go:76`.
- `Skills()` has exactly one caller: `atlas-data/job/processor.go:30` — the one
  being rewritten.

FR-5.7: `libs/atlas-constants/README.md:31` already reads
"`Id`, `Type` | Job / class IDs and type-codes" — no skill data implied, so no
edit is needed. Recorded here so the plan does not go looking for one.

### D9 — FR-1.4 verified, not assumed

- `baseline/dump.go:20-27` — `DumpTables` lists `"documents"` as a whole table;
  there is no per-`type` filter anywhere in `dump.go`, `publish.go`, or
  `restore.go`. `JOB` rows ride along.
- `tenantpurge/purge.go:21` — same, `"documents"` as a whole table.

No change to either package. A test asserting a seeded `JOB` row survives a
publish/restore round trip is listed in §5.

### D10 — UI: `major: number` → `available: ReadonlySet<number>`

`BRANCH_FLOORS`, `NODE_FLOORS`, `floorOf`, and the rationale comment block at
`job-advancement-tree.ts:110-145` are deleted. Every predicate
`floorOf(id) <= major` becomes `available.has(id)`:

| Function | Before | After |
|---|---|---|
| `visibleRoots` | `(major)` | `(available)` |
| `visibleChildrenOf` | `(id, major)` | `(id, available)` |
| `advancementChains` | `(entryId, major)` | `(entryId, available)` |
| `subtreeCount` | `(entryId, major)` | `(entryId, available)` |
| `visibleRailGroups` (`rail-groups.ts:78`) | `(major)` | `(available)` |
| `AdvancementFlow` prop (`advancement-flow.tsx:13`) | `major: number` | `available: ReadonlySet<number>` |

`JOB_GRAPH`, `JOB_ROOTS`, `childrenOf`, `rootOf`, `jobTreePath`, `tierLabel` are
untouched — they encode topology (FR-6.4). Ids present in the tenant data but
absent from `JOB_GRAPH` are simply not rendered; the graph remains the display
authority for structure, the data for existence.

**The one real regression risk: visibility is now async.** `floorOf` was
synchronous, so `JobsPage` could compute `jobIdValid` on first render.
`available` arrives from a query, and `TenantProvider` calls `queryClient.clear()`
on every tenant switch, so `available` is empty during load *and* immediately
after a tenant change. Without a guard, `JobsPage.tsx:50-54`'s normalize effect
would redirect a perfectly valid `/jobs/112` to `/jobs`, and the rail would
render empty.

Mitigation, and it is part of the design, not an afterthought:

- `jobIdValid` and the normalize effect are both gated on `jobsQuery.isSuccess`.
  While pending, no redirect fires and the current `jobId` is retained.
- `BranchRail` renders its existing skeleton while `jobsQuery.isPending`.
- `defaultJobId` is only consulted once `available` is non-empty.
- A `jobsQuery.isError` state renders an error card rather than an empty tree —
  "the backend is down" must not look like "this version has no jobs".

### D11 — UI compound-document support

`types/api/responses.ts`:

```ts
export interface JsonApiResource {
  type: string;
  id: string;
  attributes?: Record<string, unknown>;
}
export interface ApiResponse<T = unknown> {
  data: T;
  included?: JsonApiResource[];
}
```

`lib/api/client.ts` gains one primitive; `api.getList` is left exactly as it is
so no existing caller changes:

```ts
getListDocument: <T>(url, options?): Promise<ApiPagedResponse<T> & { included?: JsonApiResource[] }> =>
  apiClient.get(url, options),
```

`services/api/jobs.service.ts` gains **one** method, not two:

```ts
getJobs(opts?: { includeSkills?: boolean }): Promise<{ jobs: JobResource[]; skillsById: Map<number, SkillResource> }>
```

It requests `page[size]=250` and follows `links.next` until exhausted, so the
250-row ceiling is not load-bearing. `includeSkills` appends `include=skills` and
indexes `included` by id; default `false`.

**Named tension (FR-7.2).** With `includeSkills: false` this serves `JobsPage`.
With `includeSkills: true` it has **no production caller today**, and it should
not acquire one by rewiring `useJobSkillDefinitions`: that hook issues one
React-Query per skill id, keyed `["skill-definition", tenantId, skillId]`, so
definitions are cached per skill *across* jobs. Routing it through
`include=skills` would fetch every skill of every job — thousands of full effect
tables — on first paint, and lose the per-skill cache. The recommendation is to
keep the parameter (the PRD requires the capability, the backend implements it
either way, and a service test exercises it) and to leave
`useJobSkillDefinitions` alone. If the user would rather not ship an
un-consumed flag, dropping `includeSkills` from the UI service costs nothing on
the backend — flagged in §7.

`JobSkillsAddButton` and `useJobSkills` are untouched: they hit
`/jobs/{id}/skills`, whose shape is unchanged, through a query key already
scoped on `activeTenant.id` (FR-7.3).

### D12 — Ingest observability

The SKILL worker counts what the JOB pass wrote and logs it next to the existing
`skill icons: scanned=… extracted=… uploaded=…` line:

```go
l.Infof("job documents: written=%d", jobDocs)
if jobDocs == 0 {
    l.Warnf("Skill.wz ingest produced no JOB documents; /data/jobs will be empty for this tenant")
}
```

`RegisterJob` returns the count via a processor-held counter (or the worker sums
the pass's return values). Silent success is the exact failure mode the rejected
transitional fallback would have hidden (NFR §8).

---

## 3. Behavior change on live versions

The PRD asserts job sets drift across versions. They do, and the drift does not
match the floor table. Probed `GET /api/data/skills/{id}` from inside
`atlas-main` against all 10 live tenants, using one representative skill per job
image (`1000`→job 0, `5000000`→500, `8001000`→800, `9001000`→900, `9101000`→910,
`10001000`→1000, `11001001`→1100, `20001000`→2000, `21000000`→2100,
`20011000`→2001, `22000000`→2200):

| Version | 0 | 500 Pirate | 800 Brigadier | 900 GM | 910 SuperGM | 1000 Cygnus | 2000 Legend | 2001 Evan |
|---|---|---|---|---|---|---|---|---|
| GMS 48 | ✔ | ✘ | ✘ | ✘ | ✘ | ✘ | ✘ | ✘ |
| GMS 61 | ✔ | ✘ | ✘ | ✔ | ✔ | ✘ | ✘ | ✘ |
| GMS 72 | ✔ | ✔ | ✔ | ✔ | ✔ | ✔ | ✘ | ✘ |
| GMS 79 | ✔ | ✔ | ✔ | ✔ | ✔ | ✔ | ✔ | ✘ |
| GMS 83 | ✔ | ✔ | ✔ | ✔ | ✔ | ✔ | ✔ | ✘ |
| GMS 84/87/92/95 | ✔ | ✔ | ✔ | ✔ | ✔ | ✔ | ✔ | ✔ |
| JMS 185 | ✔ | ✔ | ✔ | ✔ | **✘** | ✔ | ✔ | ✔ |

Diffed against `BRANCH_FLOORS` (`0:1, 800:83, 1000:83, 2000:80, 2001:84`) +
`NODE_FLOORS` (`500:62`), the jobs page changes on five of ten tenants:

| Version | Change |
|---|---|
| GMS 48 | GM + Super GM **disappear** (floor 1 showed them; no `9xx` skills exist) |
| GMS 61 | GM + Super GM stay; everything else already agreed |
| GMS 72 | Maple Leaf Brigadier + the whole Cygnus branch **appear** (floors hid them at 83) |
| GMS 79 | Maple Leaf Brigadier + Cygnus **appear** (same cause) |
| JMS 185 | Super GM **disappears**; GM stays |

Every one of these moves toward the data. This is the intended outcome, but it is
a visible change on legacy tenants and belongs in the PR description.

**This table is evidence, not proof.** It probes `SKILL` documents as a *proxy*
for the presence of a per-job image, because no `JOB` document exists anywhere
yet. Two ways the proxy can be wrong: an image can exist with zero skills
(FR-2.4 — the job would then appear with an empty skill list, which `JobsPage`
already renders via its `"empty"` `SkillListState`), and a representative skill
could be absent while its image exists. The runbook's per-version
`GET /api/data/jobs` check (§6) is the authoritative verification; this table is
what sets the expectation it is checked against.

---

## 4. Component inventory

### `services/atlas-data`

| File | Change |
|---|---|
| `job/registry.go` | **new** — `GetModelRegistry()`, `sync.Once` singleton, mirrors `skill/registry.go` |
| `job/reader.go` | **new** — `Read` per D5, plus unexported `parseJobId` |
| `job/rest.go` | unchanged struct; add nothing (D2 keeps it relationship-free) |
| `job/list_rest.go` | **new** — `ListRestModel` + `ListFrom` |
| `job/processor.go` | rewritten — `NewProcessor(l, ctx, db)`, `NewStorage`, `GetSkillsForJob` via storage, `GetAllPaged`, `RegisterJob`, `Register` |
| `job/resource.go` | `InitResource(db)(si)`; add `GET /data/jobs`; existing `/{jobId}/skills` handler now 404s off storage |
| `job/mock/processor.go` | new func fields |
| `data/workers/skill.go` | one added registration pass + the D12 log lines |
| `main.go:184` | `job.InitResource(db)(GetServer())` |
| `docs/rest.md` | document `GET /data/jobs`; update the `/{jobId}/skills` 404 wording |

### `libs/atlas-constants`

`job/constants.go` (1,236 → ~180 lines), `job/model.go` (drop `skills`,
`Skills()`, `Buffs()`). `go.mod`, `README.md`, `advancement.go`, the `Id`/`Type`
blocks: untouched.

### `services/atlas-ui`

`lib/jobs/job-advancement-tree.ts`, `components/features/jobs/rail-groups.ts`,
`components/features/jobs/advancement-flow.tsx`, `pages/JobsPage.tsx`,
`types/api/responses.ts`, `lib/api/client.ts`, `services/api/jobs.service.ts`,
plus a new `lib/hooks/api/useJobs.ts`.

### Unaffected

atlas-character, atlas-skills, atlas-configurations, atlas-channel,
atlas-messages, atlas-consumables, atlas-npc-shops, atlas-cashshop, atlas-login,
atlas-pets, atlas-guilds, atlas-character-factory — all consume only `Id`
constants, `Jobs[id]` existence, `Is`/`IsA`/`GetType`/`Advancement`/
`IsFourthJob`/`FromSkillId`/`IdFromSkillId`/`MpEaterSkillId`, none of which move.

---

## 5. Testing

**`job/reader_test.go`** — XML fixtures per the D5 table: a numeric image with
skills (order preserved), a numeric image with an empty `skill` node, a numeric
image with no `skill` node, `MobSkill.img` (empty result, no error), and a
`skill` node containing a non-numeric child.

**`job/resource_test.go`** — rewritten from the hardcoded-list assertions onto
the sqlite-in-memory + seeded-storage pattern already used by
`skill/resource_test.go:68` (`setupResourceTestDB`, `testDocumentEntity`):

- `/jobs/112/skills` returns the seeded ids; `/jobs/99999/skills` 404s;
  `/jobs/notanumber/skills` 400s (existing cases, new backing).
- Two tenants on different versions, same job id, different seeded skill lists →
  different responses. This is the PRD's headline acceptance criterion.
- Job id `0` round-trips (`document_id = 0`).
- `GET /data/jobs` without `include` — relationship linkage present, **no
  `included` key**, pagination envelope matches `GET /data/skills`.
- `GET /data/jobs?include=skills` — `included` populated, deduped across jobs.

**`job/rest_test.go`** — the FR-4.4 regression guard: `Add` a `JOB` document,
read the raw `content` column back, assert it contains neither `relationships`
nor `included`. This is the test that fails if someone later merges the two
types.

**`baseline` / `tenantpurge`** — a seeded `JOB` row survives publish→restore and
is removed by purge (D9). Verifies FR-1.4 rather than asserting it.

**`libs/atlas-constants/job`** — `Jobs` has 82 entries and exactly 23 with
`fourthJob: true`; every `*Id` constant is a key. Cheap, and it is the only thing
standing between a hand-rewritten map literal and a silent data loss.

**atlas-ui** — `job-advancement-tree.test.ts` rewritten set-driven (the
`BRANCH_FLOORS` equality assertion at line 39 goes away); `JobsPage` tests for
the D10 async states: pending (no redirect, skeleton), success (tree matches the
set), error (error card, not an empty tree), and tenant switch (cache cleared →
back to pending, still no spurious redirect).

---

## 6. Rollout runbook (deliverable of this task; execution is the operator's)

Written to `docs/runbooks/job-document-backfill.md`, alongside
`canonical-version-migration.md`. Hard cutover means every tenant 404s on both
job endpoints until its `Skill.wz` is re-ingested or it is restored from a
re-published baseline.

Per version, for all 11 rows of `deploy/k8s/base/versions.json` (GMS 12.1, 48.1,
61.1, 72.1, 79.1, 83.1, 84.1, 87.1, 92.1, 95.1, JMS 185.1):

1. `POST /api/data/process?scope=shared` with `X-Atlas-Operator: 1` — re-ingest
   the version's canonical dataset from its already-uploaded WZ archives.
2. Poll `GET /api/data/process` until the job reports `succeeded`.
3. **Verify:** `GET /api/data/jobs` for a tenant on that version returns
   `meta.total > 0`, and spot-check that the id set matches the §3 expectation
   for that version. `GET /api/data/status` is *not* sufficient — it reports
   only an aggregate `documentCount` with no per-type breakdown.
4. `POST /api/data/baseline/publish` for the version, so ephemeral PR envs
   (which are baseline-only and fail fast without one) pick up `JOB` documents.

Two version-specific notes:

- **GMS 12.1** has no provisioned tenant and no ingested data in the current
  cluster — confirmed live: the tenant list holds exactly 10 tenants (GMS
  48/61/72/79/83/84/87/92/95 + JMS 185). It must be provisioned and ingested,
  not merely re-published, before step 3 can pass.
- **JMS 185** is expected to come back *without* job 910; **GMS 48** without
  900/910. Per §3 those are correct, not ingest failures.

---

## 7. Open decisions for the user

1. **`includeSkills` on the UI service (D11).** Ship the flag with no production
   caller (recommended — FR-7.2 asks for it, the backend supports it regardless,
   and a service test covers it), or drop it from the UI and keep the capability
   backend-only. Rewiring `useJobSkillDefinitions` through it is **not**
   recommended and is not on the table.
2. **`ListRestModel` name.** It is the list-endpoint projection of `RestModel`.
   `CompoundRestModel` or `JobListRestModel` are equally defensible; happy to
   take a preference before the plan phase pins it.

Neither blocks planning; if no preference is given, the recommendations stand.

---

## 8. Risks

| Risk | Mitigation |
|---|---|
| Async visibility causes spurious `/jobs` redirects and empty rails | D10's `isSuccess` gate + explicit pending/error states, each with a test |
| Hand-rewriting `Jobs` as a literal drops a job or a `fourthJob` marker | §5's 82-entry / 23-marker table test |
| A future edit merges the two REST types and leaks `included` into `content` | §5's raw-`content` assertion in `job/rest_test.go` |
| Operator ingests some versions and not others; those tenants show an empty jobs page | D12's zero-document warn; the runbook's per-version `GET /api/data/jobs` gate |
| Second XML pass slows Skill.wz ingest | ~90 images per version on a batch job; measured alternative (worker icon loop, D1) is available if it ever bites |
| A job image exists with zero skills, so a job appears with no skills | Intended per FR-2.4; `JobsPage` already renders the `"empty"` skill-list state |

---

## 9. Verification gates (CLAUDE.md §Build & Verification)

`go test -race ./...`, `go vet ./...`, `go build ./...` in `libs/atlas-constants`
and `services/atlas-data`; `tools/lint.sh --check` from the repo root;
`tools/redis-key-guard.sh` and `tools/goroutine-guard.sh`; and for atlas-ui,
`npm run build` (not `vitest` alone — the build is what type-checks). No `go.mod`
changes are expected, so `docker buildx bake atlas-data` is required only if the
implementation ends up touching one; the plan should assume it does not and
re-check before opening the PR.
