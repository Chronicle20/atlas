# Corsair Battleship (5221006) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make Battleship (5221006) a working skill-mount with a Redis-backed ship HP pool drained by damage, break-only cooldown semantics, a client HP gauge, and server-side gating of Cannon (5221007) / Torpedo (5221008).

**Architecture:** All new logic lives in atlas-channel (`battleship` package: in-memory ride mirror + Redis HP counter behind a processor), plus three small shared-lib additions (`IsBattleshipMountSkill` predicate, `TenantCounter` Lua-atomic counter, `ResolveValue` uint32 config resolver). The two client wire values (gauge pseudo-skill 5221999, vehicle item 1932000) are config-resolved from tenant writer-options tables (DOM-25), never hardcoded in service code. Break dismount and cooldown reuse the existing atlas-buffs cancel and atlas-skills `SET_COOLDOWN` surfaces — zero new topics/endpoints.

**Tech Stack:** Go, go-redis v9 (Lua script), miniredis (lib tests), Kafka (existing topics only), JSON seed templates.

## Global Constraints

- Skill ids: Battleship = 5221006, Cannon = 5221007, Torpedo = 5221008 (constants `skill.CorsairBattleshipId` etc. already exist in `libs/atlas-constants/skill/constants.go:3236-3238`).
- Ship HP formula is **version-dependent** — see [§ Post-merge reconciliation](#post-merge-reconciliation-2026-07-28). `major < 87`: `200 × (charLevel + 2×SLV − 120)`, floored at 0. `major >= 87`: `300 × charLevel + 500 × (SLV − 72)`, floored at 0. Both mirror the client's own `get_max_durability_of_vehicle`, so the gauge denominator matches.
- Wire values 5221999 (gauge pseudo-skill id) and 1932000 (vehicle item id) MUST be config-resolved per tenant (DOM-25 / PRD FR-7.2). No literal 5221999/1932000 anywhere under `services/` or `libs/atlas-constants`.
- Cooldown applies ONLY on break — never on cast, manual dismount, expiry, or logout (FR-2.3/FR-4.3). Cooldown duration comes from effect data (`e.Cooldown()`), not a hardcoded 90.
- Ship HP state resets every ride: exists only while riding; "riding but no state" = lazily re-init to full (FR-3.3/FR-5.3).
- All Redis access through `libs/atlas-redis`; `tools/redis-key-guard.sh` must stay clean (run from repo root, NO `GOWORK=off` prefix).
- Damage/attack hot paths: no new REST calls; at most one Redis round-trip, riders only (NFR).
- Break must be exactly-once per depletion under concurrent damage (NFR).
- Test setup uses the project Builder/seam patterns — no `*_testhelpers.go` files.
- Immutable models, `NewProcessor(l, ctx)` processor pattern, `tenant.MustFromContext(ctx)`.
- Commit after every task; all work on branch `task-153-corsair-battleship` in its worktree.

---

## Post-merge reconciliation (2026-07-28)

`main` was merged into this branch after the plan was written (185 commits). The
merge itself was conflict-free, but it invalidated several of the plan's premises.
Everything below was **verified this session** — against the opcode registry, the
seed templates, the live `atlas-main` tenants' ingested WZ data, and the per-version
IDBs. Do not re-derive these from memory; re-verify from the cited source if in doubt.

### R-1. Version set grew from 6 to 11

`deploy/k8s/base/versions.json` now lists 11 versions and
`services/atlas-configurations/seed-data/templates/` holds 11 templates
(`gms_12/48/61/72/79/83/84/87/92/95`, `jms_185`). The live `atlas-main` environment
has **10 tenants** — every version except `gms_12`.

The PRD's "provisioned versions (GMS v83, v84, v87, v92, v95, JMS v185)" (FR-7.1) is
stale in both directions: it omits the four legacy versions, and it assumed the newer
four were usable when they are not.

### R-2. Per-version feasibility matrix

| version | skill 5221006 in WZ | `CharacterUseSkillHandle` (cast) | `CharacterDamageHandle` (drain) | buff/cooldown writers | client gauge sentinel `5221999` | client ship max-HP formula |
|---|---|---|---|---|---|---|
| gms_12 | no live tenant | – | – | none | no IDB | – |
| gms_48 | **absent (HTTP 404)** | – | 0x27 | BuffGive/Cancel only | **absent** | **absent** |
| gms_61 | ✓ 10 levels | 0x53 | 0x2D | all 5 | ✓ | `200×(charLvl + 2×SLV − 120)` |
| gms_72 | ✓ 10 levels | 0x5A | 0x2F | all 5 | ✓ | same |
| gms_79 | ✓ 10 levels | 0x59 | 0x2E | all 5 | ✓ | same |
| gms_83 | ✓ 10 levels | 0x5B | 0x30 | all 5 | ✓ | same |
| gms_84 | ✓ 10 levels | 0x5B | 0x30 | all 5 | ✓ | same |
| gms_87 | ✓ 10 levels | **missing** | **missing** | all 5 | ✓ | **`300×charLvl + 500×(SLV − 72)`** |
| gms_92 | ✓ 10 levels | **missing** | **missing** | **none** | ✓ | **new formula** |
| gms_95 | name only — **`effects: []`** | **missing** | **missing** | all 5 | ✓ | **new formula** |
| jms_185 | ✓ 10 levels | **missing** | **missing** | all 5 | ✓ | **new formula** |

Sources: WZ column = `GET /api/data/skills/5221006` against each live tenant
(tenant headers) on `atlas-data`; handler/writer columns = the seed templates;
sentinel and formula columns = the IDBs (below).

### R-3. Battleship does not exist before v61

`gms_v48` returns **404** for skill 5221006, and the v48 binary contains **zero**
references to either `5221006` (`0x4FAA8E`) or the gauge sentinel `5221999`
(`0x4FAE6F`) — against 15 and 3 respectively in v61. The Corsair Battleship is
**n/a for gms_12 and gms_48**; those two templates get no battleship config and no
verification cell. This is a content-absence finding, not a wiring gap.

### R-4. The client ship max-HP formula changed at v87 — the design missed this

The client renders the gauge as *remaining ÷ its own computed max*. That max comes
from `get_max_durability_of_vehicle(nSkillID, nSLV, nCharLevel)` — PDB-named in the
v95 IDB, which pins the argument order. Decompiled in every IDB:

| version | address | body |
|---|---|---|
| gms_v61 | `sub_652742` | `200 * (a3 + 2*a2 - 120)` |
| gms_v72 | `sub_6B654D` | `200 * (a3 + 2*a2 - 120)` |
| gms_v79 | `sub_6E5883` | `200 * (a3 + 2*a2 - 120)` |
| gms_v83 | `sub_7665F1` | `200 * (a3 + 2*a2 - 120)` |
| gms_v84 | `sub_788A31` | `200 * (a3 + 2*a2 - 120)` |
| gms_v87 | `0x7B331D` (`get_max_durability_of_vehicle`) | **`300 * a3 + 500 * (a2 - 72)`** |
| gms_v92 | `sub_6E2030` | **`300 * a3 + 500 * (a2 - 72)`** |
| gms_v95 | `0x6ED704` (`get_max_durability_of_vehicle`) | **`300 * a3 + 500 * (a2 - 72)`** |
| jms_v185 | `sub_7DC77D` | **`300 * a3 + 500 * (a2 - 72)`** |

Cosmic's `400×SLV + max(charLvl−120,0)×200` — the PRD's FR-2.2 formula — expands to
`200×charLvl + 400×SLV − 24000`, i.e. it equals the **old** client formula for
charLvl ≥ 120 and is simply wrong for v87 and later. Shipping one formula would make
the gauge misrender on four versions (lvl 200 / SLV 10: server 20 000 vs. client max
29 000 — the bar sits at ~69 % on a full ship and the ship "breaks" with the bar
still showing fuel).

**Decision (owner, 2026-07-28): version-gate both formulas.** Task 6 implements both
arms; see R-8.

### R-5. `maxLevel` is 10 on every version — the 28 000 clamp constant is wrong

Battleship is `maxLevel: 10` on all nine tenants that have it (not 30). Recompute the
gauge clamp bound accordingly: the true ceiling is the **new** formula at SLV 10 /
charLvl 200 = **29 000**, not the 28 000 the plan's Task 9 test asserts. It still fits
`uint16` (65 535), so the defensive clamp itself stands — only the comment and the
"formula max fits" case value change.

### R-6. v61 has per-level cooldowns

`gms_v61` skill data carries **ten distinct `cooldown` values — 90, 120, 150, 180,
210, 240, 270, 300, 330, 360** — where v72 and later are a flat 90 at every level.
The plan already reads the cooldown from effect data (`e.Cooldown()`), so no code
change is needed; but the PRD §4.1 claim "`cooltime=90` at **every** level" is a
v72+ fact, not a universal one, and no test may hardcode 90 as the expected cooldown
for v61.

### R-7. v95 has no ingested skill effects at all

`gms_v95` returns `maxLevel: 0, effects: []` for 5221006 — and for every other skill
probed (5221004, 5221007, 5221008, 5121000, 1001003). This is a **tenant-wide v95 WZ
ingestion gap**, not battleship-specific, and it predates this task. Consequence: even
after the v95 handlers are wired (Task 11), the feature cannot be exercised on the v95
tenant until v95 skill data is ingested — `GetEffect` yields nothing, so there is no
statup set, no MP cost, and no cooldown value. Record this in the backfill runbook as
a blocked verification, and do not report v95 as verified on live evidence.

### R-8. Owner decisions taken on the reconciliation

1. **Version scope** — wire the missing `CharacterUseSkillHandle` /
   `CharacterDamageHandle` for gms_87, gms_92, gms_95 and jms_185 as part of this task
   (Task 11), rather than scoping the feature to v61–v84. These are core inbound
   opcodes whose absence blocks far more than battleship.
2. **HP formula** — version-gate both formulas (R-4) so the gauge is correct on every
   version, rather than shipping Cosmic parity everywhere.

### R-9. Stale code anchors — every line reference in this plan was re-pinned

`main` moved essentially all of them (`character_attack_common.go` alone grew by 543
lines). Corrected anchors, verified post-merge:

| plan said | actual (post-merge) |
|---|---|
| `constants.go:3231-3233` | `constants.go:3236-3238` |
| `registry_test.go:16` (`setupTestRedis`) | `registry_test.go:17` |
| `opcodes.WriterConfig` @ `config.go:12` | `config.go:24` (`:12` is `HandlerConfig.OpCode`) |
| `<ch>/go.mod:82` replace; "+ atlas-redis require" | require already at `go.mod:11`, replace at `go.mod:97` — **the go.mod step is now a no-op** |
| `character/buff/processor.go:20` (`Cancel`) | interface `:22`, impl `:63` |
| `character/skill/processor.go:45` (`ApplyCooldown`) | `:53` |
| `data/skill/processor.go:34` (`GetEffect`) | interface `:15`, impl `:37` |
| `data/skill/effect/rest.go:45` (`Cooldown`) | unchanged — still `:45` |
| `mount.go:22` (`MountBuffDuration`) / `:61` (`tamedMountStatups`) | `:23` / `:62` |
| `mount_test.go:72` (`mountInfo`) / `:76-82` (`mountEffect`) | `:73` / `:77` |
| `session/processor.go:330` (`Destroy`) | `:405` |
| `kafka/consumer/buff/consumer.go:63` (inline `MustFromContext`) | `handleStatusEventApplied` at `:114`, the `IsWorld(tenant.MustFromContext(ctx), …)` line at `:120` |
| `skill/handler/common.go:99` (`skillId :=`) | `:143` |
| `character_skill_use.go:70` (skill-level validation) | block **starts** at `:70` |
| `character_damage.go:31` (the TODO) / `:43` (`ChangeHP`) | `:32` / `:44` |
| `character_attack_common.go:283` (`skill3` alias) / `:290` (ownership rejection) | import `:27` / `Destroy(s)` at `:660` |
| `services/atlas-mounts/atlas.com/mounts/main.go:51-52` | unchanged |
| `<ch>/main.go` evictor `~287` / `buildListener` `~404` | `RegisterEvictor` `:281`, `buildListener` `:351`, `Socket.Writers` use `:390` |

Re-confirm each anchor immediately before editing — treat the table as a starting
point, not as gospel, since the branch will move as tasks land.

### R-10. Task 3's test fixture asserts a wrong hex constant

The plan's Task 3 fixture uses `"CORSAIR_BATTLESHIP": "0x1D7B60", // 1932000`.
**`0x1D7B60` is 1 932 128.** 1 932 000 is **`0x1D7AE0`**. As written the test fails.
Fixed in Task 3 below. (For the record: `5221006` = `0x4FAA8E`, `5221999` =
`0x4FAE6F`, `1932000` = `0x1D7AE0`.)

### R-11. New repo guards the plan predates

`main` added four guards that this task must satisfy and that the plan's Task 12
verification list omits:

- `tools/template-opcode-order-guard.sh` — handler and writer arrays must be in
  **strictly ascending `opCode` order**. Task 11 originally said to *append* the v92
  writers; new entries must instead be **inserted at their sorted position**.
- `tools/lint.sh --check` — golangci-lint v2 formatters + `standard` linters across
  every Go module (run `tools/lint.sh` with no flags to fix in place before committing).
- `tools/goroutine-guard.sh` — bare `go` statements banned outside `libs/atlas-routine`.
- `tools/service-registration-guard.sh` — only if service registration lists change
  (this task does not touch them, but run it if that changes).

Also note every template handler entry needs a `validator`; an entry without one is
silently dropped at listener build.

### R-13. v92's two missing serverbound opcodes are derived (2026-07-28)

`SPECIAL_MOVE` (`CharacterUseSkillHandle`) = **`0x66`**, `TAKE_DAMAGE`
(`CharacterDamageHandle`) = **`0x35`**. v92 has no registry column and its opcode table
cannot be interpolated, so both were read out of the binary — full method, evidence
addresses and three cross-checks in Task 11 Step 3, which is now **complete**. IDB symbols
were named (`DoActiveSkill_Heal_CUserLocal` @ `0x91B630`,
`DoActiveSkill_Prepare_CUserLocal` @ `0x91DF00`) and evidence comments set.

### R-12. v92 now has an IDB — the design's "unverified" caveat is resolved

Design §1.1 recorded v92's gauge as unverifiable for want of an IDB. One exists now
(`GMS_v92_1_DEVM.exe`). `CUserLocal::OnSkillCooltimeSet` @ `0x8EF260` carries
`if (v2 == &byte_4FAE6F) v5 = v9;` — the same 5221999 special case as every other
version. v92's gauge is **verified**, not bracketed.

---

## File Structure

| File | Responsibility |
|---|---|
| `libs/atlas-constants/skill/mount.go` (modify) | `IsBattleshipMountSkill` classification predicate |
| `libs/atlas-redis/counter.go` (create) | `TenantCounter`: Set / DecrByIfExists (Lua) / Remove |
| `libs/atlas-packet/resolve.go` (modify) | `ResolveValue`: uint32 config-value resolver |
| `services/atlas-channel/.../socket/writer/options_registry.go` (create) | per-tenant writer-options tables accessor |
| `services/atlas-channel/.../battleship/mirror.go` (create) | in-memory ride mirror (riding truth + skill level + state TTL) |
| `services/atlas-channel/.../battleship/processor.go` (create) | InitShipHP / IsRiding / Drain / Clear; sole owner of mirror + Redis state |
| `services/atlas-channel/.../kafka/consumer/buff/consumer.go` (modify) | mirror put/clear hooks on buff APPLIED/EXPIRED |
| `services/atlas-channel/.../session/processor.go` (modify) | ship-state cleanup on session destroy |
| `services/atlas-channel/.../skill/handler/common.go` (modify) | cast-cooldown carve-out + mount-gate extension |
| `services/atlas-channel/.../skill/handler/mount.go` (modify) | battleship mount arm (vehicle override + HP init) |
| `services/atlas-channel/.../socket/handler/character_skill_use.go` (modify) | cast-while-cooling rejection |
| `services/atlas-channel/.../socket/handler/character_damage.go` (modify) | drain + gauge announce (replaces the `// TODO decrease battleship hp`) |
| `services/atlas-channel/.../socket/handler/character_attack_common.go` (modify) | Cannon/Torpedo riding gate |
| `services/atlas-channel/.../main.go` (modify) | Redis connect, mirror/options eviction, options registration |
| `services/atlas-configurations/seed-data/templates/*.json` (modify) | options tables ×9 (gms_61…jms_185); v92's 5 missing writers; the 8 missing cast/damage handlers on gms_87/92/95/jms_185 — see R-2 |
| `docs/tasks/task-153-corsair-battleship/backfill.md` (create) | live-tenant config backfill runbook |

Service paths below abbreviate `services/atlas-channel/atlas.com/channel/` as `<ch>/`.

Task dependency order: 1 → 8; 2 → 6; 3 → 8,9; 4 → 8,9; 5 → 6,7,10; 6 → 7,8,9. Tasks 1–5 are mutually independent.

---

### Task 1: atlas-constants — battleship mount classification

**Files:**
- Modify: `libs/atlas-constants/skill/mount.go`
- Test: `libs/atlas-constants/skill/mount_test.go`

**Interfaces:**
- Produces: `func IsBattleshipMountSkill(id Id) bool` — true only for `CorsairBattleshipId` (5221006). `IsTamedMountSkill` and `SkillOnlyMountVehicleId` behavior unchanged (battleship stays false/(0,false) in both — the latter is an atlas-data ingestion contract; adding battleship there would hardcode the vehicle id and force WZ re-ingestion).

- [ ] **Step 1: Write the failing test**

In `libs/atlas-constants/skill/mount_test.go`, change the two existing battleship case names (they currently say "out of scope") and append a new test function:

```go
// In TestIsTamedMountSkill's table, replace:
//   {"Battleship (out of scope)", 5221006, false},
// with:
	{"Battleship (skill mount, not tamed)", 5221006, false},

// In TestSkillOnlyMountVehicleId's table, replace:
//   {"Battleship not skill-only", 5221006, 1, 0, false},
// with (same values — pins that battleship never enters the fixed-vehicle ingestion band):
	{"Battleship not fixed-vehicle", 5221006, 1, 0, false},

func TestIsBattleshipMountSkill(t *testing.T) {
	tests := []struct {
		name     string
		id       Id
		expected bool
	}{
		{"Battleship", 5221006, true},
		{"Cannon is not the mount", 5221007, false},
		{"Torpedo is not the mount", 5221008, false},
		{"Broomstick (skill-only)", 1019, false},
		{"Tamed MonsterRider", 1004, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsBattleshipMountSkill(tc.id); got != tc.expected {
				t.Errorf("IsBattleshipMountSkill(%d) = %v, want %v", tc.id, got, tc.expected)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-constants && go test ./skill/ -run TestIsBattleshipMountSkill -v`
Expected: FAIL — `undefined: IsBattleshipMountSkill`

- [ ] **Step 3: Write minimal implementation**

Append to `libs/atlas-constants/skill/mount.go`:

```go
// IsBattleshipMountSkill reports whether id is the Corsair Battleship
// (5221006) skill-mount. Battleship is deliberately NOT in
// SkillOnlyMountVehicleId: its vehicle id is a client wire value resolved
// from tenant configuration at buff-apply time in atlas-channel (DOM-25),
// not baked into ingested skill data.
func IsBattleshipMountSkill(id Id) bool {
	return id == CorsairBattleshipId
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-constants && go test ./skill/ -v -run 'TestIsBattleshipMountSkill|TestIsTamedMountSkill|TestSkillOnlyMountVehicleId'`
Expected: PASS (all three)

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-constants/skill/mount.go libs/atlas-constants/skill/mount_test.go
git commit -m "feat(constants): classify Corsair Battleship as a mount skill"
```

---

### Task 2: atlas-redis — TenantCounter with atomic decrement-if-exists

**Files:**
- Create: `libs/atlas-redis/counter.go`
- Test: `libs/atlas-redis/counter_test.go`

**Interfaces:**
- Consumes: existing `tenantEntityKey(namespace, t, entityKey)` from `libs/atlas-redis/keys.go`, `setupTestRedis(t)` from `libs/atlas-redis/registry_test.go:17` (miniredis).
- Produces:
  - `func NewTenantCounter(client *goredis.Client, namespace string) *TenantCounter`
  - `func (c *TenantCounter) Set(ctx context.Context, t tenant.Model, key string, value int64, ttl time.Duration) error`
  - `func (c *TenantCounter) DecrByIfExists(ctx context.Context, t tenant.Model, key string, delta int64, ttl time.Duration) (newValue int64, existed bool, err error)` — atomic Lua: never creates a missing key, refreshes TTL on every hit
  - `func (c *TenantCounter) Remove(ctx context.Context, t tenant.Model, key string) error` — no-op on missing key

- [ ] **Step 1: Write the failing tests**

Create `libs/atlas-redis/counter_test.go`:

```go
package redis

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestTenantCounter_SetAndTTL(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer client.Close()
	c := NewTenantCounter(client, "test-counter")
	tm := newTestTenant(t, "GMS")
	ctx := context.Background()

	if err := c.Set(ctx, tm, "42", 1000, 35*time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}
	rk := tenantEntityKey("test-counter", tm, "42")
	if got := mr.TTL(rk); got != 35*time.Minute {
		t.Fatalf("TTL = %v, want 35m", got)
	}
	newV, existed, err := c.DecrByIfExists(ctx, tm, "42", 100, 35*time.Minute)
	if err != nil {
		t.Fatalf("DecrByIfExists: %v", err)
	}
	if !existed || newV != 900 {
		t.Fatalf("DecrByIfExists = (%d, %v), want (900, true)", newV, existed)
	}
}

func TestTenantCounter_DecrRefreshesTTL(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer client.Close()
	c := NewTenantCounter(client, "test-counter")
	tm := newTestTenant(t, "GMS")
	ctx := context.Background()

	if err := c.Set(ctx, tm, "1", 500, 10*time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}
	mr.FastForward(9 * time.Minute)
	if _, _, err := c.DecrByIfExists(ctx, tm, "1", 10, 10*time.Minute); err != nil {
		t.Fatalf("DecrByIfExists: %v", err)
	}
	rk := tenantEntityKey("test-counter", tm, "1")
	if got := mr.TTL(rk); got != 10*time.Minute {
		t.Fatalf("TTL after decr = %v, want refreshed 10m", got)
	}
}

func TestTenantCounter_DecrMissingKeyDoesNotCreate(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer client.Close()
	c := NewTenantCounter(client, "test-counter")
	tm := newTestTenant(t, "GMS")

	newV, existed, err := c.DecrByIfExists(context.Background(), tm, "77", 100, time.Minute)
	if err != nil {
		t.Fatalf("DecrByIfExists: %v", err)
	}
	if existed || newV != 0 {
		t.Fatalf("DecrByIfExists = (%d, %v), want (0, false)", newV, existed)
	}
	if mr.Exists(tenantEntityKey("test-counter", tm, "77")) {
		t.Fatal("missing key was created by DecrByIfExists")
	}
}

func TestTenantCounter_RemoveIdempotent(t *testing.T) {
	client, _ := setupTestRedis(t)
	defer client.Close()
	c := NewTenantCounter(client, "test-counter")
	tm := newTestTenant(t, "GMS")
	ctx := context.Background()

	if err := c.Set(ctx, tm, "9", 5, time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := c.Remove(ctx, tm, "9"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if err := c.Remove(ctx, tm, "9"); err != nil {
		t.Fatalf("Remove (missing): %v", err)
	}
	if _, existed, _ := c.DecrByIfExists(ctx, tm, "9", 1, time.Minute); existed {
		t.Fatal("key still exists after Remove")
	}
}

// Exactly one concurrent caller observes the zero crossing
// (newValue <= 0 && newValue+delta > 0), and no decrement is lost.
func TestTenantCounter_ConcurrentDecrExactlyOneCrossing(t *testing.T) {
	client, _ := setupTestRedis(t)
	defer client.Close()
	c := NewTenantCounter(client, "test-counter")
	tm := newTestTenant(t, "GMS")
	ctx := context.Background()

	const workers = 10
	const delta = int64(100)
	if err := c.Set(ctx, tm, "ship", 500, time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}

	var wg sync.WaitGroup
	crossings := make(chan int64, workers)
	var finalMu sync.Mutex
	var lastValues []int64
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			newV, existed, err := c.DecrByIfExists(ctx, tm, "ship", delta, time.Minute)
			if err != nil || !existed {
				t.Errorf("DecrByIfExists = (%d, %v, %v)", newV, existed, err)
				return
			}
			finalMu.Lock()
			lastValues = append(lastValues, newV)
			finalMu.Unlock()
			if newV <= 0 && newV+delta > 0 {
				crossings <- newV
			}
		}()
	}
	wg.Wait()
	close(crossings)

	var crossed int
	for range crossings {
		crossed++
	}
	if crossed != 1 {
		t.Fatalf("crossings = %d, want exactly 1", crossed)
	}
	var min int64
	for _, v := range lastValues {
		if v < min {
			min = v
		}
	}
	if want := int64(500) - int64(workers)*delta; min != want {
		t.Fatalf("lowest observed value = %d, want %d (no decrement lost)", min, want)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd libs/atlas-redis && go test -run TestTenantCounter -v ./...`
Expected: FAIL — `undefined: NewTenantCounter`

- [ ] **Step 3: Write the implementation**

Create `libs/atlas-redis/counter.go`:

```go
package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

// decrByIfExistsScript atomically decrements an EXISTING counter and
// refreshes its TTL, returning {newValue, 1}. A missing key is NOT created
// (a bare DECRBY would create it at -delta, turning "state lost" into an
// instant zero-crossing); it returns {0, 0} so the caller can take a lazy
// re-initialization path instead.
var decrByIfExistsScript = goredis.NewScript(`
if redis.call("exists", KEYS[1]) == 1 then
	local v = redis.call("decrby", KEYS[1], ARGV[1])
	redis.call("pexpire", KEYS[1], ARGV[2])
	return {v, 1}
else
	return {0, 0}
end`)

// TenantCounter is a tenant-scoped int64 counter with a TTL-bounded
// lifetime. Decrements are serialized by Redis, so concurrent callers never
// lose an update and exactly one caller observes any given zero crossing
// (newValue <= 0 && newValue+delta > 0).
type TenantCounter struct {
	client    *goredis.Client
	namespace string
}

func NewTenantCounter(client *goredis.Client, namespace string) *TenantCounter {
	return &TenantCounter{client: client, namespace: namespace}
}

func (c *TenantCounter) entityKey(t tenant.Model, key string) string {
	return tenantEntityKey(c.namespace, t, key)
}

// Set stores value with ttl, replacing any prior value and TTL.
func (c *TenantCounter) Set(ctx context.Context, t tenant.Model, key string, value int64, ttl time.Duration) error {
	if err := c.client.Set(ctx, c.entityKey(t, key), value, ttl).Err(); err != nil {
		return fmt.Errorf("redis set: %w", err)
	}
	return nil
}

// DecrByIfExists atomically decrements the counter by delta and refreshes
// its TTL. Returns existed=false (without creating the key) when the
// counter is absent.
func (c *TenantCounter) DecrByIfExists(ctx context.Context, t tenant.Model, key string, delta int64, ttl time.Duration) (int64, bool, error) {
	res, err := decrByIfExistsScript.Run(ctx, c.client, []string{c.entityKey(t, key)}, delta, ttl.Milliseconds()).Int64Slice()
	if err != nil {
		return 0, false, fmt.Errorf("redis decr-if-exists: %w", err)
	}
	if len(res) != 2 {
		return 0, false, fmt.Errorf("redis decr-if-exists: unexpected reply length %d", len(res))
	}
	return res[0], res[1] == 1, nil
}

// Remove deletes the counter. Removing a missing key is a no-op.
func (c *TenantCounter) Remove(ctx context.Context, t tenant.Model, key string) error {
	if err := c.client.Del(ctx, c.entityKey(t, key)).Err(); err != nil {
		return fmt.Errorf("redis del: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-redis && go test -race -run TestTenantCounter -v ./...`
Expected: PASS (5 tests). Then full module: `go test -race ./... && go vet ./...` — clean.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-redis/counter.go libs/atlas-redis/counter_test.go
git commit -m "feat(atlas-redis): TenantCounter with atomic decrement-if-exists"
```

---

### Task 3: atlas-packet — ResolveValue (uint32 config resolver)

**Files:**
- Modify: `libs/atlas-packet/resolve.go`
- Test: `libs/atlas-packet/resolve_test.go`

**Interfaces:**
- Produces: `func ResolveValue(l logrus.FieldLogger, options map[string]interface{}, property string, key string) (uint32, bool)` — same nested-map format as `ResolveCode` (`options[property][key]`, float64 or base-0 string). Unlike `ResolveCode` (byte sentinel 99), a 4-byte wire value has no safe sentinel, so a miss logs an error and returns `(0, false)`; callers must skip the write.

- [ ] **Step 1: Write the failing tests**

Append to `libs/atlas-packet/resolve_test.go`:

```go
func TestResolveValueValid(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	options := map[string]interface{}{
		"skills": map[string]interface{}{
			"BATTLESHIP_HP_GAUGE": float64(5221999),
		},
		"vehicles": map[string]interface{}{
			"CORSAIR_BATTLESHIP": "0x1D7AE0", // 1932000 — see R-10; 0x1D7B60 is 1932128, NOT 1932000
		},
	}
	v, ok := ResolveValue(l, options, "skills", "BATTLESHIP_HP_GAUGE")
	assert.True(t, ok)
	assert.Equal(t, uint32(5221999), v)
	v, ok = ResolveValue(l, options, "vehicles", "CORSAIR_BATTLESHIP")
	assert.True(t, ok)
	assert.Equal(t, uint32(1932000), v)
}

func TestResolveValueMisses(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	cases := []struct {
		name    string
		options map[string]interface{}
	}{
		{"missing property", map[string]interface{}{}},
		{"property not a map", map[string]interface{}{"skills": "nope"}},
		{"missing key", map[string]interface{}{"skills": map[string]interface{}{}}},
		{"unparseable string", map[string]interface{}{"skills": map[string]interface{}{"BATTLESHIP_HP_GAUGE": "zz"}}},
		{"unsupported type", map[string]interface{}{"skills": map[string]interface{}{"BATTLESHIP_HP_GAUGE": true}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v, ok := ResolveValue(l, tc.options, "skills", "BATTLESHIP_HP_GAUGE")
			assert.False(t, ok)
			assert.Equal(t, uint32(0), v)
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd libs/atlas-packet && go test -run TestResolveValue -v .`
Expected: FAIL — `undefined: ResolveValue`

- [ ] **Step 3: Write the implementation**

Append to `libs/atlas-packet/resolve.go`:

```go
// ResolveValue looks up a uint32 wire value from the runtime options map.
// Same nested-map format as ResolveCode: options[property][key] = value,
// where the value may be a JSON number (float64) or a string parsable by
// strconv.ParseUint with base 0 (e.g. "0x4FAE6F"). Unlike ResolveCode there
// is no safe sentinel for a 4-byte wire value, so any miss logs an error and
// returns (0, false); callers must skip the write rather than send a guess.
func ResolveValue(l logrus.FieldLogger, options map[string]interface{}, property string, key string) (uint32, bool) {
	genericValues, ok := options[property]
	if !ok {
		l.Errorf("Property [%s] missing from options when resolving value [%s].", property, key)
		return 0, false
	}

	values, ok := genericValues.(map[string]interface{})
	if !ok {
		l.Errorf("Property [%s] is not a map when resolving value [%s].", property, key)
		return 0, false
	}

	raw, ok := values[key]
	if !ok {
		l.Errorf("Value [%s] not configured in property [%s].", key, property)
		return 0, false
	}

	switch v := raw.(type) {
	case float64:
		return uint32(v), true
	case string:
		n, err := strconv.ParseUint(v, 0, 32)
		if err != nil {
			l.WithError(err).Errorf("Value [%s] in property [%s] has unparseable value [%q].", key, property, v)
			return 0, false
		}
		return uint32(n), true
	default:
		l.Errorf("Value [%s] in property [%s] has unsupported type %T.", key, property, raw)
		return 0, false
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-packet && go test -race -run TestResolveValue -v . && go vet ./...`
Expected: PASS, vet clean

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/resolve.go libs/atlas-packet/resolve_test.go
git commit -m "feat(atlas-packet): ResolveValue uint32 config resolver"
```

---

### Task 4: atlas-channel — per-tenant writer-options registry

**Files:**
- Create: `<ch>/socket/writer/options_registry.go`
- Test: `<ch>/socket/writer/options_registry_test.go`
- Modify: `<ch>/main.go` (registration in `buildListener` — declared at `:351`, `tenantCfg.Socket.Writers` in scope at `:390`; eviction in the `listener.RegisterEvictor` block at `:281`)

**Interfaces:**
- Consumes: `opcodes.WriterConfig` (`libs/atlas-opcodes/config.go:24` — fields `OpCode`, `Writer`, `Options map[string]interface{}`, `Services []string`; note `:12` is `HandlerConfig.OpCode`, not the writer type); `tenantCfg.Socket.Writers` already in scope in `buildListener` (used at `main.go:390`).
- Produces (package `writer`, so handlers reach it via their existing `atlas-channel/socket/writer` import):
  - `func RegisterTenantWriterOptions(tenantId uuid.UUID, writers []opcodes.WriterConfig)`
  - `func TenantWriterOptions(tenantId uuid.UUID, writerName string) (map[string]interface{}, bool)`
  - `func EvictTenantWriterOptions(tenantId uuid.UUID)`

Rationale: writers get their options at encode time via `BuildWriterProducer`, but domain logic outside an encode path (the battleship mount arm resolving the vehicle id; the damage handler resolving the gauge pseudo-id before deciding to send at all) needs to read the same tables. This registry exposes the already-loaded socket configuration; no second config fetch.

- [ ] **Step 1: Write the failing test**

Create `<ch>/socket/writer/options_registry_test.go`:

```go
package writer

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-opcodes"
	"github.com/google/uuid"
)

func TestTenantWriterOptionsLifecycle(t *testing.T) {
	tid := uuid.New()
	other := uuid.New()
	RegisterTenantWriterOptions(tid, []opcodes.WriterConfig{
		{OpCode: "0xEA", Writer: "CharacterSkillCooldown", Options: map[string]interface{}{
			"skills": map[string]interface{}{"BATTLESHIP_HP_GAUGE": float64(5221999)},
		}},
		{OpCode: "0x20", Writer: "CharacterBuffGive"}, // nil options
	})
	t.Cleanup(func() { EvictTenantWriterOptions(tid) })

	opts, ok := TenantWriterOptions(tid, "CharacterSkillCooldown")
	if !ok {
		t.Fatal("expected options for CharacterSkillCooldown")
	}
	if _, hasSkills := opts["skills"]; !hasSkills {
		t.Fatal("expected skills table in options")
	}

	if _, ok := TenantWriterOptions(tid, "CharacterBuffGive"); ok {
		t.Fatal("writer with nil options should report ok=false")
	}
	if _, ok := TenantWriterOptions(tid, "NoSuchWriter"); ok {
		t.Fatal("unknown writer should report ok=false")
	}
	if _, ok := TenantWriterOptions(other, "CharacterSkillCooldown"); ok {
		t.Fatal("unregistered tenant should report ok=false")
	}

	EvictTenantWriterOptions(tid)
	if _, ok := TenantWriterOptions(tid, "CharacterSkillCooldown"); ok {
		t.Fatal("evicted tenant should report ok=false")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/writer/ -run TestTenantWriterOptions -v`
Expected: FAIL — `undefined: RegisterTenantWriterOptions`

- [ ] **Step 3: Write the implementation**

Create `<ch>/socket/writer/options_registry.go`:

```go
package writer

import (
	"sync"

	"github.com/Chronicle20/atlas/libs/atlas-opcodes"
	"github.com/google/uuid"
)

// tenantWriterOptions holds each tenant's writer options tables so domain
// logic outside an encode path can resolve the same config-driven wire
// values writers use (DOM-25) — e.g. the battleship mount arm resolving the
// vehicle id from the CharacterBuffGive entry. Populated when the tenant's
// listener is built; evicted alongside the tenant's other caches.
var (
	tenantWriterOptionsMu sync.RWMutex
	tenantWriterOptions   = map[uuid.UUID]map[string]map[string]interface{}{}
)

// RegisterTenantWriterOptions records the options table of every writer in
// the tenant's socket configuration. Writers without options are skipped.
func RegisterTenantWriterOptions(tenantId uuid.UUID, writers []opcodes.WriterConfig) {
	tables := make(map[string]map[string]interface{})
	for _, wc := range writers {
		if len(wc.Options) > 0 {
			tables[wc.Writer] = wc.Options
		}
	}
	tenantWriterOptionsMu.Lock()
	defer tenantWriterOptionsMu.Unlock()
	tenantWriterOptions[tenantId] = tables
}

// TenantWriterOptions returns the named writer's options table for the
// tenant. ok=false when the tenant is unregistered or the writer has no
// options configured.
func TenantWriterOptions(tenantId uuid.UUID, writerName string) (map[string]interface{}, bool) {
	tenantWriterOptionsMu.RLock()
	defer tenantWriterOptionsMu.RUnlock()
	tables, ok := tenantWriterOptions[tenantId]
	if !ok {
		return nil, false
	}
	opts, ok := tables[writerName]
	return opts, ok
}

// EvictTenantWriterOptions drops the tenant's tables.
func EvictTenantWriterOptions(tenantId uuid.UUID) {
	tenantWriterOptionsMu.Lock()
	defer tenantWriterOptionsMu.Unlock()
	delete(tenantWriterOptions, tenantId)
}
```

- [ ] **Step 4: Wire registration and eviction in main.go**

In `<ch>/main.go`, inside `buildListener`, immediately after `wp := produceWriterProducer(fl)(tenantCfg.Socket.Writers, writerList, rw)` (line ~404), add:

```go
		writer.RegisterTenantWriterOptions(t.Id(), tenantCfg.Socket.Writers)
```

(The `writer` package is already imported by main.go for `writer.Producer`.)

In the `listener.RegisterEvictor` callback (line ~287), add alongside the other evictions:

```go
		writer.EvictTenantWriterOptions(tid)
```

- [ ] **Step 5: Run tests and build**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/writer/ -run TestTenantWriterOptions -v && go build ./...`
Expected: PASS, build clean

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/writer/options_registry.go services/atlas-channel/atlas.com/channel/socket/writer/options_registry_test.go services/atlas-channel/atlas.com/channel/main.go
git commit -m "feat(channel): per-tenant writer-options registry"
```

---

### Task 5: atlas-channel — battleship ride mirror

**Files:**
- Create: `<ch>/battleship/mirror.go`
- Test: `<ch>/battleship/mirror_test.go`
- Modify: `<ch>/main.go` (evictor)

**Interfaces:**
- Consumes: `tenant.Model` (`t.Id()`), pattern precedent `<ch>/monster/status_mirror.go` (sync.Once singleton, per-tenant nesting, EvictTenant).
- Produces (package `battleship`):
  - `type RideState struct { SkillLevel byte; StateTTL time.Duration }` — `StateTTL` is the effect-derived ship-state TTL used to refresh the Redis entry on every drain; 0 means "use fallback".
  - `func GetRideMirror() *RideMirror`
  - `func (m *RideMirror) Put(t tenant.Model, characterId uint32, s RideState)`
  - `func (m *RideMirror) Get(t tenant.Model, characterId uint32) (RideState, bool)`
  - `func (m *RideMirror) Remove(t tenant.Model, characterId uint32)`
  - `func (m *RideMirror) EvictTenant(tid uuid.UUID)`

- [ ] **Step 1: Write the failing test**

Create `<ch>/battleship/mirror_test.go`:

```go
package battleship

import (
	"testing"
	"time"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
)

func testTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tm
}

func TestRideMirrorLifecycle(t *testing.T) {
	m := GetRideMirror()
	t1 := testTenant(t)
	t2 := testTenant(t)
	t.Cleanup(func() { m.EvictTenant(t1.Id()); m.EvictTenant(t2.Id()) })

	if _, ok := m.Get(t1, 100); ok {
		t.Fatal("empty mirror should miss")
	}

	m.Put(t1, 100, RideState{SkillLevel: 7, StateTTL: 35 * time.Minute})
	rs, ok := m.Get(t1, 100)
	if !ok || rs.SkillLevel != 7 || rs.StateTTL != 35*time.Minute {
		t.Fatalf("Get = (%+v, %v), want skillLevel 7 ttl 35m", rs, ok)
	}

	if _, ok := m.Get(t2, 100); ok {
		t.Fatal("tenant isolation violated: t2 sees t1's rider")
	}

	m.Remove(t1, 100)
	if _, ok := m.Get(t1, 100); ok {
		t.Fatal("Remove did not clear the entry")
	}
	m.Remove(t1, 100) // idempotent

	m.Put(t1, 200, RideState{SkillLevel: 1})
	m.EvictTenant(t1.Id())
	if _, ok := m.Get(t1, 200); ok {
		t.Fatal("EvictTenant did not clear entries")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./battleship/ -v`
Expected: FAIL — package does not exist / `undefined: GetRideMirror`

- [ ] **Step 3: Write the implementation**

Create `<ch>/battleship/mirror.go`:

```go
package battleship

import (
	"sync"
	"time"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
)

// RideState is the pod-local truth that a character is currently riding the
// battleship. SkillLevel is the caster's trained 5221006 level (from the
// buff event) so the break and lazy re-init paths need no skill-book fetch.
// StateTTL is the effect-derived TTL used to refresh the Redis ship-HP
// entry on every drain (FR-5.2 idle-expiry); 0 means "use the fallback".
type RideState struct {
	SkillLevel byte
	StateTTL   time.Duration
}

// RideMirror is a per-channel-process, in-memory projection of battleship
// MONSTER_RIDING buff events, read by the damage and attack hot paths with
// zero I/O. A character's socket session lives on exactly one channel pod,
// so the pod that receives its damage/attack packets is the pod whose buff
// consumer populated this mirror. Pattern precedent: monster.StatusMirror.
type RideMirror struct {
	mu        sync.RWMutex
	perTenant map[uuid.UUID]map[uint32]RideState
}

var (
	rideMirrorOnce sync.Once
	rideMirror     *RideMirror
)

// GetRideMirror returns the process-wide singleton mirror.
func GetRideMirror() *RideMirror {
	rideMirrorOnce.Do(func() {
		rideMirror = &RideMirror{perTenant: map[uuid.UUID]map[uint32]RideState{}}
	})
	return rideMirror
}

func (m *RideMirror) Put(t tenant.Model, characterId uint32, s RideState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tenantMap, ok := m.perTenant[t.Id()]
	if !ok {
		tenantMap = map[uint32]RideState{}
		m.perTenant[t.Id()] = tenantMap
	}
	tenantMap[characterId] = s
}

func (m *RideMirror) Get(t tenant.Model, characterId uint32) (RideState, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tenantMap, ok := m.perTenant[t.Id()]
	if !ok {
		return RideState{}, false
	}
	s, ok := tenantMap[characterId]
	return s, ok
}

func (m *RideMirror) Remove(t tenant.Model, characterId uint32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if tenantMap, ok := m.perTenant[t.Id()]; ok {
		delete(tenantMap, characterId)
	}
}

// EvictTenant drops every entry for the tenant (listener drain).
func (m *RideMirror) EvictTenant(tid uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.perTenant, tid)
}
```

- [ ] **Step 4: Wire eviction in main.go**

In `<ch>/main.go`'s `listener.RegisterEvictor` callback (`:281`), add:

```go
		battleship.GetRideMirror().EvictTenant(tid)
```

with import `"atlas-channel/battleship"`.

- [ ] **Step 5: Run tests and build**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./battleship/ -v && go build ./...`
Expected: PASS, build clean

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/battleship/ services/atlas-channel/atlas.com/channel/main.go
git commit -m "feat(channel): battleship ride mirror"
```

---

### Task 6: atlas-channel — battleship processor (HP pool, drain, break)

**Files:**
- Create: `<ch>/battleship/processor.go`
- Test: `<ch>/battleship/processor_test.go`
- Modify: `<ch>/main.go` (Redis connect + InitRegistry). **`<ch>/go.mod` needs no edit** — `main` already added both the require (`go.mod:11`) and the replace (`go.mod:97`); see R-9.

**Interfaces:**
- Consumes: Task 2 `redis.TenantCounter`; Task 5 mirror; `buff.NewProcessor(l, ctx).Cancel(f, characterId, sourceId)` (`<ch>/character/buff/processor.go:63`, interface `:22`); `charskill.NewProcessor(l, ctx).ApplyCooldown(f, skillId, cooldown)(characterId)` (`<ch>/character/skill/processor.go:53` — emits the existing `SET_COOLDOWN` Kafka command; the existing consumer announces the client packet); `dataskill.NewProcessor(l, ctx).GetEffect(uniqueId, level)` (`<ch>/data/skill/processor.go:37`, interface `:15`); `character.NewProcessor(l, ctx).GetById()(characterId)` → `.Level() byte`. Also `tenant.Model.IsRegion` / `.MajorAtLeast` (`libs/atlas-tenant/tenant.go:88,93`) for the formula gate.
- Produces (package `battleship`):
  - `func ShipHP(t tenant.Model, skillLevel byte, charLevel byte) int32` — the **version-gated** formula (R-4)
  - `func isPostBigBangDurability(t tenant.Model) bool` — the v87+ predicate
  - `func InitRegistry(client *goredis.Client)` — wires the production `TenantCounter` (namespace `battleship-hp`)
  - `type DrainStatus int` with `DrainNotRiding, DrainSkipped, DrainDrained, DrainBroke`
  - `type DrainResult struct { Status DrainStatus; RemainingHP int32 }`
  - `type Processor interface { InitShipHP(characterId uint32, skillLevel byte, charLevel byte, ttl time.Duration) error; IsRiding(characterId uint32) (byte, bool); Drain(f field.Model, characterId uint32, damage int32) DrainResult; Clear(characterId uint32) }`
  - `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor` (ctx must carry tenant)

Import-cycle note (verified): `battleship` imports `character`, `character/buff`, `character/skill`, `data/skill` — none of which import `session` — so `session` (Task 7) and the socket handlers may import `battleship` freely.

- [ ] **Step 1: Write the failing tests**

Create `<ch>/battleship/processor_test.go`:

```go
package battleship

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"atlas-channel/data/skill/effect"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// fakeStore is an in-memory counterStore with TenantCounter's atomicity
// semantics (serialized decrements, no key creation on miss).
type fakeStore struct {
	mu     sync.Mutex
	values map[string]int64
	ttls   map[string]time.Duration
	err    error
}

func newFakeStore() *fakeStore {
	return &fakeStore{values: map[string]int64{}, ttls: map[string]time.Duration{}}
}

func (s *fakeStore) k(t tenant.Model, key string) string { return t.Id().String() + ":" + key }

func (s *fakeStore) Set(_ context.Context, t tenant.Model, key string, value int64, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return s.err
	}
	s.values[s.k(t, key)] = value
	s.ttls[s.k(t, key)] = ttl
	return nil
}

func (s *fakeStore) DecrByIfExists(_ context.Context, t tenant.Model, key string, delta int64, ttl time.Duration) (int64, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return 0, false, s.err
	}
	v, ok := s.values[s.k(t, key)]
	if !ok {
		return 0, false, nil
	}
	v -= delta
	s.values[s.k(t, key)] = v
	s.ttls[s.k(t, key)] = ttl
	return v, true, nil
}

func (s *fakeStore) Remove(_ context.Context, t tenant.Model, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return s.err
	}
	delete(s.values, s.k(t, key))
	delete(s.ttls, s.k(t, key))
	return nil
}

type breakRecorder struct {
	mu        sync.Mutex
	cancels   int
	cooldowns []uint32
}

func setupProcessor(t *testing.T) (Processor, *fakeStore, *breakRecorder, tenant.Model, logrus.FieldLogger) {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), tm)
	l, _ := testlog.NewNullLogger()

	fs := newFakeStore()
	prevStore := store
	store = fs
	rec := &breakRecorder{}
	prevCancel, prevCooldown, prevEffect, prevLevel := cancelBuffFunc, applyCooldownFunc, effectFunc, characterLevelFunc
	cancelBuffFunc = func(_ logrus.FieldLogger, _ context.Context, _ field.Model, _ uint32) error {
		rec.mu.Lock()
		defer rec.mu.Unlock()
		rec.cancels++
		return nil
	}
	applyCooldownFunc = func(_ logrus.FieldLogger, _ context.Context, _ field.Model, cooldown uint32, _ uint32) error {
		rec.mu.Lock()
		defer rec.mu.Unlock()
		rec.cooldowns = append(rec.cooldowns, cooldown)
		return nil
	}
	effectFunc = func(_ logrus.FieldLogger, _ context.Context, _ byte) (effect.Model, error) {
		return effect.Extract(effect.RestModel{Cooldown: 90})
	}
	characterLevelFunc = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (byte, error) {
		return 150, nil
	}
	t.Cleanup(func() {
		store = prevStore
		cancelBuffFunc, applyCooldownFunc, effectFunc, characterLevelFunc = prevCancel, prevCooldown, prevEffect, prevLevel
		GetRideMirror().EvictTenant(tm.Id())
	})
	return NewProcessor(l, ctx), fs, rec, tm, l
}

// ShipHP is version-gated (R-4): the client computes the gauge's denominator
// itself via get_max_durability_of_vehicle, and that function changed at v87.
// Both arms below mirror the corresponding client exactly over the reachable
// input range (Battleship is 4th-job, so charLevel >= 120 always).
func TestShipHPFormula(t *testing.T) {
	tests := []struct {
		name       string
		region     string
		major      uint16
		skillLevel byte
		charLevel  byte
		expected   int32
	}{
		// Pre-v87 arm: 400*SLV + max(charLevel-120,0)*200. Identical to the
		// client's 200*(charLevel + 2*SLV - 120) for charLevel >= 120; the
		// clamp keeps sub-120 (unreachable) input from going negative.
		{"v83 sub-120 clamp", "GMS", 83, 1, 100, 400},
		{"v83 exactly 120", "GMS", 83, 10, 120, 4000},
		{"v83 121 adds one step", "GMS", 83, 10, 121, 4200},
		{"v83 max: SLV 10, level 200", "GMS", 83, 10, 200, 20000},
		{"v83 mid: SLV 7, level 150", "GMS", 83, 7, 150, 8800},
		{"v61 same arm as v83", "GMS", 61, 7, 150, 8800},
		{"v84 still the old arm", "GMS", 84, 10, 200, 20000},

		// v87+ arm: max(300*charLevel + 500*(SLV-72), 0).
		{"v87 crosses to the new arm", "GMS", 87, 10, 200, 29000},
		{"v92 new arm", "GMS", 92, 10, 120, 5000},
		{"v95 new arm mid", "GMS", 95, 7, 150, 12500},
		{"v95 SLV 1 at 120", "GMS", 95, 1, 120, 500},
		{"jms185 uses the new arm", "JMS", 185, 10, 200, 29000},
		{"new arm floors at zero", "GMS", 95, 1, 100, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tm, err := tenant.Create(uuid.New(), tc.region, tc.major, 1)
			if err != nil {
				t.Fatalf("tenant.Create: %v", err)
			}
			if got := ShipHP(tm, tc.skillLevel, tc.charLevel); got != tc.expected {
				t.Errorf("ShipHP(%s v%d, SLV %d, charLevel %d) = %d, want %d",
					tc.region, tc.major, tc.skillLevel, tc.charLevel, got, tc.expected)
			}
		})
	}
}

func TestInitShipHPStoresFormulaWithTTL(t *testing.T) {
	p, fs, _, tm, _ := setupProcessor(t)
	if err := p.InitShipHP(100, 7, 150, 35*time.Minute); err != nil {
		t.Fatalf("InitShipHP: %v", err)
	}
	if got := fs.values[fs.k(tm, "100")]; got != 8800 {
		t.Fatalf("stored HP = %d, want 8800", got)
	}
	if got := fs.ttls[fs.k(tm, "100")]; got != 35*time.Minute {
		t.Fatalf("TTL = %v, want 35m", got)
	}
}

func TestDrainNotRiding(t *testing.T) {
	p, _, _, _, _ := setupProcessor(t)
	res := p.Drain(field.Model{}, 100, 250)
	if res.Status != DrainNotRiding {
		t.Fatalf("Status = %v, want DrainNotRiding", res.Status)
	}
}

func TestDrainZeroDamageSkipped(t *testing.T) {
	p, _, _, tm, _ := setupProcessor(t)
	GetRideMirror().Put(tm, 100, RideState{SkillLevel: 7})
	if res := p.Drain(field.Model{}, 100, 0); res.Status != DrainSkipped {
		t.Fatalf("Status = %v, want DrainSkipped", res.Status)
	}
}

func TestDrainDecrementsAndReports(t *testing.T) {
	p, _, rec, tm, _ := setupProcessor(t)
	GetRideMirror().Put(tm, 100, RideState{SkillLevel: 7, StateTTL: 35 * time.Minute})
	if err := p.InitShipHP(100, 7, 150, 35*time.Minute); err != nil {
		t.Fatalf("InitShipHP: %v", err)
	}
	res := p.Drain(field.Model{}, 100, 300)
	if res.Status != DrainDrained || res.RemainingHP != 8500 {
		t.Fatalf("Drain = %+v, want Drained/8500", res)
	}
	if rec.cancels != 0 || len(rec.cooldowns) != 0 {
		t.Fatal("non-breaking drain must not cancel or apply cooldown")
	}
}

func TestDrainLazyReinit(t *testing.T) {
	p, fs, _, tm, _ := setupProcessor(t)
	GetRideMirror().Put(tm, 100, RideState{SkillLevel: 7}) // no Redis entry
	res := p.Drain(field.Model{}, 100, 300)
	// full = ShipHP(7, 150) = 8800 → 8500 remaining
	if res.Status != DrainDrained || res.RemainingHP != 8500 {
		t.Fatalf("Drain = %+v, want Drained/8500 after lazy re-init", res)
	}
	if got := fs.values[fs.k(tm, "100")]; got != 8500 {
		t.Fatalf("stored HP = %d, want 8500", got)
	}
}

func TestDrainLazyReinitOverkillBreaks(t *testing.T) {
	p, fs, rec, tm, _ := setupProcessor(t)
	GetRideMirror().Put(tm, 100, RideState{SkillLevel: 1}) // full = 6400 (skill 1, level 150)
	res := p.Drain(field.Model{}, 100, 9999)
	if res.Status != DrainBroke {
		t.Fatalf("Status = %v, want DrainBroke", res.Status)
	}
	if rec.cancels != 1 || len(rec.cooldowns) != 1 || rec.cooldowns[0] != 90 {
		t.Fatalf("break side effects = cancels %d cooldowns %v, want 1/[90]", rec.cancels, rec.cooldowns)
	}
	if _, ok := fs.values[fs.k(tm, "100")]; ok {
		t.Fatal("ship state must be cleared on break")
	}
	if _, riding := GetRideMirror().Get(tm, 100); riding {
		t.Fatal("mirror must be cleared on break")
	}
}

func TestDrainBreakExactlyOnceUnderConcurrency(t *testing.T) {
	p, _, rec, tm, _ := setupProcessor(t)
	GetRideMirror().Put(tm, 100, RideState{SkillLevel: 7, StateTTL: time.Minute})
	if err := p.InitShipHP(100, 7, 150, time.Minute); err != nil {
		t.Fatalf("InitShipHP: %v", err)
	}
	// 8800 HP, 10 workers × 1000 damage: crosses zero exactly once.
	var wg sync.WaitGroup
	broke := 0
	var mu sync.Mutex
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if res := p.Drain(field.Model{}, 100, 1000); res.Status == DrainBroke {
				mu.Lock()
				broke++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if broke != 1 {
		t.Fatalf("DrainBroke observed %d times, want exactly 1", broke)
	}
	if rec.cancels != 1 || len(rec.cooldowns) != 1 {
		t.Fatalf("break side effects ran %d/%d times, want once", rec.cancels, len(rec.cooldowns))
	}
}

func TestDrainRedisErrorDegrades(t *testing.T) {
	p, fs, rec, tm, _ := setupProcessor(t)
	GetRideMirror().Put(tm, 100, RideState{SkillLevel: 7})
	fs.err = errors.New("redis down")
	res := p.Drain(field.Model{}, 100, 300)
	if res.Status != DrainSkipped {
		t.Fatalf("Status = %v, want DrainSkipped on Redis error", res.Status)
	}
	if rec.cancels != 0 || len(rec.cooldowns) != 0 {
		t.Fatal("degraded drain must have no side effects")
	}
}

func TestClearIdempotent(t *testing.T) {
	p, fs, _, tm, _ := setupProcessor(t)
	GetRideMirror().Put(tm, 100, RideState{SkillLevel: 7})
	if err := p.InitShipHP(100, 7, 150, time.Minute); err != nil {
		t.Fatalf("InitShipHP: %v", err)
	}
	p.Clear(100)
	p.Clear(100) // second call is a no-op
	if _, ok := fs.values[fs.k(tm, "100")]; ok {
		t.Fatal("Clear did not remove ship state")
	}
	if _, riding := GetRideMirror().Get(tm, 100); riding {
		t.Fatal("Clear did not remove mirror entry")
	}
}

func TestIsRiding(t *testing.T) {
	p, _, _, tm, _ := setupProcessor(t)
	if _, riding := p.IsRiding(100); riding {
		t.Fatal("IsRiding true with empty mirror")
	}
	GetRideMirror().Put(tm, 100, RideState{SkillLevel: 9})
	lvl, riding := p.IsRiding(100)
	if !riding || lvl != 9 {
		t.Fatalf("IsRiding = (%d, %v), want (9, true)", lvl, riding)
	}
}
```

Add `"github.com/google/uuid"` to the test file's imports. `effect.Extract(effect.RestModel{Cooldown: 90})` is the established way tests build an `effect.Model` (`RestModel.Cooldown` field verified at `<ch>/data/skill/effect/rest.go:45`; same pattern as `mountEffect` in `<ch>/skill/handler/mount_test.go:77`) — it returns `(Model, error)`, matching the seam signature directly.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./battleship/ -v`
Expected: FAIL — `undefined: store`, `undefined: NewProcessor`, etc.

- [ ] **Step 3: Write the implementation**

Create `<ch>/battleship/processor.go`:

```go
package battleship

import (
	"context"
	"strconv"
	"time"

	"atlas-channel/character"
	"atlas-channel/character/buff"
	charskill "atlas-channel/character/skill"
	dataskill "atlas-channel/data/skill"
	"atlas-channel/data/skill/effect"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	redis "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

const registryNamespace = "battleship-hp"

// fallbackStateTTL bounds orphaned ship-HP entries when the effect-derived
// duration is unavailable (FR-5.2 safety net). Matches the 5221006 WZ buff
// duration (time=2100000 ms = 35 min).
const fallbackStateTTL = 35 * time.Minute

// counterStore is the Redis seam; production is redis.TenantCounter
// (namespace battleship-hp), wired by InitRegistry. Tests inject a fake.
type counterStore interface {
	Set(ctx context.Context, t tenant.Model, key string, value int64, ttl time.Duration) error
	DecrByIfExists(ctx context.Context, t tenant.Model, key string, delta int64, ttl time.Duration) (int64, bool, error)
	Remove(ctx context.Context, t tenant.Model, key string) error
}

var store counterStore

// InitRegistry wires the production Redis-backed ship-HP store. Call once
// from main() after redis.Connect.
func InitRegistry(client *goredis.Client) {
	store = redis.NewTenantCounter(client, registryNamespace)
}

// ShipHP is the ship's full HP pool. It is VERSION-GATED (R-4): the client
// renders the gauge as remaining ÷ its own max, computed by
// get_max_durability_of_vehicle, and that function changed at v87. Using one
// formula everywhere would desync the bar on gms_87/92/95 and jms_185.
//
//	major <  87 (gms_61…gms_84):  200 × (charLevel + 2×SLV − 120)
//	major >= 87 (gms_87…jms_185): 300 × charLevel + 500 × (SLV − 72)
//
// The pre-87 arm is expressed as 400×SLV + max(charLevel−120,0)×200, which is
// algebraically identical to the client's form for charLevel >= 120 — the only
// reachable range, since Battleship is a 4th-job skill — and clamps instead of
// going negative below it. The v87+ arm is floored at 0 for the same reason.
func ShipHP(t tenant.Model, skillLevel byte, charLevel byte) int32 {
	if isPostBigBangDurability(t) {
		hp := 300*int32(charLevel) + 500*(int32(skillLevel)-72)
		if hp < 0 {
			return 0
		}
		return hp
	}
	hp := 400 * int32(skillLevel)
	if charLevel > 120 {
		hp += (int32(charLevel) - 120) * 200
	}
	return hp
}

// isPostBigBangDurability reports whether the tenant's client uses the newer
// get_max_durability_of_vehicle formula. Follows the established version-gate
// idiom (libs/atlas-packet/field/clientbound/set_field.go:48).
func isPostBigBangDurability(t tenant.Model) bool {
	return (t.IsRegion("GMS") && t.MajorAtLeast(87)) || t.IsRegion("JMS")
}

// Collaborator seams (function vars per the skill/handler/common.go
// precedent) so Drain's break flow is unit-testable offline.
var cancelBuffFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId uint32) error {
	return buff.NewProcessor(l, ctx).Cancel(f, characterId, int32(skill2.CorsairBattleshipId))
}

var applyCooldownFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, cooldown uint32, characterId uint32) error {
	return charskill.NewProcessor(l, ctx).ApplyCooldown(f, skill2.CorsairBattleshipId, cooldown)(characterId)
}

var effectFunc = func(l logrus.FieldLogger, ctx context.Context, level byte) (effect.Model, error) {
	return dataskill.NewProcessor(l, ctx).GetEffect(uint32(skill2.CorsairBattleshipId), level)
}

var characterLevelFunc = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) (byte, error) {
	c, err := character.NewProcessor(l, ctx).GetById()(characterId)
	if err != nil {
		return 0, err
	}
	return c.Level(), nil
}

type DrainStatus int

const (
	// DrainNotRiding: character has no active battleship ride on this pod.
	DrainNotRiding DrainStatus = iota
	// DrainSkipped: zero/negative damage, Redis unavailable (degraded mode),
	// or a concurrent drain that arrived after another caller's break.
	DrainSkipped
	// DrainDrained: HP decremented, ship intact; RemainingHP is valid.
	DrainDrained
	// DrainBroke: this drain crossed zero; dismount + cooldown were emitted.
	DrainBroke
)

type DrainResult struct {
	Status      DrainStatus
	RemainingHP int32
}

// Processor owns all battleship ride state (mirror + Redis pool). Handlers
// never touch either directly, so a future move to a service-owned store
// only changes this package (PRD NFR "state home caveat").
type Processor interface {
	// InitShipHP seeds a fresh full pool (always full — never carried over).
	InitShipHP(characterId uint32, skillLevel byte, charLevel byte, ttl time.Duration) error
	// IsRiding reports the active ride and its skill level (mirror read, no I/O).
	IsRiding(characterId uint32) (byte, bool)
	// Drain applies damage to the ship pool (FR-3/FR-4). Exactly one caller
	// per depletion observes DrainBroke.
	Drain(f field.Model, characterId uint32, damage int32) DrainResult
	// Clear removes mirror + Redis state; idempotent.
	Clear(characterId uint32)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx, t: tenant.MustFromContext(ctx)}
}

func shipKey(characterId uint32) string {
	return strconv.FormatUint(uint64(characterId), 10)
}

func (p *ProcessorImpl) InitShipHP(characterId uint32, skillLevel byte, charLevel byte, ttl time.Duration) error {
	if store == nil {
		p.l.Warnf("Battleship store not initialized; ship HP for character [%d] will lazily re-initialize.", characterId)
		return nil
	}
	if ttl <= 0 {
		ttl = fallbackStateTTL
	}
	return store.Set(p.ctx, p.t, shipKey(characterId), int64(ShipHP(p.t, skillLevel, charLevel)), ttl)
}

func (p *ProcessorImpl) IsRiding(characterId uint32) (byte, bool) {
	rs, ok := GetRideMirror().Get(p.t, characterId)
	return rs.SkillLevel, ok
}

func (p *ProcessorImpl) Drain(f field.Model, characterId uint32, damage int32) DrainResult {
	if damage <= 0 {
		return DrainResult{Status: DrainSkipped}
	}
	rs, riding := GetRideMirror().Get(p.t, characterId)
	if !riding {
		return DrainResult{Status: DrainNotRiding}
	}
	if store == nil {
		p.l.Warnf("Battleship store not initialized; drain skipped for character [%d].", characterId)
		return DrainResult{Status: DrainSkipped}
	}
	ttl := rs.StateTTL
	if ttl <= 0 {
		ttl = fallbackStateTTL
	}

	newHp, existed, err := store.DecrByIfExists(p.ctx, p.t, shipKey(characterId), int64(damage), ttl)
	if err != nil {
		// Degraded mode: never fail damage processing over Redis (NFR).
		p.l.WithError(err).Warnf("Battleship drain skipped for character [%d]; Redis unavailable.", characterId)
		return DrainResult{Status: DrainSkipped}
	}
	if !existed {
		// FR-3.3 lazy re-init: state lost (Redis restart / TTL expiry) is
		// never an error and never a stuck ship — re-derive full HP.
		charLevel, lerr := characterLevelFunc(p.l, p.ctx, characterId)
		if lerr != nil {
			p.l.WithError(lerr).Warnf("Battleship lazy re-init failed for character [%d]; drain skipped.", characterId)
			return DrainResult{Status: DrainSkipped}
		}
		full := int64(ShipHP(p.t, rs.SkillLevel, charLevel))
		newHp = full - int64(damage)
		if newHp > 0 {
			if serr := store.Set(p.ctx, p.t, shipKey(characterId), newHp, ttl); serr != nil {
				p.l.WithError(serr).Warnf("Battleship lazy re-init store failed for character [%d]; drain skipped.", characterId)
				return DrainResult{Status: DrainSkipped}
			}
		}
		p.l.Debugf("Battleship ship HP lazily re-initialized for character [%d]: full [%d], damage [%d].", characterId, full, damage)
	}

	if newHp > 0 {
		p.l.Debugf("Battleship drained for character [%d]: damage [%d], remaining [%d].", characterId, damage, newHp)
		return DrainResult{Status: DrainDrained, RemainingHP: int32(newHp)}
	}
	if newHp+int64(damage) > 0 {
		// Exactly-once crossing: only the decrement that moved the value
		// from positive to <= 0 satisfies this predicate (FR-4 / NFR).
		p.breakShip(f, characterId, rs.SkillLevel)
		return DrainResult{Status: DrainBroke}
	}
	// Concurrent drain that landed after another caller's crossing.
	return DrainResult{Status: DrainSkipped}
}

// breakShip performs the FR-4 break: clear state, dismount (foreign cancel
// broadcast comes from the existing atlas-buffs → buff consumer path), and
// apply the 5221006 cooldown with the effect's cooltime via the existing
// atlas-skills SET_COOLDOWN command. Every step is idempotent, so a
// theoretical double-break degrades to a no-op.
func (p *ProcessorImpl) breakShip(f field.Model, characterId uint32, skillLevel byte) {
	p.Clear(characterId)
	if err := cancelBuffFunc(p.l, p.ctx, f, characterId); err != nil {
		p.l.WithError(err).Errorf("Battleship break: unable to cancel mount buff for character [%d].", characterId)
	}
	var cooldown uint32
	if e, err := effectFunc(p.l, p.ctx, skillLevel); err == nil {
		cooldown = e.Cooldown()
	} else {
		p.l.WithError(err).Errorf("Battleship break: unable to load effect level [%d] for character [%d]; cooldown not applied.", skillLevel, characterId)
	}
	if cooldown > 0 {
		if err := applyCooldownFunc(p.l, p.ctx, f, cooldown, characterId); err != nil {
			p.l.WithError(err).Errorf("Battleship break: unable to apply cooldown for character [%d].", characterId)
		}
	}
	p.l.Debugf("Battleship broke for character [%d]: dismounted, cooldown [%d]s.", characterId, cooldown)
}

func (p *ProcessorImpl) Clear(characterId uint32) {
	GetRideMirror().Remove(p.t, characterId)
	if store == nil {
		return
	}
	if err := store.Remove(p.ctx, p.t, shipKey(characterId)); err != nil {
		p.l.WithError(err).Warnf("Battleship state remove failed for character [%d]; TTL will expire it.", characterId)
	}
}
```

- [ ] **Step 4: go.mod + main.go wiring**

In `<ch>/go.mod`, add to the first `require` block:

```
	github.com/Chronicle20/atlas/libs/atlas-redis v0.0.0-00010101000000-000000000000
```

(Both the require and the `replace` already exist — `go.mod:11` and `go.mod:97`; `main` added them. No go.mod edit is needed.) Then, AFTER the import exists in code (this task's processor.go), run `go mod tidy` from `<ch>/` — never before (workspace footgun).

In `<ch>/main.go` `func main()`, after the logger is created (mirror `services/atlas-mounts/atlas.com/mounts/main.go:51-52`):

```go
	rc := atlas.Connect(l)
	battleship.InitRegistry(rc)
```

with import `atlas "github.com/Chronicle20/atlas/libs/atlas-redis"` (the `"atlas-channel/battleship"` import was added in Task 5).

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go mod tidy && go test -race ./battleship/ -v && go build ./...`
Expected: PASS (all processor + mirror tests), build clean

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/battleship/ services/atlas-channel/atlas.com/channel/go.mod services/atlas-channel/atlas.com/channel/go.sum services/atlas-channel/atlas.com/channel/main.go
git commit -m "feat(channel): battleship processor with Redis ship-HP pool and break flow"
```

---

### Task 7: atlas-channel — ride lifecycle hooks (buff events + session destroy)

**Files:**
- Modify: `<ch>/kafka/consumer/buff/consumer.go`
- Modify: `<ch>/session/processor.go` (`Destroy`, lines 330-348)
- Test: `<ch>/kafka/consumer/buff/consumer_test.go` (create)

**Interfaces:**
- Consumes: `buff2.StatChange{Type string, Amount int32}`, `AppliedStatusEventBody{SourceId int32, Level byte, Changes []StatChange, ...}` (`<ch>/kafka/message/buff/kafka.go`); `charconst.TemporaryStatTypeMonsterRiding` = `"MONSTER_RIDING"`; battleship mirror/processor (Tasks 5-6); `dataskill.GetEffect` for the state TTL.
- Produces: `func isBattleshipRide(sourceId int32, changes []buff2.StatChange) bool` (consumer-local helper); `var battleshipStateTTLFunc` seam.

- [ ] **Step 1: Write the failing test**

Create `<ch>/kafka/consumer/buff/consumer_test.go`:

```go
package buff

import (
	"testing"

	buff2 "atlas-channel/kafka/message/buff"
)

func TestIsBattleshipRide(t *testing.T) {
	riding := []buff2.StatChange{{Type: "MONSTER_RIDING", Amount: 1932000}}
	noRiding := []buff2.StatChange{{Type: "WEAPON_DEFENSE", Amount: 10}}
	tests := []struct {
		name     string
		sourceId int32
		changes  []buff2.StatChange
		expected bool
	}{
		{"battleship riding buff", 5221006, riding, true},
		{"battleship without riding change", 5221006, noRiding, false},
		{"other mount riding buff", 1019, riding, false},
		{"cannon is not the mount", 5221007, riding, false},
		{"empty changes", 5221006, nil, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isBattleshipRide(tc.sourceId, tc.changes); got != tc.expected {
				t.Errorf("isBattleshipRide(%d, %v) = %v, want %v", tc.sourceId, tc.changes, got, tc.expected)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./kafka/consumer/buff/ -v`
Expected: FAIL — `undefined: isBattleshipRide`

- [ ] **Step 3: Implement the consumer hooks**

In `<ch>/kafka/consumer/buff/consumer.go`:

Add imports: `"atlas-channel/battleship"`, `dataskill "atlas-channel/data/skill"`, `"time"`, `charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"`, `skillconst "github.com/Chronicle20/atlas/libs/atlas-constants/skill"`.

Add at the bottom of the file:

```go
// isBattleshipRide reports whether a buff status event is the battleship
// mount buff (MONSTER_RIDING sourced from 5221006).
func isBattleshipRide(sourceId int32, changes []buff2.StatChange) bool {
	if sourceId != int32(skillconst.CorsairBattleshipId) {
		return false
	}
	for _, c := range changes {
		if c.Type == string(charconst.TemporaryStatTypeMonsterRiding) {
			return true
		}
	}
	return false
}

// battleshipStateTTLFunc derives the ship-state TTL from the effect's buff
// duration (FR-5.2). Returns 0 on failure — the battleship package falls
// back to its own default. Seam for tests.
var battleshipStateTTLFunc = func(l logrus.FieldLogger, ctx context.Context, level byte) time.Duration {
	e, err := dataskill.NewProcessor(l, ctx).GetEffect(uint32(skillconst.CorsairBattleshipId), level)
	if err != nil || e.Duration() <= 0 {
		if err != nil {
			l.WithError(err).Warnf("Unable to derive battleship state TTL from effect level [%d]; using fallback.", level)
		}
		return 0
	}
	return time.Duration(e.Duration()) * time.Millisecond
}
```

In `handleStatusEventApplied`, at the top of the `IfPresentByCharacterId` callback (before the announce), add — note `t := tenant.MustFromContext(ctx)` must be hoisted to a variable where it is currently called inline at `:120`, inside `handleStatusEventApplied` (declared at `:114`) (`if !sc.IsWorld(tenant.MustFromContext(ctx), e.WorldId)` → `t := tenant.MustFromContext(ctx); if !sc.IsWorld(t, e.WorldId)`):

```go
			// Battleship ride begins: record the pod-local riding truth the
			// damage/attack hot paths read (mirror; FR-3.1/FR-6.2).
			if isBattleshipRide(e.Body.SourceId, e.Body.Changes) {
				battleship.GetRideMirror().Put(t, e.CharacterId, battleship.RideState{
					SkillLevel: e.Body.Level,
					StateTTL:   battleshipStateTTLFunc(l, ctx, e.Body.Level),
				})
			}
```

In `handleStatusEventExpired`, same position inside its callback (hoist `t` identically):

```go
			// Battleship ride ends (manual dismount toggle, server cancel on
			// break, expiry): clear mirror + ship HP state (FR-5.1).
			if isBattleshipRide(e.Body.SourceId, e.Body.Changes) {
				battleship.NewProcessor(l, ctx).Clear(e.CharacterId)
			}
```

- [ ] **Step 4: Implement the session-destroy cleanup**

In `<ch>/session/processor.go` `func (p *ProcessorImpl) Destroy` (`:405`), after `getRegistry().Remove(p.t.Id(), s.SessionId())`, add:

```go
	// Battleship ride state cannot outlive the session: logout, disconnect,
	// timeout, and channel change all funnel here (FR-5.1). No cooldown is
	// applied — break is the only cooldown trigger (FR-4.3).
	if s.CharacterId() != 0 {
		battleship.NewProcessor(p.l, p.ctx).Clear(s.CharacterId())
	}
```

with import `"atlas-channel/battleship"`. (Cycle-safe: `battleship` does not import `session` — verified in Task 6.)

- [ ] **Step 5: Run tests and build**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./kafka/consumer/buff/ ./battleship/ -v && go build ./...`
Expected: PASS, build clean

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/consumer/buff/ services/atlas-channel/atlas.com/channel/session/processor.go
git commit -m "feat(channel): battleship ride lifecycle hooks on buff events and session destroy"
```

---

### Task 8: atlas-channel — cast path (carve-out, mount arm, cooldown rejection)

**Files:**
- Modify: `<ch>/skill/handler/common.go` (cooldown block :93-95, mount gate :99-105)
- Modify: `<ch>/skill/handler/mount.go` (mountDeps + battleship arm + newMountDeps)
- Modify: `<ch>/socket/handler/character_skill_use.go` (cast-while-cooling rejection)
- Test: `<ch>/skill/handler/mount_test.go` (extend), `<ch>/skill/handler/common_cooldown_test.go` (create), `<ch>/socket/handler/character_skill_use_test.go` (create)

**Interfaces:**
- Consumes: Task 1 `skill2.IsBattleshipMountSkill`; Task 3 `atlaspacket.ResolveValue`; Task 4 `writer.TenantWriterOptions`; Task 6 `battleship.NewProcessor(...).InitShipHP`; existing `tamedMountStatups(e, vehicleId)` (mount.go:62 — reused for the vehicle override), `MountBuffDuration` (mount.go:23), `charpkt.CharacterBuffGiveWriter` const.
- Produces:
  - `shouldApplyCastCooldown(e effect.Model, skillId skill2.Id) bool` (common.go helper)
  - `var applyCooldownFunc` seam in common.go (test-swappable, same pattern as `loadCasterFunc`)
  - mountDeps gains: `resolveVehicleId func() (int32, bool)`, `characterLevel func(characterId uint32) (byte, error)`, `initShipHP func(characterId uint32, skillLevel byte, charLevel byte, ttl time.Duration) error`
  - `battleshipCastBlocked(skillId uint32, cooldownExpiresAt time.Time, now time.Time) bool` (character_skill_use.go helper)

- [ ] **Step 1: Write the failing tests**

Create `<ch>/skill/handler/common_cooldown_test.go`:

```go
package handler

import (
	"testing"

	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

func TestShouldApplyCastCooldown(t *testing.T) {
	tests := []struct {
		name     string
		cooldown uint32
		skillId  skill2.Id
		expected bool
	}{
		{"battleship exempt despite cooltime (FR-2.3)", 90, skill2.CorsairBattleshipId, false},
		{"other skill with cooltime applies", 90, skill2.PriestDispelId, true},
		{"other skill without cooltime skips", 0, skill2.PriestDispelId, false},
		{"battleship without cooltime skips", 0, skill2.CorsairBattleshipId, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldApplyCastCooldown(tc.cooldown, tc.skillId); got != tc.expected {
				t.Errorf("shouldApplyCastCooldown(%d, %d) = %v, want %v", tc.cooldown, tc.skillId, got, tc.expected)
			}
		})
	}
}
```

Create `<ch>/socket/handler/character_skill_use_test.go`:

```go
package handler

import (
	"testing"
	"time"
)

func TestBattleshipCastBlocked(t *testing.T) {
	now := time.Now()
	future := now.Add(30 * time.Second)
	past := now.Add(-30 * time.Second)
	tests := []struct {
		name              string
		skillId           uint32
		cooldownExpiresAt time.Time
		expected          bool
	}{
		{"battleship on cooldown blocked (FR-2.4)", 5221006, future, true},
		{"battleship cooldown expired allowed", 5221006, past, false},
		{"battleship never cooled allowed", 5221006, time.Time{}, false},
		{"other skill on cooldown not blocked here", 2311001, future, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := battleshipCastBlocked(tc.skillId, tc.cooldownExpiresAt, now); got != tc.expected {
				t.Errorf("battleshipCastBlocked(%d, %v) = %v, want %v", tc.skillId, tc.cooldownExpiresAt, got, tc.expected)
			}
		})
	}
}
```

Extend `<ch>/skill/handler/mount_test.go`: add the three new fields to `recordingDeps` and its `mountDeps()` method, then add battleship cases:

```go
// Add to recordingDeps struct:
	vehicleId       int32
	vehicleOk       bool
	charLevel       byte
	charLevelErr    error
	initCalled      bool
	initSkillLevel  byte
	initCharLevel   byte
	initTTL         time.Duration

// Add to (d *recordingDeps) mountDeps() return value:
		resolveVehicleId: func() (int32, bool) {
			return d.vehicleId, d.vehicleOk
		},
		characterLevel: func(characterId uint32) (byte, error) {
			return d.charLevel, d.charLevelErr
		},
		initShipHP: func(characterId uint32, skillLevel byte, charLevel byte, ttl time.Duration) error {
			d.initCalled = true
			d.initSkillLevel = skillLevel
			d.initCharLevel = charLevel
			d.initTTL = ttl
			return nil
		},

const battleshipSkillId = uint32(skill2.CorsairBattleshipId)

// battleshipInfo mirrors mountInfo (mount_test.go:73) but with a settable level.
func battleshipInfo(level byte) packetmodel.SkillUsageInfo {
	return packetmodel.NewSkillUsageInfoBuilder().SetSkillId(battleshipSkillId).SetSkillLevel(level).Build()
}

// battleshipEffect mirrors mountEffect (mount_test.go:77): Duration 2100000 ms
// (the WZ buff time) and a MONSTER_RIDING statup carrying atlas-data's
// skill-id placeholder amount, which the arm must override.
func battleshipEffect() effect.Model {
	e, err := effect.Extract(effect.RestModel{Statups: vehicleStatup(int32(battleshipSkillId)), Duration: 2100000})
	if err != nil {
		panic(err)
	}
	return e
}

func TestHandleMountBattleshipApplies(t *testing.T) {
	d := &recordingDeps{vehicleId: 1932000, vehicleOk: true, charLevel: 150}

	if err := HandleMount(logrus.New(), field.Model{}, 999, battleshipInfo(7), battleshipEffect(), d.mountDeps()); err != nil {
		t.Fatalf("HandleMount: %v", err)
	}
	if !d.applyCalled {
		t.Fatal("expected applyBuff to be called")
	}
	if d.applyAmount != 1932000 {
		t.Fatalf("MONSTER_RIDING amount = %d, want config-resolved 1932000", d.applyAmount)
	}
	if d.applyDur != MountBuffDuration {
		t.Fatalf("duration = %d, want MountBuffDuration", d.applyDur)
	}
	if !d.initCalled || d.initSkillLevel != 7 || d.initCharLevel != 150 {
		t.Fatalf("initShipHP = (called %v, skill %d, char %d), want (true, 7, 150)", d.initCalled, d.initSkillLevel, d.initCharLevel)
	}
	if d.initTTL != 2100000*time.Millisecond {
		t.Fatalf("init TTL = %v, want 35m from effect duration", d.initTTL)
	}
}

func TestHandleMountBattleshipAbortsOnVehicleMiss(t *testing.T) {
	d := &recordingDeps{vehicleOk: false, charLevel: 150}
	if err := HandleMount(logrus.New(), field.Model{}, 999, battleshipInfo(7), battleshipEffect(), d.mountDeps()); err != nil {
		t.Fatalf("HandleMount: %v", err)
	}
	if d.applyCalled || d.initCalled {
		t.Fatal("resolve miss must abort: no buff, no HP state")
	}
}

func TestHandleMountBattleshipToggleDismounts(t *testing.T) {
	d := &recordingDeps{mounted: true, vehicleId: 1932000, vehicleOk: true, charLevel: 150}
	if err := HandleMount(logrus.New(), field.Model{}, 999, battleshipInfo(7), battleshipEffect(), d.mountDeps()); err != nil {
		t.Fatalf("HandleMount: %v", err)
	}
	if d.cancelCount != 1 || d.applyCalled || d.initCalled {
		t.Fatalf("toggle must only cancel: cancels %d, applied %v, init %v", d.cancelCount, d.applyCalled, d.initCalled)
	}
}
```

(All construction goes through the file's existing verified helpers: `logrus.New()` logger, `vehicleStatup(...)` for the MONSTER_RIDING statup, `effect.Extract(effect.RestModel{...})`, `packetmodel.NewSkillUsageInfoBuilder()` with `SetSkillLevel(byte)`. The existing `recordingDeps` already records `applyAmount` (first statup's amount), `applyDur`, and `cancelCount` — reuse them. Add `"time"` to the test file's imports for the new fields.)

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./skill/handler/ ./socket/handler/ -run 'TestShouldApplyCastCooldown|TestBattleshipCastBlocked|TestHandleMountBattleship' -v`
Expected: FAIL — undefined helpers / missing deps fields

- [ ] **Step 3: Implement common.go changes**

In `<ch>/skill/handler/common.go`:

Add near the other seams (after `cancelStatusFunc`):

```go
// applyCooldownFunc is the cast-time cooldown emit seam tests can replace.
var applyCooldownFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, skillId skill2.Id, cooldown uint32, characterId uint32) error {
	return skill.NewProcessor(l, ctx).ApplyCooldown(f, skillId, cooldown)(characterId)
}

// shouldApplyCastCooldown gates the generic cast-time cooldown. Battleship
// (5221006) is exempt: its cooldown applies only when the ship breaks, never
// on cast (FR-2.3/FR-4.3) — the WZ cooltime would otherwise fire here.
func shouldApplyCastCooldown(cooldown uint32, skillId skill2.Id) bool {
	return cooldown > 0 && !skill2.IsBattleshipMountSkill(skillId)
}
```

Replace lines 93-105 (`if e.Cooldown() > 0 {...}` through the mount gate) with:

```go
			skillId := skill2.Id(info.SkillId())
			if shouldApplyCastCooldown(e.Cooldown(), skillId) {
				_ = applyCooldownFunc(l, ctx, f, skillId, e.Cooldown(), characterId)
			}
			// Mount toggle (tamed + skill-only + battleship). Runs BEFORE the
			// generic buff apply and short-circuits it: mounts apply
			// MONSTER_RIDING with a MaxInt32 duration and a vehicle-id amount,
			// or cancel on re-cast.
			if skill2.IsTamedMountSkill(skillId) || isSkillOnlyMount(skillId, info.SkillLevel()) || skill2.IsBattleshipMountSkill(skillId) {
				if err := HandleMount(l, f, characterId, info, e, newMountDeps(l, ctx)); err != nil {
					l.WithError(err).Errorf("Mount toggle failed for character [%d] skill [%d].", characterId, info.SkillId())
				}
				return nil
			}
```

(The original `skillId :=` declaration — now at `common.go:143` — is subsumed; do not redeclare.)

- [ ] **Step 4: Implement mount.go changes**

In `<ch>/skill/handler/mount.go`:

Add imports: `"time"`, `"atlas-channel/battleship"`, `atlaspacket "github.com/Chronicle20/atlas/libs/atlas-packet"`, `charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"`, `"atlas-channel/socket/writer"`, `tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"`.

Extend `mountDeps` (after `cancelBuff`):

```go
	// resolveVehicleId resolves the battleship vehicle item id from the
	// tenant's CharacterBuffGive writer options table (DOM-25). ok=false on
	// any miss — the mount is aborted rather than sent with a guess.
	resolveVehicleId func() (int32, bool)
	// characterLevel loads the caster's current character level (ship HP formula input).
	characterLevel func(characterId uint32) (byte, error)
	// initShipHP seeds the fresh full ship HP pool (battleship processor).
	initShipHP func(characterId uint32, skillLevel byte, charLevel byte, ttl time.Duration) error
```

In `HandleMount`, insert the battleship arm between Case 1 (the `if mounted` block) and Case 5 (the `isSkillOnlyMount` block):

```go
	// Battleship (5221006): the vehicle id is a client wire value resolved
	// from tenant configuration (DOM-25) and injected as the MONSTER_RIDING
	// amount (atlas-data emits a skill-id placeholder by design). A fresh
	// full ship HP pool is seeded on every mount (FR-2.2); no cooldown is
	// applied here — break is the only cooldown trigger (FR-2.3).
	if skill2.IsBattleshipMountSkill(skillId) {
		vehicleId, ok := deps.resolveVehicleId()
		if !ok {
			l.Errorf("Character [%d] battleship mount aborted: vehicle id unresolved from tenant config.", characterId)
			return nil
		}
		charLevel, err := deps.characterLevel(characterId)
		if err != nil {
			l.WithError(err).Errorf("Character [%d] battleship mount aborted: unable to load character level.", characterId)
			return err
		}
		if err := deps.applyBuff(f, characterId, sourceId, info.SkillLevel(), MountBuffDuration, tamedMountStatups(e, vehicleId)); err != nil {
			return err
		}
		if err := deps.initShipHP(characterId, info.SkillLevel(), charLevel, time.Duration(e.Duration())*time.Millisecond); err != nil {
			// Non-fatal: the pool lazily re-initializes to full on first
			// drain (FR-3.3), which matches the reset semantics.
			l.WithError(err).Warnf("Character [%d] battleship ship HP init failed; will lazily re-initialize.", characterId)
		}
		return nil
	}
```

In `newMountDeps`, add the production implementations:

```go
		resolveVehicleId: func() (int32, bool) {
			t := tenant.MustFromContext(ctx)
			opts, ok := writer.TenantWriterOptions(t.Id(), charpkt.CharacterBuffGiveWriter)
			if !ok {
				l.Errorf("Writer options for [%s] missing; cannot resolve battleship vehicle id.", charpkt.CharacterBuffGiveWriter)
				return 0, false
			}
			v, ok := atlaspacket.ResolveValue(l, opts, "vehicles", "CORSAIR_BATTLESHIP")
			return int32(v), ok
		},
		characterLevel: func(characterId uint32) (byte, error) {
			c, err := cp.GetById()(characterId)
			if err != nil {
				return 0, err
			}
			return c.Level(), nil
		},
		initShipHP: func(characterId uint32, skillLevel byte, charLevel byte, ttl time.Duration) error {
			return battleship.NewProcessor(l, ctx).InitShipHP(characterId, skillLevel, charLevel, ttl)
		},
```

Also update the `HandleMount` doc comment's case list to mention the battleship arm.

- [ ] **Step 5: Implement the cast rejection**

In `<ch>/socket/handler/character_skill_use.go`, after the skill-level validation block (the `if sm.Id() == 0 || sm.Level() == 0 || sm.Level() != sui.SkillLevel()` block, which now **starts** at `:70`), add:

```go
		// Battleship post-break cooldown is enforced server-side: the client
		// greys the icon, but a packet-editing client must not remount a
		// broken ship (FR-2.4). Zero extra round-trips — CooldownExpiresAt is
		// decorated onto the already-loaded skill model.
		if battleshipCastBlocked(sui.SkillId(), sm.CooldownExpiresAt(), time.Now()) {
			l.Debugf("Character [%d] attempting to cast battleship while on post-break cooldown (expires [%s]).", s.CharacterId(), sm.CooldownExpiresAt())
			err = enableActions(l)(ctx)(wp)(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to write [%s] for character [%d].", statpkt.StatChangedWriter, s.CharacterId())
			}
			return
		}
```

and at the bottom of the file:

```go
// battleshipCastBlocked reports whether a 5221006 cast must be rejected
// because the post-break cooldown is still running (FR-2.4). Scoped to
// battleship: a generic cast-time cooldown gate is out of scope here.
func battleshipCastBlocked(skillId uint32, cooldownExpiresAt time.Time, now time.Time) bool {
	return skill.Id(skillId) == skill.CorsairBattleshipId && now.Before(cooldownExpiresAt)
}
```

Add `"time"` to the file's imports.

- [ ] **Step 6: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./skill/handler/ ./socket/handler/ -v && go build ./...`
Expected: PASS (new tests + all existing mount/common tests unchanged), build clean

- [ ] **Step 7: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/skill/handler/ services/atlas-channel/atlas.com/channel/socket/handler/character_skill_use.go services/atlas-channel/atlas.com/channel/socket/handler/character_skill_use_test.go
git commit -m "feat(channel): battleship cast path - mount arm, cooldown carve-out, cast rejection"
```

---

### Task 9: atlas-channel — damage drain + HP gauge announce

**Files:**
- Modify: `<ch>/socket/handler/character_damage.go`
- Test: `<ch>/socket/handler/character_damage_test.go` (create)

**Interfaces:**
- Consumes: Task 6 `battleship.NewProcessor(l, ctx).Drain(f, characterId, damage) DrainResult`; Task 4 `writer.TenantWriterOptions`; Task 3 `atlaspacket.ResolveValue`; existing `charpkt.NewCharacterSkillCooldown(skillId uint32, cooldown uint16)` + `charpkt.CharacterSkillCooldownWriter`; `p.Damage() int32`.
- Produces: `gaugeCooldownValue(remaining int32) uint16` (clamp helper), `announceShipHpGauge(...)` (file-local).

- [ ] **Step 1: Write the failing test**

Create `<ch>/socket/handler/character_damage_test.go`:

```go
package handler

import (
	"math"
	"testing"
)

func TestGaugeCooldownValue(t *testing.T) {
	tests := []struct {
		name      string
		remaining int32
		expected  uint16
	}{
		{"normal", 8500, 8500},
		{"formula max fits (v87+ arm, SLV 10 @ 200)", 29000, 29000},
		{"defensive clamp above uint16", math.MaxUint16 + 1, math.MaxUint16},
		{"defensive floor below zero", -5, 0},
		{"one", 1, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := gaugeCooldownValue(tc.remaining); got != tc.expected {
				t.Errorf("gaugeCooldownValue(%d) = %d, want %d", tc.remaining, got, tc.expected)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run TestGaugeCooldownValue -v`
Expected: FAIL — `undefined: gaugeCooldownValue`

- [ ] **Step 3: Implement the drain integration**

In `<ch>/socket/handler/character_damage.go`:

Add imports: `"atlas-channel/battleship"`, `"math"`, `atlaspacket "github.com/Chronicle20/atlas/libs/atlas-packet"`, `tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"`.

Delete the line `// TODO decrease battleship hp` (now `:32` — it is the last of a run of TODO comments, `:23`-`:32`). After the foreign-damage announce block and immediately before the `ChangeHP` call (`:44`), add:

```go
		// Battleship: damage taken while riding drains the ship's parallel
		// HP pool (FR-3.1); the character HP change below is unaffected. A
		// non-breaking drain reports remaining ship HP via the skill-cooldown
		// packet carrying the config-resolved gauge pseudo-skill id
		// (FR-3.4 / DOM-25). Break (dismount + cooldown) is handled inside
		// Drain; the resulting client packets flow through the existing buff
		// and skill consumers.
		res := battleship.NewProcessor(l, ctx).Drain(s.Field(), s.CharacterId(), p.Damage())
		if res.Status == battleship.DrainDrained {
			announceShipHpGauge(l, ctx, wp, s, res.RemainingHP)
		}
```

At the bottom of the file:

```go
// announceShipHpGauge sends the client's ship HP gauge: the skill-cooldown
// packet with the config-resolved battleship gauge pseudo-skill id and the
// remaining ship HP as the cooldown value (verified client behavior on
// v83/v84/v87/v95/jms185 — design §1.1). On any resolve miss the packet is
// skipped entirely (fail-loud, never send a guessed wire value).
func announceShipHpGauge(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, s session.Model, remaining int32) {
	t := tenant.MustFromContext(ctx)
	opts, ok := writer.TenantWriterOptions(t.Id(), charpkt.CharacterSkillCooldownWriter)
	if !ok {
		l.Errorf("Writer options for [%s] missing; battleship HP gauge not sent.", charpkt.CharacterSkillCooldownWriter)
		return
	}
	gaugeId, ok := atlaspacket.ResolveValue(l, opts, "skills", "BATTLESHIP_HP_GAUGE")
	if !ok {
		return
	}
	if err := session.Announce(l)(ctx)(wp)(charpkt.CharacterSkillCooldownWriter)(charpkt.NewCharacterSkillCooldown(gaugeId, gaugeCooldownValue(remaining)).Encode)(s); err != nil {
		l.WithError(err).Errorf("Unable to announce battleship HP gauge to character [%d].", s.CharacterId())
	}
}

// gaugeCooldownValue clamps remaining ship HP into the packet's uint16
// field. Battleship is maxLevel 10 on every version (R-5), so the ceiling is
// the v87+ arm at SLV 10 / charLevel 200 = 29 000 — well inside uint16. The
// clamp is purely defensive.
func gaugeCooldownValue(remaining int32) uint16 {
	if remaining < 0 {
		return 0
	}
	if remaining > math.MaxUint16 {
		return math.MaxUint16
	}
	return uint16(remaining)
}
```

- [ ] **Step 4: Run tests and build**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./socket/handler/ -run TestGaugeCooldownValue -v && go build ./... && go test -race ./...`
Expected: PASS, build clean, full module tests clean

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_damage.go services/atlas-channel/atlas.com/channel/socket/handler/character_damage_test.go
git commit -m "feat(channel): battleship HP drain and gauge on damage taken"
```

---

### Task 10: atlas-channel — Cannon/Torpedo riding gate

**Files:**
- Modify: `<ch>/socket/handler/character_attack_common.go`
- Test: `<ch>/socket/handler/character_attack_battleship_gate_test.go` (create)

**Interfaces:**
- Consumes: Task 5 mirror (`battleship.GetRideMirror().Get`); constants `skill3.CorsairBattleshipCannonId` / `skill3.CorsairBattleshipTorpedoId` (in this file, `skill3` is already the alias for `libs/atlas-constants/skill` — imported at `:27`).
- Produces: `battleshipAttackPermitted(t tenant.Model, characterId uint32, skillId skill3.Id) bool`.

- [ ] **Step 1: Write the failing test**

Create `<ch>/socket/handler/character_attack_battleship_gate_test.go`:

```go
package handler

import (
	"testing"

	"atlas-channel/battleship"

	skill3 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
)

func gateTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tm
}

func TestBattleshipAttackPermitted(t *testing.T) {
	tm := gateTenant(t)
	other := gateTenant(t)
	t.Cleanup(func() {
		battleship.GetRideMirror().EvictTenant(tm.Id())
		battleship.GetRideMirror().EvictTenant(other.Id())
	})
	battleship.GetRideMirror().Put(tm, 100, battleship.RideState{SkillLevel: 7})

	tests := []struct {
		name        string
		t           tenant.Model
		characterId uint32
		skillId     skill3.Id
		expected    bool
	}{
		{"cannon while riding", tm, 100, skill3.CorsairBattleshipCannonId, true},
		{"torpedo while riding", tm, 100, skill3.CorsairBattleshipTorpedoId, true},
		{"cannon on foot rejected (FR-6.1)", tm, 200, skill3.CorsairBattleshipCannonId, false},
		{"torpedo on foot rejected", tm, 200, skill3.CorsairBattleshipTorpedoId, false},
		{"tenant isolation", other, 100, skill3.CorsairBattleshipCannonId, false},
		{"unrelated skill always passes", tm, 200, skill3.CorsairRapidFireId, true},
		{"battleship mount skill itself is not gated", tm, 200, skill3.CorsairBattleshipId, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := battleshipAttackPermitted(tc.t, tc.characterId, tc.skillId); got != tc.expected {
				t.Errorf("battleshipAttackPermitted(%d, %d) = %v, want %v", tc.characterId, tc.skillId, got, tc.expected)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run TestBattleshipAttackPermitted -v`
Expected: FAIL — `undefined: battleshipAttackPermitted`

- [ ] **Step 3: Implement the gate**

In `<ch>/socket/handler/character_attack_common.go`:

Add import `"atlas-channel/battleship"`.

Inside `processAttack` (declared at `:636`), immediately after the skill-ownership rejection block — which ends at `:660` with `return session.NewProcessor(l, ctx).Destroy(s)` and its closing `}` — add:

```go
						// Battleship Cannon/Torpedo are usable only while
						// riding the battleship (FR-6.1). Soft rejection (no
						// costs, no damage, no broadcast): a briefly desynced
						// legitimate client — e.g. the cast→BUFF_APPLIED
						// mirror window — must not be disconnected. Pure map
						// read: zero I/O in the attack hot path (FR-6.2).
						if !battleshipAttackPermitted(tenant.MustFromContext(ctx), s.CharacterId(), skill3.Id(ai.SkillId())) {
							l.WithFields(logrus.Fields{
								"character_id": s.CharacterId(),
								"skill_id":     ai.SkillId(),
							}).Debug("battleship_attack_rejected_not_riding")
							return nil
						}
```

At the bottom of the file:

```go
// battleshipAttackPermitted gates the battleship-dependent attack skills
// (Cannon 5221007, Torpedo 5221008) on an active battleship ride. Every
// attack entry point (melee/ranged/magic/energy/touch) funnels through
// processAttack, so this single gate covers them all. Skills outside the
// pair always pass.
func battleshipAttackPermitted(t tenant.Model, characterId uint32, skillId skill3.Id) bool {
	if !skill3.Is(skillId, skill3.CorsairBattleshipCannonId, skill3.CorsairBattleshipTorpedoId) {
		return true
	}
	_, riding := battleship.GetRideMirror().Get(t, characterId)
	return riding
}
```

(`tenant` and `logrus` are already imported in this file.)

- [ ] **Step 4: Run tests and build**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./socket/handler/ -v && go build ./...`
Expected: PASS (including all pre-existing attack tests), build clean

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go services/atlas-channel/atlas.com/channel/socket/handler/character_attack_battleship_gate_test.go
git commit -m "feat(channel): gate Cannon/Torpedo on active battleship ride"
```

---

### Task 11: seed templates — config tables, v92 writers, and the 4 missing cast/damage handler pairs

**Files:**
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_61_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_72_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_79_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_83_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_84_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_87_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_92_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_95_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_jms_185_1.json`

**NOT modified:** `template_gms_12_1.json` and `template_gms_48_1.json`. Skill 5221006
does not exist in either client (R-3) — `gms_v48` returns HTTP 404 from atlas-data and
its binary has zero references to `0x4FAA8E`/`0x4FAE6F`; `gms_12` predates it and has no
live tenant. Battleship is **n/a** on both. Do not add battleship config there, and do
not treat their absence as an unfinished gap.

**Interfaces:**
- Produces: per-tenant writer options consumed by Tasks 8/9 —
  - `CharacterSkillCooldown` writer entry gains `"options": {"skills": {"BATTLESHIP_HP_GAUGE": 5221999}}`
  - `CharacterBuffGive` writer entry gains `"options": {"vehicles": {"CORSAIR_BATTLESHIP": 1932000}}`
- Produces: the two inbound opcodes the feature's cast and drain paths need on the four
  versions currently missing them (R-2).
- Ground truth for `5221999`: verified in **every** IDB including v92 (R-12) —
  `CUserLocal::OnSkillCooltimeSet` compares the decoded skill id against `0x4FAE6F`
  (v61 `0x7ADDA5`, v72 `0x86851A`, v79 `0x8B3EC5`, v83 `0x95BEBB`, v84 `0x99A14F`,
  v87 `0x9DE5A0`, v92 `0x8EF260`, v95 `0x908C0F`, jms185 `0xA274D4`).
- Ground truth for `1932000` = `0x1D7AE0` (R-10) — the BATTLESHIP vehicle item id.

**Guard:** `tools/template-opcode-order-guard.sh` requires both the `handlers` and
`writers` arrays to be in **strictly ascending `opCode` order** (R-11). Every entry
added below goes at its **sorted position** — never appended to the end, never parked
next to a semantically-related neighbour.

**Validator:** every handler entry needs a `"validator"`; one without it is silently
dropped at listener build. All five already-wired versions use `"LoggedInValidator"`
for both handlers, and `"services": ["channel"]`.

- [ ] **Step 1: Add the writer options to the eight templates that already route the writers**

In `template_gms_61_1.json`, `_gms_72_`, `_gms_79_`, `_gms_83_`, `_gms_84_`, `_gms_87_`,
`_gms_95_` and `_jms_185_`, locate the two writer entries in `socket.writers` and add the
options key. Do **not** change any opCode. Shown for v83:

```json
{"opCode": "0x20", "writer": "CharacterBuffGive", "options": {"vehicles": {"CORSAIR_BATTLESHIP": 1932000}}}
```

```json
{"opCode": "0xEA", "writer": "CharacterSkillCooldown", "options": {"skills": {"BATTLESHIP_HP_GAUGE": 5221999}}}
```

Existing opCodes per file (verified post-merge 2026-07-28 — re-read while editing, do not
trust from memory):

| template | `CharacterBuffGive` | `CharacterSkillCooldown` |
|---|---|---|
| gms_61 | `0x1D` | `0xAD` |
| gms_72 | `0x1D` | `0xCE` |
| gms_79 | `0x1D` | `0xD4` |
| gms_83 | `0x20` | `0xEA` |
| gms_84 | `0x20` | `0xF0` |
| gms_87 | `0x20` | `0xFA` |
| gms_95 | `0x1F` | `0x114` |
| jms_185 | `0x1E` | `0xFB` |

Preserve each file's existing JSON entry formatting.

- [ ] **Step 2: Wire the five missing v92 writers**

`template_gms_92_1.json` routes **none** of `CharacterBuffGive`, `CharacterBuffCancel`,
`CharacterBuffGiveForeign`, `CharacterBuffCancelForeign`, `CharacterSkillCooldown`
(re-verified post-merge; 60 writers, 45 handlers). Insert all five **at their sorted
positions** in `socket.writers`:

```json
{"opCode": "0x21", "writer": "CharacterBuffGive", "options": {"vehicles": {"CORSAIR_BATTLESHIP": 1932000}}},
{"opCode": "0x22", "writer": "CharacterBuffCancel"},
{"opCode": "0xE3", "writer": "CharacterBuffGiveForeign"},
{"opCode": "0xE4", "writer": "CharacterBuffCancelForeign"},
{"opCode": "0x112", "writer": "CharacterSkillCooldown", "options": {"skills": {"BATTLESHIP_HP_GAUGE": 5221999}}}
```

These five opcodes come from `docs/packets/MapleStory Ops - ClientBound.csv` — the only
v92 clientbound source, since v92 is not a coverage-matrix column and has no
`docs/packets/registry/gms_v92.yaml`. They were cross-validated by checking that the same
CSV rows' v83/v87/v95/jms185 values match those versions' already-verified template
entries exactly. **A v92 IDB now exists** (R-12); if you want IDA confirmation rather than
CSV provenance, take it — but the CSV cross-validation is what this task relies on.

- [x] **Step 3: Derive v92's `SPECIAL_MOVE` and `TAKE_DAMAGE` serverbound opcodes — DONE 2026-07-28**

v92 has no registry column, so these two were derived directly from
`GMS_v92_1_DEVM.exe`. **Results:**

| op | handler | v92 opcode |
|---|---|---|
| `SPECIAL_MOVE` | `CharacterUseSkillHandle` | **`0x66`** (102) |
| `TAKE_DAMAGE` | `CharacterDamageHandle` | **`0x35`** (53) |

**Why derivation was required:** v92's serverbound table cannot be interpolated from its
neighbours. Measured v92-minus-v95 deltas across the already-wired handlers are `+2` at
`0x2A` (ChannelChange), `+1` through the `0x36`–`0x59` band (ChatGeneral, CashItemUse,
InventoryMove, SkillBookUse, MountFood, OwlAction, OwlWarp), `0` at `0x5B`
(TeleportRockUse), and `-1` to `-7` above that (CharacterInfoRequest, MonsterMovement,
NPCAction). The steps are op-insertion artifacts — e.g. v95 added
`USE_SHOP_SCANNER_ITEM` at `0x5A`, which v92 lacks — not a uniform shift. Both target
opcodes sit right where the delta changes.

**Method** (validated end-to-end on v95 before being applied to v92): the client *sends*
these ops, so the opcode is the immediate passed to the `COutPacket` constructor in the
sending function. Find the sender, read the `push <imm>` preceding the ctor call.

Validation on v95, where the registry gives the answer independently:

| v95 function | ctor site | preceding push | registry |
|---|---|---|---|
| `CUserLocal::SetDamaged` @ `0x9343C0` | `0x936250` | `0x93624A: 6A 34  push 34h` | `TAKE_DAMAGE` = 52 ✓ |
| `CUserLocal::DoActiveSkill_Heal` @ `0x93A830` | `0x93A99D` | `0x93A997: 6A 67  push 67h` | `SPECIAL_MOVE` = 103 ✓ |
| `CUserLocal::DoActiveSkill_Prepare` @ `0x941710` | `0x94215E` | `0x942158: 6A 69  push 69h` | `SKILL_EFFECT` = 105 ✓ |

**v92 evidence:**

- `TAKE_DAMAGE` — `SetDamaged_CUserLocal` @ `0x913BB0` (size `0x1ED0`). `COutPacket` ctor
  at `0x915795`, preceded at `0x91578F` by `6A 35  push 35h`. → **`0x35`**.
  The function was located by constant-fingerprinting rather than by symbol: the skill id
  `0x404139` appears three times inside v95's `SetDamaged` at `0x93539D`/`0x9353C4`/
  `0x935C69`, and v92 has an identically-spaced cluster (`0x27` between the first two) at
  `0x914A63`/`0x914A8A`/`0x9152B1`.
- `SPECIAL_MOVE` — `DoActiveSkill_Heal_CUserLocal` @ `0x91B630` (renamed in the IDB during
  this derivation). `COutPacket` ctor at `0x91B7C2`, preceded by `push 66h`. → **`0x66`**.
  Structural identity with v95's `DoActiveSkill_Heal` is unambiguous: same
  `ExclRequestSent` guard chain, the same `nSkillID / 1000 % 10` test with the `== 9`
  rejection, the same `GetOneTimeAction(...) > -1` bail, the same `IsOnLadder`/`IsOnRope`
  pair, the same `Encode4(update_time)` + `Encode4(skillID)` + `Encode1(SLV)` opening, and
  the same trailing `ShowSkillEffect(..., 6, 0, 0x7FFFFFFF)`.

**Three independent cross-checks on `0x66`:**

1. A second v92 skill-use sender, `sub_91B8C0`, also constructs `COutPacket(0x66)` — two
   distinct `CUserLocal` paths emitting the same op is what `SPECIAL_MOVE` looks like.
2. `SKILL_EFFECT` sits exactly two opcodes above `SPECIAL_MOVE` in **every** other version.
   v92's `DoActiveSkill_Prepare_CUserLocal` @ `0x91DF00` (also renamed) pushes `0x68` at
   `0x91E940` = `0x66` + 2 ✓. Its ctor offset within the function is `+0xA46`, against
   `+0xA4E` in v95 — the same code shape.
3. Ordering sanity for `0x35`: v95 runs `TAKE_DAMAGE 0x34`, one unused slot, `GENERAL_CHAT
   0x36`. v92 runs `TAKE_DAMAGE 0x35`, one slot, and its template already routes
   `CharacterChatGeneralHandle` at `0x37` — the same one-op gap, shifted by one.

**Symbols named in the IDB** (per the project's RE discipline): `0x91B630` →
`DoActiveSkill_Heal_CUserLocal`, `0x91DF00` → `DoActiveSkill_Prepare_CUserLocal`.
`0x913BB0` was already named `SetDamaged_CUserLocal`. Evidence comments were set at
`0x91578F`, `0x91B630` and `0x91E940`.

**Trap for anyone re-treading this:** do not trust a v92 symbol name propagated from v95
without checking the opcode. `sub_91BCD0` is named `SendSkillEffectRequest` and pushes
`0x48` — that is **not** `SKILL_EFFECT`; v95's own `SendSkillEffectRequest` pushes `0x47`,
a different op entirely. The name is accurate; the inference "this is the skill-prepare
op" is not.

**Payload note (not opcode-affecting, but check before declaring v92 done):** v92's
`SPECIAL_MOVE` body is `Encode4(time), Encode4(skillId), Encode1(SLV), Encode2(?),
Encode2(?)` where v95's is `Encode4(time), Encode4(skillId), Encode1(SLV),
Encode1(FindParty), Encode2(0)` — a 5-byte trailer against v95's 4-byte one. Wiring the
opcode gets the packet routed; confirm `SkillUsageInfo` decoding actually matches this
shape on v92 before reporting the cast path working there. Flag it if it does not — do not
paper over it.

- [ ] **Step 4: Wire the missing `CharacterUseSkillHandle` / `CharacterDamageHandle` pairs**

`gms_87`, `gms_92`, `gms_95` and `jms_185` route **neither** handler (R-2), so on those
versions the cast never arrives and damage is never processed — battleship is unreachable
there regardless of any config this task adds. This is a pre-existing tenant-wide gap that
blocks far more than battleship; the owner scoped it into this task (R-8.1).

Insert into `socket.handlers` at each file's **sorted position**:

| template | `CharacterUseSkillHandle` (`SPECIAL_MOVE`) | `CharacterDamageHandle` (`TAKE_DAMAGE`) | source |
|---|---|---|---|
| gms_87 | `0x5E` (94) | `0x32` (50) | `docs/packets/registry/gms_v87.yaml` |
| gms_95 | `0x67` (103) | `0x34` (52) | `docs/packets/registry/gms_v95.yaml` |
| jms_185 | `0x56` (86) | `0x27` (39) | `docs/packets/registry/jms_v185.yaml` |
| gms_92 | `0x66` (102) | `0x35` (53) | v92 IDB — derived in Step 3 |

Entry shape (matching the five already-wired versions exactly):

```json
{"opCode": "0x5E", "validator": "LoggedInValidator", "handler": "CharacterUseSkillHandle", "services": ["channel"]},
{"opCode": "0x32", "validator": "LoggedInValidator", "handler": "CharacterDamageHandle", "services": ["channel"]}
```

The three registry-sourced rows are not taken on faith — each was cross-validated against
already-verified entries in the same file:

- Registry `SPECIAL_MOVE` for v83 is 91 = `0x5B`, which is exactly what
  `template_gms_83_1.json` already routes for `CharacterUseSkillHandle`. Registry
  `TAKE_DAMAGE` for v83 is 48 = `0x30`, matching that file's `CharacterDamageHandle`.
  Same agreement holds for gms_61 (`0x53`/`0x2D`), gms_72 (`0x5A`/`0x2F`) and gms_79
  (`0x59`/`0x2E`). The op→handler mapping is therefore established, not assumed.
- In every version `SKILL_EFFECT` sits exactly two opcodes above `SPECIAL_MOVE`, and each
  target file already routes `CharacterSkillPrepareHandle` at that value: v87 `0x60`
  (= `0x5E`+2 ✓), v95 `0x69` (= `0x67`+2 ✓), jms185 `0x58` (= `0x56`+2 ✓).
- v95's `TAKE_DAMAGE` is `0x34`, **not** `0x33` — `0x33` is `MOVING_SHOOT_ATTACK_PREPARE`,
  a v95-only op. This is precisely the kind of off-by-one that reading the registry
  prevents and that pattern-matching against v83 would have introduced.

- [ ] **Step 5: Validate JSON and run the order guard**

Run from the worktree root:

```bash
for f in services/atlas-configurations/seed-data/templates/template_gms_61_1.json \
         services/atlas-configurations/seed-data/templates/template_gms_72_1.json \
         services/atlas-configurations/seed-data/templates/template_gms_79_1.json \
         services/atlas-configurations/seed-data/templates/template_gms_83_1.json \
         services/atlas-configurations/seed-data/templates/template_gms_84_1.json \
         services/atlas-configurations/seed-data/templates/template_gms_87_1.json \
         services/atlas-configurations/seed-data/templates/template_gms_92_1.json \
         services/atlas-configurations/seed-data/templates/template_gms_95_1.json \
         services/atlas-configurations/seed-data/templates/template_jms_185_1.json; do
  python3 -m json.tool "$f" > /dev/null && echo "OK $f" || echo "INVALID $f"
done
tools/template-opcode-order-guard.sh
```

Expected: `OK` ×9, and the order guard exits 0.

Then verify presence (9 templates carry the options; 4 gained the handler pair; the two
n-a templates carry neither):

```bash
python3 - <<'EOF'
import json
BASE = "services/atlas-configurations/seed-data/templates/"
opts = ["gms_61","gms_72","gms_79","gms_83","gms_84","gms_87","gms_92","gms_95","jms_185"]
for v in opts:
    d = json.load(open(f"{BASE}template_{v}_1.json"))
    ws = {w["writer"]: w for w in d["socket"]["writers"]}
    assert ws["CharacterBuffGive"]["options"]["vehicles"]["CORSAIR_BATTLESHIP"] == 1932000, v
    assert ws["CharacterSkillCooldown"]["options"]["skills"]["BATTLESHIP_HP_GAUGE"] == 5221999, v
    print("options verified", v)
for v in opts:
    d = json.load(open(f"{BASE}template_{v}_1.json"))
    hs = {h["handler"]: h for h in d["socket"]["handlers"]}
    for h in ("CharacterUseSkillHandle", "CharacterDamageHandle"):
        assert h in hs, f"{v} missing {h}"
        assert hs[h].get("validator"), f"{v} {h} has no validator"
    print("handlers verified", v)
for v in ("gms_12", "gms_48"):
    d = json.load(open(f"{BASE}template_{v}_1.json"))
    blob = json.dumps(d)
    assert "BATTLESHIP" not in blob, f"{v} must stay battleship-free (n-a, R-3)"
    print("n-a confirmed", v)
EOF
```

Expected: `options verified` ×9, `handlers verified` ×9, `n-a confirmed` ×2.

- [ ] **Step 6: Run atlas-configurations tests (template loading)**

Run: `cd services/atlas-configurations/atlas.com/configurations && go test ./... && go vet ./...`
Expected: PASS (seed-loading tests, if any, consume the templates)

- [ ] **Step 7: Commit**

```bash
git add services/atlas-configurations/seed-data/templates/
git commit -m "feat(config): battleship wire-value tables; wire v92 buff/cooldown writers and the missing cast/damage handlers on v87/v92/v95/jms185"
```

---


### Task 12: verification suite + live-tenant backfill runbook

**Files:**
- Create: `docs/tasks/task-153-corsair-battleship/backfill.md`

- [ ] **Step 1: Full test/vet matrix in every changed module**

From the worktree root, run each and confirm clean:

```bash
(cd libs/atlas-constants && go test -race ./... && go vet ./...)
(cd libs/atlas-redis && go test -race ./... && go vet ./...)
(cd libs/atlas-packet && go test -race ./... && go vet ./...)
(cd services/atlas-channel/atlas.com/channel && go test -race ./... && go vet ./... && go build ./...)
(cd services/atlas-configurations/atlas.com/configurations && go test -race ./... && go vet ./...)
```

Expected: all PASS / no vet findings. Report actual output; do not summarize failures away.

- [ ] **Step 2: Repo guards**

All five run from the worktree root, no `GOWORK=off` prefix. The last three are new
since this plan was written (R-11) and are CI jobs — a miss here is a wasted CI cycle.

```bash
tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/template-opcode-order-guard.sh
tools/lint.sh --check
```

Expected: all exit 0.

- `redis-key-guard` — clean; the only keyed Redis calls added live in
  `libs/atlas-redis/counter.go`.
- `goroutine-guard` — this task spawns no goroutines; it must stay clean.
- `template-opcode-order-guard` — Task 11 touched nine templates; every inserted
  handler/writer must sit at its sorted `opCode` position.
- `lint.sh --check` — run `tools/lint.sh` (no flags) first to fix formatting in place,
  then re-run with `--check`.

`tools/service-registration-guard.sh` is **not** required: this task changes no entry in
services.json, deploy/k8s, docker-bake.hcl, go.work, or tools/db-bootstrap.sh. Run it
anyway if that stops being true.

- [ ] **Step 3: docker buildx bake**

`<ch>/go.mod` is **not** modified by this task any more (R-9: `main` already added the
atlas-redis require and replace), so strictly no bake is mandated. Run it regardless —
it is cheap relative to a CI round-trip:

```bash
docker buildx bake atlas-channel
```

Expected: successful build. (`libs/atlas-redis` already has its two COPY lines in the
shared Dockerfile — atlas-mounts et al. build against it.)

- [ ] **Step 4: Write the backfill runbook**

Create `docs/tasks/task-153-corsair-battleship/backfill.md` (repo-relative paths only):

```markdown
# Battleship live-tenant config backfill

Seed templates apply only at tenant creation — existing tenants do NOT pick
up the new writer options (known gotcha: new opcodes/options never reach
live tenant configs automatically), and atlas-channel does not hot-reload
socket configuration.

## Scope

Nine of the ten live tenants need the backfill: GMS v61, v72, v79, v83, v84,
v87, v92, v95 and JMS v185.

GMS v48 is **excluded**: skill 5221006 does not exist in that client (its WZ
data returns 404 and the binary contains no reference to the skill or to the
5221999 gauge sentinel). GMS v12 has no live tenant. Neither is a gap.

## Per-tenant steps

1. Fetch the tenant's channel socket configuration from atlas-tenants
   (`GET /tenants/{tenantId}/configurations/{resourceName}` for the channel
   socket resource, via the admin UI or REST).
2. In `socket.writers`, add to the existing entries (do not change opCodes):
   - `CharacterBuffGive` → `"options": {"vehicles": {"CORSAIR_BATTLESHIP": 1932000}}`
   - `CharacterSkillCooldown` → `"options": {"skills": {"BATTLESHIP_HP_GAUGE": 5221999}}`
3. GMS v87, v92, v95, JMS v185 ONLY: these live configs are also missing
   `CharacterUseSkillHandle` and `CharacterDamageHandle` entirely (their seed
   templates were). Without both, the cast never reaches the server and damage
   is never processed — the feature is inert. Add to `socket.handlers`, at the
   sorted opCode position, with `"validator": "LoggedInValidator"` and
   `"services": ["channel"]`:
   - v87: `CharacterUseSkillHandle 0x5E`, `CharacterDamageHandle 0x32`
   - v95: `CharacterUseSkillHandle 0x67`, `CharacterDamageHandle 0x34`
   - JMS v185: `CharacterUseSkillHandle 0x56`, `CharacterDamageHandle 0x27`
   - v92: `CharacterUseSkillHandle 0x66`, `CharacterDamageHandle 0x35`
   (v87/v95/jms185 values come from docs/packets/registry/<version>.yaml and
   were cross-validated against the already-verified entries in the same
   files; v92 has no registry column; its two values were derived from its IDB and
   triple-cross-checked — see plan.md Task 11 Step 3.)
4. GMS v92 ONLY: the live config may also be missing the five buff/cooldown
   writers entirely (its seed template was). If absent, add:
   `CharacterBuffGive 0x21` (with the vehicles options),
   `CharacterBuffCancel 0x22`, `CharacterBuffGiveForeign 0xE3`,
   `CharacterBuffCancelForeign 0xE4`,
   `CharacterSkillCooldown 0x112` (with the skills options).
   These opcodes are CSV-derived (docs/packets/MapleStory Ops -
   ClientBound.csv), cross-validated against the other versions' verified
   template entries.
5. PATCH the configuration back, then restart atlas-channel (handlers and
   writers are read at listener build; the config projection does not
   hot-reload them).
6. Verify per tenant on a live client: mount/dismount visuals (self +
   foreign), gauge movement under damage, break → dismount + cooldown +
   greyed icon, remount-while-cooling rejected, Cannon/Torpedo on foot
   rejected (debug log `battleship_attack_rejected_not_riding`).

## Known blocker: GMS v95 has no ingested skill data

The v95 tenant returns `maxLevel: 0, effects: []` for skill 5221006 — and for
every other skill probed (5221004, 5221007, 5221008, 5121000, 1001003). This
is a tenant-wide WZ ingestion gap that predates this task, not a battleship
defect. Until v95 skill data is ingested, `GetEffect` yields nothing on that
tenant: no statup set, no MP cost, no cooldown value, so the mount cannot
apply. Do the config backfill anyway, then record v95 live verification as
BLOCKED — do not report it as verified.

## Ship HP is version-dependent

When eyeballing the gauge, expect different full-pool values either side of
v87. The server mirrors each client's own `get_max_durability_of_vehicle`:

- GMS v61–v84: `200 × (charLevel + 2×SLV − 120)`
- GMS v87+ and JMS: `300 × charLevel + 500 × (SLV − 72)`

A level-200 character with SLV 10 starts at 20 000 on v83 and 29 000 on v95.
Battleship is maxLevel 10 on every version.

Full sweep required — do not spot-check one tenant and declare all nine done.
```

- [ ] **Step 5: Commit**

```bash
git add docs/tasks/task-153-corsair-battleship/backfill.md
git commit -m "docs(task-153): live-tenant backfill runbook"
```

- [ ] **Step 6: Code review before PR**

Run `superpowers:requesting-code-review` (dispatches plan-adherence-reviewer + backend-guidelines-reviewer). Address findings before any PR.

---


## Acceptance criteria traceability

| PRD acceptance criterion | Covered by |
|---|---|
| Cast mounts vehicle 1932000 via MONSTER_RIDING, self + foreign | Tasks 1, 8, 11 (existing buff consumers do the announcing) |
| No cooldown on cast; cast-while-cooling rejected | Task 8 (carve-out + `battleshipCastBlocked`) |
| HP init, fresh each ride — **version-gated** (R-4): `200×(charLvl+2×SLV−120)` below v87, `300×charLvl+500×(SLV−72)` from v87 | Tasks 6 (`ShipHP`, `isPostBigBangDurability`, `InitShipHP`), 8 (mount arm) |
| Drain + parallel character HP + 5221999 gauge per drain | Tasks 6 (Drain), 9 (gauge announce; `ChangeHP` untouched) |
| Break exactly-once → dismount + effect-cooltime cooldown + state cleared | Tasks 2 (atomic counter), 6 (crossing predicate + breakShip) |
| Manual dismount / expiry / logout: no cooldown, state cleared | Task 7 (buff EXPIRED hook + session destroy hook) |
| Cannon/Torpedo rejected on foot, normal while riding | Task 10 |
| Wire values config-resolved, live tenants backfilled | Tasks 3, 4, 11, 12 (runbook) |
| Feature reachable on every version that has the skill (R-2/R-8.1) | Task 11 Steps 3–4 (cast + damage handlers for gms_87/92/95/jms_185); gms_12/gms_48 are n-a (R-3) |
| mount_test.go flips + new unit tests | Tasks 1, 2, 3, 4, 5, 6, 7, 8, 9, 10 |
| test/vet/build/bake clean; redis-key-guard, goroutine-guard, template-opcode-order-guard, lint.sh --check all clean (R-11) | Task 12 Steps 1–3 |
