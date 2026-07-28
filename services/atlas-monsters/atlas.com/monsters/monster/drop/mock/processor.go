package mock

import (
	"atlas-monsters/monster/drop"
)

type ProcessorMock struct {
	GetByMonsterIdFunc func(monsterId uint32) ([]drop.Model, error)
}

var _ drop.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetByMonsterId(monsterId uint32) ([]drop.Model, error) {
	if m.GetByMonsterIdFunc != nil {
		return m.GetByMonsterIdFunc(monsterId)
	}
	return nil, nil
}
