# Combo Attack Orb Gain/Consume Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Server-side Combo Attack orb accumulation and finisher consumption for Crusader/Hero and Dawn Warrior, broadcast to owner and observers via the existing buff writers.

**Architecture:** atlas-channel decides whether/how the COMBO stat value changes (skill levels, effect data, double-orb roll) and emits a delta-style `UPDATE_STAT_VALUE` Kafka command; atlas-buffs applies the mutation atomically (clamped increment or absolute set) on the buff stored in its registry and emits a `STAT_UPDATED` status event carrying the buff's **original** `createdAt`/`expiresAt`; atlas-channel's buff consumer re-announces via the existing `CharacterBuffGiveWriter`/`CharacterBuffGiveForeignWriter` (the packet layer encodes duration as `expiresAt − now`, so remaining duration falls out for free). No new packets, no REST changes, no DB changes, no atlas-data changes.

**Tech Stack:** Go, Kafka (segmentio/kafka-go via libs/atlas-kafka), Redis-backed tenant registry (libs/atlas-redis, miniredis in tests), testify.

**Spec:** `docs/tasks/task-142-combo-attack-orbs/design.md` (PRD: `prd.md` in the same folder).

## Global Constraints

- Worktree: run everything from `.worktrees/task-142-combo-attack-orbs` on branch `task-142-combo-attack-orbs`. Verify with `git branch --show-current` after each commit.
- Module roots: atlas-buffs = `services/atlas-buffs/atlas.com/buffs`, atlas-channel = `services/atlas-channel/atlas.com/channel`. Run `go test`/`go vet`/`go build` from those directories.
- Command type string: `UPDATE_STAT_VALUE`. Event type string: `STAT_UPDATED`. Operation strings: `INCREMENT`, `SET`. These are wire contracts — byte-identical in both services.
- COMBO stat type string on the wire is `"COMBO"` (`character.TemporaryStatTypeCombo` in `libs/atlas-constants/character/temporary_stat.go:27`).
- Skill IDs come exclusively from `libs/atlas-constants/skill` (`CrusaderComboAttackId` 1111002, `CrusaderPanicSwordId` 1111003, `CrusaderPanicAxeId` 1111004, `CrusaderComaSwordId` 1111005, `CrusaderComaAxeId` 1111006, `CrusaderShoutId` 1111008, `HeroAdvancedComboAttackId` 1120003, `DawnWarriorStage3ComboAttackId` 11111001, `DawnWarriorStage3PanicId` 11111002, `DawnWarriorStage3ComaId` 11111003, `DawnWarriorStage3AdvancedComboId` 11110005). Never hardcode numeric skill ids.
- COMBO stat semantics: value = orb count + 1. Cast applies value 1 (unchanged, atlas-data already does this). Gain clamps to `x + 1` of the governing effect. Finisher consume = SET 1. Value must never exceed cap or go below 1.
- Failure isolation: orb bookkeeping failures are logged and swallowed; the attack pipeline never fails or retries because of them.
- No `// TODO` left behind for this feature; the combo TODO line is removed when wired.
- Do not create `*_testhelpers.go` files; test helper funcs live inside `_test.go` files (existing convention, e.g. `newDoomEffect`).
- Final verification (Task 11): `go test -race ./...`, `go vet ./...`, `go build ./...` in both modules; `docker buildx bake atlas-channel atlas-buffs` from the worktree root; `tools/redis-key-guard.sh` from the worktree root (no `GOWORK=off` prefix).

---

### Task 1: atlas-buffs wire contract — UPDATE_STAT_VALUE command + STAT_UPDATED event

**Files:**
- Modify: `services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go`
- Test: `services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka_test.go`

**Interfaces:**
- Consumes: existing `Command[E]` / `StatusEvent[E]` envelopes and `StatChange` in the same file.
- Produces (later tasks rely on these exact names): constants `CommandTypeUpdateStatValue = "UPDATE_STAT_VALUE"`, `StatOperationIncrement = "INCREMENT"`, `StatOperationSet = "SET"`, `EventStatusTypeStatUpdated = "STAT_UPDATED"`; types `UpdateStatValueCommandBody{SourceId int32; StatType string; Operation string; Amount int32; Cap int32}` and `StatUpdatedStatusEventBody{SourceId int32; Level byte; Duration int32; Changes []StatChange; CreatedAt time.Time; ExpiresAt time.Time}`.

- [ ] **Step 1: Write the failing tests**

Append to `services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka_test.go` (the file already has the canonical-JSON pattern for APPLY; mirror it):

```go
// canonicalUpdateStatValueBody is the exact JSON the UPDATE_STAT_VALUE command
// body must serialize to. The identical literal is asserted in the
// atlas-channel mirror (services/atlas-channel/atlas.com/channel/kafka/message/buff/kafka_test.go)
// so the two re-declared contracts stay byte-identical on the wire.
const canonicalUpdateStatValueBody = `{"sourceId":1111002,"statType":"COMBO","operation":"INCREMENT","amount":2,"cap":6}`

func TestUpdateStatValueCommandBody_CanonicalJSON(t *testing.T) {
	b, err := json.Marshal(UpdateStatValueCommandBody{
		SourceId:  1111002,
		StatType:  "COMBO",
		Operation: StatOperationIncrement,
		Amount:    2,
		Cap:       6,
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(b) != canonicalUpdateStatValueBody {
		t.Fatalf("canonical mismatch.\n got: %s\nwant: %s", b, canonicalUpdateStatValueBody)
	}
}

func TestStatUpdatedStatusEventBody_RoundTrip(t *testing.T) {
	created := time.Date(2026, 7, 9, 12, 0, 0, 0, time.UTC)
	expires := created.Add(150 * time.Second)
	in := StatUpdatedStatusEventBody{
		SourceId:  1111002,
		Level:     20,
		Duration:  150000,
		Changes:   []StatChange{{Type: "COMBO", Amount: 3}},
		CreatedAt: created,
		ExpiresAt: expires,
	}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got StatUpdatedStatusEventBody
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.SourceId != in.SourceId || got.Level != in.Level || got.Duration != in.Duration ||
		len(got.Changes) != 1 || got.Changes[0] != in.Changes[0] ||
		!got.CreatedAt.Equal(in.CreatedAt) || !got.ExpiresAt.Equal(in.ExpiresAt) {
		t.Fatalf("round-trip mismatch: %+v vs %+v", got, in)
	}
}
```

Add `"time"` to the test file's imports.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-buffs/atlas.com/buffs && go test ./kafka/message/character/ -run 'UpdateStatValue|StatUpdated' -v`
Expected: FAIL to compile — `undefined: UpdateStatValueCommandBody`, `undefined: StatOperationIncrement`, `undefined: StatUpdatedStatusEventBody`.

- [ ] **Step 3: Add the contract to kafka.go**

In `services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go`, extend the command constants block:

```go
const (
	EnvCommandTopic            = "COMMAND_TOPIC_CHARACTER_BUFF"
	CommandTypeApply           = "APPLY"
	CommandTypeCancel          = "CANCEL"
	CommandTypeCancelAll       = "CANCEL_ALL"
	CommandTypeCancelByTypes   = "CANCEL_BY_TYPES"
	CommandTypeUpdateStatValue = "UPDATE_STAT_VALUE"

	// Operations for UPDATE_STAT_VALUE. INCREMENT adds Amount clamped to Cap;
	// SET replaces the stat amount outright (finisher consume = SET 1).
	StatOperationIncrement = "INCREMENT"
	StatOperationSet       = "SET"
)
```

(Replace the existing `const` block containing the four command types — keep the same names/values, add the three new ones.)

After `CancelByTypesCommandBody`, add:

```go
// UpdateStatValueCommandBody changes the amount of one stat on a character's
// existing buff (identified by SourceId). The body is stat-generic; task-142
// uses it for COMBO orb bookkeeping. Cap applies to INCREMENT only.
type UpdateStatValueCommandBody struct {
	SourceId  int32  `json:"sourceId"`
	StatType  string `json:"statType"`
	Operation string `json:"operation"`
	Amount    int32  `json:"amount"`
	Cap       int32  `json:"cap"`
}
```

Extend the event constants block:

```go
const (
	EnvEventStatusTopic        = "EVENT_TOPIC_CHARACTER_BUFF_STATUS"
	EventStatusTypeBuffApplied = "APPLIED"
	EventStatusTypeBuffExpired = "EXPIRED"
	EventStatusTypeStatUpdated = "STAT_UPDATED"
)
```

After `ExpiredStatusEventBody`, add:

```go
// StatUpdatedStatusEventBody is emitted when a stat value on an existing buff
// changed (not a new buff — consumers that react to APPLIED as "a buff came
// into existence" must ignore this type). CreatedAt/ExpiresAt are the buff's
// ORIGINAL timestamps so re-broadcast carries the remaining duration.
type StatUpdatedStatusEventBody struct {
	SourceId  int32        `json:"sourceId"`
	Level     byte         `json:"level"`
	Duration  int32        `json:"duration"`
	Changes   []StatChange `json:"changes"`
	CreatedAt time.Time    `json:"createdAt"`
	ExpiresAt time.Time    `json:"expiresAt"`
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-buffs/atlas.com/buffs && go test ./kafka/message/character/ -v`
Expected: PASS (all, including the pre-existing APPLY tests).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-buffs/atlas.com/buffs/kafka/message/character/
git commit -m "feat(buffs): UPDATE_STAT_VALUE command and STAT_UPDATED event contract"
```

---

### Task 2: atlas-buffs — buff.Model.WithStatAmount

**Files:**
- Modify: `services/atlas-buffs/atlas.com/buffs/buff/model.go`
- Test: `services/atlas-buffs/atlas.com/buffs/buff/model_test.go`

**Interfaces:**
- Consumes: existing `Model` (private fields `id, sourceId, level, duration, changes, createdAt, expiresAt`), `stat.NewStat(statType string, amount int32)`.
- Produces: `func (m Model) WithStatAmount(statType string, amount int32) (Model, bool)` — copy with that stat's amount replaced; `false` when the stat type isn't present. Task 3's registry calls this.

- [ ] **Step 1: Write the failing tests**

Append to `services/atlas-buffs/atlas.com/buffs/buff/model_test.go`:

```go
func TestModel_WithStatAmount_ReplacesTargetStat(t *testing.T) {
	changes := []stat.Model{stat.NewStat("COMBO", 1), stat.NewStat("WATK", 20)}
	m, err := NewBuff(1111002, 20, 150000, changes)
	if err != nil {
		t.Fatalf("NewBuff: %v", err)
	}

	updated, ok := m.WithStatAmount("COMBO", 3)
	if !ok {
		t.Fatal("expected ok=true for present stat type")
	}

	var combo, watk int32
	for _, c := range updated.Changes() {
		switch c.Type() {
		case "COMBO":
			combo = c.Amount()
		case "WATK":
			watk = c.Amount()
		}
	}
	if combo != 3 {
		t.Fatalf("COMBO amount = %d, want 3", combo)
	}
	if watk != 20 {
		t.Fatalf("WATK amount = %d, want 20 (other stats preserved)", watk)
	}
}

func TestModel_WithStatAmount_PreservesIdentityAndExpiry(t *testing.T) {
	m, err := NewBuff(1111002, 20, 150000, []stat.Model{stat.NewStat("COMBO", 1)})
	if err != nil {
		t.Fatalf("NewBuff: %v", err)
	}

	updated, ok := m.WithStatAmount("COMBO", 2)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if updated.SourceId() != m.SourceId() || updated.Level() != m.Level() || updated.Duration() != m.Duration() {
		t.Fatal("identity fields must be preserved")
	}
	if !updated.CreatedAt().Equal(m.CreatedAt()) || !updated.ExpiresAt().Equal(m.ExpiresAt()) {
		t.Fatal("createdAt/expiresAt must be preserved (remaining-duration contract)")
	}
	// original untouched (immutability)
	if m.Changes()[0].Amount() != 1 {
		t.Fatalf("original buff mutated: COMBO = %d, want 1", m.Changes()[0].Amount())
	}
}

func TestModel_WithStatAmount_MissingStatType(t *testing.T) {
	m, err := NewBuff(1111002, 20, 150000, []stat.Model{stat.NewStat("WATK", 20)})
	if err != nil {
		t.Fatalf("NewBuff: %v", err)
	}
	if _, ok := m.WithStatAmount("COMBO", 3); ok {
		t.Fatal("expected ok=false for absent stat type")
	}
}
```

No import changes needed — `model_test.go` already imports `atlas-buffs/buff/stat`, `testing`, `time`, and testify (verified during planning).

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-buffs/atlas.com/buffs && go test ./buff/ -run WithStatAmount -v`
Expected: FAIL to compile — `m.WithStatAmount undefined`.

- [ ] **Step 3: Implement WithStatAmount**

Append to `services/atlas-buffs/atlas.com/buffs/buff/model.go` (after `ExpiresAt()`, before `MarshalJSON`):

```go
// WithStatAmount returns a copy of the buff with the amount of the stat of
// the given type replaced, preserving identity (id, sourceId, level,
// duration), the other stats, and the ORIGINAL createdAt/expiresAt — value
// updates must not extend the buff's lifetime. The second return is false
// when the buff has no stat of that type.
func (m Model) WithStatAmount(statType string, amount int32) (Model, bool) {
	found := false
	changes := make([]stat.Model, 0, len(m.changes))
	for _, c := range m.changes {
		if c.Type() == statType {
			changes = append(changes, stat.NewStat(statType, amount))
			found = true
		} else {
			changes = append(changes, c)
		}
	}
	if !found {
		return Model{}, false
	}
	return Model{
		id:        m.id,
		sourceId:  m.sourceId,
		level:     m.level,
		duration:  m.duration,
		changes:   changes,
		createdAt: m.createdAt,
		expiresAt: m.expiresAt,
	}, true
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-buffs/atlas.com/buffs && go test ./buff/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-buffs/atlas.com/buffs/buff/
git commit -m "feat(buffs): buff.Model.WithStatAmount copy-mutator"
```

---

### Task 3: atlas-buffs — Registry.UpdateStatValue

**Files:**
- Modify: `services/atlas-buffs/atlas.com/buffs/character/registry.go`
- Test: `services/atlas-buffs/atlas.com/buffs/character/registry_test.go`

**Interfaces:**
- Consumes: Task 1's `character2.StatOperationIncrement`/`StatOperationSet` (import `character2 "atlas-buffs/kafka/message/character"` — registry.go is in the same package as processor.go, which already uses that alias); Task 2's `buff.Model.WithStatAmount`; existing `srcKey`, `r.characters` get/put.
- Produces: `func (r *Registry) UpdateStatValue(ctx context.Context, characterId uint32, sourceId int32, statType string, operation string, amount int32, capValue int32) (buff.Model, bool, error)` — returns the updated buff and `true` only when a mutation was stored. Task 4's processor calls this.

- [ ] **Step 1: Write the failing tests**

Append to `services/atlas-buffs/atlas.com/buffs/character/registry_test.go` (reuses the file's existing `setupTestRegistry`/`setupTestTenant`/`setupTestContext` helpers):

```go
func setupComboBuff(t *testing.T, ctx context.Context, characterId uint32, sourceId int32) {
	t.Helper()
	changes := []stat.Model{stat.NewStat("COMBO", 1)}
	_, err := GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), characterId, sourceId, byte(20), int32(150000), changes, false)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
}

func comboAmount(t *testing.T, ctx context.Context, characterId uint32, sourceId int32) int32 {
	t.Helper()
	m, err := GetRegistry().Get(ctx, characterId)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	b, ok := m.Buffs()[srcKey(sourceId)]
	if !ok {
		t.Fatalf("no buff under srcKey(%d)", sourceId)
	}
	for _, c := range b.Changes() {
		if c.Type() == "COMBO" {
			return c.Amount()
		}
	}
	t.Fatal("no COMBO stat on buff")
	return 0
}

func TestRegistry_UpdateStatValue_Increment(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))
	setupComboBuff(t, ctx, 1000, 1111002)

	updated, changed, err := GetRegistry().UpdateStatValue(ctx, 1000, 1111002, "COMBO", character2.StatOperationIncrement, 1, 6)
	assert.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, int32(2), comboAmount(t, ctx, 1000, 1111002))

	var got int32
	for _, c := range updated.Changes() {
		if c.Type() == "COMBO" {
			got = c.Amount()
		}
	}
	assert.Equal(t, int32(2), got)
}

func TestRegistry_UpdateStatValue_IncrementClampsAtCap(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))
	setupComboBuff(t, ctx, 1000, 1111002)

	// 1 -> +2 (double orb) with cap 2 must land exactly on the cap, not past it.
	_, changed, err := GetRegistry().UpdateStatValue(ctx, 1000, 1111002, "COMBO", character2.StatOperationIncrement, 2, 2)
	assert.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, int32(2), comboAmount(t, ctx, 1000, 1111002))
}

func TestRegistry_UpdateStatValue_NoChangeAtCap(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))
	setupComboBuff(t, ctx, 1000, 1111002)

	_, changed, err := GetRegistry().UpdateStatValue(ctx, 1000, 1111002, "COMBO", character2.StatOperationIncrement, 1, 6)
	assert.NoError(t, err)
	assert.True(t, changed) // 1 -> 2

	// drive to cap 2, then verify at-cap increment is a no-op
	_, changed, err = GetRegistry().UpdateStatValue(ctx, 1000, 1111002, "COMBO", character2.StatOperationIncrement, 5, 2)
	assert.NoError(t, err)
	assert.False(t, changed, "already at/above cap must be a no-op")
	assert.Equal(t, int32(2), comboAmount(t, ctx, 1000, 1111002))
}

func TestRegistry_UpdateStatValue_SetResets(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))
	setupComboBuff(t, ctx, 1000, 1111002)

	_, _, _ = GetRegistry().UpdateStatValue(ctx, 1000, 1111002, "COMBO", character2.StatOperationIncrement, 4, 6)
	assert.Equal(t, int32(5), comboAmount(t, ctx, 1000, 1111002))

	_, changed, err := GetRegistry().UpdateStatValue(ctx, 1000, 1111002, "COMBO", character2.StatOperationSet, 1, 0)
	assert.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, int32(1), comboAmount(t, ctx, 1000, 1111002))
}

func TestRegistry_UpdateStatValue_SetSameValueNoOp(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))
	setupComboBuff(t, ctx, 1000, 1111002)

	_, changed, err := GetRegistry().UpdateStatValue(ctx, 1000, 1111002, "COMBO", character2.StatOperationSet, 1, 0)
	assert.NoError(t, err)
	assert.False(t, changed, "SET to the current value must be a no-op")
}

func TestRegistry_UpdateStatValue_NoOps(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))
	setupComboBuff(t, ctx, 1000, 1111002)

	cases := []struct {
		name        string
		characterId uint32
		sourceId    int32
		statType    string
		operation   string
		amount      int32
	}{
		{"unknown character", 9999, 1111002, "COMBO", character2.StatOperationIncrement, 1},
		{"wrong sourceId", 1000, 11111001, "COMBO", character2.StatOperationIncrement, 1},
		{"wrong stat type", 1000, 1111002, "MORPH", character2.StatOperationIncrement, 1},
		{"unknown operation", 1000, 1111002, "COMBO", "MULTIPLY", 1},
		{"non-positive increment", 1000, 1111002, "COMBO", character2.StatOperationIncrement, 0},
		{"set below 1", 1000, 1111002, "COMBO", character2.StatOperationSet, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, changed, err := GetRegistry().UpdateStatValue(ctx, tc.characterId, tc.sourceId, tc.statType, tc.operation, tc.amount, 6)
			assert.NoError(t, err)
			assert.False(t, changed)
		})
	}
	assert.Equal(t, int32(1), comboAmount(t, ctx, 1000, 1111002), "no-op paths must not mutate the value")
}

func TestRegistry_UpdateStatValue_ExpiredBuffNoOp(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))

	changes := []stat.Model{stat.NewStat("COMBO", 1)}
	_, err := GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), 1000, 1111002, byte(20), int32(1), changes, false)
	assert.NoError(t, err)
	time.Sleep(5 * time.Millisecond) // duration is 1ms; let it lapse

	_, changed, err := GetRegistry().UpdateStatValue(ctx, 1000, 1111002, "COMBO", character2.StatOperationIncrement, 1, 6)
	assert.NoError(t, err)
	assert.False(t, changed, "expired buff must be a no-op")
}

func TestRegistry_UpdateStatValue_PreservesTimestamps(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))
	setupComboBuff(t, ctx, 1000, 1111002)

	before, err := GetRegistry().Get(ctx, 1000)
	assert.NoError(t, err)
	orig := before.Buffs()[srcKey(1111002)]

	updated, changed, err := GetRegistry().UpdateStatValue(ctx, 1000, 1111002, "COMBO", character2.StatOperationIncrement, 1, 6)
	assert.NoError(t, err)
	assert.True(t, changed)
	assert.True(t, updated.CreatedAt().Equal(orig.CreatedAt()), "createdAt must be unchanged")
	assert.True(t, updated.ExpiresAt().Equal(orig.ExpiresAt()), "expiresAt must be unchanged (buff must not extend)")
}
```

Add `character2 "atlas-buffs/kafka/message/character"` to the test file's imports (`context`, `time`, `world`, `channel`, `stat`, `assert` are already there).

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-buffs/atlas.com/buffs && go test ./character/ -run UpdateStatValue -v`
Expected: FAIL to compile — `GetRegistry().UpdateStatValue undefined`.

- [ ] **Step 3: Implement Registry.UpdateStatValue**

Append to `services/atlas-buffs/atlas.com/buffs/character/registry.go` (after `Cancel`), and add `character2 "atlas-buffs/kafka/message/character"` to its imports:

```go
// UpdateStatValue changes the amount of one stat on the character's active
// buff for sourceId. INCREMENT adds amount clamped to capValue (no-op when
// already at/above cap); SET replaces the amount outright. Returns the
// updated buff and true when a mutation was stored; (Model{}, false, nil)
// when the buff is missing/expired, lacks the stat, the operation is
// unknown, or the value would not change. Only whole-source
// (non-accumulate) buffs are addressed via srcKey — accumulate-mode
// per-stat buffs are out of scope for value updates. Defensive floors keep
// the value from ever exceeding cap or dropping below 1. Same
// get-modify-put shape as Cancel, serialized per character by the command
// topic's characterId partition key.
func (r *Registry) UpdateStatValue(ctx context.Context, characterId uint32, sourceId int32, statType string, operation string, amount int32, capValue int32) (buff.Model, bool, error) {
	t := tenant.MustFromContext(ctx)

	m, err := r.characters.Get(ctx, t, characterId)
	if errors.Is(err, atlas.ErrNotFound) {
		return buff.Model{}, false, nil
	}
	if err != nil {
		return buff.Model{}, false, err
	}

	b, ok := m.buffs[srcKey(sourceId)]
	if !ok || b.Expired() {
		return buff.Model{}, false, nil
	}

	var current int32
	found := false
	for _, c := range b.Changes() {
		if c.Type() == statType {
			current = c.Amount()
			found = true
			break
		}
	}
	if !found {
		return buff.Model{}, false, nil
	}

	var next int32
	switch operation {
	case character2.StatOperationIncrement:
		if amount <= 0 || current >= capValue {
			return buff.Model{}, false, nil
		}
		next = current + amount
		if next > capValue {
			next = capValue
		}
	case character2.StatOperationSet:
		if amount < 1 {
			return buff.Model{}, false, nil
		}
		next = amount
	default:
		return buff.Model{}, false, nil
	}
	if next == current {
		return buff.Model{}, false, nil
	}

	updated, ok := b.WithStatAmount(statType, next)
	if !ok {
		return buff.Model{}, false, nil
	}
	m.buffs[srcKey(sourceId)] = updated
	if err := r.characters.Put(ctx, t, characterId, m); err != nil {
		return buff.Model{}, false, err
	}
	return updated, true, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-buffs/atlas.com/buffs && go test ./character/ -v`
Expected: PASS (all, including pre-existing registry/processor tests).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-buffs/atlas.com/buffs/character/
git commit -m "feat(buffs): Registry.UpdateStatValue clamped stat mutation"
```

---

### Task 4: atlas-buffs — Processor.UpdateStatValue + STAT_UPDATED producer

**Files:**
- Modify: `services/atlas-buffs/atlas.com/buffs/character/processor.go`
- Modify: `services/atlas-buffs/atlas.com/buffs/character/producer.go`
- Test: `services/atlas-buffs/atlas.com/buffs/character/processor_test.go`

**Interfaces:**
- Consumes: Task 3's `Registry.UpdateStatValue`; Task 1's constants/body; existing `message.Emit`, `producer.SingleMessageProvider`.
- Produces: `UpdateStatValue(worldId world.Id, characterId uint32, sourceId int32, statType string, operation string, amount int32, capValue int32) error` on the `Processor` interface (Task 5's consumer calls it), and `statUpdatedStatusEventProvider(...)` in producer.go.

- [ ] **Step 1: Write the failing tests**

Append to `services/atlas-buffs/atlas.com/buffs/character/processor_test.go`. Note the existing convention: processor calls that emit are invoked with `_ =` (no Kafka broker in tests) and assertions run against the registry.

```go
func TestProcessor_UpdateStatValue_Increment(t *testing.T) {
	processor, _, ctx := setupProcessorTest(t)

	changes := []stat.Model{stat.NewStat("COMBO", 1)}
	_ = processor.Apply(world.Id(0), channel.Id(0), 1000, 1000, 1111002, byte(20), int32(150000), changes, false)

	_ = processor.UpdateStatValue(world.Id(0), 1000, 1111002, "COMBO", character2.StatOperationIncrement, 2, 6)

	m, err := GetRegistry().Get(ctx, 1000)
	assert.NoError(t, err)
	b := m.Buffs()[srcKey(1111002)]
	assert.Equal(t, int32(3), b.Changes()[0].Amount())
}

func TestProcessor_UpdateStatValue_UnknownOperationIsNoOp(t *testing.T) {
	processor, _, ctx := setupProcessorTest(t)

	changes := []stat.Model{stat.NewStat("COMBO", 1)}
	_ = processor.Apply(world.Id(0), channel.Id(0), 1000, 1000, 1111002, byte(20), int32(150000), changes, false)

	err := processor.UpdateStatValue(world.Id(0), 1000, 1111002, "COMBO", "MULTIPLY", 2, 6)
	assert.NoError(t, err, "unknown operation is a logged no-op, not an error")

	m, err := GetRegistry().Get(ctx, 1000)
	assert.NoError(t, err)
	b := m.Buffs()[srcKey(1111002)]
	assert.Equal(t, int32(1), b.Changes()[0].Amount())
}

func TestProcessor_UpdateStatValue_MissingBuffIsNoOp(t *testing.T) {
	processor, _, _ := setupProcessorTest(t)
	err := processor.UpdateStatValue(world.Id(0), 1000, 1111002, "COMBO", character2.StatOperationIncrement, 1, 6)
	assert.NoError(t, err, "missing buff is a logged no-op, not an error")
}
```

Add `character2 "atlas-buffs/kafka/message/character"` to the test file's imports.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-buffs/atlas.com/buffs && go test ./character/ -run Processor_UpdateStatValue -v`
Expected: FAIL to compile — `processor.UpdateStatValue undefined`.

- [ ] **Step 3: Implement the producer provider**

Append to `services/atlas-buffs/atlas.com/buffs/character/producer.go` (after `expiredStatusEventProvider`; all imports already present):

```go
func statUpdatedStatusEventProvider(worldId world.Id, characterId uint32, sourceId int32, level byte, duration int32, changes []stat.Model, createdAt time.Time, expiresAt time.Time) model.Provider[[]kafka.Message] {
	statups := make([]character2.StatChange, 0)
	for _, su := range changes {
		statups = append(statups, character2.StatChange{
			Type:   su.Type(),
			Amount: su.Amount(),
		})
	}

	key := producer.CreateKey(int(characterId))
	value := &character2.StatusEvent[character2.StatUpdatedStatusEventBody]{
		WorldId:     worldId,
		CharacterId: characterId,
		Type:        character2.EventStatusTypeStatUpdated,
		Body: character2.StatUpdatedStatusEventBody{
			SourceId:  sourceId,
			Level:     level,
			Duration:  duration,
			Changes:   statups,
			CreatedAt: createdAt,
			ExpiresAt: expiresAt,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

- [ ] **Step 4: Implement the processor method**

In `services/atlas-buffs/atlas.com/buffs/character/processor.go`, add to the `Processor` interface after `CancelByStatTypes`:

```go
	UpdateStatValue(worldId world.Id, characterId uint32, sourceId int32, statType string, operation string, amount int32, capValue int32) error
```

Add the implementation after `CancelByStatTypes`:

```go
// UpdateStatValue applies a stat-value mutation to an existing buff and, when
// the value actually changed, emits a STAT_UPDATED status event carrying the
// buff's original createdAt/expiresAt (so the channel re-broadcasts the
// remaining duration). Missing/expired buff and at-cap increments are Debug
// no-ops — the buff can lapse between the channel's attack and this command.
func (p *ProcessorImpl) UpdateStatValue(worldId world.Id, characterId uint32, sourceId int32, statType string, operation string, amount int32, capValue int32) error {
	if operation != character2.StatOperationIncrement && operation != character2.StatOperationSet {
		p.l.Warnf("Unknown stat value operation [%s] for character [%d] buff [%d]; ignoring.", operation, characterId, sourceId)
		return nil
	}
	return message.Emit(p.l, p.ctx)(func(buf *message.Buffer) error {
		updated, changed, err := GetRegistry().UpdateStatValue(p.ctx, characterId, sourceId, statType, operation, amount, capValue)
		if err != nil {
			return err
		}
		if !changed {
			p.l.Debugf("No stat value change for character [%d] buff [%d] stat [%s].", characterId, sourceId, statType)
			return nil
		}
		return buf.Put(character2.EnvEventStatusTopic, statUpdatedStatusEventProvider(worldId, characterId, updated.SourceId(), updated.Level(), updated.Duration(), updated.Changes(), updated.CreatedAt(), updated.ExpiresAt()))
	})
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-buffs/atlas.com/buffs && go test ./character/ -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-buffs/atlas.com/buffs/character/
git commit -m "feat(buffs): Processor.UpdateStatValue emits STAT_UPDATED on change"
```

---

### Task 5: atlas-buffs — consume UPDATE_STAT_VALUE

**Files:**
- Modify: `services/atlas-buffs/atlas.com/buffs/kafka/consumer/character/consumer.go`

**Interfaces:**
- Consumes: Task 1's `CommandTypeUpdateStatValue`/`UpdateStatValueCommandBody`; Task 4's `Processor.UpdateStatValue`.
- Produces: `handleUpdateStatValue` registered on the buff command topic (this completes the atlas-buffs side end-to-end).

Consumer handlers in this repo have no unit tests (type-guard + processor delegation only); verification is compile + vet + the processor/registry tests from Tasks 3–4.

- [ ] **Step 1: Register and implement the handler**

In `services/atlas-buffs/atlas.com/buffs/kafka/consumer/character/consumer.go`, inside `InitHandlers` after the `handleCancelByTypes` registration:

```go
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleUpdateStatValue))); err != nil {
			return err
		}
```

At the end of the file:

```go
func handleUpdateStatValue(l logrus.FieldLogger, ctx context.Context, c character2.Command[character2.UpdateStatValueCommandBody]) {
	if c.Type != character2.CommandTypeUpdateStatValue {
		return
	}

	if err := character.NewProcessor(l, ctx).UpdateStatValue(c.WorldId, c.CharacterId, c.Body.SourceId, c.Body.StatType, c.Body.Operation, c.Body.Amount, c.Body.Cap); err != nil {
		l.WithError(err).Errorf("Unable to update stat value on buff [%d] for character [%d].", c.Body.SourceId, c.CharacterId)
	}
}
```

- [ ] **Step 2: Verify build, vet, and full module tests**

Run: `cd services/atlas-buffs/atlas.com/buffs && go build ./... && go vet ./... && go test -race ./...`
Expected: all clean/PASS.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-buffs/atlas.com/buffs/kafka/consumer/character/
git commit -m "feat(buffs): consume UPDATE_STAT_VALUE buff command"
```

---

### Task 6: atlas-channel — wire contract mirror

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/buff/kafka.go`
- Create: `services/atlas-channel/atlas.com/channel/kafka/message/buff/kafka_test.go`

**Interfaces:**
- Consumes: existing channel-side `Command[E]`/`StatusEvent[E]`/`StatChange` in the same file.
- Produces (channel-side names, used by Tasks 7–10): `CommandTypeUpdateStatValue`, `StatOperationIncrement`, `StatOperationSet`, `EventStatusTypeStatUpdated`, `UpdateStatValueCommandBody`, `StatUpdatedStatusEventBody` — identical field names/tags/order to Task 1 (the channel keeps its own copy of the shapes, as it does for APPLY/CANCEL).

- [ ] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/kafka/message/buff/kafka_test.go`:

```go
package buff

import (
	"encoding/json"
	"testing"
	"time"
)

// canonicalUpdateStatValueBody is the exact JSON the UPDATE_STAT_VALUE command
// body must serialize to. The identical literal is asserted in the atlas-buffs
// owner contract (services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka_test.go)
// so the two re-declared contracts stay byte-identical on the wire.
const canonicalUpdateStatValueBody = `{"sourceId":1111002,"statType":"COMBO","operation":"INCREMENT","amount":2,"cap":6}`

func TestUpdateStatValueCommandBody_CanonicalJSON(t *testing.T) {
	b, err := json.Marshal(UpdateStatValueCommandBody{
		SourceId:  1111002,
		StatType:  "COMBO",
		Operation: StatOperationIncrement,
		Amount:    2,
		Cap:       6,
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(b) != canonicalUpdateStatValueBody {
		t.Fatalf("canonical mismatch.\n got: %s\nwant: %s", b, canonicalUpdateStatValueBody)
	}
}

func TestStatUpdatedStatusEventBody_RoundTrip(t *testing.T) {
	created := time.Date(2026, 7, 9, 12, 0, 0, 0, time.UTC)
	expires := created.Add(150 * time.Second)
	in := StatUpdatedStatusEventBody{
		SourceId:  1111002,
		Level:     20,
		Duration:  150000,
		Changes:   []StatChange{{Type: "COMBO", Amount: 3}},
		CreatedAt: created,
		ExpiresAt: expires,
	}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got StatUpdatedStatusEventBody
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.SourceId != in.SourceId || got.Level != in.Level || got.Duration != in.Duration ||
		len(got.Changes) != 1 || got.Changes[0] != in.Changes[0] ||
		!got.CreatedAt.Equal(in.CreatedAt) || !got.ExpiresAt.Equal(in.ExpiresAt) {
		t.Fatalf("round-trip mismatch: %+v vs %+v", got, in)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./kafka/message/buff/ -v`
Expected: FAIL to compile — `undefined: UpdateStatValueCommandBody` etc.

- [ ] **Step 3: Add the contract to the channel kafka.go**

In `services/atlas-channel/atlas.com/channel/kafka/message/buff/kafka.go`, replace the command constants block:

```go
const (
	EnvCommandTopic            = "COMMAND_TOPIC_CHARACTER_BUFF"
	CommandTypeApply           = "APPLY"
	CommandTypeCancel          = "CANCEL"
	CommandTypeUpdateStatValue = "UPDATE_STAT_VALUE"

	// Operations for UPDATE_STAT_VALUE. INCREMENT adds Amount clamped to Cap;
	// SET replaces the stat amount outright (finisher consume = SET 1).
	StatOperationIncrement = "INCREMENT"
	StatOperationSet       = "SET"
)
```

After `CancelCommandBody`, add:

```go
// UpdateStatValueCommandBody changes the amount of one stat on a character's
// existing buff (identified by SourceId). Owned by atlas-buffs; this is the
// channel-side mirror. Cap applies to INCREMENT only.
type UpdateStatValueCommandBody struct {
	SourceId  int32  `json:"sourceId"`
	StatType  string `json:"statType"`
	Operation string `json:"operation"`
	Amount    int32  `json:"amount"`
	Cap       int32  `json:"cap"`
}
```

Replace the event constants block:

```go
const (
	EnvEventStatusTopic        = "EVENT_TOPIC_CHARACTER_BUFF_STATUS"
	EventStatusTypeBuffApplied = "APPLIED"
	EventStatusTypeBuffExpired = "EXPIRED"
	EventStatusTypeStatUpdated = "STAT_UPDATED"
)
```

After `ExpiredStatusEventBody`, add:

```go
// StatUpdatedStatusEventBody signals a stat value change on an EXISTING buff.
// CreatedAt/ExpiresAt are the buff's original timestamps — the give writers
// encode duration as expiresAt − now, so re-broadcast carries the remaining
// duration and never extends the buff.
type StatUpdatedStatusEventBody struct {
	SourceId  int32        `json:"sourceId"`
	Level     byte         `json:"level"`
	Duration  int32        `json:"duration"`
	Changes   []StatChange `json:"changes"`
	CreatedAt time.Time    `json:"createdAt"`
	ExpiresAt time.Time    `json:"expiresAt"`
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./kafka/message/buff/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/message/buff/
git commit -m "feat(channel): mirror UPDATE_STAT_VALUE/STAT_UPDATED buff contract"
```

---

### Task 7: atlas-channel — buff producer + Processor.UpdateStatValue

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/character/buff/producer.go`
- Modify: `services/atlas-channel/atlas.com/channel/character/buff/processor.go`

**Interfaces:**
- Consumes: Task 6's channel-side contract; existing `producer.CreateKey`, `producer.ProviderImpl` wrapper, `field.Model`.
- Produces: `UpdateStatValueCommandProvider(f field.Model, characterId uint32, sourceId int32, statType string, operation string, amount int32, capValue int32) model.Provider[[]kafka.Message]` and `Processor.UpdateStatValue(f field.Model, characterId uint32, sourceId int32, statType string, operation string, amount int32, capValue int32) error`. Task 9's wiring calls the processor method.

No test file exists for this package (producers/processors here are thin emit wrappers; the wire shape is pinned by Task 6's canonical test). Verification is compile + vet.

- [ ] **Step 1: Add the command provider**

Append to `services/atlas-channel/atlas.com/channel/character/buff/producer.go` (imports already present):

```go
func UpdateStatValueCommandProvider(f field.Model, characterId uint32, sourceId int32, statType string, operation string, amount int32, capValue int32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &buff.Command[buff.UpdateStatValueCommandBody]{
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		MapId:       f.MapId(),
		Instance:    f.Instance(),
		CharacterId: characterId,
		Type:        buff.CommandTypeUpdateStatValue,
		Body: buff.UpdateStatValueCommandBody{
			SourceId:  sourceId,
			StatType:  statType,
			Operation: operation,
			Amount:    amount,
			Cap:       capValue,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

- [ ] **Step 2: Add the processor method**

In `services/atlas-channel/atlas.com/channel/character/buff/processor.go`, add to the `Processor` interface after `Cancel`:

```go
	UpdateStatValue(f field.Model, characterId uint32, sourceId int32, statType string, operation string, amount int32, capValue int32) error
```

Add the implementation after `Cancel`:

```go
func (p *ProcessorImpl) UpdateStatValue(f field.Model, characterId uint32, sourceId int32, statType string, operation string, amount int32, capValue int32) error {
	p.l.Debugf("Character [%d] updating stat [%s] on buff [%d]: %s %d (cap %d).", characterId, statType, sourceId, operation, amount, capValue)
	return producer.ProviderImpl(p.l)(p.ctx)(buff2.EnvCommandTopic)(UpdateStatValueCommandProvider(f, characterId, sourceId, statType, operation, amount, capValue))
}
```

- [ ] **Step 3: Verify build and vet**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go vet ./...`
Expected: clean. (No other type implements `buff.Processor` — verified during planning — so the interface addition breaks nothing.)

- [ ] **Step 4: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/character/buff/
git commit -m "feat(channel): buff UpdateStatValue command emitter"
```

---

### Task 8: atlas-channel — combo orb logic (new handler file + tests)

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_combo.go`
- Test: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_combo_test.go`

**Interfaces:**
- Consumes: Task 6's `buff2.StatOperationIncrement`/`StatOperationSet`; Task 7's `buff.Processor.UpdateStatValue`; existing `character.Model.Skills()`, `skill.Model` (`Id() skill3.Id`, `Level() byte`), `effect.Model` (`X() int16`, `Prop() float64`), `skill2.NewProcessor(l, ctx).GetEffect(uniqueId uint32, level byte) (effect.Model, error)`, `packetmodel.AttackInfo` (`SkillId() uint32`, `DamageInfo() []DamageInfo`).
- Produces: `comboLine` struct, `comboSkillIds([]skill.Model) (comboLine, bool)`, `isComboFinisher(skill3.Id) bool`, `comboGainAmount(advLearned bool, prop float64, roll float64) int32`, `comboOrbDeps` struct, `comboOrbProductionDeps(l, ctx, f, characterId) comboOrbDeps`, `comboOrbTryUpdate(l logrus.FieldLogger, c character.Model, ai packetmodel.AttackInfo, deps comboOrbDeps)`. Task 9 calls `comboOrbTryUpdate` + `comboOrbProductionDeps`.

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_combo_test.go`:

```go
package handler

import (
	"atlas-channel/character"
	"atlas-channel/character/skill"
	"atlas-channel/data/skill/effect"
	buff2 "atlas-channel/kafka/message/buff"
	"errors"
	"testing"

	skill3 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/sirupsen/logrus"
)

func comboTestSkill(t *testing.T, id skill3.Id, level byte) skill.Model {
	t.Helper()
	m, err := skill.Extract(skill.RestModel{Id: uint32(id), Level: level})
	if err != nil {
		t.Fatalf("skill.Extract: %v", err)
	}
	return m
}

func comboTestCharacter(t *testing.T, skills ...skill.Model) character.Model {
	t.Helper()
	return character.NewModelBuilder().SetId(1).SetSkills(skills).MustBuild()
}

func comboTestAttack(skillId uint32, hits int) packetmodel.AttackInfo {
	ai := packetmodel.NewAttackInfo(packetmodel.AttackTypeMelee)
	ai.SetSkillId(skillId)
	for i := 0; i < hits; i++ {
		ai.AddDamageInfo(*packetmodel.NewDamageInfo(1))
	}
	return *ai
}

func comboTestEffect(t *testing.T, x int16, prop float64) effect.Model {
	t.Helper()
	se, err := effect.Extract(effect.RestModel{X: x, Prop: prop})
	if err != nil {
		t.Fatalf("effect.Extract: %v", err)
	}
	return se
}

type comboEmitRecord struct {
	sourceId  int32
	operation string
	amount    int32
	capValue  int32
}

// comboTestDeps returns deps that record every emit and serve a fixed
// governing effect. roll is fixed so double-orb branches are deterministic.
func comboTestDeps(t *testing.T, se effect.Model, roll float64, emitted *[]comboEmitRecord) comboOrbDeps {
	t.Helper()
	return comboOrbDeps{
		getEffect: func(skillId uint32, level byte) (effect.Model, error) {
			return se, nil
		},
		emitUpdate: func(sourceId int32, operation string, amount int32, capValue int32) error {
			*emitted = append(*emitted, comboEmitRecord{sourceId, operation, amount, capValue})
			return nil
		},
		roll: func() float64 { return roll },
	}
}

func TestComboSkillIds(t *testing.T) {
	t.Run("adventurer line", func(t *testing.T) {
		line, ok := comboSkillIds([]skill.Model{
			comboTestSkill(t, skill3.CrusaderComboAttackId, 20),
			comboTestSkill(t, skill3.HeroAdvancedComboAttackId, 10),
		})
		if !ok {
			t.Fatal("expected ok")
		}
		if line.comboId != skill3.CrusaderComboAttackId || line.comboLevel != 20 {
			t.Fatalf("combo = %d L%d", line.comboId, line.comboLevel)
		}
		if line.advId != skill3.HeroAdvancedComboAttackId || line.advLevel != 10 {
			t.Fatalf("adv = %d L%d", line.advId, line.advLevel)
		}
	})
	t.Run("cygnus line", func(t *testing.T) {
		line, ok := comboSkillIds([]skill.Model{
			comboTestSkill(t, skill3.DawnWarriorStage3ComboAttackId, 15),
		})
		if !ok {
			t.Fatal("expected ok")
		}
		if line.comboId != skill3.DawnWarriorStage3ComboAttackId || line.comboLevel != 15 {
			t.Fatalf("combo = %d L%d", line.comboId, line.comboLevel)
		}
		if line.advId != skill3.DawnWarriorStage3AdvancedComboId || line.advLevel != 0 {
			t.Fatalf("adv = %d L%d, want DawnWarrior adv at 0", line.advId, line.advLevel)
		}
	})
	t.Run("no combo line", func(t *testing.T) {
		if _, ok := comboSkillIds([]skill.Model{comboTestSkill(t, skill3.CrusaderShoutId, 20)}); ok {
			t.Fatal("expected ok=false without Combo Attack")
		}
	})
	t.Run("combo at level 0", func(t *testing.T) {
		if _, ok := comboSkillIds([]skill.Model{comboTestSkill(t, skill3.CrusaderComboAttackId, 0)}); ok {
			t.Fatal("expected ok=false at level 0")
		}
	})
}

func TestIsComboFinisher(t *testing.T) {
	finishers := []skill3.Id{
		skill3.CrusaderPanicSwordId, skill3.CrusaderPanicAxeId,
		skill3.CrusaderComaSwordId, skill3.CrusaderComaAxeId,
		skill3.DawnWarriorStage3PanicId, skill3.DawnWarriorStage3ComaId,
	}
	for _, id := range finishers {
		if !isComboFinisher(id) {
			t.Fatalf("expected %d to be a finisher", id)
		}
	}
	for _, id := range []skill3.Id{skill3.CrusaderComboAttackId, skill3.CrusaderShoutId, skill3.Id(0)} {
		if isComboFinisher(id) {
			t.Fatalf("expected %d NOT to be a finisher", id)
		}
	}
}

func TestComboGainAmount(t *testing.T) {
	cases := []struct {
		name       string
		advLearned bool
		prop       float64
		roll       float64
		want       int32
	}{
		{"no advanced combo", false, 0.60, 0.0, 1},
		{"roll under prop", true, 0.60, 0.59, 2},
		{"roll equal prop", true, 0.60, 0.60, 1},
		{"roll over prop", true, 0.60, 0.61, 1},
		{"prop >= 1 always doubles", true, 1.0, 0.99, 2},
		{"zero prop never doubles", true, 0.0, 0.0, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := comboGainAmount(tc.advLearned, tc.prop, tc.roll); got != tc.want {
				t.Fatalf("comboGainAmount(%v, %v, %v) = %d; want %d", tc.advLearned, tc.prop, tc.roll, got, tc.want)
			}
		})
	}
}

func TestComboOrbTryUpdate(t *testing.T) {
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)

	crusader := comboTestCharacter(t, comboTestSkill(t, skill3.CrusaderComboAttackId, 20))
	hero := comboTestCharacter(t,
		comboTestSkill(t, skill3.CrusaderComboAttackId, 30),
		comboTestSkill(t, skill3.HeroAdvancedComboAttackId, 30),
	)

	t.Run("no combo line emits nothing", func(t *testing.T) {
		var emitted []comboEmitRecord
		c := comboTestCharacter(t, comboTestSkill(t, skill3.CrusaderShoutId, 20))
		comboOrbTryUpdate(l, c, comboTestAttack(0, 1), comboTestDeps(t, comboTestEffect(t, 3, 0), 0.99, &emitted))
		if len(emitted) != 0 {
			t.Fatalf("emitted %d, want 0", len(emitted))
		}
	})

	t.Run("finisher emits SET 1 even with zero hits", func(t *testing.T) {
		var emitted []comboEmitRecord
		comboOrbTryUpdate(l, crusader, comboTestAttack(uint32(skill3.CrusaderPanicSwordId), 0), comboTestDeps(t, comboTestEffect(t, 3, 0), 0.99, &emitted))
		if len(emitted) != 1 {
			t.Fatalf("emitted %d, want 1", len(emitted))
		}
		e := emitted[0]
		if e.operation != buff2.StatOperationSet || e.amount != 1 || e.sourceId != int32(skill3.CrusaderComboAttackId) {
			t.Fatalf("got %+v, want SET 1 on Combo Attack sourceId", e)
		}
	})

	t.Run("cygnus finisher targets cygnus combo sourceId", func(t *testing.T) {
		var emitted []comboEmitRecord
		dw := comboTestCharacter(t, comboTestSkill(t, skill3.DawnWarriorStage3ComboAttackId, 15))
		comboOrbTryUpdate(l, dw, comboTestAttack(uint32(skill3.DawnWarriorStage3PanicId), 1), comboTestDeps(t, comboTestEffect(t, 3, 0), 0.99, &emitted))
		if len(emitted) != 1 || emitted[0].sourceId != int32(skill3.DawnWarriorStage3ComboAttackId) {
			t.Fatalf("got %+v, want SET on DawnWarrior Combo sourceId", emitted)
		}
	})

	t.Run("shout emits nothing", func(t *testing.T) {
		var emitted []comboEmitRecord
		comboOrbTryUpdate(l, crusader, comboTestAttack(uint32(skill3.CrusaderShoutId), 3), comboTestDeps(t, comboTestEffect(t, 3, 0), 0.99, &emitted))
		if len(emitted) != 0 {
			t.Fatalf("emitted %d, want 0", len(emitted))
		}
	})

	t.Run("zero hits emits nothing on non-finisher", func(t *testing.T) {
		var emitted []comboEmitRecord
		comboOrbTryUpdate(l, crusader, comboTestAttack(0, 0), comboTestDeps(t, comboTestEffect(t, 3, 0), 0.99, &emitted))
		if len(emitted) != 0 {
			t.Fatalf("emitted %d, want 0", len(emitted))
		}
	})

	t.Run("normal attack gains one orb with cap x+1", func(t *testing.T) {
		var emitted []comboEmitRecord
		comboOrbTryUpdate(l, crusader, comboTestAttack(0, 1), comboTestDeps(t, comboTestEffect(t, 5, 0), 0.99, &emitted))
		if len(emitted) != 1 {
			t.Fatalf("emitted %d, want 1", len(emitted))
		}
		e := emitted[0]
		if e.operation != buff2.StatOperationIncrement || e.amount != 1 || e.capValue != 6 || e.sourceId != int32(skill3.CrusaderComboAttackId) {
			t.Fatalf("got %+v, want INCREMENT 1 cap 6", e)
		}
	})

	t.Run("advanced combo double orb on successful roll", func(t *testing.T) {
		var emitted []comboEmitRecord
		comboOrbTryUpdate(l, hero, comboTestAttack(0, 1), comboTestDeps(t, comboTestEffect(t, 10, 0.60), 0.10, &emitted))
		if len(emitted) != 1 {
			t.Fatalf("emitted %d, want 1", len(emitted))
		}
		e := emitted[0]
		if e.operation != buff2.StatOperationIncrement || e.amount != 2 || e.capValue != 11 {
			t.Fatalf("got %+v, want INCREMENT 2 cap 11 (adv combo effect governs)", e)
		}
	})

	t.Run("effect lookup error emits nothing", func(t *testing.T) {
		var emitted []comboEmitRecord
		deps := comboTestDeps(t, comboTestEffect(t, 5, 0), 0.99, &emitted)
		deps.getEffect = func(skillId uint32, level byte) (effect.Model, error) {
			return effect.Model{}, errors.New("boom")
		}
		comboOrbTryUpdate(l, crusader, comboTestAttack(0, 1), deps)
		if len(emitted) != 0 {
			t.Fatalf("emitted %d, want 0", len(emitted))
		}
	})

	t.Run("emit error is swallowed", func(t *testing.T) {
		var emitted []comboEmitRecord
		deps := comboTestDeps(t, comboTestEffect(t, 5, 0), 0.99, &emitted)
		deps.emitUpdate = func(sourceId int32, operation string, amount int32, capValue int32) error {
			return errors.New("kafka down")
		}
		// must not panic
		comboOrbTryUpdate(l, crusader, comboTestAttack(0, 1), deps)
	})
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run 'Combo' -v`
Expected: FAIL to compile — `undefined: comboSkillIds`, `undefined: comboOrbDeps`, etc.

- [ ] **Step 3: Implement the combo file**

Create `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_combo.go`:

```go
package handler

import (
	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/character/skill"
	skill2 "atlas-channel/data/skill"
	"atlas-channel/data/skill/effect"
	buff2 "atlas-channel/kafka/message/buff"
	"context"
	"math/rand"

	constants "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill3 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/sirupsen/logrus"
)

// comboLine is the Combo Attack skill line a character owns: the buff source
// (Combo Attack, adventurer or Cygnus variant) and its Advanced Combo
// upgrade. advLevel is 0 when Advanced Combo isn't learned.
type comboLine struct {
	comboId    skill3.Id
	comboLevel byte
	advId      skill3.Id
	advLevel   byte
}

// comboSkillIds resolves the character's combo line from owned skills.
// ok == false when neither Combo Attack variant is owned at level > 0.
func comboSkillIds(skills []skill.Model) (comboLine, bool) {
	find := func(id skill3.Id) byte {
		for _, s := range skills {
			if s.Id() == id {
				return s.Level()
			}
		}
		return 0
	}
	if lvl := find(skill3.CrusaderComboAttackId); lvl > 0 {
		return comboLine{
			comboId:    skill3.CrusaderComboAttackId,
			comboLevel: lvl,
			advId:      skill3.HeroAdvancedComboAttackId,
			advLevel:   find(skill3.HeroAdvancedComboAttackId),
		}, true
	}
	if lvl := find(skill3.DawnWarriorStage3ComboAttackId); lvl > 0 {
		return comboLine{
			comboId:    skill3.DawnWarriorStage3ComboAttackId,
			comboLevel: lvl,
			advId:      skill3.DawnWarriorStage3AdvancedComboId,
			advLevel:   find(skill3.DawnWarriorStage3AdvancedComboId),
		}, true
	}
	return comboLine{}, false
}

// isComboFinisher reports whether the skill consumes combo orbs.
func isComboFinisher(id skill3.Id) bool {
	switch id {
	case skill3.CrusaderPanicSwordId, skill3.CrusaderPanicAxeId,
		skill3.CrusaderComaSwordId, skill3.CrusaderComaAxeId,
		skill3.DawnWarriorStage3PanicId, skill3.DawnWarriorStage3ComaId:
		return true
	}
	return false
}

// comboGainAmount is the number of orbs one qualifying attack gains: 1, or 2
// when Advanced Combo is learned and its double-orb roll succeeds. Mirrors
// mpEaterShouldProc's prop handling (prop >= 1 always procs).
func comboGainAmount(advLearned bool, prop float64, roll float64) int32 {
	if !advLearned || prop <= 0 {
		return 1
	}
	if prop >= 1.0 || roll < prop {
		return 2
	}
	return 1
}

// comboOrbDeps groups the side-effecting lookups comboOrbTryUpdate needs so
// tests can drive every branch without a real processor or Kafka producer.
type comboOrbDeps struct {
	getEffect  func(skillId uint32, level byte) (effect.Model, error)
	emitUpdate func(sourceId int32, operation string, amount int32, capValue int32) error
	roll       func() float64
}

// comboOrbProductionDeps wires comboOrbDeps to the real effect lookup and the
// buff UPDATE_STAT_VALUE emitter for one attack.
func comboOrbProductionDeps(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId uint32) comboOrbDeps {
	bp := buff.NewProcessor(l, ctx)
	return comboOrbDeps{
		getEffect: skill2.NewProcessor(l, ctx).GetEffect,
		emitUpdate: func(sourceId int32, operation string, amount int32, capValue int32) error {
			return bp.UpdateStatValue(f, characterId, sourceId, string(constants.TemporaryStatTypeCombo), operation, amount, capValue)
		},
		roll: rand.Float64,
	}
}

// comboOrbTryUpdate applies Combo Attack orb bookkeeping for one melee
// attack. Finishers consume unconditionally (SET 1 — no hit or orb-count
// requirement, and the attack is never rejected); other attacks gain
// (INCREMENT clamped to the governing effect's x + 1) when at least one
// monster was hit and the skill is not Shout. Whether the COMBO buff is
// actually active is delegated to atlas-buffs, where a missing buff is a
// no-op. All failures are logged and swallowed — the attack pipeline never
// fails on orb bookkeeping.
func comboOrbTryUpdate(l logrus.FieldLogger, c character.Model, ai packetmodel.AttackInfo, deps comboOrbDeps) {
	line, ok := comboSkillIds(c.Skills())
	if !ok {
		return
	}

	sid := skill3.Id(ai.SkillId())
	if isComboFinisher(sid) {
		if err := deps.emitUpdate(int32(line.comboId), buff2.StatOperationSet, 1, 0); err != nil {
			l.WithError(err).Errorf("Combo orbs: consume emit failed for character [%d] finisher [%d].", c.Id(), sid)
		}
		return
	}

	if sid == skill3.CrusaderShoutId || len(ai.DamageInfo()) == 0 {
		return
	}

	effectId, effectLevel := uint32(line.comboId), line.comboLevel
	if line.advLevel > 0 {
		effectId, effectLevel = uint32(line.advId), line.advLevel
	}
	se, err := deps.getEffect(effectId, effectLevel)
	if err != nil {
		l.WithError(err).Errorf("Combo orbs: effect lookup failed for skill [%d] level [%d].", effectId, effectLevel)
		return
	}

	amount := comboGainAmount(line.advLevel > 0, se.Prop(), deps.roll())
	capValue := int32(se.X()) + 1
	if err := deps.emitUpdate(int32(line.comboId), buff2.StatOperationIncrement, amount, capValue); err != nil {
		l.WithError(err).Errorf("Combo orbs: gain emit failed for character [%d].", c.Id())
	}
}
```

Note: `character.Model` has `Id() uint32` (used in the error logs); Shout never reaches the gain path; normal attacks (`SkillId() == 0`) qualify for gain. Cosmic's "pre-roll value ≤ x" second-orb guard is exactly the clamp `min(v + 2, x + 1)` applied registry-side.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run 'Combo' -v`
Expected: PASS (all subtests).

- [ ] **Step 5: Run the whole handler package**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ && go vet ./socket/handler/`
Expected: PASS / clean.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_attack_combo.go services/atlas-channel/atlas.com/channel/socket/handler/character_attack_combo_test.go
git commit -m "feat(channel): combo orb gain/consume decision logic"
```

---

### Task 9: atlas-channel — wire combo orbs into processAttack

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go` (the `// TODO apply combo orbs (add or consume)` line, currently line 404)

**Interfaces:**
- Consumes: Task 8's `comboOrbTryUpdate` + `comboOrbProductionDeps`; in-scope locals `l`, `ctx`, `c` (character with `SkillModelDecorator` already applied at `character_attack_common.go:272`), `ai`, `s`.
- Produces: the live gain/consume hook. Melee only (`CloseRangeDamageHandler` parity) — the shared `processAttack` also serves ranged/magic/energy, which must not gain orbs.

- [ ] **Step 1: Replace the TODO**

In `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go`, replace the line:

```go
					// TODO apply combo orbs (add or consume)
```

with:

```go
					// Combo orb gain/consume: melee only (close-range attacks,
					// Cosmic CloseRangeDamageHandler parity). Fire-and-forget
					// beside the projectile emit — failures never abort the
					// attack. The character was fetched with SkillModelDecorator,
					// so combo skill levels are already in hand.
					if ai.AttackType() == packetmodel.AttackTypeMelee {
						comboOrbTryUpdate(l, c, ai, comboOrbProductionDeps(l, ctx, s.Field(), s.CharacterId()))
					}
```

No import changes are needed — everything referenced is already imported by the file or lives in the same package.

- [ ] **Step 2: Verify build and full handler tests**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./socket/handler/ && go vet ./...`
Expected: clean / PASS.

- [ ] **Step 3: Confirm the TODO is gone**

Run: `grep -n "combo orbs" services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go`
Expected: no `TODO` match (only the new comment, if the grep matches it at all).

- [ ] **Step 4: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go
git commit -m "feat(channel): wire combo orb bookkeeping into melee attack pipeline"
```

---

### Task 10: atlas-channel — announce STAT_UPDATED (owner + foreign re-broadcast)

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/buff/consumer.go`

**Interfaces:**
- Consumes: Task 6's `EventStatusTypeStatUpdated`/`StatUpdatedStatusEventBody`; existing `CharacterBuffGiveWriter`/`CharacterBuffGiveForeignWriter` announce path in `handleStatusEventApplied`.
- Produces: `handleStatusEventStatUpdated` registered on the buff status topic, plus a shared `announceBuffGive` helper reused by the APPLIED handler (extract, don't duplicate). The give writers encode duration as `expiresAt − now` (`libs/atlas-packet/model/character_temporary_stat.go:623`), so passing the original timestamps yields the remaining duration on the client.

Consumer handlers in this repo have no unit tests; verification is compile + vet + the unchanged behavior of the APPLIED path through existing integration usage.

- [ ] **Step 1: Extract the shared announce helper and add the STAT_UPDATED handler**

In `services/atlas-channel/atlas.com/channel/kafka/consumer/buff/consumer.go`:

Add `"time"` to the imports.

Add the helper (before `handleStatusEventApplied`):

```go
// announceBuffGive sends the buff stat set to the owner (GIVE_BUFF) and to
// all other sessions in the owner's map (GIVE_FOREIGN_BUFF). Shared by the
// APPLIED and STAT_UPDATED handlers — the packet layer derives the client
// duration from expiresAt, so callers passing a buff's original timestamps
// broadcast the remaining duration.
func announceBuffGive(l logrus.FieldLogger, ctx context.Context, sc server.Model, wp writer.Producer, characterId uint32, sourceId int32, level byte, duration int32, statChanges []buff2.StatChange, createdAt time.Time, expiresAt time.Time) {
	session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(characterId, func(s session.Model) error {
		bs := make([]buff.Model, 0)
		changes := make([]stat.Model, 0)
		for _, cm := range statChanges {
			changes = append(changes, stat.NewStat(cm.Type, cm.Amount))
		}
		bs = append(bs, buff.NewBuff(sourceId, level, duration, changes, createdAt, expiresAt))

		err := session.Announce(l)(ctx)(wp)(charpkt.CharacterBuffGiveWriter)(writer.CharacterBuffGiveBody(bs))(s)
		if err != nil {
			l.WithError(err).Errorf("Unable to write new character [%d] buffs.", characterId)
		}

		_ = _map.NewProcessor(l, ctx).ForOtherSessionsInMap(s.Field(), s.CharacterId(), func(os session.Model) error {
			err = session.Announce(l)(ctx)(wp)(charpkt.CharacterBuffGiveForeignWriter)(writer.CharacterBuffGiveForeignBody(characterId, bs))(os)
			if err != nil {
				l.WithError(err).Errorf("Unable to write new character [%d] buffs.", characterId)
				return err
			}
			return nil
		})
		return nil
	})
}
```

Rewrite `handleStatusEventApplied` to delegate to it:

```go
func handleStatusEventApplied(sc server.Model, wp writer.Producer) message.Handler[buff2.StatusEvent[buff2.AppliedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e buff2.StatusEvent[buff2.AppliedStatusEventBody]) {
		if e.Type != buff2.EventStatusTypeBuffApplied {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), e.WorldId) {
			return
		}

		announceBuffGive(l, ctx, sc, wp, e.CharacterId, e.Body.SourceId, e.Body.Level, e.Body.Duration, e.Body.Changes, e.Body.CreatedAt, e.Body.ExpiresAt)
	}
}
```

Add the new handler after it:

```go
func handleStatusEventStatUpdated(sc server.Model, wp writer.Producer) message.Handler[buff2.StatusEvent[buff2.StatUpdatedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e buff2.StatusEvent[buff2.StatUpdatedStatusEventBody]) {
		if e.Type != buff2.EventStatusTypeStatUpdated {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), e.WorldId) {
			return
		}

		announceBuffGive(l, ctx, sc, wp, e.CharacterId, e.Body.SourceId, e.Body.Level, e.Body.Duration, e.Body.Changes, e.Body.CreatedAt, e.Body.ExpiresAt)
	}
}
```

Register it in `InitHandlers` after the expired registration (same shape):

```go
					id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventStatUpdated(sc, wp))))
					if err != nil {
						return nil, err
					}
					handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
```

(`handleStatusEventExpired` keeps its own body — it uses the Cancel writers, not Give.)

- [ ] **Step 2: Verify build, vet, full module tests**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go vet ./... && go test -race ./...`
Expected: all clean / PASS.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/consumer/buff/
git commit -m "feat(channel): re-broadcast buffs on STAT_UPDATED with remaining duration"
```

---

### Task 11: Full verification sweep

**Files:** none (verification only).

Consumers of STAT_UPDATED elsewhere: atlas-effective-stats, atlas-mounts, and atlas-rates consume `EVENT_TOPIC_CHARACTER_BUFF_STATUS` but type-guard on APPLIED/EXPIRED, so the new event type is ignored by them with zero changes (design §3). No code changes in those services; the sweep below confirms nothing else regressed.

- [ ] **Step 1: atlas-buffs module checks**

Run: `cd services/atlas-buffs/atlas.com/buffs && go test -race ./... && go vet ./... && go build ./...`
Expected: all PASS / clean.

- [ ] **Step 2: atlas-channel module checks**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./... && go vet ./... && go build ./...`
Expected: all PASS / clean.

- [ ] **Step 3: Docker bake both services**

From the worktree root (`.worktrees/task-142-combo-attack-orbs`):

Run: `docker buildx bake atlas-channel atlas-buffs`
Expected: both images build clean. (Mandatory per CLAUDE.md even though no `go.mod` changed — the PRD acceptance criteria list it explicitly.)

- [ ] **Step 4: Redis key guard**

From the worktree root:

Run: `tools/redis-key-guard.sh`
Expected: clean (no raw keyed go-redis calls added — the registry work goes through `atlas.TenantRegistry`). Do NOT prefix with `GOWORK=off` (known false-FAIL).

- [ ] **Step 5: Commit any stragglers and report**

```bash
git status --short
git branch --show-current   # must be task-142-combo-attack-orbs
```

Expected: clean tree, correct branch. Report results; then run the code-review step (`superpowers:requesting-code-review`) before any PR.

**Manual acceptance (user-assisted, post-deploy):** in-game on a v83 tenant — orb gain on hits, Advanced Combo double proc, Panic/Coma consume, foreign visibility from a second client, and no buff-duration extension on gain (PRD acceptance criteria). This requires a deployed environment and two clients; it is not automatable from this plan.
