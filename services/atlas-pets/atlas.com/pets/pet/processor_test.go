package pet_test

import (
	"atlas-pets/asset"
	"atlas-pets/character"
	cm "atlas-pets/character/mock"
	"atlas-pets/compartment"
	data2 "atlas-pets/data/pet"
	pdm "atlas-pets/data/pet/mock"
	"atlas-pets/data/position"
	pm "atlas-pets/data/position/mock"
	"atlas-pets/inventory"
	"atlas-pets/kafka/message"
	pet2 "atlas-pets/kafka/message/pet"
	"atlas-pets/pet"
	"atlas-pets/pet/exclude"
	sm "atlas-pets/skill/mock"
	"context"
	"errors"
	"fmt"
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	inventory2 "github.com/Chronicle20/atlas-constants/inventory"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/skill"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"testing"
)

func testLogger() logrus.FieldLogger {
	l, _ := test.NewNullLogger()
	return l
}

func testContext() context.Context {
	t, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return tenant.WithContext(context.Background(), t)
}

func testDatabase(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	var migrators []func(db *gorm.DB) error
	migrators = append(migrators, pet.Migration, exclude.Migration)

	for _, migrator := range migrators {
		if err = migrator(db); err != nil {
			t.Fatalf("Failed to migrate database: %v", err)
		}
	}
	return db
}

func mustBuild(t *testing.T, b *pet.ModelBuilder) pet.Model {
	m, err := b.Build()
	if err != nil {
		t.Fatalf("Failed to build pet model: %v", err)
	}
	return m
}

func TestProcessor_Create(t *testing.T) {
	p := pet.NewProcessor(testLogger(), testContext(), testDatabase(t))

	// test execution
	mb := message.NewBuffer()
	i := mustBuild(t, pet.NewModelBuilder(0, 7000000, 5000017, "Mr. Roboto", 1))
	o, err := p.Create(mb)(i)
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}
	if o.Id() == 0 {
		t.Fatalf("Failed to create pet. Id was not generated")
	}
	if o.CashId() != i.CashId() {
		t.Fatalf("Failed to create pet. CashId mismatch")
	}
	if o.TemplateId() != i.TemplateId() {
		t.Fatalf("Failed to create pet. TemplateId mismatch")
	}
	if o.Name() != i.Name() {
		t.Fatalf("Failed to create pet. Name mismatch")
	}
	if o.OwnerId() != i.OwnerId() {
		t.Fatalf("Failed to create pet. OwnerId mismatch")
	}
	if o.Level() != 1 {
		t.Fatalf("Failed to create pet. Level not set to 1")
	}
	if o.Fullness() != 100 {
		t.Fatalf("Failed to create pet. Fullness not set to 100")
	}
	if o.Closeness() != 0 {
		t.Fatalf("Failed to create pet. Closeness not set to 0")
	}
	ke := mb.GetAll()
	var se []kafka.Message
	var ok bool
	if se, ok = ke[pet2.EnvStatusEventTopic]; !ok {
		t.Fatalf("Failed to get events from topic: %s", pet2.EnvStatusEventTopic)
	}
	if len(se) != 1 {
		t.Fatalf("Failed to expected events from topic: %s", pet2.EnvStatusEventTopic)
	}
}

func TestProcessor_Delete(t *testing.T) {
	p := pet.NewProcessor(testLogger(), testContext(), testDatabase(t))

	// test setup
	i, err := p.Create(message.NewBuffer())(mustBuild(t, pet.NewModelBuilder(0, 7000000, 5000017, "Mr. Roboto", 1)))
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}

	// test execution
	mb := message.NewBuffer()
	err = p.Delete(mb)(i.Id())(i.OwnerId())
	if err != nil {
		t.Fatalf("Failed to delete pet: %v", err)
	}
	ke := mb.GetAll()
	var se []kafka.Message
	var ok bool
	if se, ok = ke[pet2.EnvStatusEventTopic]; !ok {
		t.Fatalf("Failed to get events from topic: %s", pet2.EnvStatusEventTopic)
	}
	if len(se) != 1 {
		t.Fatalf("Failed to expected events from topic: %s", pet2.EnvStatusEventTopic)
	}

	_, err = p.GetById(i.Id())
	if err == nil {
		t.Fatalf("Failed to delete pet when it should not exist")
	}
	ps, err := p.GetByOwner(i.OwnerId())
	if err == nil && len(ps) != 0 {
		t.Fatalf("Failed to delete pet when it should not exist")
	}
}

func TestProcessor_DeleteOnRemove(t *testing.T) {
	characterId := uint32(1)
	templateId := uint32(5000017)
	slot := int16(15)
	petId := uint32(1)

	cp := &cm.Processor{}
	cp.GetByIdFn = func(...model.Decorator[character.Model]) func(uint32) (character.Model, error) {
		return func(uint32) (character.Model, error) {
			return character.NewModelBuilder().
				SetInventory(inventory.NewBuilder(characterId).
					SetCash(compartment.NewBuilder(uuid.New(), characterId, inventory2.TypeValueCash, 24).
						AddAsset(asset.NewBuilder[any](1, uuid.Nil, templateId, petId, asset.ReferenceTypePet).
							SetSlot(slot).
							SetReferenceData(asset.NewPetReferenceDataBuilder().
								Build()).
							Build()).
						Build()).
					Build()).
				Build(), nil
		}
	}
	p := pet.NewProcessor(testLogger(), testContext(), testDatabase(t)).With(pet.WithCharacterProcessor(cp))

	mb := message.NewBuffer()
	err := p.DeleteOnRemove(mb)(characterId)(templateId)(slot)
	if err != nil {
		t.Fatalf("Failed to delete pet: %v", err)
	}
	ke := mb.GetAll()
	var se []kafka.Message
	var ok bool
	if se, ok = ke[pet2.EnvStatusEventTopic]; !ok {
		t.Fatalf("Failed to get events from topic: %s", pet2.EnvStatusEventTopic)
	}
	if len(se) != 1 {
		t.Fatalf("Failed to expected events from topic: %s", pet2.EnvStatusEventTopic)
	}

	_, err = p.GetById(petId)
	if err == nil {
		t.Fatalf("Failed to delete pet when it should not exist")
	}
	ps, err := p.GetByOwner(characterId)
	if err == nil && len(ps) != 0 {
		t.Fatalf("Failed to delete pet when it should not exist")
	}
}

func TestProcessor_DeleteForCharacter(t *testing.T) {
	p := pet.NewProcessor(testLogger(), testContext(), testDatabase(t))
	// test setup
	_, err := p.Create(message.NewBuffer())(mustBuild(t, pet.NewModelBuilder(0, 7000000, 5000017, "Mr. Roboto 1", 1)))
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}
	_, err = p.Create(message.NewBuffer())(mustBuild(t, pet.NewModelBuilder(0, 7000001, 5000017, "Mr. Roboto 2", 1)))
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}
	_, err = p.Create(message.NewBuffer())(mustBuild(t, pet.NewModelBuilder(0, 7000002, 5000017, "Mr. Roboto 3", 1)))
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}

	// test execution
	mb := message.NewBuffer()
	err = p.DeleteForCharacter(mb)(1)
	if err != nil {
		t.Fatalf("Failed to delete pets: %v", err)
	}

	ke := mb.GetAll()
	var se []kafka.Message
	var ok bool
	if se, ok = ke[pet2.EnvStatusEventTopic]; !ok {
		t.Fatalf("Failed to get events from topic: %s", pet2.EnvStatusEventTopic)
	}
	if len(se) != 3 {
		t.Fatalf("Failed to expected events from topic: %s", pet2.EnvStatusEventTopic)
	}
}

func TestProcessor_GetById(t *testing.T) {
	p := pet.NewProcessor(testLogger(), testContext(), testDatabase(t))

	// test setup
	_, err := p.Create(message.NewBuffer())(mustBuild(t, pet.NewModelBuilder(0, 7000000, 5000017, "Mr. Roboto 1", 1)))
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}
	i, err := p.Create(message.NewBuffer())(mustBuild(t, pet.NewModelBuilder(0, 7000001, 5000017, "Mr. Roboto 2", 1)))
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}
	_, err = p.Create(message.NewBuffer())(mustBuild(t, pet.NewModelBuilder(0, 7000002, 5000017, "Mr. Roboto 3", 1)))
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}

	// test execution
	o, err := p.GetById(i.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve pet when it should exist")
	}
	if o.CashId() != i.CashId() {
		t.Fatalf("Failed to retrieve pet. CashId mismatch")
	}
	if o.Name() != i.Name() {
		t.Fatalf("Failed to retrieve pet. Name mismatch")
	}
}

func TestProcessor_GetByOwner(t *testing.T) {
	p := pet.NewProcessor(testLogger(), testContext(), testDatabase(t))

	// test setup
	_, err := p.Create(message.NewBuffer())(mustBuild(t, pet.NewModelBuilder(0, 7000000, 5000017, "Mr. Roboto 1", 1)))
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}
	i, err := p.Create(message.NewBuffer())(mustBuild(t, pet.NewModelBuilder(0, 7000001, 5000017, "Mr. Roboto 2", 2)))
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}
	_, err = p.Create(message.NewBuffer())(mustBuild(t, pet.NewModelBuilder(0, 7000002, 5000017, "Mr. Roboto 3", 3)))
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}

	// test execution
	os, err := p.GetByOwner(i.OwnerId())
	if err != nil {
		t.Fatalf("Failed to retrieve pet when it should exist")
	}
	if len(os) != 1 {
		t.Fatalf("Failed to retrieve correct number of pets")
	}
	o := os[0]
	if o.CashId() != i.CashId() {
		t.Fatalf("Failed to retrieve pet. CashId mismatch")
	}
	if o.Name() != i.Name() {
		t.Fatalf("Failed to retrieve pet. Name mismatch")
	}
}

func TestProcessor_Move(t *testing.T) {
	mfh := position.NewModel(99, 0, 95, 100, 95)
	pp := &pm.Processor{}
	pp.GetBelowFn = func(mapId uint32, x int16, y int16) model.Provider[position.Model] {
		return model.FixedProvider(mfh)
	}

	p := pet.NewProcessor(testLogger(), testContext(), testDatabase(t)).With(pet.WithPositionProcessor(pp))

	// test setup
	i, err := p.Create(message.NewBuffer())(mustBuild(t, pet.NewModelBuilder(0, 7000000, 5000017, "Mr. Roboto 1", 1)))
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}

	x := int16(50)
	y := int16(95)
	stance := byte(1)

	// test execution
	err = p.Move(i.Id(), field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(50000)).SetInstance(uuid.Nil).Build(), i.OwnerId(), x, y, stance)
	if err != nil {
		t.Fatalf("Failed to move pet: %v", err)
	}

	td := pet.GetTemporalRegistry().GetById(i.Id())
	if td == nil {
		t.Fatalf("Failed to get temporal data")
	}
	if td.X() != x {
		t.Fatalf("Failed to move pet. x mismatch")
	}
	if td.Y() != y {
		t.Fatalf("Failed to move pet. y mismatch")
	}
	if td.Stance() != stance {
		t.Fatalf("Failed to move pet. stance mismatch")
	}
	if td.FH() != int16(mfh.Id()) {
		t.Fatalf("Failed to move pet. FH mismatch")
	}

}

func TestProcessor_SpawnSingleLead(t *testing.T) {
	cp := &cm.Processor{}
	cp.GetByIdFn = func(m ...model.Decorator[character.Model]) func(uint32) (character.Model, error) {
		return func(uint32) (character.Model, error) {
			return character.NewModelBuilder().SetX(50).SetY(95).Build(), nil
		}
	}
	mfh := position.NewModel(99, 0, 95, 100, 95)
	pp := &pm.Processor{}
	pp.GetBelowFn = func(mapId uint32, x int16, y int16) model.Provider[position.Model] {
		return model.FixedProvider(mfh)
	}
	p := pet.NewProcessor(testLogger(), testContext(), testDatabase(t)).With(pet.WithCharacterProcessor(cp), pet.WithPositionProcessor(pp))

	// test setup
	i, err := p.Create(message.NewBuffer())(mustBuild(t, pet.NewModelBuilder(0, 7000000, 5000017, "Mr. Roboto 1", 1).SetSlot(-1)))
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}

	mb := message.NewBuffer()
	err = p.Spawn(mb)(i.Id())(i.OwnerId())(true)
	if err != nil {
		t.Fatalf("Failed to spawn pet: %v", err)
	}
	ke := mb.GetAll()
	var se []kafka.Message
	var ok bool
	if se, ok = ke[pet2.EnvStatusEventTopic]; !ok {
		t.Fatalf("Failed to get events from topic: %s", pet2.EnvStatusEventTopic)
	}
	if len(se) != 2 {
		t.Fatalf("Failed to expected events from topic: %s", pet2.EnvStatusEventTopic)
	}

	o, err := p.GetById(i.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve pet when it should exist")
	}
	if o.Slot() != 0 {
		t.Fatalf("Failed to spawn pet. Slot mismatch")
	}
}

func TestProcessor_SpawnMigrateLead(t *testing.T) {
	cp := &cm.Processor{}
	cp.GetByIdFn = func(m ...model.Decorator[character.Model]) func(uint32) (character.Model, error) {
		return func(uint32) (character.Model, error) {
			return character.NewModelBuilder().
				SetX(50).
				SetY(95).
				Build(), nil
		}
	}
	mfh := position.NewModel(99, 0, 95, 100, 95)
	pp := &pm.Processor{}
	pp.GetBelowFn = func(mapId uint32, x int16, y int16) model.Provider[position.Model] {
		return model.FixedProvider(mfh)
	}

	sp := &sm.Processor{}
	sp.HasSkillFn = func(characterId uint32, ids ...skill.Id) bool {
		return true
	}
	p := pet.NewProcessor(testLogger(), testContext(), testDatabase(t)).With(pet.WithCharacterProcessor(cp), pet.WithPositionProcessor(pp), pet.WithSkillProcessor(sp))

	// test setup
	p1, err := p.Create(message.NewBuffer())(mustBuild(t, pet.NewModelBuilder(0, 7000000, 5000017, "Mr. Roboto 1", 1).SetSlot(0)))
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}
	if p1.Slot() != 0 {
		t.Fatalf("Failed to spawn pet. Slot mismatch")
	}
	i, err := p.Create(message.NewBuffer())(mustBuild(t, pet.NewModelBuilder(0, 7000001, 5000017, "Mr. Roboto 2", 1).SetSlot(-1)))
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}

	mb := message.NewBuffer()
	err = p.Spawn(mb)(i.Id())(i.OwnerId())(true)
	if err != nil {
		t.Fatalf("Failed to spawn pet: %v", err)
	}
	ke := mb.GetAll()
	var se []kafka.Message
	var ok bool
	if se, ok = ke[pet2.EnvStatusEventTopic]; !ok {
		t.Fatalf("Failed to get events from topic: %s", pet2.EnvStatusEventTopic)
	}
	if len(se) != 3 {
		t.Fatalf("Failed to expected events from topic: %s", pet2.EnvStatusEventTopic)
	}

	o, err := p.GetById(i.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve pet when it should exist")
	}
	if o.Slot() != 0 {
		t.Fatalf("Failed to spawn pet. Slot mismatch")
	}
}

func TestProcessor_SpawnMissingMulti(t *testing.T) {
	cp := &cm.Processor{}
	cp.GetByIdFn = func(m ...model.Decorator[character.Model]) func(uint32) (character.Model, error) {
		return func(uint32) (character.Model, error) {
			return character.NewModelBuilder().SetX(50).SetY(95).Build(), nil
		}
	}
	mfh := position.NewModel(99, 0, 95, 100, 95)
	pp := &pm.Processor{}
	pp.GetBelowFn = func(mapId uint32, x int16, y int16) model.Provider[position.Model] {
		return model.FixedProvider(mfh)
	}
	sp := &sm.Processor{}
	sp.HasSkillFn = func(characterId uint32, ids ...skill.Id) bool {
		return false
	}
	p := pet.NewProcessor(testLogger(), testContext(), testDatabase(t)).With(pet.WithCharacterProcessor(cp), pet.WithPositionProcessor(pp), pet.WithSkillProcessor(sp))

	// test setup
	p1, err := p.Create(message.NewBuffer())(mustBuild(t, pet.NewModelBuilder(0, 7000000, 5000017, "Mr. Roboto 1", 1).SetSlot(0)))
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}
	if p1.Slot() != 0 {
		t.Fatalf("Failed to spawn pet. Slot mismatch")
	}
	i, err := p.Create(message.NewBuffer())(mustBuild(t, pet.NewModelBuilder(0, 7000001, 5000017, "Mr. Roboto 2", 1).SetSlot(-1)))
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}

	mb := message.NewBuffer()
	err = p.Spawn(mb)(i.Id())(i.OwnerId())(true)
	if !errors.Is(err, pet.ErrNeedMultiPetSkill) {
		t.Fatalf("Expected ErrNeedMultiPetSkill")
	}
}

func TestProcessor_SpawnNonLead(t *testing.T) {
	cp := &cm.Processor{}
	cp.GetByIdFn = func(m ...model.Decorator[character.Model]) func(uint32) (character.Model, error) {
		return func(uint32) (character.Model, error) {
			return character.NewModelBuilder().SetX(50).SetY(95).Build(), nil
		}
	}
	mfh := position.NewModel(99, 0, 95, 100, 95)
	pp := &pm.Processor{}
	pp.GetBelowFn = func(mapId uint32, x int16, y int16) model.Provider[position.Model] {
		return model.FixedProvider(mfh)
	}
	p := pet.NewProcessor(testLogger(), testContext(), testDatabase(t)).With(pet.WithCharacterProcessor(cp), pet.WithPositionProcessor(pp))

	// test setup
	_, err := p.Create(message.NewBuffer())(mustBuild(t, pet.NewModelBuilder(0, 7000000, 5000017, "Mr. Roboto 1", 1).SetSlot(0)))
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}
	i, err := p.Create(message.NewBuffer())(mustBuild(t, pet.NewModelBuilder(0, 7000001, 5000017, "Mr. Roboto 2", 1).SetSlot(-1)))
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}

	mb := message.NewBuffer()
	err = p.Spawn(mb)(i.Id())(i.OwnerId())(false)
	if err != nil {
		t.Fatalf("Failed to spawn pet: %v", err)
	}
	ke := mb.GetAll()
	var se []kafka.Message
	var ok bool
	if se, ok = ke[pet2.EnvStatusEventTopic]; !ok {
		t.Fatalf("Failed to get events from topic: %s", pet2.EnvStatusEventTopic)
	}
	if len(se) != 2 {
		t.Fatalf("Failed to expected events from topic: %s", pet2.EnvStatusEventTopic)
	}

	o, err := p.GetById(i.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve pet when it should exist")
	}
	if o.Slot() != 1 {
		t.Fatalf("Failed to spawn pet. Slot mismatch")
	}
}

func TestProcessor_DespawnSingleLead(t *testing.T) {
	p := pet.NewProcessor(testLogger(), testContext(), testDatabase(t))

	// test setup
	i, err := p.Create(message.NewBuffer())(mustBuild(t, pet.NewModelBuilder(0, 7000000, 5000017, "Mr. Roboto 1", 1).SetSlot(0)))
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}

	mb := message.NewBuffer()
	err = p.Despawn(mb)(i.Id())(i.OwnerId())(pet2.DespawnReasonHunger)
	if err != nil {
		t.Fatalf("Failed to despawn pet: %v", err)
	}
	ke := mb.GetAll()
	var se []kafka.Message
	var ok bool
	if se, ok = ke[pet2.EnvStatusEventTopic]; !ok {
		t.Fatalf("Failed to get events from topic: %s", pet2.EnvStatusEventTopic)
	}
	if len(se) != 2 {
		t.Fatalf("Failed to expected events from topic: %s", pet2.EnvStatusEventTopic)
	}

	o, err := p.GetById(i.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve pet when it should exist")
	}
	if o.Slot() != -1 {
		t.Fatalf("Failed to despawn pet. Slot mismatch")
	}
}

func TestProcessor_DespawnMigrateLead(t *testing.T) {
	p := pet.NewProcessor(testLogger(), testContext(), testDatabase(t))

	// test setup
	p1, err := p.Create(message.NewBuffer())(mustBuild(t, pet.NewModelBuilder(0, 7000000, 5000017, "Mr. Roboto 1", 1).SetSlot(0)))
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}
	p2, err := p.Create(message.NewBuffer())(mustBuild(t, pet.NewModelBuilder(0, 7000001, 5000017, "Mr. Roboto 2", 1).SetSlot(1)))
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}

	mb := message.NewBuffer()
	err = p.Despawn(mb)(p1.Id())(p2.OwnerId())(pet2.DespawnReasonHunger)
	if err != nil {
		t.Fatalf("Failed to despawn pet: %v", err)
	}
	ke := mb.GetAll()
	var se []kafka.Message
	var ok bool
	if se, ok = ke[pet2.EnvStatusEventTopic]; !ok {
		t.Fatalf("Failed to get events from topic: %s", pet2.EnvStatusEventTopic)
	}
	if len(se) != 3 {
		t.Fatalf("Failed to expected events from topic: %s", pet2.EnvStatusEventTopic)
	}

	o1, err := p.GetById(p1.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve pet when it should exist")
	}
	if o1.Slot() != -1 {
		t.Fatalf("Failed to despawn pet. Slot mismatch")
	}

	o2, err := p.GetById(p2.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve pet when it should exist")
	}
	if o2.Slot() != 0 {
		t.Fatalf("Failed to despawn pet. Slot mismatch")
	}
}

func TestProcessor_DespawnNonLead(t *testing.T) {
	p := pet.NewProcessor(testLogger(), testContext(), testDatabase(t))

	// test setup
	p1, err := p.Create(message.NewBuffer())(mustBuild(t, pet.NewModelBuilder(0, 7000000, 5000017, "Mr. Roboto 1", 1).SetSlot(0)))
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}
	if p1.Slot() != 0 {
		t.Fatalf("Failed to spawn pet. Slot mismatch")
	}
	p2, err := p.Create(message.NewBuffer())(mustBuild(t, pet.NewModelBuilder(0, 7000001, 5000017, "Mr. Roboto 2", 1).SetSlot(1)))
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}

	mb := message.NewBuffer()
	err = p.Despawn(mb)(p2.Id())(p2.OwnerId())(pet2.DespawnReasonHunger)
	if err != nil {
		t.Fatalf("Failed to spawn pet: %v", err)
	}
	ke := mb.GetAll()
	var se []kafka.Message
	var ok bool
	if se, ok = ke[pet2.EnvStatusEventTopic]; !ok {
		t.Fatalf("Failed to get events from topic: %s", pet2.EnvStatusEventTopic)
	}
	if len(se) != 2 {
		t.Fatalf("Failed to expected events from topic: %s", pet2.EnvStatusEventTopic)
	}

	o2, err := p.GetById(p2.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve pet when it should exist")
	}
	if o2.Slot() != -1 {
		t.Fatalf("Failed to spawn pet. Slot mismatch")
	}
}

func TestProcessor_AttemptCommand(t *testing.T) {
	templateId := uint32(5000017)
	commandId := byte(1)

	dp := &pdm.Processor{}
	dp.GetByIdFn = func(petId uint32) (data2.Model, error) {
		return data2.NewModelBuilder().
			AddSkill(data2.NewSkillModelBuilder().
				SetId(fmt.Sprintf("%d-%d", templateId, commandId)).
				Build()).
			Build(), nil
	}
	p := pet.NewProcessor(testLogger(), testContext(), testDatabase(t)).With(pet.WithDataProcessor(dp))

	// test setup
	i, err := p.Create(message.NewBuffer())(mustBuild(t, pet.NewModelBuilder(0, 7000000, templateId, "Mr. Roboto 1", 1).SetSlot(0)))
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}

	mb := message.NewBuffer()
	err = p.AttemptCommand(mb)(i.Id())(i.OwnerId())(commandId)
	if err != nil {
		t.Fatalf("Failed to execute command: %v", err)
	}
	ke := mb.GetAll()
	var se []kafka.Message
	var ok bool
	if se, ok = ke[pet2.EnvStatusEventTopic]; !ok {
		t.Fatalf("Failed to get events from topic: %s", pet2.EnvStatusEventTopic)
	}
	if len(se) != 1 {
		t.Fatalf("Failed to expected events from topic: %s", pet2.EnvStatusEventTopic)
	}
}

func TestProcessor_EvaluateHungerSpawned(t *testing.T) {
	dp := &pdm.Processor{}
	dp.GetByIdFn = func(petId uint32) (data2.Model, error) {
		return data2.NewModelBuilder().SetHunger(5).Build(), nil
	}
	p := pet.NewProcessor(testLogger(), testContext(), testDatabase(t)).With(pet.WithDataProcessor(dp))

	// test setup
	p1, err := p.Create(message.NewBuffer())(mustBuild(t, pet.NewModelBuilder(0, 7000000, 5000017, "Mr. Roboto 1", 1).SetSlot(0).SetFullness(100)))
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}
	p2, err := p.Create(message.NewBuffer())(mustBuild(t, pet.NewModelBuilder(0, 7000001, 5000017, "Mr. Roboto 2", 1).SetSlot(1).SetFullness(50)))
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}
	p3, err := p.Create(message.NewBuffer())(mustBuild(t, pet.NewModelBuilder(0, 7000002, 5000017, "Mr. Roboto 3", 1).SetSlot(-1).SetFullness(32)))
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}

	// test execution
	mb := message.NewBuffer()
	err = p.EvaluateHunger(mb)(p1.OwnerId())
	if err != nil {
		t.Fatalf("Failed to process hunger: %v", err)
	}
	ke := mb.GetAll()
	var se []kafka.Message
	var ok bool
	if se, ok = ke[pet2.EnvStatusEventTopic]; !ok {
		t.Fatalf("Failed to get events from topic: %s", pet2.EnvStatusEventTopic)
	}
	if len(se) != 2 {
		t.Fatalf("Failed to expected events from topic: %s", pet2.EnvStatusEventTopic)
	}

	o1, err := p.GetById(p1.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve pet when it should exist")
	}
	if o1.Fullness() != 95 {
		t.Fatalf("Failed to process hunger. Fullness mismatch")
	}

	o2, err := p.GetById(p2.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve pet when it should exist")
	}
	if o2.Fullness() != 45 {
		t.Fatalf("Failed to process hunger. Fullness mismatch")
	}

	o3, err := p.GetById(p3.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve pet when it should exist")
	}
	if o3.Fullness() != 32 {
		t.Fatalf("Failed to process hunger. Fullness mismatch")
	}
}

func TestProcessor_EvaluateHungerSunny(t *testing.T) {
	dp := &pdm.Processor{}
	dp.GetByIdFn = func(petId uint32) (data2.Model, error) {
		return data2.NewModelBuilder().SetHunger(5).Build(), nil
	}
	p := pet.NewProcessor(testLogger(), testContext(), testDatabase(t)).With(pet.WithDataProcessor(dp))

	// test setup
	p1, err := p.Create(message.NewBuffer())(mustBuild(t, pet.NewModelBuilder(0, 7000000, 5000017, "Mr. Roboto 1", 1).SetSlot(0).SetFullness(100)))
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}
	p2, err := p.Create(message.NewBuffer())(mustBuild(t, pet.NewModelBuilder(0, 7000001, 5000017, "Mr. Roboto 2", 1).SetSlot(1).SetFullness(50)))
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}
	p3, err := p.Create(message.NewBuffer())(mustBuild(t, pet.NewModelBuilder(0, 7000002, 5000017, "Mr. Roboto 3", 1).SetSlot(2).SetFullness(32)))
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}

	// test execution
	mb := message.NewBuffer()
	err = p.EvaluateHunger(mb)(p1.OwnerId())
	if err != nil {
		t.Fatalf("Failed to process hunger: %v", err)
	}
	ke := mb.GetAll()
	var se []kafka.Message
	var ok bool
	if se, ok = ke[pet2.EnvStatusEventTopic]; !ok {
		t.Fatalf("Failed to get events from topic: %s", pet2.EnvStatusEventTopic)
	}
	if len(se) != 3 {
		t.Fatalf("Failed to expected events from topic: %s", pet2.EnvStatusEventTopic)
	}

	o1, err := p.GetById(p1.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve pet when it should exist")
	}
	if o1.Fullness() != 95 {
		t.Fatalf("Failed to process hunger. Fullness mismatch")
	}

	o2, err := p.GetById(p2.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve pet when it should exist")
	}
	if o2.Fullness() != 45 {
		t.Fatalf("Failed to process hunger. Fullness mismatch")
	}

	o3, err := p.GetById(p3.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve pet when it should exist")
	}
	if o3.Fullness() != 27 {
		t.Fatalf("Failed to process hunger. Fullness mismatch")
	}
}

func TestProcessor_EvaluateHungerDespawn(t *testing.T) {
	dp := &pdm.Processor{}
	dp.GetByIdFn = func(petId uint32) (data2.Model, error) {
		return data2.NewModelBuilder().SetHunger(5).Build(), nil
	}
	p := pet.NewProcessor(testLogger(), testContext(), testDatabase(t)).With(pet.WithDataProcessor(dp))

	// test setup
	p1, err := p.Create(message.NewBuffer())(mustBuild(t, pet.NewModelBuilder(0, 7000000, 5000017, "Mr. Roboto 1", 1).SetSlot(0).SetFullness(100)))
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}
	p2, err := p.Create(message.NewBuffer())(mustBuild(t, pet.NewModelBuilder(0, 7000001, 5000017, "Mr. Roboto 2", 1).SetSlot(1).SetFullness(50)))
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}
	p3, err := p.Create(message.NewBuffer())(mustBuild(t, pet.NewModelBuilder(0, 7000002, 5000017, "Mr. Roboto 3", 1).SetSlot(2).SetFullness(7)))
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}

	despawned := false
	p.Despawner = func(mb *message.Buffer) func(petId uint32) func(actorId uint32) func(reason string) error {
		return func(petId uint32) func(actorId uint32) func(reason string) error {
			return func(actorId uint32) func(reason string) error {
				return func(reason string) error {
					if petId == p3.Id() {
						despawned = true
					}
					return nil
				}
			}
		}
	}

	// test execution
	mb := message.NewBuffer()
	err = p.EvaluateHunger(mb)(p1.OwnerId())
	if err != nil {
		t.Fatalf("Failed to process hunger: %v", err)
	}
	ke := mb.GetAll()
	var se []kafka.Message
	var ok bool
	if se, ok = ke[pet2.EnvStatusEventTopic]; !ok {
		t.Fatalf("Failed to get events from topic: %s", pet2.EnvStatusEventTopic)
	}
	if len(se) != 3 {
		t.Fatalf("Failed to expected events from topic: %s", pet2.EnvStatusEventTopic)
	}

	o1, err := p.GetById(p1.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve pet when it should exist")
	}
	if o1.Fullness() != 95 {
		t.Fatalf("Failed to process hunger. Fullness mismatch")
	}

	o2, err := p.GetById(p2.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve pet when it should exist")
	}
	if o2.Fullness() != 45 {
		t.Fatalf("Failed to process hunger. Fullness mismatch")
	}

	o3, err := p.GetById(p3.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve pet when it should exist")
	}
	if o3.Fullness() != 2 {
		t.Fatalf("Failed to process hunger. Fullness mismatch")
	}
	if !despawned {
		t.Fatalf("Should have despawned")
	}
}

func TestProcessor_AwardClosenessNonLevel(t *testing.T) {
	p := pet.NewProcessor(testLogger(), testContext(), testDatabase(t))

	// test setup
	p1, err := p.Create(message.NewBuffer())(mustBuild(t, pet.NewModelBuilder(0, 7000000, 5000017, "Mr. Roboto 1", 1).SetLevel(10).SetSlot(0).SetFullness(100)))
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}

	mb := message.NewBuffer()
	err = p.AwardCloseness(mb)(p1.Id())(1)
	if err != nil {
		t.Fatalf("Failed to award closeness: %v", err)
	}
	ke := mb.GetAll()
	var se []kafka.Message
	var ok bool
	if se, ok = ke[pet2.EnvStatusEventTopic]; !ok {
		t.Fatalf("Failed to get events from topic: %s", pet2.EnvStatusEventTopic)
	}
	if len(se) != 1 {
		t.Fatalf("Failed to expected events from topic: %s", pet2.EnvStatusEventTopic)
	}

	o1, err := p.GetById(p1.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve pet when it should exist")
	}
	if o1.Closeness() != 1 {
		t.Fatalf("Failed to process closeness. Closeness mismatch")
	}
}

func TestProcessor_AwardClosenessLevelSingle(t *testing.T) {
	p := pet.NewProcessor(testLogger(), testContext(), testDatabase(t))

	// test setup
	p1, err := p.Create(message.NewBuffer())(mustBuild(t, pet.NewModelBuilder(0, 7000000, 5000017, "Mr. Roboto 1", 1).SetLevel(1).SetSlot(0).SetFullness(100)))
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}

	mb := message.NewBuffer()
	err = p.AwardCloseness(mb)(p1.Id())(1)
	if err != nil {
		t.Fatalf("Failed to award closeness: %v", err)
	}
	ke := mb.GetAll()
	var se []kafka.Message
	var ok bool
	if se, ok = ke[pet2.EnvStatusEventTopic]; !ok {
		t.Fatalf("Failed to get events from topic: %s", pet2.EnvStatusEventTopic)
	}
	if len(se) != 2 {
		t.Fatalf("Failed to expected events from topic: %s", pet2.EnvStatusEventTopic)
	}

	o1, err := p.GetById(p1.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve pet when it should exist")
	}
	if o1.Closeness() != 1 {
		t.Fatalf("Failed to process closeness. Closeness mismatch")
	}
	if o1.Level() != 2 {
		t.Fatalf("Failed to process closeness. Closeness mismatch")
	}
}

func TestProcessor_AwardClosenessLevelMultiple(t *testing.T) {
	p := pet.NewProcessor(testLogger(), testContext(), testDatabase(t))

	// test setup
	p1, err := p.Create(message.NewBuffer())(mustBuild(t, pet.NewModelBuilder(0, 7000000, 5000017, "Mr. Roboto 1", 1).SetLevel(1).SetSlot(0).SetFullness(100)))
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}

	mb := message.NewBuffer()
	err = p.AwardCloseness(mb)(p1.Id())(6)
	if err != nil {
		t.Fatalf("Failed to award closeness: %v", err)
	}
	ke := mb.GetAll()
	var se []kafka.Message
	var ok bool
	if se, ok = ke[pet2.EnvStatusEventTopic]; !ok {
		t.Fatalf("Failed to get events from topic: %s", pet2.EnvStatusEventTopic)
	}
	if len(se) != 2 {
		t.Fatalf("Failed to expected events from topic: %s", pet2.EnvStatusEventTopic)
	}

	o1, err := p.GetById(p1.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve pet when it should exist")
	}
	if o1.Closeness() != 6 {
		t.Fatalf("Failed to process closeness. Closeness mismatch")
	}
	if o1.Level() != 4 {
		t.Fatalf("Failed to process closeness. Closeness mismatch")
	}
}

func TestProcessor_AwardClosenessCapacity(t *testing.T) {
	p := pet.NewProcessor(testLogger(), testContext(), testDatabase(t))

	// test setup
	p1, err := p.Create(message.NewBuffer())(mustBuild(t, pet.NewModelBuilder(0, 7000000, 5000017, "Mr. Roboto 1", 1).SetLevel(30).SetCloseness(30000).SetSlot(0).SetFullness(100)))
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}

	mb := message.NewBuffer()
	err = p.AwardCloseness(mb)(p1.Id())(6)
	if err != nil {
		t.Fatalf("Failed to award closeness: %v", err)
	}
	ke := mb.GetAll()
	var se []kafka.Message
	var ok bool
	if se, ok = ke[pet2.EnvStatusEventTopic]; !ok {
		t.Fatalf("Failed to get events from topic: %s", pet2.EnvStatusEventTopic)
	}
	if len(se) != 1 {
		t.Fatalf("Failed to expected events from topic: %s", pet2.EnvStatusEventTopic)
	}

	o1, err := p.GetById(p1.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve pet when it should exist")
	}
	if o1.Closeness() != 30000 {
		t.Fatalf("Failed to process closeness. Closeness mismatch")
	}
	if o1.Level() != 30 {
		t.Fatalf("Failed to process closeness. Closeness mismatch")
	}
}

func TestProcessor_AwardFullness(t *testing.T) {
	p := pet.NewProcessor(testLogger(), testContext(), testDatabase(t))

	// test setup
	p1, err := p.Create(message.NewBuffer())(mustBuild(t, pet.NewModelBuilder(0, 7000000, 5000017, "Mr. Roboto 1", 1).SetFullness(50)))
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}

	mb := message.NewBuffer()
	err = p.AwardFullness(mb)(p1.Id())(6)
	if err != nil {
		t.Fatalf("Failed to award fullness: %v", err)
	}
	ke := mb.GetAll()
	var se []kafka.Message
	var ok bool
	if se, ok = ke[pet2.EnvStatusEventTopic]; !ok {
		t.Fatalf("Failed to get events from topic: %s", pet2.EnvStatusEventTopic)
	}
	if len(se) != 1 {
		t.Fatalf("Failed to expected events from topic: %s", pet2.EnvStatusEventTopic)
	}

	o1, err := p.GetById(p1.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve pet when it should exist")
	}
	if o1.Fullness() != 56 {
		t.Fatalf("Failed to process fullness. Fullness mismatch")
	}
}

func TestProcessor_AwardFullnessMax(t *testing.T) {
	p := pet.NewProcessor(testLogger(), testContext(), testDatabase(t))

	// test setup
	p1, err := p.Create(message.NewBuffer())(mustBuild(t, pet.NewModelBuilder(0, 7000000, 5000017, "Mr. Roboto 1", 1).SetFullness(50)))
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}

	mb := message.NewBuffer()
	err = p.AwardFullness(mb)(p1.Id())(100)
	if err != nil {
		t.Fatalf("Failed to award fullness: %v", err)
	}
	ke := mb.GetAll()
	var se []kafka.Message
	var ok bool
	if se, ok = ke[pet2.EnvStatusEventTopic]; !ok {
		t.Fatalf("Failed to get events from topic: %s", pet2.EnvStatusEventTopic)
	}
	if len(se) != 1 {
		t.Fatalf("Failed to expected events from topic: %s", pet2.EnvStatusEventTopic)
	}

	o1, err := p.GetById(p1.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve pet when it should exist")
	}
	if o1.Fullness() != 100 {
		t.Fatalf("Failed to process fullness. Fullness mismatch")
	}
}

func TestProcessor_AwardLevel(t *testing.T) {
	p := pet.NewProcessor(testLogger(), testContext(), testDatabase(t))

	// test setup
	p1, err := p.Create(message.NewBuffer())(mustBuild(t, pet.NewModelBuilder(0, 7000000, 5000017, "Mr. Roboto 1", 1).SetLevel(1)))
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}

	mb := message.NewBuffer()
	err = p.AwardLevel(mb)(p1.Id())(1)
	if err != nil {
		t.Fatalf("Failed to award level: %v", err)
	}
	ke := mb.GetAll()
	var se []kafka.Message
	var ok bool
	if se, ok = ke[pet2.EnvStatusEventTopic]; !ok {
		t.Fatalf("Failed to get events from topic: %s", pet2.EnvStatusEventTopic)
	}
	if len(se) != 1 {
		t.Fatalf("Failed to expected events from topic: %s", pet2.EnvStatusEventTopic)
	}

	o1, err := p.GetById(p1.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve pet when it should exist")
	}
	if o1.Level() != 2 {
		t.Fatalf("Failed to process level. Level mismatch")
	}
}

func TestProcessor_AwardLevelMax(t *testing.T) {
	p := pet.NewProcessor(testLogger(), testContext(), testDatabase(t))

	// test setup
	p1, err := p.Create(message.NewBuffer())(mustBuild(t, pet.NewModelBuilder(0, 7000000, 5000017, "Mr. Roboto 1", 1).SetLevel(28)))
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}

	mb := message.NewBuffer()
	err = p.AwardLevel(mb)(p1.Id())(3)
	if err != nil {
		t.Fatalf("Failed to award level: %v", err)
	}
	ke := mb.GetAll()
	var se []kafka.Message
	var ok bool
	if se, ok = ke[pet2.EnvStatusEventTopic]; !ok {
		t.Fatalf("Failed to get events from topic: %s", pet2.EnvStatusEventTopic)
	}
	if len(se) != 1 {
		t.Fatalf("Failed to expected events from topic: %s", pet2.EnvStatusEventTopic)
	}

	o1, err := p.GetById(p1.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve pet when it should exist")
	}
	if o1.Level() != 30 {
		t.Fatalf("Failed to process level. Level mismatch")
	}
}

func TestProcessor_SetExclude(t *testing.T) {
	p := pet.NewProcessor(testLogger(), testContext(), testDatabase(t))

	// test setup
	p1, err := p.Create(message.NewBuffer())(mustBuild(t, pet.NewModelBuilder(0, 7000000, 5000017, "Mr. Roboto 1", 1)))
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}

	mb := message.NewBuffer()
	err = p.SetExclude(mb)(p1.Id())([]uint32{2059000, 2060000, 2061000})
	if err != nil {
		t.Fatalf("Failed to set exclude: %v", err)
	}
	ke := mb.GetAll()
	var se []kafka.Message
	var ok bool
	if se, ok = ke[pet2.EnvStatusEventTopic]; !ok {
		t.Fatalf("Failed to get events from topic: %s", pet2.EnvStatusEventTopic)
	}
	if len(se) != 1 {
		t.Fatalf("Failed to expected events from topic: %s", pet2.EnvStatusEventTopic)
	}

	o1, err := p.GetById(p1.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve pet when it should exist")
	}
	if len(o1.Excludes()) != 3 {
		t.Fatalf("Failed to expected excludes for pet. Length mismatch")
	}
}
