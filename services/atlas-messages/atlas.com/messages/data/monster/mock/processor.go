package mock

import "atlas-messages/data/monster"

type Processor struct {
	GetByIdFn func(monsterId uint32) (monster.Model, error)
}

func (m *Processor) GetById(monsterId uint32) (monster.Model, error) {
	if m.GetByIdFn != nil {
		return m.GetByIdFn(monsterId)
	}
	return monster.Model{}, nil
}
