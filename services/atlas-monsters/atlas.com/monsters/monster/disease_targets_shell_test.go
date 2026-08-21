package monster

import (
	"atlas-monsters/monster/mobskill"
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	monster2 "github.com/Chronicle20/atlas/libs/atlas-constants/monster"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// diseaseTargetTenant builds a tenant.Model without requiring a *testing.T,
// so diseaseTargetProcessor can stay a plain builder function. tenant.Create
// only errors on malformed inputs, which these fixed literals never produce.
func diseaseTargetTenant() tenant.Model {
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		panic(err)
	}
	return tm
}

// diseaseTargetProcessor builds a ProcessorImpl with the two seams
// getDiseaseTargets consults: the field listing and the per-character
// position lookup. positionCalls records every id positionFn was asked for,
// so a test can assert the single-target path made no lookup at all.
func diseaseTargetProcessor(inField []uint32, positions map[uint32][2]int16, positionErr map[uint32]error, positionCalls *[]uint32) *ProcessorImpl {
	var mu sync.Mutex
	emitted := 0
	p := recordingProcessor(context.Background(), diseaseTargetTenant(), &emitted)
	p.inFieldFn = func(_ field.Model) ([]uint32, error) {
		return inField, nil
	}
	p.positionFn = func(id uint32) (int16, int16, error) {
		mu.Lock()
		*positionCalls = append(*positionCalls, id)
		mu.Unlock()
		if err, ok := positionErr[id]; ok {
			return 0, 0, err
		}
		pos := positions[id]
		return pos[0], pos[1], nil
	}
	return p
}

func TestGetDiseaseTargets(t *testing.T) {
	slow := byte(monster2.SkillTypeSlow)
	seduce := byte(monster2.SkillTypeSeduce)

	tests := []struct {
		name                string
		build               func(positionCalls *[]uint32) *ProcessorImpl
		controlCharacterId  uint32
		useBoundingBox      bool
		count               uint32
		skillId             byte
		want                []uint32
		wantNoPositionCalls bool
	}{
		{
			name: "boxless with multi-count returns controller only",
			build: func(positionCalls *[]uint32) *ProcessorImpl {
				return diseaseTargetProcessor([]uint32{7, 8, 9}, nil, nil, positionCalls)
			},
			controlCharacterId:  7,
			useBoundingBox:      false,
			count:               3,
			skillId:             slow,
			want:                []uint32{7},
			wantNoPositionCalls: true,
		},
		{
			name: "boxless with no controller returns nothing",
			build: func(positionCalls *[]uint32) *ProcessorImpl {
				return diseaseTargetProcessor([]uint32{7, 8, 9}, nil, nil, positionCalls)
			},
			controlCharacterId:  0,
			useBoundingBox:      false,
			count:               3,
			skillId:             slow,
			want:                nil,
			wantNoPositionCalls: true,
		},
		{
			name: "filters by bounding box",
			build: func(positionCalls *[]uint32) *ProcessorImpl {
				positions := map[uint32][2]int16{
					1: {120, 210},
					2: {400, 210},
					3: {90, 190},
				}
				return diseaseTargetProcessor([]uint32{1, 2, 3}, positions, nil, positionCalls)
			},
			controlCharacterId: 7,
			useBoundingBox:     true,
			count:              0,
			skillId:            slow,
			want:               []uint32{1, 3},
		},
		{
			name: "preserves field listing order",
			build: func(positionCalls *[]uint32) *ProcessorImpl {
				positions := map[uint32][2]int16{
					1: {120, 210},
					2: {400, 210},
					3: {90, 190},
				}
				return diseaseTargetProcessor([]uint32{3, 1}, positions, nil, positionCalls)
			},
			controlCharacterId: 7,
			useBoundingBox:     true,
			count:              0,
			skillId:            slow,
			want:               []uint32{3, 1},
		},
		{
			name: "position failure excludes only that character",
			build: func(positionCalls *[]uint32) *ProcessorImpl {
				positions := map[uint32][2]int16{
					1: {110, 205},
					3: {112, 205},
				}
				positionErr := map[uint32]error{2: errors.New("boom")}
				return diseaseTargetProcessor([]uint32{1, 2, 3}, positions, positionErr, positionCalls)
			},
			controlCharacterId: 7,
			useBoundingBox:     true,
			count:              0,
			skillId:            slow,
			want:               []uint32{1, 3},
		},
		{
			name: "field listing failure returns nothing",
			build: func(positionCalls *[]uint32) *ProcessorImpl {
				emitted := 0
				p := recordingProcessor(context.Background(), diseaseTargetTenant(), &emitted)
				p.inFieldFn = func(_ field.Model) ([]uint32, error) {
					return nil, errors.New("boom")
				}
				p.positionFn = func(id uint32) (int16, int16, error) {
					*positionCalls = append(*positionCalls, id)
					return 0, 0, nil
				}
				return p
			},
			controlCharacterId:  7,
			useBoundingBox:      true,
			count:               0,
			skillId:             slow,
			want:                nil,
			wantNoPositionCalls: true,
		},
		{
			name: "seduce caps across the shell",
			build: func(positionCalls *[]uint32) *ProcessorImpl {
				positions := map[uint32][2]int16{
					1: {110, 205},
					2: {111, 205},
					3: {112, 205},
					4: {113, 205},
				}
				return diseaseTargetProcessor([]uint32{1, 2, 3, 4}, positions, nil, positionCalls)
			},
			controlCharacterId: 7,
			useBoundingBox:     true,
			count:              2,
			skillId:            seduce,
			want:               []uint32{1, 2},
		},
		{
			name: "concurrent lookups preserve order",
			build: func(positionCalls *[]uint32) *ProcessorImpl {
				var mu sync.Mutex
				inField := make([]uint32, 20)
				for i := range inField {
					inField[i] = uint32(i + 1)
				}

				emitted := 0
				p := recordingProcessor(context.Background(), diseaseTargetTenant(), &emitted)
				p.inFieldFn = func(_ field.Model) ([]uint32, error) {
					return inField, nil
				}
				p.positionFn = func(id uint32) (int16, int16, error) {
					mu.Lock()
					*positionCalls = append(*positionCalls, id)
					mu.Unlock()
					if id%2 == 1 {
						time.Sleep(5 * time.Millisecond)
					}
					return 110, 205, nil
				}
				return p
			},
			controlCharacterId: 7,
			useBoundingBox:     true,
			count:              0,
			skillId:            slow,
			want: func() []uint32 {
				want := make([]uint32, 20)
				for i := range want {
					want[i] = uint32(i + 1)
				}
				return want
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var positionCalls []uint32
			p := tt.build(&positionCalls)

			m := Clone(Model{}).SetX(100).SetY(200).SetControlCharacterId(tt.controlCharacterId).Build()
			builder := mobskill.NewModelBuilder().SetCount(tt.count)
			if tt.useBoundingBox {
				builder = builder.SetBoundingBox(-50, -30, 50, 30)
			}
			sd := builder.Build()

			got := p.getDiseaseTargets(m, sd, tt.skillId)

			if tt.want == nil {
				if len(got) != 0 {
					t.Fatalf("expected no targets; got %v", got)
				}
			} else if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("expected %v; got %v", tt.want, got)
			}

			if tt.wantNoPositionCalls && len(positionCalls) != 0 {
				t.Fatalf("expected no position lookups; got %v", positionCalls)
			}
		})
	}
}
