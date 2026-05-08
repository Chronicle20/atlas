# Use-Item Server Latency Cleanup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Cut the server-side latency budget for a use-item flow by removing redundant party-announce work, short-circuiting saga lookups for nil-UUID events, and parallelising independent reads in `consumable.ConsumeStandard` (and structurally similar variants).

**Architecture:** Three independent service-level cleanups plus one supporting library primitive. `libs/atlas-model` gains a typed `Group`/`Submit`/`Future` fan-in primitive (used by atlas-consumables). `atlas-channel`'s character model gains a `PartyDecorator` mirroring `InventoryDecorator`; both HP-announce paths short-circuit on solo characters and stop emitting misleading debug logs. `atlas-saga-orchestrator`'s `AcceptEvent` rejects `uuid.Nil` transaction IDs before any storage lookup. `atlas-consumables`' `ConsumeStandard`, `ConsumeTownScroll`, and `ConsumeSummoningSack` issue their independent reads concurrently via `model.Group`.

**Tech Stack:** Go 1.24 (workspaces), `golang.org/x/sync/errgroup`, atlas-model `Provider[T]` combinators, logrus, `github.com/google/uuid`, the existing JSON:API REST helpers in `libs/atlas-rest`, and the project's immutable model + builder + decorator pattern.

---

## Phase 1 — `libs/atlas-model`: `Group` / `Submit` / `Future`

### Task 1: Add `golang.org/x/sync` direct dependency to atlas-model

**Files:**
- Modify: `libs/atlas-model/go.mod`

- [ ] **Step 1: Add `golang.org/x/sync v0.20.0` as a direct require**

Edit `libs/atlas-model/go.mod` from:

```
module github.com/Chronicle20/atlas/libs/atlas-model

go 1.24.4
```

To:

```
module github.com/Chronicle20/atlas/libs/atlas-model

go 1.24.4

require golang.org/x/sync v0.20.0
```

The version matches the workspace pin already used by other modules (e.g., `services/atlas-fame/atlas.com/fame/go.mod`). The dependency is a single package (`errgroup`) with no transitive deps.

- [ ] **Step 2: Run `go mod tidy` in atlas-model**

Run: `cd libs/atlas-model && go mod tidy`

Expected: `go.sum` is populated with the `golang.org/x/sync` entry. No errors.

- [ ] **Step 3: Verify the workspace still builds**

Run: `go build ./...` from `<repo-root>`

Expected: clean build.

- [ ] **Step 4: Commit**

```bash
git add libs/atlas-model/go.mod libs/atlas-model/go.sum
git commit -m "atlas-model: add golang.org/x/sync direct dependency for Group primitive"
```

### Task 2: Write the failing tests for `Group` / `Submit` / `Future`

**Files:**
- Create: `libs/atlas-model/model/parallel_group_test.go`

- [ ] **Step 1: Write the failing test file**

```go
package model

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"context"
)

func TestGroup_TwoSuccessfulProviders(t *testing.T) {
	g, _ := NewGroup(context.Background())
	fa := Submit(g, func() (int, error) { return 1, nil })
	fb := Submit(g, func() (string, error) { return "ok", nil })

	if err := g.Wait(); err != nil {
		t.Fatalf("Wait() unexpected error: %v", err)
	}
	if fa.Get() != 1 {
		t.Errorf("fa.Get() = %d, want 1", fa.Get())
	}
	if fb.Get() != "ok" {
		t.Errorf("fb.Get() = %q, want %q", fb.Get(), "ok")
	}
}

func TestGroup_OneProviderErrors(t *testing.T) {
	wantErr := errors.New("boom")
	g, _ := NewGroup(context.Background())
	_ = Submit(g, func() (int, error) { return 0, wantErr })
	_ = Submit(g, func() (int, error) { return 7, nil })

	err := g.Wait()
	if !errors.Is(err, wantErr) {
		t.Fatalf("Wait() = %v, want %v", err, wantErr)
	}
}

func TestGroup_BothProvidersError(t *testing.T) {
	errA := errors.New("a")
	errB := errors.New("b")
	g, _ := NewGroup(context.Background())
	_ = Submit(g, func() (int, error) { return 0, errA })
	_ = Submit(g, func() (int, error) { return 0, errB })

	err := g.Wait()
	if err == nil {
		t.Fatal("Wait() returned nil, want either errA or errB")
	}
	if !errors.Is(err, errA) && !errors.Is(err, errB) {
		t.Fatalf("Wait() = %v, want errA or errB", err)
	}
}

func TestGroup_ThreeProviders_AllSucceed(t *testing.T) {
	g, _ := NewGroup(context.Background())
	fa := Submit(g, func() (int, error) { return 1, nil })
	fb := Submit(g, func() (int, error) { return 2, nil })
	fc := Submit(g, func() (int, error) { return 3, nil })

	if err := g.Wait(); err != nil {
		t.Fatalf("Wait() unexpected error: %v", err)
	}
	got := fa.Get() + fb.Get() + fc.Get()
	if got != 6 {
		t.Errorf("sum = %d, want 6", got)
	}
}

func TestGroup_ConcurrencyProof(t *testing.T) {
	const sleep = 50 * time.Millisecond
	const tolerance = 40 * time.Millisecond // wall-clock slack

	var inFlight int32
	var maxConcurrent int32

	tick := func() (int, error) {
		cur := atomic.AddInt32(&inFlight, 1)
		// track the high-water mark
		for {
			old := atomic.LoadInt32(&maxConcurrent)
			if cur <= old || atomic.CompareAndSwapInt32(&maxConcurrent, old, cur) {
				break
			}
		}
		time.Sleep(sleep)
		atomic.AddInt32(&inFlight, -1)
		return 0, nil
	}

	start := time.Now()
	g, _ := NewGroup(context.Background())
	_ = Submit(g, tick)
	_ = Submit(g, tick)
	if err := g.Wait(); err != nil {
		t.Fatalf("Wait() unexpected error: %v", err)
	}
	elapsed := time.Since(start)

	if elapsed >= 2*sleep-tolerance {
		t.Fatalf("Wait() took %v; expected <%v (parallel execution)", elapsed, 2*sleep-tolerance)
	}
	if atomic.LoadInt32(&maxConcurrent) < 2 {
		t.Fatalf("maxConcurrent = %d, want >= 2", maxConcurrent)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd libs/atlas-model && go test ./model/ -run TestGroup -v`

Expected: build/compile failure — `Group`, `Submit`, `Future`, `NewGroup` are all undefined.

### Task 3: Implement `Group` / `Submit` / `Future`

**Files:**
- Create: `libs/atlas-model/model/parallel_group.go`

- [ ] **Step 1: Write the implementation**

```go
package model

import (
	"context"

	"golang.org/x/sync/errgroup"
)

// Group runs heterogeneously-typed providers concurrently. It is a thin
// wrapper around errgroup.Group that pairs each registered provider with a
// typed Future handle so call sites can reclaim results without runtime
// type assertions.
type Group struct {
	g *errgroup.Group
}

// Future holds the result of a provider submitted to a Group. After Wait
// returns nil, Get returns the provider's successful value. Get's behaviour
// is undefined when Wait returned an error.
type Future[T any] struct {
	value T
}

// Get returns the value produced by the provider this Future represents.
// Only valid after the parent Group's Wait has returned nil.
func (f *Future[T]) Get() T { return f.value }

// NewGroup returns a Group bound to a child of ctx. The child context is
// cancelled when any submitted provider returns a non-nil error or when
// Wait completes.
func NewGroup(ctx context.Context) (*Group, context.Context) {
	g, gctx := errgroup.WithContext(ctx)
	return &Group{g: g}, gctx
}

// Submit registers a provider with the group, returning a typed Future.
// Submit is a free function rather than a method because Go does not allow
// type parameters on methods.
func Submit[T any](g *Group, p Provider[T]) *Future[T] {
	f := &Future[T]{}
	g.g.Go(func() error {
		v, err := p()
		if err != nil {
			return err
		}
		f.value = v
		return nil
	})
	return f
}

// Wait blocks until all submitted providers complete and returns the first
// non-nil error, if any.
func (g *Group) Wait() error { return g.g.Wait() }
```

- [ ] **Step 2: Run the tests to verify they pass**

Run: `cd libs/atlas-model && go test ./model/ -run TestGroup -v`

Expected: all five tests PASS.

- [ ] **Step 3: Run the entire atlas-model test suite to confirm nothing regressed**

Run: `cd libs/atlas-model && go test ./...`

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add libs/atlas-model/model/parallel_group.go libs/atlas-model/model/parallel_group_test.go
git commit -m "atlas-model: add Group/Submit/Future for typed concurrent fan-in"
```

---

## Phase 2 — `atlas-channel`: party field, builder, accessors

### Task 4: Add a minimal `NewBuilder` to `atlas-channel/party`

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/party/model.go`

`party.Model` currently has no exported constructor — the package only constructs models via `Extract(RestModel)`. The `character.Model` builder/clone tests in Task 5 need a way to build a non-zero `party.Model` directly, and so does the future `PartyDecorator` test in Task 7. A minimal builder is the smallest change that unblocks both.

- [ ] **Step 1: Append a builder to `party/model.go`**

After the `MemberModel` definition at the bottom of `model.go`, append:

```go
type modelBuilder struct {
	id       uint32
	leaderId uint32
	members  []MemberModel
}

// NewBuilder returns a new party model builder. Used by tests and any
// code path that needs to construct a party.Model in-process (the
// production path uses Extract over the REST response).
func NewBuilder() *modelBuilder {
	return &modelBuilder{}
}

func (b *modelBuilder) SetId(v uint32) *modelBuilder           { b.id = v; return b }
func (b *modelBuilder) SetLeaderId(v uint32) *modelBuilder     { b.leaderId = v; return b }
func (b *modelBuilder) SetMembers(v []MemberModel) *modelBuilder { b.members = v; return b }

func (b *modelBuilder) Build() Model {
	return Model{
		id:       b.id,
		leaderId: b.leaderId,
		members:  b.members,
	}
}

// MustBuild returns a Model unconditionally. Kept symmetric with the
// character package's MustBuild.
func (b *modelBuilder) MustBuild() Model {
	return b.Build()
}
```

This builder does no validation (an "empty" `party.Model{}` is the legitimate zero value used as the "not in a party" sentinel — see PRD §4.1 / design §4.1).

- [ ] **Step 2: Build the package**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./party/...`

Expected: clean build.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/party/model.go
git commit -m "atlas-channel/party: add minimal NewBuilder for Model"
```

### Task 5: Write the failing builder/accessor tests for the `party` field

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/character/builder_test.go`

- [ ] **Step 1: Append three new tests at the end of `builder_test.go`**

```go
func TestBuild_PartyDefaultsToZero(t *testing.T) {
	model, err := character.NewModelBuilder().
		SetId(1).
		Build()
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model.InParty() {
		t.Error("InParty() = true on undecorated model, want false")
	}
	if model.Party().Id() != 0 {
		t.Errorf("Party().Id() = %d, want 0", model.Party().Id())
	}
}

func TestBuild_SetParty(t *testing.T) {
	pm := party.NewBuilder().SetId(42).SetLeaderId(7).MustBuild()
	model, err := character.NewModelBuilder().
		SetId(1).
		SetParty(pm).
		Build()
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if !model.InParty() {
		t.Error("InParty() = false after SetParty, want true")
	}
	if model.Party().Id() != 42 {
		t.Errorf("Party().Id() = %d, want 42", model.Party().Id())
	}
}

func TestCloneModel_PreservesParty(t *testing.T) {
	pm := party.NewBuilder().SetId(99).MustBuild()
	original := character.NewModelBuilder().
		SetId(1).
		SetParty(pm).
		MustBuild()

	cloned, err := character.CloneModel(original).Build()
	if err != nil {
		t.Fatalf("CloneModel().Build() unexpected error: %v", err)
	}
	if cloned.Party().Id() != 99 {
		t.Errorf("cloned.Party().Id() = %d, want 99", cloned.Party().Id())
	}
}
```

Add these imports to the existing `import` block (the file currently imports only `"atlas-channel/character"`, `"errors"`, `"testing"`):

```go
import (
	"atlas-channel/character"
	"atlas-channel/party"
	"errors"
	"testing"
)
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./character/ -run "TestBuild_PartyDefaultsToZero|TestBuild_SetParty|TestCloneModel_PreservesParty" -v`

Expected: build failure — `SetParty`, `Party()`, `InParty()` undefined on `character.Model` / `modelBuilder`. (`party.NewBuilder` already exists from Task 4.)

### Task 6: Add `party party.Model` to `character.Model` + builder + accessors

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/character/model.go`
- Modify: `services/atlas-channel/atlas.com/channel/character/builder.go`

- [ ] **Step 1: Add `party` to the `Model` struct and import the party package**

In `model.go`, add `"atlas-channel/party"` to the import block. Add a new field to the `Model` struct (place after `quests []quest.Model`):

```go
type Model struct {
	// …existing fields…
	quests             []quest.Model
	party              party.Model
}
```

- [ ] **Step 2: Add `Party()` and `InParty()` accessors**

In `model.go`, append below the existing `Quests()` accessor:

```go
func (m Model) Party() party.Model {
	return m.party
}

// InParty reports whether the character is currently a member of a party.
// Returns false on an undecorated model (zero-valued party). Callers should
// apply PartyDecorator (or check InParty themselves before assuming party
// data is loaded) before relying on this value.
func (m Model) InParty() bool {
	return m.party.Id() != 0
}
```

- [ ] **Step 3: Add the `SetParty` top-level helper on `Model`**

In `model.go`, append (mirroring the existing `SetSkills` / `SetPets` / `SetQuests` helpers at lines 305–315):

```go
func (m Model) SetParty(p party.Model) Model {
	return CloneModel(m).SetParty(p).MustBuild()
}
```

- [ ] **Step 4: Add `party` to the `modelBuilder` struct**

In `builder.go`, add `"atlas-channel/party"` to the imports. Add a new field to `modelBuilder`:

```go
type modelBuilder struct {
	// …existing fields…
	quests             []quest.Model
	party              party.Model
}
```

- [ ] **Step 5: Propagate `party` through `CloneModel` and `Build`**

In `builder.go`, in `CloneModel`, add at the end of the struct literal (before the closing `}`):

```go
		quests:             m.quests,
		party:              m.party,
	}
```

In `Build`, add inside the returned `Model{…}` literal:

```go
		quests:             b.quests,
		party:              b.party,
	}, nil
```

- [ ] **Step 6: Add the builder setter**

In `builder.go`, append next to the other one-liner setters (next to `SetQuests`):

```go
func (b *modelBuilder) SetParty(v party.Model) *modelBuilder        { b.party = v; return b }
```

- [ ] **Step 7: Run the failing tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./character/ -run "TestBuild_PartyDefaultsToZero|TestBuild_SetParty|TestCloneModel_PreservesParty" -v`

Expected: PASS. (If `party.NewModelBuilder` does not exist, fix the test from Task 4 to construct a `party.Model` via whatever public constructor the package exposes — the goal is to assert `Party().Id() != 0` on a non-zero party.)

- [ ] **Step 8: Run the wider character package tests to confirm no regression**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./character/...`

Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/character/model.go services/atlas-channel/atlas.com/channel/character/builder.go services/atlas-channel/atlas.com/channel/character/builder_test.go
git commit -m "atlas-channel: add party field, Party()/InParty() accessors, builder support"
```

---

### Task 7: Add `PartyDecorator` to the character `Processor` interface (failing)

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/character/processor.go`
- Modify: `services/atlas-channel/atlas.com/channel/character/processor_test.go`

- [ ] **Step 1: Append the failing decorator tests**

Append to `processor_test.go` (which already imports `mock` and `character`):

```go
func TestProcessorImpl_PartyDecorator_NotInParty(t *testing.T) {
	mockProc := mock.NewMockProcessor()
	c := createTestCharacter(123, "SoloChar", 10)

	// Mock decorator is a pass-through that does NOT populate party.
	out := mockProc.PartyDecorator(c)
	if out.InParty() {
		t.Error("InParty() = true on mock-decorated solo character, want false")
	}
}

func TestProcessorImpl_PartyDecorator_InterfaceContract(t *testing.T) {
	// Compile-time assertion that PartyDecorator is on the interface.
	var _ func(character.Model) character.Model = (mock.NewMockProcessor()).PartyDecorator
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./character/ -run "TestProcessorImpl_PartyDecorator" -v`

Expected: build failure — `PartyDecorator` is not on `MockProcessor` (or the interface).

### Task 8: Implement `PartyDecorator` on the interface, ProcessorImpl, and mock

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/character/processor.go`
- Modify: `services/atlas-channel/atlas.com/channel/character/mock/processor.go`

- [ ] **Step 1: Add `PartyDecorator` to the `Processor` interface**

In `processor.go`, edit the interface block (currently lines 24–39) to add a single line right after the other decorators:

```go
type Processor interface {
	GetById(decorators ...model.Decorator[Model]) func(characterId uint32) (Model, error)
	InventoryDecorator(m Model) Model
	PetAssetEnrichmentDecorator(m Model) Model
	SkillModelDecorator(m Model) Model
	QuestModelDecorator(m Model) Model
	PartyDecorator(m Model) Model
	// …rest unchanged…
}
```

- [ ] **Step 2: Add the `PartyDecorator` method on `ProcessorImpl`**

In `processor.go`, add (right below `QuestModelDecorator` at line 152):

```go
// PartyDecorator fetches the party (if any) the character is a member of
// and attaches it via Model.SetParty. Mirrors InventoryDecorator: REST
// failures and "no party" cases both surface as the undecorated model.
// Callers must use Model.InParty() to distinguish "in a party" from
// "solo or not yet decorated".
func (p *ProcessorImpl) PartyDecorator(m Model) Model {
	pm, err := party.NewProcessor(p.l, p.ctx).GetByMemberId(m.Id())
	if err != nil {
		return m
	}
	return m.SetParty(pm)
}
```

Add `"atlas-channel/party"` to the import block in `processor.go`.

- [ ] **Step 3: Add `PartyDecorator` to the mock**

In `mock/processor.go`, add (right below the existing `QuestModelDecorator` at line 66):

```go
func (m *MockProcessor) PartyDecorator(c character.Model) character.Model {
	return c
}
```

- [ ] **Step 4: Run the previously failing tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./character/ -run "TestProcessorImpl_PartyDecorator" -v`

Expected: PASS.

- [ ] **Step 5: Run the full character package tests**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./character/...`

Expected: PASS.

- [ ] **Step 6: Verify the full atlas-channel module builds**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./...`

Expected: clean build.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/character/processor.go services/atlas-channel/atlas.com/channel/character/processor_test.go services/atlas-channel/atlas.com/channel/character/mock/processor.go
git commit -m "atlas-channel: add PartyDecorator to character Processor"
```

---

## Phase 3 — `atlas-channel`: HP-announce path cleanup

### Task 9: Update `kafka/consumer/character` HP-announce path

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/character/consumer.go` (lines 98–107 in `handleStatusEventStatChanged`)

- [ ] **Step 1: Replace the `if hpChange { … }` block**

Locate the current block (lines 98–107):

```go
if hpChange {
    // TODO - field migration this seems like a bug not using instance
    f := field.NewBuilder(e.WorldId, e.Body.ChannelId, c.MapId()).Build()
    imf := party.OtherMemberInMap(f, c.Id())
    oip := party.MemberToMemberIdMapper(party.FilteredMemberProvider(imf)(party.NewProcessor(l, ctx).ByMemberIdProvider(e.CharacterId)))
    err = session.NewProcessor(l, ctx).ForEachByCharacterId(sc.Channel())(oip, session.Announce(l)(ctx)(wp)(partycb.PartyMemberHPWriter)(partycb.NewPartyMemberHP(c.Id(), c.Hp(), c.MaxHp()).Encode))
    if err != nil {
        l.WithError(err).Debugf("Unable to announce character [%d] health to party members.", c.Id())
    }
}
```

Replace with:

```go
if hpChange {
    cp := character.NewProcessor(l, ctx)
    cd, derr := cp.GetById(cp.PartyDecorator)(c.Id())
    if derr != nil || !cd.InParty() {
        return
    }
    // TODO - field migration this seems like a bug not using instance
    f := field.NewBuilder(e.WorldId, e.Body.ChannelId, cd.MapId()).Build()
    imf := party.OtherMemberInMap(f, cd.Id())
    pmp := model.FixedProvider(cd.Party())
    oip := party.MemberToMemberIdMapper(party.FilteredMemberProvider(imf)(pmp))
    _ = session.NewProcessor(l, ctx).ForEachByCharacterId(sc.Channel())(oip, session.Announce(l)(ctx)(wp)(partycb.PartyMemberHPWriter)(partycb.NewPartyMemberHP(cd.Id(), cd.Hp(), cd.MaxHp()).Encode))
}
```

The `Unable to announce…` debug log is removed; any genuine error from `ForEachByCharacterId` is swallowed (matches the same convention used elsewhere — the announce path is best-effort and a transient failure is not actionable).

If the file does not already import `"github.com/Chronicle20/atlas/libs/atlas-model/model"`, add it. The `character` package is already imported (used for the original GET).

- [ ] **Step 2: Build the package to verify the change compiles**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./kafka/consumer/character/...`

Expected: clean build.

- [ ] **Step 3: Run the existing kafka/consumer/character tests (if any)**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./kafka/consumer/character/...`

Expected: PASS (or "no test files" if none exist; the package is intentionally light on tests).

- [ ] **Step 4: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/consumer/character/consumer.go
git commit -m "atlas-channel: short-circuit HP announce on solo characters; drop misleading debug log"
```

### Task 10: Update `kafka/consumer/map` HP-announce path

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go` (lines 225–236)

- [ ] **Step 1: Replace the party-announce goroutine**

Locate the existing block at lines 225–236:

```go
go func() {
    imf := party.OtherMemberInMap(s.Field(), s.CharacterId())
    oip := party.MemberToMemberIdMapper(party.FilteredMemberProvider(imf)(party.NewProcessor(l, ctx).ByMemberIdProvider(s.CharacterId())))
    err = session.NewProcessor(l, ctx).ForEachByCharacterId(s.Field().Channel())(oip, session.Announce(l)(ctx)(wp)(partycb.PartyMemberHPWriter)(partycb.NewPartyMemberHP(s.CharacterId(), cms[s.CharacterId()].Hp(), cms[s.CharacterId()].MaxHp()).Encode))
    if err != nil {
        l.WithError(err).Debugf("Unable to announce character [%d] health to party members.", s.CharacterId())
    }

    _ = model.ForEachSlice(oip, func(oid uint32) error {
        return session.Announce(l)(ctx)(wp)(partycb.PartyMemberHPWriter)(partycb.NewPartyMemberHP(oid, cms[oid].Hp(), cms[oid].MaxHp()).Encode)(s)
    }, model.ParallelExecute())
}()
```

Replace with:

```go
go func() {
    cp := character.NewProcessor(l, ctx)
    cd, err := cp.GetById(cp.PartyDecorator)(s.CharacterId())
    if err != nil || !cd.InParty() {
        return
    }
    pmp := model.FixedProvider(cd.Party())
    imf := party.OtherMemberInMap(s.Field(), s.CharacterId())
    oip := party.MemberToMemberIdMapper(party.FilteredMemberProvider(imf)(pmp))
    _ = session.NewProcessor(l, ctx).ForEachByCharacterId(s.Field().Channel())(oip, session.Announce(l)(ctx)(wp)(partycb.PartyMemberHPWriter)(partycb.NewPartyMemberHP(s.CharacterId(), cms[s.CharacterId()].Hp(), cms[s.CharacterId()].MaxHp()).Encode))
    _ = model.ForEachSlice(oip, func(oid uint32) error {
        return session.Announce(l)(ctx)(wp)(partycb.PartyMemberHPWriter)(partycb.NewPartyMemberHP(oid, cms[oid].Hp(), cms[oid].MaxHp()).Encode)(s)
    }, model.ParallelExecute())
}()
```

The HP-announce-back-to-joiner loop (`model.ForEachSlice` over `oip`) stays inside the same `!InParty()` short-circuit — solo characters do not need the broadcast back to themselves either.

If the `character` package is not already imported in this file, add `"atlas-channel/character"` to the import block.

- [ ] **Step 2: Build the package**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./kafka/consumer/map/...`

Expected: clean build.

- [ ] **Step 3: Run any existing map-consumer tests**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./kafka/consumer/map/...`

Expected: PASS (or "no test files").

- [ ] **Step 4: Run the full atlas-channel build to confirm no broader breakage**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./...`

Expected: clean build.

- [ ] **Step 5: Run the full atlas-channel test suite**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./...`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go
git commit -m "atlas-channel: short-circuit map-join HP announce on solo characters"
```

---

## Phase 4 — `atlas-saga-orchestrator`: nil-UUID short-circuit

### Task 11: Add the failing `TestAcceptEvent_NilTransactionId` test

**Files:**
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/accept_event_test.go`

- [ ] **Step 1: Append the new test to `accept_event_test.go`**

Append at the end of the file:

```go
func TestAcceptEvent_NilTransactionId(t *testing.T) {
	p, hook, _ := newAcceptEventTestProcessor(t)

	_, ok := p.AcceptEvent(uuid.Nil, EventKindAssetCreated)
	assert.False(t, ok, "AcceptEvent must return false for uuid.Nil")

	require.Len(t, hook.AllEntries(), 1, "exactly one debug log expected")
	entry := hook.AllEntries()[0]
	assert.Equal(t, logrus.DebugLevel, entry.Level)
	assert.Equal(t, SkipReasonNilTransactionId, entry.Data["reason"])
	assert.NotEqual(t, SkipReasonSagaNotFound, entry.Data["reason"], "must NOT log saga_not_found for nil-UUID events")

	// transaction_id field must NOT be on the log payload — there is no
	// meaningful UUID to log.
	_, hasTxId := entry.Data["transaction_id"]
	assert.False(t, hasTxId, "transaction_id should be omitted from nil-UUID skip logs")
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./saga/ -run TestAcceptEvent_NilTransactionId -v`

Expected: build failure — `SkipReasonNilTransactionId` undefined.

### Task 12: Add `SkipReasonNilTransactionId` and the `AcceptEvent` guard

**Files:**
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/event_acceptance.go`
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/processor.go`

- [ ] **Step 1: Add the new constant**

In `event_acceptance.go`, edit the `const` block (lines 215–221):

```go
const (
	SkipReasonSagaNotFound       = "saga_not_found"
	SkipReasonNoPendingStep      = "no_pending_step"
	SkipReasonActionMismatch     = "action_mismatch"
	SkipReasonTemplateIdMismatch = "template_id_mismatch"
	SkipReasonUnmatchedEvent     = "unmatched_event"
	SkipReasonNilTransactionId   = "nil_transaction_id"
)
```

- [ ] **Step 2: Add the guard at the top of `AcceptEvent`**

In `processor.go`, edit `AcceptEvent` (line 362) to add a guard as the first statement inside the function body:

```go
func (p *ProcessorImpl) AcceptEvent(transactionId uuid.UUID, kind EventKind) (AcceptDecision, bool) {
	if transactionId == uuid.Nil {
		LogSkip(p.l, logrus.Fields{
			"event_kind": kind,
		}, SkipReasonNilTransactionId)
		return AcceptDecision{}, false
	}
	s, err := p.GetById(transactionId)
	// …rest of the function unchanged…
```

The log payload deliberately omits `transaction_id` (the value is meaningless). `event_kind` is retained so volume per kind can be measured.

- [ ] **Step 3: Run the new test to verify it passes**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./saga/ -run TestAcceptEvent_NilTransactionId -v`

Expected: PASS.

- [ ] **Step 4: Run the full `accept_event_test.go` suite**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./saga/ -run TestAcceptEvent -v`

Expected: PASS — `TestAcceptEvent_SagaNotFound`, `TestAcceptEvent_NoPendingStep`, `TestAcceptEvent_ActionMismatch`, `TestAcceptEvent_Match`, `TestAcceptEvent_WarnOnceForUnmatchedEvent`, and the new test all pass.

- [ ] **Step 5: Run the full atlas-saga-orchestrator test suite**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./...`

Expected: PASS.

- [ ] **Step 6: Run the full atlas-saga-orchestrator build**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./...`

Expected: clean build.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/event_acceptance.go services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/processor.go services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/accept_event_test.go
git commit -m "atlas-saga-orchestrator: short-circuit AcceptEvent on uuid.Nil transaction id"
```

---

## Phase 5 — `atlas-consumables`: parallel independent reads

### Task 13: Refactor `ConsumeStandard` to issue its three reads concurrently

**Files:**
- Modify: `services/atlas-consumables/atlas.com/consumables/consumable/processor.go` (lines 212–242)

- [ ] **Step 1: Replace `ConsumeStandard`**

Locate the current implementation (lines 212–242). Replace the whole function with:

```go
func ConsumeStandard(transactionId uuid.UUID, characterId uint32, slot int16, itemId item2.Id) ItemConsumer {
	return func(l logrus.FieldLogger) func(ctx context.Context) error {
		return func(ctx context.Context) error {
			p := NewProcessor(l, ctx)
			cp := character.NewProcessor(l, ctx)
			mp := character2.NewProcessor(l, ctx)

			pg, _ := model.NewGroup(ctx)
			fc := model.Submit(pg, func() (character.Model, error) { return cp.GetById()(characterId) })
			fm := model.Submit(pg, func() (field.Model, error) { return mp.GetMap(characterId) })
			fi := model.Submit(pg, func() (consumable3.Model, error) { return p.cdp.GetById(uint32(itemId)) })
			if err := pg.Wait(); err != nil {
				return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, slot, err)
			}
			c, m, ci := fc.Get(), fm.Get(), fi.Get()

			err := compartment.NewProcessor(l, ctx).ConsumeItem(characterId, inventory2.TypeValueUse, transactionId, slot)
			if err != nil {
				return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, slot, err)
			}

			ApplyItemEffects(l, ctx, c, m, ci, characterId, itemId)
			return nil
		}
	}
}
```

Notes:
- `mp.GetMap(characterId)` returns `field.Model` (NOT `_map.Model`). The function signature of `ApplyItemEffects` already takes `f field.Model`, so the type lines up.
- `p.cdp` is the unexported `*consumable3.Processor` on the `Processor` struct (line 54). It is in the same package, so the closure can access it.
- The `field` package is already used elsewhere in this file (line 31 import of `"github.com/Chronicle20/atlas/libs/atlas-constants/field"`), so the type reference is in scope.

- [ ] **Step 2: Build the package**

Run: `cd services/atlas-consumables/atlas.com/consumables && go build ./consumable/...`

Expected: clean build.

- [ ] **Step 3: Run the consumable package tests**

Run: `cd services/atlas-consumables/atlas.com/consumables && go test ./consumable/...`

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-consumables/atlas.com/consumables/consumable/processor.go
git commit -m "atlas-consumables: parallelize ConsumeStandard's three independent reads"
```

### Task 14: Refactor `ConsumeTownScroll` to parallelise the two independent reads

**Files:**
- Modify: `services/atlas-consumables/atlas.com/consumables/consumable/processor.go` (lines 244–284)

- [ ] **Step 1: Replace `ConsumeTownScroll`**

```go
func ConsumeTownScroll(transactionId uuid.UUID, characterId uint32, slot int16, itemId item2.Id) ItemConsumer {
	return func(l logrus.FieldLogger) func(ctx context.Context) error {
		return func(ctx context.Context) error {
			p := NewProcessor(l, ctx)
			cpp := compartment.NewProcessor(l, ctx)
			mp := character2.NewProcessor(l, ctx)

			pg, _ := model.NewGroup(ctx)
			fm := model.Submit(pg, func() (field.Model, error) { return mp.GetMap(characterId) })
			fi := model.Submit(pg, func() (consumable3.Model, error) { return p.cdp.GetById(uint32(itemId)) })
			if err := pg.Wait(); err != nil {
				return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, slot, err)
			}
			m, ci := fm.Get(), fi.Get()

			toMapId := _map2.EmptyMapId
			if val, ok := ci.GetSpec(consumable3.SpecTypeMoveTo); ok && val > 0 {
				toMapId = _map2.Id(val)
			}
			// Dependent read: needs m.MapId() — stays sequential after Wait().
			if toMapId == _map2.EmptyMapId {
				mm, err := _map3.NewProcessor(l, ctx).GetById(m.MapId())
				if err != nil {
					return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, slot, err)
				}
				toMapId = mm.ReturnMapId()
			}

			err := cpp.ConsumeItem(characterId, inventory2.TypeValueUse, transactionId, slot)
			if err != nil {
				return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, slot, err)
			}

			toField := field.NewBuilder(m.WorldId(), m.ChannelId(), toMapId).SetInstance(m.Instance()).Build()
			err = _map.NewProcessor(l, ctx).WarpRandom(toField)(characterId)
			if err != nil {
				return err
			}
			return nil
		}
	}
}
```

The `_map3.GetById(m.MapId())` call remains sequential after `Wait()` because it depends on `m`.

- [ ] **Step 2: Build the package**

Run: `cd services/atlas-consumables/atlas.com/consumables && go build ./consumable/...`

Expected: clean build.

- [ ] **Step 3: Run consumable tests**

Run: `cd services/atlas-consumables/atlas.com/consumables && go test ./consumable/...`

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-consumables/atlas.com/consumables/consumable/processor.go
git commit -m "atlas-consumables: parallelize ConsumeTownScroll's two independent reads"
```

### Task 15: Refactor `ConsumeSummoningSack` to parallelise the two independent reads

**Files:**
- Modify: `services/atlas-consumables/atlas.com/consumables/consumable/processor.go` (lines 358–397)

- [ ] **Step 1: Replace `ConsumeSummoningSack`**

```go
func ConsumeSummoningSack(transactionId uuid.UUID, ch channel.Model, characterId uint32, slot int16, itemId item2.Id) ItemConsumer {
	return func(l logrus.FieldLogger) func(ctx context.Context) error {
		return func(ctx context.Context) error {
			p := NewProcessor(l, ctx)
			cp := character.NewProcessor(l, ctx)

			pg, _ := model.NewGroup(ctx)
			fc := model.Submit(pg, func() (character.Model, error) { return cp.GetById()(characterId) })
			fi := model.Submit(pg, func() (consumable3.Model, error) { return consumable3.NewProcessor(l, ctx).GetById(uint32(itemId)) })
			if err := pg.Wait(); err != nil {
				return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, slot, err)
			}
			c, ci := fc.Get(), fi.Get()

			// Dependent read: needs c.MapId(), c.X(), c.Y() — stays sequential.
			pos, err := position.NewProcessor(l, ctx).GetInMap(c.MapId(), c.X(), c.Y(), c.X(), c.Y())()
			if err != nil {
				return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, slot, err)
			}

			l.Debugf("Character [%d] summoning [%d] monsters at [%d,%d]. They are at [%d,%d].", characterId, len(ci.MonsterSummons()), pos.X(), pos.Y(), c.X(), c.Y())
			for _, msm := range ci.MonsterSummons() {
				roll := uint32(rand.Int31n(100))
				if roll < msm.Probability() {
					f := field.NewBuilder(ch.WorldId(), ch.Id(), c.MapId()).Build()
					err = monster.NewProcessor(l, ctx).CreateMonster(f, msm.TemplateId(), pos.X(), pos.Y(), 0, 0)
					if err != nil {
						l.WithError(err).Errorf("Unable to summon monster [%d] for character [%d] summoning bag.", msm.TemplateId(), characterId)
					} else {
						l.Debugf("Character [%d] use of summoning sack [%d] spawned monster [%d] at [%d,%d].", characterId, itemId, msm.TemplateId(), c.X(), c.Y())
					}
				}
			}

			err = compartment.NewProcessor(l, ctx).ConsumeItem(characterId, inventory2.TypeValueUse, transactionId, slot)
			if err != nil {
				return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, slot, err)
			}
			return nil
		}
	}
}
```

The original used ad-hoc `NewProcessor(l, ctx)` calls inside `ConsumeError` invocations; this version reuses `p := NewProcessor(l, ctx)` for clarity, matching the surrounding style.

- [ ] **Step 2: Build the package**

Run: `cd services/atlas-consumables/atlas.com/consumables && go build ./consumable/...`

Expected: clean build.

- [ ] **Step 3: Run consumable tests**

Run: `cd services/atlas-consumables/atlas.com/consumables && go test ./consumable/...`

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-consumables/atlas.com/consumables/consumable/processor.go
git commit -m "atlas-consumables: parallelize ConsumeSummoningSack's two independent reads"
```

### Task 16: Document why `ConsumePetFood`, `ConsumeCashPetFood`, and `ConsumeScroll` stay sequential

**Files:**
- Modify: `services/atlas-consumables/atlas.com/consumables/consumable/processor.go` (`ConsumePetFood`, `ConsumeCashPetFood`, `RequestScroll`)

- [ ] **Step 1: Add a one-line comment to `ConsumePetFood` (line 286)**

Insert immediately above the `pe, err := pp.HungriestByOwnerProvider(...)` call (line 293):

```go
// Sequential reads: PRD §4.3 names ConsumeStandard / ConsumeTownScroll /
// ConsumeSummoningSack as the parallelisation targets. ConsumePetFood's two
// reads are independent and could be parallelised in a follow-up — left
// sequential here to keep the cleanup tightly scoped.
```

- [ ] **Step 2: Add a one-line comment to `ConsumeCashPetFood` (line 321)**

Insert immediately above the `ci, err := cash.NewProcessor(l, ctx).GetById(...)` call (line 327):

```go
// Sequential reads: ci.Indexes() feeds the pet filter built immediately
// after, so reads are not independent.
```

- [ ] **Step 3: Add a one-line comment to `RequestScroll` (line 399)**

Insert immediately above the `c, err := cp.GetById(cp.InventoryDecorator)(characterId)` call (line 406):

```go
// Sequential reads: equipment lookup and reservation logic below all
// depend on c.Inventory(); the read chain is genuinely sequential.
```

- [ ] **Step 4: Build and test**

Run: `cd services/atlas-consumables/atlas.com/consumables && go build ./... && go test ./consumable/...`

Expected: clean build, PASS.

- [ ] **Step 5: Run the full atlas-consumables test suite**

Run: `cd services/atlas-consumables/atlas.com/consumables && go test ./...`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-consumables/atlas.com/consumables/consumable/processor.go
git commit -m "atlas-consumables: document why remaining Consume* variants stay sequential"
```

---

## Phase 6 — Cross-cutting verification

### Task 17: Workspace-wide build + targeted tests

- [ ] **Step 1: Build every module in the workspace**

Run from `<repo-root>`:

```bash
go build ./...
```

Expected: clean build across `libs/atlas-model`, `services/atlas-channel`, `services/atlas-saga-orchestrator`, `services/atlas-consumables`, and any consumers that transitively depend on these modules.

- [ ] **Step 2: Run targeted tests for every affected module**

```bash
(cd libs/atlas-model && go test ./...)
(cd services/atlas-channel/atlas.com/channel && go test ./...)
(cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./...)
(cd services/atlas-consumables/atlas.com/consumables && go test ./...)
```

Expected: all PASS.

- [ ] **Step 3: Confirm no stale `Unable to announce character … health to party members` strings remain**

Run:

```bash
grep -rn "Unable to announce character.*health" services/atlas-channel/
```

Expected: no results.

- [ ] **Step 4: Confirm `SkipReasonNilTransactionId` is wired in**

Run:

```bash
grep -n "SkipReasonNilTransactionId" services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/event_acceptance.go services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/processor.go services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/accept_event_test.go
```

Expected: at least one match in each of the three files.

- [ ] **Step 5: Confirm `model.Group` is used in the three intended Consume\* variants**

Run:

```bash
grep -n "model.NewGroup\|model.Submit" services/atlas-consumables/atlas.com/consumables/consumable/processor.go
```

Expected: matches inside `ConsumeStandard`, `ConsumeTownScroll`, `ConsumeSummoningSack`. No matches in `ConsumePetFood`, `ConsumeCashPetFood`, or `RequestScroll`.

- [ ] **Step 6: No commit — verification only**

This task is a guard rail; nothing to commit unless one of the checks fails (in which case fix and create a follow-up commit on the affected service).

---

## Out of Scope (do NOT do)

- Header- or topic-level filtering of non-saga events at the Kafka consumer layer.
- Adding `partyId` to atlas-character's REST contract or storage.
- Caching party state in atlas-channel.
- Removing the GET at the start of `handleStatusEventStatChanged` — the event payload does not carry HP/MaxHp/MapId.
- Parallelising `ConsumeScroll` — its reads are not independent (PRD §2 non-goals).
- Adding business-attribute span enrichment.
- Changing saga subscription topology or `acceptanceTable`.
- Frontend / atlas-ui changes.

## Acceptance Criteria Mapping

| PRD criterion | Plan task(s) |
|---|---|
| 1. Solo HP-potion: zero "Unable to announce…" lines | Tasks 9, 10 |
| 2. Both announce paths use `cp.GetById(cp.PartyDecorator)(...)` | Tasks 9, 10 |
| 3. `Party()` and `InParty()` with unit-test coverage | Tasks 4, 5, 6, 7, 8 |
| 4. Zero `saga_not_found` for nil-UUID events; new `nil_transaction_id` reason | Tasks 11, 12 |
| 5. `ConsumeStandard` reads concurrent | Task 13 |
| 6. Same parallelisation for `ConsumeTownScroll`, `ConsumeSummoningSack`; comments elsewhere | Tasks 14, 15, 16 |
| 7. p50 ≤ 200ms on root span | Verified out-of-band on a fresh Tempo trace after Tasks 13–16 ship; Task 17 confirms code paths are wired correctly. |
| 8. All affected services build and pass tests | Task 17 |
| 9. No regression in HP application / item consumption | Task 17 (test suites) + manual verification per acceptance |
