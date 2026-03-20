package character

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestSkillMacroRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := SkillMacro{
				macros: []SkillMacroEntry{
					{Name: "Buff", Shout: true, SkillId1: 1001003, SkillId2: 1001004, SkillId3: 0},
					{Name: "Attack", Shout: false, SkillId1: 1001005, SkillId2: 0, SkillId3: 0},
				},
			}
			output := SkillMacro{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if len(output.Macros()) != len(input.Macros()) {
				t.Fatalf("macros count: got %v, want %v", len(output.Macros()), len(input.Macros()))
			}
			for i, m := range output.Macros() {
				if m.Name != input.macros[i].Name {
					t.Errorf("macros[%d].Name: got %v, want %v", i, m.Name, input.macros[i].Name)
				}
				if m.Shout != input.macros[i].Shout {
					t.Errorf("macros[%d].Shout: got %v, want %v", i, m.Shout, input.macros[i].Shout)
				}
				if m.SkillId1 != input.macros[i].SkillId1 {
					t.Errorf("macros[%d].SkillId1: got %v, want %v", i, m.SkillId1, input.macros[i].SkillId1)
				}
			}
		})
	}
}
