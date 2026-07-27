# Mortal Blow (Ranger/Sniper) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the server half of pre-Big-Bang Mortal Blow: a ranged attack tagged with skill id 3110001/3210001 rolls a per-monster instant kill (HP ≤ maxHP·x/100, roll 1–100 ≤ y) delivered through atlas-monsters' standard kill flow via a new `KILL` command, with an authoritative fail-closed boss guard.

**Architecture:** atlas-channel evaluates the threshold and kill roll in the existing `onDamageApplied` post-damage callback (MP Eater's hook) and emits a new `KILL` monster command on proc success. atlas-monsters consumes `KILL`, re-checks alive + boss (fail-closed), and delivers a `math.MaxUint32` damage line through a `damageCore` helper split out of `Damage` — so damaged/killed events, EXP/drop credit, cooldown clears, registry removal, and revives all run unchanged. `Registry.ApplyDamage` clamps recorded damage to remaining HP (verified at `services/atlas-monsters/atlas.com/monsters/monster/registry.go:427-483`), so the damage summary stays honest with `MaxUint32`.

**Tech Stack:** Go, Kafka (segmentio/kafka-go via atlas-kafka), logrus, existing Redis-backed monster registry. No REST, packet, or data-model changes.

**Worktree:** all paths are relative to the `task-152-mortal-blow` worktree root (`.worktrees/task-152-mortal-blow/`). Work on branch `task-152-mortal-blow` only.

---

## ⚠ Main-Merge Reconciliation (2026-07-27) — READ FIRST, AUTHORITATIVE

`main` was merged into this branch (merge commit on `task-152-mortal-blow`; merge-base was `33aafe644`, ~176 commits behind). Main heavily rewrote every file this plan touches (`character_attack_common.go` +433, `monsters/monster/processor.go` +331, `effect/{model,rest}.go` +115/+92, `character_attack_projectile.go` +51). **All line numbers in the tasks below are stale — re-grep every anchor.** Where this section conflicts with a task's verbatim code block, **this section wins**; the superseded blocks are kept only as design context.

Two commit-footer notes: the plan's "Co-Authored-By: Claude Fable 5" line reflects who wrote the plan — use the *executing* model's footer for actual commits (session policy pins execution/review to the cheaper model). Verification list additions are at the end of this section.

### Legacy-version scope (the reason for this merge)

Main added packet support for legacy client versions **v48 / v61 / v72 / v79** (all pre-Big-Bang). **This requires NO change to the Mortal Blow server design, and no per-version code.** Rationale, verified in-tree:

- The attack packet writes `skillId` **ungated across all versions** (`libs/atlas-packet/model/attack_info.go:153` — `w.WriteInt(m.skillId)` sits above every version gate). The legacy work only gated the CRC head-block (`legacyGmsNoSkillDataCrc`/`legacyGmsSingleCrc`), the action-byte width (`legacyGmsByteAction`), and ranged bullet coords (`legacyGmsNoRangedBulletCoords`) — never the skill id. So `ai.SkillId()` is populated for a legacy ranged attack exactly as for v83.
- Mortal Blow gates purely on `isMortalBlowAttack(ai.AttackType(), ai.SkillId())` + tenant skill data (`se.X()`/`se.Y()`). It is therefore **version-agnostic**: functional wherever a client sends 3110001/3210001, inert everywhere else (post-BB v95/JMS clients never send them — PRD §4.6).
- **Contrast with task-150 (meso explosion):** that task needed per-version *packet encoding* for legacy, so legacy bring-up genuinely widened its code scope. Mortal Blow has **no packet** — it is a server-side kill roll keyed on an incoming id — so legacy bring-up widens only the *tenant set the feature can fire on*, automatically, with zero new code.
- **Unverified (client-data question, not a server blocker):** whether the v48/v61/v72/v79 clients actually tag a Mortal Blow proc with skill id 3110001/3210001 is not IDA-verified here (only v83/v84 were, PRD §4). The design is inert-safe either way. This carries the same status as PRD §4.6's v87/v92 note and PRD §10 — surface it if a legacy tenant misbehaves; do **not** add version gates or hard-coded fallbacks to compensate.

**Net effect on the plan:** no new tasks for legacy versions. The one addition is a doc note (below) so the next reader knows legacy tenants are in-scope-by-design.

### Per-task corrections

**Task 1 (`Y()` accessor) — NOW OBSOLETE.** Main already added `func (m Model) Y() int16 { return m.y }` at `data/skill/effect/model.go:167` (its doc comment references MP Recovery, not Mortal Blow — that is fine; the accessor is generic). **Do not re-add it.** Reduce Task 1 to: (a) confirm `Y()` exists and returns `m.y`; (b) if `data/skill/effect/model_test.go` does not exist, optionally add the `TestExtractThreadsXAndY` regression test from the old Task 1 (it still compiles against the current `RestModel{X,Y int16}` at `rest.go:41-42`); (c) no code commit needed for the accessor itself.

**Task 4 (`onDamageApplied` wiring + tests) — SIGNATURE CHANGED; ADD, don't replace.** The closure now lives at `character_attack_common.go:659` and its type (in `damageInfoEntryDeps`, ~line 100/114) is:

```go
onDamageApplied func(di packetmodel.DamageInfo, totalDamage uint32)
```

Main already populates it with **MP Eater + drain (task-147) + Pick Pocket**. Do **not** replace the closure (old Task 4 Step 4 does) — **add** a Mortal Blow branch inside the existing closure body, using `di.MonsterId()` for the monster id:

```go
// Mortal Blow proc: per-monster, ranged attacks tagged with the
// Ranger/Sniper Mortal Blow skill id only. Ownership was enforced
// upstream (unowned skill ids destroy the session). Failures swallowed (FR-5).
if isMortalBlowAttack(ai.AttackType(), ai.SkillId()) {
	mortalBlowTryProc(l, mortalBlowDeps{
		getMonster: mp.GetById,
		emitKill:   mp.Kill,
		roll:       func() int { return rand.Intn(100) + 1 },
	}, se, di.MonsterId(), s.Field(), s.CharacterId(), ai.SkillId())
}
```

Delete the `// TODO Mortal Blow` marker — now at `character_attack_common.go:772` (in the trailing TODO block at the end of `processAttack`); leave every other TODO line.

Task 4 **test-helper deltas** (the old Step-1 test code is stale in two spots): in `mbEntryDeps`, the struct field is `loadEffectiveStats` (not `loadVenomStats`); and `onDamageApplied`/the helper param is `func(di packetmodel.DamageInfo, totalDamage uint32)` (not `func(uint32)`) — inside the tests, read the monster id via `di.MonsterId()`. The `processDamageInfoEntry(...)` call arg list (12 args) and the other `deps` field signatures (`getReflect`, `getMonster`, `applyDamage`, `applyStatus`) are unchanged and still match `character_attack_common.go:100-115,122-134`.

**Task 6 (`Damage` → `checkReflect` + `damageCore`) — `checkReflect` ALREADY EXISTS; the plan's verbatim `damageCore` would REGRESS main.** Main already extracted `checkReflect` (called at `processor.go:557`, defined ~1532) and, since the plan was written, added (a) a **GM-hidden controller-switch guard** (`if _, isHidden := p.hiddenSet()[characterId]; isHidden { … } else { … }`, currently line ~647) and (b) switched the info fetch to `information.NewProcessor(p.l, p.ctx).GetById(...)` (line 562). Applying the old Task 6 code verbatim silently reverts both. **Use this split instead** — `Damage` (currently lines 541-681) becomes:

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

	p.damageCore(m, characterId, damages)
}

// damageCore applies damage lines to an already-fetched, alive monster and runs
// the full post-damage flow (damaged event, picker, kill handling, controller
// switch, aggro). Callers own the preceding guards: Damage does alive + reflect;
// Kill (Mortal Blow) does alive + fail-closed boss and never rolls reflect.
func (p *ProcessorImpl) damageCore(m Model, characterId uint32, damages []uint32) {
	// Fetch monster info for boss flag and revives
	var isBoss bool
	var revives []uint32
	if ma, infoErr := information.NewProcessor(p.l, p.ctx).GetById(m.MonsterId()); infoErr == nil {
		isBoss = ma.Boss()
		revives = ma.Revives()
	}

	oldHpPercentage := m.HpPercentage()

	var last DamageSummary
	hasLast := false
	killed := false
	firstHitObserved := false
	nowMs := time.Now().UnixMilli()
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
			break // discard overkill
		}
	}

	if !hasLast {
		return
	}

	// Always emit damaged so the channel writes the final HP-bar packet.
	if err := p.emit(EnvEventTopicMonsterStatus, damagedStatusEventProvider(last.Monster, last.CharacterId, last.CharacterId, isBoss, DamageSourceCharacterAttack, last.Monster.DamageSummary())); err != nil {
		p.l.WithError(err).Errorf("Monster [%d] damaged, but unable to display that for the characters in the field.", last.Monster.UniqueId())
	}

	if !killed && (firstHitObserved || last.Monster.HpPercentage() != oldHpPercentage) {
		if err := p.RepickAndEmit(last.Monster.UniqueId(), RepickReasonDamaged); err != nil {
			p.l.WithError(err).Warnf("Damage picker: monster [%d] re-pick failed.", last.Monster.UniqueId())
		}
	}

	if killed {
		// Substitution vs old Damage: the three cleanup calls use m.UniqueId()
		// (same value as the old `id` param — m was fetched by id).
		GetCooldownRegistry().ClearCooldowns(p.ctx, p.t, m.UniqueId())
		GetAttackCooldownRegistry().ClearCooldowns(p.ctx, p.t, m.UniqueId())
		GetDropTimerRegistry().Unregister(p.ctx, p.t, m.UniqueId())

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

	// Controller-switch + aggro emission. PRESERVE main's GM-hidden guard.
	controllerSwitched := false
	if characterId != last.Monster.ControlCharacterId() && last.Monster.DamageLeader() == characterId {
		if _, isHidden := p.hiddenSet()[characterId]; isHidden {
			p.l.Debugf("Skipping DPS-leader controller switch to GM-hidden character [%d] for monster [%d].", characterId, last.Monster.UniqueId())
		} else {
			inField, ferr := p.attackerInField(last.Monster.Field(), characterId)
			if ferr != nil || !inField {
				p.l.Debugf("FR-10: skipping controller switch for char [%d] not in field of monster [%d].", characterId, last.Monster.UniqueId())
			} else {
				p.l.Debugf("Character [%d] has become damage leader for monster [%d].", characterId, last.Monster.UniqueId())
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
	}

	if firstHitObserved && !controllerSwitched {
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

The `damageCore` body is the moved tail of the current `Damage` verbatim (info fetch through aggro emit), with the single `id`→`m.UniqueId()` substitution in the kill-path cleanup calls noted above. Task 6's acceptance gate is unchanged: the existing `processor_test.go` suite (DAMAGED/KILLED order, controller switch, GM-hidden skip, aggro) stays green.

**Task 7 (`Kill`) — use the non-curried info lookup.** Every "`information.GetById(p.l)(p.ctx)(m.MonsterId())`" in the old Task 7 body must become **`information.NewProcessor(p.l, p.ctx).GetById(m.MonsterId())`** (matching what `damageCore` now uses at `processor.go:562`). The `testInformationLookup` seam (`drain_mp_test.go` uses it) and `newRecordingProcessorWithBodies` (`processor_test.go:236`) are present and unchanged. Insert `Kill` on the interface after `DrainMp` (interface line ~65) and the method after `DrainMp`'s impl (~line 1634). `Registry.ApplyDamage` clamps at `registry.go:424-436` (Decision 5's `MaxUint32` still valid).

**Tasks 2, 3, 5, 8 — structurally intact, line numbers stale only.** Task 2 helpers still valid (insert anywhere among the attack-common helpers; note the shared `shouldProc(prop, roll)` helper already exists at ~line 216 — Mortal Blow's helpers are independent of it). Task 3/8 Kafka `DRAIN_MP` siblings are present (channel `producer.go:152`, `kafka.go:19`; monsters `consumer.go:162`, `kafka.go:24,99`) — mirror them for `KILL`. Task 5's stale `task-007` comment still lists Mortal Blow at `character_attack_projectile.go:119` — the fix is unchanged.

### Verification-list additions (merged CLAUDE.md)

Beyond the old Task 9 list, the merged CLAUDE.md adds guards that now gate "done." Run from the worktree root, in addition to `go test -race`/`go vet`/`go build` per module + `docker buildx bake atlas-channel atlas-monsters` + `tools/redis-key-guard.sh`:

- `tools/goroutine-guard.sh` — no bare `go` statements (this task adds none; the `rand.Intn` roll and Kafka emits use existing infra).
- `tools/lint.sh --check` — gofumpt/goimports + linters across changed modules. Run `tools/lint.sh` (fix mode) before committing.
- `tools/service-registration-guard.sh` and `tools/template-opcode-order-guard.sh` are **not** triggered (no services.json/deploy/k8s/docker-bake or template changes in this task).

### Legacy scope doc note (small addition to the deliverable)

When touching the `mortalBlowTryProc` doc comment (Task 4) or `context.md`, add one sentence: Mortal Blow is version-agnostic and applies to any tenant whose client sends skill id 3110001/3210001, including the pre-BB legacy versions v48/v61/v72/v79 brought up on main; it is inert on v95/JMS (post-BB redesign) and on any version whose client never sends the id. No per-version code exists or is needed.

---

## Global Constraints

- Skill ids come from `libs/atlas-constants/skill` (DOM-21): `skill3.RangerMortalBlowId` (3110001), `skill3.SniperMortalBlowId` (3210001) — never numeric literals in service code.
- `x` (HP threshold %) and `y` (kill chance %) come from the tenant skill effect at the character's owned level (`se.X()` / `se.Y()`) — no hard-coded thresholds or chances (FR-2/FR-3).
- Every proc failure is logged and swallowed; the attack pipeline (damage, broadcast, projectile emit) must never abort or delay (FR-5).
- Boss kills must be impossible even if the channel misfires: atlas-monsters' boss lookup is FAIL-CLOSED — lookup error ⇒ drop the kill (FR-4). This deliberately diverges from `DrainMp`'s fail-open.
- Test setup uses the project's Builder pattern (`monster.NewModelBuilder`, `information.NewModelBuilder`, `field.NewBuilder`, `effect.Extract`); no `*_testhelpers.go` files.
- No `// TODO`, stubs, or 501s in landed commits.
- Verification before "done": `go test -race ./...`, `go vet ./...`, `go build ./...` clean in BOTH changed modules (`services/atlas-channel/atlas.com/channel`, `services/atlas-monsters/atlas.com/monsters`); `docker buildx bake atlas-channel atlas-monsters` clean from the worktree root; `tools/redis-key-guard.sh` clean from the worktree root.
- Commit after every task. Commit messages end with `Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>`.

---

### Task 1: `Y()` accessor on the channel skill-effect model (FR-7)

> **⚠ SUPERSEDED by Main-Merge Reconciliation — `Y()` already exists (`model.go:167`). This task is now verify-only (optionally add the regression test). Do NOT re-add the accessor.**

The REST model already deserializes `y` (`rest.go:42`) and `Extract` already threads it into `Model.y` (`rest.go:115`); only the accessor is missing.

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/data/skill/effect/model.go` (append after `X()`, line 147)
- Test: `services/atlas-channel/atlas.com/channel/data/skill/effect/model_test.go` (create — the package has no test file yet)

**Interfaces:**
- Consumes: existing `effect.RestModel{X int16, Y int16}` and `Extract(RestModel) (Model, error)`.
- Produces: `func (m Model) Y() int16` — used by Task 4's `mortalBlowTryProc`.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/data/skill/effect/model_test.go`:

```go
package effect

import "testing"

// TestExtractThreadsXAndY verifies the generic x/y skill attributes survive
// the REST → domain transform. Mortal Blow reads X (HP threshold %) and Y
// (instant-kill chance %) from the effect resolved at the owned level.
func TestExtractThreadsXAndY(t *testing.T) {
	m, err := Extract(RestModel{X: 20, Y: 5})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if m.X() != 20 {
		t.Fatalf("X() = %d, want 20", m.X())
	}
	if m.Y() != 5 {
		t.Fatalf("Y() = %d, want 5", m.Y())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./data/skill/effect/ -run TestExtractThreadsXAndY -v`
Expected: FAIL to compile with `m.Y undefined (type Model has no field or method Y)`

- [ ] **Step 3: Write the accessor**

Append to `services/atlas-channel/atlas.com/channel/data/skill/effect/model.go` (after the `X()` method):

```go
// Y returns the integer Y attribute (a generic per-level skill value; for
// Mortal Blow it is the instant-kill chance percent rolled server-side).
func (m Model) Y() int16 {
	return m.y
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./data/skill/effect/ -run TestExtractThreadsXAndY -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/data/skill/effect/model.go services/atlas-channel/atlas.com/channel/data/skill/effect/model_test.go
git commit -m "feat(task-152): expose Y() on channel skill-effect model (FR-7)"
```

---

### Task 2: Pure Mortal Blow decision helpers (FR-1/FR-2/FR-3 math)

Three pure functions in the attack-common file, mirroring `mpEaterShouldProc`/`mpEaterAbsorbAmount`: the eligibility threshold, the kill roll, and the attack gate. Pure helpers + roll-passed-as-parameter is this file's established RNG seam (FR-8 injectability).

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go` (insert after `mpEaterAbsorbAmount`, ~line 200)
- Test: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_mortal_blow_test.go` (create)

**Interfaces:**
- Consumes: `packetmodel.AttackType` / `packetmodel.AttackTypeRanged`, `skill3.Id`, `skill3.RangerMortalBlowId`, `skill3.SniperMortalBlowId` (all already imported in the file).
- Produces:
  - `func mortalBlowEligible(hp uint32, maxHp uint32, x int16) bool`
  - `func mortalBlowKillRoll(roll int, y int16) bool`
  - `func isMortalBlowAttack(at packetmodel.AttackType, skillId uint32) bool`
  Task 4's `mortalBlowTryProc` and the `onDamageApplied` closure use all three.

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_mortal_blow_test.go`:

```go
package handler

import (
	"math"
	"testing"

	skill3 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

func TestMortalBlowEligible(t *testing.T) {
	cases := []struct {
		name  string
		hp    uint32
		maxHp uint32
		x     int16
		want  bool
	}{
		// maxHp=999, x=20 -> threshold 999*20/100 = 199 (truncating division,
		// Cosmic parity: (getStats().getHp() * getX()) / 100).
		{"hp exactly at threshold", 199, 999, 20, true},
		{"hp one above threshold", 200, 999, 20, false},
		{"hp well below threshold", 1, 999, 20, true},
		{"x zero never eligible", 1, 999, 0, false},
		{"x negative never eligible", 1, 999, -5, false},
		{"maxHp zero never eligible", 0, 0, 20, false},
		// uint64 widening: maxHp near MaxUint32 must not overflow.
		// threshold = MaxUint32*50/100 = 2147483647 (floor).
		{"no overflow at MaxUint32", 2147483647, math.MaxUint32, 50, true},
		{"no overflow above threshold", 2147483648, math.MaxUint32, 50, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := mortalBlowEligible(tc.hp, tc.maxHp, tc.x); got != tc.want {
				t.Fatalf("mortalBlowEligible(%d, %d, %d) = %v; want %v", tc.hp, tc.maxHp, tc.x, got, tc.want)
			}
		})
	}
}

func TestMortalBlowKillRoll(t *testing.T) {
	cases := []struct {
		name string
		roll int
		y    int16
		want bool
	}{
		{"roll equal to y procs", 5, 5, true},
		{"roll one above y misses", 6, 5, false},
		{"roll below y procs", 1, 5, true},
		{"y zero never procs", 1, 0, false},
		{"y negative never procs", 1, -1, false},
		{"y 100 always procs", 100, 100, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := mortalBlowKillRoll(tc.roll, tc.y); got != tc.want {
				t.Fatalf("mortalBlowKillRoll(%d, %d) = %v; want %v", tc.roll, tc.y, got, tc.want)
			}
		})
	}
}

func TestIsMortalBlowAttack(t *testing.T) {
	cases := []struct {
		name    string
		at      packetmodel.AttackType
		skillId uint32
		want    bool
	}{
		{"ranged + Ranger Mortal Blow", packetmodel.AttackTypeRanged, uint32(skill3.RangerMortalBlowId), true},
		{"ranged + Sniper Mortal Blow", packetmodel.AttackTypeRanged, uint32(skill3.SniperMortalBlowId), true},
		{"ranged + other skill", packetmodel.AttackTypeRanged, uint32(skill3.RangerStrafeId), false},
		{"ranged + no skill", packetmodel.AttackTypeRanged, 0, false},
		{"melee + Mortal Blow id", packetmodel.AttackTypeMelee, uint32(skill3.RangerMortalBlowId), false},
		{"magic + Mortal Blow id", packetmodel.AttackTypeMagic, uint32(skill3.RangerMortalBlowId), false},
		{"energy + Mortal Blow id", packetmodel.AttackTypeEnergy, uint32(skill3.SniperMortalBlowId), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isMortalBlowAttack(tc.at, tc.skillId); got != tc.want {
				t.Fatalf("isMortalBlowAttack(%v, %d) = %v; want %v", tc.at, tc.skillId, got, tc.want)
			}
		})
	}
}
```

(`skill3.RangerStrafeId` is verified to exist at `libs/atlas-constants/skill/constants.go:860`.)

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run 'TestMortalBlow|TestIsMortalBlow' -v`
Expected: FAIL to compile with `undefined: mortalBlowEligible` (and the other two)

- [ ] **Step 3: Write the helpers**

Insert into `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go`, directly after `mpEaterAbsorbAmount` (line 200):

```go
// mortalBlowEligible reports whether a monster's (pre-attack snapshot) HP
// is at or below the Mortal Blow threshold: hp ≤ maxHp × x / 100, with
// integer truncating division (Cosmic parity). Widens through uint64 so
// maxHp near MaxUint32 cannot overflow. Defensive: false when x ≤ 0 or
// maxHp == 0 (malformed/absent tenant data means the passive is inert).
func mortalBlowEligible(hp uint32, maxHp uint32, x int16) bool {
	if x <= 0 || maxHp == 0 {
		return false
	}
	return uint64(hp) <= uint64(maxHp)*uint64(x)/100
}

// mortalBlowKillRoll reports whether the instant kill procs for a uniform
// roll in [1,100]: roll ≤ y. Defensive: false when y ≤ 0.
func mortalBlowKillRoll(roll int, y int16) bool {
	if y <= 0 {
		return false
	}
	return roll <= int(y)
}

// isMortalBlowAttack reports whether an attack is a client-side Mortal Blow
// proc: a ranged attack tagged with the Ranger (3110001) or Sniper
// (3210001) passive's skill id. The v83 client only tags an attack with
// these ids on a successful point-blank normal-attack conversion, and the
// upstream ownership guard in processAttack destroys the session for
// unowned skill ids, so this gate is sufficient (no job-range check —
// PRD FR-1).
func isMortalBlowAttack(at packetmodel.AttackType, skillId uint32) bool {
	return at == packetmodel.AttackTypeRanged &&
		(skill3.Id(skillId) == skill3.RangerMortalBlowId || skill3.Id(skillId) == skill3.SniperMortalBlowId)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run 'TestMortalBlow|TestIsMortalBlow' -v`
Expected: PASS (all subtests)

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go services/atlas-channel/atlas.com/channel/socket/handler/character_attack_mortal_blow_test.go
git commit -m "feat(task-152): Mortal Blow threshold, kill-roll, and attack-gate helpers"
```

---

### Task 3: Channel-side `KILL` Kafka surface (Decision 7)

New command type + body in the channel's monster message package, a keyed provider, and a `Kill` method on the channel's monster `Processor`. Keyed by monster unique id so `KILL` lands on the same partition as the triggering `DAMAGE` and processes after it.

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka.go`
- Modify: `services/atlas-channel/atlas.com/channel/monster/producer.go`
- Modify: `services/atlas-channel/atlas.com/channel/monster/processor.go`
- Test: `services/atlas-channel/atlas.com/channel/monster/producer_test.go` (extend)

**Interfaces:**
- Consumes: existing `monster2.Command[E]` envelope, `producer.CreateKey`, `producer.SingleMessageProvider`, `producer.ProviderImpl` — all as used by the `DRAIN_MP` siblings in the same files.
- Produces:
  - `monster2.CommandTypeKill = "KILL"` and `monster2.KillCommandBody{CharacterId uint32; SkillId uint32}`
  - `func KillCommandProvider(f field.Model, monsterId uint32, characterId uint32, skillId uint32) model.Provider[[]kafka.Message]`
  - `func (p *Processor) Kill(f field.Model, monsterId uint32, characterId uint32, skillId uint32) error`
  Task 4 wires `mp.Kill` into `mortalBlowDeps.emitKill`. Task 8's consumer-side body must mirror `KillCommandBody`'s JSON exactly (`characterId`, `skillId`).

- [ ] **Step 1: Write the failing provider test**

Append to `services/atlas-channel/atlas.com/channel/monster/producer_test.go`:

```go
// TestKillCommandProvider verifies the KILL command envelope: keyed by the
// monster unique id (same partition as the triggering DAMAGE), type KILL,
// and a body carrying the caster and the Mortal Blow skill id.
func TestKillCommandProvider(t *testing.T) {
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	prov := KillCommandProvider(f, 12345, 67, 3110001)

	msgs, err := prov()
	if err != nil {
		t.Fatalf("provider error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("got %d messages, want 1", len(msgs))
	}

	var cmd monster2.Command[monster2.KillCommandBody]
	if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
		t.Fatalf("unmarshal command: %v", err)
	}
	if cmd.Type != monster2.CommandTypeKill {
		t.Fatalf("Type = %s, want %s", cmd.Type, monster2.CommandTypeKill)
	}
	if cmd.MonsterId != 12345 {
		t.Fatalf("MonsterId = %d, want 12345", cmd.MonsterId)
	}
	if cmd.Body.CharacterId != 67 {
		t.Fatalf("Body.CharacterId = %d, want 67", cmd.Body.CharacterId)
	}
	if cmd.Body.SkillId != 3110001 {
		t.Fatalf("Body.SkillId = %d, want 3110001", cmd.Body.SkillId)
	}
}
```

(All imports used above — `json`, `field`, `world`, `channel`, `_map`, `uuid`, `monster2` — are already imported by the existing tests in this file.)

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./monster/ -run TestKillCommandProvider -v`
Expected: FAIL to compile with `undefined: KillCommandProvider` / `monster2.CommandTypeKill`

- [ ] **Step 3: Add the command type and body**

In `services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka.go`, add to the command const block (after `CommandTypeDrainMp = "DRAIN_MP"`, line 18):

```go
	CommandTypeKill           = "KILL"
```

And add after `DrainMpCommandBody` (line 83):

```go
// KillCommandBody asks atlas-monsters to kill a monster outright as the
// result of a player passive (Mortal Blow). The channel owns the threshold
// (hp ≤ maxHp·x/100) and kill-chance (roll ≤ y) decisions; atlas-monsters
// re-checks alive + boss (fail-closed) and delivers the kill through the
// standard damage path so EXP and drops credit the attacker like a normal
// kill. SkillId is carried for traceability/logging only — atlas-monsters
// does not resolve it.
type KillCommandBody struct {
	CharacterId uint32 `json:"characterId"`
	SkillId     uint32 `json:"skillId"`
}
```

- [ ] **Step 4: Add the provider**

Append to `services/atlas-channel/atlas.com/channel/monster/producer.go`:

```go
// KillCommandProvider builds the KILL command for atlas-monsters to kill a
// monster as the result of a Mortal Blow proc. Keyed by the monster's
// unique id so it lands on the same partition as the triggering DAMAGE
// command and processes after it — if the attack itself already killed the
// monster, atlas-monsters finds it gone and drops the kill silently.
func KillCommandProvider(f field.Model, monsterId uint32, characterId uint32, skillId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(monsterId))
	value := &monster2.Command[monster2.KillCommandBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		MonsterId: monsterId,
		Type:      monster2.CommandTypeKill,
		Body: monster2.KillCommandBody{
			CharacterId: characterId,
			SkillId:     skillId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

- [ ] **Step 5: Add the processor method**

Append to `services/atlas-channel/atlas.com/channel/monster/processor.go`:

```go
// Kill emits a KILL command instructing atlas-monsters to kill a monster
// outright as the result of a player passive (Mortal Blow). The channel is
// the authority for the threshold and kill-chance rolls; atlas-monsters
// owns the guards only it can enforce (alive + boss, fail-closed) and
// delivers the kill through the standard damage path so EXP and drops
// credit the attacker identically to a normal kill.
func (p *Processor) Kill(f field.Model, monsterId uint32, characterId uint32, skillId uint32) error {
	p.l.Debugf("Requesting Mortal Blow kill of monster [%d] for character [%d] via skill [%d].", monsterId, characterId, skillId)
	return producer.ProviderImpl(p.l)(p.ctx)(monster2.EnvCommandTopic)(KillCommandProvider(f, monsterId, characterId, skillId))
}
```

- [ ] **Step 6: Run test to verify it passes**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./monster/ ./kafka/... -run TestKillCommandProvider -v && go build ./...`
Expected: test PASS, build clean

- [ ] **Step 7: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka.go services/atlas-channel/atlas.com/channel/monster/producer.go services/atlas-channel/atlas.com/channel/monster/processor.go services/atlas-channel/atlas.com/channel/monster/producer_test.go
git commit -m "feat(task-152): channel-side KILL monster command (type, provider, processor)"
```

---

### Task 4: `mortalBlowTryProc` and the `onDamageApplied` wiring (FR-1..FR-5, Decisions 3/5)

> **⚠ SUPERSEDED IN PART by Main-Merge Reconciliation — the `onDamageApplied` closure signature is now `func(di packetmodel.DamageInfo, totalDamage uint32)` (use `di.MonsterId()`), Mortal Blow is ADDED alongside the existing MP Eater/drain/Pick Pocket branches (not a closure replace), and the test helper uses `loadEffectiveStats`. `mortalBlowDeps`/`mortalBlowTryProc` themselves are unchanged. See the reconciliation for the corrected wiring + test deltas.**

The proc orchestrator plus its hook into the existing per-monster post-damage callback. It takes a small deps struct (mirroring `damageInfoEntryDeps` in the same file) so tests can drive every branch — snapshot miss, threshold, roll, emit failure — without a real `monster.Processor` or Kafka; this is how the design's §5 failure-isolation tests are realizable, and it refines Decision 5's `mp *monster.Processor` parameter into the file's established seam pattern (production behavior identical).

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go` (insert `mortalBlowDeps` + `mortalBlowTryProc` after the Task 2 helpers; add the ranged branch to the `onDamageApplied` closure at line ~351; delete the `// TODO Mortal Blow` line at ~421)
- Test: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_mortal_blow_test.go` (extend)

**Interfaces:**
- Consumes: `mortalBlowEligible`, `mortalBlowKillRoll`, `isMortalBlowAttack` (Task 2); `mp.GetById` (`func(uint32) (monster.Model, error)`); `mp.Kill` (Task 3, `func(field.Model, uint32, uint32, uint32) error`); `se.X()` / `se.Y()` (Task 1); `monster.NewModelBuilder(uniqueId, f, monsterId)` with `SetHp`/`SetMaxHp`/`MustBuild` for test snapshots; `effect.Extract(effect.RestModel{X:…, Y:…})` for test effects.
- Produces:
  - `type mortalBlowDeps struct { getMonster func(monsterId uint32) (monster.Model, error); emitKill func(f field.Model, monsterId uint32, characterId uint32, skillId uint32) error; roll func() int }`
  - `func mortalBlowTryProc(l logrus.FieldLogger, deps mortalBlowDeps, se effect.Model, monsterId uint32, f field.Model, characterId uint32, skillId uint32)`
  Nothing downstream consumes these — this is the feature's channel-side terminus.

- [ ] **Step 1: Write the failing flow tests**

Append to `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_mortal_blow_test.go`:

```go
// --- mortalBlowTryProc flow tests -------------------------------------
//
// The deps struct fakes the monster snapshot, the KILL emit, and the RNG,
// so every branch is pinned deterministically (FR-8).

func mbEffect(t *testing.T, x int16, y int16) effect.Model {
	t.Helper()
	m, err := effect.Extract(effect.RestModel{X: x, Y: y})
	if err != nil {
		t.Fatalf("effect.Extract: %v", err)
	}
	return m
}

func mbField() field.Model {
	return field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).Build()
}

func mbMonster(t *testing.T, uniqueId uint32, hp uint32, maxHp uint32) monster.Model {
	t.Helper()
	return monster.NewModelBuilder(uniqueId, mbField(), 1000000).SetHp(hp).SetMaxHp(maxHp).MustBuild()
}

// TestMortalBlowTryProc_InertEffectSkipsSnapshotFetch — x/y ≤ 0 in tenant
// data means the passive is inert: no snapshot fetch, no roll, no emit.
func TestMortalBlowTryProc_InertEffectSkipsSnapshotFetch(t *testing.T) {
	for _, se := range []effect.Model{mbEffect(t, 0, 5), mbEffect(t, 20, 0)} {
		fetched := false
		deps := mortalBlowDeps{
			getMonster: func(uint32) (monster.Model, error) {
				fetched = true
				return monster.Model{}, nil
			},
			emitKill: func(field.Model, uint32, uint32, uint32) error {
				t.Fatal("emitKill must not be called for inert effect")
				return nil
			},
			roll: func() int { t.Fatal("roll must not be called for inert effect"); return 0 },
		}
		mortalBlowTryProc(logrus.New(), deps, se, 42, mbField(), 7, uint32(skill3.RangerMortalBlowId))
		if fetched {
			t.Fatal("snapshot must not be fetched for inert effect")
		}
	}
}

// TestMortalBlowTryProc_SnapshotErrorSwallowed — a failed monster snapshot
// fetch (despawn race) is logged and swallowed; no roll, no emit (FR-5).
func TestMortalBlowTryProc_SnapshotErrorSwallowed(t *testing.T) {
	deps := mortalBlowDeps{
		getMonster: func(uint32) (monster.Model, error) {
			return monster.Model{}, errors.New("monster gone")
		},
		emitKill: func(field.Model, uint32, uint32, uint32) error {
			t.Fatal("emitKill must not be called on snapshot error")
			return nil
		},
		roll: func() int { t.Fatal("roll must not be called on snapshot error"); return 0 },
	}
	mortalBlowTryProc(logrus.New(), deps, mbEffect(t, 20, 5), 42, mbField(), 7, uint32(skill3.RangerMortalBlowId))
}

// TestMortalBlowTryProc_AboveThresholdNoRoll — monster HP above
// maxHp·x/100 never rolls (maxHp=1000, x=20 -> threshold 200; hp=201).
func TestMortalBlowTryProc_AboveThresholdNoRoll(t *testing.T) {
	deps := mortalBlowDeps{
		getMonster: func(uint32) (monster.Model, error) { return mbMonster(t, 42, 201, 1000), nil },
		emitKill: func(field.Model, uint32, uint32, uint32) error {
			t.Fatal("emitKill must not be called above threshold")
			return nil
		},
		roll: func() int { t.Fatal("roll must not be called above threshold"); return 0 },
	}
	mortalBlowTryProc(logrus.New(), deps, mbEffect(t, 20, 5), 42, mbField(), 7, uint32(skill3.RangerMortalBlowId))
}

// TestMortalBlowTryProc_RollFailNoEmit — at threshold, roll y+1 misses.
func TestMortalBlowTryProc_RollFailNoEmit(t *testing.T) {
	deps := mortalBlowDeps{
		getMonster: func(uint32) (monster.Model, error) { return mbMonster(t, 42, 200, 1000), nil },
		emitKill: func(field.Model, uint32, uint32, uint32) error {
			t.Fatal("emitKill must not be called on failed roll")
			return nil
		},
		roll: func() int { return 6 }, // y=5 -> 6 misses
	}
	mortalBlowTryProc(logrus.New(), deps, mbEffect(t, 20, 5), 42, mbField(), 7, uint32(skill3.RangerMortalBlowId))
}

// TestMortalBlowTryProc_ProcEmitsKill — at threshold with roll == y the
// kill is emitted with the caster, monster, and skill id.
func TestMortalBlowTryProc_ProcEmitsKill(t *testing.T) {
	var gotMonster, gotCharacter, gotSkill uint32
	emitted := false
	deps := mortalBlowDeps{
		getMonster: func(uint32) (monster.Model, error) { return mbMonster(t, 42, 200, 1000), nil },
		emitKill: func(_ field.Model, monsterId uint32, characterId uint32, skillId uint32) error {
			emitted = true
			gotMonster, gotCharacter, gotSkill = monsterId, characterId, skillId
			return nil
		},
		roll: func() int { return 5 }, // y=5 -> 5 procs
	}
	mortalBlowTryProc(logrus.New(), deps, mbEffect(t, 20, 5), 42, mbField(), 7, uint32(skill3.SniperMortalBlowId))
	if !emitted {
		t.Fatal("expected KILL emit")
	}
	if gotMonster != 42 || gotCharacter != 7 || gotSkill != uint32(skill3.SniperMortalBlowId) {
		t.Fatalf("emitKill(monster=%d, character=%d, skill=%d), want (42, 7, %d)", gotMonster, gotCharacter, gotSkill, uint32(skill3.SniperMortalBlowId))
	}
}

// TestMortalBlowTryProc_EmitErrorSwallowed — a failed KILL emit is logged
// and swallowed; mortalBlowTryProc returns normally (FR-5).
func TestMortalBlowTryProc_EmitErrorSwallowed(t *testing.T) {
	deps := mortalBlowDeps{
		getMonster: func(uint32) (monster.Model, error) { return mbMonster(t, 42, 200, 1000), nil },
		emitKill: func(field.Model, uint32, uint32, uint32) error {
			return errors.New("kafka down")
		},
		roll: func() int { return 1 },
	}
	// Must not panic and must return normally.
	mortalBlowTryProc(logrus.New(), deps, mbEffect(t, 20, 5), 42, mbField(), 7, uint32(skill3.RangerMortalBlowId))
}

// --- onDamageApplied gating through processDamageInfoEntry -------------
//
// Reflected and status-only entries must never reach onDamageApplied (and
// therefore never proc Mortal Blow); a plain damage entry must reach it.

func mbEntryDeps(onDamageApplied func(uint32)) damageInfoEntryDeps {
	return damageInfoEntryDeps{
		getReflect: func(tenant.Model, uint32, string) (monster.ReflectInfo, bool) {
			return monster.ReflectInfo{}, false
		},
		getMonster: func(uint32) (monster.Model, error) { return monster.Model{}, nil },
		applyDamage: func(field.Model, uint32, uint32, []uint32, byte) error {
			return nil
		},
		emitReflectDamage: func(field.Model, uint32, uint32, uint32, uint32, string) error {
			return nil
		},
		applyStatus: func(field.Model, uint32, uint32, uint32, uint32, map[string]int32, uint32) error {
			return nil
		},
		loadVenomStats:  func() effective_stats.RestModel { return effective_stats.RestModel{} },
		onDamageApplied: onDamageApplied,
	}
}

// TestProcessDamageInfoEntry_DamageEntryReachesOnDamageApplied — the happy
// path invokes the callback once with the entry's monster id.
func TestProcessDamageInfoEntry_DamageEntryReachesOnDamageApplied(t *testing.T) {
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	var got []uint32
	deps := mbEntryDeps(func(monsterId uint32) { got = append(got, monsterId) })

	di := *packetmodel.NewDamageInfo(1).SetMonsterId(42).SetDamages([]uint32{100})
	ai := *packetmodel.NewAttackInfo(packetmodel.AttackTypeRanged).SetSkillId(uint32(skill3.RangerMortalBlowId))
	processDamageInfoEntry(logrus.New(), di, ai, mbEffect(t, 20, 5), 1, 7, 0, 0, mbField(), tm, monster2const.ReflectKindPhysical, deps)

	if len(got) != 1 || got[0] != 42 {
		t.Fatalf("onDamageApplied calls = %v, want [42]", got)
	}
}

// TestProcessDamageInfoEntry_StatusOnlyEntrySkipsOnDamageApplied — an entry
// with no damage lines never reaches the proc callback.
func TestProcessDamageInfoEntry_StatusOnlyEntrySkipsOnDamageApplied(t *testing.T) {
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	invoked := false
	deps := mbEntryDeps(func(uint32) { invoked = true })

	di := *packetmodel.NewDamageInfo(0).SetMonsterId(42)
	ai := *packetmodel.NewAttackInfo(packetmodel.AttackTypeRanged).SetSkillId(uint32(skill3.RangerMortalBlowId))
	processDamageInfoEntry(logrus.New(), di, ai, mbEffect(t, 20, 5), 1, 7, 0, 0, mbField(), tm, monster2const.ReflectKindPhysical, deps)

	if invoked {
		t.Fatal("onDamageApplied must not run for a status-only entry")
	}
}

// TestProcessDamageInfoEntry_ReflectedEntrySkipsOnDamageApplied — when the
// monster reflects the hit, damage is not applied and the proc callback
// never runs.
func TestProcessDamageInfoEntry_ReflectedEntrySkipsOnDamageApplied(t *testing.T) {
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	invoked := false
	deps := mbEntryDeps(func(uint32) { invoked = true })
	deps.getReflect = func(tenant.Model, uint32, string) (monster.ReflectInfo, bool) {
		return monster.ReflectInfo{
			Kind:      monster2const.ReflectKindPhysical,
			Percent:   30,
			LtX:       -100,
			LtY:       -100,
			RbX:       100,
			RbY:       100,
			MaxDamage: 9999,
			ExpiresAt: time.Now().Add(time.Minute),
		}, true
	}
	deps.getMonster = func(uint32) (monster.Model, error) {
		return mbMonster(t, 42, 500, 1000), nil
	}

	di := *packetmodel.NewDamageInfo(1).SetMonsterId(42).SetDamages([]uint32{100})
	ai := *packetmodel.NewAttackInfo(packetmodel.AttackTypeRanged).SetSkillId(uint32(skill3.RangerMortalBlowId))
	processDamageInfoEntry(logrus.New(), di, ai, mbEffect(t, 20, 5), 1, 7, 0, 0, mbField(), tm, monster2const.ReflectKindPhysical, deps)

	if invoked {
		t.Fatal("onDamageApplied must not run for a reflected entry")
	}
}
```

Update the test file's import block to (final state):

```go
import (
	"errors"
	"math"
	"testing"
	"time"

	"atlas-channel/data/skill/effect"
	"atlas-channel/effective_stats"
	"atlas-channel/monster"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	monster2const "github.com/Chronicle20/atlas/libs/atlas-constants/monster"
	skill3 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)
```

(`MustBuild()` is safe here: `Build()`'s only invariant is `uniqueId != 0` — verified at `services/atlas-channel/atlas.com/channel/monster/builder.go:110-113` — and `mbMonster` always passes a non-zero id.)

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run 'TestMortalBlowTryProc|TestProcessDamageInfoEntry' -v`
Expected: FAIL to compile with `undefined: mortalBlowDeps` / `undefined: mortalBlowTryProc`

- [ ] **Step 3: Write `mortalBlowDeps` and `mortalBlowTryProc`**

Insert into `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go`, after the Task 2 helpers (i.e., after `isMortalBlowAttack`):

```go
// mortalBlowDeps groups the seams mortalBlowTryProc needs so tests can
// drive every branch (snapshot miss, threshold, roll, emit failure)
// without a real monster.Processor or Kafka — same pattern as
// damageInfoEntryDeps. Production wiring: mp.GetById, mp.Kill, and
// rand.Intn(100)+1.
type mortalBlowDeps struct {
	getMonster func(monsterId uint32) (monster.Model, error)
	emitKill   func(f field.Model, monsterId uint32, characterId uint32, skillId uint32) error
	// roll returns a uniform integer in [1,100].
	roll func() int
}

// mortalBlowTryProc evaluates and (on success) emits the Mortal Blow
// instant kill for one damaged monster. Called once per damaged monster
// after damage and status apply, only for ranged attacks tagged with the
// Mortal Blow skill ids (the attack's skill IS the passive, so se is
// already resolved at the character's owned level — no extra effect
// lookup). The threshold reads the channel's monster snapshot, which
// reflects pre-attack HP (damage propagates to atlas-monsters
// asynchronously); that is the specified Cosmic-parity timing (FR-2).
// Boss exclusion is enforced authoritatively by atlas-monsters — the
// snapshot carries no boss flag. Errors are logged at Debugf/Errorf and
// swallowed — never abort the surrounding attack pipeline (FR-5).
func mortalBlowTryProc(
	l logrus.FieldLogger,
	deps mortalBlowDeps,
	se effect.Model,
	monsterId uint32,
	f field.Model,
	characterId uint32,
	skillId uint32,
) {
	x, y := se.X(), se.Y()
	if x <= 0 || y <= 0 {
		return
	}

	mon, err := deps.getMonster(monsterId)
	if err != nil {
		l.WithError(err).Debugf("Mortal Blow: monster [%d] snapshot fetch failed.", monsterId)
		return
	}

	if !mortalBlowEligible(mon.Hp(), mon.MaxHp(), x) {
		return
	}

	roll := deps.roll()
	l.Debugf("Mortal Blow threshold pass: caster=[%d] skill=[%d] monster=[%d] (hp=%d maxHp=%d x=%d) roll=[%d] y=[%d].",
		characterId, skillId, monsterId, mon.Hp(), mon.MaxHp(), x, roll, y)
	if !mortalBlowKillRoll(roll, y) {
		return
	}

	l.Debugf("Mortal Blow proc: caster=[%d] skill=[%d] monster=[%d] roll=[%d].", characterId, skillId, monsterId, roll)
	if err := deps.emitKill(f, monsterId, characterId, skillId); err != nil {
		l.WithError(err).Errorf("Mortal Blow: KILL emit failed for monster [%d] caster [%d].", monsterId, characterId)
	}
}
```

- [ ] **Step 4: Wire the closure and delete the TODO**

In `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go`, replace the `onDamageApplied` closure (lines 348–355, inside the `deps := damageInfoEntryDeps{...}` literal in `processAttack`):

```go
			// MP Eater proc: per-monster, after status apply,
			// magic attacks only. Failures are swallowed so the
			// rest of the attack pipeline is unaffected.
			onDamageApplied: func(monsterId uint32) {
				if ai.AttackType() == packetmodel.AttackTypeMagic && ai.SkillId() > 0 {
					mpEaterTryProc(l, ctx, mp, c, monsterId, s.Field(), s.CharacterId())
				}
				// Mortal Blow proc: per-monster, ranged attacks tagged
				// with the Ranger/Sniper Mortal Blow skill id only.
				// Ownership was already enforced upstream (unowned
				// skill ids destroy the session). Failures are
				// swallowed (FR-5).
				if isMortalBlowAttack(ai.AttackType(), ai.SkillId()) {
					mortalBlowTryProc(l, mortalBlowDeps{
						getMonster: mp.GetById,
						emitKill:   mp.Kill,
						roll:       func() int { return rand.Intn(100) + 1 },
					}, se, monsterId, s.Field(), s.CharacterId(), ai.SkillId())
				}
			},
```

Then delete the line `// TODO Mortal Blow` from the TODO block at the end of `processAttack` (line ~421). Leave every other TODO line untouched.

- [ ] **Step 5: Run the tests and the full handler package**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -v -run 'TestMortalBlow|TestIsMortalBlow|TestProcessDamageInfoEntry' && go test -race ./socket/... && go build ./...`
Expected: all PASS, build clean

- [ ] **Step 6: Verify the TODO is gone**

Run: `grep -n "Mortal Blow" services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go | grep TODO`
Expected: no output (exit 1)

- [ ] **Step 7: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go services/atlas-channel/atlas.com/channel/socket/handler/character_attack_mortal_blow_test.go
git commit -m "feat(task-152): Mortal Blow proc in processAttack ranged path"
```

---

### Task 5: Correct the stale task-007 comment (FR-6, Decision 9)

IDA verification (PRD §4.5) proved the Mortal Blow shot consumes an arrow client-side; the `TODO(task-007)` comment wrongly lists it as a passive no-consume mechanic. Remove Mortal Blow from the list, leaving Expert Marksmanship / Claw Mastery untouched.

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_projectile.go:118-121`

**Interfaces:**
- Consumes: nothing. Produces: nothing (comment-only change; no behavior).

- [ ] **Step 1: Edit the comment**

In `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_projectile.go`, replace lines 118–121:

```go
	// TODO(task-007): passive no-consume mechanics (Mortal Blow, Expert Marksmanship,
	// Claw Mastery roll-to-preserve, etc.) are out of scope for v1. These require
	// reading passive skill levels and performing an RNG roll against a per-skill
	// probability to skip the consume. When added, apply before the resolvePlan call.
```

with:

```go
	// TODO(task-007): passive no-consume mechanics (Expert Marksmanship,
	// Claw Mastery roll-to-preserve, etc.) are out of scope for v1. These require
	// reading passive skill levels and performing an RNG roll against a per-skill
	// probability to skip the consume. When added, apply before the resolvePlan call.
	// (Mortal Blow is NOT in this list: IDA-verified on GMS v83, the Mortal Blow
	// shot consumes an arrow client-side like a normal shot — task-152 PRD §4.5.)
```

- [ ] **Step 2: Build and run the projectile tests**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./socket/handler/ -run TestPlan -v`
Expected: build clean; existing projectile tests PASS (comment-only change)

- [ ] **Step 3: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_attack_projectile.go
git commit -m "docs(task-152): Mortal Blow consumes an arrow — fix stale task-007 comment (FR-6)"
```

---

### Task 6: Split `Damage` into `checkReflect` + `damageCore` (Decision 6 refactor)

> **⚠ SUPERSEDED by Main-Merge Reconciliation — `checkReflect` ALREADY EXISTS on main; the verbatim `damageCore` below would REVERT main's GM-hidden controller-switch guard and its `information.NewProcessor(...).GetById` call. Use the corrected split in the reconciliation, NOT the block below.**

Behavior-preserving refactor in atlas-monsters: everything from the info-fetch down moves into an unexported `damageCore(m, characterId, damages)`. `Damage` keeps its exact behavior (registry fetch → alive check → `checkReflect` → `damageCore`). This gives Task 7's `Kill` a delivery path that never rolls reflect. No new tests — the existing `processor_test.go` damage suite (DAMAGED/KILLED ordering, controller switch, aggro) must stay green, which is this task's acceptance gate.

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/processor.go:355-496` (the `Damage` method)

**Interfaces:**
- Consumes: everything `Damage` already uses (registry, information, emitters, pickers).
- Produces: `func (p *ProcessorImpl) damageCore(m Model, characterId uint32, damages []uint32)` — Task 7's `Kill` calls it with a single `math.MaxUint32` line. `Damage`'s public signature and behavior are unchanged.

- [ ] **Step 1: Refactor**

In `services/atlas-monsters/atlas.com/monsters/monster/processor.go`, replace the `Damage` method (lines 355–496) with:

```go
// Damage applies a sequence of damage lines from a single attack to a monster.
// Lines are applied in order; if any line kills the monster, later lines are
// dropped (overkill discarded). Always emits a `damaged` event reflecting the
// final state, plus a `killed` event when the attack lands a kill, so the
// channel writes the final HP-bar packet before the death animation.
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

	p.damageCore(m, characterId, damages)
}

// damageCore applies damage lines to an already-fetched, alive monster and
// runs the full post-damage flow: damaged event, damage picker, kill
// handling (cooldown/drop-timer clears, status-cancel emits, killed event,
// registry removal, revives), controller switch, and aggro emission.
// Callers own the guards that precede it: Damage does the alive check and
// reflect; Kill (Mortal Blow) does the alive check and the fail-closed
// boss guard, and deliberately never rolls reflect — the channel already
// gated the triggering hit on reflect, and a kill "attack" has no attack
// type.
func (p *ProcessorImpl) damageCore(m Model, characterId uint32, damages []uint32) {
	// Fetch monster info for boss flag and revives
	var isBoss bool
	var revives []uint32
	if ma, infoErr := information.GetById(p.l)(p.ctx)(m.MonsterId()); infoErr == nil {
		isBoss = ma.Boss()
		revives = ma.Revives()
	}

	oldHpPercentage := m.HpPercentage()

	var last DamageSummary
	hasLast := false
	killed := false
	firstHitObserved := false
	nowMs := time.Now().UnixMilli()
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
			break // discard overkill
		}
	}

	if !hasLast {
		return
	}

	// Always emit damaged so the channel writes the final HP-bar packet,
	// even when the attack lands a kill.
	if err := p.emit(EnvEventTopicMonsterStatus, damagedStatusEventProvider(last.Monster, last.CharacterId, last.CharacterId, isBoss, DamageSourceCharacterAttack, last.Monster.DamageSummary())); err != nil {
		p.l.WithError(err).Errorf("Monster [%d] damaged, but unable to display that for the characters in the field.", last.Monster.UniqueId())
	}

	// FR-3.1: Fire the picker on every first hit (so a missed attack that
	// flips controllerHasAggro can begin casting), and on every subsequent hit
	// that changes HP percentage.
	if !killed && (firstHitObserved || last.Monster.HpPercentage() != oldHpPercentage) {
		if err := p.RepickAndEmit(last.Monster.UniqueId(), RepickReasonDamaged); err != nil {
			p.l.WithError(err).Warnf("Damage picker: monster [%d] re-pick failed.", last.Monster.UniqueId())
		}
	}

	if killed {
		// Clear cooldowns and drop timer on death
		GetCooldownRegistry().ClearCooldowns(p.ctx, p.t, m.UniqueId())
		GetAttackCooldownRegistry().ClearCooldowns(p.ctx, p.t, m.UniqueId())
		GetDropTimerRegistry().Unregister(p.ctx, p.t, m.UniqueId())

		// Emit cancellation events for any active status effects before death
		for _, se := range last.Monster.StatusEffects() {
			_ = p.emit(EnvEventTopicMonsterStatus, statusEffectCancelledEventProvider(last.Monster, se))
		}

		if err := p.emit(EnvEventTopicMonsterStatus, killedStatusEventProvider(last.Monster, last.CharacterId, isBoss, last.Monster.DamageSummary())); err != nil {
			p.l.WithError(err).Errorf("Monster [%d] killed, but unable to display that for the characters in the field.", last.Monster.UniqueId())
		}
		if _, err := GetMonsterRegistry().RemoveMonster(p.ctx, p.t, last.Monster.UniqueId()); err != nil {
			p.l.WithError(err).Errorf("Monster [%d] killed, but not removed from registry.", last.Monster.UniqueId())
		}

		// Boss revive: spawn next phase monsters
		if len(revives) > 0 {
			p.spawnRevives(last.Monster, revives)
		}
		return
	}

	// Controller-switch and aggro-flag emission.
	//
	// Decision 4 (PRD §8.4): keep the two-step StopControl + StartControl
	// rather than collapsing into a single Lua. Two concurrent damage events
	// for the same monster could interleave and produce redundant
	// STOP_CONTROL/START_CONTROL pairs; this is acceptable because Kafka
	// partition ordering preserves causality and the channel re-applies
	// idempotently for re-control to the same character.
	controllerSwitched := false
	// Controller-switch on DPS lead applies to bosses too. Only the decay sweep
	// (MonsterAggroDecayTask) treats bosses specially.
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

The body of `damageCore` is the moved tail of the old `Damage` with exactly one mechanical substitution: the three kill-path cleanup calls use `m.UniqueId()` instead of the old `id` parameter (same value — `m` was fetched by `id`). Everything else is verbatim.

- [ ] **Step 2: Run the full existing monster suite to prove behavior is preserved**

Run: `cd services/atlas-monsters/atlas.com/monsters && go test -race ./monster/ && go vet ./... && go build ./...`
Expected: PASS (all existing tests, including the DAMAGED/KILLED ordering, controller-switch, and aggro tests), vet and build clean

- [ ] **Step 3: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/processor.go
git commit -m "refactor(task-152): split Damage into checkReflect + damageCore (no behavior change)"
```

---

### Task 7: `Kill` on the atlas-monsters processor (FR-4, Decision 6)

> **⚠ SUPERSEDED IN PART by Main-Merge Reconciliation — replace every `information.GetById(p.l)(p.ctx)(...)` in the body below with `information.NewProcessor(p.l, p.ctx).GetById(...)`. Everything else (fail-closed boss guard, `math.MaxUint32` line, `testInformationLookup` seam) is unchanged.**

The authoritative half: re-check alive, re-check boss FAIL-CLOSED via the existing `testInformationLookup` seam, then deliver a single `math.MaxUint32` line through `damageCore`. `Registry.ApplyDamage` clamps the recorded entry to the HP actually removed (verified `registry.go:427-483`), so EXP/drop credit stays honest — this resolves Decision 6's verify-then-pick in favor of `MaxUint32`.

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/processor.go` (interface at line 26-58; new method after `DrainMp`; add `"math"` import)
- Test: `services/atlas-monsters/atlas.com/monsters/monster/kill_test.go` (create)

**Interfaces:**
- Consumes: `damageCore` (Task 6), `GetMonsterRegistry()`, `testInformationLookup` seam, `information.GetById`, `newRecordingProcessorWithBodies` test helper (`processor_test.go:234`), `information.NewModelBuilder().SetBoss(bool).Build()`.
- Produces: `Kill(uniqueId uint32, characterId uint32, skillId uint32)` on the `Processor` interface and `ProcessorImpl` — Task 8's `handleKillCommand` calls it. (Void return like `Damage`: every reject path is a silent drop with nothing to communicate back.)

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-monsters/atlas.com/monsters/monster/kill_test.go`:

```go
package monster

import (
	"atlas-monsters/monster/information"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
)

// TestKill_NonBoss_KilledAndRemoved — a Mortal Blow kill on a non-boss at
// any HP emits DAMAGED then KILLED, removes the monster from the registry,
// and credits the full remaining HP to the killer in the damage summary
// (ApplyDamage clamps the MaxUint32 line to the HP actually removed).
func TestKill_NonBoss_KilledAndRemoved(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	prevHook := testInformationLookup
	testInformationLookup = func(_ uint32) (information.Model, error) {
		return information.NewModelBuilder().SetBoss(false).Build(), nil
	}
	defer func() { testInformationLookup = prevHook }()

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 1000000, 0, 0, 0, 5, 0, 5000, 100)
	uniqueId := m.UniqueId()

	p, events := newRecordingProcessorWithBodies(t, ten)
	p.Kill(uniqueId, 42, 3110001)

	if len(*events) != 2 {
		t.Fatalf("expected 2 events (DAMAGED, KILLED), got %d: %v", len(*events), *events)
	}
	if (*events)[0].Type != EventMonsterStatusDamaged {
		t.Errorf("event[0].Type = %q, want %q", (*events)[0].Type, EventMonsterStatusDamaged)
	}
	if (*events)[1].Type != EventMonsterStatusKilled {
		t.Errorf("event[1].Type = %q, want %q", (*events)[1].Type, EventMonsterStatusKilled)
	}

	var body statusEventKilledBody
	if err := json.Unmarshal((*events)[1].Body, &body); err != nil {
		t.Fatalf("decode KILLED body: %v", err)
	}
	if body.ActorId != 42 {
		t.Errorf("KILLED.ActorId = %d, want 42", body.ActorId)
	}
	if len(body.DamageEntries) != 1 {
		t.Fatalf("KILLED.DamageEntries = %v, want exactly 1 entry", body.DamageEntries)
	}
	if body.DamageEntries[0].CharacterId != 42 {
		t.Errorf("DamageEntries[0].CharacterId = %d, want 42", body.DamageEntries[0].CharacterId)
	}
	if body.DamageEntries[0].Damage != 5000 {
		t.Errorf("DamageEntries[0].Damage = %d, want 5000 (clamped to HP removed, not MaxUint32)", body.DamageEntries[0].Damage)
	}

	if _, err := r.GetMonster(ten, uniqueId); err == nil {
		t.Errorf("expected monster [%d] removed from registry after kill", uniqueId)
	}
}

// TestKill_Boss_Dropped — the authoritative boss guard: no events, monster
// untouched, regardless of what the channel decided.
func TestKill_Boss_Dropped(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	prevHook := testInformationLookup
	testInformationLookup = func(_ uint32) (information.Model, error) {
		return information.NewModelBuilder().SetBoss(true).Build(), nil
	}
	defer func() { testInformationLookup = prevHook }()

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 8800000, 0, 0, 0, 5, 0, 50000, 3000)
	uniqueId := m.UniqueId()

	p, events := newRecordingProcessorWithBodies(t, ten)
	p.Kill(uniqueId, 42, 3110001)

	if len(*events) != 0 {
		t.Fatalf("expected 0 events for boss kill attempt, got %d: %v", len(*events), *events)
	}
	got, err := r.GetMonster(ten, uniqueId)
	if err != nil {
		t.Fatalf("boss must remain in registry: %v", err)
	}
	if got.Hp() != 50000 {
		t.Errorf("boss HP = %d, want 50000 (untouched)", got.Hp())
	}
}

// TestKill_InfoLookupError_DroppedFailClosed — if the boss lookup errors,
// the kill is dropped. This deliberately diverges from DrainMp's fail-open:
// losing a legitimate proc during an atlas-data hiccup is acceptable;
// killing a boss is not (FR-4).
func TestKill_InfoLookupError_DroppedFailClosed(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	prevHook := testInformationLookup
	testInformationLookup = func(_ uint32) (information.Model, error) {
		return information.Model{}, errors.New("atlas-data unavailable")
	}
	defer func() { testInformationLookup = prevHook }()

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 1000000, 0, 0, 0, 5, 0, 5000, 100)
	uniqueId := m.UniqueId()

	p, events := newRecordingProcessorWithBodies(t, ten)
	p.Kill(uniqueId, 42, 3110001)

	if len(*events) != 0 {
		t.Fatalf("expected 0 events on info-lookup error (fail-closed), got %d: %v", len(*events), *events)
	}
	got, err := r.GetMonster(ten, uniqueId)
	if err != nil {
		t.Fatalf("monster must remain in registry: %v", err)
	}
	if got.Hp() != 5000 {
		t.Errorf("HP = %d, want 5000 (untouched)", got.Hp())
	}
}

// TestKill_MissingMonster_NoOp — the triggering attack already killed the
// monster (DAMAGE and KILL share a partition; DAMAGE processed first) or
// it despawned. Nothing to do, nothing emitted.
func TestKill_MissingMonster_NoOp(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	p, events := newRecordingProcessorWithBodies(t, ten)
	p.Kill(99999999, 42, 3110001)

	if len(*events) != 0 {
		t.Fatalf("expected 0 events for missing monster, got %d: %v", len(*events), *events)
	}
}

// TestKill_DeadMonster_NoOp — a registry entry at HP 0 (killed but not yet
// removed) is dropped without events.
func TestKill_DeadMonster_NoOp(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	prevHook := testInformationLookup
	testInformationLookup = func(_ uint32) (information.Model, error) {
		return information.NewModelBuilder().SetBoss(false).Build(), nil
	}
	defer func() { testInformationLookup = prevHook }()

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 1000000, 0, 0, 0, 5, 0, 1, 50)
	uniqueId := m.UniqueId()
	// Kill it directly via the registry (no emit) so HP=0 but it remains present.
	if _, err := r.ApplyDamage(ten, 1, 999, uniqueId, 1); err != nil {
		t.Fatalf("seed ApplyDamage: %v", err)
	}

	p, events := newRecordingProcessorWithBodies(t, ten)
	p.Kill(uniqueId, 42, 3110001)

	if len(*events) != 0 {
		t.Fatalf("expected 0 events for dead monster, got %d: %v", len(*events), *events)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-monsters/atlas.com/monsters && go test ./monster/ -run TestKill_ -v`
Expected: FAIL to compile with `p.Kill undefined`

- [ ] **Step 3: Implement `Kill`**

In `services/atlas-monsters/atlas.com/monsters/monster/processor.go`:

1. Add `"math"` to the import block (it currently has `"math/rand"` only).
2. Add to the `Processor` interface, in the `// Commands` group after `DrainMp` (line 57):

```go
	Kill(uniqueId uint32, characterId uint32, skillId uint32)
```

3. Add the method after `DrainMp` (after line 1491):

```go
// Kill delivers a Mortal Blow instant kill through the shared damage core.
// The channel is the authority for the proc decision (threshold and kill
// roll against the tenant's skill data); this method owns the guards only
// it can enforce: the monster must still be present and alive, and the
// boss flag is re-checked authoritatively. The boss lookup is FAIL-CLOSED
// — if it errors, the kill is dropped. This deliberately diverges from
// DrainMp's fail-open lookup: losing a legitimate proc during an
// atlas-data hiccup is acceptable; killing a boss is not (FR-4).
//
// The kill line is math.MaxUint32 (Cosmic parity: Integer.MAX_VALUE).
// Registry.ApplyDamage clamps the recorded damage entry to the HP actually
// removed, so the damage summary that drives EXP and drop credit stays
// honest. No reflect is rolled — the channel already gated the triggering
// hit on reflect, and a kill "attack" has no attack type.
//
// Missing or dead monsters are silent drops: DAMAGE (the triggering
// attack) and KILL are keyed by the monster's unique id, so they share a
// partition and the attack's own kill may have removed the monster before
// this command lands. SkillId is traceability only.
//
// The boss check uses testInformationLookup when non-nil so unit tests can
// stub the lookup without an HTTP round-trip to atlas-data.
func (p *ProcessorImpl) Kill(uniqueId uint32, characterId uint32, skillId uint32) {
	m, err := GetMonsterRegistry().GetMonster(p.t, uniqueId)
	if err != nil {
		p.l.Debugf("KILL: monster [%d] not found; the triggering attack likely already killed it. Skill [%d].", uniqueId, skillId)
		return
	}
	if !m.Alive() {
		p.l.Debugf("KILL: monster [%d] already dead. Skill [%d].", uniqueId, skillId)
		return
	}

	var info information.Model
	var infoErr error
	if testInformationLookup != nil {
		info, infoErr = testInformationLookup(m.MonsterId())
	} else {
		info, infoErr = information.GetById(p.l)(p.ctx)(m.MonsterId())
	}
	if infoErr != nil {
		p.l.WithError(infoErr).Errorf("KILL: boss lookup failed for monster [%d]; dropping kill (fail-closed).", uniqueId)
		return
	}
	if info.Boss() {
		p.l.Debugf("KILL: monster [%d] is a boss; dropping kill from character [%d] skill [%d].", uniqueId, characterId, skillId)
		return
	}

	p.l.Debugf("Mortal Blow kill: monster [%d] by character [%d] via skill [%d].", uniqueId, characterId, skillId)
	p.damageCore(m, characterId, []uint32{math.MaxUint32})
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-monsters/atlas.com/monsters && go test -race ./monster/ -run TestKill_ -v && go test -race ./monster/ && go vet ./... && go build ./...`
Expected: new tests PASS, full monster suite still PASS, vet/build clean

- [ ] **Step 5: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/processor.go services/atlas-monsters/atlas.com/monsters/monster/kill_test.go
git commit -m "feat(task-152): authoritative Mortal Blow Kill with fail-closed boss guard"
```

---

### Task 8: atlas-monsters `KILL` consumer arm (Decision 7)

Mirror the command constant and body in the consumer package and register the type-gated handler, following the `DRAIN_MP` siblings exactly. The consumer package has no unit tests (handlers are thin type-gated delegates); compile + the Task 7 processor tests are the coverage, matching every sibling command.

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka.go`
- Modify: `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/consumer.go`

**Interfaces:**
- Consumes: `monster.NewProcessor(l, ctx).Kill(uniqueId, characterId, skillId)` (Task 7); the JSON body mirrors Task 3's `KillCommandBody` (`characterId`, `skillId`).
- Produces: nothing further — this closes the Kafka loop.

- [ ] **Step 1: Add the constant and body**

In `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka.go`, add to the const block (after `CommandTypeRemovePuppet = "REMOVE_PUPPET"`, line 25):

```go
	CommandTypeKill              = "KILL"
```

And add after `drainMpCommandBody` (line 102):

```go
// killCommandBody asks the processor to kill a monster outright as the
// result of a player passive (Mortal Blow). The channel already rolled
// the threshold and kill chance; the processor re-checks alive + boss
// (fail-closed). SkillId is carried for traceability/logging only.
type killCommandBody struct {
	CharacterId uint32 `json:"characterId"`
	SkillId     uint32 `json:"skillId"`
}
```

- [ ] **Step 2: Add and register the handler**

In `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/consumer.go`, add the handler after `handleDrainMpCommand` (line 171):

```go
func handleKillCommand(l logrus.FieldLogger, ctx context.Context, c command[killCommandBody]) {
	if c.Type != CommandTypeKill {
		return
	}

	p := monster.NewProcessor(l, ctx)
	p.Kill(c.MonsterId, c.Body.CharacterId, c.Body.SkillId)
}
```

And register it in `InitHandlers`, after the `handleDrainMpCommand` registration (line 50-52):

```go
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleKillCommand))); err != nil {
			return err
		}
```

- [ ] **Step 3: Build and test the module**

Run: `cd services/atlas-monsters/atlas.com/monsters && go build ./... && go test -race ./... && go vet ./...`
Expected: all clean

- [ ] **Step 4: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka.go services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/consumer.go
git commit -m "feat(task-152): consume KILL monster command in atlas-monsters"
```

---

### Task 9: Full verification sweep (design §7, CLAUDE.md)

Everything below runs from the worktree root except where noted. All of it must pass before the branch is called done — `go build`/`go test` do NOT catch Dockerfile COPY gaps; only bake does.

**Files:** none (verification only).

**Interfaces:** none.

- [ ] **Step 1: Module test/vet/build — atlas-channel**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./... && go vet ./... && go build ./...`
Expected: all clean

- [ ] **Step 2: Module test/vet/build — atlas-monsters**

Run: `cd services/atlas-monsters/atlas.com/monsters && go test -race ./... && go vet ./... && go build ./...`
Expected: all clean

- [ ] **Step 3: Docker bake both services (from the worktree root)**

Run: `docker buildx bake atlas-channel atlas-monsters`
Expected: both images build clean. (No shared-lib changes in this task, so no Dockerfile COPY additions are expected — but bake is still mandatory.)

- [ ] **Step 4: Redis key guard (from the worktree root)**

Run: `tools/redis-key-guard.sh`
Expected: clean (no raw keyed go-redis calls added — this task adds none).

- [ ] **Step 5: Acceptance-criteria walk**

Check each PRD §11 box against the code (cite file:line in the task notes):
- Skill-id gating incl. melee/magic exclusion → Task 2 tests.
- Threshold + roll from tenant data at owned level → Task 4 (`se.X()`/`se.Y()`).
- EXP/drops credit via standard kill flow → Task 7 test (`DamageEntries` credit).
- Boss never killed → Task 7 boss + fail-closed tests.
- Pipeline survives proc failures → Task 4 swallow tests.
- Projectile unchanged + comment fixed → Task 5.
- `Y()` threaded from REST → Task 1 test.

- [ ] **Step 6: Commit any stragglers and stop**

```bash
git status --short
```

Expected: clean tree (every task committed as it landed). Do NOT open a PR yet — code review (`superpowers:requesting-code-review`) runs first, per CLAUDE.md.

---

## Self-Review (completed at plan time)

- **Spec coverage:** FR-1/FR-2/FR-3 → Tasks 2+4; FR-4 → Tasks 6+7 (+Task 3 partition-key note); FR-5 → Task 4 (swallow tests); FR-6 → Task 5; FR-7 → Task 1; FR-8 → Tasks 1,2,4,7 test steps; Decision 7 Kafka surface → Tasks 3+8; Decision 6 verify-then-pick → resolved to `MaxUint32` (clamping verified at `registry.go:427-483`); design §7 verification → Task 9.
- **Deviation from design (declared):** Decision 5's `mortalBlowTryProc(l, mp *monster.Processor, …)` signature is refined into `mortalBlowTryProc(l, deps mortalBlowDeps, …)` — the design's own §5 test matrix (snapshot-error and emit-error isolation) is not writable against the concrete `monster.Processor` (its `GetById` is a live REST call), and `damageInfoEntryDeps` in the same file is the established seam pattern. Production wiring and behavior are exactly Decision 5's flow.
- **Type consistency:** `Kill` channel-side `(f field.Model, monsterId, characterId, skillId uint32) error` (Task 3) matches `mortalBlowDeps.emitKill` (Task 4); monsters-side `Kill(uniqueId, characterId, skillId uint32)` void (Task 7) matches `handleKillCommand` (Task 8); `KillCommandBody`/`killCommandBody` JSON keys match (`characterId`, `skillId`); helper signatures in Task 2 match their uses in Task 4.
- **Placeholder scan:** clean — every code step contains the complete code; all referenced constants and builder invariants were verified against source at plan time (`RangerStrafeId` at `constants.go:860`, `Build()` invariant at `builder.go:110-113`, `ApplyDamage` clamping at `registry.go:427-483`).
