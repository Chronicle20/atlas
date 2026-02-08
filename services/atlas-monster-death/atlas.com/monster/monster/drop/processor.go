package drop

import (
	"atlas-monster-death/data/equipment/statistics"
	"atlas-monster-death/kafka/producer"
	"atlas-monster-death/monster/drop/position"
	"context"
	"math"
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
			return SpawnDrop(l)(ctx)(f, 0, 0, mesos, posX, posY, x, y, monsterId, killerId, false, dropType, EquipmentData{})
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

			var ed EquipmentData
			if isEquipment(itemId) {
				sp := statistics.NewProcessor(l, ctx)
				s, err := sp.GetById(itemId)
				if err != nil {
					l.WithError(err).Errorf("Unable to get equipment statistics for item [%d], dropping without stats.", itemId)
				} else {
					ed = generateRandomEquipmentData(s)
				}
			}

			return SpawnDrop(l)(ctx)(f, itemId, quantity, 0, posX, posY, x, y, monsterId, killerId, false, dropType, ed)
		}
	}
}

func SpawnDrop(l logrus.FieldLogger) func(ctx context.Context) func(f field.Model, itemId uint32, quantity uint32, mesos uint32, posX int16, posY int16, monsterX int16, monsterY int16, monsterId uint32, killerId uint32, playerDrop bool, dropType byte, ed EquipmentData) error {
	return func(ctx context.Context) func(f field.Model, itemId uint32, quantity uint32, mesos uint32, posX int16, posY int16, monsterX int16, monsterY int16, monsterId uint32, killerId uint32, playerDrop bool, dropType byte, ed EquipmentData) error {
		return func(f field.Model, itemId uint32, quantity uint32, mesos uint32, posX int16, posY int16, monsterX int16, monsterY int16, monsterId uint32, killerId uint32, playerDrop bool, dropType byte, ed EquipmentData) error {
			tempX, tempY := calculateDropPosition(l)(ctx)(f.MapId(), posX, posY, monsterX, monsterY)
			tempX, tempY = calculateDropPosition(l)(ctx)(f.MapId(), tempX, tempY, tempX, tempY)
			cp := spawnDropCommandProvider(f, itemId, quantity, mesos, dropType, tempX, tempY, killerId, 0, monsterId, monsterX, monsterY, playerDrop, byte(1), ed)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(cp)
		}
	}
}

func isEquipment(itemId uint32) bool {
	return itemId/1000000 == 1
}

func generateRandomEquipmentData(s statistics.Model) EquipmentData {
	return EquipmentData{
		Strength:      getRandomStat(s.Strength(), 5),
		Dexterity:     getRandomStat(s.Dexterity(), 5),
		Intelligence:  getRandomStat(s.Intelligence(), 5),
		Luck:          getRandomStat(s.Luck(), 5),
		Hp:            getRandomStat(s.Hp(), 10),
		Mp:            getRandomStat(s.Mp(), 10),
		WeaponAttack:  getRandomStat(s.WeaponAttack(), 5),
		MagicAttack:   getRandomStat(s.MagicAttack(), 5),
		WeaponDefense: getRandomStat(s.WeaponDefense(), 10),
		MagicDefense:  getRandomStat(s.MagicDefense(), 10),
		Accuracy:      getRandomStat(s.Accuracy(), 5),
		Avoidability:  getRandomStat(s.Avoidability(), 5),
		Hands:         getRandomStat(s.Hands(), 5),
		Speed:         getRandomStat(s.Speed(), 5),
		Jump:          getRandomStat(s.Jump(), 5),
		Slots:         s.Slots(),
	}
}

func getRandomStat(defaultValue uint16, max uint16) uint16 {
	if defaultValue == 0 {
		return 0
	}
	maxRange := math.Min(math.Ceil(float64(defaultValue)*0.1), float64(max))
	return uint16(float64(defaultValue)-maxRange) + uint16(math.Floor(rand.Float64()*(maxRange*2.0+1.0)))
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
