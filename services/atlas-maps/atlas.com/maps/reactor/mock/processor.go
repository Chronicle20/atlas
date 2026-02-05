package mock

import (
	"atlas-maps/kafka/message"
	"atlas-maps/reactor"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
)

type Processor struct {
	InMapModelProviderFunc func(transactionId uuid.UUID, field field.Model) model.Provider[[]reactor.Model]
	GetInMapFunc           func(transactionId uuid.UUID, field field.Model) ([]reactor.Model, error)
	SpawnFunc              func(mb *message.Buffer) func(transactionId uuid.UUID, field field.Model) error
	SpawnAndEmitFunc       func(transactionId uuid.UUID, field field.Model) error
}

func (m *Processor) InMapModelProvider(transactionId uuid.UUID, field field.Model) model.Provider[[]reactor.Model] {
	if m.InMapModelProviderFunc != nil {
		return m.InMapModelProviderFunc(transactionId, field)
	}
	return func() ([]reactor.Model, error) {
		return nil, nil
	}
}

func (m *Processor) GetInMap(transactionId uuid.UUID, field field.Model) ([]reactor.Model, error) {
	if m.GetInMapFunc != nil {
		return m.GetInMapFunc(transactionId, field)
	}
	return nil, nil
}

func (m *Processor) Spawn(mb *message.Buffer) func(transactionId uuid.UUID, field field.Model) error {
	if m.SpawnFunc != nil {
		return m.SpawnFunc(mb)
	}
	return func(transactionId uuid.UUID, field field.Model) error {
		return nil
	}
}

func (m *Processor) SpawnAndEmit(transactionId uuid.UUID, field field.Model) error {
	if m.SpawnAndEmitFunc != nil {
		return m.SpawnAndEmitFunc(transactionId, field)
	}
	return nil
}
