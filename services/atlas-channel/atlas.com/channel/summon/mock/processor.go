package mock

import (
	summon2 "atlas-channel/kafka/message/summon"
	"atlas-channel/summon"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	InMapModelProviderFunc func(f field.Model) model.Provider[[]summon.Model]
	ForEachInMapFunc       func(f field.Model, o model.Operator[summon.Model]) error
	SpawnFunc              func(f field.Model, ownerCharacterId uint32, skillId uint32, level byte, x int16, y int16, auraLevel byte, hexLevel byte) error
	MoveFunc               func(f field.Model, summonId uint32, senderCharacterId uint32, x int16, y int16, stance byte, rawMovement []byte) error
	AttackFunc             func(f field.Model, summonId uint32, senderCharacterId uint32, direction byte, targets []summon2.AttackTargetEntry) error
	DamageFunc             func(f field.Model, summonId uint32, senderCharacterId uint32, damage int32, monsterIdFrom uint32) error
}

var _ summon.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) InMapModelProvider(f field.Model) model.Provider[[]summon.Model] {
	if m.InMapModelProviderFunc != nil {
		return m.InMapModelProviderFunc(f)
	}
	return model.FixedProvider([]summon.Model{})
}

func (m *ProcessorMock) ForEachInMap(f field.Model, o model.Operator[summon.Model]) error {
	if m.ForEachInMapFunc != nil {
		return m.ForEachInMapFunc(f, o)
	}
	return nil
}

func (m *ProcessorMock) Spawn(f field.Model, ownerCharacterId uint32, skillId uint32, level byte, x int16, y int16, auraLevel byte, hexLevel byte) error {
	if m.SpawnFunc != nil {
		return m.SpawnFunc(f, ownerCharacterId, skillId, level, x, y, auraLevel, hexLevel)
	}
	return nil
}

func (m *ProcessorMock) Move(f field.Model, summonId uint32, senderCharacterId uint32, x int16, y int16, stance byte, rawMovement []byte) error {
	if m.MoveFunc != nil {
		return m.MoveFunc(f, summonId, senderCharacterId, x, y, stance, rawMovement)
	}
	return nil
}

func (m *ProcessorMock) Attack(f field.Model, summonId uint32, senderCharacterId uint32, direction byte, targets []summon2.AttackTargetEntry) error {
	if m.AttackFunc != nil {
		return m.AttackFunc(f, summonId, senderCharacterId, direction, targets)
	}
	return nil
}

func (m *ProcessorMock) Damage(f field.Model, summonId uint32, senderCharacterId uint32, damage int32, monsterIdFrom uint32) error {
	if m.DamageFunc != nil {
		return m.DamageFunc(f, summonId, senderCharacterId, damage, monsterIdFrom)
	}
	return nil
}
