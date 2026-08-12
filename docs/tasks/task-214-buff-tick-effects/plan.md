# Player-Buff Periodic Tick Effects — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace atlas-buffs' POISON-only hard-coded tick path with a declarative periodic-effect table driven by one generic tick task, and make Dragon Blood drain and Recovery heal.

**Architecture:** A new pure `periodic` package holds the effect table (`statType -> {interval, resource, direction, floor}`) and a `Lookup` function — the only place a periodic stat type is named. `character.Registry` gains a single-pass scan yielding one entry per `(character, statType)` and a `(characterId, statType)`-keyed last-tick store replacing the character-keyed `poisonTicks` store. `character.ProcessorImpl` gains an injected clock and HP fetcher (mirroring `berserk.ProcessorImpl`) so the tick pass is unit-testable; it emits `CHANGE_HP` on `COMMAND_TOPIC_CHARACTER` exactly as the poison path does today. One `tasks.PeriodicTick` replaces `tasks.PoisonTick`.

**Tech Stack:** Go 1.x, `libs/atlas-redis` (`TenantRegistry`), `libs/atlas-constants/character` (`TemporaryStatType`), `libs/atlas-kafka` (`message.Buffer` / `message.Emit`, `producertest`), `libs/atlas-routine` (`routine.Go`), miniredis + testify for tests.

## Global Constraints

- **Worktree.** All work happens in `.worktrees/task-214-buff-tick-effects` on branch `task-214-buff-tick-effects`. Never edit the main repo.
- **Service scope.** Only `services/atlas-buffs/atlas.com/buffs/**` changes. atlas-data, atlas-character, atlas-channel, atlas-effective-stats are untouched — design.md §2 Q1/Q2 verified that `reader.go:342` (`DRAGON_BLOOD`) and `reader.go:318` (`RECOVERY`) already store the correct WZ field.
- **No new Go dependency.** `services/atlas-buffs/atlas.com/buffs/go.mod` must not change. If it does, `docker buildx bake atlas-buffs` from the worktree root becomes mandatory (CLAUDE.md item 4).
- **Goroutines** are spawned only via `routine.Go` — never a bare `go` (`tools/goroutine-guard.sh`).
- **Redis** is accessed only through `libs/atlas-redis` types (`tools/redis-key-guard.sh`); never the raw `go-redis` client.
- **Stat-type constants** come from `libs/atlas-constants/character` (`TemporaryStatTypePoison`, `TemporaryStatTypeDragonBlood`, `TemporaryStatTypeRecovery`) — no hand-written string literals in new code (DOM-21).
- **No `// TODO`, stub, or deferred-work marker** in any landed commit.
- **Test helpers** are built inline in `_test.go` files using same-package struct literals (the `berserk/processor_test.go` `testProcessor` shape). No `*_testhelpers.go` file, no test-only constructors in production files.
- Line endings are preserved on every edit; no CRLF→LF normalization.
- **Buff duration is MILLISECONDS** and `buff.NewBuff` rejects `duration <= 0` with `ErrInvalidDuration` (`buff/model.go:144-159`). Test fixtures use `600000` (10 min) for a live buff; an already-lapsed buff is made with duration `1` plus a `10 * time.Millisecond` sleep, because `buff.Model.Expired()` reads `time.Now()` directly and cannot be moved by the processor's injected clock.

**Table values (design.md §2, WZ-verified — do not re-derive, do not change):**

| statType | interval | resource | direction | floor |
|---|---|---|---|---|
| `POISON` | 1 s | HP | Drain | false |
| `DRAGON_BLOOD` | 4 s | HP | Drain | true |
| `RECOVERY` | 5 s | HP | Restore | false |

---

## File Structure

**Created**

| File | Responsibility |
|---|---|
| `services/atlas-buffs/atlas.com/buffs/periodic/model.go` | `Effect`, `Resource`, `Direction` types + accessors. No I/O, no service deps. |
| `services/atlas-buffs/atlas.com/buffs/periodic/table.go` | The effect table and `Lookup`. The single place a periodic stat type is named. |
| `services/atlas-buffs/atlas.com/buffs/periodic/table_test.go` | Row-by-row assertions on interval/resource/direction/floor. |
| `services/atlas-buffs/atlas.com/buffs/character/periodic.go` | Registry surface for the tick path: `TickKey`, `PeriodicEntry`, `GetPeriodicEntries`, the tick store accessors, `ClearPeriodicTicksFor`. Split out of `registry.go` because that file is already 420 lines and this is a self-contained responsibility. |
| `services/atlas-buffs/atlas.com/buffs/character/periodic_test.go` | Scan + tick-store tests. |
| `services/atlas-buffs/atlas.com/buffs/character/periodic_processor_test.go` | Tick-pass semantics: throttling, floor, dedupe, multi-tenancy, lifecycle clearing. |
| `services/atlas-buffs/atlas.com/buffs/tasks/periodic.go` | The `PeriodicTick` ticker task. |
| `services/atlas-buffs/atlas.com/buffs/tasks/periodic_test.go` | Ported `tasks/poison_test.go` assertions. |

**Modified**

| File | Change |
|---|---|
| `character/registry.go` | `poisonTicks` field → `periodicTicks`; delete `PoisonTickEntry`, `GetPoisonCharacters`, `GetLastPoisonTick`, `UpdatePoisonTick`, `ClearPoisonTick`. |
| `character/processor.go` | `ProcessPoisonTicks` → `ProcessPeriodicTicks` (method + package fan-out); add `now` / `getCharacterHp` fields; wire `ClearPeriodicTicksFor` into the four removal paths. |
| `character/testmain_test.go` | `InstallNoop` → `InstallCapturing`, exposing a package-level `emitted *producertest.Capture`. |
| `main.go:75` | `tasks.NewPoisonTick` → `tasks.NewPeriodicTick`. |

**Deleted**

- `services/atlas-buffs/atlas.com/buffs/tasks/poison.go`
- `services/atlas-buffs/atlas.com/buffs/tasks/poison_test.go`

**Not created:** no `audit-statups.md`. FR-5.4 permits the sweep to live as a section of `design.md`, and it already does (design.md §5, all three verdict classes with citations). Duplicating it would create two drifting copies.

**Redis migration:** none. The `buffs-poison:*` key set is abandoned in place (design.md §3.8) — the entries are ephemeral throttle state that costs a few bytes per recently-poisoned character and is self-evidently orphaned.

---

## Task 1: The `periodic` package

Pure table + lookup, no I/O. Independently testable without Redis or Kafka.

**Files:**
- Create: `services/atlas-buffs/atlas.com/buffs/periodic/model.go`
- Create: `services/atlas-buffs/atlas.com/buffs/periodic/table.go`
- Test: `services/atlas-buffs/atlas.com/buffs/periodic/table_test.go`

**Interfaces:**
- Consumes: `github.com/Chronicle20/atlas/libs/atlas-constants/character` — `TemporaryStatType` (a `string` type), constants `TemporaryStatTypePoison`, `TemporaryStatTypeDragonBlood`, `TemporaryStatTypeRecovery`.
- Produces, relied on by Tasks 2–4:
  - `periodic.Effect` with methods `StatType() character.TemporaryStatType`, `Interval() time.Duration`, `Resource() Resource`, `Direction() Direction`, `Floor() bool`
  - `periodic.Resource` (string), constant `periodic.ResourceHP`
  - `periodic.Direction` (int8), constants `periodic.Drain` (-1), `periodic.Restore` (+1)
  - `periodic.Lookup(statType string) (Effect, bool)`

- [ ] **Step 1: Write the failing test**

Create `services/atlas-buffs/atlas.com/buffs/periodic/table_test.go`:

```go
package periodic

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
)

// TestLookupRows pins every row's shape. The values are WZ-verified in
// docs/tasks/task-214-buff-tick-effects/design.md §2 — a row edited by
// accident fails here rather than in production.
func TestLookupRows(t *testing.T) {
	tests := []struct {
		statType  character.TemporaryStatType
		interval  time.Duration
		resource  Resource
		direction Direction
		floor     bool
	}{
		{character.TemporaryStatTypePoison, time.Second, ResourceHP, Drain, false},
		{character.TemporaryStatTypeDragonBlood, 4 * time.Second, ResourceHP, Drain, true},
		{character.TemporaryStatTypeRecovery, 5 * time.Second, ResourceHP, Restore, false},
	}

	for _, tc := range tests {
		t.Run(string(tc.statType), func(t *testing.T) {
			e, ok := Lookup(string(tc.statType))
			assert.True(t, ok, "expected a table row")
			assert.Equal(t, tc.statType, e.StatType())
			assert.Equal(t, tc.interval, e.Interval())
			assert.Equal(t, tc.resource, e.Resource())
			assert.Equal(t, tc.direction, e.Direction())
			assert.Equal(t, tc.floor, e.Floor())
		})
	}
}

// TestLookupRowCount fails when a row is added without a matching assertion in
// TestLookupRows, so the table cannot grow silently.
func TestLookupRowCount(t *testing.T) {
	assert.Len(t, effects, 3)
}

// TestLookupNonPeriodic covers stat types atlas-data emits that design.md §5.3
// gives an "excluded" verdict: the tick path must not pick them up.
func TestLookupNonPeriodic(t *testing.T) {
	for _, st := range []character.TemporaryStatType{
		character.TemporaryStatTypeInfinity,
		character.TemporaryStatTypeMagicGuard,
		character.TemporaryStatTypeWeaponAttack,
		character.TemporaryStatTypeHolyShield,
	} {
		_, ok := Lookup(string(st))
		assert.False(t, ok, "unexpected periodic row for %s", st)
	}
}

func TestLookupUnknownStatType(t *testing.T) {
	_, ok := Lookup("NOT_A_REAL_STAT")
	assert.False(t, ok)
}

func TestDirectionSigns(t *testing.T) {
	assert.Equal(t, Direction(-1), Drain)
	assert.Equal(t, Direction(1), Restore)
}
```

Before running, confirm the four constant names used in `TestLookupNonPeriodic` exist:

```bash
grep -n "TemporaryStatTypeInfinity\|TemporaryStatTypeMagicGuard\|TemporaryStatTypeWeaponAttack\|TemporaryStatTypeHolyShield" libs/atlas-constants/character/temporary_stat.go
```

If any name differs, substitute the actual constant name from that file — do not invent one, and do not fall back to a string literal.

- [ ] **Step 2: Run test to verify it fails**

Run from `services/atlas-buffs/atlas.com/buffs`:

```bash
go test ./periodic/... -v
```

Expected: build failure — `undefined: Lookup`, `undefined: Effect`, `undefined: ResourceHP`, `undefined: Drain`, `undefined: Restore`, `undefined: effects`.

- [ ] **Step 3: Write `model.go`**

Create `services/atlas-buffs/atlas.com/buffs/periodic/model.go`:

```go
// Package periodic holds the declarative table of temporary-stat types that
// carry an ongoing periodic change to the buffed character's own HP/MP, and
// nothing else. It is pure: no Redis, no Kafka, no REST — so the table is
// unit-testable on its own and the tick path in character/ has exactly one
// place to ask "is this stat type periodic, and on what schedule?"
// (task-214 FR-1.1/FR-1.2).
package periodic

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
)

// Resource names the character resource a periodic effect moves.
type Resource string

// ResourceHP is the only resource any current row targets. Adding an MP row
// means adding ResourceMP here AND an emit arm in character.ProcessPeriodicTicks
// — the emitter's default arm logs and skips rather than silently emitting
// nothing.
const ResourceHP Resource = "HP"

// Direction is the sign applied to an effect's per-tick magnitude.
type Direction int8

const (
	// Drain reduces the resource.
	Drain Direction = -1
	// Restore increases the resource.
	Restore Direction = 1
)

// Effect is one row of the periodic-effect table. Fields are unexported with
// accessors so the table cannot be mutated by a caller (project immutable-model
// convention).
type Effect struct {
	statType  character.TemporaryStatType
	interval  time.Duration
	resource  Resource
	direction Direction
	floor     bool
}

// StatType is the temporary-stat type stored on the buff change this row keys off.
func (e Effect) StatType() character.TemporaryStatType { return e.statType }

// Interval is the cadence between ticks for this effect.
func (e Effect) Interval() time.Duration { return e.interval }

// Resource is the character resource the tick moves.
func (e Effect) Resource() Resource { return e.resource }

// Direction is the sign applied to the per-tick magnitude.
func (e Effect) Direction() Direction { return e.direction }

// Floor reports whether the tick must clamp the resource at 1 rather than let
// it reach 0. atlas-character emits a DIED status event whenever an adjusted HP
// lands on 0, so a self-inflicted drain that must not kill sets this true.
func (e Effect) Floor() bool { return e.floor }
```

- [ ] **Step 4: Write `table.go`**

Create `services/atlas-buffs/atlas.com/buffs/periodic/table.go`:

```go
package periodic

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
)

// effects is the periodic-effect table (task-214 FR-1.1). Every value here is
// WZ-verified in docs/tasks/task-214-buff-tick-effects/design.md §2:
//
//   POISON        1s HP drain, no floor — preserves the pre-task-214 tick
//                 behavior exactly; poison is allowed to reach 0 HP.
//   DRAGON_BLOOD  4s HP drain, floor 1 — Skill.wz 1311008 level nodes carry
//                 mpCon/x/time/pad; x decreases with level while pad rises, and
//                 String.wz reads "Use 12 MP, 48 HP in every 4 seconds,
//                 Attack + 1 in 8 Seconds". x is the per-4s HP cost, pad the
//                 attack bonus.
//   RECOVERY      5s HP restore — Skill.wz 10001001 level 1 is x=4, time=30;
//                 String.wz reads "Recover HP 24 in 30 sec." 24/4 = 6 ticks
//                 over 30s = one tick per 5s (levels 2 and 3 confirm).
//
// Intervals are compile-time constants, never configuration and never fetched
// per tick (FR-1.3). This map is the ONLY place a periodic stat type is named
// (FR-1.2) — no tick-path code compares a stat type to a literal.
var effects = map[character.TemporaryStatType]Effect{
	character.TemporaryStatTypePoison: {
		statType:  character.TemporaryStatTypePoison,
		interval:  time.Second,
		resource:  ResourceHP,
		direction: Drain,
		floor:     false,
	},
	character.TemporaryStatTypeDragonBlood: {
		statType:  character.TemporaryStatTypeDragonBlood,
		interval:  4 * time.Second,
		resource:  ResourceHP,
		direction: Drain,
		floor:     true,
	},
	character.TemporaryStatTypeRecovery: {
		statType:  character.TemporaryStatTypeRecovery,
		interval:  5 * time.Second,
		resource:  ResourceHP,
		direction: Restore,
		floor:     false,
	},
}

// Lookup resolves a stored buff change's stat type to its periodic row.
// The parameter is a plain string because buff/stat.Model.Type() is a string;
// the conversion to the typed constant happens here so callers never handle a
// raw literal.
func Lookup(statType string) (Effect, bool) {
	e, ok := effects[character.TemporaryStatType(statType)]
	return e, ok
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
go test ./periodic/... -v
```

Expected: PASS — `TestLookupRows` (3 subtests), `TestLookupRowCount`, `TestLookupNonPeriodic`, `TestLookupUnknownStatType`, `TestDirectionSigns`.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-buffs/atlas.com/buffs/periodic
git commit -m "feat(task-214): periodic-effect table for buff tick effects"
```

---

## Task 2: Registry scan and `(characterId, statType)` tick store

Adds the new registry surface alongside the poison surface so the build and every existing test stay green. The poison surface is removed in Task 5, once nothing calls it.

**Files:**
- Create: `services/atlas-buffs/atlas.com/buffs/character/periodic.go`
- Modify: `services/atlas-buffs/atlas.com/buffs/character/registry.go` (the `Registry` struct at `:23-27` and `InitRegistry` at `:31-41`)
- Test: `services/atlas-buffs/atlas.com/buffs/character/periodic_test.go`

**Interfaces:**
- Consumes: `periodic.Lookup` (Task 1); `atlas.TenantRegistry[K, V]` — `Get`, `PutWithTTL`, `Remove`, `GetAllValues`; `buff/stat.Model` — `Type() string`, `Amount() int32`; `buff.Model` — `Expired() bool`, `Changes() []stat.Model`; existing test helpers `setupTestRegistry`, `setupTestTenant`, `setupTestContext` in `character/registry_test.go`.
- Produces, relied on by Tasks 3 and 4:
  - `character.TickKey{CharacterId uint32; StatType string}`
  - `character.PeriodicEntry{Tenant tenant.Model; WorldId world.Id; ChannelId channel.Id; CharacterId uint32; StatType string; Amount int32}`
  - `(*Registry).GetPeriodicEntries(ctx context.Context) []PeriodicEntry`
  - `(*Registry).GetPeriodicTick(ctx context.Context, key TickKey) (time.Time, bool)`
  - `(*Registry).UpdatePeriodicTick(ctx context.Context, key TickKey, at time.Time)`
  - `(*Registry).ClearPeriodicTick(ctx context.Context, key TickKey)`
  - `(*Registry).ClearPeriodicTicksFor(ctx context.Context, characterId uint32, changeSets ...[]stat.Model)`
  - `character.PeriodicTickTTL` (`time.Duration`)

- [ ] **Step 1: Write the failing test**

Create `services/atlas-buffs/atlas.com/buffs/character/periodic_test.go`:

```go
package character

import (
	"testing"
	"time"

	"atlas-buffs/buff/stat"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

func TestGetPeriodicEntriesEmptyRegistry(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))

	assert.Empty(t, GetRegistry().GetPeriodicEntries(ctx))
}

// TestGetPeriodicEntriesIgnoresNonPeriodicStats: a buff made entirely of flat
// combat modifiers yields nothing for the tick path.
func TestGetPeriodicEntriesIgnoresNonPeriodicStats(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))

	_, err := GetRegistry().Apply(ctx, world.Id(0), channel.Id(1), 100, 2001001, 1, 600000,
		[]stat.Model{stat.NewStat("WEAPON_ATTACK", 30)}, false, false)
	require.NoError(t, err)

	assert.Empty(t, GetRegistry().GetPeriodicEntries(ctx))
}

// TestGetPeriodicEntriesYieldsEveryPeriodicStatOnOneBuff: the pre-task-214 scan
// broke after the first match; two periodic stats on one buff must yield two
// entries.
func TestGetPeriodicEntriesYieldsEveryPeriodicStatOnOneBuff(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))

	_, err := GetRegistry().Apply(ctx, world.Id(0), channel.Id(1), 100, 1311008, 1, 600000,
		[]stat.Model{
			stat.NewStat("DRAGON_BLOOD", 48),
			stat.NewStat("POISON", 25),
			stat.NewStat("WEAPON_ATTACK", 30),
		}, false, false)
	require.NoError(t, err)

	entries := GetRegistry().GetPeriodicEntries(ctx)
	require.Len(t, entries, 2)
	// Sorted by (characterId, statType).
	assert.Equal(t, "DRAGON_BLOOD", entries[0].StatType)
	assert.Equal(t, int32(48), entries[0].Amount)
	assert.Equal(t, "POISON", entries[1].StatType)
	assert.Equal(t, int32(25), entries[1].Amount)
	assert.Equal(t, world.Id(0), entries[0].WorldId)
	assert.Equal(t, channel.Id(1), entries[0].ChannelId)
	assert.Equal(t, uint32(100), entries[0].CharacterId)
}

// TestGetPeriodicEntriesDedupesByMaxAmount: two live buffs carrying the same
// periodic stat collapse to one entry, deterministically the larger amount.
func TestGetPeriodicEntriesDedupesByMaxAmount(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))

	_, err := GetRegistry().Apply(ctx, world.Id(0), channel.Id(1), 100, 5001, 1, 600000,
		[]stat.Model{stat.NewStat("POISON", 10)}, false, false)
	require.NoError(t, err)
	_, err = GetRegistry().Apply(ctx, world.Id(0), channel.Id(1), 100, 5002, 1, 600000,
		[]stat.Model{stat.NewStat("POISON", 25)}, false, false)
	require.NoError(t, err)

	entries := GetRegistry().GetPeriodicEntries(ctx)
	require.Len(t, entries, 1)
	assert.Equal(t, int32(25), entries[0].Amount)
}

func TestGetPeriodicEntriesSkipsExpiredBuffs(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))

	// Duration is MILLISECONDS and must be > 0 (buff/model.go:145). Expired()
	// reads the real wall clock, so a 1ms buff plus a short sleep is the only
	// way to produce a lapsed buff.
	_, err := GetRegistry().Apply(ctx, world.Id(0), channel.Id(1), 100, 5001, 1, 1,
		[]stat.Model{stat.NewStat("POISON", 25)}, false, false)
	require.NoError(t, err)
	time.Sleep(10 * time.Millisecond)

	assert.Empty(t, GetRegistry().GetPeriodicEntries(ctx))
}

func TestPeriodicTickStoreRoundTrip(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))

	key := TickKey{CharacterId: 100, StatType: "POISON"}
	_, ok := GetRegistry().GetPeriodicTick(ctx, key)
	assert.False(t, ok, "no entry before the first tick")

	at := time.Now().Truncate(time.Second)
	GetRegistry().UpdatePeriodicTick(ctx, key, at)

	got, ok := GetRegistry().GetPeriodicTick(ctx, key)
	require.True(t, ok)
	assert.True(t, at.Equal(got), "expected %v, got %v", at, got)

	GetRegistry().ClearPeriodicTick(ctx, key)
	_, ok = GetRegistry().GetPeriodicTick(ctx, key)
	assert.False(t, ok, "cleared entry must not read back")
}

// TestPeriodicTickStoreKeysByStatType: two effects on one character throttle
// independently (FR-2.2).
func TestPeriodicTickStoreKeysByStatType(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))

	poison := TickKey{CharacterId: 100, StatType: "POISON"}
	blood := TickKey{CharacterId: 100, StatType: "DRAGON_BLOOD"}

	GetRegistry().UpdatePeriodicTick(ctx, poison, time.Now())
	_, ok := GetRegistry().GetPeriodicTick(ctx, blood)
	assert.False(t, ok, "DRAGON_BLOOD must not read POISON's throttle")
}

// TestPeriodicTickStoreIsTenantScoped: same character id, two tenants.
func TestPeriodicTickStoreIsTenantScoped(t *testing.T) {
	setupTestRegistry(t)
	ctxA := setupTestContext(t, setupTestTenant(t))
	ctxB := setupTestContext(t, setupTestTenant(t))

	key := TickKey{CharacterId: 100, StatType: "POISON"}
	GetRegistry().UpdatePeriodicTick(ctxA, key, time.Now())

	_, ok := GetRegistry().GetPeriodicTick(ctxB, key)
	assert.False(t, ok, "tenant B must not see tenant A's throttle")
}

// TestClearPeriodicTicksForRemovesOnlyPeriodicStats.
func TestClearPeriodicTicksForRemovesOnlyPeriodicStats(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))

	poison := TickKey{CharacterId: 100, StatType: "POISON"}
	blood := TickKey{CharacterId: 100, StatType: "DRAGON_BLOOD"}
	other := TickKey{CharacterId: 200, StatType: "POISON"}
	now := time.Now()
	GetRegistry().UpdatePeriodicTick(ctx, poison, now)
	GetRegistry().UpdatePeriodicTick(ctx, blood, now)
	GetRegistry().UpdatePeriodicTick(ctx, other, now)

	GetRegistry().ClearPeriodicTicksFor(ctx, 100,
		[]stat.Model{stat.NewStat("POISON", 25), stat.NewStat("WEAPON_ATTACK", 30)},
		[]stat.Model{stat.NewStat("DRAGON_BLOOD", 48)},
	)

	_, ok := GetRegistry().GetPeriodicTick(ctx, poison)
	assert.False(t, ok)
	_, ok = GetRegistry().GetPeriodicTick(ctx, blood)
	assert.False(t, ok)
	_, ok = GetRegistry().GetPeriodicTick(ctx, other)
	assert.True(t, ok, "another character's throttle is untouched")
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./character/... -run 'Periodic' -v
```

Expected: build failure — `undefined: TickKey`, `undefined: GetPeriodicEntries`, `undefined: GetPeriodicTick`, `undefined: UpdatePeriodicTick`, `undefined: ClearPeriodicTick`, `undefined: ClearPeriodicTicksFor`.

- [ ] **Step 3: Add the registry field**

In `services/atlas-buffs/atlas.com/buffs/character/registry.go`, add `periodicTicks` to the struct (keep `poisonTicks` for now — Task 5 removes it):

```go
type Registry struct {
	characters    *atlas.TenantRegistry[uint32, Model]
	poisonTicks   *atlas.TenantRegistry[uint32, time.Time]
	periodicTicks *atlas.TenantRegistry[TickKey, time.Time]
	tenants       *atlas.Set
}
```

and initialize it in `InitRegistry`, after the `poisonTicks` entry:

```go
		periodicTicks: atlas.NewTenantRegistry[TickKey, time.Time](client, "buffs-tick", func(k TickKey) string {
			return strconv.FormatUint(uint64(k.CharacterId), 10) + ":" + k.StatType
		}),
```

- [ ] **Step 4: Write `character/periodic.go`**

Create `services/atlas-buffs/atlas.com/buffs/character/periodic.go`:

```go
package character

import (
	"atlas-buffs/buff/stat"
	"atlas-buffs/periodic"
	"context"
	"sort"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// PeriodicTickTTL bounds a leaked throttle entry. Every live row refreshes its
// key at most one interval (<= 5s) apart, so an active entry never lapses; an
// entry whose owning buff vanished by a removal path we failed to wire
// evaporates on its own. This is belt-and-braces for FR-6.2 — the explicit
// clears in ClearPeriodicTicksFor are still the mechanism, the TTL just makes a
// missed clear "stale for <= 5 min" instead of "leaked forever".
const PeriodicTickTTL = 5 * time.Minute

// TickKey identifies one periodic effect on one character. Keying by stat type
// as well as character is what lets two periodic effects on the same character
// throttle independently (FR-2.2); the pre-task-214 poison store was keyed by
// character alone.
type TickKey struct {
	CharacterId uint32
	StatType    string
}

// PeriodicEntry is one due-able periodic effect found by the scan.
type PeriodicEntry struct {
	Tenant      tenant.Model
	WorldId     world.Id
	ChannelId   channel.Id
	CharacterId uint32
	StatType    string
	Amount      int32
}

// GetPeriodicEntries does ONE traversal of the tenant's stored characters and
// yields every (character, statType) whose stat type has a periodic-effect row
// and whose owning buff has not expired (FR-2.1). Adding a table row adds no
// scan pass.
//
// When two live buffs carry the same periodic stat type for one character, the
// largest Amount wins. Buffs are stored in a Go map, so a first-wins rule would
// pick a different buff on different passes; max-wins is deterministic. With a
// single buff — every real case today — the result is identical to the
// pre-task-214 poison scan.
//
// Results are sorted by (CharacterId, StatType) so a tick pass emits in a
// stable order.
func (r *Registry) GetPeriodicEntries(ctx context.Context) []PeriodicEntry {
	t := tenant.MustFromContext(ctx)
	vals, err := r.characters.GetAllValues(ctx, t)
	if err != nil {
		return nil
	}

	best := make(map[TickKey]PeriodicEntry)
	for _, m := range vals {
		for _, b := range m.buffs {
			if b.Expired() {
				continue
			}
			for _, c := range b.Changes() {
				if _, ok := periodic.Lookup(c.Type()); !ok {
					continue
				}
				k := TickKey{CharacterId: m.characterId, StatType: c.Type()}
				if cur, seen := best[k]; seen && cur.Amount >= c.Amount() {
					continue
				}
				best[k] = PeriodicEntry{
					Tenant:      t,
					WorldId:     m.worldId,
					ChannelId:   m.channelId,
					CharacterId: m.characterId,
					StatType:    c.Type(),
					Amount:      c.Amount(),
				}
			}
		}
	}

	results := make([]PeriodicEntry, 0, len(best))
	for _, e := range best {
		results = append(results, e)
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].CharacterId != results[j].CharacterId {
			return results[i].CharacterId < results[j].CharacterId
		}
		return results[i].StatType < results[j].StatType
	})
	return results
}

// GetPeriodicTick reports when this effect last ticked for this character.
func (r *Registry) GetPeriodicTick(ctx context.Context, key TickKey) (time.Time, bool) {
	t := tenant.MustFromContext(ctx)
	at, err := r.periodicTicks.Get(ctx, t, key)
	if err != nil {
		return time.Time{}, false
	}
	return at, true
}

// UpdatePeriodicTick records a tick.
func (r *Registry) UpdatePeriodicTick(ctx context.Context, key TickKey, at time.Time) {
	t := tenant.MustFromContext(ctx)
	_ = r.periodicTicks.PutWithTTL(ctx, t, key, at, PeriodicTickTTL)
}

// ClearPeriodicTick drops one effect's throttle entry.
func (r *Registry) ClearPeriodicTick(ctx context.Context, key TickKey) {
	t := tenant.MustFromContext(ctx)
	_ = r.periodicTicks.Remove(ctx, t, key)
}

// ClearPeriodicTicksFor drops the throttle entry for every periodic stat type
// carried by the removed buffs' change sets (FR-6.1). Callers pass the
// Changes() of each cancelled/expired buff, mirroring the variadic shape
// markBerserkDirtyOnMaxHpChange already uses in the same removal paths.
func (r *Registry) ClearPeriodicTicksFor(ctx context.Context, characterId uint32, changeSets ...[]stat.Model) {
	for _, changes := range changeSets {
		for _, c := range changes {
			if _, ok := periodic.Lookup(c.Type()); !ok {
				continue
			}
			r.ClearPeriodicTick(ctx, TickKey{CharacterId: characterId, StatType: c.Type()})
		}
	}
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
go test ./character/... -run 'Periodic' -v
go test ./... 
```

Expected: the nine new `*Periodic*` tests PASS, and the whole module still passes — nothing was removed yet.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-buffs/atlas.com/buffs/character/periodic.go \
        services/atlas-buffs/atlas.com/buffs/character/periodic_test.go \
        services/atlas-buffs/atlas.com/buffs/character/registry.go
git commit -m "feat(task-214): periodic scan and (character,statType) tick store"
```

---

## Task 3: The generic tick pass

Adds `ProcessPeriodicTicks` alongside `ProcessPoisonTicks` (still wired to the live ticker; Task 5 does the swap and the deletion). Introduces the injected clock and HP fetcher that make the pass testable.

**Files:**
- Modify: `services/atlas-buffs/atlas.com/buffs/character/processor.go` (`Processor` interface `:19-29`, `ProcessorImpl` `:31-41`, `NewProcessor` `:36-41`; append the new methods after `ProcessPoisonTicks` at `:279`)
- Modify: `services/atlas-buffs/atlas.com/buffs/character/testmain_test.go`
- Test: `services/atlas-buffs/atlas.com/buffs/character/periodic_processor_test.go`

**Interfaces:**
- Consumes: `periodic.Lookup`, `periodic.ResourceHP`, `Effect.Interval/Direction/Floor/Resource` (Task 1); `TickKey`, `PeriodicEntry`, `GetPeriodicEntries`, `GetPeriodicTick`, `UpdatePeriodicTick` (Task 2); existing `changeHPCommandProvider(worldId world.Id, channelId channel.Id, characterId uint32, amount int16)` (`character/producer.go:44`); `character2.EnvCommandTopicCharacter` = `"COMMAND_TOPIC_CHARACTER"`; `character2.CharacterCommand[character2.ChangeHPCommandBody]`; `extchar.RequestById(characterId)(l, ctx)` returning `extchar.RestModel{Hp uint16}`.
- Produces, relied on by Tasks 4 and 5:
  - `Processor` interface method `ProcessPeriodicTicks() error`
  - `ProcessorImpl` fields `now func() time.Time`, `getCharacterHp func(characterId uint32) (uint16, error)`
  - package function `character.ProcessPeriodicTicks(l logrus.FieldLogger, ctx context.Context) error`

- [ ] **Step 1: Switch the test producer to capturing**

Replace `services/atlas-buffs/atlas.com/buffs/character/testmain_test.go` with:

```go
package character

import (
	"os"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
)

// emitted records every message the package's tests produce, so the periodic
// tick tests can assert the exact CHANGE_HP amount. Capturing is a superset of
// the previous no-op install: tests that don't read it are unaffected. Install
// once per package (producer.Manager caches one writer per topic for the
// lifetime of the singleton); each test that reads it calls emitted.Reset()
// first.
var emitted *producertest.Capture

func TestMain(m *testing.M) {
	emitted = producertest.InstallCapturing()
	os.Exit(m.Run())
}
```

- [ ] **Step 2: Write the failing test**

Create `services/atlas-buffs/atlas.com/buffs/character/periodic_processor_test.go`:

```go
package character

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"atlas-buffs/buff/stat"
	character2 "atlas-buffs/kafka/message/character"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// tickProcessor builds a ProcessorImpl with a frozen clock and a stubbed HP
// read. Same-package struct literal, mirroring berserk/processor_test.go's
// testProcessor — no test-helper file, no test-only constructor.
func tickProcessor(ctx context.Context, now *time.Time, hp uint16, hpErr error, hpCalls *int) *ProcessorImpl {
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		now: func() time.Time { return *now },
		getCharacterHp: func(_ uint32) (uint16, error) {
			*hpCalls++
			return hp, hpErr
		},
	}
}

// changeHPAmounts decodes every CHANGE_HP command captured so far, in order.
func changeHPAmounts(t *testing.T) []int16 {
	t.Helper()
	var out []int16
	for _, m := range emitted.Messages(character2.EnvCommandTopicCharacter) {
		var cmd character2.CharacterCommand[character2.ChangeHPCommandBody]
		require.NoError(t, json.Unmarshal(m.Value, &cmd))
		if cmd.Type != character2.CommandChangeHP {
			continue
		}
		out = append(out, cmd.Body.Amount)
	}
	return out
}

func changeHPCommands(t *testing.T) []character2.CharacterCommand[character2.ChangeHPCommandBody] {
	t.Helper()
	var out []character2.CharacterCommand[character2.ChangeHPCommandBody]
	for _, m := range emitted.Messages(character2.EnvCommandTopicCharacter) {
		var cmd character2.CharacterCommand[character2.ChangeHPCommandBody]
		require.NoError(t, json.Unmarshal(m.Value, &cmd))
		out = append(out, cmd)
	}
	return out
}

// applyBuff stores a live buff. 600000 is MILLISECONDS (10 minutes) — long
// enough that no test's frozen-clock arithmetic can outlive it, since
// buff.Model.Expired() reads the real wall clock.
func applyBuff(t *testing.T, ctx context.Context, characterId uint32, sourceId int32, changes ...stat.Model) {
	t.Helper()
	_, err := GetRegistry().Apply(ctx, world.Id(0), channel.Id(1), characterId, sourceId, 1, 600000, changes, false, false)
	require.NoError(t, err)
}

// TestPeriodicTickPoisonParity pins the pre-task-214 poison behavior: a stored
// POISON amount of 25 emits CHANGE_HP -25, is suppressed 500ms later, and
// emits again at 1s (FR-2.4).
func TestPeriodicTickPoisonParity(t *testing.T) {
	setupTestRegistry(t)
	emitted.Reset()
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()
	calls := 0
	p := tickProcessor(ctx, &now, 100, nil, &calls)

	applyBuff(t, ctx, 100, 2111003, stat.NewStat("POISON", 25))

	require.NoError(t, p.ProcessPeriodicTicks())
	assert.Equal(t, []int16{-25}, changeHPAmounts(t))

	now = now.Add(500 * time.Millisecond)
	require.NoError(t, p.ProcessPeriodicTicks())
	assert.Equal(t, []int16{-25}, changeHPAmounts(t), "throttled inside the 1s interval")

	now = now.Add(500 * time.Millisecond)
	require.NoError(t, p.ProcessPeriodicTicks())
	assert.Equal(t, []int16{-25, -25}, changeHPAmounts(t), "emits again at 1s")

	assert.Zero(t, calls, "POISON has no floor, so no HP read is made")
}

// TestPeriodicTickCommandShape pins the emitted command's envelope.
func TestPeriodicTickCommandShape(t *testing.T) {
	setupTestRegistry(t)
	emitted.Reset()
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()
	calls := 0
	p := tickProcessor(ctx, &now, 100, nil, &calls)

	applyBuff(t, ctx, 100, 2111003, stat.NewStat("POISON", 25))
	require.NoError(t, p.ProcessPeriodicTicks())

	cmds := changeHPCommands(t)
	require.Len(t, cmds, 1)
	assert.Equal(t, uint32(100), cmds[0].CharacterId)
	assert.Equal(t, world.Id(0), cmds[0].WorldId)
	assert.Equal(t, character2.CommandChangeHP, cmds[0].Type)
	assert.Equal(t, channel.Id(1), cmds[0].Body.ChannelId)
	assert.Equal(t, int16(-25), cmds[0].Body.Amount)
}

// TestPeriodicTickSkipsNonPositiveMagnitude preserves the poison guard
// (`amount >= 0 { continue }`) generically (FR-1.5).
func TestPeriodicTickSkipsNonPositiveMagnitude(t *testing.T) {
	setupTestRegistry(t)
	emitted.Reset()
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()
	calls := 0
	p := tickProcessor(ctx, &now, 100, nil, &calls)

	applyBuff(t, ctx, 100, 2111003, stat.NewStat("POISON", 0))
	applyBuff(t, ctx, 101, 2111003, stat.NewStat("POISON", -5))

	require.NoError(t, p.ProcessPeriodicTicks())
	assert.Empty(t, changeHPAmounts(t))
}

// TestPeriodicTickIndependentCadences: POISON (1s) and DRAGON_BLOOD (4s) on one
// character throttle independently (FR-2.2).
func TestPeriodicTickIndependentCadences(t *testing.T) {
	setupTestRegistry(t)
	emitted.Reset()
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()
	calls := 0
	p := tickProcessor(ctx, &now, 500, nil, &calls)

	applyBuff(t, ctx, 100, 1311008,
		stat.NewStat("POISON", 25),
		stat.NewStat("DRAGON_BLOOD", 48))

	require.NoError(t, p.ProcessPeriodicTicks())
	assert.ElementsMatch(t, []int16{-25, -48}, changeHPAmounts(t), "t=0 emits both")

	emitted.Reset()
	now = now.Add(time.Second)
	require.NoError(t, p.ProcessPeriodicTicks())
	assert.Equal(t, []int16{-25}, changeHPAmounts(t), "t=1s: POISON only")

	emitted.Reset()
	now = now.Add(3 * time.Second)
	require.NoError(t, p.ProcessPeriodicTicks())
	assert.ElementsMatch(t, []int16{-25, -48}, changeHPAmounts(t), "t=4s: both")
}

// TestPeriodicTickDragonBloodFloorsAtOne (FR-3.4).
func TestPeriodicTickDragonBloodFloorsAtOne(t *testing.T) {
	tests := []struct {
		name string
		hp   uint16
		want []int16
	}{
		{"full hp drains the whole amount", 100, []int16{-48}},
		{"low hp is reduced so hp lands on 1", 30, []int16{-29}},
		{"exactly enough to land on 1", 49, []int16{-48}},
		{"at 1 hp emits nothing", 1, nil},
		{"at 0 hp emits nothing", 0, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setupTestRegistry(t)
			emitted.Reset()
			ctx := setupTestContext(t, setupTestTenant(t))
			now := time.Now()
			calls := 0
			p := tickProcessor(ctx, &now, tc.hp, nil, &calls)

			applyBuff(t, ctx, 100, 1311008, stat.NewStat("DRAGON_BLOOD", 48))
			require.NoError(t, p.ProcessPeriodicTicks())
			assert.Equal(t, tc.want, changeHPAmounts(t))
		})
	}
}

// TestPeriodicTickDragonBloodFailsClosedOnHpError (design D5).
func TestPeriodicTickDragonBloodFailsClosedOnHpError(t *testing.T) {
	setupTestRegistry(t)
	emitted.Reset()
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()
	calls := 0
	p := tickProcessor(ctx, &now, 0, errors.New("boom"), &calls)

	applyBuff(t, ctx, 100, 1311008, stat.NewStat("DRAGON_BLOOD", 48))
	require.NoError(t, p.ProcessPeriodicTicks(), "a failed HP read is not a pass failure")
	assert.Empty(t, changeHPAmounts(t), "never emit an unclamped drain")
}

// TestPeriodicTickHpReadIsMemoizedPerPass (FR-3.6): a character with two
// floor-sensitive rows reads HP once. Today DRAGON_BLOOD is the only floor row,
// so the bound is asserted as "at most one read per affected character per
// pass" across two passes.
func TestPeriodicTickHpReadIsMemoizedPerPass(t *testing.T) {
	setupTestRegistry(t)
	emitted.Reset()
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()
	calls := 0
	p := tickProcessor(ctx, &now, 500, nil, &calls)

	applyBuff(t, ctx, 100, 1311008,
		stat.NewStat("DRAGON_BLOOD", 48),
		stat.NewStat("POISON", 25),
		stat.NewStat("RECOVERY", 4))

	require.NoError(t, p.ProcessPeriodicTicks())
	assert.Equal(t, 1, calls, "one HP read for the one floor-sensitive character")
}

// TestPeriodicTickRecoveryOnlyMakesNoHpRead (NFR load bound).
func TestPeriodicTickRecoveryOnlyMakesNoHpRead(t *testing.T) {
	setupTestRegistry(t)
	emitted.Reset()
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()
	calls := 0
	p := tickProcessor(ctx, &now, 500, nil, &calls)

	applyBuff(t, ctx, 100, 10001001, stat.NewStat("RECOVERY", 4))
	require.NoError(t, p.ProcessPeriodicTicks())

	assert.Equal(t, []int16{4}, changeHPAmounts(t), "positive and unclamped by atlas-buffs (FR-4.4)")
	assert.Zero(t, calls)
}

// TestPeriodicTickRecoveryCadence: 5s, per WZ (FR-4.2).
func TestPeriodicTickRecoveryCadence(t *testing.T) {
	setupTestRegistry(t)
	emitted.Reset()
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()
	calls := 0
	p := tickProcessor(ctx, &now, 500, nil, &calls)

	applyBuff(t, ctx, 100, 10001001, stat.NewStat("RECOVERY", 4))

	require.NoError(t, p.ProcessPeriodicTicks())
	now = now.Add(4 * time.Second)
	require.NoError(t, p.ProcessPeriodicTicks())
	assert.Equal(t, []int16{4}, changeHPAmounts(t), "throttled at 4s")

	now = now.Add(time.Second)
	require.NoError(t, p.ProcessPeriodicTicks())
	assert.Equal(t, []int16{4, 4}, changeHPAmounts(t), "emits at 5s")
}

// TestPeriodicTickDedupesDuplicatePoisonBuffs (design D7).
func TestPeriodicTickDedupesDuplicatePoisonBuffs(t *testing.T) {
	setupTestRegistry(t)
	emitted.Reset()
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()
	calls := 0
	p := tickProcessor(ctx, &now, 500, nil, &calls)

	applyBuff(t, ctx, 100, 5001, stat.NewStat("POISON", 10))
	applyBuff(t, ctx, 100, 5002, stat.NewStat("POISON", 25))

	require.NoError(t, p.ProcessPeriodicTicks())
	assert.Equal(t, []int16{-25}, changeHPAmounts(t))
}

// TestPeriodicTickIsTenantScoped: same character id in two tenants ticks twice
// and throttles independently.
func TestPeriodicTickIsTenantScoped(t *testing.T) {
	setupTestRegistry(t)
	emitted.Reset()
	ctxA := setupTestContext(t, setupTestTenant(t))
	ctxB := setupTestContext(t, setupTestTenant(t))
	now := time.Now()
	callsA, callsB := 0, 0
	pA := tickProcessor(ctxA, &now, 500, nil, &callsA)
	pB := tickProcessor(ctxB, &now, 500, nil, &callsB)

	applyBuff(t, ctxA, 100, 2111003, stat.NewStat("POISON", 25))
	applyBuff(t, ctxB, 100, 2111003, stat.NewStat("POISON", 11))

	require.NoError(t, pA.ProcessPeriodicTicks())
	require.NoError(t, pB.ProcessPeriodicTicks())
	assert.ElementsMatch(t, []int16{-25, -11}, changeHPAmounts(t))
}
```

- [ ] **Step 3: Run test to verify it fails**

```bash
go test ./character/... -run 'PeriodicTick' -v
```

Expected: build failure — `unknown field now in struct literal`, `unknown field getCharacterHp`, `p.ProcessPeriodicTicks undefined`.

- [ ] **Step 4: Add the injected dependencies**

In `services/atlas-buffs/atlas.com/buffs/character/processor.go`, add the interface method (leave `ProcessPoisonTicks()` in place for now):

```go
type Processor interface {
	GetById(characterId uint32) (Model, error)
	Apply(worldId world.Id, channelId channel.Id, characterId uint32, fromId uint32, sourceId int32, level byte, duration int32, changes []stat.Model, accumulate bool, noExpiry bool) error
	Cancel(worldId world.Id, characterId uint32, sourceId int32) error
	CancelAll(worldId world.Id, characterId uint32) error
	CancelByStatTypes(worldId world.Id, characterId uint32, types []string) error
	UpdateStatValue(worldId world.Id, characterId uint32, sourceId int32, statType string, operation string, amount int32, capValue int32) error
	ExpireBuffs() error
	ExpireForCharacter(worldId world.Id, characterId uint32) error
	ProcessPoisonTicks() error
	ProcessPeriodicTicks() error
}
```

and the fields plus their production wiring:

```go
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	// now and getCharacterHp are injected so the periodic tick pass is
	// deterministic under test (same shape as berserk.ProcessorImpl).
	now            func() time.Time
	getCharacterHp func(characterId uint32) (uint16, error)
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
		now: time.Now,
	}
	p.getCharacterHp = func(characterId uint32) (uint16, error) {
		rm, err := extchar.RequestById(characterId)(l, ctx)
		if err != nil {
			return 0, err
		}
		return rm.Hp, nil
	}
	return p
}
```

Add the import `extchar "atlas-buffs/external/character"` to the import block.

- [ ] **Step 5: Write the tick pass**

Append to `services/atlas-buffs/atlas.com/buffs/character/processor.go`, after `ProcessPoisonTicks`:

```go
// hpLookup memoizes one character's HP read for the duration of a single tick
// pass, including the failure outcome — a character whose HP could not be read
// is not retried within the same pass (FR-3.6).
type hpLookup struct {
	hp uint16
	ok bool
}

// maxTickMagnitude clamps a per-tick magnitude before the int16 conversion the
// CHANGE_HP command body requires. No real WZ value approaches it; the clamp
// exists so a corrupt stored amount degrades to a large tick instead of
// wrapping sign and turning a drain into a heal.
const maxTickMagnitude = int32(32767)

// ProcessPeriodicTicks is one tick pass for one tenant. It scans once
// (GetPeriodicEntries), then for each due (character, statType) emits the
// resource change the periodic-effect table prescribes. A row is due when it
// has never ticked or when now - lastTick >= the row's interval, so one 1s
// driving task honors every row's own cadence (FR-2.3).
//
// The throttle read and the store update straddle buf.Put, exactly as the
// pre-task-214 poison path did: a crash between the two re-ticks on the next
// pass, a crash before Put skips a tick. Both are one-interval errors on a
// non-idempotent HP mutation. Making this exactly-once needs an idempotency key
// on the CHANGE_HP command — an atlas-character contract change, out of scope
// for task-214 (design.md §3.5).
func (p *ProcessorImpl) ProcessPeriodicTicks() error {
	entries := GetRegistry().GetPeriodicEntries(p.ctx)
	now := p.now()
	hpCache := make(map[uint32]hpLookup)

	return message.Emit(p.l, p.ctx)(func(buf *message.Buffer) error {
		for _, entry := range entries {
			eff, ok := periodic.Lookup(entry.StatType)
			if !ok {
				continue
			}

			key := TickKey{CharacterId: entry.CharacterId, StatType: entry.StatType}
			if last, ticked := GetRegistry().GetPeriodicTick(p.ctx, key); ticked && now.Sub(last) < eff.Interval() {
				continue
			}

			// A non-positive stored magnitude is skipped, preserving the
			// pre-task-214 poison guard generically (FR-1.5).
			magnitude := entry.Amount
			if magnitude <= 0 {
				continue
			}
			if magnitude > maxTickMagnitude {
				magnitude = maxTickMagnitude
			}

			// One arm per resource. The default is a guard, not a stub: it is
			// unreachable with today's rows, and its job is to make a future MP
			// row fail loudly at the first tick instead of silently emitting
			// nothing.
			switch eff.Resource() {
			case periodic.ResourceHP:
			default:
				p.l.Errorf("Periodic effect [%s] targets unmapped resource [%s]; no command emitted.", entry.StatType, eff.Resource())
				continue
			}

			amount := int16(eff.Direction()) * int16(magnitude)

			if eff.Floor() && amount < 0 {
				hp, ok := p.hpFor(hpCache, entry.CharacterId)
				if !ok {
					continue
				}
				if hp <= 1 {
					p.l.Debugf("Periodic tick [%s] for character [%d] suppressed: already at [%d] HP.", entry.StatType, entry.CharacterId, hp)
					continue
				}
				if int32(hp)+int32(amount) < 1 {
					amount = -int16(hp - 1)
				}
			}

			p.l.Debugf("Periodic tick [%s] for character [%d], amount [%d].", entry.StatType, entry.CharacterId, amount)

			if err := buf.Put(character2.EnvCommandTopicCharacter, changeHPCommandProvider(entry.WorldId, entry.ChannelId, entry.CharacterId, amount)); err != nil {
				return err
			}

			GetRegistry().UpdatePeriodicTick(p.ctx, key, now)
		}
		return nil
	})
}

// hpFor reads a character's current HP at most once per tick pass. A read
// failure is cached as a miss and logged: the caller skips the tick rather than
// emitting an unclamped drain, because one missed 4s tick is invisible and one
// unintended DIED is not (design D5).
func (p *ProcessorImpl) hpFor(cache map[uint32]hpLookup, characterId uint32) (uint16, bool) {
	if c, seen := cache[characterId]; seen {
		return c.hp, c.ok
	}
	hp, err := p.getCharacterHp(characterId)
	if err != nil {
		p.l.WithError(err).Warnf("Unable to read HP for character [%d]; skipping floor-sensitive periodic tick.", characterId)
		cache[characterId] = hpLookup{}
		return 0, false
	}
	cache[characterId] = hpLookup{hp: hp, ok: true}
	return hp, true
}

// ProcessPeriodicTicks fans one tick pass out per tenant (FR-2.5): tenant work
// runs under tenant.WithContext in a routine.Go goroutine, same shape as the
// expiration and berserk sweeps.
func ProcessPeriodicTicks(l logrus.FieldLogger, ctx context.Context) error {
	ts, err := GetRegistry().GetTenants(ctx)
	if err != nil {
		return err
	}

	for _, t := range ts {
		routine.Go(l, ctx, func(_ context.Context) {
			tctx := tenant.WithContext(ctx, t)
			if err := NewProcessor(l, tctx).ProcessPeriodicTicks(); err != nil {
				l.WithError(err).Error("Failed to process periodic ticks for tenant.")
			}
		})
	}
	return nil
}
```

Add the import `"atlas-buffs/periodic"` to the import block.

- [ ] **Step 6: Run tests to verify they pass**

```bash
go test ./character/... -run 'PeriodicTick' -v
go test ./...
```

Expected: all `PeriodicTick*` tests PASS (including the five `DragonBloodFloorsAtOne` subtests), and the rest of the module still passes.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-buffs/atlas.com/buffs/character/processor.go \
        services/atlas-buffs/atlas.com/buffs/character/periodic_processor_test.go \
        services/atlas-buffs/atlas.com/buffs/character/testmain_test.go
git commit -m "feat(task-214): generic periodic tick pass with dragon-blood HP floor"
```

---

## Task 4: Lifecycle clearing on cancel and expiry

`ClearPoisonTick` had zero callers. The replacement must actually be wired into every removal path, and a test must fail if that regresses.

**Files:**
- Modify: `services/atlas-buffs/atlas.com/buffs/character/processor.go` — `Cancel` (`:78-106`), `CancelAll` (`:108-130`), `CancelByStatTypes` (`:132-166`), `expireInto` (`:218-234`)
- Test: `services/atlas-buffs/atlas.com/buffs/character/periodic_processor_test.go` (append)

**Interfaces:**
- Consumes: `(*Registry).ClearPeriodicTicksFor` and `GetPeriodicTick` (Task 2); `ProcessorImpl.ProcessPeriodicTicks` (Task 3).
- Produces: no new exported surface — behavior only.

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-buffs/atlas.com/buffs/character/periodic_processor_test.go`:

```go
// TestPeriodicTickClearedOnRemoval covers FR-6.1/FR-6.2: every removal path
// drops the (character, statType) throttle entry. ClearPoisonTick's
// zero-caller state must not recur.
func TestPeriodicTickClearedOnRemoval(t *testing.T) {
	const characterId = uint32(100)
	const sourceId = int32(1311008)
	key := TickKey{CharacterId: characterId, StatType: "DRAGON_BLOOD"}

	tests := []struct {
		name   string
		remove func(t *testing.T, p *ProcessorImpl)
	}{
		{"cancel", func(t *testing.T, p *ProcessorImpl) {
			require.NoError(t, p.Cancel(world.Id(0), characterId, sourceId))
		}},
		{"cancel all", func(t *testing.T, p *ProcessorImpl) {
			require.NoError(t, p.CancelAll(world.Id(0), characterId))
		}},
		{"cancel by stat types", func(t *testing.T, p *ProcessorImpl) {
			require.NoError(t, p.CancelByStatTypes(world.Id(0), characterId, []string{"DRAGON_BLOOD"}))
		}},
		{"expire for character", func(t *testing.T, p *ProcessorImpl) {
			require.NoError(t, p.ExpireForCharacter(world.Id(0), characterId))
		}},
		{"expire buffs sweep", func(t *testing.T, p *ProcessorImpl) {
			require.NoError(t, p.ExpireBuffs())
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setupTestRegistry(t)
			emitted.Reset()
			ctx := setupTestContext(t, setupTestTenant(t))
			now := time.Now()
			calls := 0
			p := tickProcessor(ctx, &now, 500, nil, &calls)

			// The expiry cases need a buff that has already lapsed; the cancel
			// cases need a live one. Duration is MILLISECONDS and must be > 0,
			// so "lapsed" is 1ms plus a sleep — Expired() reads the real clock.
			expiring := tc.name == "expire for character" || tc.name == "expire buffs sweep"
			duration := int32(600000)
			if expiring {
				duration = 1
			}
			_, err := GetRegistry().Apply(ctx, world.Id(0), channel.Id(1), characterId, sourceId, 1, duration,
				[]stat.Model{stat.NewStat("DRAGON_BLOOD", 48)}, false, false)
			require.NoError(t, err)
			if expiring {
				time.Sleep(10 * time.Millisecond)
			}

			GetRegistry().UpdatePeriodicTick(ctx, key, now)
			_, ok := GetRegistry().GetPeriodicTick(ctx, key)
			require.True(t, ok, "precondition: throttle entry exists")

			tc.remove(t, p)

			_, ok = GetRegistry().GetPeriodicTick(ctx, key)
			assert.False(t, ok, "removal path must clear the throttle entry")
		})
	}
}

// TestPeriodicTickRestartsAfterRecast: with the throttle cleared on cancel, a
// re-cast ticks immediately instead of waiting out the old schedule.
func TestPeriodicTickRestartsAfterRecast(t *testing.T) {
	setupTestRegistry(t)
	emitted.Reset()
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()
	calls := 0
	p := tickProcessor(ctx, &now, 500, nil, &calls)

	applyBuff(t, ctx, 100, 1311008, stat.NewStat("DRAGON_BLOOD", 48))
	require.NoError(t, p.ProcessPeriodicTicks())
	require.Equal(t, []int16{-48}, changeHPAmounts(t))

	require.NoError(t, p.Cancel(world.Id(0), 100, 1311008))
	emitted.Reset()

	now = now.Add(time.Second) // well inside the 4s interval
	applyBuff(t, ctx, 100, 1311008, stat.NewStat("DRAGON_BLOOD", 48))
	require.NoError(t, p.ProcessPeriodicTicks())
	assert.Equal(t, []int16{-48}, changeHPAmounts(t), "cleared throttle means the re-cast ticks at once")
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./character/... -run 'PeriodicTickClearedOnRemoval|PeriodicTickRestartsAfterRecast' -v
```

Expected: FAIL — every subtest reports `removal path must clear the throttle entry` (`Should be false`), and `RestartsAfterRecast` emits nothing on the second pass.

- [ ] **Step 3: Wire the clear into all four removal paths**

In `services/atlas-buffs/atlas.com/buffs/character/processor.go`, each of the four sites already builds a `sets [][]stat.Model` (or has `ebs`) immediately before calling `markBerserkDirtyOnMaxHpChange`. Add the clear on the line above that call.

`Cancel` (currently `:100-104`):

```go
	sets := make([][]stat.Model, 0, len(cancelled))
	for _, b := range cancelled {
		sets = append(sets, b.Changes())
	}
	GetRegistry().ClearPeriodicTicksFor(p.ctx, characterId, sets...)
	markBerserkDirtyOnMaxHpChange(p.l, p.ctx, characterId, sets...)
	return nil
```

`CancelAll` (currently `:124-128`):

```go
	sets := make([][]stat.Model, 0, len(buffs))
	for _, b := range buffs {
		sets = append(sets, b.Changes())
	}
	GetRegistry().ClearPeriodicTicksFor(p.ctx, characterId, sets...)
	markBerserkDirtyOnMaxHpChange(p.l, p.ctx, characterId, sets...)
	return nil
```

`CancelByStatTypes` (currently `:160-164`):

```go
	sets := make([][]stat.Model, 0, len(cancelled))
	for _, b := range cancelled {
		sets = append(sets, b.Changes())
	}
	GetRegistry().ClearPeriodicTicksFor(p.ctx, characterId, sets...)
	markBerserkDirtyOnMaxHpChange(p.l, p.ctx, characterId, sets...)
	return nil
```

`expireInto` (currently `:226-232`) — note the existing `if len(ebs) > 0` guard; put the clear inside it, since with no expired buffs there is nothing to clear:

```go
	if len(ebs) > 0 {
		sets := make([][]stat.Model, 0, len(ebs))
		for _, eb := range ebs {
			sets = append(sets, eb.Changes())
		}
		GetRegistry().ClearPeriodicTicksFor(p.ctx, characterId, sets...)
		markBerserkDirtyOnMaxHpChange(p.l, p.ctx, characterId, sets...)
	}
	return nil
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./character/... -v
```

Expected: PASS — all five `ClearedOnRemoval` subtests and `RestartsAfterRecast`, plus every pre-existing `character` test.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-buffs/atlas.com/buffs/character/processor.go \
        services/atlas-buffs/atlas.com/buffs/character/periodic_processor_test.go
git commit -m "fix(task-214): clear periodic tick state on buff cancel and expiry"
```

---

## Task 5: Swap the ticker task and delete the poison surface

One commit that moves the live wiring to the generic path and removes everything the poison path owned, so no dead code lands.

**Files:**
- Create: `services/atlas-buffs/atlas.com/buffs/tasks/periodic.go`
- Create: `services/atlas-buffs/atlas.com/buffs/tasks/periodic_test.go`
- Delete: `services/atlas-buffs/atlas.com/buffs/tasks/poison.go`
- Delete: `services/atlas-buffs/atlas.com/buffs/tasks/poison_test.go`
- Modify: `services/atlas-buffs/atlas.com/buffs/main.go:75`
- Modify: `services/atlas-buffs/atlas.com/buffs/character/processor.go` (remove `ProcessPoisonTicks` method + interface entry + package function)
- Modify: `services/atlas-buffs/atlas.com/buffs/character/registry.go` (remove `poisonTicks` field and init, `PoisonTickEntry`, `GetPoisonCharacters`, `GetLastPoisonTick`, `UpdatePoisonTick`, `ClearPoisonTick`)
- Modify: `services/atlas-buffs/atlas.com/buffs/berserk/processor.go:268` (comment references `character.ProcessPoisonTicks`)

**Interfaces:**
- Consumes: `character.ProcessPeriodicTicks(l, ctx)` (Task 3); `tasks.Task` interface (`Run()`, `SleepTime() time.Duration`); `tasks.Register(l, ctx)(t Task)`.
- Produces: `tasks.NewPeriodicTick(l logrus.FieldLogger, interval int) *PeriodicTick`.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-buffs/atlas.com/buffs/tasks/periodic_test.go` (the three assertions ported from `poison_test.go`):

```go
package tasks

import (
	"atlas-buffs/character"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestPeriodicTick_SleepTime_RespectsConfiguredInterval(t *testing.T) {
	pt := NewPeriodicTick(logrus.New(), 750)
	require.Equal(t, 750*time.Millisecond, pt.SleepTime())
}

func TestPeriodicTick_SleepTime_DefaultMillisecondMath(t *testing.T) {
	pt := NewPeriodicTick(logrus.New(), 1000)
	require.Equal(t, time.Second, pt.SleepTime())
}

// TestPeriodicTick_Run_DoesNotPanicWithNoTenants verifies Run() is safe to
// invoke when there are no registered tenants in Redis. miniredis stands in for
// the registry's backing store so Run()'s call into
// character.ProcessPeriodicTicks reaches a real (but empty) tenant set instead
// of dereferencing a nil registry client.
func TestPeriodicTick_Run_DoesNotPanicWithNoTenants(t *testing.T) {
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	character.InitRegistry(client)

	pt := NewPeriodicTick(logrus.New(), 1000)
	require.NotPanics(t, func() { pt.Run() })
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./tasks/... -run 'PeriodicTick' -v
```

Expected: build failure — `undefined: NewPeriodicTick`.

- [ ] **Step 3: Create the task and delete the poison one**

Create `services/atlas-buffs/atlas.com/buffs/tasks/periodic.go`:

```go
package tasks

import (
	"atlas-buffs/character"
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
)

// PeriodicTick drives every row of the periodic-effect table. The interval is
// the DRIVING cadence, not any row's cadence: it must be fine enough to honor
// the shortest row (POISON, 1s), and each row is emitted only when its own
// interval has elapsed (task-214 FR-2.3).
type PeriodicTick struct {
	l        logrus.FieldLogger
	interval int
}

func NewPeriodicTick(l logrus.FieldLogger, interval int) *PeriodicTick {
	return &PeriodicTick{l, interval}
}

func (r *PeriodicTick) Run() {
	ctx, span := otel.GetTracerProvider().Tracer("atlas-buffs").Start(context.Background(), "periodic_tick_task")
	defer span.End()

	_ = character.ProcessPeriodicTicks(r.l, ctx)
}

func (r *PeriodicTick) SleepTime() time.Duration {
	return time.Millisecond * time.Duration(r.interval)
}
```

Then:

```bash
git rm services/atlas-buffs/atlas.com/buffs/tasks/poison.go \
       services/atlas-buffs/atlas.com/buffs/tasks/poison_test.go
```

- [ ] **Step 4: Rewire `main.go`**

In `services/atlas-buffs/atlas.com/buffs/main.go`, replace line 75:

```go
	routine.Go(l, rt.Context(), func(_ context.Context) {
		tasks.Register(l, rt.Context())(tasks.NewPeriodicTick(l, 1000))
	})
```

Exactly one periodic tick task is registered, alongside `NewExpiration` and `NewBerserkTick`.

- [ ] **Step 5: Delete the poison surface**

In `character/processor.go`: remove `ProcessPoisonTicks() error` from the `Processor` interface, the `func (p *ProcessorImpl) ProcessPoisonTicks() error` method (`:253-279`), and the package-level `func ProcessPoisonTicks(l logrus.FieldLogger, ctx context.Context) error` (`:281-296`).

In `character/registry.go`: remove the `poisonTicks` struct field and its `InitRegistry` entry, and delete `PoisonTickEntry` (`:292-298`), `GetPoisonCharacters` (`:300-328`), `GetLastPoisonTick` (`:330-337`), `UpdatePoisonTick` (`:339-342`), and `ClearPoisonTick` (`:344-347`). If `time` becomes an unused import in that file, drop it.

In `berserk/processor.go:268`, update the stale comment reference:

```go
// ProcessBerserkTicks fans out one ProcessTicks per tenant (ticker entry
// point; same shape as character.ProcessPeriodicTicks, character/processor.go).
```

- [ ] **Step 6: Verify no poison surface remains**

Run from `services/atlas-buffs/atlas.com/buffs`:

```bash
grep -rn "PoisonTick\|poisonTicks\|PoisonTickEntry\|GetPoisonCharacters\|buffs-poison" .
```

Expected: no output.

`TemporaryStatTypePoison` in `periodic/table.go` and the `"POISON"` strings in test data are the stat-type constant and its wire value — they stay, and the pattern above deliberately does not match them.

- [ ] **Step 7: Run the full module test suite**

```bash
go build ./...
go test -race ./...
```

Expected: build clean, all packages PASS.

- [ ] **Step 8: Commit**

```bash
git add -A services/atlas-buffs
git commit -m "refactor(task-214): replace poison tick task with generic periodic tick"
```

---

## Task 6: Full verification sweep

The project's definition of done. No code changes are expected here; anything this surfaces gets fixed on this branch, not deferred.

**Files:** none created. Fixes land in the files above.

**Interfaces:** none.

- [ ] **Step 1: Module-level checks**

From `services/atlas-buffs/atlas.com/buffs`:

```bash
go build ./...
go vet ./...
go test -race ./...
```

Expected: all three clean.

- [ ] **Step 2: Confirm `go.mod` is untouched**

From the worktree root:

```bash
git diff --stat main -- services/atlas-buffs/atlas.com/buffs/go.mod services/atlas-buffs/atlas.com/buffs/go.sum
```

Expected: empty. If either file changed, `docker buildx bake atlas-buffs` from the worktree root becomes mandatory (CLAUDE.md item 4) — run it and confirm it succeeds before proceeding.

- [ ] **Step 3: Repo guards**

From the worktree root:

```bash
tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/lint.sh --check
```

Expected: exit 0 for each. `tools/lint.sh --check` needs nvm on PATH — if it reports a frontend failure with no atlas-ui files in the diff, source nvm and re-run before treating it as real. If the formatter reports diffs, run `tools/lint.sh` (no flags) to fix in place, then re-run `--check` and amend.

The remaining guards are not applicable and need no run: no tenant socket-config template, job/skill id constant, `COMMAND_TOPIC_CHARACTER_BUFF` `duration` field, trade contract, or service-registration list changed.

- [ ] **Step 4: Acceptance-criteria walkthrough**

Confirm each PRD §10 box against the branch, citing file:line — do not check a box from memory:

```bash
git diff --stat main -- services/atlas-buffs
grep -rn "\"POISON\"\|\"DRAGON_BLOOD\"\|\"RECOVERY\"" services/atlas-buffs/atlas.com/buffs --include='*.go' | grep -v _test.go
```

The second command's only hits must be inside `periodic/` (the table). A literal in `character/` or `tasks/` non-test code violates FR-1.2 and must be replaced with a `periodic.Lookup` call.

- [ ] **Step 5: Commit any fixes**

```bash
git add -A services/atlas-buffs
git commit -m "chore(task-214): verification fixes"
```

Skip this step if steps 1–4 were clean with no edits.

---

## Notes for the reviewer

- **FR-5 (audit sweep)** is satisfied by `design.md` §5, committed on this branch: 3 wired, 1 deferred (`COMBO_DRAIN` → task-166), and the rest excluded with per-row citations. FR-5.5 result: nothing periodic-and-unowned was found, so nothing is filed as a follow-up.
- **Two accepted, documented races** (design.md §3.5, §3.6): the throttle-vs-emit window can double- or skip-tick by one interval across a crash, and the Dragon Blood HP snapshot can still be raced to 0 by concurrent damage. Both need an atlas-character contract change (an idempotency key / a non-lethal flag) that PRD §7 puts out of scope. They are recorded, not silently inherited.
- **`buffs-poison:*` Redis keys** are abandoned in place, not migrated (design.md §3.8).
- **Dragon Blood's WZ `desc`** implies cancel-on-exhaust rather than floor-at-1. The PRD's floor-at-1 choice is deliberate (design.md §2 Q1) — this is not a bug to re-file.
