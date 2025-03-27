package equipable

import (
	"atlas-consumables/kafka/message/equipable"
	"atlas-consumables/kafka/producer"
	"context"
	"github.com/sirupsen/logrus"
)

func AddStrength(amount int16) func(m *equipable.ChangeBody) {
	return func(m *equipable.ChangeBody) {
		m.Strength += amount
	}
}

func AddDexterity(amount int16) func(m *equipable.ChangeBody) {
	return func(m *equipable.ChangeBody) {
		m.Dexterity += amount
	}
}

func AddIntelligence(amount int16) func(m *equipable.ChangeBody) {
	return func(m *equipable.ChangeBody) {
		m.Intelligence += amount
	}
}

func AddLuck(amount int16) func(m *equipable.ChangeBody) {
	return func(m *equipable.ChangeBody) {
		m.Luck += amount
	}
}

func AddHP(amount int16) func(m *equipable.ChangeBody) {
	return func(m *equipable.ChangeBody) {
		m.HP += amount
	}
}

func AddMP(amount int16) func(m *equipable.ChangeBody) {
	return func(m *equipable.ChangeBody) {
		m.MP += amount
	}
}

func AddWeaponAttack(amount int16) func(m *equipable.ChangeBody) {
	return func(m *equipable.ChangeBody) {
		m.WeaponAttack += amount
	}
}

func AddMagicAttack(amount int16) func(m *equipable.ChangeBody) {
	return func(m *equipable.ChangeBody) {
		m.MagicAttack += amount
	}
}

func AddWeaponDefense(amount int16) func(m *equipable.ChangeBody) {
	return func(m *equipable.ChangeBody) {
		m.WeaponDefense += amount
	}
}

func AddMagicDefense(amount int16) func(m *equipable.ChangeBody) {
	return func(m *equipable.ChangeBody) {
		m.MagicDefense += amount
	}
}

func AddAccuracy(amount int16) func(m *equipable.ChangeBody) {
	return func(m *equipable.ChangeBody) {
		m.Accuracy += amount
	}
}

func AddAvoidability(amount int16) func(m *equipable.ChangeBody) {
	return func(m *equipable.ChangeBody) {
		m.Avoidability += amount
	}
}

func AddHands(amount int16) func(m *equipable.ChangeBody) {
	return func(m *equipable.ChangeBody) {
		m.Hands += amount
	}
}

func AddSpeed(amount int16) func(m *equipable.ChangeBody) {
	return func(m *equipable.ChangeBody) {
		m.Speed += amount
	}
}

func AddJump(amount int16) func(m *equipable.ChangeBody) {
	return func(m *equipable.ChangeBody) {
		m.Jump += amount
	}
}

func AddSlots(amount int16) func(m *equipable.ChangeBody) {
	return func(m *equipable.ChangeBody) {
		m.Slots += amount
	}
}

func AddLevel(amount int8) func(m *equipable.ChangeBody) {
	return func(m *equipable.ChangeBody) {
		m.Level += amount
	}
}

func ChangeStat(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, itemId uint32, slot int16, changes ...Change) error {
	return func(ctx context.Context) func(characterId uint32, itemId uint32, slot int16, changes ...Change) error {
		return func(characterId uint32, itemId uint32, slot int16, changes ...Change) error {
			return producer.ProviderImpl(l)(ctx)(equipable.EnvCommandTopic)(changeEquipableProvider(characterId, itemId, slot, changes...))
		}
	}
}
