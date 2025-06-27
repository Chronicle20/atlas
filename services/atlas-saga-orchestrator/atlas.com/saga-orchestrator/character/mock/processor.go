package mock

import (
	"atlas-saga-orchestrator/kafka/message"
	character2 "atlas-saga-orchestrator/kafka/message/character"
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
)

// ProcessorMock is a mock implementation of the character.Processor interface
type ProcessorMock struct {
	WarpRandomAndEmitFunc      func(transactionId uuid.UUID, characterId uint32, field field.Model) error
	WarpRandomFunc             func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, field field.Model) error
	WarpToPortalAndEmitFunc    func(transactionId uuid.UUID, characterId uint32, field field.Model, pp model.Provider[uint32]) error
	WarpToPortalFunc           func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, field field.Model, pp model.Provider[uint32]) error
	AwardExperienceAndEmitFunc func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, distributions []character2.ExperienceDistributions) error
	AwardExperienceFunc        func(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, distributions []character2.ExperienceDistributions) error
	AwardLevelAndEmitFunc      func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, amount byte) error
	AwardLevelFunc             func(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, amount byte) error
}

// WarpRandomAndEmit is a mock implementation of the character.Processor.WarpRandomAndEmit method
func (m *ProcessorMock) WarpRandomAndEmit(transactionId uuid.UUID, characterId uint32, field field.Model) error {
	if m.WarpRandomAndEmitFunc != nil {
		return m.WarpRandomAndEmitFunc(transactionId, characterId, field)
	}
	return nil
}

// WarpRandom is a mock implementation of the character.Processor.WarpRandom method
func (m *ProcessorMock) WarpRandom(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, field field.Model) error {
	if m.WarpRandomFunc != nil {
		return m.WarpRandomFunc(mb)
	}
	return func(transactionId uuid.UUID, characterId uint32, field field.Model) error {
		return nil
	}
}

// WarpToPortalAndEmit is a mock implementation of the character.Processor.WarpToPortalAndEmit method
func (m *ProcessorMock) WarpToPortalAndEmit(transactionId uuid.UUID, characterId uint32, field field.Model, pp model.Provider[uint32]) error {
	if m.WarpToPortalAndEmitFunc != nil {
		return m.WarpToPortalAndEmitFunc(transactionId, characterId, field, pp)
	}
	return nil
}

// WarpToPortal is a mock implementation of the character.Processor.WarpToPortal method
func (m *ProcessorMock) WarpToPortal(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, field field.Model, pp model.Provider[uint32]) error {
	if m.WarpToPortalFunc != nil {
		return m.WarpToPortalFunc(mb)
	}
	return func(transactionId uuid.UUID, characterId uint32, field field.Model, pp model.Provider[uint32]) error {
		return nil
	}
}

// AwardExperienceAndEmit is a mock implementation of the character.Processor.AwardExperienceAndEmit method
func (m *ProcessorMock) AwardExperienceAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, distributions []character2.ExperienceDistributions) error {
	if m.AwardExperienceAndEmitFunc != nil {
		return m.AwardExperienceAndEmitFunc(transactionId, worldId, characterId, channelId, distributions)
	}
	return nil
}

// AwardExperience is a mock implementation of the character.Processor.AwardExperience method
func (m *ProcessorMock) AwardExperience(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, distributions []character2.ExperienceDistributions) error {
	if m.AwardExperienceFunc != nil {
		return m.AwardExperienceFunc(mb)
	}
	return func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, distributions []character2.ExperienceDistributions) error {
		return nil
	}
}

// AwardLevelAndEmit is a mock implementation of the character.Processor.AwardLevelAndEmit method
func (m *ProcessorMock) AwardLevelAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, amount byte) error {
	if m.AwardLevelAndEmitFunc != nil {
		return m.AwardLevelAndEmitFunc(transactionId, worldId, characterId, channelId, amount)
	}
	return nil
}

// AwardLevel is a mock implementation of the character.Processor.AwardLevel method
func (m *ProcessorMock) AwardLevel(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, amount byte) error {
	if m.AwardLevelFunc != nil {
		return m.AwardLevelFunc(mb)
	}
	return func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, amount byte) error {
		return nil
	}
}