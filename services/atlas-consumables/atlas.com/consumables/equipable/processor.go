package equipable

import (
	"atlas-consumables/asset"
	"atlas-consumables/kafka/message/compartment"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	ChangeStat(characterId uint32, transactionId uuid.UUID, a asset.Model, changes ...Change) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
	return p
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) ChangeStat(characterId uint32, transactionId uuid.UUID, a asset.Model, changes ...Change) error {
	b := asset.Clone(a)
	for _, c := range changes {
		c(b)
	}
	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)(modifyEquipmentProvider(characterId, transactionId, b.Build()))
}

func AddStrength(amount int16) Change {
	return func(m *asset.Builder) {
		m.AddStrength(amount)
	}
}

func AddDexterity(amount int16) Change {
	return func(m *asset.Builder) {
		m.AddDexterity(amount)
	}
}

func AddIntelligence(amount int16) Change {
	return func(m *asset.Builder) {
		m.AddIntelligence(amount)
	}
}

func AddLuck(amount int16) Change {
	return func(m *asset.Builder) {
		m.AddLuck(amount)
	}
}

func AddHp(amount int16) Change {
	return func(m *asset.Builder) {
		m.AddHp(amount)
	}
}

func AddMp(amount int16) Change {
	return func(m *asset.Builder) {
		m.AddMp(amount)
	}
}

func AddWeaponAttack(amount int16) Change {
	return func(m *asset.Builder) {
		m.AddWeaponAttack(amount)
	}
}

func AddMagicAttack(amount int16) Change {
	return func(m *asset.Builder) {
		m.AddMagicAttack(amount)
	}
}

func AddWeaponDefense(amount int16) Change {
	return func(m *asset.Builder) {
		m.AddWeaponDefense(amount)
	}
}

func AddMagicDefense(amount int16) Change {
	return func(m *asset.Builder) {
		m.AddMagicDefense(amount)
	}
}

func AddAccuracy(amount int16) Change {
	return func(m *asset.Builder) {
		m.AddAccuracy(amount)
	}
}

func AddAvoidability(amount int16) Change {
	return func(m *asset.Builder) {
		m.AddAvoidability(amount)
	}
}

func AddHands(amount int16) Change {
	return func(m *asset.Builder) {
		m.AddHands(amount)
	}
}

func AddSpeed(amount int16) Change {
	return func(m *asset.Builder) {
		m.AddSpeed(amount)
	}
}

func AddJump(amount int16) Change {
	return func(m *asset.Builder) {
		m.AddJump(amount)
	}
}

func AddSlots(amount int16) Change {
	return func(m *asset.Builder) {
		m.AddSlots(amount)
	}
}

func AddLevel(amount int8) Change {
	return func(m *asset.Builder) {
		m.AddLevel(amount)
	}
}

func AddHammersApplied(amount int32) Change {
	return func(m *asset.Builder) {
		m.AddHammersApplied(amount)
	}
}

func SetSpike() Change {
	return func(m *asset.Builder) {
		m.SetSpikes(true)
	}
}

func SetCold() Change {
	return func(m *asset.Builder) {
		m.SetCold(true)
	}
}
