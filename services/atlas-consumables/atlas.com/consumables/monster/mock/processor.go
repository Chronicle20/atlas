package mock

import (
	"atlas-consumables/monster"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

type ProcessorMock struct {
	CreateMonsterFunc func(f field.Model, monsterId uint32, x int16, y int16, fh int16, team int8) error
}

var _ monster.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) CreateMonster(f field.Model, monsterId uint32, x int16, y int16, fh int16, team int8) error {
	if m.CreateMonsterFunc != nil {
		return m.CreateMonsterFunc(f, monsterId, x, y, fh, team)
	}
	return nil
}
