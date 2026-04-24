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
