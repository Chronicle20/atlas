# GM Command `@mob spawn` Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a GM-only chat command `@mob spawn <templateId> [count]` to atlas-messages that spawns 1–20 instances of a monster template at the issuing GM's current position.

**Architecture:** atlas-messages parses/authorizes the command, validates the template id and resolves the foothold via existing atlas-data REST endpoints, then emits `count` identical `SPAWN_FIELD` Kafka commands on `COMMAND_TOPIC_MONSTER`. atlas-monsters consumes each command and creates one monster through its existing `monster.Processor.Create` path. A prerequisite fix plumbs the GM's X/Y (currently dropped) through the atlas-messages `character.Model`.

**Tech Stack:** Go, JSON:API over `atlas-rest`, Kafka via `atlas-kafka`, immutable models + Builder pattern, `field.Model` from `atlas-constants`.

---

## File Structure

**atlas-messages** (`services/atlas-messages/atlas.com/messages/`):
- `character/model.go` — add `x`/`y` to `Model`, wire builder setters + `Build`/`Clone`, fix `X()`/`Y()` getters (MODIFY)
- `character/rest.go` — `Extract` copies `rm.X`/`rm.Y` (MODIFY)
- `character/rest_test.go` — regression tests for X/Y plumb-through (MODIFY)
- `data/monster/{model,rest,requests,processor}.go` + `data/monster/mock/processor.go` — template-validation client (CREATE)
- `data/monster/rest_test.go` — JSON:API contract test (CREATE)
- `data/foothold/{model,rest,requests,processor}.go` + `data/foothold/mock/processor.go` — foothold-below client (CREATE)
- `data/foothold/rest_test.go` — JSON:API contract test (CREATE)
- `kafka/message/monster/kafka.go` — `CommandTypeSpawnField`, `SpawnFieldBody`, `SpawnFieldCommandProvider` (MODIFY)
- `kafka/message/monster/kafka_test.go` — provider emits `count` messages with correct body (CREATE)
- `command/monster/commands.go` — `MobSpawnCommandProducer` + pure helpers `parseSpawnArgs`/`normalizeCount` (MODIFY)
- `command/monster/commands_test.go` — regex/GM-gate/parse/clamp tests (CREATE)
- `command/help/commands.go` — add `@mob spawn` help line (MODIFY)
- `main.go` — register `monster.MobSpawnCommandProducer` (MODIFY)

**atlas-monsters** (`services/atlas-monsters/atlas.com/monsters/`):
- `kafka/consumer/monster/kafka.go` — `CommandTypeSpawnField`, `spawnFieldCommandBody` (MODIFY)
- `kafka/consumer/monster/consumer.go` — `handleSpawnFieldCommand` + registration (MODIFY)
- `kafka/consumer/monster/kafka_test.go` — body decode test (MODIFY)

---

## Task 1: atlas-messages — plumb GM X/Y through `character.Model`

**Context:** `character.Model.X()`/`Y()` are hardcoded `return 0` stubs and the `Model` struct has no `x`/`y` fields. `character.Extract` drops `rm.X`/`rm.Y` even though the REST resource returns them. The `ModelBuilder` already declares dead `x`/`y` fields (no setters, not copied in `Build`). This task wires them end-to-end so the spawn command can read the GM's real position. `stance` is intentionally NOT plumbed (`Create` ignores stance).

**Files:**
- Modify: `services/atlas-messages/atlas.com/messages/character/model.go`
- Modify: `services/atlas-messages/atlas.com/messages/character/rest.go:99-130`
- Test: `services/atlas-messages/atlas.com/messages/character/rest_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `services/atlas-messages/atlas.com/messages/character/rest_test.go`:

```go
// TestExtract_Position verifies Extract maps X/Y onto the model (regression:
// these were previously dropped, breaking @mob spawn position).
func TestExtract_Position(t *testing.T) {
	rm := RestModel{Id: 1, X: 250, Y: -130}
	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if m.X() != 250 {
		t.Errorf("X() = %d, want 250", m.X())
	}
	if m.Y() != -130 {
		t.Errorf("Y() = %d, want -130", m.Y())
	}
}

// TestModel_PositionRoundTripsThroughSetSkills guards the SetSkills path, which
// rebuilds the model via Clone(m).SetSkills(...).Build(); position must survive.
func TestModel_PositionRoundTripsThroughSetSkills(t *testing.T) {
	m := NewModelBuilder().SetId(1).SetX(99).SetY(-7).Build()
	m2 := m.SetSkills(nil)
	if m2.X() != 99 || m2.Y() != -7 {
		t.Errorf("position lost after SetSkills: got (%d, %d), want (99, -7)", m2.X(), m2.Y())
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-messages/atlas.com/messages && go test ./character/ -run 'TestExtract_Position|TestModel_PositionRoundTripsThroughSetSkills' -v`
Expected: FAIL — `SetX`/`SetY` undefined and/or `X()` returns 0.

- [ ] **Step 3: Add `x`/`y` fields to the `Model` struct**

In `model.go`, add two fields to the `Model` struct (after `meso uint32`, before `skills`):

```go
	meso               uint32
	x                  int16
	y                  int16
	skills             []skill.Model
```

- [ ] **Step 4: Fix the `X()` / `Y()` getters**

In `model.go`, replace the stub getters:

```go
func (m Model) X() int16 {
	return m.x
}

func (m Model) Y() int16 {
	return m.y
}
```

(Leave `Stance()` returning `0` — stance is not plumbed.)

- [ ] **Step 5: Copy `x`/`y` in `Clone`**

In `model.go`, add to the `Clone` builder literal (after `meso: m.meso,`):

```go
		meso:               m.meso,
		x:                  m.x,
		y:                  m.y,
		skills:             m.skills,
```

- [ ] **Step 6: Add `SetX` / `SetY` builder setters**

In `model.go`, after the `SetMeso` setter, add:

```go
func (b *ModelBuilder) SetX(v int16) *ModelBuilder              { b.x = v; return b }
func (b *ModelBuilder) SetY(v int16) *ModelBuilder              { b.y = v; return b }
```

(The builder already declares `x int16` and `y int16` fields.)

- [ ] **Step 7: Copy `x`/`y` in `Build()`**

In `model.go`, add to the `Build()` return literal (after `meso: b.meso,`):

```go
		meso:               b.meso,
		x:                  b.x,
		y:                  b.y,
		skills:             b.skills,
```

- [ ] **Step 8: Map `X`/`Y` in `Extract`**

In `rest.go`, add to the `Extract` `Model{...}` literal (after `meso: rm.Meso,`):

```go
		meso:               rm.Meso,
		x:                  rm.X,
		y:                  rm.Y,
	}, nil
```

- [ ] **Step 9: Run tests to verify they pass**

Run: `cd services/atlas-messages/atlas.com/messages && go test ./character/ -v`
Expected: PASS (all character tests, including the two new ones).

- [ ] **Step 10: Commit**

```bash
git add services/atlas-messages/atlas.com/messages/character/model.go services/atlas-messages/atlas.com/messages/character/rest.go services/atlas-messages/atlas.com/messages/character/rest_test.go
git commit -m "fix(atlas-messages): plumb character X/Y through model for @mob spawn"
```

---

## Task 2: atlas-messages — `data/monster` template-validation client

**Context:** The command pre-validates the template id with `GET /data/monsters/{id}`. This mirrors the existing `data/skill` client. Only `Id` and `Name` are needed (Name enriches the success message). A `mock` subpackage mirrors `data/position/mock` in atlas-pets for future test injection and matches the project's mockable-Processor convention.

**Files:**
- Create: `services/atlas-messages/atlas.com/messages/data/monster/model.go`
- Create: `services/atlas-messages/atlas.com/messages/data/monster/rest.go`
- Create: `services/atlas-messages/atlas.com/messages/data/monster/requests.go`
- Create: `services/atlas-messages/atlas.com/messages/data/monster/processor.go`
- Create: `services/atlas-messages/atlas.com/messages/data/monster/mock/processor.go`
- Test: `services/atlas-messages/atlas.com/messages/data/monster/rest_test.go`

- [ ] **Step 1: Write the failing test**

Create `services/atlas-messages/atlas.com/messages/data/monster/rest_test.go`:

```go
package monster

import "testing"

func TestRestModel_GetName(t *testing.T) {
	if (RestModel{}).GetName() != "monsters" {
		t.Errorf("GetName() = %q, want %q", (RestModel{}).GetName(), "monsters")
	}
}

func TestExtract_IdAndName(t *testing.T) {
	rm := RestModel{Id: 100100, Name: "Snail"}
	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}
	if m.Id() != 100100 {
		t.Errorf("Id() = %d, want 100100", m.Id())
	}
	if m.Name() != "Snail" {
		t.Errorf("Name() = %q, want %q", m.Name(), "Snail")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-messages/atlas.com/messages && go test ./data/monster/ -v`
Expected: FAIL — package does not compile (`RestModel`, `Extract`, `Model` undefined).

- [ ] **Step 3: Create `model.go`**

```go
package monster

type Model struct {
	id   uint32
	name string
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) Name() string {
	return m.name
}
```

- [ ] **Step 4: Create `rest.go`**

```go
package monster

import "strconv"

type RestModel struct {
	Id   uint32 `json:"-"`
	Name string `json:"name"`
}

func (r RestModel) GetName() string {
	return "monsters"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(idStr string) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:   rm.Id,
		name: rm.Name,
	}, nil
}
```

- [ ] **Step 5: Create `requests.go`**

```go
package monster

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	monstersResource = "data/monsters/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestById(monsterId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+monstersResource, monsterId))
}
```

- [ ] **Step 6: Create `processor.go`**

```go
package monster

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetById(monsterId uint32) (Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

func (p *ProcessorImpl) GetById(monsterId uint32) (Model, error) {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(monsterId), Extract)()
}
```

- [ ] **Step 7: Create `mock/processor.go`**

```go
package mock

import "atlas-messages/data/monster"

type Processor struct {
	GetByIdFn func(monsterId uint32) (monster.Model, error)
}

func (m *Processor) GetById(monsterId uint32) (monster.Model, error) {
	if m.GetByIdFn != nil {
		return m.GetByIdFn(monsterId)
	}
	return monster.Model{}, nil
}
```

- [ ] **Step 8: Run test to verify it passes**

Run: `cd services/atlas-messages/atlas.com/messages && go test ./data/monster/... -v`
Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add services/atlas-messages/atlas.com/messages/data/monster/
git commit -m "feat(atlas-messages): add data/monster template-validation client"
```

---

## Task 3: atlas-messages — `data/foothold` foothold-below client

**Context:** The command resolves the foothold beneath the GM via `POST /data/maps/{mapId}/footholds/below` with body `{x, y}`. This mirrors atlas-pets `data/position`, but we only need the foothold `Id`, so `Extract` deliberately ignores the `first`/`second` points (avoiding the nil-pointer dereference present in the pets version's `Extract`). A `mock` subpackage is provided. The endpoint returns 500 (not 404) when no foothold is found; any error maps to the `Fh = 0` fallback in the command executor.

**Files:**
- Create: `services/atlas-messages/atlas.com/messages/data/foothold/model.go`
- Create: `services/atlas-messages/atlas.com/messages/data/foothold/rest.go`
- Create: `services/atlas-messages/atlas.com/messages/data/foothold/requests.go`
- Create: `services/atlas-messages/atlas.com/messages/data/foothold/processor.go`
- Create: `services/atlas-messages/atlas.com/messages/data/foothold/mock/processor.go`
- Test: `services/atlas-messages/atlas.com/messages/data/foothold/rest_test.go`

- [ ] **Step 1: Write the failing test**

Create `services/atlas-messages/atlas.com/messages/data/foothold/rest_test.go`:

```go
package foothold

import "testing"

func TestRestModel_GetName(t *testing.T) {
	if (RestModel{}).GetName() != "footholds" {
		t.Errorf("GetName() = %q, want %q", (RestModel{}).GetName(), "footholds")
	}
}

func TestPositionRestModel_GetName(t *testing.T) {
	if (PositionRestModel{}).GetName() != "positions" {
		t.Errorf("GetName() = %q, want %q", (PositionRestModel{}).GetName(), "positions")
	}
}

func TestExtract_Id(t *testing.T) {
	m, err := Extract(RestModel{Id: 42})
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}
	if m.Id() != 42 {
		t.Errorf("Id() = %d, want 42", m.Id())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-messages/atlas.com/messages && go test ./data/foothold/ -v`
Expected: FAIL — package does not compile.

- [ ] **Step 3: Create `model.go`**

```go
package foothold

type Model struct {
	id uint32
}

func (m Model) Id() uint32 {
	return m.id
}
```

- [ ] **Step 4: Create `rest.go`**

```go
package foothold

import "strconv"

type RestModel struct {
	Id uint32 `json:"-"`
}

func (r RestModel) GetName() string {
	return "footholds"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(idStr string) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{id: rm.Id}, nil
}

type PositionRestModel struct {
	Id uint32 `json:"-"`
	X  int16  `json:"x"`
	Y  int16  `json:"y"`
}

func (r PositionRestModel) GetName() string {
	return "positions"
}

func (r PositionRestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *PositionRestModel) SetID(idStr string) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}
```

- [ ] **Step 5: Create `requests.go`**

```go
package foothold

import (
	"fmt"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	footholdBelowResource = "data/maps/%d/footholds/below"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func getInMap(mapId _map.Id, x int16, y int16) requests.Request[RestModel] {
	i := PositionRestModel{X: x, Y: y}
	return requests.PostRequest[RestModel](fmt.Sprintf(getBaseRequest()+footholdBelowResource, mapId), i)
}
```

- [ ] **Step 6: Create `processor.go`**

```go
package foothold

import (
	"context"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetBelow(mapId _map.Id, x int16, y int16) (Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

func (p *ProcessorImpl) GetBelow(mapId _map.Id, x int16, y int16) (Model, error) {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(getInMap(mapId, x, y), Extract)()
}
```

- [ ] **Step 7: Create `mock/processor.go`**

```go
package mock

import (
	"atlas-messages/data/foothold"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

type Processor struct {
	GetBelowFn func(mapId _map.Id, x int16, y int16) (foothold.Model, error)
}

func (m *Processor) GetBelow(mapId _map.Id, x int16, y int16) (foothold.Model, error) {
	if m.GetBelowFn != nil {
		return m.GetBelowFn(mapId, x, y)
	}
	return foothold.Model{}, nil
}
```

- [ ] **Step 8: Run test to verify it passes**

Run: `cd services/atlas-messages/atlas.com/messages && go test ./data/foothold/... -v`
Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add services/atlas-messages/atlas.com/messages/data/foothold/
git commit -m "feat(atlas-messages): add data/foothold below-resolution client"
```

---

## Task 4: atlas-messages — `SpawnFieldCommandProvider` Kafka provider

**Context:** Emit `count` identical `SPAWN_FIELD` messages in a single provider (one `Emit`), all carrying the same `FieldCommand[SpawnFieldBody]` value and partition key (`mapId`), matching the existing field-command providers. The envelope is the existing `FieldCommand[E]` (no `MonsterId` slot) — `monsterId` is a template id and goes in the body.

**Files:**
- Modify: `services/atlas-messages/atlas.com/messages/kafka/message/monster/kafka.go`
- Test: `services/atlas-messages/atlas.com/messages/kafka/message/monster/kafka_test.go`

- [ ] **Step 1: Write the failing test**

Create `services/atlas-messages/atlas.com/messages/kafka/message/monster/kafka_test.go`:

```go
package monster

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestSpawnFieldCommandProvider_EmitsCountMessages(t *testing.T) {
	inst := uuid.New()
	msgs, err := SpawnFieldCommandProvider(1, 2, 100000000, inst, 100100, 250, -130, 7, 0, 3)()
	if err != nil {
		t.Fatalf("provider error: %v", err)
	}
	if len(msgs) != 3 {
		t.Fatalf("len(msgs) = %d, want 3", len(msgs))
	}

	var cmd FieldCommand[SpawnFieldBody]
	if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cmd.Type != CommandTypeSpawnField {
		t.Errorf("Type = %q, want %q", cmd.Type, CommandTypeSpawnField)
	}
	if cmd.MapId != 100000000 || cmd.Instance != inst {
		t.Errorf("envelope mismatch: mapId=%d instance=%s", cmd.MapId, cmd.Instance)
	}
	if cmd.Body.MonsterId != 100100 || cmd.Body.X != 250 || cmd.Body.Y != -130 || cmd.Body.Fh != 7 || cmd.Body.Team != 0 {
		t.Errorf("body mismatch: %+v", cmd.Body)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-messages/atlas.com/messages && go test ./kafka/message/monster/ -v`
Expected: FAIL — `SpawnFieldCommandProvider`, `SpawnFieldBody`, `CommandTypeSpawnField` undefined.

- [ ] **Step 3: Add the command-type constant**

In `kafka.go`, add to the `const` block:

```go
	CommandTypeDestroyField      = "DESTROY_FIELD"
	CommandTypeSpawnField        = "SPAWN_FIELD"
```

- [ ] **Step 4: Add the body struct and provider**

In `kafka.go`, append after `CancelStatusFieldCommandProvider`:

```go
type SpawnFieldBody struct {
	MonsterId uint32 `json:"monsterId"`
	X         int16  `json:"x"`
	Y         int16  `json:"y"`
	Fh        int16  `json:"fh"`
	Team      int8   `json:"team"`
}

func SpawnFieldCommandProvider(worldId world.Id, channelId channel.Id, mapId _map.Id, instance uuid.UUID, monsterId uint32, x int16, y int16, fh int16, team int8, count int) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(mapId))
	value := FieldCommand[SpawnFieldBody]{
		WorldId:   worldId,
		ChannelId: channelId,
		MapId:     mapId,
		Instance:  instance,
		Type:      CommandTypeSpawnField,
		Body: SpawnFieldBody{
			MonsterId: monsterId,
			X:         x,
			Y:         y,
			Fh:        fh,
			Team:      team,
		},
	}
	messages := make([]producer.RawMessage, count)
	for i := range messages {
		messages[i] = producer.RawMessage{Key: key, Value: value}
	}
	return producer.MessageProvider(model.FixedProvider(messages))
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd services/atlas-messages/atlas.com/messages && go test ./kafka/message/monster/ -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-messages/atlas.com/messages/kafka/message/monster/kafka.go services/atlas-messages/atlas.com/messages/kafka/message/monster/kafka_test.go
git commit -m "feat(atlas-messages): add SPAWN_FIELD kafka command provider"
```

---

## Task 5: atlas-messages — `MobSpawnCommandProducer`

**Context:** The producer mirrors `MobStatusCommandProducer`: regex match → GM gate → executor. Parsing/clamping live in two pure, unit-testable helpers (`parseSpawnArgs`, `normalizeCount`); the executor performs the REST validation, foothold resolution, Kafka emit, and pink-text feedback. Per the established command-test idiom in this repo, tests cover the regex, GM gate, and the pure helpers (the executor's REST/Kafka dispatch is not unit-tested, consistent with the other `@mob` commands). The executor uses the **incoming `f`** for world/channel/map/instance (preserving instance) and `c.X()`/`c.Y()` for position. Import the data client as `monsterdata` to avoid colliding with the `monster` (kafka message) package.

**Files:**
- Modify: `services/atlas-messages/atlas.com/messages/command/monster/commands.go`
- Test: `services/atlas-messages/atlas.com/messages/command/monster/commands_test.go`

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-messages/atlas.com/messages/command/monster/commands_test.go`:

```go
package monster

import (
	"context"
	"testing"

	"atlas-messages/character"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/sirupsen/logrus/hooks/test"
)

func testCharacter(isGm bool) character.Model {
	gm := 0
	if isGm {
		gm = 1
	}
	return character.NewModelBuilder().SetId(1).SetGm(gm).SetMapId(100000000).Build()
}

func TestParseSpawnArgs(t *testing.T) {
	testCases := []struct {
		name       string
		message    string
		wantOk     bool
		wantId     uint32
		wantRaw    int
	}{
		{"single", "@mob spawn 100100", true, 100100, 1},
		{"with count", "@mob spawn 100100 5", true, 100100, 5},
		{"extra whitespace", "@mob spawn   100100   5", true, 100100, 5},
		{"count zero", "@mob spawn 100100 0", true, 100100, 0},
		{"kill all not matched", "@mob kill all", false, 0, 0},
		{"mobstatus not matched", "@mobstatus 100", false, 0, 0},
		{"mobclear not matched", "@mobclear", false, 0, 0},
		{"plain chat not matched", "hello world", false, 0, 0},
		{"trailing junk not matched", "@mob spawn 100100 5 extra", false, 0, 0},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			id, raw, ok := parseSpawnArgs(tc.message)
			if ok != tc.wantOk {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOk)
			}
			if !tc.wantOk {
				return
			}
			if id != tc.wantId {
				t.Errorf("id = %d, want %d", id, tc.wantId)
			}
			if raw != tc.wantRaw {
				t.Errorf("raw = %d, want %d", raw, tc.wantRaw)
			}
		})
	}
}

func TestNormalizeCount(t *testing.T) {
	testCases := []struct {
		name       string
		raw        int
		wantCount  int
		wantCapped bool
		wantValid  bool
	}{
		{"one", 1, 1, false, true},
		{"mid", 5, 5, false, true},
		{"at cap", 20, 20, false, true},
		{"over cap", 21, 20, true, true},
		{"zero invalid", 0, 0, false, false},
		{"negative invalid", -3, 0, false, false},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			count, capped, valid := normalizeCount(tc.raw)
			if count != tc.wantCount || capped != tc.wantCapped || valid != tc.wantValid {
				t.Errorf("normalizeCount(%d) = (%d, %v, %v), want (%d, %v, %v)",
					tc.raw, count, capped, valid, tc.wantCount, tc.wantCapped, tc.wantValid)
			}
		})
	}
}

func TestMobSpawnCommandProducer_GmGate(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := context.Background()
	f := field.NewBuilder(1, 1, 100000000).Build()

	testCases := []struct {
		name        string
		isGm        bool
		message     string
		expectFound bool
	}{
		{"GM spawn matches", true, "@mob spawn 100100", true},
		{"GM spawn with count matches", true, "@mob spawn 100100 5", true},
		{"GM count zero still matches (executor reports error)", true, "@mob spawn 100100 0", true},
		{"non-GM does not match", false, "@mob spawn 100100", false},
		{"GM kill all does not match this producer", true, "@mob kill all", false},
		{"GM plain chat does not match", true, "hi", false},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			char := testCharacter(tc.isGm)
			executor, found := MobSpawnCommandProducer(logger)(ctx)(f, char, tc.message)
			if found != tc.expectFound {
				t.Fatalf("found = %v, want %v", found, tc.expectFound)
			}
			if tc.expectFound && executor == nil {
				t.Error("expected non-nil executor when found")
			}
			if !tc.expectFound && executor != nil {
				t.Error("expected nil executor when not found")
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-messages/atlas.com/messages && go test ./command/monster/ -v`
Expected: FAIL — `parseSpawnArgs`, `normalizeCount`, `MobSpawnCommandProducer` undefined.

- [ ] **Step 3: Add imports and helpers to `commands.go`**

In `commands.go`, update the import block to add the data clients (and keep existing imports):

```go
import (
	"atlas-messages/character"
	"atlas-messages/command"
	"atlas-messages/data/foothold"
	monsterdata "atlas-messages/data/monster"
	"atlas-messages/kafka/message/monster"
	"atlas-messages/kafka/producer"
	"atlas-messages/message"
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	monster2 "github.com/Chronicle20/atlas/libs/atlas-constants/monster"
	"github.com/sirupsen/logrus"
)
```

Then add the helpers and producer at the end of the file (before `isValidStatus`):

```go
const spawnCountCap = 20

var spawnRe = regexp.MustCompile(`^@mob spawn\s+(\d+)(?:\s+(\d+))?$`)

// parseSpawnArgs extracts the template id and raw count from a "@mob spawn"
// message. ok is false when the message is not a spawn command. A non-numeric
// or overflowing count is normalized to spawnCountCap+1 so it clamps downstream.
func parseSpawnArgs(m string) (templateId uint32, rawCount int, ok bool) {
	match := spawnRe.FindStringSubmatch(m)
	if match == nil {
		return 0, 0, false
	}
	id, err := strconv.ParseUint(match[1], 10, 32)
	if err != nil {
		return 0, 0, false
	}
	rawCount = 1
	if match[2] != "" {
		c, cerr := strconv.Atoi(match[2])
		if cerr != nil {
			c = spawnCountCap + 1
		}
		rawCount = c
	}
	return uint32(id), rawCount, true
}

// normalizeCount validates and clamps the requested spawn count. valid is false
// when below 1; capped is true when the request exceeded the cap.
func normalizeCount(raw int) (count int, capped bool, valid bool) {
	if raw < 1 {
		return 0, false, false
	}
	if raw > spawnCountCap {
		return spawnCountCap, true, true
	}
	return raw, false, true
}

func MobSpawnCommandProducer(l logrus.FieldLogger) func(ctx context.Context) func(f field.Model, c character.Model, m string) (command.Executor, bool) {
	return func(ctx context.Context) func(f field.Model, c character.Model, m string) (command.Executor, bool) {
		return func(f field.Model, c character.Model, m string) (command.Executor, bool) {
			templateId, rawCount, ok := parseSpawnArgs(m)
			if !ok {
				return nil, false
			}

			if !c.Gm() {
				return nil, false
			}

			return func(l logrus.FieldLogger) func(ctx context.Context) error {
				return func(ctx context.Context) error {
					msgProc := message.NewProcessor(l, ctx)

					count, capped, valid := normalizeCount(rawCount)
					if !valid {
						return msgProc.IssuePinkText(f, 0, "Count must be at least 1.", []uint32{c.Id()})
					}

					mon, err := monsterdata.NewProcessor(l, ctx).GetById(templateId)
					if err != nil {
						return msgProc.IssuePinkText(f, 0, fmt.Sprintf("Unknown monster template: %d", templateId), []uint32{c.Id()})
					}

					var fh int16
					if fhModel, ferr := foothold.NewProcessor(l, ctx).GetBelow(f.MapId(), c.X(), c.Y()); ferr != nil {
						l.WithError(ferr).Warnf("Unable to resolve foothold below (%d, %d) on map [%d]; spawning with fh=0.", c.X(), c.Y(), uint32(f.MapId()))
					} else {
						fh = int16(fhModel.Id())
					}

					err = producer.ProviderImpl(l)(ctx)(monster.EnvCommandTopic)(monster.SpawnFieldCommandProvider(f.WorldId(), f.ChannelId(), f.MapId(), f.Instance(), templateId, c.X(), c.Y(), fh, 0, count))
					if err != nil {
						return msgProc.IssuePinkText(f, 0, fmt.Sprintf("Failed to spawn monster %d.", templateId), []uint32{c.Id()})
					}

					text := fmt.Sprintf("Spawned %dx monster %d (%s) at (%d, %d).", count, templateId, mon.Name(), c.X(), c.Y())
					if capped {
						text += fmt.Sprintf(" Capped to %d.", spawnCountCap)
					}
					return msgProc.IssuePinkText(f, 0, text, []uint32{c.Id()})
				}
			}, true
		}
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-messages/atlas.com/messages && go test ./command/monster/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-messages/atlas.com/messages/command/monster/commands.go services/atlas-messages/atlas.com/messages/command/monster/commands_test.go
git commit -m "feat(atlas-messages): add @mob spawn command producer"
```

---

## Task 6: atlas-messages — register producer and add help text

**Context:** Register the new producer in `main.go` alongside the other `@mob` registrations, and add a help line to the `@help` output (`command/help/commands.go`).

**Files:**
- Modify: `services/atlas-messages/atlas.com/messages/main.go:56-58`
- Modify: `services/atlas-messages/atlas.com/messages/command/help/commands.go`

- [ ] **Step 1: Register the producer in `main.go`**

In `main.go`, add the registration immediately after the existing `@mob` registrations (after line 58):

```go
	command.Registry().Add(monster.MobKillAllCommandProducer)
	command.Registry().Add(monster.MobStatusCommandProducer)
	command.Registry().Add(monster.MobClearCommandProducer)
	command.Registry().Add(monster.MobSpawnCommandProducer)
```

- [ ] **Step 2: Add the help line**

In `command/help/commands.go`, add to the `commandSyntaxList` slice (after the `@mob kill all` entry):

```go
	"@mob kill all - Kill all monsters in the current map",
	"@mob spawn <templateId> [count] - Spawn a monster at your position (count 1-20)",
```

- [ ] **Step 3: Verify the messages module builds**

Run: `cd services/atlas-messages/atlas.com/messages && go build ./... && go vet ./...`
Expected: clean (no output).

- [ ] **Step 4: Run the full messages test suite**

Run: `cd services/atlas-messages/atlas.com/messages && go test -race ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-messages/atlas.com/messages/main.go services/atlas-messages/atlas.com/messages/command/help/commands.go
git commit -m "feat(atlas-messages): register @mob spawn and add help text"
```

---

## Task 7: atlas-monsters — add `SPAWN_FIELD` body and constant

**Context:** Add the `SPAWN_FIELD` command type and its body struct to the atlas-monsters consumer contract. The body must decode the exact JSON the atlas-messages provider emits (`monsterId`, `x`, `y`, `fh`, `team`). Uses the existing `fieldCommand[E]` envelope (no `MonsterId` slot).

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka.go`
- Test: `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka_test.go`

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka_test.go`:

```go
func TestSpawnFieldCommandBody_Decode(t *testing.T) {
	raw := []byte(`{"monsterId":100100,"x":250,"y":-130,"fh":7,"team":0}`)
	var body spawnFieldCommandBody
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.MonsterId != 100100 {
		t.Errorf("MonsterId = %d, want 100100", body.MonsterId)
	}
	if body.X != 250 || body.Y != -130 {
		t.Errorf("position = (%d, %d), want (250, -130)", body.X, body.Y)
	}
	if body.Fh != 7 {
		t.Errorf("Fh = %d, want 7", body.Fh)
	}
	if body.Team != 0 {
		t.Errorf("Team = %d, want 0", body.Team)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-monsters/atlas.com/monsters && go test ./kafka/consumer/monster/ -run TestSpawnFieldCommandBody_Decode -v`
Expected: FAIL — `spawnFieldCommandBody` undefined.

- [ ] **Step 3: Add the constant**

In `kafka.go`, add to the `const` block:

```go
	CommandTypeDestroyField      = "DESTROY_FIELD"
	CommandTypeSpawnField        = "SPAWN_FIELD"
```

- [ ] **Step 4: Add the body struct**

In `kafka.go`, add after `destroyFieldCommandBody`:

```go
type spawnFieldCommandBody struct {
	MonsterId uint32 `json:"monsterId"`
	X         int16  `json:"x"`
	Y         int16  `json:"y"`
	Fh        int16  `json:"fh"`
	Team      int8   `json:"team"`
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd services/atlas-monsters/atlas.com/monsters && go test ./kafka/consumer/monster/ -run TestSpawnFieldCommandBody_Decode -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka.go services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka_test.go
git commit -m "feat(atlas-monsters): add SPAWN_FIELD command type and body"
```

---

## Task 8: atlas-monsters — `handleSpawnFieldCommand` consumer

**Context:** Register a new handler discriminating on `Type == "SPAWN_FIELD"` that builds the field (with instance) and calls the existing `monster.Processor.Create`. `Create`'s internal `information.GetById` remains the server-side template safety net. Mirrors `handleDestroyFieldCommand`.

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/consumer.go`

- [ ] **Step 1: Add the handler function**

In `consumer.go`, append after `handleUseSkillFieldCommand`:

```go
func handleSpawnFieldCommand(l logrus.FieldLogger, ctx context.Context, c fieldCommand[spawnFieldCommandBody]) {
	if c.Type != CommandTypeSpawnField {
		return
	}

	f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).SetInstance(c.Instance).Build()
	p := monster.NewProcessor(l, ctx)
	_, err := p.Create(f, monster.RestModel{
		MonsterId: c.Body.MonsterId,
		X:         c.Body.X,
		Y:         c.Body.Y,
		Fh:        c.Body.Fh,
		Team:      c.Body.Team,
	})
	if err != nil {
		l.WithError(err).Errorf("SPAWN_FIELD failed for template [%d] in field [%s].", c.Body.MonsterId, f.Id())
	}
}
```

- [ ] **Step 2: Register the handler in `InitHandlers`**

In `consumer.go`, add the registration after the `handleDestroyFieldCommand` block (after line 63, before the movement-topic line):

```go
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleDestroyFieldCommand))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleSpawnFieldCommand))); err != nil {
			return err
		}
```

- [ ] **Step 3: Verify the monsters module builds and vets**

Run: `cd services/atlas-monsters/atlas.com/monsters && go build ./... && go vet ./...`
Expected: clean.

- [ ] **Step 4: Run the full monsters test suite**

Run: `cd services/atlas-monsters/atlas.com/monsters && go test -race ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/consumer.go
git commit -m "feat(atlas-monsters): consume SPAWN_FIELD command via Create path"
```

---

## Task 9: Full verification (per CLAUDE.md)

**Context:** Both changed Go modules (atlas-messages, atlas-monsters) must pass the full verification gate. No new shared lib was added, so no Dockerfile/`go.work` edits are expected. Run from the worktree root.

**Files:** None (verification only).

- [ ] **Step 1: Test + vet both modules with race detector**

```bash
cd services/atlas-messages/atlas.com/messages && go test -race ./... && go vet ./...
cd - && cd services/atlas-monsters/atlas.com/monsters && go test -race ./... && go vet ./...
```
Expected: all PASS, vet clean.

- [ ] **Step 2: Build both modules**

```bash
cd services/atlas-messages/atlas.com/messages && go build ./...
cd - && cd services/atlas-monsters/atlas.com/monsters && go build ./...
```
Expected: clean.

- [ ] **Step 3: Docker bake both services from the worktree root**

```bash
docker buildx bake atlas-messages
docker buildx bake atlas-monsters
```
Expected: both succeed.

- [ ] **Step 4: Redis key guard from the worktree root**

```bash
tools/redis-key-guard.sh
```
Expected: clean (no new raw keyed go-redis usage was introduced).

- [ ] **Step 5: Confirm worktree/branch and final status**

```bash
git rev-parse --show-toplevel   # must end with /.worktrees/task-077-gm-mob-spawn-command
git branch --show-current        # must be task-077-gm-mob-spawn-command
git status
```
Expected: correct worktree + branch, clean tree (all work committed).
