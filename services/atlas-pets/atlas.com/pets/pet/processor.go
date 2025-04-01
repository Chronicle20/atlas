package pet

import (
	"atlas-pets/character"
	"atlas-pets/character/item"
	"atlas-pets/kafka/producer"
	"atlas-pets/pet/data"
	"atlas-pets/pet/position"
	"context"
	"errors"
	"fmt"
	"github.com/Chronicle20/atlas-constants/inventory"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"math/rand"
	"sort"
)

var petExpTable = []uint16{1, 1, 3, 6, 14, 31, 60, 108, 181, 287, 434, 632, 891, 1224, 1642, 2161, 2793, 3557, 4467, 5542, 6801, 8263, 9950, 11882, 14084, 16578, 19391, 22547, 26074, 30000}

func ByIdProvider(ctx context.Context) func(db *gorm.DB) func(petId uint64) model.Provider[Model] {
	t := tenant.MustFromContext(ctx)
	return func(db *gorm.DB) func(petId uint64) model.Provider[Model] {
		return func(petId uint64) model.Provider[Model] {
			return model.Map(modelFromEntity)(getById(t.Id(), petId)(db))
		}
	}
}

func GetById(ctx context.Context) func(db *gorm.DB) func(petId uint64) (Model, error) {
	return func(db *gorm.DB) func(petId uint64) (Model, error) {
		return func(petId uint64) (Model, error) {
			return ByIdProvider(ctx)(db)(petId)()
		}
	}
}

func ByOwnerProvider(ctx context.Context) func(db *gorm.DB) func(ownerId uint32) model.Provider[[]Model] {
	t := tenant.MustFromContext(ctx)
	return func(db *gorm.DB) func(ownerId uint32) model.Provider[[]Model] {
		return func(ownerId uint32) model.Provider[[]Model] {
			return model.SliceMap(modelFromEntity)(getByOwnerId(t.Id(), ownerId)(db))(model.ParallelMap())
		}
	}
}

func GetByOwner(ctx context.Context) func(db *gorm.DB) func(ownerId uint32) ([]Model, error) {
	return func(db *gorm.DB) func(ownerId uint32) ([]Model, error) {
		return func(ownerId uint32) ([]Model, error) {
			return ByOwnerProvider(ctx)(db)(ownerId)()
		}
	}
}

func SpawnedByOwnerProvider(ctx context.Context) func(db *gorm.DB) func(ownerId uint32) model.Provider[[]Model] {
	return func(db *gorm.DB) func(ownerId uint32) model.Provider[[]Model] {
		return func(ownerId uint32) model.Provider[[]Model] {
			return model.FilteredProvider(ByOwnerProvider(ctx)(db)(ownerId), model.Filters[Model](Spawned))
		}
	}
}

func Spawned(m Model) bool {
	return m.Slot() >= 0
}

func HungryByOwnerProvider(ctx context.Context) func(db *gorm.DB) func(ownerId uint32) model.Provider[[]Model] {
	return func(db *gorm.DB) func(ownerId uint32) model.Provider[[]Model] {
		return func(ownerId uint32) model.Provider[[]Model] {
			return model.FilteredProvider(SpawnedByOwnerProvider(ctx)(db)(ownerId), model.Filters[Model](Hungry))
		}
	}
}

func Hungry(m Model) bool {
	return m.Fullness() < 100
}

func HungriestByOwnerProvider(ctx context.Context) func(db *gorm.DB) func(ownerId uint32) model.Provider[Model] {
	return func(db *gorm.DB) func(ownerId uint32) model.Provider[Model] {
		return func(ownerId uint32) model.Provider[Model] {
			ps, err := HungryByOwnerProvider(ctx)(db)(ownerId)()
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
	}
}

func CreateOnAward(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, itemId uint32, slot int16) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, itemId uint32, slot int16) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32, itemId uint32, slot int16) error {
			return func(characterId uint32, itemId uint32, slot int16) error {
				it, ok := inventory.TypeFromItemId(itemId)
				if !ok {
					return errors.New("invalid item id")
				}
				i, err := item.GetItemBySlot(l)(ctx)(characterId, it, slot)
				if err != nil {
					return err
				}

				var om Model
				txErr := db.Transaction(func(tx *gorm.DB) error {
					// TODO lookup name
					im := NewModelBuilder(0, i.Id(), itemId, "Great Pet", characterId)
					om, err = create(tx)(t, characterId, im.Build())
					if err != nil {
						return err
					}
					return nil
				})
				if txErr != nil {
					return txErr
				}
				l.Debugf("Created pet [%d] for character [%d].", om.Id(), characterId)
				return nil
			}
		}
	}
}

func DeleteOnRemove(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, itemId uint32, slot int16) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, itemId uint32, slot int16) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32, itemId uint32, slot int16) error {
			return func(characterId uint32, itemId uint32, slot int16) error {
				it, ok := inventory.TypeFromItemId(itemId)
				if !ok {
					return errors.New("invalid item id")
				}
				i, err := item.GetItemBySlot(l)(ctx)(characterId, it, slot)
				if err != nil {
					return err
				}

				var om Model
				txErr := db.Transaction(deleteByInventoryItemId(t, i.Id()))
				if txErr != nil {
					return txErr
				}
				l.Debugf("Deleted pet [%d] for character [%d].", om.Id(), characterId)
				return nil
			}
		}
	}
}

func DeleteForCharacter(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32) error {
			return func(characterId uint32) error {
				l.Debugf("Deleting all pets for character [%d], because the character has been deleted.", characterId)
				return db.Transaction(deleteForCharacter(t, characterId))
			}
		}
	}
}

func Move(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(petId uint64, m _map.Model, ownerId uint32, x int16, y int16, stance byte) error {
	return func(ctx context.Context) func(db *gorm.DB) func(petId uint64, m _map.Model, ownerId uint32, x int16, y int16, stance byte) error {
		return func(db *gorm.DB) func(petId uint64, m _map.Model, ownerId uint32, x int16, y int16, stance byte) error {
			return func(petId uint64, m _map.Model, ownerId uint32, x int16, y int16, stance byte) error {
				p, err := GetById(ctx)(db)(petId)
				if err != nil {
					l.WithError(err).Errorf("Movement issued for pet by character [%d], which pet [%d] does not exist.", ownerId, petId)
					return err
				}
				if p.OwnerId() != ownerId {
					l.WithError(err).Errorf("Character [%d] attempting to move other character [%d] pet [%d].", ownerId, p.OwnerId(), petId)
					return errors.New("pet not owned by character")
				}

				fh, err := position.GetBelow(l)(ctx)(uint32(m.MapId()), x, y)()
				if err != nil {
					return err
				}
				GetTemporalRegistry().Update(petId, x, y, stance, int16(fh.Id()))
				return nil
			}
		}
	}
}

func Spawn(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(petId uint64, actorId uint32, lead bool) error {
	return func(ctx context.Context) func(db *gorm.DB) func(petId uint64, actorId uint32, lead bool) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(petId uint64, actorId uint32, lead bool) error {
			return func(petId uint64, actorId uint32, lead bool) error {
				var p Model
				slotEvents := model.FixedProvider[[]kafka.Message]([]kafka.Message{})
				txErr := db.Transaction(func(tx *gorm.DB) error {
					var err error
					p, err = GetById(ctx)(tx)(petId)
					if err != nil {
						return err
					}
					if p.OwnerId() != actorId {
						return errors.New("pet not owned by character")
					}

					l.Debugf("Attempting to spawn [%d] for character [%d].", petId, actorId)

					sps, err := SpawnedByOwnerProvider(ctx)(tx)(actorId)()
					if err != nil {
						return err
					}

					newSlot := int8(-1)
					if lead {
						l.Debugf("Pet [%d] will be the new leader.", petId)
						if len(sps) >= 3 {
							return errors.New("too many spawned pets")
						}
						for _, sp := range sps {
							oldSlot := sp.Slot()
							newSlot := oldSlot + 1
							l.Debugf("Attempting to move existing spawned pet [%d] from [%d] to [%d].", sp.Id(), oldSlot, newSlot)
							err = updateSlot(tx)(t, sp.Id(), newSlot)
							if err != nil {
								return err
							}
							sp = sp.SetSlot(newSlot)
							slotEvents = model.MergeSliceProvider(slotEvents, slotChangedEventProvider(sp, oldSlot))
						}
						newSlot = 0
					} else {
						l.Debugf("Finding minimal open slot for [%d].", petId)
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
					oldSlot := p.Slot()
					l.Debugf("Attempting to move pet [%d] to slot [%d].", petId, newSlot)
					err = updateSlot(tx)(t, petId, newSlot)
					if err != nil {
						return err
					}
					p = p.SetSlot(newSlot)
					slotEvents = model.MergeSliceProvider(slotEvents, slotChangedEventProvider(p, oldSlot))
					return nil
				})
				if txErr != nil {
					return txErr
				}

				c, err := character.GetById(l)(ctx)()(actorId)
				if err == nil {
					fh, err := position.GetBelow(l)(ctx)(c.MapId(), c.X(), c.Y())()
					if err == nil {
						GetTemporalRegistry().Update(petId, c.X(), c.Y(), 0, int16(fh.Id()))
					}
				}
				td := GetTemporalRegistry().GetById(p.Id())

				err = producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(spawnEventProvider(p, td))
				if err != nil {
					l.WithError(err).Errorf("Unable to issue spawn events.")
				}

				err = producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(slotEvents)
				if err != nil {
					l.WithError(err).Errorf("Unable to issue slot change events.")
				}
				return err
			}
		}
	}
}

func Despawn(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(petId uint64, actorId uint32, reason string) error {
	return func(ctx context.Context) func(db *gorm.DB) func(petId uint64, actorId uint32, reason string) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(petId uint64, actorId uint32, reason string) error {
			return func(petId uint64, actorId uint32, reason string) error {
				var p Model
				var oldSlot int8
				slotEvents := model.FixedProvider[[]kafka.Message]([]kafka.Message{})
				txErr := db.Transaction(func(tx *gorm.DB) error {
					var err error
					p, err = GetById(ctx)(tx)(petId)
					if err != nil {
						return err
					}
					if p.OwnerId() != actorId {
						return errors.New("pet not owned by character")
					}

					l.Debugf("Attempting to despawn [%d] for character [%d].", petId, actorId)

					sps, err := SpawnedByOwnerProvider(ctx)(tx)(actorId)()
					if err != nil {
						return err
					}

					l.Debugf("Shifting pets to the left.")
					for i := p.Slot() + 1; i < 3; i++ {
						for _, sp := range sps {
							if sp.Slot() == i {
								oldSlot := sp.Slot()
								newSlot := oldSlot - 1

								err = updateSlot(tx)(t, sp.Id(), newSlot)
								if err != nil {
									return err
								}

								sp = sp.SetSlot(newSlot)
								slotEvents = model.MergeSliceProvider(slotEvents, slotChangedEventProvider(sp, oldSlot))
							}
						}
					}

					oldSlot = p.Slot()
					newSlot := int8(-1)
					err = updateSlot(tx)(t, petId, newSlot)
					if err != nil {
						return err
					}
					p = p.SetSlot(newSlot)
					slotEvents = model.MergeSliceProvider(slotEvents, slotChangedEventProvider(p, oldSlot))
					return nil
				})
				if txErr != nil {
					return txErr
				}

				var err error
				err = producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(despawnEventProvider(p, oldSlot, reason))
				if err != nil {
					l.WithError(err).Errorf("Unable to issue despawn events.")
				}

				err = producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(slotEvents)
				if err != nil {
					l.WithError(err).Errorf("Unable to issue slot change events.")
				}
				return err
			}
		}
	}
}

func AttemptCommand(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(petId uint64, actorId uint32, commandId byte, byName bool) error {
	return func(ctx context.Context) func(db *gorm.DB) func(petId uint64, actorId uint32, commandId byte, byName bool) error {
		return func(db *gorm.DB) func(petId uint64, actorId uint32, commandId byte, byName bool) error {
			return func(petId uint64, actorId uint32, commandId byte, byName bool) error {
				var success bool
				p, err := GetById(ctx)(db)(petId)
				if err != nil {
					return err
				}
				if p.OwnerId() != actorId {
					return errors.New("pet not owned by character")
				}
				if p.Slot() < 0 {
					return errors.New("pet not active")
				}

				pdm, err := data.GetById(l)(ctx)(p.TemplateId())
				if err != nil {
					return err
				}
				var psm *data.SkillModel
				psid := fmt.Sprintf("%d-%d", p.TemplateId(), commandId)
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
				err = AwardCloseness(l)(ctx)(db)(petId, psm.Increase(), actorId)
				if err != nil {
					return err
				}
				return producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(commandResponseEventProvider(p, commandId, success))
			}
		}
	}
}

func EvaluateHunger(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(ownerId uint32) error {
	return func(ctx context.Context) func(db *gorm.DB) func(ownerId uint32) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(ownerId uint32) error {
			return func(ownerId uint32) error {
				original := make(map[uint64]Model)
				fullnessChanged := make([]Model, 0)
				despawned := make([]Model, 0)
				txErr := db.Transaction(func(tx *gorm.DB) error {
					ps, err := SpawnedByOwnerProvider(ctx)(tx)(ownerId)()
					if err != nil {
						return err
					}
					for _, p := range ps {
						original[p.Id()] = p

						var pdm data.Model
						pdm, err = data.GetById(l)(ctx)(p.TemplateId())
						if err != nil {
							return err
						}
						newFullness := int16(p.Fullness()) - int16(pdm.Hunger())
						if newFullness < 0 {
							newFullness = 0
						}
						err = updateFullness(tx)(t, p.Id(), byte(newFullness))
						if err != nil {
							return err
						}
						if byte(newFullness) != p.Fullness() {
							fullnessChanged = append(fullnessChanged, p)
						}
						if newFullness <= 5 {
							despawned = append(despawned, p)
						}
					}
					return nil
				})
				if txErr != nil {
					return txErr
				}
				for _, p := range fullnessChanged {
					op := original[p.Id()]
					change := int8(int16(op.Fullness()) - int16(p.Fullness()))
					err := producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(fullnessChangedEventProvider(p, change))
					if err != nil {
						return err
					}
				}
				for _, p := range despawned {
					err := Despawn(l)(ctx)(db)(p.Id(), p.OwnerId(), DespawnReasonHunger)
					if err != nil {
						return err
					}
				}
				return nil
			}
		}
	}
}

func ClearPositions(ctx context.Context) func(db *gorm.DB) func(ownerId uint32) error {
	return func(db *gorm.DB) func(ownerId uint32) error {
		return func(ownerId uint32) error {
			return db.Transaction(func(tx *gorm.DB) error {
				ps, err := GetByOwner(ctx)(tx)(ownerId)
				if err != nil {
					return err
				}
				for _, p := range ps {
					GetTemporalRegistry().Remove(p.Id())
				}
				return nil
			})
		}
	}
}

func AwardCloseness(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(petId uint64, amount uint16, actorId uint32) error {
	return func(ctx context.Context) func(db *gorm.DB) func(petId uint64, amount uint16, actorId uint32) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(petId uint64, amount uint16, actorId uint32) error {
			return func(petId uint64, amount uint16, actorId uint32) error {
				var p Model
				awardLevel := false
				txErr := db.Transaction(func(tx *gorm.DB) error {
					var err error
					p, err = GetById(ctx)(tx)(petId)
					if err != nil {
						return err
					}
					newCloseness := p.Closeness() + amount
					level := p.Level()

					if newCloseness >= petExpTable[p.Level()] {
						if p.Level() >= 30 {
							newCloseness = petExpTable[len(petExpTable)-1]
						} else {
							awardLevel = true
						}
					}
					err = updateCloseness(tx)(t, petId, newCloseness)
					if err != nil {
						return err
					}
					if awardLevel {
						level += 1
						err = updateLevel(tx)(t, petId, level)
						if err != nil {
							return err
						}
					}
					p = Clone(p).SetCloseness(newCloseness).SetLevel(level).Build()
					return nil
				})
				if txErr != nil {
					return txErr
				}
				err := producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(closenessChangedEventProvider(p, int16(amount)))
				if err != nil {
					return err
				}
				if awardLevel {
					err = producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(levelChangedEventProvider(p, 1))
					if err != nil {
						return err
					}
				}
				return nil
			}
		}
	}
}

func AwardFullness(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(petId uint64, amount byte, actorId uint32) error {
	return func(ctx context.Context) func(db *gorm.DB) func(petId uint64, amount byte, actorId uint32) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(petId uint64, amount byte, actorId uint32) error {
			return func(petId uint64, amount byte, actorId uint32) error {
				var p Model
				txErr := db.Transaction(func(tx *gorm.DB) error {
					var err error
					p, err = GetById(ctx)(tx)(petId)
					if err != nil {
						return err
					}
					newFullness := p.Fullness() + amount
					if newFullness > 100 {
						newFullness = 100
					}
					err = updateFullness(tx)(t, petId, newFullness)
					if err != nil {
						return err
					}
					p = Clone(p).SetFullness(newFullness).Build()
					return nil
				})
				if txErr != nil {
					return txErr
				}
				err := producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(fullnessChangedEventProvider(p, int8(amount)))
				if err != nil {
					return err
				}
				return nil
			}
		}
	}
}

func AwardLevel(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(petId uint64, amount byte, actorId uint32) error {
	return func(ctx context.Context) func(db *gorm.DB) func(petId uint64, amount byte, actorId uint32) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(petId uint64, amount byte, actorId uint32) error {
			return func(petId uint64, amount byte, actorId uint32) error {
				var p Model
				txErr := db.Transaction(func(tx *gorm.DB) error {
					var err error
					p, err = GetById(ctx)(tx)(petId)
					if err != nil {
						return err
					}
					newLevel := p.Level() + amount
					if newLevel > 30 {
						newLevel = 30
					}
					err = updateLevel(tx)(t, petId, newLevel)
					if err != nil {
						return err
					}
					p = Clone(p).SetLevel(newLevel).Build()
					return nil
				})
				if txErr != nil {
					return txErr
				}
				err := producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(levelChangedEventProvider(p, int8(amount)))
				if err != nil {
					return err
				}
				return nil
			}
		}
	}
}

func SetExclude(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(petId uint64, items []uint32) error {
	return func(ctx context.Context) func(db *gorm.DB) func(petId uint64, items []uint32) error {
		return func(db *gorm.DB) func(petId uint64, items []uint32) error {
			return func(petId uint64, items []uint32) error {
				var p Model
				txErr := db.Transaction(func(tx *gorm.DB) error {
					err := setExcludes(tx, petId, items)
					if err != nil {
						return err
					}

					p, err = GetById(ctx)(tx)(petId)
					if err != nil {
						return err
					}
					return nil
				})
				if txErr != nil {
					return txErr
				}
				err := producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(excludeChangedEventProvider(p))
				if err != nil {
					return err
				}
				return nil
			}
		}
	}
}
