# Mob Spawn Move-Action (Stance) Byte Implementation Plan

> ## ⚠️ SUPERSEDED — see design.md ROOT-CAUSE CORRECTION
>
> This plan implements the **stance-byte** approach, which live tracing proved was
> not the root cause. The real bug was `MonsterMovementHandle` missing its `types`
> config so monster moves never decoded (fh+stance stuck at seed 0 → fall-through +
> v79 freeze). What shipped: add `types` to Monster/Pet/Summon handlers across all
> templates lacking it, fix the fold pointer-type bug, keep `ControlOnEnter`; the
> stance guards and diagnostic tracing were backed out. This plan is kept as the
> investigation record only — do not execute it.

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Emit a correct, pre-resolved, fly-aware mob move-action (stance) byte so the v83 client never resolves the stance during `CMob::Init` (the `0`/`1` sentinel → `CMob::OnResolveMoveAction` → `m_pvc` null-deref crash), and so flying/swimming mobs animate in their correct idle pose on spawn.

**Architecture:** One shared pure helper (`libs/atlas-constants/monster.IdleMoveAction`) computes the idle byte from `(isFly, fixedStance)`. `atlas-monsters` `Create` uses it to seed a fresh spawn's registry stance (replacing a hardcoded `5`). `atlas-channel` extends its `monster/information` client to carry the fly fields (reusing the existing TTL cache) and adds a narrow emit-boundary guard at the two `NewMonster(...)` sites that rewrites only a `0`/`1` input stance to the fly-aware idle value, passing `>= 2` through verbatim. No packet-layout, DB, Kafka, or `atlas-data` change.

**Tech Stack:** Go 1.24 microservices; `libs/atlas-constants` shared lib; JSON:API REST read models via api2go; immutable models (private fields + getters + Builder); tenant-scoped read-through TTL cache; standard `testing` table-driven tests.

## Global Constraints

- **Crash invariant:** No spawn or control packet ever carries a move-action byte of `0` or `1` for a spawned mob — every emitted stance satisfies `(byte & ^byte(1)) != 0`. (PRD §2, §10)
- **Encoding (client-verified, v83):** `actionIndex = isFly ? 6 : 2`; `facingBit = (fixedStance != 0) ? (fixedStance & 1) : 0` (0 = right, 1 = left); `moveAction = byte(actionIndex<<1) | facingBit`. Truth table: ground `4`/`5`, fly `12`/`13`. (design §2)
- **`isFly = Flying || Swimming`** — swim is fly-in-water, same client branch → `actionIndex 6`. (PRD FR-1.2)
- **`fixedStance` contributes ONLY the facing bit, never the action index.** A `noFlip` fly mob emits `12/13`, not `4/5`. (PRD FR-2.2)
- **Guard is narrow:** it rewrites ONLY `0`/`1`. Any stance `>= 2` (including legit mid-action stances and the fresh `4/5`/`12/13`) is emitted verbatim. (PRD FR-4.3)
- **No packet-layout change:** same fields, same order; only the `moveAction` byte's *value* changes. `libs/atlas-packet` is untouched. (PRD §4 non-goals, §10)
- **Out of scope:** live movement-broadcast relay path (FR-5.1); the zero-value `NewMonster(f, uniqueId, 0, 0, ...)` diff/placeholder at `atlas-monsters .../processor.go:1511` (FR-5.2).
- **NFR-2 (perf):** no per-mob synchronous `atlas-data` fetch on the channel spawn/control hot path — reuse the existing `monster/information` TTL cache.
- **NFR-3 (tenancy):** all template fetches stay tenant-scoped via existing context plumbing; no cross-tenant cache bleed.
- **Verification (CLAUDE.md):** `go test -race ./...`, `go vet ./...`, `go build ./...` clean in every changed module (`libs/atlas-constants`, `atlas-monsters`, `atlas-channel`); `docker buildx bake atlas-monsters` + `atlas-channel` from the worktree root; `tools/lint.sh --check`, `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh` clean from the repo root. Use the project Builder pattern for test setup — no `*_testhelpers.go`.

---

## File Structure

**New files:**
- `libs/atlas-constants/monster/stance.go` — the `IdleMoveAction` pure helper + encoding constants.
- `libs/atlas-constants/monster/stance_test.go` — §10 acceptance vectors + crash-invariant sweep.
- `services/atlas-channel/atlas.com/channel/socket/writer/monster_spawn_test.go` — guard table tests (spawn/control share the helper, one test file covers both).

**Modified files:**
- `services/atlas-monsters/atlas.com/monsters/monster/information/rest.go` — DTO `+Flying,+Swimming`; `Extract` maps `flying`/`swimming`/`fixed_stance`.
- `services/atlas-monsters/atlas.com/monsters/monster/information/model.go` — `+flying,+swimming,+fixedStance` fields + `IsFly()`, `FixedStance()` getters.
- `services/atlas-monsters/atlas.com/monsters/monster/information/builder.go` — `SetFlying`/`SetSwimming`/`SetFixedStance` setters + wire through `Build()`.
- `services/atlas-monsters/atlas.com/monsters/monster/information/rest_test.go` — `Extract` field-mapping test.
- `services/atlas-monsters/atlas.com/monsters/monster/processor.go` — `Create` (line ~192/198): route info lookup through the `testInformationLookup` seam; replace literal `5` with `monster.IdleMoveAction(ma.IsFly(), ma.FixedStance())`.
- `services/atlas-monsters/atlas.com/monsters/monster/processor_test.go` — deterministic Create-level stance test.
- `services/atlas-channel/atlas.com/channel/monster/information/rest.go` — DTO `+Flying,+Swimming,+FixedStance`; `Extract` maps them.
- `services/atlas-channel/atlas.com/channel/monster/information/model.go` — `+flying,+swimming,+fixedStance` fields + `IsFly()`, `FixedStance()` getters.
- `services/atlas-channel/atlas.com/channel/monster/information/builder.go` — `SetFlying`/`SetSwimming`/`SetFixedStance` setters + wire through `Build()`.
- `services/atlas-channel/atlas.com/channel/monster/information/rest_test.go` — `Extract` field-mapping test.
- `services/atlas-channel/atlas.com/channel/socket/writer/monster_spawn.go` — `resolveSpawnStance` helper + call before `NewMonster`.
- `services/atlas-channel/atlas.com/channel/socket/writer/monster_control.go` — call `resolveSpawnStance` before `NewMonster`.

---

## Task 1: Re-verify the client encoding (NFR-1 grounding)

Before any constant is trusted, re-verify the four facts the helper encodes against the v83 client. This is a **read-only** IDA/evidence step (PRD NFR-1, design §9). If a fact is contradicted, the helper's constants in Task 2 change — the architecture does not.

**Files:**
- Create: `docs/tasks/task-179-mob-spawn-stance-byte/grounding.md` (evidence note)

**Interfaces:**
- Produces: confirmed constants for Task 2 — `idleActionIndexGround`, `idleActionIndexFly`, the ground/fly byte truth table, and the sentinel definition. If IDA is unavailable, the task records the PRD's cited addresses as the authority and flags the constants as "cited, not live-re-verified."

- [ ] **Step 1: Select the v83 IDB and confirm the instance**

Use the IDA-MCP tooling. Per project memory, always `list_instances` first and match the binary NAME (`MapleStory_dump.exe`, v83) — the active port is the tool-routing target, not GUI focus. Corroborate with a version-distinguishing opcode if unsure.

Run (conceptually): `mcp__ida-pro__idb_list` / `list_instances`, then `select_instance(<v83 port>)`.

- [ ] **Step 2: Decompile `CMob::GetFineAction` and its callees**

Read `CMob::GetFineAction @0x671999` → `sub_671AFF` (`GetFineMoveDirAction`) → `sub_664D42`. Confirm the two-way idle branch:
`v8 = (moveAbility != 3) ? 2 : 6; return (a2 & 1) | (2*v8)` — i.e. idle `actionIndex` is `2` for ground, `6` for fly; the emitted byte is `(actionIndex << 1) | facingBit`.

Use `mcp__ida-pro__decompile` with the function address. Quote the actual decompiled lines in `grounding.md` before drawing the conclusion (CLAUDE.md grounding rule).

- [ ] **Step 3: Confirm the sentinel crash path**

Confirm `CMob::Init` routes into `CMob::OnResolveMoveAction` when the move-action byte satisfies `(byte & ~1) == 0` (i.e. `0` or `1`), and that `OnResolveMoveAction` dereferences `m_pvc`. Quote the branch condition.

- [ ] **Step 4: Write the evidence note**

Write `docs/tasks/task-179-mob-spawn-stance-byte/grounding.md` recording: the decompiled branch (quoted), the resulting truth table (ground `4/5`, fly `12/13`), the sentinel definition (`0`/`1`), and each source address. If any value differs from the design's, state the corrected value explicitly and note that Task 2's constants must follow it.

- [ ] **Step 5: Commit**

```bash
git add docs/tasks/task-179-mob-spawn-stance-byte/grounding.md
git commit -m "docs(task-179): re-verify mob move-action encoding against v83 client"
```

---

## Task 2: Shared idle-stance helper (`libs/atlas-constants/monster`)

**Files:**
- Create: `libs/atlas-constants/monster/stance.go`
- Test: `libs/atlas-constants/monster/stance_test.go`

**Interfaces:**
- Consumes: confirmed constants from Task 1 (`grounding.md`).
- Produces: `func IdleMoveAction(isFly bool, fixedStance uint32) byte` and exported `const ( FacingRight byte = 0; FacingLeft byte = 1 )` in package `monster` (import path `github.com/Chronicle20/atlas/libs/atlas-constants/monster`). Consumed by Tasks 4 and 6.

- [ ] **Step 1: Write the failing test**

Create `libs/atlas-constants/monster/stance_test.go`:

```go
package monster

import "testing"

func TestIdleMoveAction(t *testing.T) {
	tests := []struct {
		name        string
		isFly       bool
		fixedStance uint32
		want        byte
	}{
		{"ground default right", false, 0, 4},
		{"ground noFlip 4 (right)", false, 4, 4},
		{"ground noFlip 5 (left)", false, 5, 5},
		{"fly default right", true, 0, 12},
		{"fly noFlip 4 (right)", true, 4, 12},
		{"fly noFlip 5 (left)", true, 5, 13},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IdleMoveAction(tt.isFly, tt.fixedStance); got != tt.want {
				t.Fatalf("IdleMoveAction(%t, %d) = %d, want %d", tt.isFly, tt.fixedStance, got, tt.want)
			}
		})
	}
}

// TestIdleMoveActionNeverSentinel asserts the crash invariant holds by
// construction for every supported input: the emitted byte is never 0 or 1.
func TestIdleMoveActionNeverSentinel(t *testing.T) {
	for _, isFly := range []bool{false, true} {
		for _, fixed := range []uint32{0, 4, 5} {
			got := IdleMoveAction(isFly, fixed)
			if got&^byte(1) == 0 {
				t.Fatalf("IdleMoveAction(%t, %d) = %d is a 0/1 sentinel", isFly, fixed, got)
			}
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-constants && go test ./monster/ -run TestIdleMoveAction -v`
Expected: FAIL — `undefined: IdleMoveAction`.

- [ ] **Step 3: Write minimal implementation**

Create `libs/atlas-constants/monster/stance.go`:

```go
package monster

// Move-action (stance) idle encoding, verified against the v83 client
// CMob::GetFineAction @0x671999 -> sub_671AFF -> sub_664D42:
//   v8 = (moveAbility != 3) ? 2 : 6; return (a2 & 1) | (2*v8)
// Idle actionIndex is 2 for ground move-ability, 6 for fly move-ability;
// the emitted byte is (actionIndex << 1) | facingBit. See
// docs/tasks/task-179-mob-spawn-stance-byte/grounding.md.
const (
	idleActionIndexGround = 2
	idleActionIndexFly    = 6

	// FacingRight/FacingLeft are the low (facing) bit of the move-action byte.
	FacingRight byte = 0
	FacingLeft  byte = 1
)

// IdleMoveAction returns the pre-resolved idle move-action byte the client
// would otherwise compute in CMob::OnResolveMoveAction. Emitting it means the
// server never sends the 0/1 sentinel that crashes the spawn/control
// CMob::Init path (a2 & ~1 == 0 -> OnResolveMoveAction -> m_pvc null-deref).
//
// isFly is Flying || Swimming (swim = fly-in-water, same client branch).
// fixedStance is atlas-data's getFixedStance output (4/5 for noFlip mobs,
// else 0); it contributes only the facing bit, never the action index — so a
// noFlip fly mob emits 12/13, not the ground 4/5 that fixedStance alone implies.
func IdleMoveAction(isFly bool, fixedStance uint32) byte {
	actionIndex := byte(idleActionIndexGround)
	if isFly {
		actionIndex = idleActionIndexFly
	}
	facingBit := FacingRight
	if fixedStance != 0 {
		facingBit = byte(fixedStance & 1)
	}
	return actionIndex<<1 | facingBit
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd libs/atlas-constants && go test ./monster/ -v`
Expected: PASS (both `TestIdleMoveAction` and `TestIdleMoveActionNeverSentinel`).

- [ ] **Step 5: Verify vet clean**

Run: `cd libs/atlas-constants && go vet ./monster/`
Expected: no output (clean).

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-constants/monster/stance.go libs/atlas-constants/monster/stance_test.go
git commit -m "feat(task-179): shared IdleMoveAction stance helper in atlas-constants"
```

---

## Task 3: Carry fly fields on the `atlas-monsters` information model

Extend the `atlas-monsters` `monster/information` read model to carry `flying`, `swimming`, and `fixed_stance` (the DTO already has `FixedStance` but `Extract` drops it; the fly flags are absent entirely).

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/information/model.go`
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/information/builder.go`
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/information/rest.go`
- Test: `services/atlas-monsters/atlas.com/monsters/monster/information/rest_test.go`

**Interfaces:**
- Consumes: nothing from prior tasks.
- Produces: on `information.Model` — `IsFly() bool` (returns `flying || swimming`) and `FixedStance() uint32`. On `information.ModelBuilder` — `SetFlying(bool)`, `SetSwimming(bool)`, `SetFixedStance(uint32)`. Consumed by Task 4.

- [ ] **Step 1: Write the failing Extract test**

Add to `services/atlas-monsters/atlas.com/monsters/monster/information/rest_test.go` (create the file if it does not exist with `package information` and `import "testing"`):

```go
func TestExtractCarriesFlyFields(t *testing.T) {
	cases := []struct {
		name         string
		rm           RestModel
		wantIsFly    bool
		wantFixed    uint32
	}{
		{"ground", RestModel{Flying: false, Swimming: false, FixedStance: 0}, false, 0},
		{"flying", RestModel{Flying: true, Swimming: false, FixedStance: 0}, true, 0},
		{"swimming", RestModel{Flying: false, Swimming: true, FixedStance: 0}, true, 0},
		{"ground noFlip", RestModel{Flying: false, Swimming: false, FixedStance: 5}, false, 5},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m, err := Extract(c.rm)
			if err != nil {
				t.Fatalf("Extract error: %v", err)
			}
			if m.IsFly() != c.wantIsFly {
				t.Errorf("IsFly() = %t, want %t", m.IsFly(), c.wantIsFly)
			}
			if m.FixedStance() != c.wantFixed {
				t.Errorf("FixedStance() = %d, want %d", m.FixedStance(), c.wantFixed)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-monsters/atlas.com/monsters && go test ./monster/information/ -run TestExtractCarriesFlyFields -v`
Expected: FAIL — `RestModel` has no `Flying`/`Swimming` field and `Model` has no `IsFly`/`FixedStance` method.

- [ ] **Step 3: Add fields + getters to the model**

In `services/atlas-monsters/atlas.com/monsters/monster/information/model.go`, add to the `Model` struct (after `mpRecovery uint32`):

```go
	flying      bool
	swimming    bool
	fixedStance uint32
```

Add getters (after `MpRecovery()`):

```go
// IsFly reports whether the mob uses the client's fly idle branch
// (actionIndex 6). Swim mobs are fly-in-water and share the branch.
func (m Model) IsFly() bool {
	return m.flying || m.swimming
}

// FixedStance is atlas-data's getFixedStance output: 4/5 for noFlip mobs
// (fixed facing), else 0. Contributes only the move-action facing bit.
func (m Model) FixedStance() uint32 {
	return m.fixedStance
}
```

- [ ] **Step 4: Add builder setters**

In `services/atlas-monsters/atlas.com/monsters/monster/information/builder.go`, add the fields to `ModelBuilder`:

```go
	flying      bool
	swimming    bool
	fixedStance uint32
```

Add setters (before `Build()`):

```go
// SetFlying sets the flying flag on the builder.
func (b *ModelBuilder) SetFlying(v bool) *ModelBuilder {
	b.flying = v
	return b
}

// SetSwimming sets the swimming flag on the builder.
func (b *ModelBuilder) SetSwimming(v bool) *ModelBuilder {
	b.swimming = v
	return b
}

// SetFixedStance sets the fixed-stance value on the builder.
func (b *ModelBuilder) SetFixedStance(v uint32) *ModelBuilder {
	b.fixedStance = v
	return b
}
```

Wire them through the `Build()` return literal (add to the `Model{...}` fields):

```go
		flying:      b.flying,
		swimming:    b.swimming,
		fixedStance: b.fixedStance,
```

- [ ] **Step 5: Add DTO fields + map through Extract**

In `services/atlas-monsters/atlas.com/monsters/monster/information/rest.go`, add to `RestModel` (near `FixedStance uint32`; `FixedStance` already exists — add the two flags):

```go
	Flying   bool `json:"flying"`
	Swimming bool `json:"swimming"`
```

In `Extract`, add to the returned `Model{...}` literal:

```go
		flying:      rm.Flying,
		swimming:    rm.Swimming,
		fixedStance: rm.FixedStance,
```

- [ ] **Step 6: Run test to verify it passes**

Run: `cd services/atlas-monsters/atlas.com/monsters && go test ./monster/information/ -v`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/information/model.go \
        services/atlas-monsters/atlas.com/monsters/monster/information/builder.go \
        services/atlas-monsters/atlas.com/monsters/monster/information/rest.go \
        services/atlas-monsters/atlas.com/monsters/monster/information/rest_test.go
git commit -m "feat(task-179): carry flying/swimming/fixed_stance on atlas-monsters info model"
```

---

## Task 4: Fresh-spawn origin — `atlas-monsters` `Create` idle stance

Replace the hardcoded `5` at `Create` with the fly-aware idle stance, and route `Create`'s info lookup through the existing `testInformationLookup` seam so the fresh-spawn stance is deterministically unit-testable (the seam is the established pattern for the other `information.GetById` call sites in this file).

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/processor.go` (`Create`, lines ~192 and ~198)
- Test: `services/atlas-monsters/atlas.com/monsters/monster/processor_test.go`

**Interfaces:**
- Consumes: `monster.IdleMoveAction(isFly bool, fixedStance uint32) byte` from Task 2; `information.Model.IsFly()` / `.FixedStance()` and `information.NewModelBuilder().SetFlying/SetSwimming/SetFixedStance` from Task 3; `testInformationLookup` seam and `information.Model.Stance()`/registry `CreateMonster(..., stance byte, ...)` (existing).
- Produces: a fresh mob's registry `stance` is `IdleMoveAction(isFly, fixedStance)` — `12`/`13` for fly/swim, `4`/`5` for ground. Consumed downstream by Kafka → `atlas-channel` (no code interface).

- [ ] **Step 1: Write the failing test**

Add to `services/atlas-monsters/atlas.com/monsters/monster/processor_test.go`. This uses the `testInformationLookup` seam (see the existing usages at lines ~1500) and the registry to assert the created monster's `Stance()`. Add the import alias for the shared lib at the top of the file if not present: `mobconst "github.com/Chronicle20/atlas/libs/atlas-constants/monster"`.

```go
func TestCreateFreshSpawnStance(t *testing.T) {
	cases := []struct {
		name      string
		flying    bool
		swimming  bool
		fixed     uint32
		wantStance byte
	}{
		{"ground default", false, false, 0, 4},
		{"ground noFlip left", false, false, 5, 5},
		{"flying default", true, false, 0, 12},
		{"swimming default", false, true, 0, 12},
		{"flying noFlip left", true, false, 5, 13},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			prevHook := testInformationLookup
			testInformationLookup = func(_ uint32) (information.Model, error) {
				return information.NewModelBuilder().
					SetFlying(c.flying).
					SetSwimming(c.swimming).
					SetFixedStance(c.fixed).
					Build(), nil
			}
			defer func() { testInformationLookup = prevHook }()

			ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
			tctx := tenant.WithContext(context.Background(), ten)
			p := &ProcessorImpl{l: newPickerLogger(), ctx: tctx, t: ten}

			m, err := p.Create(testField(), RestModel{MonsterId: 9000000, X: 0, Y: 0})
			if err != nil {
				t.Fatalf("Create error: %v", err)
			}
			if m.Stance() != c.wantStance {
				t.Fatalf("Stance() = %d, want %d", m.Stance(), c.wantStance)
			}
			// Crash invariant: never a 0/1 sentinel.
			if m.Stance()&^byte(1) == 0 {
				t.Fatalf("Stance() = %d is a 0/1 sentinel", m.Stance())
			}
		})
	}
}
```

> Note: match the exact context/tenant construction and logger helper (`newPickerLogger`, `tenant.Create`, `tenant.WithContext`) used by neighbouring tests in this file — copy their imports (`context`, `github.com/google/uuid`, `tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"`). If `ProcessorImpl` requires additional non-nil fields to run `Create` without the picker/emit path, set them as the existing `TestSpawnPickerGuardOnAggro`-style tests do (the spawn picker only fires when `ControllerHasAggro()`, which is false for a fresh mob, so `emit`/`inFieldFn` are not exercised here — but set them to no-op funcs if a nil-deref surfaces).

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-monsters/atlas.com/monsters && go test ./monster/ -run TestCreateFreshSpawnStance -v`
Expected: FAIL — the created stance is `5` for every case (hardcoded), and `Create` ignores `testInformationLookup`, so fly cases get `5` not `12`.

- [ ] **Step 3: Route `Create`'s lookup through the seam and compute the stance**

In `services/atlas-monsters/atlas.com/monsters/monster/processor.go`, in `Create`, replace the current lookup (line ~192):

```go
	ma, err := information.NewProcessor(p.l, p.ctx).GetById(input.MonsterId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to retrieve information necessary to create monster [%d].", input.MonsterId)
		return Model{}, err
	}
```

with the seam-aware form (mirrors the existing `testInformationLookup` sites at lines ~803, ~1181, ~1481):

```go
	var ma information.Model
	var err error
	if testInformationLookup != nil {
		ma, err = testInformationLookup(input.MonsterId)
	} else {
		ma, err = information.NewProcessor(p.l, p.ctx).GetById(input.MonsterId)
	}
	if err != nil {
		p.l.WithError(err).Errorf("Unable to retrieve information necessary to create monster [%d].", input.MonsterId)
		return Model{}, err
	}
```

Then replace the hardcoded `5` in the `CreateMonster(...)` call (line ~198):

```go
	m := GetMonsterRegistry().CreateMonster(p.ctx, p.t, f, input.MonsterId, input.X, input.Y, input.Fh, mobconst.IdleMoveAction(ma.IsFly(), ma.FixedStance()), input.Team, ma.Hp(), ma.Mp())
```

Add the import to the file's import block:

```go
	mobconst "github.com/Chronicle20/atlas/libs/atlas-constants/monster"
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-monsters/atlas.com/monsters && go test ./monster/ -run TestCreateFreshSpawnStance -v`
Expected: PASS (all five cases).

- [ ] **Step 5: Run the full monster package test to confirm no regression**

Run: `cd services/atlas-monsters/atlas.com/monsters && go test -race ./monster/...`
Expected: PASS. (The existing `TestSpawnPickerGuard*` tests that previously relied on `Create` hitting the network will now use whatever `testInformationLookup` they set, or nil → real lookup as before; confirm none regress. If a pre-existing test set `testInformationLookup` and asserted nothing about stance, it is unaffected.)

- [ ] **Step 6: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/processor.go \
        services/atlas-monsters/atlas.com/monsters/monster/processor_test.go
git commit -m "feat(task-179): fresh-spawn fly-aware idle stance in atlas-monsters Create"
```

---

## Task 5: Carry fly fields on the `atlas-channel` information model

Mirror Task 3 on the channel side. The channel `monster/information` model currently carries only `monsterId` + `attacks`; add the three fly fields. The new fields ride inside the already-cached `Model` (existing TTL cache in `processor.go`/`cache.go`) — no new cache.

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/monster/information/model.go`
- Modify: `services/atlas-channel/atlas.com/channel/monster/information/builder.go`
- Modify: `services/atlas-channel/atlas.com/channel/monster/information/rest.go`
- Test: `services/atlas-channel/atlas.com/channel/monster/information/rest_test.go`

**Interfaces:**
- Consumes: nothing from prior tasks.
- Produces: on channel `information.Model` — `IsFly() bool` and `FixedStance() uint32`. On channel `information.ModelBuilder` — `SetFlying(bool)`, `SetSwimming(bool)`, `SetFixedStance(uint32)`. Consumed by Task 6.

- [ ] **Step 1: Write the failing Extract test**

Add to `services/atlas-channel/atlas.com/channel/monster/information/rest_test.go`:

```go
func TestExtractCarriesFlyFields(t *testing.T) {
	cases := []struct {
		name      string
		rm        RestModel
		wantIsFly bool
		wantFixed uint32
	}{
		{"ground", RestModel{Id: "100100", Flying: false, Swimming: false, FixedStance: 0}, false, 0},
		{"flying", RestModel{Id: "2300100", Flying: true, Swimming: false, FixedStance: 0}, true, 0},
		{"swimming", RestModel{Id: "7130020", Flying: false, Swimming: true, FixedStance: 0}, true, 0},
		{"ground noFlip", RestModel{Id: "100100", Flying: false, Swimming: false, FixedStance: 5}, false, 5},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m, err := Extract(c.rm)
			if err != nil {
				t.Fatalf("Extract error: %v", err)
			}
			if m.IsFly() != c.wantIsFly {
				t.Errorf("IsFly() = %t, want %t", m.IsFly(), c.wantIsFly)
			}
			if m.FixedStance() != c.wantFixed {
				t.Errorf("FixedStance() = %d, want %d", m.FixedStance(), c.wantFixed)
			}
		})
	}
}
```

> If `rest_test.go` does not exist, create it with `package information` and `import "testing"`.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./monster/information/ -run TestExtractCarriesFlyFields -v`
Expected: FAIL — no `Flying`/`Swimming`/`FixedStance` on `RestModel`, no `IsFly`/`FixedStance` on `Model`.

- [ ] **Step 3: Add fields + getters to the model**

In `services/atlas-channel/atlas.com/channel/monster/information/model.go`, add to `Model`:

```go
	flying      bool
	swimming    bool
	fixedStance uint32
```

Add getters:

```go
// IsFly reports whether the mob uses the client's fly idle branch
// (actionIndex 6). Swim mobs are fly-in-water and share the branch.
func (m Model) IsFly() bool {
	return m.flying || m.swimming
}

// FixedStance is atlas-data's getFixedStance output (4/5 for noFlip mobs,
// else 0); contributes only the move-action facing bit.
func (m Model) FixedStance() uint32 {
	return m.fixedStance
}
```

- [ ] **Step 4: Add builder setters**

In `services/atlas-channel/atlas.com/channel/monster/information/builder.go`, add to `ModelBuilder`:

```go
	flying      bool
	swimming    bool
	fixedStance uint32
```

Add setters before `Build()`:

```go
func (b *ModelBuilder) SetFlying(v bool) *ModelBuilder {
	b.flying = v
	return b
}

func (b *ModelBuilder) SetSwimming(v bool) *ModelBuilder {
	b.swimming = v
	return b
}

func (b *ModelBuilder) SetFixedStance(v uint32) *ModelBuilder {
	b.fixedStance = v
	return b
}
```

Wire through the `Build()` `Model{...}` literal:

```go
		flying:      b.flying,
		swimming:    b.swimming,
		fixedStance: b.fixedStance,
```

- [ ] **Step 5: Add DTO fields + map through Extract**

In `services/atlas-channel/atlas.com/channel/monster/information/rest.go`, add to `RestModel`:

```go
	Flying      bool   `json:"flying"`
	Swimming    bool   `json:"swimming"`
	FixedStance uint32 `json:"fixed_stance"`
```

In `Extract`, add to the returned `Model{...}` literal:

```go
		flying:      rm.Flying,
		swimming:    rm.Swimming,
		fixedStance: rm.FixedStance,
```

- [ ] **Step 6: Run test to verify it passes**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./monster/information/ -v`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/monster/information/model.go \
        services/atlas-channel/atlas.com/channel/monster/information/builder.go \
        services/atlas-channel/atlas.com/channel/monster/information/rest.go \
        services/atlas-channel/atlas.com/channel/monster/information/rest_test.go
git commit -m "feat(task-179): carry flying/swimming/fixed_stance on atlas-channel info model"
```

---

## Task 6: Emit-boundary guard — `resolveSpawnStance`

Add the narrow guard that rewrites only a `0`/`1` input stance to the mob's fly-aware idle stance, and unit-test it with a mocked information processor. The guard lives in package `writer` next to its callers (it needs `l`/`ctx` + the `information` processor — service plumbing, not lib-pure).

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/writer/monster_spawn.go` (add the `resolveSpawnStance` helper)
- Test: `services/atlas-channel/atlas.com/channel/socket/writer/monster_spawn_test.go` (new)

**Interfaces:**
- Consumes: `monster.IdleMoveAction` (Task 2); channel `information.Processor.GetById`, `information.Model.IsFly()`/`.FixedStance()`, `information/mock.ProcessorMock` (Task 5 + existing mock).
- Produces: `func resolveSpawnStance(l logrus.FieldLogger, ctx context.Context, stance byte, monsterId uint32) byte` in package `writer`. To make it testable without the concrete `NewProcessor`, factor the processor construction behind a package-level seam `var newInformationProcessor = information.NewProcessor` so the test can substitute the mock. Consumed by Task 7.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/socket/writer/monster_spawn_test.go`:

```go
package writer

import (
	"context"
	"errors"
	"testing"

	"atlas-channel/monster/information"
	infomock "atlas-channel/monster/information/mock"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func TestResolveSpawnStance(t *testing.T) {
	cases := []struct {
		name     string
		stance   byte
		flying   bool
		swimming bool
		fixed    uint32
		want     byte
	}{
		// Sentinels get rewritten to the fly-aware idle stance.
		{"ground sentinel 0 -> 4", 0, false, false, 0, 4},
		{"ground sentinel 1 -> 4", 1, false, false, 0, 4},
		{"ground noFlip sentinel 0 -> 5", 0, false, false, 5, 5},
		{"fly sentinel 0 -> 12", 0, true, false, 0, 12},
		{"fly sentinel 1 -> 12", 1, true, false, 0, 12},
		{"swim sentinel 0 -> 12", 0, false, true, 0, 12},
		{"fly noFlip sentinel 0 -> 13", 0, true, false, 5, 13},
		// >= 2 passes through verbatim regardless of fly class.
		{"pass-through 4", 4, true, false, 0, 4},
		{"pass-through 5", 5, false, false, 0, 5},
		{"pass-through 12", 12, false, false, 0, 12},
		{"pass-through 99", 99, true, false, 5, 99},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			prev := newInformationProcessor
			newInformationProcessor = func(_ logrus.FieldLogger, _ context.Context) information.Processor {
				return &infomock.ProcessorMock{
					GetByIdFunc: func(_ uint32) (information.Model, error) {
						return information.NewModelBuilder().
							SetFlying(c.flying).
							SetSwimming(c.swimming).
							SetFixedStance(c.fixed).
							Build(), nil
					},
				}
			}
			defer func() { newInformationProcessor = prev }()

			ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
			ctx := tenant.WithContext(context.Background(), ten)
			got := resolveSpawnStance(logrus.New(), ctx, c.stance, 9000000)
			if got != c.want {
				t.Fatalf("resolveSpawnStance(%d) = %d, want %d", c.stance, got, c.want)
			}
			if got&^byte(1) == 0 {
				t.Fatalf("resolveSpawnStance(%d) = %d is a 0/1 sentinel", c.stance, got)
			}
		})
	}
}

func TestResolveSpawnStanceFailSafe(t *testing.T) {
	prev := newInformationProcessor
	newInformationProcessor = func(_ logrus.FieldLogger, _ context.Context) information.Processor {
		return &infomock.ProcessorMock{
			GetByIdFunc: func(_ uint32) (information.Model, error) {
				return information.Model{}, errors.New("boom")
			},
		}
	}
	defer func() { newInformationProcessor = prev }()

	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), ten)
	// On lookup error, a 0/1 sentinel must never reach the wire: floor to
	// ground idle right (4). A >= 2 stance is returned verbatim without lookup.
	if got := resolveSpawnStance(logrus.New(), ctx, 0, 1); got != 4 {
		t.Fatalf("fail-safe sentinel: got %d, want 4", got)
	}
	if got := resolveSpawnStance(logrus.New(), ctx, 7, 1); got != 7 {
		t.Fatalf("pass-through must not consult lookup: got %d, want 7", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/writer/ -run TestResolveSpawnStance -v`
Expected: FAIL — `undefined: newInformationProcessor` and `undefined: resolveSpawnStance`.

- [ ] **Step 3: Implement the seam + guard**

In `services/atlas-channel/atlas.com/channel/socket/writer/monster_spawn.go`, add imports for `"atlas-channel/monster/information"` and `mobconst "github.com/Chronicle20/atlas/libs/atlas-constants/monster"`, then add:

```go
// newInformationProcessor is a package-level seam over information.NewProcessor
// so the stance guard can be unit-tested with a mock processor.
var newInformationProcessor = information.NewProcessor

// resolveSpawnStance rewrites the 0/1 idle sentinel to the mob's fly-aware idle
// stance so the client never resolves it during CMob::Init (the null-deref
// crash path). Any stance >= 2 is emitted verbatim (FR-4.3). The template
// lookup is served from the information client's existing TTL cache (NFR-2).
func resolveSpawnStance(l logrus.FieldLogger, ctx context.Context, stance byte, monsterId uint32) byte {
	if stance&^byte(1) != 0 { // stance >= 2: not a sentinel, emit verbatim
		return stance
	}
	ma, err := newInformationProcessor(l, ctx).GetById(monsterId)
	if err != nil {
		// Fail safe: never emit the crashing sentinel. Ground idle right (4)
		// is the conservative floor.
		l.WithError(err).Debugf("stance guard: info lookup failed for monster [%d]; flooring %d->4", monsterId, stance)
		return mobconst.IdleMoveAction(false, 0)
	}
	resolved := mobconst.IdleMoveAction(ma.IsFly(), ma.FixedStance())
	l.Debugf("stance guard: rewrote sentinel [%d]->[%d] for monster [%d] (isFly=%t)", stance, resolved, monsterId, ma.IsFly())
	return resolved
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/writer/ -run TestResolveSpawnStance -v`
Expected: PASS (both `TestResolveSpawnStance` and `TestResolveSpawnStanceFailSafe`).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/writer/monster_spawn.go \
        services/atlas-channel/atlas.com/channel/socket/writer/monster_spawn_test.go
git commit -m "feat(task-179): narrow 0/1-sentinel stance guard in atlas-channel writer"
```

---

## Task 7: Wire the guard into the spawn + control emit sites

Call `resolveSpawnStance` at both `NewMonster(...)` sites so no spawn/control packet emits a `0`/`1` sentinel. The spawn-wire debug log now prints the resolved value (NFR-4).

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/writer/monster_spawn.go` (`SpawnMonsterWithEffectBody`, line ~51)
- Modify: `services/atlas-channel/atlas.com/channel/socket/writer/monster_control.go` (`ControlMonsterBody`, line ~51)

**Interfaces:**
- Consumes: `resolveSpawnStance` (Task 6); `monster.Model.Stance()`, `.MonsterId()` (existing).
- Produces: no new exported interface — the wire byte at both sites is now guarded.

- [ ] **Step 1: Wire the spawn site**

In `services/atlas-channel/atlas.com/channel/socket/writer/monster_spawn.go`, inside `SpawnMonsterWithEffectBody`'s inner closure, compute the guarded stance before the debug log so the log prints the resolved value. Replace:

```go
			l.Debugf("Spawn monster wire: uniqueId=[%d] monsterId=[%d] x=[%d] y=[%d] fh=[%d] stance=[%d] newSpawn=[%t] controlled=[%t]",
				m.UniqueId(), m.MonsterId(), x, y, m.Fh(), m.Stance(), newSpawn, m.Controlled())

			mem := packetmodel.NewMonster(x, y, m.Stance(), m.Fh(), appearType, m.Team())
```

with:

```go
			stance := resolveSpawnStance(l, ctx, m.Stance(), m.MonsterId())

			l.Debugf("Spawn monster wire: uniqueId=[%d] monsterId=[%d] x=[%d] y=[%d] fh=[%d] stance=[%d] newSpawn=[%t] controlled=[%t]",
				m.UniqueId(), m.MonsterId(), x, y, m.Fh(), stance, newSpawn, m.Controlled())

			mem := packetmodel.NewMonster(x, y, stance, m.Fh(), appearType, m.Team())
```

- [ ] **Step 2: Wire the control site**

In `services/atlas-channel/atlas.com/channel/socket/writer/monster_control.go`, inside `ControlMonsterBody`'s `controlType > ControlMonsterTypeReset` branch, replace:

```go
				x, y := dmap.SnapMobPosition(l, ctx, m.MapId(), m.X(), m.Y(), m.Fh())
				mem = packetmodel.NewMonster(x, y, m.Stance(), m.Fh(), packetmodel.MonsterAppearTypeRegen, m.Team())
```

with:

```go
				x, y := dmap.SnapMobPosition(l, ctx, m.MapId(), m.X(), m.Y(), m.Fh())
				stance := resolveSpawnStance(l, ctx, m.Stance(), m.MonsterId())
				mem = packetmodel.NewMonster(x, y, stance, m.Fh(), packetmodel.MonsterAppearTypeRegen, m.Team())
```

- [ ] **Step 3: Build to confirm both sites compile**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./...`
Expected: no output (clean). (`ctx` is in scope in both closures — spawn via the outer `func(l, ctx)`, control likewise.)

- [ ] **Step 4: Run the writer package tests**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./socket/writer/...`
Expected: PASS (existing writer tests + Task 6 guard tests).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/writer/monster_spawn.go \
        services/atlas-channel/atlas.com/channel/socket/writer/monster_control.go
git commit -m "feat(task-179): apply stance guard at spawn + control emit sites"
```

---

## Task 8: Full verification + build gates

Run every verification gate from CLAUDE.md across all changed modules and confirm the whole feature is green.

**Files:** none (verification only).

**Interfaces:** none.

- [ ] **Step 1: Test + vet each changed module**

```bash
cd libs/atlas-constants && go test -race ./... && go vet ./...
cd ../../services/atlas-monsters/atlas.com/monsters && go test -race ./... && go vet ./...
cd ../../../atlas-channel/atlas.com/channel && go test -race ./... && go vet ./...
```

Expected: all PASS, vet clean (no output).

> Path note: adjust the relative `cd`s to your worktree layout — the three modules are `libs/atlas-constants`, `services/atlas-monsters/atlas.com/monsters`, `services/atlas-channel/atlas.com/channel`. Run each from the worktree root if the chained `cd`s drift.

- [ ] **Step 2: Build each changed service**

```bash
cd services/atlas-monsters/atlas.com/monsters && go build ./...
cd ../../../atlas-channel/atlas.com/channel && go build ./...
```

Expected: no output (clean).

- [ ] **Step 3: Docker bake both services (mandatory)**

From the worktree root:

```bash
docker buildx bake atlas-monsters
docker buildx bake atlas-channel
```

Expected: both succeed. (`libs/atlas-constants` is an existing shared lib already `COPY`d in the root `Dockerfile` and listed in `go.work`, so no Dockerfile/go.work edit is needed — but if the bake fails on a missing `COPY libs/atlas-constants`, add the two `COPY` lines + `go.work` entry per CLAUDE.md. Verify it is already present first.)

- [ ] **Step 4: Repo-root guards**

From the repo root:

```bash
tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/lint.sh --check
```

Expected: all clean. If `tools/lint.sh --check` reports formatting diffs, run `tools/lint.sh` (fix mode) and re-commit the formatting changes.

- [ ] **Step 5: Confirm no packet-layout change**

Confirm `libs/atlas-packet` has no diff:

```bash
git diff --stat main -- libs/atlas-packet
```

Expected: empty (no files under `libs/atlas-packet` changed). This satisfies the PRD §10 "no packet layout change" criterion — only the `moveAction` value differs, never the encoder.

- [ ] **Step 6: Final commit (if fix-mode lint rewrote anything)**

```bash
git add -A
git commit -m "chore(task-179): lint/format fixes"
```

(Skip if the tree is already clean.)

---

## Self-Review

**1. Spec coverage** — every PRD FR and acceptance criterion maps to a task:

- FR-1 (fly-class derivation): Tasks 3 + 5 carry `flying`/`swimming`; `IsFly()=flying||swimming`.
- FR-2 (idle stance computation, helper `idleMoveAction`): Task 2 (`IdleMoveAction`) + its §10-vector test.
- FR-3 (fresh-spawn origin, replace `5`): Task 4.
- FR-4 (sentinel guard at emit boundary, both sites, fly-aware, cached, narrow): Tasks 5 (fields+cache), 6 (guard+fail-safe), 7 (both emit sites).
- FR-5 (scope fences): honored — Tasks 6/7 touch only the two `NewMonster` emit sites; `processor.go:1511` and the live-move path are untouched.
- §10 acceptance criteria: helper vectors (Task 2), invariant sweep (Tasks 2/4/6), Create no-longer-`5` (Task 4), guard rewrite + pass-through (Task 6), cache-on-path (Task 5 reuse + Task 6 note), build/vet/test/bake/lint/guards (Task 8), no-layout-change (Task 8 Step 5).
- NFR-1 (client re-verification): Task 1. NFR-2 (cache): Task 5 reuse + Task 6. NFR-3 (tenancy): existing cache keyed by `(tenant.Id, monsterId)`; all fetches context-scoped. NFR-4 (observability): Task 6 debug log + Task 7 spawn-wire log prints resolved value. NFR-5 (version scope): value-only change, no version gate touched (Task 8 Step 5).

**2. Placeholder scan** — no `TBD`/`TODO`/"add error handling"/"similar to Task N" left; every code step shows complete code.

**3. Type consistency** — `IdleMoveAction(isFly bool, fixedStance uint32) byte`, `IsFly() bool`, `FixedStance() uint32`, `resolveSpawnStance(l logrus.FieldLogger, ctx context.Context, stance byte, monsterId uint32) byte`, and `newInformationProcessor = information.NewProcessor` are used identically wherever they appear across Tasks 2/4/6/7. The shared lib is imported as `mobconst` in both consuming services. Builder setters `SetFlying`/`SetSwimming`/`SetFixedStance` match between Tasks 3, 4, 5, 6.
