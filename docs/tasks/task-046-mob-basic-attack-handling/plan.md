# Mob Basic Attack Handling Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make v83 magic / ranged mobs (Samiho, Wraiths, Voodoos, Fire Boars) keep attacking after their first basic attack by implementing Cosmic's `Monster.canUseAttack` / `usedAttack` flow across atlas-data, atlas-monsters, and atlas-channel.

**Architecture:** atlas-data parses `attack{1,2,3}/info` (`conMP`, `attackAfter`) from mob WZ and exposes it on the JSON:API monster response. atlas-channel reads the cached attack metadata, optimistically forecasts post-decrement MP into `MonsterMovementAck`, and emits a `USE_BASIC_ATTACK` Kafka command on the existing monster command topic. atlas-monsters consumes the command, gates on a new `AttackCooldown` Redis registry + MP availability, and authoritatively decrements MP / registers the cooldown.

**Tech Stack:** Go, Redis (`miniredis` for tests), Kafka (existing `EnvCommandTopic`), JSON:API via `api2go/jsonapi`, XML WZ reader.

**Working directory note:** All paths are relative to the worktree root `<home>/source/atlas-ms/atlas/.worktrees/task-046-mob-basic-attack-handling`. All build / test commands assume `cd` into the relevant service directory (`services/atlas-data/atlas.com/data/`, `services/atlas-monsters/atlas.com/monsters/`, `services/atlas-channel/atlas.com/channel/`).

---

## Phase 1 — atlas-data: surface attack metadata

The bug-fix MVP needs `conMP` and `attackAfter` per attack slot exposed on
the mob REST response. This phase ends with a green
`go build ./... && go test ./...` in the atlas-data service.

### Task 1: Add `AttackInfo` type and `Attacks` field to atlas-data RestModel

**Files:**
- Modify: `services/atlas-data/atlas.com/data/monster/rest.go:5-43`
- Test: `services/atlas-data/atlas.com/data/monster/rest_test.go`

- [ ] **Step 1: Write the failing test**

Append the following to `services/atlas-data/atlas.com/data/monster/rest_test.go`:

```go
func TestRestModel_AttacksRoundTrip(t *testing.T) {
	in := RestModel{
		Id:   5100004,
		Name: "Samiho",
		Attacks: []AttackInfo{
			{Pos: 1, ConMP: 0, AttackAfter: 0},
			{Pos: 2, ConMP: 5, AttackAfter: 1500},
		},
	}
	data, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out RestModel
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(in.Attacks, out.Attacks) {
		t.Fatalf("Attacks round-trip mismatch:\n want %+v\n  got %+v", in.Attacks, out.Attacks)
	}
}
```

If the test file does not yet import `encoding/json`, `reflect`, or `testing`, add them.

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-data/atlas.com/data
go test ./monster/ -run TestRestModel_AttacksRoundTrip -v
```

Expected: FAIL — `AttackInfo` is not defined / `Attacks` field does not exist.

- [ ] **Step 3: Add `AttackInfo` and the `Attacks` field**

Edit `services/atlas-data/atlas.com/data/monster/rest.go`. After the `coolDamage` struct (currently the last `type` block before `LoseItemRestModel`), add:

```go
type AttackInfo struct {
	Pos         uint8 `json:"pos"`         // 1, 2, or 3 (matches WZ attackN naming)
	ConMP       int32 `json:"conMP"`
	AttackAfter int32 `json:"attackAfter"` // milliseconds
}
```

Then add a field to `RestModel` (insert right after `Skills []skill ...`):

```go
	Attacks            []AttackInfo      `json:"attacks"`
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./monster/ -run TestRestModel_AttacksRoundTrip -v
```

Expected: PASS.

- [ ] **Step 5: Run the full atlas-data test suite to confirm no regressions**

```bash
go test ./monster/ -v
```

Expected: PASS (all existing tests + the new one).

- [ ] **Step 6: Commit**

```bash
git add services/atlas-data/atlas.com/data/monster/rest.go services/atlas-data/atlas.com/data/monster/rest_test.go
git commit -m "feat(atlas-data): add AttackInfo to monster RestModel"
```

---

### Task 2: Parse `attack{1,2,3}/info` in atlas-data reader

**Files:**
- Modify: `services/atlas-data/atlas.com/data/monster/reader.go:90-101` (`Read` body, hook `getAttacks`)
- Modify: `services/atlas-data/atlas.com/data/monster/reader.go:end of file` (add `getAttacks`)
- Test: `services/atlas-data/atlas.com/data/monster/reader_test.go`

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-data/atlas.com/data/monster/reader_test.go`:

```go
const samihoAttackTestXML = `
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="5100004.img">
  <imgdir name="info">
    <int name="maxHP" value="3000"/>
    <int name="maxMP" value="100"/>
    <int name="level" value="50"/>
  </imgdir>
  <imgdir name="attack1">
    <canvas name="0" width="100" height="100">
      <int name="delay" value="120"/>
    </canvas>
  </imgdir>
  <imgdir name="attack2">
    <imgdir name="info">
      <int name="conMP" value="5"/>
      <int name="attackAfter" value="1500"/>
    </imgdir>
    <canvas name="0" width="100" height="100">
      <int name="delay" value="180"/>
    </canvas>
  </imgdir>
</imgdir>
`

const beetleAttackTestXML = `
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="7130003.img">
  <imgdir name="info">
    <int name="maxHP" value="500"/>
    <int name="maxMP" value="0"/>
    <int name="level" value="20"/>
  </imgdir>
  <imgdir name="attack1">
    <canvas name="0" width="100" height="100">
      <int name="delay" value="100"/>
    </canvas>
  </imgdir>
</imgdir>
`

func TestRead_ParsesAttacks_Samiho(t *testing.T) {
	tt := testTenant()
	l, _ := test.NewNullLogger()
	ctx := tenant.WithContext(context.Background(), tt)

	_, _ = GetMonsterStringRegistry().Add(tt, MonsterString{id: strconv.Itoa(5100004), name: "Samiho"})

	rm, err := Read(l)(ctx)(xml.FromByteArrayProvider([]byte(samihoAttackTestXML)))()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	if len(rm.Attacks) != 2 {
		t.Fatalf("Attacks length = %d, want 2 (attack1 + attack2): %+v", len(rm.Attacks), rm.Attacks)
	}
	want := map[uint8]AttackInfo{
		1: {Pos: 1, ConMP: 0, AttackAfter: 0},
		2: {Pos: 2, ConMP: 5, AttackAfter: 1500},
	}
	for _, a := range rm.Attacks {
		w, ok := want[a.Pos]
		if !ok {
			t.Errorf("unexpected pos %d", a.Pos)
			continue
		}
		if a != w {
			t.Errorf("pos %d: got %+v, want %+v", a.Pos, a, w)
		}
	}
}

func TestRead_ParsesAttacks_BeetleNoInfo(t *testing.T) {
	tt := testTenant()
	l, _ := test.NewNullLogger()
	ctx := tenant.WithContext(context.Background(), tt)

	_, _ = GetMonsterStringRegistry().Add(tt, MonsterString{id: strconv.Itoa(7130003), name: "Dual Beetle"})

	rm, err := Read(l)(ctx)(xml.FromByteArrayProvider([]byte(beetleAttackTestXML)))()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	// attack1 with no info subdirectory should NOT produce an entry.
	if len(rm.Attacks) != 0 {
		t.Fatalf("Attacks = %+v, want empty (no info subdirs)", rm.Attacks)
	}
}
```

If `testTenant()` does not exist in the test file, look in nearby `*_test.go` for the helper (it appears in `rest_test.go`); declare a test-local copy in `reader_test.go` if needed. The existing `testXML` test file already imports `tenant`, `xml`, and `test` (logrus hook) so most imports are present.

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd services/atlas-data/atlas.com/data
go test ./monster/ -run "TestRead_ParsesAttacks" -v
```

Expected: FAIL — `getAttacks` is not yet wired into `Read`, so `rm.Attacks` is nil/empty in both cases. (The Beetle case may pass coincidentally on len==0; the Samiho case will definitely fail.)

- [ ] **Step 3: Add `getAttacks` parser**

Append to `services/atlas-data/atlas.com/data/monster/reader.go`:

```go
// getAttacks parses attack{1,2,3}/info subnodes. Each `info` block contributes
// one AttackInfo. Slots without an `info` subdirectory are skipped — that
// matches melee mobs (Beetle) which only have animation frames under attackN.
func getAttacks(node xml.Node) []AttackInfo {
	results := make([]AttackInfo, 0)
	for pos := uint8(1); pos <= 3; pos++ {
		atk, err := node.ChildByName(fmt.Sprintf("attack%d", pos))
		if err != nil {
			continue
		}
		info, err := atk.ChildByName("info")
		if err != nil {
			continue
		}
		results = append(results, AttackInfo{
			Pos:         pos,
			ConMP:       info.GetIntegerWithDefault("conMP", 0),
			AttackAfter: info.GetIntegerWithDefault("attackAfter", 0),
		})
	}
	return results
}
```

- [ ] **Step 4: Hook `getAttacks` into `Read`**

In `services/atlas-data/atlas.com/data/monster/reader.go`, find the line:

```go
	m.AnimationTimes = getAnimationTimes(exml)
```

(currently at line 90). Add immediately after it:

```go
	m.Attacks = getAttacks(exml)
```

- [ ] **Step 5: Run the new tests to verify they pass**

```bash
go test ./monster/ -run "TestRead_ParsesAttacks" -v
```

Expected: PASS.

- [ ] **Step 6: Run the full atlas-data test suite**

```bash
go test ./...
```

Expected: PASS — including the existing `TestRest` round-trip (which now carries `Attacks: []` for Pianus, since Pianus' WZ has no `attack{N}/info` blocks).

- [ ] **Step 7: Build atlas-data**

```bash
go build ./...
```

Expected: clean build.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-data/atlas.com/data/monster/reader.go services/atlas-data/atlas.com/data/monster/reader_test.go
git commit -m "feat(atlas-data): parse attack{1,2,3}/info into RestModel.Attacks"
```

---

## Phase 2 — atlas-monsters: cooldown registry + UseBasicAttack

This phase implements the authoritative MP-decrement / cooldown side. By the end, atlas-monsters can consume `USE_BASIC_ATTACK` Kafka commands and apply them correctly.

### Task 3: Add `Attacks` to atlas-monsters `information.Model`

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/information/model.go`
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/information/rest.go`
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/information/builder.go`
- Test: `services/atlas-monsters/atlas.com/monsters/monster/information/rest_test.go`

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-monsters/atlas.com/monsters/monster/information/rest_test.go`:

```go
func TestExtract_PopulatesAttacks(t *testing.T) {
	rm := RestModel{
		Id:      "5100004",
		Hp:      3000,
		Mp:      100,
		Attacks: []AttackInfoRestModel{{Pos: 2, ConMP: 5, AttackAfter: 1500}},
	}
	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(m.Attacks()) != 1 {
		t.Fatalf("Attacks length = %d, want 1", len(m.Attacks()))
	}
	got := m.Attacks()[0]
	if got.Pos != 2 || got.ConMP != 5 || got.AttackAfter != 1500 {
		t.Fatalf("Attack[0] = %+v, want {Pos:2 ConMP:5 AttackAfter:1500}", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-monsters/atlas.com/monsters
go test ./monster/information/ -run TestExtract_PopulatesAttacks -v
```

Expected: FAIL — `AttackInfoRestModel` and `m.Attacks()` not defined.

- [ ] **Step 3: Add `AttackInfo` to `model.go`**

Edit `services/atlas-monsters/atlas.com/monsters/monster/information/model.go`. After the `Banish` struct, add:

```go
type AttackInfo struct {
	Pos         uint8
	ConMP       int32
	AttackAfter int32
}
```

Then add a field to `Model`:

```go
	attacks        []AttackInfo
```

(Insert right after `mpRecovery uint32`.)

Add a getter (place near `Skills()`):

```go
func (m Model) Attacks() []AttackInfo {
	return m.attacks
}
```

- [ ] **Step 4: Add `AttackInfoRestModel` and `Attacks` to `rest.go`**

Edit `services/atlas-monsters/atlas.com/monsters/monster/information/rest.go`. After the `banish` struct, add:

```go
type AttackInfoRestModel struct {
	Pos         uint8 `json:"pos"`
	ConMP       int32 `json:"conMP"`
	AttackAfter int32 `json:"attackAfter"`
}
```

Add a field to `RestModel` (right after `Skills []skill ...`):

```go
	Attacks            []AttackInfoRestModel `json:"attacks"`
```

Update `Extract` to translate:

```go
func Extract(rm RestModel) (Model, error) {
	skills := make([]Skill, 0, len(rm.Skills))
	for _, s := range rm.Skills {
		skills = append(skills, Skill{Id: s.Id, Level: s.Level})
	}
	attacks := make([]AttackInfo, 0, len(rm.Attacks))
	for _, a := range rm.Attacks {
		attacks = append(attacks, AttackInfo{Pos: a.Pos, ConMP: a.ConMP, AttackAfter: a.AttackAfter})
	}
	return Model{
		hp:             rm.Hp,
		mp:             rm.Mp,
		boss:           rm.Boss,
		undead:         rm.Undead,
		friendly:       rm.Friendly,
		weaponAttack:   rm.WeaponAttack,
		dropPeriod:     rm.DropPeriod,
		resistances:    rm.Resistances,
		animationTimes: rm.AnimationTimes,
		skills:         skills,
		attacks:        attacks,
		revives:        rm.Revives,
		banish:         Banish{Message: rm.Banish.Message, MapId: rm.Banish.MapId, PortalName: rm.Banish.PortalName},
		hpRecovery:     rm.HpRecovery,
		mpRecovery:     rm.MpRecovery,
	}, nil
}
```

- [ ] **Step 5: Add `SetAttacks` to the builder**

Edit `services/atlas-monsters/atlas.com/monsters/monster/information/builder.go`. Add a field to `ModelBuilder`:

```go
	attacks    []AttackInfo
```

Add a setter:

```go
// SetAttacks sets the attacks list on the builder.
func (b *ModelBuilder) SetAttacks(attacks []AttackInfo) *ModelBuilder {
	b.attacks = attacks
	return b
}
```

Update `Build()` to copy attacks:

```go
func (b *ModelBuilder) Build() Model {
	skills := b.skills
	if skills == nil {
		skills = []Skill{}
	}
	attacks := b.attacks
	if attacks == nil {
		attacks = []AttackInfo{}
	}
	return Model{
		skills:     skills,
		attacks:    attacks,
		hpRecovery: b.hpRecovery,
		mpRecovery: b.mpRecovery,
	}
}
```

- [ ] **Step 6: Run the test**

```bash
go test ./monster/information/ -run TestExtract_PopulatesAttacks -v
```

Expected: PASS.

- [ ] **Step 7: Run the information package's full tests**

```bash
go test ./monster/information/
```

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/information/
git commit -m "feat(atlas-monsters): plumb Attacks through information.Model"
```

---

### Task 4: Add `AttackCooldownRegistry`

**Files:**
- Create: `services/atlas-monsters/atlas.com/monsters/monster/attack_cooldown.go`
- Test: `services/atlas-monsters/atlas.com/monsters/monster/attack_cooldown_test.go`

- [ ] **Step 1: Write the failing test**

Create `services/atlas-monsters/atlas.com/monsters/monster/attack_cooldown_test.go`:

```go
package monster

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
)

func newTestAttackCooldownRegistry(t *testing.T) (*attackCooldownRegistry, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	return &attackCooldownRegistry{client: rc}, mr
}

func TestAttackCooldown_SetAndIsOnCooldown(t *testing.T) {
	r, mr := newTestAttackCooldownRegistry(t)
	defer mr.Close()
	ctx := context.Background()
	tm := newTestTenant(t)

	r.SetCooldown(ctx, tm, 100, uint8(1), 1500*time.Millisecond)
	if !r.IsOnCooldown(ctx, tm, 100, uint8(1)) {
		t.Fatalf("expected on cooldown for pos 1")
	}
	if r.IsOnCooldown(ctx, tm, 100, uint8(2)) {
		t.Fatalf("did not expect cooldown for pos 2")
	}
}

func TestAttackCooldown_DistinctFromSkillRegistry(t *testing.T) {
	// Sanity: same uniqueId, attack pos 0 must not collide with skill 0
	// in the OTHER registry (different key prefix). This is a simple
	// smoke test asserting different key namespaces.
	r, mr := newTestAttackCooldownRegistry(t)
	defer mr.Close()
	ctx := context.Background()
	tm := newTestTenant(t)

	r.SetCooldown(ctx, tm, 100, uint8(0), 1*time.Second)
	keys := mr.Keys()
	for _, k := range keys {
		if k == "atlas:monster-cooldown:"+tm.Id().String()+":100:0" {
			t.Fatalf("attack-cooldown key collides with skill-cooldown key namespace: %s", k)
		}
	}
}

func TestAttackCooldown_ClearAll(t *testing.T) {
	r, mr := newTestAttackCooldownRegistry(t)
	defer mr.Close()
	ctx := context.Background()
	tm := newTestTenant(t)

	r.SetCooldown(ctx, tm, 100, uint8(0), time.Minute)
	r.SetCooldown(ctx, tm, 100, uint8(1), time.Minute)
	r.SetCooldown(ctx, tm, 100, uint8(2), time.Minute)
	r.ClearCooldowns(ctx, tm, 100)

	if r.IsOnCooldown(ctx, tm, 100, uint8(0)) ||
		r.IsOnCooldown(ctx, tm, 100, uint8(1)) ||
		r.IsOnCooldown(ctx, tm, 100, uint8(2)) {
		t.Fatalf("expected all cleared")
	}
}

func TestAttackCooldown_ZeroDurationDoesNotPersist(t *testing.T) {
	r, mr := newTestAttackCooldownRegistry(t)
	defer mr.Close()
	ctx := context.Background()
	tm := newTestTenant(t)

	r.SetCooldown(ctx, tm, 100, uint8(0), 0)
	if r.IsOnCooldown(ctx, tm, 100, uint8(0)) {
		t.Fatalf("zero-duration cooldown must not register")
	}
}
```

`newTestTenant` is already defined in `cooldown_test.go` in the same package, so we can reuse it.

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./monster/ -run "TestAttackCooldown" -v
```

Expected: FAIL — `attackCooldownRegistry` not defined.

- [ ] **Step 3: Implement the registry**

Create `services/atlas-monsters/atlas.com/monsters/monster/attack_cooldown.go`:

```go
package monster

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

type attackCooldownRegistry struct {
	client *goredis.Client
}

var attackCooldownReg *attackCooldownRegistry
var attackCooldownOnce sync.Once

func InitAttackCooldownRegistry(rc *goredis.Client) {
	attackCooldownOnce.Do(func() {
		attackCooldownReg = &attackCooldownRegistry{client: rc}
	})
}

func GetAttackCooldownRegistry() *attackCooldownRegistry {
	return attackCooldownReg
}

func attackCooldownKey(t tenant.Model, monsterId uint32, attackPos uint8) string {
	return fmt.Sprintf("atlas:monster-attack-cooldown:%s:%s:%s",
		t.Id().String(),
		strconv.FormatUint(uint64(monsterId), 10),
		strconv.FormatUint(uint64(attackPos), 10),
	)
}

func attackCooldownScanPattern(t tenant.Model, monsterId uint32) string {
	return fmt.Sprintf("atlas:monster-attack-cooldown:%s:%s:*",
		t.Id().String(),
		strconv.FormatUint(uint64(monsterId), 10),
	)
}

func (r *attackCooldownRegistry) IsOnCooldown(ctx context.Context, t tenant.Model, monsterId uint32, attackPos uint8) bool {
	key := attackCooldownKey(t, monsterId, attackPos)
	result, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false
	}
	return result > 0
}

// SetCooldown registers a cooldown for the given (monsterId, attackPos) with
// Redis-managed TTL. A zero duration is a no-op (matches melee attacks
// where attackAfter == 0).
func (r *attackCooldownRegistry) SetCooldown(ctx context.Context, t tenant.Model, monsterId uint32, attackPos uint8, duration time.Duration) {
	if duration <= 0 {
		return
	}
	key := attackCooldownKey(t, monsterId, attackPos)
	expiryMs := time.Now().Add(duration).UnixMilli()
	r.client.Set(ctx, key, strconv.FormatInt(expiryMs, 10), duration)
}

func (r *attackCooldownRegistry) ClearCooldowns(ctx context.Context, t tenant.Model, monsterId uint32) {
	pattern := attackCooldownScanPattern(t, monsterId)
	var cursor uint64
	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return
		}
		if len(keys) > 0 {
			r.client.Del(ctx, keys...)
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
}
```

- [ ] **Step 4: Run the new tests to verify they pass**

```bash
go test ./monster/ -run "TestAttackCooldown" -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/attack_cooldown.go services/atlas-monsters/atlas.com/monsters/monster/attack_cooldown_test.go
git commit -m "feat(atlas-monsters): add AttackCooldownRegistry"
```

---

### Task 5: Wire `InitAttackCooldownRegistry` and clear cooldowns on monster lifecycle events

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/main.go:50` (add init)
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/processor.go:339,446,970` (clear on kill / friendly-kill / destroy)

- [ ] **Step 1: Wire the registry init in `main.go`**

Edit `services/atlas-monsters/atlas.com/monsters/main.go`. Find:

```go
	monster.InitCooldownRegistry(rc)
```

Add immediately after:

```go
	monster.InitAttackCooldownRegistry(rc)
```

- [ ] **Step 2: Clear attack cooldowns in `Damage` kill path**

Edit `services/atlas-monsters/atlas.com/monsters/monster/processor.go`. Find the line in `Damage` (around line 339):

```go
		// Clear cooldowns and drop timer on death
		GetCooldownRegistry().ClearCooldowns(p.ctx, p.t, id)
		GetDropTimerRegistry().Unregister(p.ctx, p.t, id)
```

Insert between the two:

```go
		GetAttackCooldownRegistry().ClearCooldowns(p.ctx, p.t, id)
```

- [ ] **Step 3: Clear attack cooldowns in `DamageFriendly` kill path**

In the same file, find the `if s.Killed {` block in `DamageFriendly` (around line 446):

```go
	if s.Killed {
		GetCooldownRegistry().ClearCooldowns(p.ctx, p.t, uniqueId)
		GetDropTimerRegistry().Unregister(p.ctx, p.t, uniqueId)
```

Insert:

```go
		GetAttackCooldownRegistry().ClearCooldowns(p.ctx, p.t, uniqueId)
```

- [ ] **Step 4: Clear attack cooldowns in `Destroy`**

Find `Destroy` (around line 969):

```go
func (p *ProcessorImpl) Destroy(uniqueId uint32) error {
	GetDropTimerRegistry().Unregister(p.ctx, p.t, uniqueId)
```

Insert immediately after the first line of the function body:

```go
	GetAttackCooldownRegistry().ClearCooldowns(p.ctx, p.t, uniqueId)
```

(Place it after `Unregister` for consistency with `Damage`'s ordering.)

- [ ] **Step 5: Build atlas-monsters**

```bash
cd services/atlas-monsters/atlas.com/monsters
go build ./...
```

Expected: clean build.

- [ ] **Step 6: Run existing tests to verify no regressions**

```bash
go test ./...
```

Expected: PASS. Existing `processor_test.go` tests instantiate `ProcessorImpl` directly with a stub emitter and don't touch `GetAttackCooldownRegistry` (which would be `nil` since `InitAttackCooldownRegistry` is not called in tests). The `Destroy` and kill paths in `processor_test.go` go through `Damage` / `Destroy`. **Risk:** if any existing test exercises `Damage` to a kill or `Destroy`, it will hit a nil-pointer panic on `GetAttackCooldownRegistry().ClearCooldowns(...)`.

If tests fail with nil-pointer panic in `attack_cooldown.go`, fix it by guarding `GetAttackCooldownRegistry()`:

```go
func (r *attackCooldownRegistry) ClearCooldowns(ctx context.Context, t tenant.Model, monsterId uint32) {
	if r == nil {
		return
	}
	// ... rest unchanged
}
```

Apply the same `r == nil` guard at the top of `IsOnCooldown` and `SetCooldown` so production code that forgot to call `InitAttackCooldownRegistry` (or test code) degrades to a noop instead of panicking. This mirrors the implicit safety the existing `cooldownRegistry` enjoys (since tests always call `InitCooldownRegistry`).

After fixing if needed, rerun:

```bash
go test ./...
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/main.go services/atlas-monsters/atlas.com/monsters/monster/processor.go services/atlas-monsters/atlas.com/monsters/monster/attack_cooldown.go
git commit -m "feat(atlas-monsters): wire AttackCooldownRegistry init + lifecycle cleanup"
```

---

### Task 6: Add `UseBasicAttack` to atlas-monsters Processor

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/processor.go` (interface + impl)
- Test: `services/atlas-monsters/atlas.com/monsters/monster/processor_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `services/atlas-monsters/atlas.com/monsters/monster/processor_test.go`:

```go
// stubInformationLookup forces processor.UseBasicAttack to use a fixed Model
// instead of going through information.GetById (which would hit atlas-data
// over HTTP). The processor uses information.GetById directly, so we accept
// this is integration-shaped: in this test we lean on builder + a thin
// override hook. If the existing processor test suite has a similar
// pattern, follow it; otherwise this test is structured around the
// behaviors we can drive purely via the registry + builder.
//
// For UseBasicAttack we test the gates (cooldown, MP, dead, missing info)
// by constructing pre-state in the registry and the cooldown registry and
// asserting post-state. Since UseBasicAttack calls information.GetById,
// we accept that the test environment must wire a stub. The simplest
// stub: declare a package-level testHook var read by UseBasicAttack when
// non-nil. See implementation step.

func TestUseBasicAttack_HappyPath_DeductsMpAndRegistersCooldown(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	// Wire test-only attack-cooldown registry
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer mr.Close()
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	prevAttackReg := attackCooldownReg
	attackCooldownReg = &attackCooldownRegistry{client: rc}
	defer func() { attackCooldownReg = prevAttackReg }()

	prevHook := testInformationLookup
	testInformationLookup = func(monsterId uint32) (information.Model, error) {
		return information.NewModelBuilder().
			SetAttacks([]information.AttackInfo{{Pos: 2, ConMP: 5, AttackAfter: 1500}}).
			Build(), nil
	}
	defer func() { testInformationLookup = prevHook }()

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	monsterId := uint32(5100004)
	m := r.CreateMonster(ctx, ten, f, monsterId, 0, 0, 0, 5, 0, 3000, 100)
	uniqueId := m.UniqueId()

	p := &ProcessorImpl{l: logrus.New(), ctx: tenant.WithContext(ctx, ten), t: ten}

	// pos=2 corresponds to AttackInfo.Pos=1 internally? No — we normalize
	// the wire/zero-indexed attackPos to the 1-indexed information.Pos by
	// adding 1 inside UseBasicAttack. Caller passes 0-indexed attackPos.
	p.UseBasicAttack(uniqueId, uint8(1)) // 0-indexed, matches Pos=2 (1+1)

	got, err := r.GetMonster(ten, uniqueId)
	if err != nil {
		t.Fatalf("GetMonster: %v", err)
	}
	if got.Mp() != 95 {
		t.Errorf("Mp after UseBasicAttack = %d, want 95 (100-5)", got.Mp())
	}
	if !attackCooldownReg.IsOnCooldown(ctx, ten, uniqueId, uint8(1)) {
		t.Errorf("expected attack pos 1 to be on cooldown after happy-path UseBasicAttack")
	}
}

func TestUseBasicAttack_OnCooldown_Skips(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	mr, _ := miniredis.Run()
	defer mr.Close()
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	prevAttackReg := attackCooldownReg
	attackCooldownReg = &attackCooldownRegistry{client: rc}
	defer func() { attackCooldownReg = prevAttackReg }()

	prevHook := testInformationLookup
	testInformationLookup = func(monsterId uint32) (information.Model, error) {
		return information.NewModelBuilder().
			SetAttacks([]information.AttackInfo{{Pos: 2, ConMP: 5, AttackAfter: 1500}}).
			Build(), nil
	}
	defer func() { testInformationLookup = prevHook }()

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 5100004, 0, 0, 0, 5, 0, 3000, 100)
	uniqueId := m.UniqueId()

	// Pre-register a cooldown
	attackCooldownReg.SetCooldown(ctx, ten, uniqueId, uint8(1), time.Second)

	p := &ProcessorImpl{l: logrus.New(), ctx: tenant.WithContext(ctx, ten), t: ten}
	p.UseBasicAttack(uniqueId, uint8(1))

	got, _ := r.GetMonster(ten, uniqueId)
	if got.Mp() != 100 {
		t.Errorf("Mp after on-cooldown UseBasicAttack = %d, want 100 (untouched)", got.Mp())
	}
}

func TestUseBasicAttack_InsufficientMp_Skips(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	mr, _ := miniredis.Run()
	defer mr.Close()
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	prevAttackReg := attackCooldownReg
	attackCooldownReg = &attackCooldownRegistry{client: rc}
	defer func() { attackCooldownReg = prevAttackReg }()

	prevHook := testInformationLookup
	testInformationLookup = func(monsterId uint32) (information.Model, error) {
		return information.NewModelBuilder().
			SetAttacks([]information.AttackInfo{{Pos: 2, ConMP: 50, AttackAfter: 1500}}).
			Build(), nil
	}
	defer func() { testInformationLookup = prevHook }()

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 5100004, 0, 0, 0, 5, 0, 3000, 10)
	uniqueId := m.UniqueId()

	p := &ProcessorImpl{l: logrus.New(), ctx: tenant.WithContext(ctx, ten), t: ten}
	p.UseBasicAttack(uniqueId, uint8(1))

	got, _ := r.GetMonster(ten, uniqueId)
	if got.Mp() != 10 {
		t.Errorf("Mp after insufficient-mp UseBasicAttack = %d, want 10 (untouched)", got.Mp())
	}
	if attackCooldownReg.IsOnCooldown(ctx, ten, uniqueId, uint8(1)) {
		t.Errorf("did not expect cooldown after insufficient-mp reject")
	}
}

func TestUseBasicAttack_NoAttackInfo_Skips(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	mr, _ := miniredis.Run()
	defer mr.Close()
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	prevAttackReg := attackCooldownReg
	attackCooldownReg = &attackCooldownRegistry{client: rc}
	defer func() { attackCooldownReg = prevAttackReg }()

	prevHook := testInformationLookup
	testInformationLookup = func(monsterId uint32) (information.Model, error) {
		// Beetle: no attacks at all.
		return information.NewModelBuilder().Build(), nil
	}
	defer func() { testInformationLookup = prevHook }()

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 7130003, 0, 0, 0, 5, 0, 500, 0)
	uniqueId := m.UniqueId()

	p := &ProcessorImpl{l: logrus.New(), ctx: tenant.WithContext(ctx, ten), t: ten}
	p.UseBasicAttack(uniqueId, uint8(0))

	if attackCooldownReg.IsOnCooldown(ctx, ten, uniqueId, uint8(0)) {
		t.Errorf("did not expect cooldown when monster has no attack info")
	}
}

func TestUseBasicAttack_ZeroConMpAndZeroAttackAfter_NoOp(t *testing.T) {
	// Melee parity: pos exists but conMP=0 and attackAfter=0 → no MP
	// decrement, no cooldown register, but also no error and no skip log.
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	mr, _ := miniredis.Run()
	defer mr.Close()
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	prevAttackReg := attackCooldownReg
	attackCooldownReg = &attackCooldownRegistry{client: rc}
	defer func() { attackCooldownReg = prevAttackReg }()

	prevHook := testInformationLookup
	testInformationLookup = func(monsterId uint32) (information.Model, error) {
		return information.NewModelBuilder().
			SetAttacks([]information.AttackInfo{{Pos: 1, ConMP: 0, AttackAfter: 0}}).
			Build(), nil
	}
	defer func() { testInformationLookup = prevHook }()

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 6090003, 0, 0, 0, 5, 0, 500, 0)
	uniqueId := m.UniqueId()

	p := &ProcessorImpl{l: logrus.New(), ctx: tenant.WithContext(ctx, ten), t: ten}
	p.UseBasicAttack(uniqueId, uint8(0)) // 0-indexed; Pos=1 = (0+1)

	got, _ := r.GetMonster(ten, uniqueId)
	if got.Mp() != 0 {
		t.Errorf("Mp = %d, want 0 (untouched)", got.Mp())
	}
	if attackCooldownReg.IsOnCooldown(ctx, ten, uniqueId, uint8(0)) {
		t.Errorf("did not expect cooldown for zero-attackAfter")
	}
}

func TestUseBasicAttack_DeadMonster_Skips(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	mr, _ := miniredis.Run()
	defer mr.Close()
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	prevAttackReg := attackCooldownReg
	attackCooldownReg = &attackCooldownRegistry{client: rc}
	defer func() { attackCooldownReg = prevAttackReg }()

	prevHook := testInformationLookup
	testInformationLookup = func(monsterId uint32) (information.Model, error) {
		return information.NewModelBuilder().
			SetAttacks([]information.AttackInfo{{Pos: 2, ConMP: 5, AttackAfter: 1500}}).
			Build(), nil
	}
	defer func() { testInformationLookup = prevHook }()

	p := &ProcessorImpl{l: logrus.New(), ctx: tenant.WithContext(ctx, ten), t: ten}
	// Use a uniqueId that doesn't exist in the registry. The path:
	// GetMonster → not found → return.
	p.UseBasicAttack(uint32(99999), uint8(1))

	if attackCooldownReg.IsOnCooldown(ctx, ten, uint32(99999), uint8(1)) {
		t.Errorf("did not expect cooldown for missing monster")
	}
}
```

Required additional imports for these tests: `"atlas-monsters/monster/information"`, `"github.com/alicebob/miniredis/v2"`, `goredis "github.com/redis/go-redis/v9"`, `"time"`. Add any that are not already imported by the file.

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./monster/ -run "TestUseBasicAttack" -v
```

Expected: FAIL — `UseBasicAttack`, `testInformationLookup`, and `information.AttackInfo` references all unresolved.

- [ ] **Step 3: Add the test hook + the `UseBasicAttack` method**

Edit `services/atlas-monsters/atlas.com/monsters/monster/processor.go`. Add to the `Processor` interface (right after `UseSkillGM`):

```go
	UseBasicAttack(uniqueId uint32, attackPos uint8)
```

Add a package-level variable near the top of the file (after the `emitter` type definition is fine):

```go
// testInformationLookup is a test-only override for information.GetById. When
// nil (production), UseBasicAttack calls information.GetById normally.
var testInformationLookup func(monsterId uint32) (information.Model, error)
```

Add the method (after `UseSkillGM`, before `MistDurationCapMs`):

```go
// UseBasicAttack authoritatively applies the post-conditions of a basic
// monster attack: MP decrement and cooldown registration. It is invoked
// asynchronously via Kafka after atlas-channel has already optimistically
// projected the post-decrement MP into the move ack. Every reject path
// returns silently — there is nothing to communicate back.
func (p *ProcessorImpl) UseBasicAttack(uniqueId uint32, attackPos uint8) {
	m, err := GetMonsterRegistry().GetMonster(p.t, uniqueId)
	if err != nil {
		p.l.Debugf("UseBasicAttack: monster [%d] not found.", uniqueId)
		return
	}
	if !m.Alive() {
		p.l.Debugf("UseBasicAttack: monster [%d] not alive.", uniqueId)
		return
	}

	// Look up template attack metadata. The hook lets tests inject a
	// canned response without spinning up an HTTP fake.
	var info information.Model
	if testInformationLookup != nil {
		info, err = testInformationLookup(m.MonsterId())
	} else {
		info, err = information.GetById(p.l)(p.ctx)(m.MonsterId())
	}
	if err != nil {
		p.l.WithError(err).Debugf("UseBasicAttack: cannot fetch template for monster [%d].", uniqueId)
		return
	}

	// pos in information.AttackInfo is 1-indexed; the wire/registry
	// attackPos is 0-indexed. Convert.
	wantPos := attackPos + 1
	var atk information.AttackInfo
	found := false
	for _, a := range info.Attacks() {
		if a.Pos == wantPos {
			atk = a
			found = true
			break
		}
	}
	if !found {
		p.l.Debugf("UseBasicAttack: monster [%d] has no attack info for pos %d.", uniqueId, attackPos)
		return
	}

	if GetAttackCooldownRegistry().IsOnCooldown(p.ctx, p.t, uniqueId, attackPos) {
		p.l.Debugf("UseBasicAttack: monster [%d] attack pos %d on cooldown.", uniqueId, attackPos)
		return
	}

	if atk.ConMP > 0 && uint32(m.Mp()) < uint32(atk.ConMP) {
		p.l.Debugf("UseBasicAttack: monster [%d] insufficient MP [%d] for pos %d cost [%d].", uniqueId, m.Mp(), attackPos, atk.ConMP)
		return
	}

	if atk.ConMP > 0 {
		if _, err := GetMonsterRegistry().DeductMp(p.t, uniqueId, uint16(atk.ConMP)); err != nil {
			p.l.WithError(err).Errorf("UseBasicAttack: DeductMp failed for monster [%d].", uniqueId)
			return
		}
	}

	if atk.AttackAfter > 0 {
		GetAttackCooldownRegistry().SetCooldown(p.ctx, p.t, uniqueId, attackPos, time.Duration(atk.AttackAfter)*time.Millisecond)
	}
}
```

(`information` is already imported; `time` is already imported.)

- [ ] **Step 4: Run the tests**

```bash
go test ./monster/ -run "TestUseBasicAttack" -v
```

Expected: PASS.

- [ ] **Step 5: Run the full atlas-monsters monster package tests**

```bash
go test ./monster/
```

Expected: PASS.

- [ ] **Step 6: Build atlas-monsters**

```bash
go build ./...
```

Expected: clean build.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/processor.go services/atlas-monsters/atlas.com/monsters/monster/processor_test.go
git commit -m "feat(atlas-monsters): add UseBasicAttack with MP+cooldown gates"
```

---

### Task 7: Add `USE_BASIC_ATTACK` Kafka command + handler in atlas-monsters

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka.go`
- Modify: `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/consumer.go`
- Test: `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka_test.go`

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka_test.go`:

```go
func TestUseBasicAttackCommandBody_Decode(t *testing.T) {
	raw := []byte(`{"attackPos":1}`)
	var body useBasicAttackCommandBody
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.AttackPos != 1 {
		t.Fatalf("AttackPos = %d, want 1", body.AttackPos)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-monsters/atlas.com/monsters
go test ./kafka/consumer/monster/ -run TestUseBasicAttackCommandBody_Decode -v
```

Expected: FAIL — `useBasicAttackCommandBody` undefined.

- [ ] **Step 3: Add the command-type constant + body**

Edit `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka.go`. Add to the `const` block:

```go
	CommandTypeUseBasicAttack    = "USE_BASIC_ATTACK"
```

(Place it next to `CommandTypeUseSkill`.)

Add a new body type after `useSkillFieldCommandBody`:

```go
type useBasicAttackCommandBody struct {
	AttackPos uint8 `json:"attackPos"`
}
```

- [ ] **Step 4: Add the handler**

Edit `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/consumer.go`. After `handleUseSkillCommand` (line 127):

```go
func handleUseBasicAttackCommand(l logrus.FieldLogger, ctx context.Context, c command[useBasicAttackCommandBody]) {
	if c.Type != CommandTypeUseBasicAttack {
		return
	}

	p := monster.NewProcessor(l, ctx)
	p.UseBasicAttack(c.MonsterId, c.Body.AttackPos)
}
```

Register the handler in `InitHandlers`. Find the block (around line 43):

```go
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleUseSkillCommand))); err != nil {
			return err
		}
```

Add immediately after:

```go
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleUseBasicAttackCommand))); err != nil {
			return err
		}
```

- [ ] **Step 5: Run the unit test**

```bash
go test ./kafka/consumer/monster/ -run TestUseBasicAttackCommandBody_Decode -v
```

Expected: PASS.

- [ ] **Step 6: Run the kafka consumer package tests**

```bash
go test ./kafka/consumer/monster/
```

Expected: PASS.

- [ ] **Step 7: Build atlas-monsters**

```bash
go build ./...
```

Expected: clean build.

- [ ] **Step 8: Run all atlas-monsters tests**

```bash
go test ./...
```

Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/
git commit -m "feat(atlas-monsters): add USE_BASIC_ATTACK Kafka consumer"
```

---

## Phase 3 — atlas-channel: optimistic ack + Kafka command emission

This phase reads atlas-data attack metadata, forecasts the decremented MP into `MonsterMovementAck`, and emits the `USE_BASIC_ATTACK` Kafka command back to atlas-monsters.

### Task 8: Create `monster/information` package in atlas-channel

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/monster/information/model.go`
- Create: `services/atlas-channel/atlas.com/channel/monster/information/rest.go`
- Create: `services/atlas-channel/atlas.com/channel/monster/information/requests.go`
- Create: `services/atlas-channel/atlas.com/channel/monster/information/processor.go`
- Create: `services/atlas-channel/atlas.com/channel/monster/information/builder.go`
- Test: `services/atlas-channel/atlas.com/channel/monster/information/rest_test.go`

This package proxies atlas-data's monster template endpoint, exposing the
`Attacks` slice that atlas-channel needs to compute the optimistic ack MP.
Mirrors the shape of
`services/atlas-monsters/atlas.com/monsters/monster/information/` but
calls `RootUrl("DATA")` instead of `RootUrl("MONSTERS")`.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/monster/information/rest_test.go`:

```go
package information

import "testing"

func TestExtract_PopulatesAttacks(t *testing.T) {
	rm := RestModel{
		Id: "5100004",
		Attacks: []AttackInfoRestModel{
			{Pos: 2, ConMP: 5, AttackAfter: 1500},
		},
	}
	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(m.Attacks()) != 1 {
		t.Fatalf("Attacks = %d, want 1", len(m.Attacks()))
	}
	if m.Attacks()[0].Pos != 2 || m.Attacks()[0].ConMP != 5 || m.Attacks()[0].AttackAfter != 1500 {
		t.Fatalf("Attack[0] = %+v", m.Attacks()[0])
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-channel/atlas.com/channel
go test ./monster/information/ -v
```

Expected: FAIL — package does not yet exist.

- [ ] **Step 3: Create `model.go`**

Create `services/atlas-channel/atlas.com/channel/monster/information/model.go`:

```go
package information

type AttackInfo struct {
	Pos         uint8
	ConMP       int32
	AttackAfter int32
}

type Model struct {
	monsterId uint32
	attacks   []AttackInfo
}

func (m Model) MonsterId() uint32 {
	return m.monsterId
}

func (m Model) Attacks() []AttackInfo {
	return m.attacks
}
```

- [ ] **Step 4: Create `rest.go`**

Create `services/atlas-channel/atlas.com/channel/monster/information/rest.go`:

```go
package information

import "strconv"

type RestModel struct {
	Id      string                `json:"-"`
	Attacks []AttackInfoRestModel `json:"attacks"`
}

type AttackInfoRestModel struct {
	Pos         uint8 `json:"pos"`
	ConMP       int32 `json:"conMP"`
	AttackAfter int32 `json:"attackAfter"`
}

func (r RestModel) GetName() string {
	return "monsters"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

func Extract(rm RestModel) (Model, error) {
	id, err := strconv.ParseUint(rm.Id, 10, 32)
	if err != nil {
		// id may be empty in tests; tolerate.
		id = 0
	}
	attacks := make([]AttackInfo, 0, len(rm.Attacks))
	for _, a := range rm.Attacks {
		attacks = append(attacks, AttackInfo{
			Pos:         a.Pos,
			ConMP:       a.ConMP,
			AttackAfter: a.AttackAfter,
		})
	}
	return Model{
		monsterId: uint32(id),
		attacks:   attacks,
	}, nil
}
```

- [ ] **Step 5: Create `requests.go`**

Create `services/atlas-channel/atlas.com/channel/monster/information/requests.go`:

```go
package information

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	monsterResource = "data/monsters/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestById(monsterId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+monsterResource, monsterId))
}
```

- [ ] **Step 6: Create `processor.go`**

Create `services/atlas-channel/atlas.com/channel/monster/information/processor.go`:

```go
package information

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	return &Processor{l: l, ctx: ctx}
}

func (p *Processor) GetById(monsterId uint32) (Model, error) {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(monsterId), Extract)()
}
```

- [ ] **Step 7: Create `builder.go`**

Create `services/atlas-channel/atlas.com/channel/monster/information/builder.go`:

```go
package information

// ModelBuilder builds Model instances for tests.
type ModelBuilder struct {
	monsterId uint32
	attacks   []AttackInfo
}

func NewModelBuilder() *ModelBuilder {
	return &ModelBuilder{}
}

func (b *ModelBuilder) SetMonsterId(id uint32) *ModelBuilder {
	b.monsterId = id
	return b
}

func (b *ModelBuilder) SetAttacks(attacks []AttackInfo) *ModelBuilder {
	b.attacks = attacks
	return b
}

func (b *ModelBuilder) Build() Model {
	attacks := b.attacks
	if attacks == nil {
		attacks = []AttackInfo{}
	}
	return Model{
		monsterId: b.monsterId,
		attacks:   attacks,
	}
}
```

- [ ] **Step 8: Run the test**

```bash
go test ./monster/information/ -v
```

Expected: PASS.

- [ ] **Step 9: Build atlas-channel**

```bash
go build ./...
```

Expected: clean build.

- [ ] **Step 10: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/monster/information/
git commit -m "feat(atlas-channel): add monster/information package proxying atlas-data"
```

---

### Task 9: Add basic-attack classification helper in atlas-channel

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/movement/action.go`
- Test: `services/atlas-channel/atlas.com/channel/movement/action_test.go`

- [ ] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/movement/action_test.go`:

```go
package movement

import "testing"

func TestBasicAttackPos_OutOfRange(t *testing.T) {
	cases := []int8{-1, 0, 23, 42, 60, 100}
	for _, c := range cases {
		if pos, ok := basicAttackPos(c); ok {
			t.Errorf("basicAttackPos(%d) = (%d, true), want (_, false)", c, pos)
		}
	}
}

func TestBasicAttackPos_InRange(t *testing.T) {
	cases := map[int8]uint8{
		24: 0, 25: 0,
		26: 1, 27: 1,
		28: 2, 29: 2,
		40: 8, 41: 8,
	}
	for raw, want := range cases {
		got, ok := basicAttackPos(raw)
		if !ok {
			t.Errorf("basicAttackPos(%d) = (_, false), want (%d, true)", raw, want)
			continue
		}
		if got != want {
			t.Errorf("basicAttackPos(%d) = %d, want %d", raw, got, want)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-channel/atlas.com/channel
go test ./movement/ -run "TestBasicAttackPos" -v
```

Expected: FAIL — `basicAttackPos` undefined.

- [ ] **Step 3: Create the helper**

Create `services/atlas-channel/atlas.com/channel/movement/action.go`:

```go
package movement

// basicAttackRangeLo / basicAttackRangeHi are the inclusive bounds for a
// basic mob attack action. The classification is taken from Cosmic v83's
// MoveLifeHandler.java:108 — values outside this band are not basic attacks
// (they may be movement, stand, hit, fall, or — for [42, 59] — a named
// skill, which atlas-channel handles via the existing skill-id branch).
const (
	basicAttackRangeLo int8 = 24
	basicAttackRangeHi int8 = 41
)

// basicAttackPos returns the 0-indexed attack-position derived from the
// inbound MoveLife.nActionAndDir byte, or false when the byte is outside
// the basic-attack band.
func basicAttackPos(rawActionAndDir int8) (uint8, bool) {
	if rawActionAndDir < basicAttackRangeLo || rawActionAndDir > basicAttackRangeHi {
		return 0, false
	}
	return uint8((rawActionAndDir - basicAttackRangeLo) / 2), true
}
```

- [ ] **Step 4: Run the test**

```bash
go test ./movement/ -run "TestBasicAttackPos" -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/movement/action.go services/atlas-channel/atlas.com/channel/movement/action_test.go
git commit -m "feat(atlas-channel): add basicAttackPos classification helper"
```

---

### Task 10: Add `USE_BASIC_ATTACK` Kafka producer + processor method in atlas-channel

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka.go`
- Modify: `services/atlas-channel/atlas.com/channel/monster/producer.go`
- Modify: `services/atlas-channel/atlas.com/channel/monster/processor.go`
- Test: `services/atlas-channel/atlas.com/channel/monster/producer_test.go`

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-channel/atlas.com/channel/monster/producer_test.go` (create the file if it does not exist; check first):

```go
package monster

import (
	"encoding/json"
	"testing"

	monster2 "atlas-channel/kafka/message/monster"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

func TestUseBasicAttackCommandProvider(t *testing.T) {
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).SetInstance(uuid.Nil).Build()
	prov := UseBasicAttackCommandProvider(f, uint32(5001), uint8(1))
	msgs, err := prov()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("messages = %d, want 1", len(msgs))
	}
	var cmd monster2.Command[monster2.UseBasicAttackCommandBody]
	if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cmd.Type != monster2.CommandTypeUseBasicAttack {
		t.Errorf("Type = %q, want %q", cmd.Type, monster2.CommandTypeUseBasicAttack)
	}
	if cmd.MonsterId != 5001 {
		t.Errorf("MonsterId = %d, want 5001", cmd.MonsterId)
	}
	if cmd.Body.AttackPos != 1 {
		t.Errorf("AttackPos = %d, want 1", cmd.Body.AttackPos)
	}
}
```

If `producer_test.go` already exists and has `package monster` with imports, integrate the test rather than overwriting the file.

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./monster/ -run TestUseBasicAttackCommandProvider -v
```

Expected: FAIL — `UseBasicAttackCommandProvider`, `CommandTypeUseBasicAttack`, `UseBasicAttackCommandBody` undefined.

- [ ] **Step 3: Add command-type + body**

Edit `services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka.go`. Add to the `const` block:

```go
	CommandTypeUseBasicAttack = "USE_BASIC_ATTACK"
```

Add a new body type:

```go
type UseBasicAttackCommandBody struct {
	AttackPos uint8 `json:"attackPos"`
}
```

- [ ] **Step 4: Add the producer**

Edit `services/atlas-channel/atlas.com/channel/monster/producer.go`. After `UseSkillCommandProvider` (line 49), add:

```go
func UseBasicAttackCommandProvider(f field.Model, monsterId uint32, attackPos uint8) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(monsterId))
	value := &monster2.Command[monster2.UseBasicAttackCommandBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		MonsterId: monsterId,
		Type:      monster2.CommandTypeUseBasicAttack,
		Body: monster2.UseBasicAttackCommandBody{
			AttackPos: attackPos,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

- [ ] **Step 5: Add the processor method**

Edit `services/atlas-channel/atlas.com/channel/monster/processor.go`. After `UseSkill` (line 63), add:

```go
func (p *Processor) UseBasicAttack(f field.Model, monsterId uint32, attackPos uint8) error {
	p.l.Debugf("Monster [%d] using basic attack pos [%d].", monsterId, attackPos)
	return producer.ProviderImpl(p.l)(p.ctx)(monster2.EnvCommandTopic)(UseBasicAttackCommandProvider(f, monsterId, attackPos))
}
```

- [ ] **Step 6: Run the test**

```bash
go test ./monster/ -run TestUseBasicAttackCommandProvider -v
```

Expected: PASS.

- [ ] **Step 7: Run the monster package's full tests**

```bash
go test ./monster/...
```

Expected: PASS.

- [ ] **Step 8: Build atlas-channel**

```bash
go build ./...
```

Expected: clean build.

- [ ] **Step 9: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/message/monster/ services/atlas-channel/atlas.com/channel/monster/producer.go services/atlas-channel/atlas.com/channel/monster/processor.go services/atlas-channel/atlas.com/channel/monster/producer_test.go
git commit -m "feat(atlas-channel): add USE_BASIC_ATTACK producer + processor method"
```

---

### Task 11: Wire basic-attack branch in `movement.ForMonster`

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/movement/processor.go:109-167`
- Test: `services/atlas-channel/atlas.com/channel/movement/processor_test.go` (create if needed)

This task ties everything together: classify the inbound `nActionAndDir`,
fetch attack info, compute the forecast `ackMp`, dispatch the Kafka
command. The MP value sent in `MonsterMovementAck` must be the post-decrement
forecast for basic attacks; for everything else it stays as today.

- [ ] **Step 1: Write the failing tests**

Check whether `services/atlas-channel/atlas.com/channel/movement/processor_test.go` exists. If it does not, create it. If it does, append to it.

```go
package movement

import (
	"context"
	"sync"
	"testing"

	"atlas-channel/monster/information"
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/sirupsen/logrus"
)

// captureForecastedAckMp is exercised by tests that want to assert the
// optimistic mp computation without standing up a session-writer fake.
// It mirrors the math in ForMonster.
func TestComputeAckMp_BasicAttackPath_DecrementsByConMp(t *testing.T) {
	atks := []information.AttackInfo{
		{Pos: 2, ConMP: 5, AttackAfter: 1500},
	}
	got := computeAckMp(uint16(100), uint8(1), atks)
	if got != 95 {
		t.Errorf("computeAckMp(100, pos0=1, conMP=5) = %d, want 95", got)
	}
}

func TestComputeAckMp_BasicAttackPath_NoAttackInfo_Untouched(t *testing.T) {
	got := computeAckMp(uint16(100), uint8(0), nil)
	if got != 100 {
		t.Errorf("computeAckMp with no attack info = %d, want 100", got)
	}
}

func TestComputeAckMp_BasicAttackPath_ConMpExceedsMp_ClampsToZero(t *testing.T) {
	atks := []information.AttackInfo{{Pos: 1, ConMP: 50, AttackAfter: 1500}}
	got := computeAckMp(uint16(10), uint8(0), atks)
	if got != 0 {
		t.Errorf("computeAckMp clamps to zero on overflow, got %d", got)
	}
}

func TestComputeAckMp_BasicAttackPath_PosNotFound_Untouched(t *testing.T) {
	atks := []information.AttackInfo{{Pos: 1, ConMP: 5, AttackAfter: 1500}}
	// Caller passed pos0=2, which would map to information Pos=3 — not in atks.
	got := computeAckMp(uint16(100), uint8(2), atks)
	if got != 100 {
		t.Errorf("computeAckMp with pos not found = %d, want 100", got)
	}
}

// suppressUnused silences the "imported and not used" warning on logrus
// during the early stages where we only assert the helper.
var _ = logrus.New
var _ = sync.Mutex{}
var _ = context.Background
var _ = field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(0)).Build
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./movement/ -run "TestComputeAckMp" -v
```

Expected: FAIL — `computeAckMp` undefined.

- [ ] **Step 3: Add `computeAckMp` helper to `processor.go`**

Edit `services/atlas-channel/atlas.com/channel/movement/processor.go`. Add an import:

```go
	monsterinfo "atlas-channel/monster/information"
```

Add the helper (place it near `narrowSkillBytes` at the bottom of the file):

```go
// computeAckMp returns the MP value to advertise in MoveMonsterAck for a
// basic-attack action. It looks up the attack-position's conMP in atks
// (matching the 1-indexed information.AttackInfo.Pos by adding 1 to the
// 0-indexed wire attackPos) and subtracts it from currentMp, clamping to
// zero on underflow. When no matching attack info is present (or atks is
// nil), currentMp passes through unchanged — that matches melee mobs that
// have no info subdir.
func computeAckMp(currentMp uint16, attackPos uint8, atks []monsterinfo.AttackInfo) uint16 {
	wantPos := attackPos + 1
	for _, a := range atks {
		if a.Pos != wantPos {
			continue
		}
		if a.ConMP <= 0 {
			return currentMp
		}
		if uint16(a.ConMP) >= currentMp {
			return 0
		}
		return currentMp - uint16(a.ConMP)
	}
	return currentMp
}
```

- [ ] **Step 4: Run the new tests**

```bash
go test ./movement/ -run "TestComputeAckMp" -v
```

Expected: PASS.

- [ ] **Step 5: Wire `computeAckMp` + Kafka emit into `ForMonster`**

Edit `services/atlas-channel/atlas.com/channel/movement/processor.go`. Replace the `ForMonster` body. Find the existing acknowledgement goroutine:

```go
	go func() {
		useSkills := false
		var skillIdByte, skillLevelByte byte
		if d, hit := monster.GetNextSkillInbox().TakeAndClear(p.t, objectId); hit && !d.IsSentinel() {
			useSkills = true
			skillIdByte = d.SkillId
			skillLevelByte = d.SkillLevel
			p.l.Debugf("Inbox: serving predicted skill (%d,%d) into MoveMonsterAck for monster [%d].", skillIdByte, skillLevelByte, objectId)
		}
		op := session.Announce(p.l)(p.ctx)(p.wp)(monsterpkt.MonsterMovementAckWriter)(monsterpkt.NewMonsterMovementAck(objectId, moveId, uint16(mo.Mp()), useSkills, skillIdByte, skillLevelByte).Encode)
		err = p.sp.IfPresentByCharacterId(f.Channel())(characterId, op)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to ack monster [%d] movement for character [%d].", objectId, characterId)
		}
	}()
```

Replace with this version that classifies basic attacks, fetches attack info, and feeds the forecast MP into the ack:

```go
	// Forecast the post-decrement MP for basic attacks (Cosmic compat — the
	// v83 client gates on the ack carrying decremented MP). For melee /
	// non-basic-attack actions, ackMp passes through unchanged.
	ackMp := uint16(mo.Mp())
	pos0, isBasicAttack := basicAttackPos(skill)
	if isBasicAttack {
		info, ierr := monsterinfo.NewProcessor(p.l, p.ctx).GetById(mo.MonsterId())
		if ierr != nil {
			p.l.WithError(ierr).Debugf("Unable to fetch attack info for monster template [%d]; ack uses unchanged MP.", mo.MonsterId())
		} else {
			ackMp = computeAckMp(ackMp, pos0, info.Attacks())
		}
	}

	go func() {
		useSkills := false
		var skillIdByte, skillLevelByte byte
		if d, hit := monster.GetNextSkillInbox().TakeAndClear(p.t, objectId); hit && !d.IsSentinel() {
			useSkills = true
			skillIdByte = d.SkillId
			skillLevelByte = d.SkillLevel
			p.l.Debugf("Inbox: serving predicted skill (%d,%d) into MoveMonsterAck for monster [%d].", skillIdByte, skillLevelByte, objectId)
		}
		op := session.Announce(p.l)(p.ctx)(p.wp)(monsterpkt.MonsterMovementAckWriter)(monsterpkt.NewMonsterMovementAck(objectId, moveId, ackMp, useSkills, skillIdByte, skillLevelByte).Encode)
		err = p.sp.IfPresentByCharacterId(f.Channel())(characterId, op)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to ack monster [%d] movement for character [%d].", objectId, characterId)
		}
	}()
```

(Note: pull `info` lookup out of the goroutine so the synchronous ack doesn't block on the HTTP fetch needlessly. Actually — invert: the existing skill-prediction inbox fetch is already inside the goroutine. Same pattern keeps the ack non-blocking. Re-examine: the existing pattern already lets the ack go async. The forecast computation must happen BEFORE the goroutine, because we are computing the ack content. The HTTP call cost is amortised by the request cache. Acceptable.)

After the existing goroutines (movement command emit), add the basic-attack Kafka emit:

Find the end of `ForMonster`:

```go
	if skillId > 0 {
		id, lvl, ok := narrowSkillBytes(skillId, skillLevel)
		if !ok {
			p.l.Warnf("Monster [%d] inbound skill out of range (id=%d level=%d); dropping.", objectId, skillId, skillLevel)
		} else {
			go func() {
				err := monster.NewProcessor(p.l, p.ctx).UseSkill(f, objectId, characterId, id, lvl)
				if err != nil {
					p.l.WithError(err).Errorf("Unable to issue use skill command for monster [%d].", objectId)
				}
			}()
		}
	}
	return nil
}
```

Insert immediately before `return nil` (so it lives alongside the named-skill emit):

```go
	if isBasicAttack {
		go func() {
			if err := monster.NewProcessor(p.l, p.ctx).UseBasicAttack(f, objectId, pos0); err != nil {
				p.l.WithError(err).Errorf("Unable to issue basic-attack command for monster [%d].", objectId)
			}
		}()
	}
```

- [ ] **Step 6: Build atlas-channel**

```bash
go build ./...
```

Expected: clean build. If there are unused-import errors in the test file from Step 1's `var _ = ...` placeholders, remove the placeholders that are no longer needed.

- [ ] **Step 7: Run the helper tests**

```bash
go test ./movement/ -run "TestComputeAckMp" -v
```

Expected: PASS.

- [ ] **Step 8: Run full atlas-channel tests**

```bash
go test ./...
```

Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/movement/processor.go services/atlas-channel/atlas.com/channel/movement/processor_test.go
git commit -m "feat(atlas-channel): wire basic-attack branch in movement.ForMonster"
```

---

## Phase 4 — Cross-service verification

### Task 12: Build all three services + acceptance walkthrough

This is the final gate. Each service has been individually built and tested
in its phase; this task confirms nothing broke during cross-service
integration.

- [ ] **Step 1: Build atlas-data**

```bash
cd services/atlas-data/atlas.com/data && go build ./... && go test ./...
```

Expected: clean build, all tests pass.

- [ ] **Step 2: Build atlas-monsters**

```bash
cd services/atlas-monsters/atlas.com/monsters && go build ./... && go test ./...
```

Expected: clean build, all tests pass.

- [ ] **Step 3: Build atlas-channel**

```bash
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...
```

Expected: clean build, all tests pass.

- [ ] **Step 4: Verify the full diff against the design**

Run a self-audit of the diff (compare against `design.md` § "Service-by-service changes"):

- [ ] atlas-data has `AttackInfo` type, `Attacks []AttackInfo` on RestModel, `getAttacks` parser, and `Read` plumbs `getAttacks(exml)` into `m.Attacks`.
- [ ] atlas-monsters has `AttackInfo` on `information.Model` with getter, RestModel field, Extract mapping, and `SetAttacks` builder. `attack_cooldown.go` mirrors `cooldown.go`. `UseBasicAttack` is in the Processor interface and impl, and clears cooldowns on death/destroy. New Kafka command type, body, and handler exist on `EnvCommandTopic`.
- [ ] atlas-channel has the `monster/information` package proxying atlas-data, `basicAttackPos` classification helper, `computeAckMp` MP forecast, basic-attack branch in `ForMonster` (forecast + Kafka emit), `UseBasicAttack` processor + producer, new command-type + body in `kafka/message/monster`.

If anything is missing, add a follow-up task and resolve before merge.

- [ ] **Step 5: Stage the manual gameplay test plan in the PR description**

Document for the integration tester (in the PR description, not a code change):

> Manual gameplay verification on v83 GMS:
>
> 1. Spawn into Fox Ridge map.
> 2. Engage Samiho (`5100004`) → expect repeated magic-attack casts across the encounter.
> 3. Engage `6090003` (melee) → expect repeated melee attacks (regression check).
> 4. Watch atlas-channel logs for `Monster [...] using basic attack pos [...]` debug lines.
> 5. Watch atlas-monsters logs for `UseBasicAttack: ...` debug lines (insufficient MP / on cooldown / happy path).
> 6. Confirm no new "Read a unhandled message with op 0xXX" lines appear.

- [ ] **Step 6: Final commit if any audit fixes were made**

```bash
git status
# If clean, no commit needed.
# If anything changed during the audit, commit it.
```

---

## Self-review notes

**Spec coverage** — every block in `design.md § Service-by-service changes` is mapped to at least one task:

| Design item | Task |
|---|---|
| atlas-data `RestModel` `Attacks []AttackInfo` | Task 1 |
| atlas-data `getAttacks` parser, hook into `Read` | Task 2 |
| atlas-data tests (Samiho with info, Beetle without) | Task 2 |
| atlas-monsters `AttackCooldownRegistry` | Task 4 |
| atlas-monsters cooldown registry init wiring | Task 5 |
| atlas-monsters `information.Model.Attacks()` plumbing | Task 3 |
| atlas-monsters `UseBasicAttack` processor method | Task 6 |
| atlas-monsters `USE_BASIC_ATTACK` Kafka consumer | Task 7 |
| atlas-monsters cooldown clear on monster destroy | Task 5 |
| atlas-monsters tests (cooldown / mp / happy / dead / no-info / melee) | Tasks 4 + 6 |
| atlas-monsters Kafka consumer test | Task 7 |
| atlas-channel `basicAttackPos` classification | Task 9 |
| atlas-channel `ForMonster` basic-attack branch | Task 11 |
| atlas-channel optimistic ackMp forecast | Task 11 |
| atlas-channel `UseBasicAttack` processor + producer | Task 10 |
| atlas-channel `monster/information` package | Task 8 |
| atlas-channel tests | Tasks 9, 10, 11 |
| Final cross-service build / test verification | Task 12 |

**Type-consistency check** — names referenced across tasks:

- `AttackInfo` (atlas-data, atlas-monsters/information, atlas-channel/monster/information) — three definitions, intentionally distinct types living in their own packages, all with fields `Pos uint8`, `ConMP int32`, `AttackAfter int32`. JSON tags match across all three: `pos`, `conMP`, `attackAfter`.
- `attackCooldownRegistry` (atlas-monsters) — accessed via `GetAttackCooldownRegistry()`, initialized via `InitAttackCooldownRegistry(rc)`. Tasks 4, 5, 6 use the same names.
- `CommandTypeUseBasicAttack = "USE_BASIC_ATTACK"` — defined in both `atlas-channel/kafka/message/monster/kafka.go` (Task 10) and `atlas-monsters/kafka/consumer/monster/kafka.go` (Task 7). Same string value — verified.
- `useBasicAttackCommandBody` / `UseBasicAttackCommandBody` — Task 7 (atlas-monsters, lower-case package-private) and Task 10 (atlas-channel, exported `UseBasicAttackCommandBody`). Both have field `AttackPos uint8 \`json:"attackPos"\``. JSON wire shape matches.
- `UseBasicAttack(uniqueId uint32, attackPos uint8)` — atlas-monsters processor signature (Task 6). Channel-side wrapper signature is `UseBasicAttack(f field.Model, monsterId uint32, attackPos uint8) error` (Task 10) — different because it serializes to Kafka. Confirmed intentional.
- `basicAttackPos(rawActionAndDir int8) (uint8, bool)` — Task 9. Used in Task 11.
- `computeAckMp(currentMp uint16, attackPos uint8, atks []monsterinfo.AttackInfo) uint16` — Task 11.

**Placeholder scan** — searched the plan for "TBD", "TODO", "implement later", "fill in details", "Add appropriate", "similar to Task". All code-bearing steps include the actual code.

**Known caveats**:

- Task 6's tests rely on a `testInformationLookup` package-level hook because `UseBasicAttack` calls `information.GetById` directly and there's no existing way to inject that dependency. This is a small test-only seam; it should not bleed into other code paths. If a reviewer prefers, the test could instead spin up an `httptest.Server` and configure `requests.RootUrl("DATA")` — heavier but also valid.
- Task 11 puts the synchronous `monsterinfo.NewProcessor(p.l, p.ctx).GetById(...)` HTTP call on the move-packet hot path before the goroutine that ships the ack. Atlas's `requests.Provider` is synchronous and uses an HTTP client; for high-rate movement packets this will add latency. The atlas-data response is cached at the data service tier and at `requests.Provider` (verify this — the existing pattern is used for skill effect lookups too). If profiling shows this is a hot-path bottleneck, follow-up work could move it inside the goroutine and keep `ackMp = uint16(mo.Mp())` for basic attacks where the lookup hasn't returned yet — at the cost of accepting a one-packet drift before the next move re-syncs. This is a known trade-off in the design and is out of scope for the bug-fix MVP.
