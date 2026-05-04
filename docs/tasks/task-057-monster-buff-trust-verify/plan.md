# Monster-Buff Trust-but-Verify (Doom Handler Removal) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the per-skill Priest Doom handler and the trust-the-client `applyToMobs` block with a single server-authoritative path inside `applyToMobs` that verifies the client's `affectedMobIds` against an atlas-monsters rect query, enforces the WZ-defined `mobCount` cap, rolls `prop` per target, skips reflect-active mobs, and emits structured warn logs on client/server divergence.

**Architecture:** Add `PriestDoomId` to the wire-decoder's `isMobAffectingBuff` allowlist (the precondition that opens the dual-apply window). Introduce a sibling file `services/atlas-channel/atlas.com/channel/skill/handler/mob_select.go` that holds pure helpers (bbox math, intersection, kind classification, prop carve-out). Extend `applyToMobs` in `common.go` with new orchestration that consults those helpers via package-level test seams. Delete the per-skill `doom/` subpackage and its blank import. Migrate the deleted handler's test coverage to two new files (`mob_select_test.go`, `common_apply_to_mobs_test.go`).

**Tech Stack:** Go 1.x, atlas-channel service, `libs/atlas-packet`, `libs/atlas-constants` (skill, monster, point), `libs/atlas-tenant`, logrus, standard `math/rand`.

---

## Task Map

The plan lands in the order from design §8:

1. Wire-decoder change: add Doom to `isMobAffectingBuff` (Task 1).
2. Pure helpers in `mob_select.go`, each pinned by a test (Tasks 2–6).
3. Test seams + new orchestration in `common.go` (Tasks 7–9).
4. Orchestration tests in `common_apply_to_mobs_test.go` (Task 10).
5. Tear down: delete the `doom/` subpackage and remove the blank import (Tasks 11–12).
6. Final cross-package build + tests (Task 13).

Tasks within a group commit independently so the dual-apply window only exists between Tasks 1 and 9, and is fully closed by the time Task 11 lands.

---

### Task 1: Add `PriestDoomId` to `isMobAffectingBuff`

**Files:**
- Modify: `libs/atlas-packet/model/skill_usage_info.go:73-128`
- Test: `libs/atlas-packet/model/skill_usage_info_test.go` (create if absent; otherwise append)

#### Step 1.1: Locate the existing test file (or note its absence)

- [ ] **Check whether a sibling test file exists.**

Run:
```
ls libs/atlas-packet/model/skill_usage_info_test.go 2>/dev/null && echo EXISTS || echo MISSING
```

Expected: either `EXISTS` (we will append a test) or `MISSING` (we will create one in step 1.2).

#### Step 1.2: Write the failing test

Append (or create) the file `libs/atlas-packet/model/skill_usage_info_test.go` so that it contains:

- [ ] **Add the failing test.**

```go
package model

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

func TestIsMobAffectingBuff_PriestDoom(t *testing.T) {
	if !isMobAffectingBuff(skill.PriestDoomId) {
		t.Fatalf("isMobAffectingBuff(PriestDoomId) = false, want true")
	}
}
```

If `package model` already declares `_ "github.com/Chronicle20/atlas/libs/atlas-constants/skill"` or another import group, place this test in its own file `skill_usage_info_doom_test.go` to keep the diff minimal.

#### Step 1.3: Run the test and confirm it fails

- [ ] **Run the test.**

```
cd libs/atlas-packet && go test ./model -run TestIsMobAffectingBuff_PriestDoom
```

Expected: FAIL with `isMobAffectingBuff(PriestDoomId) = false, want true`.

#### Step 1.4: Add `PriestDoomId` to the allowlist

- [ ] **Modify `libs/atlas-packet/model/skill_usage_info.go` — add one entry to `isMobAffectingBuff`'s `skill.Is` argument list.**

Insert immediately before `skill.PriestDispelId` (preserves the existing 2310/2311 grouping):

```go
		skill.PriestDoomId,
```

The resulting block fragment around lines 92-96:

```go
		skill.IceLightningArchMagicianMapleWarriorId,
		skill.ClericBlessId,
		skill.PriestDoomId,
		skill.PriestDispelId,
		skill.PriestHolySymbolId,
```

#### Step 1.5: Run the test and confirm it passes

- [ ] **Run the test.**

```
cd libs/atlas-packet && go test ./model -run TestIsMobAffectingBuff_PriestDoom
```

Expected: PASS.

Then run the full `libs/atlas-packet` test suite:

```
cd libs/atlas-packet && go test ./...
```

Expected: PASS.

#### Step 1.6: Commit

- [ ] **Commit.**

```
git add libs/atlas-packet/model/skill_usage_info.go libs/atlas-packet/model/skill_usage_info_doom_test.go
git commit -m "feat(packet): include Priest Doom in isMobAffectingBuff allowlist"
```

(Use `skill_usage_info_test.go` instead of `skill_usage_info_doom_test.go` if step 1.1 reported `EXISTS`.)

---

### Task 2: Create `mob_select.go` skeleton + bbox helpers (TDD)

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/skill/handler/mob_select.go`
- Create: `services/atlas-channel/atlas.com/channel/skill/handler/mob_select_test.go`

#### Step 2.1: Write four failing bounding-box tests

Create `services/atlas-channel/atlas.com/channel/skill/handler/mob_select_test.go`:

- [ ] **Add the failing tests.**

```go
package handler

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/point"
)

func mkPoint(x, y int16) point.Model {
	return point.NewModel(point.X(x), point.Y(y))
}

func TestBoundingBox_FacingRight_SymmetricRect(t *testing.T) {
	lt := mkPoint(-200, -100)
	rb := mkPoint(200, 100)
	x1, y1, x2, y2 := calculateBoundingBox(0, 0, false, lt, rb)
	if x1 != -200 || y1 != -100 || x2 != 200 || y2 != 100 {
		t.Fatalf("got (%d,%d,%d,%d), want (-200,-100,200,100)", x1, y1, x2, y2)
	}
}

func TestBoundingBox_FacingLeft_SymmetricRect(t *testing.T) {
	lt := mkPoint(-200, -100)
	rb := mkPoint(200, 100)
	x1, y1, x2, y2 := calculateBoundingBox(0, 0, true, lt, rb)
	if x1 != -200 || y1 != -100 || x2 != 200 || y2 != 100 {
		t.Fatalf("got (%d,%d,%d,%d), want (-200,-100,200,100)", x1, y1, x2, y2)
	}
}

func TestBoundingBox_Asymmetric_FacingRight(t *testing.T) {
	lt := mkPoint(-50, -10)
	rb := mkPoint(150, 30)
	// facing right: x1 = casterX - rb.X = 100 - 150 = -50; x2 = casterX - lt.X = 100 - (-50) = 150
	// y1 = casterY + lt.Y = 50 + (-10) = 40; y2 = casterY + rb.Y = 50 + 30 = 80
	x1, y1, x2, y2 := calculateBoundingBox(100, 50, false, lt, rb)
	if x1 != -50 || y1 != 40 || x2 != 150 || y2 != 80 {
		t.Fatalf("got (%d,%d,%d,%d), want (-50,40,150,80)", x1, y1, x2, y2)
	}
}

func TestBoundingBox_Asymmetric_FacingLeft(t *testing.T) {
	lt := mkPoint(-50, -10)
	rb := mkPoint(150, 30)
	// facing left: x1 = casterX + lt.X = 100 + (-50) = 50; x2 = casterX + rb.X = 100 + 150 = 250
	// y1 = casterY + lt.Y = 50 + (-10) = 40; y2 = casterY + rb.Y = 50 + 30 = 80
	x1, y1, x2, y2 := calculateBoundingBox(100, 50, true, lt, rb)
	if x1 != 50 || y1 != 40 || x2 != 250 || y2 != 80 {
		t.Fatalf("got (%d,%d,%d,%d), want (50,40,250,80)", x1, y1, x2, y2)
	}
}
```

#### Step 2.2: Run and confirm failing

- [ ] **Run.**

```
cd services/atlas-channel/atlas.com/channel && go test ./skill/handler -run TestBoundingBox
```

Expected: FAIL — `undefined: calculateBoundingBox`.

#### Step 2.3: Create `mob_select.go` with `calculateBoundingBox`

Create `services/atlas-channel/atlas.com/channel/skill/handler/mob_select.go`:

- [ ] **Add the file.**

```go
package handler

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/point"
)

// calculateBoundingBox derives the (x1, y1, x2, y2) target rectangle for a
// monster-buff skill cast. Mirrors Cosmic StatEffect.calculateBoundingBox.
//
// When the caster faces left, the rectangle is (casterPos + lt) → (casterPos + rb).
// When the caster faces right, the rectangle mirrors about the caster's X:
// x1 = casterX - rb.X, x2 = casterX - lt.X. The y bounds are always
// (casterY + lt.Y) → (casterY + rb.Y).
//
// The returned tuple is not normalized — atlas-monsters' GetInFieldRect
// normalizes (min, max) on its side, so callers can pass either ordering.
func calculateBoundingBox(casterX, casterY int16, facingLeft bool, lt, rb point.Model) (x1, y1, x2, y2 int16) {
	if facingLeft {
		x1 = casterX + int16(lt.X())
		y1 = casterY + int16(lt.Y())
		x2 = casterX + int16(rb.X())
		y2 = casterY + int16(rb.Y())
	} else {
		x1 = casterX - int16(rb.X())
		y1 = casterY + int16(lt.Y())
		x2 = casterX - int16(lt.X())
		y2 = casterY + int16(rb.Y())
	}
	return
}
```

#### Step 2.4: Run and confirm passing

- [ ] **Run.**

```
cd services/atlas-channel/atlas.com/channel && go test ./skill/handler -run TestBoundingBox
```

Expected: PASS (4/4).

#### Step 2.5: Commit

- [ ] **Commit.**

```
git add services/atlas-channel/atlas.com/channel/skill/handler/mob_select.go \
        services/atlas-channel/atlas.com/channel/skill/handler/mob_select_test.go
git commit -m "feat(channel/handler): add mob_select.go with calculateBoundingBox"
```

---

### Task 3: `hasEffectBbox` helper (TDD)

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/skill/handler/mob_select.go`
- Modify: `services/atlas-channel/atlas.com/channel/skill/handler/mob_select_test.go`

#### Step 3.1: Write the failing test

Append to `mob_select_test.go`:

- [ ] **Add `TestHasEffectBbox`.**

```go
func TestHasEffectBbox(t *testing.T) {
	tests := []struct {
		name string
		lt   point.Model
		rb   point.Model
		want bool
	}{
		{"all-zero is sentinel for no-rect", mkPoint(0, 0), mkPoint(0, 0), false},
		{"any non-zero on lt counts as rect", mkPoint(-1, 0), mkPoint(0, 0), true},
		{"any non-zero on rb counts as rect", mkPoint(0, 0), mkPoint(0, 1), true},
		{"full rect is rect", mkPoint(-50, -10), mkPoint(150, 30), true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := hasEffectBbox(tc.lt, tc.rb); got != tc.want {
				t.Fatalf("hasEffectBbox = %v, want %v", got, tc.want)
			}
		})
	}
}
```

#### Step 3.2: Run and confirm failing

- [ ] **Run.**

```
cd services/atlas-channel/atlas.com/channel && go test ./skill/handler -run TestHasEffectBbox
```

Expected: FAIL — `undefined: hasEffectBbox`.

#### Step 3.3: Implement `hasEffectBbox`

Append to `mob_select.go`:

- [ ] **Add the helper.**

```go
// hasEffectBbox reports whether the effect carries a non-degenerate target
// rectangle. The WZ "no rect contract" sentinel is all four components zero;
// any non-zero component (even a single int) indicates the effect prescribes
// a rect. No v83 skill ships a literal zero-area effect, so the conflation is
// safe in production.
func hasEffectBbox(lt, rb point.Model) bool {
	return lt.X() != 0 || lt.Y() != 0 || rb.X() != 0 || rb.Y() != 0
}
```

#### Step 3.4: Run and confirm passing

- [ ] **Run.**

```
cd services/atlas-channel/atlas.com/channel && go test ./skill/handler -run TestHasEffectBbox
```

Expected: PASS (1 parent test, 4 subtests).

#### Step 3.5: Commit

- [ ] **Commit.**

```
git add services/atlas-channel/atlas.com/channel/skill/handler/mob_select.go \
        services/atlas-channel/atlas.com/channel/skill/handler/mob_select_test.go
git commit -m "feat(channel/handler): add hasEffectBbox WZ-no-rect sentinel helper"
```

---

### Task 4: `intersectMobIds` helper (TDD)

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/skill/handler/mob_select.go`
- Modify: `services/atlas-channel/atlas.com/channel/skill/handler/mob_select_test.go`

#### Step 4.1: Write the failing tests

Append to `mob_select_test.go`:

- [ ] **Add intersection tests.**

```go
import "reflect"

func TestIntersectMobIds_AllInRect(t *testing.T) {
	applied, anomaly := intersectMobIds([]uint32{1, 2, 3}, []uint32{1, 2, 3})
	if !reflect.DeepEqual(applied, []uint32{1, 2, 3}) {
		t.Errorf("applied = %v, want [1 2 3]", applied)
	}
	if len(anomaly) != 0 {
		t.Errorf("anomaly = %v, want []", anomaly)
	}
}

func TestIntersectMobIds_ClientOrderPreserved(t *testing.T) {
	// client lists 5,3,1 in this order; server returns 1,3,5 (different order).
	// Result must follow client order.
	applied, anomaly := intersectMobIds([]uint32{5, 3, 1}, []uint32{1, 3, 5})
	if !reflect.DeepEqual(applied, []uint32{5, 3, 1}) {
		t.Errorf("applied = %v, want [5 3 1]", applied)
	}
	if len(anomaly) != 0 {
		t.Errorf("anomaly = %v, want []", anomaly)
	}
}

func TestIntersectMobIds_AnomalySubset(t *testing.T) {
	// client lists 1,2,3,99 — server returned 1,2,3. Mob 99 is anomaly.
	applied, anomaly := intersectMobIds([]uint32{1, 2, 3, 99}, []uint32{1, 2, 3})
	if !reflect.DeepEqual(applied, []uint32{1, 2, 3}) {
		t.Errorf("applied = %v, want [1 2 3]", applied)
	}
	if !reflect.DeepEqual(anomaly, []uint32{99}) {
		t.Errorf("anomaly = %v, want [99]", anomaly)
	}
}

func TestIntersectMobIds_ServerOnlyDropped(t *testing.T) {
	// server returned 1,2,3 — client only sent 1. The other two are NOT
	// applied (we trust client's omission as "did not target").
	applied, anomaly := intersectMobIds([]uint32{1}, []uint32{1, 2, 3})
	if !reflect.DeepEqual(applied, []uint32{1}) {
		t.Errorf("applied = %v, want [1]", applied)
	}
	if len(anomaly) != 0 {
		t.Errorf("anomaly = %v, want []", anomaly)
	}
}

func TestIntersectMobIds_EmptyClient(t *testing.T) {
	applied, anomaly := intersectMobIds(nil, []uint32{1, 2})
	if len(applied) != 0 || len(anomaly) != 0 {
		t.Errorf("applied=%v, anomaly=%v, want both empty", applied, anomaly)
	}
}
```

If `import "reflect"` would duplicate an existing block, fold it into the file's existing `import (...)` group instead of adding a second import line.

#### Step 4.2: Run and confirm failing

- [ ] **Run.**

```
cd services/atlas-channel/atlas.com/channel && go test ./skill/handler -run TestIntersectMobIds
```

Expected: FAIL — `undefined: intersectMobIds`.

#### Step 4.3: Implement `intersectMobIds`

Append to `mob_select.go`:

- [ ] **Add the helper.**

```go
// intersectMobIds partitions client mob ids into "applied" (also present in
// server) and "anomaly" (client-only) lists. Server-only ids are dropped per
// FR-4.1: the client's omission is treated as authoritative for "did not
// target". Result preserves client order (FR-4.4) so wire traces remain
// readable. Both returned slices are nil if the corresponding bucket is
// empty (callers checking len() observe the same behavior either way).
func intersectMobIds(client, server []uint32) (applied, anomaly []uint32) {
	if len(client) == 0 {
		return nil, nil
	}
	serverSet := make(map[uint32]struct{}, len(server))
	for _, id := range server {
		serverSet[id] = struct{}{}
	}
	for _, id := range client {
		if _, ok := serverSet[id]; ok {
			applied = append(applied, id)
		} else {
			anomaly = append(anomaly, id)
		}
	}
	return applied, anomaly
}
```

#### Step 4.4: Run and confirm passing

- [ ] **Run.**

```
cd services/atlas-channel/atlas.com/channel && go test ./skill/handler -run TestIntersectMobIds
```

Expected: PASS (5 tests).

#### Step 4.5: Commit

- [ ] **Commit.**

```
git add services/atlas-channel/atlas.com/channel/skill/handler/mob_select.go \
        services/atlas-channel/atlas.com/channel/skill/handler/mob_select_test.go
git commit -m "feat(channel/handler): add intersectMobIds for rect verification"
```

---

### Task 5: `mobBuffApplyKind` helper (TDD)

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/skill/handler/mob_select.go`
- Modify: `services/atlas-channel/atlas.com/channel/skill/handler/mob_select_test.go`

#### Step 5.1: Write the failing test

Append to `mob_select_test.go`:

- [ ] **Add `TestMobBuffApplyKind`.**

```go
import (
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

func TestMobBuffApplyKind(t *testing.T) {
	if got := mobBuffApplyKind(skill2.PriestDoomId); got != "MAGICAL" {
		t.Errorf("mobBuffApplyKind(PriestDoomId) = %q, want MAGICAL", got)
	}
	if got := mobBuffApplyKind(skill2.Id(999999999)); got != "" {
		t.Errorf("mobBuffApplyKind(unknown) = %q, want empty", got)
	}
}
```

(If the file already has an `import (...)` group, fold `skill2 "..."` into it
to keep one block.)

#### Step 5.2: Run and confirm failing

- [ ] **Run.**

```
cd services/atlas-channel/atlas.com/channel && go test ./skill/handler -run TestMobBuffApplyKind
```

Expected: FAIL — `undefined: mobBuffApplyKind`.

#### Step 5.3: Implement `mobBuffApplyKind`

Append to `mob_select.go`:

- [ ] **Add the helper.**

```go
import (
	monster2 "github.com/Chronicle20/atlas/libs/atlas-constants/monster"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

// mobBuffApplyKind returns the reflect-kind that gates a mob-affecting buff
// apply (FR-4.6). Today only Priest Doom is in `isMobAffectingBuff` for the
// apply branch; future apply-style status skills are added here as they are
// wired in. Returning "" tells the orchestrator to skip the reflect check
// entirely and emit a debug "unclassified kind" log — the cast still proceeds.
//
// Crash/Dispel kinds continue to come from dispelSkillClass (common.go) and
// are not handled here.
func mobBuffApplyKind(skillId skill2.Id) string {
	switch {
	case skill2.Is(skillId, skill2.PriestDoomId):
		return monster2.ReflectKindMagical
	default:
		return ""
	}
}
```

(If `mob_select.go` already imports `monster2` or `skill2`, fold the new
imports into the existing block — there must be exactly one `import (...)`
group at the top of the file.)

#### Step 5.4: Run and confirm passing

- [ ] **Run.**

```
cd services/atlas-channel/atlas.com/channel && go test ./skill/handler -run TestMobBuffApplyKind
```

Expected: PASS.

#### Step 5.5: Commit

- [ ] **Commit.**

```
git add services/atlas-channel/atlas.com/channel/skill/handler/mob_select.go \
        services/atlas-channel/atlas.com/channel/skill/handler/mob_select_test.go
git commit -m "feat(channel/handler): add mobBuffApplyKind reflect-kind classifier"
```

---

### Task 6: `propBranch` enum + `propAppliesTo` carve-out (TDD)

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/skill/handler/mob_select.go`
- Modify: `services/atlas-channel/atlas.com/channel/skill/handler/mob_select_test.go`

#### Step 6.1: Write the failing tests

Append to `mob_select_test.go`:

- [ ] **Add carve-out tests.**

```go
func TestPropAppliesTo_DefaultsTrue(t *testing.T) {
	tests := []struct {
		sid    skill2.Id
		branch propBranch
	}{
		{skill2.PriestDoomId, propBranchApply},
		{skill2.CrusaderArmorCrashId, propBranchCancel},
		{skill2.WhiteKnightMagicCrashId, propBranchCancel},
		{skill2.DragonKnightPowerCrashId, propBranchCancel},
		{skill2.PriestDispelId, propBranchCancel},
	}
	for _, tc := range tests {
		if !propAppliesTo(tc.sid, tc.branch) {
			t.Errorf("propAppliesTo(%v, %v) = false, want true (defaults)", tc.sid, tc.branch)
		}
	}
}

func TestPropAppliesTo_CarveOutHonored(t *testing.T) {
	// Install a deny entry for a synthetic id; restore on cleanup.
	id := skill2.Id(0xDEAD0001)
	prev := propCarveOut[id]
	propCarveOut[id] = map[propBranch]bool{propBranchCancel: false}
	t.Cleanup(func() {
		if prev == nil {
			delete(propCarveOut, id)
		} else {
			propCarveOut[id] = prev
		}
	})

	if propAppliesTo(id, propBranchCancel) {
		t.Errorf("propAppliesTo(synthetic, cancel) = true, want false (deny entry)")
	}
	if !propAppliesTo(id, propBranchApply) {
		t.Errorf("propAppliesTo(synthetic, apply) = false, want true (apply not carved out)")
	}
}
```

#### Step 6.2: Run and confirm failing

- [ ] **Run.**

```
cd services/atlas-channel/atlas.com/channel && go test ./skill/handler -run TestPropAppliesTo
```

Expected: FAIL — `undefined: propBranch`, `undefined: propAppliesTo`, `undefined: propCarveOut`.

#### Step 6.3: Implement the carve-out machinery

Append to `mob_select.go`:

- [ ] **Add the type, table, and helper.**

```go
// propBranch discriminates which emit branch the orchestrator is about to
// take when consulting the prop carve-out table (FR-4.5).
type propBranch int

const (
	propBranchApply propBranch = iota
	propBranchCancel
)

// propCarveOut overrides the default "prop applies to both branches" rule
// per (skillId, branch). Default is `true` for every (skill, branch) not
// listed here; an entry with value `false` suppresses the prop roll on that
// branch (treats it as "always pass" for that skill). Today the table is
// empty: every current skill takes the default. The table is the contract
// for future skills whose WZ data prescribes "prop only on apply" or "prop
// only on cancel".
var propCarveOut = map[skill2.Id]map[propBranch]bool{}

// propAppliesTo reports whether the orchestrator should roll prop for the
// given (skill, branch). Defaults to `true`.
func propAppliesTo(skillId skill2.Id, branch propBranch) bool {
	if entry, ok := propCarveOut[skillId]; ok {
		if v, set := entry[branch]; set {
			return v
		}
	}
	return true
}
```

#### Step 6.4: Run and confirm passing

- [ ] **Run.**

```
cd services/atlas-channel/atlas.com/channel && go test ./skill/handler -run TestPropAppliesTo
```

Expected: PASS.

Run the whole `mob_select_test.go` suite to confirm earlier helpers still pass:

```
cd services/atlas-channel/atlas.com/channel && go test ./skill/handler -run "TestBoundingBox|TestHasEffectBbox|TestIntersectMobIds|TestMobBuffApplyKind|TestPropAppliesTo"
```

Expected: PASS.

#### Step 6.5: Commit

- [ ] **Commit.**

```
git add services/atlas-channel/atlas.com/channel/skill/handler/mob_select.go \
        services/atlas-channel/atlas.com/channel/skill/handler/mob_select_test.go
git commit -m "feat(channel/handler): add propBranch enum and propAppliesTo carve-out"
```

---

### Task 7: Add test seams to `common.go`

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/skill/handler/common.go`

This task is a refactor only: the production behavior is unchanged at the
end of it. The seams are introduced in the same package as `applyToMobs`
so subsequent tasks can plug into them.

#### Step 7.1: Add the six package-level seam vars

- [ ] **Insert this block immediately after the `import (...)` group (top of `common.go`, before `func UseSkill`):**

```go
import (
	"math/rand"

	"atlas-channel/character"
	"atlas-channel/data/skill/effect"
	"atlas-channel/monster"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// loadCasterFunc is the caster-load seam tests can replace. Production
// calls atlas-character via character.Processor.GetById(); tests inject a
// stub returning a deterministic character.Model so the orchestrator can
// exercise its mob-selection / status-apply logic offline.
var loadCasterFunc = func(cp character.Processor, characterId uint32) (character.Model, error) {
	return cp.GetById()(characterId)
}

// rectQueryFunc is the mob-selection seam tests can replace. Production
// calls atlas-monsters via monster.Processor.GetInMapRect; tests inject a
// stub returning a fixed slice.
var rectQueryFunc = func(p *monster.Processor, f field.Model, x1, y1, x2, y2 int16, limit uint32) ([]monster.Model, error) {
	return p.GetInMapRect(f, x1, y1, x2, y2, limit)
}

// propRollFunc gates per-target apply/cancel by the skill's prop value.
// Production uses a uniform RNG; tests inject a deterministic implementation
// via a t.Cleanup-restored override.
var propRollFunc = func(prop float64) bool {
	if prop <= 0 {
		return false
	}
	if prop >= 1 {
		return true
	}
	return rand.Float64() <= prop
}

// reflectLookupFunc is the magic-reflect probe seam tests can replace.
var reflectLookupFunc = func(t tenant.Model, monsterId uint32, kind string) (monster.ReflectInfo, bool) {
	return monster.GetStatusMirror().GetReflect(t, monsterId, kind)
}

// applyStatusFunc is the status-apply emit seam tests can replace.
var applyStatusFunc = func(p *monster.Processor, f field.Model, monsterId, characterId, skillId, skillLevel uint32, statuses map[string]int32, duration uint32) error {
	return p.ApplyStatus(f, monsterId, characterId, skillId, skillLevel, statuses, duration)
}

// cancelStatusFunc is the status-cancel emit seam tests can replace.
var cancelStatusFunc = func(p *monster.Processor, f field.Model, monsterId uint32, statusTypes []string, sourceCharacterId, sourceSkillId uint32, sourceSkillClass string) error {
	return p.CancelStatus(f, monsterId, statusTypes, sourceCharacterId, sourceSkillId, sourceSkillClass)
}
```

The first import block should already exist; merge the new imports
(`math/rand`, `tenant`) into the existing parenthesized group. Do **not**
end up with two `import (...)` groups in `common.go`.

The signature of `loadCasterFunc` accepts a `character.Processor` interface
value (the same shape `character.NewProcessor` already returns), not a pointer.
This matches the pattern from the deleted `doom.go:29-31`.

#### Step 7.2: Compile to confirm no behavior change

- [ ] **Build.**

```
cd services/atlas-channel/atlas.com/channel && go build ./...
```

Expected: clean build, no errors.

```
cd services/atlas-channel/atlas.com/channel && go test ./skill/handler/...
```

Expected: existing tests (`registry_test.go`, `mob_select_test.go`) pass; no new failures.

#### Step 7.3: Commit

- [ ] **Commit.**

```
git add services/atlas-channel/atlas.com/channel/skill/handler/common.go
git commit -m "refactor(channel/handler): add applyToMobs test seams"
```

---

### Task 8: Extend `applyToMobs` orchestration

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/skill/handler/common.go:75-104`

This task replaces the existing `applyToMobs` body. Behavior changes here
introduce trust-but-verify; tests in Task 10 pin the new behavior.

#### Step 8.1: Replace the `applyToMobs` body

- [ ] **Replace the existing body of `applyToMobs` (`common.go:75-104`) with this implementation:**

```go
func applyToMobs(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId uint32, info packetmodel.SkillUsageInfo, e effect.Model) {
	mobIds := info.AffectedMobIds()
	if len(mobIds) == 0 {
		return
	}

	sid := skill2.Id(info.SkillId())
	slvl := uint32(info.SkillLevel())
	cap := e.MobCount()

	// FR-4.3 — mobCount cap. Reject the entire cast if the client claims more
	// targets than the skill's WZ definition permits. This runs before any
	// atlas-monsters round-trip; an over-cap cast produces zero emit calls.
	if uint32(len(mobIds)) > cap {
		l.WithFields(logrus.Fields{
			"event":            "monster_buff_anomaly_over_cap",
			"character_id":     characterId,
			"skill_id":         uint32(sid),
			"skill_level":      slvl,
			"mob_count_cap":    cap,
			"client_mob_count": len(mobIds),
			"client_mob_ids":   mobIds,
		}).Warn("client_target_count_exceeds_skill_cap")
		return
	}

	mp := monster.NewProcessor(l, ctx)

	var (
		applied         []uint32
		anomaly         []uint32
		mobsInRectCount = -1
		rect            [4]int16 // x1, y1, x2, y2 — only meaningful when bbox present
	)

	if !hasEffectBbox(e.LT(), e.RB()) {
		// FR-4.2 — no rect contract in WZ data; trust the client unmodified
		// for the rect check. Cap (already done), prop, reflect still apply.
		l.WithFields(logrus.Fields{
			"skill_id":         uint32(sid),
			"skill_level":      slvl,
			"client_mob_count": len(mobIds),
		}).Debug("mob_buff_no_effect_bbox")
		applied = mobIds
	} else {
		// FR-4.1 — rect verification. Bail-on-error policy: any failure
		// drops the cast. See design §5.1.
		cp := character.NewProcessor(l, ctx)
		c, cErr := loadCasterFunc(cp, characterId)
		if cErr != nil {
			l.WithError(cErr).WithFields(logrus.Fields{
				"event":        "mob_buff_caster_load_failed",
				"character_id": characterId,
				"skill_id":     uint32(sid),
			}).Error("mob_buff_caster_load_failed")
			return
		}
		facingLeft := (c.Stance() & 1) == 1
		x1, y1, x2, y2 := calculateBoundingBox(c.X(), c.Y(), facingLeft, e.LT(), e.RB())
		rect = [4]int16{x1, y1, x2, y2}

		mobs, qErr := rectQueryFunc(mp, f, x1, y1, x2, y2, cap)
		if qErr != nil {
			l.WithError(qErr).WithFields(logrus.Fields{
				"event":        "mob_buff_rect_query_failed",
				"character_id": characterId,
				"skill_id":     uint32(sid),
				"rect":         rect,
			}).Error("mob_buff_rect_query_failed")
			return
		}
		serverMobIds := make([]uint32, 0, len(mobs))
		for _, m := range mobs {
			serverMobIds = append(serverMobIds, m.UniqueId())
		}
		mobsInRectCount = len(serverMobIds)

		applied, anomaly = intersectMobIds(mobIds, serverMobIds)

		if len(anomaly) > 0 {
			l.WithFields(logrus.Fields{
				"event":           "monster_buff_anomaly_out_of_rect",
				"character_id":    characterId,
				"skill_id":        uint32(sid),
				"skill_level":     slvl,
				"rect":            map[string]int16{"x1": x1, "y1": y1, "x2": x2, "y2": y2},
				"mob_count_cap":   cap,
				"client_mob_ids":  mobIds,
				"server_mob_ids":  serverMobIds,
				"anomaly_mob_ids": anomaly,
			}).Warn("client_targeted_mob_outside_server_rect")
		}
	}

	t := tenant.MustFromContext(ctx)
	monsterStatuses := make(map[string]int32, len(e.MonsterStatus()))
	for k, v := range e.MonsterStatus() {
		monsterStatuses[k] = int32(v)
	}

	isCancel := isCrashOrDispel(sid)
	cancelClass := ""
	if isCancel {
		cancelClass = dispelSkillClass(sid)
	}

	// Branch selection mirrors the FR-4.9 rule: a skill takes EITHER the
	// cancel branch (Crash family / Priest Dispel) OR the apply branch
	// (Doom and any future entry with non-empty MonsterStatus). Never both.
	branch := propBranchApply
	if isCancel {
		branch = propBranchCancel
	} else if len(monsterStatuses) == 0 {
		// Buff-classified skill with no MonsterStatus map — defensive: nothing
		// to apply. Should not occur for skills in isMobAffectingBuff today.
		l.WithFields(logrus.Fields{
			"skill_id": uint32(sid),
		}).Debug("mob_buff_no_emit_branch")
		l.WithFields(buildSummaryFields(characterId, sid, slvl, mobsInRectCount, len(mobIds), 0, 0, 0, len(anomaly))).Debug("mob_buff_apply_summary")
		return
	}

	appliedCount, reflectSkipped, propSkipped := 0, 0, 0
	for _, mobId := range applied {
		// FR-4.6 — kind-aware reflect skip.
		var kind string
		if isCancel {
			kind = cancelClass
		} else {
			kind = mobBuffApplyKind(sid)
		}
		if kind == "" {
			l.WithFields(logrus.Fields{
				"event":    "mob_buff_unclassified_kind",
				"skill_id": uint32(sid),
				"mob_id":   mobId,
			}).Debug("mob_buff_unclassified_kind")
		} else if _, hasReflect := reflectLookupFunc(t, mobId, kind); hasReflect {
			l.WithFields(logrus.Fields{
				"skill_id": uint32(sid),
				"mob_id":   mobId,
				"kind":     kind,
			}).Debug("mob_buff_reflect_skip")
			reflectSkipped++
			continue
		}

		// FR-4.5 — prop roll, with per-skill carve-out support.
		if propAppliesTo(sid, branch) {
			if !propRollFunc(e.Prop()) {
				propSkipped++
				continue
			}
		}

		// FR-4.9 — branch emit.
		if isCancel {
			_ = cancelStatusFunc(mp, f, mobId, nil, characterId, uint32(sid), cancelClass)
		} else {
			_ = applyStatusFunc(mp, f, mobId, characterId, uint32(sid), slvl, monsterStatuses, uint32(e.Duration()))
		}
		appliedCount++
	}

	l.WithFields(buildSummaryFields(characterId, sid, slvl, mobsInRectCount, len(mobIds), appliedCount, reflectSkipped, propSkipped, len(anomaly))).Debug("mob_buff_apply_summary")
}

// buildSummaryFields packs the FR-4.8 per-cast summary fields.
func buildSummaryFields(characterId uint32, sid skill2.Id, slvl uint32, mobsInRect, clientMobCount, applied, reflectSkipped, propSkipped, outOfRectDropped int) logrus.Fields {
	return logrus.Fields{
		"caster":              characterId,
		"skill_id":            uint32(sid),
		"skill_level":         slvl,
		"mobs_in_rect":        mobsInRect,
		"client_mob_count":    clientMobCount,
		"applied":             applied,
		"reflect_skipped":     reflectSkipped,
		"prop_skipped":        propSkipped,
		"out_of_rect_dropped": outOfRectDropped,
	}
}
```

Notes for the implementer:

- The variable `cap` shadows the built-in `cap`. That is acceptable here since the function does not use `cap()` on a slice. If the linter complains, rename to `mobCap`.
- `character.NewProcessor` returns a `character.Processor` interface; `loadCasterFunc` accepts that interface. Make sure the signature in the seam (Task 7) and the call here agree.
- `monster.NewProcessor` returns `*monster.Processor`. The seam signatures from Task 7 already use `*monster.Processor`.

#### Step 8.2: Build and run existing tests

- [ ] **Build.**

```
cd services/atlas-channel/atlas.com/channel && go build ./...
```

Expected: clean build.

```
cd services/atlas-channel/atlas.com/channel && go test ./skill/handler/...
```

Expected: existing tests still pass. (Orchestration tests for new behavior land in Task 10.)

#### Step 8.3: Commit

- [ ] **Commit.**

```
git add services/atlas-channel/atlas.com/channel/skill/handler/common.go
git commit -m "feat(channel/handler): apply trust-but-verify to applyToMobs"
```

---

### Task 9: Sanity-check that the doom subpackage still builds

After Task 8 the codebase has BOTH the consolidated `applyToMobs` path AND
the per-skill Doom handler installed in the registry — the dual-apply window
the design §8 anticipates. We tear down the doom subpackage in Tasks 11–12.
Before that, confirm no compile regressions slipped in.

#### Step 9.1: Build the whole atlas-channel module

- [ ] **Build.**

```
cd services/atlas-channel/atlas.com/channel && go build ./...
```

Expected: clean.

```
cd services/atlas-channel/atlas.com/channel && go test ./...
```

Expected: pass (Doom handler tests still pass against unchanged seams; the orchestration is exercised by Task 10).

No commit — this is a safety check only.

---

### Task 10: Orchestration tests in `common_apply_to_mobs_test.go`

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/skill/handler/common_apply_to_mobs_test.go`

These tests pin the FR-4.x orchestration through the `common.go` seams. The
`installFakes` helper mirrors the pattern from the deleted `doom_test.go`.

#### Step 10.1: Add the test fixture and helpers

- [ ] **Create the file with this initial content (helpers + first test).**

```go
package handler

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"atlas-channel/character"
	"atlas-channel/data/skill/effect"
	"atlas-channel/monster"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	monster2 "github.com/Chronicle20/atlas/libs/atlas-constants/monster"
	"github.com/Chronicle20/atlas/libs/atlas-constants/point"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// applyCall captures one ApplyStatus invocation so tests can assert on it.
type applyCall struct {
	monsterId uint32
	skillId   uint32
	statuses  map[string]int32
	duration  uint32
}

// cancelCall captures one CancelStatus invocation.
type cancelCall struct {
	monsterId uint32
	skillId   uint32
	class     string
}

// fakes wires deterministic seams for one test.
type fakes struct {
	applies []applyCall
	cancels []cancelCall
}

// installFakes replaces all six seam vars with deterministic implementations
// returning the supplied data. `mobs` controls rectQueryFunc; `reflects`
// controls reflectLookupFunc (keyed by monster id, value is the reflect
// kind that should be reported as active); `propWillFire` controls
// propRollFunc; `caster` is what loadCasterFunc returns; `casterErr` /
// `rectErr` short-circuit those seams.
func installFakes(t *testing.T, caster character.Model, casterErr error, mobs []monster.Model, rectErr error, reflects map[uint32]string, propWillFire bool) *fakes {
	t.Helper()
	f := &fakes{}

	prevLoad := loadCasterFunc
	prevRect := rectQueryFunc
	prevProp := propRollFunc
	prevReflect := reflectLookupFunc
	prevApply := applyStatusFunc
	prevCancel := cancelStatusFunc

	loadCasterFunc = func(_ character.Processor, _ uint32) (character.Model, error) {
		if casterErr != nil {
			return character.Model{}, casterErr
		}
		return caster, nil
	}
	rectQueryFunc = func(_ *monster.Processor, _ field.Model, _, _, _, _ int16, _ uint32) ([]monster.Model, error) {
		if rectErr != nil {
			return nil, rectErr
		}
		return mobs, nil
	}
	propRollFunc = func(_ float64) bool { return propWillFire }
	reflectLookupFunc = func(_ tenant.Model, monsterId uint32, kind string) (monster.ReflectInfo, bool) {
		if want, ok := reflects[monsterId]; ok && want == kind {
			return monster.ReflectInfo{Kind: kind, Percent: 30, ExpiresAt: time.Now().Add(time.Minute)}, true
		}
		return monster.ReflectInfo{}, false
	}
	applyStatusFunc = func(_ *monster.Processor, _ field.Model, monsterId, _, skillId, _ uint32, statuses map[string]int32, duration uint32) error {
		f.applies = append(f.applies, applyCall{monsterId: monsterId, skillId: skillId, statuses: statuses, duration: duration})
		return nil
	}
	cancelStatusFunc = func(_ *monster.Processor, _ field.Model, monsterId uint32, _ []string, _, skillId uint32, class string) error {
		f.cancels = append(f.cancels, cancelCall{monsterId: monsterId, skillId: skillId, class: class})
		return nil
	}

	t.Cleanup(func() {
		loadCasterFunc = prevLoad
		rectQueryFunc = prevRect
		propRollFunc = prevProp
		reflectLookupFunc = prevReflect
		applyStatusFunc = prevApply
		cancelStatusFunc = prevCancel
	})
	return f
}

func mkField() field.Model {
	return field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
}

func mkMob(uniqueId uint32) monster.Model {
	return monster.NewModelBuilder(uniqueId, mkField(), 9300018).MustBuild()
}

func mkCaster(id uint32) character.Model {
	return character.NewModelBuilder().SetId(id).Build()
}

// mkInfo builds a SkillUsageInfo with the given skill id, level, and affected
// mob ids. The wire decoder is exercised in its own test suite — here we
// rely on packetmodel exposing builder fields the test can populate.
func mkInfo(skillId uint32, level uint16, mobIds []uint32) packetmodel.SkillUsageInfo {
	return packetmodel.NewSkillUsageInfoForTest(skillId, level, mobIds)
}

// withRect returns the effect with non-zero LT/RB so hasEffectBbox is true.
func withRect(rm effect.RestModel) effect.RestModel {
	rm.LT = &effect.PointRestModel{X: -200, Y: -100}
	rm.RB = &effect.PointRestModel{X: 200, Y: 100}
	return rm
}

func newDoomEffect(prop float64) effect.Model {
	rm := withRect(effect.RestModel{
		Duration:      60000,
		MonsterStatus: map[string]uint32{monster2.StatusDoom: 1},
		MobCount:      6,
		Prop:          prop,
	})
	se, _ := effect.Extract(rm)
	return se
}

// newDoomEffectNoBbox is a Doom-shaped effect with no rect (FR-4.2 fallback).
func newDoomEffectNoBbox(prop float64) effect.Model {
	se, _ := effect.Extract(effect.RestModel{
		Duration:      60000,
		MonsterStatus: map[string]uint32{monster2.StatusDoom: 1},
		MobCount:      6,
		Prop:          prop,
	})
	return se
}

// newCrashEffect models Crusader Armor Crash: cancel branch, no MonsterStatus.
func newCrashEffect(prop float64) effect.Model {
	rm := withRect(effect.RestModel{
		MobCount: 6,
		Prop:     prop,
	})
	se, _ := effect.Extract(rm)
	return se
}

func newCtx(t *testing.T) (context.Context, tenant.Model) {
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tenant.WithContext(context.Background(), tm), tm
}

func nullLogger() *logrus.Logger {
	l := logrus.New()
	l.Out = io.Discard
	return l
}

// Reference imports so unused-import linters do not fire when individual
// tests are commented out for triage.
var (
	_ = errors.New
	_ point.Model
	_ skill2.Id
)

func TestApplyToMobs_EmptyClientList_NoOp(t *testing.T) {
	f := installFakes(t, mkCaster(1001), nil, nil, nil, nil, true)
	ctx, _ := newCtx(t)
	applyToMobs(nullLogger(), ctx, mkField(), 1001, mkInfo(uint32(skill2.PriestDoomId), 30, nil), newDoomEffect(1.0))
	if len(f.applies) != 0 || len(f.cancels) != 0 {
		t.Fatalf("seam calls = (%d apply, %d cancel), want both 0", len(f.applies), len(f.cancels))
	}
}
```

**Two helpers in this fixture do not exist yet** and must be added in
sub-step 10.2 before the file compiles:

- `packetmodel.NewSkillUsageInfoForTest(skillId uint32, level uint16, mobIds []uint32) packetmodel.SkillUsageInfo` — a tiny test-helper constructor in `libs/atlas-packet/model/`.
- `character.Model{}` is already a valid zero value, but `character.NewModelBuilder().SetId(id).Build()` must return a `character.Model` (it does today; verify with one read of `services/atlas-channel/atlas.com/channel/character/model.go`).

#### Step 10.2: Add `NewSkillUsageInfoForTest` constructor in `libs/atlas-packet`

- [ ] **Create file `libs/atlas-packet/model/skill_usage_info_testhelpers.go`.**

```go
package model

// NewSkillUsageInfoForTest constructs a SkillUsageInfo with the supplied
// fields. Only consumers in test code should use this — production code
// must populate SkillUsageInfo through the wire decoder.
func NewSkillUsageInfoForTest(skillId uint32, level uint16, affectedMobIds []uint32) SkillUsageInfo {
	return SkillUsageInfo{
		skillId:        skillId,
		skillLevel:     level,
		affectedMobIds: affectedMobIds,
	}
}
```

If the actual struct field names differ (the file `skill_usage_info.go` is
the source of truth — see `libs/atlas-packet/model/skill_usage_info.go`),
adjust the assignment to match. Example: if the field is named `mobIds`
rather than `affectedMobIds`, use `mobIds: affectedMobIds`.

- [ ] **Run.**

```
cd libs/atlas-packet && go build ./...
```

Expected: clean.

#### Step 10.3: Build the orchestration test file

- [ ] **Build.**

```
cd services/atlas-channel/atlas.com/channel && go build ./skill/handler
```

Expected: clean (no compile errors). If `monster.ReflectInfo`'s `Kind`, `Percent`, `ExpiresAt` field names differ from the fixture above, adjust the literal in `installFakes` to match `monster/status_mirror.go`.

- [ ] **Run the empty-client-list test only.**

```
cd services/atlas-channel/atlas.com/channel && go test ./skill/handler -run TestApplyToMobs_EmptyClientList_NoOp -v
```

Expected: PASS.

#### Step 10.4: Add the cap-exceeded test

Append to `common_apply_to_mobs_test.go`:

- [ ] **Add the test.**

```go
func TestApplyToMobs_OverCap_Drops_AndWarns(t *testing.T) {
	// Effect.MobCount = 6 by default in newDoomEffect; client sends 7.
	f := installFakes(t, mkCaster(1001), nil, []monster.Model{mkMob(1)}, nil, nil, true)
	ctx, _ := newCtx(t)
	applyToMobs(nullLogger(), ctx, mkField(), 1001,
		mkInfo(uint32(skill2.PriestDoomId), 30, []uint32{1, 2, 3, 4, 5, 6, 7}),
		newDoomEffect(1.0))
	if len(f.applies) != 0 {
		t.Fatalf("applies = %d, want 0 (over-cap should drop)", len(f.applies))
	}
	if len(f.cancels) != 0 {
		t.Fatalf("cancels = %d, want 0 (over-cap should drop)", len(f.cancels))
	}
}
```

- [ ] **Run.**

```
cd services/atlas-channel/atlas.com/channel && go test ./skill/handler -run TestApplyToMobs_OverCap -v
```

Expected: PASS.

#### Step 10.5: Add the no-bbox fallback test

Append:

- [ ] **Add the test.**

```go
func TestApplyToMobs_NoBbox_TrustsClient(t *testing.T) {
	// effect with all-zero LT/RB → fallback path; rect query is NOT called.
	rectCalled := false
	prevRect := rectQueryFunc
	t.Cleanup(func() { rectQueryFunc = prevRect })

	f := installFakes(t, mkCaster(1001), nil, nil, nil, nil, true)
	rectQueryFunc = func(_ *monster.Processor, _ field.Model, _, _, _, _ int16, _ uint32) ([]monster.Model, error) {
		rectCalled = true
		return nil, nil
	}

	ctx, _ := newCtx(t)
	applyToMobs(nullLogger(), ctx, mkField(), 1001,
		mkInfo(uint32(skill2.PriestDoomId), 30, []uint32{1, 2, 3}),
		newDoomEffectNoBbox(1.0))

	if rectCalled {
		t.Fatalf("rectQueryFunc called; expected fallback to skip rect query")
	}
	if len(f.applies) != 3 {
		t.Fatalf("applies = %d, want 3 (no-bbox fallback applies to client list)", len(f.applies))
	}
}
```

- [ ] **Run.**

Expected: PASS.

#### Step 10.6: Add the caster-load + rect-query failure tests

Append:

- [ ] **Add both tests.**

```go
func TestApplyToMobs_CasterLoadFails_Drops(t *testing.T) {
	f := installFakes(t, mkCaster(1001), errors.New("boom"), nil, nil, nil, true)
	ctx, _ := newCtx(t)
	applyToMobs(nullLogger(), ctx, mkField(), 1001,
		mkInfo(uint32(skill2.PriestDoomId), 30, []uint32{1, 2}),
		newDoomEffect(1.0))
	if len(f.applies) != 0 || len(f.cancels) != 0 {
		t.Fatalf("seam calls = (%d apply, %d cancel), want both 0 on caster-load fail", len(f.applies), len(f.cancels))
	}
}

func TestApplyToMobs_RectQueryFails_Drops(t *testing.T) {
	f := installFakes(t, mkCaster(1001), nil, nil, errors.New("boom"), nil, true)
	ctx, _ := newCtx(t)
	applyToMobs(nullLogger(), ctx, mkField(), 1001,
		mkInfo(uint32(skill2.PriestDoomId), 30, []uint32{1, 2}),
		newDoomEffect(1.0))
	if len(f.applies) != 0 {
		t.Fatalf("applies = %d, want 0 on rect-query fail", len(f.applies))
	}
}
```

- [ ] **Run.**

```
cd services/atlas-channel/atlas.com/channel && go test ./skill/handler -run "TestApplyToMobs_CasterLoadFails|TestApplyToMobs_RectQueryFails" -v
```

Expected: PASS.

#### Step 10.7: Add the rect-intersection test

Append:

- [ ] **Add the test.**

```go
func TestApplyToMobs_RectIntersectionApplied(t *testing.T) {
	// server returns 1, 2, 3; client lists 1, 2, 3, 99 (extra).
	// Expectation: 3 applies (in client order); 99 dropped silently from
	// the applied set (and surfaced in the warn log; we do not assert log
	// content here since the file uses a discarding logger).
	f := installFakes(t, mkCaster(1001), nil, []monster.Model{mkMob(1), mkMob(2), mkMob(3)}, nil, nil, true)
	ctx, _ := newCtx(t)
	applyToMobs(nullLogger(), ctx, mkField(), 1001,
		mkInfo(uint32(skill2.PriestDoomId), 30, []uint32{1, 2, 3, 99}),
		newDoomEffect(1.0))
	if len(f.applies) != 3 {
		t.Fatalf("applies = %d, want 3", len(f.applies))
	}
	want := []uint32{1, 2, 3}
	for i, c := range f.applies {
		if c.monsterId != want[i] {
			t.Errorf("apply[%d] = %d, want %d", i, c.monsterId, want[i])
		}
	}
}
```

- [ ] **Run.**

Expected: PASS.

#### Step 10.8: Add the kind-aware reflect tests

Append:

- [ ] **Add three reflect tests.**

```go
func TestApplyToMobs_DoomMagicReflectSkipped(t *testing.T) {
	// mob 2 has MAGICAL reflect → Doom must skip it.
	f := installFakes(t, mkCaster(1001), nil,
		[]monster.Model{mkMob(1), mkMob(2), mkMob(3)}, nil,
		map[uint32]string{2: monster2.ReflectKindMagical}, true)
	ctx, _ := newCtx(t)
	applyToMobs(nullLogger(), ctx, mkField(), 1001,
		mkInfo(uint32(skill2.PriestDoomId), 30, []uint32{1, 2, 3}),
		newDoomEffect(1.0))
	if len(f.applies) != 2 {
		t.Fatalf("applies = %d, want 2", len(f.applies))
	}
	if f.applies[0].monsterId != 1 || f.applies[1].monsterId != 3 {
		t.Errorf("applies = [%d, %d], want [1, 3]", f.applies[0].monsterId, f.applies[1].monsterId)
	}
}

func TestApplyToMobs_CrashFamily_PhysicalReflectSkipped(t *testing.T) {
	// Crusader Armor Crash → cancel branch with PHYSICAL kind.
	f := installFakes(t, mkCaster(1001), nil,
		[]monster.Model{mkMob(1), mkMob(2)}, nil,
		map[uint32]string{1: monster2.ReflectKindPhysical}, true)
	ctx, _ := newCtx(t)
	applyToMobs(nullLogger(), ctx, mkField(), 1001,
		mkInfo(uint32(skill2.CrusaderArmorCrashId), 30, []uint32{1, 2}),
		newCrashEffect(1.0))
	if len(f.applies) != 0 {
		t.Fatalf("applies = %d, want 0 (cancel branch only)", len(f.applies))
	}
	if len(f.cancels) != 1 || f.cancels[0].monsterId != 2 {
		t.Errorf("cancels = %v, want exactly mob 2", f.cancels)
	}
}

func TestApplyToMobs_PriestDispel_MagicalReflectSkipped(t *testing.T) {
	f := installFakes(t, mkCaster(1001), nil,
		[]monster.Model{mkMob(1), mkMob(2)}, nil,
		map[uint32]string{2: monster2.ReflectKindMagical}, true)
	ctx, _ := newCtx(t)
	applyToMobs(nullLogger(), ctx, mkField(), 1001,
		mkInfo(uint32(skill2.PriestDispelId), 30, []uint32{1, 2}),
		newCrashEffect(1.0))
	if len(f.applies) != 0 {
		t.Fatalf("applies = %d, want 0 (cancel branch)", len(f.applies))
	}
	if len(f.cancels) != 1 || f.cancels[0].monsterId != 1 {
		t.Errorf("cancels = %v, want exactly mob 1", f.cancels)
	}
}
```

- [ ] **Run.**

```
cd services/atlas-channel/atlas.com/channel && go test ./skill/handler -run "TestApplyToMobs_DoomMagicReflectSkipped|TestApplyToMobs_CrashFamily|TestApplyToMobs_PriestDispel" -v
```

Expected: PASS.

#### Step 10.9: Add the prop tests

Append:

- [ ] **Add the prop tests.**

```go
func TestApplyToMobs_PropZero_AppliesNothing(t *testing.T) {
	// propRollFunc is set to "false"; with prop=0, the effect contract is
	// "always skip". applies should be empty.
	f := installFakes(t, mkCaster(1001), nil,
		[]monster.Model{mkMob(1), mkMob(2)}, nil, nil, false)
	ctx, _ := newCtx(t)
	applyToMobs(nullLogger(), ctx, mkField(), 1001,
		mkInfo(uint32(skill2.PriestDoomId), 30, []uint32{1, 2}),
		newDoomEffect(0.0))
	if len(f.applies) != 0 {
		t.Fatalf("applies = %d, want 0 (prop=0 should skip every mob)", len(f.applies))
	}
}

func TestApplyToMobs_PropOne_AppliesAll(t *testing.T) {
	// prop=1 with propRollFunc="true" should apply every in-rect mob.
	f := installFakes(t, mkCaster(1001), nil,
		[]monster.Model{mkMob(1), mkMob(2), mkMob(3)}, nil, nil, true)
	ctx, _ := newCtx(t)
	applyToMobs(nullLogger(), ctx, mkField(), 1001,
		mkInfo(uint32(skill2.PriestDoomId), 30, []uint32{1, 2, 3}),
		newDoomEffect(1.0))
	if len(f.applies) != 3 {
		t.Fatalf("applies = %d, want 3 (prop=1 should pass all)", len(f.applies))
	}
}
```

- [ ] **Run.**

Expected: PASS.

#### Step 10.10: Add the branch-mutex tests

Append:

- [ ] **Add the apply-branch and cancel-branch tests.**

```go
func TestApplyToMobs_DoomTakesApplyBranch(t *testing.T) {
	f := installFakes(t, mkCaster(1001), nil,
		[]monster.Model{mkMob(1)}, nil, nil, true)
	ctx, _ := newCtx(t)
	applyToMobs(nullLogger(), ctx, mkField(), 1001,
		mkInfo(uint32(skill2.PriestDoomId), 30, []uint32{1}),
		newDoomEffect(1.0))
	if len(f.applies) != 1 {
		t.Fatalf("applies = %d, want 1", len(f.applies))
	}
	if len(f.cancels) != 0 {
		t.Fatalf("cancels = %d, want 0 (Doom must not take cancel branch)", len(f.cancels))
	}
}

func TestApplyToMobs_CrashTakesCancelBranch(t *testing.T) {
	f := installFakes(t, mkCaster(1001), nil,
		[]monster.Model{mkMob(1)}, nil, nil, true)
	ctx, _ := newCtx(t)
	applyToMobs(nullLogger(), ctx, mkField(), 1001,
		mkInfo(uint32(skill2.CrusaderArmorCrashId), 30, []uint32{1}),
		newCrashEffect(1.0))
	if len(f.cancels) != 1 {
		t.Fatalf("cancels = %d, want 1", len(f.cancels))
	}
	if f.cancels[0].class != "PHYSICAL" {
		t.Errorf("cancel class = %q, want PHYSICAL", f.cancels[0].class)
	}
	if len(f.applies) != 0 {
		t.Fatalf("applies = %d, want 0 (Crash must not take apply branch)", len(f.applies))
	}
}
```

- [ ] **Run.**

Expected: PASS.

#### Step 10.11: Add the prop carve-out E2E test

Append:

- [ ] **Add the carve-out test.**

```go
func TestApplyToMobs_PropCarveOutSuppressesPropOnCancel(t *testing.T) {
	// Install a deny entry for Crusader Armor Crash on the cancel branch.
	// With propRollFunc="false" the cast would normally produce zero
	// cancels; the carve-out flips that to "always pass".
	id := skill2.CrusaderArmorCrashId
	prev := propCarveOut[id]
	propCarveOut[id] = map[propBranch]bool{propBranchCancel: false}
	t.Cleanup(func() {
		if prev == nil {
			delete(propCarveOut, id)
		} else {
			propCarveOut[id] = prev
		}
	})

	f := installFakes(t, mkCaster(1001), nil,
		[]monster.Model{mkMob(1), mkMob(2)}, nil, nil, false /* propWillFire */)
	ctx, _ := newCtx(t)
	applyToMobs(nullLogger(), ctx, mkField(), 1001,
		mkInfo(uint32(id), 30, []uint32{1, 2}),
		newCrashEffect(0.0)) // prop=0 would force-skip if rolled
	if len(f.cancels) != 2 {
		t.Fatalf("cancels = %d, want 2 (carve-out should bypass prop)", len(f.cancels))
	}
}
```

- [ ] **Run.**

Expected: PASS.

#### Step 10.12: Add the status+duration verification test (replaces deleted Doom test)

Append:

- [ ] **Add the test.**

```go
func TestApplyToMobs_PassesDoomStatusAndDuration(t *testing.T) {
	f := installFakes(t, mkCaster(1001), nil,
		[]monster.Model{mkMob(99)}, nil, nil, true)
	ctx, _ := newCtx(t)
	applyToMobs(nullLogger(), ctx, mkField(), 1001,
		mkInfo(uint32(skill2.PriestDoomId), 30, []uint32{99}),
		newDoomEffect(1.0))
	if len(f.applies) != 1 {
		t.Fatalf("applies = %d, want 1", len(f.applies))
	}
	got := f.applies[0]
	if got.statuses[monster2.StatusDoom] != 1 {
		t.Errorf("statuses[DOOM] = %d, want 1", got.statuses[monster2.StatusDoom])
	}
	if got.duration != 60000 {
		t.Errorf("duration = %d, want 60000", got.duration)
	}
	if got.skillId != uint32(skill2.PriestDoomId) {
		t.Errorf("skillId = %d, want %d", got.skillId, uint32(skill2.PriestDoomId))
	}
}
```

- [ ] **Run.**

Expected: PASS.

#### Step 10.13: Run the full handler test suite

- [ ] **Run.**

```
cd services/atlas-channel/atlas.com/channel && go test ./skill/handler/... -v
```

Expected: every test passes (existing + new). The deleted-handler tests in
`skill/handler/doom/` still pass at this point — they remain installed.

#### Step 10.14: Commit

- [ ] **Commit.**

```
git add libs/atlas-packet/model/skill_usage_info_testhelpers.go \
        services/atlas-channel/atlas.com/channel/skill/handler/common_apply_to_mobs_test.go
git commit -m "test(channel/handler): orchestration coverage for applyToMobs"
```

---

### Task 11: Delete the `doom/` subpackage

**Files:**
- Delete: `services/atlas-channel/atlas.com/channel/skill/handler/doom/doom.go`
- Delete: `services/atlas-channel/atlas.com/channel/skill/handler/doom/bbox.go`
- Delete: `services/atlas-channel/atlas.com/channel/skill/handler/doom/doom_test.go`
- Delete: `services/atlas-channel/atlas.com/channel/skill/handler/doom/bbox_test.go`

#### Step 11.1: Remove the directory

- [ ] **Delete the doom subpackage in entirety.**

```
rm -rf services/atlas-channel/atlas.com/channel/skill/handler/doom
```

#### Step 11.2: Verify the directory is gone

- [ ] **Verify.**

```
test ! -e services/atlas-channel/atlas.com/channel/skill/handler/doom && echo OK
```

Expected: `OK`.

#### Step 11.3: Build to confirm we have not broken anything except registrations.go

The only remaining reference to `atlas-channel/skill/handler/doom` is the
blank import in `registrations.go`, which we drop in Task 12. Until that
lands, the build will fail with an `imported and not used` or `cannot find
package` error pointing at the registrations file. This is expected.

- [ ] **Build (expected to fail at registrations.go only).**

```
cd services/atlas-channel/atlas.com/channel && go build ./... ; true
```

Expected: an error mentioning `atlas-channel/skill/handler/doom`. Note the
exact error text in your scratch buffer — Task 12.4 verifies the same line
no longer appears after the registrations.go edit.

No commit yet — combined with Task 12 in a single commit so HEAD is never red.

---

### Task 12: Drop the doom blank-import in `registrations.go`

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go:7`

#### Step 12.1: Edit the file

- [ ] **Replace the file body with:**

```go
// Package registrations exists solely to drive init() registration of
// per-skill handler subpackages. main.go blank-imports this package;
// each new handler subpackage is added below as a blank import.
package registrations

import (
	_ "atlas-channel/skill/handler/heal" // Cleric Heal — task 045
)
```

The doom comment ("Priest Doom — task 047") is removed along with its
import line. The `heal` import remains.

#### Step 12.2: Build

- [ ] **Build the whole atlas-channel module.**

```
cd services/atlas-channel/atlas.com/channel && go build ./...
```

Expected: clean.

#### Step 12.3: Run the full atlas-channel test suite

- [ ] **Run.**

```
cd services/atlas-channel/atlas.com/channel && go test ./...
```

Expected: PASS (handler tests + everything else).

#### Step 12.4: Confirm no residual references to the doom package

- [ ] **Grep.**

```
grep -rn '"atlas-channel/skill/handler/doom"' services/ libs/ docs/ 2>/dev/null
```

Expected: no matches in `services/` or `libs/` (matches under `docs/` are
fine — those are PRD/design references, not imports).

#### Step 12.5: Commit (pairs with Task 11 deletions)

- [ ] **Commit (deletions + registrations edit together).**

```
git add -A services/atlas-channel/atlas.com/channel/skill/handler/doom \
        services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go
git commit -m "refactor(channel/handler): remove per-skill Doom handler"
```

`git add -A <path>` correctly stages the deletions even though the
directory is gone.

---

### Task 13: Final cross-package verification

#### Step 13.1: Build every Go module the change touches

- [ ] **Build atlas-packet.**

```
cd libs/atlas-packet && go build ./... && go test ./...
```

Expected: PASS.

- [ ] **Build atlas-channel.**

```
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...
```

Expected: PASS.

- [ ] **Run any other downstream consumer of `libs/atlas-packet` to confirm the new test helper does not break them. Glob to find consumers:**

```
grep -rln 'libs/atlas-packet' services/*/atlas.com/*/go.mod 2>/dev/null
```

For every file printed, run:

```
cd <containing-service-dir> && go build ./...
```

Expected: clean. (`NewSkillUsageInfoForTest` is additive and lives in `_test.go`-style code paths only via the file name; if the file is `*_testhelpers.go` it ships in the regular package — that is fine, it is just an extra exported function.)

#### Step 13.2: Confirm the dual-apply is gone via grep

- [ ] **Grep for any leftover `applyStatusFunc` outside the handler package.**

```
grep -rln "applyStatusFunc\|cancelStatusFunc" services/atlas-channel/atlas.com/channel/ 2>/dev/null
```

Expected: matches are limited to
`services/atlas-channel/atlas.com/channel/skill/handler/common.go` and
`common_apply_to_mobs_test.go`.

#### Step 13.3: Final summary commit (only if working tree has uncommitted changes)

If the previous tasks left no uncommitted changes, skip this step.

```
git status
```

If any modifications remain (e.g., import-cleanup left over from build fixes), commit them with:

```
git commit -am "chore(channel/handler): post-consolidation tidy-up"
```

#### Step 13.4: Acceptance checklist (copy from PRD §10)

- [ ] `services/atlas-channel/atlas.com/channel/skill/handler/doom/` no longer exists on the task branch.
- [ ] `_ "atlas-channel/skill/handler/doom"` line removed from `registrations.go`; `_ "atlas-channel/skill/handler/heal"` line remains.
- [ ] `applyToMobs` performs rect verification using the caster-relative bbox formula from the deleted `doom/bbox.go`.
- [ ] `applyToMobs` enforces the `e.MobCount()` cap by dropping the cast and emitting the FR-4.7.2 warn log.
- [ ] `applyToMobs` rolls `e.Prop()` per target with per-skill carve-out support.
- [ ] `applyToMobs` skips reflect-active mobs by classified kind (Doom → MAGICAL, Crash family → PHYSICAL, Priest Dispel → MAGICAL).
- [ ] `applyToMobs` emits the FR-4.7.1 warn log once per cast when client mob list is not contained by the server query, and proceeds with the intersection.
- [ ] `applyToMobs` emits the FR-4.8 debug summary on every cast that reaches the iteration step.
- [ ] Unit tests for rect math, intersection, cap, prop, reflect kind, and apply-vs-cancel are present in `mob_select_test.go` + `common_apply_to_mobs_test.go`.
- [ ] `go build ./...` and `go test ./...` succeed in `services/atlas-channel/atlas.com/channel/`.

The two manual-cast acceptance criteria from PRD §10 (3-mob log line count;
8-mob over-cap reject; out-of-rect anomaly log) are post-merge runtime
verifications and are not part of this plan's checklist.

---

## Plan Self-Review

**Spec coverage** — every PRD FR-4.x has a corresponding step:

- FR-4.1 rect verification → Task 8 step 8.1 (intersection branch).
- FR-4.2 no-bbox fallback → Task 3 (`hasEffectBbox`) + Task 8 (fallback branch).
- FR-4.3 mobCount cap → Task 8 step 8.1 (`uint32(len(mobIds)) > cap` block).
- FR-4.4 client-order preservation → Task 4 (`intersectMobIds` keeps client order).
- FR-4.5 prop roll + carve-out → Task 6 (helpers) + Task 8 (`propAppliesTo` + `propRollFunc` calls).
- FR-4.6 kind-aware reflect skip → Task 5 (`mobBuffApplyKind`) + Task 8 (kind selection block).
- FR-4.7.1 anomaly out-of-rect warn → Task 8 (`anomaly` log block).
- FR-4.7.2 over-cap warn → Task 8 (cap log block).
- FR-4.7.3 "no other anomaly cases warn" → Task 8 (only those two warn paths exist).
- FR-4.8 per-cast summary → Task 8 (`buildSummaryFields` + Debug emit).
- FR-4.9 emit branch mutual exclusion → Task 8 (`isCancel` branch selection at branch-selection step).
- FR-4.10 per-skill handler removal → Tasks 11 + 12.
- FR-4.11 test migration → Tasks 2–6 + Task 10.
- FR-4.12 backwards compatibility — verified by passing tests in Task 13 + manual cast post-merge.

**Placeholder scan** — no "TODO", no "implement later", no "similar to". Every step has actual code. The `cap` shadow note is explicitly flagged. All file paths are absolute under the worktree.

**Type consistency** — every signature is consistent across tasks:
- `loadCasterFunc(cp character.Processor, id uint32) (character.Model, error)` is defined once (Task 7) and called once (Task 8).
- `rectQueryFunc(p *monster.Processor, f field.Model, x1,y1,x2,y2 int16, limit uint32) ([]monster.Model, error)` defined once, called once.
- `applyStatusFunc` and `cancelStatusFunc` parameter lists match `monster.Processor.ApplyStatus` and `CancelStatus` exactly (verified against `monster/processor.go:83,96`).
- `propBranch` enum + `propAppliesTo` signature consistent in Task 6 and Task 8 / Task 10.

---

## Execution Handoff

**Plan complete and saved to `docs/tasks/task-057-monster-buff-trust-verify/plan.md`.**

Recommended next step: `/clear`, then `/execute-task task-057` (which reuses this worktree and dispatches subagents per task).
