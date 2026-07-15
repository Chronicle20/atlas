package mock

import (
	"atlas-monster-death/monster"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

type ProcessorMock struct {
	CreateDropsFunc          func(f field.Model, id uint32, monsterId uint32, x int16, y int16, killerId uint32) error
	DistributeExperienceFunc func(f field.Model, monsterId uint32, damageEntries []monster.DamageEntryModel) error
}

var _ monster.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) CreateDrops(f field.Model, id uint32, monsterId uint32, x int16, y int16, killerId uint32) error {
	if m.CreateDropsFunc != nil {
		return m.CreateDropsFunc(f, id, monsterId, x, y, killerId)
	}
	return nil
}

func (m *ProcessorMock) DistributeExperience(f field.Model, monsterId uint32, damageEntries []monster.DamageEntryModel) error {
	if m.DistributeExperienceFunc != nil {
		return m.DistributeExperienceFunc(f, monsterId, damageEntries)
	}
	return nil
}
