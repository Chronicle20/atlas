# Portal ENTER Double-Execute — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stop a single portal touch from executing the matched portal rule (and its warp) twice, by never clearing the client's exclusive-request flag while a warp is in flight, by making warp saga steps acknowledge the observed warp so `FAILED` becomes trustworthy, and by adding a short Redis dedupe gate as insurance.

**Architecture:** Three independent layers. **Layer A** (`atlas-saga-orchestrator`) moves `WarpToPortal` / `WarpToSavedLocation` from self-completing to `MAP_CHANGED`-acknowledged, guarded by a new character-id constraint on `AcceptEvent` so a party-quest fan-out cannot complete the wrong step. **Layer B** (`atlas-portal-actions`) classifies each portal operation as moving/static in a table whose zero value is invalid, has the executor report whether a moving operation was *successfully dispatched*, and suppresses `EnableActions` on that signal — backed by a `PendingAction` registration and a 5 s saga timeout so a genuinely failed warp still frees the player. **Layer C** (`atlas-portal-actions`) adds a fail-open Redis `SET NX` + 2 s TTL gate in front of rule evaluation.

**Tech Stack:** Go 1.x, `libs/atlas-saga` (Builder/Step/Payload), `libs/atlas-redis` (`Lock`, `TenantRegistry`, `TenantKey`, `CompositeKey`), `libs/atlas-kafka`, `libs/atlas-script-core/operation`, testify + `logrus/hooks/test` + `miniredis/v2`.

## Global Constraints

- All work happens in the worktree `/.worktrees/task-184-portal-enter-double-execute` on branch `task-184-portal-enter-double-execute`. Never edit the main checkout.
- **No wire, schema, packet, template, or coverage-matrix change.** `tools/template-*-guard.sh` and the packet matrix are not in play.
- **No new Go dependency.** `libs/atlas-redis`, `libs/atlas-saga`, `miniredis/v2` are already in both services' `go.mod`. If no `go.mod` changes, `docker buildx bake` is **not** required (CLAUDE.md item 4 is conditional on `go.mod` being touched) — but you MUST verify with `git status` at Task 10 and bake if it did change.
- **No raw keyed `go-redis` commands** outside `libs/atlas-redis` — `tools/redis-key-guard.sh` enforces this. Use `atlas.Lock` / `atlas.TenantRegistry`.
- **No bare `go` statements** — `tools/goroutine-guard.sh`. Nothing in this plan spawns a goroutine.
- **No `// TODO`, stubs, or deferred work** in landed commits.
- **No literal home/absolute paths** (`/home/<user>/...`) in any committed file.
- Preserve existing line endings; do not reformat untouched code.
- Exact constant values, fixed by the design and PRD:
  - `enterGateTTL = 2 * time.Second` (FR-3.4)
  - `warpSagaTimeout = 5 * time.Second` (FR-2.6)
  - `pendingActionTTL = 60 * time.Second` (design §4.3)
  - `SkipReasonCharacterIdMismatch = "character_id_mismatch"` (FR-1.5)
  - dedupe Redis namespace: `"portal-enter"`
  - warp failure message: `"You cannot move there right now."`
  - transport failure message (unchanged): `"Unable to board transport at this time."`
- Task order matters: Layer A tasks (1–3) land before Layer B (4–7), which land before Layer C (8–9). Within Layer B, the pending-action safety net (Task 5) lands before the unlock suppression (Task 7).

---

## File Structure

### `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/`

| File | Change | Responsibility after change |
|---|---|---|
| `saga/character_extractor.go` | Modify | Adds `WarpToSavedLocationPayload` case (FR-1.4) |
| `saga/character_extractor_test.go` | Create | Unit coverage for the two warp payload cases |
| `saga/event_acceptance.go` | Modify | Adds `SkipReasonCharacterIdMismatch`; moves the two warp actions into the `MAP_CHANGED` acceptance block; rewrites the false comment (FR-1.1, FR-1.2, FR-1.5) |
| `saga/processor.go` | Modify | `AcceptOption` / `ForCharacter` variadic option + the character-id guard as the last check in `AcceptEvent` (FR-1.3) |
| `saga/mock/processor.go` | Modify | Mock signature tracks the interface |
| `saga/accept_event_test.go` | Modify | FR-4.3, FR-4.4, FR-4.5 tests |
| `kafka/consumer/character/consumer.go` | Modify | Sole caller passing `saga.ForCharacter(e.CharacterId)` |

### `services/atlas-portal-actions/atlas.com/portal/`

| File | Change | Responsibility after change |
|---|---|---|
| `script/optable.go` | Create | `opClass`, `opDef`, `opTable`, `validateOpTable`, `IsMovingOperation` — the single source of truth for "does this operation move the character" (FR-2.1, FR-2.2) |
| `script/optable_test.go` | Create | Classification assertions + the unset-class panic |
| `script/executor.go` | Modify | `ExecuteOperation` dispatches through `opTable`; `ExecuteOperations` returns `movedCharacter`; warp methods register a pending action and set a 5 s timeout |
| `script/executor_test.go` | Create | `movedCharacter` semantics incl. failed dispatch |
| `script/model.go` | Modify | `ProcessResult.CharacterMoved` |
| `script/processor.go` | Modify | Populates `CharacterMoved` on both returns |
| `script/consumer.go` | Modify | Dedupe gate first; conditional unlock via seam vars |
| `script/consumer_test.go` | Create | FR-4.1, FR-4.2, and the failed-dispatch-still-unlocks case |
| `action/registry.go` | Modify | `PendingAction.Kind` + `AddWithTTL` |
| `action/registry_test.go` | Modify | `AddWithTTL` round-trip + expiry |
| `dedupe/gate.go` | Create | `Gate` interface, `Key`, Redis impl, fail-open, package singleton |
| `dedupe/gate_test.go` | Create | FR-3.6 fail-open, dup-drop, tenant isolation |
| `kafka/consumer/saga/consumer.go` | Modify | `Kind`-aware default failure message; corrected log wording (FR-2.7) |
| `kafka/consumer/saga/consumer_test.go` | Create | `resolveFailureMessage` matrix |
| `main.go` | Modify | `dedupe.InitGate(rc)` beside `action.InitRegistry(rc)` |

---

## Task 1: `ExtractCharacterId` handles `WarpToSavedLocationPayload`

Satisfies FR-1.4. This must land before the guard in Task 2, or the guard cannot constrain `WarpToSavedLocation` steps and would silently treat them as unconstrained.

**Files:**
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/character_extractor.go`
- Test: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/character_extractor_test.go` (create)

**Interfaces:**
- Consumes: `saga.Step[any]` (existing), `sharedsaga` payload types re-exported into the `saga` package.
- Produces: `ExtractCharacterId(step Step[any]) uint32` now returns the character id for `WarpToSavedLocationPayload`. Task 2's guard depends on this.

**Working directory for all commands in this task:** `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`

- [x] **Step 1: Write the failing test**

Create `saga/character_extractor_test.go`:

```go
package saga

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractCharacterId_WarpToPortalPayload(t *testing.T) {
	step := Step[any]{Payload: WarpToPortalPayload{CharacterId: 4242}}
	assert.Equal(t, uint32(4242), ExtractCharacterId(step))
}

// FR-1.4: without this case the character-id guard cannot constrain a
// WarpToSavedLocation step — it would read 0 and treat the step as
// unconstrained, silently defeating the guard for that action.
func TestExtractCharacterId_WarpToSavedLocationPayload(t *testing.T) {
	step := Step[any]{Payload: WarpToSavedLocationPayload{CharacterId: 777, LocationType: "FREE_MARKET"}}
	assert.Equal(t, uint32(777), ExtractCharacterId(step))
}

func TestExtractCharacterId_UnknownPayloadIsZero(t *testing.T) {
	step := Step[any]{Payload: struct{ Foo string }{Foo: "bar"}}
	assert.Equal(t, uint32(0), ExtractCharacterId(step),
		"unknown payloads must read 0 so the guard leaves them unconstrained")
}
```

If `Step[any]{Payload: ...}` does not compile as a composite literal (the struct's fields may be unexported in this package), construct the step the same way the existing tests in `saga/accept_event_test.go` do and adapt — read `accept_event_test.go` lines 49–75 for the in-repo construction idiom before editing.

- [x] **Step 2: Run test to verify it fails**

Run: `go test ./saga/ -run TestExtractCharacterId -v`
Expected: `TestExtractCharacterId_WarpToSavedLocationPayload` FAILS with `expected: 0x309 actual: 0x0` (the `default:` arm returns 0). The other two PASS.

- [x] **Step 3: Add the payload case**

In `saga/character_extractor.go`, immediately after the existing `WarpToPortalPayload` case:

```go
	case WarpToPortalPayload:
		return p.CharacterId
	case WarpToSavedLocationPayload:
		return p.CharacterId
```

- [x] **Step 4: Run test to verify it passes**

Run: `go test ./saga/ -run TestExtractCharacterId -v`
Expected: all three PASS.

- [x] **Step 5: Commit**

```bash
git add saga/character_extractor.go saga/character_extractor_test.go
git commit -m "feat(saga-orchestrator): extract character id from WarpToSavedLocationPayload (FR-1.4)"
```

---

## Task 2: Character-id guard on `AcceptEvent`

Satisfies FR-1.3 and FR-1.5. The guard mechanism lands *before* the acceptance-table change of Task 3, because Task 3 is what makes cross-character `MAP_CHANGED` events start matching — landing them in the other order opens a window where a party-quest fan-out can complete the wrong step.

This task is testable on its own using `WarpToRandomPortal`, which **already** accepts `EventKindCharacterMapChanged` today and whose payload already yields a character id.

**Files:**
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/event_acceptance.go` (SkipReason constant block, around line 386)
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/processor.go` (interface decl ~line 63; `AcceptEvent` impl ~line 397)
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/mock/processor.go` (field ~line 42, method ~line 206)
- Test: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/accept_event_test.go`

**Interfaces:**
- Consumes: `ExtractCharacterId` from Task 1; `LogSkip`, `SkipReason*` from `event_acceptance.go`.
- Produces, used by Task 3 and by the character consumer:
  - `type AcceptOption func(*acceptOptions)`
  - `func ForCharacter(id uint32) AcceptOption`
  - `AcceptEvent(transactionId uuid.UUID, kind EventKind, opts ...AcceptOption) (AcceptDecision, bool)` — variadic, so all 60-odd existing call sites compile unchanged.
  - `const SkipReasonCharacterIdMismatch = "character_id_mismatch"`

**Working directory:** `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`

- [x] **Step 1: Write the failing test**

Append to `saga/accept_event_test.go`:

```go
// FR-1.3/FR-1.5: a MAP_CHANGED for character A must not complete a step whose
// payload names character B. WarpToRandomPortal is used here because it
// already accepts EventKindCharacterMapChanged before Task 3 lands.
func TestAcceptEvent_CharacterIdMismatchSkips(t *testing.T) {
	p, hook, ctx := newAcceptEventTestProcessor(t)
	tx := uuid.New()
	s := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("warp-1", Pending, WarpToRandomPortal, WarpToRandomPortalPayload{
			CharacterId: 100,
			WorldId:     0,
			ChannelId:   1,
			FieldId:     _map.Id(200000000),
		}).
		Build()
	putAcceptEventSaga(t, ctx, s)

	_, ok := p.AcceptEvent(tx, EventKindCharacterMapChanged, ForCharacter(999))
	assert.False(t, ok, "an event for a different character must not complete the step")

	var found bool
	for _, e := range hook.AllEntries() {
		if e.Data["reason"] == SkipReasonCharacterIdMismatch {
			found = true
			assert.Equal(t, uint32(999), e.Data["event_character_id"])
			assert.Equal(t, uint32(100), e.Data["step_character_id"])
		}
	}
	assert.True(t, found, "the skip must be logged with reason character_id_mismatch")
}

// The matching character completes normally.
func TestAcceptEvent_CharacterIdMatchAccepts(t *testing.T) {
	p, _, ctx := newAcceptEventTestProcessor(t)
	tx := uuid.New()
	s := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("warp-1", Pending, WarpToRandomPortal, WarpToRandomPortalPayload{
			CharacterId: 100,
			WorldId:     0,
			ChannelId:   1,
			FieldId:     _map.Id(200000000),
		}).
		Build()
	putAcceptEventSaga(t, ctx, s)

	decision, ok := p.AcceptEvent(tx, EventKindCharacterMapChanged, ForCharacter(100))
	require.True(t, ok)
	assert.Equal(t, "warp-1", decision.Step.StepId())
}

// A payload with no character id is unconstrained — every pre-existing
// action's behaviour is preserved.
func TestAcceptEvent_NoCharacterIdInPayloadIsUnconstrained(t *testing.T) {
	p, _, ctx := newAcceptEventTestProcessor(t)
	tx := uuid.New()
	s := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("warp-1", Pending, WarpToRandomPortal, struct{ Nothing string }{}).
		Build()
	putAcceptEventSaga(t, ctx, s)

	_, ok := p.AcceptEvent(tx, EventKindCharacterMapChanged, ForCharacter(999))
	assert.True(t, ok, "ExtractCharacterId returns 0 -> unconstrained -> accept")
}

// Calling without the option is unchanged behaviour for the other ~60 sites.
func TestAcceptEvent_NoOptionIsUnconstrained(t *testing.T) {
	p, _, ctx := newAcceptEventTestProcessor(t)
	tx := uuid.New()
	s := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("warp-1", Pending, WarpToRandomPortal, WarpToRandomPortalPayload{
			CharacterId: 100,
			WorldId:     0,
			ChannelId:   1,
			FieldId:     _map.Id(200000000),
		}).
		Build()
	putAcceptEventSaga(t, ctx, s)

	_, ok := p.AcceptEvent(tx, EventKindCharacterMapChanged)
	assert.True(t, ok)
}
```

Add `_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"` to the test file's imports. **Before writing this, read `saga/payloads.go` (or the `sharedsaga` re-export) for the actual field names on `WarpToRandomPortalPayload`** — if the field is not `FieldId`, use whatever the struct declares. The `CharacterId` field is confirmed present (it is the case `ExtractCharacterId` already handles).

- [x] **Step 2: Run test to verify it fails**

Run: `go test ./saga/ -run 'TestAcceptEvent_(CharacterId|NoCharacterId|NoOption)' -v`
Expected: compile failure — `undefined: ForCharacter`, `undefined: SkipReasonCharacterIdMismatch`.

- [x] **Step 3: Add the skip reason constant**

In `saga/event_acceptance.go`, add to the `SkipReason*` const block:

```go
	SkipReasonSagaTerminal       = "saga_terminal"
	// SkipReasonCharacterIdMismatch: a character-scoped event (today only
	// character.map_changed) carried a characterId that does not match the
	// character named by the current step's payload. Expected traffic under
	// the WarpPartyQuestMembersToMap fan-out, which stamps N warps with one
	// transaction id — see saga/handler.go handleWarpPartyQuestMembers.
	SkipReasonCharacterIdMismatch = "character_id_mismatch"
```

- [x] **Step 4: Add the option type and the guard**

In `saga/processor.go`, above the `Processor` interface (or next to `AcceptDecision`):

```go
// acceptOptions carries optional constraints applied by AcceptEvent after the
// action/kind gate. Zero value means "no additional constraint".
type acceptOptions struct {
	characterId    uint32
	hasCharacterId bool
}

// AcceptOption constrains AcceptEvent beyond the action/kind match.
type AcceptOption func(*acceptOptions)

// ForCharacter constrains acceptance to a step whose payload names this
// character. A step whose payload carries no character id (ExtractCharacterId
// returns 0) is left unconstrained, so actions this plan does not touch keep
// their current behaviour exactly.
func ForCharacter(id uint32) AcceptOption {
	return func(o *acceptOptions) {
		o.characterId = id
		o.hasCharacterId = true
	}
}
```

Change the interface declaration (currently `saga/processor.go:63`):

```go
	AcceptEvent(transactionId uuid.UUID, kind EventKind, opts ...AcceptOption) (AcceptDecision, bool)
```

Change the implementation signature (currently `saga/processor.go:397`):

```go
func (p *ProcessorImpl) AcceptEvent(transactionId uuid.UUID, kind EventKind, opts ...AcceptOption) (AcceptDecision, bool) {
	var o acceptOptions
	for _, opt := range opts {
		opt(&o)
	}
```

Then, in the same function, **replace the final `return AcceptDecision{Saga: s, Step: step}, true`** (the last statement, after the `StepAcceptsEvent` block) with:

```go
	// Character-id guard (FR-1.3). Runs last, after the action/kind gate, so
	// a mismatch is reported as its own reason rather than being masked by
	// action_mismatch. maybeWarnUnmatchedEvent is deliberately NOT called
	// here: a cross-character map_changed is expected traffic under the
	// party-quest fan-out, not an anomaly.
	if o.hasCharacterId {
		if want := ExtractCharacterId(step); want != 0 && want != o.characterId {
			LogSkip(p.l, logrus.Fields{
				"transaction_id":     transactionId.String(),
				"step_id":            step.StepId(),
				"step_action":        step.Action(),
				"event_kind":         kind,
				"event_character_id": o.characterId,
				"step_character_id":  want,
			}, SkipReasonCharacterIdMismatch)
			return AcceptDecision{}, false
		}
	}
	return AcceptDecision{Saga: s, Step: step}, true
```

- [x] **Step 5: Update the mock**

In `saga/mock/processor.go`, change the field (~line 42) and the method (~line 206):

```go
	AcceptEventFunc                      func(transactionId uuid.UUID, kind saga.EventKind, opts ...saga.AcceptOption) (saga.AcceptDecision, bool)
```

```go
// AcceptEvent is a mock implementation
func (m *ProcessorMock) AcceptEvent(transactionId uuid.UUID, kind saga.EventKind, opts ...saga.AcceptOption) (saga.AcceptDecision, bool) {
	if m.AcceptEventFunc != nil {
		return m.AcceptEventFunc(transactionId, kind, opts...)
	}
	return saga.AcceptDecision{}, false
}
```

Keep the existing fallback return value exactly as it is in the file — read it before editing and preserve it.

- [x] **Step 6: Run test to verify it passes**

Run: `go test ./saga/ -run 'TestAcceptEvent' -v`
Expected: all `TestAcceptEvent_*` PASS, including the pre-existing ones.

- [x] **Step 7: Verify no call site broke**

Run: `go build ./... && go vet ./...`
Expected: clean. The variadic parameter means every existing `AcceptEvent(tx, kind)` compiles unchanged.

- [x] **Step 8: Commit**

```bash
git add saga/processor.go saga/event_acceptance.go saga/mock/processor.go saga/accept_event_test.go
git commit -m "feat(saga-orchestrator): character-id guard on AcceptEvent (FR-1.3, FR-1.5)"
```

---

## Task 3: Warp steps acknowledge `MAP_CHANGED`

Satisfies FR-1.1, FR-1.2, FR-4.3, FR-4.5, and the FR-4.4 test on the newly-accepting actions. This is the activation task — it makes `WarpToPortal` / `WarpToSavedLocation` complete on the observed warp instead of running to a 30 s `SAGA_TIMEOUT` on every success.

**Files:**
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/event_acceptance.go:202-220`
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/character/consumer.go:68-77`
- Test: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/accept_event_test.go`

**Interfaces:**
- Consumes: `ForCharacter` and the guard from Task 2; `ExtractCharacterId` from Task 1.
- Produces: nothing new — behaviour change only. `handleCharacterMapChangedEvent` becomes the sole `ForCharacter` caller in the service.

**Working directory:** `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`

- [x] **Step 1: Write the failing tests**

Append to `saga/accept_event_test.go`:

```go
// FR-4.3: WarpToPortal completes on a MAP_CHANGED carrying the saga tx id.
func TestAcceptEvent_WarpToPortalCompletesOnMapChanged(t *testing.T) {
	p, _, ctx := newAcceptEventTestProcessor(t)
	tx := uuid.New()
	s := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("portal-action-warp").
		AddStep("warp-100", Pending, WarpToPortal, WarpToPortalPayload{
			CharacterId: 100,
			WorldId:     0,
			ChannelId:   1,
			MapId:       _map.Id(200090510),
			PortalId:    3,
		}).
		Build()
	putAcceptEventSaga(t, ctx, s)

	decision, ok := p.AcceptEvent(tx, EventKindCharacterMapChanged, ForCharacter(100))
	require.True(t, ok, "WarpToPortal must accept map_changed (FR-1.1)")
	assert.Equal(t, "warp-100", decision.Step.StepId())
}

// FR-4.3: same for WarpToSavedLocation.
func TestAcceptEvent_WarpToSavedLocationCompletesOnMapChanged(t *testing.T) {
	p, _, ctx := newAcceptEventTestProcessor(t)
	tx := uuid.New()
	s := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("portal-action-warp-saved-location").
		AddStep("warp-saved-100-FREE_MARKET", Pending, WarpToSavedLocation, WarpToSavedLocationPayload{
			CharacterId:  100,
			WorldId:      0,
			ChannelId:    1,
			LocationType: "FREE_MARKET",
		}).
		Build()
	putAcceptEventSaga(t, ctx, s)

	decision, ok := p.AcceptEvent(tx, EventKindCharacterMapChanged, ForCharacter(100))
	require.True(t, ok, "WarpToSavedLocation must accept map_changed (FR-1.1)")
	assert.Equal(t, "warp-saved-100-FREE_MARKET", decision.Step.StepId())
}

// FR-4.5: the same-map case. The old comment claimed no map_changed fires for
// a portal-to-portal warp within one map; atlas-maps
// warp.ProcessorImpl.ChangeMap emits it unconditionally. The acceptance path
// is map-agnostic, so this asserts no same-map special case was introduced.
func TestAcceptEvent_WarpToPortalSameMapCompletes(t *testing.T) {
	p, _, ctx := newAcceptEventTestProcessor(t)
	tx := uuid.New()
	const sameMap = 200090510
	s := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("portal-action-warp").
		AddStep("warp-100", Pending, WarpToPortal, WarpToPortalPayload{
			CharacterId: 100,
			WorldId:     0,
			ChannelId:   1,
			MapId:       _map.Id(sameMap), // target == origin
			PortalId:    3,
		}).
		Build()
	putAcceptEventSaga(t, ctx, s)

	_, ok := p.AcceptEvent(tx, EventKindCharacterMapChanged, ForCharacter(100))
	assert.True(t, ok, "a same-map warp acknowledges exactly like a cross-map one (FR-4.5)")
}

// FR-4.4 on the newly-accepting action: the party-quest fan-out case this
// whole guard exists for.
func TestAcceptEvent_WarpToPortalRejectsOtherCharacter(t *testing.T) {
	p, hook, ctx := newAcceptEventTestProcessor(t)
	tx := uuid.New()
	s := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("portal-action-warp").
		AddStep("warp-B", Pending, WarpToPortal, WarpToPortalPayload{
			CharacterId: 200, // character B
			WorldId:     0,
			ChannelId:   1,
			MapId:       _map.Id(200090510),
			PortalId:    3,
		}).
		Build()
	putAcceptEventSaga(t, ctx, s)

	// character A's arrival must not complete B's step
	_, ok := p.AcceptEvent(tx, EventKindCharacterMapChanged, ForCharacter(100))
	assert.False(t, ok)

	var found bool
	for _, e := range hook.AllEntries() {
		if e.Data["reason"] == SkipReasonCharacterIdMismatch {
			found = true
		}
	}
	assert.True(t, found, "reason must be character_id_mismatch (FR-1.5)")
}
```

- [x] **Step 2: Run tests to verify they fail**

Run: `go test ./saga/ -run 'TestAcceptEvent_Warp' -v`
Expected: `TestAcceptEvent_WarpToPortalCompletesOnMapChanged`, `..._WarpToSavedLocationCompletesOnMapChanged`, and `..._SameMapCompletes` FAIL — `AcceptEvent` returns false and the log carries `reason=action_mismatch`, because both actions are still `{}` in the acceptance table. `..._RejectsOtherCharacter` may pass for the wrong reason (action mismatch, not the guard) — that is why the log assertion is on `character_id_mismatch` specifically, and it will fail on that assertion.

- [x] **Step 3: Move the two actions into the acceptance block**

In `saga/event_acceptance.go`, **delete** these two lines from the "Fire-and-forget / self-completing actions" block (currently lines 219–220):

```go
	sharedsaga.WarpToPortal:               {},
	sharedsaga.WarpToSavedLocation:        {},
```

- [x] **Step 4: Replace the comment block and add the entries**

Replace the entire comment block plus the `WarpToRandomPortal` entry (currently `saga/event_acceptance.go:202-216`, ending at `sharedsaga.WarpToRandomPortal: {EventKindCharacterMapChanged},`) with:

```go
	// All three warp actions advance on character.map_changed, tagged with the
	// saga transactionId. atlas-maps warp.ProcessorImpl.ChangeMap
	// (services/atlas-maps/atlas.com/maps/character/warp/processor.go) emits
	// MAP_CHANGED *unconditionally* — there is no same-map short-circuit there
	// or in changeMapFromCommand, so a portal-to-portal warp within one map
	// acknowledges exactly like a cross-map one.
	//
	// An earlier revision of this comment asserted the opposite and left
	// WarpToPortal and WarpToSavedLocation self-completing ({}). That claim was
	// false against the code, and every portal warp saga consequently ran to
	// SAGA_TIMEOUT — including the ones that succeeded, which made FAILED
	// worthless as a "the warp did not land" signal (task-184).
	//
	// If a same-map short-circuit is ever added to ChangeMap, these three
	// entries break SILENTLY: the step hangs to timeout again with no error
	// anywhere. This comment is the only coupling record between the two
	// services.
	//
	// Correlation is (transactionId, characterId). handleCharacterMapChangedEvent
	// passes ForCharacter(e.CharacterId) so the WarpPartyQuestMembersToMap
	// fan-out — N warps stamped with one transactionId, see
	// handleWarpPartyQuestMembers — cannot complete a later step belonging to a
	// different character. The residual case it cannot separate is one saga in
	// which the fan-out warps character X and a later step is a WarpToPortal
	// also naming X; if that ever becomes necessary, give the follow-on warp
	// its own saga rather than qualifying the correlation here.
	sharedsaga.WarpToRandomPortal:  {EventKindCharacterMapChanged},
	sharedsaga.WarpToPortal:        {EventKindCharacterMapChanged},
	sharedsaga.WarpToSavedLocation: {EventKindCharacterMapChanged},
```

- [x] **Step 5: Run tests to verify they pass**

Run: `go test ./saga/ -run 'TestAcceptEvent' -v`
Expected: all PASS, including the four new ones and every pre-existing `TestAcceptEvent_*`.

- [x] **Step 6: Wire `ForCharacter` at the sole caller**

In `kafka/consumer/character/consumer.go`, in `handleCharacterMapChangedEvent` (currently line 73), change:

```go
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindCharacterMapChanged); !ok {
```

to:

```go
	// ForCharacter is passed only here. WarpPartyQuestMembersToMap fans N warps
	// out under one transactionId, so a map_changed's characterId must match
	// the character named by the current step's payload before it completes
	// that step (task-184 FR-1.3).
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindCharacterMapChanged,
		saga.ForCharacter(e.CharacterId)); !ok {
```

`StatusEvent[E].CharacterId` is on the envelope (`kafka/message/character/kafka.go:244-250`), so `e.CharacterId` is in scope.

- [x] **Step 7: Run the full service test suite**

Run: `go test -race ./... && go vet ./...`
Expected: both clean. Pay particular attention to `saga/integration_test.go`, `saga/step_event_matching_integration_test.go`, and `saga/handler_test.go` — if any pre-existing test asserted that a `WarpToPortal` step stays pending after a `map_changed`, that assertion encoded the bug and must be updated to expect completion. Do not weaken a test to make it pass; if one fails, read it and decide whether it asserted the old broken behaviour.

- [x] **Step 8: Commit**

```bash
git add saga/event_acceptance.go saga/accept_event_test.go kafka/consumer/character/consumer.go
git commit -m "fix(saga-orchestrator): warp steps complete on MAP_CHANGED instead of timing out (FR-1.1, FR-1.2)"
```

---

## Task 4: Portal operation classification table

Satisfies FR-2.1 and FR-2.2. Replaces the 45-line `switch` in `ExecuteOperation` with a table whose `class` field has an invalid zero value, so a new operation cannot silently default to "does not move the character".

**Files:**
- Create: `services/atlas-portal-actions/atlas.com/portal/script/optable.go`
- Create: `services/atlas-portal-actions/atlas.com/portal/script/optable_test.go`
- Modify: `services/atlas-portal-actions/atlas.com/portal/script/executor.go:37-86` (`ExecuteOperation`)

**Interfaces:**
- Consumes: the thirteen existing `(e *OperationExecutor) execute*` methods — signatures and bodies unchanged.
- Produces, used by Task 7:
  - `func IsMovingOperation(opType string) bool`
  - `opTable map[string]opDef` (package-private) — the single source of truth.
  - `func validateOpTable(tbl map[string]opDef) error` — callable from a test; `init()` panics on its error.

**Working directory for all commands in Tasks 4–9:** `services/atlas-portal-actions/atlas.com/portal`

- [x] **Step 1: Write the failing test**

Create `script/optable_test.go`:

```go
package script

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// FR-2.1/FR-2.2: the three moving operations are the ones whose outcome the
// client is unlocked by a SET_FIELD, not by EnableActions.
func TestOpTable_MovingOperations(t *testing.T) {
	for _, op := range []string{"warp", "warp_to_saved_location", "start_instance_transport"} {
		assert.True(t, IsMovingOperation(op), "[%s] must be classified opClassMoving", op)
	}
}

func TestOpTable_StaticOperations(t *testing.T) {
	for _, op := range []string{
		"play_portal_sound", "drop_message", "show_hint", "block_portal",
		"create_skill", "update_skill", "apply_consumable_effect",
		"cancel_consumable_effect", "save_location", "start_quest",
	} {
		assert.False(t, IsMovingOperation(op), "[%s] must be classified opClassStatic", op)
	}
}

func TestOpTable_UnknownOperationIsNotMoving(t *testing.T) {
	assert.False(t, IsMovingOperation("no_such_operation"),
		"an unknown type is not in the table and is not moving")
}

// FR-2.2: omitting the class is a failure, not a silent default.
func TestValidateOpTable_RejectsUnsetClass(t *testing.T) {
	bad := map[string]opDef{
		"forgot_to_classify": {run: func(e *OperationExecutor, f field.Model, characterId, portalId uint32, op operation.Model) error {
			return nil
		}},
	}
	err := validateOpTable(bad)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "forgot_to_classify")
	assert.Contains(t, err.Error(), "opClass")
}

func TestValidateOpTable_RejectsNilRun(t *testing.T) {
	bad := map[string]opDef{
		"no_body": {class: opClassStatic},
	}
	err := validateOpTable(bad)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no_body")
}

// The production table itself must be valid — this is what init() enforces.
func TestValidateOpTable_ProductionTableIsValid(t *testing.T) {
	assert.NoError(t, validateOpTable(opTable))
}

// Every operation the previous switch handled is still dispatchable.
func TestOpTable_CoversEveryKnownOperation(t *testing.T) {
	want := []string{
		"play_portal_sound", "warp", "drop_message", "show_hint", "block_portal",
		"create_skill", "update_skill", "start_instance_transport",
		"apply_consumable_effect", "cancel_consumable_effect", "save_location",
		"warp_to_saved_location", "start_quest",
	}
	assert.Len(t, opTable, len(want))
	for _, op := range want {
		_, ok := opTable[op]
		assert.True(t, ok, "[%s] must be in opTable", op)
	}
}
```

Add these imports to the test file:

```go
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-script-core/operation"
```

- [x] **Step 2: Run test to verify it fails**

Run: `go test ./script/ -run 'TestOpTable|TestValidateOpTable' -v`
Expected: compile failure — `undefined: IsMovingOperation`, `undefined: opDef`, `undefined: opTable`, `undefined: validateOpTable`, `undefined: opClassStatic`.

- [x] **Step 3: Create the table**

Create `script/optable.go`:

```go
package script

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-script-core/operation"
)

// opClass classifies a portal operation by whether it moves the character.
//
// This exists because of task-184: a portal outcome that moves the character
// must NOT clear the client's exclusive-request flag (EnableActions) — the
// SET_FIELD produced by the warp clears it, and clearing it early while the
// player still stands inside the portal's collision rect makes the GMS v83
// client legitimately re-fire the ENTER request.
//
// The zero value is deliberately invalid so a new table entry that omits the
// class cannot default to "does not move". validateOpTable rejects it and
// init() panics.
type opClass int

const (
	opClassUnset  opClass = iota // invalid — see validateOpTable
	opClassStatic                // leaves the character where they are
	opClassMoving                // dispatches a warp / field change
)

// opDef is one row of the portal operation dispatch table.
type opDef struct {
	class opClass
	run   func(e *OperationExecutor, f field.Model, characterId, portalId uint32, op operation.Model) error
}

// opTable is the single source of truth for both dispatch and classification.
// There is deliberately no second list of "moving" operations anywhere.
var opTable = map[string]opDef{
	"play_portal_sound": {class: opClassStatic, run: func(e *OperationExecutor, f field.Model, characterId, portalId uint32, op operation.Model) error {
		return e.executePlayPortalSound(f, characterId, op)
	}},
	"warp": {class: opClassMoving, run: func(e *OperationExecutor, f field.Model, characterId, portalId uint32, op operation.Model) error {
		return e.executeWarp(f, characterId, op)
	}},
	"drop_message": {class: opClassStatic, run: func(e *OperationExecutor, f field.Model, characterId, portalId uint32, op operation.Model) error {
		return e.executeDropMessage(f, characterId, op)
	}},
	"show_hint": {class: opClassStatic, run: func(e *OperationExecutor, f field.Model, characterId, portalId uint32, op operation.Model) error {
		return e.executeShowHint(f, characterId, op)
	}},
	"block_portal": {class: opClassStatic, run: func(e *OperationExecutor, f field.Model, characterId, portalId uint32, op operation.Model) error {
		return e.executeBlockPortal(f, characterId, portalId, op)
	}},
	"create_skill": {class: opClassStatic, run: func(e *OperationExecutor, f field.Model, characterId, portalId uint32, op operation.Model) error {
		return e.executeCreateSkill(characterId, op)
	}},
	"update_skill": {class: opClassStatic, run: func(e *OperationExecutor, f field.Model, characterId, portalId uint32, op operation.Model) error {
		return e.executeUpdateSkill(characterId, op)
	}},
	"start_instance_transport": {class: opClassMoving, run: func(e *OperationExecutor, f field.Model, characterId, portalId uint32, op operation.Model) error {
		return e.executeStartInstanceTransport(f, characterId, op)
	}},
	"apply_consumable_effect": {class: opClassStatic, run: func(e *OperationExecutor, f field.Model, characterId, portalId uint32, op operation.Model) error {
		return e.executeApplyConsumableEffect(f, characterId, op)
	}},
	"cancel_consumable_effect": {class: opClassStatic, run: func(e *OperationExecutor, f field.Model, characterId, portalId uint32, op operation.Model) error {
		return e.executeCancelConsumableEffect(f, characterId, op)
	}},
	"save_location": {class: opClassStatic, run: func(e *OperationExecutor, f field.Model, characterId, portalId uint32, op operation.Model) error {
		return e.executeSaveLocation(f, characterId, portalId, op)
	}},
	"warp_to_saved_location": {class: opClassMoving, run: func(e *OperationExecutor, f field.Model, characterId, portalId uint32, op operation.Model) error {
		return e.executeWarpToSavedLocation(f, characterId, op)
	}},
	"start_quest": {class: opClassStatic, run: func(e *OperationExecutor, f field.Model, characterId, portalId uint32, op operation.Model) error {
		return e.executeStartQuest(f, characterId, op)
	}},
}

// validateOpTable reports the first structural defect in tbl. Extracted from
// init() so a test can exercise it against a deliberately malformed table.
func validateOpTable(tbl map[string]opDef) error {
	for name, def := range tbl {
		if def.class == opClassUnset {
			return fmt.Errorf("portal operation [%s] has no opClass; classify it as opClassStatic or opClassMoving", name)
		}
		if def.run == nil {
			return fmt.Errorf("portal operation [%s] has no run function", name)
		}
	}
	return nil
}

func init() {
	if err := validateOpTable(opTable); err != nil {
		panic(err.Error())
	}
}

// IsMovingOperation reports whether an operation type dispatches a warp or
// field change. An unknown type is not moving — it is never dispatched either
// (ExecuteOperation warns and returns nil), so no character is moved by it.
func IsMovingOperation(opType string) bool {
	return opTable[opType].class == opClassMoving
}
```

- [x] **Step 4: Replace the switch with a table lookup**

In `script/executor.go`, replace the whole body of `ExecuteOperation` (currently lines 37–86, the `switch op.Type() { ... }`) with:

```go
// ExecuteOperation executes a single operation.
// portalId is the numeric ID of the current portal (for operations like block_portal).
// Dispatch goes through opTable, which is also the classification authority for
// whether an operation moves the character — see optable.go.
func (e *OperationExecutor) ExecuteOperation(f field.Model, characterId uint32, portalId uint32, op operation.Model) error {
	e.l.Debugf("Executing operation [%s] for character [%d]", op.Type(), characterId)

	def, ok := opTable[op.Type()]
	if !ok {
		e.l.Warnf("Unknown operation type [%s] for character [%d]", op.Type(), characterId)
		return nil
	}
	return def.run(e, f, characterId, portalId, op)
}
```

Leave the thirteen `execute*` methods and `ExecuteOperations` untouched in this task.

- [x] **Step 5: Run tests to verify they pass**

Run: `go test ./script/ -run 'TestOpTable|TestValidateOpTable' -v`
Expected: all PASS.

- [x] **Step 6: Run the package suite and build**

Run: `go test ./script/ && go build ./...`
Expected: clean. The pre-existing `script/processor_test.go` still passes — dispatch behaviour for every known type is identical, and the unknown-type `Warn`-and-`return nil` path is preserved.

- [x] **Step 7: Commit**

```bash
git add script/optable.go script/optable_test.go script/executor.go
git commit -m "feat(portal-actions): classify portal operations as moving or static (FR-2.1, FR-2.2)"
```

---

## Task 5: Pending-action safety net for warp sagas

Satisfies FR-2.5 and FR-2.6. This lands **before** the unlock suppression of Task 7 so the net exists before anything relies on it.

Design deviation being implemented (design §4.3, §10): FR-2.5 says `handleEnterCommand` registers the `PendingAction`. It cannot — the saga transaction id is minted inside `executeWarp` / `executeWarpToSavedLocation` and the consumer never sees it. Registration happens in those two methods, mirroring `executeStartInstanceTransport`, which already does exactly this.

**Files:**
- Modify: `services/atlas-portal-actions/atlas.com/portal/action/registry.go`
- Modify: `services/atlas-portal-actions/atlas.com/portal/action/registry_test.go`
- Modify: `services/atlas-portal-actions/atlas.com/portal/script/executor.go` (`executeWarp` ~line 121, `executeWarpToSavedLocation` ~line 562, `executeStartInstanceTransport` ~line 433)

**Interfaces:**
- Consumes: `atlas.TenantRegistry.PutWithTTL` (`libs/atlas-redis/tenant_registry.go:117`), `saga.Builder.SetTransactionId`, `saga.Builder.SetTimeout` (`libs/atlas-saga/builder.go:46`).
- Produces, used by Task 6:
  - `action.PendingAction.Kind string` with `json:"kind"`
  - `const action.KindWarp = "warp"`, `const action.KindTransport = "transport"`
  - `func (r *Registry) AddWithTTL(ctx context.Context, sagaId uuid.UUID, a PendingAction, ttl time.Duration)`

- [x] **Step 1: Write the failing test**

Append to `action/registry_test.go`:

```go
func TestRegistry_AddWithTTL_RoundTrip(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	sagaId := uuid.New()
	pa := PendingAction{
		CharacterId: 1000,
		WorldId:     1,
		ChannelId:   2,
		Kind:        KindWarp,
	}
	GetRegistry().AddWithTTL(ctx, sagaId, pa, 60*time.Second)

	got, found := GetRegistry().Get(ctx, sagaId)
	require.True(t, found)
	assert.Equal(t, KindWarp, got.Kind)
	assert.Equal(t, uint32(1000), got.CharacterId)
}

// An entry written by a pre-deploy replica decodes with Kind == "".
func TestRegistry_LegacyEntryHasEmptyKind(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	sagaId := uuid.New()
	GetRegistry().Add(ctx, sagaId, PendingAction{
		CharacterId:    1000,
		WorldId:        1,
		ChannelId:      2,
		FailureMessage: "legacy",
	})

	got, found := GetRegistry().Get(ctx, sagaId)
	require.True(t, found)
	assert.Equal(t, "", got.Kind, "a legacy entry must decode with an empty Kind")
}
```

Add `"time"` and `"github.com/stretchr/testify/require"` to the test file's imports if not already present.

- [x] **Step 2: Run test to verify it fails**

Run: `go test ./action/ -run 'TestRegistry_AddWithTTL|TestRegistry_Legacy' -v`
Expected: compile failure — `undefined: KindWarp`, `pa.Kind undefined`, `GetRegistry().AddWithTTL undefined`.

- [x] **Step 3: Add `Kind` and `AddWithTTL`**

In `action/registry.go`, add `"time"` to the imports, then:

```go
// Kind* values identify which portal operation created a PendingAction, so the
// failure path can pick a message appropriate to what actually failed.
// The empty value means "written before Kind existed" and is treated as a
// transport, preserving the pre-task-184 message (task-184 FR-2.7).
const (
	KindWarp      = "warp"
	KindTransport = "transport"
)

// PendingAction represents a pending portal action awaiting saga completion
type PendingAction struct {
	CharacterId    uint32     `json:"characterId"`
	WorldId        world.Id   `json:"worldId"`
	ChannelId      channel.Id `json:"channelId"`
	FailureMessage string     `json:"failureMessage"`
	Kind           string     `json:"kind"`
}
```

and, after `Add`:

```go
// AddWithTTL registers a pending action that self-expires. Add writes with no
// expiry, so a dropped COMPLETED event leaks the key forever; warp
// registrations use this instead. The TTL must comfortably exceed the saga's
// own timeout so the failure path can still find the entry.
func (r *Registry) AddWithTTL(ctx context.Context, sagaId uuid.UUID, a PendingAction, ttl time.Duration) {
	t := tenant.MustFromContext(ctx)
	_ = r.reg.PutWithTTL(ctx, t, sagaId, a, ttl)
}
```

- [x] **Step 4: Run test to verify it passes**

Run: `go test ./action/ -v`
Expected: all PASS, including the pre-existing registry tests.

- [x] **Step 5: Register and time-bound the two warp sagas**

In `script/executor.go`, add near the top of the file (after the imports):

```go
const (
	// warpSagaTimeout bounds a portal warp saga. The default is 30s
	// (orchestrator DefaultSagaTimeout); against a ~300ms observed end-to-end
	// warp that is 100x, and it is how long a player whose warp did not land
	// would stay frozen now that the outcome no longer unlocks them
	// eagerly (task-184 FR-2.6). start_instance_transport deliberately keeps
	// the 30s default — it does strictly more work.
	warpSagaTimeout = 5 * time.Second

	// pendingActionTTL bounds the registry entry backing a suppressed unlock.
	// It must exceed warpSagaTimeout by a wide margin so handleStatusEventFailed
	// can still find the entry when the timeout fires.
	pendingActionTTL = 60 * time.Second
)
```

`"time"` and `"atlas-portal-actions/action"` are already imported by this file — confirm before adding.

In `executeWarp`, replace the saga construction and return (currently the `s := saga.NewBuilder()....Build()` / `return e.sagaP.Create(s)` tail) with:

```go
	// The transaction id is minted here so the pending action can be registered
	// under it. If this warp is suppressed from unlocking the client
	// (consumer.go, task-184 FR-2.3), this registration is what lets
	// handleStatusEventFailed release the player when the warp does not land.
	sagaId := uuid.New()
	action.GetRegistry().AddWithTTL(e.ctx, sagaId, action.PendingAction{
		CharacterId: characterId,
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		Kind:        action.KindWarp,
	}, pendingActionTTL)

	s := saga.NewBuilder().
		SetTransactionId(sagaId).
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("portal-action-warp").
		SetTimeout(warpSagaTimeout).
		AddStep(
			fmt.Sprintf("warp-%d", characterId),
			saga.Pending,
			saga.WarpToPortal,
			saga.WarpToPortalPayload{
				CharacterId: characterId,
				WorldId:     f.WorldId(),
				ChannelId:   f.ChannelId(),
				MapId:       _map.Id(mapId),
				PortalId:    portalId,
				PortalName:  portalName,
			},
		).Build()

	return e.sagaP.Create(s)
```

In `executeWarpToSavedLocation`, apply the same shape:

```go
	sagaId := uuid.New()
	action.GetRegistry().AddWithTTL(e.ctx, sagaId, action.PendingAction{
		CharacterId: characterId,
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		Kind:        action.KindWarp,
	}, pendingActionTTL)

	s := saga.NewBuilder().
		SetTransactionId(sagaId).
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("portal-action-warp-saved-location").
		SetTimeout(warpSagaTimeout).
		AddStep(
			fmt.Sprintf("warp-saved-%d-%s", characterId, locationType),
			saga.Pending,
			saga.WarpToSavedLocation,
			saga.WarpToSavedLocationPayload{
				CharacterId:  characterId,
				WorldId:      f.WorldId(),
				ChannelId:    f.ChannelId(),
				LocationType: locationType,
			},
		).Build()

	return e.sagaP.Create(s)
```

- [x] **Step 6: Tag the existing transport registration**

In `executeStartInstanceTransport` (~line 447), add the `Kind` field to the existing `action.GetRegistry().Add(...)` call — leave it on `Add` (no TTL); changing transport's expiry semantics is out of scope and is recorded as a follow-up in context.md:

```go
	action.GetRegistry().Add(e.ctx, sagaId, action.PendingAction{
		CharacterId:    characterId,
		WorldId:        f.WorldId(),
		ChannelId:      f.ChannelId(),
		FailureMessage: failureMessage,
		Kind:           action.KindTransport,
	})
```

- [x] **Step 7: Build and run the package suite**

Run: `go build ./... && go test ./script/ ./action/`
Expected: clean.

- [x] **Step 8: Commit**

```bash
git add action/registry.go action/registry_test.go script/executor.go
git commit -m "feat(portal-actions): register pending action and 5s timeout for warp sagas (FR-2.5, FR-2.6)"
```

---

## Task 6: Warp-appropriate failure message

Satisfies FR-2.7. Today a failed portal warp would tell the player "Unable to board transport at this time."

**Files:**
- Modify: `services/atlas-portal-actions/atlas.com/portal/kafka/consumer/saga/consumer.go` (`resolveFailureMessage` ~line 118; log/comment wording at lines 51–52, 69, 97)
- Create: `services/atlas-portal-actions/atlas.com/portal/kafka/consumer/saga/consumer_test.go`

**Interfaces:**
- Consumes: `action.PendingAction.Kind`, `action.KindWarp`, `action.KindTransport` from Task 5.
- Produces: no new exported symbol. `resolveFailureMessage` keeps its `(action.PendingAction, string) string` signature.

- [x] **Step 1: Write the failing test**

Create `kafka/consumer/saga/consumer_test.go`:

```go
package saga

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"atlas-portal-actions/action"
)

const (
	warpDefaultMessage      = "You cannot move there right now."
	transportDefaultMessage = "Unable to board transport at this time."
)

// FR-2.7: a failed portal warp must not report a transport boarding failure.
func TestResolveFailureMessage_WarpDefault(t *testing.T) {
	pa := action.PendingAction{CharacterId: 1, Kind: action.KindWarp}
	assert.Equal(t, warpDefaultMessage, resolveFailureMessage(pa, ""))
}

func TestResolveFailureMessage_TransportDefault(t *testing.T) {
	pa := action.PendingAction{CharacterId: 1, Kind: action.KindTransport}
	assert.Equal(t, transportDefaultMessage, resolveFailureMessage(pa, ""))
}

// A registry entry written before Kind existed must keep today's text.
func TestResolveFailureMessage_EmptyKindDefaultsToTransport(t *testing.T) {
	pa := action.PendingAction{CharacterId: 1}
	assert.Equal(t, transportDefaultMessage, resolveFailureMessage(pa, ""))
}

// An explicit failureMessage from the script still wins over everything.
func TestResolveFailureMessage_ExplicitMessageWins(t *testing.T) {
	pa := action.PendingAction{CharacterId: 1, Kind: action.KindWarp, FailureMessage: "custom"}
	assert.Equal(t, "custom", resolveFailureMessage(pa, "TRANSPORT_CAPACITY_FULL"))
}

// The transport error codes keep their specific messages, and remain
// unreachable for a warp saga (nothing on the warp path emits them).
func TestResolveFailureMessage_ErrorCodesUnchanged(t *testing.T) {
	pa := action.PendingAction{CharacterId: 1, Kind: action.KindTransport}
	assert.Equal(t, "The transport is currently full. Please try again later.",
		resolveFailureMessage(pa, "TRANSPORT_CAPACITY_FULL"))
	assert.Equal(t, "You are already on a transport.",
		resolveFailureMessage(pa, "TRANSPORT_ALREADY_IN_TRANSIT"))
	assert.Equal(t, "Transport service is currently unavailable.",
		resolveFailureMessage(pa, "TRANSPORT_ROUTE_NOT_FOUND"))
	assert.Equal(t, "Transport service is currently unavailable.",
		resolveFailureMessage(pa, "TRANSPORT_SERVICE_ERROR"))
}
```

- [x] **Step 2: Run test to verify it fails**

Run: `go test ./kafka/consumer/saga/ -v`
Expected: `TestResolveFailureMessage_WarpDefault` FAILS — actual is `"Unable to board transport at this time."`. The others PASS.

- [x] **Step 3: Switch the default on `Kind`**

In `kafka/consumer/saga/consumer.go`, replace `resolveFailureMessage`'s `default:` arm:

```go
// resolveFailureMessage determines the appropriate failure message based on
// error code, falling back to a default chosen by the kind of portal action
// that failed (task-184 FR-2.7).
func resolveFailureMessage(pendingAction action.PendingAction, errorCode string) string {
	// Use custom failure message if provided
	if pendingAction.FailureMessage != "" {
		return pendingAction.FailureMessage
	}

	// Default messages based on error code. These codes are emitted only on the
	// transport path; a warp saga never produces them.
	switch errorCode {
	case "TRANSPORT_CAPACITY_FULL":
		return "The transport is currently full. Please try again later."
	case "TRANSPORT_ALREADY_IN_TRANSIT":
		return "You are already on a transport."
	case "TRANSPORT_ROUTE_NOT_FOUND":
		return "Transport service is currently unavailable."
	case "TRANSPORT_SERVICE_ERROR":
		return "Transport service is currently unavailable."
	}

	// No specific code: pick by what actually failed. An empty Kind means the
	// entry was written by a replica predating the field, so it keeps the
	// pre-existing transport text.
	switch pendingAction.Kind {
	case action.KindWarp:
		return "You cannot move there right now."
	default:
		return "Unable to board transport at this time."
	}
}
```

- [x] **Step 4: Correct the misleading log wording**

In the same file, the two handlers now serve warps as well as transports. Change:

- line ~51 comment `// For transport sagas, the warp already happened - just cleanup` → `// For portal action sagas (warp or transport), the move already happened - just cleanup`
- line ~69 log message `"Transport saga completed, cleaning up pending action"` → `"Portal action saga completed, cleaning up pending action"`
- line ~97 log message `"Transport saga failed, sending failure message to character"` → `"Portal action saga failed, sending failure message to character"`

Add `"kind": pendingAction.Kind,` to the `logrus.Fields` of the failure log at ~line 92 so the two paths are distinguishable in logs.

Also change `SetInitiatedBy("portal-action-transport-failure")` in `sendFailureMessage` to `SetInitiatedBy("portal-action-failure")` — it is no longer transport-specific.

- [x] **Step 5: Run test to verify it passes**

Run: `go test ./kafka/consumer/saga/ -v`
Expected: all PASS.

- [x] **Step 6: Commit**

```bash
git add kafka/consumer/saga/consumer.go kafka/consumer/saga/consumer_test.go
git commit -m "fix(portal-actions): warp failures report a warp message, not a transport one (FR-2.7)"
```

---

## Task 7: Suppress the unlock for outcomes that move the character

Satisfies FR-2.3, FR-2.4, and FR-4.1. **This is the actual fix.**

Design deviation being implemented (design §4.2, §10): the PRD phrases the condition as "the matched outcome's operations include at least one moving operation" and preserves the error branch verbatim. This implements the strictly safer "a moving operation was **successfully dispatched**", applied to the error branch too. A warp that failed before creating its saga still unlocks (there is no saga to fail and release the player); a dispatched warp followed by an unrelated operation error does not double-unlock.

**Files:**
- Modify: `services/atlas-portal-actions/atlas.com/portal/script/executor.go` (`ExecuteOperations`, ~line 88)
- Modify: `services/atlas-portal-actions/atlas.com/portal/script/model.go` (`ProcessResult`, ~line 76)
- Modify: `services/atlas-portal-actions/atlas.com/portal/script/processor.go` (`Process`, ~lines 155-180)
- Modify: `services/atlas-portal-actions/atlas.com/portal/script/consumer.go`
- Create: `services/atlas-portal-actions/atlas.com/portal/script/consumer_test.go`
- Create: `services/atlas-portal-actions/atlas.com/portal/script/executor_test.go`

**Interfaces:**
- Consumes: `IsMovingOperation` from Task 4; `portalsaga.Processor` (interface — `Create(sharedsaga.Saga) error`).
- Produces, used by Task 9:
  - `func (e *OperationExecutor) ExecuteOperations(f field.Model, characterId, portalId uint32, ops []operation.Model) (movedCharacter bool, err error)`
  - `ProcessResult.CharacterMoved bool`
  - Package-level seams in `consumer.go`: `newScriptProcessorFn func(logrus.FieldLogger, context.Context, *gorm.DB) Processor` and `enableActionsFn func(logrus.FieldLogger) func(context.Context) func(channel.Model, uint32)`
  - Test-only executor constructor `newOperationExecutorWithSaga(l, ctx, sagaP)`.

- [x] **Step 1: Write the failing executor test**

Create `script/executor_test.go`:

```go
package script

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
	"github.com/Chronicle20/atlas/libs/atlas-script-core/operation"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// fakeSagaProcessor records the sagas an executor tries to create and can be
// told to fail.
type fakeSagaProcessor struct {
	created []sharedsaga.Saga
	err     error
}

func (f *fakeSagaProcessor) Create(s sharedsaga.Saga) error {
	if f.err != nil {
		return f.err
	}
	f.created = append(f.created, s)
	return nil
}

func executorTestCtx(t *testing.T) context.Context {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return tenant.WithContext(context.Background(), tm)
}

func testField() field.Model {
	return field.NewBuilder(0, 1, 200090510).SetInstance(uuid.Nil).Build()
}

func mustOp(t *testing.T, opType string, params map[string]string) operation.Model {
	t.Helper()
	b := operation.NewBuilder().SetType(opType)
	if params != nil {
		b = b.SetParams(params)
	}
	m, err := b.Build()
	require.NoError(t, err)
	return m
}

func newTestExecutor(t *testing.T, sp *fakeSagaProcessor) (*OperationExecutor, context.Context) {
	t.Helper()
	logger, _ := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	ctx := executorTestCtx(t)
	return newOperationExecutorWithSaga(logger, ctx, sp), ctx
}

// A dispatched warp reports movedCharacter == true.
func TestExecuteOperations_WarpReportsMoved(t *testing.T) {
	sp := &fakeSagaProcessor{}
	e, _ := newTestExecutor(t, sp)

	moved, err := e.ExecuteOperations(testField(), 100, 3, []operation.Model{
		mustOp(t, "warp", map[string]string{"mapId": "200090500", "portalId": "1"}),
	})
	require.NoError(t, err)
	assert.True(t, moved)
	require.Len(t, sp.created, 1)
}

// warp_to_saved_location likewise.
func TestExecuteOperations_WarpToSavedLocationReportsMoved(t *testing.T) {
	sp := &fakeSagaProcessor{}
	e, _ := newTestExecutor(t, sp)

	moved, err := e.ExecuteOperations(testField(), 100, 3, []operation.Model{
		mustOp(t, "warp_to_saved_location", map[string]string{"locationType": "FREE_MARKET"}),
	})
	require.NoError(t, err)
	assert.True(t, moved)
}

// A static-only outcome does not report a move.
func TestExecuteOperations_StaticOnlyReportsNotMoved(t *testing.T) {
	sp := &fakeSagaProcessor{}
	e, _ := newTestExecutor(t, sp)

	moved, err := e.ExecuteOperations(testField(), 100, 3, []operation.Model{
		mustOp(t, "play_portal_sound", nil),
	})
	require.NoError(t, err)
	assert.False(t, moved)
}

// FR-2.3 strengthened: a warp that fails BEFORE its saga is created reports
// movedCharacter == false, so the caller still unlocks the client. There is no
// saga in flight to fail and release them.
func TestExecuteOperations_WarpDispatchFailureReportsNotMoved(t *testing.T) {
	sp := &fakeSagaProcessor{err: errors.New("kafka unavailable")}
	e, _ := newTestExecutor(t, sp)

	moved, err := e.ExecuteOperations(testField(), 100, 3, []operation.Model{
		mustOp(t, "warp", map[string]string{"mapId": "200090500"}),
	})
	require.Error(t, err)
	assert.False(t, moved, "a warp that never dispatched has not moved the character")
}

// A warp validation error (missing mapId) also reports not-moved.
func TestExecuteOperations_WarpParamErrorReportsNotMoved(t *testing.T) {
	sp := &fakeSagaProcessor{}
	e, _ := newTestExecutor(t, sp)

	moved, err := e.ExecuteOperations(testField(), 100, 3, []operation.Model{
		mustOp(t, "warp", map[string]string{"portalId": "1"}),
	})
	require.Error(t, err)
	assert.False(t, moved)
}

// A successful warp followed by a failing static operation still reports moved:
// the warp is in flight and its SET_FIELD will unlock the client.
func TestExecuteOperations_MovedStickyAcrossLaterError(t *testing.T) {
	sp := &fakeSagaProcessor{}
	e, _ := newTestExecutor(t, sp)

	moved, err := e.ExecuteOperations(testField(), 100, 3, []operation.Model{
		mustOp(t, "warp", map[string]string{"mapId": "200090500"}),
		mustOp(t, "start_quest", nil), // missing required params -> error
	})
	require.Error(t, err)
	assert.True(t, moved, "the warp already dispatched; the SET_FIELD will unlock the client")
}
```

Before running, read `executeStartQuest` to confirm it errors on a missing required param; if it does not, substitute an operation in that last test that does (e.g. `"warp_to_saved_location"` with no `locationType`, which is confirmed to error). Do not weaken the assertion.

- [x] **Step 2: Run test to verify it fails**

Run: `go test ./script/ -run TestExecuteOperations -v`
Expected: compile failure — `newOperationExecutorWithSaga` undefined, and `ExecuteOperations` returns one value, not two.

- [x] **Step 3: Add the test-only constructor and change `ExecuteOperations`**

In `script/executor.go`, add next to `NewOperationExecutor`:

```go
// newOperationExecutorWithSaga builds an executor over an injected saga
// processor. Used by tests to observe dispatched sagas without touching Kafka.
func newOperationExecutorWithSaga(l logrus.FieldLogger, ctx context.Context, sagaP portalsaga.Processor) *OperationExecutor {
	return &OperationExecutor{l: l, ctx: ctx, sagaP: sagaP}
}
```

Replace `ExecuteOperations` (currently lines 88–96):

```go
// ExecuteOperations executes multiple operations in order, stopping at the
// first error.
//
// portalId is the numeric ID of the current portal (for operations like
// block_portal).
//
// movedCharacter reports whether at least one MOVING operation was
// SUCCESSFULLY dispatched (its saga was created). The caller uses this to
// decide whether the client is already going to be unlocked by the resulting
// SET_FIELD — see consumer.go and task-184 prd.md §1.1.
//
// The distinction between "declared" and "successfully dispatched" is
// load-bearing: a warp that failed before creating its saga has no saga to
// fail and release the player, so the caller MUST still unlock them.
func (e *OperationExecutor) ExecuteOperations(f field.Model, characterId uint32, portalId uint32, ops []operation.Model) (bool, error) {
	movedCharacter := false
	for _, op := range ops {
		if err := e.ExecuteOperation(f, characterId, portalId, op); err != nil {
			return movedCharacter, err
		}
		if IsMovingOperation(op.Type()) {
			movedCharacter = true
		}
	}
	return movedCharacter, nil
}
```

The `movedCharacter` flag is set only *after* `ExecuteOperation` returned `nil`, which for the three moving operations means `sagaP.Create` returned `nil`. It is a plain local, so nothing races even though `OperationExecutor` is constructed per request.

- [x] **Step 4: Run the executor test to verify it passes**

Run: `go test ./script/ -run TestExecuteOperations -v`
Expected: all PASS.

- [x] **Step 5: Thread `CharacterMoved` through `ProcessResult`**

In `script/model.go`, add the field:

```go
// ProcessResult represents the result of processing a portal script
type ProcessResult struct {
	Allow       bool
	MatchedRule string
	Operations  []operation.Model
	// CharacterMoved reports that a moving operation was successfully
	// dispatched, so the client will be unlocked by the resulting SET_FIELD
	// and MUST NOT be unlocked again here (task-184 FR-2.3).
	CharacterMoved bool
	Error          error
}
```

In `script/processor.go` `Process`, change the operation-execution block (currently lines 160–178) to:

```go
			// Execute operations (pass portalId for operations like block_portal)
			outcome := rule.OnMatch()
			movedCharacter := false
			if len(outcome.Operations()) > 0 {
				var err error
				movedCharacter, err = p.executor.ExecuteOperations(f, characterId, portalId, outcome.Operations())
				if err != nil {
					p.l.WithError(err).Errorf("Failed to execute operations for rule [%s]", rule.Id())
					return ProcessResult{
						Allow:          outcome.Allow(),
						MatchedRule:    rule.Id(),
						Operations:     outcome.Operations(),
						CharacterMoved: movedCharacter,
						Error:          fmt.Errorf("operation execution failed: %w", err),
					}
				}
			}

			return ProcessResult{
				Allow:          outcome.Allow(),
				MatchedRule:    rule.Id(),
				Operations:     outcome.Operations(),
				CharacterMoved: movedCharacter,
				Error:          nil,
			}
```

The `no_script` and `no_match` returns are left untouched — they carry `CharacterMoved == false` by zero value, which is correct.

- [x] **Step 6: Write the failing consumer test**

Create `script/consumer_test.go`:

```go
package script

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// fakeProcessor is a script.Processor whose Process return value the test
// controls, and which records that it was called at all.
type fakeProcessor struct {
	Processor
	calls  int
	result ProcessResult
}

func (f *fakeProcessor) Process(_ field.Model, _ uint32, _ string, _ uint32) ProcessResult {
	f.calls++
	return f.result
}

// consumerTestHarness swaps the package seams and restores them on cleanup,
// returning a pointer to the unlock counter and the fake processor.
func consumerTestHarness(t *testing.T, result ProcessResult) (*int, *fakeProcessor, logrus.FieldLogger, context.Context) {
	t.Helper()
	unlocks := 0
	fp := &fakeProcessor{result: result}

	prevProc := newScriptProcessorFn
	prevUnlock := enableActionsFn
	newScriptProcessorFn = func(_ logrus.FieldLogger, _ context.Context, _ *gorm.DB) Processor { return fp }
	enableActionsFn = func(_ logrus.FieldLogger) func(context.Context) func(channel.Model, uint32) {
		return func(_ context.Context) func(channel.Model, uint32) {
			return func(_ channel.Model, _ uint32) { unlocks++ }
		}
	}
	t.Cleanup(func() {
		newScriptProcessorFn = prevProc
		enableActionsFn = prevUnlock
	})

	logger, _ := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return &unlocks, fp, logger, tenant.WithContext(context.Background(), tm)
}

func testEnterCommand() commandEvent[enterBody] {
	return commandEvent[enterBody]{
		WorldId:   0,
		ChannelId: 1,
		MapId:     200090510,
		Instance:  uuid.Nil,
		PortalId:  3,
		Type:      "ENTER",
		Body:      enterBody{CharacterId: 100, PortalName: "undodraco"},
	}
}

// FR-4.1: an outcome that moved the character must NOT unlock the client.
// The SET_FIELD produced by the warp clears m_bExclRequestSent; unlocking here
// while the warp is in flight is what makes the client re-fire.
func TestHandleEnterCommand_MovingOutcomeDoesNotUnlock(t *testing.T) {
	unlocks, _, l, ctx := consumerTestHarness(t, ProcessResult{
		Allow: true, MatchedRule: "r1", CharacterMoved: true,
	})
	handleEnterCommand(l, ctx, nil, testEnterCommand())
	assert.Equal(t, 0, *unlocks, "a moving outcome must not emit EnableActions")
}

// FR-4.1: an outcome that did not move the character still unlocks.
func TestHandleEnterCommand_StaticOutcomeUnlocks(t *testing.T) {
	unlocks, _, l, ctx := consumerTestHarness(t, ProcessResult{
		Allow: true, MatchedRule: "r1", CharacterMoved: false,
	})
	handleEnterCommand(l, ctx, nil, testEnterCommand())
	assert.Equal(t, 1, *unlocks)
}

// FR-2.4: every non-moving outcome keeps unlocking exactly as before.
func TestHandleEnterCommand_NonMovingOutcomesAllUnlock(t *testing.T) {
	for name, result := range map[string]ProcessResult{
		"denied":    {Allow: false, MatchedRule: "r1"},
		"no_script": {Allow: true, MatchedRule: "no_script"},
		"no_match":  {Allow: false, MatchedRule: "no_match"},
		"allowed_no_ops": {Allow: true, MatchedRule: "r1"},
	} {
		t.Run(name, func(t *testing.T) {
			unlocks, _, l, ctx := consumerTestHarness(t, result)
			handleEnterCommand(l, ctx, nil, testEnterCommand())
			assert.Equal(t, 1, *unlocks)
		})
	}
}

// FR-2.3 strengthened: a rule error where nothing was dispatched still unlocks,
// so the player is never frozen with no saga to release them.
func TestHandleEnterCommand_ErrorWithoutMoveUnlocks(t *testing.T) {
	unlocks, _, l, ctx := consumerTestHarness(t, ProcessResult{
		Allow: false, MatchedRule: "r1", CharacterMoved: false,
		Error: errors.New("rule evaluation failed"),
	})
	handleEnterCommand(l, ctx, nil, testEnterCommand())
	assert.Equal(t, 1, *unlocks)
}

// FR-2.3 strengthened: a dispatched warp followed by an unrelated operation
// error must NOT double-unlock — the warp is in flight.
func TestHandleEnterCommand_ErrorAfterMoveDoesNotUnlock(t *testing.T) {
	unlocks, _, l, ctx := consumerTestHarness(t, ProcessResult{
		Allow: true, MatchedRule: "r1", CharacterMoved: true,
		Error: errors.New("operation execution failed"),
	})
	handleEnterCommand(l, ctx, nil, testEnterCommand())
	assert.Equal(t, 0, *unlocks)
}
```

- [x] **Step 7: Run the consumer test to verify it fails**

Run: `go test ./script/ -run TestHandleEnterCommand -v`
Expected: compile failure — `newScriptProcessorFn` and `enableActionsFn` undefined.

- [x] **Step 8: Add the seams and the conditional unlock**

In `script/consumer.go`, add above `handleEnterCommand`:

```go
// Package seams. Production wiring is unchanged; tests substitute these to
// observe handleEnterCommand's unlock decision without Kafka or a database.
var (
	newScriptProcessorFn = func(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
		return NewProcessor(l, ctx, db)
	}
	enableActionsFn = character.EnableActions
)
```

Replace the body of `handleEnterCommand` from the `processor :=` line to the end:

```go
	// Create processor with tenant context from Kafka message
	processor := newScriptProcessorFn(l, ctx, db)

	// Process the portal script (pass numeric portalId for use in operations like block_portal)
	result := processor.Process(f, c.Body.CharacterId, c.Body.PortalName, c.PortalId)

	if result.Error != nil {
		l.WithError(result.Error).Errorf("Failed to process portal script [%s] for character [%d]",
			c.Body.PortalName, c.Body.CharacterId)
	} else {
		l.Debugf("Portal script [%s] result: allow=%t, matchedRule=%s, characterMoved=%t",
			c.Body.PortalName, result.Allow, result.MatchedRule, result.CharacterMoved)
	}

	// An outcome that dispatched a warp is unlocked by the SET_FIELD that warp
	// produces — CWvsContext::OnGameStageChanged clears m_bExclRequestSent on
	// every set_stage. Clearing it HERE, while the warp is still in flight and
	// the player still overlaps the portal's collision rect, is what makes the
	// GMS v83 client legitimately re-fire the ENTER request and execute the
	// whole rule a second time. See
	// docs/tasks/task-184-portal-enter-double-execute/prd.md §1.1.
	//
	// If the warp never lands, the saga's 5s timeout fails it and
	// kafka/consumer/saga/consumer.go handleStatusEventFailed unlocks the
	// player from the PendingAction registered in executor.go.
	//
	// CharacterMoved means "successfully dispatched", not "declared": a warp
	// that failed before creating its saga leaves this false, so the player is
	// unlocked here rather than waiting on a saga that does not exist.
	if result.CharacterMoved {
		return
	}
	enableActionsFn(l)(ctx)(ch, c.Body.CharacterId)
}
```

Remove the now-unused `character` import only if nothing else in the file references it — `enableActionsFn = character.EnableActions` does, so keep it.

- [x] **Step 9: Run tests to verify they pass**

Run: `go test ./script/ -v`
Expected: all PASS, including `TestHandleEnterCommand_*`, `TestExecuteOperations_*`, `TestOpTable_*`, and every pre-existing test in the package.

- [x] **Step 10: Build and vet**

Run: `go build ./... && go vet ./...`
Expected: clean.

- [x] **Step 11: Commit**

```bash
git add script/executor.go script/executor_test.go script/model.go script/processor.go script/consumer.go script/consumer_test.go
git commit -m "fix(portal-actions): do not unlock a client whose warp is in flight (FR-2.3, FR-2.4, FR-4.1)"
```

---

## Task 8: Redis dedupe gate package

Satisfies FR-3.2, FR-3.3, FR-3.4, FR-3.6. Defence in depth: the client is *designed* to re-fire while a player stands in a portal rect, so any future unlock-shaped regression in an outcome path re-opens the same hole.

**Files:**
- Create: `services/atlas-portal-actions/atlas.com/portal/dedupe/gate.go`
- Create: `services/atlas-portal-actions/atlas.com/portal/dedupe/gate_test.go`

**Interfaces:**
- Consumes: `atlas.NewLock(client, namespace)`, `(*Lock).AcquireWithTTL(ctx, key, ttl) (bool, error)` (`libs/atlas-redis/lock.go:60`), `atlas.TenantKey(t)`, `atlas.CompositeKey(parts...)` (`libs/atlas-redis/keys.go:33,55`), `tenant.MustFromContext`.
- Produces, used by Task 9:
  - `type Key struct { CharacterId uint32; MapId _map.Id; Instance uuid.UUID; PortalId uint32 }`
  - `type Gate interface { Allow(ctx context.Context, k Key) bool }`
  - `func InitGate(client *goredis.Client)`
  - `func GetGate() Gate`

- [x] **Step 1: Write the failing test**

Create `dedupe/gate_test.go`:

```go
package dedupe

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func setupGate(t *testing.T) *miniredis.Miniredis {
	t.Helper()
	mr := miniredis.RunT(t)
	InitGate(goredis.NewClient(&goredis.Options{Addr: mr.Addr()}))
	t.Cleanup(func() { gate = nil })
	return mr
}

func gateCtx(t *testing.T) context.Context {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return tenant.WithContext(context.Background(), tm)
}

func testKey() Key {
	return Key{CharacterId: 100, MapId: 200090510, Instance: uuid.Nil, PortalId: 3}
}

func nullLogger(t *testing.T) (logrus.FieldLogger, *logtest.Hook) {
	t.Helper()
	l, hook := logtest.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)
	return l, hook
}

// FR-3.1/FR-3.4: the first ENTER passes, an identical second one inside the
// window does not.
func TestGate_DropsDuplicateInsideWindow(t *testing.T) {
	setupGate(t)
	ctx := gateCtx(t)
	l, _ := nullLogger(t)

	assert.True(t, GetGate().Allow(l, ctx, testKey()), "first ENTER passes")
	assert.False(t, GetGate().Allow(l, ctx, testKey()), "identical ENTER inside the TTL is dropped")
}

// FR-3.4: after the TTL elapses the same key passes again. The lock is never
// released explicitly — TTL expiry IS the release.
func TestGate_AllowsAfterTTL(t *testing.T) {
	mr := setupGate(t)
	ctx := gateCtx(t)
	l, _ := nullLogger(t)

	require.True(t, GetGate().Allow(l, ctx, testKey()))
	mr.FastForward(enterGateTTL + time.Second)
	assert.True(t, GetGate().Allow(l, ctx, testKey()), "the gate reopens once the TTL expires")
}

// Different key components are different gates.
func TestGate_DistinctKeysDoNotCollide(t *testing.T) {
	setupGate(t)
	ctx := gateCtx(t)
	l, _ := nullLogger(t)

	base := testKey()
	require.True(t, GetGate().Allow(l, ctx, base))

	other := base
	other.PortalId = 4
	assert.True(t, GetGate().Allow(l, ctx, other), "a different portal is a different gate")

	other = base
	other.CharacterId = 101
	assert.True(t, GetGate().Allow(l, ctx, other), "a different character is a different gate")

	other = base
	other.MapId = 200090500
	assert.True(t, GetGate().Allow(l, ctx, other), "a different map is a different gate")

	other = base
	other.Instance = uuid.New()
	assert.True(t, GetGate().Allow(l, ctx, other), "a different instance is a different gate")
}

// NFR multi-tenancy: two tenants with identical character/map/portal must not
// share a gate.
func TestGate_TenantIsolation(t *testing.T) {
	setupGate(t)
	l, _ := nullLogger(t)

	ctxA := gateCtx(t)
	ctxB := gateCtx(t) // a different tenant uuid

	require.True(t, GetGate().Allow(l, ctxA, testKey()))
	assert.True(t, GetGate().Allow(l, ctxB, testKey()),
		"a second tenant's identical ENTER must not be gated by the first")
}

// FR-3.6: a Redis failure fails OPEN. Losing Redis must not make every portal
// in the game unusable.
func TestGate_FailsOpenOnRedisError(t *testing.T) {
	mr := setupGate(t)
	ctx := gateCtx(t)
	l, _ := nullLogger(t)

	mr.Close() // every subsequent command errors
	assert.True(t, GetGate().Allow(l, ctx, testKey()), "a Redis error must not block portal traversal")
}

// FR-3.6: an uninitialised gate (unit tests, misconfigured startup) allows.
func TestGate_NilGateAllows(t *testing.T) {
	gate = nil
	ctx := gateCtx(t)
	l, _ := nullLogger(t)
	assert.True(t, GetGate().Allow(l, ctx, testKey()))
}

// FR-3.5: a dropped duplicate is logged at Debug with the key components.
func TestGate_LogsDroppedDuplicateAtDebug(t *testing.T) {
	setupGate(t)
	ctx := gateCtx(t)
	l, hook := nullLogger(t)

	require.True(t, GetGate().Allow(l, ctx, testKey()))
	require.False(t, GetGate().Allow(l, ctx, testKey()))

	var found bool
	for _, e := range hook.AllEntries() {
		if e.Level == logrus.DebugLevel && e.Data["portal_id"] == uint32(3) {
			found = true
			assert.Equal(t, uint32(100), e.Data["character_id"])
			assert.NotEmpty(t, e.Data["tenant_id"])
		}
	}
	assert.True(t, found, "the drop must be logged at Debug with the key components")
}
```

Add `"time"` to the test imports.

- [x] **Step 2: Run test to verify it fails**

Run: `go test ./dedupe/ -v`
Expected: the package does not exist — `no Go files in .../dedupe` or undefined symbols.

- [x] **Step 3: Create the gate**

Create `dedupe/gate.go`:

```go
// Package dedupe gates duplicate portal ENTER commands.
//
// The GMS v83 client re-checks portal collision every frame while the player
// overlaps a portal's rect (CUserLocal::CheckPortal_Collision), and the
// scripted-portal path has no minimum re-send interval — unlike
// CField::SendTransferFieldRequest, which refuses to re-send within 500ms.
// The only thing stopping a re-send is m_bExclRequestSent, which the server
// clears via EnableActions. task-184's primary fix is to stop clearing that
// flag while a warp is in flight; this gate is defence in depth against any
// future unlock-shaped regression in an outcome path nobody is looking at.
//
// It fails OPEN: losing Redis must not make every portal in the game unusable.
package dedupe

import (
	"context"
	"strconv"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// enterGateTTL is how long one portal ENTER closes the gate for.
// Comfortably above the client's own 500ms floor for non-scripted portals, and
// comfortably below any interval at which a player could legitimately intend
// to re-enter the same portal in the same map instance (task-184 FR-3.4).
const enterGateTTL = 2 * time.Second

// lockNamespace is the Redis key namespace for the gate. Disjoint from every
// existing namespace in this service.
const lockNamespace = "portal-enter"

// Key identifies one portal entry attempt. Tenant is NOT a field: it comes
// from the context via the standard libs/atlas-redis tenant key helper.
type Key struct {
	CharacterId uint32
	MapId       _map.Id
	Instance    uuid.UUID
	PortalId    uint32
}

// Gate decides whether a portal ENTER should be processed.
type Gate interface {
	// Allow reports whether this ENTER should be processed. A duplicate inside
	// the TTL window returns false. Any Redis error returns true (fail open).
	Allow(l logrus.FieldLogger, ctx context.Context, k Key) bool
}

type redisGate struct {
	lock *atlas.Lock
}

// nilGate is returned when the gate was never initialised (unit tests, or a
// startup path that skipped InitGate). It allows everything.
type nilGate struct{}

func (nilGate) Allow(_ logrus.FieldLogger, _ context.Context, _ Key) bool { return true }

var gate Gate

// InitGate wires the gate to Redis. Call once at startup, beside
// action.InitRegistry.
func InitGate(client *goredis.Client) {
	gate = &redisGate{lock: atlas.NewLock(client, lockNamespace)}
}

// GetGate returns the process gate, or a permissive gate if InitGate was never
// called. It never returns nil, so callers need no nil check (FR-3.6).
func GetGate() Gate {
	if gate == nil {
		return nilGate{}
	}
	return gate
}

// redisKey composes the tenant-scoped key. Lock is not tenant-aware — its
// lockKey is namespacedKey(namespace, "_lock", key) with no tenant segment —
// so the tenant is composed into the caller-supplied key using the library's
// own TenantKey and CompositeKey helpers rather than hand-rolled string
// concatenation (task-184 FR-3.3, design §5.1).
func redisKey(t tenant.Model, k Key) string {
	return atlas.CompositeKey(
		atlas.TenantKey(t),
		strconv.FormatUint(uint64(k.CharacterId), 10),
		strconv.FormatUint(uint64(k.MapId), 10),
		k.Instance.String(),
		strconv.FormatUint(uint64(k.PortalId), 10),
	)
}

func (g *redisGate) Allow(l logrus.FieldLogger, ctx context.Context, k Key) bool {
	t := tenant.MustFromContext(ctx)
	rk := redisKey(t, k)

	// The lock is never released — TTL expiry IS the release. A successful
	// portal entry keeps the gate closed for the full window.
	acquired, err := g.lock.AcquireWithTTL(ctx, rk, enterGateTTL)
	if err != nil {
		l.WithError(err).WithFields(logrus.Fields{
			"tenant_id":    t.Id().String(),
			"character_id": k.CharacterId,
			"portal_id":    k.PortalId,
		}).Warn("Portal enter dedupe gate unavailable, processing command. Duplicate ENTER commands are not being suppressed.")
		return true
	}
	if !acquired {
		l.WithFields(logrus.Fields{
			"tenant_id":    t.Id().String(),
			"character_id": k.CharacterId,
			"map_id":       uint32(k.MapId),
			"instance":     k.Instance.String(),
			"portal_id":    k.PortalId,
		}).Debug("Dropping duplicate portal enter command inside the dedupe window.")
		return false
	}
	return true
}
```

- [x] **Step 4: Run tests to verify they pass**

Run: `go test ./dedupe/ -v`
Expected: all PASS.

If `TestGate_FailsOpenOnRedisError` does not error after `mr.Close()`, `miniredis` may be returning a connection error asynchronously — in that case point the client at a closed port instead: `InitGate(goredis.NewClient(&goredis.Options{Addr: "127.0.0.1:1"}))`. Do not weaken the fail-open assertion.

- [x] **Step 5: Verify the Redis key guard stays clean**

Run from the worktree root: `tools/redis-key-guard.sh`
Expected: exit 0. The gate uses `atlas.Lock`, never a raw keyed `go-redis` command.

- [x] **Step 6: Commit**

```bash
git add dedupe/gate.go dedupe/gate_test.go
git commit -m "feat(portal-actions): fail-open Redis dedupe gate for portal ENTER (FR-3.2, FR-3.3, FR-3.4, FR-3.6)"
```

---

## Task 9: Wire the dedupe gate into the ENTER path

Satisfies FR-3.1, FR-3.5, FR-4.2. The gate must be evaluated *before* any rule evaluation or operation execution.

**Files:**
- Modify: `services/atlas-portal-actions/atlas.com/portal/main.go` (~line 51, beside `action.InitRegistry(rc)`)
- Modify: `services/atlas-portal-actions/atlas.com/portal/script/consumer.go`
- Modify: `services/atlas-portal-actions/atlas.com/portal/script/consumer_test.go`

**Interfaces:**
- Consumes: `dedupe.Gate`, `dedupe.Key`, `dedupe.InitGate`, `dedupe.GetGate` from Task 8; the `newScriptProcessorFn` / `enableActionsFn` seams from Task 7.
- Produces: a third package seam `gateFn func() dedupe.Gate` in `script/consumer.go`.

- [x] **Step 1: Write the failing test**

Append to `script/consumer_test.go`:

```go
// fakeGate returns a fixed decision and counts calls.
type fakeGate struct {
	allow bool
	calls int
}

func (g *fakeGate) Allow(_ logrus.FieldLogger, _ context.Context, _ dedupe.Key) bool {
	g.calls++
	return g.allow
}

func withGate(t *testing.T, g dedupe.Gate) {
	t.Helper()
	prev := gateFn
	gateFn = func() dedupe.Gate { return g }
	t.Cleanup(func() { gateFn = prev })
}

// FR-4.2/FR-3.1: a dropped duplicate performs NO rule evaluation, executes no
// operation, and does not unlock the client.
func TestHandleEnterCommand_DuplicateDroppedBeforeProcessing(t *testing.T) {
	unlocks, fp, l, ctx := consumerTestHarness(t, ProcessResult{
		Allow: true, MatchedRule: "r1", CharacterMoved: true,
	})
	g := &fakeGate{allow: false}
	withGate(t, g)

	handleEnterCommand(l, ctx, nil, testEnterCommand())

	assert.Equal(t, 1, g.calls, "the gate is consulted")
	assert.Equal(t, 0, fp.calls, "the script processor must never be invoked for a duplicate")
	assert.Equal(t, 0, *unlocks, "a dropped duplicate emits no EnableActions")
}

// A first ENTER passes the gate and is processed normally.
func TestHandleEnterCommand_AllowedByGateIsProcessed(t *testing.T) {
	unlocks, fp, l, ctx := consumerTestHarness(t, ProcessResult{
		Allow: true, MatchedRule: "r1", CharacterMoved: false,
	})
	g := &fakeGate{allow: true}
	withGate(t, g)

	handleEnterCommand(l, ctx, nil, testEnterCommand())

	assert.Equal(t, 1, g.calls)
	assert.Equal(t, 1, fp.calls)
	assert.Equal(t, 1, *unlocks)
}
```

Add `"atlas-portal-actions/dedupe"` to the test file's imports.

The pre-existing `TestHandleEnterCommand_*` tests from Task 7 do not install a `fakeGate`; they will run against the real `gateFn`, which returns `dedupe.GetGate()` → `nilGate{}` (allow) because `InitGate` is never called in the `script` package's tests. That is intentional and requires no change to them.

- [x] **Step 2: Run test to verify it fails**

Run: `go test ./script/ -run 'TestHandleEnterCommand_(Duplicate|AllowedByGate)' -v`
Expected: compile failure — `gateFn` undefined.

- [x] **Step 3: Add the gate seam and the check**

In `script/consumer.go`, add `"atlas-portal-actions/dedupe"` to the imports and extend the seam block from Task 7:

```go
var (
	newScriptProcessorFn = func(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
		return NewProcessor(l, ctx, db)
	}
	enableActionsFn = character.EnableActions
	gateFn          = dedupe.GetGate
)
```

In `handleEnterCommand`, insert the gate check immediately after the `field` / `channel` construction and strictly before `newScriptProcessorFn`:

```go
	// Create field model from command
	ch := channel.NewModel(c.WorldId, c.ChannelId)
	f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).SetInstance(c.Instance).Build()

	// Duplicate-command gate (task-184 FR-3.1). Evaluated before any script
	// load, condition evaluation, or operation dispatch, so a duplicate has no
	// side effect at all. Fails open on a Redis error — FR-2's conditional
	// unlock is the primary fix and stands on its own; this must never become a
	// single point of failure for portal traversal.
	//
	// A non-zero drop rate here means some outcome path is still unlocking a
	// character whose warp is in flight.
	if !gateFn().Allow(l, ctx, dedupe.Key{
		CharacterId: c.Body.CharacterId,
		MapId:       c.MapId,
		Instance:    c.Instance,
		PortalId:    c.PortalId,
	}) {
		return
	}

	// Create processor with tenant context from Kafka message
	processor := newScriptProcessorFn(l, ctx, db)
```

- [x] **Step 4: Initialise the gate at startup**

In `main.go`, add `"atlas-portal-actions/dedupe"` to the imports and, immediately after `action.InitRegistry(rc)`:

```go
	rc := atlas.Connect(l)
	action.InitRegistry(rc)
	dedupe.InitGate(rc)
```

- [x] **Step 5: Run tests to verify they pass**

Run: `go test ./script/ -v`
Expected: all PASS, including the six Task 7 consumer tests and the two new ones.

- [x] **Step 6: Build and vet**

Run: `go build ./... && go vet ./...`
Expected: clean.

- [x] **Step 7: Commit**

```bash
git add script/consumer.go script/consumer_test.go main.go
git commit -m "feat(portal-actions): drop duplicate ENTER commands before rule evaluation (FR-3.1, FR-3.5, FR-4.2)"
```

---

## Task 10: Full verification sweep and PR preparation

Satisfies the design §9 checklist and the PRD §10 acceptance criteria that are machine-checkable. This task ships nothing new — it proves what shipped.

**Files:**
- Modify: `docs/tasks/task-184-portal-enter-double-execute/plan.md` (check off remaining boxes) — no code.

- [x] **Step 1: Per-module tests with the race detector**

Run from the worktree root:

```bash
(cd services/atlas-portal-actions/atlas.com/portal && go test -race ./... && go vet ./...)
(cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test -race ./... && go vet ./...)
```

Expected: both modules clean. If a pre-existing test fails, do not weaken it — read it and determine whether it encoded the pre-task-184 behaviour (a `WarpToPortal` step expected to stay pending after `map_changed`, or an outcome expected to always unlock). Fix the test to assert the corrected behaviour, and record the change in the commit message.

- [x] **Step 2: Builds**

```bash
(cd services/atlas-portal-actions/atlas.com/portal && go build ./...)
(cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./...)
```

Expected: clean.

- [x] **Step 3: Repo guards**

```bash
tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/lint.sh --check
```

Expected: all exit 0. `tools/lint.sh --check` requires nvm — if it reports a Node failure rather than a lint failure, load nvm 22 first and re-run. If it reports formatting drift, run `tools/lint.sh` (no flags) to fix in place, then re-run `--check` and commit the formatting changes separately.

`tools/service-registration-guard.sh` and `tools/template-*-guard.sh` are **not** required: no service was added and no `services.json`, `deploy/k8s`, `docker-bake.hcl`, `go.work`, or tenant socket-config template was touched. Confirm that with `git diff --stat main...HEAD` before skipping them.

- [x] **Step 4: Decide whether a docker bake is required**

```bash
git diff --stat main...HEAD -- '*/go.mod' '*/go.sum'
```

Expected: **empty**. Every library this plan uses (`libs/atlas-redis`, `libs/atlas-saga`, `miniredis/v2`, `logrus/hooks/test`, testify) is already a dependency of both modules.

If the output is NOT empty, `docker buildx bake` becomes mandatory per CLAUDE.md item 4 — run it and do not skip:

```bash
docker buildx bake atlas-portal-actions
docker buildx bake atlas-saga-orchestrator
```

- [x] **Step 5: Confirm the worktree and branch**

```bash
git rev-parse --show-toplevel   # must end with /.worktrees/task-184-portal-enter-double-execute
git branch --show-current       # must be task-184-portal-enter-double-execute
git status --short              # must be clean
```

If either of the first two is wrong, STOP and report BLOCKED — work landed in the wrong tree.

- [x] **Step 6: Code review before the PR**

Invoke `superpowers:requesting-code-review`. It dispatches `plan-adherence-reviewer` and `backend-guidelines-reviewer` (Go files changed; no `atlas-ui` TypeScript changed, so the frontend reviewer is not needed). Findings land in `docs/tasks/task-184-portal-enter-double-execute/audit.md`.

Ensure each reviewer subagent operates inside this worktree, and verify `git status` is clean after their runs.

The three intentional deviations from the PRD are listed in design.md §10 and restated in context.md — point the plan-adherence reviewer at them so they are not read as omissions:
1. FR-2.3/2.4 condition is "successfully dispatched", not "declared", and applies to the error branch.
2. FR-2.5's `PendingAction` registration happens in the two executor methods, not in `handleEnterCommand`.
3. FR-3.3's tenant scoping is composed from `redis.TenantKey` + `redis.CompositeKey` because `Lock` is not tenant-aware.

- [x] **Step 7: Address review findings and commit**

Fix anything the review surfaces, re-run Steps 1–3, and commit.

- [x] **Step 8: Live verification note for the PR body**

The two PRD acceptance criteria that cannot be machine-checked must be verified in a live environment on GMS 83.1 with the `undodraco` portal from issue #1193:

- one touch → exactly one portal `ENTER` command executed and exactly one `MAP_CHANGED`
- no `SAGA_TIMEOUT` on a successful portal warp

Record in the PR body that these are pending live confirmation, along with the rollout ordering (**`atlas-saga-orchestrator` first, then `atlas-portal-actions`** — design §8) and the residual risk from design §3.4: a database-resident NPC conversation script that batches operations *after* a `warp_to_portal` / `warp_to_saved_location` step will now execute those operations for the first time. They are currently dead (the batch stalls at the warp step until a 30 s timeout fails the saga), so this is a bug fix, but it is worth reviewing against live tenant data.

---

## Requirement Coverage

| Requirement | Task |
|---|---|
| FR-1.1 acceptance entries for both warp actions | 3 |
| FR-1.2 corrected comment citing `warp.ProcessorImpl.ChangeMap` | 3 |
| FR-1.3 character-id guard in `AcceptEvent` | 2 |
| FR-1.4 `ExtractCharacterId` handles `WarpToSavedLocationPayload` | 1 |
| FR-1.5 skip logged via `LogSkip` with a distinct reason | 2 |
| FR-2.1 moving set declared as data beside the operation definitions | 4 |
| FR-2.2 unclassified operation fails, never defaults to non-moving | 4 |
| FR-2.3 no `EnableActions` when a moving operation dispatched | 7 |
| FR-2.4 every non-moving unlock path preserved | 7 |
| FR-2.5 `PendingAction` registered for the suppressed unlock | 5 |
| FR-2.6 explicit 5 s timeout on both warp sagas | 5 |
| FR-2.7 warp-appropriate failure message | 6 |
| FR-3.1 gate evaluated before rule evaluation | 9 |
| FR-3.2 `Lock.AcquireWithTTL`, not an in-process map | 8 |
| FR-3.3 tenant/character/map/instance/portal key via library helpers | 8 |
| FR-3.4 2 s TTL | 8 |
| FR-3.5 dropped duplicate logged at `Debug` with the key components | 8, 9 |
| FR-3.6 Redis failure fails open | 8 |
| FR-4.1 moving outcome emits no `EnableActions`; non-moving does | 7 |
| FR-4.2 duplicate performs no rule evaluation, executes no operation | 9 |
| FR-4.3 both warp actions complete on `MAP_CHANGED` with the tx id | 3 |
| FR-4.4 `MAP_CHANGED` for character A does not complete B's step | 2, 3 |
| FR-4.5 same-map warp still completes the step | 3 |
| NFR multi-tenancy: two tenants do not share a gate | 8 |
| Design §9 verification checklist | 10 |
