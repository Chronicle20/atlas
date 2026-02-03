package drop

import (
	"atlas-monster-death/kafka/producer"
	"atlas-monster-death/monster/drop/position"
	"context"
	"math/rand"

	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/sirupsen/logrus"
)

func Create(l logrus.FieldLogger) func(ctx context.Context) func(f field.Model, index int, monsterId uint32, x int16, y int16, killerId uint32, dropType byte, m Model, mesoRate float64) error {
	return func(ctx context.Context) func(f field.Model, index int, monsterId uint32, x int16, y int16, killerId uint32, dropType byte, m Model, mesoRate float64) error {
		return func(f field.Model, index int, monsterId uint32, x int16, y int16, killerId uint32, dropType byte, m Model, mesoRate float64) error {
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
				return SpawnMeso(l)(ctx)(f, monsterId, x, y, killerId, dropType, m, newX, y, mesoRate)
			}
			return SpawnItem(l)(ctx)(f, m.ItemId(), monsterId, x, y, killerId, dropType, m, newX, y)
		}
	}
}

func SpawnMeso(l logrus.FieldLogger) func(ctx context.Context) func(f field.Model, monsterId uint32, x int16, y int16, killerId uint32, dropType byte, m Model, posX int16, posY int16, mesoRate float64) error {
	return func(ctx context.Context) func(f field.Model, monsterId uint32, x int16, y int16, killerId uint32, dropType byte, m Model, posX int16, posY int16, mesoRate float64) error {
		return func(f field.Model, monsterId uint32, x int16, y int16, killerId uint32, dropType byte, m Model, posX int16, posY int16, mesoRate float64) error {
			baseMesos := uint32(rand.Int31n(int32(m.MaximumQuantity()-m.MinimumQuantity())+1)) + m.MinimumQuantity()
			// Apply meso rate multiplier
			mesos := uint32(float64(baseMesos) * mesoRate)
			l.Debugf("Meso drop: base=%d, rate=%.2f, final=%d", baseMesos, mesoRate, mesos)
			return SpawnDrop(l)(ctx)(f, 0, 0, mesos, posX, posY, x, y, monsterId, killerId, false, dropType)
		}
	}
}

func SpawnItem(l logrus.FieldLogger) func(ctx context.Context) func(f field.Model, itemId uint32, monsterId uint32, x int16, y int16, killerId uint32, dropType byte, m Model, posX int16, posY int16) error {
	return func(ctx context.Context) func(f field.Model, itemId uint32, monsterId uint32, x int16, y int16, killerId uint32, dropType byte, m Model, posX int16, posY int16) error {
		return func(f field.Model, itemId uint32, monsterId uint32, x int16, y int16, killerId uint32, dropType byte, m Model, posX int16, posY int16) error {
			quantity := uint32(1)
			if m.MaximumQuantity() != 1 {
				quantity = uint32(rand.Int31n(int32(m.MaximumQuantity()-m.MinimumQuantity())+1)) + m.MinimumQuantity()
			}
			return SpawnDrop(l)(ctx)(f, itemId, quantity, 0, posX, posY, x, y, monsterId, killerId, false, dropType)
		}
	}
}

func SpawnDrop(l logrus.FieldLogger) func(ctx context.Context) func(f field.Model, itemId uint32, quantity uint32, mesos uint32, posX int16, posY int16, monsterX int16, monsterY int16, monsterId uint32, killerId uint32, playerDrop bool, dropType byte) error {
	return func(ctx context.Context) func(f field.Model, itemId uint32, quantity uint32, mesos uint32, posX int16, posY int16, monsterX int16, monsterY int16, monsterId uint32, killerId uint32, playerDrop bool, dropType byte) error {
		return func(f field.Model, itemId uint32, quantity uint32, mesos uint32, posX int16, posY int16, monsterX int16, monsterY int16, monsterId uint32, killerId uint32, playerDrop bool, dropType byte) error {
			tempX, tempY := calculateDropPosition(l)(ctx)(f.MapId(), posX, posY, monsterX, monsterY)
			tempX, tempY = calculateDropPosition(l)(ctx)(f.MapId(), tempX, tempY, tempX, tempY)
			cp := spawnDropCommandProvider(f, itemId, quantity, mesos, dropType, tempX, tempY, killerId, 0, monsterId, monsterX, monsterY, playerDrop, byte(1))
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(cp)
		}
	}
}

func calculateDropPosition(l logrus.FieldLogger) func(ctx context.Context) func(mapId _map.Id, initialX int16, initialY int16, fallbackX int16, fallbackY int16) (int16, int16) {
	return func(ctx context.Context) func(mapId _map.Id, initialX int16, initialY int16, fallbackX int16, fallbackY int16) (int16, int16) {
		return func(mapId _map.Id, initialX int16, initialY int16, fallbackX int16, fallbackY int16) (int16, int16) {
			r, err := position.GetInMap(l)(ctx)(mapId, initialX, initialY, fallbackX, fallbackY)()
			if err != nil {
				return fallbackX, fallbackY
			}
			return r.X(), r.Y()
		}
	}
}
