package pet

import (
	"atlas-pets/character"
	data2 "atlas-pets/data/pet"
	"atlas-pets/kafka/message"
	"atlas-pets/kafka/message/pet"
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
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var petExpTable = []uint16{1, 1, 3, 6, 14, 31, 60, 108, 181, 287, 434, 632, 891, 1224, 1642, 2161, 2793, 3557, 4467, 5542, 6801, 8263, 9950, 11882, 14084, 16578, 19391, 22547, 26074, 30000}

type Processor struct {
	l             logrus.FieldLogger
	ctx           context.Context
	db            *gorm.DB
	t             tenant.Model
	cp            *character.Processor
	pp            *position.Processor
	KafkaProducer producer.Provider
	GetById       func(petId uint32) (Model, error)
	GetByOwner    func(ownerId uint32) ([]Model, error)
	CreateAndEmit func(i Model) (Model, error)
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
	p.KafkaProducer = producer.ProviderImpl(l)(ctx)
	p.GetById = model.CollapseProvider(p.ByIdProvider)
	p.GetByOwner = model.CollapseProvider(p.ByOwnerProvider)
	p.CreateAndEmit = message.EmitWithResult[Model, Model](p.KafkaProducer)(p.Create)
	return p
}

func (p *Processor) WithTransaction(db *gorm.DB) *Processor {
	return &Processor{
		l:             p.l,
		ctx:           p.ctx,
		db:            db,
		t:             p.t,
		cp:            p.cp,
		pp:            p.pp,
		KafkaProducer: p.KafkaProducer,
		GetById:       p.GetById,
		GetByOwner:    p.GetByOwner,
		CreateAndEmit: p.CreateAndEmit,
	}
}

func (p *Processor) ByIdProvider(petId uint32) model.Provider[Model] {
	return model.Map(Make)(getById(p.t.Id(), petId)(p.db))
}

func (p *Processor) ByOwnerProvider(ownerId uint32) model.Provider[[]Model] {
	return model.SliceMap(Make)(getByOwnerId(p.t.Id(), ownerId)(p.db))(model.ParallelMap())
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

func (p *Processor) Create(mb *message.Buffer) func(i Model) (Model, error) {
	return func(i Model) (Model, error) {
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
			return mb.Put(pet.EnvStatusEventTopic, createdEventProvider(om))
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Unable to create pet from template [%d] for character [%d].", i.TemplateId(), i.OwnerId())
			return om, txErr
		}
		p.l.Debugf("Created pet [%d] for character [%d].", om.Id(), om.OwnerId())
		return om, nil
	}
}

func (p *Processor) DeleteOnRemoveAndEmit(characterId uint32, itemId uint32, slot int16) error {
	return message.Emit(p.KafkaProducer)(model.Flip(model.Flip(model.Flip(p.DeleteOnRemove)(characterId))(itemId))(slot))
}

func (p *Processor) DeleteOnRemove(mb *message.Buffer) func(characterId uint32) func(itemId uint32) func(slot int16) error {
	return func(characterId uint32) func(itemId uint32) func(slot int16) error {
		return func(itemId uint32) func(slot int16) error {
			return func(slot int16) error {
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
				return p.Delete(mb)(a.ReferenceId())(characterId)
			}
		}
	}
}

func (p *Processor) DeleteForCharacterAndEmit(characterId uint32) error {
	return message.Emit(p.KafkaProducer)(model.Flip(p.DeleteForCharacter)(characterId))
}

func (p *Processor) DeleteForCharacter(mb *message.Buffer) func(characterId uint32) error {
	return func(characterId uint32) error {
		p.l.Debugf("Attempting to delete all pets for character [%d].", characterId)
		txErr := p.db.Transaction(func(tx *gorm.DB) error {
			ps, err := p.GetByOwner(characterId)
			if err != nil {
				return err
			}
			for _, pm := range ps {
				err = p.Delete(mb)(pm.Id())(pm.OwnerId())
				if err != nil {
					return err
				}
			}
			return nil
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Unable to delete all pets for character [%d].", characterId)
			return txErr
		}
		p.l.Debugf("Deleted all pets for character [%d].", characterId)
		return nil
	}
}

func (p *Processor) Delete(mb *message.Buffer) func(petId uint32) func(ownerId uint32) error {
	return func(petId uint32) func(ownerId uint32) error {
		return func(ownerId uint32) error {
			p.l.Debugf("Attempting to delete pet [%d].", petId)
			txErr := p.db.Transaction(func(tx *gorm.DB) error {
				err := deleteById(p.t, petId)(tx)
				if err != nil {
					return err
				}
				return mb.Put(pet.EnvStatusEventTopic, deletedEventProvider(petId, ownerId))
			})
			if txErr != nil {
				p.l.WithError(txErr).Errorf("Unable to delete pet [%d].", petId)
				return txErr
			}
			p.l.Debugf("Deleted pet [%d].", petId)
			return nil
		}
	}
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
	p.l.Debugf("Recording pet [%d] movement. x [%d], y [%d], fh [%d].", petId, x, y, fh)
	GetTemporalRegistry().Update(petId, x, y, stance, int16(fh.Id()))
	return nil
}

func (p *Processor) SpawnAndEmit(petId uint32, actorId uint32, lead bool) error {
	return message.Emit(p.KafkaProducer)(model.Flip(model.Flip(model.Flip(p.Spawn)(petId))(actorId))(lead))
}

func (p *Processor) Spawn(mb *message.Buffer) func(petId uint32) func(actorId uint32) func(lead bool) error {
	return func(petId uint32) func(actorId uint32) func(lead bool) error {
		return func(actorId uint32) func(lead bool) error {
			return func(lead bool) error {
				p.l.Debugf("Spawning pet [%d] for character [%d]", petId, actorId)
				txErr := p.db.Transaction(func(tx *gorm.DB) error {
					pe, err := p.WithTransaction(tx).GetById(petId)
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
							err = mb.Put(pet.EnvStatusEventTopic, slotChangedEventProvider(sp, oldSlot))
							if err != nil {
								return err
							}
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
					err = mb.Put(pet.EnvStatusEventTopic, slotChangedEventProvider(pe, oldSlot))
					if err != nil {
						return err
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
					return mb.Put(pet.EnvStatusEventTopic, spawnEventProvider(pe, td))
				})
				if txErr != nil {
					p.l.WithError(txErr).Errorf("Unable to spawn pet from character [%d].", petId)
					return txErr
				}
				p.l.Debugf("Spawned pet [%d] for character [%d].", petId, actorId)
				return nil
			}
		}
	}
}

func (p *Processor) DespawnAndEmit(petId uint32, actorId uint32, reason string) error {
	return message.Emit(p.KafkaProducer)(model.Flip(model.Flip(model.Flip(p.Despawn)(petId))(actorId))(reason))
}

func (p *Processor) Despawn(mb *message.Buffer) func(petId uint32) func(actorId uint32) func(reason string) error {
	return func(petId uint32) func(actorId uint32) func(reason string) error {
		return func(actorId uint32) func(reason string) error {
			return func(reason string) error {
				p.l.Debugf("Attempting to despawn pet [%d] for character [%d].", petId, actorId)
				txErr := p.db.Transaction(func(tx *gorm.DB) error {
					pe, err := p.WithTransaction(tx).GetById(petId)
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
								err = mb.Put(pet.EnvStatusEventTopic, slotChangedEventProvider(sp, oldSlot))
								if err != nil {
									return err
								}
							}
						}
					}

					oldSlot := pe.Slot()
					newSlot := int8(-1)
					err = updateSlot(tx)(p.t, petId, newSlot)
					if err != nil {
						return err
					}
					pe = pe.SetSlot(newSlot)
					err = mb.Put(pet.EnvStatusEventTopic, slotChangedEventProvider(pe, oldSlot))
					if err != nil {
						return err
					}
					return mb.Put(pet.EnvStatusEventTopic, despawnEventProvider(pe, oldSlot, reason))
				})
				if txErr != nil {
					p.l.WithError(txErr).Errorf("Unable to despawn pet for character [%d].", petId)
					return txErr
				}
				p.l.Debugf("Despawned pet [%d] for character [%d].", petId, actorId)
				return nil
			}
		}
	}
}

func (p *Processor) AttemptCommandAndEmit(petId uint32, actorId uint32, commandId byte) error {
	return message.Emit(p.KafkaProducer)(model.Flip(model.Flip(model.Flip(p.AttemptCommand)(petId))(actorId))(commandId))
}

func (p *Processor) AttemptCommand(mb *message.Buffer) func(petId uint32) func(actorId uint32) func(commandId byte) error {
	return func(petId uint32) func(actorId uint32) func(commandId byte) error {
		return func(actorId uint32) func(commandId byte) error {
			return func(commandId byte) error {
				p.l.Debugf("Attempting command [%d] for pet [%d].", commandId, petId)
				txErr := p.db.Transaction(func(tx *gorm.DB) error {
					pe, err := p.WithTransaction(tx).GetById(petId)
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
					success := false
					if rand.Intn(100) < int(psm.Probability()) {
						success = true
					}
					err = p.WithTransaction(tx).AwardClosenessAndEmit(petId, psm.Increase())
					if err != nil {
						return err
					}
					return mb.Put(pet.EnvStatusEventTopic, commandResponseEventProvider(pe, commandId, success))
				})
				if txErr != nil {
					p.l.WithError(txErr).Errorf("Unable to attempt command [%d] for pet [%d].", commandId, petId)
					return txErr
				}
				p.l.Debugf("Performed command [%d] for pet [%d].", commandId, petId)
				return nil
			}
		}
	}
}

func (p *Processor) EvaluateHungerAndEmit(ownerId uint32) error {
	return message.Emit(p.KafkaProducer)(model.Flip(p.EvaluateHunger)(ownerId))
}

func (p *Processor) EvaluateHunger(mb *message.Buffer) func(ownerId uint32) error {
	return func(ownerId uint32) error {
		p.l.Debugf("Evaluating hunger of pets for owner [%d].", ownerId)
		txErr := p.db.Transaction(func(tx *gorm.DB) error {
			ps, err := p.WithTransaction(tx).SpawnedByOwnerProvider(ownerId)()
			if err != nil {
				return err
			}
			for _, pe := range ps {
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
					err = mb.Put(pet.EnvStatusEventTopic, fullnessChangedEventProvider(pe, int8(int16(pe.Fullness())-newFullness)))
					if err != nil {
						return err
					}
				}
				if newFullness <= 5 {
					err = p.WithTransaction(tx).Despawn(mb)(pe.Id())(pe.OwnerId())(pet.DespawnReasonHunger)
					if err != nil {
						return err
					}
				}
			}
			return nil
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Unable to evaluate hunger of pets for owner [%d].", ownerId)
			return txErr
		}
		p.l.Debugf("Evaluated hunger of pets for owner [%d]", ownerId)
		return nil
	}
}

func (p *Processor) ClearPositions(ownerId uint32) error {
	p.l.Debugf("Clearing positions of pets for owner [%d].", ownerId)
	txErr := p.db.Transaction(func(tx *gorm.DB) error {
		ps, err := p.WithTransaction(tx).GetByOwner(ownerId)
		if err != nil {
			return err
		}
		for _, pe := range ps {
			GetTemporalRegistry().Remove(pe.Id())
		}
		return nil
	})
	if txErr != nil {
		p.l.WithError(txErr).Errorf("Unable to clear positions of pets for owner [%d].", ownerId)
		return txErr
	}
	p.l.Debugf("Cleared positions of pets for owner [%d].", ownerId)
	return nil
}

func (p *Processor) AwardClosenessAndEmit(petId uint32, amount uint16) error {
	return message.Emit(p.KafkaProducer)(model.Flip(model.Flip(p.AwardCloseness)(petId))(amount))
}

func (p *Processor) AwardCloseness(mb *message.Buffer) func(petId uint32) func(amount uint16) error {
	return func(petId uint32) func(amount uint16) error {
		return func(amount uint16) error {
			p.l.Debugf("Awarding [%d] closeness for pet [%d].", amount, petId)
			txErr := p.db.Transaction(func(tx *gorm.DB) error {
				pe, err := p.WithTransaction(tx).GetById(petId)
				if err != nil {
					return err
				}
				newCloseness := pe.Closeness() + amount
				level := pe.Level()

				awardLevel := false
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
					err = p.WithTransaction(tx).AwardLevel(mb)(pe.Id())(1)
					if err != nil {
						return err
					}
					level += 1
				}
				pe = Clone(pe).SetCloseness(newCloseness).SetLevel(level).Build()
				err = mb.Put(pet.EnvStatusEventTopic, closenessChangedEventProvider(pe, int16(amount)))
				if err != nil {
					return err
				}
				return nil
			})
			if txErr != nil {
				p.l.WithError(txErr).Errorf("Unable to award closeness [%d].", petId)
				return txErr
			}
			p.l.Debugf("Awarded [%d] closeness for pet [%d].", amount, petId)
			return nil
		}
	}
}

func (p *Processor) AwardFullnessAndEmit(petId uint32, amount byte) error {
	return message.Emit(p.KafkaProducer)(model.Flip(model.Flip(p.AwardFullness)(petId))(amount))
}

func (p *Processor) AwardFullness(mb *message.Buffer) func(petId uint32) func(amount byte) error {
	return func(petId uint32) func(amount byte) error {
		return func(amount byte) error {
			p.l.Debugf("Awarding [%d] fullness for pet [%d].", amount, petId)
			txErr := p.db.Transaction(func(tx *gorm.DB) error {
				pe, err := p.WithTransaction(tx).GetById(petId)
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
				return mb.Put(pet.EnvStatusEventTopic, fullnessChangedEventProvider(pe, int8(int16(amount))))
			})
			if txErr != nil {
				p.l.WithError(txErr).Errorf("Unable to award fullness to pet [%d].", petId)
				return txErr
			}
			p.l.Debugf("Awarded [%d] fullness for pet [%d].", amount, petId)
			return nil
		}
	}
}

func (p *Processor) AwardLevelAndEmit(petId uint32, amount byte) error {
	return message.Emit(p.KafkaProducer)(model.Flip(model.Flip(p.AwardLevel)(petId))(amount))
}

func (p *Processor) AwardLevel(mb *message.Buffer) func(petId uint32) func(amount byte) error {
	return func(petId uint32) func(amount byte) error {
		return func(amount byte) error {
			p.l.Debugf("Awarding [%d] level for pet [%d].", amount, petId)
			txErr := p.db.Transaction(func(tx *gorm.DB) error {
				pe, err := p.WithTransaction(tx).GetById(petId)
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
				return mb.Put(pet.EnvStatusEventTopic, levelChangedEventProvider(pe, int8(int16(amount))))
			})
			if txErr != nil {
				p.l.WithError(txErr).Errorf("Unable to award level to pet [%d].", petId)
				return txErr
			}
			p.l.Debugf("Awarded [%d] level for pet [%d].", amount, petId)
			return nil
		}
	}
}

func (p *Processor) SetExcludeAndEmit(petId uint32, items []uint32) error {
	return message.Emit(p.KafkaProducer)(model.Flip(model.Flip(p.SetExclude)(petId))(items))
}

func (p *Processor) SetExclude(mb *message.Buffer) func(petId uint32) func(items []uint32) error {
	return func(petId uint32) func(items []uint32) error {
		return func(items []uint32) error {
			p.l.Debugf("Attempting to set [%d] exclude items for pet [%d].", len(items), petId)
			txErr := p.db.Transaction(func(tx *gorm.DB) error {
				err := setExcludes(tx, petId, items)
				if err != nil {
					return err
				}

				pe, err := p.WithTransaction(tx).GetById(petId)
				if err != nil {
					return err
				}
				return mb.Put(pet.EnvStatusEventTopic, excludeChangedEventProvider(pe))
			})
			if txErr != nil {
				p.l.WithError(txErr).Errorf("Unable to set exclude items for pet [%d].", petId)
				return txErr
			}
			p.l.Debugf("Set exclude items for pet [%d].", petId)
			return nil
		}
	}
}
