package pet

import (
	"atlas-pets/character"
	data2 "atlas-pets/data/pet"
	"atlas-pets/data/position"
	"atlas-pets/database"
	"atlas-pets/kafka/message"
	"atlas-pets/kafka/message/pet"
	"atlas-pets/kafka/producer"
	"atlas-pets/skill"
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sort"

	"github.com/Chronicle20/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas-constants/skill"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var petExpTable = []uint16{1, 1, 3, 6, 14, 31, 60, 108, 181, 287, 434, 632, 891, 1224, 1642, 2161, 2793, 3557, 4467, 5542, 6801, 8263, 9950, 11882, 14084, 16578, 19391, 22547, 26074, 30000}

const (
	MaxFullness  = 100
	MaxLevel     = 30
	MaxCloseness = 30000
)

type Processor interface {
	With(opts ...ProcessorOption) *ProcessorImpl
	ByIdProvider(petId uint32) model.Provider[Model]
	GetById(petId uint32) (Model, error)
	ByOwnerProvider(ownerId uint32) model.Provider[[]Model]
	GetByOwner(ownerId uint32) ([]Model, error)
	SpawnedByOwnerProvider(ownerId uint32) model.Provider[[]Model]
	HungryByOwnerProvider(ownerId uint32) model.Provider[[]Model]
	HungriestByOwnerProvider(ownerId uint32) model.Provider[Model]
	CreateAndEmit(i Model) (Model, error)
	Create(mb *message.Buffer) func(i Model) (Model, error)
	DeleteOnRemoveAndEmit(characterId uint32, itemId uint32, slot int16) error
	DeleteOnRemove(mb *message.Buffer) func(characterId uint32) func(itemId uint32) func(slot int16) error
	DeleteForCharacterAndEmit(characterId uint32) error
	DeleteForCharacter(mb *message.Buffer) func(characterId uint32) error
	Delete(mb *message.Buffer) func(petId uint32) func(ownerId uint32) error
	Move(petId uint32, f field.Model, ownerId uint32, x int16, y int16, stance byte) error
	SpawnAndEmit(petId uint32, actorId uint32, lead bool) error
	Spawn(mb *message.Buffer) func(petId uint32) func(actorId uint32) func(lead bool) error
	DespawnAndEmit(petId uint32, actorId uint32, reason string) error
	Despawn(mb *message.Buffer) func(petId uint32) func(actorId uint32) func(reason string) error
	AttemptCommandAndEmit(petId uint32, actorId uint32, commandId byte) error
	AttemptCommand(mb *message.Buffer) func(petId uint32) func(actorId uint32) func(commandId byte) error
	EvaluateHungerAndEmit(ownerId uint32) error
	EvaluateHunger(mb *message.Buffer) func(ownerId uint32) error
	ClearPositions(ownerId uint32) error
	AwardClosenessAndEmit(petId uint32, amount uint16) error
	AwardCloseness(mb *message.Buffer) func(petId uint32) func(amount uint16) error
	AwardClosenessWithTransactionAndEmit(transactionId uuid.UUID, petId uint32, amount uint16) error
	AwardClosenessWithTransaction(mb *message.Buffer) func(transactionId uuid.UUID, petId uint32, amount uint16) error
	AwardFullnessAndEmit(petId uint32, amount byte) error
	AwardFullness(mb *message.Buffer) func(petId uint32) func(amount byte) error
	AwardLevelAndEmit(petId uint32, amount byte) error
	AwardLevel(mb *message.Buffer) func(petId uint32) func(amount byte) error
	SetExcludeAndEmit(petId uint32, items []uint32) error
	SetExclude(mb *message.Buffer) func(petId uint32) func(items []uint32) error
}

type ProcessorImpl struct {
	l         logrus.FieldLogger
	ctx       context.Context
	db        *gorm.DB
	t         tenant.Model
	tr        TemporalRegistry
	cp        character.Processor
	pp        position.Processor
	dp        data2.Processor
	sp        skill.Processor
	kp        producer.Provider
	Despawner func(mb *message.Buffer) func(petId uint32) func(actorId uint32) func(reason string) error
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
		t:   tenant.MustFromContext(ctx),
		tr:  GetTemporalRegistry(),
		cp:  character.NewProcessor(l, ctx),
		pp:  position.NewProcessor(l, ctx),
		dp:  data2.NewProcessor(l, ctx),
		sp:  skill.NewProcessor(l, ctx),
		kp:  producer.ProviderImpl(l)(ctx),
	}
	p.Despawner = p.defaultDespawn
	return p
}

type ProcessorOption func(*ProcessorImpl)

func WithTransaction(db *gorm.DB) ProcessorOption {
	return func(p *ProcessorImpl) {
		p.db = db
	}
}

func WithCharacterProcessor(cp character.Processor) ProcessorOption {
	return func(p *ProcessorImpl) {
		p.cp = cp
	}
}

func WithPositionProcessor(pp position.Processor) ProcessorOption {
	return func(p *ProcessorImpl) {
		p.pp = pp
	}
}

func WithDataProcessor(dp data2.Processor) ProcessorOption {
	return func(p *ProcessorImpl) {
		p.dp = dp
	}
}

func WithSkillProcessor(sp skill.Processor) ProcessorOption {
	return func(p *ProcessorImpl) {
		p.sp = sp
	}
}

func (p *ProcessorImpl) With(opts ...ProcessorOption) *ProcessorImpl {
	clone := *p
	cp := &clone
	for _, opt := range opts {
		opt(cp)
	}
	return cp
}

func (p *ProcessorImpl) ByIdProvider(petId uint32) model.Provider[Model] {
	return model.Map(Make)(getById(p.t.Id(), petId)(p.db))
}

func (p *ProcessorImpl) GetById(petId uint32) (Model, error) {
	return model.CollapseProvider(p.ByIdProvider)(petId)
}

func (p *ProcessorImpl) ByOwnerProvider(ownerId uint32) model.Provider[[]Model] {
	return model.SliceMap(Make)(getByOwnerId(p.t.Id(), ownerId)(p.db))(model.ParallelMap())
}

func (p *ProcessorImpl) GetByOwner(ownerId uint32) ([]Model, error) {
	return model.CollapseProvider(p.ByOwnerProvider)(ownerId)
}

func (p *ProcessorImpl) SpawnedByOwnerProvider(ownerId uint32) model.Provider[[]Model] {
	return model.FilteredProvider(p.ByOwnerProvider(ownerId), model.Filters[Model](Spawned))
}

func Spawned(m Model) bool {
	return m.Slot() >= 0
}

func (p *ProcessorImpl) HungryByOwnerProvider(ownerId uint32) model.Provider[[]Model] {
	return model.FilteredProvider(p.SpawnedByOwnerProvider(ownerId), model.Filters[Model](Hungry))
}

func Hungry(m Model) bool {
	return m.Fullness() < MaxFullness
}

func (p *ProcessorImpl) HungriestByOwnerProvider(ownerId uint32) model.Provider[Model] {
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

func (p *ProcessorImpl) CreateAndEmit(i Model) (Model, error) {
	return message.EmitWithResult[Model, Model](p.kp)(p.Create)(i)
}

func (p *ProcessorImpl) Create(mb *message.Buffer) func(i Model) (Model, error) {
	return func(i Model) (Model, error) {
		p.l.Debugf("Attempting to create pet from template [%d] for character [%d].", i.TemplateId(), i.OwnerId())
		// TODO this needs to generate a cashId if cashId == 0
		var om Model
		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
			b := Clone(i)
			if i.Level() < 1 || i.Level() > MaxLevel {
				b.SetLevel(1)
			}
			if i.Closeness() < 0 {
				b.SetCloseness(0)
			}
			if i.Fullness() < 0 || i.Fullness() > MaxFullness {
				b.SetFullness(MaxFullness)
			}
			if i.Slot() < -1 || i.Slot() > 2 {
				b.SetSlot(-1)
			}
			var err error
			i, err = b.Build()
			if err != nil {
				return err
			}
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
		p.l.Infof("Created pet [%d] for character [%d].", om.Id(), om.OwnerId())
		return om, nil
	}
}

func (p *ProcessorImpl) DeleteOnRemoveAndEmit(characterId uint32, itemId uint32, slot int16) error {
	return message.Emit(p.kp)(model.Flip(model.Flip(model.Flip(p.DeleteOnRemove)(characterId))(itemId))(slot))
}

func (p *ProcessorImpl) DeleteOnRemove(mb *message.Buffer) func(characterId uint32) func(itemId uint32) func(slot int16) error {
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

func (p *ProcessorImpl) DeleteForCharacterAndEmit(characterId uint32) error {
	return message.Emit(p.kp)(model.Flip(p.DeleteForCharacter)(characterId))
}

func (p *ProcessorImpl) DeleteForCharacter(mb *message.Buffer) func(characterId uint32) error {
	return func(characterId uint32) error {
		p.l.Debugf("Attempting to delete all pets for character [%d].", characterId)
		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
			ps, err := p.GetByOwner(characterId)
			if err != nil {
				return err
			}
			for _, pm := range ps {
				err = p.With(WithTransaction(tx)).Delete(mb)(pm.Id())(pm.OwnerId())
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
		p.l.Infof("Deleted all pets for character [%d].", characterId)
		return nil
	}
}

func (p *ProcessorImpl) Delete(mb *message.Buffer) func(petId uint32) func(ownerId uint32) error {
	return func(petId uint32) func(ownerId uint32) error {
		return func(ownerId uint32) error {
			p.l.Debugf("Attempting to delete pet [%d].", petId)
			txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
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
			p.l.Infof("Deleted pet [%d].", petId)
			return nil
		}
	}
}

func (p *ProcessorImpl) Move(petId uint32, f field.Model, ownerId uint32, x int16, y int16, stance byte) error {
	pe, err := p.GetById(petId)
	if err != nil {
		p.l.WithError(err).Errorf("Movement issued for pet by character [%d], which pet [%d] does not exist.", ownerId, petId)
		return err
	}
	if pe.OwnerId() != ownerId {
		p.l.WithError(err).Errorf("Character [%d] attempting to move other character [%d] pet [%d].", ownerId, pe.OwnerId(), petId)
		return errors.New("pet not owned by character")
	}

	fh, err := p.pp.GetBelow(f.MapId(), x, y)()
	if err != nil {
		return err
	}
	p.l.Infof("Recording pet [%d] movement. x [%d], y [%d], fh [%d].", petId, x, y, fh)
	p.tr.Update(petId, x, y, stance, int16(fh.Id()))
	return nil
}

func (p *ProcessorImpl) SpawnAndEmit(petId uint32, actorId uint32, lead bool) error {
	return message.Emit(p.kp)(model.Flip(model.Flip(model.Flip(p.Spawn)(petId))(actorId))(lead))
}

var ErrTooManySpawnedPets = errors.New("too many pets spawned")
var ErrNeedMultiPetSkill = errors.New("need multi pet skill")

func (p *ProcessorImpl) Spawn(mb *message.Buffer) func(petId uint32) func(actorId uint32) func(lead bool) error {
	return func(petId uint32) func(actorId uint32) func(lead bool) error {
		return func(actorId uint32) func(lead bool) error {
			return func(lead bool) error {
				p.l.Debugf("Spawning pet [%d] for character [%d]", petId, actorId)
				txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
					pe, err := p.With(WithTransaction(tx)).GetById(petId)
					if err != nil {
						return err
					}
					if pe.OwnerId() != actorId {
						return errors.New("pet not owned by character")
					}

					p.l.Debugf("Attempting to spawn [%d] for character [%d].", petId, actorId)

					sps, err := p.With(WithTransaction(tx)).SpawnedByOwnerProvider(actorId)()
					if err != nil {
						return err
					}

					newSlot := int8(-1)
					if lead {
						p.l.Debugf("Pet [%d] will be the new leader.", petId)
						if len(sps) >= 3 {
							return ErrTooManySpawnedPets
						}
						if len(sps) >= 1 {
							if !p.sp.HasSkill(actorId, skill2.BeginnerMultiPetId, skill2.NoblesseMultiPetId) {
								return ErrNeedMultiPetSkill
							}
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
							p.tr.Update(petId, c.X(), c.Y(), 0, int16(fh.Id()))
						}
					}
					td := p.tr.GetById(pe.Id())
					return mb.Put(pet.EnvStatusEventTopic, spawnEventProvider(pe, td))
				})
				if txErr != nil {
					p.l.WithError(txErr).Errorf("Unable to spawn pet from character [%d].", petId)
					return txErr
				}
				p.l.Infof("Spawned pet [%d] for character [%d].", petId, actorId)
				return nil
			}
		}
	}
}

func (p *ProcessorImpl) DespawnAndEmit(petId uint32, actorId uint32, reason string) error {
	return message.Emit(p.kp)(model.Flip(model.Flip(model.Flip(p.Despawn)(petId))(actorId))(reason))
}

func (p *ProcessorImpl) Despawn(mb *message.Buffer) func(petId uint32) func(actorId uint32) func(reason string) error {
	if p.Despawner != nil {
		return p.Despawner(mb)
	}
	return p.defaultDespawn(mb)
}

func (p *ProcessorImpl) defaultDespawn(mb *message.Buffer) func(petId uint32) func(actorId uint32) func(reason string) error {
	return func(petId uint32) func(actorId uint32) func(reason string) error {
		return func(actorId uint32) func(reason string) error {
			return func(reason string) error {
				p.l.Debugf("Attempting to despawn pet [%d] for character [%d].", petId, actorId)
				txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
					pe, err := p.With(WithTransaction(tx)).GetById(petId)
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
				p.l.Infof("Despawned pet [%d] for character [%d].", petId, actorId)
				return nil
			}
		}
	}
}

func (p *ProcessorImpl) AttemptCommandAndEmit(petId uint32, actorId uint32, commandId byte) error {
	return message.Emit(p.kp)(model.Flip(model.Flip(model.Flip(p.AttemptCommand)(petId))(actorId))(commandId))
}

func (p *ProcessorImpl) AttemptCommand(mb *message.Buffer) func(petId uint32) func(actorId uint32) func(commandId byte) error {
	return func(petId uint32) func(actorId uint32) func(commandId byte) error {
		return func(actorId uint32) func(commandId byte) error {
			return func(commandId byte) error {
				p.l.Debugf("Attempting command [%d] for pet [%d].", commandId, petId)
				txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
					pe, err := p.With(WithTransaction(tx)).GetById(petId)
					if err != nil {
						return err
					}
					if pe.OwnerId() != actorId {
						return errors.New("pet not owned by character")
					}
					if pe.Slot() < 0 {
						return errors.New("pet not active")
					}

					pdm, err := p.dp.GetById(pe.TemplateId())
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
					err = p.With(WithTransaction(tx)).AwardCloseness(mb)(petId)(psm.Increase())
					if err != nil {
						return err
					}
					return mb.Put(pet.EnvStatusEventTopic, commandResponseEventProvider(pe, commandId, success))
				})
				if txErr != nil {
					p.l.WithError(txErr).Errorf("Unable to attempt command [%d] for pet [%d].", commandId, petId)
					return txErr
				}
				p.l.Infof("Performed command [%d] for pet [%d].", commandId, petId)
				return nil
			}
		}
	}
}

func (p *ProcessorImpl) EvaluateHungerAndEmit(ownerId uint32) error {
	return message.Emit(p.kp)(model.Flip(p.EvaluateHunger)(ownerId))
}

func (p *ProcessorImpl) EvaluateHunger(mb *message.Buffer) func(ownerId uint32) error {
	return func(ownerId uint32) error {
		p.l.Debugf("Evaluating hunger of pets for owner [%d].", ownerId)
		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
			ps, err := p.With(WithTransaction(tx)).SpawnedByOwnerProvider(ownerId)()
			if err != nil {
				return err
			}
			for _, pe := range ps {
				var pdm data2.Model
				pdm, err = p.dp.GetById(pe.TemplateId())
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
					err = p.With(WithTransaction(tx)).Despawn(mb)(pe.Id())(pe.OwnerId())(pet.DespawnReasonHunger)
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
		p.l.Infof("Evaluated hunger of pets for owner [%d]", ownerId)
		return nil
	}
}

func (p *ProcessorImpl) ClearPositions(ownerId uint32) error {
	p.l.Debugf("Clearing positions of pets for owner [%d].", ownerId)
	txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
		ps, err := p.With(WithTransaction(tx)).GetByOwner(ownerId)
		if err != nil {
			return err
		}
		for _, pe := range ps {
			p.tr.Remove(pe.Id())
		}
		return nil
	})
	if txErr != nil {
		p.l.WithError(txErr).Errorf("Unable to clear positions of pets for owner [%d].", ownerId)
		return txErr
	}
	p.l.Infof("Cleared positions of pets for owner [%d].", ownerId)
	return nil
}

func (p *ProcessorImpl) AwardClosenessAndEmit(petId uint32, amount uint16) error {
	return p.AwardClosenessWithTransactionAndEmit(uuid.Nil, petId, amount)
}

func (p *ProcessorImpl) AwardClosenessWithTransactionAndEmit(transactionId uuid.UUID, petId uint32, amount uint16) error {
	return message.Emit(p.kp)(func(mb *message.Buffer) error {
		return p.AwardClosenessWithTransaction(mb)(transactionId, petId, amount)
	})
}

func (p *ProcessorImpl) AwardCloseness(mb *message.Buffer) func(petId uint32) func(amount uint16) error {
	return func(petId uint32) func(amount uint16) error {
		return func(amount uint16) error {
			return p.AwardClosenessWithTransaction(mb)(uuid.Nil, petId, amount)
		}
	}
}

func (p *ProcessorImpl) AwardClosenessWithTransaction(mb *message.Buffer) func(transactionId uuid.UUID, petId uint32, amount uint16) error {
	return func(transactionId uuid.UUID, petId uint32, amount uint16) error {
		p.l.Debugf("Awarding [%d] closeness for pet [%d].", amount, petId)
		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
			pe, err := p.With(WithTransaction(tx)).GetById(petId)
			if err != nil {
				return err
			}
			if amount == 0 {
				return nil
			}

			newCloseness := pe.Closeness() + amount
			levels := byte(0)

			for {
				if pe.Level() >= MaxLevel {
					if newCloseness > MaxCloseness {
						newCloseness = MaxCloseness
					}
					break
				}

				levelExp := petExpTable[pe.Level()+levels]
				if newCloseness >= levelExp {
					if pe.Level()+levels >= MaxLevel {
						newCloseness = petExpTable[len(petExpTable)]
					} else {
						levels += 1
					}
				} else {
					break
				}
			}

			err = updateCloseness(tx)(p.t, petId, newCloseness)
			if err != nil {
				return err
			}
			if levels > 0 {
				err = p.With(WithTransaction(tx)).AwardLevel(mb)(pe.Id())(levels)
				if err != nil {
					return err
				}
			}
			pe, err = Clone(pe).SetCloseness(newCloseness).SetLevel(pe.Level() + levels).Build()
			if err != nil {
				return err
			}
			err = mb.Put(pet.EnvStatusEventTopic, closenessChangedEventWithTransactionProvider(pe, int16(amount), transactionId))
			if err != nil {
				return err
			}
			return nil
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Unable to award closeness [%d].", petId)
			return txErr
		}
		p.l.Infof("Awarded [%d] closeness for pet [%d].", amount, petId)
		return nil
	}
}

func (p *ProcessorImpl) AwardFullnessAndEmit(petId uint32, amount byte) error {
	return message.Emit(p.kp)(model.Flip(model.Flip(p.AwardFullness)(petId))(amount))
}

func (p *ProcessorImpl) AwardFullness(mb *message.Buffer) func(petId uint32) func(amount byte) error {
	return func(petId uint32) func(amount byte) error {
		return func(amount byte) error {
			p.l.Debugf("Awarding [%d] fullness for pet [%d].", amount, petId)
			txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
				pe, err := p.With(WithTransaction(tx)).GetById(petId)
				if err != nil {
					return err
				}
				newFullness := pe.Fullness() + amount
				if newFullness > MaxFullness {
					newFullness = MaxFullness
				}
				err = updateFullness(tx)(p.t, petId, newFullness)
				if err != nil {
					return err
				}
				pe, err = Clone(pe).SetFullness(newFullness).Build()
				if err != nil {
					return err
				}
				return mb.Put(pet.EnvStatusEventTopic, fullnessChangedEventProvider(pe, int8(int16(amount))))
			})
			if txErr != nil {
				p.l.WithError(txErr).Errorf("Unable to award fullness to pet [%d].", petId)
				return txErr
			}
			p.l.Infof("Awarded [%d] fullness for pet [%d].", amount, petId)
			return nil
		}
	}
}

func (p *ProcessorImpl) AwardLevelAndEmit(petId uint32, amount byte) error {
	return message.Emit(p.kp)(model.Flip(model.Flip(p.AwardLevel)(petId))(amount))
}

func (p *ProcessorImpl) AwardLevel(mb *message.Buffer) func(petId uint32) func(amount byte) error {
	return func(petId uint32) func(amount byte) error {
		return func(amount byte) error {
			p.l.Debugf("Awarding [%d] level for pet [%d].", amount, petId)
			txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
				pe, err := p.With(WithTransaction(tx)).GetById(petId)
				if err != nil {
					return err
				}
				newLevel := pe.Level() + amount
				if newLevel > MaxLevel {
					newLevel = MaxLevel
				}
				err = updateLevel(tx)(p.t, petId, newLevel)
				if err != nil {
					return err
				}
				pe, err = Clone(pe).SetLevel(newLevel).Build()
				if err != nil {
					return err
				}
				return mb.Put(pet.EnvStatusEventTopic, levelChangedEventProvider(pe, int8(int16(amount))))
			})
			if txErr != nil {
				p.l.WithError(txErr).Errorf("Unable to award level to pet [%d].", petId)
				return txErr
			}
			p.l.Infof("Awarded [%d] level for pet [%d].", amount, petId)
			return nil
		}
	}
}

func (p *ProcessorImpl) SetExcludeAndEmit(petId uint32, items []uint32) error {
	return message.Emit(p.kp)(model.Flip(model.Flip(p.SetExclude)(petId))(items))
}

func (p *ProcessorImpl) SetExclude(mb *message.Buffer) func(petId uint32) func(items []uint32) error {
	return func(petId uint32) func(items []uint32) error {
		return func(items []uint32) error {
			p.l.Debugf("Attempting to set [%d] exclude items for pet [%d].", len(items), petId)
			txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
				err := setExcludes(tx, petId, items)
				if err != nil {
					return err
				}

				pe, err := p.With(WithTransaction(tx)).GetById(petId)
				if err != nil {
					return err
				}
				return mb.Put(pet.EnvStatusEventTopic, excludeChangedEventProvider(pe))
			})
			if txErr != nil {
				p.l.WithError(txErr).Errorf("Unable to set exclude items for pet [%d].", petId)
				return txErr
			}
			p.l.Infof("Set exclude items for pet [%d].", petId)
			return nil
		}
	}
}
