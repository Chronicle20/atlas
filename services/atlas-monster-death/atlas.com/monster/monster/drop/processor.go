package drop

import (
	"atlas-monster-death/kafka/producer"
	"atlas-monster-death/monster/drop/position"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
	"math/rand"
)

func byMonsterIdProvider(l logrus.FieldLogger) func(ctx context.Context) func(monsterId uint32) model.Provider[[]Model] {
	return func(ctx context.Context) func(monsterId uint32) model.Provider[[]Model] {
		return func(monsterId uint32) model.Provider[[]Model] {
			return requests.SliceProvider[RestModel, Model](l, ctx)(requestForMonster(monsterId), Extract, model.Filters[Model]())
		}
	}
}

func GetByMonsterId(l logrus.FieldLogger) func(ctx context.Context) func(monsterId uint32) ([]Model, error) {
	return func(ctx context.Context) func(monsterId uint32) ([]Model, error) {
		return func(monsterId uint32) ([]Model, error) {
			return byMonsterIdProvider(l)(ctx)(monsterId)()
		}
	}
}

func Create(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, index int, monsterId uint32, x int16, y int16, killerId uint32, dropType byte, m Model) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, index int, monsterId uint32, x int16, y int16, killerId uint32, dropType byte, m Model) error {
		return func(worldId byte, channelId byte, mapId uint32, index int, monsterId uint32, x int16, y int16, killerId uint32, dropType byte, m Model) error {
			factor := 0
			if dropType == 3 {
				factor = 40
			} else {
				factor = 25
			}
			newX := x
			if index%2 == 0 {
				newX += int16(factor * ((index + 1) / 2))
			} else {
				newX += int16(-(factor * (index / 2)))
			}
			if m.ItemId() == 0 {
				return SpawnMeso(l)(ctx)(worldId, channelId, mapId, monsterId, x, y, killerId, dropType, m, newX, y)
			}
			return SpawnItem(l)(ctx)(worldId, channelId, mapId, m.ItemId(), monsterId, x, y, killerId, dropType, m, newX, y)
		}
	}
}

func SpawnMeso(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, monsterId uint32, x int16, y int16, killerId uint32, dropType byte, m Model, posX int16, posY int16) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, monsterId uint32, x int16, y int16, killerId uint32, dropType byte, m Model, posX int16, posY int16) error {
		return func(worldId byte, channelId byte, mapId uint32, monsterId uint32, x int16, y int16, killerId uint32, dropType byte, m Model, posX int16, posY int16) error {
			mesos := uint32(rand.Int31n(int32(m.MaximumQuantity()-m.MinimumQuantity())+1)) + m.MinimumQuantity()
			//TODO apply characters meso buff.
			return SpawnDrop(l)(ctx)(worldId, channelId, mapId, 0, 0, mesos, posX, posY, x, y, monsterId, killerId, false, dropType)
		}
	}
}

func SpawnItem(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, itemId uint32, monsterId uint32, x int16, y int16, killerId uint32, dropType byte, m Model, posX int16, posY int16) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, itemId uint32, monsterId uint32, x int16, y int16, killerId uint32, dropType byte, m Model, posX int16, posY int16) error {
		return func(worldId byte, channelId byte, mapId uint32, itemId uint32, monsterId uint32, x int16, y int16, killerId uint32, dropType byte, m Model, posX int16, posY int16) error {
			quantity := uint32(1)
			if m.MaximumQuantity() != 1 {
				quantity = uint32(rand.Int31n(int32(m.MaximumQuantity()-m.MinimumQuantity())+1)) + m.MinimumQuantity()
			}
			return SpawnDrop(l)(ctx)(worldId, channelId, mapId, itemId, quantity, 0, posX, posY, x, y, monsterId, killerId, false, dropType)
		}
	}
}

func SpawnDrop(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, itemId uint32, quantity uint32, mesos uint32, posX int16, posY int16, monsterX int16, monsterY int16, monsterId uint32, killerId uint32, playerDrop bool, dropType byte) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, itemId uint32, quantity uint32, mesos uint32, posX int16, posY int16, monsterX int16, monsterY int16, monsterId uint32, killerId uint32, playerDrop bool, dropType byte) error {
		return func(worldId byte, channelId byte, mapId uint32, itemId uint32, quantity uint32, mesos uint32, posX int16, posY int16, monsterX int16, monsterY int16, monsterId uint32, killerId uint32, playerDrop bool, dropType byte) error {
			tempX, tempY := calculateDropPosition(l)(ctx)(mapId, posX, posY, monsterX, monsterY)
			tempX, tempY = calculateDropPosition(l)(ctx)(mapId, tempX, tempY, tempX, tempY)
			cp := spawnDropCommandProvider(worldId, channelId, mapId, itemId, quantity, mesos, dropType, tempX, tempY, killerId, 0, monsterId, monsterX, monsterY, playerDrop, byte(1))
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(cp)
		}
	}
}

func calculateDropPosition(l logrus.FieldLogger) func(ctx context.Context) func(mapId uint32, initialX int16, initialY int16, fallbackX int16, fallbackY int16) (int16, int16) {
	return func(ctx context.Context) func(mapId uint32, initialX int16, initialY int16, fallbackX int16, fallbackY int16) (int16, int16) {
		return func(mapId uint32, initialX int16, initialY int16, fallbackX int16, fallbackY int16) (int16, int16) {
			r, err := position.GetInMap(l)(ctx)(mapId, initialX, initialY, fallbackX, fallbackY)()
			if err != nil {
				return fallbackX, fallbackY
			}
			return r.X(), r.Y()
		}
	}
}
