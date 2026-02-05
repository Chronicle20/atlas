package mock

import (
	"atlas-expressions/expression"
	"atlas-expressions/kafka/message"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/google/uuid"
)

type ProcessorMock struct {
	ChangeFunc        func(mb *message.Buffer, transactionId uuid.UUID, characterId uint32, field field.Model, expr uint32) (expression.Model, error)
	ChangeAndEmitFunc func(transactionId uuid.UUID, characterId uint32, field field.Model, expr uint32) (expression.Model, error)
	ClearFunc         func(mb *message.Buffer, transactionId uuid.UUID, characterId uint32) (expression.Model, error)
	ClearAndEmitFunc  func(transactionId uuid.UUID, characterId uint32) (expression.Model, error)
}

func (m *ProcessorMock) Change(mb *message.Buffer, transactionId uuid.UUID, characterId uint32, field field.Model, expr uint32) (expression.Model, error) {
	if m.ChangeFunc != nil {
		return m.ChangeFunc(mb, transactionId, characterId, field, expr)
	}
	return expression.Model{}, nil
}

func (m *ProcessorMock) ChangeAndEmit(transactionId uuid.UUID, characterId uint32, field field.Model, expr uint32) (expression.Model, error) {
	if m.ChangeAndEmitFunc != nil {
		return m.ChangeAndEmitFunc(transactionId, characterId, field, expr)
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
