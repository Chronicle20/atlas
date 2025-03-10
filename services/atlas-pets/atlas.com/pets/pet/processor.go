package pet

import (
	"atlas-pets/character/item"
	"atlas-pets/kafka/producer"
	"atlas-pets/pet/position"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-constants/inventory"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

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
					om, err = create(db)(t, characterId, im.Build())
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

type MovementSummary struct {
	X      int16
	Y      int16
	Stance byte
}

func MovementSummaryProvider(x int16, y int16, stance byte) model.Provider[MovementSummary] {
	return func() (MovementSummary, error) {
		return MovementSummary{
			X:      x,
			Y:      y,
			Stance: stance,
		}, nil
	}
}

func FoldMovementSummary(summary MovementSummary, e Element) (MovementSummary, error) {
	ms := MovementSummary{X: summary.X, Y: summary.Y, Stance: summary.Stance}
	if e.TypeStr == MovementTypeNormal {
		ms.X = e.X
		ms.Y = e.Y
		ms.Stance = e.MoveAction
	} else if e.TypeStr == MovementTypeJump || e.TypeStr == MovementTypeTeleport || e.TypeStr == MovementTypeStartFallDown {
		ms.Stance = e.MoveAction
	}
	return ms, nil
}

func Move(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(petId uint64) func(m _map.Model) func(ownerId uint32) func(movement Movement) error {
	return func(ctx context.Context) func(db *gorm.DB) func(petId uint64) func(m _map.Model) func(ownerId uint32) func(movement Movement) error {
		return func(db *gorm.DB) func(petId uint64) func(m _map.Model) func(ownerId uint32) func(movement Movement) error {
			return func(petId uint64) func(m _map.Model) func(ownerId uint32) func(movement Movement) error {
				return func(m _map.Model) func(ownerId uint32) func(movement Movement) error {
					return func(ownerId uint32) func(movement Movement) error {
						return func(movement Movement) error {
							p, err := GetById(ctx)(db)(petId)
							if err != nil {
								l.WithError(err).Errorf("Movement issued for pet by character [%d], which pet [%d] does not exist.", ownerId, petId)
								return err
							}
							if p.OwnerId() != ownerId {
								l.WithError(err).Errorf("Character [%d] attempting to move other character [%d] pet [%d].", ownerId, p.OwnerId(), petId)
								return errors.New("pet not owned by character")
							}

							msp := model.Fold(model.FixedProvider(movement.Elements), MovementSummaryProvider(movement.StartX, movement.StartY, GetTemporalRegistry().GetById(petId).Stance()), FoldMovementSummary)

							err = model.For(msp, func(ms MovementSummary) error {
								fh, err := position.GetBelow(l)(ctx)(uint32(m.MapId()), ms.X, ms.Y)()
								if err != nil {
									return err
								}
								return updateTemporal(petId, int16(fh.Id()))(ms)
							})
							if err != nil {
								return err
							}
							return producer.ProviderImpl(l)(ctx)(EnvEventTopicMovement)(moveEventProvider(m, p, movement))
						}
					}
				}
			}
		}
	}
}

func updateTemporal(petId uint64, fh int16) model.Operator[MovementSummary] {
	return func(ms MovementSummary) error {
		GetTemporalRegistry().Update(petId, ms.X, ms.Y, ms.Stance, fh)
		return nil
	}
}
