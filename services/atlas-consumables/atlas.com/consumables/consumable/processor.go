package consumable

import (
	"atlas-consumables/character"
	"atlas-consumables/character/buff"
	"atlas-consumables/character/buff/stat"
	"atlas-consumables/inventory"
	_map "atlas-consumables/map/character"
	"atlas-consumables/pet"
	"context"
	ts "github.com/Chronicle20/atlas-constants/character"
	inventory2 "github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"math"
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
			c, err := character.GetById(l)(ctx)()(characterId)
			if err != nil {
				_ = inventory.CancelItemReservation(l)(ctx)(characterId, inventory2.TypeValueUse, transactionId, slot)
				return err
			}

			m, err := _map.GetMap(characterId)
			if err != nil {
				_ = inventory.CancelItemReservation(l)(ctx)(characterId, inventory2.TypeValueUse, transactionId, slot)
				return err
			}

			ci, err := GetById(l)(ctx)(itemId)
			if err != nil {
				_ = inventory.CancelItemReservation(l)(ctx)(characterId, inventory2.TypeValueUse, transactionId, slot)
				return err
			}

			err = inventory.ConsumeItem(l)(ctx)(characterId, inventory2.TypeValueUse, transactionId, slot)
			if err != nil {
				return err
			}

			statups := make([]stat.Model, 0)
			duration := int32(0)
			if val, ok := ci.GetSpec(SpecTypeAccuracy); ok && val > 0 {
				statups = append(statups, stat.Model{
					Type:   ts.TemporaryStatTypeAccuracy,
					Amount: val,
				})
			}
			if val, ok := ci.GetSpec(SpecTypeEvasion); ok && val > 0 {
				statups = append(statups, stat.Model{
					Type:   ts.TemporaryStatTypeAvoidability,
					Amount: val,
				})
			}
			if val, ok := ci.GetSpec(SpecTypeHP); ok && val > 0 {
				_ = character.ChangeHP(l)(ctx)(m, characterId, int16(val))
			}
			if val, ok := ci.GetSpec(SpecTypeHPRecovery); ok && val > 0 {
				pct := float64(val) / float64(100)
				res := int16(math.Floor(float64(c.MaxHp()) * pct))
				_ = character.ChangeHP(l)(ctx)(m, characterId, res)
			}
			if val, ok := ci.GetSpec(SpecTypeJump); ok && val > 0 {
				statups = append(statups, stat.Model{
					Type:   ts.TemporaryStatTypeJump,
					Amount: val,
				})
			}
			if val, ok := ci.GetSpec(SpecTypeMagicAttack); ok && val > 0 {
				statups = append(statups, stat.Model{
					Type:   ts.TemporaryStatTypeMagicAttack,
					Amount: val,
				})
			}
			if val, ok := ci.GetSpec(SpecTypeMagicDefense); ok && val > 0 {
				statups = append(statups, stat.Model{
					Type:   ts.TemporaryStatTypeMagicDefense,
					Amount: val,
				})
			}
			if val, ok := ci.GetSpec(SpecTypeMP); ok && val > 0 {
				_ = character.ChangeMP(l)(ctx)(m, characterId, int16(val))
			}
			if val, ok := ci.GetSpec(SpecTypeMPRecovery); ok && val > 0 {
				pct := float64(val) / float64(100)
				res := int16(math.Floor(float64(c.MaxMp()) * pct))
				_ = character.ChangeMP(l)(ctx)(m, characterId, res)
			}
			if val, ok := ci.GetSpec(SpecTypeWeaponAttack); ok && val > 0 {
				statups = append(statups, stat.Model{
					Type:   ts.TemporaryStatTypeWeaponAttack,
					Amount: val,
				})
			}
			if val, ok := ci.GetSpec(SpecTypeWeaponDefense); ok && val > 0 {
				statups = append(statups, stat.Model{
					Type:   ts.TemporaryStatTypeWeaponDefense,
					Amount: val,
				})
			}
			if val, ok := ci.GetSpec(SpecTypeSpeed); ok && val > 0 {
				statups = append(statups, stat.Model{
					Type:   ts.TemporaryStatTypeSpeed,
					Amount: val,
				})
			}
			if val, ok := ci.GetSpec(SpecTypeTime); ok && val > 0 {
				duration = val / 1000
			}

			if len(statups) > 0 {
				_ = buff.Apply(l)(ctx)(m, characterId, -int32(itemId), duration, statups)(characterId)
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
