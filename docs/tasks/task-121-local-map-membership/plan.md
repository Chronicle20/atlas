# Local Map Membership for Broadcasts (PS-2) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace every REST call to atlas-maps on atlas-channel's broadcast recipient-resolution path with a filter over the local in-process session registry.

**Architecture:** New field-filtering providers (`InFieldModelProvider`, `InMapAllInstancesModelProvider`) are added to `session.Processor`; the `map.Processor` recipient providers are re-implemented over them with every exported signature frozen, so all 32 caller files compile unchanged. The now-dead REST plumbing in the `map` package is deleted. One prerequisite gap is fixed: login bootstrap sets only the map id and drops the instance (`SetMapId` → `SetField`).

**Tech Stack:** Go, `atlas-model` provider/operator pipeline, `atlas-constants` field/world/channel/map types, in-memory session registry (`sync.RWMutex`).

**Spec:** `docs/tasks/task-121-local-map-membership/design.md` (PRD: `prd.md` in the same folder).

## Global Constraints

- All Go work happens in module `services/atlas-channel/atlas.com/channel` (module name `atlas-channel`) inside the task worktree `.worktrees/task-121-local-map-membership`. Never edit the main repo checkout.
- **Frozen signatures:** every exported method of `map.Processor` (`CharacterIdsInMapModelProvider`, `GetCharacterIdsInMap`, `ForSessionsInSessionsMap`, `ForSessionsInMap`, `CharacterIdsInMapAllInstancesModelProvider`, `ForSessionsInMapAllInstances`, `NotCharacterIdFilter`, `OtherCharacterIdsInMapModelProvider`, `ForOtherSessionsInMap`) keeps its exact current signature (design §3.2, PRD §5).
- No changes to atlas-maps, to atlas-channel's `MAP_STATUS` consumer, or to map ENTER/EXIT command emission (PRD FR-3.3).
- No new goroutines, no new locks: all registry access via the existing `GetInTenant` RLock snapshot (PRD NFR-5).
- Tests use the existing pattern in `session/processor_test.go`: `session.NewSession(id, tenant, 0, nil)` + `session.AddSessionToRegistry` + `session.Processor` mutators + `session.ClearRegistryForTenant` cleanup. No `*_testhelpers.go` files (PRD FR-4.3).
- No `// TODO`, stubs, or dead code left in any commit. Dedup of character ids is mandatory in the new id providers (design §3.2).
- Committed docs use repo-relative paths only — never `/home/<name>/...`.
- Before claiming done: `go test -race ./...`, `go vet ./...`, `go build ./...` clean in the module; `docker buildx bake atlas-channel` and `tools/redis-key-guard.sh` clean from the worktree root (CLAUDE.md).

**Working directory for all `go` commands:** `services/atlas-channel/atlas.com/channel` (inside the worktree). **Working directory for bake/guard:** the worktree root.

---

### Task 1: Session field-filtering providers

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/session/processor.go` (add two methods after `AllInChannelProvider`, ~line 64)
- Test: `services/atlas-channel/atlas.com/channel/session/processor_test.go` (append)

**Interfaces:**
- Consumes: `getRegistry().GetInTenant(p.t.Id())` (existing, RLock snapshot returning `[]Model` value copies), `field.Model.Equals` (`libs/atlas-constants/field/model.go:92`, compares world/channel/map/instance), session accessors `CharacterId()`, `WorldId()`, `ChannelId()`, `MapId()`, `Field()`.
- Produces (Task 3 relies on these exact signatures):
  - `func (p *Processor) InFieldModelProvider(f field.Model) model.Provider[[]Model]`
  - `func (p *Processor) InMapAllInstancesModelProvider(worldId world.Id, channelId channel.Id, mapId _map.Id) model.Provider[[]Model]`

- [ ] **Step 1: Write the failing tests**

Append to `services/atlas-channel/atlas.com/channel/session/processor_test.go`. The file already imports `session`, `test`, `_map`, `uuid`; add `"github.com/Chronicle20/atlas/libs/atlas-constants/field"` to its import block.

```go
// addFieldSession registers a session in the default tenant's registry with the
// given character id (0 = no character assigned) and field, using only public API.
func addFieldSession(t *testing.T, p *session.Processor, characterId uint32, f field.Model) uuid.UUID {
	t.Helper()
	sessionId := uuid.New()
	ten := test.CreateDefaultMockTenant()
	s := session.NewSession(sessionId, ten, 0, nil)
	session.AddSessionToRegistry(ten.Id(), s)
	if characterId != 0 {
		p.SetCharacterId(sessionId, characterId)
	}
	p.SetField(sessionId, f)
	return sessionId
}

func characterIdSet(ms []session.Model) map[uint32]bool {
	r := make(map[uint32]bool)
	for _, m := range ms {
		r[m.CharacterId()] = true
	}
	return r
}

func TestInFieldModelProvider_ExactMatchIncludingInstance(t *testing.T) {
	logger, cleanup := testSetup()
	defer cleanup()
	ctx := test.CreateTestContext()
	p := session.NewProcessor(logger, ctx)

	instA := uuid.New()
	instB := uuid.New()
	fA := field.NewBuilder(0, 0, _map.Id(100000000)).SetInstance(instA).Build()
	fB := field.NewBuilder(0, 0, _map.Id(100000000)).SetInstance(instB).Build()
	addFieldSession(t, p, 100, fA)
	addFieldSession(t, p, 200, fB)

	gotA, err := p.InFieldModelProvider(fA)()
	if err != nil {
		t.Fatalf("InFieldModelProvider(fA) unexpected error: %v", err)
	}
	if len(gotA) != 1 || !characterIdSet(gotA)[100] {
		t.Errorf("InFieldModelProvider(fA) = chars %v, want exactly {100}", characterIdSet(gotA))
	}

	gotB, err := p.InFieldModelProvider(fB)()
	if err != nil {
		t.Fatalf("InFieldModelProvider(fB) unexpected error: %v", err)
	}
	if len(gotB) != 1 || !characterIdSet(gotB)[200] {
		t.Errorf("InFieldModelProvider(fB) = chars %v, want exactly {200}", characterIdSet(gotB))
	}
}

func TestInFieldModelProvider_WorldChannelDiscrimination(t *testing.T) {
	logger, cleanup := testSetup()
	defer cleanup()
	ctx := test.CreateTestContext()
	p := session.NewProcessor(logger, ctx)

	// Sessions created via NewSession sit at world 0 / channel 0.
	f := field.NewBuilder(0, 0, _map.Id(100000000)).Build()
	addFieldSession(t, p, 100, f)

	otherWorld := field.NewBuilder(1, 0, _map.Id(100000000)).Build()
	got, err := p.InFieldModelProvider(otherWorld)()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("InFieldModelProvider(world 1) = chars %v, want empty", characterIdSet(got))
	}

	otherChannel := field.NewBuilder(0, 1, _map.Id(100000000)).Build()
	got, err = p.InFieldModelProvider(otherChannel)()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("InFieldModelProvider(channel 1) = chars %v, want empty", characterIdSet(got))
	}
}

func TestInFieldModelProvider_ExcludesCharacterlessSessions(t *testing.T) {
	logger, cleanup := testSetup()
	defer cleanup()
	ctx := test.CreateTestContext()
	p := session.NewProcessor(logger, ctx)

	f := field.NewBuilder(0, 0, _map.Id(100000000)).Build()
	addFieldSession(t, p, 100, f)
	addFieldSession(t, p, 0, f) // pre-login session, no character

	got, err := p.InFieldModelProvider(f)()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || !characterIdSet(got)[100] {
		t.Errorf("InFieldModelProvider = chars %v, want exactly {100}", characterIdSet(got))
	}
}

func TestInFieldModelProvider_ExcludesOtherTenant(t *testing.T) {
	logger, cleanup := testSetup()
	defer cleanup()
	otherTenantId := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	defer session.ClearRegistryForTenant(otherTenantId)

	ctx := test.CreateTestContext()
	p := session.NewProcessor(logger, ctx)
	f := field.NewBuilder(0, 0, _map.Id(100000000)).Build()
	addFieldSession(t, p, 100, f)

	otherCtx := test.CreateTestContextWithTenant(otherTenantId)
	po := session.NewProcessor(logger, otherCtx)
	otherSessionId := uuid.New()
	os := session.NewSession(otherSessionId, test.CreateDefaultMockTenant(), 0, nil)
	session.AddSessionToRegistry(otherTenantId, os)
	po.SetCharacterId(otherSessionId, 999)
	po.SetField(otherSessionId, f)

	got, err := p.InFieldModelProvider(f)()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || !characterIdSet(got)[100] {
		t.Errorf("InFieldModelProvider = chars %v, want exactly {100} (no cross-tenant leakage)", characterIdSet(got))
	}
}

func TestInFieldModelProvider_EmptyFieldNoError(t *testing.T) {
	logger, cleanup := testSetup()
	defer cleanup()
	ctx := test.CreateTestContext()
	p := session.NewProcessor(logger, ctx)

	f := field.NewBuilder(0, 0, _map.Id(999999999)).Build()
	got, err := p.InFieldModelProvider(f)()
	if err != nil {
		t.Fatalf("unexpected error for unpopulated field: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("InFieldModelProvider(unpopulated) = %d sessions, want 0", len(got))
	}
}

func TestInMapAllInstancesModelProvider_UnionsInstances(t *testing.T) {
	logger, cleanup := testSetup()
	defer cleanup()
	ctx := test.CreateTestContext()
	p := session.NewProcessor(logger, ctx)

	fNil := field.NewBuilder(0, 0, _map.Id(100000000)).Build()
	fInst := field.NewBuilder(0, 0, _map.Id(100000000)).SetInstance(uuid.New()).Build()
	fOtherMap := field.NewBuilder(0, 0, _map.Id(200000000)).Build()
	addFieldSession(t, p, 100, fNil)
	addFieldSession(t, p, 200, fInst)
	addFieldSession(t, p, 300, fOtherMap)
	addFieldSession(t, p, 0, fNil) // characterless, excluded

	got, err := p.InMapAllInstancesModelProvider(0, 0, _map.Id(100000000))()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	set := characterIdSet(got)
	if len(got) != 2 || !set[100] || !set[200] {
		t.Errorf("InMapAllInstancesModelProvider = chars %v, want exactly {100, 200}", set)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run from `services/atlas-channel/atlas.com/channel`:
```bash
go test ./session/... -run 'TestInField|TestInMapAllInstances' -v
```
Expected: compile FAILURE with `p.InFieldModelProvider undefined` / `p.InMapAllInstancesModelProvider undefined`.

- [ ] **Step 3: Implement the providers**

In `services/atlas-channel/atlas.com/channel/session/processor.go`, immediately after `AllInChannelProvider` (ends ~line 64), add:

```go
// InFieldModelProvider returns local sessions whose field exactly matches f
// (world, channel, map, instance) and which have an assigned character.
func (p *Processor) InFieldModelProvider(f field.Model) model.Provider[[]Model] {
	return func() ([]Model, error) {
		all := getRegistry().GetInTenant(p.t.Id())
		result := make([]Model, 0)
		for _, s := range all {
			if s.CharacterId() != 0 && s.Field().Equals(f) {
				result = append(result, s)
			}
		}
		return result, nil
	}
}

// InMapAllInstancesModelProvider returns local sessions on the given
// world/channel/map across all instances, with an assigned character.
func (p *Processor) InMapAllInstancesModelProvider(worldId world.Id, channelId channel.Id, mapId _map.Id) model.Provider[[]Model] {
	return func() ([]Model, error) {
		all := getRegistry().GetInTenant(p.t.Id())
		result := make([]Model, 0)
		for _, s := range all {
			if s.CharacterId() != 0 && s.WorldId() == worldId && s.ChannelId() == channelId && s.MapId() == mapId {
				result = append(result, s)
			}
		}
		return result, nil
	}
}
```

All needed imports (`field`, `world`, `channel`, `_map`, `model`) are already present in the file. Filtering happens on the snapshot returned by `GetInTenant` — outside any lock, per design §3.1.

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test -race ./session/... -v
```
Expected: all PASS (new tests plus the existing suite).

- [ ] **Step 5: Commit**

```bash
git add session/processor.go session/processor_test.go
git commit -m "feat(channel): add field-filtering session providers for local map membership"
```

---

### Task 2: Login-bootstrap instance fix (FR-1.2) and SetMapId retirement

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/session/consumer.go:190` (one line)
- Modify: `services/atlas-channel/atlas.com/channel/session/processor.go:226-235` (delete `SetMapId`)
- Test: `services/atlas-channel/atlas.com/channel/session/processor_test.go` (replace `TestSetMapId` at :207 and `TestSetMapId_NonExistent` at :607)

**Interfaces:**
- Consumes: `session.Processor.SetField(id uuid.UUID, f field.Model) Model` (existing, sets map id + instance).
- Produces: `Processor.SetMapId` no longer exists; all field mutation after session creation goes through `SetField`. Login bootstrap preserves the instance from `location.GetField`.

Background (design §4.1): a character logging in while located in an instanced map currently gets a session with `instance == uuid.Nil` because bootstrap calls `SetMapId(s.SessionId(), f.MapId())`, dropping `f.Instance()`. Under local exact-field resolution this would become a delivery regression, so it must be fixed first. After the fix, `Processor.SetMapId` has zero non-test callers (verified by grep during planning) and is deleted — the internal `Model.setMapId` stays (used by `SetField`).

- [ ] **Step 1: Replace the SetMapId tests with SetField tests**

In `services/atlas-channel/atlas.com/channel/session/processor_test.go`, delete `TestSetMapId` (lines 207-234, the whole function) and `TestSetMapId_NonExistent` (lines 607-620, the whole function; line numbers may have shifted after Task 1 — locate by function name). Add in their place:

```go
func TestSetField_PreservesInstance(t *testing.T) {
	logger, cleanup := testSetup()
	defer cleanup()

	ctx := test.CreateTestContext()
	sessionId := uuid.New()
	tenant := test.CreateDefaultMockTenant()

	s := session.NewSession(sessionId, tenant, 0, nil)
	session.AddSessionToRegistry(tenant.Id(), s)

	p := session.NewProcessor(logger, ctx)
	inst := uuid.New()
	f := field.NewBuilder(0, 0, _map.Id(100000000)).SetInstance(inst).Build()
	updated := p.SetField(sessionId, f)

	if !updated.Field().Equals(f) {
		t.Errorf("SetField() field = %v/%v/%v/%v, want it to equal f (map 100000000, instance %s)",
			updated.WorldId(), updated.ChannelId(), updated.MapId(), updated.Instance(), inst)
	}

	retrieved, err := p.ByIdModelProvider(sessionId)()
	if err != nil {
		t.Fatalf("ByIdModelProvider() unexpected error: %v", err)
	}
	if !retrieved.Field().Equals(f) {
		t.Errorf("registry session field does not equal f; instance = %s, want %s", retrieved.Instance(), inst)
	}
}

func TestSetField_NonExistent(t *testing.T) {
	logger, cleanup := testSetup()
	defer cleanup()

	ctx := test.CreateTestContext()
	p := session.NewProcessor(logger, ctx)
	nonExistentId := uuid.New()

	f := field.NewBuilder(0, 0, _map.Id(100000000)).Build()
	result := p.SetField(nonExistentId, f)

	if result.SessionId() != uuid.Nil {
		t.Errorf("SetField() for non-existent session returned non-zero SessionId")
	}
}
```

- [ ] **Step 2: Run tests — new ones pass, suite still green**

```bash
go test ./session/... -run 'TestSetField' -v
```
Expected: PASS (SetField already preserves the instance; these tests pin the behavior the bootstrap fix depends on).

- [ ] **Step 3: Fix the bootstrap call site**

In `services/atlas-channel/atlas.com/channel/kafka/consumer/session/consumer.go:190`, change:

```go
s = sp.SetMapId(s.SessionId(), f.MapId())
```
to:
```go
s = sp.SetField(s.SessionId(), f)
```

(`f` is the `field.Model` from `location.GetField` a few lines above and carries the instance.)

- [ ] **Step 4: Delete the now-unused mutator**

In `services/atlas-channel/atlas.com/channel/session/processor.go`, delete the whole `SetMapId` method (lines 226-235 pre-Task-1; locate by name):

```go
func (p *Processor) SetMapId(id uuid.UUID, mapId _map.Id) Model {
	...
}
```

- [ ] **Step 5: Verify no references remain and build/test**

```bash
grep -rn "SetMapId" --include='*.go' .
```
Expected: exactly one non-test hit — `session/model.go:154` (the field **Builder's** `.SetMapId(id)` call inside the internal `Model.setMapId`, which stays because `SetField` uses it). No hit in `session/processor.go`, `kafka/`, or any test file.

```bash
go build ./... && go test -race ./session/... ./kafka/... 
```
Expected: build clean, tests PASS.

- [ ] **Step 6: Commit**

```bash
git add session/processor.go session/processor_test.go kafka/consumer/session/consumer.go
git commit -m "fix(channel): preserve instance on login bootstrap via SetField; retire SetMapId"
```

---

### Task 3: Swap map recipient providers to local resolution

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/map/processor.go:31-33,49-51` (two method bodies + new helper + import removal)
- Test: Create `services/atlas-channel/atlas.com/channel/map/processor_test.go`

**Interfaces:**
- Consumes (from Task 1): `session.Processor.InFieldModelProvider(f field.Model) model.Provider[[]session.Model]`, `session.Processor.InMapAllInstancesModelProvider(worldId world.Id, channelId channel.Id, mapId _map.Id) model.Provider[[]session.Model]`.
- Produces: `CharacterIdsInMapModelProvider` / `CharacterIdsInMapAllInstancesModelProvider` with unchanged signatures, now returning deduplicated character ids of local sessions, no REST. Package-private helper `characterIds(sp model.Provider[[]session.Model]) model.Provider[[]uint32]`.

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-channel/atlas.com/channel/map/processor_test.go`:

```go
package _map_test

import (
	_map "atlas-channel/map"
	"atlas-channel/session"
	"atlas-channel/test"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	mapid "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

func mapTestSetup() (*logrus.Logger, func()) {
	logger, _ := logtest.NewNullLogger()
	cleanup := func() {
		session.ClearRegistryForTenant(test.DefaultTenantId)
	}
	return logger, cleanup
}

// addFieldSession registers a session in the default tenant's registry with the
// given character id and field, using only public API. Mirrors the helper in
// session/processor_test.go (test packages cannot share unexported helpers).
func addFieldSession(t *testing.T, p *session.Processor, characterId uint32, f field.Model) uuid.UUID {
	t.Helper()
	sessionId := uuid.New()
	ten := test.CreateDefaultMockTenant()
	s := session.NewSession(sessionId, ten, 0, nil)
	session.AddSessionToRegistry(ten.Id(), s)
	if characterId != 0 {
		p.SetCharacterId(sessionId, characterId)
	}
	p.SetField(sessionId, f)
	return sessionId
}

func idSet(ids []uint32) map[uint32]bool {
	r := make(map[uint32]bool)
	for _, id := range ids {
		r[id] = true
	}
	return r
}

// Regression proof that recipient resolution no longer performs REST: no MAPS
// service URL is configured in the test environment, so any HTTP attempt errors.
func TestGetCharacterIdsInMap_LocalResolutionNoHTTP(t *testing.T) {
	logger, cleanup := mapTestSetup()
	defer cleanup()
	ctx := test.CreateTestContext()
	sp := session.NewProcessor(logger, ctx)
	p := _map.NewProcessor(logger, ctx)

	f := field.NewBuilder(0, 0, mapid.Id(100000000)).Build()
	addFieldSession(t, sp, 100, f)
	addFieldSession(t, sp, 200, f)

	ids, err := p.GetCharacterIdsInMap(f)
	if err != nil {
		t.Fatalf("GetCharacterIdsInMap() unexpected error (REST still in the path?): %v", err)
	}
	set := idSet(ids)
	if len(ids) != 2 || !set[100] || !set[200] {
		t.Errorf("GetCharacterIdsInMap() = %v, want exactly {100, 200}", ids)
	}
}

func TestCharacterIdsInMapModelProvider_DedupsCharacterIds(t *testing.T) {
	logger, cleanup := mapTestSetup()
	defer cleanup()
	ctx := test.CreateTestContext()
	sp := session.NewProcessor(logger, ctx)
	p := _map.NewProcessor(logger, ctx)

	f := field.NewBuilder(0, 0, mapid.Id(100000000)).Build()
	// Stale socket + reconnect: two registry sessions carrying the same character id.
	addFieldSession(t, sp, 100, f)
	addFieldSession(t, sp, 100, f)

	ids, err := p.CharacterIdsInMapModelProvider(f)()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 1 || ids[0] != 100 {
		t.Errorf("CharacterIdsInMapModelProvider() = %v, want exactly [100]", ids)
	}
}

func TestOtherCharacterIdsInMapModelProvider_ExcludesReference(t *testing.T) {
	logger, cleanup := mapTestSetup()
	defer cleanup()
	ctx := test.CreateTestContext()
	sp := session.NewProcessor(logger, ctx)
	p := _map.NewProcessor(logger, ctx)

	f := field.NewBuilder(0, 0, mapid.Id(100000000)).Build()
	addFieldSession(t, sp, 100, f)
	addFieldSession(t, sp, 200, f)

	ids, err := p.OtherCharacterIdsInMapModelProvider(f, 100)()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 1 || ids[0] != 200 {
		t.Errorf("OtherCharacterIdsInMapModelProvider(f, 100) = %v, want exactly [200]", ids)
	}
}

func TestCharacterIdsInMapAllInstancesModelProvider_UnionsInstances(t *testing.T) {
	logger, cleanup := mapTestSetup()
	defer cleanup()
	ctx := test.CreateTestContext()
	sp := session.NewProcessor(logger, ctx)
	p := _map.NewProcessor(logger, ctx)

	fNil := field.NewBuilder(0, 0, mapid.Id(100000000)).Build()
	fInst := field.NewBuilder(0, 0, mapid.Id(100000000)).SetInstance(uuid.New()).Build()
	addFieldSession(t, sp, 100, fNil)
	addFieldSession(t, sp, 200, fInst)

	ids, err := p.CharacterIdsInMapAllInstancesModelProvider(0, 0, mapid.Id(100000000))()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	set := idSet(ids)
	if len(ids) != 2 || !set[100] || !set[200] {
		t.Errorf("CharacterIdsInMapAllInstancesModelProvider() = %v, want exactly {100, 200}", ids)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./map/... -v
```
Expected: FAIL — `GetCharacterIdsInMap`/provider calls error attempting HTTP (no `MAPS` root URL configured in test env), proving the REST dependency the swap removes.

- [ ] **Step 3: Re-implement the two providers**

In `services/atlas-channel/atlas.com/channel/map/processor.go`:

Replace the body of `CharacterIdsInMapModelProvider` (line 31):

```go
func (p *Processor) CharacterIdsInMapModelProvider(f field.Model) model.Provider[[]uint32] {
	return characterIds(p.sp.InFieldModelProvider(f))
}
```

Replace the body of `CharacterIdsInMapAllInstancesModelProvider` (line 49):

```go
func (p *Processor) CharacterIdsInMapAllInstancesModelProvider(worldId world.Id, channelId channel.Id, mapId _map.Id) model.Provider[[]uint32] {
	return characterIds(p.sp.InMapAllInstancesModelProvider(worldId, channelId, mapId))
}
```

Add the package-private helper (dedup rationale in design §3.2 — the registry can transiently hold two sessions for one character id; without dedup the delivery operator runs twice against the same first-found session):

```go
// characterIds maps sessions to their character ids, deduplicated — the
// registry can transiently hold two sessions for one character (stale socket
// plus reconnect) and each character must be delivered to at most once.
func characterIds(sp model.Provider[[]session.Model]) model.Provider[[]uint32] {
	return func() ([]uint32, error) {
		ss, err := sp()
		if err != nil {
			return nil, err
		}
		seen := make(map[uint32]struct{}, len(ss))
		ids := make([]uint32, 0, len(ss))
		for _, s := range ss {
			id := s.CharacterId()
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
		return ids, nil
	}
}
```

Remove `"github.com/Chronicle20/atlas/libs/atlas-rest/requests"` from the import block (now unused; the compiler enforces this). Every other method in the file (`GetCharacterIdsInMap`, `ForSessionsInSessionsMap`, `ForSessionsInMap`, `ForSessionsInMapAllInstances`, `NotCharacterIdFilter`, `OtherCharacterIdsInMapModelProvider`, `ForOtherSessionsInMap`) is untouched — they compose over the two providers (FR-2.3).

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test -race ./map/... ./session/... -v
```
Expected: all PASS.

- [ ] **Step 5: Full-module regression check (frozen signatures)**

```bash
go build ./... && go test -race ./...
```
Expected: clean — the 32 caller files (kafka consumers, movement, skill handlers, socket handlers) compile against unchanged signatures and existing suites (`kafka/consumer/{door,mist,monster,mount}`, movement, skill handlers) pass unchanged.

- [ ] **Step 6: Commit**

```bash
git add map/processor.go map/processor_test.go
git commit -m "feat(channel): resolve broadcast recipients from local session registry (PS-2)"
```

---

### Task 4: Map-transition correctness test (FR-4.2)

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/map/processor_test.go` (append)

**Interfaces:**
- Consumes: everything from Tasks 1-3; `session.Processor.SetField`.
- Produces: a pinned invariant — a warped session is in exactly one map's recipient set at every observable point (registry `Update` replaces the single entry under one write lock, so the invariant is structural; this test guards it against regression).

- [ ] **Step 1: Write the transition test**

Append to `services/atlas-channel/atlas.com/channel/map/processor_test.go`:

```go
// FR-4.2: a session warped from map A to map B stops receiving A-broadcasts and
// starts receiving B-broadcasts, with no state in which it is in both or neither.
func TestTransition_WarpMovesRecipientSetAtomically(t *testing.T) {
	logger, cleanup := mapTestSetup()
	defer cleanup()
	ctx := test.CreateTestContext()
	sp := session.NewProcessor(logger, ctx)
	p := _map.NewProcessor(logger, ctx)

	fA := field.NewBuilder(0, 0, mapid.Id(100000000)).Build()
	fB := field.NewBuilder(0, 0, mapid.Id(200000000)).Build()
	addFieldSession(t, sp, 100, fA) // stays in A
	bId := addFieldSession(t, sp, 200, fA)

	// Before the warp: both in A, none in B.
	idsA, err := p.GetCharacterIdsInMap(fA)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	setA := idSet(idsA)
	if len(idsA) != 2 || !setA[100] || !setA[200] {
		t.Fatalf("pre-warp map A = %v, want exactly {100, 200}", idsA)
	}
	idsB, err := p.GetCharacterIdsInMap(fB)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(idsB) != 0 {
		t.Fatalf("pre-warp map B = %v, want empty", idsB)
	}

	// Warp B's session — same call the MAP_CHANGED consumer makes
	// (kafka/consumer/character/consumer.go, SetField before dependent broadcasts).
	sp.SetField(bId, fB)

	idsA, err = p.GetCharacterIdsInMap(fA)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(idsA) != 1 || idsA[0] != 100 {
		t.Errorf("post-warp map A = %v, want exactly [100]", idsA)
	}
	idsB, err = p.GetCharacterIdsInMap(fB)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(idsB) != 1 || idsB[0] != 200 {
		t.Errorf("post-warp map B = %v, want exactly [200]", idsB)
	}
}
```

- [ ] **Step 2: Run the test**

```bash
go test -race ./map/... -run TestTransition -v
```
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add map/processor_test.go
git commit -m "test(channel): pin map-transition recipient-set correctness (FR-4.2)"
```

---

### Task 5: Delete the REST plumbing (FR-3.2)

**Files:**
- Delete: `services/atlas-channel/atlas.com/channel/map/requests.go`
- Delete: `services/atlas-channel/atlas.com/channel/map/rest.go`

**Interfaces:**
- Consumes: nothing — Task 3 removed the last references (`requests.SliceProvider` calls and the `requests` import).
- Produces: `atlas-channel/map` package contains only `processor.go` + `processor_test.go`; no HTTP client code remains.

- [ ] **Step 1: Confirm the files are unreferenced, then delete**

Run from `services/atlas-channel/atlas.com/channel`:
```bash
grep -rn "requestCharactersInMap\|_map.RestModel\|_map.Extract" --include='*.go' . 
```
Expected: no output.

```bash
git rm map/requests.go map/rest.go
```

- [ ] **Step 2: Build and test the whole module**

```bash
go build ./... && go test -race ./... && go vet ./...
```
Expected: all clean. (This is the "hidden consumer" mitigation from design §9 — deletion compiles the whole service.)

- [ ] **Step 3: Verify the acceptance grep**

```bash
grep -rn "requestCharactersInMap" --include='*.go' .
grep -rn "requests\." map/
```
Expected: both return nothing (PRD acceptance: no `requests.SliceProvider`/HTTP usage remains in `atlas-channel/map`).

- [ ] **Step 4: Commit**

```bash
git add -A map/
git commit -m "chore(channel): delete atlas-maps REST plumbing from map recipient resolution"
```

---

### Task 6: FR-1.3 audit document, caller inventory, and full verification

**Files:**
- Create: `docs/tasks/task-121-local-map-membership/field-transition-audit.md` (at the worktree root, repo-relative)

**Interfaces:**
- Consumes: the finished code from Tasks 1-5.
- Produces: the audit artifact required by PRD FR-1.3 and design §8, plus the full CLAUDE.md verification gate.

- [ ] **Step 1: Re-verify the transition-path table against final code**

Run from `services/atlas-channel/atlas.com/channel` and record the actual line numbers the greps report:

```bash
grep -n "setWorldId\|setChannelId" session/processor.go        # expect: only inside Create
grep -rn "SetField(" --include='*.go' kafka/ session/ | grep -v _test
grep -rn "SetMapId" --include='*.go' . | grep -v setMapId       # expect: no output
```

- [ ] **Step 2: Write the audit document**

Create `docs/tasks/task-121-local-map-membership/field-transition-audit.md` with the design §4 table, substituting the line numbers observed in Step 1 (the values below are pre-implementation and WILL have shifted — verify each):

```markdown
# task-121 — Session-Field Write-Path Audit (FR-1.3)

Every code path that changes a session's world/channel/map/instance, and where
it updates the session registry. Grep basis: all call sites of the registry
field mutators (`setWorldId`, `setChannelId`, `setMapId`, `setInstance`,
`SetField`) in `services/atlas-channel/atlas.com/channel`.

| # | Transition | Site | Registry update | Verdict |
|---|-----------|------|-----------------|---------|
| 1 | Socket connect | `session/processor.go:<line>` `Create` — `setWorldId` + `setChannelId` before `Add` | world/channel fixed at creation for the session's lifetime | ✅ |
| 2 | Login spawn-in | `kafka/consumer/session/consumer.go:<line>` — `SetField(s.SessionId(), f)` with `f` from `location.GetField` (includes instance) | full field set before `SessionCreated`/SetField packet/`SpawnForSelf` | ✅ fixed in this task (was `SetMapId`, dropping the instance) |
| 3 | Every map/instance change (portal warp, GM warp, revive/forced return, transport arrival, instance enter/exit) | `kafka/consumer/character/consumer.go:<line>` — `SetField(sessionId, targetField)` from the `MAP_CHANGED` status event, before the warp packet and `SpawnForSelf` | full field set before dependent broadcasts | ✅ |
| — | Channel change | `kafka/consumer/session/consumer.go:<lines>` — old session destroyed; client reconnects to the target channel pod, which runs paths 1+2 fresh | no in-place field mutation exists | ✅ by construction |

All transition kinds enumerated in PRD FR-1.1 are server-driven through the
single `MAP_CHANGED` status event (path 3); atlas-channel has no other
field-writing entry point.

## Caller inventory (FR-3.1)

Non-test files calling the `map.Processor` recipient providers, all confirmed
to use the result only to address local sessions or reason about this pod's
map population (design §5 categories 1-3):

<paste the output of the command below>
```

Generate the caller list and paste it into the document (paths are already repo-relative when run from the module root — prefix each with `services/atlas-channel/atlas.com/channel/`):

```bash
grep -rln "ForSessionsInMap\|ForOtherSessionsInMap\|CharacterIdsInMap\|GetCharacterIdsInMap\|ForSessionsInSessionsMap" --include='*.go' . | grep -v _test | sort
```
Expected: 32 files (count observed during planning; the list includes `./map/processor.go` itself, which defines the providers — keep it in the doc with a note, or exclude it and state the caller count as 31; re-verify either way).

- [ ] **Step 3: Full verification gate (CLAUDE.md)**

From `services/atlas-channel/atlas.com/channel`:
```bash
go test -race ./... && go vet ./... && go build ./...
```
Expected: all clean.

From the worktree root (`.worktrees/task-121-local-map-membership`):
```bash
docker buildx bake atlas-channel
tools/redis-key-guard.sh
```
Expected: bake succeeds; guard clean.

- [ ] **Step 4: Commit**

```bash
git add docs/tasks/task-121-local-map-membership/field-transition-audit.md
git commit -m "docs(task-121): field-transition audit and caller inventory (FR-1.3)"
```

---

## Out of scope (do not do)

- No field-keyed index in the registry (design §2.1 rejected alternative B; the provider seam allows adding one later inside the `session` package with zero caller churn).
- No shadow verification mode (design §2.3).
- No collapsing of the id→session delivery indirection in `ForEachByCharacterId` (design §3.2 "deliberately retained indirection").
- No changes to atlas-maps, the `MAP_STATUS` consumer, or ENTER/EXIT command emission.
- Playtest on a live tenant (PRD acceptance, last item) happens at review/PR time, not as a plan task — it needs a deployed build.
