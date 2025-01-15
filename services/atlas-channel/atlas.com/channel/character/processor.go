package character

import (
	"atlas-channel/character/inventory"
	"atlas-channel/character/inventory/equipable"
	"atlas-channel/character/inventory/item"
	"atlas-channel/kafka/producer"
	"atlas-channel/socket/model"
	"context"
	"errors"
	model2 "github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func GetById(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) (Model, error) {
	return func(ctx context.Context) func(characterId uint32) (Model, error) {
		return func(characterId uint32) (Model, error) {
			return requests.Provider[RestModel, Model](l, ctx)(requestById(characterId), Extract)()
		}
	}
}

func GetByIdWithInventory(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) (Model, error) {
	return func(ctx context.Context) func(characterId uint32) (Model, error) {
		return func(characterId uint32) (Model, error) {
			return requests.Provider[RestModel, Model](l, ctx)(requestByIdWithInventory(characterId), Extract)()
		}
	}
}

func Move(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, mm model.Movement) {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, mm model.Movement) {
		moveCharacterCommandFunc := producer.ProviderImpl(l)(ctx)(EnvCommandTopicMovement)
		return func(worldId byte, channelId byte, mapId uint32, characterId uint32, mm model.Movement) {
			err := moveCharacterCommandFunc(move(worldId, channelId, mapId, characterId, mm))
			if err != nil {
				l.WithError(err).Errorf("Unable to distribute character movement to other services.")
			}
		}
	}
}

func GetEquipableInSlot(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, slot int16) model2.Provider[equipable.Model] {
	return func(ctx context.Context) func(characterId uint32, slot int16) model2.Provider[equipable.Model] {
		return func(characterId uint32, slot int16) model2.Provider[equipable.Model] {
			// TODO this needs to be more performant
			c, err := GetByIdWithInventory(l)(ctx)(characterId)
			if err != nil {
				return model2.ErrorProvider[equipable.Model](err)
			}
			for _, e := range c.Inventory().Equipable().Items() {
				if e.Slot() == slot {
					return model2.FixedProvider(e)
				}
			}
			return model2.ErrorProvider[equipable.Model](errors.New("equipable not found"))
		}
	}
}

func GetItemInSlot(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, inventoryType byte, slot int16) model2.Provider[item.Model] {
	return func(ctx context.Context) func(characterId uint32, inventoryType byte, slot int16) model2.Provider[item.Model] {
		return func(characterId uint32, inventoryType byte, slot int16) model2.Provider[item.Model] {
			// TODO this needs to be more performant
			c, err := GetByIdWithInventory(l)(ctx)(characterId)
			if err != nil {
				return model2.ErrorProvider[item.Model](err)
			}

			var inv func() inventory.ItemModel
			switch inventoryType {
			case 2:
				inv = c.Inventory().Use
			case 3:
				inv = c.Inventory().Setup
			case 4:
				inv = c.Inventory().Etc
			case 5:
				inv = c.Inventory().Cash
			}

			for _, e := range inv().Items() {
				if e.Slot() == slot {
					return model2.FixedProvider(e)
				}
			}
			return model2.ErrorProvider[item.Model](errors.New("item not found"))
		}
	}
}

func byNameProvider(l logrus.FieldLogger, ctx context.Context) func(name string) model2.Provider[[]Model] {
	return func(name string) model2.Provider[[]Model] {
		return requests.SliceProvider[RestModel, Model](l, ctx)(requestByName(name), Extract, model2.Filters[Model]())
	}
}

func GetByName(l logrus.FieldLogger, ctx context.Context) func(name string) ([]Model, error) {
	return func(name string) ([]Model, error) {
		return byNameProvider(l, ctx)(name)()
	}
}
