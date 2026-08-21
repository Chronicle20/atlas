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

func TestGetDiseaseTargets_BoxlessWithMultiCountReturnsControllerOnly(t *testing.T) {
	var positionCalls []uint32
	p := diseaseTargetProcessor([]uint32{7, 8, 9}, nil, nil, &positionCalls)

	m := Clone(Model{}).SetX(100).SetY(200).SetControlCharacterId(7).Build()
	sd := mobskill.NewModelBuilder().SetCount(3).Build()

	got := p.getDiseaseTargets(m, sd, byte(monster2.SkillTypeSlow))

	if !reflect.DeepEqual(got, []uint32{7}) {
		t.Fatalf("expected [7]; got %v", got)
	}
	if len(positionCalls) != 0 {
		t.Fatalf("expected no position lookups; got %v", positionCalls)
	}
}

func TestGetDiseaseTargets_BoxlessWithNoControllerReturnsNothing(t *testing.T) {
	var positionCalls []uint32
	p := diseaseTargetProcessor([]uint32{7, 8, 9}, nil, nil, &positionCalls)

	m := Clone(Model{}).SetX(100).SetY(200).SetControlCharacterId(0).Build()
	sd := mobskill.NewModelBuilder().SetCount(3).Build()

	got := p.getDiseaseTargets(m, sd, byte(monster2.SkillTypeSlow))

	if len(got) != 0 {
		t.Fatalf("expected no targets; got %v", got)
	}
	if len(positionCalls) != 0 {
		t.Fatalf("expected no position lookups; got %v", positionCalls)
	}
}

func TestGetDiseaseTargets_FiltersByBoundingBox(t *testing.T) {
	var positionCalls []uint32
	positions := map[uint32][2]int16{
		1: {120, 210},
		2: {400, 210},
		3: {90, 190},
	}
	p := diseaseTargetProcessor([]uint32{1, 2, 3}, positions, nil, &positionCalls)

	m := Clone(Model{}).SetX(100).SetY(200).SetControlCharacterId(7).Build()
	sd := mobskill.NewModelBuilder().SetBoundingBox(-50, -30, 50, 30).SetCount(0).Build()

	got := p.getDiseaseTargets(m, sd, byte(monster2.SkillTypeSlow))

	if !reflect.DeepEqual(got, []uint32{1, 3}) {
		t.Fatalf("expected [1 3]; got %v", got)
	}
}

func TestGetDiseaseTargets_PreservesFieldListingOrder(t *testing.T) {
	var positionCalls []uint32
	positions := map[uint32][2]int16{
		1: {120, 210},
		2: {400, 210},
		3: {90, 190},
	}
	p := diseaseTargetProcessor([]uint32{3, 1}, positions, nil, &positionCalls)

	m := Clone(Model{}).SetX(100).SetY(200).SetControlCharacterId(7).Build()
	sd := mobskill.NewModelBuilder().SetBoundingBox(-50, -30, 50, 30).SetCount(0).Build()

	got := p.getDiseaseTargets(m, sd, byte(monster2.SkillTypeSlow))

	if !reflect.DeepEqual(got, []uint32{3, 1}) {
		t.Fatalf("expected [3 1]; got %v", got)
	}
}

func TestGetDiseaseTargets_PositionFailureExcludesOnlyThatCharacter(t *testing.T) {
	var positionCalls []uint32
	positions := map[uint32][2]int16{
		1: {110, 205},
		3: {112, 205},
	}
	positionErr := map[uint32]error{2: errors.New("boom")}
	p := diseaseTargetProcessor([]uint32{1, 2, 3}, positions, positionErr, &positionCalls)

	m := Clone(Model{}).SetX(100).SetY(200).SetControlCharacterId(7).Build()
	sd := mobskill.NewModelBuilder().SetBoundingBox(-50, -30, 50, 30).SetCount(0).Build()

	got := p.getDiseaseTargets(m, sd, byte(monster2.SkillTypeSlow))

	if !reflect.DeepEqual(got, []uint32{1, 3}) {
		t.Fatalf("expected [1 3]; got %v", got)
	}
}

func TestGetDiseaseTargets_FieldListingFailureReturnsNothing(t *testing.T) {
	var positionCalls []uint32
	emitted := 0
	p := recordingProcessor(context.Background(), diseaseTargetTenant(), &emitted)
	p.inFieldFn = func(_ field.Model) ([]uint32, error) {
		return nil, errors.New("boom")
	}
	p.positionFn = func(id uint32) (int16, int16, error) {
		positionCalls = append(positionCalls, id)
		return 0, 0, nil
	}

	m := Clone(Model{}).SetX(100).SetY(200).SetControlCharacterId(7).Build()
	sd := mobskill.NewModelBuilder().SetBoundingBox(-50, -30, 50, 30).SetCount(0).Build()

	got := p.getDiseaseTargets(m, sd, byte(monster2.SkillTypeSlow))

	if len(got) != 0 {
		t.Fatalf("expected no targets; got %v", got)
	}
	if len(positionCalls) != 0 {
		t.Fatalf("expected no position lookups; got %v", positionCalls)
	}
}

func TestGetDiseaseTargets_SeduceCapsAcrossTheShell(t *testing.T) {
	var positionCalls []uint32
	positions := map[uint32][2]int16{
		1: {110, 205},
		2: {111, 205},
		3: {112, 205},
		4: {113, 205},
	}
	p := diseaseTargetProcessor([]uint32{1, 2, 3, 4}, positions, nil, &positionCalls)

	m := Clone(Model{}).SetX(100).SetY(200).SetControlCharacterId(7).Build()
	sd := mobskill.NewModelBuilder().SetBoundingBox(-50, -30, 50, 30).SetCount(2).Build()

	got := p.getDiseaseTargets(m, sd, byte(monster2.SkillTypeSeduce))

	if !reflect.DeepEqual(got, []uint32{1, 2}) {
		t.Fatalf("expected [1 2]; got %v", got)
	}
}

func TestGetDiseaseTargets_ConcurrentLookupsPreserveOrder(t *testing.T) {
	var mu sync.Mutex
	var positionCalls []uint32

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
		positionCalls = append(positionCalls, id)
		mu.Unlock()
		if id%2 == 1 {
			time.Sleep(5 * time.Millisecond)
		}
		return 110, 205, nil
	}

	m := Clone(Model{}).SetX(100).SetY(200).SetControlCharacterId(7).Build()
	sd := mobskill.NewModelBuilder().SetBoundingBox(-50, -30, 50, 30).SetCount(0).Build()

	got := p.getDiseaseTargets(m, sd, byte(monster2.SkillTypeSlow))

	want := make([]uint32, 20)
	for i := range want {
		want[i] = uint32(i + 1)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected ascending 1..20; got %v", got)
	}
}
