package mock

import (
	"atlas-party-quests/instance"
	"atlas-party-quests/kafka/message"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/google/uuid"
)

type ProcessorMock struct {
	RegisterFunc              func(mb *message.Buffer) func(questId string, partyId uint32, channelId channel.Id, mapId uint32, characters []instance.CharacterEntry) (instance.Model, error)
	RegisterAndEmitFunc       func(questId string, partyId uint32, channelId channel.Id, mapId uint32, characters []instance.CharacterEntry) (instance.Model, error)
	StartFunc                 func(mb *message.Buffer) func(instanceId uuid.UUID) error
	StartAndEmitFunc          func(instanceId uuid.UUID) error
	StageClearAttemptFunc     func(mb *message.Buffer) func(instanceId uuid.UUID) error
	StageClearAttemptAndEmitFunc func(instanceId uuid.UUID) error
	StageAdvanceFunc          func(mb *message.Buffer) func(instanceId uuid.UUID) error
	StageAdvanceAndEmitFunc   func(instanceId uuid.UUID) error
	ForfeitFunc               func(mb *message.Buffer) func(instanceId uuid.UUID) error
	ForfeitAndEmitFunc        func(instanceId uuid.UUID) error
	LeaveFunc                 func(mb *message.Buffer) func(characterId uint32, reason string) error
	LeaveAndEmitFunc          func(characterId uint32, reason string) error
	UpdateStageStateFunc      func(instanceId uuid.UUID, itemCounts map[uint32]uint32, monsterKills map[uint32]uint32) error
	UpdateCustomDataFunc      func(instanceId uuid.UUID, updates map[string]string, increments []string) error
	BroadcastMessageFunc      func(mb *message.Buffer) func(instanceId uuid.UUID, messageType string, msg string) error
	BroadcastMessageAndEmitFunc func(instanceId uuid.UUID, messageType string, msg string) error
	GetByFieldInstanceFunc    func(fieldInstance uuid.UUID) (instance.Model, error)
	DestroyFunc               func(mb *message.Buffer) func(instanceId uuid.UUID, reason string) error
	DestroyAndEmitFunc        func(instanceId uuid.UUID, reason string) error
	TickGlobalTimerFunc       func(mb *message.Buffer) error
	TickGlobalTimerAndEmitFunc func() error
	TickStageTimerFunc        func(mb *message.Buffer) error
	TickStageTimerAndEmitFunc func() error
	TickBonusTimerFunc        func(mb *message.Buffer) error
	TickBonusTimerAndEmitFunc func() error
	TickRegistrationTimerFunc func(mb *message.Buffer) error
	TickRegistrationTimerAndEmitFunc func() error
	GracefulShutdownFunc      func(mb *message.Buffer) error
	GracefulShutdownAndEmitFunc func() error
	GetByIdFunc               func(instanceId uuid.UUID) (instance.Model, error)
	GetByCharacterFunc        func(characterId uint32) (instance.Model, error)
	GetTimerByCharacterFunc   func(characterId uint32) (uint64, error)
	GetAllFunc                func() []instance.Model
}

func (m *ProcessorMock) Register(mb *message.Buffer) func(questId string, partyId uint32, channelId channel.Id, mapId uint32, characters []instance.CharacterEntry) (instance.Model, error) {
	if m.RegisterFunc != nil {
		return m.RegisterFunc(mb)
	}
	return func(string, uint32, channel.Id, uint32, []instance.CharacterEntry) (instance.Model, error) {
		return instance.Model{}, nil
	}
}

func (m *ProcessorMock) RegisterAndEmit(questId string, partyId uint32, channelId channel.Id, mapId uint32, characters []instance.CharacterEntry) (instance.Model, error) {
	if m.RegisterAndEmitFunc != nil {
		return m.RegisterAndEmitFunc(questId, partyId, channelId, mapId, characters)
	}
	return instance.Model{}, nil
}

func (m *ProcessorMock) Start(mb *message.Buffer) func(instanceId uuid.UUID) error {
	if m.StartFunc != nil {
		return m.StartFunc(mb)
	}
	return func(uuid.UUID) error { return nil }
}

func (m *ProcessorMock) StartAndEmit(instanceId uuid.UUID) error {
	if m.StartAndEmitFunc != nil {
		return m.StartAndEmitFunc(instanceId)
	}
	return nil
}

func (m *ProcessorMock) StageClearAttempt(mb *message.Buffer) func(instanceId uuid.UUID) error {
	if m.StageClearAttemptFunc != nil {
		return m.StageClearAttemptFunc(mb)
	}
	return func(uuid.UUID) error { return nil }
}

func (m *ProcessorMock) StageClearAttemptAndEmit(instanceId uuid.UUID) error {
	if m.StageClearAttemptAndEmitFunc != nil {
		return m.StageClearAttemptAndEmitFunc(instanceId)
	}
	return nil
}

func (m *ProcessorMock) StageAdvance(mb *message.Buffer) func(instanceId uuid.UUID) error {
	if m.StageAdvanceFunc != nil {
		return m.StageAdvanceFunc(mb)
	}
	return func(uuid.UUID) error { return nil }
}

func (m *ProcessorMock) StageAdvanceAndEmit(instanceId uuid.UUID) error {
	if m.StageAdvanceAndEmitFunc != nil {
		return m.StageAdvanceAndEmitFunc(instanceId)
	}
	return nil
}

func (m *ProcessorMock) Forfeit(mb *message.Buffer) func(instanceId uuid.UUID) error {
	if m.ForfeitFunc != nil {
		return m.ForfeitFunc(mb)
	}
	return func(uuid.UUID) error { return nil }
}

func (m *ProcessorMock) ForfeitAndEmit(instanceId uuid.UUID) error {
	if m.ForfeitAndEmitFunc != nil {
		return m.ForfeitAndEmitFunc(instanceId)
	}
	return nil
}

func (m *ProcessorMock) Leave(mb *message.Buffer) func(characterId uint32, reason string) error {
	if m.LeaveFunc != nil {
		return m.LeaveFunc(mb)
	}
	return func(uint32, string) error { return nil }
}

func (m *ProcessorMock) LeaveAndEmit(characterId uint32, reason string) error {
	if m.LeaveAndEmitFunc != nil {
		return m.LeaveAndEmitFunc(characterId, reason)
	}
	return nil
}

func (m *ProcessorMock) UpdateStageState(instanceId uuid.UUID, itemCounts map[uint32]uint32, monsterKills map[uint32]uint32) error {
	if m.UpdateStageStateFunc != nil {
		return m.UpdateStageStateFunc(instanceId, itemCounts, monsterKills)
	}
	return nil
}

func (m *ProcessorMock) UpdateCustomData(instanceId uuid.UUID, updates map[string]string, increments []string) error {
	if m.UpdateCustomDataFunc != nil {
		return m.UpdateCustomDataFunc(instanceId, updates, increments)
	}
	return nil
}

func (m *ProcessorMock) BroadcastMessage(mb *message.Buffer) func(instanceId uuid.UUID, messageType string, msg string) error {
	if m.BroadcastMessageFunc != nil {
		return m.BroadcastMessageFunc(mb)
	}
	return func(uuid.UUID, string, string) error { return nil }
}

func (m *ProcessorMock) BroadcastMessageAndEmit(instanceId uuid.UUID, messageType string, msg string) error {
	if m.BroadcastMessageAndEmitFunc != nil {
		return m.BroadcastMessageAndEmitFunc(instanceId, messageType, msg)
	}
	return nil
}

func (m *ProcessorMock) GetByFieldInstance(fieldInstance uuid.UUID) (instance.Model, error) {
	if m.GetByFieldInstanceFunc != nil {
		return m.GetByFieldInstanceFunc(fieldInstance)
	}
	return instance.Model{}, nil
}

func (m *ProcessorMock) Destroy(mb *message.Buffer) func(instanceId uuid.UUID, reason string) error {
	if m.DestroyFunc != nil {
		return m.DestroyFunc(mb)
	}
	return func(uuid.UUID, string) error { return nil }
}

func (m *ProcessorMock) DestroyAndEmit(instanceId uuid.UUID, reason string) error {
	if m.DestroyAndEmitFunc != nil {
		return m.DestroyAndEmitFunc(instanceId, reason)
	}
	return nil
}

func (m *ProcessorMock) TickGlobalTimer(mb *message.Buffer) error {
	if m.TickGlobalTimerFunc != nil {
		return m.TickGlobalTimerFunc(mb)
	}
	return nil
}

func (m *ProcessorMock) TickGlobalTimerAndEmit() error {
	if m.TickGlobalTimerAndEmitFunc != nil {
		return m.TickGlobalTimerAndEmitFunc()
	}
	return nil
}

func (m *ProcessorMock) TickStageTimer(mb *message.Buffer) error {
	if m.TickStageTimerFunc != nil {
		return m.TickStageTimerFunc(mb)
	}
	return nil
}

func (m *ProcessorMock) TickStageTimerAndEmit() error {
	if m.TickStageTimerAndEmitFunc != nil {
		return m.TickStageTimerAndEmitFunc()
	}
	return nil
}

func (m *ProcessorMock) TickBonusTimer(mb *message.Buffer) error {
	if m.TickBonusTimerFunc != nil {
		return m.TickBonusTimerFunc(mb)
	}
	return nil
}

func (m *ProcessorMock) TickBonusTimerAndEmit() error {
	if m.TickBonusTimerAndEmitFunc != nil {
		return m.TickBonusTimerAndEmitFunc()
	}
	return nil
}

func (m *ProcessorMock) TickRegistrationTimer(mb *message.Buffer) error {
	if m.TickRegistrationTimerFunc != nil {
		return m.TickRegistrationTimerFunc(mb)
	}
	return nil
}

func (m *ProcessorMock) TickRegistrationTimerAndEmit() error {
	if m.TickRegistrationTimerAndEmitFunc != nil {
		return m.TickRegistrationTimerAndEmitFunc()
	}
	return nil
}

func (m *ProcessorMock) GracefulShutdown(mb *message.Buffer) error {
	if m.GracefulShutdownFunc != nil {
		return m.GracefulShutdownFunc(mb)
	}
	return nil
}

func (m *ProcessorMock) GracefulShutdownAndEmit() error {
	if m.GracefulShutdownAndEmitFunc != nil {
		return m.GracefulShutdownAndEmitFunc()
	}
	return nil
}

func (m *ProcessorMock) GetById(instanceId uuid.UUID) (instance.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(instanceId)
	}
	return instance.Model{}, nil
}

func (m *ProcessorMock) GetByCharacter(characterId uint32) (instance.Model, error) {
	if m.GetByCharacterFunc != nil {
		return m.GetByCharacterFunc(characterId)
	}
	return instance.Model{}, nil
}

func (m *ProcessorMock) GetTimerByCharacter(characterId uint32) (uint64, error) {
	if m.GetTimerByCharacterFunc != nil {
		return m.GetTimerByCharacterFunc(characterId)
	}
	return 0, nil
}

func (m *ProcessorMock) GetAll() []instance.Model {
	if m.GetAllFunc != nil {
		return m.GetAllFunc()
	}
	return []instance.Model{}
}
