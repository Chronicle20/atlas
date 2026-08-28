package quest

import (
	"reflect"
	"testing"
)

// TestTransformRoundTrip confirms Transform is the faithful inverse of
// Extract: every field set by Extract survives a Transform -> Extract round
// trip, including the two bool fields on RestModel/ActionsRestModel that the
// codemod refused to pair automatically (AutoStart/AutoPreComplete/
// AutoComplete and NormalAutoStart), and the nested requirements/actions
// slices.
func TestTransformRoundTrip(t *testing.T) {
	rm := RestModel{
		Id:              1,
		Name:            "Test Quest",
		Parent:          "Parent Quest",
		Area:            2,
		Order:           3,
		AutoStart:       true,
		AutoPreComplete: false,
		AutoComplete:    true,
		TimeLimit:       100,
		TimeLimit2:      200,
		SelectedMob:     true,
		Summary:         "summary",
		DemandSummary:   "demand",
		RewardSummary:   "reward",
		StartRequirements: RequirementsRestModel{
			NpcId:           10,
			LevelMin:        1,
			LevelMax:        200,
			FameMin:         5,
			MesoMin:         100,
			MesoMax:         1000,
			Jobs:            []uint16{100, 200},
			Quests:          []QuestRequirementRest{{Id: 1, State: 2}},
			Items:           []ItemRequirementRest{{Id: 2, Count: 3}},
			Mobs:            []MobRequirementRest{{Id: 4, Count: 5}},
			FieldEnter:      []uint32{6, 7},
			Pet:             []uint32{8},
			PetTamenessMin:  9,
			DayOfWeek:       []string{"MON"},
			Start:           "start",
			End:             "end",
			Interval:        11,
			StartScript:     "startScript",
			EndScript:       "endScript",
			InfoNumber:      12,
			NormalAutoStart: true,
			CompletionCount: 13,
		},
		EndRequirements: RequirementsRestModel{
			NpcId: 20,
		},
		StartActions: ActionsRestModel{
			NpcId:      30,
			Exp:        31,
			Money:      32,
			Fame:       33,
			Items:      []ItemRewardRest{{Id: 1, Count: 2, Job: 3, Gender: 4, Prop: 5, Period: 6, DateExpire: "d", Var: 7}},
			Skills:     []SkillRewardRest{{Id: 8, Level: 9, MasterLevel: 10, Jobs: []uint16{11}}},
			NextQuest:  34,
			BuffItemId: 35,
			Interval:   36,
			LevelMin:   37,
		},
		EndActions: ActionsRestModel{
			NpcId: 40,
		},
	}

	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	rm2, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if rm2.Id != rm.Id {
		t.Errorf("Id mismatch. Expected %d, got %d", rm.Id, rm2.Id)
	}

	m2, err := Extract(rm2)
	if err != nil {
		t.Fatalf("Extract (second pass) failed: %v", err)
	}

	if !reflect.DeepEqual(m, m2) {
		t.Errorf("round trip mismatch. Expected %+v, got %+v", m, m2)
	}
}
