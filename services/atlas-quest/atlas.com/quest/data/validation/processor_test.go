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
