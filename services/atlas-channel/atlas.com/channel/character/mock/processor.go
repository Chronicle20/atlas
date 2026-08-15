package mock

import (
	"atlas-channel/asset"
	"atlas-channel/character"
	character2 "atlas-channel/kafka/message/character"
	"errors"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	inventory2 "github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
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
	// GetItemInSlotFunc lets a test control the target-item lookup used by,
	// e.g., the karma scissors handler arm (task-223), which needs the full
	// asset (locked, flag, template id) rather than just a resolved template
	// id. Defaults to the pre-existing "not implemented" error when unset.
	GetItemInSlotFunc func(characterId uint32, inventoryType inventory2.Type, slot int16) (asset.Model, error)
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

func (m *MockProcessor) InventoryDecorator(c character.Model) character.Model {
	return c
}

func (m *MockProcessor) PetAssetEnrichmentDecorator(c character.Model) character.Model {
	return c
}

func (m *MockProcessor) SkillModelDecorator(c character.Model) character.Model {
	return c
}

func (m *MockProcessor) QuestModelDecorator(c character.Model) character.Model {
	return c
}

func (m *MockProcessor) PartyDecorator(c character.Model) character.Model {
	return c
}

func (m *MockProcessor) MonsterBookDecorator(c character.Model) character.Model {
	return c
}

func (m *MockProcessor) GetEquipableInSlot(_ uint32, _ int16) model.Provider[asset.Model] {
	return model.ErrorProvider[asset.Model](errors.New("not implemented in mock"))
}

func (m *MockProcessor) GetItemInSlot(characterId uint32, inventoryType inventory2.Type, slot int16) model.Provider[asset.Model] {
	if m.GetItemInSlotFunc != nil {
		return func() (asset.Model, error) { return m.GetItemInSlotFunc(characterId, inventoryType, slot) }
	}
	return model.ErrorProvider[asset.Model](errors.New("not implemented in mock"))
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

func (m *MockProcessor) RequestDistributeAp(_ field.Model, _ uint32, _ uint32, _ []character.DistributePacket) error {
	return nil
}

func (m *MockProcessor) RequestDropMeso(_ field.Model, _ uint32, _ uint32) error {
	return nil
}

func (m *MockProcessor) RequestChangeMeso(_ field.Model, _ uint32, _ uint32, _ string, _ int32) error {
	return nil
}

func (m *MockProcessor) ChangeHP(_ field.Model, _ uint32, _ int16) error {
	return nil
}

func (m *MockProcessor) SetHP(_ field.Model, _ uint32, _ uint16) error {
	return nil
}

func (m *MockProcessor) ChangeMP(_ field.Model, _ uint32, _ int16) error {
	return nil
}

func (m *MockProcessor) RequestDistributeSp(_ field.Model, _ uint32, _ uint32, _ uint32, _ int8) error {
	return nil
}

func (m *MockProcessor) AwardExperience(_ field.Model, _ uint32, _ []character2.ExperienceDistributions, _ bool) error {
	return nil
}
