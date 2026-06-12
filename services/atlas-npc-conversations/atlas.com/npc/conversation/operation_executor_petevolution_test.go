package conversation

import (
	"context"
	"testing"

	"atlas-npc-conversations/pet"
	"atlas-npc-conversations/petdata"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

// fakePetProcessor is an inline test double for pet.Processor.
type fakePetProcessor struct {
	pets []pet.Model
}

func (f fakePetProcessor) GetPets(characterId uint32) model.Provider[[]pet.Model] {
	return func() ([]pet.Model, error) { return f.pets, nil }
}

func (f fakePetProcessor) GetPetIdBySlot(characterId uint32, slot int8) model.Provider[uint32] {
	return func() (uint32, error) {
		for _, p := range f.pets {
			if p.Slot() == slot {
				return p.Id(), nil
			}
		}
		return 0, nil
	}
}

// fakePetDataProcessor is an inline test double for petdata.Processor.
type fakePetDataProcessor struct {
	byId map[uint32]petdata.Model
}

func (f fakePetDataProcessor) GetById(petTemplateId uint32) (petdata.Model, error) {
	return f.byId[petTemplateId], nil
}

func TestEnumerateEvolvablePets(t *testing.T) {
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(rc)

	l, _ := test.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)

	var tm tenant.Model
	tctx := tenant.WithContext(context.Background(), tm)
	characterId := uint32(100)

	// Seed an initial context value so the context map is non-nil and
	// setContextValue can write back the enumerated results.
	convCtx := NewConversationContextBuilder().
		SetCharacterId(characterId).
		AddContextValue("_seed", "1").
		Build()
	GetRegistry().SetContext(tctx, characterId, convCtx)
	defer GetRegistry().ClearContext(tctx, characterId)

	// Two summoned pets sharing template 5000029: one level-20 (eligible),
	// one level-10 (below the reqPetLevel of 15).
	petP := fakePetProcessor{pets: []pet.Model{
		pet.NewModel(1, 5000029, 20, 0),
		pet.NewModel(2, 5000029, 10, 1),
	}}
	// Template 5000029 is evolvable, requiring pet level 15.
	petdataP := fakePetDataProcessor{byId: map[uint32]petdata.Model{
		5000029: petdata.NewModel(5000029, 15, 4000000, 1),
	}}

	executor := &OperationExecutorImpl{
		l:        l,
		ctx:      tctx,
		t:        tm,
		petP:     petP,
		petdataP: petdataP,
	}

	op, err := NewOperationBuilder().
		SetType("local:enumerate_evolvable_pets").
		AddParamValue("outputContextKey", "evolvablePets").
		AddParamValue("countContextKey", "evolvableCount").
		Build()
	if err != nil {
		t.Fatalf("failed to build op: %v", err)
	}

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()

	if err := executor.ExecuteOperation(f, characterId, op); err != nil {
		t.Fatalf("ExecuteOperation returned error: %v", err)
	}

	count, err := executor.getContextValue(characterId, "evolvableCount")
	if err != nil {
		t.Fatalf("failed to read evolvableCount: %v", err)
	}
	if count != "1" {
		t.Errorf("evolvableCount = %q, want %q", count, "1")
	}

	ids, err := executor.getContextValue(characterId, "evolvablePets")
	if err != nil {
		t.Fatalf("failed to read evolvablePets: %v", err)
	}
	if ids != "1" {
		t.Errorf("evolvablePets = %q, want %q", ids, "1")
	}
}
