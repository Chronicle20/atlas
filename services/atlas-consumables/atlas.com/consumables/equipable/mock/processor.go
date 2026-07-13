package mock

import (
	"atlas-consumables/asset"
	"atlas-consumables/equipable"

	"github.com/google/uuid"
)

type ProcessorMock struct {
	ChangeStatFunc func(characterId uint32, transactionId uuid.UUID, a asset.Model, changes ...equipable.Change) error
}

var _ equipable.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) ChangeStat(characterId uint32, transactionId uuid.UUID, a asset.Model, changes ...equipable.Change) error {
	if m.ChangeStatFunc != nil {
		return m.ChangeStatFunc(characterId, transactionId, a, changes...)
	}
	return nil
}
