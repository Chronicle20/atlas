package consumable

import (
	"atlas-consumables/character"
	"atlas-consumables/inventory"
	"atlas-consumables/pet"
	"context"
	inventory2 "github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func GetById(l logrus.FieldLogger) func(ctx context.Context) func(itemId uint32) (Model, error) {
	return func(ctx context.Context) func(itemId uint32) (Model, error) {
		return func(itemId uint32) (Model, error) {
			return requests.Provider[RestModel, Model](l, ctx)(requestById(itemId), Extract)()
		}
	}
}

func ConsumeStandard(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, itemId uint32, slot int16, quantity uint32, transactionId uuid.UUID) error {
	return func(ctx context.Context) func(characterId uint32, itemId uint32, slot int16, quantity uint32, transactionId uuid.UUID) error {
		return func(characterId uint32, itemId uint32, slot int16, quantity uint32, transactionId uuid.UUID) error {
			ci, err := GetById(l)(ctx)(itemId)
			if err != nil {
				_ = inventory.CancelItemReservation(l)(ctx)(characterId, inventory2.TypeValueUse, transactionId, slot)
				return err
			}

			err = inventory.ConsumeItem(l)(ctx)(characterId, inventory2.TypeValueUse, transactionId, slot)
			if err != nil {
				return err
			}
			if val, ok := ci.GetSpec(SpecTypeHP); ok {
				_ = character.ChangeHP(l)(ctx)(characterId, int16(val))
			}
			if val, ok := ci.GetSpec(SpecTypeMP); ok {
				_ = character.ChangeMP(l)(ctx)(characterId, int16(val))
			}
			return nil
		}
	}
}

func ConsumePetFood(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, itemId uint32, slot int16, quantity uint32, transactionId uuid.UUID) error {
	return func(ctx context.Context) func(characterId uint32, itemId uint32, slot int16, quantity uint32, transactionId uuid.UUID) error {
		return func(characterId uint32, itemId uint32, slot int16, quantity uint32, transactionId uuid.UUID) error {
			p, err := pet.HungriestByOwnerProvider(l)(ctx)(characterId)()
			if err != nil {
				_ = inventory.CancelItemReservation(l)(ctx)(characterId, inventory2.TypeValueUse, transactionId, slot)
				return err
			}

			ci, err := GetById(l)(ctx)(itemId)
			if err != nil {
				_ = inventory.CancelItemReservation(l)(ctx)(characterId, inventory2.TypeValueUse, transactionId, slot)
				return err
			}
			inc := byte(0)
			if val, ok := ci.GetSpec(SpecTypeInc); ok {
				inc = byte(val)
			}

			err = pet.AwardFullness(l)(ctx)(characterId, p.Id(), inc)
			if err != nil {
				return err
			}

			err = inventory.ConsumeItem(l)(ctx)(characterId, inventory2.TypeValueUse, transactionId, slot)
			if err != nil {
				return err
			}
			return nil
		}
	}
}
