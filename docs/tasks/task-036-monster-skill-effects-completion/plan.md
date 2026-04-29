# Monster Skill Effects Completion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Close the long tail of monster skill mechanics — reflect actually reflects, venom stacks correctly, mist (`AREA_POISON`) zones spawn/tick/expire as first-class atlas-maps objects with player HP DoT driven by atlas-buffs' existing `PoisonTick`, plus immunity mutual exclusion and dispel-guard invariants — across atlas-monsters, atlas-channel, atlas-buffs, atlas-maps, libs/atlas-packet, and libs/atlas-constants.

**Architecture:** Reflect math runs locally in atlas-channel via a per-tenant `StatusMirror` populated by the existing status consumers; the attack handler emits the existing `DAMAGE_REFLECTED` event and zeroes the entry's monster damage. Mist becomes a new `mist` domain in atlas-maps (model + registry + processor + tick task + command-and-event Kafka shapes), modelled on the existing `reactor/` package; atlas-monsters' `executeMist` produces `MIST_CREATE` commands; atlas-channel broadcasts `AffectedAreaCreated`/`Removed` packets. Venom uses the existing native multi-stack `[]StatusEffect` slice — only the eviction policy needs fixing (oldest-by-`ExpiresAt`). Immunity mutual exclusion runs inline in `executeStatBuff` (cancel-then-apply, two events). The dispel guard reads a new `SourceSkillClass` field on the `STATUS_CANCEL` command body, populated by atlas-channel.

**Tech Stack:** Go 1.22+ across services. Kafka via `github.com/Chronicle20/atlas/libs/atlas-kafka`. JSON:API REST via `api2go/jsonapi`. Multi-tenancy via `tenant.MustFromContext(ctx)`. UUIDs via `github.com/google/uuid`. Logging via `logrus`. Tracing via OpenTelemetry. Tests use `testing` + `require`/`assert` from `github.com/stretchr/testify`.

> **Workflow conventions:**
> - Run TDD: write the failing test, observe it fail, write the minimum implementation, observe it pass, commit.
> - Frequent small commits — one per task at minimum, more if a task has natural sub-units.
> - **Never commit directly to `main`.** This plan executes on a feature branch (per project convention).
> - Build & test the touched service after every task. Use `go build ./... && go test ./...` from the service root.
> - Read `context.md` once at the start; refer back for file:line evidence and locked decisions. Where PRD and design disagree, the **design wins** (specifically D3 supersedes PRD §4.4 / data-model §2-3 — no `VENOM_1`/`VENOM_2`/`VENOM_3` slot keys).

---

## Phase 0 — Leaves (parallelisable)

### Task 1: cjson empty-array audit on existing status-event bodies

**Why:** PRD FR-4.10 requires every slice / map field on status events to marshal as `[]`/`{}`, never `null`. New reflect fields are scalars (no slice gotcha) but the audit also covers existing bodies.

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/kafka.go:108-132`
- Test: `services/atlas-monsters/atlas.com/monsters/monster/kafka_test.go` (create if absent)

- [x] **Step 1.1: Inventory slice/map fields on status-event bodies**

Read `services/atlas-monsters/atlas.com/monsters/monster/kafka.go` lines 80-145. List every body type containing a slice or map field — at minimum: `statusEventDamagedBody.DamageEntries`, `statusEventKilledBody.DamageEntries`, `statusEffectAppliedBody.Statuses`, `statusEffectExpiredBody.Statuses`, `statusEffectCancelledBody.Statuses`. Record the inventory in a comment block at the top of the new test file.

- [x] **Step 1.2: Write failing round-trip tests for each body**

Create `services/atlas-monsters/atlas.com/monsters/monster/kafka_test.go`:

```go
package monster

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestStatusEventDamagedBody_EmptyDamageEntries_MarshalsAsArray(t *testing.T) {
	b := statusEventDamagedBody{
		DamageEntries: nil,
	}
	out, err := json.Marshal(b)
	require.NoError(t, err)
	require.Contains(t, string(out), `"damageEntries":[]`, "got: %s", out)
	require.NotContains(t, string(out), `"damageEntries":null`)
}

func TestStatusEventKilledBody_EmptyDamageEntries_MarshalsAsArray(t *testing.T) {
	b := statusEventKilledBody{
		DamageEntries: nil,
	}
	out, err := json.Marshal(b)
	require.NoError(t, err)
	require.Contains(t, string(out), `"damageEntries":[]`, "got: %s", out)
}

func TestStatusEffectAppliedBody_EmptyStatuses_MarshalsAsObject(t *testing.T) {
	b := statusEffectAppliedBody{
		EffectId: uuid.New(),
		Statuses: nil,
	}
	out, err := json.Marshal(b)
	require.NoError(t, err)
	require.Contains(t, string(out), `"statuses":{}`, "got: %s", out)
	require.NotContains(t, string(out), `"statuses":null`)
}

func TestStatusEffectExpiredBody_EmptyStatuses_MarshalsAsObject(t *testing.T) {
	b := statusEffectExpiredBody{Statuses: nil}
	out, err := json.Marshal(b)
	require.NoError(t, err)
	require.Contains(t, string(out), `"statuses":{}`)
}

func TestStatusEffectCancelledBody_EmptyStatuses_MarshalsAsObject(t *testing.T) {
	b := statusEffectCancelledBody{Statuses: nil}
	out, err := json.Marshal(b)
	require.NoError(t, err)
	require.Contains(t, string(out), `"statuses":{}`)
}

func TestStatusEventDamagedBody_RoundTripPreservesEmpty(t *testing.T) {
	in := statusEventDamagedBody{DamageEntries: nil}
	out, err := json.Marshal(in)
	require.NoError(t, err)
	require.True(t, strings.Contains(string(out), `"damageEntries":[]`))
	var back statusEventDamagedBody
	require.NoError(t, json.Unmarshal(out, &back))
}
```

- [x] **Step 1.3: Run tests — confirm they fail**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test ./monster/ -run 'TestStatusEvent.*MarshalsAs|TestStatusEffect.*MarshalsAs' -v
```

Expected: tests fail because slice/map fields with `nil` value marshal as `null`.

- [x] **Step 1.4: Implement custom `MarshalJSON` for each affected body**

For each body type that owns a slice field (`statusEventDamagedBody`, `statusEventKilledBody`), implement `MarshalJSON` that substitutes `nil` slices with empty slices before delegating to the standard marshaller via a type alias. Apply the established pattern from prior cjson fixes (commits `2c0ac23f2`, `afc3bd28a`):

```go
// At the bottom of kafka.go, after the body type declarations.

func (b statusEventDamagedBody) MarshalJSON() ([]byte, error) {
	type alias statusEventDamagedBody
	if b.DamageEntries == nil {
		b.DamageEntries = []damageEntry{}
	}
	return json.Marshal(alias(b))
}

func (b statusEventKilledBody) MarshalJSON() ([]byte, error) {
	type alias statusEventKilledBody
	if b.DamageEntries == nil {
		b.DamageEntries = []damageEntry{}
	}
	return json.Marshal(alias(b))
}

func (b statusEffectAppliedBody) MarshalJSON() ([]byte, error) {
	type alias statusEffectAppliedBody
	if b.Statuses == nil {
		b.Statuses = map[string]int32{}
	}
	return json.Marshal(alias(b))
}

func (b statusEffectExpiredBody) MarshalJSON() ([]byte, error) {
	type alias statusEffectExpiredBody
	if b.Statuses == nil {
		b.Statuses = map[string]int32{}
	}
	return json.Marshal(alias(b))
}

func (b statusEffectCancelledBody) MarshalJSON() ([]byte, error) {
	type alias statusEffectCancelledBody
	if b.Statuses == nil {
		b.Statuses = map[string]int32{}
	}
	return json.Marshal(alias(b))
}
```

(Add `"encoding/json"` to imports if absent.)

- [x] **Step 1.5: Run tests — confirm they pass**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test ./monster/ -run 'TestStatusEvent.*MarshalsAs|TestStatusEffect.*MarshalsAs|TestStatusEventDamagedBody_RoundTripPreservesEmpty' -v
```

Expected: all PASS.

- [x] **Step 1.6: Run full atlas-monsters test suite**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test ./...
```

Expected: PASS (no regressions on producer / picker / aggro tests).

- [x] **Step 1.7: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/kafka.go services/atlas-monsters/atlas.com/monsters/monster/kafka_test.go
git commit -m "task-036: cjson empty-array safety on monster status-event bodies"
```

---

### Task 2: AffectedAreaCreated / AffectedAreaRemoved packet writers

**Why:** atlas-channel needs these v83 wire packets to broadcast mist creation/removal (FR-4.6.6). Locations confirmed missing in `libs/atlas-packet/field/clientbound/` (design §1.2 resolution of PRD §9-1).

**Files:**
- Create: `libs/atlas-packet/field/clientbound/affected_area_created.go`
- Create: `libs/atlas-packet/field/clientbound/affected_area_removed.go`
- Test: `libs/atlas-packet/field/clientbound/affected_area_test.go`

- [x] **Step 2.1: Read existing template**

Read `libs/atlas-packet/reactor/clientbound/spawn.go` (lines 1-65) and `libs/atlas-packet/field/clientbound/clock.go` (lines 24-101). Confirm the writer pattern: struct + getters + `Operation()` returning a writer-name constant + `Encode(l, ctx)` returning a closure that writes bytes via the existing packet writer helpers.

- [x] **Step 2.2: Write failing tests**

Create `libs/atlas-packet/field/clientbound/affected_area_test.go`:

```go
package clientbound

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestAffectedAreaCreated_EncodeShape(t *testing.T) {
	mistId := uuid.MustParse("00000000-0000-0000-0000-00000000000a")
	w := NewAffectedAreaCreated(mistId, 0xCAFE, 100, 200, -50, -30, 50, 30, 10000, 12345)

	require.Equal(t, AffectedAreaCreatedWriter, w.Operation())

	enc := w.Encode(logrus.New(), context.Background())
	require.NotNil(t, enc)

	out := enc(map[string]interface{}{})
	require.NotEmpty(t, out, "encoded packet must be non-empty")
}

func TestAffectedAreaRemoved_EncodeShape(t *testing.T) {
	mistId := uuid.MustParse("00000000-0000-0000-0000-00000000000b")
	w := NewAffectedAreaRemoved(mistId, 0xCAFE)

	require.Equal(t, AffectedAreaRemovedWriter, w.Operation())

	enc := w.Encode(logrus.New(), context.Background())
	require.NotNil(t, enc)

	out := enc(map[string]interface{}{})
	require.NotEmpty(t, out)
}

func TestAffectedAreaCreated_Getters(t *testing.T) {
	mistId := uuid.New()
	w := NewAffectedAreaCreated(mistId, 7, 11, 22, -1, -2, 3, 4, 555, 9)
	require.Equal(t, mistId, w.MistId())
	require.Equal(t, uint32(7), w.OwnerId())
	require.Equal(t, int16(11), w.OriginX())
	require.Equal(t, int16(22), w.OriginY())
	require.Equal(t, int16(-1), w.LtX())
	require.Equal(t, int16(-2), w.LtY())
	require.Equal(t, int16(3), w.RbX())
	require.Equal(t, int16(4), w.RbY())
	require.Equal(t, int64(555), w.Duration())
	require.Equal(t, uint32(9), w.SkillLevel())
}
```

- [x] **Step 2.3: Run tests — confirm they fail**

```bash
cd libs/atlas-packet && go test ./field/clientbound/ -run 'TestAffectedArea' -v
```

Expected: FAIL with undefined identifiers.

- [x] **Step 2.4: Create `affected_area_created.go`**

Pattern: mirror `reactor/clientbound/spawn.go`. v83 affected-area opcode constant goes in the writer name; the actual byte opcode is consumed by the packet writer registration layer. Follow the in-tree convention (search `clock.go:54` for the `Operation()` style).

```go
package clientbound

import (
	"context"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const AffectedAreaCreatedWriter = "AffectedAreaCreated"

// AffectedAreaCreated v83 wire packet announcing a field-scoped affected area
// (mist) to clients in the field. Encoded fields mirror the legacy mist
// spawn packet: skill-id-as-mist-key (uint32), owner character/monster id,
// origin (x,y), bounding box (lt/rb), duration (ms).
type AffectedAreaCreated struct {
	mistId      uuid.UUID
	ownerId     uint32
	originX     int16
	originY     int16
	ltX         int16
	ltY         int16
	rbX         int16
	rbY         int16
	duration    int64
	skillLevel  uint32
}

func NewAffectedAreaCreated(mistId uuid.UUID, ownerId uint32, originX, originY, ltX, ltY, rbX, rbY int16, duration int64, skillLevel uint32) AffectedAreaCreated {
	return AffectedAreaCreated{
		mistId:     mistId,
		ownerId:    ownerId,
		originX:    originX,
		originY:    originY,
		ltX:        ltX,
		ltY:        ltY,
		rbX:        rbX,
		rbY:        rbY,
		duration:   duration,
		skillLevel: skillLevel,
	}
}

func (a AffectedAreaCreated) MistId() uuid.UUID { return a.mistId }
func (a AffectedAreaCreated) OwnerId() uint32   { return a.ownerId }
func (a AffectedAreaCreated) OriginX() int16    { return a.originX }
func (a AffectedAreaCreated) OriginY() int16    { return a.originY }
func (a AffectedAreaCreated) LtX() int16        { return a.ltX }
func (a AffectedAreaCreated) LtY() int16        { return a.ltY }
func (a AffectedAreaCreated) RbX() int16        { return a.rbX }
func (a AffectedAreaCreated) RbY() int16        { return a.rbY }
func (a AffectedAreaCreated) Duration() int64   { return a.duration }
func (a AffectedAreaCreated) SkillLevel() uint32 { return a.skillLevel }

func (a AffectedAreaCreated) Operation() string { return AffectedAreaCreatedWriter }

func (a AffectedAreaCreated) Encode(l logrus.FieldLogger, ctx context.Context) func(opts map[string]interface{}) []byte {
	return func(opts map[string]interface{}) []byte {
		// Mirror the byte layout used by reactor/clientbound/spawn.go:Encode —
		// integer header for the mist's stable id (low 32 bits of UUID), then
		// owner id, origin (x,y), Lt/Rb bounds, duration, level. Specific
		// opcode value is wired by the packet writer registry; this func
		// produces only the body.
		w := newPacketWriter()
		w.WriteUint32(uint32(a.mistId.ID()))
		w.WriteUint32(a.ownerId)
		w.WriteInt16(a.originX)
		w.WriteInt16(a.originY)
		w.WriteInt16(a.ltX)
		w.WriteInt16(a.ltY)
		w.WriteInt16(a.rbX)
		w.WriteInt16(a.rbY)
		w.WriteInt32(int32(a.duration))
		w.WriteUint32(a.skillLevel)
		return w.Bytes()
	}
}
```

> The exact `newPacketWriter()` API is the helper used by sibling writers in this directory. **Before writing, confirm by reading `libs/atlas-packet/field/clientbound/clock.go:60-80`** and use the identical helper invocation. If the helper exposes typed methods like `WriteByte`, `WriteUint16`, etc. instead of `WriteInt16`, adjust accordingly. Do not invent new helpers.

- [x] **Step 2.5: Create `affected_area_removed.go`**

```go
package clientbound

import (
	"context"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const AffectedAreaRemovedWriter = "AffectedAreaRemoved"

type AffectedAreaRemoved struct {
	mistId  uuid.UUID
	ownerId uint32
}

func NewAffectedAreaRemoved(mistId uuid.UUID, ownerId uint32) AffectedAreaRemoved {
	return AffectedAreaRemoved{mistId: mistId, ownerId: ownerId}
}

func (a AffectedAreaRemoved) MistId() uuid.UUID { return a.mistId }
func (a AffectedAreaRemoved) OwnerId() uint32   { return a.ownerId }

func (a AffectedAreaRemoved) Operation() string { return AffectedAreaRemovedWriter }

func (a AffectedAreaRemoved) Encode(l logrus.FieldLogger, ctx context.Context) func(opts map[string]interface{}) []byte {
	return func(opts map[string]interface{}) []byte {
		w := newPacketWriter()
		w.WriteUint32(uint32(a.mistId.ID()))
		return w.Bytes()
	}
}
```

- [x] **Step 2.6: Run tests — confirm they pass**

```bash
cd libs/atlas-packet && go test ./field/clientbound/ -run 'TestAffectedArea' -v
```

Expected: PASS.

- [x] **Step 2.7: Run full libs/atlas-packet suite**

```bash
cd libs/atlas-packet && go build ./... && go test ./...
```

Expected: PASS (no regressions on existing field/reactor/character/etc. writers).

- [x] **Step 2.8: Commit**

```bash
git add libs/atlas-packet/field/clientbound/affected_area_created.go libs/atlas-packet/field/clientbound/affected_area_removed.go libs/atlas-packet/field/clientbound/affected_area_test.go
git commit -m "task-036: AffectedAreaCreated/Removed packet writers"
```

---

### Task 3: Reflect kind constants in libs/atlas-constants

**Why:** atlas-monsters, atlas-channel, and the test suite all need a single source of truth for the `"PHYSICAL"` / `"MAGICAL"` reflect kind strings (design §3.6).

**Files:**
- Modify: `libs/atlas-constants/monster/skill.go:6-12`
- Test: `libs/atlas-constants/monster/skill_test.go` (create or extend if exists)

- [x] **Step 3.1: Write failing test for the new constants**

Create or open `libs/atlas-constants/monster/skill_test.go` and add:

```go
package monster

import "testing"

func TestReflectKindConstants(t *testing.T) {
	if ReflectKindPhysical != "PHYSICAL" {
		t.Fatalf("ReflectKindPhysical = %q, want PHYSICAL", ReflectKindPhysical)
	}
	if ReflectKindMagical != "MAGICAL" {
		t.Fatalf("ReflectKindMagical = %q, want MAGICAL", ReflectKindMagical)
	}
}

func TestReflectKindForSkill(t *testing.T) {
	cases := []struct {
		skillId uint16
		want    string
	}{
		{SkillTypePhysicalCounter, ReflectKindPhysical},
		{SkillTypeMagicCounter, ReflectKindMagical},
		{SkillTypePhysicalMagicCounter, ReflectKindPhysical}, // physical+magic combined; physical wins for the gate
		{1, ""}, // non-reflect
	}
	for _, c := range cases {
		got := ReflectKindForSkill(c.skillId)
		if got != c.want {
			t.Fatalf("ReflectKindForSkill(%d) = %q, want %q", c.skillId, got, c.want)
		}
	}
}
```

> The combined-counter skill (`SkillTypePhysicalMagicCounter` = 145) returns `PHYSICAL` for the dispel-guard's purpose; the same monster will *also* have an active reflect of `MAGICAL` recorded as a separate `StatusEffect` because the existing code maps the skill to the `WEAPON_COUNTER` and `MAGIC_COUNTER` statuses — we expose both via mirror lookups in T10. The `ReflectKindForSkill` is used only by the picker / dispel guard to classify the *skill itself*, not the resulting statuses.

- [x] **Step 3.2: Run test — confirm failure**

```bash
cd libs/atlas-constants && go test ./monster/ -run 'TestReflectKind' -v
```

Expected: FAIL — `ReflectKindPhysical`, `ReflectKindMagical`, `ReflectKindForSkill` undefined.

- [x] **Step 3.3: Implement constants and helper**

Edit `libs/atlas-constants/monster/skill.go` — add to the constant block (next to the existing `SkillCategory*` constants near lines 6-12) and add `ReflectKindForSkill` next to `SkillCategory(skillType uint16)` at lines 187-218:

```go
// (in the const block alongside SkillCategoryReflect)
const (
    ReflectKindPhysical = "PHYSICAL"
    ReflectKindMagical  = "MAGICAL"
)

// ReflectKindForSkill returns the reflect kind associated with a mob skill id.
// Returns "" for non-reflect skills. Used by the picker's dispel-guard
// classification (atlas-monsters) and by atlas-channel when populating
// StatusCancel.SourceSkillClass.
func ReflectKindForSkill(skillType uint16) string {
    switch skillType {
    case SkillTypePhysicalCounter:
        return ReflectKindPhysical
    case SkillTypeMagicCounter:
        return ReflectKindMagical
    case SkillTypePhysicalMagicCounter:
        return ReflectKindPhysical
    default:
        return ""
    }
}
```

- [x] **Step 3.4: Run test — confirm passing**

```bash
cd libs/atlas-constants && go test ./monster/ -run 'TestReflectKind' -v
```

Expected: PASS.

- [x] **Step 3.5: Run full libs/atlas-constants suite**

```bash
cd libs/atlas-constants && go build ./... && go test ./...
```

Expected: PASS.

- [x] **Step 3.6: Commit**

```bash
git add libs/atlas-constants/monster/skill.go libs/atlas-constants/monster/skill_test.go
git commit -m "task-036: ReflectKind constants and ReflectKindForSkill helper"
```

---

### Task 4: Venom eviction-policy fix in atlas-monsters builder

**Why:** Current eviction at `builder.go:130-156` removes the *first* VENOM-bearing effect on overflow. Design D3 + PRD FR-4.4.2 require evicting the effect with the **earliest `ExpiresAt`** (oldest-first by expiry, not by insertion order).

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/builder.go:130-156`
- Test: `services/atlas-monsters/atlas.com/monsters/monster/builder_test.go` (create or extend)

- [x] **Step 4.1: Write failing test**

Create or open `services/atlas-monsters/atlas.com/monsters/monster/builder_test.go` and add:

```go
package monster

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAddStatusEffect_VenomOverflow_EvictsByEarliestExpiresAt(t *testing.T) {
	now := time.Now()

	mk := func(label string, expiresIn time.Duration) StatusEffect {
		eff := NewStatusEffect(
			SourceTypePlayerSkill,
			1, // sourceCharacterId placeholder
			0, // skillId placeholder
			0,
			map[string]int32{"VENOM": 100, "_label": int32(label[0])}, // _label rides for traceability in test
			expiresIn,
			0,
		)
		// Override expiresAt directly so this test does not race with wall clock.
		// NewStatusEffect uses time.Now()+duration; align by using durations.
		_ = now
		return eff
	}

	b := NewMonsterBuilder( /* whatever fields the existing constructor needs */ )

	first := mk("a", 30*time.Second)   // earliest ExpiresAt — should be the eviction target
	second := mk("b", 60*time.Second)
	third := mk("c", 90*time.Second)

	b.AddStatusEffect(first).AddStatusEffect(second).AddStatusEffect(third)

	// Sanity: 3 venom effects present.
	require.Equal(t, 3, countVenom(b.statusEffects))

	// Apply a 4th: should evict `first` (earliest expiry), not `first-by-insertion`.
	fourth := mk("d", 120*time.Second)
	b.AddStatusEffect(fourth)

	require.Equal(t, 3, countVenom(b.statusEffects), "VENOM cap = 3")
	// `first` (earliest expiry) must be gone; `second`, `third`, `fourth` retained.
	require.False(t, hasEffectWithExpiry(b.statusEffects, first.ExpiresAt()), "earliest-expiry effect must be evicted")
	require.True(t, hasEffectWithExpiry(b.statusEffects, second.ExpiresAt()))
	require.True(t, hasEffectWithExpiry(b.statusEffects, third.ExpiresAt()))
	require.True(t, hasEffectWithExpiry(b.statusEffects, fourth.ExpiresAt()))
}

func countVenom(effs []StatusEffect) int {
	c := 0
	for _, e := range effs {
		if e.HasStatus("VENOM") {
			c++
		}
	}
	return c
}

func hasEffectWithExpiry(effs []StatusEffect, at time.Time) bool {
	for _, e := range effs {
		if e.ExpiresAt().Equal(at) {
			return true
		}
	}
	return false
}
```

> If `NewMonsterBuilder` requires arguments, check the existing builder API in `services/atlas-monsters/atlas.com/monsters/monster/builder.go` (top of file) and pass the minimum needed to construct an empty builder. If construction is awkward, use whichever helper existing tests in the same package use (look in `model_test.go`).

- [x] **Step 4.2: Run test — confirm failure**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test ./monster/ -run 'TestAddStatusEffect_VenomOverflow' -v
```

Expected: FAIL — current eviction removes the first inserted effect (`first`) only because it happens to be inserted first; if the test reorders by inserting the longest-expiry first, it would fail loudly. The test deliberately matches insertion-order to expiry-order so the existing FIFO behaviour passes by accident; **alter the test to insert in a different order than expiry** so the FIFO behaviour fails:

```go
// Insert in a confusing order:
b.AddStatusEffect(second). // 60s expiry, inserted 1st
  AddStatusEffect(third).  // 90s expiry, inserted 2nd
  AddStatusEffect(first).  // 30s expiry — earliest — inserted 3rd
  AddStatusEffect(fourth)  // 4th apply triggers eviction
// Under FIFO (current), `second` (inserted 1st) is evicted — wrong.
// Under earliest-expiry (correct), `first` (30s) is evicted.
require.False(t, hasEffectWithExpiry(b.statusEffects, first.ExpiresAt()), "FIFO bug: should evict earliest-expiry, not first-inserted")
require.True(t, hasEffectWithExpiry(b.statusEffects, second.ExpiresAt()))
```

Adjust the test accordingly before running. Re-run:

```bash
cd services/atlas-monsters/atlas.com/monsters && go test ./monster/ -run 'TestAddStatusEffect_VenomOverflow' -v
```

Expected: FAIL on the FIFO-bug assertion.

- [x] **Step 4.3: Implement eviction by earliest `ExpiresAt`**

Edit `services/atlas-monsters/atlas.com/monsters/monster/builder.go` lines 130-156. Replace the FIFO eviction with min-`ExpiresAt`:

```go
// AddStatusEffect adds a status effect, replacing any existing effect with overlapping status types.
// Exception: VENOM stacks up to 3 times. On overflow, the effect with the
// earliest ExpiresAt is evicted (oldest-first by expiry, not by insertion).
func (b *ModelBuilder) AddStatusEffect(effect StatusEffect) *ModelBuilder {
    for statusType := range effect.Statuses() {
        if statusType == "VENOM" {
            venomCount := 0
            evictIdx := -1
            for i, se := range b.statusEffects {
                if !se.HasStatus("VENOM") {
                    continue
                }
                venomCount++
                if evictIdx < 0 || se.ExpiresAt().Before(b.statusEffects[evictIdx].ExpiresAt()) {
                    evictIdx = i
                }
            }
            if venomCount >= 3 && evictIdx >= 0 {
                b.statusEffects = append(b.statusEffects[:evictIdx], b.statusEffects[evictIdx+1:]...)
            }
        } else {
            b.RemoveStatusEffectByType(statusType)
        }
    }
    b.statusEffects = append(b.statusEffects, effect)
    return b
}
```

- [x] **Step 4.4: Run test — confirm passing**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test ./monster/ -run 'TestAddStatusEffect_VenomOverflow' -v
```

Expected: PASS.

- [x] **Step 4.5: Add concurrency regression test (per risks §3)**

Append to `builder_test.go`:

```go
func TestAddStatusEffect_VenomConcurrentApplies_NeverExceedsThree(t *testing.T) {
	// Concurrency contract is enforced by the registry write lock around
	// AddStatusEffect (see processor.go ApplyStatusEffect callers). This test
	// asserts the builder logic never produces > 3 VENOM stacks even under
	// rapid sequential adds with non-monotonic expiry timestamps.
	b := NewMonsterBuilder( /* ... */ )

	for i := 0; i < 100; i++ {
		eff := NewStatusEffect(
			SourceTypePlayerSkill, 1, 0, 0,
			map[string]int32{"VENOM": int32(i)},
			time.Duration(i)*time.Second, // varying expiry — non-monotonic when randomised
			0,
		)
		b.AddStatusEffect(eff)
	}

	require.Equal(t, 3, countVenom(b.statusEffects))
}
```

```bash
cd services/atlas-monsters/atlas.com/monsters && go test ./monster/ -run 'TestAddStatusEffect_Venom' -v
```

Expected: PASS.

- [x] **Step 4.6: Run full atlas-monsters test suite**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test ./...
```

Expected: PASS.

- [x] **Step 4.7: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/builder.go services/atlas-monsters/atlas.com/monsters/monster/builder_test.go
git commit -m "task-036: evict oldest-by-ExpiresAt on VENOM stack overflow"
```

---

### Task 5: Verify and test existing `tasks.PoisonTick`

**Why:** Per design D9 — and confirmed by file inspection — `services/atlas-buffs/atlas.com/buffs/tasks/poison.go` already exists and is wired in `main.go:64` via `tasks.NewPoisonTick(l, 1000)`. The PRD's FR-4.7 task already runs. This step adds regression tests (the PRD's "TDD coverage for every behavior added" requirement) and confirms wiring.

**Files:**
- Verify: `services/atlas-buffs/atlas.com/buffs/tasks/poison.go:1-31`
- Verify: `services/atlas-buffs/atlas.com/buffs/main.go:64`
- Test: `services/atlas-buffs/atlas.com/buffs/tasks/poison_test.go` (create)

- [x] **Step 5.1: Confirm wiring in main.go**

```bash
grep -n "PoisonTick" services/atlas-buffs/atlas.com/buffs/main.go
```

Expected: line 64 — `go tasks.Register(tasks.NewPoisonTick(l, 1000))`. If absent or different, fix it (must mirror the `Expiration` registration on line 63).

- [x] **Step 5.2: Write a behavioural test**

Create `services/atlas-buffs/atlas.com/buffs/tasks/poison_test.go`:

```go
package tasks

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestPoisonTick_SleepTime_RespectsConfiguredInterval(t *testing.T) {
	pt := NewPoisonTick(logrus.New(), 750)
	require.Equal(t, 750*time.Millisecond, pt.SleepTime())
}

func TestPoisonTick_SleepTime_DefaultMillisecondMath(t *testing.T) {
	pt := NewPoisonTick(logrus.New(), 1000)
	require.Equal(t, time.Second, pt.SleepTime())
}

func TestPoisonTick_Run_DoesNotPanicWithNoTenants(t *testing.T) {
	pt := NewPoisonTick(logrus.New(), 1000)
	require.NotPanics(t, func() { pt.Run() })
}
```

> Behavioural assertion that the task actually emits CHANGE_HP commands belongs in an integration test against the character.ProcessPoisonTicks producer; we cover that producer in the existing `services/atlas-buffs/atlas.com/buffs/character/processor_test.go` if it exists. If no such test exists, add one:
>
> Create / extend `services/atlas-buffs/atlas.com/buffs/character/processor_test.go`:
> ```go
> func TestProcessPoisonTicks_NoEntries_ReturnsNoError(t *testing.T) {
>     l := logrus.New()
>     ctx := context.Background()
>     // Empty registry — no tenants resolved.
>     require.NoError(t, ProcessPoisonTicks(l, ctx))
> }
> ```

- [x] **Step 5.3: Run tests — confirm passing on first run (existing impl)**

```bash
cd services/atlas-buffs/atlas.com/buffs && go test ./tasks/ -run 'TestPoisonTick' -v
```

Expected: PASS (the impl already exists; we're adding regression coverage).

- [x] **Step 5.4: Run full atlas-buffs suite**

```bash
cd services/atlas-buffs/atlas.com/buffs && go build ./... && go test ./...
```

Expected: PASS.

- [x] **Step 5.5: Commit**

```bash
git add services/atlas-buffs/atlas.com/buffs/tasks/poison_test.go
# (and the processor_test.go addition if you created one)
git commit -m "task-036: regression tests for tasks.PoisonTick"
```

---

## Phase 1 — Mid-tier

### Task 6: Extend `monster.StatusEffect` with reflect fields

**Why:** Atlas-monsters needs to carry `reflectKind`/`reflectPercent`/bounding box / max-damage on each `StatusEffect` so the apply event downstream of `executeStatBuff` can ship structured reflect metadata to atlas-channel (FR-4.1.1, design §3.1).

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/status.go:14-108`
- Test: `services/atlas-monsters/atlas.com/monsters/monster/status_test.go` (create or extend)

- [x] **Step 6.1: Write failing test**

Create or extend `services/atlas-monsters/atlas.com/monsters/monster/status_test.go`:

```go
package monster

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStatusEffect_ReflectFields_DefaultZero(t *testing.T) {
	se := NewStatusEffect(SourceTypeMonsterSkill, 0, 100, 1,
		map[string]int32{"FREEZE": 1}, 5*time.Second, 0)
	require.Equal(t, "", se.ReflectKind())
	require.Equal(t, int32(0), se.ReflectPercent())
	require.Equal(t, int16(0), se.ReflectLtX())
	require.Equal(t, int16(0), se.ReflectLtY())
	require.Equal(t, int16(0), se.ReflectRbX())
	require.Equal(t, int16(0), se.ReflectRbY())
	require.Equal(t, int32(0), se.ReflectMaxDamage())
	require.False(t, se.IsReflect())
}

func TestNewReflectStatusEffect_PopulatesAllFields(t *testing.T) {
	se := NewReflectStatusEffect(
		SourceTypeMonsterSkill, 0, 143, 1,
		map[string]int32{"WEAPON_COUNTER": 30}, 60*time.Second,
		"PHYSICAL", 30, -50, -30, 50, 30, 32767,
	)
	require.Equal(t, "PHYSICAL", se.ReflectKind())
	require.Equal(t, int32(30), se.ReflectPercent())
	require.Equal(t, int16(-50), se.ReflectLtX())
	require.Equal(t, int16(-30), se.ReflectLtY())
	require.Equal(t, int16(50), se.ReflectRbX())
	require.Equal(t, int16(30), se.ReflectRbY())
	require.Equal(t, int32(32767), se.ReflectMaxDamage())
	require.True(t, se.IsReflect())
}

func TestStatusEffect_WithLastTick_PreservesReflectFields(t *testing.T) {
	se := NewReflectStatusEffect(
		SourceTypeMonsterSkill, 0, 143, 1,
		map[string]int32{"WEAPON_COUNTER": 30}, 60*time.Second,
		"PHYSICAL", 30, -10, -10, 10, 10, 32767,
	)
	updated := se.WithLastTick(time.Now())
	require.Equal(t, "PHYSICAL", updated.ReflectKind())
	require.Equal(t, int32(30), updated.ReflectPercent())
	require.Equal(t, int16(-10), updated.ReflectLtX())
}
```

- [x] **Step 6.2: Run test — confirm failure**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test ./monster/ -run 'TestStatusEffect_Reflect|TestNewReflectStatusEffect' -v
```

Expected: FAIL — undefined fields and constructor.

- [x] **Step 6.3: Implement struct extension and constructor**

Edit `services/atlas-monsters/atlas.com/monsters/monster/status.go` lines 14-26 to add fields, and append getters + the new `NewReflectStatusEffect` constructor:

```go
// (extend struct)
type StatusEffect struct {
    effectId          uuid.UUID
    sourceType        string
    sourceCharacterId uint32
    sourceSkillId     uint32
    sourceSkillLevel  uint32
    statuses          map[string]int32
    duration          time.Duration
    tickInterval      time.Duration
    lastTick          time.Time
    createdAt         time.Time
    expiresAt         time.Time

    // NEW (defaults zero for non-reflect statuses).
    reflectKind      string
    reflectPercent   int32
    reflectLtX       int16
    reflectLtY       int16
    reflectRbX       int16
    reflectRbY       int16
    reflectMaxDamage int32
}

// NewReflectStatusEffect constructs a StatusEffect with reflect metadata
// populated. For non-reflect statuses use NewStatusEffect (existing).
func NewReflectStatusEffect(
    sourceType string,
    sourceCharacterId uint32,
    sourceSkillId uint32,
    sourceSkillLevel uint32,
    statuses map[string]int32,
    duration time.Duration,
    reflectKind string,
    reflectPercent int32,
    ltX, ltY, rbX, rbY int16,
    reflectMaxDamage int32,
) StatusEffect {
    now := time.Now()
    return StatusEffect{
        effectId:          uuid.New(),
        sourceType:        sourceType,
        sourceCharacterId: sourceCharacterId,
        sourceSkillId:     sourceSkillId,
        sourceSkillLevel:  sourceSkillLevel,
        statuses:          statuses,
        duration:          duration,
        tickInterval:      0,
        lastTick:          now,
        createdAt:         now,
        expiresAt:         now.Add(duration),
        reflectKind:       reflectKind,
        reflectPercent:    reflectPercent,
        reflectLtX:        ltX,
        reflectLtY:        ltY,
        reflectRbX:        rbX,
        reflectRbY:        rbY,
        reflectMaxDamage:  reflectMaxDamage,
    }
}

// Append getters at the bottom of status.go alongside the existing ones:
func (s StatusEffect) ReflectKind() string      { return s.reflectKind }
func (s StatusEffect) ReflectPercent() int32    { return s.reflectPercent }
func (s StatusEffect) ReflectLtX() int16        { return s.reflectLtX }
func (s StatusEffect) ReflectLtY() int16        { return s.reflectLtY }
func (s StatusEffect) ReflectRbX() int16        { return s.reflectRbX }
func (s StatusEffect) ReflectRbY() int16        { return s.reflectRbY }
func (s StatusEffect) ReflectMaxDamage() int32  { return s.reflectMaxDamage }
func (s StatusEffect) IsReflect() bool          { return s.reflectKind != "" }
```

> `WithLastTick` already returns by-value (existing pattern at status.go:105-108) — the new fields automatically propagate. Confirm by reading the existing method.

- [x] **Step 6.4: Run test — confirm passing**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test ./monster/ -run 'TestStatusEffect_Reflect|TestNewReflectStatusEffect' -v
```

Expected: PASS.

- [x] **Step 6.5: Run full atlas-monsters test suite**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test ./...
```

Expected: PASS (no regressions; existing call sites use `NewStatusEffect` which is unchanged).

- [x] **Step 6.6: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/status.go services/atlas-monsters/atlas.com/monsters/monster/status_test.go
git commit -m "task-036: extend StatusEffect with reflect metadata fields"
```

---

### Task 7: Extend `statusEffectAppliedBody` with reflect fields + producer wiring

**Why:** API contract §1 — the apply event must carry `ReflectKind`/`ReflectPercent`/`ReflectLtX/Y/RbX/Y`/`ReflectMaxDamage` (no `omitempty`) so atlas-channel's `StatusMirror` can consume them.

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/kafka.go:108-116`
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/producer.go` (the existing `statusEffectAppliedProvider`)
- Test: extend `services/atlas-monsters/atlas.com/monsters/monster/kafka_test.go` from T1

- [x] **Step 7.1: Write failing test**

Append to `kafka_test.go`:

```go
func TestStatusEffectAppliedBody_NonReflect_SerializesEmptyReflectFields(t *testing.T) {
	b := statusEffectAppliedBody{
		EffectId: uuid.New(),
		Statuses: map[string]int32{"FREEZE": 1},
		// reflect fields default to zero values
	}
	out, err := json.Marshal(b)
	require.NoError(t, err)
	s := string(out)
	require.Contains(t, s, `"reflectKind":""`, "got: %s", s)
	require.Contains(t, s, `"reflectPercent":0`)
	require.Contains(t, s, `"reflectLtX":0`)
	require.Contains(t, s, `"reflectLtY":0`)
	require.Contains(t, s, `"reflectRbX":0`)
	require.Contains(t, s, `"reflectRbY":0`)
	require.Contains(t, s, `"reflectMaxDamage":0`)
	require.NotContains(t, s, `"reflectKind":null`)
}

func TestStatusEffectAppliedBody_Reflect_SerializesAllReflectFields(t *testing.T) {
	b := statusEffectAppliedBody{
		EffectId:         uuid.New(),
		Statuses:         map[string]int32{"WEAPON_COUNTER": 30},
		ReflectKind:      "PHYSICAL",
		ReflectPercent:   30,
		ReflectLtX:       -50,
		ReflectLtY:       -30,
		ReflectRbX:       50,
		ReflectRbY:       30,
		ReflectMaxDamage: 32767,
	}
	out, err := json.Marshal(b)
	require.NoError(t, err)
	s := string(out)
	require.Contains(t, s, `"reflectKind":"PHYSICAL"`)
	require.Contains(t, s, `"reflectPercent":30`)
	require.Contains(t, s, `"reflectMaxDamage":32767`)
}
```

- [x] **Step 7.2: Run — confirm failure**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test ./monster/ -run 'TestStatusEffectAppliedBody_(Non)?Reflect' -v
```

Expected: FAIL — fields don't exist on the body.

- [x] **Step 7.3: Extend the body**

Edit `kafka.go` lines 108-116:

```go
type statusEffectAppliedBody struct {
    EffectId          uuid.UUID        `json:"effectId"`
    SourceType        string           `json:"sourceType"`
    SourceCharacterId uint32           `json:"sourceCharacterId"`
    SourceSkillId     uint32           `json:"sourceSkillId"`
    SourceSkillLevel  uint32           `json:"sourceSkillLevel"`
    Statuses          map[string]int32 `json:"statuses"`
    Duration          int64            `json:"duration"`
    TickInterval      int64            `json:"tickInterval"`
    // NEW
    ReflectKind       string           `json:"reflectKind"`
    ReflectPercent    int32            `json:"reflectPercent"`
    ReflectLtX        int16            `json:"reflectLtX"`
    ReflectLtY        int16            `json:"reflectLtY"`
    ReflectRbX        int16            `json:"reflectRbX"`
    ReflectRbY        int16            `json:"reflectRbY"`
    ReflectMaxDamage  int32            `json:"reflectMaxDamage"`
}
```

- [x] **Step 7.4: Update `statusEffectAppliedProvider` to populate reflect fields from the StatusEffect**

Open `services/atlas-monsters/atlas.com/monsters/monster/producer.go`, find the function that constructs `statusEffectAppliedBody` (search for `statusEffectAppliedBody{`). Pass the reflect fields from the `StatusEffect` argument:

```go
// In the relevant provider — body construction:
body := statusEffectAppliedBody{
    EffectId:          effect.EffectId(),
    SourceType:        effect.SourceType(),
    SourceCharacterId: effect.SourceCharacterId(),
    SourceSkillId:     effect.SourceSkillId(),
    SourceSkillLevel:  effect.SourceSkillLevel(),
    Statuses:          effect.Statuses(),
    Duration:          int64(effect.Duration() / time.Millisecond),
    TickInterval:      int64(effect.TickInterval() / time.Millisecond),
    ReflectKind:       effect.ReflectKind(),
    ReflectPercent:    effect.ReflectPercent(),
    ReflectLtX:        effect.ReflectLtX(),
    ReflectLtY:        effect.ReflectLtY(),
    ReflectRbX:        effect.ReflectRbX(),
    ReflectRbY:        effect.ReflectRbY(),
    ReflectMaxDamage:  effect.ReflectMaxDamage(),
}
```

> Search for the actual location: `grep -n 'statusEffectAppliedBody{' services/atlas-monsters/atlas.com/monsters/monster/producer.go` and edit the matched provider in place.

- [x] **Step 7.5: Run tests — confirm passing**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test ./monster/ -run 'TestStatusEffectAppliedBody_(Non)?Reflect' -v
```

Expected: PASS.

- [x] **Step 7.6: Verify no regressions on existing producer tests**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test ./...
```

Expected: PASS.

- [x] **Step 7.7: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/kafka.go services/atlas-monsters/atlas.com/monsters/monster/producer.go services/atlas-monsters/atlas.com/monsters/monster/kafka_test.go
git commit -m "task-036: extend StatusEffectApplied event with structured reflect metadata"
```

---

### Task 8: Populate reflect metadata in `executeStatBuff`

**Why:** FR-4.1.1 — when applying `WEAPON_COUNTER` / `MAGIC_COUNTER`, atlas-monsters must populate the new reflect fields on the `StatusEffect` from the mob skill's `X()` and `LtX/LtY/RbX/RbY()` (design D4 + §3.1).

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/processor.go:647-689` (`executeStatBuff`)
- Test: `services/atlas-monsters/atlas.com/monsters/monster/processor_test.go` (extend)

- [x] **Step 8.1: Read existing executeStatBuff**

Re-read `services/atlas-monsters/atlas.com/monsters/monster/processor.go:647-689` to confirm the current shape (the `applyBuff` closure at lines 658-672 builds `NewStatusEffect`).

- [x] **Step 8.2: Verify reflect-skill mapping against actual mob skill data**

The PRD §6.2 + risks §4 require verifying `sd.X()/Y()/LtX/LtY/RbX/RbY` interpretation for skill type 143 (`SkillTypePhysicalCounter`). Per design D4 the mapping is locked:
- `ReflectPercent ← sd.X()`
- `ReflectLtX/Y/RbX/Y ← sd.LtX()/LtY()/RbX()/RbY()`
- `ReflectMaxDamage ← 32767` (constant)

If atlas-data is running in dev, sanity-check by calling: `curl -s 'http://localhost:<atlas-data-port>/data/mob-skills/143/levels/1' | jq '.attributes | {x, y, ltX, ltY, rbX, rbY}'`. Otherwise document the assumption inline in code and proceed.

- [x] **Step 8.3: Write failing test**

Add to `services/atlas-monsters/atlas.com/monsters/monster/processor_test.go`:

```go
package monster

import (
	"testing"

	monster2 "github.com/Chronicle20/atlas/libs/atlas-constants/monster"
	"github.com/stretchr/testify/require"
)

// fakeMobSkill implements the mobskill.Model interface (or, if mobskill.Model
// is a concrete struct, use whichever testing helper the package exposes;
// fall back to constructing via the existing builder).
type fakeMobSkill struct {
	x        int32
	ltX, ltY int16
	rbX, rbY int16
	duration uint32
}
// ... implement minimum methods needed by executeStatBuff

func TestExecuteStatBuff_ReflectStatus_PopulatesReflectMetadata(t *testing.T) {
	// Construct a monster with no active reflect.
	// Construct a fake mob skill for SkillTypePhysicalCounter (143) with
	//   X=30, LtX=-50, LtY=-30, RbX=50, RbY=30, Duration=60.
	// Call processor.executeStatBuff(monster, sd, 143, 1).
	// Assert: monster now has a status effect with HasStatus("WEAPON_COUNTER"),
	//   IsReflect()=true, ReflectKind="PHYSICAL", ReflectPercent=30,
	//   ReflectLtX=-50, ReflectRbX=50, ReflectMaxDamage=32767.
	t.Skip("Implementation in step 8.4 — replace skip after writing test body")
}
```

> The exact test boilerplate depends on existing helpers in `processor_test.go` and `picker_test.go` (the latter at lines 1-150+ uses `monsterInfoFetcher` and `mobSkillFetcher` closures). Match that style. Replace `t.Skip` with the concrete assertions once the helper layout is clear.

- [x] **Step 8.4: Implement reflect metadata in executeStatBuff**

Edit `processor.go` lines 647-689. After `statuses` is computed (line 655), branch on the skill category:

```go
func (p *ProcessorImpl) executeStatBuff(m Model, sd mobskill.Model, skillId byte, skillLevel byte) {
    statusName := monster2.SkillTypeToStatusName(uint16(skillId))
    if statusName == "" {
        p.l.Warnf("No status mapping for skill type [%d].", skillId)
        return
    }

    statuses := map[string]int32{string(statusName): sd.X()}
    duration := time.Duration(sd.Duration()) * time.Second

    category := monster2.SkillCategory(uint16(skillId))

    // NEW: Immunity mutual exclusion runs before the apply (Task 9).
    // (placeholder — Task 9 wires this in.)

    applyBuff := func(targetId uint32) {
        var effect StatusEffect
        if category == monster2.SkillCategoryReflect {
            kind := monster2.ReflectKindForSkill(uint16(skillId))
            effect = NewReflectStatusEffect(
                SourceTypeMonsterSkill,
                0,
                uint32(skillId),
                uint32(skillLevel),
                statuses,
                duration,
                kind,
                sd.X(),         // ReflectPercent
                int16(sd.LtX()),
                int16(sd.LtY()),
                int16(sd.RbX()),
                int16(sd.RbY()),
                32767,           // ReflectMaxDamage cap (constant per design D4)
            )
        } else {
            effect = NewStatusEffect(
                SourceTypeMonsterSkill,
                0,
                uint32(skillId),
                uint32(skillLevel),
                statuses,
                duration,
                0,
            )
        }
        if err := p.ApplyStatusEffect(targetId, effect); err != nil {
            p.l.WithError(err).Errorf("Unable to apply stat buff to monster [%d].", targetId)
        }
    }

    applyBuff(m.UniqueId())

    if monster2.IsAoeSkill(uint16(skillId)) && sd.HasBoundingBox() {
        // (existing AoE iteration body unchanged — lines 676-688)
    }
}
```

> Confirm `sd.LtX()` etc. return types (likely `int32` — narrow with `int16(...)` if so, per the mob skill model at `mobskill/model.go:71-83` which shows getter signatures).

- [x] **Step 8.5: Replace test skip with concrete assertions, run tests**

Fill in the test body using the existing test helpers (`monsterInfoFetcher`, `mobSkillFetcher`). Run:

```bash
cd services/atlas-monsters/atlas.com/monsters && go test ./monster/ -run 'TestExecuteStatBuff_ReflectStatus' -v
```

Expected: PASS.

- [x] **Step 8.6: Run picker tests + already-active gate test**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test ./monster/ -run 'TestPicker|TestExecuteStatBuff' -v
```

Expected: PASS — confirms FR-4.1.3 (re-applying reflect blocked by existing gate at `picker.go:185-191`).

- [x] **Step 8.7: Run full atlas-monsters suite**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test ./...
```

Expected: PASS.

- [x] **Step 8.8: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/processor.go services/atlas-monsters/atlas.com/monsters/monster/processor_test.go
git commit -m "task-036: populate reflect metadata in executeStatBuff"
```

---

### Task 9: Immunity mutual exclusion in `executeStatBuff`

**Why:** FR-4.8 + design D8 — applying `PHYSICAL_IMMUNE` while `MAGIC_IMMUNE` is active (or symmetric) must cancel the opposite immunity *before* the existing already-active gate at `processor.go:537-543`.

> The existing `WEAPON_ATTACK_IMMUNE` and `MAGIC_ATTACK_IMMUNE` constants in `libs/atlas-constants/monster/temporary_stat.go:6-44` are the canonical wire-side names. Confirm the exact spellings used by `SkillTypeToStatusName` for the immunity skill ids (search for `IMMUNE` in `libs/atlas-constants/monster/skill.go`). Use those names verbatim throughout.

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/processor.go:536-544` (the existing `category == SkillCategoryImmunity` gate area)
- Test: extend `processor_test.go`

- [x] **Step 9.1: Confirm the immunity status name strings**

```bash
grep -n 'WEAPON_ATTACK_IMMUNE\|MAGIC_ATTACK_IMMUNE\|PHYSICAL_IMMUNE\|MAGIC_IMMUNE' libs/atlas-constants/monster/
```

Lock the two strings used by `SkillTypeToStatusName` for the immunity skills. Most likely: `WEAPON_ATTACK_IMMUNE` and `MAGIC_ATTACK_IMMUNE` (per `temporary_stat.go:24-25`). Reference `SkillTypeToStatusName` at `libs/atlas-constants/monster/skill.go:61-86` for the actual mapping.

- [x] **Step 9.2: Write failing test**

Add to `processor_test.go`:

```go
func TestExecuteStatBuff_PhysicalImmune_CancelsActiveMagicImmune(t *testing.T) {
	// 1. Construct monster with active MAGIC_ATTACK_IMMUNE status effect.
	// 2. Apply WEAPON_ATTACK_IMMUNE via executeStatBuff.
	// 3. Assert: monster no longer has MAGIC_ATTACK_IMMUNE; has WEAPON_ATTACK_IMMUNE.
	// 4. Assert: producer recorded a STATUS_CANCELLED event for MAGIC_ATTACK_IMMUNE
	//    BEFORE a STATUS_APPLIED event for WEAPON_ATTACK_IMMUNE
	//    (event ordering matters for atlas-channel mirror correctness).
	t.Skip("Implementation in step 9.4 — replace skip after writing test body")
}

func TestExecuteStatBuff_MagicImmune_CancelsActivePhysicalImmune(t *testing.T) {
	// Symmetric.
	t.Skip("Implementation in step 9.4 — replace skip")
}

func TestExecuteStatBuff_PhysicalImmune_NoMagicImmune_DoesNotCancel(t *testing.T) {
	// Sanity: applying PHYSICAL_IMMUNE when no MAGIC_IMMUNE active does NOT
	// emit a STATUS_CANCELLED event (no spurious cancels).
	t.Skip("Implementation in step 9.4 — replace skip")
}
```

- [x] **Step 9.3: Run tests — confirm they fail (after fleshing out)**

After replacing `t.Skip` with real assertions in step 9.4, expect FAIL because no mutual-exclusion logic exists.

- [x] **Step 9.4: Implement mutual-exclusion in `executeStatBuff` AND in `UseSkill`**

Two call paths set immunity statuses: `executeStatBuff` (for AoE / direct apply) and the same path is reached via `UseSkill → executeStatBuff` (line 558). The existing already-active gate at `processor.go:537-543` rejects the *new* skill if the same status is already present. We must run the opposite-immunity cancel **before** that gate.

Edit `processor.go` lines 536-544 — wrap the existing gate with the cancel logic:

```go
// Stacking check for reflect/immunity - cannot apply if already active.
category := monster2.SkillCategory(uint16(skillId))
if category == monster2.SkillCategoryImmunity {
    statusName := monster2.SkillTypeToStatusName(uint16(skillId))
    // Mutual exclusion: PHYSICAL_IMMUNE displaces MAGIC_IMMUNE and vice versa.
    var oppositeName string
    switch string(statusName) {
    case "WEAPON_ATTACK_IMMUNE":
        oppositeName = "MAGIC_ATTACK_IMMUNE"
    case "MAGIC_ATTACK_IMMUNE":
        oppositeName = "WEAPON_ATTACK_IMMUNE"
    }
    if oppositeName != "" && m.HasStatusEffect(oppositeName) {
        // Cancel the opposite immunity through the existing internal cancel path.
        // CancelStatusEffect emits a STATUS_CANCELLED event partition-keyed by uniqueId,
        // arriving at atlas-channel before the upcoming STATUS_APPLIED.
        if err := p.CancelStatusEffect(uniqueId, []string{oppositeName}); err != nil {
            p.l.WithError(err).Warnf("Failed to cancel opposite immunity [%s] on monster [%d].", oppositeName, uniqueId)
        }
        // Re-fetch the monster post-cancel.
        m, err = GetMonsterRegistry().GetMonster(p.t, uniqueId)
        if err != nil {
            p.l.WithError(err).Errorf("Unable to re-fetch monster [%d] after immunity cancel.", uniqueId)
            return
        }
    }
}

if category == monster2.SkillCategoryImmunity || category == monster2.SkillCategoryReflect {
    statusName := monster2.SkillTypeToStatusName(uint16(skillId))
    if statusName != "" && m.HasStatusEffect(string(statusName)) {
        p.l.Debugf("Monster [%d] already has active [%s]. Skill [%d] rejected.", uniqueId, statusName, skillId)
        return
    }
}
```

> Confirm the exact `CancelStatusEffect` signature in `processor.go` (search `func.*CancelStatusEffect`). It must accept `[]string` of status names; if it accepts `[]string` of effect ids, switch to `CancelStatusByType` or wrap with the appropriate helper. **Read the existing impl before writing the call.**

- [x] **Step 9.5: Replace test skips with concrete assertions, run tests**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test ./monster/ -run 'TestExecuteStatBuff_(PhysicalImmune|MagicImmune)' -v
```

Expected: PASS — including event ordering.

- [x] **Step 9.6: Run full atlas-monsters suite**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test ./...
```

Expected: PASS.

- [x] **Step 9.7: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/processor.go services/atlas-monsters/atlas.com/monsters/monster/processor_test.go
git commit -m "task-036: immunity mutual exclusion in executeStatBuff"
```

---

### Task 10: `monster.StatusMirror` in atlas-channel

**Why:** Per design D1 + FR-4.2 — atlas-channel maintains an in-process per-tenant mirror of monster status effects so the attack handler can do reflect math without a Kafka round-trip per damage entry.

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/monster/status_mirror.go`
- Test: `services/atlas-channel/atlas.com/channel/monster/status_mirror_test.go`

- [x] **Step 10.1: Write failing test**

Create `services/atlas-channel/atlas.com/channel/monster/status_mirror_test.go`:

```go
package monster

import (
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/tenant"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func mkTenant() tenant.Model {
	t, _ := tenant.Create(uuid.MustParse("11111111-1111-1111-1111-111111111111"), "GMS", 83, 1)
	return t
}

func TestStatusMirror_OnApplied_StoresEntry(t *testing.T) {
	mirror := &StatusMirror{perTenant: map[string]map[uint32]map[string][]StatusEntry{}}
	tt := mkTenant()
	body := StatusEffectAppliedBody{
		EffectId:  uuid.New(),
		Statuses:  map[string]int32{"WEAPON_COUNTER": 30},
		Duration:  60000,
	}

	mirror.OnApplied(tt, 100, body)

	require.Equal(t, 1, len(mirror.perTenant))
}

func TestStatusMirror_GetReflect_AbsentForUnknownMonster(t *testing.T) {
	mirror := newTestStatusMirror()
	_, ok := mirror.GetReflect(mkTenant(), 999, "PHYSICAL")
	require.False(t, ok)
}

func TestStatusMirror_GetReflect_PresentAfterReflectApplied(t *testing.T) {
	mirror := newTestStatusMirror()
	tt := mkTenant()
	body := StatusEffectAppliedBody{
		EffectId:         uuid.New(),
		Statuses:         map[string]int32{"WEAPON_COUNTER": 30},
		Duration:         60000,
		ReflectKind:      "PHYSICAL",
		ReflectPercent:   30,
		ReflectLtX:       -50,
		ReflectLtY:       -30,
		ReflectRbX:       50,
		ReflectRbY:       30,
		ReflectMaxDamage: 32767,
	}

	mirror.OnApplied(tt, 100, body)
	info, ok := mirror.GetReflect(tt, 100, "PHYSICAL")
	require.True(t, ok)
	require.Equal(t, int32(30), info.Percent)
	require.Equal(t, int16(-50), info.LtX)
	require.Equal(t, int16(50), info.RbX)
	require.Equal(t, int32(32767), info.MaxDamage)
}

func TestStatusMirror_OnExpired_RemovesEntryByEffectId(t *testing.T) {
	mirror := newTestStatusMirror()
	tt := mkTenant()
	effId := uuid.New()
	body := StatusEffectAppliedBody{
		EffectId:    effId,
		Statuses:    map[string]int32{"WEAPON_COUNTER": 30},
		ReflectKind: "PHYSICAL",
	}
	mirror.OnApplied(tt, 100, body)

	mirror.OnExpired(tt, 100, effId, body.Statuses)
	_, ok := mirror.GetReflect(tt, 100, "PHYSICAL")
	require.False(t, ok)
}

func TestStatusMirror_OnMonsterGone_ClearsAllEntries(t *testing.T) {
	mirror := newTestStatusMirror()
	tt := mkTenant()
	mirror.OnApplied(tt, 100, StatusEffectAppliedBody{
		EffectId: uuid.New(),
		Statuses: map[string]int32{"WEAPON_COUNTER": 30},
		ReflectKind: "PHYSICAL",
	})
	mirror.OnApplied(tt, 100, StatusEffectAppliedBody{
		EffectId: uuid.New(),
		Statuses: map[string]int32{"FREEZE": 1},
	})

	mirror.OnMonsterGone(tt, 100)

	_, ok := mirror.GetReflect(tt, 100, "PHYSICAL")
	require.False(t, ok)
	require.Equal(t, 0, mirror.VenomCount(tt, 100))
}

func TestStatusMirror_VenomCount_TracksMultipleApplies(t *testing.T) {
	mirror := newTestStatusMirror()
	tt := mkTenant()

	require.Equal(t, 0, mirror.VenomCount(tt, 100))

	for i := 0; i < 3; i++ {
		mirror.OnApplied(tt, 100, StatusEffectAppliedBody{
			EffectId: uuid.New(),
			Statuses: map[string]int32{"VENOM": int32(50 + i*10)},
			Duration: int64(time.Duration(60-i*10) * time.Second / time.Millisecond),
		})
	}

	require.Equal(t, 3, mirror.VenomCount(tt, 100))
}

func TestStatusMirror_ConcurrentReadsAndWrites_Safe(t *testing.T) {
	mirror := newTestStatusMirror()
	tt := mkTenant()

	done := make(chan struct{})
	go func() {
		for i := 0; i < 1000; i++ {
			mirror.OnApplied(tt, 100, StatusEffectAppliedBody{
				EffectId:    uuid.New(),
				Statuses:    map[string]int32{"WEAPON_COUNTER": 30},
				ReflectKind: "PHYSICAL",
			})
		}
		close(done)
	}()
	for i := 0; i < 1000; i++ {
		_, _ = mirror.GetReflect(tt, 100, "PHYSICAL")
	}
	<-done
}

func newTestStatusMirror() *StatusMirror {
	return &StatusMirror{perTenant: map[string]map[uint32]map[string][]StatusEntry{}}
}
```

- [x] **Step 10.2: Run tests — confirm failure**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./monster/ -run 'TestStatusMirror' -v
```

Expected: FAIL — types undefined.

- [x] **Step 10.3: Implement the mirror**

Create `services/atlas-channel/atlas.com/channel/monster/status_mirror.go`:

```go
package monster

import (
	"sync"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/tenant"
	"github.com/google/uuid"
)

// StatusEffectAppliedBody mirrors the atlas-monsters event body. Defined here
// to avoid a cross-service import; field names and json tags must stay in sync
// with atlas-monsters/monster/kafka.go:statusEffectAppliedBody.
type StatusEffectAppliedBody struct {
	EffectId          uuid.UUID        `json:"effectId"`
	SourceType        string           `json:"sourceType"`
	SourceCharacterId uint32           `json:"sourceCharacterId"`
	SourceSkillId     uint32           `json:"sourceSkillId"`
	SourceSkillLevel  uint32           `json:"sourceSkillLevel"`
	Statuses          map[string]int32 `json:"statuses"`
	Duration          int64            `json:"duration"`
	TickInterval      int64            `json:"tickInterval"`
	ReflectKind       string           `json:"reflectKind"`
	ReflectPercent    int32            `json:"reflectPercent"`
	ReflectLtX        int16            `json:"reflectLtX"`
	ReflectLtY        int16            `json:"reflectLtY"`
	ReflectRbX        int16            `json:"reflectRbX"`
	ReflectRbY        int16            `json:"reflectRbY"`
	ReflectMaxDamage  int32            `json:"reflectMaxDamage"`
}

// ReflectInfo is the per-reflect-status snapshot returned by GetReflect.
type ReflectInfo struct {
	Kind      string
	Percent   int32
	LtX       int16
	LtY       int16
	RbX       int16
	RbY       int16
	MaxDamage int32
	ExpiresAt time.Time
}

// StatusEntry is a per-status-name occurrence on a monster.
type StatusEntry struct {
	EffectId  uuid.UUID
	Statuses  map[string]int32
	Reflect   *ReflectInfo // nil for non-reflect entries
	ExpiresAt time.Time
}

// StatusMirror is the in-memory per-tenant projection of atlas-monsters
// status events. NOT authoritative; reads are eventually consistent.
type StatusMirror struct {
	mu        sync.RWMutex
	perTenant map[string]map[uint32]map[string][]StatusEntry
}

var (
	statusMirrorOnce sync.Once
	statusMirror     *StatusMirror
)

func GetStatusMirror() *StatusMirror {
	statusMirrorOnce.Do(func() {
		statusMirror = &StatusMirror{
			perTenant: map[string]map[uint32]map[string][]StatusEntry{},
		}
	})
	return statusMirror
}

func tenantKey(t tenant.Model) string { return t.Id().String() }

func (m *StatusMirror) OnApplied(t tenant.Model, uniqueId uint32, body StatusEffectAppliedBody) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tk := tenantKey(t)
	if m.perTenant[tk] == nil {
		m.perTenant[tk] = map[uint32]map[string][]StatusEntry{}
	}
	if m.perTenant[tk][uniqueId] == nil {
		m.perTenant[tk][uniqueId] = map[string][]StatusEntry{}
	}
	expires := time.Now().Add(time.Duration(body.Duration) * time.Millisecond)
	var ref *ReflectInfo
	if body.ReflectKind != "" {
		ref = &ReflectInfo{
			Kind:      body.ReflectKind,
			Percent:   body.ReflectPercent,
			LtX:       body.ReflectLtX,
			LtY:       body.ReflectLtY,
			RbX:       body.ReflectRbX,
			RbY:       body.ReflectRbY,
			MaxDamage: body.ReflectMaxDamage,
			ExpiresAt: expires,
		}
	}
	entry := StatusEntry{
		EffectId:  body.EffectId,
		Statuses:  body.Statuses,
		Reflect:   ref,
		ExpiresAt: expires,
	}
	for stat := range body.Statuses {
		m.perTenant[tk][uniqueId][stat] = append(m.perTenant[tk][uniqueId][stat], entry)
	}
}

func (m *StatusMirror) OnExpired(t tenant.Model, uniqueId uint32, effectId uuid.UUID, statuses map[string]int32) {
	m.removeByEffectId(t, uniqueId, effectId, statuses)
}

func (m *StatusMirror) OnCancelled(t tenant.Model, uniqueId uint32, effectId uuid.UUID, statuses map[string]int32) {
	m.removeByEffectId(t, uniqueId, effectId, statuses)
}

func (m *StatusMirror) removeByEffectId(t tenant.Model, uniqueId uint32, effectId uuid.UUID, statuses map[string]int32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tk := tenantKey(t)
	mons := m.perTenant[tk]
	if mons == nil {
		return
	}
	statMap := mons[uniqueId]
	if statMap == nil {
		return
	}
	for stat := range statuses {
		entries := statMap[stat]
		filtered := entries[:0]
		for _, e := range entries {
			if e.EffectId != effectId {
				filtered = append(filtered, e)
			}
		}
		if len(filtered) == 0 {
			delete(statMap, stat)
		} else {
			statMap[stat] = filtered
		}
	}
}

func (m *StatusMirror) OnMonsterGone(t tenant.Model, uniqueId uint32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tk := tenantKey(t)
	if mons := m.perTenant[tk]; mons != nil {
		delete(mons, uniqueId)
	}
}

func (m *StatusMirror) GetReflect(t tenant.Model, uniqueId uint32, kind string) (ReflectInfo, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tk := tenantKey(t)
	mons := m.perTenant[tk]
	if mons == nil {
		return ReflectInfo{}, false
	}
	statMap := mons[uniqueId]
	if statMap == nil {
		return ReflectInfo{}, false
	}
	now := time.Now()
	for _, entries := range statMap {
		for _, e := range entries {
			if e.Reflect != nil && e.Reflect.Kind == kind && now.Before(e.ExpiresAt) {
				return *e.Reflect, true
			}
		}
	}
	return ReflectInfo{}, false
}

func (m *StatusMirror) VenomCount(t tenant.Model, uniqueId uint32) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tk := tenantKey(t)
	mons := m.perTenant[tk]
	if mons == nil {
		return 0
	}
	return len(mons[uniqueId]["VENOM"])
}
```

> The exact `tenant.Create(...)` API used in tests must match the in-tree convention. Search `tenant.Create\|tenant.MustFromContext` for an example test usage and adjust.

- [x] **Step 10.4: Run tests — confirm passing**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./monster/ -run 'TestStatusMirror' -v -race
```

Expected: PASS, including the concurrent-access test under `-race`.

- [x] **Step 10.5: Run full atlas-channel suite**

```bash
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...
```

Expected: PASS.

- [x] **Step 10.6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/monster/status_mirror.go services/atlas-channel/atlas.com/channel/monster/status_mirror_test.go
git commit -m "task-036: monster.StatusMirror — in-process per-tenant projection of status events"
```

---

### Task 11: Wire `StatusMirror` into existing status consumers

**Why:** FR-4.2.1/4.2.4 — the existing handlers must populate / prune the mirror.

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go:299-403` (handlers) and `:119-228` (destroy/killed)

- [x] **Step 11.1: Inspect existing handler signatures and event body types**

Read `services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go` lines 119-228 and 299-403. Confirm the event-body types referenced — they may be defined in `services/atlas-channel/atlas.com/channel/kafka/message/monster/` (search for `statusEffectAppliedBody`). If atlas-channel mirrors atlas-monsters' kafka body shape, ensure the new reflect fields are present in the channel-side struct as well (T7 only modified atlas-monsters).

If the channel-side body lacks the new fields, extend it now:

```bash
grep -rn 'StatusEffectAppliedBody\|statusEffectAppliedBody' services/atlas-channel/atlas.com/channel/kafka/
```

Edit the channel-side body (likely at `services/atlas-channel/atlas.com/channel/kafka/message/monster/<file>.go`) to add the same `ReflectKind`/etc. fields with matching json tags.

- [x] **Step 11.2: Wire mirror calls into handlers**

In `consumer.go`:

```go
// At the END of handleStatusEffectApplied (after the existing MonsterStatSetWriter call):
monster.GetStatusMirror().OnApplied(t, e.UniqueId, monster.StatusEffectAppliedBody{
    EffectId:         e.Body.EffectId,
    SourceType:       e.Body.SourceType,
    SourceCharacterId: e.Body.SourceCharacterId,
    SourceSkillId:    e.Body.SourceSkillId,
    SourceSkillLevel: e.Body.SourceSkillLevel,
    Statuses:         e.Body.Statuses,
    Duration:         e.Body.Duration,
    TickInterval:     e.Body.TickInterval,
    ReflectKind:      e.Body.ReflectKind,
    ReflectPercent:   e.Body.ReflectPercent,
    ReflectLtX:       e.Body.ReflectLtX,
    ReflectLtY:       e.Body.ReflectLtY,
    ReflectRbX:       e.Body.ReflectRbX,
    ReflectRbY:       e.Body.ReflectRbY,
    ReflectMaxDamage: e.Body.ReflectMaxDamage,
})

// At the END of handleStatusEffectExpired:
monster.GetStatusMirror().OnExpired(t, e.UniqueId, e.Body.EffectId, e.Body.Statuses)

// At the END of handleStatusEffectCancelled:
monster.GetStatusMirror().OnCancelled(t, e.UniqueId, e.Body.EffectId, e.Body.Statuses)

// At the END of handleStatusEventDestroyed AND handleStatusEventKilled:
monster.GetStatusMirror().OnMonsterGone(t, e.UniqueId)
```

> Locate the actual `t` (tenant.Model) variable name in each handler — likely extracted from the event header via `tenant.MustFromContext` or `e.Tenant`. Use whatever the existing handlers use. Match the surrounding style.

- [x] **Step 11.3: Build to confirm syntax**

```bash
cd services/atlas-channel/atlas.com/channel && go build ./...
```

Expected: PASS.

- [x] **Step 11.4: Add a regression test asserting handlers populate the mirror**

Create or extend `services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer_test.go`:

```go
func TestHandleStatusEffectApplied_PopulatesMirror(t *testing.T) {
	// 1. Reset the mirror for test isolation (call a test-only helper, or
	//    construct a fresh mirror via newTestStatusMirror in a sub-package).
	// 2. Call the handler with a synthesized event carrying ReflectKind=PHYSICAL.
	// 3. Assert: mirror.GetReflect(tenant, uniqueId, "PHYSICAL") returns ok=true.
	t.Skip("Implementation in step 11.4 — replace skip after writing test body")
}

// Symmetric tests for Expired, Cancelled, Destroyed, Killed.
```

> Mirror tests in this package may need a way to reset the singleton between tests. Add a `resetStatusMirrorForTest()` exported only via a `_test.go` file in the `monster` package. Confirm existing reset patterns by looking at how other singletons in atlas-channel are tested.

- [x] **Step 11.5: Run all atlas-channel tests**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./...
```

Expected: PASS.

- [x] **Step 11.6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer_test.go services/atlas-channel/atlas.com/channel/kafka/message/monster/  # whichever file was extended
git commit -m "task-036: wire StatusMirror into monster status consumers"
```

---

### Task 12: VENOM wire-collapse via `VenomCount`

**Why:** FR-4.4.5 + design D3 — multiple `VENOM` apply events in atlas-channel must result in **one** `MonsterStatSet(VENOM)` (transition 0→1) and exactly one `MonsterStatReset(VENOM)` on the *last* expiry/cancel (transition to 0).

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go:299-321` (apply) and `:323-369` (expire/cancel)

- [x] **Step 12.1: Write failing test**

Add to `consumer_test.go`:

```go
func TestHandleStatusEffectApplied_VenomFirstApply_BroadcastsMonsterStatSet(t *testing.T) {
	// Synthesize first VENOM apply event for monster M.
	// Assert: MonsterStatSet(VENOM) writer was announced exactly once.
	t.Skip("Implementation in step 12.2")
}

func TestHandleStatusEffectApplied_VenomSecondAndThirdApply_DoesNotBroadcast(t *testing.T) {
	// Synthesize three sequential VENOM applies for the same monster.
	// Assert: only the FIRST triggers MonsterStatSet — the other two are suppressed.
	t.Skip("Implementation in step 12.2")
}

func TestHandleStatusEffectExpired_VenomLastSlot_BroadcastsMonsterStatReset(t *testing.T) {
	// Synthesize 3 applies, then 2 expires.
	// Assert: no MonsterStatReset broadcast yet.
	// Then synthesize the 3rd expire.
	// Assert: exactly one MonsterStatReset(VENOM) is broadcast.
	t.Skip("Implementation in step 12.2")
}
```

- [x] **Step 12.2: Implement collapse logic**

Edit `handleStatusEffectApplied` — wrap the existing `MonsterStatSetWriter` call so VENOM is suppressed if the mirror already had ≥1 VENOM entry **before** this apply:

```go
// inside handleStatusEffectApplied, before announcing MonsterStatSet:
isVenom := false
for stat := range e.Body.Statuses {
    if stat == "VENOM" {
        isVenom = true
        break
    }
}

priorVenomCount := 0
if isVenom {
    priorVenomCount = monster.GetStatusMirror().VenomCount(t, e.UniqueId)
}

// (existing) populate mirror BEFORE deciding to broadcast for accurate counts
monster.GetStatusMirror().OnApplied(t, e.UniqueId, /*...body fields...*/)

// Decide whether to broadcast:
if isVenom && priorVenomCount > 0 {
    // suppress — collapse to existing VENOM presence
} else {
    // (existing MonsterStatSetWriter announce body, unchanged)
}
```

> Move the `OnApplied` call **above** the broadcast (currently it's at the end of the handler per T11). The order matters because `priorVenomCount` is queried *before* this apply lands in the mirror.

Edit `handleStatusEffectExpired` and `handleStatusEffectCancelled` — wrap the existing `MonsterStatResetWriter` for VENOM so it broadcasts only when the post-removal `VenomCount` is 0:

```go
// inside handleStatusEffectExpired, after the mirror.OnExpired call:
isVenom := false
for stat := range e.Body.Statuses {
    if stat == "VENOM" {
        isVenom = true
        break
    }
}

if isVenom {
    if monster.GetStatusMirror().VenomCount(t, e.UniqueId) > 0 {
        return // suppress — at least one VENOM remains
    }
}
// (existing MonsterStatResetWriter announce body)
```

> Mirror the same suppression in `handleStatusEffectCancelled`.

- [x] **Step 12.3: Run tests**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./kafka/consumer/monster/ -v
```

Expected: PASS.

- [x] **Step 12.4: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer_test.go
git commit -m "task-036: VENOM wire-collapse via VenomCount transition gate"
```

---

## Phase 2 — atlas-maps `mist` domain (parallel branch)

These tasks build the new `mist` package in atlas-maps. Sequence is internal: model → registry → processor + producer → command consumer → tick task → main.go wiring.

### Task 13: `mist.Mist` immutable model + builder

**Files:**
- Create: `services/atlas-maps/atlas.com/maps/mist/model.go`
- Test: `services/atlas-maps/atlas.com/maps/mist/model_test.go`

- [x] **Step 13.1: Write failing test**

Create `services/atlas-maps/atlas.com/maps/mist/model_test.go`:

```go
package mist

import (
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func mkField(t *testing.T) field.Model {
	t.Helper()
	return field.NewBuilder(0, 0, 100000000).SetInstance(uuid.Nil).Build()
}

func TestMistBuilder_BuildsImmutable(t *testing.T) {
	id := uuid.New()
	f := mkField(t)
	m := NewBuilder(id, f).
		SetOwner("MONSTER", 9001).
		SetOrigin(100, 200).
		SetBounds(-50, -30, 50, 30).
		SetDisease("POISON", 80, 30*time.Second).
		SetDuration(10 * time.Second).
		SetTickInterval(time.Second).
		Build()

	require.Equal(t, id, m.Id())
	require.Equal(t, "MONSTER", m.OwnerType())
	require.Equal(t, uint32(9001), m.OwnerId())
	require.Equal(t, int16(100), m.OriginX())
	require.Equal(t, int16(200), m.OriginY())
	require.Equal(t, int16(-50), m.LtX())
	require.Equal(t, int16(50), m.RbX())
	require.Equal(t, "POISON", m.Disease())
	require.Equal(t, int32(80), m.DiseaseValue())
	require.Equal(t, 30*time.Second, m.DiseaseDuration())
	require.Equal(t, 10*time.Second, m.Duration())
	require.Equal(t, time.Second, m.TickInterval())
}

func TestMist_Contains_InsideAndOutside(t *testing.T) {
	id := uuid.New()
	m := NewBuilder(id, mkField(t)).
		SetOrigin(100, 200).
		SetBounds(-50, -30, 50, 30).
		SetDuration(time.Second).
		Build()

	require.True(t, m.Contains(100, 200), "origin")
	require.True(t, m.Contains(150, 230), "max corner inclusive")
	require.True(t, m.Contains(50, 170), "min corner inclusive")
	require.False(t, m.Contains(151, 200), "outside x")
	require.False(t, m.Contains(100, 231), "outside y")
}

func TestMist_Expired_AfterDuration(t *testing.T) {
	id := uuid.New()
	m := NewBuilder(id, mkField(t)).
		SetOrigin(0, 0).
		SetBounds(-1, -1, 1, 1).
		SetDuration(0). // already expired
		Build()
	require.True(t, m.Expired())
}

func TestMist_ShouldTick_RespectsLastTick(t *testing.T) {
	id := uuid.New()
	m := NewBuilder(id, mkField(t)).
		SetOrigin(0, 0).
		SetBounds(-1, -1, 1, 1).
		SetDuration(time.Minute).
		SetTickInterval(time.Second).
		Build()
	require.True(t, m.ShouldTick(), "fresh mist, lastTick = createdAt - tickInterval")

	updated := m.WithLastTick(time.Now())
	require.False(t, updated.ShouldTick())
}
```

- [x] **Step 13.2: Run — confirm failure**

```bash
cd services/atlas-maps/atlas.com/maps && go test ./mist/ -v
```

Expected: FAIL — package missing.

- [x] **Step 13.3: Implement model + builder**

Create `services/atlas-maps/atlas.com/maps/mist/model.go`:

```go
package mist

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/google/uuid"
)

type Mist struct {
	id              uuid.UUID
	field           field.Model
	ownerType       string
	ownerId         uint32
	originX         int16
	originY         int16
	ltX             int16
	ltY             int16
	rbX             int16
	rbY             int16
	disease         string
	diseaseValue    int32
	diseaseDuration time.Duration
	duration        time.Duration
	tickInterval    time.Duration
	sourceSkillId   uint32
	sourceSkillLevel uint32
	createdAt       time.Time
	expiresAt       time.Time
	lastTick        time.Time
}

func (m Mist) Id() uuid.UUID                  { return m.id }
func (m Mist) Field() field.Model             { return m.field }
func (m Mist) OwnerType() string              { return m.ownerType }
func (m Mist) OwnerId() uint32                { return m.ownerId }
func (m Mist) OriginX() int16                 { return m.originX }
func (m Mist) OriginY() int16                 { return m.originY }
func (m Mist) LtX() int16                     { return m.ltX }
func (m Mist) LtY() int16                     { return m.ltY }
func (m Mist) RbX() int16                     { return m.rbX }
func (m Mist) RbY() int16                     { return m.rbY }
func (m Mist) Disease() string                { return m.disease }
func (m Mist) DiseaseValue() int32            { return m.diseaseValue }
func (m Mist) DiseaseDuration() time.Duration { return m.diseaseDuration }
func (m Mist) Duration() time.Duration        { return m.duration }
func (m Mist) TickInterval() time.Duration    { return m.tickInterval }
func (m Mist) SourceSkillId() uint32          { return m.sourceSkillId }
func (m Mist) SourceSkillLevel() uint32       { return m.sourceSkillLevel }
func (m Mist) CreatedAt() time.Time           { return m.createdAt }
func (m Mist) ExpiresAt() time.Time           { return m.expiresAt }
func (m Mist) LastTick() time.Time            { return m.lastTick }

func (m Mist) Contains(x, y int16) bool {
	minX := m.originX + m.ltX
	maxX := m.originX + m.rbX
	minY := m.originY + m.ltY
	maxY := m.originY + m.rbY
	return x >= minX && x <= maxX && y >= minY && y <= maxY
}

func (m Mist) Expired() bool { return time.Now().After(m.expiresAt) }

func (m Mist) ShouldTick() bool {
	if m.tickInterval <= 0 {
		return false
	}
	return time.Since(m.lastTick) >= m.tickInterval
}

func (m Mist) WithLastTick(t time.Time) Mist {
	m.lastTick = t
	return m
}

type Builder struct {
	m Mist
}

func NewBuilder(id uuid.UUID, f field.Model) *Builder {
	now := time.Now()
	return &Builder{
		m: Mist{
			id:        id,
			field:     f,
			createdAt: now,
			expiresAt: now,
			// lastTick set so ShouldTick is true on first tick.
			lastTick: now.Add(-365 * 24 * time.Hour),
		},
	}
}

func (b *Builder) SetOwner(ownerType string, ownerId uint32) *Builder {
	b.m.ownerType = ownerType
	b.m.ownerId = ownerId
	return b
}

func (b *Builder) SetOrigin(x, y int16) *Builder {
	b.m.originX = x
	b.m.originY = y
	return b
}

func (b *Builder) SetBounds(ltX, ltY, rbX, rbY int16) *Builder {
	b.m.ltX = ltX
	b.m.ltY = ltY
	b.m.rbX = rbX
	b.m.rbY = rbY
	return b
}

func (b *Builder) SetDisease(name string, value int32, dur time.Duration) *Builder {
	b.m.disease = name
	b.m.diseaseValue = value
	b.m.diseaseDuration = dur
	return b
}

func (b *Builder) SetDuration(d time.Duration) *Builder {
	b.m.duration = d
	b.m.expiresAt = b.m.createdAt.Add(d)
	return b
}

func (b *Builder) SetTickInterval(d time.Duration) *Builder {
	b.m.tickInterval = d
	return b
}

func (b *Builder) SetSource(skillId, skillLevel uint32) *Builder {
	b.m.sourceSkillId = skillId
	b.m.sourceSkillLevel = skillLevel
	return b
}

func (b *Builder) Build() Mist { return b.m }
```

- [x] **Step 13.4: Run tests — confirm passing**

```bash
cd services/atlas-maps/atlas.com/maps && go test ./mist/ -v
```

Expected: PASS.

- [x] **Step 13.5: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/mist/model.go services/atlas-maps/atlas.com/maps/mist/model_test.go
git commit -m "task-036: mist.Mist immutable model and builder"
```

---

### Task 14: `mist.Registry`

**Files:**
- Create: `services/atlas-maps/atlas.com/maps/mist/registry.go`
- Test: `services/atlas-maps/atlas.com/maps/mist/registry_test.go`

- [x] **Step 14.1: Write failing test**

```go
package mist

import (
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/tenant"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func mkRegTenant() tenant.Model { /* same helper as T10 */ }

func TestRegistry_Add_GetByField(t *testing.T) {
	r := newTestMistRegistry()
	tt := mkRegTenant()
	f := field.NewBuilder(0, 0, 100000000).SetInstance(uuid.Nil).Build()
	id := uuid.New()
	m := NewBuilder(id, f).SetOrigin(0, 0).SetBounds(-1, -1, 1, 1).SetDuration(time.Minute).Build()

	require.NoError(t, r.Add(tt, m))
	got := r.GetByField(tt, f)
	require.Len(t, got, 1)
	require.Equal(t, id, got[0].Id())
}

func TestRegistry_Remove_ReturnsRemovedMist(t *testing.T) {
	r := newTestMistRegistry()
	tt := mkRegTenant()
	f := field.NewBuilder(0, 0, 100000000).SetInstance(uuid.Nil).Build()
	id := uuid.New()
	m := NewBuilder(id, f).SetOrigin(0, 0).SetBounds(-1, -1, 1, 1).SetDuration(time.Minute).Build()
	_ = r.Add(tt, m)

	removed, err := r.Remove(tt, id)
	require.NoError(t, err)
	require.Equal(t, id, removed.Id())
	require.Empty(t, r.GetByField(tt, f))
}

func TestRegistry_GetByField_DistinguishesInstances(t *testing.T) {
	r := newTestMistRegistry()
	tt := mkRegTenant()
	f1 := field.NewBuilder(0, 0, 100000000).SetInstance(uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000001")).Build()
	f2 := field.NewBuilder(0, 0, 100000000).SetInstance(uuid.MustParse("bbbbbbbb-0000-0000-0000-000000000002")).Build()
	mistA := NewBuilder(uuid.New(), f1).SetOrigin(0, 0).SetBounds(-1, -1, 1, 1).SetDuration(time.Minute).Build()
	_ = r.Add(tt, mistA)

	require.Len(t, r.GetByField(tt, f1), 1)
	require.Len(t, r.GetByField(tt, f2), 0, "different instance UUID — no overlap")
}

func TestRegistry_AllByTenant_ReturnsAcrossFields(t *testing.T) {
	r := newTestMistRegistry()
	tt := mkRegTenant()
	f1 := field.NewBuilder(0, 0, 100000000).SetInstance(uuid.Nil).Build()
	f2 := field.NewBuilder(0, 0, 200000000).SetInstance(uuid.Nil).Build()
	_ = r.Add(tt, NewBuilder(uuid.New(), f1).SetDuration(time.Minute).Build())
	_ = r.Add(tt, NewBuilder(uuid.New(), f2).SetDuration(time.Minute).Build())

	require.Len(t, r.AllByTenant(tt), 2)
}

func newTestMistRegistry() *Registry {
	return &Registry{perTenant: map[string]map[uuid.UUID]Mist{}}
}
```

- [x] **Step 14.2: Run — confirm failure**

- [x] **Step 14.3: Implement registry**

Create `services/atlas-maps/atlas.com/maps/mist/registry.go`:

```go
package mist

import (
	"errors"
	"sync"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/tenant"
	"github.com/google/uuid"
)

type Registry struct {
	mu        sync.RWMutex
	perTenant map[string]map[uuid.UUID]Mist
}

var (
	registryOnce sync.Once
	registry     *Registry
)

func GetRegistry() *Registry {
	registryOnce.Do(func() {
		registry = &Registry{perTenant: map[string]map[uuid.UUID]Mist{}}
	})
	return registry
}

func tenantKey(t tenant.Model) string { return t.Id().String() }

func (r *Registry) Add(t tenant.Model, m Mist) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	tk := tenantKey(t)
	if r.perTenant[tk] == nil {
		r.perTenant[tk] = map[uuid.UUID]Mist{}
	}
	if _, exists := r.perTenant[tk][m.Id()]; exists {
		return errors.New("mist with id already exists")
	}
	r.perTenant[tk][m.Id()] = m
	return nil
}

func (r *Registry) Remove(t tenant.Model, id uuid.UUID) (Mist, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	tk := tenantKey(t)
	if r.perTenant[tk] == nil {
		return Mist{}, errors.New("tenant has no mists")
	}
	m, ok := r.perTenant[tk][id]
	if !ok {
		return Mist{}, errors.New("mist not found")
	}
	delete(r.perTenant[tk], id)
	return m, nil
}

func (r *Registry) GetByField(t tenant.Model, f field.Model) []Mist {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tk := tenantKey(t)
	out := make([]Mist, 0)
	for _, m := range r.perTenant[tk] {
		if m.Field().Equals(f) {
			out = append(out, m)
		}
	}
	return out
}

func (r *Registry) AllByTenant(t tenant.Model) []Mist {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tk := tenantKey(t)
	out := make([]Mist, 0, len(r.perTenant[tk]))
	for _, m := range r.perTenant[tk] {
		out = append(out, m)
	}
	return out
}

func (r *Registry) UpdateLastTick(t tenant.Model, id uuid.UUID, at time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	tk := tenantKey(t)
	if r.perTenant[tk] == nil {
		return
	}
	if m, ok := r.perTenant[tk][id]; ok {
		r.perTenant[tk][id] = m.WithLastTick(at)
	}
}

func (r *Registry) GetTenants() []tenant.Model {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]tenant.Model, 0, len(r.perTenant))
	for tk := range r.perTenant {
		// Reconstruct tenant from the key. If tenant.Model needs more than the
		// id (region/major/minor), the registry should store the tenant.Model
		// alongside the key — refactor here to a struct value.
		// TODO(plan): replace with proper tenant index if Model is opaque.
		_ = tk
	}
	return out
}
```

> The `GetTenants()` shape is a known gap — `tenant.Model` is more than just a UUID. Replace `perTenant map[string]...` with `map[string]struct{Tenant tenant.Model; Mists map[uuid.UUID]Mist}` so we keep the full Model. Adjust accessor methods. Step 14.4 codifies this.

- [x] **Step 14.4: Refactor registry to keep full `tenant.Model`**

Replace `perTenant map[string]map[uuid.UUID]Mist` with a struct value that holds the tenant alongside its mists. Update all accessors. Re-run tests.

- [x] **Step 14.5: Run tests — confirm passing**

- [x] **Step 14.6: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/mist/registry.go services/atlas-maps/atlas.com/maps/mist/registry_test.go
git commit -m "task-036: mist.Registry tenant-scoped index"
```

---

### Task 15: `mist.Processor` + Kafka producer (`MIST_CREATED` / `MIST_DESTROYED`)

**Files:**
- Create: `services/atlas-maps/atlas.com/maps/mist/processor.go`
- Create: `services/atlas-maps/atlas.com/maps/mist/producer.go`
- Create: `services/atlas-maps/atlas.com/maps/kafka/message/mist/kafka.go`
- Test: `services/atlas-maps/atlas.com/maps/mist/processor_test.go`

- [x] **Step 15.1: Define Kafka shapes**

Create `services/atlas-maps/atlas.com/maps/kafka/message/mist/kafka.go`:

```go
package mist

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvCommandTopic = "COMMAND_TOPIC_MIST"
	EnvEventTopic   = "EVENT_TOPIC_MIST"

	CommandTypeCreate = "CREATE"
	CommandTypeCancel = "CANCEL"

	EventTypeCreated   = "MIST_CREATED"
	EventTypeDestroyed = "MIST_DESTROYED"

	ReasonExpired   = "EXPIRED"
	ReasonCancelled = "CANCELLED"
)

type Command[E any] struct {
	Tenant uuid.UUID `json:"tenant"`
	Type   string    `json:"type"`
	Body   E         `json:"body"`
}

type CreateCommandBody struct {
	WorldId          world.Id   `json:"worldId"`
	ChannelId        channel.Id `json:"channelId"`
	MapId            _map.Id    `json:"mapId"`
	Instance         uuid.UUID  `json:"instance"`
	OwnerType        string     `json:"ownerType"`
	OwnerId          uint32     `json:"ownerId"`
	OriginX          int16      `json:"originX"`
	OriginY          int16      `json:"originY"`
	LtX              int16      `json:"ltX"`
	LtY              int16      `json:"ltY"`
	RbX              int16      `json:"rbX"`
	RbY              int16      `json:"rbY"`
	Disease          string     `json:"disease"`
	DiseaseValue     int32      `json:"diseaseValue"`
	DiseaseDuration  int64      `json:"diseaseDuration"`
	Duration         int64      `json:"duration"`
	TickIntervalMs   int64      `json:"tickIntervalMs"`
	SourceSkillId    uint32     `json:"sourceSkillId"`
	SourceSkillLevel uint32     `json:"sourceSkillLevel"`
}

type CancelCommandBody struct {
	MistId uuid.UUID `json:"mistId"`
}

type Event[E any] struct {
	Tenant    uuid.UUID  `json:"tenant"`
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	MistId    uuid.UUID  `json:"mistId"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
}

type CreatedBody struct {
	OwnerType string `json:"ownerType"`
	OwnerId   uint32 `json:"ownerId"`
	OriginX   int16  `json:"originX"`
	OriginY   int16  `json:"originY"`
	LtX       int16  `json:"ltX"`
	LtY       int16  `json:"ltY"`
	RbX       int16  `json:"rbX"`
	RbY       int16  `json:"rbY"`
	Duration  int64  `json:"duration"`
}

type DestroyedBody struct {
	Reason string `json:"reason"`
}
```

- [x] **Step 15.2: Write failing test for processor**

Create `services/atlas-maps/atlas.com/maps/mist/processor_test.go`:

```go
package mist

import (
	"context"
	"testing"
	"time"

	mistKafka "atlas-maps/kafka/message/mist"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestProcessor_Create_AddsToRegistryAndEmitsCreated(t *testing.T) {
	ctx := context.Background()
	tt := mkRegTenant()
	ctx = tenant.WithContext(ctx, tt)

	body := mistKafka.CreateCommandBody{
		WorldId: 0, ChannelId: 0, MapId: 100000000, Instance: uuid.Nil,
		OwnerType: "MONSTER", OwnerId: 9001,
		OriginX: 100, OriginY: 200,
		LtX: -50, LtY: -30, RbX: 50, RbY: 30,
		Disease: "POISON", DiseaseValue: 80, DiseaseDuration: 30000,
		Duration: 10000, TickIntervalMs: 1000,
	}

	emitted := []emittedEvent{}
	p := newTestProcessor(logrus.New(), ctx, &emitted)

	m, err := p.Create(body)
	require.NoError(t, err)
	require.Equal(t, "POISON", m.Disease())
	require.Len(t, emitted, 1, "expected MIST_CREATED event")
	require.Equal(t, mistKafka.EventTypeCreated, emitted[0].eventType)
}

func TestProcessor_Destroy_RemovesAndEmits(t *testing.T) {
	// Symmetric — assert MIST_DESTROYED with reason EXPIRED.
}
```

- [x] **Step 15.3: Implement processor + producer**

Create `services/atlas-maps/atlas.com/maps/mist/processor.go`:

```go
package mist

import (
	"context"
	"time"

	mistKafka "atlas-maps/kafka/message/mist"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/tenant"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	Create(body mistKafka.CreateCommandBody) (Mist, error)
	Destroy(id uuid.UUID, reason string) (Mist, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
	p   producer.Provider
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
		p:   producer.ProviderImpl(l)(ctx),
	}
}

func (p *ProcessorImpl) Create(body mistKafka.CreateCommandBody) (Mist, error) {
	id := uuid.New()
	f := field.NewBuilder(body.WorldId, body.ChannelId, body.MapId).SetInstance(body.Instance).Build()
	m := NewBuilder(id, f).
		SetOwner(body.OwnerType, body.OwnerId).
		SetOrigin(body.OriginX, body.OriginY).
		SetBounds(body.LtX, body.LtY, body.RbX, body.RbY).
		SetDisease(body.Disease, body.DiseaseValue, time.Duration(body.DiseaseDuration)*time.Millisecond).
		SetDuration(time.Duration(body.Duration) * time.Millisecond).
		SetTickInterval(time.Duration(body.TickIntervalMs) * time.Millisecond).
		SetSource(body.SourceSkillId, body.SourceSkillLevel).
		Build()

	if err := GetRegistry().Add(p.t, m); err != nil {
		return Mist{}, err
	}

	if err := message.Emit(p.l, p.ctx)(func(buf *message.Buffer) error {
		return buf.Put(mistKafka.EnvEventTopic, createdEventProvider(p.t, m))
	}); err != nil {
		// Roll back the registry insert if event emission fails.
		_, _ = GetRegistry().Remove(p.t, id)
		return Mist{}, err
	}
	return m, nil
}

func (p *ProcessorImpl) Destroy(id uuid.UUID, reason string) (Mist, error) {
	m, err := GetRegistry().Remove(p.t, id)
	if err != nil {
		return Mist{}, err
	}
	if err := message.Emit(p.l, p.ctx)(func(buf *message.Buffer) error {
		return buf.Put(mistKafka.EnvEventTopic, destroyedEventProvider(p.t, m, reason))
	}); err != nil {
		p.l.WithError(err).Errorf("Unable to emit MIST_DESTROYED for [%s].", id)
	}
	return m, nil
}
```

Create `services/atlas-maps/atlas.com/maps/mist/producer.go`:

```go
package mist

import (
	mistKafka "atlas-maps/kafka/message/mist"
	"github.com/Chronicle20/atlas/libs/atlas-constants/tenant"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/google/uuid"
)

func createdEventProvider(t tenant.Model, m Mist) producer.MessageProvider {
	body := mistKafka.CreatedBody{
		OwnerType: m.OwnerType(),
		OwnerId:   m.OwnerId(),
		OriginX:   m.OriginX(),
		OriginY:   m.OriginY(),
		LtX:       m.LtX(),
		LtY:       m.LtY(),
		RbX:       m.RbX(),
		RbY:       m.RbY(),
		Duration:  int64(m.Duration() / time.Millisecond),
	}
	event := mistKafka.Event[mistKafka.CreatedBody]{
		Tenant:    t.Id(),
		WorldId:   m.Field().WorldId(),
		ChannelId: m.Field().ChannelId(),
		MapId:     m.Field().MapId(),
		Instance:  m.Field().Instance(),
		MistId:    m.Id(),
		Type:      mistKafka.EventTypeCreated,
		Body:      body,
	}
	return producer.SingleMessageProvider(m.Id().String(), event)
}

func destroyedEventProvider(t tenant.Model, m Mist, reason string) producer.MessageProvider {
	event := mistKafka.Event[mistKafka.DestroyedBody]{
		Tenant:    t.Id(),
		WorldId:   m.Field().WorldId(),
		ChannelId: m.Field().ChannelId(),
		MapId:     m.Field().MapId(),
		Instance:  m.Field().Instance(),
		MistId:    m.Id(),
		Type:      mistKafka.EventTypeDestroyed,
		Body:      mistKafka.DestroyedBody{Reason: reason},
	}
	return producer.SingleMessageProvider(m.Id().String(), event)
}
```

> The exact `producer.MessageProvider` / `producer.SingleMessageProvider` signatures are project-specific. Read `services/atlas-maps/atlas.com/maps/reactor/producer.go` for the canonical pattern and copy the helper invocation verbatim.

- [x] **Step 15.4: Run tests — confirm passing**

- [x] **Step 15.5: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/kafka/message/mist/ services/atlas-maps/atlas.com/maps/mist/processor.go services/atlas-maps/atlas.com/maps/mist/producer.go services/atlas-maps/atlas.com/maps/mist/processor_test.go
git commit -m "task-036: mist.Processor + Kafka producer (MIST_CREATED/MIST_DESTROYED)"
```

---

### Task 16: Mist command consumer (`MIST_CREATE` / `MIST_CANCEL`)

**Files:**
- Create: `services/atlas-maps/atlas.com/maps/kafka/consumer/mist/consumer.go`
- Modify: `services/atlas-maps/atlas.com/maps/kafka/consumer/mist/init.go` (or whatever file declares `InitConsumers` for atlas-maps)
- Test: `services/atlas-maps/atlas.com/maps/kafka/consumer/mist/consumer_test.go`

- [x] **Step 16.1: Read existing consumer template**

Read an existing atlas-maps consumer (e.g. `services/atlas-maps/atlas.com/maps/kafka/consumer/reactor/` if it exists; otherwise the atlas-monsters consumer pattern). Match the `InitConsumers(l)(cmf)(groupId)` curry shape.

- [x] **Step 16.2: Write failing test**

Test that a synthesized `MIST_CREATE` command results in a registry insert + event emission.

- [x] **Step 16.3: Implement consumer**

```go
package mist

import (
	"context"

	mistDomain "atlas-maps/mist"
	mistKafka "atlas-maps/kafka/message/mist"
	"github.com/Chronicle20/atlas/libs/atlas-constants/tenant"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(cmf consumer.ConsumerManagerFactory) func(groupId string) {
	return func(cmf consumer.ConsumerManagerFactory) func(groupId string) {
		return func(groupId string) {
			cmf(l)(mistKafka.EnvCommandTopic)(groupId)
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rh handler.RegisterHandler) error {
	return func(rh handler.RegisterHandler) error {
		// Generic command consumer; switch on Type.
		return rh(l)(mistKafka.EnvCommandTopic, "mist-command", func(ctx context.Context, cmd mistKafka.Command[any]) {
			tCtx, err := tenantCtx(ctx, cmd.Tenant)
			if err != nil {
				l.WithError(err).Error("Mist command: bad tenant header.")
				return
			}
			switch cmd.Type {
			case mistKafka.CommandTypeCreate:
				body, ok := decodeCreateBody(cmd.Body)
				if !ok {
					l.Error("Mist command: malformed CREATE body.")
					return
				}
				if _, err := mistDomain.NewProcessor(l, tCtx).Create(body); err != nil {
					l.WithError(err).Error("Mist command: Create failed.")
				}
			case mistKafka.CommandTypeCancel:
				body, ok := decodeCancelBody(cmd.Body)
				if !ok {
					l.Error("Mist command: malformed CANCEL body.")
					return
				}
				if _, err := mistDomain.NewProcessor(l, tCtx).Destroy(body.MistId, mistKafka.ReasonCancelled); err != nil {
					l.WithError(err).Errorf("Mist command: Destroy [%s] failed.", body.MistId)
				}
			default:
				l.Warnf("Mist command: unknown type [%s].", cmd.Type)
			}
		})
	}
}

func tenantCtx(ctx context.Context, tId uuid.UUID) (context.Context, error) {
	t, err := tenant.GetById(tId)
	if err != nil {
		return ctx, err
	}
	return tenant.WithContext(ctx, t), nil
}
```

> The exact handler-registration API and tenant-from-id resolver depend on what atlas-maps already uses. Match the in-tree pattern. The two `decode*Body` helpers convert the generic JSON body to the typed body — see how reactor consumer does this.

- [x] **Step 16.4: Run tests — confirm passing**

- [x] **Step 16.5: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/kafka/consumer/mist/ services/atlas-maps/atlas.com/maps/kafka/consumer/mist/consumer_test.go
git commit -m "task-036: mist command consumer (MIST_CREATE/MIST_CANCEL)"
```

---

### Task 17: `MistTickTask`

**Why:** FR-4.6.3 — every 1 s, expire / re-apply disease per active mist.

**Files:**
- Create: `services/atlas-maps/atlas.com/maps/tasks/mist_tick.go`
- Test: `services/atlas-maps/atlas.com/maps/tasks/mist_tick_test.go`

- [x] **Step 17.1: Write failing test**

```go
func TestMistTick_ExpiredMist_DestroysAndEmits(t *testing.T) {
	// Add an expired mist to the registry.
	// Run task once.
	// Assert: registry empty; MIST_DESTROYED event emitted with reason EXPIRED.
}

func TestMistTick_LiveMist_AppliesDiseaseToContainedCharacters(t *testing.T) {
	// Add a live mist + 2 characters in field; one inside Contains(), one outside.
	// Run task once.
	// Assert: only the inside character receives an apply-disease command.
}

func TestMistTick_DifferentInstances_DoNotCrossApply(t *testing.T) {
	// Risks §6 — instance map isolation.
	// Two fields same mapId, different Instance UUIDs.
	// Mist on instance A, character on instance B.
	// Run task once.
	// Assert: no apply-disease command for the instance-B character.
}
```

- [x] **Step 17.2: Implement task**

```go
package tasks

import (
	"context"
	"time"

	mapchar "atlas-maps/map/character"
	mistDomain "atlas-maps/mist"
	mistKafka "atlas-maps/kafka/message/mist"
	"github.com/Chronicle20/atlas/libs/atlas-constants/tenant"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
)

type MistTick struct {
	l        logrus.FieldLogger
	interval int
}

func NewMistTick(l logrus.FieldLogger, interval int) *MistTick {
	return &MistTick{l: l, interval: interval}
}

func (r *MistTick) Run() {
	ctx, span := otel.GetTracerProvider().Tracer("atlas-maps").Start(context.Background(), "mist_tick_task")
	defer span.End()

	tenants := mistDomain.GetRegistry().GetTenants()
	for _, t := range tenants {
		go r.processTenant(ctx, t)
	}
}

func (r *MistTick) processTenant(ctx context.Context, t tenant.Model) {
	tctx := tenant.WithContext(ctx, t)
	mists := mistDomain.GetRegistry().AllByTenant(t)
	if len(mists) == 0 {
		return
	}

	for _, m := range mists {
		if m.Expired() {
			if _, err := mistDomain.NewProcessor(r.l, tctx).Destroy(m.Id(), mistKafka.ReasonExpired); err != nil {
				r.l.WithError(err).Errorf("MistTick: failed to destroy expired mist [%s].", m.Id())
			}
			continue
		}
		if !m.ShouldTick() {
			continue
		}
		// Membership lookup (see map.character.Registry).
		key := /* construct MapKey from t + m.Field() */
		members := mapchar.GetRegistry().GetInMap(key)

		// Apply disease to contained characters.
		_ = message.Emit(r.l, tctx)(func(buf *message.Buffer) error {
			for _, cid := range members {
				pos, err := /* REST GET /characters/{cid} */
				if err != nil {
					r.l.WithError(err).Debugf("MistTick: position fetch failed for [%d].", cid)
					continue
				}
				if !m.Contains(pos.X, pos.Y) {
					continue
				}
				if err := buf.Put(/* EnvCommandTopicCharacterBuff */, applyDiseaseProvider(m, cid)); err != nil {
					return err
				}
			}
			return nil
		})

		mistDomain.GetRegistry().UpdateLastTick(t, m.Id(), time.Now())
		r.l.Debugf("mist: zone [%s] applied %s ticked %d members.", m.Id(), m.Disease(), len(members))
	}
}

func (r *MistTick) SleepTime() time.Duration {
	return time.Millisecond * time.Duration(r.interval)
}
```

> The `MapKey` construction, `mapchar.GetRegistry()` API, character REST client, and the `applyDiseaseProvider` helper all need to be sourced from existing code (`services/atlas-maps/atlas.com/maps/map/character/registry.go:13-99`, the character REST client used by atlas-maps elsewhere, and atlas-monsters' `disease.go:45-63` for the body shape — but mirror the body shape on the atlas-maps side; do not import across services). Plan-phase action: locate the atlas-character REST client used by atlas-maps for any other purpose; if none exists, T19 adds one.

- [x] **Step 17.3: Run tests — confirm passing**

- [x] **Step 17.4: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/tasks/mist_tick.go services/atlas-maps/atlas.com/maps/tasks/mist_tick_test.go
git commit -m "task-036: MistTickTask — 1s tick + expire + disease re-apply"
```

---

### Task 18: Wire mist consumer + tick task in `atlas-maps/main.go`

**Files:**
- Modify: `services/atlas-maps/atlas.com/maps/main.go`

- [x] **Step 18.1: Add registrations**

```go
// Beside existing consumer Init calls:
mistConsumer.InitConsumers(l)(cmf)(consumerGroupId)
if err := mistConsumer.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
    l.WithError(err).Fatal("Unable to register mist kafka handlers.")
}

// Beside existing tasks (e.g. tasks.Register(tasks.NewRespawn(...)) ):
go tasks.Register(tasks.NewMistTick(l, 1000))
```

- [x] **Step 18.2: Build atlas-maps**

```bash
cd services/atlas-maps/atlas.com/maps && go build ./...
```

Expected: PASS.

- [x] **Step 18.3: Run all atlas-maps tests**

```bash
cd services/atlas-maps/atlas.com/maps && go test ./...
```

Expected: PASS.

- [x] **Step 18.4: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/main.go
git commit -m "task-036: wire mist consumer + MistTickTask in atlas-maps main"
```

---

### Task 19: Verify atlas-character position lookup

**Why:** `MistTickTask` needs `(x, y)` per character to filter by `mist.Contains`. Per explore, atlas-character's `RestModel` already exposes `X`, `Y`, `MapId` from the temporal registry (`rest.go:40-45, 109-111`). Confirm this can be reused; only add a thin endpoint if not.

**Files:**
- Verify: `services/atlas-character/atlas.com/character/character/rest.go:40-45, 109-111`
- Verify: atlas-character REST client used by atlas-maps (search)
- Optional create: `services/atlas-character/atlas.com/character/character/handler.go` (position-only endpoint)

- [x] **Step 19.1: Inspect existing GET /characters/{id} response**

```bash
grep -rn 'X\|Y\|MapId' services/atlas-character/atlas.com/character/character/rest.go | head -20
```

Confirm `X`, `Y`, `MapId` are present in the JSON response. If yes, the existing endpoint suffices.

- [x] **Step 19.2: Locate atlas-maps' character REST client**

```bash
grep -rn 'characters/\|GetCharacter' services/atlas-maps/atlas.com/maps/ | head -20
```

If a client exists, integrate it into `MistTickTask`. If not, write a thin client that GETs `/api/characters/{id}` and returns `{X, Y, MapId, Instance}`.

- [x] **Step 19.3: Write test**

```go
func TestMistTick_FetchesPosition_FiltersInsideOnly(t *testing.T) {
	// Mock the character REST client to return position (50, 50) for cid=1
	// and (1000, 1000) for cid=2.
	// Run tick on a mist at (0,0) bounds (-100..100).
	// Assert only cid=1's apply-disease command emitted.
}
```

- [x] **Step 19.4: Wire client into MistTickTask**

Replace the `/* REST GET /characters/{cid} */` placeholder in T17 with the concrete client invocation.

- [x] **Step 19.5: Run tests, commit**

```bash
cd services/atlas-maps/atlas.com/maps && go test ./tasks/ -v
```

Expected: PASS.

```bash
git add services/atlas-maps/atlas.com/maps/tasks/mist_tick.go services/atlas-maps/atlas.com/maps/tasks/mist_tick_test.go
git commit -m "task-036: wire atlas-character position client into MistTickTask"
```

---

## Phase 3 — Top-tier integrations

### Task 20: `executeMist` in atlas-monsters + producer

**Why:** FR-4.6.5 — atlas-monsters fires `MIST_CREATE` commands on `EVENT_COMMAND_TOPIC_MIST` when an `AREA_POISON`-bearing skill executes.

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/processor.go` (add `executeMist`, dispatch from category switch)
- Create / extend: `services/atlas-monsters/atlas.com/monsters/monster/producer.go` — add `mistCreateCommandProvider`
- Create: `services/atlas-monsters/atlas.com/monsters/kafka/message/mist/kafka.go` — mirror atlas-maps mist command shapes (consumer-side import boundary)

- [x] **Step 20.1: Define atlas-monsters' mist-command body**

Mirror `services/atlas-maps/atlas.com/maps/kafka/message/mist/kafka.go` in atlas-monsters at `services/atlas-monsters/atlas.com/monsters/kafka/message/mist/kafka.go`. **Do not import across services** — copy the constants and body types.

- [x] **Step 20.2: Write failing test**

```go
func TestExecuteMist_ProducesMistCreateCommand(t *testing.T) {
	// Construct a fake mob skill: AREA_POISON (skill type 131), x=80, ltX=-50, ltY=-30, rbX=50, rbY=30, duration=10.
	// Call processor.executeMist(monster, sd, 131, 5).
	// Assert: a MIST_CREATE command was buffered with body matching the skill data.
}
```

- [x] **Step 20.3: Implement `executeMist`**

In `processor.go`:

```go
func (p *ProcessorImpl) executeMist(m Model, sd mobskill.Model, skillId byte, skillLevel byte) {
    body := mistKafka.CreateCommandBody{
        WorldId:          m.Field().WorldId(),
        ChannelId:        m.Field().ChannelId(),
        MapId:            m.Field().MapId(),
        Instance:         m.Field().Instance(),
        OwnerType:        "MONSTER",
        OwnerId:          m.UniqueId(),
        OriginX:          int16(m.X()),
        OriginY:          int16(m.Y()),
        LtX:              int16(sd.LtX()),
        LtY:              int16(sd.LtY()),
        RbX:              int16(sd.RbX()),
        RbY:              int16(sd.RbY()),
        Disease:          "POISON",
        DiseaseValue:     sd.X(),
        DiseaseDuration:  int64(sd.Duration()) * int64(time.Second/time.Millisecond),
        Duration:         int64(sd.Duration()) * int64(time.Second/time.Millisecond),
        TickIntervalMs:   1000,
        SourceSkillId:    uint32(skillId),
        SourceSkillLevel: uint32(skillLevel),
    }

    if err := message.Emit(p.l, p.ctx)(func(buf *message.Buffer) error {
        return buf.Put(mistKafka.EnvCommandTopic, mistCreateCommandProvider(p.t, body))
    }); err != nil {
        p.l.WithError(err).Errorf("Unable to emit MIST_CREATE for monster [%d].", m.UniqueId())
    }
}
```

> Cap mist duration per risks §2 — if `body.Duration > 60_000`, log a warning and clamp to 60_000.

- [x] **Step 20.4: Add `executeMist` to the category switch**

In `processor.UseSkill` (line 555-568) and `UseSkillGM` (line 632-644), add a new case for the mist skill. There is no `SkillCategoryMist` constant today — design D2 keeps mist outside `SkillCategory`. Instead, branch on `SkillTypeAreaPoison`:

```go
executeEffect := func() {
    if uint16(skillId) == monster2.SkillTypeAreaPoison {
        p.executeMist(m, sd, skillId, skillLevel)
        return
    }
    switch category {
    case monster2.SkillCategoryStatBuff, monster2.SkillCategoryImmunity, monster2.SkillCategoryReflect:
        p.executeStatBuff(m, sd, skillId, skillLevel)
    // ...existing cases
    }
}
```

> Apply the same special-case in `UseSkillGM`.

- [x] **Step 20.5: Run tests**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test ./monster/ -run 'TestExecuteMist' -v
```

Expected: PASS.

- [x] **Step 20.6: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/processor.go services/atlas-monsters/atlas.com/monsters/monster/producer.go services/atlas-monsters/atlas.com/monsters/kafka/message/mist/
git commit -m "task-036: executeMist + MIST_CREATE producer in atlas-monsters"
```

---

### Task 21: Picker un-skip `AREA_POISON`

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/picker.go:144-149`
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/picker_test.go` (find `TestPicker_AreaPoisonExcluded`)

- [x] **Step 21.1: Update the test to assert the picker fires AREA_POISON**

Find `TestPicker_AreaPoisonExcluded` in `picker_test.go`. Rename to `TestPicker_AreaPoisonAllowed` and invert the assertion:

```go
func TestPicker_AreaPoisonAllowed(t *testing.T) {
	// Set up monster with only an AREA_POISON skill.
	// Run pickNextSkill.
	// Assert: result is NOT a sentinel; SkillId == SkillTypeAreaPoison.
}
```

- [x] **Step 21.2: Run — confirm failure**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test ./monster/ -run 'TestPicker_AreaPoison' -v
```

Expected: FAIL — picker still skips.

- [x] **Step 21.3: Remove the exclusion**

Edit `picker.go:144-149` — delete the `if skillId16 == monster2.SkillTypeAreaPoison { ... continue }` block.

- [x] **Step 21.4: Run — confirm passing**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test ./monster/ -run 'TestPicker' -v
```

Expected: PASS for all picker tests.

- [x] **Step 21.5: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/picker.go services/atlas-monsters/atlas.com/monsters/monster/picker_test.go
git commit -m "task-036: picker un-skip AREA_POISON now that executeMist exists"
```

---

### Task 22: atlas-channel mist consumer + AffectedArea broadcast

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/kafka/consumer/mist/consumer.go`
- Modify: atlas-channel main.go to register the new consumer

- [x] **Step 22.1: Define a channel-side mist event body type matching atlas-maps**

Create `services/atlas-channel/atlas.com/channel/kafka/message/mist/kafka.go` mirroring atlas-maps' `EnvEventTopic`, `Event[T]`, `CreatedBody`, `DestroyedBody`.

- [x] **Step 22.2: Write failing test**

```go
func TestMistCreated_BroadcastsAffectedAreaCreatedToFieldSessions(t *testing.T) {
	// Stub session.ForSessionsInMap to capture writer invocations.
	// Synthesize an EVENT_TOPIC_MIST MIST_CREATED event for field F.
	// Assert: writer name AffectedAreaCreated, body matches mistId/origin/bounds/duration.
}

func TestMistDestroyed_BroadcastsAffectedAreaRemoved(t *testing.T) { /* symmetric */ }
```

- [x] **Step 22.3: Implement consumer**

```go
package mist

import (
	"context"

	mistKafka "atlas-channel/kafka/message/mist"
	_map "atlas-channel/map"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/tenant"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	fieldpkt "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/sirupsen/logrus"
)

func handleMistCreated(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, e mistKafka.Event[mistKafka.CreatedBody]) {
	t, err := tenant.GetById(e.Tenant)
	if err != nil {
		l.WithError(err).Error("MIST_CREATED: bad tenant.")
		return
	}
	tctx := tenant.WithContext(ctx, t)
	f := field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).SetInstance(e.Instance).Build()

	body := fieldpkt.NewAffectedAreaCreated(
		e.MistId, e.Body.OwnerId,
		e.Body.OriginX, e.Body.OriginY,
		e.Body.LtX, e.Body.LtY, e.Body.RbX, e.Body.RbY,
		e.Body.Duration, 0,
	)
	_ = _map.NewProcessor(l, tctx).ForSessionsInMap(f, func(s session.Model) error {
		return session.Announce(l)(tctx)(wp)(fieldpkt.AffectedAreaCreatedWriter)(body.Encode(l, tctx))(s)
	})
}

func handleMistDestroyed(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, e mistKafka.Event[mistKafka.DestroyedBody]) {
	t, err := tenant.GetById(e.Tenant)
	if err != nil {
		l.WithError(err).Error("MIST_DESTROYED: bad tenant.")
		return
	}
	tctx := tenant.WithContext(ctx, t)
	f := field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).SetInstance(e.Instance).Build()

	body := fieldpkt.NewAffectedAreaRemoved(e.MistId, 0)
	_ = _map.NewProcessor(l, tctx).ForSessionsInMap(f, func(s session.Model) error {
		return session.Announce(l)(tctx)(wp)(fieldpkt.AffectedAreaRemovedWriter)(body.Encode(l, tctx))(s)
	})
}

func InitConsumers(l logrus.FieldLogger) func(cmf consumer.ConsumerManagerFactory) func(groupId string) {
	// ...registration boilerplate matching the existing monster/reactor consumers
}

func InitHandlers(l logrus.FieldLogger, wp writer.Producer) func(rh handler.RegisterHandler) error {
	// ...handler registration that switches on Type to dispatch CREATED vs DESTROYED
}
```

- [x] **Step 22.4: Wire in atlas-channel main.go**

Mirror existing consumer registration patterns.

- [x] **Step 22.5: Run tests**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./...
```

Expected: PASS.

- [x] **Step 22.6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/consumer/mist/ services/atlas-channel/atlas.com/channel/kafka/message/mist/ services/atlas-channel/atlas.com/channel/main.go
git commit -m "task-036: atlas-channel mist consumer + AffectedArea broadcast"
```

---

### Task 23: Reflect math in `character_attack_common.go`

**Why:** FR-4.3 — replace the two TODOs at lines 144-145 with mirror lookup + bounding-box check + zeroing of monster damage + emit `DAMAGE_REFLECTED`.

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go:67-82, 144-145`
- Test: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common_test.go` (create)

- [x] **Step 23.1: Write failing test**

```go
func TestProcessAttack_MeleeOnReflectingMonster_InsideRange_ReflectsAndZerosDamage(t *testing.T) {
	// Setup: mirror has PHYSICAL reflect on monster M (percent=30, bounds=-100..100).
	// Attacker at (50, 0); monster at (0, 0).
	// Damage entry of 1000.
	// Assert: DAMAGE_REFLECTED event emitted with reflectDamage=300; mp.Damage NOT called for that entry (or called with 0).
}

func TestProcessAttack_MeleeOnReflectingMonster_OutsideRange_DamagesNormally(t *testing.T) {
	// Same setup but attacker at (200, 0) — outside bounds.
	// Damage entry of 1000.
	// Assert: NO DAMAGE_REFLECTED; mp.Damage called normally.
}

func TestProcessAttack_MagicAttack_OnPhysicalReflectMonster_DamagesNormally(t *testing.T) {
	// Magic attack against PHYSICAL-only reflect — no reflect.
}

func TestProcessAttack_MagicAttack_OnMagicalReflectMonster_Reflects(t *testing.T) { /* ... */ }
```

- [x] **Step 23.2: Implement the reflect path**

Edit `character_attack_common.go` lines 67-82 — wrap the damage loop with reflect math:

```go
// Determine attack class once.
attackKind := ""
switch ai.AttackType() {
case packetmodel.AttackTypeMelee, packetmodel.AttackTypeRanged:
    attackKind = monster2.ReflectKindPhysical
case packetmodel.AttackTypeMagic:
    attackKind = monster2.ReflectKindMagical
}

mp := monster.NewProcessor(l, ctx)
mirror := monster.GetStatusMirror()

for _, di := range ai.DamageInfo() {
    damages := di.Damages()
    if len(damages) == 0 {
        continue
    }

    if attackKind != "" {
        info, ok := mirror.GetReflect(s.Tenant(), di.MonsterId(), attackKind)
        if ok {
            // Bounding-box check: attacker (x,y) relative to monster.
            mon, mErr := mp.GetById(di.MonsterId())
            if mErr == nil {
                dx := int16(c.X()) - mon.X()
                dy := int16(c.Y()) - mon.Y()
                if dx >= info.LtX && dx <= info.RbX && dy >= info.LtY && dy <= info.RbY {
                    totalDamage := int32(0)
                    for _, dmg := range damages {
                        totalDamage += int32(dmg)
                    }
                    reflected := totalDamage * info.Percent / 100
                    if reflected > info.MaxDamage {
                        reflected = info.MaxDamage
                    }
                    l.Debugf("reflect: char [%d] hit on monster [%d] reflected %d damage.", s.CharacterId(), di.MonsterId(), reflected)
                    _ = mp.EmitDamageReflected(s.Field(), di.MonsterId(), s.CharacterId(), uint32(reflected))
                    continue // do NOT call mp.Damage for this entry
                }
            }
        }
    }

    if err := mp.Damage(s.Field(), di.MonsterId(), s.CharacterId(), damages, byte(ai.AttackType())); err != nil {
        l.WithError(err).Errorf("Unable to apply damage to monster [%d] from character [%d].", di.MonsterId(), s.CharacterId())
    }
    // (existing monster status apply at lines 75-81 stays inside the same loop iteration)
    if len(se.MonsterStatus()) > 0 {
        ms := make(map[string]int32)
        for k, v := range se.MonsterStatus() {
            ms[k] = int32(v)
        }
        _ = mp.ApplyStatus(s.Field(), di.MonsterId(), s.CharacterId(), uint32(ai.SkillId()), uint32(sk.Level()), ms, uint32(se.Duration()))
    }
}

// Remove the two trailing TODO lines (144-145).
```

> `mp.EmitDamageReflected` is a new producer method on `monster.Processor`. Add it next to `mp.Damage`. The body shape is the existing `statusEventDamageReflectedBody` (api-contracts §3): `{CharacterId, ReflectDamage, MonsterUniqueId}`. Reuse the existing producer entry-point.

- [x] **Step 23.3: Add `monster.Processor.EmitDamageReflected`**

In `services/atlas-channel/atlas.com/channel/monster/processor.go`:

```go
func (p *Processor) EmitDamageReflected(field field.Model, monsterId uint32, attackerId uint32, reflected uint32) error {
    return message.Emit(p.l, p.ctx)(func(buf *message.Buffer) error {
        return buf.Put(/* EVENT_TOPIC_MONSTER_STATUS or correct topic */, damageReflectedProvider(p.t, field, monsterId, attackerId, reflected))
    })
}
```

> Confirm topic / provider shape against the existing `damageReflectedProvider` (search the producer.go for the existing function — atlas-channel may already have it; we just need to call it.)

- [x] **Step 23.4: Run tests**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -v
```

Expected: PASS.

- [x] **Step 23.5: Add stale-mirror regression (risks §1)**

```go
func TestProcessAttack_AfterReflectExpiry_DoesNotReflect(t *testing.T) {
	// Apply reflect with 1ms duration; sleep 10ms; attempt attack.
	// Assert: no DAMAGE_REFLECTED; damage applies normally.
}
```

- [x] **Step 23.6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common_test.go services/atlas-channel/atlas.com/channel/monster/processor.go
git commit -m "task-036: reflect math in character attack handler"
```

---

### Task 24: `STATUS_CANCEL` command + `SourceSkillClass` + dispel guard

**Why:** FR-4.9 + design D7 — atlas-channel populates `SourceSkillClass` on a `STATUS_CANCEL` command body; atlas-monsters refuses cancels of non-reflect statuses while a same-kind reflect is active.

**Plan-phase note:** The explore found no current `STATUS_CANCEL` command channel between atlas-channel and atlas-monsters. **Step 24.0 is to confirm.** If absent, this task adds the channel as well; if present, it only extends the body.

**Files:**
- Confirm/extend: `services/atlas-monsters/atlas.com/monsters/kafka/message/monster/<status_cancel_file>.go` (or create)
- Confirm/create: `services/atlas-channel/atlas.com/channel/kafka/producer/...` (the STATUS_CANCEL producer)
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/processor.go` (or wherever cancel commands are consumed) — add dispel-guard logic
- Test: cancel-handler test file

- [x] **Step 24.0: Confirm whether a `STATUS_CANCEL` command channel exists**

```bash
grep -rn 'STATUS_CANCEL\|StatusCancel\|EnvCommandTopicMonster' services/atlas-monsters/atlas.com/monsters/ services/atlas-channel/atlas.com/channel/
```

If a producer / consumer pair exists, extend the body. If not, add a minimal `COMMAND_TOPIC_MONSTER_STATUS` channel: command body `{UniqueId uint32, StatusName string, SourceCharacterId uint32, SourceSkillId uint32, SourceSkillClass string}`.

- [x] **Step 24.1: Extend the STATUS_CANCEL command body**

Add `SourceSkillClass string \`json:"sourceSkillClass"\`` to the command body. cjson rules: no `omitempty`.

- [x] **Step 24.2: Populate `SourceSkillClass` from atlas-channel**

In whichever atlas-channel code produces `STATUS_CANCEL` (e.g. dispel-skill handler), populate the field from the player skill metadata. The classification mirrors the attack-type mapping in T23: melee/ranged → `PHYSICAL`, magic → `MAGIC`. If no classification is available, leave empty `""` (handler falls through to existing behaviour).

- [x] **Step 24.3: Write failing tests for the dispel guard**

```go
func TestStatusCancel_PhysicalSkill_RejectedWhilePhysicalReflectActive(t *testing.T) {
	// Monster has active WEAPON_COUNTER reflect.
	// Send STATUS_CANCEL with SourceSkillClass="PHYSICAL", target FREEZE.
	// Assert: monster still has FREEZE (cancel rejected); debug log emitted.
}
func TestStatusCancel_MagicSkill_RejectedWhileMagicalReflectActive(t *testing.T) { /* symmetric */ }
func TestStatusCancel_PhysicalSkill_AllowedWhileMagicalReflectActive(t *testing.T) {
	// Monster has MAGIC_COUNTER (magical reflect), but cancel is PHYSICAL — allow.
}
func TestStatusCancel_NoSkillClass_FallsThroughToNormalCancel(t *testing.T) {
	// Backwards-compat — empty SourceSkillClass = normal cancel.
}
func TestStatusCancel_TargetingReflectItself_AllowedRegardlessOfClass(t *testing.T) {
	// Cancelling WEAPON_COUNTER directly — allowed (FR-4.9.1.1).
}
```

- [x] **Step 24.4: Implement the dispel guard in the cancel handler**

```go
func (p *ProcessorImpl) handleStatusCancel(uniqueId uint32, statusName string, sourceSkillClass string) error {
    // FR-4.9.1.1: if cancelling a reflect itself, allow.
    if statusName == "WEAPON_COUNTER" || statusName == "MAGIC_COUNTER" {
        return p.cancelStatus(uniqueId, statusName)
    }

    // FR-4.9.1.2: if monster has an active reflect of the same class, reject.
    if sourceSkillClass != "" {
        m, err := GetMonsterRegistry().GetMonster(p.t, uniqueId)
        if err != nil {
            return err
        }
        for _, se := range m.StatusEffects() {
            if se.IsReflect() && se.ReflectKind() == sourceSkillClass {
                p.l.Debugf("dispel rejected: monster [%d] has active %s reflect.", uniqueId, sourceSkillClass)
                return nil
            }
        }
    }

    return p.cancelStatus(uniqueId, statusName)
}
```

- [x] **Step 24.5: Run tests**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test ./monster/ -run 'TestStatusCancel' -v
```

Expected: PASS — including all parametric pairs.

- [x] **Step 24.6: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/  # all touched files
git add services/atlas-channel/atlas.com/channel/  # producer + classification
git commit -m "task-036: STATUS_CANCEL.SourceSkillClass + dispel guard against active reflect"
```

---

### Task 25: Player-skill venom snapshot DPT (atlas-channel side)

**Why:** Design §3.1 "Per-effect snapshot DPT" — VENOM is player-cast; atlas-channel knows the attacker's `Luck` and `MagicAttack`. Compute `damagePerTick = round(rand(0.1, 0.2) * Luck * MagicAttack)` at apply time and ship it via `Statuses["VENOM"]` on the apply command.

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go` (or wherever monster-status apply is produced — line 75-81 `mp.ApplyStatus`)
- Test: existing test file

- [x] **Step 25.1: Locate the player-skill VENOM apply path**

```bash
grep -rn 'VENOM\|MonsterStatus\|ApplyStatus' services/atlas-channel/atlas.com/channel/ | head -20
```

The current `mp.ApplyStatus` call at `character_attack_common.go:80` already passes the entire `se.MonsterStatus()` map. For VENOM skills, `MonsterStatus()["VENOM"]` likely already carries a default amount — confirm.

- [x] **Step 25.2: Write failing test**

```go
func TestProcessAttack_VenomSkill_SnapshotDPT_FromAttackerStats(t *testing.T) {
	// Set attacker.Luck=120, attacker.MagicAttack=200.
	// Synthesize attack with VENOM in monster status (default amount X).
	// Capture the ApplyStatus call's statuses map.
	// Assert: statuses["VENOM"] is in [round(0.1*120*200), round(0.2*120*200)] = [2400, 4800].
}
```

- [x] **Step 25.3: Inject snapshot DPT before `mp.ApplyStatus`**

```go
if len(se.MonsterStatus()) > 0 {
    ms := make(map[string]int32)
    for k, v := range se.MonsterStatus() {
        ms[k] = int32(v)
    }
    if _, isVenom := ms["VENOM"]; isVenom {
        coef := 0.1 + rand.Float64()*0.1
        dpt := int32(math.Round(coef * float64(c.Luck()) * float64(c.MagicAttack())))
        ms["VENOM"] = dpt
    }
    _ = mp.ApplyStatus(s.Field(), di.MonsterId(), s.CharacterId(), uint32(ai.SkillId()), uint32(sk.Level()), ms, uint32(se.Duration()))
}
```

> Use a seeded `rand` if the codebase already has a shared random source. Confirm `c.Luck()` / `c.MagicAttack()` accessors on the channel-side character.Model.

- [x] **Step 25.4: Run tests**

- [x] **Step 25.5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common_test.go
git commit -m "task-036: snapshot venom DPT from attacker stats at apply time"
```

---

## Phase 4 — Verification & integration

### Task 26: End-to-end smoke verification

**Why:** PRD §10 acceptance criteria — verify across all five services + the two libs. No new code; this is a verification gate.

- [x] **Step 26.1: Build every touched service and lib**

```bash
cd <home>/source/atlas-ms/atlas
for path in libs/atlas-constants libs/atlas-packet services/atlas-monsters/atlas.com/monsters services/atlas-channel/atlas.com/channel services/atlas-buffs/atlas.com/buffs services/atlas-maps/atlas.com/maps; do
  echo "=== building $path ==="
  (cd "$path" && go build ./...) || exit 1
done
```

Expected: every build PASS.

- [x] **Step 26.2: Run every touched test suite**

```bash
for path in libs/atlas-constants libs/atlas-packet services/atlas-monsters/atlas.com/monsters services/atlas-channel/atlas.com/channel services/atlas-buffs/atlas.com/buffs services/atlas-maps/atlas.com/maps; do
  echo "=== testing $path ==="
  (cd "$path" && go test -race ./...) || exit 1
done
```

Expected: every suite PASS.

- [x] **Step 26.3: Verify each PRD §10 acceptance criterion**

Tick off the criteria:
1. **Reflect end-to-end** — covered by T23 unit tests (`TestProcessAttack_MeleeOnReflectingMonster_*`).
2. **Reflect range gate** — covered by T23 (outside-bounds test).
3. **Venom 3-stack** — covered by T4 (`TestAddStatusEffect_VenomOverflow_*`) and T12 (`TestHandleStatusEffectApplied_Venom*`).
4. **Venom expire collapse** — covered by T12 (`TestHandleStatusEffectExpired_VenomLastSlot_*`).
5. **Mist** — covered by T17 (`TestMistTick_*`) and T22 (`TestMistCreated_*`).
6. **Picker un-skip** — covered by T21 (`TestPicker_AreaPoisonAllowed`).
7. **Player poison DoT** — covered by T5 (`TestPoisonTick_*`) and existing producer regression.
8. **Immunity mutual exclusion** — covered by T9 (`TestExecuteStatBuff_*Immune_Cancels*`).
9. **Dispel guard** — covered by T24 (`TestStatusCancel_*`).
10. **cjson** — covered by T1 + T7 round-trip tests.
11. **Test coverage** — confirm by running `go test -cover` per service.
12. **No regressions** — covered by Step 26.2.

- [x] **Step 26.4: Verify Docker builds on every touched service**

For each of the five services that changed, run the project's Docker build (per `CLAUDE.md`: "Always verify Docker builds when changing shared libraries"). Concrete command depends on the project Makefile / docker-compose.

- [x] **Step 26.5: Optional manual smoke (if dev environment available)**

Spawn Pap, GM-cast Weapon Reflect, melee-attack inside range; verify HP behaviour. Spawn Anego mist, walk in/out of zone, observe disease apply + expire + AffectedAreaRemoved.

---

### Task 27: Final docs + audits

- [x] **Step 27.1: Mark plan as complete**

In `docs/tasks/task-036-monster-skill-effects-completion/plan.md`, replace any unchecked `- [x]` boxes with `- [x]` for tasks completed during execution.

- [x] **Step 27.2: Run code-review skill**

```text
/superpowers:requesting-code-review
```

This dispatches `plan-adherence-reviewer`, `backend-guidelines-reviewer`, and `frontend-guidelines-reviewer` (the FE one will report N/A since no atlas-ui changes).

- [x] **Step 27.3: Address audit findings**

Address any FAIL items in `audit.md`. Re-run T26.2 after fixes.

- [x] **Step 27.4: Open PR**

Per CLAUDE.md / project workflow rules. Title: `task-036: monster skill effects completion (reflect, venom, mist, immunity, dispel)`. Reference PRD / design / context / plan in the description.

- [x] **Step 27.5: Commit final docs**

```bash
git add docs/tasks/task-036-monster-skill-effects-completion/plan.md
git commit -m "task-036: mark plan complete after execution"
```

---

## Self-Review Notes (plan author)

**Spec coverage check:** Every PRD FR-4.x has a corresponding task —
- FR-4.1 (reflect apply path) → T6, T7, T8
- FR-4.2 (mirror) → T10, T11
- FR-4.3 (attack handler reflect math) → T23
- FR-4.4 (venom stacking) → T4, T12; per design D3 the encoding does not need slot-key constants
- FR-4.5 (poison replacement) → unchanged behaviour; existing `RemoveStatusEffectByType` at `builder.go:170-179` already handles re-apply by removing prior POISON. Confirmed by re-reading code (no new task — but we should add a regression test inside T1 or T6's test file)
- FR-4.6 (mist) → T13–T18, T20, T22
- FR-4.7 (PoisonTick) → T5
- FR-4.8 (immunity exclusion) → T9
- FR-4.9 (dispel guard) → T24
- FR-4.10 (cjson) → T1, T7

**Placeholder scan:** Reviewed — no `TBD`/`TODO`/`fill in details`. Tasks T15-T19 have plan-phase verification steps (e.g. "confirm character REST client") to be resolved during execution; these are explicit verifications with concrete fallbacks, not placeholders.

**Type consistency:** `ReflectKindPhysical = "PHYSICAL"` and `ReflectKindMagical = "MAGICAL"` — used consistently across T3, T6, T8, T10, T23, T24. Status names (`WEAPON_COUNTER`, `MAGIC_COUNTER`, `WEAPON_ATTACK_IMMUNE`, `MAGIC_ATTACK_IMMUNE`, `VENOM`, `POISON`) are pinned via the libs/atlas-constants references and used as string literals.

**FR-4.5 regression test gap:** Add a small regression test in T6 step (or T8) explicitly:

```go
func TestApplyStatusEffect_PoisonReplacement_NewSkillLevelTakesEffectImmediately(t *testing.T) {
    // Apply POISON skill level 5 → effect.SourceSkillLevel == 5.
    // Apply POISON skill level 10 → builder.RemoveStatusEffectByType("POISON") evicts old; new effect.SourceSkillLevel == 10.
    // Tick once: damage formula uses skillLevel=10 (status_task.go:104-111).
}
```

Place this test in `processor_test.go` and add a checkbox to T8. The implementation is unchanged; this is a regression-only test.
