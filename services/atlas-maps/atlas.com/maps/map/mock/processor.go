package mock

import (
	"atlas-maps/kafka/message"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/google/uuid"
)

type Processor struct {
	EnterFunc                    func(mb *message.Buffer) func(transactionId uuid.UUID, f field.Model, characterId uint32) error
	EnterAndEmitFunc             func(transactionId uuid.UUID, f field.Model, characterId uint32) error
	ExitFunc                     func(mb *message.Buffer) func(transactionId uuid.UUID, f field.Model, characterId uint32) error
	ExitAndEmitFunc              func(transactionId uuid.UUID, f field.Model, characterId uint32) error
	TransitionMapFunc            func(mb *message.Buffer) func(transactionId uuid.UUID, newField field.Model, characterId uint32, oldField field.Model)
	TransitionMapAndEmitFunc     func(transactionId uuid.UUID, newField field.Model, characterId uint32, oldField field.Model) error
	TransitionChannelFunc        func(mb *message.Buffer) func(transactionId uuid.UUID, newField field.Model, oldChannelId channel.Id, characterId uint32)
	TransitionChannelAndEmitFunc func(transactionId uuid.UUID, newField field.Model, oldChannelId channel.Id, characterId uint32) error
	GetCharactersInMapFunc       func(transactionId uuid.UUID, f field.Model) ([]uint32, error)
}

func (m *Processor) Enter(mb *message.Buffer) func(transactionId uuid.UUID, f field.Model, characterId uint32) error {
	if m.EnterFunc != nil {
		return m.EnterFunc(mb)
	}
	return func(transactionId uuid.UUID, f field.Model, characterId uint32) error {
		return nil
	}
}

func (m *Processor) EnterAndEmit(transactionId uuid.UUID, f field.Model, characterId uint32) error {
	if m.EnterAndEmitFunc != nil {
		return m.EnterAndEmitFunc(transactionId, f, characterId)
	}
	return nil
}

func (m *Processor) Exit(mb *message.Buffer) func(transactionId uuid.UUID, f field.Model, characterId uint32) error {
	if m.ExitFunc != nil {
		return m.ExitFunc(mb)
	}
	return func(transactionId uuid.UUID, f field.Model, characterId uint32) error {
		return nil
	}
}

func (m *Processor) ExitAndEmit(transactionId uuid.UUID, f field.Model, characterId uint32) error {
	if m.ExitAndEmitFunc != nil {
		return m.ExitAndEmitFunc(transactionId, f, characterId)
	}
	return nil
}

func (m *Processor) TransitionMap(mb *message.Buffer) func(transactionId uuid.UUID, newField field.Model, characterId uint32, oldField field.Model) {
	if m.TransitionMapFunc != nil {
		return m.TransitionMapFunc(mb)
	}
	return func(transactionId uuid.UUID, newField field.Model, characterId uint32, oldField field.Model) {
	}
}

func (m *Processor) TransitionMapAndEmit(transactionId uuid.UUID, newField field.Model, characterId uint32, oldField field.Model) error {
	if m.TransitionMapAndEmitFunc != nil {
		return m.TransitionMapAndEmitFunc(transactionId, newField, characterId, oldField)
	}
	return nil
}

func (m *Processor) TransitionChannel(mb *message.Buffer) func(transactionId uuid.UUID, newField field.Model, oldChannelId channel.Id, characterId uint32) {
	if m.TransitionChannelFunc != nil {
		return m.TransitionChannelFunc(mb)
	}
	return func(transactionId uuid.UUID, newField field.Model, oldChannelId channel.Id, characterId uint32) {
	}
}

func (m *Processor) TransitionChannelAndEmit(transactionId uuid.UUID, newField field.Model, oldChannelId channel.Id, characterId uint32) error {
	if m.TransitionChannelAndEmitFunc != nil {
		return m.TransitionChannelAndEmitFunc(transactionId, newField, oldChannelId, characterId)
	}
	return nil
}

func (m *Processor) GetCharactersInMap(transactionId uuid.UUID, f field.Model) ([]uint32, error) {
	if m.GetCharactersInMapFunc != nil {
		return m.GetCharactersInMapFunc(transactionId, f)
	}
	return nil, nil
}
