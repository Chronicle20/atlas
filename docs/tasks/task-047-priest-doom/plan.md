# Priest Doom (Skill 2311005) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the remaining gaps so a Priest casting Doom (skill `2311005`) reliably polymorphs every legal target — including element-resistant mobs — into snails for the skill's duration, while bosses stay immune. Also pin the cast → status → packet path with unit tests and add a grep-friendly Doom log line.

**Architecture:** All wiring already exists end-to-end. Four targeted touches: (1) explicit DOOM short-circuit in atlas-monsters' elemental-immunity gate, (2) Doom-gated magic-reflect probe in atlas-channel's empty-damage attack branch via a freshly extracted per-`DamageInfo` helper, (3) Doom-specific Debugf in atlas-channel's monster wrapper, (4) unit tests in atlas-data, atlas-monsters, atlas-channel. No new Kafka topics, REST routes, or constants.

**Tech Stack:** Go (Go modules per service); `atlas-monsters/.../monster/processor.go` registry/processor pattern; atlas-channel socket handler + monster wrapper; atlas-data XML reader (`atlas-data/xml`).

**Source design:** `docs/tasks/task-047-priest-doom/design.md`
**Source PRD:** `docs/tasks/task-047-priest-doom/prd.md`
**Quick context:** `docs/tasks/task-047-priest-doom/context.md`

---

## File Map

| File | Action | Responsibility |
|---|---|---|
| `services/atlas-data/atlas.com/data/skill/reader_test.go` | Modify | Add reader test pinning Doom effect mapping (`MonsterStatus[DOOM]=1`, `Duration>0`). Pure additive append at EOF; no production change. |
| `services/atlas-monsters/atlas.com/monsters/monster/information/builder.go` | Modify | Extend `ModelBuilder` with `SetBoss(bool)` and `SetResistances(map[string]string)` so tests can construct an `information.Model` with the resistance/boss flags `ApplyStatusEffect` reads. |
| `services/atlas-monsters/atlas.com/monsters/monster/processor.go` | Modify | Add explicit DOOM short-circuit at top of `isElementallyImmune` (lines 1116-1131). Extend the existing `testInformationLookup` package var hook (lines 62-64) to also intercept the lookup inside `ApplyStatusEffect` (lines 1085). |
| `services/atlas-monsters/atlas.com/monsters/monster/processor_test.go` | Modify | Append three test cases: DOOM bypasses elemental immunity, DOOM rejected on bosses, DOOM re-apply replaces the existing entry (refreshing semantics — see context.md "Realized behavior" note). |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go` | Modify | Extract the per-`DamageInfo` body of `processAttack` (currently lines 151-216) into a top-level helper `processDamageInfoEntry` that takes its dependencies as explicit closures. Add a Doom-gated magic-reflect probe inside the helper's empty-damage branch in a separate task. |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common_test.go` | Modify | Append three (or four) helper tests: empty-damage Doom triggers `applyStatus`; magic-reflect blocks Doom; multi-target spread routes correctly; (optional) non-Doom empty-damage status is unaffected by the new reflect probe. |
| `services/atlas-channel/atlas.com/channel/monster/processor.go` | Modify | In `Processor.ApplyStatus` (line 70), after the existing generic Debugf, emit a Doom-targeted Debugf when the inbound `statuses` map contains `"DOOM"`. |

No changes to `libs/atlas-packet`, `libs/atlas-constants`, `services/atlas-configurations`, or any Kafka topic/event schema.

---

## Task 0: Pre-flight — verify shared constants and entry points

Smoke-check that the wiring the design assumes still exists. If anything below has shifted in a recent refactor, raise it before writing code.

**Files:**
- Inspect: `libs/atlas-constants/skill/constants.go` (around line 3067 for `PriestDoomId`)
- Inspect: `libs/atlas-constants/monster/status.go:16` (`StatusDoom`)
- Inspect: `services/atlas-data/atlas.com/data/skill/reader.go:351-352` (Doom effect branch)
- Inspect: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go:151-216` (per-`DamageInfo` body)
- Inspect: `services/atlas-monsters/atlas.com/monsters/monster/processor.go:1116-1131` (`isElementallyImmune`) and `1083-1098` (caller)

- [ ] **Step 1: Confirm constants and entry points**

Run, from repo root:

```bash
grep -n "PriestDoomId\s*=" libs/atlas-constants/skill/constants.go
grep -n "StatusDoom\s*=" libs/atlas-constants/monster/status.go
grep -n "skill.PriestDoomId" services/atlas-data/atlas.com/data/skill/reader.go
grep -n "isElementallyImmune\b" services/atlas-monsters/atlas.com/monsters/monster/processor.go
grep -n "for _, di := range ai.DamageInfo()" services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go
```

Expected: each grep returns at least one hit. If any miss, stop and surface to the user — the design rests on these being present.

- [ ] **Step 2: Run baseline tests on the three affected services**

```bash
( cd services/atlas-monsters/atlas.com/monsters && go build ./... && go test ./... )
( cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./... )
( cd services/atlas-data/atlas.com/data && go build ./... && go test ./... )
```

Expected: all three pass on `main`-equivalent. If any fail before this branch's changes, surface to the user — fixing pre-existing red is out of scope.

---

## Task 1: atlas-data — pin Doom effect mapping with a reader test

The first wiring fact every later step depends on: `getEffect(2311005, …)` populates `MonsterStatus[DOOM] = 1` and a non-zero `Duration`. Test it via the existing `Read(...)` provider pattern (mirrors `TestReader_LT_RB_Present` at `reader_test.go:2905`) so the test stays well within how the rest of the suite exercises this code.

**Files:**
- Test: `services/atlas-data/atlas.com/data/skill/reader_test.go` (append at EOF, line 2993)

- [ ] **Step 1: Append the failing test**

Add at the end of `services/atlas-data/atlas.com/data/skill/reader_test.go`:

```go
// TestReader_PriestDoom_MapsDoomStatus pins the atlas-data effect mapping for
// Priest skill 2311005 (Doom): the produced effect must carry MonsterStatus
// {"DOOM": 1} and a non-zero Duration so atlas-channel's empty-damage status
// branch and atlas-monsters' apply path both see what they expect.
func TestReader_PriestDoom_MapsDoomStatus(t *testing.T) {
	l, _ := test.NewNullLogger()
	tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), tn)

	const xmlData = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="231.img">
  <imgdir name="skill">
    <imgdir name="2311005">
      <imgdir name="level">
        <imgdir name="30">
          <int name="time" value="60"/>
          <int name="mpCon" value="35"/>
          <int name="prop" value="100"/>
          <int name="mobCount" value="6"/>
          <vector name="lt" x="-200" y="-100"/>
          <vector name="rb" x="200" y="100"/>
        </imgdir>
      </imgdir>
    </imgdir>
  </imgdir>
</imgdir>`

	rms := Read(l)(ctx)(xml.FromByteArrayProvider([]byte(xmlData)))
	rmm, err := model.CollectToMap[RestModel, string, RestModel](rms, RestModel.GetID, Identity)()
	if err != nil {
		t.Fatal(err)
	}
	rm, ok := rmm["2311005"]
	if !ok {
		t.Fatal("rmm[2311005] does not exist.")
	}
	if len(rm.Effects) != 1 {
		t.Fatalf("len(rm.Effects) = %d, want 1", len(rm.Effects))
	}
	ef := rm.Effects[0]
	if got := ef.MonsterStatus["DOOM"]; got != 1 {
		t.Fatalf("MonsterStatus[DOOM] = %d, want 1", got)
	}
	if ef.Duration <= 0 {
		t.Fatalf("Duration = %d, want > 0", ef.Duration)
	}
}
```

- [ ] **Step 2: Run the new test to verify it passes**

```bash
( cd services/atlas-data/atlas.com/data && go test ./skill -run TestReader_PriestDoom_MapsDoomStatus -v )
```

Expected: PASS. (Production code is unchanged; the Doom branch at `reader.go:351-352` already populates the map. The test pins it.)

If it fails because `Duration` is zero, investigate `reader.go:140-170` — the `time` field is multiplied by 1000 only when the initial `Duration() == -1` branch is reached. The fixture's `<int name="time" value="60"/>` should yield 60000ms; if not, surface to the user before changing the test.

- [ ] **Step 3: Run the full atlas-data suite to confirm no regression**

```bash
( cd services/atlas-data/atlas.com/data && go build ./... && go test ./... )
```

Expected: all pass.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-data/atlas.com/data/skill/reader_test.go
git commit -m "test(atlas-data): pin Priest Doom effect mapping (DOOM=1, Duration>0)"
```

---

## Task 2: atlas-monsters — extend `information.ModelBuilder` for tests

`isElementallyImmune` and `isBossAllowedStatus` both run inside `ApplyStatusEffect` and consume `information.Model`'s `Boss()` / `IsImmuneToElement()`. The current `information.ModelBuilder` only exposes skill/attack/recovery setters (see `builder.go:1-55`), so tests cannot construct a Model with `boss=true` or a custom resistance table. Extend the builder with two setters; the production code path is unchanged.

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/information/builder.go`

- [ ] **Step 1: Add `SetBoss` and `SetResistances` to `ModelBuilder`**

Apply the following patch (showing the full file after the change):

```go
package information

// ModelBuilder provides a minimal fluent interface for constructing Model
// instances in tests. Only the fields tests need are settable.
type ModelBuilder struct {
	skills      []Skill
	attacks     []AttackInfo
	hpRecovery  uint32
	mpRecovery  uint32
	boss        bool
	resistances map[string]string
}

// NewModelBuilder returns a new ModelBuilder with zero values.
func NewModelBuilder() *ModelBuilder {
	return &ModelBuilder{}
}

// SetSkills sets the skill list on the builder.
func (b *ModelBuilder) SetSkills(skills []Skill) *ModelBuilder {
	b.skills = skills
	return b
}

// SetAttacks sets the attacks list on the builder.
func (b *ModelBuilder) SetAttacks(attacks []AttackInfo) *ModelBuilder {
	b.attacks = attacks
	return b
}

func (b *ModelBuilder) SetHpRecovery(v uint32) *ModelBuilder {
	b.hpRecovery = v
	return b
}

func (b *ModelBuilder) SetMpRecovery(v uint32) *ModelBuilder {
	b.mpRecovery = v
	return b
}

// SetBoss sets the boss flag on the builder. Used by tests that drive
// boss-immunity branches in ApplyStatusEffect.
func (b *ModelBuilder) SetBoss(boss bool) *ModelBuilder {
	b.boss = boss
	return b
}

// SetResistances sets the elemental resistance map on the builder. Keys are
// element letters ("P", "I", "F", "S", "L"); value "1" means immune (per
// Model.IsImmuneToElement). Used by tests that drive elemental-immunity
// branches in ApplyStatusEffect.
func (b *ModelBuilder) SetResistances(r map[string]string) *ModelBuilder {
	b.resistances = r
	return b
}

// Build constructs an immutable Model from the builder state.
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
		skills:      skills,
		attacks:     attacks,
		hpRecovery:  b.hpRecovery,
		mpRecovery:  b.mpRecovery,
		boss:        b.boss,
		resistances: b.resistances,
	}
}
```

- [ ] **Step 2: Build the package**

```bash
( cd services/atlas-monsters/atlas.com/monsters && go build ./monster/information/... )
```

Expected: success.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/information/builder.go
git commit -m "test(atlas-monsters): expose SetBoss/SetResistances on information.ModelBuilder"
```

---

## Task 3: atlas-monsters — extend `testInformationLookup` hook to `ApplyStatusEffect`

`testInformationLookup` (processor.go:62-64) is a package-level hook used today only inside `UseBasicAttack` (line 715). To drive `ApplyStatusEffect`'s elemental-immunity and boss branches without standing up a real REST fake, extend the same indirection to the `information.GetById` call at line 1085.

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/processor.go`

- [ ] **Step 1: Update `ApplyStatusEffect` to consult the hook**

Replace the lookup at lines 1083-1099 with the hook-aware variant:

```go
	// Only check immunities for player-sourced effects
	if effect.SourceType() == SourceTypePlayerSkill {
		var info information.Model
		var infoErr error
		if testInformationLookup != nil {
			info, infoErr = testInformationLookup(m.MonsterId())
		} else {
			info, infoErr = information.GetById(p.l)(p.ctx)(m.MonsterId())
		}
		if infoErr == nil {
			// Elemental immunity check
			if blocked, element := isElementallyImmune(info, effect); blocked {
				p.l.Debugf("Monster [%d] is immune to element [%s]. Status rejected.", uniqueId, element)
				return errors.New("elemental immunity")
			}

			// Boss immunity check
			if info.Boss() && !isBossAllowedStatus(effect) {
				p.l.Debugf("Monster [%d] is a boss. Status rejected.", uniqueId)
				return errors.New("boss immunity")
			}
		}
	}
```

Also update the docstring on the package-level `var testInformationLookup` (lines 62-64) so it does not lie:

```go
// testInformationLookup is a test-only override for information.GetById. When
// nil (production), UseBasicAttack and ApplyStatusEffect call information.GetById
// normally.
var testInformationLookup func(monsterId uint32) (information.Model, error)
```

- [ ] **Step 2: Build the service**

```bash
( cd services/atlas-monsters/atlas.com/monsters && go build ./... )
```

Expected: success.

- [ ] **Step 3: Run the existing monster test suite to confirm no regression**

```bash
( cd services/atlas-monsters/atlas.com/monsters && go test ./monster/... )
```

Expected: all pass. (No prior test sets `testInformationLookup` before invoking `ApplyStatusEffect`, so production behavior is unchanged.)

- [ ] **Step 4: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/processor.go
git commit -m "refactor(atlas-monsters): route ApplyStatusEffect info lookup through testInformationLookup hook"
```

---

## Task 4: atlas-monsters — explicit DOOM short-circuit in `isElementallyImmune`

Today the gate at `processor.go:1117-1131` only switches on `POISON` / `FREEZE`, so a DOOM-only effect already falls through. The change pins intent next to the cases it overrides so a future maintainer adding `case "DOOM":` for symmetry — or shipping a multi-status combo carrying DOOM and POISON — cannot silently regress.

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/processor.go`

- [ ] **Step 1: Write the failing tests first (TDD)**

Append to `services/atlas-monsters/atlas.com/monsters/monster/processor_test.go`. Each test resets `testInformationLookup` via `t.Cleanup` so it cannot leak into the suite.

```go
// applyDoomEffectFromPlayer constructs a player-sourced DOOM status effect
// suitable for driving ApplyStatusEffect's immunity and boss branches.
func applyDoomEffectFromPlayer(durationMs int) StatusEffect {
	return NewStatusEffect(
		SourceTypePlayerSkill,
		1001, // sourceCharacterId
		2311005, // sourceSkillId (Priest Doom)
		30, // sourceSkillLevel
		map[string]int32{"DOOM": 1},
		time.Duration(durationMs)*time.Millisecond,
		0,
	)
}

// TestApplyStatusEffect_Doom_BypassesElementalImmunity verifies that DOOM is
// applied to a monster with full elemental resistance (the case Cosmic
// source treats as the skill's intended counter-niche). Pins the explicit
// short-circuit at the top of isElementallyImmune.
func TestApplyStatusEffect_Doom_BypassesElementalImmunity(t *testing.T) {
	r := GetMonsterRegistry()
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	r.Clear(ctx)

	f := testField()
	m := r.CreateMonster(ctx, tm, f, 9300018, 0, 0, 0, 0, 0, 1000, 50)

	// Resist every element including poison and ice, which would otherwise
	// fall through the existing POISON/FREEZE gates.
	resistances := map[string]string{"P": "1", "I": "1", "F": "1", "S": "1", "L": "1"}
	testInformationLookup = func(monsterId uint32) (information.Model, error) {
		return information.NewModelBuilder().
			SetBoss(false).
			SetResistances(resistances).
			Build(), nil
	}
	t.Cleanup(func() { testInformationLookup = nil })

	p, events := newRecordingProcessor(t, tm)
	p.ctx = ctx
	if err := p.ApplyStatusEffect(m.UniqueId(), applyDoomEffectFromPlayer(60000)); err != nil {
		t.Fatalf("ApplyStatusEffect(DOOM): %v", err)
	}

	got, err := r.GetMonster(tm, m.UniqueId())
	if err != nil {
		t.Fatalf("GetMonster: %v", err)
	}
	if !got.HasStatusEffect("DOOM") {
		t.Errorf("expected DOOM to be active on monster after apply")
	}
	statusApplied := 0
	for _, e := range *events {
		if e.Type == EventMonsterStatusEffectApplied {
			statusApplied++
		}
	}
	if statusApplied != 1 {
		t.Errorf("expected 1 STATUS_APPLIED event, got %d (%v)", statusApplied, *events)
	}
}

// TestApplyStatusEffect_Doom_RejectedOnBoss verifies the boss-immunity branch
// rejects DOOM. DOOM is not in isBossAllowedStatus's allow list so the
// rejection is automatic; this test pins it explicitly.
func TestApplyStatusEffect_Doom_RejectedOnBoss(t *testing.T) {
	r := GetMonsterRegistry()
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	r.Clear(ctx)

	f := testField()
	m := r.CreateMonster(ctx, tm, f, 8800000, 0, 0, 0, 0, 0, 1000, 50)

	testInformationLookup = func(monsterId uint32) (information.Model, error) {
		return information.NewModelBuilder().
			SetBoss(true).
			Build(), nil
	}
	t.Cleanup(func() { testInformationLookup = nil })

	p, events := newRecordingProcessor(t, tm)
	p.ctx = ctx
	err := p.ApplyStatusEffect(m.UniqueId(), applyDoomEffectFromPlayer(60000))
	if err == nil || err.Error() != "boss immunity" {
		t.Fatalf("ApplyStatusEffect(DOOM, boss): err=%v, want \"boss immunity\"", err)
	}
	got, gerr := r.GetMonster(tm, m.UniqueId())
	if gerr != nil {
		t.Fatalf("GetMonster: %v", gerr)
	}
	if got.HasStatusEffect("DOOM") {
		t.Errorf("expected DOOM not to be applied to boss")
	}
	for _, e := range *events {
		if e.Type == EventMonsterStatusEffectApplied {
			t.Errorf("did not expect STATUS_APPLIED event for boss reject; got %v", *events)
		}
	}
}

// TestApplyStatusEffect_Doom_ReapplyReplacesExisting pins the realized
// re-apply behavior: per builder.AddStatusEffect (builder.go:140-163), a
// non-VENOM status replaces the existing same-type entry rather than
// no-op-ing. The PRD's "no-op while already active" assumption is therefore
// inaccurate; we assert the actual behavior so any future change to that
// semantics surfaces here. (See design.md §6 "Re-apply semantics for DOOM.")
func TestApplyStatusEffect_Doom_ReapplyReplacesExisting(t *testing.T) {
	r := GetMonsterRegistry()
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	r.Clear(ctx)

	f := testField()
	m := r.CreateMonster(ctx, tm, f, 9300018, 0, 0, 0, 0, 0, 1000, 50)

	testInformationLookup = func(monsterId uint32) (information.Model, error) {
		return information.NewModelBuilder().Build()
	}
	t.Cleanup(func() { testInformationLookup = nil })

	p, events := newRecordingProcessor(t, tm)
	p.ctx = ctx

	first := applyDoomEffectFromPlayer(60000)
	if err := p.ApplyStatusEffect(m.UniqueId(), first); err != nil {
		t.Fatalf("first ApplyStatusEffect(DOOM): %v", err)
	}
	second := applyDoomEffectFromPlayer(60000)
	if err := p.ApplyStatusEffect(m.UniqueId(), second); err != nil {
		t.Fatalf("second ApplyStatusEffect(DOOM): %v", err)
	}

	got, gerr := r.GetMonster(tm, m.UniqueId())
	if gerr != nil {
		t.Fatalf("GetMonster: %v", gerr)
	}

	doomEffects := 0
	var stored StatusEffect
	for _, se := range got.StatusEffects() {
		if se.HasStatus("DOOM") {
			doomEffects++
			stored = se
		}
	}
	if doomEffects != 1 {
		t.Errorf("expected exactly 1 DOOM status effect after refresh, got %d", doomEffects)
	}
	if stored.EffectId() != second.EffectId() {
		t.Errorf("expected stored DOOM effect to be the second apply; got effectId=%s want=%s", stored.EffectId(), second.EffectId())
	}

	statusApplied := 0
	for _, e := range *events {
		if e.Type == EventMonsterStatusEffectApplied {
			statusApplied++
		}
	}
	if statusApplied != 2 {
		t.Errorf("expected 2 STATUS_APPLIED events (refresh emits a second), got %d (%v)", statusApplied, *events)
	}
}
```

If `EventMonsterStatusEffectApplied` is not the exact constant name in this package, run `grep -n 'EventMonsterStatus' services/atlas-monsters/atlas.com/monsters/monster/*.go` and use the constant the codebase actually emits for the apply event. Update all three tests in lockstep.

If `newTestTenant` and `testField` are not defined in `processor_test.go`, search the file for the existing helpers (`grep -n 'func newTestTenant\|func testField' services/atlas-monsters/atlas.com/monsters/monster/processor_test.go`) and use the realized helper names. The patterns above mirror existing tests in the same file.

- [ ] **Step 2: Run the new tests; verify they fail with the current `isElementallyImmune`**

```bash
( cd services/atlas-monsters/atlas.com/monsters && go test ./monster -run 'TestApplyStatusEffect_Doom_' -v )
```

Expected: `TestApplyStatusEffect_Doom_BypassesElementalImmunity` PASSES (current code already lets DOOM through), `TestApplyStatusEffect_Doom_RejectedOnBoss` PASSES (DOOM is not in `isBossAllowedStatus`'s allow list), `TestApplyStatusEffect_Doom_ReapplyReplacesExisting` PASSES (existing AddStatusEffect refresh semantics).

In other words: all three tests pass before the production change. That is intentional — these are pinning tests. The next step still adds the explicit DOOM short-circuit, then the tests must keep passing.

- [ ] **Step 3: Add the explicit DOOM short-circuit to `isElementallyImmune`**

Replace `isElementallyImmune` (processor.go:1116-1131) with:

```go
// isElementallyImmune checks if a monster's resistances block the given status effect.
// DOOM (Priest, 2311005) intentionally bypasses elemental immunity: the
// polymorph-to-snail effect overrides resistance — a fire-immune mob still
// becomes a snail. Source parity with Cosmic (server/StatEffect.java:1531).
func isElementallyImmune(info information.Model, effect StatusEffect) (bool, string) {
	if _, ok := effect.Statuses()[monster2.StatusDoom]; ok {
		return false, ""
	}
	for statusType := range effect.Statuses() {
		switch statusType {
		case "POISON":
			if info.IsImmuneToElement("P") {
				return true, "poison"
			}
		case "FREEZE":
			if info.IsImmuneToElement("I") {
				return true, "ice"
			}
		}
	}
	return false, ""
}
```

If `monster2` is not the import alias used at the top of `processor.go`, check the import block (around lines 14-22) and use whatever alias is bound to `github.com/Chronicle20/atlas/libs/atlas-constants/monster`. The file already aliases it as `monster2` per `processor.go:16`, so the snippet above should compile as written.

- [ ] **Step 4: Re-run the new tests to confirm the short-circuit didn't regress them**

```bash
( cd services/atlas-monsters/atlas.com/monsters && go test ./monster -run 'TestApplyStatusEffect_Doom_' -v )
```

Expected: all three PASS.

- [ ] **Step 5: Run the full atlas-monsters suite**

```bash
( cd services/atlas-monsters/atlas.com/monsters && go build ./... && go test ./... )
```

Expected: all pass.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/processor.go services/atlas-monsters/atlas.com/monsters/monster/processor_test.go
git commit -m "feat(atlas-monsters): explicit DOOM short-circuit in isElementallyImmune; pin DOOM apply path"
```

---

## Task 5: atlas-channel — extract `processDamageInfoEntry` (no behavior change)

The channel-side handler interleaves character lookup (once per packet) with per-target work (N times per packet). Extract the per-`DamageInfo` body of `processAttack` (currently `character_attack_common.go:151-216`) into a top-level helper with explicit dependencies. This is a pure refactor: every existing test must still pass.

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go`

- [ ] **Step 1: Add the helper above `processAttack`**

Insert the following just before `func processAttack(...)` (around line 76 of the current file). The signature uses function-typed parameters so the helper can be exercised with closures from tests.

```go
// damageInfoEntryDeps groups the per-attack closures and lookups that
// processDamageInfoEntry needs. Wrapping them keeps the helper signature
// readable and lets tests construct fakes with a single struct.
type damageInfoEntryDeps struct {
	getReflect        func(t tenant.Model, monsterId uint32, kind string) (monster.ReflectInfo, bool)
	getMonster        func(monsterId uint32) (monster.Model, error)
	applyDamage       func(f field.Model, monsterId, characterId uint32, damages []uint32, attackType byte) error
	emitReflectDamage func(f field.Model, uniqueId, templateId, characterId uint32, reflectDamage uint32, reflectType string) error
	applyStatus       func(f field.Model, monsterId, characterId, skillId, skillLevel uint32, statuses map[string]int32, duration uint32) error
	loadVenomStats    func() effective_stats.RestModel
}

// processDamageInfoEntry handles one DamageInfo from a magic/melee/ranged
// attack packet: damage application or reflect emission, then optional
// monster status apply. It returns the reflected flag so the caller (kept
// for now to preserve broadcast semantics) does not need to introspect the
// helper's internal state.
//
// All side-effecting calls go through deps so tests can drive each branch
// without constructing a real monster.Processor or session.
func processDamageInfoEntry(
	l logrus.FieldLogger,
	di packetmodel.DamageInfo,
	ai packetmodel.AttackInfo,
	se effect.Model,
	skillLevel uint32,
	casterId uint32,
	casterX, casterY int16,
	f field.Model,
	t tenant.Model,
	attackKind string,
	deps damageInfoEntryDeps,
) {
	damages := di.Damages()

	if len(damages) == 0 {
		if len(se.MonsterStatus()) == 0 {
			return
		}
		ms := make(map[string]int32)
		for k, v := range se.MonsterStatus() {
			ms[k] = int32(v)
		}
		if _, isVenom := ms["VENOM"]; isVenom {
			stats := deps.loadVenomStats()
			coef := 0.1 + rand.Float64()*0.1
			ms["VENOM"] = snapshotVenomDamagePerTick(int(stats.Luck), int(stats.MagicAttack), coef)
		}
		_ = deps.applyStatus(f, di.MonsterId(), casterId, uint32(ai.SkillId()), skillLevel, ms, uint32(se.Duration()))
		return
	}

	reflected := false
	if attackKind != "" {
		if info, ok := deps.getReflect(t, di.MonsterId(), attackKind); ok {
			mon, mErr := deps.getMonster(di.MonsterId())
			if mErr == nil {
				entry := make([]int32, 0, len(damages))
				for _, d := range damages {
					entry = append(entry, int32(d))
				}
				r, within := computeReflect(entry, info, casterX, casterY, mon.X(), mon.Y())
				if within {
					l.Debugf("reflect: char [%d] hit monster [%d] for %d reflected damage.", casterId, di.MonsterId(), r)
					if eErr := deps.emitReflectDamage(f, di.MonsterId(), mon.MonsterId(), casterId, uint32(r), info.Kind); eErr != nil {
						l.WithError(eErr).Errorf("Unable to emit DAMAGE_REFLECTED for monster [%d] / character [%d].", di.MonsterId(), casterId)
					}
					reflected = true
				}
			}
		}
	}

	if reflected {
		// On reflect: monster takes no damage AND no monster status is applied
		// for this entry (FREEZE/STUN/etc. would let the player slip through
		// the reflect's intent).
		return
	}

	if err := deps.applyDamage(f, di.MonsterId(), casterId, damages, byte(ai.AttackType())); err != nil {
		l.WithError(err).Errorf("Unable to apply damage to monster [%d] from character [%d].", di.MonsterId(), casterId)
	}

	// Apply monster status effects from skill (e.g., freeze, poison, stun).
	if len(se.MonsterStatus()) > 0 {
		ms := make(map[string]int32)
		for k, v := range se.MonsterStatus() {
			ms[k] = int32(v)
		}
		if _, isVenom := ms["VENOM"]; isVenom {
			stats := deps.loadVenomStats()
			coef := 0.1 + rand.Float64()*0.1
			ms["VENOM"] = snapshotVenomDamagePerTick(int(stats.Luck), int(stats.MagicAttack), coef)
		}
		_ = deps.applyStatus(f, di.MonsterId(), casterId, uint32(ai.SkillId()), skillLevel, ms, uint32(se.Duration()))
	}
}
```

- [ ] **Step 2: Replace the inline loop body in `processAttack` with a call to the helper**

In `processAttack`, replace the entire `for _, di := range ai.DamageInfo() { ... }` block (lines 151-216) with:

```go
deps := damageInfoEntryDeps{
	getReflect:        mirror.GetReflect,
	getMonster:        mp.GetById,
	applyDamage:       mp.Damage,
	emitReflectDamage: mp.EmitDamageReflected,
	applyStatus:       mp.ApplyStatus,
	loadVenomStats:    loadVenomStats,
}
for _, di := range ai.DamageInfo() {
	processDamageInfoEntry(
		l, di, ai, se, uint32(sk.Level()),
		s.CharacterId(), c.X(), c.Y(),
		s.Field(), t, attackKind,
		deps,
	)
}
```

- [ ] **Step 3: Build atlas-channel**

```bash
( cd services/atlas-channel/atlas.com/channel && go build ./... )
```

Expected: success. If a method signature mismatch surfaces (e.g. `mp.GetById` returns `(monster.Model, error)` but the helper expects something different), fix the helper's `damageInfoEntryDeps` types to match the actual `monster.Processor` method signatures rather than coercing the call sites. Report any deviation in the commit message.

- [ ] **Step 4: Run the existing atlas-channel test suite to confirm no regression**

```bash
( cd services/atlas-channel/atlas.com/channel && go test ./... )
```

Expected: all pass — particularly the existing `TestComputeReflect_*`, `TestReflectFlow_*`, and `TestSnapshotVenomDamagePerTick_*` cases, which still target their respective pure helpers (unaffected by this extraction).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go
git commit -m "refactor(atlas-channel): extract processDamageInfoEntry helper from processAttack"
```

---

## Task 6: atlas-channel — Doom-gated magic-reflect probe in the empty-damage branch

Per design §2.2 / §3.4: the existing reflect path only runs when `len(damages) > 0`. Doom carries empty damages, so today an empty-damage Doom apply would land on a magic-reflect mob unchallenged. Add a narrow probe inside `processDamageInfoEntry`'s empty-damage branch that runs only when the inbound status set contains `"DOOM"`. Doom does no damage, so on reflect we simply skip the apply (no reflect damage to emit).

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go`

- [ ] **Step 1: Insert the Doom-gated probe inside the empty-damage branch**

In `processDamageInfoEntry`, replace the empty-damage branch (the `if len(damages) == 0 { ... }` block added in Task 5) with:

```go
	if len(damages) == 0 {
		if len(se.MonsterStatus()) == 0 {
			return
		}
		ms := make(map[string]int32)
		for k, v := range se.MonsterStatus() {
			ms[k] = int32(v)
		}
		if _, isVenom := ms["VENOM"]; isVenom {
			stats := deps.loadVenomStats()
			coef := 0.1 + rand.Float64()*0.1
			ms["VENOM"] = snapshotVenomDamagePerTick(int(stats.Luck), int(stats.MagicAttack), coef)
		}

		// Doom: respect magic-reflect. Doom does no damage, so on reflect we
		// simply skip the apply (nothing to bounce back). Gated on DOOM so
		// no other empty-damage status flow changes behavior.
		if _, isDoom := ms["DOOM"]; isDoom && attackKind != "" {
			if _, ok := deps.getReflect(t, di.MonsterId(), attackKind); ok {
				l.Debugf("Doom: monster [%d] has %s reflect; status apply skipped.", di.MonsterId(), attackKind)
				return
			}
		}

		_ = deps.applyStatus(f, di.MonsterId(), casterId, uint32(ai.SkillId()), skillLevel, ms, uint32(se.Duration()))
		return
	}
```

- [ ] **Step 2: Build atlas-channel**

```bash
( cd services/atlas-channel/atlas.com/channel && go build ./... )
```

Expected: success.

- [ ] **Step 3: Run the existing atlas-channel test suite**

```bash
( cd services/atlas-channel/atlas.com/channel && go test ./... )
```

Expected: all pass. (No existing test exercises the empty-damage Doom-on-reflect path; the new tests in Task 7 will.)

- [ ] **Step 4: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go
git commit -m "feat(atlas-channel): Doom-gated magic-reflect probe on empty-damage status apply"
```

---

## Task 7: atlas-channel — helper tests for Doom cast / reflect / spread

Pin the cast-to-ApplyStatus, reflect-blocks-Doom, and multi-target-spread paths against the extracted helper. Tests inject closures-as-fakes; no real monster processor or session is constructed.

**Files:**
- Test: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common_test.go`

- [ ] **Step 1: Append a fakes harness and the new tests**

Append to `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common_test.go`. Each test uses a fresh `damageInfoEntryDeps` so state cannot leak between cases.

```go
// applyStatusCall captures one invocation of the applyStatus closure so tests
// can assert on (monsterId, statuses, duration) without inspecting Kafka.
type applyStatusCall struct {
	monsterId   uint32
	characterId uint32
	skillId     uint32
	skillLevel  uint32
	statuses    map[string]int32
	duration    uint32
}

// damageEntryFakes is a minimal in-memory recorder used by Doom helper tests.
// Only Doom-relevant interactions are tracked.
type damageEntryFakes struct {
	applyStatusCalls       []applyStatusCall
	applyDamageCalls       int
	emitReflectDamageCalls int
	reflects               map[uint32]monster.ReflectInfo
	monsters               map[uint32]monster.Model
}

func (f *damageEntryFakes) deps() damageInfoEntryDeps {
	return damageInfoEntryDeps{
		getReflect: func(_ tenant.Model, monsterId uint32, _ string) (monster.ReflectInfo, bool) {
			ri, ok := f.reflects[monsterId]
			return ri, ok
		},
		getMonster: func(monsterId uint32) (monster.Model, error) {
			m, ok := f.monsters[monsterId]
			if !ok {
				return monster.Model{}, errors.New("not found")
			}
			return m, nil
		},
		applyDamage: func(_ field.Model, _ uint32, _ uint32, _ []uint32, _ byte) error {
			f.applyDamageCalls++
			return nil
		},
		emitReflectDamage: func(_ field.Model, _ uint32, _ uint32, _ uint32, _ uint32, _ string) error {
			f.emitReflectDamageCalls++
			return nil
		},
		applyStatus: func(_ field.Model, monsterId, characterId, skillId, skillLevel uint32, statuses map[string]int32, duration uint32) error {
			f.applyStatusCalls = append(f.applyStatusCalls, applyStatusCall{
				monsterId:   monsterId,
				characterId: characterId,
				skillId:     skillId,
				skillLevel:  skillLevel,
				statuses:    statuses,
				duration:    duration,
			})
			return nil
		},
		loadVenomStats: func() effective_stats.RestModel { return effective_stats.RestModel{} },
	}
}

func newDoomEffect() effect.Model {
	return effect.NewModelBuilder().
		SetDuration(20000).
		SetMonsterStatus(map[string]uint32{"DOOM": 1}).
		Build()
}

func newDoomAttackInfo(monsterIds ...uint32) packetmodel.AttackInfo {
	dis := make([]packetmodel.DamageInfo, 0, len(monsterIds))
	for _, mid := range monsterIds {
		dis = append(dis, packetmodel.NewDamageInfoBuilder().
			SetMonsterId(mid).
			SetDamages(nil). // Doom carries empty damages
			Build())
	}
	return packetmodel.NewAttackInfoBuilder().
		SetSkillId(2311005).
		SetAttackType(packetmodel.AttackTypeMagic).
		SetDamageInfo(dis).
		Build()
}

// TestProcessDamageInfoEntry_Doom_EmptyDamagesAppliesStatus verifies the
// happy path: an empty-damage DOOM-bearing attack lands on a target with no
// reflect, producing exactly one applyStatus call with the DOOM map and the
// effect's duration. Pins the empty-damage branch the Cosmic source treats
// as Doom's normal path.
func TestProcessDamageInfoEntry_Doom_EmptyDamagesAppliesStatus(t *testing.T) {
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	l, _ := test.NewNullLogger()

	ai := newDoomAttackInfo(1)
	di := ai.DamageInfo()[0]
	se := newDoomEffect()
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()

	fk := &damageEntryFakes{}
	processDamageInfoEntry(l, di, ai, se, 30, /*casterId*/ 1001, 0, 0, f, tm, "MAGICAL", fk.deps())

	if len(fk.applyStatusCalls) != 1 {
		t.Fatalf("applyStatus calls = %d, want 1 (%v)", len(fk.applyStatusCalls), fk.applyStatusCalls)
	}
	got := fk.applyStatusCalls[0]
	if got.monsterId != 1 || got.skillId != 2311005 || got.skillLevel != 30 {
		t.Errorf("applyStatus args = %+v, want monsterId=1 skillId=2311005 skillLevel=30", got)
	}
	if got.statuses["DOOM"] != 1 {
		t.Errorf("statuses[DOOM] = %d, want 1", got.statuses["DOOM"])
	}
	if got.duration != 20000 {
		t.Errorf("duration = %d, want 20000", got.duration)
	}
	if fk.applyDamageCalls != 0 {
		t.Errorf("applyDamage called %d times, want 0", fk.applyDamageCalls)
	}
	if fk.emitReflectDamageCalls != 0 {
		t.Errorf("emitReflectDamage called %d times, want 0", fk.emitReflectDamageCalls)
	}
}

// TestProcessDamageInfoEntry_Doom_BlockedByReflect verifies the new
// Doom-gated probe: when the target has a magic-reflect window and the
// inbound status set is DOOM-bearing, the apply is skipped (no reflect
// damage is emitted because Doom does no damage).
func TestProcessDamageInfoEntry_Doom_BlockedByReflect(t *testing.T) {
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	l, _ := test.NewNullLogger()

	ai := newDoomAttackInfo(1)
	di := ai.DamageInfo()[0]
	se := newDoomEffect()
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()

	fk := &damageEntryFakes{
		reflects: map[uint32]monster.ReflectInfo{
			1: {Kind: monster2.ReflectKindMagical, Percent: 30, LtX: -100, LtY: -100, RbX: 100, RbY: 100, MaxDamage: 9999},
		},
	}
	processDamageInfoEntry(l, di, ai, se, 30, 1001, 0, 0, f, tm, monster2.ReflectKindMagical, fk.deps())

	if len(fk.applyStatusCalls) != 0 {
		t.Errorf("applyStatus calls = %d, want 0 (Doom blocked by reflect)", len(fk.applyStatusCalls))
	}
	if fk.emitReflectDamageCalls != 0 {
		t.Errorf("emitReflectDamage calls = %d, want 0 (Doom does no damage to reflect)", fk.emitReflectDamageCalls)
	}
	if fk.applyDamageCalls != 0 {
		t.Errorf("applyDamage calls = %d, want 0 (no damage path)", fk.applyDamageCalls)
	}
}

// TestProcessDamageInfoEntry_Doom_MultiTargetSpread verifies the spread case:
// three Doom targets, the middle one carries a magic-reflect window, the
// other two are clean. Helper invoked once per DamageInfo. Result: exactly
// two applyStatus calls (monsters 1 and 3); none for monster 2.
func TestProcessDamageInfoEntry_Doom_MultiTargetSpread(t *testing.T) {
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	l, _ := test.NewNullLogger()

	ai := newDoomAttackInfo(1, 2, 3)
	se := newDoomEffect()
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()

	fk := &damageEntryFakes{
		reflects: map[uint32]monster.ReflectInfo{
			2: {Kind: monster2.ReflectKindMagical, Percent: 30, LtX: -100, LtY: -100, RbX: 100, RbY: 100, MaxDamage: 9999},
		},
	}
	for _, di := range ai.DamageInfo() {
		processDamageInfoEntry(l, di, ai, se, 30, 1001, 0, 0, f, tm, monster2.ReflectKindMagical, fk.deps())
	}

	if len(fk.applyStatusCalls) != 2 {
		t.Fatalf("applyStatus calls = %d, want 2 (%v)", len(fk.applyStatusCalls), fk.applyStatusCalls)
	}
	gotIds := []uint32{fk.applyStatusCalls[0].monsterId, fk.applyStatusCalls[1].monsterId}
	if !(gotIds[0] == 1 && gotIds[1] == 3) {
		t.Errorf("applyStatus monster ids = %v, want [1 3] (monster 2 reflect-blocked)", gotIds)
	}
	if fk.emitReflectDamageCalls != 0 {
		t.Errorf("emitReflectDamage calls = %d, want 0", fk.emitReflectDamageCalls)
	}
	if fk.applyDamageCalls != 0 {
		t.Errorf("applyDamage calls = %d, want 0", fk.applyDamageCalls)
	}
}

// TestProcessDamageInfoEntry_NonDoom_EmptyDamagesIgnoresReflectProbe pins
// that the new probe is Doom-gated. A hypothetical empty-damage status that
// is not DOOM should still apply through a magic-reflect window. (No such
// status exists in atlas-data today; the test guards against future drift.)
func TestProcessDamageInfoEntry_NonDoom_EmptyDamagesIgnoresReflectProbe(t *testing.T) {
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	l, _ := test.NewNullLogger()

	se := effect.NewModelBuilder().
		SetDuration(5000).
		SetMonsterStatus(map[string]uint32{"FREEZE": 1}).
		Build()
	ai := packetmodel.NewAttackInfoBuilder().
		SetSkillId(0).
		SetAttackType(packetmodel.AttackTypeMagic).
		SetDamageInfo([]packetmodel.DamageInfo{
			packetmodel.NewDamageInfoBuilder().SetMonsterId(7).SetDamages(nil).Build(),
		}).
		Build()
	di := ai.DamageInfo()[0]
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()

	fk := &damageEntryFakes{
		reflects: map[uint32]monster.ReflectInfo{
			7: {Kind: monster2.ReflectKindMagical, Percent: 30, LtX: -100, LtY: -100, RbX: 100, RbY: 100, MaxDamage: 9999},
		},
	}
	processDamageInfoEntry(l, di, ai, se, 1, 1001, 0, 0, f, tm, monster2.ReflectKindMagical, fk.deps())

	if len(fk.applyStatusCalls) != 1 {
		t.Errorf("non-Doom empty-damage status should apply through reflect; applyStatus calls = %d, want 1", len(fk.applyStatusCalls))
	}
}
```

The test file already imports `monster`, `packetmodel`, `tenant`, and `uuid` (see existing tests). New imports needed: `errors`, `field` / `world` / `channel` / `_map` constants from `libs/atlas-constants`, `effect` (`atlas-channel/data/skill/effect`), `effective_stats` (`atlas-channel/effective_stats`), and `test` from `github.com/sirupsen/logrus/hooks/test`. Add to the import block at the top of the file. Run `goimports -w` afterward if available; otherwise edit the import block by hand.

If `packetmodel.NewAttackInfoBuilder` / `NewDamageInfoBuilder` / `SetDamages` / `SetMonsterId` / `SetSkillId` / `SetAttackType` / `SetDamageInfo` do not exist with those exact names, find the realized constructor pattern with:

```bash
grep -n "func NewAttackInfo\|func NewDamageInfo\|func .* AttackInfo\b\|func .* DamageInfo\b" libs/atlas-packet/model/*.go
```

and adjust the helpers in lockstep.

If `effect.NewModelBuilder` / `SetMonsterStatus` are not the method names in `services/atlas-channel/atlas.com/channel/data/skill/effect`, mirror the constructor pattern that exists there. The Setter for the monster status map is required because otherwise the empty-damage branch short-circuits on `len(se.MonsterStatus()) == 0`.

- [ ] **Step 2: Run the new tests**

```bash
( cd services/atlas-channel/atlas.com/channel && go test ./socket/handler -run 'TestProcessDamageInfoEntry_' -v )
```

Expected: all four PASS. If any fail because a builder/constructor name does not exist, adjust the helpers to match the realized API and re-run.

- [ ] **Step 3: Run the full atlas-channel suite**

```bash
( cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./... )
```

Expected: all pass.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common_test.go
git commit -m "test(atlas-channel): pin Doom cast/reflect/spread paths via processDamageInfoEntry"
```

---

## Task 8: atlas-channel — Doom-specific Debugf in `monster.Processor.ApplyStatus`

Add a single grep-friendly log line inside `Processor.ApplyStatus` so production diagnoses can pivot on `"Doom:"`. The generic `Applying status to monster [...]` Debugf already on that line stays.

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/monster/processor.go`

- [ ] **Step 1: Insert the conditional Debugf**

Replace `func (p *Processor) ApplyStatus(...)` (currently `processor.go:70-73`) with:

```go
func (p *Processor) ApplyStatus(f field.Model, monsterId uint32, characterId uint32, skillId uint32, skillLevel uint32, statuses map[string]int32, duration uint32) error {
	p.l.Debugf("Applying status to monster [%d]. Character [%d]. Skill [%d].", monsterId, characterId, skillId)
	if _, isDoom := statuses["DOOM"]; isDoom {
		p.l.Debugf("Doom: caster=[%d] monster=[%d] skill=[%d] level=[%d] duration=[%d]ms.", characterId, monsterId, skillId, skillLevel, duration)
	}
	return producer.ProviderImpl(p.l)(p.ctx)(monster2.EnvCommandTopic)(ApplyStatusCommandProvider(f, monsterId, characterId, skillId, skillLevel, statuses, duration))
}
```

- [ ] **Step 2: Build and test atlas-channel**

```bash
( cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./... )
```

Expected: success. (No new test for the log line itself — it is observability, not behavior. Manual verification in Task 9 covers it.)

- [ ] **Step 3: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/monster/processor.go
git commit -m "feat(atlas-channel): emit Doom-targeted Debugf alongside generic ApplyStatus log"
```

---

## Task 9: Cross-service build, full test, and manual verification handoff

Final gate. Confirm every affected service still builds and tests cleanly. Then surface a manual-verification checklist for the user — the wire-level snail render and expiry sprite restore are client-side and cannot be unit-tested.

**Files:** None modified.

- [ ] **Step 1: Build and test all three affected services**

Run sequentially — they share types only via go.work-level libs that this task does not touch.

```bash
( cd services/atlas-monsters/atlas.com/monsters && go build ./... && go test ./... )
( cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./... )
( cd services/atlas-data/atlas.com/data && go build ./... && go test ./... )
```

Expected: all green.

- [ ] **Step 2: Verify the in-repo grep for the Doom log line is unique**

```bash
grep -rn '"Doom: caster=\[' services/
```

Expected: exactly one hit, in `services/atlas-channel/atlas.com/channel/monster/processor.go`.

- [ ] **Step 3: Surface the manual end-to-end checklist to the user**

The PRD §10 acceptance criteria require a live cast against a regular mob and a boss. These cannot be unit-tested. Tell the user:

> Code changes complete. Manual verification needed against a running channel + monsters + data stack:
>
> 1. Spawn a regular mob with elemental resistance (e.g., a fire-immune mob if available; any non-boss mob is sufficient for the basic snail render).
> 2. Cast Priest Doom (skill 2311005) on it.
> 3. Confirm the v83 client renders it as a snail for the duration and restores the original sprite at expiry.
> 4. In atlas-channel logs, confirm the line `Doom: caster=[..] monster=[..] skill=[2311005] level=[..] duration=[..]ms.` appears.
> 5. Cast Doom on a boss; confirm in atlas-monsters logs the line `Monster [..] is a boss. Status rejected.` appears, and the client does not render the snail.
> 6. (Optional but valuable) Cast Doom against a mob standing on a magic-reflect window; confirm in atlas-channel logs the line `Doom: monster [..] has MAGICAL reflect; status apply skipped.` appears, and the mob is unaffected.

If observability MCPs are available, follow `reference_observability.md` to pull the relevant log lines from Loki rather than tailing a local pod.

- [ ] **Step 4: No commit needed for this step**

This step is a verification gate. Once the user confirms the manual checks, the branch is ready for `superpowers:requesting-code-review` and PR.

---

## Self-review (run by writer; results inline)

**Spec coverage** (PRD §4.1–§4.7, §7, §10):

- §4.1 cast intake — covered by Task 0 verification + Task 5/6 helper extraction (no production change to intake; we re-route the per-DamageInfo body without altering routing or cost). ✓
- §4.2 target resolution + reflect — covered by Task 6 (Doom-gated reflect probe). ✓
- §4.3 status apply — covered by Task 4 (DOOM short-circuit) + Task 1 (effect mapping pin). ✓
- §4.4 status broadcast — no code change required (existing wire mask handles DOOM bit); Task 9 manual verification confirms. ✓
- §4.5 status expiry — no code change required; Task 9 manual verification confirms. ✓
- §4.6 cast logging — covered by Task 8. ✓
- §4.7 tests — atlas-data (Task 1), atlas-monsters (Task 4), atlas-channel (Task 7). ✓
- §7 service impact rows — atlas-monsters (Tasks 2-4), atlas-channel (Tasks 5-8), atlas-data (Task 1), libs/* unchanged. ✓
- §10 acceptance criteria — covered across the eight implementation tasks plus Task 9 manual checklist. ✓

**Placeholder scan:** searched the plan for `TBD`, `TODO`, `implement later`, `Add appropriate error handling`, `similar to Task`, `implement minimal`, `etc.`. All occurrences are inside cited file paths or test bodies that show the actual code, not stand-ins.

**Type consistency:**

- `damageInfoEntryDeps` defined in Task 5 with field names `getReflect`, `getMonster`, `applyDamage`, `emitReflectDamage`, `applyStatus`, `loadVenomStats`. Same names used in Task 6 (probe) and Task 7 (test fakes builder). ✓
- `applyStatusCall` fields used in Task 7 assertions match the closure-recorded args one-for-one. ✓
- `processDamageInfoEntry` signature in Task 5 declared once; Task 6 modifies the function body only; Task 7 calls with the same parameter order. ✓
- `applyDoomEffectFromPlayer(durationMs int)` returns `StatusEffect` in Task 4; both call sites pass `60000`. ✓
- `isElementallyImmune` short-circuit references `monster2.StatusDoom`; the existing import alias in `services/atlas-monsters/atlas.com/monsters/monster/processor.go:16` is `monster2`. ✓

**Notable design deviations called out inline:**

- The PRD's "DOOM no-op while already active" assumption is rewritten in Task 4 step 1 as "DOOM re-apply replaces existing entry" because that is the realized `AddStatusEffect` semantics in `builder.go:140-163`. The design itself flagged this as a risk (design.md §6); the plan asserts the realized behavior rather than silently changing code to match a faulty PRD assumption. The PRD/design should be amended in the same commit — flagged in Task 4 step 1's commentary (and in `context.md` "Realized behavior").
- The PRD's `damageInfoEntryArgs` struct option is taken (Task 5 uses `damageInfoEntryDeps`) over a 12-positional-parameter signature for readability. Both forms were sanctioned by design §3.3.

If the implementer hits a realized-API mismatch (e.g., a builder method renamed, an event constant spelled differently), the surrounding step text directs them to grep for the actual name and adjust in lockstep. No silent renames.
