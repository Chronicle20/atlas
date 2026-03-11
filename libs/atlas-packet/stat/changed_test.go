package stat

import (
	"testing"

	constants "github.com/Chronicle20/atlas-constants/stat"
	pt "github.com/Chronicle20/atlas-packet/test"
)

func testStatOptions() map[string]interface{} {
	return map[string]interface{}{
		"statistics": []interface{}{
			"SKIN", "FACE", "HAIR", "PET_SN_1", "LEVEL", "JOB",
			"STRENGTH", "DEXTERITY", "INTELLIGENCE", "LUCK",
			"HP", "MAX_HP", "MP", "MAX_MP",
			"AVAILABLE_AP", "AVAILABLE_SP", "EXPERIENCE", "FAME",
			"MESO", "PET_SN_2", "PET_SN_3", "GACHAPON_EXPERIENCE",
		},
	}
}

func TestStatChangedSingleRoundTrip(t *testing.T) {
	opts := testStatOptions()
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewStatChanged([]Update{
				NewUpdate(constants.TypeLevel, 120),
			}, true)
			output := Changed{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, opts)
			if !output.ExclRequestSent() {
				t.Error("expected exclRequestSent to be true")
			}
			if len(output.Updates()) != 1 {
				t.Fatalf("updates count: got %v, want 1", len(output.Updates()))
			}
			if output.Updates()[0].Stat() != constants.TypeLevel {
				t.Errorf("stat type: got %v, want %v", output.Updates()[0].Stat(), constants.TypeLevel)
			}
			if output.Updates()[0].Value() != 120 {
				t.Errorf("value: got %v, want 120", output.Updates()[0].Value())
			}
		})
	}
}

func TestStatChangedMultipleRoundTrip(t *testing.T) {
	opts := testStatOptions()
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewStatChanged([]Update{
				NewUpdate(constants.TypeHp, 5000),
				NewUpdate(constants.TypeMp, 3000),
				NewUpdate(constants.TypeExperience, 100000),
				NewUpdate(constants.TypeMeso, 999999),
			}, false)
			output := Changed{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, opts)
			if output.ExclRequestSent() {
				t.Error("expected exclRequestSent to be false")
			}
			if len(output.Updates()) != 4 {
				t.Fatalf("updates count: got %v, want 4", len(output.Updates()))
			}
		})
	}
}

func TestStatChangedEmptyRoundTrip(t *testing.T) {
	opts := testStatOptions()
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewStatChanged(nil, false)
			output := Changed{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, opts)
			if len(output.Updates()) != 0 {
				t.Errorf("updates count: got %v, want 0", len(output.Updates()))
			}
		})
	}
}
