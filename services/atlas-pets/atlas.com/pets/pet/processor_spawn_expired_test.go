package pet_test

import (
	"atlas-pets/character"
	cm "atlas-pets/character/mock"
	"atlas-pets/data/position"
	pm "atlas-pets/data/position/mock"
	"atlas-pets/kafka/message"
	"atlas-pets/pet"
	"errors"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

func TestProcessor_SpawnExpiredPast(t *testing.T) {
	cp := &cm.Processor{}
	cp.GetByIdFn = func(m ...model.Decorator[character.Model]) func(uint32) (character.Model, error) {
		return func(uint32) (character.Model, error) {
			return character.NewModelBuilder().SetX(50).SetY(95).Build(), nil
		}
	}
	mfh := position.NewModel(99, 0, 95, 100, 95)
	pp := &pm.Processor{}
	pp.GetBelowFn = func(mapId _map.Id, x int16, y int16) model.Provider[position.Model] {
		return model.FixedProvider(mfh)
	}
	p := pet.NewProcessor(testLogger(), testContext(), testDatabase(t)).With(pet.WithCharacterProcessor(cp), pet.WithPositionProcessor(pp))

	// test setup: a pet whose expiration is in the past
	i, err := p.Create(message.NewBuffer())(mustBuild(t, pet.NewModelBuilder(0, 7000000, 5000017, "Mr. Roboto 1", 1).
		SetSlot(-1).
		SetExpiration(time.Now().Add(-1*time.Hour))))
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}

	mb := message.NewBuffer()
	err = p.Spawn(mb)(i.Id())(i.OwnerId())(false)
	if !errors.Is(err, pet.ErrPetExpired) {
		t.Fatalf("Expected ErrPetExpired, got: %v", err)
	}

	o, err := p.GetById(i.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve pet when it should exist")
	}
	if o.Slot() != -1 {
		t.Fatalf("Expired pet should not have been spawned. Slot mismatch")
	}
}

func TestProcessor_SpawnExpiredFuture(t *testing.T) {
	cp := &cm.Processor{}
	cp.GetByIdFn = func(m ...model.Decorator[character.Model]) func(uint32) (character.Model, error) {
		return func(uint32) (character.Model, error) {
			return character.NewModelBuilder().SetX(50).SetY(95).Build(), nil
		}
	}
	mfh := position.NewModel(99, 0, 95, 100, 95)
	pp := &pm.Processor{}
	pp.GetBelowFn = func(mapId _map.Id, x int16, y int16) model.Provider[position.Model] {
		return model.FixedProvider(mfh)
	}
	p := pet.NewProcessor(testLogger(), testContext(), testDatabase(t)).With(pet.WithCharacterProcessor(cp), pet.WithPositionProcessor(pp))

	// test setup: a pet whose expiration is in the future
	i, err := p.Create(message.NewBuffer())(mustBuild(t, pet.NewModelBuilder(0, 7000000, 5000017, "Mr. Roboto 1", 1).
		SetSlot(-1).
		SetExpiration(time.Now().Add(1*time.Hour))))
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}

	mb := message.NewBuffer()
	err = p.Spawn(mb)(i.Id())(i.OwnerId())(false)
	if err != nil {
		t.Fatalf("Failed to spawn pet: %v", err)
	}

	o, err := p.GetById(i.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve pet when it should exist")
	}
	if o.Slot() < 0 {
		t.Fatalf("Pet with future expiration should have spawned. Slot mismatch")
	}
}

func TestProcessor_SpawnExpiredZero(t *testing.T) {
	cp := &cm.Processor{}
	cp.GetByIdFn = func(m ...model.Decorator[character.Model]) func(uint32) (character.Model, error) {
		return func(uint32) (character.Model, error) {
			return character.NewModelBuilder().SetX(50).SetY(95).Build(), nil
		}
	}
	mfh := position.NewModel(99, 0, 95, 100, 95)
	pp := &pm.Processor{}
	pp.GetBelowFn = func(mapId _map.Id, x int16, y int16) model.Provider[position.Model] {
		return model.FixedProvider(mfh)
	}
	p := pet.NewProcessor(testLogger(), testContext(), testDatabase(t)).With(pet.WithCharacterProcessor(cp), pet.WithPositionProcessor(pp))

	// test setup: a permanent pet (zero-value expiration) is not "expired"
	i, err := p.Create(message.NewBuffer())(mustBuild(t, pet.NewModelBuilder(0, 7000000, 5000017, "Mr. Roboto 1", 1).
		SetSlot(-1).
		SetExpiration(time.Time{})))
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}
	if !i.Expiration().IsZero() {
		t.Fatalf("Expected zero-value expiration for permanent pet")
	}

	mb := message.NewBuffer()
	err = p.Spawn(mb)(i.Id())(i.OwnerId())(false)
	if err != nil {
		t.Fatalf("Failed to spawn pet: %v", err)
	}

	o, err := p.GetById(i.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve pet when it should exist")
	}
	if o.Slot() < 0 {
		t.Fatalf("Permanent pet should have spawned. Slot mismatch")
	}
}
