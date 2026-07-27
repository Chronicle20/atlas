# Shadow Stars (Night Lord 4121006) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make Shadow Stars (`4121006`) behave correctly end to end — carry the client-chosen star id in the `SHADOW_CLAW` buff, stop per-attack star consumption while buffed, and charge the one-time `bulletCount` cast cost of the chosen star.

**Architecture:** Three coordinated changes in `services/atlas-channel` plus one getter in `libs/atlas-packet`. The channel skill orchestrator (`skill/handler`) validates the chosen star at cast, rewrites the `SHADOW_CLAW` statup amount to the star id (mirroring how `mount.go` injects a vehicle id into `MONSTER_RIDING`), and emits a one-time `bulletCount` reserve→consume (mirroring the projectile path). The projectile gate (`socket/handler`) gains a claw + `SHADOW_CLAW` carve-out mirroring the existing Soul Arrow carve-out for bow/crossbow. `atlas-data`'s `reader.go` is unchanged — it keeps emitting the `SHADOW_CLAW` placeholder `0`, which the channel overwrites.

**Tech Stack:** Go 1.24 workspace (`go.work`), Kafka (compartment reserve/consume commands), JSON:API REST for effect data, `atlas-constants` shared types, project Builder pattern for tests.

## Global Constraints

- **No wire-format change.** `spiritJavelinItemId` is already decoded at `libs/atlas-packet/model/skill_usage_info.go:33`; this task only adds a getter. No `Encode` method exists for `SkillUsageInfo` (serverbound decode-only).
- **No `reader.go` functional change.** `services/atlas-data/.../skill/reader.go:298` continues emitting `SHADOW_CLAW` with amount `0`.
- **No new generic batch/multi-item consume framework** (PRD §2 non-goal). The star consume is a small, self-contained function.
- **Injection technique is fixed:** rewrite the `SHADOW_CLAW` statup amount in the channel layer before `buff.Apply` — the exact precedent in `mount.go:tamedMountStatups`. No Kafka payload change.
- **Star id is `uint32`**; the `SHADOW_CLAW` statup amount is `int32` (`statup.NewModel(mask string, amount int32)`); wire-encoded as an int foreign value, so the client reads the amount directly as the star id.
- **Validation is mandatory** (PRD FR-5, design decision 2): the star id must be a throwing-star classification AND owned before it drives a buff or a consume. On failure, warn-log and **abort the whole cast** (return before HP/MP/cooldown).
- **Shortfall posture** (design decision 4): consume what's available and warn; do not reject the cast for a shortfall (the client already gates on owning ≥ the required count).
- **Build/verify gates (CLAUDE.md).** Both `libs/atlas-packet` and `services/atlas-channel` `go.mod`s are touched. Before "done": `go test -race ./...`, `go vet ./...`, `go build ./...` clean in each changed module; `docker buildx bake atlas-channel` from the worktree root; `tools/redis-key-guard.sh` clean from repo root. Run guard scripts from repo root **without** a global `GOWORK=off` prefix.
- **Test setup uses the project Builder pattern** (`asset.NewModelBuilder`, `NewSkillUsageInfoBuilder`, `buff.NewBuff`/`stat.NewStat`). No `*_testhelpers.go` files.
- **All paths below are relative to the worktree root** `.worktrees/task-158-shadow-stars/`.

---

## File Structure

- `libs/atlas-packet/model/skill_usage_info.go` — add `SpiritJavelinItemId()` getter (Task 1).
- `libs/atlas-packet/model/skill_usage_info_test.go` — add byte-fixture decode test (Task 1).
- `services/atlas-channel/atlas.com/channel/data/skill/effect/model.go` — add `BulletCount()` getter (Task 2).
- `services/atlas-channel/atlas.com/channel/data/skill/effect/model_test.go` — new; `BulletCount()` test (Task 2).
- `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_projectile.go` — extract `projectileConsumptionSkipped`, add claw+`SHADOW_CLAW` carve-out (Task 3).
- `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_projectile_test.go` — add gate tests (Task 3).
- `services/atlas-channel/atlas.com/channel/skill/handler/shadow_stars.go` — new; all Shadow Stars pure helpers + consume emit + inventory seam (Tasks 4–5).
- `services/atlas-channel/atlas.com/channel/skill/handler/shadow_stars_test.go` — new; pure-helper + orchestration-decision tests (Tasks 4–5).
- `services/atlas-channel/atlas.com/channel/skill/handler/common.go` — wire the Shadow Stars branch into `UseSkill` (Task 5).

---

### Task 1: `SkillUsageInfo.SpiritJavelinItemId()` getter (FR-1)

**Files:**
- Modify: `libs/atlas-packet/model/skill_usage_info.go` (add getter next to the existing getters at lines 53–71)
- Test: `libs/atlas-packet/model/skill_usage_info_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `func (m *SkillUsageInfo) SpiritJavelinItemId() uint32` — read by Task 5.

**Decode ordering fact (verified):** `4121006` is NOT in `isAntiRepeatBuffSkill`, `isPartyBuff`, or `isMobAffectingBuff`, so a Shadow Stars `SkillUsageInfo` decodes exactly: `updateTime` (uint32 LE), `skillId` (uint32 LE), `skillLevel` (byte), `spiritJavelinItemId` (uint32 LE) — no castX/castY, no party bitmap, no mob list. The test fixture below reflects that.

- [ ] **Step 1: Write the failing test**

Add to `libs/atlas-packet/model/skill_usage_info_test.go` (add imports `"context"`, `"encoding/binary"`, `"github.com/Chronicle20/atlas/libs/atlas-socket/request"`, `"github.com/sirupsen/logrus"`):

```go
func TestSkillUsageInfoDecodeSpiritJavelinItemId(t *testing.T) {
	const (
		skillId = uint32(4121006) // NightLordShadowStars
		starId  = uint32(2070006) // Ilbi Throwing Stars
	)
	buf := make([]byte, 0, 13)
	buf = binary.LittleEndian.AppendUint32(buf, 12345) // updateTime
	buf = binary.LittleEndian.AppendUint32(buf, skillId)
	buf = append(buf, 30)                              // skillLevel
	buf = binary.LittleEndian.AppendUint32(buf, starId)

	req := request.Request(buf)
	reader := request.NewRequestReader(&req, 0)

	var info SkillUsageInfo
	info.Decode(logrus.New(), context.Background())(&reader, map[string]interface{}{})

	if got := info.SpiritJavelinItemId(); got != starId {
		t.Fatalf("SpiritJavelinItemId() = %d, want %d", got, starId)
	}
	if reader.Available() > 0 {
		t.Fatalf("reader has %d unconsumed bytes after decode", reader.Available())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-packet && go test ./model/ -run TestSkillUsageInfoDecodeSpiritJavelinItemId -v`
Expected: FAIL — `info.SpiritJavelinItemId undefined (type SkillUsageInfo has no field or method SpiritJavelinItemId)` (compile error).

- [ ] **Step 3: Add the getter**

In `libs/atlas-packet/model/skill_usage_info.go`, immediately after the `SkillLevel()` getter (line 59):

```go
func (m *SkillUsageInfo) SpiritJavelinItemId() uint32 {
	return m.spiritJavelinItemId
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd libs/atlas-packet && go test ./model/ -run TestSkillUsageInfoDecodeSpiritJavelinItemId -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/model/skill_usage_info.go libs/atlas-packet/model/skill_usage_info_test.go
git commit -m "feat(atlas-packet): expose SkillUsageInfo.SpiritJavelinItemId (task-158 FR-1)"
```

---

### Task 2: `effect.Model.BulletCount()` getter

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/data/skill/effect/model.go` (add getter next to `BulletConsume()` at lines 105–107)
- Test: `services/atlas-channel/atlas.com/channel/data/skill/effect/model_test.go` (new)

**Interfaces:**
- Consumes: nothing.
- Produces: `func (m Model) BulletCount() uint16` — the WZ one-time cast cost (200 in reference data); read by Task 5.

**Fact:** `model.go:58` holds `bulletCount uint16`; `rest.go:59,130` already populates it from JSON `bulletCount`. Only the getter is missing.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/data/skill/effect/model_test.go`:

```go
package effect

import "testing"

func TestModelBulletCount(t *testing.T) {
	m := Model{bulletCount: 200}
	if got := m.BulletCount(); got != 200 {
		t.Fatalf("BulletCount() = %d, want 200", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./data/skill/effect/ -run TestModelBulletCount -v`
Expected: FAIL — `m.BulletCount undefined`.

- [ ] **Step 3: Add the getter**

In `services/atlas-channel/atlas.com/channel/data/skill/effect/model.go`, immediately after `BulletConsume()` (line 107):

```go
// BulletCount returns the WZ `bulletCount` attribute — the one-time star
// batch charged when casting Shadow Stars (200 in reference data). Distinct
// from BulletConsume (per-attack projectile count).
func (m Model) BulletCount() uint16 {
	return m.bulletCount
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./data/skill/effect/ -run TestModelBulletCount -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/data/skill/effect/model.go services/atlas-channel/atlas.com/channel/data/skill/effect/model_test.go
git commit -m "feat(atlas-channel): expose effect.Model.BulletCount (task-158)"
```

---

### Task 3: Projectile gate — claw + `SHADOW_CLAW` carve-out (FR-3)

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_projectile.go` (lines 107–111 → extract a pure gate function; add claw case)
- Test: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_projectile_test.go` (add `TestProjectileConsumptionSkipped`)

**Interfaces:**
- Consumes: `hasBuff(buffs []buff.Model, statType ts.TemporaryStatType) bool` (existing, `character_attack_projectile.go:195`); `ts.TemporaryStatTypeSoulArrow`, `ts.TemporaryStatTypeShadowClaw`; `item.WeaponType*` constants.
- Produces: `func projectileConsumptionSkipped(weaponType item.WeaponType, buffs []buff.Model) bool` — package-local, tested here only.

**Design note:** `Plan()` loads buffs internally via Kafka/REST (`p.bp.GetByCharacterId`), so `Plan()` itself is not unit-testable offline (consistent with today — only the pure `resolvePlan` is tested). We extract the buff-gate decision into a pure function so the FR-3 behavior AND the inactive-safe regression are covered by a unit test. Existing test helpers `buffWithStat` / `expiredBuffWithStat` (test file lines 60–68) are reused.

- [ ] **Step 1: Write the failing test**

Add to `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_projectile_test.go`:

```go
func TestProjectileConsumptionSkipped(t *testing.T) {
	soulArrow := []buff.Model{buffWithStat(ts.TemporaryStatTypeSoulArrow)}
	shadowClaw := []buff.Model{buffWithStat(ts.TemporaryStatTypeShadowClaw)}
	expiredClaw := []buff.Model{expiredBuffWithStat(ts.TemporaryStatTypeShadowClaw)}

	cases := []struct {
		name   string
		weapon item.WeaponType
		buffs  []buff.Model
		want   bool
	}{
		{"bow + soul arrow -> skip", item.WeaponTypeBow, soulArrow, true},
		{"crossbow + soul arrow -> skip", item.WeaponTypeCrossbow, soulArrow, true},
		{"claw + shadow claw -> skip", item.WeaponTypeClaw, shadowClaw, true},
		{"claw + no buff -> consume", item.WeaponTypeClaw, nil, false},
		{"claw + expired shadow claw -> consume", item.WeaponTypeClaw, expiredClaw, false},
		{"claw + soul arrow -> consume", item.WeaponTypeClaw, soulArrow, false},
		{"bow + shadow claw -> consume", item.WeaponTypeBow, shadowClaw, false},
		{"gun + shadow claw -> consume", item.WeaponTypeGun, shadowClaw, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := projectileConsumptionSkipped(tc.weapon, tc.buffs); got != tc.want {
				t.Fatalf("projectileConsumptionSkipped(%v) = %v, want %v", tc.weapon, got, tc.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run TestProjectileConsumptionSkipped -v`
Expected: FAIL — `undefined: projectileConsumptionSkipped`.

- [ ] **Step 3: Add the pure function and rewire `Plan`**

In `character_attack_projectile.go`, add near `hasBuff` (after line 207):

```go
// projectileConsumptionSkipped reports whether an active buff exempts this
// weapon type from per-attack projectile consumption: Soul Arrow for
// bow/crossbow, Shadow Stars (SHADOW_CLAW) for claw. Expired buffs are ignored
// by hasBuff, so a stale buff falls through to normal consumption.
func projectileConsumptionSkipped(weaponType item.WeaponType, buffs []buff.Model) bool {
	if (weaponType == item.WeaponTypeBow || weaponType == item.WeaponTypeCrossbow) && hasBuff(buffs, ts.TemporaryStatTypeSoulArrow) {
		return true
	}
	if weaponType == item.WeaponTypeClaw && hasBuff(buffs, ts.TemporaryStatTypeShadowClaw) {
		return true
	}
	return false
}
```

Then replace the existing Soul Arrow block in `Plan()` (lines 107–111):

```go
	if (weaponType == item.WeaponTypeBow || weaponType == item.WeaponTypeCrossbow) && hasBuff(buffs, ts.TemporaryStatTypeSoulArrow) {
		p.l.WithField("characterId", c.Id()).WithField("skillId", ai.SkillId()).
			Debugf("Skipping projectile consumption: Soul Arrow active.")
		return nil, false
	}
```

with:

```go
	if projectileConsumptionSkipped(weaponType, buffs) {
		p.l.WithField("characterId", c.Id()).WithField("skillId", ai.SkillId()).
			Debugf("Skipping projectile consumption: weapon buff active (Soul Arrow / Shadow Stars).")
		return nil, false
	}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run 'TestProjectileConsumptionSkipped|TestComputeCount|TestResolvePlan' -v`
Expected: PASS (new gate test plus existing projectile tests still green).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_attack_projectile.go services/atlas-channel/atlas.com/channel/socket/handler/character_attack_projectile_test.go
git commit -m "feat(atlas-channel): skip claw projectile consume under Shadow Stars (task-158 FR-3)"
```

---

### Task 4: Shadow Stars pure helpers (FR-2, FR-4 plan, FR-5)

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/skill/handler/shadow_stars.go`
- Test: `services/atlas-channel/atlas.com/channel/skill/handler/shadow_stars_test.go` (create)

**Interfaces:**
- Consumes: `asset.Model` (`.TemplateId() uint32`, `.Slot() int16`, `.Quantity() uint32`); `statup.Model` (`statup.NewModel(mask string, amount int32)`, `.Mask() string`, `.Amount() int32`); `item.GetClassification`, `item.ClassificationConsumableThrowingStar`; `ts.TemporaryStatTypeShadowClaw` (alias `charconst`/`ts` for `atlas-constants/character`).
- Produces (all package-local, read by Task 5):
  - `type StarDraw struct { Slot int16; ItemId uint32; Quantity int16 }`
  - `func validateShadowStar(assets []asset.Model, starItemId uint32) bool`
  - `func resolveStarConsume(assets []asset.Model, starItemId uint32, count int) (draws []StarDraw, available int)`
  - `func rewriteShadowClawStatups(statups []statup.Model, starItemId uint32) []statup.Model`
  - `func resolveShadowStarsCast(assets []asset.Model, statups []statup.Model, starItemId uint32, bulletCount int) (rewritten []statup.Model, draws []StarDraw, shortfall bool, ok bool)`

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-channel/atlas.com/channel/skill/handler/shadow_stars_test.go`:

```go
package handler

import (
	"testing"

	"atlas-channel/asset"
	"atlas-channel/data/skill/effect/statup"

	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/google/uuid"
)

const (
	starIlbi   uint32 = 2070006 // Ilbi Throwing Stars (classification 207)
	starSubi   uint32 = 2070000 // Subi Throwing Stars (classification 207)
	notAStar   uint32 = 2000000 // Red Potion (classification 200)
)

func starAsset(slot int16, templateId uint32, qty uint32) asset.Model {
	return asset.NewModelBuilder(1, uuid.New(), templateId).
		SetSlot(slot).
		SetQuantity(qty).
		MustBuild()
}

func TestValidateShadowStar(t *testing.T) {
	assets := []asset.Model{starAsset(1, starIlbi, 50)}
	if !validateShadowStar(assets, starIlbi) {
		t.Fatalf("owned throwing star should validate")
	}
	if validateShadowStar(assets, starSubi) {
		t.Fatalf("unowned throwing star should not validate")
	}
	if validateShadowStar(assets, notAStar) {
		t.Fatalf("non-throwing-star id should not validate")
	}
	if validateShadowStar([]asset.Model{starAsset(1, starIlbi, 0)}, starIlbi) {
		t.Fatalf("zero-quantity star should not validate")
	}
}

func TestResolveStarConsume_SingleSlot(t *testing.T) {
	assets := []asset.Model{starAsset(1, starIlbi, 200), starAsset(2, starSubi, 200)}
	draws, available := resolveStarConsume(assets, starIlbi, 200)
	if available != 200 {
		t.Fatalf("available = %d, want 200", available)
	}
	if len(draws) != 1 || draws[0].Slot != 1 || draws[0].ItemId != starIlbi || draws[0].Quantity != 200 {
		t.Fatalf("draws = %+v, want single slot-1 draw of 200 Ilbi", draws)
	}
}

func TestResolveStarConsume_MultiSlotAndShortfall(t *testing.T) {
	// 120 across two Ilbi slots; a Subi slot must be ignored.
	assets := []asset.Model{starAsset(1, starIlbi, 80), starAsset(2, starSubi, 200), starAsset(3, starIlbi, 40)}
	draws, available := resolveStarConsume(assets, starIlbi, 200)
	if available != 120 {
		t.Fatalf("available = %d, want 120 (shortfall)", available)
	}
	total := 0
	for _, d := range draws {
		if d.ItemId != starIlbi {
			t.Fatalf("draw targeted wrong item %d, want %d", d.ItemId, starIlbi)
		}
		total += int(d.Quantity)
	}
	if total != 120 {
		t.Fatalf("drawn total = %d, want 120", total)
	}
}

func TestRewriteShadowClawStatups(t *testing.T) {
	in := []statup.Model{
		statup.NewModel(string(charconst.TemporaryStatTypeShadowClaw), 0),
		statup.NewModel(string(charconst.TemporaryStatTypeShadowPartner), 5),
	}
	out := rewriteShadowClawStatups(in, starIlbi)
	var sawClaw, sawPartner bool
	for _, su := range out {
		switch su.Mask() {
		case string(charconst.TemporaryStatTypeShadowClaw):
			sawClaw = true
			if su.Amount() != int32(starIlbi) {
				t.Fatalf("SHADOW_CLAW amount = %d, want %d", su.Amount(), starIlbi)
			}
		case string(charconst.TemporaryStatTypeShadowPartner):
			sawPartner = true
			if su.Amount() != 5 {
				t.Fatalf("non-SHADOW_CLAW statup mutated: amount = %d, want 5", su.Amount())
			}
		}
	}
	if !sawClaw || !sawPartner {
		t.Fatalf("expected both statups preserved; sawClaw=%v sawPartner=%v", sawClaw, sawPartner)
	}
}

func TestResolveShadowStarsCast(t *testing.T) {
	statups := []statup.Model{statup.NewModel(string(charconst.TemporaryStatTypeShadowClaw), 0)}

	// Invalid star -> abort, no draws, no rewrite.
	if _, draws, _, ok := resolveShadowStarsCast(nil, statups, starIlbi, 200); ok || len(draws) != 0 {
		t.Fatalf("unowned star: ok=%v draws=%d, want ok=false and no draws", ok, len(draws))
	}

	// Valid star -> SHADOW_CLAW carries star id, draws total bulletCount, no shortfall.
	assets := []asset.Model{starAsset(1, starIlbi, 200)}
	rewritten, draws, shortfall, ok := resolveShadowStarsCast(assets, statups, starIlbi, 200)
	if !ok || shortfall {
		t.Fatalf("valid star: ok=%v shortfall=%v, want ok=true shortfall=false", ok, shortfall)
	}
	if len(rewritten) != 1 || rewritten[0].Amount() != int32(starIlbi) {
		t.Fatalf("rewritten SHADOW_CLAW amount = %+v, want %d", rewritten, starIlbi)
	}
	total := 0
	for _, d := range draws {
		total += int(d.Quantity)
	}
	if total != 200 {
		t.Fatalf("drawn total = %d, want 200", total)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./skill/handler/ -run 'ShadowStar|ValidateShadowStar|ResolveStarConsume|RewriteShadowClaw|ResolveShadowStarsCast' -v`
Expected: FAIL — undefined `validateShadowStar` / `resolveStarConsume` / `rewriteShadowClawStatups` / `resolveShadowStarsCast` / `StarDraw`.

- [ ] **Step 3: Implement the pure helpers**

Create `services/atlas-channel/atlas.com/channel/skill/handler/shadow_stars.go` (consume-emit + seam added in Task 5; this step is the pure core only):

```go
package handler

import (
	"sort"

	"atlas-channel/asset"
	"atlas-channel/data/skill/effect/statup"

	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

// StarDraw is one slot-level consume of a chosen throwing star for the
// Shadow Stars cast cost.
type StarDraw struct {
	Slot     int16
	ItemId   uint32
	Quantity int16
}

// validateShadowStar reports whether starItemId is a throwing-star
// classification AND present (quantity > 0) in the caster's consumable assets.
func validateShadowStar(assets []asset.Model, starItemId uint32) bool {
	if item.GetClassification(item.Id(starItemId)) != item.ClassificationConsumableThrowingStar {
		return false
	}
	for _, a := range assets {
		if a.TemplateId() == starItemId && a.Quantity() > 0 {
			return true
		}
	}
	return false
}

// resolveStarConsume draws `count` of exactly starItemId across ascending
// consumable slots. `available` is the sum of planned draws
// (min(count, total owned)); available < count signals a shortfall.
func resolveStarConsume(assets []asset.Model, starItemId uint32, count int) (draws []StarDraw, available int) {
	matching := make([]asset.Model, 0, len(assets))
	for _, a := range assets {
		if a.TemplateId() == starItemId && a.Quantity() > 0 {
			matching = append(matching, a)
		}
	}
	if len(matching) == 0 || count <= 0 {
		return nil, 0
	}
	sort.Slice(matching, func(i, j int) bool { return matching[i].Slot() < matching[j].Slot() })

	remaining := count
	draws = make([]StarDraw, 0, len(matching))
	for _, a := range matching {
		if remaining <= 0 {
			break
		}
		draw := int(a.Quantity())
		if draw > remaining {
			draw = remaining
		}
		draws = append(draws, StarDraw{Slot: a.Slot(), ItemId: starItemId, Quantity: int16(draw)})
		remaining -= draw
		available += draw
	}
	return draws, available
}

// rewriteShadowClawStatups returns a copy of statups with the SHADOW_CLAW
// entry's amount set to starItemId. Non-SHADOW_CLAW statups pass through
// unchanged. Mirrors mount.go's tamedMountStatups for MONSTER_RIDING.
func rewriteShadowClawStatups(statups []statup.Model, starItemId uint32) []statup.Model {
	out := make([]statup.Model, 0, len(statups))
	for _, su := range statups {
		if su.Mask() == string(charconst.TemporaryStatTypeShadowClaw) {
			out = append(out, statup.NewModel(su.Mask(), int32(starItemId)))
			continue
		}
		out = append(out, su)
	}
	return out
}

// resolveShadowStarsCast validates the chosen star and resolves the buff
// statups + consume draws for a Shadow Stars cast. ok=false means the star is
// invalid (wrong classification or not owned) and the cast MUST abort — the
// returned rewritten/draws are nil. shortfall reports available < bulletCount.
func resolveShadowStarsCast(assets []asset.Model, statups []statup.Model, starItemId uint32, bulletCount int) (rewritten []statup.Model, draws []StarDraw, shortfall bool, ok bool) {
	if !validateShadowStar(assets, starItemId) {
		return nil, nil, false, false
	}
	draws, available := resolveStarConsume(assets, starItemId, bulletCount)
	rewritten = rewriteShadowClawStatups(statups, starItemId)
	return rewritten, draws, available < bulletCount, true
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./skill/handler/ -run 'ShadowStar|ValidateShadowStar|ResolveStarConsume|RewriteShadowClaw|ResolveShadowStarsCast' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/skill/handler/shadow_stars.go services/atlas-channel/atlas.com/channel/skill/handler/shadow_stars_test.go
git commit -m "feat(atlas-channel): Shadow Stars star validation/consume-plan/statup rewrite (task-158 FR-2/4/5)"
```

---

### Task 5: Consume emit + wire Shadow Stars into `UseSkill` (FR-2, FR-4, FR-5 orchestration)

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/skill/handler/shadow_stars.go` (add the inventory seam and consume emit)
- Modify: `services/atlas-channel/atlas.com/channel/skill/handler/common.go` (`UseSkill` wiring)

**Interfaces:**
- Consumes (from Task 4): `StarDraw`, `resolveShadowStarsCast`. From `common.go`: `character.Processor`, `character.NewProcessor`. From the projectile path (pattern reference): `compartment.NewProcessor`, `compartmentMsg.EnvEventTopicStatus`, `topic.EnvProvider`, `once.ReservationValidator`, `consumer.GetManager().RegisterHandler`, `message.AdaptHandler`/`message.OneTimeConfig`, `compartmentMsg.ItemBody`, `inventory.TypeValueUse`.
- Produces:
  - `var loadCasterInventoryFunc = func(cp character.Processor, characterId uint32) ([]asset.Model, error)` — a package-level seam (mirrors the existing `loadCasterFunc` at `common.go:31`).
  - `func emitStarConsume(l logrus.FieldLogger, ctx context.Context, characterId uint32, draws []StarDraw) error`

**Wiring note:** Shadow Stars is NOT a mount and NOT mob-affecting, so it flows through the generic path. Only three things change for `skillId == NightLordShadowStarsId`: (1) a pre-flight abort at the top of `UseSkill`; (2) the statups handed to `buff.Apply` are the rewritten set; (3) the star consume is emitted after `buff.Apply`. HP/MP/cooldown remain generic. The pre-flight runs BEFORE HP/MP so an invalid star burns nothing.

- [ ] **Step 1: Add the inventory seam and consume emit to `shadow_stars.go`**

Append to `services/atlas-channel/atlas.com/channel/skill/handler/shadow_stars.go`, and extend the import block to add: `"context"`, `"atlas-channel/character"`, `"atlas-channel/compartment"`, `compartmentMsg "atlas-channel/kafka/message/compartment"`, `once "atlas-channel/kafka/once/compartment"`, `"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"`, `"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"`, `"github.com/Chronicle20/atlas/libs/atlas-kafka/message"`, `"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"`, `"github.com/google/uuid"`, `"github.com/sirupsen/logrus"`:

```go
// loadCasterInventoryFunc is the caster-inventory load seam tests can replace.
// Production loads the character with the inventory decorator and returns the
// consumable (USE) compartment assets — the same decorated load the generic
// item-consume block in UseSkill uses.
var loadCasterInventoryFunc = func(cp character.Processor, characterId uint32) ([]asset.Model, error) {
	c, err := cp.GetById(cp.InventoryDecorator)(characterId)
	if err != nil {
		return nil, err
	}
	return c.Inventory().Consumable().Assets(), nil
}

// emitStarConsume charges the Shadow Stars cast cost by reserving then consuming
// each StarDraw from the USE compartment. Mirrors the projectile Emit path:
// register a one-time reservation-observed handler that issues the consume, then
// request the reservation. Reservation atomicity means a slot that no longer
// holds the item fails cleanly without over-consuming.
func emitStarConsume(l logrus.FieldLogger, ctx context.Context, characterId uint32, draws []StarDraw) error {
	if len(draws) == 0 {
		return nil
	}
	cpp := compartment.NewProcessor(l, ctx)
	t, err := topic.EnvProvider(l)(compartmentMsg.EnvEventTopicStatus)()
	if err != nil {
		return err
	}
	for _, draw := range draws {
		draw := draw
		txId := uuid.New()
		validator := once.ReservationValidator(txId, draw.ItemId)
		handler := reservedStarToConsume(l, cpp, characterId, txId, inventory.TypeValueUse, draw.Slot)
		if _, rerr := consumer.GetManager().RegisterHandler(t, message.AdaptHandler(message.OneTimeConfig(validator, handler))); rerr != nil {
			l.WithError(rerr).WithField("characterId", characterId).
				Errorf("Unable to register one-time consume handler for Shadow Stars reservation.")
			continue
		}
		reserves := []compartmentMsg.ItemBody{{Source: draw.Slot, ItemId: draw.ItemId, Quantity: draw.Quantity}}
		if rerr := cpp.RequestReserve(txId, characterId, inventory.TypeValueUse, reserves); rerr != nil {
			l.WithError(rerr).WithField("characterId", characterId).WithField("slot", draw.Slot).
				Errorf("Unable to emit Shadow Stars reservation request.")
		}
	}
	return nil
}

func reservedStarToConsume(l logrus.FieldLogger, cpp compartment.Processor, characterId uint32, txId uuid.UUID, invType inventory.Type, slot int16) message.Handler[compartmentMsg.StatusEvent[compartmentMsg.ReservedEventBody]] {
	return func(_ logrus.FieldLogger, _ context.Context, _ compartmentMsg.StatusEvent[compartmentMsg.ReservedEventBody]) {
		if err := cpp.Consume(txId, characterId, invType, slot); err != nil {
			l.WithError(err).WithField("characterId", characterId).WithField("slot", slot).
				Errorf("Unable to emit Shadow Stars consume command.")
		}
	}
}
```

- [ ] **Step 2: Verify the package still builds (no behavior change yet)**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./skill/handler/`
Expected: builds clean. (`loadCasterInventoryFunc`/`emitStarConsume` are unused until Step 3 — Go allows unused package-level funcs/vars, so this compiles.)

- [ ] **Step 3: Wire the Shadow Stars branch into `UseSkill`**

In `services/atlas-channel/atlas.com/channel/skill/handler/common.go`, inside the innermost `UseSkill` closure, **at the very top of the function body** (before the `e.HPConsume()` check at line 73), insert the pre-flight and hoist the statups variable:

```go
				// Shadow Stars pre-flight (FR-5): validate the client-chosen star
				// and resolve the cast cost BEFORE any HP/MP/cooldown spend. A bogus
				// or unowned star aborts the whole cast — no MP, no cooldown, no buff,
				// no consume — so a crafted client cannot inject an id into the buff
				// or trigger consumption of an unintended item.
				statupsToApply := e.StatUps()
				var shadowStarDraws []StarDraw
				if skill2.Id(info.SkillId()) == skill2.NightLordShadowStarsId {
					assets, invErr := loadCasterInventoryFunc(character.NewProcessor(l, ctx), characterId)
					if invErr != nil {
						l.WithError(invErr).Warnf("Character [%d] cast Shadow Stars [%d] but inventory load failed; aborting cast.", characterId, info.SkillId())
						return nil
					}
					rewritten, draws, shortfall, ok := resolveShadowStarsCast(assets, e.StatUps(), info.SpiritJavelinItemId(), int(e.BulletCount()))
					if !ok {
						l.Warnf("Character [%d] cast Shadow Stars [%d] with invalid star [%d] (not a throwing star or not owned); aborting cast.", characterId, info.SkillId(), info.SpiritJavelinItemId())
						return nil
					}
					if shortfall {
						l.Warnf("Character [%d] cast Shadow Stars [%d]: insufficient star [%d] for cast cost [%d]; consuming what's available.", characterId, info.SkillId(), info.SpiritJavelinItemId(), e.BulletCount())
					}
					statupsToApply = rewritten
					shadowStarDraws = draws
				}
```

Then change the generic buff-apply block (lines 107–111) to use `statupsToApply` and emit the consume after apply:

```go
				if e.Duration() > 0 && len(statupsToApply) > 0 {
					applyBuffFunc := buff.NewProcessor(l, ctx).Apply(f, characterId, int32(info.SkillId()), info.SkillLevel(), e.Duration(), statupsToApply)
					_ = applyBuffFunc(characterId)
					_ = applyToParty(l)(ctx)(f, characterId, info.AffectedPartyMemberBitmap())(applyBuffFunc)
				}

				// Shadow Stars cast cost (FR-4): charge bulletCount of the chosen
				// star after the buff is applied. shadowStarDraws is empty for every
				// other skill.
				if len(shadowStarDraws) > 0 {
					if err := emitStarConsume(l, ctx, characterId, shadowStarDraws); err != nil {
						l.WithError(err).Errorf("Character [%d] Shadow Stars cast-cost consume failed.", characterId)
					}
				}
```

> Note: the existing `skillId := skill2.Id(info.SkillId())` declaration at `common.go:99` (used by the mount check) stays as-is; the pre-flight uses `skill2.Id(info.SkillId())` inline to avoid reordering the mount block. Do not introduce a second `skillId` variable in the same scope.

- [ ] **Step 4: Run the full skill/handler + socket/handler test packages**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./skill/handler/ ./socket/handler/ ./data/skill/effect/ -v`
Expected: PASS — all new tests plus every existing test in those packages (mount, mob-select, recipients, projectile) stay green.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/skill/handler/shadow_stars.go services/atlas-channel/atlas.com/channel/skill/handler/common.go
git commit -m "feat(atlas-channel): wire Shadow Stars cast — SHADOW_CLAW star id + cast cost (task-158 FR-2/4/5)"
```

---

### Task 6: Full verification (CLAUDE.md build gates)

**Files:** none (verification only).

**Interfaces:** none.

- [ ] **Step 1: `libs/atlas-packet` module clean**

Run:
```bash
cd libs/atlas-packet && go test -race ./... && go vet ./... && go build ./...
```
Expected: all pass, no output failures.

- [ ] **Step 2: `atlas-channel` module clean**

Run:
```bash
cd services/atlas-channel/atlas.com/channel && go test -race ./... && go vet ./... && go build ./...
```
Expected: all pass.

- [ ] **Step 3: Redis key guard**

Run (from worktree root, no `GOWORK=off` prefix):
```bash
tools/redis-key-guard.sh
```
Expected: clean (no keyed go-redis calls added — this task adds none).

- [ ] **Step 4: Docker bake atlas-channel**

Run (from worktree root):
```bash
docker buildx bake atlas-channel
```
Expected: build succeeds. (`libs/atlas-packet` is an existing lib already COPY'd in the shared Dockerfile; no new lib added, so no Dockerfile/`go.work` edit is needed.)

- [ ] **Step 5: Commit any incidental fixes**

If Steps 1–4 surfaced fixes, commit them:
```bash
git add -A
git commit -m "chore(task-158): verification fixes"
```
If nothing changed, skip this step.

---

## Self-Review

**Spec coverage (PRD §4 / §10 acceptance criteria):**

| Requirement | Task |
|---|---|
| FR-1 `SpiritJavelinItemId()` getter + decode test | Task 1 |
| FR-2 `SHADOW_CLAW` amount == chosen star id | Task 4 (`rewriteShadowClawStatups`, `resolveShadowStarsCast`) + Task 5 (wiring) |
| FR-3 claw + `SHADOW_CLAW` skips per-attack consume; inactive-safe regression | Task 3 |
| FR-4 charge `bulletCount` of chosen star at cast; multi-slot; shortfall posture | Task 2 (`BulletCount`) + Task 4 (`resolveStarConsume`) + Task 5 (`emitStarConsume`) |
| FR-5 validate throwing-star classification + ownership; warn + abort on failure | Task 4 (`validateShadowStar`) + Task 5 (abort wiring + warn log) |
| AC: byte-fixture decode test | Task 1 |
| AC: buff statup value == star id (not 0) | Task 4 `TestResolveShadowStarsCast` |
| AC: claw + `SHADOW_CLAW` → no consume plan | Task 3 |
| AC: claw inactive still consumes (regression) | Task 3 |
| AC: consume targets chosen item id + quantity | Task 4 `TestResolveStarConsume_*` |
| AC: bogus/unowned id rejected + warn, no consume | Task 4 (`ok=false`) + Task 5 (abort + warn) |
| AC: build/vet/test/bake/redis-guard | Task 6 |

**Testing-boundary note (honest scope):** The Kafka emit paths (`emitStarConsume`, and `Plan()`'s internal buff load) are not unit-tested offline — this matches the existing codebase boundary (the projectile `Emit` and `Plan` Kafka/REST paths have no unit tests either; only the pure `resolvePlan` is tested). Coverage is placed on the pure decision functions (`resolveShadowStarsCast`, `projectileConsumptionSkipped`, `resolveStarConsume`, `validateShadowStar`, `rewriteShadowClawStatups`), which carry all the FR logic. The `UseSkill` glue and emit are verified by `go build` + `go test -race ./...` (existing suites stay green) + `docker buildx bake`.

**Placeholder scan:** No TBD/TODO/"handle edge cases"/"similar to Task N" — every code step shows complete code.

**Type consistency:** `StarDraw` fields (`Slot int16`, `ItemId uint32`, `Quantity int16`) match `compartmentMsg.ItemBody{Source int16, ItemId uint32, Quantity int16}` in the emit (`Source: draw.Slot`). `resolveShadowStarsCast` return order `(rewritten, draws, shortfall, ok)` matches every call site (Task 5 wiring and Task 4 test). `statup.NewModel(string, int32)` and `int32(starItemId)` are consistent. `loadCasterInventoryFunc(cp character.Processor, characterId uint32) ([]asset.Model, error)` matches the call in Task 5. `projectileConsumptionSkipped(item.WeaponType, []buff.Model) bool` matches its test and `Plan()` call site.
