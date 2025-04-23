package equipable

import (
	"atlas-consumables/asset"
	"atlas-consumables/kafka/message/equipable"
	"atlas-consumables/kafka/producer"
	"context"
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

func (p *Processor) ChangeStat(a asset.Model[asset.EquipableReferenceData], changes ...Change) error {
	b := asset.Clone(a)
	rd := asset.NewEquipableReferenceDataBuilder().Clone(a.ReferenceData())
	for _, c := range changes {
		c(rd)
	}
	b.SetReferenceData(rd.Build())
	return producer.ProviderImpl(p.l)(p.ctx)(equipable.EnvCommandTopic)(changeEquipableProvider(b.Build()))
}

func AddStrength(amount int16) Change {
	return func(m *asset.EquipableReferenceDataBuilder) {
		m.AddStrength(amount)
	}
}

func AddDexterity(amount int16) Change {
	return func(m *asset.EquipableReferenceDataBuilder) {
		m.AddDexterity(amount)
	}
}

func AddIntelligence(amount int16) Change {
	return func(m *asset.EquipableReferenceDataBuilder) {
		m.AddIntelligence(amount)
	}
}

func AddLuck(amount int16) Change {
	return func(m *asset.EquipableReferenceDataBuilder) {
		m.AddLuck(amount)
	}
}

func AddHP(amount int16) Change {
	return func(m *asset.EquipableReferenceDataBuilder) {
		m.AddHp(amount)
	}
}

func AddMP(amount int16) Change {
	return func(m *asset.EquipableReferenceDataBuilder) {
		m.AddMp(amount)
	}
}

func AddWeaponAttack(amount int16) Change {
	return func(m *asset.EquipableReferenceDataBuilder) {
		m.AddWeaponAttack(amount)
	}
}

func AddMagicAttack(amount int16) Change {
	return func(m *asset.EquipableReferenceDataBuilder) {
		m.AddMagicAttack(amount)
	}
}

func AddWeaponDefense(amount int16) Change {
	return func(m *asset.EquipableReferenceDataBuilder) {
		m.AddWeaponDefense(amount)
	}
}

func AddMagicDefense(amount int16) Change {
	return func(m *asset.EquipableReferenceDataBuilder) {
		m.AddMagicDefense(amount)
	}
}

func AddAccuracy(amount int16) Change {
	return func(m *asset.EquipableReferenceDataBuilder) {
		m.AddAccuracy(amount)
	}
}

func AddAvoidability(amount int16) Change {
	return func(m *asset.EquipableReferenceDataBuilder) {
		m.AddAvoidability(amount)
	}
}

func AddHands(amount int16) Change {
	return func(m *asset.EquipableReferenceDataBuilder) {
		m.AddHands(amount)
	}
}

func AddSpeed(amount int16) Change {
	return func(m *asset.EquipableReferenceDataBuilder) {
		m.AddSpeed(amount)
	}
}

func AddJump(amount int16) Change {
	return func(m *asset.EquipableReferenceDataBuilder) {
		m.AddJump(amount)
	}
}

func AddSlots(amount int16) Change {
	return func(m *asset.EquipableReferenceDataBuilder) {
		m.AddSlots(amount)
	}
}

func AddLevel(amount int8) Change {
	return func(m *asset.EquipableReferenceDataBuilder) {
		m.AddLevel(amount)
	}
}

func SetSpike() Change {
	return func(m *asset.EquipableReferenceDataBuilder) {
		m.SetSpikes(true)
	}
}

func SetCold() Change {
	return func(m *asset.EquipableReferenceDataBuilder) {
		m.SetCold(true)
	}
}
