# Monster Spawn Ground-Snap — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Non-flying monsters spawn flush with their intended foothold instead of in midair, by correcting the spawn point `Y` inside `atlas-data`'s `GetMonsters` serve path.

**Architecture:** Surface `Flying`/`Swimming` booleans on the monster template (derived from `AnimationTimes` keys). Add `findById` and `calcYOnFoothold` helpers on `FootholdTreeRestModel`. Apply a `snapToGround` transform inside `GetMonsters`: Fh-driven snap when `sp.FH != 0`, fallback to `calcPointBelow` (gated by template flags) when `sp.FH == 0`. atlas-maps and atlas-monsters are untouched; the corrected `Y` flows through the existing REST contract.

**Tech Stack:** Go 1.21+, atlas-data (`atlas-data` module), `logrus`, `gorm.io/gorm`, standard `testing` package with `logrus/hooks/test` null loggers and `tenant.WithContext` test fixtures.

---

## File Structure

**Modified files:**

- `services/atlas-data/atlas.com/data/monster/rest.go` — add `Flying bool` and `Swimming bool` on `RestModel`.
- `services/atlas-data/atlas.com/data/monster/reader.go` — derive `Flying` and `Swimming` from `AnimationTimes` map at parse time.
- `services/atlas-data/atlas.com/data/monster/reader_test.go` — assertions for the two new fields.
- `services/atlas-data/atlas.com/data/map/model.go` — add `findById` method to `FootholdTreeRestModel` and `calcYOnFoothold` package-level helper.
- `services/atlas-data/atlas.com/data/map/processor.go` — add `snapToGround`; modify `GetMonsters` signature and behavior.
- `services/atlas-data/atlas.com/data/map/resource.go` — wire monster `Storage` into `handleGetMapMonstersRequest`.

**Created files:**

- `services/atlas-data/atlas.com/data/map/model_test.go` — unit tests for `findById` and `calcYOnFoothold`.
- `services/atlas-data/atlas.com/data/map/processor_test.go` — unit tests for `snapToGround`.

No changes to other services. No new HTTP routes. No Kafka topic changes. No DB migrations.

---

## Task 1 — Surface `Flying` and `Swimming` on monster `RestModel`

**Files:**

- Modify: `services/atlas-data/atlas.com/data/monster/rest.go` (struct `RestModel`, around line 25 next to `AnimationTimes`)
- Modify: `services/atlas-data/atlas.com/data/monster/reader.go` (function `Read`, after line 88 where `m.AnimationTimes` is set)
- Test: `services/atlas-data/atlas.com/data/monster/reader_test.go` (extend existing `TestReader` and add a focused subtest)

- [ ] **Step 1.1: Add a failing test for the `Flying`/`Swimming` derivation**

The existing `testXML` (Pianus) has no `fly` or `swim` keys, so its expected values are both `false`. We assert that and add a new dedicated test using a small synthetic XML that does include a `fly` animation.

Append to `services/atlas-data/atlas.com/data/monster/reader_test.go` (after the closing brace of `TestReader` near line 1304, just before `TestRest`):

```go
func TestReaderFlyingFlag(t *testing.T) {
	tt := testTenant()
	l, _ := test.NewNullLogger()
	ctx := tenant.WithContext(context.Background(), tt)

	const flyXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="2230000.img">
  <imgdir name="info"><int name="maxHP" value="100"/></imgdir>
  <imgdir name="fly"><canvas name="0"><int name="delay" value="120"/></canvas></imgdir>
</imgdir>`

	_, _ = GetMonsterStringRegistry().Add(tt, MonsterString{id: strconv.Itoa(2230000), name: "FakeBat"})

	rm, err := Read(l)(ctx)(xml.FromByteArrayProvider([]byte(flyXML)))()
	if err != nil {
		t.Fatal(err)
	}
	if !rm.Flying {
		t.Fatalf("expected Flying=true, got false")
	}
	if rm.Swimming {
		t.Fatalf("expected Swimming=false, got true")
	}
}

func TestReaderSwimmingFlag(t *testing.T) {
	tt := testTenant()
	l, _ := test.NewNullLogger()
	ctx := tenant.WithContext(context.Background(), tt)

	const swimXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="2230100.img">
  <imgdir name="info"><int name="maxHP" value="100"/></imgdir>
  <imgdir name="hover"><canvas name="0"><int name="delay" value="120"/></canvas></imgdir>
</imgdir>`

	_, _ = GetMonsterStringRegistry().Add(tt, MonsterString{id: strconv.Itoa(2230100), name: "FakeFish"})

	rm, err := Read(l)(ctx)(xml.FromByteArrayProvider([]byte(swimXML)))()
	if err != nil {
		t.Fatal(err)
	}
	if !rm.Swimming {
		t.Fatalf("expected Swimming=true, got false")
	}
	if rm.Flying {
		t.Fatalf("expected Flying=false, got true")
	}
}

func TestReaderGroundFlags(t *testing.T) {
	tt := testTenant()
	l, _ := test.NewNullLogger()
	ctx := tenant.WithContext(context.Background(), tt)

	const groundXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="100100.img">
  <imgdir name="info"><int name="maxHP" value="100"/></imgdir>
  <imgdir name="move"><canvas name="0"><int name="delay" value="120"/></canvas></imgdir>
</imgdir>`

	_, _ = GetMonsterStringRegistry().Add(tt, MonsterString{id: strconv.Itoa(100100), name: "FakeSnail"})

	rm, err := Read(l)(ctx)(xml.FromByteArrayProvider([]byte(groundXML)))()
	if err != nil {
		t.Fatal(err)
	}
	if rm.Flying {
		t.Fatalf("expected Flying=false (ground mob), got true")
	}
	if rm.Swimming {
		t.Fatalf("expected Swimming=false (ground mob), got true")
	}
}
```

Also extend the existing `TestReader` (Pianus, around line 1281 right after the AnimationTimes loop) with an explicit assertion:

```go
	if rm.Flying {
		t.Errorf("Flying mismatch for Pianus: got true, expected false")
	}
	if rm.Swimming {
		t.Errorf("Swimming mismatch for Pianus: got true, expected false")
	}
```

- [ ] **Step 1.2: Run the new tests and confirm they fail**

```bash
cd services/atlas-data/atlas.com/data && go test ./monster/... -run 'TestReaderFlyingFlag|TestReaderSwimmingFlag|TestReaderGroundFlags' -v
```

Expected: compilation error `rm.Flying undefined` and `rm.Swimming undefined`.

- [ ] **Step 1.3: Add `Flying` and `Swimming` fields to `monster.RestModel`**

Edit `services/atlas-data/atlas.com/data/monster/rest.go`, replacing the existing `AnimationTimes` line with:

```go
	AnimationTimes     map[string]uint32 `json:"animation_times"`
	Flying             bool              `json:"flying"`
	Swimming           bool              `json:"swimming"`
```

- [ ] **Step 1.4: Derive the booleans inside `Read`**

Edit `services/atlas-data/atlas.com/data/monster/reader.go`. Find the line:

```go
		m.AnimationTimes = getAnimationTimes(exml)
```

and append immediately after it (still inside the closure, before `m.Revives = getRevives(node)`):

```go
		_, hasFly := m.AnimationTimes["fly"]
		_, hasHover := m.AnimationTimes["hover"]
		_, hasSwim := m.AnimationTimes["swim"]
		m.Flying = hasFly
		m.Swimming = hasHover || hasSwim
```

- [ ] **Step 1.5: Run the tests and confirm they pass**

```bash
cd services/atlas-data/atlas.com/data && go test ./monster/... -run 'TestReader' -v
```

Expected: `PASS` for `TestReader`, `TestReaderFlyingFlag`, `TestReaderSwimmingFlag`, `TestReaderGroundFlags`.

- [ ] **Step 1.6: Commit**

```bash
git add services/atlas-data/atlas.com/data/monster/rest.go \
        services/atlas-data/atlas.com/data/monster/reader.go \
        services/atlas-data/atlas.com/data/monster/reader_test.go
git commit -m "feat(atlas-data): expose Flying/Swimming on monster template

Derive from AnimationTimes keys (fly / hover / swim) at reader time.
Mirrors Cosmic MonsterStats.isMobile detection. Used by the upcoming
spawn-point ground-snap to skip flying and swimming mobs.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 2 — `findById` on `FootholdTreeRestModel`

**Files:**

- Modify: `services/atlas-data/atlas.com/data/map/model.go` (append after `findBelow`, around line 76)
- Test: `services/atlas-data/atlas.com/data/map/model_test.go` (new file)

- [ ] **Step 2.1: Write the failing test**

Create `services/atlas-data/atlas.com/data/map/model_test.go`:

```go
package _map

import (
	"atlas-data/point"
	"testing"
)

func buildSampleTree() *FootholdTreeRestModel {
	tree := NewFootholdTree(-1000, -1000, 1000, 1000)
	footholds := []FootholdRestModel{
		{Id: 1, First: &point.RestModel{X: -100, Y: 100}, Second: &point.RestModel{X: 100, Y: 100}},   // flat
		{Id: 2, First: &point.RestModel{X: 100, Y: 100}, Second: &point.RestModel{X: 300, Y: 200}},    // down-slope
		{Id: 3, First: &point.RestModel{X: -300, Y: 200}, Second: &point.RestModel{X: -100, Y: 100}},  // up-slope (going right means y decreases)
		{Id: 4, First: &point.RestModel{X: 500, Y: 0}, Second: &point.RestModel{X: 500, Y: 200}},      // wall
	}
	return tree.Insert(footholds)
}

func TestFootholdFindById(t *testing.T) {
	tree := buildSampleTree()

	if fh := tree.findById(1); fh == nil || fh.Id != 1 {
		t.Fatalf("findById(1) = %v, want id=1", fh)
	}
	if fh := tree.findById(4); fh == nil || fh.Id != 4 {
		t.Fatalf("findById(4) = %v, want id=4 (wall)", fh)
	}
	if fh := tree.findById(999); fh != nil {
		t.Fatalf("findById(999) = %v, want nil", fh)
	}
}
```

- [ ] **Step 2.2: Run, confirm fail**

```bash
cd services/atlas-data/atlas.com/data && go test ./map/... -run 'TestFootholdFindById' -v
```

Expected: compilation error `tree.findById undefined`.

- [ ] **Step 2.3: Implement `findById`**

Append to `services/atlas-data/atlas.com/data/map/model.go` (after the `findBelow` function ending at line 76, before `GetRelevant`):

```go
func (f *FootholdTreeRestModel) findById(id uint32) *FootholdRestModel {
	for i := range f.Footholds {
		if f.Footholds[i].Id == id {
			return &f.Footholds[i]
		}
	}
	for _, child := range []*FootholdTreeRestModel{f.NorthWest, f.NorthEast, f.SouthWest, f.SouthEast} {
		if child == nil {
			continue
		}
		if r := child.findById(id); r != nil {
			return r
		}
	}
	return nil
}
```

- [ ] **Step 2.4: Run, confirm pass**

```bash
cd services/atlas-data/atlas.com/data && go test ./map/... -run 'TestFootholdFindById' -v
```

Expected: `PASS`.

- [ ] **Step 2.5: Commit**

```bash
git add services/atlas-data/atlas.com/data/map/model.go \
        services/atlas-data/atlas.com/data/map/model_test.go
git commit -m "feat(atlas-data): add findById on FootholdTreeRestModel

Recursive lookup by foothold id across quadrants. Used by the
upcoming spawn-point ground-snap to resolve a spawn point's named
foothold without a tree-wide findBelow scan.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 3 — `calcYOnFoothold` helper

**Files:**

- Modify: `services/atlas-data/atlas.com/data/map/model.go` (append after `findById`)
- Test: `services/atlas-data/atlas.com/data/map/model_test.go` (extend)

- [ ] **Step 3.1: Write failing tests for all geometric branches**

Append to `services/atlas-data/atlas.com/data/map/model_test.go`:

```go
func TestCalcYOnFootholdFlat(t *testing.T) {
	fh := &FootholdRestModel{
		Id:     1,
		First:  &point.RestModel{X: -100, Y: 100},
		Second: &point.RestModel{X: 100, Y: 100},
	}
	y, ok := calcYOnFoothold(fh, 0)
	if !ok {
		t.Fatalf("calcYOnFoothold flat: ok=false, want true")
	}
	if y != 100 {
		t.Fatalf("calcYOnFoothold flat: y=%d, want 100", y)
	}
}

func TestCalcYOnFootholdDownSlope(t *testing.T) {
	// 200px wide, descends 100px: at x=200 (midpoint), y should be 150
	fh := &FootholdRestModel{
		Id:     2,
		First:  &point.RestModel{X: 100, Y: 100},
		Second: &point.RestModel{X: 300, Y: 200},
	}
	y, ok := calcYOnFoothold(fh, 200)
	if !ok {
		t.Fatalf("calcYOnFoothold down-slope: ok=false, want true")
	}
	if y < 145 || y > 155 {
		t.Fatalf("calcYOnFoothold down-slope mid: y=%d, want ~150", y)
	}
}

func TestCalcYOnFootholdUpSlope(t *testing.T) {
	// First.Y=200, Second.Y=100 — going right, y decreases
	fh := &FootholdRestModel{
		Id:     3,
		First:  &point.RestModel{X: -300, Y: 200},
		Second: &point.RestModel{X: -100, Y: 100},
	}
	y, ok := calcYOnFoothold(fh, -200)
	if !ok {
		t.Fatalf("calcYOnFoothold up-slope: ok=false, want true")
	}
	if y < 145 || y > 155 {
		t.Fatalf("calcYOnFoothold up-slope mid: y=%d, want ~150", y)
	}
}

func TestCalcYOnFootholdWall(t *testing.T) {
	fh := &FootholdRestModel{
		Id:     4,
		First:  &point.RestModel{X: 500, Y: 0},
		Second: &point.RestModel{X: 500, Y: 200},
	}
	if _, ok := calcYOnFoothold(fh, 500); ok {
		t.Fatalf("calcYOnFoothold wall: ok=true, want false")
	}
}

func TestCalcYOnFootholdOutOfSpan(t *testing.T) {
	fh := &FootholdRestModel{
		Id:     1,
		First:  &point.RestModel{X: -100, Y: 100},
		Second: &point.RestModel{X: 100, Y: 100},
	}
	if _, ok := calcYOnFoothold(fh, 500); ok {
		t.Fatalf("calcYOnFoothold out-of-span: ok=true, want false")
	}
	if _, ok := calcYOnFoothold(fh, -500); ok {
		t.Fatalf("calcYOnFoothold out-of-span (left): ok=true, want false")
	}
}
```

- [ ] **Step 3.2: Run, confirm fail**

```bash
cd services/atlas-data/atlas.com/data && go test ./map/... -run 'TestCalcYOnFoothold' -v
```

Expected: compilation error `undefined: calcYOnFoothold`.

- [ ] **Step 3.3: Implement `calcYOnFoothold`**

Append to `services/atlas-data/atlas.com/data/map/model.go` (after `findById`, before `GetRelevant`):

```go
func calcYOnFoothold(fh *FootholdRestModel, x int16) (int16, bool) {
	if fh == nil || fh.isWall() {
		return 0, false
	}
	if x < fh.First.X || x > fh.Second.X {
		return 0, false
	}
	if fh.First.Y == fh.Second.Y {
		return fh.First.Y, true
	}
	s1 := math.Abs(float64(fh.Second.Y - fh.First.Y))
	s2 := math.Abs(float64(fh.Second.X - fh.First.X))
	s4 := math.Abs(float64(x - fh.First.X))
	alpha := math.Atan(s2 / s1)
	beta := math.Atan(s1 / s2)
	s5 := math.Cos(alpha) * (s4 / math.Cos(beta))
	if fh.Second.Y < fh.First.Y {
		return fh.First.Y - int16(s5), true
	}
	return fh.First.Y + int16(s5), true
}
```

- [ ] **Step 3.4: Run, confirm pass**

```bash
cd services/atlas-data/atlas.com/data && go test ./map/... -run 'TestCalcYOnFoothold' -v
```

Expected: `PASS` for all five subtests.

- [ ] **Step 3.5: Commit**

```bash
git add services/atlas-data/atlas.com/data/map/model.go \
        services/atlas-data/atlas.com/data/map/model_test.go
git commit -m "feat(atlas-data): add calcYOnFoothold pure helper

Computes Y on a known foothold given X, mirroring the slope branch of
calcPointBelow. Walls and out-of-span X return (0, false). Used by the
spawn-point ground-snap to recompute Y on the spawn point's named
foothold.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 4 — `snapToGround` transform

**Files:**

- Modify: `services/atlas-data/atlas.com/data/map/processor.go` (add `snapToGround` near `calcPointBelow` around line 109)
- Test: `services/atlas-data/atlas.com/data/map/processor_test.go` (new file)

- [ ] **Step 4.1: Write failing tests for all five branches plus idempotency**

Create `services/atlas-data/atlas.com/data/map/processor_test.go`:

```go
package _map

import (
	"atlas-data/map/monster"
	monstertpl "atlas-data/monster"
	"atlas-data/point"
	"testing"
)

func snapTestTree() FootholdTreeRestModel {
	tree := NewFootholdTree(-2000, -2000, 2000, 2000)
	footholds := []FootholdRestModel{
		{Id: 10, First: &point.RestModel{X: -200, Y: 100}, Second: &point.RestModel{X: 200, Y: 100}},  // flat
		{Id: 11, First: &point.RestModel{X: 200, Y: 100}, Second: &point.RestModel{X: 400, Y: 200}},   // down-slope
	}
	return *tree.Insert(footholds)
}

func groundLookup(template uint32) (monstertpl.RestModel, error) {
	return monstertpl.RestModel{Id: template, Flying: false, Swimming: false}, nil
}
func flyingLookup(template uint32) (monstertpl.RestModel, error) {
	return monstertpl.RestModel{Id: template, Flying: true}, nil
}
func swimmingLookup(template uint32) (monstertpl.RestModel, error) {
	return monstertpl.RestModel{Id: template, Swimming: true}, nil
}
func errLookup(template uint32) (monstertpl.RestModel, error) {
	return monstertpl.RestModel{}, errMissingTemplate
}

func TestSnapToGround_FhSet_Flat_CorrectsY(t *testing.T) {
	tree := snapTestTree()
	sp := monster.RestModel{Id: 0, Template: 100100, X: 0, Y: 80, FH: 10}
	out := snapToGround(tree, sp, groundLookup)
	if out.Y != 100 {
		t.Fatalf("flat fh snap: Y=%d, want 100", out.Y)
	}
}

func TestSnapToGround_FhSet_Slope_CorrectsY(t *testing.T) {
	tree := snapTestTree()
	// midpoint of foothold 11 (x=300) is y≈150
	sp := monster.RestModel{Id: 0, Template: 100100, X: 300, Y: 80, FH: 11}
	out := snapToGround(tree, sp, groundLookup)
	if out.Y < 145 || out.Y > 155 {
		t.Fatalf("slope fh snap: Y=%d, want ~150", out.Y)
	}
}

func TestSnapToGround_FhSet_Missing_LeavesY(t *testing.T) {
	tree := snapTestTree()
	sp := monster.RestModel{Id: 0, Template: 100100, X: 0, Y: 80, FH: 9999}
	out := snapToGround(tree, sp, groundLookup)
	if out.Y != 80 {
		t.Fatalf("missing fh: Y=%d, want 80 (unchanged)", out.Y)
	}
}

func TestSnapToGround_FhZero_FlyingMob_LeavesY(t *testing.T) {
	tree := snapTestTree()
	sp := monster.RestModel{Id: 0, Template: 2230000, X: 0, Y: -300, FH: 0}
	out := snapToGround(tree, sp, flyingLookup)
	if out.Y != -300 {
		t.Fatalf("flying mob: Y=%d, want -300 (unchanged)", out.Y)
	}
}

func TestSnapToGround_FhZero_SwimmingMob_LeavesY(t *testing.T) {
	tree := snapTestTree()
	sp := monster.RestModel{Id: 0, Template: 2230100, X: 0, Y: 50, FH: 0}
	out := snapToGround(tree, sp, swimmingLookup)
	if out.Y != 50 {
		t.Fatalf("swimming mob: Y=%d, want 50 (unchanged)", out.Y)
	}
}

func TestSnapToGround_FhZero_GroundMob_FindsBelow(t *testing.T) {
	tree := snapTestTree()
	// X=0 is over flat foothold (Id=10) at Y=100. Spawn at Y=80, expect snap to ~99 (Y-1 offset).
	sp := monster.RestModel{Id: 0, Template: 100100, X: 0, Y: 80, FH: 0}
	out := snapToGround(tree, sp, groundLookup)
	if out.Y != 99 {
		t.Fatalf("ground mob fh=0 findBelow: Y=%d, want 99", out.Y)
	}
}

func TestSnapToGround_FhZero_NoFootholdBelow_LeavesY(t *testing.T) {
	tree := snapTestTree()
	// X=9999 is well outside any foothold span
	sp := monster.RestModel{Id: 0, Template: 100100, X: 9999, Y: 80, FH: 0}
	out := snapToGround(tree, sp, groundLookup)
	if out.Y != 80 {
		t.Fatalf("no foothold below: Y=%d, want 80 (unchanged)", out.Y)
	}
}

func TestSnapToGround_FhZero_TemplateLookupErr_LeavesY(t *testing.T) {
	tree := snapTestTree()
	sp := monster.RestModel{Id: 0, Template: 100100, X: 0, Y: 80, FH: 0}
	out := snapToGround(tree, sp, errLookup)
	if out.Y != 80 {
		t.Fatalf("template lookup err: Y=%d, want 80 (unchanged)", out.Y)
	}
}

func TestSnapToGround_Idempotent(t *testing.T) {
	tree := snapTestTree()
	sp := monster.RestModel{Id: 0, Template: 100100, X: 0, Y: 80, FH: 10}
	once := snapToGround(tree, sp, groundLookup)
	twice := snapToGround(tree, once, groundLookup)
	if once.Y != twice.Y {
		t.Fatalf("idempotency broken: once=%d, twice=%d", once.Y, twice.Y)
	}
}
```

- [ ] **Step 4.2: Run, confirm fail**

```bash
cd services/atlas-data/atlas.com/data && go test ./map/... -run 'TestSnapToGround' -v
```

Expected: compilation error — `undefined: snapToGround`, `undefined: errMissingTemplate`.

- [ ] **Step 4.3: Implement `snapToGround` and the sentinel error**

Edit `services/atlas-data/atlas.com/data/map/processor.go`. Add the import:

```go
import (
    // ... existing imports ...
    monstertpl "atlas-data/monster"
)
```

Note: the existing import alias `"atlas-data/map/monster"` (spawn-point package) stays as `monster`. The new import uses alias `monstertpl` to disambiguate.

Append after `calcPointBelow` (around line 127, before `type FootholdTreeConfigurator`):

```go
var errMissingTemplate = errors.New("monster template not found")

type templateLookup func(uint32) (monstertpl.RestModel, error)

func snapToGround(tree FootholdTreeRestModel, sp monster.RestModel, lookup templateLookup) monster.RestModel {
	if sp.FH != 0 {
		fh := tree.findById(uint32(sp.FH))
		if fh == nil {
			return sp
		}
		if y, ok := calcYOnFoothold(fh, sp.X); ok {
			sp.Y = y
		}
		return sp
	}
	tpl, err := lookup(sp.Template)
	if err != nil {
		return sp
	}
	if tpl.Flying || tpl.Swimming {
		return sp
	}
	if pt, ok := calcPointBelow(tree, point.RestModel{X: sp.X, Y: sp.Y - 1}); ok {
		sp.Y = pt.Y - 1
	}
	return sp
}
```

Add `"errors"` to the import block at the top of `processor.go` if it isn't already there.

- [ ] **Step 4.4: Run, confirm pass**

```bash
cd services/atlas-data/atlas.com/data && go test ./map/... -run 'TestSnapToGround' -v
```

Expected: all nine subtests `PASS`.

- [ ] **Step 4.5: Commit**

```bash
git add services/atlas-data/atlas.com/data/map/processor.go \
        services/atlas-data/atlas.com/data/map/processor_test.go
git commit -m "feat(atlas-data): add snapToGround spawn-point transform

Fh-driven snap when sp.FH != 0 (recomputes Y on the named foothold).
Falls back to calcPointBelow when sp.FH == 0, gated by template
Flying/Swimming flags so airborne and aquatic mobs keep their
intentional Y. Pure function — caller threads the template lookup.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 5 — Wire `snapToGround` into `GetMonsters`

**Files:**

- Modify: `services/atlas-data/atlas.com/data/map/processor.go` (function `monsterProvider`, `GetMonsters` near line 325-343)
- Modify: `services/atlas-data/atlas.com/data/map/resource.go` (function `handleGetMapMonstersRequest` near line 301-319)

- [ ] **Step 5.1: Update `monsterProvider` and `GetMonsters` to apply the snap**

Edit `services/atlas-data/atlas.com/data/map/processor.go`. Replace the existing `monsterProvider` and `GetMonsters` definitions (lines 325-343) with:

```go
func monsterProvider(s *Storage, ms *monstertpl.Storage) func(ctx context.Context) func(mapId _map.Id) model.Provider[[]monster.RestModel] {
	return func(ctx context.Context) func(mapId _map.Id) model.Provider[[]monster.RestModel] {
		return func(mapId _map.Id) model.Provider[[]monster.RestModel] {
			m, err := s.ByIdProvider(ctx)(strconv.Itoa(int(mapId)))()
			if err != nil {
				return model.ErrorProvider[[]monster.RestModel](err)
			}
			lookup := func(template uint32) (monstertpl.RestModel, error) {
				return ms.GetById(ctx)(strconv.Itoa(int(template)))
			}
			snapped := make([]monster.RestModel, 0, len(m.Monsters))
			for _, sp := range m.Monsters {
				snapped = append(snapped, snapToGround(m.FootholdTree, sp, lookup))
			}
			return model.FixedProvider(snapped)
		}
	}
}

func GetMonsters(s *Storage, ms *monstertpl.Storage) func(ctx context.Context) func(mapId _map.Id) ([]monster.RestModel, error) {
	return func(ctx context.Context) func(mapId _map.Id) ([]monster.RestModel, error) {
		return func(mapId _map.Id) ([]monster.RestModel, error) {
			return monsterProvider(s, ms)(ctx)(mapId)()
		}
	}
}
```

- [ ] **Step 5.2: Update the resource handler to construct and pass monster `Storage`**

Edit `services/atlas-data/atlas.com/data/map/resource.go`. Add the import:

```go
import (
    // ... existing imports ...
    monstertpl "atlas-data/monster"
)
```

Replace the body of `handleGetMapMonstersRequest` (lines 301-319) with:

```go
func handleGetMapMonstersRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseMapId(d.Logger(), func(mapId _map.Id) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				s := NewStorage(d.Logger(), db)
				ms := monstertpl.NewStorage(d.Logger(), db)
				res, err := GetMonsters(s, ms)(d.Context())(mapId)
				if err != nil {
					d.Logger().WithError(err).Debugf("Unable to locate map %d.", mapId)
					w.WriteHeader(http.StatusNotFound)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[[]monster.RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
			}
		})
	}
}
```

- [ ] **Step 5.3: Run the package's tests to confirm nothing else broke**

```bash
cd services/atlas-data/atlas.com/data && go test ./map/... ./monster/... -v
```

Expected: all `PASS`. If `TestReader` or others fail because `GetMonsters` is now used internally with a different signature, fix the test wiring (search for `GetMonsters(` references — there should be none in tests today, but verify with the next step).

- [ ] **Step 5.4: Search for any other call sites of `GetMonsters` and confirm they don't exist outside the resource layer**

```bash
grep -rn "GetMonsters\|monsterProvider" services/atlas-data/atlas.com/data
```

Expected output: only the new definitions in `processor.go` and the call in `resource.go`. If there are other callers, update them to the new signature.

- [ ] **Step 5.5: Commit**

```bash
git add services/atlas-data/atlas.com/data/map/processor.go \
        services/atlas-data/atlas.com/data/map/resource.go
git commit -m "feat(atlas-data): apply spawn-point ground-snap in GetMonsters

GetMonsters now threads monster.Storage into snapToGround for each
spawn point on the served map. Flying/swimming mobs keep their Y
verbatim; ground mobs land on the named (or detected) foothold.
atlas-maps and downstream consumers receive corrected Y through the
existing GET /data/maps/{mapId}/monsters contract.

Fixes monsters falling from midair on maps with sloppy WZ Y values
(e.g. 910310002).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 6 — Service-wide verification

**Files:** none modified directly; this task validates the full service builds and tests cleanly.

- [ ] **Step 6.1: Run all tests in atlas-data**

```bash
cd services/atlas-data/atlas.com/data && go test ./... -count=1
```

Expected: all packages `PASS`. If any test in a sibling package (e.g., `map/rest_test.go`, `resource_test.go`) was relying on `GetMonsters`'s old signature or on the un-snapped `Y`, fix the test fixtures to match the new behavior. Specifically:

- `services/atlas-data/atlas.com/data/map/resource_test.go` — likely mocks the storage and asserts on the returned shape. If it asserts spawn point `Y` values, accept the new snapped values.
- `services/atlas-data/atlas.com/data/map/rest_test.go` — assertions on `RestModel` shape. New `Flying`/`Swimming` JSON fields are additive; tests should not break unless they pin the exact JSON output.

If a test fails because it pinned the old behavior, update the assertion to the new snapped value (compute it manually using the foothold geometry in the fixture) and re-run. Don't disable tests.

- [ ] **Step 6.2: Run `go vet` and `go build` to catch wiring slips**

```bash
cd services/atlas-data/atlas.com/data && go vet ./... && go build ./...
```

Expected: no output (clean).

- [ ] **Step 6.3: Build the docker image**

```bash
cd services/atlas-data/atlas.com && docker compose build atlas-data
```

Expected: build succeeds.

- [ ] **Step 6.4: If Step 6.1 required test fixups, commit them**

```bash
git add services/atlas-data/atlas.com/data
git commit -m "test(atlas-data): adjust fixtures for ground-snap behavior

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

If no fixups were needed, skip this commit.

- [ ] **Step 6.5: Final sanity check**

```bash
git log --oneline main..HEAD
```

Expected commit list (in order):

1. `docs(task-025): add PRD and design for monster spawn ground-snap`
2. `feat(atlas-data): expose Flying/Swimming on monster template`
3. `feat(atlas-data): add findById on FootholdTreeRestModel`
4. `feat(atlas-data): add calcYOnFoothold pure helper`
5. `feat(atlas-data): add snapToGround spawn-point transform`
6. `feat(atlas-data): apply spawn-point ground-snap in GetMonsters`
7. *(optional)* `test(atlas-data): adjust fixtures for ground-snap behavior`

Six or seven commits depending on whether Step 6.4 was needed.

---

## Manual smoke (post-merge / on user return)

Not a plan task — this is for the user when they get home from the gym:

1. Bring up the stack with the worktree's atlas-data image (or rebuild via `docker compose up --build atlas-data atlas-maps atlas-monsters`).
2. Log a character into map `910310002`. Watch the monster spawn — non-flying mobs should appear flush against their platform with no fall.
3. Log into a flying-mob map (any cave with bats, or `100020000` Lith Harbor outside) — bats should still spawn at their air `Y`.
4. Log into Aquarium `230000000` if a swim mob test is desired.

If smoke passes, merge `feature/task-025-monster-spawn-ground-snap` into `main`.

If smoke fails (e.g., a fish on Aquarium snapped to seabed):

- Confirm the affected mob's `AnimationTimes` keys (`GET /data/monsters/{id}` on the running atlas-data) — if `swim`/`hover` is missing, expand the detection key set in `services/atlas-data/atlas.com/data/monster/reader.go` and add a regression test.
- Confirm `Fh` on the affected spawn point (`GET /data/maps/{mapId}/monsters`) — if `Fh != 0` for a swim mob, the `Fh != 0` branch will snap regardless of Flying/Swimming. Decide whether to gate the Fh-set branch on the template flags too. (Current design intentionally trusts `Fh` over flags because data placers wouldn't set `Fh` on a swim mob — but if WZ sometimes does, the gate needs to widen.)
