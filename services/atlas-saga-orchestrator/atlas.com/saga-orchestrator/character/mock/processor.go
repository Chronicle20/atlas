package mock

import (
	"atlas-saga-orchestrator/kafka/message"
	character2 "atlas-saga-orchestrator/kafka/message/character"
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-constants/job"
	_map "github.com/Chronicle20/atlas-constants/map"
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
	AwardMesosAndEmitFunc      func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, actorId uint32, actorType string, amount int32) error
	AwardMesosFunc             func(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, actorId uint32, actorType string, amount int32) error
	ChangeJobAndEmitFunc       func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, jobId job.Id) error
	ChangeJobFunc              func(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, jobId job.Id) error
	RequestCreateCharacterFunc func(transactionId uuid.UUID, accountId uint32, worldId byte, name string, level byte, strength uint16, dexterity uint16, intelligence uint16, luck uint16, hp uint16, mp uint16, jobId job.Id, gender byte, face uint32, hair uint32, skin byte, mapId _map.Id) error
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

// AwardMesosAndEmit is a mock implementation of the character.Processor.AwardMesosAndEmit method
func (m *ProcessorMock) AwardMesosAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, actorId uint32, actorType string, amount int32) error {
	if m.AwardMesosAndEmitFunc != nil {
		return m.AwardMesosAndEmitFunc(transactionId, worldId, characterId, channelId, actorId, actorType, amount)
	}
	return nil
}

// AwardMesos is a mock implementation of the character.Processor.AwardMesos method
func (m *ProcessorMock) AwardMesos(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, actorId uint32, actorType string, amount int32) error {
	if m.AwardMesosFunc != nil {
		return m.AwardMesosFunc(mb)
	}
	return func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, actorId uint32, actorType string, amount int32) error {
		return nil
	}
}

// ChangeJobAndEmit is a mock implementation of the character.Processor.ChangeJobAndEmit method
func (m *ProcessorMock) ChangeJobAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, jobId job.Id) error {
	if m.ChangeJobAndEmitFunc != nil {
		return m.ChangeJobAndEmitFunc(transactionId, worldId, characterId, channelId, jobId)
	}
	return nil
}

// ChangeJob is a mock implementation of the character.Processor.ChangeJob method
func (m *ProcessorMock) ChangeJob(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, jobId job.Id) error {
	if m.ChangeJobFunc != nil {
		return m.ChangeJobFunc(mb)
	}
	return func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, jobId job.Id) error {
		return nil
	}
}

// RequestCreateCharacter is a mock implementation of the character.Processor.RequestCreateCharacter method
func (m *ProcessorMock) RequestCreateCharacter(transactionId uuid.UUID, accountId uint32, worldId byte, name string, level byte, strength uint16, dexterity uint16, intelligence uint16, luck uint16, hp uint16, mp uint16, jobId job.Id, gender byte, face uint32, hair uint32, skin byte, mapId _map.Id) error {
	if m.RequestCreateCharacterFunc != nil {
		return m.RequestCreateCharacterFunc(transactionId, accountId, worldId, name, level, strength, dexterity, intelligence, luck, hp, mp, jobId, gender, face, hair, skin, mapId)
	}
	return nil
}
