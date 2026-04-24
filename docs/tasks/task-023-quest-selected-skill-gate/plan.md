# Quest `selectedSkillID` Start Gate — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Gate quest starts by `selectedSkillID` so skill-tutorial quests on shared job trees (e.g. 2418 Double Stab / 2420 Lucky Seven) are no longer both startable by the same Thief character.

**Architecture:** Parse `selectedSkillID` from `QuestInfo.img` into the atlas-data `RestModel`. Mirror that field on the atlas-quest `RestModel`. In atlas-quest's `ValidateStartRequirements`, extract condition-building into a pure helper and emit the already-supported `skillLevel` condition (handled end-to-end by atlas-query-aggregator) with `>= 1` whenever `SelectedSkillId` is non-zero. No changes in atlas-query-aggregator or atlas-skills.

**Tech Stack:** Go 1.21+, atlas-data (`atlas-data` module), atlas-quest (`atlas-quest` module), `logrus`, standard Go tests with `testing` package and `logrus/hooks/test` null loggers.

---

## File Structure

**Modified files:**

- `services/atlas-data/atlas.com/data/quest/rest.go` — add `SelectedSkillId uint32` to `RestModel`.
- `services/atlas-data/atlas.com/data/quest/reader.go` — parse `selectedSkillID` in `ReadQuestInfo`.
- `services/atlas-data/atlas.com/data/quest/reader_test.go` — extend `TestReadQuestInfo` to cover the new attribute.
- `services/atlas-quest/atlas.com/quest/data/quest/rest.go` — mirror `SelectedSkillId uint32` on the quest-service copy of `RestModel`.
- `services/atlas-quest/atlas.com/quest/data/validation/model.go` — add `SkillCondition = "skillLevel"`.
- `services/atlas-quest/atlas.com/quest/data/validation/processor.go` — extract `buildStartConditions` helper and emit the skill condition when `SelectedSkillId > 0`.

**Created files:**

- `services/atlas-quest/atlas.com/quest/data/validation/processor_test.go` — new unit tests for `buildStartConditions`.

No changes to atlas-query-aggregator, atlas-skills, Kafka topics, database schema, or REST contracts.

---

## Task 1 — atlas-data: parse `selectedSkillID` from QuestInfo

**Files:**

- Modify: `services/atlas-data/atlas.com/data/quest/rest.go` (struct `RestModel`, insert after `SelectedMob` field)
- Modify: `services/atlas-data/atlas.com/data/quest/reader.go` (function `ReadQuestInfo`, extend the `RestModel{...}` literal)
- Test: `services/atlas-data/atlas.com/data/quest/reader_test.go` (extend `testQuestInfoXML` constant + `TestReadQuestInfo` function)

- [ ] **Step 1.1: Extend the QuestInfo test XML and add an assertion**

Add the `selectedSkillID` attribute to the existing quest `2000` fixture (so nothing else has to be introduced) and a new assertion in `TestReadQuestInfo` that the field is parsed. Also add an absence check using existing quest `10000`, which has no `selectedSkillID` → expect `0`.

Edit `services/atlas-data/atlas.com/data/quest/reader_test.go`:

Replace the `2000` fixture block inside `testQuestInfoXML` (currently ends with `<int name="timeLimit" value="3600"/>`) with:

```go
  <imgdir name="2000">
    <string name="name" value="Mai's First Training"/>
    <string name="parent" value="Maple Island"/>
    <int name="area" value="10"/>
    <int name="order" value="1"/>
    <int name="autoStart" value="0"/>
    <int name="autoPreComplete" value="0"/>
    <int name="autoComplete" value="1"/>
    <int name="timeLimit" value="3600"/>
    <int name="selectedSkillID" value="4001334"/>
  </imgdir>
```

Inside `TestReadQuestInfo`, after the `q2000.TimeLimit` check (which ends at `t.Fatalf("expected timeLimit 3600, got %d", q2000.TimeLimit)`), append:

```go
	if q2000.SelectedSkillId != 4001334 {
		t.Fatalf("expected selectedSkillId 4001334, got %d", q2000.SelectedSkillId)
	}

	// Quest 10000 has no selectedSkillID attribute - field must default to 0
	if q10000.SelectedSkillId != 0 {
		t.Fatalf("expected q10000.SelectedSkillId default 0, got %d", q10000.SelectedSkillId)
	}
```

Note: `q10000` is already fetched earlier in the function at the line `q10000, exists := quests[10000]`. No extra lookup needed.

- [ ] **Step 1.2: Run the test, expect failure**

```bash
cd services/atlas-data/atlas.com/data
go test ./quest/ -run TestReadQuestInfo -v
```

Expected output: `./reader_test.go: q2000.SelectedSkillId undefined (type RestModel has no field or method SelectedSkillId)` — compilation failure because the field does not exist yet.

- [ ] **Step 1.3: Add `SelectedSkillId` to `RestModel`**

Edit `services/atlas-data/atlas.com/data/quest/rest.go`. In the `RestModel` struct (at ~line 10), insert the field after `SelectedMob`:

```go
type RestModel struct {
	Id                uint32                `json:"-"`
	Name              string                `json:"name"`
	Parent            string                `json:"parent,omitempty"`
	Area              _map.Id               `json:"area"`
	Order             uint32                `json:"order,omitempty"`
	AutoStart         bool                  `json:"autoStart"`
	AutoPreComplete   bool                  `json:"autoPreComplete"`
	AutoComplete      bool                  `json:"autoComplete"`
	TimeLimit         uint32                `json:"timeLimit,omitempty"`
	TimeLimit2        uint32                `json:"timeLimit2,omitempty"`
	SelectedMob       bool                  `json:"selectedMob,omitempty"`
	SelectedSkillId   uint32                `json:"selectedSkillId,omitempty"`
	Summary           string                `json:"summary,omitempty"`
	DemandSummary     string                `json:"demandSummary,omitempty"`
	RewardSummary     string                `json:"rewardSummary,omitempty"`
	StartRequirements RequirementsRestModel `json:"startRequirements"`
	EndRequirements   RequirementsRestModel `json:"endRequirements"`
	StartActions      ActionsRestModel      `json:"startActions"`
	EndActions        ActionsRestModel      `json:"endActions"`
}
```

- [ ] **Step 1.4: Parse it in `ReadQuestInfo`**

Edit `services/atlas-data/atlas.com/data/quest/reader.go`. In `ReadQuestInfo` (at ~line 28), extend the `RestModel{...}` literal to add one field. The updated literal:

```go
			m := RestModel{
				Id:              uint32(questId),
				Name:            questNode.GetString("name", ""),
				Parent:          questNode.GetString("parent", ""),
				Area:            uint32(questNode.GetIntegerWithDefault("area", 0)),
				Order:           uint32(questNode.GetIntegerWithDefault("order", 0)),
				AutoStart:       questNode.GetBool("autoStart", false),
				AutoPreComplete: questNode.GetBool("autoPreComplete", false),
				AutoComplete:    questNode.GetBool("autoComplete", false),
				TimeLimit:       uint32(questNode.GetIntegerWithDefault("timeLimit", 0)),
				TimeLimit2:      uint32(questNode.GetIntegerWithDefault("timeLimit2", 0)),
				SelectedMob:     questNode.GetBool("selectedMob", false),
				SelectedSkillId: uint32(questNode.GetIntegerWithDefault("selectedSkillID", 0)),
				Summary:         questNode.GetString("summary", ""),
				DemandSummary:   questNode.GetString("demandSummary", ""),
				RewardSummary:   questNode.GetString("rewardSummary", ""),
			}
```

Note: the WZ attribute is `selectedSkillID` (capital `ID`). The JSON struct tag uses `selectedSkillId` to match Go naming.

- [ ] **Step 1.5: Run the test, expect pass**

```bash
cd services/atlas-data/atlas.com/data
go test ./quest/ -run TestReadQuestInfo -v
```

Expected: `PASS`.

- [ ] **Step 1.6: Run the full atlas-data test suite as a regression check**

```bash
cd services/atlas-data/atlas.com/data
go test ./... -count=1
```

Expected: all tests pass. If anything else breaks, investigate — likely a struct-literal test fixture that's now missing the new field (unlikely since it defaults to 0, but confirm).

- [ ] **Step 1.7: Commit**

```bash
cd services/atlas-data/atlas.com/data
git add ./quest/rest.go ./quest/reader.go ./quest/reader_test.go
git commit -m "feat(atlas-data): parse selectedSkillID from QuestInfo.img"
```

---

## Task 2 — atlas-quest: mirror `SelectedSkillId` on the quest-service RestModel

This task is a type-level mirror only. atlas-quest deserializes atlas-data responses into its own `RestModel`; without the field, the value is silently dropped on the quest-service side. No dedicated test — Task 4 exercises the field end-to-end.

**Files:**

- Modify: `services/atlas-quest/atlas.com/quest/data/quest/rest.go` (struct `RestModel`, insert after `SelectedMob` field)

- [ ] **Step 2.1: Add the field**

Edit `services/atlas-quest/atlas.com/quest/data/quest/rest.go`. In the `RestModel` struct (at ~line 10), insert the field after `SelectedMob`:

```go
type RestModel struct {
	Id                uint32                `json:"-"`
	Name              string                `json:"name"`
	Parent            string                `json:"parent,omitempty"`
	Area              _map.Id               `json:"area"`
	Order             uint32                `json:"order,omitempty"`
	AutoStart         bool                  `json:"autoStart"`
	AutoPreComplete   bool                  `json:"autoPreComplete"`
	AutoComplete      bool                  `json:"autoComplete"`
	TimeLimit         uint32                `json:"timeLimit,omitempty"`
	TimeLimit2        uint32                `json:"timeLimit2,omitempty"`
	SelectedMob       bool                  `json:"selectedMob,omitempty"`
	SelectedSkillId   uint32                `json:"selectedSkillId,omitempty"`
	Summary           string                `json:"summary,omitempty"`
	DemandSummary     string                `json:"demandSummary,omitempty"`
	RewardSummary     string                `json:"rewardSummary,omitempty"`
	StartRequirements RequirementsRestModel `json:"startRequirements"`
	EndRequirements   RequirementsRestModel `json:"endRequirements"`
	StartActions      ActionsRestModel      `json:"startActions"`
	EndActions        ActionsRestModel      `json:"endActions"`
}
```

- [ ] **Step 2.2: Compile-check**

```bash
cd services/atlas-quest/atlas.com/quest
go build ./...
```

Expected: build succeeds. Any compile error at this point is unrelated — the field has zero consumers yet.

- [ ] **Step 2.3: Commit**

```bash
cd services/atlas-quest/atlas.com/quest
git add ./data/quest/rest.go
git commit -m "feat(atlas-quest): mirror SelectedSkillId on quest RestModel"
```

---

## Task 3 — atlas-quest: extract `buildStartConditions` helper (pure refactor)

This task refactors the condition-building half of `ValidateStartRequirements` into a pure helper so Task 4 can TDD a new condition against it. The refactor must not change behavior.

**Files:**

- Modify: `services/atlas-quest/atlas.com/quest/data/validation/processor.go` (function `ValidateStartRequirements`, lines ~90-167)
- Create: `services/atlas-quest/atlas.com/quest/data/validation/processor_test.go`

- [ ] **Step 3.1: Extract the helper**

Edit `services/atlas-quest/atlas.com/quest/data/validation/processor.go`. Add the following new function above `ValidateStartRequirements`:

```go
// buildStartConditions translates a quest definition's start requirements into
// the wire-format conditions submitted to query-aggregator. Pure helper; no IO.
func buildStartConditions(questDef dataquest.RestModel) []ConditionInput {
	req := questDef.StartRequirements
	var conditions []ConditionInput

	// Level requirements
	if req.LevelMin > 0 {
		conditions = append(conditions, ConditionInput{
			Type:     LevelCondition,
			Operator: ">=",
			Value:    int(req.LevelMin),
		})
	}
	if req.LevelMax > 0 {
		conditions = append(conditions, ConditionInput{
			Type:     LevelCondition,
			Operator: "<=",
			Value:    int(req.LevelMax),
		})
	}

	// Job requirements - check if character's job is in the allowed list
	if len(req.Jobs) > 0 {
		jobValues := make([]int, len(req.Jobs))
		for i, job := range req.Jobs {
			jobValues[i] = int(job)
		}
		conditions = append(conditions, ConditionInput{
			Type:     JobCondition,
			Operator: "in",
			Values:   jobValues,
		})
	}

	// Fame requirement
	if req.FameMin > 0 {
		conditions = append(conditions, ConditionInput{
			Type:     FameCondition,
			Operator: ">=",
			Value:    int(req.FameMin),
		})
	}

	// Meso requirements
	if req.MesoMin > 0 {
		conditions = append(conditions, ConditionInput{
			Type:     MesoCondition,
			Operator: ">=",
			Value:    int(req.MesoMin),
		})
	}
	if req.MesoMax > 0 {
		conditions = append(conditions, ConditionInput{
			Type:     MesoCondition,
			Operator: "<=",
			Value:    int(req.MesoMax),
		})
	}

	// Item requirements
	for _, item := range req.Items {
		if item.Count > 0 {
			conditions = append(conditions, ConditionInput{
				Type:        ItemCondition,
				Operator:    ">=",
				Value:       int(item.Count),
				ReferenceId: item.Id,
			})
		}
	}

	// Prerequisite quest requirements
	for _, quest := range req.Quests {
		conditions = append(conditions, ConditionInput{
			Type:        QuestStatusCondition,
			Operator:    "=",
			Value:       int(quest.State),
			ReferenceId: quest.Id,
		})
	}

	return conditions
}
```

Replace lines ~90-167 inside `ValidateStartRequirements` (everything from `var conditions []ConditionInput` through the closing brace of the `for _, quest := range req.Quests` loop) with a single call:

```go
	conditions := buildStartConditions(questDef)
```

The body of `ValidateStartRequirements` after the refactor is (showing the relevant tail):

```go
	// Day of week requirements
	if len(req.DayOfWeek) > 0 {
		today := now.Weekday()
		allowed := false
		for _, day := range req.DayOfWeek {
			if wd, ok := wzDayToWeekday[strings.ToLower(day)]; ok && wd == today {
				allowed = true
				break
			}
		}
		if !allowed {
			return false, []string{fmt.Sprintf("wrong_day_of_week (allowed %v)", req.DayOfWeek)}, nil
		}
	}

	conditions := buildStartConditions(questDef)

	// If no conditions, validation passes
	if len(conditions) == 0 {
		return true, nil, nil
	}

	// Call query-aggregator
	result, err := requestValidation(characterId, conditions)(p.l, p.ctx)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to validate start requirements for character [%d]", characterId)
		return false, nil, err
	}

	if result.AllPassed() {
		return true, nil, nil
	}

	return false, result.GetFailedConditions(), nil
}
```

- [ ] **Step 3.2: Compile-check the refactor**

```bash
cd services/atlas-quest/atlas.com/quest
go build ./data/validation/
```

Expected: build succeeds.

- [ ] **Step 3.3: Write unit tests for the helper, establishing baseline behavior**

Create `services/atlas-quest/atlas.com/quest/data/validation/processor_test.go` with:

```go
package validation

import (
	"reflect"
	"testing"

	dataquest "atlas-quest/data/quest"
)

func TestBuildStartConditions_Empty(t *testing.T) {
	got := buildStartConditions(dataquest.RestModel{})
	if len(got) != 0 {
		t.Fatalf("expected no conditions, got %d: %+v", len(got), got)
	}
}

func TestBuildStartConditions_LevelMinMax(t *testing.T) {
	def := dataquest.RestModel{
		StartRequirements: dataquest.RequirementsRestModel{
			LevelMin: 10,
			LevelMax: 20,
		},
	}
	got := buildStartConditions(def)
	want := []ConditionInput{
		{Type: LevelCondition, Operator: ">=", Value: 10},
		{Type: LevelCondition, Operator: "<=", Value: 20},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestBuildStartConditions_Jobs(t *testing.T) {
	def := dataquest.RestModel{
		StartRequirements: dataquest.RequirementsRestModel{
			Jobs: []uint16{400, 410, 420},
		},
	}
	got := buildStartConditions(def)
	want := []ConditionInput{
		{Type: JobCondition, Operator: "in", Values: []int{400, 410, 420}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestBuildStartConditions_FameMesoItem(t *testing.T) {
	def := dataquest.RestModel{
		StartRequirements: dataquest.RequirementsRestModel{
			FameMin: 5,
			MesoMin: 1000,
			MesoMax: 5000,
			Items: []dataquest.ItemRequirement{
				{Id: 4031013, Count: 1},
				{Id: 4031014, Count: -1}, // removal; must NOT emit a condition
			},
		},
	}
	got := buildStartConditions(def)
	want := []ConditionInput{
		{Type: FameCondition, Operator: ">=", Value: 5},
		{Type: MesoCondition, Operator: ">=", Value: 1000},
		{Type: MesoCondition, Operator: "<=", Value: 5000},
		{Type: ItemCondition, Operator: ">=", Value: 1, ReferenceId: 4031013},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestBuildStartConditions_QuestPrerequisites(t *testing.T) {
	def := dataquest.RestModel{
		StartRequirements: dataquest.RequirementsRestModel{
			Quests: []dataquest.QuestRequirement{
				{Id: 2413, State: 2},
			},
		},
	}
	got := buildStartConditions(def)
	want := []ConditionInput{
		{Type: QuestStatusCondition, Operator: "=", Value: 2, ReferenceId: 2413},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}
```

- [ ] **Step 3.4: Run tests, expect pass**

```bash
cd services/atlas-quest/atlas.com/quest
go test ./data/validation/ -v
```

Expected: all tests pass. These tests capture the pre-existing condition-building behavior and will guard the refactor from regressions.

- [ ] **Step 3.5: Commit**

```bash
cd services/atlas-quest/atlas.com/quest
git add ./data/validation/processor.go ./data/validation/processor_test.go
git commit -m "refactor(atlas-quest): extract buildStartConditions helper"
```

---

## Task 4 — atlas-quest: emit `skillLevel` condition when `SelectedSkillId > 0`

**Files:**

- Modify: `services/atlas-quest/atlas.com/quest/data/validation/model.go` (add constant to the condition-type block)
- Modify: `services/atlas-quest/atlas.com/quest/data/validation/processor.go` (extend `buildStartConditions`)
- Test: `services/atlas-quest/atlas.com/quest/data/validation/processor_test.go` (extend with new tests)

- [ ] **Step 4.1: Write failing tests first**

Append to `services/atlas-quest/atlas.com/quest/data/validation/processor_test.go`:

```go
func TestBuildStartConditions_SelectedSkillId_Emits(t *testing.T) {
	def := dataquest.RestModel{
		SelectedSkillId: 4001334,
	}
	got := buildStartConditions(def)
	want := []ConditionInput{
		{Type: SkillCondition, Operator: ">=", Value: 1, ReferenceId: 4001334},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestBuildStartConditions_SelectedSkillId_Zero_DoesNotEmit(t *testing.T) {
	def := dataquest.RestModel{
		SelectedSkillId: 0,
		StartRequirements: dataquest.RequirementsRestModel{
			LevelMin: 10,
		},
	}
	got := buildStartConditions(def)
	for _, c := range got {
		if c.Type == SkillCondition {
			t.Fatalf("expected no SkillCondition when SelectedSkillId is 0, got %+v", got)
		}
	}
}

func TestBuildStartConditions_SelectedSkillId_CombinedWithOthers(t *testing.T) {
	def := dataquest.RestModel{
		SelectedSkillId: 4001344,
		StartRequirements: dataquest.RequirementsRestModel{
			Jobs: []uint16{410, 420},
			Quests: []dataquest.QuestRequirement{
				{Id: 2413, State: 2},
			},
		},
	}
	got := buildStartConditions(def)
	want := []ConditionInput{
		{Type: JobCondition, Operator: "in", Values: []int{410, 420}},
		{Type: QuestStatusCondition, Operator: "=", Value: 2, ReferenceId: 2413},
		{Type: SkillCondition, Operator: ">=", Value: 1, ReferenceId: 4001344},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}
```

- [ ] **Step 4.2: Run tests, expect failure**

```bash
cd services/atlas-quest/atlas.com/quest
go test ./data/validation/ -run TestBuildStartConditions_SelectedSkillId -v
```

Expected output: compile failure — `undefined: SkillCondition` — because the constant doesn't exist yet. Once the constant is added but the emission isn't, the tests will run and fail with empty/incorrect slices.

- [ ] **Step 4.3: Add the `SkillCondition` constant**

Edit `services/atlas-quest/atlas.com/quest/data/validation/model.go`. In the condition-types `const (...)` block (at ~line 16), add `SkillCondition`:

```go
// Condition types supported by query-aggregator
const (
	LevelCondition       = "level"
	JobCondition         = "jobId"
	FameCondition        = "fame"
	MesoCondition        = "meso"
	ItemCondition        = "item"
	QuestStatusCondition = "questStatus"
	SkillCondition       = "skillLevel"
)
```

The string value `"skillLevel"` must match query-aggregator's wire contract (see `services/atlas-query-aggregator/atlas.com/query-aggregator/validation/model.go:47` — `SkillLevelCondition = ConditionType(sharedsaga.SkillLevelCondition)`, whose string value is `"skillLevel"`, as documented in `services/atlas-query-aggregator/docs/rest.md:126`).

- [ ] **Step 4.4: Emit the condition in `buildStartConditions`**

Edit `services/atlas-quest/atlas.com/quest/data/validation/processor.go`. In `buildStartConditions`, append the new block AFTER the `// Prerequisite quest requirements` loop and BEFORE the `return conditions` line:

```go
	// Selected skill requirement (from QuestInfo.img selectedSkillID). Gates
	// tutorial quests where shared Check.img job lists would otherwise permit
	// a character to start a tutorial for a skill they don't possess.
	if questDef.SelectedSkillId > 0 {
		conditions = append(conditions, ConditionInput{
			Type:        SkillCondition,
			Operator:    ">=",
			Value:       1,
			ReferenceId: questDef.SelectedSkillId,
		})
	}

	return conditions
```

- [ ] **Step 4.5: Run the new tests, expect pass**

```bash
cd services/atlas-quest/atlas.com/quest
go test ./data/validation/ -run TestBuildStartConditions_SelectedSkillId -v
```

Expected: all three new tests pass.

- [ ] **Step 4.6: Run the full validation package tests**

```bash
cd services/atlas-quest/atlas.com/quest
go test ./data/validation/ -v
```

Expected: all tests pass (the five from Task 3 plus the three new ones).

- [ ] **Step 4.7: Commit**

```bash
cd services/atlas-quest/atlas.com/quest
git add ./data/validation/model.go ./data/validation/processor.go ./data/validation/processor_test.go
git commit -m "feat(atlas-quest): gate quest start by selectedSkillID

When a quest has a non-zero SelectedSkillId (parsed from
QuestInfo.img's selectedSkillID), ValidateStartRequirements now
emits a skillLevel >= 1 condition, which atlas-query-aggregator
evaluates via atlas-skills. This prevents cross-job players from
starting skill-tutorial quests they cannot execute - e.g., a
Bandit who completed quest 2413 can no longer start 2420 \"Using
Lucky Seven\" alongside 2418 \"Using Double Stab\"."
```

---

## Task 5 — Whole-repo regression check

**Files:** none (build/test only)

- [ ] **Step 5.1: Build and test atlas-data end-to-end**

```bash
cd services/atlas-data/atlas.com/data
go build ./...
go test ./... -count=1
```

Expected: no build failures, all tests pass.

- [ ] **Step 5.2: Build and test atlas-quest end-to-end**

```bash
cd services/atlas-quest/atlas.com/quest
go build ./...
go test ./... -count=1
```

Expected: no build failures, all tests pass. The pre-existing `processor_test.go` in `./quest/` continues to pass because `MockValidationProcessor` short-circuits the real validator — the refactored code path is not exercised there.

- [ ] **Step 5.3: If anything fails, fix and re-commit**

If a test fails, stop and investigate. Do not skip or `-run` around failing tests. Commit fixes as separate commits with a `fix(...)` prefix.

Once both services pass cleanly, the implementation is complete.

---

## Out of Scope (per design)

- **Cleanup of characters already holding both quests.** Decision locked in during brainstorming — the fix is forward-looking. Any future cleanup is a separate task.
- **Changes to atlas-query-aggregator or atlas-skills.** Already implement `skillLevel` end-to-end.
- **Gating `Complete`.** `ValidateEndRequirements` does not emit a `skillLevel` condition. A character who passed the start gate keeps the right to complete.
- **`skipValidation = true` paths.** The gate lives inside `ValidateStartRequirements`, so force-start paths (Kafka `Body.Force`, saga overrides, admin) skip it — consistent with existing semantics.
- **Manual GMS regression against the 2418/2420 pair.** Documented in `design.md` as a post-deploy verification step; not an automated test.

## Pre-verified facts from Phase-2 context gathering

- `CheckAutoStart` in `services/atlas-quest/atlas.com/quest/quest/processor.go:787` already calls `startWithDefinition` with `skipValidation=false`. Auto-start quests will honor the new gate with no additional change.
- `atlas-query-aggregator` implements `SkillLevelCondition` in full:
  - Constant at `services/atlas-query-aggregator/atlas.com/query-aggregator/validation/model.go:47`
  - Evaluator at same file, lines 795-799
  - Fetcher at `services/atlas-query-aggregator/atlas.com/query-aggregator/validation/context.go:103-115`
  - Wire-format docs at `services/atlas-query-aggregator/docs/rest.md:126`
