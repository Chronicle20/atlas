package character

import (
	"atlas-character/database"
	"atlas-character/equipable"
	"atlas-character/inventory"
	"atlas-character/tenant"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"regexp"
)

var blockedNameErr = errors.New("blocked name")
var invalidLevelErr = errors.New("invalid level")

func byIdProvider(_ logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(characterId uint32) model.Provider[Model] {
	return func(characterId uint32) model.Provider[Model] {
		return database.ModelProvider[Model, entity](db)(getById(tenant.Id(), characterId), makeCharacter)
	}
}

func GetById(l logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(characterId uint32, decorators ...model.Decorator[Model]) (Model, error) {
	return func(characterId uint32, decorators ...model.Decorator[Model]) (Model, error) {
		return model.ApplyDecorators(byIdProvider(l, db, tenant)(characterId), decorators...)()
	}
}

func byAccountInWorldProvider(_ logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(accountId uint32, worldId byte) model.SliceProvider[Model] {
	return func(accountId uint32, worldId byte) model.SliceProvider[Model] {
		return database.ModelSliceProvider[Model, entity](db)(getForAccountInWorld(tenant.Id(), accountId, worldId), makeCharacter)
	}
}

func GetForAccountInWorld(l logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(accountId uint32, worldId byte, decorators ...model.Decorator[Model]) ([]Model, error) {
	return func(accountId uint32, worldId byte, decorators ...model.Decorator[Model]) ([]Model, error) {
		return model.ApplyDecoratorsSlice(byAccountInWorldProvider(l, db, tenant)(accountId, worldId), decorators...)()
	}
}

func byMapInWorld(_ logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(worldId byte, mapId uint32) model.SliceProvider[Model] {
	return func(worldId byte, mapId uint32) model.SliceProvider[Model] {
		return database.ModelSliceProvider[Model, entity](db)(getForMapInWorld(tenant.Id(), worldId, mapId), makeCharacter)
	}
}

func GetForMapInWorld(l logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(worldId byte, mapId uint32, decorators ...model.Decorator[Model]) ([]Model, error) {
	return func(worldId byte, mapId uint32, decorators ...model.Decorator[Model]) ([]Model, error) {
		return model.ApplyDecoratorsSlice(byMapInWorld(l, db, tenant)(worldId, mapId), decorators...)()
	}
}

func byName(_ logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(name string) model.SliceProvider[Model] {
	return func(name string) model.SliceProvider[Model] {
		return database.ModelSliceProvider[Model, entity](db)(getForName(tenant.Id(), name), makeCharacter)
	}
}

func GetForName(l logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(name string, decorators ...model.Decorator[Model]) ([]Model, error) {
	return func(name string, decorators ...model.Decorator[Model]) ([]Model, error) {
		return model.ApplyDecoratorsSlice(byName(l, db, tenant)(name), decorators...)()
	}
}

func InventoryModelDecorator(l logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) model.Decorator[Model] {
	return func(m Model) Model {
		i, err := inventory.GetInventories(l, db, tenant)(m.Id())
		if err != nil {
			return m
		}

		es, err := equipable.GetEquipment(l, db, tenant)(i.Equipable().Id())
		if err != nil {
			return m
		}

		return CloneModel(m).SetEquipment(m.GetEquipment().Apply(es)).SetInventory(i).Build()
	}
}

func IsValidName(l logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(name string) (bool, error) {
	return func(name string) (bool, error) {
		m, err := regexp.MatchString("[A-Za-z0-9\u3040-\u309F\u30A0-\u30FF\u4E00-\u9FAF]{3,12}", name)
		if err != nil {
			return false, err
		}
		if !m {
			return false, nil
		}

		cs, err := GetForName(l, db, tenant)(name)
		if len(cs) != 0 || err != nil {
			return false, nil
		}

		//TODO
		//bn, err := blocked_name.IsBlockedName(l, span)(name)
		//if bn {
		//	return false, err
		//}

		return true, nil
	}
}

func Create(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(input Model) (Model, error) {
	return func(input Model) (Model, error) {
		ok, err := IsValidName(l, db, tenant)(input.Name())
		if err != nil {
			l.WithError(err).Errorf("Error validating name [%s] during character creation.", input.Name())
			return Model{}, err
		}
		if !ok {
			l.Infof("Attempting to create a character with an invalid name [%s].", input.Name())
			return Model{}, blockedNameErr
		}
		if input.Level() < 1 || input.Level() > 200 {
			l.Infof("Attempting to create character with an invalid level [%d].", input.Level())
			return Model{}, invalidLevelErr
		}

		var res Model
		err = db.Transaction(func(tx *gorm.DB) error {
			res, err = create(tx, tenant.Id(), input.accountId, input.worldId, input.name, input.level, input.strength, input.dexterity, input.intelligence, input.luck, input.maxHp, input.maxMp, input.jobId, input.gender, input.hair, input.face, input.skinColor, input.mapId)
			if err != nil {
				l.WithError(err).Errorf("Error persisting character in database.")
				tx.Rollback()
				return err
			}

			inv, err := inventory.Create(l, tx, tenant)(res.id, 24)
			if err != nil {
				l.WithError(err).Errorf("Unable to create inventory for character during character creation.")
				tx.Rollback()
				return err
			}
			res = CloneModel(res).SetInventory(inv).Build()
			return nil
		})
		return res, err
	}
}
