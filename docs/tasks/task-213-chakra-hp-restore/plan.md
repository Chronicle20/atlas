# Chakra (Chief Bandit 4211001) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make Chakra restore HP end-to-end in `atlas-channel` — an HP-gated recovery window opened by the client's skill-prepare packet, a WZ-driven incoming-damage factor applied first in the mitigation chain, interruption on damage/movement/map-change/disconnect, and the heal itself applied when the client's `USE_SKILL` arrives at animation end.

**Architecture:** Chakra reaches the server as **two** packets: `CharacterSkillPrepareHandle` (keypress) then `CharacterUseSkillHandle` (~1500 ms later, at animation end). The prepare handler runs the activation gate and opens a tenant-scoped, in-process recovery-state entry snapshotting the WZ `x` (damage-taken %) and `y` (recovery %) for the caster's real skill-book level. The damage-taken path reads that entry and feeds `x` into `computeMitigation` as the **first** term. The `USE_SKILL` path rejects pre-cost when no entry exists, then the per-skill `Handler` registered on `skill2.ChiefBanditChakra` computes and applies the heal and clears the entry. MP and cooldown are charged exactly once by the generic `UseSkill` block; an interrupted cast never reaches it and therefore costs nothing.

**Tech Stack:** Go 1.x, `atlas-channel` service module, `libs/atlas-constants` (skill identities, field), `libs/atlas-routine` (`routine.Go`), `libs/atlas-tenant`. Table-driven `testing` with `-race`. No new module, no new dependency, no `go.mod` change.

## Global Constraints

- **No raw wire-id comparison.** Every skill-id decision routes through `constants.For(t.Region(), t.MajorVersion(), t.MinorVersion()).Skill.Resolve(...)` + `skill.IsIdentity(id, skill.ChiefBanditChakra)` (PRD FR-9.1; `tools/skill-job-id-guard.sh`).
- **No `MajorAtLeast` / version gate anywhere in this change.** Every Chakra version difference is a WZ data difference already carried per-tenant by `effect.Model` (design §4.2, §5.4). Adding a version gate here is the bug, not the fix.
- **The handler charges no MP and applies no cooldown.** The generic `UseSkill` block (`skill/handler/common.go:137-165`) owns both and runs before handler dispatch (PRD FR-8.2/8.3).
- **No XP is awarded** (PRD FR-8.4) — satisfied by omission; Chakra's handler has no `AwardExperience` call.
- **No `libs/atlas-packet` change, no `libs/atlas-constants` change, no `atlas-data` production change** (design §3.6, §3.5, §4.1). `x`/`y` are already parsed at `services/atlas-data/atlas.com/data/skill/reader.go:266-267` and exposed as `effect.Model.X()` / `.Y()`.
- **Every goroutine goes through `routine.Go`** (`tools/goroutine-guard.sh`). No Redis is used, so `tools/redis-key-guard.sh` is not engaged.
- **No `// TODO`, stub, or placeholder in any landed commit.**
- **No Cosmic file/line citation in code comments.** The community provenance of the heal base term is cited in `design.md` §3.4 only; code comments state *that* it is community-sourced and unverified, without naming the source project.
- **Integer arithmetic only** on the damage and heal paths, matching the decompiled formulas exactly (`character_damage_mitigation.go:129-132`).
- **Commit after every task.** Branch `task-213-chakra-hp-restore`, worktree `.worktrees/task-213-chakra-hp-restore`. All paths below are relative to that worktree root.

## Plan-phase decisions that deviate from `design.md`

Three corrections were forced by reading the code and the packet registry. Each is deliberate and is repeated at the task that implements it.

1. **State package location.** Design §5.1 puts the registry at `skill/handler/chakra/state.go`. That creates an import cycle: `skill/handler/*` packages import `atlas-channel/socket/handler` (see `skill/handler/heal/heal.go:16`), so `socket/handler` cannot import back into `skill/handler/chakra` — and the damage, move and prepare paths all live in `socket/handler`. **The registry and the pure formulas therefore live in `character/chakra/`**, exactly mirroring the `character/statreset` precedent (a dependency-free, tenant-keyed, in-process singleton consumed from `socket/handler`). The `skill/handler/chakra` package holds only the `Handler`.

2. **No HP re-check at `USE_SKILL`.** Design §5.2 lists `2*HP >= effectiveMaxHP → reject` at the `USE_SKILL` gate. That contradicts PRD FR-1.3 ("the threshold is checked at activation only" — external healing mid-recovery must not cancel the heal) *and* design §3.2's own finding that the client has no post-gate HP re-check. The crafted-client hole design §5.2 was closing (a client that skips prepare) is already closed by the recovery-state presence check, which is strictly stronger. **The `USE_SKILL` gate checks recovery-state presence only.**

3. **`template_gms_12_1.json` gets no edit.** Design §6.3 states v12 and v92 both bind the `CharacterSkillPrepareForeign` writer but not the handler. That is true of v92 only. `template_gms_12_1.json` is a 24-handler stub: it has **no** `CharacterUseSkillHandle`, **no** `CharacterSkillPrepareForeign` writer, and GMS 12 has **no** entry in `docs/packets/registry/` and **no** IDA export — so there is no authority for a `SKILL_EFFECT` opcode on that column and inventing one would be fabrication. v12 is recorded as out-of-reach per PRD FR-9.4 (Task 9) and is not edited. **Only `template_gms_92_1.json` is edited**, at opcode `0x68` (`docs/packets/registry/gms_v92.yaml:2704`, `SKILL_EFFECT` serverbound opcode `104`).

4. **The handler does not broadcast the cast effect.** `socket/handler/character_skill_use.go` already announces `AnnounceSkillUse` (self) and `AnnounceForeignSkillUse` (map) unconditionally after `handler.UseSkill` returns. PRD FR-8.1 is satisfied by that generic path; re-announcing inside the Chakra handler (as design §5.2 sketches, copying `heal.go`) would send the effect twice.

---

## File Structure

**Create**

| File | Responsibility |
|---|---|
| `services/atlas-channel/atlas.com/channel/character/chakra/formula.go` | Pure math: `CanActivate`, `Base`, `Recovery`, `Applied`, `EffectiveMaxHpOrBase`. No imports beyond stdlib. |
| `services/atlas-channel/atlas.com/channel/character/chakra/formula_test.go` | Table tests for the above. |
| `services/atlas-channel/atlas.com/channel/character/chakra/registry.go` | Tenant-keyed, TTL-bounded, in-process recovery-state singleton + sweeper. |
| `services/atlas-channel/atlas.com/channel/character/chakra/registry_test.go` | Expiry, clear, tenant isolation, `-race` concurrency. |
| `services/atlas-channel/atlas.com/channel/skill/handler/chakra/chakra.go` | The `Handler` registered on `skill2.ChiefBanditChakra`: computes and applies the heal, clears state. |
| `services/atlas-channel/atlas.com/channel/skill/handler/chakra/chakra_test.go` | Handler behaviour with seamed deps. |

**Modify**

| File | Change |
|---|---|
| `services/atlas-channel/atlas.com/channel/socket/handler/character_damage_mitigation.go` | `mitigationInput.chakraPct`, `mitigationBreakdown.chakraAmplified`, the first-term prologue in `computeMitigation`. |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_damage.go` | Two new `damageMitigationDeps` seams; read `x` into `chakraPct`; interrupt after damage is applied; extend the debug line. |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_skill_prepare.go` | Chakra branch: activation gate + `Start`. |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_skill_use.go` | Pre-cost recovery-state gate. |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_move.go` | Interrupt on movement. |
| `services/atlas-channel/atlas.com/channel/socket/handler/map_change.go` | Interrupt on map change. |
| `services/atlas-channel/atlas.com/channel/socket/init.go` | Clear on session destroy; start the sweeper. |
| `services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go` | Blank import of the handler package. |
| `services/atlas-configurations/seed-data/templates/template_gms_92_1.json` | Bind `CharacterSkillPrepareHandle` at `0x68`. |
| `services/atlas-data/atlas.com/data/skill/common_test.go` | Pin Chakra's v95 `common` expansion shape. |

---

## Task 1: Pure Chakra math

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/character/chakra/formula.go`
- Test: `services/atlas-channel/atlas.com/channel/character/chakra/formula_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `chakra.CanActivate(hp uint16, maxHp uint16) bool`, `chakra.Base(luck uint32) int32`, `chakra.Recovery(base int32, y int16) int32`, `chakra.Applied(heal int32, hp uint16, maxHp uint16) int16`, `chakra.EffectiveMaxHpOrBase(effective uint32, base uint16) uint16`.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/character/chakra/formula_test.go`:

```go
package chakra

import (
	"math"
	"testing"
)

// TestCanActivateBoundary pins the client's gate (design §3.2):
// HP*100/MaxHP >= 50 rejects. The integer-equivalent form is 2*HP >= MaxHP.
// PRD OQ-9: exactly 50% must NOT activate.
func TestCanActivateBoundary(t *testing.T) {
	tests := []struct {
		name  string
		hp    uint16
		maxHp uint16
		want  bool
	}{
		{"49 percent", 49, 100, true},
		{"exactly 50 percent", 50, 100, false},
		{"51 percent", 51, 100, false},
		{"full hp", 100, 100, false},
		{"zero hp", 0, 100, true},
		{"odd maxhp just under half", 50, 101, true},
		{"odd maxhp at half rounded up", 51, 101, false},
		{"zero maxhp is never castable", 10, 0, false},
		{"max uint16 maxhp under half", 32767, 65535, true},
		{"max uint16 maxhp at half", 32768, 65535, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := CanActivate(tc.hp, tc.maxHp); got != tc.want {
				t.Fatalf("CanActivate(%d, %d) = %v, want %v", tc.hp, tc.maxHp, got, tc.want)
			}
		})
	}
}

// TestCanActivateMatchesClientForm sweeps the integer-equivalence claim
// against the literal client expression floor(HP*100/MaxHP) >= 50 so the
// rewrite cannot drift at an odd MaxHP.
func TestCanActivateMatchesClientForm(t *testing.T) {
	for maxHp := 1; maxHp <= 400; maxHp++ {
		for hp := 0; hp <= maxHp; hp++ {
			clientRejects := hp*100/maxHp >= 50
			if got := CanActivate(uint16(hp), uint16(maxHp)); got == clientRejects {
				t.Fatalf("CanActivate(%d, %d) = %v but the client form rejects = %v", hp, maxHp, got, clientRejects)
			}
		}
	}
}

// TestBase pins the community-sourced base recovery term: 2.9 x effective
// LUK, integer (design §3.4). Deliberately deterministic — no RNG.
func TestBase(t *testing.T) {
	tests := []struct {
		luck uint32
		want int32
	}{
		{0, 0},
		{1, 2},
		{10, 29},
		{100, 290},
		{123, 356},
		{math.MaxUint32, math.MaxInt32},
	}
	for _, tc := range tests {
		if got := Base(tc.luck); got != tc.want {
			t.Fatalf("Base(%d) = %d, want %d", tc.luck, got, tc.want)
		}
	}
}

// TestRecovery pins healAmount = base * y / 100 across the three distinct
// per-version y tables recorded in design §4.2.
func TestRecovery(t *testing.T) {
	tests := []struct {
		name string
		base int32
		y    int16
		want int32
	}{
		{"v48 L1 y=9", 290, 9, 26},
		{"v48 L30 y=200", 290, 200, 580},
		{"v83 L1 y=68", 290, 68, 197},
		{"v83 L30 y=300", 290, 300, 870},
		{"v95 L1 y=120", 290, 120, 348},
		{"v95 L10 y=300", 290, 300, 870},
		{"zero y", 290, 0, 0},
		{"negative y", 290, -5, 0},
		{"zero base", 0, 300, 0},
		{"negative base", -10, 300, 0},
		{"overflow guard", math.MaxInt32, 300, math.MaxInt32},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Recovery(tc.base, tc.y); got != tc.want {
				t.Fatalf("Recovery(%d, %d) = %d, want %d", tc.base, tc.y, got, tc.want)
			}
		})
	}
}

// TestApplied pins the max-HP clamp (FR-3.2) and the never-negative rule
// (FR-3.5): Chakra never pushes HP past max and never applies a damage event.
func TestApplied(t *testing.T) {
	tests := []struct {
		name  string
		heal  int32
		hp    uint16
		maxHp uint16
		want  int16
	}{
		{"fits under cap", 100, 200, 1000, 100},
		{"clamped to missing", 5000, 900, 1000, 100},
		{"exactly missing", 100, 900, 1000, 100},
		{"at full hp", 500, 1000, 1000, 0},
		{"hp above max", 500, 1200, 1000, 0},
		{"zero heal", 0, 100, 1000, 0},
		{"negative heal", -50, 100, 1000, 0},
		{"int16 contract", math.MaxInt32, 0, 65535, math.MaxInt16},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Applied(tc.heal, tc.hp, tc.maxHp); got != tc.want {
				t.Fatalf("Applied(%d, %d, %d) = %d, want %d", tc.heal, tc.hp, tc.maxHp, got, tc.want)
			}
		})
	}
}

// TestEffectiveMaxHpOrBase pins the defensive narrowing: a zero or
// out-of-range effective MaxHp falls back to the character record's base.
func TestEffectiveMaxHpOrBase(t *testing.T) {
	tests := []struct {
		effective uint32
		base      uint16
		want      uint16
	}{
		{0, 4000, 4000},
		{5000, 4000, 5000},
		{math.MaxUint16 + 1, 4000, math.MaxUint16},
		{math.MaxUint32, 4000, math.MaxUint16},
	}
	for _, tc := range tests {
		if got := EffectiveMaxHpOrBase(tc.effective, tc.base); got != tc.want {
			t.Fatalf("EffectiveMaxHpOrBase(%d, %d) = %d, want %d", tc.effective, tc.base, got, tc.want)
		}
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./character/chakra/...`
Expected: FAIL — `undefined: CanActivate` (and the other four).

- [ ] **Step 3: Write the implementation**

Create `services/atlas-channel/atlas.com/channel/character/chakra/formula.go`:

```go
// Package chakra holds the server-side state and math for Chief Bandit
// Chakra (4211001).
//
// It deliberately depends on nothing inside atlas-channel. The recovery
// window is read from socket/handler (the damage, move, map-change and
// skill-prepare/use paths) and written from skill/handler/chakra, and
// skill/handler/* packages already import socket/handler — so a registry
// living under skill/handler would close an import cycle. This mirrors
// character/statreset, which is in-process, tenant-keyed, and consumed the
// same way.
package chakra

import "math"

// CanActivate reports whether Chakra may begin its recovery window.
//
// The client's gate (CUserLocal::DoActiveSkill_Prepare, design §3.2) is
//
//	if (nHP * 100 / nMHP >= 50) return 0;
//
// which is exactly 2*HP >= MaxHP in integer arithmetic — no float, and the
// exactly-50% boundary rejects. A zero MaxHP is treated as never castable
// rather than dividing by zero.
func CanActivate(hp uint16, maxHp uint16) bool {
	if maxHp == 0 {
		return false
	}
	return 2*int32(hp) < int32(maxHp)
}

// Base returns Chakra's base recovery term from the caster's effective LUK.
//
// UNVERIFIED, community-sourced. IDA on all ten available client IDBs
// (design §1, §3.4) proved the client never computes Chakra's HP restore:
// it sends a prepare packet, then a plain USE_SKILL at animation end, and
// renders whatever HP the server reports. No base term exists in any client
// binary or in WZ. 2.9 is the deterministic midpoint of the 2.3-3.5 range
// used by the long-lived open-source server lineage; the derivation and its
// provenance are recorded in docs/tasks/task-213-chakra-hp-restore/design.md
// §3.4. It is kept as a separate function from Recovery so a better-grounded
// term can replace it with a one-function, one-test-file edit.
func Base(luck uint32) int32 {
	v := int64(luck) * 29 / 10
	if v > math.MaxInt32 {
		return math.MaxInt32
	}
	return int32(v)
}

// Recovery returns the HP Chakra restores: the base term scaled by the
// level's WZ `y` (recovery rate percent). Deterministic — no RNG parameter,
// deliberately, so re-introducing randomisation is a signature change that
// forces these tests to be revisited (design §6.4).
func Recovery(base int32, y int16) int32 {
	if base <= 0 || y <= 0 {
		return 0
	}
	v := int64(base) * int64(y) / 100
	if v > math.MaxInt32 {
		return math.MaxInt32
	}
	return int32(v)
}

// Applied clamps a computed heal to the recipient's missing HP and to the
// int16 CHANGE_HP wire contract. Chakra never raises HP above maximum and
// never applies a negative delta (which would land as damage).
func Applied(heal int32, hp uint16, maxHp uint16) int16 {
	if heal <= 0 {
		return 0
	}
	missing := int32(maxHp) - int32(hp)
	if missing <= 0 {
		return 0
	}
	if heal > missing {
		heal = missing
	}
	if heal > math.MaxInt16 {
		return math.MaxInt16
	}
	return int16(heal)
}

// EffectiveMaxHpOrBase narrows the effective MaxHp from
// atlas-effective-stats into the uint16 range, falling back to the
// character record's base MaxHp when the upstream returned zero or an
// out-of-range value. Same defensive strategy as the Heal handler's
// unexported effectiveMaxHpOrBase and atlas-character's resolveEffectiveMax.
func EffectiveMaxHpOrBase(effective uint32, base uint16) uint16 {
	if effective == 0 {
		return base
	}
	if effective > math.MaxUint16 {
		return math.MaxUint16
	}
	return uint16(effective)
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./character/chakra/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/character/chakra/formula.go \
        services/atlas-channel/atlas.com/channel/character/chakra/formula_test.go
git commit -m "feat(task-213): Chakra activation gate and heal formula"
```

---

## Task 2: Recovery-state registry

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/character/chakra/registry.go`
- Test: `services/atlas-channel/atlas.com/channel/character/chakra/registry_test.go`

**Interfaces:**
- Consumes: nothing from Task 1 (same package, separate file).
- Produces: `chakra.Entry{SkillLevel byte; X int16; Y int16; StartedAt time.Time}`, `chakra.TTL`, `chakra.GetRegistry() *Registry`, and the methods `(*Registry).Start(t tenant.Model, characterId uint32, level byte, x int16, y int16, now time.Time)`, `(*Registry).Get(t tenant.Model, characterId uint32, now time.Time) (Entry, bool)`, `(*Registry).Clear(t tenant.Model, characterId uint32) bool`, `(*Registry).Sweep(now time.Time) int`, `(*Registry).StartSweeper(l logrus.FieldLogger, ctx context.Context)`.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/character/chakra/registry_test.go`:

```go
package chakra

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func testTenant(t *testing.T) tenant.Model {
	t.Helper()
	tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tn
}

// newRegistry builds an isolated Registry so tests never share the
// process-wide singleton.
func newRegistry() *Registry {
	return &Registry{entries: make(map[Key]Entry)}
}

func TestStartAndGet(t *testing.T) {
	r := newRegistry()
	tn := testTenant(t)
	now := time.Now()

	r.Start(tn, 42, 3, 99, 92, now)

	e, ok := r.Get(tn, 42, now)
	if !ok {
		t.Fatal("Get after Start returned ok=false, want true")
	}
	if e.SkillLevel != 3 || e.X != 99 || e.Y != 92 {
		t.Fatalf("entry = %+v, want SkillLevel=3 X=99 Y=92", e)
	}
}

func TestGetMissingCharacter(t *testing.T) {
	r := newRegistry()
	tn := testTenant(t)
	if _, ok := r.Get(tn, 42, time.Now()); ok {
		t.Fatal("Get on an empty registry returned ok=true, want false")
	}
}

// TestLazyExpiry pins that correctness does not depend on the sweeper: an
// entry older than TTL reads as absent even though it is still in the map.
func TestLazyExpiry(t *testing.T) {
	r := newRegistry()
	tn := testTenant(t)
	start := time.Now()
	r.Start(tn, 42, 1, 200, 9, start)

	if _, ok := r.Get(tn, 42, start.Add(TTL-time.Millisecond)); !ok {
		t.Fatal("entry just inside TTL read as absent")
	}
	if _, ok := r.Get(tn, 42, start.Add(TTL)); ok {
		t.Fatal("entry exactly at TTL read as present, want expired")
	}
	if _, ok := r.Get(tn, 42, start.Add(TTL+time.Second)); ok {
		t.Fatal("entry past TTL read as present, want expired")
	}
}

func TestClearReportsPresence(t *testing.T) {
	r := newRegistry()
	tn := testTenant(t)
	r.Start(tn, 42, 1, 99, 68, time.Now())

	if !r.Clear(tn, 42) {
		t.Fatal("Clear on a present entry returned false, want true")
	}
	if r.Clear(tn, 42) {
		t.Fatal("Clear on an absent entry returned true, want false")
	}
	if _, ok := r.Get(tn, 42, time.Now()); ok {
		t.Fatal("entry still present after Clear")
	}
}

// TestTenantIsolation pins that the same characterId in two tenants are two
// independent windows (NFR: multi-tenancy).
func TestTenantIsolation(t *testing.T) {
	r := newRegistry()
	a := testTenant(t)
	b := testTenant(t)
	now := time.Now()

	r.Start(a, 42, 1, 200, 9, now)

	if _, ok := r.Get(b, 42, now); ok {
		t.Fatal("tenant b sees tenant a's entry")
	}
	if r.Clear(b, 42) {
		t.Fatal("Clear in tenant b reported tenant a's entry")
	}
	if _, ok := r.Get(a, 42, now); !ok {
		t.Fatal("tenant a's entry was disturbed by tenant b")
	}
}

// TestSweepEvictsOnlyExpired pins that the sweeper bounds memory without
// touching live windows.
func TestSweepEvictsOnlyExpired(t *testing.T) {
	r := newRegistry()
	tn := testTenant(t)
	old := time.Now()
	r.Start(tn, 1, 1, 99, 68, old)
	r.Start(tn, 2, 1, 99, 68, old.Add(TTL))

	if n := r.Sweep(old.Add(TTL)); n != 1 {
		t.Fatalf("Sweep evicted %d entries, want 1", n)
	}
	if _, ok := r.Get(tn, 2, old.Add(TTL)); !ok {
		t.Fatal("Sweep evicted a live entry")
	}
}

// TestConcurrentAccess is the -race guard: the registry is written by the
// prepare path and read/cleared by the damage, move and use paths plus the
// sweeper, all concurrently.
func TestConcurrentAccess(t *testing.T) {
	r := newRegistry()
	tn := testTenant(t)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		id := uint32(i%3 + 1)
		wg.Add(4)
		go func() { defer wg.Done(); r.Start(tn, id, 1, 99, 68, time.Now()) }()
		go func() { defer wg.Done(); r.Get(tn, id, time.Now()) }()
		go func() { defer wg.Done(); r.Clear(tn, id) }()
		go func() { defer wg.Done(); r.Sweep(time.Now()) }()
	}
	wg.Wait()
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./character/chakra/...`
Expected: FAIL — `undefined: Registry`, `undefined: Key`, `undefined: Entry`, `undefined: TTL`.

- [ ] **Step 3: Write the implementation**

Create `services/atlas-channel/atlas.com/channel/character/chakra/registry.go`:

```go
package chakra

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-routine/routine"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TTL bounds a Chakra recovery window server-side.
//
// It is a SAFETY bound, not a timing model. The server does not simulate the
// window: the client opens it with a prepare packet and closes it by sending
// USE_SKILL at animation end. The client's own animation is 1500 ms (the
// `prepare` node is 15 frames x 100 ms delay, identical at v48 and v83), and
// the client writes its own outer bound of 5000 ms on the Chakra path
// (design §4.3). 5 s matches that bound and leaves headroom over 1500 ms for
// latency. Not level-dependent, not version-dependent.
const TTL = 5000 * time.Millisecond

// sweepInterval is how often the background sweeper evicts expired entries.
// Correctness does not depend on it — Get applies lazy expiry — so it only
// bounds memory.
const sweepInterval = 30 * time.Second

// Key scopes a recovery window to one character in one tenant.
type Key struct {
	Tenant      tenant.Model
	CharacterId uint32
}

// Entry is the snapshot taken when the recovery window opens.
//
// X and Y are captured at prepare time from the caster's REAL skill-book
// level, so the damage path never needs an atlas-data round trip per hit
// (PRD FR-2.4 / NFR hot-path cost) and a mid-window skill-book change cannot
// desync the damage factor from the heal.
type Entry struct {
	SkillLevel byte
	X          int16 // WZ `x` — damage-taken percent (design §4.1)
	Y          int16 // WZ `y` — recovery-rate percent (design §4.1)
	StartedAt  time.Time
}

type Registry struct {
	mutex   sync.RWMutex
	entries map[Key]Entry
}

var (
	registry *Registry
	once     sync.Once
)

// GetRegistry returns the process-wide recovery-state registry.
//
// In-process is the whole view, not a shard of it: a character's socket
// session lives on exactly one atlas-channel pod, and the caster is standing
// still in one map on one channel for the entire window.
func GetRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{entries: make(map[Key]Entry)}
	})
	return registry
}

// Start opens (or restarts) a recovery window.
func (r *Registry) Start(t tenant.Model, characterId uint32, level byte, x int16, y int16, now time.Time) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.entries[Key{Tenant: t, CharacterId: characterId}] = Entry{
		SkillLevel: level,
		X:          x,
		Y:          y,
		StartedAt:  now,
	}
}

// Get returns the live recovery window, if any. An entry at or past TTL
// reads as absent (lazy expiry) regardless of whether the sweeper has run.
func (r *Registry) Get(t tenant.Model, characterId uint32, now time.Time) (Entry, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	e, ok := r.entries[Key{Tenant: t, CharacterId: characterId}]
	if !ok || now.Sub(e.StartedAt) >= TTL {
		return Entry{}, false
	}
	return e, true
}

// Clear ends a recovery window and reports whether one was open. Callers
// name the reason (damaged / moved / map change / disconnect / completed) in
// their own log line, so no reason is stored here.
func (r *Registry) Clear(t tenant.Model, characterId uint32) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	k := Key{Tenant: t, CharacterId: characterId}
	if _, ok := r.entries[k]; !ok {
		return false
	}
	delete(r.entries, k)
	return true
}

// Sweep evicts expired entries and returns how many were removed.
func (r *Registry) Sweep(now time.Time) int {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	n := 0
	for k, e := range r.entries {
		if now.Sub(e.StartedAt) >= TTL {
			delete(r.entries, k)
			n++
		}
	}
	return n
}

// StartSweeper runs the eviction loop until ctx is done. Spawned via
// routine.Go per tools/goroutine-guard.sh.
func (r *Registry) StartSweeper(l logrus.FieldLogger, ctx context.Context) {
	routine.Go(l, ctx, func(c context.Context) {
		ticker := time.NewTicker(sweepInterval)
		defer ticker.Stop()
		for {
			select {
			case <-c.Done():
				return
			case <-ticker.C:
				if n := r.Sweep(time.Now()); n > 0 {
					l.Debugf("Chakra recovery sweeper evicted [%d] expired entries.", n)
				}
			}
		}
	})
}
```

If the `routine` import path or `tenant.Create` signature does not match, correct them from the live source (`socket/init.go` for `routine.Go`, `character/statreset/registry.go` for the tenant-keyed map) rather than guessing.

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./character/chakra/...`
Expected: PASS, no race reported.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/character/chakra/registry.go \
        services/atlas-channel/atlas.com/channel/character/chakra/registry_test.go
git commit -m "feat(task-213): Chakra recovery-state registry with lazy TTL expiry"
```

---

## Task 3: The damage factor as the first mitigation term

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_damage_mitigation.go`
- Test: `services/atlas-channel/atlas.com/channel/socket/handler/character_damage_mitigation_test.go`

**Interfaces:**
- Consumes: nothing from Tasks 1-2 (`chakraPct` is a plain `int32` field).
- Produces: `mitigationInput.chakraPct int32`, `mitigationBreakdown.chakraAmplified int32`.

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-channel/atlas.com/channel/socket/handler/character_damage_mitigation_test.go`:

```go
// TestChakraFactor pins design §3.1: the client rewrites the raw damage by
// the caster's Chakra level `x` percent, with a <= 1 -> 1 floor, and does so
// with NO gate on the attack source. On GMS 12/48 x is 200..112 so the term
// AMPLIFIES; on GMS 61+ x is 99..70 so it REDUCES; on GMS 95 x is 96..60.
// The WZ data carries the direction — there is deliberately no version gate.
func TestChakraFactor(t *testing.T) {
	tests := []struct {
		name      string
		raw       int32
		chakraPct int32
		wantHp    int32
	}{
		{"no window", 500, 0, 500},
		{"v48 L1 x=200 amplifies", 500, 200, 1000},
		{"v48 L30 x=112 amplifies", 500, 112, 560},
		{"v83 L1 x=99 reduces", 500, 99, 495},
		{"v83 L30 x=70 reduces", 500, 70, 350},
		{"v95 L10 x=60 reduces", 500, 60, 300},
		{"floor at one", 1, 60, 1},
		{"floor applies to the product not the input", 2, 50, 1},
		{"rounding truncates", 7, 70, 4},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := computeMitigation(mitigationInput{rawDamage: tc.raw, chakraPct: tc.chakraPct}, mobInfo{})
			if got.hpLoss != tc.wantHp {
				t.Fatalf("hpLoss = %d, want %d", got.hpLoss, tc.wantHp)
			}
		})
	}
}

// TestChakraFactorAppliedBeforeEveryOtherTerm pins design §3.3 / PRD FR-4.3:
// in CUserLocal::SetDamaged the Chakra branch writes back to the same stack
// slot that carries the damage, and every task-157 term reads that slot
// afterwards. Applying it after Achilles would produce different numbers.
func TestChakraFactorAppliedBeforeEveryOtherTerm(t *testing.T) {
	in := mitigationInput{
		rawDamage:            1000,
		chakraPct:            200,
		achillesPermille:     200,
		comboBarrierPermille: 100,
		magicGuardPct:        50,
		currentMP:            5000,
		mobSourced:           true,
	}
	got := computeMitigation(in, mobInfo{})

	// Same chain, with the Chakra factor already folded into rawDamage and
	// the term disabled — the result must be identical.
	pre := in
	pre.rawDamage = 2000
	pre.chakraPct = 0
	want := computeMitigation(pre, mobInfo{})

	if got.hpLoss != want.hpLoss || got.mpLoss != want.mpLoss {
		t.Fatalf("(hpLoss,mpLoss) = (%d,%d), want (%d,%d) — Chakra must be applied to raw damage before every other term",
			got.hpLoss, got.mpLoss, want.hpLoss, want.mpLoss)
	}
	if got.breakdown.achillesReduce == want.breakdown.achillesReduce && got.breakdown.achillesReduce == 0 {
		t.Fatal("test is not exercising Achilles")
	}
}

// TestChakraBreakdownReportsPostFactorDamage pins the observability
// requirement: without the post-factor value in the breakdown, "Chakra did
// nothing" is undiagnosable from logs.
func TestChakraBreakdownReportsPostFactorDamage(t *testing.T) {
	got := computeMitigation(mitigationInput{rawDamage: 500, chakraPct: 200}, mobInfo{})
	if got.breakdown.chakraAmplified != 1000 {
		t.Fatalf("breakdown.chakraAmplified = %d, want 1000", got.breakdown.chakraAmplified)
	}
	none := computeMitigation(mitigationInput{rawDamage: 500}, mobInfo{})
	if none.breakdown.chakraAmplified != 0 {
		t.Fatalf("breakdown.chakraAmplified = %d with no window, want 0", none.breakdown.chakraAmplified)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run TestChakra -v`
Expected: FAIL — `unknown field chakraPct in struct literal of type mitigationInput`.

- [ ] **Step 3: Write the implementation**

In `character_damage_mitigation.go`, add to `mitigationInput` immediately after the `achillesPermille` / `manaReflectPct` block (before the "Version gates" block):

```go
	// chakraPct is the WZ `x` of the caster's active Chakra recovery window
	// (0 = no window). CUserLocal::SetDamaged rewrites the raw damage by
	// this factor before every other term reads it (design §3.3), so it is
	// applied first, and it is deliberately NOT gated on the attack source —
	// there is no attackIdx, mob-sourced or magic/physical test around the
	// client's branch.
	//
	// x > 100 amplifies (GMS 12/48: 200..112); x < 100 reduces (GMS 61+:
	// 99..70; GMS 95: 96..60). The WZ data carries the direction, so there
	// is no version gate here and adding one would be the bug (design §4.2).
	chakraPct int32
```

Add to `mitigationBreakdown`:

```go
	chakraAmplified    int32
```

In `computeMitigation`, insert the prologue immediately after the existing `raw <= 0` early return and before the Achilles block:

```go
	if in.chakraPct > 0 {
		raw = raw * in.chakraPct / 100
		// The client's floor is `<= 1 -> 1`, deliberately not `< 1`, and it
		// applies to the multiplied value rather than the original.
		if raw <= 1 {
			raw = 1
		}
		r.breakdown.chakraAmplified = raw
	}
```

`r.breakdown` is overwritten wholesale further down, so also add `chakraAmplified: r.breakdown.chakraAmplified,` to the `mitigationBreakdown{...}` literal near the end of the function, alongside `achillesReduce:` etc.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run TestChakra -v`
Expected: PASS.

Run the whole package to confirm no task-157 regression: `go test ./socket/handler/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_damage_mitigation.go \
        services/atlas-channel/atlas.com/channel/socket/handler/character_damage_mitigation_test.go
git commit -m "feat(task-213): apply the Chakra damage factor first in computeMitigation"
```

---

## Task 4: Wire the damage path — read the window, interrupt after the hit

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_damage.go`
- Test: `services/atlas-channel/atlas.com/channel/socket/handler/character_damage_test.go`

**Interfaces:**
- Consumes: `chakra.Entry`, `chakra.GetRegistry()`, `(*Registry).Get`, `(*Registry).Clear` (Task 2); `mitigationInput.chakraPct` (Task 3).
- Produces: `damageMitigationDeps.getChakra func(characterId uint32) (chakra.Entry, bool)` and `damageMitigationDeps.clearChakra func(characterId uint32) bool`.

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-channel/atlas.com/channel/socket/handler/character_damage_test.go` (match the file's existing helpers for building `p`, `c` and `deps`; if it constructs `damageMitigationDeps` via a helper, extend that helper instead of hand-rolling one here):

```go
// TestDamageAppliesChakraFactorAndInterrupts pins PRD FR-4.5 / FR-5.2: the
// interrupting hit itself takes the Chakra factor, and the window is closed
// afterwards so the pending heal cannot fire.
func TestDamageAppliesChakraFactorAndInterrupts(t *testing.T) {
	var appliedHp int16
	cleared := false
	deps := newTestDamageDeps()
	deps.changeHP = func(_ field.Model, _ uint32, amount int16) error { appliedHp = amount; return nil }
	deps.getChakra = func(_ uint32) (chakra.Entry, bool) {
		return chakra.Entry{SkillLevel: 1, X: 200, Y: 9, StartedAt: time.Now()}, true
	}
	deps.clearChakra = func(_ uint32) bool { cleared = true; return true }

	processDamageTaken(testLogger(t), testTenantModel(t), testField(), testDamagePacket(500), testCharacter(), deps)

	if appliedHp != -1000 {
		t.Fatalf("applied HP delta = %d, want -1000 (500 raw x 200%%)", appliedHp)
	}
	if !cleared {
		t.Fatal("Chakra window was not cleared by the damaging hit")
	}
}

// TestDamageWithoutChakraWindowDoesNotInterrupt pins that the interrupt is
// only attempted when a window is actually open.
func TestDamageWithoutChakraWindowDoesNotInterrupt(t *testing.T) {
	var appliedHp int16
	cleared := false
	deps := newTestDamageDeps()
	deps.changeHP = func(_ field.Model, _ uint32, amount int16) error { appliedHp = amount; return nil }
	deps.getChakra = func(_ uint32) (chakra.Entry, bool) { return chakra.Entry{}, false }
	deps.clearChakra = func(_ uint32) bool { cleared = true; return true }

	processDamageTaken(testLogger(t), testTenantModel(t), testField(), testDamagePacket(500), testCharacter(), deps)

	if appliedHp != -500 {
		t.Fatalf("applied HP delta = %d, want -500 (unfactored)", appliedHp)
	}
	if cleared {
		t.Fatal("clearChakra was called with no window open")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run TestDamage.*Chakra -v`
Expected: FAIL — `unknown field getChakra in struct literal of type damageMitigationDeps`.

- [ ] **Step 3: Write the implementation**

Add the import `"atlas-channel/character/chakra"` and `"time"` to `character_damage.go`.

Extend `damageMitigationDeps` with:

```go
	getChakra   func(characterId uint32) (chakra.Entry, bool)
	clearChakra func(characterId uint32) bool
```

In `CharacterDamageHandleFunc`, add to the `deps := damageMitigationDeps{...}` literal (the `t := tenant.MustFromContext(ctx)` line already sits just above it):

```go
		getChakra: func(characterId uint32) (chakra.Entry, bool) {
			return chakra.GetRegistry().Get(t, characterId, time.Now())
		},
		clearChakra: func(characterId uint32) bool {
			return chakra.GetRegistry().Clear(t, characterId)
		},
```

In `processDamageTaken`, immediately after the `raw, adjusted := clampDamage(p.Damage())` block and its warn, insert:

```go
	// Chakra recovery window: the level's WZ `x` rewrites the raw damage
	// before every other mitigation term (design §3.3). The lookup is an
	// in-process RWMutex read — no cross-service call on the per-hit path
	// (PRD FR-2.4 / NFR hot-path cost).
	var chakraPct int32
	chakraActive := false
	if deps.getChakra != nil {
		if entry, ok := deps.getChakra(characterId); ok {
			chakraActive = true
			chakraPct = int32(entry.X)
		}
	}
```

Add `chakraPct: chakraPct,` to the `in := mitigationInput{...}` literal.

Extend the existing mitigation debug line's format string with `chakra [%d]` and append `result.breakdown.chakraAmplified` to its arguments, so the pre- and post-factor damage are both on one line.

After the `if result.reflect.amount > 0 { ... }` block at the end of `processDamageTaken`, append:

```go
	// A hit cancels the pending heal (PRD FR-5.2). Ordering is deliberate:
	// the factor is applied and the damage lands FIRST, so the interrupting
	// hit itself carries the increased damage (PRD FR-4.5). MP is not
	// refunded — nothing was spent, because the generic cost block only runs
	// when the USE_SKILL packet arrives (design §2).
	if chakraActive && deps.clearChakra != nil {
		if deps.clearChakra(characterId) {
			l.Debugf("Chakra recovery window for character [%d] interrupted by damage; pending heal cancelled.", characterId)
		}
	}
```

Death (PRD FR-5.3) needs no separate hook: the killing hit runs this same interrupt, and the pending heal only ever fires from a `USE_SKILL` packet, which `character_skill_use.go` already rejects for a character at 0 HP.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./socket/handler/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_damage.go \
        services/atlas-channel/atlas.com/channel/socket/handler/character_damage_test.go
git commit -m "feat(task-213): feed the Chakra window into the damage path and interrupt on hit"
```

---

## Task 5: Open the recovery window on the skill-prepare packet

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_skill_prepare.go`
- Test: `services/atlas-channel/atlas.com/channel/socket/handler/character_skill_prepare_test.go`

**Interfaces:**
- Consumes: `chakra.CanActivate`, `chakra.EffectiveMaxHpOrBase` (Task 1); `chakra.GetRegistry().Start` (Task 2).
- Produces: `isChakraCast(ctx context.Context, skillId uint32) bool`, `chakraPrepareDeps` and `startChakraRecovery(l logrus.FieldLogger, t tenant.Model, characterId uint32, worldId world.Id, channelId channel.Id, c character.Model, skillId uint32, deps chakraPrepareDeps)`.

- [ ] **Step 1: Write the failing test**

Create/extend `services/atlas-channel/atlas.com/channel/socket/handler/character_skill_prepare_test.go`:

```go
// TestStartChakraRecoveryGate pins PRD FR-1.1/FR-1.2/FR-1.4: below 50% of
// EFFECTIVE max HP the window opens; at or above it nothing happens at all —
// no window, no MP, no cooldown (the client sends no USE_SKILL, and even a
// crafted one is rejected by the use-skill gate in Task 6).
func TestStartChakraRecoveryGate(t *testing.T) {
	tests := []struct {
		name         string
		hp           uint16
		baseMaxHp    uint16
		effectiveMax uint32
		wantStarted  bool
	}{
		{"49 percent of effective", 490, 500, 1000, true},
		{"exactly 50 percent of effective", 500, 500, 1000, false},
		{"51 percent of effective", 510, 500, 1000, false},
		{"effective stats unavailable falls back to base", 240, 500, 0, true},
		{"base fallback rejects at half base", 250, 500, 0, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			started := false
			deps := chakraPrepareDeps{
				skillLevel:      func() byte { return 3 },
				effectiveMaxHp:  func() uint32 { return tc.effectiveMax },
				effectXY:        func(byte) (int16, int16, error) { return 99, 92, nil },
				start:           func(byte, int16, int16) { started = true },
			}
			startChakraRecoveryWith(testLogger(t), tc.hp, tc.baseMaxHp, deps)
			if started != tc.wantStarted {
				t.Fatalf("window started = %v, want %v", started, tc.wantStarted)
			}
		})
	}
}

// TestStartChakraRecoverySnapshotsWzValues pins that x and y are captured
// from the caster's REAL skill-book level (design §5.1), not from the wire.
func TestStartChakraRecoverySnapshotsWzValues(t *testing.T) {
	var gotLevel byte
	var gotX, gotY int16
	deps := chakraPrepareDeps{
		skillLevel:     func() byte { return 7 },
		effectiveMaxHp: func() uint32 { return 1000 },
		effectXY: func(level byte) (int16, int16, error) {
			if level != 7 {
				t.Fatalf("effect looked up at level %d, want the skill-book level 7", level)
			}
			return 99, 116, nil
		},
		start: func(l byte, x int16, y int16) { gotLevel, gotX, gotY = l, x, y },
	}
	startChakraRecoveryWith(testLogger(t), 100, 500, deps)
	if gotLevel != 7 || gotX != 99 || gotY != 116 {
		t.Fatalf("(level,x,y) = (%d,%d,%d), want (7,99,116)", gotLevel, gotX, gotY)
	}
}

// TestStartChakraRecoveryUnknownSkill pins that a caster who does not own
// the skill opens no window.
func TestStartChakraRecoveryUnknownSkill(t *testing.T) {
	started := false
	deps := chakraPrepareDeps{
		skillLevel:     func() byte { return 0 },
		effectiveMaxHp: func() uint32 { return 1000 },
		effectXY:       func(byte) (int16, int16, error) { return 99, 92, nil },
		start:          func(byte, int16, int16) { started = true },
	}
	startChakraRecoveryWith(testLogger(t), 100, 500, deps)
	if started {
		t.Fatal("window opened for a caster who does not own Chakra")
	}
}

// TestStartChakraRecoveryEffectLookupFailure pins that an atlas-data miss
// opens no window rather than opening one with zeroed x/y (a zero x would
// zero every hit the caster takes).
func TestStartChakraRecoveryEffectLookupFailure(t *testing.T) {
	started := false
	deps := chakraPrepareDeps{
		skillLevel:     func() byte { return 3 },
		effectiveMaxHp: func() uint32 { return 1000 },
		effectXY:       func(byte) (int16, int16, error) { return 0, 0, errors.New("upstream down") },
		start:          func(byte, int16, int16) { started = true },
	}
	startChakraRecoveryWith(testLogger(t), 100, 500, deps)
	if started {
		t.Fatal("window opened despite an effect lookup failure")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run TestStartChakra -v`
Expected: FAIL — `undefined: chakraPrepareDeps`, `undefined: startChakraRecoveryWith`.

- [ ] **Step 3: Write the implementation**

In `character_skill_prepare.go`, add imports `"atlas-channel/character/chakra"`, `dataskill "atlas-channel/data/skill"`, `"atlas-channel/effective_stats"`, `"time"`.

Add the seam type and pure core:

```go
// chakraPrepareDeps seams the three lookups the Chakra prepare gate needs so
// the gate itself is directly unit-testable without a live character,
// effective-stats or atlas-data service.
type chakraPrepareDeps struct {
	skillLevel     func() byte
	effectiveMaxHp func() uint32
	effectXY       func(level byte) (int16, int16, error)
	start          func(level byte, x int16, y int16)
}

// startChakraRecoveryWith is the activation gate (PRD FR-1). It runs at
// PREPARE time only; there is no post-gate HP re-check anywhere in the
// client (design §3.2) and PRD FR-1.3 forbids one server-side, so external
// healing that lifts the caster to >= 50% mid-window must not cancel the
// pending heal.
func startChakraRecoveryWith(l logrus.FieldLogger, hp uint16, baseMaxHp uint16, deps chakraPrepareDeps) {
	level := deps.skillLevel()
	if level == 0 {
		l.Debugf("Chakra prepare from a caster who does not own the skill; ignoring.")
		return
	}
	maxHp := chakra.EffectiveMaxHpOrBase(deps.effectiveMaxHp(), baseMaxHp)
	if !chakra.CanActivate(hp, maxHp) {
		l.Debugf("Chakra prepare rejected: hp [%d] is not below half of effective max hp [%d]. No window, no MP, no cooldown.", hp, maxHp)
		return
	}
	x, y, err := deps.effectXY(level)
	if err != nil {
		l.WithError(err).Warnf("Chakra prepare: unable to load the skill effect at level [%d]; no recovery window opened.", level)
		return
	}
	deps.start(level, x, y)
	l.Debugf("Chakra recovery window opened at level [%d] (x=[%d] damage-taken %%, y=[%d] recovery %%), hp [%d] of effective max [%d].", level, x, y, hp, maxHp)
}
```

Add the identity predicate:

```go
// isChakraCast resolves a raw wire id to its version-blind Identity and
// reports whether it is Chief Bandit Chakra. Never compares the wire id
// directly (PRD FR-9.1).
func isChakraCast(ctx context.Context, skillId uint32) bool {
	t := tenant.MustFromContext(ctx)
	set := constants.For(t.Region(), t.MajorVersion(), t.MinorVersion())
	id, ok := set.Skill.Resolve(skill.Id(skillId))
	return ok && skill.IsIdentity(id, skill.ChiefBanditChakra)
}
```

In `CharacterSkillPrepareHandleFunc`, immediately after `c` is loaded and before the `shouldBroadcastKeydown` call, insert:

```go
		if isChakraCast(ctx, info.SkillId()) {
			t := tenant.MustFromContext(ctx)
			startChakraRecoveryWith(l, c.Hp(), c.MaxHp(), chakraPrepareDeps{
				skillLevel: func() byte { return skillLevelOf(c.Skills(), skill.Id(info.SkillId())) },
				effectiveMaxHp: func() uint32 {
					stats, sErr := effective_stats.NewProcessor(l, ctx).GetByCharacterId(s.Field().WorldId(), s.Field().ChannelId(), s.CharacterId())
					if sErr != nil {
						l.WithError(sErr).Warnf("Chakra prepare: effective stats unavailable for character [%d]; using base max hp.", s.CharacterId())
						return 0
					}
					return stats.MaxHp
				},
				effectXY: func(level byte) (int16, int16, error) {
					e, eErr := dataskill.NewProcessor(l, ctx).GetEffect(info.SkillId(), level)
					if eErr != nil {
						return 0, 0, eErr
					}
					return e.X(), e.Y(), nil
				},
				start: func(level byte, x int16, y int16) {
					chakra.GetRegistry().Start(t, s.CharacterId(), level, x, y, time.Now())
				},
			})
			// Chakra is NOT a keydown skill on any version (design §3.5:
			// is_keydown_skill excludes 4211001 on v83 and v95, reproducing
			// task-161). No foreign prepare broadcast.
			return
		}
```

`skillLevelOf` already exists in this package at `character_skill_use.go` — reuse it, do not redefine it.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run TestStartChakra -v` then `go test -race ./socket/handler/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_skill_prepare.go \
        services/atlas-channel/atlas.com/channel/socket/handler/character_skill_prepare_test.go
git commit -m "feat(task-213): open the Chakra recovery window on the skill-prepare packet"
```

---

## Task 6: Pre-cost gate on `USE_SKILL`

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_skill_use.go`
- Test: `services/atlas-channel/atlas.com/channel/socket/handler/character_skill_use_test.go`

**Interfaces:**
- Consumes: `chakra.GetRegistry().Get` (Task 2); `isChakraCast` (Task 5).
- Produces: `chakraUseBlocked(hasWindow bool) bool`.

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-channel/atlas.com/channel/socket/handler/character_skill_use_test.go`:

```go
// TestChakraUseBlocked pins PRD FR-1.4 / FR-5.4: a USE_SKILL for Chakra is
// honoured only when a recovery window is open. No window means the cast was
// never prepared (a crafted client skipping the prepare packet) or was
// interrupted — either way it is rejected BEFORE handler.UseSkill, so no MP
// is charged and no cooldown is applied.
//
// Deliberately NOT re-checking HP here: the client has no post-gate HP
// re-check (design §3.2) and PRD FR-1.3 requires the threshold be evaluated
// at activation only, so external healing mid-window must not cancel the
// heal. The window-presence check already closes the crafted-client hole a
// second HP check would have covered, and closes it more tightly.
func TestChakraUseBlocked(t *testing.T) {
	if chakraUseBlocked(true) {
		t.Fatal("an open recovery window must not block the cast")
	}
	if !chakraUseBlocked(false) {
		t.Fatal("a missing recovery window must block the cast")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run TestChakraUseBlocked -v`
Expected: FAIL — `undefined: chakraUseBlocked`.

- [ ] **Step 3: Write the implementation**

Add imports `"atlas-channel/character/chakra"` to `character_skill_use.go` (`time` is already imported).

Add:

```go
// chakraUseBlocked reports whether a Chakra USE_SKILL must be rejected.
//
// The only condition is "no open recovery window": either the client never
// sent the prepare packet, or the window was interrupted by damage,
// movement, a map change or the TTL. There is deliberately no second HP
// check — the client has none (design §3.2) and PRD FR-1.3 requires the
// 50% threshold be evaluated at activation only.
func chakraUseBlocked(hasWindow bool) bool {
	return !hasWindow
}
```

In `CharacterUseSkillHandleFunc`, immediately after `castId, castIdOk := set.Skill.Resolve(...)` and before the Enrage block, insert:

```go
		// Chakra: reject before handler.UseSkill so a cast with no open
		// recovery window spends no MP and applies no cooldown, following
		// the Enrage precedent below (UseSkill charges both before it
		// dispatches to the per-skill registry — skill/handler/common.go).
		if castIdOk && skill.IsIdentity(castId, skill.ChiefBanditChakra) {
			_, hasWindow := chakra.GetRegistry().Get(t, s.CharacterId(), time.Now())
			if chakraUseBlocked(hasWindow) {
				l.Debugf("Character [%d] sent Chakra USE_SKILL with no open recovery window (never prepared, or interrupted); rejecting.", s.CharacterId())
				if aerr := enableActions(l)(ctx)(wp)(s); aerr != nil {
					l.WithError(aerr).Errorf("Unable to write [%s] for character [%d].", statpkt.StatChangedWriter, s.CharacterId())
				}
				return
			}
		}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./socket/handler/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_skill_use.go \
        services/atlas-channel/atlas.com/channel/socket/handler/character_skill_use_test.go
git commit -m "feat(task-213): reject a Chakra cast with no open recovery window before any cost"
```

---

## Task 7: The Chakra handler — apply the heal

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/skill/handler/chakra/chakra.go`
- Test: `services/atlas-channel/atlas.com/channel/skill/handler/chakra/chakra_test.go`
- Modify: `services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go`

**Interfaces:**
- Consumes: `chakrastate.Base`, `.Recovery`, `.Applied`, `.EffectiveMaxHpOrBase` (Task 1); `chakrastate.Entry`, `.GetRegistry()` (Task 2); `channelhandler.Register` (`skill/handler/registry.go:36`).
- Produces: `chakra.Apply` (a `channelhandler.Handler`), and the pure core `chakra.healDelta(entry chakrastate.Entry, luck uint32, hp uint16, maxHp uint16) int16`.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/skill/handler/chakra/chakra_test.go`:

```go
package chakra

import (
	"testing"
	"time"

	chakrastate "atlas-channel/character/chakra"

	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"

	channelhandler "atlas-channel/skill/handler"
)

// TestHealDelta pins the end-to-end heal: base = 2.9 x effective LUK,
// scaled by the window's snapshotted y, clamped to missing HP.
func TestHealDelta(t *testing.T) {
	tests := []struct {
		name  string
		y     int16
		luck  uint32
		hp    uint16
		maxHp uint16
		want  int16
	}{
		{"v83 L1 y=68", 68, 100, 100, 1000, 197},
		{"v83 L30 y=300", 300, 100, 100, 1000, 870},
		{"v48 L1 y=9", 9, 100, 100, 1000, 26},
		{"v48 L30 y=200", 200, 100, 100, 1000, 580},
		{"v95 L10 y=300", 300, 100, 100, 1000, 870},
		{"clamped to missing hp", 300, 100, 950, 1000, 50},
		{"at full hp", 300, 100, 1000, 1000, 0},
		{"zero luck", 300, 0, 100, 1000, 0},
		{"zero y", 0, 100, 100, 1000, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := chakrastate.Entry{SkillLevel: 1, X: 99, Y: tc.y, StartedAt: time.Now()}
			if got := healDelta(e, tc.luck, tc.hp, tc.maxHp); got != tc.want {
				t.Fatalf("healDelta = %d, want %d", got, tc.want)
			}
		})
	}
}

// TestRegisteredOnIdentity pins PRD FR-9.1: the handler is installed on the
// version-blind identity, so one registration covers all eleven provisioned
// versions without a raw wire-id comparison anywhere.
func TestRegisteredOnIdentity(t *testing.T) {
	if _, ok := channelhandler.Lookup(skill2.ChiefBanditChakra); !ok {
		t.Fatal("no Handler registered for skill2.ChiefBanditChakra")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./skill/handler/chakra/...`
Expected: FAIL — `undefined: healDelta` (package does not build).

- [ ] **Step 3: Write the implementation**

Create `services/atlas-channel/atlas.com/channel/skill/handler/chakra/chakra.go`:

```go
// Package chakra is the per-skill USE_SKILL handler for Chief Bandit Chakra
// (4211001). The recovery window it consumes is opened on the skill-prepare
// packet (socket/handler/character_skill_prepare.go) and lives in
// atlas-channel/character/chakra.
package chakra

import (
	"atlas-channel/character"
	chakrastate "atlas-channel/character/chakra"
	"atlas-channel/data/skill/effect"
	"atlas-channel/effective_stats"
	channelhandler "atlas-channel/skill/handler"
	"atlas-channel/socket/writer"
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func init() {
	channelhandler.Register(skill2.ChiefBanditChakra, Apply)
}

// healDelta computes the HP Chakra restores on this completion, clamped to
// the caster's missing HP.
//
// The recovery rate comes from the WINDOW's snapshot, not from the effect
// the USE_SKILL packet resolved: the window captured `y` at prepare time
// from the caster's real skill-book level, so a client that lies about its
// level on the second packet cannot inflate the heal.
func healDelta(entry chakrastate.Entry, luck uint32, hp uint16, maxHp uint16) int16 {
	return chakrastate.Applied(
		chakrastate.Recovery(chakrastate.Base(luck), entry.Y),
		hp,
		maxHp,
	)
}

// Apply is the Chakra handler installed in the per-skill registry.
//
// It runs at the COMPLETION of the recovery window — the client sends this
// USE_SKILL at the end of the 1500 ms prepare animation (design §2), which
// is why the heal lands here and not at keypress (PRD FR-3.4).
//
// It deliberately does NOT:
//   - charge MP or apply cooldown — the generic UseSkill block owns both and
//     has already run by the time this is dispatched (PRD FR-8.2/8.3);
//   - award experience — Chakra is self-only (PRD FR-8.4);
//   - broadcast the cast effect — character_skill_use.go already announces
//     AnnounceSkillUse and AnnounceForeignSkillUse unconditionally after
//     UseSkill returns, so re-announcing here would send it twice.
func Apply(l logrus.FieldLogger) func(ctx context.Context) func(
	wp writer.Producer,
	f field.Model, characterId uint32,
	info packetmodel.SkillUsageInfo, e effect.Model,
) error {
	return func(ctx context.Context) func(
		wp writer.Producer,
		f field.Model, characterId uint32,
		info packetmodel.SkillUsageInfo, e effect.Model,
	) error {
		return func(
			wp writer.Producer,
			f field.Model, characterId uint32,
			info packetmodel.SkillUsageInfo, e effect.Model,
		) error {
			t := tenant.MustFromContext(ctx)
			reg := chakrastate.GetRegistry()

			entry, ok := reg.Get(t, characterId, time.Now())
			if !ok {
				// The pre-cost gate in character_skill_use.go rejects this
				// case before UseSkill runs; reaching it here means the
				// window expired between the two checks.
				l.Debugf("Chakra: no open recovery window for character [%d] at completion; no heal applied.", characterId)
				return nil
			}
			// The window is consumed whether or not the heal lands, so a
			// failed lookup cannot leave a stale damage factor behind.
			defer reg.Clear(t, characterId)

			cp := character.NewProcessor(l, ctx)
			c, err := cp.GetById()(characterId)
			if err != nil {
				l.WithError(err).Errorf("Chakra: failed to load caster [%d]; no heal applied.", characterId)
				return nil
			}

			luck := uint32(c.Luck())
			maxHp := c.MaxHp()
			stats, sErr := effective_stats.NewProcessor(l, ctx).GetByCharacterId(f.WorldId(), f.ChannelId(), characterId)
			if sErr != nil {
				l.WithError(sErr).Warnf("Chakra: effective stats unavailable for caster [%d]; falling back to base LUK and base max hp.", characterId)
			} else {
				luck = stats.Luck
				maxHp = chakrastate.EffectiveMaxHpOrBase(stats.MaxHp, c.MaxHp())
			}

			delta := healDelta(entry, luck, c.Hp(), maxHp)
			if delta == 0 {
				l.Debugf("Chakra: caster [%d] completed at level [%d] with no HP headroom (hp [%d] of [%d]); no ChangeHP emitted.", characterId, entry.SkillLevel, c.Hp(), maxHp)
				return nil
			}

			if hpErr := cp.ChangeHP(f, characterId, delta); hpErr != nil {
				l.WithError(hpErr).Errorf("Chakra: ChangeHP failed for caster [%d].", characterId)
				return nil
			}

			l.Debugf("Chakra: caster [%d] level [%d] restored [%d] hp (luck [%d], y [%d]%%, hp [%d] of [%d]).",
				characterId, entry.SkillLevel, delta, luck, entry.Y, c.Hp(), maxHp)
			return nil
		}
	}
}
```

Then add the blank import to `skill/handler/registrations/registrations.go`, in the existing alphabetical position (between `atlas-channel/skill/handler/heal` and... note the list is alphabetical by path: `chakra` sorts before `dispel`, so it goes first):

```go
	_ "atlas-channel/skill/handler/chakra"       // Chief Bandit Chakra — task-213
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./skill/... ./character/chakra/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/skill/handler/chakra/ \
        services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go
git commit -m "feat(task-213): apply the Chakra heal at recovery completion"
```

---

## Task 8: Interruption on movement, map change and disconnect; start the sweeper

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_move.go`
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/map_change.go`
- Modify: `services/atlas-channel/atlas.com/channel/socket/init.go`
- Test: `services/atlas-channel/atlas.com/channel/character/chakra/registry_test.go` (extend)

**Interfaces:**
- Consumes: `chakra.GetRegistry()`, `.Clear`, `.StartSweeper` (Task 2).
- Produces: nothing new.

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-channel/atlas.com/channel/character/chakra/registry_test.go`:

```go
// TestClearIsIdempotentAcrossInterruptSources pins PRD FR-5.1/FR-5.5: the
// move, map-change and session-destroy paths all call Clear, and only the
// first one to arrive reports an interrupt — so the log line is emitted once
// and a second caller is a harmless no-op.
func TestClearIsIdempotentAcrossInterruptSources(t *testing.T) {
	r := newRegistry()
	tn := testTenant(t)
	r.Start(tn, 42, 1, 99, 68, time.Now())

	first := r.Clear(tn, 42)  // movement
	second := r.Clear(tn, 42) // map change arriving after
	third := r.Clear(tn, 42)  // session destroy

	if !first {
		t.Fatal("the first interrupt did not report an open window")
	}
	if second || third {
		t.Fatal("a follow-up interrupt reported an open window")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./character/chakra/ -run TestClearIsIdempotent -v`
Expected: PASS immediately — `Clear` already has this contract from Task 2. This step pins the contract the three call sites depend on; if it fails, fix `Clear` before wiring the call sites.

- [ ] **Step 3: Write the implementation**

In `socket/handler/character_move.go`, add imports `"atlas-channel/character/chakra"` and `tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"`, and insert before the `movement.NewProcessor(...)` call:

```go
		// Movement cancels a pending Chakra heal (PRD FR-5.1). This is a
		// server-authority measure, not a simulation of client behaviour:
		// CUserLocal::IsImmovable returns true for the whole window, so an
		// authentic client physically cannot walk, jump, climb or rope and
		// never triggers this (design §3.7). It closes the crafted-client
		// hole where a player kites through the window collecting a free
		// heal and a damage factor. MP is not refunded because none was
		// spent — the generic cost block only runs on USE_SKILL.
		if chakra.GetRegistry().Clear(tenant.MustFromContext(ctx), s.CharacterId()) {
			l.Debugf("Chakra recovery window for character [%d] interrupted by movement; pending heal cancelled.", s.CharacterId())
		}
```

In `socket/handler/map_change.go`, add the same imports and insert immediately after the `p.Decode(...)` / debug line, before the `p.CashShopReturn()` branch:

```go
		// A map change ends any pending Chakra recovery (PRD FR-5.5).
		if chakra.GetRegistry().Clear(tenant.MustFromContext(ctx), s.CharacterId()) {
			l.Debugf("Chakra recovery window for character [%d] cleared by map change.", s.CharacterId())
		}
```

In `socket/init.go`, add the import `"atlas-channel/character/chakra"` and, inside the existing `socket.SetDestroyer` closure alongside the `shopscanner` / `statreset` clears, add:

```go
							// Channel change and disconnect both destroy the
							// session; without this the window map leaks one
							// entry per character ever seen by this pod
							// (PRD FR-5.5, FR-2.2).
							chakra.GetRegistry().ClearCharacter(t, s.CharacterId())
```

Use `chakra.GetRegistry().Clear(t, s.CharacterId())` — the registry exposes `Clear`, not `ClearCharacter`; if you prefer the `statreset`-matching name, add `ClearCharacter` as a thin alias in Task 2's file rather than renaming `Clear` (the damage path already depends on `Clear`'s bool return).

Also in `socket/init.go`, start the sweeper once, next to the other startup wiring in the same function that reads `ctx` and `l` (before the per-port `routine.Go` loop):

```go
	chakra.GetRegistry().StartSweeper(l, ctx)
```

- [ ] **Step 4: Run the tests and build to verify**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test -race ./...`
Expected: build clean, tests PASS.

Run: `cd ../../../.. && tools/goroutine-guard.sh`
Expected: exit 0 (the sweeper uses `routine.Go`).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_move.go \
        services/atlas-channel/atlas.com/channel/socket/handler/map_change.go \
        services/atlas-channel/atlas.com/channel/socket/init.go \
        services/atlas-channel/atlas.com/channel/character/chakra/registry_test.go
git commit -m "feat(task-213): interrupt Chakra on movement, map change and session destroy"
```

---

## Task 9: Route the skill-prepare packet on GMS 92

**Files:**
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_92_1.json`
- Modify: `docs/tasks/task-213-chakra-hp-restore/design.md` (append the v12 correction to §6.3)

**Interfaces:**
- Consumes: nothing.
- Produces: nothing consumed by later tasks.

**Why this task exists:** `CharacterSkillPrepareHandle` is bound in nine of eleven seed templates. Without it on GMS 92 the prepare packet is dropped and the recovery window never opens, so Chakra silently does nothing on that column even with every Go change in place.

**Correction to design §6.3 — GMS 12 is NOT edited.** Verified in this worktree:
- `template_gms_12_1.json` has 24 handlers total, ending at `0xB1 SummonDamageHandle`. It has **no** `CharacterUseSkillHandle`, so the second half of Chakra's two-packet flow has nowhere to land regardless.
- Its writer list contains no `CharacterSkillPrepareForeign` (design §6.3 says it does — that is true of GMS 92 only).
- There is **no** `docs/packets/registry/gms_v12.yaml` and **no** `docs/packets/ida-exports/gms_v12.json`. GMS 12 has no IDB at all (design §1). There is therefore no authority for a `SKILL_EFFECT` opcode on that column, and copying a neighbour's would be a fabricated wire value.

GMS 12 is recorded as out of reach for this feature per PRD FR-9.4. Chakra is inert on that column: the prepare packet is unrouted and `USE_SKILL` is unhandled. No Go change is needed to make it inert — the `USE_SKILL` gate rejects on a missing window, so nothing half-fires.

- [ ] **Step 1: Confirm the opcode from the registry, not from a neighbouring template**

Run:

```bash
grep -n -B4 -A2 "CUserLocal::DoActiveSkill_Prepare" docs/packets/registry/gms_v92.yaml
```

Expected: `op: SKILL_EFFECT`, `direction: serverbound`, `opcode: 104`. Decimal 104 = `0x68`.

Cross-check the neighbours agree with the already-bound entries in the template (registry `105 MESO_DROP`, `106 GIVE_FAME`; the template binds `FameChangeHandle` at `0x6A` = 106). If the registry and this expectation disagree, STOP and report — do not bind a guessed opcode.

- [ ] **Step 2: Confirm the binding is currently absent**

Run:

```bash
grep -c "CharacterSkillPrepareHandle" services/atlas-configurations/seed-data/templates/template_gms_92_1.json
```

Expected: `0`.

- [ ] **Step 3: Add the handler entry at its sorted position**

`tools/template-opcode-order-guard.sh` requires strictly ascending `opCode` within `handlers` — the new entry goes at its sorted position (`0x68`, between `0x66 CharacterUseSkillHandle` and `0x6A FameChangeHandle`), never appended next to a semantically related entry. Match the shape of the nine templates that already bind it (e.g. `template_gms_87_1.json:686-694`):

```json
      {
        "opCode": "0x68",
        "validator": "LoggedInValidator",
        "handler": "CharacterSkillPrepareHandle",
        "fname": "CUserLocal::DoActiveSkill_Prepare",
        "services": [
          "channel"
        ]
      },
```

A handler entry with a missing or empty `validator` is silently dropped at load — `LoggedInValidator` is required, matching every other template's binding.

- [ ] **Step 4: Run the three template guards**

Run from the worktree root:

```bash
tools/template-opcode-order-guard.sh
tools/template-duplicate-binding-guard.sh
tools/template-movement-types-guard.sh
```

Expected: all three exit 0.

- [ ] **Step 5: Record the GMS 12 exclusion in the design doc and commit**

Append to `design.md` §6.3 a short "Plan-phase correction" paragraph carrying the four verified facts above (24-handler stub, no `CharacterUseSkillHandle`, no `CharacterSkillPrepareForeign` writer, no packet registry or IDA export for GMS 12) and the conclusion that GMS 12 is out of reach per FR-9.4.

```bash
git add services/atlas-configurations/seed-data/templates/template_gms_92_1.json \
        docs/tasks/task-213-chakra-hp-restore/design.md
git commit -m "fix(task-213): route CharacterSkillPrepareHandle on gms_92; record gms_12 as out of reach"
```

---

## Task 10: Pin the GMS 95 `common` expansion and the keydown verdict

**Files:**
- Modify: `services/atlas-data/atlas.com/data/skill/common_test.go`
- Test: `libs/atlas-constants/skill/model_test.go` (re-run only — not edited)

**Interfaces:**
- Consumes: `synthesizeCommonNodes` (`services/atlas-data/atlas.com/data/skill/common.go:232`).
- Produces: nothing.

**Why this task exists:** GMS 95 is the only column whose Chakra levels come from a `common` formula node rather than explicit `level` nodes (design §4.2). A silent expansion failure there yields `x = 0`, which would zero every hit the caster takes during the window — a bug that reads as "Chakra makes me invincible" rather than as a data error.

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-data/atlas.com/data/skill/common_test.go`:

```go
// TestChakraCommonExpansion pins the GMS 95 Chakra shape (task-213 design
// §4.2): 4211001 ships a `common` node with maxLevel 10 and the linear rules
// mpCon = 40+10L, x = 100-4L, y = 100+20L. A regression in the expansion
// engine that dropped `x` would leave the damage factor at 0, which the
// damage path treats as "no window" rather than "zero damage" — silent, and
// exactly the failure this pin exists to catch.
func TestChakraCommonExpansion(t *testing.T) {
	common := xml.Node{
		Name: "common",
		IntegerNodes: []xml.IntegerNode{
			{Name: "maxLevel", Value: "10"},
		},
		StringNodes: []xml.StringNode{
			{Name: "mpCon", Value: "40+10*x"},
			{Name: "x", Value: "100-4*x"},
			{Name: "y", Value: "100+20*x"},
			{Name: "time", Value: "1"},
		},
	}
	l, _, ctx := commonTestContext(t)
	tn := tenant.MustFromContext(ctx)
	nodes, maxLevel, failures := synthesizeCommonNodes(l, tn, 421, 4211001, &common)
	if maxLevel != 10 {
		t.Fatalf("maxLevel = %d, want 10", maxLevel)
	}
	if failures != 0 {
		t.Fatalf("failures = %d, want 0", failures)
	}
	if len(nodes) != 10 {
		t.Fatalf("len(nodes) = %d, want 10", len(nodes))
	}

	want := map[string]map[string]string{
		"1":  {"mpCon": "50", "x": "96", "y": "120"},
		"10": {"mpCon": "140", "x": "60", "y": "300"},
	}
	for _, n := range nodes {
		exp, ok := want[n.Name]
		if !ok {
			continue
		}
		got := map[string]string{}
		for _, in := range n.IntegerNodes {
			got[in.Name] = in.Value
		}
		for k, v := range exp {
			if got[k] != v {
				t.Fatalf("level %s: %s = %q, want %q", n.Name, k, got[k], v)
			}
		}
	}
}
```

If `commonTestContext` seeds a tenant whose version drives the expansion, keep it as-is — the test is pinning the expansion engine's handling of Chakra's node shape, not re-reading WZ.

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd services/atlas-data/atlas.com/data && go test ./skill/ -run TestChakraCommonExpansion -v`
Expected: FAIL initially only if the formula strings do not match the engine's expression grammar. If it passes on the first run, that is a valid outcome for a pinning test — confirm by temporarily changing an expected value to a wrong one, watching it fail, then restoring it.

- [ ] **Step 3: Re-run the keydown pin (no edit)**

Design §3.5 reproduced task-161's finding independently: `is_keydown_skill` excludes 4211001 on v83 (`0x4FB08F`) and v95 (`0x509EA0`). PRD FR-10.3 therefore applies — `libs/atlas-constants/skill/model.go` and `model_test.go:33-37` are left **untouched**.

Run: `cd libs/atlas-constants && go test ./skill/ -run Keydown -v` (adjust the `-run` pattern to the actual test name in `model_test.go`).
Expected: PASS. Confirm `ChiefBanditChakraId` is still in the `notKeydown` list and that no file in this branch's diff adds it to `IsKeyDownSkill`.

Run: `git diff main --stat -- libs/atlas-constants/`
Expected: empty.

- [ ] **Step 4: Verify the research-doc correction target**

PRD FR-10.3 also requires correcting `docs/research/missing-features/skills-and-buffs.md` P7 (line 103), which claims Chakra needs a prepare broadcast. Design §3.5 notes that file is **untracked** and does not exist in this worktree.

Run: `ls docs/research/missing-features/skills-and-buffs.md`

- If it exists in this worktree, edit P7 to state the verdict: Chakra is **not** a keydown skill on any version (`is_keydown_skill` excludes 4211001 on v83 `0x4FB08F` and v95 `0x509EA0`), the `SetDamaged` compare that mentions it is inert, and the prepare packet Atlas needs is the ordinary `CUserLocal::DoActiveSkill_Prepare` serverbound handler, not a keydown broadcast. Cite `docs/tasks/task-213-chakra-hp-restore/design.md` §3.5.
- If it does not exist, that is the expected state: the authoritative verdict already lives in `design.md` §3.5, which is the artifact the four-phase workflow consumes. Do not create the file.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-data/atlas.com/data/skill/common_test.go
git commit -m "test(task-213): pin the gms_95 Chakra common-node expansion"
```

---

## Task 11: Full verification sweep and pre-PR review

**Files:** none created or modified except fixes the gates demand.

- [ ] **Step 1: Per-module tests, vet and build**

Run from the worktree root:

```bash
(cd services/atlas-channel/atlas.com/channel && go test -race ./... && go vet ./... && go build ./...)
(cd services/atlas-data/atlas.com/data && go test -race ./... && go vet ./... && go build ./...)
(cd libs/atlas-constants && go test -race ./... && go vet ./...)
```

Expected: all clean. Quote the actual output; do not claim clean without it.

- [ ] **Step 2: Repo-root guards**

```bash
tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/skill-job-id-guard.sh
tools/template-opcode-order-guard.sh
tools/template-duplicate-binding-guard.sh
tools/template-movement-types-guard.sh
tools/lint.sh --check
```

Expected: every one exits 0. `tools/lint.sh --check` needs nvm loaded for its atlas-ui half; if it false-fails without it, load nvm and re-run rather than declaring it clean.

Fix-mode formatting before committing any fix: `tools/lint.sh` (no flags).

- [ ] **Step 3: Confirm no `go.mod` was touched**

```bash
git diff main --stat -- '**/go.mod' '**/go.sum'
```

Expected: empty. If empty, `docker buildx bake` is not required by CLAUDE.md item 4. If any `go.mod` did change, run `docker buildx bake atlas-channel` (and `atlas-data` if it changed) from the worktree root and do not skip it.

- [ ] **Step 4: Confirm the scope boundaries held**

```bash
git diff main --stat -- libs/atlas-packet libs/atlas-constants
git diff main --stat -- services/atlas-configurations/seed-data/templates/
```

Expected: the first is empty (design §3.6, §3.5). The second shows `template_gms_92_1.json` only.

- [ ] **Step 5: Code review, then commit any fixes**

Invoke `superpowers:requesting-code-review` from inside this worktree. It dispatches `plan-adherence-reviewer` and `backend-guidelines-reviewer` (no TS changed, so no frontend reviewer); findings land in `docs/tasks/task-213-chakra-hp-restore/audit.md`. Pin the reviewer subagents to Sonnet or Haiku, not an expensive model.

Verify the tree is clean and still on the right worktree/branch after the subagent run:

```bash
git rev-parse --show-toplevel   # must end with /.worktrees/task-213-chakra-hp-restore
git branch --show-current       # must be task-213-chakra-hp-restore
git status --short
```

Address the findings, re-run Steps 1-2, then commit:

```bash
git add -A
git commit -m "chore(task-213): address code-review findings"
```

---

## Self-Review

**Spec coverage** — every PRD FR and design section mapped to a task:

| Requirement | Task |
|---|---|
| FR-1.1 / FR-1.2 / OQ-9 (gate, effective MaxHP, 50% boundary) | 1 (`CanActivate`, `EffectiveMaxHpOrBase`), 5 (gate wiring) |
| FR-1.3 (activation-only threshold) | 6 (explicitly no HP re-check) |
| FR-1.4 (rejection spends nothing) | 5 (no window opened → client sends no USE_SKILL), 6 (pre-cost reject) |
| FR-1.5 (caster knows the skill) | 5 (`skillLevel() == 0` → no window); generic check retained in `character_skill_use.go:70-77` |
| FR-2.1 / FR-2.2 / FR-2.4 / OQ-3 (state, TTL, hot-path read) | 2 |
| FR-2.5 / OQ-5 (no CTS invented) | verified by Task 11 Step 4 (`libs/atlas-packet` diff empty) |
| FR-3.1 / FR-3.2 / FR-3.3 / FR-3.4 / FR-3.5 (heal, clamp, ChangeHP, timing, never negative) | 1 (`Applied`), 7 |
| FR-4.1 / FR-4.2 / FR-4.3 / FR-4.4 / OQ-4 (factor, server-authoritative, first, integer) | 3 |
| FR-4.5 (the interrupting hit is factored) | 4 |
| FR-5.1 / OQ-8 (movement) | 8 |
| FR-5.2 (damage: factor → apply → cancel) | 4 |
| FR-5.3 (death) | 4 (the killing hit clears; `character_skill_use.go` already rejects a 0-HP cast) |
| FR-5.4 (no MP refund) | 6, 7 (structural — nothing is spent before completion) |
| FR-5.5 (map / channel / disconnect) | 8 |
| FR-6.1 / FR-6.2 / FR-6.3 / OQ-6 (WZ-driven, no hardcoded tables, x/y already parsed) | 5 (reads `effect.Model.X()/.Y()`), 10 (v95 pin) |
| FR-7.1-7.6 / OQ-1 (formula, separable terms, deterministic, documented negative result) | 1 |
| FR-8.1 (cast effect) | 7 (the generic path in `character_skill_use.go` announces; no duplicate) |
| FR-8.2 / FR-8.3 (cost ownership, once) | 7 |
| FR-8.4 (no XP) | 7 (by omission) |
| FR-9.1 (identity registration) | 7 |
| FR-9.2 / FR-9.3 (all versions, no version gate) | 3, 5 (data-driven), 9 (v92 routing) |
| FR-9.4 (versions that could not be reached) | 9 (GMS 12) |
| FR-10.1-10.4 / OQ-2 (keydown verdict) | 10 |
| OQ-7 (gate vs cost ordering) | 6 |
| Design §7 test table | 1, 2, 3, 4, 5, 7, 10 |
| Design §8 verification gates | 11 |

**Placeholder scan:** no `TBD`, no "add appropriate error handling", no "similar to Task N", no "write tests for the above". Every code step carries the actual code. The one deliberately conditional step is Task 10 Step 4 (an untracked file that may or may not exist), and both branches are spelled out.

**Type consistency:** `chakra.Entry{SkillLevel byte; X int16; Y int16; StartedAt time.Time}` is used identically in Tasks 2, 4, 5, 7. `Clear` returns `bool` in Tasks 2, 4, 8. `Applied`/`healDelta` return `int16`, matching `character.ChangeHP(f field.Model, characterId uint32, amount int16) error` (`character/processor.go:44`). `Recovery`/`Base` return `int32`. `chakraPct` is `int32` in Tasks 3 and 4, sourced from `int16` `Entry.X` via an explicit conversion. `effect.Model.X()` and `.Y()` return `int16` (`data/skill/effect/model.go:177,190`) — matching `Entry.X`/`Entry.Y` with no conversion. `EffectiveMaxHpOrBase(uint32, uint16) uint16` matches `effective_stats.RestModel.MaxHp uint32` and `character.Model.MaxHp() uint16`. `character.Model.Luck()` returns `uint16`, widened to `uint32` at the one call site in Task 7 to match `chakra.Base(luck uint32)` and `RestModel.Luck uint32`.
