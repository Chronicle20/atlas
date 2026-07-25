package mock

import (
	"atlas-channel/monster"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	GetByIdFunc                func(uniqueId uint32) (monster.Model, error)
	InMapModelProviderFunc     func(f field.Model) model.Provider[[]monster.Model]
	ForEachInMapFunc           func(f field.Model, o model.Operator[monster.Model]) error
	GetInMapFunc               func(f field.Model) ([]monster.Model, error)
	InMapRectModelProviderFunc func(f field.Model, x1, y1, x2, y2 int16, limit uint32) model.Provider[[]monster.Model]
	GetInMapRectFunc           func(f field.Model, x1, y1, x2, y2 int16, limit uint32) ([]monster.Model, error)
	DamageFunc                 func(f field.Model, monsterId uint32, characterId uint32, damages []uint32, attackType byte) error
	EmitDamageReflectedFunc    func(f field.Model, uniqueId uint32, templateId uint32, characterId uint32, reflectDamage uint32, reflectType string) error
	UseSkillFunc               func(f field.Model, monsterId uint32, characterId uint32, skillId byte, skillLevel byte) error
	UseBasicAttackFunc         func(f field.Model, monsterId uint32, attackPos uint8) error
	ApplyStatusFunc            func(f field.Model, monsterId uint32, characterId uint32, skillId uint32, skillLevel uint32, statuses map[string]int32, duration uint32) error
	DamageFriendlyFunc         func(f field.Model, attackedUniqueId uint32, observerUniqueId, attackerUniqueId uint32) error
	CancelStatusFunc           func(f field.Model, monsterId uint32, statusTypes []string, sourceCharacterId uint32, sourceSkillId uint32, sourceSkillClass string) error
	DrainMpFunc                func(f field.Model, monsterId uint32, characterId uint32, skillId uint32, amount uint32) error
}

var _ monster.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetById(uniqueId uint32) (monster.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(uniqueId)
	}
	return monster.Model{}, nil
}

func (m *ProcessorMock) InMapModelProvider(f field.Model) model.Provider[[]monster.Model] {
	if m.InMapModelProviderFunc != nil {
		return m.InMapModelProviderFunc(f)
	}
	return model.FixedProvider([]monster.Model{})
}

func (m *ProcessorMock) ForEachInMap(f field.Model, o model.Operator[monster.Model]) error {
	if m.ForEachInMapFunc != nil {
		return m.ForEachInMapFunc(f, o)
	}
	return nil
}

func (m *ProcessorMock) GetInMap(f field.Model) ([]monster.Model, error) {
	if m.GetInMapFunc != nil {
		return m.GetInMapFunc(f)
	}
	return nil, nil
}

func (m *ProcessorMock) InMapRectModelProvider(f field.Model, x1, y1, x2, y2 int16, limit uint32) model.Provider[[]monster.Model] {
	if m.InMapRectModelProviderFunc != nil {
		return m.InMapRectModelProviderFunc(f, x1, y1, x2, y2, limit)
	}
	return model.FixedProvider([]monster.Model{})
}

func (m *ProcessorMock) GetInMapRect(f field.Model, x1, y1, x2, y2 int16, limit uint32) ([]monster.Model, error) {
	if m.GetInMapRectFunc != nil {
		return m.GetInMapRectFunc(f, x1, y1, x2, y2, limit)
	}
	return nil, nil
}

func (m *ProcessorMock) Damage(f field.Model, monsterId uint32, characterId uint32, damages []uint32, attackType byte) error {
	if m.DamageFunc != nil {
		return m.DamageFunc(f, monsterId, characterId, damages, attackType)
	}
	return nil
}

func (m *ProcessorMock) EmitDamageReflected(f field.Model, uniqueId uint32, templateId uint32, characterId uint32, reflectDamage uint32, reflectType string) error {
	if m.EmitDamageReflectedFunc != nil {
		return m.EmitDamageReflectedFunc(f, uniqueId, templateId, characterId, reflectDamage, reflectType)
	}
	return nil
}

func (m *ProcessorMock) UseSkill(f field.Model, monsterId uint32, characterId uint32, skillId byte, skillLevel byte) error {
	if m.UseSkillFunc != nil {
		return m.UseSkillFunc(f, monsterId, characterId, skillId, skillLevel)
	}
	return nil
}

func (m *ProcessorMock) UseBasicAttack(f field.Model, monsterId uint32, attackPos uint8) error {
	if m.UseBasicAttackFunc != nil {
		return m.UseBasicAttackFunc(f, monsterId, attackPos)
	}
	return nil
}

func (m *ProcessorMock) ApplyStatus(f field.Model, monsterId uint32, characterId uint32, skillId uint32, skillLevel uint32, statuses map[string]int32, duration uint32) error {
	if m.ApplyStatusFunc != nil {
		return m.ApplyStatusFunc(f, monsterId, characterId, skillId, skillLevel, statuses, duration)
	}
	return nil
}

func (m *ProcessorMock) DamageFriendly(f field.Model, attackedUniqueId uint32, observerUniqueId, attackerUniqueId uint32) error {
	if m.DamageFriendlyFunc != nil {
		return m.DamageFriendlyFunc(f, attackedUniqueId, observerUniqueId, attackerUniqueId)
	}
	return nil
}

func (m *ProcessorMock) CancelStatus(f field.Model, monsterId uint32, statusTypes []string, sourceCharacterId uint32, sourceSkillId uint32, sourceSkillClass string) error {
	if m.CancelStatusFunc != nil {
		return m.CancelStatusFunc(f, monsterId, statusTypes, sourceCharacterId, sourceSkillId, sourceSkillClass)
	}
	return nil
}

func (m *ProcessorMock) DrainMp(f field.Model, monsterId uint32, characterId uint32, skillId uint32, amount uint32) error {
	if m.DrainMpFunc != nil {
		return m.DrainMpFunc(f, monsterId, characterId, skillId, amount)
	}
	return nil
}
