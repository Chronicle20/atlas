package mock

import (
	"atlas-trades/data/character"

	characterconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// ProcessorMock is the injectable double for the character REST client. Each
// Func field defaults to a zero-valued success when left unset.
type ProcessorMock struct {
	GetByIdFunc      func(characterId characterconst.Id) (character.Model, error)
	ByIdProviderFunc func(characterId characterconst.Id) model.Provider[character.Model]
	HpFunc           func(characterId characterconst.Id) (uint16, error)
	LevelFunc        func(characterId characterconst.Id) (byte, error)
	NameFunc         func(characterId characterconst.Id) (string, error)
	MesoFunc         func(characterId characterconst.Id) (uint32, error)
}

func (m *ProcessorMock) GetById(characterId characterconst.Id) (character.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(characterId)
	}
	return character.Model{}, nil
}

func (m *ProcessorMock) ByIdProvider(characterId characterconst.Id) model.Provider[character.Model] {
	if m.ByIdProviderFunc != nil {
		return m.ByIdProviderFunc(characterId)
	}
	return model.FixedProvider(character.Model{})
}

func (m *ProcessorMock) Hp(characterId characterconst.Id) (uint16, error) {
	if m.HpFunc != nil {
		return m.HpFunc(characterId)
	}
	return 0, nil
}

func (m *ProcessorMock) Level(characterId characterconst.Id) (byte, error) {
	if m.LevelFunc != nil {
		return m.LevelFunc(characterId)
	}
	return 0, nil
}

func (m *ProcessorMock) Name(characterId characterconst.Id) (string, error) {
	if m.NameFunc != nil {
		return m.NameFunc(characterId)
	}
	return "", nil
}

func (m *ProcessorMock) Meso(characterId characterconst.Id) (uint32, error) {
	if m.MesoFunc != nil {
		return m.MesoFunc(characterId)
	}
	return 0, nil
}

var _ character.Processor = (*ProcessorMock)(nil)
