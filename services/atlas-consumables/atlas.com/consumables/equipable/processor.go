package equipable

import (
	"atlas-consumables/kafka/message/equipable"
	"atlas-consumables/kafka/producer"
	"context"
	"github.com/sirupsen/logrus"
)

func ChangeStat(l logrus.FieldLogger) func(ctx context.Context) func(e Model, changes ...Change) error {
	return func(ctx context.Context) func(e Model, changes ...Change) error {
		return func(e Model, changes ...Change) error {
			b := Clone(e)
			for _, c := range changes {
				c(b)
			}
			return producer.ProviderImpl(l)(ctx)(equipable.EnvCommandTopic)(changeEquipableProvider(b.Build()))
		}
	}
}

func AddStrength(amount int16) Change {
	return func(m *ModelBuilder) {
		m.AddStrength(amount)
	}
}

func AddDexterity(amount int16) Change {
	return func(m *ModelBuilder) {
		m.AddDexterity(amount)
	}
}

func AddIntelligence(amount int16) Change {
	return func(m *ModelBuilder) {
		m.AddIntelligence(amount)
	}
}

func AddLuck(amount int16) Change {
	return func(m *ModelBuilder) {
		m.AddLuck(amount)
	}
}

func AddHP(amount int16) Change {
	return func(m *ModelBuilder) {
		m.AddHp(amount)
	}
}

func AddMP(amount int16) Change {
	return func(m *ModelBuilder) {
		m.AddMp(amount)
	}
}

func AddWeaponAttack(amount int16) Change {
	return func(m *ModelBuilder) {
		m.AddWeaponAttack(amount)
	}
}

func AddMagicAttack(amount int16) Change {
	return func(m *ModelBuilder) {
		m.AddMagicAttack(amount)
	}
}

func AddWeaponDefense(amount int16) Change {
	return func(m *ModelBuilder) {
		m.AddWeaponDefense(amount)
	}
}

func AddMagicDefense(amount int16) Change {
	return func(m *ModelBuilder) {
		m.AddMagicDefense(amount)
	}
}

func AddAccuracy(amount int16) Change {
	return func(m *ModelBuilder) {
		m.AddAccuracy(amount)
	}
}

func AddAvoidability(amount int16) Change {
	return func(m *ModelBuilder) {
		m.AddAvoidability(amount)
	}
}

func AddHands(amount int16) Change {
	return func(m *ModelBuilder) {
		m.AddHands(amount)
	}
}

func AddSpeed(amount int16) Change {
	return func(m *ModelBuilder) {
		m.AddSpeed(amount)
	}
}

func AddJump(amount int16) Change {
	return func(m *ModelBuilder) {
		m.AddJump(amount)
	}
}

func AddSlots(amount int16) Change {
	return func(m *ModelBuilder) {
		m.AddSlots(amount)
	}
}

func AddLevel(amount int8) Change {
	return func(m *ModelBuilder) {
		m.AddLevel(amount)
	}
}

func SetSpike() Change {
	return func(m *ModelBuilder) {
		m.SetSpikes(true)
	}
}

func SetCold() Change {
	return func(m *ModelBuilder) {
		m.SetCold(true)
	}
}
