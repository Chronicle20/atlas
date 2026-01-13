package mock

import (
	"atlas-expressions/expression"
	"atlas-expressions/kafka/message"
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

type ProcessorMock struct {
	ChangeFunc        func(mb *message.Buffer, transactionId uuid.UUID, characterId uint32, worldId world.Id, channelId channel.Id, mapId _map.Id, expr uint32) (expression.Model, error)
	ChangeAndEmitFunc func(transactionId uuid.UUID, characterId uint32, worldId world.Id, channelId channel.Id, mapId _map.Id, expr uint32) (expression.Model, error)
	ClearFunc         func(mb *message.Buffer, transactionId uuid.UUID, characterId uint32) (expression.Model, error)
	ClearAndEmitFunc  func(transactionId uuid.UUID, characterId uint32) (expression.Model, error)
}

func (m *ProcessorMock) Change(mb *message.Buffer, transactionId uuid.UUID, characterId uint32, worldId world.Id, channelId channel.Id, mapId _map.Id, expr uint32) (expression.Model, error) {
	if m.ChangeFunc != nil {
		return m.ChangeFunc(mb, transactionId, characterId, worldId, channelId, mapId, expr)
	}
	return expression.Model{}, nil
}

func (m *ProcessorMock) ChangeAndEmit(transactionId uuid.UUID, characterId uint32, worldId world.Id, channelId channel.Id, mapId _map.Id, expr uint32) (expression.Model, error) {
	if m.ChangeAndEmitFunc != nil {
		return m.ChangeAndEmitFunc(transactionId, characterId, worldId, channelId, mapId, expr)
	}
	return expression.Model{}, nil
}

func (m *ProcessorMock) Clear(mb *message.Buffer, transactionId uuid.UUID, characterId uint32) (expression.Model, error) {
	if m.ClearFunc != nil {
		return m.ClearFunc(mb, transactionId, characterId)
	}
	return expression.Model{}, nil
}

func (m *ProcessorMock) ClearAndEmit(transactionId uuid.UUID, characterId uint32) (expression.Model, error) {
	if m.ClearAndEmitFunc != nil {
		return m.ClearAndEmitFunc(transactionId, characterId)
	}
	return expression.Model{}, nil
}
