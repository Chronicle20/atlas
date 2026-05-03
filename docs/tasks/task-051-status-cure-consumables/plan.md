# Status Cure Consumables Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire status-cure consumables (`2050000`–`2050004`) so drinking them actually removes the matching debuffs, and add a general `CancelByStatTypes` API on atlas-buffs so future cure callers (NPC blessings, future Dispel) reuse the same path.

**Architecture:** atlas-consumables collects non-zero disease cure specs in `ApplyItemEffects`, dispatches a single `CANCEL_BY_TYPES` Kafka command on `COMMAND_TOPIC_CHARACTER_BUFF` *before* HP/MP recovery. atlas-buffs's new processor + registry method scans the character's buffs, drops any whose `Changes()` intersects the requested type set, and emits one existing-shape `EXPIRED` event per cancelled buff. atlas-channel and atlas-data are untouched.

**Tech Stack:** Go 1.x, Redis (via `atlas-redis` `TenantRegistry`), Kafka (`segmentio/kafka-go`), `logrus`, `miniredis` for registry tests, `testify/assert`.

---

## Task 1: atlas-buffs Registry — `CancelByStatTypes` (filter-by-types primitive)

**Files:**
- Modify: `services/atlas-buffs/atlas.com/buffs/character/registry.go`
- Test: `services/atlas-buffs/atlas.com/buffs/character/registry_test.go`

- [ ] **Step 1: Write the failing test for empty type set**

Append to `services/atlas-buffs/atlas.com/buffs/character/registry_test.go`:

```go
func TestRegistry_CancelByStatTypes_EmptyTypes(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)

	// Apply a POISON buff so we can prove an empty type set leaves it alone.
	changes := []stat.Model{stat.NewStat("POISON", -10)}
	_, _ = GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), uint32(1000), int32(124), byte(1), int32(60), changes)

	cancelled, err := GetRegistry().CancelByStatTypes(ctx, uint32(1000), map[string]bool{})
	assert.NoError(t, err)
	assert.Nil(t, cancelled)

	m, _ := GetRegistry().Get(ctx, uint32(1000))
	assert.Len(t, m.Buffs(), 1)
}
```

- [ ] **Step 2: Run the test to verify it fails to compile**

Run: `cd services/atlas-buffs/atlas.com/buffs && go test ./character/ -run TestRegistry_CancelByStatTypes_EmptyTypes -v`
Expected: build failure — `r.CancelByStatTypes undefined`.

- [ ] **Step 3: Implement `CancelByStatTypes` on the Registry**

Append to `services/atlas-buffs/atlas.com/buffs/character/registry.go` (after `CancelAll`):

```go
// CancelByStatTypes removes any buff whose Changes() intersects typeSet.
// Returns the cancelled buffs (caller emits EXPIRED events).
// Empty typeSet returns (nil, nil) without touching Redis.
func (r *Registry) CancelByStatTypes(ctx context.Context, characterId uint32, typeSet map[string]bool) ([]buff.Model, error) {
	if len(typeSet) == 0 {
		return nil, nil
	}

	t := tenant.MustFromContext(ctx)

	m, err := r.characters.Get(ctx, t, characterId)
	if errors.Is(err, atlas.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	cancelled := make([]buff.Model, 0)
	keep := make(map[int32]buff.Model)
	for id, b := range m.buffs {
		matched := false
		for _, c := range b.Changes() {
			if typeSet[c.Type()] {
				matched = true
				break
			}
		}
		if matched {
			cancelled = append(cancelled, b)
		} else {
			keep[id] = b
		}
	}

	if len(cancelled) == 0 {
		return nil, nil
	}

	m.buffs = keep
	if err := r.characters.Put(ctx, t, characterId, m); err != nil {
		return nil, err
	}
	return cancelled, nil
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd services/atlas-buffs/atlas.com/buffs && go test ./character/ -run TestRegistry_CancelByStatTypes_EmptyTypes -v`
Expected: PASS.

- [ ] **Step 5: Add tests for no-match, single-match, multi-match, and unknown-character cases**

Append to `services/atlas-buffs/atlas.com/buffs/character/registry_test.go`:

```go
func TestRegistry_CancelByStatTypes_NoMatch(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)

	// Character has only HOLY_SYMBOL, ask to cancel POISON — should keep the buff.
	changes := []stat.Model{stat.NewStat("HOLY_SYMBOL", 30)}
	_, _ = GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), uint32(1000), int32(2311003), byte(1), int32(60), changes)

	cancelled, err := GetRegistry().CancelByStatTypes(ctx, uint32(1000), map[string]bool{"POISON": true})
	assert.NoError(t, err)
	assert.Nil(t, cancelled)

	m, _ := GetRegistry().Get(ctx, uint32(1000))
	assert.Len(t, m.Buffs(), 1)
}

func TestRegistry_CancelByStatTypes_SingleMatch(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)

	poison := []stat.Model{stat.NewStat("POISON", -10)}
	holy := []stat.Model{stat.NewStat("HOLY_SYMBOL", 30)}
	_, _ = GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), uint32(1000), int32(124), byte(1), int32(60), poison)
	_, _ = GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), uint32(1000), int32(2311003), byte(1), int32(60), holy)

	cancelled, err := GetRegistry().CancelByStatTypes(ctx, uint32(1000), map[string]bool{"POISON": true})
	assert.NoError(t, err)
	assert.Len(t, cancelled, 1)
	assert.Equal(t, int32(124), cancelled[0].SourceId())

	m, _ := GetRegistry().Get(ctx, uint32(1000))
	assert.Len(t, m.Buffs(), 1)
	_, holdsHoly := m.Buffs()[int32(2311003)]
	assert.True(t, holdsHoly)
}

func TestRegistry_CancelByStatTypes_MultiMatch(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)

	poison := []stat.Model{stat.NewStat("POISON", -10)}
	curse := []stat.Model{stat.NewStat("CURSE", -50)}
	weaken := []stat.Model{stat.NewStat("WEAKEN", -20)}
	_, _ = GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), uint32(1000), int32(124), byte(1), int32(60), poison)
	_, _ = GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), uint32(1000), int32(125), byte(1), int32(60), curse)
	_, _ = GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), uint32(1000), int32(126), byte(1), int32(60), weaken)

	cancelled, err := GetRegistry().CancelByStatTypes(ctx, uint32(1000), map[string]bool{
		"POISON": true,
		"CURSE":  true,
	})
	assert.NoError(t, err)
	assert.Len(t, cancelled, 2)

	m, _ := GetRegistry().Get(ctx, uint32(1000))
	assert.Len(t, m.Buffs(), 1)
	_, holdsWeaken := m.Buffs()[int32(126)]
	assert.True(t, holdsWeaken)
}

func TestRegistry_CancelByStatTypes_UnknownCharacter(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)

	cancelled, err := GetRegistry().CancelByStatTypes(ctx, uint32(9999), map[string]bool{"POISON": true})
	assert.NoError(t, err)
	assert.Nil(t, cancelled)
}
```

- [ ] **Step 6: Run the new tests; expect all to pass**

Run: `cd services/atlas-buffs/atlas.com/buffs && go test ./character/ -run TestRegistry_CancelByStatTypes -v`
Expected: 4 PASS.

- [ ] **Step 7: Run the entire `character` package test suite to confirm no regressions**

Run: `cd services/atlas-buffs/atlas.com/buffs && go test ./character/ -v`
Expected: all PASS.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-buffs/atlas.com/buffs/character/registry.go services/atlas-buffs/atlas.com/buffs/character/registry_test.go
git commit -m "feat(atlas-buffs): add Registry.CancelByStatTypes (task-051)"
```

---

## Task 2: atlas-buffs Processor — expose `CancelByStatTypes` and emit EXPIRED events

**Files:**
- Modify: `services/atlas-buffs/atlas.com/buffs/character/processor.go`
- Test: `services/atlas-buffs/atlas.com/buffs/character/processor_test.go`

- [ ] **Step 1: Add `CancelByStatTypes` to the `Processor` interface**

In `services/atlas-buffs/atlas.com/buffs/character/processor.go`, expand the interface:

```go
type Processor interface {
	GetById(characterId uint32) (Model, error)
	Apply(worldId world.Id, channelId channel.Id, characterId uint32, fromId uint32, sourceId int32, level byte, duration int32, changes []stat.Model) error
	Cancel(worldId world.Id, characterId uint32, sourceId int32) error
	CancelAll(worldId world.Id, characterId uint32) error
	CancelByStatTypes(worldId world.Id, characterId uint32, types []string) error
	ExpireBuffs() error
	ProcessPoisonTicks() error
}
```

- [ ] **Step 2: Write the failing test**

Append to `services/atlas-buffs/atlas.com/buffs/character/processor_test.go`:

```go
func TestProcessor_CancelByStatTypes_EmptyTypes(t *testing.T) {
	processor, _, _ := setupProcessorTest(t)

	err := processor.CancelByStatTypes(world.Id(0), uint32(1000), nil)
	assert.NoError(t, err)
}

func TestProcessor_CancelByStatTypes_NoMatch(t *testing.T) {
	processor, _, ctx := setupProcessorTest(t)

	worldId := world.Id(0)
	characterId := uint32(1000)
	holy := []stat.Model{stat.NewStat("HOLY_SYMBOL", 30)}
	_ = processor.Apply(worldId, channel.Id(0), characterId, uint32(2000), int32(2311003), byte(1), int32(60), holy)

	err := processor.CancelByStatTypes(worldId, characterId, []string{"POISON"})
	assert.NoError(t, err)

	m, _ := GetRegistry().Get(ctx, characterId)
	assert.Len(t, m.Buffs(), 1)
}

func TestProcessor_CancelByStatTypes_MultiMatch(t *testing.T) {
	processor, _, ctx := setupProcessorTest(t)

	worldId := world.Id(0)
	characterId := uint32(1000)

	_ = processor.Apply(worldId, channel.Id(0), characterId, uint32(2000), int32(124), byte(1), int32(60), []stat.Model{stat.NewStat("POISON", -10)})
	_ = processor.Apply(worldId, channel.Id(0), characterId, uint32(2000), int32(125), byte(1), int32(60), []stat.Model{stat.NewStat("CURSE", -50)})
	_ = processor.Apply(worldId, channel.Id(0), characterId, uint32(2000), int32(126), byte(1), int32(60), []stat.Model{stat.NewStat("WEAKEN", -20)})

	err := processor.CancelByStatTypes(worldId, characterId, []string{"POISON", "CURSE", "WEAKEN", "DARKNESS", "SEAL"})
	assert.NoError(t, err)

	m, _ := GetRegistry().Get(ctx, characterId)
	assert.Len(t, m.Buffs(), 0)
}

func TestProcessor_CancelByStatTypes_HolyShieldDoesNotBlockRemoval(t *testing.T) {
	// D5: Holy Shield gates application, not cure. A character with HOLY_SHIELD
	// who somehow has a debuff must still be curable.
	processor, _, ctx := setupProcessorTest(t)

	worldId := world.Id(0)
	characterId := uint32(1000)

	// Insert a POISON buff via the registry directly so the immunity check on
	// Apply can't refuse it once HOLY_SHIELD is present.
	_, _ = GetRegistry().Apply(ctx, worldId, channel.Id(0), characterId, int32(124), byte(1), int32(60), []stat.Model{stat.NewStat("POISON", -10)})
	_, _ = GetRegistry().Apply(ctx, worldId, channel.Id(0), characterId, int32(2311005), byte(1), int32(60), []stat.Model{stat.NewStat("HOLY_SHIELD", 1)})

	err := processor.CancelByStatTypes(worldId, characterId, []string{"POISON"})
	assert.NoError(t, err)

	m, _ := GetRegistry().Get(ctx, characterId)
	assert.Len(t, m.Buffs(), 1)
	_, stillHasHolyShield := m.Buffs()[int32(2311005)]
	assert.True(t, stillHasHolyShield)
}
```

- [ ] **Step 3: Run the tests to verify they fail to compile**

Run: `cd services/atlas-buffs/atlas.com/buffs && go test ./character/ -run TestProcessor_CancelByStatTypes -v`
Expected: build failure — `(*ProcessorImpl).CancelByStatTypes undefined`.

- [ ] **Step 4: Implement `CancelByStatTypes` on `ProcessorImpl`**

Append to `services/atlas-buffs/atlas.com/buffs/character/processor.go` (after `CancelAll`):

```go
func (p *ProcessorImpl) CancelByStatTypes(worldId world.Id, characterId uint32, types []string) error {
	if len(types) == 0 {
		return nil
	}
	typeSet := make(map[string]bool, len(types))
	for _, t := range types {
		typeSet[t] = true
	}

	cancelled, err := GetRegistry().CancelByStatTypes(p.ctx, characterId, typeSet)
	if err != nil {
		return err
	}
	if len(cancelled) == 0 {
		return nil
	}

	return message.Emit(p.l, p.ctx)(func(buf *message.Buffer) error {
		for _, b := range cancelled {
			if err := buf.Put(character2.EnvEventStatusTopic, expiredStatusEventProvider(worldId, characterId, b.SourceId(), b.Level(), b.Duration(), b.Changes(), b.CreatedAt(), b.ExpiresAt())); err != nil {
				return err
			}
		}
		return nil
	})
}
```

- [ ] **Step 5: Run the new tests; expect all to pass**

Run: `cd services/atlas-buffs/atlas.com/buffs && go test ./character/ -run TestProcessor_CancelByStatTypes -v`
Expected: 4 PASS.

- [ ] **Step 6: Run full package tests**

Run: `cd services/atlas-buffs/atlas.com/buffs && go test ./character/ -v`
Expected: all PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-buffs/atlas.com/buffs/character/processor.go services/atlas-buffs/atlas.com/buffs/character/processor_test.go
git commit -m "feat(atlas-buffs): expose Processor.CancelByStatTypes (task-051)"
```

---

## Task 3: atlas-buffs Kafka — `CANCEL_BY_TYPES` command type and body

**Files:**
- Modify: `services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go`

- [ ] **Step 1: Add the constant and body type**

Edit `services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go` so the command-type block becomes:

```go
const (
	EnvCommandTopic           = "COMMAND_TOPIC_CHARACTER_BUFF"
	CommandTypeApply          = "APPLY"
	CommandTypeCancel         = "CANCEL"
	CommandTypeCancelAll      = "CANCEL_ALL"
	CommandTypeCancelByTypes  = "CANCEL_BY_TYPES"
)
```

…and add the new body type just below `CancelAllCommandBody`:

```go
type CancelByTypesCommandBody struct {
	Types []string `json:"types"`
}
```

- [ ] **Step 2: Build atlas-buffs to verify the constants compile**

Run: `cd services/atlas-buffs/atlas.com/buffs && go build ./...`
Expected: build succeeds with no output.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go
git commit -m "feat(atlas-buffs): add CANCEL_BY_TYPES command shape (task-051)"
```

---

## Task 4: atlas-buffs Kafka — register `handleCancelByTypes` consumer

**Files:**
- Modify: `services/atlas-buffs/atlas.com/buffs/kafka/consumer/character/consumer.go`

- [ ] **Step 1: Register the new handler in `InitHandlers`**

Edit `services/atlas-buffs/atlas.com/buffs/kafka/consumer/character/consumer.go`. In `InitHandlers`, after the existing `handleCancelAll` registration, add:

```go
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCancelByTypes))); err != nil {
			return err
		}
```

- [ ] **Step 2: Add the `handleCancelByTypes` function at the end of the file**

```go
func handleCancelByTypes(l logrus.FieldLogger, ctx context.Context, c character2.Command[character2.CancelByTypesCommandBody]) {
	if c.Type != character2.CommandTypeCancelByTypes {
		return
	}

	if err := character.NewProcessor(l, ctx).CancelByStatTypes(c.WorldId, c.CharacterId, c.Body.Types); err != nil {
		l.WithError(err).Errorf("Unable to cancel buffs by types %v for character [%d].", c.Body.Types, c.CharacterId)
	}
}
```

- [ ] **Step 3: Build atlas-buffs**

Run: `cd services/atlas-buffs/atlas.com/buffs && go build ./...`
Expected: build succeeds.

- [ ] **Step 4: Run the full atlas-buffs test suite**

Run: `cd services/atlas-buffs/atlas.com/buffs && go test ./...`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-buffs/atlas.com/buffs/kafka/consumer/character/consumer.go
git commit -m "feat(atlas-buffs): register CANCEL_BY_TYPES consumer (task-051)"
```

---

## Task 5: atlas-consumables Kafka — mirror `CANCEL_BY_TYPES` command shape

**Files:**
- Modify: `services/atlas-consumables/atlas.com/consumables/kafka/message/character/buff/kafka.go`

- [ ] **Step 1: Add the constant and body type**

Edit `services/atlas-consumables/atlas.com/consumables/kafka/message/character/buff/kafka.go` so the constant block becomes:

```go
const (
	EnvCommandTopic          = "COMMAND_TOPIC_CHARACTER_BUFF"
	CommandTypeApply         = "APPLY"
	CommandTypeCancel        = "CANCEL"
	CommandTypeCancelByTypes = "CANCEL_BY_TYPES"
)
```

…and append below `CancelCommandBody`:

```go
type CancelByTypesCommandBody struct {
	Types []string `json:"types"`
}
```

- [ ] **Step 2: Build**

Run: `cd services/atlas-consumables/atlas.com/consumables && go build ./...`
Expected: build succeeds.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-consumables/atlas.com/consumables/kafka/message/character/buff/kafka.go
git commit -m "feat(atlas-consumables): add CANCEL_BY_TYPES command shape (task-051)"
```

---

## Task 6: atlas-consumables Kafka producer — `cancelByTypesCommandProvider`

**Files:**
- Modify: `services/atlas-consumables/atlas.com/consumables/character/buff/producer.go`

- [ ] **Step 1: Append `cancelByTypesCommandProvider` to the producer file**

Append to `services/atlas-consumables/atlas.com/consumables/character/buff/producer.go`:

```go
func cancelByTypesCommandProvider(f field.Model, characterId uint32, types []string) model.Provider[[]kafka.Message] {
	body := buff2.CancelByTypesCommandBody{
		Types: append([]string(nil), types...),
	}
	key := producer.CreateKey(int(characterId))
	value := &buff2.Command[buff2.CancelByTypesCommandBody]{
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		MapId:       f.MapId(),
		Instance:    f.Instance(),
		CharacterId: characterId,
		Type:        buff2.CommandTypeCancelByTypes,
		Body:        body,
	}
	return producer.SingleMessageProvider(key, value)
}
```

The defensive copy of `types` matches the producer's "value is owned by Kafka" pattern — callers shouldn't mutate after submit.

- [ ] **Step 2: Build**

Run: `cd services/atlas-consumables/atlas.com/consumables && go build ./...`
Expected: build succeeds.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-consumables/atlas.com/consumables/character/buff/producer.go
git commit -m "feat(atlas-consumables): add cancelByTypesCommandProvider (task-051)"
```

---

## Task 7: atlas-consumables Buff Processor — `CancelByTypes` wrapper

**Files:**
- Modify: `services/atlas-consumables/atlas.com/consumables/character/buff/processor.go`

- [ ] **Step 1: Add the `CancelByTypes` method**

Append to `services/atlas-consumables/atlas.com/consumables/character/buff/processor.go`:

```go
func (p *Processor) CancelByTypes(f field.Model, characterId uint32, types []string) error {
	return producer.ProviderImpl(p.l)(p.ctx)(buff2.EnvCommandTopic)(cancelByTypesCommandProvider(f, characterId, types))
}
```

- [ ] **Step 2: Build**

Run: `cd services/atlas-consumables/atlas.com/consumables && go build ./...`
Expected: build succeeds.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-consumables/atlas.com/consumables/character/buff/processor.go
git commit -m "feat(atlas-consumables): add buff.Processor.CancelByTypes (task-051)"
```

---

## Task 8: atlas-consumables `collectCureTypes` helper + tests

**Files:**
- Modify: `services/atlas-consumables/atlas.com/consumables/consumable/processor.go`
- Test: `services/atlas-consumables/atlas.com/consumables/consumable/processor_test.go`

- [ ] **Step 1: Write failing tests for the helper**

Add `consumable3 "atlas-consumables/data/consumable"` to the import block in `services/atlas-consumables/atlas.com/consumables/consumable/processor_test.go` if not already present, and add `"github.com/stretchr/testify/assert"` (the project already vendors testify — see `services/atlas-consumables/atlas.com/consumables/map/character/registry_test.go`).

Append:

```go
func makeCureModel(t *testing.T, specs map[consumable3.SpecType]int32) consumable3.Model {
	t.Helper()
	rm := consumable3.RestModel{Spec: specs}
	m, err := consumable3.Extract(rm)
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}
	return m
}

func TestCollectCureTypes_AntidotePot(t *testing.T) {
	ci := makeCureModel(t, map[consumable3.SpecType]int32{
		consumable3.SpecTypePoison: 1,
	})
	got := collectCureTypes(ci)
	assert.Equal(t, []string{"POISON"}, got)
}

func TestCollectCureTypes_HolyWater(t *testing.T) {
	ci := makeCureModel(t, map[consumable3.SpecType]int32{
		consumable3.SpecTypeSeal:  1,
		consumable3.SpecTypeCurse: 1,
	})
	got := collectCureTypes(ci)
	// Order is fixed (POISON, DARKNESS, WEAKEN, SEAL, CURSE) for determinism;
	// missing entries are dropped, so Holy Water yields just SEAL then CURSE.
	assert.Equal(t, []string{"SEAL", "CURSE"}, got)
}

func TestCollectCureTypes_AllCure(t *testing.T) {
	ci := makeCureModel(t, map[consumable3.SpecType]int32{
		consumable3.SpecTypePoison:   1,
		consumable3.SpecTypeDarkness: 1,
		consumable3.SpecTypeWeakness: 1,
		consumable3.SpecTypeSeal:     1,
		consumable3.SpecTypeCurse:    1,
	})
	got := collectCureTypes(ci)
	assert.Equal(t, []string{"POISON", "DARKNESS", "WEAKEN", "SEAL", "CURSE"}, got)
}

func TestCollectCureTypes_NonCureConsumable(t *testing.T) {
	// White potion: HP recovery only, no cure flags.
	ci := makeCureModel(t, map[consumable3.SpecType]int32{
		consumable3.SpecTypeHP: 1000,
	})
	got := collectCureTypes(ci)
	assert.Empty(t, got)
}

func TestCollectCureTypes_ZeroFlagsIgnored(t *testing.T) {
	// A 0-valued cure spec must be treated as "not present" (parser default).
	ci := makeCureModel(t, map[consumable3.SpecType]int32{
		consumable3.SpecTypePoison: 0,
		consumable3.SpecTypeCurse:  1,
	})
	got := collectCureTypes(ci)
	assert.Equal(t, []string{"CURSE"}, got)
}
```

- [ ] **Step 2: Run the tests to verify they fail to compile**

Run: `cd services/atlas-consumables/atlas.com/consumables && go test ./consumable/ -run TestCollectCureTypes -v`
Expected: build failure — `collectCureTypes undefined`.

- [ ] **Step 3: Implement `collectCureTypes` in `consumable/processor.go`**

Add this helper just above `ApplyItemEffects` in `services/atlas-consumables/atlas.com/consumables/consumable/processor.go`:

```go
// collectCureTypes returns the TemporaryStatType strings whose matching
// consumable cure spec is non-zero. Order is fixed
// (POISON, DARKNESS, WEAKEN, SEAL, CURSE) for deterministic Kafka payloads
// and easier testing.
func collectCureTypes(ci consumable3.Model) []string {
	pairs := []struct {
		spec consumable3.SpecType
		stat ts.TemporaryStatType
	}{
		{consumable3.SpecTypePoison, ts.TemporaryStatTypePoison},
		{consumable3.SpecTypeDarkness, ts.TemporaryStatTypeDarkness},
		{consumable3.SpecTypeWeakness, ts.TemporaryStatTypeWeaken},
		{consumable3.SpecTypeSeal, ts.TemporaryStatTypeSeal},
		{consumable3.SpecTypeCurse, ts.TemporaryStatTypeCurse},
	}
	out := make([]string, 0, len(pairs))
	for _, p := range pairs {
		if val, ok := ci.GetSpec(p.spec); ok && val > 0 {
			out = append(out, string(p.stat))
		}
	}
	return out
}
```

- [ ] **Step 4: Run the tests; expect all to pass**

Run: `cd services/atlas-consumables/atlas.com/consumables && go test ./consumable/ -run TestCollectCureTypes -v`
Expected: 5 PASS.

- [ ] **Step 5: Run the full `consumable` package**

Run: `cd services/atlas-consumables/atlas.com/consumables && go test ./consumable/ -v`
Expected: all PASS (existing tests still pass; new helper has no side effects on prior code).

- [ ] **Step 6: Commit**

```bash
git add services/atlas-consumables/atlas.com/consumables/consumable/processor.go services/atlas-consumables/atlas.com/consumables/consumable/processor_test.go
git commit -m "feat(atlas-consumables): add collectCureTypes helper (task-051)"
```

---

## Task 9: atlas-consumables `ApplyItemEffects` — dispatch cure first, then HP/MP, then buffs

**Files:**
- Modify: `services/atlas-consumables/atlas.com/consumables/consumable/processor.go`

- [ ] **Step 1: Restructure `ApplyItemEffects`**

The current function (lines `72`–`156`) builds a `statups` slice and applies HP/MP recovery inline. We need cure dispatch first. Replace the body of `ApplyItemEffects` (keep its signature and surrounding comments) with this implementation:

```go
func ApplyItemEffects(l logrus.FieldLogger, ctx context.Context, c character.Model, f field.Model, ci consumable3.Model, characterId uint32, itemId item2.Id) {
	bp := buff.NewProcessor(l, ctx)
	cp := character.NewProcessor(l, ctx)

	// 1. Cure first. Cure runs before HP/MP recovery so a queued poison tick
	// (also routed through atlas-buffs's per-character partition) lands behind
	// the cancel and cannot eat part of the heal between drink-time and
	// cancel-commit-time. See task-051 D3.
	if cureTypes := collectCureTypes(ci); len(cureTypes) > 0 {
		if err := bp.CancelByTypes(f, characterId, cureTypes); err != nil {
			l.WithError(err).Errorf("Unable to dispatch cure-by-types for character [%d] item [%d].", characterId, itemId)
		}
	}

	// 2. HP/MP recovery.
	if val, ok := ci.GetSpec(consumable3.SpecTypeHP); ok && val > 0 {
		_ = cp.ChangeHP(f, characterId, int16(val))
	}
	if val, ok := ci.GetSpec(consumable3.SpecTypeHPRecovery); ok && val > 0 {
		pct := float64(val) / float64(100)
		res := int16(math.Floor(float64(c.MaxHp()) * pct))
		_ = cp.ChangeHP(f, characterId, res)
	}
	if val, ok := ci.GetSpec(consumable3.SpecTypeMP); ok && val > 0 {
		_ = cp.ChangeMP(f, characterId, int16(val))
	}
	if val, ok := ci.GetSpec(consumable3.SpecTypeMPRecovery); ok && val > 0 {
		pct := float64(val) / float64(100)
		res := int16(math.Floor(float64(c.MaxMp()) * pct))
		_ = cp.ChangeMP(f, characterId, res)
	}

	// 3. Status-up buffs.
	statups := make([]stat.Model, 0)
	duration := int32(0)

	if val, ok := ci.GetSpec(consumable3.SpecTypeAccuracy); ok && val > 0 {
		statups = append(statups, stat.Model{Type: ts.TemporaryStatTypeAccuracy, Amount: val})
	}
	if val, ok := ci.GetSpec(consumable3.SpecTypeEvasion); ok && val > 0 {
		statups = append(statups, stat.Model{Type: ts.TemporaryStatTypeAvoidability, Amount: val})
	}
	if val, ok := ci.GetSpec(consumable3.SpecTypeJump); ok && val > 0 {
		statups = append(statups, stat.Model{Type: ts.TemporaryStatTypeJump, Amount: val})
	}
	if val, ok := ci.GetSpec(consumable3.SpecTypeMagicAttack); ok && val > 0 {
		statups = append(statups, stat.Model{Type: ts.TemporaryStatTypeMagicAttack, Amount: val})
	}
	if val, ok := ci.GetSpec(consumable3.SpecTypeMagicDefense); ok && val > 0 {
		statups = append(statups, stat.Model{Type: ts.TemporaryStatTypeMagicDefense, Amount: val})
	}
	if val, ok := ci.GetSpec(consumable3.SpecTypeWeaponAttack); ok && val > 0 {
		statups = append(statups, stat.Model{Type: ts.TemporaryStatTypeWeaponAttack, Amount: val})
	}
	if val, ok := ci.GetSpec(consumable3.SpecTypeWeaponDefense); ok && val > 0 {
		statups = append(statups, stat.Model{Type: ts.TemporaryStatTypeWeaponDefense, Amount: val})
	}
	if val, ok := ci.GetSpec(consumable3.SpecTypeSpeed); ok && val > 0 {
		statups = append(statups, stat.Model{Type: ts.TemporaryStatTypeSpeed, Amount: val})
	}
	if val, ok := ci.GetSpec(consumable3.SpecTypeMorph); ok && val > 0 {
		statups = append(statups, stat.Model{Type: ts.TemporaryStatTypeMorph, Amount: val})
	}
	if val, ok := ci.GetSpec(consumable3.SpecTypeTime); ok && val > 0 {
		duration = val / 1000
	}

	if len(statups) > 0 {
		_ = bp.Apply(f, characterId, -int32(itemId), byte(0), duration, statups)(characterId)
	}
}
```

Notes:
- Stat-up branches that previously appeared inline are preserved verbatim, just regrouped under section 3 so HP/MP can run between cure and status-up.
- `cp.ChangeHP` / `cp.ChangeMP` paths are unchanged.
- `bp.Apply` call site is unchanged.
- Error handling on `bp.CancelByTypes` is log-and-continue, matching the existing fire-and-forget treatment of `bp.Apply`'s leading `_ =`.

- [ ] **Step 2: Build atlas-consumables**

Run: `cd services/atlas-consumables/atlas.com/consumables && go build ./...`
Expected: build succeeds.

- [ ] **Step 3: Run the full atlas-consumables test suite**

Run: `cd services/atlas-consumables/atlas.com/consumables && go test ./...`
Expected: all PASS.

- [ ] **Step 4: Verify dispatch-order intent by code inspection (no runtime test — `bp` and `cp` are constructed inline, not injected)**

Re-read the function and confirm:
1. `bp.CancelByTypes` precedes any `cp.ChangeHP` / `cp.ChangeMP`.
2. `cp.Change*` precedes `bp.Apply`.
3. Empty `cureTypes` skips the cure dispatch (no spurious Kafka traffic).

If any of those is wrong, fix before committing.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-consumables/atlas.com/consumables/consumable/processor.go
git commit -m "feat(atlas-consumables): dispatch cure before HP/MP in ApplyItemEffects (task-051)"
```

---

## Task 10: Documentation updates

**Files:**
- Modify: `services/atlas-buffs/docs/domain.md`
- Modify: `services/atlas-buffs/docs/kafka.md`
- Modify: `services/atlas-consumables/docs/domain.md`

- [ ] **Step 1: Update `services/atlas-buffs/docs/domain.md`**

In the `Processors` section's table for `Processor`, append a row:

```markdown
| CancelByStatTypes | Cancel any buff whose Changes() intersects a stat-type set; emits one EXPIRED event per cancelled buff |
```

In the `Registry` section's table, append a row:

```markdown
| CancelByStatTypes | Filter and remove buffs whose Changes() intersects a stat-type set; returns the cancelled buffs |
```

Add a one-paragraph note at the bottom of the `Invariants` section:

```markdown
- `CancelByStatTypes` ignores `HOLY_SHIELD` (immunity gates application, not cure).
```

- [ ] **Step 2: Update `services/atlas-buffs/docs/kafka.md`**

In the `Command Types` table, add a row:

```markdown
| CANCEL_BY_TYPES | CancelByTypesCommandBody | Cancel buffs whose Changes() intersect a stat-type set |
```

After the `CancelAllCommandBody` section, add:

```markdown
##### CancelByTypesCommandBody

| Field | Type |
|-------|------|
| Types | []string |

Each entry is a `TemporaryStatType` string (`"POISON"`, `"DARKNESS"`, `"WEAKEN"`, `"SEAL"`, `"CURSE"`, etc.).
```

- [ ] **Step 3: Update `services/atlas-consumables/docs/domain.md`**

In the `consumable` section's `Processors` list, replace the `ApplyItemEffects` line with:

```markdown
- `ApplyItemEffects`: Applies cure → HP/MP recovery → status buffs from consumable data. Cure is dispatched first via a single `CANCEL_BY_TYPES` buff command when any disease cure spec (`poison`, `darkness`, `weakness`, `seal`, `curse`) is non-zero, so a queued poison tick cannot eat part of the heal.
```

In the `character/buff` `Processors` list, append:

```markdown
- `CancelByTypes`: Emits a single CANCEL_BY_TYPES buff command listing the disease stat types to cure
```

- [ ] **Step 4: Verify the markdown renders sensibly**

Visual check; no command needed. Look for table alignment and the new rows landing in the right tables.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-buffs/docs/domain.md services/atlas-buffs/docs/kafka.md services/atlas-consumables/docs/domain.md
git commit -m "docs(task-051): document cure-by-types contract"
```

---

## Task 11: Final cross-service build + test

**Files:** None modified — verification only.

- [ ] **Step 1: Build atlas-buffs**

Run: `cd services/atlas-buffs/atlas.com/buffs && go build ./...`
Expected: build succeeds.

- [ ] **Step 2: Test atlas-buffs**

Run: `cd services/atlas-buffs/atlas.com/buffs && go test ./...`
Expected: all PASS.

- [ ] **Step 3: Build atlas-consumables**

Run: `cd services/atlas-consumables/atlas.com/consumables && go build ./...`
Expected: build succeeds.

- [ ] **Step 4: Test atlas-consumables**

Run: `cd services/atlas-consumables/atlas.com/consumables && go test ./...`
Expected: all PASS.

- [ ] **Step 5: Confirm no other service references the old paths or constants**

Run from repo root: `grep -rn "CommandTypeCancelByTypes\|CancelByStatTypes\|CancelByTypes" services/ --include='*.go'`
Expected: every match is in atlas-buffs or atlas-consumables. No other service consumes or produces these new names.

If anything looks unexpected, stop and reconcile before declaring the task complete.
