# Version-Correct Job Hierarchy Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the job advancement hierarchy correct for every supported client version by moving names, parent edges, and release availability out of hardcoded frontend tables and silently-permissive ingest defaults into `libs/atlas-constants` and `GET /api/data/job-availability`.

**Architecture:** Four mostly-independent workstreams, one directional data flow. (A) `atlas-data`'s JOB reader stops emitting a document for a numeric image with no `skill` node, so `Skill.wz/Dragon/2200.img` can no longer blank Evan. (B) `availability.csv` + `classOf` gain a `CygnusStage4` release class marked unreleased at every version. (C) `libs/atlas-constants/job` gains a version-blind, identity-keyed `parents` table plus `Set.ParentWire`, which filters through this version's availability and wire binding. (D) `job-availability` exposes `parent` (nullable wire id) and `identity` (canonical token), and atlas-ui replaces `JOB_GRAPH`/`jobNameMap` with a `useJobGraph()` hook that intersects availability with WZ presence.

**Tech Stack:** Go 1.x (atlas-data, libs/atlas-constants), api2go JSON:API, gorm; TypeScript + React 19 + TanStack React Query 5 + Vitest (atlas-ui).

## Global Constraints

- Worktree root for every path in this plan: `.worktrees/task-202-version-correct-job-hierarchy/`. Never edit the main repo.
- Never write a literal home/absolute path into a committed file. Repo-relative only.
- No `// TODO`, stubbed handler, or 501 in a landed commit.
- Never invent a WZ/game-data value. Every factual claim in `availability-audit.md` cites `investigation.md`, the WZ, or a live query — or is recorded as **UNVERIFIED** with the blocker named.
- Supported version keys, exactly these eleven, in this order: `gms 12.1`, `gms 48.1`, `gms 61.1`, `gms 72.1`, `gms 79.1`, `gms 83.1`, `gms 84.1`, `gms 87.1`, `gms 92.1`, `gms 95.1`, `jms 185.1`.
- Cygnus 4th-job identities, exactly these five: `DawnWarriorStage4` (1112), `BlazeWizardStage4` (1212), `WindArcherStage4` (1312), `NightWalkerStage4` (1412), `ThunderBreakerStage4` (1512).
- `libs/atlas-constants/job/*_gen.go` and `libs/atlas-constants/constants/registry_gen.go` are generated — never hand-edit. Regenerate with `cd libs/atlas-constants/gen && go run .`.
- No file under `services/atlas-ui/src` may compare a tenant major/minor version to a numeric literal to decide job naming, parenting, or visibility (FR-4.7).
- atlas-ui type-checks tests: `npm run build` is the type-check gate, `npm run test` alone is not.
- Final verification gates (CLAUDE.md §Build & Verification), all run before the PR: `go test -race ./...` + `go vet ./...` in `libs/atlas-constants`, `libs/atlas-constants/gen`, `services/atlas-data`; `npm run build` in `services/atlas-ui`; `tools/lint.sh --check` from the repo root; `docker buildx bake atlas-data` if any `go.mod` changed; `superpowers:requesting-code-review` before opening the PR.

---

## File Structure

**Workstream A — atlas-data JOB ingest (FR-1)**

| File | Responsibility |
|---|---|
| `services/atlas-data/atlas.com/data/job/reader.go` | Modify: three-way image classification (no model when `skill` node absent) |
| `services/atlas-data/atlas.com/data/job/reader_test.go` | Modify: invert `TestRead_MissingSkillNode_*`; keep the empty-node case distinct |
| `services/atlas-data/atlas.com/data/job/processor_test.go` | Modify: add the both-walk-orders `Dragon/2200.img` regression test |
| `services/atlas-data/atlas.com/data/data/workers/skill.go` | Modify: replace `countingRegister`/`logJobDocCount` with a `jobStats` accumulator (images/numeric/written/skipped) |
| `services/atlas-data/atlas.com/data/data/workers/skill_test.go` | Modify: replace the `countingRegister` tests with `jobStats` tests |

**Workstream B — availability ledger (FR-2)**

| File | Responsibility |
|---|---|
| `libs/atlas-constants/gen/availability.go` | Modify: `classOf` gains a `CygnusStage4` case before the `1000..1599` range |
| `libs/atlas-constants/gen/availability_test.go` | Modify: pin the new class boundaries in both domains |
| `docs/tasks/task-187-version-aware-id-semantics/audit/availability.csv` | Modify: +11 `CygnusStage4` rows, all `released=false` |
| `libs/atlas-constants/job/version_*_gen.go` | Regenerated: `*Stage4` identities drop out of the available sets |
| `libs/atlas-constants/job/availability_test.go` | Create: `Set.Available` / `Set.Resolve` pins for the split |
| `docs/tasks/task-202-version-correct-job-hierarchy/availability-audit.md` | Create: FR-2.3 per-(class, version) verdicts |
| `docs/TODO.md` | Modify: amend the 4th-job preset entry re Cygnus |

**Workstream C — library parent relation (FR-3.1/3.2/3.4)**

| File | Responsibility |
|---|---|
| `libs/atlas-constants/job/parents.go` | Create: version-blind `parents` table, `ParentIdentity`, `Set.ParentWire` |
| `libs/atlas-constants/job/parents_test.go` | Create: v48/v72 wire-level pins + the cross-version invariants |

**Workstream D — availability API (FR-3.3/3.5)**

| File | Responsibility |
|---|---|
| `services/atlas-data/atlas.com/data/jobavailability/rest.go` | Modify: `Parent *uint16`, `Identity uint16` |
| `services/atlas-data/atlas.com/data/jobavailability/processor.go` | Modify: populate both from `Set.ParentWire` / the loop's `Identity` |
| `services/atlas-data/atlas.com/data/jobavailability/resource_test.go` | Modify: null-parent marshalling + v48 identity≠wire assertions |

**Workstream E — atlas-ui (FR-4)**

| File | Responsibility |
|---|---|
| `services/atlas-ui/src/services/api/availability.service.ts` | Modify: `JobAvailabilityEntry` carries `parent`/`identity`; split resource fetch from per-domain mapping |
| `services/atlas-ui/src/lib/jobs/job-graph.ts` | Create: `JobNode`, `buildJobGraph`, graph-first pure helpers |
| `services/atlas-ui/src/lib/jobs/__tests__/job-graph.test.ts` | Create: helper + intersection + re-rooting tests |
| `services/atlas-ui/src/lib/hooks/api/useJobGraph.ts` | Create: `useJobGraph()` + `useJobNameLookup()` |
| `services/atlas-ui/src/lib/hooks/api/__tests__/useJobGraph.test.tsx` | Create: gating + intersection tests |
| `services/atlas-ui/src/components/features/jobs/rail-groups.ts` | Modify: rail membership keys on `identity`, entries resolved through the graph |
| `services/atlas-ui/src/components/features/jobs/advancement-flow.tsx` | Modify: takes the graph |
| `services/atlas-ui/src/pages/JobsPage.tsx` | Modify: `useJobGraph()` replaces `useJobs` + `JOB_GRAPH` |
| `services/atlas-ui/src/components/features/characters/SkillsSection.tsx` | Modify: graph-parameterised `jobTreePath` |
| `services/atlas-ui/src/components/features/rankings/LeaderboardRow.tsx`, `.../presets/PresetEditor.tsx`, `PresetCard.tsx`, `JobCombobox.tsx` | Modify: `useJobNameLookup()` |
| `services/atlas-ui/src/lib/breadcrumbs/routes.ts`, `src/lib/hooks/useBreadcrumbs.ts` | Modify: `labelResolver` gains a resolver context |
| `services/atlas-ui/src/pages/characters-columns.tsx`, `CharactersPage.tsx`, `GuildDetailPage.tsx` | Modify: name resolver passed in / hook used |
| `services/atlas-ui/src/lib/hooks/usePresetJobOptions.ts` | Modify: drop the `JOB_LIST` fallback |
| `services/atlas-ui/src/lib/jobs.ts`, `src/lib/jobs/job-advancement-tree.ts` (+ its `__tests__`) | Delete |

---

## Task 1: JOB reader — absent `skill` node yields no model

**Files:**
- Modify: `services/atlas-data/atlas.com/data/job/reader.go:14-61`
- Test: `services/atlas-data/atlas.com/data/job/reader_test.go:93-99`

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces: `job.Read` returns `[]RestModel{}` (len 0) for a numeric image whose root has no `skill` child; unchanged one-element result for a present-but-empty `skill` node. `Processor.RegisterJob(path string) (int, error)` therefore returns `0` for such an image — its signature is unchanged.

- [x] **Step 1: Rewrite the missing-`skill`-node test to assert no model**

Replace `TestRead_MissingSkillNode_ProducesEmptyList` in `services/atlas-data/atlas.com/data/job/reader_test.go` with:

```go
// TestRead_MissingSkillNode_ProducesNoModel is the task-202 FR-1.1 fix. A
// numeric image with NO `skill` child is not a job document at all --
// Skill.wz/Dragon/2200.img is an Evan/Mir ANIMATION image that shares the
// real job image's name, and emitting an empty model for it let the
// document upsert (last-write-wins on (tenant, type, document_id)) blank
// the real 2200 document. Contrast TestRead_EmptySkillNode_ProducesEmptyList
// below: a PRESENT but empty `skill` node is a real job with zero skills
// (Cygnus 4th job, 1112.img) and must still produce a document. These two
// cases differ only on node presence and must never share a helper.
func TestRead_MissingSkillNode_ProducesNoModel(t *testing.T) {
	ms := readAll(t, noSkillNodeXML)
	require.Empty(t, ms)
}
```

- [x] **Step 2: Run the test and verify it fails**

```bash
cd services/atlas-data/atlas.com/data && go test ./job/ -run 'TestRead_MissingSkillNode_ProducesNoModel' -v
```

Expected: FAIL — `Should be empty, but was [{900 []}]`.

- [x] **Step 3: Make the reader branch on node presence**

In `services/atlas-data/atlas.com/data/job/reader.go`, replace lines 46-58 (the `skills := make(...)` block through the `return`) with:

```go
			ssxml, err := exml.ChildByName("skill")
			if err != nil {
				// FR-1.1 (task-202): no `skill` node at all means this image is
				// not a job document. Skill.wz/Dragon/ at v0.84+ holds Evan's
				// Mir ANIMATION images named exactly 2200.img-2218.img; emitting
				// an empty model for them let document upsert (last-write-wins
				// on (tenant, type, document_id)) blank the real Evan documents.
				// Returning no model makes the outcome independent of walk order.
				//
				// A PRESENT but empty `skill` node is the opposite case and
				// still yields a model (FR-1.2): 1112.img (Cygnus 4th job) is a
				// real job with zero skills. Never collapse these two into a
				// len(skills) == 0 check.
				l.Debugf("Image [%s] has no skill node; not a JOB document.", exml.Name)
				return model.FixedProvider([]RestModel{})
			}

			skills := make([]uint32, 0)
			for _, sxml := range ssxml.ChildNodes {
				skillId, err := strconv.ParseUint(sxml.Name, 10, 32)
				if err != nil {
					continue
				}
				skills = append(skills, uint32(skillId))
			}
			l.Debugf("Read [%d] skills for job [%d].", len(skills), jobId)

			return model.FixedProvider([]RestModel{{Id: jobId, Skills: skills}})
```

Also update the doc comment's second bullet (currently `reader.go:22-24`) to read:

```go
//   - An ABSENT `skill` child yields NO model (task-202 FR-1.1): the image is
//     not a job document. A PRESENT but empty `skill` child yields a model with
//     an empty skill list (FR-1.2) -- "the job exists with zero skills" stays
//     representable and distinguishable from "the job is absent".
```

- [x] **Step 4: Run the reader tests and verify they pass**

```bash
cd services/atlas-data/atlas.com/data && go test ./job/ -run 'TestRead_' -v
```

Expected: PASS, including `TestRead_EmptySkillNode_ProducesEmptyList` (still one model, id 800, empty skills) and `TestRead_NonNumericImage_ProducesNothingAndNoError`.

- [x] **Step 5: Add the both-walk-orders regression test**

Append to `services/atlas-data/atlas.com/data/job/processor_test.go`:

```go
// dragonAnimationImageXML is the shape of Skill.wz/Dragon/2200.img.xml at
// GMS v0.84+: the same root imgdir NAME as the real job image, an `info`
// node, and NO `skill` node. See docs/tasks/task-202-.../investigation.md
// Finding 4.
const dragonAnimationImageXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="2200.img">
  <imgdir name="info"></imgdir>
</imgdir>`

// realEvanJobImageXML is the top-level Skill.wz/2200.img.xml.
const realEvanJobImageXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="2200.img">
  <imgdir name="info"></imgdir>
  <imgdir name="skill">
    <imgdir name="22000000"/>
    <imgdir name="22001001"/>
  </imgdir>
</imgdir>`

// TestRegisterJob_DragonImageCannotBlankRealDocument pins FR-1.3. The two
// images share a document key (2200), so before the FR-1.1 fix whichever was
// registered LAST won -- and filepath.WalkDir visits "Dragon/" after every
// numeric filename, so the animation image always won. Both orders are
// exercised explicitly: the point is that the outcome no longer depends on
// order at all, which asserting only the ASCII order would not show.
func TestRegisterJob_DragonImageCannotBlankRealDocument(t *testing.T) {
	for _, tc := range []struct {
		name  string
		first string
		last  string
	}{
		{name: "real then dragon", first: realEvanJobImageXML, last: dragonAnimationImageXML},
		{name: "dragon then real", first: dragonAnimationImageXML, last: realEvanJobImageXML},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := setupResourceTestDB(t)
			l, _ := test.NewNullLogger()
			ctx := testCtx(t, uuid.New(), "GMS", 84, 1)
			p := NewProcessor(l, ctx, db)

			_, err := p.RegisterJob(writeTempImage(t, "first.img.xml", tc.first))
			require.NoError(t, err)
			_, err = p.RegisterJob(writeTempImage(t, "last.img.xml", tc.last))
			require.NoError(t, err)

			m, ok := p.GetSkillsForJob(2200)
			require.True(t, ok, "the real 2200 JOB document must exist")
			require.Equal(t, []uint32{22000000, 22001001}, m.Skills)
		})
	}
}
```

Note: `writeTempImage` (declared in `reader_test.go`) uses a fresh `t.TempDir()` per call, so the two calls above do not collide.

- [x] **Step 6: Run the whole job package suite**

```bash
cd services/atlas-data/atlas.com/data && go test -race ./job/...
```

Expected: PASS. Both subtests of `TestRegisterJob_DragonImageCannotBlankRealDocument` pass; `dragon then real` would have passed before the fix, `real then dragon` would not.

- [x] **Step 7: Commit**

```bash
git add services/atlas-data/atlas.com/data/job/
git commit -m "fix(atlas-data): a numeric Skill.wz image with no skill node yields no JOB document

FR-1.1/1.3 (task-202). Skill.wz/Dragon/ at v0.84+ holds Evan animation
images named exactly like the real job images; the reader's swallowed
ChildByName error emitted an empty model for them, which last-write-wins
document upsert used to blank every Evan stage document."
```

---

## Task 2: SKILL worker — images seen vs documents written

**Files:**
- Modify: `services/atlas-data/atlas.com/data/data/workers/skill.go:63-73,119-143`
- Test: `services/atlas-data/atlas.com/data/data/workers/skill_test.go:16-58`

**Interfaces:**
- Consumes: Task 1's `RegisterJob(path string) (int, error)` returning 0 for a skipped image.
- Produces: `jobStats` in package `workers` with `Wrap(rf func(path string) (int, error)) RegisterFunc` and `Log(l logrus.FieldLogger)`. `countingRegister` and `logJobDocCount` are removed.

- [x] **Step 1: Write the failing accumulator tests**

Replace `TestCountingRegister_SumsWrittenDocuments`, `TestCountingRegister_PropagatesErrorAndAddsNothing`, `TestLogJobDocCount_WarnsOnZero` and `TestLogJobDocCount_NoWarnWhenDocumentsWritten` in `services/atlas-data/atlas.com/data/data/workers/skill_test.go` with:

```go
func TestJobStats_CountsImagesNumericAndWritten(t *testing.T) {
	var s jobStats
	rf := s.Wrap(func(path string) (int, error) {
		if filepath.Base(path) == "MobSkill.img.xml" {
			return 0, nil
		}
		if strings.Contains(path, "Dragon") {
			return 0, nil // FR-1.1: numeric image, no skill node, no document
		}
		return 1, nil
	})

	require.NoError(t, rf(filepath.Join("Skill.wz", "112.img.xml")))
	require.NoError(t, rf(filepath.Join("Skill.wz", "MobSkill.img.xml")))
	require.NoError(t, rf(filepath.Join("Skill.wz", "Dragon", "2200.img.xml")))

	require.Equal(t, 3, s.images)
	require.Equal(t, 2, s.numeric)
	require.Equal(t, 1, s.written)
}

func TestJobStats_PropagatesErrorAndCountsNoDocument(t *testing.T) {
	var s jobStats
	rf := s.Wrap(func(path string) (int, error) { return 0, errors.New("boom") })

	require.Error(t, rf("112.img.xml"))
	require.Equal(t, 1, s.images)
	require.Equal(t, 1, s.numeric)
	require.Equal(t, 0, s.written)
}

func TestJobStats_LogReportsSkipped(t *testing.T) {
	l, hook := test.NewNullLogger()
	s := jobStats{images: 98, numeric: 88, written: 78}
	s.Log(l)

	require.Len(t, hook.Entries, 1)
	require.Equal(t, logrus.InfoLevel, hook.Entries[0].Level)
	require.Contains(t, hook.Entries[0].Message, "images=98")
	require.Contains(t, hook.Entries[0].Message, "numeric=88")
	require.Contains(t, hook.Entries[0].Message, "written=78")
	require.Contains(t, hook.Entries[0].Message, "skipped=10")
}

func TestJobStats_LogWarnsOnZeroDocuments(t *testing.T) {
	l, hook := test.NewNullLogger()
	s := jobStats{images: 3, numeric: 0, written: 0}
	s.Log(l)

	require.Len(t, hook.Entries, 2)
	require.Equal(t, logrus.InfoLevel, hook.Entries[0].Level)
	require.Equal(t, logrus.WarnLevel, hook.Entries[1].Level)
}
```

- [x] **Step 2: Run the tests and verify they fail**

```bash
cd services/atlas-data/atlas.com/data && go test ./data/workers/ -run 'TestJobStats' -v
```

Expected: FAIL to build — `undefined: jobStats`.

- [x] **Step 3: Implement the accumulator**

In `services/atlas-data/atlas.com/data/data/workers/skill.go`, replace `countingRegister` and `logJobDocCount` (lines 119-143) with:

```go
// jobStats accumulates the JOB pass's ingest summary across every image the
// registration walk hands it (FR-1.4, task-202). Three counters, not one:
// a numeric image that produces no document is EXPECTED at v0.84+ (the ten
// Skill.wz/Dragon/ animation images, see job.Read's FR-1.1 branch) but a
// silent "written=N" is exactly what let those images blank every Evan
// document for months. skipped = numeric - written makes a recurrence
// diagnosable from logs alone, without a WZ walk.
//
// Mirrors skill.StatsAccumulator's Wrap/Log shape so both passes in
// Skill.Run read the same way.
type jobStats struct {
	images  int // every .img.xml handed to the register
	numeric int // those whose basename parses as a job id
	written int // JOB documents actually upserted
}

// Wrap adapts job.Processor.RegisterJob -- which returns the number of
// documents it wrote -- to the RegisterFunc shape registerAllInDirectory
// expects, accumulating into s. A failing register contributes to images
// and numeric but never to written.
func (s *jobStats) Wrap(rf func(path string) (int, error)) RegisterFunc {
	return func(path string) error {
		s.images++
		if _, ok := imgID(strings.TrimSuffix(filepath.Base(path), ".xml")); ok {
			s.numeric++
		}
		n, err := rf(path)
		if err != nil {
			return err
		}
		s.written += n
		return nil
	}
}

// Log emits the JOB-document ingest summary. A Skill.wz pass that produced
// no JOB documents leaves /data/jobs empty for the tenant, so it escalates
// to warn: silent success here is the failure mode the rejected transitional
// fallback would have hidden (PRD §8 Observability).
func (s *jobStats) Log(l logrus.FieldLogger) {
	l.Infof("job documents: images=%d numeric=%d written=%d skipped=%d", s.images, s.numeric, s.written, s.numeric-s.written)
	if s.written == 0 {
		l.Warnf("Skill.wz ingest produced no JOB documents; /data/jobs will be empty for this tenant")
	}
}
```

Then replace the JOB pass at `skill.go:68-73` with:

```go
	var jobStats jobStats
	defer jobStats.Log(l)
	if err := registerAllInDirectory(l, ctx, filepath.Join(root, "Skill.wz"), jobStats.Wrap(job.NewProcessor(l, ctx, db).RegisterJob)); err != nil {
		return err
	}
```

The `defer` mirrors the SKILL pass above it: the summary is still emitted when the walk itself fails.

Add `"strings"` to `skill.go`'s import block (alongside `"strconv"`).

- [x] **Step 4: Run the tests and verify they pass**

```bash
cd services/atlas-data/atlas.com/data && go test -race ./data/workers/
```

Expected: PASS. The pre-existing `TestSkillWorker_SummaryEmittedOnWalkError` is untouched and still passes.

- [x] **Step 5: Build the whole service and commit**

```bash
cd services/atlas-data/atlas.com/data && go build ./... && go vet ./...
cd - && git add services/atlas-data/atlas.com/data/data/workers/
git commit -m "feat(atlas-data): JOB ingest summary distinguishes images seen from documents written

FR-1.4 (task-202). images/numeric/written/skipped replaces the bare
written=N, so a numeric image that produces no document (Skill.wz/Dragon/)
is visible in the run summary instead of silent."
```

---

## Task 3: Split Cygnus 4th job into its own release class

**Files:**
- Modify: `libs/atlas-constants/gen/availability.go:57-83`
- Modify: `docs/tasks/task-187-version-aware-id-semantics/audit/availability.csv`
- Create: `libs/atlas-constants/job/availability_test.go`
- Test: `libs/atlas-constants/gen/availability_test.go`
- Regenerated (do not hand-edit): `libs/atlas-constants/job/version_*_gen.go`

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces: release class label `"CygnusStage4"` returned by `classOf` for job tokens 1112/1212/1312/1412/1512 and skill tokens `1112xxxx`/`1212xxxx`/`1312xxxx`/`1412xxxx`/`1512xxxx`. `job.Set.Available(DawnWarriorStage4)` etc. is `false` at every version; `Set.Resolve`/`Set.Wire` still answer for them.

- [x] **Step 1: Write the failing generator test**

Append to `libs/atlas-constants/gen/availability_test.go`:

```go
// TestClassOf_CygnusStage4IsItsOwnClass pins task-202 FR-2.1/2.2. The whole
// 1000-1599 range used to map to a single "Cygnus" label, so there was no
// way to express "Cygnus, but not tier 4" -- and Cygnus 4th job is empty in
// the WZ at every supported version (docs/tasks/task-202-.../investigation.md
// Finding 3: 1112/1212/1312/1412/1512 all have a PRESENT but EMPTY skill
// node, contrast 1111 with 218 children).
//
// The five tokens are matched by an explicit list, not by arithmetic. The
// arithmetic form (t%10 == 2 && t >= 1100) is exact today but is a fact
// about today's five branches, not a rule; a sixth Cygnus branch must be
// added deliberately rather than inherited by accident.
func TestClassOf_CygnusStage4IsItsOwnClass(t *testing.T) {
	for _, tok := range []uint64{1112, 1212, 1312, 1412, 1512} {
		if got := classOf("job", tok); got != "CygnusStage4" {
			t.Errorf("classOf(job, %d) = %q, want CygnusStage4", tok, got)
		}
		// FR-2.2: the floor-by-10000 relationship must carry the split into
		// the skill domain for free.
		if got := classOf("skill", tok*10000+1000); got != "CygnusStage4" {
			t.Errorf("classOf(skill, %d) = %q, want CygnusStage4", tok*10000+1000, got)
		}
	}
}

// TestClassOf_CygnusTiers1To3Unchanged is the no-regression guard on the
// split: everything else in 1000-1599 must still be plain Cygnus.
func TestClassOf_CygnusTiers1To3Unchanged(t *testing.T) {
	for _, tok := range []uint64{1000, 1100, 1110, 1111, 1200, 1210, 1211, 1300, 1310, 1311, 1400, 1410, 1411, 1500, 1510, 1511} {
		if got := classOf("job", tok); got != "Cygnus" {
			t.Errorf("classOf(job, %d) = %q, want Cygnus", tok, got)
		}
	}
}
```

- [x] **Step 2: Run it and verify it fails**

```bash
cd libs/atlas-constants/gen && go test -run 'TestClassOf_Cygnus' -v ./...
```

Expected: FAIL — `classOf(job, 1112) = "Cygnus", want CygnusStage4`.

- [x] **Step 3: Add the `CygnusStage4` case to `classOf`**

In `libs/atlas-constants/gen/availability.go`, insert immediately **before** the `case t >= 1000 && t <= 1599:` arm:

```go
	// CygnusStage4 (task-202 FR-2.1): the five Cygnus 4th-job branches are
	// PRESENT in every supported version's Skill.wz but their `skill` node is
	// empty -- the tier was never released in the version range we support
	// (docs/tasks/task-202-version-correct-job-hierarchy/investigation.md
	// Finding 3, corroborated by a live GET /api/data/jobs/{id}/skills sweep
	// across gms 79/83/84/87/92/95 and jms 185). Presence != release, so the
	// identities stay in identities.yaml and Set.Resolve/Set.Wire keep
	// answering for them; only Set.Available flips.
	//
	// Deliberately an explicit token list rather than arithmetic: the list is
	// greppable and a sixth Cygnus branch must be added on purpose. This arm
	// MUST stay above the 1000..1599 range arm below.
	case t == 1112, t == 1212, t == 1312, t == 1412, t == 1512:
		return "CygnusStage4"
```

- [x] **Step 4: Run the generator tests and verify they pass**

```bash
cd libs/atlas-constants/gen && go test -run 'TestClassOf' -v ./...
```

Expected: PASS.

- [x] **Step 5: Add the eleven `CygnusStage4` rows to availability.csv**

In `docs/tasks/task-187-version-aware-id-semantics/audit/availability.csv`, insert ten rows immediately after the last `Cygnus` row of the main block (the `jms,185,1,job,Cygnus,...` row), and one row in the trailing `gms,12,1` block after its `Cygnus` row — matching the file's existing layout. All eleven use the identical `meymink` cell:

```
gms,48,1,job,CygnusStage4,false,"Cygnus 4th job is unreleased at every supported version: WZ evidence, not patch log. 1112/1212/1312/1412/1512 carry a PRESENT but EMPTY skill node in v0.84 Skill.wz and in the v83 extracted XML (1112.img.xml is 255 bytes), while 1111 carries 218 skill children; a live GET /api/data/jobs/{id}/skills sweep on gms 79/83/84/87/92/95 + jms 185 returns {\"skills\":[]} for all five. Per FR-2.4 the WZ wins for content questions. See docs/tasks/task-202-version-correct-job-hierarchy/investigation.md Finding 3."
```

Write all eleven explicitly (`gms,12,1` · `gms,48,1` · `gms,61,1` · `gms,72,1` · `gms,79,1` · `gms,83,1` · `gms,84,1` · `gms,87,1` · `gms,92,1` · `gms,95,1` · `jms,185,1`), every one `job,CygnusStage4,false`. Do **not** rely on `loadReleaseMatrix`'s missing-class default: it only errors when a version has *no* rows at all, so an omitted row would silently default to `false` — the right answer for the wrong reason.

- [x] **Step 6: Verify the CSV parses and the audit validator passes**

```bash
cd libs/atlas-constants/gen && go test -run 'TestAudit|TestBuildAvailability' -v ./...
```

Expected: PASS. `audit_validate_test.go` enforces per-row shape (non-empty `identityName`, non-empty `meymink`, `released ∈ {true,false}`, provisioned version key) and does not enforce a class count, so no validator change is needed.

- [x] **Step 7: Regenerate the per-version sets**

```bash
cd libs/atlas-constants/gen && go run . && go run . -check
git -C ../../.. status --short libs/atlas-constants/
```

Expected: `go run . -check` exits 0; `git status` shows modified `libs/atlas-constants/job/version_gms_79_1_gen.go`, `..._83_1_...`, `..._84_1_...`, `..._87_1_...`, `..._92_1_...`, `..._95_1_...`, `..._jms_185_1_...` (the seven versions where Cygnus was `released=true`). The pre-v79 files should be unchanged — Cygnus was already `false` there.

- [x] **Step 8: Pin the availability split with library tests**

Create `libs/atlas-constants/job/availability_test.go`:

```go
package job

import "testing"

// cygnusStage4 is the five Cygnus 4th-job identities, unreleased at every
// supported version (task-202 FR-2.1).
var cygnusStage4 = []Identity{
	DawnWarriorStage4, BlazeWizardStage4, WindArcherStage4,
	NightWalkerStage4, ThunderBreakerStage4,
}

// allVersionSets is every provisioned version column, in the order
// docs/tasks/task-187-version-aware-id-semantics uses.
func allVersionSets() map[string]Set {
	return map[string]Set{
		"gms 12.1":  newSet_gms_12_1(),
		"gms 48.1":  newSet_gms_48_1(),
		"gms 61.1":  newSet_gms_61_1(),
		"gms 72.1":  newSet_gms_72_1(),
		"gms 79.1":  newSet_gms_79_1(),
		"gms 83.1":  newSet_gms_83_1(),
		"gms 84.1":  newSet_gms_84_1(),
		"gms 87.1":  newSet_gms_87_1(),
		"gms 92.1":  newSet_gms_92_1(),
		"gms 95.1":  newSet_gms_95_1(),
		"jms 185.1": newSet_jms_185_1(),
	}
}

// TestAvailable_CygnusStage4NeverReleased pins FR-2.1 across every column.
func TestAvailable_CygnusStage4NeverReleased(t *testing.T) {
	for name, s := range allVersionSets() {
		for _, id := range cygnusStage4 {
			if s.Available(id) {
				t.Errorf("%s: Available(%d) = true, want false -- Cygnus 4th job shipped no skills at any supported version", name, id)
			}
		}
	}
}

// TestResolveWire_CygnusStage4StillPresent -- presence != release. The
// identities are genuinely in the WZ (1112.img exists, with an empty skill
// node), so Resolve/Wire must keep answering for them wherever the Cygnus
// class itself is bound. Deleting them from identities.yaml instead would
// have conflated the two axes task-187 built.
func TestResolveWire_CygnusStage4StillPresent(t *testing.T) {
	v83 := newSet_gms_83_1()
	for _, id := range cygnusStage4 {
		w, ok := v83.Wire(id)
		if !ok {
			t.Fatalf("v83 Wire(%d) should still resolve -- presence is not release", id)
		}
		if got, ok := v83.Resolve(w); !ok || got != id {
			t.Fatalf("v83 Resolve(%d) = (%v, %v), want (%d, true)", w, got, ok, id)
		}
	}
}

// TestAvailable_CygnusTiers1To3NoRegression is the guard on the split: the
// tiers that DID ship must be unaffected -- available from gms 79 onward,
// unavailable at gms 72 and earlier.
func TestAvailable_CygnusTiers1To3NoRegression(t *testing.T) {
	tiers := []Identity{
		Noblesse,
		DawnWarriorStage1, DawnWarriorStage2, DawnWarriorStage3,
		BlazeWizardStage1, BlazeWizardStage2, BlazeWizardStage3,
		WindArcherStage1, WindArcherStage2, WindArcherStage3,
		NightWalkerStage1, NightWalkerStage2, NightWalkerStage3,
		ThunderBreakerStage1, ThunderBreakerStage2, ThunderBreakerStage3,
	}
	released := map[string]bool{
		"gms 12.1": false, "gms 48.1": false, "gms 61.1": false, "gms 72.1": false,
		"gms 79.1": true, "gms 83.1": true, "gms 84.1": true, "gms 87.1": true,
		"gms 92.1": true, "gms 95.1": true, "jms 185.1": true,
	}
	for name, s := range allVersionSets() {
		for _, id := range tiers {
			// An identity with no wire binding at this version cannot be
			// available regardless of release class; only assert where the
			// version actually binds it.
			if _, bound := s.Wire(id); !bound {
				continue
			}
			if got := s.Available(id); got != released[name] {
				t.Errorf("%s: Available(%d) = %v, want %v", name, id, got, released[name])
			}
		}
	}
}
```

- [x] **Step 9: Run the library tests**

```bash
cd libs/atlas-constants && go test -race ./... && go vet ./...
```

Expected: PASS, including `constants/golden_test.go` and `gen/drift_test.go`.

- [x] **Step 10: Commit**

```bash
git add libs/atlas-constants/ docs/tasks/task-187-version-aware-id-semantics/audit/availability.csv
git commit -m "fix(atlas-constants): Cygnus 4th job is a distinct, never-released class

FR-2.1/2.2 (task-202). classOf gains CygnusStage4 for tokens
1112/1212/1312/1412/1512 (and their 1112xxxx skill tokens via the
floor-by-10000 relationship), released=false at all eleven version keys.
The identities stay in the namespace -- presence is not release."
```

---

## Task 4: FR-2.3 audit of the remaining release classes

**Files:**
- Create: `docs/tasks/task-202-version-correct-job-hierarchy/availability-audit.md`
- Modify: `docs/TODO.md:405`

**Interfaces:**
- Consumes: Task 3's `CygnusStage4` split (the audit's worked example).
- Produces: a written verdict — `CORRECT`, `OVER-CLAIMED`, or `UNVERIFIED` — for every (class, version) cell where `availability.csv` says `released=true`. Any `OVER-CLAIMED` cell is fixed by the same mechanism as Task 3 (a new class label + eleven CSV rows + regenerate) before this task is complete.

- [x] **Step 1: Enumerate the `released=true` surface**

```bash
awk -F, 'NR>1 && $6=="true" {print $5" "$1" "$2"."$3}' docs/tasks/task-187-version-aware-id-semantics/audit/availability.csv | sort
```

This is the entire audit surface. A tier-level over-claim is only observable where a class is `released=true`; where it is `false`, every tier is already unavailable and a tier split changes nothing. Record the command and its output verbatim in the audit document.

- [x] **Step 2: Close the evidenced cells from `investigation.md`**

`investigation.md`'s live-baseline sweep (`GET /api/data/jobs/{id}/skills` on `atlas-main`, gms 79/83/84/87/92/95 + jms 185) already covers Cygnus, Aran, Evan, and Pirate at those versions. For each such cell write a verdict citing that sweep. Expected verdicts, to be confirmed against the sweep rather than assumed:

- **Cygnus** — OVER-CLAIMED at tier 4, fixed in Task 3; tiers 1–3 CORRECT.
- **Evan** — CORRECT. The WZ carries the skills (`2200 → 22000000, 22001001`; `2218 → 22181000..22181003`); the blank documents are the FR-1 reader bug, not a release over-claim. Say this explicitly — it is the one cell most likely to be misread as an availability defect.
- **Aran** — verdict from the sweep.
- **Pirate** — verdict from the sweep for gms 79+; gms 72 has no sweep evidence, see Step 3.

- [x] **Step 3: Close the unevidenced cells by live query where the version is provisioned**

Unevidenced cells: GM/SuperGM at gms 12/48/61/72, and Pirate at gms 72. Close each one by querying a provisioned tenant at that version — the same method `investigation.md` used. On an `atlas-data` pod (busybox image, so `wget`, not `curl`):

```sh
wget -qO - --header 'TENANT_ID: <uuid>' --header 'REGION: GMS' \
  --header 'MAJOR_VERSION: 72' --header 'MINOR_VERSION: 1' \
  http://localhost:8080/api/data/jobs/500/skills
```

Resolve the tenant uuid from the live tenant list rather than inventing one. Where the version is **not** provisioned, record the cell as `UNVERIFIED` and name the blocker ("gms 12.1 has no provisioned tenant in atlas-main"). Never infer the verdict from the patch log, and never silently omit the cell.

- [x] **Step 4: Answer the jms 185 provenance question (PRD §9 Q6)**

Record in the audit document: the live sweep returned byte-identical Cygnus/Evan results for jms 185 and GMS, which is direct evidence for the *content* question. The `meymink` caveat concerns *release timing*, a different claim; per FR-2.4's convention (WZ wins for content, patch log wins for dates) the caveat stays as written. State this as the answer, not as an open question.

- [x] **Step 5: Write `availability-audit.md`**

One section per class (`GM`, `SuperGM`, `Pirate`, `Aran`, `Evan`, plus `Cygnus`/`CygnusStage4` as the worked example), each with a table of (version, verdict, evidence). Classes found correct get a written verdict — a silent pass is not a result (FR-2.3). Head the document with the Step 1 command + output so the surface is reproducible.

- [x] **Step 6: Amend the stale TODO entry**

In `docs/TODO.md` line ~405, amend the "Non-explorer 4th-job presets" entry: strike the Cygnus half with a pointer to this task's finding, leave the Aran/Legend half untouched.

```markdown
- [ ] **Non-explorer 4th-job presets** — extend `services/atlas-configurations/seed-data/templates/template_gms_83_1.json` with ~~Cygnus /~~ Aran / Resistance / Legend 4th-job presets. (Cygnus 4th job is struck: verified in task-202 that no Cygnus 4th-job skills exist at any supported version — the WZ `skill` node is present but empty at 1112/1212/1312/1412/1512. See `docs/tasks/task-202-version-correct-job-hierarchy/availability-audit.md`.)
```

- [x] **Step 7: Commit**

```bash
git add docs/tasks/task-202-version-correct-job-hierarchy/availability-audit.md docs/TODO.md
git commit -m "docs(task-202): FR-2.3 release-class audit with per-version verdicts"
```

If Step 2 or 3 found an OVER-CLAIMED cell, fix it first — new class label in `classOf`, eleven CSV rows, regenerate, library test — and include that fix in this commit.

---

## Task 5: Version-blind advancement parent relation

**Files:**
- Create: `libs/atlas-constants/job/parents.go`
- Test: `libs/atlas-constants/job/parents_test.go`

**Interfaces:**
- Consumes: Task 3's regenerated `available_*` maps (`Set.Available`).
- Produces, in package `job`:
  - `func ParentIdentity(id Identity) (Identity, bool)` — version-blind structural edge; `(0, false)` for the five roots and for any unknown identity.
  - `func (s Set) ParentWire(id Identity) (Id, bool)` — this version's edge in wire ids, `(0, false)` when `id` is a root, when its parent is not `Available` at this version, or when its parent has no wire binding.

- [x] **Step 1: Write the failing tests**

Create `libs/atlas-constants/job/parents_test.go`:

```go
package job

import "testing"

// TestParentIdentity_Roots -- the five identities with no advancement
// parent. Every other identity must have one.
func TestParentIdentity_Roots(t *testing.T) {
	roots := []Identity{Beginner, MapleLeafBrigadier, Noblesse, Legend, Evan}
	for _, r := range roots {
		if p, ok := ParentIdentity(r); ok {
			t.Errorf("ParentIdentity(%d) = (%d, true), want (0, false) -- %d is a root", r, p, r)
		}
	}
	rootSet := map[Identity]bool{}
	for _, r := range roots {
		rootSet[r] = true
	}
	for id := range identityNames {
		if rootSet[id] {
			continue
		}
		if _, ok := ParentIdentity(id); !ok {
			t.Errorf("ParentIdentity(%d) has no entry -- every non-root identity needs one", id)
		}
	}
}

// TestParentIdentity_GmLineIsRootedAtBeginner pins FR-3.2: task-182's
// display convention now lives here. constants.go models Gm/SuperGm as
// independent roots (the REGISTRY view); the game presents them as an
// advancement line from Beginner (the ADVANCEMENT/DISPLAY view).
func TestParentIdentity_GmLineIsRootedAtBeginner(t *testing.T) {
	if p, ok := ParentIdentity(Gm); !ok || p != Beginner {
		t.Fatalf("ParentIdentity(Gm) = (%v, %v), want (Beginner, true)", p, ok)
	}
	if p, ok := ParentIdentity(SuperGm); !ok || p != Gm {
		t.Fatalf("ParentIdentity(SuperGm) = (%v, %v), want (Gm, true)", p, ok)
	}
}

// TestParentWire_v48GmLine -- the PRD's motivating case. At gms 48.1 wire
// id 500 is Gm and 510 is SuperGm, so the wire-level edges are 500 -> 0 and
// 510 -> 500.
func TestParentWire_v48GmLine(t *testing.T) {
	v48 := newSet_gms_48_1()

	gmWire, ok := v48.Wire(Gm)
	if !ok || gmWire != 500 {
		t.Fatalf("v48 Wire(Gm) = (%v, %v), want (500, true)", gmWire, ok)
	}
	if p, ok := v48.ParentWire(Gm); !ok || p != 0 {
		t.Fatalf("v48 ParentWire(Gm) = (%v, %v), want (0, true) -- Beginner is wire id 0", p, ok)
	}
	if p, ok := v48.ParentWire(SuperGm); !ok || p != 500 {
		t.Fatalf("v48 ParentWire(SuperGm) = (%v, %v), want (500, true)", p, ok)
	}
}

// TestParentWire_v72PirateAndGmAreIndependentRoots -- at gms 72.1 wire id
// 500 is Pirate and Gm has moved to 900. Both are depth-1 children of
// Beginner (wire id 0) and neither borrows the other's edge.
func TestParentWire_v72PirateAndGmAreIndependentRoots(t *testing.T) {
	v72 := newSet_gms_72_1()

	if w, ok := v72.Wire(Pirate); !ok || w != 500 {
		t.Fatalf("v72 Wire(Pirate) = (%v, %v), want (500, true)", w, ok)
	}
	if w, ok := v72.Wire(Gm); !ok || w != 900 {
		t.Fatalf("v72 Wire(Gm) = (%v, %v), want (900, true)", w, ok)
	}
	if p, ok := v72.ParentWire(Pirate); !ok || p != 0 {
		t.Fatalf("v72 ParentWire(Pirate) = (%v, %v), want (0, true)", p, ok)
	}
	if p, ok := v72.ParentWire(Gm); !ok || p != 0 {
		t.Fatalf("v72 ParentWire(Gm) = (%v, %v), want (0, true)", p, ok)
	}
}

// TestParentWire_EdgeAlwaysPointsAtAnAvailableJob is FR-3.4's invariant
// across every version: if ParentWire returns a wire id, that wire id must
// belong to an identity that is itself available at this version. The API
// must never emit an edge pointing at a job absent from its own response.
func TestParentWire_EdgeAlwaysPointsAtAnAvailableJob(t *testing.T) {
	for name, s := range allVersionSets() {
		for _, id := range s.AvailableIdentities() {
			w, ok := s.ParentWire(id)
			if !ok {
				continue
			}
			pid, resolved := s.Resolve(w)
			if !resolved {
				t.Errorf("%s: ParentWire(%d) = %d, which resolves to no identity", name, id, w)
				continue
			}
			if !s.Available(pid) {
				t.Errorf("%s: ParentWire(%d) = %d (identity %d), which is not available", name, id, w, pid)
			}
		}
	}
}

// TestParentWire_D7PolicyGuard pins design D7. ParentWire makes an entry a
// ROOT when its parent is unavailable; it does NOT walk up to the nearest
// available ancestor. The two policies are indistinguishable on today's
// version set -- every available identity's parent is also available -- so
// the choice is currently unobservable. This test makes the day it becomes
// observable a test failure that forces a decision, rather than a silent
// rendering change.
func TestParentWire_D7PolicyGuard(t *testing.T) {
	for name, s := range allVersionSets() {
		for _, id := range s.AvailableIdentities() {
			p, hasParent := ParentIdentity(id)
			if !hasParent {
				continue
			}
			if !s.Available(p) {
				t.Errorf("%s: available identity %d has unavailable parent %d -- literal-root and nearest-available-ancestor now disagree; design D7 must be re-decided before this is silently rendered", name, id, p)
			}
		}
	}
}
```

`allVersionSets()` is declared in `availability_test.go` (Task 3); both files are in package `job`.

- [x] **Step 2: Run the tests and verify they fail**

```bash
cd libs/atlas-constants && go test ./job/ -run 'TestParent' -v
```

Expected: FAIL to build — `undefined: ParentIdentity`.

- [x] **Step 3: Write the parent table**

Create `libs/atlas-constants/job/parents.go`:

```go
package job

// parents is the job ADVANCEMENT relation: which identity a job advances
// from. It is VERSION-BLIND and identity-keyed, and that is a verified
// property, not a convenience: every job row in
// docs/tasks/task-187-version-aware-id-semantics/audit/divergences.csv is a
// WIRE-BINDING divergence (the same Identity bound to a different wire id in
// a different version) and never a structural re-parenting. Gm's parent is
// Beginner in every version we support; only the wire id differs (500 at
// gms 48.1, 900 at gms 61.1+). Set.ParentWire below composes this table with
// the per-version wire binding to reproduce each version's edges exactly, so
// a new version costs zero new edges here.
//
// FR-3.2 / task-182 -- READ BEFORE "CORRECTING" THE Gm AND SuperGm ENTRIES.
// libs/atlas-constants/job/constants.go models Gm (900) and SuperGm (910) as
// independent roots. That is the REGISTRY view and it is correct there. The
// game PRESENTS them as an advancement line beneath Beginner, which is the
// view this table encodes -- the ADVANCEMENT/DISPLAY view. atlas-ui's
// JOB_GRAPH carried this divergence privately (task-182); task-202 moved it
// here so every client gets the same answer. Reverting Gm -> Beginner and
// SuperGm -> Gm to roots silently regresses the v0.48 acceptance criterion
// ("the Special group shows Gm with Super Gm beneath it").
//
// The table is explicit rather than arithmetic. A formula (id/10*10, then
// id/100*100) covers the Explorer branches but not Gm, Evan, Aran, or the
// Cygnus stage lines; a formula with four exceptions is neither auditable
// nor greppable. Roots -- deliberately absent from this map -- are Beginner,
// MapleLeafBrigadier, Noblesse, Legend and Evan.
var parents = map[Identity]Identity{
	// Warrior
	Warrior:      Beginner,
	Fighter:      Warrior,
	Crusader:     Fighter,
	Hero:         Crusader,
	Page:         Warrior,
	WhiteKnight:  Page,
	Paladin:      WhiteKnight,
	Spearman:     Warrior,
	DragonKnight: Spearman,
	DarkKnight:   DragonKnight,

	// Magician
	Magician:                 Beginner,
	FirePoisonWizard:         Magician,
	FirePoisonMagician:       FirePoisonWizard,
	FirePoisonArchMagician:   FirePoisonMagician,
	IceLightningWizard:       Magician,
	IceLightningMagician:     IceLightningWizard,
	IceLightningArchMagician: IceLightningMagician,
	Cleric:                   Magician,
	Priest:                   Cleric,
	Bishop:                   Priest,

	// Bowman
	Bowman:      Beginner,
	Hunter:      Bowman,
	Ranger:      Hunter,
	Bowmaster:   Ranger,
	Crossbowman: Bowman,
	Sniper:      Crossbowman,
	Marksman:    Sniper,

	// Thief
	Rogue:       Beginner,
	Assassin:    Rogue,
	Hermit:      Assassin,
	NightLord:   Hermit,
	Bandit:      Rogue,
	ChiefBandit: Bandit,
	Shadower:    ChiefBandit,

	// Pirate
	Pirate:     Beginner,
	Brawler:    Pirate,
	Marauder:   Brawler,
	Buccaneer:  Marauder,
	Gunslinger: Pirate,
	Outlaw:     Gunslinger,
	Corsair:    Outlaw,

	// Admin -- the task-182 display convention; see the FR-3.2 note above.
	Gm:      Beginner,
	SuperGm: Gm,

	// Cygnus Knights
	DawnWarriorStage1:    Noblesse,
	DawnWarriorStage2:    DawnWarriorStage1,
	DawnWarriorStage3:    DawnWarriorStage2,
	DawnWarriorStage4:    DawnWarriorStage3,
	BlazeWizardStage1:    Noblesse,
	BlazeWizardStage2:    BlazeWizardStage1,
	BlazeWizardStage3:    BlazeWizardStage2,
	BlazeWizardStage4:    BlazeWizardStage3,
	WindArcherStage1:     Noblesse,
	WindArcherStage2:     WindArcherStage1,
	WindArcherStage3:     WindArcherStage2,
	WindArcherStage4:     WindArcherStage3,
	NightWalkerStage1:    Noblesse,
	NightWalkerStage2:    NightWalkerStage1,
	NightWalkerStage3:    NightWalkerStage2,
	NightWalkerStage4:    NightWalkerStage3,
	ThunderBreakerStage1: Noblesse,
	ThunderBreakerStage2: ThunderBreakerStage1,
	ThunderBreakerStage3: ThunderBreakerStage2,
	ThunderBreakerStage4: ThunderBreakerStage3,

	// Aran
	AranStage1: Legend,
	AranStage2: AranStage1,
	AranStage3: AranStage2,
	AranStage4: AranStage3,

	// Evan
	EvanStage1:  Evan,
	EvanStage2:  EvanStage1,
	EvanStage3:  EvanStage2,
	EvanStage4:  EvanStage3,
	EvanStage5:  EvanStage4,
	EvanStage6:  EvanStage5,
	EvanStage7:  EvanStage6,
	EvanStage8:  EvanStage7,
	EvanStage9:  EvanStage8,
	EvanStage10: EvanStage9,
}

// ParentIdentity returns the identity id advances from, or (0, false) if id
// is a branch root (Beginner, MapleLeafBrigadier, Noblesse, Legend, Evan) or
// is not a known identity. Version-blind -- see the parents table doc.
func ParentIdentity(id Identity) (Identity, bool) {
	p, ok := parents[id]
	return p, ok
}

// ParentWire returns THIS version's advancement edge for id, in wire ids.
// It reports (0, false) when id is a root, when id's parent is not Available
// at this version, or when the parent has no wire binding here.
//
// FR-3.4 / design D7: an unavailable parent makes the entry a ROOT. It
// deliberately does NOT walk up to the nearest available ancestor --
// reparenting would invent an edge the game never had, and a synthesised
// grandparent edge is a lie that renders convincingly. If a version ever
// ships a job whose parent it did not ship, "root" is an honest rendering of
// a genuinely odd situation. TestParentWire_D7PolicyGuard fails the day this
// becomes observable, so the choice gets re-made deliberately.
//
// Callers must treat (0, false) as "no parent", never as "parent is wire id
// 0" -- Beginner IS wire id 0 and is a legitimate parent value.
func (s Set) ParentWire(id Identity) (Id, bool) {
	p, ok := parents[id]
	if !ok {
		return 0, false
	}
	if !s.Available(p) {
		return 0, false
	}
	w, ok := s.byIdentity[p]
	if !ok {
		return 0, false
	}
	return w, true
}
```

- [x] **Step 4: Run the tests and verify they pass**

```bash
cd libs/atlas-constants && go test -race ./job/ -run 'TestParent' -v
```

Expected: PASS, all five tests.

- [x] **Step 5: Run the full library suite and commit**

```bash
cd libs/atlas-constants && go test -race ./... && go vet ./...
cd - && git add libs/atlas-constants/job/parents.go libs/atlas-constants/job/parents_test.go
git commit -m "feat(atlas-constants): version-blind job advancement parent relation

FR-3.1/3.2/3.4 (task-202). One identity-keyed table plus Set.ParentWire,
which composes it with the per-version wire binding and availability
filter. Absorbs task-182's Beginner > GM > Super GM display convention
from atlas-ui's JOB_GRAPH."
```

---

## Task 6: `job-availability` exposes `parent` and `identity`

**Files:**
- Modify: `services/atlas-data/atlas.com/data/jobavailability/rest.go:10-13`
- Modify: `services/atlas-data/atlas.com/data/jobavailability/processor.go:30-48`
- Test: `services/atlas-data/atlas.com/data/jobavailability/resource_test.go`

**Interfaces:**
- Consumes: Task 5's `job.Set.ParentWire(id Identity) (job.Id, bool)`.
- Produces: `job-availability` resources with attributes `{ "name": string, "parent": number|null, "identity": number }`. `id` semantics, pagination, and ordering (ascending by wire id) are unchanged. Wire-format example on gms 48.1: `{"id":"500","attributes":{"name":"Gm","parent":0,"identity":900}}`.

- [x] **Step 1: Write the failing resource tests**

Append to `services/atlas-data/atlas.com/data/jobavailability/resource_test.go` (extend the file's existing `jobAvailabilityResponse` attribute struct to carry `Parent *uint16 \`json:"parent"\`` and `Identity uint16 \`json:"identity"\`` first — follow the shape already in that file):

```go
// TestGetJobAvailability_RootMarshalsNullParent asserts design D8: a nil
// *uint16 marshals to JSON null, unambiguously distinct from 0. Beginner IS
// wire id 0, so "parent": 0 and "no parent" must not collide -- this is the
// one place where being wrong is invisible until a v0.48 tenant renders
// Beginner as its own child. Asserted against the raw response body because
// unmarshalling into *uint16 would hide a literal 0 encoded as null.
func TestGetJobAvailability_RootMarshalsNullParent(t *testing.T) {
	w, doc := getJobAvailability(t, "GMS", 48, 1)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"parent":null`)

	var beginner, gm, superGm *struct {
		Parent   *uint16
		Identity uint16
		Name     string
	}
	_ = beginner
	for _, d := range doc.Data {
		switch d.Id {
		case "0":
			require.Nil(t, d.Attributes.Parent, "Beginner is a root; parent must be null, not 0")
		case "500":
			gm = &struct {
				Parent   *uint16
				Identity uint16
				Name     string
			}{d.Attributes.Parent, d.Attributes.Identity, d.Attributes.Name}
		case "510":
			superGm = &struct {
				Parent   *uint16
				Identity uint16
				Name     string
			}{d.Attributes.Parent, d.Attributes.Identity, d.Attributes.Name}
		}
	}

	require.NotNil(t, gm, "gms 48.1 must expose wire id 500")
	require.Equal(t, "Gm", gm.Name)
	require.NotNil(t, gm.Parent)
	require.Equal(t, uint16(0), *gm.Parent, "Gm advances from Beginner (wire id 0)")

	require.NotNil(t, superGm, "gms 48.1 must expose wire id 510")
	require.NotNil(t, superGm.Parent)
	require.Equal(t, uint16(500), *superGm.Parent, "Super Gm advances from Gm, which is wire id 500 at v48")
}

// TestGetJobAvailability_IdentityIsCanonicalNotWire pins design D9 on the
// fixture where the two genuinely differ: at gms 48.1 wire id 500 is Gm,
// whose canonical identity token is 900. The frontend keys rail curation on
// identity precisely so a wire-keyed rail cannot put Gm in the Explorers
// group in pirate colours.
func TestGetJobAvailability_IdentityIsCanonicalNotWire(t *testing.T) {
	_, v48 := getJobAvailability(t, "GMS", 48, 1)
	for _, d := range v48.Data {
		if d.Id == "500" {
			require.Equal(t, uint16(900), d.Attributes.Identity)
			return
		}
	}
	t.Fatal("gms 48.1 response did not contain wire id 500")
}

// TestGetJobAvailability_V72IdentityMatchesWireForPirate -- the contrast
// case. At gms 72.1 wire id 500 IS Pirate, so identity == wire there.
func TestGetJobAvailability_V72IdentityMatchesWireForPirate(t *testing.T) {
	_, v72 := getJobAvailability(t, "GMS", 72, 1)
	for _, d := range v72.Data {
		if d.Id == "500" {
			require.Equal(t, "Pirate", d.Attributes.Name)
			require.Equal(t, uint16(500), d.Attributes.Identity)
			return
		}
	}
	t.Fatal("gms 72.1 response did not contain wire id 500")
}

// TestGetJobAvailability_NoParentPointsOutsideTheResponse is FR-3.4 at the
// API boundary: every non-null parent must be an id the same response also
// returns.
func TestGetJobAvailability_NoParentPointsOutsideTheResponse(t *testing.T) {
	for _, v := range []struct {
		region       string
		major, minor uint16
	}{
		{"GMS", 12, 1}, {"GMS", 48, 1}, {"GMS", 61, 1}, {"GMS", 72, 1},
		{"GMS", 79, 1}, {"GMS", 83, 1}, {"GMS", 84, 1}, {"GMS", 87, 1},
		{"GMS", 92, 1}, {"GMS", 95, 1}, {"JMS", 185, 1},
	} {
		_, doc := getJobAvailabilityQuery(t, v.region, v.major, v.minor, "page[size]=250")
		present := make(map[string]struct{}, len(doc.Data))
		for _, d := range doc.Data {
			present[d.Id] = struct{}{}
		}
		for _, d := range doc.Data {
			if d.Attributes.Parent == nil {
				continue
			}
			pid := strconv.Itoa(int(*d.Attributes.Parent))
			if _, ok := present[pid]; !ok {
				t.Errorf("%s %d.%d: job %s has parent %s, which is absent from the response", v.region, v.major, v.minor, d.Id, pid)
			}
		}
	}
}
```

Simplify the first test's local struct juggling if the file's existing helper types already give you typed access to `doc.Data[i].Attributes` — match the file's conventions rather than importing this shape verbatim. The two assertions that must survive any such simplification are the raw-body `"parent":null` check and the `*Parent == 0` check on Gm.

- [x] **Step 2: Run the tests and verify they fail**

```bash
cd services/atlas-data/atlas.com/data && go test ./jobavailability/ -run 'TestGetJobAvailability_(RootMarshals|IdentityIs|V72Identity|NoParentPoints)' -v
```

Expected: FAIL to build — the attribute struct has no `Parent`/`Identity` field.

- [x] **Step 3: Extend the RestModel**

In `services/atlas-data/atlas.com/data/jobavailability/rest.go`, replace the struct with:

```go
// RestModel is one job identity available (released/playable) at the
// requesting tenant's client version. Id is the version-appropriate wire
// id (job.Id is uint16) -- NOT the version-blind job.Identity token -- so
// the frontend preset selector can round-trip it straight back into
// whatever job-id field the tenant's version expects.
type RestModel struct {
	Id   uint16 `json:"-"`
	Name string `json:"name"`
	// Parent is this version's advancement parent, as a WIRE id, or nil for
	// a branch root. A POINTER, not a plain uint16, because Beginner is a
	// legitimate wire id 0: "parent": 0 and "parent": null must not collide
	// (task-202 design D8, FR-3.3). It also never points at a job absent
	// from this version's response (FR-3.4) -- job.Set.ParentWire filters on
	// availability.
	Parent *uint16 `json:"parent"`
	// Identity is the version-blind canonical job token. Exposed so a client
	// can key version-stable curation (atlas-ui's rail grouping and accent
	// colours) on a job CONCEPT rather than on a wire id that means
	// different things per version -- wire id 500 is Gm at gms 48.1 and
	// Pirate at gms 72.1 (task-202 design D9).
	Identity uint16 `json:"identity"`
}
```

- [x] **Step 4: Populate them in the processor**

In `services/atlas-data/atlas.com/data/jobavailability/processor.go`, replace line 45 with:

```go
		m := RestModel{Id: uint16(wire), Name: set.Job.Name(id), Identity: uint16(id)}
		if pw, ok := set.Job.ParentWire(id); ok {
			// Copy into a local: &pw would alias the loop-scoped value.
			parent := uint16(pw)
			m.Parent = &parent
		}
		ms = append(ms, m)
```

- [x] **Step 5: Run the tests and verify they pass**

```bash
cd services/atlas-data/atlas.com/data && go test -race ./jobavailability/ -v
```

Expected: PASS, including the pre-existing `TestGetJobAvailability_V48HasGmNotPirate` and `TestGetJobAvailability_V72HasPirate` (the added attributes are additive).

- [x] **Step 6: Build, vet, and commit**

```bash
cd services/atlas-data/atlas.com/data && go build ./... && go vet ./...
cd - && git add services/atlas-data/atlas.com/data/jobavailability/
git commit -m "feat(atlas-data): job-availability exposes advancement parent and canonical identity

FR-3.3/3.4 (task-202). parent is a nullable WIRE id (Beginner is a
legitimate wire id 0, so null and 0 must not collide) filtered through
this version's availability; identity is the version-blind canonical
token, so clients can key version-stable curation on a job concept."
```

---

## Task 7: atlas-ui — job graph plumbing (additive)

**Files:**
- Modify: `services/atlas-ui/src/services/api/availability.service.ts`
- Create: `services/atlas-ui/src/lib/jobs/job-graph.ts`
- Create: `services/atlas-ui/src/lib/jobs/__tests__/job-graph.test.ts`
- Create: `services/atlas-ui/src/lib/hooks/api/useJobGraph.ts`
- Create: `services/atlas-ui/src/lib/hooks/api/__tests__/useJobGraph.test.tsx`
- Modify: `services/atlas-ui/src/lib/hooks/api/useJobAvailability.ts` (result type widens)

**Interfaces:**
- Consumes: Task 6's `{ name, parent, identity }` attributes.
- Produces:
  - `JobAvailabilityEntry { id: number; name: string; parent: number | null; identity: number }` from `availability.service.ts`; `AvailabilityEntry { id: number; name: string }` stays as the skill-availability shape.
  - `JobNode { id: number; identity: number; name: string; parent: number | null }` and `JobGraph = ReadonlyMap<number, JobNode>` from `job-graph.ts`.
  - `buildJobGraph(availability: readonly JobAvailabilityEntry[], present: ReadonlySet<number>): JobGraph`.
  - Pure helpers, all graph-first: `childrenOf(graph, id): number[]`, `rootOf(graph, id): number`, `jobTreePath(graph, id): JobNode[]`, `tierLabel(graph, id): string`, `advancementChains(graph, id): number[][]`, `subtreeCount(graph, id): number`, `jobNodeName(graph, id): string`.
  - `useJobGraph(): JobGraphResult { graph: JobGraph; isSuccess: boolean; isPending: boolean; isError: boolean }` and `useJobNameLookup(): (id: number) => string` from `useJobGraph.ts`.
- Nothing is deleted in this task; `job-advancement-tree.ts` and `lib/jobs.ts` are untouched and still compile.

- [x] **Step 1: Widen the availability service**

In `services/atlas-ui/src/services/api/availability.service.ts`, replace the resource/entry types and the fetch helper:

```ts
/**
 * A job-availability JSON:API resource. The resource `id` IS the
 * version-appropriate wire id (not a version-blind identity token) --
 * atlas-data's RestModel.GetID() returns strconv.Itoa(wireId) -- so it
 * round-trips straight back into whatever job id field the tenant's version
 * expects. `attributes.name` is the version's display name (wire id 500 is
 * "Gm" pre-v0.61, "Pirate" at v0.61+).
 *
 * `parent` is the advancement parent as a WIRE id, or null for a branch
 * root. Null and 0 are distinct: Beginner is a legitimate wire id 0.
 * `identity` is the version-blind canonical token -- key version-stable
 * curation (rail grouping, accents) on THIS, never on the wire id.
 */
export interface JobAvailabilityResource {
  id: string;
  type: string;
  attributes: { name: string; parent: number | null; identity: number };
}

export interface SkillAvailabilityResource {
  id: string;
  type: string;
  attributes: { name: string };
}

export interface AvailabilityEntry {
  id: number;
  name: string;
}

export interface JobAvailabilityEntry extends AvailabilityEntry {
  parent: number | null;
  identity: number;
}
```

Replace `toEntries`/`fetchAllPages` with a resource-level fetch plus per-domain mapping:

```ts
/**
 * Follows links.next until exhausted, collecting every resource across all
 * pages. Job/skill availability is a version's RELEASED set, which for
 * skills can exceed a single page[size]=250 response, so links.next MUST be
 * followed rather than trusting the first page alone.
 */
async function fetchAllResources<T extends { id: string }>(
  startUrl: string,
): Promise<T[]> {
  let url: string | undefined = startUrl;
  const resources: T[] = [];
  const visited = new Set<string>();

  while (url) {
    if (visited.has(url) || visited.size >= MAX_PAGES) {
      throw new Error(
        `availabilityService: aborting pagination after ${visited.size} page(s) — ` +
          `links.next did not advance (url: ${url}). The backend is misbehaving.`,
      );
    }
    visited.add(url);

    const doc: ApiPagedResponse<T> = await api.getListDocument<T>(url);
    resources.push(...(doc.data ?? []));
    url = doc.links?.next;
  }

  return resources;
}

export const availabilityService = {
  /** The tenant version's RELEASED job identities: wire id, version-correct name, advancement parent, canonical identity. */
  async getJobAvailability(): Promise<JobAvailabilityEntry[]> {
    const resources =
      await fetchAllResources<JobAvailabilityResource>(JOB_BASE_PATH);
    return resources.map((r) => ({
      id: Number(r.id),
      name: r.attributes.name,
      parent: r.attributes.parent,
      identity: r.attributes.identity,
    }));
  },

  /** The tenant version's RELEASED skill identities: wire id + version-correct name. */
  async getSkillAvailability(): Promise<AvailabilityEntry[]> {
    const resources =
      await fetchAllResources<SkillAvailabilityResource>(SKILL_BASE_PATH);
    return resources.map((r) => ({ id: Number(r.id), name: r.attributes.name }));
  },
};
```

In `services/atlas-ui/src/lib/hooks/api/useJobAvailability.ts`, change the import and result type from `AvailabilityEntry` to `JobAvailabilityEntry`:

```ts
import {
  availabilityService,
  type JobAvailabilityEntry,
} from "@/services/api/availability.service";

export interface JobAvailabilityResult {
  jobs: JobAvailabilityEntry[];
}
```

- [x] **Step 2: Write the failing graph tests**

Create `services/atlas-ui/src/lib/jobs/__tests__/job-graph.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import {
  advancementChains,
  buildJobGraph,
  childrenOf,
  jobNodeName,
  jobTreePath,
  rootOf,
  subtreeCount,
  tierLabel,
  type JobGraph,
} from "@/lib/jobs/job-graph";
import type { JobAvailabilityEntry } from "@/services/api/availability.service";

// A v0.48-shaped fixture: wire id 500 is Gm (canonical identity 900) with
// Super Gm beneath it, and there is no Pirate branch at all.
const V48_AVAILABILITY: JobAvailabilityEntry[] = [
  { id: 0, name: "Beginner", parent: null, identity: 0 },
  { id: 100, name: "Warrior", parent: 0, identity: 100 },
  { id: 110, name: "Fighter", parent: 100, identity: 110 },
  { id: 111, name: "Crusader", parent: 110, identity: 111 },
  { id: 500, name: "Gm", parent: 0, identity: 900 },
  { id: 510, name: "Super Gm", parent: 500, identity: 910 },
];
const V48_PRESENT = new Set([0, 100, 110, 111, 500, 510]);

function v48(): JobGraph {
  return buildJobGraph(V48_AVAILABILITY, V48_PRESENT);
}

describe("buildJobGraph", () => {
  it("keeps only ids present in BOTH availability and the WZ job set", () => {
    const graph = buildJobGraph(V48_AVAILABILITY, new Set([0, 100, 110]));
    expect([...graph.keys()].sort((a, b) => a - b)).toEqual([0, 100, 110]);
  });

  it("drops an id the WZ has but availability does not", () => {
    const graph = buildJobGraph(V48_AVAILABILITY, new Set([0, 100, 1000]));
    expect(graph.has(1000)).toBe(false);
  });

  it("re-roots a surviving child whose parent the intersection dropped", () => {
    // 110 survives, its parent 100 does not.
    const graph = buildJobGraph(V48_AVAILABILITY, new Set([0, 110, 111]));
    expect(graph.get(110)?.parent).toBeNull();
    // 111's parent 110 DID survive, so its edge is untouched.
    expect(graph.get(111)?.parent).toBe(110);
  });

  it("carries the canonical identity, which can differ from the wire id", () => {
    expect(v48().get(500)?.identity).toBe(900);
    expect(v48().get(500)?.name).toBe("Gm");
  });
});

describe("graph helpers", () => {
  it("childrenOf returns direct children ascending", () => {
    expect(childrenOf(v48(), 0)).toEqual([100, 500]);
    expect(childrenOf(v48(), 111)).toEqual([]);
  });

  it("rootOf walks to the branch root and returns the id itself if unknown", () => {
    expect(rootOf(v48(), 510)).toBe(0);
    expect(rootOf(v48(), 0)).toBe(0);
    expect(rootOf(v48(), 9999)).toBe(9999);
  });

  it("jobTreePath is root -> node inclusive", () => {
    expect(jobTreePath(v48(), 111).map((n) => n.id)).toEqual([0, 100, 110, 111]);
    expect(jobTreePath(v48(), 9999)).toEqual([]);
  });

  it("tierLabel is Base for a root with children, else the ordinal depth", () => {
    expect(tierLabel(v48(), 0)).toBe("Base");
    expect(tierLabel(v48(), 100)).toBe("1st");
    expect(tierLabel(v48(), 111)).toBe("3rd");
    expect(tierLabel(v48(), 9999)).toBe("");
  });

  it("advancementChains lists every root-to-leaf path below the entry, entry excluded", () => {
    expect(advancementChains(v48(), 100)).toEqual([[110, 111]]);
    expect(advancementChains(v48(), 111)).toEqual([]);
  });

  it("subtreeCount counts the entry and everything below it", () => {
    expect(subtreeCount(v48(), 100)).toBe(3);
    expect(subtreeCount(v48(), 111)).toBe(1);
    expect(subtreeCount(v48(), 9999)).toBe(0);
  });

  it("jobNodeName falls back to `Job <id>` for an id the graph does not carry", () => {
    expect(jobNodeName(v48(), 500)).toBe("Gm");
    expect(jobNodeName(v48(), 9999)).toBe("Job 9999");
  });
});
```

- [x] **Step 3: Run the tests and verify they fail**

```bash
cd services/atlas-ui && npx vitest run src/lib/jobs/__tests__/job-graph.test.ts
```

Expected: FAIL — cannot resolve `@/lib/jobs/job-graph`.

- [x] **Step 4: Implement `job-graph.ts`**

Create `services/atlas-ui/src/lib/jobs/job-graph.ts`:

```ts
import type { JobAvailabilityEntry } from "@/services/api/availability.service";

/**
 * One node of the tenant version's job advancement graph.
 *
 * `id` is the version's WIRE id (the key the API round-trips). `identity` is
 * the version-blind canonical token — key version-stable curation (rail
 * grouping, accent colours) on THIS, never on `id`: wire id 500 is Gm at
 * gms 48.1 and Pirate at gms 72.1, so a wire-keyed rail would render Gm
 * inside the Explorers group in pirate colours.
 */
export interface JobNode {
  id: number;
  identity: number;
  name: string;
  parent: number | null;
}

export type JobGraph = ReadonlyMap<number, JobNode>;

/**
 * The tenant's job graph: release availability (GET /api/data/job-availability)
 * INTERSECTED with WZ presence (GET /api/data/jobs), per FR-4.1. Names and
 * parent edges come from availability, which is version-correct; presence
 * only gates visibility.
 *
 * Re-rooting is applied here and ONLY here: a parent dropped by the
 * intersection becomes null, so no downstream helper ever has to handle a
 * dangling edge. This is the same rule the backend's job.Set.ParentWire
 * applies (an unavailable parent makes the entry a root, rather than
 * synthesising a grandparent edge) — applied a second time because the
 * intersection can drop a parent whose child survives.
 */
export function buildJobGraph(
  availability: readonly JobAvailabilityEntry[],
  present: ReadonlySet<number>,
): JobGraph {
  const kept = availability.filter((e) => present.has(e.id));
  const keptIds = new Set(kept.map((e) => e.id));
  const graph = new Map<number, JobNode>();
  for (const e of kept) {
    graph.set(e.id, {
      id: e.id,
      identity: e.identity,
      name: e.name,
      parent: e.parent !== null && keptIds.has(e.parent) ? e.parent : null,
    });
  }
  return graph;
}

/** Display name for a wire id, or `Job <id>` when the graph does not carry it. */
export function jobNodeName(graph: JobGraph, id: number): string {
  return graph.get(id)?.name ?? `Job ${id}`;
}

/** Direct children of a node, ascending by wire id. */
export function childrenOf(graph: JobGraph, id: number): number[] {
  return [...graph.values()]
    .filter((n) => n.parent === id)
    .map((n) => n.id)
    .sort((a, b) => a - b);
}

/** Walk parent edges to the branch root. Returns the id itself if it is a root or absent. */
export function rootOf(graph: JobGraph, id: number): number {
  let cur = graph.get(id);
  if (!cur) return id;
  while (cur.parent !== null) {
    const next = graph.get(cur.parent);
    if (!next) break;
    cur = next;
  }
  return cur.id;
}

/** Root -> node advancement path (inclusive). Empty when the node is absent. */
export function jobTreePath(graph: JobGraph, id: number): JobNode[] {
  const path: JobNode[] = [];
  let cur = graph.get(id);
  while (cur) {
    path.unshift(cur);
    cur = cur.parent !== null ? graph.get(cur.parent) : undefined;
  }
  return path;
}

function ordinal(n: number): string {
  if (n === 1) return "1st";
  if (n === 2) return "2nd";
  if (n === 3) return "3rd";
  return `${n}th`;
}

/**
 * Tier tag for a flow chip: "Base" for a graph root with children, "" for a
 * childless root or an absent id, else the ordinal advancement depth
 * ("1st" … "10th") measured from the root.
 */
export function tierLabel(graph: JobGraph, id: number): string {
  const depth = jobTreePath(graph, id).length - 1;
  if (depth < 0) return "";
  if (depth === 0) return childrenOf(graph, id).length > 0 ? "Base" : "";
  return ordinal(depth);
}

/**
 * Every advancement chain below entryId: one array per root-to-leaf path of
 * the subtree, EXCLUDING entryId itself, DFS in ascending child order. A leaf
 * entry yields []. No availability filter is needed — the graph IS the
 * available set (buildJobGraph already intersected it).
 */
export function advancementChains(graph: JobGraph, entryId: number): number[][] {
  const walk = (id: number): number[][] => {
    const kids = childrenOf(graph, id);
    if (kids.length === 0) return [[]];
    const out: number[][] = [];
    for (const k of kids) {
      for (const rest of walk(k)) out.push([k, ...rest]);
    }
    return out;
  };
  return walk(entryId).filter((chain) => chain.length > 0);
}

/** Count of nodes in entryId's subtree, entry included (0 if the entry is absent). */
export function subtreeCount(graph: JobGraph, entryId: number): number {
  if (!graph.has(entryId)) return 0;
  return (
    1 +
    childrenOf(graph, entryId).reduce((n, k) => n + subtreeCount(graph, k), 0)
  );
}
```

- [x] **Step 5: Run the graph tests and verify they pass**

```bash
cd services/atlas-ui && npx vitest run src/lib/jobs/__tests__/job-graph.test.ts
```

Expected: PASS, all cases.

- [x] **Step 6: Write the failing hook tests**

Create `services/atlas-ui/src/lib/hooks/api/__tests__/useJobGraph.test.tsx`, following the mocking conventions already used by the repo's other `lib/hooks/api/__tests__` files (mock `@/lib/hooks/api/useJobs`, `@/lib/hooks/api/useJobAvailability`, and `@/context/tenant-context` with `vi.mock`):

```tsx
import { describe, expect, it, vi, beforeEach } from "vitest";
import { renderHook } from "@testing-library/react";

const useJobs = vi.fn();
const useJobAvailability = vi.fn();

vi.mock("@/lib/hooks/api/useJobs", () => ({ useJobs: (...a: unknown[]) => useJobs(...a) }));
vi.mock("@/lib/hooks/api/useJobAvailability", () => ({
  useJobAvailability: (...a: unknown[]) => useJobAvailability(...a),
  jobAvailabilityKeys: { list: () => ["job-availability", "list"] },
}));
vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({ activeTenant: { id: "t-1" } }),
}));

import { useJobGraph, useJobNameLookup } from "@/lib/hooks/api/useJobGraph";

function q(over: Record<string, unknown>) {
  return { isSuccess: false, isPending: false, isError: false, data: undefined, ...over };
}

const AVAILABILITY = [
  { id: 0, name: "Beginner", parent: null, identity: 0 },
  { id: 500, name: "Gm", parent: 0, identity: 900 },
  { id: 510, name: "Super Gm", parent: 500, identity: 910 },
];

beforeEach(() => {
  useJobs.mockReset();
  useJobAvailability.mockReset();
});

describe("useJobGraph", () => {
  it("is pending — with an EMPTY graph — while either query is pending", () => {
    useJobAvailability.mockReturnValue(q({ isPending: true }));
    useJobs.mockReturnValue(q({ isSuccess: true, data: { jobs: [{ id: "0" }] } }));

    const { result } = renderHook(() => useJobGraph());
    expect(result.current.isPending).toBe(true);
    expect(result.current.isSuccess).toBe(false);
    expect(result.current.graph.size).toBe(0);
  });

  it("is an error when either query errors", () => {
    useJobAvailability.mockReturnValue(q({ isError: true }));
    useJobs.mockReturnValue(q({ isSuccess: true, data: { jobs: [] } }));

    const { result } = renderHook(() => useJobGraph());
    expect(result.current.isError).toBe(true);
    expect(result.current.isSuccess).toBe(false);
  });

  it("intersects availability with the WZ job set once both succeed", () => {
    useJobAvailability.mockReturnValue(q({ isSuccess: true, data: { jobs: AVAILABILITY } }));
    useJobs.mockReturnValue(q({ isSuccess: true, data: { jobs: [{ id: "0" }, { id: "500" }] } }));

    const { result } = renderHook(() => useJobGraph());
    expect(result.current.isSuccess).toBe(true);
    expect([...result.current.graph.keys()].sort((a, b) => a - b)).toEqual([0, 500]);
    expect(result.current.graph.get(500)?.identity).toBe(900);
  });
});

describe("useJobNameLookup", () => {
  it("resolves version-correct names and falls back to `Job <id>`", () => {
    useJobAvailability.mockReturnValue(q({ isSuccess: true, data: { jobs: AVAILABILITY } }));
    useJobs.mockReturnValue(q({ isSuccess: true, data: { jobs: [{ id: "500" }] } }));

    const { result } = renderHook(() => useJobNameLookup());
    expect(result.current(500)).toBe("Gm");
    expect(result.current(999)).toBe("Job 999");
  });
});
```

- [x] **Step 7: Run them and verify they fail**

```bash
cd services/atlas-ui && npx vitest run src/lib/hooks/api/__tests__/useJobGraph.test.tsx
```

Expected: FAIL — cannot resolve `@/lib/hooks/api/useJobGraph`.

- [x] **Step 8: Implement the hook**

Create `services/atlas-ui/src/lib/hooks/api/useJobGraph.ts`:

```ts
import { useCallback, useMemo } from "react";
import { useTenant } from "@/context/tenant-context";
import { useJobs } from "@/lib/hooks/api/useJobs";
import { useJobAvailability } from "@/lib/hooks/api/useJobAvailability";
import { buildJobGraph, jobNodeName, type JobGraph } from "@/lib/jobs/job-graph";

export interface JobGraphResult {
  /** The tenant version's job graph: availability ∩ WZ presence, re-rooted. Empty unless isSuccess. */
  graph: JobGraph;
  /** Both queries succeeded. Only then may a caller treat an absent id as absent. */
  isSuccess: boolean;
  /** Either query is still pending. Means "unknown", NOT "empty". */
  isPending: boolean;
  /** Either query failed. */
  isError: boolean;
}

/**
 * The single source of the Jobs page's hierarchy: release availability
 * (version-correct names + advancement parents) intersected with the
 * tenant's ingested WZ job set (FR-4.1/4.2/4.3).
 *
 * Gating discipline, generalised from task-182 design D10 across two
 * queries: TenantProvider calls queryClient.clear() on every tenant switch,
 * so "empty graph" is the NORMAL state immediately after a switch. Callers
 * MUST gate destructive behaviour (redirecting an invalid /jobs/{id},
 * treating an id as absent) on isSuccess, never on graph.size. Both
 * underlying queries are keyed by tenant id, so no cross-tenant bleed is
 * possible even without the cache clear.
 */
export function useJobGraph(): JobGraphResult {
  const { activeTenant } = useTenant();
  const availabilityQuery = useJobAvailability(activeTenant);
  const jobsQuery = useJobs(activeTenant);

  const isSuccess = availabilityQuery.isSuccess && jobsQuery.isSuccess;
  const isPending = availabilityQuery.isPending || jobsQuery.isPending;
  const isError = availabilityQuery.isError || jobsQuery.isError;

  const graph = useMemo<JobGraph>(() => {
    if (!isSuccess) return new Map();
    const present = new Set(
      (jobsQuery.data?.jobs ?? []).map((j) => Number(j.id)),
    );
    return buildJobGraph(availabilityQuery.data?.jobs ?? [], present);
  }, [isSuccess, availabilityQuery.data, jobsQuery.data]);

  return { graph, isSuccess, isPending, isError };
}

/**
 * A version-correct job-name resolver for call sites that are not React
 * components and cannot hold the graph themselves (breadcrumb resolvers,
 * table column builders). The nearest component calls this hook and passes
 * the returned function down.
 *
 * Falls back to `Job <id>` while the graph is unknown. That is deliberate:
 * it is honest about not knowing, whereas the static table it replaces
 * asserted a v83 name to a v0.48 tenant.
 */
export function useJobNameLookup(): (id: number) => string {
  const { graph } = useJobGraph();
  return useCallback((id: number) => jobNodeName(graph, id), [graph]);
}
```

- [x] **Step 9: Run the hook tests and the type-check**

```bash
cd services/atlas-ui && npx vitest run src/lib/hooks/api/__tests__/useJobGraph.test.tsx && npm run build
```

Expected: tests PASS; `npm run build` clean (nothing was deleted yet, so every existing consumer still compiles).

- [x] **Step 10: Commit**

```bash
git add services/atlas-ui/src/services/api/availability.service.ts \
        services/atlas-ui/src/lib/jobs/job-graph.ts \
        services/atlas-ui/src/lib/jobs/__tests__/job-graph.test.ts \
        services/atlas-ui/src/lib/hooks/api/useJobGraph.ts \
        services/atlas-ui/src/lib/hooks/api/__tests__/useJobGraph.test.tsx \
        services/atlas-ui/src/lib/hooks/api/useJobAvailability.ts
git commit -m "feat(atlas-ui): useJobGraph — availability ∩ WZ presence, version-correct

FR-4.1/4.2/4.3 (task-202). Additive: the API-sourced graph, its pure
helpers, and a name resolver for non-component call sites. No consumer
migrated yet."
```

---

## Task 8: Jobs page cluster onto the API graph

**Files:**
- Modify: `services/atlas-ui/src/components/features/jobs/rail-groups.ts`
- Modify: `services/atlas-ui/src/components/features/jobs/advancement-flow.tsx`
- Modify: `services/atlas-ui/src/pages/JobsPage.tsx`
- Modify: `services/atlas-ui/src/components/features/jobs/__tests__/rail-groups.test.ts`, `__tests__/advancement-flow.test.tsx`, `__tests__/branch-rail.test.tsx`
- Modify: `services/atlas-ui/src/lib/jobs/job-advancement-tree.ts` (delete the helpers this task orphans)

**Interfaces:**
- Consumes: Task 7's `useJobGraph`, `JobGraph`, `JobNode`, and the graph-first helpers.
- Produces:
  - `RailEntry { identity: number; accent: string }` — rail membership keyed on the CANONICAL identity, not the wire id.
  - `branchEntryOf(graph: JobGraph, jobId: number): RailEntry`
  - `visibleRailGroups(graph: JobGraph): VisibleRailGroup[]`, where `VisibleRailEntry { identity: number; accent: string; id: number; name: string; count: number }` (`id` is the wire id resolved through the graph, for selection and `data-testid`s).
  - `AdvancementFlow` props become `{ graph, entryId, selectedJobId, accent, onSelect }` — the `available` prop is gone (the graph is the available set).
- After this task `job-advancement-tree.ts` retains only `JobEntry`, `JOB_GRAPH`, `JOB_LIST`, `jobName`, and `jobTreePath`; Task 9 deletes the file.

- [x] **Step 1: Rewrite the rail-groups tests**

In `services/atlas-ui/src/components/features/jobs/__tests__/rail-groups.test.ts`, build fixture graphs with `buildJobGraph` and assert against identity-keyed rails. Add these cases (keeping the file's existing coverage, retargeted to the new signatures):

```ts
// A v0.48 fixture: wire 500 is Gm (identity 900), 510 is Super Gm (910), and
// there is no Pirate branch. This is the case a wire-keyed rail gets wrong —
// it would file "Gm" under Explorers in pirate colours.
const v48 = buildJobGraph(
  [
    { id: 0, name: "Beginner", parent: null, identity: 0 },
    { id: 100, name: "Warrior", parent: 0, identity: 100 },
    { id: 500, name: "Gm", parent: 0, identity: 900 },
    { id: 510, name: "Super Gm", parent: 500, identity: 910 },
  ],
  new Set([0, 100, 500, 510]),
);

it("v0.48: the Explorers rail has no Pirate entry", () => {
  const explorers = visibleRailGroups(v48).find((g) => g.label === "Explorers");
  expect(explorers?.entries.map((e) => e.identity)).toEqual([100]);
});

it("v0.48: the Special group shows Gm, with Super Gm counted beneath it", () => {
  const special = visibleRailGroups(v48).find((g) => g.label === "Special");
  expect(special?.entries.map((e) => e.name)).toEqual(["Gm"]);
  expect(special?.entries[0]?.id).toBe(500);
  expect(special?.entries[0]?.count).toBe(2);
});

it("v0.72: the Cygnus Knights group is absent entirely", () => {
  const v72 = buildJobGraph(
    [
      { id: 0, name: "Beginner", parent: null, identity: 0 },
      { id: 100, name: "Warrior", parent: 0, identity: 100 },
      { id: 500, name: "Pirate", parent: 0, identity: 500 },
      { id: 900, name: "Gm", parent: 0, identity: 900 },
    ],
    new Set([0, 100, 500, 900]),
  );
  expect(visibleRailGroups(v72).map((g) => g.label)).not.toContain(
    "Cygnus Knights",
  );
});

it("v0.79: the Legends group is absent entirely", () => {
  const v79 = buildJobGraph(
    [
      { id: 0, name: "Beginner", parent: null, identity: 0 },
      { id: 1000, name: "Noblesse", parent: null, identity: 1000 },
      { id: 1100, name: "Dawn Warrior 1", parent: 1000, identity: 1100 },
    ],
    new Set([0, 1000, 1100]),
  );
  expect(visibleRailGroups(v79).map((g) => g.label)).not.toContain("Legends");
});
```

- [x] **Step 2: Run them and verify they fail**

```bash
cd services/atlas-ui && npx vitest run src/components/features/jobs/__tests__/rail-groups.test.ts
```

Expected: FAIL — `visibleRailGroups` still takes a `ReadonlySet<number>`, and `RailEntry` has no `identity`.

- [x] **Step 3: Rewrite `rail-groups.ts` on identity keys**

Replace `services/atlas-ui/src/components/features/jobs/rail-groups.ts` with:

```ts
import {
  jobTreePath,
  subtreeCount,
  type JobGraph,
} from "@/lib/jobs/job-graph";

export interface RailEntry {
  /**
   * The CANONICAL job identity whose advancement flow the entry shows —
   * never the wire id. Rail curation ("Explorers", "Special", the accent
   * colours) is an editorial grouping that cannot be derived from graph
   * shape: at gms 48.1 Warrior, Magician, Bowman, Rogue AND Gm are all
   * depth-1 children of Beginner, and nothing structural says which are
   * Explorers. So it needs a version-stable key for a job CONCEPT, and the
   * wire id is not one — wire id 500 is Pirate at gms 72.1 but Gm at
   * gms 48.1, which a wire-keyed rail would file under Explorers in pirate
   * colours (task-202 design D9).
   */
  identity: number;
  /** Theme token name for the branch accent, e.g. "--c-warrior" (src/index.css). */
  accent: string;
}

export interface RailGroup {
  label: string;
  entries: RailEntry[];
}

/** The Warrior rail entry; also branchEntryOf's fallback for Beginner/unknown ids. */
const WARRIOR_ENTRY: RailEntry = { identity: 100, accent: "--c-warrior" };

// Rail entries per PRD FR-3.1; accents are scoped via style={{ "--acc": `var(${accent})` }}.
export const RAIL_GROUPS: RailGroup[] = [
  {
    label: "Explorers",
    entries: [
      WARRIOR_ENTRY,
      { identity: 200, accent: "--c-magician" },
      { identity: 300, accent: "--c-bowman" },
      { identity: 400, accent: "--c-thief" },
      { identity: 500, accent: "--c-pirate" },
    ],
  },
  { label: "Cygnus Knights", entries: [{ identity: 1000, accent: "--c-cygnus" }] },
  {
    label: "Legends",
    entries: [
      { identity: 2000, accent: "--c-aran" },
      { identity: 2001, accent: "--c-evan" },
    ],
  },
  {
    label: "Special",
    entries: [
      { identity: 800, accent: "--c-special" },
      { identity: 900, accent: "--c-special" },
    ],
  },
];

/** The wire id this version binds a canonical identity to, or undefined. */
function wireIdOf(graph: JobGraph, identity: number): number | undefined {
  for (const n of graph.values()) {
    if (n.identity === identity) return n.id;
  }
  return undefined;
}

/**
 * The rail entry whose node lies on jobId's advancement path. Beginner and
 * unknown ids fall back to the first entry (Warrior) — the caller keeps the
 * job selection itself.
 */
export function branchEntryOf(graph: JobGraph, jobId: number): RailEntry {
  const pathIdentities = new Set(jobTreePath(graph, jobId).map((n) => n.identity));
  for (const g of RAIL_GROUPS) {
    for (const e of g.entries) {
      if (pathIdentities.has(e.identity)) return e;
    }
  }
  return WARRIOR_ENTRY;
}

export interface VisibleRailEntry extends RailEntry {
  /** This version's wire id for the entry — what selection and routing use. */
  id: number;
  name: string;
  count: number;
}

export interface VisibleRailGroup {
  label: string;
  entries: VisibleRailEntry[];
}

/**
 * Rail groups for the tenant's job graph, with display name + subtree count;
 * empty groups dropped (FR-4.6). The graph IS the intersected available set,
 * so a class the version never released — or never ingested — takes its whole
 * group with it.
 */
export function visibleRailGroups(graph: JobGraph): VisibleRailGroup[] {
  return RAIL_GROUPS.map((g) => ({
    label: g.label,
    entries: g.entries.flatMap((e) => {
      const id = wireIdOf(graph, e.identity);
      if (id === undefined) return [];
      return [
        {
          ...e,
          id,
          name: graph.get(id)?.name ?? `Job ${id}`,
          count: subtreeCount(graph, id),
        },
      ];
    }),
  })).filter((g) => g.entries.length > 0);
}
```

- [x] **Step 4: Run the rail-groups tests and verify they pass**

```bash
cd services/atlas-ui && npx vitest run src/components/features/jobs/__tests__/rail-groups.test.ts
```

Expected: PASS.

- [x] **Step 5: Convert `AdvancementFlow` to take the graph**

In `services/atlas-ui/src/components/features/jobs/advancement-flow.tsx`:

- Replace the import block with `import { advancementChains, jobNodeName, jobTreePath, tierLabel, type JobGraph } from "@/lib/jobs/job-graph";`
- Change `AdvancementFlowProps` to `{ graph: JobGraph; entryId: number; selectedJobId: number; accent: string; onSelect: (id: number) => void }` — the `available` prop is removed.
- `FlowChip` gains a `graph: JobGraph` prop; its body becomes `const tier = tierLabel(graph, id);` and its label `{jobNodeName(graph, id)}`.
- `anchors` becomes `useMemo(() => jobTreePath(graph, entryId).map((n) => n.id), [graph, entryId])`.
- `chains` becomes `useMemo(() => advancementChains(graph, entryId), [graph, entryId])`.
- Every `<FlowChip .../>` call site passes `graph={graph}`.

Update `__tests__/advancement-flow.test.tsx` and `__tests__/branch-rail.test.tsx` to build their fixtures with `buildJobGraph` and pass `graph` instead of `available`. Add a v0.83 case:

```tsx
it("v0.83: every Cygnus branch ends at 3rd job", () => {
  const v83 = buildJobGraph(
    [
      { id: 1000, name: "Noblesse", parent: null, identity: 1000 },
      { id: 1100, name: "Dawn Warrior 1", parent: 1000, identity: 1100 },
      { id: 1110, name: "Dawn Warrior 2", parent: 1100, identity: 1110 },
      { id: 1111, name: "Dawn Warrior 3", parent: 1110, identity: 1111 },
    ],
    new Set([1000, 1100, 1110, 1111]),
  );
  render(
    <AdvancementFlow
      graph={v83}
      entryId={1000}
      selectedJobId={1000}
      accent="--c-cygnus"
      onSelect={() => {}}
    />,
  );
  expect(screen.getByTestId("flow-cell-1111")).toBeInTheDocument();
  expect(screen.queryByTestId("flow-cell-1112")).not.toBeInTheDocument();
});
```

- [x] **Step 6: Rewire `JobsPage.tsx`**

In `services/atlas-ui/src/pages/JobsPage.tsx`:

- Delete the `useJobs` and `JOB_GRAPH` imports; add `import { useJobGraph } from "@/lib/hooks/api/useJobGraph";` and `import { jobNodeName } from "@/lib/jobs/job-graph";`.
- Replace the `jobsQuery` / `available` / `groups` block with:

```tsx
  // The tenant version's job graph: release availability ∩ WZ presence, with
  // version-correct names and parent edges (FR-4.1/4.2/4.3). Empty while
  // either query is pending — which is why every consumer below is gated on
  // isSuccess rather than on the graph being non-empty. TenantProvider calls
  // queryClient.clear() on every tenant switch, so "pending with an empty
  // graph" is the state right after a switch, not an error.
  const { graph, isSuccess, isPending, isError } = useJobGraph();
  const groups = useMemo(
    () => (isSuccess ? visibleRailGroups(graph) : []),
    [isSuccess, graph],
  );
```

- Replace every remaining `jobsQuery.isSuccess` with `isSuccess`, `jobsQuery.isPending` with `isPending`, and `jobsQuery.isError` with `isError`. The `jobs-load-error` card now covers a failure of EITHER query (FR-4.5).
- `jobIdValid` becomes:

```tsx
  const jobIdValid =
    parsedJobId !== null &&
    Number.isInteger(parsedJobId) &&
    isSuccess &&
    graph.has(parsedJobId);
```

  (the separate `JOB_GRAPH[parsedJobId] !== undefined` check is gone — graph membership already means "this version has it").
- `const jobName = jobNodeName(graph, jobId);`
- `const entry = branchEntryOf(graph, jobId);`
- `<BranchRail ... isPending={isPending} />`
- `<AdvancementFlow graph={graph} entryId={entry.id} selectedJobId={jobId} accent={entry.accent} onSelect={selectJob} />` — note `entry.id` is now the `VisibleRailEntry` wire id; `branchEntryOf` returns a bare `RailEntry` (identity + accent only), so resolve the entry through `groups` and fall back to the selected job's own root:

```tsx
  const railEntry = branchEntryOf(graph, jobId);
  const entry = useMemo(() => {
    for (const g of groups) {
      const match = g.entries.find((e) => e.identity === railEntry.identity);
      if (match) return match;
    }
    return { ...railEntry, id: rootOf(graph, jobId), name: "", count: 0 };
  }, [groups, railEntry, graph, jobId]);
```

  Import `rootOf` from `@/lib/jobs/job-graph` for the fallback.
- The `defaultJobId` line is unchanged (`groups[0]?.entries[0]?.id ?? 100`).

- [x] **Step 7: Add the JobsPage gating tests**

In the JobsPage test file (create `services/atlas-ui/src/pages/__tests__/JobsPage.test.tsx` if the repo has none; otherwise extend it), mock `@/lib/hooks/api/useJobGraph` and assert:

```tsx
it("does not redirect a valid /jobs/112 while either query is pending (task-182 D10)", () => {
  // useJobGraph mocked to { graph: new Map(), isSuccess: false, isPending: true, isError: false }
  // Render at /jobs/112 and assert navigate was NOT called.
});

it("renders the load-error card when either query fails", () => {
  // useJobGraph mocked to { graph: new Map(), isSuccess: false, isPending: false, isError: true }
  // expect(screen.getByTestId("jobs-load-error")).toBeInTheDocument();
});
```

Write both bodies out concretely against the file's existing render/router harness — the assertions above are the contract, not placeholders.

- [x] **Step 8: Delete the now-orphaned helpers**

From `services/atlas-ui/src/lib/jobs/job-advancement-tree.ts`, delete `JOB_ROOTS`, `childrenOf`, `rootOf`, `visibleRoots`, `visibleChildrenOf`, `advancementChains`, `tierLabel`, and `subtreeCount`. Keep `JobEntry`, `JOB_GRAPH`, `JOB_LIST`, `jobName`, and `jobTreePath` — Task 9's consumers still reference them, and Task 9 deletes the file outright. Trim `src/lib/jobs/__tests__/job-advancement-tree.test.ts` to match.

- [x] **Step 9: Run the atlas-ui suite and type-check**

```bash
cd services/atlas-ui && npm run test && npm run build
```

Expected: both clean.

- [x] **Step 10: Commit**

```bash
git add services/atlas-ui/src/components/features/jobs/ \
        services/atlas-ui/src/pages/JobsPage.tsx \
        services/atlas-ui/src/pages/__tests__/ \
        services/atlas-ui/src/lib/jobs/job-advancement-tree.ts \
        services/atlas-ui/src/lib/jobs/__tests__/job-advancement-tree.test.ts
git commit -m "feat(atlas-ui): Jobs page hierarchy comes from the API, rails key on identity

FR-4.1-4.7 (task-202). JobsPage, rail-groups and advancement-flow now
consume useJobGraph. Rail membership keys on the canonical identity, so a
v0.48 tenant files wire id 500 under Special as Gm rather than under
Explorers as a pirate."
```

---

## Task 9: Retire the static name tables

**Files:**
- Modify: `services/atlas-ui/src/components/features/characters/SkillsSection.tsx`
- Modify: `services/atlas-ui/src/components/features/rankings/LeaderboardRow.tsx`
- Modify: `services/atlas-ui/src/components/features/characters/presets/PresetEditor.tsx`, `PresetCard.tsx`, `JobCombobox.tsx`
- Modify: `services/atlas-ui/src/lib/breadcrumbs/routes.ts`, `services/atlas-ui/src/lib/hooks/useBreadcrumbs.ts`
- Modify: `services/atlas-ui/src/pages/characters-columns.tsx`, `CharactersPage.tsx`, `GuildDetailPage.tsx`
- Modify: `services/atlas-ui/src/lib/hooks/usePresetJobOptions.ts`
- Delete: `services/atlas-ui/src/lib/jobs.ts`, `services/atlas-ui/src/lib/jobs/job-advancement-tree.ts`, `services/atlas-ui/src/lib/jobs/__tests__/job-advancement-tree.test.ts`
- Modify: the colocated tests for each touched component

**Interfaces:**
- Consumes: Task 7's `useJobNameLookup()` and `jobTreePath(graph, id)`; Task 8's remaining exports.
- Produces: `JobNameResolver = (id: number) => string`, threaded into non-component call sites. `RouteConfig.labelResolver` becomes `(params: Record<string, string>, ctx: BreadcrumbResolverContext) => string` with `BreadcrumbResolverContext { jobName: JobNameResolver }`. `getColumns` gains a required `jobName: JobNameResolver` prop.
- After this task, `grep -rn "JOB_GRAPH\|jobNameMap\|getJobNameById\|JOB_LIST" services/atlas-ui/src` returns nothing.

- [x] **Step 1: Migrate the component-level name consumers**

Each of these is a React component and can call the hook directly. In every case, replace the static import with `import { useJobNameLookup } from "@/lib/hooks/api/useJobGraph";`, add `const jobName = useJobNameLookup();` at the top of the component, and leave the call sites (`jobName(attrs.jobId)`) textually unchanged:

- `components/features/rankings/LeaderboardRow.tsx` (line 7 import, line 80 use)
- `components/features/characters/presets/PresetEditor.tsx` (line 5, line 73)
- `components/features/characters/presets/PresetCard.tsx` (line 7, line 109)
- `components/features/characters/presets/JobCombobox.tsx` (line 11, line 34 — it stays a fallback: `jobs.find((j) => j.id === value)?.name ?? jobName(value)`)

- [x] **Step 2: Migrate `SkillsSection` to the graph-parameterised path**

In `components/features/characters/SkillsSection.tsx`, replace the `jobTreePath` import with:

```tsx
import { useJobGraph } from "@/lib/hooks/api/useJobGraph";
import { jobTreePath } from "@/lib/jobs/job-graph";
```

and line 16 with:

```tsx
  const { graph } = useJobGraph();
  const path = jobTreePath(graph, character.attributes.jobId);
```

While the graph is unknown, `path` is `[]` — the same shape the old helper returned for an unmapped id, so the component's existing empty-path handling covers it.

- [x] **Step 3: Thread a resolver into the breadcrumb route table**

In `services/atlas-ui/src/lib/breadcrumbs/routes.ts`:

```ts
/** A version-correct job-name resolver, supplied by the calling component. */
export type JobNameResolver = (id: number) => string;

/**
 * Context a labelResolver may need but cannot fetch itself: the route table
 * is a plain module-level array, not a component, so it cannot call hooks.
 * useBreadcrumbs supplies this from useJobNameLookup().
 */
export interface BreadcrumbResolverContext {
  jobName: JobNameResolver;
}
```

Change the `RouteConfig` field to `labelResolver?: (params: Record<string, string>, ctx: BreadcrumbResolverContext) => string;`, replace the `/jobs/[id]` entry's resolver and its comment with:

```ts
    // Deliberately no `entityType`: job names come from the tenant's job
    // graph via labelResolver, not from an async entity resolver. Declaring
    // one would mark the crumb `dynamic` and the resolver lookup would miss,
    // overwriting the label with "Unknown". The graph is version-correct —
    // the static table this replaced named wire id 500 "Pirate" even on a
    // v0.48 tenant, where it is Gm (task-202 FR-4.2).
    labelResolver: (params, ctx) => ctx.jobName(Number(params.id)),
```

and change `getBreadcrumbsFromRoute` to accept and forward the context:

```ts
export function getBreadcrumbsFromRoute(
  pathname: string,
  ctx: BreadcrumbResolverContext,
): Partial<BreadcrumbSegment>[] {
```

with the label line becoming `label: config.labelResolver ? config.labelResolver(params, ctx) : config.label,`.

Delete `services/atlas-ui/src/lib/jobs.ts`.

- [x] **Step 4: Supply the context from `useBreadcrumbs`**

In `services/atlas-ui/src/lib/hooks/useBreadcrumbs.ts`, add `const jobName = useJobNameLookup();`, build `const resolverCtx = useMemo(() => ({ jobName }), [jobName]);`, pass it to every `getBreadcrumbsFromRoute(pathname, resolverCtx)` call, and add `resolverCtx` to the dependency array of the memo/effect that produces `initialBreadcrumbs`. `useJobNameLookup` returns a `useCallback`-stable function, so this does not re-fire on every render.

Update `lib/breadcrumbs/__tests__/routes.test.ts:360` to `expect(detail?.labelResolver?.({ id: "110" }, { jobName: () => "Fighter" })).toBe("Fighter");`.

- [x] **Step 5: Thread the resolver into the two table column builders**

`pages/characters-columns.tsx`: add `jobName: JobNameResolver;` to `ColumnProps` (importing the type from `@/lib/breadcrumbs/routes`), destructure it in `getColumns`, and replace line 124 with `name = jobName(id);`. Delete the `getJobNameById` import.

`pages/CharactersPage.tsx`: add `const jobName = useJobNameLookup();` and pass `jobName` in the `getColumns({...})` call at line 58.

`pages/GuildDetailPage.tsx`: it is a page component, so add `const jobName = useJobNameLookup();` and replace line 158 with `name = jobName(id);`. Delete the `getJobNameById` import.

- [x] **Step 6: Drop the `JOB_LIST` fallback from `usePresetJobOptions`**

Replace the last two paragraphs of `usePresetJobOptions`'s doc comment and its body:

```ts
/**
 * ...
 * useJobAvailability's pending/error state means "unknown", not "empty"
 * (TenantProvider clears the query cache on every tenant switch). Until
 * availability is known this returns an EMPTY list, and the combobox renders
 * its pending affordance. It used to fall back to the static JOB_LIST on the
 * reasoning that "a picker must never be blank" — but that offered v83 job
 * names to a v0.48 tenant, which is exactly the bug task-202 removed. An
 * empty list plus a pending state is honest; a wrong list is not.
 */
export function usePresetJobOptions(): PresetJobOption[] {
  const { activeTenant } = useTenant();
  const availabilityQuery = useJobAvailability(activeTenant);

  return useMemo(() => {
    if (!availabilityQuery.isSuccess) return [];
    return availabilityQuery.data.jobs
      .map((j) => ({ id: j.id, name: j.name }))
      .sort((a, b) => a.id - b.id);
  }, [availabilityQuery.isSuccess, availabilityQuery.data]);
}
```

Delete the `JOB_LIST` import. Update `components/features/characters/presets/__tests__/JobCombobox.test.tsx` and any other test asserting the old fallback: the pending state now yields an empty option list.

- [x] **Step 7: Delete the static advancement tree**

```bash
git rm services/atlas-ui/src/lib/jobs/job-advancement-tree.ts \
       services/atlas-ui/src/lib/jobs/__tests__/job-advancement-tree.test.ts
```

- [x] **Step 8: Verify no static table or version literal survives**

```bash
cd services/atlas-ui && \
  grep -rn "JOB_GRAPH\|jobNameMap\|getJobNameById\|JOB_LIST\|job-advancement-tree" src ; \
  grep -rnE "major(Version)?\s*(<=|<|>=|>|===|==)\s*[0-9]+" src
```

Expected: the first grep prints nothing. The second prints nothing under any job-related file; investigate and remove any hit that decides job naming, parenting, or visibility (FR-4.7). Both greps exiting non-zero (no matches) is the pass condition.

- [x] **Step 9: Run the full atlas-ui gate**

```bash
cd services/atlas-ui && npm run test && npm run build
```

Expected: both clean. `npm run build` is the type-check that catches a missed call site; `npm run test` alone would not.

- [x] **Step 10: Commit**

```bash
git add -A services/atlas-ui/src
git commit -m "refactor(atlas-ui): delete the static job name/graph tables

FR-4.2 (task-202). Every consumer of jobName/getJobNameById/JOB_LIST now
resolves through the tenant's job graph. Non-component call sites
(breadcrumb resolver, table column builders) take a JobNameResolver from
the nearest component. usePresetJobOptions no longer falls back to the
v83 JOB_LIST — an empty list plus a pending state beats a wrong list."
```

---

## Task 10: Full verification and pre-PR review

**Files:** none modified except as fixes require.

**Interfaces:**
- Consumes: Tasks 1–9.
- Produces: a branch that satisfies every CLAUDE.md §Build & Verification gate, with `audit.md` written by the review agents.

- [x] **Step 1: Go gates on every changed module**

```bash
(cd libs/atlas-constants && go test -race ./... && go vet ./...)
(cd libs/atlas-constants/gen && go test -race ./... && go vet ./... && go run . -check)
(cd services/atlas-data/atlas.com/data && go test -race ./... && go vet ./... && go build ./...)
```

Expected: all clean; `go run . -check` exits 0 (no generated-file drift).

- [x] **Step 2: Repo-root guards**

```bash
tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/skill-job-id-guard.sh
tools/lint.sh --check
```

Expected: all exit 0. `tools/lint.sh --check` needs nvm on PATH; if it false-fails, source nvm (`nvm use 22`) and re-run before treating a failure as real. `tools/skill-job-id-guard.sh` matters here: this branch adds code near the divergent job wire constants, and any raw `==`/`case` against them outside the version-aware resolver is a hard fail.

No template guard is needed — this branch touches no file under `services/atlas-configurations/seed-data/templates/`. No service-registration guard is needed — no service was added.

- [x] **Step 3: atlas-ui gate**

```bash
cd services/atlas-ui && npm run build && npm run test
```

- [x] **Step 4: Docker bake if any `go.mod` changed**

```bash
git diff --name-only origin/main...HEAD -- '**/go.mod'
```

If that prints anything, run `docker buildx bake atlas-data` (and any other listed service) from the worktree root and confirm it succeeds. If it prints nothing, record that the bake was correctly skipped — do not skip silently.

- [x] **Step 5: Verify against the merge result, not the branch tip**

```bash
git fetch origin main && git merge origin/main
```

Re-run Steps 1–3 after the merge. A branch that passes at its own tip but not against current `main` is not verified.

- [x] **Step 6: Code review before the PR**

Invoke `superpowers:requesting-code-review`. Both guideline reviewers apply (Go and TS changed). Ensure the agents run inside this worktree and write to `docs/tasks/task-202-version-correct-job-hierarchy/audit.md`. Address findings, then re-run Steps 1–3.

- [x] **Step 7: Confirm the PRD acceptance criteria**

Walk `prd.md` §10 item by item and check each box, citing the test or command that proves it. Any item that cannot be proven is a blocker, not a footnote — except the two explicitly out of scope (the Evan re-ingest/baseline republish, PRD §9 Q4), which must be restated in the PR description as a known, accepted gap with the operational follow-up named.

- [ ] **Step 8: Commit any fixes and push**

```bash
git add -A && git commit -m "chore(task-202): address code-review findings"
git push -u origin task-202-version-correct-job-hierarchy
```

---

## Self-Review

**Spec coverage.** Design D1→Task 1 · D2→Task 2 · D3→Task 3 · D4→Task 4 · D5→Task 5 · D6→Task 5 Step 3 (the mandated definition-site comment) · D7→Task 5 (`ParentWire` + `TestParentWire_D7PolicyGuard`) · D8→Task 6 (raw-body `"parent":null` assertion) · D9→Task 6 + Task 8 (identity-keyed rails) · D10→Task 7 (`buildJobGraph` + helpers) · D11→Task 9 (both tables deleted, all eleven consumers migrated, the `usePresetJobOptions` behaviour change called out) · D12→Task 7 (`isSuccess`/`isPending`/`isError` composition) + Task 8 Step 7. Design §6's test list is distributed across Tasks 1, 2, 3, 5, 6, 7, 8. Design §8's verification list is Task 10. PRD FR-1.1–1.4 → Tasks 1–2; FR-2.1–2.5 → Tasks 3–4; FR-3.1–3.5 → Tasks 5–6; FR-4.1–4.7 → Tasks 7–9.

**Deviations from the design, and why.**
1. Design D10 puts the graph-first helpers back in `job-advancement-tree.ts`. This plan puts them in a new `src/lib/jobs/job-graph.ts` because the old and new `childrenOf`/`rootOf`/`tierLabel`/`advancementChains`/`subtreeCount` cannot coexist in one module, and a single-task big-bang rewrite of fourteen files could not be reviewed or type-checked incrementally. The end state is identical: `job-advancement-tree.ts` is deleted in Task 9.
2. Design §6's walk-order test is described as "worker-level"; this plan puts it in `job/processor_test.go`, which already has the `setupResourceTestDB` harness `RegisterJob` needs. Package `workers` has no DB harness, so a worker-level version would have to build one for no added coverage.

**Known-brittle spots for the executing engineer.** Task 6 Step 1's test code is written against a `jobAvailabilityResponse` shape the existing `resource_test.go` declares locally — match that file's conventions rather than pasting verbatim; the two assertions that must survive are the raw-body `"parent":null` check and `*Parent == 0` on Gm. Task 8 Step 7's JobsPage tests are specified by contract because the repo may not yet have a `JobsPage.test.tsx` harness; write them out concretely against whatever render/router harness the neighbouring page tests use.
