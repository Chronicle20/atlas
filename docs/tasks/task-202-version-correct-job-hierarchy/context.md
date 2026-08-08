# task-202 — Implementation Context

Companion to [`plan.md`](plan.md). Everything an engineer with zero prior
context needs about the code this task touches, verified against the worktree
on 2026-08-07. Prior artifacts: [`prd.md`](prd.md), [`design.md`](design.md),
[`investigation.md`](investigation.md).

Worktree: `.worktrees/task-202-version-correct-job-hierarchy/`, branch
`task-202-version-correct-job-hierarchy`. Never edit the main repo.

---

## 1. The four defects in one sentence each

1. **Evan blanked** — `job.Read` emits a document for a numeric `Skill.wz`
   image with no `skill` node, so `Skill.wz/Dragon/2200.img` (an Evan/Mir
   *animation* image, same filename as the real job image) wins a
   last-write-wins document upsert and blanks Evan at v0.84+.
2. **Cygnus 4th job over-claimed** — `availability.csv` tracks release at
   *class* granularity, so "Cygnus" being released implies tier 4 is too. It is
   not: `1112/1212/1312/1412/1512` have a present-but-**empty** `skill` node at
   every supported version.
3. **No advancement edges in the API** — `job-availability` returns a flat
   `{id, name}`, so the frontend keeps its own hardcoded parent graph.
4. **The frontend graph is v83-keyed** — `JOB_GRAPH` renders wire ids 500/510
   as "Pirate → Brawler" on a v0.48 tenant, where they are Gm → Super Gm.

---

## 2. Key files, as they stand today

### atlas-data — JOB ingest

| File | What it does now |
|---|---|
| `services/atlas-data/atlas.com/data/job/reader.go:47` | `if ssxml, err := exml.ChildByName("skill"); err == nil { … }` — the `err != nil` branch falls through and emits a model with zero skills. **This is the defect.** |
| `services/atlas-data/atlas.com/data/job/processor.go:72` | `RegisterJob(path) (int, error)` — reads one image, upserts every model, returns the count. Signature unchanged by this task. |
| `services/atlas-data/atlas.com/data/document/db_storage.go` | Upsert on `(tenant_id, type, document_id)` with `DoUpdates: content`. Last write wins — the mechanism that makes the blanking possible. |
| `services/atlas-data/atlas.com/data/data/workers/walk.go:33` | `registerAllInDirectory` is `filepath.WalkDir` — **recursive**, which is why `Dragon/` is visited at all. `imgID(name)` trims a `.img` suffix and parses the rest. |
| `services/atlas-data/atlas.com/data/data/workers/skill.go:68-73,119-143` | The JOB pass, `countingRegister`, `logJobDocCount("written=%d")`. |

`skill.Read` (the sibling) *hard-errors* on a missing `skill` node and skill
documents are keyed per skill id, which is why only the JOB document is
destroyable this way.

**Mirror to copy:** `skill.StatsAccumulator` (used at `skill.go:54-56`) already
has the `Wrap`/`Log` shape Task 2 needs. `defer stats.Log(l)` immediately after
declaring the accumulator, so the summary survives a walk-level error.

**Test harness:** `job/reader_test.go` has `readAll` (byte-array provider) and
`writeTempImage` (fresh `t.TempDir()` per call). `job/processor_test.go` /
`job/resource_test.go` have `setupResourceTestDB(t)` and `testCtx(t, uuid,
region, major, minor)`. Package `workers` has **no** DB harness — that is why
the walk-order test belongs in `job/`.

### libs/atlas-constants — availability + identities

| File | What it does now |
|---|---|
| `libs/atlas-constants/gen/availability.go:57` | `classOf(domain, canonicalToken)` — version-independent class labels. `1000..1599` → `"Cygnus"` wholesale. The `skill` domain floors by 10000 first, which is what carries a job's class to its skills. |
| `docs/tasks/task-187-version-aware-id-semantics/audit/availability.csv` | 99 data rows = 9 classes × 11 versions, `domain` always `job` (the matrix is domain-inert; `AvailabilityMap.Job` and `.Skill` point at the same map). Layout: one block per class, then a trailing `gms,12,1` block for all nine classes. |
| `libs/atlas-constants/gen/main.go` | `cd libs/atlas-constants/gen && go run .` regenerates; `go run . -check` is the drift gate. Never hand-edit `*_gen.go`. |
| `libs/atlas-constants/job/identities_gen.go` | 82 job identities, keyed by canonical (v83-era) wire token. `identityNames` maps each to a display name. |
| `libs/atlas-constants/job/identity.go:33` | `Set{byWire, byIdentity, available, names}` with `Resolve`, `Wire`, `Available`, `Name`, `AvailableIdentities`. `ParentWire` (Task 5) is a method on this type and may read `s.byIdentity` directly. |
| `libs/atlas-constants/constants/registry_gen.go` | Maps each `(region, major, minor)` to `{Skill, Job}` sets via the exported `job.NewSetGMS481()` constructors. Tests use the unexported `newSet_gms_48_1()` form. |

The eleven columns and their two constructor spellings:

| Version | unexported (tests) | exported (registry) |
|---|---|---|
| gms 12.1 | `newSet_gms_12_1()` | `NewSetGMS121()` |
| gms 48.1 | `newSet_gms_48_1()` | `NewSetGMS481()` |
| gms 61.1 | `newSet_gms_61_1()` | `NewSetGMS611()` |
| gms 72.1 | `newSet_gms_72_1()` | `NewSetGMS721()` |
| gms 79.1 | `newSet_gms_79_1()` | `NewSetGMS791()` |
| gms 83.1 | `newSet_gms_83_1()` | `NewSetGMS831()` |
| gms 84.1 | `newSet_gms_84_1()` | `NewSetGMS841()` |
| gms 87.1 | `newSet_gms_87_1()` | `NewSetGMS871()` |
| gms 92.1 | `newSet_gms_92_1()` | `NewSetGMS921()` |
| gms 95.1 | `newSet_gms_95_1()` | `NewSetGMS951()` |
| jms 185.1 | `newSet_jms_185_1()` | `NewSetJMS1851()` |

**Existing test style to match:** `libs/atlas-constants/job/identity_test.go`
uses plain `t.Fatalf` (no testify) and pins the v48/v72 divergence directly.

### atlas-data — the availability API

`services/atlas-data/atlas.com/data/jobavailability/` is four files:
`processor.go` (loops `AvailableIdentities`, resolves each to a wire id),
`rest.go` (`RestModel{Id uint16 json:"-"; Name string}` + `GetName`/`GetID`/
`SetID`), `resource.go` (`GET /data/job-availability`, paginated via
`paginate.ParseParams`/`paginate.Slice`), `resource_test.go`.

The endpoint is read-only, so `SetID` and unmarshalling are unaffected by the
new attributes.

### atlas-ui

| File | What it does now |
|---|---|
| `src/lib/jobs/job-advancement-tree.ts` | `JOB_GRAPH` (82 hardcoded v83 entries), `JOB_ROOTS`, `JOB_LIST`, `jobName`, `childrenOf`, `rootOf`, `visibleRoots`, `visibleChildrenOf`, `jobTreePath`, `advancementChains`, `tierLabel`, `subtreeCount`. Deleted by Task 9. |
| `src/lib/jobs.ts` | `jobNameMap` + `getJobNameById` — a near-duplicate name table with *different* names ("Fire Poison Wizard" vs `JOB_GRAPH`'s "Wizard (F/P)"). Deleted by Task 9. |
| `src/components/features/jobs/rail-groups.ts` | `RAIL_GROUPS` keyed on **wire ids** — the D9 bug. |
| `src/pages/JobsPage.tsx` | `useJobs` + `JOB_GRAPH`; `isSuccess` gating already correct (task-182 D10). |
| `src/services/api/availability.service.ts` | `fetchAllPages` is generic over `{id, attributes:{name}}` and maps to entries inside the pagination loop — Task 7 splits fetch from mapping so the job and skill shapes can diverge. |
| `src/lib/hooks/api/useJobAvailability.ts`, `useJobs.ts` | Both keyed by tenant id, `staleTime` 30 min, `gcTime` 24 h, `enabled: !!tenant?.id`. |
| `src/lib/hooks/usePresetJobOptions.ts` | Falls back to `JOB_LIST` while availability is pending. Task 9 removes the fallback — the one behaviour change outside the Jobs page. |

**Every consumer of the static tables** (verified by grep; `templates/jobNames.ts`'s
`KNOWN_CLASSES`/`templateLabels` and `skill-list.tsx`'s `jobName` *prop* are
unrelated and stay):

- `jobName` — `LeaderboardRow.tsx:80`, `PresetEditor.tsx:73`, `PresetCard.tsx:109`, `JobCombobox.tsx:34`
- `getJobNameById` — `lib/breadcrumbs/routes.ts:173`, `pages/characters-columns.tsx:124`, `pages/GuildDetailPage.tsx:158`
- `JOB_GRAPH` — `JobsPage.tsx:57,84`, `rail-groups.ts:86`, `advancement-flow.tsx:42`
- `jobTreePath` — `rail-groups.ts:57`, `advancement-flow.tsx:74`, `SkillsSection.tsx:16`
- `JOB_LIST` — `usePresetJobOptions.ts:34`

Three of those are **not** components and cannot call a hook:
`lib/breadcrumbs/routes.ts` is a module-level array whose `labelResolver` is
invoked from `getBreadcrumbsFromRoute` (`routes.ts:507`), itself called from the
`useBreadcrumbs` hook — which *can* call `useJobNameLookup()` and pass a context
down. `pages/characters-columns.tsx`'s `getColumns({...})` is called from
`CharactersPage.tsx:58` (a component). `GuildDetailPage.tsx` is itself a page
component, so it uses the hook directly.

---

## 3. Decisions already made — do not re-litigate

| Question | Answer | Where decided |
|---|---|---|
| Does the identity-level parent vary by version? | **No.** Every job row in `divergences.csv` is a wire-binding divergence, never a re-parenting. One version-blind `map[Identity]Identity` + the existing wire binding reproduces all eleven columns. | design D5 (PRD §9 Q1) |
| How is a null parent represented? | `*uint16` with `json:"parent"` → `null`. Beginner is a legitimate wire id 0, so `null` and `0` must not collide. Asserted by test, not assumed. | design D8 (PRD §9 Q2) |
| Rail grouping key? | The canonical `identity`, a **new third attribute** on the resource (an amendment to PRD §5). Wire-keyed rails put Gm in Explorers in pirate colours at v0.48. | design D9 |
| Unavailable parent → ? | The entry becomes a **root**. No walking up to the nearest available ancestor — that invents an edge the game never had. Currently unobservable; pinned by `TestParentWire_D7PolicyGuard`. | design D7 |
| Name-migration scope? | **Complete.** Both static tables deleted, all eleven consumers migrated. A table left in place gets re-adopted. | design D11 (PRD §9 Q3) |
| Cygnus 4th job — delete the identities? | **No.** They are genuinely present in the WZ; only `Available` flips. Deleting them would conflate presence with release, the exact distinction task-187 built. | design D3 |
| FR-2.3 audit method? | **Desk audit** over existing evidence, no MinIO WZ pull. Surface reduces to the `released=true` cells; unevidenced cells closed by live query where provisioned, else recorded `UNVERIFIED`. | design D4 |
| `classOf` match form? | **Explicit token list**, not `t%10 == 2 && t >= 1100`. The arithmetic is exact today but is a fact about today's five branches, not a rule. | design D3 |
| GM rootedness conflict? | `constants.go` (roots) is the **registry** view; the new `parents` table (Beginner → Gm → SuperGm) is the **advancement/display** view. The comment at the definition site is mandatory. | design D6 (FR-3.2) |

---

## 4. Out of scope — restated so execution does not drift

- **Re-ingesting `Skill.wz` and republishing baselines.** The reader fix lands
  with tests; the already-persisted blank Evan documents in live environments
  are untouched. **Evan will still show zero skills everywhere after this task
  merges** — that is the accepted outcome, and it belongs in the PR
  description, not in a silent footnote. The UI will render Evan's stages
  correctly *structurally* (names, parents, visibility) with empty skill lists.
- Skill membership and skill-effect parsing.
- Cygnus 4th-job content the game never shipped.
- Jobs page layout, rail grouping *semantics*, skill-detail panel.
- Resistance, Dual Blade, Mechanic — no identity in the namespace; `classOf`
  never returns them and their CSV rows stay inert.

---

## 5. Dependency order

```
Task 1 ──> Task 2                     (atlas-data ingest)
Task 3 ──> Task 4                     (availability ledger)
Task 3 ──> Task 5 ──> Task 6          (parents relation, then the API)
Task 6 ──> Task 7 ──> Task 8 ──> Task 9   (atlas-ui)
Tasks 1-9 ──> Task 10                 (verification + review)
```

Tasks 1–2, 3–4, and 5–6 are independent of each other and could run in any
order. Tasks 7→8→9 must be sequential: each leaves the tree type-checking, which
a single big-bang UI rewrite would not.

---

## 6. Gotchas worth knowing before you start

- **`npm run test` does not type-check.** Vitest transpiles without checking.
  `npm run build` (`tsc -b && vite build`) is the gate and it covers test files
  under the same strict flags. A branch can pass its whole suite while being
  uncompilable.
- **atlas-ui strict flags that bite here:** `noUncheckedIndexedAccess` (`arr[0]`
  is `T | undefined`) and `exactOptionalPropertyTypes` (an `x?: string`
  property *rejects* an explicitly-passed `undefined`). Both are on.
- **`tools/lint.sh --check` false-fails without nvm on PATH**, and under
  cross-worktree golangci-lint lock contention. Re-run before believing a
  failure.
- **`tools/skill-job-id-guard.sh`** bans raw `==`/`case`/`Is(` comparisons
  against the divergent job wire constants (e.g. `job.GmId` = 500, which means
  Gm at v0.48 and Pirate at v0.61+) outside the version-aware resolver. This
  branch works right next to those constants — run the guard.
- **Never `go work` in this repo** and never hand-edit a `*_gen.go`. Regenerate.
- **`writeTempImage`** uses a fresh `t.TempDir()` per call, so two calls in one
  test do not collide — the Task 1 walk-order test relies on that.
- **`loadReleaseMatrix` only errors when a version has NO rows at all.** An
  omitted `CygnusStage4` row would silently default to `false` — the right
  answer for the wrong reason. Write all eleven explicitly.
- **`audit_validate_test.go` does not enforce a class count**, so adding a
  tenth class label needs no validator change.
- **`imgID` trims only `.img`**, and the register receives basenames like
  `2200.img.xml` — so `imgID(strings.TrimSuffix(base, ".xml"))`.
- **Verify against the merge commit, not the branch tip.** `git fetch origin
  main && git merge origin/main`, then re-run the gates.
