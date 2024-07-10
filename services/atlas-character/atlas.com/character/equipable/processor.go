package equipable

import (
	"atlas-character/database"
	"atlas-character/equipable/statistics"
	"atlas-character/slottable"
	"atlas-character/tenant"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func byInventoryProvider(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(inventoryId uint32) model.SliceProvider[Model] {
	return func(inventoryId uint32) model.SliceProvider[Model] {
		return database.ModelSliceProvider[Model, entity](db)(getByInventory(tenant.Id, inventoryId), makeModel)
	}
}

func GetByInventory(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(inventoryId uint32) ([]Model, error) {
	return func(inventoryId uint32) ([]Model, error) {
		return model.SliceMap(byInventoryProvider(l, db, span, tenant)(inventoryId), decorateWithStatistics(l, span, tenant), model.ParallelMap())()
	}
}

func GetEquipment(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(inventoryId uint32) ([]Model, error) {
	return func(inventoryId uint32) ([]Model, error) {
		fp := model.FilteredProvider[Model](byInventoryProvider(l, db, span, tenant)(inventoryId), FilterOutInventory)
		return model.SliceMap(fp, decorateWithStatistics(l, span, tenant), model.ParallelMap())()
	}
}

func InInventoryProvider(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(inventoryId uint32) model.SliceProvider[Model] {
	return func(inventoryId uint32) model.SliceProvider[Model] {
		fp := model.FilteredProvider[Model](byInventoryProvider(l, db, span, tenant)(inventoryId), FilterOutEquipment)
		return model.SliceMap(fp, decorateWithStatistics(l, span, tenant), model.ParallelMap())
	}
}

func GetInInventory(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(inventoryId uint32) ([]Model, error) {
	return func(inventoryId uint32) ([]Model, error) {
		return InInventoryProvider(l, db, span, tenant)(inventoryId)()
	}
}

func GetBySlot(_ logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(characterId uint32, slot int16) (Model, error) {
	return func(characterId uint32, slot int16) (Model, error) {
		return database.ModelProvider[Model, entity](db)(getBySlot(tenant.Id, characterId, slot), makeModel)()
	}
}

func FilterOutInventory(e Model) bool {
	return e.Slot() < 0
}

func FilterOutEquipment(e Model) bool {
	return e.Slot() > 0
}

func CreateItem(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(characterId uint32, inventoryId uint32, inventoryType int8, itemId uint32, quantity uint32) model.Provider[slottable.Slottable] {
	return func(characterId uint32, inventoryId uint32, inventoryType int8, itemId uint32, quantity uint32) model.Provider[slottable.Slottable] {
		slot, err := GetNextFreeSlot(l, db, span, tenant)(inventoryId)()
		if err != nil {
			l.WithError(err).Errorf("Unable to locate a free slot to create the item.")
			return model.ErrorProvider[slottable.Slottable](err)
		}

		id, err := statistics.Create(l, span, tenant)(itemId)
		if err != nil {
			l.WithError(err).Errorf("Unable to generate equipment [%d] in equipable storage service for character [%d].", itemId, characterId)
			return model.ErrorProvider[slottable.Slottable](err)
		}

		sm, err := statistics.GetById(l, span, tenant)(id)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve generated equipment statistics for character [%d] new item [%d].", characterId, itemId)
			return model.ErrorProvider[slottable.Slottable](err)
		}

		i, err := createItem(db, tenant, inventoryId, itemId, slot, sm.Id())
		if err != nil {
			return model.ErrorProvider[slottable.Slottable](err)
		}
		rmp := model.Map[Model, Model](model.FixedProvider[Model](i), model.Decorate[Model](statisticsDecorator(sm)))
		return model.Map(rmp, slottableTransformer)
	}
}

func GetNextFreeSlot(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(inventoryId uint32) model.Provider[int16] {
	return func(inventoryId uint32) model.Provider[int16] {
		ms, err := GetByInventory(l, db, span, tenant)(inventoryId)
		if err != nil {
			return model.ErrorProvider[int16](err)
		}
		slot, err := slottable.GetNextFreeSlot(model.SliceMap(model.FixedSliceProvider(ms), slottableTransformer))
		if err != nil {
			return model.ErrorProvider[int16](err)
		}
		return model.FixedProvider[int16](slot)
	}
}

func makeWithStatistics(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(e entity) (Model, error) {
	return func(e entity) (Model, error) {
		m, err := makeModel(e)
		if err != nil {
			return Model{}, err
		}

		sm, err := statistics.GetById(l, span, tenant)(e.ReferenceId)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve generated equipment [%d] statistics.", e.ID)
			return m, nil
		}
		return statisticsDecorator(sm)(m), nil
	}
}

func decorateWithStatistics(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(e Model) (Model, error) {
	return func(e Model) (Model, error) {
		sm, err := statistics.GetById(l, span, tenant)(e.ReferenceId())
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve generated equipment [%d] statistics.", e.Id())
			return e, nil
		}
		return statisticsDecorator(sm)(e), nil
	}
}

func statisticsDecorator(sm statistics.Model) model.Decorator[Model] {
	return func(m Model) Model {
		m.strength = sm.Strength()
		m.dexterity = sm.Dexterity()
		m.intelligence = sm.Intelligence()
		m.luck = sm.Luck()
		m.hp = sm.HP()
		m.mp = sm.MP()
		m.weaponAttack = sm.WeaponAttack()
		m.magicAttack = sm.MagicAttack()
		m.weaponDefense = sm.WeaponDefense()
		m.magicDefense = sm.MagicDefense()
		m.accuracy = sm.Accuracy()
		m.avoidability = sm.Avoidability()
		m.hands = sm.Hands()
		m.speed = sm.Speed()
		m.jump = sm.Jump()
		m.slots = sm.Slots()
		return m
	}
}

func slottableTransformer(m Model) (slottable.Slottable, error) {
	return m, nil
}

func UpdateSlot(_ logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(id uint32, slot int16) error {
	return func(id uint32, slot int16) error {
		return updateSlot(db, tenant.Id, id, slot)
	}
}

func DeleteByReferenceId(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(referenceId uint32) error {
	return func(referenceId uint32) error {
		l.Debugf("Attempting to delete equipment referencing [%d].", referenceId)
		err := statistics.Delete(l, span, tenant)(referenceId)
		if err != nil {
			return err
		}
		return delete(db, tenant.Id, referenceId)
	}
}
