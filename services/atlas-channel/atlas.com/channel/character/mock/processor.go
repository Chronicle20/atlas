package mock

import (
	"atlas-channel/asset"
	"atlas-channel/character"
	"errors"

	inventory2 "github.com/Chronicle20/atlas-constants/inventory"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
)

// MockProcessor is a test double for character.Processor
type MockProcessor struct {
	// Characters stores characters by ID for lookup
	Characters map[uint32]character.Model
	// CharactersByName stores characters by name for lookup
	CharactersByName map[string]character.Model
	// Errors can be set to simulate failures
	GetByIdError   error
	GetByNameError error
}

// NewMockProcessor creates a new MockProcessor instance
func NewMockProcessor() *MockProcessor {
	return &MockProcessor{
		Characters:       make(map[uint32]character.Model),
		CharactersByName: make(map[string]character.Model),
	}
}

// AddCharacter adds a character to the mock for lookup
func (m *MockProcessor) AddCharacter(c character.Model) {
	m.Characters[c.Id()] = c
	m.CharactersByName[c.Name()] = c
}

func (m *MockProcessor) GetById(decorators ...model.Decorator[character.Model]) func(characterId uint32) (character.Model, error) {
	return func(characterId uint32) (character.Model, error) {
		if m.GetByIdError != nil {
			return character.Model{}, m.GetByIdError
		}
		c, ok := m.Characters[characterId]
		if !ok {
			return character.Model{}, errors.New("character not found")
		}
		for _, d := range decorators {
			c = d(c)
		}
		return c, nil
	}
}

func (m *MockProcessor) PetModelDecorator(c character.Model) character.Model {
	return c
}

func (m *MockProcessor) InventoryDecorator(c character.Model) character.Model {
	return c
}

func (m *MockProcessor) SkillModelDecorator(c character.Model) character.Model {
	return c
}

func (m *MockProcessor) GetEquipableInSlot(characterId uint32, slot int16) model.Provider[asset.Model[any]] {
	return model.ErrorProvider[asset.Model[any]](errors.New("not implemented in mock"))
}

func (m *MockProcessor) GetItemInSlot(characterId uint32, inventoryType inventory2.Type, slot int16) model.Provider[asset.Model[any]] {
	return model.ErrorProvider[asset.Model[any]](errors.New("not implemented in mock"))
}

func (m *MockProcessor) ByNameProvider(name string) model.Provider[[]character.Model] {
	return func() ([]character.Model, error) {
		if m.GetByNameError != nil {
			return nil, m.GetByNameError
		}
		c, ok := m.CharactersByName[name]
		if !ok {
			return []character.Model{}, nil
		}
		return []character.Model{c}, nil
	}
}

func (m *MockProcessor) GetByName(name string) (character.Model, error) {
	if m.GetByNameError != nil {
		return character.Model{}, m.GetByNameError
	}
	c, ok := m.CharactersByName[name]
	if !ok {
		return character.Model{}, errors.New("character not found")
	}
	return c, nil
}

func (m *MockProcessor) RequestDistributeAp(mapModel _map.Model, characterId uint32, updateTime uint32, distributes []character.DistributePacket) error {
	return nil
}

func (m *MockProcessor) RequestDropMeso(mapModel _map.Model, characterId uint32, amount uint32) error {
	return nil
}

func (m *MockProcessor) ChangeHP(mapModel _map.Model, characterId uint32, amount int16) error {
	return nil
}

func (m *MockProcessor) ChangeMP(mapModel _map.Model, characterId uint32, amount int16) error {
	return nil
}

func (m *MockProcessor) RequestDistributeSp(mapModel _map.Model, characterId uint32, updateTime uint32, skillId uint32, amount int8) error {
	return nil
}
