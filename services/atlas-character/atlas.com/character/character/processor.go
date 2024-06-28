package character

import (
	"atlas-character/database"
	"atlas-character/equipable"
	"atlas-character/inventory"
	"atlas-character/tenant"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

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
		es, err := equipable.GetEquipment(l, db, tenant)(m.Id())
		if err != nil {
			return m
		}

		i, err := inventory.GetInventories(l, db, tenant)(m.Id())
		if err != nil {
			return m
		}
		return CloneModel(m).SetEquipment(m.GetEquipment().Apply(es)).SetInventory(i).Build()
	}
}
