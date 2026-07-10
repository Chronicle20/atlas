# Time Leap Cooldown Reset (Buccaneer 5121010) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Casting Time Leap clears every active skill cooldown — except Time Leap's own — for the caster and in-range party members, with each cleared cooldown reflected on the affected client.

**Architecture:** atlas-channel gains a per-skill handler package `skill/handler/timeleap` that resolves recipients (caster + in-range party members via the existing `SelectInRangePartyMembers`) and emits one generic `RESET_COOLDOWNS` Kafka command per recipient. atlas-skills gains the `RESET_COOLDOWNS` command type, a consumer handler, a processor method pair `ResetCooldowns`/`ResetCooldownsAndEmit`, and a per-character registry enumeration `GetAllForCharacter`. Each cleared cooldown emits one existing-shape `COOLDOWN_EXPIRED` status event; the existing `handleCooldownExpired` → `CharacterSkillCooldownWriter` path clears the client UI unchanged.

**Tech Stack:** Go, Kafka (segmentio/kafka-go via atlas-kafka), Redis (go-redis via `libs/atlas-redis` TenantRegistry), miniredis for tests, testify in registry tests / stdlib testing elsewhere.

## Global Constraints

- **No `libs/` changes.** `skill.BuccaneerTimeLeapId` (= `Id(5121010)`) already exists in `libs/atlas-constants/skill/constants.go:3212`; `libs/atlas-redis` already provides `GetAllEntries`; `libs/atlas-packet` already decodes the party bitmap for 5121010.
- **All Redis access stays inside the existing `Registry` wrapper** in `services/atlas-skills/atlas.com/skills/skill/cooldown_registry.go` — `tools/redis-key-guard.sh` (run from repo root, WITHOUT a `GOWORK=off` prefix) must stay clean.
- **Prefix safety invariant:** character-key prefixes always use the trailing colon `"<charId>:"` so charId 100 never matches 1000/1001 (same as `ClearAll`, `cooldown_registry.go:59-64`).
- **Explicitly unchanged:** the logout/death `ClearAll` path, `handleCooldownExpired`/`CharacterSkillCooldownWriter` in atlas-channel, atlas-buffs, and `SetCooldownCommandProvider`'s call site (it now emits explicit zero `TransactionId`/`WorldId` — same bytes-on-wire semantics as today's decode).
- **Test setup uses the project Builder pattern**; do NOT create `*_testhelpers.go` files. Match each test file's existing assertion style (testify in `cooldown_registry_test.go`, plain `t.Fatalf`/`t.Errorf` in `processor_test.go`/`consumer_test.go` and all atlas-channel handler tests).
- **No `// TODO`, stubs, or 501s in landed commits.**
- **Committed files use repo-relative paths only** — never literal `/home/<name>/...`.
- **Module roots** (run `go test`/`go vet`/`go build` from these):
  - `services/atlas-skills/atlas.com/skills`
  - `services/atlas-channel/atlas.com/channel`
- **Final verification gate** (CLAUDE.md, mandatory before claiming done): `go test -race ./...`, `go vet ./...`, `go build ./...` clean in both modules; `docker buildx bake atlas-skills atlas-channel` from the worktree root; `tools/redis-key-guard.sh` clean from the repo root.
- **Worktree:** all work happens in `.worktrees/task-155-time-leap-cooldown-reset` on branch `task-155-time-leap-cooldown-reset`. Verify `git branch --show-current` after each commit.

---

### Task 1: atlas-skills — `Registry.GetAllForCharacter`

**Files:**
- Modify: `services/atlas-skills/atlas.com/skills/skill/cooldown_registry.go`
- Test: `services/atlas-skills/atlas.com/skills/skill/cooldown_registry_test.go`

**Interfaces:**
- Consumes: existing `TenantRegistry.GetAllEntries(ctx, t) (map[string]time.Time, error)` (`libs/atlas-redis/tenant_registry.go:175`); existing test harness `setupCooldownRegistryTest(t)` / `setupCooldownTestTenant(t)` / `cooldownTestCtx(ten)` already in `cooldown_registry_test.go`.
- Produces: `func (r *Registry) GetAllForCharacter(ctx context.Context, characterId uint32) (map[uint32]time.Time, error)` — skillId → expiresAt for the current tenant. Task 2 calls this.

- [ ] **Step 1: Write the failing tests**

Append to `services/atlas-skills/atlas.com/skills/skill/cooldown_registry_test.go` (internal test package `skill` — it can reach the private `reg` field for the malformed-suffix fixture):

```go
func TestGetAllForCharacter_ReturnsOnlyCharacterEntries(t *testing.T) {
	setupCooldownRegistryTest(t)
	ten := setupCooldownTestTenant(t)
	ctx := cooldownTestCtx(ten)
	r := GetRegistry()

	assert.NoError(t, r.Apply(ctx, 100, 5121010, 2940))
	assert.NoError(t, r.Apply(ctx, 100, 1311006, 300))
	assert.NoError(t, r.Apply(ctx, 200, 5221006, 60))

	got, err := r.GetAllForCharacter(ctx, 100)
	assert.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Contains(t, got, uint32(5121010))
	assert.Contains(t, got, uint32(1311006))
}

func TestGetAllForCharacter_PrefixSafety(t *testing.T) {
	setupCooldownRegistryTest(t)
	ten := setupCooldownTestTenant(t)
	ctx := cooldownTestCtx(ten)
	r := GetRegistry()

	// charId 100 must not match 1000 or 1001 entries.
	assert.NoError(t, r.Apply(ctx, 1000, 5121010, 60))
	assert.NoError(t, r.Apply(ctx, 1001, 1311006, 60))

	got, err := r.GetAllForCharacter(ctx, 100)
	assert.NoError(t, err)
	assert.Empty(t, got)

	got1000, err := r.GetAllForCharacter(ctx, 1000)
	assert.NoError(t, err)
	assert.Len(t, got1000, 1)
	assert.Contains(t, got1000, uint32(5121010))
}

func TestGetAllForCharacter_UnknownCharacter_Empty(t *testing.T) {
	setupCooldownRegistryTest(t)
	ten := setupCooldownTestTenant(t)
	ctx := cooldownTestCtx(ten)

	got, err := GetRegistry().GetAllForCharacter(ctx, 42)
	assert.NoError(t, err)
	assert.Empty(t, got)
}

func TestGetAllForCharacter_SkipsMalformedSuffixes(t *testing.T) {
	setupCooldownRegistryTest(t)
	ten := setupCooldownTestTenant(t)
	ctx := cooldownTestCtx(ten)
	r := GetRegistry()

	assert.NoError(t, r.Apply(ctx, 100, 5121010, 60))
	// Malformed suffix under the same character prefix — non-numeric skill id.
	assert.NoError(t, r.reg.Put(ctx, ten, "100:notaskill", time.Now().Add(time.Minute)))

	got, err := r.GetAllForCharacter(ctx, 100)
	assert.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Contains(t, got, uint32(5121010))
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run (from `services/atlas-skills/atlas.com/skills`): `go test ./skill/ -run TestGetAllForCharacter -v`
Expected: FAIL to compile with `r.GetAllForCharacter undefined`.

- [ ] **Step 3: Write the implementation**

Add to `services/atlas-skills/atlas.com/skills/skill/cooldown_registry.go` (after `ClearAll`; `strconv`, `strings`, `time` are already imported):

```go
// GetAllForCharacter returns the character's active cooldowns under the
// current tenant, keyed by skill id. The prefix "<charId>:" (with trailing
// colon) keeps the same safe-prefix invariant as ClearAll, so charId 100
// never matches 1000 or 1001. Malformed suffixes are skipped (same
// tolerance as GetAll).
func (r *Registry) GetAllForCharacter(ctx context.Context, characterId uint32) (map[uint32]time.Time, error) {
	t := tenant.MustFromContext(ctx)
	entries, err := r.reg.GetAllEntries(ctx, t)
	if err != nil {
		return nil, err
	}
	charPrefix := strconv.FormatUint(uint64(characterId), 10) + ":"
	result := make(map[uint32]time.Time)
	for suffix, expiresAt := range entries {
		if !strings.HasPrefix(suffix, charPrefix) {
			continue
		}
		sId, pErr := strconv.ParseUint(strings.TrimPrefix(suffix, charPrefix), 10, 32)
		if pErr != nil {
			continue
		}
		result[uint32(sId)] = expiresAt
	}
	return result, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./skill/ -run TestGetAllForCharacter -v`
Expected: all four tests PASS.

- [ ] **Step 5: Run the full package to catch regressions**

Run: `go test -race ./skill/...`
Expected: PASS (existing `ClearAll`/`GetAll` tests unchanged).

- [ ] **Step 6: Commit**

```bash
git add services/atlas-skills/atlas.com/skills/skill/cooldown_registry.go services/atlas-skills/atlas.com/skills/skill/cooldown_registry_test.go
git commit -m "feat(skills): add per-character cooldown enumeration to registry"
```

---

### Task 2: atlas-skills — `RESET_COOLDOWNS` message types + processor methods + mock

**Files:**
- Modify: `services/atlas-skills/atlas.com/skills/kafka/message/skill/kafka.go`
- Modify: `services/atlas-skills/atlas.com/skills/skill/processor.go`
- Modify: `services/atlas-skills/atlas.com/skills/skill/mock/processor.go`
- Test: `services/atlas-skills/atlas.com/skills/skill/processor_test.go`

**Interfaces:**
- Consumes: `Registry.GetAllForCharacter(ctx, characterId) (map[uint32]time.Time, error)` (Task 1); existing `GetRegistry().Clear(ctx, characterId, skillId) error`; existing `statusEventCooldownExpiredProvider(transactionId uuid.UUID, worldId world.Id, characterId uint32, id uint32)` (`producer.go:129`); `message.Buffer.Put` / `message.Emit`.
- Produces:
  - `CommandTypeResetCooldowns = "RESET_COOLDOWNS"` and `ResetCooldownsBody{ExceptSkillIds []uint32; SourceSkillId uint32}` in `kafka/message/skill` — Task 3 consumes.
  - `Processor.ResetCooldowns(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, exceptSkillIds []uint32) ([]uint32, error)` and `Processor.ResetCooldownsAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, exceptSkillIds []uint32) ([]uint32, error)` — Task 3 consumes `ResetCooldownsAndEmit`.

- [ ] **Step 1: Add the message types**

In `services/atlas-skills/atlas.com/skills/kafka/message/skill/kafka.go`, extend the command const block:

```go
const (
	EnvCommandTopic           = "COMMAND_TOPIC_SKILL"
	CommandTypeRequestCreate  = "REQUEST_CREATE"
	CommandTypeRequestUpdate  = "REQUEST_UPDATE"
	CommandTypeRequestDelete  = "REQUEST_DELETE"
	CommandTypeSetCooldown    = "SET_COOLDOWN"
	CommandTypeResetCooldowns = "RESET_COOLDOWNS"
)
```

and add after `SetCooldownBody`:

```go
// ResetCooldownsBody clears every active cooldown for the character except
// the listed skill ids. SourceSkillId identifies the triggering skill
// (5121010 for Time Leap) and is observability-only; generic senders may
// pass 0.
type ResetCooldownsBody struct {
	ExceptSkillIds []uint32 `json:"exceptSkillIds"`
	SourceSkillId  uint32   `json:"sourceSkillId"`
}
```

- [ ] **Step 2: Write the failing processor tests**

Append to `services/atlas-skills/atlas.com/skills/skill/processor_test.go` (external package `skill_test`; `message`, `skill`, `test`, `world`, `uuid`, `logtest` already imported). Add `encoding/json` and `skillmsg "atlas-skills/kafka/message/skill"` to the imports.

```go
func setupResetProcessor(t *testing.T) (skill.Processor, context.Context, func()) {
	t.Helper()
	setupCooldownRegistry(t)
	db := test.SetupTestDB(t)
	ctx := test.CreateTestContext()
	logger, _ := logtest.NewNullLogger()
	processor := skill.NewProcessor(logger, ctx, db)
	return processor, ctx, func() { test.CleanupTestDB(db) }
}

func decodeCooldownExpiredEvents(t *testing.T, mb *message.Buffer) []skillmsg.StatusEvent[skillmsg.StatusEventCooldownExpiredBody] {
	t.Helper()
	events := make([]skillmsg.StatusEvent[skillmsg.StatusEventCooldownExpiredBody], 0)
	for _, m := range mb.GetAll()[skillmsg.EnvStatusEventTopic] {
		var e skillmsg.StatusEvent[skillmsg.StatusEventCooldownExpiredBody]
		if err := json.Unmarshal(m.Value, &e); err != nil {
			t.Fatalf("failed to decode buffered event: %v", err)
		}
		events = append(events, e)
	}
	return events
}

func TestResetCooldowns_ClearsAllButExcepted(t *testing.T) {
	processor, ctx, cleanup := setupResetProcessor(t)
	defer cleanup()

	characterId := uint32(100)
	if err := skill.GetRegistry().Apply(ctx, characterId, 5121010, 2940); err != nil {
		t.Fatalf("Apply() unexpected error: %v", err)
	}
	if err := skill.GetRegistry().Apply(ctx, characterId, 1311006, 300); err != nil {
		t.Fatalf("Apply() unexpected error: %v", err)
	}
	if err := skill.GetRegistry().Apply(ctx, characterId, 5221006, 60); err != nil {
		t.Fatalf("Apply() unexpected error: %v", err)
	}

	transactionId := uuid.New()
	worldId := world.Id(1)
	mb := message.NewBuffer()
	cleared, err := processor.ResetCooldowns(mb)(transactionId, worldId, characterId, []uint32{5121010})
	if err != nil {
		t.Fatalf("ResetCooldowns() unexpected error: %v", err)
	}
	if len(cleared) != 2 {
		t.Fatalf("cleared = %v, want 2 entries", cleared)
	}
	for _, id := range cleared {
		if id == 5121010 {
			t.Fatalf("excepted skill 5121010 was cleared")
		}
	}

	// Registry: excepted survives, others are gone.
	if _, err := skill.GetRegistry().Get(ctx, characterId, 5121010); err != nil {
		t.Fatalf("excepted cooldown 5121010 missing from registry: %v", err)
	}
	if _, err := skill.GetRegistry().Get(ctx, characterId, 1311006); err == nil {
		t.Fatalf("cooldown 1311006 still in registry after reset")
	}
	if _, err := skill.GetRegistry().Get(ctx, characterId, 5221006); err == nil {
		t.Fatalf("cooldown 5221006 still in registry after reset")
	}

	// Events: one COOLDOWN_EXPIRED per cleared skill with real ids.
	events := decodeCooldownExpiredEvents(t, mb)
	if len(events) != 2 {
		t.Fatalf("buffered %d events, want 2", len(events))
	}
	seen := map[uint32]bool{}
	for _, e := range events {
		if e.Type != skillmsg.StatusEventTypeCooldownExpired {
			t.Errorf("event type = %s, want COOLDOWN_EXPIRED", e.Type)
		}
		if e.TransactionId != transactionId {
			t.Errorf("event transactionId = %s, want %s", e.TransactionId, transactionId)
		}
		if e.WorldId != worldId {
			t.Errorf("event worldId = %d, want %d", e.WorldId, worldId)
		}
		if e.CharacterId != characterId {
			t.Errorf("event characterId = %d, want %d", e.CharacterId, characterId)
		}
		seen[e.SkillId] = true
	}
	if !seen[1311006] || !seen[5221006] {
		t.Errorf("events cover skills %v, want 1311006 and 5221006", seen)
	}
	if seen[5121010] {
		t.Errorf("event emitted for excepted skill 5121010")
	}
}

func TestResetCooldowns_NoActiveCooldowns_NoOp(t *testing.T) {
	processor, _, cleanup := setupResetProcessor(t)
	defer cleanup()

	mb := message.NewBuffer()
	cleared, err := processor.ResetCooldowns(mb)(uuid.New(), world.Id(0), 100, []uint32{5121010})
	if err != nil {
		t.Fatalf("ResetCooldowns() unexpected error: %v", err)
	}
	if len(cleared) != 0 {
		t.Fatalf("cleared = %v, want empty", cleared)
	}
	if len(decodeCooldownExpiredEvents(t, mb)) != 0 {
		t.Fatalf("events buffered for no-op reset")
	}
}

func TestResetCooldowns_AllExcepted_NoOp(t *testing.T) {
	processor, ctx, cleanup := setupResetProcessor(t)
	defer cleanup()

	characterId := uint32(100)
	if err := skill.GetRegistry().Apply(ctx, characterId, 5121010, 2940); err != nil {
		t.Fatalf("Apply() unexpected error: %v", err)
	}

	mb := message.NewBuffer()
	cleared, err := processor.ResetCooldowns(mb)(uuid.New(), world.Id(0), characterId, []uint32{5121010})
	if err != nil {
		t.Fatalf("ResetCooldowns() unexpected error: %v", err)
	}
	if len(cleared) != 0 {
		t.Fatalf("cleared = %v, want empty", cleared)
	}
	if len(decodeCooldownExpiredEvents(t, mb)) != 0 {
		t.Fatalf("events buffered when every cooldown was excepted")
	}
	if _, err := skill.GetRegistry().Get(ctx, characterId, 5121010); err != nil {
		t.Fatalf("excepted cooldown removed: %v", err)
	}
}

func TestResetCooldowns_MultipleExceptions(t *testing.T) {
	processor, ctx, cleanup := setupResetProcessor(t)
	defer cleanup()

	characterId := uint32(100)
	for _, id := range []uint32{5121010, 1311006, 5221006} {
		if err := skill.GetRegistry().Apply(ctx, characterId, id, 60); err != nil {
			t.Fatalf("Apply() unexpected error: %v", err)
		}
	}

	mb := message.NewBuffer()
	cleared, err := processor.ResetCooldowns(mb)(uuid.New(), world.Id(0), characterId, []uint32{5121010, 1311006})
	if err != nil {
		t.Fatalf("ResetCooldowns() unexpected error: %v", err)
	}
	if len(cleared) != 1 || cleared[0] != 5221006 {
		t.Fatalf("cleared = %v, want [5221006]", cleared)
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run (from `services/atlas-skills/atlas.com/skills`): `go test ./skill/ -run TestResetCooldowns -v`
Expected: FAIL to compile with `processor.ResetCooldowns undefined`.

- [ ] **Step 4: Implement the processor methods**

In `services/atlas-skills/atlas.com/skills/skill/processor.go`, add to the `Processor` interface after `SetCooldownAndEmit`:

```go
	// ResetCooldowns clears every active cooldown for the character except
	// the listed skill ids, buffering one COOLDOWN_EXPIRED status event per
	// cleared skill. Returns the cleared skill ids. Registry-only — never
	// touches the DB. An empty enumeration (or all-excepted) is a
	// successful no-op.
	ResetCooldowns(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, exceptSkillIds []uint32) ([]uint32, error)

	// ResetCooldownsAndEmit wraps ResetCooldowns with the producer emit flow.
	ResetCooldownsAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, exceptSkillIds []uint32) ([]uint32, error)
```

Add the implementations after `SetCooldownAndEmit`:

```go
// ResetCooldowns clears every active cooldown for the character except the
// listed skill ids. Per-skill Clear failures are logged and skipped —
// partial success beats none, and command re-delivery is a harmless no-op.
// Unlike SetCooldown this never touches the DB: it is registry + events only.
func (p *ProcessorImpl) ResetCooldowns(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, exceptSkillIds []uint32) ([]uint32, error) {
	return func(transactionId uuid.UUID, worldId world.Id, characterId uint32, exceptSkillIds []uint32) ([]uint32, error) {
		cooldowns, err := GetRegistry().GetAllForCharacter(p.ctx, characterId)
		if err != nil {
			return nil, err
		}
		except := make(map[uint32]struct{}, len(exceptSkillIds))
		for _, id := range exceptSkillIds {
			except[id] = struct{}{}
		}
		cleared := make([]uint32, 0, len(cooldowns))
		for skillId := range cooldowns {
			if _, skip := except[skillId]; skip {
				continue
			}
			if cErr := GetRegistry().Clear(p.ctx, characterId, skillId); cErr != nil {
				p.l.WithError(cErr).Errorf("Unable to clear cooldown for character [%d] skill [%d]; continuing.", characterId, skillId)
				continue
			}
			_ = mb.Put(skill2.EnvStatusEventTopic, statusEventCooldownExpiredProvider(transactionId, worldId, characterId, skillId))
			cleared = append(cleared, skillId)
		}
		return cleared, nil
	}
}

// ResetCooldownsAndEmit clears cooldowns and emits the buffered status events.
func (p *ProcessorImpl) ResetCooldownsAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, exceptSkillIds []uint32) ([]uint32, error) {
	var cleared []uint32
	err := message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		var rErr error
		cleared, rErr = p.ResetCooldowns(buf)(transactionId, worldId, characterId, exceptSkillIds)
		return rErr
	})
	if err != nil {
		return cleared, err
	}
	p.l.WithFields(logrus.Fields{
		"transaction_id":   transactionId.String(),
		"character_id":     characterId,
		"cleared_count":    len(cleared),
		"except_skill_ids": exceptSkillIds,
	}).Info("Reset skill cooldowns.")
	return cleared, nil
}
```

- [ ] **Step 5: Update the mock**

In `services/atlas-skills/atlas.com/skills/skill/mock/processor.go`, add (extend the import block with `"github.com/Chronicle20/atlas/libs/atlas-constants/world"` and `"github.com/google/uuid"`):

```go
// ResetCooldowns mocks the ResetCooldowns method
func (m *ProcessorMock) ResetCooldowns(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, exceptSkillIds []uint32) ([]uint32, error) {
	args := m.Called(mb)
	return args.Get(0).(func(transactionId uuid.UUID, worldId world.Id, characterId uint32, exceptSkillIds []uint32) ([]uint32, error))
}

// ResetCooldownsAndEmit mocks the ResetCooldownsAndEmit method
func (m *ProcessorMock) ResetCooldownsAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, exceptSkillIds []uint32) ([]uint32, error) {
	args := m.Called(transactionId, worldId, characterId, exceptSkillIds)
	return args.Get(0).([]uint32), args.Error(1)
}
```

Note: the existing mock methods (`CreateAndEmit` etc.) use an older curried shape that predates the `transactionId`/`worldId` interface signatures. Do NOT rewrite them — only add the two new methods, matching the real interface exactly.

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./skill/... -run TestResetCooldowns -v`
Expected: all four tests PASS.

- [ ] **Step 7: Run the whole module**

Run: `go test -race ./... && go vet ./...`
Expected: PASS/clean (existing `ClearAll` consumer tests must be untouched).

- [ ] **Step 8: Commit**

```bash
git add services/atlas-skills/atlas.com/skills/kafka/message/skill/kafka.go services/atlas-skills/atlas.com/skills/skill/processor.go services/atlas-skills/atlas.com/skills/skill/mock/processor.go services/atlas-skills/atlas.com/skills/skill/processor_test.go
git commit -m "feat(skills): add RESET_COOLDOWNS processor methods and message types"
```

---

### Task 3: atlas-skills — consumer handler for `RESET_COOLDOWNS`

**Files:**
- Modify: `services/atlas-skills/atlas.com/skills/kafka/consumer/skill/consumer.go`
- Test: Create `services/atlas-skills/atlas.com/skills/kafka/consumer/skill/consumer_internal_test.go`

**Interfaces:**
- Consumes: `skill.NewProcessor(l, ctx, db).ResetCooldownsAndEmit(transactionId, worldId, characterId, exceptSkillIds) ([]uint32, error)` (Task 2); `skill2.CommandTypeResetCooldowns` / `skill2.ResetCooldownsBody` (Task 2).
- Produces: `handleCommandResetCooldowns(db *gorm.DB) message.Handler[skill2.Command[skill2.ResetCooldownsBody]]`, registered on the `EnvCommandTopic` handler chain in `InitHandlers`.

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-skills/atlas.com/skills/kafka/consumer/skill/consumer_internal_test.go`. Internal package (`package skill`) because the handler funcs are unexported; the existing `consumer_test.go` stays external (`skill_test`) and untouched.

```go
package skill

import (
	skill2 "atlas-skills/kafka/message/skill"
	"atlas-skills/skill"
	"atlas-skills/test"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

func setupResetConsumerTest(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	skill.InitRegistry(client)
}

// Wrong command type must leave the registry untouched.
func TestHandleCommandResetCooldowns_TypeGuard(t *testing.T) {
	setupResetConsumerTest(t)
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)
	ctx := test.CreateTestContext()
	logger, _ := logtest.NewNullLogger()

	characterId := uint32(100)
	if err := skill.GetRegistry().Apply(ctx, characterId, 1311006, 300); err != nil {
		t.Fatalf("Apply() unexpected error: %v", err)
	}

	c := skill2.Command[skill2.ResetCooldownsBody]{
		TransactionId: uuid.New(),
		WorldId:       world.Id(0),
		CharacterId:   characterId,
		Type:          skill2.CommandTypeSetCooldown, // wrong type
		Body:          skill2.ResetCooldownsBody{ExceptSkillIds: []uint32{5121010}},
	}
	handleCommandResetCooldowns(db)(logger, ctx, c)

	if _, err := skill.GetRegistry().Get(ctx, characterId, 1311006); err != nil {
		t.Fatalf("registry modified by wrong-type command: %v", err)
	}
}

// Happy path: cooldowns cleared except the exclusion list. The Kafka emit
// step fails in the test environment (no broker/env topic) AFTER the
// registry mutation, which is the documented partial-success behavior —
// assert on registry state only.
func TestHandleCommandResetCooldowns_ClearsRegistry(t *testing.T) {
	setupResetConsumerTest(t)
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)
	ctx := test.CreateTestContext()
	logger, _ := logtest.NewNullLogger()

	characterId := uint32(100)
	for _, id := range []uint32{5121010, 1311006, 5221006} {
		if err := skill.GetRegistry().Apply(ctx, characterId, id, 300); err != nil {
			t.Fatalf("Apply() unexpected error: %v", err)
		}
	}

	c := skill2.Command[skill2.ResetCooldownsBody]{
		TransactionId: uuid.New(),
		WorldId:       world.Id(0),
		CharacterId:   characterId,
		Type:          skill2.CommandTypeResetCooldowns,
		Body: skill2.ResetCooldownsBody{
			ExceptSkillIds: []uint32{5121010},
			SourceSkillId:  5121010,
		},
	}
	handleCommandResetCooldowns(db)(logger, ctx, c)

	if _, err := skill.GetRegistry().Get(ctx, characterId, 5121010); err != nil {
		t.Fatalf("excepted cooldown 5121010 removed: %v", err)
	}
	if _, err := skill.GetRegistry().Get(ctx, characterId, 1311006); err == nil {
		t.Fatalf("cooldown 1311006 not cleared")
	}
	if _, err := skill.GetRegistry().Get(ctx, characterId, 5221006); err == nil {
		t.Fatalf("cooldown 5221006 not cleared")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run (from `services/atlas-skills/atlas.com/skills`): `go test ./kafka/consumer/skill/ -run TestHandleCommandResetCooldowns -v`
Expected: FAIL to compile with `undefined: handleCommandResetCooldowns`.

- [ ] **Step 3: Implement the handler and register it**

In `services/atlas-skills/atlas.com/skills/kafka/consumer/skill/consumer.go`, add after `handleCommandSetCooldown`:

```go
// handleCommandResetCooldowns clears every active cooldown for the
// character except the command's exclusion list. SourceSkillId is
// observability-only (5121010 for Time Leap; generic senders may pass 0).
func handleCommandResetCooldowns(db *gorm.DB) message.Handler[skill2.Command[skill2.ResetCooldownsBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c skill2.Command[skill2.ResetCooldownsBody]) {
		if c.Type != skill2.CommandTypeResetCooldowns {
			return
		}
		if _, err := skill.NewProcessor(l, ctx, db).ResetCooldownsAndEmit(c.TransactionId, c.WorldId, c.CharacterId, c.Body.ExceptSkillIds); err != nil {
			l.WithError(err).WithFields(logrus.Fields{
				"transaction_id":  c.TransactionId.String(),
				"character_id":    c.CharacterId,
				"source_skill_id": c.Body.SourceSkillId,
			}).Error("Unable to reset cooldowns.")
		}
	}
}
```

Register it in `InitHandlers` after the `handleCommandSetCooldown` registration:

```go
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandResetCooldowns(db)))); err != nil {
				return err
			}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./kafka/consumer/skill/ -v`
Expected: both new tests PASS and every pre-existing consumer test still PASSES (regression gate for the logout/death `ClearAll` path).

- [ ] **Step 5: Run the whole module**

Run: `go test -race ./... && go vet ./... && go build ./...`
Expected: clean.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-skills/atlas.com/skills/kafka/consumer/skill/consumer.go services/atlas-skills/atlas.com/skills/kafka/consumer/skill/consumer_internal_test.go
git commit -m "feat(skills): consume RESET_COOLDOWNS commands"
```

---

### Task 4: atlas-channel — command envelope upgrade + `RESET_COOLDOWNS` plumbing

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/skill/kafka.go`
- Modify: `services/atlas-channel/atlas.com/channel/data/skill/producer.go`
- Modify: `services/atlas-channel/atlas.com/channel/character/skill/processor.go`
- Test: Create `services/atlas-channel/atlas.com/channel/data/skill/producer_test.go`

**Interfaces:**
- Consumes: existing `producer.CreateKey(int) []byte` and `producer.SingleMessageProvider` from `libs/atlas-kafka/producer`; existing `producer.ProviderImpl` from `atlas-channel/kafka/producer`; `field.Model.WorldId()`.
- Produces:
  - Channel mirror `Command[E]` gains `TransactionId uuid.UUID` + `WorldId world.Id` (field-for-field match with the skills-side envelope; skills-side decode of old `SET_COOLDOWN` messages already tolerates the fields — wire-compatible both directions).
  - `CommandTypeResetCooldowns = "RESET_COOLDOWNS"`, `ResetCooldownsBody{ExceptSkillIds []uint32; SourceSkillId uint32}` in `atlas-channel/kafka/message/skill`.
  - `ResetCooldownsCommandProvider(transactionId uuid.UUID, worldId world.Id, characterId uint32, exceptSkillIds []uint32, sourceSkillId uint32) model.Provider[[]kafka.Message]` in `atlas-channel/data/skill`.
  - `Processor.ResetCooldowns(transactionId uuid.UUID, f field.Model, exceptSkillIds []uint32, sourceSkillId uint32) model.Operator[uint32]` on `atlas-channel/character/skill` — Task 5 consumes this.

- [ ] **Step 1: Write the failing producer test**

Create `services/atlas-channel/atlas.com/channel/data/skill/producer_test.go` (internal package `skill`; the unaliased `"atlas-channel/kafka/message/skill"` import shadows the package name, same as `producer.go`):

```go
package skill

import (
	"bytes"
	"encoding/json"
	"testing"

	"atlas-channel/kafka/message/skill"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/google/uuid"
)

func TestResetCooldownsCommandProvider_EncodesEnvelopeAndBody(t *testing.T) {
	transactionId := uuid.New()
	worldId := world.Id(1)
	characterId := uint32(100)

	msgs, err := ResetCooldownsCommandProvider(transactionId, worldId, characterId, []uint32{5121010}, 5121010)()
	if err != nil {
		t.Fatalf("provider returned error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("provider returned %d messages, want 1", len(msgs))
	}
	if !bytes.Equal(msgs[0].Key, producer.CreateKey(int(characterId))) {
		t.Fatalf("message key = %v, want CreateKey(%d)", msgs[0].Key, characterId)
	}

	var c skill.Command[skill.ResetCooldownsBody]
	if err := json.Unmarshal(msgs[0].Value, &c); err != nil {
		t.Fatalf("failed to decode command: %v", err)
	}
	if c.Type != skill.CommandTypeResetCooldowns {
		t.Errorf("type = %s, want RESET_COOLDOWNS", c.Type)
	}
	if c.TransactionId != transactionId {
		t.Errorf("transactionId = %s, want %s", c.TransactionId, transactionId)
	}
	if c.WorldId != worldId {
		t.Errorf("worldId = %d, want %d", c.WorldId, worldId)
	}
	if c.CharacterId != characterId {
		t.Errorf("characterId = %d, want %d", c.CharacterId, characterId)
	}
	if len(c.Body.ExceptSkillIds) != 1 || c.Body.ExceptSkillIds[0] != 5121010 {
		t.Errorf("exceptSkillIds = %v, want [5121010]", c.Body.ExceptSkillIds)
	}
	if c.Body.SourceSkillId != 5121010 {
		t.Errorf("sourceSkillId = %d, want 5121010", c.Body.SourceSkillId)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run (from `services/atlas-channel/atlas.com/channel`): `go test ./data/skill/ -v`
Expected: FAIL to compile with `undefined: ResetCooldownsCommandProvider`.

- [ ] **Step 3: Upgrade the message mirror**

Replace the command section of `services/atlas-channel/atlas.com/channel/kafka/message/skill/kafka.go`:

```go
package skill

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvCommandTopic           = "COMMAND_TOPIC_SKILL"
	CommandTypeSetCooldown    = "SET_COOLDOWN"
	CommandTypeResetCooldowns = "RESET_COOLDOWNS"
)

// Command mirrors atlas-skills' command envelope field-for-field.
// TransactionId/WorldId were added with RESET_COOLDOWNS (task-155);
// SET_COOLDOWN keeps emitting zero values for both — the same values the
// skills-side decoder produced for the old, field-less JSON.
type Command[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       world.Id  `json:"worldId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type SetCooldownBody struct {
	SkillId  uint32 `json:"skillId"`
	Cooldown uint32 `json:"cooldown"`
}

// ResetCooldownsBody clears every active cooldown for the character except
// the listed skill ids. SourceSkillId identifies the triggering skill
// (5121010 for Time Leap) and is observability-only.
type ResetCooldownsBody struct {
	ExceptSkillIds []uint32 `json:"exceptSkillIds"`
	SourceSkillId  uint32   `json:"sourceSkillId"`
}
```

Leave the status-event section of the file exactly as it is.

- [ ] **Step 4: Add the command provider**

Append to `services/atlas-channel/atlas.com/channel/data/skill/producer.go` (extend imports with `"github.com/Chronicle20/atlas/libs/atlas-constants/world"` and `"github.com/google/uuid"`):

```go
func ResetCooldownsCommandProvider(transactionId uuid.UUID, worldId world.Id, characterId uint32, exceptSkillIds []uint32, sourceSkillId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &skill.Command[skill.ResetCooldownsBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		CharacterId:   characterId,
		Type:          skill.CommandTypeResetCooldowns,
		Body: skill.ResetCooldownsBody{
			ExceptSkillIds: exceptSkillIds,
			SourceSkillId:  sourceSkillId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

- [ ] **Step 5: Add the processor method**

In `services/atlas-channel/atlas.com/channel/character/skill/processor.go`, add to the `Processor` interface (extend imports with `"github.com/google/uuid"`):

```go
	ResetCooldowns(transactionId uuid.UUID, f field.Model, exceptSkillIds []uint32, sourceSkillId uint32) model.Operator[uint32]
```

and the implementation after `ApplyCooldown`:

```go
// ResetCooldowns emits a RESET_COOLDOWNS command for the operated-on
// character, clearing every cooldown except exceptSkillIds. Mirrors
// ApplyCooldown's operator shape so callers can fan out over recipients.
func (p *ProcessorImpl) ResetCooldowns(transactionId uuid.UUID, f field.Model, exceptSkillIds []uint32, sourceSkillId uint32) model.Operator[uint32] {
	return func(characterId uint32) error {
		return producer.ProviderImpl(p.l)(p.ctx)(skill2.EnvCommandTopic)(skill3.ResetCooldownsCommandProvider(transactionId, f.WorldId(), characterId, exceptSkillIds, sourceSkillId))
	}
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./data/skill/ ./character/skill/ ./kafka/message/skill/ -v`
Expected: `TestResetCooldownsCommandProvider_EncodesEnvelopeAndBody` PASS, no other failures.

- [ ] **Step 7: Build the whole module (envelope-change ripple check)**

Run: `go build ./... && go vet ./...`
Expected: clean. If any other file constructs `skill.Command[...]` positionally it will fail here — fix by switching that construction to named fields (all known call sites already use named fields).

- [ ] **Step 8: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/message/skill/kafka.go services/atlas-channel/atlas.com/channel/data/skill/producer.go services/atlas-channel/atlas.com/channel/data/skill/producer_test.go services/atlas-channel/atlas.com/channel/character/skill/processor.go
git commit -m "feat(channel): RESET_COOLDOWNS command plumbing with transaction-correlated envelope"
```

---

### Task 5: atlas-channel — `timeleap` per-skill handler

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/skill/handler/timeleap/timeleap.go`
- Create: `services/atlas-channel/atlas.com/channel/skill/handler/timeleap/timeleap_test.go`
- Modify: `services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go`

**Interfaces:**
- Consumes: `channelhandler.Register(id skill2.Id, h Handler)` / `channelhandler.Lookup` (`skill/handler/registry.go`); `channelhandler.SelectInRangePartyMembers(l, ctx, f, casterId, casterX, casterY, e, memberBitmap) []channelhandler.PartyRecipient` (`recipients.go:94` — returns nil on missing rect); `channelhandler.PartyRecipient.Id() uint32`; `skillproc.NewProcessor(l, ctx).ResetCooldowns(transactionId, f, exceptSkillIds, sourceSkillId) model.Operator[uint32]` (Task 4); `character.NewProcessor(l, ctx).GetById()(characterId)`.
- Produces: registered handler for `skill2.BuccaneerTimeLeapId`, dispatched from `UseSkill` (`skill/handler/common.go:117`) after MP charge and Time Leap's own `SET_COOLDOWN` — the handler adds ONLY the reset emission (PRD FR-4; the socket handler already broadcasts the cast effect).

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-channel/atlas.com/channel/skill/handler/timeleap/timeleap_test.go` (internal package `timeleap`, seam-swap with `t.Cleanup` restore — pattern: `mysticdoor_test.go`):

```go
package timeleap

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
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const (
	testCharId = uint32(1001)
	testX      = int16(100)
	testY      = int16(200)
)

func testLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetLevel(logrus.DebugLevel)
	return l
}

func testField() field.Model {
	return field.NewBuilder(world.Id(1), channel.Id(0), _map.Id(100000000)).Build()
}

func testInfo(bitmap byte) packetmodel.SkillUsageInfo {
	return packetmodel.NewSkillUsageInfoBuilder().
		SetSkillId(uint32(skill2.BuccaneerTimeLeapId)).
		SetSkillLevel(1).
		SetAffectedPartyMemberBitmap(bitmap).
		Build()
}

type resetCall struct {
	transactionId  uuid.UUID
	characterId    uint32
	exceptSkillIds []uint32
	sourceSkillId  uint32
}

// invokeApply swaps the three seams, runs Apply, and returns the captured
// reset emissions.
func invokeApply(
	t *testing.T,
	casterLoader func(logrus.FieldLogger, context.Context, uint32) (int16, int16, error),
	partySelector func(logrus.FieldLogger, context.Context, field.Model, uint32, int16, int16, effect.Model, byte) []channelhandler.PartyRecipient,
	emitErr error,
	bitmap byte,
) []resetCall {
	t.Helper()
	origCaster := loadCaster
	origParty := selectParty
	origEmit := emitReset
	t.Cleanup(func() {
		loadCaster = origCaster
		selectParty = origParty
		emitReset = origEmit
	})

	var calls []resetCall
	loadCaster = casterLoader
	if partySelector != nil {
		selectParty = partySelector
	}
	emitReset = func(_ logrus.FieldLogger, _ context.Context, transactionId uuid.UUID, _ field.Model, exceptSkillIds []uint32, sourceSkillId uint32, characterId uint32) error {
		calls = append(calls, resetCall{
			transactionId:  transactionId,
			characterId:    characterId,
			exceptSkillIds: exceptSkillIds,
			sourceSkillId:  sourceSkillId,
		})
		return emitErr
	}

	err := Apply(testLogger())(context.Background())(nil, testField(), testCharId, testInfo(bitmap), effect.Model{})
	if err != nil {
		t.Fatalf("Apply returned unexpected error: %v", err)
	}
	return calls
}

func okCasterLoader(_ logrus.FieldLogger, _ context.Context, _ uint32) (int16, int16, error) {
	return testX, testY, nil
}

func partyOf(ids ...uint32) func(logrus.FieldLogger, context.Context, field.Model, uint32, int16, int16, effect.Model, byte) []channelhandler.PartyRecipient {
	return func(_ logrus.FieldLogger, _ context.Context, _ field.Model, _ uint32, _ int16, _ int16, _ effect.Model, _ byte) []channelhandler.PartyRecipient {
		out := make([]channelhandler.PartyRecipient, 0, len(ids))
		for _, id := range ids {
			out = append(out, channelhandler.NewPartyRecipientBuilder().SetId(id).Build())
		}
		return out
	}
}

// Solo cast: exactly one command, for the caster, excepting Time Leap.
func TestTimeLeapSoloCast_ResetsCasterOnly(t *testing.T) {
	calls := invokeApply(t, okCasterLoader, partyOf(), nil, 0)
	if len(calls) != 1 {
		t.Fatalf("emitted %d commands, want 1", len(calls))
	}
	if calls[0].characterId != testCharId {
		t.Errorf("recipient = %d, want caster %d", calls[0].characterId, testCharId)
	}
	if len(calls[0].exceptSkillIds) != 1 || calls[0].exceptSkillIds[0] != uint32(skill2.BuccaneerTimeLeapId) {
		t.Errorf("exceptSkillIds = %v, want [5121010]", calls[0].exceptSkillIds)
	}
	if calls[0].sourceSkillId != uint32(skill2.BuccaneerTimeLeapId) {
		t.Errorf("sourceSkillId = %d, want 5121010", calls[0].sourceSkillId)
	}
}

// Party cast: one command per member + caster, one shared transactionId,
// every command excepting Time Leap.
func TestTimeLeapPartyCast_ResetsAllRecipients(t *testing.T) {
	calls := invokeApply(t, okCasterLoader, partyOf(2002, 3003), nil, 0x06)
	if len(calls) != 3 {
		t.Fatalf("emitted %d commands, want 3", len(calls))
	}
	want := map[uint32]bool{testCharId: false, 2002: false, 3003: false}
	for _, c := range calls {
		if _, ok := want[c.characterId]; !ok {
			t.Errorf("unexpected recipient %d", c.characterId)
		}
		want[c.characterId] = true
		if c.transactionId != calls[0].transactionId {
			t.Errorf("transactionId differs across recipients: %s vs %s", c.transactionId, calls[0].transactionId)
		}
		if len(c.exceptSkillIds) != 1 || c.exceptSkillIds[0] != uint32(skill2.BuccaneerTimeLeapId) {
			t.Errorf("recipient %d exceptSkillIds = %v, want [5121010]", c.characterId, c.exceptSkillIds)
		}
	}
	for id, seen := range want {
		if !seen {
			t.Errorf("recipient %d received no command", id)
		}
	}
}

// Caster load failure: zero commands, no panic, nil error.
func TestTimeLeapCasterLoadFailure_EmitsNothing(t *testing.T) {
	failLoader := func(_ logrus.FieldLogger, _ context.Context, _ uint32) (int16, int16, error) {
		return 0, 0, errors.New("character service down")
	}
	calls := invokeApply(t, failLoader, partyOf(2002), nil, 0x02)
	if len(calls) != 0 {
		t.Fatalf("emitted %d commands after caster load failure, want 0", len(calls))
	}
}

// Missing rectangle (zero-value effect.Model) through the REAL selector:
// SelectInRangePartyMembers returns nil before any I/O, so the cast
// degrades to caster-only.
func TestTimeLeapMissingRect_CasterOnly(t *testing.T) {
	calls := invokeApply(t, okCasterLoader, nil, nil, 0x02)
	if len(calls) != 1 {
		t.Fatalf("emitted %d commands, want 1 (caster only)", len(calls))
	}
	if calls[0].characterId != testCharId {
		t.Errorf("recipient = %d, want caster %d", calls[0].characterId, testCharId)
	}
}

// Emission failure for one recipient must not abort the rest.
func TestTimeLeapEmissionFailure_ContinuesWithRemaining(t *testing.T) {
	calls := invokeApply(t, okCasterLoader, partyOf(2002, 3003), errors.New("kafka down"), 0x06)
	if len(calls) != 3 {
		t.Fatalf("emitted %d commands, want all 3 attempted despite failures", len(calls))
	}
}

// Registration: the blank-importable init() must install the handler.
func TestTimeLeapRegistered(t *testing.T) {
	if _, ok := channelhandler.Lookup(skill2.BuccaneerTimeLeapId); !ok {
		t.Fatal("Lookup(BuccaneerTimeLeapId) returned ok=false; init() registration missing")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run (from `services/atlas-channel/atlas.com/channel`): `go test ./skill/handler/timeleap/ -v`
Expected: FAIL to compile — package `timeleap` does not exist yet.

- [ ] **Step 3: Write the handler**

Create `services/atlas-channel/atlas.com/channel/skill/handler/timeleap/timeleap.go`:

```go
// Package timeleap implements the Buccaneer Time Leap (5121010) per-skill
// handler. Time Leap's entire effect is server-side: it clears every active
// skill cooldown — except Time Leap's own — for the caster and for in-range
// party members (reference: Cosmic StatEffect removeAllCooldownsExcept).
//
// By the time this handler runs, UseSkill has already charged MP and applied
// Time Leap's own cooldown via SET_COOLDOWN, and the socket handler
// broadcasts the cast effect after UseSkill returns — this handler adds ONLY
// the RESET_COOLDOWNS emission (PRD FR-4). WZ-verified: 5121010 has an LT/RB
// rectangle at every level and no statups, so the generic buff path never
// fires and the rect-based party selector is the correct recipient filter.
package timeleap

import (
	"context"
	"sync"

	"atlas-channel/character"
	skillproc "atlas-channel/character/skill"
	"atlas-channel/data/skill/effect"
	channelhandler "atlas-channel/skill/handler"
	"atlas-channel/socket/writer"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func init() {
	channelhandler.Register(skill2.BuccaneerTimeLeapId, Apply)
}

// loadCaster returns the caster's (X, Y) — only the position is needed for
// the recipient rectangle. Test seam (pattern: mysticdoor).
var loadCaster = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) (int16, int16, error) {
	c, err := character.NewProcessor(l, ctx).GetById()(characterId)
	if err != nil {
		return 0, 0, err
	}
	return c.X(), c.Y(), nil
}

// selectParty resolves in-range party members. Test seam.
var selectParty = func(l logrus.FieldLogger, ctx context.Context, f field.Model, casterId uint32, x, y int16, e effect.Model, bitmap byte) []channelhandler.PartyRecipient {
	return channelhandler.SelectInRangePartyMembers(l, ctx, f, casterId, x, y, e, bitmap)
}

// emitReset sends one RESET_COOLDOWNS command for the recipient. Test seam.
var emitReset = func(l logrus.FieldLogger, ctx context.Context, transactionId uuid.UUID, f field.Model, exceptSkillIds []uint32, sourceSkillId uint32, characterId uint32) error {
	return skillproc.NewProcessor(l, ctx).ResetCooldowns(transactionId, f, exceptSkillIds, sourceSkillId)(characterId)
}

// warnedRectangles dedupes the missing-rectangle warning per (skill, level).
var warnedRectangles sync.Map

// warnIfMissingRectangle logs once per (skill, level) when the effect has no
// LT/RB rectangle. Defensive only — WZ carries the rect at every Time Leap
// level. Duplicated from heal (package-private there); hoist to handler if a
// third handler needs it.
func warnIfMissingRectangle(skillId skill2.Id, skillLevel byte, e effect.Model, logf func()) {
	lt, rb := e.LT(), e.RB()
	if lt.X() != 0 || lt.Y() != 0 || rb.X() != 0 || rb.Y() != 0 {
		return
	}
	key := uint64(skillId)<<8 | uint64(skillLevel)
	if _, loaded := warnedRectangles.LoadOrStore(key, struct{}{}); loaded {
		return
	}
	logf()
}

// Apply is the Time Leap handler installed in the per-skill registry.
//
// Lifecycle:
//  1. Load caster (X, Y). On failure: log, skip the reset entirely — the
//     cast continues (per-step failures never abort the cast, heal policy).
//  2. Resolve in-range party members (missing rect or bitmap 0 → empty →
//     caster-only; FR-5).
//  3. Emit one RESET_COOLDOWNS per recipient (caster + members) with a
//     single per-cast transactionId and exceptSkillIds=[5121010] — every
//     recipient keeps their own Time Leap cooldown (FR-3). Per-recipient
//     emission failures are logged and do not abort remaining recipients.
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
			x, y, err := loadCaster(l, ctx, characterId)
			if err != nil {
				l.WithError(err).Errorf("Time Leap: failed to load caster [%d].", characterId)
				return nil
			}

			warnIfMissingRectangle(skill2.Id(info.SkillId()), info.SkillLevel(), e, func() {
				l.Warnf("Time Leap: skill effect [%d] level [%d] has no LT/RB rectangle — falling back to caster-only.", info.SkillId(), info.SkillLevel())
			})

			party := selectParty(l, ctx, f, characterId, x, y, e, info.AffectedPartyMemberBitmap())

			transactionId := uuid.New()
			except := []uint32{uint32(skill2.BuccaneerTimeLeapId)}
			source := uint32(skill2.BuccaneerTimeLeapId)

			if eErr := emitReset(l, ctx, transactionId, f, except, source, characterId); eErr != nil {
				l.WithError(eErr).Errorf("Time Leap: reset emission failed for caster [%d].", characterId)
			}
			for _, r := range party {
				if eErr := emitReset(l, ctx, transactionId, f, except, source, r.Id()); eErr != nil {
					l.WithError(eErr).Errorf("Time Leap: reset emission failed for recipient [%d] from caster [%d].", r.Id(), characterId)
				}
			}

			l.Debugf("Time Leap: caster=[%d] recipients=[%d] transaction=[%s].",
				characterId, 1+len(party), transactionId)
			return nil
		}
	}
}
```

- [ ] **Step 4: Register the package**

In `services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go`, add the blank import:

```go
import (
	_ "atlas-channel/skill/handler/heal"       // Cleric Heal — task 045
	_ "atlas-channel/skill/handler/mysticdoor" // Priest Mystic Door — task-093
	_ "atlas-channel/skill/handler/timeleap"   // Buccaneer Time Leap — task-155
)
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./skill/handler/... -v`
Expected: all 7 new timeleap tests PASS; existing handler/heal/mysticdoor tests unchanged and PASSING.

- [ ] **Step 6: Run the whole module**

Run: `go test -race ./... && go vet ./... && go build ./...`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/skill/handler/timeleap/ services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go
git commit -m "feat(channel): Time Leap handler resets cooldowns for caster and in-range party"
```

---

### Task 6: Full verification gate

**Files:** none (verification only; fix-and-recommit if anything fails).

**Interfaces:**
- Consumes: everything above.
- Produces: the evidence required before this branch may be called done (CLAUDE.md Build & Verification + PRD acceptance criteria).

- [ ] **Step 1: atlas-skills module gate**

Run (from `services/atlas-skills/atlas.com/skills`):

```bash
go test -race ./... && go vet ./... && go build ./...
```

Expected: all clean. Existing logout/death `ClearAll` consumer tests pass unchanged (PRD acceptance).

- [ ] **Step 2: atlas-channel module gate**

Run (from `services/atlas-channel/atlas.com/channel`):

```bash
go test -race ./... && go vet ./... && go build ./...
```

Expected: all clean.

- [ ] **Step 3: Docker bake both services**

Run from the worktree root (`.worktrees/task-155-time-leap-cooldown-reset`):

```bash
docker buildx bake atlas-skills atlas-channel
```

Expected: both images build. This is mandatory — `go build` against `go.work` cannot catch Dockerfile `COPY libs/...` gaps. (No new libs were added, but the gate is non-optional.)

- [ ] **Step 4: Redis key guard**

Run from the repo root (NOT prefixed with `GOWORK=off`):

```bash
tools/redis-key-guard.sh
```

Expected: clean — all new Redis access went through the existing `Registry` wrapper.

- [ ] **Step 5: Confirm worktree/branch integrity**

```bash
git rev-parse --show-toplevel   # must end with .worktrees/task-155-time-leap-cooldown-reset
git branch --show-current       # must be task-155-time-leap-cooldown-reset
git status --short              # must be clean (all work committed)
```

- [ ] **Step 6: Check off the PRD acceptance criteria that are code-verifiable**

Unit-verifiable now: processor reset-with-exclusions incl. no-op (Task 2), registry prefix safety 100 vs 1000 (Task 1), channel recipient selection + command emission (Task 5), `ClearAll` regression (Task 3 Step 4). The live-client criteria (cooldown UI clearing without relog, cross-map isolation) are exercised at deploy time, after PR review.

---

## Self-Review Notes (writing-plans checklist)

- **Spec coverage:** design §4.3 registry → Task 1; §4.3 message/processor/mock (FR-6, FR-8, FR-9, FR-10) → Task 2; §4.3 consumer (FR-7) → Task 3; §3.4/§4.2 envelope + producer + channel processor → Task 4; §4.1 handler + registration (FR-1..FR-5) → Task 5; §6 verification gate → Task 6. §4.4 "explicitly unchanged" is enforced by Global Constraints. FR-11 needs no task (existing `handleCooldownExpired` reused as-is).
- **Type consistency:** `ResetCooldowns(mb) func(uuid.UUID, world.Id, uint32, []uint32) ([]uint32, error)` is identical in the interface (Task 2 Step 4), mock (Task 2 Step 5), and consumer call (Task 3 Step 3). Channel-side `ResetCooldowns(transactionId uuid.UUID, f field.Model, exceptSkillIds []uint32, sourceSkillId uint32) model.Operator[uint32]` is identical in Task 4 Step 5 and Task 5's `emitReset` seam. `ResetCooldownsBody{ExceptSkillIds, SourceSkillId}` matches field-for-field on both sides.
- **Placeholder scan:** no TBDs; every code step carries the full code.
