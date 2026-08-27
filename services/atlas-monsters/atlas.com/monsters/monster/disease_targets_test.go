package monster

import (
	"atlas-monsters/monster/mobskill"
	"testing"

	monster2 "github.com/Chronicle20/atlas/libs/atlas-constants/monster"
)

func TestSelectDiseaseTargets(t *testing.T) {
	slow := byte(monster2.SkillTypeSlow)
	seduce := byte(monster2.SkillTypeSeduce)

	tests := []struct {
		name       string
		skillId    byte
		count      uint32
		candidates []positionedCharacter
		want       []uint32
	}{
		{
			name:    "in box and out of box",
			skillId: slow,
			count:   0,
			candidates: []positionedCharacter{
				{id: 1, x: 120, y: 210},
				{id: 2, x: 400, y: 210},
			},
			want: []uint32{1},
		},
		{
			name:    "boundary is inclusive",
			skillId: slow,
			count:   0,
			candidates: []positionedCharacter{
				{id: 1, x: 50, y: 170},
				{id: 2, x: 150, y: 230},
				{id: 3, x: 49, y: 200},
				{id: 4, x: 100, y: 231},
			},
			want: []uint32{1, 2},
		},
		{
			name:    "y outside box",
			skillId: slow,
			count:   0,
			candidates: []positionedCharacter{
				{id: 1, x: 120, y: 169},
				{id: 2, x: 120, y: 210},
			},
			want: []uint32{2},
		},
		{
			name:    "non-seduce ignores count",
			skillId: slow,
			count:   2,
			candidates: []positionedCharacter{
				{id: 1, x: 110, y: 205},
				{id: 2, x: 111, y: 205},
				{id: 3, x: 112, y: 205},
				{id: 4, x: 113, y: 205},
			},
			want: []uint32{1, 2, 3, 4},
		},
		{
			name:    "seduce caps at count",
			skillId: seduce,
			count:   2,
			candidates: []positionedCharacter{
				{id: 1, x: 110, y: 205},
				{id: 2, x: 111, y: 205},
				{id: 3, x: 112, y: 205},
				{id: 4, x: 113, y: 205},
			},
			want: []uint32{1, 2},
		},
		{
			name:    "seduce count zero does not cap",
			skillId: seduce,
			count:   0,
			candidates: []positionedCharacter{
				{id: 1, x: 110, y: 205},
				{id: 2, x: 111, y: 205},
				{id: 3, x: 112, y: 205},
			},
			want: []uint32{1, 2, 3},
		},
		{
			name:    "seduce count above candidate count",
			skillId: seduce,
			count:   5,
			candidates: []positionedCharacter{
				{id: 1, x: 110, y: 205},
				{id: 2, x: 111, y: 205},
				{id: 3, x: 112, y: 205},
			},
			want: []uint32{1, 2, 3},
		},
		{
			name:    "seduce cap applies after box filter",
			skillId: seduce,
			count:   2,
			candidates: []positionedCharacter{
				{id: 9, x: 400, y: 205},
				{id: 1, x: 110, y: 205},
				{id: 2, x: 111, y: 205},
				{id: 3, x: 112, y: 205},
			},
			want: []uint32{1, 2},
		},
		{
			name:       "no candidates",
			skillId:    seduce,
			count:      2,
			candidates: []positionedCharacter{},
			want:       nil,
		},
		{
			name:    "all candidates out of box",
			skillId: slow,
			count:   0,
			candidates: []positionedCharacter{
				{id: 1, x: 400, y: 205},
				{id: 2, x: 401, y: 205},
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sd := mobskill.NewBuilder().
				SetBoundingBox(-50, -30, 50, 30).
				SetCount(tt.count).
				Build()

			got := selectDiseaseTargets(100, 200, sd, tt.skillId, tt.candidates)

			if tt.want == nil {
				if len(got) != 0 {
					t.Fatalf("selectDiseaseTargets() = %v, want empty", got)
				}
				return
			}

			if len(got) != len(tt.want) {
				t.Fatalf("selectDiseaseTargets() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("selectDiseaseTargets() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestSelectDiseaseTargets_IsDeterministic(t *testing.T) {
	sd := mobskill.NewBuilder().
		SetBoundingBox(-50, -30, 50, 30).
		SetCount(2).
		Build()

	candidates := []positionedCharacter{
		{id: 1, x: 110, y: 205},
		{id: 2, x: 111, y: 205},
		{id: 3, x: 112, y: 205},
		{id: 4, x: 113, y: 205},
	}
	skillId := byte(monster2.SkillTypeSeduce)

	want := []uint32{1, 2}

	got1 := selectDiseaseTargets(100, 200, sd, skillId, candidates)
	got2 := selectDiseaseTargets(100, 200, sd, skillId, candidates)

	if len(got1) != len(want) || len(got2) != len(want) {
		t.Fatalf("got1 = %v, got2 = %v, want %v", got1, got2, want)
	}
	for i := range want {
		if got1[i] != want[i] {
			t.Fatalf("got1 = %v, want %v", got1, want)
		}
		if got2[i] != want[i] {
			t.Fatalf("got2 = %v, want %v", got2, want)
		}
	}
}
