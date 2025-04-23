package pet

import (
	"atlas-pets/character"
	data2 "atlas-pets/data/pet"
	pet2 "atlas-pets/kafka/message/pet"
	"atlas-pets/kafka/producer"
	"atlas-pets/pet/position"
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sort"

	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var petExpTable = []uint16{1, 1, 3, 6, 14, 31, 60, 108, 181, 287, 434, 632, 891, 1224, 1642, 2161, 2793, 3557, 4467, 5542, 6801, 8263, 9950, 11882, 14084, 16578, 19391, 22547, 26074, 30000}

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	t   tenant.Model
	cp  *character.Processor
	pp  *position.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) *Processor {
	p := &Processor{
		l:   l,
		ctx: ctx,
		db:  db,
		t:   tenant.MustFromContext(ctx),
		cp:  character.NewProcessor(l, ctx),
		pp:  position.NewProcessor(l, ctx),
	}
	return p
}

func (p *Processor) WithTransaction(db *gorm.DB) *Processor {
	return &Processor{
		l:   p.l,
		ctx: p.ctx,
		db:  db,
	}
}

func (p *Processor) ByIdProvider(petId uint32) model.Provider[Model] {
	return model.Map(Make)(getById(p.t.Id(), petId)(p.db))
}

func (p *Processor) GetById(petId uint32) (Model, error) {
	return p.ByIdProvider(petId)()
}

func (p *Processor) ByOwnerProvider(ownerId uint32) model.Provider[[]Model] {
	return model.SliceMap(Make)(getByOwnerId(p.t.Id(), ownerId)(p.db))(model.ParallelMap())
}

func (p *Processor) GetByOwner(ownerId uint32) ([]Model, error) {
	return p.ByOwnerProvider(ownerId)()
}

func (p *Processor) SpawnedByOwnerProvider(ownerId uint32) model.Provider[[]Model] {
	return model.FilteredProvider(p.ByOwnerProvider(ownerId), model.Filters[Model](Spawned))
}

func Spawned(m Model) bool {
	return m.Slot() >= 0
}

func (p *Processor) HungryByOwnerProvider(ownerId uint32) model.Provider[[]Model] {
	return model.FilteredProvider(p.SpawnedByOwnerProvider(ownerId), model.Filters[Model](Hungry))
}

func Hungry(m Model) bool {
	return m.Fullness() < 100
}

func (p *Processor) HungriestByOwnerProvider(ownerId uint32) model.Provider[Model] {
	ps, err := p.HungryByOwnerProvider(ownerId)()
	if err != nil {
		return model.ErrorProvider[Model](err)
	}
	if len(ps) == 0 {
		return model.ErrorProvider[Model](errors.New("empty slice"))
	}

	sort.Slice(ps, func(i, j int) bool {
		return ps[i].Fullness() < ps[j].Fullness()
	})
	return model.FixedProvider(ps[0])
}

func (p *Processor) Create(i Model) (Model, error) {
	p.l.Debugf("Attempting to create pet from template [%d] for character [%d].", i.TemplateId(), i.OwnerId())
	// TODO this needs to generate a cashId if cashId == 0
	var om Model
	txErr := p.db.Transaction(func(tx *gorm.DB) error {
		b := Clone(i)
		if i.Level() < 1 || i.Level() > 30 {
			b.SetLevel(1)
		}
		if i.Closeness() < 0 {
			b.SetCloseness(0)
		}
		if i.Fullness() < 0 || i.Fullness() > 100 {
			b.SetFullness(100)
		}
		if i.Slot() < -1 || i.Slot() > 2 {
			b.SetSlot(-1)
		}
		i = b.Build()
		var err error
		om, err = create(tx)(p.t, i.OwnerId(), i)
		if err != nil {
			return err
		}
		return nil
	})
	if txErr != nil {
		p.l.WithError(txErr).Errorf("Unable to create pet from template [%d] for character [%d].", i.TemplateId(), i.OwnerId())
		return om, txErr
	}
	p.l.Debugf("Created pet [%d] for character [%d].", om.Id(), om.OwnerId())
	return om, nil
}

func (p *Processor) DeleteOnRemove(characterId uint32, itemId uint32, slot int16) error {
	c, err := p.cp.GetById(p.cp.InventoryDecorator)(characterId)
	if err != nil {
		return err
	}
	a, ok := c.Inventory().Cash().FindBySlot(slot)
	if !ok {
		return errors.New("pet not found")
	}
	if a.TemplateId() != itemId {
		return errors.New("item mismatch")
	}

	var om Model
	txErr := p.db.Transaction(deleteById(p.t, a.ReferenceId()))
	if txErr != nil {
		return txErr
	}
	p.l.Debugf("Deleted pet [%d] for character [%d].", om.Id(), characterId)
	return nil
}

func (p *Processor) DeleteForCharacter(characterId uint32) error {
	p.l.Debugf("Deleting all pets for character [%d], because the character has been deleted.", characterId)
	return p.db.Transaction(deleteForCharacter(p.t, characterId))
}

func (p *Processor) Move(petId uint32, m _map.Model, ownerId uint32, x int16, y int16, stance byte) error {
	pe, err := p.GetById(petId)
	if err != nil {
		p.l.WithError(err).Errorf("Movement issued for pet by character [%d], which pet [%d] does not exist.", ownerId, petId)
		return err
	}
	if pe.OwnerId() != ownerId {
		p.l.WithError(err).Errorf("Character [%d] attempting to move other character [%d] pet [%d].", ownerId, pe.OwnerId(), petId)
		return errors.New("pet not owned by character")
	}

	fh, err := p.pp.GetBelow(uint32(m.MapId()), x, y)()
	if err != nil {
		return err
	}
	GetTemporalRegistry().Update(petId, x, y, stance, int16(fh.Id()))
	return nil
}

func (p *Processor) Spawn(petId uint32, actorId uint32, lead bool) error {
	var pe Model
	slotEvents := model.FixedProvider[[]kafka.Message]([]kafka.Message{})
	txErr := p.db.Transaction(func(tx *gorm.DB) error {
		var err error
		pe, err = p.WithTransaction(tx).GetById(petId)
		if err != nil {
			return err
		}
		if pe.OwnerId() != actorId {
			return errors.New("pet not owned by character")
		}

		p.l.Debugf("Attempting to spawn [%d] for character [%d].", petId, actorId)

		sps, err := p.WithTransaction(tx).SpawnedByOwnerProvider(actorId)()
		if err != nil {
			return err
		}

		newSlot := int8(-1)
		if lead {
			p.l.Debugf("Pet [%d] will be the new leader.", petId)
			if len(sps) >= 3 {
				return errors.New("too many spawned pets")
			}
			for _, sp := range sps {
				oldSlot := sp.Slot()
				newSlot := oldSlot + 1
				p.l.Debugf("Attempting to move existing spawned pet [%d] from [%d] to [%d].", sp.Id(), oldSlot, newSlot)
				err = updateSlot(tx)(p.t, sp.Id(), newSlot)
				if err != nil {
					return err
				}
				sp = sp.SetSlot(newSlot)
				slotEvents = model.MergeSliceProvider(slotEvents, slotChangedEventProvider(sp, oldSlot))
			}
			newSlot = 0
		} else {
			p.l.Debugf("Finding minimal open slot for [%d].", petId)
			for i := int8(0); i < 3; i++ {
				found := false
				for _, sp := range sps {
					if sp.Slot() == i {
						found = true
						break
					}
				}
				if !found {
					newSlot = i
					break
				}
			}
		}
		oldSlot := pe.Slot()
		p.l.Debugf("Attempting to move pet [%d] to slot [%d].", petId, newSlot)
		err = updateSlot(tx)(p.t, petId, newSlot)
		if err != nil {
			return err
		}
		pe = pe.SetSlot(newSlot)
		slotEvents = model.MergeSliceProvider(slotEvents, slotChangedEventProvider(pe, oldSlot))
		return nil
	})
	if txErr != nil {
		return txErr
	}

	c, err := p.cp.GetById()(actorId)
	if err == nil {
		var fh position.Model
		fh, err = p.pp.GetBelow(c.MapId(), c.X(), c.Y())()
		if err == nil {
			GetTemporalRegistry().Update(petId, c.X(), c.Y(), 0, int16(fh.Id()))
		}
	}
	td := GetTemporalRegistry().GetById(pe.Id())

	err = producer.ProviderImpl(p.l)(p.ctx)(pet2.EnvStatusEventTopic)(spawnEventProvider(pe, td))
	if err != nil {
		p.l.WithError(err).Errorf("Unable to issue spawn events.")
	}

	err = producer.ProviderImpl(p.l)(p.ctx)(pet2.EnvStatusEventTopic)(slotEvents)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to issue slot change events.")
	}
	return err
}

func (p *Processor) Despawn(petId uint32, actorId uint32, reason string) error {
	var pe Model
	var oldSlot int8
	slotEvents := model.FixedProvider[[]kafka.Message]([]kafka.Message{})
	txErr := p.db.Transaction(func(tx *gorm.DB) error {
		var err error
		pe, err = p.WithTransaction(tx).GetById(petId)
		if err != nil {
			return err
		}
		if pe.OwnerId() != actorId {
			return errors.New("pet not owned by character")
		}

		p.l.Debugf("Attempting to despawn [%d] for character [%d].", petId, actorId)

		sps, err := p.SpawnedByOwnerProvider(actorId)()
		if err != nil {
			return err
		}

		p.l.Debugf("Shifting pets to the left.")
		for i := pe.Slot() + 1; i < 3; i++ {
			for _, sp := range sps {
				if sp.Slot() == i {
					oldSlot := sp.Slot()
					newSlot := oldSlot - 1

					err = updateSlot(tx)(p.t, sp.Id(), newSlot)
					if err != nil {
						return err
					}

					sp = sp.SetSlot(newSlot)
					slotEvents = model.MergeSliceProvider(slotEvents, slotChangedEventProvider(sp, oldSlot))
				}
			}
		}

		oldSlot = pe.Slot()
		newSlot := int8(-1)
		err = updateSlot(tx)(p.t, petId, newSlot)
		if err != nil {
			return err
		}
		pe = pe.SetSlot(newSlot)
		slotEvents = model.MergeSliceProvider(slotEvents, slotChangedEventProvider(pe, oldSlot))
		return nil
	})
	if txErr != nil {
		return txErr
	}

	var err error
	err = producer.ProviderImpl(p.l)(p.ctx)(pet2.EnvStatusEventTopic)(despawnEventProvider(pe, oldSlot, reason))
	if err != nil {
		p.l.WithError(err).Errorf("Unable to issue despawn events.")
	}

	err = producer.ProviderImpl(p.l)(p.ctx)(pet2.EnvStatusEventTopic)(slotEvents)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to issue slot change events.")
	}
	return err
}

func (p *Processor) AttemptCommand(petId uint32, actorId uint32, commandId byte, byName bool) error {
	var success bool
	var pe Model
	txErr := p.db.Transaction(func(tx *gorm.DB) error {
		var err error
		pe, err = p.WithTransaction(tx).GetById(petId)
		if err != nil {
			return err
		}
		if pe.OwnerId() != actorId {
			return errors.New("pet not owned by character")
		}
		if pe.Slot() < 0 {
			return errors.New("pet not active")
		}

		pdm, err := data2.GetById(p.l)(p.ctx)(pe.TemplateId())
		if err != nil {
			return err
		}
		var psm *data2.SkillModel
		psid := fmt.Sprintf("%d-%d", pe.TemplateId(), commandId)
		for _, rps := range pdm.Skills() {
			if rps.Id() == psid {
				psm = &rps
				break
			}
		}
		if psm == nil {
			return errors.New("no such pet skill")
		}
		if rand.Intn(100) < int(psm.Probability()) {
			success = true
		} else {
			success = false
		}
		err = p.WithTransaction(tx).AwardCloseness(petId, psm.Increase(), actorId)
		if err != nil {
			return err
		}
		return nil
	})
	if txErr != nil {
		return txErr
	}
	return producer.ProviderImpl(p.l)(p.ctx)(pet2.EnvStatusEventTopic)(commandResponseEventProvider(pe, commandId, success))
}

func (p *Processor) EvaluateHunger(ownerId uint32) error {
	original := make(map[uint32]Model)
	fullnessChanged := make([]Model, 0)
	despawned := make([]Model, 0)
	txErr := p.db.Transaction(func(tx *gorm.DB) error {
		ps, err := p.WithTransaction(tx).SpawnedByOwnerProvider(ownerId)()
		if err != nil {
			return err
		}
		for _, pe := range ps {
			original[pe.Id()] = pe

			var pdm data2.Model
			pdm, err = data2.GetById(p.l)(p.ctx)(pe.TemplateId())
			if err != nil {
				return err
			}
			newFullness := int16(pe.Fullness()) - int16(pdm.Hunger())
			if newFullness < 0 {
				newFullness = 0
			}
			err = updateFullness(tx)(p.t, pe.Id(), byte(newFullness))
			if err != nil {
				return err
			}
			if byte(newFullness) != pe.Fullness() {
				fullnessChanged = append(fullnessChanged, pe)
			}
			if newFullness <= 5 {
				despawned = append(despawned, pe)
			}
		}
		return nil
	})
	if txErr != nil {
		return txErr
	}
	for _, pe := range fullnessChanged {
		op := original[pe.Id()]
		change := int8(int16(op.Fullness()) - int16(pe.Fullness()))
		err := producer.ProviderImpl(p.l)(p.ctx)(pet2.EnvStatusEventTopic)(fullnessChangedEventProvider(pe, change))
		if err != nil {
			return err
		}
	}
	for _, pe := range despawned {
		err := p.WithTransaction(p.db).Despawn(pe.Id(), pe.OwnerId(), pet2.DespawnReasonHunger)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Processor) ClearPositions(ownerId uint32) error {
	return p.db.Transaction(func(tx *gorm.DB) error {
		ps, err := p.WithTransaction(tx).GetByOwner(ownerId)
		if err != nil {
			return err
		}
		for _, pe := range ps {
			GetTemporalRegistry().Remove(pe.Id())
		}
		return nil
	})
}

func (p *Processor) AwardCloseness(petId uint32, amount uint16, actorId uint32) error {
	var pe Model
	awardLevel := false
	txErr := p.db.Transaction(func(tx *gorm.DB) error {
		var err error
		pe, err = p.WithTransaction(tx).GetById(petId)
		if err != nil {
			return err
		}
		newCloseness := pe.Closeness() + amount
		level := pe.Level()

		if newCloseness >= petExpTable[pe.Level()] {
			if pe.Level() >= 30 {
				newCloseness = petExpTable[len(petExpTable)-1]
			} else {
				awardLevel = true
			}
		}
		err = updateCloseness(tx)(p.t, petId, newCloseness)
		if err != nil {
			return err
		}
		if awardLevel {
			level += 1
			err = updateLevel(tx)(p.t, petId, level)
			if err != nil {
				return err
			}
		}
		pe = Clone(pe).SetCloseness(newCloseness).SetLevel(level).Build()
		return nil
	})
	if txErr != nil {
		return txErr
	}
	err := producer.ProviderImpl(p.l)(p.ctx)(pet2.EnvStatusEventTopic)(closenessChangedEventProvider(pe, int16(amount)))
	if err != nil {
		return err
	}
	if awardLevel {
		err = producer.ProviderImpl(p.l)(p.ctx)(pet2.EnvStatusEventTopic)(levelChangedEventProvider(pe, 1))
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Processor) AwardFullness(petId uint32, amount byte, actorId uint32) error {
	var pe Model
	txErr := p.db.Transaction(func(tx *gorm.DB) error {
		var err error
		pe, err = p.WithTransaction(tx).GetById(petId)
		if err != nil {
			return err
		}
		newFullness := pe.Fullness() + amount
		if newFullness > 100 {
			newFullness = 100
		}
		err = updateFullness(tx)(p.t, petId, newFullness)
		if err != nil {
			return err
		}
		pe = Clone(pe).SetFullness(newFullness).Build()
		return nil
	})
	if txErr != nil {
		return txErr
	}
	err := producer.ProviderImpl(p.l)(p.ctx)(pet2.EnvStatusEventTopic)(fullnessChangedEventProvider(pe, int8(amount)))
	if err != nil {
		return err
	}
	return nil
}

func (p *Processor) AwardLevel(petId uint32, amount byte, actorId uint32) error {
	var pe Model
	txErr := p.db.Transaction(func(tx *gorm.DB) error {
		var err error
		pe, err = p.WithTransaction(tx).GetById(petId)
		if err != nil {
			return err
		}
		newLevel := pe.Level() + amount
		if newLevel > 30 {
			newLevel = 30
		}
		err = updateLevel(tx)(p.t, petId, newLevel)
		if err != nil {
			return err
		}
		pe = Clone(pe).SetLevel(newLevel).Build()
		return nil
	})
	if txErr != nil {
		return txErr
	}
	err := producer.ProviderImpl(p.l)(p.ctx)(pet2.EnvStatusEventTopic)(levelChangedEventProvider(pe, int8(amount)))
	if err != nil {
		return err
	}
	return nil
}

func (p *Processor) SetExclude(petId uint32, items []uint32) error {
	var pe Model
	txErr := p.db.Transaction(func(tx *gorm.DB) error {
		err := setExcludes(tx, petId, items)
		if err != nil {
			return err
		}

		pe, err = p.WithTransaction(tx).GetById(petId)
		if err != nil {
			return err
		}
		return nil
	})
	if txErr != nil {
		return txErr
	}
	err := producer.ProviderImpl(p.l)(p.ctx)(pet2.EnvStatusEventTopic)(excludeChangedEventProvider(pe))
	if err != nil {
		return err
	}
	return nil
}
