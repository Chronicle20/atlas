package character

import (
	"atlas-channel/character/inventory"
	"atlas-channel/character/inventory/equipable"
	"atlas-channel/character/inventory/item"
	"atlas-channel/character/skill"
	"atlas-channel/kafka/producer"
	"atlas-channel/pet"
	"context"
	"errors"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
	"sort"
)

func GetById(l logrus.FieldLogger) func(ctx context.Context) func(decorators ...model.Decorator[Model]) func(characterId uint32) (Model, error) {
	return func(ctx context.Context) func(decorators ...model.Decorator[Model]) func(characterId uint32) (Model, error) {
		return func(decorators ...model.Decorator[Model]) func(characterId uint32) (Model, error) {
			return func(characterId uint32) (Model, error) {
				p := requests.Provider[RestModel, Model](l, ctx)(requestById(characterId), Extract)
				return model.Map(model.Decorate(decorators))(p)()
			}
		}
	}
}

func GetByIdWithInventory(l logrus.FieldLogger) func(ctx context.Context) func(decorators ...model.Decorator[Model]) func(characterId uint32) (Model, error) {
	return func(ctx context.Context) func(decorators ...model.Decorator[Model]) func(characterId uint32) (Model, error) {
		return func(decorators ...model.Decorator[Model]) func(characterId uint32) (Model, error) {
			return func(characterId uint32) (Model, error) {
				p := requests.Provider[RestModel, Model](l, ctx)(requestByIdWithInventory(characterId), Extract)
				return model.Map(model.Decorate(decorators))(p)()
			}
		}
	}
}

func SkillModelDecorator(l logrus.FieldLogger) func(ctx context.Context) model.Decorator[Model] {
	return func(ctx context.Context) model.Decorator[Model] {
		return func(m Model) Model {
			ms, err := skill.GetByCharacterId(l)(ctx)(m.Id())
			if err != nil {
				return m
			}
			return m.SetSkills(ms)
		}
	}
}

func PetModelDecorator(l logrus.FieldLogger) func(ctx context.Context) model.Decorator[Model] {
	return func(ctx context.Context) model.Decorator[Model] {
		return func(m Model) Model {
			ms, err := pet.GetByOwner(l)(ctx)(m.Id())
			if err != nil {
				return m
			}
			if len(ms) == 0 {
				return m
			}
			sort.Slice(ms, func(i, j int) bool {
				return ms[i].Slot() < ms[j].Slot()
			})
			return m.SetPets(ms)
		}
	}
}

func GetEquipableInSlot(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, slot int16) model.Provider[equipable.Model] {
	return func(ctx context.Context) func(characterId uint32, slot int16) model.Provider[equipable.Model] {
		return func(characterId uint32, slot int16) model.Provider[equipable.Model] {
			// TODO this needs to be more performant
			c, err := GetByIdWithInventory(l)(ctx)()(characterId)
			if err != nil {
				return model.ErrorProvider[equipable.Model](err)
			}
			for _, e := range c.Inventory().Equipable().Items() {
				if e.Slot() == slot {
					return model.FixedProvider(e)
				}
			}
			return model.ErrorProvider[equipable.Model](errors.New("equipable not found"))
		}
	}
}

func GetItemInSlot(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, inventoryType byte, slot int16) model.Provider[item.Model] {
	return func(ctx context.Context) func(characterId uint32, inventoryType byte, slot int16) model.Provider[item.Model] {
		return func(characterId uint32, inventoryType byte, slot int16) model.Provider[item.Model] {
			// TODO this needs to be more performant
			c, err := GetByIdWithInventory(l)(ctx)()(characterId)
			if err != nil {
				return model.ErrorProvider[item.Model](err)
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
					return model.FixedProvider(e)
				}
			}
			return model.ErrorProvider[item.Model](errors.New("item not found"))
		}
	}
}

func ByNameProvider(l logrus.FieldLogger, ctx context.Context) func(name string) model.Provider[[]Model] {
	return func(name string) model.Provider[[]Model] {
		return requests.SliceProvider[RestModel, Model](l, ctx)(requestByName(name), Extract, model.Filters[Model]())
	}
}

func GetByName(l logrus.FieldLogger, ctx context.Context) func(name string) (Model, error) {
	return func(name string) (Model, error) {
		return model.FirstProvider(ByNameProvider(l, ctx)(name), model.Filters[Model]())()
	}
}

type DistributePacket struct {
	Flag  uint32
	Value uint32
}

func RequestDistributeAp(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, characterId uint32, updateTime uint32, distributes []DistributePacket) error {
	return func(ctx context.Context) func(m _map.Model, characterId uint32, updateTime uint32, distributes []DistributePacket) error {
		return func(m _map.Model, characterId uint32, updateTime uint32, distributes []DistributePacket) error {
			var distributions = make([]DistributePair, 0)
			for _, d := range distributes {
				a, err := abilityFromFlag(d.Flag)
				if err != nil {
					l.WithError(err).Errorf("Character [%d] passed invalid flag when attempting to distribute AP.", characterId)
					return err
				}

				distributions = append(distributions, DistributePair{
					Ability: a,
					Amount:  int8(d.Value),
				})
			}
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(requestDistributeApCommandProvider(m, characterId, distributions))
		}
	}
}

func abilityFromFlag(flag uint32) (string, error) {
	switch flag {
	case 64:
		return CommandDistributeApAbilityStrength, nil
	case 128:
		return CommandDistributeApAbilityDexterity, nil
	case 256:
		return CommandDistributeApAbilityIntelligence, nil
	case 512:
		return CommandDistributeApAbilityLuck, nil
	case 2048:
		return CommandDistributeApAbilityHp, nil
	case 8192:
		return CommandDistributeApAbilityMp, nil
	}
	return "", errors.New("invalid flag")
}

func RequestDropMeso(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, characterId uint32, amount uint32) error {
	return func(ctx context.Context) func(m _map.Model, characterId uint32, amount uint32) error {
		return func(m _map.Model, characterId uint32, amount uint32) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(requestDropMesoCommandProvider(m, characterId, amount))
		}
	}
}

func ChangeHP(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, characterId uint32, amount int16) error {
	return func(ctx context.Context) func(m _map.Model, characterId uint32, amount int16) error {
		return func(m _map.Model, characterId uint32, amount int16) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(changeHPCommandProvider(m, characterId, amount))
		}
	}
}

func ChangeMP(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, characterId uint32, amount int16) error {
	return func(ctx context.Context) func(m _map.Model, characterId uint32, amount int16) error {
		return func(m _map.Model, characterId uint32, amount int16) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(changeMPCommandProvider(m, characterId, amount))
		}
	}
}

func RequestDistributeSp(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, characterId uint32, updateTime uint32, skillId uint32, amount int8) error {
	return func(ctx context.Context) func(m _map.Model, characterId uint32, updateTime uint32, skillId uint32, amount int8) error {
		return func(m _map.Model, characterId uint32, updateTime uint32, skillId uint32, amount int8) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(requestDistributeSpCommandProvider(m, characterId, skillId, amount))
		}
	}
}
