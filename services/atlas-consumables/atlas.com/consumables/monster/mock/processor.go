package mock

import (
	"atlas-consumables/monster"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

type ProcessorMock struct {
	CreateMonsterFunc func(f field.Model, monsterId uint32, x int16, y int16, fh int16, team int8) error
	RequestCatchFunc  func(f field.Model, monsterUniqueId uint32, characterId uint32, itemId uint32) error
}

var _ monster.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) CreateMonster(f field.Model, monsterId uint32, x int16, y int16, fh int16, team int8) error {
	if m.CreateMonsterFunc != nil {
		return m.CreateMonsterFunc(f, monsterId, x, y, fh, team)
	}
	return nil
}

func (m *ProcessorMock) RequestCatch(f field.Model, monsterUniqueId uint32, characterId uint32, itemId uint32) error {
	if m.RequestCatchFunc != nil {
		return m.RequestCatchFunc(f, monsterUniqueId, characterId, itemId)
	}
	return nil
}
