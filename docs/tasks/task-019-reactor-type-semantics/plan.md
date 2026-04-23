# Reactor Type Semantics & Timer-Driven Progression Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Hit-break reactors destroy correctly on terminal state (fixing reactor 2001 stuck-state bug), and type-101 timer-driven reactors auto-advance on their configured `timeOut`.

**Architecture:** Two coordinated changes. (1) atlas-data stops fabricating type-999 events and honors both `timeOut` / `timeout` casings; it also extracts per-state `timeoutNextState` from type-101 events. (2) atlas-reactors swaps the global "any event matches" persist check for a state-local rule driven by the matched event's type, and adds a process-local timer mechanism modeled on the existing item-reactor activation pattern.

**Tech Stack:** Go, JSON:API over HTTP between atlas-data and atlas-reactors, Redis registry, miniredis + testify for tests, `time.AfterFunc` + `sync.Mutex` for timers.

---

## File Structure

### Modified

| file | responsibility |
|---|---|
| `services/atlas-data/atlas.com/data/reactor/reader.go` | wz → RestModel: remove synthesis at line 124-130, honor `timeOut`/`timeout` casing at line 89, extract type-101 `state` into `TimeoutNextStateInfo`. |
| `services/atlas-data/atlas.com/data/reactor/rest.go` | Add `TimeoutNextStateInfo map[int8]int8`. |
| `services/atlas-data/atlas.com/data/reactor/reader_test.go` | Add fixtures + cases for empty terminal state, `timeOut` casing, and `TimeoutNextStateInfo` extraction. |
| `services/atlas-reactors/atlas.com/reactors/reactor/data/rest.go` | Mirror `TimeoutNextStateInfo` on the consumer side; thread it through `Extract`. |
| `services/atlas-reactors/atlas.com/reactors/reactor/data/model.go` | Add `Timeout(state)` and `TimeoutNextState(state)` accessors on the immutable model. |
| `services/atlas-reactors/atlas.com/reactors/reactor/data/model_json.go` | Add the new field to local Redis round-trip JSON. |
| `services/atlas-reactors/atlas.com/reactors/reactor/processor.go` | Replace `persistsAtFinalState(stateInfo)` with `persistsAtEndState(eventType int32)`; capture the matched event's type in `Hit` and pass it to both call sites; wire timer scheduling into `Create`, `Hit`, `Destroy`, `Teardown`. |
| `services/atlas-reactors/atlas.com/reactors/reactor/processor_test.go` | Add persist-rule cases. |
| `services/atlas-reactors/docs/domain.md` | Rewrite the "State Transitions" section (lines 94-147) — drop type-999 language, describe the new taxonomy and timer mechanism. |

### Created

| file | responsibility |
|---|---|
| `services/atlas-reactors/atlas.com/reactors/reactor/timer.go` | State-timeout mechanism: `scheduleStateTimeout`, `cancelStateTimeout`, `cancelAllStateTimeouts`. |
| `services/atlas-reactors/atlas.com/reactors/reactor/timer_test.go` | Covers fire-and-transition, cancel-on-hit, cancel-on-destroy, re-arm-after-fire, and cancelAll. |

---

## Task 1: atlas-data — Add `TimeoutNextStateInfo` to REST model

**Files:**
- Modify: `services/atlas-data/atlas.com/data/reactor/rest.go`

- [ ] **Step 1: Add the field to `RestModel`**

Edit `services/atlas-data/atlas.com/data/reactor/rest.go` — add `TimeoutNextStateInfo` alongside `TimeoutInfo`:

```go
type RestModel struct {
	Id                   uint32                           `json:"-"`
	Name                 string                           `json:"name"`
	TL                   point.RestModel                  `json:"tl"`
	BR                   point.RestModel                  `json:"br"`
	StateInfo            map[int8][]ReactorStateRestModel `json:"stateInfo"`
	TimeoutInfo          map[int8]int32                   `json:"timeoutInfo"`
	TimeoutNextStateInfo map[int8]int8                    `json:"timeoutNextStateInfo"`
}
```

- [ ] **Step 2: Build the package to confirm it still compiles**

Run: `cd services/atlas-data/atlas.com/data && go build ./reactor/...`
Expected: exits 0, no errors.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-data/atlas.com/data/reactor/rest.go
git commit -m "feat(atlas-data): add TimeoutNextStateInfo to reactor REST model"
```

---

## Task 2: atlas-data — Reader fix: stop synthesis, fix `timeOut` casing, extract `TimeoutNextStateInfo`

**Files:**
- Modify: `services/atlas-data/atlas.com/data/reactor/reader.go`
- Modify: `services/atlas-data/atlas.com/data/reactor/reader_test.go`

- [ ] **Step 1: Add failing test — empty terminal state must not appear in `StateInfo`**

The existing `infoFallbackTestXML` fixture (lines 315-338) already has two states; add a new fixture for a reactor whose terminal state has no `event` subtree. Append to `services/atlas-data/atlas.com/data/reactor/reader_test.go`:

```go
const terminalEmptyTestXML = `
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="0002001.img">
  <imgdir name="info">
    <string name="info" value="리엑터"/>
  </imgdir>
  <imgdir name="0">
    <imgdir name="event">
      <imgdir name="0">
        <int name="type" value="0"/>
        <int name="state" value="1"/>
      </imgdir>
    </imgdir>
  </imgdir>
  <imgdir name="1">
  </imgdir>
</imgdir>
`

var terminalEmptyNodeProvider = func(path string, id uint32) model.Provider[xml.Node] {
	return xml.FromByteArrayProvider([]byte(terminalEmptyTestXML))
}

func TestReaderTerminalEmptyStateOmitted(t *testing.T) {
	l, _ := test.NewNullLogger()
	rm, err := Read(l)("", 0, terminalEmptyNodeProvider)()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := rm.StateInfo[1]; ok {
		t.Fatalf("state 1 should be absent from StateInfo (no event subtree); got entries: %+v", rm.StateInfo[1])
	}
	if _, ok := rm.TimeoutInfo[1]; ok {
		t.Fatalf("state 1 should be absent from TimeoutInfo; got %+v", rm.TimeoutInfo[1])
	}
	if len(rm.StateInfo) != 1 {
		t.Fatalf("expected exactly 1 state in StateInfo (state 0); got %d", len(rm.StateInfo))
	}
}
```

- [ ] **Step 2: Run the test — confirm it fails against today's reader**

Run: `cd services/atlas-data/atlas.com/data && go test ./reactor/ -run TestReaderTerminalEmptyStateOmitted -v`
Expected: FAIL — today's `reader.go:124-130` synthesises a type-999 entry and populates `TimeoutInfo[1] = -1`.

- [ ] **Step 3: Add failing test — `timeOut` (upper-case O) must populate `TimeoutInfo`**

Append to `services/atlas-data/atlas.com/data/reactor/reader_test.go`:

```go
const timeOutCasingTestXML = `
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="9101000.img">
  <imgdir name="info">
    <string name="info" value="MoonFlower"/>
  </imgdir>
  <imgdir name="0">
    <imgdir name="event">
      <imgdir name="0">
        <int name="type" value="101"/>
        <int name="state" value="1"/>
      </imgdir>
      <int name="timeOut" value="5000"/>
    </imgdir>
  </imgdir>
  <imgdir name="1">
    <imgdir name="event">
      <imgdir name="0">
        <int name="type" value="0"/>
        <int name="state" value="2"/>
      </imgdir>
    </imgdir>
  </imgdir>
</imgdir>
`

var timeOutCasingNodeProvider = func(path string, id uint32) model.Provider[xml.Node] {
	return xml.FromByteArrayProvider([]byte(timeOutCasingTestXML))
}

func TestReaderHonorsTimeOutCasing(t *testing.T) {
	l, _ := test.NewNullLogger()
	rm, err := Read(l)("", 0, timeOutCasingNodeProvider)()
	if err != nil {
		t.Fatal(err)
	}
	got, ok := rm.TimeoutInfo[0]
	if !ok {
		t.Fatal("TimeoutInfo[0] missing")
	}
	if got != 5000 {
		t.Fatalf("TimeoutInfo[0] = %d, want 5000", got)
	}
}

func TestReaderExtractsTimeoutNextState(t *testing.T) {
	l, _ := test.NewNullLogger()
	rm, err := Read(l)("", 0, timeOutCasingNodeProvider)()
	if err != nil {
		t.Fatal(err)
	}
	got, ok := rm.TimeoutNextStateInfo[0]
	if !ok {
		t.Fatal("TimeoutNextStateInfo[0] missing")
	}
	if got != 1 {
		t.Fatalf("TimeoutNextStateInfo[0] = %d, want 1 (from type-101 event's state field)", got)
	}
	if _, present := rm.TimeoutNextStateInfo[1]; present {
		t.Fatalf("TimeoutNextStateInfo[1] should be absent (state 1 has no type-101 event); got %+v", rm.TimeoutNextStateInfo[1])
	}
}
```

- [ ] **Step 4: Run tests — confirm they fail**

Run: `cd services/atlas-data/atlas.com/data && go test ./reactor/ -run 'TestReaderHonorsTimeOutCasing|TestReaderExtractsTimeoutNextState' -v`
Expected: FAIL — `timeOut` is not read (lowercase only), and `TimeoutNextStateInfo` does not yet exist as a populated field.

- [ ] **Step 5: Implement the reader fix**

Edit `services/atlas-data/atlas.com/data/reactor/reader.go`. Three changes:

a) Initialize `TimeoutNextStateInfo` in the main `RestModel` literal (around line 81):

```go
m := RestModel{
	Id:                   reactorId,
	Name:                 name,
	StateInfo:            map[int8][]ReactorStateRestModel{},
	TimeoutInfo:          map[int8]int32{},
	TimeoutNextStateInfo: map[int8]int8{},
}
```

Also initialize it in the info-less short-circuit literal at the top (around line 47-56 — where the synthesised state-0 fallback builds a RestModel). That literal stays unchanged otherwise; just add the new field as an empty map:

```go
m := RestModel{
	Id: reactorId,
	StateInfo: map[int8][]ReactorStateRestModel{
		0: {{Type: 999, ReactorItem: nil, ActiveSkills: nil, NextState: 0}},
	},
	TimeoutInfo: map[int8]int32{
		0: -1,
	},
	TimeoutNextStateInfo: map[int8]int8{},
}
```

(This info-less branch synthesises a type-999 placeholder so that a reactor with a completely missing `info` block still has a harmless state 0. That is a separate code path from the per-state synthesis we are removing — leave it alone.)

b) Fix the `timeout` read at line 89 to prefer `timeOut`:

```go
timeout := ed.GetIntegerWithDefault("timeOut", -1)
if timeout == -1 {
	timeout = ed.GetIntegerWithDefault("timeout", -1)
}
```

c) Remove the synthesis in the `else` branch (lines 124-130) and extract `timeoutNextState` from the first type-101 event. Replace the entire `ed != nil { ... } else { ... }` block (currently lines 87-130) with:

```go
ed, _ := rid.ChildByName("event")
if ed != nil {
	timeout := ed.GetIntegerWithDefault("timeOut", -1)
	if timeout == -1 {
		timeout = ed.GetIntegerWithDefault("timeout", -1)
	}

	var timeoutNextState int8 = -1
	timeoutNextStateSet := false

	for _, md := range ed.ChildNodes {
		t := md.GetIntegerWithDefault("type", 0)
		var ri *ReactorItemRestModel
		if t == 100 {
			itemId := uint32(md.GetIntegerWithDefault("0", 0))
			quantity := uint16(md.GetIntegerWithDefault("1", 0))
			ri = &ReactorItemRestModel{ItemId: itemId, Quantity: quantity}
			if !areaSet || loadArea {
				x, y := md.GetPoint("tl", 0, 0)
				m.TL = point.RestModel{
					X: int16(x),
					Y: int16(y),
				}
				x, y = md.GetPoint("rb", 0, 0)
				m.BR = point.RestModel{
					X: int16(x),
					Y: int16(y),
				}
				areaSet = true
			}
		}
		skillIds := make([]uint32, 0)
		activeSkillId, _ := md.ChildByName("activeSkillID")
		if activeSkillId != nil {
			for _, s := range activeSkillId.ChildNodes {
				skillIds = append(skillIds, uint32(md.GetIntegerWithDefault(s.Name, 0)))
			}
		}
		ns := int8(md.GetIntegerWithDefault("state", 0))
		if t == 101 && !timeoutNextStateSet {
			timeoutNextState = ns
			timeoutNextStateSet = true
		}
		sdl = append(sdl, ReactorStateRestModel{Type: t, ReactorItem: ri, ActiveSkills: skillIds, NextState: ns})
	}
	m.StateInfo[i] = sdl
	m.TimeoutInfo[i] = timeout
	if timeoutNextStateSet {
		m.TimeoutNextStateInfo[i] = timeoutNextState
	}
}
```

Note: the `else { ... }` branch that previously synthesised `{Type:999, NextState:i+1}` is deleted entirely. A state with no `event` subtree contributes no entry to `StateInfo`, `TimeoutInfo`, or `TimeoutNextStateInfo`.

- [ ] **Step 6: Run the three new tests and expect them to pass**

Run: `cd services/atlas-data/atlas.com/data && go test ./reactor/ -run 'TestReaderTerminalEmptyStateOmitted|TestReaderHonorsTimeOutCasing|TestReaderExtractsTimeoutNextState' -v`
Expected: all three PASS.

- [ ] **Step 7: Verify existing tests still pass**

Run: `cd services/atlas-data/atlas.com/data && go test ./reactor/...`
Expected: the full reactor package test suite passes. Note: `TestReader` (using `testXML`) currently expects `len(rm.StateInfo) == 6`. That fixture defines states 0-5, and states 0-4 each have a `hit` subtree but no `event` subtree; state 5 has a full `event` subtree. After the synthesis removal, `StateInfo` will only contain state 5. This test will FAIL and must be updated:

```go
func TestReader(t *testing.T) {
	l, _ := test.NewNullLogger()

	rm, err := Read(l)("", 0, fixedNodeProvider)()
	if err != nil {
		t.Fatal(err)
	}
	if rm.Id != 1002000 {
		t.Fatal("id != 1002000")
	}
	if rm.Name != "거대병아리" {
		t.Fatalf("name != 거대병아리, got %s", rm.Name)
	}
	if len(rm.StateInfo) != 1 {
		t.Fatalf("len(rm.StateInfo) != 1 (only state 5 has an event subtree), got %d", len(rm.StateInfo))
	}
	if _, ok := rm.StateInfo[5]; !ok {
		t.Fatal("StateInfo[5] missing")
	}
	if rm.TimeoutInfo[5] != 1000 {
		t.Fatalf("TimeoutInfo[5] = %d, want 1000", rm.TimeoutInfo[5])
	}
	tns, ok := rm.TimeoutNextStateInfo[5]
	if !ok {
		t.Fatal("TimeoutNextStateInfo[5] missing")
	}
	if tns != 0 {
		t.Fatalf("TimeoutNextStateInfo[5] = %d, want 0 (type-101 event's state field)", tns)
	}
}
```

Update `TestLinkedReader` the same way — it uses the same underlying `testXML` via the link:

```go
func TestLinkedReader(t *testing.T) {
	l, _ := test.NewNullLogger()

	rm, err := Read(l)("", 0, linkedNodeProvider)()
	if err != nil {
		t.Fatal(err)
	}
	if rm.Id != 1020008 {
		t.Fatal("id != 1020008")
	}
	if rm.Name != "거대병아리" {
		t.Fatalf("name != 거대병아리, got %s", rm.Name)
	}
	if len(rm.StateInfo) != 1 {
		t.Fatalf("len(rm.StateInfo) != 1, got %d", len(rm.StateInfo))
	}
}
```

`TestReaderInfoFallback` uses `infoFallbackTestXML` which has two states, each with `event` subtrees — its `len(rm.StateInfo) != 2` assertion is already correct. Leave it alone.

Re-run `cd services/atlas-data/atlas.com/data && go test ./reactor/...` — expect all tests PASS.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-data/atlas.com/data/reactor/reader.go services/atlas-data/atlas.com/data/reactor/reader_test.go
git commit -m "fix(atlas-data): stop fabricating type-999 events; honor timeOut casing; extract timeoutNextState"
```

---

## Task 3: atlas-reactors — Extend `data` package with `TimeoutNextStateInfo` + `Timeout`/`TimeoutNextState` accessors

**Files:**
- Modify: `services/atlas-reactors/atlas.com/reactors/reactor/data/rest.go`
- Modify: `services/atlas-reactors/atlas.com/reactors/reactor/data/model.go`
- Modify: `services/atlas-reactors/atlas.com/reactors/reactor/data/model_json.go`

- [ ] **Step 1: Add `TimeoutNextStateInfo` to `data.RestModel` and thread through `Extract`**

Edit `services/atlas-reactors/atlas.com/reactors/reactor/data/rest.go`. Update the struct:

```go
type RestModel struct {
	Id                   uint32                     `json:"-"`
	Name                 string                     `json:"name"`
	TL                   point.RestModel            `json:"tl"`
	BR                   point.RestModel            `json:"br"`
	StateInfo            map[int8][]state.RestModel `json:"stateInfo"`
	TimeoutInfo          map[int8]int32             `json:"timeoutInfo"`
	TimeoutNextStateInfo map[int8]int8              `json:"timeoutNextStateInfo"`
}
```

Update the `Extract` function's return literal:

```go
	return Model{
		name:                 rm.Name,
		tl:                   tl,
		br:                   br,
		stateInfo:            si,
		timeoutInfo:          rm.TimeoutInfo,
		timeoutNextStateInfo: rm.TimeoutNextStateInfo,
	}, nil
```

- [ ] **Step 2: Add the field + accessors to the immutable `data.Model`**

Edit `services/atlas-reactors/atlas.com/reactors/reactor/data/model.go`:

```go
package data

import (
	"atlas-reactors/reactor/data/point"
	"atlas-reactors/reactor/data/state"
)

type Model struct {
	name                 string
	tl                   point.Model
	br                   point.Model
	stateInfo            map[int8][]state.Model
	timeoutInfo          map[int8]int32
	timeoutNextStateInfo map[int8]int8
}

func (m Model) Name() string {
	return m.name
}

func (m Model) StateInfo() map[int8][]state.Model {
	return m.stateInfo
}

func (m Model) TL() point.Model {
	return m.tl
}

func (m Model) BR() point.Model {
	return m.br
}

// Timeout returns the per-state timeout in milliseconds, or -1 if this state
// has no timeout configured.
func (m Model) Timeout(state int8) int32 {
	if m.timeoutInfo == nil {
		return -1
	}
	v, ok := m.timeoutInfo[state]
	if !ok {
		return -1
	}
	return v
}

// TimeoutNextState returns the state to transition to when this state's timer
// fires. The bool is false if no timer transition is configured for this state
// (i.e. no type-101 event was present in the .wz).
func (m Model) TimeoutNextState(state int8) (int8, bool) {
	if m.timeoutNextStateInfo == nil {
		return 0, false
	}
	v, ok := m.timeoutNextStateInfo[state]
	return v, ok
}
```

- [ ] **Step 3: Thread the new field through the local Redis round-trip**

Edit `services/atlas-reactors/atlas.com/reactors/reactor/data/model_json.go`:

```go
package data

import (
	"atlas-reactors/reactor/data/point"
	"atlas-reactors/reactor/data/state"
	"encoding/json"
)

type modelJSON struct {
	Name                 string                 `json:"name"`
	Tl                   point.Model            `json:"tl"`
	Br                   point.Model            `json:"br"`
	StateInfo            map[int8][]state.Model `json:"stateInfo"`
	TimeoutInfo          map[int8]int32         `json:"timeoutInfo"`
	TimeoutNextStateInfo map[int8]int8          `json:"timeoutNextStateInfo"`
}

func (m Model) MarshalJSON() ([]byte, error) {
	return json.Marshal(modelJSON{
		Name:                 m.name,
		Tl:                   m.tl,
		Br:                   m.br,
		StateInfo:            m.stateInfo,
		TimeoutInfo:          m.timeoutInfo,
		TimeoutNextStateInfo: m.timeoutNextStateInfo,
	})
}

func (m *Model) UnmarshalJSON(data []byte) error {
	var j modelJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	m.name = j.Name
	m.tl = j.Tl
	m.br = j.Br
	m.stateInfo = j.StateInfo
	m.timeoutInfo = j.TimeoutInfo
	m.timeoutNextStateInfo = j.TimeoutNextStateInfo
	return nil
}
```

- [ ] **Step 4: Build atlas-reactors**

Run: `cd services/atlas-reactors/atlas.com/reactors && go build ./...`
Expected: exits 0, no errors.

- [ ] **Step 5: Run the existing reactor package tests**

Run: `cd services/atlas-reactors/atlas.com/reactors && go test ./reactor/...`
Expected: existing tests still PASS — this task is purely additive.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-reactors/atlas.com/reactors/reactor/data/
git commit -m "feat(atlas-reactors): add Timeout and TimeoutNextState accessors to reactor data model"
```

---

## Task 4: atlas-reactors — Replace `persistsAtFinalState` with `persistsAtEndState(eventType)` and rewire `Hit`

**Files:**
- Modify: `services/atlas-reactors/atlas.com/reactors/reactor/processor.go`
- Modify: `services/atlas-reactors/atlas.com/reactors/reactor/processor_test.go`

- [ ] **Step 1: Add failing test — hit to terminal on a type-0 reactor destroys (no persistence)**

Append to `services/atlas-reactors/atlas.com/reactors/reactor/processor_test.go` the following helper and test. Place helpers above the tests that use them; tests at the end of the file.

First, helper — a DSL for building a `data.Model` from a `data.RestModel` literal:

```go
// newTestData returns a data.Model populated from the given state/timeout maps.
// Uses data.Extract so the model mirrors what production code consumes from
// atlas-data, including state.Model conversion.
func newTestData(t *testing.T, stateInfo map[int8][]state.RestModel, timeoutInfo map[int8]int32, timeoutNextStateInfo map[int8]int8) data.Model {
	t.Helper()
	if timeoutInfo == nil {
		timeoutInfo = map[int8]int32{}
	}
	if timeoutNextStateInfo == nil {
		timeoutNextStateInfo = map[int8]int8{}
	}
	m, err := data.Extract(data.RestModel{
		Name:                 "test",
		StateInfo:            stateInfo,
		TimeoutInfo:          timeoutInfo,
		TimeoutNextStateInfo: timeoutNextStateInfo,
	})
	if err != nil {
		t.Fatalf("data.Extract failed: %v", err)
	}
	return m
}
```

Add the required imports to the existing import block in `processor_test.go`:

```go
"atlas-reactors/reactor/data/state"
```

Now add the first persist-rule test. This test exercises `Hit` without invoking Kafka — since `producer.ProviderImpl` is used inside and may no-op when no broker is configured in the test environment, we accept that emission may warn-log but the state transition and destroy/persist decision must still be observable via the registry. If your test environment blocks on Kafka publish, set `KAFKA_BROKERS=""` before running tests — unit tests in this package already rely on that being unset. No extra work needed here.

```go
// TestHit_BreakableReactorDestroysOnTerminal verifies the fix for reactor 2001:
// a reactor with only type-0 events and no synthesized 999s must destroy and
// record cooldown on the terminal transition.
func TestHit_BreakableReactorDestroysOnTerminal(t *testing.T) {
	setupTestRegistry(t)
	l := setupTestLogger()
	ten := setupTestTenant()
	ctx := setupTestContext(ten)

	// Shape mirrors what atlas-data now returns for reactor 2001:
	// state 0 -> 1 via type-0 event; state 1 has no events (terminal).
	d := newTestData(t,
		map[int8][]state.RestModel{
			0: {{Type: 0, NextState: 1, ActiveSkills: []uint32{}}},
		},
		nil, nil,
	)

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(1000000)).Build()
	builder := NewModelBuilder(ten, f, 2001, "reactor-2001").
		SetState(0).SetPosition(231, 253).SetDelay(5000).SetDirection(0).SetData(d)
	created, err := GetRegistry().Create(ten, builder)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := Hit(l)(ctx)(created.Id(), 0, 0); err != nil {
		t.Fatalf("Hit returned error: %v", err)
	}

	// After the hit: reactor must be gone from the registry.
	if _, err := GetRegistry().Get(ten, created.Id()); err == nil {
		t.Fatal("reactor should have been destroyed on terminal-state transition, but still exists")
	}

	// Cooldown must be recorded at its (classification,x,y) for the map.
	mk := NewMapKey(f)
	if !GetRegistry().IsOnCooldown(ten, mk, 2001, 231, 253) {
		t.Fatal("cooldown should have been recorded after destroy")
	}
}
```

- [ ] **Step 2: Add two more failing persist-rule tests — item-reactor and skill-reactor must persist at terminal**

Append:

```go
// TestHit_ItemReactorPersistsAtTerminal verifies that a reactor whose matched
// hit event is type 100 is kept alive at the terminal state (moonflower-style).
func TestHit_ItemReactorPersistsAtTerminal(t *testing.T) {
	setupTestRegistry(t)
	l := setupTestLogger()
	ten := setupTestTenant()
	ctx := setupTestContext(ten)

	// State 0 -> 1 via a type-100 event. State 1 has no events (terminal).
	d := newTestData(t,
		map[int8][]state.RestModel{
			0: {{Type: 100, NextState: 1, ActiveSkills: []uint32{}}},
		},
		nil, nil,
	)

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(1000000)).Build()
	builder := NewModelBuilder(ten, f, 9108000, "moonflower").
		SetState(0).SetPosition(100, 100).SetDelay(0).SetData(d)
	created, _ := GetRegistry().Create(ten, builder)

	if err := Hit(l)(ctx)(created.Id(), 0, 0); err != nil {
		t.Fatalf("Hit returned error: %v", err)
	}

	// Reactor should still exist at state 1.
	got, err := GetRegistry().Get(ten, created.Id())
	if err != nil {
		t.Fatalf("reactor should have been kept alive at terminal (type-100 event); got error: %v", err)
	}
	if got.State() != 1 {
		t.Fatalf("state = %d, want 1", got.State())
	}
}

// TestHit_SkillReactorPersistsAtTerminal verifies types 5/6/7 (GPQ skill-gated)
// also persist at terminal.
func TestHit_SkillReactorPersistsAtTerminal(t *testing.T) {
	setupTestRegistry(t)
	l := setupTestLogger()
	ten := setupTestTenant()
	ctx := setupTestContext(ten)

	// State 0 -> 1 via a type-5 event (any skill matches since ActiveSkills
	// is empty in our test; in production types 5/6/7 carry activeSkillID —
	// we're only exercising the persist rule here).
	d := newTestData(t,
		map[int8][]state.RestModel{
			0: {{Type: 5, NextState: 1, ActiveSkills: []uint32{}}},
		},
		nil, nil,
	)

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(1000000)).Build()
	builder := NewModelBuilder(ten, f, 6109013, "gpq-skill-reactor").
		SetState(0).SetPosition(100, 100).SetDelay(0).SetData(d)
	created, _ := GetRegistry().Create(ten, builder)

	if err := Hit(l)(ctx)(created.Id(), 0, 0); err != nil {
		t.Fatalf("Hit returned error: %v", err)
	}

	got, err := GetRegistry().Get(ten, created.Id())
	if err != nil {
		t.Fatalf("skill reactor should persist at terminal; got error: %v", err)
	}
	if got.State() != 1 {
		t.Fatalf("state = %d, want 1", got.State())
	}
}
```

Run: `cd services/atlas-reactors/atlas.com/reactors && go test ./reactor/ -run 'TestHit_BreakableReactorDestroysOnTerminal|TestHit_ItemReactorPersistsAtTerminal|TestHit_SkillReactorPersistsAtTerminal' -v`
Expected under the OLD rule:
- `TestHit_BreakableReactorDestroysOnTerminal` — PASSes already (no type-100/999 anywhere in state info → destroys).
- `TestHit_ItemReactorPersistsAtTerminal` — PASSes already (type-100 is present → old rule persists, matches new rule here).
- `TestHit_SkillReactorPersistsAtTerminal` — FAILs (type-5 is not type-100/999 → old rule destroys). This is our driver test for the rule change.

- [ ] **Step 3: Replace `persistsAtFinalState` with `persistsAtEndState(eventType)` and rewire `Hit`**

Edit `services/atlas-reactors/atlas.com/reactors/reactor/processor.go`. Two changes.

a) Replace the function at lines 259-270 with:

```go
// persistsAtEndState returns true if a reactor that has just transitioned via
// an event of the given type should remain alive rather than be destroyed.
// Taxonomy (from the wz reactor survey):
//
//	100       item-drop reactors (moonflowers, etc.)
//	101       timer-driven cyclic reactors (Balrog altars, PQ cycles)
//	5, 6, 7   GPQ skill-gated reactors
//
// All other types (0, 1, 2) are breakable hit reactors and destroy on end.
func persistsAtEndState(eventType int32) bool {
	switch eventType {
	case 100, 101, 5, 6, 7:
		return true
	default:
		return false
	}
}
```

b) Rewrite the matched-event loop and both persist call sites in `Hit`. The current loop (around lines 172-178) becomes:

```go
var nextState int8 = -1
var matchedEventType int32 = 0
for _, event := range stateEvents {
	if len(event.ActiveSkills()) == 0 || containsSkill(event.ActiveSkills(), skillId) {
		nextState = event.NextState()
		matchedEventType = event.Type()
		break
	}
}
```

Then update the two `persistsAtFinalState(stateInfo)` call sites to `persistsAtEndState(matchedEventType)`:

```go
_, hasNextState := stateInfo[nextState]
if !hasNextState {
	if persistsAtEndState(matchedEventType) {
		updated, err := GetRegistry().Update(t, reactorId, func(b *ModelBuilder) {
			b.SetState(nextState)
		})
		if err != nil {
			l.WithError(err).Errorf("Unable to update reactor [%d] state.", reactorId)
			return err
		}
		l.Debugf("Reactor [%d] hit. State changed from [%d] to final state [%d]. Keeping reactor alive (event type %d).", reactorId, r.State(), nextState, matchedEventType)
		Trigger(l)(ctx)(updated, characterId)
		return producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(hitStatusEventProvider(updated, false))
	}
	l.Debugf("Reactor [%d] next state [%d] not in state info. Triggering and destroying.", reactorId, nextState)
	return TriggerAndDestroy(l)(ctx)(r, characterId)
}

updated, err := GetRegistry().Update(t, reactorId, func(b *ModelBuilder) {
	b.SetState(nextState)
})
if err != nil {
	l.WithError(err).Errorf("Unable to update reactor [%d] state.", reactorId)
	return err
}

if isTerminalState(stateInfo, nextState) {
	if persistsAtEndState(matchedEventType) {
		l.Debugf("Reactor [%d] hit. State changed from [%d] to terminal state [%d]. Keeping reactor alive (event type %d).", reactorId, r.State(), nextState, matchedEventType)
		Trigger(l)(ctx)(updated, characterId)
		return producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(hitStatusEventProvider(updated, false))
	}
	l.Debugf("Reactor [%d] hit. State changed from [%d] to terminal state [%d]. Triggering and destroying.", reactorId, r.State(), nextState)
	return TriggerAndDestroy(l)(ctx)(updated, characterId)
}
```

- [ ] **Step 4: Run the three new persist-rule tests — expect all PASS**

Run: `cd services/atlas-reactors/atlas.com/reactors && go test ./reactor/ -run 'TestHit_BreakableReactorDestroysOnTerminal|TestHit_ItemReactorPersistsAtTerminal|TestHit_SkillReactorPersistsAtTerminal' -v`
Expected: all three PASS.

- [ ] **Step 5: Run the full reactor package test suite**

Run: `cd services/atlas-reactors/atlas.com/reactors && go test ./reactor/...`
Expected: all PASS. If any existing test relied on the old global scan (e.g. an assertion that a reactor with a type-999 anywhere persists), update it to match the new state-local rule.

- [ ] **Step 6: Check for any other callers of `persistsAtFinalState`**

Run: `grep -rn "persistsAtFinalState" services/atlas-reactors/`
Expected: no matches — the function is only defined and called in `processor.go`, and both sites were rewritten. If the grep returns a stray reference, delete it.

- [ ] **Step 7: Remove now-unused `state` import if any became dead**

Run: `cd services/atlas-reactors/atlas.com/reactors && go build ./reactor/...`
Expected: builds cleanly. If `go build` complains about an unused import (the `state` import in `processor.go` was only used by the old function signature — `isTerminalState` still uses it, so it should remain needed), leave it; if it warns unused, remove it.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-reactors/atlas.com/reactors/reactor/processor.go services/atlas-reactors/atlas.com/reactors/reactor/processor_test.go
git commit -m "fix(atlas-reactors): make persist-at-end-state rule state-local based on matched event type"
```

---

## Task 5: atlas-reactors — Create the state-timer mechanism (no wiring yet)

**Files:**
- Create: `services/atlas-reactors/atlas.com/reactors/reactor/timer.go`
- Create: `services/atlas-reactors/atlas.com/reactors/reactor/timer_test.go`

- [ ] **Step 1: Write the failing test for `scheduleStateTimeout` firing and transitioning**

Create `services/atlas-reactors/atlas.com/reactors/reactor/timer_test.go`:

```go
package reactor

import (
	"atlas-reactors/reactor/data"
	"atlas-reactors/reactor/data/state"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// timerTestData builds a reactor data.Model where state 0 auto-advances to
// state 1 via a timer, and state 1 has no events.
func timerTestData(t *testing.T, timeoutMs int32) data.Model {
	t.Helper()
	m, err := data.Extract(data.RestModel{
		Name: "timer-test",
		StateInfo: map[int8][]state.RestModel{
			0: {{Type: 101, NextState: 1, ActiveSkills: []uint32{}}},
		},
		TimeoutInfo:          map[int8]int32{0: timeoutMs},
		TimeoutNextStateInfo: map[int8]int8{0: 1},
	})
	if err != nil {
		t.Fatalf("data.Extract: %v", err)
	}
	return m
}

// TestScheduleStateTimeout_FiresAndTransitions verifies the core loop: a state
// with Timeout+TimeoutNextState set arms a timer; on fire, the reactor moves
// to the configured next state.
func TestScheduleStateTimeout_FiresAndTransitions(t *testing.T) {
	setupTestRegistry(t)
	l := setupTestLogger()
	ten := setupTestTenant()
	ctx := setupTestContext(ten)

	d := timerTestData(t, 50) // 50ms

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(1000000)).Build()
	builder := NewModelBuilder(ten, f, 9101000, "moon-bunny").
		SetState(0).SetPosition(100, 100).SetDelay(0).SetData(d)
	created, err := GetRegistry().Create(ten, builder)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	scheduleStateTimeout(l, ctx, created)

	// Wait for the timer to fire and the transition to complete.
	time.Sleep(150 * time.Millisecond)

	got, err := GetRegistry().Get(ten, created.Id())
	if err != nil {
		t.Fatalf("reactor gone after timer fire; timer should transition not destroy for type-101 cyclic: %v", err)
	}
	if got.State() != 1 {
		t.Fatalf("state = %d, want 1 (timer-driven transition)", got.State())
	}

	cancelAllStateTimeouts() // cleanup
}

// TestCancelStateTimeout_StopsPendingFire verifies that cancel prevents the
// transition from happening.
func TestCancelStateTimeout_StopsPendingFire(t *testing.T) {
	setupTestRegistry(t)
	l := setupTestLogger()
	ten := setupTestTenant()
	ctx := setupTestContext(ten)

	d := timerTestData(t, 200) // long enough to cancel before fire

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(1000000)).Build()
	builder := NewModelBuilder(ten, f, 9101000, "moon-bunny").
		SetState(0).SetPosition(100, 100).SetDelay(0).SetData(d)
	created, _ := GetRegistry().Create(ten, builder)

	scheduleStateTimeout(l, ctx, created)
	time.Sleep(50 * time.Millisecond)
	cancelStateTimeout(created.Id())

	time.Sleep(250 * time.Millisecond)

	got, _ := GetRegistry().Get(ten, created.Id())
	if got.State() != 0 {
		t.Fatalf("state = %d, want 0 (timer was cancelled before firing)", got.State())
	}
}

// TestCancelAllStateTimeouts_DoesNotPanicWhenEmpty verifies teardown safety.
func TestCancelAllStateTimeouts_DoesNotPanicWhenEmpty(t *testing.T) {
	cancelAllStateTimeouts() // should be a no-op with no panic
}
```

- [ ] **Step 2: Run the tests — confirm they fail to compile (no timer.go yet)**

Run: `cd services/atlas-reactors/atlas.com/reactors && go test ./reactor/ -run 'TestScheduleStateTimeout|TestCancelStateTimeout|TestCancelAllStateTimeouts' -v`
Expected: FAIL — `scheduleStateTimeout`, `cancelStateTimeout`, `cancelAllStateTimeouts` are undefined.

- [ ] **Step 3: Create `timer.go` with the mechanism**

Create `services/atlas-reactors/atlas.com/reactors/reactor/timer.go`:

```go
package reactor

import (
	"atlas-reactors/kafka/producer"
	"context"
	"sync"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

var (
	pendingStateTimeouts     = make(map[uint32]*time.Timer)
	pendingStateTimeoutsLock sync.Mutex
)

// scheduleStateTimeout arms a process-local timer for a reactor's current
// state if both Timeout(state) > 0 and TimeoutNextState(state) are set.
// A previously-armed timer for this reactor is cancelled before a new one is
// armed (idempotent).
//
// On fire the callback re-fetches the reactor from the registry, verifies the
// state has not changed (a hit or another transition would have cancelled the
// timer, but this guards against races), transitions to the configured next
// state, emits a TRIGGER, and re-arms if the new state also has a timer.
// If the new state is terminal and the matched event's type (101) is still
// cyclic, the reactor persists and we stop here.
func scheduleStateTimeout(l logrus.FieldLogger, ctx context.Context, r Model) {
	d := r.Data()
	timeoutMs := d.Timeout(r.State())
	nextState, hasNext := d.TimeoutNextState(r.State())
	if timeoutMs <= 0 || !hasNext {
		return
	}

	reactorId := r.Id()
	t := tenant.MustFromContext(ctx)

	pendingStateTimeoutsLock.Lock()
	defer pendingStateTimeoutsLock.Unlock()

	if existing, ok := pendingStateTimeouts[reactorId]; ok {
		existing.Stop()
		delete(pendingStateTimeouts, reactorId)
	}

	delay := time.Duration(timeoutMs) * time.Millisecond
	l.Debugf("Arming state-timeout for reactor [%d] at state [%d]: %v -> state [%d].", reactorId, r.State(), delay, nextState)

	pendingStateTimeouts[reactorId] = time.AfterFunc(delay, func() {
		pendingStateTimeoutsLock.Lock()
		delete(pendingStateTimeouts, reactorId)
		pendingStateTimeoutsLock.Unlock()

		current, err := GetRegistry().Get(t, reactorId)
		if err != nil {
			l.Debugf("State-timeout fired for reactor [%d], but it no longer exists. Skipping.", reactorId)
			return
		}
		if current.State() != r.State() {
			l.Debugf("State-timeout fired for reactor [%d], but state changed [%d] -> [%d]. Skipping stale fire.", reactorId, r.State(), current.State())
			return
		}

		updated, err := GetRegistry().Update(t, reactorId, func(b *ModelBuilder) {
			b.SetState(nextState)
		})
		if err != nil {
			l.WithError(err).Errorf("Unable to update reactor [%d] state on timer fire.", reactorId)
			return
		}

		l.Debugf("Reactor [%d] timer-advanced from state [%d] to [%d].", reactorId, r.State(), nextState)
		Trigger(l)(ctx)(updated, 0)

		if err := producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(hitStatusEventProvider(updated, false)); err != nil {
			l.WithError(err).Warnf("Failed to emit HIT status event for reactor [%d] after timer fire.", reactorId)
		}

		// Re-arm for the new state if it also has a timer configured.
		scheduleStateTimeout(l, ctx, updated)
	})
}

// cancelStateTimeout stops any pending state timer for a reactor.
func cancelStateTimeout(reactorId uint32) {
	pendingStateTimeoutsLock.Lock()
	defer pendingStateTimeoutsLock.Unlock()

	if timer, ok := pendingStateTimeouts[reactorId]; ok {
		timer.Stop()
		delete(pendingStateTimeouts, reactorId)
	}
}

// cancelAllStateTimeouts stops every pending state timer. Called during
// service teardown alongside CancelAllPendingActivations.
func cancelAllStateTimeouts() {
	pendingStateTimeoutsLock.Lock()
	defer pendingStateTimeoutsLock.Unlock()

	for id, timer := range pendingStateTimeouts {
		timer.Stop()
		delete(pendingStateTimeouts, id)
	}
}
```

- [ ] **Step 4: Run the timer tests — expect PASS**

Run: `cd services/atlas-reactors/atlas.com/reactors && go test ./reactor/ -run 'TestScheduleStateTimeout|TestCancelStateTimeout|TestCancelAllStateTimeouts' -v`
Expected: all three PASS.

- [ ] **Step 5: Full reactor package test run to confirm no regressions**

Run: `cd services/atlas-reactors/atlas.com/reactors && go test ./reactor/...`
Expected: all PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-reactors/atlas.com/reactors/reactor/timer.go services/atlas-reactors/atlas.com/reactors/reactor/timer_test.go
git commit -m "feat(atlas-reactors): add state-timeout mechanism for type-101 timer-driven reactors"
```

---

## Task 6: atlas-reactors — Wire the timer into Create / Hit / Destroy / Teardown

**Files:**
- Modify: `services/atlas-reactors/atlas.com/reactors/reactor/processor.go`

- [ ] **Step 1: Arm a timer after `Create` succeeds**

In `services/atlas-reactors/atlas.com/reactors/reactor/processor.go`, inside the `Create` function, after the successful create line `l.Debugf("Created reactor [%d] of [%d]."...)` and before the `return producer.ProviderImpl(l)(...)` line, add:

```go
scheduleStateTimeout(l, ctx, r)
```

So the tail of `Create` becomes:

```go
GetRegistry().ClearCooldown(t, mk, r.Classification(), r.X(), r.Y())
l.Debugf("Created reactor [%d] of [%d].", r.Id(), r.Classification())
scheduleStateTimeout(l, ctx, r)
return producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(createdStatusEventProvider(r))
```

- [ ] **Step 2: Cancel pending timer on entry to `Hit` and re-arm on transitions**

In `Hit`, right after fetching the reactor via `GetById`, add a cancellation:

```go
r, err := GetById(l)(ctx)(reactorId)
if err != nil {
	l.WithError(err).Errorf("Unable to get reactor [%d] for hit.", reactorId)
	return err
}

// A hit interrupts any pending state timer for this reactor.
cancelStateTimeout(reactorId)
```

At each of the three points where `Hit` returns successfully after a state transition that leaves the reactor alive (not `TriggerAndDestroy`), re-arm the timer for the new state. Specifically, after each `GetRegistry().Update(...)` success block where the reactor survives, insert `scheduleStateTimeout(l, ctx, updated)` before the final `return producer.ProviderImpl(...)`. There are three such branches in `Hit`:

(a) The "next-state-not-in-stateInfo but persists" branch:

```go
if !hasNextState {
	if persistsAtEndState(matchedEventType) {
		updated, err := GetRegistry().Update(t, reactorId, func(b *ModelBuilder) {
			b.SetState(nextState)
		})
		if err != nil {
			l.WithError(err).Errorf("Unable to update reactor [%d] state.", reactorId)
			return err
		}
		l.Debugf("Reactor [%d] hit. State changed from [%d] to final state [%d]. Keeping reactor alive (event type %d).", reactorId, r.State(), nextState, matchedEventType)
		Trigger(l)(ctx)(updated, characterId)
		scheduleStateTimeout(l, ctx, updated)
		return producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(hitStatusEventProvider(updated, false))
	}
	...
}
```

(b) The "transitioned to terminal state but persists" branch:

```go
if isTerminalState(stateInfo, nextState) {
	if persistsAtEndState(matchedEventType) {
		l.Debugf("Reactor [%d] hit. State changed from [%d] to terminal state [%d]. Keeping reactor alive (event type %d).", reactorId, r.State(), nextState, matchedEventType)
		Trigger(l)(ctx)(updated, characterId)
		scheduleStateTimeout(l, ctx, updated)
		return producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(hitStatusEventProvider(updated, false))
	}
	...
}
```

(c) The normal "transitioned to a non-terminal state" branch (the last `return` in the function):

```go
l.Debugf("Reactor [%d] hit. State changed from [%d] to [%d].", reactorId, r.State(), nextState)
scheduleStateTimeout(l, ctx, updated)
return producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(hitStatusEventProvider(updated, false))
```

- [ ] **Step 3: Cancel timer in `Destroy` and `DestroyInField`**

In `Destroy`, add `cancelStateTimeout(m.Id())` immediately after the existing `CancelPendingActivation(m.Id())` call:

```go
func Destroy(l logrus.FieldLogger) func(ctx context.Context) model.Operator[Model] {
	return func(ctx context.Context) model.Operator[Model] {
		return func(m Model) error {
			CancelPendingActivation(m.Id())
			cancelStateTimeout(m.Id())
			t := tenant.MustFromContext(ctx)
			...
```

In `DestroyInField`, inside the `for _, r := range reactors` loop, add `cancelStateTimeout(r.Id())` alongside the existing `CancelPendingActivation(r.Id())`:

```go
for _, r := range reactors {
	CancelPendingActivation(r.Id())
	cancelStateTimeout(r.Id())
	GetRegistry().Remove(t, r.Id())
	...
}
```

- [ ] **Step 4: Cancel all timers in `Teardown`**

Update `Teardown` to call `cancelAllStateTimeouts` alongside `CancelAllPendingActivations`:

```go
func Teardown(l logrus.FieldLogger) func() {
	return func() {
		CancelAllPendingActivations()
		cancelAllStateTimeouts()

		ctx, span := otel.GetTracerProvider().Tracer("atlas-reactors").Start(context.Background(), "teardown")
		defer span.End()
		...
```

- [ ] **Step 5: Add a failing test — hit cancels a pending state timer**

Append to `services/atlas-reactors/atlas.com/reactors/reactor/timer_test.go`:

```go
// TestHit_CancelsPendingStateTimer verifies that hitting a reactor with a
// pending timer cancels that timer (a new one may be armed for the new state,
// but the original fire MUST NOT happen).
func TestHit_CancelsPendingStateTimer(t *testing.T) {
	setupTestRegistry(t)
	l := setupTestLogger()
	ten := setupTestTenant()
	ctx := setupTestContext(ten)

	// State 0: type-0 hit event -> state 2, AND a type-101 timer -> state 1.
	// If the timer were to fire we'd see state 1; if the hit lands first and
	// cancels the timer, we see state 2 without any further transition.
	m, err := data.Extract(data.RestModel{
		Name: "hit-cancels-timer",
		StateInfo: map[int8][]state.RestModel{
			0: {{Type: 0, NextState: 2, ActiveSkills: []uint32{}}},
			2: {{Type: 0, NextState: 3, ActiveSkills: []uint32{}}},
		},
		TimeoutInfo:          map[int8]int32{0: 100},
		TimeoutNextStateInfo: map[int8]int8{0: 1},
	})
	if err != nil {
		t.Fatalf("data.Extract: %v", err)
	}

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(1000000)).Build()
	builder := NewModelBuilder(ten, f, 9999, "test").
		SetState(0).SetPosition(100, 100).SetDelay(0).SetData(m)
	created, _ := GetRegistry().Create(ten, builder)

	// Create has armed a 100ms timer. Hit immediately — should cancel it.
	if err := Hit(l)(ctx)(created.Id(), 0, 0); err != nil {
		t.Fatalf("Hit error: %v", err)
	}

	// Wait well past the original timer's fire time.
	time.Sleep(200 * time.Millisecond)

	got, err := GetRegistry().Get(ten, created.Id())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	// State must be 2 (hit result). If the timer had fired we'd see 1 or 3.
	if got.State() != 2 {
		t.Fatalf("state = %d, want 2 (hit landed, timer should have been cancelled)", got.State())
	}

	cancelAllStateTimeouts()
}

// TestDestroy_CancelsPendingStateTimer verifies Destroy cancels the timer.
func TestDestroy_CancelsPendingStateTimer(t *testing.T) {
	setupTestRegistry(t)
	l := setupTestLogger()
	ten := setupTestTenant()
	ctx := setupTestContext(ten)

	d := timerTestData(t, 100)

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(1000000)).Build()
	builder := NewModelBuilder(ten, f, 9101000, "moon-bunny").
		SetState(0).SetPosition(100, 100).SetDelay(0).SetData(d)
	created, _ := GetRegistry().Create(ten, builder)

	// Create armed the timer. Destroy should cancel.
	if err := Destroy(l)(ctx)(created); err != nil {
		t.Fatalf("Destroy: %v", err)
	}

	// No panic / no attempt to transition a deleted reactor.
	time.Sleep(200 * time.Millisecond)

	if _, err := GetRegistry().Get(ten, created.Id()); err == nil {
		t.Fatal("reactor should be gone after Destroy")
	}
}
```

- [ ] **Step 6: Run the new tests — expect PASS**

Run: `cd services/atlas-reactors/atlas.com/reactors && go test ./reactor/ -run 'TestHit_CancelsPendingStateTimer|TestDestroy_CancelsPendingStateTimer' -v`
Expected: both PASS.

- [ ] **Step 7: Full reactor package test run**

Run: `cd services/atlas-reactors/atlas.com/reactors && go test ./reactor/...`
Expected: all PASS.

- [ ] **Step 8: Full atlas-reactors build**

Run: `cd services/atlas-reactors/atlas.com/reactors && go build ./...`
Expected: exits 0, no errors.

- [ ] **Step 9: Commit**

```bash
git add services/atlas-reactors/atlas.com/reactors/reactor/processor.go services/atlas-reactors/atlas.com/reactors/reactor/timer_test.go
git commit -m "feat(atlas-reactors): wire state-timeout into Create/Hit/Destroy/Teardown"
```

---

## Task 7: atlas-reactors — Add re-arm-after-fire test

**Files:**
- Modify: `services/atlas-reactors/atlas.com/reactors/reactor/timer_test.go`

- [ ] **Step 1: Add failing test — timer re-arms for a new state that also has a timeout**

Append to `services/atlas-reactors/atlas.com/reactors/reactor/timer_test.go`:

```go
// TestScheduleStateTimeout_ReArmsForNewState verifies that when a timer fires
// and the new state also has a timeout+timeoutNextState configured, a fresh
// timer is armed and the reactor continues to advance.
func TestScheduleStateTimeout_ReArmsForNewState(t *testing.T) {
	setupTestRegistry(t)
	l := setupTestLogger()
	ten := setupTestTenant()
	ctx := setupTestContext(ten)

	// State 0 -> 1 after 50ms; state 1 -> 2 after 50ms; state 2 is terminal.
	m, err := data.Extract(data.RestModel{
		Name: "chained-timer",
		StateInfo: map[int8][]state.RestModel{
			0: {{Type: 101, NextState: 1, ActiveSkills: []uint32{}}},
			1: {{Type: 101, NextState: 2, ActiveSkills: []uint32{}}},
		},
		TimeoutInfo:          map[int8]int32{0: 50, 1: 50},
		TimeoutNextStateInfo: map[int8]int8{0: 1, 1: 2},
	})
	if err != nil {
		t.Fatalf("data.Extract: %v", err)
	}

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(1000000)).Build()
	builder := NewModelBuilder(ten, f, 9101000, "chained").
		SetState(0).SetPosition(100, 100).SetDelay(0).SetData(m)
	created, _ := GetRegistry().Create(ten, builder)

	scheduleStateTimeout(l, ctx, created)

	// 50ms -> state 1; another 50ms -> state 2. Total 150ms window covers both.
	time.Sleep(200 * time.Millisecond)

	got, err := GetRegistry().Get(ten, created.Id())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	// type-101 is a persist type; the final terminal transition keeps the
	// reactor alive at state 2.
	if got.State() != 2 {
		t.Fatalf("state = %d, want 2 (timer re-armed and fired again)", got.State())
	}

	cancelAllStateTimeouts()
}
```

- [ ] **Step 2: Run the test — expect PASS**

Run: `cd services/atlas-reactors/atlas.com/reactors && go test ./reactor/ -run TestScheduleStateTimeout_ReArmsForNewState -v`
Expected: PASS (the re-arm call in `timer.go`'s fire callback already does this).

- [ ] **Step 3: Full reactor package test run**

Run: `cd services/atlas-reactors/atlas.com/reactors && go test ./reactor/...`
Expected: all PASS.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-reactors/atlas.com/reactors/reactor/timer_test.go
git commit -m "test(atlas-reactors): cover timer re-arm across chained timed states"
```

---

## Task 8: Update domain documentation

**Files:**
- Modify: `services/atlas-reactors/docs/domain.md`

- [ ] **Step 1: Rewrite the "State Transitions" section**

Open `services/atlas-reactors/docs/domain.md`. Locate the "State Transitions" section (around line 96) and the surrounding persist-behavior copy (around lines 94, 109, 147 — any reference to "type 999" or "type 100 or 999"). Replace the `## State Transitions` section wholesale with the content below. Lines around 94 and 147 that contain "type 999" wording should also be updated to remove the type-999 references and reflect the new taxonomy.

Exact replacement for the state-transition explanation (adapt surrounding prose to match):

```markdown
## State Transitions

Reactors advance through states via two mechanisms:

1. **Hit** — a player attack triggers an event on the current state. The matched event's `nextState` becomes the reactor's new state. A hit can deliver a skill id; if a state's event lists `activeSkills`, only a matching skill id (or an event with empty `activeSkills`) is eligible.
2. **State timeout** — if a state has a `timeout` set and a paired `timeoutNextState`, an in-process `time.AfterFunc` timer advances the reactor automatically. Timers are cancelled on hit, destroy, or teardown.

### Event-type taxonomy

Each event carries a `type` field that determines what happens on a terminal transition:

| type    | meaning                               | end-state behavior   |
|---------|---------------------------------------|----------------------|
| 0       | hit by any attack (default breakable) | destroy + cooldown   |
| 1, 2    | directional hit                       | destroy + cooldown   |
| 5, 6, 7 | GPQ skill-gated                       | **persist**          |
| 100     | item-drop trigger                     | **persist**          |
| 101     | timer-driven cyclic                   | **persist (cyclic)** |

The persist-vs-destroy decision is **state-local**: it is based on the type of the event that led to the terminal transition, not on whether a persist-type event appears anywhere in the reactor's state machine.

### Timer-driven reactors

State-timeout timers are armed whenever a reactor enters a state with `timeout > 0` and a configured `timeoutNextState`. Arming happens in:

- `Create` (for the reactor's initial state).
- `Hit` (for the new state after a transition that keeps the reactor alive).
- The timer callback itself (re-arming for the new state after a fire).

Cancellation happens in:

- `Hit` (on function entry — a hit interrupts the timer).
- `Destroy` / `DestroyInField` (any code path removing the reactor).
- `Teardown` (`cancelAllStateTimeouts` alongside `CancelAllPendingActivations`).

When a timer fires, the callback re-reads the reactor from the registry, bails if it no longer exists or its state has changed since arming, transitions to the configured `nextState`, emits a TRIGGER, re-arms for the new state if applicable, and persists (type-101 is always a persist type).

### Notes & caveats

- Timers are process-local. A replica that owns the timer at arming time owns the fire. Process crashes lose pending timers on that process; other replicas' timers proceed independently. This is not load-balanced, but it is simple and correct.
- Reactors whose `.wz` defines states with no `event` subtree are represented with no entry for that state in `StateInfo` (atlas-data no longer synthesises placeholders). A hit on such a state flows through the "no state events" branch and destroys the reactor unless the matched event type is a persist type — which, in this branch, it cannot be, because there was no matched event.
- Moon Bunny (`9101000`) currently has neither events nor a `timeOut` in its `.wz` snippet and so will remain at state 0 until teardown. A proper fix requires richer per-state data or an explicit script hook and is tracked separately.
```

Remove any remaining references to type-999 in the file (bullet at ~line 94 and note at ~line 147). The `Create`, `Destroy`, `Hit` operational bullets elsewhere in the file are unaffected.

- [ ] **Step 2: Sanity-check the doc renders and reads correctly**

Run: `grep -n 'type 999\|999' services/atlas-reactors/docs/domain.md`
Expected: zero matches (or only matches that clearly do not refer to reactor event type 999 — verify by reading each).

- [ ] **Step 3: Commit**

```bash
git add services/atlas-reactors/docs/domain.md
git commit -m "docs(atlas-reactors): rewrite state-transitions section with new event-type taxonomy and timer mechanism"
```

---

## Task 9: Cross-service verification

**Files:**
- None (validation only).

- [ ] **Step 1: Build atlas-data**

Run: `cd services/atlas-data/atlas.com/data && go build ./...`
Expected: exits 0.

- [ ] **Step 2: Test atlas-data**

Run: `cd services/atlas-data/atlas.com/data && go test ./...`
Expected: all PASS.

- [ ] **Step 3: Build atlas-reactors**

Run: `cd services/atlas-reactors/atlas.com/reactors && go build ./...`
Expected: exits 0.

- [ ] **Step 4: Test atlas-reactors**

Run: `cd services/atlas-reactors/atlas.com/reactors && go test ./...`
Expected: all PASS.

- [ ] **Step 5: Grep for leftover type-999 references across the two services**

Run: `grep -rn '\b999\b' services/atlas-data/atlas.com/data/reactor/ services/atlas-reactors/atlas.com/reactors/`
Expected: only the deliberate survivor — the info-less short-circuit in `services/atlas-data/atlas.com/data/reactor/reader.go` (which represents a reactor with no `info` block at all, not a per-state synthesis). No other 999 references should remain.

- [ ] **Step 6: Manual playtest note**

This step is for the reviewer / QA, not the implementing agent. Record in the PR description: "In a test map populated with 5 instances of reactor 2001, verify that hitting each reactor drives state 0→1→2→3→4→destroy, and that atlas-maps's 10-second spawn tick re-spawns them after the cooldown. Also verify a type-101 reactor with `timeOut` auto-advances on its own." No code change required here.

- [ ] **Step 7: Commit (if any stray changes)**

If steps 1-5 required fixups, commit them individually with descriptive messages. Otherwise, no commit.

---

## Self-Review Notes (already applied)

- **Spec coverage:** All PRD goals 1-4 and success criteria 1-7 map to tasks above. Goal 1 / criterion 1 (reactor 2001 destroys correctly) → Task 4. Goal 2 / criteria 2 (item/skill/timer reactors persist) → Task 4 + Task 5. Goal 3 / criterion 3 (timer auto-advance) → Tasks 5-6. Goal 4 / criterion 4 (no synthetic 999) → Task 2 + Task 9 step 5. Criterion 5 (`timeOut` casing) → Task 2. Criterion 6 (tests pass) → Tasks 2, 4, 5, 6, 7, 9. Criterion 7 (cross-service cycle) → Task 9 step 6 (manual playtest). Moon Bunny non-regression is flagged in domain.md (Task 8).
- **Placeholder scan:** No TBDs or "similar to" references. All code blocks are complete and runnable.
- **Type consistency:** `int8` for states, `int32` for event types and timeouts, `uint32` for reactor IDs. `Timeout(state int8) int32`, `TimeoutNextState(state int8) (int8, bool)`, `persistsAtEndState(eventType int32) bool` — all consistent across tasks 3-7. Map types `map[int8]int32` (timeouts) and `map[int8]int8` (next states) consistent across atlas-data RestModel and atlas-reactors RestModel / Model / modelJSON.
