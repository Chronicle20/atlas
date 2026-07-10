# MP Recovery (Brawler 5101005) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the server-side effect of Brawler MP Recovery (skill 5101005): casting costs `floor(MaxHP / x)` HP and restores `floor(hpLost * y / 100)` MP, driven by the tenant version's skill effect data.

**Architecture:** A new per-skill handler subpackage `skill/handler/mprecovery` in atlas-channel, registered via `init()` in the existing handler registry and dispatched from the end of `UseSkill`. The handler loads the caster's MaxHp, computes the amounts with a pure formula function, and emits `ChangeHP` / `ChangeMP` commands through the existing `character.Processor` (Kafka → atlas-character → existing stat-changed client flow). Side effects are package-level `var` seams (mysticdoor style) so tests override them without mock plumbing. No packet, REST, Kafka-topic, or migration changes.

**Tech Stack:** Go, logrus, existing atlas-channel `character.Processor`, `data/skill/effect` model, `libs/atlas-constants/skill` (`BrawlerMPRecoveryId` already exists), Go stdlib testing.

## Global Constraints

- Only **atlas-channel** is touched (`services/atlas-channel/atlas.com/channel/`); atlas-data and libs need no change.
- `x`/`y` MUST be read from the effect model (`e.X()`, `e.Y()`), never hardcoded (v83 ground truth for tests only: x=10 all levels; y=55 L1, 75 L5, 100 L10).
- **No low-HP guard** (owner decision, PRD FR-3): the handler emits the full `-hpLost` regardless of current HP; atlas-character's existing ChangeHP semantics own the 0-floor/death path. No HP floor logic in the handler.
- `mpGain` is computed from the full intended `hpLost`, not any post-clamp delta (Cosmic parity, PRD FR-2).
- Never a partial effect where MP is gained without the HP cost having been requested (PRD FR-5).
- Tests use the project Builder pattern / in-package construction; **no `*_testhelpers.go` files**.
- Verification gates before done: `go test -race ./...`, `go vet ./...`, `go build ./...` clean in the atlas-channel module; `tools/redis-key-guard.sh` clean from the worktree root. `docker buildx bake` NOT required (no `go.mod` change).
- Never write literal home/absolute paths into committed files.
- All `go` commands below run from `services/atlas-channel/atlas.com/channel/` (the module root). All `git` commands run from the worktree root with repo-relative paths.

---

### Task 1: `Y()` getter on the skill effect model

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/data/skill/effect/model.go` (append after the `X()` getter at the end of the file, line 147)
- Test: `services/atlas-channel/atlas.com/channel/data/skill/effect/model_test.go` (create)

**Interfaces:**
- Consumes: existing `Model` struct — the `y int16` field already exists (`model.go:41`) and is already populated from REST (`rest.go:115`, `Y int16 \`json:"y"\``).
- Produces: `func (m Model) Y() int16` — Task 3's handler calls `e.Y()`.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/data/skill/effect/model_test.go`:

```go
package effect

import "testing"

// TestModelY pins the Y() getter added for MP Recovery (task-151): y is
// already populated from REST (rest.go); only the getter was missing.
func TestModelY(t *testing.T) {
	m := Model{y: 55}
	if got := m.Y(); got != 55 {
		t.Fatalf("Y() = %d, want 55", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run (from `services/atlas-channel/atlas.com/channel/`): `go test ./data/skill/effect/ -run TestModelY -v`
Expected: FAIL to compile with `m.Y undefined (type Model has no field or method Y)`

- [ ] **Step 3: Write minimal implementation**

Append to `services/atlas-channel/atlas.com/channel/data/skill/effect/model.go` (after the `X()` getter):

```go
// Y returns the integer Y attribute (for MP Recovery it is the percent of
// the HP loss returned as MP).
func (m Model) Y() int16 {
	return m.y
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./data/skill/effect/ -run TestModelY -v`
Expected: `--- PASS: TestModelY` then `PASS`

- [ ] **Step 5: Commit**

From the worktree root:

```bash
git add services/atlas-channel/atlas.com/channel/data/skill/effect/model.go services/atlas-channel/atlas.com/channel/data/skill/effect/model_test.go
git commit -m "feat(channel): expose Y() on skill effect model for MP Recovery"
```

---

### Task 2: `Amounts` formula (pure function)

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/skill/handler/mprecovery/formula.go`
- Test: `services/atlas-channel/atlas.com/channel/skill/handler/mprecovery/formula_test.go`

**Interfaces:**
- Consumes: nothing from other tasks (pure stdlib).
- Produces: `func Amounts(maxHp uint16, x int16, y int16) (int16, int16)` returning `(hpLost, mpGain)` — Task 3's handler calls it. `x <= 0` returns `(0, 0)`; results are clamped so pathological data cannot wrap negative.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/skill/handler/mprecovery/formula_test.go`:

```go
package mprecovery

import (
	"math"
	"testing"
)

// TestAmounts pins the Cosmic SpecialMoveHandler formula against
// WZ-verified v83 values for 5101005 (x=10 at every level; y=55 at L1,
// 75 at L5, 100 at L10). Integer floor division at each step.
func TestAmounts(t *testing.T) {
	tests := []struct {
		name       string
		maxHp      uint16
		x          int16
		y          int16
		wantHpLost int16
		wantMpGain int16
	}{
		{name: "L1 v83 (x=10,y=55) maxHp=1234", maxHp: 1234, x: 10, y: 55, wantHpLost: 123, wantMpGain: 67},
		{name: "L5 v83 (x=10,y=75) maxHp=1234", maxHp: 1234, x: 10, y: 75, wantHpLost: 123, wantMpGain: 92},
		{name: "L10 v83 (x=10,y=100) maxHp=1234", maxHp: 1234, x: 10, y: 100, wantHpLost: 123, wantMpGain: 123},
		{name: "L10 v83 maxHp=30000", maxHp: 30000, x: 10, y: 100, wantHpLost: 3000, wantMpGain: 3000},
		{name: "floor on mpGain (hpLost*y not divisible)", maxHp: 100, x: 10, y: 55, wantHpLost: 10, wantMpGain: 5},
		{name: "maxHp below x floors hpLost to zero", maxHp: 9, x: 10, y: 100, wantHpLost: 0, wantMpGain: 0},
		{name: "x=0 returns zeros (bad tenant data)", maxHp: 1234, x: 0, y: 55, wantHpLost: 0, wantMpGain: 0},
		{name: "x negative returns zeros", maxHp: 1234, x: -5, y: 55, wantHpLost: 0, wantMpGain: 0},
		{name: "pathological x=1 at uint16 max clamps, never wraps", maxHp: math.MaxUint16, x: 1, y: 100, wantHpLost: math.MaxInt16, wantMpGain: math.MaxInt16},
		{name: "negative y floors mpGain at zero", maxHp: 1234, x: 10, y: -50, wantHpLost: 123, wantMpGain: 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotHp, gotMp := Amounts(tc.maxHp, tc.x, tc.y)
			if gotHp != tc.wantHpLost || gotMp != tc.wantMpGain {
				t.Fatalf("Amounts(%d, %d, %d) = (%d, %d), want (%d, %d)",
					tc.maxHp, tc.x, tc.y, gotHp, gotMp, tc.wantHpLost, tc.wantMpGain)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./skill/handler/mprecovery/ -v`
Expected: FAIL to compile with `undefined: Amounts`

- [ ] **Step 3: Write minimal implementation**

Create `services/atlas-channel/atlas.com/channel/skill/handler/mprecovery/formula.go`:

```go
package mprecovery

import "math"

// Amounts returns (hpLost, mpGain) per Cosmic SpecialMoveHandler.java:118-124:
// hpLost = maxHp / x, mpGain = hpLost * y / 100, integer floor division.
// mpGain is computed from the full intended loss, not any post-clamp delta.
// x <= 0 returns (0, 0) — the caller treats that as "skip, warn" (bad tenant
// data). Computation is int32 then narrowed with a MaxInt16 clamp so
// pathological tenant data can never wrap negative; a negative y floors
// mpGain at zero.
func Amounts(maxHp uint16, x int16, y int16) (int16, int16) {
	if x <= 0 {
		return 0, 0
	}
	hpLost := int32(maxHp) / int32(x)
	mpGain := hpLost * int32(y) / 100
	if hpLost > math.MaxInt16 {
		hpLost = math.MaxInt16
	}
	if mpGain > math.MaxInt16 {
		mpGain = math.MaxInt16
	}
	if mpGain < 0 {
		mpGain = 0
	}
	return int16(hpLost), int16(mpGain)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./skill/handler/mprecovery/ -v`
Expected: all `TestAmounts` subtests PASS

- [ ] **Step 5: Commit**

From the worktree root:

```bash
git add services/atlas-channel/atlas.com/channel/skill/handler/mprecovery/formula.go services/atlas-channel/atlas.com/channel/skill/handler/mprecovery/formula_test.go
git commit -m "feat(channel): MP Recovery amount formula (Cosmic parity)"
```

---

### Task 3: MP Recovery handler with seams and registration

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/skill/handler/mprecovery/mprecovery.go`
- Test: `services/atlas-channel/atlas.com/channel/skill/handler/mprecovery/mprecovery_test.go`

**Interfaces:**
- Consumes:
  - `Amounts(maxHp uint16, x int16, y int16) (int16, int16)` from Task 2.
  - `effect.Model.X() int16` (existing) and `effect.Model.Y() int16` from Task 1.
  - `channelhandler.Register(id skill2.Id, h channelhandler.Handler)` and `channelhandler.Lookup(id skill2.Id) (Handler, bool)` from `atlas-channel/skill/handler/registry.go`.
  - `channelhandler.Handler` signature: `func(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer, f field.Model, characterId uint32, info packetmodel.SkillUsageInfo, e effect.Model) error` (`registry.go:18-24`).
  - `character.NewProcessor(l, ctx).GetById()(characterId) (character.Model, error)`; `character.Model.MaxHp() uint16` (`character/model.go:135`).
  - `character.Processor.ChangeHP(f field.Model, characterId uint32, amount int16) error` and `ChangeMP(f field.Model, characterId uint32, amount int16) error` (`character/processor.go:41-42`).
  - `skill2.BrawlerMPRecoveryId` (= `Id(5101005)`, `libs/atlas-constants/skill/constants.go:3194`).
- Produces: `mprecovery.Apply` registered under `skill2.BrawlerMPRecoveryId` via `init()`; package-level seams `loadCaster`, `changeHP`, `changeMP` (test overrides). Task 4 blank-imports this package.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/skill/handler/mprecovery/mprecovery_test.go`:

```go
package mprecovery

import (
	"context"
	"errors"
	"testing"

	"atlas-channel/data/skill/effect"
	channelhandler "atlas-channel/skill/handler"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/sirupsen/logrus"
)

const (
	testCharId = uint32(1001)
	testLevel  = byte(5)
	testMaxHp  = uint16(1234)
)

func testLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetLevel(logrus.DebugLevel)
	return l
}

func testField() field.Model {
	return field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).Build()
}

func testInfo() packetmodel.SkillUsageInfo {
	return packetmodel.NewSkillUsageInfoBuilder().
		SetSkillId(uint32(skill2.BrawlerMPRecoveryId)).
		SetSkillLevel(testLevel).
		Build()
}

// testEffect builds an effect.Model with the given x/y through the REST
// extract path (no builder exists on the model; Extract is the production
// construction seam and exercises the Y() getter end-to-end).
func testEffect(t *testing.T, x int16, y int16) effect.Model {
	t.Helper()
	e, err := effect.Extract(effect.RestModel{X: x, Y: y})
	if err != nil {
		t.Fatalf("effect.Extract returned error: %v", err)
	}
	return e
}

// call records one seam invocation, preserving order across seams.
type call struct {
	name   string
	amount int16
}

// invokeApply overrides the three seams, calls Apply, and returns the
// ordered seam calls plus Apply's error.
func invokeApply(
	t *testing.T,
	casterLoader func(logrus.FieldLogger, context.Context, uint32) (uint16, error),
	hpErr error,
	mpErr error,
	e effect.Model,
) ([]call, error) {
	t.Helper()
	origCaster, origHP, origMP := loadCaster, changeHP, changeMP
	t.Cleanup(func() {
		loadCaster, changeHP, changeMP = origCaster, origHP, origMP
	})

	var calls []call
	loadCaster = casterLoader
	changeHP = func(_ logrus.FieldLogger, _ context.Context, _ field.Model, _ uint32, amount int16) error {
		calls = append(calls, call{name: "changeHP", amount: amount})
		return hpErr
	}
	changeMP = func(_ logrus.FieldLogger, _ context.Context, _ field.Model, _ uint32, amount int16) error {
		calls = append(calls, call{name: "changeMP", amount: amount})
		return mpErr
	}

	err := Apply(testLogger())(context.Background())(nil, testField(), testCharId, testInfo(), e)
	return calls, err
}

func happyCasterLoader(_ logrus.FieldLogger, _ context.Context, _ uint32) (uint16, error) {
	return testMaxHp, nil
}

// TestMPRecoveryRegistered: init() installs Apply in the shared registry.
func TestMPRecoveryRegistered(t *testing.T) {
	h, ok := channelhandler.Lookup(skill2.BrawlerMPRecoveryId)
	if !ok || h == nil {
		t.Fatalf("Lookup(BrawlerMPRecoveryId) = (%v, %v), want non-nil handler", h, ok)
	}
}

// TestMPRecoveryHappyPath: v83 L5 (x=10, y=75), MaxHp 1234 -> ChangeHP(-123)
// then ChangeMP(+92), in that order. The handler has no currentHP input, so
// this also pins FR-3: the full unclamped loss is emitted regardless of the
// caster's current HP (atlas-character owns the 0-floor/death path).
func TestMPRecoveryHappyPath(t *testing.T) {
	calls, err := invokeApply(t, happyCasterLoader, nil, nil, testEffect(t, 10, 75))
	if err != nil {
		t.Fatalf("Apply returned unexpected error: %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("got %d seam calls %v, want 2 (changeHP then changeMP)", len(calls), calls)
	}
	if calls[0].name != "changeHP" || calls[0].amount != -123 {
		t.Fatalf("first call = %+v, want changeHP with -123", calls[0])
	}
	if calls[1].name != "changeMP" || calls[1].amount != 92 {
		t.Fatalf("second call = %+v, want changeMP with +92", calls[1])
	}
}

// TestMPRecoveryCasterLoadError: FR-5 — load failure emits nothing and
// surfaces the error.
func TestMPRecoveryCasterLoadError(t *testing.T) {
	loadErr := errors.New("character service unavailable")
	calls, err := invokeApply(t,
		func(_ logrus.FieldLogger, _ context.Context, _ uint32) (uint16, error) {
			return 0, loadErr
		},
		nil, nil, testEffect(t, 10, 75))
	if !errors.Is(err, loadErr) {
		t.Fatalf("Apply error = %v, want %v", err, loadErr)
	}
	if len(calls) != 0 {
		t.Fatalf("got seam calls %v, want none on caster load failure", calls)
	}
}

// TestMPRecoveryChangeHPError: FR-5 — never MP gain without the HP cost
// having been requested; a ChangeHP emit failure skips ChangeMP.
func TestMPRecoveryChangeHPError(t *testing.T) {
	hpErr := errors.New("emit failed")
	calls, err := invokeApply(t, happyCasterLoader, hpErr, nil, testEffect(t, 10, 75))
	if !errors.Is(err, hpErr) {
		t.Fatalf("Apply error = %v, want %v", err, hpErr)
	}
	if len(calls) != 1 || calls[0].name != "changeHP" {
		t.Fatalf("got seam calls %v, want exactly one changeHP call", calls)
	}
}

// TestMPRecoveryChangeMPError: the HP cost was already requested (matches
// Cosmic, which applies the loss first); the MP error is surfaced.
func TestMPRecoveryChangeMPError(t *testing.T) {
	mpErr := errors.New("emit failed")
	calls, err := invokeApply(t, happyCasterLoader, nil, mpErr, testEffect(t, 10, 75))
	if !errors.Is(err, mpErr) {
		t.Fatalf("Apply error = %v, want %v", err, mpErr)
	}
	if len(calls) != 2 {
		t.Fatalf("got seam calls %v, want changeHP then changeMP", calls)
	}
}

// TestMPRecoveryBadDataSkips: x=0 (bad tenant data) -> warn, nil error,
// zero emits. Never divide by zero, never emit a zero delta.
func TestMPRecoveryBadDataSkips(t *testing.T) {
	calls, err := invokeApply(t, happyCasterLoader, nil, nil, testEffect(t, 0, 75))
	if err != nil {
		t.Fatalf("Apply returned unexpected error: %v", err)
	}
	if len(calls) != 0 {
		t.Fatalf("got seam calls %v, want none for x=0", calls)
	}
}

// TestMPRecoveryZeroMpGainSkipsChangeMP: hpLost > 0 but mpGain floors to 0
// (MaxHp 10, x=10, y=75 -> hpLost 1, mpGain 0) — the HP cost still applies,
// the zero-delta MP emit is skipped.
func TestMPRecoveryZeroMpGainSkipsChangeMP(t *testing.T) {
	calls, err := invokeApply(t,
		func(_ logrus.FieldLogger, _ context.Context, _ uint32) (uint16, error) {
			return 10, nil
		},
		nil, nil, testEffect(t, 10, 75))
	if err != nil {
		t.Fatalf("Apply returned unexpected error: %v", err)
	}
	if len(calls) != 1 || calls[0].name != "changeHP" || calls[0].amount != -1 {
		t.Fatalf("got seam calls %v, want exactly changeHP with -1", calls)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./skill/handler/mprecovery/ -v`
Expected: FAIL to compile with `undefined: loadCaster` (and `changeHP`, `changeMP`, `Apply`)

- [ ] **Step 3: Write minimal implementation**

Create `services/atlas-channel/atlas.com/channel/skill/handler/mprecovery/mprecovery.go`:

```go
package mprecovery

import (
	"context"

	"atlas-channel/character"
	"atlas-channel/data/skill/effect"
	channelhandler "atlas-channel/skill/handler"
	"atlas-channel/socket/writer"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/sirupsen/logrus"
)

func init() {
	channelhandler.Register(skill2.BrawlerMPRecoveryId, Apply)
}

// loadCaster returns the caster's max HP from the character service.
var loadCaster = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) (uint16, error) {
	c, err := character.NewProcessor(l, ctx).GetById()(characterId)
	if err != nil {
		return 0, err
	}
	return c.MaxHp(), nil
}

// changeHP emits the HP-change command to atlas-character.
var changeHP = func(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId uint32, amount int16) error {
	return character.NewProcessor(l, ctx).ChangeHP(f, characterId, amount)
}

// changeMP emits the MP-change command to atlas-character.
var changeMP = func(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId uint32, amount int16) error {
	return character.NewProcessor(l, ctx).ChangeMP(f, characterId, amount)
}

// Apply is the MP Recovery handler installed in the per-skill registry.
//
// By the time this runs, UseSkill has already applied the cooldown (5101005
// has no hpCon/mpCon/duration in WZ, so those generic branches are no-ops).
// The effect is entirely server-authoritative: lose MaxHP/x HP, gain
// hpLost*y/100 MP (Cosmic parity). Deliberately no low-HP guard — a caster
// at or below MaxHP/x current HP dies through atlas-character's existing
// ChangeHP 0-floor/death path (owner decision, task-151 PRD FR-3).
func Apply(l logrus.FieldLogger) func(ctx context.Context) func(
	wp writer.Producer,
	f field.Model,
	characterId uint32,
	info packetmodel.SkillUsageInfo,
	e effect.Model,
) error {
	return func(ctx context.Context) func(
		wp writer.Producer,
		f field.Model,
		characterId uint32,
		info packetmodel.SkillUsageInfo,
		e effect.Model,
	) error {
		return func(
			wp writer.Producer,
			f field.Model,
			characterId uint32,
			info packetmodel.SkillUsageInfo,
			e effect.Model,
		) error {
			maxHp, err := loadCaster(l, ctx, characterId)
			if err != nil {
				l.WithError(err).Errorf("MP Recovery: failed to load caster [%d].", characterId)
				return err
			}

			hpLost, mpGain := Amounts(maxHp, e.X(), e.Y())
			if hpLost == 0 {
				l.Warnf("MP Recovery: no HP cost for caster [%d] (maxHp=[%d] x=[%d]); skipping.",
					characterId, maxHp, e.X())
				return nil
			}

			if err := changeHP(l, ctx, f, characterId, -hpLost); err != nil {
				l.WithError(err).Errorf("MP Recovery: ChangeHP failed for caster [%d]; skipping MP gain.", characterId)
				return err
			}
			if mpGain > 0 {
				if err := changeMP(l, ctx, f, characterId, mpGain); err != nil {
					l.WithError(err).Errorf("MP Recovery: ChangeMP failed for caster [%d].", characterId)
					return err
				}
			}

			l.Debugf("MP Recovery: caster=[%d] level=[%d] hpLost=[%d] mpGain=[%d].",
				characterId, info.SkillLevel(), hpLost, mpGain)
			return nil
		}
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./skill/handler/mprecovery/ -v`
Expected: all 8 top-level tests PASS (TestAmounts, TestMPRecoveryRegistered, TestMPRecoveryHappyPath, TestMPRecoveryCasterLoadError, TestMPRecoveryChangeHPError, TestMPRecoveryChangeMPError, TestMPRecoveryBadDataSkips, TestMPRecoveryZeroMpGainSkipsChangeMP)

- [ ] **Step 5: Commit**

From the worktree root:

```bash
git add services/atlas-channel/atlas.com/channel/skill/handler/mprecovery/mprecovery.go services/atlas-channel/atlas.com/channel/skill/handler/mprecovery/mprecovery_test.go
git commit -m "feat(channel): MP Recovery (5101005) per-skill handler"
```

---

### Task 4: Production registration wiring + full verification

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go` (+1 blank import)

**Interfaces:**
- Consumes: the `mprecovery` package's `init()` from Task 3. `main.go:58` already blank-imports `atlas-channel/skill/handler/registrations`, so this line makes the handler live in production.
- Produces: nothing new — final wiring + verification gates.

- [ ] **Step 1: Add the blank import**

Edit `services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go` to:

```go
// Package registrations exists solely to drive init() registration of
// per-skill handler subpackages. main.go blank-imports this package;
// each new handler subpackage is added below as a blank import.
package registrations

import (
	_ "atlas-channel/skill/handler/heal"       // Cleric Heal — task 045
	_ "atlas-channel/skill/handler/mprecovery" // Brawler MP Recovery — task-151
	_ "atlas-channel/skill/handler/mysticdoor" // Priest Mystic Door — task-093
)
```

(Alphabetical order; run `gofmt -w` on the file so import comment alignment matches gofmt output.)

- [ ] **Step 2: Run the full atlas-channel verification gates**

From `services/atlas-channel/atlas.com/channel/`:

```bash
go build ./...
go vet ./...
go test -race ./...
```

Expected: all three exit 0; `go test -race` shows `ok` for every package including `atlas-channel/skill/handler/mprecovery` and `atlas-channel/data/skill/effect`, no `FAIL`, no race reports.

- [ ] **Step 3: Run the redis key guard**

From the worktree root:

```bash
tools/redis-key-guard.sh
```

Expected: exit 0 / PASS (this change adds no Redis usage). Do NOT prefix with `GOWORK=off`.

- [ ] **Step 4: Commit**

From the worktree root:

```bash
git add services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go
git commit -m "feat(channel): register MP Recovery handler"
```

---

## Out of Scope (do not implement)

- Chakra (4211001) — separate task.
- Any packet/opcode/writer/handler change — the existing stat-changed flow renders the result.
- Any atlas-data or libs/atlas-constants change.
- Low-HP cast rejection or HP clamping in the handler.
- `docker buildx bake` — not required (no `go.mod` touched); harmless if run.

## After the plan is complete

Run code review (`superpowers:requesting-code-review`) before opening a PR — mandatory per CLAUDE.md.
