package asset

import (
	"atlas-inventory/cash"
	"atlas-inventory/equipable"
	"atlas-inventory/pet"
	"atlas-inventory/stackable"
	"context"
	"errors"

	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func GetByCompartmentId(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(compartmentId uuid.UUID, inventoryType inventory.Type) ([]Model[any], error) {
	return func(ctx context.Context) func(db *gorm.DB) func(compartmentId uuid.UUID, inventoryType inventory.Type) ([]Model[any], error) {
		return func(db *gorm.DB) func(compartmentId uuid.UUID, inventoryType inventory.Type) ([]Model[any], error) {
			return func(compartmentId uuid.UUID, inventoryType inventory.Type) ([]Model[any], error) {
				return ByCompartmentIdProvider(l)(ctx)(db)(compartmentId, inventoryType)()
			}
		}
	}
}

func ByCompartmentIdProvider(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(compartmentId uuid.UUID, inventoryType inventory.Type) model.Provider[[]Model[any]] {
	return func(ctx context.Context) func(db *gorm.DB) func(compartmentId uuid.UUID, inventoryType inventory.Type) model.Provider[[]Model[any]] {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(compartmentId uuid.UUID, inventoryType inventory.Type) model.Provider[[]Model[any]] {
			return func(compartmentId uuid.UUID, inventoryType inventory.Type) model.Provider[[]Model[any]] {
				ap := model.SliceMap(Make)(getByCompartmentId(t.Id(), compartmentId)(db))(model.ParallelMap())
				var refDecorator model.Transformer[Model[any], Model[any]]
				if inventoryType == inventory.TypeValueEquip {
					refDecorator = DecorateEquipable(l)(ctx)
				} else if inventoryType == inventory.TypeValueUse || inventoryType == inventory.TypeValueSetup || inventoryType == inventory.TypeValueETC {
					sm, err := model.CollectToMap(stackable.ByCompartmentIdProvider(l)(ctx)(db)(compartmentId), stackable.Identity, stackable.This)()
					if err != nil {
						return model.ErrorProvider[[]Model[any]](err)
					}
					refDecorator = DecorateStackable(sm)
				} else if inventoryType == inventory.TypeValueCash {
					refDecorator = DecorateCash(l)(ctx)
				}
				return model.SliceMap(refDecorator)(ap)(model.ParallelMap())
			}
		}
	}
}

func DecorateEquipable(l logrus.FieldLogger) func(ctx context.Context) model.Transformer[Model[any], Model[any]] {
	return func(ctx context.Context) model.Transformer[Model[any], Model[any]] {
		return func(m Model[any]) (Model[any], error) {
			e, err := equipable.GetById(l)(ctx)(m.ReferenceId())
			if err != nil {
				return Model[any]{}, nil
			}
			return Clone(m).
				SetReferenceData(EquipableReferenceData{
					strength:       e.Strength(),
					dexterity:      e.Dexterity(),
					intelligence:   e.Intelligence(),
					luck:           e.Luck(),
					hp:             e.HP(),
					mp:             e.MP(),
					weaponAttack:   e.WeaponAttack(),
					magicAttack:    e.MagicAttack(),
					weaponDefense:  e.WeaponDefense(),
					magicDefense:   e.MagicDefense(),
					accuracy:       e.Accuracy(),
					avoidability:   e.Avoidability(),
					hands:          e.Hands(),
					speed:          e.Speed(),
					jump:           e.Jump(),
					slots:          e.Slots(),
					ownerName:      e.OwnerName(),
					locked:         e.Locked(),
					spikes:         e.Spikes(),
					karmaUsed:      e.KarmaUsed(),
					cold:           e.Cold(),
					canBeTraded:    e.CanBeTraded(),
					levelType:      e.LevelType(),
					level:          e.Level(),
					experience:     e.Experience(),
					hammersApplied: e.HammersApplied(),
					expiration:     e.Expiration(),
				}).
				Build(), nil
		}
	}
}

func DecorateStackable(sm map[uint32]stackable.Model) model.Transformer[Model[any], Model[any]] {
	return func(m Model[any]) (Model[any], error) {
		var s stackable.Model
		var ok bool
		if s, ok = sm[m.ReferenceId()]; !ok {
			return m, errors.New("cannot locate reference")
		}

		var rd any
		if m.ReferenceType() == "consumable" {
			rd = ConsumableReferenceData{
				quantity:     s.Quantity(),
				owner:        s.Owner(),
				flag:         s.Flag(),
				rechargeable: s.Rechargeable(),
			}
		} else if m.ReferenceType() == "setup" {
			rd = SetupReferenceData{
				quantity: s.Quantity(),
				owner:    s.Owner(),
				flag:     s.Flag(),
			}
		} else if m.ReferenceType() == "etc" {
			rd = EtcReferenceData{
				quantity: s.Quantity(),
				owner:    s.Owner(),
				flag:     s.Flag(),
			}
		}

		return Clone(m).
			SetReferenceData(rd).
			Build(), nil
	}
}

func DecorateCash(l logrus.FieldLogger) func(ctx context.Context) model.Transformer[Model[any], Model[any]] {
	return func(ctx context.Context) model.Transformer[Model[any], Model[any]] {
		return func(m Model[any]) (Model[any], error) {
			if m.ReferenceType() == "cash" {
				ci, err := cash.GetById(l)(ctx)(m.ReferenceId())
				if err != nil {
					return m, errors.New("cannot locate reference")
				}
				return Clone(m).
					SetReferenceData(CashReferenceData{
						quantity:   ci.Quantity(),
						owner:      ci.Owner(),
						flag:       ci.Flag(),
						purchaseBy: ci.PurchasedBy(),
					}).
					Build(), nil
			} else if m.ReferenceType() == "pet" {
				ci, err := cash.GetById(l)(ctx)(m.ReferenceId())
				if err != nil {
					return m, errors.New("cannot locate reference")
				}
				pi, err := pet.GetById(l)(ctx)(m.ReferenceId())
				if err != nil {
					return m, errors.New("cannot locate reference")
				}
				return Clone(m).
					SetReferenceData(PetReferenceData{
						cashId:     ci.CashId(),
						owner:      ci.Owner(),
						flag:       ci.Flag(),
						purchaseBy: ci.PurchasedBy(),
						name:       pi.Name(),
						level:      pi.Level(),
						closeness:  pi.Closeness(),
						fullness:   pi.Fullness(),
						expiration: pi.Expiration(),
					}).
					Build(), nil
			}
			return m, nil
		}
	}
}
