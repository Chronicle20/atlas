# Character Spawn Point Plumbing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `spawnPoint` propagate faithfully through every service that mirrors the `atlas-character` model — un-stub eight `Model.SpawnPoint()` accessors, decode the field in the four `Extract`s that drop it, and narrow to `byte` only at the two wire call sites — with observable wire output unchanged.

**Architecture:** Eight independent Go modules each get the same three-part edit: accessor returns `m.spawnPoint` typed `uint32`; `Extract` decodes `spawnPoint`; consumers of the accessor cast at the boundary. Each service is its own Go module, so the `byte` → `uint32` return-type change is a compile break contained entirely within that service's task — there is no cross-module ordering hazard. `atlas-character` and `libs/atlas-packet` are not touched.

**Tech Stack:** Go 1.27, standard `testing` package, project Builder pattern, JSON:API via `github.com/jtumidanski/api2go/jsonapi`.

**Spec:** `docs/tasks/task-272-character-spawn-point-plumbing/design.md` (PRD: `prd.md`)

## Global Constraints

- `Model.SpawnPoint()` MUST return `uint32` in all eight services and MUST return `m.spawnPoint`. The `byte` return type MUST NOT be retained anywhere. (PRD FR-1, FR-6)
- Narrowing to `byte` happens **only** at the two wire call sites, via an explicit `byte(...)` cast. (PRD FR-2, design §3)
- `services/atlas-character/**` MUST NOT appear in the diff. It is the system of record and is already correct.
- `libs/atlas-packet/**` MUST NOT appear in the diff. The wire format is correct as-is.
- **Every new or amended test fixture MUST use a non-zero `spawnPoint` value.** A zero-valued fixture passes against the stub and proves nothing. (design §4.2, §9)
- **Do not write `Extract∘Transform` idempotence assertions as the proof for this fix.** A field that `Extract` drops is zero on both sides, so that shape is blind to this exact defect class (design §4.1). Every new assertion compares against the literal input value.
- Do not add a producer for `spawnPoint`; nothing should call `UpdateSpawnPoint`. Do not deduplicate the eight model copies. Do not touch the `Rank`/`RankMove`/`JobRank`/`JobRankMove` stubs or `atlas-messages`' `Stance()` stub.
- Do not widen `atlas-pets` beyond `spawnPoint`. Its other ~26 dropped fields stay dropped.
- Test setup constructing a `Model` uses the package `Builder`. Constructing a `RestModel` struct literal is **not** a test-only constructor — it is the declared parameter type of `Extract` and is the sanctioned input shape (design §5.3).
- Commit per task. Do not run `tools/verify.sh` inside an implementer task; Task 9 owns the repo-wide gate.

---

## Task 1: atlas-channel — accessor, Extract, and the CHARACTER_DATA wire cast

**Module root (all `go build` / `go test` run from here):** `services/atlas-channel/atlas.com/channel`

### Files

- `services/atlas-channel/atlas.com/channel/character/model.go` — un-stub `SpawnPoint()` at line 240
- `services/atlas-channel/atlas.com/channel/character/rest.go` — add `spawnPoint` to `Extract` (starts line 125)
- `services/atlas-channel/atlas.com/channel/socket/writer/character_data.go` — `byte(...)` cast at line 47
- `services/atlas-channel/atlas.com/channel/character/rest_test.go` — add a value assertion to the existing round-trip test
- `services/atlas-channel/atlas.com/channel/socket/writer/character_data_test.go` — new test function

Patterns to copy: `services/atlas-channel/atlas.com/channel/socket/writer/character_data_test.go:41-56` (`TestBuildCharacterData_TeleportMaps` — the `character.NewBuilder().SetId(99).SetSp("0").MustBuild()` fixture; a bare `character.Model{}` panics in `RemainingSp()`).

**Interfaces:**
- Produces: `func (m Model) SpawnPoint() uint32` in package `atlas-channel/character`. Task 9 greps for the absence of `return 0` bodies.
- Consumes: nothing from other tasks.

- [ ] **Step 1: Write the failing tests**

Two files.

**(a)** In `character/rest_test.go`, inside the existing `TestTransformRoundTrip`, immediately after the `m, err := Extract(rm)` error check (the fixture already carries `SpawnPoint: 11` at line 43), add:

```go
	// spawnPoint is the field task-272 fixes. The DeepEqual round-trip below
	// cannot see a dropped field -- it is zero on both sides -- so assert the
	// decoded value against the RestModel literal directly.
	if got := m.SpawnPoint(); got != 11 {
		t.Errorf("SpawnPoint() = %d, want 11", got)
	}
```

**(b)** In `socket/writer/character_data_test.go`, add a new test at the end of the file:

```go
// TestBuildCharacterData_SpawnPoint pins both halves of the task-272 fix: the
// model value reaches the wire struct, and the uint32 -> byte narrowing at the
// wire boundary truncates rather than erroring. Truncation above 255 is a
// pre-existing property of the wire format (one byte), asserted here so it is
// documented rather than latent.
func TestBuildCharacterData_SpawnPoint(t *testing.T) {
	tests := []struct {
		name  string
		set   uint32
		want  byte
	}{
		{name: "in range", set: 7, want: 7},
		{name: "truncates above 255", set: 256, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := character.NewBuilder().
				SetId(99).
				SetSp("0").
				SetSpawnPoint(tt.set).
				MustBuild()

			cd := BuildCharacterData(logrus.New(), context.Background(), c, buddylist.Model{}, _map.Id(0), teleportrock.Model{})

			if cd.Stats.SpawnPoint != tt.want {
				t.Errorf("Stats.SpawnPoint = %d, want %d", cd.Stats.SpawnPoint, tt.want)
			}
		})
	}
}
```

All imports this needs (`character`, `buddylist`, `teleportrock`, `context`, `testing`, `logrus`, `_map`) are already present in that file.

- [ ] **Step 2: Run the tests and verify they fail**

Run from `services/atlas-channel/atlas.com/channel`:

```
go test ./character/... ./socket/writer/... -run 'TransformRoundTrip|SpawnPoint' -v
```

Expected: both fail. `TestTransformRoundTrip` fails `SpawnPoint() = 0, want 11`. `TestBuildCharacterData_SpawnPoint/in_range` fails `Stats.SpawnPoint = 0, want 7`. (`truncates above 255` passes vacuously against the stub — that is expected and not evidence of anything yet.)

- [ ] **Step 3: Un-stub the accessor**

`character/model.go:240`, replace:

```go
func (m Model) SpawnPoint() byte {
	return 0
}
```

with:

```go
func (m Model) SpawnPoint() uint32 {
	return m.spawnPoint
}
```

- [ ] **Step 4: Decode `spawnPoint` in `Extract`**

`character/rest.go`, in `Extract` (starts line 125), insert between the `sp:` and `gm:` lines so the field order mirrors `Transform` (line 116):

```go
		spawnPoint:         m.SpawnPoint,
```

- [ ] **Step 5: Narrow at the wire boundary**

`socket/writer/character_data.go:47`, replace `SpawnPoint: c.SpawnPoint(),` with:

```go
			SpawnPoint: byte(c.SpawnPoint()),
```

- [ ] **Step 6: Build and run the full module test suite**

```
go build ./... && go test ./...
```

Expected: build succeeds (confirming `character_data.go:47` was the only in-service consumer of the changed signature) and all tests pass.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/character/model.go \
        services/atlas-channel/atlas.com/channel/character/rest.go \
        services/atlas-channel/atlas.com/channel/character/rest_test.go \
        services/atlas-channel/atlas.com/channel/socket/writer/character_data.go \
        services/atlas-channel/atlas.com/channel/socket/writer/character_data_test.go
git commit -m "fix(atlas-channel): decode and return spawnPoint, narrow at the wire"
```

---

## Task 2: atlas-login — accessor, Extract, and the CHARACTER_LIST wire cast

**Module root:** `services/atlas-login/atlas.com/login`

### Files

- `services/atlas-login/atlas.com/login/character/model.go` — un-stub `SpawnPoint()` at line 222
- `services/atlas-login/atlas.com/login/character/rest.go` — add `.SetSpawnPoint(...)` to `Extract`'s builder chain (starts line 129)
- `services/atlas-login/atlas.com/login/socket/writer/character_list.go` — `byte(...)` cast at line 56
- `services/atlas-login/atlas.com/login/character/rest_test.go` — amend the fixture, the stale doc comment, and add an assertion
- `services/atlas-login/atlas.com/login/socket/writer/character_list_test.go` — **new file**

Read-only reference: `services/atlas-login/atlas.com/login/character/builder.go:82` — `SetSpawnPoint(v uint32) *Builder` already exists; do not add it.

Patterns to copy: `services/atlas-login/atlas.com/login/socket/writer/server_list_test.go:24-30` (package `writer` test; `pt.CreateContext("GMS", 83, 1)` for a tenant-carrying context and `testlog.NewNullLogger()` for a silent logger).

**Interfaces:**
- Produces: `func (m Model) SpawnPoint() uint32` in package `atlas-login/character`.
- Consumes: nothing from other tasks.

**Note on the new writer test:** `toCharacterListEntry` calls `location.GetField`, which issues a REST request. With no `MAPS_SERVICE`/base URL set in the test environment the request fails immediately (unsupported protocol scheme) — it does not hang and does not panic; the function logs a warning and renders `mapId = 0`. That is the intended tested path, and the null logger keeps it quiet.

- [ ] **Step 1: Write the failing tests**

Two files.

**(a)** In `character/rest_test.go`: the doc comment at lines 11-15 asserts that `Extract`'s builder chain "never calls `SetSpawnPoint`" — after this task that is false, so it must be corrected in the same edit. Replace the `SetSpawnPoint, ` fragment of the comment's list so it reads:

```go
// TestTransformRoundTrip asserts Extract(Transform(m)) reproduces every
// field Extract actually populates. Extract's builder chain never calls
// SetPets, SetEquipment, SetInventory, SetRank, SetRankMove, SetJobRank,
// or SetJobRankMove, so those Model fields do not survive a round trip
// regardless of what Transform emits; see
// docs/tasks/task-263-backend-guideline-conformance/handwork-notes.md.
// spawnPoint DOES survive as of task-272 and is asserted below.
```

Add `SetSpawnPoint(25).` to the builder chain in the fixture, immediately after `SetSp("1,2,3").`, and add this assertion after the `got, err := Extract(rm)` error check:

```go
	if got.SpawnPoint() != 25 {
		t.Errorf("SpawnPoint() = %d, want 25", got.SpawnPoint())
	}
	if got.spawnPoint != m.spawnPoint {
		t.Errorf("spawnPoint mismatch. Expected %v, got %v", m.spawnPoint, got.spawnPoint)
	}
```

**(b)** Create `socket/writer/character_list_test.go`:

```go
package writer

import (
	"atlas-login/character"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestToCharacterListEntry_SpawnPoint pins both halves of the task-272 fix on
// the CHARACTER_LIST path: the model value reaches the packet statistics
// struct, and the uint32 -> byte narrowing at the wire boundary truncates
// rather than erroring. Truncation above 255 is a pre-existing property of the
// wire format (one byte), asserted here so it is documented rather than latent.
//
// toCharacterListEntry calls location.GetField, which fails fast with no MAPS
// base URL configured; the entry then renders mapId = 0. That is expected and
// irrelevant to this assertion.
func TestToCharacterListEntry_SpawnPoint(t *testing.T) {
	tests := []struct {
		name string
		set  uint32
		want byte
	}{
		{name: "in range", set: 7, want: 7},
		{name: "truncates above 255", set: 256, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := pt.CreateContext("GMS", 83, 1)
			l, _ := testlog.NewNullLogger()

			c := character.NewBuilder().
				SetId(99).
				SetSp("0").
				SetSpawnPoint(tt.set).
				Build()

			entry := toCharacterListEntry(l, ctx, c, false)

			if got := entry.Statistics().SpawnPoint(); got != tt.want {
				t.Errorf("SpawnPoint() = %d, want %d", got, tt.want)
			}
		})
	}
}
```

Both accessors this test reads are confirmed to exist and need no new export: `func (m CharacterListEntry) Statistics() CharacterStatistics` at `libs/atlas-packet/model/character_list_entry.go:37`, and `func (m CharacterStatistics) SpawnPoint() byte` at `libs/atlas-packet/model/character_statistics.go:86`. Do not modify `libs/atlas-packet` to add anything.

- [ ] **Step 2: Run the tests and verify they fail**

Run from `services/atlas-login/atlas.com/login`:

```
go test ./character/... ./socket/writer/... -run 'TransformRoundTrip|SpawnPoint' -v
```

Expected: `TestTransformRoundTrip` fails `SpawnPoint() = 0, want 25`; `TestToCharacterListEntry_SpawnPoint/in_range` fails `SpawnPoint() = 0, want 7`.

- [ ] **Step 3: Un-stub the accessor**

`character/model.go:222`:

```go
func (m Model) SpawnPoint() uint32 {
	return m.spawnPoint
}
```

- [ ] **Step 4: Decode `spawnPoint` in `Extract`**

`character/rest.go`, in `Extract`'s builder chain, insert after `SetSp(m.Sp).`:

```go
		SetSpawnPoint(m.SpawnPoint).
```

- [ ] **Step 5: Narrow at the wire boundary**

`socket/writer/character_list.go:56`, replace `uint32(mapId), c.SpawnPoint(),` with:

```go
		uint32(mapId), byte(c.SpawnPoint()),
```

- [ ] **Step 6: Build and run the full module test suite**

```
go build ./... && go test ./...
```

Expected: build succeeds and all tests pass.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-login/atlas.com/login/character/model.go \
        services/atlas-login/atlas.com/login/character/rest.go \
        services/atlas-login/atlas.com/login/character/rest_test.go \
        services/atlas-login/atlas.com/login/socket/writer/character_list.go \
        services/atlas-login/atlas.com/login/socket/writer/character_list_test.go
git commit -m "fix(atlas-login): decode and return spawnPoint, narrow at the wire"
```

---

## Task 3: atlas-query-aggregator — accessor and the un-cast REST re-serve

**Module root:** `services/atlas-query-aggregator/atlas.com/query-aggregator`

This is the one service whose *live response value* changes: `Transform` currently launders the hardcoded `0` back out as the `spawnPoint` attribute. Its `Extract` (line 139) already decodes the field correctly and MUST NOT be changed.

### Files

- `services/atlas-query-aggregator/atlas.com/query-aggregator/character/model.go` — un-stub `SpawnPoint()` at line 224
- `services/atlas-query-aggregator/atlas.com/query-aggregator/character/rest.go` — drop the `uint32(...)` cast at line 128 (inside `Transform`, which starts at line 92)
- `services/atlas-query-aggregator/atlas.com/query-aggregator/character/rest_test.go` — **new file**

Read-only references: `.../character/builder.go:92` (`NewBuilder`), `:116` (`SetSp(v string)`), `:123` (`SetSpawnPoint(v uint32)`), `:133` (`Build()`).

**Interfaces:**
- Produces: `func (m Model) SpawnPoint() uint32` in package `atlas-query-aggregator/character`.
- Consumes: nothing from other tasks.

- [ ] **Step 1: Write the failing test**

Create `character/rest_test.go`. `Transform` calls `m.Sp()`, which splits `m.sp` on `","`, so the builder must set a parseable `Sp`.

```go
package character

import (
	"testing"
)

// TestExtract_SpawnPoint asserts the inbound seam: the RestModel value reaches
// the model. Anchored to the RestModel literal rather than to a second derived
// model -- an Extract/Transform idempotence assertion is blind to a dropped
// field, because a dropped field is zero on both sides of the comparison.
func TestExtract_SpawnPoint(t *testing.T) {
	rm := RestModel{Id: 1, Sp: "0", SpawnPoint: 11}

	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if got := m.SpawnPoint(); got != 11 {
		t.Errorf("SpawnPoint() = %d, want 11", got)
	}
}

// TestTransform_SpawnPointPreservesUint32 pins the outbound seam at full
// uint32 fidelity. This service re-serves spawnPoint over JSON:API rather than
// over the wire, so the value must NOT be narrowed to a byte here; 300 is
// above the byte ceiling precisely so a reintroduced cast fails loudly.
func TestTransform_SpawnPointPreservesUint32(t *testing.T) {
	m := NewBuilder().
		SetId(1).
		SetSp("0").
		SetSpawnPoint(300).
		Build()

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if rm.SpawnPoint != 300 {
		t.Errorf("SpawnPoint = %d, want 300", rm.SpawnPoint)
	}
}
```

- [ ] **Step 2: Run the tests and verify they fail**

Run from `services/atlas-query-aggregator/atlas.com/query-aggregator`:

```
go test ./character/... -run SpawnPoint -v
```

Expected: `TestExtract_SpawnPoint` fails `SpawnPoint() = 0, want 11`; `TestTransform_SpawnPointPreservesUint32` fails `SpawnPoint = 0, want 300`.

- [ ] **Step 3: Un-stub the accessor**

`character/model.go:224`:

```go
func (m Model) SpawnPoint() uint32 {
	return m.spawnPoint
}
```

- [ ] **Step 4: Drop the narrowing cast on the REST re-serve**

`character/rest.go:128`, replace `SpawnPoint: uint32(m.SpawnPoint()),` with:

```go
		SpawnPoint:         m.SpawnPoint(),
```

- [ ] **Step 5: Build and run the full module test suite**

```
go build ./... && go test ./...
```

Expected: build succeeds and all tests pass. Confirm `Extract` at line 139 is untouched by the diff.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-query-aggregator/atlas.com/query-aggregator/character/model.go \
        services/atlas-query-aggregator/atlas.com/query-aggregator/character/rest.go \
        services/atlas-query-aggregator/atlas.com/query-aggregator/character/rest_test.go
git commit -m "fix(atlas-query-aggregator): serve the real spawnPoint at uint32 fidelity"
```

---

## Task 4: atlas-cashshop — accessor and Extract

**Module root:** `services/atlas-cashshop/atlas.com/cashshop`

No wire or REST consumer of this accessor exists in this service; the fix is latent-copy hygiene so that wiring a writer to it later does not silently reintroduce the bug.

### Files

- `services/atlas-cashshop/atlas.com/cashshop/character/model.go` — un-stub `SpawnPoint()` at line 211
- `services/atlas-cashshop/atlas.com/cashshop/character/rest.go` — add `spawnPoint` to `Extract` (starts line 128)
- `services/atlas-cashshop/atlas.com/cashshop/character/rest_test.go` — replace the zero fixture value and add an assertion

Read-only reference: `.../character/rest.go:120` — `Transform` already emits `SpawnPoint`.

**Interfaces:**
- Produces: `func (m Model) SpawnPoint() uint32` in package `atlas-cashshop/character`.
- Consumes: nothing from other tasks.

- [ ] **Step 1: Write the failing test**

In `character/rest_test.go`'s `TestTransformRoundTrip`, change the fixture line `SpawnPoint: 0,` (line 43) to:

```go
		SpawnPoint:         11,
```

and add, after the `m, err := Extract(rm)` error check:

```go
	// A zero-valued spawnPoint fixture passes against the hardcoded stub this
	// task removes, and the DeepEqual round-trip below cannot see a dropped
	// field. Assert the decoded value against the RestModel literal directly.
	if got := m.SpawnPoint(); got != 11 {
		t.Errorf("SpawnPoint() = %d, want 11", got)
	}
```

- [ ] **Step 2: Run the test and verify it fails**

Run from `services/atlas-cashshop/atlas.com/cashshop`:

```
go test ./character/... -run TransformRoundTrip -v
```

Expected: FAIL with `SpawnPoint() = 0, want 11`.

- [ ] **Step 3: Un-stub the accessor**

`character/model.go:211`:

```go
func (m Model) SpawnPoint() uint32 {
	return m.spawnPoint
}
```

- [ ] **Step 4: Decode `spawnPoint` in `Extract`**

`character/rest.go`, in `Extract` (starts line 128), insert between the `sp:` and `gm:` lines:

```go
		spawnPoint:         m.SpawnPoint,
```

- [ ] **Step 5: Build and run the full module test suite**

```
go build ./... && go test ./...
```

Expected: build succeeds and all tests pass.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-cashshop/atlas.com/cashshop/character/model.go \
        services/atlas-cashshop/atlas.com/cashshop/character/rest.go \
        services/atlas-cashshop/atlas.com/cashshop/character/rest_test.go
git commit -m "fix(atlas-cashshop): decode and return spawnPoint"
```

---

## Task 5: atlas-pets — accessor, Extract, and Transform

**Module root:** `services/atlas-pets/atlas.com/pets`

`atlas-pets` is the only service where `Transform` also drops `spawnPoint` (it emits just `Id`/`X`/`Y`/`Stance`), so a one-sided fix would leave the seam broken outbound. Design §5.1 decides both legs are in scope for **`spawnPoint` only**. `character.Transform` in this service has zero callers outside `rest_test.go`, so no live payload changes. **The remaining ~26 fields pets drops in both directions stay out of scope — do not "finish" the struct literals.**

### Files

- `services/atlas-pets/atlas.com/pets/character/model.go` — un-stub `SpawnPoint()` at line 207
- `services/atlas-pets/atlas.com/pets/character/rest.go` — add `spawnPoint` to `Extract` (line 97) and `SpawnPoint` to `Transform` (line 106)
- `services/atlas-pets/atlas.com/pets/character/rest_test.go` — amend the fixture and add a test

Read-only reference: `.../character/rest.go:38` — `RestModel.SpawnPoint uint32` already exists.

**Interfaces:**
- Produces: `func (m Model) SpawnPoint() uint32` in package `atlas-pets/character`.
- Consumes: nothing from other tasks.

- [ ] **Step 1: Write the failing tests**

In `character/rest_test.go`, add `spawnPoint: 55,` to the existing `TestTransformRoundTrip` fixture (after `stance: 44,`) and add an explicit assertion before the `DeepEqual` check:

```go
	if got.SpawnPoint() != 55 {
		t.Errorf("SpawnPoint() = %d, want 55", got.SpawnPoint())
	}
```

Then add a new test asserting the inbound leg against a `RestModel` literal:

```go
// TestExtract_SpawnPoint asserts the inbound seam directly. The round-trip
// test above cannot prove it alone: a field dropped by both Extract and
// Transform is zero on both sides of a DeepEqual and hides in plain sight.
func TestExtract_SpawnPoint(t *testing.T) {
	rm := RestModel{Id: 11, SpawnPoint: 55}

	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	if got := m.SpawnPoint(); got != 55 {
		t.Errorf("SpawnPoint() = %d, want 55", got)
	}
}
```

- [ ] **Step 2: Run the tests and verify they fail**

Run from `services/atlas-pets/atlas.com/pets`:

```
go test ./character/... -run SpawnPoint -v
```

Expected: `TestExtract_SpawnPoint` fails `SpawnPoint() = 0, want 55`.

Also run `go test ./character/... -run TransformRoundTrip -v`: it fails `SpawnPoint() = 0, want 55`.

- [ ] **Step 3: Un-stub the accessor**

`character/model.go:207`:

```go
func (m Model) SpawnPoint() uint32 {
	return m.spawnPoint
}
```

- [ ] **Step 4: Add `spawnPoint` to both legs**

`character/rest.go`, `Extract` (line 97) becomes:

```go
func Extract(m RestModel) (Model, error) {
	return Model{
		id:         m.Id,
		x:          m.X,
		y:          m.Y,
		stance:     m.Stance,
		spawnPoint: m.SpawnPoint,
	}, nil
}
```

and `Transform` (line 106) becomes:

```go
func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:         m.id,
		X:          m.x,
		Y:          m.y,
		Stance:     m.stance,
		SpawnPoint: m.spawnPoint,
	}, nil
}
```

- [ ] **Step 5: Build and run the full module test suite**

```
go build ./... && go test ./...
```

Expected: build succeeds and all tests pass, including the unchanged `DeepEqual` round-trip (both legs now carry the field, so `m` still equals `got`).

- [ ] **Step 6: Commit**

```bash
git add services/atlas-pets/atlas.com/pets/character/model.go \
        services/atlas-pets/atlas.com/pets/character/rest.go \
        services/atlas-pets/atlas.com/pets/character/rest_test.go
git commit -m "fix(atlas-pets): plumb spawnPoint through both Extract and Transform"
```

---

## Task 6: atlas-npc-shops — accessor, positional fields, and the builder setters

**Module root:** `services/atlas-npc-shops/atlas.com/npc`

Two defects here. The accessor stub is the same as everywhere else. Separately, `Extract` drops `x`, `y`, and `stance` even though `Model` carries them with real accessors and `Clone`/`Build` round-trip them (PRD FR-7).

**Design §5.2 overrides PRD FR-8:** `SetX`/`SetY`/`SetStance` **are** added to this service's builder. They will have no production caller — that is expected and must not be reported as a defect. The `Builder` struct already declares the three fields (`builder.go:75-77`) and `Build` already copies them (`:146-150`); only the setters are missing, which is why non-zero positional values cannot be originated through the sanctioned Builder path today.

### Files

- `services/atlas-npc-shops/atlas.com/npc/character/model.go` — un-stub `SpawnPoint()` at line 208
- `services/atlas-npc-shops/atlas.com/npc/character/rest.go` — add `x`/`y`/`stance` to `Extract` (starts line 134)
- `services/atlas-npc-shops/atlas.com/npc/character/builder.go` — add three setters
- `services/atlas-npc-shops/atlas.com/npc/character/rest_test.go` — add assertions and a builder-anchored test

Read-only references: `.../character/model.go:224` (`X() int16`), `:228` (`Y() int16`), `:232` (`Stance() byte`); `.../character/rest.go:99` (`Transform`, already emits all four fields); `.../character/rest.go:160` (`Extract` already decodes `spawnPoint` — leave that line alone).

Patterns to copy: `services/atlas-npc-shops/atlas.com/npc/character/builder.go:88-125` (the one-line `func (b *Builder) SetFoo(v T) *Builder { b.foo = v; return b }` form, gofmt-aligned as a block).

**Interfaces:**
- Produces: `func (m Model) SpawnPoint() uint32`, plus `func (b *Builder) SetX(v int16) *Builder`, `func (b *Builder) SetY(v int16) *Builder`, `func (b *Builder) SetStance(v byte) *Builder` in package `atlas-npc/character`.
- Consumes: nothing from other tasks.

- [ ] **Step 1: Write the failing tests**

In `character/rest_test.go`'s `TestTransformRoundTrip`, the fixture already carries `SpawnPoint: 11, X: 10, Y: 12, Stance: 14` (lines 40-43). Add, after the `m, err := Extract(rm)` error check:

```go
	// The DeepEqual round-trip below is blind to a field Extract drops -- it
	// is zero on both sides. Assert each fixed field against the RestModel
	// literal directly. Values are distinct so an aliased assignment fails.
	if got := m.SpawnPoint(); got != 11 {
		t.Errorf("SpawnPoint() = %d, want 11", got)
	}
	if got := m.X(); got != 10 {
		t.Errorf("X() = %d, want 10", got)
	}
	if got := m.Y(); got != 12 {
		t.Errorf("Y() = %d, want 12", got)
	}
	if got := m.Stance(); got != 14 {
		t.Errorf("Stance() = %d, want 14", got)
	}
```

Then add a builder-anchored test that exercises the three new setters on the outbound leg:

```go
// TestTransform_PositionalFieldsFromBuilder covers the SetX/SetY/SetStance
// setters added by task-272 (design 5.2, overriding PRD FR-8). They have no
// production caller by design: the Builder struct and Build already carried
// x/y/stance, and only the setters were missing, so a Model with non-zero
// positional values could not be originated through the sanctioned path.
func TestTransform_PositionalFieldsFromBuilder(t *testing.T) {
	m := NewBuilder().
		SetId(1).
		SetSp("0").
		SetSpawnPoint(11).
		SetX(10).
		SetY(12).
		SetStance(14).
		Build()

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if rm.SpawnPoint != 11 {
		t.Errorf("SpawnPoint = %d, want 11", rm.SpawnPoint)
	}
	if rm.X != 10 {
		t.Errorf("X = %d, want 10", rm.X)
	}
	if rm.Y != 12 {
		t.Errorf("Y = %d, want 12", rm.Y)
	}
	if rm.Stance != 14 {
		t.Errorf("Stance = %d, want 14", rm.Stance)
	}
}
```

- [ ] **Step 2: Run the tests and verify they fail**

Run from `services/atlas-npc-shops/atlas.com/npc`:

```
go test ./character/... -run 'TransformRoundTrip|PositionalFieldsFromBuilder' -v
```

Expected: `TestTransformRoundTrip` fails `SpawnPoint() = 0, want 11` (and `X()`/`Y()`/`Stance()` all `= 0`). `TestTransform_PositionalFieldsFromBuilder` fails to **compile** — `SetX` undefined — which is the correct red state for the setters.

- [ ] **Step 3: Un-stub the accessor**

`character/model.go:208`:

```go
func (m Model) SpawnPoint() uint32 {
	return m.spawnPoint
}
```

- [ ] **Step 4: Add the three builder setters**

`character/builder.go`, immediately after `func (b *Builder) SetGm(v int) *Builder` (line 124):

```go
func (b *Builder) SetX(v int16) *Builder                   { b.x = v; return b }
func (b *Builder) SetY(v int16) *Builder                   { b.y = v; return b }
func (b *Builder) SetStance(v byte) *Builder               { b.stance = v; return b }
```

Run `gofmt -w character/builder.go` afterwards so the alignment of the surrounding one-line setter block is correct.

- [ ] **Step 5: Decode `x`/`y`/`stance` in `Extract`**

`character/rest.go`, in `Extract` (starts line 134), after the existing `meso: rm.Meso,` line:

```go
		x:                  rm.X,
		y:                  rm.Y,
		stance:             rm.Stance,
```

Do not touch the existing `spawnPoint: rm.SpawnPoint,` line — this service already decoded it.

- [ ] **Step 6: Build and run the full module test suite**

```
go build ./... && go test ./...
```

Expected: build succeeds and all tests pass.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-npc-shops/atlas.com/npc/character/model.go \
        services/atlas-npc-shops/atlas.com/npc/character/rest.go \
        services/atlas-npc-shops/atlas.com/npc/character/builder.go \
        services/atlas-npc-shops/atlas.com/npc/character/rest_test.go
git commit -m "fix(atlas-npc-shops): return spawnPoint and decode x/y/stance"
```

---

## Task 7: atlas-consumables — accessor only

**Module root:** `services/atlas-consumables/atlas.com/consumables`

`Extract` (line 124) already decodes `spawnPoint` correctly and MUST NOT be changed. The accessor stub is the only break.

### Files

- `services/atlas-consumables/atlas.com/consumables/character/model.go` — un-stub `SpawnPoint()` at line 213
- `services/atlas-consumables/atlas.com/consumables/character/rest_test.go` — add an assertion

Read-only references: `.../character/rest.go:124` (`Extract`, already correct); `:158` (`Transform`, already correct).

**Interfaces:**
- Produces: `func (m Model) SpawnPoint() uint32` in package `atlas-consumables/character`.
- Consumes: nothing from other tasks.

- [ ] **Step 1: Write the failing test**

`character/rest_test.go`'s `TestTransformRoundTrip` already builds a `Model` literal carrying `spawnPoint: 275`. That value is above the byte ceiling, which makes it a useful uint32-fidelity witness. Add, before the `DeepEqual` check:

```go
	// 275 is above the byte ceiling on purpose: it fails both against the
	// hardcoded stub this task removes and against any narrowing reintroduced
	// inside the accessor. Narrowing belongs at the wire, and this service has
	// no wire path.
	if got.SpawnPoint() != 275 {
		t.Errorf("SpawnPoint() = %d, want 275", got.SpawnPoint())
	}
```

- [ ] **Step 2: Run the test and verify it fails**

Run from `services/atlas-consumables/atlas.com/consumables`:

```
go test ./character/... -run TransformRoundTrip -v
```

Expected: FAIL — the assertion does not compile against a `byte`-returning accessor (`275` overflows `byte` in the untyped-constant comparison), or, after the type widens, fails `SpawnPoint() = 0, want 275`. Either is a valid red state.

- [ ] **Step 3: Un-stub the accessor**

`character/model.go:213`:

```go
func (m Model) SpawnPoint() uint32 {
	return m.spawnPoint
}
```

- [ ] **Step 4: Build and run the full module test suite**

```
go build ./... && go test ./...
```

Expected: build succeeds and all tests pass. Confirm `character/rest.go` is untouched by the diff.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-consumables/atlas.com/consumables/character/model.go \
        services/atlas-consumables/atlas.com/consumables/character/rest_test.go
git commit -m "fix(atlas-consumables): return the real spawnPoint from the accessor"
```

---

## Task 8: atlas-messages — accessor only

**Module root:** `services/atlas-messages/atlas.com/messages`

`Extract` (line 163) already decodes `spawnPoint` correctly and MUST NOT be changed. The accessor stub is the only break.

**Out of scope:** this service has a second stub, `Stance() byte { return 0 }` at `character/model.go:225`, despite its `Extract` decoding `stance`. Same shape, different field, explicitly outside this task (PRD §9 item 3). Leave it.

### Files

- `services/atlas-messages/atlas.com/messages/character/model.go` — un-stub `SpawnPoint()` at line 205
- `services/atlas-messages/atlas.com/messages/character/rest_test.go` — add an assertion

Read-only references: `.../character/rest.go:163` (`Extract`, already correct); `:129` (`Transform`, already correct); `.../character/model.go:225` (`Stance()` stub — do not touch).

**Interfaces:**
- Produces: `func (m Model) SpawnPoint() uint32` in package `atlas-messages/character`.
- Consumes: nothing from other tasks.

- [ ] **Step 1: Write the failing test**

In `character/rest_test.go`'s `TestTransformRoundTrip` (line 374), the fixture already carries `SpawnPoint: 11` (line 401). Add, after the `m, err := Extract(rm)` error check:

```go
	// The DeepEqual round-trip below cannot see a masked accessor: Extract
	// decodes spawnPoint correctly here, and it is the accessor that lies.
	// Assert the decoded value against the RestModel literal directly.
	if got := m.SpawnPoint(); got != 11 {
		t.Errorf("SpawnPoint() = %d, want 11", got)
	}
```

Leave `TestExtract` (line 162) and `TestExtract_ZeroValues` (line 235) alone; their `SpawnPoint: 0` fixtures are fine as-is because they are not the assertion carrying this fix.

- [ ] **Step 2: Run the test and verify it fails**

Run from `services/atlas-messages/atlas.com/messages`:

```
go test ./character/... -run TransformRoundTrip -v
```

Expected: FAIL with `SpawnPoint() = 0, want 11`.

- [ ] **Step 3: Un-stub the accessor**

`character/model.go:205`:

```go
func (m Model) SpawnPoint() uint32 {
	return m.spawnPoint
}
```

- [ ] **Step 4: Build and run the full module test suite**

```
go build ./... && go test ./...
```

Expected: build succeeds and all tests pass. Confirm `character/rest.go` and the `Stance()` stub are untouched by the diff.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-messages/atlas.com/messages/character/model.go \
        services/atlas-messages/atlas.com/messages/character/rest_test.go
git commit -m "fix(atlas-messages): return the real spawnPoint from the accessor"
```

---

## Task 9: Acceptance sweep and the repo-wide gate

No production code changes. This task proves the PRD's acceptance criteria mechanically and is the only place `tools/verify.sh` runs.

### Files

- No files are modified. This task produces evidence only.

**Interfaces:**
- Consumes: the eight `func (m Model) SpawnPoint() uint32` implementations from Tasks 1-8.
- Produces: nothing.

- [ ] **Step 1: Confirm no stubbed accessor survives**

```
grep -rn "func (m Model) SpawnPoint()" services/ libs/
```

Expected: exactly nine hits (eight fixed services plus `atlas-character`), every one returning `uint32`. Then:

```
grep -rn -A2 "func (m Model) SpawnPoint()" services/ | grep "return 0"
```

Expected: **no output**. Any hit is a failure.

- [ ] **Step 2: Confirm the untouchable surfaces are untouched**

```
git diff --stat b284bcebf -- services/atlas-character libs/atlas-packet
```

Expected: **no output** — zero files changed under either path. This is the mechanical half of the FR-9 byte-identity argument (design §6): the encoder and its inputs other than `Stats.SpawnPoint` provably did not move.

- [ ] **Step 3: Confirm the four unchanged `Extract`s stayed unchanged**

```
git diff b284bcebf -- services/atlas-consumables/atlas.com/consumables/character/rest.go \
                      services/atlas-messages/atlas.com/messages/character/rest.go \
                      services/atlas-query-aggregator/atlas.com/query-aggregator/character/rest.go
```

Expected: the only hunk anywhere in this output is the `SpawnPoint` cast removal in `atlas-query-aggregator`'s `Transform`. No `Extract` body appears.

- [ ] **Step 4: Confirm the two wire casts are present**

```
grep -n "SpawnPoint" services/atlas-channel/atlas.com/channel/socket/writer/character_data.go \
                     services/atlas-login/atlas.com/login/socket/writer/character_list.go
```

Expected: both show an explicit `byte(c.SpawnPoint())`.

- [ ] **Step 5: Run the repo-wide gate**

```
tools/verify.sh
```

Flagless — `--quick` and `--no-docker` do not count for this task. Expected: exit 0.

- [ ] **Step 6: Report**

Record the exit status and the grep outputs. If `tools/verify.sh` fails, report the first failing block verbatim; do not summarize it. There is nothing to commit in this task.
