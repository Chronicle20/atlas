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
	AwardExperienceAndEmitFunc  func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, distributions []character2.ExperienceDistributions) error
	AwardExperienceFunc         func(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, distributions []character2.ExperienceDistributions) error
	DeductExperienceAndEmitFunc func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, amount uint32) error
	DeductExperienceFunc        func(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, amount uint32) error
	AwardLevelAndEmitFunc       func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, amount byte) error
	AwardLevelFunc             func(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, amount byte) error
	AwardMesosAndEmitFunc      func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, actorId uint32, actorType string, amount int32) error
	AwardMesosFunc             func(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, actorId uint32, actorType string, amount int32) error
	AwardFameAndEmitFunc       func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, amount int16) error
	AwardFameFunc              func(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, amount int16) error
	ChangeJobAndEmitFunc       func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, jobId job.Id) error
	ChangeJobFunc              func(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, jobId job.Id) error
	ChangeHairAndEmitFunc      func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, styleId uint32) error
	ChangeHairFunc             func(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, styleId uint32) error
	ChangeFaceAndEmitFunc      func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, styleId uint32) error
	ChangeFaceFunc             func(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, styleId uint32) error
	ChangeSkinAndEmitFunc      func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, styleId byte) error
	ChangeSkinFunc             func(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, styleId byte) error
	RequestCreateCharacterFunc func(transactionId uuid.UUID, accountId uint32, worldId world.Id, name string, level byte, strength uint16, dexterity uint16, intelligence uint16, luck uint16, hp uint16, mp uint16, jobId job.Id, gender byte, face uint32, hair uint32, skin byte, mapId _map.Id) error
	SetHPAndEmitFunc           func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, amount uint16) error
	SetHPFunc                  func(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, amount uint16) error
	ResetStatsAndEmitFunc      func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id) error
	ResetStatsFunc             func(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id) error
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

// DeductExperienceAndEmit is a mock implementation of the character.Processor.DeductExperienceAndEmit method
func (m *ProcessorMock) DeductExperienceAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, amount uint32) error {
	if m.DeductExperienceAndEmitFunc != nil {
		return m.DeductExperienceAndEmitFunc(transactionId, worldId, characterId, channelId, amount)
	}
	return nil
}

// DeductExperience is a mock implementation of the character.Processor.DeductExperience method
func (m *ProcessorMock) DeductExperience(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, amount uint32) error {
	if m.DeductExperienceFunc != nil {
		return m.DeductExperienceFunc(mb)
	}
	return func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, amount uint32) error {
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

// AwardFameAndEmit is a mock implementation of the character.Processor.AwardFameAndEmit method
func (m *ProcessorMock) AwardFameAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, amount int16) error {
	if m.AwardFameAndEmitFunc != nil {
		return m.AwardFameAndEmitFunc(transactionId, worldId, characterId, channelId, amount)
	}
	return nil
}

// AwardFame is a mock implementation of the character.Processor.AwardFame method
func (m *ProcessorMock) AwardFame(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, amount int16) error {
	if m.AwardFameFunc != nil {
		return m.AwardFameFunc(mb)
	}
	return func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, amount int16) error {
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

// ChangeHairAndEmit is a mock implementation of the character.Processor.ChangeHairAndEmit method
func (m *ProcessorMock) ChangeHairAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, styleId uint32) error {
	if m.ChangeHairAndEmitFunc != nil {
		return m.ChangeHairAndEmitFunc(transactionId, worldId, characterId, channelId, styleId)
	}
	return nil
}

// ChangeHair is a mock implementation of the character.Processor.ChangeHair method
func (m *ProcessorMock) ChangeHair(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, styleId uint32) error {
	if m.ChangeHairFunc != nil {
		return m.ChangeHairFunc(mb)
	}
	return func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, styleId uint32) error {
		return nil
	}
}

// ChangeFaceAndEmit is a mock implementation of the character.Processor.ChangeFaceAndEmit method
func (m *ProcessorMock) ChangeFaceAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, styleId uint32) error {
	if m.ChangeFaceAndEmitFunc != nil {
		return m.ChangeFaceAndEmitFunc(transactionId, worldId, characterId, channelId, styleId)
	}
	return nil
}

// ChangeFace is a mock implementation of the character.Processor.ChangeFace method
func (m *ProcessorMock) ChangeFace(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, styleId uint32) error {
	if m.ChangeFaceFunc != nil {
		return m.ChangeFaceFunc(mb)
	}
	return func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, styleId uint32) error {
		return nil
	}
}

// ChangeSkinAndEmit is a mock implementation of the character.Processor.ChangeSkinAndEmit method
func (m *ProcessorMock) ChangeSkinAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, styleId byte) error {
	if m.ChangeSkinAndEmitFunc != nil {
		return m.ChangeSkinAndEmitFunc(transactionId, worldId, characterId, channelId, styleId)
	}
	return nil
}

// ChangeSkin is a mock implementation of the character.Processor.ChangeSkin method
func (m *ProcessorMock) ChangeSkin(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, styleId byte) error {
	if m.ChangeSkinFunc != nil {
		return m.ChangeSkinFunc(mb)
	}
	return func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, styleId byte) error {
		return nil
	}
}

// RequestCreateCharacter is a mock implementation of the character.Processor.RequestCreateCharacter method
func (m *ProcessorMock) RequestCreateCharacter(transactionId uuid.UUID, accountId uint32, worldId world.Id, name string, level byte, strength uint16, dexterity uint16, intelligence uint16, luck uint16, hp uint16, mp uint16, jobId job.Id, gender byte, face uint32, hair uint32, skin byte, mapId _map.Id) error {
	if m.RequestCreateCharacterFunc != nil {
		return m.RequestCreateCharacterFunc(transactionId, accountId, worldId, name, level, strength, dexterity, intelligence, luck, hp, mp, jobId, gender, face, hair, skin, mapId)
	}
	return nil
}

// SetHPAndEmit is a mock implementation of the character.Processor.SetHPAndEmit method
func (m *ProcessorMock) SetHPAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, amount uint16) error {
	if m.SetHPAndEmitFunc != nil {
		return m.SetHPAndEmitFunc(transactionId, worldId, characterId, channelId, amount)
	}
	return nil
}

// SetHP is a mock implementation of the character.Processor.SetHP method
func (m *ProcessorMock) SetHP(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, amount uint16) error {
	if m.SetHPFunc != nil {
		return m.SetHPFunc(mb)
	}
	return func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, amount uint16) error {
		return nil
	}
}

// ResetStatsAndEmit is a mock implementation of the character.Processor.ResetStatsAndEmit method
func (m *ProcessorMock) ResetStatsAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id) error {
	if m.ResetStatsAndEmitFunc != nil {
		return m.ResetStatsAndEmitFunc(transactionId, worldId, characterId, channelId)
	}
	return nil
}

// ResetStats is a mock implementation of the character.Processor.ResetStats method
func (m *ProcessorMock) ResetStats(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id) error {
	if m.ResetStatsFunc != nil {
		return m.ResetStatsFunc(mb)
	}
	return func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id) error {
		return nil
	}
}
