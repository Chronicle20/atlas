# Monster Aggro & Controller Switching — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add per-monster damage attribution with `lastHitMs`, switch the monster controller to the current DPS leader on damage, broadcast a two-state `controllerHasAggro` flag end-to-end, and decay non-boss aggro on a 1500ms background sweep.

**Architecture:** atlas-monsters owns the per-monster damage table in Redis, applied via Lua scripts for atomicity. The damage script returns a `{wasFirstHit, monster}` envelope so Go can decide whether to emit `AGGRO_CHANGED`. A new `MonsterAggroDecayTask` does a hybrid Go-prefilter / Lua-write decay sweep, emitting `STOP_CONTROL` when a non-boss's table fully clears. atlas-channel forwards the `controllerHasAggro` flag into `StartControlMonsterBody` and adds an `AGGRO_CHANGED` consumer that re-sends `MonsterControlWriter` to the controller's session.

**Tech Stack:** Go, Redis (go-redis + Lua), Kafka (segmentio/kafka-go), miniredis for tests, logrus, immutable Model + Builder pattern, JSON:API only on the REST surface (not relevant for this task).

**Companions:** `prd.md`, `design.md`, `data-model.md`, `context.md`. Read `context.md` first for the file/symbol cheat-sheet.

---

## Service paths

- atlas-monsters: `services/atlas-monsters/atlas.com/monsters/`
- atlas-channel: `services/atlas-channel/atlas.com/channel/`

All file paths below are relative to the repository root.

---

## Task 1 — atlas-monsters: storedDamageEntry/entry adds LastHitMs + legacy migration

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/registry.go` (storedDamageEntry, toStored, fromStored)
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/model.go` (entry, NewMonster default)
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/builder.go` (Clone copies new field; AddDamageEntry takes nowMs)
- Test: `services/atlas-monsters/atlas.com/monsters/monster/registry_test.go`

- [ ] **Step 1: Write the failing test for legacy migration**

Append this test to `services/atlas-monsters/atlas.com/monsters/monster/registry_test.go`:

```go
// TestFromStoredCollapsesLegacyDamageEntries verifies that a Redis blob with
// the old multi-row-per-character shape and no lastHitMs round-trips into a
// single aggregated entry per character with LastHitMs == 0.
func TestFromStoredCollapsesLegacyDamageEntries(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := testContext(ten)
	r.Clear(ctx)

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 100, 50)

	legacy := `{"uniqueId":` + strconv.FormatUint(uint64(m.UniqueId()), 10) +
		`,"tenantId":"` + ten.Id().String() + `","tenantRegion":"GMS"` +
		`,"tenantMajorVersion":83,"tenantMinorVersion":1` +
		`,"worldId":0,"channelId":0,"mapId":40000` +
		`,"instance":"00000000-0000-0000-0000-000000000000"` +
		`,"maxHp":100,"hp":100,"maxMp":50,"mp":50` +
		`,"monsterId":9300018,"controlCharacterId":0` +
		`,"x":0,"y":0,"fh":0,"stance":5,"team":0` +
		`,"damageEntries":[` +
		`{"characterId":7,"damage":10},` +
		`{"characterId":7,"damage":15},` +
		`{"characterId":9,"damage":5}` +
		`],"statusEffects":[]}`
	testMiniRedis.Set(monsterKey(ten, m.UniqueId()), legacy)

	got, err := r.GetMonster(ten, m.UniqueId())
	if err != nil {
		t.Fatalf("GetMonster failed: %v", err)
	}
	entries := got.DamageEntries()
	if len(entries) != 2 {
		t.Fatalf("expected 2 aggregated entries, got %d (%+v)", len(entries), entries)
	}
	for _, e := range entries {
		if e.CharacterId == 7 && e.Damage != 25 {
			t.Errorf("character 7 damage: expected 25, got %d", e.Damage)
		}
		if e.CharacterId == 9 && e.Damage != 5 {
			t.Errorf("character 9 damage: expected 5, got %d", e.Damage)
		}
		if e.LastHitMs != 0 {
			t.Errorf("legacy entry should default LastHitMs=0, got %d", e.LastHitMs)
		}
	}
}
```

- [ ] **Step 2: Run the test, see it fail**

```
cd services/atlas-monsters/atlas.com/monsters
go test ./monster -run TestFromStoredCollapsesLegacyDamageEntries -v
```
Expected: FAIL — current `entry` has no `LastHitMs`, and `fromStored` does not collapse rows.

- [ ] **Step 3: Add LastHitMs to types and collapse logic**

In `monster/model.go`, change `entry`:

```go
type entry struct {
	CharacterId uint32
	Damage      uint32
	LastHitMs   int64
}
```

In `monster/registry.go`, change `storedDamageEntry`:

```go
type storedDamageEntry struct {
	CharacterId uint32 `json:"characterId"`
	Damage      uint32 `json:"damage"`
	LastHitMs   int64  `json:"lastHitMs"`
}
```

In `monster/registry.go` `toStored`, copy `LastHitMs`:

```go
des := make([]storedDamageEntry, 0, len(m.damageEntries))
for _, e := range m.damageEntries {
	des = append(des, storedDamageEntry{
		CharacterId: e.CharacterId,
		Damage:      e.Damage,
		LastHitMs:   e.LastHitMs,
	})
}
```

In `monster/registry.go` `fromStored`, collapse and migrate:

```go
agg := make(map[uint32]*entry)
order := make([]uint32, 0, len(sm.DamageEntries))
for _, de := range sm.DamageEntries {
	if existing, ok := agg[de.CharacterId]; ok {
		existing.Damage += de.Damage
		// Take the latest non-zero lastHitMs; legacy rows have 0.
		if de.LastHitMs > existing.LastHitMs {
			existing.LastHitMs = de.LastHitMs
		}
		continue
	}
	agg[de.CharacterId] = &entry{
		CharacterId: de.CharacterId,
		Damage:      de.Damage,
		LastHitMs:   de.LastHitMs,
	}
	order = append(order, de.CharacterId)
}
des := make([]entry, 0, len(order))
for _, cid := range order {
	des = append(des, *agg[cid])
}
```

In `monster/builder.go` `AddDamageEntry` — keep the same signature for now (callers will be updated in Task 6 if needed). The `Damage` Go method on Model uses this, but the production hot path is the Lua script. For now, only ensure Clone copies the slice unchanged:

```go
// Clone(m) already does damageEntries: m.damageEntries — leave as is.
// AddDamageEntry stays unchanged for this task; nowMs threading happens in T6.
```

- [ ] **Step 4: Run the test, see it pass**

```
go test ./monster -run TestFromStoredCollapsesLegacyDamageEntries -v
```
Expected: PASS.

- [ ] **Step 5: Run the full registry test suite to confirm nothing else broke**

```
go test ./monster -run 'Test' -count=1
```
Expected: PASS for all existing tests except those that count multi-line damage entries. `TestCreateMoveDamageKill` expects 2 entries (different characters) — passes. `TestConcurrentDamage` asserts `len == numAttackers*hitsPerAttacker` (50*10=500). With aggregation this becomes `numAttackers` (10). **Update `TestConcurrentDamage` to expect `numAttackers` entries instead of `numAttackers*hitsPerAttacker`** before running:

In `services/atlas-monsters/atlas.com/monsters/monster/registry_test.go`, replace:

```go
if len(got.DamageEntries()) != numAttackers*hitsPerAttacker {
	t.Fatalf("Expected %d damage entries, got %d", numAttackers*hitsPerAttacker, len(got.DamageEntries()))
}
```

with:

```go
if len(got.DamageEntries()) != numAttackers {
	t.Fatalf("Expected %d aggregated damage entries, got %d", numAttackers, len(got.DamageEntries()))
}
```

Run again:

```
go test ./monster -count=1
```
Expected: PASS.

- [ ] **Step 6: Commit**

```
git add services/atlas-monsters/atlas.com/monsters/monster/registry.go \
        services/atlas-monsters/atlas.com/monsters/monster/model.go \
        services/atlas-monsters/atlas.com/monsters/monster/registry_test.go
git commit -m "feat(atlas-monsters): aggregate damage entries with lastHitMs and migrate legacy state"
```

---

## Task 2 — atlas-monsters: storedMonster/Model gain controllerHasAggro

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/registry.go` (storedMonster, toStored, fromStored)
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/model.go` (Model field + getter, NewMonster default false)
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/builder.go` (Clone, ModelBuilder, SetControllerHasAggro, Build)
- Test: `services/atlas-monsters/atlas.com/monsters/monster/registry_test.go`

- [ ] **Step 1: Write the failing test**

Append to `monster/registry_test.go`:

```go
// TestControllerHasAggroRoundTrip verifies that storedMonster.ControllerHasAggro
// round-trips through Redis and that legacy blobs missing the field default to
// false.
func TestControllerHasAggroRoundTrip(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := testContext(ten)
	r.Clear(ctx)

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 100, 50)
	if m.ControllerHasAggro() {
		t.Fatal("freshly spawned monster should default ControllerHasAggro=false")
	}

	updated := Clone(m).SetControllerHasAggro(true).Build()
	r.UpdateMonster(ten, m.UniqueId(), updated)
	got, err := r.GetMonster(ten, m.UniqueId())
	if err != nil {
		t.Fatalf("GetMonster failed: %v", err)
	}
	if !got.ControllerHasAggro() {
		t.Fatal("expected ControllerHasAggro=true after persisted update")
	}

	// Legacy blob missing the field
	legacy := `{"uniqueId":` + strconv.FormatUint(uint64(m.UniqueId()), 10) +
		`,"tenantId":"` + ten.Id().String() + `","tenantRegion":"GMS"` +
		`,"tenantMajorVersion":83,"tenantMinorVersion":1` +
		`,"worldId":0,"channelId":0,"mapId":40000` +
		`,"instance":"00000000-0000-0000-0000-000000000000"` +
		`,"maxHp":100,"hp":100,"maxMp":50,"mp":50` +
		`,"monsterId":9300018,"controlCharacterId":0` +
		`,"x":0,"y":0,"fh":0,"stance":5,"team":0` +
		`,"damageEntries":[],"statusEffects":[]}`
	testMiniRedis.Set(monsterKey(ten, m.UniqueId()), legacy)
	got, err = r.GetMonster(ten, m.UniqueId())
	if err != nil {
		t.Fatalf("GetMonster legacy failed: %v", err)
	}
	if got.ControllerHasAggro() {
		t.Fatal("legacy missing field must default to false")
	}
}
```

- [ ] **Step 2: Run, see fail**

```
go test ./monster -run TestControllerHasAggroRoundTrip -v
```
Expected: FAIL — `Model.ControllerHasAggro`, `SetControllerHasAggro` undefined.

- [ ] **Step 3: Implement field, getter, builder, persistence**

In `monster/model.go`, add field and getter:

```go
type Model struct {
	// ... existing fields ...
	statusEffects      []StatusEffect
	controllerHasAggro bool
}

func (m Model) ControllerHasAggro() bool {
	return m.controllerHasAggro
}
```

`NewMonster` default is the zero value (`false`); no change needed unless an explicit init line is preferred — leave implicit.

In `monster/builder.go`, extend `Clone`, `ModelBuilder`, add setter, and `Build`:

```go
// Clone:
return &ModelBuilder{
	// ... existing copies ...
	statusEffects:      effects,
	controllerHasAggro: m.controllerHasAggro,
}

// ModelBuilder:
type ModelBuilder struct {
	// ... existing ...
	statusEffects      []StatusEffect
	controllerHasAggro bool
}

// New setter:
func (b *ModelBuilder) SetControllerHasAggro(v bool) *ModelBuilder {
	b.controllerHasAggro = v
	return b
}

// Build:
return Model{
	// ... existing ...
	statusEffects:      b.statusEffects,
	controllerHasAggro: b.controllerHasAggro,
}
```

In `monster/registry.go`, add field and threading:

```go
type storedMonster struct {
	// ... existing ...
	StatusEffects      statusEffectList `json:"statusEffects"`
	ControllerHasAggro bool             `json:"controllerHasAggro"`
}

// toStored (append):
return storedMonster{
	// ... existing ...
	StatusEffects:      ses,
	ControllerHasAggro: m.controllerHasAggro,
}

// fromStored (append in Model literal):
return t, Model{
	// ... existing ...
	statusEffects:      ses,
	controllerHasAggro: sm.ControllerHasAggro,
}, nil
```

- [ ] **Step 4: Run, see pass**

```
go test ./monster -run TestControllerHasAggroRoundTrip -v
```
Expected: PASS.

- [ ] **Step 5: Run the full suite**

```
go test ./monster -count=1
```
Expected: PASS.

- [ ] **Step 6: Commit**

```
git add services/atlas-monsters/atlas.com/monsters/monster/registry.go \
        services/atlas-monsters/atlas.com/monsters/monster/model.go \
        services/atlas-monsters/atlas.com/monsters/monster/builder.go \
        services/atlas-monsters/atlas.com/monsters/monster/registry_test.go
git commit -m "feat(atlas-monsters): add controllerHasAggro to monster Model and stored representation"
```

---

## Task 3 — atlas-monsters: aggro.go constants + IsAggroIdle

**Files:**
- Create: `services/atlas-monsters/atlas.com/monsters/monster/aggro.go`
- Test: `services/atlas-monsters/atlas.com/monsters/monster/aggro_test.go` (new)

- [ ] **Step 1: Write the failing test**

Create `monster/aggro_test.go`:

```go
package monster

import "testing"

func TestIsAggroIdleBoundary(t *testing.T) {
	now := int64(20_000)

	// Exactly at threshold: not yet idle.
	e := entry{LastHitMs: now - AggroIdleThresholdMs}
	if IsAggroIdle(e, now) {
		t.Errorf("entry at exactly threshold (delta=%d) should NOT be idle", AggroIdleThresholdMs)
	}

	// Past threshold by 1 ms: idle.
	e = entry{LastHitMs: now - AggroIdleThresholdMs - 1}
	if !IsAggroIdle(e, now) {
		t.Errorf("entry past threshold by 1ms should be idle")
	}

	// Just-hit: not idle.
	e = entry{LastHitMs: now}
	if IsAggroIdle(e, now) {
		t.Errorf("just-hit entry should not be idle")
	}
}

func TestAggroConstants(t *testing.T) {
	if AggroIdleThresholdMs != 10_000 {
		t.Errorf("AggroIdleThresholdMs: expected 10000, got %d", AggroIdleThresholdMs)
	}
	if AggroDecayMultiplier != 0.85 {
		t.Errorf("AggroDecayMultiplier: expected 0.85, got %v", AggroDecayMultiplier)
	}
	if AggroDecayFloor != 1 {
		t.Errorf("AggroDecayFloor: expected 1, got %d", AggroDecayFloor)
	}
	if AggroSweepInterval.Milliseconds() != 1500 {
		t.Errorf("AggroSweepInterval: expected 1500ms, got %v", AggroSweepInterval)
	}
}
```

- [ ] **Step 2: Run, see fail**

```
go test ./monster -run TestIsAggroIdleBoundary -v
```
Expected: FAIL — file doesn't exist.

- [ ] **Step 3: Create aggro.go**

`services/atlas-monsters/atlas.com/monsters/monster/aggro.go`:

```go
package monster

import "time"

// Aggro decay constants. Mirror Cosmic's MonsterAggroCoordinator
// (handlers/MonsterAggroCoordinator.java:110-148) so behavior matches reference.
const (
	// AggroIdleThresholdMs is the duration in milliseconds an entry can sit without
	// a fresh hit before the decay sweep begins reducing it.
	AggroIdleThresholdMs = int64(10_000)

	// AggroDecayMultiplier is applied to a damage entry's accumulated damage on
	// each sweep tick once the entry is idle (15% reduction per 1.5s tick).
	AggroDecayMultiplier = 0.85

	// AggroDecayFloor is the minimum damage value an entry can hold; once a
	// decayed value falls below this floor the entry is pruned.
	AggroDecayFloor = uint32(1)
)

// AggroSweepInterval is the cadence at which MonsterAggroDecayTask runs.
const AggroSweepInterval = 1500 * time.Millisecond

// IsAggroIdle reports whether the entry's last hit is older than the idle
// threshold.
func IsAggroIdle(e entry, nowMs int64) bool {
	return nowMs-e.LastHitMs > AggroIdleThresholdMs
}
```

- [ ] **Step 4: Run, see pass**

```
go test ./monster -run 'TestIsAggroIdleBoundary|TestAggroConstants' -v
```
Expected: PASS.

- [ ] **Step 5: Commit**

```
git add services/atlas-monsters/atlas.com/monsters/monster/aggro.go \
        services/atlas-monsters/atlas.com/monsters/monster/aggro_test.go
git commit -m "feat(atlas-monsters): add aggro decay constants and IsAggroIdle helper"
```

---

## Task 4 — atlas-monsters: rewrite applyDamageScript with envelope, threading nowMs

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/registry.go` (applyDamageScript, ApplyDamage signature, DamageSummary)
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/model.go` (DamageSummary adds WasFirstHit)
- Test: `services/atlas-monsters/atlas.com/monsters/monster/registry_test.go`

This task changes the public signature of `Registry.ApplyDamage`. The existing call sites compile-break until Task 6 updates them. To keep the build green during this commit, do the call-site updates in Task 6 BEFORE merging — but for TDD, write tests against the new signature here and let intermediate test runs scope to `./monster` only.

Actually, to keep `go test ./...` green between commits, we'll do the signature change *and* the call-site updates together in this task. Task 6 then becomes redundant; we keep it as a verification step.

**Combine Task 4 + Task 6 into this commit.**

Existing callers of `ApplyDamage`:
- `monster/processor.go:274` (in `Damage`)
- `monster/processor.go:373` (in `DamageFriendly`)
- `monster/status_task.go:86` (in `processDoTTick`)
- `monster/processor_test.go:202` (`r.ApplyDamage(ten, 1, 999, uniqueId)`)
- `monster/registry_test.go` various.

- [ ] **Step 1: Write failing test for wasFirstHit and aggregation**

Append to `monster/registry_test.go`:

```go
// TestApplyDamageWasFirstHit verifies the WasFirstHit flag in DamageSummary.
// First hit on a monster that has a controller flips controllerHasAggro from
// false to true and reports WasFirstHit=true. Subsequent hits report false.
// A monster with no controller never reports WasFirstHit=true.
func TestApplyDamageWasFirstHit(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := testContext(ten)
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()

	// (a) No controller -> WasFirstHit is always false.
	m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 1000, 50)
	now := int64(1_000_000)
	s, err := r.ApplyDamage(ten, 1, 10, m.UniqueId(), now)
	if err != nil {
		t.Fatalf("ApplyDamage: %v", err)
	}
	if s.WasFirstHit {
		t.Errorf("WasFirstHit should be false when no controller is set")
	}

	// (b) With controller -> first hit flips aggro and reports true.
	m2 := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 1000, 50)
	if _, err := r.ControlMonster(ten, m2.UniqueId(), 42); err != nil {
		t.Fatalf("ControlMonster: %v", err)
	}
	s, err = r.ApplyDamage(ten, 7, 10, m2.UniqueId(), now)
	if err != nil {
		t.Fatalf("ApplyDamage: %v", err)
	}
	if !s.WasFirstHit {
		t.Errorf("first hit on controlled monster should report WasFirstHit=true")
	}
	if !s.Monster.ControllerHasAggro() {
		t.Errorf("ControllerHasAggro must flip true after first hit")
	}

	// (c) Second hit on same monster reports false even from a different attacker.
	s, err = r.ApplyDamage(ten, 8, 5, m2.UniqueId(), now+1)
	if err != nil {
		t.Fatalf("ApplyDamage 2: %v", err)
	}
	if s.WasFirstHit {
		t.Errorf("subsequent hits must report WasFirstHit=false")
	}
}

// TestApplyDamageAggregatesByCharacterId verifies that two hits from the same
// character produce a single aggregated entry with summed damage and the most
// recent lastHitMs.
func TestApplyDamageAggregatesByCharacterId(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := testContext(ten)
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 1000, 50)

	if _, err := r.ApplyDamage(ten, 1, 10, m.UniqueId(), 100); err != nil {
		t.Fatalf("ApplyDamage 1: %v", err)
	}
	if _, err := r.ApplyDamage(ten, 1, 25, m.UniqueId(), 200); err != nil {
		t.Fatalf("ApplyDamage 2: %v", err)
	}
	got, err := r.GetMonster(ten, m.UniqueId())
	if err != nil {
		t.Fatalf("GetMonster: %v", err)
	}
	entries := got.DamageEntries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 aggregated entry, got %d (%+v)", len(entries), entries)
	}
	if entries[0].CharacterId != 1 || entries[0].Damage != 35 {
		t.Errorf("entry: expected charId=1 damage=35, got %+v", entries[0])
	}
	if entries[0].LastHitMs != 200 {
		t.Errorf("expected LastHitMs=200, got %d", entries[0].LastHitMs)
	}
}
```

- [ ] **Step 2: Run, see fail**

```
go test ./monster -run 'TestApplyDamageWasFirstHit|TestApplyDamageAggregatesByCharacterId' -v
```
Expected: FAIL — `ApplyDamage` signature wrong (4 args vs 5), `WasFirstHit` undefined.

- [ ] **Step 3: Update DamageSummary**

In `monster/model.go`:

```go
type DamageSummary struct {
	CharacterId   uint32
	Monster       Model
	VisibleDamage uint32
	ActualDamage  int64
	Killed        bool
	WasFirstHit   bool
}
```

- [ ] **Step 4: Rewrite applyDamageScript and Registry.ApplyDamage**

In `monster/registry.go`, replace the existing `applyDamageScript`:

```go
var applyDamageScript = goredis.NewScript(`
local key = KEYS[1]
local charId = tonumber(ARGV[1])
local damage = tonumber(ARGV[2])
local nowMs = tonumber(ARGV[3])
local j = redis.call('GET', key)
if not j then
    return redis.error_reply("monster not found")
end
local m = cjson.decode(j)
local hp = m.hp
local actual = hp - math.max(hp - damage, 0)
m.hp = hp - actual

local entries = m.damageEntries
if type(entries) ~= 'table' then
    entries = {}
end

local found = false
for _, e in ipairs(entries) do
    if e.characterId == charId then
        e.damage = e.damage + actual
        e.lastHitMs = nowMs
        found = true
        break
    end
end
if not found then
    table.insert(entries, {
        characterId = charId,
        damage = actual,
        lastHitMs = nowMs
    })
end
m.damageEntries = entries

local hadAggro = m.controllerHasAggro
local wasFirstHit = false
if m.controlCharacterId ~= 0 and not hadAggro then
    m.controllerHasAggro = true
    wasFirstHit = true
end

redis.call('SET', key, cjson.encode(m))
return cjson.encode({wasFirstHit = wasFirstHit, monster = m})
`)
```

Replace `Registry.ApplyDamage`:

```go
func (r *Registry) ApplyDamage(t tenant.Model, characterId uint32, damage uint32, uniqueId uint32, nowMs int64) (DamageSummary, error) {
	ctx := context.Background()
	key := monsterKey(t, uniqueId)

	result, err := applyDamageScript.Run(ctx, r.client, []string{key},
		strconv.FormatUint(uint64(characterId), 10),
		strconv.FormatUint(uint64(damage), 10),
		strconv.FormatInt(nowMs, 10),
	).Result()
	if err != nil {
		return DamageSummary{}, errors.New("monster not found")
	}

	resultStr, ok := result.(string)
	if !ok {
		return DamageSummary{}, errors.New("unexpected response type")
	}

	var env struct {
		WasFirstHit bool          `json:"wasFirstHit"`
		Monster     storedMonster `json:"monster"`
	}
	if err := json.Unmarshal([]byte(resultStr), &env); err != nil {
		return DamageSummary{}, err
	}
	_, m, err := fromStored(env.Monster)
	if err != nil {
		return DamageSummary{}, err
	}

	return DamageSummary{
		CharacterId:   characterId,
		Monster:       m,
		VisibleDamage: damage,
		Killed:        m.Hp() == 0,
		WasFirstHit:   env.WasFirstHit,
	}, nil
}
```

(Note: the existing `ActualDamage: int64(m.Hp() - m.Hp())` line is a no-op `0` and was clearly a bug. Drop it; default zero-value is the same. We don't add a behavior change here.)

- [ ] **Step 5: Update all call sites of ApplyDamage**

`monster/processor.go:244-337` (Damage method): replace the `ApplyDamage` call inside the loop:

```go
nowMs := time.Now().UnixMilli()
// ... existing loop header ...
for _, d := range damages {
	s, err := GetMonsterRegistry().ApplyDamage(p.t, characterId, d, m.UniqueId(), nowMs)
	if err != nil {
		// existing error log + break
```

`monster/processor.go` `DamageFriendly` (line ~373): pass `time.Now().UnixMilli()` as the new arg:

```go
s, err := GetMonsterRegistry().ApplyDamage(p.t, attackerUniqueId, damage, uniqueId, time.Now().UnixMilli())
```

`monster/status_task.go` `processDoTTick` (line ~86):

```go
ds, err := GetMonsterRegistry().ApplyDamage(ten, se.SourceCharacterId(), totalDamage, m.UniqueId(), time.Now().UnixMilli())
```

`monster/processor_test.go` line 202:

```go
r.ApplyDamage(ten, 1, 999, uniqueId, time.Now().UnixMilli())
```

(Add `"time"` to the test file's imports.)

`monster/registry_test.go` — search for all `r.ApplyDamage(` calls and append a `nowMs` arg. The existing `TestCreateMoveDamageKill` and `TestConcurrentDamage` and `TestLoadMonsterWithCjsonEmptyObjectArrays` calls will all need a fifth arg. Use `time.Now().UnixMilli()` (add the `"time"` import — it's already there).

For example:
```go
ds, err := r.ApplyDamage(ten, 1, 30, m.UniqueId(), time.Now().UnixMilli())
```

- [ ] **Step 6: Run focused tests**

```
go test ./monster -run 'TestApplyDamageWasFirstHit|TestApplyDamageAggregatesByCharacterId' -v
```
Expected: PASS.

- [ ] **Step 7: Run full suite**

```
go test ./... -count=1
```
Expected: PASS for atlas-monsters. Fix any remaining call-site compile errors found.

- [ ] **Step 8: Commit**

```
git add services/atlas-monsters/atlas.com/monsters/monster/registry.go \
        services/atlas-monsters/atlas.com/monsters/monster/model.go \
        services/atlas-monsters/atlas.com/monsters/monster/processor.go \
        services/atlas-monsters/atlas.com/monsters/monster/status_task.go \
        services/atlas-monsters/atlas.com/monsters/monster/processor_test.go \
        services/atlas-monsters/atlas.com/monsters/monster/registry_test.go
git commit -m "feat(atlas-monsters): rewrite applyDamageScript with envelope and thread nowMs through ApplyDamage"
```

---

## Task 5 — atlas-monsters: simplify DamageSummary/DamageEntries/DamageLeader on aggregated entries

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/model.go` (DamageSummary method becomes passthrough)
- Test: `services/atlas-monsters/atlas.com/monsters/monster/model_test.go` (new file)

- [ ] **Step 1: Write failing test**

Create `services/atlas-monsters/atlas.com/monsters/monster/model_test.go`:

```go
package monster

import (
	"sort"
	"testing"
)

func makeModelWithEntries(entries []entry) Model {
	return Model{damageEntries: entries}
}

func TestDamageSummaryPassthrough(t *testing.T) {
	src := []entry{
		{CharacterId: 1, Damage: 100, LastHitMs: 10},
		{CharacterId: 2, Damage: 200, LastHitMs: 20},
	}
	m := makeModelWithEntries(src)
	got := m.DamageSummary()
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(got))
	}
	sort.Slice(got, func(i, j int) bool { return got[i].CharacterId < got[j].CharacterId })
	if got[0].CharacterId != 1 || got[0].Damage != 100 {
		t.Errorf("got[0]: %+v", got[0])
	}
	if got[1].CharacterId != 2 || got[1].Damage != 200 {
		t.Errorf("got[1]: %+v", got[1])
	}
}

func TestDamageLeaderOverAggregatedEntries(t *testing.T) {
	m := makeModelWithEntries([]entry{
		{CharacterId: 1, Damage: 50, LastHitMs: 1},
		{CharacterId: 2, Damage: 200, LastHitMs: 2},
		{CharacterId: 3, Damage: 150, LastHitMs: 3},
	})
	leader := m.DamageLeader()
	if leader != 2 {
		t.Fatalf("expected leader=2, got %d", leader)
	}
}
```

- [ ] **Step 2: Run, see fail**

```
go test ./monster -run 'TestDamageSummaryPassthrough|TestDamageLeaderOverAggregatedEntries' -v
```
Expected: existing `DamageSummary` collapses via map (still passes — but we want to make it a passthrough explicitly). `TestDamageLeaderOverAggregatedEntries` should already PASS because `DamageLeader` doesn't depend on uniqueness. Confirm both pass — if `DamageSummary` already passes too, **proceed and replace the impl with a passthrough anyway** to satisfy the design's "DamageSummary becomes a passthrough" requirement. The test will continue to pass.

- [ ] **Step 3: Replace DamageSummary**

In `monster/model.go`, replace the `DamageSummary()` method:

```go
// DamageSummary returns the per-character damage entries. Entries are now
// pre-aggregated by characterId at write time (Task 1+4), so this is a
// straight passthrough of m.damageEntries.
func (m Model) DamageSummary() []entry {
	return m.damageEntries
}
```

Remove the now-unused `math` import if it becomes unused (`m.Damage` still uses it).

- [ ] **Step 4: Run**

```
go test ./monster -count=1
```
Expected: PASS.

- [ ] **Step 5: Commit**

```
git add services/atlas-monsters/atlas.com/monsters/monster/model.go \
        services/atlas-monsters/atlas.com/monsters/monster/model_test.go
git commit -m "refactor(atlas-monsters): DamageSummary becomes passthrough since entries are pre-aggregated"
```

---

## Task 6 — atlas-monsters: verify call sites pass nowMs (no changes expected; verification only)

This task is folded into Task 4. Run the verification step to confirm the build is green.

- [ ] **Step 1: Verify**

```
cd services/atlas-monsters/atlas.com/monsters
go build ./...
go test ./...
```
Expected: PASS.

(No commit — nothing changes here. If something fails, return to Task 4 and fix.)

---

## Task 7 — atlas-monsters: decayDamageEntriesScript + Registry.DecayDamageEntries

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/registry.go` (new script + method + DecaySummary type)
- Test: `services/atlas-monsters/atlas.com/monsters/monster/registry_test.go`

- [ ] **Step 1: Write failing tests**

Append to `monster/registry_test.go`:

```go
// TestDecayDamageEntriesIdleEntriesDecay verifies that an entry idle past the
// threshold is decayed by AggroDecayMultiplier on each call, and that an entry
// not idle is unchanged.
func TestDecayDamageEntriesIdleEntriesDecay(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := testContext(ten)
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 1000, 50)

	// Two entries: one idle (lastHitMs=0), one fresh (lastHitMs=now).
	now := int64(20_000)
	if _, err := r.ApplyDamage(ten, 1, 100, m.UniqueId(), 0); err != nil {
		t.Fatalf("ApplyDamage 1: %v", err)
	}
	if _, err := r.ApplyDamage(ten, 2, 50, m.UniqueId(), now); err != nil {
		t.Fatalf("ApplyDamage 2: %v", err)
	}

	summary, err := r.DecayDamageEntries(ten, m.UniqueId(), now)
	if err != nil {
		t.Fatalf("DecayDamageEntries: %v", err)
	}
	if summary.ControllerCleared {
		t.Error("controllerCleared should be false (no controller was set)")
	}
	entries := summary.Monster.DamageEntries()
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	for _, e := range entries {
		if e.CharacterId == 1 && e.Damage != 85 {
			t.Errorf("idle entry should decay 100 -> 85, got %d", e.Damage)
		}
		if e.CharacterId == 2 && e.Damage != 50 {
			t.Errorf("fresh entry should remain 50, got %d", e.Damage)
		}
	}
}

// TestDecayDamageEntriesPrunesBelowFloor verifies the deterministic decay
// sequence and that entries below AggroDecayFloor are pruned.
func TestDecayDamageEntriesPrunesBelowFloor(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := testContext(ten)
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 1000, 50)

	if _, err := r.ApplyDamage(ten, 1, 100, m.UniqueId(), 0); err != nil {
		t.Fatalf("ApplyDamage: %v", err)
	}

	expected := []uint32{85, 72, 61, 51, 43, 36, 30, 25, 21, 17, 14, 11, 9, 7, 5, 4, 3, 2, 1}
	now := int64(20_000)
	for i, want := range expected {
		summary, err := r.DecayDamageEntries(ten, m.UniqueId(), now)
		if err != nil {
			t.Fatalf("Decay %d: %v", i, err)
		}
		entries := summary.Monster.DamageEntries()
		if len(entries) != 1 {
			t.Fatalf("Decay %d: expected 1 entry, got %d", i, len(entries))
		}
		if entries[0].Damage != want {
			t.Errorf("Decay %d: damage want=%d got=%d", i, want, entries[0].Damage)
		}
	}
	// Next decay drops below floor and prunes.
	summary, err := r.DecayDamageEntries(ten, m.UniqueId(), now)
	if err != nil {
		t.Fatalf("Decay final: %v", err)
	}
	if len(summary.Monster.DamageEntries()) != 0 {
		t.Fatalf("expected pruned (0 entries), got %d", len(summary.Monster.DamageEntries()))
	}
}

// TestDecayDamageEntriesClearsController verifies the FR-19 path: when all
// entries prune and a controller exists, ControllerCleared=true and
// PrevControllerId is set.
func TestDecayDamageEntriesClearsController(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := testContext(ten)
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 1000, 50)

	if _, err := r.ControlMonster(ten, m.UniqueId(), 42); err != nil {
		t.Fatalf("ControlMonster: %v", err)
	}
	// Single tiny entry that will be floored on first decay.
	if _, err := r.ApplyDamage(ten, 1, 1, m.UniqueId(), 0); err != nil {
		t.Fatalf("ApplyDamage: %v", err)
	}
	now := int64(20_000)
	summary, err := r.DecayDamageEntries(ten, m.UniqueId(), now)
	if err != nil {
		t.Fatalf("DecayDamageEntries: %v", err)
	}
	if !summary.ControllerCleared {
		t.Fatal("expected ControllerCleared=true")
	}
	if summary.PrevControllerId != 42 {
		t.Errorf("expected PrevControllerId=42, got %d", summary.PrevControllerId)
	}
	if summary.Monster.ControlCharacterId() != 0 {
		t.Errorf("expected post-state controller=0, got %d", summary.Monster.ControlCharacterId())
	}
	if summary.Monster.ControllerHasAggro() {
		t.Error("expected post-state controllerHasAggro=false")
	}
}

// TestDecayDamageEntriesNoOpWhenAllFresh verifies a no-op when nothing is idle.
func TestDecayDamageEntriesNoOpWhenAllFresh(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := testContext(ten)
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 1000, 50)

	now := int64(20_000)
	if _, err := r.ApplyDamage(ten, 1, 100, m.UniqueId(), now); err != nil {
		t.Fatalf("ApplyDamage: %v", err)
	}
	summary, err := r.DecayDamageEntries(ten, m.UniqueId(), now)
	if err != nil {
		t.Fatalf("DecayDamageEntries: %v", err)
	}
	entries := summary.Monster.DamageEntries()
	if len(entries) != 1 || entries[0].Damage != 100 {
		t.Errorf("fresh entry should be untouched: %+v", entries)
	}
}
```

- [ ] **Step 2: Run, see fail**

```
go test ./monster -run 'TestDecayDamageEntries' -v
```
Expected: FAIL — `DecayDamageEntries` undefined.

- [ ] **Step 3: Implement script + method + summary type**

Append to `monster/registry.go`:

```go
// DecaySummary is returned by DecayDamageEntries.
type DecaySummary struct {
	Monster           Model
	PrevControllerId  uint32
	ControllerCleared bool
}

var decayDamageEntriesScript = goredis.NewScript(`
local key = KEYS[1]
local now = tonumber(ARGV[1])
local idleMs = tonumber(ARGV[2])
local mult = tonumber(ARGV[3])
local floorVal = tonumber(ARGV[4])
local j = redis.call('GET', key)
if not j then
    return redis.error_reply("monster not found")
end
local m = cjson.decode(j)

local entries = m.damageEntries
if type(entries) ~= 'table' then
    entries = {}
end

local kept = {}
for _, e in ipairs(entries) do
    if (now - e.lastHitMs) > idleMs then
        e.damage = math.floor(e.damage * mult)
    end
    if e.damage >= floorVal then
        table.insert(kept, e)
    end
end
m.damageEntries = kept

local prevControllerId = m.controlCharacterId
local controllerCleared = false
if #kept == 0 then
    if m.controlCharacterId ~= 0 then
        m.controlCharacterId = 0
        controllerCleared = true
    end
    m.controllerHasAggro = false
end

redis.call('SET', key, cjson.encode(m))
return cjson.encode({
    controllerCleared = controllerCleared,
    prevControllerId = prevControllerId,
    monster = m,
})
`)

func (r *Registry) DecayDamageEntries(t tenant.Model, uniqueId uint32, nowMs int64) (DecaySummary, error) {
	ctx := context.Background()
	key := monsterKey(t, uniqueId)

	result, err := decayDamageEntriesScript.Run(ctx, r.client, []string{key},
		strconv.FormatInt(nowMs, 10),
		strconv.FormatInt(AggroIdleThresholdMs, 10),
		strconv.FormatFloat(AggroDecayMultiplier, 'f', -1, 64),
		strconv.FormatUint(uint64(AggroDecayFloor), 10),
	).Result()
	if err != nil {
		return DecaySummary{}, err
	}
	resultStr, ok := result.(string)
	if !ok {
		return DecaySummary{}, errors.New("unexpected response type")
	}

	var env struct {
		ControllerCleared bool          `json:"controllerCleared"`
		PrevControllerId  uint32        `json:"prevControllerId"`
		Monster           storedMonster `json:"monster"`
	}
	if err := json.Unmarshal([]byte(resultStr), &env); err != nil {
		return DecaySummary{}, err
	}
	_, m, err := fromStored(env.Monster)
	if err != nil {
		return DecaySummary{}, err
	}
	return DecaySummary{
		Monster:           m,
		PrevControllerId:  env.PrevControllerId,
		ControllerCleared: env.ControllerCleared,
	}, nil
}
```

- [ ] **Step 4: Run, see pass**

```
go test ./monster -run 'TestDecayDamageEntries' -v
```
Expected: PASS for all four tests. The deterministic sequence test pins the floor math `100 → 85 → 72 → 61 → 51 → 43 → 36 → 30 → 25 → 21 → 17 → 14 → 11 → 9 → 7 → 5 → 4 → 3 → 2 → 1 → 0(prune)`.

- [ ] **Step 5: Run full suite**

```
go test ./monster -count=1
```
Expected: PASS.

- [ ] **Step 6: Commit**

```
git add services/atlas-monsters/atlas.com/monsters/monster/registry.go \
        services/atlas-monsters/atlas.com/monsters/monster/registry_test.go
git commit -m "feat(atlas-monsters): add decayDamageEntriesScript and DecayDamageEntries registry method"
```

---

## Task 8 — atlas-monsters: attackerInField helper for FR-10

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/processor.go` (add inFieldFn injection seam, helper method)
- Test: `services/atlas-monsters/atlas.com/monsters/monster/processor_test.go`

- [ ] **Step 1: Write failing test**

Append to `monster/processor_test.go`:

```go
// TestAttackerInField verifies the FR-10 helper:
//   - returns true when the attacker's id is in the field's character id list
//   - returns false when not
//   - returns false (fail-closed) on provider error
func TestAttackerInField(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()

	tests := []struct {
		name    string
		ids     []uint32
		err     error
		wantIn  bool
	}{
		{"in field", []uint32{1, 7, 9}, nil, true},
		{"not in field", []uint32{1, 9}, nil, false},
		{"empty field", []uint32{}, nil, false},
		{"provider error fails closed", nil, errors.New("boom"), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &ProcessorImpl{
				l:   logrus.New(),
				ctx: ctx,
				t:   ten,
				inFieldFn: func(_ field.Model) ([]uint32, error) {
					return tc.ids, tc.err
				},
			}
			got, err := p.attackerInField(f, 7)
			if tc.err != nil {
				if err == nil {
					t.Errorf("expected error from helper, got nil")
				}
			}
			if got != tc.wantIn {
				t.Errorf("attackerInField=%v want %v", got, tc.wantIn)
			}
		})
	}
}
```

(Add `"errors"` and `"github.com/Chronicle20/atlas/libs/atlas-constants/field"` to the test imports if not already present. Note: `processor_test.go` currently imports `"github.com/Chronicle20/atlas/libs/atlas-constants/field"` — confirm before editing.)

- [ ] **Step 2: Run, see fail**

```
go test ./monster -run TestAttackerInField -v
```
Expected: FAIL — `inFieldFn` field and `attackerInField` method don't exist.

- [ ] **Step 3: Add seam and helper to ProcessorImpl**

In `monster/processor.go`, modify `ProcessorImpl`:

```go
type ProcessorImpl struct {
	l         logrus.FieldLogger
	ctx       context.Context
	t         tenant.Model
	emit      emitter
	inFieldFn func(f field.Model) ([]uint32, error)
}
```

Update `NewProcessor` to default the seam:

```go
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
		emit: func(topic string, provider model.Provider[[]kafka.Message]) error {
			return producer.ProviderImpl(l)(ctx)(topic)(provider)
		},
	}
	p.inFieldFn = func(f field.Model) ([]uint32, error) {
		return _map.CharacterIdsInFieldProvider(p.l)(p.ctx)(f)()
	}
	return p
}
```

Add the helper:

```go
// attackerInField reports whether characterId is currently in the monster's
// field. Returns (false, err) on provider error so callers can fail closed
// (FR-10): we don't grant control to an attacker we cannot verify.
func (p *ProcessorImpl) attackerInField(f field.Model, characterId uint32) (bool, error) {
	ids, err := p.inFieldFn(f)
	if err != nil {
		return false, err
	}
	for _, id := range ids {
		if id == characterId {
			return true, nil
		}
	}
	return false, nil
}
```

- [ ] **Step 4: Default inFieldFn in the existing test helper**

In `monster/processor_test.go`, update `newRecordingProcessor` so future tests that exercise the controller-switch branch through it don't trip on a nil `inFieldFn`. Inside the helper, after constructing the `ProcessorImpl` literal, add:

```go
p.inFieldFn = func(_ field.Model) ([]uint32, error) {
	return nil, nil
}
```

(Add `field` to the imports if it isn't already; `processor_test.go` already imports `"github.com/Chronicle20/atlas/libs/atlas-constants/field"`.)

- [ ] **Step 5: Run, see pass**

```
go test ./monster -run TestAttackerInField -v
```
Expected: PASS.

- [ ] **Step 6: Run full suite**

```
go test ./monster -count=1
```
Expected: PASS.

- [ ] **Step 7: Commit**

```
git add services/atlas-monsters/atlas.com/monsters/monster/processor.go \
        services/atlas-monsters/atlas.com/monsters/monster/processor_test.go
git commit -m "feat(atlas-monsters): add inFieldFn injection seam and attackerInField helper for FR-10"
```

---

## Task 9 — atlas-monsters: refactor Damage with firstHitObserved + controllerSwitched + AGGRO_CHANGED

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/processor.go` (Damage method; route StartControl/StopControl through p.emit)
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/kafka.go` (add AggroChanged constant + body)
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/producer.go` (add aggroChangedStatusEventProvider)
- Test: `services/atlas-monsters/atlas.com/monsters/monster/processor_test.go`

This is the largest task. Decompose into substeps. Note that `StartControl`, `StopControl`, and `Create` currently call `producer.ProviderImpl(...)` directly rather than `p.emit`. To make tests fully observable, route those emissions through `p.emit` as well.

- [ ] **Step 1: Write failing tests**

Append to `monster/processor_test.go`:

```go
// helpers used by the new tests
type emittedBody struct {
	Topic string
	Type  string
	Body  json.RawMessage
}

func newRecordingProcessorWithBodies(t *testing.T, ten tenant.Model) (*ProcessorImpl, *[]emittedBody) {
	t.Helper()
	var events []emittedBody
	p := &ProcessorImpl{
		l:   logrus.New(),
		ctx: context.Background(),
		t:   ten,
		emit: func(topic string, provider model.Provider[[]kafka.Message]) error {
			msgs, err := provider()
			if err != nil {
				t.Fatalf("provider error: %v", err)
			}
			for _, m := range msgs {
				var env struct {
					Type string          `json:"type"`
					Body json.RawMessage `json:"body"`
				}
				if err := json.Unmarshal(m.Value, &env); err != nil {
					t.Fatalf("decode emitted: %v", err)
				}
				events = append(events, emittedBody{Topic: topic, Type: env.Type, Body: env.Body})
			}
			return nil
		},
		inFieldFn: func(_ field.Model) ([]uint32, error) {
			return []uint32{1, 2, 3, 4}, nil
		},
	}
	return p, &events
}

// TestDamageControllerSwitchOnDpsLead — character 2 takes lead from character 1
// (the current controller). Expect STOP_CONTROL (for 1) then START_CONTROL
// (for 2) with controllerHasAggro=true. AGGRO_CHANGED suppressed because the
// switch carries the flag.
func TestDamageControllerSwitchOnDpsLead(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 1000, 50)
	uniqueId := m.UniqueId()
	// Pre-populate: character 1 controls and leads damage.
	if _, err := r.ControlMonster(ten, uniqueId, 1); err != nil {
		t.Fatalf("ControlMonster: %v", err)
	}
	if _, err := r.ApplyDamage(ten, 1, 50, uniqueId, time.Now().UnixMilli()); err != nil {
		t.Fatalf("seed: %v", err)
	}

	p, events := newRecordingProcessorWithBodies(t, ten)
	p.Damage(uniqueId, 2, []uint32{500}, 0)

	var types []string
	for _, e := range *events {
		types = append(types, e.Type)
	}
	// Expected order: DAMAGED, STOP_CONTROL, START_CONTROL.
	if len(types) != 3 ||
		types[0] != EventMonsterStatusDamaged ||
		types[1] != EventMonsterStatusStopControl ||
		types[2] != EventMonsterStatusStartControl {
		t.Fatalf("unexpected event order: %v", types)
	}
	// START_CONTROL body must carry controllerHasAggro=true.
	var body statusEventStartControlBody
	if err := json.Unmarshal((*events)[2].Body, &body); err != nil {
		t.Fatalf("decode start control: %v", err)
	}
	if !body.ControllerHasAggro {
		t.Errorf("START_CONTROL body controllerHasAggro=true expected, got false")
	}
	if body.ActorId != 2 {
		t.Errorf("START_CONTROL ActorId=2 expected, got %d", body.ActorId)
	}
}

// TestDamageNoSwitchWhenLeaderUnchanged — current controller takes more damage
// and stays leader. No STOP/START, but AGGRO_CHANGED should fire (first hit
// flips the flag).
func TestDamageNoSwitchWhenLeaderUnchanged(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 1000, 50)
	uniqueId := m.UniqueId()
	// Controller is set; controllerHasAggro starts false.
	if _, err := r.ControlMonster(ten, uniqueId, 1); err != nil {
		t.Fatalf("ControlMonster: %v", err)
	}

	p, events := newRecordingProcessorWithBodies(t, ten)
	p.Damage(uniqueId, 1, []uint32{30}, 0)

	var types []string
	for _, e := range *events {
		types = append(types, e.Type)
	}
	if len(types) != 2 ||
		types[0] != EventMonsterStatusDamaged ||
		types[1] != EventMonsterStatusAggroChanged {
		t.Fatalf("expected DAMAGED + AGGRO_CHANGED, got %v", types)
	}
	var body statusEventAggroChangedBody
	if err := json.Unmarshal((*events)[1].Body, &body); err != nil {
		t.Fatalf("decode aggro changed: %v", err)
	}
	if body.ControllerCharacterId != 1 || !body.ControllerHasAggro {
		t.Errorf("AGGRO_CHANGED body unexpected: %+v", body)
	}
}

// TestDamageAggroChangedSuppressedOnSwitch — when first hit also triggers a
// controller switch, AGGRO_CHANGED is NOT emitted.
func TestDamageAggroChangedSuppressedOnSwitch(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 1000, 50)
	uniqueId := m.UniqueId()
	if _, err := r.ControlMonster(ten, uniqueId, 1); err != nil {
		t.Fatalf("ControlMonster: %v", err)
	}

	p, events := newRecordingProcessorWithBodies(t, ten)
	// Character 2 hits first AND becomes leader.
	p.Damage(uniqueId, 2, []uint32{500}, 0)

	for _, e := range *events {
		if e.Type == EventMonsterStatusAggroChanged {
			t.Fatalf("AGGRO_CHANGED must be suppressed when controller switch carries the flag")
		}
	}
}

// TestDamageFR9NoStopWhenControllerZero — controller is 0; first attacker
// becomes controller via a single START_CONTROL with no preceding STOP_CONTROL.
func TestDamageFR9NoStopWhenControllerZero(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 1000, 50)
	uniqueId := m.UniqueId()

	p, events := newRecordingProcessorWithBodies(t, ten)
	p.Damage(uniqueId, 7, []uint32{30}, 0)

	for _, e := range *events {
		if e.Type == EventMonsterStatusStopControl {
			t.Fatalf("STOP_CONTROL must NOT precede START_CONTROL when controller was 0")
		}
	}
	// We expect DAMAGED + START_CONTROL (first hit on monster with no controller
	// keeps WasFirstHit=false, so no AGGRO_CHANGED at this stage).
	var saw bool
	for _, e := range *events {
		if e.Type == EventMonsterStatusStartControl {
			saw = true
		}
	}
	if !saw {
		t.Errorf("expected START_CONTROL, got %v", *events)
	}
}

// TestDamageFR10OutOfFieldSkipsSwitch — attacker not in field: damage applies,
// controller is NOT switched.
func TestDamageFR10OutOfFieldSkipsSwitch(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 1000, 50)
	uniqueId := m.UniqueId()
	if _, err := r.ControlMonster(ten, uniqueId, 1); err != nil {
		t.Fatalf("ControlMonster: %v", err)
	}
	// Seed the existing controller as leader.
	if _, err := r.ApplyDamage(ten, 1, 50, uniqueId, time.Now().UnixMilli()); err != nil {
		t.Fatalf("seed: %v", err)
	}

	p, events := newRecordingProcessorWithBodies(t, ten)
	// Override inFieldFn so character 2 is NOT in field.
	p.inFieldFn = func(_ field.Model) ([]uint32, error) {
		return []uint32{1}, nil
	}
	p.Damage(uniqueId, 2, []uint32{500}, 0)

	for _, e := range *events {
		if e.Type == EventMonsterStatusStopControl || e.Type == EventMonsterStatusStartControl {
			t.Fatalf("FR-10: out-of-field attacker should not switch controller, got %s", e.Type)
		}
	}
	// Damage still applied.
	got, _ := r.GetMonster(ten, uniqueId)
	if got.Hp() != 500 {
		t.Errorf("expected HP=500 after 500 damage, got %d", got.Hp())
	}
}
```

(Imports: this file already has `context`, `encoding/json`, `testing`, `field`, `channel`, `_map`, `world`, `tenant`, `uuid`, `kafka-go`, `logrus`, `model`. Add `"time"` if not present.)

- [ ] **Step 2: Run, see fail**

```
go test ./monster -run 'TestDamageControllerSwitchOnDpsLead|TestDamageNoSwitchWhenLeaderUnchanged|TestDamageAggroChangedSuppressedOnSwitch|TestDamageFR9NoStopWhenControllerZero|TestDamageFR10OutOfFieldSkipsSwitch' -v
```
Expected: FAIL — `EventMonsterStatusAggroChanged` and `statusEventAggroChangedBody` don't exist; `statusEventStartControlBody.ControllerHasAggro` doesn't exist.

- [ ] **Step 3: Add Kafka constant and body**

In `monster/kafka.go`, add to the const block:

```go
EventMonsterStatusAggroChanged = "AGGRO_CHANGED"
```

Add `ControllerHasAggro` to the existing `statusEventStartControlBody`:

```go
type statusEventStartControlBody struct {
	ActorId            uint32 `json:"actorId"`
	X                  int16  `json:"x"`
	Y                  int16  `json:"y"`
	Stance             byte   `json:"stance"`
	FH                 int16  `json:"fh"`
	Team               int8   `json:"team"`
	ControllerHasAggro bool   `json:"controllerHasAggro"`
}
```

Add the new body type at the end of the file:

```go
type statusEventAggroChangedBody struct {
	ControllerCharacterId uint32 `json:"controllerCharacterId"`
	ControllerHasAggro    bool   `json:"controllerHasAggro"`
}
```

- [ ] **Step 4: Add producer + update existing producer**

In `monster/producer.go`, replace `startControlStatusEventProvider`:

```go
func startControlStatusEventProvider(m Model) model.Provider[[]kafka.Message] {
	return statusEventProvider(m.Field(), m.UniqueId(), m.MonsterId(), EventMonsterStatusStartControl, statusEventStartControlBody{
		ActorId:            m.ControlCharacterId(),
		X:                  m.X(),
		Y:                  m.Y(),
		Stance:             m.Stance(),
		FH:                 m.Fh(),
		Team:               m.Team(),
		ControllerHasAggro: m.ControllerHasAggro(),
	})
}
```

Add a new provider:

```go
func aggroChangedStatusEventProvider(m Model, controllerCharacterId uint32, hasAggro bool) model.Provider[[]kafka.Message] {
	return statusEventProvider(m.Field(), m.UniqueId(), m.MonsterId(), EventMonsterStatusAggroChanged, statusEventAggroChangedBody{
		ControllerCharacterId: controllerCharacterId,
		ControllerHasAggro:    hasAggro,
	})
}
```

- [ ] **Step 5: Refactor Damage and route StartControl/StopControl through p.emit**

In `monster/processor.go`:

5a. Replace `StartControl` to use `p.emit`:

```go
func (p *ProcessorImpl) StartControl(uniqueId uint32, controllerId uint32) (Model, error) {
	m, err := p.GetById(uniqueId)
	if err != nil {
		return Model{}, err
	}

	if m.ControlCharacterId() != 0 {
		err = p.StopControl(m)
		if err != nil {
			return Model{}, err
		}
	}

	m, err = GetMonsterRegistry().ControlMonster(p.t, uniqueId, controllerId)
	if err == nil {
		_ = p.emit(EnvEventTopicMonsterStatus, startControlStatusEventProvider(m))
	}
	return m, err
}
```

(Drop the redundant intermediate `GetById` after `StopControl` — `ControlMonster` is the source of truth and re-loads internally.)

5b. Replace `StopControl` to use `p.emit`:

```go
func (p *ProcessorImpl) StopControl(m Model) error {
	oldControllerId := m.ControlCharacterId()
	m, err := GetMonsterRegistry().ClearControl(p.t, m.UniqueId())
	if err == nil {
		_ = p.emit(EnvEventTopicMonsterStatus, stopControlStatusEventProvider(m, oldControllerId))
	}
	return err
}
```

5c. Replace `Damage`:

```go
func (p *ProcessorImpl) Damage(id uint32, characterId uint32, damages []uint32, attackType byte) {
	if len(damages) == 0 {
		return
	}

	m, err := GetMonsterRegistry().GetMonster(p.t, id)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to get monster [%d].", id)
		return
	}
	if !m.Alive() {
		p.l.Debugf("Character [%d] trying to apply damage to an already dead monster [%d].", characterId, id)
		return
	}

	// Reflect runs once per attack, not once per line.
	p.checkReflect(m, characterId, attackType)

	var isBoss bool
	var revives []uint32
	if ma, infoErr := information.GetById(p.l)(p.ctx)(m.MonsterId()); infoErr == nil {
		isBoss = ma.Boss()
		revives = ma.Revives()
	}

	nowMs := time.Now().UnixMilli()
	var last DamageSummary
	hasLast := false
	killed := false
	firstHitObserved := false
	for _, d := range damages {
		s, err := GetMonsterRegistry().ApplyDamage(p.t, characterId, d, m.UniqueId(), nowMs)
		if err != nil {
			p.l.WithError(err).Errorf("Error applying damage to monster %d from character %d.", m.UniqueId(), characterId)
			break
		}
		last = s
		hasLast = true
		if s.WasFirstHit {
			firstHitObserved = true
		}
		if s.Killed {
			killed = true
			break
		}
	}

	if !hasLast {
		return
	}

	if err := p.emit(EnvEventTopicMonsterStatus, damagedStatusEventProvider(last.Monster, last.CharacterId, last.CharacterId, isBoss, DamageSourceCharacterAttack, last.Monster.DamageSummary())); err != nil {
		p.l.WithError(err).Errorf("Monster [%d] damaged, but unable to display that for the characters in the field.", last.Monster.UniqueId())
	}

	if killed {
		GetCooldownRegistry().ClearCooldowns(p.ctx, p.t, id)
		GetDropTimerRegistry().Unregister(p.ctx, p.t, id)

		for _, se := range last.Monster.StatusEffects() {
			_ = p.emit(EnvEventTopicMonsterStatus, statusEffectCancelledEventProvider(last.Monster, se))
		}

		if err := p.emit(EnvEventTopicMonsterStatus, killedStatusEventProvider(last.Monster, last.CharacterId, isBoss, last.Monster.DamageSummary())); err != nil {
			p.l.WithError(err).Errorf("Monster [%d] killed, but unable to display that for the characters in the field.", last.Monster.UniqueId())
		}
		if _, err := GetMonsterRegistry().RemoveMonster(p.ctx, p.t, last.Monster.UniqueId()); err != nil {
			p.l.WithError(err).Errorf("Monster [%d] killed, but not removed from registry.", last.Monster.UniqueId())
		}

		if len(revives) > 0 {
			p.spawnRevives(last.Monster, revives)
		}
		return
	}

	// Controller-switch and aggro-flag emission.
	//
	// Decision 4 (PRD §8.4): we keep the two-step StopControl + StartControl
	// rather than collapsing into a single Lua. Two concurrent damage events
	// for the same monster could interleave and produce redundant
	// STOP_CONTROL/START_CONTROL pairs; this is acceptable because Kafka
	// partition ordering preserves causality and the channel re-applies
	// idempotently for re-control to the same character.
	controllerSwitched := false
	if characterId != last.Monster.ControlCharacterId() && last.Monster.DamageLeader() == characterId {
		inField, ferr := p.attackerInField(last.Monster.Field(), characterId)
		if ferr != nil || !inField {
			p.l.Debugf("FR-10: skipping controller switch for char [%d] not in field of monster [%d].", characterId, last.Monster.UniqueId())
		} else {
			p.l.Debugf("Character [%d] has become damage leader for monster [%d].", characterId, last.Monster.UniqueId())
			// FR-9: only emit STOP_CONTROL when there's actually a previous controller.
			if last.Monster.ControlCharacterId() != 0 {
				if err := p.StopControl(last.Monster); err != nil {
					p.l.WithError(err).Errorf("Unable to stop [%d] from controlling monster [%d].", last.Monster.ControlCharacterId(), last.Monster.UniqueId())
				}
			}
			if _, err := p.StartControl(last.Monster.UniqueId(), characterId); err != nil {
				p.l.WithError(err).Errorf("Unable to start [%d] controlling monster [%d].", characterId, last.Monster.UniqueId())
			} else {
				controllerSwitched = true
			}
		}
	}

	if firstHitObserved && !controllerSwitched {
		// AGGRO_CHANGED is suppressed when a switch happened because START_CONTROL
		// already carries controllerHasAggro: true (FR-22).
		latest, err := GetMonsterRegistry().GetMonster(p.t, last.Monster.UniqueId())
		if err != nil {
			p.l.WithError(err).Errorf("Unable to re-load monster [%d] for AGGRO_CHANGED emit.", last.Monster.UniqueId())
		} else {
			_ = p.emit(EnvEventTopicMonsterStatus, aggroChangedStatusEventProvider(latest, latest.ControlCharacterId(), latest.ControllerHasAggro()))
			p.l.Debugf("Monster [%d] aggro changed for controller [%d].", latest.UniqueId(), latest.ControlCharacterId())
		}
	}
}
```

Note: the existing `Damage` method calls `producer.ProviderImpl(...)` for several emissions (damaged, killed, status-cancelled). The new version routes them through `p.emit` so tests can observe them. Other callers of `producer.ProviderImpl` in `processor.go` (e.g., in `Create`, `executeHeal`, `ApplyStatusEffect`) are out of scope for this task — leave them unchanged.

Note also: the existing `Create` method emits a `CREATED` event and then immediately calls `p.StartControl`. That `StartControl` now emits via `p.emit` — but `Create` itself still uses `producer.ProviderImpl` for its `CREATED` emission. Leave `Create` as-is for this task; the change is local to controller events.

- [ ] **Step 6: Run failing tests**

```
go test ./monster -run 'TestDamageControllerSwitchOnDpsLead|TestDamageNoSwitchWhenLeaderUnchanged|TestDamageAggroChangedSuppressedOnSwitch|TestDamageFR9NoStopWhenControllerZero|TestDamageFR10OutOfFieldSkipsSwitch' -v
```
Expected: PASS.

- [ ] **Step 7: Run full suite**

```
go test ./monster -count=1
```
Expected: PASS. The pre-existing `TestDamageMultiLineKillOnLastLine`, `TestDamageMultiLineKillOnMiddleLine`, `TestDamageSingleLineKill` still expect 2 events (DAMAGED + KILLED) — verify they still pass; the new code preserves that behavior because killed-path returns before the controller-switch + aggro-changed branch.

- [ ] **Step 8: Commit**

```
git add services/atlas-monsters/atlas.com/monsters/monster/processor.go \
        services/atlas-monsters/atlas.com/monsters/monster/kafka.go \
        services/atlas-monsters/atlas.com/monsters/monster/producer.go \
        services/atlas-monsters/atlas.com/monsters/monster/processor_test.go
git commit -m "feat(atlas-monsters): switch controller on DPS lead and emit AGGRO_CHANGED on aggro flip"
```

---

## Task 10 — atlas-monsters: encode-level test for AGGRO_CHANGED + START_CONTROL bodies

**Files:**
- Test: `services/atlas-monsters/atlas.com/monsters/monster/producer_test.go` (new)

This task adds a focused encode test independent of the Damage flow.

- [ ] **Step 1: Write failing test**

Create `monster/producer_test.go`:

```go
package monster

import (
	"encoding/json"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

func TestStartControlBodyEncodesControllerHasAggro(t *testing.T) {
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := Clone(NewMonster(f, 1, 9300018, 0, 0, 0, 5, 0, 100, 50)).
		SetControlCharacterId(42).
		SetControllerHasAggro(true).
		Build()
	msgs, err := startControlStatusEventProvider(m)()
	if err != nil {
		t.Fatalf("provider error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	var env struct {
		Type string                       `json:"type"`
		Body statusEventStartControlBody `json:"body"`
	}
	if err := json.Unmarshal(msgs[0].Value, &env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if env.Type != EventMonsterStatusStartControl {
		t.Errorf("type=%s, want %s", env.Type, EventMonsterStatusStartControl)
	}
	if env.Body.ActorId != 42 {
		t.Errorf("ActorId=%d, want 42", env.Body.ActorId)
	}
	if !env.Body.ControllerHasAggro {
		t.Errorf("ControllerHasAggro=%v, want true", env.Body.ControllerHasAggro)
	}
}

func TestAggroChangedBodyEncoding(t *testing.T) {
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).
		SetInstance(uuid.Nil).Build()
	m := Clone(NewMonster(f, 5, 9300018, 0, 0, 0, 5, 0, 100, 50)).
		SetControlCharacterId(7).
		SetControllerHasAggro(true).
		Build()
	msgs, err := aggroChangedStatusEventProvider(m, 7, true)()
	if err != nil {
		t.Fatalf("provider error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	var env struct {
		Type string                        `json:"type"`
		Body statusEventAggroChangedBody  `json:"body"`
	}
	if err := json.Unmarshal(msgs[0].Value, &env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if env.Type != EventMonsterStatusAggroChanged {
		t.Errorf("type=%s, want %s", env.Type, EventMonsterStatusAggroChanged)
	}
	if env.Body.ControllerCharacterId != 7 || !env.Body.ControllerHasAggro {
		t.Errorf("body unexpected: %+v", env.Body)
	}
}
```

- [ ] **Step 2: Run, see pass**

```
go test ./monster -run 'TestStartControlBodyEncodesControllerHasAggro|TestAggroChangedBodyEncoding' -v
```
Expected: PASS (Task 9 already added the producers and bodies).

- [ ] **Step 3: Commit**

```
git add services/atlas-monsters/atlas.com/monsters/monster/producer_test.go
git commit -m "test(atlas-monsters): pin AGGRO_CHANGED and START_CONTROL body encoding"
```

---

## Task 11 — atlas-monsters: confirm boss exemption for controller-switch path is documented

The PRD says bosses still controller-switch on damage; only the decay sweep skips them. Task 9 doesn't gate the switch on `isBoss`, so this is already correct. Just document it.

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/processor.go` (one-line comment at the controller-switch site)

- [ ] **Step 1: Add the comment**

In the `Damage` method, just above the `if characterId != last.Monster.ControlCharacterId() && ...` block, prepend:

```go
// Controller-switch on DPS lead applies to bosses too. Only the decay sweep
// (MonsterAggroDecayTask) treats bosses specially.
```

- [ ] **Step 2: Verify**

```
go vet ./monster
go test ./monster -count=1
```
Expected: PASS.

- [ ] **Step 3: Commit**

```
git add services/atlas-monsters/atlas.com/monsters/monster/processor.go
git commit -m "docs(atlas-monsters): clarify boss exemption applies only to decay, not controller-switch"
```

---

## Task 12 — atlas-monsters: MonsterAggroDecayTask

**Files:**
- Create: `services/atlas-monsters/atlas.com/monsters/monster/aggro_task.go`
- Test: `services/atlas-monsters/atlas.com/monsters/monster/aggro_task_test.go` (new)

The task injects a `bossLookupFn func(uint32) bool` and an `emit emitter` for testability.

- [ ] **Step 1: Write failing tests**

Create `monster/aggro_task_test.go`:

```go
package monster

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

type recordedEmit struct {
	Topic   string
	Type    string
	ActorId uint32
}

func newAggroTaskWithRecorder(t *testing.T, bossIds map[uint32]bool) (*MonsterAggroDecayTask, *[]recordedEmit, *int) {
	t.Helper()
	var events []recordedEmit
	bossCalls := 0
	tk := &MonsterAggroDecayTask{
		l:        logrus.New(),
		ctx:      context.Background(),
		interval: AggroSweepInterval,
		bossLookupFn: func(monsterId uint32) bool {
			bossCalls++
			return bossIds[monsterId]
		},
		emit: func(_ tenant.Model, topic string, provider model.Provider[[]kafka.Message]) error {
			msgs, err := provider()
			if err != nil {
				t.Fatalf("provider err: %v", err)
			}
			for _, m := range msgs {
				var env struct {
					Type string `json:"type"`
					Body struct {
						ActorId uint32 `json:"actorId"`
					} `json:"body"`
				}
				if err := json.Unmarshal(m.Value, &env); err != nil {
					t.Fatalf("decode: %v", err)
				}
				events = append(events, recordedEmit{Topic: topic, Type: env.Type, ActorId: env.Body.ActorId})
			}
			return nil
		},
	}
	return tk, &events, &bossCalls
}

// TestAggroDecayTaskFullClearEmitsStopControl seeds a non-boss monster with a
// tiny entry and a controller, fast-forwards the wall-clock past the idle
// threshold, runs Run(), and asserts STOP_CONTROL is emitted with the previous
// controller id.
func TestAggroDecayTaskFullClearEmitsStopControl(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 1000, 50)
	if _, err := r.ControlMonster(ten, m.UniqueId(), 42); err != nil {
		t.Fatalf("ControlMonster: %v", err)
	}
	if _, err := r.ApplyDamage(ten, 1, 1, m.UniqueId(), 0); err != nil {
		t.Fatalf("ApplyDamage: %v", err)
	}

	tk, events, _ := newAggroTaskWithRecorder(t, nil /* no bosses */)
	// Override now to be far in the future, satisfying the idle threshold.
	tk.nowFn = func() int64 { return AggroIdleThresholdMs + 1_000 }
	tk.Run()

	if len(*events) != 1 {
		t.Fatalf("expected 1 event, got %d (%+v)", len(*events), *events)
	}
	if (*events)[0].Type != EventMonsterStatusStopControl {
		t.Errorf("type=%s, want STOP_CONTROL", (*events)[0].Type)
	}
	if (*events)[0].ActorId != 42 {
		t.Errorf("ActorId=%d, want 42", (*events)[0].ActorId)
	}
}

// TestAggroDecayTaskBossExemption: boss monsters skip decay entirely.
func TestAggroDecayTaskBossExemption(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	bossTemplate := uint32(8800000)
	m := r.CreateMonster(ctx, ten, f, bossTemplate, 0, 0, 0, 5, 0, 1000, 50)
	if _, err := r.ControlMonster(ten, m.UniqueId(), 42); err != nil {
		t.Fatalf("ControlMonster: %v", err)
	}
	if _, err := r.ApplyDamage(ten, 1, 1, m.UniqueId(), 0); err != nil {
		t.Fatalf("ApplyDamage: %v", err)
	}

	tk, events, _ := newAggroTaskWithRecorder(t, map[uint32]bool{bossTemplate: true})
	tk.nowFn = func() int64 { return AggroIdleThresholdMs + 1_000 }
	tk.Run()

	if len(*events) != 0 {
		t.Fatalf("boss should be skipped, got %d events", len(*events))
	}
	got, _ := r.GetMonster(ten, m.UniqueId())
	if got.ControlCharacterId() != 42 {
		t.Errorf("boss controller cleared unexpectedly")
	}
	if len(got.DamageEntries()) != 1 {
		t.Errorf("boss damage entries decayed unexpectedly")
	}
}

// TestAggroDecayTaskBossCacheHitsLookupOncePerTemplate verifies the per-tick
// boss-flag cache.
func TestAggroDecayTaskBossCacheHitsLookupOncePerTemplate(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()

	// 3 monsters of the same template, plus 1 of a different template.
	for i := 0; i < 3; i++ {
		m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 1000, 50)
		if _, err := r.ApplyDamage(ten, 1, 100, m.UniqueId(), 0); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	other := r.CreateMonster(ctx, ten, f, 9300019, 0, 0, 0, 5, 0, 1000, 50)
	if _, err := r.ApplyDamage(ten, 1, 100, other.UniqueId(), 0); err != nil {
		t.Fatalf("seed: %v", err)
	}

	tk, _, calls := newAggroTaskWithRecorder(t, nil)
	tk.nowFn = func() int64 { return AggroIdleThresholdMs + 1_000 }
	tk.Run()

	// Two distinct templates -> 2 lookups regardless of monster count.
	if *calls != 2 {
		t.Errorf("bossLookupFn called %d times, want 2", *calls)
	}
}

// TestAggroDecayTaskNoOpWhenAllFresh: monsters whose entries are all fresh are
// not touched and don't emit STOP_CONTROL.
func TestAggroDecayTaskNoOpWhenAllFresh(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 1000, 50)
	if _, err := r.ControlMonster(ten, m.UniqueId(), 42); err != nil {
		t.Fatalf("ControlMonster: %v", err)
	}
	now := int64(20_000)
	if _, err := r.ApplyDamage(ten, 1, 100, m.UniqueId(), now); err != nil {
		t.Fatalf("ApplyDamage: %v", err)
	}

	tk, events, _ := newAggroTaskWithRecorder(t, nil)
	tk.nowFn = func() int64 { return now }
	tk.Run()
	if len(*events) != 0 {
		t.Fatalf("expected no events, got %d", len(*events))
	}
}

// SleepTime returns the configured interval.
func TestAggroDecayTaskSleepTime(t *testing.T) {
	tk := &MonsterAggroDecayTask{interval: 1500 * time.Millisecond}
	if tk.SleepTime() != 1500*time.Millisecond {
		t.Errorf("SleepTime=%v, want 1500ms", tk.SleepTime())
	}
}
```

- [ ] **Step 2: Run, see fail**

```
go test ./monster -run 'TestAggroDecayTask' -v
```
Expected: FAIL — `MonsterAggroDecayTask` undefined.

- [ ] **Step 3: Implement aggro_task.go**

Create `services/atlas-monsters/atlas.com/monsters/monster/aggro_task.go`:

```go
package monster

import (
	"atlas-monsters/kafka/producer"
	"atlas-monsters/monster/information"
	"context"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// taskEmitter publishes a kafka message provider on behalf of a tenant.
// Injected for tests.
type taskEmitter func(t tenant.Model, topic string, provider model.Provider[[]kafka.Message]) error

type MonsterAggroDecayTask struct {
	l            logrus.FieldLogger
	ctx          context.Context
	interval     time.Duration
	bossLookupFn func(monsterTemplateId uint32) bool
	emit         taskEmitter
	nowFn        func() int64
}

func NewMonsterAggroDecayTask(l logrus.FieldLogger, ctx context.Context, interval time.Duration) *MonsterAggroDecayTask {
	l.Infof("Initializing monster aggro decay task to run every %dms.", interval.Milliseconds())
	tk := &MonsterAggroDecayTask{
		l:        l,
		ctx:      ctx,
		interval: interval,
		nowFn:    func() int64 { return time.Now().UnixMilli() },
	}
	tk.bossLookupFn = func(monsterTemplateId uint32) bool {
		ma, err := information.GetById(tk.l)(tk.ctx)(monsterTemplateId)
		if err != nil {
			// Best-effort: treat as non-boss so decay proceeds.
			return false
		}
		return ma.Boss()
	}
	tk.emit = func(t tenant.Model, topic string, provider model.Provider[[]kafka.Message]) error {
		tctx := tenant.WithContext(tk.ctx, t)
		return producer.ProviderImpl(tk.l)(tctx)(topic)(provider)
	}
	return tk
}

func (tk *MonsterAggroDecayTask) SleepTime() time.Duration {
	return tk.interval
}

func (tk *MonsterAggroDecayTask) Run() {
	monsters := GetMonsterRegistry().GetMonsters()
	bossCache := make(map[uint32]bool)
	nowMs := tk.nowFn()

	for ten, mons := range monsters {
		for _, m := range mons {
			templateId := m.MonsterId()
			isBoss, ok := bossCache[templateId]
			if !ok {
				isBoss = tk.bossLookupFn(templateId)
				bossCache[templateId] = isBoss
			}
			if isBoss {
				continue
			}
			entries := m.DamageEntries()
			if len(entries) == 0 {
				continue
			}
			needsWork := false
			for _, e := range entries {
				if IsAggroIdle(e, nowMs) {
					needsWork = true
					break
				}
			}
			if !needsWork {
				continue
			}
			summary, err := GetMonsterRegistry().DecayDamageEntries(ten, m.UniqueId(), nowMs)
			if err != nil {
				tk.l.WithError(err).Errorf("Decay failed for monster [%d].", m.UniqueId())
				continue
			}
			if summary.ControllerCleared {
				_ = tk.emit(ten, EnvEventTopicMonsterStatus, stopControlStatusEventProvider(summary.Monster, summary.PrevControllerId))
				tk.l.Debugf("Aggro decay cleared controller [%d] for monster [%d].", summary.PrevControllerId, summary.Monster.UniqueId())
			}
		}
	}
}
```

- [ ] **Step 4: Run, see pass**

```
go test ./monster -run 'TestAggroDecayTask' -v
```
Expected: PASS.

- [ ] **Step 5: Run full suite**

```
go test ./monster -count=1
```
Expected: PASS.

- [ ] **Step 6: Commit**

```
git add services/atlas-monsters/atlas.com/monsters/monster/aggro_task.go \
        services/atlas-monsters/atlas.com/monsters/monster/aggro_task_test.go
git commit -m "feat(atlas-monsters): add MonsterAggroDecayTask with boss-cache and Go pre-filter"
```

---

## Task 13 — atlas-monsters: register MonsterAggroDecayTask in main.go

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/main.go`

- [ ] **Step 1: Add the registration**

In `services/atlas-monsters/atlas.com/monsters/main.go`, after the existing
```go
tasks.Register(l, tdm.Context())(monster.NewStatusExpirationTask(l, tdm.Context(), time.Second))
```
line, append:
```go
tasks.Register(l, tdm.Context())(monster.NewMonsterAggroDecayTask(l, tdm.Context(), monster.AggroSweepInterval))
```

- [ ] **Step 2: Verify**

```
cd services/atlas-monsters/atlas.com/monsters
go build ./...
go test ./...
```
Expected: PASS.

- [ ] **Step 3: Commit**

```
git add services/atlas-monsters/atlas.com/monsters/main.go
git commit -m "feat(atlas-monsters): register MonsterAggroDecayTask alongside StatusExpirationTask"
```

---

## Task 14 — atlas-monsters: docs/kafka.md updates

**Files:**
- Modify: `services/atlas-monsters/docs/kafka.md`

- [ ] **Step 1: Update START_CONTROL section**

In `services/atlas-monsters/docs/kafka.md` under `#### START_CONTROL`, replace the JSON body to include `controllerHasAggro` and add an explanatory paragraph beneath:

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "uniqueId": 0,
  "monsterId": 0,
  "type": "START_CONTROL",
  "body": {
    "actorId": 0,
    "x": 0,
    "y": 0,
    "stance": 0,
    "fh": 0,
    "team": 0,
    "controllerHasAggro": false
  }
}
```

> `controllerHasAggro` is `true` when the controller is engaged with the monster (active control — auto-attack timer running) and `false` when idle (passive). atlas-channel uses this to pick `ControlMonsterTypeActiveRequest` vs `ControlMonsterTypeActiveInit` when calling `StartControlMonsterBody`.

- [ ] **Step 2: Add AGGRO_CHANGED section**

After `#### STOP_CONTROL`, add a new heading and JSON block:

```markdown
#### AGGRO_CHANGED

Emitted when `controllerHasAggro` flips on a monster *without* a controller change. The most common case is the existing controller landing the first hit on a freshly-spawned mob: the controller stays the same, but the monster transitions from passive (idle/wander) to active (engaged). atlas-channel responds by re-sending `MonsterControlWriter` to the controller's session with the new control type, but does NOT emit a STOP_CONTROL.

Suppressed when the same damage line that flipped the flag also triggered a controller switch — in that case the new `START_CONTROL` carries `controllerHasAggro: true` and an additional event would be redundant.

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "uniqueId": 0,
  "monsterId": 0,
  "type": "AGGRO_CHANGED",
  "body": {
    "controllerCharacterId": 0,
    "controllerHasAggro": false
  }
}
```
```

- [ ] **Step 3: Add Background Tasks entry**

Locate the "Background Tasks" section if one exists; otherwise, search for "StatusExpirationTask" to find where to insert. Add a new subsection (placement near other background-task descriptions):

```markdown
#### MonsterAggroDecayTask

Runs every 1500ms. For each non-boss monster across all tenants:
- Skips bosses (`information.Boss() == true`) and monsters with empty damage tables.
- Pre-filters in Go: if no damage entry has been idle longer than `AggroIdleThresholdMs` (10s), the monster is skipped without a Redis write.
- Otherwise calls `decayDamageEntriesScript` (Lua) which decays idle entries by `AggroDecayMultiplier` (0.85) per tick and prunes any below `AggroDecayFloor` (1).
- When all entries are pruned and the monster has a controller, the script clears the controller and the task emits `STOP_CONTROL` with the previous controller id. `controllerHasAggro` resets to `false`.

Boss monsters retain their damage table and controller until death.
```

- [ ] **Step 4: Verify**

The doc has no compile-time checks. Skim the diff for accuracy.

```
git diff services/atlas-monsters/docs/kafka.md
```

- [ ] **Step 5: Commit**

```
git add services/atlas-monsters/docs/kafka.md
git commit -m "docs(atlas-monsters): document AGGRO_CHANGED event and aggro decay task"
```

---

## Task 15 — atlas-channel: extend StatusEventStartControlBody and add AGGRO_CHANGED types

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka.go`
- Test: `services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka_test.go`

- [ ] **Step 1: Write failing test**

Append to `kafka_test.go`:

```go
func TestStatusEventStartControlBody_DecodesControllerHasAggro(t *testing.T) {
	raw := []byte(`{"actorId":42,"x":1,"y":2,"stance":3,"fh":4,"team":5,"controllerHasAggro":true}`)
	var body StatusEventStartControlBody
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !body.ControllerHasAggro {
		t.Errorf("expected ControllerHasAggro=true, got %v", body.ControllerHasAggro)
	}
	if body.ActorId != 42 {
		t.Errorf("expected ActorId=42, got %d", body.ActorId)
	}
}

func TestStatusEventStartControlBody_LegacyDefaultsFalse(t *testing.T) {
	raw := []byte(`{"actorId":42,"x":1,"y":2,"stance":3,"fh":4,"team":5}`)
	var body StatusEventStartControlBody
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.ControllerHasAggro {
		t.Errorf("missing field should default to false")
	}
}

func TestStatusEventAggroChangedBody_RoundTrip(t *testing.T) {
	body := StatusEventAggroChangedBody{ControllerCharacterId: 7, ControllerHasAggro: true}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `{"controllerCharacterId":7,"controllerHasAggro":true}`
	if string(raw) != want {
		t.Errorf("got %s, want %s", string(raw), want)
	}
	var got StatusEventAggroChangedBody
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.ControllerCharacterId != 7 || !got.ControllerHasAggro {
		t.Errorf("round-trip mismatch: %+v", got)
	}
}

func TestEventStatusAggroChangedConstant(t *testing.T) {
	if EventStatusAggroChanged != "AGGRO_CHANGED" {
		t.Errorf("EventStatusAggroChanged=%q, want AGGRO_CHANGED", EventStatusAggroChanged)
	}
}
```

- [ ] **Step 2: Run, see fail**

```
cd services/atlas-channel/atlas.com/channel
go test ./kafka/message/monster -run 'StartControl|AggroChanged|EventStatusAggroChangedConstant' -v
```
Expected: FAIL — types and constant missing.

- [ ] **Step 3: Implement**

In `services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka.go`:

Add `EventStatusAggroChanged` to the const block:

```go
const (
	// ... existing ...
	EventStatusAggroChanged    = "AGGRO_CHANGED"
)
```

(Place near `EventStatusDamageReflected`.)

Modify `StatusEventStartControlBody`:

```go
type StatusEventStartControlBody struct {
	ActorId            uint32 `json:"actorId"`
	X                  int16  `json:"x"`
	Y                  int16  `json:"y"`
	Stance             byte   `json:"stance"`
	FH                 int16  `json:"fh"`
	Team               int8   `json:"team"`
	ControllerHasAggro bool   `json:"controllerHasAggro"`
}
```

Add the new body type:

```go
type StatusEventAggroChangedBody struct {
	ControllerCharacterId uint32 `json:"controllerCharacterId"`
	ControllerHasAggro    bool   `json:"controllerHasAggro"`
}
```

- [ ] **Step 4: Run, see pass**

```
go test ./kafka/message/monster -count=1
```
Expected: PASS.

- [ ] **Step 5: Commit**

```
git add services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka.go \
        services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka_test.go
git commit -m "feat(atlas-channel): add ControllerHasAggro to START_CONTROL body and AGGRO_CHANGED message types"
```

---

## Task 16 — atlas-channel: forward ControllerHasAggro through StartControlMonsterBody

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go`

- [ ] **Step 1: Replace hardcoded false**

In `services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go`, the `handleStatusEventStartControl` function currently has at line ~241:

```go
sf := session.Announce(l)(ctx)(wp)(monsterpkt.MonsterControlWriter)(writer.StartControlMonsterBody(m, false))
```

Replace `false` with `e.Body.ControllerHasAggro`:

```go
sf := session.Announce(l)(ctx)(wp)(monsterpkt.MonsterControlWriter)(writer.StartControlMonsterBody(m, e.Body.ControllerHasAggro))
```

- [ ] **Step 2: Verify**

```
cd services/atlas-channel/atlas.com/channel
go build ./...
go test ./...
```
Expected: PASS.

- [ ] **Step 3: Commit**

```
git add services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go
git commit -m "fix(atlas-channel): forward controllerHasAggro from START_CONTROL into StartControlMonsterBody"
```

---

## Task 17 — atlas-channel: handleStatusEventAggroChanged consumer

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go`

- [ ] **Step 1: Add the handler and register it**

Append a new handler function in the same file as the other `handleStatusEvent*` functions:

```go
func handleStatusEventAggroChanged(sc server.Model, wp writer.Producer) message.Handler[monster2.StatusEvent[monster2.StatusEventAggroChangedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e monster2.StatusEvent[monster2.StatusEventAggroChangedBody]) {
		if e.Type != monster2.EventStatusAggroChanged {
			return
		}
		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		m, err := monster.NewProcessor(l, ctx).GetById(e.UniqueId)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve monster [%d] for aggro change.", e.UniqueId)
			return
		}
		sf := session.Announce(l)(ctx)(wp)(monsterpkt.MonsterControlWriter)(writer.StartControlMonsterBody(m, e.Body.ControllerHasAggro))
		err = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.Body.ControllerCharacterId, sf)
		if err != nil {
			l.WithError(err).Errorf("Unable to refresh control state for monster [%d] for character [%d].", e.UniqueId, e.Body.ControllerCharacterId)
		}
	}
}
```

In `InitHandlers`, after the existing `handleStatusEventStopControl` registration, register the new handler:

```go
if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventAggroChanged(sc, wp)))); err != nil {
	return err
}
```

- [ ] **Step 2: Verify**

```
cd services/atlas-channel/atlas.com/channel
go build ./...
go test ./...
```
Expected: PASS.

- [ ] **Step 3: Commit**

```
git add services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go
git commit -m "feat(atlas-channel): consume AGGRO_CHANGED and re-send MonsterControlWriter to controller"
```

---

## Task 18 — atlas-channel: docs/kafka.md updates

**Files:**
- Modify: `services/atlas-channel/docs/kafka.md`

- [ ] **Step 1: Update**

Find the section describing the `START_CONTROL` consumer and update it to mention that `controllerHasAggro` is now read from the event body and passed to `StartControlMonsterBody` (which selects `ControlMonsterTypeActiveRequest` for `true` or `ControlMonsterTypeActiveInit` for `false`).

Add a new subsection describing the `AGGRO_CHANGED` consumer:

```markdown
#### AGGRO_CHANGED

`handleStatusEventAggroChanged` (kafka/consumer/monster/consumer.go) responds to atlas-monsters' `AGGRO_CHANGED` event. It loads the monster via `monster.NewProcessor(l, ctx).GetById(e.UniqueId)`, then re-sends `MonsterControlWriter` to the controller's session with the new aggro state via `writer.StartControlMonsterBody(m, e.Body.ControllerHasAggro)`. No `STOP_CONTROL` is emitted to the client — the active/passive control type carries the state change.
```

(Place this near where the `START_CONTROL` consumer is described.)

- [ ] **Step 2: Verify**

```
git diff services/atlas-channel/docs/kafka.md
```

Skim for accuracy.

- [ ] **Step 3: Commit**

```
git add services/atlas-channel/docs/kafka.md
git commit -m "docs(atlas-channel): document AGGRO_CHANGED consumer and START_CONTROL aggro forwarding"
```

---

## Task 19 — Final verification

- [ ] **Step 1: Run atlas-monsters build and tests**

```
cd services/atlas-monsters/atlas.com/monsters
go build ./...
go test ./... -count=1
```
Expected: PASS.

- [ ] **Step 2: Run atlas-channel build and tests**

```
cd services/atlas-channel/atlas.com/channel
go build ./...
go test ./... -count=1
```
Expected: PASS.

- [ ] **Step 3: Run go vet**

```
cd services/atlas-monsters/atlas.com/monsters && go vet ./...
cd services/atlas-channel/atlas.com/channel && go vet ./...
```
Expected: no warnings.

- [ ] **Step 4: Docker builds**

Locate the Dockerfiles:

```
find services/atlas-monsters services/atlas-channel -name 'Dockerfile' -maxdepth 4
```

Build each (paths may differ):

```
docker build -f <atlas-monsters-Dockerfile> services/atlas-monsters
docker build -f <atlas-channel-Dockerfile>  services/atlas-channel
```

Expected: success. If a Dockerfile uses a multi-stage build that runs `go test`, those will already be exercised by Step 1+2; verify the build stage succeeds.

- [ ] **Step 5: Final commit (only if any cleanup was needed)**

If everything passed in Steps 1-4, no commit is needed for this task. Otherwise commit fixes.

---

## Acceptance criteria coverage (cross-check vs. PRD §10)

| AC | Covered by task |
|---|---|
| Damage entries aggregated per attacker with `lastHitMs`; legacy migration | T1, T4 |
| Non-controller takes DPS lead → STOP_CONTROL + START_CONTROL with `controllerHasAggro: true` | T9, T10 |
| Current controller stays leader → no STOP/START | T9 |
| First damage flips `controllerHasAggro` and emits AGGRO_CHANGED (no controller change) | T9, T10 |
| Decay task runs every 1500ms with Cosmic schedule | T7, T12, T13 |
| Non-boss full-clear → STOP_CONTROL + `controllerHasAggro: false`; boss never triggers this | T7, T12 |
| atlas-channel `START_CONTROL` passes through `controllerHasAggro` | T15, T16 |
| atlas-channel `AGGRO_CHANGED` re-sends `MonsterControlWriter` | T15, T17 |
| Boss retains state until death (decay-task only) | T11, T12 |
| Reflect / `DamageSourceHeal` don't write damage entries | Already true; T6 verifies callers pass `nowMs` correctly |
| Existing tests updated for aggregated entries | T4 (TestConcurrentDamage), T5 |
| Docs updated | T14, T18 |
| Builds + tests pass | T19 |

---

## Notes for the executing subagent

- The `superpowers:test-driven-development` skill applies to every task. Always run the test before implementing — see it fail, then make it pass.
- Use `superpowers:verification-before-completion` before claiming a task complete: actually run the listed `go test` invocation and confirm output.
- Read `context.md` for the file/symbol cheat-sheet — it captures the existing seam patterns (`emitter`, `inFieldFn`, `bossLookupFn`, `nowFn`) so tests don't need to spin up Kafka or wall-clock dependencies.
- Stay in the test seam pattern throughout: prefer test-injectable function fields on processor/task structs over package-level globals.
- The `Damage` method change in T9 routes events through `p.emit`. Do NOT propagate this refactor to other processor methods (Create, executeHeal, ApplyStatusEffect, etc.) — out of scope.
- Keep the build green between commits. If a task introduces an API break, update all call sites in the same commit (Task 4 demonstrates this pattern).
