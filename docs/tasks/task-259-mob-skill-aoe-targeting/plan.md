# Mob Skill AoE Target Selection Fidelity — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace `atlas-monsters`' whole-map mob-skill disease selector with a box-scoped, SEDUCE-only-capped, deterministic target selector backed by real character positions.

**Architecture:** A pure selection function (`selectDiseaseTargets`) holds the box test, the cap rule, and the ordering rule with no I/O. An I/O shell (`ProcessorImpl.getDiseaseTargets`) decides single-target vs AoE, lists the field through the existing `inFieldFn` seam, resolves positions through a new `positionFn` seam with bounded concurrency, and hands an ordered candidate list to the pure function. Positions come from a new read-only `character/position` client mirroring `atlas-maps`' equivalent.

**Tech Stack:** Go, JSON:API over `libs/atlas-rest/requests`, stdlib `sync.WaitGroup` + buffered-channel semaphore (no `errgroup`), Kafka via the existing `emitter` seam.

**Spec:** `docs/tasks/task-259-mob-skill-aoe-targeting/design.md` (PRD: `prd.md`)

## Global Constraints

- Module root for every `go build` / `go test` in this plan: `services/atlas-monsters/atlas.com/monsters` (module `atlas-monsters`).
- Never introduce a literal `128` for seduce. Use `monster2.SkillTypeSeduce` from `github.com/Chronicle20/atlas/libs/atlas-constants/monster` (already imported in `monster/processor.go` as `monster2`).
- The bounding-box comparison MUST use the exact inclusive form already used at `monster/processor.go:1170-1178` (executeBuff) and `monster/processor.go:1201-1211` (executeHeal): `dx := int32(c.x) - int32(mobX)` / `dy := int32(c.y) - int32(mobY)`, kept iff `dx >= sd.LtX() && dx <= sd.RbX() && dy >= sd.LtY() && dy <= sd.RbY()`.
- No facing-direction mirroring of the rectangle.
- No dead-character (`hp`) filtering; `hp` is NOT projected in the new RestModel.
- `math/rand` stays imported in `monster/processor.go` — `rand.Intn` at `monster/processor.go:709` still uses it. Only the `rand.Shuffle` call goes.
- No field-wide fallback. If positions cannot be resolved the candidate set only ever shrinks.
- `golang.org/x/sync` must stay an indirect dependency — do not add `errgroup`.
- Test setup uses the project's Builder pattern. Do not create `*_testhelpers.go` files.
- Concurrency bound for position lookups: `8`.
- Out of scope (settled with the user at plan time): the PRD's `docs/research/missing-features/monsters-and-bosses.md` doc-update acceptance criterion. That file is untracked in the main working tree and does not exist on this branch. See `context.md`.

---

## Task 1: character/position REST client

Read-only client for `atlas-character`'s `characters` resource, projecting only `x` and `y`. This is a self-contained package with no dependency on any other task.

### Files

- `services/atlas-monsters/atlas.com/monsters/character/position/rest.go` — **new file**; `RestModel` + api2go interface methods
- `services/atlas-monsters/atlas.com/monsters/character/position/requests.go` — **new file**; `baseURLProvider` seam + `requestById`
- `services/atlas-monsters/atlas.com/monsters/character/position/processor.go` — **new file**; `Processor` interface + `ProcessorImpl.GetPosition`
- `services/atlas-monsters/atlas.com/monsters/character/position/processor_test.go` — **new file**; httptest-backed projection tests
- `services/atlas-monsters/atlas.com/monsters/character/position/mock/processor.go` — **new file**; `ProcessorMock`

Patterns to copy (read-only, do not edit):
- `services/atlas-maps/atlas.com/maps/character/rest.go` — the RestModel shape, minus the `Hp` field
- `services/atlas-maps/atlas.com/maps/character/requests.go` — `baseURLProvider` / `requestById` verbatim except the package name
- `services/atlas-maps/atlas.com/maps/character/processor.go` — the Processor/ProcessorImpl/`var _` shape
- `services/atlas-maps/atlas.com/maps/character/processor_test.go:47-90` — `withBaseURL` helper and the httptest setup
- `services/atlas-monsters/atlas.com/monsters/map/mock/processor.go` — the `ProcessorMock` shape used in this service (`XxxFunc` field, `var _ pkg.Processor = (*ProcessorMock)(nil)`, nil-guard returning a zero value)

Module root: `services/atlas-monsters/atlas.com/monsters`.

### Interfaces

- Consumes: nothing from other tasks.
- Produces, for Task 3:
  - `position.Processor` with `GetPosition(characterId uint32) (int16, int16, error)`
  - `position.NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor`
  - `mock.ProcessorMock` with field `GetPositionFunc func(characterId uint32) (int16, int16, error)`

- [ ] **Step 1: Write the failing tests**

`services/atlas-monsters/atlas.com/monsters/character/position/processor_test.go`, package `position`. Two test functions plus the `withBaseURL` helper copied from `services/atlas-maps/atlas.com/maps/character/processor_test.go:47-53`. Imports, httptest wiring, and `defer`/`Cleanup` follow that file.

JSON:API fixture (note the extra `mapId`/`hp` attributes — they exist upstream and must be ignored by the projection):

```json
{
    "data": {
        "type": "characters",
        "id": "1001",
        "attributes": {
            "mapId": 100000000,
            "hp": 875,
            "x": 123,
            "y": -456
        }
    }
}
```

| test function | server behavior | call | expected |
|---|---|---|---|
| `TestProcessor_GetPosition_ProjectsCoordinates` | 200 + the fixture above; asserts `r.Method == http.MethodGet` and `r.URL.Path == "/characters/1001"`; `Content-Type: application/vnd.api+json` | `NewProcessor(logger, context.Background()).GetPosition(1001)` | `x == int16(123)`, `y == int16(-456)`, `err == nil` |
| `TestProcessor_GetPosition_PropagatesNotFound` | `w.WriteHeader(http.StatusNotFound)` | `GetPosition(9999)` | `require.ErrorIs(err, requests.ErrNotFound)` |

- [ ] **Step 2: Run the tests to verify they fail**

Run from `services/atlas-monsters/atlas.com/monsters`:

```bash
go test ./character/position/... -run TestProcessor_GetPosition -v
```

Expected: build failure — package `position` has no `NewProcessor`.

- [ ] **Step 3: Write `rest.go`**

```go
package position

import "strconv"

// RestModel is the minimal projection of the atlas-character JSON:API
// resource needed by atlas-monsters' AoE target selection. Only the fields
// consumed are declared: position. HP is deliberately absent — mob-skill
// AoE does not filter dead characters (task-259 FR-5.5).
type RestModel struct {
	Id uint32 `json:"-"`
	X  int16  `json:"x"`
	Y  int16  `json:"y"`
}

// GetName returns the JSON:API resource type. Must match atlas-character.
func (r RestModel) GetName() string {
	return "characters"
}

// GetID returns the JSON:API resource id.
func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

// SetID parses the JSON:API resource id back into the model.
func (r *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

// SetToOneReferenceID and SetToManyReferenceIDs are required by api2go
// (jsonapi.Unmarshal) if the upstream response ever carries a
// `relationships` block. See libs/atlas-rest/CLAUDE.md.
func (r *RestModel) SetToOneReferenceID(_, _ string) error            { return nil }
func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }
```

- [ ] **Step 4: Write `requests.go`**

```go
package position

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "characters"
	ById     = Resource + "/%d"
)

// baseURLProvider is the seam used by tests to redirect requests to an
// httptest server. Production code resolves the caller's environment via
// requests.RootUrlFor("CHARACTERS").
var baseURLProvider = func(ctx context.Context) (string, error) { return requests.RootUrlFor(ctx, "CHARACTERS") }

func requestById(ctx context.Context, id uint32) requests.Request[RestModel] {
	root, err := baseURLProvider(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+ById, id))
}
```

- [ ] **Step 5: Write `processor.go`**

```go
package position

import (
	"context"

	"github.com/sirupsen/logrus"
)

// Processor is atlas-monsters' read-only client for atlas-character. Only
// the minimum surface needed by mob-skill AoE target selection is exposed.
type Processor interface {
	// GetPosition returns the character's last known world coordinates.
	// Errors propagate from the underlying REST call (e.g.
	// requests.ErrNotFound when the character does not exist).
	GetPosition(characterId uint32) (int16, int16, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewProcessor constructs a Processor scoped to the supplied tenant context.
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

// GetPosition fetches the character resource and projects (x, y) out of it.
func (p *ProcessorImpl) GetPosition(characterId uint32) (int16, int16, error) {
	rm, err := requestById(p.ctx, characterId)(p.l, p.ctx)
	if err != nil {
		return 0, 0, err
	}
	return rm.X, rm.Y, nil
}
```

- [ ] **Step 6: Write `mock/processor.go`**

```go
package mock

import (
	"atlas-monsters/character/position"
)

type ProcessorMock struct {
	GetPositionFunc func(characterId uint32) (int16, int16, error)
}

var _ position.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetPosition(characterId uint32) (int16, int16, error) {
	if m.GetPositionFunc != nil {
		return m.GetPositionFunc(characterId)
	}
	return 0, 0, nil
}
```

- [ ] **Step 7: Run the tests to verify they pass**

```bash
go build ./... && go test ./character/position/... -v
```

Expected: both tests PASS.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/character/position
git commit -m "feat(atlas-monsters): add read-only character position client"
```

---

## Task 2: the pure target selector

`selectDiseaseTargets` — box filter, SEDUCE-only cap, stable order, zero I/O. Also adds `SetCount` to `mobskill.ModelBuilder`, which the tests need and which does not exist today.

### Files

- `services/atlas-monsters/atlas.com/monsters/monster/mobskill/builder.go` — add a `count` field, `SetCount`, and wire `count` into `Build()`
- `services/atlas-monsters/atlas.com/monsters/monster/disease_targets.go` — **new file**; `positionedCharacter` and `selectDiseaseTargets`
- `services/atlas-monsters/atlas.com/monsters/monster/disease_targets_test.go` — **new file**; the table test

Read-only references:
- `services/atlas-monsters/atlas.com/monsters/monster/mobskill/model.go:59-99` — `Count()`, `HasBoundingBox()`, `LtX/LtY/RbX/RbY` all return `int32`/`uint32`
- `services/atlas-monsters/atlas.com/monsters/monster/processor.go` — the inclusive comparison form to copy, at lines 1201-1211 (`executeHeal`)
- `libs/atlas-constants/monster/skill.go:40` — `SkillTypeSeduce = 128`

Module root: `services/atlas-monsters/atlas.com/monsters`.

### Interfaces

- Consumes: nothing from Task 1.
- Produces, for Task 3:
  - `type positionedCharacter struct { id uint32; x int16; y int16 }` (package `monster`)
  - `func selectDiseaseTargets(mobX, mobY int16, sd mobskill.Model, skillId byte, candidates []positionedCharacter) []uint32`
  - `func (b *ModelBuilder) SetCount(count uint32) *ModelBuilder` (package `mobskill`)

- [ ] **Step 1: Write the failing test**

`services/atlas-monsters/atlas.com/monsters/monster/disease_targets_test.go`, package `monster`.

`TestSelectDiseaseTargets` — table-driven, one `t.Run` per case name below. Builder setup:
`mobskill.NewModelBuilder().SetBoundingBox(ltX, ltY, rbX, rbY).SetCount(count).Build()`.
Candidates are built as `[]positionedCharacter{{id: 1, x: 120, y: 210}, ...}`.

**The mob is at `(100, 200)` in every case. Box is `lt(-50, -30)`, `rb(50, 30)` in every case except `no bounding box`** — i.e. the absolute in-box rectangle is `x ∈ [50, 150]`, `y ∈ [170, 230]`.

Skill id column: `seduce` = `byte(monster2.SkillTypeSeduce)`, `slow` = `byte(monster2.SkillTypeSlow)`.

| case name | skill | count | candidates (id:x,y) | want |
|---|---|---|---|---|
| `in box and out of box` | slow | 0 | 1:120,210 · 2:400,210 | `[]uint32{1}` |
| `boundary is inclusive` | slow | 0 | 1:50,170 · 2:150,230 · 3:49,200 · 4:100,231 | `[]uint32{1, 2}` |
| `y outside box` | slow | 0 | 1:120,169 · 2:120,210 | `[]uint32{2}` |
| `non-seduce ignores count` | slow | 2 | 1:110,205 · 2:111,205 · 3:112,205 · 4:113,205 | `[]uint32{1, 2, 3, 4}` |
| `seduce caps at count` | seduce | 2 | 1:110,205 · 2:111,205 · 3:112,205 · 4:113,205 | `[]uint32{1, 2}` |
| `seduce count zero does not cap` | seduce | 0 | 1:110,205 · 2:111,205 · 3:112,205 | `[]uint32{1, 2, 3}` |
| `seduce count above candidate count` | seduce | 5 | 1:110,205 · 2:111,205 · 3:112,205 | `[]uint32{1, 2, 3}` |
| `seduce cap applies after box filter` | seduce | 2 | 9:400,205 · 1:110,205 · 2:111,205 · 3:112,205 | `[]uint32{1, 2}` |
| `no candidates` | seduce | 2 | *(empty slice)* | *(empty)* |
| `all candidates out of box` | slow | 0 | 1:400,205 · 2:401,205 | *(empty)* |

Assertion rule: for the two *(empty)* rows assert `len(got) == 0` (the function may return nil); for every other row assert exact slice equality against `want`.

`TestSelectDiseaseTargets_IsDeterministic` — separate function, no table. Builds the `seduce caps at count` inputs once, calls `selectDiseaseTargets` **twice** with the identical argument values, and asserts both results equal `[]uint32{1, 2}` and equal each other. This is the direct FR-4.2 assertion.

- [ ] **Step 2: Run the test to verify it fails**

```bash
go test ./monster/... -run TestSelectDiseaseTargets -v
```

Expected: build failure — `selectDiseaseTargets`, `positionedCharacter`, and `SetCount` are undefined.

- [ ] **Step 3: Add `SetCount` to the mobskill builder**

In `services/atlas-monsters/atlas.com/monsters/monster/mobskill/builder.go`: add `count uint32` to the `ModelBuilder` struct (after `duration`), add the setter, and add `count: b.count,` to the `Model` literal in `Build()`.

```go
// SetCount sets the maximum number of targets the skill affects. Only the
// SEDUCE disease honours this cap (task-259 FR-3.1); it is carried on every
// skill because the WZ data declares it on every skill.
func (b *ModelBuilder) SetCount(count uint32) *ModelBuilder {
	b.count = count
	return b
}
```

- [ ] **Step 4: Write `disease_targets.go`**

```go
package monster

import (
	"atlas-monsters/monster/mobskill"

	monster2 "github.com/Chronicle20/atlas/libs/atlas-constants/monster"
)

// positionedCharacter pairs a character id with the world coordinates that
// were successfully resolved for it. Characters whose position could not be
// resolved never become a positionedCharacter (FR-1.4).
type positionedCharacter struct {
	id uint32
	x  int16
	y  int16
}

// selectDiseaseTargets picks the characters a mob skill's disease applies to.
//
// It is a pure function of its arguments — no I/O, no randomness — so that a
// fixed candidate list always yields the same target list (FR-4.2). The
// bounding box is the skill's lt/rb offsets translated by the caster's
// position, tested inclusively, using the same arithmetic as the ally-heal
// AoE in executeHeal. The rectangle is never mirrored by facing (FR-1.3).
//
// The count cap is applied to SEDUCE only, matching the reference server,
// where the `i < count` guard sits inside the seduce branch. Every other
// disease — plus banish and dispel, which share this selector — applies to
// every candidate inside the rectangle.
func selectDiseaseTargets(mobX, mobY int16, sd mobskill.Model, skillId byte, candidates []positionedCharacter) []uint32 {
	var ids []uint32
	for _, c := range candidates {
		dx := int32(c.x) - int32(mobX)
		dy := int32(c.y) - int32(mobY)
		if dx >= sd.LtX() && dx <= sd.RbX() && dy >= sd.LtY() && dy <= sd.RbY() {
			ids = append(ids, c.id)
		}
	}

	if uint16(skillId) == monster2.SkillTypeSeduce && sd.Count() > 0 && uint32(len(ids)) > sd.Count() {
		ids = ids[:sd.Count()]
	}
	return ids
}
```

- [ ] **Step 5: Run the tests to verify they pass**

```bash
go build ./... && go test ./monster/... -run TestSelectDiseaseTargets -v
```

Expected: every subtest PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/disease_targets.go \
        services/atlas-monsters/atlas.com/monsters/monster/disease_targets_test.go \
        services/atlas-monsters/atlas.com/monsters/monster/mobskill/builder.go
git commit -m "feat(atlas-monsters): add pure box-scoped mob skill target selector"
```

---

## Task 3: the I/O shell and the positionFn seam

Wire `positionFn` onto `ProcessorImpl`, add the bounded position fan-out, and rewrite `getDiseaseTargets` to use the pure selector. Thread `skillId` through the three callers so the selector can apply the SEDUCE-only cap.

### Files

- `services/atlas-monsters/atlas.com/monsters/monster/processor.go` — add the `positionFn` struct field (`monster/processor.go:99-101`), wire it in `NewProcessor` (`monster/processor.go:113-121`), delete the old `getDiseaseTargets` (`monster/processor.go:1279-1305`), change the `executeBanish`/`executeDispel` signatures and their three call sites (`monster/processor.go:1216-1225`, `1247`, `1271`) and the `getDiseaseTargets` call sites (`monster/processor.go:1237`, `1261`, `1272`)
- `services/atlas-monsters/atlas.com/monsters/monster/disease_targets.go` — add `positionLookupConcurrency`, `resolvePositions`, and the new `getDiseaseTargets`
- `services/atlas-monsters/atlas.com/monsters/monster/disease_targets_shell_test.go` — **new file**; seam-driven shell tests

Read-only references:
- `services/atlas-monsters/atlas.com/monsters/monster/force_control_test.go:13-18` — how a test overrides `inFieldFn`
- `services/atlas-monsters/atlas.com/monsters/monster/control_assignment_test.go:17-29` — `recordingProcessor(ctx, tm, *int)`, the emit-counting `ProcessorImpl` constructor these tests build on
- `services/atlas-monsters/atlas.com/monsters/monster/builder.go:13-40` — `Clone(Model) *ModelBuilder` is the only builder entry point for `monster.Model`; `Clone(Model{})` is how a test builds one from scratch

Module root: `services/atlas-monsters/atlas.com/monsters`.

### Interfaces

- Consumes:
  - `position.NewProcessor(l, ctx).GetPosition(characterId uint32) (int16, int16, error)` (Task 1)
  - `selectDiseaseTargets(mobX, mobY int16, sd mobskill.Model, skillId byte, candidates []positionedCharacter) []uint32` and `positionedCharacter` (Task 2)
  - `mobskill.NewModelBuilder().SetCount(...)` (Task 2)
- Produces, for Task 4:
  - `ProcessorImpl.positionFn func(characterId uint32) (int16, int16, error)`
  - `func (p *ProcessorImpl) getDiseaseTargets(m Model, sd mobskill.Model, skillId byte) []uint32`
  - `func (p *ProcessorImpl) executeBanish(m Model, sd mobskill.Model, skillId byte)`
  - `func (p *ProcessorImpl) executeDispel(m Model, sd mobskill.Model, skillId byte)`

- [ ] **Step 1: Write the failing tests**

`services/atlas-monsters/atlas.com/monsters/monster/disease_targets_shell_test.go`, package `monster`.

A package-local helper (not a `*_testhelpers.go` file — this is a `_test.go` file):

```go
// diseaseTargetProcessor builds a ProcessorImpl with the two seams
// getDiseaseTargets consults: the field listing and the per-character
// position lookup. positionCalls records every id positionFn was asked for,
// so a test can assert the single-target path made no lookup at all.
func diseaseTargetProcessor(inField []uint32, positions map[uint32][2]int16, positionErr map[uint32]error, positionCalls *[]uint32) *ProcessorImpl
```

Build it on `recordingProcessor(context.Background(), newTestTenant(t), &emitted)` from `monster/control_assignment_test.go:17`, then assign `inFieldFn` and `positionFn`. `positionFn` appends the id to `*positionCalls`, returns `positionErr[id]` when that key is present, otherwise the `positions[id]` pair.

The mob is built with `Clone(Model{}).SetX(100).SetY(200).SetControlCharacterId(7).Build()` unless the case says otherwise. The AoE skill is `mobskill.NewModelBuilder().SetBoundingBox(-50, -30, 50, 30).SetCount(count).Build()`; the boxless skill is `mobskill.NewModelBuilder().SetCount(count).Build()`.

| test function | setup | expected |
|---|---|---|
| `TestGetDiseaseTargets_BoxlessWithMultiCountReturnsControllerOnly` | boxless skill, `count: 3`; mob `ControlCharacterId() == 7`; `inFieldFn` returns `[]uint32{7, 8, 9}` | returns `[]uint32{7}`; `*positionCalls` is empty (FR-2.3) |
| `TestGetDiseaseTargets_BoxlessWithNoControllerReturnsNothing` | boxless skill, `count: 3`; mob built with `SetControlCharacterId(0)` | `len(got) == 0`; `*positionCalls` is empty |
| `TestGetDiseaseTargets_FiltersByBoundingBox` | box skill, `count: 0`, skill `byte(monster2.SkillTypeSlow)`; `inFieldFn` returns `[]uint32{1, 2, 3}`; positions `1:(120,210)`, `2:(400,210)`, `3:(90,190)` | returns `[]uint32{1, 3}` |
| `TestGetDiseaseTargets_PreservesFieldListingOrder` | as above but `inFieldFn` returns `[]uint32{3, 1}` | returns `[]uint32{3, 1}` — output order is field-listing order regardless of goroutine completion order |
| `TestGetDiseaseTargets_PositionFailureExcludesOnlyThatCharacter` | box skill, `count: 0`, skill slow; `inFieldFn` returns `[]uint32{1, 2, 3}`; positions `1:(110,205)`, `3:(112,205)`; `positionErr[2] = errors.New("boom")` | returns `[]uint32{1, 3}` (FR-1.4) |
| `TestGetDiseaseTargets_FieldListingFailureReturnsNothing` | box skill; `inFieldFn` returns `(nil, errors.New("boom"))` | `len(got) == 0`; `*positionCalls` is empty (FR-5.4) |
| `TestGetDiseaseTargets_SeduceCapsAcrossTheShell` | box skill, `count: 2`, skill `byte(monster2.SkillTypeSeduce)`; `inFieldFn` returns `[]uint32{1, 2, 3, 4}`; all four at `(110, 205)`, `(111, 205)`, `(112, 205)`, `(113, 205)` | returns `[]uint32{1, 2}` |
| `TestGetDiseaseTargets_ConcurrentLookupsPreserveOrder` | box skill, `count: 0`, skill slow; `inFieldFn` returns 20 ids `1..20`; `positionFn` for **odd** ids sleeps 5ms before returning, all at `(110, 205)`; `positionCalls` guarded by a `sync.Mutex` in this test | returns exactly `[]uint32{1, 2, ..., 20}` in ascending order — proves index-based assembly, not completion order |

- [ ] **Step 2: Run the tests to verify they fail**

```bash
go test ./monster/... -run TestGetDiseaseTargets -v
```

Expected: build failure — `positionFn` is not a field of `ProcessorImpl`, and `getDiseaseTargets` takes two arguments, not three.

- [ ] **Step 3: Add the `positionFn` seam**

In `services/atlas-monsters/atlas.com/monsters/monster/processor.go`, add to the `ProcessorImpl` struct after `locationFn`:

```go
	positionFn func(characterId uint32) (int16, int16, error)
```

and in `NewProcessor`, after the `p.locationFn = ...` assignment:

```go
	p.positionFn = func(characterId uint32) (int16, int16, error) {
		return position.NewProcessor(p.l, p.ctx).GetPosition(characterId)
	}
```

Add `"atlas-monsters/character/position"` to the import block alongside `"atlas-monsters/character/hidden"`.

- [ ] **Step 4: Delete the old selector**

Remove the entire `getDiseaseTargets` function at `services/atlas-monsters/atlas.com/monsters/monster/processor.go:1279-1305`, including its doc comment. This removes the only `rand.Shuffle` call and the only remaining direct `_map.NewProcessor` construction in the skill path. Do **not** remove the `math/rand` import — `rand.Intn` at line 709 still uses it. Leave the `_map` import; it is still used elsewhere in the file (`monster/processor.go:113-121`).

- [ ] **Step 5: Thread `skillId` through the callers**

In `services/atlas-monsters/atlas.com/monsters/monster/processor.go`:

- `executeDebuff`'s dispel branch: `p.executeDispel(m, sd)` → `p.executeDispel(m, sd, skillId)`
- `executeDebuff`'s banish branch: `p.executeBanish(m, sd)` → `p.executeBanish(m, sd, skillId)`
- `executeDebuff`'s own selector call: `p.getDiseaseTargets(m, sd)` → `p.getDiseaseTargets(m, sd, skillId)`
- `func (p *ProcessorImpl) executeBanish(m Model, sd mobskill.Model)` → `func (p *ProcessorImpl) executeBanish(m Model, sd mobskill.Model, skillId byte)`, and its `p.getDiseaseTargets(m, sd)` → `p.getDiseaseTargets(m, sd, skillId)`
- `func (p *ProcessorImpl) executeDispel(m Model, sd mobskill.Model)` → `func (p *ProcessorImpl) executeDispel(m Model, sd mobskill.Model, skillId byte)`, same call-site change

Threading the real `skillId` rather than re-deriving `monster2.SkillTypeBanish`/`SkillTypeDispel` inside each function keeps the cap rule stated exactly once, in the selector.

- [ ] **Step 6: Write the shell into `disease_targets.go`**

Append to `services/atlas-monsters/atlas.com/monsters/monster/disease_targets.go`:

```go
// positionLookupConcurrency bounds the number of in-flight atlas-character
// reads for a single AoE cast. A crowded field must not serialize N round
// trips into the mob's skill execution path (FR-5.3), and must not open N
// sockets at once either.
const positionLookupConcurrency = 8

// resolvePositions looks up each character's world position through the
// positionFn seam, bounded to positionLookupConcurrency in flight.
//
// Results are assembled by input index, not by completion order, so the
// returned slice is always in field-listing order no matter how the
// goroutines interleave — that is what makes the concurrency compatible
// with the selector's determinism guarantee (FR-4.1).
//
// A character whose position cannot be resolved is logged at warn and
// dropped. One unresolvable character never aborts the cast for the others
// (FR-1.4), and the candidate set only ever shrinks — it never widens back
// to "everyone in the field".
func (p *ProcessorImpl) resolvePositions(uniqueId uint32, ids []uint32) []positionedCharacter {
	slots := make([]*positionedCharacter, len(ids))
	sem := make(chan struct{}, positionLookupConcurrency)
	var wg sync.WaitGroup
	for i, id := range ids {
		wg.Add(1)
		go func(i int, id uint32) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			x, y, err := p.positionFn(id)
			if err != nil {
				p.l.WithError(err).Warnf("Unable to resolve position for character [%d] targeted by monster [%d].", id, uniqueId)
				return
			}
			slots[i] = &positionedCharacter{id: id, x: x, y: y}
		}(i, id)
	}
	wg.Wait()

	out := make([]positionedCharacter, 0, len(ids))
	for _, s := range slots {
		if s != nil {
			out = append(out, *s)
		}
	}
	return out
}

// getDiseaseTargets returns the character ids a mob skill's disease, banish,
// or dispel applies to.
//
// A skill with no bounding box targets the mob's controller and nothing
// else, regardless of the skill's count (FR-2.1) — that path makes no
// position lookup at all. A skill with a bounding box lists the field,
// resolves each character's position, and hands the ordered candidate list
// to selectDiseaseTargets.
func (p *ProcessorImpl) getDiseaseTargets(m Model, sd mobskill.Model, skillId byte) []uint32 {
	if !sd.HasBoundingBox() {
		if m.ControlCharacterId() == 0 {
			return nil
		}
		return []uint32{m.ControlCharacterId()}
	}

	ids, err := p.inFieldFn(m.Field())
	if err != nil {
		p.l.WithError(err).Errorf("Unable to get characters in field for monster [%d] disease targeting.", m.UniqueId())
		return nil
	}

	candidates := p.resolvePositions(m.UniqueId(), ids)
	targets := selectDiseaseTargets(m.X(), m.Y(), sd, skillId, candidates)
	p.l.Debugf("Monster [%d] skill [%d] AoE: [%d] in field, [%d] positioned, [%d] targeted.",
		m.UniqueId(), skillId, len(ids), len(candidates), len(targets))
	return targets
}
```

Add `"sync"` to `disease_targets.go`'s import block.

- [ ] **Step 7: Run the tests to verify they pass**

```bash
go build ./... && go test ./monster/... -v
```

Expected: the new `TestGetDiseaseTargets_*` tests PASS and every pre-existing test in `./monster/...` still PASSes.

- [ ] **Step 8: Run the race detector on the fan-out**

```bash
go test ./monster/... -race -run TestGetDiseaseTargets
```

Expected: PASS with no race reports. The concurrent fan-out writes disjoint slice indices; this is the check that it actually does.

- [ ] **Step 9: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/processor.go \
        services/atlas-monsters/atlas.com/monsters/monster/disease_targets.go \
        services/atlas-monsters/atlas.com/monsters/monster/disease_targets_shell_test.go
git commit -m "feat(atlas-monsters): scope mob skill AoE targeting to the skill bounding box"
```

---

## Task 4: caller inheritance — banish and dispel

Banish and dispel inherit box scoping and the no-cap rule purely because they share the selector. Prove it. These two executors emit through `producer.ProviderImpl` directly, so they must first be routed through the existing `emit` seam — the same indirection `Damage`, `UseSkill`, and `DrainMp` already use — which changes no topic and no payload.

### Files

- `services/atlas-monsters/atlas.com/monsters/monster/processor.go` — route the three disease-path emissions through `p.emit` (`monster/processor.go:1240`, `1263`, `1274`), and add the `testInformationLookup` guard in `executeBanish` (`monster/processor.go:1249`)
- `services/atlas-monsters/atlas.com/monsters/monster/information/builder.go` — add `SetBanish`
- `services/atlas-monsters/atlas.com/monsters/monster/disease_callers_test.go` — **new file**

Read-only references:
- `services/atlas-monsters/atlas.com/monsters/monster/processor.go:1692-1696` — the `testInformationLookup` guard form to copy
- `services/atlas-monsters/atlas.com/monsters/monster/drain_mp_test.go:28-32` — how a test swaps and restores `testInformationLookup`
- `services/atlas-monsters/atlas.com/monsters/monster/information/model.go:26-30` — the `Banish` struct (`Message`, `MapId`, `PortalName`)
- `services/atlas-monsters/atlas.com/monsters/monster/processor_test.go:37-66` — `newRecordingProcessor`, which records `{Topic, Type}` per emission

Module root: `services/atlas-monsters/atlas.com/monsters`.

### Interfaces

- Consumes: `executeBanish(m, sd, skillId)` / `executeDispel(m, sd, skillId)` / `getDiseaseTargets(m, sd, skillId)` and the `positionFn` seam (Task 3); `mobskill.NewModelBuilder().SetCount(...)` (Task 2).
- Produces: `func (b *ModelBuilder) SetBanish(banish Banish) *ModelBuilder` (package `information`). Nothing else downstream.

- [ ] **Step 1: Write the failing tests**

`services/atlas-monsters/atlas.com/monsters/monster/disease_callers_test.go`, package `monster`.

Setup for all four: a `ProcessorImpl` from `newRecordingProcessor(t, newTestTenant(t))` (which returns `(*ProcessorImpl, *[]emittedEvent)`), with `inFieldFn` and `positionFn` assigned exactly as in `diseaseTargetProcessor` from Task 3's test file — reuse that helper if its signature fits, otherwise assign the two seams inline. Mob: `Clone(Model{}).SetX(100).SetY(200).SetControlCharacterId(7).Build()`. Box skill: `mobskill.NewModelBuilder().SetBoundingBox(-50, -30, 50, 30).SetCount(2).Build()`.

Field listing is `[]uint32{1, 2, 3}` with positions `1:(110, 205)` (in box), `2:(400, 205)` (out of box), `3:(112, 205)` (in box).

| test function | call | expected |
|---|---|---|
| `TestExecuteDispel_TargetsOnlyInBoxCharacters` | `p.executeDispel(mob, sd, byte(monster2.SkillTypeDispel))` | exactly 2 events, both on `EnvCommandTopicCharacterBuff` — the cap of 2 is *not* what limits this; assert the count is 2 because characters 1 and 3 are in the box and 2 is not |
| `TestExecuteDispel_NoCapForNonSeduce` | same seams but field listing `[]uint32{1, 2, 3, 4}` with positions `1:(110,205)`, `2:(111,205)`, `3:(112,205)`, `4:(113,205)` — all four in box — and `SetCount(2)` | exactly 4 events on `EnvCommandTopicCharacterBuff` (FR-3.1) |
| `TestExecuteBanish_TargetsOnlyInBoxCharacters` | `testInformationLookup` stubbed to return `information.NewModelBuilder().SetBanish(information.Banish{MapId: 104000000}).Build()`; `p.executeBanish(mob, sd, byte(monster2.SkillTypeBanish))` | exactly 2 events, both on `EnvCommandTopicPortal` |
| `TestExecuteBanish_NoBanishMapEmitsNothing` | `testInformationLookup` returns `information.NewModelBuilder().Build()` (MapId 0) | 0 events; unchanged early-return behavior |

Each banish test restores the previous hook: `prevHook := testInformationLookup; defer func() { testInformationLookup = prevHook }()`, per `monster/drain_mp_test.go:28-32`.

- [ ] **Step 2: Run the tests to verify they fail**

```bash
go test ./monster/... -run 'TestExecuteDispel|TestExecuteBanish' -v
```

Expected: build failure on `SetBanish` (undefined), and — once that compiles — the emission assertions fail at 0 recorded events, because both executors still publish through `producer.ProviderImpl` rather than the recorded `emit` seam.

- [ ] **Step 3: Add `SetBanish` to the information builder**

In `services/atlas-monsters/atlas.com/monsters/monster/information/builder.go`: add `banish Banish` to the `ModelBuilder` struct, add `banish: b.banish,` to the `Model` literal in `Build()`, and add:

```go
// SetBanish sets the banish destination on the builder. Used by tests that
// drive executeBanish, which early-returns when Banish().MapId is 0.
func (b *ModelBuilder) SetBanish(banish Banish) *ModelBuilder {
	b.banish = banish
	return b
}
```

- [ ] **Step 4: Route the disease-path emissions through the emit seam**

In `services/atlas-monsters/atlas.com/monsters/monster/processor.go`, replace the three `producer.ProviderImpl(p.l)(p.ctx)(TOPIC)(PROVIDER)` calls in `executeDebuff`, `executeBanish`, and `executeDispel` with `p.emit(TOPIC, PROVIDER)`. The topic constants and provider expressions are unchanged; only the publish indirection changes, and `NewProcessor` already wires `emit` to `producer.ProviderImpl`.

```go
	// executeDebuff
	err := p.emit(EnvCommandTopicCharacterBuff, applyDiseaseCommandProvider(m.Field(), characterId, uint16(skillId), uint16(skillLevel), diseaseName, value, duration))

	// executeBanish
	err := p.emit(EnvCommandTopicPortal, warpCommandProvider(m.Field(), characterId, map2.Id(banishMapId)))

	// executeDispel
	err := p.emit(EnvCommandTopicCharacterBuff, cancelAllBuffsCommandProvider(m.Field(), characterId))
```

- [ ] **Step 5: Add the `testInformationLookup` guard to `executeBanish`**

Replace the direct lookup at the top of `executeBanish`:

```go
	var ma information.Model
	var err error
	if testInformationLookup != nil {
		ma, err = testInformationLookup(m.MonsterId())
	} else {
		ma, err = information.NewProcessor(p.l, p.ctx).GetById(m.MonsterId())
	}
	if err != nil {
		p.l.WithError(err).Errorf("Unable to get monster info for banish from monster [%d].", m.UniqueId())
		return
	}
```

- [ ] **Step 6: Run the tests to verify they pass**

```bash
go build ./... && go test ./monster/... -v
```

Expected: all four new tests PASS and every pre-existing test in `./monster/...` still PASSes.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/processor.go \
        services/atlas-monsters/atlas.com/monsters/monster/information/builder.go \
        services/atlas-monsters/atlas.com/monsters/monster/disease_callers_test.go
git commit -m "test(atlas-monsters): assert banish and dispel inherit box-scoped targeting"
```

---

## Final verification

- [ ] **Step 1: Full-module build and test**

```bash
cd services/atlas-monsters/atlas.com/monsters && go build ./... && go test ./... -race
```

Expected: all packages PASS, no race reports.

- [ ] **Step 2: Confirm `rand.Shuffle` is gone and `rand.Intn` remains**

```bash
grep -rn 'rand\.' services/atlas-monsters/atlas.com/monsters/monster/processor.go
```

Expected: only the `rand.Intn` damage-formula lines (the comment and the call). No `rand.Shuffle`.

- [ ] **Step 3: Confirm no literal seduce id was introduced**

```bash
grep -rn 'SkillTypeSeduce' services/atlas-monsters/atlas.com/monsters/monster/
```

Expected: the hit in `disease_targets.go` is inside the cap condition, and reads
`monster2.SkillTypeSeduce` — never a literal `128`.

- [ ] **Step 4: Repo gate**

Run the flagless gate from the repository root:

```bash
tools/verify.sh
```

Expected: exit 0. This is the branch's "done" gate; `--quick` does not count.

- [ ] **Step 5: Code review**

Run `superpowers:requesting-code-review` before opening the PR. `backend-guidelines-reviewer` applies (Go service changes); `plan-adherence-reviewer` applies (this plan).
