package mock

import (
	"atlas-asset-expiration/character"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

type ProcessorMock struct {
	CheckAndExpireFunc func(pp producer.Provider) func(characterId, accountId uint32, worldId world.Id)
}

var _ character.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) CheckAndExpire(pp producer.Provider) func(characterId, accountId uint32, worldId world.Id) {
	if m.CheckAndExpireFunc != nil {
		return m.CheckAndExpireFunc(pp)
	}
	return func(characterId, accountId uint32, worldId world.Id) {}
}
