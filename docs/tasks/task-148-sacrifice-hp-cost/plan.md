# Sacrifice Self-HP Cost (Dragon Knight 1311005) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** After a Dragon Knight Sacrifice (skill 1311005) attack, deduct `firstDamageLine × X / 100` HP from the caster — clamped so the caster always keeps at least 1 HP — replacing the TODO at `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go:405`.

**Architecture:** Follow the MP Eater precedent already in `character_attack_common.go`: two pure, unit-testable helper functions (`sacrificeFirstDamageLine`, `sacrificeHpCost`) plus one thin gated orchestration block in `processAttack`, placed in the post-broadcast side-effects section where the TODO sits today. The HP change rides the existing fire-and-forget `ChangeHP` Kafka command to atlas-character; errors are logged and swallowed, never aborting the attack pipeline.

**Tech Stack:** Go (atlas-channel service), table-driven `testing` unit tests, `libs/atlas-packet` model builders, `libs/atlas-constants/skill` id constants.

## Global Constraints

- Only `services/atlas-channel` changes. No changes to atlas-character, atlas-data, any lib, any `go.mod`, any REST/Kafka/packet contract (PRD §5–7).
- Use `skill3.DragonKnightSacrificeId` from `libs/atlas-constants/skill` — never a literal `1311005` (FR-1). Note: in `character_attack_common.go` the constants package is imported under the alias `skill3` (`character_attack_common.go:22`); bare `skill` there is `atlas-channel/character/skill`.
- Cost basis is **only** `DamageInfo()[0].Damages()[0]` — never sum lines or targets (FR-2, interview decision #2).
- Truncating integer division (`uint64` math, `/ 100`), Cosmic parity (FR-3).
- Survival clamp channel-side against `c.Hp()`: cost ≥ current HP → `Hp − 1`; `Hp ≤ 1` → cost 0, no emit (FR-4).
- No new network fetches — reuse `c`, `se`, `cp` already in scope in `processAttack` (PRD §8).
- No version branching — no `MajorVersion()` gates (PRD §8, interview decision #4).
- The generic `hpCon`/`mpCon` cast-cost block at `character_attack_common.go:303-310` stays untouched (FR-9).
- Only the line-405 TODO is removed; every other TODO in that block stays (FR-8).
- Tests use plain table tests / Builder pattern; no `*_testhelpers.go` files (FR-7, CLAUDE.md).
- Errors from `ChangeHP` are logged at `Errorf` and swallowed (FR-6); applied cost logged at `Debugf` with caster id, skill id, first line, X, clamped cost (PRD §8).

---

### Task 1: `sacrificeHpCost` pure helper (TDD)

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_sacrifice_test.go`
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go` (add one function near `mpEaterAbsorbAmount`, which ends at line 200)

**Interfaces:**
- Consumes: nothing from other tasks. `math` is already imported in `character_attack_common.go:16`.
- Produces: `func sacrificeHpCost(firstLine uint32, x int16, currentHp uint16) uint16` — Task 3's orchestration block calls this exact signature.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_sacrifice_test.go`:

```go
package handler

import (
	"math"
	"testing"
)

func TestSacrificeHpCost(t *testing.T) {
	cases := []struct {
		name      string
		firstLine uint32
		x         int16
		currentHp uint16
		want      uint16
	}{
		{"normal computation", 1000, 30, 5000, 300},
		{"truncating division", 99, 30, 5000, 29},
		{"x zero", 1000, 0, 5000, 0},
		{"x negative", 1000, -5, 5000, 0},
		{"miss (first line zero)", 0, 30, 5000, 0},
		{"clamp to hp minus one", 100000, 100, 500, 499},
		{"exact-kill boundary clamps", 1000, 100, 1000, 999},
		{"hp one is a no-op", 1000, 30, 1, 0},
		{"hp zero is a no-op", 1000, 30, 0, 0},
		{"narrowing guard caps at MaxInt16", 100000, 100, 65535, math.MaxInt16},
		{"max uint32 line does not wrap", math.MaxUint32, 100, 30000, 29999},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := sacrificeHpCost(tc.firstLine, tc.x, tc.currentHp); got != tc.want {
				t.Fatalf("sacrificeHpCost(%d, %d, %d) = %d; want %d", tc.firstLine, tc.x, tc.currentHp, got, tc.want)
			}
		})
	}
}
```

Case-by-case rationale (all pinned by the design §4.1 and §7):
- `1000 × 30 / 100 = 300` — plain formula (FR-3).
- `99 × 30 / 100 = 29.7 → 29` — truncation, not rounding (FR-3).
- `x ≤ 0` → 0 (FR-3).
- `firstLine = 0` (miss) → 0 (FR-2/FR-3).
- `100000 × 100 / 100 = 100000 ≥ 500` → `500 − 1 = 499` (FR-4).
- `1000 × 100 / 100 = 1000 == hp` → `999` — the `>=` boundary (FR-4).
- `hp ≤ 1` → 0, no emit possible (FR-4).
- `hp=65535`: clamp gives `65534`, still overflows `int16`, cap to `32767` — proves `-int16(cost)` at the call site cannot overflow (design §4.1 rule 4).
- `math.MaxUint32 × 100` overflows `uint32` but not `uint64`; widened math yields `4294967295`, clamped to `29999` (design §4.1 rule 2).

- [ ] **Step 2: Run test to verify it fails**

Run (from `services/atlas-channel/atlas.com/channel`):
```bash
go test ./socket/handler/ -run TestSacrificeHpCost -v
```
Expected: FAIL to build with `undefined: sacrificeHpCost`.

- [ ] **Step 3: Write minimal implementation**

In `character_attack_common.go`, insert immediately after `mpEaterAbsorbAmount` (after line 200, before the `mpEaterTryProc` comment at line 202):

```go
// sacrificeHpCost computes the self-HP cost of Dragon Knight Sacrifice:
// firstLine × x / 100 (truncating integer division, Cosmic parity),
// clamped so the caster is left with at least 1 HP. Returns 0 when the
// first line is 0 (miss), x is non-positive, or currentHp <= 1. The
// MaxInt16 cap is a defensive narrowing guard: on supported versions max
// HP <= 30000 so the survival clamp already bounds the result, but Hp()
// is uint16 and the call site negates into int16 — the cap makes that
// narrowing safe by construction instead of by data assumption.
func sacrificeHpCost(firstLine uint32, x int16, currentHp uint16) uint16 {
	if firstLine == 0 || x <= 0 || currentHp <= 1 {
		return 0
	}
	cost := uint64(firstLine) * uint64(x) / 100
	if cost >= uint64(currentHp) {
		cost = uint64(currentHp) - 1
	}
	if cost > math.MaxInt16 {
		cost = math.MaxInt16
	}
	return uint16(cost)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run (from `services/atlas-channel/atlas.com/channel`):
```bash
go test ./socket/handler/ -run TestSacrificeHpCost -v
```
Expected: PASS (all 11 subtests).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go services/atlas-channel/atlas.com/channel/socket/handler/character_attack_sacrifice_test.go
git commit -m "feat(atlas-channel): add sacrificeHpCost helper for Dragon Knight Sacrifice"
```

---

### Task 2: `sacrificeFirstDamageLine` extraction helper (TDD)

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_sacrifice_test.go` (append tests)
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go` (add one function directly below `sacrificeHpCost` from Task 1)

**Interfaces:**
- Consumes: `packetmodel.AttackInfo` / `packetmodel.DamageInfo` from `libs/atlas-packet/model` — already imported as `packetmodel` at `character_attack_common.go:25`. Builders used in tests: `packetmodel.NewAttackInfo(attackType)` (`attack_info.go:22`), `(*AttackInfo).AddDamageInfo(di DamageInfo)` (`attack_info.go:433`), `packetmodel.NewDamageInfo(hits byte)` (`damage_info.go:12`), `(*DamageInfo).SetMonsterId` / `(*DamageInfo).SetDamages` (`damage_info.go:94,104`).
- Produces: `func sacrificeFirstDamageLine(ai packetmodel.AttackInfo) uint32` — Task 3's orchestration block calls this exact signature (note: value parameter, matching `processAttack`'s `ai`).

- [ ] **Step 1: Write the failing test**

Append to `character_attack_sacrifice_test.go` (and add the import):

```go
import (
	"math"
	"testing"

	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)
```

```go
func TestSacrificeFirstDamageLine(t *testing.T) {
	entry := func(monsterId uint32, damages []uint32) packetmodel.DamageInfo {
		return *packetmodel.NewDamageInfo(byte(len(damages))).
			SetMonsterId(monsterId).
			SetDamages(damages)
	}

	cases := []struct {
		name string
		ai   packetmodel.AttackInfo
		want uint32
	}{
		{
			"no damage entries",
			*packetmodel.NewAttackInfo(packetmodel.AttackTypeMelee),
			0,
		},
		{
			"first entry has no lines",
			*packetmodel.NewAttackInfo(packetmodel.AttackTypeMelee).
				AddDamageInfo(entry(100100, nil)),
			0,
		},
		{
			"multi-line first entry returns line zero only",
			*packetmodel.NewAttackInfo(packetmodel.AttackTypeMelee).
				AddDamageInfo(entry(100100, []uint32{4000, 9999, 12345})),
			4000,
		},
		{
			"multi-target attack ignores second entry (FR-2 pin)",
			*packetmodel.NewAttackInfo(packetmodel.AttackTypeMelee).
				AddDamageInfo(entry(100100, []uint32{4000})).
				AddDamageInfo(entry(100101, []uint32{99999})),
			4000,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := sacrificeFirstDamageLine(tc.ai); got != tc.want {
				t.Fatalf("sacrificeFirstDamageLine() = %d; want %d", got, tc.want)
			}
		})
	}
}
```

The last case is the FR-2 pin: a future edit that sums lines or targets breaks a named test, not just Cosmic parity.

- [ ] **Step 2: Run test to verify it fails**

Run (from `services/atlas-channel/atlas.com/channel`):
```bash
go test ./socket/handler/ -run TestSacrificeFirstDamageLine -v
```
Expected: FAIL to build with `undefined: sacrificeFirstDamageLine`.

- [ ] **Step 3: Write minimal implementation**

In `character_attack_common.go`, insert directly below `sacrificeHpCost`:

```go
// sacrificeFirstDamageLine returns the first damage line of the first
// damage entry, or 0 when the attack has no entries or the first entry
// has no lines. Sacrifice's self-HP cost basis is only ever this line —
// additional lines and targets are deliberately ignored (Cosmic
// damageLines().getFirst() parity; PRD FR-2).
func sacrificeFirstDamageLine(ai packetmodel.AttackInfo) uint32 {
	di := ai.DamageInfo()
	if len(di) == 0 || len(di[0].Damages()) == 0 {
		return 0
	}
	return di[0].Damages()[0]
}
```

- [ ] **Step 4: Run test to verify it passes**

Run (from `services/atlas-channel/atlas.com/channel`):
```bash
go test ./socket/handler/ -run TestSacrificeFirstDamageLine -v
```
Expected: PASS (all 4 subtests).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go services/atlas-channel/atlas.com/channel/socket/handler/character_attack_sacrifice_test.go
git commit -m "feat(atlas-channel): add sacrificeFirstDamageLine helper pinning FR-2 first-line basis"
```

---

### Task 3: Orchestration block in `processAttack` + TODO removal

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go:396-405` (insert block after the projectile-emit block, delete the line-405 TODO)

**Interfaces:**
- Consumes: `sacrificeHpCost(firstLine uint32, x int16, currentHp uint16) uint16` (Task 1), `sacrificeFirstDamageLine(ai packetmodel.AttackInfo) uint32` (Task 2), and identifiers already in scope inside `processAttack`: `cp` (`character.Processor`, line 271), `c` (`character.Model`, line 272; `Hp() uint16` at `character/model.go:131`), `se` (`effect.Model`, line 292; `X() int16` at `data/skill/effect/model.go:144`), `s` (`session.Model`), `ai` (`packetmodel.AttackInfo`), `skill3.DragonKnightSacrificeId` (`libs/atlas-constants/skill/constants.go:2997`), `cp.ChangeHP(f field.Model, characterId uint32, amount int16) error` (`character/processor.go:271`).
- Produces: nothing consumed by later tasks; the runtime behavior itself.

- [ ] **Step 1: Insert the gated block and remove the TODO**

In `processAttack`, the tail of the function currently reads (lines 393-406):

```go
					// Projectile reservation + consume emits run fire-and-forget after the
					// broadcast. Classic semantics: the projectile is expended the moment the
					// server accepts the attack, regardless of broadcast success.
					if hasProjectilePlan {
						if perr := pp.Emit(s.CharacterId(), projectilePlan); perr != nil {
							l.WithError(perr).Errorf("Failed to emit projectile consumption for character [%d].", s.CharacterId())
						}
					}

					// TODO apply cooldown
					// TODO cancel dark sight / wind walk
					// TODO apply combo orbs (add or consume)
					// TODO decrease HP from DragonKnight Sacrifice
					// TODO apply attack effect (heal, mp consumption, dispel, cure all, combo reset, etc)
```

Change it to (new block inserted after the projectile `if`, and the `// TODO decrease HP from DragonKnight Sacrifice` line deleted — all other TODO lines stay exactly as they are):

```go
					// Projectile reservation + consume emits run fire-and-forget after the
					// broadcast. Classic semantics: the projectile is expended the moment the
					// server accepts the attack, regardless of broadcast success.
					if hasProjectilePlan {
						if perr := pp.Emit(s.CharacterId(), projectilePlan); perr != nil {
							l.WithError(perr).Errorf("Failed to emit projectile consumption for character [%d].", s.CharacterId())
						}
					}

					// Dragon Knight Sacrifice trades the caster's HP for the hit:
					// firstDamageLine × X / 100, clamped to leave at least 1 HP
					// (Cosmic parity — Sacrifice can never kill the caster). This
					// damage-proportional cost is separate from the generic
					// hpCon/mpCon cast cost above, which continues to apply.
					// Fire-and-forget like the projectile emit: failures are
					// logged and never abort the attack pipeline.
					if skill3.Id(ai.SkillId()) == skill3.DragonKnightSacrificeId {
						firstLine := sacrificeFirstDamageLine(ai)
						cost := sacrificeHpCost(firstLine, se.X(), c.Hp())
						if cost > 0 {
							l.Debugf("Sacrifice self-HP cost: caster=[%d] skill=[%d] firstLine=[%d] x=[%d] cost=[%d].",
								s.CharacterId(), ai.SkillId(), firstLine, se.X(), cost)
							if herr := cp.ChangeHP(s.Field(), s.CharacterId(), -int16(cost)); herr != nil {
								l.WithError(herr).Errorf("Sacrifice: CHANGE_HP emit failed for caster [%d] skill [%d].", s.CharacterId(), ai.SkillId())
							}
						}
					}

					// TODO apply cooldown
					// TODO cancel dark sight / wind walk
					// TODO apply combo orbs (add or consume)
					// TODO apply attack effect (heal, mp consumption, dispel, cure all, combo reset, etc)
```

Notes for the implementer:
- `skill3` is the constants alias (`character_attack_common.go:22`); do not use the bare `skill` package (that's the character-owned skill model).
- The gate implies `ai.SkillId() > 0`, so `se` was populated by the fetch at line 292 — the zero-value `effect.Model` case (`X() == 0`) is additionally covered by `sacrificeHpCost` returning 0.
- Use a fresh `herr` (not the enclosing `err`) so the swallowed emit error can never leak into a later `err` check.
- `-int16(cost)` is safe: `sacrificeHpCost` caps at `math.MaxInt16` (Task 1).

- [ ] **Step 2: Build and vet**

Run (from `services/atlas-channel/atlas.com/channel`):
```bash
go build ./... && go vet ./...
```
Expected: both clean, no output.

- [ ] **Step 3: Run the full handler package tests**

Run (from `services/atlas-channel/atlas.com/channel`):
```bash
go test -race ./socket/handler/ -v -run 'TestSacrifice|TestMpEater'
```
Expected: PASS — the two new Sacrifice tests plus the existing MP Eater tests, proving the shared file still compiles and behaves.

(The orchestration block itself is thin glue over the two tested helpers — same untested-call-site precedent as `mpEaterTryProc`; the wiring is covered by manual validation, Task 4.)

- [ ] **Step 4: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go
git commit -m "feat(atlas-channel): apply Dragon Knight Sacrifice self-HP cost after attack

Resolves the TODO at character_attack_common.go:405. Cost is
firstDamageLine * X / 100 from the tenant skill effect, clamped so the
caster keeps at least 1 HP (Cosmic parity)."
```

---

### Task 4: Full verification sweep

**Files:** none created or modified (verification only; fix-forward in place if anything fails).

**Interfaces:**
- Consumes: the complete change set from Tasks 1-3.
- Produces: the evidence trail required before code review / PR (CLAUDE.md Build & Verification).

- [ ] **Step 1: Full module test/vet/build**

Run (from `services/atlas-channel/atlas.com/channel`):
```bash
go test -race ./... && go vet ./... && go build ./...
```
Expected: all three clean. This is the only changed module.

- [ ] **Step 2: Redis key guard**

Run from the worktree root (`/…/.worktrees/task-148-sacrifice-hp-cost` — no `GOWORK=off` prefix):
```bash
tools/redis-key-guard.sh
```
Expected: clean / PASS.

- [ ] **Step 3: Confirm no `go.mod` was touched**

Run from the worktree root:
```bash
git diff main --name-only -- '**/go.mod' 'go.work*'
```
Expected: no output. (If anything shows up, something went off-plan — `docker buildx bake atlas-channel` would then be mandatory per CLAUDE.md.)

- [ ] **Step 4: Confirm the TODO is gone and no new TODOs were added**

Run from the worktree root:
```bash
grep -n "decrease HP from DragonKnight Sacrifice" services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go; git diff main -- services/atlas-channel | grep -c "^+.*TODO"
```
Expected: no grep match (exit 1 on the first command) and `0` from the count.

- [ ] **Step 5: Acceptance-criteria walkthrough**

Re-read PRD §10 and check each box that is code-verifiable (formula, clamp, miss = free, cast-cost untouched, non-Sacrifice unchanged, unit-test coverage list, TODO removed, verification suite). The two manual criteria — in-game validation on v83 and v95 tenants — remain for the human after deploy; they are not blockers for code review but must be called out in the PR description.

---

## Manual Validation (post-merge / deploy, human-driven)

Per PRD acceptance criteria and interview decision #4 — not automatable in this plan:
- On a v83 tenant and a v95 tenant, attack with Sacrifice and confirm the HP drop equals `firstDamageLine × X / 100` for the character's skill level.
- Confirm a low-HP caster is left at exactly 1 HP, never dead; at 1 HP the attack costs nothing beyond the normal cast cost.
- Confirm a miss costs nothing beyond the normal cast cost.
