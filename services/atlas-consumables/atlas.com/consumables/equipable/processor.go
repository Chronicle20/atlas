package equipable

import (
	"atlas-consumables/asset"
	"atlas-consumables/kafka/message/compartment"
	"atlas-consumables/kafka/producer"
	"context"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	p := &Processor{
		l:   l,
		ctx: ctx,
	}
	return p
}

func (p *Processor) ChangeStat(characterId uint32, transactionId uuid.UUID, a asset.Model, changes ...Change) error {
	b := asset.Clone(a)
	for _, c := range changes {
		c(b)
	}
	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)(modifyEquipmentProvider(characterId, transactionId, b.Build()))
}

func AddStrength(amount int16) Change {
	return func(m *asset.ModelBuilder) {
		m.AddStrength(amount)
	}
}

func AddDexterity(amount int16) Change {
	return func(m *asset.ModelBuilder) {
		m.AddDexterity(amount)
	}
}

func AddIntelligence(amount int16) Change {
	return func(m *asset.ModelBuilder) {
		m.AddIntelligence(amount)
	}
}

func AddLuck(amount int16) Change {
	return func(m *asset.ModelBuilder) {
		m.AddLuck(amount)
	}
}

func AddHp(amount int16) Change {
	return func(m *asset.ModelBuilder) {
		m.AddHp(amount)
	}
}

func AddMp(amount int16) Change {
	return func(m *asset.ModelBuilder) {
		m.AddMp(amount)
	}
}

func AddWeaponAttack(amount int16) Change {
	return func(m *asset.ModelBuilder) {
		m.AddWeaponAttack(amount)
	}
}

func AddMagicAttack(amount int16) Change {
	return func(m *asset.ModelBuilder) {
		m.AddMagicAttack(amount)
	}
}

func AddWeaponDefense(amount int16) Change {
	return func(m *asset.ModelBuilder) {
		m.AddWeaponDefense(amount)
	}
}

func AddMagicDefense(amount int16) Change {
	return func(m *asset.ModelBuilder) {
		m.AddMagicDefense(amount)
	}
}

func AddAccuracy(amount int16) Change {
	return func(m *asset.ModelBuilder) {
		m.AddAccuracy(amount)
	}
}

func AddAvoidability(amount int16) Change {
	return func(m *asset.ModelBuilder) {
		m.AddAvoidability(amount)
	}
}

func AddHands(amount int16) Change {
	return func(m *asset.ModelBuilder) {
		m.AddHands(amount)
	}
}

func AddSpeed(amount int16) Change {
	return func(m *asset.ModelBuilder) {
		m.AddSpeed(amount)
	}
}

func AddJump(amount int16) Change {
	return func(m *asset.ModelBuilder) {
		m.AddJump(amount)
	}
}

func AddSlots(amount int16) Change {
	return func(m *asset.ModelBuilder) {
		m.AddSlots(amount)
	}
}

func AddLevel(amount int8) Change {
	return func(m *asset.ModelBuilder) {
		m.AddLevel(amount)
	}
}

func SetSpike() Change {
	return func(m *asset.ModelBuilder) {
		m.SetSpikes(true)
	}
}

func SetCold() Change {
	return func(m *asset.ModelBuilder) {
		m.SetCold(true)
	}
}
